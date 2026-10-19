package bitfinex

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// newAccountTestExchange returns an exchange named after the test, with placeholder credentials and the fills feed on,
// and an account connection that is not connected, recording the JSON messages handling account events sends
func newAccountTestExchange(t *testing.T) (*Exchange, *sentConnection) {
	t.Helper()
	ex := newTestExchange(t)
	setTestCredentials(ex, &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})
	ex.Features.Enabled.FillsFeed = true
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	conn, err := ex.Websocket.CreateTestConnection(wsAccountConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	return ex, &sentConnection{Connection: conn}
}

// sentConnection is a connection recording the JSON messages sent through it, as encoded, rather than sending them
type sentConnection struct {
	websocket.Connection
	mu   sync.Mutex
	sent []string
}

// SendJSONMessage records the message
func (c *sentConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, data any) error {
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, string(msg))
	return nil
}

// messages returns the messages sent, in order, forgetting them
func (c *sentConnection) messages() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	sent := c.sent
	c.sent = nil
	return sent
}

// accountFixtureFrames returns testdata/wsAccount.json's frames, Bitfinex's documented account events, one a line
func accountFixtureFrames(t *testing.T) [][]byte {
	t.Helper()
	f, err := os.Open("testdata/wsAccount.json")
	require.NoError(t, err, "Opening the account fixture must not error")
	defer func() { assert.NoError(t, f.Close(), "Closing the account fixture should not error") }()
	var frames [][]byte
	s := bufio.NewScanner(f)
	for s.Scan() {
		frames = append(frames, bytes.Clone(s.Bytes()))
	}
	require.NoError(t, s.Err(), "Reading the account fixture must not error")
	return frames
}

