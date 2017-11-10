// Package spec defines cluster specifications
package spec

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/vechain/solidb/crypto"
)

// Spec contains all information of cluster
type Spec struct {
	// revision of spec. Usually it's auto incremental.
	Revision int `json:"revision"`
	// Slice allocation table for whole data collection
	SAT SAT `json:"sat"`
}

// OptSpec optional spec
type OptSpec struct {
	V *Spec
}

// Validate validates spec. Returns non-nil error if not valid.
func (s *Spec) Validate() error {
	set := make(map[string]bool)
	for _, e := range s.SAT.Entries {
		if set[e.ID] {
			return errors.New("duplicated entry, ID " + e.ID)
		}
		set[e.ID] = true
	}
	return nil
}

// Hash returns hash of spec marshaled into JSON
func (s *Spec) Hash() crypto.Hash {
	data, _ := json.Marshal(s)
	return crypto.HashSum(data)
}
