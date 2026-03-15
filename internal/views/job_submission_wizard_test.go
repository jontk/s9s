package views

import (
	"testing"

	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/dao"
	"github.com/stretchr/testify/assert"
)

func TestBuiltinTemplates_ReturnsExpectedCount(t *testing.T) {
	templates := BuiltinTemplates()
	assert.Len(t, templates, 8, "BuiltinTemplates should return 8 templates")
}

func TestBuiltinTemplates_AllHaveRequiredFields(t *testing.T) {
	for _, tmpl := range BuiltinTemplates() {
		t.Run(tmpl.Name, func(t *testing.T) {
			assert.NotEmpty(t, tmpl.Name, "template name must not be empty")
			assert.NotEmpty(t, tmpl.Description, "template description must not be empty")
			assert.NotEmpty(t, tmpl.JobSubmission.Script, "template script must not be empty")
			assert.NotEmpty(t, tmpl.JobSubmission.Partition, "template partition must not be empty")
		})
	}
}

func TestOverlayJobDefaults_OverlaysNonZeroFields(t *testing.T) {
	dst := &dao.JobSubmission{
		Name:        "original",
		Partition:   "normal",
		TimeLimit:   "01:00:00",
		Nodes:       1,
		CPUs:        1,
		Constraints: "old-feature",
		Reservation: "old-res",
	}

	src := &dao.JobSubmission{
		Partition:    "gpu",
		CPUs:         8,
		GPUs:         2,
		Memory:       "16G",
		Constraints:  "gpu,nvlink",
		NTasks:       16,
		Reservation:  "new-res",
		CPUBinding:   "cores",
		Dependencies: []string{"123", "456"},
		Nice:         -5,
	}

	overlayJobDefaults(dst, src)

	// Overwritten by src
	assert.Equal(t, "gpu", dst.Partition)
	assert.Equal(t, 8, dst.CPUs)
	assert.Equal(t, 2, dst.GPUs)
	assert.Equal(t, "16G", dst.Memory)
	assert.Equal(t, "gpu,nvlink", dst.Constraints)
	assert.Equal(t, 16, dst.NTasks)
	assert.Equal(t, "new-res", dst.Reservation)
	assert.Equal(t, "cores", dst.CPUBinding)
	assert.Equal(t, []string{"123", "456"}, dst.Dependencies)
	assert.Equal(t, -5, dst.Nice)
	// Preserved from dst (src was zero)
	assert.Equal(t, "original", dst.Name)
	assert.Equal(t, "01:00:00", dst.TimeLimit)
	assert.Equal(t, 1, dst.Nodes)
}

