// Package blob defines content-addressable type to contain arbitrary data.
package blob

const DataLenHardLimit = 64 * 1024

// Blob data type stored in solidb
type Blob struct {
	data      []byte
	cachedKey *Key
}

// New construct a blob
func New(data []byte) *Blob {
	return &Blob{data: data}
}

// Data get blob data
func (blob *Blob) Data() []byte {
	return blob.data
}

// Key compute key of blob data
// the computed key is cached
func (blob *Blob) Key() Key {
	if key := blob.cachedKey; key != nil {
		return *key
	}
	key := KeyOfData(blob.data)
	blob.cachedKey = &key
	return key
}