func TestWebsocketHandleAccountEvents(t *testing.T) {
	t.Parallel()
	ex, conn := newAccountTestExchange(t)
	auth, err := conn.MatchReturnResponses(t.Context(), wsAuthSignature, 1)
	require.NoError(t, err, "MatchReturnResponses must not error")
	for _, frame := range accountFixtureFrames(t) {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, frame), "wsHandleData must not error for %s", frame)
	}
	assert.Contains(t, string(matchedResponse(t, auth)), `"status":"OK"`, "the auth reply should reach the authentication request")

	relayed := relayedPayloads(ex)
	require.Len(t, relayed, 23, "every account event must be relayed, a trade's execution update as a fill too")
	ethUSD := currency.NewPair(currency.ETH, currency.USD)
	assert.Equal(t, []order.Detail{{
		Exchange: ex.Name, OrderID: "34930659963", ClientOrderID: "1574955083558", Pair: ethUSD, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit,
		TimeInForce: order.PostOnly, Status: order.Active, Price: 120, Amount: 0.201104, RemainingAmount: 0.201104, Date: time.UnixMilli(1574955083558), LastUpdated: time.UnixMilli(1574955083573),
	}}, relayed[0], "the order snapshot should be relayed as order details, post only as its meta data reports")
	assert.Equal(t, &order.Detail{
		Exchange: ex.Name, OrderID: "1185815098", ClientOrderID: "1575287436979", Pair: ethUSD, AssetType: asset.Margin, Side: order.Sell, Type: order.Limit,
		Status: order.Active, Price: 230, Amount: 2.5, RemainingAmount: 2.5, Date: time.UnixMilli(1575287438515), LastUpdated: time.UnixMilli(1575287438519),
	}, relayed[1], "a new margin order should be relayed as an order detail")
	assert.IsType(t, &order.Detail{}, relayed[2], "an order update should be relayed as an order detail")
	cancelled, ok := relayed[3].(*order.Detail)
	require.True(t, ok, "an order close must be relayed as an order detail")
	assert.Equal(t, order.Cancelled, cancelled.Status, "a cancelled order should be relayed as cancelled")

	ethUST := currency.NewPair(currency.ETH, currency.UST)
	positions, ok := relayed[4].([]Position)
	require.True(t, ok, "the position snapshot must be relayed as positions")
	require.Len(t, positions, 1, "the position snapshot must hold its position")
	assert.Equal(t, uint64(142420429), positions[0].ID, "the snapshot's position should be relayed")
	assert.IsType(t, &Position{}, relayed[5], "a position update should be relayed as a position")

	assert.Equal(t, &AccountTrade{
		ID: 402088407, Symbol: "tETHUST", ExecutedAt: types.Time(time.UnixMilli(1574963975602)), OrderID: 34938060782, ExecutedAmount: -0.2, ExecutedPrice: 153.57,
		OrderType: OrderTypeMarket, Maker: -1,
	}, relayed[6], "a trade executed should be relayed without its fee")
	assert.IsType(t, &AccountTrade{}, relayed[7], "a trade execution update should be relayed as an account trade")
	assert.Equal(t, []fill.Data{{
		ID: "402088407", Timestamp: time.UnixMilli(1574963975602), Exchange: ex.Name, AssetType: asset.Margin, CurrencyPair: ethUST, Side: order.Sell,
		OrderID: "34938060782", ClientOrderID: "1714466193700", TradeID: "402088407", Price: 153.57, Amount: 0.2,
	}}, relayed[8], "a trade execution update should be relayed as a fill")

	assert.IsType(t, []FundingOffer{}, relayed[9], "the funding offer snapshot should be relayed as funding offers")
	assert.IsType(t, &FundingOffer{}, relayed[10], "a new funding offer should be relayed as a funding offer")
	assert.IsType(t, []FundingCredit{}, relayed[11], "the funding credit snapshot should be relayed as funding credits")
	assert.IsType(t, &FundingCredit{}, relayed[12], "a funding credit update should be relayed as a funding credit")
	assert.IsType(t, []FundingCredit{}, relayed[13], "the funding loan snapshot should be relayed as funding credits")

	wallets, ok := relayed[14].(accounts.SubAccounts)
	require.True(t, ok, "the wallet snapshot must be relayed as sub accounts")
	require.Len(t, wallets, 3, "the wallet snapshot must be relayed under each wallet's asset")
	for _, tc := range []struct {
		a                 asset.Item
		c                 currency.Code
		total, hold, free float64
	}{
		// A wallet whose available balance Bitfinex has yet to calculate, as the wallet update leaves BTC's, is held
		{a: asset.Spot, c: currency.BTC, total: 1.71169184, hold: 1.71169184},
		{a: asset.Spot, c: currency.NewCode("SAN"), total: 19.76, hold: 19.76},
		{a: asset.Margin, c: currency.UST, total: 100, hold: 20, free: 80},
		{a: asset.MarginFunding, c: currency.USD, total: 820, hold: 820},
	} {
		b, err := ex.Accounts.GetBalance("", &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret}, tc.a, tc.c)
		require.NoErrorf(t, err, "GetBalance must not error for %s %s", tc.a, tc.c)
		assert.InDeltaf(t, tc.total, b.Total, 1e-12, "the stored %s %s total should be the wallet's balance", tc.a, tc.c)
		assert.InDeltaf(t, tc.hold, b.Hold, 1e-12, "the stored %s %s hold should be what the wallet's available balance leaves", tc.a, tc.c)
		assert.InDeltaf(t, tc.free, b.Free, 1e-12, "the stored %s %s free balance should be the wallet's available balance", tc.a, tc.c)
	}
	updated, ok := relayed[15].(accounts.SubAccounts)
	require.True(t, ok, "a wallet update must be relayed as sub accounts")
	require.Len(t, updated, 1, "a wallet update must be relayed under its wallet's asset")
	assert.Equal(t, asset.Spot, updated[0].AssetType, "an exchange wallet update should be relayed for spot")
	assert.Equal(t, []string{`[0,"calc",null,[["wallet_exchange_SAN"]]]`, `[0,"calc",null,[["wallet_exchange_BTC"]]]`}, conn.messages(), "Bitfinex should be asked to calculate each wallet's available balance it has yet to")

	assert.Equal(t, &BalanceInfo{AssetsUnderManagement: 4131.85, NetAssetsUnderManagement: 4131.85}, relayed[16], "a balance update should be relayed as balance info")
	assert.Equal(t, &MarginInfoBaseResponse{Key: marginInfoKeyBase, Info: MarginInfoBase{UserProfitLoss: -13.014640000000007, MarginBalance: 49331.70267297, MarginNet: 49318.68803297, MarginMinimum: 27}}, relayed[17], "a margin summary should be relayed")
	assert.Equal(t, &MarginInfoSymbolResponse{Key: marginInfoKeySymbol, Symbol: "tETHUSD", Info: MarginInfoSymbol{TradableBalance: 149361.09689202666, GrossBalance: 149639.26293509, BuyAmount: 830.0182168075556, SellAmount: 895.0658432466332}}, relayed[18], "a trading pair's margin should be relayed")
	assert.Equal(t, &FundingInfoResponse{Key: "sym", Symbol: "fUSD", Info: FundingInfo{YieldLoan: 0.0008595462068208099, DurationLoan: 1.8444560185185186}}, relayed[19], "a funding info update should be relayed")
	assert.Equal(t, &AccountFundingTrade{ID: 636854, Symbol: "fUSD", CreatedAt: types.Time(time.UnixMilli(1575282446000)), OfferID: 41238905, Amount: -1000, Rate: 0.002, Period: 7}, relayed[20], "a funding trade should be relayed")
	notification, ok := relayed[21].(*Notification[json.RawMessage])
	require.True(t, ok, "a notification no request awaits must be relayed")
	assert.Equal(t, wsFundingOfferNewRequest, notification.Type, "the notification should be relayed as sent")
	alert, ok := relayed[22].(*Notification[json.RawMessage])
	require.True(t, ok, "a price alert must be relayed as a notification")
	assert.Equal(t, "uca", alert.Type, "the price alert should be relayed as sent")
}