func TestOverlayJobDefaults_AllFields(t *testing.T) {
	// Start with a fully zeroed dst
	dst := &dao.JobSubmission{}

	// Set every field on src
	src := &dao.JobSubmission{
		Name:                "job",
		Script:              "#!/bin/bash",
		Partition:           "gpu",
		Account:             "research",
		QoS:                 "high",
		Nodes:               4,
		CPUs:                16,
		Memory:              "32G",
		GPUs:                2,
		TimeLimit:           "12:00:00",
		WorkingDir:          "/scratch",
		OutputFile:          "out.log",
		ErrorFile:           "err.log",
		EmailNotify:         true,
		Email:               "a@b.com",
		ArraySpec:           "1-10",
		Exclusive:           true,
		Requeue:             true,
		Constraints:         "gpu,nvlink",
		NTasks:              8,
		NTasksPerNode:       4,
		Gres:                "gpu:a100:2",
		Hold:                true,
		Reservation:         "res1",
		Licenses:            "matlab:2",
		Wckey:               "proj-a",
		ExcludeNodes:        "node01",
		Priority:            100,
		Nice:                -5,
		MemoryPerCPU:        "4G",
		BeginTime:           "tomorrow",
		Comment:             "test",
		Distribution:        "cyclic",
		Prefer:              "a100",
		RequiredNodes:       "node03",
		StandardInput:       "/tmp/in",
		Container:           "/oci/bundle",
		ThreadsPerCore:      2,
		TasksPerCore:        1,
		TasksPerSocket:      4,
		SocketsPerNode:      2,
		MaximumNodes:        10,
		MaximumCPUs:         64,
		MinimumCPUsPerNode:  8,
		TimeMinimum:         "00:30:00",
		Contiguous:          true,
		Overcommit:          true,
		KillOnNodeFail:      true,
		WaitAllNodes:        true,
		OpenMode:            "append",
		TRESPerTask:         "gres/gpu:1",
		TRESPerSocket:       "gres/gpu:2",
		Signal:              "B:USR1@300",
		TmpDiskPerNode:      1024,
		Deadline:            "2024-12-31",
		NTasksPerTRES:       4,
		CPUBinding:          "cores",
		CPUFrequency:        "high",
		Network:             "sn_all",
		X11:                 "batch",
		Immediate:           true,
		BurstBuffer:         "#BB spec",
		BatchFeatures:       "haswell",
		TRESBind:            "gpu:verbose",
		TRESFreq:            "gpu:high",
		CoreSpecification:   2,
		ThreadSpecification: 4,
		MemoryBinding:       "local",
		MinimumCPUs:         16,
		TRESPerJob:          "gres/gpu:8",
		CPUsPerTRES:         "gres/gpu:2",
		MemoryPerTRES:       "gres/gpu:4096",
		Argv:                "arg1 arg2",
		Flags:               "SPREAD_JOB",
		ProfileTypes:        "ENERGY",
		CPUBindingFlags:     "VERBOSE",
		MemoryBindingType:   "LOCAL",
		RequiredSwitches:    3,
		WaitForSwitch:       120,
		ClusterConstraint:   "gpu_cluster",
		Clusters:            "cluster1",
		Dependencies:        []string{"123", "456"},
	}

	overlayJobDefaults(dst, src)

	// Verify every field was overlaid
	assert.Equal(t, "job", dst.Name)
	assert.Equal(t, "#!/bin/bash", dst.Script)
	assert.Equal(t, "gpu", dst.Partition)
	assert.Equal(t, "research", dst.Account)
	assert.Equal(t, "high", dst.QoS)
	assert.Equal(t, 4, dst.Nodes)
	assert.Equal(t, 16, dst.CPUs)
	assert.Equal(t, "32G", dst.Memory)
	assert.Equal(t, 2, dst.GPUs)
	assert.Equal(t, "12:00:00", dst.TimeLimit)
	assert.Equal(t, "/scratch", dst.WorkingDir)
	assert.Equal(t, "out.log", dst.OutputFile)
	assert.Equal(t, "err.log", dst.ErrorFile)
	assert.True(t, dst.EmailNotify)
	assert.Equal(t, "a@b.com", dst.Email)
	assert.Equal(t, "1-10", dst.ArraySpec)
	assert.True(t, dst.Exclusive)
	assert.True(t, dst.Requeue)
	assert.Equal(t, "gpu,nvlink", dst.Constraints)
	assert.Equal(t, 8, dst.NTasks)
	assert.Equal(t, 4, dst.NTasksPerNode)
	assert.Equal(t, "gpu:a100:2", dst.Gres)
	assert.True(t, dst.Hold)
	assert.Equal(t, "res1", dst.Reservation)
	assert.Equal(t, "matlab:2", dst.Licenses)
	assert.Equal(t, "proj-a", dst.Wckey)
	assert.Equal(t, "node01", dst.ExcludeNodes)
	assert.Equal(t, 100, dst.Priority)
	assert.Equal(t, -5, dst.Nice)
	assert.Equal(t, "4G", dst.MemoryPerCPU)
	assert.Equal(t, "tomorrow", dst.BeginTime)
	assert.Equal(t, "test", dst.Comment)
	assert.Equal(t, "cyclic", dst.Distribution)
	assert.Equal(t, "a100", dst.Prefer)
	assert.Equal(t, "node03", dst.RequiredNodes)
	assert.Equal(t, "/tmp/in", dst.StandardInput)
	assert.Equal(t, "/oci/bundle", dst.Container)
	assert.Equal(t, 2, dst.ThreadsPerCore)
	assert.Equal(t, 1, dst.TasksPerCore)
	assert.Equal(t, 4, dst.TasksPerSocket)
	assert.Equal(t, 2, dst.SocketsPerNode)
	assert.Equal(t, 10, dst.MaximumNodes)
	assert.Equal(t, 64, dst.MaximumCPUs)
	assert.Equal(t, 8, dst.MinimumCPUsPerNode)
	assert.Equal(t, "00:30:00", dst.TimeMinimum)
	assert.True(t, dst.Contiguous)
	assert.True(t, dst.Overcommit)
	assert.True(t, dst.KillOnNodeFail)
	assert.True(t, dst.WaitAllNodes)
	assert.Equal(t, "append", dst.OpenMode)
	assert.Equal(t, "gres/gpu:1", dst.TRESPerTask)
	assert.Equal(t, "gres/gpu:2", dst.TRESPerSocket)
	assert.Equal(t, "B:USR1@300", dst.Signal)
	assert.Equal(t, 1024, dst.TmpDiskPerNode)
	assert.Equal(t, "2024-12-31", dst.Deadline)
	assert.Equal(t, 4, dst.NTasksPerTRES)
	assert.Equal(t, "cores", dst.CPUBinding)
	assert.Equal(t, "high", dst.CPUFrequency)
	assert.Equal(t, "sn_all", dst.Network)
	assert.Equal(t, "batch", dst.X11)
	assert.True(t, dst.Immediate)
	assert.Equal(t, "#BB spec", dst.BurstBuffer)
	assert.Equal(t, "haswell", dst.BatchFeatures)
	assert.Equal(t, "gpu:verbose", dst.TRESBind)
	assert.Equal(t, "gpu:high", dst.TRESFreq)
	assert.Equal(t, 2, dst.CoreSpecification)
	assert.Equal(t, 4, dst.ThreadSpecification)
	assert.Equal(t, "local", dst.MemoryBinding)
	assert.Equal(t, 16, dst.MinimumCPUs)
	assert.Equal(t, "gres/gpu:8", dst.TRESPerJob)
	assert.Equal(t, "gres/gpu:2", dst.CPUsPerTRES)
	assert.Equal(t, "gres/gpu:4096", dst.MemoryPerTRES)
	assert.Equal(t, "arg1 arg2", dst.Argv)
	assert.Equal(t, "SPREAD_JOB", dst.Flags)
	assert.Equal(t, "ENERGY", dst.ProfileTypes)
	assert.Equal(t, "VERBOSE", dst.CPUBindingFlags)
	assert.Equal(t, "LOCAL", dst.MemoryBindingType)
	assert.Equal(t, 3, dst.RequiredSwitches)
	assert.Equal(t, 120, dst.WaitForSwitch)
	assert.Equal(t, "gpu_cluster", dst.ClusterConstraint)
	assert.Equal(t, "cluster1", dst.Clusters)
	assert.Equal(t, []string{"123", "456"}, dst.Dependencies)
}

