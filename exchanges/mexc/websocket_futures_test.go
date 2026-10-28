package mexc

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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

func TestWebsocketFuturesHandleTrades(t *testing.T) {
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

func TestWebsocketFuturesHandleCandle(t *testing.T) {
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

func TestWebsocketFuturesHandleFill(t *testing.T) {
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

func TestWebsocketFuturesHandlePosition(t *testing.T) {
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
}

func TestWebsocketFuturesHandleData(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	for name, frame := range map[string]string{
		"pong":                  `{"channel":"pong","data":1792842501046,"ts":1792842501046}`,
		"unsubscribable push":   `{"channel":"push.personal.leverage.mode","data":{"leverageMode":1},"ts":1760942212000}`,
		"late subscription ack": `{"channel":"rs.sub.ticker","data":"success","ts":1792842495741}`,
	} {
		require.NoErrorf(t, ex.WebsocketFuturesHandleData(t.Context(), &futuresReplyConnection{}, []byte(frame)), "WebsocketFuturesHandleData must not error for a %s", name)
	}
	requireNothingRelayed(t, ex)

	for _, frame := range []string{`{"channel":"push.unknown","data":{}}`, `{"channel":"surprise"}`, `null`} {
		require.NoErrorf(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(frame)), "WebsocketFuturesHandleData must not error for %s", frame)
		warning, ok := relayed(t, ex).(websocket.UnhandledMessageWarning)
		require.Truef(t, ok, "%s must be reported as unhandled", frame)
		assert.Containsf(t, warning.Message, frame, "the warning should carry %s", frame)
	}

	err := ex.WebsocketFuturesHandleData(t.Context(), &futuresReplyConnection{}, []byte(`{"channel":"rs.error","data":"Contract [FOO_BAR] not exists"}`))
	assert.ErrorIs(t, err, errFuturesWebsocketRejected, "a refusal no request waits for should be reported")
	assert.Error(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(`{"channel":`)), "a malformed frame should be refused")
}

// futuresReplyConnection answers each futures websocket request with a recorded reply, or fails to send it with err,
// and routes no reply to a waiting request
type futuresReplyConnection struct {
	websocket.Connection
	mu    sync.Mutex
	reply []byte
	err   error
	sent  []*WebsocketFuturesRequest
	// late, when set, holds the late reply which is awaited until the test sends it
	late    chan websocket.MatchedResponse
	awaited bool
}

// SendMessageReturnResponse records a request and returns the recorded reply, or fails to send it
func (c *futuresReplyConnection) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, signature, payload any) ([]byte, error) {
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
	if c.late != nil {
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
	for reply, exp := range map[string]error{
		`{"channel":"rs.sub.ticker","data":"success","ts":1792842495741}`:                  nil,
		`{"channel":"rs.error","data":"Contract [FOO_BAR] not exists","ts":1792842495741}`: errFuturesWebsocketRejected,
		`{"channel":"rs.sub.ticker","data":"Not support interval","ts":1792842495741}`:     errFuturesWebsocketRejected,
		`{"channel":"rs.sub.deal","data":"success","ts":1792842495741}`:                    errFuturesWebsocketRejected,
		`{"channel":"rs.sub.ticker","data":{"code":1,"msg":"refused"},"ts":1792842495741}`: errFuturesWebsocketRejected,
		`null`: common.ErrNoResponse,
	} {
		err := ex.futuresWebsocketRequest(t.Context(), &futuresReplyConnection{reply: []byte(reply)}, req)
		assert.ErrorIsf(t, err, exp, "the reply %s should be read", reply)
	}
	assert.Error(t, ex.futuresWebsocketRequest(t.Context(), &futuresReplyConnection{reply: []byte(`{"channel":`)}, req), "a malformed reply should be refused")
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

func TestFuturesWebsocketRequestHoldsLock(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	req := &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}}
	ctx := &lockCheckingContext{Context: t.Context(), lock: &ex.futuresWebsocketRequestMu}
	require.NoError(t, ex.futuresWebsocketRequest(ctx, &futuresReplyConnection{reply: []byte(`{"channel":"rs.sub.ticker","data":"success"}`)}, req),
		"futuresWebsocketRequest must not error")
	require.NotEmpty(t, ctx.held, "the request's context must be checked before it is sent")
	assert.NotContains(t, ctx.held, false, "the request's context should be checked under the request lock, so that one ending while it waits for the lock is not sent")

	synctest.Test(t, func(t *testing.T) {
		ex := newFuturesWebsocketTestExchange(t)
		conn := &futuresReplyConnection{err: websocket.ErrSignatureTimeout, late: make(chan websocket.MatchedResponse, 1)}
		done := make(chan error, 1)
		go func() { done <- ex.futuresWebsocketRequest(t.Context(), conn, req) }()
		synctest.Wait()
		held := !ex.futuresWebsocketRequestMu.TryLock()
		if !held {
			ex.futuresWebsocketRequestMu.Unlock()
		}
		conn.late <- websocket.MatchedResponse{Err: websocket.ErrSignatureTimeout}
		require.ErrorIs(t, <-done, websocket.ErrSignatureTimeout, "futuresWebsocketRequest must return the timeout")
		assert.True(t, conn.awaited, "the late reply should be awaited")
		assert.True(t, held, "the request lock should be held until the late reply is read, so that no other request is sent meanwhile")
	})
}

func TestFuturesWebsocketRequestSendFailure(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	errSendFailed := errors.New("websocket connection: cannot send message")
	conn := &futuresReplyConnection{err: errSendFailed}
	err := ex.futuresWebsocketRequest(t.Context(), conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}})
	require.ErrorIs(t, err, errSendFailed, "a request which could not be sent must return why")
	assert.False(t, conn.awaited, "a request which was never sent should not wait for a reply")
}

