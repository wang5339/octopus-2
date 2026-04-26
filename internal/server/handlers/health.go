package handlers

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	// 健康检查端点不需要认证
	router.NewGroupRouter("").
		AddRoute(
			router.NewRoute("/health", http.MethodGet).
				Handle(healthCheck),
		).
		AddRoute(
			router.NewRoute("/readiness", http.MethodGet).
				Handle(readinessCheck),
		)
}

// healthCheck 基本健康检查，只要服务启动就返回 200
func healthCheck(c *gin.Context) {
	resp.Success(c, gin.H{
		"status": "ok",
	})
}

// readinessCheck 就绪检查，检查数据库连接等关键依赖
func readinessCheck(c *gin.Context) {
	// 检查数据库连接
	sqlDB, err := db.GetDB().DB()
	if err != nil {
		resp.Error(c, http.StatusServiceUnavailable, "database connection error")
		return
	}

	if err := sqlDB.Ping(); err != nil {
		resp.Error(c, http.StatusServiceUnavailable, "database ping failed")
		return
	}

	resp.Success(c, gin.H{
		"status":   "ready",
		"database": "ok",
	})
}
