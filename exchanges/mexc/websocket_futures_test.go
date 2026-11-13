package mexc

import (
	"cmp"
	"context"
	"encoding/hex"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// The public frames below were pushed by the venue on 2026-10-24; the private ones are its documented examples
const (
	futuresTickerFrame = `{"symbol":"BTC_USDT","data":{"symbol":"BTC_USDT","lastPrice":80591.4,"riseFallRate":-0.0129,` +
		`"fairPrice":80591.5,"indexPrice":80628.1,"volume24":361800379,"amount24":2939324762.13142,"maxBidPrice":88690.9,` +
		`"minAskPrice":72565.2,"lower24Price":80409.4,"high24Price":82032.7,"timestamp":1792842497019,"bid1":80590,` +
		`"ask1":80590.1,"holdVol":489668514,"riseFallValue":-1059.8,"fundingRate":0.000034,"zone":"UTC+8",` +
		`"riseFallRates":[-0.0129,-0.0089,-0.0401,0.2562,0.0331,-0.2662],"riseFallRatesOfTimezone":[-0.0130,-0.0070,-0.0129]},` +
		`"channel":"push.ticker","ts":1792842497019}`
	futuresDealsFrame = `{"symbol":"BTC_USDT","data":[{"p":80585.5,"v":7739,"T":2,"O":1,"M":2,"t":1792842498973,` +
		`"i":"16745103311","cts":"1792842498973"},{"p":80590.1,"v":4,"T":1,"O":3,"M":1,"t":1792842498334,"i":"16745103221",` +
		`"cts":"1792842498334"}],"channel":"push.deal","ts":1792842498973}`
	futuresDealFrame = `{"symbol":"BTC_USDT","data":{"p":80590,"v":5,"T":2,"O":3,"M":1,"t":1792842497333,"i":"16745103083",` +
		`"cts":"1792842497333"},"channel":"push.deal","ts":1792842497333}`
	futuresDepthFrame = `{"symbol":"BTC_USDT","data":{"cts":1792842498876,"asks":[[80590.1,46440,12],[80590.4,184,1]],` +
		`"bids":[[80590.0,1074,2],[80589.8,184,1]],"version":42904820315},"channel":"push.depth.full","ts":1792842498880}`
	futuresKlineFrame = `{"symbol":"BTC_USD","data":{"symbol":"BTC_USD","interval":"Min15","t":1792842300,"o":80479.5,` +
		`"c":80490.8,"h":80490.8,"l":80479.4,"a":0.0708234966710365,"q":57,"ro":80479.4,"rc":80490.8,"rh":80490.8,` +
		`"rl":80479.4},"channel":"push.kline","ts":1792842500159}`
	futuresOrderFrame = `{"channel":"push.personal.order","data":{"orderId":123456789,"symbol":"BTC_USDT","positionId":987654321,` +
		`"price":45000.5,"vol":10,"leverage":20,"side":1,"category":1,"orderType":1,"dealAvgPrice":45000.0,"dealVol":5,` +
		`"orderMargin":2250.0,"usedMargin":1125.0,"takerFee":0.1125,"makerFee":0.09,"profit":0,"feeCurrency":"USDT",` +
		`"openType":1,"state":2,"errorCode":0,"externalOid":"ext_001","createTime":1760942212000,"updateTime":1760942213000,` +
		`"remainVol":5,"positionMode":1,"reduceOnly":false,"bboTypeNum":0,"makerFeeRate":0.0002,"takerFeeRate":0.0004},` +
		`"ts":1760942212000}`
	futuresFillFrame = `{"channel":"push.personal.order.deal","data":{"id":987654321,"symbol":"BTC_USDT","side":4,"vol":10,` +
		`"price":45000.5,"feeCurrency":"USDT","fee":0.225,"timestamp":1760942212000,"profit":0,"isTaker":true,"category":1,` +
		`"orderId":123456789,"isSelf":false,"externalOid":"ext_001","positionMode":1,"reduceOnly":false,"opponentUid":987654321},` +
		`"ts":1760942212000}`
	futuresAssetFrame = `{"channel":"push.personal.asset","data":{"currency":"USDT","positionMargin":5000.0,` +
		`"frozenBalance":1000.0,"availableBalance":9000.0,"bonus":100.0},"ts":1760942212000}`
	futuresPositionFrame = `{"channel":"push.personal.position","data":{"positionId":123456789,"symbol":"BTC_USDT",` +
		`"holdVol":10,"positionType":2,"openType":1,"state":1,"frozenVol":0,"closeVol":0,"holdAvgPrice":45000.5,` +
		`"openAvgPrice":45000.5,"liquidatePrice":50000.0,"oim":2250.025,"adlLevel":1,"im":2250.025,"holdFee":0,"realised":-0.1125,` +
		`"leverage":20,"autoAddIm":false,"pnl":100.5,"marginRatio":0.2,"createTime":1760942212000,"updateTime":1760942213000,` +
		`"version":3},"ts":1760942212000}`
)

// futuresTickerFramePrice returns futuresTickerFrame as an exchange stores it. The fair price is the mark price, and
// the base volume and open interest are the contracts times their size
func futuresTickerFramePrice(exchangeName string) *ticker.Price {
	return &ticker.Price{
		Last:         80591.4,
		High:         82032.7,
		Low:          80409.4,
		Bid:          80590,
		Ask:          80590.1,
		BaseVolume:   36180.0379,
		QuoteVolume:  2939324762.13142,
		OpenInterest: 48966.8514,
		MarkPrice:    80591.5,
		IndexPrice:   80628.1,
		Pair:         futuresTradablePair,
		ExchangeName: exchangeName,
		AssetType:    asset.Futures,
		LastUpdated:  time.UnixMilli(1792842497019),
	}
}

// futuresOrderFrameDetail returns futuresOrderFrame as an exchange relays it. An open order with fills is partially
// filled, its cost is the filled contracts' value, and its fee is its taker and maker fees so far
func futuresOrderFrameDetail(exchangeName string) *order.Detail {
	return &order.Detail{
		TimeInForce:          order.GoodTillCancel,
		Leverage:             20,
		Price:                45000.5,
		Amount:               10,
		ContractAmount:       10,
		AverageExecutedPrice: 45000,
		ExecutedAmount:       5,
		RemainingAmount:      5,
		Cost:                 22.5,
		CostAsset:            currency.USDT,
		Fee:                  0.2025,
		FeeAsset:             currency.USDT,
		Exchange:             exchangeName,
		OrderID:              "123456789",
		ClientOrderID:        "ext_001",
		Type:                 order.Limit,
		Side:                 order.Buy,
		Status:               order.PartiallyFilled,
		AssetType:            asset.Futures,
		Date:                 time.UnixMilli(1760942212000),
		LastUpdated:          time.UnixMilli(1760942213000),
		Pair:                 futuresTradablePair,
		MarginType:           margin.Isolated,
	}
}

// newFuturesWebsocketTestExchange returns an exchange named for the test which holds the futures contracts' details,
// so its handlers read no contract over REST
func newFuturesWebsocketTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	require.NoError(t, ex.setEnabledPairs(spotTradablePair), "setEnabledPairs must not error")
	var contracts []*FuturesContractResponse
	require.NoError(t, json.Unmarshal([]byte(futuresContractsReply), &contracts), "the contracts must decode")
	ex.storeFuturesContracts(contracts)
	return ex
}

// subscribeFuturesPrivate records a subscription to a private futures channel, which the handler checks before
// relaying the channel's pushes
func subscribeFuturesPrivate(t *testing.T, ex *Exchange, channel string) {
	t.Helper()
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(nil, &subscription.Subscription{Channel: channel, Asset: asset.Futures, Authenticated: true}),
		"AddSuccessfulSubscriptions must not error")
}

// relayed returns the next payload relayed by an exchange's websocket
func relayed(t *testing.T, ex *Exchange) any {
	t.Helper()
	select {
	case p := <-ex.Websocket.DataHandler.C:
		return p.Data
	case <-time.After(5 * time.Second):
		require.FailNow(t, "a payload must be relayed")
		return nil
	}
}

// requireNothingRelayed checks an exchange's websocket has relayed nothing
func requireNothingRelayed(t *testing.T, ex *Exchange) {
	t.Helper()
	select {
	case p := <-ex.Websocket.DataHandler.C:
		require.FailNowf(t, "nothing must be relayed", "relayed %T: %+v", p.Data, p.Data)
	default:
	}
}

func TestWebsocketFuturesHandleTicker(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresTickerFrame)), "WebsocketFuturesHandleData must not error")

	tick, ok := relayed(t, ex).(*ticker.Price)
	require.True(t, ok, "a ticker must be relayed")
	stored, err := ticker.GetTicker(ex.Name, futuresTradablePair, asset.Futures)
	require.NoError(t, err, "the ticker must be stored before it is relayed")
	assert.Equal(t, tick, stored, "the relayed ticker should be the stored one")
	assert.Equal(t, futuresTickerFramePrice(ex.Name), tick, "the ticker should be mapped")
}

func TestWebsocketFuturesHandleOrderbook(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresDepthFrame)), "WebsocketFuturesHandleData must not error")

	book, err := orderbook.Get(ex.Name, futuresTradablePair, asset.Futures)
	require.NoError(t, err, "the snapshot must be loaded")
	assert.False(t, book.InsertedAt.IsZero(), "the book should record when it was loaded")
	book.InsertedAt = time.Time{}
	// The version is the update ID, the matching engine's time the update time, and the push's time is recorded
	exp := &orderbook.Book{
		Bids:              orderbook.Levels{{Price: 80590, Amount: 1074, OrderCount: 2}, {Price: 80589.8, Amount: 184, OrderCount: 1}},
		Asks:              orderbook.Levels{{Price: 80590.1, Amount: 46440, OrderCount: 12}, {Price: 80590.4, Amount: 184, OrderCount: 1}},
		Exchange:          ex.Name,
		Pair:              futuresTradablePair,
		Asset:             asset.Futures,
		LastUpdated:       time.UnixMilli(1792842498876),
		LastPushed:        time.UnixMilli(1792842498880),
		LastUpdateID:      42904820315,
		ValidateOrderbook: true,
	}
	assert.Equal(t, exp, book, "the snapshot should be loaded")
}

func TestWebsocketFuturesHandleOrder(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresOrderFrame)), "WebsocketFuturesHandleData must not error")
	requireNothingRelayed(t, ex)

	subscribeFuturesPrivate(t, ex, subscription.MyOrdersChannel)
	require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresOrderFrame)), "WebsocketFuturesHandleData must not error")
	detail, ok := relayed(t, ex).(*order.Detail)
	require.True(t, ok, "an order must be relayed")
	assert.Equal(t, futuresOrderFrameDetail(ex.Name), detail, "the order should be mapped")

	// The order store takes only a pushed fee and remaining amount which are set
	stored := &order.Detail{Exchange: ex.Name, OrderID: detail.OrderID, Amount: 10, RemainingAmount: 10}
	require.NoError(t, stored.UpdateOrderFromDetail(detail), "UpdateOrderFromDetail must not error")
	assert.Equal(t, 5.0, stored.RemainingAmount, "the stored order should have five contracts left")
	assert.InDelta(t, 0.2025, stored.Fee, 1e-12, "the stored order should carry the fees so far")
}

func TestWebsocketFuturesHandleAsset(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	// Balances are stored by the account's credentials
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
	require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresAssetFrame)), "WebsocketFuturesHandleData must not error")
	requireNothingRelayed(t, ex)

	subscribeFuturesPrivate(t, ex, subscription.MyAccountChannel)
	require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresAssetFrame)), "WebsocketFuturesHandleData must not error")
	subAccounts, ok := relayed(t, ex).(accounts.SubAccounts)
	require.True(t, ok, "the balance must be relayed")
	require.Len(t, subAccounts, 1, "the futures account must be relayed")
	b := subAccounts[0].Balances[currency.USDT]
	assert.False(t, b.UpdatedAt.IsZero(), "the balance should record when it was updated")
	b.UpdatedAt = time.Time{}
	subAccounts[0].Balances[currency.USDT] = b
	// Frozen funds and position margin are held, and the total is what is free and held
	exp := accounts.SubAccounts{{
		AssetType: asset.Futures,
		Balances:  accounts.CurrencyBalances{currency.USDT: {Currency: currency.USDT, Total: 15000, Hold: 6000, Free: 9000}},
	}}
	assert.Equal(t, exp, subAccounts, "the balance should be relayed")

	err := ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(strings.Replace(futuresAssetFrame, `"currency":"USDT"`, `"currency":""`, 1)))
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "a balance without a currency should be refused")
}

