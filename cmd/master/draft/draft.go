package draft

import (
	"encoding/hex"
	"errors"
	"sort"

	"github.com/vechain/solidb/spec"
	yaml "gopkg.in/yaml.v2"
)

type Node struct {
	ID     string
	Addr   string
	Weight int
}

type Draft struct {
	Replicas int
	Nodes    []Node
}

func New(replicas int) (*Draft, error) {
	draft := &Draft{
		Replicas: replicas,
	}
	if err := draft.Validate(); err != nil {
		return nil, err
	}
	return draft, nil
}

func Unmarshal(data []byte) (*Draft, error) {
	var draft Draft
	if err := yaml.Unmarshal(data, &draft); err != nil {
		return nil, err
	}
	if err := draft.Validate(); err != nil {
		return nil, err
	}
	return &draft, nil
}

func (draft *Draft) Marshal() ([]byte, error) {
	return yaml.Marshal(draft)
}

func (draft *Draft) Validate() error {
	if draft.Replicas < 1 {
		return errors.New("replicas must be >= 1")
	}
	idset := make(map[string]bool)
	for _, n := range draft.Nodes {
		if idset[n.ID] {
			return errors.New("duplicated node")
		}
		idset[n.ID] = true
	}
	return nil
}

func (draft *Draft) WorkingNodes() []Node {
	wns := make([]Node, 0, len(draft.Nodes))
	for _, n := range draft.Nodes {
		if n.Weight > 0 {
			wns = append(wns, n)
		}
	}
	return wns
}

var sliceSet = func() []string {
	var slices []string
	for i := 0; i < 256; i++ {
		slices = append(slices, hex.EncodeToString([]byte{byte(i)}))
	}
	return slices
}()

func (draft *Draft) Alloc() (*spec.SAT, error) {
	workingNodeCount := 0
	weightSum := 0
	for _, n := range draft.Nodes {
		if n.Weight > 0 {
			workingNodeCount++
			weightSum += n.Weight
		}
	}
	if workingNodeCount < draft.Replicas {
		return nil, errors.New("not enough nodes")
	}

	allSlices := make([]string, 0, len(sliceSet)*draft.Replicas)
	for i := 0; i < draft.Replicas; i++ {
		allSlices = append(allSlices, sliceSet...)
	}

	var slots []slot
	for _, n := range draft.Nodes {
		slots = append(slots, slot{
			totalSliceCount: len(allSlices),
			weightSum:       weightSum,
			weight:          n.Weight,
			slices:          make(map[string]bool),
		})
	}

	for {
		remainedSliceCount := len(allSlices)
		if remainedSliceCount == 0 {
			break
		}
		for _, slot := range slots {
			if slot.isFull() {
				continue
			}
			allSlices = slot.pickOneFrom(allSlices)
		}

		if len(allSlices) == remainedSliceCount {
			// no more slice alloced, assign to each slot
			for _, slot := range slots {
				if slot.weight <= 0 {
					continue
				}
				allSlices = slot.pickOneFrom(allSlices)
			}
		}
	}
	sat := spec.SAT{}
	for i, n := range draft.Nodes {
		sat.Entries = append(sat.Entries, spec.Entry{
			ID:     n.ID,
			Addr:   n.Addr,
			Slices: slots[i].sortedSlices(),
		})
	}
	return &sat, nil
}

type slot struct {
	totalSliceCount int
	weightSum       int
	weight          int
	slices          map[string]bool
}

func (s *slot) isFull() bool {
	return s.weight <= 0 || len(s.slices) >= s.weight*s.totalSliceCount/s.weightSum
}

func (s *slot) pickOneFrom(slices []string) []string {
	for i, slice := range slices {
		if !s.slices[slice] {
			s.slices[slice] = true
			return append(slices[0:i], slices[i+1:]...)
		}
	}
	return slices
}

func (s *slot) sortedSlices() []string {
	var slices []string
	for slice := range s.slices {
		slices = append(slices, slice)
	}
	sort.SliceStable(slices, func(i1, i2 int) bool {
		return slices[i1] < slices[i2]
	})
	return slices
}
