package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireDestructiveConfirm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		header    string
		query     string
		wantAllow bool
	}{
		{name: "header confirmation", header: "clear-logs", wantAllow: true},
		{name: "query confirmation", query: "clear-logs", wantAllow: true},
		{name: "missing confirmation", wantAllow: false},
		{name: "wrong confirmation", header: "delete-model", wantAllow: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodDelete, "/danger?confirm="+tt.query, nil)
			if tt.header != "" {
				req.Header.Set(destructiveConfirmHeader, tt.header)
			}
			c.Request = req

			gotAllow := requireDestructiveConfirm(c, "clear-logs")
			if gotAllow != tt.wantAllow {
				t.Fatalf("expected allow=%v, got %v", tt.wantAllow, gotAllow)
			}
			if !tt.wantAllow && w.Code != http.StatusPreconditionRequired {
				t.Fatalf("expected 428 for rejected request, got %d", w.Code)
			}
		})
	}
}

func TestUpdateHandlersRequireConfirmWhenDeletingChildren(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		body    string
		handler gin.HandlerFunc
	}{
		{
			name:    "channel key deletion in update",
			body:    `{"id":1,"keys_to_delete":[2]}`,
			handler: updateChannel,
		},
		{
			name:    "group item deletion in update",
			body:    `{"id":1,"items_to_delete":[2]}`,
			handler: updateGroup,
		},
		{
			name:    "channel model deletion in upstream apply",
			body:    `{"id":1,"remove_models":["gpt-4o"]}`,
			handler: applyChannelUpstreamUpdates,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			tt.handler(c)

			if w.Code != http.StatusPreconditionRequired {
				t.Fatalf("expected 428 when deleting children without confirmation, got %d", w.Code)
			}
		})
	}
}
