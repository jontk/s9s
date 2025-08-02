package dao

import (
	"context"
	"fmt"
	"strings"
	"time"

	slurm "github.com/jontk/slurm-client"
	"github.com/jontk/slurm-client/pkg/auth"
	slurmconfig "github.com/jontk/slurm-client/pkg/config"
	"github.com/jontk/s9s/internal/config"
)

// SlurmAdapter wraps the slurm-client library and provides version abstraction
type SlurmAdapter struct {
	client slurm.SlurmClient
	config *config.ClusterConfig
	ctx    context.Context
}

// NewSlurmAdapter creates a new SLURM adapter instance
func NewSlurmAdapter(ctx context.Context, cfg *config.ClusterConfig) (*SlurmAdapter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cluster config is required")
	}

	// Parse timeout
	timeout := 30 * time.Second
	if cfg.Timeout != "" {
		if t, err := time.ParseDuration(cfg.Timeout); err == nil {
			timeout = t
		}
	}

	// Create slurm-client config
	slurmCfg := &slurmconfig.Config{
		BaseURL: cfg.Endpoint,
		Timeout: timeout,
	}

	// Create client options
	opts := []slurm.ClientOption{
		slurm.WithConfig(slurmCfg),
	}

	// Add authentication if token is provided
	if cfg.Token != "" {
		authProvider := auth.NewTokenAuth(cfg.Token)
		opts = append(opts, slurm.WithAuth(authProvider))
	}

	// Create the client
	var slurmClient slurm.SlurmClient
	var err error
	
	if cfg.APIVersion != "" {
		slurmClient, err = slurm.NewClientWithVersion(ctx, cfg.APIVersion, opts...)
	} else {
		// Auto-detect version
		slurmClient, err = slurm.NewClient(ctx, opts...)
	}
	
	if err != nil {
		return nil, fmt.Errorf("creating slurm client: %w", err)
	}

	return &SlurmAdapter{
		client: slurmClient,
		config: cfg,
		ctx:    ctx,
	}, nil
}

// Close closes the adapter and releases resources
func (s *SlurmAdapter) Close() error {
	return nil
}

// Jobs returns the jobs manager
func (s *SlurmAdapter) Jobs() JobManager {
	return &jobManager{
		client: s.client.Jobs(),
		ctx:    s.ctx,
	}
}

// Nodes returns the nodes manager
func (s *SlurmAdapter) Nodes() NodeManager {
	return &nodeManager{
		client: s.client.Nodes(),
		ctx:    s.ctx,
	}
}

// Partitions returns the partitions manager
func (s *SlurmAdapter) Partitions() PartitionManager {
	return &partitionManager{
		client: s.client.Partitions(),
		ctx:    s.ctx,
	}
}

// Reservations returns the reservations manager
func (s *SlurmAdapter) Reservations() ReservationManager {
	return &reservationManager{
		client: s.client.Reservations(),
		ctx:    s.ctx,
	}
}

// Info returns the info manager for cluster information
func (s *SlurmAdapter) Info() InfoManager {
	return &infoManager{
		client: s.client.Info(),
		ctx:    s.ctx,
	}
}

// ClusterInfo returns cluster information
func (s *SlurmAdapter) ClusterInfo() (*ClusterInfo, error) {
	info, err := s.client.Info().Get(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("getting cluster info: %w", err)
	}

	return &ClusterInfo{
		Name:     info.ClusterName,
		Endpoint: s.config.Endpoint,
		Version:  info.Version,
	}, nil
}

// jobManager implements JobManager
type jobManager struct {
	client slurm.JobManager
	ctx    context.Context
}

func (j *jobManager) List(opts *ListJobsOptions) (*JobList, error) {
	// Convert options to slurm-client format
	clientOpts := &slurm.ListJobsOptions{}
	if opts != nil {
		clientOpts.States = opts.States
		clientOpts.Limit = opts.Limit
		clientOpts.Offset = opts.Offset
		// Note: Users, Partitions, Accounts may need to be handled differently
		// depending on the actual slurm-client API
	}

	// Call the client
	result, err := j.client.List(j.ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("listing jobs: %w", err)
	}

	// Convert to our types
	jobs := make([]*Job, len(result.Jobs))
	for i, job := range result.Jobs {
		jobs[i] = convertJob(&job)
	}

	return &JobList{
		Jobs:  jobs,
		Total: result.Total,
	}, nil
}

