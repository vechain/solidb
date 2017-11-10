package node

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vechain/solidb/crypto"
	"github.com/vechain/solidb/kv"
	"github.com/vechain/solidb/node/syncstate"
	"github.com/vechain/solidb/spec"
	"github.com/vechain/solidb/specmgr"
)

var (
	nodeKeyKey   = []byte(".node-key")
	clusterIDKey = []byte(".cluster-id")
)

func getOrGenerateNodeKey(store kv.Store) (*crypto.Identity, error) {
	value, err := store.Get(nodeKeyKey)
	if err != nil {
		return nil, err
	}
	if value.V != nil {
		return crypto.NewIdentity(value.V)
	}

	identity, err := crypto.GenerateIdentity()
	if err != nil {
		return nil, err
	}
	data := identity.PrivateKey()
	if err := store.Put(nodeKeyKey, data); err != nil {
		return nil, err
	}
	return identity, nil
}

// Node defines local node of solidb.
type Node struct {
	store              kv.Store
	identity           *crypto.Identity
	clusterID          string
	specMgr            *specmgr.SpecManager
	syncRequest        chan int
	lastSyncRequestRev int

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates node instance
func New(store kv.Store, specMgr *specmgr.SpecManager) (*Node, error) {
	identity, err := getOrGenerateNodeKey(store)
	if err != nil {
		return nil, err
	}
	clusterIDData, err := store.Get(clusterIDKey)
	if err != nil {
		return nil, err
	}

	return &Node{
		store:     store,
		identity:  identity,
		clusterID: string(clusterIDData.V),

		specMgr:     specMgr,
		syncRequest: make(chan int),
	}, nil
}

// Start start running node
func (n *Node) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	n.cancel = cancel
	n.wg.Add(2)
	go n.syncSlicesLoop(ctx)
	go n.healFaultsLoop(ctx)
}

// Shutdown terminate running node and block until stopped.
func (n *Node) Shutdown() {
	n.cancel()
	n.wg.Wait()
}

// satEntry return SAT entry of local node according to revision
func (n *Node) satEntry(revision int) (*spec.OptEntry, error) {
	c, err := n.specMgr.GetByRevision(revision)
	if err != nil {
		return nil, err
	}
	if c.V == nil {
		return nil, errors.Errorf("no spec @rev%d", revision)
	}

	entry := c.V.SAT.FindEntry(n.ID())
	if entry == nil {
		return &spec.OptEntry{}, nil
	}
	return &spec.OptEntry{V: entry}, nil
}

// syncSlicesLoop
func (n *Node) syncSlicesLoop(ctx context.Context) {
	log.Info("enter sync loop")
	defer func() {
		if err := recover(); err != nil {
			log.Warnln("sync slices loop recovered:", err)
		}
		n.wg.Done()
		log.Info("leave sync loop")
	}()

	for {
		select {
		case rev := <-n.syncRequest:
			log.Infof("sync slices: start @rev%d", rev)
			if err := n.syncSlices(ctx, rev); err != nil {
				log.Errorf("sync slices: %v", err)
			} else {
				log.Infof("sync slices: completed @rev%d", rev)
			}
		case <-ctx.Done():
			return
		}
	}
}

// healFaultsLoop
func (n *Node) healFaultsLoop(ctx context.Context) {
	log.Info("enter faults-healing loop")

	gap := time.Second * 10
	timer := time.NewTimer(gap)
	defer func() {
		if err := recover(); err != nil {
			log.Warnln("heal faults loop recovered:", err)
		}
		timer.Stop()
		n.wg.Done()
		log.Info("leave faults-healing loop")
	}()

	for {
		select {
		case <-timer.C:
			if err := n.healFaults(ctx); err != nil {
				log.Errorf("faults-healing: %v", err)
			}
			timer.Reset(gap)
		case <-ctx.Done():
			return
		}
	}
}

// ID returns ID of node.
func (n *Node) ID() string {
	return n.identity.ID()
}

// ClusterID returns cluster ID the node belongs to.
func (n *Node) ClusterID() string {
	return n.clusterID
}

// Invite invite the node to join a cluster.
func (n *Node) Invite(clusterID string, initSpec *spec.Spec) error {
	if n.clusterID != "" {
		if n.clusterID != clusterID {
			return errors.New("already in cluster")
		}
		// allow re-enter
		approved, err := n.specMgr.GetByTag(specmgr.TagApproved)
		if err != nil {
			return err
		}
		if approved.V != nil {
			if initSpec != nil && initSpec.Revision != approved.V.Revision {
				return errors.New("rejected")
			}
		}
	}

	if initSpec != nil {
		if err := n.specMgr.Commit(*initSpec); err != nil {
			return err
		}
		if err := n.specMgr.Tag(initSpec.Revision, specmgr.TagApproved); err != nil {
			return err
		}
	}

	if err := n.store.Put(clusterIDKey, []byte(clusterID)); err != nil {
		return err
	}
	n.clusterID = clusterID
	return nil
}