func TestWebsocketFuturesHandleData(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	// The cases share ex in order, and the check after them finds that none of them relayed anything
	for name, frame := range map[string]string{
		"pong":                               `{"channel":"pong","data":1792842501046,"ts":1792842501046}`,
		"unsubscribable push":                `{"channel":"push.personal.leverage.mode","data":{"leverageMode":1},"ts":1760942212000}`,
		"late subscription ack":              `{"channel":"rs.sub.ticker","data":"success","ts":1792842495741}`,
		"late marker refusal":                `{"channel":"rs.error","data":"Contract [gct-resync-1] not exists","ts":1792842495741}`,
		"late marker refusal in other words": `{"channel":"rs.error","data":"marker gct-resync-234 refused","ts":1792842495741}`,
	} {
		t.Run(name, func(t *testing.T) {
			require.NoErrorf(t, ex.WebsocketFuturesHandleData(t.Context(), &futuresReplyConnection{}, []byte(frame)), "WebsocketFuturesHandleData must not error for a %s", name)
		})
	}
	requireNothingRelayed(t, ex)

	// The cases share ex in order, each taking the next payload it relays
	for _, frame := range []string{`{"channel":"push.unknown","data":{}}`, `{"channel":"surprise"}`, `null`} {
		t.Run(frame, func(t *testing.T) {
			require.NoErrorf(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(frame)), "WebsocketFuturesHandleData must not error for %s", frame)
			warning, ok := relayed(t, ex).(websocket.UnhandledMessageWarning)
			require.Truef(t, ok, "%s must be reported as unhandled", frame)
			assert.Containsf(t, warning.Message, frame, "the warning should carry %s", frame)
		})
	}

	err := ex.WebsocketFuturesHandleData(t.Context(), &futuresReplyConnection{}, []byte(`{"channel":"rs.error","data":"Contract [FOO_BAR] not exists"}`))
	assert.ErrorIs(t, err, errFuturesWebsocketRejected, "a refusal no request waits for should be reported")
	assert.Error(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(`{"channel":`)), "a malformed frame should be refused")

	t.Run("trades", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		ex.SetSaveTradeDataStatus(false)
		ex.SetTradeFeedStatus(false)
		require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresDealsFrame)), "WebsocketFuturesHandleData must not error")
		requireNothingRelayed(t, ex)

		ex.SetTradeFeedStatus(true)
		require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresDealsFrame)), "WebsocketFuturesHandleData must not error")
		trades, ok := relayed(t, ex).([]trade.Data)
		require.True(t, ok, "trades must be relayed")
		// A taker sell is a sell and a taker buy a buy, and the amounts are in contracts
		exp := []trade.Data{
			{TID: "16745103311", Exchange: ex.Name, CurrencyPair: futuresTradablePair, AssetType: asset.Futures, Side: order.Sell, Price: 80585.5, Amount: 7739, Timestamp: time.UnixMilli(1792842498973)},
			{TID: "16745103221", Exchange: ex.Name, CurrencyPair: futuresTradablePair, AssetType: asset.Futures, Side: order.Buy, Price: 80590.1, Amount: 4, Timestamp: time.UnixMilli(1792842498334)},
		}
		assert.Equal(t, exp, trades, "every merged trade should be relayed")

		require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresDealFrame)), "WebsocketFuturesHandleData must not error for an unmerged trade")
		trades, ok = relayed(t, ex).([]trade.Data)
		require.True(t, ok, "an unmerged trade must be relayed")
		exp = []trade.Data{{TID: "16745103083", Exchange: ex.Name, CurrencyPair: futuresTradablePair, AssetType: asset.Futures, Side: order.Sell, Price: 80590, Amount: 5, Timestamp: time.UnixMilli(1792842497333)}}
		assert.Equal(t, exp, trades, "the unmerged trade should be relayed")
	})

	t.Run("candle", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresKlineFrame)), "WebsocketFuturesHandleData must not error")

		item, ok := relayed(t, ex).(kline.Item)
		require.True(t, ok, "a candle must be relayed")
		// The candle is stamped with its open in seconds, an inverse contract's base volume is its turnover, and a candle
		// still forming is marked partial
		exp := kline.Item{
			Exchange: ex.Name,
			Pair:     futuresInversePair,
			Asset:    asset.Futures,
			Interval: kline.FifteenMin,
			Candles: []kline.Candle{{
				Time:             time.Unix(1792842300, 0),
				Open:             80479.5,
				High:             80490.8,
				Low:              80479.4,
				Close:            80490.8,
				Volume:           0.0708234966710365,
				ValidationIssues: kline.PartialCandle,
			}},
		}
		assert.Equal(t, exp, item, "the candle should be relayed")

		err := ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(strings.Replace(futuresKlineFrame, `"interval":"Min15"`, `"interval":"Min3"`, 1)))
		assert.ErrorIs(t, err, kline.ErrInvalidInterval, "an unknown interval should be refused")
	})

	t.Run("fill", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresFillFrame)), "WebsocketFuturesHandleData must not error")
		requireNothingRelayed(t, ex)

		subscribeFuturesPrivate(t, ex, subscription.MyTradesChannel)
		require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresFillFrame)), "WebsocketFuturesHandleData must not error")
		fills, ok := relayed(t, ex).([]fill.Data)
		require.True(t, ok, "fills must be relayed")
		// Closing a long is a sell, and the amount is in contracts
		exp := []fill.Data{{
			ID:            "987654321",
			Timestamp:     time.UnixMilli(1760942212000),
			Exchange:      ex.Name,
			AssetType:     asset.Futures,
			CurrencyPair:  futuresTradablePair,
			Side:          order.Sell,
			OrderID:       "123456789",
			ClientOrderID: "ext_001",
			TradeID:       "987654321",
			Price:         45000.5,
			Amount:        10,
		}}
		assert.Equal(t, exp, fills, "the fill should be relayed")
	})

	t.Run("position", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresPositionFrame)), "WebsocketFuturesHandleData must not error")
		requireNothingRelayed(t, ex)

		subscribeFuturesPrivate(t, ex, FuturesPositionsChannel)
		require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresPositionFrame)), "WebsocketFuturesHandleData must not error")
		positions, ok := relayed(t, ex).([]futures.Position)
		require.True(t, ok, "the position must be relayed")
		// A held short position is open and short, its size is in contracts, its unrealised PNL is read from the push's
		// pnl, its version is the update ID and its collateral the settlement currency
		exp := []futures.Position{{
			Exchange:                  ex.Name,
			Asset:                     asset.Futures,
			Pair:                      futuresTradablePair,
			Underlying:                currency.BTC,
			CollateralCurrency:        currency.USDT,
			Leverage:                  decimal.NewFromInt(20),
			PositionMargin:            decimal.MustFromFloat(2250.025),
			InitialMarginRequirement:  decimal.MustFromFloat(2250.025),
			EstimatedLiquidationPrice: decimal.MustFromFloat(50000),
			UpdateID:                  3,
			RealisedPNL:               decimal.MustFromFloat(-0.1125),
			UnrealisedPNL:             decimal.MustFromFloat(100.5),
			Status:                    order.Open,
			OpeningDate:               time.UnixMilli(1760942212000),
			OpeningPrice:              decimal.MustFromFloat(45000.5),
			OpeningDirection:          order.Short,
			LatestSize:                decimal.MustFromFloat(10),
			LatestDirection:           order.Short,
			LastUpdated:               time.UnixMilli(1760942213000),
		}}
		assert.Equal(t, exp, positions, "the open position should be relayed")

		require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(strings.Replace(futuresPositionFrame, `"state":1`, `"state":3`, 1))),
			"WebsocketFuturesHandleData must not error for a closed position")
		positions, ok = relayed(t, ex).([]futures.Position)
		require.True(t, ok, "the closed position must be relayed")
		// A closed position closes at its update
		exp[0].Status, exp[0].CloseDate = order.Closed, time.UnixMilli(1760942213000)
		assert.Equal(t, exp, positions, "the closed position should be relayed")
	})
}

// futuresReplyConnection answers each futures websocket request with a recorded reply, or fails to send it with err,
// and routes no reply to a waiting request
type futuresReplyConnection struct {
	websocket.Connection
	mu    sync.Mutex
	reply []byte
	err   error
	sent  []*WebsocketFuturesRequest
	// late, when set, holds the late reply which is awaited until the test sends it, and lateErr refuses the wait.
	// awaiting, when set, is closed once the late reply is awaited
	late     chan websocket.MatchedResponse
	lateErr  error
	awaited  bool
	awaiting chan struct{}
	// hold, when set, holds each request until it is closed
	hold chan struct{}
	// markerReplies are the replies to each marker request, which is recorded in markers with the number of replies
	// its wait expects in expected
	markerReplies [][]byte
	markers       []*WebsocketFuturesRequest
	expected      []int
}

// SendMessageReturnResponsesWithInspector records a marker request and returns the recorded replies to it, or fails to
// send it
func (c *futuresReplyConnection) SendMessageReturnResponsesWithInspector(_ context.Context, _ request.EndpointLimit, signature, payload any, expected int, _ websocket.Inspector) ([][]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if signature != futuresWebsocketReplySignature {
		return nil, common.ErrTypeAssertFailure
	}
	req, ok := payload.(*WebsocketFuturesRequest)
	if !ok {
		return nil, common.ErrTypeAssertFailure
	}
	c.expected = append(c.expected, expected)
	if c.err != nil {
		return nil, c.err
	}
	c.markers = append(c.markers, req)
	return c.markerReplies, nil
}

