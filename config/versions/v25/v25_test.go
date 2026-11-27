package v25_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v25 "github.com/thrasher-corp/gocryptotrader/config/versions/v25"
)

func savedSubscription(channel, asset, interval string) string {
	s := `{"enabled":true,"channel":"` + channel + `"`
	if asset != "" {
		s += `,"asset":"` + asset + `"`
	}
	if interval != "" {
		s += `,"interval":"` + interval + `"`
	}
	return s + "}"
}

func assetDefaults(a string) []string {
	return []string{
		savedSubscription("ticker", a, ""),
		savedSubscription("allTrades", a, ""),
		savedSubscription("candles", a, "1m"),
		savedSubscription("orderbook", a, "100ms"),
	}
}

var (
	previousDefaults = assetDefaults("spot")
	currentDefaults  = slices.Concat(assetDefaults("spot"), assetDefaults("usdtmarginedfutures"), assetDefaults("coinmarginedfutures"), assetDefaults("options"))
)

func list(entries []string) string {
	return "[" + strings.Join(entries, ",") + "]"
}

func binance(subscriptions string) string {
	return `{"name":"Binance","enabled":true,"features":{"enabled":{"websocketAPI":true},"subscriptions":` + subscriptions + `},"orderbook":{}}`
}

// customised returns configurations which differ from the defaults by one detail, each of which must stay as saved
func customised(defaults []string) map[string]string {
	edit := func(i int, f func(string) string) string {
		entries := slices.Clone(defaults)
		entries[i] = f(entries[i])
		return binance(list(entries))
	}
	replace := func(i int, old, replacement string) string {
		return edit(i, func(s string) string { return strings.Replace(s, old, replacement, 1) })
	}
	ticker, candles, orderbook := 0, slices.Index(defaults, savedSubscription("candles", "spot", "1m")), slices.Index(defaults, savedSubscription("orderbook", "spot", "100ms"))
	swapIntervals := func() string {
		entries := slices.Clone(defaults)
		entries[candles] = strings.Replace(entries[candles], `"1m"`, `"100ms"`, 1)
		entries[orderbook] = strings.Replace(entries[orderbook], `"100ms"`, `"1m"`, 1)
		return binance(list(entries))
	}
	return map[string]string{
		"a default left out":                   binance(list(defaults[1:])),
		"another subscription added":           binance(list(append(slices.Clone(defaults), savedSubscription("ticker", "margin", "")))),
		"a default given twice":                binance(list(append(slices.Clone(defaults[1:]), defaults[1]))),
		"every default with one given again":   binance(list(append(slices.Clone(defaults), defaults[0]))),
		"no subscriptions saved":               binance(`[]`),
		"a default disabled":                   replace(ticker, `"enabled":true`, `"enabled":false`),
		"enabled absent":                       replace(ticker, `"enabled":true,`, ``),
		"enabled saved as text":                replace(ticker, `"enabled":true`, `"enabled":"true"`),
		"a null enabled":                       replace(ticker, `"enabled":true`, `"enabled":null`),
		"a field the defaults do not use":      replace(ticker, `}`, `,"authenticated":false}`),
		"pairs saved as an empty list":         replace(ticker, `}`, `,"pairs":[]}`),
		"pairs saved as empty text":            replace(ticker, `}`, `,"pairs":""}`),
		"empty params":                         replace(ticker, `}`, `,"params":{}}`),
		"null params":                          replace(ticker, `}`, `,"params":null}`),
		"a field given twice":                  replace(ticker, `"enabled":true`, `"enabled":true,"enabled":true`),
		"a field given again as null":          replace(ticker, `"enabled":true`, `"enabled":true,"enabled":null`),
		"channel given again as null":          replace(ticker, `"channel":"ticker"`, `"channel":"ticker","channel":null`),
		"an unused field given again as null":  replace(ticker, `}`, `,"authenticated":true,"authenticated":null}`),
		"a field in another case":              replace(ticker, `"enabled"`, `"Enabled"`),
		"a false field in another case":        replace(ticker, `}`, `,"Authenticated":false}`),
		"a field matching ignoring case":       replace(ticker, `"asset"`, `"aſſet"`),
		"another channel":                      replace(ticker, `"ticker"`, `"Ticker"`),
		"a channel with a space":               replace(ticker, `"ticker"`, `"ticker "`),
		"a channel with a suffix":              replace(ticker, `"ticker"`, `"ticker@custom"`),
		"an escaped channel":                   replace(ticker, `"ticker"`, `"tick\u0065r"`),
		"an escaped asset":                     replace(ticker, `"spot"`, `"\u0073pot"`),
		"a null asset":                         replace(ticker, `"spot"`, `null`),
		"an empty asset":                       replace(ticker, `"spot"`, `""`),
		"an asset with a space":                replace(ticker, `"spot"`, `" spot"`),
		"another asset":                        replace(ticker, `"spot"`, `"margin"`),
		"an asset matching beyond ASCII case":  replace(ticker, `"spot"`, `"ſpot"`),
		"every asset":                          replace(ticker, `"spot"`, `"all"`),
		"an interval on a ticker":              replace(ticker, `}`, `,"interval":"1m"}`),
		"a zero interval on a ticker":          replace(ticker, `}`, `,"interval":0}`),
		"a null interval on a ticker":          replace(ticker, `}`, `,"interval":null}`),
		"a null entry":                         edit(ticker, func(string) string { return "null" }),
		"an entry saved as text":               edit(ticker, func(string) string { return `"ticker"` }),
		"another interval":                     replace(candles, `"1m"`, `"5m"`),
		"an interval in another unit":          replace(candles, `"1m"`, `"60s"`),
		"an interval in another case":          replace(candles, `"1m"`, `"1M"`),
		"an interval with a space":             replace(candles, `"1m"`, `" 1m"`),
		"an interval with more after it":       replace(candles, `"1m"`, `"1m30s"`),
		"intervals swapped between defaults":   swapIntervals(),
		"an interval in nanoseconds":           replace(candles, `"1m"`, `60000000000`),
		"an escaped interval":                  replace(candles, `"1m"`, `"1\u006d"`),
		"a candle without its interval":        replace(candles, `,"interval":"1m"`, ``),
		"subscriptions saved as an object":     binance(`{"0":` + defaults[0] + `}`),
		"null subscriptions":                   binance(`null`),
		"malformed subscriptions":              binance(`[` + defaults[0] + `,{"enabled":true,"channel"}]`),
		"no subscriptions field":               `{"name":"Binance","features":{"enabled":{}}}`,
		"no features":                          `{"name":"Binance"}`,
		"null features":                        `{"name":"Binance","features":null}`,
		"features in another case":             `{"name":"Binance","Features":{"subscriptions":` + list(defaults) + `}}`,
		"features given twice":                 `{"name":"Binance","features":{"subscriptions":` + list(defaults) + `},"features":{}}`,
		"features given twice ignoring case":   `{"name":"Binance","features":{"subscriptions":` + list(defaults) + `},"Features":{}}`,
		"a field matching features":            `{"name":"Binance","features":{"subscriptions":` + list(defaults) + `},"featureſ":{}}`,
		"subscriptions in another case":        `{"name":"Binance","features":{"Subscriptions":` + list(defaults) + `}}`,
		"subscriptions given twice":            `{"name":"Binance","features":{"subscriptions":` + list(defaults) + `,"subscriptions":[]}}`,
		"a field matching subscriptions":       `{"name":"Binance","features":{"subscriptions":` + list(defaults) + `,"ſubscriptions":[]}}`,
		"subscriptions again as another type":  `{"name":"Binance","features":{"subscriptions":` + list(defaults) + `,"SUBSCRIPTIONS":true}}`,
		"a lower-cased match of subscriptions": `{"name":"Binance","features":{"subscrİptions":[{"authenticated":true,"levels":5}],"subscriptions":` + list(defaults) + `}}`,
		"an escaped match of subscriptions":    `{"name":"Binance","features":{"subscr\u0130ptions":[{"authenticated":true,"levels":5}],"subscriptions":` + list(defaults) + `}}`,
		"name given twice ignoring case":       `{"name":"Binance","Name":"Kraken","features":{"subscriptions":` + list(defaults) + `}}`,
		"name given twice":                     `{"name":"Binance","name":"Kraken","features":{"subscriptions":` + list(defaults) + `}}`,
		"no name":                              `{"features":{"subscriptions":` + list(defaults) + `}}`,
		"a name saved as a number":             `{"name":0,"features":{"subscriptions":` + list(defaults) + `}}`,
		"features after a malformed field":     `{"name":"Binance","api":{,"features":{"subscriptions":` + list(defaults) + `}}`,
		"an entry with a malformed field":      replace(ticker, `"enabled":true`, `"enabled":yes`),
		"an entry with a malformed separator":  replace(ticker, `"enabled":true,`, `"enabled":true `),
		"an entry with a trailing separator":   replace(ticker, `}`, `,}`),
		"subscriptions with a trailing comma":  binance(`[` + strings.Join(defaults, ",") + `,]`),
		"subscriptions missing a separator":    binance(`[` + strings.Join(defaults, " ") + `]`),
		"subscriptions holding an empty entry": binance(`[` + strings.Join(defaults, ",") + `,{}]`),
	}
}

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Binance"}, new(v25.Version).Exchanges())
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	shuffled := slices.Clone(previousDefaults)
	slices.Reverse(shuffled)
	withoutAssets := []string{
		savedSubscription("ticker", "", ""),
		savedSubscription("allTrades", "", ""),
		savedSubscription("candles", "", "1m"),
		savedSubscription("orderbook", "", "100ms"),
	}
	upgraded := binance(list(currentDefaults))
	for _, tc := range []struct {
		name, in, exp string
	}{
		{"previous defaults", binance(list(previousDefaults)), upgraded},
		{"previous defaults in another order", binance(list(shuffled)), upgraded},
		{"previous defaults saved before subscriptions had assets", binance(list(withoutAssets)), upgraded},
		{"previous defaults with assets in other cases", binance(list([]string{
			savedSubscription("ticker", "Spot", ""),
			savedSubscription("allTrades", "SPOT", ""),
			savedSubscription("candles", "sPoT", "1m"),
			savedSubscription("orderbook", "", "100ms"),
		})), upgraded},
		{
			"previous defaults as the configuration saves them",
			"{\n   \"name\": \"Binance\",\n   \"features\": {\n    \"subscriptions\": [\n     {\n      \"enabled\": true,\n      \"channel\": \"ticker\",\n      \"asset\": \"spot\"\n     },\n     {\n      \"enabled\": true,\n      \"channel\": \"allTrades\",\n      \"asset\": \"spot\"\n     },\n     {\n      \"enabled\": true,\n      \"channel\": \"candles\",\n      \"asset\": \"spot\",\n      \"interval\": \"1m\"\n     },\n     {\n      \"enabled\": true,\n      \"channel\": \"orderbook\",\n      \"asset\": \"spot\",\n      \"interval\": \"100ms\"\n     }\n    ]\n   }\n  }",
			"{\n   \"name\": \"Binance\",\n   \"features\": {\n    \"subscriptions\": " + list(currentDefaults) + "\n   }\n  }",
		},
		{"current defaults", upgraded, upgraded},
		{"previous defaults with an all-assets ticker instead of spot's", binance(list(slices.Concat([]string{savedSubscription("ticker", "all", "")}, previousDefaults[1:]))), binance(list(previousDefaults[1:]))},
		{"previous defaults beside an all-assets ticker", binance(list(append(slices.Clone(previousDefaults), savedSubscription("ticker", "all", "")))), upgraded},
		{
			"all-assets subscriptions",
			binance(list([]string{savedSubscription("ticker", "spot", ""), savedSubscription("ticker", "all", ""), savedSubscription("orderbook", "ALL", "1s"), `{"enabled":false,"channel":"candles","asset":"All","interval":"5m"}`})),
			binance(list([]string{savedSubscription("ticker", "spot", "")})),
		},
		{"all-assets subscriptions alone", binance(list([]string{savedSubscription("ticker", "all", ""), savedSubscription("orderbook", "all", "100ms")})), binance(`[]`)},
		{"an all-assets subscription under a field in another case", binance(list([]string{savedSubscription("ticker", "spot", ""), `{"enabled":true,"channel":"ticker","Asset":"all"}`})), binance(list([]string{savedSubscription("ticker", "spot", "")}))},
		{"an escaped all-assets subscription", binance(list([]string{savedSubscription("ticker", "spot", ""), `{"enabled":true,"channel":"ticker","asset":"\u0061ll"}`})), binance(list([]string{savedSubscription("ticker", "spot", "")}))},
		{"an all-assets subscription with its asset given twice", binance(list([]string{savedSubscription("ticker", "spot", ""), `{"enabled":true,"channel":"ticker","asset":"all","ASSET":"all"}`})), binance(list([]string{savedSubscription("ticker", "spot", "")}))},
		{
			"a USDⓈ-M ticker with its channel given twice without an order book",
			binance(list([]string{`{"enabled":true,"channel":"ticker","Channel":"ticker","asset":"usdtmarginedfutures"}`})),
			binance(list([]string{`{"enabled":true,"channel":"ticker","Channel":"ticker","asset":"usdtmarginedfutures"}`, savedSubscription("orderbook", "usdtmarginedfutures", "100ms")})),
		},
		{
			"a USDⓈ-M ticker without an order book",
			binance(list([]string{savedSubscription("ticker", "spot", ""), savedSubscription("ticker", "usdtmarginedfutures", "")})),
			binance(list([]string{savedSubscription("ticker", "spot", ""), savedSubscription("ticker", "usdtmarginedfutures", ""), savedSubscription("orderbook", "usdtmarginedfutures", "100ms")})),
		},
		{
			"disabled COIN-M candles in another case without an order book",
			binance(list([]string{`{"enabled":false,"channel":"candles","asset":"CoinMarginedFutures","interval":"5m"}`})),
			binance(list([]string{`{"enabled":false,"channel":"candles","asset":"CoinMarginedFutures","interval":"5m"}`, savedSubscription("orderbook", "coinmarginedfutures", "100ms")})),
		},
		{
			"an all-assets ticker beside USDⓈ-M and COIN-M ones",
			binance(list([]string{savedSubscription("ticker", "all", ""), savedSubscription("ticker", "usdtmarginedfutures", ""), savedSubscription("allTrades", "coinmarginedfutures", "")})),
			binance(list([]string{savedSubscription("ticker", "usdtmarginedfutures", ""), savedSubscription("allTrades", "coinmarginedfutures", ""), savedSubscription("orderbook", "usdtmarginedfutures", "100ms"), savedSubscription("orderbook", "coinmarginedfutures", "100ms")})),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v25.Version).UpgradeExchange(t.Context(), []byte(tc.in))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.Equal(t, tc.exp, string(out), "UpgradeExchange should return the expected configuration")
			again, err := new(v25.Version).UpgradeExchange(t.Context(), out)
			require.NoError(t, err, "UpgradeExchange must not error on its own output")
			assert.Equal(t, tc.exp, string(again), "UpgradeExchange should leave its own output as it is")
		})
	}
	unchanged := map[string]string{
		"a USDⓈ-M order book":                                binance(list([]string{savedSubscription("ticker", "usdtmarginedfutures", ""), savedSubscription("orderbook", "usdtmarginedfutures", "500ms")})),
		"a disabled COIN-M order book":                       binance(list([]string{savedSubscription("ticker", "coinmarginedfutures", ""), `{"enabled":false,"channel":"orderbook","asset":"coinmarginedfutures","interval":"100ms"}`})),
		"USDⓈ-M myOrders alone":                              binance(list([]string{savedSubscription("ticker", "spot", ""), `{"enabled":true,"channel":"myOrders","asset":"usdtmarginedfutures","authenticated":true}`})),
		"options without an order book":                      binance(list([]string{savedSubscription("ticker", "options", "")})),
		"an all-assets asset given twice":                    binance(list([]string{`{"enabled":true,"channel":"ticker","asset":"all","Asset":"spot"}`})),
		"all-assets myOrders":                                binance(list([]string{savedSubscription("ticker", "spot", ""), `{"enabled":true,"channel":"myOrders","asset":"all","authenticated":true}`})),
		"an all-assets asset only the default decoder reads": binance(list([]string{savedSubscription("ticker", "spot", ""), `{"enabled":true,"channel":"ticker","aſſet":"all"}`})),
		"an all-assets asset saved as a number":              binance(list([]string{`{"enabled":true,"channel":"ticker","asset":1}`, savedSubscription("ticker", "all", "")})),
		"a USDⓈ-M channel given twice":                       binance(list([]string{`{"enabled":true,"channel":"ticker","Channel":"orderbook","asset":"usdtmarginedfutures"}`})),
		"an all-assets ticker beside a null":                 binance(list([]string{savedSubscription("ticker", "all", ""), `null`})),
	}
	for name, in := range customised(previousDefaults) {
		if name == "every asset" {
			// The all-assets ticker streamed nothing in the previous version, so it is dropped, as a row above shows
			continue
		}
		unchanged[name] = in
	}
	for name, in := range unchanged {
		t.Run("customised: "+name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v25.Version).UpgradeExchange(t.Context(), []byte(in))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.Equal(t, in, string(out), "UpgradeExchange should leave the configuration as saved")
		})
	}
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	shuffled := slices.Clone(currentDefaults)
	slices.Reverse(shuffled)
	otherCases := slices.Clone(currentDefaults)
	for i, s := range otherCases {
		if i%2 == 1 {
			start := strings.Index(s, `"asset":"`) + len(`"asset":"`)
			end := start + strings.Index(s[start:], `"`)
			otherCases[i] = s[:start] + strings.ToUpper(s[start:end]) + s[end:]
		}
	}
	spotWithoutAssets := slices.Clone(currentDefaults)
	for i := range 4 {
		spotWithoutAssets[i] = strings.Replace(spotWithoutAssets[i], `,"asset":"spot"`, ``, 1)
	}
	// The loader lower-cases assets with strings.ToLower, which turns İ into i
	beyondASCII := slices.Clone(currentDefaults)
	for i := range beyondASCII {
		beyondASCII[i] = strings.Replace(beyondASCII[i], `"options"`, `"optİons"`, 1)
	}
	// Each asset and each channel keeps its count of four, but two entries are no longer defaults
	channelsSwapped := slices.Clone(currentDefaults)
	channelsSwapped[0] = savedSubscription("allTrades", "spot", "")
	channelsSwapped[13] = savedSubscription("ticker", "options", "")
	downgraded := binance(list(previousDefaults))
	for _, tc := range []struct {
		name, in, exp string
	}{
		{"current defaults", binance(list(currentDefaults)), downgraded},
		{"current defaults in another order", binance(list(shuffled)), downgraded},
		{"current defaults with assets in other cases", binance(list(otherCases)), downgraded},
		{"current defaults with assets in cases beyond ASCII", binance(list(beyondASCII)), downgraded},
		{"current defaults with spot assets absent", binance(list(spotWithoutAssets)), downgraded},
		{"previous defaults", downgraded, downgraded},
		{"current defaults with channels swapped across assets", binance(list(channelsSwapped)), binance(list(channelsSwapped))},
		{
			"a customised list with the USDⓈ-M order book its upgrade added",
			binance(list([]string{savedSubscription("ticker", "spot", ""), savedSubscription("ticker", "usdtmarginedfutures", ""), savedSubscription("orderbook", "usdtmarginedfutures", "100ms")})),
			binance(list([]string{savedSubscription("ticker", "spot", ""), savedSubscription("ticker", "usdtmarginedfutures", ""), savedSubscription("orderbook", "usdtmarginedfutures", "100ms")})),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v25.Version).DowngradeExchange(t.Context(), []byte(tc.in))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.Equal(t, tc.exp, string(out), "DowngradeExchange should return the expected configuration")
		})
	}
	for name, in := range customised(currentDefaults) {
		t.Run("customised: "+name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v25.Version).DowngradeExchange(t.Context(), []byte(in))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.Equal(t, in, string(out), "DowngradeExchange should leave the configuration as saved")
		})
	}
}
