package frankfurter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type Rate struct {
	Date  string  `json:"date"`
	Quote string  `json:"quote"`
	Rate  float64 `json:"rate"`
}

// Rates fetches the latest rate for every currency Frankfurter has, relative to base
func (c *Client) Rates(ctx context.Context, base string) ([]Rate, error) {
	endpoint := c.baseURL + "/v2/rates?base=" + base
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { io.Copy(io.Discard, resp.Body); resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("frankfurter returned %s", resp.Status)
	}

	var rates []Rate
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		return nil, err
	}

	return rates, nil
}
