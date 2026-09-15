// Package v14 migrates GateIO's default spot orderbook websocket subscription to V2.
package v14

import (
	"context"
	"encoding/json" //nolint:depguard // Used instead of gct encoding/json so that we can ensure consistent library functionality between versions
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
)

const (
	legacyOrderbookChannel      = "orderbook"
	legacyOrderbookAliasChannel = "spot.order_book_update"
	spotAsset                   = "spot"
	allAsset                    = "all"
	spotOrderbookV2Channel      = "spot.obu"
)

// Version implements ExchangeVersion for GateIO's spot orderbook subscription migration.
type Version struct{}

// Exchanges returns just GateIO.
func (*Version) Exchanges() []string { return []string{"GateIO"} }

// UpgradeExchange replaces the previous default spot orderbook subscription with V2.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return migrateSubscriptions(exchange, true)
}

// DowngradeExchange restores the previous default spot orderbook subscription.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return migrateSubscriptions(exchange, false)
}

func migrateSubscriptions(exchange []byte, upgrade bool) ([]byte, error) {
	raw, _, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	if err != nil {
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return exchange, nil
		}
		return exchange, fmt.Errorf("error getting GateIO subscriptions: %w", err)
	}

	var subscriptions []struct {
		Enabled       *bool           `json:"enabled"`
		Channel       string          `json:"channel"`
		Asset         string          `json:"asset"`
		Interval      json.RawMessage `json:"interval"`
		Levels        int             `json:"levels"`
		Pairs         string          `json:"pairs"`
		Authenticated bool            `json:"authenticated"`
	}
	if err := json.Unmarshal(raw, &subscriptions); err != nil {
		return exchange, fmt.Errorf("error decoding GateIO subscriptions: %w", err)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return exchange, fmt.Errorf("error decoding GateIO subscription entries: %w", err)
	}

	legacyIndex, v2Index := -1, -1
	for i := range subscriptions {
		isSpot := strings.EqualFold(subscriptions[i].Asset, spotAsset)
		isAll := strings.EqualFold(subscriptions[i].Asset, allAsset)
		if ((subscriptions[i].Channel == legacyOrderbookAliasChannel && isSpot) ||
			((subscriptions[i].Channel == legacyOrderbookAliasChannel ||
				subscriptions[i].Channel == legacyOrderbookChannel ||
				subscriptions[i].Channel == spotOrderbookV2Channel) && isAll)) &&
			subscriptions[i].Enabled != nil && *subscriptions[i].Enabled {
			return exchange, nil
		}
		if !isSpot {
			continue
		}
		switch subscriptions[i].Channel {
		case legacyOrderbookChannel:
			if legacyIndex != -1 {
				return exchange, nil
			}
			legacyIndex = i
		case spotOrderbookV2Channel:
			if v2Index != -1 {
				return exchange, nil
			}
			v2Index = i
		}
	}
	if legacyIndex == -1 ||
		string(subscriptions[legacyIndex].Interval) != `"100ms"` || subscriptions[legacyIndex].Pairs != "" ||
		subscriptions[legacyIndex].Authenticated ||
		subscriptions[legacyIndex].Enabled == nil ||
		*subscriptions[legacyIndex].Enabled != upgrade {
		return exchange, nil
	}
	if v2Index == -1 {
		if !upgrade {
			return exchange, nil
		}
		v2Index = len(entries)
		entries = append(entries, json.RawMessage(`{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}`))
		updated, err := json.Marshal(entries)
		if err != nil {
			return exchange, fmt.Errorf("error encoding GateIO subscription entries: %w", err)
		}
		exchange, err = jsonparser.Set(exchange, updated, "features", "subscriptions")
		if err != nil {
			return exchange, fmt.Errorf("error adding GateIO V2 spot orderbook subscription: %w", err)
		}
	} else {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(entries[v2Index], &fields); err != nil {
			return exchange, fmt.Errorf("error decoding GateIO V2 subscription: %w", err)
		}
		if len(fields) != 4 || subscriptions[v2Index].Levels != 50 ||
			subscriptions[v2Index].Enabled == nil || *subscriptions[v2Index].Enabled == upgrade {
			return exchange, nil
		}
	}

	exchange, err = jsonparser.Set(exchange, []byte(strconv.FormatBool(!upgrade)), "features", "subscriptions", "["+strconv.Itoa(legacyIndex)+"]", "enabled")
	if err != nil {
		return exchange, fmt.Errorf("error setting GateIO legacy spot orderbook subscription: %w", err)
	}
	exchange, err = jsonparser.Set(exchange, []byte(strconv.FormatBool(upgrade)), "features", "subscriptions", "["+strconv.Itoa(v2Index)+"]", "enabled")
	if err != nil {
		return exchange, fmt.Errorf("error setting GateIO V2 spot orderbook subscription: %w", err)
	}
	return exchange, nil
}
