package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	pluginv1 "github.com/Wei-Shaw/sub2api/pkg/pluginapi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type patchTestClient struct {
	pluginv1.TransportPluginClient
	patch func(context.Context, *pluginv1.PatchRequest) (*pluginv1.PatchResponse, error)
}

func (c *patchTestClient) Patch(ctx context.Context, request *pluginv1.PatchRequest, _ ...grpc.CallOption) (*pluginv1.PatchResponse, error) {
	return c.patch(ctx, request)
}

func newPatchTestRuntime(client pluginv1.TransportPluginClient) *pluginRuntime {
	return &pluginRuntime{api: client}
}

func TestPluginManifestAcceptsAnthropicOAuthPatchAndScopesAccounts(t *testing.T) {
	manifest := testPluginManifest(nil)
	manifest.Capabilities = []PluginCapability{{
		ID:          PluginCapabilityAnthropicOAuthPatch,
		Platform:    PlatformAnthropic,
		AccountType: AccountTypeOAuth,
	}}

	require.NoError(t, manifest.Validate())
	scope := pluginAccountScopeFromManifest(manifest)
	assert.True(t, scope.Contains(PlatformAnthropic, AccountTypeOAuth))
	assert.False(t, scope.Contains(PlatformAnthropic, AccountTypeAPIKey))
	assert.False(t, scope.Contains(PlatformOpenAI, AccountTypeOAuth))
}

func TestPluginCapabilityRolloutsAreIndependentAndLegacyRouteConstructionWorks(t *testing.T) {
	manager := &PluginManager{}
	manager.route.Store(&pluginRoute{
		rollouts: map[string]int{
			PluginCapabilityOpenAIOAuthOutbound: 100,
			PluginCapabilityAnthropicOAuthPatch: 0,
		},
		rolloutPercent: 100,
	})

	assert.True(t, manager.ShouldRouteOpenAIOAuth(&Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth}))
	assert.False(t, manager.shouldRouteCapability(PluginCapabilityAnthropicOAuthPatch, &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeOAuth}))

	legacy := &PluginManager{}
	legacy.route.Store(&pluginRoute{rolloutPercent: 100})
	assert.True(t, legacy.ShouldRouteOpenAIOAuth(&Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth}))
}

func TestPluginRuntimePatchCompatibilityValidationAndLimits(t *testing.T) {
	var received *pluginv1.PatchRequest
	client := &patchTestClient{patch: func(_ context.Context, request *pluginv1.PatchRequest) (*pluginv1.PatchResponse, error) {
		received = request
		return &pluginv1.PatchResponse{
			Applied: true,
			Body:    []byte(`{"patched":true}`),
			Headers: map[string]*pluginv1.HeaderValues{
				"anthropic-beta": &pluginv1.HeaderValues{Values: []string{"prompt-caching-2024-07-31"}},
			},
		}, nil
	}}
	runtime := newPatchTestRuntime(client)
	request, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", strings.NewReader(`{"input":true}`))
	require.NoError(t, err)
	request.Host = "api.anthropic.com"
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("X-Api-Key", "secret")
	request.Header.Set("Proxy-Authorization", "secret")
	request.Header.Set("Host", "api.anthropic.com")
	request.Header.Set("Content-Length", "999")
	request.Header.Set("Content-Type", "application/json")

	require.NoError(t, runtime.patch(context.Background(), request, &Account{ID: 7, Platform: PlatformAnthropic, Type: AccountTypeOAuth}, "messages"))
	require.NotNil(t, received)
	assert.Equal(t, "messages", received.Endpoint)
	assert.Equal(t, int64(7), received.AccountId)
	assert.NotContains(t, received.Headers, "Authorization")
	assert.NotContains(t, received.Headers, "X-Api-Key")
	assert.NotContains(t, received.Headers, "Proxy-Authorization")
	assert.NotContains(t, received.Headers, "Host")
	assert.NotContains(t, received.Headers, "Content-Length")
	assert.Equal(t, "api.anthropic.com", received.Host)
	assert.Equal(t, "https://api.anthropic.com/v1/messages", received.Url)
	body, readErr := io.ReadAll(request.Body)
	require.NoError(t, readErr)
	assert.JSONEq(t, `{"patched":true}`, string(body))
	assert.Equal(t, "prompt-caching-2024-07-31", request.Header.Get("Anthropic-Beta"))
	assert.Equal(t, int64(len(`{"patched":true}`)), request.ContentLength)

	for name, response := range map[string]*pluginv1.PatchResponse{
		"credential":  &pluginv1.PatchResponse{Applied: true, Body: []byte("x"), Headers: map[string]*pluginv1.HeaderValues{"Authorization": &pluginv1.HeaderValues{Values: []string{"x"}}}},
		"destination": &pluginv1.PatchResponse{Applied: true, Body: []byte("x"), Headers: map[string]*pluginv1.HeaderValues{"Host": &pluginv1.HeaderValues{Values: []string{"evil"}}}},
		"proxy":       &pluginv1.PatchResponse{Applied: true, Body: []byte("x"), Headers: map[string]*pluginv1.HeaderValues{"Proxy": &pluginv1.HeaderValues{Values: []string{"evil"}}}},
	} {
		client.patch = func(context.Context, *pluginv1.PatchRequest) (*pluginv1.PatchResponse, error) { return response, nil }
		err := runtime.patch(context.Background(), request, &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth}, "messages")
		assert.ErrorContains(t, err, "禁止修改请求头", name)
	}

	client.patch = func(context.Context, *pluginv1.PatchRequest) (*pluginv1.PatchResponse, error) {
		return &pluginv1.PatchResponse{Applied: true, Body: make([]byte, pluginPatchMaxBodyBytes+1)}, nil
	}
	assert.ErrorContains(t, runtime.patch(context.Background(), request, &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth}, "messages"), "结果超过大小限制")

	largeRequest, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", io.LimitReader(strings.NewReader(strings.Repeat("x", pluginPatchMaxBodyBytes+1)), pluginPatchMaxBodyBytes+1))
	require.NoError(t, err)
	assert.ErrorContains(t, runtime.patch(context.Background(), largeRequest, &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth}, "messages"), "请求体超过大小限制")

	client.patch = func(ctx context.Context, _ *pluginv1.PatchRequest) (*pluginv1.PatchResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	started := time.Now()
	err = runtime.patch(context.Background(), request, &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth}, "messages")
	assert.Error(t, err)
	assert.Less(t, time.Since(started), 3*time.Second)
}

