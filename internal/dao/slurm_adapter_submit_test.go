package dao

import (
	"testing"

	slurm "github.com/jontk/slurm-client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to dereference pointer or return zero value
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func derefUint32(p *uint32) uint32 {
	if p == nil {
		return 0
	}
	return *p
}

func TestConvertJobSubmissionToJobCreate_CoreFields(t *testing.T) {
	job := &JobSubmission{
		Name:       "test-job",
		Script:     "#!/bin/bash\necho hello",
		Partition:  "gpu",
		Account:    "research",
		CPUs:       4,
		Nodes:      2,
		TimeLimit:  "01:00:00",
		WorkingDir: "/home/user/work",
	}

	jc := convertJobSubmissionToJobCreate(job)

	require.NotNil(t, jc)
	assert.Equal(t, "test-job", derefString(jc.Name))
	assert.Equal(t, "#!/bin/bash\necho hello", derefString(jc.Script))
	assert.Equal(t, "gpu", derefString(jc.Partition))
	assert.Equal(t, "research", derefString(jc.Account))
	assert.Equal(t, int32(4), derefInt32(jc.CPUsPerTask))
	assert.Equal(t, int32(2), derefInt32(jc.MinimumNodes))
	assert.Equal(t, uint32(60), derefUint32(jc.TimeLimit))
	assert.Equal(t, "/home/user/work", derefString(jc.CurrentWorkingDirectory))
}

func TestConvertJobSubmissionToJobCreate_TimeParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint32
	}{
		{"HH:MM:SS format", "01:00:00", 60},
		{"two hours thirty", "02:30:00", 150},
		{"D-HH:MM:SS format", "1-12:30:00", 2190}, // 1*24*60 + 12*60 + 30 = 2190
		{"bare minutes", "60", 60},
		{"bare minutes large", "120", 120},
		{"invalid string defaults to 60", "invalid", 60},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &JobSubmission{
				Name:      "test",
				Script:    "#!/bin/bash\necho hi",
				TimeLimit: tt.input,
			}
			jc := convertJobSubmissionToJobCreate(job)
			assert.Equal(t, tt.expected, derefUint32(jc.TimeLimit))
		})
	}
}

func TestConvertJobSubmissionToJobCreate_MemoryParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint64
	}{
		{"gigabytes", "4G", 4096},
		{"megabytes", "1024M", 1024},
		{"bare number treated as MB", "512", 512},
		{"empty not set", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &JobSubmission{
				Name:   "test",
				Script: "#!/bin/bash\necho hi",
				Memory: tt.input,
			}
			jc := convertJobSubmissionToJobCreate(job)
			if tt.expected == 0 {
				assert.Nil(t, jc.MemoryPerNode, "MemoryPerNode should be nil for empty memory")
			} else {
				require.NotNil(t, jc.MemoryPerNode)
				assert.Equal(t, tt.expected, *jc.MemoryPerNode)
			}
		})
	}
}

func TestConvertJobSubmissionToJobCreate_ScriptShebang(t *testing.T) {
	t.Run("script without shebang gets one prepended", func(t *testing.T) {
		job := &JobSubmission{
			Name:   "test",
			Script: "echo hello world",
		}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Equal(t, "#!/bin/bash\necho hello world", derefString(jc.Script))
	})

	t.Run("script with shebang stays as-is", func(t *testing.T) {
		job := &JobSubmission{
			Name:   "test",
			Script: "#!/usr/bin/env python3\nprint('hello')",
		}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Equal(t, "#!/usr/bin/env python3\nprint('hello')", derefString(jc.Script))
	})

	t.Run("empty script stays empty", func(t *testing.T) {
		job := &JobSubmission{
			Name:   "test",
			Script: "",
		}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Equal(t, "", derefString(jc.Script))
	})
}

