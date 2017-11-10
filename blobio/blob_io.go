// Package blobio provides IO operations for blob
package blobio

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
	"github.com/vechain/solidb/blob"
	"github.com/vechain/solidb/kv"
)

var (
	blobPrefix = []byte("/")
)

func makeBlobKey(blobKey blob.Key) []byte {
	return append([]byte(blobPrefix), blobKey[:]...)
}

// OptBlob presents optional blob.
type OptBlob struct {
	V *blob.Blob
}

// GetBlob get blob from kv reader by key.
func GetBlob(reader kv.Reader, blobKey blob.Key) (*OptBlob, error) {
	key := makeBlobKey(blobKey)
	value, err := reader.Get(key)
	if err != nil {
		return nil, errors.Wrap(err, "get blob")
	}
	if value.V == nil {
		return &OptBlob{}, nil
	}
	return &OptBlob{blob.New(value.V)}, nil
}

// PutBlob  store blob to kv writer
func PutBlob(writer kv.Writer, blob *blob.Blob) error {
	key := makeBlobKey(blob.Key())
	if err := writer.Put(key, blob.Data()); err != nil {
		return errors.Wrap(err, "put blob")
	}
	return nil
}

// ReadBlob read blob from reader
func ReadBlob(reader io.Reader) (*OptBlob, error) {
	var key blob.Key
	// firstly read key
	if _, err := io.ReadFull(reader, key[:]); err != nil {
		return nil, errors.Wrap(err, "read blob")
	}
	if key == blob.EmptyKey {
		// reach the end of stream
		return &OptBlob{}, nil
	}

	ind := [4]byte{}
	// then read 4 bytes, which indicate length of blob data
	if _, err := io.ReadFull(reader, ind[:]); err != nil {
		return nil, errors.Wrap(err, "read blob")
	}
	blobLen := binary.BigEndian.Uint32(ind[:])
	data := make([]byte, blobLen)
	// finally read blob data
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, errors.Wrap(err, "read blob")
	}

	blob := blob.New(data)
	// verify key
	if blob.Key() != key {
		return nil, errors.New("read blob: key value mismatch")
	}
	return &OptBlob{blob}, nil
}

// WriteBlob write blob to writer
func WriteBlob(writer io.Writer, blob *blob.Blob) error {
	key := blob.Key()
	// write blob key
	if _, err := writer.Write(key[:]); err != nil {
		return errors.Wrap(err, "write blob")
	}
	ind := [4]byte{}
	binary.BigEndian.PutUint32(ind[:], uint32(len(blob.Data())))
	// write blob data size indicator
	if _, err := writer.Write(ind[:]); err != nil {
		return errors.Wrap(err, "write blob")
	}
	// write blob data
	if _, err := writer.Write(blob.Data()); err != nil {
		return errors.Wrap(err, "write blob")
	}
	return nil
}

// EndWriteBlob end the write stream
func EndWriteBlob(writer io.Writer) error {
	// end
	if _, err := writer.Write(blob.EmptyKey[:]); err != nil {
		return errors.Wrap(err, "end write blob")
	}
	return nil
}
