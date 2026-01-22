package queueserviceclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"queue-common/models"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) FetchNodesMetrics(ctx context.Context) (models.NodesMetricsResponse, []byte, error) {
	var out models.NodesMetricsResponse
	body, err := c.get(ctx, "/nodes/metrics")
	if err != nil {
		return out, nil, err
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, body, fmt.Errorf("decode /nodes/metrics response: %w", err)
	}
	return out, body, nil
}

func (c *Client) FetchResourcesMetrics(ctx context.Context) (models.ResourcesSessionMetricsResponse, []byte, error) {
	var out models.ResourcesSessionMetricsResponse
	body, err := c.get(ctx, "/resources/metrics")
	if err != nil {
		return out, nil, err
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, body, fmt.Errorf("decode /resources/metrics response: %w", err)
	}
	return out, body, nil
}

func (c *Client) get(ctx context.Context, p string) ([]byte, error) {
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid QUEUE_SERVICE_BASE_URL %q: %w", c.BaseURL, err)
	}
	u.Path = path.Join(strings.TrimRight(u.Path, "/"), p)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("GET %s: %s", u.String(), msg)
	}
	return b, nil
}
