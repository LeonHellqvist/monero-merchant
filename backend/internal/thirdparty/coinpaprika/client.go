package coinpaprika

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.coinpaprika.com"

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 10 * time.Second}}
}

type Price struct {
	Value     float64
	UpdatedAt time.Time
}

type tickerResponse struct {
	LastUpdated time.Time `json:"last_updated"`
	Quotes      map[string]struct {
		Price float64 `json:"price"`
	} `json:"quotes"`
}

// XMRPrice fetches the XMR price in the given currency
func (c *Client) XMRPrice(ctx context.Context, currency string) (Price, error) {
	endpoint := baseURL + "/v1/tickers/xmr-monero?quotes=" + currency
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Price{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Price{}, err
	}
	defer func() { io.Copy(io.Discard, resp.Body); resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return Price{}, fmt.Errorf("coinpaprika returned %s", resp.Status)
	}

	var ticker tickerResponse
	if err := json.NewDecoder(resp.Body).Decode(&ticker); err != nil {
		return Price{}, err
	}

	price := ticker.Quotes[currency].Price
	if price <= 0 {
		return Price{}, fmt.Errorf("coinpaprika returned no %s price", currency)
	}

	return Price{Value: price, UpdatedAt: ticker.LastUpdated}, nil
}
