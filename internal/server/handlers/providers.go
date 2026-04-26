package handlers

import (
	"io"
	"net/http"

	"github.com/bestruirui/octopus/internal/assets"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/gin-gonic/gin"
)

// GetProviders returns the list of providers from the embedded providers.json.
// Note: 使用本地嵌入版本，因为本分支对 channel_type 进行了重新编号
// (OpenAIImageGeneration=6, GithubCopilot=7, Antigravity=8, Zen=9),
// 与上游 erguotou520/octopus 的 providers.json 不兼容。
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
		AddRoute(
			router.NewRoute("", http.MethodGet).Handle(GetProviders),
		)
}
