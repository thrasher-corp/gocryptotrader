package mexc

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
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
	assert.Equal(t, 80591.4, tick.Last, "the last price should be decoded")
	assert.Equal(t, 80590.0, tick.Bid, "the best bid should be decoded")
	assert.Equal(t, 80591.5, tick.MarkPrice, "the fair price should be the mark price")
	assert.InDelta(t, 36180.0379, tick.BaseVolume, 1e-6, "the base volume should be the contracts times their size")
	assert.Equal(t, 2939324762.13142, tick.QuoteVolume, "the quote volume should be the turnover")
	assert.InDelta(t, 48966.8514, tick.OpenInterest, 1e-6, "the open interest should be the contracts held times their size")
	assert.Equal(t, int64(1792842497019), tick.LastUpdated.UnixMilli(), "the ticker's time should be decoded")
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
	require.Len(t, trades, 2, "every merged trade must be relayed")
	assert.Equal(t, "16745103311", trades[0].TID, "the trade ID should be decoded")
	assert.Equal(t, order.Sell, trades[0].Side, "a taker sell should be a sell")
	assert.Equal(t, 7739.0, trades[0].Amount, "the amount should be in contracts")
	assert.Equal(t, order.Buy, trades[1].Side, "a taker buy should be a buy")
	assert.True(t, trades[0].CurrencyPair.Equal(futuresTradablePair), "the contract should be decoded")

	require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresDealFrame)), "WebsocketFuturesHandleData must not error for an unmerged trade")
	trades, ok = relayed(t, ex).([]trade.Data)
	require.True(t, ok, "an unmerged trade must be relayed")
	require.Len(t, trades, 1, "the trade must be relayed")
	assert.Equal(t, int64(1792842497333), trades[0].Timestamp.UnixMilli(), "the trade's time should be decoded")
}

func TestWebsocketFuturesHandleOrderbook(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresDepthFrame)), "WebsocketFuturesHandleData must not error")

	book, err := orderbook.Get(ex.Name, futuresTradablePair, asset.Futures)
	require.NoError(t, err, "the snapshot must be loaded")
	assert.Equal(t, int64(42904820315), book.LastUpdateID, "the version should be the update ID")
	assert.Equal(t, int64(1792842498876), book.LastUpdated.UnixMilli(), "the matching engine's time should be the update time")
	assert.Equal(t, int64(1792842498880), book.LastPushed.UnixMilli(), "the push time should be recorded")
	require.Len(t, book.Asks, 2, "every ask must be loaded")
	assert.Equal(t, orderbook.Level{Price: 80590.1, Amount: 46440, OrderCount: 12}, book.Asks[0], "the best ask should be decoded")
	require.Len(t, book.Bids, 2, "every bid must be loaded")
	assert.Equal(t, 80590.0, book.Bids[0].Price, "the best bid should be decoded")
}

