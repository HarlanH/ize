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

// TestGenerateClusterNames_Integration tests generating names for multiple clusters
// Run with: go test -tags=integration -v ./internal/anthropic/... -run MultiCluster
func TestGenerateClusterNames_MultiCluster_Integration(t *testing.T) {
	// Load config to get API key
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

	testClusters := []ClusterStats{
		{
			Size: 15,
			TopFacets: []FacetInfo{
				{Name: "category", Value: "Clothing", Percentage: 100.0},
				{Name: "brand", Value: "Nike", Percentage: 60.0},
				{Name: "type", Value: "Shoes", Percentage: 80.0},
			},
		},
		{
			Size: 8,
			TopFacets: []FacetInfo{
				{Name: "category", Value: "Home", Percentage: 100.0},
				{Name: "material", Value: "Wood", Percentage: 75.0},
				{Name: "style", Value: "Modern", Percentage: 50.0},
			},
		},
		{
			Size: 20,
			TopFacets: []FacetInfo{
				{Name: "category", Value: "Books", Percentage: 100.0},
				{Name: "genre", Value: "Fiction", Percentage: 85.0},
				{Name: "format", Value: "Paperback", Percentage: 70.0},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	names, err := client.GenerateClusterNames(ctx, testClusters)
	if err != nil {
		t.Fatalf("GenerateClusterNames() error = %v", err)
	}

	t.Logf("Generated %d cluster names:", len(names))
	for i, name := range names {
		t.Logf("  Cluster %d: %q (size=%d)", i+1, name, testClusters[i].Size)
	}

	// Basic validation
	if len(names) != len(testClusters) {
		t.Errorf("GenerateClusterNames() returned %d names, want %d", len(names), len(testClusters))
	}

	for i, name := range names {
		if name == "" {
			t.Errorf("Cluster %d has empty name", i)
		}
	}
}