func TestOverlayJobDefaults_EmptySrcNoChange(t *testing.T) {
	dst := &dao.JobSubmission{
		Name:      "keep",
		Partition: "normal",
		Nodes:     4,
		CPUs:      16,
	}
	src := &dao.JobSubmission{}

	overlayJobDefaults(dst, src)

	assert.Equal(t, "keep", dst.Name)
	assert.Equal(t, "normal", dst.Partition)
	assert.Equal(t, 4, dst.Nodes)
	assert.Equal(t, 16, dst.CPUs)
}

func TestOverlayJobDefaults_BooleanFields(t *testing.T) {
	dst := &dao.JobSubmission{}
	src := &dao.JobSubmission{
		EmailNotify: true,
		Exclusive:   true,
		Requeue:     true,
	}

	overlayJobDefaults(dst, src)

	assert.True(t, dst.EmailNotify)
	assert.True(t, dst.Exclusive)
	assert.True(t, dst.Requeue)
}

func TestOverlayJobDefaults_BooleanFalseDoesNotOverwrite(t *testing.T) {
	dst := &dao.JobSubmission{
		EmailNotify: true,
		Exclusive:   true,
	}
	src := &dao.JobSubmission{
		EmailNotify: false,
		Exclusive:   false,
	}

	overlayJobDefaults(dst, src)

	// false is zero-value for bool, so dst should keep its true values
	assert.True(t, dst.EmailNotify)
	assert.True(t, dst.Exclusive)
}

func TestIsValidTimeFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"standard HH:MM:SS", "01:00:00", true},
		{"days format", "7-00:00:00", true},
		{"short time", "00:15:00", true},
		{"minutes only", "30:00", true},
		{"missing colon", "3600", false},
		{"empty string", "", false},
		{"multiple dashes", "1-2-3:00:00", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidTimeFormat(tt.input))
		})
	}
}

func TestIsValidMemoryFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"gigabytes", "4G", true},
		{"megabytes", "1024M", true},
		{"large memory", "256G", true},
		{"no suffix", "1024", false},
		{"wrong suffix", "4T", false},
		{"empty string", "", false},
		{"only suffix", "G", false},
		{"decimal number", "4.5G", false},
		{"lowercase suffix", "4g", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidMemoryFormat(tt.input))
		})
	}
}

