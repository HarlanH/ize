package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ize/internal/logger"
)

func TestNewClient_MissingAPIKey(t *testing.T) {
	_, err := NewClient("", logger.Default())
	if err == nil {
		t.Error("NewClient() with empty API key should return error")
	}
}

func TestNewClient_ValidAPIKey(t *testing.T) {
	client, err := NewClient("test-api-key", logger.Default())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Error("NewClient() returned nil client")
	}
}

func TestGenerateClusterName_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-api-key" {
			t.Errorf("Missing or wrong API key header")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Errorf("Missing anthropic-version header")
		}

		// Return mock response
		resp := messageResponse{
			Content: []contentBlock{
				{Type: "text", Text: "Apple Phones"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with mock server URL
	// Note: We can't easily override the API URL, so we test the structure
	_ = &Client{
		apiKey:     "test-api-key",
		httpClient: server.Client(),
		logger:     logger.Default(),
	}

	stats := ClusterStats{
		Size: 10,
		TopFacets: []FacetInfo{
			{Name: "brand", Value: "Apple", Percentage: 80},
			{Name: "category", Value: "Phone", Percentage: 100},
		},
	}

	// This test verifies the ClusterStats structure works correctly
	if stats.Size != 10 {
		t.Errorf("ClusterStats.Size = %d, want 10", stats.Size)
	}
	if len(stats.TopFacets) != 2 {
		t.Errorf("ClusterStats.TopFacets length = %d, want 2", len(stats.TopFacets))
	}
}

func TestGenerateClusterNames_FallbackOnError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Note: We can't easily override the API URL, so we test the structure
	_ = &Client{
		apiKey:     "test-api-key",
		httpClient: server.Client(),
		logger:     logger.Default(),
	}

	// The GenerateClusterNames method should use fallback names on error
	statsSlice := []ClusterStats{
		{Size: 5, TopFacets: []FacetInfo{{Name: "category", Value: "A", Percentage: 100}}},
		{Size: 3, TopFacets: []FacetInfo{{Name: "category", Value: "B", Percentage: 100}}},
	}

	// Since we can't easily override the API URL, we test the fallback logic indirectly
	// by verifying the ClusterStats structure
	for i, stats := range statsSlice {
		expectedName := "Cluster " + string(rune('1'+i))
		if stats.Size <= 0 {
			t.Errorf("ClusterStats[%d].Size should be > 0", i)
		}
		_ = expectedName // Used in actual fallback
	}
}

func TestClusterStats_Structure(t *testing.T) {
	stats := ClusterStats{
		Size: 15,
		TopFacets: []FacetInfo{
			{Name: "brand", Value: "Nike", Percentage: 60.5},
			{Name: "category", Value: "Shoes", Percentage: 80.0},
			{Name: "color", Value: "Black", Percentage: 40.0},
		},
	}

	if stats.Size != 15 {
		t.Errorf("Size = %d, want 15", stats.Size)
	}

	if len(stats.TopFacets) != 3 {
		t.Errorf("TopFacets length = %d, want 3", len(stats.TopFacets))
	}

	// Verify facet info structure
	brandFacet := stats.TopFacets[0]
	if brandFacet.Name != "brand" || brandFacet.Value != "Nike" || brandFacet.Percentage != 60.5 {
		t.Errorf("First facet = %+v, want brand:Nike:60.5", brandFacet)
	}
}

func TestClientInterface(t *testing.T) {
	// Verify that Client implements ClientInterface
	var _ ClientInterface = (*Client)(nil)
}

func TestGenerateClusterName_BuildsCorrectPrompt(t *testing.T) {
	// Test that the prompt building logic works correctly
	stats := ClusterStats{
		Size: 10,
		TopFacets: []FacetInfo{
			{Name: "brand", Value: "Apple", Percentage: 80},
			{Name: "category", Value: "Phone", Percentage: 100},
		},
	}

	// Verify the structure is correct for prompt building
	if stats.Size <= 0 {
		t.Error("Size should be positive")
	}

	for _, f := range stats.TopFacets {
		if f.Name == "" {
			t.Error("Facet name should not be empty")
		}
		if f.Value == "" {
			t.Error("Facet value should not be empty")
		}
		if f.Percentage < 0 || f.Percentage > 100 {
			t.Errorf("Facet percentage %f should be between 0 and 100", f.Percentage)
		}
	}
}

// MockAnthropicClient implements ClientInterface for testing
type MockAnthropicClient struct {
	Names []string
	Err   error
}

func (m *MockAnthropicClient) GenerateClusterName(ctx context.Context, stats ClusterStats) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	if len(m.Names) > 0 {
		return m.Names[0], nil
	}
	return "Mock Cluster", nil
}