// SendMessageReturnResponse records a request and returns the recorded reply, or fails to send it
func (c *futuresReplyConnection) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, signature, payload any) ([]byte, error) {
	c.mu.Lock()
	hold := c.hold
	c.mu.Unlock()
	if hold != nil {
		<-hold
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if signature != futuresWebsocketReplySignature {
		return nil, common.ErrTypeAssertFailure
	}
	req, ok := payload.(*WebsocketFuturesRequest)
	if !ok {
		return nil, common.ErrTypeAssertFailure
	}
	if c.err != nil {
		return nil, c.err
	}
	c.sent = append(c.sent, req)
	return c.reply, nil
}

// MatchReturnResponses records that a late reply is waited for, and ends the wait at once without one
func (c *futuresReplyConnection) MatchReturnResponses(context.Context, any, int) (<-chan websocket.MatchedResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.awaited = true
	if c.lateErr != nil {
		return nil, c.lateErr
	}
	if c.late != nil {
		if c.awaiting != nil {
			close(c.awaiting)
			c.awaiting = nil
		}
		return c.late, nil
	}
	matched := make(chan websocket.MatchedResponse, 1)
	matched <- websocket.MatchedResponse{Err: websocket.ErrSignatureTimeout}
	return matched, nil
}

// IncomingWithData reports that no request waits for a reply
func (*futuresReplyConnection) IncomingWithData(any, []byte) bool {
	return false
}

func TestFuturesWebsocketRequest(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	req := &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}}
	// The cases share ex and req in order, so each also checks that a request keeps nothing from the last
	for name, tc := range map[string]struct {
		reply string
		err   error
	}{
		"acknowledged":                 {`{"channel":"rs.sub.ticker","data":"success","ts":1792842495741}`, nil},
		"refused":                      {`{"channel":"rs.error","data":"Contract [FOO_BAR] not exists","ts":1792842495741}`, errFuturesWebsocketRejected},
		"unsuccessful acknowledgement": {`{"channel":"rs.sub.ticker","data":"Not support interval","ts":1792842495741}`, errFuturesWebsocketRejected},
		"another channel acknowledged": {`{"channel":"rs.sub.deal","data":"success","ts":1792842495741}`, errFuturesWebsocketRejected},
		"refusal object":               {`{"channel":"rs.sub.ticker","data":{"code":1,"msg":"refused"},"ts":1792842495741}`, errFuturesWebsocketRejected},
		"null":                         {`null`, common.ErrNoResponse},
	} {
		t.Run(name, func(t *testing.T) {
			err := ex.futuresWebsocketRequest(t.Context(), &futuresReplyConnection{reply: []byte(tc.reply)}, req)
			assert.ErrorIsf(t, err, tc.err, "the reply %s should be read", tc.reply)
		})
	}
	assert.Error(t, ex.futuresWebsocketRequest(t.Context(), &futuresReplyConnection{reply: []byte(`{"channel":`)}, req), "a malformed reply should be refused")

	t.Run("holds the request lock", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		req := &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}}
		acknowledging := &futuresReplyConnection{reply: []byte(`{"channel":"rs.sub.ticker","data":"success"}`)}
		ctx := &lockCheckingContext{Context: t.Context(), lock: &ex.futuresConnectionState(acknowledging).mu}
		require.NoError(t, ex.futuresWebsocketRequest(ctx, acknowledging, req), "futuresWebsocketRequest must not error")
		require.NotEmpty(t, ctx.held, "the request's context must be checked before it is sent")
		assert.NotContains(t, ctx.held, false, "the request's context should be checked under the request lock, so that one ending while it waits for the lock is not sent")

		synctest.Test(t, func(t *testing.T) {
			ex := newFuturesWebsocketTestExchange(t)
			conn := &futuresReplyConnection{err: websocket.ErrSignatureTimeout, late: make(chan websocket.MatchedResponse, 1)}
			done := make(chan error, 1)
			// Taken before the request starts, as a lookup waiting on a lock the request holds would hang the bubble
			state := ex.futuresConnectionState(conn)
			go func() { done <- ex.futuresWebsocketRequest(t.Context(), conn, req) }()
			synctest.Wait()
			held := !state.mu.TryLock()
			if !held {
				state.mu.Unlock()
			}
			conn.late <- websocket.MatchedResponse{Err: websocket.ErrSignatureTimeout}
			require.ErrorIs(t, <-done, websocket.ErrSignatureTimeout, "futuresWebsocketRequest must return the timeout")
			assert.True(t, conn.awaited, "the late reply should be awaited")
			assert.True(t, held, "the request lock should be held until the late reply is read, so that no other request is sent meanwhile")
			assert.Equal(t, map[websocket.Connection]int{conn: 1}, futuresRepliesOwed(ex), "a connection whose late reply did not come should owe it")
		})

		synctest.Test(t, func(t *testing.T) {
			ex := newFuturesWebsocketTestExchange(t)
			hold := make(chan struct{})
			conn := &futuresReplyConnection{reply: []byte(`{"channel":"rs.sub.ticker","data":"success"}`), hold: hold}
			state := ex.futuresConnectionState(conn)
			done := make(chan error, 1)
			go func() { done <- ex.futuresWebsocketRequest(t.Context(), conn, req) }()
			synctest.Wait()
			held := !state.mu.TryLock()
			if !held {
				state.mu.Unlock()
			}
			close(hold)
			require.NoError(t, <-done, "futuresWebsocketRequest must not error")
			assert.True(t, held, "the request lock should be held while the request waits for its reply, so that no other request is sent meanwhile")
		})
	})

	t.Run("keeps its connection's count exact across concurrent requests", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		req := &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}}
		conn := new(unansweredFuturesConnection)
		const callers, requests = 4, 250
		var wg sync.WaitGroup
		for range callers {
			wg.Go(func() {
				for range requests {
					assert.ErrorIs(t, ex.futuresWebsocketRequest(t.Context(), conn, req), websocket.ErrSignatureTimeout, "each request should go unanswered")
				}
			})
		}
		finished := make(chan struct{})
		go func() {
			wg.Wait()
			close(finished)
		}()
		select {
		case <-finished:
		case <-time.After(time.Minute):
			require.FailNow(t, "every concurrent request must return")
		}
		assert.Equal(t, int64(callers*requests-1), conn.markers.Load(), "every request after the first should resync the reply the one before it owes")
		assert.Equal(t, map[websocket.Connection]int{conn: 1}, futuresRepliesOwed(ex), "only the last request's reply should be owed")
	})

	t.Run("holds its own connection's lock alone", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		req := &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}}
		awaiting := make(chan struct{})
		waiting := &futuresReplyConnection{err: websocket.ErrSignatureTimeout, late: make(chan websocket.MatchedResponse, 1), awaiting: awaiting}
		done := make(chan error, 1)
		go func() { done <- ex.futuresWebsocketRequest(t.Context(), waiting, req) }()
		select {
		case <-awaiting:
		case err := <-done:
			require.Failf(t, "futuresWebsocketRequest returned early", "the request must wait for its late reply, not return %v", err)
		case <-time.After(time.Minute):
			require.FailNow(t, "the request must reach its wait for a late reply")
		}
		// A request held up by the other connection's lock is let through when the late reply comes, a minute later
		release := time.AfterFunc(time.Minute, func() { waiting.late <- websocket.MatchedResponse{Err: websocket.ErrSignatureTimeout} })
		other := &futuresReplyConnection{reply: []byte(`{"channel":"rs.sub.ticker","data":"success"}`)}
		require.NoError(t, ex.futuresWebsocketRequest(t.Context(), other, req), "a request on another connection must be answered")
		if assert.True(t, release.Stop(), "a request on another connection should not wait for one awaiting its late reply") {
			waiting.late <- websocket.MatchedResponse{Err: websocket.ErrSignatureTimeout}
		}
		select {
		case err := <-done:
			require.ErrorIs(t, err, websocket.ErrSignatureTimeout, "futuresWebsocketRequest must return the timeout")
		case <-time.After(time.Minute):
			require.FailNow(t, "the request must return once its late reply is read")
		}
		assert.Equal(t, map[websocket.Connection]int{waiting: 1}, futuresRepliesOwed(ex), "only the connection whose late reply did not come should owe it")
	})

	t.Run("send failure", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		errSendFailed := errors.New("websocket connection: cannot send message")
		conn := &futuresReplyConnection{err: errSendFailed}
		err := ex.futuresWebsocketRequest(t.Context(), conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}})
		require.ErrorIs(t, err, errSendFailed, "a request which could not be sent must return why")
		assert.False(t, conn.awaited, "a request which was never sent should not wait for a reply")
		assert.Empty(t, futuresRepliesOwed(ex), "a request which was never sent should leave no reply owed")
	})

	t.Run("late reply wait refused", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		conn := &futuresReplyConnection{err: websocket.ErrSignatureTimeout, lateErr: errors.New("signature collision")}
		err := ex.futuresWebsocketRequest(t.Context(), conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}})
		require.ErrorIs(t, err, websocket.ErrSignatureTimeout, "futuresWebsocketRequest must return the timeout")
		assert.True(t, conn.awaited, "the late reply should be waited for")
		assert.Equal(t, 1, futuresRepliesOwed(ex)[conn], "a connection whose late reply could not be waited for should owe it")
	})

	t.Run("owed reply", func(t *testing.T) {
		t.Parallel()
		const (
			owed    = `{"channel":"rs.sub.ticker","data":"success","ts":1792842495741}`
			refusal = `{"channel":"rs.error","data":"Contract [gct-resync-1] not exists","ts":1792842495741}`
		)
		errSendFailed := errors.New("websocket connection: cannot send message")
		req := &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}}
		for _, tc := range []struct {
			name    string
			replies []string
			sendErr error
			err     error
			// owedAfter is how many replies the connection owes after a failed marker: the one it owed, and the
			// marker's own when the marker was sent
			owedAfter int
		}{
			{name: "marker answered", replies: []string{refusal}},
			{name: "marker answered after the reply owed", replies: []string{owed, refusal}},
			{name: "marker refused in other words", replies: []string{`{"channel":"rs.error","data":"marker gct-resync-1 refused"}`}},
			{name: "marker unanswered after its replies", replies: []string{owed, owed}, err: errFuturesWebsocketOutOfStep, owedAfter: 2},
			{name: "no replies", err: errFuturesWebsocketOutOfStep, owedAfter: 2},
			{name: "marker not sent", sendErr: errSendFailed, err: errSendFailed, owedAfter: 1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFuturesWebsocketTestExchange(t)
				conn := &futuresReplyConnection{reply: []byte(`{"channel":"rs.sub.ticker","data":"success"}`), err: tc.sendErr}
				for _, r := range tc.replies {
					conn.markerReplies = append(conn.markerReplies, []byte(r))
				}
				ex.futuresConnectionState(conn).repliesOwed = 1
				err := ex.futuresWebsocketRequest(t.Context(), conn, req)
				if tc.err != nil {
					require.ErrorIs(t, err, tc.err, "futuresWebsocketRequest must return why the connection is out of step")
					assert.ErrorIs(t, err, errFuturesWebsocketOutOfStep, "the request should fail as out of step")
					assert.Empty(t, conn.sent, "a request should not be sent until a marker is answered")
					assert.Equal(t, tc.owedAfter, futuresRepliesOwed(ex)[conn], "the connection should still owe its reply, and a marker's when one was sent")
					return
				}
				require.NoError(t, err, "futuresWebsocketRequest must not error")
				assert.Equal(t, []*WebsocketFuturesRequest{req}, conn.sent, "the request should be sent once a marker is answered")
				assert.Empty(t, futuresRepliesOwed(ex), "no reply should be owed once a marker is answered")
				exp := []*WebsocketFuturesRequest{{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "gct-resync-1"}}}
				assert.Equal(t, exp, conn.markers, "the marker should be a subscription to a symbol no contract has")
			})
		}
	})
}

type lockCheckingContext struct {
	context.Context //nolint:containedctx // lockCheckingContext implements context.Context
	lock            *sync.Mutex
	held            []bool
}

func (c *lockCheckingContext) Err() error {
	held := !c.lock.TryLock()
	if !held {
		c.lock.Unlock()
	}
	c.held = append(c.held, held)
	return c.Context.Err()
}

// futuresLateReplyVenue is a mock futures venue which acknowledges a BTC_USDT subscription once its hold has passed, or
// never when the hold is negative, refuses any other once the symbol's delay has passed, never when that is negative,
// and records the symbols it is sent
type futuresLateReplyVenue struct {
	hold    time.Duration
	delays  map[string]time.Duration
	mu      sync.Mutex
	symbols []string
}

func (v *futuresLateReplyVenue) handle(_ testing.TB, msg []byte, conn *gws.Conn) error {
	var req struct {
		Param WebsocketFuturesChannelParams `json:"param"`
	}
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	v.mu.Lock()
	v.symbols = append(v.symbols, req.Param.Symbol)
	v.mu.Unlock()
	if delay, ok := v.delays[req.Param.Symbol]; ok {
		if delay < 0 {
			return nil
		}
		time.Sleep(delay)
	}
	reply := `{"channel":"rs.error","data":"Contract [` + req.Param.Symbol + `] not exists","ts":1792842495741}`
	if req.Param.Symbol == "BTC_USDT" {
		if v.hold < 0 {
			return nil
		}
		time.Sleep(v.hold)
		reply = `{"channel":"rs.sub.ticker","data":"success","ts":1792842495741}`
	}
	return conn.WriteMessage(gws.TextMessage, []byte(reply))
}

func (v *futuresLateReplyVenue) sent() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.symbols)
}

// dialFuturesTestVenue dials a venue in memory, so that a synctest bubble can wait on the connection, and handles what
// the venue sends until the test ends
func dialFuturesTestVenue(t *testing.T, ex *Exchange, handle mockws.WsMockFunc) websocket.Connection {
	t.Helper()
	server, dialer := mockws.NewTestServer(t, mockws.CurryWsMockUpgrader(t, handle))
	conn, err := ex.Websocket.CreateTestConnection(futuresPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	conn.SetURL("ws" + strings.TrimPrefix(server.URL, "http"))
	require.NoError(t, conn.Dial(t.Context(), dialer, http.Header{}, nil), "Dial must not error")
	read := make(chan struct{})
	go func() {
		defer close(read)
		for resp := conn.ReadMessage(); resp.Raw != nil; resp = conn.ReadMessage() {
			assert.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), conn, resp.Raw), "WebsocketFuturesHandleData should not error")
		}
	}()
	t.Cleanup(func() {
		assert.NoError(t, conn.Shutdown(), "Shutdown should not error")
		<-read
	})
	return conn
}

func TestFuturesWebsocketRequestAwaitsLateReply(t *testing.T) {
	t.Parallel()
	limit := exchange.DefaultWebsocketResponseMaxLimit
	for _, tc := range []struct {
		name string
		// hold is how long the venue holds the first request's reply; a negative hold is never answered
		hold time.Duration
		ctx  func(context.Context) (context.Context, context.CancelFunc)
		err  error
		// returned is when the first request returns, on the test's fake clock
		returned time.Duration
		sent     []string
	}{
		{
			name: "deadline passed while waiting",
			hold: 200 * time.Millisecond,
			ctx: func(ctx context.Context) (context.Context, context.CancelFunc) {
				return context.WithTimeout(ctx, 50*time.Millisecond)
			},
			err:      context.DeadlineExceeded,
			returned: 200 * time.Millisecond,
			sent:     []string{"BTC_USDT", "BTC_USD"},
		},
		{
			name: "cancelled while waiting",
			hold: 200 * time.Millisecond,
			ctx: func(ctx context.Context) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(ctx)
				time.AfterFunc(50*time.Millisecond, cancel)
				return ctx, cancel
			},
			err:      context.Canceled,
			returned: 200 * time.Millisecond,
			sent:     []string{"BTC_USDT", "BTC_USD"},
		},
		{
			name:     "signature timeout",
			hold:     limit + time.Second,
			ctx:      context.WithCancel,
			err:      websocket.ErrSignatureTimeout,
			returned: limit + time.Second,
			sent:     []string{"BTC_USDT", "BTC_USD"},
		},
		{
			name:     "reply after both wait limits",
			hold:     2*limit + time.Second,
			ctx:      context.WithCancel,
			err:      websocket.ErrSignatureTimeout,
			returned: 2 * limit,
			sent:     []string{"BTC_USDT", "gct-resync-1", "BTC_USD"},
		},
		{
			name: "never answered",
			hold: -1,
			ctx: func(ctx context.Context) (context.Context, context.CancelFunc) {
				return context.WithTimeout(ctx, 50*time.Millisecond)
			},
			err:      context.DeadlineExceeded,
			returned: 50*time.Millisecond + limit,
			sent:     []string{"BTC_USDT", "gct-resync-1", "BTC_USD"},
		},
		{
			name: "deadline passed before sending",
			hold: 200 * time.Millisecond,
			ctx: func(ctx context.Context) (context.Context, context.CancelFunc) {
				return context.WithDeadline(ctx, time.Now())
			},
			err:  context.DeadlineExceeded,
			sent: []string{"BTC_USD"},
		},
		{
			name: "cancelled before sending",
			hold: 200 * time.Millisecond,
			ctx: func(ctx context.Context) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				return ctx, cancel
			},
			err:  context.Canceled,
			sent: []string{"BTC_USD"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				// Lets a held reply and a wait for one end, should a failed assertion end the case early
				defer synctest.Sleep(time.Minute)
				ex := newFuturesWebsocketTestExchange(t)
				require.Equal(t, limit, ex.WebsocketResponseMaxLimit, "the test config must set the default response limit")
				venue := &futuresLateReplyVenue{hold: tc.hold}
				conn := dialFuturesTestVenue(t, ex, venue.handle)
				ctx, cancel := tc.ctx(t.Context())
				defer cancel()
				start := time.Now()
				err := ex.futuresWebsocketRequest(ctx, conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}})
				require.ErrorIs(t, err, tc.err, "the unacknowledged request must return why it was given up on")
				assert.Equal(t, tc.returned, time.Since(start), "the request should return once its late reply is read, or its wait for one ends")
				err = ex.futuresWebsocketRequest(t.Context(), conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USD"}})
				assert.ErrorIs(t, err, errFuturesWebsocketRejected, "the next request should be answered by its own reply, not the late one")
				assert.Equal(t, tc.sent, venue.sent(), "only a request whose context has not ended should be sent, after a marker while a reply is owed")
				assert.Empty(t, futuresRepliesOwed(ex), "no reply should be owed once the next request is answered")
			})
		})
	}
}

