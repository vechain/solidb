package spec

// SAT slice allocation table
type SAT struct {
	Entries []Entry `json:"entries"`
}

// FindEntry find entry by ID
func (sat SAT) FindEntry(id string) *Entry {
	for _, e := range sat.Entries {
		if e.ID == id {
			return &e
		}
	}
	return nil
}

// Locate find out all entries containing the key
func (sat SAT) Locate(key string) []Entry {
	var entries []Entry
	for _, e := range sat.Entries {
		if e.ContainsKey(key) {
			entries = append(entries, e)
		}
	}
	return entries
}
