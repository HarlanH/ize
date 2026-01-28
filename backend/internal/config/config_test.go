package config

import (
	"os"
	"testing"
)

func TestLoad_FromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("ALGOLIA_APP_ID", "test-app-id")
	os.Setenv("ALGOLIA_API_KEY", "test-api-key")
	os.Setenv("ALGOLIA_INDEX_NAME", "test-index")
	os.Setenv("PORT", "9090")
	defer func() {
		os.Unsetenv("ALGOLIA_APP_ID")
		os.Unsetenv("ALGOLIA_API_KEY")
		os.Unsetenv("ALGOLIA_INDEX_NAME")
		os.Unsetenv("PORT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AlgoliaAppID != "test-app-id" {
		t.Errorf("Load() AlgoliaAppID = %q, want %q", cfg.AlgoliaAppID, "test-app-id")
	}
	if cfg.AlgoliaAPIKey != "test-api-key" {
		t.Errorf("Load() AlgoliaAPIKey = %q, want %q", cfg.AlgoliaAPIKey, "test-api-key")
	}
	if cfg.AlgoliaIndexName != "test-index" {
		t.Errorf("Load() AlgoliaIndexName = %q, want %q", cfg.AlgoliaIndexName, "test-index")
	}
	if cfg.Port != "9090" {
		t.Errorf("Load() Port = %q, want %q", cfg.Port, "9090")
	}
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	// Clear all env vars
	os.Unsetenv("ALGOLIA_APP_ID")
	os.Unsetenv("ALGOLIA_API_KEY")
	os.Unsetenv("ALGOLIA_INDEX_NAME")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for missing required fields, got nil")
	}
}
