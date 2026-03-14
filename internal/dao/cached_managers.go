package dao

import (
	"fmt"
	"strings"
)

// cachedJobManager wraps a jobManager with TTL caching for List operations
type cachedJobManager struct {
	inner *jobManager
	cache *DAOCache
}

func buildJobListCacheKey(opts *ListJobsOptions) string {
	if opts == nil {
		return "jobs:list:"
	}
	return fmt.Sprintf("jobs:list:%s:%d:%d:%s:%s",
		strings.Join(opts.States, ","),
		opts.Limit, opts.Offset,
		strings.Join(opts.Users, ","),
		strings.Join(opts.Partitions, ","))
}

func (c *cachedJobManager) List(opts *ListJobsOptions) (*JobList, error) {
	key := buildJobListCacheKey(opts)
	if cached, ok := c.cache.Get(key); ok {
		return copyJobList(cached.(*JobList)), nil
	}
	result, err := c.inner.List(opts)
	if err != nil {
		return nil, err
	}
	c.cache.Set(key, result, 0)
	return result, nil
}

func (c *cachedJobManager) Get(id string) (*Job, error)               { return c.inner.Get(id) }
func (c *cachedJobManager) Submit(job *JobSubmission) (string, error) { return c.inner.Submit(job) }
func (c *cachedJobManager) Cancel(id string) error {
	c.cache.InvalidatePrefix("jobs:")
	return c.inner.Cancel(id)
}
func (c *cachedJobManager) Hold(id string) error {
	c.cache.InvalidatePrefix("jobs:")
	return c.inner.Hold(id)
}
func (c *cachedJobManager) Release(id string) error {
	c.cache.InvalidatePrefix("jobs:")
	return c.inner.Release(id)
}
func (c *cachedJobManager) Requeue(id string) (*Job, error) {
	c.cache.InvalidatePrefix("jobs:")
	return c.inner.Requeue(id)
}
func (c *cachedJobManager) GetOutput(id string) (string, error) { return c.inner.GetOutput(id) }
func (c *cachedJobManager) Notify(id, message string) error {
	return c.inner.Notify(id, message)
}

// cachedNodeManager wraps a nodeManager with TTL caching for List operations
type cachedNodeManager struct {
	inner *nodeManager
	cache *DAOCache
}

func buildNodeListCacheKey(opts *ListNodesOptions) string {
	if opts == nil {
		return "nodes:list:"
	}
	return fmt.Sprintf("nodes:list:%s:%s",
		strings.Join(opts.States, ","),
		strings.Join(opts.Partitions, ","))
}

func (c *cachedNodeManager) List(opts *ListNodesOptions) (*NodeList, error) {
	key := buildNodeListCacheKey(opts)
	if cached, ok := c.cache.Get(key); ok {
		return copyNodeList(cached.(*NodeList)), nil
	}
	result, err := c.inner.List(opts)
	if err != nil {
		return nil, err
	}
	c.cache.Set(key, result, 0)
	return result, nil
}

func (c *cachedNodeManager) Get(name string) (*Node, error) { return c.inner.Get(name) }
func (c *cachedNodeManager) Drain(name, reason string) error {
	c.cache.InvalidatePrefix("nodes:")
	return c.inner.Drain(name, reason)
}
func (c *cachedNodeManager) Resume(name string) error {
	c.cache.InvalidatePrefix("nodes:")
	return c.inner.Resume(name)
}
func (c *cachedNodeManager) SetState(name, state string) error {
	c.cache.InvalidatePrefix("nodes:")
	return c.inner.SetState(name, state)
}

// cachedPartitionManager wraps a partitionManager with TTL caching for List operations
type cachedPartitionManager struct {
	inner *partitionManager
	cache *DAOCache
}

func (c *cachedPartitionManager) List() (*PartitionList, error) {
	key := "partitions:list"
	if cached, ok := c.cache.Get(key); ok {
		return copyPartitionList(cached.(*PartitionList)), nil
	}
	result, err := c.inner.List()
	if err != nil {
		return nil, err
	}
	c.cache.Set(key, result, 0)
	return result, nil
}

func (c *cachedPartitionManager) Get(name string) (*Partition, error) { return c.inner.Get(name) }

// copyJobList returns a shallow copy so callers can't corrupt the cached data.
func copyJobList(src *JobList) *JobList {
	jobs := make([]*Job, len(src.Jobs))
	copy(jobs, src.Jobs)
	return &JobList{Jobs: jobs, Total: src.Total}
}

// copyNodeList returns a shallow copy so callers can't corrupt the cached data.
func copyNodeList(src *NodeList) *NodeList {
	nodes := make([]*Node, len(src.Nodes))
	copy(nodes, src.Nodes)
	return &NodeList{Nodes: nodes, Total: src.Total}
}

// copyPartitionList returns a shallow copy so callers can't corrupt the cached data.
func copyPartitionList(src *PartitionList) *PartitionList {
	partitions := make([]*Partition, len(src.Partitions))
	copy(partitions, src.Partitions)
	return &PartitionList{Partitions: partitions}
}
