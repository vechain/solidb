// Package specmgr manages cluster specs in kv store.
package specmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/vechain/solidb/kv"
	"github.com/vechain/solidb/spec"
)

// predefined spec tags
const (
	TagSynced   = "synced"
	TagApproved = "approved"
)
const (
	revisionPrefix = ".spec/rev/"
	tagPrefix      = ".spec/tags/"
)

func makeRevisionKey(revision int) []byte {
	return []byte(revisionPrefix + fmt.Sprintf("%010d", revision))
}

// SpecManager to manage cluster specs
type SpecManager struct {
	store kv.Store

	newestCache *spec.Spec
	taggedCache struct {
		sync.Mutex
		cache map[string]*spec.Spec
	}
}

// New create a spec manager instance
func New(store kv.Store) *SpecManager {
	m := &SpecManager{
		store: store,
	}
	m.taggedCache.cache = make(map[string]*spec.Spec)
	return m
}

// GetByRevision get spec by revision
func (m *SpecManager) GetByRevision(revision int) (*spec.OptSpec, error) {
	key := makeRevisionKey(revision)
	value, err := m.store.Get(key)
	if err != nil {
		return nil, errors.Wrap(err, "get by revision")
	}
	if value.V == nil {
		return &spec.OptSpec{}, nil
	}
	var s spec.Spec
	if err := json.Unmarshal(value.V, &s); err != nil {
		return nil, errors.Wrap(err, "get by revision")
	}
	return &spec.OptSpec{V: &s}, nil
}

// Commit store a spec, later the spec can be retrieved by its revision.
func (m *SpecManager) Commit(s spec.Spec) error {
	data, err := json.Marshal(&s)
	if err != nil {
		return errors.Wrap(err, "commit")
	}

	revKey := makeRevisionKey(s.Revision)

	od, err := m.store.Get(revKey)
	if err != nil {
		return errors.Wrap(err, "commit")
	}
	if od.V != nil {
		if !bytes.Equal(od.V, data) {
			return errors.New("commit: inconsistent spec")
		}
		return nil
	}

	if err := m.store.Put(revKey, data); err != nil {
		return errors.Wrap(err, "commit")
	}
	m.newestCache = nil
	return nil
}

// GetNewest returns spec with largest revision number
func (m *SpecManager) GetNewest() (*spec.OptSpec, error) {
	if s := m.newestCache; s != nil {
		return &spec.OptSpec{V: s}, nil
	}

	rng := kv.NewRangeWithBytesPrefix([]byte(revisionPrefix))
	iter := m.store.NewIterator(rng)
	defer iter.Release()
	if iter.Last() {
		data := iter.Value()
		var s spec.Spec
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, errors.Wrap(err, "get newest")
		}
		m.newestCache = &s
		return &spec.OptSpec{V: &s}, nil
	}
	return &spec.OptSpec{}, nil
}

func (m *SpecManager) getCached(tag string) *spec.Spec {
	m.taggedCache.Lock()
	defer m.taggedCache.Unlock()
	return m.taggedCache.cache[tag]
}

func (m *SpecManager) setCached(tag string, s *spec.Spec) {
	m.taggedCache.Lock()
	defer m.taggedCache.Unlock()
	m.taggedCache.cache[tag] = s
}

// Tag tag a spec by its revision
func (m *SpecManager) Tag(revision int, tag string) error {
	hasRev, err := m.store.Has(makeRevisionKey(revision))
	if err != nil {
		return errors.Wrap(err, "tag")
	}
	if !hasRev {
		return errors.New("tag: revision not found")
	}
	revStr := strconv.Itoa(revision)

	if err := m.store.Put([]byte(tagPrefix+tag), []byte(revStr)); err != nil {
		return errors.Wrap(err, "tag")
	}
	m.setCached(tag, nil)
	return nil
}

// GetByTag retrieve spec by tag
func (m *SpecManager) GetByTag(tag string) (*spec.OptSpec, error) {
	if s := m.getCached(tag); s != nil {
		return &spec.OptSpec{V: s}, nil
	}
	revStr, err := m.store.Get([]byte(tagPrefix + tag))
	if err != nil {
		return nil, errors.Wrap(err, "get by tag")
	}
	if revStr.V == nil {
		return &spec.OptSpec{}, nil
	}
	rev, err := strconv.Atoi(string(revStr.V))
	if err != nil {
		return nil, errors.Wrap(err, "get by tag")
	}
	c, err := m.GetByRevision(rev)
	if err != nil {
		return nil, errors.Wrap(err, "get by tag")
	}
	m.setCached(tag, c.V)
	return c, nil
}
