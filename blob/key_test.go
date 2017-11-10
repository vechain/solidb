package blob_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	. "github.com/vechain/solidb/blob"
)

func TestKey(t *testing.T) {
	assert := assert.New(t)

	key := KeyOfData([]byte{})
	_key, _ := ParseHexKey(key.ToHex())
	assert.Equal(key, *_key)

	data, _ := json.Marshal(&key)
	assert.Equal(string(data), "\""+key.ToHex()+"\"")

	var k Key
	json.Unmarshal([]byte("\""+key.ToHex()+"\""), &k)
	assert.Equal(k, key)
}
