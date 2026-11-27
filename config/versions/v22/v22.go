// Package v22 replaces GateIO's old default REST endpoints, which carried the API version prefix that GateIO now adds
// to a bare host.
package v22

import (
	"context"
	"encoding/json" //nolint:depguard // Used instead of gct encoding/json so that we can ensure consistent library functionality between versions
	"errors"
	"strings"

	"github.com/buger/jsonparser"
)

// Version is an ExchangeVersion for GateIO's default REST endpoints, which no longer carry the API version prefix
type Version struct{}

const apiVersionPrefix = "/api/v4"

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

// Exchanges returns every exchange; UpgradeExchange and DowngradeExchange pick out GateIO themselves, matching its name
// in any case as the engine does
func (*Version) Exchanges() []string { return []string{"*"} }

// UpgradeExchange removes the old default endpoints, so the new defaults apply. Custom endpoints are kept, as GateIO
// still uses an endpoint ending in a slash or with a path as the root of the versioned API, as earlier versions use every
// endpoint
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
		if url == ep.oldDefault && ep.oldDefault != ep.newDefault {
			e = jsonparser.Delete(e, "api", "urlEndpoints", ep.key)
		}
	}
	return e, nil
}

// DowngradeExchange removes the new default endpoints, so the old defaults apply. It adds the version prefix to a custom
// bare host and a final slash to a custom endpoint with a path, since earlier versions append each request's path to
// the endpoint as written
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
		switch {
		case isBareHost(url):
			url += apiVersionPrefix + "/"
		case strings.Contains(url, "://") && !strings.HasSuffix(url, "/"):
			url += "/"
		default:
			continue
		}
		if e, err = setEndpoint(e, ep.key, url); err != nil {
			return e, err
		}
	}
	return e, nil
}

// isBareHost reports whether an endpoint is a scheme and host, with no path or trailing slash, which GateIO now prefixes
// with the API version while earlier versions use any other endpoint as written. It is a local copy of Gate's check, so
// a later change to Gate's leaves this migration as it is
func isBareHost(endpoint string) bool {
	_, rest, hasScheme := strings.Cut(endpoint, "://")
	return hasScheme && !strings.Contains(rest, "/")
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