func TestFuturesWebsocketRequestResync(t *testing.T) {
	t.Parallel()
	limit := exchange.DefaultWebsocketResponseMaxLimit
	for _, tc := range []struct {
		name string
		// gap is how long after the first marker's wait ends the next request is sent; the venue refuses that marker a
		// second after its wait ends, and the next a second after it is sent, so that their refusals arrive apart
		gap time.Duration
	}{
		{name: "late marker refusal during the next marker's wait"},
		{name: "late marker refusal while no request waits", gap: 2 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				defer synctest.Sleep(time.Minute)
				ex := newFuturesWebsocketTestExchange(t)
				venue := &futuresLateReplyVenue{hold: -1, delays: map[string]time.Duration{"gct-resync-1": 2*limit + time.Second, "gct-resync-2": time.Second}}
				conn := dialFuturesTestVenue(t, ex, venue.handle)
				subscribe := func(symbol string) error {
					return ex.futuresWebsocketRequest(t.Context(), conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: symbol}})
				}
				require.ErrorIs(t, subscribe("BTC_USDT"), websocket.ErrSignatureTimeout, "an unanswered request must time out")
				start := time.Now()
				err := subscribe("BTC_USD")
				require.ErrorIs(t, err, errFuturesWebsocketOutOfStep, "a request must fail when the marker before it goes unanswered")
				assert.ErrorIs(t, err, websocket.ErrSignatureTimeout, "the failure should carry the marker's timeout")
				assert.Equal(t, 2*limit, time.Since(start), "an unanswered marker should be waited for twice the response limit")
				assert.Equal(t, []string{"BTC_USDT", "gct-resync-1"}, venue.sent(), "a request should not be sent until a marker is answered")
				assert.Equal(t, 2, futuresRepliesOwed(ex)[conn], "the connection should owe the request's reply and the marker's")
				synctest.Sleep(tc.gap)
				err = subscribe("BTC_USD")
				assert.ErrorIs(t, err, errFuturesWebsocketRejected, "a request should be answered once a later marker is")
				assert.ErrorContains(t, err, "Contract [BTC_USD]", "the request should be answered by its own reply, not an earlier marker's")
				assert.Equal(t, []string{"BTC_USDT", "gct-resync-1", "gct-resync-2", "BTC_USD"}, venue.sent(), "each resync should send a marker of its own")
				assert.Empty(t, futuresRepliesOwed(ex), "no reply should be owed once a marker is answered")
			})
		})
	}
}

func TestFuturesWebsocketRequestResyncContextEnds(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		defer synctest.Sleep(time.Minute)
		ex := newFuturesWebsocketTestExchange(t)
		venue := &futuresLateReplyVenue{hold: -1, delays: map[string]time.Duration{"gct-resync-1": -1}}
		conn := dialFuturesTestVenue(t, ex, venue.handle)
		err := ex.futuresWebsocketRequest(t.Context(), conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}})
		require.ErrorIs(t, err, websocket.ErrSignatureTimeout, "an unanswered request must time out")
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		start := time.Now()
		err = ex.futuresWebsocketRequest(ctx, conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USD"}})
		require.ErrorIs(t, err, errFuturesWebsocketOutOfStep, "a request whose context ends during the marker's wait must fail as out of step")
		assert.ErrorIs(t, err, context.DeadlineExceeded, "the failure should carry the context's error")
		assert.Equal(t, 50*time.Millisecond, time.Since(start), "the request should return when its context ends")
		assert.Equal(t, []string{"BTC_USDT", "gct-resync-1"}, venue.sent(), "a request should not be sent until a marker is answered")
		assert.Equal(t, 2, futuresRepliesOwed(ex)[conn], "the connection should owe the request's reply and the marker's")
	})
}

func TestResyncFuturesReplies(t *testing.T) {
	t.Parallel()
	const (
		owed    = `{"channel":"rs.sub.ticker","data":"success","ts":1792842495741}`
		refusal = `{"channel":"rs.error","data":"Contract [gct-resync-1] not exists","ts":1792842495741}`
	)
	errSendFailed := errors.New("websocket connection: cannot send message")
	marker := []*WebsocketFuturesRequest{{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "gct-resync-1"}}}
	for _, tc := range []struct {
		name    string
		replies []string
		sendErr error
		err     error
		// owedAfter is how many replies the connection owes afterwards, from the two it owed before
		owedAfter int
		markers   []*WebsocketFuturesRequest
	}{
		{name: "marker answered after the replies owed", replies: []string{owed, owed, refusal}, markers: marker},
		{name: "marker unanswered after the replies owed", replies: []string{owed, owed}, err: errFuturesWebsocketOutOfStep, owedAfter: 3, markers: marker},
		{name: "no replies", err: errFuturesWebsocketOutOfStep, owedAfter: 3, markers: marker},
		{name: "marker's wait ended", sendErr: websocket.ErrSignatureTimeout, err: websocket.ErrSignatureTimeout, owedAfter: 3},
		{name: "marker not sent", sendErr: errSendFailed, err: errSendFailed, owedAfter: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newFuturesWebsocketTestExchange(t)
			conn := &futuresReplyConnection{err: tc.sendErr}
			for _, r := range tc.replies {
				conn.markerReplies = append(conn.markerReplies, []byte(r))
			}
			state := ex.futuresConnectionState(conn)
			state.repliesOwed = 2
			err := ex.resyncFuturesReplies(t.Context(), conn, state)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "resyncFuturesReplies must return why the connection is still out of step")
				assert.ErrorIs(t, err, errFuturesWebsocketOutOfStep, "the connection should be reported out of step")
			} else {
				require.NoError(t, err, "resyncFuturesReplies must not error")
			}
			assert.Equal(t, tc.owedAfter, state.repliesOwed, "the connection should owe the replies still to come, and the marker's when it was sent and went unanswered")
			assert.Equal(t, []int{3}, conn.expected, "the marker's wait should expect the replies owed and its own")
			assert.Equal(t, tc.markers, conn.markers, "the marker should be a subscription to a symbol no contract has")
		})
	}

	t.Run("an earlier marker's refusal does not answer a later marker", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		conn := &futuresReplyConnection{markerReplies: [][]byte{[]byte(refusal)}}
		state := ex.futuresConnectionState(conn)
		require.NoError(t, ex.resyncFuturesReplies(t.Context(), conn, state), "resyncFuturesReplies must not error when its marker is answered")
		assert.ErrorIs(t, ex.resyncFuturesReplies(t.Context(), conn, state), errFuturesWebsocketOutOfStep, "a later marker should not be answered by an earlier one's refusal")
		exp := []*WebsocketFuturesRequest{marker[0], {Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "gct-resync-2"}}}
		assert.Equal(t, exp, conn.markers, "each marker should be numbered afresh")
		assert.Equal(t, 1, state.repliesOwed, "the later marker, unanswered, should be owed")
	})
}

func TestFuturesConnectionState(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	undialled, err := ex.Websocket.CreateTestConnection(futuresPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	_, ok := undialled.(interface{ IsConnected() bool })
	require.True(t, ok, "a futures connection must report whether it is connected, so that a closed one is forgotten")
	server, dialer := mockws.NewTestServer(t, mockws.CurryWsMockUpgrader(t, new(futuresLateReplyVenue).handle))
	closed, err := ex.Websocket.CreateTestConnection(futuresPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	closed.SetURL("ws" + strings.TrimPrefix(server.URL, "http"))
	require.NoError(t, closed.Dial(t.Context(), dialer, http.Header{}, nil), "Dial must not error")
	require.NoError(t, closed.Shutdown(), "Shutdown must not error")
	open := dialFuturesTestVenue(t, ex, new(futuresLateReplyVenue).handle)
	unknown := &futuresReplyConnection{}
	for _, conn := range []websocket.Connection{undialled, closed, open, unknown} {
		ex.futuresConnectionState(conn).repliesOwed++
	}
	ex.futuresConnectionState(open).repliesOwed += 2
	assert.Equal(t, map[websocket.Connection]int{open: 3, unknown: 1}, futuresRepliesOwed(ex),
		"a connection never dialled or shut down should be forgotten, and an open one or one which does not report it kept, owing every reply recorded")
	state := ex.futuresConnectionState(open)
	assert.Same(t, state, ex.futuresConnectionState(open), "a connection should keep one request state, and so one request lock")
	ex.futuresConnectionState(closed).repliesOwed++
	assert.Zero(t, ex.futuresConnectionState(closed).repliesOwed, "a closed connection's replies should be forgotten when it is asked for again, as they can no longer arrive")
}

// unansweredFuturesConnection leaves every request unanswered, even after a wait for a late reply, answers each marker
// request with the venue's refusal of its symbol, and counts the markers
type unansweredFuturesConnection struct {
	websocket.Connection
	markers atomic.Int64
}

// SendMessageReturnResponse leaves the request unanswered
func (*unansweredFuturesConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return nil, websocket.ErrSignatureTimeout
}

// MatchReturnResponses ends the wait for a late reply at once without one
func (*unansweredFuturesConnection) MatchReturnResponses(context.Context, any, int) (<-chan websocket.MatchedResponse, error) {
	matched := make(chan websocket.MatchedResponse, 1)
	matched <- websocket.MatchedResponse{Err: websocket.ErrSignatureTimeout}
	return matched, nil
}

// SendMessageReturnResponsesWithInspector counts a marker request and answers it with its refusal
func (c *unansweredFuturesConnection) SendMessageReturnResponsesWithInspector(_ context.Context, _ request.EndpointLimit, _, payload any, _ int, _ websocket.Inspector) ([][]byte, error) {
	req, ok := payload.(*WebsocketFuturesRequest)
	if !ok {
		return nil, common.ErrTypeAssertFailure
	}
	params, ok := req.Param.(*WebsocketFuturesChannelParams)
	if !ok {
		return nil, common.ErrTypeAssertFailure
	}
	c.markers.Add(1)
	return [][]byte{[]byte(`{"channel":"rs.error","data":"Contract [` + params.Symbol + `] not exists"}`)}, nil
}

// futuresRepliesOwed returns each futures connection's count of the replies it may still owe, where that isn't zero
func futuresRepliesOwed(ex *Exchange) map[websocket.Connection]int {
	ex.futuresConnectionsMu.Lock()
	states := maps.Clone(ex.futuresConnections)
	ex.futuresConnectionsMu.Unlock()
	owed := make(map[websocket.Connection]int)
	for conn, state := range states {
		state.mu.Lock()
		if state.repliesOwed != 0 {
			owed[conn] = state.repliesOwed
		}
		state.mu.Unlock()
	}
	return owed
}

func TestFuturesWebsocketRequestRateLimited(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		ctx  func(context.Context) (context.Context, context.CancelFunc)
		err  error
		// returned is when the request returns, on the test's fake clock
		returned time.Duration
	}{
		{
			name: "cancelled while rate limited",
			ctx: func(ctx context.Context) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(ctx)
				time.AfterFunc(50*time.Millisecond, cancel)
				return ctx, cancel
			},
			err:      context.Canceled,
			returned: 50 * time.Millisecond,
		},
		{
			name: "deadline before its turn",
			ctx: func(ctx context.Context) (context.Context, context.CancelFunc) {
				return context.WithTimeout(ctx, 50*time.Millisecond)
			},
			err: context.DeadlineExceeded,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				defer synctest.Sleep(time.Minute)
				ex := newFuturesWebsocketTestExchange(t)
				venue := &futuresLateReplyVenue{}
				conn := dialFuturesTestVenue(t, ex, venue.handle)
				// The first request takes the connection's rate limit token, so the next waits about 100 ms for its turn
				require.NoError(t, ex.futuresWebsocketRequest(t.Context(), conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}}),
					"futuresWebsocketRequest must not error")
				ctx, cancel := tc.ctx(t.Context())
				defer cancel()
				start := time.Now()
				err := ex.futuresWebsocketRequest(ctx, conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USD"}})
				require.ErrorIs(t, err, tc.err, "a request whose context ends before its turn must return why")
				assert.Equal(t, tc.returned, time.Since(start), "a request which was never sent should not wait for a reply")
				assert.Equal(t, []string{"BTC_USDT"}, venue.sent(), "a request whose context ends before its turn should not be sent")
				assert.Empty(t, futuresRepliesOwed(ex), "a request which was never sent should leave no reply owed")
			})
		})
	}
}

func TestWebsocketFuturesLogin(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	ex.API.AuthenticatedWebsocketSupport = true
	conn := &futuresReplyConnection{reply: []byte(`{"channel":"rs.login","data":"success","ts":"1587442022003"}`)}
	require.ErrorIs(t, ex.websocketFuturesLogin(t.Context(), conn), exchange.ErrCredentialsAreEmpty, "a login without credentials must be refused before it is sent")
	require.Empty(t, conn.sent, "a login without credentials must not be sent")

	ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
	for range 2 {
		require.NoError(t, ex.websocketFuturesLogin(t.Context(), conn), "websocketFuturesLogin must not error")
	}
	require.Len(t, conn.sent, 1, "a connection must be logged in once")
	params, ok := conn.sent[0].Param.(*WebsocketFuturesLoginParams)
	require.True(t, ok, "the login parameters must be sent")
	mac, err := crypto.GetHMAC(crypto.HashSHA256, []byte("test-key"+params.RequestTime), []byte("test-secret"))
	require.NoError(t, err, "GetHMAC must not error")
	exp := &WebsocketFuturesRequest{
		Method: futuresMethodLogin,
		Param:  &WebsocketFuturesLoginParams{APIKey: "test-key", RequestTime: params.RequestTime, Signature: hex.EncodeToString(mac)},
	}
	assert.Equal(t, exp, conn.sent[0], "the login should be signed over the key and the request time")

	refused := &futuresReplyConnection{reply: []byte(`{"channel":"rs.error","data":"Apikey doesn't exist!","ts":1792842495741}`)}
	require.ErrorIs(t, ex.websocketFuturesLogin(t.Context(), refused), errFuturesWebsocketRejected, "a refused login must be returned")
	require.ErrorIs(t, ex.websocketFuturesLogin(t.Context(), refused), errFuturesWebsocketRejected, "a refused login must be tried again")
	assert.Len(t, refused.sent, 2, "a refused login should not be remembered")
}

