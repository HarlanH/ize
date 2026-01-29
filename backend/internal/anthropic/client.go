package anthropic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"ize/internal/logger"
)

const (
	apiURL          = "https://api.anthropic.com/v1/messages"
	apiVersion      = "2023-06-01"
	model           = "claude-3-haiku-20240307"
	defaultCacheTTL = 1 * time.Hour
)

// cacheEntry holds a cached cluster name with expiration
type cacheEntry struct {
	name      string
	expiresAt time.Time
}

// Client wraps the Anthropic API client
type Client struct {
	apiKey     string
	httpClient *http.Client
	logger     *logger.Logger
	cacheTTL   time.Duration
	cache      map[string]cacheEntry
	cacheMu    sync.RWMutex
}

// NewClient creates a new Anthropic client
func NewClient(apiKey string, log *logger.Logger) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}

	log.Info("anthropic client initialized",
		"cache_ttl", defaultCacheTTL.String(),
	)

	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:   log,
		cacheTTL: defaultCacheTTL,
		cache:    make(map[string]cacheEntry),
	}, nil
}

// cacheKey generates a deterministic cache key from ClusterStats
func (c *Client) cacheKey(stats ClusterStats) string {
	// Build a deterministic string representation
	var parts []string
	parts = append(parts, fmt.Sprintf("size:%d", stats.Size))

	// Sort facets for deterministic ordering
	facetStrings := make([]string, 0, len(stats.TopFacets))
	for _, f := range stats.TopFacets {
		facetStrings = append(facetStrings, fmt.Sprintf("%s:%s:%.1f", f.Name, f.Value, f.Percentage))
	}
	sort.Strings(facetStrings)
	parts = append(parts, facetStrings...)

	// Hash for shorter key
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:16]) // Use first 16 bytes (32 hex chars)
}

// getCached returns a cached name if it exists and hasn't expired
func (c *Client) getCached(key string) (string, bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	entry, ok := c.cache[key]
	if !ok {
		return "", false
	}
	if time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.name, true
}

// setCache stores a name in the cache
func (c *Client) setCache(key, name string) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	c.cache[key] = cacheEntry{
		name:      name,
		expiresAt: time.Now().Add(c.cacheTTL),
	}
}

// ClusterStats holds statistics about a cluster for labeling
type ClusterStats struct {
	Size      int
	TopFacets []FacetInfo
}

// FacetInfo holds facet information for the prompt
type FacetInfo struct {
	Name       string
	Value      string
	Percentage float64
}

// messageRequest represents the Anthropic API request format
type messageRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

// message represents a single message in the conversation
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// messageResponse represents the Anthropic API response format
type messageResponse struct {
	Content []contentBlock `json:"content"`
	Error   *apiError      `json:"error,omitempty"`
}

// contentBlock represents a content block in the response
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// apiError represents an API error
type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// isRetryableStatus returns true if the HTTP status code indicates a transient error
func isRetryableStatus(status int) bool {
	return status == 429 || // Rate limited
		status == 500 || // Internal server error
		status == 502 || // Bad gateway
		status == 503 || // Service unavailable
		status == 529 // Overloaded
}

// GenerateClusterName generates a pithy 1-3 word label for a cluster
// Includes retry logic for transient errors with exponential backoff
// Results are cached in-memory with a 1 hour TTL
func (c *Client) GenerateClusterName(ctx context.Context, stats ClusterStats) (string, error) {
	log := c.logger.WithContext(ctx)

	// Check cache first
	key := c.cacheKey(stats)
	if cached, ok := c.getCached(key); ok {
		log.Debug("cluster name cache hit",
			"cluster_size", stats.Size,
			"name", cached,
		)
		return cached, nil
	}

	// Build the prompt
	var facetLines []string
	for _, f := range stats.TopFacets {
		facetLines = append(facetLines, fmt.Sprintf("- %s:%s (%.0f%%)", f.Name, f.Value, f.Percentage))
	}
	facetList := strings.Join(facetLines, "\n")

	prompt := fmt.Sprintf(`Given these facet characteristics of a product cluster:
%s
- %d items total

Generate a pithy 1-3 word label for this cluster that captures what makes these items similar.
Respond with ONLY the label, nothing else. No quotes, no punctuation, just the label words.`, facetList, stats.Size)

	log.Debug("generating cluster name (cache miss)",
		"cluster_size", stats.Size,
		"top_facets_count", len(stats.TopFacets),
	)

	// Retry configuration
	maxRetries := 3
	baseDelay := 500 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 500ms, 1s, 2s
			delay := baseDelay * time.Duration(1<<(attempt-1))
			log.Debug("retrying anthropic API call",
				"attempt", attempt+1,
				"delay_ms", delay.Milliseconds(),
			)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		label, statusCode, err := c.doGenerateRequest(ctx, prompt)
		if err == nil {
			// Cache the successful result
			c.setCache(key, label)
			log.Debug("generated cluster name",
				"label", label,
			)
			return label, nil
		}

		lastErr = err

		// Only retry on transient errors
		if statusCode > 0 && !isRetryableStatus(statusCode) {
			log.Error("anthropic API returned non-retryable error",
				"status", statusCode,
				"error", err,
			)
			return "", err
		}

		if attempt < maxRetries {
			log.Warn("anthropic API call failed, will retry",
				"attempt", attempt+1,
				"max_retries", maxRetries,
				"error", err,
			)
		}
	}

	return "", fmt.Errorf("failed after %d retries: %w", maxRetries+1, lastErr)
}

// doGenerateRequest makes a single API request and returns the label, status code, and error
func (c *Client) doGenerateRequest(ctx context.Context, prompt string) (string, int, error) {
	reqBody := messageRequest{
		Model:     model,
		MaxTokens: 20,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var msgResp messageResponse
	if err := json.Unmarshal(body, &msgResp); err != nil {
		return "", resp.StatusCode, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if msgResp.Error != nil {
		return "", resp.StatusCode, fmt.Errorf("API error: %s - %s", msgResp.Error.Type, msgResp.Error.Message)
	}

	if len(msgResp.Content) == 0 || msgResp.Content[0].Type != "text" {
		return "", resp.StatusCode, fmt.Errorf("unexpected response format")
	}

	label := strings.TrimSpace(msgResp.Content[0].Text)
	return label, resp.StatusCode, nil
}

// GenerateClusterNames generates names for multiple clusters in parallel
func (c *Client) GenerateClusterNames(ctx context.Context, statsSlice []ClusterStats) ([]string, error) {
	log := c.logger.WithContext(ctx)

	if len(statsSlice) == 0 {
		return []string{}, nil
	}

	start := time.Now()
	results := make([]string, len(statsSlice))

	// Result struct for channel communication
	type result struct {
		index int
		name  string
		err   error
	}

	resultCh := make(chan result, len(statsSlice))

	// Launch parallel goroutines for each cluster
	for i, stats := range statsSlice {
		go func(idx int, s ClusterStats) {
			name, err := c.GenerateClusterName(ctx, s)
			resultCh <- result{index: idx, name: name, err: err}
		}(i, stats)
	}

	// Collect results
	var errorCount int
	for range statsSlice {
		r := <-resultCh
		if r.err != nil {
			log.Warn("failed to generate cluster name, using fallback",
				"cluster_index", r.index,
				"error", r.err,
			)
			results[r.index] = fmt.Sprintf("Cluster %d", r.index+1)
			errorCount++
		} else {
			results[r.index] = r.name
		}
	}

	log.Info("generated cluster names in parallel",
		"cluster_count", len(statsSlice),
		"errors", errorCount,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return results, nil
}
