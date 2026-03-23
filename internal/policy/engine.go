package policy

import (
	"fmt"
	"os"
	"sync"

	"github.com/YoungsoonLee/aegis/internal/config"
	"gopkg.in/yaml.v3"
)

type Engine struct {
	policies []Policy
	mu       sync.RWMutex
}

func NewEngine(cfg config.PolicyConfig) (*Engine, error) {
	e := &Engine{}

	if cfg.Path != "" {
		if err := e.LoadFromFile(cfg.Path); err != nil {
			return nil, err
		}
	}

	return e, nil
}

func (e *Engine) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading policy file %s: %w", path, err)
	}

	var pf PolicyFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return fmt.Errorf("parsing policy file %s: %w", path, err)
	}

	e.mu.Lock()
	e.policies = pf.Policies
	e.mu.Unlock()

	return nil
}

func (e *Engine) GetPolicies() []Policy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]Policy, len(e.policies))
	copy(result, e.policies)
	return result
}

func (e *Engine) Reload() error {
	// Placeholder for hot-reload support
	return nil
}
