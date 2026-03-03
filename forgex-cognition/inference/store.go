// Package inference provides vector-based semantic memory capabilities.
// It stores memory blocks along with their embeddings and allows for similarity search.
package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

// MemoryBlock represents a single piece of inferential memory.
type MemoryBlock struct {
	ID        string            `json:"id"`
	Text      string            `json:"text"`
	Vector    []float32         `json:"vector"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// SearchResult represents a single result from a semantic search.
type SearchResult struct {
	Block MemoryBlock
	Score float32 // Cosine similarity score (0-1, higher is better)
}

// Store provides vector semantic search capabilities.
type Store struct {
	mu       sync.RWMutex
	blocks   []MemoryBlock
	filePath string
	llm      provider.Provider
}

// NewStore initializes a new vector memory store.
func NewStore(filePath string, llm provider.Provider) *Store {
	s := &Store{
		blocks:   make([]MemoryBlock, 0),
		filePath: filePath,
		llm:      llm,
	}
	s.load()
	return s
}

// EmbedAndStore generates an embedding for the text and stores it.
func (s *Store) EmbedAndStore(ctx context.Context, id, text string, metadata map[string]string) error {
	if s.llm == nil {
		return fmt.Errorf("LLM provider is required for embedding")
	}

	start := time.Now()
	vector, err := s.llm.Embed(ctx, text, nil)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	block := MemoryBlock{
		ID:        id,
		Text:      text,
		Vector:    vector,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	// Update if exists, otherwise append
	updated := false
	for i, existing := range s.blocks {
		if existing.ID == id {
			s.blocks[i] = block
			updated = true
			break
		}
	}
	if !updated {
		s.blocks = append(s.blocks, block)
	}

	logger.L().Infow("🧠 Inference embedded and stored",
		"id", id,
		"text_len", len(text),
		"vector_dim", len(vector),
		"duration", time.Since(start).String(),
	)

	return s.save()
}

// SearchSemantic finds the top-K most similar memory blocks.
func (s *Store) SearchSemantic(ctx context.Context, query string, k int) ([]SearchResult, error) {
	if s.llm == nil {
		return nil, fmt.Errorf("LLM provider is required for embedding generation")
	}

	queryVector, err := s.llm.Embed(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.blocks) == 0 {
		return []SearchResult{}, nil
	}

	var results []SearchResult
	for _, block := range s.blocks {
		if len(block.Vector) != len(queryVector) {
			logger.L().Warnw("Dimension mismatch during search",
				"block_dim", len(block.Vector), "query_dim", len(queryVector))
			continue
		}

		score := float32(cosineSimilarity(queryVector, block.Vector))
		results = append(results, SearchResult{
			Block: block,
			Score: score,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit to K
	if len(results) > k {
		results = results[:k]
	}

	return results, nil
}

// Stats returns the number of blocks in the store.
func (s *Store) Stats() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.blocks)
}

func (s *Store) load() {
	if s.filePath == "" {
		return
	}
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return // File might not exist yet
	}
	var loaded []MemoryBlock
	if err := json.Unmarshal(data, &loaded); err == nil {
		s.blocks = loaded
		logger.L().Infow("Loaded inference vectors", "count", len(s.blocks), "file", s.filePath)
	}
}

func (s *Store) save() error {
	if s.filePath == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.blocks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

// cosineSimilarity computes the cosine similarity between two vectors.
// Returns a value between -1.0 and 1.0 (typically 0.0 to 1.0 for text embeddings).
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProd, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProd += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProd / float32(math.Sqrt(float64(normA))*math.Sqrt(float64(normB)))
}
