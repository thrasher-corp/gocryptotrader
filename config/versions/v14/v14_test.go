package v14_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v14 "github.com/thrasher-corp/gocryptotrader/config/versions/v14"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"GateIO"}, new(v14.Version).Exchanges())
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	version := new(v14.Version)

	tests := []struct {
		name string
		in   string
		exp  string
	}{
		{
			name: "previous defaults",
			in:   `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
			exp:  `{"name":"GateIO","features":{"subscriptions":[{"enabled":false,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
		},
		{
			name: "custom selection",
			in:   `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
			exp:  `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
		},
		{
			name: "numeric candle interval",
			in:   `{"features":{"subscriptions":[{"enabled":true,"channel":"candles","asset":"spot","interval":300000000000},{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
			exp:  `{"features":{"subscriptions":[{"enabled":true,"channel":"candles","asset":"spot","interval":300000000000},{"enabled":false,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
		},
		{
			name: "numeric orderbook interval",
			in:   `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":100000000},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
			exp:  `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":100000000},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
		},
		{
			name: "missing V2 entry preserves unrelated fields",
			in:   `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms","custom":"retained"},{"enabled":true,"channel":"candles","asset":"spot","interval":300000000000}]}}`,
			exp:  `{"features":{"subscriptions":[{"enabled":false,"channel":"orderbook","asset":"spot","interval":"100ms","custom":"retained"},{"enabled":true,"channel":"candles","asset":"spot","interval":300000000000},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
		},
		{
			name: "custom settings",
			in:   `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"10ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":20}]}}`,
			exp:  `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"10ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":20}]}}`,
		},
		{
			name: "missing subscriptions",
			in:   `{"name":"GateIO"}`,
			exp:  `{"name":"GateIO"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := version.UpgradeExchange(t.Context(), []byte(tc.in))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, tc.exp, string(got), "UpgradeExchange should migrate only previous defaults")
			again, err := version.UpgradeExchange(t.Context(), got)
			require.NoError(t, err, "repeated UpgradeExchange must not error")
			assert.Equal(t, got, again, "UpgradeExchange should be idempotent")
		})
	}
}

