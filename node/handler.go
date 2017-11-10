package node

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/vechain/solidb/blob"
	"github.com/vechain/solidb/blobio"
	"github.com/vechain/solidb/crypto"
	"github.com/vechain/solidb/spec"
	"github.com/vechain/solidb/utils/httpx"
)

const (
	// HTTPPathPrefix path prefix
	HTTPPathPrefix = "/node/"

	signatureHeaderKey = "x-solidb-signature"
	targetIDHeaderKey  = "x-solidb-target-id"
)

// NewHTTPHandler create http handler to expose operations to local node
func NewHTTPHandler(node *Node) http.Handler {
	router := mux.NewRouter()

	sub := router.PathPrefix(HTTPPathPrefix).Subrouter()
	sub.Methods(http.MethodPost).Path("/invitation").HandlerFunc(httpx.WrapHandlerFunc(node.handleInvite))
	sub.Methods(http.MethodPost).Path("/specs").HandlerFunc(httpx.WrapHandlerFunc(node.handleProposeSpec))
	sub.Methods(http.MethodPost).Path("/specs/{revision}").Queries("action", "{action}").HandlerFunc(httpx.WrapHandlerFunc(node.handleSpecAction))

	sub.Methods(http.MethodGet).Path("/status").HandlerFunc(httpx.WrapHandlerFunc(node.handleGetStatus))
	sub.Methods(http.MethodGet).Path("/status/sync").Queries("revision", "{revision}").HandlerFunc(httpx.WrapHandlerFunc(node.handleGetSyncStatus))

	sub.Methods(http.MethodGet).Path("/blobs/{key}").HandlerFunc(httpx.WrapHandlerFunc(node.handleGetBlob))
	sub.Methods(http.MethodPost).Path("/blobs").HandlerFunc(httpx.WrapHandlerFunc(node.handlePutBlob))
	sub.Methods(http.MethodGet).Path("/blobs").Queries("prefix", "{prefix}").HandlerFunc(httpx.WrapHandlerFunc(node.handleGetBlobSlice))

	return router
}

func (n *Node) handleSignedRequest(req *http.Request, signerID *string) ([]byte, error) {
	targetID := req.Header.Get(targetIDHeaderKey)
	if targetID != "" && targetID != n.ID() {
		return nil, errors.New("not the target")
	}

	sig, err := hex.DecodeString(req.Header.Get(signatureHeaderKey))
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	vdata := append([]byte(targetID+"\n"+req.RequestURI+"\n"), data...)
	hash := crypto.HashSum(vdata)
	sid, err := crypto.RecoverID(hash, sig)
	if err != nil {
		return nil, err
	}

	if *signerID == "" {
		*signerID = sid
	} else {
		if *signerID != sid {
			return nil, errors.New("not the master")
		}
	}
	return data, nil
}

func (n *Node) handleInvite(w http.ResponseWriter, req *http.Request) error {
	signerID := ""
	data, err := n.handleSignedRequest(req, &signerID)
	if err != nil {
		return httpx.Error(err, http.StatusUnauthorized)
	}

	var reqBody InviteRequest
	if err := json.Unmarshal(data, &reqBody); err != nil {
		return httpx.Error(err, http.StatusBadRequest)
	}

	if err := n.Invite(signerID, reqBody.InitSpec); err != nil {
		return err
	}

	return httpx.ResponseJSON(w, &InviteResponse{
		NodeID: n.ID(),
	})
}

func (n *Node) handleProposeSpec(w http.ResponseWriter, req *http.Request) error {
	signerID := n.clusterID
	data, err := n.handleSignedRequest(req, &signerID)
	if err != nil {
		return httpx.Error(err, http.StatusUnauthorized)
	}
	var s spec.Spec
	if err := json.Unmarshal(data, &s); err != nil {
		return httpx.Error(err, http.StatusBadRequest)
	}
	return n.ProposeSpec(s)
}

func (n *Node) handleSpecAction(w http.ResponseWriter, req *http.Request) error {
	signerID := n.clusterID
	_, err := n.handleSignedRequest(req, &signerID)
	if err != nil {
		return httpx.Error(err, http.StatusUnauthorized)
	}

	vars := mux.Vars(req)
	revStr := vars["revision"]
	rev, err := strconv.Atoi(revStr)
	if err != nil {
		return httpx.Error(err, http.StatusBadRequest)
	}
	action := vars["action"]
	if action == "approve" {
		return n.ApproveSpec(rev)
	} else if action == "sync" {
		return n.RequestSync(rev)
	} else {
		return httpx.Error(errors.New("unknown action"), http.StatusBadRequest)
	}
}

func (n *Node) handleGetStatus(w http.ResponseWriter, req *http.Request) error {
	status, err := n.GetStatus()
	if err != nil {
		return err
	}
	return httpx.ResponseJSON(w, status)
}

func (n *Node) handleGetSyncStatus(w http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	revStr := vars["revision"]
	rev, err := strconv.Atoi(revStr)
	if err != nil {
		return httpx.Error(err, http.StatusBadRequest)
	}

	status, err := n.GetSyncStatus(rev)
	if err != nil {
		return err
	}
	return httpx.ResponseJSON(w, status)
}

func (n *Node) handleGetBlob(w http.ResponseWriter, req *http.Request) error {
	keyHex := mux.Vars(req)["key"]
	key, err := blob.ParseHexKey(keyHex)
	if err != nil {
		return httpx.Error(err, http.StatusBadRequest)
	}

	blob, err := blobio.GetBlob(n.store, *key)
	if err != nil {
		return err
	}
	if blob.V == nil {
		return httpx.Error(nil, http.StatusNoContent)
	}

	w.Header().Set("Content-Type", httpx.OctetStreamContentType)
	w.Write(blob.V.Data())
	return nil
}

func (n *Node) handlePutBlob(w http.ResponseWriter, req *http.Request) error {
	if req.ContentLength > blob.DataLenHardLimit {
		return httpx.Error(errors.New("content length exceeds limit"), http.StatusNotAcceptable)
	}
	if req.ContentLength < 0 {
		return httpx.Error(errors.New("content length unknown"), http.StatusNotAcceptable)
	}
	data := make([]byte, req.ContentLength)
	if _, err := io.ReadFull(req.Body, data); err != nil {
		return err
	}

	blob := blob.New(data)
	if err := blobio.PutBlob(n.store, blob); err != nil {
		return err
	}

	return httpx.ResponseJSON(w, &PutBlobResponse{
		Key: blob.Key(),
	})
}

func (n *Node) handleGetBlobSlice(w http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	prefix := vars["prefix"]

	blobIter, err := blobio.NewBlobIterator(n.store, prefix)
	if err != nil {
		return httpx.Error(err, http.StatusBadRequest)
	}
	defer blobIter.Release()

	// chunked
	w.Header().Set("Content-Type", httpx.OctetStreamContentType)
	w.(http.Flusher).Flush()

	for blobIter.Next() {
		blob, err := blobIter.Blob()
		if err != nil {
			log.Errorf("iterate blob: %v", err)
			return nil
		}

		if err := blobio.WriteBlob(w, blob); err != nil {
			log.Error(err)
			return nil
		}
	}

	if err := blobIter.Error(); err != nil {
		log.Error("Failed to iterate slice, cause:", err)
		return nil
	}

	if err := blobio.EndWriteBlob(w); err != nil {
		log.Error(err)
	}

	return nil
}
