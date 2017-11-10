package kv_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	. "github.com/vechain/solidb/kv"
)

func TestRange(t *testing.T) {
	assert := assert.New(t)

	r1, _ := NewRangeWithHexPrefix("a")

	from, _ := hex.DecodeString("a0")
	to, _ := hex.DecodeString("b0")
	r2 := NewRange(from, to)

	assert.Equal(r1, r2)
}
