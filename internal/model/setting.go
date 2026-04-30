package model

import (
	"fmt"
	"net/url"
	"strconv"
)

type SettingKey string

const (
	SettingKeyProxyURL                  SettingKey = "proxy_url"
	SettingKeyAPIBaseURL                SettingKey = "api_base_url"                 // 系统 API 地址（用于文档展示）
	SettingKeyStatsSaveInterval         SettingKey = "stats_save_interval"          // 将统计信息写入数据库的周期(分钟)
	SettingKeyModelInfoUpdateInterval   SettingKey = "model_info_update_interval"   // 模型信息更新间隔(小时)
	SettingKeySyncLLMInterval           SettingKey = "sync_llm_interval"            // LLM 同步间隔(小时)
	SettingKeyRelayLogKeepPeriod        SettingKey = "relay_log_keep_period"        // 日志保存时间范围(天)
	SettingKeyRelayLogKeepEnabled       SettingKey = "relay_log_keep_enabled"       // 是否保留历史日志
	SettingKeyCORSAllowOrigins          SettingKey = "cors_allow_origins"           // 跨域白名单(逗号分隔, 如 "example.com,example2.com"). 为空不允许跨域, "*"允许所有
	SettingKeyCircuitBreakerThreshold   SettingKey = "circuit_breaker_threshold"    // 熔断触发阈值（连续失败次数）
	SettingKeyCircuitBreakerCooldown    SettingKey = "circuit_breaker_cooldown"     // 熔断基础冷却时间（秒）
	SettingKeyCircuitBreakerMaxCooldown SettingKey = "circuit_breaker_max_cooldown" // 熔断最大冷却时间（秒），指数退避上限
)

type Setting struct {
	Key   SettingKey `json:"key" gorm:"primaryKey"`
	Value string     `json:"value" gorm:"not null"`
}

func DefaultSettings() []Setting {
	return []Setting{
		{Key: SettingKeyProxyURL, Value: ""},
		{Key: SettingKeyAPIBaseURL, Value: "http://localhost:8080"}, // 默认系统 API 地址
		{Key: SettingKeyStatsSaveInterval, Value: "10"},             // 默认10分钟保存一次统计信息
		{Key: SettingKeyCORSAllowOrigins, Value: ""},                // CORS 默认不允许跨域，设置为 "*" 才允许所有来源
		{Key: SettingKeyModelInfoUpdateInterval, Value: "24"},       // 默认24小时更新一次模型信息
		{Key: SettingKeySyncLLMInterval, Value: "24"},               // 默认24小时同步一次LLM
		{Key: SettingKeyRelayLogKeepPeriod, Value: "7"},             // 默认日志保存7天
		{Key: SettingKeyRelayLogKeepEnabled, Value: "true"},         // 默认保留历史日志
		{Key: SettingKeyCircuitBreakerThreshold, Value: "5"},        // 默认连续失败5次触发熔断
		{Key: SettingKeyCircuitBreakerCooldown, Value: "60"},        // 默认基础冷却60秒
		{Key: SettingKeyCircuitBreakerMaxCooldown, Value: "600"},    // 默认最大冷却600秒（10分钟）
	}
}

func (s *Setting) Validate() error {
	switch s.Key {
	case SettingKeyStatsSaveInterval:
		return s.validateIntRange(1, 1440) // 1 分钟 ~ 1 天
	case SettingKeyModelInfoUpdateInterval, SettingKeySyncLLMInterval:
		return s.validateIntRange(1, 168) // 1 小时 ~ 7 天
	case SettingKeyRelayLogKeepPeriod:
		return s.validateIntRange(0, 3650) // 0 表示不按天数清理，最大 10 年
	case SettingKeyCircuitBreakerThreshold:
		return s.validateIntRange(1, 1000)
	case SettingKeyCircuitBreakerCooldown:
		return s.validateIntRange(1, 86400) // 最大 1 天
	case SettingKeyCircuitBreakerMaxCooldown:
		return s.validateIntRange(1, 604800) // 最大 7 天
	case SettingKeyRelayLogKeepEnabled:
		if s.Value != "true" && s.Value != "false" {
			return fmt.Errorf("relay log keep enabled must be true or false")
		}
		return nil
	case SettingKeyProxyURL:
		if s.Value == "" {
			return nil
		}
		parsedURL, err := url.Parse(s.Value)
		if err != nil {
			return fmt.Errorf("proxy URL is invalid: %w", err)
		}
		validSchemes := map[string]bool{
			"http":   true,
			"https":  true,
			"socks":  true,
			"socks5": true,
		}
		if !validSchemes[parsedURL.Scheme] {
			return fmt.Errorf("proxy URL scheme must be http, https, socks, or socks5")
		}
		if parsedURL.Host == "" {
			return fmt.Errorf("proxy URL must have a host")
		}
		return nil
	case SettingKeyAPIBaseURL:
		if s.Value == "" {
			return nil
		}
		parsedURL, err := url.Parse(s.Value)
		if err != nil {
			return fmt.Errorf("api base URL is invalid: %w", err)
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("api base URL scheme must be http or https")
		}
		if parsedURL.Host == "" {
			return fmt.Errorf("api base URL must have a host")
		}
		return nil
	}

	return nil
}

func (s *Setting) validateIntRange(minValue, maxValue int) error {
	value, err := strconv.Atoi(s.Value)
	if err != nil {
		return fmt.Errorf("%s must be an integer", s.Key)
	}
	if value < minValue || value > maxValue {
		return fmt.Errorf("%s must be between %d and %d", s.Key, minValue, maxValue)
	}
	return nil
}