func TestFuturesChannelRequest(t *testing.T) {
	t.Parallel()
	pair := currency.Pairs{currency.NewBTCUSDT()}
	for _, tc := range []struct {
		name   string
		sub    *subscription.Subscription
		method string
		params WebsocketFuturesChannelParams
	}{
		{"ticker", &subscription.Subscription{Channel: subscription.TickerChannel, Pairs: pair}, "ticker", WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}},
		{"trades", &subscription.Subscription{Channel: subscription.AllTradesChannel, Pairs: pair}, "deal", WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}},
		{"orderbook", &subscription.Subscription{Channel: subscription.OrderbookChannel, Pairs: pair}, "depth.full", WebsocketFuturesChannelParams{Symbol: "BTC_USDT", Limit: 20}},
		{"orderbook of 5 levels", &subscription.Subscription{Channel: subscription.OrderbookChannel, Pairs: pair, Levels: 5}, "depth.full", WebsocketFuturesChannelParams{Symbol: "BTC_USDT", Limit: 5}},
		{"candles", &subscription.Subscription{Channel: subscription.CandlesChannel, Pairs: pair, Interval: kline.FourHour}, "kline", WebsocketFuturesChannelParams{Symbol: "BTC_USDT", Interval: "Hour4"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			method, params, err := futuresChannelRequest(tc.sub)
			require.NoErrorf(t, err, "futuresChannelRequest must not error for %s", tc.sub.Channel)
			assert.Equalf(t, tc.method, method, "%s should be the venue's %s channel", tc.sub.Channel, tc.method)
			assert.Equalf(t, tc.params, *params, "%s should carry its parameters", tc.sub.Channel)
		})
	}
	_, _, err := futuresChannelRequest(&subscription.Subscription{Channel: subscription.TickerChannel})
	assert.ErrorIs(t, err, subscription.ErrNotSinglePair)
	_, _, err = futuresChannelRequest(&subscription.Subscription{Channel: subscription.MyOrdersChannel, Pairs: pair})
	assert.ErrorIs(t, err, subscription.ErrNotSupported)
	_, _, err = futuresChannelRequest(&subscription.Subscription{Channel: subscription.CandlesChannel, Pairs: pair, Interval: kline.TenMin})
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval)
}

func TestFuturesQualifiedChannel(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp string
	}{
		{&subscription.Subscription{Channel: subscription.TickerChannel}, "ticker@BTC_USDT"},
		{&subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 5}, "depth.full@BTC_USDT@5"},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.FourHour}, "kline@BTC_USDT@Hour4"},
		{&subscription.Subscription{Channel: subscription.MyOrdersChannel}, "personal.order"},
	} {
		t.Run(tc.sub.Channel, func(t *testing.T) {
			t.Parallel()
			qualified, err := futuresQualifiedChannel(tc.sub, pair)
			require.NoErrorf(t, err, "futuresQualifiedChannel must not error for %s", tc.sub.Channel)
			assert.Equalf(t, tc.exp, qualified, "%s should be qualified as the venue names it", tc.sub.Channel)
		})
	}
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		err  error
	}{
		{"10 minute candles", &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.TenMin}, kline.ErrUnsupportedInterval},
		{"my wallet", &subscription.Subscription{Channel: subscription.MyWalletChannel}, subscription.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			qualified, err := futuresQualifiedChannel(tc.sub, pair)
			assert.ErrorIsf(t, err, tc.err, "futuresQualifiedChannel should return why the venue does not serve %s", tc.sub.Channel)
			assert.Emptyf(t, qualified, "futuresQualifiedChannel should qualify no channel for %s", tc.sub.Channel)
		})
	}
}

func TestGenerateFuturesSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	ex.Features.Subscriptions = defaultSubscriptions.Clone()

	public, err := ex.generateFuturesPublicSubscriptions()
	require.NoError(t, err, "generateFuturesPublicSubscriptions must not error")
	got := make([]string, 0, len(public))
	for _, s := range public {
		require.Equalf(t, asset.Futures, s.Asset, "%s must be a futures subscription", s.QualifiedChannel)
		require.Lenf(t, s.Pairs, 1, "%s must carry its pair", s.QualifiedChannel)
		got = append(got, s.QualifiedChannel)
	}
	assert.ElementsMatch(t, []string{
		"ticker@BTC_USDT", "ticker@BTC_USD",
		"depth.full@BTC_USDT@20", "depth.full@BTC_USD@20",
		"deal@BTC_USDT", "deal@BTC_USD",
		"kline@BTC_USDT@Min15", "kline@BTC_USD@Min15",
	}, got, "every public channel should be subscribed for each contract")

	private, err := ex.generateFuturesPrivateSubscriptions()
	require.NoError(t, err, "generateFuturesPrivateSubscriptions must not error")
	assert.Empty(t, private, "no private channel should be subscribed without the authenticated websocket")

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	private, err = ex.generateFuturesPrivateSubscriptions()
	require.NoError(t, err, "generateFuturesPrivateSubscriptions must not error")
	got = got[:0]
	for _, s := range private {
		got = append(got, s.QualifiedChannel)
	}
	assert.ElementsMatch(t, []string{"personal.order", "personal.order.deal", "personal.asset", "personal.position"}, got,
		"every private channel should be subscribed once")

	spot, err := ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	for _, s := range spot {
		assert.Equalf(t, asset.Spot, s.Asset, "the spot connection should not carry %s", s.QualifiedChannel)
	}

	t.Run("private channels kept once a spot listen key is refused", func(t *testing.T) {
		t.Parallel()
		listenKeyRequests := make(chan struct{}, 8)
		refusing := newPrivateTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v3/userDataStream" {
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			select {
			case listenKeyRequests <- struct{}{}:
			default:
			}
			// MEXC documents SPOT_ACCOUNT_READ as required for a listen key, which a key made for futures alone lacks
			http.Error(w, `{"code":700007,"msg":"No permission to access the endpoint"}`, http.StatusForbidden)
		})
		var contracts []*FuturesContractResponse
		require.NoError(t, json.Unmarshal([]byte(futuresContractsReply), &contracts), "the contracts must decode")
		refusing.storeFuturesContracts(contracts)
		refusing.Features.Subscriptions = defaultSubscriptions.Clone()
		refusing.API.AuthenticatedWebsocketSupport = true
		refusing.Websocket.SetCanUseAuthenticatedEndpoints(true)
		spotPublic, err := refusing.generatePublicSubscriptions()
		require.NoError(t, err, "generatePublicSubscriptions must not error")
		require.NotEmpty(t, spotPublic, "the public spot channels must be generated")
		spotPrivate, err := refusing.generatePrivateSubscriptions()
		require.NoError(t, err, "generatePrivateSubscriptions must not error")
		require.NotEmpty(t, spotPrivate, "the private spot channels must be generated before a listen key is refused")

		// The refusal holds whether the next private dial fails or not, and across the public dials around it
		require.NoError(t, refusing.WsConnect(t.Context(), &listenKeyTestConn{url: "wss://public"}), "WsConnect must not error")
		require.Error(t, refusing.wsConnectPrivate(t.Context(), &failingDialConn{url: "wss://private"}), "wsConnectPrivate must report its failed dial")
		require.NoError(t, refusing.WsConnect(t.Context(), &listenKeyTestConn{url: "wss://public"}), "WsConnect must not error")
		require.NoError(t, refusing.wsConnectPrivate(t.Context(), &listenKeyTestConn{url: "wss://private"}), "wsConnectPrivate must not error")
		assert.Len(t, listenKeyRequests, 1, "a refused listen key should not be asked for again")

		// The venue would accept every channel here, so a private one sent without a listen key would be stored
		replies := map[string]string{spotPublic[0].QualifiedChannel: `{"id":0,"code":0,"msg":"` + spotPublic[0].QualifiedChannel + `"}`}
		for _, s := range spotPrivate {
			replies[s.QualifiedChannel] = `{"id":0,"code":0,"msg":"` + s.QualifiedChannel + `"}`
		}
		conn := &subscriptionTestConn{replies: replies}
		assert.ErrorIs(t, refusing.Subscribe(t.Context(), conn, subscription.List{spotPublic[0], spotPrivate[0]}), errNoListenKey,
			"a batch carrying a private spot channel should be refused whole")
		assert.NoError(t, refusing.Subscribe(t.Context(), conn, spotPublic[:1]), "a public spot channel should still be subscribed")
		assert.ErrorIs(t, refusing.Subscribe(t.Context(), conn, spotPrivate), errNoListenKey, "the private spot channels should be refused without a listen key")
		unflagged := spotPrivate[0].Clone()
		unflagged.Authenticated = false
		assert.ErrorIs(t, refusing.Subscribe(t.Context(), conn, subscription.List{unflagged}), errNoListenKey,
			"a private spot channel configured without the Authenticated flag should be refused too")
		stored := refusing.Websocket.GetSubscriptions()
		require.Len(t, stored, 1, "only the public spot channel must be subscribed")
		assert.Equal(t, spotPublic[0].QualifiedChannel, stored[0].QualifiedChannel, "only the public spot channel should be subscribed")
		left, err := refusing.generatePrivateSubscriptions()
		require.NoError(t, err, "generatePrivateSubscriptions must not error")
		assert.Empty(t, left, "the private spot channels should be left out once a listen key is refused")
		refusing.Features.Subscriptions = append(defaultSubscriptions.Clone(), &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: unflagged.Channel})
		left, err = refusing.generatePrivateSubscriptions()
		require.NoError(t, err, "generatePrivateSubscriptions must not error")
		assert.Empty(t, left, "a private spot channel configured without the Authenticated flag should be left out too")

		futuresPrivate, err := refusing.generateFuturesPrivateSubscriptions()
		require.NoError(t, err, "generateFuturesPrivateSubscriptions must not error")
		channels := make([]string, 0, len(futuresPrivate))
		for _, s := range futuresPrivate {
			channels = append(channels, s.QualifiedChannel)
		}
		assert.ElementsMatch(t, []string{"personal.order", "personal.order.deal", "personal.asset", "personal.position"}, channels,
			"the private futures channels, which log in with the API key, should be kept")
		venue := new(futuresVenue)
		futuresConn := dialFuturesTestVenue(t, refusing, venue.handle)
		require.NoError(t, refusing.websocketFuturesSubscribePrivate(t.Context(), futuresConn, futuresPrivate), "websocketFuturesSubscribePrivate must not error")
		assert.Equal(t, []string{futuresMethodLogin}, venue.methods(), "the private futures connection should log in")

		// Neither the futures login nor an unsubscription lifts the refusal
		assert.NoError(t, refusing.Unsubscribe(t.Context(), conn, spotPublic[:1]), "the public spot channel should be unsubscribed")
		left, err = refusing.generatePrivateSubscriptions()
		require.NoError(t, err, "generatePrivateSubscriptions must not error")
		assert.Empty(t, left, "the private spot channels should stay left out")
		require.NoError(t, refusing.wsConnectPrivate(t.Context(), &listenKeyTestConn{url: "wss://private"}), "wsConnectPrivate must not error")
		assert.Len(t, listenKeyRequests, 1, "a refused listen key should not be asked for again")
	})

	t.Run("private channels kept once a spot listen key request fails", func(t *testing.T) {
		t.Parallel()
		for _, failure := range []string{"timed out", "connection dropped", "malformed reply"} {
			t.Run(failure, func(t *testing.T) {
				t.Parallel()
				failing := newPrivateTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost || r.URL.Path != "/api/v3/userDataStream" {
						t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					}
					switch failure {
					case "timed out":
						<-r.Context().Done()
					case "connection dropped":
						hijacker, ok := w.(http.Hijacker)
						if !assert.True(t, ok, "the test server should let a handler take its connection") {
							return
						}
						if conn, _, err := hijacker.Hijack(); assert.NoError(t, err, "Hijack should not error") {
							_ = conn.Close()
						}
					case "malformed reply":
						_, _ = w.Write([]byte("<html>"))
					}
				})
				if failure == "timed out" {
					require.NoError(t, failing.Requester.SetHTTPClientTimeout(50*time.Millisecond), "SetHTTPClientTimeout must not error")
				}
				var contracts []*FuturesContractResponse
				require.NoError(t, json.Unmarshal([]byte(futuresContractsReply), &contracts), "the contracts must decode")
				failing.storeFuturesContracts(contracts)
				failing.Features.Subscriptions = defaultSubscriptions.Clone()
				failing.API.AuthenticatedWebsocketSupport = true
				failing.Websocket.SetCanUseAuthenticatedEndpoints(true)
				require.NoError(t, failing.wsConnectPrivate(t.Context(), &listenKeyTestConn{url: "wss://private"}), "wsConnectPrivate must not error")
				left, err := failing.generatePrivateSubscriptions()
				require.NoError(t, err, "generatePrivateSubscriptions must not error")
				assert.Empty(t, left, "the private spot channels should be left out once the listen key request fails")
				futuresPrivate, err := failing.generateFuturesPrivateSubscriptions()
				require.NoError(t, err, "generateFuturesPrivateSubscriptions must not error")
				assert.Len(t, futuresPrivate, 4, "the private futures channels should be kept")
			})
		}
	})

	t.Run("private futures channels kept across flushes once a spot listen key is refused", func(t *testing.T) {
		t.Parallel()
		listenKeyRequests := make(chan struct{}, 8)
		ex := newPrivateTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v3/userDataStream" {
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			select {
			case listenKeyRequests <- struct{}{}:
			default:
			}
			http.Error(w, `{"code":700007,"msg":"No permission to access the endpoint"}`, http.StatusForbidden)
		})
		var contracts []*FuturesContractResponse
		require.NoError(t, json.Unmarshal([]byte(futuresContractsReply), &contracts), "the contracts must decode")
		ex.storeFuturesContracts(contracts)
		// Credentials are checked, with REST authentication off, so the futures login rests on the websocket's own
		// authenticated support
		ex.SkipAuthCheck = false
		ex.API.AuthenticatedSupport = false
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		venue := &futuresVenue{beforeLogin: futuresAssetFrame}
		require.ErrorIs(t, connectFuturesVenue(t, ex, venue, subscription.List{
			{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
			{Enabled: true, Asset: asset.Futures, Channel: subscription.MyAccountChannel, Authenticated: true},
			{Enabled: true, Asset: asset.Futures, Channel: FuturesPositionsChannel, Authenticated: true},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
		}), errNoListenKey, "the first connect must report the private spot channel it couldn't subscribe")
		_, ok := relayed(t, ex).(accounts.SubAccounts)
		require.True(t, ok, "the first connect's futures login must deliver an asset push")

		// A flush reconnects under the manager's lock, where the connection monitor can't redial first as it can between a
		// Shutdown and a Connect
		require.NoError(t, ex.Websocket.FlushChannels(context.WithoutCancel(t.Context())), "FlushChannels must not error")
		require.NoError(t, ex.Websocket.FlushChannels(context.WithoutCancel(t.Context())), "FlushChannels must not error")
		var logins int
		for _, m := range venue.methods() {
			if m == futuresMethodLogin {
				logins++
			}
		}
		assert.Equal(t, 3, logins, "the futures connection should log in at every connect")
		var private []string
		for _, s := range ex.Websocket.GetSubscriptions() {
			if s.Authenticated {
				private = append(private, s.Asset.String()+" "+s.QualifiedChannel)
			}
		}
		assert.ElementsMatch(t, []string{"futures personal.asset", "futures personal.position"}, private,
			"the private futures subscriptions should be kept, and the private spot one left out")
		assert.Len(t, listenKeyRequests, 1, "a refused listen key should not be asked for again")
	})
}

