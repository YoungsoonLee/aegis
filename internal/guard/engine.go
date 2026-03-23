package guard

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/YoungsoonLee/aegis/internal/config"
	"github.com/YoungsoonLee/aegis/internal/policy"
)

type Engine struct {
	guards       []Guard
	policyEngine *policy.Engine
	logger       *slog.Logger
	mu           sync.RWMutex
}

func NewEngine(cfg config.GuardsConfig, pe *policy.Engine, logger *slog.Logger) *Engine {
	e := &Engine{
		policyEngine: pe,
		logger:       logger,
	}

	e.registerGuards(cfg)
	return e
}

func (e *Engine) registerGuards(cfg config.GuardsConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.guards = nil

	if cfg.PII.Enabled {
		e.guards = append(e.guards, NewPIIGuard(cfg.PII))
		e.logger.Info("guard registered", "name", "pii", "action", cfg.PII.Action)
	}

	if cfg.Injection.Enabled {
		e.guards = append(e.guards, NewInjectionGuard(cfg.Injection))
		e.logger.Info("guard registered", "name", "injection", "action", cfg.Injection.Action)
	}

	if cfg.Content.Enabled {
		e.guards = append(e.guards, NewContentGuard(cfg.Content))
		e.logger.Info("guard registered", "name", "content", "action", cfg.Content.Action)
	}

	if cfg.Token.Enabled {
		e.guards = append(e.guards, NewTokenGuard(cfg.Token))
		e.logger.Info("guard registered", "name", "token")
	}
}

// Process runs all registered guards in parallel against the content.
// Returns aggregated results. If any guard blocks, the overall result is blocked.
func (e *Engine) Process(ctx context.Context, content *Content) ([]*Result, error) {
	e.mu.RLock()
	guards := make([]Guard, len(e.guards))
	copy(guards, e.guards)
	e.mu.RUnlock()

	if len(guards) == 0 {
		return nil, nil
	}

	type guardResult struct {
		result *Result
		err    error
	}

	results := make([]guardResult, len(guards))
	var wg sync.WaitGroup

	for i, g := range guards {
		wg.Add(1)
		go func(idx int, guard Guard) {
			defer wg.Done()
			start := time.Now()

			r, err := guard.Check(ctx, content)
			if err != nil {
				e.logger.Error("guard check failed",
					"guard", guard.Name(),
					"error", err,
					"duration", time.Since(start),
				)
				results[idx] = guardResult{err: err}
				return
			}

			e.logger.Debug("guard check completed",
				"guard", guard.Name(),
				"action", r.Action,
				"blocked", r.Blocked,
				"duration", time.Since(start),
			)
			results[idx] = guardResult{result: r}
		}(i, g)
	}

	wg.Wait()

	var aggregated []*Result
	for _, gr := range results {
		if gr.err != nil {
			return aggregated, gr.err
		}
		if gr.result != nil {
			aggregated = append(aggregated, gr.result)
		}
	}

	return aggregated, nil
}

// IsBlocked checks if any result in the set is a block action.
func IsBlocked(results []*Result) (*Result, bool) {
	for _, r := range results {
		if r.Blocked {
			return r, true
		}
	}
	return nil, false
}

// Guards returns the list of registered guard names.
func (e *Engine) Guards() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, len(e.guards))
	for i, g := range e.guards {
		names[i] = g.Name()
	}
	return names
}
