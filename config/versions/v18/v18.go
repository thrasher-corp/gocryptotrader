// Package v18 adds subscriptions introduced for existing Deribit and OKX configurations.
package v18

import (
	"context"
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"errors"
	"reflect"
	"strings"

	"github.com/buger/jsonparser"
)

const (
	previousOkx           = `[{"enabled":true,"channel":"allTrades","asset":"all"},{"enabled":true,"channel":"orderbook","asset":"all"},{"enabled":true,"channel":"ticker","asset":"all"},{"enabled":true,"channel":"myOrders","asset":"all","authenticated":true},{"enabled":true,"channel":"myAccount","authenticated":true}]`
	previousDeribit       = `[{"enabled":true,"channel":"candles","asset":"all","interval":"24h"},{"enabled":true,"channel":"orderbook","asset":"all","interval":"100ms"},{"enabled":true,"channel":"ticker","asset":"all","interval":"100ms"},{"enabled":true,"channel":"allTrades","asset":"all","interval":"100ms"},{"enabled":true,"channel":"myOrders","asset":"all","interval":"100ms","authenticated":true},{"enabled":true,"channel":"myTrades","asset":"all","interval":"100ms","authenticated":true}]`
	deribit               = "Deribit"
	okx                   = "Okx"
	deribitAccount        = `{"enabled":true,"channel":"myAccount","authenticated":true}`
	okxOptionSummary      = `{"enabled":true,"channel":"opt-summary","asset":"options"}`
	okxBalanceAndPosition = `{"enabled":true,"channel":"balance_and_position","authenticated":true}`
	okxAccountGreeks      = `{"enabled":true,"channel":"account-greeks","authenticated":true}`
)

// Version implements ExchangeVersion for the subscriptions added in version 18.
type Version struct{}

// Exchanges returns the exchanges affected by this migration.
func (*Version) Exchanges() []string {
	// Match names locally so historical migrations retain their existing selection rules.
	return []string{"*"}
}

// UpgradeExchange extends recognisable previous defaults. Empty lists retain runtime
// fallback; customised defaults remain unchanged, including explicit disables.
// A skipped configuration still advances version; users must opt into new channels manually.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	name, err := jsonparser.GetString(exchange, "name")
	if err != nil {
		return exchange, err
	}
	var defaults []json.RawMessage
	var previousJSON string
	switch {
	case strings.EqualFold(name, deribit):
		previousJSON = previousDeribit
		defaults = []json.RawMessage{json.RawMessage(deribitAccount)}
	case strings.EqualFold(name, okx):
		previousJSON = previousOkx
		defaults = []json.RawMessage{
			json.RawMessage(okxOptionSummary),
			json.RawMessage(okxBalanceAndPosition),
			json.RawMessage(okxAccountGreeks),
		}
	default:
		return exchange, nil
	}

	subscriptionsJSON, valueType, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
		return exchange, nil
	case err != nil:
		return exchange, err
	case valueType != jsonparser.Array:
		return exchange, nil
	}

	var subscriptions []json.RawMessage
	if err := json.Unmarshal(subscriptionsJSON, &subscriptions); err != nil {
		return exchange, err
	}
	if len(subscriptions) == 0 {
		return exchange, nil
	}
	var previous, configured []map[string]any
	if err := json.Unmarshal([]byte(previousJSON), &previous); err != nil {
		return exchange, err
	}
	if err := json.Unmarshal(subscriptionsJSON, &configured); err != nil {
		return exchange, err
	}
	for _, entries := range [][]map[string]any{previous, configured} {
		for _, entry := range entries {
			if scope, ok := entry["asset"].(string); ok {
				entry["asset"] = strings.ToLower(scope)
			}
		}
	}
	for _, expected := range previous {
		matches := 0
		for _, entry := range configured {
			if entry["channel"] != expected["channel"] {
				continue
			}
			if !reflect.DeepEqual(entry, expected) {
				return exchange, nil
			}
			matches++
		}
		if matches != 1 {
			return exchange, nil
		}
	}
	for i := range defaults {
		channel, err := jsonparser.GetString(defaults[i], "channel")
		if err != nil {
			return exchange, err
		}
		scope, _ := jsonparser.GetString(defaults[i], "asset")
		covered := false
		for _, entry := range configured {
			if entry["channel"] != channel {
				continue
			}
			asset, _ := entry["asset"].(string)
			if scope == "" || asset == "" || asset == "all" || asset == scope {
				covered = true
				break
			}
		}
		if !covered {
			subscriptions = append(subscriptions, defaults[i])
		}
	}
	updated, err := json.Marshal(subscriptions)
	if err != nil {
		return exchange, err
	}
	return jsonparser.Set(exchange, updated, "features", "subscriptions")
}

// DowngradeExchange removes channels unavailable before version 18, including
// customised entries for those channels. Unrelated subscriptions remain intact.
// OKX account channels remain supported in version 17, even when added by this migration.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	name, err := jsonparser.GetString(exchange, "name")
	if err != nil {
		return exchange, err
	}
	if !strings.EqualFold(name, deribit) && !strings.EqualFold(name, okx) {
		return exchange, nil
	}
	raw, kind, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	if errors.Is(err, jsonparser.KeyPathNotFoundError) || kind != jsonparser.Array {
		return exchange, nil
	}
	if err != nil {
		return exchange, err
	}
	var subscriptions []json.RawMessage
	if err := json.Unmarshal(raw, &subscriptions); err != nil {
		return exchange, err
	}
	retained := make([]json.RawMessage, 0, len(subscriptions))
	for _, entry := range subscriptions {
		channel, err := jsonparser.GetString(entry, "channel")
		if err != nil {
			return exchange, err
		}
		if strings.EqualFold(name, deribit) && channel == "myAccount" ||
			strings.EqualFold(name, okx) && channel == "opt-summary" {
			continue
		}
		retained = append(retained, entry)
	}
	if len(retained) == len(subscriptions) {
		return exchange, nil
	}
	updated, err := json.Marshal(retained)
	if err != nil {
		return exchange, err
	}
	return jsonparser.Set(exchange, updated, "features", "subscriptions")
}
