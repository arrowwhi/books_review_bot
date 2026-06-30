package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Client interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

type client struct {
	httpClient *http.Client
	apiKey     string
	logger     *zap.Logger
}

func New(apiKey string, logger *zap.Logger) Client {
	return &client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
		logger:     logger,
	}
}

func (c *client) Complete(ctx context.Context, prompt string) (string, error) {
	if c.apiKey == "" {
		return "", errors.New("anthropic api key is empty")
	}

	reqBody := struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		Messages  []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}{
		Model:     "claude-sonnet-4-6",
		MaxTokens: 1024,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "user", Content: prompt},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("claude complete: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("claude complete: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude complete: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("claude complete: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude api error %d: %s", resp.StatusCode, body)
	}

	var respBody struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &respBody); err != nil {
		return "", fmt.Errorf("claude complete: %w", err)
	}

	c.logger.Info("claude complete", zap.Int("prompt_len", len(prompt)))

	return respBody.Content[0].Text, nil
}
