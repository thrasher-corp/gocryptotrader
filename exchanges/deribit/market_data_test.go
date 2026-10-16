package deribit

import (
	"context"
	"errors"
	"io"
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
	// marketDataPerpetual is a perpetual and marketDataPerpetualID its instrument ID, which mock and live tests share
	marketDataPerpetual   = "BTC-PERPETUAL"
	marketDataPerpetualID = 210838
	// marketDataSpot is a spot pair, and marketDataCoinbaseRoutedSpot one whose orders Coinbase Exchange matches
	marketDataSpot               = "BTC_USDC"
	marketDataCoinbaseRoutedSpot = "AAVE_USDC"
	marketDataIndexName          = "btc_usd"
	marketDataBasketIndexName    = "COIN50"
	// marketDataUnknownInstrument is an instrument Deribit has never listed
	marketDataUnknownInstrument = "BTC-01JAN20"
	// mockMarketDataOption and mockMarketDataFutureCombo are the option and future combo the mock responses were
	// recorded for; live tests look up listed ones instead
	mockMarketDataOption      = "BTC-27NOV26-90000-C"
	mockMarketDataFutureCombo = "BTC-FS-27NOV26_PERP"
	// marketDataRangeStart and marketDataRangeEnd bound the 4 hours from 2026-10-15T00:00:00Z, in Unix milliseconds
	marketDataRangeStart = 1792022400000
	marketDataRangeEnd   = 1792036800000
	// mockMarkPriceStart and mockMarkPriceEnd bound the mock mark price history; Deribit keeps about 5 minutes of it,
	// so live tests ask for the last minute instead
	mockMarkPriceStart = 1792109784000
	mockMarkPriceEnd   = 1792109814000
	// testAPRBefore is the epoch day the APR history tests page back from
	testAPRBefore = 20700
	// mockLastTradesRangeStart and mockLastTradesRangeEnd bound the hour from 2026-10-15T23:00:00Z, and
	// mockLastTradesStartID, mockLastTradesEndID, mockLastTradesStartSequence and mockLastTradesEndSequence its first
	// BTC-PERPETUAL trades, which the mock trades were recorded for; Deribit only serves the last 24 hours of trades,
	// so live tests ask for recent ones
	mockLastTradesRangeStart    = 1792105200000
	mockLastTradesRangeEnd      = 1792108800000
	mockLastTradesStartID       = "452869730"
	mockLastTradesEndID         = "452869734"
	mockLastTradesStartSequence = 306491081
	mockLastTradesEndSequence   = 306491083
)

var errNoListedMarketDataInstrument = errors.New("no listed instrument found for the live tests")

// marketDataMilli converts a Unix millisecond timestamp, as Deribit sends most times, into a types.Time
func marketDataMilli(timestamp int64) types.Time {
	return types.Time(time.UnixMilli(timestamp))
}

// marketDataNano converts a Unix nanosecond timestamp, as Deribit sends Starbase times, into a types.Time
func marketDataNano(timestamp int64) types.Time {
	return types.Time(time.Unix(0, timestamp))
}

// listedMarketDataInstrument returns the first open instrument of a kind settled in BTC that expires at least a day
// from now, which live tests use in place of the mock instruments
func listedMarketDataInstrument(kind string) (string, error) {
	instruments, err := e.GetInstruments(context.Background(), &InstrumentsRequest{Currency: currency.BTC, Kind: kind})
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
	listedMarketDataOption      = sync.OnceValues(func() (string, error) { return listedMarketDataInstrument(KindOption) })
	listedMarketDataFutureCombo = sync.OnceValues(func() (string, error) { return listedMarketDataInstrument(KindFutureCombo) })
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

// testAPRHistoryRequest pages back from testAPRBefore
func testAPRHistoryRequest() *APRHistoryRequest {
	return &APRHistoryRequest{Currency: currency.STETH, Limit: 5, Before: testAPRBefore}
}

// testAPRHistory is the mock APR history
func testAPRHistory() *APRHistoryResponse {
	return &APRHistoryResponse{
		Data: []APRRecord{
			{Day: 20699, APR: 2.191},
			{Day: 20698, APR: 2.182},
			{Day: 20697, APR: 2.212},
			{Day: 20696, APR: 2.265},
			{Day: 20695, APR: 2.206},
		},
		Continuation: 20695,
	}
}

func TestGetAPRHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAPRHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAPRHistory must reject a nil request")
	_, err = e.GetAPRHistory(t.Context(), &APRHistoryRequest{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAPRHistory must reject an empty currency")

	result, err := e.GetAPRHistory(t.Context(), testAPRHistoryRequest())
	require.NoError(t, err, "GetAPRHistory must not error")
	if mockTests {
		assert.Equal(t, testAPRHistory(), result, "GetAPRHistory should decode every field")
	} else {
		assert.Len(t, result.Data, 5, "GetAPRHistory should return the days requested")
	}

	result, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"data":[]}}`).GetAPRHistory(t.Context(), testAPRHistoryRequest())
	require.NoError(t, err, "GetAPRHistory must not error for the last page")
	assert.Equal(t, &APRHistoryResponse{Data: []APRRecord{}}, result, "GetAPRHistory should decode a last page without a continuation")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetAPRHistory(t.Context(), testAPRHistoryRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAPRHistory should reject a null result")
	_, err = newUnavailableServerExchange(t).GetAPRHistory(t.Context(), testAPRHistoryRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAPRHistory should return the request failure")
}

// testBasketIndexComposition is the mock basket index composition; the recorder blanks every name field
func testBasketIndexComposition() *BasketIndexCompositionResponse {
	return &BasketIndexCompositionResponse{
		ProductID:          "COIN50",
		Divisor:            3458940467.066527,
		Timestamp:          time.Date(2026, 10, 15, 14, 0, 2, 0, time.UTC),
		InceptionTimestamp: time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC),
		LastRebalance:      time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC),
		Constituents: []BasketIndexConstituent{
			{
				Symbol:         "ADA-USD",
				Rank:           1,
				CapFactor:      0.999999998164512,
				Amount:         36699593933,
				MarketCap:      8027001504.672193,
				IndexMarketCap: 8027001489.938728,
				Weight:         0.007561883224394102,
				RunningWeight:  0.007561883224394102,
			},
			{
				Symbol:         "APT-USD",
				Rank:           2,
				CapFactor:      0.999999997932116,
				Amount:         857933031,
				MarketCap:      653434130.3248502,
				IndexMarketCap: 653434128.9736242,
				Weight:         0.0006155714041321224,
				RunningWeight:  0.008177454628526225,
			},
			{
				Symbol:         "ARB-USD",
				Rank:           3,
				CapFactor:      0.999999998017801,
				Amount:         6678075931,
				MarketCap:      1155411848.0937486,
				IndexMarketCap: 1155411845.8034925,
				Weight:         0.0010884624183761485,
				RunningWeight:  0.009265917046902373,
			},
			{
				Symbol:         "BCH-USD",
				Rank:           4,
				CapFactor:      0.999999998190321,
				Amount:         20079868,
				MarketCap:      5268939726.865828,
				IndexMarketCap: 5268939717.330738,
				Weight:         0.004963635164235063,
				RunningWeight:  0.014229552211137437,
			},
			{
				Symbol:         "BTC-USD",
				Rank:           5,
				CapFactor:      0.337583116548643,
				Amount:         20074765,
				MarketCap:      1607981823736.3523,
				IndexMarketCap: 542827515410.4885,
				Weight:         0.5113738034890333,
				RunningWeight:  0.5256033557001707,
			},
		},
	}
}

func TestGetBasketIndexComposition(t *testing.T) {
	t.Parallel()
	_, err := e.GetBasketIndexComposition(t.Context(), "")
	require.ErrorIs(t, err, errIndexNameRequired, "GetBasketIndexComposition must reject an empty index name")

	result, err := e.GetBasketIndexComposition(t.Context(), marketDataBasketIndexName)
	require.NoError(t, err, "GetBasketIndexComposition must not error")
	if mockTests {
		assert.Equal(t, testBasketIndexComposition(), result, "GetBasketIndexComposition should decode every field")
	} else {
		assert.Equal(t, marketDataBasketIndexName, result.ProductID, "GetBasketIndexComposition should return the index requested")
		assert.NotEmpty(t, result.Constituents, "GetBasketIndexComposition should return the index's constituents")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetBasketIndexComposition(t.Context(), marketDataBasketIndexName)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetBasketIndexComposition should reject a null result")
	_, err = newUnavailableServerExchange(t).GetBasketIndexComposition(t.Context(), marketDataBasketIndexName)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBasketIndexComposition should return the request failure")
}

// testOptionBookSummaries are the mock book summaries of BTC options
func testOptionBookSummaries() []BookSummary {
	return []BookSummary{
		{
			InstrumentName:         "BTC-25DEC26-104000-C",
			BaseCurrency:           currency.BTC,
			QuoteCurrency:          currency.BTC,
			CreationTimestamp:      marketDataMilli(1792109780877),
			BidPrice:               0.0075,
			AskPrice:               0.0085,
			MidPrice:               0.008,
			MarkPrice:              0.00797093,
			LastPrice:              0.008,
			High:                   0.0085,
			Low:                    0.008,
			PriceChange:            -5.8824,
			Volume:                 2.8,
			VolumeUSD:              1873.51,
			OpenInterest:           177.6,
			EstimatedDeliveryPrice: 82258.35,
			MarkImpliedVolatility:  38.27,
			UnderlyingIndex:        "BTC-25DEC26",
			UnderlyingPrice:        83154.18,
		},
		{
			InstrumentName:         "BTC-24SEP27-60000-C",
			BaseCurrency:           currency.BTC,
			QuoteCurrency:          currency.BTC,
			CreationTimestamp:      marketDataMilli(1792109780877),
			BidPrice:               0.3415,
			AskPrice:               0.3445,
			MidPrice:               0.343,
			MarkPrice:              0.34290887,
			EstimatedDeliveryPrice: 82258.35,
			MarkImpliedVolatility:  43.52,
			UnderlyingIndex:        "BTC-24SEP27",
			UnderlyingPrice:        86424.43,
		},
		{
			InstrumentName:         "BTC-27NOV26-70000-C",
			BaseCurrency:           currency.BTC,
			QuoteCurrency:          currency.BTC,
			CreationTimestamp:      marketDataMilli(1792109780877),
			BidPrice:               0.1615,
			AskPrice:               0.1655,
			MidPrice:               0.1635,
			MarkPrice:              0.1634272,
			LastPrice:              0.184,
			OpenInterest:           3.3,
			EstimatedDeliveryPrice: 82258.35,
			MarkImpliedVolatility:  43.54,
			UnderlyingIndex:        "BTC-27NOV26",
			UnderlyingPrice:        82809.86,
		},
		{
			InstrumentName:         "BTC-16OCT26-88000-P",
			BaseCurrency:           currency.BTC,
			QuoteCurrency:          currency.BTC,
			CreationTimestamp:      marketDataMilli(1792109780877),
			BidPrice:               0.0675,
			AskPrice:               0.072,
			MidPrice:               0.06975,
			MarkPrice:              0.069781,
			LastPrice:              0.067,
			OpenInterest:           17.3,
			EstimatedDeliveryPrice: 82258.35,
			MarkImpliedVolatility:  63.37,
			UnderlyingIndex:        "BTC-16OCT26",
			UnderlyingPrice:        82267.17,
		},
		{
			InstrumentName:         "BTC-19OCT26-78000-C",
			BaseCurrency:           currency.BTC,
			QuoteCurrency:          currency.BTC,
			CreationTimestamp:      marketDataMilli(1792109780877),
			BidPrice:               0.051,
			AskPrice:               0.0555,
			MidPrice:               0.05325,
			MarkPrice:              0.05327254,
			EstimatedDeliveryPrice: 82258.35,
			MarkImpliedVolatility:  37.43,
			UnderlyingIndex:        "BTC-19OCT26",
			UnderlyingPrice:        82301.33,
		},
	}
}

func TestGetBookSummaryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetBookSummaryByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBookSummaryByCurrency must reject a nil request")
	_, err = e.GetBookSummaryByCurrency(t.Context(), &BookSummaryByCurrencyRequest{Kind: KindOption})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBookSummaryByCurrency must reject an empty currency")
	_, err = e.GetBookSummaryByCurrency(t.Context(), &BookSummaryByCurrencyRequest{Currency: currency.BTC, Kind: KindCombo})
	require.ErrorIs(t, err, errInvalidKind, "GetBookSummaryByCurrency must reject a kind it does not accept")

	result, err := e.GetBookSummaryByCurrency(t.Context(), &BookSummaryByCurrencyRequest{Currency: currency.BTC, Kind: KindOption})
	require.NoError(t, err, "GetBookSummaryByCurrency must not error")
	if mockTests {
		assert.Equal(t, testOptionBookSummaries(), result, "GetBookSummaryByCurrency should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetBookSummaryByCurrency should return BTC's options")
	}

	_, err = newUnavailableServerExchange(t).GetBookSummaryByCurrency(t.Context(), &BookSummaryByCurrencyRequest{Currency: currency.BTC})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBookSummaryByCurrency should return the request failure")
}

// testPerpetualBookSummary is the mock book summary of marketDataPerpetual
func testPerpetualBookSummary() []BookSummary {
	return []BookSummary{
		{
			InstrumentName:         "BTC-PERPETUAL",
			BaseCurrency:           currency.BTC,
			QuoteCurrency:          currency.USD,
			CreationTimestamp:      marketDataMilli(1792109782275),
			BidPrice:               82290,
			AskPrice:               82290.5,
			MidPrice:               82290.25,
			MarkPrice:              82277.35,
			LastPrice:              82288.5,
			High:                   83000,
			Low:                    79709.5,
			PriceChange:            0.6483,
			Volume:                 26595.77338167,
			VolumeUSD:              2161658900,
			VolumeNotional:         2161658900,
			OpenInterest:           862167020,
			EstimatedDeliveryPrice: 82262.67,
			Funding8Hour:           4.146e-05,
		},
	}
}

func TestGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetBookSummaryByInstrument(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetBookSummaryByInstrument must reject an empty instrument name")

	result, err := e.GetBookSummaryByInstrument(t.Context(), marketDataPerpetual)
	require.NoError(t, err, "GetBookSummaryByInstrument must not error")
	if mockTests {
		assert.Equal(t, testPerpetualBookSummary(), result, "GetBookSummaryByInstrument should decode every field")
	} else {
		require.Len(t, result, 1, "GetBookSummaryByInstrument must return one summary")
		assert.Equal(t, marketDataPerpetual, result[0].InstrumentName, "GetBookSummaryByInstrument should return the instrument requested")
	}

	_, err = newUnavailableServerExchange(t).GetBookSummaryByInstrument(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBookSummaryByInstrument should return the request failure")
}

func TestGetContractSize(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractSize(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetContractSize must reject an empty instrument name")

	result, err := e.GetContractSize(t.Context(), marketDataPerpetual)
	require.NoError(t, err, "GetContractSize must not error")
	assert.Equal(t, &ContractSizeResponse{ContractSize: 10}, result, "GetContractSize should decode BTC-PERPETUAL's contract size of 10 USD")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetContractSize(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetContractSize should reject a null result")
	_, err = newUnavailableServerExchange(t).GetContractSize(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetContractSize should return the request failure")
}

// testPublicCurrencies are the mock currencies
func testPublicCurrencies() []CurrencyDetail {
	return []CurrencyDetail{
		{
			Currency:                   currency.NewCode("ZORA"),
			CurrencyLong:               "Zora",
			CurrencyUUID:               "0c26413b-4404-407d-a8fc-e25d1dd8608e",
			CoinType:                   "ZORA",
			Decimals:                   6,
			OnchainOperationsPrecision: 6,
			WithdrawalPriorities:       []CurrencyWithdrawalPriority{},
			NetworkCurrency:            currency.NewCode("ZORA"),
			CoinbaseNetworks: []CurrencyCoinbaseNetwork{
				{
					ResourceName: "networks/base-mainnet",
					DisplayName:  "Base",
				},
			},
		},
		{
			Currency:                   currency.NewCode("XRP"),
			CurrencyLong:               "XRP",
			CurrencyUUID:               "e17a44c8-6ea1-564f-a02c-2a9ca1d8eec4",
			CoinType:                   "XRP",
			Decimals:                   6,
			OnchainOperationsPrecision: 6,
			MinConfirmations:           1,
			WithdrawalFee:              0.0001,
			MinWithdrawalFee:           0.0001,
			WithdrawalPriorities:       []CurrencyWithdrawalPriority{},
			NetworkFee:                 1.5e-05,
			NetworkCurrency:            currency.NewCode("XRP"),
			InCrossCollateralPool:      true,
			CoinbaseNetworks: []CurrencyCoinbaseNetwork{
				{
					ResourceName: "networks/ripple-mainnet",
					DisplayName:  "Ripple",
				},
			},
		},
		{
			Currency:                   currency.NewCode("XLM"),
			CurrencyLong:               "Stellar",
			CurrencyUUID:               "13b83335-5ede-595b-821e-5bcdfa80560f",
			CoinType:                   "XLM",
			Decimals:                   6,
			OnchainOperationsPrecision: 6,
			WithdrawalPriorities:       []CurrencyWithdrawalPriority{},
			NetworkCurrency:            currency.NewCode("XLM"),
			CoinbaseNetworks: []CurrencyCoinbaseNetwork{
				{
					ResourceName: "networks/stellar-mainnet",
					DisplayName:  "Stellar",
				},
			},
		},
		{
			Currency:                   currency.NewCode("VET"),
			CurrencyLong:               "VeChain",
			CurrencyUUID:               "3212e00d-e1c8-5776-89d7-50551b5c42d9",
			CoinType:                   "VET",
			Decimals:                   6,
			OnchainOperationsPrecision: 6,
			WithdrawalPriorities:       []CurrencyWithdrawalPriority{},
			NetworkCurrency:            currency.NewCode("VET"),
			CoinbaseNetworks: []CurrencyCoinbaseNetwork{
				{
					ResourceName: "networks/vechain-mainnet",
					DisplayName:  "VeChainThor",
				},
			},
		},
		{
			Currency:                   currency.NewCode("USYC"),
			CurrencyLong:               "USYC",
			CurrencyUUID:               "4c559e06-44d0-411c-bb36-1a4812df4510",
			CoinType:                   "USYC",
			Decimals:                   6,
			OnchainOperationsPrecision: 6,
			MinConfirmations:           1,
			WithdrawalFee:              0.007251,
			MinWithdrawalFee:           0.0001,
			WithdrawalPriorities:       []CurrencyWithdrawalPriority{},
			NetworkFee:                 3.3849334000000004e-06,
			NetworkCurrency:            currency.ETH,
			InCrossCollateralPool:      true,
		},
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetCurrencies(t.Context())
	require.NoError(t, err, "GetCurrencies must not error")
	if mockTests {
		assert.Equal(t, testPublicCurrencies(), result, "GetCurrencies should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetCurrencies should return Deribit's currencies")
	}

	result, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":[{"currency":"BTC","withdrawal_priorities":[{"value":0.15,"name":"very_low"},{"value":1.5,"name":"very_high"}]}]}`).GetCurrencies(t.Context())
	require.NoError(t, err, "GetCurrencies must not error for withdrawal priorities")
	exp := []CurrencyDetail{{Currency: currency.BTC, WithdrawalPriorities: []CurrencyWithdrawalPriority{{Name: "very_low", Value: 0.15}, {Name: "very_high", Value: 1.5}}}}
	assert.Equal(t, exp, result, "GetCurrencies should decode Deribit's documented withdrawal priorities, which the recording leaves empty")
	_, err = newUnavailableServerExchange(t).GetCurrencies(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCurrencies should return the request failure")
}

