package handlers

import (
	"io"
	"net/http"

	"github.com/bestruirui/octopus/internal/assets"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/gin-gonic/gin"
)

// GetProviders returns the list of providers from the embedded providers.json.
// Note: 使用本地嵌入版本，因为本分支显式保留 Zen=9。
// 这样可以避免历史数据库里的 Zen 渠道编号被错位解析。
func GetProviders(c *gin.Context) {
	file, err := assets.ProvidersFS.Open("providers.json")
	if err != nil {
		log.Errorf("Failed to open embedded providers.json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load providers"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Errorf("Failed to read embedded providers.json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load providers"})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

func init() {
	router.NewGroupRouter("/api/v1/providers").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("", http.MethodGet).Handle(GetProviders),
		)
}
