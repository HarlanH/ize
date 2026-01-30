package config

import (
	"os"
	"testing"
)

func TestExtractField(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		path     string
		expected string
	}{
		{
			name:     "simple top-level string",
			data:     map[string]interface{}{"name": "Test Product"},
			path:     "name",
			expected: "Test Product",
		},
		{
			name:     "nested field with dot notation",
			data:     map[string]interface{}{"name": map[string]interface{}{"en-US": "English Name"}},
			path:     "name.en-US",
			expected: "English Name",
		},
		{
			name: "deeply nested field",
			data: map[string]interface{}{
				"attributes": map[string]interface{}{
					"Product_Description": "A great product",
				},
			},
			path:     "attributes.Product_Description",
			expected: "A great product",
		},
		{
			name: "array index at top level",
			data: map[string]interface{}{
				"images": []interface{}{"img1.jpg", "img2.jpg", "img3.jpg"},
			},
			path:     "images[0]",
			expected: "img1.jpg",
		},
		{
			name: "array index with nested access",
			data: map[string]interface{}{
				"attributes": map[string]interface{}{
					"Style": []interface{}{
						map[string]interface{}{"label": "Modern"},
						map[string]interface{}{"label": "Contemporary"},
					},
				},
			},
			path:     "attributes.Style[0].label",
			expected: "Modern",
		},
		{
			name:     "missing field returns empty",
			data:     map[string]interface{}{"name": "Test"},
			path:     "nonexistent",
			expected: "",
		},
		{
			name:     "missing nested field returns empty",
			data:     map[string]interface{}{"name": map[string]interface{}{"en": "English"}},
			path:     "name.fr",
			expected: "",
		},
		{
			name: "array index out of bounds returns empty",
			data: map[string]interface{}{
				"images": []interface{}{"img1.jpg"},
			},
			path:     "images[5]",
			expected: "",
		},
		{
			name:     "empty path returns empty",
			data:     map[string]interface{}{"name": "Test"},
			path:     "",
			expected: "",
		},
		{
			name:     "numeric value converted to string",
			data:     map[string]interface{}{"price": 99.99},
			path:     "price",
			expected: "99.99",
		},
		{
			name:     "integer value converted to string",
			data:     map[string]interface{}{"count": float64(42)}, // JSON numbers are float64
			path:     "count",
			expected: "42",
		},
		{
			name:     "boolean value converted to string",
			data:     map[string]interface{}{"active": true},
			path:     "active",
			expected: "true",
		},
		{
			name: "second array element",
			data: map[string]interface{}{
				"tags": []interface{}{"tag1", "tag2", "tag3"},
			},
			path:     "tags[1]",
			expected: "tag2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractField(tt.data, tt.path)
			if result != tt.expected {
				t.Errorf("ExtractField() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConfig_GetFacetFields(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected []string
	}{
		{
			name:     "empty facets returns wildcard",
			config:   Config{},
			expected: []string{"*"},
		},
		{
			name: "configured facets returns field names",
			config: Config{
				Facets: []FacetConfig{
					{Field: "attributes.Brand", DisplayName: "Brand"},
					{Field: "attributes.Style.label", DisplayName: "Style"},
				},
			},
			expected: []string{"attributes.Brand", "attributes.Style.label"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetFacetFields()
			if len(result) != len(tt.expected) {
				t.Errorf("GetFacetFields() returned %d items, want %d", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("GetFacetFields()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestConfig_GetFacetDisplayName(t *testing.T) {
	config := Config{
		Facets: []FacetConfig{
			{Field: "attributes.Brand", DisplayName: "Brand"},
			{Field: "attributes.Style.label", DisplayName: "Style"},
		},
	}

	tests := []struct {
		field    string
		expected string
	}{
		{"attributes.Brand", "Brand"},
		{"attributes.Style.label", "Style"},
		{"unknown.field", "unknown.field"}, // Falls back to field name
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := config.GetFacetDisplayName(tt.field)
			if result != tt.expected {
				t.Errorf("GetFacetDisplayName(%q) = %q, want %q", tt.field, result, tt.expected)
			}
		})
	}
}

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
