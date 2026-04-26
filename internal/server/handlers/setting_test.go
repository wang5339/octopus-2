package handlers

import (
	"testing"

	"github.com/bestruirui/octopus/internal/model"
)

func TestDecodeDBDumpWithStatsGroup(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "direct dump",
			body: `{"version":1,"include_stats":true,"stats_group":[{"group_name":"gpt-4","request_success":2}]}`,
		},
		{
			name: "wrapped api response dump",
			body: `{"code":0,"message":"success","data":{"version":1,"include_stats":true,"stats_group":[{"group_name":"gpt-4","request_success":2}]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dump model.DBDump
			if err := decodeDBDump([]byte(tt.body), &dump); err != nil {
				t.Fatalf("decodeDBDump returned error: %v", err)
			}
			if len(dump.StatsGroup) != 1 {
				t.Fatalf("expected 1 stats_group row, got %d", len(dump.StatsGroup))
			}
			if dump.StatsGroup[0].GroupName != "gpt-4" {
				t.Fatalf("unexpected group_name: %q", dump.StatsGroup[0].GroupName)
			}
			if dump.StatsGroup[0].RequestSuccess != 2 {
				t.Fatalf("unexpected request_success: %d", dump.StatsGroup[0].RequestSuccess)
			}
		})
	}
}
