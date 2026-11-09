package sshUtils

import (
	"fmt"
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

func TestHostMetadata_ExtractionAndLineNumbers(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_test")
	if err := os.WriteFile(keyPath, []byte("fake-ssh-key"), 0600); err != nil {
		t.Fatalf("failed to write test key: %v", err)
	}
	if err := os.Chmod(keyPath, 0600); err != nil {
		t.Fatalf("failed to chmod test key: %v", err)
	}

	configPath := filepath.Join(tmpDir, "config")
	configContent := fmt.Sprintf(`# SSH Config Test File
# Line 2 comment
Host prod-web
    HostName 192.168.1.10
    User deploy
    Port 2222
    Tag prod web
    IdentityFile %s
    RemoteCommand htop
    LocalForward 5432 127.0.0.1:5432
    DynamicForward 1080
    RemoteForward 8080 127.0.0.1:80
    ProxyJump bastion.corp
    ProxyCommand ssh -W %%h:%%p bastion.corp

Host default-host
    HostName 10.0.0.1
`, keyPath)

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	hosts, err := GetAllHosts(configPath)
	if err != nil {
		t.Fatalf("GetAllHosts failed: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	meta, ok := HostsMeta["prod-web"]
	if !ok {
		t.Fatalf("HostsMeta missing prod-web")
	}

	if meta.Alias != "prod-web" {
		t.Errorf("expected Alias 'prod-web', got %q", meta.Alias)
	}
	if meta.HostName != "192.168.1.10" {
		t.Errorf("expected HostName '192.168.1.10', got %q", meta.HostName)
	}
	if meta.User != "deploy" {
		t.Errorf("expected User 'deploy', got %q", meta.User)
	}
	if meta.Port != "2222" {
		t.Errorf("expected Port '2222', got %q", meta.Port)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "prod" || meta.Tags[1] != "web" {
		t.Errorf("expected Tags ['prod', 'web'], got %v", meta.Tags)
	}
	if meta.IdentityFile != keyPath {
		t.Errorf("expected IdentityFile %q, got %q", keyPath, meta.IdentityFile)
	}
	if !meta.KeyExists {
		t.Errorf("expected KeyExists true, got false")
	}
	if meta.KeyError != "" {
		t.Errorf("expected empty KeyError, got %q", meta.KeyError)
	}
	if meta.RemoteCommand != "htop" {
		t.Errorf("expected RemoteCommand 'htop', got %q", meta.RemoteCommand)
	}
	if meta.LocalForward != "5432 127.0.0.1:5432" {
		t.Errorf("expected LocalForward '5432 127.0.0.1:5432', got %q", meta.LocalForward)
	}
	if meta.DynamicForward != "1080" {
		t.Errorf("expected DynamicForward '1080', got %q", meta.DynamicForward)
	}
	if meta.RemoteForward != "8080 127.0.0.1:80" {
		t.Errorf("expected RemoteForward '8080 127.0.0.1:80', got %q", meta.RemoteForward)
	}
	if meta.ProxyJump != "bastion.corp" {
		t.Errorf("expected ProxyJump 'bastion.corp', got %q", meta.ProxyJump)
	}
	if meta.ProxyCommand != "ssh -W %h:%p bastion.corp" {
		t.Errorf("expected ProxyCommand 'ssh -W %%h:%%p bastion.corp', got %q", meta.ProxyCommand)
	}
	absConfigPath, _ := filepath.Abs(configPath)
	if meta.SourceFile != absConfigPath {
		t.Errorf("expected SourceFile %q, got %q", absConfigPath, meta.SourceFile)
	}
	if meta.SourceLine != 3 {
		t.Errorf("expected SourceLine 3, got %d", meta.SourceLine)
	}

	// Test default-host
	defMeta, ok := HostsMeta["default-host"]
	if !ok {
		t.Fatalf("HostsMeta missing default-host")
	}
	if defMeta.Port != "22" {
		t.Errorf("expected default Port '22', got %q", defMeta.Port)
	}
	if defMeta.IdentityFile != "" {
		t.Errorf("expected empty IdentityFile for default-host, got %q", defMeta.IdentityFile)
	}
	if !defMeta.KeyExists {
		t.Errorf("expected KeyExists true for default-host, got false")
	}
	if defMeta.KeyError != "" {
		t.Errorf("expected empty KeyError for default-host, got %q", defMeta.KeyError)
	}
	if defMeta.SourceLine != 16 {
		t.Errorf("expected SourceLine 16, got %d", defMeta.SourceLine)
	}
}

func TestHostMetadata_KeyPermissions(t *testing.T) {
	tmpDir := t.TempDir()

	// Insecure key (0644)
	insecureKey := filepath.Join(tmpDir, "insecure_key")
	if err := os.WriteFile(insecureKey, []byte("key"), 0644); err != nil {
		t.Fatalf("failed to write insecure key: %v", err)
	}
	if err := os.Chmod(insecureKey, 0644); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}

	missingKey := filepath.Join(tmpDir, "missing_key")

	configPath := filepath.Join(tmpDir, "config")
	configContent := fmt.Sprintf(`
Host insecure-host
    HostName 10.0.0.1
    IdentityFile %s

Host missing-key-host
    HostName 10.0.0.2
    IdentityFile %s
`, insecureKey, missingKey)

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err := GetAllHosts(configPath)
	if err != nil {
		t.Fatalf("GetAllHosts failed: %v", err)
	}

	insecureMeta := HostsMeta["insecure-host"]
	if !insecureMeta.KeyExists {
		t.Errorf("expected KeyExists true for insecure key, got false")
	}
	if insecureMeta.KeyError != "insecure permissions (0644)" {
		t.Errorf("expected KeyError 'insecure permissions (0644)', got %q", insecureMeta.KeyError)
	}

	missingMeta := HostsMeta["missing-key-host"]
	if missingMeta.KeyExists {
		t.Errorf("expected KeyExists false for missing key, got true")
	}
	if missingMeta.KeyError != "file not found" {
		t.Errorf("expected KeyError 'file not found', got %q", missingMeta.KeyError)
	}
}

func TestHostMetadata_IncludeSourceTracking(t *testing.T) {
	tmpDir := t.TempDir()
	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("failed to create conf.d dir: %v", err)
	}

	subFile := filepath.Join(confDir, "sub.conf")
	subContent := `# Line 1
# Line 2
Host sub-host
    HostName 10.20.30.40
    User subuser
`
	if err := os.WriteFile(subFile, []byte(subContent), 0600); err != nil {
		t.Fatalf("failed to write sub.conf: %v", err)
	}

	mainFile := filepath.Join(tmpDir, "config")
	mainContent := `
Host root-host
    HostName 1.2.3.4

Include conf.d/*.conf
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0600); err != nil {
		t.Fatalf("failed to write main config: %v", err)
	}

	_, err := GetAllHosts(mainFile)
	if err != nil {
		t.Fatalf("GetAllHosts failed: %v", err)
	}

	absMain, _ := filepath.Abs(mainFile)
	absSub, _ := filepath.Abs(subFile)

	rootMeta, ok := HostsMeta["root-host"]
	if !ok {
		t.Fatalf("missing root-host in HostsMeta")
	}
	if rootMeta.SourceFile != absMain {
		t.Errorf("expected root-host SourceFile %q, got %q", absMain, rootMeta.SourceFile)
	}
	if rootMeta.SourceLine != 2 {
		t.Errorf("expected root-host SourceLine 2, got %d", rootMeta.SourceLine)
	}

	subMeta, ok := HostsMeta["sub-host"]
	if !ok {
		t.Fatalf("missing sub-host in HostsMeta")
	}
	if subMeta.SourceFile != absSub {
		t.Errorf("expected sub-host SourceFile %q, got %q", absSub, subMeta.SourceFile)
	}
	if subMeta.SourceLine != 3 {
		t.Errorf("expected sub-host SourceLine 3, got %d", subMeta.SourceLine)
	}
	if subMeta.HostName != "10.20.30.40" {
		t.Errorf("expected sub-host HostName '10.20.30.40', got %q", subMeta.HostName)
	}
}

func TestSearchHostname_AdvancedQueries(t *testing.T) {
	tmpDir := t.TempDir()

	validKey := filepath.Join(tmpDir, "id_valid")
	if err := os.WriteFile(validKey, []byte("valid-key"), 0600); err != nil {
		t.Fatalf("failed to write valid key: %v", err)
	}

	insecureKey := filepath.Join(tmpDir, "id_insecure")
	if err := os.WriteFile(insecureKey, []byte("insecure-key"), 0644); err != nil {
		t.Fatalf("failed to write insecure key: %v", err)
	}
	if err := os.Chmod(insecureKey, 0644); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}

	missingKey := filepath.Join(tmpDir, "id_missing")

	configPath := filepath.Join(tmpDir, "config")
	configContent := fmt.Sprintf(`
Host prod-web
    HostName 192.168.1.10
    User deploy
    Tag prod web
    IdentityFile %s

Host prod-db
    HostName 192.168.1.20
    User postgres
    Tag prod db database
    ProxyJump bastion.corp
    LocalForward 5432 127.0.0.1:5432

Host bastion-box
    HostName 10.0.0.254
    User admin
    Tag infra
    ProxyCommand ssh -W %%h:%%p gateway.internal
    DynamicForward 1080

Host backup-runner
    HostName 10.0.0.50
    User backup
    Tag backup
    RemoteForward 8080 127.0.0.1:80
    RemoteCommand run-backup.sh

Host badkey-missing
    HostName 10.0.0.60
    User ubuntu
    Tag test
    IdentityFile %s

Host badkey-insecure
    HostName 10.0.0.70
    User ubuntu
    IdentityFile %s

Host staging-api
    HostName api.staging.internal
    User ubuntu
    Tag staging api

Host plain-host
    HostName 10.0.0.99
    User guest
`, validKey, missingKey, insecureKey)

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	hosts, err := GetAllHosts(configPath)
	if err != nil {
		t.Fatalf("GetAllHosts failed: %v", err)
	}

	allExpected := []string{
		"prod-web", "prod-db", "bastion-box", "backup-runner",
		"badkey-missing", "badkey-insecure", "staging-api", "plain-host",
	}

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		// Tag prefix (#)
		{
			name:     "#prod matches prod hosts",
			query:    "#prod",
			expected: []string{"prod-web", "prod-db"},
		},
		{
			name:     "#web matches web host",
			query:    "#web",
			expected: []string{"prod-web"},
		},
		{
			name:     "#db matches db host",
			query:    "#db",
			expected: []string{"prod-db"},
		},
		{
			name:     "#infra matches bastion",
			query:    "#infra",
			expected: []string{"bastion-box"},
		},
		{
			name:     "# prefix alone matches all tagged hosts",
			query:    "#",
			expected: []string{"prod-web", "prod-db", "bastion-box", "backup-runner", "badkey-missing", "staging-api"},
		},

		// Tag prefix (tag:)
		{
			name:     "tag:prod matches prod hosts",
			query:    "tag:prod",
			expected: []string{"prod-web", "prod-db"},
		},
		{
			name:     "tag:api matches staging-api",
			query:    "tag:api",
			expected: []string{"staging-api"},
		},
		{
			name:     "TAG:PROD case-insensitive",
			query:    "TAG:PROD",
			expected: []string{"prod-web", "prod-db"},
		},

		// Colon directive (:tag <val>, :t <val>)
		{
			name:     ":tag prod matches prod hosts",
			query:    ":tag prod",
			expected: []string{"prod-web", "prod-db"},
		},
		{
			name:     ":tag database matches prod-db",
			query:    ":tag database",
			expected: []string{"prod-db"},
		},
		{
			name:     ":t infra matches bastion-box",
			query:    ":t infra",
			expected: []string{"bastion-box"},
		},
		{
			name:     ":t web matches prod-web",
			query:    ":t web",
			expected: []string{"prod-web"},
		},

		// Colon directive (:proxy, :bastion)
		{
			name:     ":proxy matches ProxyJump and ProxyCommand hosts",
			query:    ":proxy",
			expected: []string{"prod-db", "bastion-box"},
		},
		{
			name:     ":bastion matches ProxyJump and ProxyCommand hosts",
			query:    ":bastion",
			expected: []string{"prod-db", "bastion-box"},
		},
		{
			name:     ":PROXY case-insensitive",
			query:    ":PROXY",
			expected: []string{"prod-db", "bastion-box"},
		},

		// Colon directive (:fwd, :forward, :tunnel)
		{
			name:     ":fwd matches port forward hosts",
			query:    ":fwd",
			expected: []string{"prod-db", "bastion-box", "backup-runner"},
		},
		{
			name:     ":forward matches port forward hosts",
			query:    ":forward",
			expected: []string{"prod-db", "bastion-box", "backup-runner"},
		},
		{
			name:     ":tunnel matches port forward hosts",
			query:    ":tunnel",
			expected: []string{"prod-db", "bastion-box", "backup-runner"},
		},

		// Colon directive (:cmd)
		{
			name:     ":cmd matches RemoteCommand hosts",
			query:    ":cmd",
			expected: []string{"backup-runner"},
		},

		// Colon directive (:keys, :badkeys)
		{
			name:     ":keys matches hosts with configured IdentityFile",
			query:    ":keys",
			expected: []string{"prod-web", "badkey-missing", "badkey-insecure"},
		},
		{
			name:     ":badkeys matches missing or insecure key hosts",
			query:    ":badkeys",
			expected: []string{"badkey-missing", "badkey-insecure"},
		},

		// Colon directive (:all, :)
		{
			name:     ":all matches all hosts",
			query:    ":all",
			expected: allExpected,
		},
		{
			name:     ": alone matches all hosts",
			query:    ":",
			expected: allExpected,
		},
		{
			name:     ":unknown matches nothing",
			query:    ":nonexistent",
			expected: nil,
		},

		// Inverted matches (!<val>)
		{
			name:  "!staging excludes staging-api",
			query: "!staging",
			expected: []string{
				"prod-web", "prod-db", "bastion-box", "backup-runner",
				"badkey-missing", "badkey-insecure", "plain-host",
			},
		},
		{
			name:  "!prod excludes prod hosts",
			query: "!prod",
			expected: []string{
				"bastion-box", "backup-runner", "badkey-missing", "badkey-insecure",
				"staging-api", "plain-host",
			},
		},
		{
			name:     "! alone matches all hosts",
			query:    "!",
			expected: allExpected,
		},

		// Fallback substring matching with tags
		{
			name:     "fallback substring matches tag 'database'",
			query:    "database",
			expected: []string{"prod-db"},
		},
		{
			name:     "fallback substring matches user 'guest'",
			query:    "guest",
			expected: []string{"plain-host"},
		},
		{
			name:     "empty query matches all",
			query:    "",
			expected: allExpected,
		},
		{
			name:     "whitespace query matches all",
			query:    "   ",
			expected: allExpected,
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

func TestSearchHostname_FallbackWithoutHostsMeta(t *testing.T) {
	// Test that SearchHostname does not panic and resolves directives when HostsMeta is unpopulated
	oldMeta := HostsMeta
	defer func() { HostsMeta = oldMeta }()
	HostsMeta = make(map[string]HostMetadata)

	results := SearchHostname([]string{"some-host"}, "")
	if len(results) != 1 || results[0] != "some-host" {
		t.Errorf("expected ['some-host'], got %v", results)
	}

	results = SearchHostname([]string{"some-host"}, "some")
	if len(results) != 1 || results[0] != "some-host" {
		t.Errorf("expected ['some-host'], got %v", results)
	}

	results = SearchHostname([]string{"some-host"}, "#nonexistent")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %v", results)
	}
}
