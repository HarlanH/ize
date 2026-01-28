package ize

import (
	"ize/internal/algolia"
)

// Result represents a processed search result from the ize module
type Result struct {
	ID          string
	Name        string
	Description string
	Image       string
}

// Processor defines the interface for processing search results.
// This allows for different algorithm implementations to be plugged in.
type Processor interface {
	Process(query string, algoliaResults *algolia.SearchResult) []Result
}

// DefaultProcessor is the default pass-through processor.
type DefaultProcessor struct{}

// Process implements the Processor interface with a pass-through algorithm.
// It maps Algolia hits to our result format without modification.
func (p *DefaultProcessor) Process(query string, algoliaResults *algolia.SearchResult) []Result {
	if algoliaResults == nil {
		return []Result{}
	}

	results := make([]Result, 0, len(algoliaResults.Hits))
	for _, hit := range algoliaResults.Hits {
		results = append(results, Result{
			ID:          hit.ObjectID,
			Name:        hit.Name,
			Description: hit.Description,
			Image:       hit.Image,
		})
	}

	return results
}

// defaultProcessor is the singleton instance used by the Process function.
var defaultProcessor Processor = &DefaultProcessor{}

// Process is a convenience function that uses the default processor.
// For custom algorithms, create a new Processor implementation and call it directly.
func Process(query string, algoliaResults *algolia.SearchResult) []Result {
	return defaultProcessor.Process(query, algoliaResults)
}

// SetProcessor allows changing the default processor (useful for testing or future experiment toggling).
func SetProcessor(p Processor) {
	defaultProcessor = p
}