// testDeliveryPricesRequest skips the latest delivery price
func testDeliveryPricesRequest() *DeliveryPricesRequest {
	return &DeliveryPricesRequest{IndexName: marketDataIndexName, Offset: 1, Count: 5}
}

// testDeliveryPrices are the mock delivery prices
func testDeliveryPrices() *DeliveryPricesResponse {
	return &DeliveryPricesResponse{
		Data: []DeliveryPrice{
			{Date: "2026-10-14", DeliveryPrice: 83396.69},
			{Date: "2026-10-13", DeliveryPrice: 83826.89},
			{Date: "2026-10-12", DeliveryPrice: 83911.12},
			{Date: "2026-10-11", DeliveryPrice: 82952.11},
			{Date: "2026-10-10", DeliveryPrice: 82674.28},
		},
		RecordsTotal: 2639,
	}
}

func TestGetDeliveryPrices(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryPrices(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDeliveryPrices must reject a nil request")
	_, err = e.GetDeliveryPrices(t.Context(), &DeliveryPricesRequest{Count: 5})
	require.ErrorIs(t, err, errIndexNameRequired, "GetDeliveryPrices must reject an empty index name")

	result, err := e.GetDeliveryPrices(t.Context(), testDeliveryPricesRequest())
	require.NoError(t, err, "GetDeliveryPrices must not error")
	if mockTests {
		assert.Equal(t, testDeliveryPrices(), result, "GetDeliveryPrices should decode every field")
	} else {
		assert.Len(t, result.Data, 5, "GetDeliveryPrices should return the prices requested")
		assert.NotZero(t, result.RecordsTotal, "GetDeliveryPrices should return the number of prices available")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetDeliveryPrices(t.Context(), testDeliveryPricesRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetDeliveryPrices should reject a null result")
	_, err = newUnavailableServerExchange(t).GetDeliveryPrices(t.Context(), testDeliveryPricesRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDeliveryPrices should return the request failure")
}

func TestGetExpirations(t *testing.T) {
	t.Parallel()
	_, err := e.GetExpirations(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExpirations must reject a nil request")
	_, err = e.GetExpirations(t.Context(), &ExpirationsRequest{Kind: KindSpot})
	require.ErrorIs(t, err, errInvalidKind, "GetExpirations must reject a kind it does not accept")

	result, err := e.GetExpirations(t.Context(), &ExpirationsRequest{Currency: currency.ANY, Kind: KindFuture})
	require.NoError(t, err, "GetExpirations must not error for every currency")
	if mockTests {
		exp := &ExpirationsResponse{Future: []string{"16OCT26", "17OCT26", "18OCT26", "19OCT26", "23OCT26"}}
		assert.Equal(t, exp, result, "GetExpirations should decode the labels merged across currencies")
	} else {
		assert.Contains(t, result.Future, "PERPETUAL", "GetExpirations should return the perpetuals' label")
	}

	result, err = e.GetExpirations(t.Context(), &ExpirationsRequest{Currency: currency.BTC, Kind: KindOption})
	require.NoError(t, err, "GetExpirations must not error for a currency")
	if mockTests {
		exp := &ExpirationsResponse{ByCurrency: map[string]Expirations{"btc": {Option: []string{"16OCT26", "17OCT26", "18OCT26", "19OCT26", "23OCT26"}}}}
		assert.Equal(t, exp, result, "GetExpirations should decode the labels keyed by currency")
	} else {
		assert.NotEmpty(t, result.ByCurrency["btc"].Option, "GetExpirations should return BTC's option labels")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetExpirations(t.Context(), &ExpirationsRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetExpirations should reject a null result")
	_, err = newUnavailableServerExchange(t).GetExpirations(t.Context(), &ExpirationsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetExpirations should return the request failure")
}

func TestExpirationsResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input string
		exp   ExpirationsResponse
	}{
		{
			name:  "merged",
			input: `{"future":["21SEP24","PERPETUAL"],"option":["21SEP24"]}`,
			exp:   ExpirationsResponse{Future: []string{"21SEP24", "PERPETUAL"}, Option: []string{"21SEP24"}},
		},
		{
			name:  "keyed",
			input: `{"btc":{"future":["PERPETUAL"]},"eth_usdc":{"option":[]}}`,
			exp:   ExpirationsResponse{ByCurrency: map[string]Expirations{"btc": {Future: []string{"PERPETUAL"}}, "eth_usdc": {Option: []string{}}}},
		},
		{name: "empty", input: `{}`, exp: ExpirationsResponse{ByCurrency: map[string]Expirations{}}},
	} {
		var result ExpirationsResponse
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &result), "Unmarshal must not error for the %s labels", tc.name)
		assert.Equalf(t, tc.exp, result, "UnmarshalJSON should decode the %s labels", tc.name)
	}
	var result ExpirationsResponse
	assert.Error(t, json.Unmarshal([]byte(`["PERPETUAL"]`), &result), "UnmarshalJSON should reject labels that are not an object")
	assert.Error(t, json.Unmarshal([]byte(`{"btc":"PERPETUAL"}`), &result), "UnmarshalJSON should reject a currency's labels that are not an object")
}

// testFundingChartData is the mock funding chart of marketDataPerpetual
func testFundingChartData() *FundingChartDataResponse {
	return &FundingChartDataResponse{
		Interest8Hour: 4.1548439151455586e-05,
		Data: []FundingChartPoint{
			{
				Timestamp:     marketDataMilli(1792080960000),
				IndexPrice:    82319.2,
				Interest8Hour: 4.2762806246973e-05,
			},
			{
				Timestamp:     marketDataMilli(1792081020000),
				IndexPrice:    82284.04,
				Interest8Hour: 4.616922066551297e-05,
			},
			{
				Timestamp:     marketDataMilli(1792081080000),
				IndexPrice:    82222.94,
				Interest8Hour: 3.3619145703041406e-05,
			},
			{
				Timestamp:     marketDataMilli(1792081140000),
				IndexPrice:    82226.52,
				Interest8Hour: 5.282200924949977e-05,
			},
			{
				Timestamp:     marketDataMilli(1792081200000),
				IndexPrice:    82274.61,
				Interest8Hour: 8.436803893685746e-05,
			},
		},
	}
}

func TestGetFundingChartData(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingChartData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingChartData must reject a nil request")
	_, err = e.GetFundingChartData(t.Context(), &FundingChartDataRequest{Length: FundingChartLength8Hour})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetFundingChartData must reject an empty instrument name")
	_, err = e.GetFundingChartData(t.Context(), &FundingChartDataRequest{InstrumentName: marketDataPerpetual, Length: "2h"})
	require.ErrorIs(t, err, errInvalidFundingChartLength, "GetFundingChartData must reject a length Deribit does not chart")

	arg := &FundingChartDataRequest{InstrumentName: marketDataPerpetual, Length: FundingChartLength8Hour}
	result, err := e.GetFundingChartData(t.Context(), arg)
	require.NoError(t, err, "GetFundingChartData must not error")
	if mockTests {
		assert.Equal(t, testFundingChartData(), result, "GetFundingChartData should decode every field")
	} else {
		assert.NotEmpty(t, result.Data, "GetFundingChartData should return the funding chart")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetFundingChartData(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFundingChartData should reject a null result")
	_, err = newUnavailableServerExchange(t).GetFundingChartData(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingChartData should return the request failure")
}

// testFundingRangeRequest covers marketDataPerpetual's funding over the 4 hours from 2026-10-15T00:00:00Z
func testFundingRangeRequest() *InstrumentTimeRangeRequest {
	return &InstrumentTimeRangeRequest{InstrumentName: marketDataPerpetual, StartTime: time.UnixMilli(marketDataRangeStart), EndTime: time.UnixMilli(marketDataRangeEnd)}
}

// testFundingRateHistory is the mock hourly funding of marketDataPerpetual, which ends each hour after the range's
// start
func testFundingRateHistory() []FundingRateHistory {
	return []FundingRateHistory{
		{
			Timestamp:          marketDataMilli(1792026000000),
			IndexPrice:         81556.85,
			PreviousIndexPrice: 81576.93,
			Interest1Hour:      1.3808122211552417e-05,
			Interest8Hour:      6.311465614099483e-05,
		},
		{
			Timestamp:          marketDataMilli(1792029600000),
			IndexPrice:         81539.22,
			PreviousIndexPrice: 81556.85,
			Interest1Hour:      1.623641763878923e-05,
			Interest8Hour:      7.03250178202503e-05,
		},
		{
			Timestamp:          marketDataMilli(1792033200000),
			IndexPrice:         81416.4,
			PreviousIndexPrice: 81539.22,
			Interest1Hour:      8.76204974250811e-06,
			Interest8Hour:      7.342520538245485e-05,
		},
		{
			Timestamp:          marketDataMilli(1792036800000),
			IndexPrice:         80827.82,
			PreviousIndexPrice: 81416.4,
			Interest1Hour:      1.105698059002636e-05,
			Interest8Hour:      7.917005619894089e-05,
		},
	}
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingRateHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingRateHistory must reject a nil request")
	_, err = e.GetFundingRateHistory(t.Context(), &InstrumentTimeRangeRequest{InstrumentName: marketDataPerpetual})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetFundingRateHistory must reject an unset time range")

	result, err := e.GetFundingRateHistory(t.Context(), testFundingRangeRequest())
	require.NoError(t, err, "GetFundingRateHistory must not error")
	if mockTests {
		assert.Equal(t, testFundingRateHistory(), result, "GetFundingRateHistory should decode every field")
	} else {
		assert.Len(t, result, 4, "GetFundingRateHistory should return an hour for each hour ending in the range")
	}

	_, err = newUnavailableServerExchange(t).GetFundingRateHistory(t.Context(), testFundingRangeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingRateHistory should return the request failure")
}

func TestGetFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingRateValue(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingRateValue must reject a nil request")
	_, err = e.GetFundingRateValue(t.Context(), &InstrumentTimeRangeRequest{StartTime: time.UnixMilli(marketDataRangeStart), EndTime: time.UnixMilli(marketDataRangeEnd)})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetFundingRateValue must reject an empty instrument name")

	result, err := e.GetFundingRateValue(t.Context(), testFundingRangeRequest())
	require.NoError(t, err, "GetFundingRateValue must not error")
	if mockTests {
		assert.Equal(t, 4.989342119697176e-05, result, "GetFundingRateValue should decode the funding rate")
	} else {
		assert.NotZero(t, result, "GetFundingRateValue should return the funding rate")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetFundingRateValue(t.Context(), testFundingRangeRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFundingRateValue should reject a null result")
	_, err = newUnavailableServerExchange(t).GetFundingRateValue(t.Context(), testFundingRangeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingRateValue should return the request failure")
}

func TestInstrumentTimeRangeRequestParams(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(marketDataRangeStart), time.UnixMilli(marketDataRangeEnd)
	for _, tc := range []struct {
		name string
		arg  *InstrumentTimeRangeRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty instrument name", arg: &InstrumentTimeRangeRequest{StartTime: start, EndTime: end}, err: errInstrumentNameRequired},
		{name: "unset start", arg: &InstrumentTimeRangeRequest{InstrumentName: marketDataPerpetual, EndTime: end}, err: common.ErrDateUnset},
		{name: "unset end", arg: &InstrumentTimeRangeRequest{InstrumentName: marketDataPerpetual, StartTime: start}, err: common.ErrDateUnset},
		{name: "start after end", arg: &InstrumentTimeRangeRequest{InstrumentName: marketDataPerpetual, StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		_, err := tc.arg.params()
		assert.ErrorIsf(t, err, tc.err, "params should reject a request with an %s", tc.name)
	}

	arg := &InstrumentTimeRangeRequest{InstrumentName: marketDataPerpetual, StartTime: start, EndTime: end}
	params, err := arg.params()
	require.NoError(t, err, "params must not error")
	encoded, err := json.Marshal(params)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"instrument_name":"BTC-PERPETUAL","start_timestamp":1792022400000,"end_timestamp":1792036800000}`, string(encoded), "params should send the time range in Unix milliseconds")
}

// testHistoricalVolatility is the mock realised volatility of BTC
func testHistoricalVolatility() []HistoricalVolatilityPoint {
	return []HistoricalVolatilityPoint{
		{
			Timestamp:  marketDataMilli(1790737200000),
			Volatility: 35.43876495973024,
		},
		{
			Timestamp:  marketDataMilli(1790737200000),
			Volatility: 35.43876495973024,
		},
		{
			Timestamp:  marketDataMilli(1790740800000),
			Volatility: 35.44102537206597,
		},
		{
			Timestamp:  marketDataMilli(1790744400000),
			Volatility: 35.408148591776786,
		},
		{
			Timestamp:  marketDataMilli(1790748000000),
			Volatility: 35.40829104147803,
		},
	}
}

func TestGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalVolatility(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetHistoricalVolatility must reject an empty currency")

	result, err := e.GetHistoricalVolatility(t.Context(), currency.BTC)
	require.NoError(t, err, "GetHistoricalVolatility must not error")
	if mockTests {
		assert.Equal(t, testHistoricalVolatility(), result, "GetHistoricalVolatility should decode every point")
	} else {
		assert.NotEmpty(t, result, "GetHistoricalVolatility should return BTC's realised volatility")
	}

	_, err = newUnavailableServerExchange(t).GetHistoricalVolatility(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetHistoricalVolatility should return the request failure")
}

func TestHistoricalVolatilityPointUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result HistoricalVolatilityPoint
	require.NoError(t, json.Unmarshal([]byte(`[1549720800000,14.747743607344217]`), &result), "Unmarshal must not error")
	assert.Equal(t, HistoricalVolatilityPoint{Timestamp: marketDataMilli(1549720800000), Volatility: 14.747743607344217}, result, "UnmarshalJSON should decode the timestamp and volatility")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[1549720800000]`), &result), errInvalidTuple, "UnmarshalJSON should reject a tuple without a volatility")
	assert.Error(t, json.Unmarshal([]byte(`[1549720800000,"high"]`), &result), "UnmarshalJSON should reject a volatility that is not a number")
}

// testIndexChart is the mock hour of the btc_usd index chart
func testIndexChart() []IndexChartPoint {
	return []IndexChartPoint{
		{Timestamp: marketDataMilli(1792106196000), Price: 82098.246667},
		{Timestamp: marketDataMilli(1792106202000), Price: 82095.938333},
		{Timestamp: marketDataMilli(1792106208000), Price: 82095.711667},
		{Timestamp: marketDataMilli(1792106214000), Price: 82096.98},
		{Timestamp: marketDataMilli(1792106220000), Price: 82096.57},
	}
}

func TestGetIndexChartData(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexChartData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIndexChartData must reject a nil request")
	_, err = e.GetIndexChartData(t.Context(), &IndexChartDataRequest{Range: IndexChartRangeOneHour})
	require.ErrorIs(t, err, errIndexNameRequired, "GetIndexChartData must reject an empty index name")
	_, err = e.GetIndexChartData(t.Context(), &IndexChartDataRequest{IndexName: marketDataIndexName, Range: "2h"})
	require.ErrorIs(t, err, errInvalidIndexChartRange, "GetIndexChartData must reject a range Deribit does not chart")

	arg := &IndexChartDataRequest{IndexName: marketDataIndexName, Range: IndexChartRangeOneHour}
	result, err := e.GetIndexChartData(t.Context(), arg)
	require.NoError(t, err, "GetIndexChartData must not error")
	if mockTests {
		assert.Equal(t, testIndexChart(), result, "GetIndexChartData should decode every point")
	} else {
		assert.NotEmpty(t, result, "GetIndexChartData should return the index chart")
	}

	_, err = newUnavailableServerExchange(t).GetIndexChartData(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetIndexChartData should return the request failure")
}

func TestIndexChartPointUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result IndexChartPoint
	require.NoError(t, json.Unmarshal([]byte(`[1573228800000,8751.7138636]`), &result), "Unmarshal must not error")
	assert.Equal(t, IndexChartPoint{Timestamp: marketDataMilli(1573228800000), Price: 8751.7138636}, result, "UnmarshalJSON should decode the timestamp and price")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[1573228800000,8751.7138636,1]`), &result), errInvalidTuple, "UnmarshalJSON should reject a tuple with an extra element")
	assert.Error(t, json.Unmarshal([]byte(`{"price":8751.7138636}`), &result), "UnmarshalJSON should reject a point that is not a tuple")
}

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPrice(t.Context(), "")
	require.ErrorIs(t, err, errIndexNameRequired, "GetIndexPrice must reject an empty index name")

	result, err := e.GetIndexPrice(t.Context(), marketDataIndexName)
	require.NoError(t, err, "GetIndexPrice must not error")
	if mockTests {
		assert.Equal(t, &IndexPriceResponse{IndexPrice: 82285.99, EstimatedDeliveryPrice: 82285.99}, result, "GetIndexPrice should decode every field")
	} else {
		assert.Positive(t, result.IndexPrice, "GetIndexPrice should return the index price")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetIndexPrice(t.Context(), marketDataIndexName)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetIndexPrice should reject a null result")
	_, err = newUnavailableServerExchange(t).GetIndexPrice(t.Context(), marketDataIndexName)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetIndexPrice should return the request failure")
}

// testIndexPriceNames are the mock index names, which Deribit sends as bare strings without extended details
func testIndexPriceNames() []IndexPriceName {
	return []IndexPriceName{{Name: "ada_usd"}, {Name: "ada_usdc"}, {Name: "ada_usdt"}, {Name: "algo_usd"}, {Name: "algo_usdc"}}
}

func TestGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	result, err := e.GetIndexPriceNames(t.Context(), false)
	require.NoError(t, err, "GetIndexPriceNames must not error")
	if mockTests {
		assert.Equal(t, testIndexPriceNames(), result, "GetIndexPriceNames should decode every name")
	} else {
		assert.Contains(t, result, IndexPriceName{Name: marketDataIndexName}, "GetIndexPriceNames should return the btc_usd index")
		result, err = e.GetIndexPriceNames(t.Context(), true)
		require.NoError(t, err, "GetIndexPriceNames must not error with extended details")
		assert.Contains(t, result, IndexPriceName{Name: marketDataIndexName, FutureComboCreationEnabled: true, OptionComboCreationEnabled: true}, "GetIndexPriceNames should return btc_usd's combo creation settings")
	}

	// The recorder blanks name fields, so the extended details are served as Deribit sent them
	result, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":[{"name":"ada_usd"},{"name":"btc_usd","future_combo_creation_enabled":true,"option_combo_creation_enabled":true},{"name":"btc_usdc","option_combo_creation_enabled":true}]}`).GetIndexPriceNames(t.Context(), true)
	require.NoError(t, err, "GetIndexPriceNames must not error with extended details")
	exp := []IndexPriceName{{Name: "ada_usd"}, {Name: "btc_usd", FutureComboCreationEnabled: true, OptionComboCreationEnabled: true}, {Name: "btc_usdc", OptionComboCreationEnabled: true}}
	assert.Equal(t, exp, result, "GetIndexPriceNames should decode the extended details")
	_, err = newUnavailableServerExchange(t).GetIndexPriceNames(t.Context(), false)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetIndexPriceNames should return the request failure")
}

func TestIndexPriceNameUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   IndexPriceName
	}{
		{input: `"btc_usd"`, exp: IndexPriceName{Name: "btc_usd"}},
		{input: `{"name":"btc_usdc"}`, exp: IndexPriceName{Name: "btc_usdc"}},
		{input: `{"name":"btc_usd","future_combo_creation_enabled":true,"option_combo_creation_enabled":true}`, exp: IndexPriceName{Name: "btc_usd", FutureComboCreationEnabled: true, OptionComboCreationEnabled: true}},
		{input: `null`},
	} {
		var result IndexPriceName
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &result), "Unmarshal must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, result, "UnmarshalJSON should decode %s", tc.input)
	}
	var result IndexPriceName
	assert.Error(t, json.Unmarshal([]byte(`["btc_usd"]`), &result), "UnmarshalJSON should reject a name that is neither a string nor an object")
}

// testOptionInstrument is the mock specification of mockMarketDataOption
func testOptionInstrument() *Instrument {
	return &Instrument{
		InstrumentName:      "BTC-27NOV26-90000-C",
		InstrumentID:        683239,
		Kind:                "option",
		InstrumentType:      "reversed",
		OptionType:          "call",
		Strike:              90000,
		State:               "open",
		IsActive:            true,
		BaseCurrency:        currency.BTC,
		QuoteCurrency:       currency.BTC,
		CounterCurrency:     currency.USD,
		BaseCurrencyUUID:    "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
		QuoteCurrencyUUID:   "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
		SettlementCurrency:  currency.BTC,
		SettlementPeriod:    "month",
		PriceIndex:          "btc_usd",
		IndexID:             1000033,
		ProductGroup:        "BTC",
		UnderlyingType:      "crypto",
		CreationTimestamp:   marketDataMilli(1787817624000),
		ExpirationTimestamp: marketDataMilli(1795766400000),
		TickSize:            0.0001,
		TickSizeSteps: []TickSizeStep{
			{AbovePrice: 0.005, TickSize: 0.0005},
		},
		ContractSize:             1,
		MinTradeAmount:           0.1,
		LotSize:                  1,
		MakerCommission:          0.0003,
		TakerCommission:          0.0003,
		BlockTradeCommission:     0.0003,
		BlockTradeTickSize:       0.0001,
		BlockTradeMinTradeAmount: 25,
	}
}

// testCoinbaseRoutedInstrument is the mock specification of marketDataCoinbaseRoutedSpot
func testCoinbaseRoutedInstrument() *Instrument {
	return &Instrument{
		InstrumentName:           "AAVE_USDC",
		InstrumentID:             700476,
		Kind:                     "spot",
		InstrumentType:           "linear",
		State:                    "open",
		IsActive:                 true,
		BaseCurrency:             currency.AAVE,
		QuoteCurrency:            currency.USDC,
		CounterCurrency:          currency.USDC,
		BaseCurrencyUUID:         "b1646860-6b23-5f7d-a506-cba0902f0ca9",
		QuoteCurrencyUUID:        "2b92315d-eab7-5bef-84fa-089a131333f5",
		PriceIndex:               "aave_usdc",
		IndexID:                  34000004,
		ProductGroup:             "TIER_3",
		UnderlyingType:           "crypto",
		CreationTimestamp:        marketDataMilli(1790089556000),
		ExpirationTimestamp:      marketDataMilli(32503708800000),
		TickSize:                 0.01,
		TickSizeSteps:            []TickSizeStep{},
		ContractSize:             0.001,
		MinTradeAmount:           0.001,
		MakerCommission:          0.0002,
		TakerCommission:          0.0005,
		MaxLeverage:              1,
		IsCoinbaseSpotRouted:     true,
		IsCoinbaseExchangeRouted: true,
	}
}

// testInstrumentNotFound is the error Deribit returns for marketDataUnknownInstrument
func testInstrumentNotFound() *APIError {
	return &APIError{Code: -32602, Message: "Invalid params", Data: json.RawMessage(`{"param":"instrument_name","reason":"instrument not found"}`)}
}

func TestGetInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstrument(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetInstrument must reject an empty instrument name")

	result, err := e.GetInstrument(t.Context(), marketDataOption(t))
	require.NoError(t, err, "GetInstrument must not error for an option")
	if mockTests {
		assert.Equal(t, testOptionInstrument(), result, "GetInstrument should decode every field of an option")
	} else {
		assert.Equal(t, KindOption, result.Kind, "GetInstrument should return an option")
	}

	result, err = e.GetInstrument(t.Context(), marketDataCoinbaseRoutedSpot)
	require.NoError(t, err, "GetInstrument must not error for Coinbase-routed spot")
	if mockTests {
		assert.Equal(t, testCoinbaseRoutedInstrument(), result, "GetInstrument should decode every field of Coinbase-routed spot")
	} else {
		assert.True(t, result.IsCoinbaseExchangeRouted, "GetInstrument should report spot routed to Coinbase Exchange")
	}

	_, err = e.GetInstrument(t.Context(), marketDataUnknownInstrument)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "GetInstrument must return Deribit's error for an unknown instrument")
	if mockTests {
		assert.Equal(t, testInstrumentNotFound(), apiErr, "GetInstrument should decode Deribit's error")
	} else {
		assert.Equal(t, int64(-32602), apiErr.Code, "GetInstrument should return Deribit's invalid params error")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetInstrument(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetInstrument should reject a null result")
	_, err = newUnavailableServerExchange(t).GetInstrument(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetInstrument should return the request failure")
}

func TestInstrumentUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result Instrument
	require.NoError(t, json.Unmarshal([]byte(`{"instrument_name":"BTC-PERPETUAL","creation_timestamp":1534242287000,"expiration_timestamp":32503708800000}`), &result), "Unmarshal must not error")
	exp := Instrument{InstrumentName: marketDataPerpetual, CreationTimestamp: marketDataMilli(1534242287000), ExpirationTimestamp: marketDataMilli(32503708800000)}
	assert.Equal(t, exp, result, "UnmarshalJSON should decode every field, reading the expiration in milliseconds")
	assert.Equal(t, 3000, result.ExpirationTimestamp.Time().UTC().Year(), "UnmarshalJSON should decode a perpetual's expiration in the year 3000")

	result = Instrument{ExpirationTimestamp: marketDataMilli(1795766400000)}
	require.NoError(t, json.Unmarshal([]byte(`{"instrument_name":"BTC-27NOV26-90000-C"}`), &result), "Unmarshal must not error without an expiration")
	assert.Equal(t, marketDataMilli(1795766400000), result.ExpirationTimestamp, "UnmarshalJSON should leave the expiration unchanged when it is left out")
	require.NoError(t, json.Unmarshal([]byte(`{"expiration_timestamp":0}`), &result), "Unmarshal must not error for a zero expiration")
	assert.Zero(t, result.ExpirationTimestamp, "UnmarshalJSON should decode a zero expiration as the zero time")
	assert.Error(t, json.Unmarshal([]byte(`{"expiration_timestamp":"soon"}`), &result), "UnmarshalJSON should reject an expiration that is not a number")
	assert.Error(t, json.Unmarshal([]byte(`{"is_active":"yes"}`), &result), "UnmarshalJSON should reject a field of the wrong type")
}

// testFutureInstruments are the mock extended specifications of BTC futures
func testFutureInstruments() []Instrument {
	return []Instrument{
		{
			InstrumentName:           "BTC-PERPETUAL",
			InstrumentID:             210838,
			Kind:                     "future",
			InstrumentType:           "reversed",
			FutureType:               "reversed",
			State:                    "open",
			IsActive:                 true,
			BaseCurrency:             currency.BTC,
			QuoteCurrency:            currency.USD,
			CounterCurrency:          currency.USD,
			BaseCurrencyUUID:         "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			QuoteCurrencyUUID:        "dc311642-e9fb-4ba9-a312-c75c1aef4ae1",
			SettlementCurrency:       currency.BTC,
			SettlementPeriod:         "perpetual",
			PriceIndex:               "btc_usd",
			IndexID:                  1000033,
			ProductGroup:             "BTC",
			UnderlyingType:           "crypto",
			CreationTimestamp:        marketDataMilli(1534242287000),
			ExpirationTimestamp:      marketDataMilli(32503708800000),
			TickSize:                 0.5,
			TickSizeSteps:            []TickSizeStep{},
			ContractSize:             10,
			MinTradeAmount:           10,
			LotSize:                  1,
			MakerCommission:          0.00015,
			TakerCommission:          0.00035,
			MaxLiquidationCommission: 0.01,
			MakerMaxCommission:       0.25,
			TakerMaxCommission:       0.25,
			LiquidationMaxCommission: 0.25,
			SettlementMaxCommission:  0.25,
			MarginParameters: &MarginParameters{
				InitialIncrement: 0.0175,
				MarginFee:        0.01,
				MarginIncrement:  5e-05,
				MarginInitial:    0.0025,
				MarginPower:      1,
			},
			TradingBandwidthFraction: 0.015,
			PerpetualDampener:        0.00025,
			PerpetualMaxFunding:      0.005,
			BlockTradeCommission:     0.00025,
			BlockTradeTickSize:       0.01,
			BlockTradeMinTradeAmount: 200000,
			MaxLeverage:              50,
			MaxNonDefaultLeverage:    5,
		},
		{
			InstrumentName:           "BTC-16OCT26",
			InstrumentID:             701628,
			Kind:                     "future",
			InstrumentType:           "reversed",
			FutureType:               "reversed",
			State:                    "open",
			IsActive:                 true,
			BaseCurrency:             currency.BTC,
			QuoteCurrency:            currency.USD,
			CounterCurrency:          currency.USD,
			BaseCurrencyUUID:         "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			QuoteCurrencyUUID:        "dc311642-e9fb-4ba9-a312-c75c1aef4ae1",
			SettlementCurrency:       currency.BTC,
			SettlementPeriod:         "week",
			PriceIndex:               "btc_usd",
			IndexID:                  1000033,
			ProductGroup:             "BTC",
			UnderlyingType:           "crypto",
			CreationTimestamp:        marketDataMilli(1790236823000),
			ExpirationTimestamp:      marketDataMilli(1792137600000),
			TickSize:                 2.5,
			TickSizeSteps:            []TickSizeStep{},
			ContractSize:             10,
			MinTradeAmount:           10,
			LotSize:                  1,
			MakerCommission:          0.00015,
			TakerCommission:          0.00035,
			MaxLiquidationCommission: 0.01,
			MakerMaxCommission:       0.25,
			TakerMaxCommission:       0.25,
			LiquidationMaxCommission: 0.25,
			SettlementMaxCommission:  0.25,
			MarginParameters: &MarginParameters{
				InitialIncrement: 0.0275,
				MarginFee:        0.01,
				MarginIncrement:  5e-05,
				MarginInitial:    0.0125,
				MarginPower:      1,
			},
			TradingBandwidthFraction: 0.015,
			BlockTradeCommission:     0.00025,
			BlockTradeTickSize:       0.01,
			BlockTradeMinTradeAmount: 200000,
			MaxLeverage:              25,
			MaxNonDefaultLeverage:    5,
		},
		{
			InstrumentName:           "BTC-17OCT26",
			InstrumentID:             713918,
			Kind:                     "future",
			InstrumentType:           "reversed",
			FutureType:               "reversed",
			State:                    "open",
			IsActive:                 true,
			BaseCurrency:             currency.BTC,
			QuoteCurrency:            currency.USD,
			CounterCurrency:          currency.USD,
			BaseCurrencyUUID:         "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			QuoteCurrencyUUID:        "dc311642-e9fb-4ba9-a312-c75c1aef4ae1",
			SettlementCurrency:       currency.BTC,
			SettlementPeriod:         "day",
			PriceIndex:               "btc_usd",
			IndexID:                  1000033,
			ProductGroup:             "BTC",
			UnderlyingType:           "crypto",
			CreationTimestamp:        marketDataMilli(1791878412000),
			ExpirationTimestamp:      marketDataMilli(1792224000000),
			TickSize:                 2.5,
			TickSizeSteps:            []TickSizeStep{},
			ContractSize:             10,
			MinTradeAmount:           10,
			LotSize:                  1,
			MakerCommission:          0.00015,
			TakerCommission:          0.00035,
			MaxLiquidationCommission: 0.01,
			MakerMaxCommission:       0.25,
			TakerMaxCommission:       0.25,
			LiquidationMaxCommission: 0.25,
			SettlementMaxCommission:  0.25,
			MarginParameters: &MarginParameters{
				InitialIncrement: 0.0275,
				MarginFee:        0.01,
				MarginIncrement:  5e-05,
				MarginInitial:    0.0125,
				MarginPower:      1,
			},
			TradingBandwidthFraction: 0.015,
			BlockTradeCommission:     0.00025,
			BlockTradeTickSize:       0.01,
			BlockTradeMinTradeAmount: 200000,
			MaxLeverage:              25,
			MaxNonDefaultLeverage:    5,
		},
		{
			InstrumentName:           "BTC-18OCT26",
			InstrumentID:             714449,
			Kind:                     "future",
			InstrumentType:           "reversed",
			FutureType:               "reversed",
			State:                    "open",
			IsActive:                 true,
			BaseCurrency:             currency.BTC,
			QuoteCurrency:            currency.USD,
			CounterCurrency:          currency.USD,
			BaseCurrencyUUID:         "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			QuoteCurrencyUUID:        "dc311642-e9fb-4ba9-a312-c75c1aef4ae1",
			SettlementCurrency:       currency.BTC,
			SettlementPeriod:         "day",
			PriceIndex:               "btc_usd",
			IndexID:                  1000033,
			ProductGroup:             "BTC",
			UnderlyingType:           "crypto",
			CreationTimestamp:        marketDataMilli(1791964810000),
			ExpirationTimestamp:      marketDataMilli(1792310400000),
			TickSize:                 2.5,
			TickSizeSteps:            []TickSizeStep{},
			ContractSize:             10,
			MinTradeAmount:           10,
			LotSize:                  1,
			MakerCommission:          0.00015,
			TakerCommission:          0.00035,
			MaxLiquidationCommission: 0.01,
			MakerMaxCommission:       0.25,
			TakerMaxCommission:       0.25,
			LiquidationMaxCommission: 0.25,
			SettlementMaxCommission:  0.25,
			MarginParameters: &MarginParameters{
				InitialIncrement: 0.0275,
				MarginFee:        0.01,
				MarginIncrement:  5e-05,
				MarginInitial:    0.0125,
				MarginPower:      1,
			},
			TradingBandwidthFraction: 0.015,
			BlockTradeCommission:     0.00025,
			BlockTradeTickSize:       0.01,
			BlockTradeMinTradeAmount: 200000,
			MaxLeverage:              25,
			MaxNonDefaultLeverage:    5,
		},
		{
			InstrumentName:           "BTC-19OCT26",
			InstrumentID:             715012,
			Kind:                     "future",
			InstrumentType:           "reversed",
			FutureType:               "reversed",
			State:                    "open",
			IsActive:                 true,
			BaseCurrency:             currency.BTC,
			QuoteCurrency:            currency.USD,
			CounterCurrency:          currency.USD,
			BaseCurrencyUUID:         "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			QuoteCurrencyUUID:        "dc311642-e9fb-4ba9-a312-c75c1aef4ae1",
			SettlementCurrency:       currency.BTC,
			SettlementPeriod:         "day",
			PriceIndex:               "btc_usd",
			IndexID:                  1000033,
			ProductGroup:             "BTC",
			UnderlyingType:           "crypto",
			CreationTimestamp:        marketDataMilli(1792051211000),
			ExpirationTimestamp:      marketDataMilli(1792396800000),
			TickSize:                 2.5,
			TickSizeSteps:            []TickSizeStep{},
			ContractSize:             10,
			MinTradeAmount:           10,
			LotSize:                  1,
			MakerCommission:          0.00015,
			TakerCommission:          0.00035,
			MaxLiquidationCommission: 0.01,
			MakerMaxCommission:       0.25,
			TakerMaxCommission:       0.25,
			LiquidationMaxCommission: 0.25,
			SettlementMaxCommission:  0.25,
			MarginParameters: &MarginParameters{
				InitialIncrement: 0.0275,
				MarginFee:        0.01,
				MarginIncrement:  5e-05,
				MarginInitial:    0.0125,
				MarginPower:      1,
			},
			TradingBandwidthFraction: 0.015,
			BlockTradeCommission:     0.00025,
			BlockTradeTickSize:       0.01,
			BlockTradeMinTradeAmount: 200000,
			MaxLeverage:              25,
			MaxNonDefaultLeverage:    5,
		},
	}
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstruments(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInstruments must reject a nil request")
	_, err = e.GetInstruments(t.Context(), &InstrumentsRequest{Kind: KindFuture})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetInstruments must reject an empty currency")
	_, err = e.GetInstruments(t.Context(), &InstrumentsRequest{Currency: currency.BTC, Kind: KindCombo})
	require.ErrorIs(t, err, errInvalidKind, "GetInstruments must reject a kind it does not accept")

	arg := &InstrumentsRequest{Currency: currency.BTC, Kind: KindFuture, Extended: true}
	result, err := e.GetInstruments(t.Context(), arg)
	require.NoError(t, err, "GetInstruments must not error")
	if mockTests {
		assert.Equal(t, testFutureInstruments(), result, "GetInstruments should decode every field")
	} else {
		require.NotEmpty(t, result, "GetInstruments must return BTC's futures")
		assert.Equal(t, marketDataPerpetual, result[0].InstrumentName, "GetInstruments should list the perpetual first")
		assert.NotNil(t, result[0].MarginParameters, "GetInstruments should return the extended margin parameters")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			assert.JSONEq(t, `{"jsonrpc":"2.0","method":"public/get_instruments","params":{"currency":"any","kind":"spot"}}`, string(body), "GetInstruments should send currency.ANY in lowercase, as Deribit rejects ANY")
		}
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","result":[]}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = ex.GetInstruments(t.Context(), &InstrumentsRequest{Currency: currency.ANY, Kind: KindSpot})
	require.NoError(t, err, "GetInstruments must not error for every currency")
	_, err = newUnavailableServerExchange(t).GetInstruments(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetInstruments should return the request failure")
}

func TestMarketDataCurrency(t *testing.T) {
	t.Parallel()
	for _, ccy := range []currency.Code{currency.ANY, currency.NewCode("any"), currency.NewCode("Any")} {
		assert.Equalf(t, "any", marketDataCurrency(ccy).String(), "marketDataCurrency should lowercase %s", ccy)
	}
	assert.Equal(t, "BTC", marketDataCurrency(currency.BTC).String(), "marketDataCurrency should leave other currencies unchanged")
	assert.True(t, marketDataCurrency(currency.EMPTYCODE).IsEmpty(), "marketDataCurrency should leave an empty currency empty")
}

// testLastSettlementsByCurrencyRequest requests BTC's last 3 deliveries before 2026-10-15
func testLastSettlementsByCurrencyRequest() *LastSettlementsByCurrencyRequest {
	return &LastSettlementsByCurrencyRequest{Currency: currency.BTC, Type: "delivery", Count: 3, SearchStartTime: time.UnixMilli(marketDataRangeStart)}
}

// testLastDeliveries are the mock deliveries of BTC futures
func testLastDeliveries() *SettlementsResponse {
	return &SettlementsResponse{
		Settlements: []Settlement{
			{
				Type:              "delivery",
				Timestamp:         marketDataMilli(1791964800086),
				InstrumentName:    "BTC-14OCT26",
				IndexPrice:        83396.69,
				MarkPrice:         83396.69,
				Position:          8052900,
				ProfitLoss:        0.18440584600000007,
				SessionProfitLoss: 4229.903116782007,
			},
			{
				Type:              "delivery",
				Timestamp:         marketDataMilli(1791878400102),
				InstrumentName:    "BTC-13OCT26",
				IndexPrice:        83826.89,
				MarkPrice:         83826.89,
				Position:          18316830,
				ProfitLoss:        0.012648237000000239,
				SessionProfitLoss: 3572.261750989998,
			},
			{
				Type:              "delivery",
				Timestamp:         marketDataMilli(1791792000074),
				InstrumentName:    "BTC-12OCT26",
				IndexPrice:        83911.12,
				MarkPrice:         83911.12,
				Position:          29955640,
				ProfitLoss:        -0.6642972930000006,
				SessionProfitLoss: 15997.845396779021,
			},
		},
		Continuation: "47y8wJn3q8WP6ZCDrkVUTDD9T6b17Un3yBAzYgz5Fjd9qcEwSZcDC4SAw55t1mqM1y",
	}
}

func TestGetLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastSettlementsByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastSettlementsByCurrency must reject a nil request")
	_, err = e.GetLastSettlementsByCurrency(t.Context(), &LastSettlementsByCurrencyRequest{Count: 3})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLastSettlementsByCurrency must reject an empty currency")
	_, err = e.GetLastSettlementsByCurrency(t.Context(), &LastSettlementsByCurrencyRequest{Currency: currency.BTC, Type: "exercise"})
	require.ErrorIs(t, err, errInvalidLastSettlementsType, "GetLastSettlementsByCurrency must reject a type Deribit does not filter by")

	result, err := e.GetLastSettlementsByCurrency(t.Context(), testLastSettlementsByCurrencyRequest())
	require.NoError(t, err, "GetLastSettlementsByCurrency must not error")
	if mockTests {
		assert.Equal(t, testLastDeliveries(), result, "GetLastSettlementsByCurrency should decode every field")
	} else {
		assert.Len(t, result.Settlements, 3, "GetLastSettlementsByCurrency should return the deliveries requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLastSettlementsByCurrency(t.Context(), testLastSettlementsByCurrencyRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastSettlementsByCurrency should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLastSettlementsByCurrency(t.Context(), testLastSettlementsByCurrencyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastSettlementsByCurrency should return the request failure")
}

// testLastSettlementsByInstrumentRequest requests marketDataPerpetual's last 2 settlements before 2026-10-15
func testLastSettlementsByInstrumentRequest() *LastSettlementsByInstrumentRequest {
	return &LastSettlementsByInstrumentRequest{InstrumentName: marketDataPerpetual, Type: "settlement", Count: 2, SearchStartTime: time.UnixMilli(marketDataRangeStart)}
}

// testPerpetualLastSettlements are the mock settlements of marketDataPerpetual
func testPerpetualLastSettlements() *SettlementsResponse {
	return &SettlementsResponse{
		Settlements: []Settlement{
			{
				Type:              "settlement",
				Timestamp:         marketDataMilli(1791964800086),
				InstrumentName:    "BTC-PERPETUAL",
				IndexPrice:        83489.6,
				MarkPrice:         83515.41,
				Position:          841636120,
				ProfitLoss:        0.46555281100000034,
				SessionProfitLoss: 115806.49386799286,
			},
			{
				Type:              "settlement",
				Timestamp:         marketDataMilli(1791878400102),
				InstrumentName:    "BTC-PERPETUAL",
				IndexPrice:        83773.42,
				MarkPrice:         83808.52,
				Position:          842468950,
				ProfitLoss:        1.5328155339999916,
				SessionProfitLoss: 86988.37391506313,
			},
		},
		Continuation: "23nYDkVbTdcxaMk3a5XVnq1BJhj8RNQsELkLABrdtXWNmgRHvotUVWKWMEjA8Ux9aCbc6",
	}
}

func TestGetLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastSettlementsByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastSettlementsByInstrument must reject a nil request")
	_, err = e.GetLastSettlementsByInstrument(t.Context(), &LastSettlementsByInstrumentRequest{Count: 2})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetLastSettlementsByInstrument must reject an empty instrument name")
	_, err = e.GetLastSettlementsByInstrument(t.Context(), &LastSettlementsByInstrumentRequest{InstrumentName: marketDataPerpetual, Type: "funding"})
	require.ErrorIs(t, err, errInvalidLastSettlementsType, "GetLastSettlementsByInstrument must reject a type Deribit does not filter by")

	result, err := e.GetLastSettlementsByInstrument(t.Context(), testLastSettlementsByInstrumentRequest())
	require.NoError(t, err, "GetLastSettlementsByInstrument must not error")
	if mockTests {
		assert.Equal(t, testPerpetualLastSettlements(), result, "GetLastSettlementsByInstrument should decode every field")
	} else {
		assert.Len(t, result.Settlements, 2, "GetLastSettlementsByInstrument should return the settlements requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLastSettlementsByInstrument(t.Context(), testLastSettlementsByInstrumentRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastSettlementsByInstrument should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLastSettlementsByInstrument(t.Context(), testLastSettlementsByInstrumentRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastSettlementsByInstrument should return the request failure")
}

func TestValidateLastSettlementsType(t *testing.T) {
	t.Parallel()
	for _, settlementType := range []string{"", "settlement", "delivery", "bankruptcy"} {
		assert.NoErrorf(t, validateLastSettlementsType(settlementType), "validateLastSettlementsType should accept type %q", settlementType)
	}
	for _, settlementType := range []string{"exercise", "Delivery"} {
		assert.ErrorIsf(t, validateLastSettlementsType(settlementType), errInvalidLastSettlementsType, "validateLastSettlementsType should reject type %q", settlementType)
	}
}

// testLastTradesByCurrencyRequest requests BTC futures trades by trade ID, which live tests leave out, as Deribit only
// serves the last 24 hours of trades
func testLastTradesByCurrencyRequest() *LastTradesByCurrencyRequest {
	if mockTests {
		return &LastTradesByCurrencyRequest{Currency: currency.BTC, Kind: KindFuture, StartID: mockLastTradesStartID, EndID: mockLastTradesEndID, Sorting: "asc"}
	}
	return &LastTradesByCurrencyRequest{Currency: currency.BTC, Kind: KindFuture, Count: 5}
}

// testLastFutureTrades are the mock BTC futures trades between mockLastTradesStartID and mockLastTradesEndID
func testLastFutureTrades() *LastTradesResponse {
	return &LastTradesResponse{
		Trades: []PublicTrade{
			{
				TradeID:           "452869730",
				TradeSequence:     306491081,
				InstrumentName:    "BTC-PERPETUAL",
				Timestamp:         marketDataMilli(1792105201401),
				StarbaseTimestamp: marketDataNano(1792105201401648000),
				StarbaseMatchID:   236624182618632200,
				Direction:         "sell",
				TickDirection:     3,
				Price:             82385,
				Amount:            40,
				Contracts:         4,
				MarkPrice:         82379.55,
				IndexPrice:        82364.9,
			},
			{
				TradeID:           "452869731",
				TradeSequence:     15924,
				InstrumentName:    "BTC-16OCT26",
				Timestamp:         marketDataMilli(1792105204454),
				StarbaseTimestamp: marketDataNano(1792105204454435600),
				StarbaseMatchID:   236624195423842300,
				Direction:         "sell",
				TickDirection:     2,
				Price:             82370,
				Amount:            8240,
				Contracts:         824,
				MarkPrice:         82368.14,
				IndexPrice:        82364.79,
			},
			{
				TradeID:           "452869732",
				TradeSequence:     331,
				InstrumentName:    "BTC-6NOV26",
				Timestamp:         marketDataMilli(1792105204777),
				StarbaseTimestamp: marketDataNano(1792105204777546200),
				StarbaseMatchID:   236624196778602500,
				Direction:         "sell",
				TickDirection:     2,
				Price:             82652.5,
				Amount:            2500,
				Contracts:         250,
				MarkPrice:         82652.76,
				IndexPrice:        82364.79,
			},
			{
				TradeID:           "452869733",
				TradeSequence:     306491082,
				InstrumentName:    "BTC-PERPETUAL",
				Timestamp:         marketDataMilli(1792105204790),
				StarbaseTimestamp: marketDataNano(1792105204790407700),
				StarbaseMatchID:   236624196833128450,
				Direction:         "sell",
				TickDirection:     3,
				Price:             82385,
				Amount:            2320,
				Contracts:         232,
				MarkPrice:         82379.91,
				IndexPrice:        82364.79,
			},
			{
				TradeID:           "452869734",
				TradeSequence:     306491083,
				InstrumentName:    "BTC-PERPETUAL",
				Timestamp:         marketDataMilli(1792105204790),
				StarbaseTimestamp: marketDataNano(1792105204790407700),
				StarbaseMatchID:   236624196833128450,
				Direction:         "sell",
				TickDirection:     3,
				Price:             82385,
				Amount:            27780,
				Contracts:         2778,
				MarkPrice:         82379.91,
				IndexPrice:        82364.79,
			},
		},
	}
}

func TestGetLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastTradesByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastTradesByCurrency must reject a nil request")
	_, err = e.GetLastTradesByCurrency(t.Context(), &LastTradesByCurrencyRequest{Kind: KindFuture})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLastTradesByCurrency must reject an empty currency")
	_, err = e.GetLastTradesByCurrency(t.Context(), &LastTradesByCurrencyRequest{Currency: currency.BTC, Kind: "perpetual"})
	require.ErrorIs(t, err, errInvalidKind, "GetLastTradesByCurrency must reject a kind it does not accept")
	_, err = e.GetLastTradesByCurrency(t.Context(), &LastTradesByCurrencyRequest{Currency: currency.BTC, Sorting: "up"})
	require.ErrorIs(t, err, errInvalidLastTradesSorting, "GetLastTradesByCurrency must reject a sorting Deribit does not document")
	_, err = e.GetLastTradesByCurrency(t.Context(), &LastTradesByCurrencyRequest{Currency: currency.BTC, StartTime: time.UnixMilli(mockLastTradesRangeEnd), EndTime: time.UnixMilli(mockLastTradesRangeStart)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLastTradesByCurrency must reject a time range starting after it ends")

	result, err := e.GetLastTradesByCurrency(t.Context(), testLastTradesByCurrencyRequest())
	require.NoError(t, err, "GetLastTradesByCurrency must not error")
	if mockTests {
		assert.Equal(t, testLastFutureTrades(), result, "GetLastTradesByCurrency should decode every field")
	} else {
		assert.NotEmpty(t, result.Trades, "GetLastTradesByCurrency should return BTC's latest futures trades")
	}

	result, err = e.GetLastTradesByCurrency(t.Context(), &LastTradesByCurrencyRequest{Currency: currency.BTC, Kind: KindCombo, Count: 1})
	require.NoError(t, err, "GetLastTradesByCurrency must not error for combo trades")
	if mockTests {
		exp := &LastTradesResponse{
			Trades: []PublicTrade{
				{
					TradeID:           "452874333",
					TradeSequence:     1,
					InstrumentName:    "BTC-CS-23OCT26-79000_82000",
					Timestamp:         marketDataMilli(1792108848174),
					StarbaseTimestamp: marketDataNano(1792108848174399000),
					StarbaseMatchID:   236639478293213200,
					Direction:         "buy",
					TickDirection:     1,
					Price:             0.026,
					Amount:            0.1,
					Contracts:         0.1,
					MarkPrice:         0.02545714,
					IndexPrice:        82139.7,
					Legs: []PublicTrade{
						{
							TradeID:           "452874335",
							TradeSequence:     94,
							InstrumentName:    "BTC-23OCT26-82000-C",
							Timestamp:         marketDataMilli(1792108848174),
							StarbaseTimestamp: marketDataNano(1792108848174399000),
							StarbaseMatchID:   236639478293213200,
							Direction:         "sell",
							Price:             0.0202,
							Amount:            0.1,
							Contracts:         0.1,
							MarkPrice:         0.02051307,
							IndexPrice:        82139.7,
							ImpliedVolatility: 33.04,
							ComboID:           "BTC-CS-23OCT26-79000_82000",
							ComboTradeID:      "452874333",
						},
						{
							TradeID:           "452874334",
							TradeSequence:     3,
							InstrumentName:    "BTC-23OCT26-79000-C",
							Timestamp:         marketDataMilli(1792108848174),
							StarbaseTimestamp: marketDataNano(1792108848174399000),
							StarbaseMatchID:   236639478293213200,
							Direction:         "buy",
							Price:             0.0462,
							Amount:            0.1,
							Contracts:         0.1,
							MarkPrice:         0.0459702,
							IndexPrice:        82139.7,
							ImpliedVolatility: 37.43,
							ComboID:           "BTC-CS-23OCT26-79000_82000",
							ComboTradeID:      "452874333",
						},
					},
				},
			},
			HasMore: true,
		}
		assert.Equal(t, exp, result, "GetLastTradesByCurrency should decode a combo trade's legs")
	} else {
		for i := range result.Trades {
			assert.NotEmptyf(t, result.Trades[i].Legs, "GetLastTradesByCurrency should return combo trade %s's legs", result.Trades[i].TradeID)
		}
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLastTradesByCurrency(t.Context(), testLastTradesByCurrencyRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastTradesByCurrency should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLastTradesByCurrency(t.Context(), testLastTradesByCurrencyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastTradesByCurrency should return the request failure")
}

// lastTradesRange returns the mock trade range, or the last hour when testing live, as Deribit only serves the last 24
// hours of trades
func lastTradesRange() (start, end time.Time) {
	if mockTests {
		return time.UnixMilli(mockLastTradesRangeStart), time.UnixMilli(mockLastTradesRangeEnd)
	}
	end = time.Now()
	return end.Add(-time.Hour), end
}

// testLastTradesByCurrencyAndTimeRequest requests BTC's first 3 futures trades of the trade range
func testLastTradesByCurrencyAndTimeRequest() *LastTradesByCurrencyAndTimeRequest {
	start, end := lastTradesRange()
	return &LastTradesByCurrencyAndTimeRequest{Currency: currency.BTC, Kind: KindFuture, StartTime: start, EndTime: end, Count: 3, Sorting: "asc"}
}

// testLastFutureTradesByTime are the mock first BTC futures trades of the trade range
func testLastFutureTradesByTime() *LastTradesResponse {
	return &LastTradesResponse{
		Trades: []PublicTrade{
			{
				TradeID:           "452869730",
				TradeSequence:     306491081,
				InstrumentName:    "BTC-PERPETUAL",
				Timestamp:         marketDataMilli(1792105201401),
				StarbaseTimestamp: marketDataNano(1792105201401648000),
				StarbaseMatchID:   236624182618632200,
				Direction:         "sell",
				TickDirection:     3,
				Price:             82385,
				Amount:            40,
				Contracts:         4,
				MarkPrice:         82379.55,
				IndexPrice:        82364.9,
			},
			{
				TradeID:           "452869731",
				TradeSequence:     15924,
				InstrumentName:    "BTC-16OCT26",
				Timestamp:         marketDataMilli(1792105204454),
				StarbaseTimestamp: marketDataNano(1792105204454435600),
				StarbaseMatchID:   236624195423842300,
				Direction:         "sell",
				TickDirection:     2,
				Price:             82370,
				Amount:            8240,
				Contracts:         824,
				MarkPrice:         82368.14,
				IndexPrice:        82364.79,
			},
			{
				TradeID:           "452869732",
				TradeSequence:     331,
				InstrumentName:    "BTC-6NOV26",
				Timestamp:         marketDataMilli(1792105204777),
				StarbaseTimestamp: marketDataNano(1792105204777546200),
				StarbaseMatchID:   236624196778602500,
				Direction:         "sell",
				TickDirection:     2,
				Price:             82652.5,
				Amount:            2500,
				Contracts:         250,
				MarkPrice:         82652.76,
				IndexPrice:        82364.79,
			},
		},
		HasMore: true,
	}
}

func TestGetLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	start, end := lastTradesRange()
	_, err := e.GetLastTradesByCurrencyAndTime(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastTradesByCurrencyAndTime must reject a nil request")
	_, err = e.GetLastTradesByCurrencyAndTime(t.Context(), &LastTradesByCurrencyAndTimeRequest{StartTime: start})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLastTradesByCurrencyAndTime must reject an empty currency")
	_, err = e.GetLastTradesByCurrencyAndTime(t.Context(), &LastTradesByCurrencyAndTimeRequest{Currency: currency.BTC, Kind: "perpetual", StartTime: start})
	require.ErrorIs(t, err, errInvalidKind, "GetLastTradesByCurrencyAndTime must reject a kind it does not accept")
	_, err = e.GetLastTradesByCurrencyAndTime(t.Context(), &LastTradesByCurrencyAndTimeRequest{Currency: currency.BTC, Sorting: "up", StartTime: start})
	require.ErrorIs(t, err, errInvalidLastTradesSorting, "GetLastTradesByCurrencyAndTime must reject a sorting Deribit does not document")
	_, err = e.GetLastTradesByCurrencyAndTime(t.Context(), &LastTradesByCurrencyAndTimeRequest{Currency: currency.BTC})
	require.ErrorIs(t, err, errLastTradesTimeRangeRequired, "GetLastTradesByCurrencyAndTime must reject a request without a start or end time")
	_, err = e.GetLastTradesByCurrencyAndTime(t.Context(), &LastTradesByCurrencyAndTimeRequest{Currency: currency.BTC, StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLastTradesByCurrencyAndTime must reject a time range starting after it ends")

	result, err := e.GetLastTradesByCurrencyAndTime(t.Context(), testLastTradesByCurrencyAndTimeRequest())
	require.NoError(t, err, "GetLastTradesByCurrencyAndTime must not error")
	if mockTests {
		assert.Equal(t, testLastFutureTradesByTime(), result, "GetLastTradesByCurrencyAndTime should decode every field")
	} else {
		assert.NotEmpty(t, result.Trades, "GetLastTradesByCurrencyAndTime should return BTC's futures trades of the last hour")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLastTradesByCurrencyAndTime(t.Context(), testLastTradesByCurrencyAndTimeRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastTradesByCurrencyAndTime should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLastTradesByCurrencyAndTime(t.Context(), testLastTradesByCurrencyAndTimeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastTradesByCurrencyAndTime should return the request failure")
}

// testLastTradesByInstrumentRequest requests marketDataPerpetual's trades by sequence number, which live tests leave
// out, as Deribit only serves the last 24 hours of trades
func testLastTradesByInstrumentRequest() *LastTradesByInstrumentRequest {
	if mockTests {
		return &LastTradesByInstrumentRequest{InstrumentName: marketDataPerpetual, StartSequence: mockLastTradesStartSequence, EndSequence: mockLastTradesEndSequence, Sorting: "asc"}
	}
	return &LastTradesByInstrumentRequest{InstrumentName: marketDataPerpetual, Count: 3}
}

// testLastPerpetualTrades are the mock trades of marketDataPerpetual between mockLastTradesStartSequence and
// mockLastTradesEndSequence
func testLastPerpetualTrades() *LastTradesResponse {
	return &LastTradesResponse{
		Trades: []PublicTrade{
			{
				TradeID:           "452869730",
				TradeSequence:     306491081,
				InstrumentName:    "BTC-PERPETUAL",
				Timestamp:         marketDataMilli(1792105201401),
				StarbaseTimestamp: marketDataNano(1792105201401648000),
				StarbaseMatchID:   236624182618632200,
				Direction:         "sell",
				TickDirection:     3,
				Price:             82385,
				Amount:            40,
				Contracts:         4,
				MarkPrice:         82379.55,
				IndexPrice:        82364.9,
			},
			{
				TradeID:           "452869733",
				TradeSequence:     306491082,
				InstrumentName:    "BTC-PERPETUAL",
				Timestamp:         marketDataMilli(1792105204790),
				StarbaseTimestamp: marketDataNano(1792105204790407700),
				StarbaseMatchID:   236624196833128450,
				Direction:         "sell",
				TickDirection:     3,
				Price:             82385,
				Amount:            2320,
				Contracts:         232,
				MarkPrice:         82379.91,
				IndexPrice:        82364.79,
			},
			{
				TradeID:           "452869734",
				TradeSequence:     306491083,
				InstrumentName:    "BTC-PERPETUAL",
				Timestamp:         marketDataMilli(1792105204790),
				StarbaseTimestamp: marketDataNano(1792105204790407700),
				StarbaseMatchID:   236624196833128450,
				Direction:         "sell",
				TickDirection:     3,
				Price:             82385,
				Amount:            27780,
				Contracts:         2778,
				MarkPrice:         82379.91,
				IndexPrice:        82364.79,
			},
		},
	}
}

func TestGetLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastTradesByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastTradesByInstrument must reject a nil request")
	_, err = e.GetLastTradesByInstrument(t.Context(), &LastTradesByInstrumentRequest{Count: 3})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetLastTradesByInstrument must reject an empty instrument name")
	_, err = e.GetLastTradesByInstrument(t.Context(), &LastTradesByInstrumentRequest{InstrumentName: marketDataPerpetual, Sorting: "up"})
	require.ErrorIs(t, err, errInvalidLastTradesSorting, "GetLastTradesByInstrument must reject a sorting Deribit does not document")
	_, err = e.GetLastTradesByInstrument(t.Context(), &LastTradesByInstrumentRequest{InstrumentName: marketDataPerpetual, StartTime: time.UnixMilli(mockLastTradesRangeEnd), EndTime: time.UnixMilli(mockLastTradesRangeStart)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLastTradesByInstrument must reject a time range starting after it ends")

	result, err := e.GetLastTradesByInstrument(t.Context(), testLastTradesByInstrumentRequest())
	require.NoError(t, err, "GetLastTradesByInstrument must not error")
	if mockTests {
		assert.Equal(t, testLastPerpetualTrades(), result, "GetLastTradesByInstrument should decode every field")
	} else {
		assert.Len(t, result.Trades, 3, "GetLastTradesByInstrument should return the trades requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLastTradesByInstrument(t.Context(), testLastTradesByInstrumentRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastTradesByInstrument should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLastTradesByInstrument(t.Context(), testLastTradesByInstrumentRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastTradesByInstrument should return the request failure")
}

// testLastTradesByInstrumentAndTimeRequest requests marketDataPerpetual's last 2 trades of the trade range
func testLastTradesByInstrumentAndTimeRequest() *LastTradesByInstrumentAndTimeRequest {
	_, end := lastTradesRange()
	return &LastTradesByInstrumentAndTimeRequest{InstrumentName: marketDataPerpetual, EndTime: end, Count: 2}
}

// testLastPerpetualTradesByTime are the mock last trades of marketDataPerpetual in the trade range
func testLastPerpetualTradesByTime() *LastTradesResponse {
	return &LastTradesResponse{
		Trades: []PublicTrade{
			{
				TradeID:           "452874219",
				TradeSequence:     306495158,
				InstrumentName:    "BTC-PERPETUAL",
				Timestamp:         marketDataMilli(1792108799309),
				StarbaseTimestamp: marketDataNano(1792108799309626600),
				StarbaseMatchID:   236639273338548220,
				Direction:         "buy",
				Price:             82165,
				Amount:            200,
				Contracts:         20,
				MarkPrice:         82158.38,
				IndexPrice:        82132.5,
			},
			{
				TradeID:           "452874218",
				TradeSequence:     306495157,
				InstrumentName:    "BTC-PERPETUAL",
				Timestamp:         marketDataMilli(1792108799305),
				StarbaseTimestamp: marketDataNano(1792108799305114400),
				StarbaseMatchID:   236639273321771000,
				Direction:         "buy",
				Price:             82159,
				Amount:            10,
				Contracts:         1,
				MarkPrice:         82158.38,
				IndexPrice:        82132.5,
			},
		},
		HasMore: true,
	}
}

func TestGetLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	start, end := lastTradesRange()
	_, err := e.GetLastTradesByInstrumentAndTime(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastTradesByInstrumentAndTime must reject a nil request")
	_, err = e.GetLastTradesByInstrumentAndTime(t.Context(), &LastTradesByInstrumentAndTimeRequest{EndTime: end})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetLastTradesByInstrumentAndTime must reject an empty instrument name")
	_, err = e.GetLastTradesByInstrumentAndTime(t.Context(), &LastTradesByInstrumentAndTimeRequest{InstrumentName: marketDataPerpetual, EndTime: end, Sorting: "up"})
	require.ErrorIs(t, err, errInvalidLastTradesSorting, "GetLastTradesByInstrumentAndTime must reject a sorting Deribit does not document")
	_, err = e.GetLastTradesByInstrumentAndTime(t.Context(), &LastTradesByInstrumentAndTimeRequest{InstrumentName: marketDataPerpetual})
	require.ErrorIs(t, err, errLastTradesTimeRangeRequired, "GetLastTradesByInstrumentAndTime must reject a request without a start or end time")
	_, err = e.GetLastTradesByInstrumentAndTime(t.Context(), &LastTradesByInstrumentAndTimeRequest{InstrumentName: marketDataPerpetual, StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLastTradesByInstrumentAndTime must reject a time range starting after it ends")

	result, err := e.GetLastTradesByInstrumentAndTime(t.Context(), testLastTradesByInstrumentAndTimeRequest())
	require.NoError(t, err, "GetLastTradesByInstrumentAndTime must not error")
	if mockTests {
		assert.Equal(t, testLastPerpetualTradesByTime(), result, "GetLastTradesByInstrumentAndTime should decode every field")
	} else {
		assert.Len(t, result.Trades, 2, "GetLastTradesByInstrumentAndTime should return the trades requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLastTradesByInstrumentAndTime(t.Context(), testLastTradesByInstrumentAndTimeRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastTradesByInstrumentAndTime should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLastTradesByInstrumentAndTime(t.Context(), testLastTradesByInstrumentAndTimeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastTradesByInstrumentAndTime should return the request failure")
}

func TestValidateLastTradesSorting(t *testing.T) {
	t.Parallel()
	for _, sorting := range []string{"", "asc", "desc", "default"} {
		assert.NoErrorf(t, validateLastTradesSorting(sorting), "validateLastTradesSorting should accept sorting %q", sorting)
	}
	for _, sorting := range []string{"up", "ASC"} {
		assert.ErrorIsf(t, validateLastTradesSorting(sorting), errInvalidLastTradesSorting, "validateLastTradesSorting should reject sorting %q", sorting)
	}
}

func TestValidateLastTradesTimeRange(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(mockLastTradesRangeStart), time.UnixMilli(mockLastTradesRangeEnd)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		err        error
	}{
		{name: "an unset range"},
		{name: "a start alone", start: start},
		{name: "an end alone", end: end},
		{name: "a range", start: start, end: end},
		{name: "an instant", start: start, end: start},
		{name: "a range starting after it ends", start: end, end: start, err: common.ErrStartAfterEnd},
	} {
		assert.ErrorIsf(t, validateLastTradesTimeRange(tc.start, tc.end), tc.err, "validateLastTradesTimeRange should check %s", tc.name)
	}
}

func TestValidateLastTradesRequiredTimeRange(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(mockLastTradesRangeStart), time.UnixMilli(mockLastTradesRangeEnd)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		err        error
	}{
		{name: "an unset range", err: errLastTradesTimeRangeRequired},
		{name: "a start alone", start: start},
		{name: "an end alone", end: end},
		{name: "a range", start: start, end: end},
		{name: "a range starting after it ends", start: end, end: start, err: common.ErrStartAfterEnd},
	} {
		assert.ErrorIsf(t, validateLastTradesRequiredTimeRange(tc.start, tc.end), tc.err, "validateLastTradesRequiredTimeRange should check %s", tc.name)
	}
}

func TestMarketDataUnixMilli(t *testing.T) {
	t.Parallel()
	assert.Zero(t, marketDataUnixMilli(time.Time{}), "marketDataUnixMilli should return zero for a zero time")
	assert.Equal(t, int64(mockLastTradesRangeStart), marketDataUnixMilli(time.UnixMilli(mockLastTradesRangeStart)), "marketDataUnixMilli should return the time in Unix milliseconds")
}

// testMarkPriceRequest returns the mock mark price range of mockMarketDataOption, or a listed option's last minute when
// testing live, as Deribit keeps about 5 minutes of mark price history
func testMarkPriceRequest(t *testing.T) *InstrumentTimeRangeRequest {
	t.Helper()
	if mockTests {
		return &InstrumentTimeRangeRequest{InstrumentName: mockMarketDataOption, StartTime: time.UnixMilli(mockMarkPriceStart), EndTime: time.UnixMilli(mockMarkPriceEnd)}
	}
	end := time.Now()
	return &InstrumentTimeRangeRequest{InstrumentName: marketDataOption(t), StartTime: end.Add(-time.Minute), EndTime: end}
}

// testMarkPrices are the mock mark prices of mockMarketDataOption
func testMarkPrices() []MarkPricePoint {
	return []MarkPricePoint{
		{Timestamp: marketDataMilli(1792109784500), MarkPrice: 0.01918451202747332},
		{Timestamp: marketDataMilli(1792109785511), MarkPrice: 0.019204651518477292},
		{Timestamp: marketDataMilli(1792109787796), MarkPrice: 0.019204650590899233},
		{Timestamp: marketDataMilli(1792109787898), MarkPrice: 0.019215704655928683},
		{Timestamp: marketDataMilli(1792109788532), MarkPrice: 0.01921425781947812},
	}
}

func TestGetMarkPriceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarkPriceHistory must reject a nil request")
	_, err = e.GetMarkPriceHistory(t.Context(), &InstrumentTimeRangeRequest{InstrumentName: mockMarketDataOption, StartTime: time.UnixMilli(mockMarkPriceEnd), EndTime: time.UnixMilli(mockMarkPriceStart)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarkPriceHistory must reject a time range starting after it ends")

	arg := testMarkPriceRequest(t)
	result, err := e.GetMarkPriceHistory(t.Context(), arg)
	require.NoError(t, err, "GetMarkPriceHistory must not error")
	if mockTests {
		assert.Equal(t, testMarkPrices(), result, "GetMarkPriceHistory should decode every point")
	} else {
		// Only the options Deribit's volatility index is calculated from have a history, which a listed option may not
		// be
		for i := range result {
			assert.WithinRangef(t, result[i].Timestamp.Time(), arg.StartTime, arg.EndTime, "GetMarkPriceHistory should return mark price %d within the range", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetMarkPriceHistory(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMarkPriceHistory should return the request failure")
}

func TestMarkPricePointUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result MarkPricePoint
	require.NoError(t, json.Unmarshal([]byte(`[1608142381229,0.5165791606037885]`), &result), "Unmarshal must not error")
	assert.Equal(t, MarkPricePoint{Timestamp: marketDataMilli(1608142381229), MarkPrice: 0.5165791606037885}, result, "UnmarshalJSON should decode the timestamp and mark price")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[]`), &result), errInvalidTuple, "UnmarshalJSON should reject an empty tuple")
	assert.Error(t, json.Unmarshal([]byte(`[true,0.5]`), &result), "UnmarshalJSON should reject a timestamp that is not a number")
}

// testPerpetualOrderBook is the mock order book of marketDataPerpetual, 5 levels deep
func testPerpetualOrderBook() *OrderBookResponse {
	return &OrderBookResponse{
		InstrumentName: "BTC-PERPETUAL",
		Timestamp:      marketDataMilli(1792109800150),
		State:          "open",
		Stats: TickerStats{
			High:           83000,
			Low:            79709.5,
			PriceChange:    0.6593,
			Volume:         26599.40035538,
			VolumeUSD:      2161957390,
			VolumeNotional: 2161957390,
		},
		BestBidPrice:           82302.5,
		BestBidAmount:          17840,
		BestAskPrice:           82303,
		BestAskAmount:          20010,
		LastPrice:              82300,
		MarkPrice:              82293.77,
		IndexPrice:             82278.49,
		MinPrice:               81059,
		MaxPrice:               83528.5,
		OpenInterest:           862171840,
		EstimatedDeliveryPrice: 82278.49,
		SettlementPrice:        81445.04,
		Funding8Hour:           4.146e-05,
		InterestValue:          0.025669184954054635,
		ChangeID:               178649040260,
		Bids: []OrderBookLevel{
			{Price: 82302.5, Amount: 17840},
			{Price: 82302, Amount: 10},
			{Price: 82301.5, Amount: 500},
			{Price: 82301, Amount: 3600},
			{Price: 82300.5, Amount: 36830},
		},
		Asks: []OrderBookLevel{
			{Price: 82303, Amount: 20010},
			{Price: 82303.5, Amount: 36760},
			{Price: 82304.5, Amount: 10},
			{Price: 82305, Amount: 18510},
			{Price: 82305.5, Amount: 43520},
		},
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderBook must reject a nil request")
	_, err = e.GetOrderBook(t.Context(), &OrderBookRequest{Depth: 5})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetOrderBook must reject an empty instrument name")

	arg := &OrderBookRequest{InstrumentName: marketDataPerpetual, Depth: 5}
	result, err := e.GetOrderBook(t.Context(), arg)
	require.NoError(t, err, "GetOrderBook must not error")
	if mockTests {
		assert.Equal(t, testPerpetualOrderBook(), result, "GetOrderBook should decode every field")
	} else {
		assert.Equal(t, marketDataPerpetual, result.InstrumentName, "GetOrderBook should return the instrument requested")
		assert.NotEmpty(t, result.Bids, "GetOrderBook should return the book's bids")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetOrderBook(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderBook should reject a null result")
	_, err = newUnavailableServerExchange(t).GetOrderBook(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderBook should return the request failure")
}

// testPerpetualOrderBookByID is the mock order book of marketDataPerpetualID, 1 level deep
func testPerpetualOrderBookByID() *OrderBookResponse {
	return &OrderBookResponse{
		InstrumentName: "BTC-PERPETUAL",
		Timestamp:      marketDataMilli(1792109800918),
		State:          "open",
		Stats: TickerStats{
			High:           83000,
			Low:            79709.5,
			PriceChange:    0.6691,
			Volume:         26599.96537297,
			VolumeUSD:      2162003890,
			VolumeNotional: 2162003890,
		},
		BestBidPrice:           82302,
		BestBidAmount:          2940,
		BestAskPrice:           82302.5,
		BestAskAmount:          22950,
		LastPrice:              82302,
		MarkPrice:              82295.07,
		IndexPrice:             82279.45,
		MinPrice:               81060.5,
		MaxPrice:               83529.5,
		OpenInterest:           862171490,
		EstimatedDeliveryPrice: 82279.45,
		SettlementPrice:        81445.04,
		Funding8Hour:           4.146e-05,
		InterestValue:          0.025669184954054635,
		ChangeID:               178649041476,
		Bids: []OrderBookLevel{
			{Price: 82302, Amount: 2940},
		},
		Asks: []OrderBookLevel{
			{Price: 82302.5, Amount: 22950},
		},
	}
}

func TestGetOrderBookByInstrumentID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBookByInstrumentID(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderBookByInstrumentID must reject a nil request")
	_, err = e.GetOrderBookByInstrumentID(t.Context(), &OrderBookByInstrumentIDRequest{Depth: 1})
	require.ErrorIs(t, err, errInstrumentIDRequired, "GetOrderBookByInstrumentID must reject an empty instrument ID")

	arg := &OrderBookByInstrumentIDRequest{InstrumentID: marketDataPerpetualID, Depth: 1}
	result, err := e.GetOrderBookByInstrumentID(t.Context(), arg)
	require.NoError(t, err, "GetOrderBookByInstrumentID must not error")
	if mockTests {
		assert.Equal(t, testPerpetualOrderBookByID(), result, "GetOrderBookByInstrumentID should decode every field")
	} else {
		assert.Equal(t, marketDataPerpetual, result.InstrumentName, "GetOrderBookByInstrumentID should return the instrument requested")
		assert.Len(t, result.Asks, 1, "GetOrderBookByInstrumentID should return the depth requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetOrderBookByInstrumentID(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderBookByInstrumentID should reject a null result")
	_, err = newUnavailableServerExchange(t).GetOrderBookByInstrumentID(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderBookByInstrumentID should return the request failure")
}

func TestOrderBookLevelUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result []OrderBookLevel
	require.NoError(t, json.Unmarshal([]byte(`[[3955.75,30.0],[3940.75,102020.0]]`), &result), "Unmarshal must not error")
	assert.Equal(t, []OrderBookLevel{{Price: 3955.75, Amount: 30}, {Price: 3940.75, Amount: 102020}}, result, "UnmarshalJSON should decode each price and amount")
	var level OrderBookLevel
	assert.ErrorIs(t, json.Unmarshal([]byte(`["new",3955.75,30.0]`), &level), errInvalidTuple, "UnmarshalJSON should reject a subscription change, which has an action too")
	assert.Error(t, json.Unmarshal([]byte(`[3955.75,"30"]`), &level), "UnmarshalJSON should reject an amount that is not a number")
}

func TestGetSupportedIndexNames(t *testing.T) {
	t.Parallel()
	_, err := e.GetSupportedIndexNames(t.Context(), "future")
	require.ErrorIs(t, err, errInvalidIndexType, "GetSupportedIndexNames must reject a type Deribit does not document")

	result, err := e.GetSupportedIndexNames(t.Context(), IndexTypeSpot)
	require.NoError(t, err, "GetSupportedIndexNames must not error")
	if mockTests {
		assert.Equal(t, []string{"aave_usdc", "ada_usdc", "aero_usdc", "algo_usdc", "apt_usdc"}, result, "GetSupportedIndexNames should decode every name")
	} else {
		assert.Contains(t, result, "btc_usdc", "GetSupportedIndexNames should return the btc_usdc spot index")
	}

	_, err = newUnavailableServerExchange(t).GetSupportedIndexNames(t.Context(), IndexTypeSpot)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSupportedIndexNames should return the request failure")
}

// testTradeVolumes are the mock extended trade volumes, of the first currencies alphabetically
func testTradeVolumes() []TradeVolume {
	return []TradeVolume{
		{Currency: currency.AAVE, CurrencyPair: "aave_usd"},
		{Currency: currency.NewCode("ADA"), CurrencyPair: "ada_usd"},
		{Currency: currency.NewCode("AERO"), CurrencyPair: "aero_usd"},
		{Currency: currency.NewCode("ALGO"), CurrencyPair: "algo_usd"},
		{Currency: currency.NewCode("APT"), CurrencyPair: "apt_usd"},
	}
}

func TestGetTradeVolumes(t *testing.T) {
	t.Parallel()
	result, err := e.GetTradeVolumes(t.Context(), true)
	require.NoError(t, err, "GetTradeVolumes must not error")
	if mockTests {
		assert.Equal(t, testTradeVolumes(), result, "GetTradeVolumes should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetTradeVolumes should return each currency's volumes")
	}

	// The recorded currencies traded nothing, so a busy currency is served as Deribit sent it
	result, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":[{"calls_volume":13836.1,"calls_volume_30d":378436.4,"calls_volume_7d":91834.2,"currency":"BTC","currency_pair":"btc_usd","futures_volume":27450.21422905,"futures_volume_30d":919845.41557178,"futures_volume_7d":215379.62855405,"puts_volume":15084.5,"puts_volume_30d":367294.8,"puts_volume_7d":86913.9,"spot_volume":1.1,"spot_volume_30d":30.5,"spot_volume_7d":6.4}]}`).GetTradeVolumes(t.Context(), true)
	require.NoError(t, err, "GetTradeVolumes must not error for a busy currency")
	exp := []TradeVolume{{
		Currency:           currency.BTC,
		CurrencyPair:       "btc_usd",
		CallsVolume:        13836.1,
		PutsVolume:         15084.5,
		FuturesVolume:      27450.21422905,
		SpotVolume:         1.1,
		CallsVolume7Day:    91834.2,
		PutsVolume7Day:     86913.9,
		FuturesVolume7Day:  215379.62855405,
		SpotVolume7Day:     6.4,
		CallsVolume30Day:   378436.4,
		PutsVolume30Day:    367294.8,
		FuturesVolume30Day: 919845.41557178,
		SpotVolume30Day:    30.5,
	}}
	assert.Equal(t, exp, result, "GetTradeVolumes should decode every volume")
	_, err = newUnavailableServerExchange(t).GetTradeVolumes(t.Context(), false)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTradeVolumes should return the request failure")
}

// testTradingViewChartDataRequest requests marketDataPerpetual's hourly candles opening in the 4 hours from
// 2026-10-15T00:00:00Z
func testTradingViewChartDataRequest() *TradingViewChartDataRequest {
	return &TradingViewChartDataRequest{InstrumentName: marketDataPerpetual, StartTime: time.UnixMilli(marketDataRangeStart), EndTime: time.UnixMilli(marketDataRangeEnd), Resolution: kline.OneHour}
}

// testPerpetualCandles are the mock hourly candles of marketDataPerpetual
func testPerpetualCandles() *TradingViewChartDataResponse {
	return &TradingViewChartDataResponse{
		Status: "ok",
		Ticks:  []types.Time{marketDataMilli(1792022400000), marketDataMilli(1792026000000), marketDataMilli(1792029600000), marketDataMilli(1792033200000), marketDataMilli(1792036800000)},
		Open:   []float64{81600, 81589, 81575.5, 81452.5, 80833.5},
		High:   []float64{81763, 81861.5, 81731.5, 81466.5, 81674},
		Low:    []float64{81312, 81422, 80150, 80645.5, 80756.5},
		Close:  []float64{81588, 81579, 81452.5, 80834, 81520},
		Volume: []float64{962.20147922, 1051.02155566, 2529.7376939, 1355.97310084, 1107.5785469},
		Cost:   []float64{78526730, 85850140, 205028530, 109806350, 89991390},
	}
}

func TestGetTradingViewChartData(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradingViewChartData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradingViewChartData must reject a nil request")
	_, err = e.GetTradingViewChartData(t.Context(), &TradingViewChartDataRequest{StartTime: time.UnixMilli(marketDataRangeStart), EndTime: time.UnixMilli(marketDataRangeEnd), Resolution: kline.OneHour})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetTradingViewChartData must reject an empty instrument name")
	_, err = e.GetTradingViewChartData(t.Context(), &TradingViewChartDataRequest{InstrumentName: marketDataPerpetual, Resolution: kline.OneHour})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetTradingViewChartData must reject an unset time range")
	_, err = e.GetTradingViewChartData(t.Context(), &TradingViewChartDataRequest{InstrumentName: marketDataPerpetual, StartTime: time.UnixMilli(marketDataRangeStart), EndTime: time.UnixMilli(marketDataRangeEnd), Resolution: kline.FourHour})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetTradingViewChartData must reject a resolution Deribit does not chart")

	result, err := e.GetTradingViewChartData(t.Context(), testTradingViewChartDataRequest())
	require.NoError(t, err, "GetTradingViewChartData must not error")
	if mockTests {
		assert.Equal(t, testPerpetualCandles(), result, "GetTradingViewChartData should decode every field")
	} else {
		assert.Equal(t, "ok", result.Status, "GetTradingViewChartData should return candles")
		assert.Len(t, result.Ticks, 5, "GetTradingViewChartData should return each hour's candle")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "GetTradingViewChartData should send a GET request, as Deribit answers a POST with Method not found")
		assert.Equal(t, "/api/v2/public/get_tradingview_chart_data", r.URL.Path, "GetTradingViewChartData should request the method's path")
		assert.Equal(t, "end_timestamp=1792036800000&instrument_name=BTC-PERPETUAL&resolution=60&start_timestamp=1792022400000", r.URL.RawQuery, "GetTradingViewChartData should send its params in the query string")
		_, err := w.Write([]byte(`{"jsonrpc":"2.0","result":{"volume":[],"ticks":[],"status":"no_data","open":[],"low":[],"high":[],"cost":[],"close":[]}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	result, err = ex.GetTradingViewChartData(t.Context(), testTradingViewChartDataRequest())
	require.NoError(t, err, "GetTradingViewChartData must not error without candles")
	exp := &TradingViewChartDataResponse{Status: "no_data", Ticks: []types.Time{}, Open: []float64{}, High: []float64{}, Low: []float64{}, Close: []float64{}, Volume: []float64{}, Cost: []float64{}}
	assert.Equal(t, exp, result, "GetTradingViewChartData should decode a range without candles")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetTradingViewChartData(t.Context(), testTradingViewChartDataRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTradingViewChartData should reject a null result")
	_, err = newUnavailableServerExchange(t).GetTradingViewChartData(t.Context(), testTradingViewChartDataRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTradingViewChartData should return the request failure")
}

func TestTradingViewResolution(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		exp      string
	}{
		{interval: kline.OneMin, exp: "1"},
		{interval: kline.ThreeMin, exp: "3"},
		{interval: kline.FiveMin, exp: "5"},
		{interval: kline.TenMin, exp: "10"},
		{interval: kline.FifteenMin, exp: "15"},
		{interval: kline.ThirtyMin, exp: "30"},
		{interval: kline.OneHour, exp: "60"},
		{interval: kline.TwoHour, exp: "120"},
		{interval: kline.ThreeHour, exp: "180"},
		{interval: kline.SixHour, exp: "360"},
		{interval: kline.TwelveHour, exp: "720"},
		{interval: kline.OneDay, exp: "1D"},
	} {
		resolution, err := e.tradingViewResolution(tc.interval)
		require.NoErrorf(t, err, "tradingViewResolution must not error for interval %s", tc.interval)
		assert.Equalf(t, tc.exp, resolution, "tradingViewResolution should name interval %s", tc.interval)
	}
	for _, interval := range []kline.Interval{kline.HundredMilliseconds, kline.Raw, kline.FourHour, kline.OneWeek} {
		_, err := e.tradingViewResolution(interval)
		assert.ErrorIsf(t, err, kline.ErrUnsupportedInterval, "tradingViewResolution should reject interval %s", interval)
	}
}

// testVolatilityIndexDataRequest requests BTC's hourly volatility index candles of the 4 hours from
// 2026-10-15T00:00:00Z
func testVolatilityIndexDataRequest() *VolatilityIndexDataRequest {
	return &VolatilityIndexDataRequest{Currency: currency.BTC, StartTime: time.UnixMilli(marketDataRangeStart), EndTime: time.UnixMilli(marketDataRangeEnd), Resolution: kline.OneHour}
}

// testVolatilityIndexCandles are the mock hourly volatility index candles of BTC
func testVolatilityIndexCandles() *VolatilityIndexDataResponse {
	return &VolatilityIndexDataResponse{
		Data: []VolatilityIndexCandle{
			{
				Timestamp: marketDataMilli(1792022400000),
				Open:      39.12,
				High:      39.14,
				Low:       39.02,
				Close:     39.1,
			},
			{
				Timestamp: marketDataMilli(1792026000000),
				Open:      39.1,
				High:      39.1,
				Low:       38.63,
				Close:     38.89,
			},
			{
				Timestamp: marketDataMilli(1792029600000),
				Open:      38.89,
				High:      39.84,
				Low:       38.73,
				Close:     39,
			},
			{
				Timestamp: marketDataMilli(1792033200000),
				Open:      39,
				High:      39,
				Low:       38.66,
				Close:     38.73,
			},
			{
				Timestamp: marketDataMilli(1792036800000),
				Open:      38.73,
				High:      38.76,
				Low:       38.52,
				Close:     38.52,
			},
		},
	}
}

func TestGetVolatilityIndexData(t *testing.T) {
	t.Parallel()
	_, err := e.GetVolatilityIndexData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetVolatilityIndexData must reject a nil request")
	_, err = e.GetVolatilityIndexData(t.Context(), &VolatilityIndexDataRequest{StartTime: time.UnixMilli(marketDataRangeStart), EndTime: time.UnixMilli(marketDataRangeEnd), Resolution: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetVolatilityIndexData must reject an empty currency")
	_, err = e.GetVolatilityIndexData(t.Context(), &VolatilityIndexDataRequest{Currency: currency.BTC, Resolution: kline.OneHour})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetVolatilityIndexData must reject an unset time range")
	_, err = e.GetVolatilityIndexData(t.Context(), &VolatilityIndexDataRequest{Currency: currency.BTC, StartTime: time.UnixMilli(marketDataRangeStart), EndTime: time.UnixMilli(marketDataRangeEnd), Resolution: kline.FiveMin})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetVolatilityIndexData must reject a resolution Deribit does not support")

	result, err := e.GetVolatilityIndexData(t.Context(), testVolatilityIndexDataRequest())
	require.NoError(t, err, "GetVolatilityIndexData must not error")
	if mockTests {
		assert.Equal(t, testVolatilityIndexCandles(), result, "GetVolatilityIndexData should decode every field")
	} else {
		assert.Len(t, result.Data, 5, "GetVolatilityIndexData should return each hour's candle")
	}

	result, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"data":[[1792036800000,38.73,38.73,38.73,38.73]],"continuation":1792035799000}}`).GetVolatilityIndexData(t.Context(), testVolatilityIndexDataRequest())
	require.NoError(t, err, "GetVolatilityIndexData must not error for a page with more candles")
	exp := &VolatilityIndexDataResponse{Data: []VolatilityIndexCandle{{Timestamp: marketDataMilli(1792036800000), Open: 38.73, High: 38.73, Low: 38.73, Close: 38.73}}, Continuation: marketDataMilli(1792035799000)}
	assert.Equal(t, exp, result, "GetVolatilityIndexData should decode the continuation of a page with more candles")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetVolatilityIndexData(t.Context(), testVolatilityIndexDataRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetVolatilityIndexData should reject a null result")
	_, err = newUnavailableServerExchange(t).GetVolatilityIndexData(t.Context(), testVolatilityIndexDataRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetVolatilityIndexData should return the request failure")
}

func TestVolatilityIndexResolution(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		exp      string
	}{
		{interval: kline.ThousandMilliseconds, exp: "1"},
		{interval: kline.OneMin, exp: "60"},
		{interval: kline.OneHour, exp: "3600"},
		{interval: kline.TwelveHour, exp: "43200"},
		{interval: kline.OneDay, exp: "1D"},
	} {
		resolution, err := volatilityIndexResolution(tc.interval)
		require.NoErrorf(t, err, "volatilityIndexResolution must not error for interval %s", tc.interval)
		assert.Equalf(t, tc.exp, resolution, "volatilityIndexResolution should name interval %s", tc.interval)
	}
	for _, interval := range []kline.Interval{kline.Raw, kline.FiveMin, kline.FourHour} {
		_, err := volatilityIndexResolution(interval)
		assert.ErrorIsf(t, err, kline.ErrUnsupportedInterval, "volatilityIndexResolution should reject interval %s", interval)
	}
}

func TestVolatilityIndexCandleUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result VolatilityIndexCandle
	require.NoError(t, json.Unmarshal([]byte(`[1598019300000,0.210084879,0.212860821,0.210084879,0.212860821]`), &result), "Unmarshal must not error")
	exp := VolatilityIndexCandle{Timestamp: marketDataMilli(1598019300000), Open: 0.210084879, High: 0.212860821, Low: 0.210084879, Close: 0.212860821}
	assert.Equal(t, exp, result, "UnmarshalJSON should decode the timestamp and prices")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[1598019300000,0.21,0.21,0.21]`), &result), errInvalidTuple, "UnmarshalJSON should reject a candle without a close")
}

// testInstrumentTickers are the mock tickers of a perpetual, an option, Coinbase-routed spot and a future combo
func testInstrumentTickers() map[string]*TickerResponse {
	return map[string]*TickerResponse{
		marketDataPerpetual: {
			InstrumentName: "BTC-PERPETUAL",
			Timestamp:      marketDataMilli(1792109806455),
			State:          "open",
			Stats: TickerStats{
				High:           83000,
				Low:            79709.5,
				PriceChange:    0.6733,
				Volume:         26600.15102531,
				VolumeUSD:      2162019170,
				VolumeNotional: 2162019170,
			},
			BestBidPrice:           82312.5,
			BestBidAmount:          96920,
			BestAskPrice:           82313,
			BestAskAmount:          10,
			LastPrice:              82306.5,
			MarkPrice:              82302.33,
			IndexPrice:             82285.56,
			MinPrice:               81067.5,
			MaxPrice:               83537,
			OpenInterest:           862179360,
			EstimatedDeliveryPrice: 82285.56,
			SettlementPrice:        81445.04,
			Funding8Hour:           4.146e-05,
			InterestValue:          0.025669184954054635,
		},
		mockMarketDataOption: {
			InstrumentName: "BTC-27NOV26-90000-C",
			Timestamp:      marketDataMilli(1792109806659),
			State:          "open",
			Stats: TickerStats{
				High:        0.0205,
				Low:         0.015,
				PriceChange: 11.4286,
				Volume:      329.4,
				VolumeUSD:   501564.61,
			},
			BestBidPrice:           0.019,
			BestBidAmount:          14.6,
			BestAskPrice:           0.0195,
			BestAskAmount:          7.4,
			LastPrice:              0.0195,
			MarkPrice:              0.0192,
			IndexPrice:             82290.66,
			MinPrice:               0.0043,
			MaxPrice:               0.0475,
			OpenInterest:           3141.1,
			EstimatedDeliveryPrice: 82290.66,
			SettlementPrice:        0.0182148,
			Greeks:                 &Greeks{Delta: 0.27071, Gamma: 3e-05, Vega: 93.38987, Theta: -39.95688, Rho: 24.15561},
			BidImpliedVolatility:   36.02,
			AskImpliedVolatility:   36.46,
			MarkImpliedVolatility:  36.21,
			UnderlyingIndex:        "BTC-27NOV26",
			UnderlyingPrice:        82839.12,
		},
		marketDataSpot: {
			InstrumentName: "BTC_USDC",
			Timestamp:      marketDataMilli(1792109807938),
			State:          "open",
			Stats:          TickerStats{High: 82555.08, Low: 79676.02, PriceChange: 0.907, Volume: 6939.10739095},
			BestBidPrice:   82295.99,
			BestBidAmount:  0.15916902,
			BestAskPrice:   82296,
			BestAskAmount:  0.0263751,
			LastPrice:      82279.83,
			MarkPrice:      82290.66,
			IndexPrice:     82290.66,
			MinPrice:       80644.84,
			MaxPrice:       83936.48,
		},
		mockMarketDataFutureCombo: {
			InstrumentName:  "BTC-FS-27NOV26_PERP",
			Timestamp:       marketDataMilli(1792109808447),
			State:           "open",
			Stats:           TickerStats{High: 534.5, Low: 513, VolumeUSD: 290, VolumeNotional: 290},
			BestBidPrice:    534.5,
			BestBidAmount:   225000,
			BestAskPrice:    535,
			BestAskAmount:   50000,
			LastPrice:       534.5,
			MarkPrice:       537.09,
			IndexPrice:      82296.37,
			MinPrice:        190,
			MaxPrice:        884.5,
			SettlementPrice: 531,
			ComboState:      "active",
			ImpliedBid:      530,
			ImpliedAsk:      535.5,
		},
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetTicker must reject an empty instrument name")

	for instrument, exp := range testInstrumentTickers() {
		switch instrument {
		case mockMarketDataOption:
			instrument = marketDataOption(t)
		case mockMarketDataFutureCombo:
			instrument = marketDataFutureCombo(t)
		}
		result, err := e.GetTicker(t.Context(), instrument)
		require.NoErrorf(t, err, "GetTicker must not error for %s", instrument)
		if mockTests {
			assert.Equalf(t, exp, result, "GetTicker should decode every field of %s", instrument)
			continue
		}
		assert.Equalf(t, instrument, result.InstrumentName, "GetTicker should return %s", instrument)
		assert.Equalf(t, exp.Greeks != nil, result.Greeks != nil, "GetTicker should return greeks for %s only if it is an option", instrument)
		assert.Equalf(t, exp.ComboState, result.ComboState, "GetTicker should return a combo state for %s only if it is a combo", instrument)
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetTicker(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTicker should reject a null result")
	_, err = newUnavailableServerExchange(t).GetTicker(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTicker should return the request failure")
}

func TestValidateMarketDataKind(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"", KindFuture, KindOption} {
		assert.NoErrorf(t, validateMarketDataKind(kind, KindFuture, KindOption), "validateMarketDataKind should accept kind %q", kind)
	}
	for _, kind := range []string{KindSpot, "Future"} {
		assert.ErrorIsf(t, validateMarketDataKind(kind, KindFuture, KindOption), errInvalidKind, "validateMarketDataKind should reject kind %q", kind)
	}
}
