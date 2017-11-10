package kv_test

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	. "github.com/vechain/solidb/kv"
)

func TestBasic(t *testing.T) {
	assert := assert.New(t)
	dbPath, _ := ioutil.TempDir(os.TempDir(), "db")
	defer os.RemoveAll(dbPath)

	db, _ := NewStore(dbPath, Options{})
	defer db.Close()

	cases := []struct {
		op    string
		key   string
		value []byte
	}{
		{op: "get", key: "k1"},
		{op: "put", key: "k1", value: []byte("v")},
		{op: "get", key: "k1", value: []byte("v")},
		{op: "del", key: "k1"},
		{op: "get", key: "k1"},
	}

	for _, c := range cases {
		switch c.op {
		case "put":
			db.Put([]byte(c.key), c.value)
		case "get":
			opt, _ := db.Get([]byte(c.key))
			assert.Equal(opt.V, c.value)
		case "del":
			db.Delete([]byte(c.key))
		}
	}
}

func BenchmarkStore(b *testing.B) {
	dbPath, _ := ioutil.TempDir(os.TempDir(), "db")
	defer os.RemoveAll(dbPath)

	db, _ := NewStore(dbPath, Options{})
	defer db.Close()

	key := make([]byte, 32)
	value := make([]byte, 65536)

	b.Run("put", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			kl := rand.Int()%len(key) + 1
			vl := rand.Int()%len(value) + 1

			k := key[:kl]
			rand.Read(k)
			v := (value[:vl])
			rand.Read(v)

			db.Put(k, v)
		}
	})

	b.Run("get", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			kl := rand.Int()%len(key) + 1

			k := key[:kl]
			rand.Read(k)

			db.Get(k)
		}
	})

	b.Run("batch-put", func(b *testing.B) {

		batch := db.NewBatch()
		for i := 0; i < b.N; i++ {
			kl := rand.Int()%len(key) + 1
			vl := rand.Int()%len(value) + 1

			k := key[:kl]
			rand.Read(k)
			v := (value[:vl])
			rand.Read(v)

			batch.Put(k, v)
			if batch.Len() >= 50 {
				batch.Write()
				batch.Reset()
			}
		}
	})
	b.Run("iterator", func(b *testing.B) {
		prefix := make([]byte, 1)
		for i := 0; i < b.N; i++ {
			rand.Read(prefix)
			iter := db.NewIterator(NewRangeWithBytesPrefix(prefix))
			iter.Next()
			defer iter.Release()
		}
	})
}