func (j *jobManager) Get(id string) (*Job, error) {
	job, err := j.client.Get(j.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting job %s: %w", id, err)
	}
	return convertJob(job), nil
}

func (j *jobManager) Cancel(id string) error {
	return j.client.Cancel(j.ctx, id)
}

func (j *jobManager) Hold(id string) error {
	// Note: May need to implement if slurm-client supports it
	return fmt.Errorf("hold operation not supported by slurm-client")
}

func (j *jobManager) Release(id string) error {
	// Note: May need to implement if slurm-client supports it
	return fmt.Errorf("release operation not supported by slurm-client")
}

func (j *jobManager) GetOutput(id string) (string, error) {
	// Note: May need to implement if slurm-client supports it
	return "", fmt.Errorf("get output operation not supported by slurm-client")
}

// nodeManager implements NodeManager
type nodeManager struct {
	client slurm.NodeManager
	ctx    context.Context
}

func (n *nodeManager) List(opts *ListNodesOptions) (*NodeList, error) {
	// Convert options to slurm-client format
	clientOpts := &slurm.ListNodesOptions{}
	if opts != nil {
		clientOpts.States = opts.States
		// Note: Partitions may need to be handled differently
	}

	result, err := n.client.List(n.ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	// Convert to our types
	nodes := make([]*Node, len(result.Nodes))
	for i, node := range result.Nodes {
		nodes[i] = convertNode(&node)
	}

	return &NodeList{
		Nodes: nodes,
		Total: result.Total,
	}, nil
}

func (n *nodeManager) Get(name string) (*Node, error) {
	node, err := n.client.Get(n.ctx, name)
	if err != nil {
		return nil, fmt.Errorf("getting node %s: %w", name, err)
	}
	return convertNode(node), nil
}

func (n *nodeManager) Drain(name string, reason string) error {
	// Note: May need to implement if slurm-client supports it
	return fmt.Errorf("drain operation not supported by slurm-client")
}

func (n *nodeManager) Resume(name string) error {
	// Note: May need to implement if slurm-client supports it
	return fmt.Errorf("resume operation not supported by slurm-client")
}

func (n *nodeManager) SetState(name string, state string) error {
	// Note: May need to implement if slurm-client supports it
	return fmt.Errorf("set state operation not supported by slurm-client")
}

// partitionManager implements PartitionManager
type partitionManager struct {
	client slurm.PartitionManager
	ctx    context.Context
}

func (p *partitionManager) List() (*PartitionList, error) {
	result, err := p.client.List(p.ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("listing partitions: %w", err)
	}

	// Convert to our types
	partitions := make([]*Partition, len(result.Partitions))
	for i, partition := range result.Partitions {
		partitions[i] = convertPartition(&partition)
	}

	return &PartitionList{
		Partitions: partitions,
	}, nil
}

func (p *partitionManager) Get(name string) (*Partition, error) {
	partition, err := p.client.Get(p.ctx, name)
	if err != nil {
		return nil, fmt.Errorf("getting partition %s: %w", name, err)
	}
	return convertPartition(partition), nil
}

// reservationManager implements ReservationManager
type reservationManager struct {
	client slurm.ReservationManager
	ctx    context.Context
}

func (r *reservationManager) List() (*ReservationList, error) {
	result, err := r.client.List(r.ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("listing reservations: %w", err)
	}

	// Convert to our types
	reservations := make([]*Reservation, len(result.Reservations))
	for i, res := range result.Reservations {
		reservations[i] = convertReservation(&res)
	}

	return &ReservationList{
		Reservations: reservations,
	}, nil
}

func (r *reservationManager) Get(name string) (*Reservation, error) {
	reservation, err := r.client.Get(r.ctx, name)
	if err != nil {
		return nil, fmt.Errorf("getting reservation %s: %w", name, err)
	}
	return convertReservation(reservation), nil
}

// infoManager implements InfoManager
type infoManager struct {
	client slurm.InfoManager
	ctx    context.Context
}

func (i *infoManager) GetClusterInfo() (*ClusterInfo, error) {
	info, err := i.client.Get(i.ctx)
	if err != nil {
		return nil, fmt.Errorf("getting cluster info: %w", err)
	}

	return &ClusterInfo{
		Name:     info.ClusterName,
		Endpoint: "",
		Version:  info.Version,
	}, nil
}

func (i *infoManager) GetStats() (*ClusterMetrics, error) {
	stats, err := i.client.Stats(i.ctx)
	if err != nil {
		return nil, fmt.Errorf("getting cluster stats: %w", err)
	}

	return &ClusterMetrics{
		TotalJobs:   stats.TotalJobs,
		RunningJobs: stats.RunningJobs,
		PendingJobs: stats.PendingJobs,
		TotalNodes:  stats.TotalNodes,
		ActiveNodes: stats.AllocatedNodes,
		IdleNodes:   stats.IdleNodes,
		DownNodes:   0, // Not available in basic ClusterStats
		CPUUsage:    float64(stats.AllocatedCPUs) / float64(stats.TotalCPUs) * 100,
		MemoryUsage: 0.0, // Not available in basic ClusterStats
		LastUpdated: time.Now(),
	}, nil
}

// Conversion functions
func convertJob(job *slurm.Job) *Job {
	var exitCode *int
	if job.ExitCode != 0 {
		exitCode = &job.ExitCode
	}
	
	return &Job{
		ID:          job.ID,
		Name:        job.Name,
		User:        job.UserID,
		Account:     "", // Not available in basic Job struct
		Partition:   job.Partition,
		State:       job.State,
		Priority:    float64(job.Priority),
		QOS:         "", // Not available in basic Job struct
		NodeCount:   len(job.Nodes),
		TimeLimit:   fmt.Sprintf("%d", job.TimeLimit),
		TimeUsed:    "", // Not available in basic Job struct
		SubmitTime:  job.SubmitTime,
		StartTime:   job.StartTime,
		EndTime:     job.EndTime,
		NodeList:    strings.Join(job.Nodes, ","),
		Command:     job.Command,
		WorkingDir:  job.WorkingDir,
		StdOut:      "", // Not available in basic Job struct
		StdErr:      "", // Not available in basic Job struct
		ExitCode:    exitCode,
	}
}

func convertNode(node *slurm.Node) *Node {
	return &Node{
		Name:            node.Name,
		State:           node.State,
		Partitions:      node.Partitions,
		CPUsTotal:       node.CPUs,
		CPUsAllocated:   0, // Not available in basic Node struct
		CPUsIdle:        node.CPUs, // Assume all idle for now
		MemoryTotal:     int64(node.Memory),
		MemoryAllocated: 0, // Not available in basic Node struct
		MemoryFree:      int64(node.Memory), // Assume all free for now
		Features:        node.Features,
		Reason:          node.Reason,
		ReasonTime:      node.LastBusy,
		AllocatedJobs:   []string{}, // Not available in basic Node struct
	}
}

func convertPartition(partition *slurm.Partition) *Partition {
	return &Partition{
		Name:        partition.Name,
		State:       partition.State,
		TotalNodes:  partition.TotalNodes,
		TotalCPUs:   partition.TotalCPUs,
		DefaultTime: fmt.Sprintf("%d", partition.DefaultTime),
		MaxTime:     fmt.Sprintf("%d", partition.MaxTime),
		QOS:         []string{}, // Not available in basic Partition struct
		Nodes:       []string{}, // Not available in basic Partition struct
	}
}

func convertReservation(res *slurm.Reservation) *Reservation {
	return &Reservation{
		Name:      res.Name,
		State:     res.State,
		StartTime: res.StartTime,
		EndTime:   res.EndTime,
		Duration:  res.EndTime.Sub(res.StartTime),
		Nodes:     res.Nodes,
		NodeCount: res.NodeCount,
		CoreCount: res.CoreCount,
		Users:     res.Users,
		Accounts:  res.Accounts,
	}
}