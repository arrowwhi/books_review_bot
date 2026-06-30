package googlebooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/client/openlibrary"
)

type client struct {
	httpClient *http.Client
	logger     *zap.Logger
}

func New(logger *zap.Logger) openlibrary.Client {
	return &client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

func (c *client) Search(ctx context.Context, title string) ([]openlibrary.Book, error) {
	apiURL := "https://www.googleapis.com/books/v1/volumes?q=intitle:" + url.QueryEscape(title) + "&maxResults=3"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("googlebooks search: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("googlebooks search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID         string `json:"id"`
			VolumeInfo struct {
				Title      string   `json:"title"`
				Authors    []string `json:"authors"`
				ImageLinks struct {
					Thumbnail string `json:"thumbnail"`
				} `json:"imageLinks"`
			} `json:"volumeInfo"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("googlebooks search: %w", err)
	}

	books := make([]openlibrary.Book, 0, len(result.Items))
	for _, item := range result.Items {
		b := openlibrary.Book{
			Key:         item.ID,
			Title:       item.VolumeInfo.Title,
			AuthorNames: item.VolumeInfo.Authors,
			CoverURL:    strings.Replace(item.VolumeInfo.ImageLinks.Thumbnail, "http://", "https://", 1),
		}
		books = append(books, b)
	}

	c.logger.Info("googlebooks search", zap.String("title", title), zap.Int("results", len(books)))

	return books, nil
}
