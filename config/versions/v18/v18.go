// Package v18 moves GateIO's REST API version prefix out of its configured endpoints and into its request paths.
package v18

import (
	"context"
	"encoding/json" //nolint:depguard // Used instead of gct encoding/json so that we can ensure consistent library functionality between versions
	"errors"
	"strings"

	"github.com/buger/jsonparser"
)

// Version is an ExchangeVersion for GateIO REST endpoints, which no longer carry the API version prefix
type Version struct{}

const (
	apiVersionPrefix = "/api/v4"
	// nonDefaultPlaceholder marks an endpoint the user has not configured, which the exchange skips
	nonDefaultPlaceholder = "NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"
)

// endpoint is a GateIO REST endpoint key with its default before and after this version
type endpoint struct {
	key        string
	oldDefault string
	newDefault string
}

// endpoints lists the REST endpoints GateIO prefixes with the API version. The testnet default never carried the
// prefix and is unchanged, so only a custom testnet endpoint changes.
var endpoints = []endpoint{
	{key: "RestSpotURL", oldDefault: "https://api.gateio.ws/api/v4/", newDefault: "https://api.gateio.ws"},
	{key: "RestFuturesURL", oldDefault: "https://fx-api.gateio.ws/api/v4/", newDefault: "https://fx-api.gateio.ws"},
	{key: "RestSpotSupplementaryURL", oldDefault: "https://fx-api-testnet.gateio.ws", newDefault: "https://fx-api-testnet.gateio.ws"},
}

// Exchanges returns every exchange, as the version manager matches names exactly while the engine accepts GateIO's in
// any case; UpgradeExchange and DowngradeExchange pick out GateIO themselves
func (*Version) Exchanges() []string { return []string{"*"} }

// UpgradeExchange removes the old default endpoints, so the new defaults apply, and strips the version prefix from
// custom endpoints, since GateIO now adds it to every request path
func (*Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if !isGateIO(e) {
		return e, nil
	}
	for _, ep := range endpoints {
		url, err := jsonparser.GetString(e, "api", "urlEndpoints", ep.key)
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			continue
		}
		if err != nil {
			return e, err
		}
		if url == ep.oldDefault {
			if ep.oldDefault != ep.newDefault {
				e = jsonparser.Delete(e, "api", "urlEndpoints", ep.key)
			}
			continue
		}
		if base := strings.TrimSuffix(url, "/"); strings.HasSuffix(base, apiVersionPrefix) {
			if e, err = setEndpoint(e, ep.key, strings.TrimSuffix(base, apiVersionPrefix)); err != nil {
				return e, err
			}
		}
	}
	return e, nil
}

// DowngradeExchange removes the new default endpoints, so the old defaults apply, and restores the version prefix to
// custom endpoints, since earlier versions expect each endpoint to carry it
func (*Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if !isGateIO(e) {
		return e, nil
	}
	for _, ep := range endpoints {
		url, err := jsonparser.GetString(e, "api", "urlEndpoints", ep.key)
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			continue
		}
		if err != nil {
			return e, err
		}
		if url == ep.newDefault {
			if ep.oldDefault != ep.newDefault {
				e = jsonparser.Delete(e, "api", "urlEndpoints", ep.key)
			}
			continue
		}
		if url == "" || url == nonDefaultPlaceholder || strings.HasSuffix(strings.TrimSuffix(url, "/"), apiVersionPrefix) {
			continue
		}
		if e, err = setEndpoint(e, ep.key, strings.TrimSuffix(url, "/")+apiVersionPrefix+"/"); err != nil {
			return e, err
		}
	}
	return e, nil
}

// isGateIO reports whether an exchange config is GateIO's, matching its name in any case
func isGateIO(e []byte) bool {
	name, err := jsonparser.GetString(e, "name")
	return err == nil && strings.EqualFold(name, "GateIO")
}

func setEndpoint(e []byte, key, url string) ([]byte, error) {
	v, err := json.Marshal(url)
	if err != nil {
		return e, err
	}
	return jsonparser.Set(e, v, "api", "urlEndpoints", key)
}
