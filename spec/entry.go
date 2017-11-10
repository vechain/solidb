package spec

import "strings"

// Entry describes a SAT entry.
type Entry struct {
	ID     string   `json:"ID"`
	Addr   string   `json:"Addr"`
	Slices []string `json:"slices"`
}

type OptEntry struct {
	V *Entry
}

// String returns summary of entry
func (e Entry) String() string {
	return e.ID + "@" + e.Addr
}

// ContainsKey tests whether entry contains the key.
func (e Entry) ContainsKey(key string) bool {
	for _, slice := range e.Slices {
		if strings.HasPrefix(key, slice) {
			return true
		}
	}
	return false
}