func TestConvertJobSubmissionToJobCreate_Environment(t *testing.T) {
	t.Run("empty env inherits os.Environ", func(t *testing.T) {
		job := &JobSubmission{
			Name:   "test",
			Script: "#!/bin/bash\necho hi",
		}
		jc := convertJobSubmissionToJobCreate(job)
		assert.NotEmpty(t, jc.Environment, "should inherit environment variables when none specified")
	})

	t.Run("custom env converted to KEY=value format", func(t *testing.T) {
		job := &JobSubmission{
			Name:   "test",
			Script: "#!/bin/bash\necho hi",
			Environment: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Len(t, jc.Environment, 2)
		assert.Contains(t, jc.Environment, "FOO=bar")
		assert.Contains(t, jc.Environment, "BAZ=qux")
	})
}

func TestConvertJobSubmissionToJobCreate_GPUs(t *testing.T) {
	job := &JobSubmission{
		Name:   "test",
		Script: "#!/bin/bash\necho hi",
		GPUs:   2,
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.TRESPerNode)
	assert.Equal(t, "gres/gpu:2", *jc.TRESPerNode)
}

func TestConvertJobSubmissionToJobCreate_GresOverridesGPUs(t *testing.T) {
	job := &JobSubmission{
		Name:   "test",
		Script: "#!/bin/bash\necho hi",
		GPUs:   2,
		Gres:   "gres/gpu:a100:4",
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.TRESPerNode)
	assert.Equal(t, "gres/gpu:a100:4", *jc.TRESPerNode, "Gres should override GPUs")
}

func TestConvertJobSubmissionToJobCreate_StringPointerFields(t *testing.T) {
	t.Run("QoS set", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", QoS: "high"}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.QoS)
		assert.Equal(t, "high", *jc.QoS)
	})

	t.Run("OutputFile set", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", OutputFile: "/tmp/out.log"}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.StandardOutput)
		assert.Equal(t, "/tmp/out.log", *jc.StandardOutput)
	})

	t.Run("ErrorFile set", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", ErrorFile: "/tmp/err.log"}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.StandardError)
		assert.Equal(t, "/tmp/err.log", *jc.StandardError)
	})

	t.Run("StandardInput set", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", StandardInput: "/tmp/input.dat"}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.StandardInput)
		assert.Equal(t, "/tmp/input.dat", *jc.StandardInput)
	})
}

func TestConvertJobSubmissionToJobCreate_Email(t *testing.T) {
	t.Run("email notify enabled with address", func(t *testing.T) {
		job := &JobSubmission{
			Name:        "test",
			Script:      "#!/bin/bash\n",
			EmailNotify: true,
			Email:       "user@test.com",
		}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Equal(t, []slurm.MailTypeValue{"ALL"}, jc.MailType)
		require.NotNil(t, jc.MailUser)
		assert.Equal(t, "user@test.com", *jc.MailUser)
	})

	t.Run("email notify disabled", func(t *testing.T) {
		job := &JobSubmission{
			Name:        "test",
			Script:      "#!/bin/bash\n",
			EmailNotify: false,
			Email:       "user@test.com",
		}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Nil(t, jc.MailType)
		assert.Nil(t, jc.MailUser)
	})
}

func TestConvertJobSubmissionToJobCreate_ArraySpec(t *testing.T) {
	job := &JobSubmission{
		Name:      "test",
		Script:    "#!/bin/bash\n",
		ArraySpec: "1-100",
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.Array)
	assert.Equal(t, "1-100", *jc.Array)
}

func TestConvertJobSubmissionToJobCreate_Exclusive(t *testing.T) {
	job := &JobSubmission{
		Name:      "test",
		Script:    "#!/bin/bash\n",
		Exclusive: true,
	}
	jc := convertJobSubmissionToJobCreate(job)
	assert.Equal(t, []slurm.SharedValue{"EXCLUSIVE"}, jc.Shared)
}

func TestConvertJobSubmissionToJobCreate_BoolFields(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*JobSubmission)
		check func(*testing.T, *slurm.JobCreate)
	}{
		{
			name:  "Requeue",
			setup: func(j *JobSubmission) { j.Requeue = true },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Requeue)
				assert.True(t, *jc.Requeue)
			},
		},
		{
			name:  "Hold",
			setup: func(j *JobSubmission) { j.Hold = true },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Hold)
				assert.True(t, *jc.Hold)
			},
		},
		{
			name:  "Contiguous",
			setup: func(j *JobSubmission) { j.Contiguous = true },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Contiguous)
				assert.True(t, *jc.Contiguous)
			},
		},
		{
			name:  "Overcommit",
			setup: func(j *JobSubmission) { j.Overcommit = true },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Overcommit)
				assert.True(t, *jc.Overcommit)
			},
		},
		{
			name:  "KillOnNodeFail",
			setup: func(j *JobSubmission) { j.KillOnNodeFail = true },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.KillOnNodeFail)
				assert.True(t, *jc.KillOnNodeFail)
			},
		},
		{
			name:  "WaitAllNodes",
			setup: func(j *JobSubmission) { j.WaitAllNodes = true },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.WaitAllNodes)
				assert.True(t, *jc.WaitAllNodes)
			},
		},
		{
			name:  "Immediate",
			setup: func(j *JobSubmission) { j.Immediate = true },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Immediate)
				assert.True(t, *jc.Immediate)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
			tt.setup(job)
			jc := convertJobSubmissionToJobCreate(job)
			tt.check(t, jc)
		})
	}
}

