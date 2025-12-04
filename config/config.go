package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var (
	configDir  = filepath.Join(os.Getenv("HOME"), ".sentinel")
	configPath = filepath.Join(configDir, "config.json")
)

func LoadConfig() (*SentinelConfig, error) {
	os.MkdirAll(configDir, 0755)

	data, err := os.ReadFile(configPath)
	if err != nil {
		cfg := defaultConfig()
		_ = SaveConfig(cfg)
		return cfg, nil
	}

	var cfg SentinelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		cfg := defaultConfig()
		_ = SaveConfig(cfg)
		return cfg, nil
	}

	return &cfg, nil
}

func SaveConfig(cfg *SentinelConfig) error {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(configPath, data, 0644)
}

func defaultConfig() *SentinelConfig {
	return &SentinelConfig{
		CPUThreshold:  80,
		MemThreshold:  80,
		ActiveWebhook: "",
		Webhooks:      map[string]string{},
	}
}

func ConfigPath() string {
    return configPath
}