func TestGenerateFuturesSubscriptionsUnserved(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		sub *subscription.Subscription
		err error
	}{
		{sub: &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.TenMin}, err: kline.ErrUnsupportedInterval},
		{sub: &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.MyWalletChannel}, err: subscription.ErrNotSupported},
	} {
		t.Run(tc.sub.Channel, func(t *testing.T) {
			t.Parallel()
			// Both cases hand tc.sub itself to the generators, which could change it, so they run in order
			for _, pairs := range []currency.Pairs{{futuresTradablePair}, {futuresTradablePair, futuresInversePair}} {
				t.Run(pairs.Join(), func(t *testing.T) {
					ex := newFuturesWebsocketTestExchange(t)
					require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Futures, pairs, true), "StorePairs must not error")
					ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
					ex.Features.Subscriptions = subscription.List{
						{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Interval: kline.HundredMilliseconds},
						{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
						tc.sub,
					}
					public, err := ex.generateFuturesPublicSubscriptions()
					require.ErrorIsf(t, err, websocket.ErrSubscriptionPartial, "a %s subscription the venue does not serve must leave the others for %s", tc.sub.Channel, pairs.Join())
					assert.ErrorIsf(t, err, tc.err, "the expansion should say why the venue does not serve %s", tc.sub.Channel)
					exp := make([]string, len(pairs))
					for i, p := range pairs {
						exp[i] = "ticker@" + p.String()
					}
					got := make([]string, len(public))
					for i, s := range public {
						got[i] = s.QualifiedChannel
					}
					assert.ElementsMatchf(t, exp, got, "the futures ticker should still be subscribed for %s", pairs.Join())
					_, err = ex.generateFuturesPrivateSubscriptions()
					assert.NoError(t, err, "the private futures connection should not report a public subscription")
					spot, err := ex.generatePublicSubscriptions()
					assert.NoError(t, err, "the spot connection should not report a futures subscription")
					assert.Len(t, spot, 1, "the spot ticker should still be subscribed")
					_, err = ex.generatePrivateSubscriptions()
					assert.NoError(t, err, "the private spot connection should not report a futures subscription")
				})
			}
		})
	}
}

func TestGeneratePublicSubscriptions(t *testing.T) {
	t.Parallel()
	t.Run("subscription for every asset", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		ex.Features.Subscriptions = subscription.List{
			{Enabled: true, Asset: asset.All, Channel: channelMiniTickerV3},
			{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.TenMin},
		}
		spot, err := ex.generatePublicSubscriptions()
		require.NoError(t, err, "the spot connection must not report a futures subscription")
		require.Len(t, spot, 1, "a subscription for every asset must still be expanded for spot")
		assert.Equal(t, "spot@"+channelMiniTickerV3+"@BTCUSDT@"+miniTickerTimezone, spot[0].QualifiedChannel, "the spot mini ticker should be subscribed")
	})
}

func TestGenerateSubscriptionsFailureReportedOnce(t *testing.T) {
	t.Parallel()
	// A subscription for every asset to a channel without a symbol yields one record where spot and futures need one
	// each, so it can't be expanded with both enabled; the venue serves no 10 minute futures candles
	for _, tc := range []struct {
		name     string
		sub      *subscription.Subscription
		auth     bool
		reporter string
	}{
		{name: "every asset", sub: &subscription.Subscription{Enabled: true, Asset: asset.All, Channel: channelMiniTickersV3}, auth: true, reporter: "spot public"},
		{name: "every asset without authentication", sub: &subscription.Subscription{Enabled: true, Asset: asset.All, Channel: channelMiniTickersV3}, reporter: "spot public"},
		{name: "every asset authenticated", sub: &subscription.Subscription{Enabled: true, Asset: asset.All, Channel: channelAccountV3, Authenticated: true}, auth: true, reporter: "spot private"},
		{name: "every asset authenticated without authentication", sub: &subscription.Subscription{Enabled: true, Asset: asset.All, Channel: channelAccountV3, Authenticated: true}},
		{name: "every asset private without the authenticated flag", sub: &subscription.Subscription{Enabled: true, Asset: asset.All, Channel: channelAccountV3}, reporter: "spot public"},
		{name: "futures", sub: &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.TenMin}, auth: true, reporter: "futures public"},
		{name: "futures authenticated", sub: &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.TenMin, Authenticated: true}, auth: true, reporter: "futures private"},
		{name: "futures authenticated without authentication", sub: &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.TenMin, Authenticated: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, pairs := range []currency.Pairs{nil, {futuresTradablePair}} {
				t.Run(cmp.Or(pairs.Join(), "no explicit pairs"), func(t *testing.T) {
					t.Parallel()
					ex := newFuturesWebsocketTestExchange(t)
					ex.Websocket.SetCanUseAuthenticatedEndpoints(tc.auth)
					sub := tc.sub.Clone()
					sub.Pairs = pairs
					ex.Features.Subscriptions = subscription.List{
						{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Interval: kline.HundredMilliseconds},
						sub,
					}
					// The generators share ex in order, so each also checks that a generation keeps nothing from the last
					for _, g := range connectionGenerators(ex) {
						t.Run(g.name, func(t *testing.T) {
							subs, err := g.generate()
							if g.name == "spot public" {
								assert.Lenf(t, subs, 1, "the spot ticker should still be subscribed with %d explicit pairs", len(pairs))
							}
							if g.name != tc.reporter {
								assert.NoErrorf(t, err, "the %s connection should not report a failure %q reports, with %d explicit pairs", g.name, tc.reporter, len(pairs))
								return
							}
							assert.ErrorIsf(t, err, websocket.ErrSubscriptionPartial, "the %s connection should report the subscription which can't be expanded, with %d explicit pairs", g.name, len(pairs))
							assert.ErrorContainsf(t, err, sub.Channel, "the %s connection should name the subscription which can't be expanded, with %d explicit pairs", g.name, len(pairs))
						})
					}
				})
			}
		})
	}
}

func TestGenerateSubscriptionsPairListFailureReportedOnce(t *testing.T) {
	t.Parallel()
	// Without a request format the futures pairs can't be listed, which fails a futures subscription before its template
	// is executed, a private channel's included
	for _, tc := range []struct {
		name          string
		authenticated bool
		auth          bool
		reporter      string
	}{
		{name: "unflagged", reporter: "futures public"},
		{name: "unflagged with authentication", auth: true, reporter: "futures public"},
		{name: "authenticated", authenticated: true, auth: true, reporter: "futures private"},
		{name: "authenticated without authentication", authenticated: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newFuturesWebsocketTestExchange(t)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(tc.auth)
			futuresStore := *ex.CurrencyPairs.Pairs[asset.Futures]
			futuresStore.RequestFormat = nil
			ex.CurrencyPairs.Pairs[asset.Futures] = &futuresStore
			ex.Features.Subscriptions = subscription.List{
				{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Interval: kline.HundredMilliseconds},
				{Enabled: true, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: tc.authenticated},
			}
			// The generators share ex in order, so each also checks that a generation keeps nothing from the last
			for _, g := range connectionGenerators(ex) {
				t.Run(g.name, func(t *testing.T) {
					_, err := g.generate()
					if g.name != tc.reporter {
						assert.NoErrorf(t, err, "the %s connection should not report a failure %q reports", g.name, tc.reporter)
						return
					}
					assert.ErrorIsf(t, err, websocket.ErrSubscriptionPartial, "the %s connection should report the subscription which can't be expanded", g.name)
					assert.ErrorIsf(t, err, currency.ErrPairFormatIsNil, "the %s connection should say why the subscription can't be expanded", g.name)
				})
			}
		})
	}
}

// connectionGenerator is one of an exchange's connections' subscription generators, by the connection's name
type connectionGenerator struct {
	name     string
	generate func() (subscription.List, error)
}

// connectionGenerators returns the subscription generators of an exchange's four connections
func connectionGenerators(ex *Exchange) []connectionGenerator {
	return []connectionGenerator{
		{name: "spot public", generate: ex.generatePublicSubscriptions},
		{name: "spot private", generate: ex.generatePrivateSubscriptions},
		{name: "futures public", generate: ex.generateFuturesPublicSubscriptions},
		{name: "futures private", generate: ex.generateFuturesPrivateSubscriptions},
	}
}

func TestWebsocketFuturesConnectUnservedSubscription(t *testing.T) {
	t.Parallel()
	for _, pairs := range []currency.Pairs{{futuresTradablePair}, {futuresTradablePair, futuresInversePair}} {
		t.Run(pairs.Join(), func(t *testing.T) {
			t.Parallel()
			ex := newFuturesWebsocketTestExchange(t)
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Futures, pairs, true), "StorePairs must not error")
			err := connectFuturesVenue(t, ex, new(futuresVenue), subscription.List{
				{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
				{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.TenMin},
			})
			require.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "Connect must report the subscription the venue does not serve")
			assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "Connect should say why the venue does not serve it")
			assert.True(t, ex.Websocket.IsConnected(), "the websocket should stay connected for the subscriptions the venue serves")
			assert.Len(t, ex.Websocket.GetSubscriptions(), len(pairs), "the futures ticker should be subscribed for each contract")
		})
	}
}

