package kv

import (
	"encoding/hex"

	"github.com/pkg/errors"
)

// Range describes key range of kv store.
type Range struct {
	from []byte
	to   []byte
}

// NewRange create a range.
func NewRange(from []byte, to []byte) *Range {
	return &Range{
		from: from,
		to:   to,
	}
}

// NewRangeWithBytesPrefix create a range defined by bytes prefix.
func NewRangeWithBytesPrefix(prefix []byte) *Range {
	var limit []byte
	for i := len(prefix) - 1; i >= 0; i-- {
		c := prefix[i]
		if c < 0xff {
			limit = make([]byte, i+1)
			copy(limit, prefix)
			limit[i] = c + 1
			break
		}
	}
	return &Range{from: prefix, to: limit}
}

// NewRangeWithHexPrefix create a range defined by hex prefix.
// The hex can be odd.
func NewRangeWithHexPrefix(hexPrefix string) (*Range, error) {
	if len(hexPrefix)%2 > 0 {
		// odd hex
		start, err := hex.DecodeString(hexPrefix + "0")
		if err != nil {
			return nil, errors.Wrap(err, "new range")
		}
		end, err := hex.DecodeString(hexPrefix + "f")
		if err != nil {
			return nil, errors.Wrap(err, "new range")
		}
		return &Range{
			from: start,
			to:   NewRangeWithBytesPrefix(end).to,
		}, nil
	}
	// even hex
	prefix, err := hex.DecodeString(hexPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "new range")
	}

	return NewRangeWithBytesPrefix(prefix), nil
}
