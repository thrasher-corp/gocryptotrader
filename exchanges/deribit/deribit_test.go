package deribit

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey    = ""
	apiSecret = ""

	canManipulateRealOrders   = false
	canManipulateAPIEndpoints = false
	btcPerpInstrument         = "BTC-PERPETUAL"
	useTestNet                = false
)

var (
	e                                                                     *Exchange
	optionsTradablePair, optionComboTradablePair, futureComboTradablePair currency.Pair
	spotTradablePair                                                      = currency.NewPairWithDelimiter(currencyBTC, "USDC", "_")
	futuresTradablePair                                                   = currency.NewPairWithDelimiter(currencyBTC, perpString, "-")
	assetTypeToPairsMap                                                   map[asset.Item]currency.Pair
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Deribit Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	if useTestNet {
		deribitWebsocketAddress = "wss://test.deribit.com/ws" + deribitAPIVersion
		if err := e.Websocket.SetWebsocketURL(deribitWebsocketAddress, false, true); err != nil {
			log.Fatalf("Deribit SetWebsocketURL error: %s", err)
		}
		for k, v := range e.API.Endpoints.GetURLMap() {
			v = strings.Replace(v, "www.deribit.com", "test.deribit.com", 1)
			if err := e.API.Endpoints.SetRunningURL(k, v); err != nil {
				log.Fatalf("Deribit SetRunningURL error: %s", err)
			}
		}
	}

	instantiateTradablePairs()
	assetTypeToPairsMap = map[asset.Item]currency.Pair{
		asset.Futures:     futuresTradablePair,
		asset.Spot:        spotTradablePair,
		asset.Options:     optionsTradablePair,
		asset.OptionCombo: optionComboTradablePair,
		asset.FutureCombo: futureComboTradablePair,
	}
	setupWs()
	os.Exit(m.Run())
}