func TestWebsocketFuturesHandleCandle(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(futuresKlineFrame)), "WebsocketFuturesHandleData must not error")

	item, ok := relayed(t, ex).(kline.Item)
	require.True(t, ok, "a candle must be relayed")
	assert.True(t, item.Pair.Equal(futuresInversePair), "the contract should be decoded")
	assert.Equal(t, kline.FifteenMin, item.Interval, "the interval should be decoded")
	require.Len(t, item.Candles, 1, "the candle must be relayed")
	c := item.Candles[0]
	assert.Equal(t, time.Unix(1792842300, 0), c.Time, "the candle should be stamped with its open in seconds")
	assert.Equal(t, 80479.5, c.Open, "the open should be decoded")
	assert.Equal(t, 0.0708234966710365, c.Volume, "an inverse contract's base volume should be its turnover")
	assert.Equal(t, kline.PartialCandle, c.ValidationIssues, "a candle still forming should be marked partial")

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
	assert.Equal(t, "123456789", detail.OrderID, "the order ID should be decoded")
	assert.Equal(t, "ext_001", detail.ClientOrderID, "the external order ID should be the client order ID")
	assert.Equal(t, order.PartiallyFilled, detail.Status, "an open order with fills should be partially filled")
	assert.Equal(t, order.Limit, detail.Type, "the order type should be decoded")
	assert.Equal(t, 5.0, detail.ExecutedAmount, "the filled contracts should be decoded")
	assert.InDelta(t, 22.50000, detail.Cost, 1e-9, "the cost should be the filled contracts' value")
	assert.InDelta(t, 0.2025, detail.Fee, 1e-12, "the fee should be the order's taker and maker fees so far")
	assert.Equal(t, currency.USDT, detail.FeeAsset, "the fee currency should be decoded")
	assert.Equal(t, 5.0, detail.RemainingAmount, "the unfilled contracts should remain")
	assert.Equal(t, int64(1760942213000), detail.LastUpdated.UnixMilli(), "the update time should be decoded")

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
	require.Len(t, fills, 1, "the fill must be relayed")
	assert.Equal(t, "987654321", fills[0].TradeID, "the fill ID should be decoded")
	assert.Equal(t, "123456789", fills[0].OrderID, "the order ID should be decoded")
	assert.Equal(t, "ext_001", fills[0].ClientOrderID, "the external order ID should be decoded")
	assert.Equal(t, order.Sell, fills[0].Side, "closing a long should be a sell")
	assert.Equal(t, 10.0, fills[0].Amount, "the amount should be in contracts")
	assert.True(t, fills[0].CurrencyPair.Equal(futuresTradablePair), "the contract should be decoded")
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
	assert.Equal(t, 9000.0, b.Free, "the available balance should be free")
	assert.Equal(t, 6000.0, b.Hold, "frozen funds and position margin should be held")
	assert.Equal(t, 15000.0, b.Total, "the total should be what is free and held")

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
	require.Len(t, positions, 1, "the position must be relayed")
	p := positions[0]
	assert.Equal(t, order.Short, p.LatestDirection, "a short position should be short")
	assert.Equal(t, order.Open, p.Status, "a held position should be open")
	assert.Equal(t, 10.0, p.LatestSize.InexactFloat64(), "the size should be in contracts")
	assert.Equal(t, 20.0, p.Leverage.InexactFloat64(), "the leverage should be decoded")
	assert.Equal(t, 100.5, p.UnrealisedPNL.InexactFloat64(), "the unrealised PNL should be read from the push's pnl")
	assert.Equal(t, int64(3), p.UpdateID, "the version should be the update ID")
	assert.Equal(t, currency.USDT, p.CollateralCurrency, "the collateral should be the settlement currency")
	assert.True(t, p.CloseDate.IsZero(), "an open position should have no close date")

	require.NoError(t, ex.WebsocketFuturesHandleData(t.Context(), nil, []byte(strings.Replace(futuresPositionFrame, `"state":1`, `"state":3`, 1))),
		"WebsocketFuturesHandleData must not error for a closed position")
	positions, ok = relayed(t, ex).([]futures.Position)
	require.True(t, ok, "the closed position must be relayed")
	assert.Equal(t, order.Closed, positions[0].Status, "a closed position should be closed")
	assert.Equal(t, int64(1760942213000), positions[0].CloseDate.UnixMilli(), "a closed position should close at its update")
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

// futuresReplyConnection answers each futures websocket request with a recorded reply, and routes no reply to a
// waiting request
type futuresReplyConnection struct {
	websocket.Connection
	mu    sync.Mutex
	reply []byte
	sent  []*WebsocketFuturesRequest
}

// SendMessageReturnResponse records a request and returns the recorded reply
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
	c.sent = append(c.sent, req)
	return c.reply, nil
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
}

func TestFuturesWebsocketRequestAwaitsLateReply(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, c *gws.Conn) error {
			var req struct {
				Param WebsocketFuturesChannelParams `json:"param"`
			}
			if err := json.Unmarshal(msg, &req); err != nil {
				return err
			}
			if req.Param.Symbol == "BTC_USDT" {
				// Acknowledged after its request has given up on it
				time.Sleep(200 * time.Millisecond)
				return c.WriteMessage(gws.TextMessage, []byte(`{"channel":"rs.sub.ticker","data":"success","ts":1792842495741}`))
			}
			return c.WriteMessage(gws.TextMessage, []byte(`{"channel":"rs.error","data":"Contract [BTC_USD] not exists","ts":1792842495741}`))
		})
	}))
	// The connector dials with a dialer of its own, which reaches a listening server only
	server.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
	conn, err := ex.Websocket.CreateTestConnection(futuresPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.NoError(t, ex.WebsocketFuturesConnect(t.Context(), conn), "WebsocketFuturesConnect must not error")
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

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	err = ex.futuresWebsocketRequest(ctx, conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USDT"}})
	require.ErrorIs(t, err, context.DeadlineExceeded, "the unacknowledged request must time out")
	err = ex.futuresWebsocketRequest(t.Context(), conn, &WebsocketFuturesRequest{Method: "sub.ticker", Param: &WebsocketFuturesChannelParams{Symbol: "BTC_USD"}})
	assert.ErrorIs(t, err, errFuturesWebsocketRejected, "the next request should be answered by its own reply, not the late one")
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
	assert.Equal(t, futuresMethodLogin, conn.sent[0].Method, "the login method should be sent")
	params, ok := conn.sent[0].Param.(*WebsocketFuturesLoginParams)
	require.True(t, ok, "the login parameters must be sent")
	mac, err := crypto.GetHMAC(crypto.HashSHA256, []byte("test-key"+params.RequestTime), []byte("test-secret"))
	require.NoError(t, err, "GetHMAC must not error")
	assert.Equal(t, "test-key", params.APIKey, "the API key should be sent")
	assert.Equal(t, hex.EncodeToString(mac), params.Signature, "the login should be signed over the key and the request time")

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
	assert.Equal(t, 80591.4, tick.Last, "the pushed ticker should be stored")

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
	assert.Equal(t, "123456789", detail.OrderID, "the pushed order should be relayed")
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
