package blobio

import (
	"github.com/pkg/errors"
	"github.com/vechain/solidb/blob"
	"github.com/vechain/solidb/kv"
)

const (
	// FaultBlobMark mark indicates that a blob needs to be resent
	FaultBlobMark = "fault"
	markPrefix    = ".marks/"
)

func makeMarkKey(blobKey blob.Key, mark string) []byte {
	return append([]byte(markPrefix+mark), blobKey[:]...)
}

func extractBlobKey(markKey []byte, mark string) (*blob.Key, error) {
	prefix := markPrefix + mark
	if len(markKey)-len(prefix) != blob.KeyLength {
		return nil, errors.New("invalid blob mark")
	}
	var blobKey blob.Key
	copy(blobKey[:], markKey[len(prefix):])
	return &blobKey, nil
}

// MarkBlob mark a blob
func MarkBlob(store kv.Store, blobKey blob.Key, mark string) error {
	key := makeMarkKey(blobKey, mark)
	return errors.Wrap(store.Put(key, []byte{}), "mark blob")
}

// UnmarkBlob delete mark to a blob
func UnmarkBlob(store kv.Store, blobKey blob.Key, mark string) error {
	key := makeMarkKey(blobKey, mark)
	return errors.Wrap(store.Delete(key), "unmark blob")
}

// NewMarkIterator returns an iterator for all blob keys marked with mark
func NewMarkIterator(store kv.Store, mark string) *MarkIterator {
	prefix := markPrefix + mark
	rng := kv.NewRangeWithBytesPrefix([]byte(prefix))
	return &MarkIterator{
		mark: mark,
		it:   store.NewIterator(rng),
	}
}

// MarkIterator iterates marked blob keys
type MarkIterator struct {
	mark string
	it   kv.Iterator
}

// Next move iterator next
func (mi *MarkIterator) Next() bool {
	return mi.it.Next()
}

// Release release the iterator
func (mi *MarkIterator) Release() {
	mi.it.Release()
}

// Error returns error occurred
func (mi *MarkIterator) Error() error {
	return errors.Wrap(mi.it.Error(), "iterate mark")
}

// BlobKey returns blob key
func (mi *MarkIterator) BlobKey() (*blob.Key, error) {
	storeKey := mi.it.Key()
	blobKey, err := extractBlobKey(storeKey, mi.mark)
	if err != nil {
		return nil, errors.Wrap(err, "iterate mark")
	}
	return blobKey, nil
}
