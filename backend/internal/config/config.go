package config

import (
	"encoding/json"
	"fmt"
	"os"

	"ize/internal/logger"
)

type Config struct {
	AlgoliaAppID    string `json:"algolia_app_id"`
	AlgoliaAPIKey   string `json:"algolia_api_key"`
	AlgoliaIndexName string `json:"algolia_index_name"`
	Port            string `json:"port"`
}

func Load() (*Config, error) {
	log := logger.Default()
	cfg := &Config{}

	// Try to load from config.json first
	if data, err := os.ReadFile("config.json"); err == nil {
		log.Debug("loading configuration from config.json")
		if err := json.Unmarshal(data, cfg); err != nil {
			log.ErrorWithErr("failed to parse config.json", err)
			return nil, fmt.Errorf("failed to parse config.json: %w", err)
		}
		log.Debug("configuration loaded from config.json")
	} else {
		log.Debug("config.json not found, using environment variables only", "error", err)
	}

	// Override with environment variables if set
	envVarsSet := []string{}
	if appID := os.Getenv("ALGOLIA_APP_ID"); appID != "" {
		cfg.AlgoliaAppID = appID
		envVarsSet = append(envVarsSet, "ALGOLIA_APP_ID")
	}
	if apiKey := os.Getenv("ALGOLIA_API_KEY"); apiKey != "" {
		cfg.AlgoliaAPIKey = apiKey
		envVarsSet = append(envVarsSet, "ALGOLIA_API_KEY")
	}
	if indexName := os.Getenv("ALGOLIA_INDEX_NAME"); indexName != "" {
		cfg.AlgoliaIndexName = indexName
		envVarsSet = append(envVarsSet, "ALGOLIA_INDEX_NAME")
	}
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
		envVarsSet = append(envVarsSet, "PORT")
	}
	
	if len(envVarsSet) > 0 {
		log.Debug("configuration overridden by environment variables", "vars", envVarsSet)
	}

	// Validate required fields
	if cfg.AlgoliaAppID == "" {
		log.Error("missing required configuration", "field", "ALGOLIA_APP_ID")
		return nil, fmt.Errorf("ALGOLIA_APP_ID is required")
	}
	if cfg.AlgoliaAPIKey == "" {
		log.Error("missing required configuration", "field", "ALGOLIA_API_KEY")
		return nil, fmt.Errorf("ALGOLIA_API_KEY is required")
	}
	if cfg.AlgoliaIndexName == "" {
		log.Error("missing required configuration", "field", "ALGOLIA_INDEX_NAME")
		return nil, fmt.Errorf("ALGOLIA_INDEX_NAME is required")
	}

	log.Debug("configuration validation passed",
		"port", cfg.Port,
		"algolia_index", cfg.AlgoliaIndexName,
	)

	return cfg, nil
}
