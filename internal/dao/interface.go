package dao

import (
	"context"
)

// SlurmClient is the main interface for interacting with the SLURM cluster
type SlurmClient interface {
	// Jobs returns the job manager
	Jobs() JobManager
	
	// Nodes returns the node manager
	Nodes() NodeManager
	
	// Partitions returns the partition manager
	Partitions() PartitionManager
	
	// Reservations returns the reservation manager
	Reservations() ReservationManager
	
	// Info returns the info manager for cluster information
	Info() InfoManager
	
	// ClusterInfo returns basic cluster information
	ClusterInfo() (*ClusterInfo, error)
	
	// Close closes the client and releases resources
	Close() error
}

// JobManager provides operations for managing SLURM jobs
type JobManager interface {
	// List returns a list of jobs
	List(opts *ListJobsOptions) (*JobList, error)
	
	// Get returns details for a specific job
	Get(id string) (*Job, error)
	
	// Cancel cancels a job
	Cancel(id string) error
	
	// Hold holds a job
	Hold(id string) error
	
	// Release releases a held job
	Release(id string) error
	
	// GetOutput returns the output of a completed job
	GetOutput(id string) (string, error)
}

// NodeManager provides operations for managing SLURM nodes
type NodeManager interface {
	// List returns a list of nodes
	List(opts *ListNodesOptions) (*NodeList, error)
	
	// Get returns details for a specific node
	Get(name string) (*Node, error)
	
	// Drain drains a node with the given reason
	Drain(name string, reason string) error
	
	// Resume resumes a drained node
	Resume(name string) error
	
	// SetState sets the state of a node
	SetState(name string, state string) error
}

// PartitionManager provides operations for managing SLURM partitions
type PartitionManager interface {
	// List returns a list of partitions
	List() (*PartitionList, error)
	
	// Get returns details for a specific partition
	Get(name string) (*Partition, error)
}

// ReservationManager provides operations for managing SLURM reservations
type ReservationManager interface {
	// List returns a list of reservations
	List() (*ReservationList, error)
	
	// Get returns details for a specific reservation
	Get(name string) (*Reservation, error)
}

// InfoManager provides cluster information and statistics
type InfoManager interface {
	// GetClusterInfo returns basic cluster information
	GetClusterInfo() (*ClusterInfo, error)
	
	// GetStats returns cluster statistics
	GetStats() (*ClusterMetrics, error)
}

// ClusterInfo represents basic cluster information
type ClusterInfo struct {
	Name     string
	Endpoint string
	Version  string
}

// SlurmClientFactory creates SLURM clients for different clusters
type SlurmClientFactory interface {
	// NewClient creates a new SLURM client for the given cluster config
	NewClient(ctx context.Context, clusterName string) (SlurmClient, error)
	
	// GetCurrent returns the current active client
	GetCurrent() (SlurmClient, error)
	
	// SetCurrent sets the current active cluster
	SetCurrent(clusterName string) error
	
	// ListClusters returns available cluster names
	ListClusters() []string
}