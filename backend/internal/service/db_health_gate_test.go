package service

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDBHealthGateStateTransitions(t *testing.T) {
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	gate := newDBHealthGate(func() time.Time { return now })

	for i := 0; i < dbHealthGateMinFailures-1; i++ {
		gate.MarkFailure()
		require.True(t, gate.IsHealthy())
	}

	gate.MarkFailure()
	require.False(t, gate.IsHealthy())

	now = now.Add(dbHealthGateUnhealthyDuration - time.Second)
	require.False(t, gate.IsHealthy())

	now = now.Add(2 * time.Second)
	require.True(t, gate.IsHealthy())

	gate.MarkSuccess()
	require.True(t, gate.IsHealthy())
}

func TestDBHealthGateFailureRateThreshold(t *testing.T) {
	now := time.Date(2026, 5, 25, 11, 0, 0, 0, time.UTC)
	gate := newDBHealthGate(func() time.Time { return now })

	for i := 0; i < dbHealthGateMinFailures; i++ {
		gate.MarkSuccess()
	}
	for i := 0; i < dbHealthGateMinFailures; i++ {
		gate.MarkFailure()
		gate.MarkSuccess()
	}

	require.True(t, gate.IsHealthy())

	now = now.Add(dbHealthGateWindow + time.Second)
	for i := 0; i < dbHealthGateMinFailures; i++ {
		gate.MarkFailure()
	}
	require.False(t, gate.IsHealthy())
}

func TestDBHealthGateSkipLogRateLimit(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	gate := newDBHealthGate(func() time.Time { return now })

	require.True(t, gate.ShouldLogSkip("scheduler:poll", time.Minute))
	require.False(t, gate.ShouldLogSkip("scheduler:poll", time.Minute))

	now = now.Add(time.Minute)
	require.True(t, gate.ShouldLogSkip("scheduler:poll", time.Minute))
}

func TestDBHealthGateConcurrentAccess(t *testing.T) {
	gate := newDBHealthGate(time.Now)
	var wg sync.WaitGroup

	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%3 == 0 {
				gate.MarkSuccess()
			} else {
				gate.MarkFailure()
			}
			_ = gate.IsHealthy()
			_ = gate.ShouldLogSkip("concurrent", time.Millisecond)
		}(i)
	}

	wg.Wait()
}
