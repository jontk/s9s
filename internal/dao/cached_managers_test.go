package dao

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildJobListCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		opts     *ListJobsOptions
		expected string
	}{
		{
			name:     "nil options",
			opts:     nil,
			expected: "jobs:list:",
		},
		{
			name:     "empty options",
			opts:     &ListJobsOptions{},
			expected: "jobs:list::0:0::",
		},
		{
			name: "with all fields",
			opts: &ListJobsOptions{
				States:     []string{"RUNNING", "PENDING"},
				Limit:      100,
				Offset:     50,
				Users:      []string{"alice"},
				Partitions: []string{"gpu"},
			},
			expected: "jobs:list:RUNNING,PENDING:100:50:alice:gpu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildJobListCacheKey(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildNodeListCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		opts     *ListNodesOptions
		expected string
	}{
		{
			name:     "nil options",
			opts:     nil,
			expected: "nodes:list:",
		},
		{
			name:     "empty options",
			opts:     &ListNodesOptions{},
			expected: "nodes:list::",
		},
		{
			name: "with fields",
			opts: &ListNodesOptions{
				States:     []string{"IDLE", "MIXED"},
				Partitions: []string{"compute"},
			},
			expected: "nodes:list:IDLE,MIXED:compute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildNodeListCacheKey(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCopyJobList(t *testing.T) {
	original := &JobList{
		Jobs:  []*Job{{ID: "1"}, {ID: "2"}},
		Total: 2,
	}

	copied := copyJobList(original)

	assert.Equal(t, original.Total, copied.Total)
	assert.Equal(t, len(original.Jobs), len(copied.Jobs))
	assert.Equal(t, original.Jobs[0].ID, copied.Jobs[0].ID)

	// Modifying the copy's slice should not affect the original
	copied.Jobs[0] = &Job{ID: "999"}
	assert.Equal(t, "1", original.Jobs[0].ID)
}

func TestCopyNodeList(t *testing.T) {
	original := &NodeList{
		Nodes: []*Node{{Name: "node1"}, {Name: "node2"}},
		Total: 2,
	}

	copied := copyNodeList(original)

	assert.Equal(t, original.Total, copied.Total)
	assert.Equal(t, len(original.Nodes), len(copied.Nodes))

	copied.Nodes[0] = &Node{Name: "changed"}
	assert.Equal(t, "node1", original.Nodes[0].Name)
}

func TestCopyPartitionList(t *testing.T) {
	original := &PartitionList{
		Partitions: []*Partition{{Name: "gpu"}, {Name: "cpu"}},
	}

	copied := copyPartitionList(original)

	assert.Equal(t, len(original.Partitions), len(copied.Partitions))

	copied.Partitions[0] = &Partition{Name: "changed"}
	assert.Equal(t, "gpu", original.Partitions[0].Name)
}
