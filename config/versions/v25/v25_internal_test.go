package v25

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
		exp, key            string
		ok                  bool
	}{
		{"sole field", `{"features":{"enabled":{}},"name":"Binance"}`, "features", jsonparser.Object, `{"enabled":{}}`, "features", true},
		{"sole text field", `{"name":"Binance"}`, "name", jsonparser.String, "Binance", "name", true},
		{"sole field escaped", `{"f\u0065atures":{}}`, "features", jsonparser.Object, "{}", "features", true},
		{"sole field in another case", `{"Features":{}}`, "features", jsonparser.Object, "{}", "Features", true},
		{"sole field in upper case", `{"SUBSCRIPTIONS":[]}`, "subscriptions", jsonparser.Array, "[]", "SUBSCRIPTIONS", true},
		{"sole field escaped in another case", `{"F\u0065atures":{}}`, "features", jsonparser.Object, "{}", "Features", true},
		{"sole field of another type", `{"features":[]}`, "features", jsonparser.Object, "", "", false},
		{"sole text field of another type", `{"name":0}`, "name", jsonparser.String, "", "", false},
		{"sole null field", `{"features":null}`, "features", jsonparser.Object, "", "", false},
		{"field given twice", `{"features":{},"features":{}}`, "features", jsonparser.Object, "", "", false},
		{"field given twice ignoring case", `{"features":{},"FEATURES":{}}`, "features", jsonparser.Object, "", "", false},
		{"field given twice, once escaped", `{"features":{},"f\u0065atures":{}}`, "features", jsonparser.Object, "", "", false},
		{"field matching ignoring case beyond ASCII", `{"features":{},"featureſ":{}}`, "features", jsonparser.Object, "", "", false},
		{"sole field matching ignoring case beyond ASCII", `{"featureſ":{}}`, "features", jsonparser.Object, "", "", false},
		{"field matching once lower-cased beyond ASCII", `{"subscrİptions":[],"subscriptions":[]}`, "subscriptions", jsonparser.Array, "", "", false},
		{"sole field matching once lower-cased beyond ASCII", `{"subscrİptions":[]}`, "subscriptions", jsonparser.Array, "", "", false},
		{"field matching once unescaped and lower-cased", `{"subscriptions":[],"subscript\u0130ons":[]}`, "subscriptions", jsonparser.Array, "", "", false},
		{"field given again as another type ignoring case", `{"subscriptions":[],"SUBSCRIPTIONS":true}`, "subscriptions", jsonparser.Array, "", "", false},
		{"null field before one in another case", `{"features":null,"Features":{}}`, "features", jsonparser.Object, "", "", false},
		{"null field after one in another case", `{"Features":{},"features":null}`, "features", jsonparser.Object, "", "", false},
		{"sole field in mixed case", `{"sUbScRiPtIoNs":[]}`, "subscriptions", jsonparser.Array, "[]", "sUbScRiPtIoNs", true},
		{"field matching neither build's comparison", `{"subscriptions":[],"ſubscrİptions":[]}`, "subscriptions", jsonparser.Array, "[]", "subscriptions", true},
		{"no field", `{"name":"Binance"}`, "features", jsonparser.Object, "", "", false},
		{"empty object", `{}`, "features", jsonparser.Object, "", "", false},
		{"malformed object", `{"features":{},`, "features", jsonparser.Object, "", "", false},
		{"not an object", `[]`, "features", jsonparser.Object, "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, key, ok := soleField([]byte(tc.object), tc.field, tc.valueType)
			require.Equal(t, tc.ok, ok, "soleField must report whether the field is the sole match both decoders read, with its type")
			if ok {
				assert.Equal(t, tc.exp, string(value), "soleField should return the field's value")
				assert.Equal(t, tc.key, key, "soleField should return the name the field is saved under")
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
		{"a field the default does not use, named as an extension", spotCandles, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m","x-note":"kept"}`, false},
		{"another order book interval", defaultSubscription{channel: "orderbook", asset: "spot", interval: "100ms"}, `{"enabled":true,"channel":"orderbook","asset":"spot","interval":"1s"}`, false},
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
		{"defaults under features in another case", `{"name":"Binance","Features":{"subscriptions":` + savedPreviousDefaults + `}}`, `{"name":"Binance","Features":{"subscriptions":` + string(marshal(currentDefaults)) + `}}`},
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

func TestReadEntry(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, raw      string
		dataType       jsonparser.ValueType
		asset, channel string
		futuresBook    bool
		ok             bool
	}{
		{"asset and channel", `{"enabled":true,"channel":"ticker","asset":"usdtmarginedfutures"}`, jsonparser.Object, "usdtmarginedfutures", "ticker", false, true},
		{"asset lower-cased", `{"channel":"ticker","asset":"ALL"}`, jsonparser.Object, "all", "ticker", false, true},
		{"asset under a field in another case", `{"channel":"ticker","ASSET":"spot"}`, jsonparser.Object, "spot", "ticker", false, true},
		{"escaped asset", `{"channel":"ticker","asset":"\u0061ll"}`, jsonparser.Object, "all", "ticker", false, true},
		{"no asset", `{"channel":"ticker"}`, jsonparser.Object, "", "ticker", false, true},
		{"null asset", `{"channel":"ticker","asset":null}`, jsonparser.Object, "", "ticker", false, true},
		{"channel kept in its case", `{"channel":"Ticker"}`, jsonparser.Object, "", "Ticker", false, true},
		{"asset given twice with one value", `{"asset":"all","Asset":"ALL","channel":"ticker"}`, jsonparser.Object, "all", "ticker", false, true},
		{"asset saved as a number", `{"channel":"ticker","asset":1}`, jsonparser.Object, "", "", false, false},
		{"channel saved as an object", `{"channel":{},"asset":"spot"}`, jsonparser.Object, "", "", false, false},
		{"asset given twice with two values", `{"asset":"all","Asset":"spot"}`, jsonparser.Object, "", "", false, false},
		{"asset given as null and text", `{"asset":null,"Asset":"all"}`, jsonparser.Object, "", "", false, false},
		{"channel given twice with two values", `{"channel":"ticker","CHANNEL":"orderbook"}`, jsonparser.Object, "", "", false, false},
		{"asset under a field only the default decoder reads", `{"channel":"ticker","aſſet":"all"}`, jsonparser.Object, "", "", false, false},
		{"entry saved as text", `"ticker"`, jsonparser.String, "", "", false, false},
		{"null entry", `null`, jsonparser.Null, "", "", false, false},
		{"malformed entry", `{"channel","asset":"spot"}`, jsonparser.Object, "", "", false, false},
		{"all-assets order book without an interval or levels", `{"channel":"orderbook","asset":"all"}`, jsonparser.Object, "all", "orderbook", true, true},
		{"disabled all-assets order book at 100ms", `{"enabled":false,"channel":"orderbook","asset":"ALL","interval":"100ms"}`, jsonparser.Object, "all", "orderbook", true, true},
		{"all-assets order book at 500ms with 20 levels", `{"channel":"orderbook","asset":"all","interval":"500ms","levels":20}`, jsonparser.Object, "all", "orderbook", true, true},
		{"all-assets order book at 1s", `{"channel":"orderbook","asset":"all","interval":"1s"}`, jsonparser.Object, "all", "orderbook", false, true},
		{"all-assets order book with 25 levels", `{"channel":"orderbook","asset":"all","interval":"100ms","levels":25}`, jsonparser.Object, "all", "orderbook", false, true},
		{"all-assets order book with its interval under a field in another case", `{"channel":"orderbook","asset":"all","Interval":"1s"}`, jsonparser.Object, "all", "orderbook", false, true},
		{"all-assets order book with its interval given twice with one value", `{"channel":"orderbook","asset":"all","interval":"1s","INTERVAL":"1s"}`, jsonparser.Object, "all", "orderbook", false, true},
		{"all-assets order book with its interval given twice with two values", `{"channel":"orderbook","asset":"all","interval":"100ms","Interval":"1s"}`, jsonparser.Object, "", "", false, false},
		{"all-assets order book with its interval under a field only sonic reads", `{"channel":"orderbook","asset":"all","İnterval":"1s"}`, jsonparser.Object, "", "", false, false},
		{"all-assets order book with its levels under a field only the default decoder reads", `{"channel":"orderbook","asset":"all","levelſ":25}`, jsonparser.Object, "", "", false, false},
		{"all-assets order book with its levels given twice with two values", `{"channel":"orderbook","asset":"all","levels":5,"Levels":25}`, jsonparser.Object, "", "", false, false},
		{"all-assets order book given two refused update speeds", `{"channel":"orderbook","asset":"all","interval":"1s","Interval":"250ms"}`, jsonparser.Object, "", "", false, false},
		{"all-assets order book at a refused update speed given two depths", `{"channel":"orderbook","asset":"all","interval":"1s","levels":5,"Levels":10}`, jsonparser.Object, "", "", false, false},
		{"all-assets order book at a refused depth given two update speeds", `{"channel":"orderbook","asset":"all","interval":"100ms","Interval":"500ms","levels":25}`, jsonparser.Object, "", "", false, false},
		{"asset given twice as synonyms", `{"channel":"ticker","asset":"option","Asset":"options"}`, jsonparser.Object, "", "", false, false},
		{"channel given twice, once escaped", `{"channel":"ticker","Channel":"tick\u0065r","asset":"usdtmarginedfutures"}`, jsonparser.Object, "usdtmarginedfutures", "ticker", false, true},
		{"all-assets order book given one update speed in two spellings", `{"channel":"orderbook","asset":"all","interval":"100ms","Interval":"0.1s"}`, jsonparser.Object, "", "", false, false},
		{"all-assets order book given no levels as 0 and -0", `{"channel":"orderbook","asset":"all","levels":0,"Levels":-0}`, jsonparser.Object, "", "", false, false},
		{"all-assets order book given its interval three times, the middle one different", `{"channel":"orderbook","asset":"all","interval":"1s","Interval":"100ms","INTERVAL":"1s"}`, jsonparser.Object, "", "", false, false},
		{"all-assets order book given two refused depths", `{"channel":"orderbook","asset":"all","levels":25,"Levels":50}`, jsonparser.Object, "", "", false, false},
		{"interval of an all-assets ticker not read", `{"channel":"ticker","asset":"all","interval":"100ms","İnterval":"1s"}`, jsonparser.Object, "all", "ticker", false, true},
		{"interval of a USDⓈ-M order book not read", `{"channel":"orderbook","asset":"usdtmarginedfutures","interval":"100ms","Interval":"1s"}`, jsonparser.Object, "usdtmarginedfutures", "orderbook", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			entry, ok := readEntry([]byte(tc.raw), tc.dataType)
			require.Equal(t, tc.ok, ok, "readEntry must report whether the loader reads the entry unambiguously")
			if !ok {
				return
			}
			assert.Equal(t, tc.asset, entry.asset, "readEntry should read the asset as the loader does")
			assert.Equal(t, tc.channel, entry.channel, "readEntry should read the channel as the loader does")
			assert.Equal(t, tc.futuresBook, entry.futuresBook, "readEntry should report whether the futures streams take an all-assets order book")
		})
	}
}

func TestTexts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		values []savedValue
		exp    []string
		ok     bool
	}{
		{"no values", nil, []string{}, true},
		{"text", []savedValue{{raw: "all", valueType: jsonparser.String, bothDecoders: true}}, []string{"all"}, true},
		{"escaped text", []savedValue{{raw: `\u0061ll`, valueType: jsonparser.String, bothDecoders: true}}, []string{"all"}, true},
		{"null", []savedValue{{raw: "null", valueType: jsonparser.Null, bothDecoders: true}}, []string{""}, true},
		{"text and null", []savedValue{{raw: "all", valueType: jsonparser.String, bothDecoders: true}, {raw: "null", valueType: jsonparser.Null, bothDecoders: true}}, []string{"all", ""}, true},
		{"a number", []savedValue{{raw: "1", valueType: jsonparser.Number, bothDecoders: true}}, nil, false},
		{"malformed escape", []savedValue{{raw: `\u00`, valueType: jsonparser.String, bothDecoders: true}}, nil, false},
		{"text under a field only one decoder reads", []savedValue{{raw: "all", valueType: jsonparser.String}}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, ok := texts(tc.values)
			assert.Equal(t, tc.ok, ok, "texts should report whether every value is text or null under a field both decoders read")
			assert.Equal(t, tc.exp, out, "texts should return each value's text")
		})
	}
}

