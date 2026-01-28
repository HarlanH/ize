package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ize/internal/algolia"
	"ize/internal/logger"
)

// mockAlgoliaClient is a mock implementation of ClientInterface for testing
type mockAlgoliaClient struct {
	searchFunc      func(ctx context.Context, query string, facetFilters [][]string) (*algolia.SearchResult, error)
	searchRipperFunc func(ctx context.Context, query string, facetFilters [][]string) (*algolia.SearchResult, error)
}

func (m *mockAlgoliaClient) Search(ctx context.Context, query string, facetFilters [][]string) (*algolia.SearchResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query, facetFilters)
	}
	return &algolia.SearchResult{Hits: []algolia.Hit{}}, nil
}

func (m *mockAlgoliaClient) SearchRipper(ctx context.Context, query string, facetFilters [][]string) (*algolia.SearchResult, error) {
	if m.searchRipperFunc != nil {
		return m.searchRipperFunc(ctx, query, facetFilters)
	}
	return &algolia.SearchResult{Hits: []algolia.Hit{}}, nil
}

func TestSearchHandler_HandleSearch(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    SearchRequest
		mockSearchFunc func(ctx context.Context, query string, facetFilters [][]string) (*algolia.SearchResult, error)
		wantStatus     int
		wantHitsCount  int
	}{
		{
			name:        "successful search",
			requestBody: SearchRequest{Query: "test"},
			mockSearchFunc: func(ctx context.Context, query string, facetFilters [][]string) (*algolia.SearchResult, error) {
				return &algolia.SearchResult{
					Hits: []algolia.Hit{
						{
							ObjectID:    "1",
							Name:        "Test Product",
							Description: "A test product",
							Image:       "https://example.com/image.jpg",
						},
					},
				}, nil
			},
			wantStatus:    http.StatusOK,
			wantHitsCount: 1,
		},
		{
			name:        "empty query",
			requestBody: SearchRequest{Query: ""},
			wantStatus:   http.StatusOK,
			wantHitsCount: 0,
		},
		{
			name:        "empty results",
			requestBody: SearchRequest{Query: "test"},
			mockSearchFunc: func(ctx context.Context, query string, facetFilters [][]string) (*algolia.SearchResult, error) {
				return &algolia.SearchResult{Hits: []algolia.Hit{}}, nil
			},
			wantStatus:    http.StatusOK,
			wantHitsCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &SearchHandler{
				algoliaClient: &mockAlgoliaClient{searchFunc: tt.mockSearchFunc},
				logger:        logger.Default(),
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/search", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.HandleSearch(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleSearch() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var response SearchResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if len(response.Hits) != tt.wantHitsCount {
					t.Errorf("HandleSearch() hits count = %d, want %d", len(response.Hits), tt.wantHitsCount)
				}
			}
		})
	}
}

func TestSearchHandler_HandleSearch_MethodNotAllowed(t *testing.T) {
	handler := &SearchHandler{
		algoliaClient: &mockAlgoliaClient{},
		logger:        logger.Default(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/search", nil)
	w := httptest.NewRecorder()

	handler.HandleSearch(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleSearch() status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestSearchHandler_HandleSearch_InvalidJSON(t *testing.T) {
	handler := &SearchHandler{
		algoliaClient: &mockAlgoliaClient{},
		logger:        logger.Default(),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/search", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleSearch() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
