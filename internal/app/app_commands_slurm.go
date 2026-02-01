package app

import (
	"fmt"
	"strings"
	"time"
)

// cmdCancelJob cancels a SLURM job
func (s *S9s) cmdCancelJob(args []string) CommandResult {
	jobID := args[0]
	if err := s.client.Jobs().Cancel(jobID); err != nil {
		return CommandResult{
			Success: false,
			Message: fmt.Sprintf("Failed to cancel job %s: %v", jobID, err),
			Error:   err,
		}
	}
	s.refreshCurrentViewAsync()
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Job %s canceled", jobID),
	}
}

// cmdHoldJob holds a SLURM job
func (s *S9s) cmdHoldJob(args []string) CommandResult {
	jobID := args[0]
	if err := s.client.Jobs().Hold(jobID); err != nil {
		return CommandResult{
			Success: false,
			Message: fmt.Sprintf("Failed to hold job %s: %v", jobID, err),
			Error:   err,
		}
	}
	s.refreshCurrentViewAsync()
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Job %s held", jobID),
	}
}

// cmdReleaseJob releases a held SLURM job
func (s *S9s) cmdReleaseJob(args []string) CommandResult {
	jobID := args[0]
	if err := s.client.Jobs().Release(jobID); err != nil {
		return CommandResult{
			Success: false,
			Message: fmt.Sprintf("Failed to release job %s: %v", jobID, err),
			Error:   err,
		}
	}
	s.refreshCurrentViewAsync()
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Job %s released", jobID),
	}
}

// cmdRequeueJob requeues a SLURM job
func (s *S9s) cmdRequeueJob(args []string) CommandResult {
	jobID := args[0]
	if _, err := s.client.Jobs().Requeue(jobID); err != nil {
		return CommandResult{
			Success: false,
			Message: fmt.Sprintf("Failed to requeue job %s: %v", jobID, err),
			Error:   err,
		}
	}
	s.refreshCurrentViewAsync()
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Job %s requeued", jobID),
	}
}

// cmdDrainNode drains a SLURM node
func (s *S9s) cmdDrainNode(args []string) CommandResult {
	nodeName := args[0]
	reason := "Drained via s9s command"
	if len(args) > 1 {
		reason = strings.Join(args[1:], " ")
	}
	if err := s.client.Nodes().Drain(nodeName, reason); err != nil {
		return CommandResult{
			Success: false,
			Message: fmt.Sprintf("Failed to drain node %s: %v", nodeName, err),
			Error:   err,
		}
	}
	s.refreshCurrentViewAsync()
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Node %s drained: %s", nodeName, reason),
	}
}

// cmdResumeNode resumes a drained SLURM node
func (s *S9s) cmdResumeNode(args []string) CommandResult {
	nodeName := args[0]
	if err := s.client.Nodes().Resume(nodeName); err != nil {
		return CommandResult{
			Success: false,
			Message: fmt.Sprintf("Failed to resume node %s: %v", nodeName, err),
			Error:   err,
		}
	}
	s.refreshCurrentViewAsync()
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Node %s resumed", nodeName),
	}
}

// refreshCurrentViewAsync refreshes the current view after a delay
func (s *S9s) refreshCurrentViewAsync() {
	go func() {
		time.Sleep(500 * time.Millisecond)
		if cv, err := s.viewMgr.GetCurrentView(); err == nil {
			_ = cv.Refresh()
		}
	}()
}
