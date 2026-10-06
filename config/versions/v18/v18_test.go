package v18_test

import (
	"os"
	"strings"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v18 "github.com/thrasher-corp/gocryptotrader/config/versions/v18"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deribit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okx"
)

const (
	oldDeribit = `[{"enabled":true,"channel":"candles","asset":"all","interval":"24h"},{"enabled":true,"channel":"orderbook","asset":"all","interval":"100ms"},{"enabled":true,"channel":"ticker","asset":"all","interval":"100ms"},{"enabled":true,"channel":"allTrades","asset":"all","interval":"100ms"},{"enabled":true,"channel":"myOrders","asset":"all","interval":"100ms","authenticated":true},{"enabled":true,"channel":"myTrades","asset":"all","interval":"100ms","authenticated":true}]`
	oldOkx     = `[{"enabled":true,"channel":"allTrades","asset":"all"},{"enabled":true,"channel":"orderbook","asset":"all"},{"enabled":true,"channel":"ticker","asset":"all"},{"enabled":true,"channel":"myOrders","asset":"all","authenticated":true},{"enabled":true,"channel":"myAccount","authenticated":true}]`
)

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, exchange, subscriptions string
		added                         int
	}{
		{"Deribit defaults", "Deribit", oldDeribit, 1},
		{"Deribit lower case", "deribit", oldDeribit, 1},
		{"OKX defaults", "Okx", oldOkx, 3},
		{"OKX upper case", "OKX", oldOkx, 3},
		{"asset case", "OKX", strings.ReplaceAll(oldOkx, `"all"`, `"ALL"`), 3},
		{"empty", "Deribit", `[]`, 0},
		{"null", "Okx", `null`, 0},
		{"custom", "Okx", `[{"enabled":true,"channel":"ticker"}]`, 0},
		{"disabled defaults", "Okx", strings.ReplaceAll(oldOkx, `"enabled":true`, `"enabled":false`), 0},
		{"single disabled default", "Deribit", strings.Replace(oldDeribit, `"enabled":true`, `"enabled":false`, 1), 0},
		{"omitted enabled", "Okx", strings.Replace(oldOkx, `"enabled":true,`, ``, 1), 0},
		{"null enabled", "Okx", strings.Replace(oldOkx, `"enabled":true`, `"enabled":null`, 1), 0},
		{"disabled options override", "Okx", strings.TrimSuffix(oldOkx, "]") + `,{"enabled":false,"channel":"opt-summary","asset":"options"}]`, 2},
		{"disabled wildcard override", "Okx", strings.TrimSuffix(oldOkx, "]") + `,{"enabled":false,"channel":"opt-summary","asset":"ALL"}]`, 2},
		{"unrelated asset scope", "Okx", strings.TrimSuffix(oldOkx, "]") + `,{"enabled":false,"channel":"opt-summary","asset":"spot"}]`, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := []byte(`{"name":"` + tc.exchange + `","features":{"subscriptions":` + tc.subscriptions + `}}`)
			got, err := new(v18.Version).UpgradeExchange(t.Context(), input)
			require.NoError(t, err, "upgrade must succeed")
			var before, after []json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(tc.subscriptions), &before), "input must decode")
			raw, _, _, err := jsonparser.Get(got, "features", "subscriptions")
			require.NoError(t, err, "subscriptions must remain present")
			require.NoError(t, json.Unmarshal(raw, &after), "result must decode")
			assert.Len(t, after, len(before)+tc.added, "upgrade should append only eligible defaults")
			for i := range before {
				assert.JSONEq(t, string(before[i]), string(after[i]), "existing choices should remain unchanged")
			}
			if tc.added == 0 {
				assert.Equal(t, input, got, "ineligible config should remain byte-identical")
			}
			again, err := new(v18.Version).UpgradeExchange(t.Context(), got)
			require.NoError(t, err, "second upgrade must succeed")
			assert.Equal(t, got, again, "upgrade should be idempotent")
		})
	}
	for _, input := range []string{`{"name":"Deribit"}`, `{"name":"Okx","features":{}}`, `{"name":"Kraken","features":{"subscriptions":[]}}`} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			got, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(input))
			require.NoError(t, err, "upgrade must succeed")
			assert.Equal(t, input, string(got), "missing or unrelated settings should remain unchanged")
		})
	}
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"Deribit", "deribit", "Okx", "OKX"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			old := oldOkx
			if strings.EqualFold(name, "Deribit") {
				old = oldDeribit
			}
			original := []byte(`{"name":"` + name + `","features":{"subscriptions":` + old + `}}`)
			upgraded, err := new(v18.Version).UpgradeExchange(t.Context(), original)
			require.NoError(t, err, "upgrade must succeed")
			got, err := new(v18.Version).DowngradeExchange(t.Context(), upgraded)
			require.NoError(t, err, "downgrade must succeed")
			assert.JSONEq(t, string(original), string(got), "downgrade should restore the previous defaults")
		})
	}
	for _, tc := range []struct{ input, expected string }{
		{`{"name":"Okx","features":{"subscriptions":[{"enabled":false,"channel":"opt-summary","asset":"spot"},{"enabled":true,"channel":"ticker","custom":true}]}}`, `{"name":"Okx","features":{"subscriptions":[{"enabled":true,"channel":"ticker","custom":true}]}}`},
		{`{"name":"Deribit","features":{"subscriptions":[]}}`, `{"name":"Deribit","features":{"subscriptions":[]}}`},
		{`{"name":"Okx","features":{"subscriptions":null}}`, `{"name":"Okx","features":{"subscriptions":null}}`},
		{`{"name":"Deribit"}`, `{"name":"Deribit"}`},
		{`{"name":"Kraken","features":{"subscriptions":[{"channel":"myAccount"}]}}`, `{"name":"Kraken","features":{"subscriptions":[{"channel":"myAccount"}]}}`},
	} {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(tc.input))
			require.NoError(t, err, "downgrade must succeed")
			assert.JSONEq(t, tc.expected, string(got), "rollback should remove only channels unsupported by the older exchange implementation")
		})
	}
}

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"*"}, new(v18.Version).Exchanges(), "migration should select names case-insensitively itself")
}