func TestConvertJobSubmissionToJobCreate_BoolFieldsDefaultFalse(t *testing.T) {
	job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
	jc := convertJobSubmissionToJobCreate(job)

	assert.Nil(t, jc.Requeue, "Requeue should be nil when false")
	assert.Nil(t, jc.Hold, "Hold should be nil when false")
	assert.Nil(t, jc.Contiguous, "Contiguous should be nil when false")
	assert.Nil(t, jc.Overcommit, "Overcommit should be nil when false")
	assert.Nil(t, jc.KillOnNodeFail, "KillOnNodeFail should be nil when false")
	assert.Nil(t, jc.WaitAllNodes, "WaitAllNodes should be nil when false")
	assert.Nil(t, jc.Immediate, "Immediate should be nil when false")
}

func TestConvertJobSubmissionToJobCreate_Dependencies(t *testing.T) {
	job := &JobSubmission{
		Name:         "test",
		Script:       "#!/bin/bash\n",
		Dependencies: []string{"123", "456"},
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.Dependency)
	assert.Equal(t, "afterok:123:456", *jc.Dependency)
}

func TestConvertJobSubmissionToJobCreate_Signal(t *testing.T) {
	t.Run("B:USR1@300", func(t *testing.T) {
		job := &JobSubmission{
			Name:   "test",
			Script: "#!/bin/bash\n",
			Signal: "B:USR1@300",
		}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.KillWarningSignal)
		assert.Equal(t, "USR1", *jc.KillWarningSignal)
		require.NotNil(t, jc.KillWarningDelay)
		assert.Equal(t, uint16(300), *jc.KillWarningDelay)
		assert.Equal(t, []slurm.KillWarningFlagsValue{"BATCH_JOB"}, jc.KillWarningFlags)
	})

	t.Run("USR2@60 without B: prefix", func(t *testing.T) {
		job := &JobSubmission{
			Name:   "test",
			Script: "#!/bin/bash\n",
			Signal: "USR2@60",
		}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.KillWarningSignal)
		assert.Equal(t, "USR2", *jc.KillWarningSignal)
		require.NotNil(t, jc.KillWarningDelay)
		assert.Equal(t, uint16(60), *jc.KillWarningDelay)
		assert.Nil(t, jc.KillWarningFlags, "no B: prefix means no BATCH_JOB flag")
	})

	t.Run("signal without delay", func(t *testing.T) {
		job := &JobSubmission{
			Name:   "test",
			Script: "#!/bin/bash\n",
			Signal: "B:TERM",
		}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.KillWarningSignal)
		assert.Equal(t, "TERM", *jc.KillWarningSignal)
		assert.Nil(t, jc.KillWarningDelay)
		assert.Equal(t, []slurm.KillWarningFlagsValue{"BATCH_JOB"}, jc.KillWarningFlags)
	})
}

func TestConvertJobSubmissionToJobCreate_MemoryPerCPU(t *testing.T) {
	t.Run("gigabytes", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", MemoryPerCPU: "4G"}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.MemoryPerCPU)
		assert.Equal(t, uint64(4096), *jc.MemoryPerCPU)
	})

	t.Run("megabytes", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", MemoryPerCPU: "2048M"}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.MemoryPerCPU)
		assert.Equal(t, uint64(2048), *jc.MemoryPerCPU)
	})
}

func TestConvertJobSubmissionToJobCreate_TimeMinimum(t *testing.T) {
	job := &JobSubmission{
		Name:        "test",
		Script:      "#!/bin/bash\n",
		TimeMinimum: "00:30:00",
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.TimeMinimum)
	assert.Equal(t, uint32(30), *jc.TimeMinimum)
}

