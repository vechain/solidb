package crypto

import (
	"encoding/hex"
	"encoding/json"

	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

// HashLength length of hash in bytes
const HashLength = 32

// Hash hash type
type Hash [HashLength]byte

// ToHex convert hash into hex string (without '0x' prefix)
func (h Hash) ToHex() string {
	return hex.EncodeToString(h[:])
}

// UnmarshalJSON unmarshal JSON
func (h *Hash) UnmarshalJSON(b []byte) error {
	var hex string
	if err := json.Unmarshal(b, &hex); err != nil {
		return errors.Wrap(err, "unmarshal hash")
	}
	hash, err := HexToHash(hex)
	if err != nil {
		return errors.Wrap(err, "unmarshal hash")
	}
	*h = *hash
	return nil
}

// MarshalJSON Marshal JSON
func (h Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.ToHex())
}

// HexToHash parse hex string to hash
func HexToHash(str string) (*Hash, error) {

	bin, err := hex.DecodeString(str)
	if err != nil {
		return nil, errors.Wrap(err, "hex to hash")
	}

	if len(bin) != HashLength {
		return nil, errors.New("invalid hash length")
	}
	var hash Hash
	copy(hash[:], bin)
	return &hash, nil
}

// HashSum compute hash of data
// default algorithm is blake2b
func HashSum(data []byte) (hash Hash) {
	return blake2b.Sum256(data)
}