func TestSoleValue(t *testing.T) {
	t.Parallel()
	interval := savedValue{raw: "1s", valueType: jsonparser.String, bothDecoders: true}
	for _, tc := range []struct {
		name   string
		values []savedValue
		exp    savedValue
		ok     bool
	}{
		{"no value", nil, savedValue{valueType: jsonparser.NotExist, bothDecoders: true}, true},
		{"one value", []savedValue{interval}, interval, true},
		{"one value given twice", []savedValue{interval, interval}, interval, true},
		{"two values", []savedValue{interval, {raw: "100ms", valueType: jsonparser.String, bothDecoders: true}}, interval, false},
		{"one value given as text and as a number", []savedValue{{raw: "5", valueType: jsonparser.String, bothDecoders: true}, {raw: "5", valueType: jsonparser.Number, bothDecoders: true}}, savedValue{raw: "5", valueType: jsonparser.String, bothDecoders: true}, false},
		{"a value under a field only one decoder reads", []savedValue{{raw: "1s", valueType: jsonparser.String}}, savedValue{raw: "1s", valueType: jsonparser.String}, false},
		{"one value given again under a field only one decoder reads", []savedValue{interval, {raw: "1s", valueType: jsonparser.String}}, interval, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, ok := soleValue(tc.values)
			require.Equal(t, tc.ok, ok, "soleValue must report whether the field holds one value both decoders read")
			if ok {
				assert.Equal(t, tc.exp, value, "soleValue should return the field's value")
			}
		})
	}
}

func TestTakesFuturesInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		raw       string
		valueType jsonparser.ValueType
		exp       bool
	}{
		{"no interval", "", jsonparser.NotExist, true},
		{"100ms", "100ms", jsonparser.String, true},
		{"500ms", "500ms", jsonparser.String, true},
		{"100ms in seconds", "0.1s", jsonparser.String, true},
		{"500ms in microseconds", "500000us", jsonparser.String, true},
		{"100ms with a sign", "+100ms", jsonparser.String, true},
		{"zero", "0", jsonparser.String, true},
		{"zero seconds", "0s", jsonparser.String, true},
		{"negative zero seconds", "-0s", jsonparser.String, true},
		{"zero of a unit only the loader reads", "0sec", jsonparser.String, true},
		{"zeros of a unit only the loader reads", "00days", jsonparser.String, true},
		{"zero of a unit in upper case only the loader reads", "0SEC", jsonparser.String, true},
		{"100ms written in two units", "0s100ms", jsonparser.String, true},
		{"zero months", "0M", jsonparser.String, true},
		{"100ms in nanoseconds", "100000000", jsonparser.Number, true},
		{"500ms in nanoseconds", "500000000", jsonparser.Number, true},
		{"100ms and a nanosecond", "100000001", jsonparser.Number, false},
		{"100ms more than 2^32 nanoseconds", "4394967296", jsonparser.Number, false},
		{"100ms and half a millisecond", "100.5ms", jsonparser.String, false},
		{"half a millisecond", "0.5ms", jsonparser.String, false},
		{"zero nanoseconds", "0", jsonparser.Number, true},
		{"negative zero nanoseconds", "-0", jsonparser.Number, true},
		{"1s", "1s", jsonparser.String, false},
		{"1000ms", "1000ms", jsonparser.String, false},
		{"250ms", "250ms", jsonparser.String, false},
		{"-100ms", "-100ms", jsonparser.String, false},
		{"one of a unit only the loader reads", "1sec", jsonparser.String, false},
		{"a count of days with a leading zero", "01days", jsonparser.String, false},
		{"raw", "raw", jsonparser.String, false},
		{"empty text, which the loader refuses", "", jsonparser.String, false},
		{"escaped 1s, which the loader refuses", `\u0031s`, jsonparser.String, false},
		{"1s in nanoseconds", "1000000000", jsonparser.Number, false},
		{"100ms in nanoseconds with an exponent, which the loader refuses", "1e8", jsonparser.Number, false},
		{"null, which the loader refuses", "null", jsonparser.Null, false},
		{"true, which the loader refuses", "true", jsonparser.Boolean, false},
		{"an object, which the loader refuses", "{}", jsonparser.Object, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, takesFuturesInterval(savedValue{raw: tc.raw, valueType: tc.valueType, bothDecoders: true}), "takesFuturesInterval should report whether the futures streams take the update speed")
		})
	}
}

