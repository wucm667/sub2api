package service

import (
	"context"
	"fmt"
	"net/http"
)

func (s *GatewayService) applyAnthropicOAuthRequestPatch(ctx context.Context, request *http.Request, account *Account, endpoint string) error {
	if s == nil || s.pluginManager == nil || account == nil || account.Platform != PlatformAnthropic || account.Type != AccountTypeOAuth {
		return nil
	}
	if request == nil || request.URL == nil {
		return fmt.Errorf("Anthropic OAuth 请求补丁目标无效")
	}
	return s.pluginManager.PatchAnthropicOAuth(ctx, request, account, endpoint)
}
