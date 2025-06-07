package binance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	useTestNet              = false

	apiStreamingIsNotConnected = "API streaming is not connected"
)

var (
	b = &Binance{}

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
func getTime() (start, end time.Time) {
	if mockTests {
		return time.Unix(1577836800, 0), time.Unix(1580515200, 0)
	}
	tn := time.Now()
	offset := time.Hour * 24 * 6
	return tn.Add(-offset), tn
}

func TestUServerTime(t *testing.T) {
	t.Parallel()
	result, err := b.UServerTime(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetServerTime(t.Context(), asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	assetTypes := b.GetAssetTypes(true)
	for a := range assetTypes {
		st, err := b.GetServerTime(t.Context(), assetTypes[a])
		require.NoError(t, err)
		require.NotEmpty(t, st)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	for assetType, pair := range assetToTradablePairMap {
		r, err := b.UpdateTicker(t.Context(), pair, assetType)
		assert.NoErrorf(t, err, "expected nil, got %v for asset type: %s pair: %v", err, assetType, pair)
		assert.NotNilf(t, r, "unexpected value nil for asset type: %s pair: %v", assetType, pair)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	enabledAssets := b.GetAssetTypes(true)
	for _, assetType := range enabledAssets {
		err := b.UpdateTickers(t.Context(), assetType)
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
		result, err := b.UpdateOrderbook(t.Context(), tp, assetType)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

// USDT Margined Futures

func TestUExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := b.UExchangeInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.UFuturesOrderbook(t.Context(), "", 1000)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.UFuturesOrderbook(t.Context(), "BTCUSDT", 1000)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestURecentTrades(t *testing.T) {
	t.Parallel()
	_, err := b.URecentTrades(t.Context(), "", "", 1000)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.URecentTrades(t.Context(), "BTCUSDT", "", 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCompressedTrades(t *testing.T) {
	t.Parallel()
	_, err := b.UCompressedTrades(t.Context(), "", "", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.UCompressedTrades(t.Context(), "LTCUSDT", "", 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := time.UnixMilli(1744462163396), time.UnixMilli(1744552163396)
	if !mockTests {
		start, end = time.Now().Add(-time.Hour*25), time.Now()
	}
	result, err = b.UCompressedTrades(t.Context(), "LTCUSDT", "", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.UKlineData(t.Context(), "", "1d", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.UKlineData(t.Context(), usdtmTradablePair.String(), "", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := b.UKlineData(t.Context(), usdtmTradablePair.String(), "1d", 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	start, end := getTime()
	result, err = b.UKlineData(t.Context(), usdtmTradablePair.String(), "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUFuturesContinuousKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.GetUFuturesContinuousKlineData(t.Context(), currency.EMPTYPAIR, "CURRENT_QUARTER", "1d", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.GetUFuturesContinuousKlineData(t.Context(), usdtmTradablePair, "", "1d", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errContractTypeIsRequired)

	_, err = b.GetUFuturesContinuousKlineData(t.Context(), usdtmTradablePair, "CURRENT_QUARTER", "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := b.GetUFuturesContinuousKlineData(t.Context(), usdtmTradablePair, "CURRENT_QUARTER", "1d", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexOrCandlesticPriceKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexOrCandlesticPriceKlineData(t.Context(), currency.EMPTYPAIR, "1d", time.Time{}, time.Now(), 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.GetIndexOrCandlesticPriceKlineData(t.Context(), usdtmTradablePair, "", time.Time{}, time.Now(), 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := b.GetIndexOrCandlesticPriceKlineData(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "1d", time.Time{}, time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPriceKlineCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkPriceKlineCandlesticks(t.Context(), "", "1d", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.GetMarkPriceKlineCandlesticks(t.Context(), "BTCUSDT", "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := b.GetMarkPriceKlineCandlesticks(t.Context(), "BTCUSDT", "1d", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPremiumIndexKlineCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := b.GetPremiumIndexKlineCandlesticks(t.Context(), "", "1d", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.GetPremiumIndexKlineCandlesticks(t.Context(), "BTCUSDT", "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := b.GetPremiumIndexKlineCandlesticks(t.Context(), "BTCUSDT", "1d", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := b.UGetMarkPrice(t.Context(), usdtmTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.UGetMarkPrice(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetFundingHistory(t *testing.T) {
	t.Parallel()
	result, err := b.UGetFundingHistory(t.Context(), usdtmTradablePair.String(), 1000, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.UGetFundingHistory(t.Context(), usdtmTradablePair.String(), 1000, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestU24HTickerPriceChangeStats(t *testing.T) {
	t.Parallel()
	result, err := b.U24HTickerPriceChangeStats(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.U24HTickerPriceChangeStats(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	result, err := b.USymbolPriceTickerV1(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.USymbolPriceTickerV1(t.Context(), currency.EMPTYPAIR)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUSymbolPriceTickerV2(t *testing.T) {
	t.Parallel()
	result, err := b.USymbolPriceTickerV2(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.USymbolPriceTickerV2(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	result, err := b.USymbolOrderbookTicker(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.USymbolOrderbookTicker(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := b.UOpenInterest(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.UOpenInterest(t.Context(), usdtmTradablePair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuarterlyContractSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetQuarterlyContractSettlementPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := b.GetQuarterlyContractSettlementPrice(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUOpenInterestStats(t *testing.T) {
	t.Parallel()
	_, err := b.UOpenInterestStats(t.Context(), "", "5m", 1, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.UOpenInterestStats(t.Context(), usdtmTradablePair.String(), "", 1, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.UOpenInterestStats(t.Context(), usdtmTradablePair.String(), "5m", 1, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	start, end := getTime()
	result, err = b.UOpenInterestStats(t.Context(), usdtmTradablePair.String(), "1d", 10, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUTopAcccountsLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := b.UTopAcccountsLongShortRatio(t.Context(), "", "5m", 2, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.UTopAcccountsLongShortRatio(t.Context(), "BTCUSDT", "", 2, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.UTopAcccountsLongShortRatio(t.Context(), "BTCUSDT", "5m", 2, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	start, end := getTime()

	result, err = b.UTopAcccountsLongShortRatio(t.Context(), "BTCUSDT", "5m", 2, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUTopPostionsLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := b.UTopPostionsLongShortRatio(t.Context(), "", "5m", 3, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.UTopPostionsLongShortRatio(t.Context(), "BTCUSDT", "", 3, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.UTopPostionsLongShortRatio(t.Context(), "BTCUSDT", "5m", 3, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.UTopPostionsLongShortRatio(t.Context(), "BTCUSDT", "1d", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGlobalLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := b.UGlobalLongShortRatio(t.Context(), "", "5m", 3, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.UGlobalLongShortRatio(t.Context(), "BTCUSDT", "", 3, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.UGlobalLongShortRatio(t.Context(), "BTCUSDT", "5m", 3, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.UGlobalLongShortRatio(t.Context(), "BTCUSDT", "4h", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUTakerBuySellVol(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	_, err := b.UTakerBuySellVol(t.Context(), "", "", 10, start, end)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.UTakerBuySellVol(t.Context(), "BTCUSDT", "", 10, start, end)
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.UTakerBuySellVol(t.Context(), "BTCUSDT", "5m", 10, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBasis(t *testing.T) {
	t.Parallel()
	_, err := b.GetBasis(t.Context(), currency.EMPTYPAIR, "CURRENT_QUARTER", "15m", time.Time{}, time.Time{}, 20)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.GetBasis(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", "15m", time.Time{}, time.Time{}, 20)
	require.ErrorIs(t, err, errContractTypeIsRequired)

	_, err = b.GetBasis(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "CURRENT_QUARTER", "", time.Time{}, time.Time{}, 20)
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.GetBasis(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "CURRENT_QUARTER", "15m", time.Time{}, time.Time{}, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetBasis(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "NEXT_QUARTER", "15m", time.Time{}, time.Time{}, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetBasis(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "PERPETUAL", "15m", time.Time{}, time.Time{}, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalBLVTNAVCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := b.GetHistoricalBLVTNAVCandlesticks(t.Context(), "", "15m", time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.GetHistoricalBLVTNAVCandlesticks(t.Context(), "BTCDOWN", "", time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := b.GetHistoricalBLVTNAVCandlesticks(t.Context(), "BTCDOWN", "15m", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCompositeIndexInfo(t *testing.T) {
	t.Parallel()
	result, err := b.UCompositeIndexInfo(t.Context(), usdtmTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.UCompositeIndexInfo(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMultiAssetModeAssetIndex(t *testing.T) {
	t.Parallel()
	result, err := b.GetMultiAssetModeAssetIndex(t.Context(), "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetMultiAssetModeAssetIndex(t.Context(), "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceConstituents(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexPriceConstituents(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.GetIndexPriceConstituents(t.Context(), "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesNewOrder(t *testing.T) {
	t.Parallel()
	_, err := b.UFuturesNewOrder(t.Context(), &UFuturesNewOrderRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &UFuturesNewOrderRequest{
		ReduceOnly:   true,
		PositionSide: "position-side",
	}
	_, err = b.UFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPositionSide)

	arg.PositionSide = "LONG"
	arg.WorkingType = "abc"
	_, err = b.UFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidWorkingType)

	arg.WorkingType = "MARK_PRICE"
	arg.NewOrderRespType = "abc"
	_, err = b.UFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidNewOrderResponseType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UFuturesNewOrder(t.Context(),
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
	_, err := b.UModifyOrder(t.Context(), &USDTOrderUpdateParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &USDTOrderUpdateParams{PriceMatch: "1234"}
	_, err = b.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = 1234
	_, err = b.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = currency.NewPair(currency.BTC, currency.USD)
	_, err = b.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = b.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Amount = 1
	_, err = b.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UModifyOrder(t.Context(), &USDTOrderUpdateParams{
		OrderID:           1,
		OrigClientOrderID: "",
		Side:              order.Sell.String(),
		PriceMatch:        "TAKE_PROFIT",
		Symbol:            currency.NewPair(currency.BTC, currency.USD),
		Amount:            0.0000001,
		Price:             123455554,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPlaceBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := b.UPlaceBatchOrders(t.Context(), []PlaceBatchOrderData{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := PlaceBatchOrderData{
		TimeInForce:  "GTC",
		PositionSide: "abc",
	}
	_, err = b.UPlaceBatchOrders(t.Context(), []PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidPositionSide)

	arg.PositionSide = "SHORT"
	arg.WorkingType = "abc"
	_, err = b.UPlaceBatchOrders(t.Context(), []PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidWorkingType)

	arg.WorkingType = "CONTRACT_TYPE"
	arg.NewOrderRespType = "abc"
	_, err = b.UPlaceBatchOrders(t.Context(), []PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidNewOrderResponseType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	tempData := PlaceBatchOrderData{
		Symbol:      currency.Pair{Base: currency.BTC, Quote: currency.USDT},
		Side:        "BUY",
		OrderType:   order.Limit.String(),
		Quantity:    4,
		Price:       1,
		TimeInForce: "GTC",
	}
	result, err := b.UPlaceBatchOrders(t.Context(), []PlaceBatchOrderData{tempData})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := b.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := USDTOrderUpdateParams{}
	_, err = b.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = 1
	_, err = b.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{arg})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = spotTradablePair
	_, err = b.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = b.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Amount = 0.0001
	_, err = b.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{arg})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UModifyMultipleOrders(t.Context(), []USDTOrderUpdateParams{
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
	_, err := b.GetUSDTOrderModifyHistory(t.Context(), currency.EMPTYPAIR, "", 1234, 10, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.GetUSDTOrderModifyHistory(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", 0, 10, time.Time{}, time.Time{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUSDTOrderModifyHistory(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", 1234, 10, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetOrderData(t *testing.T) {
	t.Parallel()
	_, err := b.UGetOrderData(t.Context(), "", "123", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UGetOrderData(t.Context(), "BTCUSDT", "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := b.UCancelOrder(t.Context(), "", "123", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UCancelOrder(t.Context(), "BTCUSDT", "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := b.UCancelAllOpenOrders(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UCancelAllOpenOrders(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := b.UCancelBatchOrders(t.Context(), "", []string{"123"}, []string{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UCancelBatchOrders(t.Context(), "BTCUSDT", []string{"123"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := b.UAutoCancelAllOpenOrders(t.Context(), "", 30)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UAutoCancelAllOpenOrders(t.Context(), "BTCUSDT", 30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFetchOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UFetchOpenOrder(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAllAccountOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAllAccountOpenOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAllAccountOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAllAccountOrders(t.Context(), "", 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.UAllAccountOrders(t.Context(), "BTCUSDT", 0, 5, time.Now().Add(-time.Hour*4), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountBalanceV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAccountBalanceV2(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountInformationV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAccountInformationV2(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUChangeInitialLeverageRequest(t *testing.T) {
	t.Parallel()
	_, err := b.UChangeInitialLeverageRequest(t.Context(), "BTCUSDT", 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UChangeInitialLeverageRequest(t.Context(), "BTCUSDT", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUChangeInitialMarginType(t *testing.T) {
	t.Parallel()
	err := b.UChangeInitialMarginType(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.UChangeInitialMarginType(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "ISOLATED")
	assert.NoError(t, err)
}

func TestUModifyIsolatedPositionMarginReq(t *testing.T) {
	t.Parallel()
	_, err := b.UModifyIsolatedPositionMarginReq(t.Context(), "BTCUSDT", "LONG", "", 5)
	require.ErrorIs(t, err, errMarginChangeTypeInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UModifyIsolatedPositionMarginReq(t.Context(), "BTCUSDT", "LONG", "add", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPositionMarginChangeHistory(t *testing.T) {
	t.Parallel()
	_, err := b.UPositionMarginChangeHistory(t.Context(), "BTCUSDT", "", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errMarginChangeTypeInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UPositionMarginChangeHistory(t.Context(), "BTCUSDT", "add", 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPositionsInfoV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UPositionsInfoV2(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetCommissionRates(t *testing.T) {
	t.Parallel()
	_, err := b.UGetCommissionRates(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UGetCommissionRates(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUSDTUserRateLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUSDTUserRateLimits(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDownloadIDForFuturesTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDownloadIDForFuturesTransactionHistory(t.Context(), time.Now().Add(-time.Hour*24*6), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTransactionHistoryDownloadLinkByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesTransactionHistoryDownloadLinkByID(t.Context(), "download-id-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderHistoryDownloadLinkByID(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesOrderHistoryDownloadLinkByID(t.Context(), "")
	require.ErrorIs(t, err, errDownloadIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesOrderHistoryDownloadLinkByID(t.Context(), "download-id-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTradeDownloadLinkByID(t *testing.T) {
	t.Parallel()
	_, err := b.FuturesTradeDownloadLinkByID(t.Context(), "")
	require.ErrorIs(t, err, errDownloadIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesTradeDownloadLinkByID(t.Context(), "download-id-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesOrderHistoryDownloadID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UFuturesOrderHistoryDownloadID(t.Context(), time.Now().Add(-time.Hour*24*6), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTradeHistoryDownloadID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesTradeHistoryDownloadID(t.Context(), time.Now().Add(-time.Hour*24*6), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountTradesHistory(t *testing.T) {
	t.Parallel()
	_, err := b.UAccountTradesHistory(t.Context(), "", "", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAccountTradesHistory(t.Context(), "BTCUSDT", "", 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountIncomeHistory(t *testing.T) {
	t.Parallel()
	_, err := b.UAccountIncomeHistory(t.Context(), "", "something-else", 5, time.Now().Add(-time.Hour*48), time.Now())
	require.ErrorIs(t, err, errIncomeTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAccountIncomeHistory(t.Context(), "BTCUSDT", "", 5, time.Now().Add(-time.Hour*48), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UGetNotionalAndLeverageBrackets(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UPositionsADLEstimate(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountForcedOrders(t *testing.T) {
	t.Parallel()
	_, err := b.UAccountForcedOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "something-else", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidAutoCloseType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAccountForcedOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "ADL", 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesTradingWuantitativeRulesIndicators(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UFuturesTradingWuantitativeRulesIndicators(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := b.FuturesExchangeInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesOrderbook(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPublicTrades(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesPublicTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPastPublicTrades(t *testing.T) {
	t.Parallel()
	result, err := b.GetPastPublicTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTradesList(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesAggregatedTradesList(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 0, 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPerpsExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := b.GetPerpMarkets(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexAndMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := b.GetIndexAndMarkPrice(t.Context(), "", "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesKlineData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1Mo", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	result, err := b.GetFuturesKlineData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetFuturesKlineData(t.Context(), currency.NewPairWithDelimiter("LTCUSD", "PERP", "_"), "5m", 5, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContinuousKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.GetContinuousKlineData(t.Context(), "", "CURRENT_QUARTER", "1M", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.GetContinuousKlineData(t.Context(), "BTCUSD", "", "1M", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errContractTypeIsRequired)

	_, err = b.GetContinuousKlineData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	result, err := b.GetContinuousKlineData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "1M", 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	_, err = b.GetContinuousKlineData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "1M", 5, start, end)
	assert.NoError(t, err)
}

func TestGetIndexPriceKlines(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexPriceKlines(t.Context(), "BTCUSD", "", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	result, err := b.GetIndexPriceKlines(t.Context(), "BTCUSD", "1M", 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	_, err = b.GetIndexPriceKlines(t.Context(), "BTCUSD", "1M", 5, start, end)
	assert.NoError(t, err)
}

func TestGetFuturesSwapTickerChangeStats(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesSwapTickerChangeStats(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetFuturesSwapTickerChangeStats(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetFuturesSwapTickerChangeStats(t.Context(), currency.EMPTYPAIR, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesGetFundingHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.FuturesGetFundingHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 50, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesHistoricalTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 5)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetFuturesHistoricalTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesSymbolPriceTicker(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderbookTicker(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesOrderbookTicker(t.Context(), currency.EMPTYPAIR, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetFuturesOrderbookTicker(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCFuturesIndexPriceConstituents(t *testing.T) {
	t.Parallel()
	_, err := b.GetCFuturesIndexPriceConstituents(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.GetCFuturesIndexPriceConstituents(t.Context(), currency.NewPair(currency.BTC, currency.USD))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOpenInterest(t *testing.T) {
	t.Parallel()
	result, err := b.OpenInterest(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCFuturesQuarterlyContractSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := b.CFuturesQuarterlyContractSettlementPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := b.CFuturesQuarterlyContractSettlementPrice(t.Context(), currency.NewPair(currency.BTC, currency.USD))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterestStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenInterestStats(t.Context(), "BTCUSD", "QUARTER", "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errContractTypeIsRequired)

	_, err = b.GetOpenInterestStats(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5mo", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.GetOpenInterestStats(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetOpenInterestStats(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTraderFuturesAccountRatio(t *testing.T) {
	t.Parallel()
	_, err := b.GetTraderFuturesAccountRatio(t.Context(), currency.EMPTYPAIR, "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.GetTraderFuturesAccountRatio(t.Context(), usdtmTradablePair, "", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.GetTraderFuturesAccountRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetTraderFuturesAccountRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTraderFuturesPositionsRatio(t *testing.T) {
	t.Parallel()
	_, err := b.GetTraderFuturesPositionsRatio(t.Context(), currency.EMPTYPAIR, "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.GetTraderFuturesPositionsRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5mo", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.GetTraderFuturesPositionsRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetTraderFuturesPositionsRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketRatio(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketRatio(t.Context(), currency.EMPTYPAIR, "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.GetMarketRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5mo", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.GetMarketRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetMarketRatio(t.Context(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTakerVolume(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesTakerVolume(t.Context(), currency.EMPTYPAIR, "ALL", "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.GetFuturesTakerVolume(t.Context(), currency.NewPair(currency.BTC, currency.USD), "abc", "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errContractTypeIsRequired)

	_, err = b.GetFuturesTakerVolume(t.Context(), currency.NewPair(currency.BTC, currency.USD), "ALL", "5mo", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	result, err := b.GetFuturesTakerVolume(t.Context(), currency.NewPair(currency.BTC, currency.USD), "ALL", "5m", 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetFuturesTakerVolume(t.Context(), currency.NewPair(currency.BTC, currency.USD), "ALL", "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesBasisData(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesBasisData(t.Context(), currency.EMPTYPAIR, "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = b.GetFuturesBasisData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "QUARTER", "5m", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errContractTypeIsRequired)
	_, err = b.GetFuturesBasisData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "CURRENT_QUARTER", "5mo", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval)

	result, err := b.GetFuturesBasisData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start := time.UnixMilli(1577836800000)
	end := time.UnixMilli(1580515200000)
	if !mockTests {
		start = time.Now().Add(-time.Second * 240)
		end = time.Now()
	}
	result, err = b.GetFuturesBasisData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "CURRENT_QUARTER", "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesNewOrder(t *testing.T) {
	t.Parallel()
	arg := &FuturesNewOrderRequest{Symbol: usdtmTradablePair, Side: "BUY", OrderType: order.Limit.String(), PositionSide: "abcd", TimeInForce: order.GoodTillCancel.String(), Quantity: 1, Price: 1}
	_, err := b.FuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPositionSide)

	arg.PositionSide = ""
	arg.WorkingType = "abc"
	_, err = b.FuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidWorkingType)

	arg.WorkingType = ""
	arg.NewOrderRespType = "abcd"
	_, err = b.FuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidNewOrderResponseType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesNewOrder(t.Context(), &FuturesNewOrderRequest{Symbol: currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), Side: "BUY", OrderType: order.Limit.String(), TimeInForce: order.GoodTillCancel.String(), Quantity: 1, Price: 1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesBatchOrder(t *testing.T) {
	t.Parallel()
	_, err := b.FuturesBatchOrder(t.Context(), []PlaceBatchOrderData{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := PlaceBatchOrderData{
		Symbol:       currency.Pair{Base: currency.BTC, Quote: currency.NewCode("USD_PERP")},
		Side:         "BUY",
		OrderType:    order.Limit.String(),
		Quantity:     1,
		Price:        1,
		TimeInForce:  "GTC",
		PositionSide: "abcd",
	}
	_, err = b.FuturesBatchOrder(t.Context(), []PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidPositionSide)

	arg.PositionSide = ""
	arg.WorkingType = "abcd"
	_, err = b.FuturesBatchOrder(t.Context(), []PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidWorkingType)

	arg.NewOrderRespType = "abcd"
	arg.WorkingType = ""
	_, err = b.FuturesBatchOrder(t.Context(), []PlaceBatchOrderData{arg})
	require.ErrorIs(t, err, errInvalidNewOrderResponseType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesBatchOrder(t.Context(), []PlaceBatchOrderData{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesBatchCancelOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), []string{"123"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesGetOrderData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesGetOrderData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesCancelAllOpenOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.AutoCancelAllOpenOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 30000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesOpenOrderData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesOpenOrderData(t.Context(), currency.NewPair(currency.BTC, currency.USD), "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesAllOpenOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllFuturesOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesChangeMarginType(t *testing.T) {
	t.Parallel()
	_, err := b.FuturesChangeMarginType(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "abcd")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesChangeMarginType(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "ISOLATED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesAccountBalance(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesAccountInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesChangeInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := b.FuturesChangeInitialLeverage(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 129)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesChangeInitialLeverage(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyIsolatedPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := b.ModifyIsolatedPositionMargin(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", "abcd", 0)
	require.ErrorIs(t, err, errMarginChangeTypeInvalid)

	_, err = b.ModifyIsolatedPositionMargin(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "abcd", "", 0)
	require.ErrorIs(t, err, errInvalidPositionSide)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ModifyIsolatedPositionMargin(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "BOTH", "add", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesMarginChangeHistory(t *testing.T) {
	t.Parallel()
	_, err := b.FuturesMarginChangeHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "abc", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMarginChangeTypeInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesMarginChangeHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "add", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesPositionsInfo(t.Context(), "BTCUSD", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesTradeHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", time.Time{}, time.Time{}, 5, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesIncomeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesIncomeHistory(t.Context(), currency.EMPTYPAIR, "TRANSFER", time.Time{}, time.Time{}, 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesForceOrders(t *testing.T) {
	t.Parallel()
	_, err := b.FuturesForceOrders(t.Context(), currency.EMPTYPAIR, "abcd", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidAutoCloseType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesForceOrders(t.Context(), currency.EMPTYPAIR, "ADL", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetNotionalLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesNotionalBracket(t.Context(), "BTCUSD")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.FuturesNotionalBracket(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesPositionsADLEstimate(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkPriceKline(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1Mo", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	result, err := b.GetMarkPriceKline(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	result, err := b.GetPremiumIndexKlineData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := b.GetExchangeInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
	if !mockTests {
		assert.WithinRange(t, result.ServerTime.Time(), time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), "ServerTime should be within a day of now")
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := b.FetchTradablePairs(t.Context(), asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	assetTypes := b.GetAssetTypes(true)
	for a := range assetTypes {
		results, err := b.FetchTradablePairs(t.Context(), assetTypes[a])
		assert.NoError(t, err)
		assert.NotNil(t, results)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	result, err := b.GetOrderBook(t.Context(),
		OrderBookDataRequestParams{
			Symbol: currency.NewBTCUSDT(),
			Limit:  1000,
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMostRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetMostRecentTrades(t.Context(), &RecentTradeRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	result, err := b.GetMostRecentTrades(t.Context(), &RecentTradeRequestParams{Symbol: currency.NewPair(currency.BTC, currency.USDT), Limit: 15})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetHistoricalTrades(t.Context(), "", 5, -1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.GetHistoricalTrades(t.Context(), "BTCUSDT", 5, -1)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetAggregatedTrades(t.Context(), &AggregatedTradeRequestParams{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.GetAggregatedTrades(t.Context(), &AggregatedTradeRequestParams{Symbol: "BTCUSDT", Limit: 5})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	result, err := b.GetSpotKline(t.Context(), &KlinesRequestParams{Symbol: currency.NewPair(currency.BTC, currency.USDT), Interval: kline.FiveMin.Short(), Limit: 24, StartTime: start, EndTime: end})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUIKline(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	result, err := b.GetUIKline(t.Context(), &KlinesRequestParams{Symbol: currency.NewPair(currency.BTC, currency.USDT), Interval: kline.FiveMin.Short(), Limit: 24, StartTime: start, EndTime: end})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()
	result, err := b.GetAveragePrice(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()
	result, err := b.GetPriceChangeStats(t.Context(), currency.NewPair(currency.BTC, currency.USDT), currency.Pairs{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradingDayTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTradingDayTicker(t.Context(), []currency.Pair{}, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)
	_, err = b.GetTradingDayTicker(t.Context(), []currency.Pair{currency.EMPTYPAIR}, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := b.GetTradingDayTicker(t.Context(), []currency.Pair{spotTradablePair}, "", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	result, err := b.GetLatestSpotPrice(t.Context(), currency.NewPair(currency.BTC, currency.USDT), currency.Pairs{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()
	result, err := b.GetBestPrice(t.Context(), spotTradablePair, currency.Pairs{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickerData(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickerData(t.Context(), []currency.Pair{}, time.Minute*20, "FULL")
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	result, err := b.GetTickerData(t.Context(), []currency.Pair{spotTradablePair}, time.Minute*20, "FULL")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	t.Parallel()
	_, err := b.GetDepositAddressForCurrency(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDepositAddressForCurrency(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetsThatCanBeConvertedIntoBNB(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAssetsThatCanBeConvertedIntoBNB(t.Context(), "MINI")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawCrypto(t.Context(), currency.EMPTYCODE, "123435", "", "address-here", "", "", 100, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.WithdrawCrypto(t.Context(), currency.USDT, "", "", "", "", "", 100, false)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = b.WithdrawCrypto(t.Context(), currency.USDT, "123435", "", "address-here", "123213", "", 0, false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.WithdrawCrypto(t.Context(), currency.USDT, "123435", "", "address", "", "", 100, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDustTransfer(t *testing.T) {
	t.Parallel()
	_, err := b.DustTransfer(t.Context(), []string{}, "SPOT")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.DustTransfer(t.Context(), []string{"BTC", "USDT"}, "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetDevidendRecords(t *testing.T) {
	t.Parallel()
	_, err := b.GetAssetDevidendRecords(t.Context(), currency.EMPTYCODE, time.Now().Add(-time.Hour*48), time.Now(), 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAssetDevidendRecords(t.Context(), currency.BTC, time.Now().Add(-time.Hour*48), time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAssetDetail(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeFees(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetTradeFees(t.Context(), currency.Pair{Base: currency.BTC, Quote: currency.USDT})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUserUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := b.UserUniversalTransfer(t.Context(), 0, 123.234, currency.BTC, "", "")
	require.ErrorIs(t, err, errTransferTypeRequired)
	_, err = b.UserUniversalTransfer(t.Context(), ttMainUMFuture, 123.234, currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.UserUniversalTransfer(t.Context(), ttMainUMFuture, 0, currency.BTC, "", "")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UserUniversalTransfer(t.Context(), ttMainUMFuture, 123.234, currency.BTC, "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserUniversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetUserUniversalTransferHistory(t.Context(), 0, time.Time{}, time.Time{}, 0, 0, "BTC", "USDT")
	require.ErrorIs(t, err, errTransferTypeRequired)
	_, err = b.GetUserUniversalTransferHistory(t.Context(), ttUMFutureMargin, time.Time{}, time.Time{}, 0, 0, "BTC", "USDT")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserUniversalTransferHistory(t.Context(), ttUMFutureMargin, time.Time{}, time.Time{}, 1, 1234, "BTC", "USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFundingAssets(t.Context(), currency.BTC, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserAssets(t.Context(), currency.BTC, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConvertBUSD(t *testing.T) {
	t.Parallel()
	_, err := b.ConvertBUSD(t.Context(), "", "MAIN", currency.ETH, currency.USD, 1234)
	require.ErrorIs(t, err, errTransactionIDRequired)
	_, err = b.ConvertBUSD(t.Context(), "12321412312", "MAIN", currency.EMPTYCODE, currency.USD, 1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.ConvertBUSD(t.Context(), "12321412312", "MAIN", currency.ETH, currency.USD, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.ConvertBUSD(t.Context(), "12321412312", "MAIN", currency.ETH, currency.EMPTYCODE, 1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ConvertBUSD(t.Context(), "12321412312", "MAIN", currency.ETH, currency.USD, 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBUSDConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.BUSDConvertHistory(t.Context(), "transaction-id", "233423423", "CARD", currency.BTC, time.Now().Add(-time.Hour*48*10), time.Now(), 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCloudMiningPaymentAndRefundHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCloudMiningPaymentAndRefundHistory(t.Context(), "1234", currency.BTC, time.Now().Add(-time.Hour*480), time.Now(), 1232313, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserAccountInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAPIKeyPermission(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAPIKeyPermission(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoConvertingStableCoins(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAutoConvertingStableCoins(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSwitchOnOffBUSDAndStableCoinsConversion(t *testing.T) {
	t.Parallel()
	err := b.SwitchOnOffBUSDAndStableCoinsConversion(t.Context(), currency.EMPTYCODE, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.SwitchOnOffBUSDAndStableCoinsConversion(t.Context(), currency.BTC, false)
	assert.NoError(t, err)
}

func TestOneClickArrivalDepositApply(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.OneClickArrivalDepositApply(t.Context(), "", 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressListWithNetwork(t *testing.T) {
	t.Parallel()
	_, err := b.GetDepositAddressListWithNetwork(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDepositAddressListWithNetwork(t.Context(), currency.BTC, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserWalletBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserWalletBalance(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserDelegationHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetUserDelegationHistory(t.Context(), "", "Delegate", time.Now().Add(-time.Hour*24*12), time.Now(), currency.BTC, 0, 0)
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserDelegationHistory(t.Context(), "someone@thrasher.com", "Delegate", time.Now().Add(-time.Hour*24*12), time.Now(), currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolsDelistScheduleForSpot(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSymbolsDelistScheduleForSpot(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateVirtualSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CreateVirtualSubAccount(t.Context(), "something-string")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountList(t.Context(), "testsub@gmail.com", false, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountSpotAssetTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountSpotAssetTransferHistory(t.Context(), "", "", time.Time{}, time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountFuturesAssetTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetSubAccountFuturesAssetTransferHistory(t.Context(), "", time.Time{}, time.Now(), 2, 0, 0)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.GetSubAccountFuturesAssetTransferHistory(t.Context(), "someone@gmail.com", time.Time{}, time.Now(), 0, 0, 0)
	require.ErrorIs(t, err, errInvalidFuturesType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountFuturesAssetTransferHistory(t.Context(), "someone@gmail.com", time.Time{}, time.Now(), 2, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountFuturesAssetTransfer(t *testing.T) {
	t.Parallel()
	_, err := b.SubAccountFuturesAssetTransfer(t.Context(), "from_someone", "to_someont@thrasher.io", 1, currency.USDT, 0.1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.SubAccountFuturesAssetTransfer(t.Context(), "from_someone@thrasher.io", "to_someont", 1, currency.USDT, 0.1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.SubAccountFuturesAssetTransfer(t.Context(), "from_someone@thrasher.io", "to_someont@thrasher.io", -1, currency.USDT, 0.1)
	require.ErrorIs(t, err, errInvalidFuturesType)
	_, err = b.SubAccountFuturesAssetTransfer(t.Context(), "from_someone@thrasher.io", "to_someont@thrasher.io", 1, currency.EMPTYCODE, 0.1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubAccountFuturesAssetTransfer(t.Context(), "from_someone@thrasher.io", "to_someont@thrasher.io", 1, currency.USDT, 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAssets(t *testing.T) {
	t.Parallel()
	_, err := b.GetSubAccountAssets(t.Context(), "email_address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountAssets(t.Context(), "email_address@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountList(t.Context(), "address@gmail.com", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountTransactionStatistics(t *testing.T) {
	t.Parallel()
	_, err := b.GetSubAccountTransactionStatistics(t.Context(), "addressio")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountTransactionStatistics(t.Context(), "address@thrasher.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := b.GetManagedSubAccountDepositAddress(t.Context(), currency.ETH, "destination", "")
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.GetManagedSubAccountDepositAddress(t.Context(), currency.EMPTYCODE, "destination@thrasher.io", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountDepositAddress(t.Context(), currency.ETH, "destination@thrasher.io", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableOptionsForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.EnableOptionsForSubAccount(t.Context(), "")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableOptionsForSubAccount(t.Context(), "address@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountTransferLog(t *testing.T) {
	t.Parallel()
	_, err := b.GetManagedSubAccountTransferLog(t.Context(), time.Now().Add(-time.Hour*24*30), time.Now().Add(-time.Hour*24*50), 1, 10, "", "MARGIN")
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = b.GetManagedSubAccountTransferLog(t.Context(), time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*30), -1, 10, "", "MARGIN")
	require.ErrorIs(t, err, errPageNumberRequired)
	_, err = b.GetManagedSubAccountTransferLog(t.Context(), time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*30), 1, -1, "", "MARGIN")
	require.ErrorIs(t, err, errLimitNumberRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountTransferLog(t.Context(), time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*30), 1, 10, "", "MARGIN")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountSpotAssetsSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountSpotAssetsSummary(t.Context(), "the_address@thrasher.io", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := b.GetSubAccountDepositAddress(t.Context(), "", "BTC", "", 0.1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.GetSubAccountDepositAddress(t.Context(), "the_address@thrasher.io", "", "", 0.1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountDepositAddress(t.Context(), "the_address@thrasher.io", "BTC", "", 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetSubAccountDepositHistory(t.Context(), "someoneio", "BTC", time.Time{}, time.Now(), 0, 0, 10)
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountDepositHistory(t.Context(), "someone@thrasher.io", "BTC", time.Time{}, time.Now(), 0, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountStatusOnMarginFutures(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountStatusOnMarginFutures(t.Context(), "myemail@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableMarginForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.EnableMarginForSubAccount(t.Context(), "sampleemaicom")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableMarginForSubAccount(t.Context(), "sampleemail@email.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailOnSubAccountMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := b.GetDetailOnSubAccountMarginAccount(t.Context(), "com")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDetailOnSubAccountMarginAccount(t.Context(), "test@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfSubAccountMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSummaryOfSubAccountMarginAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableFuturesSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.EnableFuturesSubAccount(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableFuturesSubAccount(t.Context(), "address@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailSubAccountFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := b.GetDetailSubAccountFuturesAccount(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDetailSubAccountFuturesAccount(t.Context(), "address@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfSubAccountFuturesAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetSummaryOfSubAccountFuturesAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV1FuturesPositionRiskSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.GetV1FuturesPositionRiskSubAccount(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetV1FuturesPositionRiskSubAccount(t.Context(), "address@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositionRiskSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.GetV2FuturesPositionRiskSubAccount(t.Context(), "address", 1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.GetV2FuturesPositionRiskSubAccount(t.Context(), "address@mail.com", -1)
	require.ErrorIs(t, err, errInvalidFuturesType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetV2FuturesPositionRiskSubAccount(t.Context(), "address@mail.com", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableLeverageTokenForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.EnableLeverageTokenForSubAccount(t.Context(), "email-address", false)
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableLeverageTokenForSubAccount(t.Context(), "someone@thrasher.io", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := b.GetIPRestrictionForSubAccountAPIKeyV2(t.Context(), "emailaddress", apiKey)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.GetIPRestrictionForSubAccountAPIKeyV2(t.Context(), "emailaddress@thrasher.io", "")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIPRestrictionForSubAccountAPIKeyV2(t.Context(), "emailaddress@thrasher.io", apiKey)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteIPListForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := b.DeleteIPListForSubAccountAPIKey(t.Context(), "emailaddress", apiKey, "196.168.4.1")
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.DeleteIPListForSubAccountAPIKey(t.Context(), "emailaddress@thrasher.io", "", "196.168.4.1")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.DeleteIPListForSubAccountAPIKey(t.Context(), "emailaddress@thrasher.io", apiKey, "196.168.4.1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAddIPRestrictionForSubAccountAPIkey(t *testing.T) {
	t.Parallel()
	_, err := b.AddIPRestrictionForSubAccountAPIkey(t.Context(), "addressthrasher", apiKey, "", true)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.AddIPRestrictionForSubAccountAPIkey(t.Context(), "address@thrasher.io", "", "", true)
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.AddIPRestrictionForSubAccountAPIkey(t.Context(), "address@thrasher.io", apiKey, "", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDepositAssetsIntoTheManagedSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail", currency.BTC, 0.0001)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail@mail.com", currency.EMPTYCODE, 0.0001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail@mail.com", currency.BTC, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail@mail.com", currency.BTC, 0.0001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountAssetsDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetManagedSubAccountAssetsDetails(t.Context(), "emailaddress")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetManagedSubAccountAssetsDetails(t.Context(), "emailaddress@thrashser.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawAssetsFromManagedSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawAssetsFromManagedSubAccount(t.Context(), "source", currency.BTC, 0.0000001, time.Now().Add(-time.Hour*24*50))
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.WithdrawAssetsFromManagedSubAccount(t.Context(), "source@email.com", currency.EMPTYCODE, 0.0000001, time.Now().Add(-time.Hour*24*50))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.WithdrawAssetsFromManagedSubAccount(t.Context(), "source@email.com", currency.BTC, 0, time.Now().Add(-time.Hour*24*50))
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.WithdrawAssetsFromManagedSubAccount(t.Context(), "source@email.com", currency.BTC, 0.0000001, time.Now().Add(-time.Hour*24*50))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountSnapshot(t *testing.T) {
	t.Parallel()
	_, err := b.GetManagedSubAccountSnapshot(t.Context(), "address", "SPOT", time.Time{}, time.Now(), 10)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.GetManagedSubAccountSnapshot(t.Context(), "address@thrasher.io", "", time.Time{}, time.Now(), 10)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountSnapshot(t.Context(), "address@thrasher.io", "SPOT", time.Time{}, time.Now(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountTransferLogForInvestorMasterAccount(t *testing.T) {
	t.Parallel()
	_, err := b.GetManagedSubAccountTransferLogForInvestorMasterAccount(t.Context(), "address.com", "TO", "SPOT", time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*20), 1, 10)
	require.ErrorIs(t, err, errValidEmailRequired)

	_, err = b.GetManagedSubAccountTransferLogForInvestorMasterAccount(t.Context(), "address@gmail.com", "TO", "SPOT", time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*20), -1, 10)
	require.ErrorIs(t, err, errPageNumberRequired)

	_, err = b.GetManagedSubAccountTransferLogForInvestorMasterAccount(t.Context(), "address@gmail.com", "TO", "SPOT", time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*20), 1, 0)
	require.ErrorIs(t, err, errLimitNumberRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountTransferLogForInvestorMasterAccount(t.Context(), "address@gmail.com", "TO", "SPOT", time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*20), 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountTransferLogForTradingTeam(t *testing.T) {
	t.Parallel()
	_, err := b.GetManagedSubAccountTransferLogForTradingTeam(t.Context(), "address", "FROM", "ISOLATED_MARGIN", time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*20), 1, 10)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.GetManagedSubAccountTransferLogForTradingTeam(t.Context(), "address@gmail.com", "FROM", "ISOLATED_MARGIN", time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*20), -1, 10)
	require.ErrorIs(t, err, errPageNumberRequired)
	_, err = b.GetManagedSubAccountTransferLogForTradingTeam(t.Context(), "address@gmail.com", "FROM", "ISOLATED_MARGIN", time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*20), 1, 0)
	require.ErrorIs(t, err, errLimitNumberRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountTransferLogForTradingTeam(t.Context(), "address@gmail.com", "FROM", "ISOLATED_MARGIN", time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*20), 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountFutureesAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetManagedSubAccountFutureesAssetDetails(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountFutureesAssetDetails(t.Context(), "address@email.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountMarginAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetManagedSubAccountMarginAssetDetails(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountMarginAssetDetails(t.Context(), "address@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTransferSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.FuturesTransferSubAccount(t.Context(), "someone.com", currency.BTC, 1.1, 1)
	require.ErrorIs(t, err, errValidEmailRequired)

	_, err = b.FuturesTransferSubAccount(t.Context(), "someone@mail.com", currency.EMPTYCODE, 1.1, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.FuturesTransferSubAccount(t.Context(), "someone@mail.com", currency.BTC, 0, 1)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	_, err = b.FuturesTransferSubAccount(t.Context(), "someone@mail.com", currency.BTC, 1.1, 0)
	require.ErrorIs(t, err, errTransferTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesTransferSubAccount(t.Context(), "someone@mail.com", currency.BTC, 1.1, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginTransferForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.MarginTransferForSubAccount(t.Context(), "someone", currency.BTC, 1.1, 1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.MarginTransferForSubAccount(t.Context(), "someone@mail.com", currency.EMPTYCODE, 1.1, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.MarginTransferForSubAccount(t.Context(), "someone@mail.com", currency.BTC, 0, 1)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.MarginTransferForSubAccount(t.Context(), "someone@mail.com", currency.BTC, 1.1, -1)
	require.ErrorIs(t, err, errTransferTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.MarginTransferForSubAccount(t.Context(), "someone@mail.com", currency.BTC, 1.1, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAssetsV3(t *testing.T) {
	t.Parallel()
	_, err := b.GetSubAccountAssetsV3(t.Context(), "")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountAssetsV3(t.Context(), "someone@thrasher.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferToSubAccountOfSameMaster(t *testing.T) {
	t.Parallel()
	_, err := b.TransferToSubAccountOfSameMaster(t.Context(), "thrasher", currency.ETH, 10)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.TransferToSubAccountOfSameMaster(t.Context(), "toEmail@thrasher.io", currency.EMPTYCODE, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.TransferToSubAccountOfSameMaster(t.Context(), "toEmail@thrasher.io", currency.ETH, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.TransferToSubAccountOfSameMaster(t.Context(), "toEmail@thrasher.io", currency.ETH, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFromSubAccountTransferToMaster(t *testing.T) {
	t.Parallel()
	_, err := b.FromSubAccountTransferToMaster(t.Context(), currency.EMPTYCODE, 0.1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.FromSubAccountTransferToMaster(t.Context(), currency.LTC, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FromSubAccountTransferToMaster(t.Context(), currency.LTC, 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubAccountTransferHistory(t.Context(), currency.BTC, 1, 10, time.Time{}, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransferHistoryForSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubAccountTransferHistoryForSubAccount(t.Context(), currency.LTC, 2, 0, time.Time{}, time.Now(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUniversalTransferForMasterAccount(t *testing.T) {
	t.Parallel()
	_, err := b.UniversalTransferForMasterAccount(t.Context(), &UniversalTransferParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &UniversalTransferParams{
		ClientTransactionID: "transaction-id",
	}
	_, err = b.UniversalTransferForMasterAccount(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAccountType)

	arg.ToAccountType = "SPOT"
	_, err = b.UniversalTransferForMasterAccount(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAccountType)

	arg.FromAccountType = "ISOLATED_MARGIN"
	_, err = b.UniversalTransferForMasterAccount(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Asset = currency.BTC
	_, err = b.UniversalTransferForMasterAccount(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UniversalTransferForMasterAccount(t.Context(), &UniversalTransferParams{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUniversalTransferHistoryForMasterAccount(t.Context(), "", "", "", time.Time{}, time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailOnSubAccountsFuturesAccountV2(t *testing.T) {
	t.Parallel()
	_, err := b.GetDetailOnSubAccountsFuturesAccountV2(t.Context(), "thrasher", 1)
	require.ErrorIs(t, err, errValidEmailRequired)
	_, err = b.GetDetailOnSubAccountsFuturesAccountV2(t.Context(), "address@thrasher.io", 0)
	require.ErrorIs(t, err, errInvalidFuturesType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDetailOnSubAccountsFuturesAccountV2(t.Context(), "address@thrasher.io", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfSubAccountsFuturesAccountV2(t *testing.T) {
	t.Parallel()
	_, err := b.GetSummaryOfSubAccountsFuturesAccountV2(t.Context(), 0, 0, 10)
	require.ErrorIs(t, err, errInvalidFuturesType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSummaryOfSubAccountsFuturesAccountV2(t.Context(), 1, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.QueryOrder(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", 1337)
	require.False(t, sharedtestvalues.AreAPICredentialsSet(b) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests, "expecting an error when no keys are set")
	assert.False(t, mockTests && err != nil, err)
}

func TestCancelExistingOrderAndSendNewOrder(t *testing.T) {
	t.Parallel()
	_, err := b.CancelExistingOrderAndSendNewOrder(t.Context(), &CancelReplaceOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &CancelReplaceOrderParams{
		TimeInForce: "GTC",
	}
	_, err = b.CancelExistingOrderAndSendNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTCUSDT"
	_, err = b.CancelExistingOrderAndSendNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = b.CancelExistingOrderAndSendNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = b.CancelExistingOrderAndSendNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errCancelReplaceModeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelExistingOrderAndSendNewOrder(t.Context(), &CancelReplaceOrderParams{
		Symbol:            "BTCUSDT",
		Side:              "BUY",
		OrderType:         order.Limit.String(),
		CancelReplaceMode: "STOP_ON_FAILURE",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.OpenOrders(t.Context(), currency.EMPTYPAIR)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	p := currency.NewPair(currency.BTC, currency.USDT)
	result, err = b.OpenOrders(t.Context(), p)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrderOnSymbol(t *testing.T) {
	t.Parallel()
	_, err := b.CancelAllOpenOrderOnSymbol(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllOpenOrderOnSymbol(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.AllOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", "")
	require.False(t, sharedtestvalues.AreAPICredentialsSet(b) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests, "expecting an error when no keys are set")
	assert.False(t, mockTests && err != nil, err)
}

func TestNewOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := b.NewOCOOrder(t.Context(), &OCOOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &OCOOrderParam{
		TrailingDelta: 1,
	}
	_, err = b.NewOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Symbol = currency.NewPair(currency.BTC, currency.USDT)
	_, err = b.NewOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Buy"
	_, err = b.NewOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Amount = 0.1
	_, err = b.NewOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 0.001
	_, err = b.NewOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewOCOOrder(t.Context(), &OCOOrderParam{
		Symbol:             currency.NewPair(currency.BTC, currency.USDT),
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
	_, err := b.CancelOCOOrder(t.Context(), "", "", "newderID", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.CancelOCOOrder(t.Context(), "LTCBTC", "", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelOCOOrder(t.Context(), "LTCBTC", "", "newderID", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOCOOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetOCOOrders(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOCOOrders(t.Context(), "123456", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOCOOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetAllOCOOrders(t.Context(), "", time.Now(), time.Now().Add(-time.Hour*24*10), 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllOCOOrders(t.Context(), "", time.Time{}, time.Now().Add(-time.Hour*24*10), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOCOList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOpenOCOList(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOrderUsingSOR(t *testing.T) {
	t.Parallel()
	_, err := b.NewOrderUsingSOR(t.Context(), &SOROrderRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &SOROrderRequestParams{
		TimeInForce: "GTC",
	}
	_, err = b.NewOrderUsingSOR(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = currency.Pair{Base: currency.BTC, Quote: currency.LTC}
	_, err = b.NewOrderUsingSOR(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = b.NewOrderUsingSOR(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = b.NewOrderUsingSOR(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewOrderUsingSOR(t.Context(), &SOROrderRequestParams{
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
	_, err := b.NewOrderUsingSORTest(t.Context(), &SOROrderRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewOrderUsingSORTest(t.Context(), &SOROrderRequestParams{
		Symbol:    currency.Pair{Base: currency.BTC, Quote: currency.LTC},
		Side:      "Buy",
		OrderType: order.Limit.String(),
		Quantity:  0.001,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	_, err := b.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	feeBuilder := setFeeBuilder()
	result, err := b.GetFeeByType(t.Context(), feeBuilder)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	if !sharedtestvalues.AreAPICredentialsSet(b) || mockTests {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	if sharedtestvalues.AreAPICredentialsSet(b) && mockTests {
		// CryptocurrencyTradeFee Basic
		_, err := b.GetFee(t.Context(), feeBuilder)
		require.NoError(t, err)

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		_, err = b.GetFee(t.Context(), feeBuilder)
		require.NoError(t, err)

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		_, err = b.GetFee(t.Context(), feeBuilder)
		require.NoError(t, err)

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		_, err = b.GetFee(t.Context(), feeBuilder)
		require.NoError(t, err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err := b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = b.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err)
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := b.FormatWithdrawPermissions()
	require.Equal(t, expectedResult, withdrawPermissions)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     currency.Pairs{pair},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	result, err := b.GetActiveOrders(t.Context(), &getOrdersRequest)
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
	_, err := b.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	getOrdersRequest.Pairs = []currency.Pair{
		currency.NewPair(currency.LTC,
			currency.BTC),
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOrderTest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.NewOrderTest(t.Context(), &NewOrderRequest{
		Symbol:      currency.NewPair(currency.LTC, currency.BTC),
		Side:        order.Buy.String(),
		TradeType:   order.Limit.String(),
		Price:       0.0025,
		Quantity:    100000,
		TimeInForce: order.GoodTillCancel.String(),
	}, false)
	require.NoError(t, err)

	err = b.NewOrderTest(t.Context(), &NewOrderRequest{
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
	start := time.Unix(1577977445, 0)  // 2020-01-02 15:04:05
	end := start.Add(15 * time.Minute) // 2020-01-02 15:19:05
	if b.IsAPIStreamConnected() {
		start = time.Now().Add(-time.Hour * 10)
		end = time.Now().Add(-time.Hour)
	}
	result, err := b.GetHistoricTrades(t.Context(), p, asset.Spot, start, end)
	require.NoError(t, err)
	expected := 2134
	if b.IsAPIStreamConnected() {
		expected = len(result)
	} else if mockTests {
		expected = 1002
	}
	require.Equal(t, expected, len(result), "GetHistoricTrades should return correct number of entries")
	for _, r := range result {
		if !assert.WithinRange(t, r.Timestamp, start, end, "All trades should be within time range") {
			break
		}
	}
	result, err = b.GetHistoricTrades(t.Context(), optionsTradablePair, asset.Options, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTradesBatched(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTCUSDT")
	require.NoError(t, err)

	start, err := time.Parse(time.RFC3339, "2020-01-02T15:04:05Z")
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
				Symbol:    currencyPair.String(),
				StartTime: start,
				EndTime:   start.Add(75 * time.Minute),
			},
			numExpected:  1012,
			lastExpected: time.Date(2020, 1, 2, 16, 18, 31, int(919*time.Millisecond), time.UTC),
		},
		{
			name: "batch with timerange",
			args: &AggregatedTradeRequestParams{
				Symbol:    currencyPair.String(),
				StartTime: start,
				EndTime:   start.Add(75 * time.Minute),
			},
			numExpected:  12130,
			lastExpected: expectTime,
		},
		{
			name: "mock custom limit with start time set, no end time",
			mock: true,
			args: &AggregatedTradeRequestParams{
				Symbol:    currency.NewPair(currency.BTC, currency.USDT).String(),
				StartTime: start,
				Limit:     1001,
			},
			numExpected:  1001,
			lastExpected: time.Date(2020, 1, 2, 15, 18, 39, int(226*time.Millisecond), time.UTC),
		},
		{
			name: "custom limit with start time set, no end time",
			args: &AggregatedTradeRequestParams{
				Symbol:    "BTCUSDT",
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
				Symbol: "BTCUSDT",
				Limit:  3,
			},
			numExpected:  3,
			lastExpected: time.Date(2020, 1, 2, 16, 19, 5, int(200*time.Millisecond), time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.mock != mockTests {
				t.Skip("mock mismatch, skipping")
			}
			result, err := b.GetAggregatedTrades(t.Context(), tt.args)
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
	start, err := time.Parse(time.RFC3339, "2020-01-02T15:04:05Z")
	require.NoError(t, err)
	tests := []struct {
		name string
		args *AggregatedTradeRequestParams
	}{
		{
			name: "get recent trades does not support custom limit",
			args: &AggregatedTradeRequestParams{
				Symbol: "BTCUSDT",
				Limit:  1001,
			},
		},
		{
			name: "start time and fromId cannot be both set",
			args: &AggregatedTradeRequestParams{
				Symbol:    "BTCUSDT",
				StartTime: start,
				EndTime:   start.Add(75 * time.Minute),
				FromID:    2,
			},
		},
		{
			name: "can't get most recent 5000 (more than 1000 not allowed)",
			args: &AggregatedTradeRequestParams{
				Symbol: "BTCUSDT",
				Limit:  5000,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := b.GetAggregatedTrades(t.Context(), tt.args)
			require.Error(t, err)
		})
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// -----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubmitOrder(t.Context(), &order.Submit{
		Exchange: b.Name,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currency.NewPair(currency.LTC, currency.BTC),
		AssetType: asset.Spot,
	})
	assert.NoError(t, err)
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllOrders(t.Context(), &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      spotTradablePair,
		AssetType: asset.Spot,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.CancelAllOrders(t.Context(), &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      optionsTradablePair,
		AssetType: asset.Options,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
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
			result, err := b.UpdateAccountInfo(t.Context(), assetType)
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestWrapperGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{spotTradablePair},
		AssetType: asset.Spot,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{coinmTradablePair},
		AssetType: asset.CoinMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{usdtmTradablePair},
		AssetType: asset.USDTMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
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
	_, err := b.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures})
	assert.Error(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	p, err := currency.NewPairFromString("EOSUSD_PERP")
	require.NoError(t, err)
	result, err := b.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Type:        order.AnyType,
		Side:        order.AnySide,
		FromOrderID: "123",
		Pairs:       currency.Pairs{p},
		AssetType:   asset.CoinMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	p2, err := currency.NewPairFromString("BTCUSDT")
	require.NoError(t, err)

	result, err = b.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Type:        order.AnyType,
		Side:        order.AnySide,
		FromOrderID: "123",
		Pairs:       currency.Pairs{p2},
		AssetType:   asset.USDTMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	p, err := currency.NewPairFromString("EOS-USDT")
	require.NoError(t, err)
	fPair, err := b.FormatExchangeCurrency(p, asset.CoinMarginedFutures)
	require.NoError(t, err)
	err = b.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.CoinMarginedFutures,
		Pair:      fPair,
		OrderID:   "1234",
	})
	require.NoError(t, err)
	p2, err := currency.NewPairFromString("BTC-USDT")
	require.NoError(t, err)
	fpair2, err := b.FormatExchangeCurrency(p2, asset.USDTMarginedFutures)
	require.NoError(t, err)
	err = b.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.USDTMarginedFutures,
		Pair:      fpair2,
		OrderID:   "1234",
	})
	require.NoError(t, err)
	err = b.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.Options,
		Pair:      fpair2,
		OrderID:   "1234",
	})
	assert.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	tradablePairs, err := b.FetchTradablePairs(t.Context(),
		asset.CoinMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, tradablePairs, "no tradable pairs")
	result, err := b.GetOrderInfo(t.Context(), "123", tradablePairs[0], asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := b.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetAllCoinsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCoinsInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.WithdrawCryptocurrencyFunds(t.Context(),
		&withdraw.Request{
			Exchange:    b.Name,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.DepositHistory(t.Context(), currency.ETH, "", time.Time{}, time.Time{}, 0, 10000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWithdrawalsHistory(t.Context(), currency.ETH, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.Equal(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	require.Equal(t, err, common.ErrFunctionNotSupported)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := b.GetDepositAddress(t.Context(), currency.USDT, "", currency.BNB.String())
	require.False(t, sharedtestvalues.AreAPICredentialsSet(b) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests, "error cannot be nil")
	assert.False(t, mockTests && err != nil, err)
}

func BenchmarkWsHandleData(bb *testing.B) {
	bb.ReportAllocs()
	ap, err := b.CurrencyPairs.GetPairs(asset.Spot, false)
	require.NoError(bb, err)
	err = b.CurrencyPairs.StorePairs(asset.Spot, ap, true)
	require.NoError(bb, err)

	data, err := os.ReadFile("testdata/wsHandleData.json")
	require.NoError(bb, err)
	lines := bytes.Split(data, []byte("\n"))
	require.Len(bb, lines, 8)
	go func() {
		for {
			<-b.Websocket.DataHandler
		}
	}()
	for bb.Loop() {
		for x := range lines {
			require.NoError(bb, b.wsHandleData(lines[x]))
		}
	}
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	b := new(Binance) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	channels, err := b.generateSubscriptions() // Note: We grab this before it's overwritten by MockWsInstance below
	require.NoError(t, err, "generateSubscriptions must not error")
	if mockTests {
		exp := []string{"btcusdt@depth@100ms", "btcusdt@kline_1m", "btcusdt@ticker", "btcusdt@trade", "dogeusdt@depth@100ms", "dogeusdt@kline_1m", "dogeusdt@ticker", "dogeusdt@trade"}
		mock := func(tb testing.TB, msg []byte, w *gws.Conn) error {
			tb.Helper()
			var req WsPayload
			require.NoError(tb, json.Unmarshal(msg, &req), "Unmarshal should not error")
			require.ElementsMatch(tb, req.Params, exp, "Params should have correct channels")
			return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"result":null,"id":%d}`, req.ID))
		}
		b = testexch.MockWsInstance[Binance](t, mockws.CurryWsMockUpgrader(t, mock))
	} else {
		testexch.SetupWs(t, b)
	}
	err = b.Subscribe(channels)
	require.NoError(t, err)
	err = b.Unsubscribe(channels)
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
		require.NoError(tb, err, "Unmarshal should not error")
		return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"result":{"error":"carrots"},"id":%d}`, req.ID))
	}
	b := testexch.MockWsInstance[Binance](t, mockws.CurryWsMockUpgrader(t, mock)) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	err := b.Subscribe(channels)
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "Subscribe should error ErrSubscriptionFailure")
	require.ErrorIs(t, err, common.ErrUnknownError, "Subscribe should error errUnknownError")
	assert.ErrorContains(t, err, "carrots", "Subscribe should error containing the carrots")
}

func TestWsTickerUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"btcusdt@ticker","data":{"e":"24hrTicker","E":1580254809477,"s":"BTCUSDT","p":"420.97000000","P":"4.720","w":"9058.27981278","x":"8917.98000000","c":"9338.96000000","Q":"0.17246300","b":"9338.03000000","B":"0.18234600","a":"9339.70000000","A":"0.14097600","o":"8917.99000000","h":"9373.19000000","l":"8862.40000000","v":"72229.53692000","q":"654275356.16896672","O":1580168409456,"C":1580254809456,"F":235294268,"L":235894703,"n":600436}}`)
	err := b.wsHandleData(pressXToJSON)
	assert.NoError(t, err)
}

func TestWsKlineUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"btcusdt@kline_1m","data":{
	  "e": "kline",
	  "E": 1234567891,   
	  "s": "BTCUSDT",    
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
	err := b.wsHandleData(pressXToJSON)
	assert.NoError(t, err)
}

func TestWsTradeUpdate(t *testing.T) {
	t.Parallel()
	b.SetSaveTradeDataStatus(true)
	pressXToJSON := []byte(`{"stream":"btcusdt@trade","data":{
	  "e": "trade",     
	  "E": 1234567891,   
	  "s": "BTCUSDT",    
	  "t": 12345,       
	  "p": "0.001",     
	  "q": "100",       
	  "b": 88,          
	  "a": 50,          
	  "T": 1234567851,   
	  "m": true,        
	  "M": true         
	}}`)
	err := b.wsHandleData(pressXToJSON)
	assert.NoError(t, err)
}

func TestWsDepthUpdate(t *testing.T) {
	t.Parallel()
	b := new(Binance) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	b.setupOrderbookManager()
	seedLastUpdateID := int64(161)
	book := OrderBook{
		Asks: OrderbookTranches{
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
		},
		Bids: OrderbookTranches{
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
		},
		LastUpdateID: seedLastUpdateID,
	}

	update1 := []byte(`{"stream":"btcusdt@depth","data":{
	  "e": "depthUpdate", 
	  "E": 1234567881,     
	  "s": "BTCUSDT",      
	  "U": 157,           
	  "u": 160,           
	  "b": [              
		["6621.45", "0.3"]
	  ],
	  "a": [              
		["6622.46", "1.5"]
	  ]
	}}`)

	p := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	err := b.SeedLocalCacheWithBook(p, &book)
	require.NoError(t, err)

	err = b.wsHandleData(update1)
	require.NoError(t, err)

	b.obm.state[currency.BTC][currency.USDT][asset.Spot].fetchingBook = false

	ob, err := b.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
	require.NoError(t, err)

	exp, got := seedLastUpdateID, ob.LastUpdateID
	require.Equalf(t, exp, got, "Last update id of orderbook for old update. Exp: %d, got: %d", exp, got)
	expAmnt, gotAmnt := 2.3, ob.Asks[2].Amount
	require.Equalf(t, expAmnt, gotAmnt, "Ask altered by outdated update. Exp: %f, got %f", expAmnt, gotAmnt)
	expAmnt, gotAmnt = 0.163526, ob.Bids[1].Amount
	require.Equalf(t, expAmnt, gotAmnt, "Bid altered by outdated update. Exp: %f, got %f", expAmnt, gotAmnt)

	update2 := []byte(`{"stream":"btcusdt@depth","data":{
	  "e": "depthUpdate", 
	  "E": 1234567892,     
	  "s": "BTCUSDT",      
	  "U": 161,           
	  "u": 165,           
	  "b": [           
		["6621.45", "0.163526"]
	  ],
	  "a": [             
		["6622.46", "2.3"], 
		["6622.47", "1.9"]
	  ]
	}}`)

	err = b.wsHandleData(update2)
	require.NoError(t, err)

	ob, err = b.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
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
	b.obm.state[currency.BTC][currency.USDT][asset.Spot].lastUpdateID = 0
}

func TestWsBalanceUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{
	"e": "balanceUpdate",         
	"E": 1573200697110,           
	"a": "BTC",                   
	"d": "100.00000000",          
	"T": 1573200697068            
	}}`)
	err := b.wsHandleData(pressXToJSON)
	assert.NoError(t, err)
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
	err := b.wsHandleData(pressXToJSON)
	assert.NoError(t, err)
}

func TestGetWsAuthStreamKey(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	key, err := b.GetWsAuthStreamKey(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, key)
}

func TestMaintainWsAuthStreamKey(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.MaintainWsAuthStreamKey(t.Context())
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
	start, end := getTime()
	for assetType, pair := range assetToTradablePairMap {
		result, err := b.GetHistoricCandles(t.Context(), pair, assetType, kline.OneDay, start, end)
		require.NoErrorf(t, err, "%v %v", assetType, err)
		require.NotNil(t, result)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	for assetType, pair := range assetToTradablePairMap {
		result, err := b.GetHistoricCandlesExtended(t.Context(), pair, assetType, kline.OneDay, start, end)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestBinance_FormatExchangeKlineInterval(t *testing.T) {
	testCases := []struct {
		name     string
		interval kline.Interval
		output   string
	}{
		{
			"OneMin",
			kline.OneMin,
			"1m",
		},
		{
			"OneDay",
			kline.OneDay,
			"1d",
		},
		{
			"OneWeek",
			kline.OneWeek,
			"1w",
		},
		{
			"OneMonth",
			kline.OneMonth,
			"1M",
		},
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			ret := b.FormatExchangeKlineInterval(test.interval)
			require.Equal(t, ret, test.output, "unexpected result return expected: %v received: %v", test.output, ret)
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	result, err := b.GetRecentTrades(t.Context(), pair, asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetRecentTrades(t.Context(),
		pair, asset.USDTMarginedFutures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	pair.Base = currency.NewCode("BTCUSD")
	pair.Quote = currency.PERP
	result, err = b.GetRecentTrades(t.Context(), pair, asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := b.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.False(t, sharedtestvalues.AreAPICredentialsSet(b) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests, "error cannot be nil")
	assert.False(t, mockTests && err != nil, err)
}

func TestSeedLocalCache(t *testing.T) {
	t.Parallel()
	err := b.SeedLocalCache(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	exp := subscription.List{}
	pairs, err := b.GetEnabledPairs(asset.Spot)
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
	subs, err := b.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions should not error")
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

var websocketDepthUpdate = []byte(`{"E":1608001030784,"U":7145637266,"a":[["19455.19000000","0.59490200"],["19455.37000000","0.00000000"],["19456.11000000","0.00000000"],["19456.16000000","0.00000000"],["19458.67000000","0.06400000"],["19460.73000000","0.05139800"],["19461.43000000","0.00000000"],["19464.59000000","0.00000000"],["19466.03000000","0.45000000"],["19466.36000000","0.00000000"],["19508.67000000","0.00000000"],["19572.96000000","0.00217200"],["24386.00000000","0.00256600"]],"b":[["19455.18000000","2.94649200"],["19453.15000000","0.01233600"],["19451.18000000","0.00000000"],["19446.85000000","0.11427900"],["19446.74000000","0.00000000"],["19446.73000000","0.00000000"],["19444.45000000","0.14937800"],["19426.75000000","0.00000000"],["19416.36000000","0.36052100"]],"e":"depthUpdate","s":"BTCUSDT","u":7145637297}`)

func TestProcessUpdate(t *testing.T) {
	t.Parallel()
	b := new(Binance) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	b.setupOrderbookManager()
	p := currency.NewBTCUSDT()
	var depth WebsocketDepthStream
	err := json.Unmarshal(websocketDepthUpdate, &depth)
	require.NoError(t, err)

	err = b.obm.stageWsUpdate(&depth, p, asset.Spot)
	require.NoError(t, err)

	err = b.obm.fetchBookViaREST(p)
	require.NoError(t, err)

	err = b.obm.cleanup(p)
	require.NoError(t, err)

	// reset order book sync status
	b.obm.state[currency.BTC][currency.USDT][asset.Spot].lastUpdateID = 0
}

func TestUFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := b.UFuturesHistoricalTrades(t.Context(), "", "", 5)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UFuturesHistoricalTrades(t.Context(), "BTCUSDT", "", 5)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.UFuturesHistoricalTrades(t.Context(), "BTCUSDT", "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetExchangeOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	assetTypes := b.GetAssetTypes(true)
	for a := range assetTypes {
		err := b.UpdateOrderExecutionLimits(t.Context(), assetTypes[a])
		require.NoError(t, err)
	}

	err := b.UpdateOrderExecutionLimits(t.Context(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	cmfCP, err := currency.NewPairFromStrings("BTCUSD", "PERP")
	require.NoError(t, err)

	limit, err := b.GetOrderExecutionLimits(asset.CoinMarginedFutures, cmfCP)
	require.NoError(t, err)
	require.NotEmpty(t, limit, "exchange limit should be loaded")

	err = limit.Conforms(0.000001, 0.1, order.Limit)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	err = limit.Conforms(0.01, 1, order.Limit)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
}

func TestWsOrderExecutionReport(t *testing.T) {
	t.Parallel()
	b := new(Binance) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	payload := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"executionReport","E":1616627567900,"s":"BTCUSDT","c":"c4wyKsIhoAaittTYlIVLqk","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00028400","p":"52789.10000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":5340845958,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":"BTC","T":1616627567900,"t":-1,"I":11388173160,"w":true,"m":false,"M":false,"O":1616627567900,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1616627567900}}`)
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
		Pair:                 currency.NewPair(currency.BTC, currency.USDT),
		TimeInForce:          order.GoodTillCancel,
	}
	// empty the channel. otherwise mock_test will fail
	for len(b.Websocket.DataHandler) > 0 {
		<-b.Websocket.DataHandler
	}

	err := b.wsHandleData(payload)
	require.NoError(t, err)
	res := <-b.Websocket.DataHandler
	switch r := res.(type) {
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
	err = b.wsHandleData(payload)
	assert.NoError(t, err)
}

func TestWsOutboundAccountPosition(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"outboundAccountPosition","E":1616628815745,"u":1616628815745,"B":[{"a":"BTC","f":"0.00225109","l":"0.00123000"},{"a":"BNB","f":"0.00000000","l":"0.00000000"},{"a":"USDT","f":"54.43390661","l":"0.00000000"}]}}`)
	err := b.wsHandleData(payload)
	assert.NoError(t, err)
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
			pair:              currency.NewPairWithDelimiter("btcusdt", "211231", currency.UnderscoreDelimiter),
			asset:             asset.USDTMarginedFutures,
			expectedDelimiter: currency.UnderscoreDelimiter,
		},
	}
	for i := range testerinos {
		tt := testerinos[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := b.FormatExchangeCurrency(tt.pair, tt.asset)
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
			pair:           currency.NewPairWithDelimiter("btcusdt", "211231", currency.UnderscoreDelimiter),
			asset:          asset.USDTMarginedFutures,
			expectedString: "BTCUSDT_211231",
		},
	}
	for _, tt := range testerinos {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := b.FormatSymbol(tt.pair, tt.asset)
			require.NoError(t, err)
			require.Equal(t, tt.expectedString, result)
		})
	}
}

func TestFormatUSDTMarginedFuturesPair(t *testing.T) {
	t.Parallel()
	pairFormat := currency.PairFormat{Uppercase: true}
	resp := b.formatUSDTMarginedFuturesPair(currency.NewPair(currency.DOGE, currency.USDT), pairFormat)
	require.Equal(t, "DOGEUSDT", resp.String())

	resp = b.formatUSDTMarginedFuturesPair(currency.NewPair(currency.DOGE, currency.NewCode("1234567890")), pairFormat)
	assert.Equal(t, "DOGE_1234567890", resp.String())
}

func TestFetchExchangeLimits(t *testing.T) {
	t.Parallel()
	limits, err := b.FetchExchangeLimits(t.Context(), asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, limits, "Should get some limits back")

	limits, err = b.FetchExchangeLimits(t.Context(), asset.Margin)
	require.NoError(t, err)
	require.NotEmpty(t, limits, "Should get some limits back")

	_, err = b.FetchExchangeLimits(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchExchangeLimits should error on other asset types")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	tests := map[asset.Item]currency.Pair{
		asset.Spot:   currency.NewBTCUSDT(),
		asset.Margin: currency.NewPair(currency.ETH, currency.BTC),
	}
	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Options} {
		pairs, err := b.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs should not error for %s", a)
		require.NotEmptyf(t, pairs, "Should get some pairs for %s", a)
		tests[a] = pairs[0]
	}
	for _, a := range b.GetAssetTypes(false) {
		err := b.UpdateOrderExecutionLimits(t.Context(), a)
		require.NoErrorf(t, err, "UpdateOrderExecutionLimits should not error for %v: but %v", a, err)
		p := tests[a]
		limits, err := b.GetOrderExecutionLimits(a, p)
		require.NoErrorf(t, err, "GetOrderExecutionLimits should not error for %s pair %s : %v", a, p, err)
		require.Positivef(t, limits.MinPrice, "MinPrice must be positive for %s pair %s", a, p)
		require.Positivef(t, limits.MaxPrice, "MaxPrice must be positive for %s pair %s", a, p)
		require.Positivef(t, limits.PriceStepIncrementSize, "PriceStepIncrementSize must be positive for %s pair %s", a, p)
		require.Positivef(t, limits.MinimumBaseAmount, "MinimumBaseAmount must be positive for %s pair %s", a, p)
		require.Positivef(t, limits.MaximumBaseAmount, "MaximumBaseAmount must be positive for %s pair %s", a, p)
		require.Positivef(t, limits.AmountStepIncrementSize, "AmountStepIncrementSize must be positive for %s pair %s", a, p)
		require.Positivef(t, limits.MarketMaxQty, "MarketMaxQty must be positive for %s pair %s", a, p)
		require.Positivef(t, limits.MaxTotalOrders, "MaxTotalOrders must be positive for %s pair %s", a, p)
		switch a {
		case asset.Spot, asset.Margin:
			require.Positivef(t, limits.MaxIcebergParts, "MaxIcebergParts must be positive for %s pair %s", a, p)
		case asset.USDTMarginedFutures:
			require.Positivef(t, limits.MinNotional, "MinNotional must be positive for %s pair %s", a, p)
			fallthrough
		case asset.CoinMarginedFutures:
			require.Positivef(t, limits.MultiplierUp, "MultiplierUp must be positive for %s pair %s", a, p)
			require.Positivef(t, limits.MultiplierDown, "MultiplierDown must be positive for %s pair %s", a, p)
			require.Positivef(t, limits.MarketMinQty, "MarketMinQty must be positive for %s pair %s", a, p)
			require.Positivef(t, limits.MarketStepIncrementSize, "MarketStepIncrementSize must be positive for %s pair %s", a, p)
			require.Positivef(t, limits.MaxAlgoOrders, "MaxAlgoOrders must be positive for %s pair %s", a, p)
		}
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	s, e := getTime()
	_, err := b.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		StartDate:            s,
		EndDate:              e,
		IncludePayments:      true,
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	_, err = b.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewBTCUSDT(),
		StartDate:       s,
		EndDate:         e,
		PaymentCurrency: currency.DOGE,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	r := &fundingrate.HistoricalRatesRequest{
		Asset:     asset.USDTMarginedFutures,
		Pair:      currency.NewBTCUSDT(),
		StartDate: s,
		EndDate:   e,
	}
	if sharedtestvalues.AreAPICredentialsSet(b) {
		r.IncludePayments = true
	}
	result, err := b.GetHistoricalFundingRates(t.Context(), r)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	r.Asset = asset.CoinMarginedFutures
	r.Pair, err = currency.NewPairFromString("BTCUSD_PERP")
	require.NoError(t, err)

	result, err = b.GetHistoricalFundingRates(t.Context(), r)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	cp := currency.NewBTCUSDT()
	_, err := b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 cp,
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
	err = b.CurrencyPairs.EnablePair(asset.USDTMarginedFutures, cp)
	require.True(t, err == nil || errors.Is(err, currency.ErrPairAlreadyEnabled), err)

	result, err := b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  cp,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := b.IsPerpetualFutureCurrency(asset.Binary, currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	require.False(t, is)

	is, err = b.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	require.False(t, is)
	is, err = b.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewPair(currency.BTC, currency.PERP))
	require.NoError(t, err)
	require.True(t, is)

	is, err = b.IsPerpetualFutureCurrency(asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	require.True(t, is)

	is, err = b.IsPerpetualFutureCurrency(asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.PERP))
	require.NoError(t, err)
	assert.False(t, is)
}

func TestGetUserMarginInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserMarginInterestHistory(t.Context(), currency.USDT, "BTCUSDT", time.Now().Add(-time.Hour*24), time.Now(), 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetForceLiquidiationRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetForceLiquidiationRecord(t.Context(), time.Now().Add(-time.Hour*24), time.Now(), "BTCUSDT", 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCrossMarginAccountDetail(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOrder(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginAccountsOrder(t.Context(), "", "", false, 112233424)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.GetMarginAccountsOrder(t.Context(), "BTCUSDT", "", false, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountsOrder(t.Context(), "BTCUSDT", "", false, 112233424)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountsOpenOrders(t.Context(), "BNBBTC", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountAllOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginAccountAllOrders(t.Context(), "", true, time.Time{}, time.Time{}, 0, 20)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountAllOrders(t.Context(), "BNBBTC", true, time.Time{}, time.Time{}, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetAssetsMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	is, err := b.GetAssetsMode(t.Context())
	require.NoError(t, err)

	err = b.SetAssetsMode(t.Context(), !is)
	require.NoError(t, err)

	err = b.SetAssetsMode(t.Context(), is)
	assert.NoError(t, err)
}

func TestGetAssetsMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAssetsMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	_, err := b.GetCollateralMode(t.Context(), asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = b.GetCollateralMode(t.Context(), asset.CoinMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetCollateralMode(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	err := b.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.PortfolioMode)
	require.ErrorIs(t, err, order.ErrCollateralInvalid)
	err = b.SetCollateralMode(t.Context(), asset.Spot, collateral.SingleMode)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	err = b.SetCollateralMode(t.Context(), asset.CoinMarginedFutures, collateral.SingleMode)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.MultiMode)
	require.NoError(t, err)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
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

	_, err = b.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset:          asset.Spot,
		Pair:           p,
		UnderlyingPair: currency.NewPair(currency.BTC, currency.USD),
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	bb := currency.NewBTCUSDT()
	result, err := b.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  bb,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	bb.Quote = currency.BUSD
	result, err = b.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  bb,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	bb.Quote = currency.USD
	result, err = b.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset:          asset.CoinMarginedFutures,
		Pair:           p,
		UnderlyingPair: bb,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
		Asset:                     asset.USDTMarginedFutures,
		Pairs:                     []currency.Pair{currency.NewBTCUSDT()},
		StartDate:                 time.Now().Add(-time.Hour * 24 * 70),
		RespectOrderHistoryLimits: true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
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
	err := b.SetMarginType(t.Context(), asset.Spot, currency.NewPair(currency.BTC, currency.USDT), margin.Isolated)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.SetMarginType(t.Context(), asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.USDT), margin.Isolated)
	require.NoError(t, err)

	err = b.SetMarginType(t.Context(), asset.CoinMarginedFutures, coinmTradablePair, margin.Isolated)
	assert.NoError(t, err)
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := b.GetLeverage(t.Context(), asset.Spot, currency.NewBTCUSDT(), 0, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), 0, order.UnknownSide)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetLeverage(t.Context(), asset.CoinMarginedFutures, coinmTradablePair, 0, order.UnknownSide)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	err := b.SetLeverage(t.Context(), asset.Spot, spotTradablePair, margin.Multi, 5, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 5, order.UnknownSide)
	require.NoError(t, err)
	err = b.SetLeverage(t.Context(), asset.CoinMarginedFutures, coinmTradablePair, margin.Multi, 5, order.UnknownSide)
	require.NoError(t, err)
}

func TestGetCryptoLoansIncomeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanIncomeHistory(t.Context(), currency.USDT, "", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanBorrow(t *testing.T) {
	t.Parallel()
	_, err := b.CryptoLoanBorrow(t.Context(), currency.EMPTYCODE, 1000, currency.BTC, 1, 7)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.CryptoLoanBorrow(t.Context(), currency.USDT, 1000, currency.EMPTYCODE, 1, 7)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = b.CryptoLoanBorrow(t.Context(), currency.USDT, 0, currency.BTC, 1, 0)
	require.ErrorIs(t, err, errLoanTermMustBeSet)
	_, err = b.CryptoLoanBorrow(t.Context(), currency.USDT, 0, currency.BTC, 0, 7)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CryptoLoanBorrow(t.Context(), currency.USDT, 1000, currency.BTC, 1, 7)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanBorrowHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanBorrowHistory(t.Context(), 0, currency.USDT, currency.BTC, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanOngoingOrders(t.Context(), 0, currency.USDT, currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanRepay(t *testing.T) {
	t.Parallel()
	_, err := b.CryptoLoanRepay(t.Context(), 0, 1000, 1, false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = b.CryptoLoanRepay(t.Context(), 42069, 0, 1, false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CryptoLoanRepay(t.Context(), 42069, 1000, 1, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanRepaymentHistory(t.Context(), 0, currency.USDT, currency.BTC, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanAdjustLTV(t *testing.T) {
	t.Parallel()
	_, err := b.CryptoLoanAdjustLTV(t.Context(), 0, true, 1)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = b.CryptoLoanAdjustLTV(t.Context(), 42069, true, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CryptoLoanAdjustLTV(t.Context(), 42069, true, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanLTVAdjustmentHistory(t.Context(), 0, currency.USDT, currency.BTC, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanAssetsData(t.Context(), currency.EMPTYCODE, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanCollateralAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanCollateralAssetsData(t.Context(), currency.EMPTYCODE, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanCheckCollateralRepayRate(t *testing.T) {
	t.Parallel()
	_, err := b.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.EMPTYCODE, currency.BNB, 69)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.BUSD, currency.EMPTYCODE, 69)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = b.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.BUSD, currency.BNB, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.BUSD, currency.BNB, 69)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanCustomiseMarginCall(t *testing.T) {
	t.Parallel()
	_, err := b.CryptoLoanCustomiseMarginCall(t.Context(), 0, currency.BTC, 0)
	require.ErrorIs(t, err, errMarginCallValueRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CryptoLoanCustomiseMarginCall(t.Context(), 1337, currency.BTC, .70)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanBorrow(t *testing.T) {
	t.Parallel()
	_, err := b.FlexibleLoanBorrow(t.Context(), currency.EMPTYCODE, currency.USDC, 1, 0)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.FlexibleLoanBorrow(t.Context(), currency.ATOM, currency.EMPTYCODE, 1, 0)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = b.FlexibleLoanBorrow(t.Context(), currency.ATOM, currency.USDC, 0, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FlexibleLoanBorrow(t.Context(), currency.ATOM, currency.USDC, 1, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleLoanOngoingOrders(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanBorrowHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleLoanBorrowHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanRepay(t *testing.T) {
	t.Parallel()
	_, err := b.FlexibleLoanRepay(t.Context(), currency.EMPTYCODE, currency.BTC, 1, false, false)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.FlexibleLoanRepay(t.Context(), currency.USDT, currency.EMPTYCODE, 1, false, false)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = b.FlexibleLoanRepay(t.Context(), currency.USDT, currency.BTC, 0, false, false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FlexibleLoanRepay(t.Context(), currency.ATOM, currency.USDC, 1, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanRepayHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleLoanRepayHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanCollateralRepayment(t *testing.T) {
	t.Parallel()
	_, err := b.FlexibleLoanCollateralRepayment(t.Context(), currency.EMPTYCODE, currency.USDT, 1000, true)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.FlexibleLoanCollateralRepayment(t.Context(), currency.BTC, currency.EMPTYCODE, 1000, true)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = b.FlexibleLoanCollateralRepayment(t.Context(), currency.BTC, currency.USDT, 0, true)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FlexibleLoanCollateralRepayment(t.Context(), currency.BTC, currency.USDT, 1000, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckCollateralRepayRate(t *testing.T) {
	t.Parallel()
	_, err := b.CheckCollateralRepayRate(t.Context(), currency.EMPTYCODE, currency.USDT)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.CheckCollateralRepayRate(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CheckCollateralRepayRate(t.Context(), currency.BTC, currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleLoanLiquidiationHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleLoanLiquidiationHistory(t.Context(), currency.BTC, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanAdjustLTV(t *testing.T) {
	t.Parallel()
	_, err := b.FlexibleLoanAdjustLTV(t.Context(), currency.EMPTYCODE, currency.BTC, 1, true)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.FlexibleLoanAdjustLTV(t.Context(), currency.USDT, currency.EMPTYCODE, 1, true)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = b.FlexibleLoanAdjustLTV(t.Context(), currency.USDT, currency.BTC, 0, true)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FlexibleLoanAdjustLTV(t.Context(), currency.USDT, currency.BTC, 1, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleLoanLTVAdjustmentHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleLoanAssetsData(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleCollateralAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleCollateralAssetsData(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = b.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := b.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	result, err := b.GetFundingRateInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	result, err := b.UGetFundingRateInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsUFuturesConnect(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	err := b.WsUFuturesConnect()
	require.NoError(t, err)
}

var messageMap = map[string]string{
	"Asset Index":                   `{"stream": "!assetIndex@arr", "data": [{ "e":"assetIndexUpdate", "E":1686749230000, "s":"ADAUSD", "i":"0.27462452", "b":"0.10000000", "a":"0.10000000", "B":"0.24716207", "A":"0.30208698", "q":"0.05000000", "g":"0.05000000", "Q":"0.26089330", "G":"0.28835575" }, { "e":"assetIndexUpdate", "E":1686749230000, "s":"USDTUSD", "i":"0.99987691", "b":"0.00010000", "a":"0.00010000", "B":"0.99977692", "A":"0.99997689", "q":"0.00010000", "g":"0.00010000", "Q":"0.99977692", "G":"0.99997689" } ]}`,
	"Contract Info":                 `{"stream": "!contractInfo", "data": {"e":"contractInfo", "E":1669356423908, "s":"IOTAUSDT", "ps":"IOTAUSDT", "ct":"PERPETUAL", "dt":4133404800000, "ot":1569398400000, "cs":"TRADING", "bks":[ { "bs":1, "bnf":0, "bnc":5000, "mmr":0.01, "cf":0, "mi":21, "ma":50 }, { "bs":2, "bnf":5000, "bnc":25000, "mmr":0.025, "cf":75, "mi":11, "ma":20 } ] }}`,
	"Force Order":                   `{"stream": "!forceOrder@arr", "data": {"e":"forceOrder", "E":1568014460893, "o":{ "s":"BTCUSDT", "S":"SELL", "o":"LIMIT", "f":"IOC", "q":"0.014", "p":"9910", "ap":"9910", "X":"FILLED", "l":"0.014", "z":"0.014", "T":1568014460893 }}}`,
	"All BookTicker":                `{"stream": "!bookTicker","data":{"e":"bookTicker","u":3682854202063,"s":"NEARUSDT","b":"2.4380","B":"20391","a":"2.4390","A":"271","T":1703015198639,"E":1703015198640}}`,
	"Multiple Market Ticker":        `{"stream": "!ticker@arr", "data": [{"e":"24hrTicker","E":1703018247910,"s":"ICPUSDT","p":"-0.540000","P":"-5.395","w":"9.906194","c":"9.470000","Q":"1","o":"10.010000","h":"10.956000","l":"9.236000","v":"34347035","q":"340248403.001000","O":1702931820000,"C":1703018247909,"F":78723309,"L":80207941,"n":1484628},{"e":"24hrTicker","E":1703018247476,"s":"MEMEUSDT","p":"0.0020900","P":"7.331","w":"0.0300554","c":"0.0305980","Q":"7568","o":"0.0285080","h":"0.0312730","l":"0.0284120","v":"5643663185","q":"169622568.3721920","O":1702931820000,"C":1703018247475,"F":88665791,"L":89517438,"n":851643},{"e":"24hrTicker","E":1703018247822,"s":"SOLUSDT","p":"0.8680","P":"1.192","w":"74.4933","c":"73.6900","Q":"21","o":"72.8220","h":"76.3840","l":"71.8000","v":"26283647","q":"1957955612.4830","O":1702931820000,"C":1703018247820,"F":1126774871,"L":1129007642,"n":2232761},{"e":"24hrTicker","E":1703018247254,"s":"IMXUSDT","p":"0.0801","P":"3.932","w":"2.1518","c":"2.1171","Q":"225","o":"2.0370","h":"2.2360","l":"2.0319","v":"59587050","q":"128216496.4538","O":1702931820000,"C":1703018247252,"F":169814879,"L":170587124,"n":772246},{"e":"24hrTicker","E":1703018247309,"s":"DYDXUSDT","p":"-0.036","P":"-1.255","w":"2.896","c":"2.832","Q":"169.6","o":"2.868","h":"2.987","l":"2.782","v":"81690098.5","q":"236599791.383","O":1702931820000,"C":1703018247308,"F":385238821,"L":385888621,"n":649799},{"e":"24hrTicker","E":1703018247240,"s":"ONTUSDT","p":"0.0022","P":"1.011","w":"0.2213","c":"0.2197","Q":"45.7","o":"0.2175","h":"0.2251","l":"0.2157","v":"60880132.6","q":"13471239.8637","O":1702931820000,"C":1703018247238,"F":186008331,"L":186088275,"n":79945},{"e":"24hrTicker","E":1703018247658,"s":"AAVEUSDT","p":"4.660","P":"4.778","w":"102.969","c":"102.190","Q":"0.4","o":"97.530","h":"108.000","l":"97.370","v":"1205430.6","q":"124121750.870","O":1702931820000,"C":1703018247657,"F":343017862,"L":343487276,"n":469414},{"e":"24hrTicker","E":1703018247545,"s":"USTCUSDT","p":"0.0018500","P":"5.628","w":"0.0348991","c":"0.0347200","Q":"2316","o":"0.0328700","h":"0.0371100","l":"0.0328000","v":"2486985654","q":"86793545.3903700","O":1702931820000,"C":1703018247544,"F":32136013,"L":32601947,"n":465935},{"e":"24hrTicker","E":1703018247997,"s":"FTMUSDT","p":"-0.005000","P":"-1.221","w":"0.409721","c":"0.404400","Q":"1421","o":"0.409400","h":"0.421200","l":"0.392100","v":"471077518","q":"193010517.884400","O":1702931820000,"C":1703018247996,"F":716077491,"L":716712548,"n":635055},{"e":"24hrTicker","E":1703018247338,"s":"LRCUSDT","p":"-0.00290","P":"-1.104","w":"0.26531","c":"0.25980","Q":"113","o":"0.26270","h":"0.27190","l":"0.25590","v":"142488749","q":"37803477.10260","O":1702931820000,"C":1703018247336,"F":318115460,"L":318317340,"n":201880},{"e":"24hrTicker","E":1703018247776,"s":"TRBUSDT","p":"25.037","P":"21.840","w":"131.860","c":"139.677","Q":"0.3","o":"114.640","h":"143.900","l":"113.600","v":"3955845.0","q":"521616257.947","O":1702931820000,"C":1703018247775,"F":417041483,"L":419226886,"n":2185249},{"e":"24hrTicker","E":1703018247513,"s":"ACEUSDT","p":"0.108200","P":"0.826","w":"13.544944","c":"13.211400","Q":"14.37","o":"13.103200","h":"15.131200","l":"12.402900","v":"41359842.25","q":"560216757.038015","O":1702931820000,"C":1703018247512,"F":2261106,"L":4779982,"n":2518828},{"e":"24hrTicker","E":1703018247995,"s":"KEYUSDT","p":"0.0000270","P":"0.506","w":"0.0054583","c":"0.0053660","Q":"3540","o":"0.0053390","h":"0.0056230","l":"0.0053220","v":"1658962254","q":"9055176.9144700","O":1702931820000,"C":1703018247993,"F":32127330,"L":32236546,"n":109217},{"e":"24hrTicker","E":1703018247825,"s":"SUIUSDT","p":"0.094400","P":"15.783","w":"0.658766","c":"0.692500","Q":"157.6","o":"0.598100","h":"0.719600","l":"0.596400","v":"538807943.2","q":"354948524.988570","O":1702931820000,"C":1703018247824,"F":129572611,"L":130637476,"n":1064863},{"e":"24hrTicker","E":1703018247328,"s":"AGLDUSDT","p":"0.0738000","P":"7.016","w":"1.1222224","c":"1.1257000","Q":"49","o":"1.0519000","h":"1.1936000","l":"1.0471000","v":"63230369","q":"70958539.3508000","O":1702931820000,"C":1703018247327,"F":40498492,"L":41170995,"n":672503},{"e":"24hrTicker","E":1703018247882,"s":"BTCUSDT","p":"412.30","P":"0.986","w":"42651.76","c":"42247.00","Q":"0.003","o":"41834.70","h":"43550.00","l":"41792.00","v":"366582.423","q":"15635385730.76","O":1702931820000,"C":1703018247880,"F":4392041494,"L":4395950440,"n":3908934},{"e":"24hrTicker","E":1703018247531,"s":"WLDUSDT","p":"-0.0475000","P":"-1.232","w":"3.9879959","c":"3.8089000","Q":"50","o":"3.8564000","h":"4.3320000","l":"3.7237000","v":"119350666","q":"475969966.2747000","O":1702931820000,"C":1703018247530,"F":183723717,"L":186154953,"n":2431230},{"e":"24hrTicker","E":1703018247595,"s":"WAVESUSDT","p":"0.1108","P":"4.876","w":"2.4490","c":"2.3833","Q":"8.1","o":"2.2725","h":"2.5775","l":"2.2658","v":"54051344.0","q":"132369622.6356","O":1702931820000,"C":1703018247593,"F":503343992,"L":504167968,"n":823975},{"e":"24hrTicker","E":1703018247943,"s":"BLZUSDT","p":"0.00441","P":"1.274","w":"0.34477","c":"0.35043","Q":"35","o":"0.34602","h":"0.35844","l":"0.33146","v":"224686045","q":"77465133.09517","O":1702931820000,"C":1703018247942,"F":301286442,"L":301919432,"n":632991},{"e":"24hrTicker","E":1703018248027,"s":"ALGOUSDT","p":"0.0044","P":"2.329","w":"0.1982","c":"0.1933","Q":"1724.4","o":"0.1889","h":"0.2053","l":"0.1883","v":"418107041.7","q":"82860752.3534","O":1702931820000,"C":1703018248025,"F":317274252,"L":317530189,"n":255937},{"e":"24hrTicker","E":1703018247795,"s":"LUNA2USDT","p":"0.0849000","P":"9.610","w":"0.9622720","c":"0.9684000","Q":"91","o":"0.8835000","h":"1.0234000","l":"0.8800000","v":"132211955","q":"127223857.1990000","O":1702931820000,"C":1703018247793,"F":143814989,"L":144504341,"n":689350},{"e":"24hrTicker","E":1703018247557,"s":"DOGEUSDT","p":"-0.000290","P":"-0.320","w":"0.091710","c":"0.090210","Q":"1211","o":"0.090500","h":"0.093550","l":"0.089300","v":"4695249277","q":"430603554.425970","O":1702931820000,"C":1703018247556,"F":1408300026,"L":1409042131,"n":742103},{"e":"24hrTicker","E":1703018247578,"s":"SUSHIUSDT","p":"0.0024","P":"0.217","w":"1.1263","c":"1.1097","Q":"34","o":"1.1073","h":"1.1479","l":"1.0921","v":"34830643","q":"39229338.9293","O":1702931820000,"C":1703018247576,"F":389676753,"L":389892337,"n":215584},{"e":"24hrTicker","E":1703018247636,"s":"ROSEUSDT","p":"0.00859","P":"9.826","w":"0.09344","c":"0.09601","Q":"300","o":"0.08742","h":"0.09842","l":"0.08724","v":"768803655","q":"71837497.60153","O":1702931820000,"C":1703018247635,"F":145874088,"L":146347778,"n":473689},{"e":"24hrTicker","E":1703018247446,"s":"CTKUSDT","p":"0.05240","P":"6.933","w":"0.76993","c":"0.80820","Q":"16","o":"0.75580","h":"0.81760","l":"0.73560","v":"39275735","q":"30239750.04250","O":1702931820000,"C":1703018247445,"F":129601557,"L":129911270,"n":309714},{"e":"24hrTicker","E":1703018247083,"s":"MATICUSDT","p":"-0.02260","P":"-2.883","w":"0.78657","c":"0.76130","Q":"11","o":"0.78390","h":"0.82380","l":"0.74930","v":"510723474","q":"401719478.20480","O":1702931820000,"C":1703018247081,"F":899425701,"L":900164133,"n":738432},{"e":"24hrTicker","E":1703018247954,"s":"INJUSDT","p":"3.554000","P":"10.740","w":"37.577625","c":"36.646000","Q":"9.3","o":"33.092000","h":"39.988000","l":"32.803000","v":"30119373.7","q":"1131814520.584100","O":1702931820000,"C":1703018247953,"F":210846748,"L":214612851,"n":3766053},{"e":"24hrTicker","E":1703018247559,"s":"OCEANUSDT","p":"0.00890","P":"1.805","w":"0.50791","c":"0.50200","Q":"147","o":"0.49310","h":"0.52090","l":"0.49170","v":"42754656","q":"21715597.51239","O":1702931820000,"C":1703018247557,"F":243729859,"L":243879437,"n":149578},{"e":"24hrTicker","E":1703018247779,"s":"UNIUSDT","p":"0.0220","P":"0.378","w":"5.9288","c":"5.8470","Q":"10","o":"5.8250","h":"6.0440","l":"5.7520","v":"11324960","q":"67143423.4300","O":1702931820000,"C":1703018247778,"F":356204442,"L":356430119,"n":225678},{"e":"24hrTicker","E":1703018247999,"s":"1000BONKUSDT","p":"-0.0004410","P":"-2.245","w":"0.0205588","c":"0.0192000","Q":"1562","o":"0.0196410","h":"0.0231060","l":"0.0188770","v":"30632634003","q":"629769968.4590968","O":1702931820000,"C":1703018247998,"F":75958362,"L":80131721,"n":4173351},{"e":"24hrTicker","E":1703018247559,"s":"ARUSDT","p":"-0.382","P":"-4.176","w":"9.030","c":"8.765","Q":"4.5","o":"9.147","h":"9.467","l":"8.571","v":"3178087.5","q":"28698158.147","O":1702931820000,"C":1703018247557,"F":143756455,"L":143985699,"n":229244},{"e":"24hrTicker","E":1703018247344,"s":"AUCTIONUSDT","p":"9.690000","P":"31.369","w":"38.302392","c":"40.580000","Q":"0.65","o":"30.890000","h":"43.400000","l":"30.650000","v":"15656989.13","q":"599700134.856300","O":1702931820000,"C":1703018247343,"F":2451094,"L":5013398,"n":2561767},{"e":"24hrTicker","E":1703018247959,"s":"XRPUSDT","p":"-0.0021","P":"-0.346","w":"0.6083","c":"0.6045","Q":"396.3","o":"0.6066","h":"0.6170","l":"0.5973","v":"744301855.7","q":"452752948.7478","O":1702931820000,"C":1703018247957,"F":1344388341,"L":1344913573,"n":525224},{"e":"24hrTicker","E":1703018247813,"s":"EGLDUSDT","p":"-0.130","P":"-0.223","w":"58.569","c":"58.070","Q":"1.1","o":"58.200","h":"60.240","l":"56.670","v":"802381.7","q":"46994956.463","O":1702931820000,"C":1703018247811,"F":235206699,"L":235456030,"n":249331},{"e":"24hrTicker","E":1703018247990,"s":"ETHUSDT","p":"-10.21","P":"-0.468","w":"2206.89","c":"2170.39","Q":"0.060","o":"2180.60","h":"2256.64","l":"2135.03","v":"3187161.031","q":"7033700225.77","O":1702931820000,"C":1703018247988,"F":3443398114,"L":3446512406,"n":3114283},{"e":"24hrTicker","E":1703018247096,"s":"PENDLEUSDT","p":"-0.0059000","P":"-0.569","w":"1.0590403","c":"1.0319000","Q":"12","o":"1.0378000","h":"1.0960000","l":"1.0120000","v":"7593669","q":"8042001.5937000","O":1702931820000,"C":1703018247095,"F":16663914,"L":16782530,"n":118617}]}`,
	"Single Market Ticker":          `{"stream": "BTCUSDT@ticker", "data": { "e": "24hrTicker", "E": 1571889248277, "s": "BTCUSDT", "p": "0.0015", "P": "250.00", "w": "0.0018", "c": "0.0025", "Q": "10", "o": "0.0010", "h": "0.0025", "l": "0.0010", "v": "10000", "q": "18", "O": 0, "C": 1703019429985, "F": 0, "L": 18150, "n": 18151 } }`,
	"Multiple Mini Tickers":         `{"stream": "!miniTicker@arr","data":[{"e":"24hrMiniTicker","E":1703019429455,"s":"BICOUSDT","c":"0.3667000","o":"0.3792000","h":"0.3892000","l":"0.3639000","v":"28768370","q":"10779000.9922000"},{"e":"24hrMiniTicker","E":1703019429985,"s":"API3USDT","c":"1.6834","o":"1.7326","h":"1.8406","l":"1.6699","v":"12371516.4","q":"21642153.0574"},{"e":"24hrMiniTicker","E":1703019429111,"s":"ICPUSDT","c":"9.414000","o":"10.126000","h":"10.956000","l":"9.236000","v":"34262192","q":"339148145.539000"},{"e":"24hrMiniTicker","E":1703019429945,"s":"SOLUSDT","c":"73.0930","o":"73.2180","h":"76.3840","l":"71.8000","v":"26319095","q":"1960871540.2620"}]}`,
	"Multi Asset Mode Asset":        `{"stream": "!assetIndex@arr", "data":[{ "e":"assetIndexUpdate", "E":1686749230000, "s":"ADAUSD","i":"0.27462452","b":"0.10000000","a":"0.10000000","B":"0.24716207","A":"0.30208698","q":"0.05000000","g":"0.05000000","Q":"0.26089330","G":"0.28835575"}, { "e":"assetIndexUpdate", "E":1686749230000, "s":"USDTUSD", "i":"0.99987691", "b":"0.00010000", "a":"0.00010000", "B":"0.99977692", "A":"0.99997689", "q":"0.00010000", "g":"0.00010000", "Q":"0.99977692", "G":"0.99997689" }]}`,
	"Composite Index Symbol":        `{"stream": "BTCUSDT@compositeIndex", "data":{ "e":"compositeIndex", "E":1602310596000, "s":"DEFIUSDT", "p":"554.41604065", "C":"baseAsset", "c":[ { "b":"BAL", "q":"USDT", "w":"1.04884844", "W":"0.01457800", "i":"24.33521021" }, { "b":"BAND", "q":"USDT" , "w":"3.53782729", "W":"0.03935200", "i":"7.26420084" } ] } }`,
	"Diff Book Depth Stream":        `{"stream": "BTCUSDT@depth@500ms", "data": { "e": "depthUpdate", "E": 1571889248277, "T": 1571889248276, "s": "BTCUSDT", "U": 157, "u": 160, "pu": 149, "b": [ [ "0.0024", "10" ] ], "a": [ [ "0.0026", "100" ] ] } }`,
	"Partial Book Depth Stream":     `{"stream": "BTCUSDT@depth5", "data":{ "e": "depthUpdate", "E": 1571889248277, "T": 1571889248276, "s": "BTCUSDT", "U": 390497796, "u": 390497878, "pu": 390497794, "b": [ [ "7403.89", "0.002" ], [ "7403.90", "3.906" ], [ "7404.00", "1.428" ], [ "7404.85", "5.239" ], [ "7405.43", "2.562" ] ], "a": [ [ "7405.96", "3.340" ], [ "7406.63", "4.525" ], [ "7407.08", "2.475" ], [ "7407.15", "4.800" ], [ "7407.20","0.175"]]}}`,
	"Individual Symbol Mini Ticker": `{"stream": "BTCUSDT@miniTicker", "data": { "e": "24hrMiniTicker", "E": 1571889248277, "s": "BTCUSDT", "c": "0.0025", "o": "0.0010", "h": "0.0025", "l": "0.0010", "v": "10000", "q": "18"}}`,
}

func TestHandleData(t *testing.T) {
	t.Parallel()
	for x := range messageMap {
		err := b.wsHandleFuturesData([]byte(messageMap[x]), asset.USDTMarginedFutures)
		assert.NoError(t, err)
	}
}

func TestListSubscriptions(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	if !b.Websocket.IsConnected() {
		err := b.WsUFuturesConnect()
		require.NoError(t, err)
	}
	result, err := b.ListSubscriptions()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetProperty(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	if !b.Websocket.IsConnected() {
		err := b.WsUFuturesConnect()
		require.NoError(t, err)
	}
	err := b.SetProperty("combined", true)
	require.NoError(t, err)
}

func TestGetWsOrderbook(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWsOrderbook(&OrderBookDataRequestParams{Symbol: currency.NewPair(currency.BTC, currency.USDT), Limit: 1000})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsMostRecentTrades(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWsMostRecentTrades(&RecentTradeRequestParams{
		Symbol: currency.NewPair(currency.BTC, currency.USDT),
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
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWsAggregatedTrades(&WsAggregateTradeRequestParams{
		Symbol: "BTCUSDT",
		Limit:  5,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsKlines(t *testing.T) {
	t.Parallel()
	_, err := b.GetWsCandlestick(&KlinesRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &KlinesRequestParams{Timezone: "GMT+2"}
	_, err = b.GetWsCandlestick(arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = spotTradablePair
	_, err = b.GetWsCandlestick(arg)
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	if mockTests {
		t.SkipNow()
	}
	start, end := getTime()
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWsCandlestick(&KlinesRequestParams{
		Symbol:    currency.NewPair(currency.BTC, currency.USDT),
		Interval:  kline.FiveMin.Short(),
		Limit:     24,
		StartTime: start,
		EndTime:   end,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsOptimizedCandlestick(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	start, end := getTime()
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWsOptimizedCandlestick(&KlinesRequestParams{
		Symbol:    currency.NewPair(currency.BTC, currency.USDT),
		Interval:  kline.FiveMin.Short(),
		Limit:     24,
		StartTime: start,
		EndTime:   end,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func setupWs() {
	err := b.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestGetCurrenctAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetWsCurrenctAveragePrice(currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	if mockTests {
		t.SkipNow()
	}
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWsCurrenctAveragePrice(currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWs24HourPriceChanges(t *testing.T) {
	t.Parallel()
	_, err := b.GetWs24HourPriceChanges(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = b.GetWs24HourPriceChanges(&PriceChangeRequestParam{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	if mockTests {
		t.SkipNow()
	}
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWs24HourPriceChanges(&PriceChangeRequestParam{Symbols: []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsTradingDayTickers(t *testing.T) {
	t.Parallel()
	_, err := b.GetWsTradingDayTickers(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = b.GetWsTradingDayTickers(&PriceChangeRequestParam{Timezone: "GMT+3"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	if mockTests {
		t.SkipNow()
	}
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWsTradingDayTickers(&PriceChangeRequestParam{
		Symbols: []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetSymbolPriceTicker(currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	if mockTests {
		t.SkipNow()
	}
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetSymbolPriceTicker(currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetWsSymbolOrderbookTicker([]currency.Pair{currency.EMPTYPAIR})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	if mockTests {
		t.SkipNow()
	}
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWsSymbolOrderbookTicker([]currency.Pair{currency.NewPair(currency.BTC, currency.USDT)})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetWsSymbolOrderbookTicker([]currency.Pair{
		currency.NewPair(currency.BTC, currency.USDT),
		currency.NewPair(currency.ETH, currency.USDT),
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuerySessionStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetQuerySessionStatus()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLogOutOfSession(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetLogOutOfSession()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsPlaceNewOrder(&TradeOrderRequestParam{
		Symbol:      "BTCUSDT",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	err := b.ValidatePlaceNewOrderRequest(&TradeOrderRequestParam{
		Symbol:      "BTCUSDT",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsQueryOrder(&QueryOrderParam{
		Symbol:  "BTCUSDT",
		OrderID: 12345,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSignRequest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	_, signature, err := b.SignRequest(map[string]interface{}{
		"name": "nameValue",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, signature, "unexpected signature")
}

func TestWsCancelAndReplaceTradeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsCancelAndReplaceTradeOrder(&WsCancelAndReplaceParam{
		Symbol:                    "BTCUSDT",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsCurrentOpenOrders(currency.NewPair(currency.BTC, currency.USDT), 6000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsCancelOpenOrders(currency.NewPair(currency.BTC, currency.USDT), 6000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPlaceOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := b.WsPlaceOCOOrder(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &PlaceOCOOrderParam{StopLimitTimeInForce: "GTC"}
	_, err = b.WsPlaceOCOOrder(arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = "BTCUSDT"
	_, err = b.WsPlaceOCOOrder(arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = b.WsPlaceOCOOrder(arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsPlaceOCOOrder(&PlaceOCOOrderParam{
		Symbol:               "BTCUSDT",
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
	_, err := b.WsQueryOCOOrder("", 0, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsQueryOCOOrder("123456788", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := b.WsCancelOCOOrder(currency.EMPTYPAIR, "someID", "12354", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.WsCancelOCOOrder(spotTradablePair, "", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsCancelOCOOrder(
		currency.NewPair(currency.BTC, currency.USDT), "someID", "12354", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCurrentOpenOCOOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsCurrentOpenOCOOrders(0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPlaceNewSOROrder(t *testing.T) {
	t.Parallel()
	_, err := b.WsPlaceNewSOROrder(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &WsOSRPlaceOrderParams{TimeInForce: "GTC"}
	_, err = b.WsPlaceNewSOROrder(arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = spotTradablePair.String()
	_, err = b.WsPlaceNewSOROrder(arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = b.WsPlaceNewSOROrder(arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = b.WsPlaceNewSOROrder(arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsPlaceNewSOROrder(&WsOSRPlaceOrderParams{
		Symbol:      "BTCUSDT",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	err := b.WsTestNewOrderUsingSOR(&WsOSRPlaceOrderParams{
		Symbol:      "BTCUSDT",
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
	result, err := b.ToMap(input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSortingTest(t *testing.T) {
	params := map[string]any{"apiKey": "wwhj3r3amR", "signature": "f89c6e5c0b", "timestamp": 1704873175325, "symbol": "BTCUSDT", "startTime": 1704009175325, "endTime": 1704873175325, "limit": 5}
	sortedKeys := []string{"apiKey", "endTime", "limit", "signature", "startTime", "symbol", "timestamp"}
	keys := SortMap(params)
	require.Len(t, keys, len(sortedKeys), "unexptected keys length")
	for a := range keys {
		require.Equal(t, keys[a], sortedKeys[a])
	}
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWsAccountInfo(0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsQueryAccountOrderRateLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsQueryAccountOrderRateLimits(0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsQueryAccountOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := b.WsQueryAccountOrderHistory(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = b.WsQueryAccountOrderHistory(&AccountOrderRequestParam{Limit: 5})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsQueryAccountOrderHistory(&AccountOrderRequestParam{
		Symbol:    "BTCUSDT",
		StartTime: time.Now().Add(-time.Hour * 24 * 10).UnixMilli(),
		EndTime:   time.Now().Add(-time.Hour * 6).UnixMilli(),
		Limit:     5,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsQueryAccountOCOOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsQueryAccountOCOOrderHistory(0, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAccountTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := b.WsAccountTradeHistory(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = b.WsAccountTradeHistory(&AccountOrderRequestParam{OrderID: 1234})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsAccountTradeHistory(&AccountOrderRequestParam{Symbol: "BTCUSDT", OrderID: 1234})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAccountPreventedMatches(t *testing.T) {
	t.Parallel()
	_, err := b.WsAccountPreventedMatches(currency.EMPTYPAIR, 1223456, 0, 0, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.WsAccountPreventedMatches(spotTradablePair, 0, 0, 0, 0, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsAccountPreventedMatches(currency.NewPair(currency.BTC, currency.USDT), 1223456, 0, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAccountAllocation(t *testing.T) {
	t.Parallel()
	_, err := b.WsAccountAllocation(currency.EMPTYPAIR, time.Time{}, time.Now(), 0, 0, 0, 19)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsAccountAllocation(spotTradablePair, time.Time{}, time.Now(), 0, 0, 0, 19)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAccountCommissionRates(t *testing.T) {
	t.Parallel()
	_, err := b.WsAccountCommissionRates(currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsAccountCommissionRates(spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsStartUserDataStream(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsStartUserDataStream()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPingUserDataStream(t *testing.T) {
	t.Parallel()
	err := b.WsPingUserDataStream("")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	err = b.WsPingUserDataStream("xs0mRXdAKlIPDRFrlPcw0qI41Eh3ixNntmymGyhrhgqo7L6FuLaWArTD7RLP")
	require.NoError(t, err)
}

func TestWsStopUserDataStream(t *testing.T) {
	t.Parallel()
	err := b.WsStopUserDataStream("")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	err = b.WsStopUserDataStream("xs0mRXdAKlIPDRFrlPcw0qI41Eh3ixNntmymGyhrhgqo7L6FuLaWArTD7RLP")
	require.NoError(t, err)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.Spot,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result)

	result, err = b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.NewCode("BTCUSD").Item,
		Quote: currency.PERP.Item,
		Asset: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestSystemStatus(t *testing.T) {
	t.Parallel()
	result, err := b.GetSystemStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDailyAccountSnapshot(t *testing.T) {
	t.Parallel()
	_, err := b.GetDailyAccountSnapshot(t.Context(), "", time.Time{}, time.Now(), 0)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDailyAccountSnapshot(t.Context(), "SPOT", time.Time{}, time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDisableFastWithdrawalSwitch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.DisableFastWithdrawalSwitch(t.Context())
	assert.NoError(t, err)
}

func TestEnableFastWithdrawalSwitch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.EnableFastWithdrawalSwitch(t.Context())
	assert.NoError(t, err)
}

func TestGetAccountStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTradingAPIStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountTradingAPIStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDustLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDustLog(t.Context(), "MARGIN", time.Time{}, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckServerTime(t *testing.T) {
	t.Parallel()
	result, err := b.GetExchangeServerTime(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccount(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := b.GetAccountTradeList(t.Context(), "", "", time.Now().Add(-time.Hour*5), time.Now(), 0, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountTradeList(t.Context(), "BNBBTC", "", time.Now().Add(-time.Hour*5), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentOrderCountUsage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrentOrderCountUsage(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPreventedMatches(t *testing.T) {
	t.Parallel()
	_, err := b.GetPreventedMatches(t.Context(), "", 0, 12, 0, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.GetPreventedMatches(t.Context(), "BTCUSDT", 0, 0, 0, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPreventedMatches(t.Context(), "BTCUSDT", 0, 12, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllocations(t *testing.T) {
	t.Parallel()
	_, err := b.GetAllocations(t.Context(), "", time.Time{}, time.Time{}, 10, 10, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllocations(t.Context(), "BTCUSDT", time.Time{}, time.Time{}, 10, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := b.GetCommissionRates(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCommissionRates(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountBorrowRepay(t *testing.T) {
	t.Parallel()
	_, err := b.MarginAccountBorrowRepay(t.Context(), currency.ETH, "", "BORROW", false, 0.1234)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.MarginAccountBorrowRepay(t.Context(), currency.EMPTYCODE, "BTCUSDT", "BORROW", false, 0.1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.MarginAccountBorrowRepay(t.Context(), currency.ETH, "BTCUSDT", "", false, 0.1234)
	require.ErrorIs(t, err, errLendingTypeRequired)
	_, err = b.MarginAccountBorrowRepay(t.Context(), currency.ETH, "BTCUSDT", "BORROW", false, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.MarginAccountBorrowRepay(t.Context(), currency.ETH, "BTCUSDT", "BORROW", false, 0.1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowOrRepayRecordsInMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBorrowOrRepayRecordsInMarginAccount(t.Context(), currency.LTC, "", "REPAY", 0, 10, 0, time.Now().Add(-time.Hour*12), time.Now().Add(-time.Hour*6))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllMarginAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllMarginAssets(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCrossMarginPairs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCrossMarginPairs(t.Context(), "BNBBTC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginPriceIndex(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginPriceIndex(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginPriceIndex(t.Context(), "BNBBTC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPostMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := b.PostMarginAccountOrder(t.Context(), &MarginAccountOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &MarginAccountOrderParam{AutoRepayAtCancel: true}
	_, err = b.PostMarginAccountOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.NewPair(currency.BTC, currency.USDT)
	_, err = b.PostMarginAccountOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy.String()
	_, err = b.PostMarginAccountOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.PostMarginAccountOrder(t.Context(), &MarginAccountOrderParam{
		Symbol:    currency.NewPair(currency.BTC, currency.USDT),
		Side:      order.Buy.String(),
		OrderType: order.Limit.String(),
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := b.CancelMarginAccountOrder(t.Context(), "", "", "", true, 12314234)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.CancelMarginAccountOrder(t.Context(), "BTCUSDT", "", "", true, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelMarginAccountOrder(t.Context(), "BTCUSDT", "", "", true, 12314234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountCancelAllOpenOrdersOnSymbol(t *testing.T) {
	t.Parallel()
	_, err := b.MarginAccountCancelAllOpenOrdersOnSymbol(t.Context(), "", true)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.MarginAccountCancelAllOpenOrdersOnSymbol(t.Context(), "BTCUSDT", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUnmarshalJSONForAssetIndex(t *testing.T) {
	t.Parallel()
	var resp AssetIndexResponse
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.ChangePositionMode(t.Context(), false)
	assert.NoError(t, err)
}

func TestGetCurrentPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrentPositionMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ---------------------------  European Option Endpoints test -----------------------------------

func TestCheckEOptionsServerTime(t *testing.T) {
	t.Parallel()
	serverTime, err := b.CheckEOptionsServerTime(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, serverTime)
}

func TestGetEOptionsOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetEOptionsOrderbook(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.GetEOptionsOrderbook(t.Context(), optionsTradablePair.String(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetEOptionsRecentTrades(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetEOptionsRecentTrades(t.Context(), "BTC-240330-80500-P", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetEOptionsTradeHistory(t.Context(), "", 0, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetEOptionsTradeHistory(t.Context(), "BTC-240330-80500-P", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := b.GetEOptionsCandlesticks(t.Context(), "", kline.OneDay, time.Time{}, time.Time{}, 1000)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.GetEOptionsCandlesticks(t.Context(), optionsTradablePair.String(), 0, time.Time{}, time.Time{}, 1000)
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	start, end := time.UnixMilli(1744459370269), time.UnixMilli(1744549370269)
	if !mockTests {
		start, end = time.Now().Add(-time.Hour*25), time.Now()
	}
	result, err := b.GetEOptionsCandlesticks(t.Context(), optionsTradablePair.String(), kline.OneDay, start, end, 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionMarkPrice(t *testing.T) {
	t.Parallel()
	optionsTradablePairString := "ETH-240927-3800-P"
	if !mockTests {
		optionsTradablePairString = optionsTradablePair.String()
	}
	result, err := b.GetOptionMarkPrice(t.Context(), optionsTradablePairString)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptions24hrTickerPriceChangeStatistics(t *testing.T) {
	t.Parallel()
	optionsTradablePairString := "ETH-240927-3800-P"
	if !mockTests {
		optionsTradablePairString = optionsTradablePair.String()
	}
	result, err := b.GetEOptions24hrTickerPriceChangeStatistics(t.Context(), optionsTradablePairString)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetEOptionsSymbolPriceTicker(t.Context(), "")
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	result, err := b.GetEOptionsSymbolPriceTicker(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsHistoricalExerciseRecords(t *testing.T) {
	t.Parallel()
	result, err := b.GetEOptionsHistoricalExerciseRecords(t.Context(), "BTCUSDT", time.Time{}, time.Now(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsOpenInterests(t *testing.T) {
	t.Parallel()
	expTime := time.UnixMilli(1744633637579)
	if !mockTests {
		expTime = time.Now().Add(time.Hour * 24)
	}
	_, err := b.GetEOptionsOpenInterests(t.Context(), currency.EMPTYCODE, expTime)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.GetEOptionsOpenInterests(t.Context(), currency.ETH, time.Time{})
	require.ErrorIs(t, err, errExpirationTimeRequired)

	result, err := b.GetEOptionsOpenInterests(t.Context(), currency.ETH, expTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOptionsAccountInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOptionsOrder(t *testing.T) {
	t.Parallel()
	arg := &OptionsOrderParams{}
	_, err := b.NewOptionsOrder(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.PostOnly = true
	_, err = b.NewOptionsOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.Pair{Base: currency.NewCode("BTC"), Delimiter: currency.DashDelimiter, Quote: currency.NewCode("200730-9000-C")}
	_, err = b.NewOptionsOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = b.NewOptionsOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = b.NewOptionsOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewOptionsOrder(t.Context(), &OptionsOrderParams{
		Symbol:                  currency.Pair{Base: currency.NewCode("BTC"), Delimiter: currency.DashDelimiter, Quote: currency.NewCode("200730-9000-C")},
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
	arg := OptionsOrderParams{}
	_, err := b.PlaceBatchEOptionsOrder(t.Context(), []OptionsOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = b.PlaceBatchEOptionsOrder(t.Context(), []OptionsOrderParams{arg})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.PostOnly = true
	_, err = b.PlaceBatchEOptionsOrder(t.Context(), []OptionsOrderParams{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.Pair{Base: currency.BTC, Delimiter: currency.DashDelimiter, Quote: currency.NewCode("200730-9000-C")}
	_, err = b.PlaceBatchEOptionsOrder(t.Context(), []OptionsOrderParams{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = b.PlaceBatchEOptionsOrder(t.Context(), []OptionsOrderParams{arg})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = b.PlaceBatchEOptionsOrder(t.Context(), []OptionsOrderParams{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.PlaceBatchEOptionsOrder(t.Context(), []OptionsOrderParams{
		{
			Symbol:                  currency.Pair{Base: currency.BTC, Delimiter: currency.DashDelimiter, Quote: currency.NewCode("200730-9000-C")},
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
			Symbol:                  currency.Pair{Base: currency.BTC, Delimiter: currency.DashDelimiter, Quote: currency.NewCode("200730-9000-C")},
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
	_, err := b.GetSingleEOptionsOrder(t.Context(), "", "", 4611875134427365377)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.GetSingleEOptionsOrder(t.Context(), "BTC-200730-9000-C", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSingleEOptionsOrder(t.Context(), "BTC-200730-9000-C", "", 4611875134427365377)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOptionsOrder(t *testing.T) {
	t.Parallel()
	_, err := b.CancelOptionsOrder(t.Context(), "", "213123", "4611875134427365377")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.CancelOptionsOrder(t.Context(), "BTC-200730-9000-C", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelOptionsOrder(t.Context(), "BTC-200730-9000-C", "213123", "4611875134427365377")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelBatchOptionsOrders(t *testing.T) {
	t.Parallel()
	_, err := b.CancelBatchOptionsOrders(t.Context(), "", []int64{4611875134427365377}, []string{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.CancelBatchOptionsOrders(t.Context(), "BTC-200730-9000-C", []int64{}, []string{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelBatchOptionsOrders(t.Context(), "BTC-200730-9000-C", []int64{4611875134427365377}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOptionOrdersOnSpecificSymbol(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.CancelAllOptionOrdersOnSpecificSymbol(t.Context(), "BTC-200730-9000-C")
	assert.NoError(t, err)
}

func TestCancelAllOptionsOrdersByUnderlying(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllOptionsOrdersByUnderlying(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentOpenOptionsOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	results, err := b.GetCurrentOpenOptionsOrders(t.Context(), "BTC-200730-9000-C", time.Time{}, time.Time{}, 4611875134427365377, 0)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestGetOptionsOrdersHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	results, err := b.GetOptionsOrdersHistory(t.Context(), "BTC-200730-9000-C", time.Time{}, time.Time{}, 4611875134427365377, 0)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestGetOptionPositionInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOptionPositionInformation(t.Context(), "BTC-200730-9000-C")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsAccountTradeList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetEOptionsAccountTradeList(t.Context(), "BTC-200730-9000-C", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserOptionsExerciseRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserOptionsExerciseRecord(t.Context(), "BTC-200730-9000-C", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingFlow(t *testing.T) {
	t.Parallel()
	_, err := b.GetAccountFundingFlow(t.Context(), currency.EMPTYCODE, 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountFundingFlow(t.Context(), currency.USDT, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDownloadIDForOptionTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDownloadIDForOptionTransactionHistory(t.Context(), time.Now().Add(-time.Hour*24*10), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionTransactionHistoryDownloadLinkByID(t *testing.T) {
	t.Parallel()
	_, err := b.GetOptionTransactionHistoryDownloadLinkByID(t.Context(), "")
	require.ErrorIs(t, err, errDownloadIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOptionTransactionHistoryDownloadLinkByID(t.Context(), "download-id")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionMarginAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOptionMarginAccountInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetMarketMakerProtectionConfig(t *testing.T) {
	t.Parallel()
	_, err := b.SetOptionsMarketMakerProtectionConfig(t.Context(), &MarketMakerProtectionConfig{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = b.SetOptionsMarketMakerProtectionConfig(t.Context(), &MarketMakerProtectionConfig{
		WindowTimeInMilliseconds: 3000,
		FrozenTimeInMilliseconds: 300000,
		QuantityLimit:            1.5,
		NetDeltaLimit:            1.5,
	})
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SetOptionsMarketMakerProtectionConfig(t.Context(), &MarketMakerProtectionConfig{
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
	_, err := b.GetOptionsMarketMakerProtection(t.Context(), "")
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOptionsMarketMakerProtection(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetMarketMaketProtection(t *testing.T) {
	t.Parallel()
	_, err := b.ResetMarketMaketProtection(t.Context(), "")
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.ResetMarketMaketProtection(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetOptionsAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := b.SetOptionsAutoCancelAllOpenOrders(t.Context(), "", 30000)
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.SetOptionsAutoCancelAllOpenOrders(t.Context(), "BTCUSDT", 30000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoCancelAllOpenOrdersConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAutoCancelAllOpenOrdersConfig(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsAutoCancelAllOpenOrdersHeartbeat(t *testing.T) {
	t.Parallel()
	_, err := b.GetOptionsAutoCancelAllOpenOrdersHeartbeat(t.Context(), []string{})
	require.ErrorIs(t, err, errUnderlyingIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetOptionsAutoCancelAllOpenOrdersHeartbeat(t.Context(), []string{"ETHUSDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsOptionsConnect(t *testing.T) {
	t.Parallel()
	err := b.WsOptionsConnect()
	assert.NoError(t, err)
}

func TestGetOptionsExchangeInformation(t *testing.T) {
	t.Parallel()
	exchangeinformation, err := b.GetOptionsExchangeInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, exchangeinformation)
}

// ---------------------------------------   Portfolio Margin  ---------------------------------------------

func TestNewUMOrder(t *testing.T) {
	t.Parallel()
	_, err := b.NewUMOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &UMOrderParam{ReduceOnly: true}
	_, err = b.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTCUSDT"
	_, err = b.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = b.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = b.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, errTimeInForceRequired)

	arg.TimeInForce = "GTC"
	_, err = b.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Quantity = 1.
	_, err = b.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 1234
	arg.OrderType = "market"
	arg.Quantity = 0
	_, err = b.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.OrderType = "stop"
	_, err = b.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewUMOrder(t.Context(), &UMOrderParam{
		Symbol:       "BTCUSDT",
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
	_, err := b.NewCMOrder(t.Context(), &UMOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &UMOrderParam{
		ReduceOnly: true,
	}
	_, err = b.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTCUSDT"
	_, err = b.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = b.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "OCO"
	_, err = b.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.OrderType = "MARKET"
	_, err = b.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.OrderType = order.Limit.String()
	_, err = b.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, errTimeInForceRequired)

	arg.TimeInForce = "GTC"
	_, err = b.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Quantity = .1
	_, err = b.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewCMOrder(t.Context(), &UMOrderParam{
		Symbol:       "BTCUSDT",
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
	_, err := b.NewMarginOrder(t.Context(), &MarginOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &MarginOrderParam{
		TimeInForce: "GTC",
	}
	_, err = b.NewMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = spotTradablePair.String()
	_, err = b.NewMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = b.NewMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewMarginOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountBorrow(t *testing.T) {
	t.Parallel()
	_, err := b.MarginAccountBorrow(t.Context(), currency.EMPTYCODE, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.MarginAccountBorrow(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.MarginAccountBorrow(t.Context(), currency.USDT, 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountRepay(t *testing.T) {
	t.Parallel()
	_, err := b.MarginAccountRepay(t.Context(), currency.EMPTYCODE, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.MarginAccountRepay(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.MarginAccountRepay(t.Context(), currency.USDT, 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountNewOCO(t *testing.T) {
	t.Parallel()
	_, err := b.MarginAccountNewOCO(t.Context(), &OCOOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &OCOOrderParam{
		TrailingDelta: 1,
	}
	_, err = b.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.NewPair(currency.BTC, currency.USDT)
	_, err = b.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Buy"
	_, err = b.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Amount = 0.1
	_, err = b.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 0.001
	_, err = b.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewOCOOrder(t.Context(), &OCOOrderParam{
		Symbol:             currency.NewPair(currency.BTC, currency.USDT),
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
	_, err := b.NewOCOOrderList(t.Context(), &OCOOrderListParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &OCOOrderListParams{
		AboveTimeInForce: "GTC",
	}
	_, err = b.NewOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "LTCBTC"
	_, err = b.NewOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = b.NewOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Quantity = 1
	_, err = b.NewOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.AboveType = "STOP_LOSS_LIMIT"
	_, err = b.NewOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.NewOCOOrderList(t.Context(), &OCOOrderListParams{
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
	_, err := b.NewUMConditionalOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &ConditionalOrderParam{PriceProtect: true}
	_, err = b.NewUMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTCUSDT"
	_, err = b.NewUMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = b.NewUMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, errStrategyTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewUMConditionalOrder(t.Context(), &ConditionalOrderParam{
		Symbol:       "BTCUSDT",
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
	_, err := b.NewCMConditionalOrder(t.Context(), &ConditionalOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &ConditionalOrderParam{
		PositionSide: "LONG",
	}
	_, err = b.NewCMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTCUSD_200925"
	_, err = b.NewCMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Buy"
	_, err = b.NewCMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, errStrategyTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewCMConditionalOrder(t.Context(), &ConditionalOrderParam{
		Symbol:       "BTCUSD_200925",
		Side:         "Buy",
		PositionSide: "LONG",
		StrategyType: "TAKE_PROFIT",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelUMOrder(t *testing.T) {
	t.Parallel()
	_, err := b.CancelUMOrder(t.Context(), "", "", 1234132)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.CancelUMOrder(t.Context(), "BTCUSDT", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelUMOrder(t.Context(), "BTCUSDT", "", 1234132)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelCMOrder(t *testing.T) {
	t.Parallel()
	_, err := b.CancelCMOrder(t.Context(), "", "", 21321312)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.CancelCMOrder(t.Context(), "BTCUSDT", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelCMOrder(t.Context(), "BTCUSDT", "", 21321312)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllUMOrders(t *testing.T) {
	t.Parallel()
	_, err := b.CancelAllUMOrders(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllUMOrders(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 200, result.Code)
}

func TestCancelAllCMOrders(t *testing.T) {
	t.Parallel()
	_, err := b.CancelAllCMOrders(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllCMOrders(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPMCancelMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := b.PMCancelMarginAccountOrder(t.Context(), "", "", 12314)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.PMCancelMarginAccountOrder(t.Context(), "LTCBTC", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.PMCancelMarginAccountOrder(t.Context(), "LTCBTC", "", 12314)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllMarginOpenOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := b.CancelAllMarginOpenOrdersBySymbol(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllMarginOpenOrdersBySymbol(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMarginAccountOCOOrders(t *testing.T) {
	t.Parallel()
	_, err := b.CancelMarginAccountOCOOrders(t.Context(), "", "", "", 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelMarginAccountOCOOrders(t.Context(), "LTCBTC", "", "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelUMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := b.CancelUMConditionalOrder(t.Context(), "", "", 2000)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.CancelUMConditionalOrder(t.Context(), "LTCBTC", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelUMConditionalOrder(t.Context(), "LTCBTC", "", 2000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelCMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := b.CancelCMConditionalOrder(t.Context(), "", "", 1231231)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.CancelCMConditionalOrder(t.Context(), "LTCBTC", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelCMConditionalOrder(t.Context(), "LTCBTC", "", 1231231)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllUMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	_, err := b.CancelAllUMOpenConditionalOrders(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllUMOpenConditionalOrders(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllCMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	_, err := b.CancelAllCMOpenConditionalOrders(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllCMOpenConditionalOrders(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMOrder(t *testing.T) {
	t.Parallel()
	_, err := b.GetUMOrder(t.Context(), "", "", 1234)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.GetUMOrder(t.Context(), "BTCUSDT", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMOrder(t.Context(), "BTCUSDT", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := b.GetUMOpenOrder(t.Context(), "", "", 1234)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.GetUMOpenOrder(t.Context(), "BTCUSDT", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMOpenOrder(t.Context(), "BTCUSDT", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllUMOpenOrders(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllUMOrders(t.Context(), "BTCUSDT", time.Now().Add(-time.Hour*24*6), time.Now().Add(-time.Hour*2), 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMOrder(t *testing.T) {
	t.Parallel()
	_, err := b.GetCMOrder(t.Context(), "", "", 1234)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMOrder(t.Context(), "BTCLTC", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := b.GetCMOpenOrder(t.Context(), "", "", 1234)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.GetCMOpenOrder(t.Context(), "BTCLTC", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMOpenOrder(t.Context(), "BTCLTC", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetAllCMOpenOrders(t.Context(), "", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCMOpenOrders(t.Context(), "BTCUSD_200925", "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetAllCMOrders(t.Context(), "", "", time.Time{}, time.Time{}, 0, 20)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCMOrders(t.Context(), "BTCUSD_200925", "BTCUSD", time.Time{}, time.Time{}, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenUMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenUMConditionalOrder(t.Context(), "BTCUSDT", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOpenUMConditionalOrder(t.Context(), "BTCUSDT", "newClientStrategyId", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllUMOpenConditionalOrders(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMConditionalOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllUMConditionalOrderHistory(t.Context(), "BTCUSDT", "abc", 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllUMConditionalOrders(t.Context(), "BTCUSDT", time.Time{}, time.Now(), 0, 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenCMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenCMConditionalOrder(t.Context(), "BTCUSD", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOpenCMConditionalOrder(t.Context(), "BTCUSD", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCMOpenConditionalOrders(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMConditionalOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCMConditionalOrderHistory(t.Context(), "BTCUSDT", "abc", 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCMConditionalOrders(t.Context(), "BTCUSDT", time.Time{}, time.Now(), 0, 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginAccountOrder(t.Context(), "", "", 12434)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.GetMarginAccountOrder(t.Context(), "BNBBTC", "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountOrder(t.Context(), "BNBBTC", "", 12434)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentMarginOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrentMarginOpenOrder(t.Context(), "BNBBTC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllMarginAccountOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetAllMarginAccountOrders(t.Context(), "", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllMarginAccountOrders(t.Context(), "BNBBTC", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountOCO(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountOCO(t.Context(), 0, "123421-abcde")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPMMarginAccountAllOCO(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPMMarginAccountAllOCO(t.Context(), time.Now().Add(-time.Hour*24), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOpenOCO(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountsOpenOCO(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPMMarginAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := b.GetPMMarginAccountTradeList(t.Context(), "", time.Time{}, time.Time{}, 0, 0, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPMMarginAccountTradeList(t.Context(), "BNBBTC", time.Time{}, time.Time{}, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountBalance(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPortfolioMarginAccountInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginMaxBorrow(t *testing.T) {
	t.Parallel()
	_, err := b.GetPMMarginMaxBorrow(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPMMarginMaxBorrow(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginMaxWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginMaxWithdrawal(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginMaxWithdrawal(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMPositionInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMPositionInformation(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMPositionInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMPositionInformation(t.Context(), currency.ETH, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeUMInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := b.ChangeUMInitialLeverage(t.Context(), "", 29)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.ChangeUMInitialLeverage(t.Context(), "BTCUSDT", 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeUMInitialLeverage(t.Context(), "BTCUSDT", 29)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeCMInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := b.ChangeCMInitialLeverage(t.Context(), "", 29)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.ChangeCMInitialLeverage(t.Context(), "BTCUSDT", 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeCMInitialLeverage(t.Context(), "BTCUSDT", 29)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeUMPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeUMPositionMode(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeCMPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeCMPositionMode(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMCurrentPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMCurrentPositionMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMCurrentPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMCurrentPositionMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := b.GetUMAccountTradeList(t.Context(), "", time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMAccountTradeList(t.Context(), "BTCUSDT", time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := b.GetCMAccountTradeList(t.Context(), "", "", time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMAccountTradeList(t.Context(), "BTCUSD_200626", "BTCUSDT", time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMNotionalAndLeverageBrackets(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMNotionalAndLeverageBrackets(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersMarginForceOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUsersMarginForceOrders(t.Context(), time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersUMForceOrderst(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUsersUMForceOrders(t.Context(), "BTCUSDT", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersCMForceOrderst(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUsersCMForceOrders(t.Context(), "BTCUSDT", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginUMTradingQuantitativeRulesIndicator(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPortfolioMarginUMTradingQuantitativeRulesIndicator(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMUserCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := b.GetUMUserCommissionRate(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMUserCommissionRate(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMUserCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := b.GetCMUserCommissionRate(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMUserCommissionRate(t.Context(), "BTCUSD_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginLoanRecord(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginLoanRecord(t.Context(), currency.EMPTYCODE, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginLoanRecord(t.Context(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginRepayRecord(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginRepayRecord(t.Context(), currency.EMPTYCODE, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginRepayRecord(t.Context(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginBorrowOrLoanInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginBorrowOrLoanInterestHistory(t.Context(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginNegativeBalanceInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPortfolioMarginNegativeBalanceInterestHistory(t.Context(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundAutoCollection(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FundAutoCollection(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundCollectionByAsset(t *testing.T) {
	t.Parallel()
	_, err := b.FundCollectionByAsset(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FundCollectionByAsset(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBNBTransferClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.BNBTransferClassic(t.Context(), 0.0001, "TO_UM")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBNBTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.BNBTransfer(t.Context(), 0.0001, "TO_UM")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMAccountDetail(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMAccountDetail(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.ChangeAutoRepayFuturesStatus(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAutoRepayFuturesStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayFuturesNegativeBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RepayFuturesNegativeBalance(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMPositionADLQuantileEstimation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMPositionADLQuantileEstimation(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMPositionADLQuantileEstimation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMPositionADLQuantileEstimation(t.Context(), "BTCUSD_200925")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserRateLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserRateLimits(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginAssetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetPortfolioMarginAssetIndexPrice(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPortfolioMarginAssetIndexPrice(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAdjustCrossMarginMaxLeverage(t *testing.T) {
	t.Parallel()
	_, err := b.AdjustCrossMarginMaxLeverage(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.AdjustCrossMarginMaxLeverage(t.Context(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCrossMarginTransferHistory(t.Context(), currency.ETH, "ROLL_IN", "", time.Time{}, time.Time{}, 10, 30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewMarginAccountOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := b.NewMarginAccountOCOOrder(t.Context(), &MarginOCOOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &MarginOCOOrderParam{
		IsIsolated: true,
	}

	_, err = b.NewMarginAccountOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.NewPair(currency.BTC, currency.USDT)
	_, err = b.NewMarginAccountOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy.String()
	_, err = b.NewMarginAccountOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Quantity = 0.000001
	_, err = b.NewMarginAccountOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 12312
	_, err = b.NewMarginAccountOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewMarginAccountOCOOrder(t.Context(), &MarginOCOOrderParam{
		Symbol:    currency.NewPair(currency.BTC, currency.USDT),
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
	_, err := b.CancelMarginAccountOCOOrder(t.Context(), "", "12345678", "", true, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelMarginAccountOCOOrder(t.Context(), "LTCBTC", "12345678", "", true, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountOCOOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountOCOOrder(t.Context(), "LTCBTC", "12345", 0, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountAllOCO(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountAllOCO(t.Context(), "LTCBTC", true, time.Now().Add(-time.Hour*24), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOpenOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginAccountsOpenOCOOrder(t.Context(), true, "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountsOpenOCOOrder(t.Context(), true, usdtmTradablePair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginAccountTradeList(t.Context(), "", true, time.Time{}, time.Time{}, 0, 0, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountTradeList(t.Context(), "BNBBTC", true, time.Time{}, time.Time{}, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaxBorrow(t *testing.T) {
	t.Parallel()
	_, err := b.GetMaxBorrow(t.Context(), currency.EMPTYCODE, "BTCETH")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMaxBorrow(t.Context(), currency.ETH, "BTCETH")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaxTransferOutAmount(t *testing.T) {
	t.Parallel()
	_, err := b.GetMaxTransferOutAmount(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMaxTransferOutAmount(t.Context(), currency.ETH, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSummaryOfMarginAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIsolatedMarginAccountInfo(t.Context(), []string{"BTCUSDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDisableIsolatedMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := b.DisableIsolatedMarginAccount(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.DisableIsolatedMarginAccount(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableIsolatedMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := b.EnableIsolatedMarginAccount(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.EnableIsolatedMarginAccount(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEnabledIsolatedMarginAccountLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetEnabledIsolatedMarginAccountLimit(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllIsolatedMarginSymbols(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllIsolatedMarginSymbols(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestToggleBNBBurn(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ToggleBNBBurn(t.Context(), true, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBNBBurnStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetBNBBurnStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginInterestRateHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginInterestRateHistory(t.Context(), currency.EMPTYCODE, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginInterestRateHistory(t.Context(), currency.ETH, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginFeeData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCrossMarginFeeData(t.Context(), 0, currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMaringFeeData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIsolatedMaringFeeData(t.Context(), 1, "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginTierData(t *testing.T) {
	t.Parallel()
	_, err := b.GetIsolatedMarginTierData(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIsolatedMarginTierData(t.Context(), "BTCUSDT", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyMarginOrderCountUsage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrencyMarginOrderCountUsage(t.Context(), true, "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCrossMarginCollateralRatio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCrossMarginCollateralRatio(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmallLiabilityExchangeCoinList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSmallLiabilityExchangeCoinList(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginSmallLiabilityExchange(t *testing.T) {
	t.Parallel()
	_, err := b.MarginSmallLiabilityExchange(t.Context(), []string{})
	require.ErrorIs(t, err, errEmptyCurrencyCodes)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.MarginSmallLiabilityExchange(t.Context(), []string{"BTC", "ETH"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmallLiabilityExchangeHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetSmallLiabilityExchangeHistory(t.Context(), 0, 10, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errPageNumberRequired)
	_, err = b.GetSmallLiabilityExchangeHistory(t.Context(), 1, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errPageSizeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSmallLiabilityExchangeHistory(t.Context(), 1, 10, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFutureHourlyInterestRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFutureHourlyInterestRate(t.Context(), []string{"BTC", "ETH"}, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossOrIsolatedMarginCapitalFlow(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCrossOrIsolatedMarginCapitalFlow(t.Context(), currency.ETH, "", "BORROW", time.Time{}, time.Time{}, 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTokensOrSymbolsDelistSchedule(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetTokensOrSymbolsDelistSchedule(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAvailableInventory(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginAvailableInventory(t.Context(), "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAvailableInventory(t.Context(), "ISOLATED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginManualLiquidiation(t *testing.T) {
	t.Parallel()
	_, err := b.MarginManualLiquidiation(t.Context(), "", "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.MarginManualLiquidiation(t.Context(), "ISOLATED", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLiabilityCoinLeverageBracketInCrossMarginProMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLiabilityCoinLeverageBracketInCrossMarginProMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSimpleEarnFlexibleProductList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSimpleEarnFlexibleProductList(t.Context(), currency.BTC, 2, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSimpleEarnLockedProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSimpleEarnLockedProducts(t.Context(), currency.BTC, 2, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeToFlexibleProducts(t *testing.T) {
	t.Parallel()
	_, err := b.SubscribeToFlexibleProducts(t.Context(), "", "FUND", 1, false)
	require.ErrorIs(t, err, errProductIDRequired)
	_, err = b.SubscribeToFlexibleProducts(t.Context(), "project-id", "FUND", 0, false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubscribeToFlexibleProducts(t.Context(), "product-id", "FUND", 1, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeToLockedProducts(t *testing.T) {
	t.Parallel()
	_, err := b.SubscribeToLockedProducts(t.Context(), "", "SPOT", 1, false)
	require.ErrorIs(t, err, errProjectIDRequired)
	_, err = b.SubscribeToLockedProducts(t.Context(), "project-id", "SPOT", 0, false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubscribeToLockedProducts(t.Context(), "project-id", "SPOT", 1, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemFlexibleProduct(t *testing.T) {
	t.Parallel()
	_, err := b.RedeemFlexibleProduct(t.Context(), "", "FUND", true, 0.1234)
	require.ErrorIs(t, err, errProductIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RedeemFlexibleProduct(t.Context(), "product-id", "FUND", true, 0.1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemLockedProduct(t *testing.T) {
	t.Parallel()
	_, err := b.RedeemLockedProduct(t.Context(), 0)
	require.ErrorIs(t, err, errPositionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RedeemLockedProduct(t.Context(), 12345)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleProductPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleProductPosition(t.Context(), currency.BTC, "", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedProductPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedProductPosition(t.Context(), currency.ETH, "", "", 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSimpleAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.SimpleAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleSubscriptionRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleSubscriptionRecord(t.Context(), "", "", currency.ETH, time.Now().Add(-time.Hour*48), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedSubscriptionsRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedSubscriptionsRecords(t.Context(), "", currency.ETH, time.Now().Add(-time.Hour*480), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleRedemptionRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleRedemptionRecord(t.Context(), "", "1234", currency.LTC, time.Now().Add(-time.Hour*48), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedRedemptionRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedRedemptionRecord(t.Context(), "", "1234", currency.LTC, time.Now().Add(-time.Hour*48), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleRewardHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleRewardHistory(t.Context(), "product-type", "", currency.BTC, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2), 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedRewardHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedRewardHistory(t.Context(), "12345", currency.BTC, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2), 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetFlexibleAutoSusbcribe(t *testing.T) {
	t.Parallel()
	_, err := b.SetFlexibleAutoSusbcribe(t.Context(), "", true)
	require.ErrorIs(t, err, errProductIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SetFlexibleAutoSusbcribe(t.Context(), "product-id", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLockedAutoSubscribe(t *testing.T) {
	t.Parallel()
	_, err := b.SetLockedAutoSubscribe(t.Context(), "", true)
	require.ErrorIs(t, err, errPositionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SetLockedAutoSubscribe(t.Context(), "position-id", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexiblePersonalLeftQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexiblePersonalLeftQuota(t.Context(), "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedPersonalLeftQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedPersonalLeftQuota(t.Context(), "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleSubscriptionPreview(t *testing.T) {
	t.Parallel()
	_, err := b.GetFlexibleSubscriptionPreview(t.Context(), "", 0.0001)
	require.ErrorIs(t, err, errProductIDRequired)
	_, err = b.GetFlexibleSubscriptionPreview(t.Context(), "1234", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleSubscriptionPreview(t.Context(), "1234", 0.0001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedSubscriptionPreview(t *testing.T) {
	t.Parallel()
	_, err := b.GetLockedSubscriptionPreview(t.Context(), "", 0.1234, false)
	require.ErrorIs(t, err, errProjectIDRequired)
	_, err = b.GetLockedSubscriptionPreview(t.Context(), "12345", 0, false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedSubscriptionPreview(t.Context(), "12345", 0.1234, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLockedProductRedeemOption(t *testing.T) {
	t.Parallel()
	_, err := b.SetLockedProductRedeemOption(t.Context(), "", "abcdefg")
	require.ErrorIs(t, err, errPositionIDRequired)
	_, err = b.SetLockedProductRedeemOption(t.Context(), "12345", "")
	require.ErrorIs(t, err, errRedemptionAccountRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err = b.SetLockedProductRedeemOption(t.Context(), "12345", "abcdefg")
	assert.NoError(t, err)
}

func TestGetSimpleEarnRatehistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSimpleEarnRatehistory(t.Context(), "project-id", time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSimpleEarnCollateralRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSimpleEarnCollateralRecord(t.Context(), "project-id", time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDualInvestmentProductList(t *testing.T) {
	t.Parallel()
	_, err := b.GetDualInvestmentProductList(t.Context(), "", currency.BTC, currency.ETH, 0, 0)
	require.ErrorIs(t, err, errOptionTypeRequired)
	_, err = b.GetDualInvestmentProductList(t.Context(), "CALL", currency.EMPTYCODE, currency.ETH, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.GetDualInvestmentProductList(t.Context(), "CALL", currency.BTC, currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetDualInvestmentProductList(t.Context(), "CALL", currency.BTC, currency.ETH, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeDualInvestmentProducts(t *testing.T) {
	t.Parallel()
	_, err := b.SubscribeDualInvestmentProducts(t.Context(), "", "order-id", "STANDARD", 0.1)
	require.ErrorIs(t, err, errProductIDRequired)
	_, err = b.SubscribeDualInvestmentProducts(t.Context(), "1234", "", "STANDARD", 0.1)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = b.SubscribeDualInvestmentProducts(t.Context(), "1234", "order-id", "STANDARD", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.SubscribeDualInvestmentProducts(t.Context(), "1234", "order-id", "", 1)
	require.ErrorIs(t, err, errPlanTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubscribeDualInvestmentProducts(t.Context(), "1234", "order-id", "STANDARD", 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDualInvestmentPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDualInvestmentPositions(t.Context(), "PURCHASE_FAIL", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckDualInvestmentAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CheckDualInvestmentAccounts(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAutoCompoundStatus(t *testing.T) {
	t.Parallel()
	_, err := b.ChangeAutoCompoundStatus(t.Context(), "", "STANDARD")
	require.ErrorIs(t, err, errPositionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeAutoCompoundStatus(t.Context(), "123456789", "STANDARD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTargetAssetList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetTargetAssetList(t.Context(), currency.BTC, 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTargetAssetROIData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetTargetAssetROIData(t.Context(), currency.ETH, "THREE_YEAR")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllSourceAssetAndTargetAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllSourceAssetAndTargetAsset(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSourceAssetList(t *testing.T) {
	t.Parallel()
	_, err := b.GetSourceAssetList(t.Context(), currency.BTC, 123, "", "MAIN_SITE", true)
	require.ErrorIs(t, err, errUsageTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSourceAssetList(t.Context(), currency.BTC, 123, "RECURRING", "MAIN_SITE", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInvestmentPlanCreation(t *testing.T) {
	t.Parallel()
	_, err := b.InvestmentPlanCreation(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &InvestmentPlanParams{}
	_, err = b.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, errSourceTypeRequired)

	arg.SourceType = "MAIN_SITE"
	_, err = b.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, errPlanTypeRequired)

	arg.PlanType = "SINGLE"
	_, err = b.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.SubscriptionAmount = 4
	_, err = b.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubscriptionStartTime)

	arg.SubscriptionStartDay = 1
	arg.SubscriptionStartTime = 8
	_, err = b.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.SourceAsset = currency.USDT
	_, err = b.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, errPortfolioDetailRequired)

	arg.Details = []PortfolioDetail{{}}
	_, err = b.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Details = []PortfolioDetail{{TargetAsset: currency.BTC, Percentage: -1}}
	_, err = b.InvestmentPlanCreation(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPercentageAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.InvestmentPlanCreation(t.Context(), &InvestmentPlanParams{
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
	_, err := b.InvestmentPlanAdjustment(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &AdjustInvestmentPlan{}
	_, err = b.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errPlanIDRequired)

	arg.PlanID = 1234232
	_, err = b.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.SubscriptionAmount = 4
	_, err = b.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubscriptionCycle)

	arg.SubscriptionCycle = "H4"
	arg.SubscriptionStartTime = -1
	_, err = b.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubscriptionStartTime)

	arg.SubscriptionStartTime = 8
	_, err = b.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.SourceAsset = currency.USDT
	_, err = b.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errPortfolioDetailRequired)

	arg.Details = []PortfolioDetail{{}}
	_, err = b.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Details = []PortfolioDetail{{TargetAsset: currency.BTC, Percentage: -1}}
	_, err = b.InvestmentPlanAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPercentageAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.InvestmentPlanAdjustment(t.Context(), &AdjustInvestmentPlan{
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
	_, err := b.ChangePlanStatus(t.Context(), 0, "PAUSED")
	require.ErrorIs(t, err, errPlanIDRequired)

	_, err = b.ChangePlanStatus(t.Context(), 12345, "")
	require.ErrorIs(t, err, errPlanStatusRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangePlanStatus(t.Context(), 12345, "PAUSED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetListOfPlans(t *testing.T) {
	t.Parallel()
	_, err := b.GetListOfPlans(t.Context(), "")
	require.ErrorIs(t, err, errPlanTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetListOfPlans(t.Context(), "SINGLE")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHoldingDetailsOfPlan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetHoldingDetailsOfPlan(t.Context(), 1234, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubscriptionsTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubscriptionsTransactionHistory(t.Context(), 1232, 20, 0, time.Time{}, time.Time{}, currency.BTC, "PORTFOLIO")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexDetail(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexDetail(t.Context(), 0)
	require.ErrorIs(t, err, errIndexIDIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIndexDetail(t.Context(), 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexLinkedPlanPositionDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexLinkedPlanPositionDetails(t.Context(), 0)
	require.ErrorIs(t, err, errIndexIDIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIndexLinkedPlanPositionDetails(t.Context(), 123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOneTimeTransaction(t *testing.T) {
	t.Parallel()
	_, err := b.OneTimeTransaction(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &OneTimeTransactionParams{}
	_, err = b.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, errSourceTypeRequired)

	arg.SourceType = "MAIN_SITE"
	_, err = b.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.SubscriptionAmount = 12
	_, err = b.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.SourceAsset = currency.USDT
	_, err = b.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, errPortfolioDetailRequired)

	_, err = b.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, errPortfolioDetailRequired)

	arg.Details = []PortfolioDetail{{}}
	_, err = b.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Details = []PortfolioDetail{{TargetAsset: currency.BTC}}
	_, err = b.OneTimeTransaction(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPercentageAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.OneTimeTransaction(t.Context(), &OneTimeTransactionParams{
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
	_, err := b.GetOneTimeTransactionStatus(t.Context(), 0, "")
	require.ErrorIs(t, err, errTransactionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOneTimeTransactionStatus(t.Context(), 1234, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIndexLinkedPlanRedemption(t *testing.T) {
	t.Parallel()
	_, err := b.IndexLinkedPlanRedemption(t.Context(), 0, 30, "")
	require.ErrorIs(t, err, errIndexIDIsRequired)
	_, err = b.IndexLinkedPlanRedemption(t.Context(), 12333, 0, "")
	require.ErrorIs(t, err, errInvalidPercentageAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.IndexLinkedPlanRedemption(t.Context(), 12333, 30, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexLinkedPlanRedemption(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexLinkedPlanRedemption(t.Context(), "", time.Now().Add(-time.Hour*48), time.Now(), currency.ETH, 0, 10)
	require.ErrorIs(t, err, errRequestIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetIndexLinkedPlanRedemption(t.Context(), "123123", time.Now().Add(-time.Hour*48), time.Now(), currency.ETH, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexLinkedPlanRebalanceDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIndexLinkedPlanRebalanceDetails(t.Context(), time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubscribeETHStaking(t *testing.T) {
	t.Parallel()
	_, err := b.GetSubscribeETHStaking(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubscribeETHStaking(t.Context(), 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSusbcribeETHStakingV2(t *testing.T) {
	t.Parallel()
	_, err := b.SusbcribeETHStakingV2(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.SusbcribeETHStakingV2(t.Context(), 0.123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemETH(t *testing.T) {
	t.Parallel()
	_, err := b.RedeemETH(t.Context(), 0, currency.ETH)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RedeemETH(t.Context(), 0.123, currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHStakingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetETHStakingHistory(t.Context(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHRedemptionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetETHRedemptionHistory(t.Context(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBETHRewardsDistributionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBETHRewardsDistributionHistory(t.Context(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentETHStakingQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrentETHStakingQuota(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHRateHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWBETHRateHistory(t.Context(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHStakingAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetETHStakingAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHStakingAccountV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetETHStakingAccountV2(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWrapBETH(t *testing.T) {
	t.Parallel()
	_, err := b.WrapBETH(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.WrapBETH(t.Context(), 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHWrapHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWBETHWrapHistory(t.Context(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHUnwrapHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWBETHUnwrapHistory(t.Context(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHRewardHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWBETHRewardHistory(t.Context(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSOLStakingAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSOLStakingAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSOLStakingQuotaDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSOLStakingQuotaDetails(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeToSOLStaking(t *testing.T) {
	t.Parallel()
	_, err := b.SubscribeToSOLStaking(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubscribeToSOLStaking(t.Context(), 1.2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemSOL(t *testing.T) {
	t.Parallel()
	_, err := b.RedeemSOL(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RedeemSOL(t.Context(), 1.2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClaimBoostRewards(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ClaimBoostRewards(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSOLStakingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSOLStakingHistory(t.Context(), time.Now().Add(-time.Hour*30), time.Now(), 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSOLRedemptionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSOLRedemptionHistory(t.Context(), time.Now().Add(-time.Hour*30), time.Now(), 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBNSOLRewardsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBNSOLRewardsHistory(t.Context(), time.Now().Add(-time.Hour*30), time.Now(), 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBNSOLRateHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBNSOLRateHistory(t.Context(), time.Now().Add(-time.Hour*30), time.Now(), 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBoostRewardsHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetBoostRewardsHistory(t.Context(), "", time.Now().Add(-time.Hour*30), time.Now(), 0, 100)
	require.ErrorIs(t, err, errRewardTypeMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBoostRewardsHistory(t.Context(), "CLAIM", time.Now().Add(-time.Hour*30), time.Now(), 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnclaimedRewards(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUnclaimedRewards(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcquiringAlgorithm(t *testing.T) {
	t.Parallel()
	result, err := b.AcquiringAlgorithm(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCoinNames(t *testing.T) {
	t.Parallel()
	result, err := b.GetCoinNames(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailMinerList(t *testing.T) {
	t.Parallel()
	_, err := b.GetDetailMinerList(t.Context(), "sha256", "", "bhdc1.16A10404B")
	require.ErrorIs(t, err, errNameRequired)
	_, err = b.GetDetailMinerList(t.Context(), "", "sams", "bhdc1.16A10404B")
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = b.GetDetailMinerList(t.Context(), "sha256", "sams", "")
	require.ErrorIs(t, err, errNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDetailMinerList(t.Context(), "sha256", "sams", "bhdc1.16A10404B")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMinersList(t *testing.T) {
	t.Parallel()
	_, err := b.GetMinersList(t.Context(), "", "sams", true, 0, 10, 10)
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = b.GetMinersList(t.Context(), "sha256", "", true, 0, 10, 10)
	require.ErrorIs(t, err, errNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMinersList(t.Context(), "sha256", "sams", true, 0, 10, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarningList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetEarningList(t.Context(), "sha256", "sams", currency.ETH, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExtraBonousList(t *testing.T) {
	t.Parallel()
	_, err := b.ExtraBonousList(t.Context(), "", "sams", currency.ETH, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = b.ExtraBonousList(t.Context(), "sha256", "", currency.ETH, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errUsernameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.ExtraBonousList(t.Context(), "sha256", "sams", currency.ETH, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHashrateRescaleList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetHashrateRescaleList(t.Context(), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHashrateRescaleDetail(t *testing.T) {
	t.Parallel()
	_, err := b.GetHashRateRescaleDetail(t.Context(), "", "sams", 10, 20)
	require.ErrorIs(t, err, errConfigIDRequired)
	_, err = b.GetHashRateRescaleDetail(t.Context(), "168", "", 10, 20)
	require.ErrorIs(t, err, errUsernameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetHashRateRescaleDetail(t.Context(), "168", "sams", 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHashrateRescaleRequest(t *testing.T) {
	t.Parallel()
	_, err := b.HashRateRescaleRequest(t.Context(), "", "sha256", "S19pro", time.Time{}, time.Time{}, 10000)
	require.ErrorIs(t, err, errUsernameRequired)
	_, err = b.HashRateRescaleRequest(t.Context(), "sams", "", "S19pro", time.Time{}, time.Time{}, 10000)
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = b.HashRateRescaleRequest(t.Context(), "sams", "sha256", "S19pro", time.Now(), time.Time{}, 10000)
	require.ErrorIs(t, err, common.ErrDateUnset)
	_, err = b.HashRateRescaleRequest(t.Context(), "sams", "sha256", "", time.Now().Add(-time.Hour*240), time.Now(), 10000)
	require.ErrorIs(t, err, errAccountRequired)
	_, err = b.HashRateRescaleRequest(t.Context(), "sams", "sha256", "S19pro", time.Now().Add(-time.Hour*240), time.Now(), 0)
	require.ErrorIs(t, err, errHashRateRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.HashRateRescaleRequest(t.Context(), "sams", "sha256", "S19pro", time.Time{}, time.Time{}, 10000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelHashrateRescaleConfiguration(t *testing.T) {
	t.Parallel()
	_, err := b.CancelHashrateRescaleConfiguration(t.Context(), "", "sams")
	require.ErrorIs(t, err, errConfigIDRequired)
	_, err = b.CancelHashrateRescaleConfiguration(t.Context(), "189", "")
	require.ErrorIs(t, err, errUsernameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelHashrateRescaleConfiguration(t.Context(), "189", "sams")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestStatisticsList(t *testing.T) {
	t.Parallel()
	_, err := b.StatisticsList(t.Context(), "", "sams")
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = b.StatisticsList(t.Context(), "sha256", "")
	require.ErrorIs(t, err, errUsernameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.StatisticsList(t.Context(), "sha256", "sams")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountList(t *testing.T) {
	t.Parallel()
	_, err := b.GetAccountList(t.Context(), "", "sams")
	require.ErrorIs(t, err, errTransferAlgorithmRequired)
	_, err = b.GetAccountList(t.Context(), "sha256", "")
	require.ErrorIs(t, err, errUsernameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountList(t.Context(), "sha256", "sams")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMiningAccountEarningRate(t *testing.T) {
	t.Parallel()
	_, err := b.GetMiningAccountEarningRate(t.Context(), "", time.Now().Add(-time.Hour*240), time.Now(), 0, 10)
	require.ErrorIs(t, err, errTransferAlgorithmRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMiningAccountEarningRate(t.Context(), "sha256", time.Now().Add(-time.Hour*240), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewFuturesAccountTransfer(t *testing.T) {
	t.Parallel()
	_, err := b.NewFuturesAccountTransfer(t.Context(), currency.EMPTYCODE, 0.001, 2)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.NewFuturesAccountTransfer(t.Context(), currency.ETH, 0, 2)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.NewFuturesAccountTransfer(t.Context(), currency.ETH, 0.001, 0)
	require.ErrorIs(t, err, errTransferTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewFuturesAccountTransfer(t.Context(), currency.ETH, 0.001, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountTransactionHistoryList(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesAccountTransactionHistoryList(t.Context(), currency.BTC, time.Time{}, time.Time{}, 10, 20)
	require.ErrorIs(t, err, errStartTimeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetFuturesAccountTransactionHistoryList(t.Context(), currency.BTC, time.Now().Add(-time.Hour*20), time.Time{}, 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFutureTickLevelOrderbookHistoricalDataDownloadLink(t *testing.T) {
	t.Parallel()
	_, err := b.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), "", "T_DEPTH", time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*3))
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), "BTCUSDT", "T_DEPTH", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errStartTimeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), "BTCUSDT", "T_DEPTH", time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*3))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVolumeParticipationNewOrder(t *testing.T) {
	t.Parallel()
	_, err := b.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = b.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{Urgency: "HIGH"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{
		Symbol:       "BTCUSDT",
		PositionSide: "BOTH",
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = b.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{
		Symbol:       "BTCUSDT",
		Side:         order.Sell.String(),
		PositionSide: "BOTH",
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{
		Symbol:       "BTCUSDT",
		Side:         order.Sell.String(),
		PositionSide: "BOTH",
		Quantity:     0.012,
	})
	require.ErrorIs(t, err, errPossibleValuesRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.VolumeParticipationNewOrder(t.Context(), &VolumeParticipationOrderParams{
		Symbol:       "BTCUSDT",
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
	_, err := b.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = b.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{
		Duration: 1000,
	})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{
		Symbol: "BTCUSDT",
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = b.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{
		Symbol: "BTCUSDT",
		Side:   order.Sell.String(),
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{
		Symbol:   "BTCUSDT",
		Side:     order.Sell.String(),
		Quantity: 0.012,
	})
	require.ErrorIs(t, err, errDurationRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesTWAPOrder(t.Context(), &TWAPOrderParams{
		Symbol:       "BTCUSDT",
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
	_, err := b.CancelFuturesAlgoOrder(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelFuturesAlgoOrder(t.Context(), 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentAlgoOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesCurrentAlgoOpenOrders(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalAlgoOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesHistoricalAlgoOrders(t.Context(), "BNBUSDT", "BUY", time.Time{}, time.Time{}, 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesSubOrders(t.Context(), 0, 0, 40)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesSubOrders(t.Context(), 1234, 0, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTWAPNewOrder(t *testing.T) {
	t.Parallel()
	_, err := b.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = b.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{Duration: 86400})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = b.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{Symbol: "BTCUSDT", Side: order.Sell.String()})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{Symbol: "BTCUSDT", Side: order.Sell.String(), Quantity: 0.012})
	require.ErrorIs(t, err, errDurationRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SpotTWAPNewOrder(t.Context(), &SpotTWAPOrderParam{
		Symbol:   "BTCUSDT",
		Side:     order.Sell.String(),
		Quantity: 0.012,
		Duration: 86400,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSpotAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelSpotAlgoOrder(t.Context(), 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentSpotAlgoOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrentSpotAlgoOpenOrder(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotHistoricalAlgoOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotHistoricalAlgoOrders(t.Context(), "BNBUSDT", "BUY", time.Time{}, time.Time{}, 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotSubOrders(t.Context(), 1234, 0, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPortfolioMarginAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetClassicPortfolioMarginAccountInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPortfolioMarginCollateralRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetClassicPortfolioMarginCollateralRate(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPortfolioMarginBankruptacyLoanAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetClassicPortfolioMarginBankruptacyLoanAmount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayClassicPMBankruptacyLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RepayClassicPMBankruptacyLoan(t.Context(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPMNegativeBalanceInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetClassicPMNegativeBalanceInterestHistory(t.Context(), currency.ETH, time.Now().Add(-time.Hour*48*100), time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPMAssetIndexPrice(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPMAssetIndexPrice(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClassicPMFundAutoCollection(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ClassicPMFundAutoCollection(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClassicFundCollectionByAsset(t *testing.T) {
	t.Parallel()
	_, err := b.ClassicFundCollectionByAsset(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ClassicFundCollectionByAsset(t.Context(), currency.LTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAutoRepayFuturesStatusClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeAutoRepayFuturesStatusClassic(t.Context(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoRepayFuturesStatusClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAutoRepayFuturesStatusClassic(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayFuturesNegativeBalanceClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RepayFuturesNegativeBalanceClassic(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginAssetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPortfolioMarginAssetLeverage(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserNegativeBalanceAutoExchangeRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetUserNegativeBalanceAutoExchangeRecord(t.Context(), time.Time{}, time.Time{})
	require.ErrorIs(t, err, errStartAndEndTimeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserNegativeBalanceAutoExchangeRecord(t.Context(), time.Now().Add(-time.Hour*24), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBLVTInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBLVTInfo(t.Context(), "BTCDOWN")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeBLVT(t *testing.T) {
	t.Parallel()
	_, err := b.SubscribeBLVT(t.Context(), "", 0.011)
	require.ErrorIs(t, err, errNameRequired)
	_, err = b.SubscribeBLVT(t.Context(), "BTCUP", 0)
	require.ErrorIs(t, err, errCostRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubscribeBLVT(t.Context(), "BTCUP", 0.011)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSusbcriptionRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSusbcriptionRecords(t.Context(), "BTCDOWN", time.Time{}, time.Time{}, 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemBLVT(t *testing.T) {
	t.Parallel()
	_, err := b.RedeemBLVT(t.Context(), "", 2)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = b.RedeemBLVT(t.Context(), "BTCUSDT", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.RedeemBLVT(t.Context(), "BTCUSDT", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRedemptionRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetRedemptionRecord(t.Context(), "BTCDOWN", time.Time{}, time.Time{}, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBLVTUserLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBLVTUserLimitInfo(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatDepositAndWithdrawalHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetFiatDepositAndWithdrawalHistory(t.Context(), time.Time{}, time.Time{}, -5, 0, 50)
	require.ErrorIs(t, err, errInvalidTransactionType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFiatDepositAndWithdrawalHistory(t.Context(), time.Time{}, time.Time{}, 1, 0, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatPaymentHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetFiatPaymentHistory(t.Context(), time.Time{}, time.Time{}, -1, 0, 50)
	require.ErrorIs(t, err, errInvalidTransactionType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFiatPaymentHistory(t.Context(), time.Time{}, time.Time{}, 1, 0, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetC2CTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetC2CTradeHistory(t.Context(), "", time.Time{}, time.Time{}, 0, 50)
	require.ErrorIs(t, err, errTradeTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetC2CTradeHistory(t.Context(), order.Sell.String(), time.Time{}, time.Time{}, 0, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPLoanOngoingOrders(t.Context(), 1232, 21231, 0, 10, currency.BTC, currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanRepay(t *testing.T) {
	t.Parallel()
	_, err := b.VIPLoanRepay(t.Context(), 0, 0.2)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = b.VIPLoanRepay(t.Context(), 1234, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.VIPLoanRepay(t.Context(), 1234, 0.2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPayTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPayTradeHistory(t.Context(), time.Now().Add(-time.Hour*480), time.Now().Add(-time.Hour*24), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllConvertPairs(t *testing.T) {
	t.Parallel()
	_, err := b.GetAllConvertPairs(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := b.GetAllConvertPairs(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderQuantityPrecisionPerAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOrderQuantityPrecisionPerAsset(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSendQuoteRequest(t *testing.T) {
	t.Parallel()
	_, err := b.SendQuoteRequest(t.Context(), currency.EMPTYCODE, currency.USDT, 10, 20, "FUNDING", "1m")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.SendQuoteRequest(t.Context(), currency.BTC, currency.EMPTYCODE, 10, 20, "FUNDING", "1m")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.SendQuoteRequest(t.Context(), currency.BTC, currency.USDT, 0, 0, "FUNDING", "1m")
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SendQuoteRequest(t.Context(), currency.BTC, currency.USDT, 10, 20, "FUNDING", "1m")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcceptQuote(t *testing.T) {
	t.Parallel()
	_, err := b.AcceptQuote(t.Context(), "")
	require.ErrorIs(t, err, errQuoteIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.AcceptQuote(t.Context(), "933256278426274426")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertOrderStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetConvertOrderStatus(t.Context(), "933256278426274426", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceLimitOrder(t *testing.T) {
	t.Parallel()
	arg := &ConvertPlaceLimitOrderParam{}
	_, err := b.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.ExpiredType = "7_D"
	_, err = b.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.BaseAsset = currency.BTC
	arg.QuoteAsset = currency.ETH
	_, err = b.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.LimitPrice = 0.0122
	_, err = b.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	arg.ExpiredType = ""
	_, err = b.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, errExpiredTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.PlaceLimitOrder(t.Context(), &ConvertPlaceLimitOrderParam{
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
	_, err := b.CancelLimitOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelLimitOrder(t.Context(), "123434")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLimitOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLimitOpenOrders(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetConvertTradeHistory(t.Context(), time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotRebateHistoryRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotRebateHistoryRecords(t.Context(), time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetNFTTransactionHistory(t.Context(), -1, time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10, 40)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetNFTTransactionHistory(t.Context(), 1, time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetNFTDepositHistory(t.Context(), time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetNFTWithdrawalHistory(t.Context(), time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetNFTAsset(t.Context(), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSingleTokenGiftCard(t *testing.T) {
	t.Parallel()
	_, err := b.CreateSingleTokenGiftCard(t.Context(), currency.EMPTYCODE, 0.1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.CreateSingleTokenGiftCard(t.Context(), currency.BUSD, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CreateSingleTokenGiftCard(t.Context(), currency.BUSD, 0.1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateDualTokenGiftCard(t *testing.T) {
	t.Parallel()
	_, err := b.CreateDualTokenGiftCard(t.Context(), currency.EMPTYCODE, currency.BNB, 10, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.CreateDualTokenGiftCard(t.Context(), currency.BUSD, currency.EMPTYCODE, 10, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.CreateDualTokenGiftCard(t.Context(), currency.BUSD, currency.BNB, 0, 10)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.CreateDualTokenGiftCard(t.Context(), currency.BUSD, currency.BNB, 10, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CreateDualTokenGiftCard(t.Context(), currency.BUSD, currency.BNB, 10, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemBinanaceGiftCard(t *testing.T) {
	t.Parallel()
	_, err := b.RedeemBinanaceGiftCard(t.Context(), "", "12345")
	require.ErrorIs(t, err, errCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RedeemBinanaceGiftCard(t.Context(), "0033002328060227", "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVerifyBinanceGiftCardNumber(t *testing.T) {
	t.Parallel()
	_, err := b.VerifyBinanceGiftCardNumber(t.Context(), "")
	require.ErrorIs(t, err, errReferenceNumberRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.VerifyBinanceGiftCardNumber(t.Context(), "123456")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchRSAPublicKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FetchRSAPublicKey(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTokenLimit(t *testing.T) {
	t.Parallel()
	_, err := b.FetchTokenLimit(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FetchTokenLimit(t.Context(), currency.BUSD)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPLoanRepaymentHistory(t.Context(), currency.ETH, time.Now().Add(-time.Hour*48), time.Now(), 1234, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVIPLoanRenew(t *testing.T) {
	t.Parallel()
	_, err := b.VIPLoanRenew(t.Context(), 0, 60)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.VIPLoanRenew(t.Context(), 1234, 60)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckLockedValueVIPCollateralAccount(t *testing.T) {
	t.Parallel()
	_, err := b.CheckLockedValueVIPCollateralAccount(t.Context(), 0, 40)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = b.CheckLockedValueVIPCollateralAccount(t.Context(), 1223, 0)
	require.ErrorIs(t, err, errAccountIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CheckLockedValueVIPCollateralAccount(t.Context(), 1223, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVIPLoanBorrow(t *testing.T) {
	t.Parallel()
	_, err := b.VIPLoanBorrow(t.Context(), 0, 30, currency.ETH, currency.LTC, 123, "1234", false)
	require.ErrorIs(t, err, errAccountIDRequired)
	_, err = b.VIPLoanBorrow(t.Context(), 1234, 30, currency.EMPTYCODE, currency.LTC, 123, "1234", false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.VIPLoanBorrow(t.Context(), 1234, 30, currency.ETH, currency.LTC, 0, "1234", false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.VIPLoanBorrow(t.Context(), 1234, 30, currency.ETH, currency.LTC, 1.2, "", false)
	require.ErrorIs(t, err, errAccountIDRequired)
	_, err = b.VIPLoanBorrow(t.Context(), 1234, 30, currency.ETH, currency.EMPTYCODE, 123, "1234", false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.VIPLoanBorrow(t.Context(), 1234, 0, currency.ETH, currency.LTC, 123, "1234", false)
	require.ErrorIs(t, err, errLoanTermMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.VIPLoanBorrow(t.Context(), 1234, 30, currency.ETH, currency.LTC, 123, "1234", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanableAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPLoanableAssetsData(t.Context(), currency.BTC, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPCollateralAssetData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPCollateralAssetData(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPApplicationStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPApplicationStatus(t.Context(), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPBorrowInterestRate(t *testing.T) {
	t.Parallel()
	_, err := b.GetVIPBorrowInterestRate(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPBorrowInterestRate(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanAccruedInterest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPLoanAccruedInterest(t.Context(), "12345", currency.BTC, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanInterestRateHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetVIPLoanInterestRateHistory(t.Context(), currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPLoanInterestRateHistory(t.Context(), currency.BTC, time.Now().Add(-time.Hour*48), time.Now(), 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSpotListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CreateSpotListenKey(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKeepListenKeyAlive(t *testing.T) {
	t.Parallel()
	err := b.KeepSpotListenKeyAlive(t.Context(), "")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.KeepSpotListenKeyAlive(t.Context(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	require.NoError(t, err)
}

func TestCloseListenKey(t *testing.T) {
	t.Parallel()
	err := b.CloseSpotListenKey(t.Context(), "")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.CloseSpotListenKey(t.Context(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	require.NoError(t, err)
}

func TestCreateMarginListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CreateMarginListenKey(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKeepMarginListenKeyAlive(t *testing.T) {
	t.Parallel()
	err := b.KeepMarginListenKeyAlive(t.Context(), "")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err = b.KeepMarginListenKeyAlive(t.Context(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	assert.NoError(t, err)
}

func TestCloseMarginListenKey(t *testing.T) {
	t.Parallel()
	err := b.CloseMarginListenKey(t.Context(), "")
	require.ErrorIs(t, err, errListenKeyIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err = b.CloseMarginListenKey(t.Context(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	assert.NoError(t, err)
}

func TestCreateCrossMarginListenKey(t *testing.T) {
	t.Parallel()
	_, err := b.CreateCrossMarginListenKey(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CreateCrossMarginListenKey(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKeepCrossMarginListenKeyAlive(t *testing.T) {
	t.Parallel()
	err := b.KeepCrossMarginListenKeyAlive(t.Context(), "BTCUSDT", "")
	require.ErrorIs(t, err, errListenKeyIsRequired)
	err = b.KeepCrossMarginListenKeyAlive(t.Context(), "", "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err = b.KeepCrossMarginListenKeyAlive(t.Context(), "BTCUSDT", "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	assert.NoError(t, err)
}

func TestCloseCrossMarginListenKey(t *testing.T) {
	t.Parallel()
	err := b.CloseCrossMarginListenKey(t.Context(), "BTCUSDT", "")
	require.ErrorIs(t, err, errListenKeyIsRequired)
	err = b.CloseCrossMarginListenKey(t.Context(), "", "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.CloseCrossMarginListenKey(t.Context(), "BTCUSDT", "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
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

func (b *Binance) populateTradablePairs() error {
	err := b.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		return err
	}
	tradablePairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return fmt.Errorf("%w for %v", currency.ErrCurrencyPairsEmpty, asset.Spot)
	}
	spotTradablePair, err = b.FormatExchangeCurrency(tradablePairs[0], asset.Spot)
	if err != nil {
		return err
	}
	tradablePairs, err = b.GetEnabledPairs(asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		usdtmTradablePair = currency.NewPair(currency.BTC, currency.USDT)
	} else {
		usdtmTradablePair, err = b.FormatExchangeCurrency(tradablePairs[0], asset.USDTMarginedFutures)
		if err != nil {
			return err
		}
	}
	tradablePairs, err = b.GetEnabledPairs(asset.CoinMarginedFutures)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		coinmTradablePair, err = currency.NewPairFromString("ETHUSD_PERP")
		if err != nil {
			return err
		}
	} else {
		coinmTradablePair, err = b.FormatExchangeCurrency(tradablePairs[0], asset.CoinMarginedFutures)
		if err != nil {
			return err
		}
	}
	tradablePairs, err = b.GetEnabledPairs(asset.Options)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return fmt.Errorf("%w for %v", currency.ErrCurrencyPairsEmpty, asset.Options)
	}
	optionsTradablePair, err = b.FormatExchangeCurrency(tradablePairs[0], asset.Options)
	return err
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, b)
	for _, a := range b.GetAssetTypes(false) {
		pairs, err := b.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := b.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		require.NotEmpty(t, resp)
	}
}

func TestFetchOptionsExchangeLimits(t *testing.T) {
	t.Parallel()
	limits, err := b.FetchOptionsExchangeLimits(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, limits, "Should get some limits back")
}

func TestUnmarshalJSONOrderbookTranches(t *testing.T) {
	t.Parallel()
	data := `[[123.4, 321.0], ["123.6", "9"]]`
	var resp OrderbookTranches
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)
	require.Len(t, resp, 2)
	assert.Equal(t, 123.4, resp[0].Price)
	assert.Equal(t, 321.0, resp[0].Amount)
	assert.Equal(t, 123.6, resp[1].Price)
	assert.Equal(t, 9.0, resp[1].Amount)
}

// ----------------- Copy Trading endpoints unit-tests ----------------

func TestGetFuturesLeadTraderStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesLeadTraderStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesLeadTradingSymbolWhitelist(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesLeadTradingSymbolWhitelist(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.WithdrawalHistoryV1(t.Context(), []string{"1234"}, []string{"0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268"}, []string{}, "", "0", 0, 100, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawalHistoryV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.WithdrawalHistoryV2(t.Context(), []string{"1234"}, []string{"0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268"}, []string{}, "", "0", 0, 100, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitDepositQuestionnaire(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubmitDepositQuestionnaire(t.Context(), "765127651", map[string]interface{}{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLocalEntitiesDepositHistory(t.Context(), []string{}, []string{}, []string{}, "BNB", currency.USDT, "1", false, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOnboardedVASPList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOnboardedVASPList(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CreateSubAccount(t.Context(), "tag-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccounts(t.Context(), "1", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableFuturesForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.EnableFuturesForSubAccount(t.Context(), "", false)
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableFuturesForSubAccount(t.Context(), "1", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateAPIKeyForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.CreateAPIKeyForSubAccount(t.Context(), "", false, true, true)
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CreateAPIKeyForSubAccount(t.Context(), "1", false, true, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeSubAccountAPIPermission(t *testing.T) {
	t.Parallel()
	_, err := b.ChangeSubAccountAPIPermission(t.Context(), "", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", false, true, true)
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = b.ChangeSubAccountAPIPermission(t.Context(), "1", "", false, true, true)
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeSubAccountAPIPermission(t.Context(), "", "", false, true, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableUniversalTransferPermissionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := b.EnableUniversalTransferPermissionForSubAccountAPIKey(t.Context(), "", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", false)
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = b.EnableUniversalTransferPermissionForSubAccountAPIKey(t.Context(), "1", "", false)
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableUniversalTransferPermissionForSubAccountAPIKey(t.Context(), "1", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := b.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), "", "", "2", "")
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = b.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), "123", "", "2", "")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)
	_, err = b.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), "123", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", "", "")
	require.ErrorIs(t, err, errSubAccountStatusMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), "123", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", "2", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := b.DeleteIPRestrictionForSubAccountAPIKey(t.Context(), "", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", "")
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = b.DeleteIPRestrictionForSubAccountAPIKey(t.Context(), "123", "", "")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.DeleteIPRestrictionForSubAccountAPIKey(t.Context(), "123", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := b.DeleteSubAccountAPIKey(t.Context(), "", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A")
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = b.DeleteSubAccountAPIKey(t.Context(), "123", "")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.DeleteSubAccountAPIKey(t.Context(), "123", "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeSubAccountCommission(t *testing.T) {
	t.Parallel()
	_, err := b.ChangeSubAccountCommission(t.Context(), "", 1., 2., 0, 0)
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = b.ChangeSubAccountCommission(t.Context(), "2", 0, 2., 0, 0)
	require.ErrorIs(t, err, errCommissionValueRequired)
	_, err = b.ChangeSubAccountCommission(t.Context(), "2", 1., 0, 0, 0)
	require.ErrorIs(t, err, errCommissionValueRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeSubAccountCommission(t.Context(), "2", 1., 2., 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBNBBurnStatusForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := b.GetBNBBurnStatusForSubAccount(t.Context(), "")
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBNBBurnStatusForSubAccount(t.Context(), "1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransferWithSpotBroker(t *testing.T) {
	t.Parallel()
	_, err := b.SubAccountTransferWithSpotBroker(t.Context(), currency.EMPTYCODE, "", "", "", 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.SubAccountTransferWithSpotBroker(t.Context(), currency.BTC, "", "", "", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubAccountTransferWithSpotBroker(t.Context(), currency.BTC, "", "", "", 13)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotBrokerSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotBrokerSubAccountTransferHistory(t.Context(), "", "", "", true, time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransferWithFuturesBroker(t *testing.T) {
	t.Parallel()
	_, err := b.SubAccountTransferWithFuturesBroker(t.Context(), currency.EMPTYCODE, "", "", "", 1, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.SubAccountTransferWithFuturesBroker(t.Context(), currency.BTC, "", "", "", 2, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubAccountTransferWithFuturesBroker(t.Context(), currency.BTC, "", "", "", 1, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesBrokerSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesBrokerSubAccountTransferHistory(t.Context(), false, "", "", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositHistoryWithBroker(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountDepositHistoryWithBroker(t.Context(), "", currency.BTC, time.Time{}, time.Time{}, 0, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountSpotAssetInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountSpotAssetInfo(t.Context(), "1234", 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountMarginAssetInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountMarginAssetInfo(t.Context(), "", 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountFuturesAssetInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountFuturesAssetInfo(t.Context(), "1234", true, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUniversalTransferWithBroker(t *testing.T) {
	t.Parallel()
	_, err := b.UniversalTransferWithBroker(t.Context(), "", "USDT_FUTURE", "", "", "", currency.BTC, 1)
	require.ErrorIs(t, err, errInvalidAccountType)
	_, err = b.UniversalTransferWithBroker(t.Context(), "SPOT", "", "", "", "", currency.BTC, 1)
	require.ErrorIs(t, err, errInvalidAccountType)
	_, err = b.UniversalTransferWithBroker(t.Context(), "SPOT", "USDT_FUTURE", "", "", "", currency.EMPTYCODE, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = b.UniversalTransferWithBroker(t.Context(), "SPOT", "USDT_FUTURE", "", "", "", currency.BTC, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UniversalTransferWithBroker(t.Context(), "SPOT", "USDT_FUTURE", "", "", "", currency.BTC, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUniversalTransferHistoryThroughBroker(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUniversalTransferHistoryThroughBroker(t.Context(), "", "", "", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBrokerSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CreateBrokerSubAccount(t.Context(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBrokerSubAccounts(t.Context(), "123", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableOrDisableBNBBurnForSubAccountMarginInterest(t *testing.T) {
	t.Parallel()
	_, err := b.EnableOrDisableBNBBurnForSubAccountMarginInterest(t.Context(), "", false)
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableOrDisableBNBBurnForSubAccountMarginInterest(t.Context(), "3", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableOrDisableBNBBurnForSubAccountSpotAndMargin(t *testing.T) {
	t.Parallel()
	_, err := b.EnableOrDisableBNBBurnForSubAccountSpotAndMargin(t.Context(), "", true)
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableOrDisableBNBBurnForSubAccountSpotAndMargin(t.Context(), "1", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLinkAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.LinkAccountInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := b.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "", spotTradablePair.String(), 1, 10)
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = b.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "234", "", 1, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "234", spotTradablePair.String(), 0, 10)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "234", spotTradablePair.String(), 1, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "234", spotTradablePair.String(), 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountUSDMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := b.GetSubAccountUSDMarginedFuturesCommissionAdjustment(t.Context(), "", usdtmTradablePair.String())
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountUSDMarginedFuturesCommissionAdjustment(t.Context(), "123", usdtmTradablePair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := b.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "", coinmTradablePair.String(), 1., 2.)
	require.ErrorIs(t, err, errSubAccountIDMissing)
	_, err = b.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "231", "", 1., 2.)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = b.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "231", coinmTradablePair.String(), 0, 2.)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = b.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "231", coinmTradablePair.String(), 1., 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "231", coinmTradablePair.String(), 1., 2.)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountCoinMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := b.GetSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "", coinmTradablePair.String())
	require.ErrorIs(t, err, errSubAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "123", coinmTradablePair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerCommissionRebateRecentRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotBrokerCommissionRebateRecentRecord(t.Context(), "1234", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesBrokerCommissionRebateRecentRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesBrokerCommissionRebateRecentRecord(t.Context(), false, false, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ---------- Binance Link endpoints ----------------------------------

func TestGetInfoAboutIfUserIsNew(t *testing.T) {
	t.Parallel()
	_, err := b.GetSpotInfoAboutIfUserIsNew(t.Context(), "")
	require.ErrorIs(t, err, errCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotInfoAboutIfUserIsNew(t.Context(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCustomizeIDForClient(t *testing.T) {
	t.Parallel()
	_, err := b.CustomizeSpotPartnerClientID(t.Context(), "", "someone@thrasher.io")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = b.CustomizeSpotPartnerClientID(t.Context(), "1233", "")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CustomizeSpotPartnerClientID(t.Context(), "1233", "someone@thrasher.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClientEmailCustomizedID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotClientEmailCustomizedID(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesClientEmailCustomizedID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesClientEmailCustomizedID(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCustomizeOwnClientID(t *testing.T) {
	t.Parallel()
	_, err := b.CustomizeSpotOwnClientID(t.Context(), "", "ABCDEFG")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = b.CustomizeSpotOwnClientID(t.Context(), "the-unique-id", "")
	require.ErrorIs(t, err, errCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CustomizeSpotOwnClientID(t.Context(), "the-unique-id", "ABCDEFG")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCustomizeFuturesOwnClientID(t *testing.T) {
	t.Parallel()
	_, err := b.CustomizeFuturesOwnClientID(t.Context(), "", "ABCDEFG")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = b.CustomizeFuturesOwnClientID(t.Context(), "the-unique-id", "")
	require.ErrorIs(t, err, errCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CustomizeFuturesOwnClientID(t.Context(), "the-unique-id", "ABCDEFG")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersCustomizedID(t *testing.T) {
	t.Parallel()
	_, err := b.GetSpotUsersCustomizedID(t.Context(), "")
	require.ErrorIs(t, err, errCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotUsersCustomizedID(t.Context(), "1234ABCD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesUsersCustomizedID(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesUsersCustomizedID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesUsersCustomizedID(t.Context(), "1234ABCD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOthersRebateRecentRecord(t *testing.T) {
	t.Parallel()
	_, err := b.GetSpotOthersRebateRecentRecord(t.Context(), "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotOthersRebateRecentRecord(t.Context(), "123123", time.Now().Add(-time.Hour*24), time.Now(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOwnRebateRecentRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotOwnRebateRecentRecords(t.Context(), time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesClientIfNewUser(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesClientIfNewUser(t.Context(), "", 1)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesClientIfNewUser(t.Context(), "1234", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCustomizeFuturesPartnerClientID(t *testing.T) {
	t.Parallel()
	_, err := b.CustomizeFuturesPartnerClientID(t.Context(), "", "someone@thrasher.io")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = b.CustomizeFuturesPartnerClientID(t.Context(), "1233", "")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CustomizeFuturesPartnerClientID(t.Context(), "1233", "someone@thrasher.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesUserIncomeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesUserIncomeHistory(t.Context(), "BTCUSDT", "COMMISSION", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesReferredTradersNumber(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesReferredTradersNumber(t.Context(), true, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesRebateDataOverview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesRebateDataOverview(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTradeVolume(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserTradeVolume(t.Context(), true, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRebateVolume(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetRebateVolume(t.Context(), false, time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTraderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetTraderDetail(t.Context(), "sde001", true, time.Now().Add(-time.Hour*48), time.Now(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesClientifNewUser(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesClientifNewUser(t.Context(), "", false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesClientifNewUser(t.Context(), "123123", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCustomizeIDForClientToReferredUser(t *testing.T) {
	t.Parallel()
	_, err := b.CustomizeIDForClientToReferredUser(t.Context(), "", "1234")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = b.CustomizeIDForClientToReferredUser(t.Context(), "1234", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CustomizeIDForClientToReferredUser(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersCustomizeIDs(t *testing.T) {
	t.Parallel()
	_, err := b.GetUsersCustomizeIDs(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUsersCustomizeIDs(t.Context(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFastAPIUserStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	_, err := b.CreateAPIKey(t.Context(), "", "12312", "1", "", "", true, true, false, true)
	require.ErrorIs(t, err, errAPIKeyNameRequired)
	_, err = b.CreateAPIKey(t.Context(), "12345", "", "1", "", "", true, true, false, true)
	require.ErrorIs(t, err, errEmptySubAccountAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CreateAPIKey(t.Context(), "", "", "1", "", "", true, true, false, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

var orderTypeFromStringList = []struct {
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

func TestOrderTypeFromString(t *testing.T) {
	t.Parallel()
	for _, val := range orderTypeFromStringList {
		result, err := StringToOrderType(val.String)
		require.ErrorIs(t, err, val.Error)
		assert.Equal(t, result, val.OrderType)
	}
}

var orderTypeStringToTypeList = []struct {
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

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	for _, value := range orderTypeStringToTypeList {
		result, err := OrderTypeString(value.OrderType)
		require.ErrorIs(t, err, value.Error)
		assert.Equal(t, result, value.String)
	}
}

var timeInForceStringList = []struct {
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

func TestTimeInForceString(t *testing.T) {
	t.Parallel()
	for _, val := range timeInForceStringList {
		result := timeInForceString(val.TIF, val.OType)
		assert.Equal(t, val.String, result)
	}
}