func TestWebsocketWalletCalcs(t *testing.T) {
	t.Parallel()
	ex, conn := newAccountTestExchange(t)
	creds := &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret}
	uncalculated := []byte(`[0,"wu",["exchange","BTC",1,0,null,null,null]]`)
	require.NoError(t, ex.wsHandleData(t.Context(), conn, uncalculated), "a wallet update must be handled")
	assert.Equal(t, []string{`[0,"calc",null,[["wallet_exchange_BTC"]]]`}, conn.messages(), "a wallet's uncalculated available balance should be asked for")
	b, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.BTC)
	require.NoError(t, err, "GetBalance must not error")
	assert.Zero(t, b.Free, "a wallet whose available balance is uncalculated should not be free")
	assert.Equal(t, 1.0, b.Hold, "a wallet whose available balance is uncalculated should be held")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, uncalculated), "a wallet update must be handled")
	assert.Empty(t, conn.messages(), "an available balance already asked for should not be asked for again")

	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`[0,"wu",["exchange","BTC",1,0,0.25,null,null]]`)), "a wallet update must be handled")
	assert.Empty(t, conn.messages(), "a calculated available balance should not be asked for")
	b, err = ex.Accounts.GetBalance("", creds, asset.Spot, currency.BTC)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 0.25, b.Free, "a wallet's calculated available balance should be free")
	assert.Equal(t, 0.75, b.Hold, "what a wallet's calculated available balance leaves should be held")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, uncalculated), "a wallet update must be handled")
	assert.Len(t, conn.messages(), 1, "an available balance should be asked for again once one has arrived")

	ex.wsWalletCalcs.reset()
	disconnected, err := ex.Websocket.CreateTestConnection(wsAccountConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	assert.Error(t, ex.wsHandleData(t.Context(), disconnected, uncalculated), "a calc request that cannot be sent should error")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, uncalculated), "a wallet update must be handled")
	assert.Len(t, conn.messages(), 1, "an available balance whose request was not sent should be asked for again")

	var snapshot bytes.Buffer
	snapshot.WriteString(`[0,"ws",[`)
	for i := range wsMaxCalcRequests + 1 {
		if i > 0 {
			snapshot.WriteByte(',')
		}
		snapshot.WriteString(`["funding","C` + strconv.Itoa(i) + `",1,0,null,null,null]`)
	}
	snapshot.WriteString(`]]`)
	require.NoError(t, ex.wsHandleData(t.Context(), conn, snapshot.Bytes()), "a wallet snapshot must be handled")
	sent := conn.messages()
	require.Len(t, sent, 2, "the calculations must be asked for in as few requests as Bitfinex takes")
	var requests [][1]string
	require.NoError(t, json.Unmarshal([]byte(sent[0][len(`[0,"calc",null,`):len(sent[0])-1]), &requests), "the first calc request's calculations must decode")
	assert.Len(t, requests, wsMaxCalcRequests, "a calc request should ask for as many calculations as Bitfinex takes")
	assert.Equal(t, `[0,"calc",null,[["wallet_funding_C30"]]]`, sent[1], "the calculations left over should be asked for in the next request")
}

