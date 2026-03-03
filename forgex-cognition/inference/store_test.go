package inference_test

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/awch-D/ForgeX/forgex-cognition/inference"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

// MockProvider generates deterministic pseudo-embeddings for testing
type MockProvider struct{}

func (m *MockProvider) Generate(ctx context.Context, messages []provider.Message, opts *provider.Options) (*provider.Response, error) {
	return nil, nil // Not used in this test
}

// Embed generates a fake embedding. Texts containing "database" get a certain vector,
// texts containing "UI" get another, to test similarity matching.
func (m *MockProvider) Embed(ctx context.Context, text string, opts *provider.EmbeddingOpts) ([]float32, error) {
	if text == "database" || text == "SQL" || text == "query" {
		return []float32{1.0, 0.0, 0.0}, nil
	}
	if text == "frontend" || text == "React" || text == "UI" {
		return []float32{0.0, 1.0, 0.0}, nil
	}
	if text == "network" || text == "TCP" || text == "socket" {
		return []float32{0.0, 0.0, 1.0}, nil
	}
	// Fallback generic vector
	return []float32{0.5, 0.5, 0.5}, nil
}

func TestStore_SearchSemantic(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "inference.json")

	store := inference.NewStore(filePath, &MockProvider{})

	ctx := context.Background()
	// Populate with different semantic concepts
	store.EmbedAndStore(ctx, "req1", "SQL", nil)
	store.EmbedAndStore(ctx, "req2", "React", nil)
	store.EmbedAndStore(ctx, "req3", "TCP", nil)

	// Small sleep to ensure write completion in real world (not strictly needed for mapped slice but good practice)
	time.Sleep(10 * time.Millisecond)

	// Test 1: Query for database related stuff
	t.Run("Database Search", func(t *testing.T) {
		res, err := store.SearchSemantic(ctx, "database", 2)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(res) == 0 || res[0].Block.Text != "SQL" {
			t.Errorf("Expected top result: SQL, got: %v", res)
		}
		// Expect dot product / magnitude = 1.0
		if math.Abs(float64(res[0].Score-1.0)) > 0.01 {
			t.Errorf("Expected score ~1.0, got %f", res[0].Score)
		}
	})

	// Test 2: Overwrite existing ID
	t.Run("Update Existing", func(t *testing.T) {
		store.EmbedAndStore(ctx, "req2", "UI", nil) // Change "React" to "UI" (same conceptual vector in our mock)
		if store.Stats() != 3 {
			t.Errorf("Expected 3 blocks, got %d", store.Stats())
		}
	})

	// Test 3: Reload from disk
	t.Run("Persistence Reload", func(t *testing.T) {
		store2 := inference.NewStore(filePath, &MockProvider{})
		if store2.Stats() != 3 {
			t.Errorf("Loaded store expected 3 blocks, got %d", store2.Stats())
		}
	})
}
