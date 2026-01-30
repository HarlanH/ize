package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"ize/internal/logger"
)

// FieldMapping configures how Algolia hit fields map to display fields.
// Supports dot notation (e.g., "attributes.Product_Description") and
// array index notation (e.g., "images[0]") for accessing nested fields.
type FieldMapping struct {
	Name        string `json:"name"`        // Path to name field, e.g., "name.en-US" or "name_ecomm"
	Description string `json:"description"` // Path to description field, e.g., "attributes.Product_Description"
	Image       string `json:"image"`       // Path to image field, e.g., "images[0]"
}

// FacetConfig configures which facets to retrieve and how to display them.
type FacetConfig struct {
	Field        string `json:"field"`                  // Algolia facet name, e.g., "attributes.Brand"
	DisplayName  string `json:"displayName"`            // User-friendly name for UI, e.g., "Brand"
	RemovePrefix string `json:"removePrefix,omitempty"` // Optional prefix to strip from facet values, e.g., "Materials > "
}

type Config struct {
	AlgoliaAppID     string        `json:"algolia_app_id"`
	AlgoliaAPIKey    string        `json:"algolia_api_key"`
	AlgoliaIndexName string        `json:"algolia_index_name"`
	AnthropicAPIKey  string        `json:"anthropic_api_key"`
	Port             string        `json:"port"`
	FieldMapping     *FieldMapping `json:"field_mapping,omitempty"`
	Facets           []FacetConfig `json:"facets,omitempty"`
}

// GetFacetFields returns the list of facet field names to request from Algolia.
// Returns ["*"] if no facets are configured.
func (c *Config) GetFacetFields() []string {
	if len(c.Facets) == 0 {
		return []string{"*"}
	}
	fields := make([]string, len(c.Facets))
	for i, f := range c.Facets {
		fields[i] = f.Field
	}
	return fields
}

// GetFacetDisplayName returns the display name for a facet field.
// Returns the field name itself if no display name is configured.
func (c *Config) GetFacetDisplayName(field string) string {
	for _, f := range c.Facets {
		if f.Field == field {
			return f.DisplayName
		}
	}
	return field
}

// arrayIndexRegex matches array index notation like "[0]" or "[123]"
var arrayIndexRegex = regexp.MustCompile(`\[(\d+)\]`)

// ExtractField extracts a value from a nested map using dot notation and array indexing.
// Examples:
//   - "name" -> data["name"]
//   - "name.en-US" -> data["name"]["en-US"]
//   - "images[0]" -> data["images"][0]
//   - "attributes.Style[0].label" -> data["attributes"]["Style"][0]["label"]
func ExtractField(data map[string]interface{}, path string) string {
	if path == "" {
		return ""
	}

	// Split path by dots, but preserve array indices
	parts := splitPath(path)

	var current interface{} = data
	for _, part := range parts {
		if current == nil {
			return ""
		}

		// Check if this part has an array index
		if matches := arrayIndexRegex.FindStringSubmatch(part); len(matches) > 1 {
			// Extract the key name (before the bracket)
			keyName := arrayIndexRegex.ReplaceAllString(part, "")
			index, _ := strconv.Atoi(matches[1])

			// If there's a key name, navigate to it first
			if keyName != "" {
				if m, ok := current.(map[string]interface{}); ok {
					current = m[keyName]
				} else {
					return ""
				}
			}

			// Navigate to array index
			if arr, ok := current.([]interface{}); ok {
				if index >= 0 && index < len(arr) {
					current = arr[index]
				} else {
					return ""
				}
			} else {
				return ""
			}
		} else {
			// Simple key access
			if m, ok := current.(map[string]interface{}); ok {
				current = m[part]
			} else {
				return ""
			}
		}
	}

	// Convert final value to string
	switch v := current.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ""
	}
}

// ExtractFieldValue extracts a value from nested data using dot notation and array indices.
// Unlike ExtractField, this returns the raw value (preserving arrays, objects, etc.)
// which is needed for facet processing in the Ripper algorithm.
//
// Special handling for arrays of objects:
// If the path navigates into an array and the next part is a property name (not an index),
// it extracts that property from ALL objects in the array.
// Example: "attributes.Primary_Color_Family.label" with data containing an array of
// objects like [{label: "Blue"}, {label: "Navy"}] returns ["Blue", "Navy"].
func ExtractFieldValue(data map[string]interface{}, path string) interface{} {
	if path == "" {
		return nil
	}

	// Split path by dots, but preserve array indices
	parts := splitPath(path)

	var current interface{} = data
	for i, part := range parts {
		if current == nil {
			return nil
		}

		// Check if this part has an array index
		if matches := arrayIndexRegex.FindStringSubmatch(part); len(matches) > 1 {
			// Extract the key name (before the bracket)
			keyName := arrayIndexRegex.ReplaceAllString(part, "")
			index, _ := strconv.Atoi(matches[1])

			// If there's a key name, navigate to it first
			if keyName != "" {
				if m, ok := current.(map[string]interface{}); ok {
					current = m[keyName]
				} else {
					return nil
				}
			}

			// Navigate to array index
			if arr, ok := current.([]interface{}); ok {
				if index >= 0 && index < len(arr) {
					current = arr[index]
				} else {
					return nil
				}
			} else {
				return nil
			}
		} else {
			// Simple key access
			if m, ok := current.(map[string]interface{}); ok {
				current = m[part]
			} else if arr, ok := current.([]interface{}); ok {
				// Special case: current is an array of objects and we want a property from each
				// Extract the property from all objects in the array
				remainingPath := strings.Join(parts[i:], ".")
				var results []interface{}
				for _, item := range arr {
					if obj, ok := item.(map[string]interface{}); ok {
						val := ExtractFieldValue(obj, remainingPath)
						if val != nil {
							// If the value is itself an array, flatten it
							if innerArr, ok := val.([]interface{}); ok {
								results = append(results, innerArr...)
							} else {
								results = append(results, val)
							}
						}
					}
				}
				if len(results) > 0 {
					return results
				}
				return nil
			} else {
				return nil
			}
		}
	}

	return current
}

// splitPath splits a path by dots while preserving array brackets.
// "a.b[0].c" -> ["a", "b[0]", "c"]
func splitPath(path string) []string {
	var parts []string
	var current strings.Builder
	inBracket := false

	for _, r := range path {
		switch r {
		case '[':
			inBracket = true
			current.WriteRune(r)
		case ']':
			inBracket = false
			current.WriteRune(r)
		case '.':
			if inBracket {
				current.WriteRune(r)
			} else {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
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
	if anthropicKey := os.Getenv("ANTHROPIC_API_KEY"); anthropicKey != "" {
		cfg.AnthropicAPIKey = anthropicKey
		envVarsSet = append(envVarsSet, "ANTHROPIC_API_KEY")
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
