package node

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vechain/solidb/blob"
	"github.com/vechain/solidb/blobio"
	"github.com/vechain/solidb/spec"
	"github.com/vechain/solidb/specmgr"
)

func (n *Node) healFaults(ctx context.Context) error {
	var (
		nIter   = 0
		nHealed = 0
	)
	defer func() {
		if nIter > 0 {
			log.Infof("head faults: healed %d/%d", nHealed, nIter)
		}
	}()

	iter := blobio.NewMarkIterator(n.store, blobio.FaultBlobMark)
	defer iter.Release()
	for iter.Next() {
		nIter++
		blobKey, err := iter.BlobKey()
		if err != nil {
			log.Warnf("heal faults: %v", err)
			continue
		}
		blob, err := blobio.GetBlob(n.store, *blobKey)
		if err != nil {
			log.Warnf("heal faults: %v", err)
			continue
		}
		if blob.V == nil {
			log.Warn("heal faults: blob not found locally")
			continue
		}
		entries, err := n.locateBlob(*blobKey)
		if err != nil {
			return err
		}

		if err := n.broadcastBlob(ctx, blob.V, entries); err != nil {
			return err
		}
		if err := blobio.UnmarkBlob(n.store, *blobKey, blobio.FaultBlobMark); err != nil {
			return err
		}
		nHealed++
	}
	return iter.Error()
}

// locateBlob locate which entries the blob stored according to approved spec
func (n *Node) locateBlob(blobKey blob.Key) ([]spec.Entry, error) {
	approved, err := n.specMgr.GetByTag(specmgr.TagApproved)
	if err != nil {
		return nil, err
	}
	if approved.V == nil {
		return nil, errors.New("no approved spec")
	}
	newest, err := n.specMgr.GetNewest()
	if err != nil {
		return nil, err
	}
	if newest.V == nil {
		return nil, errors.New("no spec")
	}

	entries := approved.V.SAT.Entries
	if newest.V.Revision != approved.V.Revision {
		// merge approved entries and newest entries
		for _, entry := range newest.V.SAT.Entries {
			if approved.V.SAT.FindEntry(entry.ID) == nil {
				entries = append(entries, entry)
			}
		}
	}
	return entries, nil
}

// broadcastBlob put blob to nodes it belongs to
func (n *Node) broadcastBlob(ctx context.Context, blob *blob.Blob, entries []spec.Entry) error {
	// TODO concurrent
	rpc := NewRPC().WithContext(ctx)
	for _, entry := range entries {
		if entry.ID == n.ID() {
			// skip local node
			continue
		}
		rpc := rpc.WithAddr(entry.Addr)
		if err := rpc.PutBlob(blob); err != nil {
			return err
		}
	}
	return nil
}
