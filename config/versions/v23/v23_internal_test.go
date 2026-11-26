package v23

import (
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const savedPreviousDefaults = `[{"enabled":true,"channel":"ticker","asset":"spot"},{"enabled":true,"channel":"allTrades","asset":"spot"},{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"},{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"}]`

func TestAssetDefaults(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []defaultSubscription{
		{channel: "ticker", asset: "options"},
		{channel: "allTrades", asset: "options"},
		{channel: "candles", asset: "options", interval: "1m"},
		{channel: "orderbook", asset: "options", interval: "100ms"},
	}, assetDefaults("options"), "assetDefaults should return the asset's four defaults")
	assert.Len(t, currentDefaults, 16, "currentDefaults should hold every asset's defaults")
}

func TestMarshal(t *testing.T) {
	t.Parallel()
	assert.JSONEq(t, savedPreviousDefaults, string(marshal(previousDefaults)), "marshal should save the defaults as GCT does")
	assert.JSONEq(t, "[]", string(marshal(nil)), "marshal should save no subscriptions as an empty list")
}

func TestSoleField(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, object, field string
		valueType           jsonparser.ValueType
		exp                 string
		ok                  bool
	}{
		{"sole field", `{"features":{"enabled":{}},"name":"Binance"}`, "features", jsonparser.Object, `{"enabled":{}}`, true},
		{"sole text field", `{"name":"Binance"}`, "name", jsonparser.String, "Binance", true},
		{"sole field escaped", `{"f\u0065atures":{}}`, "features", jsonparser.Object, "{}", true},
		{"sole field of another type", `{"features":[]}`, "features", jsonparser.Object, "", false},
		{"sole text field of another type", `{"name":0}`, "name", jsonparser.String, "", false},
		{"sole null field", `{"features":null}`, "features", jsonparser.Object, "", false},
		{"sole field in another case", `{"Features":{}}`, "features", jsonparser.Object, "", false},
		{"field given twice", `{"features":{},"features":{}}`, "features", jsonparser.Object, "", false},
		{"field given twice ignoring case", `{"features":{},"FEATURES":{}}`, "features", jsonparser.Object, "", false},
		{"field given twice, once escaped", `{"features":{},"f\u0065atures":{}}`, "features", jsonparser.Object, "", false},
		{"field matching ignoring case beyond ASCII", `{"features":{},"featureſ":{}}`, "features", jsonparser.Object, "", false},
		{"field matching once lower-cased beyond ASCII", `{"subscrİptions":[],"subscriptions":[]}`, "subscriptions", jsonparser.Array, "", false},
		{"sole field matching once lower-cased beyond ASCII", `{"subscrİptions":[]}`, "subscriptions", jsonparser.Array, "", false},
		{"field matching once unescaped and lower-cased", `{"subscriptions":[],"subscript\u0130ons":[]}`, "subscriptions", jsonparser.Array, "", false},
		{"field given again as another type ignoring case", `{"subscriptions":[],"SUBSCRIPTIONS":true}`, "subscriptions", jsonparser.Array, "", false},
		{"field matching neither build's comparison", `{"subscriptions":[],"ſubscrİptions":[]}`, "subscriptions", jsonparser.Array, "[]", true},
		{"no field", `{"name":"Binance"}`, "features", jsonparser.Object, "", false},
		{"empty object", `{}`, "features", jsonparser.Object, "", false},
		{"malformed object", `{"features":{},`, "features", jsonparser.Object, "", false},
		{"not an object", `[]`, "features", jsonparser.Object, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, ok := soleField([]byte(tc.object), tc.field, tc.valueType)
			require.Equal(t, tc.ok, ok, "soleField must report whether the field is the sole match with its type")
			if ok {
				assert.Equal(t, tc.exp, string(value), "soleField should return the field's value")
			}
		})
	}
}