func TestConvertJobSubmissionToJobCreate_StringToPointerFields(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*JobSubmission)
		check func(*testing.T, *slurm.JobCreate)
	}{
		{
			name:  "Constraints",
			setup: func(j *JobSubmission) { j.Constraints = "gpu,nvlink" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Constraints)
				assert.Equal(t, "gpu,nvlink", *jc.Constraints)
			},
		},
		{
			name:  "Distribution",
			setup: func(j *JobSubmission) { j.Distribution = "cyclic" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Distribution)
				assert.Equal(t, "cyclic", *jc.Distribution)
			},
		},
		{
			name:  "Comment",
			setup: func(j *JobSubmission) { j.Comment = "test comment" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Comment)
				assert.Equal(t, "test comment", *jc.Comment)
			},
		},
		{
			name:  "Container",
			setup: func(j *JobSubmission) { j.Container = "/path/to/container" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Container)
				assert.Equal(t, "/path/to/container", *jc.Container)
			},
		},
		{
			name:  "Prefer",
			setup: func(j *JobSubmission) { j.Prefer = "a100" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Prefer)
				assert.Equal(t, "a100", *jc.Prefer)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
			tt.setup(job)
			jc := convertJobSubmissionToJobCreate(job)
			tt.check(t, jc)
		})
	}
}

func TestConvertJobSubmissionToJobCreate_IntToPointerFields(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*JobSubmission)
		check func(*testing.T, *slurm.JobCreate)
	}{
		{
			name:  "NTasks to Tasks",
			setup: func(j *JobSubmission) { j.NTasks = 8 },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Tasks)
				assert.Equal(t, int32(8), *jc.Tasks)
			},
		},
		{
			name:  "NTasksPerNode to TasksPerNode",
			setup: func(j *JobSubmission) { j.NTasksPerNode = 4 },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.TasksPerNode)
				assert.Equal(t, int32(4), *jc.TasksPerNode)
			},
		},
		{
			name:  "ThreadsPerCore",
			setup: func(j *JobSubmission) { j.ThreadsPerCore = 2 },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.ThreadsPerCore)
				assert.Equal(t, int32(2), *jc.ThreadsPerCore)
			},
		},
		{
			name:  "TasksPerCore",
			setup: func(j *JobSubmission) { j.TasksPerCore = 1 },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.TasksPerCore)
				assert.Equal(t, int32(1), *jc.TasksPerCore)
			},
		},
		{
			name:  "TasksPerSocket",
			setup: func(j *JobSubmission) { j.TasksPerSocket = 4 },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.TasksPerSocket)
				assert.Equal(t, int32(4), *jc.TasksPerSocket)
			},
		},
		{
			name:  "SocketsPerNode",
			setup: func(j *JobSubmission) { j.SocketsPerNode = 2 },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.SocketsPerNode)
				assert.Equal(t, int32(2), *jc.SocketsPerNode)
			},
		},
		{
			name:  "MaximumNodes",
			setup: func(j *JobSubmission) { j.MaximumNodes = 10 },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.MaximumNodes)
				assert.Equal(t, int32(10), *jc.MaximumNodes)
			},
		},
		{
			name:  "MaximumCPUs",
			setup: func(j *JobSubmission) { j.MaximumCPUs = 64 },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.MaximumCPUs)
				assert.Equal(t, int32(64), *jc.MaximumCPUs)
			},
		},
		{
			name:  "MinimumCPUsPerNode",
			setup: func(j *JobSubmission) { j.MinimumCPUsPerNode = 8 },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.MinimumCPUsPerNode)
				assert.Equal(t, int32(8), *jc.MinimumCPUsPerNode)
			},
		},
		{
			name:  "MinimumCPUs",
			setup: func(j *JobSubmission) { j.MinimumCPUs = 16 },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.MinimumCPUs)
				assert.Equal(t, int32(16), *jc.MinimumCPUs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
			tt.setup(job)
			jc := convertJobSubmissionToJobCreate(job)
			tt.check(t, jc)
		})
	}
}

func TestConvertJobSubmissionToJobCreate_OpenMode(t *testing.T) {
	job := &JobSubmission{
		Name:     "test",
		Script:   "#!/bin/bash\n",
		OpenMode: "append",
	}
	jc := convertJobSubmissionToJobCreate(job)
	assert.Equal(t, []slurm.OpenModeValue{"APPEND"}, jc.OpenMode)
}

