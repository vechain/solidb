package kv

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var writeOpt = &opt.WriteOptions{
	Sync: true, // to prevent data loss
}

// implements Store interface
type levelDB struct {
	db *leveldb.DB
}

func newLevelDB(s storage.Storage, cacheSize int, openFilesCacheCapacity int) (*levelDB, error) {
	if cacheSize < 128 {
		cacheSize = 128
	}

	if openFilesCacheCapacity < 64 {
		openFilesCacheCapacity = 64
	}

	db, err := leveldb.Open(s, &opt.Options{
		OpenFilesCacheCapacity: openFilesCacheCapacity,
		BlockCacheCapacity:     cacheSize / 2 * opt.MiB,
		WriteBuffer:            cacheSize / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})

	if err != nil {
		return nil, errors.Wrap(err, "new level db")
	}
	return &levelDB{db: db}, nil
}

func newMemLevelDB(cacheSize int) (*levelDB, error) {
	s := storage.NewMemStorage()
	return newLevelDB(s, cacheSize, 0)
}

func newFSLevelDB(filePath string, cacheSize int, openFilesCacheCapacity int) (*levelDB, error) {
	s, err := storage.OpenFile(filePath, false)
	if err != nil {
		return nil, errors.Wrap(err, "new fs level db")
	}
	return newLevelDB(s, cacheSize, openFilesCacheCapacity)
}

func (ldb *levelDB) Has(key []byte) (bool, error) {
	return ldb.db.Has(key, nil)
}

func (ldb *levelDB) Get(key []byte) (*OptValue, error) {
	data, err := ldb.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return &OptValue{}, nil
		}
		return nil, errors.Wrap(err, "get")
	}
	return &OptValue{data}, nil
}

func (ldb *levelDB) Put(key []byte, value []byte) error {
	return errors.Wrap(ldb.db.Put(key, value, writeOpt), "put")
}

func (ldb *levelDB) NewIterator(r *Range) Iterator {
	return ldb.db.NewIterator(&util.Range{Start: r.from, Limit: r.to}, nil)
}

func (ldb *levelDB) Delete(key []byte) error {
	if err := ldb.db.Delete(key, writeOpt); err != nil {
		return errors.Wrap(err, "delete")
	}
	return nil
}

func (ldb *levelDB) Close() error {
	return errors.Wrap(ldb.db.Close(), "close")
}

func (ldb *levelDB) NewBatch() Batch {
	return &levelDBBatch{
		db:    ldb.db,
		batch: &leveldb.Batch{},
	}
}

// implements Batch interface
type levelDBBatch struct {
	db    *leveldb.DB
	batch *leveldb.Batch
}

func (batch *levelDBBatch) Delete(key []byte) error {
	batch.batch.Delete(key)
	return nil
}

func (batch *levelDBBatch) Put(key []byte, value []byte) error {
	batch.batch.Put(key, value)
	return nil
}

func (batch *levelDBBatch) Reset() {
	batch.batch.Reset()
}

func (batch *levelDBBatch) Write() error {
	return errors.Wrap(batch.db.Write(batch.batch, writeOpt), "write batch")
}

func (batch *levelDBBatch) Len() int {
	return batch.batch.Len()
}