func TestMigratedSubscriptionsCoverFutures(t *testing.T) {
	t.Parallel()
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	data, err := os.ReadFile(filepath.Join(root, "testdata", "configtest.json"))
	require.NoError(t, err, "ReadFile must load the test config")
	var saved map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &saved), "Unmarshal must decode the test config")
	var exchanges []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(saved["exchanges"], &exchanges), "Unmarshal must decode the saved exchanges")
	i := slices.IndexFunc(exchanges, func(e map[string]json.RawMessage) bool { return string(e["name"]) == `"MEXC"` })
	require.NotEqual(t, -1, i, "the test config must hold MEXC")
	var features map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(exchanges[i]["features"], &features), "Unmarshal must decode MEXC's features")
	// A version 18 config holds the spot defaults, here beside a disabled subscription which names no default
	spotDefaults := slices.DeleteFunc(defaultSubscriptions.Clone(), func(s *subscription.Subscription) bool { return s.Asset != asset.Spot })
	previous, err := json.Marshal(spotDefaults)
	require.NoError(t, err, "Marshal must encode the spot defaults")
	features["subscriptions"] = json.RawMessage(strings.TrimSuffix(string(previous), "]") +
		`,{"enabled":false,"channel":"public.bookTicker.batch.v3.api.pb","asset":"spot"}]`)
	exchanges[i]["features"], err = json.Marshal(features)
	require.NoError(t, err, "Marshal must encode MEXC's features")
	saved["exchanges"], err = json.Marshal(exchanges)
	require.NoError(t, err, "Marshal must encode the saved exchanges")
	saved["version"] = json.RawMessage(`18`)
	data, err = json.Marshal(saved)
	require.NoError(t, err, "Marshal must encode the version 18 config")
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, data, 0o600), "WriteFile must save the version 18 config")

	var cfg config.Config
	require.NoError(t, cfg.ReadConfigFromFile(path, true), "ReadConfigFromFile must upgrade the version 18 config")
	exchCfg, err := cfg.GetExchangeConfig("MEXC")
	require.NoError(t, err, "GetExchangeConfig must find MEXC")
	ex := new(Exchange)
	ex.SetDefaults()
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(t, ex.Setup(exchCfg), "Setup must accept the upgraded config")
	assert.Len(t, ex.Config.Features.Subscriptions, len(defaultSubscriptions)+1, "the upgraded config should hold every default and the disabled subscription")

	public, err := ex.generateFuturesPublicSubscriptions()
	require.NoError(t, err, "generateFuturesPublicSubscriptions must not error")
	got := make([]string, 0, len(public))
	for _, s := range public {
		got = append(got, s.QualifiedChannel)
	}
	assert.ElementsMatch(t, []string{
		"ticker@BTC_USDT", "ticker@ETH_USDT",
		"depth.full@BTC_USDT@20", "depth.full@ETH_USDT@20",
		"deal@BTC_USDT", "deal@ETH_USDT",
		"kline@BTC_USDT@Min15", "kline@ETH_USDT@Min15",
	}, got, "the upgraded config should subscribe to every public futures channel for each enabled contract")
	spot, err := ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	for _, s := range spot {
		assert.NotContainsf(t, s.QualifiedChannel, "bookTicker.batch", "the disabled subscription should not subscribe, but %s does", s.QualifiedChannel)
	}
}

// futuresVenue is a mock futures websocket venue. It acknowledges subscriptions and logins, refuses the contract or API
// key it is told to, and a marker request's symbol, as no contract has it, and answers an unsubscription only to refuse
// one. It pushes a frame after a subscription to a channel it holds one for, and beforeLogin ahead of its reply to an
// accepted login, and records what it is sent
type futuresVenue struct {
	refuse      string
	pushes      map[string]string
	beforeLogin string
	// atLogin is called when a login is accepted, before anything is written
	atLogin func()
	mu      sync.Mutex
	conn    *gws.Conn
	sent    []string
}

// delist makes the venue refuse a contract from now on, as it does one it no longer lists
func (v *futuresVenue) delist(symbol string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.refuse = symbol
}

// write sends a frame, one at a time
func (v *futuresVenue) write(conn *gws.Conn, frame string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.conn = conn
	return conn.WriteMessage(gws.TextMessage, []byte(frame))
}

// push sends a frame to the connection the venue last wrote to
func (v *futuresVenue) push(t *testing.T, frame string) {
	t.Helper()
	v.mu.Lock()
	defer v.mu.Unlock()
	require.NotNil(t, v.conn, "the venue must have a connection to push to")
	require.NoError(t, v.conn.WriteMessage(gws.TextMessage, []byte(frame)), "the venue must push the frame")
}

// methods returns the methods the venue has been sent
func (v *futuresVenue) methods() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.sent)
}

// handle answers a request sent to the venue
func (v *futuresVenue) handle(_ testing.TB, msg []byte, conn *gws.Conn) error {
	var req struct {
		Method string          `json:"method"`
		Param  json.RawMessage `json:"param"`
	}
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	v.mu.Lock()
	v.sent = append(v.sent, req.Method)
	refuse := v.refuse
	v.mu.Unlock()
	switch {
	case req.Method == "ping":
		return v.write(conn, `{"channel":"pong","data":1792842501046,"ts":1792842501046}`)
	case strings.HasPrefix(req.Method, futuresUnsubscribePrefix):
		var params WebsocketFuturesChannelParams
		if err := json.Unmarshal(req.Param, &params); err != nil {
			return err
		}
		if params.Symbol == refuse {
			return v.write(conn, `{"channel":"rs.error","data":"Contract [`+params.Symbol+`] not exists","ts":1792842495741}`)
		}
		return nil
	case req.Method == futuresMethodLogin:
		var login WebsocketFuturesLoginParams
		if err := json.Unmarshal(req.Param, &login); err != nil {
			return err
		}
		mac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(login.APIKey+login.RequestTime), []byte("test-secret"))
		if err != nil {
			return err
		}
		if login.APIKey != "test-key" || login.Signature != hex.EncodeToString(mac) || login.APIKey == refuse {
			return v.write(conn, `{"channel":"rs.error","data":"Apikey doesn't exist!","ts":1792842495741}`)
		}
		if v.atLogin != nil {
			v.atLogin()
		}
		if v.beforeLogin != "" {
			if err := v.write(conn, v.beforeLogin); err != nil {
				return err
			}
		}
		return v.write(conn, `{"channel":"rs.login","data":"success","ts":"1587442022003"}`)
	case strings.HasPrefix(req.Method, futuresSubscribePrefix):
		var params WebsocketFuturesChannelParams
		if err := json.Unmarshal(req.Param, &params); err != nil {
			return err
		}
		if params.Symbol == refuse || strings.HasPrefix(params.Symbol, futuresResyncMarkerPrefix) {
			return v.write(conn, `{"channel":"rs.error","data":"Contract [`+params.Symbol+`] not exists","ts":1792842495741}`)
		}
		if err := v.write(conn, `{"channel":"rs.`+req.Method+`","data":"success","ts":1792842495741}`); err != nil {
			return err
		}
		if push, ok := v.pushes[req.Method]; ok {
			return v.write(conn, push)
		}
		return nil
	}
	return v.write(conn, `{"channel":"rs.error","data":"Not support method","ts":1792842495741}`)
}

// connectFuturesVenue connects an exchange's futures websocket connections to a mock venue, subscribing to subs
func connectFuturesVenue(t *testing.T, ex *Exchange, venue *futuresVenue, subs subscription.List) error {
	t.Helper()
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, venue.handle)
	}))
	// The exchange's connector dials with its own dialer, which reaches a listening server only
	server.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
	ex.Features.Subscriptions = subs
	t.Cleanup(func() {
		if ex.Websocket.IsConnected() {
			assert.NoError(t, ex.Websocket.Shutdown(), "Shutdown should not error")
		}
		// Disabled too, or the connection monitor goes on redialling the closed venue after the test
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
	})
	// The connection monitors outlive a cancelled context by reconnecting with it, so the connection is shut down by
	// the cleanup rather than by the test's context
	return ex.Websocket.Connect(context.WithoutCancel(t.Context()))
}

func TestWebsocketFuturesSubscribe(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	venue := &futuresVenue{pushes: map[string]string{"sub.ticker": futuresTickerFrame, "sub.depth.full": futuresDepthFrame}}
	require.NoError(t, connectFuturesVenue(t, ex, venue, subscription.List{
		{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.Futures, Channel: subscription.OrderbookChannel, Levels: 5},
	}), "Connect must not error")

	subs := ex.Websocket.GetSubscriptions()
	got := make([]string, 0, len(subs))
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "%s should be subscribed", s.QualifiedChannel)
		got = append(got, s.QualifiedChannel)
	}
	assert.ElementsMatch(t, []string{"ticker@BTC_USDT", "ticker@BTC_USD", "depth.full@BTC_USDT@5", "depth.full@BTC_USD@5"}, got, "every acknowledged subscription should be recorded")
	assert.ElementsMatch(t, []string{"sub.ticker", "sub.ticker", "sub.depth.full", "sub.depth.full"}, venue.methods(), "each subscription should be requested once")

	require.Eventually(t, func() bool {
		_, err := orderbook.Get(ex.Name, futuresTradablePair, asset.Futures)
		return err == nil
	}, 5*time.Second, 10*time.Millisecond, "the pushed book must be loaded")
	tick, err := ticker.GetTicker(ex.Name, futuresTradablePair, asset.Futures)
	require.NoError(t, err, "the pushed ticker must be stored")
	assert.Equal(t, futuresTickerFramePrice(ex.Name), tick, "the pushed ticker should be stored")

	conn, err := ex.Websocket.GetConnection(futuresPublicConnection)
	require.NoError(t, err, "GetConnection must not error")
	method, _, err := futuresChannelRequest(subs[0])
	require.NoError(t, err, "futuresChannelRequest must not error")
	require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, subs[:1]), "UnsubscribeChannels must not error")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 3, "the unsubscribed channel should be removed")
	require.Eventually(t, func() bool {
		return slices.Contains(venue.methods(), futuresUnsubscribePrefix+method)
	}, 5*time.Second, 10*time.Millisecond, "the unsubscription must reach the venue")
}

func TestWebsocketFuturesSubscribeRefused(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	venue := &futuresVenue{refuse: "BTC_USD"}
	err := connectFuturesVenue(t, ex, venue, subscription.List{{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel}})
	require.ErrorIs(t, err, errFuturesWebsocketRejected, "a refused subscription must be reported")
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "a refused subscription must fail the subscription")

	subs := ex.Websocket.GetSubscriptions()
	require.Len(t, subs, 1, "only the accepted subscription must be recorded")
	assert.Equal(t, "ticker@BTC_USDT", subs[0].QualifiedChannel, "the accepted subscription should be recorded")
}

func TestWebsocketFuturesUnsubscribeRefused(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	venue := new(futuresVenue)
	require.NoError(t, connectFuturesVenue(t, ex, venue, subscription.List{{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel}}), "Connect must not error")
	conn, err := ex.Websocket.GetConnection(futuresPublicConnection)
	require.NoError(t, err, "GetConnection must not error")
	var delisted subscription.List
	for _, s := range ex.Websocket.GetSubscriptions() {
		if s.QualifiedChannel == "ticker@BTC_USD" {
			delisted = append(delisted, s)
		}
	}
	require.Len(t, delisted, 1, "the subscription to the contract to delist must be recorded")

	// The venue refuses to unsubscribe from a contract it no longer lists, and the refusal comes before the reply to a
	// subscription sent after it on the connection
	venue.delist("BTC_USD")
	require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, delisted), "UnsubscribeChannels must not error when the venue holds no such subscription")
	next := &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{futuresTradablePair}, QualifiedChannel: "deal@BTC_USDT"}
	require.NoError(t, ex.Websocket.SubscribeToChannels(t.Context(), conn, subscription.List{next}), "a subscription after a refused unsubscription must be answered by its own reply")
	got := make([]string, 0, 2)
	for _, s := range ex.Websocket.GetSubscriptions() {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "%s should be subscribed", s.QualifiedChannel)
		got = append(got, s.QualifiedChannel)
	}
	assert.ElementsMatch(t, []string{"ticker@BTC_USDT", "deal@BTC_USDT"}, got, "the refused unsubscription should be removed and the next subscription recorded")
	assert.Equal(t, []string{"sub.ticker", "sub.ticker", "unsub.ticker", "sub.ticker", "sub.deal"}, venue.methods(),
		"the unsubscription should be followed by a marker request before the next subscription")
	err = relayedError(t, ex)
	assert.ErrorIs(t, err, errFuturesWebsocketRejected, "the refused unsubscription should be reported")
	assert.ErrorContains(t, err, "Contract [BTC_USD]", "the refusal reported should be the unsubscription's")
}

// relayedError returns the next error relayed by an exchange's websocket
func relayedError(t *testing.T, ex *Exchange) error {
	t.Helper()
	err, ok := relayed(t, ex).(error)
	require.True(t, ok, "an error must be relayed")
	return err
}