func TestWebsocketDerivativesWallet(t *testing.T) {
	t.Parallel()
	ex, conn := newAccountTestExchange(t)
	creds := &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret}
	// The margin wallet is also the derivatives wallet, holding each derivative's collateral as a derivative currency
	snapshot := []byte(`[0,"ws",[["margin","UST",100,0,80,null,null],["margin","USTF0",500,0,300,null,null],["margin","BTCF0",0.01,0,null,null,null]]]`)
	require.NoError(t, ex.wsHandleData(t.Context(), conn, snapshot), "a wallet snapshot must be handled")
	relayed := relayedPayloads(ex)
	require.Len(t, relayed, 1, "the wallet snapshot must be relayed")
	subAccts, ok := relayed[0].(accounts.SubAccounts)
	require.True(t, ok, "the wallet snapshot must be relayed as sub accounts")
	assets := make([]asset.Item, len(subAccts))
	for i := range subAccts {
		assets[i] = subAccts[i].AssetType
	}
	assert.ElementsMatch(t, []asset.Item{asset.Margin, asset.USDTMarginedFutures, asset.CoinMarginedFutures}, assets, "the margin wallet's balances should be relayed under the asset each funds")
	for _, tc := range []struct {
		a                 asset.Item
		c                 currency.Code
		total, hold, free float64
	}{
		{a: asset.Margin, c: currency.UST, total: 100, hold: 20, free: 80},
		{a: asset.USDTMarginedFutures, c: testDerivativePair.Quote, total: 500, hold: 200, free: 300},
		{a: asset.CoinMarginedFutures, c: testCoinMarginedPair.Quote, total: 0.01, hold: 0.01},
	} {
		b, err := ex.Accounts.GetBalance("", creds, tc.a, tc.c)
		require.NoErrorf(t, err, "GetBalance must not error for %s %s", tc.a, tc.c)
		assert.Equalf(t, tc.total, b.Total, "the stored %s %s total should be the wallet's balance", tc.a, tc.c)
		assert.Equalf(t, tc.hold, b.Hold, "the stored %s %s hold should be what the wallet's available balance leaves", tc.a, tc.c)
		assert.Equalf(t, tc.free, b.Free, "the stored %s %s free balance should be the wallet's available balance", tc.a, tc.c)
	}
	_, err := ex.Accounts.GetBalance("", creds, asset.Margin, testDerivativePair.Quote)
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "the derivatives' collateral should not be stored as a margin balance")
	assert.Equal(t, []string{`[0,"calc",null,[["wallet_margin_BTCF0"]]]`}, conn.messages(), "Bitfinex should be asked to calculate the collateral's available balance")
}

func TestWebsocketHandleAccountEventErrors(t *testing.T) {
	t.Parallel()
	ex, conn := newAccountTestExchange(t)
	for _, tc := range []struct {
		frame string
		err   error
	}{
		{frame: `[0,[1]]`, err: errWebsocketMalformedFrame},
		{frame: `[0,"hb`, err: errWebsocketMalformedFrame},
		{frame: `[0,"miu",["novel",[1]]]`, err: errWebsocketMalformedFrame},
		{frame: `[0,"miu",[]]`, err: errWebsocketMalformedFrame},
		{frame: `[0,"on",[1,null,null,"tETHUSD",null,null,1,1,"NOVEL"]]`, err: errUnknownOrderType},
		{frame: `[0,"os",[[1,null,null,"tETHUSD",null,null,0,0,"LIMIT"]]]`, err: errOrderSideUnresolved},
		{frame: `[0,"wu",["novel","BTC",1,0,null,null,null]]`, err: errUnknownWalletType},
		{frame: `[0,"tu",[1,"tETHUST",1574963975602,2,0,153.57,"MARKET",0,-1,0,"USD",0]]`, err: errOrderSideUnresolved},
		{frame: `[0,"tu",[1,"tX",1574963975602,2,1,153.57,"MARKET",0,-1,0,"USD",0]]`, err: errSymbolUnparsable},
		{frame: `[0,"bu",{}]`, err: errTupleNotArray},
	} {
		assert.ErrorIsf(t, ex.wsHandleData(t.Context(), conn, []byte(tc.frame)), tc.err, "wsHandleData should return the expected error for %s", tc.frame)
	}
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`[0,"novel",[]]`)), "an unknown account event must be handled")
	relayed := relayedPayloads(ex)
	require.NotEmpty(t, relayed, "an unknown account event must be relayed")
	assert.IsType(t, websocket.UnhandledMessageWarning{}, relayed[len(relayed)-1], "an unknown account event should be relayed as unhandled")

	quiet, quietConn := newAccountTestExchange(t)
	quiet.Features.Enabled.FillsFeed = false
	require.NoError(t, quiet.wsHandleData(t.Context(), quietConn, []byte(`[0,"tu",[402088407,"tETHUST",1574963975602,34938060782,-0.2,153.57,"MARKET",0,-1,-0.061668,"USD",0]]`)), "a trade execution update must be handled with the fills feed off")
	relayed = relayedPayloads(quiet)
	require.Len(t, relayed, 1, "only the account trade must be relayed with the fills feed off")
	assert.IsType(t, &AccountTrade{}, relayed[0], "the account trade should be relayed with the fills feed off")
}

