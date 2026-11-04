// Package llm implements one OpenAI-compatible chat adapter.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
)

const (
	defaultBaseURL = "http://localhost:11434/v1"
	defaultModel   = "lfm2.5-1.2b-instruct"
	defaultTimeout = 60 * time.Second
)

// Client is one OpenAI-compatible chat client.
type Client struct {
	BaseURL    string
	Model      string
	APIKey     string
	JSONMode   bool
	HTTPClient *http.Client
	Timeout    time.Duration
}

// New returns a client with defaults.
func New(baseURL, model, apiKey string, jsonMode bool, timeout time.Duration) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	if strings.TrimSpace(model) == "" {
		model = defaultModel
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Model:      model,
		APIKey:     apiKey,
		JSONMode:   jsonMode,
		HTTPClient: &http.Client{Timeout: timeout},
		Timeout:    timeout,
	}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model          string    `json:"model"`
	Messages       []message `json:"messages"`
	Temperature    float64   `json:"temperature"`
	ResponseFormat any       `json:"response_format,omitempty"`
}

type response struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete sends one chat request and returns the message content.
func (c *Client) Complete(ctx context.Context, system, user string, schema json.RawMessage) (string, error) {
	payload := request{
		Model:       c.Model,
		Temperature: 0.1,
		Messages: []message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	if c.JSONMode {
		payload.ResponseFormat = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "analysis",
				"strict": true,
				"schema": schema,
			},
		}
	} else {
		payload.ResponseFormat = map[string]any{"type": "json_object"}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode llm request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if key := strings.TrimSpace(c.APIKey); key != "" && key != "not-needed" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := c.client().Do(req)
	if err != nil {
		if isTimeout(ctx, err) {
			return "", analysis.ErrLLMTimeout
		}
		return "", fmt.Errorf("call llm: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read llm response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm returned %d", resp.StatusCode)
	}
	var decoded response
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", fmt.Errorf("decode llm response: %w", err)
	}
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return "", errors.New("llm response has no content")
	}
	return decoded.Choices[0].Message.Content, nil
}

func (c *Client) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: c.Timeout}
}

func isTimeout(ctx context.Context, err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

var _ analysis.LLM = (*Client)(nil)
