package config

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Targets  []Target       `yaml:"targets"`
	Guards   GuardsConfig   `yaml:"guards"`
	Policies PolicyConfig   `yaml:"policies"`
	Audit    AuditConfig    `yaml:"audit"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	Listen      string    `yaml:"listen"`
	AdminListen string    `yaml:"admin_listen"`
	TLS         TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type Target struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
	Default bool              `yaml:"default"`
}

type GuardsConfig struct {
	PII       PIIGuardConfig       `yaml:"pii"`
	Injection InjectionGuardConfig `yaml:"injection"`
	Content   ContentGuardConfig   `yaml:"content"`
	Token     TokenGuardConfig     `yaml:"token"`
}

type PIIGuardConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Action   string   `yaml:"action"`
	Entities []string `yaml:"entities"`
}

type InjectionGuardConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Action      string `yaml:"action"`
	Sensitivity string `yaml:"sensitivity"`
}

type ContentGuardConfig struct {
	Enabled         bool                       `yaml:"enabled"`
	Action          string                     `yaml:"action"`
	DeniedTopics    []string                   `yaml:"denied_topics"`
	Categories      map[string]CategoryConfig  `yaml:"categories"`
	AllowedContexts []string                   `yaml:"allowed_contexts"`
}

type CategoryConfig struct {
	Action   string   `yaml:"action"`
	Keywords []string `yaml:"keywords"`
	Phrases  []string `yaml:"phrases"`
	Severity string   `yaml:"severity"`
}

type TokenGuardConfig struct {
	Enabled       bool  `yaml:"enabled"`
	MaxPerRequest int64 `yaml:"max_per_request"`
	MaxPerMinute  int64 `yaml:"max_per_minute"`
}

type PolicyConfig struct {
	Path string `yaml:"path"`
}

type AuditConfig struct {
	Enabled bool         `yaml:"enabled"`
	Sinks   []SinkConfig `yaml:"sinks"`
}

type SinkConfig struct {
	Type   string `yaml:"type"`
	Path   string `yaml:"path,omitempty"`
	Format string `yaml:"format,omitempty"`
	URL    string `yaml:"url,omitempty"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}
