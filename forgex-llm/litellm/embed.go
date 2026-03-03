package litellm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	fxerr "github.com/awch-D/ForgeX/forgex-core/errors"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

type openAIEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed implements provider.Provider for generating vector embeddings.
// It assumes the provider supports the OpenAI compatible /v1/embeddings endpoint.
func (c *Client) Embed(ctx context.Context, text string, opts *provider.EmbeddingOpts) ([]float32, error) {
	model := "text-embedding-3-small" // Default OpenAI embedding model
	if opts != nil && opts.Model != "" {
		model = opts.Model
	} else if c.DefaultOpts.Model != "" {
		// Use default chat model only if explicitly told, otherwise default to a known embedding model
		// It's usually better to have a dedicated setting, but this suffices for fallback
		if opts != nil && opts.Model != "" {
			model = opts.Model
		}
	}

	reqBody := openAIEmbeddingRequest{
		Model: model,
		Input: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fxerr.Wrap(fxerr.ErrInvalidInput, "failed to marshal embedding request", err)
	}

	url := fmt.Sprintf("%s/v1/embeddings", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fxerr.Wrap(fxerr.ErrLLMConnection, "failed to create http request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fxerr.Wrap(fxerr.ErrLLMConnection, "http request failed", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fxerr.Wrap(fxerr.ErrLLMConnection, "failed to read response body", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fxerr.New(fxerr.ErrLLMBadResponse, fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(bodyBytes)))
	}

	var oResp openAIEmbeddingResponse
	if err := json.Unmarshal(bodyBytes, &oResp); err != nil {
		return nil, fxerr.Wrap(fxerr.ErrLLMBadResponse, "failed to decode json response", err)
	}

	if len(oResp.Data) == 0 || len(oResp.Data[0].Embedding) == 0 {
		return nil, fxerr.New(fxerr.ErrLLMBadResponse, "API returned empty embedding data")
	}

	actualModel := oResp.Model
	if actualModel == "" {
		actualModel = model
	}

	// Cost tracking: using outputTokens = 0 for embeddings
	cost.Global().Add(actualModel, oResp.Usage.PromptTokens, 0)

	return oResp.Data[0].Embedding, nil
}