func TestConvertJobSubmissionToJobCreate_Flags(t *testing.T) {
	job := &JobSubmission{
		Name:   "test",
		Script: "#!/bin/bash\n",
		Flags:  "SPREAD_JOB,KILL_INVALID_DEPENDENCY",
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.Len(t, jc.Flags, 2)
	assert.Equal(t, slurm.FlagsValue("SPREAD_JOB"), jc.Flags[0])
	assert.Equal(t, slurm.FlagsValue("KILL_INVALID_DEPENDENCY"), jc.Flags[1])
}

func TestConvertJobSubmissionToJobCreate_CPUBindingFlags(t *testing.T) {
	job := &JobSubmission{
		Name:            "test",
		Script:          "#!/bin/bash\n",
		CPUBindingFlags: "VERBOSE,CPU_BIND_TO_CORES",
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.Len(t, jc.CPUBindingFlags, 2)
	assert.Contains(t, jc.CPUBindingFlags, slurm.CPUBindingFlagsValue("VERBOSE"))
	assert.Contains(t, jc.CPUBindingFlags, slurm.CPUBindingFlagsValue("CPU_BIND_TO_CORES"))
}

func TestConvertJobSubmissionToJobCreate_MemoryBindingType(t *testing.T) {
	job := &JobSubmission{
		Name:              "test",
		Script:            "#!/bin/bash\n",
		MemoryBindingType: "LOCAL,VERBOSE",
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.Len(t, jc.MemoryBindingType, 2)
	assert.Contains(t, jc.MemoryBindingType, slurm.MemoryBindingTypeValue("LOCAL"))
	assert.Contains(t, jc.MemoryBindingType, slurm.MemoryBindingTypeValue("VERBOSE"))
}

func TestConvertJobSubmissionToJobCreate_ProfileTypes(t *testing.T) {
	job := &JobSubmission{
		Name:         "test",
		Script:       "#!/bin/bash\n",
		ProfileTypes: "ENERGY,NETWORK",
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.Len(t, jc.Profile, 2)
	assert.Contains(t, jc.Profile, slurm.ProfileValue("ENERGY"))
	assert.Contains(t, jc.Profile, slurm.ProfileValue("NETWORK"))
}

func TestConvertJobSubmissionToJobCreate_TRESStringFields(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*JobSubmission)
		check func(*testing.T, *slurm.JobCreate)
	}{
		{
			name:  "CPUsPerTRES",
			setup: func(j *JobSubmission) { j.CPUsPerTRES = "gres/gpu:2" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.CPUsPerTRES)
				assert.Equal(t, "gres/gpu:2", *jc.CPUsPerTRES)
			},
		},
		{
			name:  "MemoryPerTRES",
			setup: func(j *JobSubmission) { j.MemoryPerTRES = "gres/gpu:4096" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.MemoryPerTRES)
				assert.Equal(t, "gres/gpu:4096", *jc.MemoryPerTRES)
			},
		},
		{
			name:  "TRESPerTask",
			setup: func(j *JobSubmission) { j.TRESPerTask = "gres/gpu:1" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.TRESPerTask)
				assert.Equal(t, "gres/gpu:1", *jc.TRESPerTask)
			},
		},
		{
			name:  "TRESPerSocket",
			setup: func(j *JobSubmission) { j.TRESPerSocket = "gres/gpu:2" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.TRESPerSocket)
				assert.Equal(t, "gres/gpu:2", *jc.TRESPerSocket)
			},
		},
		{
			name:  "TRESPerJob",
			setup: func(j *JobSubmission) { j.TRESPerJob = "gres/gpu:8" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.TRESPerJob)
				assert.Equal(t, "gres/gpu:8", *jc.TRESPerJob)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
			tt.setup(job)
			jc := convertJobSubmissionToJobCreate(job)
			tt.check(t, jc)
		})
	}
}

func TestConvertJobSubmissionToJobCreate_Argv(t *testing.T) {
	job := &JobSubmission{
		Name:   "test",
		Script: "#!/bin/bash\n",
		Argv:   "arg1 arg2 arg3",
	}
	jc := convertJobSubmissionToJobCreate(job)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, jc.Argv)
}

func TestConvertJobSubmissionToJobCreate_RequiredSwitches(t *testing.T) {
	job := &JobSubmission{
		Name:             "test",
		Script:           "#!/bin/bash\n",
		RequiredSwitches: 3,
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.RequiredSwitches)
	assert.Equal(t, uint32(3), *jc.RequiredSwitches)
}

func TestConvertJobSubmissionToJobCreate_WaitForSwitch(t *testing.T) {
	job := &JobSubmission{
		Name:          "test",
		Script:        "#!/bin/bash\n",
		WaitForSwitch: 120,
	}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.WaitForSwitch)
	assert.Equal(t, int32(120), *jc.WaitForSwitch)
}

func TestConvertJobSubmissionToJobCreate_ClusterFields(t *testing.T) {
	t.Run("ClusterConstraint", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", ClusterConstraint: "gpu_cluster"}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.ClusterConstraint)
		assert.Equal(t, "gpu_cluster", *jc.ClusterConstraint)
	})

	t.Run("Clusters", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", Clusters: "cluster1,cluster2"}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.Clusters)
		assert.Equal(t, "cluster1,cluster2", *jc.Clusters)
	})
}

