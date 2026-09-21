// Package v16 adds subscriptions introduced for existing Deribit and OKX configurations.
package v16

import (
	"context"
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"errors"

	"github.com/buger/jsonparser"
)

const (
	deribit               = "Deribit"
	okx                   = "Okx"
	deribitAccount        = `{"enabled":true,"channel":"myAccount","authenticated":true}`
	okxOptionSummary      = `{"enabled":true,"channel":"opt-summary","asset":"options"}`
	okxBalanceAndPosition = `{"enabled":true,"channel":"balance_and_position","authenticated":true}`
	okxAccountGreeks      = `{"enabled":true,"channel":"account-greeks","authenticated":true}`
)

// Version implements ExchangeVersion for the subscriptions added in version 16.
type Version struct{}

// Exchanges returns the exchanges affected by this migration.
func (*Version) Exchanges() []string {
	return []string{deribit, okx}
}

// UpgradeExchange adds missing subscriptions while preserving existing custom entries.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	name, err := jsonparser.GetString(exchange, "name")
	if err != nil {
		return exchange, err
	}
	var defaults []json.RawMessage
	switch name {
	case deribit:
		defaults = []json.RawMessage{json.RawMessage(deribitAccount)}
	case okx:
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
	existing := make(map[string]bool, len(subscriptions))
	for i := range subscriptions {
		channel, err := jsonparser.GetString(subscriptions[i], "channel")
		if err == nil {
			existing[channel] = true
		}
	}
	for i := range defaults {
		channel, err := jsonparser.GetString(defaults[i], "channel")
		if err != nil {
			return exchange, err
		}
		if !existing[channel] {
			subscriptions = append(subscriptions, defaults[i])
		}
	}
	updated, err := json.Marshal(subscriptions)
	if err != nil {
		return exchange, err
	}
	return jsonparser.Set(exchange, updated, "features", "subscriptions")
}

// DowngradeExchange preserves subscriptions because custom and migrated entries cannot be distinguished safely.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return exchange, nil
}