func TestIsDefaults(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, subscriptions string
		exp                 bool
	}{
		{"defaults", savedPreviousDefaults, true},
		{"defaults in another order", `[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"},{"enabled":true,"channel":"allTrades","asset":"spot"},{"enabled":true,"channel":"ticker","asset":"spot"}]`, true},
		{"a default left out", `[{"enabled":true,"channel":"ticker","asset":"spot"},{"enabled":true,"channel":"allTrades","asset":"spot"},{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"}]`, false},
		{"a default given twice", `[{"enabled":true,"channel":"ticker","asset":"spot"},{"enabled":true,"channel":"ticker","asset":"spot"},{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"},{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"}]`, false},
		{"every default with one given again", savedPreviousDefaults[:len(savedPreviousDefaults)-1] + `,{"enabled":true,"channel":"ticker","asset":"spot"}]`, false},
		{"another subscription added", savedPreviousDefaults[:len(savedPreviousDefaults)-1] + `,{"enabled":true,"channel":"ticker","asset":"margin"}]`, false},
		{"a null entry", `[null,{"enabled":true,"channel":"allTrades","asset":"spot"},{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"},{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"}]`, false},
		{"no subscriptions", `[]`, false},
		{"malformed list", `[{"enabled":true,"channel":"ticker","asset":"spot"},`, false},
		{"defaults with a separator the loader rejects", savedPreviousDefaults[:len(savedPreviousDefaults)-2] + `,}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, isDefaults([]byte(tc.subscriptions), previousDefaults), "isDefaults should report whether the list is the defaults")
		})
	}
}

func TestDefaultSubscriptionSavedAs(t *testing.T) {
	t.Parallel()
	spotCandles := defaultSubscription{channel: "candles", asset: "spot", interval: "1m"}
	optionsCandles := defaultSubscription{channel: "candles", asset: "options", interval: "1m"}
	spotTicker := defaultSubscription{channel: "ticker", asset: "spot"}
	for _, tc := range []struct {
		name  string
		d     defaultSubscription
		entry string
		exp   bool
	}{
		{"default", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"}`, true},
		{"fields in another order", spotCandles, `{"interval":"1m","asset":"spot","channel":"candles","enabled":true}`, true},
		{"asset in another case", optionsCandles, `{"enabled":true,"channel":"candles","asset":"OPTIONS","interval":"1m"}`, true},
		{"asset in a case the loader lower-cases beyond ASCII", optionsCandles, `{"enabled":true,"channel":"candles","asset":"optİons","interval":"1m"}`, true},
		{"spot asset absent", spotCandles, `{"enabled":true,"channel":"candles","interval":"1m"}`, true},
		{"another asset absent", optionsCandles, `{"enabled":true,"channel":"candles","interval":"1m"}`, false},
		{"another asset", spotCandles, `{"enabled":true,"channel":"candles","asset":"options","interval":"1m"}`, false},
		{"asset matching beyond ASCII case", spotCandles, `{"enabled":true,"channel":"candles","asset":"ſpot","interval":"1m"}`, false},
		{"asset with a space", spotCandles, `{"enabled":true,"channel":"candles","asset":" spot","interval":"1m"}`, false},
		{"null asset", spotCandles, `{"enabled":true,"channel":"candles","asset":null,"interval":"1m"}`, false},
		{"disabled", spotCandles, `{"enabled":false,"channel":"candles","asset":"spot","interval":"1m"}`, false},
		{"enabled absent", spotCandles, `{"channel":"candles","asset":"spot","interval":"1m"}`, false},
		{"enabled saved as text", spotCandles, `{"enabled":"true","channel":"candles","asset":"spot","interval":"1m"}`, false},
		{"null enabled", spotCandles, `{"enabled":null,"channel":"candles","asset":"spot","interval":"1m"}`, false},
		{"another channel", spotCandles, `{"enabled":true,"channel":"Candles","asset":"spot","interval":"1m"}`, false},
		{"channel absent", spotCandles, `{"enabled":true,"asset":"spot","interval":"1m"}`, false},
		{"another interval spelling", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"60s"}`, false},
		{"interval in nanoseconds", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":60000000000}`, false},
		{"interval absent", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot"}`, false},
		{"interval on a ticker", spotTicker, `{"enabled":true,"channel":"ticker","asset":"spot","interval":"1m"}`, false},
		{"ticker", spotTicker, `{"enabled":true,"channel":"ticker","asset":"spot"}`, true},
		{"a field the default does not use", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m","levels":0}`, false},
		{"pairs saved as an empty list", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m","pairs":[]}`, false},
		{"empty params", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m","params":{}}`, false},
		{"a field the default does not use given again as null", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m","authenticated":true,"authenticated":null}`, false},
		{"enabled given twice", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m","enabled":true}`, false},
		{"enabled given again as null", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m","enabled":null}`, false},
		{"channel given twice", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m","channel":"candles"}`, false},
		{"asset given twice", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m","asset":"spot"}`, false},
		{"interval given twice", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m","interval":"1m"}`, false},
		{"empty interval on a ticker", spotTicker, `{"enabled":true,"channel":"ticker","asset":"spot","interval":""}`, false},
		{"a field in another case", spotCandles, `{"Enabled":true,"channel":"candles","asset":"spot","interval":"1m"}`, false},
		{"malformed entry", spotCandles, `{"enabled":true,"channel","asset":"spot","interval":"1m"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, tc.d.savedAs([]byte(tc.entry)), "savedAs should report whether the entry is the default as GCT saves it")
		})
	}
}

func TestReplaceDefaults(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, exchange, exp string
	}{
		{"defaults", `{"name":"Binance","features":{"subscriptions":` + savedPreviousDefaults + `}}`, `{"name":"Binance","features":{"subscriptions":` + string(marshal(currentDefaults)) + `}}`},
		{"no name", `{"features":{"subscriptions":` + savedPreviousDefaults + `}}`, ""},
		{"no features", `{"name":"Binance"}`, ""},
		{"features not an object", `{"name":"Binance","features":[]}`, ""},
		{"no subscriptions", `{"name":"Binance","features":{}}`, ""},
		{"subscriptions not a list", `{"name":"Binance","features":{"subscriptions":{}}}`, ""},
		{"subscriptions the loader rejects", `{"name":"Binance","features":{"subscriptions":` + savedPreviousDefaults[:len(savedPreviousDefaults)-1] + `,]}}`, ""},
		{"customised subscriptions", `{"name":"Binance","features":{"subscriptions":[{"enabled":true,"channel":"ticker","asset":"spot"}]}}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := replaceDefaults([]byte(tc.exchange), previousDefaults, currentDefaults)
			require.NoError(t, err, "replaceDefaults must not error")
			exp := tc.exp
			if exp == "" {
				exp = tc.exchange
			}
			assert.Equal(t, exp, string(out), "replaceDefaults should replace only the defaults")
		})
	}
}