func TestMigrationPreservesCustomSubscriptions(t *testing.T) {
	t.Parallel()
	for name, input := range map[string]string{
		"legacy pairs":                `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms","pairs":"BTC_USDT"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
		"V2 pairs":                    `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50,"pairs":"ETH_USDT"}]}}`,
		"different pairs":             `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms","pairs":"BTC_USDT"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50,"pairs":"ETH_USDT"}]}}`,
		"legacy pairs without V2":     `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms","pairs":"BTC_USDT"}]}}`,
		"numeric interval without V2": `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":100000000}]}}`,
		"disabled legacy without V2":  `{"features":{"subscriptions":[{"enabled":false,"channel":"orderbook","asset":"spot","interval":"100ms"}]}}`,
		"downgrade restricted legacy": `{"features":{"subscriptions":[{"enabled":false,"channel":"orderbook","asset":"spot","interval":"100ms","pairs":"BTC_USDT"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
		"downgrade restricted V2":     `{"features":{"subscriptions":[{"enabled":false,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50,"pairs":"ETH_USDT"}]}}`,
		"duplicate missing enabled":   `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"channel":"spot.obu","asset":"spot","levels":50},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
		"legacy missing enabled":      `{"features":{"subscriptions":[{"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			version := new(v14.Version)
			got, err := version.UpgradeExchange(t.Context(), []byte(input))
			require.NoError(t, err, "UpgradeExchange must accept custom subscriptions")
			assert.Equal(t, input, string(got), "UpgradeExchange should preserve custom subscriptions byte-for-byte")
			got, err = version.DowngradeExchange(t.Context(), []byte(input))
			require.NoError(t, err, "DowngradeExchange must accept custom subscriptions")
			assert.Equal(t, input, string(got), "DowngradeExchange should preserve custom subscriptions byte-for-byte")
		})
	}
}

func TestMigrationPreservesLegacyAuthentication(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		field         string
		authenticated bool
	}{
		{name: "authenticated", field: `,"authenticated":true`, authenticated: true},
		{name: "unauthenticated", field: `,"authenticated":false`},
		{name: "omitted"},
		{name: "null", field: `,"authenticated":null`},
	} {
		for _, upgrade := range []bool{true, false} {
			for _, hasV2 := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/upgrade=%t/V2=%t", test.name, upgrade, hasV2), func(t *testing.T) {
					t.Parallel()
					v2 := ""
					if hasV2 {
						v2 = fmt.Sprintf(`,{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50}`, !upgrade)
					}
					input := fmt.Sprintf(`{"features":{"subscriptions":[{"enabled":%t,"channel":"orderbook","asset":"spot","interval":"100ms"%s}%s,{"enabled":true,"channel":"spot.order_book","asset":"spot"}]}}`, upgrade, test.field, v2)
					version := new(v14.Version)
					migrate := version.UpgradeExchange
					if !upgrade {
						migrate = version.DowngradeExchange
					}
					got, err := migrate(t.Context(), []byte(input))
					require.NoError(t, err, "migration must accept legacy authentication settings")
					if test.authenticated || (!upgrade && !hasV2) {
						assert.Equal(t, input, string(got), "ineligible migrations should preserve the input byte-for-byte")
					} else {
						legacy := fmt.Sprintf(`{"enabled":%t,"channel":"orderbook","asset":"spot","interval":"100ms"%s}`, !upgrade, test.field)
						v2 = fmt.Sprintf(`{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50}`, upgrade)
						snapshot := `{"enabled":true,"channel":"spot.order_book","asset":"spot"}`
						entries := legacy + "," + v2 + "," + snapshot
						if !hasV2 {
							entries = legacy + "," + snapshot + "," + v2
						}
						assert.JSONEq(t, `{"features":{"subscriptions":[`+entries+`]}}`, string(got), "unauthenticated legacy entries should allow migration without changing other fields")
					}
					again, err := migrate(t.Context(), got)
					require.NoError(t, err, "repeated migration must not error")
					assert.Equal(t, got, again, "migration should be idempotent")
				})
			}
		}
	}
}

func TestMigrationRequiresDefaultV2Entry(t *testing.T) {
	t.Parallel()
	for name, entry := range map[string]string{
		"authenticated":   `{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50,"authenticated":true}`,
		"explicit false":  `{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50,"authenticated":false}`,
		"custom interval": `{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50,"interval":"20ms"}`,
		"unknown field":   `{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50,"custom":"retained"}`,
		"empty pairs":     `{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50,"pairs":""}`,
		"null enabled":    `{"enabled":null,"channel":"spot.obu","asset":"spot","levels":50}`,
		"missing enabled": `{"channel":"spot.obu","asset":"spot","levels":50}`,
	} {
		for _, upgrade := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/upgrade=%t", name, upgrade), func(t *testing.T) {
				t.Parallel()
				v2 := strings.ReplaceAll(entry, "%t", strconv.FormatBool(!upgrade))
				input := fmt.Sprintf(`{"features":{"subscriptions":[{"enabled":%t,"channel":"orderbook","asset":"spot","interval":"100ms"},%s]}}`, upgrade, v2)
				version := new(v14.Version)
				migrate := version.UpgradeExchange
				if !upgrade {
					migrate = version.DowngradeExchange
				}
				got, err := migrate(t.Context(), []byte(input))
				require.NoError(t, err, "migration must accept customised V2 entries")
				assert.Equal(t, input, string(got), "migration should preserve customised V2 entries byte-for-byte")
			})
		}
	}
}

func TestMigrationPreservesRawLegacyChannel(t *testing.T) {
	t.Parallel()
	const (
		rawLegacyChannel = "spot.order_book_update"
		allAssets        = "all"
		enabledTrue      = `"enabled":true,`
	)
	for _, upgrade := range []bool{true, false} {
		for _, test := range []struct {
			name          string
			channel       string
			asset         string
			enabledField  string
			wantMigration bool
		}{
			{name: "enabled", channel: rawLegacyChannel, asset: "spot", enabledField: enabledTrue},
			{name: "disabled", channel: rawLegacyChannel, asset: "spot", enabledField: `"enabled":false,`, wantMigration: true},
			{name: "missing enabled", channel: rawLegacyChannel, asset: "spot", wantMigration: true},
			{name: "null enabled", channel: rawLegacyChannel, asset: "spot", enabledField: `"enabled":null,`, wantMigration: true},
			{name: "all assets enabled", channel: rawLegacyChannel, asset: allAssets, enabledField: enabledTrue},
			{name: "all assets disabled", channel: rawLegacyChannel, asset: allAssets, enabledField: `"enabled":false,`, wantMigration: true},
			{name: "all assets missing enabled", channel: rawLegacyChannel, asset: allAssets, wantMigration: true},
			{name: "all assets null enabled", channel: rawLegacyChannel, asset: allAssets, enabledField: `"enabled":null,`, wantMigration: true},
			{name: "generic all assets enabled", channel: "orderbook", asset: allAssets, enabledField: enabledTrue},
			{name: "generic all assets disabled", channel: "orderbook", asset: allAssets, enabledField: `"enabled":false,`, wantMigration: true},
			{name: "generic all assets missing enabled", channel: "orderbook", asset: allAssets, wantMigration: true},
			{name: "generic all assets null enabled", channel: "orderbook", asset: allAssets, enabledField: `"enabled":null,`, wantMigration: true},
			{name: "uppercase spot enabled", channel: rawLegacyChannel, asset: "SPOT", enabledField: enabledTrue},
			{name: "uppercase all assets enabled", channel: "orderbook", asset: "ALL", enabledField: enabledTrue},
			{name: "all assets V2 enabled", channel: "spot.obu", asset: allAssets, enabledField: enabledTrue},
		} {
			t.Run(fmt.Sprintf("upgrade=%t/%s", upgrade, test.name), func(t *testing.T) {
				t.Parallel()
				settings := `"interval":"100ms","pairs":"ETH_USDT"`
				if test.channel == "spot.obu" {
					settings = `"levels":50,"pairs":"ETH_USDT"`
				}
				raw := `{` + test.enabledField + `"channel":"` + test.channel + `","asset":"` + test.asset + `",` + settings + `}`
				input := fmt.Sprintf(`{"features":{"subscriptions":[{"enabled":%t,"channel":"orderbook","asset":"spot","interval":"100ms"},%s,{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50}]}}`, upgrade, raw, !upgrade)
				version := new(v14.Version)
				migrate := version.UpgradeExchange
				if !upgrade {
					migrate = version.DowngradeExchange
				}
				got, err := migrate(t.Context(), []byte(input))
				require.NoError(t, err, "migration must accept raw legacy channels")
				if test.wantMigration {
					expected := fmt.Sprintf(`{"features":{"subscriptions":[{"enabled":%t,"channel":"orderbook","asset":"spot","interval":"100ms"},%s,{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50}]}}`, !upgrade, raw, upgrade)
					assert.JSONEq(t, expected, string(got), "inactive raw legacy entries should allow migration without being changed")
				} else {
					assert.Equal(t, input, string(got), "migration should preserve the raw legacy feed and its generic snapshot dependency byte-for-byte")
				}
				again, err := migrate(t.Context(), got)
				require.NoError(t, err, "repeated migration must not error")
				assert.Equal(t, got, again, "migration should be idempotent")
			})
		}
	}
}

func TestMigrationMatchesSpotAssetCaseInsensitively(t *testing.T) {
	t.Parallel()
	for _, assetName := range []string{"SPOT", "Spot"} {
		for _, upgrade := range []bool{true, false} {
			t.Run(fmt.Sprintf("asset=%s/upgrade=%t", assetName, upgrade), func(t *testing.T) {
				t.Parallel()
				input := fmt.Sprintf(`{"features":{"subscriptions":[{"enabled":%t,"channel":"orderbook","asset":%q,"interval":"100ms"},{"enabled":%t,"channel":"spot.obu","asset":%q,"levels":50}]}}`, upgrade, assetName, !upgrade, assetName)
				expected := fmt.Sprintf(`{"features":{"subscriptions":[{"enabled":%t,"channel":"orderbook","asset":%q,"interval":"100ms"},{"enabled":%t,"channel":"spot.obu","asset":%q,"levels":50}]}}`, !upgrade, assetName, upgrade, assetName)
				version := new(v14.Version)
				migrate := version.UpgradeExchange
				if !upgrade {
					migrate = version.DowngradeExchange
				}
				got, err := migrate(t.Context(), []byte(input))
				require.NoError(t, err, "migration must accept case-insensitive spot assets")
				assert.JSONEq(t, expected, string(got), "migration should update case-insensitive spot assets")
			})
		}
	}
}

func TestMissingV2RoundTrip(t *testing.T) {
	t.Parallel()
	version := new(v14.Version)
	in := []byte(`{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"}]}}`)
	upgraded, err := version.UpgradeExchange(t.Context(), in)
	require.NoError(t, err, "UpgradeExchange must add the missing V2 entry")
	down, err := version.DowngradeExchange(t.Context(), upgraded)
	require.NoError(t, err, "DowngradeExchange must restore the legacy feed")
	assert.JSONEq(t, `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`, string(down), "DowngradeExchange should retain the new V2 entry disabled")
	again, err := version.UpgradeExchange(t.Context(), down)
	require.NoError(t, err, "UpgradeExchange must support a downgraded config")
	assert.JSONEq(t, string(upgraded), string(again), "re-upgrade should restore the V2 feed")
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	in := []byte(`{"name":"GateIO","features":{"subscriptions":[{"enabled":false,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`)
	exp := `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`
	got, err := new(v14.Version).DowngradeExchange(t.Context(), in)
	require.NoError(t, err, "DowngradeExchange must not error")
	assert.JSONEq(t, exp, string(got), "DowngradeExchange should restore previous defaults")
}

func TestUpgradeExchangeRejectsMalformedSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := new(v14.Version).UpgradeExchange(t.Context(), []byte(`{"features":{"subscriptions":{}}}`))
	require.Error(t, err, "UpgradeExchange must reject malformed subscriptions")
}

func TestDowngradeExchangeRejectsMalformedSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := new(v14.Version).DowngradeExchange(t.Context(), []byte(`{"features":{"subscriptions":{}}}`))
	require.Error(t, err, "DowngradeExchange must reject malformed subscriptions")
}
