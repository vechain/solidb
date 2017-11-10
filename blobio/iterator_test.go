package blobio_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vechain/solidb/blob"
	. "github.com/vechain/solidb/blobio"
	"github.com/vechain/solidb/kv"
)

func TestIter(t *testing.T) {
	assert := assert.New(t)

	db, _ := kv.NewMemStore(kv.Options{})
	defer db.Close()

	var blobs []*blob.Blob
	for i := 0; i < 10; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		blob := blob.New(data)
		blobs = append(blobs, blob)
		PutBlob(db, blob)
	}

	iter, _ := NewBlobIterator(db, "")
	defer iter.Release()
	count := 0
	for iter.Next() {
		count++
		blob, _ := iter.Blob()

		found := false
		for _, b := range blobs {
			if b.Key() == blob.Key() {
				found = true
				break
			}
		}
		assert.True(found)
	}
	assert.Equal(count, len(blobs))
}