func TestConfigValuesToJobSubmission_AllFields(t *testing.T) {
	v := config.JobSubmissionValues{
		Name:                "test-job",
		Script:              "#!/bin/bash\necho hi",
		Partition:           "gpu",
		Account:             "myaccount",
		QoS:                 "high",
		Nodes:               4,
		CPUs:                16,
		Memory:              "32G",
		GPUs:                2,
		TimeLimit:           "12:00:00",
		WorkingDir:          "/tmp/work",
		OutputFile:          "out.log",
		ErrorFile:           "err.log",
		EmailNotify:         true,
		Email:               "user@example.com",
		ArraySpec:           "1-10",
		Exclusive:           true,
		Requeue:             true,
		Constraints:         "gpu,nvlink",
		NTasks:              8,
		NTasksPerNode:       4,
		Gres:                "gres/gpu:a100:2",
		Hold:                true,
		Reservation:         "gpu_reservation",
		Licenses:            "matlab:2",
		Wckey:               "project-alpha",
		ExcludeNodes:        "node01,node02",
		Priority:            100,
		Nice:                -10,
		MemoryPerCPU:        "4G",
		BeginTime:           "2024-06-15T14:30:00",
		Comment:             "test comment",
		Distribution:        "cyclic",
		Prefer:              "a100",
		RequiredNodes:       "node03,node04",
		StandardInput:       "/tmp/input.dat",
		Container:           "/path/to/container",
		ThreadsPerCore:      2,
		TasksPerCore:        1,
		TasksPerSocket:      4,
		SocketsPerNode:      2,
		MaximumNodes:        10,
		MaximumCPUs:         64,
		MinimumCPUsPerNode:  8,
		TimeMinimum:         "00:30:00",
		Contiguous:          true,
		Overcommit:          true,
		KillOnNodeFail:      true,
		WaitAllNodes:        true,
		OpenMode:            "append",
		TRESPerTask:         "gres/gpu:1",
		TRESPerSocket:       "gres/gpu:2",
		Signal:              "B:USR1@300",
		TmpDiskPerNode:      1024,
		Deadline:            "2024-06-16T00:00:00",
		NTasksPerTRES:       4,
		CPUBinding:          "cores",
		CPUFrequency:        "high",
		Network:             "sn_all:torus",
		X11:                 "batch",
		Immediate:           true,
		BurstBuffer:         "#BB create_persistent",
		BatchFeatures:       "haswell",
		TRESBind:            "gpu:verbose",
		TRESFreq:            "gpu:high",
		CoreSpecification:   2,
		ThreadSpecification: 4,
		MemoryBinding:       "local",
		MinimumCPUs:         16,
		TRESPerJob:          "gres/gpu:8",
		CPUsPerTRES:         "gres/gpu:2",
		MemoryPerTRES:       "gres/gpu:4096",
		Argv:                "arg1 arg2",
		Flags:               "SPREAD_JOB",
		ProfileTypes:        "ENERGY,NETWORK",
		CPUBindingFlags:     "VERBOSE",
		MemoryBindingType:   "LOCAL",
		RequiredSwitches:    3,
		WaitForSwitch:       120,
		ClusterConstraint:   "gpu_cluster",
		Clusters:            "cluster1,cluster2",
		Dependencies:        []string{"123", "456"},
	}

	js := ConfigValuesToJobSubmission(v)

	// Core fields
	assert.Equal(t, v.Name, js.Name)
	assert.Equal(t, v.Script, js.Script)
	assert.Equal(t, v.Partition, js.Partition)
	assert.Equal(t, v.Account, js.Account)
	assert.Equal(t, v.QoS, js.QoS)
	assert.Equal(t, v.Nodes, js.Nodes)
	assert.Equal(t, v.CPUs, js.CPUs)
	assert.Equal(t, v.Memory, js.Memory)
	assert.Equal(t, v.GPUs, js.GPUs)
	assert.Equal(t, v.TimeLimit, js.TimeLimit)
	assert.Equal(t, v.WorkingDir, js.WorkingDir)
	assert.Equal(t, v.OutputFile, js.OutputFile)
	assert.Equal(t, v.ErrorFile, js.ErrorFile)
	assert.Equal(t, v.EmailNotify, js.EmailNotify)
	assert.Equal(t, v.Email, js.Email)
	assert.Equal(t, v.ArraySpec, js.ArraySpec)
	assert.Equal(t, v.Exclusive, js.Exclusive)
	assert.Equal(t, v.Requeue, js.Requeue)

	// Extended fields
	assert.Equal(t, v.Constraints, js.Constraints)
	assert.Equal(t, v.NTasks, js.NTasks)
	assert.Equal(t, v.NTasksPerNode, js.NTasksPerNode)
	assert.Equal(t, v.Gres, js.Gres)
	assert.Equal(t, v.Hold, js.Hold)
	assert.Equal(t, v.Reservation, js.Reservation)
	assert.Equal(t, v.Licenses, js.Licenses)
	assert.Equal(t, v.Wckey, js.Wckey)
	assert.Equal(t, v.ExcludeNodes, js.ExcludeNodes)
	assert.Equal(t, v.Priority, js.Priority)
	assert.Equal(t, v.Nice, js.Nice)
	assert.Equal(t, v.MemoryPerCPU, js.MemoryPerCPU)
	assert.Equal(t, v.BeginTime, js.BeginTime)
	assert.Equal(t, v.Comment, js.Comment)
	assert.Equal(t, v.Distribution, js.Distribution)
	assert.Equal(t, v.Prefer, js.Prefer)
	assert.Equal(t, v.RequiredNodes, js.RequiredNodes)
	assert.Equal(t, v.StandardInput, js.StandardInput)
	assert.Equal(t, v.Container, js.Container)
	assert.Equal(t, v.ThreadsPerCore, js.ThreadsPerCore)
	assert.Equal(t, v.TasksPerCore, js.TasksPerCore)
	assert.Equal(t, v.TasksPerSocket, js.TasksPerSocket)
	assert.Equal(t, v.SocketsPerNode, js.SocketsPerNode)
	assert.Equal(t, v.MaximumNodes, js.MaximumNodes)
	assert.Equal(t, v.MaximumCPUs, js.MaximumCPUs)
	assert.Equal(t, v.MinimumCPUsPerNode, js.MinimumCPUsPerNode)
	assert.Equal(t, v.TimeMinimum, js.TimeMinimum)
	assert.Equal(t, v.Contiguous, js.Contiguous)
	assert.Equal(t, v.Overcommit, js.Overcommit)
	assert.Equal(t, v.KillOnNodeFail, js.KillOnNodeFail)
	assert.Equal(t, v.WaitAllNodes, js.WaitAllNodes)
	assert.Equal(t, v.OpenMode, js.OpenMode)
	assert.Equal(t, v.TRESPerTask, js.TRESPerTask)
	assert.Equal(t, v.TRESPerSocket, js.TRESPerSocket)
	assert.Equal(t, v.Signal, js.Signal)
	assert.Equal(t, v.TmpDiskPerNode, js.TmpDiskPerNode)
	assert.Equal(t, v.Deadline, js.Deadline)
	assert.Equal(t, v.NTasksPerTRES, js.NTasksPerTRES)
	assert.Equal(t, v.CPUBinding, js.CPUBinding)
	assert.Equal(t, v.CPUFrequency, js.CPUFrequency)
	assert.Equal(t, v.Network, js.Network)
	assert.Equal(t, v.X11, js.X11)
	assert.Equal(t, v.Immediate, js.Immediate)
	assert.Equal(t, v.BurstBuffer, js.BurstBuffer)
	assert.Equal(t, v.BatchFeatures, js.BatchFeatures)
	assert.Equal(t, v.TRESBind, js.TRESBind)
	assert.Equal(t, v.TRESFreq, js.TRESFreq)
	assert.Equal(t, v.CoreSpecification, js.CoreSpecification)
	assert.Equal(t, v.ThreadSpecification, js.ThreadSpecification)
	assert.Equal(t, v.MemoryBinding, js.MemoryBinding)
	assert.Equal(t, v.MinimumCPUs, js.MinimumCPUs)
	assert.Equal(t, v.TRESPerJob, js.TRESPerJob)
	assert.Equal(t, v.CPUsPerTRES, js.CPUsPerTRES)
	assert.Equal(t, v.MemoryPerTRES, js.MemoryPerTRES)
	assert.Equal(t, v.Argv, js.Argv)
	assert.Equal(t, v.Flags, js.Flags)
	assert.Equal(t, v.ProfileTypes, js.ProfileTypes)
	assert.Equal(t, v.CPUBindingFlags, js.CPUBindingFlags)
	assert.Equal(t, v.MemoryBindingType, js.MemoryBindingType)
	assert.Equal(t, v.RequiredSwitches, js.RequiredSwitches)
	assert.Equal(t, v.WaitForSwitch, js.WaitForSwitch)
	assert.Equal(t, v.ClusterConstraint, js.ClusterConstraint)
	assert.Equal(t, v.Clusters, js.Clusters)
	assert.Equal(t, v.Dependencies, js.Dependencies)
}

func TestConfigValuesToJobSubmission_ZeroValues(t *testing.T) {
	v := config.JobSubmissionValues{}
	js := ConfigValuesToJobSubmission(v)

	assert.Equal(t, dao.JobSubmission{}, js)
}
