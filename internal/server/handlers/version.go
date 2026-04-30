package handlers

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/conf"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/version").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("", http.MethodGet).
				Handle(getNowVersion),
		)
}

func getNowVersion(c *gin.Context) {
	resp.Success(c, conf.Version)
}