func TestWebsocketFuturesUnsubscribe(t *testing.T) {
	t.Parallel()
	tickerSubscription := func(symbol string) *subscription.Subscription {
		pair := currency.NewPairWithDelimiter(strings.TrimSuffix(symbol, "_USDT"), "USDT", currency.UnderscoreDelimiter)
		return &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel, Pairs: currency.Pairs{pair}, QualifiedChannel: "ticker@" + symbol}
	}
	// serve answers a marker request with the frames given, then the marker's refusal unless told to leave it
	// unanswered, answers nothing else, and records each request's method and symbol
	serve := func(beforeMarker []string, refuseMarker bool) (mockws.WsMockFunc, func() []string) {
		var mu sync.Mutex
		var requests []string
		return func(_ testing.TB, msg []byte, conn *gws.Conn) error {
				var req struct {
					Method string                        `json:"method"`
					Param  WebsocketFuturesChannelParams `json:"param"`
				}
				if err := json.Unmarshal(msg, &req); err != nil {
					return err
				}
				mu.Lock()
				requests = append(requests, req.Method+" "+req.Param.Symbol)
				mu.Unlock()
				if !strings.HasPrefix(req.Param.Symbol, futuresResyncMarkerPrefix) {
					return nil
				}
				for _, frame := range beforeMarker {
					if err := conn.WriteMessage(gws.TextMessage, []byte(frame)); err != nil {
						return err
					}
				}
				if !refuseMarker {
					return nil
				}
				return conn.WriteMessage(gws.TextMessage, []byte(`{"channel":"rs.error","data":"Contract [`+req.Param.Symbol+`] not exists","ts":1792842495741}`))
			}, func() []string {
				mu.Lock()
				defer mu.Unlock()
				return slices.Clone(requests)
			}
	}
	const refusal = `{"channel":"rs.error","data":"Contract [BTC_USDT] not exists","ts":1792842495741}`

	t.Run("nothing to send", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		handle, requests := serve(nil, true)
		conn := dialFuturesTestVenue(t, ex, handle)
		unpaired := tickerSubscription("BTC_USDT")
		unpaired.Pairs = nil
		err := ex.WebsocketFuturesUnsubscribe(t.Context(), conn, subscription.List{unpaired})
		assert.ErrorIs(t, err, subscription.ErrNotSinglePair, "an unsubscription the venue can't take should be returned")
		assert.Empty(t, requests(), "no marker should follow when no unsubscription was sent")
	})

	t.Run("holds its own connection's lock alone", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		waitingHandle, waitingRequests := serve(nil, false)
		waiting := dialFuturesTestVenue(t, ex, waitingHandle)
		answeredHandle, answeredRequests := serve(nil, true)
		answered := dialFuturesTestVenue(t, ex, answeredHandle)
		waitingSub, answeredSub := tickerSubscription("BTC_USDT"), tickerSubscription("ETH_USDT")
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(waiting, waitingSub), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(answered, answeredSub), "AddSuccessfulSubscriptions must not error")
		// The waiting connection's marker goes unanswered, so its unsubscription holds that connection's lock for twice
		// the response limit, or until the test cancels it
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- ex.WebsocketFuturesUnsubscribe(ctx, waiting, subscription.List{waitingSub}) }()
		require.Eventually(t, func() bool { return len(waitingRequests()) == 2 }, 10*time.Second, time.Millisecond, "the waiting connection's marker must be sent")
		require.NoError(t, ex.WebsocketFuturesUnsubscribe(t.Context(), answered, subscription.List{answeredSub}), "WebsocketFuturesUnsubscribe must not error")
		select {
		case <-done:
			assert.Fail(t, "an unsubscription on another connection should not wait for one awaiting its marker's reply")
		default:
			cancel()
			select {
			case err := <-done:
				require.NoError(t, err, "an unsubscription whose marker goes unanswered must still succeed")
			case <-time.After(time.Minute):
				require.FailNow(t, "the unsubscription must return once its context is cancelled")
			}
		}
		requests := answeredRequests()
		require.Len(t, requests, 2, "the other connection must be sent its unsubscription and a marker")
		assert.Equal(t, "unsub.ticker ETH_USDT", requests[0], "the other connection should be sent its unsubscription")
		assert.True(t, strings.HasPrefix(requests[1], "sub.ticker "+futuresResyncMarkerPrefix), "the other connection's unsubscription should be followed by its own marker")
		assert.Equal(t, map[websocket.Connection]int{waiting: 2}, futuresRepliesOwed(ex), "only the connection whose marker went unanswered should owe replies")
	})

	// A reply owed by an earlier request comes before the unsubscription's refusal and the marker's, and all are drained
	t.Run("reply owed", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		handle, requests := serve([]string{`{"channel":"rs.sub.ticker","data":"success","ts":1792842495741}`, refusal}, true)
		conn := dialFuturesTestVenue(t, ex, handle)
		sub := tickerSubscription("BTC_USDT")
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")
		ex.futuresConnectionState(conn).repliesOwed = 1
		require.NoError(t, ex.WebsocketFuturesUnsubscribe(t.Context(), conn, subscription.List{sub}), "WebsocketFuturesUnsubscribe must not error once the marker is answered")
		assert.Empty(t, futuresRepliesOwed(ex), "no reply should be owed once the marker is answered")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "the unsubscribed channel should be removed")
		assert.Equal(t, []string{"unsub.ticker BTC_USDT", "sub.ticker gct-resync-1"}, requests(), "the unsubscription should be followed by one marker")
		err := relayedError(t, ex)
		assert.ErrorIs(t, err, errFuturesWebsocketRejected, "the unsubscription's refusal should be reported")
		assert.ErrorContains(t, err, "Contract [BTC_USDT]", "the refusal reported should be the unsubscription's")
		requireNothingRelayed(t, ex)
	})

	// The marker's wait can end at its context's deadline, after the replies it waits for or with none, and each refusal
	// it read is reported all the same
	for name, beforeMarker := range map[string][]string{
		"marker unanswered":                   nil,
		"marker unanswered after a refusal":   {refusal},
		"marker unanswered after its replies": {refusal, `{"channel":"rs.sub.ticker","data":"success","ts":1792842495741}`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				defer synctest.Sleep(time.Minute)
				ex := newFuturesWebsocketTestExchange(t)
				handle, requests := serve(beforeMarker, false)
				conn := dialFuturesTestVenue(t, ex, handle)
				sub := tickerSubscription("BTC_USDT")
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")
				// Long enough for the connection's rate limiter to let the marker follow the unsubscription
				ctx, cancel := context.WithTimeout(t.Context(), time.Second)
				defer cancel()
				done := make(chan error, 1)
				// Taken before the unsubscription starts, as a lookup waiting on a lock it holds would hang the bubble
				state := ex.futuresConnectionState(conn)
				go func() { done <- ex.WebsocketFuturesUnsubscribe(ctx, conn, subscription.List{sub}) }()
				synctest.Wait()
				held := !state.mu.TryLock()
				if !held {
					state.mu.Unlock()
				}
				require.NoError(t, <-done, "an unsubscription whose marker goes unanswered must still succeed, leaving the replies owed")
				assert.True(t, held, "the request lock should be held until the marker's wait ends, so that no other request is sent meanwhile")
				assert.Equal(t, 2, futuresRepliesOwed(ex)[conn], "the connection should owe the unsubscription's reply and the marker's")
				assert.Empty(t, ex.Websocket.GetSubscriptions(), "the unsubscription was sent, so its channel should be removed")
				assert.Equal(t, []string{"unsub.ticker BTC_USDT", "sub.ticker gct-resync-1"}, requests(), "the unsubscription should be followed by one marker")
				if beforeMarker != nil {
					err := relayedError(t, ex)
					assert.ErrorIs(t, err, errFuturesWebsocketRejected, "a refusal the marker's wait collected should be reported")
					assert.ErrorContains(t, err, "Contract [BTC_USDT]", "the refusal reported should be the unsubscription's")
				}
				assert.ErrorIs(t, relayedError(t, ex), errFuturesWebsocketOutOfStep, "the unanswered marker should be reported")
				requireNothingRelayed(t, ex)
			})
		})
	}

	// The replies to unsubscriptions and their marker can come after the marker's wait ends; the next request then
	// drains them all before it is sent, and gets its own reply
	t.Run("replies after the marker's wait", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			defer synctest.Sleep(time.Minute)
			ex := newFuturesWebsocketTestExchange(t)
			var mu sync.Mutex
			var requests []string
			conn := dialFuturesTestVenue(t, ex, func(_ testing.TB, msg []byte, c *gws.Conn) error {
				var req struct {
					Method string                        `json:"method"`
					Param  WebsocketFuturesChannelParams `json:"param"`
				}
				if err := json.Unmarshal(msg, &req); err != nil {
					return err
				}
				mu.Lock()
				requests = append(requests, req.Method+" "+req.Param.Symbol)
				mu.Unlock()
				// Every request is answered 400 ms after it is read: a listed contract's subscription is acknowledged, and
				// any other request refused
				time.Sleep(400 * time.Millisecond)
				reply := `{"channel":"rs.error","data":"Contract [` + req.Param.Symbol + `] not exists","ts":1792842495741}`
				if req.Param.Symbol == "BTC_USDT" {
					reply = `{"channel":"rs.` + req.Method + `","data":"success","ts":1792842495741}`
				}
				return c.WriteMessage(gws.TextMessage, []byte(reply))
			})
			delisted := subscription.List{tickerSubscription("NOPE_USDT"), tickerSubscription("GONE_USDT")}
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, delisted...), "AddSuccessfulSubscriptions must not error")
			ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
			defer cancel()
			require.NoError(t, ex.WebsocketFuturesUnsubscribe(ctx, conn, delisted), "an unsubscription whose marker's wait ends must still succeed")
			assert.Equal(t, 3, futuresRepliesOwed(ex)[conn], "the connection should owe both refusals and the marker's")
			assert.ErrorIs(t, relayedError(t, ex), errFuturesWebsocketOutOfStep, "the marker's ended wait should be reported")
			err := ex.futuresWebsocketRequest(t.Context(), conn, &WebsocketFuturesRequest{Method: "sub.deal", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}})
			require.NoError(t, err, "the next request must be answered by its own reply once the replies owed are drained")
			assert.Empty(t, futuresRepliesOwed(ex), "no reply should be owed once a marker is answered")
			// The refusals drained are reported, and the earlier marker's refusal, drained with them, is not
			for _, symbol := range []string{"NOPE_USDT", "GONE_USDT"} {
				err := relayedError(t, ex)
				assert.ErrorIsf(t, err, errFuturesWebsocketRejected, "the refused unsubscription from %s should be reported", symbol)
				assert.ErrorContainsf(t, err, "Contract ["+symbol+"]", "the refusal reported should be %s's", symbol)
			}
			requireNothingRelayed(t, ex)
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []string{"unsub.ticker NOPE_USDT", "unsub.ticker GONE_USDT", "sub.ticker gct-resync-1", "sub.ticker gct-resync-2", "sub.deal BTC_USDT"}, requests,
				"the next request should be sent after one marker drains the replies owed")
		})
	})
}

func TestReportFuturesWebsocketError(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	ex.Websocket.DataHandler = stream.NewRelay(1)
	relayedFirst := errors.New("relayed")
	ex.reportFuturesWebsocketError(t.Context(), relayedFirst)
	// The relay is full, so the next error is logged rather than relayed, without waiting for room
	ex.reportFuturesWebsocketError(t.Context(), errors.New("logged"))
	assert.Equal(t, relayedFirst, relayed(t, ex), "the error reported while the relay had room should be relayed")
	requireNothingRelayed(t, ex)
}

func TestWebsocketFuturesSubscribePrivate(t *testing.T) {
	t.Parallel()

	t.Run("logs in once and relays the pushes of the channels subscribed", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		venue := new(futuresVenue)
		require.NoError(t, connectFuturesVenue(t, ex, venue, subscription.List{
			{Enabled: true, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: true},
			{Enabled: true, Asset: asset.Futures, Channel: subscription.MyAccountChannel, Authenticated: true},
		}), "Connect must not error")
		assert.Equal(t, []string{futuresMethodLogin}, venue.methods(), "the connection should log in once and subscribe to nothing, as the venue pushes every private channel")
		assert.Len(t, ex.Websocket.GetSubscriptions(), 2, "the private subscriptions should be recorded")

		venue.push(t, futuresOrderFrame)
		detail, ok := relayed(t, ex).(*order.Detail)
		require.True(t, ok, "the pushed order must be relayed")
		assert.Equal(t, futuresOrderFrameDetail(ex.Name), detail, "the pushed order should be relayed")
	})

	t.Run("relays a push that comes with the login's reply", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		// The venue pushes the private channels once a login is accepted, so a push can come no later than its reply
		venue := &futuresVenue{beforeLogin: futuresOrderFrame}
		atLogin := make(chan []subscription.State, 1)
		venue.atLogin = func() {
			subs := ex.Websocket.GetSubscriptions()
			states := make([]subscription.State, 0, len(subs))
			for _, s := range subs {
				states = append(states, s.State())
			}
			atLogin <- states
		}
		require.NoError(t, connectFuturesVenue(t, ex, venue, subscription.List{
			{Enabled: true, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: true},
		}), "Connect must not error")
		select {
		case states := <-atLogin:
			assert.Equal(t, []subscription.State{subscription.SubscribingState}, states, "the private subscription should be recorded, as subscribing, before the login is sent")
		default:
			require.Fail(t, "the login must reach the venue")
		}
		detail, ok := relayed(t, ex).(*order.Detail)
		require.True(t, ok, "the order pushed with the login's reply must be relayed")
		assert.Equal(t, futuresOrderFrameDetail(ex.Name), detail, "the order pushed with the login's reply should be relayed")
		subs := ex.Websocket.GetSubscriptions()
		require.Len(t, subs, 1, "the private subscription must be recorded")
		assert.Equal(t, subscription.SubscribedState, subs[0].State(), "the private subscription should be subscribed once the login is accepted")
	})

	t.Run("refuses a subscription already held", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		venue := new(futuresVenue)
		conn := dialFuturesTestVenue(t, ex, venue.handle)
		held := &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: true, QualifiedChannel: futuresChannelOrder}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, held), "AddSuccessfulSubscriptions must not error")
		err := ex.websocketFuturesSubscribePrivate(t.Context(), conn, subscription.List{held})
		require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "a subscription already held must fail to subscribe")
		assert.ErrorIs(t, err, subscription.ErrDuplicate, "the failure should say the subscription is held already")
		assert.Empty(t, venue.methods(), "nothing should be sent when no subscription could be recorded")
		assert.Equal(t, subscription.SubscribedState, held.State(), "the subscription already held should stay subscribed")
	})

	t.Run("keeps the public subscriptions when the login is refused", func(t *testing.T) {
		t.Parallel()
		ex := newFuturesWebsocketTestExchange(t)
		ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		err := connectFuturesVenue(t, ex, &futuresVenue{refuse: "test-key"}, subscription.List{
			{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
			{Enabled: true, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: true},
		})
		require.ErrorIs(t, err, errFuturesWebsocketRejected, "a refused login must be reported")
		subs := ex.Websocket.GetSubscriptions()
		require.Len(t, subs, 2, "the public subscriptions must survive a refused login")
		for _, s := range subs {
			assert.Equalf(t, subscription.TickerChannel, s.Channel, "%s should not be recorded after a refused login", s.QualifiedChannel)
		}
	})
}