func TestTakesFuturesLevels(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		raw       string
		valueType jsonparser.ValueType
		exp       bool
	}{
		{"no levels", "", jsonparser.NotExist, true},
		{"null", "null", jsonparser.Null, true},
		{"zero", "0", jsonparser.Number, true},
		{"negative zero", "-0", jsonparser.Number, true},
		{"5", "5", jsonparser.Number, true},
		{"10", "10", jsonparser.Number, true},
		{"20", "20", jsonparser.Number, true},
		{"1", "1", jsonparser.Number, false},
		{"-1", "-1", jsonparser.Number, false},
		{"15", "15", jsonparser.Number, false},
		{"25", "25", jsonparser.Number, false},
		{"50", "50", jsonparser.Number, false},
		{"261, which a byte holds as 5", "261", jsonparser.Number, false},
		{"5 with a fraction, which the loader refuses", "5.0", jsonparser.Number, false},
		{"5 as text, which the loader refuses", "5", jsonparser.String, false},
		{"true, which the loader refuses", "true", jsonparser.Boolean, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, takesFuturesLevels(savedValue{raw: tc.raw, valueType: tc.valueType, bothDecoders: true}), "takesFuturesLevels should report whether the futures streams take the depth")
		})
	}
}

func TestMatchesField(t *testing.T) {
	t.Parallel()
	for key, exp := range map[string]bool{
		"asset":   true,
		"Asset":   true,
		"ASSET":   true,
		"aſſet":   true,
		"assets":  false,
		"channel": false,
	} {
		assert.Equalf(t, exp, matchesField(key, "asset"), "matchesField should report whether the loader reads %q as asset", key)
	}
	assert.True(t, matchesField("subscrİptions", "subscriptions"), "matchesField should match what sonic's lower-casing matches")
}