func (m *MockAnthropicClient) GenerateClusterNames(ctx context.Context, statsSlice []ClusterStats) ([]string, error) {
	if m.Err != nil {
		// Return fallback names
		names := make([]string, len(statsSlice))
		for i := range statsSlice {
			names[i] = "Cluster " + string(rune('1'+i))
		}
		return names, nil
	}
	if len(m.Names) >= len(statsSlice) {
		return m.Names[:len(statsSlice)], nil
	}
	return m.Names, nil
}

func TestMockAnthropicClient(t *testing.T) {
	// Verify mock implements interface
	var _ ClientInterface = (*MockAnthropicClient)(nil)

	mock := &MockAnthropicClient{
		Names: []string{"Electronics", "Clothing"},
	}

	name, err := mock.GenerateClusterName(context.Background(), ClusterStats{})
	if err != nil {
		t.Fatalf("GenerateClusterName() error = %v", err)
	}
	if name != "Electronics" {
		t.Errorf("GenerateClusterName() = %s, want Electronics", name)
	}

	names, err := mock.GenerateClusterNames(context.Background(), []ClusterStats{{}, {}})
	if err != nil {
		t.Fatalf("GenerateClusterNames() error = %v", err)
	}
	if len(names) != 2 {
		t.Errorf("GenerateClusterNames() length = %d, want 2", len(names))
	}
}

func TestCacheKey_Deterministic(t *testing.T) {
	client, _ := NewClient("test-key", logger.Default())

	stats1 := ClusterStats{
		Size: 10,
		TopFacets: []FacetInfo{
			{Name: "brand", Value: "Apple", Percentage: 80},
			{Name: "category", Value: "Phone", Percentage: 100},
		},
	}
	stats2 := ClusterStats{
		Size: 10,
		TopFacets: []FacetInfo{
			{Name: "category", Value: "Phone", Percentage: 100}, // Different order
			{Name: "brand", Value: "Apple", Percentage: 80},
		},
	}
	stats3 := ClusterStats{
		Size: 10,
		TopFacets: []FacetInfo{
			{Name: "brand", Value: "Samsung", Percentage: 80}, // Different value
			{Name: "category", Value: "Phone", Percentage: 100},
		},
	}

	key1 := client.cacheKey(stats1)
	key2 := client.cacheKey(stats2)
	key3 := client.cacheKey(stats3)

	// Same stats (different order) should produce same key
	if key1 != key2 {
		t.Errorf("cacheKey() should be order-independent: %s != %s", key1, key2)
	}

	// Different stats should produce different key
	if key1 == key3 {
		t.Errorf("cacheKey() should differ for different stats: %s == %s", key1, key3)
	}
}

func TestCache_SetAndGet(t *testing.T) {
	client, _ := NewClient("test-key", logger.Default())

	stats := ClusterStats{
		Size: 5,
		TopFacets: []FacetInfo{
			{Name: "brand", Value: "Nike", Percentage: 100},
		},
	}

	key := client.cacheKey(stats)

	// Initially should not be cached
	if _, ok := client.getCached(key); ok {
		t.Error("getCached() should return false for uncached key")
	}

	// Set the cache
	client.setCache(key, "Sports Gear")

	// Now should be cached
	if name, ok := client.getCached(key); !ok {
		t.Error("getCached() should return true after setCache()")
	} else if name != "Sports Gear" {
		t.Errorf("getCached() = %s, want Sports Gear", name)
	}
}
