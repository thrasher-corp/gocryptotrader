// Package v18 moves LBank's REST endpoint from its previous default host to the host LBank documents now.
package v18

import (
	"context"
	"errors"
	"strings"

	"github.com/buger/jsonparser"
)

// Version is an ExchangeVersion for LBank's endpoints
type Version struct{}

const (
	restSpotKey     = "RestSpotURL"
	previousRESTURL = "https://api.lbkex.com"
	currentRESTURL  = "https://api.lbank.info"
)

// currentDefaults are the endpoints this version defaults to; the previous version has no contract or websocket endpoint
var currentDefaults = map[string]string{
	restSpotKey:                  currentRESTURL,
	"RestUSDTMarginedFuturesURL": "https://lbkperp.lbank.com",
	"WebsocketSpotURL":           "wss://api.lbank.info/ws/V2/",
}

// Exchanges returns every exchange, as the version manager matches names exactly while the engine accepts LBank's in
// any case; UpgradeExchange and DowngradeExchange pick out LBank themselves
func (*Version) Exchanges() []string { return []string{"*"} }

// UpgradeExchange removes LBank's REST endpoint when it is the previous default, so the current default applies. The
// previous host still serves the same API, so an endpoint the user set to anything else is kept
func (*Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if !isLBank(e) {
		return e, nil
	}
	url, err := jsonparser.GetString(e, "api", "urlEndpoints", restSpotKey)
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
		return e, nil
	case err != nil:
		return e, err
	case url == previousRESTURL:
		e = jsonparser.Delete(e, "api", "urlEndpoints", restSpotKey)
	}
	return e, nil
}

// DowngradeExchange removes LBank's endpoints that are at this version's defaults, so the previous version's defaults
// apply: its REST host, and no contract or websocket endpoint, which it would warn it has no default for
func (*Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if !isLBank(e) {
		return e, nil
	}
	for key, def := range currentDefaults {
		url, err := jsonparser.GetString(e, "api", "urlEndpoints", key)
		switch {
		case errors.Is(err, jsonparser.KeyPathNotFoundError):
		case err != nil:
			return e, err
		case url == def:
			e = jsonparser.Delete(e, "api", "urlEndpoints", key)
		}
	}
	return e, nil
}

// isLBank reports whether an exchange config is LBank's, matching its name in any case as the engine does
func isLBank(e []byte) bool {
	name, err := jsonparser.GetString(e, "name")
	return err == nil && strings.EqualFold(name, "Lbank")
}
