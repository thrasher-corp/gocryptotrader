package binance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
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

	// this pair is used to ensure that endpoints match it correctly
	testPairMapping = currency.NewPair(currency.DOGE, currency.USDT)
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
	result, err := b.UServerTime(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetServerTime(context.Background(), asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	st, err := b.GetServerTime(context.Background(), asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, st)

	st, err = b.GetServerTime(context.Background(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, st)

	st, err = b.GetServerTime(context.Background(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, st)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	for assetType, pair := range assetToTradablePairMap {
		r, err := b.UpdateTicker(context.Background(), pair, assetType)
		require.NoErrorf(t, err, "expected nil, got %v for asset type: %s pair: %v", err, assetType, pair)
		require.NotNilf(t, r, "unexpected value nil for asset type: %s pair: %v", assetType, pair)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	enabledAssets := b.GetAssetTypes(true)
	for _, assetType := range enabledAssets {
		err := b.UpdateTickers(context.Background(), assetType)
		require.NoError(t, err)
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
		result, err := b.UpdateOrderbook(context.Background(), tp, assetType)
		require.NoError(t, err)
		require.NotNil(t, result)
	}
}

// USDT Margined Futures

func TestUExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := b.UExchangeInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesOrderbook(t *testing.T) {
	t.Parallel()
	result, err := b.UFuturesOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.USDT), 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestURecentTrades(t *testing.T) {
	t.Parallel()
	result, err := b.URecentTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "", 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCompressedTrades(t *testing.T) {
	t.Parallel()
	result, err := b.UCompressedTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "", 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	start, end := getTime()
	result, err = b.UCompressedTrades(context.Background(), currency.NewPair(currency.LTC, currency.USDT), "", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUKlineData(t *testing.T) {
	t.Parallel()
	result, err := b.UKlineData(context.Background(), usdtmTradablePair, "1d", 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	start, end := getTime()
	result, err = b.UKlineData(context.Background(), usdtmTradablePair, "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUFuturesContinuousKlineData(t *testing.T) {
	t.Parallel()
	result, err := b.GetUFuturesContinuousKlineData(context.Background(), usdtmTradablePair, "CURRENT_QUARTER", "1d", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexOrCandlesticPriceKlineData(t *testing.T) {
	t.Parallel()
	result, err := b.GetIndexOrCandlesticPriceKlineData(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "1d", time.Time{}, time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPriceKlineCandlesticks(t *testing.T) {
	t.Parallel()
	result, err := b.GetMarkPriceKlineCandlesticks(context.Background(), "BTCUSDT", "1d", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPremiumIndexKlineCandlesticks(t *testing.T) {
	t.Parallel()
	result, err := b.GetPremiumIndexKlineCandlesticks(context.Background(), "BTCUSDT", "1d", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := b.UGetMarkPrice(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.UGetMarkPrice(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetFundingHistory(t *testing.T) {
	t.Parallel()
	result, err := b.UGetFundingHistory(context.Background(), currency.NewPair(currency.BTC, currency.USDT), 1000, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.UGetFundingHistory(context.Background(), currency.NewPair(currency.LTC, currency.USDT), 1000, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestU24HTickerPriceChangeStats(t *testing.T) {
	t.Parallel()
	result, err := b.U24HTickerPriceChangeStats(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.U24HTickerPriceChangeStats(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	result, err := b.USymbolPriceTickerV1(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.USymbolPriceTickerV1(context.Background(), currency.EMPTYPAIR)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUSymbolPriceTickerV2(t *testing.T) {
	t.Parallel()
	result, err := b.USymbolPriceTickerV2(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.USymbolPriceTickerV2(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	result, err := b.USymbolOrderbookTicker(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.USymbolOrderbookTicker(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUOpenInterest(t *testing.T) {
	t.Parallel()
	result, err := b.UOpenInterest(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuarterlyContractSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetQuarterlyContractSettlementPrice(context.Background(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := b.GetQuarterlyContractSettlementPrice(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUOpenInterestStats(t *testing.T) {
	t.Parallel()
	result, err := b.UOpenInterestStats(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "5m", 1, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	start, end := getTime()
	result, err = b.UOpenInterestStats(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "1d", 10, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUTopAcccountsLongShortRatio(t *testing.T) {
	t.Parallel()
	result, err := b.UTopAcccountsLongShortRatio(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "5m", 2, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	start, end := getTime()
	result, err = b.UTopAcccountsLongShortRatio(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "5m", 2, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUTopPostionsLongShortRatio(t *testing.T) {
	t.Parallel()
	result, err := b.UTopPostionsLongShortRatio(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "5m", 3, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	start, end := getTime()
	result, err = b.UTopPostionsLongShortRatio(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "1d", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGlobalLongShortRatio(t *testing.T) {
	t.Parallel()
	result, err := b.UGlobalLongShortRatio(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "5m", 3, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	start, end := getTime()
	result, err = b.UGlobalLongShortRatio(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "4h", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUTakerBuySellVol(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	result, err := b.UTakerBuySellVol(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "5m", 10, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBasis(t *testing.T) {
	t.Parallel()
	result, err := b.GetBasis(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "CURRENT_QUARTER", "15m", time.Time{}, time.Time{}, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetBasis(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "NEXT_QUARTER", "15m", time.Time{}, time.Time{}, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetBasis(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "PERPETUAL", "15m", time.Time{}, time.Time{}, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalBLVTNAVCandlesticks(t *testing.T) {
	t.Parallel()
	result, err := b.GetHistoricalBLVTNAVCandlesticks(context.Background(), "BTCDOWN", "15m", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCompositeIndexInfo(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("DEFI-USDT")
	require.NoError(t, err)
	result, err := b.UCompositeIndexInfo(context.Background(), cp)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.UCompositeIndexInfo(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMultiAssetModeAssetIndex(t *testing.T) {
	t.Parallel()
	result, err := b.GetMultiAssetModeAssetIndex(context.Background(), "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetMultiAssetModeAssetIndex(context.Background(), "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceConstituents(t *testing.T) {
	t.Parallel()
	result, err := b.GetIndexPriceConstituents(context.Background(), "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UFuturesNewOrder(context.Background(),
		&UFuturesNewOrderRequest{
			Symbol:      currency.NewPair(currency.BTC, currency.USDT),
			Side:        "BUY",
			OrderType:   "LIMIT",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UModifyOrder(context.Background(), &USDTOrderUpdateParams{
		OrderID:           1,
		OrigClientOrderID: "",
		Side:              "SELL",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	var data []PlaceBatchOrderData
	var tempData PlaceBatchOrderData
	tempData.Symbol = "BTCUSDT"
	tempData.Side = "BUY"
	tempData.OrderType = "LIMIT"
	tempData.Quantity = 4
	tempData.Price = 1
	tempData.TimeInForce = "GTC"
	data = append(data, tempData)
	result, err := b.UPlaceBatchOrders(context.Background(), data)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UModifyMultipleOrders(context.Background(), []USDTOrderUpdateParams{
		{
			OrderID:           1,
			OrigClientOrderID: "",
			Side:              "SELL",
			PriceMatch:        "TAKE_PROFIT",
			Symbol:            currency.NewPair(currency.BTC, currency.USD),
			Amount:            0.0000001,
			Price:             123455554,
		},
		{
			OrderID:           1,
			OrigClientOrderID: "",
			Side:              "BUY",
			PriceMatch:        "LIMIT",
			Symbol:            currency.NewPair(currency.BTC, currency.USD),
			Amount:            0.0000001,
			Price:             123455554,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUSDTOrderModifyHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUSDTOrderModifyHistory(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "", 1234, 10, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetOrderData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UGetOrderData(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UCancelOrder(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UCancelAllOpenOrders(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UCancelBatchOrders(context.Background(), currency.NewPair(currency.BTC, currency.USDT), []string{"123"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UAutoCancelAllOpenOrders(context.Background(), currency.NewPair(currency.BTC, currency.USDT), 30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFetchOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UFetchOpenOrder(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAllAccountOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAllAccountOpenOrders(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAllAccountOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAllAccountOrders(context.Background(), currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.UAllAccountOrders(context.Background(), currency.NewPair(currency.BTC, currency.USDT), 0, 5, time.Now().Add(-time.Hour*4), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountBalanceV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAccountBalanceV2(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountInformationV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAccountInformationV2(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUChangeInitialLeverageRequest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UChangeInitialLeverageRequest(context.Background(), currency.NewPair(currency.BTC, currency.USDT), 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUChangeInitialMarginType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.UChangeInitialMarginType(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "ISOLATED")
	assert.NoError(t, err)
}

func TestUModifyIsolatedPositionMarginReq(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UModifyIsolatedPositionMarginReq(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "LONG", "add", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPositionMarginChangeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UPositionMarginChangeHistory(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "add", 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPositionsInfoV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UPositionsInfoV2(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetCommissionRates(t *testing.T) {
	t.Parallel()
	_, err := b.UGetCommissionRates(context.Background(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UGetCommissionRates(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUSDTUserRateLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUSDTUserRateLimits(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDownloadIDForFuturesTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDownloadIDForFuturesTransactionHistory(context.Background(), time.Now().Add(-time.Hour*24*6), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTransactionHistoryDownloadLinkByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesTransactionHistoryDownloadLinkByID(context.Background(), "download-id-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderHistoryDownloadLinkByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesOrderHistoryDownloadLinkByID(context.Background(), "download-id-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTradeDownloadLinkByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesTradeDownloadLinkByID(context.Background(), "download-id-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesOrderHistoryDownloadID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UFuturesOrderHistoryDownloadID(context.Background(), time.Now().Add(-time.Hour*24*6), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTradeHistoryDownloadID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesTradeHistoryDownloadID(context.Background(), time.Now().Add(-time.Hour*24*6), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountTradesHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAccountTradesHistory(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "", 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountIncomeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAccountIncomeHistory(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "", 5, time.Now().Add(-time.Hour*48), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UGetNotionalAndLeverageBrackets(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UPositionsADLEstimate(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUAccountForcedOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UAccountForcedOrders(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "ADL", 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUFuturesTradingWuantitativeRulesIndicators(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.UFuturesTradingWuantitativeRulesIndicators(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// Coin Margined Futures

func TestGetFuturesExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := b.FuturesExchangeInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesOrderbook(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPublicTrades(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesPublicTrades(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPastPublicTrades(t *testing.T) {
	t.Parallel()
	result, err := b.GetPastPublicTrades(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTradesList(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesAggregatedTradesList(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 0, 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPerpsExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := b.GetPerpMarkets(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexAndMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := b.GetIndexAndMarkPrice(context.Background(), "", "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesKlineData(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesKlineData(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetFuturesKlineData(context.Background(), currency.NewPairWithDelimiter("LTCUSD", "PERP", "_"), "5m", 5, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContinuousKlineData(t *testing.T) {
	t.Parallel()
	result, err := b.GetContinuousKlineData(context.Background(), "BTCUSD", "CURRENT_QUARTER", "1M", 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetContinuousKlineData(context.Background(), "BTCUSD", "CURRENT_QUARTER", "1M", 5, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceKlines(t *testing.T) {
	t.Parallel()
	result, err := b.GetIndexPriceKlines(context.Background(), "BTCUSD", "1M", 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetIndexPriceKlines(context.Background(), "BTCUSD", "1M", 5, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesSwapTickerChangeStats(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesSwapTickerChangeStats(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetFuturesSwapTickerChangeStats(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetFuturesSwapTickerChangeStats(context.Background(), currency.EMPTYPAIR, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesGetFundingHistory(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.FuturesGetFundingHistory(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 50, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesHistoricalTrades(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 5)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetFuturesHistoricalTrades(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesSymbolPriceTicker(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderbookTicker(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesOrderbookTicker(context.Background(), currency.EMPTYPAIR, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetFuturesOrderbookTicker(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCFuturesIndexPriceConstituents(t *testing.T) {
	t.Parallel()
	_, err := b.GetCFuturesIndexPriceConstituents(context.Background(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.GetCFuturesIndexPriceConstituents(context.Background(), currency.NewPair(currency.BTC, currency.USD))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOpenInterest(t *testing.T) {
	t.Parallel()
	result, err := b.OpenInterest(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCFuturesQuarterlyContractSettlementPrice(t *testing.T) {
	t.Parallel()
	result, err := b.CFuturesQuarterlyContractSettlementPrice(context.Background(), currency.NewPair(currency.BTC, currency.USD))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterestStats(t *testing.T) {
	t.Parallel()
	result, err := b.GetOpenInterestStats(context.Background(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetOpenInterestStats(context.Background(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTraderFuturesAccountRatio(t *testing.T) {
	t.Parallel()
	result, err := b.GetTraderFuturesAccountRatio(context.Background(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetTraderFuturesAccountRatio(context.Background(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTraderFuturesPositionsRatio(t *testing.T) {
	t.Parallel()
	result, err := b.GetTraderFuturesPositionsRatio(context.Background(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetTraderFuturesPositionsRatio(context.Background(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketRatio(t *testing.T) {
	t.Parallel()
	result, err := b.GetMarketRatio(context.Background(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetMarketRatio(context.Background(), currency.NewPair(currency.BTC, currency.USD), "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTakerVolume(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesTakerVolume(context.Background(), currency.NewPair(currency.BTC, currency.USD), "ALL", "5m", 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start, end := getTime()
	result, err = b.GetFuturesTakerVolume(context.Background(), currency.NewPair(currency.BTC, currency.USD), "ALL", "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesBasisData(t *testing.T) {
	t.Parallel()
	result, err := b.GetFuturesBasisData(context.Background(), currency.NewPair(currency.BTC, currency.USD), "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	start := time.UnixMilli(1577836800000)
	end := time.UnixMilli(1580515200000)
	if !mockTests {
		start = time.Now().Add(-time.Second * 240)
		end = time.Now()
	}
	result, err = b.GetFuturesBasisData(context.Background(), currency.NewPair(currency.BTC, currency.USD), "CURRENT_QUARTER", "5m", 0, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesNewOrder(
		context.Background(),
		&FuturesNewOrderRequest{
			Symbol:      currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
			Side:        "BUY",
			OrderType:   "LIMIT",
			TimeInForce: BinanceRequestParamsTimeGTC,
			Quantity:    1,
			Price:       1,
		},
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesBatchOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	var data []PlaceBatchOrderData
	var tempData PlaceBatchOrderData
	tempData.Symbol = "BTCUSD_PERP"
	tempData.Side = "BUY"
	tempData.OrderType = "LIMIT"
	tempData.Quantity = 1
	tempData.Price = 1
	tempData.TimeInForce = "GTC"

	data = append(data, tempData)
	result, err := b.FuturesBatchOrder(context.Background(), data)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesBatchCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesBatchCancelOrders(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), []string{"123"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesGetOrderData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesGetOrderData(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesCancelAllOpenOrders(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.AutoCancelAllOpenOrders(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 30000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesOpenOrderData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesOpenOrderData(context.Background(), currency.NewPair(currency.BTC, currency.USD), "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesAllOpenOrders(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllFuturesOrders(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesChangeMarginType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesChangeMarginType(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "ISOLATED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesAccountBalance(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesAccountInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesChangeInitialLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesChangeInitialLeverage(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyIsolatedPositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ModifyIsolatedPositionMargin(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "BOTH", "add", 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesMarginChangeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesMarginChangeHistory(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "add", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesPositionsInfo(context.Background(), "BTCUSD", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesTradeHistory(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", time.Time{}, time.Time{}, 5, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesIncomeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesIncomeHistory(context.Background(), currency.EMPTYPAIR, "TRANSFER", time.Time{}, time.Time{}, 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesForceOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesForceOrders(context.Background(), currency.EMPTYPAIR, "ADL", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetNotionalLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesNotionalBracket(context.Background(), "BTCUSD")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.FuturesNotionalBracket(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FuturesPositionsADLEstimate(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	result, err := b.GetMarkPriceKline(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	result, err := b.GetPremiumIndexKlineData(context.Background(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := b.GetExchangeInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	results, err := b.FetchTradablePairs(context.Background(), asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	results, err = b.FetchTradablePairs(context.Background(), asset.CoinMarginedFutures)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	results, err = b.FetchTradablePairs(context.Background(), asset.USDTMarginedFutures)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	results, err = b.FetchTradablePairs(context.Background(), asset.Options)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	result, err := b.GetOrderBook(context.Background(),
		OrderBookDataRequestParams{
			Symbol: currency.NewPair(currency.BTC, currency.USDT),
			Limit:  1000,
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMostRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetMostRecentTrades(context.Background(), &RecentTradeRequestParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	result, err := b.GetMostRecentTrades(context.Background(), &RecentTradeRequestParams{
		Symbol: currency.NewPair(currency.BTC, currency.USDT),
		Limit:  15})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetHistoricalTrades(context.Background(), "", 5, -1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.GetHistoricalTrades(context.Background(), "BTCUSDT", 5, -1)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetAggregatedTrades(context.Background(), &AggregatedTradeRequestParams{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := b.GetAggregatedTrades(context.Background(),
		&AggregatedTradeRequestParams{
			Symbol: "BTCUSDT", Limit: 5})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	result, err := b.GetSpotKline(context.Background(),
		&KlinesRequestParams{
			Symbol:   currency.NewPair(currency.BTC, currency.USDT),
			Interval: kline.FiveMin.Short(), Limit: 24,
			StartTime: start, EndTime: end})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUIKline(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	result, err := b.GetUIKline(context.Background(),
		&KlinesRequestParams{
			Symbol:   currency.NewPair(currency.BTC, currency.USDT),
			Interval: kline.FiveMin.Short(), Limit: 24,
			StartTime: start, EndTime: end})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()
	result, err := b.GetAveragePrice(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()
	result, err := b.GetPriceChangeStats(context.Background(), currency.NewPair(currency.BTC, currency.USDT), currency.Pairs{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradingDayTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTradingDayTicker(context.Background(), []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)}, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)
	_, err = b.GetTradingDayTicker(context.Background(), []currency.Pair{currency.EMPTYPAIR}, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := b.GetTradingDayTicker(context.Background(), []currency.Pair{currency.EMPTYPAIR}, "", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	result, err := b.GetLatestSpotPrice(context.Background(), currency.NewPair(currency.BTC, currency.USDT), currency.Pairs{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()
	result, err := b.GetBestPrice(context.Background(), currency.NewPair(currency.BTC, currency.USDT), currency.Pairs{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickerData(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickerData(context.Background(), []currency.Pair{}, time.Minute*20, "FULL")
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	result, err := b.GetTickerData(context.Background(), []currency.Pair{{Base: currency.BTC, Quote: currency.USDT}}, time.Minute*20, "FULL")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDepositAddressForCurrency(context.Background(), "BTC", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetsThatCanBeConvertedIntoBNB(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAssetsThatCanBeConvertedIntoBNB(context.Background(), "MINI")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDustTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.DustTransfer(context.Background(), []string{"BTC", "USDT"}, "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetDevidendRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAssetDevidendRecords(context.Background(), currency.BTC, time.Now().Add(-time.Hour*48), time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAssetDetail(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeFees(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetTradeFees(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUserUniversalTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UserUniversalTransfer(context.Background(), ttMainUMFuture, 123.234, currency.BTC, "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserUniversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetUserUniversalTransferHistory(context.Background(), ttUMFutureMargin, time.Time{}, time.Time{}, 0, 0, "BTC", "USDT")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserUniversalTransferHistory(context.Background(), ttUMFutureMargin, time.Time{}, time.Time{}, 1, 1234, "BTC", "USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFundingAssets(context.Background(), currency.BTC, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserAssets(context.Background(), currency.BTC, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConvertBUSD(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ConvertBUSD(context.Background(), "12321412312", "MAIN", currency.ETH, currency.USD, 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBUSDConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.BUSDConvertHistory(context.Background(), "transaction-id", "233423423", "CARD", currency.BTC, time.Now().Add(-time.Hour*48*10), time.Now(), 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCloudMiningPaymentAndRefundHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCloudMiningPaymentAndRefundHistory(context.Background(), "1234", currency.BTC, time.Now().Add(-time.Hour*480), time.Now(), 1232313, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAPIKeyPermission(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAPIKeyPermission(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoConvertingStableCoins(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAutoConvertingStableCoins(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSwitchOnOffBUSDAndStableCoinsConversion(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SwitchOnOffBUSDAndStableCoinsConversion(context.Background(), currency.BTC, false)
	assert.NoError(t, err)
}

func TestOneClickArrivalDepositApply(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.OneClickArrivalDepositApply(context.Background(), "", 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressListWithNetwork(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDepositAddressListWithNetwork(context.Background(), currency.BTC, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserWalletBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserWalletBalance(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserDelegationHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserDelegationHistory(context.Background(), "someone@thrasher.com", "Delegate", time.Now().Add(-time.Hour*24*12), time.Now(), currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolsDelistScheduleForSpot(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSymbolsDelistScheduleForSpot(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateVirtualSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CreateVirtualSubAccount(context.Background(), "something-string")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountList(context.Background(), "testsub@gmail.com", false, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountSpotAssetTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountSpotAssetTransferHistory(context.Background(), "", "", time.Time{}, time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountFuturesAssetTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountFuturesAssetTransferHistory(context.Background(), "someone@gmail.com", time.Time{}, time.Now(), 2, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountFuturesAssetTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubAccountFuturesAssetTransfer(context.Background(), "from_someone@thrasher.io", "to_someont@thrasher.io", 1, currency.USDT, 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountAssets(context.Background(), "email_address@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountList(context.Background(), "address@gmail.com", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountTransactionStatistics(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountTransactionStatistics(context.Background(), "address@thrasher.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountDepositAddress(context.Background(), currency.ETH, "destination@thrasher.io", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableOptionsForSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableOptionsForSubAccount(context.Background(), "address@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountTransferLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountTransferLog(context.Background(), time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*30), 1, 10, "", "MARGIN")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountSpotAssetsSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountSpotAssetsSummary(context.Background(), "the_address@thrasher.io", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountDepositAddress(context.Background(), "the_address@thrasher.io", "BTC", "", 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountDepositHistory(context.Background(), "someone@thrasher.io", "BTC", time.Time{}, time.Now(), 0, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountStatusOnMarginFutures(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubAccountStatusOnMarginFutures(context.Background(), "myemail@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableMarginForSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableMarginForSubAccount(context.Background(), "sampleemail@email.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailOnSubAccountMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := b.GetDetailOnSubAccountMarginAccount(context.Background(), "com")
	require.ErrorIs(t, err, errValidEmailRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDetailOnSubAccountMarginAccount(context.Background(), "test@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfSubAccountMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSummaryOfSubAccountMarginAccount(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableFuturesSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableFuturesSubAccount(context.Background(), "address@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailSubAccountFuturesAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDetailSubAccountFuturesAccount(context.Background(), "address@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfSubAccountFuturesAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetSummaryOfSubAccountFuturesAccount(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV1FuturesPositionRiskSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetV1FuturesPositionRiskSubAccount(context.Background(), "address@mail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestGetFuturesPositionRiskSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetV2FuturesPositionRiskSubAccount(context.Background(), "address@mail.com", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableLeverageTokenForSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.EnableLeverageTokenForSubAccount(context.Background(), "someone@thrasher.io", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIPRestrictionForSubAccountAPIKeyV2(context.Background(), "emailaddress@thrasher.io", apiKey)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteIPListForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.DeleteIPListForSubAccountAPIKey(context.Background(), "emailaddress@thrasher.io", apiKey, "196.168.4.1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAddIPRestrictionForSubAccountAPIkey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.AddIPRestrictionForSubAccountAPIkey(context.Background(), "address@thrasher.io", apiKey, "", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDepositAssetsIntoTheManagedSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.DepositAssetsIntoTheManagedSubAccount(context.Background(), "toemail@mail.com", currency.BTC, 0.0001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountAssetsDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetManagedSubAccountAssetsDetails(context.Background(), "emailaddress@thrashser.io")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawAssetsFromManagedSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.WithdrawAssetsFromManagedSubAccount(context.Background(), "source@email.com", currency.BTC, 0.0000001, time.Now().Add(-time.Hour*24*50))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountSnapshot(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountSnapshot(context.Background(), "address@thrasher.io", "SPOT", time.Time{}, time.Now(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountTransferLogForInvestorMasterAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountTransferLogForInvestorMasterAccount(context.Background(), "address@gmail.com", "TO", "SPOT", time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*20), 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountTransferLogForTradingTeam(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountTransferLogForTradingTeam(context.Background(), "address@gmail.com", "FROM", "ISOLATED_MARGIN", time.Now().Add(-time.Hour*24*50), time.Now().Add(-time.Hour*24*20), 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountFutureesAssetDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountFutureesAssetDetails(context.Background(), "address@email.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetManagedSubAccountMarginAssetDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetManagedSubAccountMarginAssetDetails(context.Background(), "address@gmail.com")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesTransferSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesTransferSubAccount(context.Background(), "someone@mail.com", currency.BTC, 1.1, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginTransferForSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.MarginTransferForSubAccount(context.Background(), "someone@mail.com", currency.BTC, 1.1, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferToSubAccountOfSameMaster(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.TransferToSubAccountOfSameMaster(context.Background(), "toEmail@thrasher.io", currency.ETH, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFromSubAccountTransferToMaster(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FromSubAccountTransferToMaster(context.Background(), currency.LTC, 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubAccountTransferHistory(context.Background(), currency.BTC, 1, 10, time.Time{}, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransferHistoryForSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubAccountTransferHistoryForSubAccount(context.Background(), currency.LTC, 2, 0, time.Time{}, time.Now(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUniversalTransferForMasterAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.UniversalTransferForMasterAccount(context.Background(), &UniversalTransferParams{
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
	result, err := b.GetUniversalTransferHistoryForMasterAccount(context.Background(), "", "", "", time.Time{}, time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailOnSubAccountsFuturesAccountV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDetailOnSubAccountsFuturesAccountV2(context.Background(), "address@thrasher.io", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfSubAccountsFuturesAccountV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSummaryOfSubAccountsFuturesAccountV2(context.Background(), 1, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.QueryOrder(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "", 1337)
	require.False(t, sharedtestvalues.AreAPICredentialsSet(b) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests, "expecting an error when no keys are set")
	assert.False(t, mockTests && err != nil, err)
}

func TestCancelExistingOrderAndSendNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelExistingOrderAndSendNewOrder(context.Background(), &CancelReplaceOrderParams{
		Symbol:            "BTCUSDT",
		Side:              "BUY",
		OrderType:         "LIMIT",
		CancelReplaceMode: "STOP_ON_FAILURE",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.OpenOrders(context.Background(), currency.EMPTYPAIR)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	p := currency.NewPair(currency.BTC, currency.USDT)
	result, err = b.OpenOrders(context.Background(), p)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrderOnSymbol(t *testing.T) {
	t.Parallel()
	_, err := b.CancelAllOpenOrderOnSymbol(context.Background(), "BTCUSDT")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllOpenOrderOnSymbol(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.AllOrders(context.Background(), currency.NewPair(currency.BTC, currency.USDT), "", "")
	require.False(t, sharedtestvalues.AreAPICredentialsSet(b) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests, "expecting an error when no keys are set")
	assert.False(t, mockTests && err != nil, err)
}

func TestNewOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := b.NewOCOOrder(context.Background(), &OCOOrderParam{})
	require.ErrorIs(t, err, errNilArgument)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewOCOOrder(context.Background(), &OCOOrderParam{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelOCOOrder(context.Background(), "LTCBTC", "", "newderID", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOCOOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOCOOrders(context.Background(), "123456", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOCOOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllOCOOrders(context.Background(), "", time.Time{}, time.Now().Add(-time.Hour*24*10), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOCOList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOpenOCOList(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOrderUsingSOR(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewOrderUsingSOR(context.Background(), &SOROrderRequestParams{
		Symbol:    currency.Pair{Base: currency.BTC, Quote: currency.LTC},
		Side:      "Buy",
		OrderType: "LIMIT",
		Quantity:  0.001,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOrderUsingSORTest(t *testing.T) {
	t.Parallel()
	_, err := b.NewOrderUsingSORTest(context.Background(), &SOROrderRequestParams{})
	require.ErrorIs(t, err, errNilArgument)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewOrderUsingSORTest(context.Background(), &SOROrderRequestParams{
		Symbol:    currency.Pair{Base: currency.BTC, Quote: currency.LTC},
		Side:      "Buy",
		OrderType: "LIMIT",
		Quantity:  0.001,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	result, err := b.GetFeeByType(context.Background(), feeBuilder)
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
	var feeBuilder = setFeeBuilder()
	if sharedtestvalues.AreAPICredentialsSet(b) && mockTests {
		// CryptocurrencyTradeFee Basic
		_, err := b.GetFee(context.Background(), feeBuilder)
		require.NoError(t, err)

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		_, err = b.GetFee(context.Background(), feeBuilder)
		require.NoError(t, err)

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		_, err = b.GetFee(context.Background(), feeBuilder)
		require.NoError(t, err)

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		_, err = b.GetFee(context.Background(), feeBuilder)
		require.NoError(t, err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err := b.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = b.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = b.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = b.GetFee(context.Background(), feeBuilder)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     currency.Pairs{pair},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	result, err := b.GetActiveOrders(context.Background(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := b.GetOrderHistory(context.Background(), &getOrdersRequest)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	getOrdersRequest.Pairs = []currency.Pair{
		currency.NewPair(currency.LTC,
			currency.BTC)}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOrderHistory(context.Background(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOrderTest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.NewOrderTest(context.Background(), &NewOrderRequest{
		Symbol:      currency.NewPair(currency.LTC, currency.BTC),
		Side:        order.Buy.String(),
		TradeType:   BinanceRequestParamsOrderLimit,
		Price:       0.0025,
		Quantity:    100000,
		TimeInForce: BinanceRequestParamsTimeGTC,
	}, false)
	require.NoError(t, err)

	err = b.NewOrderTest(context.Background(), &NewOrderRequest{
		Symbol:        currency.NewPair(currency.LTC, currency.BTC),
		Side:          order.Sell.String(),
		TradeType:     BinanceRequestParamsOrderMarket,
		Price:         0.0045,
		QuoteOrderQty: 10,
	}, true)
	assert.NoError(t, err)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	p := currency.NewPair(currency.BTC, currency.USDT)
	start := time.Unix(1577977445, 0)  // 2020-01-02 15:04:05
	end := start.Add(15 * time.Minute) // 2020-01-02 15:19:05
	if b.IsAPIStreamConnected() {
		start = time.Now().Add(-time.Hour * 10)
		end = time.Now().Add(-time.Hour)
	}
	result, err := b.GetHistoricTrades(context.Background(), p, asset.Spot, start, end)
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
	result, err = b.GetHistoricTrades(context.Background(), optionsTradablePair, asset.Options, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTradesBatched(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	start, err := time.Parse(time.RFC3339, "2020-01-02T15:04:05Z")
	if err != nil {
		t.Fatal(err)
	}
	expectTime, err := time.Parse(time.RFC3339Nano, "2020-01-02T16:19:04.831Z")
	if err != nil {
		t.Fatal(err)
	}
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
			result, err := b.GetAggregatedTrades(context.Background(), tt.args)
			if err != nil {
				t.Error(err)
			}
			if len(result) != tt.numExpected {
				t.Errorf("GetAggregatedTradesBatched() expected %v entries, got %v", tt.numExpected, len(result))
			}
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
			_, err := b.GetAggregatedTrades(context.Background(), tt.args)
			require.Error(t, err)
		})
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// -----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubmitOrder(context.Background(), &order.Submit{
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
	err := b.CancelOrder(context.Background(), &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	})
	assert.NoError(t, err)
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllOrders(context.Background(), &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          spotTradablePair,
		AssetType:     asset.Spot,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.CancelAllOrders(context.Background(), &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          optionsTradablePair,
		AssetType:     asset.Options,
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
			result, err := b.UpdateAccountInfo(context.Background(), assetType)
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestWrapperGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{spotTradablePair},
		AssetType: asset.Spot,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{coinmTradablePair},
		AssetType: asset.CoinMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{usdtmTradablePair},
		AssetType: asset.USDTMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
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
	_, err := b.GetOrderHistory(context.Background(), &order.MultiOrderRequest{
		AssetType: asset.USDTMarginedFutures,
	})
	assert.Error(t, err, "expecting an error since invalid param combination is given. Got err: %v", err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	p, err := currency.NewPairFromString("EOSUSD_PERP")
	require.NoError(t, err)
	result, err := b.GetOrderHistory(context.Background(), &order.MultiOrderRequest{
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

	result, err = b.GetOrderHistory(context.Background(), &order.MultiOrderRequest{
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
	err = b.CancelOrder(context.Background(), &order.Cancel{
		AssetType: asset.CoinMarginedFutures,
		Pair:      fPair,
		OrderID:   "1234",
	})
	require.NoError(t, err)
	p2, err := currency.NewPairFromString("BTC-USDT")
	require.NoError(t, err)
	fpair2, err := b.FormatExchangeCurrency(p2, asset.USDTMarginedFutures)
	require.NoError(t, err)
	err = b.CancelOrder(context.Background(), &order.Cancel{
		AssetType: asset.USDTMarginedFutures,
		Pair:      fpair2,
		OrderID:   "1234",
	})
	require.NoError(t, err)
	err = b.CancelOrder(context.Background(), &order.Cancel{
		AssetType: asset.Options,
		Pair:      fpair2,
		OrderID:   "1234",
	})
	assert.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	tradablePairs, err := b.FetchTradablePairs(context.Background(),
		asset.CoinMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, tradablePairs, "no tradable pairs")
	result, err := b.GetOrderInfo(context.Background(), "123", tradablePairs[0], asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := b.ModifyOrder(context.Background(),
		&order.Modify{AssetType: asset.Spot})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetAllCoinsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCoinsInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.WithdrawCryptocurrencyFunds(context.Background(),
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
	result, err := b.DepositHistory(context.Background(), currency.ETH, "", time.Time{}, time.Time{}, 0, 10000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWithdrawalsHistory(context.Background(), currency.ETH, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawFiatFunds(context.Background(),
		&withdraw.Request{})
	assert.Equal(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawFiatFundsToInternationalBank(context.Background(),
		&withdraw.Request{})
	require.Equal(t, err, common.ErrFunctionNotSupported)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := b.GetDepositAddress(context.Background(), currency.USDT, "", currency.BNB.String())
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
	bb.ResetTimer()
	for range bb.N {
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
		mock := func(msg []byte, w *websocket.Conn) error {
			var req WsPayload
			require.NoError(t, json.Unmarshal(msg, &req), "Unmarshal should not error")
			require.ElementsMatch(t, req.Params, exp, "Params should have correct channels")
			return w.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"result":null,"id":%d}`, req.ID)))
		}
		b = testexch.MockWsInstance[Binance](t, testexch.CurryWsMockUpgrader(t, mock))
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
	mock := func(msg []byte, w *websocket.Conn) error {
		var req WsPayload
		err := json.Unmarshal(msg, &req)
		require.NoError(t, err, "Unmarshal should not error")
		return w.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"result":{"error":"carrots"},"id":%d}`, req.ID)))
	}
	b := testexch.MockWsInstance[Binance](t, testexch.CurryWsMockUpgrader(t, mock)) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	err := b.Subscribe(channels)
	require.ErrorIs(t, err, stream.ErrSubscriptionFailure, "Subscribe should error ErrSubscriptionFailure")
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
	  "E": 123456789,   
	  "s": "BTCUSDT",    
	  "k": {
		"t": 123400000, 
		"T": 123460000, 
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
	  "E": 123456789,   
	  "s": "BTCUSDT",    
	  "t": 12345,       
	  "p": "0.001",     
	  "q": "100",       
	  "b": 88,          
	  "a": 50,          
	  "T": 123456785,   
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
		Asks: []OrderbookItem{
			{Price: 6621.80000000, Quantity: 0.00198100},
			{Price: 6622.14000000, Quantity: 4.00000000},
			{Price: 6622.46000000, Quantity: 2.30000000},
			{Price: 6622.47000000, Quantity: 1.18633300},
			{Price: 6622.64000000, Quantity: 4.00000000},
			{Price: 6622.73000000, Quantity: 0.02900000},
			{Price: 6622.76000000, Quantity: 0.12557700},
			{Price: 6622.81000000, Quantity: 2.08994200},
			{Price: 6622.82000000, Quantity: 0.01500000},
			{Price: 6623.17000000, Quantity: 0.16831300},
		},
		Bids: []OrderbookItem{
			{Price: 6621.55000000, Quantity: 0.16356700},
			{Price: 6621.45000000, Quantity: 0.16352600},
			{Price: 6621.41000000, Quantity: 0.86091200},
			{Price: 6621.25000000, Quantity: 0.16914100},
			{Price: 6621.23000000, Quantity: 0.09193600},
			{Price: 6621.22000000, Quantity: 0.00755100},
			{Price: 6621.13000000, Quantity: 0.08432000},
			{Price: 6621.03000000, Quantity: 0.00172000},
			{Price: 6620.94000000, Quantity: 0.30506700},
			{Price: 6620.93000000, Quantity: 0.00200000},
		},
		LastUpdateID: seedLastUpdateID,
	}

	update1 := []byte(`{"stream":"btcusdt@depth","data":{
	  "e": "depthUpdate", 
	  "E": 123456788,     
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
	  "E": 123456789,     
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
	key, err := b.GetWsAuthStreamKey(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, key)
}

func TestMaintainWsAuthStreamKey(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.MaintainWsAuthStreamKey(context.Background())
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
	result, err := b.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.OneDay, start, end)
	require.NoErrorf(t, err, "%v %v", asset.Spot, err)
	require.NotNil(t, result)
	result, err = b.GetHistoricCandles(context.Background(), usdtmTradablePair, asset.USDTMarginedFutures, kline.OneDay, start, end)
	require.NoErrorf(t, err, "%v %v", asset.USDTMarginedFutures, err)
	require.NotNil(t, result)
	result, err = b.GetHistoricCandles(context.Background(), coinmTradablePair, asset.CoinMarginedFutures, kline.OneDay, start, end)
	require.NoErrorf(t, err, "%v %v", asset.CoinMarginedFutures, err)
	require.NotNil(t, result)
	result, err = b.GetHistoricCandles(context.Background(), optionsTradablePair, asset.Options, kline.OneDay, start, end)
	require.NoErrorf(t, err, "%v %v", asset.Options, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	result, err := b.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.OneDay, start, end)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetHistoricCandlesExtended(context.Background(), usdtmTradablePair, asset.USDTMarginedFutures, kline.OneDay, start, end)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetHistoricCandlesExtended(context.Background(), coinmTradablePair, asset.CoinMarginedFutures, kline.OneDay, start, end)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetHistoricCandlesExtended(context.Background(), optionsTradablePair, asset.Options, kline.OneDay, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
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
	result, err := b.GetRecentTrades(context.Background(),
		pair, asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetRecentTrades(context.Background(),
		pair, asset.USDTMarginedFutures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	pair.Base = currency.NewCode("BTCUSD")
	pair.Quote = currency.PERP
	result, err = b.GetRecentTrades(context.Background(), pair, asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := b.GetAvailableTransferChains(context.Background(), currency.BTC)
	require.False(t, sharedtestvalues.AreAPICredentialsSet(b) && err != nil, err)
	require.False(t, !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests, "error cannot be nil")
	assert.False(t, mockTests && err != nil, err)
}

func TestSeedLocalCache(t *testing.T) {
	t.Parallel()
	err := b.SeedLocalCache(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
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
	p := currency.NewPair(currency.BTC, currency.USDT)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	cp, err := currency.NewPairFromString("BTCUSDT")
	require.NoError(t, err)

	result, err := b.UFuturesHistoricalTrades(context.Background(), cp, "", 5)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.UFuturesHistoricalTrades(context.Background(), cp, "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetExchangeOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := b.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	require.NoError(t, err)
	err = b.UpdateOrderExecutionLimits(context.Background(), asset.CoinMarginedFutures)
	require.NoError(t, err)

	err = b.UpdateOrderExecutionLimits(context.Background(), asset.USDTMarginedFutures)
	require.NoError(t, err)

	err = b.UpdateOrderExecutionLimits(context.Background(), asset.Options)
	require.NoError(t, err)

	err = b.UpdateOrderExecutionLimits(context.Background(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	cmfCP, err := currency.NewPairFromStrings("BTCUSD", "PERP")
	require.NoError(t, err)

	limit, err := b.GetOrderExecutionLimits(asset.CoinMarginedFutures, cmfCP)
	require.NoError(t, err)
	require.NotEmpty(t, limit, "exchange limit should be loaded")

	err = limit.Conforms(0.000001, 0.1, order.Limit)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	err = limit.Conforms(0.01, 1, order.Limit)
	assert.ErrorIs(t, err, order.ErrPriceBelowMin)
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
	limits, err := b.FetchExchangeLimits(context.Background(), asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, limits, "Should get some limits back")

	limits, err = b.FetchExchangeLimits(context.Background(), asset.Margin)
	require.NoError(t, err)
	require.NotEmpty(t, limits, "Should get some limits back")

	_, err = b.FetchExchangeLimits(context.Background(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "FetchExchangeLimits should error on other asset types")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	tests := map[asset.Item]currency.Pair{
		asset.Spot:   currency.NewPair(currency.BTC, currency.USDT),
		asset.Margin: currency.NewPair(currency.ETH, currency.BTC),
	}
	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Options} {
		pairs, err := b.FetchTradablePairs(context.Background(), a)
		require.NoErrorf(t, err, "FetchTradablePairs should not error for %s", a)
		require.NotEmptyf(t, pairs, "Should get some pairs for %s", a)
		tests[a] = pairs[0]
	}
	for _, a := range b.GetAssetTypes(false) {
		err := b.UpdateOrderExecutionLimits(context.Background(), a)
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

func TestGetFundingRates(t *testing.T) {
	t.Parallel()
	s, e := getTime()
	_, err := b.GetHistoricalFundingRates(context.Background(), &fundingrate.HistoricalRatesRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewPair(currency.BTC, currency.USDT),
		StartDate:            s,
		EndDate:              e,
		IncludePayments:      true,
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	_, err = b.GetHistoricalFundingRates(context.Background(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewPair(currency.BTC, currency.USDT),
		StartDate:       s,
		EndDate:         e,
		PaymentCurrency: currency.DOGE,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	r := &fundingrate.HistoricalRatesRequest{
		Asset:     asset.USDTMarginedFutures,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		StartDate: s,
		EndDate:   e,
	}
	if sharedtestvalues.AreAPICredentialsSet(b) {
		r.IncludePayments = true
	}
	result, err := b.GetHistoricalFundingRates(context.Background(), r)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	r.Asset = asset.CoinMarginedFutures
	r.Pair, err = currency.NewPairFromString("BTCUSD_PERP")
	require.NoError(t, err)

	result, err = b.GetHistoricalFundingRates(context.Background(), r)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	cp := currency.NewPair(currency.BTC, currency.USDT)
	_, err := b.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 cp,
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
	err = b.CurrencyPairs.EnablePair(asset.USDTMarginedFutures, cp)
	require.True(t, err == nil || errors.Is(err, currency.ErrPairAlreadyEnabled), err)

	result, err := b.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  cp,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
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
	result, err := b.GetUserMarginInterestHistory(context.Background(), currency.USDT, "BTCUSDT", time.Now().Add(-time.Hour*24), time.Now(), 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetForceLiquidiationRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetForceLiquidiationRecord(context.Background(), time.Now().Add(-time.Hour*24), time.Now(), "BTCUSDT", 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCrossMarginAccountDetail(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountsOrder(context.Background(), "BTCUSDT", "", false, 112233424)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountsOpenOrders(context.Background(), "BNBBTC", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountAllOrders(context.Background(), "BNBBTC", true, time.Time{}, time.Time{}, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetAssetsMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	is, err := b.GetAssetsMode(context.Background())
	require.NoError(t, err)

	err = b.SetAssetsMode(context.Background(), !is)
	require.NoError(t, err)

	err = b.SetAssetsMode(context.Background(), is)
	assert.NoError(t, err)
}

func TestGetAssetsMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAssetsMode(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	_, err := b.GetCollateralMode(context.Background(), asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = b.GetCollateralMode(context.Background(), asset.CoinMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetCollateralMode(context.Background(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	err := b.SetCollateralMode(context.Background(), asset.USDTMarginedFutures, collateral.PortfolioMode)
	assert.ErrorIs(t, err, order.ErrCollateralInvalid)
	err = b.SetCollateralMode(context.Background(), asset.Spot, collateral.SingleMode)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	err = b.SetCollateralMode(context.Background(), asset.CoinMarginedFutures, collateral.SingleMode)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.SetCollateralMode(context.Background(), asset.USDTMarginedFutures, collateral.MultiMode)
	require.NoError(t, err)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangePositionMargin(context.Background(), &margin.PositionChangeRequest{
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
	_, err = b.GetFuturesPositionSummary(context.Background(), &futures.PositionSummaryRequest{
		Asset:          asset.Spot,
		Pair:           p,
		UnderlyingPair: currency.NewPair(currency.BTC, currency.USD),
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	bb := currency.NewBTCUSDT()
	result, err := b.GetFuturesPositionSummary(context.Background(), &futures.PositionSummaryRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  bb,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	bb.Quote = currency.BUSD
	result, err = b.GetFuturesPositionSummary(context.Background(), &futures.PositionSummaryRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  bb,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	bb.Quote = currency.USD
	result, err = b.GetFuturesPositionSummary(context.Background(), &futures.PositionSummaryRequest{
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
	result, err := b.GetFuturesPositionOrders(context.Background(), &futures.PositionsRequest{
		Asset:                     asset.USDTMarginedFutures,
		Pairs:                     []currency.Pair{currency.NewBTCUSDT()},
		StartDate:                 time.Now().Add(-time.Hour * 24 * 70),
		RespectOrderHistoryLimits: true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetFuturesPositionOrders(context.Background(), &futures.PositionsRequest{
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
	err := b.SetMarginType(context.Background(), asset.Spot, currency.NewPair(currency.BTC, currency.USDT), margin.Isolated)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.SetMarginType(context.Background(), asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.USDT), margin.Isolated)
	require.NoError(t, err)

	err = b.SetMarginType(context.Background(), asset.CoinMarginedFutures, coinmTradablePair, margin.Isolated)
	assert.NoError(t, err)
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := b.GetLeverage(context.Background(), asset.Spot, currency.NewBTCUSDT(), 0, order.UnknownSide)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLeverage(context.Background(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), 0, order.UnknownSide)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = b.GetLeverage(context.Background(), asset.CoinMarginedFutures, coinmTradablePair, 0, order.UnknownSide)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	err := b.SetLeverage(context.Background(), asset.Spot, spotTradablePair, margin.Multi, 5, order.UnknownSide)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err = b.SetLeverage(context.Background(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 5, order.UnknownSide)
	require.NoError(t, err)
	err = b.SetLeverage(context.Background(), asset.CoinMarginedFutures, coinmTradablePair, margin.Multi, 5, order.UnknownSide)
	require.NoError(t, err)
}

func TestGetCryptoLoansIncomeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanIncomeHistory(context.Background(), currency.USDT, "", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanBorrow(t *testing.T) {
	t.Parallel()
	_, err := b.CryptoLoanBorrow(context.Background(), currency.EMPTYCODE, 1000, currency.BTC, 1, 7)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.CryptoLoanBorrow(context.Background(), currency.USDT, 1000, currency.EMPTYCODE, 1, 7)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = b.CryptoLoanBorrow(context.Background(), currency.USDT, 0, currency.BTC, 1, 0)
	require.ErrorIs(t, err, errLoanTermMustBeSet)
	_, err = b.CryptoLoanBorrow(context.Background(), currency.USDT, 0, currency.BTC, 0, 7)
	require.ErrorIs(t, err, errEitherLoanOrCollateralAmountsMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CryptoLoanBorrow(context.Background(), currency.USDT, 1000, currency.BTC, 1, 7)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanBorrowHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanBorrowHistory(context.Background(), 0, currency.USDT, currency.BTC, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanOngoingOrders(context.Background(), 0, currency.USDT, currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanRepay(t *testing.T) {
	t.Parallel()
	_, err := b.CryptoLoanRepay(context.Background(), 0, 1000, 1, false)
	require.ErrorIs(t, err, errOrderIDMustBeSet)
	_, err = b.CryptoLoanRepay(context.Background(), 42069, 0, 1, false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CryptoLoanRepay(context.Background(), 42069, 1000, 1, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanRepaymentHistory(context.Background(), 0, currency.USDT, currency.BTC, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanAdjustLTV(t *testing.T) {
	t.Parallel()
	_, err := b.CryptoLoanAdjustLTV(context.Background(), 0, true, 1)
	require.ErrorIs(t, err, errOrderIDMustBeSet)
	_, err = b.CryptoLoanAdjustLTV(context.Background(), 42069, true, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CryptoLoanAdjustLTV(context.Background(), 42069, true, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanLTVAdjustmentHistory(context.Background(), 0, currency.USDT, currency.BTC, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanAssetsData(context.Background(), currency.EMPTYCODE, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanCollateralAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanCollateralAssetsData(context.Background(), currency.EMPTYCODE, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanCheckCollateralRepayRate(t *testing.T) {
	t.Parallel()
	_, err := b.CryptoLoanCheckCollateralRepayRate(context.Background(), currency.EMPTYCODE, currency.BNB, 69)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.CryptoLoanCheckCollateralRepayRate(context.Background(), currency.BUSD, currency.EMPTYCODE, 69)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = b.CryptoLoanCheckCollateralRepayRate(context.Background(), currency.BUSD, currency.BNB, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CryptoLoanCheckCollateralRepayRate(context.Background(), currency.BUSD, currency.BNB, 69)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCryptoLoanCustomiseMarginCall(t *testing.T) {
	t.Parallel()
	_, err := b.CryptoLoanCustomiseMarginCall(context.Background(), 0, currency.BTC, 0)
	assert.NotEmpty(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CryptoLoanCustomiseMarginCall(context.Background(), 1337, currency.BTC, .70)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanBorrow(t *testing.T) {
	t.Parallel()
	_, err := b.FlexibleLoanBorrow(context.Background(), currency.EMPTYCODE, currency.USDC, 1, 0)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.FlexibleLoanBorrow(context.Background(), currency.ATOM, currency.EMPTYCODE, 1, 0)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = b.FlexibleLoanBorrow(context.Background(), currency.ATOM, currency.USDC, 0, 0)
	require.ErrorIs(t, err, errEitherLoanOrCollateralAmountsMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FlexibleLoanBorrow(context.Background(), currency.ATOM, currency.USDC, 1, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleLoanOngoingOrders(context.Background(), currency.EMPTYCODE, currency.EMPTYCODE, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanBorrowHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleLoanBorrowHistory(context.Background(), currency.EMPTYCODE, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanRepay(t *testing.T) {
	t.Parallel()
	_, err := b.FlexibleLoanRepay(context.Background(), currency.EMPTYCODE, currency.BTC, 1, false, false)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.FlexibleLoanRepay(context.Background(), currency.USDT, currency.EMPTYCODE, 1, false, false)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)
	_, err = b.FlexibleLoanRepay(context.Background(), currency.USDT, currency.BTC, 0, false, false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FlexibleLoanRepay(context.Background(), currency.ATOM, currency.USDC, 1, false, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanRepayHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleLoanRepayHistory(context.Background(), currency.EMPTYCODE, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanAdjustLTV(t *testing.T) {
	t.Parallel()
	_, err := b.FlexibleLoanAdjustLTV(context.Background(), currency.EMPTYCODE, currency.BTC, 1, true)
	require.ErrorIs(t, err, errLoanCoinMustBeSet)
	_, err = b.FlexibleLoanAdjustLTV(context.Background(), currency.USDT, currency.EMPTYCODE, 1, true)
	require.ErrorIs(t, err, errCollateralCoinMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FlexibleLoanAdjustLTV(context.Background(), currency.USDT, currency.BTC, 1, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleLoanLTVAdjustmentHistory(context.Background(), currency.EMPTYCODE, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleLoanAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleLoanAssetsData(context.Background(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFlexibleCollateralAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FlexibleCollateralAssetsData(context.Background(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesContractDetails(context.Background(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = b.GetFuturesContractDetails(context.Background(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	result, err := b.GetFuturesContractDetails(context.Background(), asset.USDTMarginedFutures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = b.GetFuturesContractDetails(context.Background(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	result, err := b.GetFundingRateInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	result, err := b.UGetFundingRateInfo(context.Background())
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
	"Single Market Ticker":          `{"stream": "<symbol>@ticker", "data": { "e": "24hrTicker", "E": 123456789, "s": "BTCUSDT", "p": "0.0015", "P": "250.00", "w": "0.0018", "c": "0.0025", "Q": "10", "o": "0.0010", "h": "0.0025", "l": "0.0010", "v": "10000", "q": "18", "O": 0, "C": 86400000, "F": 0, "L": 18150, "n": 18151 } }`,
	"Multiple Mini Tickers":         `{"stream": "!miniTicker@arr","data":[{"e":"24hrMiniTicker","E":1703019429455,"s":"BICOUSDT","c":"0.3667000","o":"0.3792000","h":"0.3892000","l":"0.3639000","v":"28768370","q":"10779000.9922000"},{"e":"24hrMiniTicker","E":1703019429985,"s":"API3USDT","c":"1.6834","o":"1.7326","h":"1.8406","l":"1.6699","v":"12371516.4","q":"21642153.0574"},{"e":"24hrMiniTicker","E":1703019429111,"s":"ICPUSDT","c":"9.414000","o":"10.126000","h":"10.956000","l":"9.236000","v":"34262192","q":"339148145.539000"},{"e":"24hrMiniTicker","E":1703019429945,"s":"SOLUSDT","c":"73.0930","o":"73.2180","h":"76.3840","l":"71.8000","v":"26319095","q":"1960871540.2620"}]}`,
	"Multi Asset Mode Asset":        `{"stream": "!assetIndex@arr", "data":[{ "e":"assetIndexUpdate", "E":1686749230000, "s":"ADAUSD","i":"0.27462452","b":"0.10000000","a":"0.10000000","B":"0.24716207","A":"0.30208698","q":"0.05000000","g":"0.05000000","Q":"0.26089330","G":"0.28835575"}, { "e":"assetIndexUpdate", "E":1686749230000, "s":"USDTUSD", "i":"0.99987691", "b":"0.00010000", "a":"0.00010000", "B":"0.99977692", "A":"0.99997689", "q":"0.00010000", "g":"0.00010000", "Q":"0.99977692", "G":"0.99997689" }]}`,
	"Composite Index Symbol":        `{"stream": "<symbol>@compositeIndex", "data":{ "e":"compositeIndex", "E":1602310596000, "s":"DEFIUSDT", "p":"554.41604065", "C":"baseAsset", "c":[ { "b":"BAL", "q":"USDT", "w":"1.04884844", "W":"0.01457800", "i":"24.33521021" }, { "b":"BAND", "q":"USDT" , "w":"3.53782729", "W":"0.03935200", "i":"7.26420084" } ] } }`,
	"Diff Book Depth Stream":        `{"stream": "<symbol>@depth@500ms", "data": { "e": "depthUpdate", "E": 123456789, "T": 123456788, "s": "BTCUSDT", "U": 157, "u": 160, "pu": 149, "b": [ [ "0.0024", "10" ] ], "a": [ [ "0.0026", "100" ] ] } }`,
	"Partial Book Depth Stream":     `{"stream": "<symbol>@depth<levels>", "data":{ "e": "depthUpdate", "E": 1571889248277, "T": 1571889248276, "s": "BTCUSDT", "U": 390497796, "u": 390497878, "pu": 390497794, "b": [ [ "7403.89", "0.002" ], [ "7403.90", "3.906" ], [ "7404.00", "1.428" ], [ "7404.85", "5.239" ], [ "7405.43", "2.562" ] ], "a": [ [ "7405.96", "3.340" ], [ "7406.63", "4.525" ], [ "7407.08", "2.475" ], [ "7407.15", "4.800" ], [ "7407.20","0.175"]]}}`,
	"Individual Symbol Mini Ticker": `{"stream": "<symbol>@miniTicker", "data": { "e": "24hrMiniTicker", "E": 123456789, "s": "BTCUSDT", "c": "0.0025", "o": "0.0010", "h": "0.0025", "l": "0.0010", "v": "10000", "q": "18"}}`,
}

func TestHandleData(t *testing.T) {
	t.Parallel()
	for x := range messageMap {
		err := b.wsHandleFuturesData([]byte(messageMap[x]), asset.USDTMarginedFutures)
		require.NoError(t, err)
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
	if mockTests {
		t.SkipNow()
	}
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.GetWs24HourPriceChanges(&PriceChangeRequestParam{
		Symbols: []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWsTradingDayTickers(t *testing.T) {
	t.Parallel()
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
		Side:        "SELL",
		OrderType:   "LIMIT",
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
		Side:        "SELL",
		OrderType:   "LIMIT",
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
		Side:                      "SELL",
		OrderType:                 "LIMIT",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsPlaceOCOOrder(&PlaceOCOOrderParam{
		Symbol:               "BTCUSDT",
		Side:                 "SELL",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsPlaceNewSOROrder(&WsOSRPlaceOrderParams{
		Symbol:      "BTCUSDT",
		Side:        "BUY",
		OrderType:   "LIMIT",
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
		OrderType:   "LIMIT",
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
	params := map[string]interface{}{"apiKey": "wwhj3r3amR", "signature": "f89c6e5c0b", "timestamp": 1704873175325, "symbol": "BTCUSDT", "startTime": 1704009175325, "endTime": 1704873175325, "limit": 5}
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsAccountTradeHistory(&AccountOrderRequestParam{
		Symbol:  "BTCUSDT",
		OrderID: 1234,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAccountPreventedMatches(t *testing.T) {
	t.Parallel()
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsAccountAllocation(currency.NewPair(currency.BTC, currency.USDT), time.Time{}, time.Now(), 0, 0, 0, 19)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAccountCommissionRates(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	result, err := b.WsAccountCommissionRates(currency.NewPair(currency.BTC, currency.USDT))
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	err := b.WsPingUserDataStream("xs0mRXdAKlIPDRFrlPcw0qI41Eh3ixNntmymGyhrhgqo7L6FuLaWArTD7RLP")
	require.NoError(t, err)
}

func TestWsStopUserDataStream(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if !b.IsAPIStreamConnected() {
		t.Skip(apiStreamingIsNotConnected)
	}
	err := b.WsStopUserDataStream("xs0mRXdAKlIPDRFrlPcw0qI41Eh3ixNntmymGyhrhgqo7L6FuLaWArTD7RLP")
	require.NoError(t, err)
}
func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.Spot,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := b.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result)

	result, err = b.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  currency.NewCode("BTCUSD").Item,
		Quote: currency.PERP.Item,
		Asset: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestSystemStatus(t *testing.T) {
	t.Parallel()
	result, err := b.GetSystemStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDailyAccountSnapshot(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDailyAccountSnapshot(context.Background(), "SPOT", time.Time{}, time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDisableFastWithdrawalSwitch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.DisableFastWithdrawalSwitch(context.Background())
	assert.NoError(t, err)
}

func TestEnableFastWithdrawalSwitch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.EnableFastWithdrawalSwitch(context.Background())
	assert.NoError(t, err)
}

func TestGetAccountStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTradingAPIStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountTradingAPIStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDustLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDustLog(context.Background(), "MARGIN", time.Time{}, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckServerTime(t *testing.T) {
	t.Parallel()
	result, err := b.GetExchangeServerTime(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccount(context.Background(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTradeList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountTradeList(context.Background(), "BNBBTC", "", time.Now().Add(-time.Hour*5), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentOrderCountUsage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrentOrderCountUsage(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPreventedMatches(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPreventedMatches(context.Background(), "BTCUSDT", 0, 12, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllocations(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllocations(context.Background(), "BTCUSDT", time.Time{}, time.Time{}, 10, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCommissionRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCommissionRates(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountBorrowRepay(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.MarginAccountBorrowRepay(context.Background(), currency.ETH, "BTCUSDT", "BORROW", false, 0.1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowOrRepayRecordsInMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBorrowOrRepayRecordsInMarginAccount(context.Background(), currency.LTC, "", "REPAY", 0, 10, 0, time.Now().Add(-time.Hour*12), time.Now().Add(-time.Hour*6))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllMarginAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllMarginAssets(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCrossMarginPairs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCrossMarginPairs(context.Background(), "BNBBTC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginPriceIndex(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginPriceIndex(context.Background(), "BNBBTC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPostMarginAccountOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.PostMarginAccountOrder(context.Background(), &MarginAccountOrderParam{
		Symbol:    currency.NewPair(currency.BTC, currency.USDT),
		Side:      order.Buy.String(),
		OrderType: order.Limit.String(),
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMarginAccountOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelMarginAccountOrder(context.Background(), "BTCUSDT", "", "", true, 12314234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountCancelAllOpenOrdersOnSymbol(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.MarginAccountCancelAllOpenOrdersOnSymbol(context.Background(), "BTCUSDT", true)
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
	err := b.ChangePositionMode(context.Background(), false)
	assert.NoError(t, err)
}

func TestGetCurrentPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrentPositionMode(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ---------------------------  European Option Endpoints test -----------------------------------

func TestCheckEOptionsServerTime(t *testing.T) {
	t.Parallel()
	serverTime, err := b.CheckEOptionsServerTime(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, serverTime)
}

func TestGetEOptionsOrderbook(t *testing.T) {
	t.Parallel()
	optionsTradablePairString := "ETH-240927-3800-P"
	if !mockTests {
		optionsTradablePairString = optionsTradablePair.String()
	}
	result, err := b.GetEOptionsOrderbook(context.Background(), optionsTradablePairString, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetEOptionsRecentTrades(context.Background(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetEOptionsRecentTrades(context.Background(), "BTC-240330-80500-P", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetEOptionsTradeHistory(context.Background(), "", 0, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetEOptionsTradeHistory(context.Background(), "BTC-240330-80500-P", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsCandlesticks(t *testing.T) {
	t.Parallel()
	optionsTradablePairString := "ETH-240927-3800-P"
	if !mockTests {
		optionsTradablePairString = optionsTradablePair.String()
	}
	start, end := getTime()
	result, err := b.GetEOptionsCandlesticks(context.Background(), optionsTradablePairString, kline.OneDay, start, end, 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionMarkPrice(t *testing.T) {
	t.Parallel()
	optionsTradablePairString := "ETH-240927-3800-P"
	if !mockTests {
		optionsTradablePairString = optionsTradablePair.String()
	}
	result, err := b.GetOptionMarkPrice(context.Background(), optionsTradablePairString)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptions24hrTickerPriceChangeStatistics(t *testing.T) {
	t.Parallel()
	optionsTradablePairString := "ETH-240927-3800-P"
	if !mockTests {
		optionsTradablePairString = optionsTradablePair.String()
	}
	result, err := b.GetEOptions24hrTickerPriceChangeStatistics(context.Background(), optionsTradablePairString)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	result, err := b.GetEOptionsSymbolPriceTicker(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsHistoricalExerciseRecords(t *testing.T) {
	t.Parallel()
	result, err := b.GetEOptionsHistoricalExerciseRecords(context.Background(), "BTCUSDT", time.Time{}, time.Now(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsOpenInterests(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip("endpoint has problem")
	}
	result, err := b.GetEOptionsOpenInterests(context.Background(), "ETH", time.Now().Add(time.Hour*24))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOptionsAccountInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOptionsOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewOptionsOrder(context.Background(), &OptionsOrderParams{
		Symbol:                  currency.Pair{Base: currency.NewCode("BTC"), Delimiter: currency.DashDelimiter, Quote: currency.NewCode("200730-9000-C")},
		Side:                    "Sell",
		OrderType:               "LIMIT",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.PlaceBatchEOptionsOrder(context.Background(), []OptionsOrderParams{
		{
			Symbol:                  currency.Pair{Base: currency.NewCode("BTC"), Delimiter: currency.DashDelimiter, Quote: currency.NewCode("200730-9000-C")},
			Side:                    "Sell",
			OrderType:               "LIMIT",
			Amount:                  0.00001,
			Price:                   0.00001,
			ReduceOnly:              false,
			PostOnly:                true,
			NewOrderResponseType:    "RESULT",
			ClientOrderID:           "the-client-order-id",
			IsMarketMakerProtection: true,
		}, {
			Symbol:                  currency.Pair{Base: currency.NewCode("BTC"), Delimiter: currency.DashDelimiter, Quote: currency.NewCode("200730-9000-C")},
			Side:                    "Buy",
			OrderType:               "Market",
			Amount:                  0.00001,
			PostOnly:                true,
			NewOrderResponseType:    "RESULT",
			ClientOrderID:           "the-client-order-id-2",
			IsMarketMakerProtection: true,
		}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSingleEOptionsOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSingleEOptionsOrder(context.Background(), "BTC-200730-9000-C", "", 4611875134427365377)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOptionsOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelOptionsOrder(context.Background(), "BTC-200730-9000-C", "213123", 4611875134427365377)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelBatchOptionsOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelBatchOptionsOrders(context.Background(), "BTC-200730-9000-C", []int64{4611875134427365377}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOptionOrdersOnSpecificSymbol(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.CancelAllOptionOrdersOnSpecificSymbol(context.Background(), "BTC-200730-9000-C")
	assert.NoError(t, err)
}

func TestCancelAllOptionsOrdersByUnderlying(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllOptionsOrdersByUnderlying(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentOpenOptionsOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	results, err := b.GetCurrentOpenOptionsOrders(context.Background(), "BTC-200730-9000-C", time.Time{}, time.Time{}, 4611875134427365377, 0)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestGetOptionsOrdersHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	results, err := b.GetOptionsOrdersHistory(context.Background(), "BTC-200730-9000-C", time.Time{}, time.Time{}, 4611875134427365377, 0)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestGetOptionPositionInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOptionPositionInformation(context.Background(), "BTC-200730-9000-C")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEOptionsAccountTradeList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetEOptionsAccountTradeList(context.Background(), "BTC-200730-9000-C", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserOptionsExerciseRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUserOptionsExerciseRecord(context.Background(), "BTC-200730-9000-C", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingFlow(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountFundingFlow(context.Background(), currency.USDT, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDownloadIDForOptionTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDownloadIDForOptionTransactionHistory(context.Background(), time.Now().Add(-time.Hour*24*10), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionTransactionHistoryDownloadLinkByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOptionTransactionHistoryDownloadLinkByID(context.Background(), "download-id")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionMarginAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOptionMarginAccountInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetMarketMakerProtectionConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SetOptionsMarketMakerProtectionConfig(context.Background(), &MarketMakerProtectionConfig{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOptionsMarketMakerProtection(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetMarketMaketProtection(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.ResetMarketMaketProtection(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetOptionsAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.SetOptionsAutoCancelAllOpenOrders(context.Background(), "BTCUSDT", 30000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoCancelAllOpenOrdersConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAutoCancelAllOpenOrdersConfig(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsAutoCancelAllOpenOrdersHeartbeat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetOptionsAutoCancelAllOpenOrdersHeartbeat(context.Background(), []string{"ETHUSDT"})
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
	exchangeinformation, err := b.GetOptionsExchangeInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, exchangeinformation)
}

// ---------------------------------------   Portfolio Margin  ---------------------------------------------

func TestNewUMOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewUMOrder(context.Background(), &UMOrderParam{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewCMOrder(context.Background(), &UMOrderParam{
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

func TestMarginAccountBorrow(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.MarginAccountBorrow(context.Background(), currency.USDT, 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountRepay(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.MarginAccountRepay(context.Background(), currency.USDT, 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginAccountNewOCO(t *testing.T) {
	t.Parallel()
	_, err := b.MarginAccountNewOCO(context.Background(), &OCOOrderParam{})
	require.ErrorIs(t, err, errNilArgument)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewOCOOrder(context.Background(), &OCOOrderParam{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.NewOCOOrderList(context.Background(), &OCOOrderListParams{
		Symbol:     "LTCBTC",
		Side:       "SELL",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewUMConditionalOrder(context.Background(), &ConditionalOrderParam{
		Symbol:       "BTCUSDT",
		Side:         "Sell",
		PositionSide: "SHORT",
		StrategyType: "STOP_MARKET",
		PriceProtect: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewCMConditionalOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewCMConditionalOrder(context.Background(), &ConditionalOrderParam{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelUMOrder(context.Background(), "BTCUSDT", "", 1234132)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelCMOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelCMOrder(context.Background(), "BTCUSDT", "", 21321312)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllUMOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllUMOrders(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 200, result.Code)
}

func TestCancelAllCMOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllCMOrders(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPMCancelMarginAccountOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.PMCancelMarginAccountOrder(context.Background(), "LTCBTC", "", 12314)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllMarginOpenOrdersBySymbol(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllMarginOpenOrdersBySymbol(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMarginAccountOCOOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelMarginAccountOCOOrders(context.Background(), "LTCBTC", "", "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelUMConditionalOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelUMConditionalOrder(context.Background(), "LTCBTC", "", 2000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelCMConditionalOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelCMConditionalOrder(context.Background(), "LTCBTC", "", 1231231)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllUMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllUMOpenConditionalOrders(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllCMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelAllCMOpenConditionalOrders(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMOrder(context.Background(), "BTCUSDT", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMOpenOrder(context.Background(), "BTCUSDT", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllUMOpenOrders(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllUMOrders(context.Background(), "BTCUSDT", time.Now().Add(-time.Hour*24*6), time.Now().Add(-time.Hour*2), 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMOrder(context.Background(), "BTCLTC", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMOpenOrder(context.Background(), "BTCLTC", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCMOpenOrders(context.Background(), "BTCUSD_200925", "BTCUSD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCMOrders(context.Background(), "BTCUSD_200925", "BTCUSD", time.Time{}, time.Time{}, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenUMConditionalOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOpenUMConditionalOrder(context.Background(), "BTCUSDT", "newClientStrategyId", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllUMOpenConditionalOrders(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMConditionalOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllUMConditionalOrderHistory(context.Background(), "BTCUSDT", "abc", 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUMConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllUMConditionalOrders(context.Background(), "BTCUSDT", time.Time{}, time.Now(), 0, 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenCMConditionalOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOpenCMConditionalOrder(context.Background(), "BTCUSD", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCMOpenConditionalOrders(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMConditionalOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCMConditionalOrderHistory(context.Background(), "BTCUSDT", "abc", 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllCMConditionalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllCMConditionalOrders(context.Background(), "BTCUSDT", time.Time{}, time.Now(), 0, 123432423)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountOrder(context.Background(), "BNBBTC", "", 12434)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentMarginOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrentMarginOpenOrder(context.Background(), "BNBBTC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllMarginAccountOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllMarginAccountOrders(context.Background(), "BNBBTC", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountOCO(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountOCO(context.Background(), 0, "123421-abcde")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPMMarginAccountAllOCO(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPMMarginAccountAllOCO(context.Background(), time.Now().Add(-time.Hour*24), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOpenOCO(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountsOpenOCO(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPMMarginAccountTradeList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPMMarginAccountTradeList(context.Background(), "BNBBTC", time.Time{}, time.Time{}, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountBalance(context.Background(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPortfolioMarginAccountInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginMaxBorrow(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPMMarginMaxBorrow(context.Background(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginMaxWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginMaxWithdrawal(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMPositionInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMPositionInformation(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMPositionInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMPositionInformation(context.Background(), currency.ETH, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeUMInitialLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeUMInitialLeverage(context.Background(), "BTCUSDT", 29)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeCMInitialLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeCMInitialLeverage(context.Background(), "BTCUSDT", 29)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeUMPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeUMPositionMode(context.Background(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeCMPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeCMPositionMode(context.Background(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMCurrentPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMCurrentPositionMode(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMCurrentPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMCurrentPositionMode(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMAccountTradeList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMAccountTradeList(context.Background(), "BTCUSDT", time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMAccountTradeList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMAccountTradeList(context.Background(), "BTCUSD_200626", "BTCUSDT", time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMNotionalAndLeverageBrackets(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMNotionalAndLeverageBrackets(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersMarginForceOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUsersMarginForceOrders(context.Background(), time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 5)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersUMForceOrderst(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUsersUMForceOrders(context.Background(), "BTCUSDT", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersCMForceOrderst(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUsersCMForceOrders(context.Background(), "BTCUSDT", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginUMTradingQuantitativeRulesIndicator(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPortfolioMarginUMTradingQuantitativeRulesIndicator(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMUserCommissionRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMUserCommissionRate(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestGetCMUserCommissionRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMUserCommissionRate(context.Background(), "BTCUSD_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginLoanRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginLoanRecord(context.Background(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginRepayRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginRepayRecord(context.Background(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginBorrowOrLoanInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginBorrowOrLoanInterestHistory(context.Background(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 0, 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginNegativeBalanceInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPortfolioMarginNegativeBalanceInterestHistory(context.Background(), currency.ETH, time.Now().Add(-time.Hour*24*5), time.Now().Add(-time.Hour*24), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundAutoCollection(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FundAutoCollection(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundCollectionByAsset(t *testing.T) {
	t.Parallel()
	_, err := b.FundCollectionByAsset(context.Background(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FundCollectionByAsset(context.Background(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBNBTransferClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.BNBTransferClassic(context.Background(), 0.0001, "TO_UM")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBNBTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.BNBTransfer(context.Background(), 0.0001, "TO_UM")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMAccountDetail(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMAccountDetail(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.ChangeAutoRepayFuturesStatus(context.Background(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAutoRepayFuturesStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayFuturesNegativeBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RepayFuturesNegativeBalance(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUMPositionADLQuantileEstimation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetUMPositionADLQuantileEstimation(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCMPositionADLQuantileEstimation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCMPositionADLQuantileEstimation(context.Background(), "BTCUSD_200925")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAdjustCrossMarginMaxLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.AdjustCrossMarginMaxLeverage(context.Background(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCrossMarginTransferHistory(context.Background(), currency.ETH, "ROLL_IN", "", time.Time{}, time.Time{}, 10, 30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewMarginAccountOCOOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewMarginAccountOCOOrder(context.Background(), &MarginOCOOrderParam{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelMarginAccountOCOOrder(context.Background(), "LTCBTC", "12345678", "", true, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountOCOOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountOCOOrder(context.Background(), "LTCBTC", "12345", 0, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountAllOCO(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountAllOCO(context.Background(), "LTCBTC", true, time.Now().Add(-time.Hour*24), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountsOpenOCOOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountsOpenOCOOrder(context.Background(), true, usdtmTradablePair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountTradeList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAccountTradeList(context.Background(), "BNBBTC", true, time.Time{}, time.Time{}, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaxBorrow(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMaxBorrow(context.Background(), currency.ETH, "BTCETH")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaxTransferOutAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMaxTransferOutAmount(context.Background(), currency.ETH, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSummaryOfMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSummaryOfMarginAccount(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIsolatedMarginAccountInfo(context.Background(), []string{"BTCUSDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDisableIsolatedMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.DisableIsolatedMarginAccount(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableIsolatedMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.EnableIsolatedMarginAccount(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEnabledIsolatedMarginAccountLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetEnabledIsolatedMarginAccountLimit(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllIsolatedMarginSymbols(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllIsolatedMarginSymbols(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestToggleBNBBurn(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ToggleBNBBurn(context.Background(), true, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBNBBurnStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetBNBBurnStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginInterestRateHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginInterestRateHistory(context.Background(), currency.ETH, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginFeeData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCrossMarginFeeData(context.Background(), 0, currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMaringFeeData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIsolatedMaringFeeData(context.Background(), 1, "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginTierData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIsolatedMarginTierData(context.Background(), "BTCUSDT", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyMarginOrderCountUsage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrencyMarginOrderCountUsage(context.Background(), true, "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCrossMarginCollateralRatio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCrossMarginCollateralRatio(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmallLiabilityExchangeCoinList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSmallLiabilityExchangeCoinList(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginSmallLiabilityExchange(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.MarginSmallLiabilityExchange(context.Background(), []string{"BTC", "ETH"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmallLiabilityExchangeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSmallLiabilityExchangeHistory(context.Background(), 1, 10, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFutureHourlyInterestRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFutureHourlyInterestRate(context.Background(), []string{"BTC", "ETH"}, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossOrIsolatedMarginCapitalFlow(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCrossOrIsolatedMarginCapitalFlow(context.Background(), currency.ETH, "", "BORROW", time.Time{}, time.Time{}, 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTokensOrSymbolsDelistSchedule(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetTokensOrSymbolsDelistSchedule(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAvailableInventory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMarginAvailableInventory(context.Background(), "ISOLATED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginManualLiquidiation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.MarginManualLiquidiation(context.Background(), "ISOLATED", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLiabilityCoinLeverageBracketInCrossMarginProMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLiabilityCoinLeverageBracketInCrossMarginProMode(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSimpleEarnFlexibleProductList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSimpleEarnFlexibleProductList(context.Background(), currency.BTC, 2, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSimpleEarnLockedProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSimpleEarnLockedProducts(context.Background(), currency.BTC, 2, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeToFlexibleProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubscribeToFlexibleProducts(context.Background(), "product-id", "FUND", 1, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeToLockedProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubscribeToLockedProducts(context.Background(), "project-id", "SPOT", 1, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemFlexibleProduct(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RedeemFlexibleProduct(context.Background(), "product-id", "FUND", true, 0.1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemLockedProduct(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RedeemLockedProduct(context.Background(), 12345)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleProductPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleProductPosition(context.Background(), currency.BTC, "", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedProductPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedProductPosition(context.Background(), currency.ETH, "", "", 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSimpleAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.SimpleAccount(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleSubscriptionRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleSubscriptionRecord(context.Background(), "", "", currency.ETH, time.Now().Add(-time.Hour*48), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedSubscriptionsRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedSubscriptionsRecords(context.Background(), "", currency.ETH, time.Now().Add(-time.Hour*480), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleRedemptionRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleRedemptionRecord(context.Background(), "", "1234", currency.LTC, time.Now().Add(-time.Hour*48), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedRedemptionRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedRedemptionRecord(context.Background(), "", "1234", currency.LTC, time.Now().Add(-time.Hour*48), time.Now(), 0, 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleRewardHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleRewardHistory(context.Background(), "product-type", "", currency.BTC, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2), 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedRewardHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedRewardHistory(context.Background(), "12345", currency.BTC, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2), 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetFlexibleAutoSusbcribe(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SetFlexibleAutoSusbcribe(context.Background(), "product-id", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLockedAutoSubscribe(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SetLockedAutoSubscribe(context.Background(), "position-id", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexiblePersonalLeftQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexiblePersonalLeftQuota(context.Background(), "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedPersonalLeftQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedPersonalLeftQuota(context.Background(), "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlexibleSubscriptionPreview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFlexibleSubscriptionPreview(context.Background(), "1234", 0.0001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLockedSubscriptionPreview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLockedSubscriptionPreview(context.Background(), "12345", 0.1234, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSimpleEarnRatehistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSimpleEarnRatehistory(context.Background(), "project-id", time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSimpleEarnCollateralRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSimpleEarnCollateralRecord(context.Background(), "project-id", time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDualInvestmentProductList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetDualInvestmentProductList(context.Background(), "CALL", currency.BTC, currency.ETH, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeDualInvestmentProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubscribeDualInvestmentProducts(context.Background(), "1234", "order-id", "STANDARD", 0.1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDualInvestmentPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDualInvestmentPositions(context.Background(), "PURCHASE_FAIL", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckDualInvestmentAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CheckDualInvestmentAccounts(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAutoCompoundStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeAutoCompoundStatus(context.Background(), "123456789", "STANDARD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTargetAssetList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetTargetAssetList(context.Background(), currency.BTC, 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTargetAssetROIData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetTargetAssetROIData(context.Background(), currency.ETH, "THREE_YEAR")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllSourceAssetAndTargetAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAllSourceAssetAndTargetAsset(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSourceAssetList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSourceAssetList(context.Background(), currency.BTC, 123, "RECURRING", "MAIN_SITE", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInvestmentPlanCreation(t *testing.T) {
	t.Parallel()
	_, err := b.InvestmentPlanCreation(context.Background(), &InvestmentPlanParams{})
	require.ErrorIs(t, errNilArgument, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.InvestmentPlanCreation(context.Background(), &InvestmentPlanParams{
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
	_, err := b.InvestmentPlanAdjustment(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.InvestmentPlanAdjustment(context.Background(), &AdjustInvestmentPlan{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangePlanStatus(context.Background(), 12345, "PAUSED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetListOfPlans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetListOfPlans(context.Background(), "SINGLE")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHoldingDetailsOfPlan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetHoldingDetailsOfPlan(context.Background(), 1234, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubscriptionsTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubscriptionsTransactionHistory(context.Background(), 1232, 20, 0, time.Time{}, time.Time{}, currency.BTC, "PORTFOLIO")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIndexDetail(context.Background(), 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexLinkedPlanPositionDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIndexLinkedPlanPositionDetails(context.Background(), 123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOneTimeTransaction(t *testing.T) {
	t.Parallel()
	_, err := b.OneTimeTransaction(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.OneTimeTransaction(context.Background(), &OneTimeTransactionParams{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOneTimeTransactionStatus(context.Background(), 1234, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIndexLinkedPlanRedemption(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.IndexLinkedPlanRedemption(context.Background(), 12333, 30, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexLinkedPlanRedemption(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetIndexLinkedPlanRedemption(context.Background(), "123123", time.Now().Add(-time.Hour*48), time.Now(), currency.ETH, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexLinkedPlanRebalanceDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetIndexLinkedPlanRebalanceDetails(context.Background(), time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubscribeETHStaking(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSubscribeETHStaking(context.Background(), 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSusbcribeETHStakingV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.SusbcribeETHStakingV2(context.Background(), 0.123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemETH(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RedeemETH(context.Background(), 0.123, currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHStakingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetETHStakingHistory(context.Background(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHRedemptionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetETHRedemptionHistory(context.Background(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBETHRewardsDistributionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBETHRewardsDistributionHistory(context.Background(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentETHStakingQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrentETHStakingQuota(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHRateHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWBETHRateHistory(context.Background(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHStakingAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetETHStakingAccount(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetETHStakingAccountV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetETHStakingAccountV2(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWrapBETH(t *testing.T) {
	t.Parallel()
	_, err := b.WrapBETH(context.Background(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.WrapBETH(context.Background(), 0.001)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHWrapHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWBETHWrapHistory(context.Background(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHUnwrapHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWBETHUnwrapHistory(context.Background(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWBETHRewardHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWBETHRewardHistory(context.Background(), time.Now().Add(-time.Hour*48), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcquiringAlgorithm(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.AcquiringAlgorithm(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCoinNames(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCoinNames(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailMinerList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetDetailMinerList(context.Background(), "sha256", "sams", "bhdc1.16A10404B")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMinersList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMinersList(context.Background(), "sha256", "sams", true, 0, 10, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarningList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetEarningList(context.Background(), "sha256", "sams", currency.ETH, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExtraBonousList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.ExtraBonousList(context.Background(), "sha256", "sams", currency.ETH, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHashrateRescaleList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetHashrateRescaleList(context.Background(), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHashrateRescaleDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetHashrateRescaleDetail(context.Background(), "168", "sams", 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHashrateRescaleRequest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.HashrateRescaleRequest(context.Background(), "sams", "sha256", "S19pro", time.Time{}, time.Time{}, 10000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelHashrateRescaleConfiguration(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelHashrateRescaleConfiguration(context.Background(), "189", "sams")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestStatisticsList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.StatisticsList(context.Background(), "sha256", "sams")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAccountList(context.Background(), "sha256", "sams")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMiningAccountEarningRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetMiningAccountEarningRate(context.Background(), "sha256", time.Now().Add(-time.Hour*240), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewFuturesAccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.NewFuturesAccountTransfer(context.Background(), currency.ETH, 0.001, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountTransactionHistoryList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.GetFuturesAccountTransactionHistoryList(context.Background(), currency.BTC, time.Now().Add(-time.Hour*20), time.Time{}, 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFutureTickLevelOrderbookHistoricalDataDownloadLink(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(context.Background(), "BTCUSDT", "T_DEPTH", time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*3))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVolumeParticipationNewOrder(t *testing.T) {
	t.Parallel()
	_, err := b.VolumeParticipationNewOrder(context.Background(), &VolumeParticipationOrderParams{})
	require.ErrorIs(t, err, errNilArgument)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.VolumeParticipationNewOrder(context.Background(), &VolumeParticipationOrderParams{
		Symbol:       "BTCUSDT",
		Side:         "SELL",
		PositionSide: "BOTH",
		Quantity:     0.012,
		Urgency:      "HIGH",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTWAPOrder(t *testing.T) {
	t.Parallel()
	_, err := b.FuturesTWAPOrder(context.Background(), &TWAPOrderParams{})
	require.ErrorIs(t, err, errNilArgument)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.FuturesTWAPOrder(context.Background(), &TWAPOrderParams{
		Symbol:       "BTCUSDT",
		Side:         "SELL",
		PositionSide: "BOTH",
		Quantity:     0.012,
		Duration:     1000,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelFuturesAlgoOrder(context.Background(), 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentAlgoOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesCurrentAlgoOpenOrders(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalAlgoOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesHistoricalAlgoOrders(context.Background(), "BNBUSDT", "BUY", time.Time{}, time.Time{}, 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFuturesSubOrders(context.Background(), 1234, 0, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTWAPNewOrder(t *testing.T) {
	t.Parallel()
	_, err := b.SpotTWAPNewOrder(context.Background(), &SpotTWAPOrderParam{})
	require.ErrorIs(t, err, errNilArgument)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SpotTWAPNewOrder(context.Background(), &SpotTWAPOrderParam{
		Symbol:   "BTCUSDT",
		Side:     "SELL",
		Quantity: 0.012,
		Duration: 86400,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSpotAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelSpotAlgoOrder(context.Background(), 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentSpotAlgoOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetCurrentSpotAlgoOpenOrder(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotHistoricalAlgoOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotHistoricalAlgoOrders(context.Background(), "BNBUSDT", "BUY", time.Time{}, time.Time{}, 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotSubOrders(context.Background(), 1234, 0, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPortfolioMarginAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetClassicPortfolioMarginAccountInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPortfolioMarginCollateralRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetClassicPortfolioMarginCollateralRate(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPortfolioMarginBankruptacyLoanAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetClassicPortfolioMarginBankruptacyLoanAmount(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayClassicPMBankruptacyLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RepayClassicPMBankruptacyLoan(context.Background(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetClassicPMNegativeBalanceInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetClassicPMNegativeBalanceInterestHistory(context.Background(), currency.ETH, time.Now().Add(-time.Hour*48*100), time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPMAssetIndexPrice(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPMAssetIndexPrice(context.Background(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClassicPMFundAutoCollection(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ClassicPMFundAutoCollection(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClassicFundCollectionByAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ClassicFundCollectionByAsset(context.Background(), currency.LTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAutoRepayFuturesStatusClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.ChangeAutoRepayFuturesStatusClassic(context.Background(), false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAutoRepayFuturesStatusClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetAutoRepayFuturesStatusClassic(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayFuturesNegativeBalanceClassic(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RepayFuturesNegativeBalanceClassic(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginAssetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPortfolioMarginAssetLeverage(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBLVTInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBLVTInfo(context.Background(), "BTCDOWN")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeBLVT(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SubscribeBLVT(context.Background(), "BTCUP", 0.011)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSusbcriptionRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSusbcriptionRecords(context.Background(), "BTCDOWN", time.Time{}, time.Time{}, 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemBLVT(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.RedeemBLVT(context.Background(), "BTCUSDT", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRedemptionRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetRedemptionRecord(context.Background(), "BTCDOWN", time.Time{}, time.Time{}, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBLVTUserLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetBLVTUserLimitInfo(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatDepositAndWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFiatDepositAndWithdrawalHistory(context.Background(), time.Time{}, time.Time{}, 1, 0, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatPaymentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetFiatPaymentHistory(context.Background(), time.Time{}, time.Time{}, 1, 0, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetC2CTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetC2CTradeHistory(context.Background(), "SELL", time.Time{}, time.Time{}, 0, 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPLoanOngoingOrders(context.Background(), 1232, 21231, 0, 10, currency.BTC, currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanRepay(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.VIPLoanRepay(context.Background(), 1234, 0.2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPayTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetPayTradeHistory(context.Background(), time.Now().Add(-time.Hour*480), time.Now().Add(-time.Hour*24), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllConvertPairs(t *testing.T) {
	t.Parallel()
	result, err := b.GetAllConvertPairs(context.Background(), currency.BTC, currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderQuantityPrecisionPerAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetOrderQuantityPrecisionPerAsset(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSendQuoteRequest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.SendQuoteRequest(context.Background(), currency.BTC, currency.USDT, 10, 20, "FUNDING", "1m")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcceptQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.AcceptQuote(context.Background(), "933256278426274426")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertOrderStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetConvertOrderStatus(context.Background(), "933256278426274426", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceLimitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.PlaceLimitOrder(context.Background(), &ConvertPlaceLimitOrderParam{
		BaseAsset:   currency.BTC,
		QuoteAsset:  currency.ETH,
		LimitPrice:  0.0122,
		Side:        "SELL",
		ExpiredType: "7_D",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelLimitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CancelLimitOrder(context.Background(), "123434")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLimitOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetLimitOpenOrders(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetConvertTradeHistory(context.Background(), time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotRebateHistoryRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetSpotRebateHistoryRecords(context.Background(), time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetNFTTransactionHistory(context.Background(), 1, time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetNFTDepositHistory(context.Background(), time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetNFTWithdrawalHistory(context.Background(), time.Now().Add(-time.Hour*240), time.Now().Add(-time.Hour*120), 10, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNFTAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetNFTAsset(context.Background(), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSingleTokenGiftCard(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CreateSingleTokenGiftCard(context.Background(), "BUSD", 0.1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateDualTokenGiftCard(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.CreateDualTokenGiftCard(context.Background(), currency.BUSD.String(), currency.BNB.String(), 10, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemBinanaceGiftCard(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.RedeemBinanaceGiftCard(context.Background(), "0033002328060227", "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVerifyBinanceGiftCardNumber(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.VerifyBinanceGiftCardNumber(context.Background(), "123456")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchRSAPublicKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FetchRSAPublicKey(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTokenLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.FetchTokenLimit(context.Background(), currency.BUSD.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPLoanRepaymentHistory(context.Background(), currency.ETH, time.Now().Add(-time.Hour*48), time.Now(), 1234, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVIPLoanRenew(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.VIPLoanRenew(context.Background(), 1234, 60)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCheckLockedValueVIPCollateralAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CheckLockedValueVIPCollateralAccount(context.Background(), 1223, 40)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVIPLoanBorrow(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	result, err := b.VIPLoanBorrow(context.Background(), 1234, 30, currency.ETH, currency.LTC, 123, "1234", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanableAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPLoanableAssetsData(context.Background(), currency.BTC, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPCollateralAssetData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPCollateralAssetData(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPApplicationStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPApplicationStatus(context.Background(), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPBorrowInterestRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetVIPBorrowInterestRate(context.Background(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSpotListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CreateSpotListenKey(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKeepListenKeyAlive(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.KeepSpotListenKeyAlive(context.Background(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	require.NoError(t, err)
}

func TestCloseListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.CloseSpotListenKey(context.Background(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	require.NoError(t, err)
}

func TestCreateMarginListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CreateMarginListenKey(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKeepMarginListenKeyAlive(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.KeepMarginListenKeyAlive(context.Background(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	assert.NoError(t, err)
}

func TestCloseMarginListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.CloseMarginListenKey(context.Background(), "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	assert.NoError(t, err)
}

func TestCreateCrossMarginListenKey(t *testing.T) {
	t.Parallel()
	_, err := b.CreateCrossMarginListenKey(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.CreateCrossMarginListenKey(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKeepCrossMarginListenKeyAlive(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.KeepCrossMarginListenKeyAlive(context.Background(), "BTCUSDT", "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
	assert.NoError(t, err)
}

func TestCloseCrossMarginListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.CloseCrossMarginListenKey(context.Background(), "BTCUSDT", "T3ee22BIYuWqmvne0HNq2A2WsFlEtLhvWCtItw6ffhhdmjifQ2tRbuKkTHhr")
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
	spotTradablePair = tradablePairs[0]
	tradablePairs, err = b.GetEnabledPairs(asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		usdtmTradablePair = currency.NewPair(currency.BTC, currency.USDT)
	} else {
		usdtmTradablePair = tradablePairs[0]
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
		coinmTradablePair = tradablePairs[0]
	}
	tradablePairs, err = b.GetEnabledPairs(asset.Options)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return fmt.Errorf("%w for %v", currency.ErrCurrencyPairsEmpty, asset.Options)
	}
	optionsTradablePair = tradablePairs[0]
	return nil
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, b)
	for _, a := range b.GetAssetTypes(false) {
		pairs, err := b.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := b.GetCurrencyTradeURL(context.Background(), a, pairs[0])
		require.NoError(t, err)
		require.NotEmpty(t, resp)
	}
}

func TestFetchOptionsExchangeLimits(t *testing.T) {
	t.Parallel()
	limits, err := b.FetchOptionsExchangeLimits(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, limits, "Should get some limits back")
}