func TestConvertJobSubmissionToJobCreate_EmptyFieldsAreNil(t *testing.T) {
	job := &JobSubmission{
		Name:   "test",
		Script: "#!/bin/bash\n",
	}
	jc := convertJobSubmissionToJobCreate(job)

	assert.Nil(t, jc.QoS, "QoS should be nil when empty")
	assert.Nil(t, jc.StandardOutput, "StandardOutput should be nil when empty")
	assert.Nil(t, jc.StandardError, "StandardError should be nil when empty")
	assert.Nil(t, jc.StandardInput, "StandardInput should be nil when empty")
	assert.Nil(t, jc.MemoryPerNode, "MemoryPerNode should be nil when empty")
	assert.Nil(t, jc.TRESPerNode, "TRESPerNode should be nil when no GPUs/Gres")
	assert.Nil(t, jc.Array, "Array should be nil when empty")
	assert.Nil(t, jc.Dependency, "Dependency should be nil when no deps")
	assert.Nil(t, jc.Constraints, "Constraints should be nil when empty")
	assert.Nil(t, jc.Tasks, "Tasks should be nil when 0")
	assert.Nil(t, jc.TasksPerNode, "TasksPerNode should be nil when 0")
	assert.Nil(t, jc.Comment, "Comment should be nil when empty")
	assert.Nil(t, jc.Distribution, "Distribution should be nil when empty")
	assert.Nil(t, jc.Prefer, "Prefer should be nil when empty")
	assert.Nil(t, jc.Container, "Container should be nil when empty")
	assert.Nil(t, jc.ThreadsPerCore, "ThreadsPerCore should be nil when 0")
	assert.Nil(t, jc.TasksPerCore, "TasksPerCore should be nil when 0")
	assert.Nil(t, jc.TasksPerSocket, "TasksPerSocket should be nil when 0")
	assert.Nil(t, jc.SocketsPerNode, "SocketsPerNode should be nil when 0")
	assert.Nil(t, jc.MaximumNodes, "MaximumNodes should be nil when 0")
	assert.Nil(t, jc.MaximumCPUs, "MaximumCPUs should be nil when 0")
	assert.Nil(t, jc.MinimumCPUsPerNode, "MinimumCPUsPerNode should be nil when 0")
	assert.Nil(t, jc.TimeMinimum, "TimeMinimum should be nil when empty")
	assert.Nil(t, jc.MemoryPerCPU, "MemoryPerCPU should be nil when empty")
	assert.Nil(t, jc.RequiredSwitches, "RequiredSwitches should be nil when 0")
	assert.Nil(t, jc.WaitForSwitch, "WaitForSwitch should be nil when 0")
	assert.Nil(t, jc.ClusterConstraint, "ClusterConstraint should be nil when empty")
	assert.Nil(t, jc.Clusters, "Clusters should be nil when empty")
	assert.Nil(t, jc.CPUsPerTRES, "CPUsPerTRES should be nil when empty")
	assert.Nil(t, jc.MemoryPerTRES, "MemoryPerTRES should be nil when empty")
	assert.Nil(t, jc.TRESPerTask, "TRESPerTask should be nil when empty")
	assert.Nil(t, jc.TRESPerSocket, "TRESPerSocket should be nil when empty")
	assert.Nil(t, jc.TRESPerJob, "TRESPerJob should be nil when empty")
	assert.Nil(t, jc.Flags, "Flags should be nil when empty")
	assert.Nil(t, jc.Profile, "Profile should be nil when empty")
	assert.Nil(t, jc.CPUBindingFlags, "CPUBindingFlags should be nil when empty")
	assert.Nil(t, jc.MemoryBindingType, "MemoryBindingType should be nil when empty")
	assert.Empty(t, jc.Shared, "Shared should be empty when not exclusive")
	assert.Nil(t, jc.MailType, "MailType should be nil when no email notify")
}

func TestConvertJobSubmissionToJobCreate_ExcludeNodes(t *testing.T) {
	t.Run("wraps entire comma-separated string in single-element slice", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", ExcludeNodes: "node01,node02"}
		jc := convertJobSubmissionToJobCreate(job)
		// The whole string is wrapped in one element, NOT split on commas
		assert.Equal(t, []string{"node01,node02"}, jc.ExcludedNodes)
	})

	t.Run("single node", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", ExcludeNodes: "node01"}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Equal(t, []string{"node01"}, jc.ExcludedNodes)
	})

	t.Run("empty stays nil", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Nil(t, jc.ExcludedNodes)
	})
}

