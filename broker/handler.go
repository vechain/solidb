package broker

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/vechain/solidb/blob"
	"github.com/vechain/solidb/node"
	"github.com/vechain/solidb/utils/httpx"
)

const HTTPPathPrefix = "/"

func NewHTTPHandler(broker *Broker) http.Handler {
	router := mux.NewRouter()
	sub := router.PathPrefix(HTTPPathPrefix).Subrouter()

	sub.Methods(http.MethodGet).Path("/blobs/{key}").HandlerFunc(httpx.WrapHandlerFunc(broker.handleGet))
	sub.Methods(http.MethodPost).Path("/blobs").HandlerFunc(httpx.WrapHandlerFunc(broker.handlePut))
	return router
}

func (b *Broker) handleGet(w http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	key, err := blob.ParseHexKey(vars["key"])
	if err != nil {
		return httpx.Error(err, http.StatusBadRequest)
	}

	blob, err := b.GetBlob(req.Context(), *key)
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

func (b *Broker) handlePut(w http.ResponseWriter, req *http.Request) error {
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
	if err := b.PutBlob(req.Context(), blob); err != nil {
		return err
	}

	return httpx.ResponseJSON(w, node.PutBlobResponse{
		Key: blob.Key(),
	})
}
