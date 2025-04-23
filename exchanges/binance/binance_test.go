package binance

import (
	"bytes"
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
)

var (
	b = &Binance{}
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
	_, err := b.UServerTime(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetServerTime(t.Context(), asset.Empty)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	st, err := b.GetServerTime(t.Context(), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if st.IsZero() {
		t.Fatal("expected a time")
	}

	st, err = b.GetServerTime(t.Context(), asset.USDTMarginedFutures)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if st.IsZero() {
		t.Fatal("expected a time")
	}

	st, err = b.GetServerTime(t.Context(), asset.CoinMarginedFutures)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if st.IsZero() {
		t.Fatal("expected a time")
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	r, err := b.UpdateTicker(t.Context(), testPairMapping, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if r.Pair.Base != currency.DOGE && r.Pair.Quote != currency.USDT {
		t.Error("invalid pair values")
	}
	tradablePairs, err := b.FetchTradablePairs(t.Context(), asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	_, err = b.UpdateTicker(t.Context(), tradablePairs[0], asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	usdtMarginedPairs, err := b.FetchTradablePairs(t.Context(), asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	if len(usdtMarginedPairs) == 0 {
		t.Errorf("no pairs are enabled")
	}
	_, err = b.UpdateTicker(t.Context(), usdtMarginedPairs[0], asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	err := b.UpdateTickers(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}

	err = b.UpdateTickers(t.Context(), asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	err = b.UpdateTickers(t.Context(), asset.USDTMarginedFutures)
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
	_, err = b.UpdateOrderbook(t.Context(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateOrderbook(t.Context(), cp, asset.Margin)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateOrderbook(t.Context(), cp, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	cp2, err := currency.NewPairFromString("BTCUSD_PERP")
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateOrderbook(t.Context(), cp2, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

// USDT Margined Futures

func TestUExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := b.UExchangeInfo(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestUFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.UFuturesOrderbook(t.Context(), currency.NewPair(currency.BTC, currency.USDT), 1000)
	if err != nil {
		t.Error(err)
	}
}

func TestURecentTrades(t *testing.T) {
	t.Parallel()
	_, err := b.URecentTrades(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", 1000)
	if err != nil {
		t.Error(err)
	}
}

func TestUCompressedTrades(t *testing.T) {
	t.Parallel()
	_, err := b.UCompressedTrades(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.UCompressedTrades(t.Context(), currency.NewPair(currency.LTC, currency.USDT), "", 0, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestUKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.UKlineData(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "1d", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.UKlineData(t.Context(), currency.NewPair(currency.LTC, currency.USDT), "5m", 0, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestUGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := b.UGetMarkPrice(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	_, err = b.UGetMarkPrice(t.Context(), currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestUGetFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := b.UGetFundingHistory(t.Context(), currency.NewPair(currency.BTC, currency.USDT), 1, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.UGetFundingHistory(t.Context(), currency.NewPair(currency.LTC, currency.USDT), 1, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestU24HTickerPriceChangeStats(t *testing.T) {
	t.Parallel()
	_, err := b.U24HTickerPriceChangeStats(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	_, err = b.U24HTickerPriceChangeStats(t.Context(), currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestUSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := b.USymbolPriceTicker(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	_, err = b.USymbolPriceTicker(t.Context(), currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestUSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := b.USymbolOrderbookTicker(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	_, err = b.USymbolOrderbookTicker(t.Context(), currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestUOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := b.UOpenInterest(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUOpenInterestStats(t *testing.T) {
	t.Parallel()
	_, err := b.UOpenInterestStats(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "5m", 1, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.UOpenInterestStats(t.Context(), currency.NewPair(currency.LTC, currency.USDT), "1d", 10, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestUTopAcccountsLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := b.UTopAcccountsLongShortRatio(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "5m", 2, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.UTopAcccountsLongShortRatio(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "5m", 2, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestUTopPostionsLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := b.UTopPostionsLongShortRatio(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "5m", 3, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.UTopPostionsLongShortRatio(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "1d", 0, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestUGlobalLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := b.UGlobalLongShortRatio(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "5m", 3, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.UGlobalLongShortRatio(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "4h", 0, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestUTakerBuySellVol(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	_, err := b.UTakerBuySellVol(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "5m", 10, start, end)
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
	_, err = b.UCompositeIndexInfo(t.Context(), cp)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UCompositeIndexInfo(t.Context(), currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestUFuturesNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.UFuturesNewOrder(t.Context(),
		&UFuturesNewOrderRequest{
			Symbol:      currency.NewPair(currency.BTC, currency.USDT),
			Side:        "BUY",
			OrderType:   "LIMIT",
			TimeInForce: "GTC",
			Quantity:    1,
			Price:       1,
		},
	)
	if err != nil {
		t.Error(err)
	}
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
	_, err := b.UPlaceBatchOrders(t.Context(), data)
	if err != nil {
		t.Error(err)
	}
}

func TestUGetOrderData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UGetOrderData(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestUCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.UCancelOrder(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestUCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.UCancelAllOpenOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.UCancelBatchOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT), []string{"123"}, []string{})
	if err != nil {
		t.Error(err)
	}
}

func TestUAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.UAutoCancelAllOpenOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT), 30)
	if err != nil {
		t.Error(err)
	}
}

func TestUFetchOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UFetchOpenOrder(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestUAllAccountOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UAllAccountOpenOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUAllAccountOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UAllAccountOrders(t.Context(), currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.UAllAccountOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT), 0, 5, time.Now().Add(-time.Hour*4), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestUAccountBalanceV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UAccountBalanceV2(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestUAccountInformationV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UAccountInformationV2(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestUChangeInitialLeverageRequest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.UChangeInitialLeverageRequest(t.Context(), currency.NewPair(currency.BTC, currency.USDT), 2)
	if err != nil {
		t.Error(err)
	}
}

func TestUChangeInitialMarginType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.UChangeInitialMarginType(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "ISOLATED")
	if err != nil {
		t.Error(err)
	}
}

func TestUModifyIsolatedPositionMarginReq(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.UModifyIsolatedPositionMarginReq(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "LONG", "add", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestUPositionMarginChangeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.UPositionMarginChangeHistory(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "add", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestUPositionsInfoV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UPositionsInfoV2(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUAccountTradesHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UAccountTradesHistory(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestUAccountIncomeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UAccountIncomeHistory(t.Context(), currency.EMPTYPAIR, "", 5, time.Now().Add(-time.Hour*48), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestUGetNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UGetNotionalAndLeverageBrackets(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UPositionsADLEstimate(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestUAccountForcedOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UAccountForcedOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "ADL", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

// Coin Margined Futures

func TestGetFuturesExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := b.FuturesExchangeInfo(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesOrderbook(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 1000)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesPublicTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesPublicTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPastPublicTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetPastPublicTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAggregatedTradesList(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesAggregatedTradesList(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 0, 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetPerpsExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetPerpMarkets(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetIndexAndMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexAndMarkPrice(t.Context(), "", "BTCUSD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesKlineData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}

	start, end := getTime()
	_, err = b.GetFuturesKlineData(t.Context(), currency.NewPairWithDelimiter("LTCUSD", "PERP", "_"), "5m", 5, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestGetContinuousKlineData(t *testing.T) {
	t.Parallel()
	_, err := b.GetContinuousKlineData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "1M", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.GetContinuousKlineData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "1M", 5, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestGetIndexPriceKlines(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexPriceKlines(t.Context(), "BTCUSD", "1M", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.GetIndexPriceKlines(t.Context(), "BTCUSD", "1M", 5, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesSwapTickerChangeStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesSwapTickerChangeStats(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesSwapTickerChangeStats(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesSwapTickerChangeStats(t.Context(), currency.EMPTYPAIR, "")
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FuturesGetFundingHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.FuturesGetFundingHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 50, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetFuturesHistoricalTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 5)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesHistoricalTrades(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesSymbolPriceTicker(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesOrderbookTicker(t.Context(), currency.EMPTYPAIR, "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFuturesOrderbookTicker(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	if err != nil {
		t.Error(err)
	}
}

func TestOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := b.OpenInterest(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"))
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenInterestStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenInterestStats(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.GetOpenInterestStats(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTraderFuturesAccountRatio(t *testing.T) {
	t.Parallel()
	_, err := b.GetTraderFuturesAccountRatio(t.Context(), "BTCUSD", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.GetTraderFuturesAccountRatio(t.Context(), "BTCUSD", "5m", 0, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTraderFuturesPositionsRatio(t *testing.T) {
	t.Parallel()
	_, err := b.GetTraderFuturesPositionsRatio(t.Context(), "BTCUSD", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.GetTraderFuturesPositionsRatio(t.Context(), "BTCUSD", "5m", 0, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketRatio(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketRatio(t.Context(), "BTCUSD", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.GetMarketRatio(t.Context(), "BTCUSD", "5m", 0, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesTakerVolume(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesTakerVolume(t.Context(), "BTCUSD", "ALL", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.GetFuturesTakerVolume(t.Context(), "BTCUSD", "ALL", "5m", 0, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesBasisData(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesBasisData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	start, end := getTime()
	_, err = b.GetFuturesBasisData(t.Context(), "BTCUSD", "CURRENT_QUARTER", "5m", 0, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.FuturesNewOrder(
		t.Context(),
		&FuturesNewOrderRequest{
			Symbol:      currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
			Side:        "BUY",
			OrderType:   "LIMIT",
			TimeInForce: BinanceRequestParamsTimeGTC,
			Quantity:    1,
			Price:       1,
		},
	)
	if err != nil {
		t.Error(err)
	}
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
	_, err := b.FuturesBatchOrder(t.Context(), data)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesBatchCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.FuturesBatchCancelOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), []string{"123"}, []string{})
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesGetOrderData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FuturesGetOrderData(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.FuturesCancelAllOpenOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"))
	if err != nil {
		t.Error(err)
	}
}

func TestAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.AutoCancelAllOpenOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 30000)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesOpenOrderData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FuturesOpenOrderData(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetFuturesAllOpenOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAllFuturesOrders(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 2)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesChangeMarginType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.FuturesChangeMarginType(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "ISOLATED")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetFuturesAccountBalance(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetFuturesAccountInfo(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesChangeInitialLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.FuturesChangeInitialLeverage(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), 5)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyIsolatedPositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.ModifyIsolatedPositionMargin(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "BOTH", "add", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesMarginChangeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FuturesMarginChangeHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "add", time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FuturesPositionsInfo(t.Context(), "BTCUSD", "")
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FuturesTradeHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "", time.Time{}, time.Time{}, 5, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesIncomeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FuturesIncomeHistory(t.Context(), currency.EMPTYPAIR, "TRANSFER", time.Time{}, time.Time{}, 5)
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesForceOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FuturesForceOrders(t.Context(), currency.EMPTYPAIR, "ADL", time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestUGetNotionalLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FuturesNotionalBracket(t.Context(), "BTCUSD")
	if err != nil {
		t.Error(err)
	}
	_, err = b.FuturesNotionalBracket(t.Context(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestFuturesPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FuturesPositionsADLEstimate(t.Context(), currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkPriceKline(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"), "1M", 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	info, err := b.GetExchangeInfo(t.Context())
	require.NoError(t, err, "GetExchangeInfo must not error")
	if mockTests {
		exp := time.Date(2024, 5, 10, 6, 8, 1, int(707*time.Millisecond), time.UTC)
		assert.True(t, info.ServerTime.Time().Equal(exp), "expected %v received %v", exp.UTC(), info.ServerTime.Time().UTC())
	} else {
		assert.WithinRange(t, info.ServerTime.Time(), time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), "ServerTime should be within a day of now")
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := b.FetchTradablePairs(t.Context(), asset.Spot)
	if err != nil {
		t.Error("Binance FetchTradablePairs(asset asets.AssetType) error", err)
	}

	_, err = b.FetchTradablePairs(t.Context(), asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.FetchTradablePairs(t.Context(), asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook(t.Context(),
		OrderBookDataRequestParams{
			Symbol: currency.NewPair(currency.BTC, currency.USDT),
			Limit:  1000,
		})
	if err != nil {
		t.Error("Binance GetOrderBook() error", err)
	}
}

func TestGetMostRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetMostRecentTrades(t.Context(),
		RecentTradeRequestParams{
			Symbol: currency.NewPair(currency.BTC, currency.USDT),
			Limit:  15,
		})
	if err != nil {
		t.Error("Binance GetMostRecentTrades() error", err)
	}
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetHistoricalTrades(t.Context(), "BTCUSDT", 5, -1)
	if !mockTests && err == nil {
		t.Errorf("Binance GetHistoricalTrades() error: %v", "expected error")
	} else if mockTests && err != nil {
		t.Errorf("Binance GetHistoricalTrades() error: %v", err)
	}
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetAggregatedTrades(t.Context(),
		&AggregatedTradeRequestParams{
			Symbol: currency.NewPair(currency.BTC, currency.USDT),
			Limit:  5,
		})
	if err != nil {
		t.Error("Binance GetAggregatedTrades() error", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	_, err := b.GetSpotKline(t.Context(),
		&KlinesRequestParams{
			Symbol:    currency.NewPair(currency.BTC, currency.USDT),
			Interval:  kline.FiveMin.Short(),
			Limit:     24,
			StartTime: start,
			EndTime:   end,
		})
	if err != nil {
		t.Error("Binance GetSpotKline() error", err)
	}
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetAveragePrice(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetAveragePrice() error", err)
	}
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()

	_, err := b.GetPriceChangeStats(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetPriceChangeStats() error", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickers(t.Context())
	require.NoError(t, err)

	resp, err := b.GetTickers(t.Context(),
		currency.NewPair(currency.BTC, currency.USDT),
		currency.NewPair(currency.ETH, currency.USDT))
	require.NoError(t, err)
	require.Len(t, resp, 2)
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetLatestSpotPrice(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetLatestSpotPrice() error", err)
	}
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetBestPrice(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetBestPrice() error", err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()

	_, err := b.QueryOrder(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", 1337)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("QueryOrder() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("QueryOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock QueryOrder() error", err)
	}
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.OpenOrders(t.Context(), currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}

	p := currency.NewPair(currency.BTC, currency.USDT)
	_, err = b.OpenOrders(t.Context(), p)
	if err != nil {
		t.Error(err)
	}
}

func TestAllOrders(t *testing.T) {
	t.Parallel()

	_, err := b.AllOrders(t.Context(), currency.NewPair(currency.BTC, currency.USDT), "", "")
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("AllOrders() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("AllOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock AllOrders() error", err)
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()

	feeBuilder := setFeeBuilder()
	_, err := b.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(b) || mockTests {
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

	feeBuilder := setFeeBuilder()

	if sharedtestvalues.AreAPICredentialsSet(b) && mockTests {
		// CryptocurrencyTradeFee Basic
		if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
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
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     currency.Pairs{pair},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err = b.GetActiveOrders(t.Context(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("GetActiveOrders() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("GetActiveOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetActiveOrders() error", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()

	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := b.GetOrderHistory(t.Context(), &getOrdersRequest)
	if err == nil {
		t.Error("Expected: 'At least one currency is required to fetch order history'. received nil")
	}

	getOrdersRequest.Pairs = []currency.Pair{
		currency.NewPair(currency.LTC,
			currency.BTC),
	}

	_, err = b.GetOrderHistory(t.Context(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("GetOrderHistory() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
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

	err := b.NewOrderTest(t.Context(), req)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("NewOrderTest() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
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

	err = b.NewOrderTest(t.Context(), req)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("NewOrderTest() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("NewOrderTest() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock NewOrderTest() error", err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	p := currency.NewPair(currency.BTC, currency.USDT)
	start := time.Unix(1577977445, 0)  // 2020-01-02 15:04:05
	end := start.Add(15 * time.Minute) // 2020-01-02 15:19:05
	result, err := b.GetHistoricTrades(t.Context(), p, asset.Spot, start, end)
	assert.NoError(t, err, "GetHistoricTrades should not error")
	expected := 2134
	if mockTests {
		expected = 1002
	}
	assert.Equal(t, expected, len(result), "GetHistoricTrades should return correct number of entries")
	for _, r := range result {
		if !assert.WithinRange(t, r.Timestamp, start, end, "All trades should be within time range") {
			break
		}
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
			numExpected:  1012,
			lastExpected: time.Date(2020, 1, 2, 16, 18, 31, int(919*time.Millisecond), time.UTC),
		},
		{
			name: "batch with timerange",
			args: &AggregatedTradeRequestParams{
				Symbol:    currencyPair,
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
				Symbol:    currency.NewPair(currency.BTC, currency.USDT),
				StartTime: start,
				Limit:     1001,
			},
			numExpected:  1001,
			lastExpected: time.Date(2020, 1, 2, 15, 18, 39, int(226*time.Millisecond), time.UTC),
		},
		{
			name: "custom limit with start time set, no end time",
			args: &AggregatedTradeRequestParams{
				Symbol:    currency.NewPair(currency.BTC, currency.USDT),
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
				Symbol: currency.NewPair(currency.BTC, currency.USDT),
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := b.GetAggregatedTrades(t.Context(), tt.args)
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

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	}

	orderSubmission := &order.Submit{
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
	}

	_, err := b.SubmitOrder(t.Context(), orderSubmission)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("SubmitOrder() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("SubmitOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock SubmitOrder() error", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	}
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currency.NewPair(currency.LTC, currency.BTC),
		AssetType: asset.Spot,
	}

	err := b.CancelOrder(t.Context(), orderCancellation)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("CancelExchangeOrder() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("CancelExchangeOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock CancelExchangeOrder() error", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	}
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currency.NewPair(currency.LTC, currency.BTC),
		AssetType: asset.Spot,
	}

	_, err := b.CancelAllOrders(t.Context(), orderCancellation)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("CancelAllExchangeOrders() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("CancelAllExchangeOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock CancelAllExchangeOrders() error", err)
	}
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
			_, err := b.UpdateAccountInfo(t.Context(), assetType)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestWrapperGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	p, err := currency.NewPairFromString("EOS-USDT")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
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
	_, err = b.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	p, err := currency.NewPairFromString("EOSUSD_PERP")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Type:        order.AnyType,
		Side:        order.AnySide,
		FromOrderID: "123",
		Pairs:       currency.Pairs{p},
		AssetType:   asset.CoinMarginedFutures,
	})
	if err != nil {
		t.Error(err)
	}

	p2, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Type:        order.AnyType,
		Side:        order.AnySide,
		FromOrderID: "123",
		Pairs:       currency.Pairs{p2},
		AssetType:   asset.USDTMarginedFutures,
	})
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.USDTMarginedFutures,
	})
	if err == nil {
		t.Errorf("expecting an error since invalid param combination is given. Got err: %v", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	p, err := currency.NewPairFromString("EOS-USDT")
	if err != nil {
		t.Error(err)
	}
	fPair, err := b.FormatExchangeCurrency(p, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	err = b.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.CoinMarginedFutures,
		Pair:      fPair,
		OrderID:   "1234",
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
	err = b.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.USDTMarginedFutures,
		Pair:      fpair2,
		OrderID:   "1234",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	tradablePairs, err := b.FetchTradablePairs(t.Context(),
		asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	_, err = b.GetOrderInfo(t.Context(),
		"123", tradablePairs[0], asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := b.ModifyOrder(t.Context(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() error cannot be nil")
	}
}

func TestGetAllCoinsInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetAllCoinsInfo(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	}

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    b.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := b.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("Withdraw() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("Withdraw() expecting an error when no keys are set")
	}
}

func TestDepositHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	}
	_, err := b.DepositHistory(t.Context(), currency.ETH, "", time.Time{}, time.Time{}, 0, 10000)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error(err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("expecting an error when no keys are set")
	}
}

func TestWithdrawHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	}
	_, err := b.GetWithdrawalsHistory(t.Context(), currency.ETH, asset.Spot)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("GetWithdrawalsHistory() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("GetWithdrawalsHistory() expecting an error when no keys are set")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawFiatFunds(t.Context(),
		&withdraw.Request{})
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdraw.Request{})
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := b.GetDepositAddress(t.Context(), currency.USDT, "", currency.BNB.String())
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error("GetDepositAddress() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("GetDepositAddress() error cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock GetDepositAddress() error", err)
	}
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
			assert.NoError(bb, b.wsHandleData(lines[x]))
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
	require.NoError(t, err, "Subscribe should not error")
	err = b.Unsubscribe(channels)
	require.NoError(t, err, "Unsubscribe should not error")
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
	assert.ErrorIs(t, err, common.ErrUnknownError, "Subscribe should error correctly")
	assert.ErrorContains(t, err, "carrots", "Subscribe should error containing the carrots")
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
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Error(err)
	}
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
	if err := b.SeedLocalCacheWithBook(p, &book); err != nil {
		t.Fatal(err)
	}

	if err := b.wsHandleData(update1); err != nil {
		t.Fatal(err)
	}

	b.obm.state[currency.BTC][currency.USDT][asset.Spot].fetchingBook = false

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
  "T": 1573200697068}}`)
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
	key, err := b.GetWsAuthStreamKey(t.Context())
	switch {
	case mockTests && err != nil,
		!mockTests && sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Fatal(err)
	case !mockTests && !sharedtestvalues.AreAPICredentialsSet(b) && err == nil:
		t.Fatal("Expected error")
	}

	if key == "" && (sharedtestvalues.AreAPICredentialsSet(b) || mockTests) {
		t.Error("Expected key")
	}
}

func TestMaintainWsAuthStreamKey(t *testing.T) {
	err := b.MaintainWsAuthStreamKey(t.Context())
	switch {
	case mockTests && err != nil,
		!mockTests && sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Fatal(err)
	case !mockTests && !sharedtestvalues.AreAPICredentialsSet(b) && err == nil:
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
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end := startTime.Add(time.Hour * 24 * 7)
	bAssets := b.GetAssetTypes(false)
	for i := range bAssets {
		cps, err := b.GetAvailablePairs(bAssets[i])
		if err != nil {
			t.Error(err)
		}
		err = b.CurrencyPairs.EnablePair(bAssets[i], cps[0])
		if err != nil && !errors.Is(err, currency.ErrPairAlreadyEnabled) {
			t.Fatal(err)
		}
		_, err = b.GetHistoricCandles(t.Context(), cps[0], bAssets[i], kline.OneDay, startTime, end)
		if err != nil {
			t.Error(err)
		}
	}

	pair, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	startTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = b.GetHistoricCandles(t.Context(), pair, asset.Spot, kline.Interval(time.Hour*7), startTime, end)
	if !errors.Is(err, kline.ErrRequestExceedsExchangeLimits) {
		t.Fatalf("received: '%v', but expected: '%v'", err, kline.ErrRequestExceedsExchangeLimits)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end := startTime.Add(time.Hour * 24 * 7)
	bAssets := b.GetAssetTypes(false)
	for i := range bAssets {
		cps, err := b.GetAvailablePairs(bAssets[i])
		if err != nil {
			t.Error(err)
		}
		err = b.CurrencyPairs.EnablePair(bAssets[i], cps[0])
		if err != nil && !errors.Is(err, currency.ErrPairAlreadyEnabled) {
			t.Fatal(err)
		}
		_, err = b.GetHistoricCandlesExtended(t.Context(), cps[0], bAssets[i], kline.OneDay, startTime, end)
		if err != nil {
			t.Error(err)
		}
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
	pair := currency.NewPair(currency.BTC, currency.USDT)
	_, err := b.GetRecentTrades(t.Context(),
		pair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetRecentTrades(t.Context(),
		pair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	pair.Base = currency.NewCode("BTCUSD")
	pair.Quote = currency.PERP
	_, err = b.GetRecentTrades(t.Context(),
		pair, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := b.GetAvailableTransferChains(t.Context(), currency.BTC)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(b) && err != nil:
		t.Error(err)
	case !sharedtestvalues.AreAPICredentialsSet(b) && err == nil && !mockTests:
		t.Error("error cannot be nil")
	case mockTests && err != nil:
		t.Error(err)
	}
}

func TestSeedLocalCache(t *testing.T) {
	t.Parallel()
	err := b.SeedLocalCache(t.Context(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	exp := subscription.List{}
	pairs, err := b.GetEnabledPairs(asset.Spot)
	assert.NoError(t, err, "GetEnabledPairs should not error")
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	cp, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Error(err)
	}
	_, err = b.UFuturesHistoricalTrades(t.Context(), cp, "", 5)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UFuturesHistoricalTrades(t.Context(), cp, "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetExchangeOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := b.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	err = b.UpdateOrderExecutionLimits(t.Context(), asset.CoinMarginedFutures)
	if err != nil {
		t.Fatal(err)
	}

	err = b.UpdateOrderExecutionLimits(t.Context(), asset.USDTMarginedFutures)
	if err != nil {
		t.Fatal(err)
	}

	err = b.UpdateOrderExecutionLimits(t.Context(), asset.Binary)
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

	if limit == (order.MinMaxLevel{}) {
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
	if err != nil {
		t.Fatal(err)
	}
	res := <-b.Websocket.DataHandler
	switch r := res.(type) {
	case *order.Detail:
		if !reflect.DeepEqual(expectedResult, *r) {
			t.Errorf("Results do not match:\nexpected: %v\nreceived: %v", expectedResult, *r)
		}
	default:
		t.Fatalf("expected type order.Detail, found %T", res)
	}

	payload = []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"executionReport","E":1616633041556,"s":"BTCUSDT","c":"YeULctvPAnHj5HXCQo9Mob","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00028600","p":"52436.85000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":5341783271,"l":"0.00028600","z":"0.00028600","L":"52436.85000000","n":"0.00000029","N":"BTC","T":1616633041555,"t":726946523,"I":11390206312,"w":false,"m":false,"M":true,"O":1616633041555,"Z":"14.99693910","Y":"14.99693910","Q":"0.00000000","W":1616633041555}}`)
	err = b.wsHandleData(payload)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsOutboundAccountPosition(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"outboundAccountPosition","E":1616628815745,"u":1616628815745,"B":[{"a":"BTC","f":"0.00225109","l":"0.00123000"},{"a":"BNB","f":"0.00000000","l":"0.00000000"},{"a":"USDT","f":"54.43390661","l":"0.00000000"}]}}`)
	if err := b.wsHandleData(payload); err != nil {
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
			if err != nil {
				t.Error(err)
			}
			if result.Delimiter != tt.expectedDelimiter {
				t.Errorf("received '%v' expected '%v'", result.Delimiter, tt.expectedDelimiter)
			}
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
			if err != nil {
				t.Error(err)
			}
			if result != tt.expectedString {
				t.Errorf("received '%v' expected '%v'", result, tt.expectedString)
			}
		})
	}
}

func TestFormatUSDTMarginedFuturesPair(t *testing.T) {
	t.Parallel()
	pairFormat := currency.PairFormat{Uppercase: true}
	resp := b.formatUSDTMarginedFuturesPair(currency.NewPair(currency.DOGE, currency.USDT), pairFormat)
	if resp.String() != "DOGEUSDT" {
		t.Errorf("received '%v' expected '%v'", resp.String(), "DOGEUSDT")
	}

	resp = b.formatUSDTMarginedFuturesPair(currency.NewPair(currency.DOGE, currency.NewCode("1234567890")), pairFormat)
	if resp.String() != "DOGE_1234567890" {
		t.Errorf("received '%v' expected '%v'", resp.String(), "DOGE_1234567890")
	}
}

func TestFetchExchangeLimits(t *testing.T) {
	t.Parallel()
	limits, err := b.FetchExchangeLimits(t.Context(), asset.Spot)
	assert.NoError(t, err, "FetchExchangeLimits should not error")
	assert.NotEmpty(t, limits, "Should get some limits back")

	limits, err = b.FetchExchangeLimits(t.Context(), asset.Margin)
	assert.NoError(t, err, "FetchExchangeLimits should not error")
	assert.NotEmpty(t, limits, "Should get some limits back")

	_, err = b.FetchExchangeLimits(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "FetchExchangeLimits should error on other asset types")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()

	tests := map[asset.Item]currency.Pair{
		asset.Spot:   currency.NewPair(currency.BTC, currency.USDT),
		asset.Margin: currency.NewPair(currency.ETH, currency.BTC),
	}
	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
		pairs, err := b.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs should not error for %s", a)
		require.NotEmptyf(t, pairs, "Should get some pairs for %s", a)
		tests[a] = pairs[0]
	}

	for _, a := range b.GetAssetTypes(false) {
		err := b.UpdateOrderExecutionLimits(t.Context(), a)
		require.NoError(t, err, "UpdateOrderExecutionLimits should not error")

		p := tests[a]
		limits, err := b.GetOrderExecutionLimits(a, p)
		require.NoErrorf(t, err, "GetOrderExecutionLimits should not error for %s pair %s", a, p)
		assert.Positivef(t, limits.MinPrice, "MinPrice must be positive for %s pair %s", a, p)
		assert.Positivef(t, limits.MaxPrice, "MaxPrice must be positive for %s pair %s", a, p)
		assert.Positivef(t, limits.PriceStepIncrementSize, "PriceStepIncrementSize must be positive for %s pair %s", a, p)
		assert.Positivef(t, limits.MinimumBaseAmount, "MinimumBaseAmount must be positive for %s pair %s", a, p)
		assert.Positivef(t, limits.MaximumBaseAmount, "MaximumBaseAmount must be positive for %s pair %s", a, p)
		assert.Positivef(t, limits.AmountStepIncrementSize, "AmountStepIncrementSize must be positive for %s pair %s", a, p)
		assert.Positivef(t, limits.MarketMaxQty, "MarketMaxQty must be positive for %s pair %s", a, p)
		assert.Positivef(t, limits.MaxTotalOrders, "MaxTotalOrders must be positive for %s pair %s", a, p)
		switch a {
		case asset.Spot, asset.Margin:
			assert.Positivef(t, limits.MaxIcebergParts, "MaxIcebergParts must be positive for %s pair %s", a, p)
		case asset.USDTMarginedFutures:
			assert.Positivef(t, limits.MinNotional, "MinNotional must be positive for %s pair %s", a, p)
			fallthrough
		case asset.CoinMarginedFutures:
			assert.Positivef(t, limits.MultiplierUp, "MultiplierUp must be positive for %s pair %s", a, p)
			assert.Positivef(t, limits.MultiplierDown, "MultiplierDown must be positive for %s pair %s", a, p)
			assert.Positivef(t, limits.MarketMinQty, "MarketMinQty must be positive for %s pair %s", a, p)
			assert.Positivef(t, limits.MarketStepIncrementSize, "MarketStepIncrementSize must be positive for %s pair %s", a, p)
			assert.Positivef(t, limits.MaxAlgoOrders, "MaxAlgoOrders must be positive for %s pair %s", a, p)
		}
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	s, e := getTime()
	_, err := b.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewPair(currency.BTC, currency.USDT),
		StartDate:            s,
		EndDate:              e,
		IncludePayments:      true,
		IncludePredictedRate: true,
	})
	if !errors.Is(err, common.ErrFunctionNotSupported) {
		t.Error(err)
	}

	_, err = b.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewPair(currency.BTC, currency.USDT),
		StartDate:       s,
		EndDate:         e,
		PaymentCurrency: currency.DOGE,
	})
	if !errors.Is(err, common.ErrFunctionNotSupported) {
		t.Error(err)
	}

	r := &fundingrate.HistoricalRatesRequest{
		Asset:     asset.USDTMarginedFutures,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		StartDate: s,
		EndDate:   e,
	}
	if sharedtestvalues.AreAPICredentialsSet(b) {
		r.IncludePayments = true
	}
	_, err = b.GetHistoricalFundingRates(t.Context(), r)
	if err != nil {
		t.Error(err)
	}

	r.Asset = asset.CoinMarginedFutures
	r.Pair, err = currency.NewPairFromString("BTCUSD_PERP")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricalFundingRates(t.Context(), r)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	cp := currency.NewPair(currency.BTC, currency.USDT)
	_, err := b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 cp,
		IncludePredictedRate: true,
	})
	if !errors.Is(err, common.ErrFunctionNotSupported) {
		t.Error(err)
	}
	err = b.CurrencyPairs.EnablePair(asset.USDTMarginedFutures, cp)
	if err != nil && !errors.Is(err, currency.ErrPairAlreadyEnabled) {
		t.Fatal(err)
	}
	_, err = b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  cp,
	})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.CoinMarginedFutures,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := b.IsPerpetualFutureCurrency(asset.Binary, currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	if is {
		t.Error("expected false")
	}

	is, err = b.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	if is {
		t.Error("expected false")
	}
	is, err = b.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewPair(currency.BTC, currency.PERP))
	if err != nil {
		t.Error(err)
	}
	if !is {
		t.Error("expected true")
	}

	is, err = b.IsPerpetualFutureCurrency(asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	if !is {
		t.Error("expected true")
	}
	is, err = b.IsPerpetualFutureCurrency(asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.PERP))
	if err != nil {
		t.Error(err)
	}
	if is {
		t.Error("expected false")
	}
}

func TestGetUserMarginInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetUserMarginInterestHistory(t.Context(), currency.USDT, currency.NewPair(currency.BTC, currency.USDT), time.Now().Add(-time.Hour*24), time.Now(), 1, 10, false)
	if err != nil {
		t.Error(err)
	}
}

func TestSetAssetsMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	is, err := b.GetAssetsMode(t.Context())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}

	err = b.SetAssetsMode(t.Context(), !is)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}

	err = b.SetAssetsMode(t.Context(), is)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}

func TestGetAssetsMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAssetsMode(t.Context())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.GetCollateralMode(t.Context(), asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v', expected '%v'", err, asset.ErrNotSupported)
	}
	_, err = b.GetCollateralMode(t.Context(), asset.CoinMarginedFutures)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v', expected '%v'", err, asset.ErrNotSupported)
	}
	_, err = b.GetCollateralMode(t.Context(), asset.USDTMarginedFutures)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetCollateralMode(t.Context(), asset.Spot, collateral.SingleMode)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v', expected '%v'", err, asset.ErrNotSupported)
	}
	err = b.SetCollateralMode(t.Context(), asset.CoinMarginedFutures, collateral.SingleMode)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v', expected '%v'", err, asset.ErrNotSupported)
	}
	err = b.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.MultiMode)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	err = b.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.PortfolioMode)
	if !errors.Is(err, order.ErrCollateralInvalid) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrCollateralInvalid)
	}
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
		Pair:                    currency.NewBTCUSDT(),
		Asset:                   asset.USDTMarginedFutures,
		MarginType:              margin.Isolated,
		OriginalAllocatedMargin: 1337,
		NewAllocatedMargin:      1333337,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetPositionSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	bb := currency.NewBTCUSDT()
	_, err := b.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  bb,
	})
	if err != nil {
		t.Error(err)
	}

	bb.Quote = currency.BUSD
	_, err = b.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  bb,
	})
	if err != nil {
		t.Error(err)
	}

	p, err := currency.NewPairFromString("BTCUSD_PERP")
	if err != nil {
		t.Fatal(err)
	}
	bb.Quote = currency.USD
	_, err = b.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset:          asset.CoinMarginedFutures,
		Pair:           p,
		UnderlyingPair: bb,
	})
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
		Asset:          asset.Spot,
		Pair:           p,
		UnderlyingPair: bb,
	})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
		Asset:                     asset.USDTMarginedFutures,
		Pairs:                     []currency.Pair{currency.NewBTCUSDT()},
		StartDate:                 time.Now().Add(-time.Hour * 24 * 70),
		RespectOrderHistoryLimits: true,
	})
	if err != nil {
		t.Error(err)
	}

	p, err := currency.NewPairFromString("ADAUSD_PERP")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
		Asset:                     asset.CoinMarginedFutures,
		Pairs:                     []currency.Pair{p},
		StartDate:                 time.Now().Add(time.Hour * 24 * -70),
		RespectOrderHistoryLimits: true,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	err := b.SetMarginType(t.Context(), asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.USDT), margin.Isolated)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	p, err := currency.NewPairFromString("BTCUSD_PERP")
	if err != nil {
		t.Fatal(err)
	}
	err = b.SetMarginType(t.Context(), asset.CoinMarginedFutures, p, margin.Isolated)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	err = b.SetMarginType(t.Context(), asset.Spot, currency.NewPair(currency.BTC, currency.USDT), margin.Isolated)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), 0, order.UnknownSide)
	if err != nil {
		t.Error(err)
	}

	p, err := currency.NewPairFromString("BTCUSD_PERP")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetLeverage(t.Context(), asset.CoinMarginedFutures, p, 0, order.UnknownSide)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetLeverage(t.Context(), asset.Spot, currency.NewBTCUSDT(), 0, order.UnknownSide)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 5, order.UnknownSide)
	if err != nil {
		t.Error(err)
	}

	p, err := currency.NewPairFromString("BTCUSD_PERP")
	if err != nil {
		t.Fatal(err)
	}
	err = b.SetLeverage(t.Context(), asset.CoinMarginedFutures, p, margin.Multi, 5, order.UnknownSide)
	if err != nil {
		t.Error(err)
	}
	err = b.SetLeverage(t.Context(), asset.Spot, p, margin.Multi, 5, order.UnknownSide)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}
}

func TestGetCryptoLoansIncomeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.CryptoLoanIncomeHistory(t.Context(), currency.USDT, "", time.Time{}, time.Time{}, 100); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanBorrow(t *testing.T) {
	t.Parallel()
	if _, err := b.CryptoLoanBorrow(t.Context(), currency.EMPTYCODE, 1000, currency.BTC, 1, 7); !errors.Is(err, errLoanCoinMustBeSet) {
		t.Errorf("received %v, expected %v", err, errLoanCoinMustBeSet)
	}
	if _, err := b.CryptoLoanBorrow(t.Context(), currency.USDT, 1000, currency.EMPTYCODE, 1, 7); !errors.Is(err, errCollateralCoinMustBeSet) {
		t.Errorf("received %v, expected %v", err, errCollateralCoinMustBeSet)
	}
	if _, err := b.CryptoLoanBorrow(t.Context(), currency.USDT, 0, currency.BTC, 1, 0); !errors.Is(err, errLoanTermMustBeSet) {
		t.Errorf("received %v, expected %v", err, errLoanTermMustBeSet)
	}
	if _, err := b.CryptoLoanBorrow(t.Context(), currency.USDT, 0, currency.BTC, 0, 7); !errors.Is(err, errEitherLoanOrCollateralAmountsMustBeSet) {
		t.Errorf("received %v, expected %v", err, errEitherLoanOrCollateralAmountsMustBeSet)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if _, err := b.CryptoLoanBorrow(t.Context(), currency.USDT, 1000, currency.BTC, 1, 7); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanBorrowHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.CryptoLoanBorrowHistory(t.Context(), 0, currency.USDT, currency.BTC, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.CryptoLoanOngoingOrders(t.Context(), 0, currency.USDT, currency.BTC, 0, 0); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanRepay(t *testing.T) {
	t.Parallel()
	if _, err := b.CryptoLoanRepay(t.Context(), 0, 1000, 1, false); !errors.Is(err, errOrderIDMustBeSet) {
		t.Errorf("received %v, expected %v", err, errOrderIDMustBeSet)
	}
	if _, err := b.CryptoLoanRepay(t.Context(), 42069, 0, 1, false); !errors.Is(err, errAmountMustBeSet) {
		t.Errorf("received %v, expected %v", err, errAmountMustBeSet)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if _, err := b.CryptoLoanRepay(t.Context(), 42069, 1000, 1, false); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.CryptoLoanRepaymentHistory(t.Context(), 0, currency.USDT, currency.BTC, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanAdjustLTV(t *testing.T) {
	t.Parallel()
	if _, err := b.CryptoLoanAdjustLTV(t.Context(), 0, true, 1); !errors.Is(err, errOrderIDMustBeSet) {
		t.Errorf("received %v, expected %v", err, errOrderIDMustBeSet)
	}
	if _, err := b.CryptoLoanAdjustLTV(t.Context(), 42069, true, 0); !errors.Is(err, errAmountMustBeSet) {
		t.Errorf("received %v, expected %v", err, errAmountMustBeSet)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if _, err := b.CryptoLoanAdjustLTV(t.Context(), 42069, true, 1); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.CryptoLoanLTVAdjustmentHistory(t.Context(), 0, currency.USDT, currency.BTC, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.CryptoLoanAssetsData(t.Context(), currency.EMPTYCODE, 0); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanCollateralAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.CryptoLoanCollateralAssetsData(t.Context(), currency.EMPTYCODE, 0); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanCheckCollateralRepayRate(t *testing.T) {
	t.Parallel()
	if _, err := b.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.EMPTYCODE, currency.BNB, 69); !errors.Is(err, errLoanCoinMustBeSet) {
		t.Errorf("received %v, expected %v", err, errLoanCoinMustBeSet)
	}
	if _, err := b.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.BUSD, currency.EMPTYCODE, 69); !errors.Is(err, errCollateralCoinMustBeSet) {
		t.Errorf("received %v, expected %v", err, errCollateralCoinMustBeSet)
	}
	if _, err := b.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.BUSD, currency.BNB, 0); !errors.Is(err, errAmountMustBeSet) {
		t.Errorf("received %v, expected %v", err, errAmountMustBeSet)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.CryptoLoanCheckCollateralRepayRate(t.Context(), currency.BUSD, currency.BNB, 69); err != nil {
		t.Error(err)
	}
}

func TestCryptoLoanCustomiseMarginCall(t *testing.T) {
	t.Parallel()
	if _, err := b.CryptoLoanCustomiseMarginCall(t.Context(), 0, currency.BTC, 0); err == nil {
		t.Error("expected an error")
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if _, err := b.CryptoLoanCustomiseMarginCall(t.Context(), 1337, currency.BTC, .70); err != nil {
		t.Error(err)
	}
}

func TestFlexibleLoanBorrow(t *testing.T) {
	t.Parallel()
	if _, err := b.FlexibleLoanBorrow(t.Context(), currency.EMPTYCODE, currency.USDC, 1, 0); !errors.Is(err, errLoanCoinMustBeSet) {
		t.Errorf("received %v, expected %v", err, errLoanCoinMustBeSet)
	}
	if _, err := b.FlexibleLoanBorrow(t.Context(), currency.ATOM, currency.EMPTYCODE, 1, 0); !errors.Is(err, errCollateralCoinMustBeSet) {
		t.Errorf("received %v, expected %v", err, errCollateralCoinMustBeSet)
	}
	if _, err := b.FlexibleLoanBorrow(t.Context(), currency.ATOM, currency.USDC, 0, 0); !errors.Is(err, errEitherLoanOrCollateralAmountsMustBeSet) {
		t.Errorf("received %v, expected %v", err, errEitherLoanOrCollateralAmountsMustBeSet)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if _, err := b.FlexibleLoanBorrow(t.Context(), currency.ATOM, currency.USDC, 1, 0); err != nil {
		t.Error(err)
	}
}

func TestFlexibleLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.FlexibleLoanOngoingOrders(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, 0, 0); err != nil {
		t.Error(err)
	}
}

func TestFlexibleLoanBorrowHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.FlexibleLoanBorrowHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Error(err)
	}
}

func TestFlexibleLoanRepay(t *testing.T) {
	t.Parallel()

	if _, err := b.FlexibleLoanRepay(t.Context(), currency.EMPTYCODE, currency.BTC, 1, false, false); !errors.Is(err, errLoanCoinMustBeSet) {
		t.Errorf("received %v, expected %v", err, errLoanCoinMustBeSet)
	}
	if _, err := b.FlexibleLoanRepay(t.Context(), currency.USDT, currency.EMPTYCODE, 1, false, false); !errors.Is(err, errCollateralCoinMustBeSet) {
		t.Errorf("received %v, expected %v", err, errCollateralCoinMustBeSet)
	}
	if _, err := b.FlexibleLoanRepay(t.Context(), currency.USDT, currency.BTC, 0, false, false); !errors.Is(err, errAmountMustBeSet) {
		t.Errorf("received %v, expected %v", err, errAmountMustBeSet)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if _, err := b.FlexibleLoanRepay(t.Context(), currency.ATOM, currency.USDC, 1, false, false); err != nil {
		t.Error(err)
	}
}

func TestFlexibleLoanRepayHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.FlexibleLoanRepayHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Error(err)
	}
}

func TestFlexibleLoanAdjustLTV(t *testing.T) {
	t.Parallel()
	if _, err := b.FlexibleLoanAdjustLTV(t.Context(), currency.EMPTYCODE, currency.BTC, 1, true); !errors.Is(err, errLoanCoinMustBeSet) {
		t.Errorf("received %v, expected %v", err, errLoanCoinMustBeSet)
	}
	if _, err := b.FlexibleLoanAdjustLTV(t.Context(), currency.USDT, currency.EMPTYCODE, 1, true); !errors.Is(err, errCollateralCoinMustBeSet) {
		t.Errorf("received %v, expected %v", err, errCollateralCoinMustBeSet)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	if _, err := b.FlexibleLoanAdjustLTV(t.Context(), currency.USDT, currency.BTC, 1, true); err != nil {
		t.Error(err)
	}
}

func TestFlexibleLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.FlexibleLoanLTVAdjustmentHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Error(err)
	}
}

func TestFlexibleLoanAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.FlexibleLoanAssetsData(t.Context(), currency.EMPTYCODE); err != nil {
		t.Error(err)
	}
}

func TestFlexibleCollateralAssetsData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if _, err := b.FlexibleCollateralAssetsData(t.Context(), currency.EMPTYCODE); err != nil {
		t.Error(err)
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesContractDetails(t.Context(), asset.Spot)
	if !errors.Is(err, futures.ErrNotFuturesAsset) {
		t.Error(err)
	}
	_, err = b.GetFuturesContractDetails(t.Context(), asset.Futures)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}
	_, err = b.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	_, err = b.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
}

func TestGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetFundingRateInfo(t.Context())
	assert.NoError(t, err)
}

func TestUGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	_, err := b.UGetFundingRateInfo(t.Context())
	assert.NoError(t, err)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	resp, err := b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.NewCode("BTCUSD").Item,
		Quote: currency.PERP.Item,
		Asset: asset.CoinMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	_, err = b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.Spot,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)
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
		assert.NotEmpty(t, resp)
	}
}