func TestMatchesBoth(t *testing.T) {
	t.Parallel()
	for key, exp := range map[string]bool{
		"asset": true,
		"ASSET": true,
		"aſſet": false,
		"other": false,
	} {
		assert.Equalf(t, exp, matchesBoth(key, "asset"), "matchesBoth should report whether both decoders read %q as asset", key)
	}
}

func TestAllSame(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		same func() bool
		exp  bool
	}{
		{"no values", func() bool { return allSame[string](nil) }, true},
		{"one value given twice", func() bool { return allSame([]string{"all", "all"}) }, true},
		{"two values", func() bool { return allSame([]string{"all", "spot"}) }, false},
		{"a different value between two equal ones", func() bool { return allSame([]string{"all", "spot", "all"}) }, false},
		{"a different value after two equal ones", func() bool { return allSame([]string{"all", "all", "spot"}) }, false},
		{"one saved value given twice", func() bool {
			return allSame([]savedValue{{raw: "5", valueType: jsonparser.Number}, {raw: "5", valueType: jsonparser.Number}})
		}, true},
		{"one text saved as two types", func() bool {
			return allSame([]savedValue{{raw: "5", valueType: jsonparser.Number}, {raw: "5", valueType: jsonparser.String}})
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, tc.same(), "allSame should report whether every value is the first")
		})
	}
}