// GetStatus returns node status.
func (n *Node) GetStatus() (*StatusResponse, error) {
	newestRev := -1
	syncedRev := -1
	approvedRev := -1

	if newest, err := n.specMgr.GetNewest(); err != nil {
		return nil, err
	} else if newest.V != nil {
		newestRev = newest.V.Revision
	}
	if synced, err := n.specMgr.GetByTag(specmgr.TagSynced); err != nil {
		return nil, err
	} else if synced.V != nil {
		syncedRev = synced.V.Revision
	}
	if approved, err := n.specMgr.GetByTag(specmgr.TagApproved); err != nil {
		return nil, err
	} else if approved.V != nil {
		approvedRev = approved.V.Revision
	}

	return &StatusResponse{
		NodeID:    n.ID(),
		ClusterID: n.ClusterID(),
		SpecRevisions: Revisions{
			Newest:   newestRev,
			Synced:   syncedRev,
			Approved: approvedRev,
		},
	}, nil
}

// GetSyncStatus returns progress of slice syncing
func (n *Node) GetSyncStatus(revision int) (*SyncStatusResponse, error) {
	entry, err := n.satEntry(revision)
	if err != nil {
		return nil, err
	}
	if entry.V == nil {
		return &SyncStatusResponse{}, nil
	}

	unsynced, err := syncstate.GetUnsyncedSlices(n.store, entry.V.Slices)
	if err != nil {
		return nil, err
	}

	return &SyncStatusResponse{
		SyncedSliceCount: len(entry.V.Slices) - len(unsynced),
		TotalSliceCount:  len(entry.V.Slices),
	}, nil
}

// ProposeSpec propose a new spec.
// The revision of proposed one should be >= newest.
func (n *Node) ProposeSpec(s spec.Spec) error {
	if newest, err := n.specMgr.GetNewest(); err != nil {
		return err
	} else if newest.V != nil {
		if s.Revision < newest.V.Revision {
			return errors.New("revision too low")
		}
	}

	if err := n.specMgr.Commit(s); err != nil {
		return err
	}

	if s.Revision == 0 {
		// special case for revision 0
		entry, err := n.satEntry(s.Revision)
		if err != nil {
			return err
		}
		if entry.V == nil {
			return errors.Errorf("no entry @rev%d", s.Revision)
		}
		if err := syncstate.SetSlicesSynced(n.store, false, entry.V.Slices...); err != nil {
			return err
		}
		if err := n.specMgr.Tag(s.Revision, specmgr.TagSynced); err != nil {
			return err
		}
		if err := n.specMgr.Tag(0, specmgr.TagApproved); err != nil {
			return err
		}
	}
	return nil
}

// RequestSync request to sync slices according to spec specified by revision.
func (n *Node) RequestSync(revision int) error {
	synced, err := n.specMgr.GetByTag(specmgr.TagSynced)
	if err != nil {
		return err
	}
	if synced.V != nil {
		if revision < synced.V.Revision {
			return errors.New("revision too low")
		}
	}

	c, err := n.specMgr.GetByRevision(revision)
	if err != nil {
		return err
	}
	if c.V == nil {
		return errors.New("spec not found")
	}

	select {
	case n.syncRequest <- revision:
		n.lastSyncRequestRev = revision
		return nil
	default:
		if n.lastSyncRequestRev == revision {
			return nil
		}
		return errors.New("pending")
	}
}

// ApproveSpec approve spec by revision.
// The spec to be approved should be synced.
func (n *Node) ApproveSpec(revision int) error {
	synced, err := n.specMgr.GetByTag(specmgr.TagSynced)
	if err != nil {
		return err
	}
	if synced.V == nil {
		return errors.New("no synced spec")
	}

	if synced.V.Revision != revision {
		return errors.New("spec not synced")
	}

	if err := n.specMgr.Tag(revision, specmgr.TagApproved); err != nil {
		return err
	}
	entry, err := n.satEntry(revision)
	if err != nil {
		return err
	}
	if entry.V == nil {
		return errors.New("not in cluster")
	}
	return syncstate.SetSlicesSynced(n.store, true, entry.V.Slices...)
}
