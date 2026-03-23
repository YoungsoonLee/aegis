package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.Server.Listen != ":8080" {
		t.Errorf("default listen = %q, want %q", cfg.Server.Listen, ":8080")
	}
	if cfg.Server.AdminListen != ":9090" {
		t.Errorf("default admin listen = %q, want %q", cfg.Server.AdminListen, ":9090")
	}
	if !cfg.Guards.PII.Enabled {
		t.Error("PII guard should be enabled by default")
	}
	if cfg.Guards.PII.Action != "mask" {
		t.Errorf("PII default action = %q, want %q", cfg.Guards.PII.Action, "mask")
	}
	if !cfg.Guards.Injection.Enabled {
		t.Error("injection guard should be enabled by default")
	}
	if cfg.Guards.Injection.Action != "block" {
		t.Errorf("injection default action = %q, want %q", cfg.Guards.Injection.Action, "block")
	}
	if cfg.Guards.Content.Enabled {
		t.Error("content guard should be disabled by default")
	}
	if cfg.Guards.Token.Enabled {
		t.Error("token guard should be disabled by default")
	}
	if !cfg.Audit.Enabled {
		t.Error("audit should be enabled by default")
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("default log level = %q, want %q", cfg.Logging.Level, "info")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Setenv("AEGIS_CONFIG", "/nonexistent/path/aegis.yaml")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("missing config file should return defaults, got error: %v", err)
	}
	if cfg.Server.Listen != ":8080" {
		t.Error("should return default config when file is missing")
	}
}

func TestLoad_ValidYAML(t *testing.T) {
	content := `
server:
  listen: ":3000"
  admin_listen: ":3001"
guards:
  pii:
    enabled: false
  injection:
    enabled: true
    action: warn
    sensitivity: high
logging:
  level: debug
  format: json
`
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AEGIS_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Listen != ":3000" {
		t.Errorf("listen = %q, want %q", cfg.Server.Listen, ":3000")
	}
	if cfg.Server.AdminListen != ":3001" {
		t.Errorf("admin listen = %q, want %q", cfg.Server.AdminListen, ":3001")
	}
	if cfg.Guards.PII.Enabled {
		t.Error("PII should be disabled per config")
	}
	if cfg.Guards.Injection.Action != "warn" {
		t.Errorf("injection action = %q, want %q", cfg.Guards.Injection.Action, "warn")
	}
	if cfg.Guards.Injection.Sensitivity != "high" {
		t.Errorf("sensitivity = %q, want %q", cfg.Guards.Injection.Sensitivity, "high")
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("log level = %q, want %q", cfg.Logging.Level, "debug")
	}
}

func TestLoad_EnvVarExpansion(t *testing.T) {
	content := `
targets:
  - name: openai
    url: https://api.openai.com
    headers:
      Authorization: "Bearer ${TEST_API_KEY}"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "env-config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AEGIS_CONFIG", path)
	t.Setenv("TEST_API_KEY", "sk-test-12345")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Targets) != 1 {
		t.Fatalf("got %d targets, want 1", len(cfg.Targets))
	}
	want := "Bearer sk-test-12345"
	if cfg.Targets[0].Headers["Authorization"] != want {
		t.Errorf("Authorization = %q, want %q", cfg.Targets[0].Headers["Authorization"], want)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AEGIS_CONFIG", path)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_FullConfig(t *testing.T) {
	content := `
server:
  listen: ":4000"
  admin_listen: ":4001"
  tls:
    enabled: true
    cert_file: /path/to/cert
    key_file: /path/to/key
targets:
  - name: openai
    url: https://api.openai.com
    default: true
    headers:
      Authorization: "Bearer test"
  - name: anthropic
    url: https://api.anthropic.com
    headers:
      x-api-key: "test-key"
guards:
  pii:
    enabled: true
    action: block
    entities: [email, phone]
  injection:
    enabled: true
    action: block
    sensitivity: high
  content:
    enabled: true
    action: block
    denied_topics: [violence, drugs]
  token:
    enabled: true
    max_per_request: 4096
    max_per_minute: 100000
audit:
  enabled: true
  sinks:
    - type: file
      path: /tmp/audit.jsonl
    - type: stdout
      format: json
logging:
  level: warn
  format: json
`
	dir := t.TempDir()
	path := filepath.Join(dir, "full-config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AEGIS_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Server.TLS.Enabled {
		t.Error("TLS should be enabled")
	}
	if len(cfg.Targets) != 2 {
		t.Errorf("got %d targets, want 2", len(cfg.Targets))
	}
	if cfg.Guards.PII.Action != "block" {
		t.Errorf("PII action = %q, want %q", cfg.Guards.PII.Action, "block")
	}
	if len(cfg.Guards.PII.Entities) != 2 {
		t.Errorf("PII entities count = %d, want 2", len(cfg.Guards.PII.Entities))
	}
	if !cfg.Guards.Content.Enabled {
		t.Error("content guard should be enabled")
	}
	if len(cfg.Guards.Content.DeniedTopics) != 2 {
		t.Errorf("denied topics count = %d, want 2", len(cfg.Guards.Content.DeniedTopics))
	}
	if cfg.Guards.Token.MaxPerRequest != 4096 {
		t.Errorf("max per request = %d, want 4096", cfg.Guards.Token.MaxPerRequest)
	}
	if len(cfg.Audit.Sinks) != 2 {
		t.Errorf("audit sinks count = %d, want 2", len(cfg.Audit.Sinks))
	}
}
