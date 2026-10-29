package rates

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/monero-merchant/monero-merchant/backend/internal/thirdparty/coinpaprika"
	"github.com/monero-merchant/monero-merchant/backend/internal/thirdparty/frankfurter"
)

// CoinPaprika prices XMR in this currency and Frankfurter converts from it
const baseCurrency = "EUR"

const (
	xmrTTL       = 5 * time.Minute // CoinPaprika updates about every 10 minutes
	fiatTTL      = 12 * time.Hour  // Frankfurter publishes once a day
	retryAfter   = time.Minute
	fetchTimeout = 5 * time.Second
	staleAfter   = 30 * time.Minute
)

var (
	ErrUnsupportedCurrency = errors.New("unsupported currency")
	ErrUnavailable         = errors.New("exchange rates not available yet")
)

type Rate struct {
	Rate float64 `json:"rate"`
	Date string  `json:"date"`
}

type Rates struct {
	Base         string          `json:"base"`
	Sources      []string        `json:"sources"`
	XMRUpdatedAt time.Time       `json:"xmr_updated_at"`
	Stale        bool            `json:"stale"`
	Rates        map[string]Rate `json:"rates"`
}

type RatesService struct {
	xmr  cache[coinpaprika.Price]
	fiat cache[map[string]Rate]
}

func NewRatesService(coinPaprika *coinpaprika.Client, frankfurterClient *frankfurter.Client) *RatesService {
	s := &RatesService{}
	s.xmr = cache[coinpaprika.Price]{name: "XMR price", ttl: xmrTTL, fetch: func(ctx context.Context) (coinpaprika.Price, error) {
		return coinPaprika.XMRPrice(ctx, baseCurrency)
	}}
	s.fiat = cache[map[string]Rate]{name: "fiat rates", ttl: fiatTTL, fetch: func(ctx context.Context) (map[string]Rate, error) {
		rates, err := frankfurterClient.Rates(ctx, baseCurrency)
		if err != nil {
			return nil, err
		}
		table := fiatTable(rates)
		if len(table) == 0 {
			return nil, errors.New("frankfurter returned no supported currencies")
		}
		return table, nil
	}}
	return s
}

// ParseCurrencies parses a comma separated list like "eur,CHF"
func ParseCurrencies(param string) ([]string, error) {
	var currencies []string
	for _, c := range strings.Split(param, ",") {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c == "" {
			continue
		}
		if !supported[c] {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedCurrency, c)
		}
		currencies = append(currencies, c)
	}
	if len(currencies) == 0 {
		return nil, fmt.Errorf("%w: none given", ErrUnsupportedCurrency)
	}
	return currencies, nil
}

func (s *RatesService) GetRates(ctx context.Context, currencies []string) (*Rates, error) {
	price, ok := s.xmr.get(ctx)
	if !ok {
		return nil, ErrUnavailable
	}

	resp := &Rates{
		Base:         "XMR",
		Sources:      []string{"CoinPaprika"},
		XMRUpdatedAt: price.UpdatedAt,
		Stale:        time.Since(price.UpdatedAt) > staleAfter,
		Rates:        make(map[string]Rate, len(currencies)),
	}

	var fiat map[string]Rate
	for _, c := range currencies {
		if c == baseCurrency {
			resp.Rates[c] = Rate{Rate: roundPrice(price.Value), Date: price.UpdatedAt.UTC().Format(time.DateOnly)}
			continue
		}
		if fiat == nil {
			if fiat, ok = s.fiat.get(ctx); !ok {
				return nil, ErrUnavailable
			}
			resp.Sources = append(resp.Sources, "Frankfurter")
		}
		rate, ok := fiat[c]
		if !ok {
			return nil, fmt.Errorf("%w: no rate for %s", ErrUnavailable, c)
		}
		resp.Rates[c] = Rate{Rate: roundPrice(price.Value * rate.Rate), Date: rate.Date}
	}

	return resp, nil
}

// Round to 2 decimals
func roundPrice(v float64) float64 {
	return math.Round(v*100) / 100
}

func fiatTable(rates []frankfurter.Rate) map[string]Rate {
	table := make(map[string]Rate)
	for _, r := range rates {
		if supported[r.Quote] && r.Rate > 0 {
			table[r.Quote] = Rate{Rate: r.Rate, Date: r.Date}
		}
	}
	for _, c := range SupportedCurrencies {
		if _, ok := table[c]; !ok && c != baseCurrency {
			log.Printf("Frankfurter returned no rate for %s", c)
		}
	}
	return table
}

// cache keeps the last fetched value. Requests that come in during a fetch wait for it,
// and after a failed fetch the old value is used for a minute before trying again.
type cache[T any] struct {
	name  string
	ttl   time.Duration
	fetch func(context.Context) (T, error)

	mu        sync.Mutex
	value     T
	ok        bool
	fetchedAt time.Time
	failedAt  time.Time
}

func (c *cache[T]) get(ctx context.Context) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ok && time.Since(c.fetchedAt) < c.ttl {
		return c.value, true
	}
	if time.Since(c.failedAt) < retryAfter {
		return c.value, c.ok
	}

	// Don't cancel the fetch if this request goes away, others may be waiting for it
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), fetchTimeout)
	defer cancel()

	value, err := c.fetch(ctx)
	if err != nil {
		log.Printf("Failed to fetch %s: %v", c.name, err)
		c.failedAt = time.Now()
		return c.value, c.ok
	}

	c.value, c.ok = value, true
	c.fetchedAt = time.Now()
	return c.value, true
}