func TestConvertJobSubmissionToJobCreate_RequiredNodes(t *testing.T) {
	t.Run("wraps entire comma-separated string in single-element slice", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", RequiredNodes: "node03,node04"}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Equal(t, []string{"node03,node04"}, jc.RequiredNodes)
	})

	t.Run("empty stays nil", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Nil(t, jc.RequiredNodes)
	})
}

func TestConvertJobSubmissionToJobCreate_X11(t *testing.T) {
	t.Run("string uppercased to X11Value slice", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", X11: "batch"}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Equal(t, []slurm.X11Value{"BATCH"}, jc.X11)
	})

	t.Run("already uppercase", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", X11: "FORWARD"}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Equal(t, []slurm.X11Value{"FORWARD"}, jc.X11)
	})

	t.Run("empty stays nil", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Nil(t, jc.X11)
	})
}

func TestConvertJobSubmissionToJobCreate_Priority(t *testing.T) {
	t.Run("positive priority", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", Priority: 100}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.Priority)
		assert.Equal(t, uint32(100), *jc.Priority)
	})

	t.Run("zero stays nil", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Nil(t, jc.Priority)
	})
}

func TestConvertJobSubmissionToJobCreate_Nice(t *testing.T) {
	t.Run("negative nice value", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", Nice: -10}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.Nice)
		assert.Equal(t, int32(-10), *jc.Nice)
	})

	t.Run("positive nice value", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", Nice: 50}
		jc := convertJobSubmissionToJobCreate(job)
		require.NotNil(t, jc.Nice)
		assert.Equal(t, int32(50), *jc.Nice)
	})

	t.Run("zero stays nil", func(t *testing.T) {
		job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
		jc := convertJobSubmissionToJobCreate(job)
		assert.Nil(t, jc.Nice)
	})
}

func TestConvertJobSubmissionToJobCreate_Deadline(t *testing.T) {
	// Deadline uses parseBeginTime, which accepts ISO date-time strings
	job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", Deadline: "2024-06-15T14:30:00"}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.Deadline, "Deadline should be set for valid datetime string")
	assert.Greater(t, *jc.Deadline, int64(0), "Deadline should be a positive unix timestamp")
}

func TestConvertJobSubmissionToJobCreate_TmpDiskPerNode(t *testing.T) {
	job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", TmpDiskPerNode: 1024}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.TemporaryDiskPerNode)
	assert.Equal(t, int32(1024), *jc.TemporaryDiskPerNode)
}

func TestConvertJobSubmissionToJobCreate_NTasksPerTRES(t *testing.T) {
	job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", NTasksPerTRES: 4}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.NtasksPerTRES)
	assert.Equal(t, int32(4), *jc.NtasksPerTRES)
}

func TestConvertJobSubmissionToJobCreate_CoreSpecification(t *testing.T) {
	job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", CoreSpecification: 2}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.CoreSpecification)
	assert.Equal(t, int32(2), *jc.CoreSpecification)
}

func TestConvertJobSubmissionToJobCreate_ThreadSpecification(t *testing.T) {
	job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n", ThreadSpecification: 4}
	jc := convertJobSubmissionToJobCreate(job)
	require.NotNil(t, jc.ThreadSpecification)
	assert.Equal(t, int32(4), *jc.ThreadSpecification)
}

