// Package litellm provides an OpenAI-compatible HTTP client
// that delegates requests to a LiteLLM proxy or directly to OpenAI API.
package litellm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/awch-D/ForgeX/forgex-core/config"
	fxerr "github.com/awch-D/ForgeX/forgex-core/errors"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

type Client struct {
	BaseURL     string
	APIKey      string
	DefaultOpts *provider.Options
	httpClient  *http.Client
}

// NewClient creates a new LiteLLM / OpenAI API compatible client.
func NewClient(cfg *config.LLMConfig) *Client {
	baseURL := strings.TrimSuffix(cfg.Endpoint, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		// Attempt to auto-append v1 if missing from typical base paths
		baseURL = baseURL + "/v1"
	}

	return &Client{
		BaseURL: baseURL,
		APIKey:  cfg.APIKey,
		DefaultOpts: &provider.Options{
			Model:       cfg.Model,
			Temperature: cfg.Temperature,
			MaxTokens:   cfg.MaxTokens,
		},
		httpClient: &http.Client{},
	}
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	ResponseFmt *responseFormat `json:"response_format,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Generate implements provider.Provider.
func (c *Client) Generate(ctx context.Context, messages []provider.Message, opts *provider.Options) (*provider.Response, error) {
	if opts == nil {
		opts = c.DefaultOpts
	}
	model := opts.Model
	if model == "" {
		model = c.DefaultOpts.Model
	}

	reqBody := openAIRequest{
		Model:       model,
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
		Messages:    make([]openAIMessage, len(messages)),
	}

	if opts.JSONMode {
		reqBody.ResponseFmt = &responseFormat{Type: "json_object"}
	}

	for i, m := range messages {
		reqBody.Messages[i] = openAIMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fxerr.Wrap(fxerr.ErrInvalidInput, "failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/chat/completions", c.BaseURL)
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

	var oResp openAIResponse
	if err := json.Unmarshal(bodyBytes, &oResp); err != nil {
		return nil, fxerr.Wrap(fxerr.ErrLLMBadResponse, "failed to decode json response", err)
	}

	if len(oResp.Choices) == 0 {
		return nil, fxerr.New(fxerr.ErrLLMBadResponse, "API returned empty choices")
	}

	actualModel := oResp.Model
	if actualModel == "" {
		actualModel = model
	}

	// Record cost
	cost.Global().Add(actualModel, oResp.Usage.PromptTokens, oResp.Usage.CompletionTokens)

	return &provider.Response{
		Content:      oResp.Choices[0].Message.Content,
		PromptTokens: oResp.Usage.PromptTokens,
		OutputTokens: oResp.Usage.CompletionTokens,
		TotalTokens:  oResp.Usage.TotalTokens,
		Model:        actualModel,
	}, nil
}
