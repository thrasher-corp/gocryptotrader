package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b Binance

// this lock guards against orderbook tests race
var binanceOrderBookLock = &sync.Mutex{}

func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials()
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

func TestUServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.UServerTime()
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	spotPairs, err := b.FetchTradablePairs(asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if len(spotPairs) == 0 {
		t.Error("no tradable pairs")
	}
	spotCP, err := currency.NewPairFromString(spotPairs[0])
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateTicker(spotCP, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	tradablePairs, err := b.FetchTradablePairs(asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	cp, err := currency.NewPairFromString(tradablePairs[0])
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateTicker(cp, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	usdtMarginedPairs, err := b.FetchTradablePairs(asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	if len(usdtMarginedPairs) == 0 {
		t.Errorf("no pairs are enabled")
	}
	ucp, err := currency.NewPairFromString(usdtMarginedPairs[0])
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateTicker(ucp, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateOrderbook(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateOrderbook(cp, asset.Margin)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateOrderbook(cp, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	cp2, err := currency.NewPairFromString("BTCUSD_PERP")
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateOrderbook(cp2, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

// USDT Margined Futures

func TestUExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := b.UExchangeInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestUFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.UFuturesOrderbook(currency.Pair{Delimiter: "_", Base: currency.BTC, Quote: currency.USDT}, 1000)
	if err != nil {
		t.Error(err)
	}
}

func TestURecentTrades(t *testing.T) {
	t.Parallel()
	_, err := b.URecentTrades(currency.NewPair(currency.BTC, currency.USDT), "", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestUCompressedTrades(t *testing.T) {
	t.Parallel()
	_, err := b.UCompressedTrades(currency.NewPair(currency.BTC, currency.USDT), "", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.UCompressedTrades(currency.NewPair(currency.LTC, currency.USDT), "", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestUKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.UKlineData(currency.NewPair(currency.BTC, currency.USDT), "1d", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.UKlineData(currency.NewPair(currency.LTC, currency.USDT), "5m", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestUGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := b.UGetMarkPrice(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	_, err = b.UGetMarkPrice(currency.Pair{})
	if err != nil {
		t.Error(err)
	}
}

func TestUGetFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := b.UGetFundingHistory(currency.NewPair(currency.BTC, currency.USDT), 1, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.UGetFundingHistory(currency.NewPair(currency.LTC, currency.USDT), 1, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestU24HTickerPriceChangeStats(t *testing.T) {
	t.Parallel()
	_, err := b.U24HTickerPriceChangeStats(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	_, err = b.U24HTickerPriceChangeStats(currency.Pair{})
	if err != nil {
		t.Error(err)
	}
}

func TestUSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := b.USymbolPriceTicker(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	_, err = b.USymbolPriceTicker(currency.Pair{})
	if err != nil {
		t.Error(err)
	}
}

func TestUSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := b.USymbolOrderbookTicker(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	_, err = b.USymbolOrderbookTicker(currency.Pair{})
	if err != nil {
		t.Error(err)
	}
}

func TestULiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := b.ULiquidationOrders(currency.NewPair(currency.BTC, currency.USDT), 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.ULiquidationOrders(currency.NewPair(currency.LTC, currency.USDT), 5, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestUOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := b.UOpenInterest(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUOpenInterestStats(t *testing.T) {
	t.Parallel()
	_, err := b.UOpenInterestStats(currency.NewPair(currency.BTC, currency.USDT), "5m", 1, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.UOpenInterestStats(currency.NewPair(currency.LTC, currency.USDT), "1d", 10, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestUTopAcccountsLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := b.UTopAcccountsLongShortRatio(currency.NewPair(currency.BTC, currency.USDT), "5m", 2, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.UTopAcccountsLongShortRatio(currency.NewPair(currency.BTC, currency.USDT), "5m", 2, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestUTopPostionsLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := b.UTopPostionsLongShortRatio(currency.NewPair(currency.BTC, currency.USDT), "5m", 3, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.UTopPostionsLongShortRatio(currency.NewPair(currency.BTC, currency.USDT), "1d", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestUGlobalLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := b.UGlobalLongShortRatio(currency.NewPair(currency.BTC, currency.USDT), "5m", 3, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.UGlobalLongShortRatio(currency.NewPair(currency.BTC, currency.USDT), "4h", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestUTakerBuySellVol(t *testing.T) {
	t.Parallel()
	_, err := b.UTakerBuySellVol(currency.NewPair(currency.BTC, currency.USDT), "5m", 10, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestUCompositeIndexInfo(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("DEFI-USDT")
	if err != nil {
		t.Error(err)
	}
	_, err = b.UCompositeIndexInfo(cp)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UCompositeIndexInfo(currency.Pair{})
	if err != nil {
		t.Error(err)
	}
}

func TestUFuturesNewOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.UFuturesNewOrder(currency.NewPair(currency.BTC, currency.USDT), "BUY", "", "LIMIT", "GTC", "", "", "", "", 1, 1, 0, 0, 0, false)
	if err != nil {
		t.Error(err)
	}
}

func TestUPlaceBatchOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	var data []PlaceBatchOrderData
	var tempData PlaceBatchOrderData
	tempData.Symbol = "BTCUSDT"
	tempData.Side = "BUY"
	tempData.OrderType = "LIMIT"
	tempData.Quantity = 4
	tempData.Price = 1
	tempData.TimeInForce = "GTC"
	data = append(data, tempData)
	_, err := b.UPlaceBatchOrders(data)
	if err != nil {
		t.Error(err)
	}
}

func TestUGetOrderData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UGetOrderData(currency.NewPair(currency.BTC, currency.USDT), "123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestUCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.UCancelOrder(currency.NewPair(currency.BTC, currency.USDT), "123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestUCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.UCancelAllOpenOrders(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUCancelBatchOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.UCancelBatchOrders(currency.NewPair(currency.BTC, currency.USDT), []string{"123"}, []string{})
	if err != nil {
		t.Error(err)
	}
}

func TestUAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.UAutoCancelAllOpenOrders(currency.NewPair(currency.BTC, currency.USDT), 30)
	if err != nil {
		t.Error(err)
	}
}

func TestUFetchOpenOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UFetchOpenOrder(currency.NewPair(currency.BTC, currency.USDT), "123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestUAllAccountOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UAllAccountOpenOrders(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUAllAccountOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UAllAccountOrders(currency.Pair{}, 0, 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.UAllAccountOrders(currency.NewPair(currency.BTC, currency.USDT), 0, 5, time.Now().Add(-time.Hour*4), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestUAccountBalanceV2(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UAccountBalanceV2()
	if err != nil {
		t.Error(err)
	}
}

func TestUAccountInformationV2(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UAccountInformationV2()
	if err != nil {
		t.Error(err)
	}
}

func TestUChangeInitialLeverageRequest(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.UChangeInitialLeverageRequest(currency.NewPair(currency.BTC, currency.USDT), 2)
	if err != nil {
		t.Error(err)
	}
}

func TestUChangeInitialMarginType(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	err := b.UChangeInitialMarginType(currency.NewPair(currency.BTC, currency.USDT), "ISOLATED")
	if err != nil {
		t.Error(err)
	}
}

func TestUModifyIsolatedPositionMarginReq(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.UModifyIsolatedPositionMarginReq(currency.NewPair(currency.BTC, currency.USDT), "LONG", "add", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestUPositionMarginChangeHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UPositionMarginChangeHistory(currency.NewPair(currency.BTC, currency.USDT), "add", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestUPositionsInfoV2(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UPositionsInfoV2(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUAccountTradesHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UAccountTradesHistory(currency.NewPair(currency.BTC, currency.USDT), "", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestUAccountIncomeHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UAccountIncomeHistory(currency.Pair{}, "", 5, time.Now().Add(-time.Hour*48), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestUGetNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UGetNotionalAndLeverageBrackets(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UPositionsADLEstimate(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUAccountForcedOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.UAccountForcedOrders(currency.NewPair(currency.BTC, currency.USDT), "ADL", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

// Coin Margined Futures

func TestGetFuturesExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := b.FuturesExchangeInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestGetInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetInterestHistory()
	if err != nil {
		t.Error(err)
	}
}

func TestGetCrossMarginInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetCrossMarginInterestHistory()
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRates(t *testing.T) {
	t.Parallel()
	_, err := b.GetFundingRates(currency.NewPair(currency.BTC, currency.USDT), "", time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFundingRates(currency.NewPair(currency.BTC, currency.USDT), "2", time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesOrderbook(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 1000)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesPublicTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesPublicTrades(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPastPublicTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetPastPublicTrades(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAggregatedTradesList(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesAggregatedTradesList(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 0, 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetPerpsExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetPerpMarkets()
	if err != nil {
		t.Error(err)
	}
}

func TestGetIndexAndMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexAndMarkPrice("", "BTCUSD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesKlineData(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetFuturesKlineData(currency.NewPairWithDelimiter("LTCUSD", "PERP", "_"), "5m", 5, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetContinuousKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.GetContinuousKlineData("BTCUSD", "CURRENT_QUARTER", "1M", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetContinuousKlineData("BTCUSD", "CURRENT_QUARTER", "1M", 5, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetIndexPriceKlines(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexPriceKlines("BTCUSD", "1M", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetIndexPriceKlines("BTCUSD", "1M", 5, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesSwapTickerChangeStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesSwapTickerChangeStats(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesSwapTickerChangeStats(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesSwapTickerChangeStats(currency.Pair{}, "")
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesGetFundingHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys")
	}
	_, err := b.FuturesGetFundingHistory(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.FuturesGetFundingHistory(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 50, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.GetFuturesHistoricalTrades(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 5)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesHistoricalTrades(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesSymbolPriceTicker(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesOrderbookTicker(currency.Pair{}, "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesOrderbookTicker(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesLiquidationOrders(currency.Pair{}, "", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesLiquidationOrders(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenInterest(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"))
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenInterestStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenInterestStats("BTCUSD", "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOpenInterestStats("BTCUSD", "CURRENT_QUARTER", "5m", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetTraderFuturesAccountRatio(t *testing.T) {
	t.Parallel()
	_, err := b.GetTraderFuturesAccountRatio("BTCUSD", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTraderFuturesAccountRatio("BTCUSD", "5m", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetTraderFuturesPositionsRatio(t *testing.T) {
	t.Parallel()
	_, err := b.GetTraderFuturesPositionsRatio("BTCUSD", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTraderFuturesPositionsRatio("BTCUSD", "5m", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketRatio(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketRatio("BTCUSD", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetMarketRatio("BTCUSD", "5m", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesTakerVolume(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesTakerVolume("BTCUSD", "ALL", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesTakerVolume("BTCUSD", "ALL", "5m", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesBasisData(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesBasisData("BTCUSD", "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesBasisData("BTCUSD", "CURRENT_QUARTER", "5m", 0, time.Unix(1577836800, 0), time.Unix(1580515200, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesNewOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.FuturesNewOrder(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "BUY", "", "LIMIT", "GTC", "", "", "", "", 1, 1, 0, 0, 0, false)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesBatchOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	var data []PlaceBatchOrderData
	var tempData PlaceBatchOrderData
	tempData.Symbol = "BTCUSD_PERP"
	tempData.Side = "BUY"
	tempData.OrderType = "LIMIT"
	tempData.Quantity = 1
	tempData.Price = 1
	tempData.TimeInForce = "GTC"

	data = append(data, tempData)
	_, err := b.FuturesBatchOrder(data)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesBatchCancelOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.FuturesBatchCancelOrders(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), []string{"123"}, []string{})
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesGetOrderData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.FuturesGetOrderData(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.FuturesCancelAllOpenOrders(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"))
	if err != nil {
		t.Error(err)
	}
}

func TestAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.AutoCancelAllOpenOrders(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 30000)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesOpenOrderData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.FuturesOpenOrderData(currency.NewPair(currency.BTC, currency.USDT), "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesAllOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.GetFuturesAllOpenOrders(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.GetAllFuturesOrders(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", time.Time{}, time.Time{}, 0, 2)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesChangeMarginType(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.FuturesChangeMarginType(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "ISOLATED")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesAccountBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.GetFuturesAccountBalance()
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.GetFuturesAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesChangeInitialLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.FuturesChangeInitialLeverage(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyIsolatedPositionMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := b.ModifyIsolatedPositionMargin(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "BOTH", "add", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesMarginChangeHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.FuturesMarginChangeHistory(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "add", time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesPositionsInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.FuturesPositionsInfo("BTCUSD_PERP", "")
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.FuturesTradeHistory(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", time.Time{}, time.Time{}, 5, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesIncomeHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.FuturesIncomeHistory(currency.Pair{}, "TRANSFER", time.Time{}, time.Time{}, 5)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesForceOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.FuturesForceOrders(currency.Pair{}, "ADL", time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestUGetNotionalLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.FuturesNotionalBracket("BTCUSD")
	if err != nil {
		t.Error(err)
	}
	_, err = b.FuturesNotionalBracket("")
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := b.FuturesPositionsADLEstimate(currency.Pair{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkPriceKline(currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginMarkets()
	if err != nil {
		t.Error(err)
	}
}

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	info, err := b.GetExchangeInfo()
	if err != nil {
		t.Error(err)
	}
	if mockTests {
		serverTime := time.Date(2021, 1, 27, 2, 43, 18, int(593*time.Millisecond), time.UTC)
		if !info.Servertime.Equal(serverTime) {
			t.Errorf("Expected %v, got %v", serverTime, info.Servertime)
		}
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := b.FetchTradablePairs(asset.Spot)
	if err != nil {
		t.Error("Binance FetchTradablePairs(asset asets.AssetType) error", err)
	}

	_, err = b.FetchTradablePairs(asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.FetchTradablePairs(asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: currency.NewPair(currency.BTC, currency.USDT),
		Limit:  1000,
	})

	if err != nil {
		t.Error("Binance GetOrderBook() error", err)
	}
}

func TestGetMostRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetMostRecentTrades(RecentTradeRequestParams{
		Symbol: currency.NewPair(currency.BTC, currency.USDT),
		Limit:  15,
	})

	if err != nil {
		t.Error("Binance GetMostRecentTrades() error", err)
	}
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetHistoricalTrades("BTCUSDT", 5, -1)
	if err != nil {
		t.Errorf("Binance GetHistoricalTrades() error: %v", err)
	}
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetAggregatedTrades(&AggregatedTradeRequestParams{
		Symbol: currency.NewPair(currency.BTC, currency.USDT),
		Limit:  5,
	})
	if err != nil {
		t.Error("Binance GetAggregatedTrades() error", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := b.GetSpotKline(&KlinesRequestParams{
		Symbol:    currency.NewPair(currency.BTC, currency.USDT),
		Interval:  kline.FiveMin.Short(),
		Limit:     24,
		StartTime: time.Unix(1577836800, 0),
		EndTime:   time.Unix(1580515200, 0),
	})
	if err != nil {
		t.Error("Binance GetSpotKline() error", err)
	}
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetAveragePrice(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetAveragePrice() error", err)
	}
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()

	_, err := b.GetPriceChangeStats(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetPriceChangeStats() error", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()

	_, err := b.GetTickers()
	if err != nil {
		t.Error("Binance TestGetTickers error", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetLatestSpotPrice(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetLatestSpotPrice() error", err)
	}
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetBestPrice(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetBestPrice() error", err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()

	_, err := b.QueryOrder(currency.NewPair(currency.BTC, currency.USDT), "", 1337)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("QueryOrder() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("QueryOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock QueryOrder() error", err)
	}
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := b.OpenOrders(currency.Pair{})
	if err != nil {
		t.Error(err)
	}

	p := currency.NewPair(currency.BTC, currency.USDT)
	_, err = b.OpenOrders(p)
	if err != nil {
		t.Error(err)
	}
}

func TestAllOrders(t *testing.T) {
	t.Parallel()

	_, err := b.AllOrders(currency.NewPair(currency.BTC, currency.USDT), "", "")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("AllOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("AllOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock AllOrders() error", err)
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()

	var feeBuilder = setFeeBuilder()
	b.GetFeeByType(feeBuilder)
	if !areTestAPIKeysSet() || mockTests {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()

	var feeBuilder = setFeeBuilder()

	if areTestAPIKeysSet() && mockTests {
		// CryptocurrencyTradeFee Basic
		if _, err := b.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := b.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := b.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := b.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()

	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		Pairs:     currency.Pairs{pair},
		AssetType: asset.Spot,
	}

	_, err = b.GetActiveOrders(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetActiveOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetActiveOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetActiveOrders() error", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()

	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := b.GetOrderHistory(&getOrdersRequest)
	if err == nil {
		t.Error("Expected: 'At least one currency is required to fetch order history'. received nil")
	}

	getOrdersRequest.Pairs = []currency.Pair{
		currency.NewPair(currency.LTC,
			currency.BTC)}

	_, err = b.GetOrderHistory(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetOrderHistory() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetOrderHistory() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetOrderHistory() error", err)
	}
}

func TestNewOrderTest(t *testing.T) {
	t.Parallel()

	req := &NewOrderRequest{
		Symbol:      currency.NewPair(currency.LTC, currency.BTC),
		Side:        order.Buy.String(),
		TradeType:   BinanceRequestParamsOrderLimit,
		Price:       0.0025,
		Quantity:    100000,
		TimeInForce: BinanceRequestParamsTimeGTC,
	}

	err := b.NewOrderTest(req)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("NewOrderTest() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("NewOrderTest() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock NewOrderTest() error", err)
	}

	req = &NewOrderRequest{
		Symbol:        currency.NewPair(currency.LTC, currency.BTC),
		Side:          order.Sell.String(),
		TradeType:     BinanceRequestParamsOrderMarket,
		Price:         0.0045,
		QuoteOrderQty: 10,
	}

	err = b.NewOrderTest(req)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("NewOrderTest() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("NewOrderTest() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock NewOrderTest() error", err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	start, err := time.Parse(time.RFC3339, "2020-01-02T15:04:05Z")
	if err != nil {
		t.Fatal(err)
	}
	result, err := b.GetHistoricTrades(currencyPair, asset.Spot, start, start.Add(15*time.Minute))
	if err != nil {
		t.Error(err)
	}
	var expected int
	if mockTests {
		expected = 5
	} else {
		expected = 2134
	}
	if len(result) != expected {
		t.Errorf("GetHistoricTrades() expected %v entries, got %v", expected, len(result))
	}
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
	mockExpectTime, err := time.Parse(time.RFC3339, "2020-01-02T16:19:04.8Z")
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
				Symbol:    currencyPair,
				StartTime: start,
				EndTime:   start.Add(75 * time.Minute),
			},
			numExpected:  3,
			lastExpected: mockExpectTime,
		},
		{
			name: "batch with timerange",
			args: &AggregatedTradeRequestParams{
				Symbol:    currencyPair,
				StartTime: start,
				EndTime:   start.Add(75 * time.Minute),
			},
			numExpected:  4303,
			lastExpected: expectTime,
		},
		{
			name: "mock custom limit with start time set, no end time",
			mock: true,
			args: &AggregatedTradeRequestParams{
				Symbol:    currency.NewPair(currency.BTC, currency.USDT),
				StartTime: start,
				Limit:     1001,
			},
			numExpected:  4,
			lastExpected: time.Date(2020, 1, 2, 16, 19, 5, int(200*time.Millisecond), time.UTC),
		},
		{
			name: "custom limit with start time set, no end time",
			args: &AggregatedTradeRequestParams{
				Symbol:    currency.NewPair(currency.BTC, currency.USDT),
				StartTime: time.Date(2020, 11, 18, 12, 0, 0, 0, time.UTC),
				Limit:     1001,
			},
			numExpected:  1001,
			lastExpected: time.Date(2020, 11, 18, 13, 0, 0, int(34*time.Millisecond), time.UTC),
		},
		{
			name: "mock recent trades",
			mock: true,
			args: &AggregatedTradeRequestParams{
				Symbol: currency.NewPair(currency.BTC, currency.USDT),
				Limit:  3,
			},
			numExpected:  3,
			lastExpected: time.Date(2020, 1, 2, 16, 19, 5, int(200*time.Millisecond), time.UTC),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.mock != mockTests {
				t.Skip()
			}
			result, err := b.GetAggregatedTrades(tt.args)
			if err != nil {
				t.Error(err)
			}
			if len(result) != tt.numExpected {
				t.Errorf("GetAggregatedTradesBatched() expected %v entries, got %v", tt.numExpected, len(result))
			}
			lastTradeTime := result[len(result)-1].TimeStamp
			if !lastTradeTime.Equal(tt.lastExpected) {
				t.Errorf("last trade expected %v, got %v", tt.lastExpected, lastTradeTime)
			}
		})
	}
}

func TestGetAggregatedTradesErrors(t *testing.T) {
	t.Parallel()
	start, err := time.Parse(time.RFC3339, "2020-01-02T15:04:05Z")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		args *AggregatedTradeRequestParams
	}{
		{
			name: "get recent trades does not support custom limit",
			args: &AggregatedTradeRequestParams{
				Symbol: currency.NewPair(currency.BTC, currency.USDT),
				Limit:  1001,
			},
		},
		{
			name: "start time and fromId cannot be both set",
			args: &AggregatedTradeRequestParams{
				Symbol:    currency.NewPair(currency.BTC, currency.USDT),
				StartTime: start,
				EndTime:   start.Add(75 * time.Minute),
				FromID:    2,
			},
		},
		{
			name: "can't get most recent 5000 (more than 1000 not allowed)",
			args: &AggregatedTradeRequestParams{
				Symbol: currency.NewPair(currency.BTC, currency.USDT),
				Limit:  5000,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := b.GetAggregatedTrades(tt.args)
			if err == nil {
				t.Errorf("Binance.GetAggregatedTrades() error = %v, wantErr true", err)
				return
			}
		})
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// -----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
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
	}

	_, err := b.SubmitOrder(orderSubmission)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("SubmitOrder() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("SubmitOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock SubmitOrder() error", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}

	err := b.CancelOrder(orderCancellation)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("CancelExchangeOrder() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("CancelExchangeOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock CancelExchangeOrder() error", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}

	_, err := b.CancelAllOrders(orderCancellation)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("CancelAllExchangeOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("CancelAllExchangeOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock CancelAllExchangeOrders() error", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	t.Parallel()
	items := asset.Items{
		asset.CoinMarginedFutures,
		asset.USDTMarginedFutures,
		asset.Spot,
		asset.Margin,
	}
	for i := range items {
		assetType := items[i]
		t.Run(fmt.Sprintf("Update info of account [%s]", assetType.String()), func(t *testing.T) {
			_, err := b.UpdateAccountInfo(assetType)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestWrapperGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	p, err := currency.NewPairFromString("EOS-USDT")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetActiveOrders(&order.GetOrdersRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{p},
		AssetType: asset.CoinMarginedFutures,
	})
	if err != nil {
		t.Error(err)
	}

	p2, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetActiveOrders(&order.GetOrdersRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{p2},
		AssetType: asset.USDTMarginedFutures,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestWrapperGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	p, err := currency.NewPairFromString("EOSUSD_PERP")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrderHistory(&order.GetOrdersRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		OrderID:   "123",
		Pairs:     currency.Pairs{p},
		AssetType: asset.CoinMarginedFutures,
	})
	if err != nil {
		t.Error(err)
	}

	p2, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrderHistory(&order.GetOrdersRequest{
		Type:      order.AnyType,
		Side:      order.AnySide,
		OrderID:   "123",
		Pairs:     currency.Pairs{p2},
		AssetType: asset.USDTMarginedFutures,
	})
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderHistory(&order.GetOrdersRequest{
		AssetType: asset.USDTMarginedFutures,
	})
	if err == nil {
		t.Errorf("expecting an error since invalid param combination is given. Got err: %v", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	p, err := currency.NewPairFromString("EOS-USDT")
	if err != nil {
		t.Error(err)
	}
	fpair, err := b.FormatExchangeCurrency(p, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	err = b.CancelOrder(&order.Cancel{
		AssetType: asset.CoinMarginedFutures,
		Pair:      fpair,
		ID:        "1234",
	})
	if err != nil {
		t.Error(err)
	}

	p2, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Error(err)
	}
	fpair2, err := b.FormatExchangeCurrency(p2, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	err = b.CancelOrder(&order.Cancel{
		AssetType: asset.USDTMarginedFutures,
		Pair:      fpair2,
		ID:        "1234",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	tradablePairs, err := b.FetchTradablePairs(asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	cp, err := currency.NewPairFromString(tradablePairs[0])
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrderInfo("123", cp, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := b.ModifyOrder(&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() error cannot be nil")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    b.Name,
		Amount:      0.00001337,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := b.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Withdraw() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Withdraw() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error(err)
	}
}

func TestWithdrawHistory(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.GetWithdrawalsHistory(currency.XBT)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetWithdrawalsHistory() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetWithdrawalsHistory() expecting an error when no keys are set")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawFiatFunds(&withdraw.Request{})
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawFiatFundsToInternationalBank(&withdraw.Request{})
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := b.GetDepositAddress(currency.BTC, "")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetDepositAddress() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetDepositAddress() error cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock GetDepositAddress() error", err)
	}
}

func TestWSSubscriptionHandling(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{
  "method": "SUBSCRIBE",
  "params": [
    "btcusdt@aggTrade",
    "btcusdt@depth"
  ],
  "id": 1
}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWSUnsubscriptionHandling(t *testing.T) {
	pressXToJSON := []byte(`{
  "method": "UNSUBSCRIBE",
  "params": [
    "btcusdt@depth"
  ],
  "id": 312
}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTickerUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"btcusdt@ticker","data":{"e":"24hrTicker","E":1580254809477,"s":"BTCUSDT","p":"420.97000000","P":"4.720","w":"9058.27981278","x":"8917.98000000","c":"9338.96000000","Q":"0.17246300","b":"9338.03000000","B":"0.18234600","a":"9339.70000000","A":"0.14097600","o":"8917.99000000","h":"9373.19000000","l":"8862.40000000","v":"72229.53692000","q":"654275356.16896672","O":1580168409456,"C":1580254809456,"F":235294268,"L":235894703,"n":600436}}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsKlineUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"btcusdt@kline_1m","data":{
	  "e": "kline",     
	  "E": 123456789,   
	  "s": "BNBBTC",    
	  "k": {
		"t": 123400000, 
		"T": 123460000, 
		"s": "BNBBTC",  
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
	if err != nil {
		t.Error(err)
	}
}

func TestWsTradeUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"btcusdt@trade","data":{
	  "e": "trade",     
	  "E": 123456789,   
	  "s": "BNBBTC",    
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
	if err != nil {
		t.Error(err)
	}
}

func TestWsDepthUpdate(t *testing.T) {
	binanceOrderBookLock.Lock()
	defer binanceOrderBookLock.Unlock()
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
	if err := b.SeedLocalCacheWithBook(p, &book); err != nil {
		t.Error(err)
	}

	if err := b.wsHandleData(update1); err != nil {
		t.Error(err)
	}

	ob, err := b.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if exp, got := seedLastUpdateID, ob.LastUpdateID; got != exp {
		t.Fatalf("Unexpected Last update id of orderbook for old update. Exp: %d, got: %d", exp, got)
	}
	if exp, got := 2.3, ob.Asks[2].Amount; got != exp {
		t.Fatalf("Ask altered by outdated update. Exp: %f, got %f", exp, got)
	}
	if exp, got := 0.163526, ob.Bids[1].Amount; got != exp {
		t.Fatalf("Bid altered by outdated update. Exp: %f, got %f", exp, got)
	}

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

	if err = b.wsHandleData(update2); err != nil {
		t.Error(err)
	}

	ob, err = b.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if exp, got := int64(165), ob.LastUpdateID; got != exp {
		t.Fatalf("Unexpected Last update id of orderbook for new update. Exp: %d, got: %d", exp, got)
	}
	if exp, got := 2.3, ob.Asks[2].Amount; got != exp {
		t.Fatalf("Unexpected Ask amount. Exp: %f, got %f", exp, got)
	}
	if exp, got := 1.9, ob.Asks[3].Amount; got != exp {
		t.Fatalf("Unexpected Ask amount. Exp: %f, got %f", exp, got)
	}
	if exp, got := 0.163526, ob.Bids[1].Amount; got != exp {
		t.Fatalf("Unexpected Bid amount. Exp: %f, got %f", exp, got)
	}

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
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWsAuthStreamKey(t *testing.T) {
	key, err := b.GetWsAuthStreamKey()
	switch {
	case mockTests && err != nil,
		!mockTests && areTestAPIKeysSet() && err != nil:
		t.Fatal(err)
	case !mockTests && !areTestAPIKeysSet() && err == nil:
		t.Fatal("Expected error")
	}

	if key == "" && (areTestAPIKeysSet() || mockTests) {
		t.Error("Expected key")
	}
}

func TestMaintainWsAuthStreamKey(t *testing.T) {
	err := b.MaintainWsAuthStreamKey()
	switch {
	case mockTests && err != nil,
		!mockTests && areTestAPIKeysSet() && err != nil:
		t.Fatal(err)
	case !mockTests && !areTestAPIKeysSet() && err == nil:
		t.Fatal("Expected error")
	}
}

func TestExecutionTypeToOrderStatus(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []TestCases{
		{Case: "NEW", Result: order.New},
		{Case: "CANCELED", Result: order.Cancelled},
		{Case: "REJECTED", Result: order.Rejected},
		{Case: "TRADE", Result: order.PartiallyFilled},
		{Case: "EXPIRED", Result: order.Expired},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := stringToOrderStatus(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Exepcted: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestGetHistoricCandles(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Unix(1546300800, 0)
	end := time.Unix(1577836799, 0)
	_, err = b.GetHistoricCandles(currencyPair, asset.Spot, startTime, end, kline.OneDay)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetHistoricCandles(currencyPair, asset.Spot, startTime, end, kline.Interval(time.Hour*7))
	if err == nil {
		t.Fatal("unexpected result")
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}

	startTime := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2021, 2, 15, 0, 0, 0, 0, time.UTC)
	_, err = b.GetHistoricCandlesExtended(currencyPair, asset.Spot, startTime, end, kline.OneDay)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetHistoricCandlesExtended(currencyPair, asset.Spot, startTime, end, kline.Interval(time.Hour*7))
	if err == nil {
		t.Error("unexpected result")
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

			if ret != test.output {
				t.Fatalf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetRecentTrades(currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestSeedLocalCache(t *testing.T) {
	t.Parallel()
	err := b.SeedLocalCache(currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	subs, err := b.GenerateSubscriptions()
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 4 {
		t.Fatal("unexpected subscription length")
	}
}

var websocketDepthUpdate = []byte(`{"E":1608001030784,"U":7145637266,"a":[["19455.19000000","0.59490200"],["19455.37000000","0.00000000"],["19456.11000000","0.00000000"],["19456.16000000","0.00000000"],["19458.67000000","0.06400000"],["19460.73000000","0.05139800"],["19461.43000000","0.00000000"],["19464.59000000","0.00000000"],["19466.03000000","0.45000000"],["19466.36000000","0.00000000"],["19508.67000000","0.00000000"],["19572.96000000","0.00217200"],["24386.00000000","0.00256600"]],"b":[["19455.18000000","2.94649200"],["19453.15000000","0.01233600"],["19451.18000000","0.00000000"],["19446.85000000","0.11427900"],["19446.74000000","0.00000000"],["19446.73000000","0.00000000"],["19444.45000000","0.14937800"],["19426.75000000","0.00000000"],["19416.36000000","0.36052100"]],"e":"depthUpdate","s":"BTCUSDT","u":7145637297}`)

func TestProcessUpdate(t *testing.T) {
	t.Parallel()
	binanceOrderBookLock.Lock()
	defer binanceOrderBookLock.Unlock()
	p := currency.NewPair(currency.BTC, currency.USDT)
	var depth WebsocketDepthStream
	err := json.Unmarshal(websocketDepthUpdate, &depth)
	if err != nil {
		t.Fatal(err)
	}

	err = b.obm.stageWsUpdate(&depth, p, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	err = b.obm.fetchBookViaREST(p)
	if err != nil {
		t.Fatal(err)
	}

	err = b.obm.cleanup(p)
	if err != nil {
		t.Fatal(err)
	}

	// reset order book sync status
	b.obm.state[currency.BTC][currency.USDT][asset.Spot].lastUpdateID = 0
}

func TestUFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	cp, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Error(err)
	}
	_, err = b.UFuturesHistoricalTrades(cp, "", 5)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UFuturesHistoricalTrades(cp, "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetExchangeOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := b.UpdateOrderExecutionLimits(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	err = b.UpdateOrderExecutionLimits(asset.CoinMarginedFutures)
	if err != nil {
		t.Fatal(err)
	}

	err = b.UpdateOrderExecutionLimits(asset.USDTMarginedFutures)
	if err != nil {
		t.Fatal(err)
	}

	err = b.UpdateOrderExecutionLimits(asset.Binary)
	if err == nil {
		t.Fatal("expected unhandled case")
	}

	cmfCP, err := currency.NewPairFromStrings("BTCUSD", "PERP")
	if err != nil {
		t.Fatal(err)
	}

	limit, err := b.GetOrderExecutionLimits(asset.CoinMarginedFutures, cmfCP)
	if err != nil {
		t.Fatal(err)
	}

	if limit == nil {
		t.Fatal("exchange limit should be loaded")
	}

	err = limit.Conforms(0.000001, 0.1, order.Limit)
	if !errors.Is(err, order.ErrAmountBelowMin) {
		t.Fatalf("expected %v, but received %v", order.ErrAmountBelowMin, err)
	}

	err = limit.Conforms(0.01, 1, order.Limit)
	if !errors.Is(err, order.ErrPriceBelowMin) {
		t.Fatalf("expected %v, but received %v", order.ErrPriceBelowMin, err)
	}
}

func TestWsOrderExecutionReport(t *testing.T) {
	// cannot run in parallel due to inspecting the DataHandler result
	payload := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"executionReport","E":1616627567900,"s":"BTCUSDT","c":"c4wyKsIhoAaittTYlIVLqk","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00028400","p":"52789.10000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":5340845958,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1616627567900,"t":-1,"I":11388173160,"w":true,"m":false,"M":false,"O":1616627567900,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}}`)
	expRes := order.Detail{
		Price:           52789.1,
		Amount:          0.00028400,
		Exchange:        "Binance",
		ID:              "5340845958",
		ClientOrderID:   "c4wyKsIhoAaittTYlIVLqk",
		Side:            order.Buy,
		Type:            order.Limit,
		Status:          order.New,
		AssetType:       asset.Spot,
		Pair:            currency.NewPair(currency.BTC, currency.USDT),
		RemainingAmount: 0.000284,
		Date:            time.Unix(0, 1616627567900*int64(time.Millisecond)),
	}
	// empty the channel. otherwise mock_test will fail
	for len(b.Websocket.DataHandler) > 0 {
		<-b.Websocket.DataHandler
	}

	err := b.wsHandleData(payload)
	if err != nil {
		t.Fatal(err)
	}
	res := <-b.Websocket.DataHandler
	switch r := res.(type) {
	case *order.Detail:
		if !reflect.DeepEqual(expRes, *r) {
			t.Errorf("Results do not match:\nexpected: %v\nreceived: %v", expRes, *r)
		}
	default:
		t.Fatalf("expected type order.Detail, found %T", res)
	}

	payload = []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"executionReport","E":1616633041556,"s":"BTCUSDT","c":"YeULctvPAnHj5HXCQo9Mob","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00028600","p":"52436.85000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":5341783271,"l":"0.00028600","z":"0.00028600","L":"52436.85000000","n":"0.00000029","N":"BTC","T":1616633041555,"t":726946523,"I":11390206312,"w":false,"m":false,"M":true,"O":1616633041555,"Z":"14.99693910","Y":"14.99693910","Q":"0.00000000"}}`)
	err = b.wsHandleData(payload)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsOutboundAccountPosition(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"outboundAccountPosition","E":1616628815745,"u":1616628815745,"B":[{"a":"BTC","f":"0.00225109","l":"0.00123000"},{"a":"BNB","f":"0.00000000","l":"0.00000000"},{"a":"USDT","f":"54.43390661","l":"0.00000000"}]}}`)
	err := b.wsHandleData(payload)
	if err != nil {
		t.Fatal(err)
	}
}
