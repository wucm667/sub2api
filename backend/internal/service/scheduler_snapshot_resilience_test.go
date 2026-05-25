package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type schedulerResilienceCache struct {
	buckets           []SchedulerBucket
	watermark         int64
	setWatermarkCalls int
}

func (c *schedulerResilienceCache) GetSnapshot(context.Context, SchedulerBucket) ([]*Account, bool, error) {
	return nil, false, nil
}

func (c *schedulerResilienceCache) SetSnapshot(context.Context, SchedulerBucket, []Account) error {
	return nil
}

func (c *schedulerResilienceCache) GetAccount(context.Context, int64) (*Account, error) {
	return nil, nil
}

func (c *schedulerResilienceCache) SetAccount(context.Context, *Account) error {
	return nil
}

func (c *schedulerResilienceCache) DeleteAccount(context.Context, int64) error {
	return nil
}

func (c *schedulerResilienceCache) UpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}

func (c *schedulerResilienceCache) TryLockBucket(context.Context, SchedulerBucket, time.Duration) (bool, error) {
	return true, nil
}

func (c *schedulerResilienceCache) UnlockBucket(context.Context, SchedulerBucket) error {
	return nil
}

func (c *schedulerResilienceCache) ListBuckets(context.Context) ([]SchedulerBucket, error) {
	return c.buckets, nil
}

func (c *schedulerResilienceCache) GetOutboxWatermark(context.Context) (int64, error) {
	return c.watermark, nil
}

func (c *schedulerResilienceCache) SetOutboxWatermark(_ context.Context, id int64) error {
	c.setWatermarkCalls++
	c.watermark = id
	return nil
}

type schedulerOutboxRepoStub struct {
	listAfterCalls int
	listErr        error
}

func (r *schedulerOutboxRepoStub) ListAfter(context.Context, int64, int) ([]SchedulerOutboxEvent, error) {
	r.listAfterCalls++
	if r.listErr != nil {
		return nil, r.listErr
	}
	return nil, nil
}

func (r *schedulerOutboxRepoStub) MaxID(context.Context) (int64, error) {
	return 0, nil
}

type schedulerFailingAccountRepo struct {
	listUngroupedByPlatformCalls int
	err                          error
}

func (r *schedulerFailingAccountRepo) Create(context.Context, *Account) error { return nil }
func (r *schedulerFailingAccountRepo) GetByID(context.Context, int64) (*Account, error) {
	return nil, ErrAccountNotFound
}
func (r *schedulerFailingAccountRepo) GetByIDs(context.Context, []int64) ([]*Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) ExistsByID(context.Context, int64) (bool, error) {
	return false, nil
}
func (r *schedulerFailingAccountRepo) GetByCRSAccountID(context.Context, string) (*Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) FindByExtraField(context.Context, string, any) ([]Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) Update(context.Context, *Account) error { return nil }
func (r *schedulerFailingAccountRepo) Delete(context.Context, int64) error    { return nil }
func (r *schedulerFailingAccountRepo) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *schedulerFailingAccountRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, int64, string) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *schedulerFailingAccountRepo) ListByGroup(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) ListActive(context.Context) ([]Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) ListByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) UpdateLastUsed(context.Context, int64) error { return nil }
func (r *schedulerFailingAccountRepo) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}
func (r *schedulerFailingAccountRepo) SetError(context.Context, int64, string) error {
	return nil
}
func (r *schedulerFailingAccountRepo) ClearError(context.Context, int64) error { return nil }
func (r *schedulerFailingAccountRepo) SetSchedulable(context.Context, int64, bool) error {
	return nil
}
func (r *schedulerFailingAccountRepo) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *schedulerFailingAccountRepo) BindGroups(context.Context, int64, []int64) error {
	return nil
}
func (r *schedulerFailingAccountRepo) ListSchedulable(context.Context) ([]Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	return nil, nil
}

func (r *schedulerFailingAccountRepo) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	r.listUngroupedByPlatformCalls++
	return nil, r.err
}

func (r *schedulerFailingAccountRepo) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (r *schedulerFailingAccountRepo) SetRateLimited(context.Context, int64, time.Time) error {
	return nil
}
func (r *schedulerFailingAccountRepo) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	return nil
}
func (r *schedulerFailingAccountRepo) SetOverloaded(context.Context, int64, time.Time) error {
	return nil
}
func (r *schedulerFailingAccountRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (r *schedulerFailingAccountRepo) ClearTempUnschedulable(context.Context, int64) error {
	return nil
}
func (r *schedulerFailingAccountRepo) ClearRateLimit(context.Context, int64) error { return nil }
func (r *schedulerFailingAccountRepo) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return nil
}
func (r *schedulerFailingAccountRepo) ClearModelRateLimits(context.Context, int64) error {
	return nil
}
func (r *schedulerFailingAccountRepo) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	return nil
}
func (r *schedulerFailingAccountRepo) UpdateExtra(context.Context, int64, map[string]any) error {
	return nil
}
func (r *schedulerFailingAccountRepo) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (r *schedulerFailingAccountRepo) IncrementQuotaUsed(context.Context, int64, float64) error {
	return nil
}
func (r *schedulerFailingAccountRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func TestSchedulerSnapshotPollOutboxSkipsWhenDBUnhealthy(t *testing.T) {
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	gate := newDBHealthGate(func() time.Time { return now })
	for i := 0; i < dbHealthGateMinFailures; i++ {
		gate.MarkFailure()
	}

	outbox := &schedulerOutboxRepoStub{}
	svc := NewSchedulerSnapshotService(&schedulerResilienceCache{}, outbox, nil, nil, nil)
	svc.dbGate = gate

	svc.pollOutbox()

	require.Zero(t, outbox.listAfterCalls)
}

func TestSchedulerSnapshotPollOutboxFailureDoesNotAdvanceWatermark(t *testing.T) {
	gate := newDBHealthGate(time.Now)
	cache := &schedulerResilienceCache{watermark: 10}
	outbox := &schedulerOutboxRepoStub{listErr: errors.New("db down")}
	svc := NewSchedulerSnapshotService(cache, outbox, nil, nil, nil)
	svc.dbGate = gate

	for i := 0; i < dbHealthGateMinFailures; i++ {
		svc.pollOutbox()
	}

	require.Equal(t, dbHealthGateMinFailures, outbox.listAfterCalls)
	require.Zero(t, cache.setWatermarkCalls)
	require.False(t, gate.IsHealthy())
}

func TestSchedulerSnapshotRebuildBackoffSkipsAfterContinuousFailures(t *testing.T) {
	now := time.Date(2026, 5, 25, 10, 30, 0, 0, time.UTC)
	gate := newDBHealthGate(func() time.Time { return now })
	cache := &schedulerResilienceCache{
		buckets: []SchedulerBucket{{GroupID: 0, Platform: PlatformOpenAI, Mode: SchedulerModeSingle}},
	}
	accountRepo := &schedulerFailingAccountRepo{err: errors.New("db down")}
	svc := NewSchedulerSnapshotService(cache, nil, accountRepo, nil, nil)
	svc.dbGate = gate
	svc.now = func() time.Time { return now }

	for i := 0; i < schedulerRebuildBackoffFailureThreshold; i++ {
		require.Error(t, svc.triggerFullRebuild("test"))
	}
	require.True(t, now.Before(svc.rebuildBackoffUntil))

	calls := accountRepo.listUngroupedByPlatformCalls
	require.NoError(t, svc.triggerFullRebuild("test"))
	require.Equal(t, calls, accountRepo.listUngroupedByPlatformCalls)
}