func TestSavedSubscriptions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, exchange, exp string
		path                []string
		ok                  bool
	}{
		{"saved subscriptions", `{"name":"Binance","features":{"subscriptions":[]}}`, "[]", []string{"features", "subscriptions"}, true},
		{"features in another case", `{"name":"Binance","Features":{"subscriptions":[]}}`, "[]", []string{"Features", "subscriptions"}, true},
		{"subscriptions in upper case", `{"name":"Binance","features":{"SUBSCRIPTIONS":[]}}`, "[]", []string{"features", "SUBSCRIPTIONS"}, true},
		{"features and subscriptions in mixed case", `{"name":"Binance","FeAtUrEs":{"sUbScRiPtIoNs":[]}}`, "[]", []string{"FeAtUrEs", "sUbScRiPtIoNs"}, true},
		{"null features beside features in another case", `{"name":"Binance","features":null,"Features":{"subscriptions":[]}}`, "", nil, false},
		{"no name", `{"features":{"subscriptions":[]}}`, "", nil, false},
		{"features given twice ignoring case", `{"name":"Binance","features":{"subscriptions":[]},"Features":{}}`, "", nil, false},
		{"features under a field only the default decoder reads", `{"name":"Binance","featureſ":{"subscriptions":[]}}`, "", nil, false},
		{"subscriptions given twice ignoring case", `{"name":"Binance","features":{"subscriptions":[],"Subscriptions":[]}}`, "", nil, false},
		{"subscriptions saved as an object", `{"name":"Binance","features":{"subscriptions":{}}}`, "", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			subscriptions, path, ok := savedSubscriptions([]byte(tc.exchange))
			require.Equal(t, tc.ok, ok, "savedSubscriptions must report whether the loader reads one subscriptions list")
			if ok {
				assert.Equal(t, tc.exp, string(subscriptions), "savedSubscriptions should return the saved list")
				assert.Equal(t, tc.path, path, "savedSubscriptions should return the field names the list is saved under")
			}
		})
	}
}

