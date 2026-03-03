// Package litellm provides an HTTP client supporting both OpenAI-compatible
// and Anthropic Messages API formats for LLM proxies.
package litellm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/awch-D/ForgeX/forgex-core/config"
	fxerr "github.com/awch-D/ForgeX/forgex-core/errors"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

type Client struct {
	BaseURL     string
	APIKey      string
	APIFormat   string // "openai" or "anthropic"
	DefaultOpts *provider.Options
	httpClient  *http.Client
}

// NewClient creates a new LLM API client.
// It auto-detects whether to use OpenAI or Anthropic format based on config.
func NewClient(cfg *config.LLMConfig) *Client {
	baseURL := strings.TrimSuffix(cfg.Endpoint, "/")

	// Auto-detect API format from model name
	apiFormat := "openai"
	if strings.Contains(cfg.Model, "claude") {
		apiFormat = "anthropic"
	}

	return &Client{
		BaseURL:   baseURL,
		APIKey:    cfg.APIKey,
		APIFormat: apiFormat,
		DefaultOpts: &provider.Options{
			Model:       cfg.Model,
			Temperature: cfg.Temperature,
			MaxTokens:   cfg.MaxTokens,
		},
		httpClient: &http.Client{
			Timeout: 3 * time.Minute, // prevent API proxy hangs
		},
	}
}

// Generate implements provider.Provider.
func (c *Client) Generate(ctx context.Context, messages []provider.Message, opts *provider.Options) (*provider.Response, error) {
	if c.APIFormat == "anthropic" {
		return c.generateAnthropic(ctx, messages, opts)
	}
	return c.generateOpenAI(ctx, messages, opts)
}

// ==================== Anthropic Messages API ====================

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Temperature float64            `json:"temperature,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) generateAnthropic(ctx context.Context, messages []provider.Message, opts *provider.Options) (*provider.Response, error) {
	if opts == nil {
		opts = c.DefaultOpts
	}
	model := opts.Model
	if model == "" {
		model = c.DefaultOpts.Model
	}
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 16384
	}

	// Separate system message from conversation messages
	var systemContent string
	var convMessages []anthropicMessage

	for _, m := range messages {
		if m.Role == provider.RoleSystem {
			systemContent += m.Content + "\n"
		} else {
			role := string(m.Role)
			convMessages = append(convMessages, anthropicMessage{
				Role:    role,
				Content: m.Content,
			})
		}
	}

	// If JSON mode is requested, append instruction to system prompt
	if opts.JSONMode && systemContent != "" {
		systemContent += "\nIMPORTANT: You MUST respond with valid JSON only. No markdown, no extra text."
	}

	reqBody := anthropicRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		System:      strings.TrimSpace(systemContent),
		Messages:    convMessages,
		Temperature: opts.Temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fxerr.Wrap(fxerr.ErrInvalidInput, "failed to marshal anthropic request", err)
	}

	url := fmt.Sprintf("%s/v1/messages", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fxerr.Wrap(fxerr.ErrLLMConnection, "failed to create http request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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
		return nil, fxerr.New(fxerr.ErrLLMBadResponse, fmt.Sprintf("Anthropic API returned status %d: %s", resp.StatusCode, string(bodyBytes)))
	}

	var aResp anthropicResponse
	if err := json.Unmarshal(bodyBytes, &aResp); err != nil {
		return nil, fxerr.Wrap(fxerr.ErrLLMBadResponse, "failed to decode anthropic json response", err)
	}

	if aResp.Error != nil {
		return nil, fxerr.New(fxerr.ErrLLMBadResponse, fmt.Sprintf("Anthropic error: %s", aResp.Error.Message))
	}

	// Extract text content
	var content string
	for _, block := range aResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	actualModel := aResp.Model
	if actualModel == "" {
		actualModel = model
	}

	cost.Global().Add(actualModel, aResp.Usage.InputTokens, aResp.Usage.OutputTokens)

	return &provider.Response{
		Content:      content,
		PromptTokens: aResp.Usage.InputTokens,
		OutputTokens: aResp.Usage.OutputTokens,
		TotalTokens:  aResp.Usage.InputTokens + aResp.Usage.OutputTokens,
		Model:        actualModel,
	}, nil
}

// ==================== OpenAI Compatible API ====================

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
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *Client) generateOpenAI(ctx context.Context, messages []provider.Message, opts *provider.Options) (*provider.Response, error) {
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
		reqBody.Messages[i] = openAIMessage{Role: string(m.Role), Content: m.Content}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fxerr.Wrap(fxerr.ErrInvalidInput, "failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.BaseURL)
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

	cost.Global().Add(actualModel, oResp.Usage.PromptTokens, oResp.Usage.CompletionTokens)

	return &provider.Response{
		Content:      oResp.Choices[0].Message.Content,
		PromptTokens: oResp.Usage.PromptTokens,
		OutputTokens: oResp.Usage.CompletionTokens,
		TotalTokens:  oResp.Usage.TotalTokens,
		Model:        actualModel,
	}, nil
}
