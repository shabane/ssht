package sshUtils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchHostname(t *testing.T) {
	// Create a temporary ssh config for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	configContent := `
Host prod-web
    HostName 192.168.1.10
    User deploy

Host prod-db
    HostName 192.168.1.20
    User postgres
    ProxyJump bastion.corp

Host staging-api
    HostName api.staging.internal
    User ubuntu
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	hosts, err := GetAllHosts(configPath)
	if err != nil {
		t.Fatalf("GetAllHosts failed: %v", err)
	}

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "Search by alias exact",
			query:    "prod-web",
			expected: []string{"prod-web"},
		},
		{
			name:     "Search by alias case-insensitive substring",
			query:    "PROD",
			expected: []string{"prod-web", "prod-db"},
		},
		{
			name:     "Search by IP",
			query:    "192.168.1.20",
			expected: []string{"prod-db"},
		},
		{
			name:     "Search by IP prefix",
			query:    "192.168.1.",
			expected: []string{"prod-web", "prod-db"},
		},
		{
			name:     "Search by User",
			query:    "ubuntu",
			expected: []string{"staging-api"},
		},
		{
			name:     "Search by user@host",
			query:    "deploy@192.168",
			expected: []string{"prod-web"},
		},
		{
			name:     "Search by ProxyJump bastion",
			query:    "bastion.corp",
			expected: []string{"prod-db"},
		},
		{
			name:     "Empty query matches all",
			query:    "",
			expected: []string{"prod-web", "prod-db", "staging-api"},
		},
		{
			name:     "Non-existent query matches none",
			query:    "nonexistent-server-xyz",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SearchHostname(hosts, tt.query)
			if len(results) != len(tt.expected) {
				t.Fatalf("query %q: expected %d hosts (%v), got %d (%v)",
					tt.query, len(tt.expected), tt.expected, len(results), results)
			}
			for i, exp := range tt.expected {
				if results[i] != exp {
					t.Errorf("query %q at index %d: expected %s, got %s", tt.query, i, exp, results[i])
				}
			}
		})
	}
}

func TestGetAllHosts_IncludeAndWildcards(t *testing.T) {
	tmpDir := t.TempDir()
	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("failed to create conf.d dir: %v", err)
	}

	// Sub-config 1
	sub1 := `
Host included-host-1
    HostName 10.0.0.1
`
	if err := os.WriteFile(filepath.Join(confDir, "sub1.conf"), []byte(sub1), 0600); err != nil {
		t.Fatalf("failed to write sub1: %v", err)
	}

	// Sub-config 2
	sub2 := `
Host included-host-2
    HostName 10.0.0.2
`
	if err := os.WriteFile(filepath.Join(confDir, "sub2.conf"), []byte(sub2), 0600); err != nil {
		t.Fatalf("failed to write sub2: %v", err)
	}

	// Main config with wildcards and Include
	mainConfig := `
Host *
    ServerAliveInterval 60

Host root-host
    HostName 172.16.0.1

Host *.internal
    User defaultuser

Include conf.d/*.conf
`
	mainPath := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(mainPath, []byte(mainConfig), 0600); err != nil {
		t.Fatalf("failed to write main config: %v", err)
	}

	hosts, err := GetAllHosts(mainPath)
	if err != nil {
		t.Fatalf("GetAllHosts failed: %v", err)
	}

	expected := []string{"root-host", "included-host-1", "included-host-2"}
	if len(hosts) != len(expected) {
		t.Fatalf("expected %d hosts (%v), got %d (%v)", len(expected), expected, len(hosts), hosts)
	}
	for i, exp := range expected {
		if hosts[i] != exp {
			t.Errorf("at index %d: expected %s, got %s", i, exp, hosts[i])
		}
	}

	if MaxHostLen < len("included-host-1") {
		t.Errorf("MaxHostLen expected at least %d, got %d", len("included-host-1"), MaxHostLen)
	}
}

func TestGetAllHosts_MissingFile(t *testing.T) {
	_, err := GetAllHosts("/nonexistent/file/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
