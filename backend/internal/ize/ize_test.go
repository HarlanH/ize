package ize

import (
	"ize/internal/algolia"
	"testing"
)

func TestDefaultProcessor_Process_PassThrough(t *testing.T) {
	processor := &DefaultProcessor{}
	
	tests := []struct {
		name           string
		query          string
		algoliaResults *algolia.SearchResult
		wantCount      int
		wantFirstID    string
	}{
		{
			name:  "empty results",
			query: "test",
			algoliaResults: &algolia.SearchResult{
				Hits: []algolia.Hit{},
			},
			wantCount:   0,
			wantFirstID: "",
		},
		{
			name:  "single result",
			query: "test",
			algoliaResults: &algolia.SearchResult{
				Hits: []algolia.Hit{
					{
						ObjectID:    "123",
						Name:        "Test Product",
						Description: "A test product",
						Image:       "https://example.com/image.jpg",
					},
				},
			},
			wantCount:   1,
			wantFirstID: "123",
		},
		{
			name:  "multiple results",
			query: "test",
			algoliaResults: &algolia.SearchResult{
				Hits: []algolia.Hit{
					{
						ObjectID:    "123",
						Name:        "Product 1",
						Description: "First product",
						Image:       "https://example.com/img1.jpg",
					},
					{
						ObjectID:    "456",
						Name:        "Product 2",
						Description: "Second product",
						Image:       "https://example.com/img2.jpg",
					},
				},
			},
			wantCount:   2,
			wantFirstID: "123",
		},
		{
			name:           "nil results",
			query:          "test",
			algoliaResults: nil,
			wantCount:      0,
			wantFirstID:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processor.Process(tt.query, tt.algoliaResults)

			if len(got) != tt.wantCount {
				t.Errorf("Process() returned %d results, want %d", len(got), tt.wantCount)
			}

			if tt.wantCount > 0 {
				if got[0].ID != tt.wantFirstID {
					t.Errorf("Process() first result ID = %q, want %q", got[0].ID, tt.wantFirstID)
				}
				// Verify pass-through mapping
				if tt.algoliaResults != nil && len(tt.algoliaResults.Hits) > 0 {
					firstHit := tt.algoliaResults.Hits[0]
					if got[0].Name != firstHit.Name {
						t.Errorf("Process() first result Name = %q, want %q", got[0].Name, firstHit.Name)
					}
					if got[0].Description != firstHit.Description {
						t.Errorf("Process() first result Description = %q, want %q", got[0].Description, firstHit.Description)
					}
					if got[0].Image != firstHit.Image {
						t.Errorf("Process() first result Image = %q, want %q", got[0].Image, firstHit.Image)
					}
				}
			}
		})
	}
}

func TestProcess_PassThrough(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		algoliaResults *algolia.SearchResult
		wantCount      int
		wantFirstID    string
	}{
		{
			name:  "empty results",
			query: "test",
			algoliaResults: &algolia.SearchResult{
				Hits: []algolia.Hit{},
			},
			wantCount:   0,
			wantFirstID: "",
		},
		{
			name:  "single result",
			query: "test",
			algoliaResults: &algolia.SearchResult{
				Hits: []algolia.Hit{
					{
						ObjectID:    "123",
						Name:        "Test Product",
						Description: "A test product",
						Image:       "https://example.com/image.jpg",
					},
				},
			},
			wantCount:   1,
			wantFirstID: "123",
		},
		{
			name:  "multiple results",
			query: "test",
			algoliaResults: &algolia.SearchResult{
				Hits: []algolia.Hit{
					{
						ObjectID:    "123",
						Name:        "Product 1",
						Description: "First product",
						Image:       "https://example.com/img1.jpg",
					},
					{
						ObjectID:    "456",
						Name:        "Product 2",
						Description: "Second product",
						Image:       "https://example.com/img2.jpg",
					},
				},
			},
			wantCount:   2,
			wantFirstID: "123",
		},
		{
			name:           "nil results",
			query:          "test",
			algoliaResults: nil,
			wantCount:      0,
			wantFirstID:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Process(tt.query, tt.algoliaResults)

			if len(got) != tt.wantCount {
				t.Errorf("Process() returned %d results, want %d", len(got), tt.wantCount)
			}

			if tt.wantCount > 0 {
				if got[0].ID != tt.wantFirstID {
					t.Errorf("Process() first result ID = %q, want %q", got[0].ID, tt.wantFirstID)
				}
				// Verify pass-through mapping
				if tt.algoliaResults != nil && len(tt.algoliaResults.Hits) > 0 {
					firstHit := tt.algoliaResults.Hits[0]
					if got[0].Name != firstHit.Name {
						t.Errorf("Process() first result Name = %q, want %q", got[0].Name, firstHit.Name)
					}
					if got[0].Description != firstHit.Description {
						t.Errorf("Process() first result Description = %q, want %q", got[0].Description, firstHit.Description)
					}
					if got[0].Image != firstHit.Image {
						t.Errorf("Process() first result Image = %q, want %q", got[0].Image, firstHit.Image)
					}
				}
			}
		})
	}
}

func TestProcess_ResultShape(t *testing.T) {
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{
				ObjectID:    "test-id",
				Name:        "Test Name",
				Description: "Test Description",
				Image:       "https://test.com/image.jpg",
			},
		},
	}

	results := Process("test", algoliaResults)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.ID != "test-id" {
		t.Errorf("Result ID = %q, want %q", result.ID, "test-id")
	}
	if result.Name != "Test Name" {
		t.Errorf("Result Name = %q, want %q", result.Name, "Test Name")
	}
}

func TestSetProcessor(t *testing.T) {
	// Save original processor
	original := defaultProcessor

	// Create a custom processor for testing
	customProcessor := &DefaultProcessor{}
	SetProcessor(customProcessor)

	if defaultProcessor != customProcessor {
		t.Error("SetProcessor() did not update the default processor")
	}

	// Restore original
	SetProcessor(original)
}
