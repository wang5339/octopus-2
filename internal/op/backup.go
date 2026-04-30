package op

import (
	"context"
	"fmt"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const dbDumpVersion = 1

func DBExportAll(ctx context.Context, includeLogs, includeStats bool) (*model.DBDump, error) {
	conn := db.GetDB().WithContext(ctx)

	d := &model.DBDump{
		Version:      dbDumpVersion,
		ExportedAt:   time.Now().UTC(),
		IncludeLogs:  includeLogs,
		IncludeStats: includeStats,
	}

	if err := conn.Find(&d.Channels).Error; err != nil {
		return nil, fmt.Errorf("export channels: %w", err)
	}
	if err := conn.Find(&d.ChannelKeys).Error; err != nil {
		return nil, fmt.Errorf("export channel_keys: %w", err)
	}
	if err := conn.Find(&d.Groups).Error; err != nil {
		return nil, fmt.Errorf("export groups: %w", err)
	}
	if err := conn.Find(&d.GroupItems).Error; err != nil {
		return nil, fmt.Errorf("export group_items: %w", err)
	}
	if err := conn.Find(&d.LLMInfos).Error; err != nil {
		return nil, fmt.Errorf("export llm_infos: %w", err)
	}
	if err := conn.Find(&d.APIKeys).Error; err != nil {
		return nil, fmt.Errorf("export api_keys: %w", err)
	}
	if err := conn.Find(&d.Settings).Error; err != nil {
		return nil, fmt.Errorf("export settings: %w", err)
	}

	if includeStats {
		if err := conn.Find(&d.StatsTotal).Error; err != nil {
			return nil, fmt.Errorf("export stats_total: %w", err)
		}
		if err := conn.Find(&d.StatsDaily).Error; err != nil {
			return nil, fmt.Errorf("export stats_daily: %w", err)
		}
		if err := conn.Find(&d.StatsHourly).Error; err != nil {
			return nil, fmt.Errorf("export stats_hourly: %w", err)
		}
		if err := conn.Find(&d.StatsChannel).Error; err != nil {
			return nil, fmt.Errorf("export stats_channel: %w", err)
		}
		if err := conn.Find(&d.StatsAPIKey).Error; err != nil {
			return nil, fmt.Errorf("export stats_api_key: %w", err)
		}
		if err := conn.Find(&d.StatsGroup).Error; err != nil {
			return nil, fmt.Errorf("export stats_group: %w", err)
		}
	}

	if includeLogs {
		if err := conn.Find(&d.RelayLogs).Error; err != nil {
			return nil, fmt.Errorf("export relay_logs: %w", err)
		}
	}

	return d, nil
}

func DBImportIncremental(ctx context.Context, dump *model.DBDump) (*model.DBImportResult, error) {
	if dump == nil {
		return nil, fmt.Errorf("empty dump")
	}

	if dump.Version != 0 && dump.Version != dbDumpVersion {
		return nil, fmt.Errorf("unsupported dump version: %d", dump.Version)
	}

	conn := db.GetDB().WithContext(ctx)
	res := &model.DBImportResult{RowsAffected: map[string]int64{}}

	err := conn.Transaction(func(tx *gorm.DB) error {
		if err := validateDBImportIdentityConflicts(tx, dump); err != nil {
			return err
		}

		// base tables
		if n, err := createDoNothing(tx, dump.Channels); err != nil {
			return fmt.Errorf("import channels: %w", err)
		} else {
			res.RowsAffected["channels"] = n
		}
		if n, err := createDoNothing(tx, dump.ChannelKeys); err != nil {
			return fmt.Errorf("import channel_keys: %w", err)
		} else {
			res.RowsAffected["channel_keys"] = n
		}
		if n, err := createDoNothing(tx, dump.Groups); err != nil {
			return fmt.Errorf("import groups: %w", err)
		} else {
			res.RowsAffected["groups"] = n
		}
		if n, err := createDoNothing(tx, dump.GroupItems); err != nil {
			return fmt.Errorf("import group_items: %w", err)
		} else {
			res.RowsAffected["group_items"] = n
		}
		if n, err := createUpsertAll(tx, dump.LLMInfos, []clause.Column{{Name: "name"}}); err != nil {
			return fmt.Errorf("import llm_infos: %w", err)
		} else {
			res.RowsAffected["llm_infos"] = n
		}
		if n, err := createDoNothing(tx, dump.APIKeys); err != nil {
			return fmt.Errorf("import api_keys: %w", err)
		} else {
			res.RowsAffected["api_keys"] = n
		}
		if n, err := createUpsertSettings(tx, dump.Settings); err != nil {
			return fmt.Errorf("import settings: %w", err)
		} else {
			res.RowsAffected["settings"] = n
		}

		if dump.IncludeStats {
			if n, err := createUpsertAll(tx, dump.StatsTotal, []clause.Column{{Name: "id"}}); err != nil {
				return fmt.Errorf("import stats_total: %w", err)
			} else {
				res.RowsAffected["stats_total"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsDaily, []clause.Column{{Name: "date"}}); err != nil {
				return fmt.Errorf("import stats_daily: %w", err)
			} else {
				res.RowsAffected["stats_daily"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsHourly, []clause.Column{{Name: "hour"}}); err != nil {
				return fmt.Errorf("import stats_hourly: %w", err)
			} else {
				res.RowsAffected["stats_hourly"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsChannel, []clause.Column{{Name: "channel_id"}}); err != nil {
				return fmt.Errorf("import stats_channel: %w", err)
			} else {
				res.RowsAffected["stats_channel"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsAPIKey, []clause.Column{{Name: "api_key_id"}}); err != nil {
				return fmt.Errorf("import stats_api_key: %w", err)
			} else {
				res.RowsAffected["stats_api_key"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsGroup, []clause.Column{{Name: "group_name"}}); err != nil {
				return fmt.Errorf("import stats_group: %w", err)
			} else {
				res.RowsAffected["stats_group"] = n
			}
		}

		if dump.IncludeLogs {
			if n, err := createDoNothing(tx, dump.RelayLogs); err != nil {
				return fmt.Errorf("import relay_logs: %w", err)
			} else {
				res.RowsAffected["relay_logs"] = n
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func validateDBImportIdentityConflicts(tx *gorm.DB, dump *model.DBDump) error {
	if err := validateChannelImportConflicts(tx, dump.Channels); err != nil {
		return err
	}
	if err := validateGroupImportConflicts(tx, dump.Groups); err != nil {
		return err
	}
	if err := validateAPIKeyImportConflicts(tx, dump.APIKeys); err != nil {
		return err
	}
	return nil
}

func validateChannelImportConflicts(tx *gorm.DB, channels []model.Channel) error {
	byID := make(map[int]model.Channel, len(channels))
	byName := make(map[string]model.Channel, len(channels))
	ids := make([]int, 0, len(channels))
	names := make([]string, 0, len(channels))
	for _, channel := range channels {
		if channel.ID != 0 {
			if prev, ok := byID[channel.ID]; ok && prev.Name != channel.Name {
				return fmt.Errorf("import channels: duplicate channel id %d maps to both %q and %q", channel.ID, prev.Name, channel.Name)
			}
			if _, ok := byID[channel.ID]; !ok {
				ids = append(ids, channel.ID)
			}
			byID[channel.ID] = channel
		}
		if channel.Name != "" {
			if prev, ok := byName[channel.Name]; ok && prev.ID != channel.ID {
				return fmt.Errorf("import channels: duplicate channel name %q maps to both id %d and id %d", channel.Name, prev.ID, channel.ID)
			}
			if _, ok := byName[channel.Name]; !ok {
				names = append(names, channel.Name)
			}
			byName[channel.Name] = channel
		}
	}
	if len(ids) == 0 && len(names) == 0 {
		return nil
	}

	var existing []model.Channel
	query := tx.Model(&model.Channel{})
	switch {
	case len(ids) > 0 && len(names) > 0:
		query = query.Where("id IN ? OR name IN ?", ids, names)
	case len(ids) > 0:
		query = query.Where("id IN ?", ids)
	default:
		query = query.Where("name IN ?", names)
	}
	if err := query.Find(&existing).Error; err != nil {
		return err
	}

	for _, current := range existing {
		if incoming, ok := byID[current.ID]; ok && incoming.Name != "" && current.Name != incoming.Name {
			return fmt.Errorf("import channels: channel id %d already belongs to %q, dump has %q", current.ID, current.Name, incoming.Name)
		}
		if incoming, ok := byName[current.Name]; ok && incoming.ID != 0 && current.ID != incoming.ID {
			return fmt.Errorf("import channels: channel name %q already belongs to id %d, dump has id %d", current.Name, current.ID, incoming.ID)
		}
	}
	return nil
}

func validateGroupImportConflicts(tx *gorm.DB, groups []model.Group) error {
	byID := make(map[int]model.Group, len(groups))
	byName := make(map[string]model.Group, len(groups))
	ids := make([]int, 0, len(groups))
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		if group.ID != 0 {
			if prev, ok := byID[group.ID]; ok && prev.Name != group.Name {
				return fmt.Errorf("import groups: duplicate group id %d maps to both %q and %q", group.ID, prev.Name, group.Name)
			}
			if _, ok := byID[group.ID]; !ok {
				ids = append(ids, group.ID)
			}
			byID[group.ID] = group
		}
		if group.Name != "" {
			if prev, ok := byName[group.Name]; ok && prev.ID != group.ID {
				return fmt.Errorf("import groups: duplicate group name %q maps to both id %d and id %d", group.Name, prev.ID, group.ID)
			}
			if _, ok := byName[group.Name]; !ok {
				names = append(names, group.Name)
			}
			byName[group.Name] = group
		}
	}
	if len(ids) == 0 && len(names) == 0 {
		return nil
	}

	var existing []model.Group
	query := tx.Model(&model.Group{})
	switch {
	case len(ids) > 0 && len(names) > 0:
		query = query.Where("id IN ? OR name IN ?", ids, names)
	case len(ids) > 0:
		query = query.Where("id IN ?", ids)
	default:
		query = query.Where("name IN ?", names)
	}
	if err := query.Find(&existing).Error; err != nil {
		return err
	}

	for _, current := range existing {
		if incoming, ok := byID[current.ID]; ok && incoming.Name != "" && current.Name != incoming.Name {
			return fmt.Errorf("import groups: group id %d already belongs to %q, dump has %q", current.ID, current.Name, incoming.Name)
		}
		if incoming, ok := byName[current.Name]; ok && incoming.ID != 0 && current.ID != incoming.ID {
			return fmt.Errorf("import groups: group name %q already belongs to id %d, dump has id %d", current.Name, current.ID, incoming.ID)
		}
	}
	return nil
}

func validateAPIKeyImportConflicts(tx *gorm.DB, apiKeys []model.APIKey) error {
	byID := make(map[int]model.APIKey, len(apiKeys))
	byHash := make(map[string]model.APIKey, len(apiKeys))
	ids := make([]int, 0, len(apiKeys))
	hashes := make([]string, 0, len(apiKeys))
	for _, apiKey := range apiKeys {
		if apiKey.ID != 0 {
			if prev, ok := byID[apiKey.ID]; ok && prev.APIKeyHash != apiKey.APIKeyHash {
				return fmt.Errorf("import api_keys: duplicate api key id %d has different hashes", apiKey.ID)
			}
			if _, ok := byID[apiKey.ID]; !ok {
				ids = append(ids, apiKey.ID)
			}
			byID[apiKey.ID] = apiKey
		}
		if apiKey.APIKeyHash != "" {
			if prev, ok := byHash[apiKey.APIKeyHash]; ok && prev.ID != apiKey.ID {
				return fmt.Errorf("import api_keys: duplicate api key hash maps to both id %d and id %d", prev.ID, apiKey.ID)
			}
			if _, ok := byHash[apiKey.APIKeyHash]; !ok {
				hashes = append(hashes, apiKey.APIKeyHash)
			}
			byHash[apiKey.APIKeyHash] = apiKey
		}
	}
	if len(ids) == 0 && len(hashes) == 0 {
		return nil
	}

	var existing []model.APIKey
	query := tx.Model(&model.APIKey{})
	switch {
	case len(ids) > 0 && len(hashes) > 0:
		query = query.Where("id IN ? OR api_key_hash IN ?", ids, hashes)
	case len(ids) > 0:
		query = query.Where("id IN ?", ids)
	default:
		query = query.Where("api_key_hash IN ?", hashes)
	}
	if err := query.Find(&existing).Error; err != nil {
		return err
	}

	for _, current := range existing {
		if incoming, ok := byID[current.ID]; ok && incoming.APIKeyHash != "" && current.APIKeyHash != incoming.APIKeyHash {
			return fmt.Errorf("import api_keys: api key id %d already has a different hash", current.ID)
		}
		if incoming, ok := byHash[current.APIKeyHash]; ok && incoming.ID != 0 && current.ID != incoming.ID {
			return fmt.Errorf("import api_keys: api key hash already belongs to id %d, dump has id %d", current.ID, incoming.ID)
		}
	}
	return nil
}

func createDoNothing[T any](tx *gorm.DB, rows []T) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
	return result.RowsAffected, result.Error
}

func createUpsertAll[T any](tx *gorm.DB, rows []T, columns []clause.Column) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   columns,
		UpdateAll: true,
	}).Create(&rows)
	return result.RowsAffected, result.Error
}

func createUpsertSettings(tx *gorm.DB, rows []model.Setting) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&rows)
	return result.RowsAffected, result.Error
}
