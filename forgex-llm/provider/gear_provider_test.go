package provider_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

// mockProvider records the Options it receives for assertion.
type mockProvider struct {
	lastOpts *provider.Options
}

func (m *mockProvider) Generate(ctx context.Context, messages []provider.Message, opts *provider.Options) (*provider.Response, error) {
	m.lastOpts = opts
	return &provider.Response{Content: "mock response"}, nil
}

func (m *mockProvider) Embed(ctx context.Context, text string, opts *provider.EmbeddingOpts) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}

func TestGearProvider_InjectsGearLevel(t *testing.T) {
	mock := &mockProvider{}
	gp := provider.WithGear(mock, 3)

	ctx := context.Background()
	msgs := []provider.Message{{Role: provider.RoleUser, Content: "hello"}}

	_, err := gp.Generate(ctx, msgs, &provider.Options{Temperature: 0.5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastOpts == nil {
		t.Fatal("expected opts to be passed, got nil")
	}
	if mock.lastOpts.GearLevel != 3 {
		t.Errorf("expected GearLevel=3, got %d", mock.lastOpts.GearLevel)
	}
	if mock.lastOpts.Temperature != 0.5 {
		t.Errorf("expected Temperature=0.5, got %f", mock.lastOpts.Temperature)
	}
}

func TestGearProvider_DoesNotOverrideExplicitGearLevel(t *testing.T) {
	mock := &mockProvider{}
	gp := provider.WithGear(mock, 3)

	ctx := context.Background()
	msgs := []provider.Message{{Role: provider.RoleUser, Content: "hello"}}

	_, err := gp.Generate(ctx, msgs, &provider.Options{GearLevel: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastOpts.GearLevel != 1 {
		t.Errorf("expected explicit GearLevel=1 to be preserved, got %d", mock.lastOpts.GearLevel)
	}
}

func TestGearProvider_NilOpts(t *testing.T) {
	mock := &mockProvider{}
	gp := provider.WithGear(mock, 2)

	ctx := context.Background()
	msgs := []provider.Message{{Role: provider.RoleUser, Content: "hello"}}

	_, err := gp.Generate(ctx, msgs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastOpts == nil {
		t.Fatal("expected opts to be created, got nil")
	}
	if mock.lastOpts.GearLevel != 2 {
		t.Errorf("expected GearLevel=2, got %d", mock.lastOpts.GearLevel)
	}
}

func TestGearProvider_EmbedDelegatesUnchanged(t *testing.T) {
	mock := &mockProvider{}
	gp := provider.WithGear(mock, 4)

	ctx := context.Background()
	vec, err := gp.Embed(ctx, "test text", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != 2 {
		t.Errorf("expected 2-dim vector, got %d", len(vec))
	}
}

func TestGearProvider_AllLevels(t *testing.T) {
	for _, level := range []int{1, 2, 3, 4} {
		t.Run(fmt.Sprintf("Level%d", level), func(t *testing.T) {
			mock := &mockProvider{}
			gp := provider.WithGear(mock, level)

			ctx := context.Background()
			msgs := []provider.Message{{Role: provider.RoleUser, Content: "hello"}}
			_, _ = gp.Generate(ctx, msgs, nil)

			if mock.lastOpts.GearLevel != level {
				t.Errorf("expected GearLevel=%d, got %d", level, mock.lastOpts.GearLevel)
			}
		})
	}
}
