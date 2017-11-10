package blobio

import (
	"bytes"
	"encoding/hex"

	"github.com/pkg/errors"
	"github.com/vechain/solidb/blob"
	"github.com/vechain/solidb/kv"
)

// BlobIterator iterates blobs in kv store
type BlobIterator struct {
	iter kv.Iterator
}

// NewBlobIterator create blob iterator
func NewBlobIterator(store kv.Store, blobKeyHexPrefix string) (*BlobIterator, error) {
	hexPrefix := hex.EncodeToString(blobPrefix) + blobKeyHexPrefix
	rng, err := kv.NewRangeWithHexPrefix(hexPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "new blob iterator")
	}
	return &BlobIterator{
		iter: store.NewIterator(rng),
	}, nil
}

// Next advance iterator
func (bi *BlobIterator) Next() bool {
	return bi.iter.Next()
}

// Release release resource alloced for iterator
func (bi *BlobIterator) Release() {
	bi.iter.Release()
}

// Error returns error occurred
func (bi *BlobIterator) Error() error {
	return errors.Wrap(bi.iter.Error(), "blob iterator")
}

// Blob returns current blob
func (bi *BlobIterator) Blob() (*blob.Blob, error) {
	blob := blob.New(bi.iter.Value())
	if !bytes.Equal(makeBlobKey(blob.Key()), bi.iter.Key()) {
		return nil, errors.Wrap(errors.New("key and value mismatch"), "blob iterator")
	}
	return blob, nil
}
