// Package alternativeme is the library behind the alternativeme command line:
// the HTTP client, request shaping, and the typed data models for the
// Alternative.me Crypto Fear & Greed Index.
//
// The Client sets a real User-Agent, paces requests to stay polite, and
// retries transient failures (429 and 5xx). Build your endpoint calls and
// JSON decoding on top of it.
package alternativeme

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Host is the API hostname.
const Host = "api.alternative.me"

// BaseURL is the root every request is built from.
const BaseURL = "https://api.alternative.me"

// Config holds all tunables for a Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   BaseURL,
		UserAgent: "tamnd-alternativeme-cli/0.1 (tamnd87@gmail.com)",
		Rate:      200 * time.Millisecond,
		Retries:   3,
		Timeout:   10 * time.Second,
	}
}

// Client talks to alternative.me over HTTP.
type Client struct {
	HTTP      *http.Client
	cfg       Config
	last      time.Time
}

// NewClient returns a Client configured with DefaultConfig.
func NewClient() *Client {
	cfg := DefaultConfig()
	return &Client{
		HTTP: &http.Client{Timeout: cfg.Timeout},
		cfg:  cfg,
	}
}

// NewClientFromConfig builds a Client from the given Config.
func NewClientFromConfig(cfg Config) *Client {
	return &Client{
		HTTP: &http.Client{Timeout: cfg.Timeout},
		cfg:  cfg,
	}
}

// FearGreedRecord is one day's Fear & Greed reading.
type FearGreedRecord struct {
	Timestamp      string `json:"timestamp" kit:"id"`
	Value          int    `json:"value"`
	Classification string `json:"classification"`
}

// wire types — match the JSON the API actually returns.

type wireResponse struct {
	Name     string       `json:"name"`
	Data     []wireRecord `json:"data"`
	Metadata wireMeta     `json:"metadata"`
}

type wireRecord struct {
	Value               string `json:"value"`
	ValueClassification string `json:"value_classification"`
	Timestamp           string `json:"timestamp"`
	TimeUntilUpdate     string `json:"time_until_update"`
}

type wireMeta struct {
	Error *string `json:"error"`
}

// GetFearGreed fetches limit days of the Crypto Fear & Greed Index.
// Limit 1 returns today's reading only. The timestamp is converted from a
// Unix second string to "2006-01-02" format; value is converted from a JSON
// string ("18") to int.
func (c *Client) GetFearGreed(ctx context.Context, limit int) ([]FearGreedRecord, error) {
	if limit < 1 {
		limit = 1
	}
	url := fmt.Sprintf("%s/fng/?limit=%d", c.cfg.BaseURL, limit)
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}

	var wire wireResponse
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode fear greed response: %w", err)
	}
	if wire.Metadata.Error != nil && *wire.Metadata.Error != "" {
		return nil, fmt.Errorf("api error: %s", *wire.Metadata.Error)
	}

	out := make([]FearGreedRecord, 0, len(wire.Data))
	for _, w := range wire.Data {
		v, err := strconv.Atoi(w.Value)
		if err != nil {
			return nil, fmt.Errorf("parse value %q: %w", w.Value, err)
		}
		ts, err := strconv.ParseInt(w.Timestamp, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp %q: %w", w.Timestamp, err)
		}
		date := time.Unix(ts, 0).UTC().Format("2006-01-02")
		out = append(out, FearGreedRecord{
			Timestamp:      date,
			Value:          v,
			Classification: w.ValueClassification,
		})
	}
	return out, nil
}

// get fetches url and returns the response body, with pacing and retries.
func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the last request.
func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
