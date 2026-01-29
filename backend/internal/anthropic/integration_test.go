//go:build integration
// +build integration

package anthropic

import (
	"context"
	"os"
	"testing"
	"time"

	"ize/internal/config"
	"ize/internal/logger"
)

// TestGenerateClusterName_Integration tests the real Anthropic API
// Run with: go test -tags=integration -v ./internal/anthropic/...
func TestGenerateClusterName_Integration(t *testing.T) {
	// Load config to get API key
	// Change to backend directory for config loading
	if err := os.Chdir("../.."); err != nil {
		t.Skipf("Skipping integration test: cannot change to backend directory: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Skipping integration test: cannot load config: %v", err)
	}

	if cfg.AnthropicAPIKey == "" {
		t.Skip("Skipping integration test: ANTHROPIC_API_KEY not configured")
	}

	client, err := NewClient(cfg.AnthropicAPIKey, logger.Default())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	stats := ClusterStats{
		Size: 12,
		TopFacets: []FacetInfo{
			{Name: "brand", Value: "Apple", Percentage: 83.3},
			{Name: "category", Value: "Electronics", Percentage: 100.0},
			{Name: "type", Value: "Phone", Percentage: 66.7},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name, err := client.GenerateClusterName(ctx, stats)
	if err != nil {
		t.Fatalf("GenerateClusterName() error = %v", err)
	}

	t.Logf("Generated cluster name: %q", name)

	// Basic validation - should be 1-3 words
	if name == "" {
		t.Error("GenerateClusterName() returned empty string")
	}
	if len(name) > 50 {
		t.Errorf("GenerateClusterName() returned very long name: %q", name)
	}
}
