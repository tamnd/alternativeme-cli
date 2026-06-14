package alternativeme_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/alternativeme-cli/alternativeme"
)

const mockFearGreedResponse = `{
  "name": "Fear and Greed Index",
  "data": [
    {
      "value": "18",
      "value_classification": "Extreme Fear",
      "timestamp": "1749859200",
      "time_until_update": "43200"
    }
  ],
  "metadata": {"error": null}
}`

const mockFearGreedMultiResponse = `{
  "name": "Fear and Greed Index",
  "data": [
    {"value": "18", "value_classification": "Extreme Fear",  "timestamp": "1749859200", "time_until_update": ""},
    {"value": "22", "value_classification": "Extreme Fear",  "timestamp": "1749772800", "time_until_update": ""},
    {"value": "35", "value_classification": "Fear",          "timestamp": "1749686400", "time_until_update": ""},
    {"value": "51", "value_classification": "Greed",         "timestamp": "1749600000", "time_until_update": ""},
    {"value": "76", "value_classification": "Extreme Greed", "timestamp": "1749513600", "time_until_update": ""}
  ],
  "metadata": {"error": null}
}`

func newTestClient(srv *httptest.Server) *alternativeme.Client {
	cfg := alternativeme.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 3
	cfg.Timeout = 5 * time.Second
	return alternativeme.NewClientFromConfig(cfg)
}

func TestGetFearGreed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockFearGreedResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	records, err := c.GetFearGreed(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetFearGreed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	r := records[0]
	if r.Value != 18 {
		t.Errorf("Value = %d, want 18", r.Value)
	}
	if r.Classification != "Extreme Fear" {
		t.Errorf("Classification = %q, want Extreme Fear", r.Classification)
	}
	// timestamp "1749859200" → 2025-06-14
	if !strings.HasPrefix(r.Timestamp, "20") || len(r.Timestamp) != 10 {
		t.Errorf("Timestamp = %q, want YYYY-MM-DD format", r.Timestamp)
	}
}

func TestGetFearGreedLimit(t *testing.T) {
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(mockFearGreedMultiResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	records, err := c.GetFearGreed(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetFearGreed: %v", err)
	}
	if len(records) != 5 {
		t.Errorf("len(records) = %d, want 5", len(records))
	}
	if !strings.Contains(gotURL, "limit=5") {
		t.Errorf("URL %q does not contain limit=5", gotURL)
	}
}

func TestRetry503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(mockFearGreedResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	start := time.Now()
	records, err := c.GetFearGreed(context.Background(), 1)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetFearGreed after retries: %v", err)
	}
	if len(records) == 0 {
		t.Error("expected records after retry")
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if elapsed < 500*time.Millisecond {
		t.Errorf("retries did not back off: elapsed %v < 500ms", elapsed)
	}
}

func TestUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(mockFearGreedResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, _ = c.GetFearGreed(context.Background(), 1)
	if gotUA == "" {
		t.Error("request carried no User-Agent")
	}
}
