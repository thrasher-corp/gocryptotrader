package coinbase

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// testMarketDataPerpetual is the USDC perpetual and testMarketDataPerpetualID its instrument ID, and
	// testMarketDataInversePerpetual the inverse BTC perpetual, which mock and live tests share
	testMarketDataPerpetual        = "BTC_USDC-PERPETUAL"
	testMarketDataPerpetualID      = 211704
	testMarketDataInversePerpetual = "BTC-PERPETUAL"
	// testMarketDataSpot is a spot pair whose orders Coinbase Exchange matches
	testMarketDataSpot      = "BTC_USDC"
	testMarketDataIndexName = "btc_usdc"
	// testMarketDataUnknownInstrument is an instrument the gateway has never listed
	testMarketDataUnknownInstrument = "BTC-01JAN20"
	// mockMarketDataOption, mockMarketDataDatedFuture and mockMarketDataFutureCombo are the option, dated future and
	// future combo the mock responses were recorded for; live tests look up listed ones instead
	mockMarketDataOption      = "BTC-6NOV26-82000-C"
	mockMarketDataDatedFuture = "BTC_USDC-27NOV26"
	mockMarketDataFutureCombo = "BTC-FS-27NOV26_PERP"
	// mockMarketDataTradesStartID and mockMarketDataTradesEndID bound the first USDC futures trades of 2026-10-19, and
	// mockMarketDataTradesStartSequence and mockMarketDataTradesEndSequence the first testMarketDataPerpetual trades,
	// which the mock trades were recorded for; the gateway only serves the last 24 hours of trades, so live tests ask
	// for recent ones
	mockMarketDataTradesStartID       = "USDC-80578495"
	mockMarketDataTradesEndID         = "USDC-80578499"
	mockMarketDataTradesStartSequence = 19540034
	mockMarketDataTradesEndSequence   = 19540036
	// mockMarketDataMarkPriceStart and mockMarketDataMarkPriceEnd bound the mock mark price history, in Unix
	// milliseconds; the gateway keeps about 5 minutes of it, so live tests ask for the last minute instead
	mockMarketDataMarkPriceStart = 1792372167000
	mockMarketDataMarkPriceEnd   = 1792372187000
	// testMarketDataYear3000 is the expiration, in Unix milliseconds, of perpetuals and spot
	testMarketDataYear3000 = 32503708800000
)

var (
	// testMarketDataRangeStart and testMarketDataRangeEnd bound the 4 hours from 2026-10-18T00:00:00Z
	testMarketDataRangeStart = time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	testMarketDataRangeEnd   = testMarketDataRangeStart.Add(4 * time.Hour)
	// mockMarketDataTradesWindowStart and mockMarketDataTradesWindowEnd bound the hour from 2026-10-19T00:00:00Z, which
	// the mock trades were recorded for
	mockMarketDataTradesWindowStart = time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	mockMarketDataTradesWindowEnd   = mockMarketDataTradesWindowStart.Add(time.Hour)
)

var errNoListedMarketDataInstrument = errors.New("no listed instrument found for the live tests")

// testMarketDataMilli converts a Unix millisecond timestamp, as the gateway sends most times, into a types.Time
func testMarketDataMilli(timestamp int64) types.Time {
	return types.Time(time.UnixMilli(timestamp))
}

// testMarketDataNano converts a Unix nanosecond timestamp, as the gateway sends Starbase times, into a types.Time
func testMarketDataNano(timestamp int64) types.Time {
	return types.Time(time.Unix(0, timestamp))
}

// listedMarketDataInstrument returns the first open instrument of a kind settled in BTC that expires at least a day
// from now, which live tests use in place of the mock instruments
func listedMarketDataInstrument(kind string) (string, error) {
	instruments, err := e.GlobalDerivatives().GetInstruments(context.Background(), &DerivativesInstrumentsRequest{Currency: currency.BTC, Kind: kind})
	if err != nil {
		return "", err
	}
	for i := range instruments {
		if instruments[i].State == "open" && time.Until(instruments[i].ExpirationTimestamp.Time()) > 24*time.Hour {
			return instruments[i].InstrumentName, nil
		}
	}
	return "", errNoListedMarketDataInstrument
}

var (
	listedMarketDataOption      = sync.OnceValues(func() (string, error) { return listedMarketDataInstrument(DerivativesKindOption) })
	listedMarketDataFutureCombo = sync.OnceValues(func() (string, error) { return listedMarketDataInstrument(DerivativesKindFutureCombo) })
)

// marketDataOption returns mockMarketDataOption, or a listed option when testing live
func marketDataOption(t *testing.T) string {
	t.Helper()
	if mockTests {
		return mockMarketDataOption
	}
	name, err := listedMarketDataOption()
	require.NoError(t, err, "listedMarketDataOption must not error")
	return name
}

// marketDataFutureCombo returns mockMarketDataFutureCombo, or a listed future combo when testing live
func marketDataFutureCombo(t *testing.T) string {
	t.Helper()
	if mockTests {
		return mockMarketDataFutureCombo
	}
	name, err := listedMarketDataFutureCombo()
	require.NoError(t, err, "listedMarketDataFutureCombo must not error")
	return name
}

// mockMarketDataFutureSummaries are the mock book summaries of USDC futures
func mockMarketDataFutureSummaries() []DerivativesBookSummaryResponse {
	return []DerivativesBookSummaryResponse{
		{
			InstrumentName:         "BTC_USDC-27NOV26",
			BaseCurrency:           currency.BTC,
			QuoteCurrency:          currency.USDC,
			CreationTimestamp:      testMarketDataMilli(1792372154018),
			BidPrice:               82900,
			AskPrice:               82902.5,
			MidPrice:               82901.25,
			MarkPrice:              82917.97,
			LastPrice:              82827.5,
			High:                   82895,
			Low:                    81207.5,
			PriceChange:            1.0769,
			Volume:                 5.5899,
			VolumeUSD:              458727.17,
			VolumeNotional:         458818.93875,
			OpenInterest:           19.9515,
			EstimatedDeliveryPrice: 82394.38,
		},
		{
			InstrumentName:         "BTC_USDC-PERPETUAL",
			BaseCurrency:           currency.BTC,
			QuoteCurrency:          currency.USDC,
			CreationTimestamp:      testMarketDataMilli(1792372154018),
			BidPrice:               82415.8,
			AskPrice:               82415.9,
			MidPrice:               82415.85,
			MarkPrice:              82420.01,
			LastPrice:              82415.9,
			High:                   82481.3,
			Low:                    80583.1,
			PriceChange:            1.1176,
			Volume:                 35718.947,
			VolumeUSD:              2909527627.77,
			VolumeNotional:         2910109815.0091496,
			OpenInterest:           1777.3109,
			EstimatedDeliveryPrice: 82394.38,
			CurrentFunding:         0.00006106,
			Funding8Hour:           0.00006347,
		},
		{
			InstrumentName:         "ETH_USDC-PERPETUAL",
			BaseCurrency:           currency.ETH,
			QuoteCurrency:          currency.USDC,
			CreationTimestamp:      testMarketDataMilli(1792372154019),
			BidPrice:               2420.17,
			AskPrice:               2420.18,
			MidPrice:               2420.175,
			MarkPrice:              2420.31,
			LastPrice:              2420.17,
			High:                   2428.09,
			Low:                    2326.03,
			PriceChange:            0.6316,
			Volume:                 812637.2373,
			VolumeUSD:              1941471055.75,
			VolumeNotional:         1941859504.179324,
			OpenInterest:           26025.3773,
			EstimatedDeliveryPrice: 2419.47,
			CurrentFunding:         0.000097,
			Funding8Hour:           0.000024,
		},
	}
}

// mockMarketDataOptionSummaries are the mock book summaries of BTC options, filtered to mockMarketDataOption's
func mockMarketDataOptionSummaries() []DerivativesBookSummaryResponse {
	return []DerivativesBookSummaryResponse{
		{
			InstrumentName:         "BTC-6NOV26-82000-C",
			BaseCurrency:           currency.BTC,
			QuoteCurrency:          currency.BTC,
			CreationTimestamp:      testMarketDataMilli(1792372155682),
			BidPrice:               0.0365,
			AskPrice:               0.0375,
			MidPrice:               0.037,
			MarkPrice:              0.03702931,
			LastPrice:              0.0345,
			High:                   0.0345,
			Low:                    0.0295,
			PriceChange:            9.5238,
			Volume:                 19.4,
			VolumeUSD:              49513.82,
			OpenInterest:           379.4,
			EstimatedDeliveryPrice: 82394.32,
			MarkImpliedVolatility:  37.01,
			UnderlyingIndex:        "BTC-6NOV26",
			UnderlyingPrice:        82655.44,
		},
	}
}

func TestGlobalDerivativesGetBookSummaryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetBookSummaryByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBookSummaryByCurrency must reject a nil request")
	_, err = e.GlobalDerivatives().GetBookSummaryByCurrency(t.Context(), &DerivativesBookSummaryByCurrencyRequest{Kind: DerivativesKindFuture})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBookSummaryByCurrency must require a currency")
	_, err = e.GlobalDerivatives().GetBookSummaryByCurrency(t.Context(), &DerivativesBookSummaryByCurrencyRequest{Currency: currency.USDC, Kind: DerivativesKindAny})
	require.ErrorIs(t, err, errDerivativesInvalidMarketDataKind, "GetBookSummaryByCurrency must refuse a kind the gateway does not summarise")

	resp, err := e.GlobalDerivatives().GetBookSummaryByCurrency(t.Context(), &DerivativesBookSummaryByCurrencyRequest{Currency: currency.USDC, Kind: DerivativesKindFuture})
	require.NoError(t, err, "GetBookSummaryByCurrency must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFutureSummaries(), resp, "GetBookSummaryByCurrency should decode every field of a future")
	} else {
		assert.NotEmpty(t, resp, "GetBookSummaryByCurrency should return USDC's futures")
	}

	resp, err = e.GlobalDerivatives().GetBookSummaryByCurrency(t.Context(), &DerivativesBookSummaryByCurrencyRequest{Currency: currency.BTC, Kind: DerivativesKindOption})
	require.NoError(t, err, "GetBookSummaryByCurrency must not error for options")
	if mockTests {
		assert.Equal(t, mockMarketDataOptionSummaries(), resp, "GetBookSummaryByCurrency should decode every field of an option")
	} else {
		assert.NotEmpty(t, resp, "GetBookSummaryByCurrency should return BTC's options")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetBookSummaryByCurrency(t.Context(), &DerivativesBookSummaryByCurrencyRequest{Currency: currency.USDC})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBookSummaryByCurrency should return the request failure")
}

// mockMarketDataPerpetualSummary is the mock book summary of testMarketDataPerpetual
func mockMarketDataPerpetualSummary() []DerivativesBookSummaryResponse {
	return []DerivativesBookSummaryResponse{
		{
			InstrumentName:         "BTC_USDC-PERPETUAL",
			BaseCurrency:           currency.BTC,
			QuoteCurrency:          currency.USDC,
			CreationTimestamp:      testMarketDataMilli(1792372156831),
			BidPrice:               82415.8,
			AskPrice:               82415.9,
			MidPrice:               82415.85,
			MarkPrice:              82419.23,
			LastPrice:              82415.9,
			High:                   82481.3,
			Low:                    80583.1,
			PriceChange:            1.1176,
			Volume:                 35719.147,
			VolumeUSD:              2909544107.65,
			VolumeNotional:         2910126298.18915,
			OpenInterest:           1777.389,
			EstimatedDeliveryPrice: 82393.97,
			CurrentFunding:         0.00005658,
			Funding8Hour:           0.00006348,
		},
	}
}

func TestGlobalDerivativesGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetBookSummaryByInstrument(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetBookSummaryByInstrument must require an instrument name")

	resp, err := e.GlobalDerivatives().GetBookSummaryByInstrument(t.Context(), testMarketDataPerpetual)
	require.NoError(t, err, "GetBookSummaryByInstrument must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataPerpetualSummary(), resp, "GetBookSummaryByInstrument should decode every field")
	} else {
		require.Len(t, resp, 1, "GetBookSummaryByInstrument must return one summary")
		assert.Equal(t, testMarketDataPerpetual, resp[0].InstrumentName, "GetBookSummaryByInstrument should return the instrument requested")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetBookSummaryByInstrument(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBookSummaryByInstrument should return the request failure")
}

func TestGlobalDerivativesGetContractSize(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetContractSize(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetContractSize must require an instrument name")

	resp, err := e.GlobalDerivatives().GetContractSize(t.Context(), testMarketDataPerpetual)
	require.NoError(t, err, "GetContractSize must not error")
	assert.Equal(t, &DerivativesContractSizeResponse{ContractSize: 0.0001}, resp, "GetContractSize should decode BTC_USDC-PERPETUAL's contract size of 0.0001 BTC")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetContractSize(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetContractSize should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetContractSize(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetContractSize should return the request failure")
}

// mockMarketDataCurrencies are the mock currencies, filtered to USDC and BTC
func mockMarketDataCurrencies() []DerivativesCurrencyResponse {
	return []DerivativesCurrencyResponse{
		{
			Currency:                   currency.USDC,
			CurrencyLongName:           "USD Coin",
			CurrencyUUID:               "2b92315d-eab7-5bef-84fa-089a131333f5",
			CoinType:                   "USDC",
			Decimals:                   8,
			OnchainOperationsPrecision: 6,
			MinimumConfirmations:       50,
			WithdrawalFee:              5,
			MinimumWithdrawalFee:       5,
			WithdrawalPriorities:       []DerivativesWithdrawalPriority{},
			NetworkFee:                 0.0000019562366000000002,
			NetworkCurrency:            currency.ETH,
			InCrossCollateralPool:      true,
			APR:                        3.3999999999999995,
			CoinbaseNetworks: []DerivativesCoinbaseNetwork{
				{ResourceName: "networks/arbitrum-mainnet", DisplayName: "Arbitrum"},
				{ResourceName: "networks/avacchain-mainnet", DisplayName: "Avalanche C-Chain"},
				{ResourceName: "networks/base-mainnet", DisplayName: "Base"},
				{ResourceName: "networks/ethereum-mainnet", DisplayName: "Ethereum"},
				{ResourceName: "networks/optimism-mainnet", DisplayName: "Optimism"},
			},
		},
		{
			Currency:                   currency.BTC,
			CurrencyLongName:           "Bitcoin",
			CurrencyUUID:               "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			CoinType:                   "BTC",
			Decimals:                   8,
			OnchainOperationsPrecision: 8,
			MinimumConfirmations:       1,
			WithdrawalFee:              0.00001,
			MinimumWithdrawalFee:       0.00001,
			WithdrawalPriorities:       []DerivativesWithdrawalPriority{},
			NetworkFee:                 0.000003072,
			NetworkCurrency:            currency.BTC,
			InCrossCollateralPool:      true,
			CoinbaseNetworks: []DerivativesCoinbaseNetwork{
				{ResourceName: "networks/bitcoin-mainnet", DisplayName: "Bitcoin"},
			},
		},
	}
}

func TestGlobalDerivativesGetCurrencies(t *testing.T) {
	t.Parallel()
	resp, err := e.GlobalDerivatives().GetCurrencies(t.Context())
	require.NoError(t, err, "GetCurrencies must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataCurrencies(), resp, "GetCurrencies should decode every field")
	} else {
		assert.NotEmpty(t, resp, "GetCurrencies should return the gateway's currencies")
	}

	// The gateway sends no withdrawal priorities today, so the documented example's are served
	resp, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":[{"currency":"BTC","withdrawal_priorities":[{"name":"very_low","value":0.15},{"name":"very_high","value":1.5}]}]}`).GlobalDerivatives().GetCurrencies(t.Context())
	require.NoError(t, err, "GetCurrencies must not error for withdrawal priorities")
	exp := []DerivativesCurrencyResponse{{Currency: currency.BTC, WithdrawalPriorities: []DerivativesWithdrawalPriority{{Name: "very_low", Value: 0.15}, {Name: "very_high", Value: 1.5}}}}
	assert.Equal(t, exp, resp, "GetCurrencies should decode the documented withdrawal priorities")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetCurrencies(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCurrencies should return the request failure")
}

// testMarketDataDeliveryPricesRequest skips the latest delivery price of testMarketDataIndexName
func testMarketDataDeliveryPricesRequest() *DerivativesDeliveryPricesRequest {
	return &DerivativesDeliveryPricesRequest{IndexName: testMarketDataIndexName, Offset: 1, Count: 5}
}

// mockMarketDataDeliveryPrices are the mock delivery prices of testMarketDataIndexName
func mockMarketDataDeliveryPrices() *DerivativesDeliveryPricesResponse {
	return &DerivativesDeliveryPricesResponse{
		Data: []DerivativesDeliveryPrice{
			{Date: "2026-10-17", DeliveryPrice: 81593},
			{Date: "2026-10-16", DeliveryPrice: 82111.39},
			{Date: "2026-10-15", DeliveryPrice: 81232.05},
			{Date: "2026-10-14", DeliveryPrice: 83396.72},
			{Date: "2026-10-13", DeliveryPrice: 83826.87},
		},
		RecordsTotal: 1684,
	}
}

func TestGlobalDerivativesGetDeliveryPrices(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetDeliveryPrices(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDeliveryPrices must reject a nil request")
	_, err = e.GlobalDerivatives().GetDeliveryPrices(t.Context(), &DerivativesDeliveryPricesRequest{Count: 5})
	require.ErrorIs(t, err, errDerivativesIndexNameRequired, "GetDeliveryPrices must require an index name")

	resp, err := e.GlobalDerivatives().GetDeliveryPrices(t.Context(), testMarketDataDeliveryPricesRequest())
	require.NoError(t, err, "GetDeliveryPrices must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataDeliveryPrices(), resp, "GetDeliveryPrices should decode every field")
	} else {
		assert.Len(t, resp.Data, 5, "GetDeliveryPrices should return the prices requested")
		assert.NotZero(t, resp.RecordsTotal, "GetDeliveryPrices should return the number of prices available")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetDeliveryPrices(t.Context(), testMarketDataDeliveryPricesRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetDeliveryPrices should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetDeliveryPrices(t.Context(), testMarketDataDeliveryPricesRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDeliveryPrices should return the request failure")
}

// mockMarketDataUSDCExpirations are the mock labels of USDC futures, keyed by currency
func mockMarketDataUSDCExpirations() *DerivativesExpirationsResponse {
	return &DerivativesExpirationsResponse{ByCurrency: map[string]DerivativesExpirations{"usdc": {Future: []string{"19OCT26", "20OCT26", "21OCT26", "22OCT26", "23OCT26"}}}}
}

func TestGlobalDerivativesGetExpirations(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetExpirations(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExpirations must reject a nil request")
	_, err = e.GlobalDerivatives().GetExpirations(t.Context(), &DerivativesExpirationsRequest{Kind: DerivativesKindSpot})
	require.ErrorIs(t, err, errDerivativesInvalidMarketDataKind, "GetExpirations must refuse a kind without expirations")

	resp, err := e.GlobalDerivatives().GetExpirations(t.Context(), &DerivativesExpirationsRequest{Currency: currency.ANY, Kind: DerivativesKindFuture})
	require.NoError(t, err, "GetExpirations must not error for every currency")
	if mockTests {
		exp := &DerivativesExpirationsResponse{Future: []string{"19OCT26", "20OCT26", "21OCT26", "22OCT26", "23OCT26"}}
		assert.Equal(t, exp, resp, "GetExpirations should decode the labels merged across currencies")
	} else {
		assert.Contains(t, resp.Future, "PERPETUAL", "GetExpirations should return the perpetuals' label")
	}

	resp, err = e.GlobalDerivatives().GetExpirations(t.Context(), &DerivativesExpirationsRequest{Currency: currency.USDC, Kind: DerivativesKindFuture})
	require.NoError(t, err, "GetExpirations must not error for a currency")
	if mockTests {
		assert.Equal(t, mockMarketDataUSDCExpirations(), resp, "GetExpirations should decode the labels keyed by currency")
	} else {
		assert.Contains(t, resp.ByCurrency["usdc"].Future, "PERPETUAL", "GetExpirations should return USDC's perpetuals' label")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, ok := readJSONRPC(t, r)
		if ok {
			assert.JSONEq(t, `{"currency":"grouped","kind":"any","currency_pair":"btc_usdc"}`, string(req.Params), "GetExpirations should send grouped in lower case")
		}
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"btc":{"option":[],"future":["PERPETUAL"]}}}`)
	}))
	resp, err = ex.GlobalDerivatives().GetExpirations(t.Context(), &DerivativesExpirationsRequest{Currency: currency.NewCode("GROUPED"), Kind: DerivativesKindAny, CurrencyPair: testMarketDataIndexName})
	require.NoError(t, err, "GetExpirations must not error for grouped labels")
	exp := &DerivativesExpirationsResponse{ByCurrency: map[string]DerivativesExpirations{"btc": {Option: []string{}, Future: []string{"PERPETUAL"}}}}
	assert.Equal(t, exp, resp, "GetExpirations should decode grouped labels")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetExpirations(t.Context(), &DerivativesExpirationsRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetExpirations should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetExpirations(t.Context(), &DerivativesExpirationsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetExpirations should return the request failure")
}

func TestDerivativesExpirationsResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input string
		exp   DerivativesExpirationsResponse
	}{
		{
			name:  "merged",
			input: `{"future":["21SEP24","PERPETUAL"],"option":["21SEP24"]}`,
			exp:   DerivativesExpirationsResponse{Future: []string{"21SEP24", "PERPETUAL"}, Option: []string{"21SEP24"}},
		},
		{
			name:  "keyed",
			input: `{"btc":{"future":["PERPETUAL"]},"eth_usdc":{"option":[]}}`,
			exp:   DerivativesExpirationsResponse{ByCurrency: map[string]DerivativesExpirations{"btc": {Future: []string{"PERPETUAL"}}, "eth_usdc": {Option: []string{}}}},
		},
		{name: "empty", input: `{}`, exp: DerivativesExpirationsResponse{ByCurrency: map[string]DerivativesExpirations{}}},
	} {
		var resp DerivativesExpirationsResponse
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &resp), "Unmarshal must not error for the %s labels", tc.name)
		assert.Equalf(t, tc.exp, resp, "UnmarshalJSON should decode the %s labels", tc.name)
	}
	var resp DerivativesExpirationsResponse
	assert.Error(t, json.Unmarshal([]byte(`["PERPETUAL"]`), &resp), "UnmarshalJSON should refuse labels that are not an object")
	assert.Error(t, json.Unmarshal([]byte(`{"btc":"PERPETUAL"}`), &resp), "UnmarshalJSON should refuse a currency's labels that are not an object")
}

// mockMarketDataFundingChart is the mock funding chart of testMarketDataPerpetual
func mockMarketDataFundingChart() *DerivativesFundingChartDataResponse {
	return &DerivativesFundingChartDataResponse{
		CurrentInterest: 0.00005147372590936692,
		Interest8Hour:   0.0000634859353147155,
		Data: []DerivativesFundingChartPoint{
			{Timestamp: testMarketDataMilli(1792343400000), IndexPrice: 81089.87, Interest8Hour: 0.00011539705884353567},
			{Timestamp: testMarketDataMilli(1792343460000), IndexPrice: 81088.46, Interest8Hour: 0.00012687236876859751},
			{Timestamp: testMarketDataMilli(1792343520000), IndexPrice: 81072.57, Interest8Hour: 0.00009635635702663913},
			{Timestamp: testMarketDataMilli(1792343580000), IndexPrice: 81062.13, Interest8Hour: 0.00008221431511845314},
			{Timestamp: testMarketDataMilli(1792343640000), IndexPrice: 81070.59, Interest8Hour: 0.00010376577375355929},
		},
	}
}

func TestGlobalDerivativesGetFundingChartData(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetFundingChartData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingChartData must reject a nil request")
	_, err = e.GlobalDerivatives().GetFundingChartData(t.Context(), &DerivativesFundingChartDataRequest{Length: DerivativesFundingChartLength8Hour})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetFundingChartData must require an instrument name")
	_, err = e.GlobalDerivatives().GetFundingChartData(t.Context(), &DerivativesFundingChartDataRequest{InstrumentName: testMarketDataPerpetual, Length: "2h"})
	require.ErrorIs(t, err, errDerivativesInvalidFundingChartLength, "GetFundingChartData must refuse a length the gateway does not chart")

	arg := &DerivativesFundingChartDataRequest{InstrumentName: testMarketDataPerpetual, Length: DerivativesFundingChartLength8Hour}
	resp, err := e.GlobalDerivatives().GetFundingChartData(t.Context(), arg)
	require.NoError(t, err, "GetFundingChartData must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFundingChart(), resp, "GetFundingChartData should decode every field")
	} else {
		assert.NotEmpty(t, resp.Data, "GetFundingChartData should return the funding chart")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetFundingChartData(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFundingChartData should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetFundingChartData(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingChartData should return the request failure")
}

// testMarketDataFundingRequest covers testMarketDataPerpetual's funding over the 4 hours from 2026-10-18T00:00:00Z
func testMarketDataFundingRequest() *DerivativesInstrumentTimeRangeRequest {
	return &DerivativesInstrumentTimeRangeRequest{InstrumentName: testMarketDataPerpetual, StartTime: testMarketDataRangeStart, EndTime: testMarketDataRangeEnd}
}

// mockMarketDataFundingRateHistory is the mock hourly funding of testMarketDataPerpetual, each hour ending after the
// range's start
func mockMarketDataFundingRateHistory() []DerivativesFundingRateHistoryResponse {
	return []DerivativesFundingRateHistoryResponse{
		{
			Timestamp:          testMarketDataMilli(1792285200000),
			IndexPrice:         81435.22,
			PreviousIndexPrice: 81387.79,
			Interest1Hour:      0.000010156278056231139,
			Interest8Hour:      0.00009515254416843552,
		},
		{
			Timestamp:          testMarketDataMilli(1792288800000),
			IndexPrice:         81552.51,
			PreviousIndexPrice: 81435.22,
			Interest1Hour:      0.000010994193662501815,
			Interest8Hour:      0.00009062602119074658,
		},
		{
			Timestamp:          testMarketDataMilli(1792292400000),
			IndexPrice:         81720.27,
			PreviousIndexPrice: 81552.51,
			Interest1Hour:      0.000008290183359086795,
			Interest8Hour:      0.00008405625894908967,
		},
		{
			Timestamp:          testMarketDataMilli(1792296000000),
			IndexPrice:         81845.54,
			PreviousIndexPrice: 81720.27,
			Interest1Hour:      0.000005315612301354282,
			Interest8Hour:      0.000076141566404786,
		},
	}
}

func TestGlobalDerivativesGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetFundingRateHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingRateHistory must reject a nil request")
	_, err = e.GlobalDerivatives().GetFundingRateHistory(t.Context(), &DerivativesInstrumentTimeRangeRequest{InstrumentName: testMarketDataPerpetual})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetFundingRateHistory must require a time range")

	resp, err := e.GlobalDerivatives().GetFundingRateHistory(t.Context(), testMarketDataFundingRequest())
	require.NoError(t, err, "GetFundingRateHistory must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFundingRateHistory(), resp, "GetFundingRateHistory should decode every field")
	} else {
		assert.Len(t, resp, 4, "GetFundingRateHistory should return an hour for each hour ending in the range")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetFundingRateHistory(t.Context(), testMarketDataFundingRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingRateHistory should return the request failure")
}

func TestGlobalDerivativesGetFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetFundingRateValue(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingRateValue must reject a nil request")
	_, err = e.GlobalDerivatives().GetFundingRateValue(t.Context(), &DerivativesInstrumentTimeRangeRequest{StartTime: testMarketDataRangeStart, EndTime: testMarketDataRangeEnd})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetFundingRateValue must require an instrument name")

	resp, err := e.GlobalDerivatives().GetFundingRateValue(t.Context(), testMarketDataFundingRequest())
	require.NoError(t, err, "GetFundingRateValue must not error")
	assert.Equal(t, 0.00003480116782906063, resp, "GetFundingRateValue should decode the funding over the range")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetFundingRateValue(t.Context(), testMarketDataFundingRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFundingRateValue should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetFundingRateValue(t.Context(), testMarketDataFundingRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingRateValue should return the request failure")
}

func TestDerivativesInstrumentTimeRangeRequestParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *DerivativesInstrumentTimeRangeRequest
		err  error
	}{
		{name: "a nil request", err: common.ErrNilPointer},
		{name: "an empty instrument name", req: &DerivativesInstrumentTimeRangeRequest{StartTime: testMarketDataRangeStart, EndTime: testMarketDataRangeEnd}, err: errInstrumentNameRequired},
		{name: "an unset start", req: &DerivativesInstrumentTimeRangeRequest{InstrumentName: testMarketDataPerpetual, EndTime: testMarketDataRangeEnd}, err: common.ErrDateUnset},
		{name: "an unset end", req: &DerivativesInstrumentTimeRangeRequest{InstrumentName: testMarketDataPerpetual, StartTime: testMarketDataRangeStart}, err: common.ErrDateUnset},
		{name: "a reversed range", req: &DerivativesInstrumentTimeRangeRequest{InstrumentName: testMarketDataPerpetual, StartTime: testMarketDataRangeEnd, EndTime: testMarketDataRangeStart}, err: common.ErrStartAfterEnd},
	} {
		_, err := tc.req.params()
		assert.ErrorIsf(t, err, tc.err, "params should refuse %s", tc.name)
	}

	params, err := testMarketDataFundingRequest().params()
	require.NoError(t, err, "params must not error")
	assert.Equal(t, &DerivativesInstrumentTimeRangeWireRequest{InstrumentName: testMarketDataPerpetual, StartTimestamp: 1792281600000, EndTimestamp: 1792296000000}, params, "params should send the time range in Unix milliseconds")
}

// mockMarketDataHistoricalVolatility is the mock realised volatility of BTC
func mockMarketDataHistoricalVolatility() []DerivativesHistoricalVolatilityResponse {
	return []DerivativesHistoricalVolatilityResponse{
		{Timestamp: testMarketDataMilli(1791000000000), Volatility: 37.10409279955044},
		{Timestamp: testMarketDataMilli(1791000000000), Volatility: 37.10409279955044},
		{Timestamp: testMarketDataMilli(1791003600000), Volatility: 37.023424633380706},
		{Timestamp: testMarketDataMilli(1791007200000), Volatility: 37.010590590864396},
		{Timestamp: testMarketDataMilli(1791010800000), Volatility: 37.01031876504902},
	}
}

func TestGlobalDerivativesGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetHistoricalVolatility(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetHistoricalVolatility must require a currency")

	resp, err := e.GlobalDerivatives().GetHistoricalVolatility(t.Context(), currency.BTC)
	require.NoError(t, err, "GetHistoricalVolatility must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataHistoricalVolatility(), resp, "GetHistoricalVolatility should decode every point")
	} else {
		assert.NotEmpty(t, resp, "GetHistoricalVolatility should return BTC's realised volatility")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetHistoricalVolatility(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetHistoricalVolatility should return the request failure")
}

func TestDerivativesHistoricalVolatilityUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var resp DerivativesHistoricalVolatilityResponse
	require.NoError(t, json.Unmarshal([]byte(`[1549720800000,14.747743607344217]`), &resp), "Unmarshal must not error")
	assert.Equal(t, DerivativesHistoricalVolatilityResponse{Timestamp: testMarketDataMilli(1549720800000), Volatility: 14.747743607344217}, resp, "UnmarshalJSON should decode the timestamp and volatility")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[1549720800000]`), &resp), errInvalidTuple, "UnmarshalJSON should refuse a tuple without a volatility")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[1549720800000,"high"]`), &resp), errInvalidTuple, "UnmarshalJSON should refuse a volatility that is not a number")
}

// mockMarketDataIndexChart is the start of the mock hour of the testMarketDataIndexName chart
func mockMarketDataIndexChart() []DerivativesIndexChartPointResponse {
	return []DerivativesIndexChartPointResponse{
		{Timestamp: testMarketDataMilli(1792368576000), Price: 82122.113333},
		{Timestamp: testMarketDataMilli(1792368582000), Price: 82123.983333},
		{Timestamp: testMarketDataMilli(1792368588000), Price: 82124.706667},
		{Timestamp: testMarketDataMilli(1792368594000), Price: 82125.773333},
		{Timestamp: testMarketDataMilli(1792368600000), Price: 82116.801667},
	}
}

func TestGlobalDerivativesGetIndexChartData(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetIndexChartData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIndexChartData must reject a nil request")
	_, err = e.GlobalDerivatives().GetIndexChartData(t.Context(), &DerivativesIndexChartDataRequest{Range: DerivativesIndexChartRangeOneHour})
	require.ErrorIs(t, err, errDerivativesIndexNameRequired, "GetIndexChartData must require an index name")
	_, err = e.GlobalDerivatives().GetIndexChartData(t.Context(), &DerivativesIndexChartDataRequest{IndexName: testMarketDataIndexName, Range: "2h"})
	require.ErrorIs(t, err, errDerivativesInvalidIndexChartRange, "GetIndexChartData must refuse a range the gateway does not chart")

	arg := &DerivativesIndexChartDataRequest{IndexName: testMarketDataIndexName, Range: DerivativesIndexChartRangeOneHour}
	resp, err := e.GlobalDerivatives().GetIndexChartData(t.Context(), arg)
	require.NoError(t, err, "GetIndexChartData must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataIndexChart(), resp, "GetIndexChartData should decode every point")
	} else {
		assert.NotEmpty(t, resp, "GetIndexChartData should return the index chart")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetIndexChartData(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetIndexChartData should return the request failure")
}

func TestDerivativesIndexChartPointUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var resp DerivativesIndexChartPointResponse
	require.NoError(t, json.Unmarshal([]byte(`[1573228800000,8751.7138636]`), &resp), "Unmarshal must not error")
	assert.Equal(t, DerivativesIndexChartPointResponse{Timestamp: testMarketDataMilli(1573228800000), Price: 8751.7138636}, resp, "UnmarshalJSON should decode the timestamp and price")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[1573228800000,8751.7138636,1]`), &resp), errInvalidTuple, "UnmarshalJSON should refuse a tuple with an extra element")
	assert.ErrorIs(t, json.Unmarshal([]byte(`{"price":8751.7138636}`), &resp), errInvalidTuple, "UnmarshalJSON should refuse a point that is not a tuple")
}

func TestGlobalDerivativesGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetIndexPrice(t.Context(), "")
	require.ErrorIs(t, err, errDerivativesIndexNameRequired, "GetIndexPrice must require an index name")

	resp, err := e.GlobalDerivatives().GetIndexPrice(t.Context(), testMarketDataIndexName)
	require.NoError(t, err, "GetIndexPrice must not error")
	if mockTests {
		assert.Equal(t, &DerivativesIndexPriceResponse{IndexPrice: 82405.01, EstimatedDeliveryPrice: 82405.01}, resp, "GetIndexPrice should decode every field")
	} else {
		assert.Positive(t, resp.IndexPrice, "GetIndexPrice should return the index price")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetIndexPrice(t.Context(), testMarketDataIndexName)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetIndexPrice should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetIndexPrice(t.Context(), testMarketDataIndexName)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetIndexPrice should return the request failure")
}

// mockMarketDataIndexPriceNames are the mock index names, which the gateway sends as bare strings without extended
// details
func mockMarketDataIndexPriceNames() []DerivativesIndexPriceNameResponse {
	return []DerivativesIndexPriceNameResponse{{Name: "ada_usd"}, {Name: "ada_usdc"}, {Name: "ada_usdt"}, {Name: "algo_usd"}, {Name: "algo_usdc"}}
}

func TestGlobalDerivativesGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	resp, err := e.GlobalDerivatives().GetIndexPriceNames(t.Context(), false)
	require.NoError(t, err, "GetIndexPriceNames must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataIndexPriceNames(), resp, "GetIndexPriceNames should decode every name")
	} else {
		assert.Contains(t, resp, DerivativesIndexPriceNameResponse{Name: testMarketDataIndexName}, "GetIndexPriceNames should return the btc_usdc index")
	}

	resp, err = e.GlobalDerivatives().GetIndexPriceNames(t.Context(), true)
	require.NoError(t, err, "GetIndexPriceNames must not error with extended details")
	if mockTests {
		exp := []DerivativesIndexPriceNameResponse{{Name: "ada_usd"}, {Name: "btc_usd", FutureComboCreationEnabled: true, OptionComboCreationEnabled: true}, {Name: "btc_usdc", OptionComboCreationEnabled: true}}
		assert.Equal(t, exp, resp, "GetIndexPriceNames should decode the extended details")
	} else {
		assert.Contains(t, resp, DerivativesIndexPriceNameResponse{Name: testMarketDataIndexName, OptionComboCreationEnabled: true}, "GetIndexPriceNames should return btc_usdc's combo creation settings")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetIndexPriceNames(t.Context(), false)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetIndexPriceNames should return the request failure")
}

func TestDerivativesIndexPriceNameUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   DerivativesIndexPriceNameResponse
	}{
		{input: `"btc_usdc"`, exp: DerivativesIndexPriceNameResponse{Name: "btc_usdc"}},
		{input: `{"name":"btc_usdc"}`, exp: DerivativesIndexPriceNameResponse{Name: "btc_usdc"}},
		{input: `{"name":"btc_usd","future_combo_creation_enabled":true,"option_combo_creation_enabled":true}`, exp: DerivativesIndexPriceNameResponse{Name: "btc_usd", FutureComboCreationEnabled: true, OptionComboCreationEnabled: true}},
		{input: `null`},
	} {
		var resp DerivativesIndexPriceNameResponse
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &resp), "Unmarshal must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, resp, "UnmarshalJSON should decode %s", tc.input)
	}
	var resp DerivativesIndexPriceNameResponse
	assert.Error(t, json.Unmarshal([]byte(`["btc_usd"]`), &resp), "UnmarshalJSON should refuse a name that is neither a string nor an object")
}

// mockMarketDataInstruments are the mock specifications of the perpetual, spot pair and option the tests use
func mockMarketDataInstruments() map[string]*DerivativesInstrumentResponse {
	return map[string]*DerivativesInstrumentResponse{
		testMarketDataPerpetual: {
			InstrumentName:               "BTC_USDC-PERPETUAL",
			InstrumentID:                 211704,
			Kind:                         "future",
			InstrumentType:               "linear",
			FutureType:                   "linear",
			State:                        "open",
			IsActive:                     true,
			BaseCurrency:                 currency.BTC,
			QuoteCurrency:                currency.USDC,
			CounterCurrency:              currency.USDC,
			BaseCurrencyUUID:             "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			QuoteCurrencyUUID:            "2b92315d-eab7-5bef-84fa-089a131333f5",
			SettlementCurrency:           currency.USDC,
			SettlementPeriod:             "perpetual",
			PriceIndex:                   "btc_usdc",
			IndexID:                      1000004,
			ProductGroup:                 "BTC",
			UnderlyingType:               "crypto",
			CreationTimestamp:            testMarketDataMilli(1646824342000),
			ExpirationTimestamp:          testMarketDataMilli(testMarketDataYear3000),
			TickSize:                     0.1,
			TickSizeSteps:                []DerivativesTickSizeStep{},
			ContractSize:                 0.0001,
			MinimumTradeAmount:           0.0001,
			LotSize:                      1,
			MakerCommission:              0.00015,
			TakerCommission:              0.00035,
			MaximumLiquidationCommission: 0.01,
			BlockTradeCommission:         0.00025,
			BlockTradeTickSize:           0.1,
			BlockTradeMinimumTradeAmount: 200000,
			MaximumLeverage:              50,
			MaximumNonDefaultLeverage:    5,
		},
		testMarketDataSpot: {
			InstrumentName:           "BTC_USDC",
			InstrumentID:             254613,
			Kind:                     "spot",
			InstrumentType:           "linear",
			State:                    "open",
			IsActive:                 true,
			BaseCurrency:             currency.BTC,
			QuoteCurrency:            currency.USDC,
			CounterCurrency:          currency.USDC,
			BaseCurrencyUUID:         "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			QuoteCurrencyUUID:        "2b92315d-eab7-5bef-84fa-089a131333f5",
			PriceIndex:               "btc_usdc",
			IndexID:                  1000004,
			ProductGroup:             "BTC",
			UnderlyingType:           "crypto",
			CreationTimestamp:        testMarketDataMilli(1682341202000),
			ExpirationTimestamp:      testMarketDataMilli(testMarketDataYear3000),
			TickSize:                 0.01,
			TickSizeSteps:            []DerivativesTickSizeStep{},
			ContractSize:             0.00000001,
			MinimumTradeAmount:       0.00000001,
			MakerCommission:          0.0002,
			TakerCommission:          0.0005,
			MaximumLeverage:          10,
			IsCoinbaseSpotRouted:     true,
			IsCoinbaseExchangeRouted: true,
		},
		mockMarketDataOption: {
			InstrumentName:               "BTC-6NOV26-82000-C",
			InstrumentID:                 715112,
			Kind:                         "option",
			InstrumentType:               "reversed",
			OptionType:                   "call",
			Strike:                       82000,
			State:                        "open",
			IsActive:                     true,
			BaseCurrency:                 currency.BTC,
			QuoteCurrency:                currency.BTC,
			CounterCurrency:              currency.USD,
			BaseCurrencyUUID:             "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			QuoteCurrencyUUID:            "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			SettlementCurrency:           currency.BTC,
			SettlementPeriod:             "week",
			PriceIndex:                   "btc_usd",
			IndexID:                      1000033,
			ProductGroup:                 "BTC",
			UnderlyingType:               "crypto",
			CreationTimestamp:            testMarketDataMilli(1792051211000),
			ExpirationTimestamp:          testMarketDataMilli(1793952000000),
			TickSize:                     0.0001,
			TickSizeSteps:                []DerivativesTickSizeStep{{AbovePrice: 0.005, TickSize: 0.0005}},
			ContractSize:                 1,
			MinimumTradeAmount:           0.1,
			LotSize:                      1,
			MakerCommission:              0.0003,
			TakerCommission:              0.0003,
			BlockTradeCommission:         0.0003,
			BlockTradeTickSize:           0.0001,
			BlockTradeMinimumTradeAmount: 25,
		},
	}
}

// testMarketDataInstrumentNotFound is the error the gateway returns for testMarketDataUnknownInstrument
func testMarketDataInstrumentNotFound() *DerivativesError {
	return &DerivativesError{Code: -32602, Message: "Invalid params", Data: json.RawMessage(`{"reason":"instrument not found","param":"instrument_name"}`)}
}

func TestGlobalDerivativesGetInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetInstrument(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetInstrument must require an instrument name")

	for name, exp := range mockMarketDataInstruments() {
		if name == mockMarketDataOption {
			name = marketDataOption(t)
		}
		resp, err := e.GlobalDerivatives().GetInstrument(t.Context(), name)
		require.NoErrorf(t, err, "GetInstrument must not error for %s", name)
		if mockTests {
			assert.Equalf(t, exp, resp, "GetInstrument should decode every field of %s", name)
			continue
		}
		assert.Equalf(t, name, resp.InstrumentName, "GetInstrument should return %s", name)
		assert.Equalf(t, exp.Kind, resp.Kind, "GetInstrument should return %s's kind", name)
	}

	_, err = e.GlobalDerivatives().GetInstrument(t.Context(), testMarketDataUnknownInstrument)
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.True(t, ok, "GetInstrument must return the gateway's error for an unknown instrument")
	if mockTests {
		assert.Equal(t, testMarketDataInstrumentNotFound(), derivativesErr, "GetInstrument should decode the gateway's error")
	} else {
		assert.Equal(t, int64(-32602), derivativesErr.Code, "GetInstrument should return the gateway's invalid params error")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetInstrument(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetInstrument should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetInstrument(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetInstrument should return the request failure")
}

func TestDerivativesInstrumentUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var resp DerivativesInstrumentResponse
	require.NoError(t, json.Unmarshal([]byte(`{"instrument_name":"BTC_USDC-PERPETUAL","creation_timestamp":1646824342000,"expiration_timestamp":32503708800000}`), &resp), "Unmarshal must not error")
	exp := DerivativesInstrumentResponse{InstrumentName: testMarketDataPerpetual, CreationTimestamp: testMarketDataMilli(1646824342000), ExpirationTimestamp: testMarketDataMilli(testMarketDataYear3000)}
	assert.Equal(t, exp, resp, "UnmarshalJSON should decode every field, reading the expiration in milliseconds")
	assert.Equal(t, 3000, resp.ExpirationTimestamp.Time().UTC().Year(), "UnmarshalJSON should decode a perpetual's expiration in the year 3000")

	resp = DerivativesInstrumentResponse{ExpirationTimestamp: testMarketDataMilli(1793952000000)}
	require.NoError(t, json.Unmarshal([]byte(`{"instrument_name":"BTC-6NOV26-82000-C"}`), &resp), "Unmarshal must not error without an expiration")
	assert.Equal(t, testMarketDataMilli(1793952000000), resp.ExpirationTimestamp, "UnmarshalJSON should leave the expiration unchanged when it is left out")
	require.NoError(t, json.Unmarshal([]byte(`{"expiration_timestamp":0}`), &resp), "Unmarshal must not error for a zero expiration")
	assert.Zero(t, resp.ExpirationTimestamp, "UnmarshalJSON should decode a zero expiration as the zero time")
	assert.Error(t, json.Unmarshal([]byte(`{"expiration_timestamp":"soon"}`), &resp), "UnmarshalJSON should refuse an expiration that is not a number")
	assert.Error(t, json.Unmarshal([]byte(`{"is_active":"yes"}`), &resp), "UnmarshalJSON should refuse a field of the wrong type")
}

// mockMarketDataFutures are the mock extended specifications of USDC futures, filtered to testMarketDataPerpetual and
// mockMarketDataDatedFuture
func mockMarketDataFutures() []DerivativesInstrumentResponse {
	return []DerivativesInstrumentResponse{
		{
			InstrumentName:               "BTC_USDC-PERPETUAL",
			InstrumentID:                 211704,
			Kind:                         "future",
			InstrumentType:               "linear",
			FutureType:                   "linear",
			State:                        "open",
			IsActive:                     true,
			BaseCurrency:                 currency.BTC,
			QuoteCurrency:                currency.USDC,
			CounterCurrency:              currency.USDC,
			BaseCurrencyUUID:             "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			QuoteCurrencyUUID:            "2b92315d-eab7-5bef-84fa-089a131333f5",
			SettlementCurrency:           currency.USDC,
			SettlementPeriod:             "perpetual",
			PriceIndex:                   "btc_usdc",
			IndexID:                      1000004,
			ProductGroup:                 "BTC",
			UnderlyingType:               "crypto",
			CreationTimestamp:            testMarketDataMilli(1646824342000),
			ExpirationTimestamp:          testMarketDataMilli(testMarketDataYear3000),
			TickSize:                     0.1,
			TickSizeSteps:                []DerivativesTickSizeStep{},
			ContractSize:                 0.0001,
			MinimumTradeAmount:           0.0001,
			LotSize:                      1,
			MakerCommission:              0.00015,
			TakerCommission:              0.00035,
			MaximumLiquidationCommission: 0.01,
			MakerMaximumCommission:       0.25,
			TakerMaximumCommission:       0.25,
			LiquidationMaximumCommission: 0.25,
			SettlementMaximumCommission:  0.25,
			MarginParameters: &DerivativesMarginParameters{
				InitialIncrement: 0.0175,
				MarginFee:        0.01,
				MarginIncrement:  0.00005,
				MarginInitial:    0.0025,
				MarginPower:      1,
			},
			TradingBandwidthFraction:     0.015,
			PerpetualDampener:            0.00025,
			PerpetualMaximumFunding:      0.05,
			BlockTradeCommission:         0.00025,
			BlockTradeTickSize:           0.1,
			BlockTradeMinimumTradeAmount: 200000,
			MaximumLeverage:              50,
			MaximumNonDefaultLeverage:    5,
		},
		{
			InstrumentName:               "BTC_USDC-27NOV26",
			InstrumentID:                 701306,
			Kind:                         "future",
			InstrumentType:               "linear",
			FutureType:                   "linear",
			State:                        "open",
			IsActive:                     true,
			BaseCurrency:                 currency.BTC,
			QuoteCurrency:                currency.USDC,
			CounterCurrency:              currency.USDC,
			BaseCurrencyUUID:             "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			QuoteCurrencyUUID:            "2b92315d-eab7-5bef-84fa-089a131333f5",
			SettlementCurrency:           currency.USDC,
			SettlementPeriod:             "month",
			PriceIndex:                   "btc_usdc",
			IndexID:                      1000004,
			ProductGroup:                 "BTC",
			UnderlyingType:               "crypto",
			CreationTimestamp:            testMarketDataMilli(1790236823000),
			ExpirationTimestamp:          testMarketDataMilli(1795766400000),
			TickSize:                     2.5,
			TickSizeSteps:                []DerivativesTickSizeStep{},
			ContractSize:                 0.0001,
			MinimumTradeAmount:           0.0001,
			LotSize:                      1,
			MakerCommission:              0.00015,
			TakerCommission:              0.00035,
			MaximumLiquidationCommission: 0.01,
			MakerMaximumCommission:       0.25,
			TakerMaximumCommission:       0.25,
			LiquidationMaximumCommission: 0.25,
			SettlementMaximumCommission:  0.25,
			MarginParameters: &DerivativesMarginParameters{
				InitialIncrement: 0.0275,
				MarginFee:        0.01,
				MarginIncrement:  0.00005,
				MarginInitial:    0.0125,
				MarginPower:      1,
			},
			TradingBandwidthFraction:     0.015,
			BlockTradeCommission:         0.00025,
			BlockTradeTickSize:           1,
			BlockTradeMinimumTradeAmount: 200000,
			MaximumLeverage:              25,
			MaximumNonDefaultLeverage:    5,
		},
	}
}

// testMarketDataInstrumentsRequest selects USDC futures with their extended details
func testMarketDataInstrumentsRequest() *DerivativesInstrumentsRequest {
	return &DerivativesInstrumentsRequest{Currency: currency.USDC, Kind: DerivativesKindFuture, Extended: true}
}

func TestGlobalDerivativesGetInstruments(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetInstruments(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInstruments must reject a nil request")
	_, err = e.GlobalDerivatives().GetInstruments(t.Context(), &DerivativesInstrumentsRequest{Kind: DerivativesKindFuture})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetInstruments must require a currency")
	_, err = e.GlobalDerivatives().GetInstruments(t.Context(), &DerivativesInstrumentsRequest{Currency: currency.USDC, Kind: DerivativesKindCombo})
	require.ErrorIs(t, err, errDerivativesInvalidMarketDataKind, "GetInstruments must refuse a kind it does not list")

	resp, err := e.GlobalDerivatives().GetInstruments(t.Context(), testMarketDataInstrumentsRequest())
	require.NoError(t, err, "GetInstruments must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFutures(), resp, "GetInstruments should decode every field")
	} else {
		require.NotEmpty(t, resp, "GetInstruments must return USDC's futures")
		assert.NotNil(t, resp[0].MarginParameters, "GetInstruments should return the extended margin parameters")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, ok := readJSONRPC(t, r)
		if ok {
			assert.JSONEq(t, `{"currency":"any","kind":"spot","expired":true}`, string(req.Params), "GetInstruments should send any in lower case, as the gateway refuses ANY")
		}
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":[]}`)
	}))
	resp, err = ex.GlobalDerivatives().GetInstruments(t.Context(), &DerivativesInstrumentsRequest{Currency: currency.ANY, Kind: DerivativesKindSpot, Expired: true})
	require.NoError(t, err, "GetInstruments must not error for every currency")
	assert.Empty(t, resp, "GetInstruments should decode an empty list")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetInstruments(t.Context(), testMarketDataInstrumentsRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetInstruments should return the request failure")
}

// testMarketDataLastSettlementsByCurrencyRequest requests USDC's last 3 settlements before 2026-10-18
func testMarketDataLastSettlementsByCurrencyRequest() *DerivativesLastSettlementsByCurrencyRequest {
	return &DerivativesLastSettlementsByCurrencyRequest{Currency: currency.USDC, Type: DerivativesSettlementTypeSettlement, Count: 3, SearchStartTime: testMarketDataRangeStart}
}

// mockMarketDataCurrencySettlements are the mock settlements of USDC perpetuals
func mockMarketDataCurrencySettlements() *DerivativesSettlementsResponse {
	return &DerivativesSettlementsResponse{
		Settlements: []DerivativesSettlement{
			{
				Type:              "settlement",
				Timestamp:         testMarketDataMilli(1792224001906),
				InstrumentName:    "1000BONK_USDC-PERPETUAL",
				IndexPrice:        0.003033,
				MarkPrice:         0.003032,
				Position:          66237,
				ProfitLoss:        -308.03030014500024,
				SessionProfitLoss: 189123200.29036632,
			},
			{
				Type:              "settlement",
				Timestamp:         testMarketDataMilli(1792224001906),
				InstrumentName:    "1000MOG_USDC-PERPETUAL",
				IndexPrice:        0.00010216,
				MarkPrice:         0.00010214,
				Position:          55535,
				ProfitLoss:        0.4803246399998784,
				SessionProfitLoss: 101566163.11890042,
			},
			{
				Type:              "settlement",
				Timestamp:         testMarketDataMilli(1792224001906),
				InstrumentName:    "1000PEPE_USDC-PERPETUAL",
				IndexPrice:        0.004,
				MarkPrice:         0.004,
				Position:          240795,
				ProfitLoss:        -680.3010581330005,
				SessionProfitLoss: 381748682.06759036,
			},
		},
		Continuation: "G5FDKCooRMMK7aKsooaKC1cVjSMCDG4LvgxWcrDpYYg6m9XF3y5Z6SKPjPyHJbokUnw3TGjv4dP2V8hDsq",
	}
}

func TestGlobalDerivativesGetLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetLastSettlementsByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastSettlementsByCurrency must reject a nil request")
	_, err = e.GlobalDerivatives().GetLastSettlementsByCurrency(t.Context(), &DerivativesLastSettlementsByCurrencyRequest{Count: 3})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLastSettlementsByCurrency must require a currency")
	_, err = e.GlobalDerivatives().GetLastSettlementsByCurrency(t.Context(), &DerivativesLastSettlementsByCurrencyRequest{Currency: currency.USDC, Type: "exercise"})
	require.ErrorIs(t, err, errDerivativesInvalidLastSettlementsType, "GetLastSettlementsByCurrency must refuse a type the gateway does not filter by")
	_, err = e.GlobalDerivatives().GetLastSettlementsByCurrency(t.Context(), &DerivativesLastSettlementsByCurrencyRequest{Currency: currency.USDC, Count: derivativesMarketDataMaximumCount + 1})
	require.ErrorIs(t, err, errInvalidLimit, "GetLastSettlementsByCurrency must refuse a count above the gateway's maximum")

	resp, err := e.GlobalDerivatives().GetLastSettlementsByCurrency(t.Context(), testMarketDataLastSettlementsByCurrencyRequest())
	require.NoError(t, err, "GetLastSettlementsByCurrency must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataCurrencySettlements(), resp, "GetLastSettlementsByCurrency should decode every field")
	} else {
		assert.Len(t, resp.Settlements, 3, "GetLastSettlementsByCurrency should return the settlements requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetLastSettlementsByCurrency(t.Context(), testMarketDataLastSettlementsByCurrencyRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastSettlementsByCurrency should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetLastSettlementsByCurrency(t.Context(), testMarketDataLastSettlementsByCurrencyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastSettlementsByCurrency should return the request failure")
}

// testMarketDataLastSettlementsByInstrumentRequest requests testMarketDataPerpetual's last 2 settlements before
// 2026-10-18
func testMarketDataLastSettlementsByInstrumentRequest() *DerivativesLastSettlementsByInstrumentRequest {
	return &DerivativesLastSettlementsByInstrumentRequest{InstrumentName: testMarketDataPerpetual, Type: DerivativesSettlementTypeSettlement, Count: 2, SearchStartTime: testMarketDataRangeStart}
}

// mockMarketDataPerpetualSettlements are the mock settlements of testMarketDataPerpetual
func mockMarketDataPerpetualSettlements() *DerivativesSettlementsResponse {
	return &DerivativesSettlementsResponse{
		Settlements: []DerivativesSettlement{
			{
				Type:              "settlement",
				Timestamp:         testMarketDataMilli(1792224001906),
				InstrumentName:    "BTC_USDC-PERPETUAL",
				IndexPrice:        81619.54,
				MarkPrice:         81648.79,
				Position:          145340218,
				ProfitLoss:        -72506.87527715569,
				SessionProfitLoss: 8349439064.67079,
			},
			{
				Type:              "settlement",
				Timestamp:         testMarketDataMilli(1792137601784),
				InstrumentName:    "BTC_USDC-PERPETUAL",
				IndexPrice:        82193.8,
				MarkPrice:         82229.09,
				Position:          147112373,
				ProfitLoss:        36180.60428452791,
				SessionProfitLoss: 16459767180.23242,
			},
		},
		Continuation: "XGcgAdUWsgKA2zcP4sySuEZCCHPp9hNAPPdFLSXixgHxk3dhaVycjGLbQwFRLfDDeTjcFWfxvcp",
	}
}

func TestGlobalDerivativesGetLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetLastSettlementsByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastSettlementsByInstrument must reject a nil request")
	_, err = e.GlobalDerivatives().GetLastSettlementsByInstrument(t.Context(), &DerivativesLastSettlementsByInstrumentRequest{Count: 2})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetLastSettlementsByInstrument must require an instrument name")
	_, err = e.GlobalDerivatives().GetLastSettlementsByInstrument(t.Context(), &DerivativesLastSettlementsByInstrumentRequest{InstrumentName: testMarketDataPerpetual, Type: "funding"})
	require.ErrorIs(t, err, errDerivativesInvalidLastSettlementsType, "GetLastSettlementsByInstrument must refuse a type the gateway does not filter by")
	_, err = e.GlobalDerivatives().GetLastSettlementsByInstrument(t.Context(), &DerivativesLastSettlementsByInstrumentRequest{InstrumentName: testMarketDataPerpetual, Count: derivativesMarketDataMaximumCount + 1})
	require.ErrorIs(t, err, errInvalidLimit, "GetLastSettlementsByInstrument must refuse a count above the gateway's maximum")

	resp, err := e.GlobalDerivatives().GetLastSettlementsByInstrument(t.Context(), testMarketDataLastSettlementsByInstrumentRequest())
	require.NoError(t, err, "GetLastSettlementsByInstrument must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataPerpetualSettlements(), resp, "GetLastSettlementsByInstrument should decode every field")
	} else {
		assert.Len(t, resp.Settlements, 2, "GetLastSettlementsByInstrument should return the settlements requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetLastSettlementsByInstrument(t.Context(), testMarketDataLastSettlementsByInstrumentRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastSettlementsByInstrument should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetLastSettlementsByInstrument(t.Context(), testMarketDataLastSettlementsByInstrumentRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastSettlementsByInstrument should return the request failure")
}

func TestValidateDerivativesLastSettlementsType(t *testing.T) {
	t.Parallel()
	for _, settlementType := range []string{"", DerivativesSettlementTypeSettlement, DerivativesSettlementTypeDelivery, DerivativesSettlementTypeBankruptcy} {
		assert.NoErrorf(t, validateDerivativesLastSettlementsType(settlementType), "validateDerivativesLastSettlementsType should accept type %q", settlementType)
	}
	for _, settlementType := range []string{"exercise", "Delivery"} {
		assert.ErrorIsf(t, validateDerivativesLastSettlementsType(settlementType), errDerivativesInvalidLastSettlementsType, "validateDerivativesLastSettlementsType should refuse type %q", settlementType)
	}
}

// testMarketDataLastTradesByCurrencyRequest requests USDC futures trades by trade ID, which live tests leave out, as
// the gateway only serves the last 24 hours of trades
func testMarketDataLastTradesByCurrencyRequest() *DerivativesLastTradesByCurrencyRequest {
	if mockTests {
		return &DerivativesLastTradesByCurrencyRequest{Currency: currency.USDC, Kind: DerivativesKindFuture, StartID: mockMarketDataTradesStartID, EndID: mockMarketDataTradesEndID, Sorting: DerivativesSortingAscending}
	}
	return &DerivativesLastTradesByCurrencyRequest{Currency: currency.USDC, Kind: DerivativesKindFuture, Count: 5}
}

// mockMarketDataFutureTrades are the mock USDC futures trades between mockMarketDataTradesStartID and
// mockMarketDataTradesEndID
func mockMarketDataFutureTrades() *DerivativesLastTradesResponse {
	return &DerivativesLastTradesResponse{
		Trades: []DerivativesPublicTrade{
			{
				TradeID:           "USDC-80578495",
				TradeSequence:     20375818,
				InstrumentName:    "ETH_USDC-PERPETUAL",
				Timestamp:         testMarketDataMilli(1792368000086),
				StarbaseTimestamp: testMarketDataNano(1792368000086620771),
				StarbaseMatchID:   237726440194322477,
				Direction:         "sell",
				TickDirection:     2,
				Price:             2408.41,
				Amount:            5,
				Contracts:         50000,
				MarkPrice:         2408.27,
				IndexPrice:        2407.67,
			},
			{
				TradeID:           "USDC-80578496",
				TradeSequence:     20375819,
				InstrumentName:    "ETH_USDC-PERPETUAL",
				Timestamp:         testMarketDataMilli(1792368000098),
				StarbaseTimestamp: testMarketDataNano(1792368000098082351),
				StarbaseMatchID:   237726440244654080,
				Direction:         "buy",
				TickDirection:     2,
				Price:             2408.28,
				Amount:            3.5305,
				Contracts:         35305,
				MarkPrice:         2408.27,
				IndexPrice:        2407.67,
			},
			{
				TradeID:           "USDC-80578497",
				TradeSequence:     1741,
				InstrumentName:    "SOXS_USDC-PERPETUAL",
				Timestamp:         testMarketDataMilli(1792368000139),
				StarbaseTimestamp: testMarketDataNano(1792368000139782374),
				StarbaseMatchID:   237726440416620544,
				Direction:         "sell",
				Price:             48.95,
				Amount:            0.01,
				Contracts:         1,
				MarkPrice:         48.901,
				IndexPrice:        48.9022,
			},
			{
				TradeID:           "USDC-80578498",
				TradeSequence:     20483,
				InstrumentName:    "TSLA_USDC-PERPETUAL",
				Timestamp:         testMarketDataMilli(1792368000145),
				StarbaseTimestamp: testMarketDataNano(1792368000145390525),
				StarbaseMatchID:   237726440441786368,
				Direction:         "buy",
				TickDirection:     2,
				Price:             363.12,
				Amount:            0.99,
				Contracts:         99,
				MarkPrice:         363.17,
				IndexPrice:        363.24,
			},
			{
				TradeID:           "USDC-80578499",
				TradeSequence:     126741,
				InstrumentName:    "NVDA_USDC-PERPETUAL",
				Timestamp:         testMarketDataMilli(1792368000147),
				StarbaseTimestamp: testMarketDataNano(1792368000147602509),
				StarbaseMatchID:   237726440450174976,
				Direction:         "sell",
				Price:             215.29,
				Amount:            0.1,
				Contracts:         10,
				MarkPrice:         215.12,
				IndexPrice:        215.08,
			},
		},
	}
}

// mockMarketDataComboTrades is the mock latest BTC combo trade, with its legs
func mockMarketDataComboTrades() *DerivativesLastTradesResponse {
	return &DerivativesLastTradesResponse{
		Trades: []DerivativesPublicTrade{
			{
				TradeID:           "453623390",
				TradeSequence:     1,
				InstrumentName:    "BTC-FS-25JUN27_21OCT26",
				Timestamp:         testMarketDataMilli(1792371167455),
				StarbaseTimestamp: testMarketDataNano(1792371167455039697),
				StarbaseMatchID:   237739725102788610,
				Direction:         "buy",
				TickDirection:     1,
				Price:             3045,
				Amount:            50000,
				Contracts:         5000,
				MarkPrice:         3052.65,
				IndexPrice:        82348.74,
				Legs: []DerivativesPublicTrade{
					{
						TradeID:           "453623392",
						TradeSequence:     140930,
						InstrumentName:    "BTC-25JUN27",
						Timestamp:         testMarketDataMilli(1792371167455),
						StarbaseTimestamp: testMarketDataNano(1792371167455039697),
						StarbaseMatchID:   237739725102788610,
						Direction:         "buy",
						Price:             85432.5,
						Amount:            50000,
						Contracts:         5000,
						MarkPrice:         85437.18,
						IndexPrice:        82348.74,
						ComboID:           "BTC-FS-25JUN27_21OCT26",
						ComboTradeID:      "453623390",
					},
					{
						TradeID:           "453623391",
						TradeSequence:     2227,
						InstrumentName:    "BTC-21OCT26",
						Timestamp:         testMarketDataMilli(1792371167455),
						StarbaseTimestamp: testMarketDataNano(1792371167455039697),
						StarbaseMatchID:   237739725102788610,
						Direction:         "sell",
						TickDirection:     1,
						Price:             82387.5,
						Amount:            50000,
						Contracts:         5000,
						MarkPrice:         82385.85,
						IndexPrice:        82348.74,
						ComboID:           "BTC-FS-25JUN27_21OCT26",
						ComboTradeID:      "453623390",
					},
				},
			},
		},
		HasMore: true,
	}
}

func TestGlobalDerivativesGetLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetLastTradesByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastTradesByCurrency must reject a nil request")
	_, err = e.GlobalDerivatives().GetLastTradesByCurrency(t.Context(), &DerivativesLastTradesByCurrencyRequest{Kind: DerivativesKindFuture})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLastTradesByCurrency must require a currency")
	_, err = e.GlobalDerivatives().GetLastTradesByCurrency(t.Context(), &DerivativesLastTradesByCurrencyRequest{Currency: currency.USDC, Kind: "perpetual"})
	require.ErrorIs(t, err, errDerivativesInvalidMarketDataKind, "GetLastTradesByCurrency must refuse an unknown kind")
	_, err = e.GlobalDerivatives().GetLastTradesByCurrency(t.Context(), &DerivativesLastTradesByCurrencyRequest{Currency: currency.USDC, Sorting: "up"})
	require.ErrorIs(t, err, errDerivativesInvalidLastTradesSorting, "GetLastTradesByCurrency must refuse an undocumented sorting")
	_, err = e.GlobalDerivatives().GetLastTradesByCurrency(t.Context(), &DerivativesLastTradesByCurrencyRequest{Currency: currency.USDC, Count: derivativesMarketDataMaximumCount + 1})
	require.ErrorIs(t, err, errInvalidLimit, "GetLastTradesByCurrency must refuse a count above the gateway's maximum")
	_, err = e.GlobalDerivatives().GetLastTradesByCurrency(t.Context(), &DerivativesLastTradesByCurrencyRequest{Currency: currency.USDC, StartTime: mockMarketDataTradesWindowEnd, EndTime: mockMarketDataTradesWindowStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLastTradesByCurrency must refuse a reversed window")

	resp, err := e.GlobalDerivatives().GetLastTradesByCurrency(t.Context(), testMarketDataLastTradesByCurrencyRequest())
	require.NoError(t, err, "GetLastTradesByCurrency must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFutureTrades(), resp, "GetLastTradesByCurrency should decode every field")
	} else {
		assert.NotEmpty(t, resp.Trades, "GetLastTradesByCurrency should return USDC's latest futures trades")
	}

	resp, err = e.GlobalDerivatives().GetLastTradesByCurrency(t.Context(), &DerivativesLastTradesByCurrencyRequest{Currency: currency.BTC, Kind: DerivativesKindCombo, Count: 1})
	require.NoError(t, err, "GetLastTradesByCurrency must not error for combo trades")
	if mockTests {
		assert.Equal(t, mockMarketDataComboTrades(), resp, "GetLastTradesByCurrency should decode a combo trade's legs")
	} else {
		for i := range resp.Trades {
			assert.NotEmptyf(t, resp.Trades[i].Legs, "GetLastTradesByCurrency should return combo trade %s's legs", resp.Trades[i].TradeID)
		}
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetLastTradesByCurrency(t.Context(), testMarketDataLastTradesByCurrencyRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastTradesByCurrency should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetLastTradesByCurrency(t.Context(), testMarketDataLastTradesByCurrencyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastTradesByCurrency should return the request failure")
}

// testMarketDataTradesWindow returns the mock trades window, or the last hour when testing live, as the gateway only
// serves the last 24 hours of trades
func testMarketDataTradesWindow() (start, end time.Time) {
	if mockTests {
		return mockMarketDataTradesWindowStart, mockMarketDataTradesWindowEnd
	}
	end = time.Now()
	return end.Add(-time.Hour), end
}

// testMarketDataLastTradesByCurrencyAndTimeRequest requests USDC's first 3 futures trades of the trades window
func testMarketDataLastTradesByCurrencyAndTimeRequest() *DerivativesLastTradesByCurrencyAndTimeRequest {
	start, end := testMarketDataTradesWindow()
	return &DerivativesLastTradesByCurrencyAndTimeRequest{Currency: currency.USDC, Kind: DerivativesKindFuture, StartTime: start, EndTime: end, Count: 3, Sorting: DerivativesSortingAscending}
}

// mockMarketDataFutureTradesByTime are the mock first USDC futures trades of the trades window
func mockMarketDataFutureTradesByTime() *DerivativesLastTradesResponse {
	return &DerivativesLastTradesResponse{
		Trades:  mockMarketDataFutureTrades().Trades[:3],
		HasMore: true,
	}
}

func TestGlobalDerivativesGetLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	start, end := testMarketDataTradesWindow()
	_, err := e.GlobalDerivatives().GetLastTradesByCurrencyAndTime(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastTradesByCurrencyAndTime must reject a nil request")
	_, err = e.GlobalDerivatives().GetLastTradesByCurrencyAndTime(t.Context(), &DerivativesLastTradesByCurrencyAndTimeRequest{StartTime: start})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLastTradesByCurrencyAndTime must require a currency")
	_, err = e.GlobalDerivatives().GetLastTradesByCurrencyAndTime(t.Context(), &DerivativesLastTradesByCurrencyAndTimeRequest{Currency: currency.USDC, Kind: "perpetual", StartTime: start})
	require.ErrorIs(t, err, errDerivativesInvalidMarketDataKind, "GetLastTradesByCurrencyAndTime must refuse an unknown kind")
	_, err = e.GlobalDerivatives().GetLastTradesByCurrencyAndTime(t.Context(), &DerivativesLastTradesByCurrencyAndTimeRequest{Currency: currency.USDC, Sorting: "up", StartTime: start})
	require.ErrorIs(t, err, errDerivativesInvalidLastTradesSorting, "GetLastTradesByCurrencyAndTime must refuse an undocumented sorting")
	_, err = e.GlobalDerivatives().GetLastTradesByCurrencyAndTime(t.Context(), &DerivativesLastTradesByCurrencyAndTimeRequest{Currency: currency.USDC})
	require.ErrorIs(t, err, errDerivativesLastTradesWindowRequired, "GetLastTradesByCurrencyAndTime must require a window start or end")
	_, err = e.GlobalDerivatives().GetLastTradesByCurrencyAndTime(t.Context(), &DerivativesLastTradesByCurrencyAndTimeRequest{Currency: currency.USDC, StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLastTradesByCurrencyAndTime must refuse a reversed window")

	resp, err := e.GlobalDerivatives().GetLastTradesByCurrencyAndTime(t.Context(), testMarketDataLastTradesByCurrencyAndTimeRequest())
	require.NoError(t, err, "GetLastTradesByCurrencyAndTime must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFutureTradesByTime(), resp, "GetLastTradesByCurrencyAndTime should decode every field")
	} else {
		assert.NotEmpty(t, resp.Trades, "GetLastTradesByCurrencyAndTime should return USDC's futures trades of the last hour")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetLastTradesByCurrencyAndTime(t.Context(), testMarketDataLastTradesByCurrencyAndTimeRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastTradesByCurrencyAndTime should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetLastTradesByCurrencyAndTime(t.Context(), testMarketDataLastTradesByCurrencyAndTimeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastTradesByCurrencyAndTime should return the request failure")
}

// testMarketDataLastTradesByInstrumentRequest requests testMarketDataPerpetual's trades by sequence number, which live
// tests leave out, as the gateway only serves the last 24 hours of trades
func testMarketDataLastTradesByInstrumentRequest() *DerivativesLastTradesByInstrumentRequest {
	if mockTests {
		return &DerivativesLastTradesByInstrumentRequest{InstrumentName: testMarketDataPerpetual, StartSequence: mockMarketDataTradesStartSequence, EndSequence: mockMarketDataTradesEndSequence, Sorting: DerivativesSortingAscending}
	}
	return &DerivativesLastTradesByInstrumentRequest{InstrumentName: testMarketDataPerpetual, Count: 3}
}

// mockMarketDataPerpetualTrades are the mock trades of testMarketDataPerpetual between
// mockMarketDataTradesStartSequence and mockMarketDataTradesEndSequence
func mockMarketDataPerpetualTrades() *DerivativesLastTradesResponse {
	return &DerivativesLastTradesResponse{
		Trades: []DerivativesPublicTrade{
			{
				TradeID:           "USDC-80578500",
				TradeSequence:     19540034,
				InstrumentName:    "BTC_USDC-PERPETUAL",
				Timestamp:         testMarketDataMilli(1792368000169),
				StarbaseTimestamp: testMarketDataNano(1792368000169672098),
				StarbaseMatchID:   237726440542449664,
				Direction:         "sell",
				TickDirection:     2,
				Price:             82031.3,
				Amount:            0.006,
				Contracts:         60,
				MarkPrice:         82034.46,
				IndexPrice:        82011.64,
			},
			{
				TradeID:           "USDC-80578506",
				TradeSequence:     19540035,
				InstrumentName:    "BTC_USDC-PERPETUAL",
				Timestamp:         testMarketDataMilli(1792368000270),
				StarbaseTimestamp: testMarketDataNano(1792368000270287513),
				StarbaseMatchID:   237726440966074368,
				Direction:         "buy",
				TickDirection:     2,
				Price:             82021.4,
				Amount:            0.0062,
				Contracts:         62,
				MarkPrice:         82034.46,
				IndexPrice:        82011.64,
			},
			{
				TradeID:           "USDC-80578555",
				TradeSequence:     19540036,
				InstrumentName:    "BTC_USDC-PERPETUAL",
				Timestamp:         testMarketDataMilli(1792368000568),
				StarbaseTimestamp: testMarketDataNano(1792368000568839685),
				StarbaseMatchID:   237726442215976960,
				Direction:         "buy",
				TickDirection:     3,
				Price:             82021.4,
				Amount:            0.1,
				Contracts:         1000,
				MarkPrice:         82034.46,
				IndexPrice:        82011.64,
			},
		},
	}
}

func TestGlobalDerivativesGetLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetLastTradesByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastTradesByInstrument must reject a nil request")
	_, err = e.GlobalDerivatives().GetLastTradesByInstrument(t.Context(), &DerivativesLastTradesByInstrumentRequest{Count: 3})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetLastTradesByInstrument must require an instrument name")
	_, err = e.GlobalDerivatives().GetLastTradesByInstrument(t.Context(), &DerivativesLastTradesByInstrumentRequest{InstrumentName: testMarketDataPerpetual, Sorting: "up"})
	require.ErrorIs(t, err, errDerivativesInvalidLastTradesSorting, "GetLastTradesByInstrument must refuse an undocumented sorting")
	_, err = e.GlobalDerivatives().GetLastTradesByInstrument(t.Context(), &DerivativesLastTradesByInstrumentRequest{InstrumentName: testMarketDataPerpetual, StartTime: mockMarketDataTradesWindowEnd, EndTime: mockMarketDataTradesWindowStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLastTradesByInstrument must refuse a reversed window")

	resp, err := e.GlobalDerivatives().GetLastTradesByInstrument(t.Context(), testMarketDataLastTradesByInstrumentRequest())
	require.NoError(t, err, "GetLastTradesByInstrument must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataPerpetualTrades(), resp, "GetLastTradesByInstrument should decode every field")
	} else {
		assert.Len(t, resp.Trades, 3, "GetLastTradesByInstrument should return the trades requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetLastTradesByInstrument(t.Context(), testMarketDataLastTradesByInstrumentRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastTradesByInstrument should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetLastTradesByInstrument(t.Context(), testMarketDataLastTradesByInstrumentRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastTradesByInstrument should return the request failure")
}

// testMarketDataLastTradesByInstrumentAndTimeRequest requests testMarketDataPerpetual's last 2 trades of the trades
// window
func testMarketDataLastTradesByInstrumentAndTimeRequest() *DerivativesLastTradesByInstrumentAndTimeRequest {
	_, end := testMarketDataTradesWindow()
	return &DerivativesLastTradesByInstrumentAndTimeRequest{InstrumentName: testMarketDataPerpetual, EndTime: end, Count: 2}
}

// mockMarketDataPerpetualTradesByTime are the mock last testMarketDataPerpetual trades of the trades window
func mockMarketDataPerpetualTradesByTime() *DerivativesLastTradesResponse {
	return &DerivativesLastTradesResponse{
		Trades: []DerivativesPublicTrade{
			{
				TradeID:           "USDC-80648826",
				TradeSequence:     19558455,
				InstrumentName:    "BTC_USDC-PERPETUAL",
				Timestamp:         testMarketDataMilli(1792371594970),
				StarbaseTimestamp: testMarketDataNano(1792371594970659673),
				StarbaseMatchID:   237741518230663171,
				Direction:         "buy",
				TickDirection:     1,
				Price:             82229.8,
				Amount:            0.0023,
				Contracts:         23,
				MarkPrice:         82227.06,
				IndexPrice:        82202.49,
			},
			{
				TradeID:           "USDC-80648825",
				TradeSequence:     19558454,
				InstrumentName:    "BTC_USDC-PERPETUAL",
				Timestamp:         testMarketDataMilli(1792371594970),
				StarbaseTimestamp: testMarketDataNano(1792371594970659673),
				StarbaseMatchID:   237741518230663170,
				Direction:         "buy",
				Price:             82229.8,
				Amount:            0.1977,
				Contracts:         1977,
				MarkPrice:         82227.06,
				IndexPrice:        82202.49,
			},
		},
		HasMore: true,
	}
}

func TestGlobalDerivativesGetLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	start, end := testMarketDataTradesWindow()
	_, err := e.GlobalDerivatives().GetLastTradesByInstrumentAndTime(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastTradesByInstrumentAndTime must reject a nil request")
	_, err = e.GlobalDerivatives().GetLastTradesByInstrumentAndTime(t.Context(), &DerivativesLastTradesByInstrumentAndTimeRequest{EndTime: end})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetLastTradesByInstrumentAndTime must require an instrument name")
	_, err = e.GlobalDerivatives().GetLastTradesByInstrumentAndTime(t.Context(), &DerivativesLastTradesByInstrumentAndTimeRequest{InstrumentName: testMarketDataPerpetual, EndTime: end, Sorting: "up"})
	require.ErrorIs(t, err, errDerivativesInvalidLastTradesSorting, "GetLastTradesByInstrumentAndTime must refuse an undocumented sorting")
	_, err = e.GlobalDerivatives().GetLastTradesByInstrumentAndTime(t.Context(), &DerivativesLastTradesByInstrumentAndTimeRequest{InstrumentName: testMarketDataPerpetual})
	require.ErrorIs(t, err, errDerivativesLastTradesWindowRequired, "GetLastTradesByInstrumentAndTime must require a window start or end")
	_, err = e.GlobalDerivatives().GetLastTradesByInstrumentAndTime(t.Context(), &DerivativesLastTradesByInstrumentAndTimeRequest{InstrumentName: testMarketDataPerpetual, StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLastTradesByInstrumentAndTime must refuse a reversed window")

	resp, err := e.GlobalDerivatives().GetLastTradesByInstrumentAndTime(t.Context(), testMarketDataLastTradesByInstrumentAndTimeRequest())
	require.NoError(t, err, "GetLastTradesByInstrumentAndTime must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataPerpetualTradesByTime(), resp, "GetLastTradesByInstrumentAndTime should decode every field")
	} else {
		assert.Len(t, resp.Trades, 2, "GetLastTradesByInstrumentAndTime should return the trades requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetLastTradesByInstrumentAndTime(t.Context(), testMarketDataLastTradesByInstrumentAndTimeRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastTradesByInstrumentAndTime should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetLastTradesByInstrumentAndTime(t.Context(), testMarketDataLastTradesByInstrumentAndTimeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastTradesByInstrumentAndTime should return the request failure")
}

func TestValidateDerivativesLastTrades(t *testing.T) {
	t.Parallel()
	for _, sorting := range []string{"", DerivativesSortingAscending, DerivativesSortingDescending, DerivativesSortingDefault} {
		assert.NoErrorf(t, validateDerivativesLastTrades(derivativesMarketDataMaximumCount, sorting), "validateDerivativesLastTrades should accept sorting %q", sorting)
	}
	for _, sorting := range []string{"up", "ASC"} {
		assert.ErrorIsf(t, validateDerivativesLastTrades(0, sorting), errDerivativesInvalidLastTradesSorting, "validateDerivativesLastTrades should refuse sorting %q", sorting)
	}
	assert.ErrorIs(t, validateDerivativesLastTrades(derivativesMarketDataMaximumCount+1, ""), errInvalidLimit, "validateDerivativesLastTrades should refuse a count above the maximum")
}

func TestValidateDerivativesMarketDataCount(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateDerivativesMarketDataCount(0), "validateDerivativesMarketDataCount should accept an unset count")
	assert.NoError(t, validateDerivativesMarketDataCount(derivativesMarketDataMaximumCount), "validateDerivativesMarketDataCount should accept the maximum")
	assert.ErrorIs(t, validateDerivativesMarketDataCount(derivativesMarketDataMaximumCount+1), errInvalidLimit, "validateDerivativesMarketDataCount should refuse a count above the maximum")
}

func TestCheckDerivativesLastTradesWindow(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		start, end time.Time
		err        error
	}{
		{name: "an unset window", err: errDerivativesLastTradesWindowRequired},
		{name: "a start alone", start: mockMarketDataTradesWindowStart},
		{name: "an end alone", end: mockMarketDataTradesWindowEnd},
		{name: "a window", start: mockMarketDataTradesWindowStart, end: mockMarketDataTradesWindowEnd},
		{name: "a reversed window", start: mockMarketDataTradesWindowEnd, end: mockMarketDataTradesWindowStart, err: common.ErrStartAfterEnd},
	} {
		assert.ErrorIsf(t, checkDerivativesLastTradesWindow(tc.start, tc.end), tc.err, "checkDerivativesLastTradesWindow should check %s", tc.name)
	}
}

func TestDerivativesUnixMilli(t *testing.T) {
	t.Parallel()
	milli, err := derivativesUnixMilli(time.Time{})
	require.NoError(t, err, "derivativesUnixMilli must not error for an unset time")
	assert.Zero(t, milli, "derivativesUnixMilli should return zero for an unset time")
	milli, err = derivativesUnixMilli(mockMarketDataTradesWindowStart)
	require.NoError(t, err, "derivativesUnixMilli must not error for a time after 1970")
	assert.Equal(t, uint64(1792368000000), milli, "derivativesUnixMilli should return the time in Unix milliseconds")
	milli, err = derivativesUnixMilli(time.Unix(0, 0))
	require.NoError(t, err, "derivativesUnixMilli must not error for the epoch")
	assert.Zero(t, milli, "derivativesUnixMilli should return the epoch as zero")
	_, err = derivativesUnixMilli(time.Unix(0, 0).Add(-time.Millisecond))
	assert.ErrorIs(t, err, errDerivativesTimeBeforeEpoch, "derivativesUnixMilli should refuse a time before 1970")

	start, end, err := derivativesUnixMilliRange(mockMarketDataTradesWindowStart, mockMarketDataTradesWindowStart.Add(time.Hour))
	require.NoError(t, err, "derivativesUnixMilliRange must not error")
	assert.Equal(t, uint64(1792368000000), start, "derivativesUnixMilliRange should return the start in Unix milliseconds")
	assert.Equal(t, uint64(1792371600000), end, "derivativesUnixMilliRange should return the end in Unix milliseconds")
	before := time.Date(1969, 12, 31, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ start, end time.Time }{{before, mockMarketDataTradesWindowStart}, {mockMarketDataTradesWindowStart, before}} {
		_, _, err := derivativesUnixMilliRange(tc.start, tc.end)
		assert.ErrorIsf(t, err, errDerivativesTimeBeforeEpoch, "derivativesUnixMilliRange should refuse %s to %s", tc.start, tc.end)
	}
}

// testMarketDataMarkPriceRequest returns the mock mark price window of mockMarketDataOption, or a listed option's last
// minute when testing live, as the gateway keeps about 5 minutes of mark price history
func testMarketDataMarkPriceRequest(t *testing.T) *DerivativesInstrumentTimeRangeRequest {
	t.Helper()
	if mockTests {
		return &DerivativesInstrumentTimeRangeRequest{InstrumentName: mockMarketDataOption, StartTime: time.UnixMilli(mockMarketDataMarkPriceStart), EndTime: time.UnixMilli(mockMarketDataMarkPriceEnd)}
	}
	end := time.Now()
	return &DerivativesInstrumentTimeRangeRequest{InstrumentName: marketDataOption(t), StartTime: end.Add(-time.Minute), EndTime: end}
}

// mockMarketDataMarkPrices are the first mock mark prices of mockMarketDataOption
func mockMarketDataMarkPrices() []DerivativesMarkPriceResponse {
	return []DerivativesMarkPriceResponse{
		{Timestamp: testMarketDataMilli(1792372167484), MarkPrice: 0.037090317611473314},
		{Timestamp: testMarketDataMilli(1792372168525), MarkPrice: 0.03709451170934973},
		{Timestamp: testMarketDataMilli(1792372169193), MarkPrice: 0.03709451170934973},
		{Timestamp: testMarketDataMilli(1792372171207), MarkPrice: 0.037093366598038434},
		{Timestamp: testMarketDataMilli(1792372172213), MarkPrice: 0.03709298623904806},
	}
}

func TestGlobalDerivativesGetMarkPriceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetMarkPriceHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarkPriceHistory must reject a nil request")
	_, err = e.GlobalDerivatives().GetMarkPriceHistory(t.Context(), &DerivativesInstrumentTimeRangeRequest{InstrumentName: mockMarketDataOption, StartTime: time.UnixMilli(mockMarketDataMarkPriceEnd), EndTime: time.UnixMilli(mockMarketDataMarkPriceStart)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarkPriceHistory must refuse a reversed range")

	arg := testMarketDataMarkPriceRequest(t)
	resp, err := e.GlobalDerivatives().GetMarkPriceHistory(t.Context(), arg)
	require.NoError(t, err, "GetMarkPriceHistory must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataMarkPrices(), resp, "GetMarkPriceHistory should decode every point")
	} else {
		// Only the options the volatility index is calculated from have a history, which a listed option may not be
		for i := range resp {
			assert.WithinRangef(t, resp[i].Timestamp.Time(), arg.StartTime, arg.EndTime, "GetMarkPriceHistory should return mark price %d within the range", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetMarkPriceHistory(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMarkPriceHistory should return the request failure")
}

func TestDerivativesMarkPriceUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var resp DerivativesMarkPriceResponse
	require.NoError(t, json.Unmarshal([]byte(`[1608142381229,0.5165791606037885]`), &resp), "Unmarshal must not error")
	assert.Equal(t, DerivativesMarkPriceResponse{Timestamp: testMarketDataMilli(1608142381229), MarkPrice: 0.5165791606037885}, resp, "UnmarshalJSON should decode the timestamp and mark price")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[]`), &resp), errInvalidTuple, "UnmarshalJSON should refuse an empty tuple")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[true,0.5]`), &resp), errInvalidTuple, "UnmarshalJSON should refuse a timestamp that is not a number")
}

// mockMarketDataOrderBook is the mock order book of testMarketDataPerpetual, 5 levels deep
func mockMarketDataOrderBook() *DerivativesOrderBookResponse {
	return &DerivativesOrderBookResponse{
		InstrumentName: "BTC_USDC-PERPETUAL",
		Timestamp:      testMarketDataMilli(1792372177889),
		State:          "open",
		Stats: DerivativesTickerStats{
			High:           82481.3,
			Low:            80583.1,
			PriceChange:    1.1354,
			Volume:         35736.9806,
			VolumeUSD:      2911013760.69,
			VolumeNotional: 2911596245.2117195,
		},
		BestBidPrice:           82429.8,
		BestBidAmount:          1.323,
		BestAskPrice:           82429.9,
		BestAskAmount:          5.4321,
		LastPrice:              82430.4,
		MarkPrice:              82430.32,
		IndexPrice:             82404.51,
		MinimumPrice:           81193.8,
		MaximumPrice:           83666.8,
		OpenInterest:           1777.1944,
		EstimatedDeliveryPrice: 82404.51,
		SettlementPrice:        81598.81,
		CurrentFunding:         0.00006321,
		Funding8Hour:           0.00006352,
		InterestValue:          351530666.5309431,
		ChangeID:               208575582204,
		Bids: []DerivativesOrderBookLevel{
			{Price: 82429.8, Amount: 1.323},
			{Price: 82429.3, Amount: 1.2774},
			{Price: 82429.2, Amount: 0.6068},
			{Price: 82429, Amount: 1.2029},
			{Price: 82428.5, Amount: 0.0124},
		},
		Asks: []DerivativesOrderBookLevel{
			{Price: 82429.9, Amount: 5.4321},
			{Price: 82430.4, Amount: 2.3275},
			{Price: 82430.5, Amount: 3.0889},
			{Price: 82432, Amount: 1.4747},
			{Price: 82432.7, Amount: 1.3965},
		},
	}
}

func TestGlobalDerivativesGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOrderBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderBook must reject a nil request")
	_, err = e.GlobalDerivatives().GetOrderBook(t.Context(), &DerivativesOrderBookRequest{Depth: 5})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetOrderBook must require an instrument name")
	_, err = e.GlobalDerivatives().GetOrderBook(t.Context(), &DerivativesOrderBookRequest{InstrumentName: testMarketDataPerpetual, Depth: derivativesMaximumOrderBookDepth + 1})
	require.ErrorIs(t, err, errDerivativesInvalidDepth, "GetOrderBook must refuse a depth above the gateway's maximum")

	arg := &DerivativesOrderBookRequest{InstrumentName: testMarketDataPerpetual, Depth: 5}
	resp, err := e.GlobalDerivatives().GetOrderBook(t.Context(), arg)
	require.NoError(t, err, "GetOrderBook must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataOrderBook(), resp, "GetOrderBook should decode every field")
	} else {
		assert.Equal(t, testMarketDataPerpetual, resp.InstrumentName, "GetOrderBook should return the instrument requested")
		assert.Len(t, resp.Bids, 5, "GetOrderBook should return the depth requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetOrderBook(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderBook should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOrderBook(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderBook should return the request failure")
}

// mockMarketDataOrderBookByID is the mock order book of testMarketDataPerpetualID, 1 level deep
func mockMarketDataOrderBookByID() *DerivativesOrderBookResponse {
	return &DerivativesOrderBookResponse{
		InstrumentName: "BTC_USDC-PERPETUAL",
		Timestamp:      testMarketDataMilli(1792372178626),
		State:          "open",
		Stats: DerivativesTickerStats{
			High:           82481.3,
			Low:            80583.1,
			PriceChange:    1.1354,
			Volume:         35736.9806,
			VolumeUSD:      2911013760.69,
			VolumeNotional: 2911596245.2117195,
		},
		BestBidPrice:           82429.8,
		BestBidAmount:          1.323,
		BestAskPrice:           82429.9,
		BestAskAmount:          5.1821,
		LastPrice:              82430.4,
		MarkPrice:              82430.52,
		IndexPrice:             82404.73,
		MinimumPrice:           81194,
		MaximumPrice:           83667,
		OpenInterest:           1777.1944,
		EstimatedDeliveryPrice: 82404.73,
		SettlementPrice:        81598.81,
		CurrentFunding:         0.00006297,
		Funding8Hour:           0.00006352,
		InterestValue:          351536392.8101931,
		ChangeID:               208575586390,
		Bids:                   []DerivativesOrderBookLevel{{Price: 82429.8, Amount: 1.323}},
		Asks:                   []DerivativesOrderBookLevel{{Price: 82429.9, Amount: 5.1821}},
	}
}

func TestGlobalDerivativesGetOrderBookByInstrumentID(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOrderBookByInstrumentID(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderBookByInstrumentID must reject a nil request")
	_, err = e.GlobalDerivatives().GetOrderBookByInstrumentID(t.Context(), &DerivativesOrderBookByInstrumentIDRequest{Depth: 1})
	require.ErrorIs(t, err, errDerivativesInstrumentIDRequired, "GetOrderBookByInstrumentID must require an instrument ID")
	_, err = e.GlobalDerivatives().GetOrderBookByInstrumentID(t.Context(), &DerivativesOrderBookByInstrumentIDRequest{InstrumentID: testMarketDataPerpetualID, Depth: derivativesMaximumOrderBookDepth + 1})
	require.ErrorIs(t, err, errDerivativesInvalidDepth, "GetOrderBookByInstrumentID must refuse a depth above the gateway's maximum")

	arg := &DerivativesOrderBookByInstrumentIDRequest{InstrumentID: testMarketDataPerpetualID, Depth: 1}
	resp, err := e.GlobalDerivatives().GetOrderBookByInstrumentID(t.Context(), arg)
	require.NoError(t, err, "GetOrderBookByInstrumentID must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataOrderBookByID(), resp, "GetOrderBookByInstrumentID should decode every field")
	} else {
		assert.Equal(t, testMarketDataPerpetual, resp.InstrumentName, "GetOrderBookByInstrumentID should return the instrument requested")
		assert.Len(t, resp.Asks, 1, "GetOrderBookByInstrumentID should return the depth requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetOrderBookByInstrumentID(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderBookByInstrumentID should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOrderBookByInstrumentID(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderBookByInstrumentID should return the request failure")
}

func TestValidateDerivativesOrderBookDepth(t *testing.T) {
	t.Parallel()
	for _, depth := range []uint64{0, 1, 3, derivativesMaximumOrderBookDepth} {
		assert.NoErrorf(t, validateDerivativesOrderBookDepth(depth), "validateDerivativesOrderBookDepth should accept a depth of %d, which the gateway rounds up", depth)
	}
	assert.ErrorIs(t, validateDerivativesOrderBookDepth(derivativesMaximumOrderBookDepth+1), errDerivativesInvalidDepth, "validateDerivativesOrderBookDepth should refuse a depth above the maximum")
}

func TestDerivativesOrderBookLevelUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var levels []DerivativesOrderBookLevel
	require.NoError(t, json.Unmarshal([]byte(`[[82429.8,1.323],[82429.3,1.2774]]`), &levels), "Unmarshal must not error")
	assert.Equal(t, []DerivativesOrderBookLevel{{Price: 82429.8, Amount: 1.323}, {Price: 82429.3, Amount: 1.2774}}, levels, "UnmarshalJSON should decode each price and amount")
	var level DerivativesOrderBookLevel
	assert.ErrorIs(t, json.Unmarshal([]byte(`["new",82429.8,1.323]`), &level), errInvalidTuple, "UnmarshalJSON should refuse a subscription change, which carries an action too")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[82429.8,"1.323"]`), &level), errInvalidTuple, "UnmarshalJSON should refuse an amount that is not a number")
}

func TestGlobalDerivativesGetSupportedIndexNames(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetSupportedIndexNames(t.Context(), "future")
	require.ErrorIs(t, err, errDerivativesInvalidIndexType, "GetSupportedIndexNames must refuse an undocumented type")

	resp, err := e.GlobalDerivatives().GetSupportedIndexNames(t.Context(), DerivativesIndexTypeSpot)
	require.NoError(t, err, "GetSupportedIndexNames must not error")
	if mockTests {
		assert.Equal(t, []string{"aave_usdc", "ada_usdc", "aero_usdc", "algo_usdc", "apt_usdc"}, resp, "GetSupportedIndexNames should decode every name")
	} else {
		assert.Contains(t, resp, testMarketDataIndexName, "GetSupportedIndexNames should return the btc_usdc spot index")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetSupportedIndexNames(t.Context(), DerivativesIndexTypeSpot)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSupportedIndexNames should return the request failure")
}

// mockMarketDataTradeVolumes are the mock extended trade volumes, filtered to BTC's, which trades options and futures,
// and USDE's, which trades spot
func mockMarketDataTradeVolumes() []DerivativesTradeVolumeResponse {
	return []DerivativesTradeVolumeResponse{
		{
			Currency:           currency.BTC,
			CurrencyPair:       "btc_usd",
			CallsVolume:        4133.3,
			PutsVolume:         6508.3,
			FuturesVolume:      12607.46785883,
			CallsVolume7Day:    60481,
			PutsVolume7Day:     74226.9,
			FuturesVolume7Day:  127995.28255997,
			CallsVolume30Day:   436156,
			PutsVolume30Day:    338301.2,
			FuturesVolume30Day: 436371.37583575,
		},
		{
			Currency:        currency.NewCode("USDE"),
			CurrencyPair:    "usde_usd",
			SpotVolume:      180.000124,
			SpotVolume7Day:  307075.789719,
			SpotVolume30Day: 3097245.653766,
		},
	}
}

func TestGlobalDerivativesGetTradeVolumes(t *testing.T) {
	t.Parallel()
	resp, err := e.GlobalDerivatives().GetTradeVolumes(t.Context(), true)
	require.NoError(t, err, "GetTradeVolumes must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataTradeVolumes(), resp, "GetTradeVolumes should decode every field")
	} else {
		assert.NotEmpty(t, resp, "GetTradeVolumes should return each currency's volumes")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetTradeVolumes(t.Context(), false)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTradeVolumes should return the request failure")
}

// testMarketDataTradingViewChartDataRequest requests testMarketDataPerpetual's hourly candles opening in the 4 hours
// from 2026-10-18T00:00:00Z
func testMarketDataTradingViewChartDataRequest() *DerivativesTradingViewChartDataRequest {
	return &DerivativesTradingViewChartDataRequest{InstrumentName: testMarketDataPerpetual, StartTime: testMarketDataRangeStart, EndTime: testMarketDataRangeEnd, Resolution: kline.OneHour}
}

// mockMarketDataCandles are the mock hourly candles of testMarketDataPerpetual
func mockMarketDataCandles() *DerivativesTradingViewChartDataResponse {
	return &DerivativesTradingViewChartDataResponse{
		Status: "ok",
		Ticks:  []types.Time{testMarketDataMilli(1792281600000), testMarketDataMilli(1792285200000), testMarketDataMilli(1792288800000), testMarketDataMilli(1792292400000), testMarketDataMilli(1792296000000)},
		Open:   []float64{81411.8, 81466.2, 81584.7, 81749, 81869.6},
		High:   []float64{81541.9, 81824.6, 81765.6, 81999, 81878.1},
		Low:    []float64{81320.3, 81442.3, 81509.6, 81688.6, 81702.1},
		Close:  []float64{81466.2, 81581.9, 81747.3, 81869.5, 81722.9},
		Volume: []float64{1363.5861, 1761.8297, 1793.3117, 1721.1838, 616.035},
		Cost:   []float64{111005979.72961, 143816485.88879, 146415006.32242, 140837910.57745, 50380492.26359},
	}
}

func TestGlobalDerivativesGetTradingViewChartData(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetTradingViewChartData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradingViewChartData must reject a nil request")
	_, err = e.GlobalDerivatives().GetTradingViewChartData(t.Context(), &DerivativesTradingViewChartDataRequest{StartTime: testMarketDataRangeStart, EndTime: testMarketDataRangeEnd, Resolution: kline.OneHour})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetTradingViewChartData must require an instrument name")
	_, err = e.GlobalDerivatives().GetTradingViewChartData(t.Context(), &DerivativesTradingViewChartDataRequest{InstrumentName: testMarketDataPerpetual, Resolution: kline.OneHour})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetTradingViewChartData must require a time range")
	_, err = e.GlobalDerivatives().GetTradingViewChartData(t.Context(), &DerivativesTradingViewChartDataRequest{InstrumentName: testMarketDataPerpetual, StartTime: testMarketDataRangeStart, EndTime: testMarketDataRangeEnd, Resolution: kline.FourHour})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetTradingViewChartData must refuse a resolution the gateway does not chart")

	resp, err := e.GlobalDerivatives().GetTradingViewChartData(t.Context(), testMarketDataTradingViewChartDataRequest())
	require.NoError(t, err, "GetTradingViewChartData must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataCandles(), resp, "GetTradingViewChartData should decode every field")
	} else {
		assert.Equal(t, "ok", resp.Status, "GetTradingViewChartData should return candles")
		assert.Len(t, resp.Ticks, 5, "GetTradingViewChartData should return each hour's candle")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "GetTradingViewChartData should send a GET request, as the gateway answers a posted body with Method not found")
		assert.Equal(t, "/api/v2/public/get_tradingview_chart_data", r.URL.Path, "GetTradingViewChartData should request the method's path")
		assert.Equal(t, "end_timestamp=1792296000000&instrument_name=BTC_USDC-PERPETUAL&resolution=60&start_timestamp=1792281600000", r.URL.RawQuery, "GetTradingViewChartData should send its params in the query string")
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"volume":[],"ticks":[],"status":"no_data","open":[],"low":[],"high":[],"cost":[],"close":[]}}`)
	}))
	resp, err = ex.GlobalDerivatives().GetTradingViewChartData(t.Context(), testMarketDataTradingViewChartDataRequest())
	require.NoError(t, err, "GetTradingViewChartData must not error without candles")
	exp := &DerivativesTradingViewChartDataResponse{Status: "no_data", Ticks: []types.Time{}, Open: []float64{}, High: []float64{}, Low: []float64{}, Close: []float64{}, Volume: []float64{}, Cost: []float64{}}
	assert.Equal(t, exp, resp, "GetTradingViewChartData should decode a range without candles")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetTradingViewChartData(t.Context(), testMarketDataTradingViewChartDataRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTradingViewChartData should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetTradingViewChartData(t.Context(), testMarketDataTradingViewChartDataRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTradingViewChartData should return the request failure")
}

func TestDerivativesChartResolution(t *testing.T) {
	t.Parallel()
	for interval, exp := range map[kline.Interval]string{
		kline.OneMin:     "1",
		kline.ThreeMin:   "3",
		kline.FiveMin:    "5",
		kline.TenMin:     "10",
		kline.FifteenMin: "15",
		kline.ThirtyMin:  "30",
		kline.OneHour:    "60",
		kline.TwoHour:    "120",
		kline.ThreeHour:  "180",
		kline.SixHour:    "360",
		kline.TwelveHour: "720",
		kline.OneDay:     "1D",
	} {
		resolution, err := derivativesChartResolution(interval)
		require.NoErrorf(t, err, "derivativesChartResolution must not error for interval %s", interval)
		assert.Equalf(t, exp, resolution, "derivativesChartResolution should name interval %s", interval)
	}
	for _, interval := range []kline.Interval{kline.Raw, kline.HundredMilliseconds, kline.FourHour, kline.OneWeek} {
		_, err := derivativesChartResolution(interval)
		assert.ErrorIsf(t, err, kline.ErrUnsupportedInterval, "derivativesChartResolution should refuse interval %s", interval)
	}
}

// testMarketDataVolatilityIndexDataRequest requests BTC's hourly volatility index candles of the 4 hours from
// 2026-10-18T00:00:00Z
func testMarketDataVolatilityIndexDataRequest() *DerivativesVolatilityIndexDataRequest {
	return &DerivativesVolatilityIndexDataRequest{Currency: currency.BTC, StartTime: testMarketDataRangeStart, EndTime: testMarketDataRangeEnd, Resolution: kline.OneHour}
}

// mockMarketDataVolatilityIndexCandles are the mock hourly volatility index candles of BTC
func mockMarketDataVolatilityIndexCandles() *DerivativesVolatilityIndexDataResponse {
	return &DerivativesVolatilityIndexDataResponse{
		Data: []DerivativesVolatilityIndexCandle{
			{Timestamp: testMarketDataMilli(1792281600000), Open: 37.93, High: 38.04, Low: 37.92, Close: 38.04},
			{Timestamp: testMarketDataMilli(1792285200000), Open: 38.04, High: 38.05, Low: 37.96, Close: 37.97},
			{Timestamp: testMarketDataMilli(1792288800000), Open: 37.97, High: 37.97, Low: 37.92, Close: 37.93},
			{Timestamp: testMarketDataMilli(1792292400000), Open: 37.93, High: 37.94, Low: 37.84, Close: 37.85},
			{Timestamp: testMarketDataMilli(1792296000000), Open: 37.85, High: 37.92, Low: 37.85, Close: 37.9},
		},
	}
}

func TestGlobalDerivativesGetVolatilityIndexData(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetVolatilityIndexData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetVolatilityIndexData must reject a nil request")
	_, err = e.GlobalDerivatives().GetVolatilityIndexData(t.Context(), &DerivativesVolatilityIndexDataRequest{StartTime: testMarketDataRangeStart, EndTime: testMarketDataRangeEnd, Resolution: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetVolatilityIndexData must require a currency")
	_, err = e.GlobalDerivatives().GetVolatilityIndexData(t.Context(), &DerivativesVolatilityIndexDataRequest{Currency: currency.BTC, Resolution: kline.OneHour})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetVolatilityIndexData must require a time range")
	_, err = e.GlobalDerivatives().GetVolatilityIndexData(t.Context(), &DerivativesVolatilityIndexDataRequest{Currency: currency.BTC, StartTime: testMarketDataRangeStart, EndTime: testMarketDataRangeEnd, Resolution: kline.FiveMin})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetVolatilityIndexData must refuse a resolution the gateway does not support")

	resp, err := e.GlobalDerivatives().GetVolatilityIndexData(t.Context(), testMarketDataVolatilityIndexDataRequest())
	require.NoError(t, err, "GetVolatilityIndexData must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataVolatilityIndexCandles(), resp, "GetVolatilityIndexData should decode every field")
	} else {
		assert.Len(t, resp.Data, 5, "GetVolatilityIndexData should return each hour's candle")
	}

	resp, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"data":[[1792296000000,37.85,37.85,37.85,37.85]],"continuation":1792294999000}}`).GlobalDerivatives().GetVolatilityIndexData(t.Context(), testMarketDataVolatilityIndexDataRequest())
	require.NoError(t, err, "GetVolatilityIndexData must not error for a page with more candles")
	exp := &DerivativesVolatilityIndexDataResponse{Data: []DerivativesVolatilityIndexCandle{{Timestamp: testMarketDataMilli(1792296000000), Open: 37.85, High: 37.85, Low: 37.85, Close: 37.85}}, Continuation: testMarketDataMilli(1792294999000)}
	assert.Equal(t, exp, resp, "GetVolatilityIndexData should decode the continuation of a page with more candles")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetVolatilityIndexData(t.Context(), testMarketDataVolatilityIndexDataRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetVolatilityIndexData should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetVolatilityIndexData(t.Context(), testMarketDataVolatilityIndexDataRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetVolatilityIndexData should return the request failure")
}

func TestDerivativesVolatilityIndexResolution(t *testing.T) {
	t.Parallel()
	for interval, exp := range map[kline.Interval]string{
		kline.ThousandMilliseconds: "1",
		kline.OneMin:               "60",
		kline.OneHour:              "3600",
		kline.TwelveHour:           "43200",
		kline.OneDay:               "1D",
	} {
		resolution, err := derivativesVolatilityIndexResolution(interval)
		require.NoErrorf(t, err, "derivativesVolatilityIndexResolution must not error for interval %s", interval)
		assert.Equalf(t, exp, resolution, "derivativesVolatilityIndexResolution should name interval %s", interval)
	}
	for _, interval := range []kline.Interval{kline.Raw, kline.FiveMin, kline.FourHour} {
		_, err := derivativesVolatilityIndexResolution(interval)
		assert.ErrorIsf(t, err, kline.ErrUnsupportedInterval, "derivativesVolatilityIndexResolution should refuse interval %s", interval)
	}
}

func TestDerivativesVolatilityIndexCandleUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var resp DerivativesVolatilityIndexCandle
	require.NoError(t, json.Unmarshal([]byte(`[1598019300000,0.210084879,0.212860821,0.210084879,0.212860821]`), &resp), "Unmarshal must not error")
	exp := DerivativesVolatilityIndexCandle{Timestamp: testMarketDataMilli(1598019300000), Open: 0.210084879, High: 0.212860821, Low: 0.210084879, Close: 0.212860821}
	assert.Equal(t, exp, resp, "UnmarshalJSON should decode the timestamp and prices")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[1598019300000,0.21,0.21,0.21]`), &resp), errInvalidTuple, "UnmarshalJSON should refuse a candle without a close")
}

// mockMarketDataTickers are the mock tickers of a perpetual, an option, Coinbase-routed spot and a future combo
func mockMarketDataTickers() map[string]*DerivativesTickerResponse {
	return map[string]*DerivativesTickerResponse{
		testMarketDataPerpetual: {
			InstrumentName: "BTC_USDC-PERPETUAL",
			Timestamp:      testMarketDataMilli(1792372182952),
			State:          "open",
			Stats: DerivativesTickerStats{
				High:           82481.3,
				Low:            80583.1,
				PriceChange:    1.1408,
				Volume:         35737.6298,
				VolumeUSD:      2911067265.17,
				VolumeNotional: 2911649760.3965893,
			},
			BestBidPrice:           82434,
			BestBidAmount:          2.2618,
			BestAskPrice:           82434.1,
			BestAskAmount:          4.5784,
			LastPrice:              82434,
			MarkPrice:              82433.33,
			IndexPrice:             82407.36,
			MinimumPrice:           81196.8,
			MaximumPrice:           83669.9,
			OpenInterest:           1778.015,
			EstimatedDeliveryPrice: 82407.36,
			SettlementPrice:        81598.81,
			CurrentFunding:         0.00006514,
			Funding8Hour:           0.00006353,
			InterestValue:          351558808.97178566,
		},
		mockMarketDataOption: {
			InstrumentName: "BTC-6NOV26-82000-C",
			Timestamp:      testMarketDataMilli(1792372183290),
			State:          "open",
			Stats: DerivativesTickerStats{
				High:        0.0345,
				Low:         0.0295,
				PriceChange: 9.5238,
				Volume:      19.4,
				VolumeUSD:   49513.82,
			},
			BestBidPrice:           0.0365,
			BestBidAmount:          11.7,
			BestAskPrice:           0.0375,
			BestAskAmount:          2.1,
			LastPrice:              0.0345,
			MarkPrice:              0.0371,
			IndexPrice:             82409.95,
			MinimumPrice:           0.0125,
			MaximumPrice:           0.071,
			OpenInterest:           379.4,
			EstimatedDeliveryPrice: 82409.95,
			SettlementPrice:        0.03195938,
			Greeks: &DerivativesGreeks{
				Delta: 0.55548,
				Gamma: 0.00006,
				Vega:  73.10109,
				Theta: -73.96825,
				Rho:   21.4675,
			},
			BidImpliedVolatility:  36.32,
			AskImpliedVolatility:  37.45,
			MarkImpliedVolatility: 37,
			UnderlyingIndex:       "BTC-6NOV26",
			UnderlyingPrice:       82668.93,
		},
		testMarketDataSpot: {
			InstrumentName: "BTC_USDC",
			Timestamp:      testMarketDataMilli(1792372184538),
			State:          "open",
			Stats: DerivativesTickerStats{
				High:        82454.5,
				Low:         80564.57,
				PriceChange: 1.0559,
				Volume:      2905.15932169,
			},
			BestBidPrice:  82419.98,
			BestBidAmount: 0.00008742,
			BestAskPrice:  82419.99,
			BestAskAmount: 0.09462947,
			LastPrice:     82430,
			MarkPrice:     82409.79,
			IndexPrice:    82409.79,
			MinimumPrice:  80761.59,
			MaximumPrice:  84057.99,
		},
		mockMarketDataFutureCombo: {
			InstrumentName: "BTC-FS-27NOV26_PERP",
			Timestamp:      testMarketDataMilli(1792372185221),
			State:          "open",
			Stats: DerivativesTickerStats{
				High:           510,
				Low:            499,
				VolumeUSD:      860910,
				VolumeNotional: 860910,
			},
			BestBidPrice:    502.5,
			BestBidAmount:   201910,
			BestAskPrice:    503,
			BestAskAmount:   150,
			LastPrice:       506.5,
			MarkPrice:       504.96,
			IndexPrice:      82409.79,
			MinimumPrice:    158.5,
			MaximumPrice:    851.5,
			SettlementPrice: 480.05,
			ComboState:      "inactive",
			ImpliedBid:      504.5,
			ImpliedAsk:      510,
		},
	}
}

func TestGlobalDerivativesGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetTicker(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetTicker must require an instrument name")

	for name, exp := range mockMarketDataTickers() {
		switch name {
		case mockMarketDataOption:
			name = marketDataOption(t)
		case mockMarketDataFutureCombo:
			name = marketDataFutureCombo(t)
		}
		resp, err := e.GlobalDerivatives().GetTicker(t.Context(), name)
		require.NoErrorf(t, err, "GetTicker must not error for %s", name)
		if mockTests {
			assert.Equalf(t, exp, resp, "GetTicker should decode every field of %s", name)
			continue
		}
		assert.Equalf(t, name, resp.InstrumentName, "GetTicker should return %s", name)
		assert.Equalf(t, exp.Greeks != nil, resp.Greeks != nil, "GetTicker should return greeks for %s only if it is an option", name)
		assert.Equalf(t, exp.ComboState != "", resp.ComboState != "", "GetTicker should return a combo state for %s only if it is a combo", name)
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetTicker(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTicker should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetTicker(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTicker should return the request failure")
}

// testMarketDataTickersByCurrencyRequest requests USDC futures' tickers with 3 levels of their books
func testMarketDataTickersByCurrencyRequest() *DerivativesTickersByCurrencyRequest {
	return &DerivativesTickersByCurrencyRequest{Currency: currency.USDC, Kinds: []string{DerivativesKindFuture}, Depth: 3}
}

// mockMarketDataTickersByCurrency are the mock tickers of USDC futures, filtered to testMarketDataPerpetual's and
// mockMarketDataDatedFuture's
func mockMarketDataTickersByCurrency() []DerivativesOrderBookResponse {
	return []DerivativesOrderBookResponse{
		{
			InstrumentName: "BTC_USDC-PERPETUAL",
			Timestamp:      testMarketDataMilli(1792372186228),
			State:          "open",
			Stats: DerivativesTickerStats{
				High:           82481.3,
				Low:            80583.1,
				PriceChange:    1.1456,
				Volume:         35742.9474,
				VolumeUSD:      2911505579.98,
				VolumeNotional: 2912088162.8894296,
			},
			BestBidPrice:           82438.7,
			BestBidAmount:          1.4026,
			BestAskPrice:           82438.8,
			BestAskAmount:          3.1091,
			LastPrice:              82438.7,
			MarkPrice:              82440.76,
			IndexPrice:             82414.71,
			MinimumPrice:           81204.1,
			MaximumPrice:           83677.4,
			OpenInterest:           1777.7,
			EstimatedDeliveryPrice: 82414.71,
			SettlementPrice:        81598.81,
			CurrentFunding:         0.00006608,
			Funding8Hour:           0.00006354,
			InterestValue:          351573113.82718563,
			ChangeID:               208575641260,
			Bids: []DerivativesOrderBookLevel{
				{Price: 82438.7, Amount: 1.4026},
				{Price: 82438.3, Amount: 1.0592},
				{Price: 82438.1, Amount: 0.6068},
			},
			Asks: []DerivativesOrderBookLevel{
				{Price: 82438.8, Amount: 3.1091},
				{Price: 82439.6, Amount: 1.2},
				{Price: 82439.7, Amount: 0.0019},
			},
		},
		{
			InstrumentName: "BTC_USDC-27NOV26",
			Timestamp:      testMarketDataMilli(1792372186109),
			State:          "open",
			Stats: DerivativesTickerStats{
				High:           82895,
				Low:            81207.5,
				PriceChange:    1.0769,
				Volume:         5.5899,
				VolumeUSD:      458727.17,
				VolumeNotional: 458818.93875,
			},
			BestBidPrice:           82915,
			BestBidAmount:          0.0151,
			BestAskPrice:           82917.5,
			BestAskAmount:          0.1112,
			LastPrice:              82827.5,
			MarkPrice:              82940.04,
			IndexPrice:             82414.71,
			MinimumPrice:           81695,
			MaximumPrice:           84185,
			OpenInterest:           19.9515,
			EstimatedDeliveryPrice: 82414.71,
			SettlementPrice:        82067.72,
			ChangeID:               208575636277,
			Bids: []DerivativesOrderBookLevel{
				{Price: 82915, Amount: 0.0151},
				{Price: 82910, Amount: 0.3137},
				{Price: 82907.5, Amount: 0.1332},
			},
			Asks: []DerivativesOrderBookLevel{
				{Price: 82917.5, Amount: 0.1112},
				{Price: 82927.5, Amount: 0.2913},
				{Price: 82932.5, Amount: 0.304},
			},
		},
	}
}

func TestGlobalDerivativesGetTickersByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetTickersByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTickersByCurrency must reject a nil request")
	_, err = e.GlobalDerivatives().GetTickersByCurrency(t.Context(), &DerivativesTickersByCurrencyRequest{Kinds: []string{DerivativesKindFuture}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTickersByCurrency must require a currency")
	_, err = e.GlobalDerivatives().GetTickersByCurrency(t.Context(), &DerivativesTickersByCurrencyRequest{Currency: currency.USDC, Kinds: []string{DerivativesKindFuture, DerivativesKindCombo}})
	require.ErrorIs(t, err, errDerivativesInvalidMarketDataKind, "GetTickersByCurrency must refuse a kind the gateway does not accept")
	_, err = e.GlobalDerivatives().GetTickersByCurrency(t.Context(), &DerivativesTickersByCurrencyRequest{Currency: currency.USDC, Kinds: []string{""}})
	require.ErrorIs(t, err, errDerivativesInvalidMarketDataKind, "GetTickersByCurrency must refuse an empty kind")
	_, err = e.GlobalDerivatives().GetTickersByCurrency(t.Context(), &DerivativesTickersByCurrencyRequest{Currency: currency.USDC, Depth: 5})
	require.ErrorIs(t, err, errDerivativesInvalidDepth, "GetTickersByCurrency must refuse a depth other than 1, 3 or 10")

	resp, err := e.GlobalDerivatives().GetTickersByCurrency(t.Context(), testMarketDataTickersByCurrencyRequest())
	require.NoError(t, err, "GetTickersByCurrency must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataTickersByCurrency(), resp, "GetTickersByCurrency should decode every field")
	} else {
		require.NotEmpty(t, resp, "GetTickersByCurrency must return USDC's futures")
		assert.Len(t, resp[0].Bids, 3, "GetTickersByCurrency should return each ticker's book at the depth requested")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, ok := readJSONRPC(t, r)
		if ok {
			assert.JSONEq(t, `{"currency":"BTC","kind":["future","option"]}`, string(req.Params), "GetTickersByCurrency should send its kinds as a list and leave the default depth out")
		}
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":[]}`)
	}))
	_, err = ex.GlobalDerivatives().GetTickersByCurrency(t.Context(), &DerivativesTickersByCurrencyRequest{Currency: currency.BTC, Kinds: []string{DerivativesKindFuture, DerivativesKindOption}})
	require.NoError(t, err, "GetTickersByCurrency must not error for several kinds")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetTickersByCurrency(t.Context(), testMarketDataTickersByCurrencyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTickersByCurrency should return the request failure")
}

func TestDerivativesMarketDataCurrency(t *testing.T) {
	t.Parallel()
	for _, ccy := range []currency.Code{currency.ANY, currency.NewCode("any"), currency.NewCode("Any")} {
		assert.Equalf(t, "any", derivativesMarketDataCurrency(ccy), "derivativesMarketDataCurrency should lower case %s", ccy)
	}
	for _, ccy := range []currency.Code{currency.NewCode(DerivativesCurrencyGrouped), currency.NewCode("GROUPED")} {
		assert.Equalf(t, DerivativesCurrencyGrouped, derivativesMarketDataCurrency(ccy), "derivativesMarketDataCurrency should lower case %s", ccy)
	}
	assert.Equal(t, "USDC", derivativesMarketDataCurrency(currency.USDC.Lower()), "derivativesMarketDataCurrency should upper case other currencies")
	assert.Empty(t, derivativesMarketDataCurrency(currency.EMPTYCODE), "derivativesMarketDataCurrency should leave an empty currency empty")
}

func TestValidateDerivativesMarketDataKind(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"", DerivativesKindFuture, DerivativesKindOption} {
		assert.NoErrorf(t, validateDerivativesMarketDataKind(kind, DerivativesKindFuture, DerivativesKindOption), "validateDerivativesMarketDataKind should accept kind %q", kind)
	}
	for _, kind := range []string{DerivativesKindSpot, "Future"} {
		assert.ErrorIsf(t, validateDerivativesMarketDataKind(kind, DerivativesKindFuture, DerivativesKindOption), errDerivativesInvalidMarketDataKind, "validateDerivativesMarketDataKind should refuse kind %q", kind)
	}
}

func TestDerivativesEstimatedDeliveryPriceUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   DerivativesEstimatedDeliveryPrice
		err   error
	}{
		{input: `82394.38`, exp: 82394.38},
		{input: `"expired"`},
		{input: `null`},
		{input: `"soon"`, err: errDerivativesInvalidEstimatedDeliveryPrice},
	} {
		var price DerivativesEstimatedDeliveryPrice
		err := json.Unmarshal([]byte(tc.input), &price)
		assert.ErrorIsf(t, err, tc.err, "UnmarshalJSON should return the expected error for %s", tc.input)
		assert.Equalf(t, tc.exp, price, "UnmarshalJSON should decode %s", tc.input)
	}
}

func TestGetTickerAndOrderBookDeliveredInstrument(t *testing.T) {
	t.Parallel()
	// A delivered future's ticker and book as the gateway sent them on 20 October 2026, less the envelopes' timings, its
	// estimated delivery price then "expired"
	ex, _ := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		switch req.Method {
		case "public/ticker":
			return `{"jsonrpc":"2.0","id":1,"result":{"timestamp":1792483205250,"state":"delivered","stats":{"high":82575.0,"low":81172.5,"price_change":-1.3563,"volume":22.90362211,"volume_usd":1870260.0,"volume_notional":1870260.0},"index_price":81441.35,"instrument_name":"BTC-20OCT26","last_price":81455.0,"min_price":80237.5,"max_price":82682.5,"open_interest":822660,"mark_price":81459.59,"estimated_delivery_price":"expired","delivery_price":81459.59,"best_ask_price":0.0,"best_bid_price":0.0,"best_ask_amount":0.0,"best_bid_amount":0.0},"testnet":false}`
		case "public/get_order_book":
			return `{"jsonrpc":"2.0","id":1,"result":{"timestamp":1792483205250,"state":"delivered","stats":{"high":82575.0,"low":81172.5,"price_change":-1.3563,"volume":22.90362211,"volume_usd":1870260.0,"volume_notional":1870260.0},"change_id":179190796196,"index_price":81441.35,"instrument_name":"BTC-20OCT26","bids":[],"asks":[],"last_price":81455.0,"min_price":80237.5,"max_price":82682.5,"open_interest":822660,"mark_price":81459.59,"estimated_delivery_price":"expired","delivery_price":81459.59,"best_ask_price":0.0,"best_bid_price":0.0,"best_ask_amount":0.0,"best_bid_amount":0.0},"testnet":false}`
		}
		assert.Failf(t, "the test should only read the ticker and the book", "sent %s", req.Method)
		return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
	})
	exp := DerivativesTickerResponse{
		InstrumentName: "BTC-20OCT26",
		Timestamp:      testMarketDataMilli(1792483205250),
		State:          "delivered",
		Stats: DerivativesTickerStats{
			High:           82575,
			Low:            81172.5,
			PriceChange:    -1.3563,
			Volume:         22.90362211,
			VolumeUSD:      1870260,
			VolumeNotional: 1870260,
		},
		LastPrice:     81455,
		MarkPrice:     81459.59,
		IndexPrice:    81441.35,
		MinimumPrice:  80237.5,
		MaximumPrice:  82682.5,
		OpenInterest:  822660,
		DeliveryPrice: 81459.59,
	}
	ticker, err := ex.GlobalDerivatives().GetTicker(t.Context(), "BTC-20OCT26")
	require.NoError(t, err, "GetTicker must decode a delivered instrument's ticker")
	assert.Equal(t, &exp, ticker, "GetTicker should decode a delivered instrument's ticker, with no estimated delivery price")
	book, err := ex.GlobalDerivatives().GetOrderBook(t.Context(), &DerivativesOrderBookRequest{InstrumentName: "BTC-20OCT26"})
	require.NoError(t, err, "GetOrderBook must decode a delivered instrument's book")
	assert.Equal(t, &DerivativesOrderBookResponse{DerivativesTickerResponse: exp, ChangeID: 179190796196, Bids: []DerivativesOrderBookLevel{}, Asks: []DerivativesOrderBookLevel{}}, book, "GetOrderBook should decode a delivered instrument's empty book")
}
