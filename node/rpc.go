package node

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/vechain/solidb/blob"
	"github.com/vechain/solidb/blobio"
	"github.com/vechain/solidb/crypto"
	"github.com/vechain/solidb/spec"
	"github.com/vechain/solidb/utils/httpx"
)

type RPC struct {
	client   *http.Client
	baseURL  string
	ctx      context.Context
	identity *crypto.Identity
	targetID string
}

var defaultTransport = http.Transport{}

func NewRPC() *RPC {
	return &RPC{
		client: &http.Client{Transport: &defaultTransport},
		ctx:    context.Background(),
	}
}

func (rpc *RPC) WithAddr(addr string) *RPC {
	cp := *rpc
	cp.baseURL = "http://" + addr + HTTPPathPrefix
	return &cp
}

func (rpc *RPC) WithContext(ctx context.Context) *RPC {
	if ctx == nil {
		panic("nil ctx")
	}
	cp := *rpc
	cp.ctx = ctx
	return &cp
}

func (rpc *RPC) WithIdentity(identity *crypto.Identity, targetID string) *RPC {
	cp := *rpc
	cp.identity = identity
	cp.targetID = targetID
	return &cp
}

func (rpc *RPC) doRequest(req *http.Request) (*http.Response, []byte, error) {
	if rpc.identity != nil {
		var data []byte
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, nil, err
			}
			data, err = ioutil.ReadAll(body)
			if err != nil {
				return nil, nil, err
			}
		}
		vdata := append([]byte(rpc.targetID+"\n"+req.URL.RequestURI()+"\n"), data...)

		hash := crypto.HashSum(vdata)
		sig, err := rpc.identity.Sign(hash)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set(signatureHeaderKey, hex.EncodeToString(sig))
		req.Header.Set(targetIDHeaderKey, rpc.targetID)
	}

	req = req.WithContext(rpc.ctx)
	resp, err := rpc.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if err := httpx.HandleResponseError(resp); err != nil {
		return nil, nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return resp, data, nil
}

func (rpc *RPC) Invite(initSpec *spec.Spec) (nodeID string, err error) {
	data, err := json.Marshal(&InviteRequest{
		InitSpec: initSpec,
	})
	if err != nil {
		return "", err
	}

	body := bytes.NewReader(data)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		rpc.baseURL+"invitation",
		body,
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", httpx.JSONContentType)
	_, data, err = rpc.doRequest(req)
	if err != nil {
		return "", err
	}
	var respBody InviteResponse
	if err := json.Unmarshal(data, &respBody); err != nil {
		return "", err
	}
	return respBody.NodeID, nil
}

func (rpc *RPC) GetStatus() (*StatusResponse, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		rpc.baseURL+"status",
		nil,
	)
	if err != nil {
		return nil, err
	}
	_, data, err := rpc.doRequest(req)
	if err != nil {
		return nil, err
	}

	var status StatusResponse
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (rpc *RPC) GetSyncStatus(revision int) (*SyncStatusResponse, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		rpc.baseURL+"status/sync?revision="+strconv.Itoa(revision),
		nil,
	)
	if err != nil {
		return nil, err
	}
	_, data, err := rpc.doRequest(req)
	if err != nil {
		return nil, err
	}
	var status SyncStatusResponse
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (rpc *RPC) GetBlob(blobKey blob.Key) (*blobio.OptBlob, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		rpc.baseURL+"blobs/"+blobKey.ToHex(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	resp, data, err := rpc.doRequest(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNoContent {
		return &blobio.OptBlob{}, nil
	}
	blob := blob.New(data)
	if blob.Key() != blobKey {
		return nil, errors.New("get blob with wrong key")
	}
	return &blobio.OptBlob{blob}, nil
}

func (rpc *RPC) PutBlob(blob *blob.Blob) error {
	req, err := http.NewRequest(
		http.MethodPost,
		rpc.baseURL+"blobs",
		bytes.NewReader(blob.Data()),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", httpx.OctetStreamContentType)
	_, data, err := rpc.doRequest(req)
	if err != nil {
		return err
	}
	var respBody PutBlobResponse
	if err := json.Unmarshal(data, &respBody); err != nil {
		return err
	}
	if respBody.Key != blob.Key() {
		return errors.New("put blob returned incorrect key")
	}
	return nil
}

func (rpc *RPC) GetBlobSlice(prefix string) (io.ReadCloser, error) {

	req, err := http.NewRequest(
		http.MethodGet,
		rpc.baseURL+"blobs?prefix="+url.QueryEscape(prefix),
		nil,
	)
	req = req.WithContext(rpc.ctx)
	resp, err := rpc.client.Do(req)
	if err != nil {
		return nil, err
	}
	if err := httpx.HandleResponseError(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}
	return resp.Body, nil
}

func (rpc *RPC) ProposeSpec(s spec.Spec) error {
	data, err := json.Marshal(&s)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(
		http.MethodPost,
		rpc.baseURL+"specs",
		bytes.NewReader(data),
	)
	if _, _, err := rpc.doRequest(req); err != nil {
		return err
	}
	return nil
}

func (rpc *RPC) performSpecAction(revision int, action string) error {
	req, err := http.NewRequest(
		http.MethodPost,
		rpc.baseURL+"specs/"+strconv.Itoa(revision)+"?action="+url.QueryEscape(action),
		nil,
	)
	if err != nil {
		return err
	}
	if _, _, err := rpc.doRequest(req); err != nil {
		return err
	}
	return nil
}
func (rpc *RPC) ApproveSpec(revision int) error {
	return rpc.performSpecAction(revision, "approve")
}

func (rpc *RPC) SyncToSpec(revision int) error {
	return rpc.performSpecAction(revision, "sync")
}
