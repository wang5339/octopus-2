package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/auth"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type apiKeyResponse struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	APIKey          string  `json:"api_key"`
	Enabled         bool    `json:"enabled"`
	ExpireAt        int64   `json:"expire_at,omitempty"`
	MaxCost         float64 `json:"max_cost,omitempty"`
	SupportedModels string  `json:"supported_models,omitempty"`
}

func newAPIKeyResponse(apiKey model.APIKey) apiKeyResponse {
	return apiKeyResponse{
		ID:              apiKey.ID,
		Name:            apiKey.Name,
		APIKey:          apiKey.APIKey,
		Enabled:         apiKey.Enabled,
		ExpireAt:        apiKey.ExpireAt,
		MaxCost:         apiKey.MaxCost,
		SupportedModels: apiKey.SupportedModels,
	}
}

func init() {
	router.NewGroupRouter("/api/v1/apikey").
		Use(middleware.Auth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/create", http.MethodPost).
				Handle(createAPIKey),
		).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(listAPIKey),
		).
		AddRoute(
			router.NewRoute("/update", http.MethodPost).
				Handle(updateAPIKey),
		).
		AddRoute(
			router.NewRoute("/delete/:id", http.MethodDelete).
				Handle(deleteAPIKey),
		)
	router.NewGroupRouter("/api/v1/apikey").
		Use(middleware.APIKeyAuth()).
		AddRoute(
			router.NewRoute("/stats", http.MethodGet).
				Handle(getStatsAPIKeyById),
		).
		AddRoute(
			router.NewRoute("/login", http.MethodGet).
				Handle(loginAPIKey),
		)
}

func createAPIKey(c *gin.Context) {
	var req model.APIKey
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	req.APIKey = auth.GenerateAPIKey()
	if err := op.APIKeyCreate(&req, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	// 仅创建响应返回一次明文 key；后续列表/统计只返回 masked key。
	resp.Success(c, newAPIKeyResponse(req))
}

func listAPIKey(c *gin.Context) {
	apiKeys, err := op.APIKeyList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	result := make([]apiKeyResponse, 0, len(apiKeys))
	for _, apiKey := range apiKeys {
		result = append(result, newAPIKeyResponse(apiKey))
	}
	resp.Success(c, result)
}

type updateAPIKeyRequest struct {
	ID              int      `json:"id"`
	Name            *string  `json:"name,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
	ExpireAt        *int64   `json:"expire_at,omitempty"`
	MaxCost         *float64 `json:"max_cost,omitempty"`
	SupportedModels *string  `json:"supported_models,omitempty"`
}

func (req updateAPIKeyRequest) applyTo(key *model.APIKey) {
	if req.Name != nil {
		key.Name = *req.Name
	}
	if req.Enabled != nil {
		key.Enabled = *req.Enabled
	}
	if req.ExpireAt != nil {
		key.ExpireAt = *req.ExpireAt
	}
	if req.MaxCost != nil {
		key.MaxCost = *req.MaxCost
	}
	if req.SupportedModels != nil {
		key.SupportedModels = *req.SupportedModels
	}
}

func updateAPIKey(c *gin.Context) {
	var req updateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if req.ID == 0 {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidParam)
		return
	}
	apiKey, err := op.APIKeyGet(req.ID, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	req.applyTo(&apiKey)
	if err := op.APIKeyUpdate(&apiKey, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, newAPIKeyResponse(apiKey))
}

func deleteAPIKey(c *gin.Context) {
	if !requireDestructiveConfirm(c, "delete-apikey") {
		return
	}
	id := c.Param("id")
	idNum, err := strconv.Atoi(id)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidParam)
		return
	}
	if err := op.APIKeyDelete(idNum, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

func getStatsAPIKeyById(c *gin.Context) {
	id := c.GetInt("api_key_id")
	stats := op.StatsAPIKeyGet(id)
	info, err := op.APIKeyGet(id, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	models, err := op.GroupListModel(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	var modelsString string
	if info.SupportedModels == "" {
		modelsString = strings.Join(models, ", ")
	} else {
		supportedModels := lo.Map(strings.Split(info.SupportedModels, ","), func(s string, _ int) string {
			return strings.TrimSpace(s)
		})
		models = lo.Filter(models, func(m string, _ int) bool {
			return lo.Contains(supportedModels, m)
		})
		modelsString = strings.Join(models, ", ")
	}
	info.SupportedModels = modelsString
	resp.Success(c, map[string]any{
		"stats": stats,
		"info":  newAPIKeyResponse(info),
	})
}

func loginAPIKey(c *gin.Context) {
	resp.Success(c, nil)
}
