package crypto_test

import (
	"testing"

	. "github.com/vechain/solidb/crypto"
)

func BenchmarkHashSum(b *testing.B) {
	b.Run("64k", func(b *testing.B) {
		var data [65536]byte
		for i := 0; i < b.N; i++ {
			HashSum(data[:])
		}
	})
}
