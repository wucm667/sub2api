package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGroupHandlerSimpleModeSanitizesCreateAndUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newStubAdminService()
	handler := NewGroupHandlerWithConfig(svc, nil, nil, &config.Config{RunMode: config.RunModeSimple})
	router := gin.New()
	router.POST("/api/v1/admin/groups", handler.Create)
	router.PUT("/api/v1/admin/groups/:id", handler.Update)

	createBody := []byte(`{
		"name":"simple",
		"platform":"anthropic",
		"rate_multiplier":7,
		"is_exclusive":true,
		"subscription_type":"subscription",
		"daily_limit_usd":10,
		"weekly_limit_usd":20,
		"monthly_limit_usd":30,
		"allow_image_generation":true,
		"mcp_xml_inject":false,
		"rpm_limit":99
	}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/groups", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)

	require.Equal(t, http.StatusOK, createResp.Code)
	require.Len(t, svc.createdGroups, 1)
	created := svc.createdGroups[0]
	require.Equal(t, 1.0, created.RateMultiplier)
	require.False(t, created.IsExclusive)
	require.Equal(t, service.SubscriptionTypeStandard, created.SubscriptionType)
	require.Nil(t, created.DailyLimitUSD)
	require.Nil(t, created.WeeklyLimitUSD)
	require.Nil(t, created.MonthlyLimitUSD)
	require.False(t, created.AllowImageGeneration)
	require.Nil(t, created.MCPXMLInject)
	require.Zero(t, created.RPMLimit)

	updateBody := []byte(`{
		"name":"renamed",
		"rate_multiplier":9,
		"is_exclusive":true,
		"subscription_type":"subscription",
		"daily_limit_usd":12,
		"weekly_limit_usd":22,
		"monthly_limit_usd":32,
		"allow_image_generation":true,
		"status":"inactive",
		"mcp_xml_inject":false,
		"rpm_limit":123
	}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/2", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	router.ServeHTTP(updateResp, updateReq)

	require.Equal(t, http.StatusOK, updateResp.Code)
	require.Len(t, svc.updatedGroups, 1)
	updated := svc.updatedGroups[0]
	require.NotNil(t, updated.RateMultiplier)
	require.Equal(t, 1.0, *updated.RateMultiplier)
	require.NotNil(t, updated.IsExclusive)
	require.False(t, *updated.IsExclusive)
	require.Equal(t, service.SubscriptionTypeStandard, updated.SubscriptionType)
	require.Nil(t, updated.DailyLimitUSD)
	require.Nil(t, updated.WeeklyLimitUSD)
	require.Nil(t, updated.MonthlyLimitUSD)
	require.Empty(t, updated.Status)
	require.NotNil(t, updated.AllowImageGeneration)
	require.False(t, *updated.AllowImageGeneration)
	require.Nil(t, updated.MCPXMLInject)
	require.NotNil(t, updated.RPMLimit)
	require.Zero(t, *updated.RPMLimit)
}

func TestGroupHandlerSimpleModeDeleteOnlyEmptyGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newStubAdminService()
	svc.groups = []service.Group{
		{ID: 1, Name: "empty", Status: service.StatusActive, AccountCount: 0},
		{ID: 2, Name: "bound", Status: service.StatusActive, AccountCount: 2},
	}
	handler := NewGroupHandlerWithConfig(svc, nil, nil, &config.Config{RunMode: config.RunModeSimple})
	router := gin.New()
	router.DELETE("/api/v1/admin/groups/:id", handler.Delete)

	blockedReq := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/groups/2", nil)
	blockedResp := httptest.NewRecorder()
	router.ServeHTTP(blockedResp, blockedReq)
	require.Equal(t, http.StatusBadRequest, blockedResp.Code)

	emptyReq := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/groups/1", nil)
	emptyResp := httptest.NewRecorder()
	router.ServeHTTP(emptyResp, emptyReq)
	require.Equal(t, http.StatusOK, emptyResp.Code)
}

func TestAccountHandlerKeepsGroupUpdateAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newStubAdminService()
	handler := NewAccountHandler(svc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.PUT("/api/v1/admin/accounts/:id", handler.Update)

	body := []byte(`{"name":"account","group_ids":[2]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/3", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, svc.updatedAccounts, 1)
	require.NotNil(t, svc.updatedAccounts[0].GroupIDs)
	require.Equal(t, []int64{2}, *svc.updatedAccounts[0].GroupIDs)
}

func TestGroupHandlerStandardModePreservesCommercialFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newStubAdminService()
	handler := NewGroupHandlerWithConfig(svc, nil, nil, &config.Config{RunMode: config.RunModeStandard})
	router := gin.New()
	router.POST("/api/v1/admin/groups", handler.Create)

	body := []byte(`{
		"name":"standard",
		"platform":"anthropic",
		"rate_multiplier":2.5,
		"is_exclusive":true,
		"subscription_type":"subscription",
		"daily_limit_usd":10
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, svc.createdGroups, 1)
	created := svc.createdGroups[0]
	require.Equal(t, 2.5, created.RateMultiplier)
	require.True(t, created.IsExclusive)
	require.Equal(t, service.SubscriptionTypeSubscription, created.SubscriptionType)
	require.NotNil(t, created.DailyLimitUSD)
	require.Equal(t, 10.0, *created.DailyLimitUSD)
}
