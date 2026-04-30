package op

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
	"gorm.io/gorm"
)

var apiKeyCache = cache.New[int, model.APIKey](16)
var apiKeyIDMap = cache.New[string, int](16)

func APIKeyHash(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])
}

func APIKeyMask(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return ""
	}
	const mask = "********"
	if len(apiKey) <= 8 {
		return mask
	}
	if len(apiKey) <= 15 {
		return apiKey[:4] + mask + apiKey[len(apiKey)-4:]
	}
	return apiKey[:11] + mask + apiKey[len(apiKey)-4:]
}

func normalizeAPIKeyForStorage(key *model.APIKey) (string, error) {
	rawKey := strings.TrimSpace(key.APIKey)
	if rawKey == "" {
		return "", fmt.Errorf("API key is empty")
	}
	key.APIKeyHash = APIKeyHash(rawKey)
	key.APIKey = APIKeyMask(rawKey)
	return rawKey, nil
}

func APIKeyCreate(key *model.APIKey, ctx context.Context) error {
	rawKey, err := normalizeAPIKeyForStorage(key)
	if err != nil {
		return err
	}
	if err := db.GetDB().WithContext(ctx).Create(key).Error; err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	apiKeyCache.Set(key.ID, *key)
	apiKeyIDMap.Set(key.APIKeyHash, key.ID)
	key.APIKey = rawKey
	return nil
}

func APIKeyUpdate(key *model.APIKey, ctx context.Context) error {
	existing, ok := apiKeyCache.Get(key.ID)
	if !ok {
		return fmt.Errorf("API key not found")
	}
	key.APIKey = existing.APIKey
	key.APIKeyHash = existing.APIKeyHash
	if err := db.GetDB().WithContext(ctx).Omit("api_key", "api_key_hash").Save(key).Error; err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}
	apiKeyCache.Set(key.ID, *key)
	return nil
}

func APIKeyList(ctx context.Context) ([]model.APIKey, error) {
	keys := make([]model.APIKey, 0, apiKeyCache.Len())
	for _, apiKey := range apiKeyCache.GetAll() {
		keys = append(keys, apiKey)
	}
	return keys, nil
}

func APIKeyGet(id int, ctx context.Context) (model.APIKey, error) {
	apiKey, ok := apiKeyCache.Get(id)
	if !ok {
		return model.APIKey{}, fmt.Errorf("API key not found")
	}
	return apiKey, nil
}

func APIKeyGetByAPIKey(apiKey string, ctx context.Context) (model.APIKey, error) {
	id, ok := apiKeyIDMap.Get(APIKeyHash(apiKey))
	if !ok {
		return model.APIKey{}, fmt.Errorf("API key not found")
	}
	return APIKeyGet(id, ctx)
}

func APIKeyDelete(id int, ctx context.Context) error {
	existing, ok := apiKeyCache.Get(id)
	if !ok {
		return fmt.Errorf("API key not found")
	}
	if err := StatsAPIKeyDel(id); err != nil {
		return fmt.Errorf("failed to delete stats API key: %v", err)
	}
	result := db.GetDB().WithContext(ctx).Delete(&model.APIKey{ID: id})
	if result.RowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}
	if result.Error != nil {
		return fmt.Errorf("failed to delete API key: %w", result.Error)
	}
	apiKeyCache.Del(existing.ID)
	apiKeyIDMap.Del(existing.APIKeyHash)
	return nil
}

func apiKeyRefreshCache(ctx context.Context) error {
	apiKeys := []model.APIKey{}
	conn := db.GetDB().WithContext(ctx)
	if err := conn.Find(&apiKeys).Error; err != nil {
		return err
	}
	apiKeyCache.Clear()
	apiKeyIDMap.Clear()
	for _, apiKey := range apiKeys {
		if err := migrateLegacyAPIKeyIfNeeded(conn, &apiKey); err != nil {
			return err
		}
		apiKeyCache.Set(apiKey.ID, apiKey)
		apiKeyIDMap.Set(apiKey.APIKeyHash, apiKey.ID)
	}
	return nil
}

func migrateLegacyAPIKeyIfNeeded(conn *gorm.DB, apiKey *model.APIKey) error {
	if strings.TrimSpace(apiKey.APIKeyHash) != "" {
		masked := APIKeyMask(apiKey.APIKey)
		if apiKey.APIKey != "" && apiKey.APIKey != masked {
			apiKey.APIKey = masked
			return conn.Model(&model.APIKey{}).
				Where("id = ?", apiKey.ID).
				Update("api_key", apiKey.APIKey).Error
		}
		apiKey.APIKey = masked
		return nil
	}

	rawKey := strings.TrimSpace(apiKey.APIKey)
	if rawKey == "" || strings.Contains(rawKey, "*") {
		return fmt.Errorf("API key %d cannot be migrated because plaintext key is unavailable", apiKey.ID)
	}

	apiKey.APIKeyHash = APIKeyHash(rawKey)
	apiKey.APIKey = APIKeyMask(rawKey)
	return conn.Model(&model.APIKey{}).
		Where("id = ?", apiKey.ID).
		Updates(map[string]any{
			"api_key":      apiKey.APIKey,
			"api_key_hash": apiKey.APIKeyHash,
		}).Error
}
