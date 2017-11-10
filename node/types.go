package node

import (
	"github.com/vechain/solidb/blob"
	"github.com/vechain/solidb/spec"
)

// Revisions gather all types of revision
type Revisions struct {
	Newest   int `json:"newest"`
	Synced   int `json:"synced"`
	Approved int `json:"approved"`
}

// StatusResponse status of node
type StatusResponse struct {
	NodeID        string    `json:"nodeID"`
	ClusterID     string    `json:"clusterID"`
	SpecRevisions Revisions `json:"specRevisions"`
}

// SyncStatusResponse sync status
type SyncStatusResponse struct {
	SyncedSliceCount int
	TotalSliceCount  int
}

// InviteRequest request body struct for invitation
type InviteRequest struct {
	InitSpec *spec.Spec `json:"initSpec"`
}

// InviteResponse response body struct for invitation
type InviteResponse struct {
	NodeID string `json:"nodeID"`
}

// PutBlobResponse response body struct for put blob
type PutBlobResponse struct {
	Key blob.Key `json:"key"`
}
