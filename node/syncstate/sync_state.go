// Package syncstate provides functions to maintain sync state
package syncstate

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/vechain/solidb/kv"
)

const syncedSlicePrefx = ".synced-slice/"

func makeKey(slice string) []byte {
	return []byte(syncedSlicePrefx + slice)
}

// SetSlicesSynced mark slices synced state.
// If exclusive set to true, other slices will be set to unsynced.
func SetSlicesSynced(store kv.Store, exclusive bool, slices ...string) error {
	batch := store.NewBatch()
	if exclusive {
		synced, err := GetSyncedSlices(store)
		if err != nil {
			return err
		}
		// delete all synced
		for _, s := range synced {
			batch.Delete(makeKey(s))
		}
	}

	for _, s := range slices {
		batch.Put(makeKey(s), []byte{})
	}
	return errors.Wrap(batch.Write(), "set slices synced")
}

// GetSyncedSlices returns synced slices
func GetSyncedSlices(store kv.Store) ([]string, error) {
	var slices []string
	iter := store.NewIterator(kv.NewRangeWithBytesPrefix([]byte(syncedSlicePrefx)))
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()
		slices = append(slices, string(key[len(syncedSlicePrefx):]))
	}
	if err := iter.Error(); err != nil {
		return nil, errors.Wrap(err, "get synced slices")
	}
	return slices, nil
}

// GetUnsyncedSlices returns unsynced slices according to alloced slices
func GetUnsyncedSlices(store kv.Store, allocedSlices []string) ([]string, error) {
	syncedSlices, err := GetSyncedSlices(store)
	if err != nil {
		return nil, err
	}

	var unsyncedSlices []string
	for _, alloced := range allocedSlices {
		found := false
		for _, synced := range syncedSlices {
			if strings.HasPrefix(alloced, synced) {
				found = true
				break
			}
		}
		if !found {
			unsyncedSlices = append(unsyncedSlices, alloced)
		}
	}
	return unsyncedSlices, nil
}
