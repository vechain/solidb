package node

import (
	"context"
	"math/rand"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vechain/solidb/blobio"
	"github.com/vechain/solidb/node/syncstate"
	"github.com/vechain/solidb/specmgr"
)

// syncSlices sync allocated slices defined in spec at revision
func (n *Node) syncSlices(ctx context.Context, revision int) error {
	entry, err := n.satEntry(revision)
	if err != nil {
		return err
	}
	if entry.V == nil {
		return errors.Errorf("no entry @rev%d", revision)
	}

	approved, err := n.specMgr.GetByTag(specmgr.TagApproved)
	if err != nil {
		return err
	}
	if approved.V == nil {
		return errors.Errorf("no approved spec")
	}

	unsyncedSlices, err := syncstate.GetUnsyncedSlices(n.store, entry.V.Slices)
	if err != nil {
		return err
	}

	var syncedSlices []string
	for _, unsyncedSlice := range unsyncedSlices {
		// only rely on nodes in approved spec
		entries := approved.V.SAT.Locate(unsyncedSlice)

		// shuffle the nodes to spread network traffic and disk IO
		for _, i := range rand.Perm(len(entries)) {

			entry := entries[i]
			log.Infof("syncing slice %s from %s ...", unsyncedSlice, entry.Addr)
			count, err := n.importRemoteBlobSlice(ctx, entry.Addr, unsyncedSlice)
			if err != nil {
				log.Warnf("sync slice %s from %s: %v", unsyncedSlice, entry.Addr, err)
				// try another node
				continue
			}

			if err := syncstate.SetSlicesSynced(n.store, false, unsyncedSlice); err != nil {
				return err
			}

			syncedSlices = append(syncedSlices, unsyncedSlice)
			log.Infof("synced slice %s from %s: blob count %d", unsyncedSlice, entry.Addr, count)
			// slice synced
			break
		}
	}

	if len(syncedSlices) != len(unsyncedSlices) {
		return errors.New("not full synced")
	}
	return n.specMgr.Tag(revision, specmgr.TagSynced)
}

// importRemoteBlobSlice
func (n *Node) importRemoteBlobSlice(ctx context.Context, remoteAddr string, prefix string) (int, error) {
	rpc := NewRPC().WithContext(ctx).WithAddr(remoteAddr)
	reader, err := rpc.GetBlobSlice(prefix)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	nCount := 0
	// use batch to optimize write performance
	const batchLen = 100
	batch := n.store.NewBatch()
	for {
		blob, err := blobio.ReadBlob(reader)
		if err != nil {
			return 0, err
		}
		if blob.V == nil {
			break
		}
		nCount++
		if err := blobio.PutBlob(batch, blob.V); err != nil {
			return 0, err
		}
		if batch.Len() >= batchLen {
			if err := batch.Write(); err != nil {
				return 0, err
			}
			batch.Reset()
		}
	}
	if batch.Len() > 0 {
		if err := batch.Write(); err != nil {
			return 0, err
		}
	}
	return nCount, nil
}
