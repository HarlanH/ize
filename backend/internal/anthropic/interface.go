package anthropic

import "context"

// ClientInterface defines the interface for the Anthropic client
// This allows for mocking in tests
type ClientInterface interface {
	GenerateClusterName(ctx context.Context, stats ClusterStats) (string, error)
	GenerateClusterNames(ctx context.Context, statsSlice []ClusterStats) ([]string, error)
}
