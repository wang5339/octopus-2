package model

import "testing"

func TestSettingValidateRejectsOutOfRangeIntegers(t *testing.T) {
	tests := []Setting{
		{Key: SettingKeyStatsSaveInterval, Value: "0"},
		{Key: SettingKeyModelInfoUpdateInterval, Value: "0"},
		{Key: SettingKeySyncLLMInterval, Value: "9999"},
		{Key: SettingKeyRelayLogKeepPeriod, Value: "-1"},
		{Key: SettingKeyCircuitBreakerThreshold, Value: "0"},
		{Key: SettingKeyCircuitBreakerCooldown, Value: "0"},
		{Key: SettingKeyCircuitBreakerMaxCooldown, Value: "999999999"},
	}

	for _, tt := range tests {
		if err := tt.Validate(); err == nil {
			t.Fatalf("expected %s=%q to be rejected", tt.Key, tt.Value)
		}
	}
}

func TestSettingValidateAllowsDocumentedIntegerRanges(t *testing.T) {
	tests := []Setting{
		{Key: SettingKeyStatsSaveInterval, Value: "1"},
		{Key: SettingKeyModelInfoUpdateInterval, Value: "168"},
		{Key: SettingKeySyncLLMInterval, Value: "24"},
		{Key: SettingKeyRelayLogKeepPeriod, Value: "0"},
		{Key: SettingKeyRelayLogKeepPeriod, Value: "3650"},
		{Key: SettingKeyCircuitBreakerThreshold, Value: "1000"},
		{Key: SettingKeyCircuitBreakerCooldown, Value: "86400"},
		{Key: SettingKeyCircuitBreakerMaxCooldown, Value: "604800"},
	}

	for _, tt := range tests {
		if err := tt.Validate(); err != nil {
			t.Fatalf("expected %s=%q to be accepted, got %v", tt.Key, tt.Value, err)
		}
	}
}

func TestSettingValidateAcceptsSocksProxyURL(t *testing.T) {
	setting := Setting{Key: SettingKeyProxyURL, Value: "socks://127.0.0.1:1080"}
	if err := setting.Validate(); err != nil {
		t.Fatalf("expected socks proxy URL to be accepted, got %v", err)
	}
}
