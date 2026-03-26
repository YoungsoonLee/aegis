package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigPath = "aegis.yaml"
	envPrefix         = "AEGIS_"
)

func Load() (*Config, error) {
	path := os.Getenv("AEGIS_CONFIG")
	if path == "" {
		path = defaultConfigPath
	}

	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	expanded := os.Expand(string(data), envVarMapper)

	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Listen:      ":8080",
			AdminListen: ":9090",
		},
		Guards: GuardsConfig{
			PII: PIIGuardConfig{
				Enabled:  true,
				Action:   "mask",
				Entities: []string{"email", "phone", "ssn", "credit_card", "api_key"},
			},
			Injection: InjectionGuardConfig{
				Enabled:     true,
				Action:      "block",
				Sensitivity: "medium",
			},
			Content: ContentGuardConfig{
				Enabled: false,
				Action:  "block",
			},
		Token: TokenGuardConfig{
			Enabled:       false,
			Action:        "block",
			MaxPerRequest: 8192,
			MaxPerMinute:  200000,
		},
			Schema: SchemaGuardConfig{
				Enabled: false,
				Action:  "warn",
			},
		},
		Audit: AuditConfig{
			Enabled: true,
			Sinks: []SinkConfig{
				{Type: "stdout", Format: "json"},
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func envVarMapper(key string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	aegisKey := envPrefix + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if val, ok := os.LookupEnv(aegisKey); ok {
		return val
	}
	return ""
}
