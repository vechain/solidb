// Package crypto provides crypto algos
package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"math/big"

	"github.com/pkg/errors"
)

var curve = elliptic.P256()

type signature struct {
	R   *big.Int
	S   *big.Int
	Pub []byte
}

// publicKeyToID convert public key to ID in string.
func publicKeyToID(pub []byte) string {
	hash := HashSum(pub)
	return hex.EncodeToString(hash[12:])
}

// Identity wrap ECDSA private key to identify some one.
type Identity struct {
	privKey *ecdsa.PrivateKey

	cachedID string
}

// GenerateIdentity generate a new identity
func GenerateIdentity() (*Identity, error) {
	privKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "generate identity")
	}
	return &Identity{privKey: privKey}, nil
}

// NewIdentity create identity from private key.
func NewIdentity(privKey []byte) (*Identity, error) {
	if len(privKey) != curve.Params().BitSize/8 {
		return nil, errors.New("invalid private key length")
	}
	priv := ecdsa.PrivateKey{}
	priv.Curve = curve
	priv.D = new(big.Int)
	priv.D.SetBytes(privKey)
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(privKey)
	return &Identity{privKey: &priv}, nil
}

// PrivateKey returns private key in bytes.
func (identity *Identity) PrivateKey() []byte {
	return identity.privKey.D.Bytes()
}

// PublicKey returns public key in bytes.
func (identity *Identity) publicKey() []byte {
	return elliptic.Marshal(curve, identity.privKey.X, identity.privKey.Y)
}

// ID returns ID of identity in string.
// The ID is derived from public key, and the result is cached.
func (identity *Identity) ID() string {
	id := identity.cachedID
	if id != "" {
		return id
	}
	id = publicKeyToID(identity.publicKey())
	identity.cachedID = id
	return id
}

// Sign sign message hash and returns signature.
func (identity *Identity) Sign(msgHash Hash) ([]byte, error) {
	r, s, err := ecdsa.Sign(rand.Reader, identity.privKey, msgHash[:])
	if err != nil {
		return nil, errors.Wrap(err, "sign")
	}
	sig := signature{
		R:   r,
		S:   s,
		Pub: identity.publicKey(),
	}
	data, err := json.Marshal(&sig)
	if err != nil {
		return nil, errors.Wrap(err, "sign")
	}
	return data, nil
}

// RecoverID recover ID from message hash and signature.
func RecoverID(msgHash Hash, sig []byte) (string, error) {
	var sigt signature
	if err := json.Unmarshal(sig, &sigt); err != nil {
		return "", errors.Wrap(err, "recover id")
	}
	x, y := elliptic.Unmarshal(curve, sigt.Pub)
	if !ecdsa.Verify(&ecdsa.PublicKey{X: x, Y: y, Curve: curve}, msgHash[:], sigt.R, sigt.S) {
		return "", errors.New("recover id: verify signature failed")
	}

	return publicKeyToID(sigt.Pub), nil
}

// AbbrevID returns abbreviation of ID
func AbbrevID(id string) string {
	if len(id) == 40 {
		return id[:4] + "…" + id[len(id)-4:]
	}
	return id
}
