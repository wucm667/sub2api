package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type emailServiceSettingRepoStub struct {
	values map[string]string
	err    error
}

func (s *emailServiceSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *emailServiceSettingRepoStub) GetValue(context.Context, string) (string, error) {
	return "", ErrSettingNotFound
}

func (s *emailServiceSettingRepoStub) Set(context.Context, string, string) error {
	return nil
}

func (s *emailServiceSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *emailServiceSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}

func (s *emailServiceSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return nil, nil
}

func (s *emailServiceSettingRepoStub) Delete(context.Context, string) error {
	return nil
}

func TestEmailService_IsConfigured(t *testing.T) {
	t.Run("returns false when smtp host is missing", func(t *testing.T) {
		svc := NewEmailService(&emailServiceSettingRepoStub{values: map[string]string{}}, nil)

		require.False(t, svc.IsConfigured(context.Background()))
	})

	t.Run("returns true when smtp host is configured", func(t *testing.T) {
		svc := NewEmailService(&emailServiceSettingRepoStub{
			values: map[string]string{SettingKeySMTPHost: "smtp.example.com"},
		}, nil)

		require.True(t, svc.IsConfigured(context.Background()))
	})

	t.Run("returns true for non configuration errors", func(t *testing.T) {
		svc := NewEmailService(&emailServiceSettingRepoStub{err: errors.New("db down")}, nil)

		require.True(t, svc.IsConfigured(context.Background()))
	})
}