func TestPluginRuntimePatchTreatsUnimplementedAsLegacyCompatibility(t *testing.T) {
	client := &patchTestClient{patch: func(context.Context, *pluginv1.PatchRequest) (*pluginv1.PatchResponse, error) {
		return nil, status.Error(codes.Unimplemented, "unknown method")
	}}
	runtime := newPatchTestRuntime(client)
	request, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", strings.NewReader("body"))
	require.NoError(t, err)

	require.NoError(t, runtime.patch(context.Background(), request, &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth}, "messages"))
	body, readErr := io.ReadAll(request.Body)
	require.NoError(t, readErr)
	assert.Equal(t, "body", string(body))
}

func TestAnthropicOAuthPatchHookInvokedForMessagesAndCountTokensOnly(t *testing.T) {
	var mu sync.Mutex
	var endpoints []string
	client := &patchTestClient{patch: func(_ context.Context, request *pluginv1.PatchRequest) (*pluginv1.PatchResponse, error) {
		mu.Lock()
		endpoints = append(endpoints, request.Endpoint)
		mu.Unlock()
		return &pluginv1.PatchResponse{Applied: true, Body: append([]byte(nil), request.Body...)}, nil
	}}
	manager := &PluginManager{}
	manager.route.Store(&pluginRoute{runtime: newPatchTestRuntime(client), rollouts: map[string]int{PluginCapabilityAnthropicOAuthPatch: 100}})
	service := &GatewayService{pluginManager: manager}
	account := &Account{ID: 42, Platform: PlatformAnthropic, Type: AccountTypeOAuth}

	messagesRequest, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", strings.NewReader(`{}`))
	require.NoError(t, err)
	countTokensRequest, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages/count_tokens", strings.NewReader(`{}`))
	require.NoError(t, err)
	require.NoError(t, service.applyAnthropicOAuthRequestPatch(context.Background(), messagesRequest, account, "messages"))
	require.NoError(t, service.applyAnthropicOAuthRequestPatch(context.Background(), countTokensRequest, account, "count_tokens"))

	assert.Equal(t, []string{"messages", "count_tokens"}, endpoints)
}

func TestAnthropicPatchAndRoutingLeaveAPIKeyAndNonAnthropicPathsUntouched(t *testing.T) {
	called := false
	client := &patchTestClient{patch: func(context.Context, *pluginv1.PatchRequest) (*pluginv1.PatchResponse, error) {
		called = true
		return nil, errors.New("must not be called")
	}}
	manager := &PluginManager{}
	manager.route.Store(&pluginRoute{runtime: newPatchTestRuntime(client), rollouts: map[string]int{PluginCapabilityAnthropicOAuthPatch: 100}})
	service := &GatewayService{pluginManager: manager}
	for _, account := range []*Account{
		{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeOAuth},
		{ID: 3, Platform: PlatformGemini, Type: AccountTypeOAuth},
	} {
		request, err := http.NewRequest(http.MethodPost, "https://example.com/v1/messages", strings.NewReader("body"))
		require.NoError(t, err)
		require.NoError(t, service.applyAnthropicOAuthRequestPatch(context.Background(), request, account, "messages"))
		body, readErr := io.ReadAll(request.Body)
		require.NoError(t, readErr)
		assert.Equal(t, "body", string(body))
	}
	assert.False(t, called)
}
