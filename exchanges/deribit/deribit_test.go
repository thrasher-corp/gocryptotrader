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
	d                                                                     = &Deribit{}
	optionsTradablePair, optionComboTradablePair, futureComboTradablePair currency.Pair
	spotTradablePair                                                      = currency.NewPairWithDelimiter(currencyBTC, "USDC", "_")
	futuresTradablePair                                                   = currency.NewPairWithDelimiter(currencyBTC, perpString, "-")
	assetTypeToPairsMap                                                   map[asset.Item]currency.Pair
)

func TestMain(m *testing.M) {
	if err := testexch.Setup(d); err != nil {
		log.Fatalf("Deribit Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		d.API.AuthenticatedSupport = true
		d.API.AuthenticatedWebsocketSupport = true
		d.SetCredentials(apiKey, apiSecret, "", "", "", "")
		d.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	if useTestNet {
		deribitWebsocketAddress = "wss://test.deribit.com/ws" + deribitAPIVersion
		if err := d.Websocket.SetWebsocketURL(deribitWebsocketAddress, false, true); err != nil {
			log.Fatalf("Deribit SetWebsocketURL error: %s", err)
		}
		for k, v := range d.API.Endpoints.GetURLMap() {
			v = strings.Replace(v, "www.deribit.com", "test.deribit.com", 1)
			if err := d.API.Endpoints.SetRunningURL(k, v); err != nil {
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
	if err := d.UpdateTradablePairs(context.Background(), true); err != nil {
		log.Fatalf("Failed to update tradable pairs. Error: %v", err)
	}

	handleError := func(err error, msg string) {
		if err != nil {
			log.Fatalf("%s. Error: %v", msg, err)
		}
	}

	updateTradablePair := func(assetType asset.Item, tradablePair *currency.Pair) {
		if d.CurrencyPairs.IsAssetEnabled(assetType) == nil {
			pairs, err := d.GetEnabledPairs(assetType)
			handleError(err, fmt.Sprintf("Failed to get enabled pairs for asset type %v", assetType))

			if len(pairs) == 0 {
				handleError(currency.ErrCurrencyPairsEmpty, fmt.Sprintf("No enabled pairs for asset type %v", assetType))
			}

			if assetType == asset.Options {
				*tradablePair, err = d.FormatExchangeCurrency(pairs[0], assetType)
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
	_, err := d.UpdateTicker(t.Context(), currency.Pair{}, asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	for assetType, cp := range assetTypeToPairsMap {
		result, err := d.UpdateTicker(t.Context(), cp, assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	for assetType, cp := range assetTypeToPairsMap {
		result, err := d.UpdateOrderbook(t.Context(), cp, assetType)
		require.NoErrorf(t, err, "asset type: %v", assetType)
		require.NotNil(t, result)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricTrades(t.Context(), futureComboTradablePair, asset.FutureCombo, time.Now().Add(-time.Minute*10), time.Now())
	require.ErrorIs(t, err, asset.ErrNotSupported)
	for assetType, cp := range map[asset.Item]currency.Pair{asset.Spot: spotTradablePair, asset.Futures: futuresTradablePair} {
		_, err = d.GetHistoricTrades(t.Context(), cp, assetType, time.Now().Add(-time.Minute*10), time.Now())
		require.NoErrorf(t, err, "asset type: %v", assetType)
	}
}

func TestFetchRecentTrades(t *testing.T) {
	t.Parallel()
	for assetType, cp := range assetTypeToPairsMap {
		result, err := d.GetRecentTrades(t.Context(), cp, assetType)
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
		resp, err := d.GetHistoricCandles(t.Context(), info.Pair, assetType, kline.FifteenMin, start, end)
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
		resp, err := d.GetHistoricCandlesExtended(t.Context(), instance.Pair, assetType, kline.OneDay, start, end)
		require.ErrorIs(t, err, instance.Error)
		if instance.Error == nil {
			require.NotEmpty(t, resp)
		}
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	assetToPairStringMap := map[asset.Item]currency.Pair{
		asset.Options:     optionsTradablePair,
		asset.FutureCombo: futureComboTradablePair,
		asset.Futures:     futuresTradablePair,
	}
	var result *order.SubmitResponse
	var err error
	var info *InstrumentData
	for assetType, cp := range assetToPairStringMap {
		info, err = d.GetInstrument(t.Context(), d.formatPairString(assetType, cp))
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)

		result, err = d.SubmitOrder(
			t.Context(),
			&order.Submit{
				Exchange:  d.Name,
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

	_, err = d.GetMarkPriceHistory(t.Context(), "", time.Now().Add(-5*time.Minute), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result []MarkPriceHistory
	for _, ps := range []string{
		d.optionPairToString(optionsTradablePair),
		spotTradablePair.String(),
		btcPerpInstrument,
		futureComboTradablePair.String(),
	} {
		result, err = d.GetMarkPriceHistory(t.Context(), ps, time.Now().Add(-5*time.Minute), time.Now())
		require.NoErrorf(t, err, "expected nil, got %v for pair %s", err, ps)
		require.NotNilf(t, result, "expected result not to be nil for pair %s", ps)
	}
}

func TestWSRetrieveMarkPriceHistory(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveMarkPriceHistory(t.Context(), "", time.Now().Add(-4*time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result []MarkPriceHistory
	for _, ps := range []string{
		d.optionPairToString(optionsTradablePair),
		spotTradablePair.String(),
		btcPerpInstrument,
		futureComboTradablePair.String(),
	} {
		result, err = d.WSRetrieveMarkPriceHistory(t.Context(), ps, time.Now().Add(-4*time.Hour), time.Now())
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
	_, err = d.GetBookSummaryByCurrency(t.Context(), currency.EMPTYCODE, "future")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.GetBookSummaryByCurrency(t.Context(), currency.BTC, "option")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveBookBySummary(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveBookBySummary(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	result, err := d.WSRetrieveBookBySummary(t.Context(), currency.SOL, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetBookSummaryByInstrument(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result []BookSummaryData
	for _, ps := range []string{
		btcPerpInstrument,
		spotTradablePair.String(),
		futureComboTradablePair.String(),
		d.optionPairToString(optionsTradablePair),
		optionComboTradablePair.String(),
	} {
		result, err = d.GetBookSummaryByInstrument(t.Context(), ps)
		require.NoErrorf(t, err, "expected nil, got %v for pair %s", err, ps)
		require.NotNilf(t, result, "expected result not to be nil for pair %s", ps)
	}
}

func TestWSRetrieveBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveBookSummaryByInstrument(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)
	var result []BookSummaryData
	for _, ps := range []string{
		btcPerpInstrument,
		spotTradablePair.String(),
		futureComboTradablePair.String(),
		d.optionPairToString(optionsTradablePair),
		optionComboTradablePair.String(),
	} {
		result, err = d.WSRetrieveBookSummaryByInstrument(t.Context(), ps)
		require.NoErrorf(t, err, "expected nil, got %v for pair %s", err, ps)
		require.NotNilf(t, result, "expected result not to be nil for pair %s", ps)
	}
}

func TestGetContractSize(t *testing.T) {
	t.Parallel()
	_, err := d.GetContractSize(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)
	result, err := d.GetContractSize(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveContractSize(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveContractSize(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := d.WSRetrieveContractSize(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	result, err := d.GetCurrencies(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveCurrencies(t *testing.T) {
	t.Parallel()
	result, err := d.WSRetrieveCurrencies(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDeliveryPrices(t *testing.T) {
	t.Parallel()
	_, err := d.GetDeliveryPrices(t.Context(), "", 0, 5)
	require.ErrorIs(t, err, errUnsupportedIndexName)

	result, err := d.GetDeliveryPrices(t.Context(), "btc_usd", 0, 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveDeliveryPrices(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveDeliveryPrices(t.Context(), "", 0, 5)
	require.ErrorIs(t, err, errUnsupportedIndexName)

	result, err := d.WSRetrieveDeliveryPrices(t.Context(), "btc_usd", 0, 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingChartData(t *testing.T) {
	t.Parallel()
	// only for perpetual instruments
	_, err := d.GetFundingChartData(t.Context(), "", "8h")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := d.GetFundingChartData(t.Context(), btcPerpInstrument, "8h")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveFundingChartData(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveFundingChartData(t.Context(), "", "8h")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := d.WSRetrieveFundingChartData(t.Context(), btcPerpInstrument, "8h")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingRateHistory(t.Context(), "", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := d.GetFundingRateHistory(t.Context(), btcPerpInstrument, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveFundingRateHistory(t.Context(), "", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)
	result, err := d.WSRetrieveFundingRateHistory(t.Context(), btcPerpInstrument, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingRateValue(t.Context(), "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = d.GetFundingRateValue(t.Context(), btcPerpInstrument, time.Now(), time.Now().Add(-time.Hour*8))
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := d.GetFundingRateValue(t.Context(), btcPerpInstrument, time.Now().Add(-time.Hour*8), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveFundingRateValue(t.Context(), btcPerpInstrument, time.Now(), time.Now().Add(-time.Hour*8))
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := d.WSRetrieveFundingRateValue(t.Context(), btcPerpInstrument, time.Now().Add(-time.Hour*8), time.Now())
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
	_, err := d.GetHistoricalVolatility(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.GetHistoricalVolatility(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveHistoricalVolatility(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveHistoricalVolatility(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.WSRetrieveHistoricalVolatility(t.Context(), currency.SOL)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := d.GetCurrencyIndexPrice(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	result, err := d.GetCurrencyIndexPrice(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveCurrencyIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveCurrencyIndexPrice(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	result, err := d.WSRetrieveCurrencyIndexPrice(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := d.GetIndexPrice(t.Context(), "")
	require.ErrorIs(t, err, errUnsupportedIndexName)
	result, err := d.GetIndexPrice(t.Context(), "ada_usd")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveIndexPrice(t.Context(), "")
	require.ErrorIs(t, err, errUnsupportedIndexName)
	result, err := d.WSRetrieveIndexPrice(t.Context(), "ada_usd")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	result, err := d.GetIndexPriceNames(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveIndexPriceNames(t *testing.T) {
	t.Parallel()
	result, err := d.WSRetrieveIndexPriceNames(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstrumentData(t *testing.T) {
	t.Parallel()
	_, err := d.GetInstrument(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result *InstrumentData
	for assetType, cp := range assetTypeToPairsMap {
		result, err = d.GetInstrument(t.Context(), d.formatPairString(assetType, cp))
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestWSRetrieveInstrumentData(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveInstrumentData(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result *InstrumentData
	for assetType, cp := range assetTypeToPairsMap {
		result, err = d.WSRetrieveInstrumentData(t.Context(), d.formatPairString(assetType, cp))
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	result, err := d.GetInstruments(t.Context(), currency.EMPTYCODE, "future", false)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = d.GetInstruments(t.Context(), currency.BTC, "", false)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = d.GetInstruments(t.Context(), currency.BTC, "", true)
	require.NoError(t, err)
	for a := range result {
		require.Falsef(t, result[a].IsActive, "expected expired instrument, but got active instrument %s", result[a].InstrumentName)
	}
}

func TestWSRetrieveInstrumentsData(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveInstrumentsData(t.Context(), currency.EMPTYCODE, "", false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.WSRetrieveInstrumentsData(t.Context(), currency.BTC, "", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastSettlementsByCurrency(t.Context(), currency.EMPTYCODE, "delivery", "5", 0, time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.GetLastSettlementsByCurrency(t.Context(), currency.BTC, "delivery", "5", 0, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveLastSettlementsByCurrency(t.Context(), currency.EMPTYCODE, "delivery", "5", 0, time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.WSRetrieveLastSettlementsByCurrency(t.Context(), currency.BTC, "delivery", "5", 0, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveLastSettlementsByInstrument(t.Context(), "", "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := d.WSRetrieveLastSettlementsByInstrument(t.Context(), d.formatFuturesTradablePair(futuresTradablePair), "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastSettlementsByInstrument(t.Context(), "", "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := d.GetLastSettlementsByInstrument(t.Context(), d.formatFuturesTradablePair(futuresTradablePair), "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByCurrency(t.Context(), currency.EMPTYCODE, "option", "36798", "36799", "asc", 0, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.GetLastTradesByCurrency(t.Context(), currency.BTC, "option", "36798", "36799", "asc", 0, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveLastTradesByCurrency(t.Context(), currency.EMPTYCODE, "option", "36798", "36799", "asc", 0, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.WSRetrieveLastTradesByCurrency(t.Context(), currency.BTC, "option", "36798", "36799", "asc", 0, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByCurrencyAndTime(t.Context(), currency.EMPTYCODE, "", "", 0, time.Now().Add(-8*time.Hour), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.GetLastTradesByCurrencyAndTime(t.Context(), currency.BTC, "", "", 0, time.Now().Add(-8*time.Hour), time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = d.GetLastTradesByCurrencyAndTime(t.Context(), currency.BTC, "option", "asc", 25, time.Now().Add(-8*time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveLastTradesByCurrencyAndTime(t.Context(), currency.EMPTYCODE, "", "", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.WSRetrieveLastTradesByCurrencyAndTime(t.Context(), currency.BTC, "", "", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = d.WSRetrieveLastTradesByCurrencyAndTime(t.Context(), currency.BTC, "option", "asc", 25, false, time.Now().Add(-8*time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByInstrument(t.Context(), "", "", "", "", 0, false)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	for assetType, cp := range assetTypeToPairsMap {
		result, err := d.GetLastTradesByInstrument(t.Context(), d.formatPairString(assetType, cp), "30500", "31500", "desc", 0, true)
		require.NoErrorf(t, err, "expected %v, got %v currency asset %v pair %v", nil, err, assetType, cp)
		require.NotNilf(t, result, "expected value not to be nil for asset %v pair: %v", assetType, cp)
	}
}

func TestWSRetrieveLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveLastTradesByInstrument(t.Context(), "", "", "", "", 0, false)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	for assetType, cp := range assetTypeToPairsMap {
		result, err := d.WSRetrieveLastTradesByInstrument(t.Context(), d.formatPairString(assetType, cp), "30500", "31500", "desc", 0, true)
		require.NoErrorf(t, err, "expected %v, got %v currency asset %v pair %v", nil, err, assetType, cp)
		require.NotNilf(t, result, "expected value not to be nil for asset %v pair: %v", assetType, cp)
	}
}

func TestGetLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByInstrumentAndTime(t.Context(), "", "", 0, time.Now().Add(-8*time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	for assetType, cp := range assetTypeToPairsMap {
		result, err := d.GetLastTradesByInstrumentAndTime(t.Context(), d.formatPairString(assetType, cp), "", 0, time.Now().Add(-8*time.Hour), time.Now())
		require.NoErrorf(t, err, "expected %v, got %v currency pair %v", nil, err, cp)
		require.NotNilf(t, result, "expected value not to be nil for pair: %v", cp)
	}
}

func TestWSRetrieveLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveLastTradesByInstrumentAndTime(t.Context(), "", "", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	for assetType, cp := range assetTypeToPairsMap {
		result, err := d.WSRetrieveLastTradesByInstrumentAndTime(t.Context(), d.formatPairString(assetType, cp), "", 0, true, time.Now().Add(-8*time.Hour), time.Now())
		require.NoErrorf(t, err, "expected %v, got %v currency pair %v", nil, err, cp)
		require.NotNilf(t, result, "expected value not to be nil for pair: %v", cp)
	}
}

func TestWSProcessTrades(t *testing.T) {
	t.Parallel()

	d := new(Deribit) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(d), "Setup instance must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.json", d.wsHandleData)
	close(d.Websocket.DataHandler)

	p, a, err := d.getAssetPairByInstrument("BTC-PERPETUAL")
	require.NoError(t, err, "getAssetPairByInstrument must not error")

	exp := []trade.Data{
		{
			Exchange:     d.Name,
			CurrencyPair: p,
			Timestamp:    time.UnixMilli(1742627465811).UTC(),
			Price:        84295.5,
			Amount:       8430.0,
			Side:         order.Buy,
			TID:          "356130997",
			AssetType:    a,
		},
		{
			Exchange:     d.Name,
			CurrencyPair: p,
			Timestamp:    time.UnixMilli(1742627361899).UTC(),
			Price:        84319.0,
			Amount:       580.0,
			Side:         order.Sell,
			TID:          "356130979",
			AssetType:    a,
		},
	}
	require.Len(t, d.Websocket.DataHandler, len(exp), "Must see the correct number of trades")
	for resp := range d.Websocket.DataHandler {
		switch v := resp.(type) {
		case trade.Data:
			i := 1 - len(d.Websocket.DataHandler)
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
	_, err := d.GetOrderbook(t.Context(), "", 0)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result *Orderbook
	for assetType, cp := range assetTypeToPairsMap {
		result, err = d.GetOrderbook(t.Context(), d.formatPairString(assetType, cp), 0)
		require.NoErrorf(t, err, "expected %v, got %v currency pair %v", nil, err, cp)
		require.NotNilf(t, result, "expected value not to be nil for pair: %v", cp)
	}
}

func TestWSRetrieveOrderbookData(t *testing.T) {
	t.Parallel()
	if !d.Websocket.IsConnected() {
		t.Skip("websocket is not connected")
	}
	_, err := d.WSRetrieveOrderbookData(t.Context(), "", 0)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	var result *Orderbook
	for assetType, cp := range assetTypeToPairsMap {
		result, err = d.WSRetrieveOrderbookData(t.Context(), d.formatPairString(assetType, cp), 0)
		require.NoErrorf(t, err, "expected %v, got %v currency pair %v", nil, err, cp)
		require.NotNilf(t, result, "expected value not to be nil for pair: %v", cp)
	}
}

func TestGetOrderbookByInstrumentID(t *testing.T) {
	t.Parallel()
	combos, err := d.GetComboIDs(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	if len(combos) == 0 {
		t.Skip("no combo instance found for currency BTC")
	}
	_, err = d.GetOrderbookByInstrumentID(t.Context(), 0, 50)
	require.ErrorIs(t, err, errInvalidInstrumentID)

	comboD, err := d.GetComboDetails(t.Context(), combos[0])
	require.NoError(t, err)
	require.NotNil(t, comboD)

	result, err := d.GetOrderbookByInstrumentID(t.Context(), comboD.InstrumentID, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOrderbookByInstrumentID(t *testing.T) {
	t.Parallel()
	combos, err := d.WSRetrieveComboIDs(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	if len(combos) == 0 {
		t.Skip("no combo instance found for currency BTC")
	}
	_, err = d.WSRetrieveOrderbookByInstrumentID(t.Context(), 0, 50)
	require.ErrorIs(t, err, errInvalidInstrumentID)
	comboD, err := d.WSRetrieveComboDetails(t.Context(), combos[0])
	require.NoError(t, err)
	require.NotNil(t, comboD)

	result, err := d.WSRetrieveOrderbookByInstrumentID(t.Context(), comboD.InstrumentID, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSupportedIndexNames(t *testing.T) {
	t.Parallel()
	result, err := d.GetSupportedIndexNames(t.Context(), "derivative")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrieveSupportedIndexNames(t *testing.T) {
	t.Parallel()
	result, err := d.WsRetrieveSupportedIndexNames(t.Context(), "derivative")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRequestForQuote(t *testing.T) {
	t.Parallel()
	_, err := d.GetRequestForQuote(t.Context(), currency.EMPTYCODE, d.GetAssetKind(asset.Futures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	result, err := d.GetRequestForQuote(t.Context(), currency.BTC, d.GetAssetKind(asset.Futures))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveRequestForQuote(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveRequestForQuote(t.Context(), currency.EMPTYCODE, d.GetAssetKind(asset.Futures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	result, err := d.WSRetrieveRequestForQuote(t.Context(), currency.BTC, d.GetAssetKind(asset.Futures))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeVolumes(t *testing.T) {
	t.Parallel()
	result, err := d.GetTradeVolumes(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveTradeVolumes(t *testing.T) {
	t.Parallel()
	result, err := d.WSRetrieveTradeVolumes(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradingViewChartData(t *testing.T) {
	t.Parallel()
	_, err := d.GetTradingViewChart(t.Context(), "", "60", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := d.GetTradingViewChart(t.Context(), btcPerpInstrument, "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = d.GetTradingViewChart(t.Context(), spotTradablePair.String(), "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrievesTradingViewChartData(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrievesTradingViewChartData(t.Context(), "", "60", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)
	result, err := d.WSRetrievesTradingViewChartData(t.Context(), btcPerpInstrument, "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = d.WSRetrievesTradingViewChartData(t.Context(), spotTradablePair.String(), "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVolatilityIndexData(t *testing.T) {
	t.Parallel()
	_, err := d.GetVolatilityIndex(t.Context(), currency.EMPTYCODE, "60", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.GetVolatilityIndex(t.Context(), currency.BTC, "", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errResolutionNotSet)
	_, err = d.GetVolatilityIndex(t.Context(), currency.BTC, "60", time.Now(), time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := d.GetVolatilityIndex(t.Context(), currency.BTC, "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveVolatilityIndexData(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveVolatilityIndexData(t.Context(), currency.EMPTYCODE, "60", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.WSRetrieveVolatilityIndexData(t.Context(), currency.BTC, "", time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errResolutionNotSet)
	_, err = d.WSRetrieveVolatilityIndexData(t.Context(), currency.BTC, "60", time.Now(), time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := d.WSRetrieveVolatilityIndexData(t.Context(), currency.BTC, "60", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicTicker(t *testing.T) {
	t.Parallel()
	_, err := d.GetPublicTicker(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := d.GetPublicTicker(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrievePublicTicker(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrievePublicTicker(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	result, err := d.WSRetrievePublicTicker(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	_, err := d.GetAccountSummary(t.Context(), currency.EMPTYCODE, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetAccountSummary(t.Context(), currency.BTC, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveAccountSummary(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveAccountSummary(t.Context(), currency.EMPTYCODE, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveAccountSummary(t.Context(), currency.BTC, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelTransferByID(t *testing.T) {
	t.Parallel()
	_, err := d.CancelTransferByID(t.Context(), currency.EMPTYCODE, "", 23487)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.CancelTransferByID(t.Context(), currency.BTC, "", 0)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.CancelTransferByID(t.Context(), currency.BTC, "", 23487)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCancelTransferByID(t *testing.T) {
	t.Parallel()
	_, err := d.WSCancelTransferByID(t.Context(), currency.EMPTYCODE, "", 23487)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.WSCancelTransferByID(t.Context(), currency.BTC, "", 0)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSCancelTransferByID(t.Context(), currency.BTC, "", 23487)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getTransferResponseJSON = `{"count": 2, "data":[{"amount": 0.2, "created_timestamp": 1550579457727, "currency": "BTC", "direction": "payment", "id": 2, "other_side": "2MzyQc5Tkik61kJbEpJV5D5H9VfWHZK9Sgy", "state": "prepared", "type": "user", "updated_timestamp": 1550579457727}, { "amount": 0.3, "created_timestamp": 1550579255800, "currency": "BTC", "direction": "payment", "id": 1, "other_side": "new_user_1_1", "state": "confirmed", "type": "subaccount", "updated_timestamp": 1550579255800} ] }`

func TestGetTransfers(t *testing.T) {
	t.Parallel()
	var resp *TransfersData
	err := json.Unmarshal([]byte(getTransferResponseJSON), &resp)
	require.NoError(t, err)
	_, err = d.GetTransfers(t.Context(), currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetTransfers(t.Context(), currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveTransfers(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveTransfers(t.Context(), currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveTransfers(t.Context(), currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const cancelWithdrawlPushDataJSON = `{"address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.5, "confirmed_timestamp": null, "created_timestamp": 1550571443070, "currency": "BTC", "fee": 0.0001, "id": 1, "priority": 0.15, "state": "cancelled", "transaction_id": null, "updated_timestamp": 1550571443070}`

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	var resp *CancelWithdrawalData
	err := json.Unmarshal([]byte(cancelWithdrawlPushDataJSON), &resp)
	require.NoError(t, err)
	_, err = d.CancelWithdrawal(t.Context(), currency.EMPTYCODE, 123844)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.CancelWithdrawal(t.Context(), currency.BTC, 0)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.CancelWithdrawal(t.Context(), currency.BTC, 123844)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := d.WSCancelWithdrawal(t.Context(), currency.EMPTYCODE, 123844)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.WSCancelWithdrawal(t.Context(), currency.BTC, 0)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSCancelWithdrawal(t.Context(), currency.BTC, 123844)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := d.CreateDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.CreateDepositAddress(t.Context(), currency.SOL)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCreateDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := d.WSCreateDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSCreateDepositAddress(t.Context(), currency.SOL)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := d.GetCurrentDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetCurrentDepositAddress(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveCurrentDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveCurrentDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveCurrentDepositAddress(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getDepositPushDataJSON = `{"count": 1, "data": [ { "address": "2N35qDKDY22zmJq9eSyiAerMD4enJ1xx6ax", "amount": 5, "currency": "BTC", "received_timestamp": 1549295017670, "state": "completed", "transaction_id": "230669110fdaf0a0dbcdc079b6b8b43d5af29cc73683835b9bc6b3406c065fda", "updated_timestamp": 1549295130159} ] }`

func TestGetDeposits(t *testing.T) {
	t.Parallel()
	var resp *DepositsData
	err := json.Unmarshal([]byte(getDepositPushDataJSON), &resp)
	require.NoError(t, err)
	_, err = d.GetDeposits(t.Context(), currency.EMPTYCODE, 25, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetDeposits(t.Context(), currency.BTC, 25, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveDeposits(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveDeposits(t.Context(), currency.EMPTYCODE, 25, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveDeposits(t.Context(), currency.BTC, 25, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getWithdrawalResponseJSON = `{"count": 1, "data": [ { "address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.5, "confirmed_timestamp": null, "created_timestamp": 1550571443070, "currency": "BTC", "fee": 0.0001, "id": 1, "priority": 0.15, "state": "unconfirmed", "transaction_id": null, "updated_timestamp": 1550571443070} ] }`

func TestGetWithdrawals(t *testing.T) {
	t.Parallel()
	var resp *WithdrawalsData
	err := json.Unmarshal([]byte(getWithdrawalResponseJSON), &resp)
	require.NoError(t, err)
	_, err = d.GetWithdrawals(t.Context(), currency.EMPTYCODE, 25, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetWithdrawals(t.Context(), currency.BTC, 25, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveWithdrawals(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveWithdrawals(t.Context(), currency.EMPTYCODE, 25, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveWithdrawals(t.Context(), currency.BTC, 25, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitTransferBetweenSubAccounts(t *testing.T) {
	t.Parallel()
	_, err := d.SubmitTransferBetweenSubAccounts(t.Context(), currency.EMPTYCODE, 12345, 2, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.SubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 0, 2, "")
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.SubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 12345, -1, "")
	require.ErrorIs(t, err, errInvalidDestinationID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 12345, 4, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsSubmitTransferBetweenSubAccounts(t *testing.T) {
	t.Parallel()
	_, err := d.WsSubmitTransferBetweenSubAccounts(t.Context(), currency.EMPTYCODE, 12345, 2, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.WsSubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 0, 2, "")
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.WsSubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 12345, -1, "")
	require.ErrorIs(t, err, errInvalidDestinationID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WsSubmitTransferBetweenSubAccounts(t.Context(), currency.EURR, 12345, 2, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitTransferToSubAccount(t *testing.T) {
	t.Parallel()
	_, err := d.SubmitTransferToSubAccount(t.Context(), currency.EMPTYCODE, 0.01, 13434)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.SubmitTransferToSubAccount(t.Context(), currency.BTC, 0, 13434)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.SubmitTransferToSubAccount(t.Context(), currency.BTC, 0.01, 0)
	require.ErrorIs(t, err, errInvalidDestinationID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitTransferToSubAccount(t.Context(), currency.BTC, 0.01, 13434)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitTransferToSubAccount(t *testing.T) {
	t.Parallel()
	_, err := d.WSSubmitTransferToSubAccount(t.Context(), currency.EMPTYCODE, 0.01, 13434)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.WSSubmitTransferToSubAccount(t.Context(), currency.BTC, 0, 13434)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.WSSubmitTransferToSubAccount(t.Context(), currency.BTC, 0.01, 0)
	require.ErrorIs(t, err, errInvalidDestinationID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitTransferToSubAccount(t.Context(), currency.BTC, 0.01, 13434)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitTransferToUser(t *testing.T) {
	t.Parallel()
	_, err := d.SubmitTransferToUser(t.Context(), currency.EMPTYCODE, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.SubmitTransferToUser(t.Context(), currency.BTC, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.SubmitTransferToUser(t.Context(), currency.BTC, "", "", 0.001)
	require.ErrorIs(t, err, errInvalidCryptoAddress)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitTransferToUser(t.Context(), currency.BTC, "", "13434", 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitTransferToUser(t *testing.T) {
	t.Parallel()
	_, err := d.WSSubmitTransferToUser(t.Context(), currency.EMPTYCODE, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.WSSubmitTransferToUser(t.Context(), currency.BTC, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.WSSubmitTransferToUser(t.Context(), currency.BTC, "", "", 0.001)
	require.ErrorIs(t, err, errInvalidCryptoAddress)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitTransferToUser(t.Context(), currency.BTC, "", "", 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const submitWithdrawalResponseJSON = `{"address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.4, "confirmed_timestamp": null, "created_timestamp": 1550574558607, "currency": "BTC", "fee": 0.0001, "id": 4, "priority": 1, "state": "unconfirmed", "transaction_id": null, "updated_timestamp": 1550574558607}`

func TestSubmitWithdraw(t *testing.T) {
	t.Parallel()
	var resp *WithdrawData
	err := json.Unmarshal([]byte(submitWithdrawalResponseJSON), &resp)
	require.NoError(t, err)
	_, err = d.SubmitWithdraw(t.Context(), currency.EMPTYCODE, core.BitcoinDonationAddress, "", 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.SubmitWithdraw(t.Context(), currency.BTC, core.BitcoinDonationAddress, "", 0)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.SubmitWithdraw(t.Context(), currency.BTC, "", "", 0.001)
	require.ErrorIs(t, err, errInvalidCryptoAddress)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitWithdraw(t.Context(), currency.BTC, core.BitcoinDonationAddress, "", 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitWithdraw(t *testing.T) {
	_, err := d.WSSubmitWithdraw(t.Context(), currency.EMPTYCODE, core.BitcoinDonationAddress, "", 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.WSSubmitWithdraw(t.Context(), currency.BTC, core.BitcoinDonationAddress, "", 0)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.WSSubmitWithdraw(t.Context(), currency.BTC, "", "", 0.001)
	require.ErrorIs(t, err, errInvalidCryptoAddress)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitWithdraw(t.Context(), currency.BTC, core.BitcoinDonationAddress, "", 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	result, err := d.GetAnnouncements(t.Context(), time.Now(), 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveAnnouncements(t *testing.T) {
	t.Parallel()
	result, err := d.WSRetrieveAnnouncements(t.Context(), time.Now(), 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccessLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetAccessLog(t.Context(), 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveAccessLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveAccessLog(t.Context(), 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAPIKeyName(t *testing.T) {
	t.Parallel()
	_, err := d.ChangeAPIKeyName(t.Context(), 0, "TestKey123")
	require.ErrorIs(t, err, errInvalidID)
	_, err = d.ChangeAPIKeyName(t.Context(), 2, "TestKey123$")
	require.ErrorIs(t, err, errUnacceptableAPIKey)
	_, err = d.ChangeAPIKeyName(t.Context(), 2, "#$#")
	require.ErrorIs(t, err, errUnacceptableAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.ChangeAPIKeyName(t.Context(), 1, "TestKey123")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSChangeAPIKeyName(t *testing.T) {
	t.Parallel()
	_, err := d.WSChangeAPIKeyName(t.Context(), 0, "TestKey123")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.WSChangeAPIKeyName(t.Context(), 1, "TestKey123")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeMarginModel(t *testing.T) {
	t.Parallel()
	_, err := d.ChangeMarginModel(t.Context(), 2, "", false)
	require.ErrorIs(t, err, errInvalidMarginModel)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.ChangeMarginModel(t.Context(), 2, "segregated_pm", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsChangeMarginModel(t *testing.T) {
	t.Parallel()
	_, err := d.WsChangeMarginModel(t.Context(), 2, "", false)
	require.ErrorIs(t, err, errInvalidMarginModel)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)

	result, err := d.WsChangeMarginModel(t.Context(), 2, "segregated_pm", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeScopeInAPIKey(t *testing.T) {
	t.Parallel()
	_, err := d.ChangeScopeInAPIKey(t.Context(), -1, "account:read_write")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.ChangeScopeInAPIKey(t.Context(), 1, "account:read_write")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSChangeScopeInAPIKey(t *testing.T) {
	t.Parallel()
	_, err := d.WSChangeScopeInAPIKey(t.Context(), 0, "account:read_write")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.WSChangeScopeInAPIKey(t.Context(), 1, "account:read_write")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeSubAccountName(t *testing.T) {
	t.Parallel()
	err := d.ChangeSubAccountName(t.Context(), 0, "new_sub")
	require.ErrorIs(t, err, errInvalidID)
	err = d.ChangeSubAccountName(t.Context(), 312313, "")
	require.ErrorIs(t, err, errInvalidUsername)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.ChangeSubAccountName(t.Context(), 1, "new_sub")
	assert.NoError(t, err)
}

func TestWSChangeSubAccountName(t *testing.T) {
	t.Parallel()
	err := d.WSChangeSubAccountName(t.Context(), 0, "new_sub")
	require.ErrorIs(t, err, errInvalidID)
	err = d.WSChangeSubAccountName(t.Context(), 312313, "")
	require.ErrorIs(t, err, errInvalidUsername)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.WSChangeSubAccountName(t.Context(), 1, "new_sub")
	assert.NoError(t, err)
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.CreateAPIKey(t.Context(), "account:read_write", "new_sub", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCreateAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.WSCreateAPIKey(t.Context(), "account:read_write", "new_sub", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.CreateSubAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCreateSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSCreateSubAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDisableAPIKey(t *testing.T) {
	t.Parallel()
	_, err := d.DisableAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.DisableAPIKey(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSDisableAPIKey(t *testing.T) {
	t.Parallel()
	_, err := d.WSDisableAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.WSDisableAPIKey(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEditAPIKey(t *testing.T) {
	t.Parallel()
	_, err := d.EditAPIKey(t.Context(), 0, "trade", "", false, []string{"read", "read_write"}, []string{})
	require.ErrorIs(t, err, errInvalidAPIKeyID)
	_, err = d.EditAPIKey(t.Context(), 1234, "", "", false, []string{"read", "read_write"}, []string{})
	require.ErrorIs(t, err, errMaxScopeIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.EditAPIKey(t.Context(), 1234, "trade", "", false, []string{"read", "read_write"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsEditAPIKey(t *testing.T) {
	t.Parallel()
	_, err := d.WsEditAPIKey(t.Context(), 0, "trade", "", false, []string{"read", "read_write"}, []string{})
	require.ErrorIs(t, err, errInvalidAPIKeyID)
	_, err = d.WsEditAPIKey(t.Context(), 1234, "", "", false, []string{"read", "read_write"}, []string{})
	require.ErrorIs(t, err, errMaxScopeIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.WsEditAPIKey(t.Context(), 1234, "trade", "", false, []string{"read", "read_write"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableAffiliateProgram(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err := d.EnableAffiliateProgram(t.Context())
	assert.NoError(t, err)
}

func TestWSEnableAffiliateProgram(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err := d.WSEnableAffiliateProgram(t.Context())
	assert.NoError(t, err)
}

func TestEnableAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.EnableAPIKey(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSEnableAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.WSEnableAPIKey(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateProgramInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetAffiliateProgramInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveAffiliateProgramInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveAffiliateProgramInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEmailLanguage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetEmailLanguage(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveEmailLanguage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveEmailLanguage(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNewAnnouncements(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetNewAnnouncements(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveNewAnnouncements(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveNewAnnouncements(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPosition(t *testing.T) {
	t.Parallel()
	_, err := d.GetPosition(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetPosition(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrievePosition(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrievePosition(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrievePosition(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetSubAccounts(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveSubAccounts(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDetails(t *testing.T) {
	t.Parallel()
	_, err := d.GetSubAccountDetails(t.Context(), currency.EMPTYCODE, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetSubAccountDetails(t.Context(), currency.BTC, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveSubAccountDetails(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveSubAccountDetails(t.Context(), currency.EMPTYCODE, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveSubAccountDetails(t.Context(), currency.BTC, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	_, err := d.GetPositions(t.Context(), currency.EMPTYCODE, "option")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetPositions(t.Context(), currency.BTC, "option")
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = d.GetPositions(t.Context(), currency.ETH, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrievePositions(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrievePositions(t.Context(), currency.EMPTYCODE, "option")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrievePositions(t.Context(), currency.BTC, "option")
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = d.WSRetrievePositions(t.Context(), currency.ETH, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getTransactionLogResponseJSON = `{"logs": [ { "username": "TestUser", "user_seq": 6009, "user_id": 7, "type": "transfer", "trade_id": null, "timestamp": 1613659830333, "side": "-", "price": null, "position": null, "order_id": null, "interest_pl": null, "instrument_name": null, "info": { "transfer_type": "subaccount", "other_user_id": 27, "other_user": "Subaccount" }, "id": 61312, "equity": 3000.9275869, "currency": "BTC", "commission": 0, "change": -2.5, "cashflow": -2.5, "balance": 3001.22270418 } ], "continuation": 61282 }`

func TestGetTransactionLog(t *testing.T) {
	t.Parallel()
	var resp *TransactionsData
	err := json.Unmarshal([]byte(getTransactionLogResponseJSON), &resp)
	require.NoError(t, err)
	_, err = d.GetTransactionLog(t.Context(), currency.EMPTYCODE, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetTransactionLog(t.Context(), currency.BTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveTransactionLog(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveTransactionLog(t.Context(), currency.EMPTYCODE, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveTransactionLog(t.Context(), currency.BTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserLocks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetUserLocks(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserLocks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveUserLocks(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListAPIKeys(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.ListAPIKeys(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSListAPIKeys(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSListAPIKeys(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCustodyAccounts(t *testing.T) {
	t.Parallel()
	_, err := d.GetCustodyAccounts(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetCustodyAccounts(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrieveCustodyAccounts(t *testing.T) {
	t.Parallel()
	_, err := d.WsRetrieveCustodyAccounts(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WsRetrieveCustodyAccounts(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRemoveAPIKey(t *testing.T) {
	t.Parallel()
	err := d.RemoveAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	err = d.RemoveAPIKey(t.Context(), 1)
	assert.NoError(t, err)
}

func TestWSRemoveAPIKey(t *testing.T) {
	t.Parallel()
	err := d.WSRemoveAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	err = d.WSRemoveAPIKey(t.Context(), 1)
	assert.NoError(t, err)
}

func TestRemoveSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	err := d.RemoveSubAccount(t.Context(), 1)
	assert.NoError(t, err)
}

func TestWSRemoveSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	err := d.WSRemoveSubAccount(t.Context(), 1)
	assert.NoError(t, err)
}

func TestResetAPIKey(t *testing.T) {
	t.Parallel()
	_, err := d.ResetAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.ResetAPIKey(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSResetAPIKey(t *testing.T) {
	t.Parallel()
	err := d.WSResetAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	err = d.WSResetAPIKey(t.Context(), 1)
	assert.NoError(t, err)
}

func TestSetAnnouncementAsRead(t *testing.T) {
	t.Parallel()
	err := d.SetAnnouncementAsRead(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidID)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.SetAnnouncementAsRead(t.Context(), 1)
	assert.NoError(t, err)
}

func TestSetEmailForSubAccount(t *testing.T) {
	t.Parallel()
	err := d.SetEmailForSubAccount(t.Context(), 0, "wrongemail@wrongemail.com")
	require.ErrorIs(t, err, errInvalidID)
	err = d.SetEmailForSubAccount(t.Context(), 1, "")
	require.ErrorIs(t, err, errInvalidEmailAddress)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.SetEmailForSubAccount(t.Context(), 1, "wrongemail@wrongemail.com")
	assert.NoError(t, err)
}

func TestWSSetEmailForSubAccount(t *testing.T) {
	t.Parallel()
	err := d.WSSetEmailForSubAccount(t.Context(), 0, "wrongemail@wrongemail.com")
	require.ErrorIs(t, err, errInvalidID)
	err = d.WSSetEmailForSubAccount(t.Context(), 1, "")
	require.ErrorIs(t, err, errInvalidEmailAddress)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.WSSetEmailForSubAccount(t.Context(), 1, "wrongemail@wrongemail.com")
	assert.NoError(t, err)
}

func TestSetEmailLanguage(t *testing.T) {
	t.Parallel()
	err := d.SetEmailLanguage(t.Context(), "")
	require.ErrorIs(t, err, errLanguageIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.SetEmailLanguage(t.Context(), "en")
	assert.NoError(t, err)
}

func TestWSSetEmailLanguage(t *testing.T) {
	t.Parallel()
	err := d.WSSetEmailLanguage(t.Context(), "")
	require.ErrorIs(t, err, errLanguageIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.WSSetEmailLanguage(t.Context(), "en")
	assert.NoError(t, err)
}

func TestSetSelfTradingConfig(t *testing.T) {
	t.Parallel()
	_, err := d.SetSelfTradingConfig(t.Context(), "", false)
	require.ErrorIs(t, err, errTradeModeIsRequired)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SetSelfTradingConfig(t.Context(), "reject_taker", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsSetSelfTradingConfig(t *testing.T) {
	t.Parallel()
	_, err := d.WsSetSelfTradingConfig(t.Context(), "", false)
	require.ErrorIs(t, err, errTradeModeIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WsSetSelfTradingConfig(t.Context(), "reject_taker", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestToggleNotificationsFromSubAccount(t *testing.T) {
	t.Parallel()
	err := d.ToggleNotificationsFromSubAccount(t.Context(), 0, false)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.ToggleNotificationsFromSubAccount(t.Context(), 1, false)
	assert.NoError(t, err)
}

func TestWSToggleNotificationsFromSubAccount(t *testing.T) {
	t.Parallel()
	err := d.WSToggleNotificationsFromSubAccount(t.Context(), 0, false)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.WSToggleNotificationsFromSubAccount(t.Context(), 1, false)
	assert.NoError(t, err)
}

func TestTogglePortfolioMargining(t *testing.T) {
	t.Parallel()
	_, err := d.TogglePortfolioMargining(t.Context(), 0, false, false)
	require.ErrorIs(t, err, errUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.TogglePortfolioMargining(t.Context(), 1234, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSTogglePortfolioMargining(t *testing.T) {
	t.Parallel()
	_, err := d.WSTogglePortfolioMargining(t.Context(), 0, false, false)
	require.ErrorIs(t, err, errUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSTogglePortfolioMargining(t.Context(), 1234, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestToggleSubAccountLogin(t *testing.T) {
	t.Parallel()
	err := d.ToggleSubAccountLogin(t.Context(), -1, false)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.ToggleSubAccountLogin(t.Context(), 1, false)
	assert.NoError(t, err)
}

func TestWSToggleSubAccountLogin(t *testing.T) {
	t.Parallel()
	err := d.WSToggleSubAccountLogin(t.Context(), -1, false)
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err = d.WSToggleSubAccountLogin(t.Context(), 1, false)
	assert.NoError(t, err)
}

func TestSubmitBuy(t *testing.T) {
	t.Parallel()
	pairs, err := d.GetEnabledPairs(asset.Futures)
	require.NoError(t, err)
	_, err = d.SubmitBuy(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = d.SubmitBuy(t.Context(), &OrderBuyAndSellParams{
		Instrument: "", OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "",
		Amount: 30, Price: 500000,
		MaxShow: 0, TriggerPrice: 0,
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitBuy(t.Context(), &OrderBuyAndSellParams{
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
	_, err := d.WSSubmitBuy(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = d.WSSubmitBuy(t.Context(), &OrderBuyAndSellParams{
		Instrument: "", OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "",
		Amount: 30, Price: 500000,
		MaxShow: 0, TriggerPrice: 0,
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitBuy(t.Context(), &OrderBuyAndSellParams{
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
	_, err := d.SubmitSell(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	info, err := d.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	_, err = d.SubmitSell(t.Context(), &OrderBuyAndSellParams{OrderType: "limit", Label: "testOrder", TimeInForce: "", Trigger: "", Advanced: "", Amount: info.ContractSize * 3, Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false, RejectPostOnly: false, ReduceOnly: false, MMP: false})
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitSell(t.Context(), &OrderBuyAndSellParams{Instrument: btcPerpInstrument, OrderType: "limit", Label: "testOrder", TimeInForce: "", Trigger: "", Advanced: "", Amount: info.ContractSize * 3, Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false, RejectPostOnly: false, ReduceOnly: false, MMP: false})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitSell(t *testing.T) {
	t.Parallel()
	info, err := d.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	_, err = d.WSSubmitSell(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = d.WSSubmitSell(t.Context(), &OrderBuyAndSellParams{
		Instrument: "", OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "", Amount: info.ContractSize * 3,
		Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false,
		RejectPostOnly: false, ReduceOnly: false, MMP: false,
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitSell(t.Context(), &OrderBuyAndSellParams{
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
	_, err := d.EditOrderByLabel(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = d.EditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: "",
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = d.EditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 0, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.ErrorIs(t, err, errInvalidAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.EditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSEditOrderByLabel(t *testing.T) {
	t.Parallel()
	_, err := d.WSEditOrderByLabel(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = d.WSEditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: "",
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = d.WSEditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
		Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 0, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false,
	})
	require.ErrorIs(t, err, errInvalidAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSEditOrderByLabel(t.Context(), &OrderBuyAndSellParams{
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
	_, err = d.SubmitCancel(t.Context(), "")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitCancel(t.Context(), "incorrectID")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancel(t *testing.T) {
	t.Parallel()
	_, err := d.WSSubmitCancel(t.Context(), "")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitCancel(t.Context(), "incorrectID")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelAll(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitCancelAll(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancelAll(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitCancelAll(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelAllByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.SubmitCancelAllByCurrency(t.Context(), currency.EMPTYCODE, "option", "", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitCancelAllByCurrency(t.Context(), currency.BTC, "option", "", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancelAllByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.WSSubmitCancelAllByCurrency(t.Context(), currency.EMPTYCODE, "option", "", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitCancelAllByCurrency(t.Context(), currency.BTC, "option", "", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelAllByKind(t *testing.T) {
	t.Parallel()
	_, err := d.SubmitCancelAllByKind(t.Context(), currency.EMPTYCODE, "option_combo", "trigger_all", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitCancelAllByKind(t.Context(), currency.ETH, "option_combo", "trigger_all", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsSubmitCancelAllByKind(t *testing.T) {
	t.Parallel()
	_, err := d.WsSubmitCancelAllByKind(t.Context(), currency.EMPTYCODE, "option_combo", "trigger_all", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WsSubmitCancelAllByKind(t.Context(), currency.ETH, "option_combo", "trigger_all", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelAllByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.SubmitCancelAllByInstrument(t.Context(), "", "all", true, true)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitCancelAllByInstrument(t.Context(), btcPerpInstrument, "all", true, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancelAllByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.WSSubmitCancelAllByInstrument(t.Context(), "", "all", true, true)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitCancelAllByInstrument(t.Context(), btcPerpInstrument, "all", true, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelByLabel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitCancelByLabel(t.Context(), "incorrectOrderLabel", currency.EMPTYCODE, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancelByLabel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitCancelByLabel(t.Context(), "incorrectOrderLabel", currency.EMPTYCODE, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitCancelQuotes(t *testing.T) {
	t.Parallel()
	_, err := d.SubmitCancelQuotes(t.Context(), currency.EMPTYCODE, 0, 0, "all", "", futuresTradablePair.String(), "future", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitCancelQuotes(t.Context(), currency.BTC, 0, 0, "all", "", futuresTradablePair.String(), "future", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitCancelQuotes(t *testing.T) {
	t.Parallel()
	_, err := d.WSSubmitCancelQuotes(t.Context(), currency.EMPTYCODE, 0, 0, "all", "", futuresTradablePair.String(), "future", true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitCancelQuotes(t.Context(), currency.BTC, 0, 0, "all", "", futuresTradablePair.String(), "future", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitClosePosition(t *testing.T) {
	t.Parallel()
	_, err := d.SubmitClosePosition(t.Context(), "", "limit", 35000)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitClosePosition(t.Context(), d.formatFuturesTradablePair(futuresTradablePair), "limit", 35000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitClosePosition(t *testing.T) {
	t.Parallel()
	_, err := d.WSSubmitClosePosition(t.Context(), "", "limit", 35000)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitClosePosition(t.Context(), d.formatFuturesTradablePair(futuresTradablePair), "limit", 35000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMargins(t *testing.T) {
	t.Parallel()
	_, err := d.GetMargins(t.Context(), "", 5, 35000)
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = d.GetMargins(t.Context(), d.formatFuturesTradablePair(futuresTradablePair), 0, 35000)
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.GetMargins(t.Context(), d.formatFuturesTradablePair(futuresTradablePair), 5, -1)
	require.ErrorIs(t, err, errInvalidPrice)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetMargins(t.Context(), d.formatFuturesTradablePair(futuresTradablePair), 5, 35000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveMargins(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveMargins(t.Context(), "", 5, 35000)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveMargins(t.Context(), d.formatFuturesTradablePair(futuresTradablePair), 5, 35000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMMPConfig(t *testing.T) {
	t.Parallel()
	_, err := d.GetMMPConfig(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetMMPConfig(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveMMPConfig(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveMMPConfig(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveMMPConfig(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getOpenOrdersByCurrencyResponseJSON = `[{ "time_in_force": "good_til_cancelled", "reduce_only": false, "profit_loss": 0, "price": 0.0028, "post_only": false, "order_type": "limit", "order_state": "open", "order_id": "146062", "max_show": 10, "last_update_timestamp": 1550050597036, "label": "", "is_liquidation": false, "instrument_name": "BTC-15FEB19-3250-P", "filled_amount": 0, "direction": "buy", "creation_timestamp": 1550050597036, "commission": 0, "average_price": 0, "api": true, "amount": 10 } ]`

func TestGetOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	var resp []OrderData
	err := json.Unmarshal([]byte(getOpenOrdersByCurrencyResponseJSON), &resp)
	require.NoError(t, err)
	_, err = d.GetOpenOrdersByCurrency(t.Context(), currency.EMPTYCODE, "option", "all")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetOpenOrdersByCurrency(t.Context(), currency.BTC, "option", "all")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveOpenOrdersByCurrency(t.Context(), currency.EMPTYCODE, "option", "all")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveOpenOrdersByCurrency(t.Context(), currency.BTC, "option", "all")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrdersByLabel(t *testing.T) {
	t.Parallel()
	_, err := d.GetOpenOrdersByLabel(t.Context(), currency.EMPTYCODE, "the-label")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetOpenOrdersByLabel(t.Context(), currency.EURR, "the-label")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOpenOrdersByLabel(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveOpenOrdersByLabel(t.Context(), currency.EMPTYCODE, "the-label")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSRetrieveOpenOrdersByLabel(t.Context(), currency.EURR, "the-label")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetOpenOrdersByInstrument(t.Context(), "", "all")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetOpenOrdersByInstrument(t.Context(), btcPerpInstrument, "all")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveOpenOrdersByInstrument(t.Context(), "", "all")
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveOpenOrdersByInstrument(t.Context(), btcPerpInstrument, "all")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetOrderHistoryByCurrency(t.Context(), currency.EMPTYCODE, "future", 0, 0, false, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetOrderHistoryByCurrency(t.Context(), currency.BTC, "future", 0, 0, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveOrderHistoryByCurrency(t.Context(), currency.EMPTYCODE, "future", 0, 0, false, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveOrderHistoryByCurrency(t.Context(), currency.BTC, "future", 0, 0, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetOrderHistoryByInstrument(t.Context(), "", 0, 0, false, false)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetOrderHistoryByInstrument(t.Context(), btcPerpInstrument, 0, 0, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveOrderHistoryByInstrument(t.Context(), "", 0, 0, false, false)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveOrderHistoryByInstrument(t.Context(), btcPerpInstrument, 0, 0, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderMarginsByID(t *testing.T) {
	t.Parallel()
	_, err := d.GetOrderMarginsByID(t.Context(), []string{})
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetOrderMarginsByID(t.Context(), []string{"21422175153", "21422175154"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveOrderMarginsByID(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveOrderMarginsByID(t.Context(), []string{})
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveOrderMarginsByID(t.Context(), []string{"ETH-349280", "ETH-349279", "ETH-349278"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderState(t *testing.T) {
	t.Parallel()
	_, err := d.GetOrderState(t.Context(), "")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetOrderState(t.Context(), "brokenid123")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrievesOrderState(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrievesOrderState(t.Context(), "")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrievesOrderState(t.Context(), "brokenid123")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderStateByLabel(t *testing.T) {
	t.Parallel()
	_, err := d.GetOrderStateByLabel(t.Context(), currency.EMPTYCODE, "the-label")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetOrderStateByLabel(t.Context(), currency.EURR, "the-label")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrieveOrderStateByLabel(t *testing.T) {
	t.Parallel()
	_, err := d.WsRetrieveOrderStateByLabel(t.Context(), currency.EMPTYCODE, "the-label")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WsRetrieveOrderStateByLabel(t.Context(), currency.EURR, "the-label")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := d.GetTriggerOrderHistory(t.Context(), currency.EMPTYCODE, "", "", 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetTriggerOrderHistory(t.Context(), currency.ETH, "", "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveTriggerOrderHistory(t.Context(), currency.EMPTYCODE, "", "", 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveTriggerOrderHistory(t.Context(), currency.ETH, "", "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getUserTradesByCurrencyResponseJSON = `{"trades": [ { "underlying_price": 204.5, "trade_seq": 3, "trade_id": "ETH-2696060", "timestamp": 1590480363130, "tick_direction": 2, "state": "filled", "reduce_only": false, "price": 0.361, "post_only": false, "order_type": "limit", "order_id": "ETH-584827850", "matching_id": null, "mark_price": 0.364585, "liquidity": "T", "iv": 0, "instrument_name": "ETH-29MAY20-130-C", "index_price": 203.72, "fee_currency": "ETH", "fee": 0.002, "direction": "sell", "amount": 5 }, { "underlying_price": 204.82, "trade_seq": 3, "trade_id": "ETH-2696062", "timestamp": 1590480416119, "tick_direction": 0, "state": "filled", "reduce_only": false, "price": 0.015, "post_only": false, "order_type": "limit", "order_id": "ETH-584828229", "matching_id": null, "mark_price": 0.000596, "liquidity": "T", "iv": 352.91, "instrument_name": "ETH-29MAY20-140-P", "index_price": 204.06, "fee_currency": "ETH", "fee": 0.002, "direction": "buy", "amount": 5 } ], "has_more": true }`

func TestGetUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	var resp *UserTradesData
	err := json.Unmarshal([]byte(getUserTradesByCurrencyResponseJSON), &resp)
	require.NoError(t, err)
	_, err = d.GetUserTradesByCurrency(t.Context(), currency.EMPTYCODE, "future", "", "", "asc", 0, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetUserTradesByCurrency(t.Context(), currency.ETH, "future", "", "", "asc", 0, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveUserTradesByCurrency(t.Context(), currency.EMPTYCODE, "future", "", "", "asc", 0, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveUserTradesByCurrency(t.Context(), currency.ETH, "future", "", "", "asc", 0, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetUserTradesByCurrencyAndTime(t.Context(), currency.EMPTYCODE, "future", "default", 5, time.Now().Add(-time.Hour*10), time.Now().Add(-time.Hour*1))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetUserTradesByCurrencyAndTime(t.Context(), currency.ETH, "future", "default", 5, time.Now().Add(-time.Hour*10), time.Now().Add(-time.Hour*1))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveUserTradesByCurrencyAndTime(t.Context(), currency.EMPTYCODE, "future", "default", 5, time.Now().Add(-time.Hour*4), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveUserTradesByCurrencyAndTime(t.Context(), currency.ETH, "future", "default", 5, time.Now().Add(-time.Hour*4), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetUserTradesByInstrument(t.Context(), "", "asc", 5, 10, 4, true)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetUserTradesByInstrument(t.Context(), btcPerpInstrument, "asc", 5, 10, 4, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrieveUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.WsRetrieveUserTradesByInstrument(t.Context(), "", "asc", 5, 10, 4, true)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WsRetrieveUserTradesByInstrument(t.Context(), btcPerpInstrument, "asc", 5, 10, 4, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetUserTradesByInstrumentAndTime(t.Context(), "", "asc", 10, time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetUserTradesByInstrumentAndTime(t.Context(), btcPerpInstrument, "asc", 10, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveUserTradesByInstrumentAndTime(t.Context(), "", "asc", 10, false, time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveUserTradesByInstrumentAndTime(t.Context(), btcPerpInstrument, "asc", 10, false, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTradesByOrder(t *testing.T) {
	t.Parallel()
	_, err := d.GetUserTradesByOrder(t.Context(), "", "default")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetUserTradesByOrder(t.Context(), "wrongOrderID", "default")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserTradesByOrder(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveUserTradesByOrder(t.Context(), "", "default")
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveUserTradesByOrder(t.Context(), "wrongOrderID", "default")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetMMP(t *testing.T) {
	t.Parallel()
	err := d.ResetMMP(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err = d.ResetMMP(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestWSResetMMP(t *testing.T) {
	t.Parallel()
	err := d.WSResetMMP(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err = d.WSResetMMP(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestSendRequestForQuote(t *testing.T) {
	t.Parallel()
	err := d.SendRequestForQuote(t.Context(), "", 1000, order.Buy)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err = d.SendRequestForQuote(t.Context(), d.formatFuturesTradablePair(futuresTradablePair), 1000, order.Buy)
	assert.NoError(t, err)
}

func TestWSSendRequestForQuote(t *testing.T) {
	t.Parallel()
	err := d.WSSendRequestForQuote(t.Context(), "", 1000, order.Buy)
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err = d.WSSendRequestForQuote(t.Context(), d.formatFuturesTradablePair(futuresTradablePair), 1000, order.Buy)
	assert.NoError(t, err)
}

func TestSetMMPConfig(t *testing.T) {
	t.Parallel()
	err := d.SetMMPConfig(t.Context(), currency.EMPTYCODE, kline.FiveMin, 5, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err = d.SetMMPConfig(t.Context(), currency.BTC, kline.FiveMin, 5, 0, 0)
	assert.NoError(t, err)
}

func TestWSSetMMPConfig(t *testing.T) {
	t.Parallel()
	err := d.WSSetMMPConfig(t.Context(), currency.EMPTYCODE, kline.FiveMin, 5, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err = d.WSSetMMPConfig(t.Context(), currency.BTC, kline.FiveMin, 5, 0, 0)
	assert.NoError(t, err)
}

func TestGetSettlementHistoryByCurency(t *testing.T) {
	t.Parallel()
	_, err := d.GetSettlementHistoryByCurency(t.Context(), currency.EMPTYCODE, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetSettlementHistoryByCurency(t.Context(), currency.BTC, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveSettlementHistoryByCurency(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveSettlementHistoryByCurency(t.Context(), currency.EMPTYCODE, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveSettlementHistoryByCurency(t.Context(), currency.BTC, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const getSettlementHistoryByInstrumentResponseJSON = `{"settlements": [ { "type": "settlement", "timestamp": 1550475692526, "session_profit_loss": 0.038358299, "profit_loss": -0.001783937, "position": -66, "mark_price": 121.67, "instrument_name": "ETH-22FEB19", "index_price": 119.8 } ], "continuation": "xY7T6cusbMBNpH9SNmKb94jXSBxUPojJEdCPL4YociHBUgAhWQvEP" }`

func TestGetSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	var result *PrivateSettlementsHistoryData
	err := json.Unmarshal([]byte(getSettlementHistoryByInstrumentResponseJSON), &result)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err = d.GetSettlementHistoryByInstrument(t.Context(), btcPerpInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveSettlementHistoryByInstrument(t.Context(), "", "settlement", "", 10, time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, errInvalidInstrumentName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveSettlementHistoryByInstrument(t.Context(), btcPerpInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitEdit(t *testing.T) {
	t.Parallel()
	_, err := d.SubmitEdit(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = d.SubmitEdit(t.Context(), &OrderBuyAndSellParams{OrderID: "", Advanced: "", TriggerPrice: 0.001, Price: 100000, Amount: 123})
	require.ErrorIs(t, err, errInvalidID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.SubmitEdit(t.Context(), &OrderBuyAndSellParams{OrderID: "incorrectID", Advanced: "", TriggerPrice: 0.001, Price: 100000, Amount: 123})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSSubmitEdit(t *testing.T) {
	t.Parallel()
	_, err := d.WSSubmitEdit(t.Context(), &OrderBuyAndSellParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSSubmitEdit(t.Context(), &OrderBuyAndSellParams{
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
	_, err := d.GetComboIDs(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.GetComboIDs(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveComboIDS(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveComboIDs(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	combos, err := d.WSRetrieveComboIDs(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	assert.NotEmpty(t, combos)
}

func TestGetComboDetails(t *testing.T) {
	t.Parallel()
	_, err := d.GetComboDetails(t.Context(), "")
	require.ErrorIs(t, err, errInvalidComboID)

	result, err := d.GetComboDetails(t.Context(), futureComboTradablePair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveComboDetails(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveComboDetails(t.Context(), "")
	require.ErrorIs(t, err, errInvalidComboID)

	result, err := d.WSRetrieveComboDetails(t.Context(), futureComboTradablePair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCombos(t *testing.T) {
	t.Parallel()
	_, err := d.GetCombos(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.GetCombos(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateCombo(t *testing.T) {
	t.Parallel()
	_, err := d.CreateCombo(t.Context(), []ComboParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	instruments, err := d.GetEnabledPairs(asset.Futures)
	require.NoError(t, err)
	if len(instruments) < 2 {
		t.Skip("no enough instrument found")
	}
	_, err = d.CreateCombo(t.Context(), []ComboParam{
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
	_, err = d.CreateCombo(t.Context(), []ComboParam{
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
	_, err = d.CreateCombo(t.Context(), []ComboParam{
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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.CreateCombo(t.Context(), []ComboParam{
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
	_, err := d.WSCreateCombo(t.Context(), []ComboParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	instruments, err := d.GetEnabledPairs(asset.Futures)
	require.NoError(t, err)
	if len(instruments) < 2 {
		t.Skip("no enough instrument found")
	}
	_, err = d.WSCreateCombo(t.Context(), []ComboParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	_, err = d.WSCreateCombo(t.Context(), []ComboParam{
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
	_, err = d.WSCreateCombo(t.Context(), []ComboParam{
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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSCreateCombo(t.Context(), []ComboParam{
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
	_, err := d.VerifyBlockTrade(t.Context(), time.Now(), "", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingNonce)
	_, err = d.VerifyBlockTrade(t.Context(), time.Now(), "nonce-string", "", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = d.VerifyBlockTrade(t.Context(), time.Now(), "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	info, err := d.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	require.NotNil(t, info)
	_, err = d.VerifyBlockTrade(t.Context(), time.Time{}, "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.ErrorIs(t, err, errZeroTimestamp)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.VerifyBlockTrade(t.Context(), time.Now(), "something", "maker", currency.EMPTYCODE, []BlockTradeParam{
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
	_, err := d.WSVerifyBlockTrade(t.Context(), time.Now(), "", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingNonce)
	_, err = d.WSVerifyBlockTrade(t.Context(), time.Now(), "nonce-string", "", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = d.WSVerifyBlockTrade(t.Context(), time.Now(), "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	info, err := d.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	require.NotNil(t, info)
	_, err = d.WSVerifyBlockTrade(t.Context(), time.Time{}, "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.ErrorIs(t, err, errZeroTimestamp)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSVerifyBlockTrade(t.Context(), time.Now(), "sdjkafdad", "maker", currency.EMPTYCODE, []BlockTradeParam{
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
	err := d.WsInvalidateBlockTradeSignature(t.Context(), "")
	require.ErrorIs(t, err, errMissingSignature)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err = d.InvalidateBlockTradeSignature(t.Context(), "verified_signature_string")
	assert.NoError(t, err)
}

func TestWsInvalidateBlockTradeSignature(t *testing.T) {
	t.Parallel()
	err := d.WsInvalidateBlockTradeSignature(t.Context(), "")
	require.ErrorIs(t, err, errMissingSignature)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err = d.WsInvalidateBlockTradeSignature(t.Context(), "verified_signature_string")
	assert.NoError(t, err)
}

func TestExecuteBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := d.ExecuteBlockTrade(t.Context(), time.Now(), "", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingNonce)
	_, err = d.ExecuteBlockTrade(t.Context(), time.Now(), "nonce-string", "", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = d.ExecuteBlockTrade(t.Context(), time.Now(), "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	info, err := d.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	require.NotNil(t, info)
	_, err = d.ExecuteBlockTrade(t.Context(), time.Time{}, "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	require.ErrorIs(t, err, errZeroTimestamp)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.ExecuteBlockTrade(t.Context(), time.Now(), "something", "maker", currency.EMPTYCODE, []BlockTradeParam{
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
	_, err := d.WSExecuteBlockTrade(t.Context(), time.Now(), "", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingNonce)
	_, err = d.WSExecuteBlockTrade(t.Context(), time.Now(), "nonce-string", "", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = d.WSExecuteBlockTrade(t.Context(), time.Now(), "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	info, err := d.GetInstrument(t.Context(), btcPerpInstrument)
	require.NoError(t, err)
	require.NotNil(t, info)
	_, err = d.WSExecuteBlockTrade(t.Context(), time.Time{}, "nonce-string", "maker", currency.EMPTYCODE, []BlockTradeParam{{
		Price:          0.777 * 22000,
		InstrumentName: btcPerpInstrument,
		Direction:      "buy",
		Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
	}})
	require.ErrorIs(t, err, errZeroTimestamp)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSExecuteBlockTrade(t.Context(), time.Now(), "sdjkafdad", "maker", currency.EMPTYCODE, []BlockTradeParam{
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
	_, err = d.GetUserBlockTrade(t.Context(), "")
	require.ErrorIs(t, err, errMissingBlockTradeID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetUserBlockTrade(t.Context(), "12345567")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveUserBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveUserBlockTrade(t.Context(), "")
	require.ErrorIs(t, err, errMissingBlockTradeID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveUserBlockTrade(t.Context(), "12345567")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLastBlockTradesbyCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastBlockTradesByCurrency(t.Context(), currency.EMPTYCODE, "", "", 5)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetLastBlockTradesByCurrency(t.Context(), currency.SOL, "", "", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSRetrieveLastBlockTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveLastBlockTradesByCurrency(t.Context(), currency.EMPTYCODE, "", "", 5)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WSRetrieveLastBlockTradesByCurrency(t.Context(), currency.SOL, "", "", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMovePositions(t *testing.T) {
	t.Parallel()
	_, err := d.MovePositions(t.Context(), currency.EMPTYCODE, 123, 345, []BlockTradeParam{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.MovePositions(t.Context(), currency.BTC, 0, 345, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingSubAccountID)
	_, err = d.MovePositions(t.Context(), currency.BTC, 123, 0, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingSubAccountID)
	_, err = d.MovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "",
			Direction:      "buy",
			Amount:         100,
		},
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = d.MovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         0,
		},
	})
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.MovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          -4,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         20,
		},
	})
	require.ErrorIs(t, err, errInvalidPrice)
	info, err := d.GetInstrument(t.Context(), "BTC-PERPETUAL")
	require.NoError(t, err)
	require.NotNil(t, info)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.MovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
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
	_, err := d.WSMovePositions(t.Context(), currency.EMPTYCODE, 123, 345, []BlockTradeParam{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = d.WSMovePositions(t.Context(), currency.BTC, 0, 345, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingSubAccountID)
	_, err = d.WSMovePositions(t.Context(), currency.BTC, 123, 0, []BlockTradeParam{})
	require.ErrorIs(t, err, errMissingSubAccountID)
	_, err = d.WSMovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "",
			Direction:      "buy",
			Amount:         100,
		},
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = d.WSMovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         0,
		},
	})
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.WSMovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
		{
			Price:          -4,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         20,
		},
	})
	require.ErrorIs(t, err, errInvalidPrice)
	info, err := d.GetInstrument(t.Context(), "BTC-PERPETUAL")
	require.NoError(t, err)
	require.NotNil(t, info)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WSMovePositions(t.Context(), currency.BTC, 123, 345, []BlockTradeParam{
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
	_, err := d.SimulateBlockTrade(t.Context(), "", []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = d.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	_, err = d.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "",
			Direction:      "buy",
			Amount:         10,
		},
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = d.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "",
			Amount:         10,
		},
	})
	require.ErrorIs(t, err, errInvalidOrderSideOrDirection)
	_, err = d.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "sell",
			Amount:         0,
		},
	})
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          -1,
			InstrumentName: btcPerpInstrument,
			Direction:      "sell",
			Amount:         10,
		},
	})
	require.ErrorIs(t, err, errInvalidPrice)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	info, err := d.GetInstrument(t.Context(), "BTC-PERPETUAL")
	require.NoError(t, err)
	result, err := d.SimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
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
	_, err := d.WsSimulateBlockTrade(t.Context(), "", []BlockTradeParam{})
	require.ErrorIs(t, err, errInvalidTradeRole)
	_, err = d.WsSimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{})
	require.ErrorIs(t, err, errNoArgumentPassed)
	_, err = d.WsSimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "",
			Direction:      "buy",
			Amount:         10,
		},
	})
	require.ErrorIs(t, err, errInvalidInstrumentName)
	_, err = d.WsSimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "",
			Amount:         100,
		},
	})
	require.ErrorIs(t, err, errInvalidOrderSideOrDirection)
	_, err = d.WsSimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "sell",
			Amount:         0,
		},
	})
	require.ErrorIs(t, err, errInvalidAmount)
	_, err = d.WsSimulateBlockTrade(t.Context(), "maker", []BlockTradeParam{
		{
			Price:          -1,
			InstrumentName: btcPerpInstrument,
			Direction:      "sell",
			Amount:         100,
		},
	})
	require.ErrorIs(t, err, errInvalidPrice)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	info, err := d.GetInstrument(t.Context(), "BTC-PERPETUAL")
	require.NoError(t, err)
	require.NotNil(t, info)
	result, err := d.WsSimulateBlockTrade(t.Context(), "taker", []BlockTradeParam{
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
	if !d.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(d) {
		d.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := d.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	d := new(Deribit) //nolint:govet // Intentional lexical scope shadow
	require.NoError(t, testexch.Setup(d), "Test instance Setup must not error")

	d.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := d.generateSubscriptions()
	require.NoError(t, err)
	exp := subscription.List{}
	for _, s := range d.Features.Subscriptions {
		for _, a := range d.GetAssetTypes(true) {
			if !d.IsAssetWebsocketSupported(a) {
				continue
			}
			pairs, err := d.GetEnabledPairs(a)
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

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.UpdateAccountInfo(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetAccountFundingHistory(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Empty)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	var result []trade.Data
	var err error
	for assetType, cp := range assetTypeToPairsMap {
		result, err = d.GetRecentTrades(t.Context(), cp, assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
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
		result, err = d.CancelAllOrders(t.Context(), orderCancellation)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	for assetType, cp := range assetTypeToPairsMap {
		result, err := d.GetOrderInfo(t.Context(), "1234", cp, assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.GetDepositAddress(t.Context(), currency.BTC, "", "")
	require.ErrorIs(t, err, common.ErrNoResponse)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:    d.Name,
		Amount:      1,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	getOrdersRequest := order.MultiOrderRequest{
		Type: order.AnyType, AssetType: asset.Futures,
		Side: order.AnySide, Pairs: currency.Pairs{futuresTradablePair},
	}

	for assetType, cp := range assetTypeToPairsMap {
		getOrdersRequest.Pairs = []currency.Pair{cp}
		getOrdersRequest.AssetType = assetType
		result, err := d.GetActiveOrders(t.Context(), &getOrdersRequest)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	for assetType, cp := range assetTypeToPairsMap {
		result, err := d.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			Type: order.AnyType, AssetType: assetType,
			Side: order.AnySide, Pairs: []currency.Pair{cp},
		})
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s pair %s", err, assetType, cp)
		require.NotNilf(t, result, "expected result not to be nil for asset type %s pair %s", assetType, cp)
	}
}

func TestGetAssetFromPair(t *testing.T) {
	var assetTypeNew asset.Item
	for _, assetType := range []asset.Item{asset.Spot, asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo} {
		availablePairs, err := d.GetEnabledPairs(assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s", err, assetType)
		require.NotNilf(t, availablePairs, "expected result not to be nil for asset type %s", assetType)

		format, err := d.GetPairFormat(assetType, true)
		require.NoError(t, err)

		for id, cp := range availablePairs {
			t.Run(strconv.Itoa(id), func(t *testing.T) {
				assetTypeNew, err = getAssetFromPair(cp.Format(format))
				require.Equalf(t, assetType, assetTypeNew, "expected %s, but found %s for pair string %s", assetType.String(), assetTypeNew.String(), cp.Format(format))
			})
		}
	}

	cp, err := currency.NewPairFromString("some_thing_else")
	require.NoError(t, err)
	_, err = getAssetFromPair(cp)
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat)
}

func TestGetAssetPairByInstrument(t *testing.T) {
	t.Parallel()
	for _, assetType := range []asset.Item{asset.Spot, asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo} {
		availablePairs, err := d.GetAvailablePairs(assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type %s", err, assetType)
		require.NotNilf(t, availablePairs, "expected result not to be nil for asset type %s", assetType)
		for _, cp := range availablePairs {
			t.Run(fmt.Sprintf("%s %s", assetType, cp), func(t *testing.T) {
				t.Parallel()
				extractedPair, extractedAsset, err := d.getAssetPairByInstrument(cp.String())
				assert.NoError(t, err)
				assert.Equal(t, cp.String(), extractedPair.String())
				assert.Equal(t, assetType.String(), extractedAsset.String())
			})
		}
	}
	t.Run("empty asset, empty pair", func(t *testing.T) {
		t.Parallel()
		_, _, err := d.getAssetPairByInstrument("")
		assert.ErrorIs(t, err, errInvalidInstrumentName)
	})
	t.Run("thisIsAFakeCurrency", func(t *testing.T) {
		t.Parallel()
		_, _, err := d.getAssetPairByInstrument("thisIsAFakeCurrency")
		assert.ErrorIs(t, err, errUnsupportedInstrumentFormat)
	})
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
	result, err := d.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
	if !sharedtestvalues.AreAPICredentialsSet(d) {
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
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat)
}

func TestGetTime(t *testing.T) {
	t.Parallel()
	result, err := d.GetTime(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	result, err := d.GetServerTime(t.Context(), asset.Empty)
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
		_, err := d.ModifyOrder(t.Context(), param)
		require.ErrorIs(t, err, errIncoming)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Futures, OrderID: "1234", Pair: futuresTradablePair, Amount: 2})
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = d.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Options, OrderID: "1234", Pair: optionsTradablePair, Amount: 2})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
	}
	for assetType, cp := range assetTypeToPairsMap {
		orderCancellation.AssetType = assetType
		orderCancellation.Pair = cp
		err := d.CancelOrder(t.Context(), orderCancellation)
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
			err := d.wsHandleData(t.Context(), []byte(v))
			require.NoErrorf(t, err, "%s: Received unexpected error for", k)
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
			instrument := d.formatFuturesTradablePair(pair)
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
		assert.Equal(t, exp, d.optionPairToString(pair), "optionPairToString should return correctly")
	}
}

func TestWSRetrieveCombos(t *testing.T) {
	t.Parallel()
	_, err := d.WSRetrieveCombos(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := d.WSRetrieveCombos(t.Context(), futureComboTradablePair.Base)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := d.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)
	result, err := d.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  futuresTradablePair,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := d.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	require.NoErrorf(t, err, "Error fetching %s pairs for test: %v", asset.Spot, err)
	instrumentInfo, err := d.GetInstruments(t.Context(), currency.BTC, d.GetAssetKind(asset.Spot), false)
	require.NoError(t, err)
	require.NotEmpty(t, instrumentInfo, "instrument information must not be empty")
	limits, err := d.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
	require.NoErrorf(t, err, "Asset: %s Pair: %s Err: %v", asset.Spot, spotTradablePair, err)
	var instrumentDetail *InstrumentData
	for a := range instrumentInfo {
		if instrumentInfo[a].InstrumentName == spotTradablePair.String() {
			instrumentDetail = instrumentInfo[a]
			break
		}
	}
	require.NotNil(t, instrumentDetail, "instrument required to be found")
	require.Equalf(t, instrumentDetail.TickSize, limits.PriceStepIncrementSize, "Asset: %s Pair: %s Expected: %f Got: %f", asset.Spot, spotTradablePair, instrumentDetail.TickSize, limits.MinimumBaseAmount)
	assert.Equalf(t, instrumentDetail.MinimumTradeAmount, limits.MinimumBaseAmount, "Pair: %s Expected: %f Got: %f", spotTradablePair, instrumentDetail.MinimumTradeAmount, limits.MinimumBaseAmount)
}

func TestGetLockedStatus(t *testing.T) {
	t.Parallel()
	result, err := d.GetLockedStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSayHello(t *testing.T) {
	t.Parallel()
	result, err := d.SayHello(t.Context(), "Thrasher", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrieveCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WsRetrieveCancelOnDisconnect(t.Context(), "connection")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsDisableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WsDisableCancelOnDisconnect(t.Context(), "connection")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.EnableCancelOnDisconnect(t.Context(), "account")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsEnableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	result, err := d.WsEnableCancelOnDisconnect(t.Context(), "connection")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogout(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err := d.WsLogout(t.Context(), true)
	assert.NoError(t, err)
}

func TestExchangeToken(t *testing.T) {
	t.Parallel()
	_, err := d.ExchangeToken(t.Context(), "", 1234)
	require.ErrorIs(t, err, errRefreshTokenRequired)
	_, err = d.ExchangeToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", 0)
	require.ErrorIs(t, err, errSubjectIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.ExchangeToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsExchangeToken(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	result, err := d.WsExchangeToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestForkToken(t *testing.T) {
	t.Parallel()
	_, err := d.ForkToken(t.Context(), "", "Sami")
	require.ErrorIs(t, err, errRefreshTokenRequired)
	_, err = d.ForkToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", "")
	require.ErrorIs(t, err, errSessionNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.ForkToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", "Sami")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsForkToken(t *testing.T) {
	t.Parallel()
	_, err := d.WsForkToken(t.Context(), "", "Sami")
	require.ErrorIs(t, err, errRefreshTokenRequired)
	_, err = d.WsForkToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", "")
	require.ErrorIs(t, err, errSessionNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateAPIEndpoints)
	result, err := d.WsForkToken(t.Context(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", "Sami")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := d.GetFuturesContractDetails(t.Context(), asset.Binary)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	result, err := d.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = d.GetFuturesContractDetails(t.Context(), asset.FutureCombo)
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
		_, err := d.GetFuturesPositionSummary(t.Context(), param)
		require.ErrorIs(t, err, errIncoming)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	req := &futures.PositionSummaryRequest{
		Asset: asset.Futures,
		Pair:  currency.NewPair(currency.BTC, currency.NewCode(perpString)),
	}
	result, err := d.GetFuturesPositionSummary(t.Context(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := d.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.SOL.Item,
		Quote: currency.USDC.Item,
		Asset: asset.Spot,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = d.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  optionsTradablePair.Base.Item,
		Quote: optionsTradablePair.Quote.Item,
		Asset: asset.Options,
	})
	require.NoError(t, err)

	_, err = d.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.NewCode(perpString).Item,
		Asset: asset.Futures,
	})
	require.NoError(t, err)

	_, err = d.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.NewCode("XRP").Item,
		Quote: currency.NewCode("USDC-PERPETUAL").Item,
		Asset: asset.Futures,
	})
	require.NoError(t, err)

	_, err = d.GetOpenInterest(t.Context(), key.PairAsset{
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
				is, err := d.IsPerpetualFutureCurrency(assetType, instances[i].Pair)
				require.ErrorIsf(t, err, instances[i].Error, "expected %v, got %v for asset: %s pair: %s", instances[i].Error, err, assetType.String(), instances[i].Pair.String())
				require.Equalf(t, is, instances[i].Response, "expected %v, got %v for asset: %s pair: %s", instances[i].Response, is, assetType.String(), instances[i].Pair.String())
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
	_, err = d.GetHistoricalFundingRates(t.Context(), r)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	r.Asset = asset.Futures
	result, err := d.GetHistoricalFundingRates(t.Context(), r)
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
		result, err := d.GetResolutionFromInterval(intervalStringMap[x].Interval)
		require.ErrorIs(t, err, intervalStringMap[x].Error)
		require.Equal(t, intervalStringMap[x].IntervalString, result)
	}
}

func TestGetValidatedCurrencyCode(t *testing.T) {
	t.Parallel()
	pairs := map[currency.Pair]string{
		currency.NewPairWithDelimiter(currencySOL, "21OCT22-20-C", "-"): currencySOL,
		currency.NewPairWithDelimiter(currencyBTC, perpString, "-"):     currencyBTC,
		currency.NewPairWithDelimiter(currencyETH, perpString, "-"):     currencyETH,
		currency.NewPairWithDelimiter(currencySOL, perpString, "-"):     currencySOL,
		currency.NewPairWithDelimiter("AVAX_USDC", perpString, "-"):     currencyUSDC,
		currency.NewPairWithDelimiter(currencyBTC, "USDC", "_"):         currencyBTC,
		currency.NewPairWithDelimiter(currencyETH, "USDC", "_"):         currencyETH,
		currency.NewPairWithDelimiter("DOT", "USDC-PERPETUAL", "_"):     currencyUSDC,
		currency.NewPairWithDelimiter("DOT", "USDT-PERPETUAL", "_"):     currencyUSDT,
		currency.EMPTYPAIR: "any",
	}
	for x := range pairs {
		result := getValidatedCurrencyCode(x)
		require.Equalf(t, pairs[x], result, "expected: %s actual  : %s for currency pair: %v", x, result, pairs[x])
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := d.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	for _, a := range d.GetAssetTypes(false) {
		var pairs currency.Pairs
		pairs, err = d.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		var resp string
		resp, err = d.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
	// specific test to ensure perps work
	cp := currency.NewPair(currency.BTC, currency.NewCode("USDC-PERPETUAL"))
	resp, err := d.GetCurrencyTradeURL(t.Context(), asset.Futures, cp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
	// specific test to ensure options with dates work
	cp = currency.NewPair(currency.BTC, currency.NewCode("14JUN24-62000-C"))
	resp, err = d.GetCurrencyTradeURL(t.Context(), asset.Options, cp)
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
