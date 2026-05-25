package service

import (
	"log/slog"
	"sync"
	"time"
)

const (
	dbHealthGateWindow            = 60 * time.Second
	dbHealthGateUnhealthyDuration = 60 * time.Second
	dbHealthGateMinFailures       = 10
	dbHealthGateFailureRate       = 0.8
	dbHealthGateSkipLogInterval   = 5 * time.Minute
)

type dbHealthGate struct {
	mu sync.Mutex

	now func() time.Time

	windowStart         time.Time
	total               int
	failures            int
	consecutiveFailures int

	unhealthyUntil time.Time
	skipLogAt      map[string]time.Time
}

var defaultDBHealthGate = newDBHealthGate(time.Now)

func newDBHealthGate(now func() time.Time) *dbHealthGate {
	if now == nil {
		now = time.Now
	}
	return &dbHealthGate{
		now:         now,
		windowStart: now(),
		skipLogAt:   make(map[string]time.Time),
	}
}

func DefaultDBHealthGate() *dbHealthGate {
	return defaultDBHealthGate
}

func (g *dbHealthGate) IsHealthy() bool {
	if g == nil {
		return true
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now()
	return !now.Before(g.unhealthyUntil)
}

func (g *dbHealthGate) MarkFailure() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now()
	g.rollWindowLocked(now)

	g.total++
	g.failures++
	g.consecutiveFailures++
	if g.failures < dbHealthGateMinFailures {
		return
	}
	if g.consecutiveFailures < dbHealthGateMinFailures && float64(g.failures)/float64(g.total) < dbHealthGateFailureRate {
		return
	}

	unhealthyUntil := now.Add(dbHealthGateUnhealthyDuration)
	if unhealthyUntil.After(g.unhealthyUntil) {
		g.unhealthyUntil = unhealthyUntil
	}
}

func (g *dbHealthGate) MarkSuccess() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now()
	if !g.unhealthyUntil.IsZero() {
		g.windowStart = now
		g.total = 0
		g.failures = 0
		g.consecutiveFailures = 0
		g.unhealthyUntil = time.Time{}
		return
	}

	g.rollWindowLocked(now)
	g.total++
	g.consecutiveFailures = 0
}

func (g *dbHealthGate) ShouldLogSkip(key string, interval time.Duration) bool {
	if g == nil {
		return false
	}
	if interval <= 0 {
		interval = dbHealthGateSkipLogInterval
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now()
	if g.skipLogAt == nil {
		g.skipLogAt = make(map[string]time.Time)
	}
	last := g.skipLogAt[key]
	if !last.IsZero() && now.Sub(last) < interval {
		return false
	}
	g.skipLogAt[key] = now
	return true
}

func (g *dbHealthGate) LogSkip(component, operation string) {
	if g == nil {
		return
	}
	key := component + ":" + operation
	if !g.ShouldLogSkip(key, dbHealthGateSkipLogInterval) {
		return
	}
	slog.Info(
		"[DBHealthGate] skipping background DB work while unhealthy",
		"component", component,
		"operation", operation,
		"unhealthy_until", g.UnhealthyUntil().Format(time.RFC3339),
	)
}

func (g *dbHealthGate) UnhealthyUntil() time.Time {
	if g == nil {
		return time.Time{}
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.unhealthyUntil
}

func (g *dbHealthGate) Reset() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	g.windowStart = g.now()
	g.total = 0
	g.failures = 0
	g.consecutiveFailures = 0
	g.unhealthyUntil = time.Time{}
	g.skipLogAt = make(map[string]time.Time)
}

func (g *dbHealthGate) rollWindowLocked(now time.Time) {
	if g.windowStart.IsZero() || now.Sub(g.windowStart) >= dbHealthGateWindow {
		g.windowStart = now
		g.total = 0
		g.failures = 0
		g.consecutiveFailures = 0
	}
}
