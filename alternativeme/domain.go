package alternativeme

import (
	"context"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes alternativeme as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/alternativeme-cli/alternativeme"
//
// The init below registers it; the host then routes alternativeme:// URIs to
// the operations Register installs. The same Domain also builds the standalone
// alternativeme binary, so the binary and a host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the alternativeme driver. It carries no state; the per-run client
// is built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme and identity for both the binary and the URI driver.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "alternativeme",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "alternativeme",
			Short:  "Crypto Fear & Greed Index from alternative.me.",
			Long: `alternativeme reads the Crypto Fear & Greed Index from alternative.me.

It returns a value from 0 (Extreme Fear) to 100 (Extreme Greed) for each day
requested. No API key required.`,
			Site: "alternative.me",
			Repo: "https://github.com/tamnd/alternativeme-cli",
		},
	}
}

// Register installs the client factory and operations onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "feargreed",
		Group:   "read",
		List:    true,
		Summary: "Fetch the Crypto Fear & Greed Index",
		URIType: "index",
	}, getFearGreed)
}

// newClient builds the Client from kit.Config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClientFromConfig(c), nil
}

// feargreedInput carries the flags for the feargreed op.
type feargreedInput struct {
	Limit  int     `kit:"flag,inherit" help:"number of days (default 1)"`
	Client *Client `kit:"inject"`
}

// getFearGreed is the handler for the feargreed op.
func getFearGreed(ctx context.Context, in feargreedInput, emit func(*FearGreedRecord) error) error {
	limit := in.Limit
	if limit < 1 {
		limit = 1
	}
	records, err := in.Client.GetFearGreed(ctx, limit)
	if err != nil {
		return err
	}
	for i := range records {
		if err := emit(&records[i]); err != nil {
			return err
		}
	}
	return nil
}

// Classify turns any accepted input into (uriType, id).
// "feargreed" or "index" maps to ("index", "feargreed").
// A bare date like "2024-01-01" maps to ("date", "2024-01-01").
func (Domain) Classify(input string) (uriType, id string, err error) {
	switch input {
	case "feargreed", "index":
		return "index", "feargreed", nil
	}
	// accept bare YYYY-MM-DD dates
	if len(input) == 10 && input[4] == '-' && input[7] == '-' {
		return "date", input, nil
	}
	return "", "", errs.Usage("unrecognized alternativeme reference: %q", input)
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "index":
		return "https://alternative.me/crypto/fear-and-greed-index/", nil
	case "date":
		return "https://alternative.me/crypto/fear-and-greed-index/", nil
	}
	return "", errs.Usage("alternativeme has no resource type %q", uriType)
}
