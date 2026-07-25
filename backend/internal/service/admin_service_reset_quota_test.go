//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type resetAccountQuotaRepoStub struct {
	mockAccountRepoForGemini
	account             *Account
	getByIDErr          error
	resetQuotaErr       error
	clearRateLimitErr   error
	resetQuotaCalls     int
	clearRateLimitCalls int
	callOrder           []string
}

func (r *resetAccountQuotaRepoStub) GetByID(context.Context, int64) (*Account, error) {
	return r.account, r.getByIDErr
}

func (r *resetAccountQuotaRepoStub) ResetQuotaUsed(context.Context, int64) error {
	r.resetQuotaCalls++
	r.callOrder = append(r.callOrder, "reset_quota")
	return r.resetQuotaErr
}

func (r *resetAccountQuotaRepoStub) ClearRateLimit(context.Context, int64) error {
	r.clearRateLimitCalls++
	r.callOrder = append(r.callOrder, "clear_rate_limit")
	return r.clearRateLimitErr
}

func TestResetAccountQuota_ClearsSchedulerRateLimit(t *testing.T) {
	repo := &resetAccountQuotaRepoStub{account: &Account{ID: 42}}
	svc := &adminServiceImpl{accountRepo: repo}

	err := svc.ResetAccountQuota(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, 1, repo.resetQuotaCalls)
	require.Equal(t, 1, repo.clearRateLimitCalls)
	require.Equal(t, []string{"reset_quota", "clear_rate_limit"}, repo.callOrder)
}

func TestResetAccountQuota_PreservesLookupAndSparkShadowShortCircuits(t *testing.T) {
	t.Run("lookup failure", func(t *testing.T) {
		getErr := errors.New("get account failed")
		repo := &resetAccountQuotaRepoStub{getByIDErr: getErr}
		svc := &adminServiceImpl{accountRepo: repo}

		err := svc.ResetAccountQuota(context.Background(), 42)

		require.ErrorIs(t, err, getErr)
		require.Zero(t, repo.resetQuotaCalls)
		require.Zero(t, repo.clearRateLimitCalls)
	})

	t.Run("spark shadow", func(t *testing.T) {
		parentID := int64(7)
		repo := &resetAccountQuotaRepoStub{
			account: &Account{ID: 42, ParentAccountID: &parentID},
		}
		svc := &adminServiceImpl{accountRepo: repo}

		err := svc.ResetAccountQuota(context.Background(), 42)

		require.Error(t, err)
		require.Zero(t, repo.resetQuotaCalls)
		require.Zero(t, repo.clearRateLimitCalls)
	})
}

func TestResetAccountQuota_DoesNotClearRateLimitWhenQuotaResetFails(t *testing.T) {
	resetErr := errors.New("reset quota failed")
	repo := &resetAccountQuotaRepoStub{
		account:       &Account{ID: 42},
		resetQuotaErr: resetErr,
	}
	svc := &adminServiceImpl{accountRepo: repo}

	err := svc.ResetAccountQuota(context.Background(), 42)

	require.ErrorIs(t, err, resetErr)
	require.Equal(t, 1, repo.resetQuotaCalls)
	require.Zero(t, repo.clearRateLimitCalls)
}

func TestResetAccountQuota_PropagatesRateLimitClearFailure(t *testing.T) {
	clearErr := errors.New("clear rate limit failed")
	repo := &resetAccountQuotaRepoStub{
		account:           &Account{ID: 42},
		clearRateLimitErr: clearErr,
	}
	svc := &adminServiceImpl{accountRepo: repo}

	err := svc.ResetAccountQuota(context.Background(), 42)

	require.ErrorIs(t, err, clearErr)
	require.Equal(t, 1, repo.resetQuotaCalls)
	require.Equal(t, 1, repo.clearRateLimitCalls)
}
