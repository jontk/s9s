package discovery

import (
	"testing"
)

func TestParseScontrolPingOutput(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedFirst ScontrolResult
		expectError   bool
	}{
		{
			name:          "single primary controller UP",
			input:         "Slurmctld(primary) at slurm-controller1 is UP",
			expectedCount: 1,
			expectedFirst: ScontrolResult{
				Hostname: "slurm-controller1",
				Role:     "primary",
				Status:   "UP",
			},
			expectError: false,
		},
		{
			name:          "single backup controller DOWN",
			input:         "Slurmctld(backup) at slurm-controller2 is DOWN",
			expectedCount: 1,
			expectedFirst: ScontrolResult{
				Hostname: "slurm-controller2",
				Role:     "backup",
				Status:   "DOWN",
			},
			expectError: false,
		},
		{
			name: "multiple controllers",
			input: `Slurmctld(primary) at slurm-controller1 is UP
Slurmctld(backup) at slurm-controller2 is DOWN`,
			expectedCount: 2,
			expectedFirst: ScontrolResult{
				Hostname: "slurm-controller1",
				Role:     "primary",
				Status:   "UP",
			},
			expectError: false,
		},
		{
			name:          "controller without role",
			input:         "Slurmctld at slurm-controller is UP",
			expectedCount: 1,
			expectedFirst: ScontrolResult{
				Hostname: "slurm-controller",
				Role:     "primary",
				Status:   "UP",
			},
			expectError: false,
		},
		{
			name:          "empty output",
			input:         "",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "whitespace only",
			input:         "   \n   \n   ",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "controller with FQDN",
			input:         "Slurmctld(primary) at slurm-controller1.cluster.local is UP",
			expectedCount: 1,
			expectedFirst: ScontrolResult{
				Hostname: "slurm-controller1.cluster.local",
				Role:     "primary",
				Status:   "UP",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseScontrolPingOutput(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(results) != tt.expectedCount {
				t.Errorf("expected %d results, got %d", tt.expectedCount, len(results))
				return
			}

			if tt.expectedCount > 0 {
				if results[0].Hostname != tt.expectedFirst.Hostname {
					t.Errorf("expected hostname %q, got %q", tt.expectedFirst.Hostname, results[0].Hostname)
				}
				if results[0].Role != tt.expectedFirst.Role {
					t.Errorf("expected role %q, got %q", tt.expectedFirst.Role, results[0].Role)
				}
				if results[0].Status != tt.expectedFirst.Status {
					t.Errorf("expected status %q, got %q", tt.expectedFirst.Status, results[0].Status)
				}
			}
		})
	}
}

func TestGetControllerHostname(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedHostname string
		expectedFound    bool
	}{
		{
			name:             "primary UP",
			input:            "Slurmctld(primary) at slurm-controller1 is UP",
			expectedHostname: "slurm-controller1",
			expectedFound:    true,
		},
		{
			name:             "primary DOWN, backup UP",
			input:            "Slurmctld(primary) at slurm-controller1 is DOWN\nSlurmctld(backup) at slurm-controller2 is UP",
			expectedHostname: "slurm-controller2",
			expectedFound:    true,
		},
		{
			name:             "all DOWN",
			input:            "Slurmctld(primary) at slurm-controller1 is DOWN",
			expectedHostname: "",
			expectedFound:    false,
		},
		{
			name:             "empty input",
			input:            "",
			expectedHostname: "",
			expectedFound:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostname, found := GetControllerHostname(tt.input)

			if found != tt.expectedFound {
				t.Errorf("expected found=%v, got %v", tt.expectedFound, found)
			}

			if hostname != tt.expectedHostname {
				t.Errorf("expected hostname %q, got %q", tt.expectedHostname, hostname)
			}
		})
	}
}

func TestScontrolDiscoveryName(t *testing.T) {
	sd := NewScontrolDiscovery()
	if sd.Name() != "scontrol" {
		t.Errorf("expected name 'scontrol', got %q", sd.Name())
	}
}

func TestScontrolDiscoveryPriority(t *testing.T) {
	sd := NewScontrolDiscovery()
	if sd.Priority() != 5 {
		t.Errorf("expected priority 5, got %d", sd.Priority())
	}
}

func TestResultToCluster(t *testing.T) {
	sd := &ScontrolDiscovery{
		defaultPort: 6820,
	}

	tests := []struct {
		name          string
		result        ScontrolResult
		expectNil     bool
		expectedHost  string
		expectedPort  int
		minConfidence float64
	}{
		{
			name: "primary UP",
			result: ScontrolResult{
				Hostname: "slurm-controller1",
				Role:     "primary",
				Status:   "UP",
			},
			expectNil:     false,
			expectedHost:  "slurm-controller1",
			expectedPort:  6820,
			minConfidence: 0.9,
		},
		{
			name: "backup UP",
			result: ScontrolResult{
				Hostname: "slurm-controller2",
				Role:     "backup",
				Status:   "UP",
			},
			expectNil:     false,
			expectedHost:  "slurm-controller2",
			expectedPort:  6820,
			minConfidence: 0.85,
		},
		{
			name: "controller DOWN",
			result: ScontrolResult{
				Hostname: "slurm-controller1",
				Role:     "primary",
				Status:   "DOWN",
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := sd.resultToCluster(tt.result)

			if tt.expectNil {
				if cluster != nil {
					t.Errorf("expected nil cluster, got %+v", cluster)
				}
				return
			}

			if cluster == nil {
				t.Errorf("expected non-nil cluster")
				return
			}

			if cluster.Host != tt.expectedHost {
				t.Errorf("expected host %q, got %q", tt.expectedHost, cluster.Host)
			}

			if cluster.Port != tt.expectedPort {
				t.Errorf("expected port %d, got %d", tt.expectedPort, cluster.Port)
			}

			if cluster.Confidence < tt.minConfidence {
				t.Errorf("expected confidence >= %f, got %f", tt.minConfidence, cluster.Confidence)
			}

			if len(cluster.RestEndpoints) == 0 {
				t.Errorf("expected at least one REST endpoint")
			}
		})
	}
}
