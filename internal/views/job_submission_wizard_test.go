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
		Name:      "original",
		Partition: "normal",
		TimeLimit: "01:00:00",
		Nodes:     1,
		CPUs:      1,
	}

	src := &dao.JobSubmission{
		Partition: "gpu",
		CPUs:      8,
		GPUs:      2,
		Memory:    "16G",
	}

	overlayJobDefaults(dst, src)

	// Overwritten by src
	assert.Equal(t, "gpu", dst.Partition)
	assert.Equal(t, 8, dst.CPUs)
	assert.Equal(t, 2, dst.GPUs)
	assert.Equal(t, "16G", dst.Memory)
	// Preserved from dst (src was zero)
	assert.Equal(t, "original", dst.Name)
	assert.Equal(t, "01:00:00", dst.TimeLimit)
	assert.Equal(t, 1, dst.Nodes)
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
		Name:        "test-job",
		Script:      "#!/bin/bash\necho hi",
		Partition:   "gpu",
		Account:     "myaccount",
		QoS:         "high",
		Nodes:       4,
		CPUs:        16,
		Memory:      "32G",
		GPUs:        2,
		TimeLimit:   "12:00:00",
		WorkingDir:  "/tmp/work",
		OutputFile:  "out.log",
		ErrorFile:   "err.log",
		EmailNotify: true,
		Email:       "user@example.com",
		ArraySpec:   "1-10",
		Exclusive:   true,
		Requeue:     true,
	}

	js := ConfigValuesToJobSubmission(v)

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
}

func TestConfigValuesToJobSubmission_ZeroValues(t *testing.T) {
	v := config.JobSubmissionValues{}
	js := ConfigValuesToJobSubmission(v)

	assert.Equal(t, dao.JobSubmission{}, js)
}
