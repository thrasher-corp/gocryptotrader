package binance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                      = ""
	apiSecret                   = ""
	canManipulateRealOrders     = false
	canManipulateAPICredentials = false
	useTestNet                  = false

	apiStreamingIsNotConnected = "API streaming is not connected"
)

var (
	e *Exchange

	// enabled and active tradable pairs used to test endpoints.
	spotTradablePair, usdtmTradablePair, coinmTradablePair, optionsTradablePair currency.Pair

	assetToTradablePairMap map[asset.Item]currency.Pair
)

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

// getTime returns a static time for mocking endpoints, if mock is not enabled
// this will default to time now with a window size of 30 days.
// Mock details are unix seconds; start = 1577836800 and end = 1580515200
func getTime() (startTime, endTime time.Time) {
	// if mockTests {
	return time.UnixMilli(1744103854944), time.UnixMilli(1744190254944)
	// }
	// tn := time.Now()
	// offset := time.Hour * 24 * 6
	// return tn.Add(-offset), tn
}

func TestUServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.UServerTime(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	assetTypes := e.GetAssetTypes(true)
	for _, a := range assetTypes {
		st, err := e.GetServerTime(t.Context(), a)
		require.NoError(t, err)
		assert.NotEmpty(t, st)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	for assetType, pair := range assetToTradablePairMap {
		r, err := e.UpdateTicker(t.Context(), pair, assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type: %s pair: %v", err, assetType, pair)
		assert.NotNilf(t, r, "unexpected value nil for asset type: %s pair: %v", assetType, pair)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	enabledAssets := e.GetAssetTypes(true)
	for _, assetType := range enabledAssets {
		err := e.UpdateTickers(t.Context(), assetType)
		assert.NoError(t, err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	assetPairMapTempo := map[asset.Item]currency.Pair{}
	if mockTests {
		cp, err := currency.NewPairFromString("BTCUSDT")
		require.NoError(t, err)
		assetPairMapTempo[asset.Spot], assetPairMapTempo[asset.Margin], assetPairMapTempo[asset.USDTMarginedFutures] = cp, cp, cp
		cp, err = currency.NewPairFromString("BTCUSD_PERP")
		require.NoError(t, err)
		assetPairMapTempo[asset.CoinMarginedFutures] = cp
		cp, err = currency.NewPairFromString("ETH-240927-3800-P")
		require.NoError(t, err)
		assetPairMapTempo[asset.Options] = cp
	} else {
		assetPairMapTempo = assetToTradablePairMap
	}
	for assetType, tp := range assetPairMapTempo {
		result, err := e.UpdateOrderbook(t.Context(), tp, assetType)
		require.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestUExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := e.UExchangeInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UFuturesOrderbook(t.Context(), currency.EMPTYPAIR, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.UFuturesOrderbook(t.Context(), usdtmTradablePair, 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestURecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.URecentTrades(t.Context(), currency.EMPTYPAIR, "", 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.URecentTrades(t.Context(), usdtmTradablePair, "", 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCompressedTrades(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.UCompressedTrades(t.Context(), currency.EMPTYPAIR, "", 5, startTime, endTime)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.UCompressedTrades(t.Context(), usdtmTradablePair, "", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UCompressedTrades(t.Context(), usdtmTradablePair, "", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.UKlineData(t.Context(), currency.EMPTYPAIR, "1d", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UKlineData(t.Context(), usdtmTradablePair, "", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	startTime, endTime := getTime()
	_, err = e.UKlineData(t.Context(), usdtmTradablePair, "", 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.UKlineData(t.Context(), usdtmTradablePair, "1d", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UKlineData(t.Context(), usdtmTradablePair, "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUFuturesContinuousKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetUFuturesContinuousKlineData(t.Context(), currency.EMPTYPAIR, "CURRENT_QUARTER", "1d", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetUFuturesContinuousKlineData(t.Context(), usdtmTradablePair, "", "1d", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errContractTypeIsRequired)

	_, err = e.GetUFuturesContinuousKlineData(t.Context(), usdtmTradablePair, "CURRENT_QUARTER", "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	startTime, endTime := getTime()
	_, err = e.GetUFuturesContinuousKlineData(t.Context(), usdtmTradablePair, "CURRENT_QUARTER", "", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetUFuturesContinuousKlineData(t.Context(), usdtmTradablePair, "CURRENT_QUARTER", "1d", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexOrCandlesticPriceKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexOrCandlesticPriceKlineData(t.Context(), currency.EMPTYPAIR, "1d", time.Time{}, time.Now(), 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetIndexOrCandlesticPriceKlineData(t.Context(), usdtmTradablePair, "", time.Time{}, time.Now(), 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetIndexOrCandlesticPriceKlineData(t.Context(), usdtmTradablePair, "1d", time.Time{}, time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPriceKlineCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceKlineCandlesticks(t.Context(), currency.EMPTYPAIR, "1d", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetMarkPriceKlineCandlesticks(t.Context(), usdtmTradablePair, "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	startTime, endTime := getTime()
	_, err = e.GetMarkPriceKlineCandlesticks(t.Context(), usdtmTradablePair, "1d", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetMarkPriceKlineCandlesticks(t.Context(), usdtmTradablePair, "1d", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPremiumIndexKlineCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetPremiumIndexKlineCandlesticks(t.Context(), currency.EMPTYPAIR, "1d", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetPremiumIndexKlineCandlesticks(t.Context(), usdtmTradablePair, "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	startTime, endTime := getTime()
	_, err = e.GetPremiumIndexKlineCandlesticks(t.Context(), usdtmTradablePair, "1d", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetPremiumIndexKlineCandlesticks(t.Context(), usdtmTradablePair, "1d", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := e.UGetMarkPrice(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UGetMarkPrice(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetFundingHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	result, err := e.UGetFundingHistory(t.Context(), usdtmTradablePair, 1000, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err = e.UGetFundingHistory(t.Context(), usdtmTradablePair, 1000, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestU24HTickerPriceChangeStats(t *testing.T) {
	t.Parallel()
	result, err := e.U24HTickerPriceChangeStats(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.U24HTickerPriceChangeStats(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	result, err := e.USymbolPriceTickerV1(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.USymbolPriceTickerV1(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUSymbolPriceTickerV2(t *testing.T) {
	t.Parallel()
	result, err := e.USymbolPriceTickerV2(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
	result, err = e.USymbolPriceTickerV2(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	result, err := e.USymbolOrderbookTicker(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
	result, err = e.USymbolOrderbookTicker(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.UOpenInterest(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.UOpenInterest(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuarterlyContractSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetQuarterlyContractSettlementPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetQuarterlyContractSettlementPrice(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUOpenInterestStats(t *testing.T) {
	t.Parallel()
	_, err := e.UOpenInterestStats(t.Context(), currency.EMPTYPAIR, "5m", 1, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UOpenInterestStats(t.Context(), usdtmTradablePair, "", 1, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	startTime, endTime := getTime()
	_, err = e.UOpenInterestStats(t.Context(), usdtmTradablePair, "5m", 1, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.UOpenInterestStats(t.Context(), usdtmTradablePair, "5m", 1, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UOpenInterestStats(t.Context(), usdtmTradablePair, "1d", 10, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUTopAcccountsLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := e.UTopAcccountsLongShortRatio(t.Context(), currency.EMPTYPAIR, "5m", 2, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UTopAcccountsLongShortRatio(t.Context(), usdtmTradablePair, "", 2, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	startTime, endTime := getTime()
	_, err = e.UTopAcccountsLongShortRatio(t.Context(), usdtmTradablePair, "5m", 2, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.UTopAcccountsLongShortRatio(t.Context(), usdtmTradablePair, "5m", 2, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UTopAcccountsLongShortRatio(t.Context(), usdtmTradablePair, "5m", 2, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUTopPostionsLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := e.UTopPostionsLongShortRatio(t.Context(), currency.EMPTYPAIR, "5m", 3, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UTopPostionsLongShortRatio(t.Context(), usdtmTradablePair, "", 3, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	startTime, endTime := getTime()
	_, err = e.UTopPostionsLongShortRatio(t.Context(), usdtmTradablePair, "5m", 3, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.UTopPostionsLongShortRatio(t.Context(), usdtmTradablePair, "5m", 3, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UTopPostionsLongShortRatio(t.Context(), usdtmTradablePair, "1d", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGlobalLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := e.UGlobalLongShortRatio(t.Context(), currency.EMPTYPAIR, "5m", 3, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UGlobalLongShortRatio(t.Context(), usdtmTradablePair, "", 3, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	startTime, endTime := getTime()
	_, err = e.UGlobalLongShortRatio(t.Context(), usdtmTradablePair, "5m", 3, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.UGlobalLongShortRatio(t.Context(), usdtmTradablePair, "5m", 3, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UGlobalLongShortRatio(t.Context(), usdtmTradablePair, "4h", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUTakerBuySellVol(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.UTakerBuySellVol(t.Context(), currency.EMPTYPAIR, "", 10, startTime, endTime)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UTakerBuySellVol(t.Context(), usdtmTradablePair, "", 10, startTime, endTime)
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	_, err = e.UTakerBuySellVol(t.Context(), usdtmTradablePair, "", 10, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.UTakerBuySellVol(t.Context(), usdtmTradablePair, "5m", 10, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBasis(t *testing.T) {
	t.Parallel()
	_, err := e.GetBasis(t.Context(), currency.EMPTYPAIR, "CURRENT_QUARTER", "15m", time.Time{}, time.Time{}, 20)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetBasis(t.Context(), usdtmTradablePair, "", "15m", time.Time{}, time.Time{}, 20)
	require.ErrorIs(t, err, errContractTypeIsRequired)

	_, err = e.GetBasis(t.Context(), usdtmTradablePair, "CURRENT_QUARTER", "", time.Time{}, time.Time{}, 20)
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	startTime, endTime := getTime()
	result, err := e.GetBasis(t.Context(), usdtmTradablePair, "CURRENT_QUARTER", "15m", startTime, endTime, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
	result, err = e.GetBasis(t.Context(), usdtmTradablePair, "NEXT_QUARTER", "15m", startTime, endTime, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
	result, err = e.GetBasis(t.Context(), usdtmTradablePair, "PERPETUAL", "15m", startTime, endTime, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalBLVTNAVCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalBLVTNAVCandlesticks(t.Context(), currency.EMPTYPAIR, "15m", time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetHistoricalBLVTNAVCandlesticks(t.Context(), spotTradablePair, "", time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	startTime, endTime := getTime()
	_, err = e.GetHistoricalBLVTNAVCandlesticks(t.Context(), spotTradablePair, "15m", endTime, startTime, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetHistoricalBLVTNAVCandlesticks(t.Context(), spotTradablePair, "15m", startTime, endTime, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCompositeIndexInfo(t *testing.T) {
	t.Parallel()
	result, err := e.UCompositeIndexInfo(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UCompositeIndexInfo(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMultiAssetModeAssetIndex(t *testing.T) {
	t.Parallel()
	result, err := e.GetMultiAssetModeAssetIndex(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetMultiAssetModeAssetIndex(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceConstituents(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceConstituents(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetIndexPriceConstituents(t.Context(), "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UFuturesNewOrder(t.Context(), &UFuturesNewOrderRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &UFuturesNewOrderRequest{
		ReduceOnly:   true,
		PositionSide: "position-side",
	}
	_, err = e.UFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPositionSide)

	arg.PositionSide = "LONG"
	arg.WorkingType = "abc"
	_, err = e.UFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidWorkingType)

	arg.WorkingType = "MARK_PRICE"
	arg.NewOrderRespType = "abc"
	_, err = e.UFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidNewOrderResponseType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UFuturesNewOrder(t.Context(),
		&UFuturesNewOrderRequest{
			Symbol:      currency.NewBTCUSDT(),
			Side:        "BUY",
			OrderType:   order.Limit.String(),
			TimeInForce: "GTC",
			Quantity:    1,
			Price:       1,
		},
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UModifyOrder(t.Context(), &USDTOrderUpdateParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &USDTOrderUpdateParams{PriceMatch: "1234"}
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = 1234
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.NewPair(currency.BTC, currency.USD)
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UModifyOrder(t.Context(), &USDTOrderUpdateParams{
		OrderID:           1,
		OrigClientOrderID: "",
		Side:              order.Sell.String(),
		PriceMatch:        "TAKE_PROFIT",
		Symbol:            currency.NewPair(currency.BTC, currency.USD),
		Amount:            0.0000001,
		Price:             123455554,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPlaceBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UPlaceBatchOrders(t.Context(), []PlaceBatchOrderData{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := PlaceBatchOrderData{
		TimeInForce:  "GTC",
		PositionSide: "abc",
	}
	_, err = e.UPlaceBatchOrders(t.Context(), []PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidPositionSide)

	arg.PositionSide = "SHORT"
	arg.WorkingType = "abc"
	_, err = e.UPlaceBatchOrders(t.Context(), []PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidWorkingType)

	arg.WorkingType = "CONTRACT_TYPE"
	arg.NewOrderRespType = "abc"
	_, err = e.UPlaceBatchOrders(t.Context(), []PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidNewOrderResponseType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	tempData := PlaceBatchOrderData{
		Symbol:      currency.Pair{Base: currency.BTC, Quote: currency.USDT},
		Side:        "BUY",
		OrderType:   order.Limit.String(),
		Quantity:    4,
		Price:       1,
		TimeInForce: "GTC",
	}
	result, err := e.UPlaceBatchOrders(t.Context(), []PlaceBatchOrderData{tempData})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := USDTOrderUpdateParams{}
	_, err = e.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = 1
	_, err = e.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = spotTradablePair
	_, err = e.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 0.0001
	_, err = e.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{arg})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{
		{
			OrderID:           1,
			OrigClientOrderID: "",
			Side:              order.Sell.String(),
			PriceMatch:        "TAKE_PROFIT",
			Symbol:            spotTradablePair,
			Amount:            0.0000001,
			Price:             123455554,
		},
		{
			OrderID:           1,
			OrigClientOrderID: "",
			Side:              "BUY",
			PriceMatch:        order.Limit.String(),
			Symbol:            spotTradablePair,
			Amount:            0.0000001,
			Price:             123455554,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUSDTOrderModifyHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTOrderModifyHistory(t.Context(), currency.EMPTYPAIR, "", 1234, 10, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetUSDTOrderModifyHistory(t.Context(), usdtmTradablePair, "", 0, 10, time.Time{}, time.Time{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	startTime, endTime := getTime()
	_, err = e.GetUSDTOrderModifyHistory(t.Context(), usdtmTradablePair, "", 0, 10, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUSDTOrderModifyHistory(t.Context(), usdtmTradablePair, "", 1234, 10, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetOrderData(t *testing.T) {
	t.Parallel()
	_, err := e.UGetOrderData(t.Context(), currency.EMPTYPAIR, "123", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UGetOrderData(t.Context(), usdtmTradablePair, "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UCancelOrder(t.Context(), currency.EMPTYPAIR, "123", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UCancelOrder(t.Context(), usdtmTradablePair, "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UCancelAllOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UCancelAllOpenOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UCancelBatchOrders(t.Context(), currency.EMPTYPAIR, []string{"123"}, []string{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UCancelBatchOrders(t.Context(), usdtmTradablePair, []string{"123"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UAutoCancelAllOpenOrders(t.Context(), currency.EMPTYPAIR, 30)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UAutoCancelAllOpenOrders(t.Context(), usdtmTradablePair, 30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFetchOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UFetchOpenOrder(t.Context(), usdtmTradablePair, "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAllAccountOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UAllAccountOpenOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAllAccountOrders(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.UAllAccountOrders(t.Context(), currency.EMPTYPAIR, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.UAllAccountOrders(t.Context(), currency.EMPTYPAIR, 0, 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err = e.UAllAccountOrders(t.Context(), usdtmTradablePair, 0, 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountBalanceV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UAccountBalanceV2(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountInformationV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UAccountInformationV2(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUChangeInitialLeverageRequest(t *testing.T) {
	t.Parallel()
	_, err := e.UChangeInitialLeverageRequest(t.Context(), usdtmTradablePair, 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UChangeInitialLeverageRequest(t.Context(), usdtmTradablePair, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUChangeInitialMarginType(t *testing.T) {
	t.Parallel()
	err := e.UChangeInitialMarginType(t.Context(), usdtmTradablePair, "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.UChangeInitialMarginType(t.Context(), usdtmTradablePair, "ISOLATED")
	assert.NoError(t, err)
}

func TestUModifyIsolatedPositionMarginReq(t *testing.T) {
	t.Parallel()
	_, err := e.UModifyIsolatedPositionMarginReq(t.Context(), usdtmTradablePair, "LONG", "", 5)
	require.ErrorIs(t, err, errMarginChangeTypeInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UModifyIsolatedPositionMarginReq(t.Context(), usdtmTradablePair, "LONG", "add", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPositionMarginChangeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.UPositionMarginChangeHistory(t.Context(), usdtmTradablePair, "", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errMarginChangeTypeInvalid)

	startTime, endTime := getTime()
	_, err = e.UPositionMarginChangeHistory(t.Context(), usdtmTradablePair, "", 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UPositionMarginChangeHistory(t.Context(), usdtmTradablePair, "add", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPositionsInfoV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UPositionsInfoV2(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetCommissionRates(t *testing.T) {
	t.Parallel()
	_, err := e.UGetCommissionRates(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UGetCommissionRates(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUSDTUserRateLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUSDTUserRateLimits(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDownloadIDForFuturesTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetDownloadIDForFuturesTransactionHistory(t.Context(), endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDownloadIDForFuturesTransactionHistory(t.Context(), startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTransactionHistoryDownloadLinkByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesTransactionHistoryDownloadLinkByID(t.Context(), "download-id-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderHistoryDownloadLinkByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderHistoryDownloadLinkByID(t.Context(), "")
	require.ErrorIs(t, err, errDownloadIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesOrderHistoryDownloadLinkByID(t.Context(), "download-id-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTradeDownloadLinkByID(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesTradeDownloadLinkByID(t.Context(), "")
	require.ErrorIs(t, err, errDownloadIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesTradeDownloadLinkByID(t.Context(), "download-id-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesOrderHistoryDownloadID(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.UFuturesOrderHistoryDownloadID(t.Context(), endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UFuturesOrderHistoryDownloadID(t.Context(), startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTradeHistoryDownloadID(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.FuturesTradeHistoryDownloadID(t.Context(), endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesTradeHistoryDownloadID(t.Context(), startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountTradesHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.UAccountTradesHistory(t.Context(), currency.EMPTYPAIR, "", 5, startTime, endTime)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UAccountTradesHistory(t.Context(), usdtmTradablePair, "", 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UAccountTradesHistory(t.Context(), usdtmTradablePair, "", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountIncomeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.UAccountIncomeHistory(t.Context(), currency.EMPTYPAIR, "something-else", 5, startTime, endTime)
	require.ErrorIs(t, err, errIncomeTypeRequired)
	_, err = e.UAccountIncomeHistory(t.Context(), usdtmTradablePair, "something-else", 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UAccountIncomeHistory(t.Context(), usdtmTradablePair, "", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UGetNotionalAndLeverageBrackets(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UPositionsADLEstimate(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountForcedOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UAccountForcedOrders(t.Context(), usdtmTradablePair, "something-else", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidAutoCloseType)

	startTime, endTime := getTime()
	_, err = e.UAccountForcedOrders(t.Context(), usdtmTradablePair, "ADL", 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UAccountForcedOrders(t.Context(), usdtmTradablePair, "ADL", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesTradingWuantitativeRulesIndicators(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UFuturesTradingWuantitativeRulesIndicators(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := e.FuturesExchangeInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesOrderbook(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPublicTrades(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesPublicTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPastPublicTrades(t *testing.T) {
	t.Parallel()
	result, err := e.GetPastPublicTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTradesList(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFuturesAggregatedTradesList(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 0, 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetFuturesAggregatedTradesList(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 0, 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPerpsExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetPerpMarkets(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexAndMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetIndexAndMarkPrice(t.Context(), "", "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesKlineData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1Mo", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	startTime, endTime := getTime()
	_, err = e.GetFuturesKlineData(t.Context(), currency.NewPairWithDelimiter("LTCUSD", "PERP", "_"), "5m", 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetFuturesKlineData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetFuturesKlineData(t.Context(), currency.NewPairWithDelimiter("LTCUSD", "PERP", "_"), "5m", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContinuousKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetContinuousKlineData(t.Context(), "", "CURRENT_QUARTER", "1M", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetContinuousKlineData(t.Context(), "BTCUSD", "", "1M", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errContractTypeIsRequired)

	_, err = e.GetContinuousKlineData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	startTime, endTime := getTime()
	_, err = e.GetContinuousKlineData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "1M", 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetContinuousKlineData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "1M", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = e.GetContinuousKlineData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "1M", 5, startTime, endTime)
	assert.NoError(t, err)
}

func TestGetIndexPriceKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceKlines(t.Context(), "BTCUSD", "", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	startTime, endTime := getTime()
	_, err = e.GetIndexPriceKlines(t.Context(), "BTCUSD", "1M", 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetIndexPriceKlines(t.Context(), "BTCUSD", "1M", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesSwapTickerChangeStats(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesSwapTickerChangeStats(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
	result, err = e.GetFuturesSwapTickerChangeStats(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
	result, err = e.GetFuturesSwapTickerChangeStats(t.Context(), currency.EMPTYPAIR, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesGetFundingHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.FuturesGetFundingHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesGetFundingHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.FuturesGetFundingHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 50, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesHistoricalTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetFuturesHistoricalTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesSymbolPriceTicker(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderbookTicker(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesOrderbookTicker(t.Context(), currency.EMPTYPAIR, "")
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetFuturesOrderbookTicker(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCFuturesIndexPriceConstituents(t *testing.T) {
	t.Parallel()
	_, err := e.GetCFuturesIndexPriceConstituents(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetCFuturesIndexPriceConstituents(t.Context(), currency.NewPair(currency.BTC, currency.USD))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOpenInterest(t *testing.T) {
	t.Parallel()
	result, err := e.OpenInterest(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCFuturesQuarterlyContractSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := e.CFuturesQuarterlyContractSettlementPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.CFuturesQuarterlyContractSettlementPrice(t.Context(), currency.NewPair(currency.BTC, currency.USD))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterestStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterestStats(t.Context(), "BTCUSD", "QUARTER", "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errContractTypeIsRequired)

	_, err = e.GetOpenInterestStats(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5mo", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	startTime, endTime := getTime()
	_, err = e.GetOpenInterestStats(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetOpenInterestStats(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOpenInterestStats(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTraderFuturesAccountRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetTraderFuturesAccountRatio(t.Context(), currency.EMPTYPAIR, "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetTraderFuturesAccountRatio(t.Context(), usdtmTradablePair, "", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	startTime, endTime := getTime()
	_, err = e.GetTraderFuturesAccountRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetTraderFuturesAccountRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetTraderFuturesAccountRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTraderFuturesPositionsRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetTraderFuturesPositionsRatio(t.Context(), currency.EMPTYPAIR, "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetTraderFuturesPositionsRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5mo", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	startTime, endTime := getTime()
	_, err = e.GetTraderFuturesPositionsRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetTraderFuturesPositionsRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetTraderFuturesPositionsRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketRatio(t.Context(), currency.EMPTYPAIR, "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetMarketRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5mo", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	startTime, endTime := getTime()
	_, err = e.GetMarketRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetMarketRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetMarketRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTakerVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTakerVolume(t.Context(), currency.EMPTYPAIR, "ALL", "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetFuturesTakerVolume(t.Context(), currency.NewPair(currency.BTC, currency.USD), "abc", "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errContractTypeIsRequired)

	_, err = e.GetFuturesTakerVolume(t.Context(), currency.NewPair(currency.BTC, currency.USD), "ALL", "5mo", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	startTime, endTime := getTime()
	_, err = e.GetFuturesTakerVolume(t.Context(), currency.NewPair(currency.BTC, currency.USD), "ALL", "5m", 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetFuturesTakerVolume(t.Context(), currency.NewPair(currency.BTC, currency.USD), "ALL", "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetFuturesTakerVolume(t.Context(), currency.NewPair(currency.BTC, currency.USD), "ALL", "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesBasisData(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesBasisData(t.Context(), currency.EMPTYPAIR, "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetFuturesBasisData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "QUARTER", "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errContractTypeIsRequired)
	_, err = e.GetFuturesBasisData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "CURRENT_QUARTER", "5mo", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := e.GetFuturesBasisData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	startTime := time.UnixMilli(1577836800000)
	endTime := time.UnixMilli(1580515200000)
	if !mockTests {
		startTime = time.Now().Add(-time.Second * 240)
		endTime = time.Now()
	}
	_, err = e.GetFuturesBasisData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "CURRENT_QUARTER", "5m", 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err = e.GetFuturesBasisData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "CURRENT_QUARTER", "5m", 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesNewOrder(t *testing.T) {
	t.Parallel()
	arg := &FuturesNewOrderRequest{Symbol: usdtmTradablePair, Side: "BUY", OrderType: order.Limit.String(), PositionSide: "abcd", TimeInForce: order.GoodTillCancel.String(), Quantity: 1, Price: 1}
	_, err := e.FuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPositionSide)

	arg.PositionSide = ""
	arg.WorkingType = "abc"
	_, err = e.FuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidWorkingType)

	arg.WorkingType = ""
	arg.NewOrderRespType = "abcd"
	_, err = e.FuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidNewOrderResponseType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FuturesNewOrder(t.Context(), &FuturesNewOrderRequest{Symbol: currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), Side: "BUY", OrderType: order.Limit.String(), TimeInForce: order.GoodTillCancel.String(), Quantity: 1, Price: 1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesBatchOrder(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &PlaceBatchOrderData{
		Symbol:       currency.Pair{Base: currency.BTC, Quote: currency.NewCode("USD_PERP")},
		Side:         "BUY",
		OrderType:    order.Limit.String(),
		Quantity:     1,
		Price:        1,
		TimeInForce:  "GTC",
		PositionSide: "abcd",
	}
	_, err = e.FuturesBatchOrder(t.Context(), []*PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidPositionSide)

	arg.PositionSide = ""
	arg.WorkingType = "abcd"
	_, err = e.FuturesBatchOrder(t.Context(), []*PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidWorkingType)

	arg.NewOrderRespType = "abcd"
	arg.WorkingType = ""
	_, err = e.FuturesBatchOrder(t.Context(), []*PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidNewOrderResponseType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FuturesBatchOrder(t.Context(), []*PlaceBatchOrderData{
		{
			Symbol:      currency.Pair{Base: currency.BTC, Quote: currency.NewCode("USD_PERP")},
			Side:        "BUY",
			OrderType:   order.Limit.String(),
			Quantity:    1,
			Price:       1,
			TimeInForce: "GTC",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesBatchCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FuturesBatchCancelOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), []string{"123"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesGetOrderData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesGetOrderData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FuturesCancelAllOpenOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AutoCancelAllOpenOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 30000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesOpenOrderData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesOpenOrderData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesAllOpenOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllFuturesOrders(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAllFuturesOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), currency.EMPTYPAIR, endTime, startTime, 1, 2)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllFuturesOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), currency.EMPTYPAIR, startTime, endTime, 0, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesChangeMarginType(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesChangeMarginType(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "abcd")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FuturesChangeMarginType(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "ISOLATED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesAccountBalance(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesAccountInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesChangeInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesChangeInitialLeverage(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 129)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FuturesChangeInitialLeverage(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyIsolatedPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyIsolatedPositionMargin(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", "abcd", 0)
	require.ErrorIs(t, err, errMarginChangeTypeInvalid)

	_, err = e.ModifyIsolatedPositionMargin(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "abcd", "", 0)
	require.ErrorIs(t, err, errInvalidPositionSide)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ModifyIsolatedPositionMargin(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "BOTH", "add", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesMarginChangeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesMarginChangeHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "abc", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMarginChangeTypeInvalid)

	startTime, endTime := getTime()
	_, err = e.FuturesMarginChangeHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "add", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesMarginChangeHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "add", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesPositionsInfo(t.Context(), "BTCUSD", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.FuturesTradeHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", endTime, startTime, 5, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesTradeHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", startTime, endTime, 5, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesIncomeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.FuturesIncomeHistory(t.Context(), currency.EMPTYPAIR, "TRANSFER", endTime, startTime, 5)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesIncomeHistory(t.Context(), currency.EMPTYPAIR, "TRANSFER", startTime, endTime, 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesForceOrders(t.Context(), currency.EMPTYPAIR, "abcd", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidAutoCloseType)

	startTime, endTime := getTime()
	_, err = e.FuturesForceOrders(t.Context(), currency.EMPTYPAIR, "abcd", endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesForceOrders(t.Context(), currency.EMPTYPAIR, "ADL", startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetNotionalLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesNotionalBracket(t.Context(), "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.FuturesNotionalBracket(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FuturesPositionsADLEstimate(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceKline(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1Mo", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	startTime, endTime := getTime()
	_, err = e.GetMarkPriceKline(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetMarkPriceKline(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPremiumIndexKlineData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetPremiumIndexKlineData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetExchangeInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
	if !mockTests {
		assert.WithinRange(t, result.ServerTime.Time(), time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), "ServerTime should be within a day of now")
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	assetTypes := e.GetAssetTypes(true)
	for a := range assetTypes {
		results, err := e.FetchTradablePairs(t.Context(), assetTypes[a])
		assert.NoError(t, err)
		assert.NotNil(t, results)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	result, err := e.GetOrderBook(t.Context(),
		OrderBookDataRequestParams{
			Symbol: currency.NewBTCUSDT(),
			Limit:  1000,
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMostRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetMostRecentTrades(t.Context(), &RecentTradeRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	result, err := e.GetMostRecentTrades(t.Context(), &RecentTradeRequestParams{Symbol: usdtmTradablePair, Limit: 15})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalTrades(t.Context(), currency.EMPTYPAIR, 5, -1)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetHistoricalTrades(t.Context(), usdtmTradablePair, 5, -1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequestParams{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequestParams{Symbol: usdtmTradablePair, Limit: 5})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSpotKline(t.Context(), &KlinesRequestParams{Symbol: usdtmTradablePair, Interval: kline.FiveMin.Short(), Limit: 24, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetSpotKline(t.Context(), &KlinesRequestParams{Symbol: usdtmTradablePair, Interval: kline.FiveMin.Short(), Limit: 24, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUIKline(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUIKline(t.Context(), &KlinesRequestParams{Symbol: usdtmTradablePair, Interval: kline.FiveMin.Short(), Limit: 24, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetUIKline(t.Context(), &KlinesRequestParams{Symbol: usdtmTradablePair, Interval: kline.FiveMin.Short(), Limit: 24, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetAveragePrice(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()
	result, err := e.GetPriceChangeStats(t.Context(), usdtmTradablePair, currency.Pairs{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradingDayTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradingDayTicker(t.Context(), []currency.Pair{}, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)
	_, err = e.GetTradingDayTicker(t.Context(), []currency.Pair{currency.EMPTYPAIR}, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetTradingDayTicker(t.Context(), []currency.Pair{spotTradablePair}, "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetLatestSpotPrice(t.Context(), usdtmTradablePair, currency.Pairs{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetBestPrice(t.Context(), spotTradablePair, currency.Pairs{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickerData(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickerData(t.Context(), []currency.Pair{}, time.Minute*20, "FULL")
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	result, err := e.GetTickerData(t.Context(), []currency.Pair{spotTradablePair}, time.Minute*20, "FULL")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddressForCurrency(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositAddressForCurrency(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetsThatCanBeConvertedIntoBNB(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAssetsThatCanBeConvertedIntoBNB(t.Context(), "MINI")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCrypto(t.Context(), currency.EMPTYCODE, "123435", "", "address-here", "", "", 100, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WithdrawCrypto(t.Context(), currency.USDT, "", "", "", "", "", 100, false)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = e.WithdrawCrypto(t.Context(), currency.USDT, "123435", "", "address-here", "123213", "", 0, false)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawCrypto(t.Context(), currency.USDT, "123435", "", "address", "", "", 100, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDustTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.DustTransfer(t.Context(), []string{}, "SPOT")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.DustTransfer(t.Context(), []string{"BTC", "USDT"}, "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetDevidendRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetAssetDevidendRecords(t.Context(), currency.EMPTYCODE, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	startTime, endTime := getTime()
	_, err = e.GetAssetDevidendRecords(t.Context(), currency.BTC, endTime, startTime, 1)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAssetDevidendRecords(t.Context(), currency.BTC, startTime, endTime, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAssetDetail(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeFees(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTradeFees(t.Context(), currency.Pair{Base: currency.BTC, Quote: currency.USDT})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUserUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.UserUniversalTransfer(t.Context(), 0, 123.234, currency.BTC, currency.EMPTYPAIR, currency.EMPTYPAIR)
	require.ErrorIs(t, err, errTransferTypeRequired)
	_, err = e.UserUniversalTransfer(t.Context(), ttMainUMFuture, 123.234, currency.EMPTYCODE, currency.EMPTYPAIR, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.UserUniversalTransfer(t.Context(), ttMainUMFuture, 0, currency.BTC, currency.EMPTYPAIR, currency.EMPTYPAIR)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UserUniversalTransfer(t.Context(), ttMainUMFuture, 123.234, currency.BTC, currency.EMPTYPAIR, currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserUniversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserUniversalTransferHistory(t.Context(), 0, time.Time{}, time.Time{}, 0, 0, currency.BTC, currency.USDT)
	require.ErrorIs(t, err, errTransferTypeRequired)
	_, err = e.GetUserUniversalTransferHistory(t.Context(), ttUMFutureMargin, time.Time{}, time.Time{}, 0, 0, currency.BTC, currency.USDT)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	startTime, endTime := getTime()
	_, err = e.GetUserUniversalTransferHistory(t.Context(), ttUMFutureMargin, endTime, startTime, 0, 0, currency.BTC, currency.USDT)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserUniversalTransferHistory(t.Context(), ttUMFutureMargin, startTime, endTime, 1, 1234, currency.BTC, currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundingAssets(t.Context(), currency.BTC, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAssets(t.Context(), currency.BTC, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConvertBUSD(t *testing.T) {
	t.Parallel()
	_, err := e.ConvertBUSD(t.Context(), "", "MAIN", currency.ETH, currency.USD, 1234)
	require.ErrorIs(t, err, errTransactionIDRequired)
	_, err = e.ConvertBUSD(t.Context(), "12321412312", "MAIN", currency.EMPTYCODE, currency.USD, 1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.ConvertBUSD(t.Context(), "12321412312", "MAIN", currency.ETH, currency.USD, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.ConvertBUSD(t.Context(), "12321412312", "MAIN", currency.ETH, currency.EMPTYCODE, 1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ConvertBUSD(t.Context(), "12321412312", "MAIN", currency.ETH, currency.USD, 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBUSDConvertHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.BUSDConvertHistory(t.Context(), "transaction-id", "233423423", "CARD", currency.BTC, endTime, startTime, 0, 20)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.BUSDConvertHistory(t.Context(), "transaction-id", "233423423", "CARD", currency.BTC, startTime, endTime, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCloudMiningPaymentAndRefundHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetCloudMiningPaymentAndRefundHistory(t.Context(), "1234", currency.BTC, endTime, startTime, 1232313, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCloudMiningPaymentAndRefundHistory(t.Context(), "1234", currency.BTC, startTime, endTime, 1232313, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAccountInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAPIKeyPermission(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAPIKeyPermission(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoConvertingStableCoins(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAutoConvertingStableCoins(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSwitchOnOffBUSDAndStableCoinsConversion(t *testing.T) {
	t.Parallel()
	err := e.SwitchOnOffBUSDAndStableCoinsConversion(t.Context(), currency.EMPTYCODE, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SwitchOnOffBUSDAndStableCoinsConversion(t.Context(), currency.BTC, false)
	assert.NoError(t, err)
}

func TestOneClickArrivalDepositApply(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.OneClickArrivalDepositApply(t.Context(), "", 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressListWithNetwork(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddressListWithNetwork(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositAddressListWithNetwork(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserWalletBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserWalletBalance(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserDelegationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserDelegationHistory(t.Context(), "", "Delegate", time.Time{}, time.Time{}, currency.BTC, 0, 0)
	require.ErrorIs(t, err, errValidEmailRequired)

	startTime, endTime := getTime()
	_, err = e.GetUserDelegationHistory(t.Context(), "someone@thrasher.com", "Delegate", startTime, endTime, currency.BTC, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserDelegationHistory(t.Context(), "someone@thrasher.com", "Delegate", startTime, endTime, currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolsDelistScheduleForSpot(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSymbolsDelistScheduleForSpot(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateVirtualSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateVirtualSubAccount(t.Context(), "something-string")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountList(t.Context(), "testsub@gmail.com", false, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountSpotAssetTransferHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSubAccountSpotAssetTransferHistory(t.Context(), "", "", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountSpotAssetTransferHistory(t.Context(), "", "", time.Time{}, time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountFuturesAssetTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountFuturesAssetTransferHistory(t.Context(), "", time.Time{}, time.Now(), 2, 0, 0)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.GetSubAccountFuturesAssetTransferHistory(t.Context(), "someone@gmail.com", time.Time{}, time.Now(), 0, 0, 0)
	require.ErrorIs(t, err, errInvalidFuturesType)

	startTime, endTime := getTime()
	_, err = e.GetSubAccountFuturesAssetTransferHistory(t.Context(), "someone@gmail.com", endTime, startTime, 0, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountFuturesAssetTransferHistory(t.Context(), "someone@gmail.com", startTime, endTime, 2, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountFuturesAssetTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountFuturesAssetTransfer(t.Context(), "from_someone", "to_someont@thrasher.io", 1, currency.USDT, 0.1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.SubAccountFuturesAssetTransfer(t.Context(), "from_someone@thrasher.io", "to_someont", 1, currency.USDT, 0.1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.SubAccountFuturesAssetTransfer(t.Context(), "from_someone@thrasher.io", "to_someont@thrasher.io", -1, currency.USDT, 0.1)
	require.ErrorIs(t, err, errInvalidFuturesType)
	_, err = e.SubAccountFuturesAssetTransfer(t.Context(), "from_someone@thrasher.io", "to_someont@thrasher.io", 1, currency.EMPTYCODE, 0.1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubAccountFuturesAssetTransfer(t.Context(), "from_someone@thrasher.io", "to_someont@thrasher.io", 1, currency.USDT, 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAssets(t.Context(), "email_address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountAssets(t.Context(), "email_address@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetManagedSubAccountList(t.Context(), "address@gmail.com", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountTransactionStatistics(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountTransactionStatistics(t.Context(), "addressio")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountTransactionStatistics(t.Context(), "address@thrasher.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountDepositAddress(t.Context(), currency.ETH, "destination", "")
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.GetManagedSubAccountDepositAddress(t.Context(), currency.EMPTYCODE, "destination@thrasher.io", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetManagedSubAccountDepositAddress(t.Context(), currency.ETH, "destination@thrasher.io", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableOptionsForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.EnableOptionsForSubAccount(t.Context(), "")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableOptionsForSubAccount(t.Context(), "address@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountTransferLog(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetManagedSubAccountTransferLog(t.Context(), endTime, startTime, 1, 10, "", "MARGIN")
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetManagedSubAccountTransferLog(t.Context(), startTime, endTime, -1, 10, "", "MARGIN")
	require.ErrorIs(t, err, errPageNumberRequired)
	_, err = e.GetManagedSubAccountTransferLog(t.Context(), startTime, endTime, 1, -1, "", "MARGIN")
	require.ErrorIs(t, err, errLimitNumberRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetManagedSubAccountTransferLog(t.Context(), startTime, endTime, 1, 10, "", "MARGIN")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountSpotAssetsSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountSpotAssetsSummary(t.Context(), "the_address@thrasher.io", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountDepositAddress(t.Context(), "", "BTC", "", 0.1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.GetSubAccountDepositAddress(t.Context(), "the_address@thrasher.io", "", "", 0.1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountDepositAddress(t.Context(), "the_address@thrasher.io", "BTC", "", 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountDepositHistory(t.Context(), "someoneio", "BTC", time.Time{}, time.Now(), 0, 0, 10)
	require.ErrorIs(t, err, errValidEmailRequired)

	startTime, endTime := getTime()
	_, err = e.GetSubAccountDepositHistory(t.Context(), "someoneio", "BTC", endTime, startTime, 0, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountDepositHistory(t.Context(), "someone@thrasher.io", "BTC", startTime, endTime, 1, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountStatusOnMarginFutures(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountStatusOnMarginFutures(t.Context(), "myemail@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableMarginForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.EnableMarginForSubAccount(t.Context(), "sampleemaicom")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableMarginForSubAccount(t.Context(), "sampleemail@email.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailOnSubAccountMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetDetailOnSubAccountMarginAccount(t.Context(), "com")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDetailOnSubAccountMarginAccount(t.Context(), "test@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfSubAccountMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSummaryOfSubAccountMarginAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableFuturesSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.EnableFuturesSubAccount(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableFuturesSubAccount(t.Context(), "address@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailSubAccountFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetDetailSubAccountFuturesAccount(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDetailSubAccountFuturesAccount(t.Context(), "address@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfSubAccountFuturesAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetSummaryOfSubAccountFuturesAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV1FuturesPositionRiskSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetV1FuturesPositionRiskSubAccount(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetV1FuturesPositionRiskSubAccount(t.Context(), "address@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositionRiskSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetV2FuturesPositionRiskSubAccount(t.Context(), "address", 1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.GetV2FuturesPositionRiskSubAccount(t.Context(), "address@mail.com", -1)
	require.ErrorIs(t, err, errInvalidFuturesType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetV2FuturesPositionRiskSubAccount(t.Context(), "address@mail.com", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableLeverageTokenForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.EnableLeverageTokenForSubAccount(t.Context(), "email-address", false)
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableLeverageTokenForSubAccount(t.Context(), "someone@thrasher.io", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.GetIPRestrictionForSubAccountAPIKeyV2(t.Context(), "emailaddress", apiKey)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.GetIPRestrictionForSubAccountAPIKeyV2(t.Context(), "emailaddress@thrasher.io", "")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIPRestrictionForSubAccountAPIKeyV2(t.Context(), "emailaddress@thrasher.io", apiKey)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteIPListForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteIPListForSubAccountAPIKey(t.Context(), "emailaddress", apiKey, "196.168.4.1")
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.DeleteIPListForSubAccountAPIKey(t.Context(), "emailaddress@thrasher.io", "", "196.168.4.1")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.DeleteIPListForSubAccountAPIKey(t.Context(), "emailaddress@thrasher.io", apiKey, "196.168.4.1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAddIPRestrictionForSubAccountAPIkey(t *testing.T) {
	t.Parallel()
	_, err := e.AddIPRestrictionForSubAccountAPIkey(t.Context(), "addressthrasher", apiKey, "", true)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.AddIPRestrictionForSubAccountAPIkey(t.Context(), "address@thrasher.io", "", "", true)
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AddIPRestrictionForSubAccountAPIkey(t.Context(), "address@thrasher.io", apiKey, "", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDepositAssetsIntoTheManagedSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail", currency.BTC, 0.0001)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail@mail.com", currency.EMPTYCODE, 0.0001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail@mail.com", currency.BTC, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail@mail.com", currency.BTC, 0.0001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountAssetsDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountAssetsDetails(t.Context(), "emailaddress")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetManagedSubAccountAssetsDetails(t.Context(), "emailaddress@thrashser.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawAssetsFromManagedSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawAssetsFromManagedSubAccount(t.Context(), "source", currency.BTC, 0.0000001, time.Now().Add(-time.Hour*24*50))
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.WithdrawAssetsFromManagedSubAccount(t.Context(), "source@email.com", currency.EMPTYCODE, 0.0000001, time.Now().Add(-time.Hour*24*50))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WithdrawAssetsFromManagedSubAccount(t.Context(), "source@email.com", currency.BTC, 0, time.Now().Add(-time.Hour*24*50))
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawAssetsFromManagedSubAccount(t.Context(), "source@email.com", currency.BTC, 0.0000001, time.Now().Add(-time.Hour*24*50))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountSnapshot(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountSnapshot(t.Context(), "address", "SPOT", time.Time{}, time.Now(), 10)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.GetManagedSubAccountSnapshot(t.Context(), "address@thrasher.io", "", time.Time{}, time.Now(), 10)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	startTime, endTime := getTime()
	_, err = e.GetManagedSubAccountSnapshot(t.Context(), "address@thrasher.io", "", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetManagedSubAccountSnapshot(t.Context(), "address@thrasher.io", "SPOT", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountTransferLogForInvestorMasterAccount(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetManagedSubAccountTransferLogForInvestorMasterAccount(t.Context(), "address.com", "TO", "SPOT", startTime, endTime, 1, 10)
	require.ErrorIs(t, err, errValidEmailRequired)

	_, err = e.GetManagedSubAccountTransferLogForInvestorMasterAccount(t.Context(), "address@gmail.com", "TO", "SPOT", startTime, endTime, -1, 10)
	require.ErrorIs(t, err, errPageNumberRequired)

	_, err = e.GetManagedSubAccountTransferLogForInvestorMasterAccount(t.Context(), "address@gmail.com", "TO", "SPOT", startTime, endTime, 1, 0)
	require.ErrorIs(t, err, errLimitNumberRequired)

	_, err = e.GetManagedSubAccountTransferLogForInvestorMasterAccount(t.Context(), "address@gmail.com", "TO", "SPOT", endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetManagedSubAccountTransferLogForInvestorMasterAccount(t.Context(), "address@gmail.com", "TO", "SPOT", startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountTransferLogForTradingTeam(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountTransferLogForTradingTeam(t.Context(), "address", "FROM", "ISOLATED_MARGIN", time.Time{}, time.Time{}, 1, 10)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.GetManagedSubAccountTransferLogForTradingTeam(t.Context(), "address@gmail.com", "FROM", "ISOLATED_MARGIN", time.Time{}, time.Time{}, -1, 10)
	require.ErrorIs(t, err, errPageNumberRequired)
	_, err = e.GetManagedSubAccountTransferLogForTradingTeam(t.Context(), "address@gmail.com", "FROM", "ISOLATED_MARGIN", time.Time{}, time.Time{}, 1, 0)
	require.ErrorIs(t, err, errLimitNumberRequired)

	startTime, endTime := getTime()
	_, err = e.GetManagedSubAccountTransferLogForTradingTeam(t.Context(), "address@gmail.com", "FROM", "ISOLATED_MARGIN", endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetManagedSubAccountTransferLogForTradingTeam(t.Context(), "address@gmail.com", "FROM", "ISOLATED_MARGIN", startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountFutureesAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountFutureesAssetDetails(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetManagedSubAccountFutureesAssetDetails(t.Context(), "address@email.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountMarginAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountMarginAssetDetails(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetManagedSubAccountMarginAssetDetails(t.Context(), "address@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTransferSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesTransferSubAccount(t.Context(), "someone.com", currency.BTC, 1.1, 1)
	require.ErrorIs(t, err, errValidEmailRequired)

	_, err = e.FuturesTransferSubAccount(t.Context(), "someone@mail.com", currency.EMPTYCODE, 1.1, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.FuturesTransferSubAccount(t.Context(), "someone@mail.com", currency.BTC, 0, 1)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.FuturesTransferSubAccount(t.Context(), "someone@mail.com", currency.BTC, 1.1, 0)
	require.ErrorIs(t, err, errTransferTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FuturesTransferSubAccount(t.Context(), "someone@mail.com", currency.BTC, 1.1, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginTransferForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.MarginTransferForSubAccount(t.Context(), "someone", currency.BTC, 1.1, 1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.MarginTransferForSubAccount(t.Context(), "someone@mail.com", currency.EMPTYCODE, 1.1, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.MarginTransferForSubAccount(t.Context(), "someone@mail.com", currency.BTC, 0, 1)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.MarginTransferForSubAccount(t.Context(), "someone@mail.com", currency.BTC, 1.1, -1)
	require.ErrorIs(t, err, errTransferTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.MarginTransferForSubAccount(t.Context(), "someone@mail.com", currency.BTC, 1.1, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAssetsV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAssetsV3(t.Context(), "")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountAssetsV3(t.Context(), "someone@thrasher.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferToSubAccountOfSameMaster(t *testing.T) {
	t.Parallel()
	_, err := e.TransferToSubAccountOfSameMaster(t.Context(), "thrasher", currency.ETH, 10)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.TransferToSubAccountOfSameMaster(t.Context(), "toEmail@thrasher.io", currency.EMPTYCODE, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.TransferToSubAccountOfSameMaster(t.Context(), "toEmail@thrasher.io", currency.ETH, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.TransferToSubAccountOfSameMaster(t.Context(), "toEmail@thrasher.io", currency.ETH, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFromSubAccountTransferToMaster(t *testing.T) {
	t.Parallel()
	_, err := e.FromSubAccountTransferToMaster(t.Context(), currency.EMPTYCODE, 0.1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.FromSubAccountTransferToMaster(t.Context(), currency.LTC, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FromSubAccountTransferToMaster(t.Context(), currency.LTC, 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.SubAccountTransferHistory(t.Context(), currency.BTC, 1, 10, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubAccountTransferHistory(t.Context(), currency.BTC, 1, 10, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransferHistoryForSubAccount(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.SubAccountTransferHistoryForSubAccount(t.Context(), currency.LTC, 2, 0, endTime, startTime, true)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubAccountTransferHistoryForSubAccount(t.Context(), currency.LTC, 2, 0, startTime, endTime, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUniversalTransferForMasterAccount(t *testing.T) {
	t.Parallel()
	_, err := e.UniversalTransferForMasterAccount(t.Context(), &UniversalTransferParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &UniversalTransferParams{
		ClientTransactionID: "transaction-id",
	}
	_, err = e.UniversalTransferForMasterAccount(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAccountType)

	arg.ToAccountType = "SPOT"
	_, err = e.UniversalTransferForMasterAccount(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAccountType)

	arg.FromAccountType = "ISOLATED_MARGIN"
	_, err = e.UniversalTransferForMasterAccount(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Asset = currency.BTC
	_, err = e.UniversalTransferForMasterAccount(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UniversalTransferForMasterAccount(t.Context(), &UniversalTransferParams{
		FromEmail:           "source@thrasher.io",
		ToEmail:             "destination@thrasher.io",
		FromAccountType:     "ISOLATED_MARGIN",
		ToAccountType:       "SPOT",
		ClientTransactionID: "transaction-id",
		Symbol:              "",
		Asset:               currency.BTC,
		Amount:              0.0003,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUniversalTransferHistoryForMasterAccount(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUniversalTransferHistoryForMasterAccount(t.Context(), "", "", "", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUniversalTransferHistoryForMasterAccount(t.Context(), "", "", "", startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailOnSubAccountsFuturesAccountV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetDetailOnSubAccountsFuturesAccountV2(t.Context(), "thrasher", 1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = e.GetDetailOnSubAccountsFuturesAccountV2(t.Context(), "address@thrasher.io", 0)
	require.ErrorIs(t, err, errInvalidFuturesType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDetailOnSubAccountsFuturesAccountV2(t.Context(), "address@thrasher.io", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfSubAccountsFuturesAccountV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetSummaryOfSubAccountsFuturesAccountV2(t.Context(), 0, 0, 10)
	require.ErrorIs(t, err, errInvalidFuturesType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSummaryOfSubAccountsFuturesAccountV2(t.Context(), 1, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.QueryOrder(t.Context(), usdtmTradablePair, "", 1337)
	require.False(t, sharedtestvalues.AreAPICredentialsSet(e) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests, "expecting an error when no keys are set")
	assert.False(t, mockTests && err != nil, err)
}

func TestCancelExistingOrderAndSendNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelExistingOrderAndSendNewOrder(t.Context(), &CancelReplaceOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &CancelReplaceOrderParams{
		TimeInForce: "GTC",
	}
	_, err = e.CancelExistingOrderAndSendNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = usdtmTradablePair
	_, err = e.CancelExistingOrderAndSendNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = e.CancelExistingOrderAndSendNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = e.CancelExistingOrderAndSendNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errCancelReplaceModeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelExistingOrderAndSendNewOrder(t.Context(), &CancelReplaceOrderParams{
		Symbol:            usdtmTradablePair,
		Side:              "BUY",
		OrderType:         order.Limit.String(),
		CancelReplaceMode: "STOP_ON_FAILURE",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.OpenOrders(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
	p := usdtmTradablePair
	result, err = e.OpenOrders(t.Context(), p)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrderOnSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOpenOrderOnSymbol(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOpenOrderOnSymbol(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.AllOrders(t.Context(), usdtmTradablePair, "", "")
	require.False(t, sharedtestvalues.AreAPICredentialsSet(e) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests, "expecting an error when no keys are set")
	assert.False(t, mockTests && err != nil, err)
}

func TestNewOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewOCOOrder(t.Context(), &OCOOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &OCOOrderParam{
		TrailingDelta: 1,
	}
	_, err = e.NewOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Symbol = usdtmTradablePair
	_, err = e.NewOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Buy"
	_, err = e.NewOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 0.1
	_, err = e.NewOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.Price = 0.001
	_, err = e.NewOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewOCOOrder(t.Context(), &OCOOrderParam{
		Symbol:             usdtmTradablePair,
		ListClientOrderID:  "1231231231231",
		Side:               "Buy",
		Amount:             0.1,
		LimitClientOrderID: "3423423",
		Price:              0.001,
		StopPrice:          1234.21,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOCOOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOCOOrder(t.Context(), currency.EMPTYPAIR, "", "newderID", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelOCOOrder(t.Context(), spotTradablePair, "", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelOCOOrder(t.Context(), spotTradablePair, "", "newderID", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOCOOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOCOOrders(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOCOOrders(t.Context(), "123456", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOCOOrders(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAllOCOOrders(t.Context(), "", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllOCOOrders(t.Context(), "", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOCOList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOCOList(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOrderUsingSOR(t *testing.T) {
	t.Parallel()
	_, err := e.NewOrderUsingSOR(t.Context(), &SOROrderRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &SOROrderRequestParams{
		TimeInForce: "GTC",
	}
	_, err = e.NewOrderUsingSOR(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.Pair{Base: currency.BTC, Quote: currency.LTC}
	_, err = e.NewOrderUsingSOR(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.NewOrderUsingSOR(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = e.NewOrderUsingSOR(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewOrderUsingSOR(t.Context(), &SOROrderRequestParams{
		Symbol:    currency.Pair{Base: currency.BTC, Quote: currency.LTC},
		Side:      "Buy",
		OrderType: order.Limit.String(),
		Quantity:  0.001,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOrderUsingSORTest(t *testing.T) {
	t.Parallel()
	_, err := e.NewOrderUsingSORTest(t.Context(), &SOROrderRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewOrderUsingSORTest(t.Context(), &SOROrderRequestParams{
		Symbol:    currency.Pair{Base: currency.BTC, Quote: currency.LTC},
		Side:      "Buy",
		OrderType: order.Limit.String(),
		Quantity:  0.001,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	feeBuilder := setFeeBuilder()
	result, err := e.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err)
	assert.NotNil(t, result)

	if !sharedtestvalues.AreAPICredentialsSet(e) || mockTests {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	if sharedtestvalues.AreAPICredentialsSet(e) && mockTests {
		// CryptocurrencyTradeFee Basic
		_, err := e.GetFee(t.Context(), feeBuilder)
		require.NoError(t, err)

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		_, err = e.GetFee(t.Context(), feeBuilder)
		require.NoError(t, err)

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		_, err = e.GetFee(t.Context(), feeBuilder)
		require.NoError(t, err)

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		_, err = e.GetFee(t.Context(), feeBuilder)
		require.NoError(t, err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err := e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = e.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err)
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := e.FormatWithdrawPermissions()
	require.Equal(t, expectedResult, withdrawPermissions)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     currency.Pairs{pair},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	result, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	getOrdersRequest.Pairs = []currency.Pair{
		currency.NewPair(currency.LTC,
			currency.BTC),
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOrderTest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.NewOrderTest(t.Context(), &NewOrderRequest{
		Symbol:      currency.NewPair(currency.LTC, currency.BTC),
		Side:        order.Buy.String(),
		TradeType:   order.Limit.String(),
		Price:       0.0025,
		Quantity:    100000,
		TimeInForce: order.GoodTillCancel.String(),
	}, false)
	require.NoError(t, err)

	err = e.NewOrderTest(t.Context(), &NewOrderRequest{
		Symbol:        currency.NewPair(currency.LTC, currency.BTC),
		Side:          order.Sell.String(),
		TradeType:     order.Market.String(),
		Price:         0.0045,
		QuoteOrderQty: 10,
	}, true)
	assert.NoError(t, err)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	p := currency.NewBTCUSDT()
	startTime := time.Unix(1577977445, 0)      // 2020-01-02 15:04:05
	endTime := startTime.Add(15 * time.Minute) // 2020-01-02 15:19:05
	if e.IsAPIStreamConnected() {
		startTime = time.Now().Add(-time.Hour * 10)
		endTime = time.Now().Add(-time.Hour)
	}
	result, err := e.GetHistoricTrades(t.Context(), p, asset.Spot, startTime, endTime)
	require.NoError(t, err)
	expected := 2134
	if e.IsAPIStreamConnected() {
		expected = len(result)
	} else if mockTests {
		expected = 1002
	}
	require.Equal(t, expected, len(result), "GetHistoricTrades should return correct number of entries")
	for _, r := range result {
		require.WithinRange(t, r.Timestamp, startTime, endTime, "All trades must be within time range")
	}
	result, err = e.GetHistoricTrades(t.Context(), optionsTradablePair, asset.Options, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestGetAggregatedTradesBatched exercises TestGetAggregatedTradesBatched to ensure our date and limit scanning works correctly
// This test is susceptible to failure if volumes change a lot, during wash trading or zero-fee periods
// In live tests, 45 minutes is expected to return more than 1000 records
func TestGetAggregatedTradesBatched(t *testing.T) {
	t.Parallel()
	startTime, err := time.Parse(time.RFC3339, "2020-01-02T15:04:05Z")
	require.NoError(t, err)

	expectTime, err := time.Parse(time.RFC3339Nano, "2020-01-02T16:19:04.831Z")
	require.NoError(t, err)

	tests := []struct {
		name string
		// mock test or live test
		mock         bool
		args         *AggregatedTradeRequestParams
		numExpected  int
		lastExpected time.Time
	}{
		{
			name: "mock batch with timerange",
			mock: true,
			args: &AggregatedTradeRequestParams{
				Symbol:    usdtmTradablePair,
				StartTime: startTime,
				EndTime:   startTime.Add(75 * time.Minute),
			},
			numExpected:  1012,
			lastExpected: time.Date(2020, 1, 2, 16, 18, 31, int(919*time.Millisecond), time.UTC),
		},
		{
			name: "batch with timerange",
			args: &AggregatedTradeRequestParams{
				Symbol:    usdtmTradablePair,
				StartTime: startTime,
				EndTime:   startTime.Add(75 * time.Minute),
			},
			numExpected:  12130,
			lastExpected: expectTime,
		},
		{
			name: "mock custom limit with start time set, no end time",
			mock: true,
			args: &AggregatedTradeRequestParams{
				Symbol:    usdtmTradablePair,
				StartTime: startTime,
				Limit:     1001,
			},
			numExpected:  1001,
			lastExpected: time.Date(2020, 1, 2, 15, 18, 39, int(226*time.Millisecond), time.UTC),
		},
		{
			name: "custom limit with start time set, no end time",
			args: &AggregatedTradeRequestParams{
				Symbol:    usdtmTradablePair,
				StartTime: time.Date(2020, 11, 18, 23, 0, 28, 921, time.UTC),
				Limit:     1001,
			},
			numExpected:  1001,
			lastExpected: time.Date(2020, 11, 18, 23, 1, 33, int(62*time.Millisecond*10), time.UTC),
		},
		{
			name: "mock recent trades",
			mock: true,
			args: &AggregatedTradeRequestParams{
				Symbol: usdtmTradablePair,
				Limit:  3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.mock != mockTests {
				t.Skip("mock mismatch, skipping")
			}
			result, err := e.GetAggregatedTrades(t.Context(), tt.args)
			assert.NoError(t, err)

			assert.Len(t, result, tt.numExpected)
			lastTradeTime := result[len(result)-1].TimeStamp
			if !lastTradeTime.Time().Equal(tt.lastExpected) {
				t.Errorf("last trade expected %v, got %v", tt.lastExpected.UTC(), lastTradeTime.Time().UTC())
			}
		})
	}
}

func TestGetAggregatedTradesErrors(t *testing.T) {
	t.Parallel()
	startTime, err := time.Parse(time.RFC3339, "2020-01-02T15:04:05Z")
	require.NoError(t, err)
	tests := []struct {
		name string
		args *AggregatedTradeRequestParams
	}{
		{
			name: "get recent trades does not support custom limit",
			args: &AggregatedTradeRequestParams{
				Symbol: usdtmTradablePair,
				Limit:  1001,
			},
		},
		{
			name: "start time and fromId cannot be both set",
			args: &AggregatedTradeRequestParams{
				Symbol:    usdtmTradablePair,
				StartTime: startTime,
				EndTime:   startTime.Add(75 * time.Minute),
				FromID:    2,
			},
		},
		{
			name: "can't get most recent 5000 (more than 1000 not allowed)",
			args: &AggregatedTradeRequestParams{
				Symbol: usdtmTradablePair,
				Limit:  5000,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetAggregatedTrades(t.Context(), tt.args)
			require.Error(t, err)
		})
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// -----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange: e.Name,
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.LTC,
			Quote:     currency.BTC,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currency.NewPair(currency.LTC, currency.BTC),
		AssetType: asset.Spot,
	})
	assert.NoError(t, err)
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOrders(t.Context(), &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      spotTradablePair,
		AssetType: asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      optionsTradablePair,
		AssetType: asset.Options,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	items := asset.Items{
		asset.CoinMarginedFutures,
		asset.USDTMarginedFutures,
		asset.Spot,
		asset.Margin,
	}
	for i := range items {
		assetType := items[i]
		t.Run(fmt.Sprintf("Update info of account [%s]", assetType.String()), func(t *testing.T) {
			t.Parallel()
			result, err := e.UpdateAccountBalances(t.Context(), assetType)
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestWrapperGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{spotTradablePair},
		AssetType: asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{coinmTradablePair},
		AssetType: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{usdtmTradablePair},
		AssetType: asset.USDTMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{optionsTradablePair},
		AssetType: asset.Options,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWrapperGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures})
	assert.Error(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	p, err := currency.NewPairFromString("EOSUSD_PERP")
	require.NoError(t, err)
	result, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Type:        order.AnyType,
		Side:        order.AnySide,
		FromOrderID: "123",
		Pairs:       currency.Pairs{p},
		AssetType:   asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Type:        order.AnyType,
		Side:        order.AnySide,
		FromOrderID: "123",
		Pairs:       currency.Pairs{usdtmTradablePair},
		AssetType:   asset.USDTMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	p, err := currency.NewPairFromString("EOS-USDT")
	require.NoError(t, err)
	fPair, err := e.FormatExchangeCurrency(p, asset.CoinMarginedFutures)
	require.NoError(t, err)
	err = e.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.CoinMarginedFutures,
		Pair:      fPair,
		OrderID:   "1234",
	})
	require.NoError(t, err)
	p2, err := currency.NewPairFromString("BTC-USDT")
	require.NoError(t, err)
	fpair2, err := e.FormatExchangeCurrency(p2, asset.USDTMarginedFutures)
	require.NoError(t, err)
	err = e.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.USDTMarginedFutures,
		Pair:      fpair2,
		OrderID:   "1234",
	})
	require.NoError(t, err)
	err = e.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.Options,
		Pair:      fpair2,
		OrderID:   "1234",
	})
	assert.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	tradablePairs, err := e.FetchTradablePairs(t.Context(),
		asset.CoinMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, tradablePairs, "no tradable pairs")
	result, err := e.GetOrderInfo(t.Context(), "123", tradablePairs[0], asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetAllCoinsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllCoinsInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(),
		&withdraw.Request{
			Exchange:    e.Name,
			Amount:      -1,
			Currency:    currency.BTC,
			Description: "WITHDRAW IT ALL",
			Crypto: withdraw.CryptoRequest{
				Address: core.BitcoinDonationAddress,
			},
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDepositHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.DepositHistory(t.Context(), currency.ETH, "", endTime, startTime, 0, 10000)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.DepositHistory(t.Context(), currency.ETH, "", startTime, endTime, 0, 10000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.ETH, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.Equal(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	require.Equal(t, err, common.ErrFunctionNotSupported)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.USDT, "", currency.BNB.String())
	require.False(t, sharedtestvalues.AreAPICredentialsSet(e) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests, "error cannot be nil")
	assert.False(t, mockTests && err != nil, err)
}

func BenchmarkWsHandleData(b *testing.B) {
	b.ReportAllocs()
	ap, err := e.CurrencyPairs.GetPairs(asset.Spot, false)
	require.NoError(b, err)
	err = e.CurrencyPairs.StorePairs(asset.Spot, ap, true)
	require.NoError(b, err)

	data, err := os.ReadFile("testdata/wsHandleData.json")
	require.NoError(b, err)
	lines := bytes.Split(data, []byte("\n"))
	require.Len(b, lines, 8)
	go func() {
		for {
			select {
			case _, ok := <-e.Websocket.DataHandler.C:
				if !ok {
					return
				}
			}
		}
	}()
	for b.Loop() {
		for x := range lines {
			assert.NoError(b, e.wsHandleData(b.Context(), lines[x]))
		}
	}
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	channels, err := e.generateSubscriptions() // Note: We grab this before it's overwritten by MockWsInstance below
	require.NoError(t, err, "generateSubscriptions must not error")
	if mockTests {
		exp := []string{"btcusdt@depth@100ms", "btcusdt@kline_1m", "btcusdt@ticker", "btcusdt@trade", "dogeusdt@depth@100ms", "dogeusdt@kline_1m", "dogeusdt@ticker", "dogeusdt@trade"}
		mock := func(tb testing.TB, msg []byte, w *gws.Conn) error {
			tb.Helper()
			var req WsPayload
			require.NoError(tb, json.Unmarshal(msg, &req), "Unmarshal must not error")
			require.ElementsMatch(tb, req.Params, exp, "Params must have correct channels")
			return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"result":null,"id":"%s"}`, req.ID))
		}
		e = testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, mock))
	} else {
		testexch.SetupWs(t, e)
	}
	conn, err := e.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, conn)

	err = e.Subscribe(t.Context(), conn, channels)
	require.NoError(t, err)
	err = e.Unsubscribe(t.Context(), conn, channels)
	assert.NoError(t, err)
}

func TestSubscribeBadResp(t *testing.T) {
	t.Parallel()
	channels := subscription.List{
		{Channel: "moons@ticker"},
	}
	mock := func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		var req WsPayload
		err := json.Unmarshal(msg, &req)
		require.NoError(tb, err, "Unmarshal must not error")
		return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"result":{"error":"carrots"},"id":"%s"}`, req.ID))
	}
	e := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, mock))

	conn, err := e.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, conn)

	err = e.Subscribe(t.Context(), conn, channels)
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "Subscribe should error ErrSubscriptionFailure")
	require.ErrorIs(t, err, common.ErrUnknownError, "Subscribe should error errUnknownError")
	assert.ErrorContains(t, err, "carrots", "Subscribe should error containing the carrots")
}

func TestWsTickerUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"btcusdt@ticker","data":{"e":"24hrTicker","E":1580254809477,"s":"ETHBTC","p":"420.97000000","P":"4.720","w":"9058.27981278","x":"8917.98000000","c":"9338.96000000","Q":"0.17246300","b":"9338.03000000","B":"0.18234600","a":"9339.70000000","A":"0.14097600","o":"8917.99000000","h":"9373.19000000","l":"8862.40000000","v":"72229.53692000","q":"654275356.16896672","O":1580168409456,"C":1580254809456,"F":235294268,"L":235894703,"n":600436}}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsKlineUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"btcusdt@kline_1m","data":{
	  "e": "kline",
	  "E": 1234567891,   
	  "s": "ETHBTC",    
	  "k": {
		"t": 1234000001, 
		"T": 1234600001, 
		"s": "BTCUSDT",  
		"i": "1m",      
		"f": 100,       
		"L": 200,       
		"o": "0.0010",  
		"c": "0.0020",  
		"h": "0.0025",  
		"l": "0.0015",  
		"v": "1000",    
		"n": 100,       
		"x": false,     
		"q": "1.0000",  
		"V": "500",     
		"Q": "0.500",   
		"B": "123456"   
	  }
	}}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTradeUpdate(t *testing.T) {
	t.Parallel()
	e.SetSaveTradeDataStatus(true)
	pressXToJSON := []byte(`{"stream":"btcusdt@trade","data":{
	  "e": "trade",     
	  "E": 1234567891,   
	  "s": "ETHBTC",    
	  "t": 12345,       
	  "p": "0.001",     
	  "q": "100",       
	  "b": 88,          
	  "a": 50,          
	  "T": 1234567851,   
	  "m": true,        
	  "M": true         
	}}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsDepthUpdate(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	e.setupOrderbookManager(t.Context())
	seedLastUpdateID := int64(161)
	book := OrderBook{
		Asks: orderbook.LevelsArrayPriceAmount(orderbook.Levels{
			{Price: 6621.80000000, Amount: 0.00198100},
			{Price: 6622.14000000, Amount: 4.00000000},
			{Price: 6622.46000000, Amount: 2.30000000},
			{Price: 6622.47000000, Amount: 1.18633300},
			{Price: 6622.64000000, Amount: 4.00000000},
			{Price: 6622.73000000, Amount: 0.02900000},
			{Price: 6622.76000000, Amount: 0.12557700},
			{Price: 6622.81000000, Amount: 2.08994200},
			{Price: 6622.82000000, Amount: 0.01500000},
			{Price: 6623.17000000, Amount: 0.16831300},
		}),
		Bids: orderbook.LevelsArrayPriceAmount(orderbook.Levels{
			{Price: 6621.55000000, Amount: 0.16356700},
			{Price: 6621.45000000, Amount: 0.16352600},
			{Price: 6621.41000000, Amount: 0.86091200},
			{Price: 6621.25000000, Amount: 0.16914100},
			{Price: 6621.23000000, Amount: 0.09193600},
			{Price: 6621.22000000, Amount: 0.00755100},
			{Price: 6621.13000000, Amount: 0.08432000},
			{Price: 6621.03000000, Amount: 0.00172000},
			{Price: 6620.94000000, Amount: 0.30506700},
			{Price: 6620.93000000, Amount: 0.00200000},
		}),
		LastUpdateID: seedLastUpdateID,
	}

	update1 := []byte(`{"stream":"btcusdt@depth","data":{ "e": "depthUpdate", "E": 1234567881, "s": usdtmTradablePair, "U": 157, "u": 160, "b": [ ["6621.45", "0.3"] ], "a": [ ["6622.46", "1.5"] ] }}`)

	p := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	err := e.SeedLocalCacheWithBook(p, &book)
	require.NoError(t, err)

	if err := e.wsHandleData(t.Context(), update1); err != nil {
		t.Fatal(err)
	}

	e.obm.state[currency.BTC][currency.USDT][asset.Spot].fetchingBook = false

	ob, err := e.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
	require.NoError(t, err)

	exp, got := seedLastUpdateID, ob.LastUpdateID
	require.Equalf(t, exp, got, "Last update id of orderbook for old update. Exp: %d, got: %d", exp, got)
	expAmnt, gotAmnt := 2.3, ob.Asks[2].Amount
	require.Equalf(t, expAmnt, gotAmnt, "Ask altered by outdated update. Exp: %f, got %f", expAmnt, gotAmnt)
	expAmnt, gotAmnt = 0.163526, ob.Bids[1].Amount
	require.Equalf(t, expAmnt, gotAmnt, "Bid altered by outdated update. Exp: %f, got %f", expAmnt, gotAmnt)

	update2 := []byte(`{"stream":"btcusdt@depth","data":{ "e": "depthUpdate", "E": 1234567892, "s": usdtmTradablePair, "U": 161, "u": 165, "b": [ ["6621.45", "0.163526"] ], "a": [ ["6622.46", "2.3"], ["6622.47", "1.9"] ] }}`)

	if err = e.wsHandleData(t.Context(), update2); err != nil {
		t.Error(err)
	}

	ob, err = e.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
	require.NoError(t, err)
	exp, got = int64(165), ob.LastUpdateID
	require.Equalf(t, exp, got, "Unexpected Last update id of orderbook for new update. Exp: %d, got: %d", exp, got)
	expAmnt, gotAmnt = 2.3, ob.Asks[2].Amount
	require.Equalf(t, expAmnt, gotAmnt, "Unexpected Ask amount. Exp: %f, got %f", expAmnt, gotAmnt)
	expAmnt, gotAmnt = 1.9, ob.Asks[3].Amount
	require.Equal(t, expAmnt, gotAmnt, "Unexpected Ask amount. Exp: %f, got %f", exp, got)
	expAmnt, gotAmnt = 0.163526, ob.Bids[1].Amount
	require.Equal(t, expAmnt, gotAmnt, "Unexpected Bid amount. Exp: %f, got %f", exp, got)

	// reset order book sync status
	e.obm.state[currency.BTC][currency.USDT][asset.Spot].lastUpdateID = 0
}

func TestWsBalanceUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{
  "e": "balanceUpdate",         
  "E": 1573200697110,           
  "a": "BTC",                   
  "d": "100.00000000",          
  "T": 1573200697068}}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOCO(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{
  "e": "listStatus",                
  "E": 1564035303637,               
  "s": "ETHBTC",                    
  "g": 2,                           
  "c": "OCO",                       
  "l": "EXEC_STARTED",              
  "L": "EXECUTING",                 
  "r": "NONE",                      
  "C": "F4QN4G8DlFATFlIUQ0cjdD",    
  "T": 1564035303625,               
  "O": [                            
    {
      "s": "ETHBTC",                
      "i": 17,                      
      "c": "AJYsMjErWJesZvqlJCTUgL" 
    },
    {
      "s": "ETHBTC",
      "i": 18,
      "c": "bfYPSQdLoqAJeNrOr9adzq"
    }
  ]
}}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWsAuthStreamKey(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	streamKey, err := e.GetWsAuthStreamKey(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, streamKey)
}

func TestMaintainWsAuthStreamKey(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.MaintainWsAuthStreamKey(t.Context())
	require.NoError(t, err)
}

func TestExecutionTypeToOrderStatus(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []TestCases{
		{Case: "NEW", Result: order.New},
		{Case: "PARTIALLY_FILLED", Result: order.PartiallyFilled},
		{Case: "FILLED", Result: order.Filled},
		{Case: "CANCELED", Result: order.Cancelled},
		{Case: "PENDING_CANCEL", Result: order.PendingCancel},
		{Case: "REJECTED", Result: order.Rejected},
		{Case: "EXPIRED", Result: order.Expired},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := stringToOrderStatus(testCases[i].Case)
		require.Equal(t, result, testCases[i].Result, "Expected: %v, received: %v", testCases[i].Result, result)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	for assetType, pair := range assetToTradablePairMap {
		result, err := e.GetHistoricCandles(t.Context(), pair, assetType, kline.OneDay, startTime, endTime)
		require.NoErrorf(t, err, "%v %v", assetType, err)
		require.NotNil(t, result)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	for assetType, pair := range assetToTradablePairMap {
		result, err := e.GetHistoricCandlesExtended(t.Context(), pair, assetType, kline.OneDay, startTime, endTime)
		require.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		output   string
	}{
		{
			kline.OneMin,
			"1m",
		},
		{
			kline.OneDay,
			"1d",
		},
		{
			kline.OneWeek,
			"1w",
		},
		{
			kline.OneMonth,
			"1M",
		},
	} {
		t.Run(tc.output, func(t *testing.T) {
			t.Parallel()
			ret := e.FormatExchangeKlineInterval(tc.interval)
			require.Equal(t, ret, tc.output, "unexpected result return expected: %v received: %v", tc.output, ret)
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	pair := usdtmTradablePair
	result, err := e.GetRecentTrades(t.Context(), pair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
	result, err = e.GetRecentTrades(t.Context(),
		pair, asset.USDTMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
	pair.Base = currency.NewCode("BTCUSD")
	pair.Quote = currency.PERP
	result, err = e.GetRecentTrades(t.Context(), pair, asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.False(t, sharedtestvalues.AreAPICredentialsSet(e) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests, "error cannot be nil")
	assert.False(t, mockTests && err != nil, err)
}

func TestSeedLocalCache(t *testing.T) {
	t.Parallel()
	err := e.SeedLocalCache(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	exp := subscription.List{}
	pairs, err := e.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)
	wsFmt := currency.PairFormat{Uppercase: false, Delimiter: ""}
	baseExp := subscription.List{
		{Channel: subscription.CandlesChannel, QualifiedChannel: "kline_1m", Asset: asset.Spot, Interval: kline.OneMin},
		{Channel: subscription.OrderbookChannel, QualifiedChannel: "depth@100ms", Asset: asset.Spot, Interval: kline.HundredMilliseconds},
		{Channel: subscription.TickerChannel, QualifiedChannel: "ticker", Asset: asset.Spot},
		{Channel: subscription.AllTradesChannel, QualifiedChannel: "trade", Asset: asset.Spot},
	}
	for _, p := range pairs {
		for _, baseSub := range baseExp {
			sub := baseSub.Clone()
			sub.Pairs = currency.Pairs{p}
			sub.QualifiedChannel = wsFmt.Format(p) + "@" + sub.QualifiedChannel
			exp = append(exp, sub)
		}
	}
	subs, err := e.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, subs)
}

// TestFormatChannelInterval exercises formatChannelInterval
func TestFormatChannelInterval(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "@1000ms", formatChannelInterval(&subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.ThousandMilliseconds}), "1s should format correctly for Orderbook")
	assert.Equal(t, "@1m", formatChannelInterval(&subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.OneMin}), "Orderbook should format correctly")
	assert.Equal(t, "_15m", formatChannelInterval(&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.FifteenMin}), "Candles should format correctly")
}

// TestFormatChannelLevels exercises formatChannelLevels
func TestFormatChannelLevels(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "10", formatChannelLevels(&subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 10}), "Levels should format correctly")
	assert.Empty(t, formatChannelLevels(&subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 0}), "Levels should format correctly")
}

func TestProcessOrderbookUpdate(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	e.setupOrderbookManager(t.Context())
	p := currency.NewBTCUSDT()
	var depth WebsocketDepthStream
	err := json.Unmarshal([]byte(`{"E":1608001030784,"U":7145637266,"a":[["19455.19000000","0.59490200"],["19455.37000000","0.00000000"],["19456.11000000","0.00000000"],["19456.16000000","0.00000000"],["19458.67000000","0.06400000"],["19460.73000000","0.05139800"],["19461.43000000","0.00000000"],["19464.59000000","0.00000000"],["19466.03000000","0.45000000"],["19466.36000000","0.00000000"],["19508.67000000","0.00000000"],["19572.96000000","0.00217200"],["24386.00000000","0.00256600"]],"b":[["19455.18000000","2.94649200"],["19453.15000000","0.01233600"],["19451.18000000","0.00000000"],["19446.85000000","0.11427900"],["19446.74000000","0.00000000"],["19446.73000000","0.00000000"],["19444.45000000","0.14937800"],["19426.75000000","0.00000000"],["19416.36000000","0.36052100"]],"e":"depthUpdate","s":usdtmTradablePair,"u":7145637297}`),
		&depth)
	require.NoError(t, err)

	err = e.obm.stageWsUpdate(&depth, p, asset.Spot)
	require.NoError(t, err)

	err = e.obm.fetchBookViaREST(p)
	require.NoError(t, err)

	err = e.obm.cleanup(p)
	require.NoError(t, err)

	// reset order book sync status
	e.obm.state[currency.BTC][currency.USDT][asset.Spot].lastUpdateID = 0
}

func TestUFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := e.UFuturesHistoricalTrades(t.Context(), currency.EMPTYPAIR, "", 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UFuturesHistoricalTrades(t.Context(), usdtmTradablePair, "", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UFuturesHistoricalTrades(t.Context(), usdtmTradablePair, "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetExchangeOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	assetTypes := e.GetAssetTypes(true)
	for a := range assetTypes {
		err := e.UpdateOrderExecutionLimits(t.Context(), assetTypes[a])
		require.NoError(t, err)
	}

	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	cmfCP, err := currency.NewPairFromStrings("BTCUSD", "PERP")
	require.NoError(t, err)

	l, err := e.GetOrderExecutionLimits(asset.CoinMarginedFutures, cmfCP)
	require.NoError(t, err)
	require.NotEmpty(t, l, "exchange limit should be loaded")

	err = l.Validate(0.000001, 0.1, order.Limit)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	err = l.Validate(0.01, 1, order.Limit)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
}

func TestWsOrderExecutionReport(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	payload := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"executionReport","E":1616627567900,"s":usdtmTradablePair,"c":"c4wyKsIhoAaittTYlIVLqk","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00028400","p":"52789.10000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":5340845958,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":"BTC","T":1616627567900,"t":-1,"I":11388173160,"w":true,"m":false,"M":false,"O":1616627567900,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1616627567900}}`)
	// this is a buy BTC order, normally commission is charged in BTC, vice versa.
	expectedResult := order.Detail{
		Price:                52789.1,
		Amount:               0.00028400,
		AverageExecutedPrice: 0,
		QuoteAmount:          0,
		ExecutedAmount:       0,
		RemainingAmount:      0.00028400,
		Cost:                 0,
		CostAsset:            currency.USDT,
		Fee:                  0,
		FeeAsset:             currency.BTC,
		Exchange:             "Binance",
		OrderID:              "5340845958",
		ClientOrderID:        "c4wyKsIhoAaittTYlIVLqk",
		Type:                 order.Limit,
		Side:                 order.Buy,
		Status:               order.New,
		AssetType:            asset.Spot,
		Date:                 time.UnixMilli(1616627567900),
		LastUpdated:          time.UnixMilli(1616627567900),
		Pair:                 usdtmTradablePair,
		TimeInForce:          order.GoodTillCancel,
	}
	// empty the channel. otherwise mock_test will fail
drain:
	for {
		select {
		case <-e.Websocket.DataHandler.C:
		default:
			break drain
		}
	}

	err := e.wsHandleData(t.Context(), payload)
	if err != nil {
		t.Fatal(err)
	}
	res := <-e.Websocket.DataHandler.C
	switch r := res.Data.(type) {
	case *order.Detail:
		// The WebSocket handler returns two order details for a single symbol:
		// one for spot and one for margin. To avoid mismatches due to asset type
		// precedence, we align the expected asset type with the received one.
		if r.AssetType == asset.Margin {
			expectedResult.AssetType = asset.Margin
		}
		require.True(t, reflect.DeepEqual(expectedResult, *r), "results do not match:\nexpected: %v\nreceived: %v", expectedResult, *r)
	default:
		t.Fatalf("expected type order.Detail, found %T", res)
	}

	payload = []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"executionReport","E":1616633041556,"s":"BTCUSDT","c":"YeULctvPAnHj5HXCQo9Mob","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00028600","p":"52436.85000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":5341783271,"l":"0.00028600","z":"0.00028600","L":"52436.85000000","n":"0.00000029","N":"BTC","T":1616633041555,"t":726946523,"I":11390206312,"w":false,"m":false,"M":true,"O":1616633041555,"Z":"14.99693910","Y":"14.99693910","Q":"0.00000000","W":1616633041555}}`)
	err = e.wsHandleData(t.Context(), payload)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsOutboundAccountPosition(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"outboundAccountPosition","E":1616628815745,"u":1616628815745,"B":[{"a":"BTC","f":"0.00225109","l":"0.00123000"},{"a":"BNB","f":"0.00000000","l":"0.00000000"},{"a":"USDT","f":"54.43390661","l":"0.00000000"}]}}`)
	if err := e.wsHandleData(t.Context(), payload); err != nil {
		t.Fatal(err)
	}
}

func TestFormatExchangeCurrency(t *testing.T) {
	t.Parallel()
	type testos struct {
		name              string
		pair              currency.Pair
		asset             asset.Item
		expectedDelimiter string
	}
	testerinos := []testos{
		{
			name:              "spot-btcusdt",
			pair:              currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
			asset:             asset.Spot,
			expectedDelimiter: "",
		},
		{
			name:              "coinmarginedfutures-btcusd_perp",
			pair:              currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.DashDelimiter),
			asset:             asset.CoinMarginedFutures,
			expectedDelimiter: currency.UnderscoreDelimiter,
		},
		{
			name:              "coinmarginedfutures-btcusd_211231",
			pair:              currency.NewPairWithDelimiter("BTCUSD", "211231", currency.DashDelimiter),
			asset:             asset.CoinMarginedFutures,
			expectedDelimiter: currency.UnderscoreDelimiter,
		},
		{
			name:              "margin-ltousdt",
			pair:              currency.NewPairWithDelimiter("LTO", "USDT", currency.UnderscoreDelimiter),
			asset:             asset.Margin,
			expectedDelimiter: "",
		},
		{
			name:              "usdtmarginedfutures-btcusdt",
			pair:              currency.NewPairWithDelimiter("btc", "usdt", currency.DashDelimiter),
			asset:             asset.USDTMarginedFutures,
			expectedDelimiter: "",
		},
		{
			name:              "usdtmarginedfutures-btcusdt_211231",
			pair:              usdtmTradablePair,
			asset:             asset.USDTMarginedFutures,
			expectedDelimiter: currency.UnderscoreDelimiter,
		},
	}
	for i := range testerinos {
		tt := testerinos[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.FormatExchangeCurrency(tt.pair, tt.asset)
			require.NoError(t, err)
			require.Equal(t, tt.expectedDelimiter, result.Delimiter)
		})
	}
}

func TestFormatSymbol(t *testing.T) {
	t.Parallel()
	type testos struct {
		name           string
		pair           currency.Pair
		asset          asset.Item
		expectedString string
	}
	testerinos := []testos{
		{
			name:           "spot-BTCUSDT",
			pair:           currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
			asset:          asset.Spot,
			expectedString: "BTCUSDT",
		},
		{
			name:           "coinmarginedfutures-btcusdperp",
			pair:           currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.DashDelimiter),
			asset:          asset.CoinMarginedFutures,
			expectedString: "BTCUSD_PERP",
		},
		{
			name:           "coinmarginedfutures-BTCUSD_211231",
			pair:           currency.NewPairWithDelimiter("BTCUSD", "211231", currency.DashDelimiter),
			asset:          asset.CoinMarginedFutures,
			expectedString: "BTCUSD_211231",
		},
		{
			name:           "margin-LTOUSDT",
			pair:           currency.NewPairWithDelimiter("LTO", "USDT", currency.UnderscoreDelimiter),
			asset:          asset.Margin,
			expectedString: "LTOUSDT",
		},
		{
			name:           "usdtmarginedfutures-BTCUSDT",
			pair:           currency.NewPairWithDelimiter("btc", "usdt", currency.DashDelimiter),
			asset:          asset.USDTMarginedFutures,
			expectedString: "BTCUSDT",
		},
		{
			name:           "usdtmarginedfutures-BTCUSDT_211231",
			pair:           usdtmTradablePair,
			asset:          asset.USDTMarginedFutures,
			expectedString: "BTCUSDT_211231",
		},
	}
	for _, tt := range testerinos {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.FormatSymbol(tt.pair, tt.asset)
			require.NoError(t, err)
			require.Equal(t, tt.expectedString, result)
		})
	}
}

func TestFormatUSDTMarginedFuturesPair(t *testing.T) {
	t.Parallel()
	pairFormat := currency.PairFormat{Uppercase: true}
	resp := e.formatUSDTMarginedFuturesPair(currency.NewPair(currency.DOGE, currency.USDT), pairFormat)
	require.Equal(t, "DOGEUSDT", resp.String())

	resp = e.formatUSDTMarginedFuturesPair(currency.NewPair(currency.DOGE, currency.NewCode("1234567890")), pairFormat)
	assert.Equal(t, "DOGE_1234567890", resp.String())
}

func TestFetchExchangeLimits(t *testing.T) {
	t.Parallel()
	l, err := e.FetchExchangeLimits(t.Context(), asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, l, "Should get some limits back")

	l, err = e.FetchExchangeLimits(t.Context(), asset.Margin)
	require.NoError(t, err)
	require.NotEmpty(t, l, "Should get some limits back")

	_, err = e.FetchExchangeLimits(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchExchangeLimits should error on other asset types")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.CurrencyPairs.GetPairs(a, false)
			require.NoError(t, err, "GetPairs must not error")
			l, err := e.GetOrderExecutionLimits(a, pairs[0])
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			assert.Positive(t, l.MinPrice, "MinPrice should be positive")
			assert.Positive(t, l.MaxPrice, "MaxPrice should be positive")
			assert.Positive(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive")
			assert.Positive(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive")
			assert.Positive(t, l.MaximumBaseAmount, "MaximumBaseAmount should be positive")
			assert.Positive(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive")
			assert.Positive(t, l.MarketMaxQty, "MarketMaxQty should be positive")
			assert.Positive(t, l.MaxTotalOrders, "MaxTotalOrders should be positive")
			switch a {
			case asset.Spot, asset.Margin:
				assert.Positive(t, l.MaxIcebergParts, "MaxIcebergParts should be positive")
			case asset.USDTMarginedFutures:
				assert.Positive(t, l.MinNotional, "MinNotional should be positive")
				fallthrough
			case asset.CoinMarginedFutures:
				assert.Positive(t, l.MultiplierUp, "MultiplierUp should be positive")
				assert.Positive(t, l.MultiplierDown, "MultiplierDown should be positive")
				assert.Positive(t, l.MarketMinQty, "MarketMinQty should be positive")
				assert.Positive(t, l.MarketStepIncrementSize, "MarketStepIncrementSize should be positive")
				assert.Positive(t, l.MaxAlgoOrders, "MaxAlgoOrders should be positive")
			}
		})
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		StartDate:            startTime,
		EndDate:              endTime,
		IncludePayments:      true,
		IncludePredictedRate: true,
	})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewBTCUSDT(),
		StartDate:       startTime,
		EndDate:         endTime,
		PaymentCurrency: currency.DOGE,
	})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)

	r := &fundingrate.HistoricalRatesRequest{
		Asset:     asset.USDTMarginedFutures,
		Pair:      currency.NewBTCUSDT(),
		StartDate: startTime,
		EndDate:   endTime,
	}
	if sharedtestvalues.AreAPICredentialsSet(e) {
		r.IncludePayments = true
	}
	result, err := e.GetHistoricalFundingRates(t.Context(), r)
	require.NoError(t, err)
	assert.NotNil(t, result)

	r.Asset = asset.CoinMarginedFutures
	r.Pair, err = currency.NewPairFromString("BTCUSD_PERP")
	require.NoError(t, err)

	result, err = e.GetHistoricalFundingRates(t.Context(), r)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	cp := currency.NewBTCUSDT()
	_, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 cp,
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
	err = e.CurrencyPairs.EnablePair(asset.USDTMarginedFutures, cp)
	require.True(t, err == nil || errors.Is(err, currency.ErrPairAlreadyEnabled), err)

	result, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  cp,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Binary, usdtmTradablePair)
	require.NoError(t, err)
	require.False(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, usdtmTradablePair)
	require.NoError(t, err)
	require.False(t, is)
	is, err = e.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewPair(currency.BTC, currency.PERP))
	require.NoError(t, err)
	require.True(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.USDTMarginedFutures, usdtmTradablePair)
	require.NoError(t, err)
	require.True(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.PERP))
	require.NoError(t, err)
	assert.False(t, is)
}

func TestGetUserMarginInterestHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUserMarginInterestHistory(t.Context(), currency.USDT, usdtmTradablePair, endTime, startTime, 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserMarginInterestHistory(t.Context(), currency.USDT, usdtmTradablePair, startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetForceLiquidiationRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetForceLiquidiationRecord(t.Context(), endTime, startTime, usdtmTradablePair, 0, 12)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetForceLiquidiationRecord(t.Context(), startTime, endTime, usdtmTradablePair, 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCrossMarginAccountDetail(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountsOrder(t.Context(), currency.EMPTYPAIR, "", false, 112233424)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetMarginAccountsOrder(t.Context(), usdtmTradablePair, "", false, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAccountsOrder(t.Context(), usdtmTradablePair, "", false, 112233424)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAccountsOpenOrders(t.Context(), assetToTradablePairMap[asset.Margin], false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountAllOrders(t.Context(), currency.EMPTYPAIR, true, time.Time{}, time.Time{}, 0, 20)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := getTime()
	_, err = e.GetMarginAccountAllOrders(t.Context(), assetToTradablePairMap[asset.Margin], true, endTime, startTime, 0, 20)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAccountAllOrders(t.Context(), assetToTradablePairMap[asset.Margin], true, startTime, endTime, 1, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetAssetsMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	is, err := e.GetAssetsMode(t.Context())
	require.NoError(t, err)

	err = e.SetAssetsMode(t.Context(), !is)
	require.NoError(t, err)

	err = e.SetAssetsMode(t.Context(), is)
	assert.NoError(t, err)
}

func TestGetAssetsMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAssetsMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	_, err := e.GetCollateralMode(t.Context(), asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.GetCollateralMode(t.Context(), asset.CoinMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetCollateralMode(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	err := e.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.PortfolioMode)
	require.ErrorIs(t, err, order.ErrCollateralInvalid)
	err = e.SetCollateralMode(t.Context(), asset.Spot, collateral.SingleMode)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	err = e.SetCollateralMode(t.Context(), asset.CoinMarginedFutures, collateral.SingleMode)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.MultiMode)
	require.NoError(t, err)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
		Pair:                    currency.NewBTCUSDT(),
		Asset:                   asset.USDTMarginedFutures,
		MarginType:              margin.Isolated,
		OriginalAllocatedMargin: 1337,
		NewAllocatedMargin:      1333337,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionSummary(t *testing.T) {
	t.Parallel()
	p, err := currency.NewPairFromString("BTCUSD_PERP")
	require.NoError(t, err)

	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset:          asset.Spot,
		Pair:           p,
		UnderlyingPair: currency.NewPair(currency.BTC, currency.USD),
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	bb := currency.NewBTCUSDT()
	result, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  bb,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	bb.Quote = currency.BUSD
	result, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  bb,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	bb.Quote = currency.USD
	result, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset:          asset.CoinMarginedFutures,
		Pair:           p,
		UnderlyingPair: bb,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
		Asset:                     asset.USDTMarginedFutures,
		Pairs:                     []currency.Pair{currency.NewBTCUSDT()},
		StartDate:                 time.Now().Add(-time.Hour * 24 * 70),
		RespectOrderHistoryLimits: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
		Asset:                     asset.CoinMarginedFutures,
		Pairs:                     []currency.Pair{coinmTradablePair},
		StartDate:                 time.Now().Add(time.Hour * 24 * -70),
		RespectOrderHistoryLimits: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	err := e.SetMarginType(t.Context(), asset.Spot, usdtmTradablePair, margin.Isolated)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetMarginType(t.Context(), asset.USDTMarginedFutures, usdtmTradablePair, margin.Isolated)
	require.NoError(t, err)

	err = e.SetMarginType(t.Context(), asset.CoinMarginedFutures, coinmTradablePair, margin.Isolated)
	assert.NoError(t, err)
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverage(t.Context(), asset.Spot, currency.NewBTCUSDT(), 0, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), 0, order.UnknownSide)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetLeverage(t.Context(), asset.CoinMarginedFutures, coinmTradablePair, 0, order.UnknownSide)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	err := e.SetLeverage(t.Context(), asset.Spot, spotTradablePair, margin.Multi, 5, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 5, order.UnknownSide)
	require.NoError(t, err)
	err = e.SetLeverage(t.Context(), asset.CoinMarginedFutures, coinmTradablePair, margin.Multi, 5, order.UnknownSide)
	require.NoError(t, err)
}

func TestGetCryptoLoansIncomeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.CryptoLoanIncomeHistory(t.Context(), currency.USDT, "", endTime, startTime, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CryptoLoanIncomeHistory(t.Context(), currency.USDT, "", startTime, endTime, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.CryptoLoanBorrow(t.Context(), currency.EMPTYCODE, 1000, currency.BTC, 1, 7)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = e.CryptoLoanBorrow(t.Context(), currency.USDT, 1000, currency.EMPTYCODE, 1, 7)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = e.CryptoLoanBorrow(t.Context(), currency.USDT, 0, currency.BTC, 1, 0)
	require.ErrorIs(t, err, errLoanTermMustBeSet)
	_, err = e.CryptoLoanBorrow(t.Context(), currency.USDT, 0, currency.BTC, 0, 7)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CryptoLoanBorrow(t.Context(), currency.USDT, 1000, currency.BTC, 1, 7)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanBorrowHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.CryptoLoanBorrowHistory(t.Context(), 0, currency.USDT, currency.BTC, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CryptoLoanBorrowHistory(t.Context(), 0, currency.USDT, currency.BTC, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CryptoLoanOngoingOrders(t.Context(), 0, currency.USDT, currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanRepay(t *testing.T) {
	t.Parallel()
	_, err := e.CryptoLoanRepay(t.Context(), 0, 1000, 1, false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.CryptoLoanRepay(t.Context(), 42069, 0, 1, false)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CryptoLoanRepay(t.Context(), 42069, 1000, 1, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.CryptoLoanRepaymentHistory(t.Context(), 0, currency.USDT, currency.BTC, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CryptoLoanRepaymentHistory(t.Context(), 0, currency.USDT, currency.BTC, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanAdjustLTV(t *testing.T) {
	t.Parallel()
	_, err := e.CryptoLoanAdjustLTV(t.Context(), 0, true, 1)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CryptoLoanAdjustLTV(t.Context(), 42069, true, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CryptoLoanAdjustLTV(t.Context(), 42069, true, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.CryptoLoanLTVAdjustmentHistory(t.Context(), 0, currency.USDT, currency.BTC, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CryptoLoanLTVAdjustmentHistory(t.Context(), 0, currency.USDT, currency.BTC, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CryptoLoanAssetsData(t.Context(), currency.EMPTYCODE, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanCollateralAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CryptoLoanCollateralAssetsData(t.Context(), currency.EMPTYCODE, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanCheckCollateralRepayRate(t *testing.T) {
	t.Parallel()
	_, err := e.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.EMPTYCODE, currency.BNB, 69)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = e.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.BUSD, currency.EMPTYCODE, 69)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = e.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.BUSD, currency.BNB, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.BUSD, currency.BNB, 69)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanCustomiseMarginCall(t *testing.T) {
	t.Parallel()
	_, err := e.CryptoLoanCustomiseMarginCall(t.Context(), 0, currency.BTC, 0)
	require.ErrorIs(t, err, errMarginCallValueRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CryptoLoanCustomiseMarginCall(t.Context(), 1337, currency.BTC, .70)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanBorrow(t.Context(), currency.EMPTYCODE, currency.USDC, 1, 0)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = e.FlexibleLoanBorrow(t.Context(), currency.ATOM, currency.EMPTYCODE, 1, 0)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = e.FlexibleLoanBorrow(t.Context(), currency.ATOM, currency.USDC, 0, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FlexibleLoanBorrow(t.Context(), currency.ATOM, currency.USDC, 1, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FlexibleLoanOngoingOrders(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanBorrowHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.FlexibleLoanBorrowHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FlexibleLoanBorrowHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanRepay(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanRepay(t.Context(), currency.EMPTYCODE, currency.BTC, 1, false, false)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = e.FlexibleLoanRepay(t.Context(), currency.USDT, currency.EMPTYCODE, 1, false, false)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = e.FlexibleLoanRepay(t.Context(), currency.USDT, currency.BTC, 0, false, false)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FlexibleLoanRepay(t.Context(), currency.ATOM, currency.USDC, 1, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanRepayHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.FlexibleLoanRepayHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FlexibleLoanRepayHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanCollateralRepayment(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanCollateralRepayment(t.Context(), currency.EMPTYCODE, currency.USDT, 1000, true)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = e.FlexibleLoanCollateralRepayment(t.Context(), currency.BTC, currency.EMPTYCODE, 1000, true)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = e.FlexibleLoanCollateralRepayment(t.Context(), currency.BTC, currency.USDT, 0, true)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FlexibleLoanCollateralRepayment(t.Context(), currency.BTC, currency.USDT, 1000, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckCollateralRepayRate(t *testing.T) {
	t.Parallel()
	_, err := e.CheckCollateralRepayRate(t.Context(), currency.EMPTYCODE, currency.USDT)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = e.CheckCollateralRepayRate(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CheckCollateralRepayRate(t.Context(), currency.BTC, currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleLoanLiquidiationHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFlexibleLoanLiquidiationHistory(t.Context(), currency.BTC, currency.EMPTYCODE, endTime, startTime, 0, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFlexibleLoanLiquidiationHistory(t.Context(), currency.BTC, currency.EMPTYCODE, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanAdjustLTV(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanAdjustLTV(t.Context(), currency.EMPTYCODE, currency.BTC, 1, true)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = e.FlexibleLoanAdjustLTV(t.Context(), currency.USDT, currency.EMPTYCODE, 1, true)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = e.FlexibleLoanAdjustLTV(t.Context(), currency.USDT, currency.BTC, 0, true)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FlexibleLoanAdjustLTV(t.Context(), currency.USDT, currency.BTC, 1, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.FlexibleLoanLTVAdjustmentHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FlexibleLoanLTVAdjustmentHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FlexibleLoanAssetsData(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleCollateralAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FlexibleCollateralAssetsData(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
	result, err = e.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetFundingRateInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	result, err := e.UGetFundingRateInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsUFuturesConnect(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	conn, err := e.Websocket.GetConnection(asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotNil(t, conn)

	err = e.WsUFuturesConnect(t.Context(), conn)
	require.NoError(t, err)
}

func TestHandleData(t *testing.T) {
	t.Parallel()
	for k, v := range map[string]string{
		"Asset Index":                   `{"stream": "!assetIndex@arr", "data": [{ "e":"assetIndexUpdate", "E":1686749230000, "s":"ADAUSD", "i":"0.27462452", "b":"0.10000000", "a":"0.10000000", "B":"0.24716207", "A":"0.30208698", "q":"0.05000000", "g":"0.05000000", "Q":"0.26089330", "G":"0.28835575" }, { "e":"assetIndexUpdate", "E":1686749230000, "s":"USDTUSD", "i":"0.99987691", "b":"0.00010000", "a":"0.00010000", "B":"0.99977692", "A":"0.99997689", "q":"0.00010000", "g":"0.00010000", "Q":"0.99977692", "G":"0.99997689" } ]}`,
		"Contract Info":                 `{"stream": "!contractInfo", "data": {"e":"contractInfo", "E":1669356423908, "s":"IOTAUSDT", "ps":"IOTAUSDT", "ct":"PERPETUAL", "dt":4133404800000, "ot":1569398400000, "cs":"TRADING", "bks":[ { "bs":1, "bnf":0, "bnc":5000, "mmr":0.01, "cf":0, "mi":21, "ma":50 }, { "bs":2, "bnf":5000, "bnc":25000, "mmr":0.025, "cf":75, "mi":11, "ma":20 } ] }}`,
		"Force Order":                   `{"stream": "!forceOrder@arr", "data": {"e":"forceOrder", "E":1568014460893, "o":{ "s":usdtmTradablePair, "S":"SELL", "o":"LIMIT", "f":"IOC", "q":"0.014", "p":"9910", "ap":"9910", "X":"FILLED", "l":"0.014", "z":"0.014", "T":1568014460893 }}}`,
		"All BookTicker":                `{"stream": "!bookTicker","data":{"e":"bookTicker","u":3682854202063,"s":"NEARUSDT","b":"2.4380","B":"20391","a":"2.4390","A":"271","T":1703015198639,"E":1703015198640}}`,
		"Multiple Market Ticker":        `{"stream": "!ticker@arr", "data": [{"e":"24hrTicker","E":1703018247910,"s":"ICPUSDT","p":"-0.540000","P":"-5.395","w":"9.906194","c":"9.470000","Q":"1","o":"10.010000","h":"10.956000","l":"9.236000","v":"34347035","q":"340248403.001000","O":1702931820000,"C":1703018247909,"F":78723309,"L":80207941,"n":1484628},{"e":"24hrTicker","E":1703018247476,"s":"MEMEUSDT","p":"0.0020900","P":"7.331","w":"0.0300554","c":"0.0305980","Q":"7568","o":"0.0285080","h":"0.0312730","l":"0.0284120","v":"5643663185","q":"169622568.3721920","O":1702931820000,"C":1703018247475,"F":88665791,"L":89517438,"n":851643},{"e":"24hrTicker","E":1703018247822,"s":"SOLUSDT","p":"0.8680","P":"1.192","w":"74.4933","c":"73.6900","Q":"21","o":"72.8220","h":"76.3840","l":"71.8000","v":"26283647","q":"1957955612.4830","O":1702931820000,"C":1703018247820,"F":1126774871,"L":1129007642,"n":2232761},{"e":"24hrTicker","E":1703018247254,"s":"IMXUSDT","p":"0.0801","P":"3.932","w":"2.1518","c":"2.1171","Q":"225","o":"2.0370","h":"2.2360","l":"2.0319","v":"59587050","q":"128216496.4538","O":1702931820000,"C":1703018247252,"F":169814879,"L":170587124,"n":772246},{"e":"24hrTicker","E":1703018247309,"s":"DYDXUSDT","p":"-0.036","P":"-1.255","w":"2.896","c":"2.832","Q":"169.6","o":"2.868","h":"2.987","l":"2.782","v":"81690098.5","q":"236599791.383","O":1702931820000,"C":1703018247308,"F":385238821,"L":385888621,"n":649799},{"e":"24hrTicker","E":1703018247240,"s":"ONTUSDT","p":"0.0022","P":"1.011","w":"0.2213","c":"0.2197","Q":"45.7","o":"0.2175","h":"0.2251","l":"0.2157","v":"60880132.6","q":"13471239.8637","O":1702931820000,"C":1703018247238,"F":186008331,"L":186088275,"n":79945},{"e":"24hrTicker","E":1703018247658,"s":"AAVEUSDT","p":"4.660","P":"4.778","w":"102.969","c":"102.190","Q":"0.4","o":"97.530","h":"108.000","l":"97.370","v":"1205430.6","q":"124121750.870","O":1702931820000,"C":1703018247657,"F":343017862,"L":343487276,"n":469414},{"e":"24hrTicker","E":1703018247545,"s":"USTCUSDT","p":"0.0018500","P":"5.628","w":"0.0348991","c":"0.0347200","Q":"2316","o":"0.0328700","h":"0.0371100","l":"0.0328000","v":"2486985654","q":"86793545.3903700","O":1702931820000,"C":1703018247544,"F":32136013,"L":32601947,"n":465935},{"e":"24hrTicker","E":1703018247997,"s":"FTMUSDT","p":"-0.005000","P":"-1.221","w":"0.409721","c":"0.404400","Q":"1421","o":"0.409400","h":"0.421200","l":"0.392100","v":"471077518","q":"193010517.884400","O":1702931820000,"C":1703018247996,"F":716077491,"L":716712548,"n":635055},{"e":"24hrTicker","E":1703018247338,"s":"LRCUSDT","p":"-0.00290","P":"-1.104","w":"0.26531","c":"0.25980","Q":"113","o":"0.26270","h":"0.27190","l":"0.25590","v":"142488749","q":"37803477.10260","O":1702931820000,"C":1703018247336,"F":318115460,"L":318317340,"n":201880},{"e":"24hrTicker","E":1703018247776,"s":"TRBUSDT","p":"25.037","P":"21.840","w":"131.860","c":"139.677","Q":"0.3","o":"114.640","h":"143.900","l":"113.600","v":"3955845.0","q":"521616257.947","O":1702931820000,"C":1703018247775,"F":417041483,"L":419226886,"n":2185249},{"e":"24hrTicker","E":1703018247513,"s":"ACEUSDT","p":"0.108200","P":"0.826","w":"13.544944","c":"13.211400","Q":"14.37","o":"13.103200","h":"15.131200","l":"12.402900","v":"41359842.25","q":"560216757.038015","O":1702931820000,"C":1703018247512,"F":2261106,"L":4779982,"n":2518828},{"e":"24hrTicker","E":1703018247995,"s":"KEYUSDT","p":"0.0000270","P":"0.506","w":"0.0054583","c":"0.0053660","Q":"3540","o":"0.0053390","h":"0.0056230","l":"0.0053220","v":"1658962254","q":"9055176.9144700","O":1702931820000,"C":1703018247993,"F":32127330,"L":32236546,"n":109217},{"e":"24hrTicker","E":1703018247825,"s":"SUIUSDT","p":"0.094400","P":"15.783","w":"0.658766","c":"0.692500","Q":"157.6","o":"0.598100","h":"0.719600","l":"0.596400","v":"538807943.2","q":"354948524.988570","O":1702931820000,"C":1703018247824,"F":129572611,"L":130637476,"n":1064863},{"e":"24hrTicker","E":1703018247328,"s":"AGLDUSDT","p":"0.0738000","P":"7.016","w":"1.1222224","c":"1.1257000","Q":"49","o":"1.0519000","h":"1.1936000","l":"1.0471000","v":"63230369","q":"70958539.3508000","O":1702931820000,"C":1703018247327,"F":40498492,"L":41170995,"n":672503},{"e":"24hrTicker","E":1703018247882,"s":usdtmTradablePair,"p":"412.30","P":"0.986","w":"42651.76","c":"42247.00","Q":"0.003","o":"41834.70","h":"43550.00","l":"41792.00","v":"366582.423","q":"15635385730.76","O":1702931820000,"C":1703018247880,"F":4392041494,"L":4395950440,"n":3908934},{"e":"24hrTicker","E":1703018247531,"s":"WLDUSDT","p":"-0.0475000","P":"-1.232","w":"3.9879959","c":"3.8089000","Q":"50","o":"3.8564000","h":"4.3320000","l":"3.7237000","v":"119350666","q":"475969966.2747000","O":1702931820000,"C":1703018247530,"F":183723717,"L":186154953,"n":2431230},{"e":"24hrTicker","E":1703018247595,"s":"WAVESUSDT","p":"0.1108","P":"4.876","w":"2.4490","c":"2.3833","Q":"8.1","o":"2.2725","h":"2.5775","l":"2.2658","v":"54051344.0","q":"132369622.6356","O":1702931820000,"C":1703018247593,"F":503343992,"L":504167968,"n":823975},{"e":"24hrTicker","E":1703018247943,"s":"BLZUSDT","p":"0.00441","P":"1.274","w":"0.34477","c":"0.35043","Q":"35","o":"0.34602","h":"0.35844","l":"0.33146","v":"224686045","q":"77465133.09517","O":1702931820000,"C":1703018247942,"F":301286442,"L":301919432,"n":632991},{"e":"24hrTicker","E":1703018248027,"s":"ALGOUSDT","p":"0.0044","P":"2.329","w":"0.1982","c":"0.1933","Q":"1724.4","o":"0.1889","h":"0.2053","l":"0.1883","v":"418107041.7","q":"82860752.3534","O":1702931820000,"C":1703018248025,"F":317274252,"L":317530189,"n":255937},{"e":"24hrTicker","E":1703018247795,"s":"LUNA2USDT","p":"0.0849000","P":"9.610","w":"0.9622720","c":"0.9684000","Q":"91","o":"0.8835000","h":"1.0234000","l":"0.8800000","v":"132211955","q":"127223857.1990000","O":1702931820000,"C":1703018247793,"F":143814989,"L":144504341,"n":689350},{"e":"24hrTicker","E":1703018247557,"s":"DOGEUSDT","p":"-0.000290","P":"-0.320","w":"0.091710","c":"0.090210","Q":"1211","o":"0.090500","h":"0.093550","l":"0.089300","v":"4695249277","q":"430603554.425970","O":1702931820000,"C":1703018247556,"F":1408300026,"L":1409042131,"n":742103},{"e":"24hrTicker","E":1703018247578,"s":"SUSHIUSDT","p":"0.0024","P":"0.217","w":"1.1263","c":"1.1097","Q":"34","o":"1.1073","h":"1.1479","l":"1.0921","v":"34830643","q":"39229338.9293","O":1702931820000,"C":1703018247576,"F":389676753,"L":389892337,"n":215584},{"e":"24hrTicker","E":1703018247636,"s":"ROSEUSDT","p":"0.00859","P":"9.826","w":"0.09344","c":"0.09601","Q":"300","o":"0.08742","h":"0.09842","l":"0.08724","v":"768803655","q":"71837497.60153","O":1702931820000,"C":1703018247635,"F":145874088,"L":146347778,"n":473689},{"e":"24hrTicker","E":1703018247446,"s":"CTKUSDT","p":"0.05240","P":"6.933","w":"0.76993","c":"0.80820","Q":"16","o":"0.75580","h":"0.81760","l":"0.73560","v":"39275735","q":"30239750.04250","O":1702931820000,"C":1703018247445,"F":129601557,"L":129911270,"n":309714},{"e":"24hrTicker","E":1703018247083,"s":"MATICUSDT","p":"-0.02260","P":"-2.883","w":"0.78657","c":"0.76130","Q":"11","o":"0.78390","h":"0.82380","l":"0.74930","v":"510723474","q":"401719478.20480","O":1702931820000,"C":1703018247081,"F":899425701,"L":900164133,"n":738432},{"e":"24hrTicker","E":1703018247954,"s":"INJUSDT","p":"3.554000","P":"10.740","w":"37.577625","c":"36.646000","Q":"9.3","o":"33.092000","h":"39.988000","l":"32.803000","v":"30119373.7","q":"1131814520.584100","O":1702931820000,"C":1703018247953,"F":210846748,"L":214612851,"n":3766053},{"e":"24hrTicker","E":1703018247559,"s":"OCEANUSDT","p":"0.00890","P":"1.805","w":"0.50791","c":"0.50200","Q":"147","o":"0.49310","h":"0.52090","l":"0.49170","v":"42754656","q":"21715597.51239","O":1702931820000,"C":1703018247557,"F":243729859,"L":243879437,"n":149578},{"e":"24hrTicker","E":1703018247779,"s":"UNIUSDT","p":"0.0220","P":"0.378","w":"5.9288","c":"5.8470","Q":"10","o":"5.8250","h":"6.0440","l":"5.7520","v":"11324960","q":"67143423.4300","O":1702931820000,"C":1703018247778,"F":356204442,"L":356430119,"n":225678},{"e":"24hrTicker","E":1703018247999,"s":"1000BONKUSDT","p":"-0.0004410","P":"-2.245","w":"0.0205588","c":"0.0192000","Q":"1562","o":"0.0196410","h":"0.0231060","l":"0.0188770","v":"30632634003","q":"629769968.4590968","O":1702931820000,"C":1703018247998,"F":75958362,"L":80131721,"n":4173351},{"e":"24hrTicker","E":1703018247559,"s":"ARUSDT","p":"-0.382","P":"-4.176","w":"9.030","c":"8.765","Q":"4.5","o":"9.147","h":"9.467","l":"8.571","v":"3178087.5","q":"28698158.147","O":1702931820000,"C":1703018247557,"F":143756455,"L":143985699,"n":229244},{"e":"24hrTicker","E":1703018247344,"s":"AUCTIONUSDT","p":"9.690000","P":"31.369","w":"38.302392","c":"40.580000","Q":"0.65","o":"30.890000","h":"43.400000","l":"30.650000","v":"15656989.13","q":"599700134.856300","O":1702931820000,"C":1703018247343,"F":2451094,"L":5013398,"n":2561767},{"e":"24hrTicker","E":1703018247959,"s":"XRPUSDT","p":"-0.0021","P":"-0.346","w":"0.6083","c":"0.6045","Q":"396.3","o":"0.6066","h":"0.6170","l":"0.5973","v":"744301855.7","q":"452752948.7478","O":1702931820000,"C":1703018247957,"F":1344388341,"L":1344913573,"n":525224},{"e":"24hrTicker","E":1703018247813,"s":"EGLDUSDT","p":"-0.130","P":"-0.223","w":"58.569","c":"58.070","Q":"1.1","o":"58.200","h":"60.240","l":"56.670","v":"802381.7","q":"46994956.463","O":1702931820000,"C":1703018247811,"F":235206699,"L":235456030,"n":249331},{"e":"24hrTicker","E":1703018247990,"s":"ETHUSDT","p":"-10.21","P":"-0.468","w":"2206.89","c":"2170.39","Q":"0.060","o":"2180.60","h":"2256.64","l":"2135.03","v":"3187161.031","q":"7033700225.77","O":1702931820000,"C":1703018247988,"F":3443398114,"L":3446512406,"n":3114283},{"e":"24hrTicker","E":1703018247096,"s":"PENDLEUSDT","p":"-0.0059000","P":"-0.569","w":"1.0590403","c":"1.0319000","Q":"12","o":"1.0378000","h":"1.0960000","l":"1.0120000","v":"7593669","q":"8042001.5937000","O":1702931820000,"C":1703018247095,"F":16663914,"L":16782530,"n":118617}]}`,
		"Single Market Ticker":          `{"stream": "BTCUSDT@ticker", "data": { "e": "24hrTicker", "E": 1571889248277, "s": usdtmTradablePair, "p": "0.0015", "P": "250.00", "w": "0.0018", "c": "0.0025", "Q": "10", "o": "0.0010", "h": "0.0025", "l": "0.0010", "v": "10000", "q": "18", "O": 0, "C": 1703019429985, "F": 0, "L": 18150, "n": 18151 } }`,
		"Multiple Mini Tickers":         `{"stream": "!miniTicker@arr","data":[{"e":"24hrMiniTicker","E":1703019429455,"s":"BICOUSDT","c":"0.3667000","o":"0.3792000","h":"0.3892000","l":"0.3639000","v":"28768370","q":"10779000.9922000"},{"e":"24hrMiniTicker","E":1703019429985,"s":"API3USDT","c":"1.6834","o":"1.7326","h":"1.8406","l":"1.6699","v":"12371516.4","q":"21642153.0574"},{"e":"24hrMiniTicker","E":1703019429111,"s":"ICPUSDT","c":"9.414000","o":"10.126000","h":"10.956000","l":"9.236000","v":"34262192","q":"339148145.539000"},{"e":"24hrMiniTicker","E":1703019429945,"s":"SOLUSDT","c":"73.0930","o":"73.2180","h":"76.3840","l":"71.8000","v":"26319095","q":"1960871540.2620"}]}`,
		"Multi Asset Mode Asset":        `{"stream": "!assetIndex@arr", "data":[{ "e":"assetIndexUpdate", "E":1686749230000, "s":"ADAUSD","i":"0.27462452","b":"0.10000000","a":"0.10000000","B":"0.24716207","A":"0.30208698","q":"0.05000000","g":"0.05000000","Q":"0.26089330","G":"0.28835575"}, { "e":"assetIndexUpdate", "E":1686749230000, "s":"USDTUSD", "i":"0.99987691", "b":"0.00010000", "a":"0.00010000", "B":"0.99977692", "A":"0.99997689", "q":"0.00010000", "g":"0.00010000", "Q":"0.99977692", "G":"0.99997689" }]}`,
		"Composite Index Symbol":        `{"stream": "BTCUSDT@compositeIndex", "data":{ "e":"compositeIndex", "E":1602310596000, "s":"DEFIUSDT", "p":"554.41604065", "C":"baseAsset", "c":[ { "b":"BAL", "q":"USDT", "w":"1.04884844", "W":"0.01457800", "i":"24.33521021" }, { "b":"BAND", "q":"USDT" , "w":"3.53782729", "W":"0.03935200", "i":"7.26420084" } ] } }`,
		"Diff Book Depth Stream":        `{"stream": "BTCUSDT@depth@500ms", "data": { "e": "depthUpdate", "E": 1571889248277, "T": 1571889248276, "s": usdtmTradablePair, "U": 157, "u": 160, "pu": 149, "b": [ [ "0.0024", "10" ] ], "a": [ [ "0.0026", "100" ] ] } }`,
		"Partial Book Depth Stream":     `{"stream": "BTCUSDT@depth5", "data":{ "e": "depthUpdate", "E": 1571889248277, "T": 1571889248276, "s": usdtmTradablePair, "U": 390497796, "u": 390497878, "pu": 390497794, "b": [ [ "7403.89", "0.002" ], [ "7403.90", "3.906" ], [ "7404.00", "1.428" ], [ "7404.85", "5.239" ], [ "7405.43", "2.562" ] ], "a": [ [ "7405.96", "3.340" ], [ "7406.63", "4.525" ], [ "7407.08", "2.475" ], [ "7407.15", "4.800" ], [ "7407.20","0.175"]]}}`,
		"Individual Symbol Mini Ticker": `{"stream": "BTCUSDT@miniTicker", "data": { "e": "24hrMiniTicker", "E": 1571889248277, "s": usdtmTradablePair, "c": "0.0025", "o": "0.0010", "h": "0.0025", "l": "0.0010", "v": "10000", "q": "18"}}`,
	} {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			err := e.wsHandleFuturesData(t.Context(), []byte(v), asset.USDTMarginedFutures)
			assert.NoError(t, err)
		})
	}
}

func TestListSubscriptions(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	conn, err := e.Websocket.GetConnection(asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotNil(t, conn)

	if !e.Websocket.IsConnected() {
		err = e.WsUFuturesConnect(t.Context(), conn)
		require.NoError(t, err)
	}
	result, err := e.ListSubscriptions(t.Context(), conn)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetProperty(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	conn, err := e.Websocket.GetConnection(asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotNil(t, conn)

	if !e.Websocket.IsConnected() {
		err = e.WsUFuturesConnect(t.Context(), conn)
		require.NoError(t, err)
	}

	err = e.SetProperty(t.Context(), conn, "combined", true)
	require.NoError(t, err)
}

func TestGetWsOrderbook(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetWsOrderbook(&OrderBookDataRequestParams{Symbol: usdtmTradablePair, Limit: 1000})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsMostRecentTrades(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetWsMostRecentTrades(&RecentTradeRequestParams{
		Symbol: usdtmTradablePair,
		Limit:  15,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsAggregatedTrades(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetWsAggregatedTrades(&WsAggregateTradeRequestParams{
		Symbol: usdtmTradablePair,
		Limit:  5,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsCandlestick(&KlinesRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &KlinesRequestParams{Timezone: "GMT+2"}
	_, err = e.GetWsCandlestick(arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = spotTradablePair
	_, err = e.GetWsCandlestick(arg)
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	if mockTests {
		t.SkipNow()
	}
	startTime, endTime := getTime()
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetWsCandlestick(&KlinesRequestParams{
		Symbol:    usdtmTradablePair,
		Interval:  kline.FiveMin.Short(),
		Limit:     24,
		StartTime: startTime,
		EndTime:   endTime,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsOptimizedCandlestick(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	startTime, endTime := getTime()
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetWsOptimizedCandlestick(&KlinesRequestParams{
		Symbol:    usdtmTradablePair,
		Interval:  kline.FiveMin.Short(),
		Limit:     24,
		StartTime: startTime,
		EndTime:   endTime,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrenctAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsCurrenctAveragePrice(currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	if mockTests {
		t.SkipNow()
	}
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetWsCurrenctAveragePrice(usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWs24HourPriceChanges(t *testing.T) {
	t.Parallel()
	_, err := e.GetWs24HourPriceChanges(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.GetWs24HourPriceChanges(&PriceChangeRequestParam{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	if mockTests {
		t.SkipNow()
	}
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetWs24HourPriceChanges(&PriceChangeRequestParam{Symbols: []currency.Pair{usdtmTradablePair}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsTradingDayTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsTradingDayTickers(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.GetWsTradingDayTickers(&PriceChangeRequestParam{Timezone: "GMT+3"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	if mockTests {
		t.SkipNow()
	}
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetWsTradingDayTickers(&PriceChangeRequestParam{
		Symbols: []currency.Pair{usdtmTradablePair},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbolPriceTicker(currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	if mockTests {
		t.SkipNow()
	}
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetSymbolPriceTicker(usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsSymbolOrderbookTicker([]currency.Pair{currency.EMPTYPAIR})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	if mockTests {
		t.SkipNow()
	}
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetWsSymbolOrderbookTicker([]currency.Pair{usdtmTradablePair})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetWsSymbolOrderbookTicker([]currency.Pair{
		usdtmTradablePair,
		currency.NewPair(currency.ETH, currency.USDT),
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuerySessionStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetQuerySessionStatus()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLogOutOfSession(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetLogOutOfSession()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsPlaceNewOrder(&TradeOrderRequestParam{
		Symbol:      usdtmTradablePair,
		Side:        order.Sell.String(),
		OrderType:   order.Limit.String(),
		TimeInForce: "GTC",
		Price:       1234,
		Quantity:    1,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestValidatePlaceNewOrderRequest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	err := e.ValidatePlaceNewOrderRequest(&TradeOrderRequestParam{
		Symbol:      usdtmTradablePair,
		Side:        order.Sell.String(),
		OrderType:   order.Limit.String(),
		TimeInForce: "GTC",
		Price:       1234,
		Quantity:    1,
	})
	require.NoError(t, err)
}

func TestWsQueryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsQueryOrder(&QueryOrderParam{
		Symbol:  usdtmTradablePair,
		OrderID: 12345,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSignRequest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	_, signature, err := e.SignRequest(map[string]interface{}{
		"name": "nameValue",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, signature, "unexpected signature")
}

func TestWsCancelAndReplaceTradeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsCancelAndReplaceTradeOrder(&WsCancelAndReplaceParam{
		Symbol:                    usdtmTradablePair,
		CancelReplaceMode:         "ALLOW_FAILURE",
		CancelOriginClientOrderID: "4d96324ff9d44481926157",
		Side:                      order.Sell.String(),
		OrderType:                 order.Limit.String(),
		TimeInForce:               "GTC",
		Price:                     23416.10000000,
		Quantity:                  0.00847000,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCurrentOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsCurrentOpenOrders(usdtmTradablePair, 6000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsCancelOpenOrders(usdtmTradablePair, 6000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPlaceOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsPlaceOCOOrder(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &PlaceOCOOrderParam{StopLimitTimeInForce: "GTC"}
	_, err = e.WsPlaceOCOOrder(arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = usdtmTradablePair
	_, err = e.WsPlaceOCOOrder(arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.WsPlaceOCOOrder(arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsPlaceOCOOrder(&PlaceOCOOrderParam{
		Symbol:               usdtmTradablePair,
		Side:                 order.Sell.String(),
		Price:                23420.00000000,
		Quantity:             0.00650000,
		StopPrice:            23410.00000000,
		StopLimitPrice:       23405.00000000,
		StopLimitTimeInForce: "GTC",
		NewOrderRespType:     "RESULT",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsQueryOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsQueryOCOOrder("", 0, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsQueryOCOOrder("123456788", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelOCOOrder(currency.EMPTYPAIR, "someID", "12354", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WsCancelOCOOrder(spotTradablePair, "", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsCancelOCOOrder(
		usdtmTradablePair, "someID", "12354", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCurrentOpenOCOOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsCurrentOpenOCOOrders(0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPlaceNewSOROrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsPlaceNewSOROrder(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &WsOSRPlaceOrderParams{TimeInForce: "GTC"}
	_, err = e.WsPlaceNewSOROrder(arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = spotTradablePair
	_, err = e.WsPlaceNewSOROrder(arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = e.WsPlaceNewSOROrder(arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = e.WsPlaceNewSOROrder(arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsPlaceNewSOROrder(&WsOSRPlaceOrderParams{
		Symbol:      usdtmTradablePair,
		Side:        "BUY",
		OrderType:   order.Limit.String(),
		Quantity:    0.5,
		TimeInForce: "GTC",
		Price:       31000,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsTestNewOrderUsingSOR(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	err := e.WsTestNewOrderUsingSOR(&WsOSRPlaceOrderParams{
		Symbol:      usdtmTradablePair,
		Side:        "BUY",
		OrderType:   order.Limit.String(),
		Quantity:    0.5,
		TimeInForce: "GTC",
		Price:       31000,
	})
	require.NoError(t, err)
}

func TestToMap(t *testing.T) {
	t.Parallel()
	input := &struct {
		Zebiba bool   `json:"zebiba"`
		Value  int64  `json:"value"`
		Abebe  string `json:"abebe"`
		Name   string `json:"name"`
	}{
		Name:  "theName",
		Value: 347,
	}
	result, err := e.ToMap(input)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSortingTest(t *testing.T) {
	params := map[string]any{"apiKey": "wwhj3r3amR", "signature": "f89c6e5c0b", "timestamp": 1704873175325, "symbol": usdtmTradablePair, "startTime": 1704009175325, "endTime": 1704873175325, "limit": 5}
	sortedKeys := []string{"apiKey", "endTime", "limit", "signature", "startTime", "symbol", "timestamp"}
	keys := SortMap(params)
	require.Len(t, keys, len(sortedKeys), "unexptected keys length")
	for a := range keys {
		require.Equal(t, keys[a], sortedKeys[a])
	}
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.GetWsAccountInfo(0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsQueryAccountOrderRateLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsQueryAccountOrderRateLimits(0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsQueryAccountOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.WsQueryAccountOrderHistory(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.WsQueryAccountOrderHistory(&AccountOrderRequestParam{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsQueryAccountOrderHistory(&AccountOrderRequestParam{
		Symbol:    usdtmTradablePair,
		StartTime: time.Now().Add(-time.Hour * 24 * 10).UnixMilli(),
		EndTime:   time.Now().Add(-time.Hour * 6).UnixMilli(),
		Limit:     5,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsQueryAccountOCOOrderHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.WsQueryAccountOCOOrderHistory(0, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsQueryAccountOCOOrderHistory(0, 0, 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAccountTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.WsAccountTradeHistory(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.WsAccountTradeHistory(&AccountOrderRequestParam{OrderID: 1234})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsAccountTradeHistory(&AccountOrderRequestParam{Symbol: usdtmTradablePair, OrderID: 1234})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAccountPreventedMatches(t *testing.T) {
	t.Parallel()
	_, err := e.WsAccountPreventedMatches(currency.EMPTYPAIR, 1223456, 0, 0, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WsAccountPreventedMatches(spotTradablePair, 0, 0, 0, 0, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsAccountPreventedMatches(usdtmTradablePair, 1223456, 0, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAccountAllocation(t *testing.T) {
	t.Parallel()
	_, err := e.WsAccountAllocation(currency.EMPTYPAIR, time.Time{}, time.Now(), 0, 0, 0, 19)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsAccountAllocation(spotTradablePair, time.Time{}, time.Now(), 0, 0, 0, 19)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAccountCommissionRates(t *testing.T) {
	t.Parallel()
	_, err := e.WsAccountCommissionRates(currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsAccountCommissionRates(spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsStartUserDataStream(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := e.WsStartUserDataStream()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPingUserDataStream(t *testing.T) {
	t.Parallel()
	err := e.WsPingUserDataStream("")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	err = e.WsPingUserDataStream("xs0mRXdAKlIPDRFrlPcw0qI41Eh3ixNntmymGyhrhgqo7L6FuLaWArTD7RLP")
	require.NoError(t, err)
}

func TestWsStopUserDataStream(t *testing.T) {
	t.Parallel()
	err := e.WsStopUserDataStream("")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if !e.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	err = e.WsStopUserDataStream("xs0mRXdAKlIPDRFrlPcw0qI41Eh3ixNntmymGyhrhgqo7L6FuLaWArTD7RLP")
	require.NoError(t, err)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.Spot,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result)

	result, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.NewCode("BTCUSD").Item,
		Quote: currency.PERP.Item,
		Asset: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestSystemStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDailyAccountSnapshot(t *testing.T) {
	t.Parallel()
	_, err := e.GetDailyAccountSnapshot(t.Context(), "", time.Time{}, time.Now(), 0)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDailyAccountSnapshot(t.Context(), "SPOT", time.Time{}, time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDisableFastWithdrawalSwitch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.DisableFastWithdrawalSwitch(t.Context())
	assert.NoError(t, err)
}

func TestEnableFastWithdrawalSwitch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.EnableFastWithdrawalSwitch(t.Context())
	assert.NoError(t, err)
}

func TestGetAccountStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTradingAPIStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountTradingAPIStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDustLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDustLog(t.Context(), "MARGIN", time.Time{}, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetExchangeServerTime(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccount(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountTradeList(t.Context(), currency.EMPTYPAIR, "", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := getTime()
	_, err = e.GetAccountTradeList(t.Context(), assetToTradablePairMap[asset.Margin], "", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountTradeList(t.Context(), assetToTradablePairMap[asset.Margin], "", startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentOrderCountUsage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrentOrderCountUsage(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPreventedMatches(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreventedMatches(t.Context(), currency.EMPTYPAIR, 0, 12, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetPreventedMatches(t.Context(), usdtmTradablePair, 0, 0, 0, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPreventedMatches(t.Context(), usdtmTradablePair, 0, 12, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllocations(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllocations(t.Context(), currency.EMPTYPAIR, time.Time{}, time.Time{}, 10, 10, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := getTime()
	_, err = e.GetAllocations(t.Context(), usdtmTradablePair, endTime, startTime, 10, 10, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllocations(t.Context(), usdtmTradablePair, startTime, endTime, 10, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetCommissionRates(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCommissionRates(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountBorrowRepay(t *testing.T) {
	t.Parallel()
	_, err := e.MarginAccountBorrowRepay(t.Context(), currency.ETH, currency.EMPTYPAIR, "BORROW", false, 0.1234)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.MarginAccountBorrowRepay(t.Context(), currency.EMPTYCODE, usdtmTradablePair, "BORROW", false, 0.1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.MarginAccountBorrowRepay(t.Context(), currency.ETH, usdtmTradablePair, "", false, 0.1234)
	require.ErrorIs(t, err, errLendingTypeRequired)
	_, err = e.MarginAccountBorrowRepay(t.Context(), currency.ETH, usdtmTradablePair, "BORROW", false, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.MarginAccountBorrowRepay(t.Context(), currency.ETH, usdtmTradablePair, "BORROW", false, 0.1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowOrRepayRecordsInMarginAccount(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBorrowOrRepayRecordsInMarginAccount(t.Context(), currency.LTC, "", "REPAY", 0, 10, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBorrowOrRepayRecordsInMarginAccount(t.Context(), currency.LTC, "", "REPAY", 0, 10, 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllMarginAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllMarginAssets(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCrossMarginPairs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllCrossMarginPairs(t.Context(), assetToTradablePairMap[asset.Margin])
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginPriceIndex(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginPriceIndex(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginPriceIndex(t.Context(), assetToTradablePairMap[asset.Margin])
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPostMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PostMarginAccountOrder(t.Context(), &MarginAccountOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &MarginAccountOrderParam{AutoRepayAtCancel: true}
	_, err = e.PostMarginAccountOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = usdtmTradablePair
	_, err = e.PostMarginAccountOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy.String()
	_, err = e.PostMarginAccountOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PostMarginAccountOrder(t.Context(), &MarginAccountOrderParam{
		Symbol:    usdtmTradablePair,
		Side:      order.Buy.String(),
		OrderType: order.Limit.String(),
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginAccountOrder(t.Context(), currency.EMPTYPAIR, "", "", true, 12314234)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelMarginAccountOrder(t.Context(), usdtmTradablePair, "", "", true, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelMarginAccountOrder(t.Context(), usdtmTradablePair, "", "", true, 12314234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountCancelAllOpenOrdersOnSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.MarginAccountCancelAllOpenOrdersOnSymbol(t.Context(), currency.EMPTYPAIR, true)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.MarginAccountCancelAllOpenOrdersOnSymbol(t.Context(), usdtmTradablePair, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUnmarshalJSONForAssetIndex(t *testing.T) {
	t.Parallel()
	var resp *AssetIndexResponse
	data := [][]byte{
		[]byte(`{ "symbol": "ADAUSD", "time": 1635740268004, "index": "1.92957370", "bidBuffer": "0.10000000", "askBuffer": "0.10000000", "bidRate": "1.73661633", "askRate": "2.12253107", "autoExchangeBidBuffer": "0.05000000", "autoExchangeAskBuffer": "0.05000000", "autoExchangeBidRate": "1.83309501", "autoExchangeAskRate": "2.02605238" }`),
		[]byte(`[ { "symbol": "ADAUSD", "time": 1635740268004, "index": "1.92957370", "bidBuffer": "0.10000000", "askBuffer": "0.10000000", "bidRate": "1.73661633", "askRate": "2.12253107", "autoExchangeBidBuffer": "0.05000000", "autoExchangeAskBuffer": "0.05000000", "autoExchangeBidRate": "1.83309501", "autoExchangeAskRate": "2.02605238" } ]`),
	}
	err := json.Unmarshal(data[0], &resp)
	require.NoError(t, err)
	err = json.Unmarshal(data[1], &resp)
	assert.NoError(t, err)
}

func TestChangePositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.ChangePositionMode(t.Context(), false)
	assert.NoError(t, err)
}

func TestGetCurrentPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrentPositionMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ---------------------------  European Option Endpoints test -----------------------------------

func TestCheckEOptionsServerTime(t *testing.T) {
	t.Parallel()
	serverTime, err := e.CheckEOptionsServerTime(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, serverTime)
}

func TestGetEOptionsOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetEOptionsOrderbook(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetEOptionsOrderbook(t.Context(), optionsTradablePair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetEOptionsRecentTrades(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetEOptionsRecentTrades(t.Context(), optionsTradablePair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetEOptionsTradeHistory(t.Context(), currency.EMPTYPAIR, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetEOptionsTradeHistory(t.Context(), optionsTradablePair, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsCandlesticks(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetEOptionsCandlesticks(t.Context(), currency.EMPTYPAIR, kline.OneDay, startTime, endTime, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetEOptionsCandlesticks(t.Context(), optionsTradablePair, 0, startTime, endTime, 1000)
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	result, err := e.GetEOptionsCandlesticks(t.Context(), optionsTradablePair, kline.OneDay, startTime, endTime, 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetOptionMarkPrice(t.Context(), optionsTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptions24hrTickerPriceChangeStatistics(t *testing.T) {
	t.Parallel()
	result, err := e.GetEOptions24hrTickerPriceChangeStatistics(t.Context(), optionsTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetEOptionsSymbolPriceTicker(t.Context(), "")
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	result, err := e.GetEOptionsSymbolPriceTicker(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsHistoricalExerciseRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetEOptionsHistoricalExerciseRecords(t.Context(), "BTCUSDT", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetEOptionsHistoricalExerciseRecords(t.Context(), "BTCUSDT", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsOpenInterests(t *testing.T) {
	t.Parallel()
	expTime := time.UnixMilli(1744633637579)
	if !mockTests {
		expTime = time.Now().Add(time.Hour * 24)
	}
	_, err := e.GetEOptionsOpenInterests(t.Context(), currency.EMPTYCODE, expTime)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetEOptionsOpenInterests(t.Context(), currency.ETH, time.Time{})
	require.ErrorIs(t, err, errExpirationTimeRequired)

	result, err := e.GetEOptionsOpenInterests(t.Context(), currency.ETH, expTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOptionsAccountInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOptionsOrder(t *testing.T) {
	t.Parallel()
	arg := &OptionsOrderParams{}
	_, err := e.NewOptionsOrder(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.PostOnly = true
	_, err = e.NewOptionsOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = optionsTradablePair
	_, err = e.NewOptionsOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.NewOptionsOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = e.NewOptionsOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewOptionsOrder(t.Context(), &OptionsOrderParams{
		Symbol:                  optionsTradablePair,
		Side:                    order.Sell.String(),
		OrderType:               order.Limit.String(),
		Amount:                  0.00001,
		Price:                   0.00001,
		ReduceOnly:              false,
		PostOnly:                true,
		NewOrderResponseType:    "RESULT",
		ClientOrderID:           "the-client-order-id",
		IsMarketMakerProtection: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceEOptionsOrder(t *testing.T) {
	t.Parallel()
	arg := &OptionsOrderParams{}
	_, err := e.PlaceBatchEOptionsOrder(t.Context(), []*OptionsOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.PlaceBatchEOptionsOrder(t.Context(), []*OptionsOrderParams{arg})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.PostOnly = true
	_, err = e.PlaceBatchEOptionsOrder(t.Context(), []*OptionsOrderParams{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.Pair{Base: currency.BTC, Delimiter: currency.DashDelimiter, Quote: currency.NewCode("200730-9000-C")}
	_, err = e.PlaceBatchEOptionsOrder(t.Context(), []*OptionsOrderParams{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.PlaceBatchEOptionsOrder(t.Context(), []*OptionsOrderParams{arg})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = e.PlaceBatchEOptionsOrder(t.Context(), []*OptionsOrderParams{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceBatchEOptionsOrder(t.Context(), []*OptionsOrderParams{
		{
			Symbol:                  optionsTradablePair,
			Side:                    order.Sell.String(),
			OrderType:               order.Limit.String(),
			Amount:                  0.00001,
			Price:                   0.00001,
			ReduceOnly:              false,
			PostOnly:                true,
			NewOrderResponseType:    "RESULT",
			ClientOrderID:           "the-client-order-id",
			IsMarketMakerProtection: true,
		}, {
			Symbol:                  optionsTradablePair,
			Side:                    "Buy",
			OrderType:               "Market",
			Amount:                  0.00001,
			PostOnly:                true,
			NewOrderResponseType:    "RESULT",
			ClientOrderID:           "the-client-order-id-2",
			IsMarketMakerProtection: true,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSingleEOptionsOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleEOptionsOrder(t.Context(), currency.EMPTYPAIR, "", 4611875134427365377)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetSingleEOptionsOrder(t.Context(), optionsTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSingleEOptionsOrder(t.Context(), optionsTradablePair, "", 4611875134427365377)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOptionsOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOptionsOrder(t.Context(), currency.EMPTYPAIR, "213123", "4611875134427365377")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelOptionsOrder(t.Context(), optionsTradablePair, "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelOptionsOrder(t.Context(), optionsTradablePair, "213123", "4611875134427365377")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelBatchOptionsOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOptionsOrders(t.Context(), currency.EMPTYPAIR, []int64{4611875134427365377}, []string{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelBatchOptionsOrders(t.Context(), optionsTradablePair, []int64{}, []string{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelBatchOptionsOrders(t.Context(), optionsTradablePair, []int64{4611875134427365377}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOptionOrdersOnSpecificSymbol(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.CancelAllOptionOrdersOnSpecificSymbol(t.Context(), optionsTradablePair)
	assert.NoError(t, err)
}

func TestCancelAllOptionsOrdersByUnderlying(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOptionsOrdersByUnderlying(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentOpenOptionsOrders(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetCurrentOpenOptionsOrders(t.Context(), optionsTradablePair, endTime, startTime, 4611875134427365377, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	results, err := e.GetCurrentOpenOptionsOrders(t.Context(), optionsTradablePair, startTime, endTime, 4611875134427365377, 0)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestGetOptionsOrdersHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetOptionsOrdersHistory(t.Context(), optionsTradablePair, endTime, startTime, 4611875134427365377, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	results, err := e.GetOptionsOrdersHistory(t.Context(), optionsTradablePair, startTime, endTime, 4611875134427365377, 10)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestGetOptionPositionInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOptionPositionInformation(t.Context(), optionsTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsAccountTradeList(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetEOptionsAccountTradeList(t.Context(), optionsTradablePair, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetEOptionsAccountTradeList(t.Context(), optionsTradablePair, 0, 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserOptionsExerciseRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUserOptionsExerciseRecord(t.Context(), optionsTradablePair, endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserOptionsExerciseRecord(t.Context(), optionsTradablePair, startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingFlow(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountFundingFlow(t.Context(), currency.EMPTYCODE, 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	startTime, endTime := getTime()
	_, err = e.GetAccountFundingFlow(t.Context(), currency.EMPTYCODE, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingFlow(t.Context(), currency.USDT, 0, 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDownloadIDForOptionTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetDownloadIDForOptionTransactionHistory(t.Context(), endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDownloadIDForOptionTransactionHistory(t.Context(), startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionTransactionHistoryDownloadLinkByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionTransactionHistoryDownloadLinkByID(t.Context(), "")
	require.ErrorIs(t, err, errDownloadIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOptionTransactionHistoryDownloadLinkByID(t.Context(), "download-id")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionMarginAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOptionMarginAccountInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetMarketMakerProtectionConfig(t *testing.T) {
	t.Parallel()
	_, err := e.SetOptionsMarketMakerProtectionConfig(t.Context(), &MarketMakerProtectionConfig{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.SetOptionsMarketMakerProtectionConfig(t.Context(), &MarketMakerProtectionConfig{
		WindowTimeInMilliseconds: 3000,
		FrozenTimeInMilliseconds: 300000,
		QuantityLimit:            1.5,
		NetDeltaLimit:            1.5,
	})
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetOptionsMarketMakerProtectionConfig(t.Context(), &MarketMakerProtectionConfig{
		Underlying:               "BTCUSDT",
		WindowTimeInMilliseconds: 3000,
		FrozenTimeInMilliseconds: 300000,
		QuantityLimit:            1.5,
		NetDeltaLimit:            1.5,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsMarketMakerProtection(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsMarketMakerProtection(t.Context(), "")
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOptionsMarketMakerProtection(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetMarketMaketProtection(t *testing.T) {
	t.Parallel()
	_, err := e.ResetMarketMaketProtection(t.Context(), "")
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ResetMarketMaketProtection(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetOptionsAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.SetOptionsAutoCancelAllOpenOrders(t.Context(), "", 30000)
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.SetOptionsAutoCancelAllOpenOrders(t.Context(), "BTCUSDT", 30000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoCancelAllOpenOrdersConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAutoCancelAllOpenOrdersConfig(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsAutoCancelAllOpenOrdersHeartbeat(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsAutoCancelAllOpenOrdersHeartbeat(t.Context(), []string{})
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetOptionsAutoCancelAllOpenOrdersHeartbeat(t.Context(), []string{"ETHUSDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsOptionsConnect(t *testing.T) {
	t.Parallel()
	conn, err := e.Websocket.GetConnection(asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotNil(t, conn)

	err = e.WsOptionsConnect(t.Context(), conn)
	assert.NoError(t, err)
}

func TestGetOptionsExchangeInformation(t *testing.T) {
	t.Parallel()
	exchangeinformation, err := e.GetOptionsExchangeInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, exchangeinformation)
}

// ---------------------------------------   Portfolio Margin  ---------------------------------------------

func TestNewUMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewUMOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &UMOrderParam{ReduceOnly: true}
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = usdtmTradablePair
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, errTimeInForceRequired)

	arg.TimeInForce = "GTC"
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Quantity = 1.
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.Price = 1234
	arg.OrderType = "market"
	arg.Quantity = 0
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.OrderType = "stop"
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewUMOrder(t.Context(), &UMOrderParam{
		Symbol:       usdtmTradablePair,
		Side:         "BUY",
		PositionSide: "BOTH",
		OrderType:    "market",
		Quantity:     1,
		ReduceOnly:   false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewCMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewCMOrder(t.Context(), &UMOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &UMOrderParam{
		ReduceOnly: true,
	}
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = usdtmTradablePair
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "OCO"
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.OrderType = "MARKET"
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.OrderType = order.Limit.String()
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, errTimeInForceRequired)

	arg.TimeInForce = "GTC"
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Quantity = .1
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewCMOrder(t.Context(), &UMOrderParam{
		Symbol:       usdtmTradablePair,
		Side:         "BUY",
		PositionSide: "BOTH",
		OrderType:    "limit",
		Quantity:     1,
		ReduceOnly:   false,
		TimeInForce:  "GTD",
		Price:        000.1,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewMarginOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewMarginOrder(t.Context(), &MarginOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &MarginOrderParam{
		TimeInForce: "GTC",
	}
	_, err = e.NewMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = spotTradablePair.String()
	_, err = e.NewMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.NewMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewMarginOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.MarginAccountBorrow(t.Context(), currency.EMPTYCODE, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.MarginAccountBorrow(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.MarginAccountBorrow(t.Context(), currency.USDT, 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountRepay(t *testing.T) {
	t.Parallel()
	_, err := e.MarginAccountRepay(t.Context(), currency.EMPTYCODE, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.MarginAccountRepay(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.MarginAccountRepay(t.Context(), currency.USDT, 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountNewOCO(t *testing.T) {
	t.Parallel()
	_, err := e.MarginAccountNewOCO(t.Context(), &OCOOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &OCOOrderParam{
		TrailingDelta: 1,
	}
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = usdtmTradablePair
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Buy"
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 0.1
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.Price = 0.001
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewOCOOrder(t.Context(), &OCOOrderParam{
		Symbol:             usdtmTradablePair,
		ListClientOrderID:  "1231231231231",
		Side:               "Buy",
		Amount:             0.1,
		LimitClientOrderID: "3423423",
		Price:              0.001,
		StopPrice:          1234.21,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOCOOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.NewOCOOrderList(t.Context(), &OCOOrderListParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &OCOOrderListParams{
		AboveTimeInForce: "GTC",
	}
	_, err = e.NewOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = "LTCBTC"
	_, err = e.NewOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.NewOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Quantity = 1
	_, err = e.NewOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.AboveType = "STOP_LOSS_LIMIT"
	_, err = e.NewOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.NewOCOOrderList(t.Context(), &OCOOrderListParams{
		Symbol:     "LTCBTC",
		Side:       order.Sell.String(),
		Quantity:   1,
		AbovePrice: 100,
		AboveType:  "STOP_LOSS_LIMIT",
		BelowType:  "LIMIT_MAKER",
		BelowPrice: 25,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewUMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewUMConditionalOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &ConditionalOrderParam{PriceProtect: true}
	_, err = e.NewUMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = usdtmTradablePair
	_, err = e.NewUMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.NewUMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, errStrategyTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewUMConditionalOrder(t.Context(), &ConditionalOrderParam{
		Symbol:       usdtmTradablePair,
		Side:         order.Sell.String(),
		PositionSide: "SHORT",
		StrategyType: "STOP_MARKET",
		PriceProtect: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewCMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewCMConditionalOrder(t.Context(), &ConditionalOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &ConditionalOrderParam{
		PositionSide: "LONG",
	}
	_, err = e.NewCMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = coinmTradablePair
	_, err = e.NewCMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Buy"
	_, err = e.NewCMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, errStrategyTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewCMConditionalOrder(t.Context(), &ConditionalOrderParam{
		Symbol:       coinmTradablePair,
		Side:         "Buy",
		PositionSide: "LONG",
		StrategyType: "TAKE_PROFIT",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelUMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelUMOrder(t.Context(), currency.EMPTYPAIR, "", 1234132)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelUMOrder(t.Context(), usdtmTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelUMOrder(t.Context(), usdtmTradablePair, "", 1234132)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelCMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCMOrder(t.Context(), currency.EMPTYPAIR, "", 21321312)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.CancelCMOrder(t.Context(), usdtmTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelCMOrder(t.Context(), usdtmTradablePair, "", 21321312)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllUMOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllUMOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllUMOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 200, result.Code)
}

func TestCancelAllCMOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllCMOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllCMOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPMCancelMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PMCancelMarginAccountOrder(t.Context(), currency.EMPTYPAIR, "", 12314)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.PMCancelMarginAccountOrder(t.Context(), assetToTradablePairMap[asset.Margin], "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PMCancelMarginAccountOrder(t.Context(), assetToTradablePairMap[asset.Margin], "", 12314)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllMarginOpenOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllMarginOpenOrdersBySymbol(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllMarginOpenOrdersBySymbol(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMarginAccountOCOOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginAccountOCOOrders(t.Context(), currency.EMPTYPAIR, "", "", 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelMarginAccountOCOOrders(t.Context(), assetToTradablePairMap[asset.Margin], "", "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelUMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelUMConditionalOrder(t.Context(), currency.EMPTYPAIR, "", 2000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.CancelUMConditionalOrder(t.Context(), usdtmTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelUMConditionalOrder(t.Context(), usdtmTradablePair, "", 2000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelCMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCMConditionalOrder(t.Context(), currency.EMPTYPAIR, "", 1231231)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelCMConditionalOrder(t.Context(), usdtmTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelCMConditionalOrder(t.Context(), usdtmTradablePair, "", 1231231)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllUMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllUMOpenConditionalOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllUMOpenConditionalOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllCMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllCMOpenConditionalOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllCMOpenConditionalOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetUMOrder(t.Context(), currency.EMPTYPAIR, "", 1234)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetUMOrder(t.Context(), usdtmTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUMOrder(t.Context(), usdtmTradablePair, "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetUMOpenOrder(t.Context(), currency.EMPTYPAIR, "", 1234)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetUMOpenOrder(t.Context(), usdtmTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUMOpenOrder(t.Context(), usdtmTradablePair, "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllUMOpenOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMOrders(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAllUMOrders(t.Context(), usdtmTradablePair, endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllUMOrders(t.Context(), usdtmTradablePair, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetCMOrder(t.Context(), currency.EMPTYPAIR, "", 1234)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCMOrder(t.Context(), coinmTradablePair, "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetCMOpenOrder(t.Context(), currency.EMPTYPAIR, "", 1234)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetCMOpenOrder(t.Context(), coinmTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCMOpenOrder(t.Context(), coinmTradablePair, "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllCMOpenOrders(t.Context(), currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllCMOpenOrders(t.Context(), coinmTradablePair, "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllCMOrders(t.Context(), currency.EMPTYPAIR, "", time.Time{}, time.Time{}, 0, 20)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := getTime()
	_, err = e.GetAllCMOrders(t.Context(), coinmTradablePair, "BTCUSD", endTime, startTime, 0, 20)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllCMOrders(t.Context(), coinmTradablePair, "BTCUSD", startTime, endTime, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenUMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenUMConditionalOrder(t.Context(), usdtmTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenUMConditionalOrder(t.Context(), usdtmTradablePair, "newClientStrategyId", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllUMOpenConditionalOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMConditionalOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllUMConditionalOrderHistory(t.Context(), usdtmTradablePair, "abc", 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllUMConditionalOrders(t.Context(), usdtmTradablePair, time.Time{}, time.Now(), 0, 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenCMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenCMConditionalOrder(t.Context(), coinmTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenCMConditionalOrder(t.Context(), coinmTradablePair, "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllCMOpenConditionalOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMConditionalOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllCMConditionalOrderHistory(t.Context(), usdtmTradablePair, "abc", 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllCMConditionalOrders(t.Context(), usdtmTradablePair, time.Time{}, time.Now(), 0, 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountOrder(t.Context(), currency.EMPTYPAIR, "", 12434)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetMarginAccountOrder(t.Context(), assetToTradablePairMap[asset.Margin], "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAccountOrder(t.Context(), assetToTradablePairMap[asset.Margin], "", 12434)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentMarginOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrentMarginOpenOrder(t.Context(), assetToTradablePairMap[asset.Margin])
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllMarginAccountOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllMarginAccountOrders(t.Context(), currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := getTime()
	_, err = e.GetAllMarginAccountOrders(t.Context(), currency.EMPTYPAIR, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllMarginAccountOrders(t.Context(), assetToTradablePairMap[asset.Margin], startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountOCO(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAccountOCO(t.Context(), 0, "123421-abcde")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPMMarginAccountAllOCO(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPMMarginAccountAllOCO(t.Context(), endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPMMarginAccountAllOCO(t.Context(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOpenOCO(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAccountsOpenOCO(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPMMarginAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPMMarginAccountTradeList(t.Context(), currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := getTime()
	_, err = e.GetPMMarginAccountTradeList(t.Context(), assetToTradablePairMap[asset.Margin], startTime, endTime, 0, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPMMarginAccountTradeList(t.Context(), assetToTradablePairMap[asset.Margin], startTime, endTime, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountBalance(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioMarginAccountInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginMaxBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.GetPMMarginMaxBorrow(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPMMarginMaxBorrow(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginMaxWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginMaxWithdrawal(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginMaxWithdrawal(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMPositionInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUMPositionInformation(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMPositionInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCMPositionInformation(t.Context(), currency.ETH, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeUMInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeUMInitialLeverage(t.Context(), currency.EMPTYPAIR, 29)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.ChangeUMInitialLeverage(t.Context(), usdtmTradablePair, 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeUMInitialLeverage(t.Context(), usdtmTradablePair, 29)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeCMInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeCMInitialLeverage(t.Context(), currency.EMPTYPAIR, 29)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.ChangeCMInitialLeverage(t.Context(), usdtmTradablePair, 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeCMInitialLeverage(t.Context(), usdtmTradablePair, 29)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeUMPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeUMPositionMode(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeCMPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeCMPositionMode(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMCurrentPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUMCurrentPositionMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMCurrentPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCMCurrentPositionMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetUMAccountTradeList(t.Context(), currency.EMPTYPAIR, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUMAccountTradeList(t.Context(), usdtmTradablePair, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetCMAccountTradeList(t.Context(), currency.EMPTYPAIR, "", time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCMAccountTradeList(t.Context(), coinmTradablePair, "", time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUMNotionalAndLeverageBrackets(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCMNotionalAndLeverageBrackets(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersMarginForceOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersMarginForceOrders(t.Context(), time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersUMForceOrderst(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUsersUMForceOrders(t.Context(), usdtmTradablePair, "", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersUMForceOrders(t.Context(), usdtmTradablePair, "", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersCMForceOrderst(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUsersCMForceOrders(t.Context(), usdtmTradablePair, "", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersCMForceOrders(t.Context(), usdtmTradablePair, "", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginUMTradingQuantitativeRulesIndicator(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioMarginUMTradingQuantitativeRulesIndicator(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMUserCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetUMUserCommissionRate(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUMUserCommissionRate(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMUserCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetCMUserCommissionRate(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCMUserCommissionRate(t.Context(), coinmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginLoanRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginLoanRecord(t.Context(), currency.EMPTYCODE, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginLoanRecord(t.Context(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginRepayRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginRepayRecord(t.Context(), currency.EMPTYCODE, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginRepayRecord(t.Context(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginBorrowOrLoanInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginBorrowOrLoanInterestHistory(t.Context(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginNegativeBalanceInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioMarginNegativeBalanceInterestHistory(t.Context(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundAutoCollection(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FundAutoCollection(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundCollectionByAsset(t *testing.T) {
	t.Parallel()
	_, err := e.FundCollectionByAsset(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FundCollectionByAsset(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBNBTransferClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.BNBTransferClassic(t.Context(), 0.0001, "TO_UM")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBNBTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.BNBTransfer(t.Context(), 0.0001, "TO_UM")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUMAccountDetail(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCMAccountDetail(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ChangeAutoRepayFuturesStatus(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAutoRepayFuturesStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayFuturesNegativeBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RepayFuturesNegativeBalance(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMPositionADLQuantileEstimation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUMPositionADLQuantileEstimation(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMPositionADLQuantileEstimation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCMPositionADLQuantileEstimation(t.Context(), coinmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserRateLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserRateLimits(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginAssetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginAssetIndexPrice(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioMarginAssetIndexPrice(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAdjustCrossMarginMaxLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.AdjustCrossMarginMaxLeverage(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.AdjustCrossMarginMaxLeverage(t.Context(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginTransferHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetCrossMarginTransferHistory(t.Context(), currency.ETH, "ROLL_IN", currency.EMPTYPAIR, endTime, startTime, 10, 30)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCrossMarginTransferHistory(t.Context(), currency.ETH, "ROLL_IN", currency.EMPTYPAIR, startTime, endTime, 10, 30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewMarginAccountOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewMarginAccountOCOOrder(t.Context(), &MarginOCOOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &MarginOCOOrderParam{
		IsIsolated: true,
	}
	_, err = e.NewMarginAccountOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = usdtmTradablePair
	_, err = e.NewMarginAccountOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy.String()
	_, err = e.NewMarginAccountOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Quantity = 0.000001
	_, err = e.NewMarginAccountOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.Price = 12312
	_, err = e.NewMarginAccountOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewMarginAccountOCOOrder(t.Context(), &MarginOCOOrderParam{
		Symbol:    usdtmTradablePair,
		Side:      order.Buy.String(),
		Quantity:  0.000001,
		Price:     12312,
		StopPrice: 12345,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMarginAccountOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginAccountOCOOrder(t.Context(), currency.EMPTYPAIR, "12345678", "", true, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelMarginAccountOCOOrder(t.Context(), assetToTradablePairMap[asset.Margin], "12345678", "", true, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountOCOOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAccountOCOOrder(t.Context(), assetToTradablePairMap[asset.Margin], "12345", 0, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountAllOCO(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetMarginAccountAllOCO(t.Context(), assetToTradablePairMap[asset.Margin], true, endTime, startTime, 0, 12)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAccountAllOCO(t.Context(), assetToTradablePairMap[asset.Margin], true, startTime, endTime, 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOpenOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountsOpenOCOOrder(t.Context(), true, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAccountsOpenOCOOrder(t.Context(), true, usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountTradeList(t.Context(), currency.EMPTYPAIR, true, time.Time{}, time.Time{}, 0, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := getTime()
	_, err = e.GetMarginAccountTradeList(t.Context(), currency.EMPTYPAIR, true, endTime, startTime, 0, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAccountTradeList(t.Context(), assetToTradablePairMap[asset.Margin], true, startTime, endTime, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaxBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxBorrow(t.Context(), currency.EMPTYCODE, assetToTradablePairMap[asset.Margin])
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMaxBorrow(t.Context(), currency.ETH, assetToTradablePairMap[asset.Margin])
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaxTransferOutAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxTransferOutAmount(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMaxTransferOutAmount(t.Context(), currency.ETH, currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSummaryOfMarginAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIsolatedMarginAccountInfo(t.Context(), []string{usdtmTradablePair.String()})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDisableIsolatedMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := e.DisableIsolatedMarginAccount(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.DisableIsolatedMarginAccount(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableIsolatedMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := e.EnableIsolatedMarginAccount(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.EnableIsolatedMarginAccount(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEnabledIsolatedMarginAccountLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetEnabledIsolatedMarginAccountLimit(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllIsolatedMarginSymbols(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllIsolatedMarginSymbols(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestToggleBNBBurn(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ToggleBNBBurn(t.Context(), true, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBNBBurnStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetBNBBurnStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginInterestRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginInterestRateHistory(t.Context(), currency.EMPTYCODE, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	startTime, endTime := getTime()
	_, err = e.GetMarginInterestRateHistory(t.Context(), currency.ETH, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginInterestRateHistory(t.Context(), currency.ETH, 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginFeeData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCrossMarginFeeData(t.Context(), 0, currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMaringFeeData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIsolatedMaringFeeData(t.Context(), 1, usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginTierData(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginTierData(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIsolatedMarginTierData(t.Context(), usdtmTradablePair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyMarginOrderCountUsage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrencyMarginOrderCountUsage(t.Context(), true, usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCrossMarginCollateralRatio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCrossMarginCollateralRatio(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmallLiabilityExchangeCoinList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSmallLiabilityExchangeCoinList(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginSmallLiabilityExchange(t *testing.T) {
	t.Parallel()
	_, err := e.MarginSmallLiabilityExchange(t.Context(), []string{})
	require.ErrorIs(t, err, errEmptyCurrencyCodes)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.MarginSmallLiabilityExchange(t.Context(), []string{"BTC", "ETH"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmallLiabilityExchangeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSmallLiabilityExchangeHistory(t.Context(), 0, 10, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errPageNumberRequired)
	_, err = e.GetSmallLiabilityExchangeHistory(t.Context(), 1, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errPageSizeRequired)

	startTime, endTime := getTime()
	_, err = e.GetSmallLiabilityExchangeHistory(t.Context(), 1, 10, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSmallLiabilityExchangeHistory(t.Context(), 1, 10, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFutureHourlyInterestRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFutureHourlyInterestRate(t.Context(), []string{"BTC", "ETH"}, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossOrIsolatedMarginCapitalFlow(t *testing.T) {
	t.Parallel()

	startTime, endTime := getTime()
	_, err := e.GetCrossOrIsolatedMarginCapitalFlow(t.Context(), currency.ETH, currency.EMPTYPAIR, "BORROW", endTime, startTime, 10, 20)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCrossOrIsolatedMarginCapitalFlow(t.Context(), currency.ETH, currency.EMPTYPAIR, "BORROW", startTime, endTime, 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTokensOrSymbolsDelistSchedule(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTokensOrSymbolsDelistSchedule(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAvailableInventory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAvailableInventory(t.Context(), "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMarginAvailableInventory(t.Context(), "ISOLATED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginManualLiquidiation(t *testing.T) {
	t.Parallel()
	_, err := e.MarginManualLiquidiation(t.Context(), "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.MarginManualLiquidiation(t.Context(), "ISOLATED", currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLiabilityCoinLeverageBracketInCrossMarginProMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLiabilityCoinLeverageBracketInCrossMarginProMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSimpleEarnFlexibleProductList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSimpleEarnFlexibleProductList(t.Context(), currency.BTC, 2, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSimpleEarnLockedProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSimpleEarnLockedProducts(t.Context(), currency.BTC, 2, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeToFlexibleProducts(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeToFlexibleProducts(t.Context(), "", "FUND", 1, false)
	require.ErrorIs(t, err, errProductIDRequired)
	_, err = e.SubscribeToFlexibleProducts(t.Context(), "project-id", "FUND", 0, false)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubscribeToFlexibleProducts(t.Context(), "product-id", "FUND", 1, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeToLockedProducts(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeToLockedProducts(t.Context(), "", "SPOT", 1, false)
	require.ErrorIs(t, err, errProjectIDRequired)
	_, err = e.SubscribeToLockedProducts(t.Context(), "project-id", "SPOT", 0, false)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubscribeToLockedProducts(t.Context(), "project-id", "SPOT", 1, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemFlexibleProduct(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemFlexibleProduct(t.Context(), "", "FUND", true, 0.1234)
	require.ErrorIs(t, err, errProductIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RedeemFlexibleProduct(t.Context(), "product-id", "FUND", true, 0.1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemLockedProduct(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemLockedProduct(t.Context(), 0)
	require.ErrorIs(t, err, errPositionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RedeemLockedProduct(t.Context(), 12345)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleProductPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFlexibleProductPosition(t.Context(), currency.BTC, "", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedProductPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLockedProductPosition(t.Context(), currency.ETH, "", "", 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSimpleAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.SimpleAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleSubscriptionRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFlexibleSubscriptionRecord(t.Context(), "", "", currency.ETH, endTime, startTime, 0, 12)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFlexibleSubscriptionRecord(t.Context(), "", "", currency.ETH, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedSubscriptionsRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetLockedSubscriptionsRecords(t.Context(), "", currency.ETH, startTime, endTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLockedSubscriptionsRecords(t.Context(), "", currency.ETH, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleRedemptionRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFlexibleRedemptionRecord(t.Context(), "", "1234", currency.LTC, endTime, startTime, 0, 12)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFlexibleRedemptionRecord(t.Context(), "", "1234", currency.LTC, startTime, endTime, 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedRedemptionRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetLockedRedemptionRecord(t.Context(), "", "1234", currency.LTC, endTime, startTime, 0, 12)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLockedRedemptionRecord(t.Context(), "", "1234", currency.LTC, startTime, endTime, 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleRewardHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFlexibleRewardHistory(t.Context(), "product-type", "", currency.BTC, endTime, startTime, 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFlexibleRewardHistory(t.Context(), "product-type", "", currency.BTC, startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedRewardHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetLockedRewardHistory(t.Context(), "12345", currency.BTC, endTime, startTime, 10, 40)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLockedRewardHistory(t.Context(), "12345", currency.BTC, startTime, endTime, 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetFlexibleAutoSusbcribe(t *testing.T) {
	t.Parallel()
	_, err := e.SetFlexibleAutoSusbcribe(t.Context(), "", true)
	require.ErrorIs(t, err, errProductIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetFlexibleAutoSusbcribe(t.Context(), "product-id", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLockedAutoSubscribe(t *testing.T) {
	t.Parallel()
	_, err := e.SetLockedAutoSubscribe(t.Context(), "", true)
	require.ErrorIs(t, err, errPositionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetLockedAutoSubscribe(t.Context(), "position-id", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexiblePersonalLeftQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFlexiblePersonalLeftQuota(t.Context(), "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedPersonalLeftQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLockedPersonalLeftQuota(t.Context(), "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleSubscriptionPreview(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleSubscriptionPreview(t.Context(), "", 0.0001)
	require.ErrorIs(t, err, errProductIDRequired)
	_, err = e.GetFlexibleSubscriptionPreview(t.Context(), "1234", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFlexibleSubscriptionPreview(t.Context(), "1234", 0.0001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedSubscriptionPreview(t *testing.T) {
	t.Parallel()
	_, err := e.GetLockedSubscriptionPreview(t.Context(), "", 0.1234, false)
	require.ErrorIs(t, err, errProjectIDRequired)
	_, err = e.GetLockedSubscriptionPreview(t.Context(), "12345", 0, false)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLockedSubscriptionPreview(t.Context(), "12345", 0.1234, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLockedProductRedeemOption(t *testing.T) {
	t.Parallel()
	_, err := e.SetLockedProductRedeemOption(t.Context(), "", "abcdefg")
	require.ErrorIs(t, err, errPositionIDRequired)
	_, err = e.SetLockedProductRedeemOption(t.Context(), "12345", "")
	require.ErrorIs(t, err, errRedemptionAccountRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.SetLockedProductRedeemOption(t.Context(), "12345", "abcdefg")
	assert.NoError(t, err)
}

func TestGetSimpleEarnRatehistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSimpleEarnRatehistory(t.Context(), "project-id", endTime, startTime, 0, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSimpleEarnRatehistory(t.Context(), "project-id", startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSimpleEarnCollateralRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSimpleEarnCollateralRecord(t.Context(), "project-id", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSimpleEarnCollateralRecord(t.Context(), "project-id", startTime, endTime.Add(-time.Hour*2), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDualInvestmentProductList(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualInvestmentProductList(t.Context(), "", currency.BTC, currency.ETH, 0, 0)
	require.ErrorIs(t, err, errOptionTypeRequired)
	_, err = e.GetDualInvestmentProductList(t.Context(), "CALL", currency.EMPTYCODE, currency.ETH, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetDualInvestmentProductList(t.Context(), "CALL", currency.BTC, currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetDualInvestmentProductList(t.Context(), "CALL", currency.BTC, currency.ETH, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeDualInvestmentProducts(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeDualInvestmentProducts(t.Context(), "", "order-id", "STANDARD", 0.1)
	require.ErrorIs(t, err, errProductIDRequired)
	_, err = e.SubscribeDualInvestmentProducts(t.Context(), "1234", "", "STANDARD", 0.1)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.SubscribeDualInvestmentProducts(t.Context(), "1234", "order-id", "STANDARD", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.SubscribeDualInvestmentProducts(t.Context(), "1234", "order-id", "", 1)
	require.ErrorIs(t, err, errPlanTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubscribeDualInvestmentProducts(t.Context(), "1234", "order-id", "STANDARD", 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDualInvestmentPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDualInvestmentPositions(t.Context(), "PURCHASE_FAIL", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckDualInvestmentAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CheckDualInvestmentAccounts(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAutoCompoundStatus(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeAutoCompoundStatus(t.Context(), "", "STANDARD")
	require.ErrorIs(t, err, errPositionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeAutoCompoundStatus(t.Context(), "123456789", "STANDARD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTargetAssetList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTargetAssetList(t.Context(), currency.BTC, 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTargetAssetROIData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTargetAssetROIData(t.Context(), currency.ETH, "THREE_YEAR")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllSourceAssetAndTargetAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllSourceAssetAndTargetAsset(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSourceAssetList(t *testing.T) {
	t.Parallel()
	_, err := e.GetSourceAssetList(t.Context(), currency.BTC, 123, "", "MAIN_SITE", true)
	require.ErrorIs(t, err, errUsageTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSourceAssetList(t.Context(), currency.BTC, 123, "RECURRING", "MAIN_SITE", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInvestmentPlanCreation(t *testing.T) {
	t.Parallel()
	_, err := e.InvestmentPlanCreation(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &InvestmentPlanParams{}
	_, err = e.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, errSourceTypeRequired)

	arg.SourceType = "MAIN_SITE"
	_, err = e.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, errPlanTypeRequired)

	arg.PlanType = "SINGLE"
	_, err = e.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.SubscriptionAmount = 4
	_, err = e.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubscriptionStartTime)

	arg.SubscriptionStartDay = 1
	arg.SubscriptionStartTime = 8
	_, err = e.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.SourceAsset = currency.USDT
	_, err = e.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, errPortfolioDetailRequired)

	arg.Details = []PortfolioDetail{{}}
	_, err = e.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Details = []PortfolioDetail{{TargetAsset: currency.BTC, Percentage: -1}}
	_, err = e.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPercentageAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.InvestmentPlanCreation(t.Context(), &InvestmentPlanParams{
		SourceType:            "MAIN_SITE",
		PlanType:              "SINGLE",
		SubscriptionAmount:    4,
		SubscriptionCycle:     "H4",
		SubscriptionStartTime: 8,
		SourceAsset:           currency.USDT,
		Details: []PortfolioDetail{
			{
				TargetAsset: currency.ETH,
				Percentage:  12,
			},
			{
				TargetAsset: currency.ETH,
				Percentage:  20,
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInvestmentPlanAdjustment(t *testing.T) {
	t.Parallel()
	_, err := e.InvestmentPlanAdjustment(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &AdjustInvestmentPlan{}
	_, err = e.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errPlanIDRequired)

	arg.PlanID = 1234232
	_, err = e.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.SubscriptionAmount = 4
	_, err = e.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubscriptionCycle)

	arg.SubscriptionCycle = "H4"
	arg.SubscriptionStartTime = -1
	_, err = e.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubscriptionStartTime)

	arg.SubscriptionStartTime = 8
	_, err = e.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.SourceAsset = currency.USDT
	_, err = e.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errPortfolioDetailRequired)

	arg.Details = []PortfolioDetail{{}}
	_, err = e.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Details = []PortfolioDetail{{TargetAsset: currency.BTC, Percentage: -1}}
	_, err = e.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPercentageAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.InvestmentPlanAdjustment(t.Context(), &AdjustInvestmentPlan{
		PlanID:                1234232,
		SubscriptionAmount:    4,
		SubscriptionCycle:     "H4",
		SubscriptionStartTime: 8,
		SourceAsset:           currency.USDT,
		Details: []PortfolioDetail{
			{
				TargetAsset: currency.ETH,
				Percentage:  12,
			},
			{
				TargetAsset: currency.ETH,
				Percentage:  20,
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangePlanStatus(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePlanStatus(t.Context(), 0, "PAUSED")
	require.ErrorIs(t, err, errPlanIDRequired)

	_, err = e.ChangePlanStatus(t.Context(), 12345, "")
	require.ErrorIs(t, err, errPlanStatusRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangePlanStatus(t.Context(), 12345, "PAUSED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetListOfPlans(t *testing.T) {
	t.Parallel()
	_, err := e.GetListOfPlans(t.Context(), "")
	require.ErrorIs(t, err, errPlanTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetListOfPlans(t.Context(), "SINGLE")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHoldingDetailsOfPlan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetHoldingDetailsOfPlan(t.Context(), 1234, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubscriptionsTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSubscriptionsTransactionHistory(t.Context(), 1232, 20, 0, endTime, startTime, currency.BTC, "PORTFOLIO")
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubscriptionsTransactionHistory(t.Context(), 1232, 20, 0, startTime, endTime, currency.BTC, "PORTFOLIO")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexDetail(t.Context(), 0)
	require.ErrorIs(t, err, errIndexIDIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIndexDetail(t.Context(), 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexLinkedPlanPositionDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexLinkedPlanPositionDetails(t.Context(), 0)
	require.ErrorIs(t, err, errIndexIDIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIndexLinkedPlanPositionDetails(t.Context(), 123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOneTimeTransaction(t *testing.T) {
	t.Parallel()
	_, err := e.OneTimeTransaction(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &OneTimeTransactionParams{}
	_, err = e.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, errSourceTypeRequired)

	arg.SourceType = "MAIN_SITE"
	_, err = e.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.SubscriptionAmount = 12
	_, err = e.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.SourceAsset = currency.USDT
	_, err = e.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, errPortfolioDetailRequired)

	_, err = e.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, errPortfolioDetailRequired)

	arg.Details = []PortfolioDetail{{}}
	_, err = e.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Details = []PortfolioDetail{{TargetAsset: currency.BTC}}
	_, err = e.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPercentageAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.OneTimeTransaction(t.Context(), &OneTimeTransactionParams{
		SourceType:         "MAIN_SITE",
		SubscriptionAmount: 12,
		SourceAsset:        currency.USDT,
		Details: []PortfolioDetail{
			{
				TargetAsset: currency.BTC,
				Percentage:  30,
			},
			{
				TargetAsset: currency.ETH,
				Percentage:  50,
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOneTimeTransactionStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetOneTimeTransactionStatus(t.Context(), 0, "")
	require.ErrorIs(t, err, errTransactionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOneTimeTransactionStatus(t.Context(), 1234, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIndexLinkedPlanRedemption(t *testing.T) {
	t.Parallel()
	_, err := e.IndexLinkedPlanRedemption(t.Context(), 0, 30, "")
	require.ErrorIs(t, err, errIndexIDIsRequired)
	_, err = e.IndexLinkedPlanRedemption(t.Context(), 12333, 0, "")
	require.ErrorIs(t, err, errInvalidPercentageAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.IndexLinkedPlanRedemption(t.Context(), 12333, 30, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexLinkedPlanRedemption(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexLinkedPlanRedemption(t.Context(), "", time.Time{}, time.Time{}, currency.ETH, 0, 10)
	require.ErrorIs(t, err, errRequestIDRequired)

	startTime, endTime := getTime()
	_, err = e.GetIndexLinkedPlanRedemption(t.Context(), "123123", startTime, endTime, currency.ETH, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetIndexLinkedPlanRedemption(t.Context(), "123123", startTime, endTime, currency.ETH, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexLinkedPlanRebalanceDetails(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetIndexLinkedPlanRebalanceDetails(t.Context(), endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIndexLinkedPlanRebalanceDetails(t.Context(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubscribeETHStaking(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubscribeETHStaking(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubscribeETHStaking(t.Context(), 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSusbcribeETHStakingV2(t *testing.T) {
	t.Parallel()
	_, err := e.SusbcribeETHStakingV2(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.SusbcribeETHStakingV2(t.Context(), 0.123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemETH(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemETH(t.Context(), 0, currency.ETH)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RedeemETH(t.Context(), 0.123, currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHStakingHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetETHStakingHistory(t.Context(), endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetETHStakingHistory(t.Context(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHRedemptionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetETHRedemptionHistory(t.Context(), endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetETHRedemptionHistory(t.Context(), startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBETHRewardsDistributionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBETHRewardsDistributionHistory(t.Context(), endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBETHRewardsDistributionHistory(t.Context(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentETHStakingQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrentETHStakingQuota(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHRateHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetWBETHRateHistory(t.Context(), endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWBETHRateHistory(t.Context(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHStakingAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetETHStakingAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHStakingAccountV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetETHStakingAccountV2(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWrapBETH(t *testing.T) {
	t.Parallel()
	_, err := e.WrapBETH(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WrapBETH(t.Context(), 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHWrapHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetWBETHWrapHistory(t.Context(), endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWBETHWrapHistory(t.Context(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHUnwrapHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetWBETHUnwrapHistory(t.Context(), endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWBETHUnwrapHistory(t.Context(), startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHRewardHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetWBETHRewardHistory(t.Context(), endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWBETHRewardHistory(t.Context(), startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSOLStakingAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSOLStakingAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSOLStakingQuotaDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSOLStakingQuotaDetails(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeToSOLStaking(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeToSOLStaking(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubscribeToSOLStaking(t.Context(), 1.2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemSOL(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemSOL(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RedeemSOL(t.Context(), 1.2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClaimBoostRewards(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ClaimBoostRewards(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSOLStakingHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSOLStakingHistory(t.Context(), endTime, startTime, 0, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSOLStakingHistory(t.Context(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSOLRedemptionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSOLRedemptionHistory(t.Context(), endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSOLRedemptionHistory(t.Context(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBNSOLRewardsHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBNSOLRewardsHistory(t.Context(), endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBNSOLRewardsHistory(t.Context(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBNSOLRateHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBNSOLRateHistory(t.Context(), endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBNSOLRateHistory(t.Context(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBoostRewardsHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBoostRewardsHistory(t.Context(), "", startTime, endTime, 0, 100)
	require.ErrorIs(t, err, errRewardTypeMissing)
	_, err = e.GetBoostRewardsHistory(t.Context(), "", endTime, startTime, 0, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBoostRewardsHistory(t.Context(), "CLAIM", startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnclaimedRewards(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUnclaimedRewards(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcquiringAlgorithm(t *testing.T) {
	t.Parallel()
	result, err := e.AcquiringAlgorithm(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCoinNames(t *testing.T) {
	t.Parallel()
	result, err := e.GetCoinNames(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailMinerList(t *testing.T) {
	t.Parallel()
	_, err := e.GetDetailMinerList(t.Context(), "sha256", "", "bhdc1.16A10404B")
	require.ErrorIs(t, err, errNameRequired)
	_, err = e.GetDetailMinerList(t.Context(), "", "sams", "bhdc1.16A10404B")
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = e.GetDetailMinerList(t.Context(), "sha256", "sams", "")
	require.ErrorIs(t, err, errNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDetailMinerList(t.Context(), "sha256", "sams", "bhdc1.16A10404B")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMinersList(t *testing.T) {
	t.Parallel()
	_, err := e.GetMinersList(t.Context(), "", "sams", true, 0, 10, 10)
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = e.GetMinersList(t.Context(), "sha256", "", true, 0, 10, 10)
	require.ErrorIs(t, err, errNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMinersList(t.Context(), "sha256", "sams", true, 0, 10, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarningList(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetEarningList(t.Context(), "sha256", "sams", currency.ETH, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetEarningList(t.Context(), "sha256", "sams", currency.ETH, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExtraBonousList(t *testing.T) {
	t.Parallel()
	_, err := e.ExtraBonousList(t.Context(), "", "sams", currency.ETH, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = e.ExtraBonousList(t.Context(), "sha256", "", currency.ETH, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errUsernameRequired)

	startTime, endTime := getTime()
	_, err = e.ExtraBonousList(t.Context(), "sha256", "sams", currency.ETH, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ExtraBonousList(t.Context(), "sha256", "sams", currency.ETH, startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHashrateRescaleList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetHashrateRescaleList(t.Context(), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHashrateRescaleDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetHashRateRescaleDetail(t.Context(), "", "sams", 10, 20)
	require.ErrorIs(t, err, errConfigIDRequired)
	_, err = e.GetHashRateRescaleDetail(t.Context(), "168", "", 10, 20)
	require.ErrorIs(t, err, errUsernameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetHashRateRescaleDetail(t.Context(), "168", "sams", 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHashrateRescaleRequest(t *testing.T) {
	t.Parallel()
	_, err := e.HashRateRescaleRequest(t.Context(), "", "sha256", "S19pro", time.Time{}, time.Time{}, 10000)
	require.ErrorIs(t, err, errUsernameRequired)
	_, err = e.HashRateRescaleRequest(t.Context(), "sams", "", "S19pro", time.Time{}, time.Time{}, 10000)
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = e.HashRateRescaleRequest(t.Context(), "sams", "sha256", "S19pro", time.Time{}, time.Time{}, 10000)
	require.ErrorIs(t, err, common.ErrDateUnset)
	_, err = e.HashRateRescaleRequest(t.Context(), "sams", "sha256", "", time.Time{}, time.Time{}, 10000)
	require.ErrorIs(t, err, errAccountRequired)
	_, err = e.HashRateRescaleRequest(t.Context(), "sams", "sha256", "S19pro", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errHashRateRequired)

	startTime, endTime := getTime()
	_, err = e.HashRateRescaleRequest(t.Context(), "sams", "sha256", "S19pro", endTime, startTime, 10000)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.HashRateRescaleRequest(t.Context(), "sams", "sha256", "S19pro", startTime, endTime, 10000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelHashrateRescaleConfiguration(t *testing.T) {
	t.Parallel()
	_, err := e.CancelHashrateRescaleConfiguration(t.Context(), "", "sams")
	require.ErrorIs(t, err, errConfigIDRequired)
	_, err = e.CancelHashrateRescaleConfiguration(t.Context(), "189", "")
	require.ErrorIs(t, err, errUsernameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelHashrateRescaleConfiguration(t.Context(), "189", "sams")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestStatisticsList(t *testing.T) {
	t.Parallel()
	_, err := e.StatisticsList(t.Context(), "", "sams")
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = e.StatisticsList(t.Context(), "sha256", "")
	require.ErrorIs(t, err, errUsernameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.StatisticsList(t.Context(), "sha256", "sams")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountList(t.Context(), "", "sams")
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = e.GetAccountList(t.Context(), "sha256", "")
	require.ErrorIs(t, err, errUsernameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountList(t.Context(), "sha256", "sams")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMiningAccountEarningRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetMiningAccountEarningRate(t.Context(), "", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errTransferAlgorithmRequired)

	startTime, endTime := getTime()
	_, err = e.GetMiningAccountEarningRate(t.Context(), "", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMiningAccountEarningRate(t.Context(), "sha256", startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewFuturesAccountTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.NewFuturesAccountTransfer(t.Context(), currency.EMPTYCODE, 0.001, 2)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.NewFuturesAccountTransfer(t.Context(), currency.ETH, 0, 2)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.NewFuturesAccountTransfer(t.Context(), currency.ETH, 0.001, 0)
	require.ErrorIs(t, err, errTransferTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewFuturesAccountTransfer(t.Context(), currency.ETH, 0.001, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountTransactionHistoryList(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFuturesAccountTransactionHistoryList(t.Context(), currency.BTC, endTime, startTime, 10, 20)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetFuturesAccountTransactionHistoryList(t.Context(), currency.BTC, startTime, endTime, 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFutureTickLevelOrderbookHistoricalDataDownloadLink(t *testing.T) {
	t.Parallel()
	_, err := e.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), currency.EMPTYPAIR, "T_DEPTH", time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), usdtmTradablePair, "T_DEPTH", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errStartTimeRequired)

	startTime, endTime := getTime()
	_, err = e.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), usdtmTradablePair, "T_DEPTH", endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), usdtmTradablePair, "T_DEPTH", startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVolumeParticipationNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{Urgency: "HIGH"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{
		Symbol:       usdtmTradablePair,
		PositionSide: "BOTH",
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{
		Symbol:       usdtmTradablePair,
		Side:         order.Sell.String(),
		PositionSide: "BOTH",
	})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{
		Symbol:       usdtmTradablePair,
		Side:         order.Sell.String(),
		PositionSide: "BOTH",
		Quantity:     0.012,
	})
	require.ErrorIs(t, err, errPossibleValuesRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{
		Symbol:       usdtmTradablePair,
		Side:         order.Sell.String(),
		PositionSide: "BOTH",
		Quantity:     0.012,
		Urgency:      "HIGH",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTWAPOrder(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{
		Duration: 1000,
	})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{
		Symbol: usdtmTradablePair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{
		Symbol: usdtmTradablePair,
		Side:   order.Sell.String(),
	})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{
		Symbol:   usdtmTradablePair,
		Side:     order.Sell.String(),
		Quantity: 0.012,
	})
	require.ErrorIs(t, err, errDurationRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{
		Symbol:       usdtmTradablePair,
		Side:         order.Sell.String(),
		PositionSide: "BOTH",
		Quantity:     0.012,
		Duration:     1000,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesAlgoOrder(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelFuturesAlgoOrder(t.Context(), 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentAlgoOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesCurrentAlgoOpenOrders(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalAlgoOrders(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFuturesHistoricalAlgoOrders(t.Context(), usdtmTradablePair, "BUY", endTime, startTime, 10, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesHistoricalAlgoOrders(t.Context(), usdtmTradablePair, "BUY", startTime, endTime, 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesSubOrders(t.Context(), 0, 0, 40)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesSubOrders(t.Context(), 1234, 0, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTWAPNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{Duration: 86400})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{Symbol: usdtmTradablePair})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{Symbol: usdtmTradablePair, Side: order.Sell.String()})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{Symbol: usdtmTradablePair, Side: order.Sell.String(), Quantity: 0.012})
	require.ErrorIs(t, err, errDurationRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{
		Symbol:   usdtmTradablePair,
		Side:     order.Sell.String(),
		Quantity: 0.012,
		Duration: 86400,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSpotAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelSpotAlgoOrder(t.Context(), 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentSpotAlgoOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrentSpotAlgoOpenOrder(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotHistoricalAlgoOrders(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSpotHistoricalAlgoOrders(t.Context(), usdtmTradablePair, "BUY", endTime, startTime, 10, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotHistoricalAlgoOrders(t.Context(), usdtmTradablePair, "BUY", startTime, endTime, 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotSubOrders(t.Context(), 1234, 1, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPortfolioMarginAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetClassicPortfolioMarginAccountInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPortfolioMarginCollateralRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetClassicPortfolioMarginCollateralRate(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPortfolioMarginBankruptacyLoanAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetClassicPortfolioMarginBankruptacyLoanAmount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayClassicPMBankruptacyLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RepayClassicPMBankruptacyLoan(t.Context(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPMNegativeBalanceInterestHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetClassicPMNegativeBalanceInterestHistory(t.Context(), currency.ETH, endTime, startTime, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetClassicPMNegativeBalanceInterestHistory(t.Context(), currency.ETH, startTime, endTime, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPMAssetIndexPrice(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPMAssetIndexPrice(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClassicPMFundAutoCollection(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ClassicPMFundAutoCollection(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClassicFundCollectionByAsset(t *testing.T) {
	t.Parallel()
	_, err := e.ClassicFundCollectionByAsset(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ClassicFundCollectionByAsset(t.Context(), currency.LTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAutoRepayFuturesStatusClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeAutoRepayFuturesStatusClassic(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoRepayFuturesStatusClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAutoRepayFuturesStatusClassic(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayFuturesNegativeBalanceClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RepayFuturesNegativeBalanceClassic(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginAssetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioMarginAssetLeverage(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserNegativeBalanceAutoExchangeRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUserNegativeBalanceAutoExchangeRecord(t.Context(), endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserNegativeBalanceAutoExchangeRecord(t.Context(), startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBLVTInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBLVTInfo(t.Context(), "BTCDOWN")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeBLVT(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeBLVT(t.Context(), "", 0.011)
	require.ErrorIs(t, err, errNameRequired)
	_, err = e.SubscribeBLVT(t.Context(), "BTCUP", 0)
	require.ErrorIs(t, err, errCostRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubscribeBLVT(t.Context(), "BTCUP", 0.011)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSusbcriptionRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSusbcriptionRecords(t.Context(), "BTCDOWN", endTime, startTime, 10, 20)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSusbcriptionRecords(t.Context(), "BTCDOWN", startTime, endTime, 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemBLVT(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemBLVT(t.Context(), currency.EMPTYPAIR, 2)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.RedeemBLVT(t.Context(), usdtmTradablePair, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.RedeemBLVT(t.Context(), usdtmTradablePair, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRedemptionRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetRedemptionRecord(t.Context(), "BTCDOWN", endTime, startTime, 10, 20)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRedemptionRecord(t.Context(), "BTCDOWN", startTime, endTime, 1, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBLVTUserLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBLVTUserLimitInfo(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatDepositAndWithdrawalHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatDepositAndWithdrawalHistory(t.Context(), time.Time{}, time.Time{}, -5, 0, 50)
	require.ErrorIs(t, err, errInvalidTransactionType)

	startTime, endTime := getTime()
	_, err = e.GetFiatDepositAndWithdrawalHistory(t.Context(), endTime, startTime, -5, 0, 50)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFiatDepositAndWithdrawalHistory(t.Context(), startTime, endTime, 1, 10, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatPaymentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatPaymentHistory(t.Context(), time.Time{}, time.Time{}, -1, 0, 50)
	require.ErrorIs(t, err, errInvalidTransactionType)

	startTime, endTime := getTime()
	_, err = e.GetFiatPaymentHistory(t.Context(), endTime, startTime, 1, 0, 50)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFiatPaymentHistory(t.Context(), startTime, endTime, 1, 0, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetC2CTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetC2CTradeHistory(t.Context(), "", time.Time{}, time.Time{}, 0, 50)
	require.ErrorIs(t, err, errTradeTypeRequired)

	startTime, endTime := getTime()
	_, err = e.GetC2CTradeHistory(t.Context(), order.Sell.String(), endTime, startTime, 1, 50)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetC2CTradeHistory(t.Context(), order.Sell.String(), startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPLoanOngoingOrders(t.Context(), 1232, 21231, 0, 10, currency.BTC, currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanRepay(t *testing.T) {
	t.Parallel()
	_, err := e.VIPLoanRepay(t.Context(), 0, 0.2)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.VIPLoanRepay(t.Context(), 1234, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.VIPLoanRepay(t.Context(), 1234, 0.2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPayTradeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPayTradeHistory(t.Context(), endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPayTradeHistory(t.Context(), startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllConvertPairs(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllConvertPairs(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetAllConvertPairs(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderQuantityPrecisionPerAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderQuantityPrecisionPerAsset(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSendQuoteRequest(t *testing.T) {
	t.Parallel()
	_, err := e.SendQuoteRequest(t.Context(), currency.EMPTYCODE, currency.USDT, 10, 20, "FUNDING", "1m")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SendQuoteRequest(t.Context(), currency.BTC, currency.EMPTYCODE, 10, 20, "FUNDING", "1m")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SendQuoteRequest(t.Context(), currency.BTC, currency.USDT, 0, 0, "FUNDING", "1m")
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SendQuoteRequest(t.Context(), currency.BTC, currency.USDT, 10, 20, "FUNDING", "1m")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcceptQuote(t *testing.T) {
	t.Parallel()
	_, err := e.AcceptQuote(t.Context(), "")
	require.ErrorIs(t, err, errQuoteIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AcceptQuote(t.Context(), "933256278426274426")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertOrderStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetConvertOrderStatus(t.Context(), "933256278426274426", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceLimitOrder(t *testing.T) {
	t.Parallel()
	arg := &ConvertPlaceLimitOrderParam{}
	_, err := e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.ExpiredType = "7_D"
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.BaseAsset = currency.BTC
	arg.QuoteAsset = currency.ETH
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.LimitPrice = 0.0122
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	arg.ExpiredType = ""
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, errExpiredTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceLimitOrder(t.Context(), &ConvertPlaceLimitOrderParam{
		BaseAsset:   currency.BTC,
		QuoteAsset:  currency.ETH,
		LimitPrice:  0.0122,
		Side:        order.Sell.String(),
		ExpiredType: "7_D",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelLimitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelLimitOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelLimitOrder(t.Context(), "123434")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLimitOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLimitOpenOrders(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertTradeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetConvertTradeHistory(t.Context(), endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetConvertTradeHistory(t.Context(), startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotRebateHistoryRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSpotRebateHistoryRecords(t.Context(), endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotRebateHistoryRecords(t.Context(), startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetNFTTransactionHistory(t.Context(), -1, time.Time{}, time.Time{}, 10, 40)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	startTime, endTime := getTime()
	_, err = e.GetNFTTransactionHistory(t.Context(), 1, endTime, startTime, 10, 40)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetNFTTransactionHistory(t.Context(), 1, startTime, endTime, 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTDepositHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetNFTDepositHistory(t.Context(), endTime, startTime, 10, 40)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetNFTDepositHistory(t.Context(), startTime, endTime, 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTWithdrawalHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetNFTWithdrawalHistory(t.Context(), endTime, startTime, 10, 40)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetNFTWithdrawalHistory(t.Context(), startTime, endTime, 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetNFTAsset(t.Context(), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSingleTokenGiftCard(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSingleTokenGiftCard(t.Context(), currency.EMPTYCODE, 0.1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.CreateSingleTokenGiftCard(t.Context(), currency.BUSD, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateSingleTokenGiftCard(t.Context(), currency.BUSD, 0.1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateDualTokenGiftCard(t *testing.T) {
	t.Parallel()
	_, err := e.CreateDualTokenGiftCard(t.Context(), currency.EMPTYCODE, currency.BNB, 10, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.CreateDualTokenGiftCard(t.Context(), currency.BUSD, currency.EMPTYCODE, 10, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.CreateDualTokenGiftCard(t.Context(), currency.BUSD, currency.BNB, 0, 10)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.CreateDualTokenGiftCard(t.Context(), currency.BUSD, currency.BNB, 10, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateDualTokenGiftCard(t.Context(), currency.BUSD, currency.BNB, 10, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemBinanaceGiftCard(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemBinanaceGiftCard(t.Context(), "", "12345")
	require.ErrorIs(t, err, errCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RedeemBinanaceGiftCard(t.Context(), "0033002328060227", "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVerifyBinanceGiftCardNumber(t *testing.T) {
	t.Parallel()
	_, err := e.VerifyBinanceGiftCardNumber(t.Context(), "")
	require.ErrorIs(t, err, errReferenceNumberRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.VerifyBinanceGiftCardNumber(t.Context(), "123456")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchRSAPublicKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FetchRSAPublicKey(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTokenLimit(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTokenLimit(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FetchTokenLimit(t.Context(), currency.BUSD)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetVIPLoanRepaymentHistory(t.Context(), currency.ETH, endTime, startTime, 1234, 0, 20)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPLoanRepaymentHistory(t.Context(), currency.ETH, startTime, endTime, 1234, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVIPLoanRenew(t *testing.T) {
	t.Parallel()
	_, err := e.VIPLoanRenew(t.Context(), 0, 60)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.VIPLoanRenew(t.Context(), 1234, 60)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckLockedValueVIPCollateralAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CheckLockedValueVIPCollateralAccount(t.Context(), 0, 40)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CheckLockedValueVIPCollateralAccount(t.Context(), 1223, 0)
	require.ErrorIs(t, err, errAccountIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CheckLockedValueVIPCollateralAccount(t.Context(), 1223, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVIPLoanBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.VIPLoanBorrow(t.Context(), 0, 30, currency.ETH, currency.LTC, 123, "1234", false)
	require.ErrorIs(t, err, errAccountIDRequired)
	_, err = e.VIPLoanBorrow(t.Context(), 1234, 30, currency.EMPTYCODE, currency.LTC, 123, "1234", false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.VIPLoanBorrow(t.Context(), 1234, 30, currency.ETH, currency.LTC, 0, "1234", false)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.VIPLoanBorrow(t.Context(), 1234, 30, currency.ETH, currency.LTC, 1.2, "", false)
	require.ErrorIs(t, err, errAccountIDRequired)
	_, err = e.VIPLoanBorrow(t.Context(), 1234, 30, currency.ETH, currency.EMPTYCODE, 123, "1234", false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.VIPLoanBorrow(t.Context(), 1234, 0, currency.ETH, currency.LTC, 123, "1234", false)
	require.ErrorIs(t, err, errLoanTermMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.VIPLoanBorrow(t.Context(), 1234, 30, currency.ETH, currency.LTC, 123, "1234", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanableAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPLoanableAssetsData(t.Context(), currency.BTC, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPCollateralAssetData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPCollateralAssetData(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPApplicationStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPApplicationStatus(t.Context(), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPBorrowInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetVIPBorrowInterestRate(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPBorrowInterestRate(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanAccruedInterest(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetVIPLoanAccruedInterest(t.Context(), "12345", currency.BTC, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPLoanAccruedInterest(t.Context(), "12345", currency.BTC, startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanInterestRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetVIPLoanInterestRateHistory(t.Context(), currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	startTime, endTime := getTime()
	_, err = e.GetVIPLoanInterestRateHistory(t.Context(), currency.BTC, endTime, startTime, 0, 20)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPLoanInterestRateHistory(t.Context(), currency.BTC, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSpotListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CreateSpotListenKey(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKeepListenKeyAlive(t *testing.T) {
	t.Parallel()
	err := e.KeepSpotListenKeyAlive(t.Context(), "")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.KeepSpotListenKeyAlive(t.Context(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	require.NoError(t, err)
}

func TestCloseListenKey(t *testing.T) {
	t.Parallel()
	err := e.CloseSpotListenKey(t.Context(), "")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CloseSpotListenKey(t.Context(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	require.NoError(t, err)
}

func TestCreateMarginListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CreateMarginListenKey(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKeepMarginListenKeyAlive(t *testing.T) {
	t.Parallel()
	err := e.KeepMarginListenKeyAlive(t.Context(), "")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.KeepMarginListenKeyAlive(t.Context(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	assert.NoError(t, err)
}

func TestCloseMarginListenKey(t *testing.T) {
	t.Parallel()
	err := e.CloseMarginListenKey(t.Context(), "")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.CloseMarginListenKey(t.Context(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	assert.NoError(t, err)
}

func TestCreateCrossMarginListenKey(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCrossMarginListenKey(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CreateCrossMarginListenKey(t.Context(), usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKeepCrossMarginListenKeyAlive(t *testing.T) {
	t.Parallel()
	err := e.KeepCrossMarginListenKeyAlive(t.Context(), usdtmTradablePair, "")
	require.ErrorIs(t, err, errListenKeyIsRequired)
	err = e.KeepCrossMarginListenKeyAlive(t.Context(), currency.EMPTYPAIR, "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.KeepCrossMarginListenKeyAlive(t.Context(), usdtmTradablePair, "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	assert.NoError(t, err)
}

func TestCloseCrossMarginListenKey(t *testing.T) {
	t.Parallel()
	err := e.CloseCrossMarginListenKey(t.Context(), usdtmTradablePair, "")
	require.ErrorIs(t, err, errListenKeyIsRequired)
	err = e.CloseCrossMarginListenKey(t.Context(), currency.EMPTYPAIR, "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CloseCrossMarginListenKey(t.Context(), usdtmTradablePair, "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	assert.NoError(t, err)
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()
	data := []byte(`{"data":[{"1":"0.6"}, {"2":"0.6"}]}`)
	resp := &struct {
		Data WalletAssetCosts `json:"data"`
	}{}
	err := json.Unmarshal(data, resp)
	require.NoError(t, err)
	require.Equal(t, 0.6, resp.Data[0]["1"].Float64())
	assert.Equal(t, 0.6, resp.Data[1]["2"].Float64())
}

func (e *Exchange) populateTradablePairs() error {
	if err := e.UpdateTradablePairs(context.Background()); err != nil {
		return err
	}
	tradablePairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return fmt.Errorf("%w for %v", currency.ErrCurrencyPairsEmpty, asset.Spot)
	}
	spotTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Spot)
	if err != nil {
		return err
	}
	tradablePairs, err = e.GetEnabledPairs(asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	if len(tradablePairs) != 0 {
		usdtmTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.USDTMarginedFutures)
		if err != nil {
			return err
		}
	}
	tradablePairs, err = e.GetEnabledPairs(asset.CoinMarginedFutures)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		coinmTradablePair, err = currency.NewPairFromString("ETHUSD_PERP")
		if err != nil {
			return err
		}
	} else {
		coinmTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.CoinMarginedFutures)
		if err != nil {
			return err
		}
	}
	tradablePairs, err = e.GetEnabledPairs(asset.Options)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return fmt.Errorf("%w for %v", currency.ErrCurrencyPairsEmpty, asset.Options)
	}
	optionsTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Options)
	return err
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		require.NotEmpty(t, resp)
	}
}

func TestFetchOptionsExchangeLimits(t *testing.T) {
	t.Parallel()
	l, err := e.FetchOptionsExchangeLimits(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, l, "Should get some limits back")
}

// ----------------- Copy Trading endpoints unit-tests ----------------

func TestGetFuturesLeadTraderStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesLeadTraderStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesLeadTradingSymbolWhitelist(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesLeadTradingSymbolWhitelist(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawalHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.WithdrawalHistoryV1(t.Context(), []string{"1234"}, []string{"0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268"}, []string{}, "", "0", 0, 100, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WithdrawalHistoryV1(t.Context(), []string{"1234"}, []string{"0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268"}, []string{}, "", "0", 0, 100, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawalHistoryV2(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.WithdrawalHistoryV2(t.Context(), []string{"1234"}, []string{"0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268"}, []string{}, "", "0", 0, 100, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WithdrawalHistoryV2(t.Context(), []string{"1234"}, []string{"0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268"}, []string{}, "", "0", 0, 100, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitDepositQuestionnaire(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitDepositQuestionnaire(t.Context(), "765127651", map[string]interface{}{
		"isAddressOwner": 2,
		"sendTo":         1,
		"vaspCountry":    "cn",
		"vaspRegion":     "notNortheasternProvinces",
		"txnPurpose":     "3",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLocalEntitiesDepositHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetLocalEntitiesDepositHistory(t.Context(), []string{}, []string{}, []string{}, "BNB", currency.USDT, "1", false, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLocalEntitiesDepositHistory(t.Context(), []string{}, []string{}, []string{}, "BNB", currency.USDT, "1", false, startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOnboardedVASPList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOnboardedVASPList(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateSubAccount(t.Context(), "tag-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccounts(t.Context(), "1", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableFuturesForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.EnableFuturesForSubAccount(t.Context(), "", false)
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableFuturesForSubAccount(t.Context(), "1", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateAPIKeyForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAPIKeyForSubAccount(t.Context(), "", false, true, true)
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateAPIKeyForSubAccount(t.Context(), "1", false, true, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeSubAccountAPIPermission(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeSubAccountAPIPermission(t.Context(), "", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", false, true, true)
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = e.ChangeSubAccountAPIPermission(t.Context(), "1", "", false, true, true)
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeSubAccountAPIPermission(t.Context(), "", "", false, true, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableUniversalTransferPermissionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.EnableUniversalTransferPermissionForSubAccountAPIKey(t.Context(), "", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", false)
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = e.EnableUniversalTransferPermissionForSubAccountAPIKey(t.Context(), "1", "", false)
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableUniversalTransferPermissionForSubAccountAPIKey(t.Context(), "1", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), "", "", "2", "")
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = e.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), "123", "", "2", "")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)
	_, err = e.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), "123", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", "", "")
	require.ErrorIs(t, err, errSubAccountStatusMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), "123", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", "2", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteIPRestrictionForSubAccountAPIKey(t.Context(), "", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", "")
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = e.DeleteIPRestrictionForSubAccountAPIKey(t.Context(), "123", "", "")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.DeleteIPRestrictionForSubAccountAPIKey(t.Context(), "123", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteSubAccountAPIKey(t.Context(), "", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A")
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = e.DeleteSubAccountAPIKey(t.Context(), "123", "")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.DeleteSubAccountAPIKey(t.Context(), "123", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeSubAccountCommission(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeSubAccountCommission(t.Context(), "", 1., 2., 0, 0)
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = e.ChangeSubAccountCommission(t.Context(), "2", 0, 2., 0, 0)
	require.ErrorIs(t, err, errCommissionValueRequired)
	_, err = e.ChangeSubAccountCommission(t.Context(), "2", 1., 0, 0, 0)
	require.ErrorIs(t, err, errCommissionValueRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeSubAccountCommission(t.Context(), "2", 1., 2., 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBNBBurnStatusForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetBNBBurnStatusForSubAccount(t.Context(), "")
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBNBBurnStatusForSubAccount(t.Context(), "1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransferWithSpotBroker(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountTransferWithSpotBroker(t.Context(), currency.EMPTYCODE, "", "", "", 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SubAccountTransferWithSpotBroker(t.Context(), currency.BTC, "", "", "", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubAccountTransferWithSpotBroker(t.Context(), currency.BTC, "", "", "", 13)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotBrokerSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSpotBrokerSubAccountTransferHistory(t.Context(), "", "", "", true, endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotBrokerSubAccountTransferHistory(t.Context(), "", "", "", true, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransferWithFuturesBroker(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountTransferWithFuturesBroker(t.Context(), currency.EMPTYCODE, "", "", "", 1, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SubAccountTransferWithFuturesBroker(t.Context(), currency.BTC, "", "", "", 2, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubAccountTransferWithFuturesBroker(t.Context(), currency.BTC, "", "", "", 1, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesBrokerSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFuturesBrokerSubAccountTransferHistory(t.Context(), false, "", "", endTime, startTime, 0, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesBrokerSubAccountTransferHistory(t.Context(), false, "", "", startTime, endTime, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositHistoryWithBroker(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSubAccountDepositHistoryWithBroker(t.Context(), "", currency.BTC, endTime, startTime, 0, 10, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountDepositHistoryWithBroker(t.Context(), "", currency.BTC, startTime, endTime, 0, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountSpotAssetInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountSpotAssetInfo(t.Context(), "1234", 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountMarginAssetInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountMarginAssetInfo(t.Context(), "", 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountFuturesAssetInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountFuturesAssetInfo(t.Context(), "1234", true, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUniversalTransferWithBroker(t *testing.T) {
	t.Parallel()
	_, err := e.UniversalTransferWithBroker(t.Context(), "", "USDT_FUTURE", "", "", "", currency.BTC, 1)
	require.ErrorIs(t, err, errInvalidAccountType)
	_, err = e.UniversalTransferWithBroker(t.Context(), "SPOT", "", "", "", "", currency.BTC, 1)
	require.ErrorIs(t, err, errInvalidAccountType)
	_, err = e.UniversalTransferWithBroker(t.Context(), "SPOT", "USDT_FUTURE", "", "", "", currency.EMPTYCODE, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.UniversalTransferWithBroker(t.Context(), "SPOT", "USDT_FUTURE", "", "", "", currency.BTC, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UniversalTransferWithBroker(t.Context(), "SPOT", "USDT_FUTURE", "", "", "", currency.BTC, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUniversalTransferHistoryThroughBroker(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUniversalTransferHistoryThroughBroker(t.Context(), "", "", "", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUniversalTransferHistoryThroughBroker(t.Context(), "", "", "", startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBrokerSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateBrokerSubAccount(t.Context(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerSubAccounts(t.Context(), "123", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableOrDisableBNBBurnForSubAccountMarginInterest(t *testing.T) {
	t.Parallel()
	_, err := e.EnableOrDisableBNBBurnForSubAccountMarginInterest(t.Context(), "", false)
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableOrDisableBNBBurnForSubAccountMarginInterest(t.Context(), "3", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableOrDisableBNBBurnForSubAccountSpotAndMargin(t *testing.T) {
	t.Parallel()
	_, err := e.EnableOrDisableBNBBurnForSubAccountSpotAndMargin(t.Context(), "", true)
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableOrDisableBNBBurnForSubAccountSpotAndMargin(t.Context(), "1", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLinkAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.LinkAccountInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "", spotTradablePair, 1, 10)
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = e.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "234", currency.EMPTYPAIR, 1, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "234", spotTradablePair, 0, 10)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "234", spotTradablePair, 1, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "234", spotTradablePair, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountUSDMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountUSDMarginedFuturesCommissionAdjustment(t.Context(), "", usdtmTradablePair)
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountUSDMarginedFuturesCommissionAdjustment(t.Context(), "123", usdtmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "", coinmTradablePair, 1., 2.)
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = e.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "231", currency.EMPTYPAIR, 1., 2.)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "231", coinmTradablePair, 0, 2.)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "231", coinmTradablePair, 1., 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "231", coinmTradablePair, 1., 2.)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountCoinMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "", coinmTradablePair)
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "123", coinmTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerCommissionRebateRecentRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSpotBrokerCommissionRebateRecentRecord(t.Context(), "1234", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotBrokerCommissionRebateRecentRecord(t.Context(), "1234", startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesBrokerCommissionRebateRecentRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFuturesBrokerCommissionRebateRecentRecord(t.Context(), false, false, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesBrokerCommissionRebateRecentRecord(t.Context(), false, false, startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ---------- Binance Link endpoints ----------------------------------

func TestGetInfoAboutIfUserIsNew(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotInfoAboutIfUserIsNew(t.Context(), "")
	require.ErrorIs(t, err, errCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotInfoAboutIfUserIsNew(t.Context(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCustomizeIDForClient(t *testing.T) {
	t.Parallel()
	_, err := e.CustomizeSpotPartnerClientID(t.Context(), "", "someone@thrasher.io")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CustomizeSpotPartnerClientID(t.Context(), "1233", "")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CustomizeSpotPartnerClientID(t.Context(), "1233", "someone@thrasher.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClientEmailCustomizedID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotClientEmailCustomizedID(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesClientEmailCustomizedID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesClientEmailCustomizedID(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCustomizeOwnClientID(t *testing.T) {
	t.Parallel()
	_, err := e.CustomizeSpotOwnClientID(t.Context(), "", "ABCDEFG")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CustomizeSpotOwnClientID(t.Context(), "the-unique-id", "")
	require.ErrorIs(t, err, errCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CustomizeSpotOwnClientID(t.Context(), "the-unique-id", "ABCDEFG")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCustomizeFuturesOwnClientID(t *testing.T) {
	t.Parallel()
	_, err := e.CustomizeFuturesOwnClientID(t.Context(), "", "ABCDEFG")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CustomizeFuturesOwnClientID(t.Context(), "the-unique-id", "")
	require.ErrorIs(t, err, errCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CustomizeFuturesOwnClientID(t.Context(), "the-unique-id", "ABCDEFG")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersCustomizedID(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotUsersCustomizedID(t.Context(), "")
	require.ErrorIs(t, err, errCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotUsersCustomizedID(t.Context(), "1234ABCD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesUsersCustomizedID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesUsersCustomizedID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesUsersCustomizedID(t.Context(), "1234ABCD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOthersRebateRecentRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOthersRebateRecentRecord(t.Context(), "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	startTime, endTime := getTime()
	_, err = e.GetSpotOthersRebateRecentRecord(t.Context(), "123123", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotOthersRebateRecentRecord(t.Context(), "123123", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOwnRebateRecentRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSpotOwnRebateRecentRecords(t.Context(), endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotOwnRebateRecentRecords(t.Context(), startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesClientIfNewUser(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesClientIfNewUser(t.Context(), "", 1)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesClientIfNewUser(t.Context(), "1234", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCustomizeFuturesPartnerClientID(t *testing.T) {
	t.Parallel()
	_, err := e.CustomizeFuturesPartnerClientID(t.Context(), "", "someone@thrasher.io")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CustomizeFuturesPartnerClientID(t.Context(), "1233", "")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CustomizeFuturesPartnerClientID(t.Context(), "1233", "someone@thrasher.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesUserIncomeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFuturesUserIncomeHistory(t.Context(), usdtmTradablePair, "COMMISSION", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesUserIncomeHistory(t.Context(), usdtmTradablePair, "COMMISSION", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesReferredTradersNumber(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFuturesReferredTradersNumber(t.Context(), false, endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesReferredTradersNumber(t.Context(), true, startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesRebateDataOverview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesRebateDataOverview(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTradeVolume(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUserTradeVolume(t.Context(), false, endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTradeVolume(t.Context(), true, startTime, endTime, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRebateVolume(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetRebateVolume(t.Context(), false, endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRebateVolume(t.Context(), false, startTime, endTime, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTraderDetail(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetTraderDetail(t.Context(), "sde001", true, endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTraderDetail(t.Context(), "sde001", true, startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesClientifNewUser(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesClientifNewUser(t.Context(), "", false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesClientifNewUser(t.Context(), "123123", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCustomizeIDForClientToReferredUser(t *testing.T) {
	t.Parallel()
	_, err := e.CustomizeIDForClientToReferredUser(t.Context(), "", "1234")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CustomizeIDForClientToReferredUser(t.Context(), "1234", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CustomizeIDForClientToReferredUser(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersCustomizeIDs(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersCustomizeIDs(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersCustomizeIDs(t.Context(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFastAPIUserStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAPIKey(t.Context(), "", "12312", "1", "", "", true, true, false, true)
	require.ErrorIs(t, err, errAPIKeyNameRequired)
	_, err = e.CreateAPIKey(t.Context(), "12345", "", "1", "", "", true, true, false, true)
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPICredentials)
	result, err := e.CreateAPIKey(t.Context(), "", "", "1", "", "", true, true, false, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrderTypeFromString(t *testing.T) {
	t.Parallel()
	orderTypeFromStringList := []struct {
		String    string
		OrderType order.Type
		Error     error
	}{
		{"STOP_MARKET", order.StopMarket, nil},
		{"TAKE_PROFIT", order.TakeProfit, nil},
		{"TAKE_PROFIT_MARKET", order.TakeProfitMarket, nil},
		{"TRAILING_STOP_MARKET", order.TrailingStop, nil},
		{"STOP_LOSS_LIMIT", order.StopLimit, nil},
		{"TAKE_PROFIT_LIMIT", order.TakeProfitLimit, nil},
		{"LIMIT_MAKER", order.LimitMaker, nil},
		{"LIMIT", order.Limit, nil},
		{"MARKET", order.Market, nil},
		{"STOP", order.Stop, nil},
		{"OCO", order.OCO, nil},
		{"OTO", order.OTO, nil},
		{"STOP_LOSS", order.Stop, nil},
		{"abcd", order.UnknownType, order.ErrUnsupportedOrderType},
	}
	for _, val := range orderTypeFromStringList {
		result, err := StringToOrderType(val.String)
		require.ErrorIs(t, err, val.Error)
		assert.Equal(t, result, val.OrderType)
	}
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	orderTypeStringToTypeList := []struct {
		OrderType order.Type
		String    string
		Error     error
	}{
		{order.Limit, "LIMIT", nil},
		{order.StopMarket, "STOP_MARKET", nil},
		{order.TakeProfit, "TAKE_PROFIT", nil},
		{order.TakeProfitMarket, "TAKE_PROFIT_MARKET", nil},
		{order.TrailingStop, "TRAILING_STOP_MARKET", nil},
		{order.StopLimit, "STOP_LOSS_LIMIT", nil},
		{order.TakeProfitLimit, "TAKE_PROFIT_LIMIT", nil},
		{order.LimitMaker, "LIMIT_MAKER", nil},
		{order.Market, "MARKET", nil},
		{order.OCO, "OCO", nil},
		{order.OTO, "OTO", nil},
		{order.Stop, "STOP_LOSS", nil},
		{order.IOS, "", order.ErrUnsupportedOrderType},
	}
	for _, value := range orderTypeStringToTypeList {
		result, err := OrderTypeString(value.OrderType)
		require.ErrorIs(t, err, value.Error)
		assert.Equal(t, result, value.String)
	}
}

func TestTimeInForceString(t *testing.T) {
	t.Parallel()
	timeInForceStringList := []struct {
		TIF    order.TimeInForce
		OType  order.Type
		String string
	}{
		{order.FillOrKill, 0, "FOK"},
		{order.ImmediateOrCancel, 0, "IOC"},
		{order.GoodTillCancel, 0, "GTC"},
		{order.GoodTillDay, 0, "GTD"},
		{order.GoodTillCrossing, 0, "GTX"},
		{order.UnknownTIF, order.Limit, "GTC"},
		{order.UnknownTIF, order.Market, "IOC"},
		{order.UnknownTIF, order.UnknownType, ""},
	}
	for _, val := range timeInForceStringList {
		result := timeInForceString(val.TIF, val.OType)
		assert.Equal(t, val.String, result)
	}
}
