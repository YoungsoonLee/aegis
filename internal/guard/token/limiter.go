package token

import (
	"sync"
	"time"
)

// LimiterConfig configures the sliding-window rate limiter.
type LimiterConfig struct {
	MaxPerMinute int64
	MaxPerHour   int64
}

// LimitResult contains the outcome of a rate limit check.
type LimitResult struct {
	Allowed    bool
	RetryAfter time.Duration
	Used       int64
	Limit      int64
	Window     string // "minute" or "hour"
}

// Limiter enforces per-client token usage limits using a sliding window log.
// Thread-safe for concurrent use from multiple goroutines.
type Limiter struct {
	cfg     LimiterConfig
	mu      sync.Mutex
	clients map[string]*clientState
	stopCh  chan struct{}
}

type clientState struct {
	entries []entry
}

type entry struct {
	ts     time.Time
	tokens int64
}

// NewLimiter creates a rate limiter with background cleanup of expired entries.
// Call Stop() when the limiter is no longer needed.
func NewLimiter(cfg LimiterConfig) *Limiter {
	l := &Limiter{
		cfg:     cfg,
		clients: make(map[string]*clientState),
		stopCh:  make(chan struct{}),
	}
	go l.cleanupLoop()
	return l
}

// Allow checks rate limits and records token usage for the client.
// Usage is always recorded (even when exceeding limits) to prevent gaming.
func (l *Limiter) Allow(clientID string, tokens int64) LimitResult {
	return l.AllowAt(clientID, tokens, time.Now())
}

// AllowAt is like Allow but accepts an explicit timestamp for deterministic testing.
func (l *Limiter) AllowAt(clientID string, tokens int64, now time.Time) LimitResult {
	l.mu.Lock()
	defer l.mu.Unlock()

	state := l.clients[clientID]
	if state == nil {
		state = &clientState{}
		l.clients[clientID] = state
	}

	state.pruneOlderThan(now.Add(-time.Hour))
	state.entries = append(state.entries, entry{ts: now, tokens: tokens})

	if l.cfg.MaxPerMinute > 0 {
		windowStart := now.Add(-time.Minute)
		used := state.sumSince(windowStart)
		if used > l.cfg.MaxPerMinute {
			return LimitResult{
				Allowed:    false,
				RetryAfter: state.retryAfter(windowStart, now, time.Minute),
				Used:       used,
				Limit:      l.cfg.MaxPerMinute,
				Window:     "minute",
			}
		}
	}

	if l.cfg.MaxPerHour > 0 {
		windowStart := now.Add(-time.Hour)
		used := state.sumSince(windowStart)
		if used > l.cfg.MaxPerHour {
			return LimitResult{
				Allowed:    false,
				RetryAfter: state.retryAfter(windowStart, now, time.Hour),
				Used:       used,
				Limit:      l.cfg.MaxPerHour,
				Window:     "hour",
			}
		}
	}

	return LimitResult{Allowed: true}
}

// ClientUsage returns current token usage for a client within the given window.
func (l *Limiter) ClientUsage(clientID string, window time.Duration) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	state := l.clients[clientID]
	if state == nil {
		return 0
	}
	return state.sumSince(time.Now().Add(-window))
}

// Stop terminates the background cleanup goroutine.
func (l *Limiter) Stop() {
	close(l.stopCh)
}

func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stopCh:
			return
		}
	}
}

func (l *Limiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-time.Hour)
	for id, state := range l.clients {
		state.pruneOlderThan(cutoff)
		if len(state.entries) == 0 {
			delete(l.clients, id)
		}
	}
}

func (cs *clientState) pruneOlderThan(cutoff time.Time) {
	i := 0
	for i < len(cs.entries) && cs.entries[i].ts.Before(cutoff) {
		i++
	}
	if i > 0 {
		cs.entries = cs.entries[i:]
	}
}

func (cs *clientState) sumSince(since time.Time) int64 {
	var sum int64
	for _, e := range cs.entries {
		if !e.ts.Before(since) {
			sum += e.tokens
		}
	}
	return sum
}

// retryAfter calculates how long until the oldest entry in the window expires.
func (cs *clientState) retryAfter(windowStart, now time.Time, windowDuration time.Duration) time.Duration {
	for _, e := range cs.entries {
		if !e.ts.Before(windowStart) {
			ra := e.ts.Add(windowDuration).Sub(now)
			if ra > 0 {
				return ra
			}
		}
	}
	return time.Second
}
