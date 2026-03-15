package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobSubmissionFromMap_FullMap(t *testing.T) {
	m := map[string]any{
		"name":        "test-job",
		"script":      "#!/bin/bash\necho hello",
		"partition":   "gpu",
		"account":     "research",
		"qos":         "high",
		"nodes":       4,
		"cpus":        16,
		"memory":      "32G",
		"gpus":        2,
		"timeLimit":   "12:00:00",
		"workingDir":  "/home/user/work",
		"outputFile":  "out_%j.log",
		"errorFile":   "err_%j.log",
		"emailNotify": true,
		"email":       "user@example.com",
		"arraySpec":   "1-100",
		"exclusive":   true,
		"requeue":     true,
	}

	js := JobSubmissionFromMap(m)

	assert.Equal(t, "test-job", js.Name)
	assert.Equal(t, "#!/bin/bash\necho hello", js.Script)
	assert.Equal(t, "gpu", js.Partition)
	assert.Equal(t, "research", js.Account)
	assert.Equal(t, "high", js.QoS)
	assert.Equal(t, 4, js.Nodes)
	assert.Equal(t, 16, js.CPUs)
	assert.Equal(t, "32G", js.Memory)
	assert.Equal(t, 2, js.GPUs)
	assert.Equal(t, "12:00:00", js.TimeLimit)
	assert.Equal(t, "/home/user/work", js.WorkingDir)
	assert.Equal(t, "out_%j.log", js.OutputFile)
	assert.Equal(t, "err_%j.log", js.ErrorFile)
	assert.True(t, js.EmailNotify)
	assert.Equal(t, "user@example.com", js.Email)
	assert.Equal(t, "1-100", js.ArraySpec)
	assert.True(t, js.Exclusive)
	assert.True(t, js.Requeue)
}

func TestJobSubmissionFromMap_PartialMap(t *testing.T) {
	m := map[string]any{
		"name":      "partial-job",
		"partition": "normal",
		"cpus":      8,
	}

	js := JobSubmissionFromMap(m)

	assert.Equal(t, "partial-job", js.Name)
	assert.Equal(t, "normal", js.Partition)
	assert.Equal(t, 8, js.CPUs)
	// Unset fields should be zero values
	assert.Equal(t, "", js.Script)
	assert.Equal(t, "", js.Account)
	assert.Equal(t, "", js.QoS)
	assert.Equal(t, 0, js.Nodes)
	assert.Equal(t, "", js.Memory)
	assert.Equal(t, 0, js.GPUs)
	assert.Equal(t, "", js.TimeLimit)
	assert.Equal(t, "", js.WorkingDir)
	assert.Equal(t, "", js.OutputFile)
	assert.Equal(t, "", js.ErrorFile)
	assert.False(t, js.EmailNotify)
	assert.Equal(t, "", js.Email)
	assert.Equal(t, "", js.ArraySpec)
	assert.False(t, js.Exclusive)
	assert.False(t, js.Requeue)
}

func TestJobSubmissionFromMap_EmptyMap(t *testing.T) {
	js := JobSubmissionFromMap(map[string]any{})

	assert.Equal(t, JobSubmissionValues{}, js)
}

func TestJobSubmissionFromMap_Float64Numbers(t *testing.T) {
	// YAML unmarshaling into map[string]any produces float64 for numbers
	m := map[string]any{
		"nodes": float64(8),
		"cpus":  float64(32),
		"gpus":  float64(4),
	}

	js := JobSubmissionFromMap(m)

	assert.Equal(t, 8, js.Nodes)
	assert.Equal(t, 32, js.CPUs)
	assert.Equal(t, 4, js.GPUs)
}

func TestJobSubmissionFromMap_Dependencies(t *testing.T) {
	m := map[string]any{
		"dependencies": []any{"123", "456", "789"},
	}
	js := JobSubmissionFromMap(m)
	assert.Equal(t, []string{"123", "456", "789"}, js.Dependencies)
}

func TestJobSubmissionFromMap_DependenciesEmpty(t *testing.T) {
	m := map[string]any{
		"dependencies": []any{},
	}
	js := JobSubmissionFromMap(m)
	assert.Nil(t, js.Dependencies)
}

func TestResolveTemplateSources_NilConfig(t *testing.T) {
	sources := ResolveTemplateSources(nil)
	assert.Equal(t, []string{"builtin", "config", "saved"}, sources)
}

func TestResolveTemplateSources_ExplicitSources(t *testing.T) {
	cfg := &JobSubmissionConfig{
		TemplateSources: []string{"config", "saved"},
	}
	sources := ResolveTemplateSources(cfg)
	assert.Equal(t, []string{"config", "saved"}, sources)
}

func TestResolveTemplateSources_BackwardCompatShowBuiltinFalse(t *testing.T) {
	val := false
	cfg := &JobSubmissionConfig{
		ShowBuiltinTemplates: &val,
	}
	sources := ResolveTemplateSources(cfg)
	assert.Equal(t, []string{"config", "saved"}, sources)
}

func TestResolveTemplateSources_BackwardCompatShowBuiltinTrue(t *testing.T) {
	val := true
	cfg := &JobSubmissionConfig{
		ShowBuiltinTemplates: &val,
	}
	sources := ResolveTemplateSources(cfg)
	assert.Equal(t, []string{"builtin", "config", "saved"}, sources)
}

func TestResolveTemplateSources_ExplicitOverridesShowBuiltin(t *testing.T) {
	val := false
	cfg := &JobSubmissionConfig{
		ShowBuiltinTemplates: &val,
		TemplateSources:      []string{"builtin"}, // explicit wins
	}
	sources := ResolveTemplateSources(cfg)
	assert.Equal(t, []string{"builtin"}, sources)
}

func TestHasTemplateSource(t *testing.T) {
	sources := []string{"builtin", "config"}
	assert.True(t, HasTemplateSource(sources, "builtin"))
	assert.True(t, HasTemplateSource(sources, "config"))
	assert.False(t, HasTemplateSource(sources, "saved"))
}