func TestConvertJobSubmissionToJobCreate_RemainingStringPointerFields(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*JobSubmission)
		check func(*testing.T, *slurm.JobCreate)
	}{
		{
			name:  "Reservation",
			setup: func(j *JobSubmission) { j.Reservation = "gpu_reservation" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Reservation)
				assert.Equal(t, "gpu_reservation", *jc.Reservation)
			},
		},
		{
			name:  "Licenses",
			setup: func(j *JobSubmission) { j.Licenses = "matlab:2,ansys:1" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Licenses)
				assert.Equal(t, "matlab:2,ansys:1", *jc.Licenses)
			},
		},
		{
			name:  "Wckey",
			setup: func(j *JobSubmission) { j.Wckey = "project-alpha" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Wckey)
				assert.Equal(t, "project-alpha", *jc.Wckey)
			},
		},
		{
			name:  "CPUBinding",
			setup: func(j *JobSubmission) { j.CPUBinding = "cores" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.CPUBinding)
				assert.Equal(t, "cores", *jc.CPUBinding)
			},
		},
		{
			name:  "CPUFrequency",
			setup: func(j *JobSubmission) { j.CPUFrequency = "high" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.CPUFrequency)
				assert.Equal(t, "high", *jc.CPUFrequency)
			},
		},
		{
			name:  "Network",
			setup: func(j *JobSubmission) { j.Network = "sn_all:torus" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.Network)
				assert.Equal(t, "sn_all:torus", *jc.Network)
			},
		},
		{
			name:  "BurstBuffer",
			setup: func(j *JobSubmission) { j.BurstBuffer = "#BB create_persistent name=bb1 capacity=100GB" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.BurstBuffer)
				assert.Equal(t, "#BB create_persistent name=bb1 capacity=100GB", *jc.BurstBuffer)
			},
		},
		{
			name:  "BatchFeatures",
			setup: func(j *JobSubmission) { j.BatchFeatures = "haswell" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.BatchFeatures)
				assert.Equal(t, "haswell", *jc.BatchFeatures)
			},
		},
		{
			name:  "TRESBind",
			setup: func(j *JobSubmission) { j.TRESBind = "gpu:verbose" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.TRESBind)
				assert.Equal(t, "gpu:verbose", *jc.TRESBind)
			},
		},
		{
			name:  "TRESFreq",
			setup: func(j *JobSubmission) { j.TRESFreq = "gpu:high" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.TRESFreq)
				assert.Equal(t, "gpu:high", *jc.TRESFreq)
			},
		},
		{
			name:  "MemoryBinding",
			setup: func(j *JobSubmission) { j.MemoryBinding = "local" },
			check: func(t *testing.T, jc *slurm.JobCreate) {
				require.NotNil(t, jc.MemoryBinding)
				assert.Equal(t, "local", *jc.MemoryBinding)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
			tt.setup(job)
			jc := convertJobSubmissionToJobCreate(job)
			tt.check(t, jc)
		})
	}
}

func TestConvertJobSubmissionToJobCreate_EmptyFieldsAreNil_Extended(t *testing.T) {
	job := &JobSubmission{Name: "test", Script: "#!/bin/bash\n"}
	jc := convertJobSubmissionToJobCreate(job)

	assert.Nil(t, jc.Reservation, "Reservation should be nil when empty")
	assert.Nil(t, jc.Licenses, "Licenses should be nil when empty")
	assert.Nil(t, jc.Wckey, "Wckey should be nil when empty")
	assert.Nil(t, jc.ExcludedNodes, "ExcludedNodes should be nil when empty")
	assert.Nil(t, jc.Priority, "Priority should be nil when 0")
	assert.Nil(t, jc.Nice, "Nice should be nil when 0")
	assert.Nil(t, jc.Deadline, "Deadline should be nil when empty")
	assert.Nil(t, jc.TemporaryDiskPerNode, "TemporaryDiskPerNode should be nil when 0")
	assert.Nil(t, jc.NtasksPerTRES, "NtasksPerTRES should be nil when 0")
	assert.Nil(t, jc.CPUBinding, "CPUBinding should be nil when empty")
	assert.Nil(t, jc.CPUFrequency, "CPUFrequency should be nil when empty")
	assert.Nil(t, jc.Network, "Network should be nil when empty")
	assert.Nil(t, jc.X11, "X11 should be nil when empty")
	assert.Nil(t, jc.BurstBuffer, "BurstBuffer should be nil when empty")
	assert.Nil(t, jc.BatchFeatures, "BatchFeatures should be nil when empty")
	assert.Nil(t, jc.TRESBind, "TRESBind should be nil when empty")
	assert.Nil(t, jc.TRESFreq, "TRESFreq should be nil when empty")
	assert.Nil(t, jc.CoreSpecification, "CoreSpecification should be nil when 0")
	assert.Nil(t, jc.ThreadSpecification, "ThreadSpecification should be nil when 0")
	assert.Nil(t, jc.MemoryBinding, "MemoryBinding should be nil when empty")
	assert.Nil(t, jc.RequiredNodes, "RequiredNodes should be nil when empty")
}

func TestConvertJobSubmissionToJobCreate_EnvironmentFiltersPosixKeys(t *testing.T) {
	// Set an env var with a valid POSIX name and verify it shows up
	// when no custom environment is specified.
	key := "S9S_TEST_POSIX_VAR"
	t.Setenv(key, "testvalue")

	job := &JobSubmission{
		Name:   "test",
		Script: "#!/bin/bash\n",
	}
	jc := convertJobSubmissionToJobCreate(job)

	found := false
	for _, e := range jc.Environment {
		if e == key+"=testvalue" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected to find S9S_TEST_POSIX_VAR in inherited environment")
}