func instantiateTradablePairs() {
	if err := e.UpdateTradablePairs(context.Background()); err != nil {
		log.Fatalf("Failed to update tradable pairs. Error: %v", err)
	}

	handleError := func(err error, msg string) {
		if err != nil {
			log.Fatalf("%s. Error: %v", msg, err)
		}
	}

	updateTradablePair := func(assetType asset.Item, tradablePair *currency.Pair) {
		if e.CurrencyPairs.IsAssetEnabled(assetType) == nil {
			pairs, err := e.GetEnabledPairs(assetType)
			handleError(err, fmt.Sprintf("Failed to get enabled pairs for asset type %v", assetType))

			if len(pairs) == 0 {
				handleError(currency.ErrCurrencyPairsEmpty, fmt.Sprintf("No enabled pairs for asset type %v", assetType))
			}

			if assetType == asset.Options {
				*tradablePair, err = e.FormatExchangeCurrency(pairs[0], assetType)
				handleError(err, "Failed to format exchange currency for options pair")
			} else {
				*tradablePair = pairs[0]
			}
		}
	}
	updateTradablePair(asset.Options, &optionsTradablePair)
	updateTradablePair(asset.OptionCombo, &optionComboTradablePair)
	updateTradablePair(asset.FutureCombo, &futureComboTradablePair)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.Pair{}, asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	for assetType, cp := range assetTypeToPairsMap {
		result, err := e.UpdateTicker(t.Context(), cp, assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	for assetType, cp := range assetTypeToPairsMap {
		result, err := e.UpdateOrderbook(t.Context(), cp, assetType)
		require.NoErrorf(t, err, "asset type: %v", assetType)
		require.NotNil(t, result)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), futureComboTradablePair, asset.FutureCombo, time.Now().Add(-time.Minute*10), time.Now())
	require.ErrorIs(t, err, asset.ErrNotSupported)
	for assetType, cp := range map[asset.Item]currency.Pair{asset.Spot: spotTradablePair, asset.Futures: futuresTradablePair} {
		_, err = e.GetHistoricTrades(t.Context(), cp, assetType, time.Now().Add(-time.Minute*10), time.Now())
		require.NoErrorf(t, err, "asset type: %v", assetType)
	}
}

func TestFetchRecentTrades(t *testing.T) {
	t.Parallel()
	for assetType, cp := range assetTypeToPairsMap {
		result, err := e.GetRecentTrades(t.Context(), cp, assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	assetTypesToPairMap := map[asset.Item]struct {
		Pair  currency.Pair
		Error error
	}{
		asset.Futures:     {Pair: futuresTradablePair},
		asset.Spot:        {Pair: spotTradablePair},
		asset.Options:     {Pair: optionsTradablePair, Error: asset.ErrNotSupported},
		asset.FutureCombo: {Pair: futureComboTradablePair, Error: asset.ErrNotSupported},
		asset.OptionCombo: {Pair: optionComboTradablePair, Error: asset.ErrNotSupported},
	}
	for assetType, info := range assetTypesToPairMap {
		resp, err := e.GetHistoricCandles(t.Context(), info.Pair, assetType, kline.FifteenMin, start, end)
		require.ErrorIs(t, err, info.Error)
		if info.Error == nil {
			require.NotEmpty(t, resp)
		}
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start := time.Now().Add(-time.Hour * 24 * 90).Truncate(kline.OneDay.Duration()).UTC()
	end := time.Now().UTC()
	assetsToPairsMap := map[asset.Item]struct {
		Pair  currency.Pair
		Error error
	}{
		asset.Futures:     {Pair: futuresTradablePair},
		asset.Spot:        {Pair: spotTradablePair},
		asset.Options:     {Pair: optionsTradablePair, Error: asset.ErrNotSupported},
		asset.FutureCombo: {Pair: futureComboTradablePair, Error: asset.ErrNotSupported},
		asset.OptionCombo: {Pair: optionComboTradablePair, Error: asset.ErrNotSupported},
	}
	for assetType, instance := range assetsToPairsMap {
		resp, err := e.GetHistoricCandlesExtended(t.Context(), instance.Pair, assetType, kline.OneDay, start, end)
		require.ErrorIs(t, err, instance.Error)
		if instance.Error == nil {
			require.NotEmpty(t, resp)
		}
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	assetToPairStringMap := map[asset.Item]currency.Pair{
		asset.Options:     optionsTradablePair,
		asset.FutureCombo: futureComboTradablePair,
		asset.Futures:     futuresTradablePair,
	}
	var result *order.SubmitResponse
	var err error
	var info *InstrumentData
	for assetType, cp := range assetToPairStringMap {
		info, err = e.GetInstrument(t.Context(), formatPairString(assetType, cp))
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)

		result, err = e.SubmitOrder(
			t.Context(),
			&order.Submit{
				Exchange:  e.Name,
				Price:     10,
				Amount:    info.ContractSize * 3,
				Type:      order.Limit,
				AssetType: assetType,
				Side:      order.Buy,
				Pair:      cp,
			},
		)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetMarkPriceHistory(t *testing.T) {
	t.Parallel()
	var resp []MarkPriceHistory
	err := json.Unmarshal([]byte(`[[1608142381229,0.5165791606037885],[1608142380231,0.5165737855432504],[1608142379227,0.5165768236356326]]`), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 3)

	_, err = e.GetMarkPriceHistory(t.Context(), "", time.Now().Add(-5*time.Minute), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result []MarkPriceHistory
	for _, ps := range []string{
		optionPairToString(optionsTradablePair),
		spotTradablePair.String(),
		btcPerpInstrument,
		futureComboPairToString(futureComboTradablePair),
	} {
		result, err = e.GetMarkPriceHistory(t.Context(), ps, time.Now().Add(-5*time.Minute), time.Now())
		require.NoErrorf(t, err, "expected nil, got %v for pair %s", err, ps)
		require.NotNilf(t, result, "expected result not to be nil for pair %s", ps)
	}
}

func TestWSRetrieveMarkPriceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveMarkPriceHistory(t.Context(), "", time.Now().Add(-4*time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result []MarkPriceHistory
	for _, ps := range []string{
		optionPairToString(optionsTradablePair),
		spotTradablePair.String(),
		btcPerpInstrument,
		futureComboPairToString(futureComboTradablePair),
	} {
		result, err = e.WSRetrieveMarkPriceHistory(t.Context(), ps, time.Now().Add(-4*time.Hour), time.Now())
		require.NoErrorf(t, err, "expected %v, got %v currency pair %v", nil, err, ps)
		require.NotNilf(t, result, "expected value not to be nil for pair: %v", ps)
	}
}

func TestGetBookSummaryByCurrency(t *testing.T) {
	t.Parallel()
	var response BookSummaryData
	err := json.Unmarshal([]byte(`{	"volume_usd": 0,	"volume": 0,	"quote_currency": "USD",
	"price_change": -11.1896349,	"open_interest": 0,	"mid_price": null,	"mark_price": 3579.73,	"low": null,
	"last": null,	"instrument_name": "BTC-22FEB19",	"high": null,	"estimated_delivery_price": 3579.73,	"creation_timestamp": 1550230036440,
	"bid_price": null,	"base_currency": "BTC",	"ask_price": null}`), &response)
	require.NoError(t, err)
	_, err = e.GetBookSummaryByCurrency(t.Context(), currency.EMPTYCODE, "future")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetBookSummaryByCurrency(t.Context(), currency.BTC, "option")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveBookBySummary(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveBookBySummary(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	result, err := e.WSRetrieveBookBySummary(t.Context(), currency.SOL, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetBookSummaryByInstrument(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	for _, ps := range []string{
		btcPerpInstrument,
		spotTradablePair.String(),
		futureComboPairToString(futureComboTradablePair),
		optionPairToString(optionsTradablePair),
		optionComboPairToString(optionComboTradablePair),
	} {
		t.Run(ps, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetBookSummaryByInstrument(t.Context(), ps)
			require.NoError(t, err, "GetBookSummaryByInstrument must not error")
			require.NotNil(t, result, "result must not be nil")
		})
	}
}

func TestWSRetrieveBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveBookSummaryByInstrument(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)
	var result []BookSummaryData
	for _, ps := range []string{
		btcPerpInstrument,
		spotTradablePair.String(),
		futureComboPairToString(futureComboTradablePair),
		optionPairToString(optionsTradablePair),
		optionComboPairToString(optionComboTradablePair),
	} {
		result, err = e.WSRetrieveBookSummaryByInstrument(t.Context(), ps)
		require.NoErrorf(t, err, "expected nil, got %v for pair %s", err, ps)
		require.NotNilf(t, result, "expected result not to be nil for pair %s", ps)
	}
}

func TestGetContractSize(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractSize(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)
	result, err := e.GetContractSize(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveContractSize(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveContractSize(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := e.WSRetrieveContractSize(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetCurrencies(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.WSRetrieveCurrencies(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDeliveryPrices(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryPrices(t.Context(), "", 0, 5)
	require.ErrorIs(t, err, errUnsupportedIndexName)

	result, err := e.GetDeliveryPrices(t.Context(), "btc_usd", 0, 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveDeliveryPrices(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveDeliveryPrices(t.Context(), "", 0, 5)
	require.ErrorIs(t, err, errUnsupportedIndexName)

	result, err := e.WSRetrieveDeliveryPrices(t.Context(), "btc_usd", 0, 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingChartData(t *testing.T) {
	t.Parallel()
	// only for perpetual instruments
	_, err := e.GetFundingChartData(t.Context(), "", "8h")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := e.GetFundingChartData(t.Context(), btcPerpInstrument, "8h")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveFundingChartData(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveFundingChartData(t.Context(), "", "8h")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := e.WSRetrieveFundingChartData(t.Context(), btcPerpInstrument, "8h")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingRateHistory(t.Context(), "", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := e.GetFundingRateHistory(t.Context(), btcPerpInstrument, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveFundingRateHistory(t.Context(), "", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)
	result, err := e.WSRetrieveFundingRateHistory(t.Context(), btcPerpInstrument, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingRateValue(t.Context(), "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = e.GetFundingRateValue(t.Context(), btcPerpInstrument, time.Now(), time.Now().Add(-time.Hour*8))
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetFundingRateValue(t.Context(), btcPerpInstrument, time.Now().Add(-time.Hour*8), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveFundingRateValue(t.Context(), btcPerpInstrument, time.Now(), time.Now().Add(-time.Hour*8))
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.WSRetrieveFundingRateValue(t.Context(), btcPerpInstrument, time.Now().Add(-time.Hour*8), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHistoricalVolatilityDataUnmarshalJSON(t *testing.T) {
	t.Parallel()
	data := []byte(`[[1746532800000,33.926694663144644],[1746536400000,33.86888345738641],[1746540000000,33.87689653120242],[1746543600000,33.92229949556179],[1746547200000,33.35430439982866],[1746550800000,33.405720857822644],[1746554400000,33.041661194903895],[1746558000000,33.026907604467596],[1746561600000,33.147012362654635],[1746565200000,32.948314953334105],[1746568800000,32.97264616801311],[1746572400000,32.97051874896058],[1746576000000,33.94405253940284],[1746579600000,34.01745935786804],[1746583200000,34.133772136604854],[1746586800000,33.89032454069847],[1746590400000,34.008502172420556],[1746594000000,34.01444591222428],[1746597600000,34.01154352323321],[1746601200000,33.97800061398224],[1746604800000,33.980501315033024]]`)
	var targets []HistoricalVolatilityData
	err := json.Unmarshal(data, &targets)
	require.NoError(t, err)
	require.Len(t, targets, 21)
	assert.Equal(t, HistoricalVolatilityData{
		Timestamp: types.Time(time.UnixMilli(1746532800000)),
		Value:     33.926694663144644,
	}, targets[0])
}

func TestGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalVolatility(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetHistoricalVolatility(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveHistoricalVolatility(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveHistoricalVolatility(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.WSRetrieveHistoricalVolatility(t.Context(), currency.SOL)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPrice(t.Context(), "")
	require.ErrorIs(t, err, errUnsupportedIndexName)
	result, err := e.GetIndexPrice(t.Context(), "ada_usd")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveIndexPrice(t.Context(), "")
	require.ErrorIs(t, err, errUnsupportedIndexName)
	result, err := e.WSRetrieveIndexPrice(t.Context(), "ada_usd")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	result, err := e.GetIndexPriceNames(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveIndexPriceNames(t *testing.T) {
	t.Parallel()
	result, err := e.WSRetrieveIndexPriceNames(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstrumentData(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstrument(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result *InstrumentData
	for assetType, cp := range assetTypeToPairsMap {
		result, err = e.GetInstrument(t.Context(), formatPairString(assetType, cp))
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestWSRetrieveInstrumentData(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveInstrumentData(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)
	for assetType, cp := range assetTypeToPairsMap {
		t.Run(fmt.Sprintf("%s %s", assetType, cp), func(t *testing.T) {
			t.Parallel()
			result, err := e.WSRetrieveInstrumentData(request.WithVerbose(t.Context()), formatPairString(assetType, cp))
			require.NoError(t, err)
			require.NotNil(t, result, "result must not be nil")
		})
	}
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	result, err := e.GetInstruments(t.Context(), currency.EMPTYCODE, "future", false)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.GetInstruments(t.Context(), currency.BTC, "", false)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.GetInstruments(t.Context(), currency.BTC, "", true)
	require.NoError(t, err)
	for a := range result {
		require.Falsef(t, result[a].IsActive, "expected expired instrument, but got active instrument %s", result[a].InstrumentName)
	}
}

func TestWSRetrieveInstrumentsData(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveInstrumentsData(t.Context(), currency.EMPTYCODE, "", false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.WSRetrieveInstrumentsData(t.Context(), currency.BTC, "", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastSettlementsByCurrency(t.Context(), currency.EMPTYCODE, "delivery", "5", 0, time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetLastSettlementsByCurrency(t.Context(), currency.BTC, "delivery", "5", 0, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveLastSettlementsByCurrency(t.Context(), currency.EMPTYCODE, "delivery", "5", 0, time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.WSRetrieveLastSettlementsByCurrency(t.Context(), currency.BTC, "delivery", "5", 0, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveLastSettlementsByInstrument(t.Context(), "", "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := e.WSRetrieveLastSettlementsByInstrument(t.Context(), formatFuturesTradablePair(futuresTradablePair), "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastSettlementsByInstrument(t.Context(), "", "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := e.GetLastSettlementsByInstrument(t.Context(), formatFuturesTradablePair(futuresTradablePair), "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastTradesByCurrency(t.Context(), currency.EMPTYCODE, "option", "36798", "36799", "asc", 0, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetLastTradesByCurrency(t.Context(), currency.BTC, "option", "36798", "36799", "asc", 0, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveLastTradesByCurrency(t.Context(), currency.EMPTYCODE, "option", "36798", "36799", "asc", 0, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.WSRetrieveLastTradesByCurrency(t.Context(), currency.BTC, "option", "36798", "36799", "asc", 0, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastTradesByCurrencyAndTime(t.Context(), currency.EMPTYCODE, "", "", 0, time.Now().Add(-8*time.Hour), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetLastTradesByCurrencyAndTime(t.Context(), currency.BTC, "", "", 0, time.Now().Add(-8*time.Hour), time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.GetLastTradesByCurrencyAndTime(t.Context(), currency.BTC, "option", "asc", 25, time.Now().Add(-8*time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveLastTradesByCurrencyAndTime(t.Context(), currency.EMPTYCODE, "", "", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.WSRetrieveLastTradesByCurrencyAndTime(t.Context(), currency.BTC, "", "", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.WSRetrieveLastTradesByCurrencyAndTime(t.Context(), currency.BTC, "option", "asc", 25, false, time.Now().Add(-8*time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastTradesByInstrument(t.Context(), "", "", "", "", 0, false)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	for assetType, cp := range assetTypeToPairsMap {
		result, err := e.GetLastTradesByInstrument(t.Context(), formatPairString(assetType, cp), "30500", "31500", "desc", 0, true)
		require.NoErrorf(t, err, "expected %v, got %v currency asset %v pair %v", nil, err, assetType, cp)
		require.NotNilf(t, result, "expected value not to be nil for asset %v pair: %v", assetType, cp)
	}
}

func TestWSRetrieveLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveLastTradesByInstrument(t.Context(), "", "", "", "", 0, false)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	for assetType, cp := range assetTypeToPairsMap {
		result, err := e.WSRetrieveLastTradesByInstrument(t.Context(), formatPairString(assetType, cp), "30500", "31500", "desc", 0, true)
		require.NoErrorf(t, err, "expected %v, got %v currency asset %v pair %v", nil, err, assetType, cp)
		require.NotNilf(t, result, "expected value not to be nil for asset %v pair: %v", assetType, cp)
	}
}

func TestGetLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastTradesByInstrumentAndTime(t.Context(), "", "", 0, time.Now().Add(-8*time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	for assetType, cp := range assetTypeToPairsMap {
		result, err := e.GetLastTradesByInstrumentAndTime(t.Context(), formatPairString(assetType, cp), "", 0, time.Now().Add(-8*time.Hour), time.Now())
		require.NoErrorf(t, err, "expected %v, got %v currency pair %v", nil, err, cp)
		require.NotNilf(t, result, "expected value not to be nil for pair: %v", cp)
	}
}

func TestWSRetrieveLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveLastTradesByInstrumentAndTime(t.Context(), "", "", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	for assetType, cp := range assetTypeToPairsMap {
		result, err := e.WSRetrieveLastTradesByInstrumentAndTime(t.Context(), formatPairString(assetType, cp), "", 0, true, time.Now().Add(-8*time.Hour), time.Now())
		require.NoErrorf(t, err, "expected %v, got %v currency pair %v", nil, err, cp)
		require.NotNilf(t, result, "expected value not to be nil for pair: %v", cp)
	}
}

func TestWSProcessTrades(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup instance must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.json", e.wsHandleData)
	e.Websocket.DataHandler.Close()

	a, p, err := getAssetPairByInstrument("BTC-PERPETUAL")
	require.NoError(t, err, "getAssetPairByInstrument must not error")

	exp := []trade.Data{
		{
			Exchange:     e.Name,
			CurrencyPair: p,
			Timestamp:    time.UnixMilli(1742627465811).UTC(),
			Price:        84295.5,
			Amount:       8430.0,
			Side:         order.Buy,
			TID:          "356130997",
			AssetType:    a,
		},
		{
			Exchange:     e.Name,
			CurrencyPair: p,
			Timestamp:    time.UnixMilli(1742627361899).UTC(),
			Price:        84319.0,
			Amount:       580.0,
			Side:         order.Sell,
			TID:          "356130979",
			AssetType:    a,
		},
	}
	require.Len(t, e.Websocket.DataHandler.C, len(exp), "Must see the correct number of trades")
	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case trade.Data:
			i := 1 - len(e.Websocket.DataHandler.C)
			require.Equalf(t, exp[i], v, "Trade [%d] must be correct", i)
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T(%s)", v, v)
		}
	}
}

func TestGetOrderbookData(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), "", 0)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result *Orderbook
	for assetType, cp := range assetTypeToPairsMap {
		result, err = e.GetOrderbook(t.Context(), formatPairString(assetType, cp), 0)
		require.NoErrorf(t, err, "expected %v, got %v currency pair %v", nil, err, cp)
		require.NotNilf(t, result, "expected value not to be nil for pair: %v", cp)
	}
}

func TestWSRetrieveOrderbookData(t *testing.T) {
	t.Parallel()
	if !e.Websocket.IsConnected() {
		t.Skip("websocket is not connected")
	}
	_, err := e.WSRetrieveOrderbookData(t.Context(), "", 0)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result *Orderbook
	for assetType, cp := range assetTypeToPairsMap {
		result, err = e.WSRetrieveOrderbookData(t.Context(), formatPairString(assetType, cp), 0)
		require.NoErrorf(t, err, "expected %v, got %v currency pair %v", nil, err, cp)
		require.NotNilf(t, result, "expected value not to be nil for pair: %v", cp)
	}
}

func TestGetOrderbookByInstrumentID(t *testing.T) {
	t.Parallel()
	combos, err := e.GetComboIDs(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	if len(combos) == 0 {
		t.Skip("no combo instance found for currency BTC")
	}
	_, err = e.GetOrderbookByInstrumentID(t.Context(), 0, 50)
	require.ErrorIs(t, err, errInvalidInstrumentID)

	comboD, err := e.GetComboDetails(t.Context(), combos[0])
	require.NoError(t, err)
	require.NotNil(t, comboD)

	result, err := e.GetOrderbookByInstrumentID(t.Context(), comboD.InstrumentID, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOrderbookByInstrumentID(t *testing.T) {
	t.Parallel()
	combos, err := e.WSRetrieveComboIDs(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	if len(combos) == 0 {
		t.Skip("no combo instance found for currency BTC")
	}
	_, err = e.WSRetrieveOrderbookByInstrumentID(t.Context(), 0, 50)
	require.ErrorIs(t, err, errInvalidInstrumentID)
	comboD, err := e.WSRetrieveComboDetails(t.Context(), combos[0])
	require.NoError(t, err)
	require.NotNil(t, comboD)

	result, err := e.WSRetrieveOrderbookByInstrumentID(t.Context(), comboD.InstrumentID, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSupportedIndexNames(t *testing.T) {
	t.Parallel()
	result, err := e.GetSupportedIndexNames(t.Context(), "derivative")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrieveSupportedIndexNames(t *testing.T) {
	t.Parallel()
	result, err := e.WsRetrieveSupportedIndexNames(t.Context(), "derivative")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeVolumes(t *testing.T) {
	t.Parallel()
	result, err := e.GetTradeVolumes(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveTradeVolumes(t *testing.T) {
	t.Parallel()
	result, err := e.WSRetrieveTradeVolumes(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradingViewChartData(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradingViewChart(t.Context(), "", "60", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := e.GetTradingViewChart(t.Context(), btcPerpInstrument, "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.GetTradingViewChart(t.Context(), spotTradablePair.String(), "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrievesTradingViewChartData(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrievesTradingViewChartData(t.Context(), "", "60", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)
	result, err := e.WSRetrievesTradingViewChartData(t.Context(), btcPerpInstrument, "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.WSRetrievesTradingViewChartData(t.Context(), spotTradablePair.String(), "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVolatilityIndexData(t *testing.T) {
	t.Parallel()
	_, err := e.GetVolatilityIndex(t.Context(), currency.EMPTYCODE, "60", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetVolatilityIndex(t.Context(), currency.BTC, "", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errResolutionNotSet)
	_, err = e.GetVolatilityIndex(t.Context(), currency.BTC, "60", time.Now(), time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetVolatilityIndex(t.Context(), currency.BTC, "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveVolatilityIndexData(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveVolatilityIndexData(t.Context(), currency.EMPTYCODE, "60", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WSRetrieveVolatilityIndexData(t.Context(), currency.BTC, "", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errResolutionNotSet)
	_, err = e.WSRetrieveVolatilityIndexData(t.Context(), currency.BTC, "60", time.Now(), time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.WSRetrieveVolatilityIndexData(t.Context(), currency.BTC, "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicTicker(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := e.GetPublicTicker(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrievePublicTicker(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrievePublicTicker(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := e.WSRetrievePublicTicker(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountSummary(t.Context(), currency.EMPTYCODE, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountSummary(t.Context(), currency.BTC, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveAccountSummary(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveAccountSummary(t.Context(), currency.EMPTYCODE, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveAccountSummary(t.Context(), currency.BTC, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelTransferByID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelTransferByID(t.Context(), currency.EMPTYCODE, "", 23487)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.CancelTransferByID(t.Context(), currency.BTC, "", 0)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelTransferByID(t.Context(), currency.BTC, "", 23487)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCancelTransferByID(t *testing.T) {
	t.Parallel()
	_, err := e.WSCancelTransferByID(t.Context(), currency.EMPTYCODE, "", 23487)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WSCancelTransferByID(t.Context(), currency.BTC, "", 0)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSCancelTransferByID(t.Context(), currency.BTC, "", 23487)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getTransferResponseJSON = `{"count": 2, "data":[{"amount": 0.2, "created_timestamp": 1550579457727, "currency": "BTC", "direction": "payment", "id": 2, "other_side": "2MzyQc5Tkik61kJbEpJV5D5H9VfWHZK9Sgy", "state": "prepared", "type": "user", "updated_timestamp": 1550579457727}, { "amount": 0.3, "created_timestamp": 1550579255800, "currency": "BTC", "direction": "payment", "id": 1, "other_side": "new_user_1_1", "state": "confirmed", "type": "subaccount", "updated_timestamp": 1550579255800} ] }`

func TestGetTransfers(t *testing.T) {
	t.Parallel()
	var resp *TransfersData
	err := json.Unmarshal([]byte(getTransferResponseJSON), &resp)
	require.NoError(t, err)
	_, err = e.GetTransfers(t.Context(), currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTransfers(t.Context(), currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveTransfers(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveTransfers(t.Context(), currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveTransfers(t.Context(), currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const cancelWithdrawlPushDataJSON = `{"address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.5, "confirmed_timestamp": null, "created_timestamp": 1550571443070, "currency": "BTC", "fee": 0.0001, "id": 1, "priority": 0.15, "state": "cancelled", "transaction_id": null, "updated_timestamp": 1550571443070}`

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	var resp *CancelWithdrawalData
	err := json.Unmarshal([]byte(cancelWithdrawlPushDataJSON), &resp)
	require.NoError(t, err)
	_, err = e.CancelWithdrawal(t.Context(), currency.EMPTYCODE, 123844)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.CancelWithdrawal(t.Context(), currency.BTC, 0)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelWithdrawal(t.Context(), currency.BTC, 123844)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.WSCancelWithdrawal(t.Context(), currency.EMPTYCODE, 123844)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WSCancelWithdrawal(t.Context(), currency.BTC, 0)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSCancelWithdrawal(t.Context(), currency.BTC, 123844)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.CreateDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateDepositAddress(t.Context(), currency.SOL)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCreateDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.WSCreateDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSCreateDepositAddress(t.Context(), currency.SOL)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrentDepositAddress(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveCurrentDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveCurrentDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveCurrentDepositAddress(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getDepositPushDataJSON = `{"count": 1, "data": [ { "address": "2N35qDKDY22zmJq9eSyiAerMD4enJ1xx6ax", "amount": 5, "currency": "BTC", "received_timestamp": 1549295017670, "state": "completed", "transaction_id": "230669110fdaf0a0dbcdc079b6b8b43d5af29cc73683835b9bc6b3406c065fda", "updated_timestamp": 1549295130159} ] }`

func TestGetDeposits(t *testing.T) {
	t.Parallel()
	var resp *DepositsData
	err := json.Unmarshal([]byte(getDepositPushDataJSON), &resp)
	require.NoError(t, err)
	_, err = e.GetDeposits(t.Context(), currency.EMPTYCODE, 25, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDeposits(t.Context(), currency.BTC, 25, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveDeposits(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveDeposits(t.Context(), currency.EMPTYCODE, 25, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveDeposits(t.Context(), currency.BTC, 25, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getWithdrawalResponseJSON = `{"count": 1, "data": [ { "address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.5, "confirmed_timestamp": null, "created_timestamp": 1550571443070, "currency": "BTC", "fee": 0.0001, "id": 1, "priority": 0.15, "state": "unconfirmed", "transaction_id": null, "updated_timestamp": 1550571443070} ] }`

func TestGetWithdrawals(t *testing.T) {
	t.Parallel()
	var resp *WithdrawalsData
	err := json.Unmarshal([]byte(getWithdrawalResponseJSON), &resp)
	require.NoError(t, err)
	_, err = e.GetWithdrawals(t.Context(), currency.EMPTYCODE, 25, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawals(t.Context(), currency.BTC, 25, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveWithdrawals(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveWithdrawals(t.Context(), currency.EMPTYCODE, 25, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveWithdrawals(t.Context(), currency.BTC, 25, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitTransferBetweenSubAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitTransferBetweenSubAccounts(t.Context(), currency.EMPTYCODE, 12345, 2, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 0, 2, "")
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.SubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 12345, -1, "")
	require.ErrorIs(t, err, errInvalidDestinationID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 12345, 4, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsSubmitTransferBetweenSubAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.WsSubmitTransferBetweenSubAccounts(t.Context(), currency.EMPTYCODE, 12345, 2, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WsSubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 0, 2, "")
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.WsSubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 12345, -1, "")
	require.ErrorIs(t, err, errInvalidDestinationID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WsSubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 12345, 2, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitTransferToSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitTransferToSubAccount(t.Context(), currency.EMPTYCODE, 0.01, 13434)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SubmitTransferToSubAccount(t.Context(), currency.BTC, 0, 13434)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.SubmitTransferToSubAccount(t.Context(), currency.BTC, 0.01, 0)
	require.ErrorIs(t, err, errInvalidDestinationID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitTransferToSubAccount(t.Context(), currency.BTC, 0.01, 13434)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitTransferToSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.WSSubmitTransferToSubAccount(t.Context(), currency.EMPTYCODE, 0.01, 13434)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WSSubmitTransferToSubAccount(t.Context(), currency.BTC, 0, 13434)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.WSSubmitTransferToSubAccount(t.Context(), currency.BTC, 0.01, 0)
	require.ErrorIs(t, err, errInvalidDestinationID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitTransferToSubAccount(t.Context(), currency.BTC, 0.01, 13434)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitTransferToUser(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitTransferToUser(t.Context(), currency.EMPTYCODE, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SubmitTransferToUser(t.Context(), currency.BTC, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.SubmitTransferToUser(t.Context(), currency.BTC, "", "", 0.001)
	require.ErrorIs(t, err, errInvalidCryptoAddress)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitTransferToUser(t.Context(), currency.BTC, "", "13434", 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitTransferToUser(t *testing.T) {
	t.Parallel()
	_, err := e.WSSubmitTransferToUser(t.Context(), currency.EMPTYCODE, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WSSubmitTransferToUser(t.Context(), currency.BTC, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.WSSubmitTransferToUser(t.Context(), currency.BTC, "", "", 0.001)
	require.ErrorIs(t, err, errInvalidCryptoAddress)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitTransferToUser(t.Context(), currency.BTC, "", "", 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const submitWithdrawalResponseJSON = `{"address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.4, "confirmed_timestamp": null, "created_timestamp": 1550574558607, "currency": "BTC", "fee": 0.0001, "id": 4, "priority": 1, "state": "unconfirmed", "transaction_id": null, "updated_timestamp": 1550574558607}`

func TestSubmitWithdraw(t *testing.T) {
	t.Parallel()
	var resp *WithdrawData
	err := json.Unmarshal([]byte(submitWithdrawalResponseJSON), &resp)
	require.NoError(t, err)
	_, err = e.SubmitWithdraw(t.Context(), currency.EMPTYCODE, core.BitcoinDonationAddress, "", 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SubmitWithdraw(t.Context(), currency.BTC, core.BitcoinDonationAddress, "", 0)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.SubmitWithdraw(t.Context(), currency.BTC, "", "", 0.001)
	require.ErrorIs(t, err, errInvalidCryptoAddress)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitWithdraw(t.Context(), currency.BTC, core.BitcoinDonationAddress, "", 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitWithdraw(t *testing.T) {
	_, err := e.WSSubmitWithdraw(t.Context(), currency.EMPTYCODE, core.BitcoinDonationAddress, "", 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WSSubmitWithdraw(t.Context(), currency.BTC, core.BitcoinDonationAddress, "", 0)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.WSSubmitWithdraw(t.Context(), currency.BTC, "", "", 0.001)
	require.ErrorIs(t, err, errInvalidCryptoAddress)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitWithdraw(t.Context(), currency.BTC, core.BitcoinDonationAddress, "", 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	result, err := e.GetAnnouncements(t.Context(), time.Now(), 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveAnnouncements(t *testing.T) {
	t.Parallel()
	result, err := e.WSRetrieveAnnouncements(t.Context(), time.Now(), 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccessLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccessLog(t.Context(), 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveAccessLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveAccessLog(t.Context(), 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAPIKeyName(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeAPIKeyName(t.Context(), 0, "TestKey123")
	require.ErrorIs(t, err, errInvalidID)
	_, err = e.ChangeAPIKeyName(t.Context(), 2, "TestKey123$")
	require.ErrorIs(t, err, errUnacceptableAPIKey)
	_, err = e.ChangeAPIKeyName(t.Context(), 2, "#$#")
	require.ErrorIs(t, err, errUnacceptableAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.ChangeAPIKeyName(t.Context(), 1, "TestKey123")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSChangeAPIKeyName(t *testing.T) {
	t.Parallel()
	_, err := e.WSChangeAPIKeyName(t.Context(), 0, "TestKey123")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.WSChangeAPIKeyName(t.Context(), 1, "TestKey123")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeMarginModel(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeMarginModel(t.Context(), 2, "", false)
	require.ErrorIs(t, err, errInvalidMarginModel)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeMarginModel(t.Context(), 2, "segregated_pm", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsChangeMarginModel(t *testing.T) {
	t.Parallel()
	_, err := e.WsChangeMarginModel(t.Context(), 2, "", false)
	require.ErrorIs(t, err, errInvalidMarginModel)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	result, err := e.WsChangeMarginModel(t.Context(), 2, "segregated_pm", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeScopeInAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeScopeInAPIKey(t.Context(), -1, "account:read_write")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.ChangeScopeInAPIKey(t.Context(), 1, "account:read_write")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSChangeScopeInAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.WSChangeScopeInAPIKey(t.Context(), 0, "account:read_write")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.WSChangeScopeInAPIKey(t.Context(), 1, "account:read_write")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeSubAccountName(t *testing.T) {
	t.Parallel()
	err := e.ChangeSubAccountName(t.Context(), 0, "new_sub")
	require.ErrorIs(t, err, errInvalidID)
	err = e.ChangeSubAccountName(t.Context(), 312313, "")
	require.ErrorIs(t, err, errInvalidUsername)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.ChangeSubAccountName(t.Context(), 1, "new_sub")
	assert.NoError(t, err)
}

func TestWSChangeSubAccountName(t *testing.T) {
	t.Parallel()
	err := e.WSChangeSubAccountName(t.Context(), 0, "new_sub")
	require.ErrorIs(t, err, errInvalidID)
	err = e.WSChangeSubAccountName(t.Context(), 312313, "")
	require.ErrorIs(t, err, errInvalidUsername)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.WSChangeSubAccountName(t.Context(), 1, "new_sub")
	assert.NoError(t, err)
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.CreateAPIKey(t.Context(), "account:read_write", "new_sub", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCreateAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.WSCreateAPIKey(t.Context(), "account:read_write", "new_sub", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateSubAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCreateSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSCreateSubAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDisableAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.DisableAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.DisableAPIKey(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSDisableAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.WSDisableAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.WSDisableAPIKey(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEditAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.EditAPIKey(t.Context(), 0, "trade", "", false, []string{"read", "read_write"}, []string{})
	require.ErrorIs(t, err, errInvalidAPIKeyID)
	_, err = e.EditAPIKey(t.Context(), 1234, "", "", false, []string{"read", "read_write"}, []string{})
	require.ErrorIs(t, err, errMaxScopeIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.EditAPIKey(t.Context(), 1234, "trade", "", false, []string{"read", "read_write"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsEditAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.WsEditAPIKey(t.Context(), 0, "trade", "", false, []string{"read", "read_write"}, []string{})
	require.ErrorIs(t, err, errInvalidAPIKeyID)
	_, err = e.WsEditAPIKey(t.Context(), 1234, "", "", false, []string{"read", "read_write"}, []string{})
	require.ErrorIs(t, err, errMaxScopeIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.WsEditAPIKey(t.Context(), 1234, "trade", "", false, []string{"read", "read_write"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableAffiliateProgram(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.EnableAffiliateProgram(t.Context())
	assert.NoError(t, err)
}

func TestWSEnableAffiliateProgram(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.WSEnableAffiliateProgram(t.Context())
	assert.NoError(t, err)
}

func TestEnableAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.EnableAPIKey(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSEnableAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.WSEnableAPIKey(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateProgramInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateProgramInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveAffiliateProgramInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveAffiliateProgramInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEmailLanguage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetEmailLanguage(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveEmailLanguage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveEmailLanguage(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNewAnnouncements(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetNewAnnouncements(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveNewAnnouncements(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveNewAnnouncements(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetPosition(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPosition(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrievePosition(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrievePosition(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrievePosition(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccounts(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveSubAccounts(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountDetails(t.Context(), currency.EMPTYCODE, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountDetails(t.Context(), currency.BTC, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveSubAccountDetails(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveSubAccountDetails(t.Context(), currency.EMPTYCODE, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveSubAccountDetails(t.Context(), currency.BTC, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositions(t.Context(), currency.EMPTYCODE, "option")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPositions(t.Context(), currency.BTC, "option")
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.GetPositions(t.Context(), currency.ETH, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrievePositions(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrievePositions(t.Context(), currency.EMPTYCODE, "option")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrievePositions(t.Context(), currency.BTC, "option")
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.WSRetrievePositions(t.Context(), currency.ETH, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getTransactionLogResponseJSON = `{"logs": [ { "username": "TestUser", "user_seq": 6009, "user_id": 7, "type": "transfer", "trade_id": null, "timestamp": 1613659830333, "side": "-", "price": null, "position": null, "order_id": null, "interest_pl": null, "instrument_name": null, "info": { "transfer_type": "subaccount", "other_user_id": 27, "other_user": "Subaccount" }, "id": 61312, "equity": 3000.9275869, "currency": "BTC", "commission": 0, "change": -2.5, "cashflow": -2.5, "balance": 3001.22270418 } ], "continuation": 61282 }`

func TestGetTransactionLog(t *testing.T) {
	t.Parallel()
	var resp *TransactionsData
	err := json.Unmarshal([]byte(getTransactionLogResponseJSON), &resp)
	require.NoError(t, err)
	_, err = e.GetTransactionLog(t.Context(), currency.EMPTYCODE, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTransactionLog(t.Context(), currency.BTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveTransactionLog(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveTransactionLog(t.Context(), currency.EMPTYCODE, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveTransactionLog(t.Context(), currency.BTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserLocks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserLocks(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserLocks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveUserLocks(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListAPIKeys(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ListAPIKeys(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSListAPIKeys(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSListAPIKeys(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCustodyAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetCustodyAccounts(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCustodyAccounts(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrieveCustodyAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.WsRetrieveCustodyAccounts(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsRetrieveCustodyAccounts(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRemoveAPIKey(t *testing.T) {
	t.Parallel()
	err := e.RemoveAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	err = e.RemoveAPIKey(t.Context(), 1)
	assert.NoError(t, err)
}

func TestWSRemoveAPIKey(t *testing.T) {
	t.Parallel()
	err := e.WSRemoveAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	err = e.WSRemoveAPIKey(t.Context(), 1)
	assert.NoError(t, err)
}

func TestRemoveSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	err := e.RemoveSubAccount(t.Context(), 1)
	assert.NoError(t, err)
}

func TestWSRemoveSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	err := e.WSRemoveSubAccount(t.Context(), 1)
	assert.NoError(t, err)
}

func TestResetAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.ResetAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.ResetAPIKey(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSResetAPIKey(t *testing.T) {
	t.Parallel()
	err := e.WSResetAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	err = e.WSResetAPIKey(t.Context(), 1)
	assert.NoError(t, err)
}

func TestSetAnnouncementAsRead(t *testing.T) {
	t.Parallel()
	err := e.SetAnnouncementAsRead(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetAnnouncementAsRead(t.Context(), 1)
	assert.NoError(t, err)
}

func TestSetEmailForSubAccount(t *testing.T) {
	t.Parallel()
	err := e.SetEmailForSubAccount(t.Context(), 0, "wrongemail@wrongemail.com")
	require.ErrorIs(t, err, errInvalidID)
	err = e.SetEmailForSubAccount(t.Context(), 1, "")
	require.ErrorIs(t, err, errInvalidEmailAddress)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetEmailForSubAccount(t.Context(), 1, "wrongemail@wrongemail.com")
	assert.NoError(t, err)
}

func TestWSSetEmailForSubAccount(t *testing.T) {
	t.Parallel()
	err := e.WSSetEmailForSubAccount(t.Context(), 0, "wrongemail@wrongemail.com")
	require.ErrorIs(t, err, errInvalidID)
	err = e.WSSetEmailForSubAccount(t.Context(), 1, "")
	require.ErrorIs(t, err, errInvalidEmailAddress)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.WSSetEmailForSubAccount(t.Context(), 1, "wrongemail@wrongemail.com")
	assert.NoError(t, err)
}

func TestSetEmailLanguage(t *testing.T) {
	t.Parallel()
	err := e.SetEmailLanguage(t.Context(), "")
	require.ErrorIs(t, err, errLanguageIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetEmailLanguage(t.Context(), "en")
	assert.NoError(t, err)
}

func TestWSSetEmailLanguage(t *testing.T) {
	t.Parallel()
	err := e.WSSetEmailLanguage(t.Context(), "")
	require.ErrorIs(t, err, errLanguageIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.WSSetEmailLanguage(t.Context(), "en")
	assert.NoError(t, err)
}

func TestSetSelfTradingConfig(t *testing.T) {
	t.Parallel()
	_, err := e.SetSelfTradingConfig(t.Context(), "", false)
	require.ErrorIs(t, err, errTradeModeIsRequired)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetSelfTradingConfig(t.Context(), "reject_taker", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsSetSelfTradingConfig(t *testing.T) {
	t.Parallel()
	_, err := e.WsSetSelfTradingConfig(t.Context(), "", false)
	require.ErrorIs(t, err, errTradeModeIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WsSetSelfTradingConfig(t.Context(), "reject_taker", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestToggleNotificationsFromSubAccount(t *testing.T) {
	t.Parallel()
	err := e.ToggleNotificationsFromSubAccount(t.Context(), 0, false)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.ToggleNotificationsFromSubAccount(t.Context(), 1, false)
	assert.NoError(t, err)
}

func TestWSToggleNotificationsFromSubAccount(t *testing.T) {
	t.Parallel()
	err := e.WSToggleNotificationsFromSubAccount(t.Context(), 0, false)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.WSToggleNotificationsFromSubAccount(t.Context(), 1, false)
	assert.NoError(t, err)
}

func TestTogglePortfolioMargining(t *testing.T) {
	t.Parallel()
	_, err := e.TogglePortfolioMargining(t.Context(), 0, false, false)
	require.ErrorIs(t, err, errUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.TogglePortfolioMargining(t.Context(), 1234, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSTogglePortfolioMargining(t *testing.T) {
	t.Parallel()
	_, err := e.WSTogglePortfolioMargining(t.Context(), 0, false, false)
	require.ErrorIs(t, err, errUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSTogglePortfolioMargining(t.Context(), 1234, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestToggleSubAccountLogin(t *testing.T) {
	t.Parallel()
	err := e.ToggleSubAccountLogin(t.Context(), -1, false)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.ToggleSubAccountLogin(t.Context(), 1, false)
	assert.NoError(t, err)
}

func TestWSToggleSubAccountLogin(t *testing.T) {
	t.Parallel()
	err := e.WSToggleSubAccountLogin(t.Context(), -1, false)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.WSToggleSubAccountLogin(t.Context(), 1, false)
	assert.NoError(t, err)
}

func TestSubmitBuy(t *testing.T) {
	t.Parallel()
	pairs, err := e.GetEnabledPairs(asset.Futures)
	require.NoError(t, err)
	_, err = e.SubmitBuy(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.SubmitBuy(t.Context(), &OrderBuyAndSellParams{
		Instrument: "", OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "",
		Amount: 30, Price: 500000,
		MaxShow: 0, TriggerPrice: 0,
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitBuy(t.Context(), &OrderBuyAndSellParams{
		Instrument: pairs[0].String(), OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "",
		Amount: 30, Price: 500000,
		MaxShow: 0, TriggerPrice: 0,
		PostOnly: false, RejectPostOnly: false,
		ReduceOnly: false, MMP: false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitBuy(t *testing.T) {
	t.Parallel()
	_, err := e.WSSubmitBuy(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.WSSubmitBuy(t.Context(), &OrderBuyAndSellParams{
		Instrument: "", OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "",
		Amount: 30, Price: 500000,
		MaxShow: 0, TriggerPrice: 0,
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitBuy(t.Context(), &OrderBuyAndSellParams{
		Instrument: btcPerpInstrument, OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "",
		Amount: 30, Price: 500000,
		MaxShow: 0, TriggerPrice: 0,
		PostOnly: false, RejectPostOnly: false,
		ReduceOnly: false, MMP: false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitSell(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitSell(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	info, err := e.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	_, err = e.SubmitSell(t.Context(), &OrderBuyAndSellParams{OrderType: "limit", Label: "testOrder", TimeInForce: "", Trigger: "", Advanced: "", Amount: info.ContractSize * 3, Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false, RejectPostOnly: false, ReduceOnly: false, MMP: false})
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitSell(t.Context(), &OrderBuyAndSellParams{Instrument: btcPerpInstrument, OrderType: "limit", Label: "testOrder", TimeInForce: "", Trigger: "", Advanced: "", Amount: info.ContractSize * 3, Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false, RejectPostOnly: false, ReduceOnly: false, MMP: false})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitSell(t *testing.T) {
	t.Parallel()
	info, err := e.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	_, err = e.WSSubmitSell(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.WSSubmitSell(t.Context(), &OrderBuyAndSellParams{
		Instrument: "", OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "", Amount: info.ContractSize * 3,
		Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false,
		RejectPostOnly: false, ReduceOnly: false, MMP: false,
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitSell(t.Context(), &OrderBuyAndSellParams{
		Instrument: btcPerpInstrument, OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "", Amount: info.ContractSize * 3,
		Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false,
		RejectPostOnly: false, ReduceOnly: false, MMP: false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEditOrderByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.EditOrderByLabel(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.EditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: "",
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = e.EditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 0, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.ErrorIs(t, err, errInvalidAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSEditOrderByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.WSEditOrderByLabel(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.WSEditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: "",
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = e.WSEditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 0, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.ErrorIs(t, err, errInvalidAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSEditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const submitCancelResponseJSON = `{"triggered": false, "trigger": "index_price", "time_in_force": "good_til_cancelled", "trigger_price": 144.73, "reduce_only": false, "profit_loss": 0, "price": "1234", "post_only": false, "order_type": "stop_market", "order_state": "untriggered", "order_id": "ETH-SLIS-12", "max_show": 5, "last_update_timestamp": 1550575961291, "label": "", "is_liquidation": false, "instrument_name": "ETH-PERPETUAL", "direction": "sell", "creation_timestamp": 1550575961291, "api": false, "amount": 5 }`

func TestSubmitCancel(t *testing.T) {
	t.Parallel()
	var resp *PrivateCancelData
	err := json.Unmarshal([]byte(submitCancelResponseJSON), &resp)
	require.NoError(t, err)
	_, err = e.SubmitCancel(t.Context(), "")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitCancel(t.Context(), "incorrectID")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancel(t *testing.T) {
	t.Parallel()
	_, err := e.WSSubmitCancel(t.Context(), "")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitCancel(t.Context(), "incorrectID")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelAll(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitCancelAll(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancelAll(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitCancelAll(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelAllByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitCancelAllByCurrency(t.Context(), currency.EMPTYCODE, "option", "", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitCancelAllByCurrency(t.Context(), currency.BTC, "option", "", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancelAllByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WSSubmitCancelAllByCurrency(t.Context(), currency.EMPTYCODE, "option", "", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitCancelAllByCurrency(t.Context(), currency.BTC, "option", "", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelAllByKind(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitCancelAllByKind(t.Context(), currency.EMPTYCODE, "option_combo", "trigger_all", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitCancelAllByKind(t.Context(), currency.ETH, "option_combo", "trigger_all", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsSubmitCancelAllByKind(t *testing.T) {
	t.Parallel()
	_, err := e.WsSubmitCancelAllByKind(t.Context(), currency.EMPTYCODE, "option_combo", "trigger_all", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WsSubmitCancelAllByKind(t.Context(), currency.ETH, "option_combo", "trigger_all", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelAllByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitCancelAllByInstrument(t.Context(), "", "all", true, true)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitCancelAllByInstrument(t.Context(), btcPerpInstrument, "all", true, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancelAllByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.WSSubmitCancelAllByInstrument(t.Context(), "", "all", true, true)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitCancelAllByInstrument(t.Context(), btcPerpInstrument, "all", true, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelByLabel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitCancelByLabel(t.Context(), "incorrectOrderLabel", currency.EMPTYCODE, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancelByLabel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitCancelByLabel(t.Context(), "incorrectOrderLabel", currency.EMPTYCODE, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelQuotes(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitCancelQuotes(t.Context(), currency.EMPTYCODE, 0, 0, "all", "", formatFuturesTradablePair(futuresTradablePair), "future", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitCancelQuotes(t.Context(), currency.BTC, 0, 0, "all", "", formatFuturesTradablePair(futuresTradablePair), "future", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancelQuotes(t *testing.T) {
	t.Parallel()
	_, err := e.WSSubmitCancelQuotes(t.Context(), currency.EMPTYCODE, 0, 0, "all", "", formatFuturesTradablePair(futuresTradablePair), "future", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitCancelQuotes(t.Context(), currency.BTC, 0, 0, "all", "", formatFuturesTradablePair(futuresTradablePair), "future", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitClosePosition(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitClosePosition(t.Context(), "", "limit", 35000)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitClosePosition(t.Context(), formatFuturesTradablePair(futuresTradablePair), "limit", 35000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitClosePosition(t *testing.T) {
	t.Parallel()
	_, err := e.WSSubmitClosePosition(t.Context(), "", "limit", 35000)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitClosePosition(t.Context(), formatFuturesTradablePair(futuresTradablePair), "limit", 35000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMargins(t *testing.T) {
	t.Parallel()
	_, err := e.GetMargins(t.Context(), "", 5, 35000)
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = e.GetMargins(t.Context(), formatFuturesTradablePair(futuresTradablePair), 0, 35000)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.GetMargins(t.Context(), formatFuturesTradablePair(futuresTradablePair), 5, -1)
	require.ErrorIs(t, err, errInvalidPrice)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMargins(t.Context(), formatFuturesTradablePair(futuresTradablePair), 5, 35000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveMargins(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveMargins(t.Context(), "", 5, 35000)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveMargins(t.Context(), formatFuturesTradablePair(futuresTradablePair), 5, 35000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMMPConfig(t *testing.T) {
	t.Parallel()
	_, err := e.GetMMPConfig(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMMPConfig(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveMMPConfig(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveMMPConfig(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveMMPConfig(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getOpenOrdersByCurrencyResponseJSON = `[{ "time_in_force": "good_til_cancelled", "reduce_only": false, "profit_loss": 0, "price": 0.0028, "post_only": false, "order_type": "limit", "order_state": "open", "order_id": "146062", "max_show": 10, "last_update_timestamp": 1550050597036, "label": "", "is_liquidation": false, "instrument_name": "BTC-15FEB19-3250-P", "filled_amount": 0, "direction": "buy", "creation_timestamp": 1550050597036, "commission": 0, "average_price": 0, "api": true, "amount": 10 } ]`

func TestGetOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	var resp []OrderData
	err := json.Unmarshal([]byte(getOpenOrdersByCurrencyResponseJSON), &resp)
	require.NoError(t, err)
	_, err = e.GetOpenOrdersByCurrency(t.Context(), currency.EMPTYCODE, "option", "all")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrdersByCurrency(t.Context(), currency.BTC, "option", "all")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveOpenOrdersByCurrency(t.Context(), currency.EMPTYCODE, "option", "all")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveOpenOrdersByCurrency(t.Context(), currency.BTC, "option", "all")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrdersByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrdersByLabel(t.Context(), currency.EMPTYCODE, "the-label")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrdersByLabel(t.Context(), currency.EURR, "the-label")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOpenOrdersByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveOpenOrdersByLabel(t.Context(), currency.EMPTYCODE, "the-label")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSRetrieveOpenOrdersByLabel(t.Context(), currency.EURR, "the-label")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrdersByInstrument(t.Context(), "", "all")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrdersByInstrument(t.Context(), btcPerpInstrument, "all")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveOpenOrdersByInstrument(t.Context(), "", "all")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveOpenOrdersByInstrument(t.Context(), btcPerpInstrument, "all")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistoryByCurrency(t.Context(), currency.EMPTYCODE, "future", 0, 0, false, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderHistoryByCurrency(t.Context(), currency.BTC, "future", 0, 0, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveOrderHistoryByCurrency(t.Context(), currency.EMPTYCODE, "future", 0, 0, false, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveOrderHistoryByCurrency(t.Context(), currency.BTC, "future", 0, 0, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistoryByInstrument(t.Context(), "", 0, 0, false, false)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderHistoryByInstrument(t.Context(), btcPerpInstrument, 0, 0, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveOrderHistoryByInstrument(t.Context(), "", 0, 0, false, false)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveOrderHistoryByInstrument(t.Context(), btcPerpInstrument, 0, 0, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderMarginsByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderMarginsByID(t.Context(), []string{})
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderMarginsByID(t.Context(), []string{"21422175153", "21422175154"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOrderMarginsByID(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveOrderMarginsByID(t.Context(), []string{})
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveOrderMarginsByID(t.Context(), []string{"ETH-349280", "ETH-349279", "ETH-349278"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderState(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderState(t.Context(), "")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderState(t.Context(), "brokenid123")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrievesOrderState(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrievesOrderState(t.Context(), "")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrievesOrderState(t.Context(), "brokenid123")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderStateByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderStateByLabel(t.Context(), currency.EMPTYCODE, "the-label")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderStateByLabel(t.Context(), currency.EURR, "the-label")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrieveOrderStateByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.WsRetrieveOrderStateByLabel(t.Context(), currency.EMPTYCODE, "the-label")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsRetrieveOrderStateByLabel(t.Context(), currency.EURR, "the-label")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTriggerOrderHistory(t.Context(), currency.EMPTYCODE, "", "", 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTriggerOrderHistory(t.Context(), currency.ETH, "", "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveTriggerOrderHistory(t.Context(), currency.EMPTYCODE, "", "", 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveTriggerOrderHistory(t.Context(), currency.ETH, "", "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getUserTradesByCurrencyResponseJSON = `{"trades": [ { "underlying_price": 204.5, "trade_seq": 3, "trade_id": "ETH-2696060", "timestamp": 1590480363130, "tick_direction": 2, "state": "filled", "reduce_only": false, "price": 0.361, "post_only": false, "order_type": "limit", "order_id": "ETH-584827850", "matching_id": null, "mark_price": 0.364585, "liquidity": "T", "iv": 0, "instrument_name": "ETH-29MAY20-130-C", "index_price": 203.72, "fee_currency": "ETH", "fee": 0.002, "direction": "sell", "amount": 5 }, { "underlying_price": 204.82, "trade_seq": 3, "trade_id": "ETH-2696062", "timestamp": 1590480416119, "tick_direction": 0, "state": "filled", "reduce_only": false, "price": 0.015, "post_only": false, "order_type": "limit", "order_id": "ETH-584828229", "matching_id": null, "mark_price": 0.000596, "liquidity": "T", "iv": 352.91, "instrument_name": "ETH-29MAY20-140-P", "index_price": 204.06, "fee_currency": "ETH", "fee": 0.002, "direction": "buy", "amount": 5 } ], "has_more": true }`

func TestGetUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	var resp *UserTradesData
	err := json.Unmarshal([]byte(getUserTradesByCurrencyResponseJSON), &resp)
	require.NoError(t, err)
	_, err = e.GetUserTradesByCurrency(t.Context(), currency.EMPTYCODE, "future", "", "", "asc", 0, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTradesByCurrency(t.Context(), currency.ETH, "future", "", "", "asc", 0, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveUserTradesByCurrency(t.Context(), currency.EMPTYCODE, "future", "", "", "asc", 0, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveUserTradesByCurrency(t.Context(), currency.ETH, "future", "", "", "asc", 0, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTradesByCurrencyAndTime(t.Context(), currency.EMPTYCODE, "future", "default", 5, time.Now().Add(-time.Hour*10), time.Now().Add(-time.Hour*1))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTradesByCurrencyAndTime(t.Context(), currency.ETH, "future", "default", 5, time.Now().Add(-time.Hour*10), time.Now().Add(-time.Hour*1))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveUserTradesByCurrencyAndTime(t.Context(), currency.EMPTYCODE, "future", "default", 5, time.Now().Add(-time.Hour*4), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveUserTradesByCurrencyAndTime(t.Context(), currency.ETH, "future", "default", 5, time.Now().Add(-time.Hour*4), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTradesByInstrument(t.Context(), "", "asc", 5, 10, 4, true)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTradesByInstrument(t.Context(), btcPerpInstrument, "asc", 5, 10, 4, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrieveUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.WsRetrieveUserTradesByInstrument(t.Context(), "", "asc", 5, 10, 4, true)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsRetrieveUserTradesByInstrument(t.Context(), btcPerpInstrument, "asc", 5, 10, 4, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTradesByInstrumentAndTime(t.Context(), "", "asc", 10, time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTradesByInstrumentAndTime(t.Context(), btcPerpInstrument, "asc", 10, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveUserTradesByInstrumentAndTime(t.Context(), "", "asc", 10, false, time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveUserTradesByInstrumentAndTime(t.Context(), btcPerpInstrument, "asc", 10, false, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTradesByOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTradesByOrder(t.Context(), "", "default")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTradesByOrder(t.Context(), "wrongOrderID", "default")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserTradesByOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveUserTradesByOrder(t.Context(), "", "default")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveUserTradesByOrder(t.Context(), "wrongOrderID", "default")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetMMP(t *testing.T) {
	t.Parallel()
	err := e.ResetMMP(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.ResetMMP(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestWSResetMMP(t *testing.T) {
	t.Parallel()
	err := e.WSResetMMP(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.WSResetMMP(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestSetMMPConfig(t *testing.T) {
	t.Parallel()
	err := e.SetMMPConfig(t.Context(), currency.EMPTYCODE, kline.FiveMin, 5, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.SetMMPConfig(t.Context(), currency.BTC, kline.FiveMin, 5, 0, 0)
	assert.NoError(t, err)
}

func TestWSSetMMPConfig(t *testing.T) {
	t.Parallel()
	err := e.WSSetMMPConfig(t.Context(), currency.EMPTYCODE, kline.FiveMin, 5, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.WSSetMMPConfig(t.Context(), currency.BTC, kline.FiveMin, 5, 0, 0)
	assert.NoError(t, err)
}

func TestGetSettlementHistoryByCurency(t *testing.T) {
	t.Parallel()
	_, err := e.GetSettlementHistoryByCurency(t.Context(), currency.EMPTYCODE, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSettlementHistoryByCurency(t.Context(), currency.BTC, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveSettlementHistoryByCurency(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveSettlementHistoryByCurency(t.Context(), currency.EMPTYCODE, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveSettlementHistoryByCurency(t.Context(), currency.BTC, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getSettlementHistoryByInstrumentResponseJSON = `{"settlements": [ { "type": "settlement", "timestamp": 1550475692526, "session_profit_loss": 0.038358299, "profit_loss": -0.001783937, "position": -66, "mark_price": 121.67, "instrument_name": "ETH-22FEB19", "index_price": 119.8 } ], "continuation": "xY7T6cusbMBNpH9SNmKb94jXSBxUPojJEdCPL4YociHBUgAhWQvEP" }`

func TestGetSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	var result *PrivateSettlementsHistoryData
	err := json.Unmarshal([]byte(getSettlementHistoryByInstrumentResponseJSON), &result)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err = e.GetSettlementHistoryByInstrument(t.Context(), btcPerpInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveSettlementHistoryByInstrument(t.Context(), "", "settlement", "", 10, time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveSettlementHistoryByInstrument(t.Context(), btcPerpInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitEdit(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitEdit(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.SubmitEdit(t.Context(), &OrderBuyAndSellParams{OrderID: "", Advanced: "", TriggerPrice: 0.001, Price: 100000, Amount: 123})
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitEdit(t.Context(), &OrderBuyAndSellParams{OrderID: "incorrectID", Advanced: "", TriggerPrice: 0.001, Price: 100000, Amount: 123})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitEdit(t *testing.T) {
	t.Parallel()
	_, err := e.WSSubmitEdit(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSSubmitEdit(t.Context(), &OrderBuyAndSellParams{
		OrderID:      "incorrectID",
		Advanced:     "",
		TriggerPrice: 0.001,
		Price:        100000,
		Amount:       123,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// Combo Books Endpoints

func TestGetComboIDS(t *testing.T) {
	t.Parallel()
	_, err := e.GetComboIDs(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetComboIDs(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveComboIDS(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveComboIDs(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	combos, err := e.WSRetrieveComboIDs(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	assert.NotEmpty(t, combos)
}

func TestGetComboDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetComboDetails(t.Context(), "")
	require.ErrorIs(t, err, errInvalidComboID)

	result, err := e.GetComboDetails(t.Context(), futureComboPairToString(futureComboTradablePair))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveComboDetails(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveComboDetails(t.Context(), "")
	require.ErrorIs(t, err, errInvalidComboID)

	result, err := e.WSRetrieveComboDetails(t.Context(), futureComboPairToString(futureComboTradablePair))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCombos(t *testing.T) {
	t.Parallel()
	_, err := e.GetCombos(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetCombos(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateCombo(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCombo(t.Context(), []ComboParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	instruments, err := e.GetEnabledPairs(asset.Futures)
	require.NoError(t, err)
	if len(instruments) < 2 {
		t.Skip("no enough instrument found")
	}
	_, err = e.CreateCombo(t.Context(), []ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Direction:      "sell",
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "sell",
			Amount:         1200,
		},
	})
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.CreateCombo(t.Context(), []ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Amount:         123,
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "sell",
			Amount:         1200,
		},
	})
	require.ErrorIs(t, err, errInvalidOrderSideOrDirection)
	_, err = e.CreateCombo(t.Context(), []ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Direction:      "buy",
			Amount:         123,
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "buy",
			Amount:         1200,
		},
	})
	require.ErrorIs(t, err, errDifferentInstruments)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateCombo(t.Context(), []ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Direction:      "buy",
			Amount:         123,
		},
		{
			InstrumentName: instruments[0].String(),
			Direction:      "sell",
			Amount:         1200,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCreateCombo(t *testing.T) {
	t.Parallel()
	_, err := e.WSCreateCombo(t.Context(), []ComboParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	instruments, err := e.GetEnabledPairs(asset.Futures)
	require.NoError(t, err)
	if len(instruments) < 2 {
		t.Skip("no enough instrument found")
	}
	_, err = e.WSCreateCombo(t.Context(), []ComboParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	_, err = e.WSCreateCombo(t.Context(), []ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Direction:      "sell",
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "sell",
			Amount:         1200,
		},
	})
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.WSCreateCombo(t.Context(), []ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Amount:         123,
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "sell",
			Amount:         1200,
		},
	})
	require.ErrorIs(t, err, errInvalidOrderSideOrDirection)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSCreateCombo(t.Context(), []ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Direction:      "sell",
			Amount:         123,
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "buy",
			Amount:         1200,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVerifyBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.VerifyBlockTrade(t.Context(), time.Now(), "", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingNonce)
	_, err = e.VerifyBlockTrade(t.Context(), time.Now(), "nonce-string", "", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = e.VerifyBlockTrade(t.Context(), time.Now(), "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	info, err := e.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	require.NotNil(t, info)
	_, err = e.VerifyBlockTrade(t.Context(), time.Time{}, "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.ErrorIs(t, err, errZeroTimestamp)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.VerifyBlockTrade(t.Context(), time.Now(), "something", "maker", currency.EMPTYCODE, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      order.Buy.Lower(),
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSVerifyBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.WSVerifyBlockTrade(t.Context(), time.Now(), "", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingNonce)
	_, err = e.WSVerifyBlockTrade(t.Context(), time.Now(), "nonce-string", "", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = e.WSVerifyBlockTrade(t.Context(), time.Now(), "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	info, err := e.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	require.NotNil(t, info)
	_, err = e.WSVerifyBlockTrade(t.Context(), time.Time{}, "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.ErrorIs(t, err, errZeroTimestamp)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSVerifyBlockTrade(t.Context(), time.Now(), "sdjkafdad", "maker", currency.EMPTYCODE, []BlockTradeParam{
		{
			Price:          0.777 * 28000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInvalidateBlockTradeSignature(t *testing.T) {
	t.Parallel()
	err := e.WsInvalidateBlockTradeSignature(t.Context(), "")
	require.ErrorIs(t, err, errMissingSignature)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.InvalidateBlockTradeSignature(t.Context(), "verified_signature_string")
	assert.NoError(t, err)
}

func TestWsInvalidateBlockTradeSignature(t *testing.T) {
	t.Parallel()
	err := e.WsInvalidateBlockTradeSignature(t.Context(), "")
	require.ErrorIs(t, err, errMissingSignature)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.WsInvalidateBlockTradeSignature(t.Context(), "verified_signature_string")
	assert.NoError(t, err)
}

func TestExecuteBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteBlockTrade(t.Context(), time.Now(), "", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingNonce)
	_, err = e.ExecuteBlockTrade(t.Context(), time.Now(), "nonce-string", "", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = e.ExecuteBlockTrade(t.Context(), time.Now(), "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	info, err := e.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	require.NotNil(t, info)
	_, err = e.ExecuteBlockTrade(t.Context(), time.Time{}, "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.ErrorIs(t, err, errZeroTimestamp)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ExecuteBlockTrade(t.Context(), time.Now(), "something", "maker", currency.EMPTYCODE, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSExecuteBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.WSExecuteBlockTrade(t.Context(), time.Now(), "", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingNonce)
	_, err = e.WSExecuteBlockTrade(t.Context(), time.Now(), "nonce-string", "", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = e.WSExecuteBlockTrade(t.Context(), time.Now(), "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	info, err := e.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	require.NotNil(t, info)
	_, err = e.WSExecuteBlockTrade(t.Context(), time.Time{}, "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{{
		Price:          0.777 * 22000,
		InstrumentName: btcPerpInstrument,
		Direction:      "buy",
		Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
	}})
	require.ErrorIs(t, err, errZeroTimestamp)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSExecuteBlockTrade(t.Context(), time.Now(), "sdjkafdad", "maker", currency.EMPTYCODE, []BlockTradeParam{
		{
			Price:          0.777 * 22000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getUserBlocTradeResponseJSON = `[ { "trade_seq": 37, "trade_id": "92437", "timestamp": 1565089523719, "tick_direction": 3, "state": "filled", "price": 0.0001, "order_type": "limit", "order_id": "343062", "matching_id": null, "liquidity": "T", "iv": 0, "instrument_name": "BTC-9AUG19-10250-C", "index_price": 11738, "fee_currency": "BTC", "fee": 0.00025, "direction": "sell", "block_trade_id": "61", "amount": 10 }, { "trade_seq": 25350, "trade_id": "92435", "timestamp": 1565089523719, "tick_direction": 3, "state": "filled", "price": 11590, "order_type": "limit", "order_id": "343058", "matching_id": null, "liquidity": "T", "instrument_name": "BTC-PERPETUAL", "index_price": 11737.98, "fee_currency": "BTC", "fee": 0.00000164, "direction": "buy", "block_trade_id": "61", "amount": 190 } ]`

func TestGetUserBlocTrade(t *testing.T) {
	t.Parallel()
	var resp []BlockTradeData
	err := json.Unmarshal([]byte(getUserBlocTradeResponseJSON), &resp)
	require.NoError(t, err)
	_, err = e.GetUserBlockTrade(t.Context(), "")
	require.ErrorIs(t, err, errMissingBlockTradeID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserBlockTrade(t.Context(), "12345567")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveUserBlockTrade(t.Context(), "")
	require.ErrorIs(t, err, errMissingBlockTradeID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveUserBlockTrade(t.Context(), "12345567")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastBlockTradesbyCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastBlockTradesByCurrency(t.Context(), currency.EMPTYCODE, "", "", 5)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLastBlockTradesByCurrency(t.Context(), currency.SOL, "", "", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveLastBlockTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveLastBlockTradesByCurrency(t.Context(), currency.EMPTYCODE, "", "", 5)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WSRetrieveLastBlockTradesByCurrency(t.Context(), currency.SOL, "", "", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMovePositions(t *testing.T) {
	t.Parallel()
	_, err := e.MovePositions(t.Context(), currency.EMPTYCODE, 123, 345, []BlockTradeParam{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.MovePositions(t.Context(), currency.BTC, 0, 345, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingSubAccountID)
	_, err = e.MovePositions(t.Context(), currency.BTC, 123, 0, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingSubAccountID)
	_, err = e.MovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "",
			Direction:      "buy",
			Amount:         100,
		},
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = e.MovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         0,
		},
	})
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.MovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          -4,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         20,
		},
	})
	require.ErrorIs(t, err, errInvalidPrice)
	info, err := e.GetInstrument(t.Context(), "BTC-PERPETUAL")
	require.NoError(t, err)
	require.NotNil(t, info)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.MovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSMovePositions(t *testing.T) {
	t.Parallel()
	_, err := e.WSMovePositions(t.Context(), currency.EMPTYCODE, 123, 345, []BlockTradeParam{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WSMovePositions(t.Context(), currency.BTC, 0, 345, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingSubAccountID)
	_, err = e.WSMovePositions(t.Context(), currency.BTC, 123, 0, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingSubAccountID)
	_, err = e.WSMovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "",
			Direction:      "buy",
			Amount:         100,
		},
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = e.WSMovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         0,
		},
	})
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.WSMovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          -4,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         20,
		},
	})
	require.ErrorIs(t, err, errInvalidPrice)
	info, err := e.GetInstrument(t.Context(), "BTC-PERPETUAL")
	require.NoError(t, err)
	require.NotNil(t, info)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WSMovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSimulateBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.SimulateBlockTrade(t.Context(), "", []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = e.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	_, err = e.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "",
			Direction:      "buy",
			Amount:         10,
		},
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = e.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "",
			Amount:         10,
		},
	})
	require.ErrorIs(t, err, errInvalidOrderSideOrDirection)
	_, err = e.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "sell",
			Amount:         0,
		},
	})
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          -1,
			InstrumentName: btcPerpInstrument,
			Direction:      "sell",
			Amount:         10,
		},
	})
	require.ErrorIs(t, err, errInvalidPrice)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	info, err := e.GetInstrument(t.Context(), "BTC-PERPETUAL")
	require.NoError(t, err)
	result, err := e.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsSimulateBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.WsSimulateBlockTrade(t.Context(), "", []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = e.WsSimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	_, err = e.WsSimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "",
			Direction:      "buy",
			Amount:         10,
		},
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = e.WsSimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "",
			Amount:         100,
		},
	})
	require.ErrorIs(t, err, errInvalidOrderSideOrDirection)
	_, err = e.WsSimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "sell",
			Amount:         0,
		},
	})
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = e.WsSimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          -1,
			InstrumentName: btcPerpInstrument,
			Direction:      "sell",
			Amount:         100,
		},
	})
	require.ErrorIs(t, err, errInvalidPrice)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	info, err := e.GetInstrument(t.Context(), "BTC-PERPETUAL")
	require.NoError(t, err)
	require.NotNil(t, info)
	result, err := e.WsSimulateBlockTrade(t.Context(), "taker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func setupWs() {
	if !e.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := e.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := e.generateSubscriptions()
	require.NoError(t, err)
	exp := subscription.List{}
	for _, s := range e.Features.Subscriptions {
		for _, a := range e.GetAssetTypes(true) {
			if !e.IsAssetWebsocketSupported(a) {
				continue
			}
			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.Asset = a
			if isSymbolChannel(s) {
				for i, p := range pairs {
					s := s.Clone() //nolint:govet // Intentional lexical scope shadow
					s.QualifiedChannel = channelName(s) + "." + p.String()
					if s.Interval != 0 {
						s.QualifiedChannel += "." + channelInterval(s)
					}
					s.Pairs = pairs[i : i+1]
					exp = append(exp, s)
				}
			} else {
				s.Pairs = pairs
				s.QualifiedChannel = channelName(s)
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

func TestChannelInterval(t *testing.T) {
	t.Parallel()

	for _, i := range []int64{1, 3, 5, 10, 15, 30, 60, 120, 180, 360, 720} {
		a := channelInterval(&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.Interval(i * int64(time.Minute))})
		assert.Equal(t, strconv.Itoa(int(i)), a)
	}

	a := channelInterval(&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneDay})
	assert.Equal(t, "1D", a)

	assert.Panics(t, func() {
		channelInterval(&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneMonth})
	})

	a = channelInterval(&subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.ThousandMilliseconds})
	assert.Equal(t, "agg2", a, "1 second should expand to agg2")

	a = channelInterval(&subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds})
	assert.Equal(t, "100ms", a, "100ms should expand correctly")

	a = channelInterval(&subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.Raw})
	assert.Equal(t, "raw", a, "raw should expand correctly")

	assert.Panics(t, func() {
		channelInterval(&subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.OneMonth})
	})

	a = channelInterval(&subscription.Subscription{Channel: userAccessLogChannel})
	assert.Empty(t, a, "Anything else should return empty")
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, tickerChannel, channelName(&subscription.Subscription{Channel: subscription.TickerChannel}))
	assert.Equal(t, userLockChannel, channelName(&subscription.Subscription{Channel: userLockChannel}))
	assert.Panics(t, func() { channelName(&subscription.Subscription{Channel: "wibble"}) }, "Unknown channels should panic")
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UpdateAccountBalances(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Empty)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	for assetType, cp := range assetTypeToPairsMap {
		t.Run(fmt.Sprintf("%s %s", assetType, cp), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRecentTrades(t.Context(), cp, assetType)
			require.NoError(t, err, "GetRecentTrades must not error")
			require.NotNil(t, result, "result must not be nil")
		})
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      futuresTradablePair,
		AssetType: asset.Futures,
	}
	var result order.CancelAllResponse
	var err error
	for assetType, cp := range assetTypeToPairsMap {
		orderCancellation.AssetType = assetType
		orderCancellation.Pair = cp
		result, err = e.CancelAllOrders(t.Context(), orderCancellation)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	for assetType, cp := range assetTypeToPairsMap {
		result, err := e.GetOrderInfo(t.Context(), "1234", cp, assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
	require.ErrorIs(t, err, common.ErrNoResponse)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:    e.Name,
		Amount:      -0.1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: "0x1nv4l1d",
			Chain:   "tetheruse",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	getOrdersRequest := order.MultiOrderRequest{
		Type: order.AnyType, AssetType: asset.Futures,
		Side: order.AnySide, Pairs: currency.Pairs{futuresTradablePair},
	}

	for assetType, cp := range assetTypeToPairsMap {
		getOrdersRequest.Pairs = []currency.Pair{cp}
		getOrdersRequest.AssetType = assetType
		result, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	for assetType, cp := range assetTypeToPairsMap {
		result, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			Type: order.AnyType, AssetType: assetType,
			Side: order.AnySide, Pairs: []currency.Pair{cp},
		})
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetAssetPairByInstrument(t *testing.T) {
	t.Parallel()
	for _, assetType := range []asset.Item{asset.Spot, asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo} {
		availablePairs, err := e.GetAvailablePairs(assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s", err, assetType)
		require.NotNilf(t, availablePairs, "expected result not to be nil for asset type %s", assetType)
		for _, cp := range availablePairs {
			instrument := formatPairString(assetType, cp)
			t.Run(fmt.Sprintf("%s %s", assetType, instrument), func(t *testing.T) {
				t.Parallel()
				extractedAsset, extractedPair, err := getAssetPairByInstrument(instrument)
				assert.NoError(t, err)
				fPair, err := e.FormatExchangeCurrency(extractedPair, assetType)
				require.NoError(t, err, "FormatExchangeCurrency must not error")
				assert.Equal(t, cp.String(), fPair.String())
				assert.Equal(t, assetType.String(), extractedAsset.String(), "asset should match for")
			})
		}
	}
	t.Run("empty asset, empty pair", func(t *testing.T) {
		t.Parallel()
		_, _, err := getAssetPairByInstrument("")
		assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	})
	t.Run("thisIsAFakeCurrency", func(t *testing.T) {
		t.Parallel()
		_, _, err := getAssetPairByInstrument("thisIsAFakeCurrency")
		assert.ErrorIs(t, err, errUnsupportedInstrumentFormat)
	})
}

func TestGetAssetFromInstrument(t *testing.T) {
	t.Parallel()
	tc := []struct {
		instrument    string
		expectedAsset asset.Item
		expectedError error
	}{
		{"BNB_USDC", asset.Spot, nil},
		{"BTC-30DEC22", asset.Futures, nil},
		{"BTCDVOL_USDC-1OCT25", asset.Futures, nil},
		{"ADA_USDC-PERPETUAL", asset.Futures, nil},
		{"PAXG_USDC-12SEP25-3320-P", asset.Options, nil},
		{"ETH-3OCT25-4800-P", asset.Options, nil},
		{"ETH-FS-26JUN26_26DEC25", asset.FutureCombo, nil},
		{"BTC-FS-28NOV25_PERP", asset.FutureCombo, nil},
		{"BTC-USDC-FS-28NOV25_PERP", asset.FutureCombo, nil},
		{"BTC_USDC-PBUT-31OCT25-90000_100000_102000", asset.OptionCombo, nil},
		{"BTC_USDC-CS-31OCT25-107000_111000", asset.OptionCombo, nil},
		{"BTC-ICOND-14NOV25-100000_105000_125000_130000", asset.OptionCombo, nil},
		{"BTC-PCAL-14NOV25_7NOV25-112000", asset.OptionCombo, nil},
		{"XRP_USDC-CBUT-26SEP25-2d9_3d2_3d4", asset.OptionCombo, nil},
		{"ETH-CS-26SEP25-5000_5500", asset.OptionCombo, nil},
		{"HELLOMOTO", asset.Empty, errUnsupportedInstrumentFormat},
		{"hi-my-name-is-moto", asset.Empty, errUnsupportedInstrumentFormat},
	}
	for _, test := range tc {
		t.Run(test.instrument, func(t *testing.T) {
			t.Parallel()
			a, err := getAssetFromInstrument(test.instrument)
			if test.expectedError != nil {
				assert.ErrorIs(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedAsset, a)
			}
		})
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                futuresTradablePair,
		IsMaker:             false,
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
	result, err := e.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		assert.Equalf(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equalf(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType, "expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
	}
}

func TestCalculateTradingFee(t *testing.T) {
	t.Parallel()
	feeBuilder := &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.Pair{Base: currency.BTC, Quote: currency.USD, Delimiter: currency.DashDelimiter},
		IsMaker:       true,
		Amount:        1,
		PurchasePrice: 1000,
	}
	var result float64
	result, err := calculateTradingFee(feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equalf(t, 1e-1, result, "expected result %f, got %f", 1e-1, result)
	// futures
	feeBuilder.Pair, err = currency.NewPairFromString("BTC-21OCT22")
	require.NoError(t, err)
	result, err = calculateTradingFee(feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equalf(t, 0.1, result, "expected 0.1 but found %f", result)
	// options
	feeBuilder.Pair, err = currency.NewPairFromString("SOL-21OCT22-20-C")
	require.NoError(t, err)
	feeBuilder.IsMaker = false
	result, err = calculateTradingFee(feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equalf(t, 0.3, result, "expected 0.3 but found %f", result)
	// options
	feeBuilder.Pair, err = currency.NewPairFromString("SOL-21OCT22-20-C,SOL-21OCT22-20-P")
	require.NoError(t, err)
	feeBuilder.IsMaker = true
	_, err = calculateTradingFee(feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equalf(t, 0.3, result, "expected 0.3 but found %f", result)
	// option_combo
	feeBuilder.Pair, err = currency.NewPairFromString("BTC-STRG-21OCT22-19000_21000")
	require.NoError(t, err)
	feeBuilder.IsMaker = false
	result, err = calculateTradingFee(feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
	// future_combo
	feeBuilder.Pair, err = currency.NewPairFromString("SOL-FS-30DEC22_28OCT22")
	require.NoError(t, err)
	feeBuilder.IsMaker = false
	result, err = calculateTradingFee(feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
	feeBuilder.Pair, err = currency.NewPairFromString("some_instrument_builder")
	require.NoError(t, err)
	require.NotNil(t, result)
	_, err = calculateTradingFee(feeBuilder)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetTime(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetServerTime(t.Context(), asset.Empty)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	modifyParamToErrorMap := map[*order.Modify]error{
		{OrderID: "1234"}:       order.ErrPairIsEmpty,
		{AssetType: asset.Spot}: order.ErrPairIsEmpty,
		{AssetType: asset.Margin, OrderID: "1234", Pair: spotTradablePair}:     asset.ErrNotSupported,
		{AssetType: asset.Futures, OrderID: "1234", Pair: futuresTradablePair}: errInvalidAmount,
		{AssetType: asset.Futures, Pair: futuresTradablePair, Amount: 2}:       order.ErrOrderIDNotSet,
	}
	for param, errIncoming := range modifyParamToErrorMap {
		_, err := e.ModifyOrder(t.Context(), param)
		require.ErrorIs(t, err, errIncoming)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Futures, OrderID: "1234", Pair: futuresTradablePair, Amount: 2})
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Options, OrderID: "1234", Pair: optionsTradablePair, Amount: 2})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
	}
	for assetType, cp := range assetTypeToPairsMap {
		orderCancellation.AssetType = assetType
		orderCancellation.Pair = cp
		err := e.CancelOrder(t.Context(), orderCancellation)
		require.NoError(t, err)
	}
}

var websocketPushData = map[string]string{
	"Announcement":                           `{"jsonrpc": "2.0","method": "subscription","params": {"channel": "announcements","data": {            "action": "new",            "body": "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",            "id": 1532593832021,            "important": true,            "publication_timestamp": 1532593832021,            "title": "Example announcement"        }    }}`,
	"Orderbook":                              `{"jsonrpc": "2.0", "method": "subscription", "params": { "channel": "book.BTC-PERPETUAL.100ms", "data": { "type": "snapshot", "timestamp": 1677589058217, "instrument_name": "BTC-PERPETUAL", "change_id": 53639437695, "bids": [ [ "new", 23461.0, 47800.0 ], [ "new", 23460.5, 37820.0 ], [ "new", 23460.0, 45720.0 ], [ "new", 23459.5, 24030.0 ], [ "new", 23459.0, 63600.0 ], [ "new", 23458.5, 60480.0 ], [ "new", 23458.0, 7960.0 ], [ "new", 23457.5, 2310.0 ], [ "new", 23457.0, 4270.0 ], [ "new", 23456.5, 44070.0 ], [ "new", 23456.0, 88690.0 ], [ "new", 23455.5, 5650.0 ], [ "new", 23455.0, 13420.0 ], [ "new", 23454.5, 116710.0 ], [ "new", 23454.0, 2010.0 ], [ "new", 23453.5, 200000.0 ], [ "new", 23452.5, 19950.0 ], [ "new", 23452.0, 39360.0 ], [ "new", 23451.5, 10000.0 ], [ "new", 23451.0, 239510.0 ], [ "new", 23450.5, 6250.0 ], [ "new", 23450.0, 40080.0 ], [ "new", 23449.5, 2000.0 ], [ "new", 23448.5, 500.0 ], [ "new", 23447.5, 179810.0 ], [ "new", 23447.0, 11000.0 ], [ "new", 23446.0, 57730.0 ], [ "new", 23445.0, 3640.0 ], [ "new", 23444.0, 17640.0 ], [ "new", 23443.5, 50000.0 ], [ "new", 23443.0, 6250.0 ], [ "new", 23441.5, 30330.0 ], [ "new", 23440.5, 76990.0 ], [ "new", 23440.0, 23910.0 ], [ "new", 23439.5, 3000.0 ], [ "new", 23439.0, 990.0 ], [ "new", 23438.0, 20760.0 ], [ "new", 23437.5, 500.0 ], [ "new", 23437.0, 84970.0 ], [ "new", 23436.5, 30040.0 ], [ "new", 23435.5, 322380.0 ], [ "new", 23434.0, 86280.0 ], [ "new", 23433.5, 187860.0 ], [ "new", 23433.0, 102360.0 ], [ "new", 23432.5, 48250.0 ], [ "new", 23432.0, 29070.0 ], [ "new", 23430.0, 119780.0 ], [ "new", 23429.0, 10.0 ], [ "new", 23428.0, 1510.0 ], [ "new", 23427.5, 2000.0 ], [ "new", 23427.0, 10.0 ], [ "new", 23426.5, 1840.0 ], [ "new", 23425.5, 2000.0 ], [ "new", 23425.0, 2250.0 ], [ "new", 23424.5, 600000.0 ], [ "new", 23424.0, 40870.0 ], [ "new", 23423.0, 117200.0 ], [ "new", 23422.0, 5000.0 ], [ "new", 23421.5, 80970.0 ], [ "new", 23420.0, 2420.0 ], [ "new", 23419.5, 200.0 ], [ "new", 23418.5, 40000.0 ], [ "new", 23415.0, 8020.0 ], [ "new", 23414.5, 57730.0 ], [ "new", 23413.5, 133250.0 ], [ "new", 23412.0, 40000.0 ], [ "new", 23410.5, 24000.0 ], [ "new", 23410.0, 80.0 ], [ "new", 23408.0, 36000.0 ], [ "new", 23407.0, 550000.0 ], [ "new", 23406.0, 30.0 ], [ "new", 23404.5, 230.0 ], [ "new", 23402.5, 57730.0 ], [ "new", 23401.0, 300010.0 ], [ "new", 23400.0, 520.0 ], [ "new", 23398.0, 28980.0 ], [ "new", 23394.5, 10.0 ], [ "new", 23391.5, 200.0 ], [ "new", 23391.0, 150000.0 ], [ "new", 23390.0, 80.0 ], [ "new", 23387.0, 403640.0 ], [ "new", 23385.5, 110.0 ], [ "new", 23385.0, 50.0 ], [ "new", 23384.5, 4690.0 ], [ "new", 23381.0, 200.0 ], [ "new", 23101.0, 9240.0 ], [ "new", 23100.5, 2320.0 ], [ "new", 23100.0, 15360.0 ], [ "new", 23096.0, 3000.0 ], [ "new", 23090.0, 90.0 ], [ "new", 23088.0, 3000.0 ], [ "new", 23087.0, 60.0 ], [ "new", 23081.5, 100.0 ], [ "new", 23080.0, 5400.0 ], [ "new", 23072.0, 3000.0 ], [ "new", 23070.0, 80.0 ], [ "new", 23064.0, 3000.0 ], [ "new", 23062.0, 3270.0 ], [ "new", 23060.0, 80.0 ], [ "new", 23056.0, 98000.0 ], [ "new", 23053.0, 3500.0 ], [ "new", 23050.5, 2370.0 ], [ "new", 23050.0, 32510.0 ], [ "new", 23048.0, 3000.0 ], [ "new", 23040.0, 3080.0 ], [ "new", 23038.0, 1000.0 ], [ "new", 23032.0, 5310.0 ], [ "new", 23030.0, 100.0 ], [ "new", 23024.0, 29000.0 ], [ "new", 23021.0, 2080.0 ], [ "new", 23020.0, 80.0 ], [ "new", 23016.0, 4150.0 ], [ "new", 23010.0, 80.0 ], [ "new", 23008.0, 3000.0 ], [ "new", 23005.0, 80.0 ], [ "new", 23004.5, 79200.0 ], [ "new", 23002.0, 20470.0 ], [ "new", 23001.0, 1000.0 ], [ "new", 23000.0, 8940.0 ], [ "new", 22992.0, 3000.0 ], [ "new", 22990.0, 2080.0 ], [ "new", 22984.0, 3000.0 ], [ "new", 22980.5, 2320.0 ], [ "new", 22980.0, 80.0 ], [ "new", 22976.0, 3000.0 ], [ "new", 22975.0, 52000.0 ], [ "new", 22971.0, 3600.0 ], [ "new", 22970.0, 2400.0 ], [ "new", 22968.0, 3000.0 ], [ "new", 22965.0, 270.0 ], [ "new", 22960.0, 3080.0 ], [ "new", 22956.0, 1000.0 ], [ "new", 22952.0, 3000.0 ], [ "new", 22951.0, 60.0 ], [ "new", 22950.0, 40200.0 ], [ "new", 22949.0, 1500.0 ], [ "new", 22944.0, 3000.0 ], [ "new", 22936.0, 3000.0 ], [ "new", 22934.0, 3000.0 ], [ "new", 22928.0, 3000.0 ], [ "new", 22925.0, 2370.0 ], [ "new", 22922.0, 80.0 ], [ "new", 22920.0, 3000.0 ], [ "new", 22916.0, 1150.0 ], [ "new", 22912.0, 3000.0 ], [ "new", 22904.5, 220.0 ], [ "new", 22904.0, 3000.0 ], [ "new", 22900.0, 273290.0 ], [ "new", 22896.0, 3000.0 ], [ "new", 22889.5, 100.0 ], [ "new", 22888.0, 7580.0 ], [ "new", 22880.0, 683400.0 ], [ "new", 22875.0, 400.0 ], [ "new", 22872.0, 3000.0 ], [ "new", 22870.0, 100.0 ], [ "new", 22864.0, 3000.0 ], [ "new", 22860.0, 2320.0 ], [ "new", 22856.0, 3000.0 ], [ "new", 22854.0, 10.0 ], [ "new", 22853.0, 500.0 ], [ "new", 22850.0, 1020.0 ], [ "new", 22848.0, 3000.0 ], [ "new", 22844.0, 25730.0 ], [ "new", 22840.0, 3000.0 ], [ "new", 22834.0, 3000.0 ], [ "new", 22832.0, 3000.0 ], [ "new", 22831.0, 200.0 ], [ "new", 22827.0, 40120.0 ], [ "new", 22824.0, 3000.0 ], [ "new", 22816.0, 4140.0 ], [ "new", 22808.0, 3000.0 ], [ "new", 22804.5, 220.0 ], [ "new", 22802.0, 50.0 ], [ "new", 22801.0, 1150.0 ], [ "new", 22800.0, 14050.0 ], [ "new", 22797.0, 10.0 ], [ "new", 22792.0, 3000.0 ], [ "new", 22789.0, 3000.0 ], [ "new", 22787.5, 5000.0 ], [ "new", 22784.0, 3000.0 ], [ "new", 22776.0, 3000.0 ], [ "new", 22775.0, 10000.0 ], [ "new", 22770.0, 200.0 ], [ "new", 22768.0, 14380.0 ], [ "new", 22760.0, 3000.0 ], [ "new", 22756.5, 2370.0 ], [ "new", 22752.0, 3000.0 ], [ "new", 22751.0, 47780.0 ], [ "new", 22750.0, 59970.0 ], [ "new", 22749.0, 50.0 ], [ "new", 22744.0, 3000.0 ], [ "new", 22736.0, 3000.0 ], [ "new", 22728.0, 3000.0 ], [ "new", 22726.0, 2320.0 ], [ "new", 22725.0, 20000.0 ], [ "new", 22720.0, 3000.0 ], [ "new", 22713.5, 250.0 ], [ "new", 22712.0, 3000.0 ], [ "new", 22709.0, 25000.0 ], [ "new", 22704.5, 220.0 ], [ "new", 22704.0, 3000.0 ], [ "new", 22702.0, 50.0 ], [ "new", 22700.0, 10230.0 ], [ "new", 22697.5, 10.0 ], [ "new", 22696.0, 3000.0 ], [ "new", 22688.0, 3000.0 ], [ "new", 22684.0, 10.0 ], [ "new", 22680.0, 3000.0 ], [ "new", 22672.0, 3000.0 ], [ "new", 22667.0, 2270.0 ], [ "new", 22664.0, 3000.0 ], [ "new", 22662.5, 2320.0 ], [ "new", 22657.5, 2340.0 ], [ "new", 22656.0, 3000.0 ], [ "new", 22655.0, 50.0 ], [ "new", 22653.0, 500.0 ], [ "new", 22650.0, 360120.0 ], [ "new", 22648.0, 3000.0 ], [ "new", 22640.0, 5320.0 ], [ "new", 22635.5, 2350.0 ], [ "new", 22632.0, 3000.0 ], [ "new", 22628.5, 2000.0 ], [ "new", 22626.5, 2350.0 ], [ "new", 22625.0, 400.0 ], [ "new", 22624.0, 3000.0 ], [ "new", 22616.0, 3000.0 ], [ "new", 22608.0, 3000.0 ], [ "new", 22604.5, 220.0 ], [ "new", 22601.0, 22600.0 ], [ "new", 22600.0, 696120.0 ], [ "new", 22598.5, 2320.0 ], [ "new", 22592.0, 3000.0 ], [ "new", 22584.0, 3000.0 ], [ "new", 22576.0, 3000.0 ], [ "new", 22568.0, 3000.0 ], [ "new", 22560.0, 25560.0 ], [ "new", 22552.5, 20.0 ], [ "new", 22550.0, 35760.0 ], [ "new", 22533.0, 2320.0 ], [ "new", 22530.0, 2320.0 ], [ "new", 22520.0, 1000.0 ], [ "new", 22505.0, 20000.0 ], [ "new", 22504.5, 220.0 ], [ "new", 22501.0, 45000.0 ], [ "new", 22500.0, 27460.0 ], [ "new", 22497.5, 1500.0 ], [ "new", 22485.0, 810.0 ], [ "new", 22481.0, 300.0 ], [ "new", 22465.5, 2320.0 ], [ "new", 22456.0, 2350.0 ], [ "new", 22453.0, 500.0 ], [ "new", 22450.0, 25000.0 ], [ "new", 22433.0, 141000.0 ], [ "new", 22431.0, 1940.0 ], [ "new", 22420.0, 2320.0 ], [ "new", 22419.5, 1000000.0 ], [ "new", 22400.0, 14280.0 ], [ "new", 22388.5, 30.0 ], [ "new", 22381.0, 100.0 ] ] } } }`,
	"Orderbook Update":                       `{"params" : {"data" : {"type" : "snapshot","timestamp" : 1554373962454,"instrument_name" : "BTC-PERPETUAL","change_id" : 297217,"bids" : [["new",5042.34,30],["new",5041.94,20]],"asks" : [["new",5042.64,40],["new",5043.3,40]]},"channel" : "book.BTC-PERPETUAL.100ms"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Candlestick":                            `{"params" : {"data" : {"volume" : 0.05219351,"tick" : 1573645080000,"open" : 8869.79,"low" : 8788.25,"high" : 8870.31,"cost" : 460,"close" : 8791.25},"channel" : "chart.trades.BTC-PERPETUAL.1"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Index Price":                            `{"params" : {"data" : {"timestamp" : 1550588002899,"price" : 3937.89,"index_name" : "btc_usd"},"channel" : "deribit_price_index.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Price Ranking":                          `{"params" : {"data" :[{"weight" : 14.29,"timestamp" : 1573202284040,"price" : 9109.35,"original_price" : 9109.35,"identifier" : "bitfinex","enabled" : true},{"weight" : 14.29,"timestamp" : 1573202284055,"price" : 9084.83,"original_price" : 9084.83,"identifier" : "bitstamp","enabled" : true		},		{		  "weight" : 14.29,"timestamp" : 1573202283191,"price" : 9079.91,"original_price" : 9079.91,"identifier" : "bittrex","enabled" : true		},		{		  "weight" : 14.29,"timestamp" : 1573202284094,"price" : 9085.81,"original_price" : 9085.81,"identifier" : "coinbase","enabled" : true		},		{		  "weight" : 14.29,"timestamp" : 1573202283881,"price" : 9086.27,"original_price" : 9086.27,"identifier" : "gemini","enabled" : true		},		{		  "weight" : 14.29,"timestamp" : 1573202283420,"price" : 9088.38,"original_price" : 9088.38,"identifier" : "itbit","enabled" : true		},		{		  "weight" : 14.29,"timestamp" : 1573202283459,"price" : 9083.6,"original_price" : 9083.6,"identifier" : "kraken","enabled" : true		},		{		  "weight" : 0,"timestamp" : 0,"price" : null,"original_price" : null,"identifier" : "lmax","enabled" : false		}	  ],	  "channel" : "deribit_price_ranking.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Price Statistics":                       `{"params" : {"data" : {"low24h" : 58012.08,"index_name" : "btc_usd","high_volatility" : false,"high24h" : 59311.42,"change24h" : 1009.61},"channel" : "deribit_price_statistics.btc_usd"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Volatility Index":                       `{"params" : {"data" : {"volatility" : 129.36,"timestamp" : 1619777946007,"index_name" : "btc_usd","estimated_delivery" : 129.36},"channel" : "deribit_volatility_index.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Estimated Expiration Price":             `{"params" : {"data" : {"seconds" : 180929,"price" : 3939.73,"is_estimated" : false},"channel" : "estimated_expiration_price.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Incremental Ticker":                     `{"jsonrpc": "2.0", "method": "subscription", "params": { "channel": "incremental_ticker.BTC-PERPETUAL", "data": { "type": "snapshot", "timestamp": 1677592580023, "stats": { "volume_usd": 224579520.0, "volume": 9581.70741368, "price_change": -1.2945, "low": 23123.5, "high": 23900.0 }, "state": "open", "settlement_price": 23240.71, "open_interest": 333091400, "min_price": 23057.4, "max_price": 23759.65, "mark_price": 23408.41, "last_price": 23409.0, "interest_value": 0.0, "instrument_name": "BTC-PERPETUAL", "index_price": 23406.85, "funding_8h": 0.0, "estimated_delivery_price": 23406.85, "current_funding": 0.0, "best_bid_price": 23408.5, "best_bid_amount": 53270.0, "best_ask_price": 23409.0, "best_ask_amount": 46990.0 } } }`,
	"Instrument State":                       `{"params" : {"data" : {"timestamp" : 1553080940000,"state" : "created","instrument_name" : "BTC-22MAR19"},"channel" : "instrument.state.any.any"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Currency Trades":                        `{"params":{"data":[{"trade_seq":2,"trade_id" : "48079289","timestamp" : 1590484589306,"tick_direction" : 2,"price" : 0.0075,"mark_price" : 0.01062686,"iv" : 47.58,"instrument_name" : "BTC-27MAY20-9000-C","index_price" : 8956.17,"direction" : "sell","amount" : 3}],"channel" : "trades.option.BTC.raw"},"method":"subscription","jsonrpc":"2.0"}`,
	"Change Updates":                         `{"params" : {"data" : {"trades" : [{"trade_seq" : 866638,"trade_id" : "1430914","timestamp" : 1605780344032,"tick_direction" : 1,"state" : "filled","self_trade" : false,"reduce_only" : false,"profit_loss" : 0.00004898,"price" : 17391,"post_only" : false,"order_type" : "market","order_id" : "3398016","matching_id" : null,"mark_price" : 17391,"liquidity" : "T","instrument_name" : "BTC-PERPETUAL","index_price" : 17501.88,"fee_currency" : "BTC","fee" : 1.6e-7,"direction" : "sell","amount" : 10		  }		],"positions" : [		  {			"total_profit_loss" : 1.69711368,			"size_currency" : 10.646886321,			"size" : 185160,			"settlement_price" : 16025.83,			"realized_profit_loss" : 0.012454598,			"realized_funding" : 0.01235663,			"open_orders_margin" : 0,			"mark_price" : 17391,			"maintenance_margin" : 0.234575865,			"leverage" : 33,			"kind" : "future",			"interest_value" : 1.7362511643080387,			"instrument_name" : "BTC-PERPETUAL",			"initial_margin" : 0.319750953,			"index_price" : 17501.88,			"floating_profit_loss" : 0.906961435,			"direction" : "buy",			"delta" : 10.646886321,			"average_price" : 15000		  }		],"orders" : [		  {			"web" : true,			"time_in_force" : "good_til_cancelled",			"replaced" : false,			"reduce_only" : false,			"profit_loss" : 0.00009166,			"price" : 15665.5,			"post_only" : false,			"order_type" : "market",			"order_state" : "filled",			"order_id" : "3398016",			"max_show" : 10,			"last_update_timestamp" : 1605780344032,			"label" : "",			"is_liquidation" : false,			"instrument_name" : "BTC-PERPETUAL",			"filled_amount" : 10,			"direction" : "sell",			"creation_timestamp" : 1605780344032,			"commission" : 1.6e-7,			"average_price" : 17391,			"api" : false,			"amount" : 10}],"instrument_name" : "BTC-PERPETUAL"	  },	  "channel" : "user.changes.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Currency Changes Updates":               `{"params" : {"data" : {"trades" : [{"trade_seq" : 866638,"trade_id" : "1430914","timestamp" : 1605780344032,"tick_direction" : 1,"state" : "filled","self_trade" : false,"reduce_only" : false,"profit_loss" : 0.00004898,"price" : 17391,"post_only" : false,"order_type" : "market","order_id" : "3398016","matching_id" : null,"mark_price" : 17391,"liquidity" : "T","instrument_name" : "BTC-PERPETUAL","index_price" : 17501.88,"fee_currency" : "BTC","fee" : 1.6e-7,"direction" : "sell","amount" : 10		  }		],"positions" : [		  {			"total_profit_loss" : 1.69711368,			"size_currency" : 10.646886321,			"size" : 185160,			"settlement_price" : 16025.83,			"realized_profit_loss" : 0.012454598,			"realized_funding" : 0.01235663,			"open_orders_margin" : 0,			"mark_price" : 17391,			"maintenance_margin" : 0.234575865,			"leverage" : 33,			"kind" : "future",			"interest_value" : 1.7362511643080387,			"instrument_name" : "BTC-PERPETUAL",			"initial_margin" : 0.319750953,			"index_price" : 17501.88,			"floating_profit_loss" : 0.906961435,			"direction" : "buy",			"delta" : 10.646886321,			"average_price" : 15000		  }		],"orders" : [		  {			"web" : true,			"time_in_force" : "good_til_cancelled",			"replaced" : false,			"reduce_only" : false,			"profit_loss" : 0.00009166,			"price" : 15665.5,			"post_only" : false,			"order_type" : "market",			"order_state" : "filled",			"order_id" : "3398016",			"max_show" : 10,			"last_update_timestamp" : 1605780344032,			"label" : "",			"is_liquidation" : false,			"instrument_name" : "BTC-PERPETUAL",			"filled_amount" : 10,			"direction" : "sell",			"creation_timestamp" : 1605780344032,			"commission" : 1.6e-7,			"average_price" : 17391,			"api" : false,			"amount" : 10		  }		],"instrument_name" : "BTC-PERPETUAL"	  },	  "channel" : "user.changes.future.BTC.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Orders Raw Instrument":             `{"params" : {"data" : {"time_in_force" : "good_til_cancelled","replaced" : false,"reduce_only" : false,"profit_loss" : 0,"price" : 10502.52,"post_only" : false,"original_order_type" : "market","order_type" : "limit","order_state" : "open","order_id" : "5","max_show" : 200,"last_update_timestamp" : 1581507423789,"label" : "","is_liquidation" : false,"instrument_name" : "BTC-PERPETUAL","filled_amount" : 0,"direction" : "buy","creation_timestamp" : 1581507423789,"commission" : 0,"average_price" : 0,"api" : false,"amount" : 200	  },	  "channel" : "user.orders.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Orders By Instrument WithInterval": `{"params" : {"data" : [{"time_in_force" : "good_til_cancelled","replaced" : false,"reduce_only" : false,"profit_loss" : 0,"price" : 10460.43,"post_only" : false,"original_order_type" : "market","order_type" : "limit","order_state" : "open","order_id" : "4","max_show" : 200,"last_update_timestamp" : 1581507159533,"label" : "","is_liquidation" : false,"instrument_name" : "BTC-PERPETUAL","filled_amount" : 0,"direction" : "buy","creation_timestamp" : 1581507159533,"commission" : 0,"average_price" : 0,"api" : false,"amount" : 200		}	  ],	  "channel" : "user.orders.BTC-PERPETUAL.100ms"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Order By Currency Raw":             `{"params" : {"data" : {"time_in_force" : "good_til_cancelled","replaced" : false,"reduce_only" : false,"profit_loss" : 0,"price" : 10542.68,"post_only" : false,"original_order_type" : "market","order_type" : "limit","order_state" : "open","order_id" : "6","max_show" : 200,"last_update_timestamp" : 1581507583024,"label" : "","is_liquidation" : false,"instrument_name" : "BTC-PERPETUAL","filled_amount" : 0,"direction" : "buy","creation_timestamp" : 1581507583024,"commission" : 0,"average_price" : 0,"api" : false,"amount" : 200	  },	  "channel" : "user.orders.any.any.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Order By Currency WithInterval":    `{"params" : {"data" : [{"time_in_force" : "good_til_cancelled","reduce_only" : false,"profit_loss" : 0,"price" : 3928.5,"post_only" : false,"order_type" : "limit","order_state" : "open","order_id" : "476137","max_show" : 120,"last_update_timestamp" : 1550826337209,"label" : "","is_liquidation" : false,"instrument_name" : "BTC-PERPETUAL","filled_amount" : 0,"direction" : "buy","creation_timestamp" : 1550826337209,"commission" : 0,"average_price" : 0,"api" : false,"amount" : 120		}	  ],	  "channel" : "user.orders.future.BTC.100ms"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Portfolio":                         `{"params" : {"data" : {"total_pl" : 0.00000425,"session_upl" : 0.00000425,"session_rpl" : -2e-8,"projected_maintenance_margin" : 0.00009141,"projected_initial_margin" : 0.00012542,"projected_delta_total" : 0.0043,"portfolio_margining_enabled" : false,"options_vega" : 0,"options_value" : 0,"options_theta" : 0,"options_session_upl" : 0,"options_session_rpl" : 0,"options_pl" : 0,"options_gamma" : 0,"options_delta" : 0,"margin_balance" : 0.2340038,"maintenance_margin" : 0.00009141,"initial_margin" : 0.00012542,"futures_session_upl" : 0.00000425,"futures_session_rpl" : -2e-8,"futures_pl" : 0.00000425,"estimated_liquidation_ratio" : 0.01822795,"equity" : 0.2340038,"delta_total" : 0.0043,"currency" : "BTC","balance" : 0.23399957,"available_withdrawal_funds" : 0.23387415,"available_funds" : 0.23387838	  },	  "channel" : "user.portfolio.btc"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Trades":                            `{"params" : {"data" : [{"trade_seq" :30289432,"trade_id":"48079254","timestamp":1590484156350,"tick_direction" : 0,"state" : "filled","self_trade" : false,"reduce_only" : false,"price" : 8954,"post_only" : false,"order_type" : "market","order_id" : "4008965646","matching_id" : null,"mark_price" : 8952.86,"liquidity" : "T","instrument_name" : "BTC-PERPETUAL","index_price" : 8956.73,"fee_currency" : "BTC","fee" : 0.00000168,"direction" : "sell","amount" : 20		},		{		  "trade_seq" : 30289433,"trade_id" : "48079255","timestamp" : 1590484156350,"tick_direction" : 1,"state" : "filled","self_trade" : false,"reduce_only" : false,"price" : 8954,"post_only" : false,"order_type" : "market","order_id" : "4008965646","matching_id" : null,"mark_price" : 8952.86,"liquidity" : "T","instrument_name" : "BTC-PERPETUAL","index_price" : 8956.73,"fee_currency" : "BTC","fee" : 0.00000168,"direction" : "sell","amount" : 20	}],"channel" : "user.trades.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Trades With Currency":              `{"params" : {"data" : [{"trade_seq" :74405,	"trade_id":"48079262","timestamp":1590484255886,"tick_direction" : 2,"state" : "filled","self_trade" : false,"reduce_only" : false,"price" : 8947,"post_only" : false,"order_type" : "limit","order_id" : "4008978075","matching_id" : null,"mark_price" : 8970.03,"liquidity" : "T","instrument_name" : "BTC-25SEP20","index_price" : 8953.53,"fee_currency" : "BTC","fee" : 0.00049961,"direction" : "sell","amount" : 8940		}	  ],	  "channel" : "user.trades.future.BTC.100ms"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Mark Price Options":                     `{"params" : {"data" : [{"timestamp" : 1622470378005,"mark_price" : 0.0333,"iv" : 0.9,"instrument_name" : "BTC-2JUN21-37000-P"},{"timestamp" : 1622470378005,"mark_price" : 0.117,"iv" : 0.9,"instrument_name" : "BTC-4JUN21-40500-P"},{"timestamp" : 1622470378005,"mark_price" : 0.0177,"iv" : 0.9,"instrument_name" : "BTC-4JUN21-38250-C"},{"timestamp" : 1622470378005,"mark_price" : 0.0098,"iv" : 0.9,"instrument_name" : "BTC-1JUN21-37000-C"		},		{		  "timestamp" : 1622470378005,"mark_price" : 0.0371,"iv" : 0.9,"instrument_name" : "BTC-4JUN21-36500-P"		}	  ],	  "channel" : "markprice.options.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Platform State":                         `{"params" : {"data" : {"allow_unauthenticated_public_requests" : true},"channel" : "platform_state.public_methods_state"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Quote Ticker":                           `{"params" : {"data" : {"timestamp" : 1550658624149,"instrument_name" : "BTC-PERPETUAL","best_bid_price" : 3914.97,"best_bid_amount" : 40,"best_ask_price" : 3996.61,"best_ask_amount" : 50},"channel" : "quote.BTC-PERPETUAL"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Rfq":                                    `{"params" : {"data" : {"state" : true,"side" : null,"last_rfq_tstamp" : 1634816143836,"instrument_name" : "BTC-PERPETUAL","amount" : null	  },"channel" : "rfq.btc"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Instrument Trades":                      `{"params":{"data":[{"trade_seq":30289442,"trade_id" : "48079269","timestamp" : 1590484512188,"tick_direction" : 2,"price" : 8950,"mark_price" : 8948.9,"instrument_name" : "BTC-PERPETUAL","index_price" : 8955.88,"direction" : "sell","amount" : 10}],"channel" : "trades.BTC-PERPETUAL.raw"},"method":"subscription","jsonrpc":"2.0"}`,
	"Instruments Ticker":                     `{"params" : {"data" : {"timestamp" : 1623060194301,"stats" : {"volume_usd" : 284061480,"volume" : 7871.02139035,"price_change" : 0.7229,"low" : 35213.5,"high" : 36824.5},"state" : "open","settlement_price" : 36169.49,"open_interest" : 502097590,"min_price" : 35898.37,"max_price" : 36991.72,"mark_price" : 36446.51,"last_price" : 36457.5,"interest_value" : 1.7362511643080387,"instrument_name" : "BTC-PERPETUAL","index_price" : 36441.64,"funding_8h" : 0.0000211,"estimated_delivery_price" : 36441.64,"current_funding" : 0,"best_bid_price" : 36442.5,"best_bid_amount" : 5000,"best_ask_price" : 36443,"best_ask_amount" : 100	  },	  "channel" : "ticker.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
}

func TestProcessPushData(t *testing.T) {
	t.Parallel()
	for k, v := range websocketPushData {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			err := e.wsHandleData(t.Context(), []byte(v))
			require.NoError(t, err, "wsHandleData must not error")
		})
	}
}

func TestFormatFuturesTradablePair(t *testing.T) {
	t.Parallel()
	futuresInstrumentsOutputList := map[currency.Pair]string{
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode(perpString)}:                    "BTC-PERPETUAL",
		{Delimiter: currency.DashDelimiter, Base: currency.AVAX, Quote: currency.NewCode("USDC-PERPETUAL")}:             "AVAX_USDC-PERPETUAL",
		{Delimiter: currency.DashDelimiter, Base: currency.ETH, Quote: currency.NewCode("30DEC22")}:                     "ETH-30DEC22",
		{Delimiter: currency.DashDelimiter, Base: currency.SOL, Quote: currency.NewCode("30DEC22")}:                     "SOL-30DEC22",
		{Delimiter: currency.DashDelimiter, Base: currency.NewCode("BTCDVOL"), Quote: currency.NewCode("USDC-28JUN23")}: "BTCDVOL_USDC-28JUN23",
	}
	for pair, instrumentID := range futuresInstrumentsOutputList {
		t.Run(instrumentID, func(t *testing.T) {
			t.Parallel()
			instrument := formatFuturesTradablePair(pair)
			require.Equal(t, instrumentID, instrument)
		})
	}
}

func TestOptionPairToString(t *testing.T) {
	t.Parallel()
	for pair, exp := range map[currency.Pair]string{
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("30MAY24-61000-C")}:      "BTC-30MAY24-61000-C",
		{Delimiter: currency.DashDelimiter, Base: currency.ETH, Quote: currency.NewCode("1JUN24-3200-P")}:        "ETH-1JUN24-3200-P",
		{Delimiter: currency.DashDelimiter, Base: currency.SOL, Quote: currency.NewCode("USDC-31MAY24-162-P")}:   "SOL_USDC-31MAY24-162-P",
		{Delimiter: currency.DashDelimiter, Base: currency.MATIC, Quote: currency.NewCode("USDC-6APR24-0d98-P")}: "MATIC_USDC-6APR24-0d98-P",
		{Delimiter: currency.DashDelimiter, Base: currency.MATIC, Quote: currency.NewCode("USDC-8JUN24-0D99-P")}: "MATIC_USDC-8JUN24-0d99-P",
		{Delimiter: currency.DashDelimiter, Base: currency.MATIC, Quote: currency.NewCode("USDC-6DEC29-0D87-C")}: "MATIC_USDC-6DEC29-0d87-C",
	} {
		assert.Equal(t, exp, optionPairToString(pair), "optionPairToString should return correctly")
	}
}

func TestFutureComboPairToString(t *testing.T) {
	t.Parallel()
	for pair, exp := range map[currency.Pair]string{
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("FS-28NOV25_PERP")}:      "BTC-FS-28NOV25_PERP",
		{Delimiter: currency.DashDelimiter, Base: currency.ETH, Quote: currency.NewCode("FS-28NOV25_PERP")}:      "ETH-FS-28NOV25_PERP",
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("FS-30JAN26_26DEC25")}:   "BTC-FS-30JAN26_26DEC25",
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("USDC-FS-28NOV25_PERP")}: "BTC_USDC-FS-28NOV25_PERP",
		{Delimiter: currency.DashDelimiter, Base: currency.ETH, Quote: currency.NewCode("USDC-FS-28NOV25_PERP")}: "ETH_USDC-FS-28NOV25_PERP",
		{Base: currency.BTC, Quote: currency.USDT}:                                                               "BTCUSDT",            // no dash at all
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("USDC-PERPETUAL")}:       "BTC-USDC-PERPETUAL", // USDC- prefix but no dash after (3 parts)
	} {
		assert.Equal(t, exp, futureComboPairToString(pair), "futureComboPairToString should return correctly")
	}
}

func TestWSRetrieveCombos(t *testing.T) {
	t.Parallel()
	_, err := e.WSRetrieveCombos(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.WSRetrieveCombos(t.Context(), futureComboTradablePair.Base)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)
	result, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  futuresTradablePair,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.CurrencyPairs.GetPairs(a, true)
			require.NoError(t, err, "GetPairs must not error")
			l, err := e.GetOrderExecutionLimits(a, pairs[0])
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			assert.Positive(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive")
			assert.Positive(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive")
		})
	}
}

func TestGetLockedStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetLockedStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSayHello(t *testing.T) {
	t.Parallel()
	result, err := e.SayHello(t.Context(), "Thrasher", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrieveCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsRetrieveCancelOnDisconnect(t.Context(), "connection")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsDisableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WsDisableCancelOnDisconnect(t.Context(), "connection")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableCancelOnDisconnect(t.Context(), "account")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsEnableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WsEnableCancelOnDisconnect(t.Context(), "connection")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogout(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.WsLogout(t.Context(), true)
	assert.NoError(t, err)
}

func TestExchangeToken(t *testing.T) {
	t.Parallel()
	_, err := e.ExchangeToken(t.Context(), "", 1234)
	require.ErrorIs(t, err, errRefreshTokenRequired)
	_, err = e.ExchangeToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", 0)
	require.ErrorIs(t, err, errSubjectIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ExchangeToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsExchangeToken(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsExchangeToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestForkToken(t *testing.T) {
	t.Parallel()
	_, err := e.ForkToken(t.Context(), "", "Sami")
	require.ErrorIs(t, err, errRefreshTokenRequired)
	_, err = e.ForkToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", "")
	require.ErrorIs(t, err, errSessionNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.ForkToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", "Sami")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsForkToken(t *testing.T) {
	t.Parallel()
	_, err := e.WsForkToken(t.Context(), "", "Sami")
	require.ErrorIs(t, err, errRefreshTokenRequired)
	_, err = e.WsForkToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", "")
	require.ErrorIs(t, err, errSessionNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.WsForkToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", "Sami")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Binary)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	result, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	paramToErrorMap := map[*futures.PositionSummaryRequest]error{
		nil:                    common.ErrNilPointer,
		{}:                     futures.ErrNotPerpetualFuture,
		{Asset: asset.Futures}: currency.ErrCurrencyPairEmpty,
	}

	for param, errIncoming := range paramToErrorMap {
		_, err := e.GetFuturesPositionSummary(t.Context(), param)
		require.ErrorIs(t, err, errIncoming)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	req := &futures.PositionSummaryRequest{
		Asset: asset.Futures,
		Pair:  currency.NewPair(currency.BTC, currency.NewCode(perpString)),
	}
	result, err := e.GetFuturesPositionSummary(t.Context(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.SOL.Item,
		Quote: currency.USDC.Item,
		Asset: asset.Spot,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  optionsTradablePair.Base.Item,
		Quote: optionsTradablePair.Quote.Item,
		Asset: asset.Options,
	})
	require.NoError(t, err)

	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.NewCode(perpString).Item,
		Asset: asset.Futures,
	})
	require.NoError(t, err)

	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.NewCode("XRP").Item,
		Quote: currency.NewCode("USDC-PERPETUAL").Item,
		Asset: asset.Futures,
	})
	require.NoError(t, err)

	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  futureComboTradablePair.Base.Item,
		Quote: futureComboTradablePair.Quote.Item,
		Asset: asset.FutureCombo,
	})
	require.NoError(t, err)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	assetPairToErrorMap := map[asset.Item][]struct {
		Pair     currency.Pair
		Error    error
		Response bool
	}{
		asset.Spot: {
			{Pair: currency.EMPTYPAIR, Error: currency.ErrCurrencyPairEmpty, Response: false},
			{Pair: spotTradablePair, Error: nil, Response: false},
		},
		asset.Futures: {
			{Pair: currency.NewPair(currency.BTC, currency.NewCode(perpString)), Response: true},
		},
		asset.FutureCombo: {
			{Pair: currency.NewPair(currency.NewCode("BTC"), currency.NewCode("FS-27SEP24_PERP")), Response: false},
		},
		asset.OptionCombo: {
			{Pair: currency.NewPair(currency.NewCode(currencyBTC), currency.NewCode("STRG-21OCT22")), Error: nil, Response: false},
		},
	}
	for assetType, instances := range assetPairToErrorMap {
		for i := range instances {
			t.Run(fmt.Sprintf("Asset: %s Pair: %s", assetType.String(), instances[i].Pair.String()), func(t *testing.T) {
				t.Parallel()
				is, err := e.IsPerpetualFutureCurrency(assetType, instances[i].Pair)
				require.ErrorIs(t, err, instances[i].Error)
				require.Equal(t, is, instances[i].Response)
			})
		}
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-PERPETUAL")
	require.NoError(t, err)
	r := &fundingrate.HistoricalRatesRequest{
		Asset:           asset.Spot,
		Pair:            cp,
		PaymentCurrency: currency.USDT,
		StartDate:       time.Now().Add(-time.Hour * 24 * 2),
		EndDate:         time.Now(),
	}
	_, err = e.GetHistoricalFundingRates(t.Context(), r)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	r.Asset = asset.Futures
	result, err := e.GetHistoricalFundingRates(t.Context(), r)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMultipleCancelResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	resp := &struct {
		Result *MultipleCancelResponse `json:"result"`
	}{}
	data := `{ "jsonrpc": "2.0", "id": 8748, "result": 37 }`
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)
	require.Equal(t, int64(37), resp.Result.CancelCount)
	data = `{"jsonrpc":"2.0","id":1599612810505,"result":[{"instrument_name":"BTC_USDC","currency":"BTC_USDC","result":[{"is_rebalance":false,"risk_reducing":false,"order_type":"limit","creation_timestamp":1715302998260,"order_state":"cancelled","contracts":300000.0,"average_price":0.0,"post_only":false,"last_update_timestamp":1715303041949,"filled_amount":0.0,"replaced":false,"web":false,"api":true,"mmp":false,"cancel_reason":"user_request","instrument_name":"BTC_USDC","order_id":"BTC_USDC-13133482","max_show":30.0,"time_in_force":"good_til_cancelled","direction":"buy","amount":30.0,"price":30.0,"label":"test"}],"type":"limit"}]}`
	err = json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Result.CancelCount)
	require.Len(t, resp.Result.CancelDetails, 1)
	require.Len(t, resp.Result.CancelDetails, 1)
}

func TestGetResolutionFromInterval(t *testing.T) {
	t.Parallel()
	intervalStringMap := []struct {
		Interval       kline.Interval
		IntervalString string
		Error          error
	}{
		{Interval: kline.HundredMilliseconds, IntervalString: "100ms"},
		{Interval: kline.OneMin, IntervalString: "1"},
		{Interval: kline.ThreeMin, IntervalString: "3"},
		{Interval: kline.FiveMin, IntervalString: "5"},
		{Interval: kline.TenMin, IntervalString: "10"},
		{Interval: kline.FifteenMin, IntervalString: "15"},
		{Interval: kline.ThirtyMin, IntervalString: "30"},
		{Interval: kline.OneHour, IntervalString: "60"},
		{Interval: kline.TwoHour, IntervalString: "120"},
		{Interval: kline.ThreeHour, IntervalString: "180"},
		{Interval: kline.SixHour, IntervalString: "360"},
		{Interval: kline.TwelveHour, IntervalString: "720"},
		{Interval: kline.OneDay, IntervalString: "1D"},
		{Interval: kline.Raw, IntervalString: "raw"},
		{Interval: kline.FourHour, Error: kline.ErrUnsupportedInterval},
	}
	for x := range intervalStringMap {
		result, err := e.GetResolutionFromInterval(intervalStringMap[x].Interval)
		require.ErrorIs(t, err, intervalStringMap[x].Error)
		require.Equal(t, intervalStringMap[x].IntervalString, result)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	for _, a := range e.GetAssetTypes(false) {
		var pairs currency.Pairs
		pairs, err = e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		var resp string
		resp, err = e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
	// specific test to ensure perps work
	cp := currency.NewPair(currency.BTC, currency.NewCode("USDC-PERPETUAL"))
	resp, err := e.GetCurrencyTradeURL(t.Context(), asset.Futures, cp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
	// specific test to ensure options with dates work
	cp = currency.NewPair(currency.BTC, currency.NewCode("14JUN24-62000-C"))
	resp, err = e.GetCurrencyTradeURL(t.Context(), asset.Options, cp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestFormatChannelPair(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.NewCode("USDC-PERPETUAL"))
	pair.Delimiter = "-"
	assert.Equal(t, "BTC_USDC-PERPETUAL", formatChannelPair(pair))

	pair = currency.NewPair(currency.BTC, currency.NewCode("PERPETUAL"))
	pair.Delimiter = "-"
	assert.Equal(t, "BTC-PERPETUAL", formatChannelPair(pair))
}

var timeInForceList = []struct {
	String   string
	PostOnly bool
	TIF      order.TimeInForce
	Error    error
}{
	{"good_til_cancelled", false, order.GoodTillCancel, nil},
	{"good_til_cancelled", true, order.GoodTillCancel | order.PostOnly, nil},
	{"good_til_day", false, order.GoodTillDay, nil},
	{"good_til_day", true, order.GoodTillDay | order.PostOnly, nil},
	{"fill_or_kill", false, order.FillOrKill, nil},
	{"immediate_or_cancel", false, order.ImmediateOrCancel, nil},
	{"abcd", false, order.UnknownTIF, order.ErrInvalidTimeInForce},
	{"", false, order.UnknownTIF, nil},
}

func TestTimeInForceFromString(t *testing.T) {
	t.Parallel()
	for i := range timeInForceList {
		result, err := timeInForceFromString(timeInForceList[i].String, timeInForceList[i].PostOnly)
		assert.Equalf(t, timeInForceList[i].TIF, result, "expected  %s, got %s", timeInForceList[i].TIF.String(), result.String())
		require.ErrorIs(t, err, timeInForceList[i].Error)
	}
}

func TestOptionsComboFormatting(t *testing.T) {
	t.Parallel()
	availablePairs, err := e.GetAvailablePairs(asset.OptionCombo)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.GreaterOrEqual(t, len(availablePairs), 5, "availablePairs must be greater than or equal 5")
	for _, cp := range availablePairs[:5] {
		t.Run(cp.String(), func(t *testing.T) {
			t.Parallel()
			_, err := e.GetPublicTicker(t.Context(), optionComboPairToString(cp))
			assert.NoError(t, err, "GetPublicTicker should not error")
		})
	}
}

func TestAppendCandles(t *testing.T) {
	t.Parallel()
	_, err := appendCandles(nil, time.Time{})
	assert.ErrorIs(t, err, kline.ErrNoTimeSeriesDataToConvert)

	candles := &TVChartData{
		Ticks: []int64{1337},
	}
	_, err = appendCandles(candles, time.Time{})
	assert.ErrorIs(t, err, kline.ErrInsufficientCandleData)

	candles = &TVChartData{
		Open:   []float64{1337},
		High:   []float64{1337},
		Low:    []float64{1337},
		Close:  []float64{1337},
		Volume: []float64{1337},
		Ticks:  []int64{1337},
	}
	resp, err := appendCandles(candles, time.Time{})
	assert.NoError(t, err)
	assert.Len(t, resp, 1)

	candles = &TVChartData{
		Open:   []float64{1337},
		High:   []float64{1337},
		Low:    []float64{1337},
		Close:  []float64{1337},
		Volume: []float64{1337},
		Ticks:  []int64{1337},
	}
	resp, err = appendCandles(candles, time.Unix(1338, 0))
	assert.NoError(t, err)
	assert.Empty(t, resp)
}
