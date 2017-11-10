package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	. "github.com/vechain/solidb/crypto"
)

func TestIdentity(t *testing.T) {
	assert := assert.New(t)

	identity1, _ := GenerateIdentity()
	identity2, _ := NewIdentity(identity1.PrivateKey())

	assert.Equal(identity1, identity2)
}

func TestSign(t *testing.T) {
	assert := assert.New(t)

	i1, _ := GenerateIdentity()
	hash := HashSum([]byte("hello world"))

	sig, _ := i1.Sign(hash)
	id, _ := RecoverID(hash, sig)
	assert.Equal(id, i1.ID())
}

func BenchmarkSign(b *testing.B) {
	hash := HashSum([]byte("hello world"))
	id, _ := GenerateIdentity()
	for i := 0; i < b.N; i++ {
		id.Sign(hash)
	}
}

func BenchmarkRecoverID(b *testing.B) {
	hash := HashSum([]byte("hello world"))
	id, _ := GenerateIdentity()
	sig, _ := id.Sign(hash)
	for i := 0; i < b.N; i++ {
		RecoverID(hash, sig)
	}
}
