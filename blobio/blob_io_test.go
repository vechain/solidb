package blobio_test

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vechain/solidb/blob"
	. "github.com/vechain/solidb/blobio"
)

func TestReadWrite(t *testing.T) {
	assert := assert.New(t)
	rand.Seed(time.Now().UnixNano())

	var blobs []*blob.Blob
	for i := 0; i < 10; i++ {
		data := make([]byte, rand.Int()%65536)
		rand.Read(data)
		blobs = append(blobs, blob.New(data))
	}

	// write
	buf := bytes.NewBuffer([]byte{})
	for _, blob := range blobs {
		WriteBlob(buf, blob)
	}
	EndWriteBlob(buf)

	// read
	for _, blob := range blobs {
		opt, _ := ReadBlob(buf)
		assert.Equal(opt.V.Data(), blob.Data())
	}

	opt, _ := ReadBlob(buf)
	assert.True(opt.V == nil)
}