func TestWebsocketHandleOrder(t *testing.T) {
	t.Parallel()
	ex, conn := newAccountTestExchange(t)
	const partial = `[1185815098,null,1575287436979,"tETHUSD",1575287438515,1575287438519,-0.75,-1,"LIMIT",null,null,null,0,"PARTIALLY FILLED @ 230.0(-0.25)",null,null,230,230,0,0,null,null,null,0,0,null,null,null,"API>BFX",null,null,null]`
	for _, tc := range []struct {
		closed bool
		exp    order.Status
	}{
		{exp: order.PartiallyFilled},
		{closed: true, exp: order.PartiallyFilledCancelled},
	} {
		require.NoErrorf(t, ex.wsHandleOrder(t.Context(), tc.closed, []byte(`[0,"ou",`+partial+`]`)), "wsHandleOrder must not error, closed %t", tc.closed)
		relayed := relayedPayloads(ex)
		require.Lenf(t, relayed, 1, "wsHandleOrder must relay the order, closed %t", tc.closed)
		d, ok := relayed[0].(*order.Detail)
		require.Truef(t, ok, "wsHandleOrder must relay the order as an order detail, closed %t", tc.closed)
		assert.Equalf(t, tc.exp, d.Status, "wsHandleOrder should report a partially filled order %s, closed %t", tc.exp, tc.closed)
	}
	for event, exp := range map[string]order.Status{"on": order.PartiallyFilled, "ou": order.PartiallyFilled, "oc": order.PartiallyFilledCancelled} {
		require.NoErrorf(t, ex.wsHandleAccountEvent(t.Context(), conn, []byte(`[0,"`+event+`",`+partial+`]`)), "wsHandleAccountEvent must not error for an %s event", event)
		relayed := relayedPayloads(ex)
		require.Lenf(t, relayed, 1, "wsHandleAccountEvent must relay the order of an %s event", event)
		d, ok := relayed[0].(*order.Detail)
		require.Truef(t, ok, "wsHandleAccountEvent must relay the order of an %s event as an order detail", event)
		assert.Equalf(t, exp, d.Status, "wsHandleAccountEvent should report a partially filled order of an %s event %s", event, exp)
	}
	assert.ErrorIs(t, ex.wsHandleOrder(t.Context(), true, []byte(`[0,"oc",{}]`)), errTupleNotArray, "wsHandleOrder should refuse an order that is not an array")
	assert.ErrorIs(t, ex.wsHandleOrder(t.Context(), true, []byte(`[0,"oc",[1,null,null,"tETHUSD",null,null,1,1,"NOVEL"]]`)), errUnknownOrderType, "wsHandleOrder should refuse an order of an unknown type")
}

func TestWebsocketNotificationSignature(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		notification string
		exp          string
	}{
		{notification: `[1575287438515,"on-req",null,null,[1185815098,null,1575287436979,"tETHUSD"],null,"SUCCESS",""]`, exp: "on-req:1575287436979"},
		{notification: `[1575287438515,"ou-req",null,null,[1185815100,null,1575289350475],null,"SUCCESS",""]`, exp: "ou-req:1185815100"},
		{notification: `[1575287438515,"oc-req",null,null,[1185815100],null,"SUCCESS",""]`, exp: "oc-req:1185815100"},
		{notification: `[1575287438515,"foc-req",null,null,[41238934,"fUST"],null,"SUCCESS",""]`, exp: "foc-req:41238934"},
		{notification: `[1575287438515,"fon-req",null,null,[41238934],null,"SUCCESS",""]`, exp: wsFundingOfferNewRequest},
		{notification: `[1575287438515,"oc_multi-req",null,null,[],null,"SUCCESS",""]`, exp: wsOrderCancelMultiRequest},
		{notification: `[1575287438515,"on-req",null,null,[null,null,null],null,"ERROR",""]`, exp: ""},
		{notification: `[1575287438515,"uca",null,null,["k","price"],null,"SUCCESS",""]`, exp: ""},
	} {
		var n Notification[json.RawMessage]
		require.NoErrorf(t, n.UnmarshalJSON([]byte(tc.notification)), "UnmarshalJSON must not error for %s", tc.notification)
		assert.Equalf(t, tc.exp, wsNotificationSignature(&n), "wsNotificationSignature should match %s with its request", tc.notification)
	}
}
