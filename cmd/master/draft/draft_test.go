package draft_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vechain/solidb/cmd/master/draft"
)

func TestAllocation(t *testing.T) {
	assert := assert.New(t)

	_, err := draft.New(0)
	assert.NotNil(err)

	d, err := draft.New(2)
	assert.Nil(err)
	assert.NotNil(d)

	_, err = d.Alloc()
	assert.NotNil(err)

	d.Nodes = []draft.Node{
		{
			ID:     "1",
			Addr:   "n1",
			Weight: 1,
		},
		{
			ID:     "2",
			Addr:   "n2",
			Weight: 1,
		},
	}
	sat, err := d.Alloc()
	assert.Nil(err)
	assert.Equal(len(sat.Entries), len(d.Nodes))

}
