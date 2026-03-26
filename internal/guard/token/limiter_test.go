package token

import (
	"sync"
	"testing"
	"time"
)

func TestLimiter_AllowWithinLimit(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 1000})
	defer l.Stop()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	result := l.AllowAt("client1", 100, now)
	if !result.Allowed {
		t.Error("should allow request within limit")
	}
}

func TestLimiter_PerMinuteLimitExceeded(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 100})
	defer l.Stop()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	result := l.AllowAt("client1", 80, now)
	if !result.Allowed {
		t.Fatal("first request should be allowed")
	}

	result = l.AllowAt("client1", 30, now.Add(time.Second))
	if result.Allowed {
		t.Fatal("should block: 80+30=110 > 100")
	}
	if result.Window != "minute" {
		t.Errorf("window = %q, want %q", result.Window, "minute")
	}
	if result.Used != 110 {
		t.Errorf("used = %d, want 110", result.Used)
	}
	if result.Limit != 100 {
		t.Errorf("limit = %d, want 100", result.Limit)
	}
}

func TestLimiter_PerHourLimitExceeded(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 1000, MaxPerHour: 200})
	defer l.Stop()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 4; i++ {
		ts := now.Add(time.Duration(i) * 2 * time.Minute)
		result := l.AllowAt("client1", 50, ts)
		if !result.Allowed {
			t.Fatalf("request %d should be allowed (total %d <= 200)", i+1, (i+1)*50)
		}
	}

	result := l.AllowAt("client1", 50, now.Add(8*time.Minute))
	if result.Allowed {
		t.Fatal("should block: 250 > 200 per-hour limit")
	}
	if result.Window != "hour" {
		t.Errorf("window = %q, want %q", result.Window, "hour")
	}
}

func TestLimiter_SlidingWindowExpiry(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 100})
	defer l.Stop()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	result := l.AllowAt("client1", 90, now)
	if !result.Allowed {
		t.Fatal("should allow initial request")
	}

	result = l.AllowAt("client1", 20, now.Add(30*time.Second))
	if result.Allowed {
		t.Fatal("should block: 90+20=110 > 100")
	}

	// 61 seconds later: the 90-token entry drops out of the minute window.
	// Only the 20-token (rejected but recorded) entry remains in window.
	// New 20 tokens → 20 + 20 = 40 ≤ 100
	result = l.AllowAt("client1", 20, now.Add(61*time.Second))
	if !result.Allowed {
		t.Fatal("should allow after old entries expire from window")
	}
}

func TestLimiter_MultipleClients(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 100})
	defer l.Stop()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	l.AllowAt("client1", 90, now)
	l.AllowAt("client2", 50, now)

	result := l.AllowAt("client1", 20, now.Add(time.Second))
	if result.Allowed {
		t.Error("client1 should be blocked (90+20=110 > 100)")
	}

	result = l.AllowAt("client2", 40, now.Add(time.Second))
	if !result.Allowed {
		t.Error("client2 should be allowed (50+40=90 ≤ 100)")
	}
}

func TestLimiter_RetryAfter(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 100})
	defer l.Stop()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	l.AllowAt("client1", 80, now)
	result := l.AllowAt("client1", 30, now.Add(10*time.Second))

	if result.Allowed {
		t.Fatal("should be blocked")
	}
	// Oldest entry at t=0 expires at t+60s. Now is t+10s → retry ≈ 50s
	if result.RetryAfter < 49*time.Second || result.RetryAfter > 51*time.Second {
		t.Errorf("RetryAfter = %v, want ~50s", result.RetryAfter)
	}
}

func TestLimiter_NoLimitsConfigured(t *testing.T) {
	l := NewLimiter(LimiterConfig{})
	defer l.Stop()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 100; i++ {
		result := l.AllowAt("client1", 999999, now.Add(time.Duration(i)*time.Second))
		if !result.Allowed {
			t.Fatalf("request %d should be allowed when no limits configured", i)
		}
	}
}

func TestLimiter_ConcurrentAccess(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 100000})
	defer l.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			clientID := "client1"
			if id%2 == 0 {
				clientID = "client2"
			}
			l.Allow(clientID, 10)
		}(i)
	}
	wg.Wait()

	usage := l.ClientUsage("client1", time.Minute)
	if usage == 0 {
		t.Error("client1 should have some usage after concurrent requests")
	}
}

func TestLimiter_ZeroTokenRequest(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 100})
	defer l.Stop()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	result := l.AllowAt("client1", 0, now)
	if !result.Allowed {
		t.Error("zero-token request should always be allowed")
	}
}

func TestLimiter_ExactLimit(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 100})
	defer l.Stop()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	result := l.AllowAt("client1", 100, now)
	if !result.Allowed {
		t.Error("request at exact limit should be allowed")
	}

	result = l.AllowAt("client1", 1, now.Add(time.Second))
	if result.Allowed {
		t.Error("request exceeding limit by 1 should be blocked")
	}
}

func TestLimiter_ClientUsage(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 10000})
	defer l.Stop()

	l.Allow("client1", 50)
	l.Allow("client1", 30)

	usage := l.ClientUsage("client1", time.Minute)
	if usage != 80 {
		t.Errorf("usage = %d, want 80", usage)
	}

	usage = l.ClientUsage("unknown", time.Minute)
	if usage != 0 {
		t.Errorf("unknown client usage = %d, want 0", usage)
	}
}

func TestLimiter_BothWindowsConfigured(t *testing.T) {
	l := NewLimiter(LimiterConfig{MaxPerMinute: 100, MaxPerHour: 500})
	defer l.Stop()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Use 90 tokens (under minute limit)
	result := l.AllowAt("client1", 90, now)
	if !result.Allowed {
		t.Fatal("should allow first request")
	}

	// 20 more exceeds per-minute (110 > 100)
	result = l.AllowAt("client1", 20, now.Add(time.Second))
	if result.Allowed {
		t.Fatal("should block: per-minute exceeded")
	}
	if result.Window != "minute" {
		t.Errorf("window = %q, want %q", result.Window, "minute")
	}
}
