package master

import (
	"fmt"

	"github.com/vechain/solidb/node"
)

type nodeStatus struct {
	status *node.StatusResponse
	err    error
}

func (ns nodeStatus) String() string {
	if ns.err != nil {
		return fmt.Sprintf("\"%v\"", ns.err)
	}

	r := &ns.status.SpecRevisions
	return fmt.Sprintf("%d,%d,%d", r.Newest, r.Synced, r.Approved)
}

type nodeSyncStatus struct {
	syncStatus *node.SyncStatusResponse
	err        error
}

func (nss nodeSyncStatus) String() string {
	if nss.err != nil {
		return fmt.Sprintf("\"%v\"", nss.err)
	}
	ss := nss.syncStatus
	return fmt.Sprintf("%d/%d", ss.SyncedSliceCount, ss.TotalSliceCount)
}

type nodeLoc struct {
	id   string
	addr string
}

func queryNodeStatus(nodeLocs []nodeLoc) chan nodeStatus {
	c := make(chan nodeStatus)
	go func() {
		rpc := node.NewRPC()
		for _, loc := range nodeLocs {
			status, err := rpc.WithAddr(loc.addr).GetStatus()
			c <- nodeStatus{
				status: status,
				err:    err,
			}
		}
		close(c)
	}()
	return c
}

func queryNodeSyncStatus(nodeLocs []nodeLoc, revision int) chan nodeSyncStatus {
	c := make(chan nodeSyncStatus)
	go func() {
		rpc := node.NewRPC()
		for _, loc := range nodeLocs {
			status, err := rpc.WithAddr(loc.addr).GetSyncStatus(revision)
			c <- nodeSyncStatus{
				syncStatus: status,
				err:        err,
			}
		}
		close(c)
	}()
	return c
}