func TestStreamFuturesOrderbooks(t *testing.T) {
	t.Parallel()
	const (
		usdtmBook = `{"enabled":true,"channel":"orderbook","asset":"usdtmarginedfutures","interval":"100ms"}`
		coinmBook = `{"enabled":true,"channel":"orderbook","asset":"coinmarginedfutures","interval":"100ms"}`
	)
	for _, tc := range []struct {
		name, in, exp string
		ok            bool
	}{
		{"spot alone stays as saved", `[{"enabled":true,"channel":"ticker","asset":"spot"}]`, `[{"enabled":true,"channel":"ticker","asset":"spot"}]`, true},
		{"all-assets entries kept, naming the futures assets", `[{"enabled":true,"channel":"ticker","asset":"spot"},{"enabled":true,"channel":"ticker","asset":"all"}]`, `[{"enabled":true,"channel":"ticker","asset":"spot"},{"enabled":true,"channel":"ticker","asset":"all"},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"a disabled all-assets entry naming the futures assets", `[{"enabled":false,"channel":"candles","asset":"all","interval":"5m"}]`, `[{"enabled":false,"channel":"candles","asset":"all","interval":"5m"},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"all-assets myOrders kept, naming nothing", `[{"enabled":true,"channel":"myOrders","asset":"all"}]`, `[{"enabled":true,"channel":"myOrders","asset":"all"}]`, true},
		{"COIN-M without an order book gains one", `[{"enabled":true,"channel":"ticker","asset":"coinmarginedfutures"}]`, `[{"enabled":true,"channel":"ticker","asset":"coinmarginedfutures"},` + coinmBook + `]`, true},
		{"USDⓈ-M with an order book stays as saved", `[{"enabled":false,"channel":"orderbook","asset":"usdtmarginedfutures"}]`, `[{"enabled":false,"channel":"orderbook","asset":"usdtmarginedfutures"}]`, true},
		{"an all-assets order book the futures streams take", `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"500ms"}]`, `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"500ms"}]`, true},
		{"a disabled all-assets order book the futures streams take", `[{"enabled":false,"channel":"orderbook","asset":"ALL","interval":"100ms"}]`, `[{"enabled":false,"channel":"orderbook","asset":"ALL","interval":"100ms"}]`, true},
		{"an all-assets order book at an update speed the futures streams refuse", `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s"}]`, `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s"},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"an all-assets order book at a depth the futures streams refuse", `[{"enabled":true,"channel":"orderbook","asset":"all","levels":25}]`, `[{"enabled":true,"channel":"orderbook","asset":"all","levels":25},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"an all-assets order book the futures streams refuse beside a USDⓈ-M one", `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s"},{"enabled":false,"channel":"orderbook","asset":"usdtmarginedfutures","interval":"1s"}]`, `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s"},{"enabled":false,"channel":"orderbook","asset":"usdtmarginedfutures","interval":"1s"},` + coinmBook + `]`, true},
		{"a usable all-assets order book after a refused one", `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s"},{"enabled":true,"channel":"orderbook","asset":"all","interval":"100ms"}]`, `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s"},{"enabled":true,"channel":"orderbook","asset":"all","interval":"100ms"}]`, true},
		{"a usable all-assets order book before a refused one", `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"100ms"},{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s"}]`, `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"100ms"},{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s"}]`, true},
		{"a disabled all-assets order book at an update speed the futures streams refuse", `[{"enabled":false,"channel":"orderbook","asset":"all","interval":"1s"}]`, `[{"enabled":false,"channel":"orderbook","asset":"all","interval":"1s"},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"an all-assets order book at a negative depth", `[{"enabled":true,"channel":"orderbook","asset":"all","levels":-1}]`, `[{"enabled":true,"channel":"orderbook","asset":"all","levels":-1},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"an all-assets order book at a count of days with a leading zero", `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"01days"}]`, `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"01days"},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"a USDⓈ-M order book channel in another case", `[{"enabled":true,"channel":"OrderBook","asset":"usdtmarginedfutures"}]`, `[{"enabled":true,"channel":"OrderBook","asset":"usdtmarginedfutures"},` + usdtmBook + `]`, true},
		{"all-assets myOrders in another case", `[{"enabled":true,"channel":"myorders","asset":"all"}]`, `[{"enabled":true,"channel":"myorders","asset":"all"},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"an all-assets entry without a channel", `[{"enabled":true,"asset":"all"}]`, `[{"enabled":true,"asset":"all"},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"USDⓈ-M and COIN-M without order books gain both", `[{"enabled":true,"channel":"ticker","asset":"usdtmarginedfutures"},{"enabled":true,"channel":"ticker","asset":"coinmarginedfutures"}]`, `[{"enabled":true,"channel":"ticker","asset":"usdtmarginedfutures"},{"enabled":true,"channel":"ticker","asset":"coinmarginedfutures"},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"a COIN-M asset in a case the loader lower-cases beyond ASCII", `[{"enabled":true,"channel":"ticker","asset":"COİNMARGINEDFUTURES"}]`, `[{"enabled":true,"channel":"ticker","asset":"COİNMARGINEDFUTURES"},` + coinmBook + `]`, true},
		{"a COIN-M order book before another COIN-M subscription", `[{"enabled":true,"channel":"orderbook","asset":"coinmarginedfutures","levels":5},{"enabled":true,"channel":"ticker","asset":"coinmarginedfutures"}]`, `[{"enabled":true,"channel":"orderbook","asset":"coinmarginedfutures","levels":5},{"enabled":true,"channel":"ticker","asset":"coinmarginedfutures"}]`, true},
		{"a USDⓈ-M order book channel with a space", `[{"enabled":true,"channel":"orderbook ","asset":"usdtmarginedfutures"}]`, `[{"enabled":true,"channel":"orderbook ","asset":"usdtmarginedfutures"},` + usdtmBook + `]`, true},
		{"an all-assets entry with a null channel", `[{"enabled":true,"channel":null,"asset":"all"}]`, `[{"enabled":true,"channel":null,"asset":"all"},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"all-assets order books each refused on one count", `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"100ms","levels":25},{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s","levels":5}]`, `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"100ms","levels":25},{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s","levels":5},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"an entry holding a list with a closing bracket in its text", `[{"enabled":true,"channel":"ticker","asset":"usdtmarginedfutures","params":{"window":["]"]}}]`, `[{"enabled":true,"channel":"ticker","asset":"usdtmarginedfutures","params":{"window":["]"]}},` + usdtmBook + `]`, true},
		{"an authenticated USDⓈ-M ticker without an order book", `[{"enabled":true,"channel":"ticker","asset":"usdtmarginedfutures","authenticated":true}]`, `[{"enabled":true,"channel":"ticker","asset":"usdtmarginedfutures","authenticated":true},` + usdtmBook + `]`, true},
		{"an authenticated USDⓈ-M order book", `[{"enabled":true,"channel":"orderbook","asset":"usdtmarginedfutures","authenticated":true}]`, `[{"enabled":true,"channel":"orderbook","asset":"usdtmarginedfutures","authenticated":true}]`, true},
		{"a USDⓈ-M ticker saved without enabled", `[{"channel":"ticker","asset":"usdtmarginedfutures"}]`, `[{"channel":"ticker","asset":"usdtmarginedfutures"},` + usdtmBook + `]`, true},
		{"a USDⓈ-M order book saved without enabled", `[{"channel":"orderbook","asset":"usdtmarginedfutures"},{"enabled":true,"channel":"ticker","asset":"usdtmarginedfutures"}]`, `[{"channel":"orderbook","asset":"usdtmarginedfutures"},{"enabled":true,"channel":"ticker","asset":"usdtmarginedfutures"}]`, true},
		{"an order book saved without an asset beside an all-assets ticker", `[{"enabled":true,"channel":"orderbook","interval":"100ms"},{"enabled":true,"channel":"ticker","asset":"all"}]`, `[{"enabled":true,"channel":"orderbook","interval":"100ms"},{"enabled":true,"channel":"ticker","asset":"all"},` + usdtmBook + `,` + coinmBook + `]`, true},
		{"spacing kept", "[ {\"enabled\":true,\"channel\":\"ticker\",\"asset\":\"all\"}\n ]", "[ {\"enabled\":true,\"channel\":\"ticker\",\"asset\":\"all\"}\n ," + usdtmBook + "," + coinmBook + "]", true},
		{"invalid JSON", `[{"enabled":true,"channel":"ticker","asset":"all"},]`, "", false},
		{"an entry that is not an object", `[{"enabled":true,"channel":"ticker","asset":"all"},null]`, "", false},
		{"an entry the decoders could read differently", `[{"enabled":true,"channel":"ticker","aſſet":"all"}]`, "", false},
		{"an all-assets order book whose interval the decoders could read differently", `[{"enabled":true,"channel":"orderbook","asset":"all","İnterval":"1s"}]`, "", false},
		{"an all-assets order book given two refused update speeds", `[{"enabled":true,"channel":"orderbook","asset":"all","interval":"1s","Interval":"250ms"}]`, "", false},
		{"an all-assets order book given two refused depths", `[{"enabled":true,"channel":"orderbook","asset":"all","levels":25,"Levels":50}]`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, ok := streamFuturesOrderbooks([]byte(tc.in))
			require.Equal(t, tc.ok, ok, "streamFuturesOrderbooks must report whether it reads the list unambiguously")
			assert.Equal(t, tc.exp, string(out), "streamFuturesOrderbooks should return the expected list")
		})
	}
}

func TestMarshalEntry(t *testing.T) {
	t.Parallel()
	assert.JSONEq(t, `{"enabled":true,"channel":"ticker","asset":"spot"}`, string(marshalEntry(defaultSubscription{channel: "ticker", asset: "spot"})), "marshalEntry should save a subscription without an interval as GCT does")
	assert.JSONEq(t, `{"enabled":true,"channel":"orderbook","asset":"usdtmarginedfutures","interval":"100ms"}`, string(marshalEntry(defaultSubscription{channel: "orderbook", asset: "usdtmarginedfutures", interval: "100ms"})), "marshalEntry should save a subscription's interval as GCT does")
}
