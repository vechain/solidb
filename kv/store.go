// Package kv provides key-value based storage.
package kv

type OptValue struct {
	V []byte
}

// Writer kv writing methods
type Writer interface {
	// Put store value for the given key.
	Put(key []byte, value []byte) error

	// Delete delete value for the given key.
	Delete(key []byte) error
}

// Reader kv reading methods
type Reader interface {
	// Has returns whether a key exists
	Has(key []byte) (bool, error)

	// Get retrieve value for the given key.
	Get(key []byte) (*OptValue, error)
}

// Store interface of key-value storage.
type Store interface {
	Reader
	Writer

	// NewBatch create batch object for batch writes.
	NewBatch() Batch

	// NewIterator create iterator to iterates kv pairs for the given range.
	NewIterator(r *Range) Iterator

	// Close close the store.
	Close() error
}

// Batch interface of batched operation.
type Batch interface {
	Writer

	// Reset clear pending operations.
	Reset()

	// Write flush pending operations.
	// It will not clear pending operations.
	Write() error

	// Len returns pending operations count.
	Len() int
}

// Iterator see iterator in leveldb.
type Iterator interface {
	First() bool
	Last() bool
	Seek(key []byte) bool
	Next() bool
	Prev() bool

	Release()

	Error() error

	Key() []byte
	Value() []byte
}

// Options options to initialize store
type Options struct {
	CacheSize              int
	OpenFilesCacheCapacity int
}

// NewStore create/open kv store at specified file path.
func NewStore(filePath string, options Options) (Store, error) {
	return newFSLevelDB(
		filePath,
		options.CacheSize,
		options.OpenFilesCacheCapacity)
}

// NewMemStore create kv store in memory, for test purpose
func NewMemStore(options Options) (Store, error) {
	return newMemLevelDB(options.CacheSize)
}
