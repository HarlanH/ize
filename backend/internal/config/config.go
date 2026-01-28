package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	AlgoliaAppID    string `json:"algolia_app_id"`
	AlgoliaAPIKey   string `json:"algolia_api_key"`
	AlgoliaIndexName string `json:"algolia_index_name"`
	Port            string `json:"port"`
}

func Load() (*Config, error) {
	cfg := &Config{}

	// Try to load from config.json first
	if data, err := os.ReadFile("config.json"); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config.json: %w", err)
		}
	}

	// Override with environment variables if set
	if appID := os.Getenv("ALGOLIA_APP_ID"); appID != "" {
		cfg.AlgoliaAppID = appID
	}
	if apiKey := os.Getenv("ALGOLIA_API_KEY"); apiKey != "" {
		cfg.AlgoliaAPIKey = apiKey
	}
	if indexName := os.Getenv("ALGOLIA_INDEX_NAME"); indexName != "" {
		cfg.AlgoliaIndexName = indexName
	}
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}

	// Validate required fields
	if cfg.AlgoliaAppID == "" {
		return nil, fmt.Errorf("ALGOLIA_APP_ID is required")
	}
	if cfg.AlgoliaAPIKey == "" {
		return nil, fmt.Errorf("ALGOLIA_API_KEY is required")
	}
	if cfg.AlgoliaIndexName == "" {
		return nil, fmt.Errorf("ALGOLIA_INDEX_NAME is required")
	}

	return cfg, nil
}
