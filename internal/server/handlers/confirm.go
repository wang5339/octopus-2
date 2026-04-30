package handlers

import (
	"net/http"
	"strings"

	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/gin-gonic/gin"
)

const destructiveConfirmHeader = "X-Octopus-Confirm"

// requireDestructiveConfirm 为不可逆/高风险操作提供后端强确认。
// 前端 confirm 只能防误点，不能作为安全边界；后端必须收到精确确认值才执行。
func requireDestructiveConfirm(c *gin.Context, expected string) bool {
	actual := strings.TrimSpace(c.GetHeader(destructiveConfirmHeader))
	if actual == "" {
		actual = strings.TrimSpace(c.Query("confirm"))
	}
	if actual == expected {
		return true
	}

	resp.Error(c, http.StatusPreconditionRequired, "confirmation required: set X-Octopus-Confirm or confirm to "+expected)
	return false
}
