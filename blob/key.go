package blob

import (
	"encoding/hex"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/vechain/solidb/crypto"
)

// KeyLength length of blob key. It's shorter then crypto hash length for extension purpose.
const KeyLength = crypto.HashLength - 1

// EmptyKey a key with all zero bytes
var EmptyKey Key

// Key key of blob, derrived from blob data
type Key [KeyLength]byte

// ToHex convert key into hex string (without '0x' prefix)
func (k Key) ToHex() string {
	return hex.EncodeToString(k[:])
}

// ParseHexKey parse hex string to blob key
func ParseHexKey(str string) (*Key, error) {
	bin, err := hex.DecodeString(str)
	if err != nil {
		return nil, errors.Wrap(err, "parse hex key")
	}

	if len(bin) != KeyLength {
		return nil, errors.Wrap(errors.New("invalid length"), "parse hex key")
	}
	var key Key
	copy(key[:], bin)
	return &key, nil
}

// KeyOfData compute key of data
func KeyOfData(data []byte) Key {
	hash := crypto.HashSum(data)
	key := Key{}
	copy(key[:], hash[:KeyLength])
	return key
}

// UnmarshalJSON unmarshal JSON
func (k *Key) UnmarshalJSON(b []byte) error {
	var hex string
	if err := json.Unmarshal(b, &hex); err != nil {
		return errors.Wrap(err, "unmarshal key")
	}
	key, err := ParseHexKey(hex)
	if err != nil {
		return errors.Wrap(err, "unmarshal key")
	}
	*k = *key
	return nil
}

// MarshalJSON Marshal JSON
func (k Key) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(k.ToHex())
	if err != nil {
		return nil, errors.Wrap(err, "marshal key")
	}
	return data, nil
}
