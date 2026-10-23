package coinbase

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// testAppSpotPriceDate is the day the historic spot price fixture was recorded for
var testAppSpotPriceDate = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

func TestAppGetFiatCurrencies(t *testing.T) {
	t.Parallel()
	resp, err := e.App().GetFiatCurrencies(t.Context())
	require.NoError(t, err, "GetFiatCurrencies must not error")
	if mockTests {
		exp := []AppFiatCurrencyResponse{
			{ID: currency.NewCode("AED"), Name: "United Arab Emirates Dirham", MinimumSize: 0.01},
			{ID: currency.NewCode("AFN"), Name: "Afghan Afghani", MinimumSize: 0.01},
			{ID: currency.NewCode("ALL"), Name: "Albanian Lek", MinimumSize: 0.01},
			{ID: currency.NewCode("AMD"), Name: "Armenian Dram", MinimumSize: 0.01},
			{ID: currency.NewCode("ANG"), Name: "Netherlands Antillean Gulden", MinimumSize: 0.01},
		}
		assert.Equal(t, exp, resp, "GetFiatCurrencies should decode every field")
	} else {
		require.NotEmpty(t, resp, "GetFiatCurrencies must return currencies")
		assert.Positive(t, resp[0].MinimumSize.Float64(), "GetFiatCurrencies should return each currency's minimum size")
	}

	_, err = newUnavailableServerExchange(t).App().GetFiatCurrencies(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFiatCurrencies should return the request failure")
}

func TestAppGetCryptocurrencies(t *testing.T) {
	t.Parallel()
	resp, err := e.App().GetCryptocurrencies(t.Context())
	require.NoError(t, err, "GetCryptocurrencies must not error")
	if mockTests {
		exp := []AppCurrencyResponse{
			{
				AssetID:      "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
				Code:         currency.BTC,
				Name:         "Bitcoin",
				Colour:       "#F7931A",
				SortIndex:    100,
				Exponent:     8,
				Type:         "crypto",
				AddressRegex: "^([13][a-km-zA-HJ-NP-Z1-9]{25,34})|^(bc1[pqzry9x8gf2tvdw0s3jn54khce6mua7l]([qpzry9x8gf2tvdw0s3jn54khce6mua7l]{38}|[qpzry9x8gf2tvdw0s3jn54khce6mua7l]{58}))$",
			},
			{
				AssetID:      "d85dce9b-5b73-5c3c-8978-522ce1d1c1b4",
				Code:         currency.ETH,
				Name:         "Ethereum",
				Colour:       "#627EEA",
				SortIndex:    101,
				Exponent:     8,
				Type:         "crypto",
				AddressRegex: "^(?:0x)?[0-9a-fA-F]{40}$",
			},
			{
				AssetID:      "c9c24c6e-c045-5fde-98a2-00ea7f520437",
				Code:         currency.LTC,
				Name:         "Litecoin",
				Colour:       "#A6A9AA",
				SortIndex:    102,
				Exponent:     8,
				Type:         "crypto",
				AddressRegex: "^((L|M)[a-km-zA-HJ-NP-Z1-9]{25,34})|^(ltc1([qpzry9x8gf2tvdw0s3jn54khce6mua7l]{39}|[qpzry9x8gf2tvdw0s3jn54khce6mua7l]{59}))$",
			},
			{
				AssetID:      "45f99e13-b522-57d7-8058-c57bf92fe7a3",
				Code:         currency.BCH,
				Name:         "Bitcoin Cash",
				Colour:       "#8DC351",
				SortIndex:    103,
				Exponent:     8,
				Type:         "crypto",
				AddressRegex: "^([13][a-km-zA-HJ-NP-Z1-9]{25,34})|^((bitcoincash:)?(q|p)[a-z0-9]{41})|^((BITCOINCASH:)?(Q|P)[A-Z0-9]{41})$",
			},
			{
				AssetID:      "c16df856-0345-5358-8a70-2a78c804e61f",
				Code:         currency.ETC,
				Name:         "Ethereum Classic",
				Colour:       "#59D4AF",
				SortIndex:    104,
				Exponent:     8,
				Type:         "crypto",
				AddressRegex: "^(?:0x)?[0-9a-fA-F]{40}$",
			},
			{
				AssetID:             "e17a44c8-6ea1-564f-a02c-2a9ca1d8eec4",
				Code:                currency.XRP,
				Name:                "XRP",
				Colour:              "#222222",
				SortIndex:           109,
				Exponent:            6,
				Type:                "crypto",
				AddressRegex:        "^r[1-9a-km-zA-HJ-NP-Z]{25,35}$",
				DestinationTagName:  "XRP Tag",
				DestinationTagRegex: `^\d{1,10}$`,
			},
		}
		assert.Equal(t, exp, resp, "GetCryptocurrencies should decode every field")
	} else {
		require.NotEmpty(t, resp, "GetCryptocurrencies must return currencies")
		assert.NotEmpty(t, resp[0].AssetID, "GetCryptocurrencies should return each currency's asset ID")
	}

	_, err = newUnavailableServerExchange(t).App().GetCryptocurrencies(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCryptocurrencies should return the request failure")
}

func TestAppGetExchangeRates(t *testing.T) {
	t.Parallel()
	// The currency is sent upper case, as Coinbase names currencies
	resp, err := e.App().GetExchangeRates(t.Context(), currency.BTC.Lower())
	require.NoError(t, err, "GetExchangeRates must not error")
	if mockTests {
		exp := &AppExchangeRatesResponse{
			Currency: currency.BTC,
			Rates: map[currency.Code]types.Number{
				currency.NewCode("00"):    8743236.443365486,
				currency.NewCode("1INCH"): 886394.6289905091,
				currency.NewCode("2Z"):    2743207.9439252336,
				currency.NewCode("A8"):    17747033.038220685,
				currency.NewCode("AAVE"):  541.4771496642842,
			},
		}
		assert.Equal(t, exp, resp, "GetExchangeRates should decode every field")
	} else {
		assert.True(t, resp.Currency.Equal(currency.BTC), "GetExchangeRates should return the base currency's rates")
		assert.Positive(t, resp.Rates[currency.USD].Float64(), "GetExchangeRates should return the base currency's USD rate")
	}

	resp, err = e.App().GetExchangeRates(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "GetExchangeRates must not error without a base currency")
	if mockTests {
		exp := &AppExchangeRatesResponse{
			Currency: currency.USD,
			Rates: map[currency.Code]types.Number{
				currency.NewCode("00"):    106.38286555014304,
				currency.NewCode("1INCH"): 10.785159620362382,
				currency.NewCode("2Z"):    33.37783711615487,
				currency.NewCode("A8"):    215.93608291945583,
				currency.NewCode("AAVE"):  0.0065776491481944,
			},
		}
		assert.Equal(t, exp, resp, "GetExchangeRates should decode every field of the default rates")
	} else {
		assert.True(t, resp.Currency.Equal(currency.USD), "GetExchangeRates should default to USD's rates")
	}

	_, err = newResponseServerExchange(t, `null`).App().GetExchangeRates(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetExchangeRates should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetExchangeRates(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetExchangeRates should return the request failure")
}

// testAppPrice checks a price a test received, against the fixture's amount in mock mode
func testAppPrice(t *testing.T, name string, amount float64, resp *AppPriceResponse) {
	t.Helper()
	if mockTests {
		assert.Equalf(t, &AppPriceResponse{Amount: types.Number(amount), Base: currency.BTC, Currency: currency.USD}, resp, "%s should decode every field", name)
		return
	}
	assert.Positivef(t, resp.Amount.Float64(), "%s should return a price", name)
	assert.Truef(t, resp.Base.Equal(currency.BTC), "%s should return the pair's base currency", name)
	assert.Truef(t, resp.Currency.Equal(currency.USD), "%s should return the price in the pair's quote currency", name)
}

func TestAppGetBuyPrice(t *testing.T) {
	t.Parallel()
	_, err := e.App().GetBuyPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetBuyPrice must require a pair")

	resp, err := e.App().GetBuyPrice(t.Context(), currency.NewBTCUSD())
	require.NoError(t, err, "GetBuyPrice must not error")
	testAppPrice(t, "GetBuyPrice", 82190.155, resp)

	_, err = newResponseServerExchange(t, `null`).App().GetBuyPrice(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetBuyPrice should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetBuyPrice(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBuyPrice should return the request failure")
}

func TestAppGetSellPrice(t *testing.T) {
	t.Parallel()
	_, err := e.App().GetSellPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSellPrice must require a pair")

	resp, err := e.App().GetSellPrice(t.Context(), currency.NewBTCUSD())
	require.NoError(t, err, "GetSellPrice must not error")
	testAppPrice(t, "GetSellPrice", 82174.295, resp)

	_, err = newResponseServerExchange(t, `null`).App().GetSellPrice(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSellPrice should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetSellPrice(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSellPrice should return the request failure")
}

func TestAppGetSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := e.App().GetSpotPrice(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotPrice must reject a nil request")
	_, err = e.App().GetSpotPrice(t.Context(), &AppSpotPriceRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotPrice must require a pair")

	resp, err := e.App().GetSpotPrice(t.Context(), &AppSpotPriceRequest{Pair: currency.NewBTCUSD()})
	require.NoError(t, err, "GetSpotPrice must not error")
	testAppPrice(t, "GetSpotPrice", 82174.295, resp)
	// The date is sent as the UTC day it falls on, here the day before its local one
	resp, err = e.App().GetSpotPrice(t.Context(), &AppSpotPriceRequest{Pair: currency.NewBTCUSD(), Date: testAppSpotPriceDate.Add(19 * time.Hour).In(time.FixedZone("AEST", 10*60*60))})
	require.NoError(t, err, "GetSpotPrice must not error for a past day")
	testAppPrice(t, "GetSpotPrice", 83556.145, resp)

	_, err = newResponseServerExchange(t, `null`).App().GetSpotPrice(t.Context(), &AppSpotPriceRequest{Pair: currency.NewBTCUSD()})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSpotPrice should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetSpotPrice(t.Context(), &AppSpotPriceRequest{Pair: currency.NewBTCUSD()})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSpotPrice should return the request failure")
}

func TestAppGetCurrentTime(t *testing.T) {
	t.Parallel()
	resp, err := e.App().GetCurrentTime(t.Context())
	require.NoError(t, err, "GetCurrentTime must not error")
	if mockTests {
		exp := &AppServerTimeResponse{
			ISO:   time.Date(2026, 10, 19, 1, 2, 5, 0, time.UTC),
			Epoch: types.Time(time.Unix(1792371725, 0)),
		}
		assert.Equal(t, exp, resp, "GetCurrentTime should decode every field")
	} else {
		assert.WithinDuration(t, time.Now(), resp.Epoch.Time(), time.Minute, "GetCurrentTime should return the current time")
	}

	_, err = newResponseServerExchange(t, `null`).App().GetCurrentTime(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetCurrentTime should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetCurrentTime(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCurrentTime should return the request failure")
}

func TestAppGetPrice(t *testing.T) {
	t.Parallel()
	for _, pair := range []currency.Pair{currency.EMPTYPAIR, {Base: currency.BTC}, {Quote: currency.USD}} {
		_, err := e.App().getPrice(t.Context(), pair, "spot", nil)
		assert.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "getPrice should refuse %q", pair)
	}
	for _, kind := range []string{"buy", "sell", "spot"} {
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equalf(t, http.MethodGet, r.Method, "getPrice should get the %s price", kind)
			assert.Empty(t, r.Header.Get("Authorization"), "getPrice should not authenticate")
			// The pair is sent upper case and dash delimited, whatever its format
			assert.Equalf(t, "/v2/prices/ETH-USDC/"+kind, r.URL.Path, "getPrice should get the pair's %s price", kind)
			assert.Equal(t, "date=2026-10-01", r.URL.RawQuery, "getPrice should send the params")
			_, err := w.Write([]byte(`{"data":{"amount":"3000.5","base":"ETH","currency":"USDC"}}`))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		resp, err := ex.App().getPrice(t.Context(), currency.NewPairWithDelimiter("eth", "usdc", "_"), kind, map[string][]string{"date": {"2026-10-01"}})
		require.NoErrorf(t, err, "getPrice must not error for the %s price", kind)
		assert.Equalf(t, &AppPriceResponse{Amount: 3000.5, Base: currency.ETH, Currency: currency.USDC}, resp, "getPrice should return the %s price", kind)
	}
}
