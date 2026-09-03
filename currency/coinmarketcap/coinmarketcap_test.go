package coinmarketcap

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Please set API keys to test endpoint
const (
	apikey              = ""
	apiAccountPlanLevel = ""
)

func skipIfLiveCredentialsUnavailable(t *testing.T, c *Coinmarketcap, minAllowable uint8) {
	t.Helper()
	switch {
	case apiAccountPlanLevel != "" && apikey != "":
		if err := c.CheckAccountPlan(minAllowable); err != nil {
			t.Skip("CoinMarketCap account plan not allowed for function, please review or upgrade plan to test")
		}
		return
	default:
		t.Skip("CoinMarketCap API key or account plan not set")
	}
}

func newConfiguredClient(t *testing.T) *Coinmarketcap {
	t.Helper()

	c := &Coinmarketcap{}
	c.SetDefaults()
	plan := apiAccountPlanLevel
	if plan == "" {
		plan = "basic"
	}
	cfg := Settings{
		APIKey:      apikey,
		AccountPlan: plan,
		Enabled:     true,
	}
	err := c.Setup(cfg)
	require.NoError(t, err, "Setup must configure the client")
	return c
}

func newSyntheticClient(t *testing.T, responses map[string]string) (client *Coinmarketcap, closeFn func()) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, ok := responses[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"status":{"error_code":404,"error_message":"not found"}}`))
			return
		}
		_, _ = w.Write([]byte(resp))
	}))
	c := &Coinmarketcap{}
	c.SetDefaults()
	c.APIUrl = server.URL
	c.APIkey = "test"
	c.Plan = Enterprise
	return c, server.Close
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var c Coinmarketcap
		c.SetDefaults()
		assert.Equal(t, "CoinMarketCap", c.Name, "SetDefaults should set the name")
		assert.Equal(t, baseURL, c.APIUrl, "SetDefaults should set the default URL")
		assert.Empty(t, c.APIkey, "SetDefaults should not populate the API key")
		require.NotNil(t, c.Requester, "SetDefaults must populate the requester")

		definitions := c.Requester.GetRateLimiterDefinitions()
		require.Len(t, definitions, 6, "SetDefaults must configure all account-plan rate limits")
		limiter := definitions[basicEPL]
		require.NotNil(t, limiter, "SetDefaults must configure the request limiter")
		start := time.Now()
		require.NoError(t, limiter.RateLimit(t.Context()), "RateLimit must allow the first request")
		require.NoError(t, limiter.RateLimit(t.Context()), "RateLimit must allow the second request")
		assert.Equal(t, rateInterval/time.Duration(basicRequestRate), time.Since(start), "SetDefaults should use the Basic request rate")

		var other Coinmarketcap
		other.SetDefaults()
		assert.NotSame(t, limiter, other.Requester.GetRateLimiterDefinitions()[basicEPL], "SetDefaults should configure independent client rate limits")
	})
}

func TestSetup(t *testing.T) {
	t.Parallel()
	var c Coinmarketcap
	c.SetDefaults()
	cfg := Settings{
		APIKey:      apikey,
		AccountPlan: apiAccountPlanLevel,
		Enabled:     true,
	}
	if cfg.AccountPlan == "" {
		cfg.AccountPlan = "basic"
	}

	err := c.Setup(cfg)
	require.NoError(t, err)
}

func TestCheckAccountPlan(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		plan      uint8
		min       uint8
		expectErr bool
	}{
		{name: "basic allows basic", plan: Basic, min: Basic},
		{name: "basic blocks builder", plan: Basic, min: Builder, expectErr: true},
		{name: "builder allows builder", plan: Builder, min: Builder},
		{name: "startup allows builder", plan: Startup, min: Builder},
		{name: "startup blocks growth", plan: Startup, min: Growth, expectErr: true},
		{name: "growth allows growth", plan: Growth, min: Growth},
		{name: "enterprise allows professional", plan: Enterprise, min: Professional},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &Coinmarketcap{Plan: tc.plan}
			err := c.CheckAccountPlan(tc.min)
			if tc.expectErr {
				assert.ErrorIs(t, err, errFunctionUseNotAllowed, "CheckAccountPlan should return expected error")
				return
			}
			assert.NoError(t, err, "CheckAccountPlan should not error")
		})
	}
}

func TestGetCryptocurrencyInfo(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetCryptocurrencyInfo(1)
	assert.NoError(t, err)
}

func TestGetCryptocurrencyIDMap(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	data, err := c.GetCryptocurrencyIDMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestGetCryptocurrencyHistoricalListings(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	_, err := c.GetCryptocurrencyHistoricalListings()
	assert.Error(t, err)
}

func TestGetCryptocurrencyLatestListing(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetCryptocurrencyLatestListing(0, 0)
	assert.NoError(t, err)
}

func TestGetCryptocurrencyLatestListingDecodesV3Payload(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v3/cryptocurrency/listings/latest": `{
			"data":[{
				"id":1,
				"name":"Bitcoin",
				"symbol":"BTC",
				"slug":"bitcoin",
				"platform":{"id":1027,"slug":"ethereum","name":"Ethereum","symbol":"ETH","token_address":"0xabc"},
				"quote":[{
					"id":2781,
					"symbol":"USD",
					"price":1,
					"volume_24h":2,
					"cex_volume_24h":3,
					"dex_volume_24h":4,
					"volume_24h_reported":5,
					"volume_7d":6,
					"volume_7d_reported":7,
					"volume_30d":8,
					"volume_30d_reported":9,
					"volume_change_24h":10,
					"percent_change_1h":11,
					"percent_change_24h":12,
					"percent_change_7d":13,
					"percent_change_30d":14,
					"percent_change_60d":15,
					"percent_change_90d":16,
					"market_cap":17,
					"market_cap_dominance":18,
					"fully_diluted_market_cap":19,
					"minted_market_cap":20,
					"tvl":null,
					"market_cap_by_total_supply":22,
					"last_updated":"2026-06-19T14:02:00Z"
				}],
				"tags":["mineable"],
				"is_active":1,
				"infinite_supply":true,
				"is_market_cap_included_in_calc":1,
				"is_fiat":1,
				"circulating_supply":100,
				"total_supply":110,
				"max_supply":null,
				"date_added":"2010-07-13T00:00:00Z",
				"num_market_pairs":140,
				"cmc_rank":1,
				"last_updated":"2026-06-19T14:02:00Z",
				"tvl_ratio":1.5,
				"self_reported_circulating_supply":150,
				"self_reported_market_cap":160,
				"unlocked_circulating_supply":170,
				"unlocked_market_cap":180,
				"minted_market_cap":130
			}],
			"status":{"error_code":"0","error_message":"","notice":"synthetic"}
		}`,
	})
	t.Cleanup(closeFn)

	result, err := c.GetCryptocurrencyLatestListing(1, 1)
	require.NoError(t, err, "GetCryptocurrencyLatestListing must not error")
	expected := []CryptocurrencyLatestListings{
		{
			ID:                            1,
			Name:                          "Bitcoin",
			Symbol:                        "BTC",
			Slug:                          "bitcoin",
			Platform:                      []byte(`{"id":1027,"slug":"ethereum","name":"Ethereum","symbol":"ETH","token_address":"0xabc"}`),
			Tags:                          []byte(`["mineable"]`),
			IsActive:                      1,
			InfiniteSupply:                true,
			IsMarketCapIncludedInCalc:     1,
			IsFiat:                        1,
			CirculatingSupply:             100,
			TotalSupply:                   110,
			MaxSupply:                     0,
			DateAdded:                     time.Date(2010, 7, 13, 0, 0, 0, 0, time.UTC),
			NumMarketPairs:                140,
			CmcRank:                       1,
			LastUpdated:                   time.Date(2026, 6, 19, 14, 2, 0, 0, time.UTC),
			TVLRatio:                      1.5,
			SelfReportedCirculatingSupply: 150,
			SelfReportedMarketCap:         160,
			UnlockedCirculatingSupply:     170,
			UnlockedMarketCap:             180,
			MintedMarketCap:               130,
			Quote: CryptocurrencyLatestQuoteMap{
				"USD": {
					ID:                     2781,
					Symbol:                 "USD",
					Price:                  1,
					Volume24Hour:           2,
					CEXVolume24Hour:        3,
					DEXVolume24Hour:        4,
					Volume24HourReported:   5,
					Volume7Day:             6,
					Volume7DayReported:     7,
					Volume30Day:            8,
					Volume30DayReported:    9,
					VolumeChange24Hour:     10,
					PercentChange1Hour:     11,
					PercentChange24Hour:    12,
					PercentChange7Day:      13,
					PercentChange30Day:     14,
					PercentChange60Day:     15,
					PercentChange90Day:     16,
					MarketCap:              17,
					MarketCapDominance:     18,
					FullyDilutedMarketCap:  19,
					MintedMarketCap:        20,
					MarketCapByTotalSupply: 22,
					LastUpdated:            time.Date(2026, 6, 19, 14, 2, 0, 0, time.UTC),
				},
			},
		},
	}
	assert.Equal(t, expected, result, "GetCryptocurrencyLatestListing should return the complete V3 payload")
}

func TestCryptocurrencyLatestListingsUnmarshalNullMetrics(t *testing.T) {
	t.Parallel()
	var result CryptocurrencyLatestListings
	err := json.Unmarshal([]byte(`{
		"max_supply":null,
		"tvl_ratio":null,
		"self_reported_circulating_supply":null,
		"self_reported_market_cap":null,
		"unlocked_circulating_supply":null,
		"unlocked_market_cap":null,
		"minted_market_cap":null
	}`), &result)
	require.NoError(t, err, "Unmarshal must not error for null latest-listing metrics")
	assert.Equal(t, CryptocurrencyLatestListings{}, result, "Unmarshal should leave null latest-listing metrics at zero values")
}

func TestGetCryptocurrencyLatestMarketPairs(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Growth)
	_, err := c.GetCryptocurrencyLatestMarketPairs(1, 0, 0)
	assert.NoError(t, err)
}

func TestGetCryptocurrencyOHLCHistorical(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Startup)
	_, err := c.GetCryptocurrencyOHLCHistorical(1, time.Now(), time.Now())
	assert.NoError(t, err)
}

func TestGetCryptocurrencyOHLCLatest(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Startup)
	_, err := c.GetCryptocurrencyOHLCLatest(1)
	assert.NoError(t, err)
}

func TestGetCryptocurrencyLatestQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetCryptocurrencyLatestQuotes(1)
	assert.NoError(t, err)
}

func TestGetCryptocurrencyLatestQuotesDecodesV3Payload(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v3/cryptocurrency/quotes/latest": `{
			"data":[{
				"id":1,
				"name":"Bitcoin",
				"symbol":"BTC",
				"slug":"bitcoin",
				"is_active":1,
				"infinite_supply":false,
				"is_market_cap_included_in_calc":1,
				"is_fiat":0,
				"circulating_supply":100,
				"total_supply":110,
				"max_supply":120,
				"date_added":"2010-07-13T00:00:00Z",
				"num_market_pairs":140,
				"cmc_rank":1,
				"last_updated":"2026-06-19T14:02:00Z",
				"tvl_ratio":1.5,
				"self_reported_circulating_supply":150,
				"self_reported_market_cap":160,
				"unlocked_circulating_supply":170,
				"unlocked_market_cap":180,
				"tags":[{"slug":"mineable"}],
				"platform":null,
				"quote":[{
					"id":2781,
					"symbol":"USD",
					"price":1,
					"volume_24h":2,
					"cex_volume_24h":3,
					"dex_volume_24h":4,
					"volume_24h_reported":5,
					"volume_7d":6,
					"volume_7d_reported":7,
					"volume_30d":8,
					"volume_30d_reported":9,
					"volume_change_24h":10,
					"percent_change_1h":11,
					"percent_change_24h":12,
					"percent_change_7d":13,
					"percent_change_30d":14,
					"percent_change_60d":15,
					"percent_change_90d":16,
					"market_cap":17,
					"market_cap_dominance":18,
					"fully_diluted_market_cap":19,
					"minted_market_cap":20,
					"tvl":null,
					"market_cap_by_total_supply":22,
					"last_updated":"2026-06-19T14:02:00Z"
				}]
			}],
			"status":{"error_code":"0","error_message":""}
		}`,
	})
	t.Cleanup(closeFn)

	result, err := c.GetCryptocurrencyLatestQuotes(1)
	require.NoError(t, err, "GetCryptocurrencyLatestQuotes must not error")
	expected := CryptocurrencyLatestQuotes{
		{
			ID:                            1,
			Name:                          "Bitcoin",
			Symbol:                        "BTC",
			Slug:                          "bitcoin",
			IsActive:                      1,
			InfiniteSupply:                false,
			IsMarketCapIncludedInCalc:     1,
			IsFiat:                        0,
			CirculatingSupply:             100,
			TotalSupply:                   110,
			MaxSupply:                     120,
			DateAdded:                     time.Date(2010, 7, 13, 0, 0, 0, 0, time.UTC),
			NumMarketPairs:                140,
			CmcRank:                       1,
			LastUpdated:                   time.Date(2026, 6, 19, 14, 2, 0, 0, time.UTC),
			TVLRatio:                      1.5,
			SelfReportedCirculatingSupply: 150,
			SelfReportedMarketCap:         160,
			UnlockedCirculatingSupply:     170,
			UnlockedMarketCap:             180,
			MintedMarketCap:               0,
			Tags:                          []byte(`[{"slug":"mineable"}]`),
			Platform:                      []byte(`null`),
			Quote: CryptocurrencyLatestQuoteMap{
				"USD": {
					ID:                     2781,
					Symbol:                 "USD",
					Price:                  1,
					Volume24Hour:           2,
					CEXVolume24Hour:        3,
					DEXVolume24Hour:        4,
					Volume24HourReported:   5,
					Volume7Day:             6,
					Volume7DayReported:     7,
					Volume30Day:            8,
					Volume30DayReported:    9,
					VolumeChange24Hour:     10,
					PercentChange1Hour:     11,
					PercentChange24Hour:    12,
					PercentChange7Day:      13,
					PercentChange30Day:     14,
					PercentChange60Day:     15,
					PercentChange90Day:     16,
					MarketCap:              17,
					MarketCapDominance:     18,
					FullyDilutedMarketCap:  19,
					MintedMarketCap:        20,
					MarketCapByTotalSupply: 22,
					LastUpdated:            time.Date(2026, 6, 19, 14, 2, 0, 0, time.UTC),
				},
			},
		},
	}
	assert.Equal(t, expected, result, "GetCryptocurrencyLatestQuotes should return the complete V3 payload")
}

func TestGetCryptocurrencyHistoricalQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Now(), time.Now())
	assert.NoError(t, err)
}

func TestGetCryptocurrencyHistoricalQuotesDecodesResultMap(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v3/cryptocurrency/quotes/historical": `{"data":{"1":{"id":1,"name":"Bitcoin","symbol":"BTC","quotes":[{"timestamp":"2018-06-22T00:00:00Z","quote":{"USD":{"price":6242.48}}}]}},"status":{"error_code":0}}`,
	})
	t.Cleanup(closeFn)

	result, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Unix(1, 0), time.Unix(2, 0))
	require.NoError(t, err, "GetCryptocurrencyHistoricalQuotes must not error")
	assert.Equal(t, int64(1), result.ID, "GetCryptocurrencyHistoricalQuotes should return the correct ID")
	require.Len(t, result.Quotes, 1, "GetCryptocurrencyHistoricalQuotes must return one quote")
	assert.Equal(t, 6242.48, result.Quotes[0].Quote.USD.Price, "GetCryptocurrencyHistoricalQuotes should return the correct USD price")
}

func TestGetCryptocurrencyHistoricalQuotesRejectsMissingResult(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v3/cryptocurrency/quotes/historical": `{"data":{"2":{"id":2}},"status":{"error_code":0}}`,
	})
	t.Cleanup(closeFn)

	_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Unix(1, 0), time.Unix(2, 0))
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetCryptocurrencyHistoricalQuotes should return common.ErrNoResponse when the requested ID is absent")
}

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetExchangeInfo(1)
	assert.NoError(t, err)
}

func TestGetExchangeInfoAllowsBasicPlan(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v1/exchange/info": `{"data":{"1":{"id":270,"name":"Binance","slug":"binance"}},"status":{"error_code":0}}`,
	})
	t.Cleanup(closeFn)
	c.Plan = Basic

	result, err := c.GetExchangeInfo(1)
	require.NoError(t, err, "GetExchangeInfo must not error")
	require.Contains(t, result, "1", "GetExchangeInfo must return the requested exchange")
	assert.Equal(t, int64(270), result["1"].ID, "GetExchangeInfo should return the correct ID")
}

func TestGetExchangeMap(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetExchangeMap(0, 0)
	assert.NoError(t, err)
}

func TestGetExchangeMapAllowsBasicPlan(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v1/exchange/map": `{"data":[{"id":270,"name":"Binance","slug":"binance","is_active":1}],"status":{"error_code":0}}`,
	})
	t.Cleanup(closeFn)
	c.Plan = Basic

	result, err := c.GetExchangeMap(1, 10)
	require.NoError(t, err, "GetExchangeMap must not error")
	require.Len(t, result, 1, "GetExchangeMap must return one exchange")
	assert.Equal(t, int64(270), result[0].ID, "GetExchangeMap should return the correct ID")
}

func TestGetExchangeHistoricalListings(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	_, err := c.GetExchangeHistoricalListings()
	// TODO: update this once the feature above is implemented
	assert.ErrorIs(t, err, errEndpointNotAvailable, "GetExchangeHistoricalListings should return expected error")
}

func TestGetExchangeLatestListings(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	_, err := c.GetExchangeLatestListings()
	// TODO: update this once the feature above is implemented
	assert.ErrorIs(t, err, errEndpointNotAvailable, "GetExchangeLatestListings should return expected error")
}

func TestGetExchangeLatestMarketPairs(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Growth)
	_, err := c.GetExchangeLatestMarketPairs(1, 0, 0)
	assert.NoError(t, err)
}

func TestGetExchangeLatestQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Growth)
	_, err := c.GetExchangeLatestQuotes(1)
	assert.NoError(t, err)
}

func TestGetExchangeLatestQuotesDecodesResultMap(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v1/exchange/quotes/latest": `{"data":{"1":{"id":270,"name":"Binance","slug":"binance","quote":{"USD":{"volume_24h":768478308.52}}},"2":{"id":89,"name":"Coinbase Exchange","slug":"coinbase-exchange","quote":{"USD":{"volume_24h":1234.5}}}},"status":{"error_code":0}}`,
	})
	t.Cleanup(closeFn)

	result, err := c.GetExchangeLatestQuotes(1)
	require.NoError(t, err, "GetExchangeLatestQuotes must not error")
	assert.Equal(t, int64(270), result.Binance.ID, "GetExchangeLatestQuotes should populate Binance correctly")
	require.Len(t, result.Exchanges, 2, "GetExchangeLatestQuotes must return every exchange")
	assert.Equal(t, "Coinbase Exchange", result.Exchanges["2"].Name, "GetExchangeLatestQuotes should return the correct exchange name")
	assert.Equal(t, 1234.5, result.Exchanges["2"].Quote["USD"].Volume24Hour, "GetExchangeLatestQuotes should return the correct exchange volume")
}

func TestGetExchangeLatestQuotesWithoutBinance(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v1/exchange/quotes/latest": `{"data":{"2":{"id":89,"name":"Coinbase Exchange","slug":"coinbase-exchange"}},"status":{"error_code":0}}`,
	})
	t.Cleanup(closeFn)

	result, err := c.GetExchangeLatestQuotes(2)
	require.NoError(t, err, "GetExchangeLatestQuotes must not error")
	require.Len(t, result.Exchanges, 1, "GetExchangeLatestQuotes must return every exchange")
	assert.Zero(t, result.Binance, "GetExchangeLatestQuotes should leave Binance empty when absent")
}

func TestGetExchangeHistoricalQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetExchangeHistoricalQuotes(1, time.Now(), time.Now())
	assert.NoError(t, err)
}

func TestGetExchangeHistoricalQuotesDecodesResultMap(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v1/exchange/quotes/historical": `{"data":{"1":{"id":270,"name":"Binance","slug":"binance","quotes":[{"timestamp":"2018-06-03T00:00:00Z","quote":{"USD":{"volume_24h":1632390000}},"num_market_pairs":338}]}},"status":{"error_code":0}}`,
	})
	t.Cleanup(closeFn)

	result, err := c.GetExchangeHistoricalQuotes(1, time.Unix(1, 0), time.Unix(2, 0))
	require.NoError(t, err, "GetExchangeHistoricalQuotes must not error")
	assert.Equal(t, int64(270), result.ID, "GetExchangeHistoricalQuotes should return the correct ID")
	require.Len(t, result.Quotes, 1, "GetExchangeHistoricalQuotes must return one quote")
	assert.Equal(t, 1632390000.0, result.Quotes[0].Quote["USD"].Volume24Hour, "GetExchangeHistoricalQuotes should return the correct volume")
}

func TestGetExchangeHistoricalQuotesRejectsMissingResult(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v1/exchange/quotes/historical": `{"data":{"2":{"id":2}},"status":{"error_code":0}}`,
	})
	t.Cleanup(closeFn)

	_, err := c.GetExchangeHistoricalQuotes(1, time.Unix(1, 0), time.Unix(2, 0))
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetExchangeHistoricalQuotes should return common.ErrNoResponse when the requested ID is absent")
}

func TestGetGlobalMeticLatestQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetGlobalMeticLatestQuotes()
	assert.NoError(t, err)
}

func TestGetGlobalMeticHistoricalQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetGlobalMeticHistoricalQuotes(time.Now(), time.Now())
	assert.NoError(t, err)
}

func TestGetPriceConversion(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Builder)
	_, err := c.GetPriceConversion(0, 1, time.Now())
	assert.NoError(t, err)
}

func TestGetPriceConversionDecodesNumericID(t *testing.T) {
	t.Parallel()
	c, closeFn := newSyntheticClient(t, map[string]string{
		"/v2/tools/price-conversion": `{"data":{"symbol":"BTC","id":1,"name":"Bitcoin","amount":50,"last_updated":"2018-06-06T08:04:36Z","quote":{"USD":{"price":284656.08}}},"status":{"error_code":0}}`,
	})
	t.Cleanup(closeFn)

	result, err := c.GetPriceConversion(50, 1, time.Time{})
	require.NoError(t, err, "GetPriceConversion must not error")
	assert.Equal(t, int64(1), result.ID, "GetPriceConversion should return the correct ID")
	assert.Equal(t, 284656.08, result.Quote["USD"].Price, "GetPriceConversion should return the correct price")
}

func TestGetPriceConversionPlanAccess(t *testing.T) {
	t.Parallel()
	responses := map[string]string{
		"/v2/tools/price-conversion": `{"data":{"symbol":"BTC","id":1,"name":"Bitcoin","amount":1,"quote":{"USD":{"price":2}}},"status":{"error_code":0}}`,
	}
	c, closeFn := newSyntheticClient(t, responses)
	t.Cleanup(closeFn)

	c.Plan = 0
	_, err := c.GetPriceConversion(1, 1, time.Time{})
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetPriceConversion should return errFunctionUseNotAllowed for an unset plan")

	c.Plan = Basic
	_, err = c.GetPriceConversion(1, 1, time.Time{})
	require.NoError(t, err, "GetPriceConversion must not error")

	_, err = c.GetPriceConversion(1, 1, time.Now())
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetPriceConversion should return errFunctionUseNotAllowed for Basic historical conversion")

	historicalClient, historicalCloseFn := newSyntheticClient(t, responses)
	t.Cleanup(historicalCloseFn)
	historicalClient.Plan = Builder
	_, err = historicalClient.GetPriceConversion(1, 1, time.Now())
	assert.NoError(t, err, "GetPriceConversion should not error")
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		plan        uint8
		rateLimit   request.EndpointLimit
		values      url.Values
		expectedURI string
	}{
		{name: "unset", rateLimit: basicEPL, expectedURI: "/test"},
		{name: "basic with query", plan: Basic, rateLimit: basicEPL, values: url.Values{"a": {"b"}}, expectedURI: "/test?a=b"},
		{name: "builder", plan: Builder, rateLimit: builderEPL, expectedURI: "/test"},
		{name: "startup", plan: Startup, rateLimit: startupEPL, expectedURI: "/test"},
		{name: "growth", plan: Growth, rateLimit: growthEPL, expectedURI: "/test"},
		{name: "professional", plan: Professional, rateLimit: professionalEPL, expectedURI: "/test"},
		{name: "enterprise", plan: Enterprise, rateLimit: enterpriseEPL, expectedURI: "/test"},
		{name: "unknown", plan: 255, rateLimit: basicEPL, expectedURI: "/test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			received := make(chan []string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received <- []string{r.Method, r.URL.RequestURI(), r.Header.Get("X-CMC_PRO_API_KEY"), r.Header.Get("Accept")}
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(server.Close)

			var c Coinmarketcap
			c.SetDefaults()
			c.APIUrl = server.URL
			c.APIkey = "test-key"
			c.Plan = tc.plan
			definitions := c.Requester.GetRateLimiterDefinitions()
			for key := range definitions {
				if key != tc.rateLimit {
					delete(definitions, key)
				}
			}

			result := struct {
				OK bool `json:"ok"`
			}{}
			err := c.SendHTTPRequest(http.MethodGet, "test", tc.values, &result)
			require.NoError(t, err, "SendHTTPRequest must use the selected plan rate limit")
			assert.True(t, result.OK, "SendHTTPRequest should decode the response")
			assert.Equal(t, []string{http.MethodGet, tc.expectedURI, "test-key", "application/json"}, <-received, "SendHTTPRequest should send the correct request")
		})
	}

	t.Run("missing selected rate limit", func(t *testing.T) {
		t.Parallel()
		var c Coinmarketcap
		c.SetDefaults()
		c.Plan = Growth
		delete(c.Requester.GetRateLimiterDefinitions(), growthEPL)

		err := c.SendHTTPRequest(http.MethodGet, "test", nil, nil)
		require.ErrorIs(t, err, common.ErrNilPointer, "SendHTTPRequest must reject a missing selected rate limit")
		assert.ErrorContains(t, err, "failed to rate limit HTTP request", "SendHTTPRequest should return the rate-limit failure")
	})

	t.Run("nil requester", func(t *testing.T) {
		t.Parallel()
		c := Coinmarketcap{APIUrl: "https://example.invalid"}
		err := c.SendHTTPRequest(http.MethodGet, "test", nil, nil)
		assert.ErrorIs(t, err, request.ErrRequestSystemIsNil, "SendHTTPRequest should reject a nil requester")
	})

	t.Run("nil rate limit definitions", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		t.Cleanup(server.Close)

		requester, err := request.New("CoinMarketCap", common.NewHTTPClientWithTimeout(defaultTimeOut))
		require.NoError(t, err, "request.New must create a requester without rate limits")
		c := Coinmarketcap{APIUrl: server.URL, Plan: Growth, Requester: requester}
		err = c.SendHTTPRequest(http.MethodGet, "test", nil, nil)
		assert.NoError(t, err, "SendHTTPRequest should support explicitly disabled rate limiting")
	})
}

func TestSetAccountPlan(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                string
		accountPlan         string
		expected            uint8
		expectedRateLimit   request.EndpointLimit
		expectedRequestRate int
		err                 error
	}{
		{name: "basic", accountPlan: "basic", expected: Basic, expectedRateLimit: basicEPL, expectedRequestRate: basicRequestRate},
		{name: "builder", accountPlan: "builder", expected: Builder, expectedRateLimit: builderEPL, expectedRequestRate: builderRequestRate},
		{name: "startup", accountPlan: "startup", expected: Startup, expectedRateLimit: startupEPL, expectedRequestRate: startupRequestRate},
		{name: "growth", accountPlan: "growth", expected: Growth, expectedRateLimit: growthEPL, expectedRequestRate: growthRequestRate},
		{name: "professional", accountPlan: "professional", expected: Professional, expectedRateLimit: professionalEPL, expectedRequestRate: professionalRequestRate},
		{name: "enterprise", accountPlan: "enterprise", expected: Enterprise, expectedRateLimit: enterpriseEPL, expectedRequestRate: enterpriseRequestRate},
		{name: "normalised", accountPlan: " Growth ", expected: Growth, expectedRateLimit: growthEPL, expectedRequestRate: growthRequestRate},
		{name: "empty", expectedRateLimit: enterpriseEPL, expectedRequestRate: enterpriseRequestRate, err: errInvalidAccountPlan},
		{name: "legacy hobbyist", accountPlan: "hobbyist", expectedRateLimit: enterpriseEPL, expectedRequestRate: enterpriseRequestRate, err: errInvalidAccountPlan},
		{name: "legacy standard", accountPlan: "standard", expectedRateLimit: enterpriseEPL, expectedRequestRate: enterpriseRequestRate, err: errInvalidAccountPlan},
		{name: "unknown", accountPlan: "unknown", expectedRateLimit: enterpriseEPL, expectedRequestRate: enterpriseRequestRate, err: errInvalidAccountPlan},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				c := Coinmarketcap{Plan: Enterprise}
				c.SetDefaults()
				limiter := c.Requester.GetRateLimiterDefinitions()[tc.expectedRateLimit]
				err := c.SetAccountPlan(tc.accountPlan)
				if tc.err != nil {
					require.ErrorIs(t, err, tc.err, "SetAccountPlan must error correctly for invalid or obsolete plans")
					assert.Equal(t, Enterprise, c.Plan, "SetAccountPlan should not change Plan on error")
				} else {
					require.NoError(t, err, "SetAccountPlan must not error")
					assert.Equal(t, tc.expected, c.Plan, "SetAccountPlan should set Plan correctly")
				}

				assert.Equal(t, tc.expectedRateLimit, planRateLimit(c.Plan), "SetAccountPlan should select the correct tier rate limit")
				assert.Same(t, limiter, c.Requester.GetRateLimiterDefinitions()[tc.expectedRateLimit], "SetAccountPlan should retain the tier limiter")
				require.NotNil(t, limiter, "SetAccountPlan must retain the request limiter")
				start := time.Now()
				require.NoError(t, limiter.RateLimit(t.Context()), "RateLimit must allow the first request")
				require.NoError(t, limiter.RateLimit(t.Context()), "RateLimit must allow the second request")
				assert.Equal(t, rateInterval/time.Duration(tc.expectedRequestRate), time.Since(start), "SetAccountPlan should configure the expected request rate")
			})
		})
	}
}

func TestSetAccountPlanWithoutRequester(t *testing.T) {
	t.Parallel()
	c := Coinmarketcap{Plan: Enterprise}
	err := c.SetAccountPlan("builder")
	require.NoError(t, err, "SetAccountPlan must support an uninitialised requester")
	assert.Equal(t, Builder, c.Plan, "SetAccountPlan should set Plan without a requester")
	assert.Equal(t, builderEPL, planRateLimit(c.Plan), "SetAccountPlan should select the correct tier rate limit")
}

func TestSetAccountPlanWithoutRateLimiter(t *testing.T) {
	t.Parallel()

	t.Run("nil definitions", func(t *testing.T) {
		t.Parallel()
		c := Coinmarketcap{Plan: Enterprise, Requester: new(request.Requester)}
		err := c.SetAccountPlan("builder")
		require.ErrorIs(t, err, errRateLimiterNotSet, "SetAccountPlan must reject nil rate-limit definitions")
		assert.ErrorIs(t, err, common.ErrNilPointer, "SetAccountPlan should wrap the underlying nil limiter error")
		assert.Equal(t, Enterprise, c.Plan, "SetAccountPlan should not change Plan when rate-limit definitions are nil")
	})

	t.Run("missing selected definition", func(t *testing.T) {
		t.Parallel()
		c := Coinmarketcap{Plan: Enterprise}
		c.SetDefaults()
		delete(c.Requester.GetRateLimiterDefinitions(), builderEPL)
		err := c.SetAccountPlan("builder")
		require.ErrorIs(t, err, errRateLimiterNotSet, "SetAccountPlan must reject a missing selected rate limit")
		assert.ErrorIs(t, err, common.ErrNilPointer, "SetAccountPlan should wrap the underlying nil limiter error")
		assert.Equal(t, Enterprise, c.Plan, "SetAccountPlan should not change Plan when the selected rate limit is missing")
	})
}

func TestSetAccountPlanPreservesRateLimitBudget(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var c Coinmarketcap
		c.SetDefaults()
		require.NoError(t, c.SetAccountPlan("basic"), "SetAccountPlan must select the Basic plan")
		limiter := c.Requester.GetRateLimiterDefinitions()[basicEPL]
		start := time.Now()
		require.NoError(t, limiter.RateLimit(t.Context()), "RateLimit must allow the first Basic request")
		require.NoError(t, c.SetAccountPlan("builder"), "SetAccountPlan must select the Builder plan")
		require.NoError(t, c.SetAccountPlan("basic"), "SetAccountPlan must reselect the Basic plan")
		require.NoError(t, limiter.RateLimit(t.Context()), "RateLimit must allow the second Basic request")
		assert.Equal(t, rateInterval/time.Duration(basicRequestRate), time.Since(start), "SetAccountPlan should preserve the Basic rate-limit budget")
	})
}

func TestNewFromSettingsAndSetupDisabled(t *testing.T) {
	t.Parallel()
	cfg := Settings{Enabled: true, Verbose: true, AccountPlan: "basic", APIKey: "x"}
	client, err := NewFromSettings(cfg)
	require.NoError(t, err, "NewFromSettings must configure an enabled client")
	assert.True(t, client.Enabled, "NewFromSettings should set Enabled correctly")
	assert.True(t, client.Verbose, "Setup should set Verbose correctly")
	assert.Equal(t, "x", client.APIkey, "Setup should set APIkey correctly")
	assert.Equal(t, Basic, client.Plan, "NewFromSettings should set Plan correctly")

	var disabled Coinmarketcap
	disabled.SetDefaults()
	err = disabled.Setup(Settings{Enabled: false})
	require.NoError(t, err, "Setup must accept a disabled client")
	assert.False(t, disabled.Enabled, "Setup should leave the client disabled")
}

func TestSetupRejectsInvalidPlanWithoutMutation(t *testing.T) {
	t.Parallel()
	client := Coinmarketcap{
		Enabled: true,
		Verbose: true,
		APIkey:  "old-api-key",
		Plan:    Enterprise,
	}

	err := client.Setup(Settings{
		Enabled:     true,
		Verbose:     false,
		APIKey:      "new-api-key",
		AccountPlan: "invalid",
	})
	require.ErrorIs(t, err, errInvalidAccountPlan, "Setup must reject an invalid account plan")
	assert.True(t, client.Enabled, "Setup should not change Enabled on error")
	assert.True(t, client.Verbose, "Setup should not change Verbose on error")
	assert.Equal(t, "old-api-key", client.APIkey, "Setup should not change APIkey on error")
	assert.Equal(t, Enterprise, client.Plan, "Setup should not change Plan on error")
}

func TestQuoteMapUnmarshal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		input    string
		initial  QuoteMap
		expected QuoteMap
		wantErr  bool
	}{
		{
			name:    "object merges with receiver",
			input:   `{"USD":{"price":1.23},"BTC":{"price":0.1}}`,
			initial: QuoteMap{"OLD": {Price: 9}},
			expected: QuoteMap{
				"OLD": {Price: 9},
				"USD": {Price: 1.23},
				"BTC": {Price: 0.1},
			},
		},
		{
			name:     "empty object preserves receiver",
			input:    `{}`,
			initial:  QuoteMap{"OLD": {Price: 9}},
			expected: QuoteMap{"OLD": {Price: 9}},
		},
		{
			name:  "array of maps on fresh receiver",
			input: `[{"USD":{"price":2.34}},{"ETH":{"price":3.45}}]`,
			expected: QuoteMap{
				"USD": {Price: 2.34},
				"ETH": {Price: 3.45},
			},
		},
		{
			name:     "array merges with receiver",
			input:    `[{"USD":{"price":1.23}}]`,
			initial:  QuoteMap{"BTC": {Price: 0.1}},
			expected: QuoteMap{"BTC": {Price: 0.1}, "USD": {Price: 1.23}},
		},
		{
			name:     "array duplicate keys use last value",
			input:    `[{"USD":{"price":1.23}},{"USD":{"price":2.34}}]`,
			expected: QuoteMap{"USD": {Price: 2.34}},
		},
		{name: "empty array", input: `[]`},
		{
			name:     "empty array preserves receiver",
			input:    `[]`,
			initial:  QuoteMap{"OLD": {Price: 9}},
			expected: QuoteMap{"OLD": {Price: 9}},
		},
		{name: "null clears receiver", input: `null`, initial: QuoteMap{"OLD": {Price: 9}}},
		{
			name:     "malformed object preserves receiver",
			input:    `{"USD":{"price":"invalid"}}`,
			initial:  QuoteMap{"OLD": {Price: 9}},
			expected: QuoteMap{"OLD": {Price: 9}},
			wantErr:  true,
		},
		{
			name:     "partially invalid array preserves receiver",
			input:    `[{"USD":{"price":1.23}},{"BTC":{"price":"invalid"}}]`,
			initial:  QuoteMap{"OLD": {Price: 9}},
			expected: QuoteMap{"OLD": {Price: 9}},
			wantErr:  true,
		},
		{name: "invalid collection", input: `true`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := maps.Clone(tc.initial)
			err := actual.UnmarshalJSON([]byte(tc.input))
			if tc.wantErr {
				assert.ErrorIs(t, err, common.ErrInvalidResponse, "UnmarshalJSON should error correctly for invalid quote collections")
			} else {
				require.NoError(t, err, "UnmarshalJSON must not error")
			}
			assert.Equal(t, tc.expected, actual, "UnmarshalJSON should return the correct QuoteMap")
		})
	}
}

func TestCryptocurrencyLatestQuoteMapUnmarshal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		input    string
		initial  CryptocurrencyLatestQuoteMap
		expected CryptocurrencyLatestQuoteMap
		wantErr  bool
	}{
		{
			name:    "object replaces receiver",
			input:   `{"USD":{"id":2781,"symbol":"USD","price":1}}`,
			initial: CryptocurrencyLatestQuoteMap{"OLD": {Symbol: "OLD", Price: 9}},
			expected: CryptocurrencyLatestQuoteMap{
				"USD": {ID: 2781, Symbol: "USD", Price: 1},
			},
		},
		{name: "empty object", input: `{}`, expected: CryptocurrencyLatestQuoteMap{}},
		{
			name: "v3 array with all documented fields",
			input: `[{
				"id":2781,
				"symbol":"USD",
				"price":1,
				"volume_24h":2,
				"cex_volume_24h":3,
				"dex_volume_24h":4,
				"volume_24h_reported":5,
				"volume_7d":6,
				"volume_7d_reported":7,
				"volume_30d":8,
				"volume_30d_reported":9,
				"volume_change_24h":10,
				"percent_change_1h":11,
				"percent_change_24h":12,
				"percent_change_7d":13,
				"percent_change_30d":14,
				"percent_change_60d":15,
				"percent_change_90d":16,
				"market_cap":17,
				"market_cap_dominance":18,
				"fully_diluted_market_cap":19,
				"minted_market_cap":20,
				"tvl":21,
				"market_cap_by_total_supply":22,
				"last_updated":"2026-06-19T14:02:00Z"
			}]`,
			expected: CryptocurrencyLatestQuoteMap{
				"USD": {
					ID:                     2781,
					Symbol:                 "USD",
					Price:                  1,
					Volume24Hour:           2,
					CEXVolume24Hour:        3,
					DEXVolume24Hour:        4,
					Volume24HourReported:   5,
					Volume7Day:             6,
					Volume7DayReported:     7,
					Volume30Day:            8,
					Volume30DayReported:    9,
					VolumeChange24Hour:     10,
					PercentChange1Hour:     11,
					PercentChange24Hour:    12,
					PercentChange7Day:      13,
					PercentChange30Day:     14,
					PercentChange60Day:     15,
					PercentChange90Day:     16,
					MarketCap:              17,
					MarketCapDominance:     18,
					FullyDilutedMarketCap:  19,
					MintedMarketCap:        20,
					TVL:                    21,
					MarketCapByTotalSupply: 22,
					LastUpdated:            time.Date(2026, 6, 19, 14, 2, 0, 0, time.UTC),
				},
			},
		},
		{
			name:     "null tvl decodes to zero",
			input:    `[{"id":2781,"symbol":"USD","tvl":null}]`,
			expected: CryptocurrencyLatestQuoteMap{"USD": {ID: 2781, Symbol: "USD"}},
		},
		{name: "empty array", input: `[]`, expected: CryptocurrencyLatestQuoteMap{}},
		{
			name:    "null clears receiver",
			input:   `null`,
			initial: CryptocurrencyLatestQuoteMap{"OLD": {Symbol: "OLD", Price: 9}},
		},
		{name: "missing symbol", input: `[{"id":2781}]`, wantErr: true},
		{name: "empty symbol", input: `[{"id":2781,"symbol":""}]`, wantErr: true},
		{name: "whitespace symbol", input: `[{"id":2781,"symbol":" USD "}]`, wantErr: true},
		{name: "duplicate symbol", input: `[{"symbol":"USD"},{"symbol":"USD"}]`, wantErr: true},
		{name: "invalid element", input: `[{"id":"invalid","symbol":"USD"}]`, wantErr: true},
		{
			name:     "invalid second element preserves receiver",
			input:    `[{"id":2781,"symbol":"USD"},{"id":"invalid","symbol":"ETH"}]`,
			initial:  CryptocurrencyLatestQuoteMap{"OLD": {Symbol: "OLD", Price: 9}},
			expected: CryptocurrencyLatestQuoteMap{"OLD": {Symbol: "OLD", Price: 9}},
			wantErr:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := maps.Clone(tc.initial)
			err := actual.UnmarshalJSON([]byte(tc.input))
			if tc.wantErr {
				assert.ErrorIs(t, err, common.ErrInvalidResponse, "UnmarshalJSON should error correctly for invalid cryptocurrency quote collections")
			} else {
				require.NoError(t, err, "UnmarshalJSON must not error")
			}
			assert.Equal(t, tc.expected, actual, "UnmarshalJSON should return the correct CryptocurrencyLatestQuoteMap")
		})
	}
}

func TestStatusUnmarshal(t *testing.T) {
	t.Parallel()
	var status Status
	err := json.Unmarshal([]byte(`{
		"timestamp":"2026-06-19T14:03:33.664Z",
		"error_code":"0",
		"error_message":"",
		"elapsed":8,
		"credit_count":1,
		"notice":"synthetic"
	}`), &status)
	require.NoError(t, err, "Unmarshal must not error")
	assert.Equal(t, Status{
		Timestamp:   "2026-06-19T14:03:33.664Z",
		Elapsed:     8,
		CreditCount: 1,
		Notice:      "synthetic",
	}, status, "Unmarshal should return the complete API status payload")
}

func TestAPIErrorCodeUnmarshal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		input    string
		expected APIErrorCode
		wantErr  bool
	}{
		{name: "numeric zero", input: `0`},
		{name: "quoted zero", input: `"0"`},
		{name: "numeric nonzero", input: `123`, expected: 123},
		{name: "quoted nonzero", input: `"456"`, expected: 456},
		{name: "invalid quoted numeric", input: `"bad"`, wantErr: true},
		{name: "non-string JSON", input: `true`, wantErr: true},
		{name: "malformed JSON", input: `{`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var actual APIErrorCode
			err := actual.UnmarshalJSON([]byte(tc.input))
			if tc.wantErr {
				assert.Error(t, err, "UnmarshalJSON should reject an invalid API error code")
				return
			}
			require.NoError(t, err, "UnmarshalJSON must not error")
			assert.Equal(t, tc.expected, actual, "UnmarshalJSON should return the correct API error code")
		})
	}
}

func TestCoinmarketcapEndpointSuccessSynthetic(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		path    string
		payload string
		invoke  func(*Coinmarketcap) error
	}{
		{"GetCryptocurrencyInfo", "/v2/cryptocurrency/info", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyInfo(1); return err }},
		{"GetCryptocurrencyIDMap", "/v1/cryptocurrency/map", `{"data":[],"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyIDMap(); return err }},
		{"GetCryptocurrencyLatestListing", "/v3/cryptocurrency/listings/latest", `{"data":[],"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestListing(1, 2); return err }},
		{"GetCryptocurrencyLatestMarketPairs", "/v2/cryptocurrency/market-pairs/latest", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestMarketPairs(1, 1, 2); return err }},
		{"GetCryptocurrencyOHLCHistorical", "/v2/cryptocurrency/ohlcv/historical", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error {
			_, err := c.GetCryptocurrencyOHLCHistorical(1, time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetCryptocurrencyOHLCLatest", "/v2/cryptocurrency/ohlcv/latest", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyOHLCLatest(1); return err }},
		{"GetCryptocurrencyLatestQuotes", "/v3/cryptocurrency/quotes/latest", `{"data":[],"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestQuotes(1); return err }},
		{"GetCryptocurrencyHistoricalQuotes", "/v3/cryptocurrency/quotes/historical", `{"data":{"1":{}},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error {
			_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetExchangeInfo", "/v1/exchange/info", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeInfo(1); return err }},
		{"GetExchangeMap", "/v1/exchange/map", `{"data":[],"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeMap(1, 2); return err }},
		{"GetExchangeLatestMarketPairs", "/v1/exchange/market-pairs/latest", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeLatestMarketPairs(1, 1, 2); return err }},
		{"GetExchangeLatestQuotes", "/v1/exchange/quotes/latest", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeLatestQuotes(1); return err }},
		{"GetExchangeHistoricalQuotes", "/v1/exchange/quotes/historical", `{"data":{"1":{}},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error {
			_, err := c.GetExchangeHistoricalQuotes(1, time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetGlobalMeticLatestQuotes", "/v1/global-metrics/quotes/latest", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetGlobalMeticLatestQuotes(); return err }},
		{"GetGlobalMeticHistoricalQuotes", "/v1/global-metrics/quotes/historical", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error {
			_, err := c.GetGlobalMeticHistoricalQuotes(time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetPriceConversion", "/v2/tools/price-conversion", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetPriceConversion(1, 1, time.Now()); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, done := newSyntheticClient(t, map[string]string{tc.path: tc.payload})
			defer done()
			err := tc.invoke(client)
			require.NoErrorf(t, err, "%s must not error", tc.name)
		})
	}
}

func TestCoinmarketcapEndpointStatusErrorSynthetic(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		path    string
		payload string
		invoke  func(*Coinmarketcap) error
	}{
		{"GetCryptocurrencyInfo", "/v2/cryptocurrency/info", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyInfo(1); return err }},
		{"GetCryptocurrencyIDMap", "/v1/cryptocurrency/map", `{"data":[],"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyIDMap(); return err }},
		{"GetCryptocurrencyLatestListing", "/v3/cryptocurrency/listings/latest", `{"data":[],"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestListing(1, 2); return err }},
		{"GetCryptocurrencyLatestMarketPairs", "/v2/cryptocurrency/market-pairs/latest", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestMarketPairs(1, 1, 2); return err }},
		{"GetCryptocurrencyOHLCHistorical", "/v2/cryptocurrency/ohlcv/historical", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error {
			_, err := c.GetCryptocurrencyOHLCHistorical(1, time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetCryptocurrencyLatestQuotes", "/v3/cryptocurrency/quotes/latest", `{"data":[],"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestQuotes(1); return err }},
		{"GetCryptocurrencyHistoricalQuotes", "/v3/cryptocurrency/quotes/historical", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error {
			_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetExchangeInfo", "/v1/exchange/info", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeInfo(1); return err }},
		{"GetExchangeMap", "/v1/exchange/map", `{"data":[],"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeMap(1, 2); return err }},
		{"GetExchangeLatestMarketPairs", "/v1/exchange/market-pairs/latest", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeLatestMarketPairs(1, 1, 2); return err }},
		{"GetExchangeLatestQuotes", "/v1/exchange/quotes/latest", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeLatestQuotes(1); return err }},
		{"GetExchangeHistoricalQuotes", "/v1/exchange/quotes/historical", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error {
			_, err := c.GetExchangeHistoricalQuotes(1, time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetGlobalMeticLatestQuotes", "/v1/global-metrics/quotes/latest", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetGlobalMeticLatestQuotes(); return err }},
		{"GetGlobalMeticHistoricalQuotes", "/v1/global-metrics/quotes/historical", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error {
			_, err := c.GetGlobalMeticHistoricalQuotes(time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetPriceConversion", "/v2/tools/price-conversion", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetPriceConversion(1, 1, time.Now()); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, done := newSyntheticClient(t, map[string]string{tc.path: tc.payload})
			defer done()
			err := tc.invoke(client)
			assert.ErrorIs(t, err, errAPIResponse, "endpoint should return expected error")
			assert.ErrorContains(t, err, "boom", "endpoint should include API error message")
		})
	}
}

func TestCoinmarketcapEndpointRequestFailureSynthetic(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		invoke func(*Coinmarketcap) error
	}{
		{"GetCryptocurrencyInfo", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyInfo(1); return err }},
		{"GetCryptocurrencyIDMap", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyIDMap(); return err }},
		{"GetCryptocurrencyLatestListing", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestListing(1, 2); return err }},
		{"GetCryptocurrencyLatestMarketPairs", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestMarketPairs(1, 1, 2); return err }},
		{"GetCryptocurrencyOHLCHistorical", func(c *Coinmarketcap) error {
			_, err := c.GetCryptocurrencyOHLCHistorical(1, time.Now().Add(-time.Hour), time.Time{})
			return err
		}},
		{"GetCryptocurrencyOHLCLatest", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyOHLCLatest(1); return err }},
		{"GetCryptocurrencyLatestQuotes", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestQuotes(1); return err }},
		{"GetCryptocurrencyHistoricalQuotes", func(c *Coinmarketcap) error {
			_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Now().Add(-time.Hour), time.Time{})
			return err
		}},
		{"GetExchangeInfo", func(c *Coinmarketcap) error { _, err := c.GetExchangeInfo(1); return err }},
		{"GetExchangeMap", func(c *Coinmarketcap) error { _, err := c.GetExchangeMap(1, 2); return err }},
		{"GetExchangeLatestMarketPairs", func(c *Coinmarketcap) error { _, err := c.GetExchangeLatestMarketPairs(1, 1, 2); return err }},
		{"GetExchangeLatestQuotes", func(c *Coinmarketcap) error { _, err := c.GetExchangeLatestQuotes(1); return err }},
		{"GetExchangeHistoricalQuotes", func(c *Coinmarketcap) error {
			_, err := c.GetExchangeHistoricalQuotes(1, time.Now().Add(-time.Hour), time.Time{})
			return err
		}},
		{"GetGlobalMeticLatestQuotes", func(c *Coinmarketcap) error { _, err := c.GetGlobalMeticLatestQuotes(); return err }},
		{"GetGlobalMeticHistoricalQuotes", func(c *Coinmarketcap) error {
			_, err := c.GetGlobalMeticHistoricalQuotes(time.Now().Add(-time.Hour), time.Time{})
			return err
		}},
		{"GetPriceConversion", func(c *Coinmarketcap) error { _, err := c.GetPriceConversion(1, 1, time.Time{}); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, done := newSyntheticClient(t, map[string]string{})
			defer done()
			err := tc.invoke(client)
			assert.Error(t, err)
		})
	}
}

func TestCoinmarketcapAccountPlanGatesSynthetic(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		operation   string
		path        string
		payload     string
		minimumPlan uint8
		invoke      func(*Coinmarketcap) error
	}{
		{
			name:        "cryptocurrency market pairs require growth",
			operation:   "GetCryptocurrencyLatestMarketPairs",
			path:        "/v2/cryptocurrency/market-pairs/latest",
			payload:     `{"data":{},"status":{"error_code":0}}`,
			minimumPlan: Growth,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetCryptocurrencyLatestMarketPairs(1, 0, 0)
				return err
			},
		},
		{
			name:        "cryptocurrency historical OHLCV requires startup",
			operation:   "GetCryptocurrencyOHLCHistorical",
			path:        "/v2/cryptocurrency/ohlcv/historical",
			payload:     `{"data":{},"status":{"error_code":0}}`,
			minimumPlan: Startup,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetCryptocurrencyOHLCHistorical(1, time.Now(), time.Now())
				return err
			},
		},
		{
			name:        "cryptocurrency latest OHLCV requires startup",
			operation:   "GetCryptocurrencyOHLCLatest",
			path:        "/v2/cryptocurrency/ohlcv/latest",
			payload:     `{"data":{},"status":{"error_code":0}}`,
			minimumPlan: Startup,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetCryptocurrencyOHLCLatest(1)
				return err
			},
		},
		{
			name:        "cryptocurrency historical quotes allow basic",
			operation:   "GetCryptocurrencyHistoricalQuotes",
			path:        "/v3/cryptocurrency/quotes/historical",
			payload:     `{"data":{"1":{}},"status":{"error_code":0}}`,
			minimumPlan: Basic,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Now(), time.Now())
				return err
			},
		},
		{
			name:        "exchange info allows basic",
			operation:   "GetExchangeInfo",
			path:        "/v1/exchange/info",
			payload:     `{"data":{},"status":{"error_code":0}}`,
			minimumPlan: Basic,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetExchangeInfo(1)
				return err
			},
		},
		{
			name:        "exchange map allows basic",
			operation:   "GetExchangeMap",
			path:        "/v1/exchange/map",
			payload:     `{"data":[],"status":{"error_code":0}}`,
			minimumPlan: Basic,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetExchangeMap(1, 1)
				return err
			},
		},
		{
			name:        "exchange market pairs require growth",
			operation:   "GetExchangeLatestMarketPairs",
			path:        "/v1/exchange/market-pairs/latest",
			payload:     `{"data":{},"status":{"error_code":0}}`,
			minimumPlan: Growth,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetExchangeLatestMarketPairs(1, 1, 1)
				return err
			},
		},
		{
			name:        "exchange latest quotes require growth",
			operation:   "GetExchangeLatestQuotes",
			path:        "/v1/exchange/quotes/latest",
			payload:     `{"data":{},"status":{"error_code":0}}`,
			minimumPlan: Growth,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetExchangeLatestQuotes(1)
				return err
			},
		},
		{
			name:        "exchange historical quotes allow basic",
			operation:   "GetExchangeHistoricalQuotes",
			path:        "/v1/exchange/quotes/historical",
			payload:     `{"data":{"1":{}},"status":{"error_code":0}}`,
			minimumPlan: Basic,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetExchangeHistoricalQuotes(1, time.Now(), time.Now())
				return err
			},
		},
		{
			name:        "global historical quotes allow basic",
			operation:   "GetGlobalMeticHistoricalQuotes",
			path:        "/v1/global-metrics/quotes/historical",
			payload:     `{"data":{},"status":{"error_code":0}}`,
			minimumPlan: Basic,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetGlobalMeticHistoricalQuotes(time.Now(), time.Now())
				return err
			},
		},
		{
			name:        "historical price conversion requires builder",
			operation:   "GetPriceConversion",
			path:        "/v2/tools/price-conversion",
			payload:     `{"data":{},"status":{"error_code":0}}`,
			minimumPlan: Builder,
			invoke: func(c *Coinmarketcap) error {
				_, err := c.GetPriceConversion(1, 1, time.Now())
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, done := newSyntheticClient(t, map[string]string{tc.path: tc.payload})
			defer done()

			c.Plan = tc.minimumPlan
			err := tc.invoke(c)
			require.NoError(t, err, tc.operation+" must not error")

			c.Plan = tc.minimumPlan >> 1
			err = tc.invoke(c)
			assert.ErrorIs(t, err, errFunctionUseNotAllowed, tc.operation+" should error correctly for a lower plan")
		})
	}
}
