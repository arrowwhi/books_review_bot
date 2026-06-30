package openlibrary

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

type Book struct {
	Key              string
	Title            string
	AuthorNames      []string
	FirstPublishYear int
	CoverID          int64
	CoverURL         string
}

type Client interface {
	Search(ctx context.Context, title string) ([]Book, error)
}

type client struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// Deprecated: use googlebooks.New instead.
func New(logger *zap.Logger) Client {
	return &client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

func (c *client) Search(ctx context.Context, title string) ([]Book, error) {
	apiURL := "https://openlibrary.org/search.json?title=" + url.QueryEscape(title) + "&limit=3"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("openlibrary search: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openlibrary search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Docs []struct {
			Key              string   `json:"key"`
			Title            string   `json:"title"`
			AuthorName       []string `json:"author_name"`
			FirstPublishYear int      `json:"first_publish_year"`
			CoverI           int64    `json:"cover_i"`
		} `json:"docs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openlibrary search: %w", err)
	}

	books := make([]Book, 0, len(result.Docs))
	for _, doc := range result.Docs {
		b := Book{
			Key:              doc.Key,
			Title:            doc.Title,
			AuthorNames:      doc.AuthorName,
			FirstPublishYear: doc.FirstPublishYear,
			CoverID:          doc.CoverI,
		}
		if doc.CoverI > 0 {
			b.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg", doc.CoverI)
		}
		books = append(books, b)
	}

	c.logger.Info("openlibrary search", zap.String("title", title), zap.Int("results", len(books)))

	return books, nil
}