func TestRegisteredUpgrade(t *testing.T) {
	t.Parallel()
	fixture, err := os.ReadFile("../../../testdata/configtest.json")
	require.NoError(t, err, "config fixture must load")
	for _, name := range []string{"Deribit", "deribit", "Okx", "OKX"} {
		modes := []string{"defaults", "empty", "null", "missing", "missing features", "custom"}
		if strings.EqualFold(name, "Okx") {
			modes = append(modes, "disabled options", "disabled wildcard", "unrelated scope")
		}
		for _, mode := range modes {
			t.Run(name+"/"+mode, func(t *testing.T) {
				t.Parallel()
				var ex exchange.IBotExchange
				old := oldOkx
				count := 8
				if strings.EqualFold(name, "Deribit") {
					ex = new(deribit.Exchange)
					old = oldDeribit
					count = 7
				} else {
					ex = new(okx.Exchange)
				}
				var cfg config.Config
				require.NoError(t, json.Unmarshal(fixture, &cfg), "fixture must decode")
				saved, err := cfg.GetExchangeConfig(name)
				require.NoError(t, err, "exchange config must exist")
				raw, err := json.Marshal(saved)
				require.NoError(t, err, "exchange config must encode")
				raw, err = jsonparser.Set(raw, []byte(`"`+name+`"`), "name")
				require.NoError(t, err, "name must be set")
				switch mode {
				case "defaults":
					raw, err = jsonparser.Set(raw, []byte(old), "features", "subscriptions")
				case "empty":
					raw, err = jsonparser.Set(raw, []byte(`[]`), "features", "subscriptions")
				case "null":
					raw, err = jsonparser.Set(raw, []byte(`null`), "features", "subscriptions")
				case "missing":
					raw = jsonparser.Delete(raw, "features", "subscriptions")
				case "missing features":
					raw = jsonparser.Delete(raw, "features")
				case "disabled options", "disabled wildcard", "unrelated scope":
					scope := "options"
					if mode == "disabled wildcard" {
						scope = "ALL"
					}
					if mode == "unrelated scope" {
						scope = "spot"
					} else {
						count = 7
					}
					savedList := strings.TrimSuffix(old, "]") + `,{"enabled":false,"channel":"opt-summary","asset":"` + scope + `"}]`
					raw, err = jsonparser.Set(raw, []byte(savedList), "features", "subscriptions")

				case "custom":
					raw, err = jsonparser.Set(raw, []byte(`[{"enabled":true,"channel":"ticker","asset":"all"}]`), "features", "subscriptions")
					count = 1
				}
				require.NoError(t, err, "input settings must encode")
				input := append([]byte(`{"version":17,"exchanges":[`), raw...)
				input = append(input, []byte(`]}`)...)
				migrated, err := versions.Manager.Deploy(t.Context(), input, versions.UseLatestVersion)
				require.NoError(t, err, "registered migration must succeed")
				var loaded config.Config
				require.NoError(t, json.Unmarshal(migrated, &loaded), "migrated config must decode")
				ex.SetDefaults()
				require.NoError(t, ex.Setup(&loaded.Exchanges[0]), "migrated config must set up")
				assert.Len(t, ex.GetBase().Features.Subscriptions, count, "runtime should retain full defaults or explicit custom subscriptions")
				expanded, err := ex.GetBase().Features.Subscriptions.ExpandTemplates(ex)
				require.NoError(t, err, "migrated subscriptions must expand")
				assert.NotEmpty(t, expanded, "runtime subscriptions should remain usable")
				if mode == "defaults" {
					rolledBack, err := versions.Manager.Deploy(t.Context(), migrated, 17)
					require.NoError(t, err, "registered downgrade must succeed")
					restored, _, _, err := jsonparser.Get(rolledBack, "exchanges", "[0]", "features", "subscriptions")
					require.NoError(t, err, "rollback must retain subscriptions")
					assert.JSONEq(t, old, string(restored), "rollback should remove only unsupported additions")
				}
			})
		}
	}
}
