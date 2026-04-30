package op

import (
	"context"
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
)

func TestDBImportIncrementalRejectsConflictingChannelIdentity(t *testing.T) {
	initOpCacheTestDB(t)

	if err := db.GetDB().Create(&model.Channel{
		ID:   1,
		Name: "current-channel",
	}).Error; err != nil {
		t.Fatalf("create existing channel returned error: %v", err)
	}

	dump := &model.DBDump{
		Version: dbDumpVersion,
		Channels: []model.Channel{
			{ID: 1, Name: "imported-channel"},
		},
		ChannelKeys: []model.ChannelKey{
			{ID: 100, ChannelID: 1, ChannelKey: "imported-key"},
		},
	}

	_, err := DBImportIncremental(context.Background(), dump)
	if err == nil {
		t.Fatal("DBImportIncremental returned nil error, want conflicting channel identity rejected")
	}
	if !strings.Contains(err.Error(), "channel id 1 already belongs to") {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int64
	if err := db.GetDB().Model(&model.ChannelKey{}).Where("id = ?", 100).Count(&count).Error; err != nil {
		t.Fatalf("count channel key returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("conflicting import inserted %d channel key rows, want 0", count)
	}
}

func TestDBImportIncrementalAllowsMatchingChannelIdentity(t *testing.T) {
	initOpCacheTestDB(t)

	if err := db.GetDB().Create(&model.Channel{
		ID:   1,
		Name: "same-channel",
	}).Error; err != nil {
		t.Fatalf("create existing channel returned error: %v", err)
	}

	dump := &model.DBDump{
		Version: dbDumpVersion,
		Channels: []model.Channel{
			{ID: 1, Name: "same-channel"},
		},
		ChannelKeys: []model.ChannelKey{
			{ID: 100, ChannelID: 1, ChannelKey: "imported-key"},
		},
	}

	if _, err := DBImportIncremental(context.Background(), dump); err != nil {
		t.Fatalf("DBImportIncremental returned error: %v", err)
	}

	var key model.ChannelKey
	if err := db.GetDB().First(&key, 100).Error; err != nil {
		t.Fatalf("expected imported channel key, got error: %v", err)
	}
	if key.ChannelID != 1 || key.ChannelKey != "imported-key" {
		t.Fatalf("imported key = %#v, want channel_id=1 channel_key=imported-key", key)
	}
}

func TestDBImportIncrementalRejectsConflictingGroupIdentity(t *testing.T) {
	initOpCacheTestDB(t)

	if err := db.GetDB().Create(&model.Group{
		ID:   1,
		Name: "current-group",
		Mode: model.GroupModeRoundRobin,
	}).Error; err != nil {
		t.Fatalf("create existing group returned error: %v", err)
	}

	dump := &model.DBDump{
		Version: dbDumpVersion,
		Groups: []model.Group{
			{ID: 1, Name: "imported-group", Mode: model.GroupModeRoundRobin},
		},
		GroupItems: []model.GroupItem{
			{ID: 100, GroupID: 1, ChannelID: 1, ModelName: "gpt-4o"},
		},
	}

	_, err := DBImportIncremental(context.Background(), dump)
	if err == nil {
		t.Fatal("DBImportIncremental returned nil error, want conflicting group identity rejected")
	}
	if !strings.Contains(err.Error(), "group id 1 already belongs to") {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int64
	if err := db.GetDB().Model(&model.GroupItem{}).Where("id = ?", 100).Count(&count).Error; err != nil {
		t.Fatalf("count group item returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("conflicting import inserted %d group item rows, want 0", count)
	}
}

func TestDBImportIncrementalRejectsConflictingAPIKeyIdentity(t *testing.T) {
	initOpCacheTestDB(t)

	if err := db.GetDB().Create(&model.APIKey{
		ID:         1,
		Name:       "current-key",
		APIKey:     "********",
		APIKeyHash: "hash-current",
	}).Error; err != nil {
		t.Fatalf("create existing api key returned error: %v", err)
	}

	dump := &model.DBDump{
		Version: dbDumpVersion,
		APIKeys: []model.APIKey{
			{ID: 1, Name: "imported-key", APIKey: "********", APIKeyHash: "hash-imported"},
		},
	}

	_, err := DBImportIncremental(context.Background(), dump)
	if err == nil {
		t.Fatal("DBImportIncremental returned nil error, want conflicting api key identity rejected")
	}
	if !strings.Contains(err.Error(), "api key id 1 already has a different hash") {
		t.Fatalf("unexpected error: %v", err)
	}
}