// futuresLateReplyVenue is a mock futures venue which acknowledges a BTC_USDT subscription once its hold has passed, or
// never when the hold is negative, refuses any other at once, and records the symbols it is sent
type futuresLateReplyVenue struct {
	hold    time.Duration
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
			name: "never answered",
			hold: -1,
			ctx: func(ctx context.Context) (context.Context, context.CancelFunc) {
				return context.WithTimeout(ctx, 50*time.Millisecond)
			},
			err:      context.DeadlineExceeded,
			returned: 50*time.Millisecond + limit,
			sent:     []string{"BTC_USDT", "BTC_USD"},
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
				assert.Equal(t, tc.sent, venue.sent(), "only a request whose context has not ended should be sent")
			})
		})
	}
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
		sub    *subscription.Subscription
		method string
		params WebsocketFuturesChannelParams
	}{
		{&subscription.Subscription{Channel: subscription.TickerChannel, Pairs: pair}, "ticker", WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}},
		{&subscription.Subscription{Channel: subscription.AllTradesChannel, Pairs: pair}, "deal", WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}},
		{&subscription.Subscription{Channel: subscription.OrderbookChannel, Pairs: pair}, "depth.full", WebsocketFuturesChannelParams{Symbol: "BTC_USDT", Limit: 20}},
		{&subscription.Subscription{Channel: subscription.OrderbookChannel, Pairs: pair, Levels: 5}, "depth.full", WebsocketFuturesChannelParams{Symbol: "BTC_USDT", Limit: 5}},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Pairs: pair, Interval: kline.FourHour}, "kline", WebsocketFuturesChannelParams{Symbol: "BTC_USDT", Interval: "Hour4"}},
	} {
		method, params, err := futuresChannelRequest(tc.sub)
		require.NoErrorf(t, err, "futuresChannelRequest must not error for %s", tc.sub.Channel)
		assert.Equalf(t, tc.method, method, "%s should be the venue's %s channel", tc.sub.Channel, tc.method)
		assert.Equalf(t, tc.params, *params, "%s should carry its parameters", tc.sub.Channel)
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
		qualified, err := futuresQualifiedChannel(tc.sub, pair)
		require.NoErrorf(t, err, "futuresQualifiedChannel must not error for %s", tc.sub.Channel)
		assert.Equalf(t, tc.exp, qualified, "%s should be qualified as the venue names it", tc.sub.Channel)
	}
	for _, tc := range []struct {
		sub *subscription.Subscription
		err error
	}{
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.TenMin}, kline.ErrUnsupportedInterval},
		{&subscription.Subscription{Channel: subscription.MyWalletChannel}, subscription.ErrNotSupported},
	} {
		qualified, err := futuresQualifiedChannel(tc.sub, pair)
		assert.ErrorIsf(t, err, tc.err, "futuresQualifiedChannel should return why the venue does not serve %s", tc.sub.Channel)
		assert.Emptyf(t, qualified, "futuresQualifiedChannel should qualify no channel for %s", tc.sub.Channel)
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
		for _, pairs := range []currency.Pairs{{futuresTradablePair}, {futuresTradablePair, futuresInversePair}} {
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
		}
	}
}

func TestGenerateSubscriptionsWildcardAsset(t *testing.T) {
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
	// A version 17 config holds the spot defaults, here beside a disabled subscription which names no default
	spotDefaults := slices.DeleteFunc(defaultSubscriptions.Clone(), func(s *subscription.Subscription) bool { return s.Asset != asset.Spot })
	previous, err := json.Marshal(spotDefaults)
	require.NoError(t, err, "Marshal must encode the spot defaults")
	features["subscriptions"] = json.RawMessage(strings.TrimSuffix(string(previous), "]") +
		`,{"enabled":false,"channel":"public.bookTicker.batch.v3.api.pb","asset":"spot"}]`)
	exchanges[i]["features"], err = json.Marshal(features)
	require.NoError(t, err, "Marshal must encode MEXC's features")
	saved["exchanges"], err = json.Marshal(exchanges)
	require.NoError(t, err, "Marshal must encode the saved exchanges")
	saved["version"] = json.RawMessage(`17`)
	data, err = json.Marshal(saved)
	require.NoError(t, err, "Marshal must encode the version 17 config")
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, data, 0o600), "WriteFile must save the version 17 config")

	var cfg config.Config
	require.NoError(t, cfg.ReadConfigFromFile(path, true), "ReadConfigFromFile must upgrade the version 17 config")
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
// key it is told to, pushes a frame after a subscription to a channel it holds one for, and records what it is sent
type futuresVenue struct {
	refuse string
	pushes map[string]string
	mu     sync.Mutex
	conn   *gws.Conn
	sent   []string
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
	v.mu.Unlock()
	switch {
	case req.Method == "ping":
		return v.write(conn, `{"channel":"pong","data":1792842501046,"ts":1792842501046}`)
	case strings.HasPrefix(req.Method, futuresUnsubscribePrefix):
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
		if login.APIKey != "test-key" || login.Signature != hex.EncodeToString(mac) || login.APIKey == v.refuse {
			return v.write(conn, `{"channel":"rs.error","data":"Apikey doesn't exist!","ts":1792842495741}`)
		}
		return v.write(conn, `{"channel":"rs.login","data":"success","ts":"1587442022003"}`)
	case strings.HasPrefix(req.Method, futuresSubscribePrefix):
		var params WebsocketFuturesChannelParams
		if err := json.Unmarshal(req.Param, &params); err != nil {
			return err
		}
		if params.Symbol == v.refuse {
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

func TestWebsocketFuturesSubscribePrivate(t *testing.T) {
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
}

func TestWebsocketFuturesSubscribePrivateRefused(t *testing.T) {
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
}
