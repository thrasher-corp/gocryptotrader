package gateio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
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
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestProcessFuturesTickers(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	payload := []byte(`{"time":1541659086,"channel":"futures.tickers","event":"update","result":[{"contract":"BTC_USDT","last":"118.4","mark_price":"118.35","index_price":"118.36","volume_24h_quote":"1665006","volume_24h_base":"5526","low_24h":"99.2","high_24h":"132.5"}]}`)
	require.NoError(t, ex.processFuturesTickers(t.Context(), payload, asset.USDTMarginedFutures), "futures ticker processing must succeed")

	select {
	case msg := <-ex.Websocket.DataHandler.C:
		got, ok := msg.Data.([]ticker.Price)
		require.True(t, ok, "message must contain futures ticker prices")
		require.Len(t, got, 1, "message must contain one futures ticker")
		assert.Equal(t, 118.35, got[0].MarkPrice, "mark price should match the response")
		assert.Equal(t, 118.36, got[0].IndexPrice, "index price should match the response")
		assert.Equal(t, asset.USDTMarginedFutures, got[0].AssetType, "asset should be USDT margined futures")
		assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), got[0].Pair, "ticker pair should match the response")
		assert.Equal(t, time.Unix(1541659086, 0), got[0].LastUpdated, "ticker timestamp should match the response")
	default:
		require.Fail(t, "WebSocket futures ticker payload must be emitted")
	}
}

func TestGenerateFuturesPayload(t *testing.T) {
	t.Parallel()

	t.Run("empty channels", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, nil)
		require.ErrorIs(t, err, errNoChannelsSupplied)
	})

	t.Run("not single pair", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresTickersChannel, Pairs: nil},
		})
		require.ErrorIs(t, err, subscription.ErrNotSinglePair)
	})

	t.Run("frequency invalid interval", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresOrderbookUpdateChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"frequency": kline.Interval(time.Duration(-1))},
			},
		})
		require.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	})

	t.Run("candlestick interval invalid", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresCandlesticksChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"interval": kline.Interval(time.Duration(-1))},
			},
		})
		require.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	})

	t.Run("orderbook update with snapshot missing level", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrderbookV2, Pairs: currency.Pairs{BTCUSDT}, Params: map[string]any{}},
		})
		require.ErrorIs(t, err, common.ErrParameterRequired)
	})

	t.Run("orderbook update with snapshot bad level type", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrderbookV2, Pairs: currency.Pairs{BTCUSDT}, Params: map[string]any{"level": 50}},
		})
		require.ErrorIs(t, err, common.ErrTypeAssertFailure)
	})

	t.Run("orderbook update with snapshot empty pair", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrderbookV2, Pairs: currency.Pairs{currency.EMPTYPAIR}, Params: map[string]any{"level": uint64(50)}},
		})
		require.ErrorIs(t, err, common.ErrParameterRequired)
	})

	t.Run("happy path unauthenticated - params", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadTest"
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)

		got, err := ex.generateFuturesPayload(context.Background(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresOrderbookUpdateChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params: map[string]any{
					"frequency": kline.TwentyMilliseconds,
					"level":     "20",
					"limit":     100,
					"accuracy":  "0",
				},
			},
			&subscription.Subscription{
				Channel: futuresCandlesticksChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"interval": kline.FiveMin},
			},
			&subscription.Subscription{
				Channel: futuresOrderbookChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"interval": "0", "limit": 100},
			},
			&subscription.Subscription{
				Channel: futuresOrderbookV2,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"level": uint64(50)},
			},
		})
		require.NoError(t, err, "generateFuturesPayload must not error")
		require.Len(t, got, 4)

		for i := range got {
			require.NotZero(t, got[i].ID)
			require.Equal(t, subscribeEvent, got[i].Event)
			require.NotEmpty(t, got[i].Channel)
			require.NotZero(t, got[i].Time)
			require.Nil(t, got[i].Auth, "Auth must be nil when unauthenticated")
			require.NotEmpty(t, got[i].Payload, "Payload must not be empty")
		}

		require.Equal(t, []string{BTCUSDT.String(), "20ms", "20", "100", "0"}, got[0].Payload)
		require.Equal(t, []string{"5m", BTCUSDT.String()}, got[1].Payload)
		require.Equal(t, []string{BTCUSDT.String(), "100", "0"}, got[2].Payload)
		require.Equal(t, []string{"ob." + BTCUSDT.String() + ".50"}, got[3].Payload)
	})

	t.Run("authenticated channel - missing creds disables auth", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadAuthDisableTest"

		// Force path into GetCredentials() by allowing authenticated endpoints.
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

		got, err := ex.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresBalancesChannel,
				Pairs:   currency.Pairs{BTCUSDT},
			},
		})
		require.NoError(t, err, "generateFuturesPayload must not error")
		require.Len(t, got, 1)
		require.Nil(t, got[0].Auth, "Auth must be nil when GetCredentials fails")
		require.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticated endpoints must be disabled on GetCredentials error")
	})

	t.Run("authenticated channel - user param inserted + signature", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadAuthTest"
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})

		got, err := ex.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresBalancesChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"user": "user123"},
			},
		})
		require.NoError(t, err, "generateFuturesPayload must not error")
		require.Len(t, got, 1)

		require.NotNil(t, got[0].Auth, "Auth must not be nil when authenticated")
		require.Equal(t, "api_key", got[0].Auth.Method)
		require.Equal(t, "key", got[0].Auth.Key)
		require.NotEmpty(t, got[0].Auth.Sign)

		require.Equal(t, []string{"user123", BTCUSDT.String()}, got[0].Payload)

		sig, err := ex.generateWsSignature("secret", subscribeEvent, futuresBalancesChannel, got[0].Time)
		require.NoError(t, err)
		require.Equal(t, sig, got[0].Auth.Sign)
	})

	t.Run("authenticated positions all contracts", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadAllPositionsTest"
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})

		got, err := ex.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresPositionsChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params: map[string]any{
					contractPayloadOverrideParam: allFuturesContracts,
					requiresUserPlaceholderParam: true,
				},
			},
		})
		require.NoError(t, err, "generateFuturesPayload must not error")
		require.Len(t, got, 1, "all-contract positions must generate one payload")
		require.Equal(t, []string{"", "!all"}, got[0].Payload,
			"all-contract positions payload must use the documented selector")
		require.NotNil(t, got[0].Auth, "all-contract positions payload must be authenticated")
	})

	t.Run("authenticated position closes require user ID", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadPositionClosesTest"
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})

		_, err := ex.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresAutoPositionCloseChannel,
				Pairs:   currency.Pairs{BTCUSDT},
			},
		})
		require.ErrorIs(t, err, common.ErrParameterRequired,
			"position closes payload without a user ID must error")
	})
}

func TestWebsocketHandleFuturesDataRelaysResult(t *testing.T) {
	t.Parallel()
	contract := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		channel string
		result  string
		exp     any
	}{
		{
			channel: futuresOrderbookTickerChannel,
			result:  `{"t":1615366379123,"u":2517661076,"s":"BTC_USDT","b":"54696.6","B":"37000.5","a":"54696.7","A":"47061.25"}`,
			exp:     &WebsocketFuturesOrderbookTicker{Timestamp: types.Time(time.UnixMilli(1615366379123)), UpdateID: 2517661076, CurrencyPair: "BTC_USDT", BestBidPrice: 54696.6, BestBidAmount: 37000.5, BestAskPrice: 54696.7, BestAskAmount: 47061.25},
		},
		{
			channel: futuresLiquidatesChannel,
			result:  `[{"entry_price":209,"fill_price":215.1,"left":"3.5","leverage":"10","liq_price":213,"margin":0.007816722941,"mark_price":213.5,"order_id":4093362,"order_price":215.2,"size":"-124.5","time":1541486601,"time_ms":1541486601123,"contract":"BTC_USDT","user":"1040"}]`,
			exp: &websocketResponse[[]*WebsocketFuturesLiquidationNotification]{
				Time: types.Time(time.Unix(1630576300, 0)), Channel: futuresLiquidatesChannel, Event: "update",
				Result: []*WebsocketFuturesLiquidationNotification{{
					EntryPrice: 209, FillPrice: 215.1, RemainingAmount: 3.5, Leverage: 10, LiquidationPrice: 213, Margin: 0.007816722941,
					MarkPrice: 213.5, OrderID: 4093362, OrderPrice: 215.2, Size: -124.5, Time: types.Time(time.UnixMilli(1541486601123)),
					Contract: contract, User: "1040",
				}},
			},
		},
		{
			// Gate documents the sizes as strings or integers
			channel: futuresAutoDeleveragesChannel + " integers",
			result:  `[{"entry_price":209,"fill_price":215.1,"position_size":10,"trade_size":-2,"time":1541486601,"time_ms":1541486601123,"contract":"BTC_USDT","user":"1040"}]`,
			exp: &websocketResponse[[]*WebsocketFuturesAutoDeleveragesNotification]{
				Time: types.Time(time.Unix(1630576300, 0)), Channel: futuresAutoDeleveragesChannel, Event: "update",
				Result: []*WebsocketFuturesAutoDeleveragesNotification{{
					EntryPrice: 209, FillPrice: 215.1, PositionSize: 10, TradeSize: -2, Time: types.Time(time.UnixMilli(1541486601123)),
					Contract: contract, User: "1040",
				}},
			},
		},
		{
			// Size decimal connections receive both sizes as strings
			channel: futuresAutoDeleveragesChannel,
			result:  `[{"entry_price":209,"fill_price":215.1,"position_size":"10.5","trade_size":"-2.25","time":1541486601,"time_ms":1541486601123,"contract":"BTC_USDT","user":"1040"}]`,
			exp: &websocketResponse[[]*WebsocketFuturesAutoDeleveragesNotification]{
				Time: types.Time(time.Unix(1630576300, 0)), Channel: futuresAutoDeleveragesChannel, Event: "update",
				Result: []*WebsocketFuturesAutoDeleveragesNotification{{
					EntryPrice: 209, FillPrice: 215.1, PositionSize: 10.5, TradeSize: -2.25, Time: types.Time(time.UnixMilli(1541486601123)),
					Contract: contract, User: "1040",
				}},
			},
		},
	} {
		t.Run(tc.channel, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			push := `{"time":1630576300,"time_ms":1630576300123,"channel":"` + strings.Fields(tc.channel)[0] + `","event":"update","result":` + tc.result + `}`
			require.NoError(t, ex.WebsocketHandleFuturesData(t.Context(), nil, []byte(push), asset.USDTMarginedFutures), "WebsocketHandleFuturesData must not error")
			require.Len(t, ex.Websocket.DataHandler.C, 1, "WebsocketHandleFuturesData must relay the push")
			assert.Equal(t, tc.exp, (<-ex.Websocket.DataHandler.C).Data, "WebsocketHandleFuturesData should relay the decoded push")
		})
	}
}

func TestWebsocketHandleFuturesDataNullResult(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	var err error
	require.NotPanics(t, func() {
		err = ex.WebsocketHandleFuturesData(t.Context(), nil, []byte(`{"time":1,"channel":"futures.order_book","event":"all","result":null}`), asset.USDTMarginedFutures)
	}, "WebsocketHandleFuturesData must not panic on a null orderbook snapshot")
	assert.ErrorContains(t, err, "currency pair not set", "WebsocketHandleFuturesData should reject a null orderbook snapshot")
}

// dialRecorderConnection records a dial in place of connecting
type dialRecorderConnection struct {
	websocket.Connection
	url     string
	dialErr error
	header  http.Header
	dialled bool
	pinged  bool
}

func (c *dialRecorderConnection) Dial(_ context.Context, _ *gws.Dialer, header http.Header, _ url.Values) error {
	c.dialled = true
	c.header = header
	return c.dialErr
}

func (c *dialRecorderConnection) GetURL() string { return c.url }

func (c *dialRecorderConnection) SetupPingHandler(request.EndpointLimit, websocket.PingHandler) {
	c.pinged = true
}

func TestWebsocketFuturesConnect(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")

	// A configured coin margined URL no longer decides which asset the connection checks
	conn := &dialRecorderConnection{url: "wss://proxy.example/v4/ws/btc"}
	require.NoError(t, ex.WebsocketFuturesConnect(t.Context(), conn, asset.CoinMarginedFutures), "WebsocketFuturesConnect must dial an enabled coin margined connection")
	assert.True(t, conn.dialled, "WebsocketFuturesConnect should dial")
	assert.Equal(t, "1", conn.header.Get(sizeDecimalHeader), "WebsocketFuturesConnect should request decimal sizes")
	assert.True(t, conn.pinged, "WebsocketFuturesConnect should set up the ping handler")

	conn = &dialRecorderConnection{url: usdtFuturesWebsocketURL}
	assert.ErrorIs(t, ex.WebsocketFuturesConnect(t.Context(), conn, asset.USDTMarginedFutures), asset.ErrNotEnabled, "WebsocketFuturesConnect should reject a disabled asset")
	assert.False(t, conn.dialled, "WebsocketFuturesConnect should not dial for a disabled asset")

	errDial := errors.New("dial failed")
	conn = &dialRecorderConnection{url: btcFuturesWebsocketURL, dialErr: errDial}
	assert.ErrorIs(t, ex.WebsocketFuturesConnect(t.Context(), conn, asset.CoinMarginedFutures), errDial, "WebsocketFuturesConnect should return a dial error")
	assert.False(t, conn.pinged, "WebsocketFuturesConnect should not set up the ping handler after a failed dial")
}

func TestProcessFuturesTrades(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.SetTradeFeedStatus(false)
	ex.SetSaveTradeDataStatus(false)
	require.NoError(t, ex.processFuturesTrades([]byte(`{`), asset.CoinMarginedFutures), "processFuturesTrades must ignore pushes while the trade feed and saving are off")
	ex.SetTradeFeedStatus(true)
	assert.Error(t, ex.processFuturesTrades([]byte(`{`), asset.CoinMarginedFutures), "processFuturesTrades should reject malformed JSON")
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	// Gate signs the size by the taker's side, so a negative size is a taker sell
	require.NoError(t, ex.processFuturesTrades([]byte(`{"channel":"futures.trades","event":"update","time":1541503698,"result":[{"size":-108,"id":27753479,"create_time":1545136464,"create_time_ms":1545136464123,"price":"96.4","contract":"BTC_USD"},{"size":"2.5","id":27753480,"create_time_ms":1545136465123,"price":"96.5","contract":"BTC_USD"}]}`), asset.CoinMarginedFutures), "processFuturesTrades must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "processFuturesTrades must relay the trades")
	pair := currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter)
	exp := []trade.Data{
		{TID: "27753479", Exchange: ex.Name, CurrencyPair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Sell, Price: 96.4, Amount: 108, Timestamp: time.UnixMilli(1545136464123)},
		{TID: "27753480", Exchange: ex.Name, CurrencyPair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Price: 96.5, Amount: 2.5, Timestamp: time.UnixMilli(1545136465123)},
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processFuturesTrades should relay the taker side and an unsigned amount")
}

func TestProcessFuturesOrderbookSnapshot(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	btcusdt := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	ethusdt := currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter)

	assert.Error(t, ex.processFuturesOrderbookSnapshot(t.Context(), "all", []byte(`{`), asset.USDTMarginedFutures, time.Now()), "processFuturesOrderbookSnapshot should reject a malformed snapshot")
	require.NoError(t, ex.processFuturesOrderbookSnapshot(t.Context(), "all", []byte(`{"t":1615366379123,"contract":"BTC_USDT","id":2,"asks":[{"p":"54696.7","s":1}],"bids":[{"p":"54696.6","s":2}]}`), asset.USDTMarginedFutures, time.Now()), "processFuturesOrderbookSnapshot must load a full snapshot")
	ob, err := ex.Websocket.Orderbook.GetOrderbook(btcusdt, asset.USDTMarginedFutures)
	require.NoError(t, err, "GetOrderbook must find the snapshot")
	assert.Equal(t, orderbook.Levels{{Price: 54696.6, Amount: 2}}, ob.Bids, "processFuturesOrderbookSnapshot should load the snapshot's bids")
	assert.Equal(t, orderbook.Levels{{Price: 54696.7, Amount: 1}}, ob.Asks, "processFuturesOrderbookSnapshot should load the snapshot's asks")

	assert.Error(t, ex.processFuturesOrderbookSnapshot(t.Context(), "update", []byte(`{`), asset.USDTMarginedFutures, time.Now()), "processFuturesOrderbookSnapshot should reject a malformed update")
	assert.ErrorContains(t, ex.processFuturesOrderbookSnapshot(t.Context(), "update", []byte(`[]`), asset.USDTMarginedFutures, time.Now()), "missing orderbook ask and bid data", "processFuturesOrderbookSnapshot should reject an empty update")
	assert.ErrorIs(t, ex.processFuturesOrderbookSnapshot(t.Context(), "update", []byte(`[{"p":"1","s":1,"c":"_"}]`), asset.USDTMarginedFutures, time.Now()), currency.ErrCreatingPair, "processFuturesOrderbookSnapshot should reject an invalid contract")
	// A positive size is a bid and a negative one an ask; each contract gets its own book
	require.NoError(t, ex.processFuturesOrderbookSnapshot(t.Context(), "update", []byte(`[{"p":"100","s":3,"c":"ETH_USDT","id":1},{"p":"101","s":-4,"c":"ETH_USDT","id":2},{"p":"54000","s":5,"c":"BTC_USDT","id":3},{"p":"99","s":6,"c":"ETH_USDT","id":4},{"p":"102","s":-7,"c":"ETH_USDT","id":5}]`), asset.USDTMarginedFutures, time.Now()), "processFuturesOrderbookSnapshot must load an update event")
	ob, err = ex.Websocket.Orderbook.GetOrderbook(ethusdt, asset.USDTMarginedFutures)
	require.NoError(t, err, "GetOrderbook must find the ETH_USDT book")
	assert.Equal(t, orderbook.Levels{{Price: 100, Amount: 3}, {Price: 99, Amount: 6}}, ob.Bids, "processFuturesOrderbookSnapshot should load every positive size as a bid")
	assert.Equal(t, orderbook.Levels{{Price: 101, Amount: 4}, {Price: 102, Amount: 7}}, ob.Asks, "processFuturesOrderbookSnapshot should load every negative size as an unsigned ask")
	ob, err = ex.Websocket.Orderbook.GetOrderbook(btcusdt, asset.USDTMarginedFutures)
	require.NoError(t, err, "GetOrderbook must find the BTC_USDT book")
	assert.Equal(t, orderbook.Levels{{Price: 54000, Amount: 5}}, ob.Bids, "processFuturesOrderbookSnapshot should replace the BTC_USDT book")
	assert.Empty(t, ob.Asks, "processFuturesOrderbookSnapshot should replace the BTC_USDT asks")

	ex.ValidateOrderbook = true
	assert.ErrorIs(t, ex.processFuturesOrderbookSnapshot(t.Context(), "update", []byte(`[{"p":"0","s":1,"c":"BTC_USDT"}]`), asset.USDTMarginedFutures, time.Now()), orderbook.ErrPriceZero, "processFuturesOrderbookSnapshot should return a failure to load the book")
}

func TestWebsocketHandleFuturesDataErrors(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	assert.ErrorIs(t, ex.WebsocketHandleFuturesData(t.Context(), nil, []byte(`{"time_ms":"x","channel":"futures.tickers"}`), asset.USDTMarginedFutures), strconv.ErrSyntax, "WebsocketHandleFuturesData should reject an invalid header")
	assert.ErrorContains(t, ex.WebsocketHandleFuturesData(t.Context(), nil, []byte(`{"time":1541505434,"channel":"futures.orders","event":"update","result":[{"status":"bogus"}]}`), asset.USDTMarginedFutures), "unrecognised order status", "WebsocketHandleFuturesData should return an orders push failure")
	require.Empty(t, ex.Websocket.DataHandler.C, "WebsocketHandleFuturesData must not relay a failed push")

	require.NoError(t, ex.WebsocketHandleFuturesData(t.Context(), nil, []byte(`{"time":1541505434,"channel":"futures.unknown","event":"update","result":[]}`), asset.USDTMarginedFutures), "WebsocketHandleFuturesData must not error on an unknown channel")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "WebsocketHandleFuturesData must relay an unhandled message warning")
	_, ok := (<-ex.Websocket.DataHandler.C).Data.(websocket.UnhandledMessageWarning)
	assert.True(t, ok, "WebsocketHandleFuturesData should relay an unknown channel as an unhandled message")
}

func TestWebsocketFuturesPointNumbers(t *testing.T) {
	t.Parallel()
	for name, value := range map[string]string{"number": `0.25`, "string": `"0.25"`} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var tradeUpdate WebsocketFuturesUserTrade
			require.NoError(t, json.Unmarshal([]byte(`{"point_fee":`+value+`}`), &tradeUpdate), "fractional point_fee must decode")
			assert.Equal(t, types.Number(0.25), tradeUpdate.PointFee, "point_fee should retain the fraction")

			var position WebsocketFuturesPosition
			loss := strings.ReplaceAll(value, "0.25", "-0.25")
			require.NoError(t, json.Unmarshal([]byte(`{"history_point":`+loss+`}`), &position), "fractional history_point must decode")
			assert.Equal(t, types.Number(-0.25), position.HistoryPoint, "history_point should retain the negative fraction")
		})
	}
}

// TestFuturesDataHandler ensures that messages from various futures channels do not error
func TestFuturesDataHandler(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsFutures.json", func(ctx context.Context, m []byte) error {
		if strings.Contains(string(m), "futures.balances") {
			ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "test", Secret: "test"})
		}
		return e.WebsocketHandleFuturesData(ctx, nil, m, asset.CoinMarginedFutures)
	})
	e.Websocket.DataHandler.Close()
	assert.Len(t, e.Websocket.DataHandler.C, 15, "Should see the correct number of messages")
	var sawPosition, sawPositionClose bool
	for resp := range e.Websocket.DataHandler.C {
		if err, isErr := resp.Data.(error); isErr {
			assert.NoError(t, err, "Should not get any errors down the data handler")
		}
		if positions, ok := resp.Data.([]futures.Position); ok {
			require.Len(t, positions, 1, "position update must contain one position")
			assert.Equal(t, asset.CoinMarginedFutures, positions[0].Asset, "asset should match the websocket")
			assert.Equal(t, "BTC_USD", positions[0].Pair.String(), "pair should be normalised")
			assert.Equal(t, currency.BTC, positions[0].Underlying, "underlying should be normalised")
			assert.Equal(t, currency.BTC, positions[0].CollateralCurrency, "collateral currency should be normalised")
			assert.Equal(t, order.Long, positions[0].LatestDirection, "direction should be normalised")
			if positions[0].CloseDate.IsZero() {
				sawPosition = true
				assert.Equal(t, order.Open, positions[0].Status, "position status should be open")
				assert.Equal(t, "3", positions[0].LatestSize.String(), "size should be normalised")
				assert.Equal(t, "5", positions[0].Leverage.String(), "replacement cross-margin leverage should take precedence")
				assert.True(t, positions[0].PositionMargin.Equal(decimal.MustFromFloat(49.999890611186)), "position margin should be populated")
				assert.True(t, positions[0].MaintenanceMarginFraction.Equal(decimal.MustFromFloat(0.005)), "maintenance margin rate should be populated")
				assert.True(t, positions[0].EstimatedLiquidationPrice.Equal(decimal.MustFromFloat(0.1)), "liquidation price should be populated")
				assert.Equal(t, int64(42), positions[0].UpdateID, "update ID should be populated")
			} else {
				sawPositionClose = true
				assert.Equal(t, order.Closed, positions[0].Status, "position close status should be closed")
				assert.True(t, positions[0].LatestSize.IsZero(), "close size should be zero")
			}
		}
	}
	require.True(t, sawPosition, "futures fixture must emit a normalised position")
	require.True(t, sawPositionClose, "futures fixture must emit a normalised position close")
}

func TestProcessPositionCloseData(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name       string
		text       string
		riskStatus order.Status
	}{
		{name: "web", text: "web"},
		{name: "liquidation", text: "liquidation", riskStatus: order.Liquidated},
		{name: "liquidation prefix", text: "liq-123", riskStatus: order.Liquidated},
		{name: "hedge liquidation prefix", text: "hedge-liq-123", riskStatus: order.Liquidated},
		{name: "portfolio margin liquidation", text: "pm_liquidate", riskStatus: order.Liquidated},
		{name: "combined margin liquidation", text: "comb_margin_liquidate", riskStatus: order.Liquidated},
		{name: "single currency margin liquidation", text: "scm_liquidate", riskStatus: order.Liquidated},
		{name: "insurance", text: "insurance", riskStatus: order.Liquidated},
		{name: "auto deleveraging", text: "auto_deleveraging", riskStatus: order.AutoDeleverage},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
			payload := []byte(`{"time":1541505434,"channel":"futures.position_closes","event":"update","result":[{"contract":"BTC_USD","pnl":-0.000624354791,"side":"long","text":"` + tt.text + `","time":1547198562,"time_ms":1547198562123,"user":"211xxxx"}]}`)
			require.NoError(t, ex.processPositionCloseData(t.Context(), payload, asset.CoinMarginedFutures),
				"processPositionCloseData must process the close update")
			message := <-ex.Websocket.DataHandler.C
			positions, ok := message.Data.([]futures.Position)
			require.True(t, ok, "position close must emit canonical futures positions")
			require.Len(t, positions, 1, "position close must emit one position")
			assert.Equal(t, order.Closed, positions[0].Status, "position close status should identify a completed lifecycle")
			assert.Equal(t, tt.riskStatus, positions[0].RiskStatus, "position risk status should preserve its close cause")
			assert.Equal(t, order.Long, positions[0].OpeningDirection, "opening direction should be preserved")
			assert.Equal(t, order.Long, positions[0].LatestDirection, "latest direction should be preserved")
		})
	}
}

func TestProcessFuturesCandlesticksIntervalMapping(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")

	payload := []byte(`{"time":1606292600,"channel":"futures.candlesticks","event":"update","result":[{"t":1606292580,"v":"2362.32035","c":"19128.1","h":"19128.1","l":"19128.1","o":"19128.1","n":"1m_BTC_USDT"}]}`)
	require.NoError(t, ex.processFuturesCandlesticks(t.Context(), payload, asset.USDTMarginedFutures))

	select {
	case msg := <-ex.Websocket.DataHandler.C:
		got, ok := msg.Data.([]kline.Item)
		expectedPair := currency.NewPairWithDelimiter("BTC", "USDT", "_")
		require.True(t, ok, "expected []kline.Item")
		assert.Equal(t, []kline.Item{{
			Pair:     expectedPair,
			Asset:    asset.USDTMarginedFutures,
			Exchange: ex.Name,
			Interval: kline.OneMin,
			Candles: []kline.Candle{{
				Time:   time.Unix(1606292580, 0),
				Open:   19128.1,
				Close:  19128.1,
				High:   19128.1,
				Low:    19128.1,
				Volume: 2362.32035,
			}},
		}}, got)
	default:
		require.Fail(t, "expected futures websocket candle payload")
	}
}

func TestGenerateFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	_, err := e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Options must not use the perpetual futures subscription generator")
	subs, err := e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	var orderbooks int
	for _, sub := range subs {
		if sub.Channel != futuresOrderbookV2 {
			continue
		}
		orderbooks++
		assert.Equal(t, "ob."+sub.Pairs[0].String()+".50", sub.QualifiedChannel, "V2 recovery key should identify the pair and depth")
		for _, event := range []string{subscribeEvent, unsubscribeEvent} {
			payload, err := e.generateFuturesPayload(t.Context(), event, subscription.List{sub})
			require.NoError(t, err, "V2 payload generation must succeed")
			require.Len(t, payload, 1, "V2 subscription must generate one request")
			assert.Equal(t, []string{sub.QualifiedChannel}, payload[0].Payload, "V2 payload should match the recovery key")
		}
	}
	require.Positive(t, orderbooks, "futures defaults must include V2 orderbooks")
	subs, err = e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	var accountRequests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		accountRequests.Add(1)
		if _, err := w.Write([]byte(`{"user":20011}`)); err != nil {
			t.Errorf("Mock futures account response should be written: %v", err)
		}
	}))
	require.NoError(t, e.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for endpoint := range e.API.Endpoints.GetURLMap() {
		require.NoError(t, e.API.Endpoints.SetRunningURL(endpoint, server.URL+"/"), "SetRunningURL must not error")
	}
	e.API.AuthenticatedSupport = true
	e.API.AuthenticatedWebsocketSupport = true
	e.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	e.setFuturesUserID("key", currency.USDT, "20011")
	continueBootstrap, err := e.Bootstrap(t.Context())
	require.NoError(t, err, "Bootstrap must not error")
	assert.True(t, continueBootstrap, "Bootstrap should continue with common startup actions")
	assert.Equal(t, int64(1), accountRequests.Load(), "Bootstrap should request the uncached futures account ID")
	subs, err = e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.Equal(t, int64(1), accountRequests.Load(), "Subscription generation should not perform another account request")
	require.NoError(t, e.Websocket.Disable(), "Websocket.Disable must not error")
	e.SetCredentials(&accounts.Credentials{Key: "rotated-key", Secret: "secret"})
	continueBootstrap, err = e.Bootstrap(t.Context())
	require.NoError(t, err, "Bootstrap must not error")
	assert.True(t, continueBootstrap, "Bootstrap should continue with common startup actions")
	assert.Equal(t, int64(1), accountRequests.Load(), "Bootstrap should not request futures account IDs when websocket support is disabled")
	authenticatedChannels := map[string]bool{
		futuresOrdersChannel:            false,
		futuresUserTradesChannel:        false,
		futuresLiquidatesChannel:        false,
		futuresAutoDeleveragesChannel:   false,
		futuresBalancesChannel:          false,
		futuresReduceRiskLimitsChannel:  false,
		futuresPositionsChannel:         false,
		futuresAutoPositionCloseChannel: false,
		futuresAutoOrdersChannel:        false,
	}
	accountSubscriptions := make(map[string]*subscription.Subscription)
	for _, sub := range subs {
		if _, ok := authenticatedChannels[sub.Channel]; ok {
			authenticatedChannels[sub.Channel] = true
			accountSubscriptions[sub.Channel] = sub
		}
	}
	for channel, found := range authenticatedChannels {
		assert.Truef(t, found, "%s should be generated for authenticated futures", channel)
	}
	require.Len(t, accountSubscriptions, len(authenticatedChannels),
		"authenticated futures channels must each have one account-wide subscription")
	positionsSubscription := accountSubscriptions[futuresPositionsChannel]
	require.NotNil(t, positionsSubscription, "positions subscription must be generated")
	require.Len(t, positionsSubscription.Pairs, 1, "account-wide subscription must retain one identity pair")
	assert.NotEqual(t, allFuturesContracts, positionsSubscription.Pairs[0].String(),
		"account-wide subscription should not register the selector as a currency")
	assert.Equal(t, allFuturesContracts, positionsSubscription.Params[contractPayloadOverrideParam],
		"account-wide subscription should serialise the documented selector")
	requiresUserPlaceholder, ok := positionsSubscription.Params[requiresUserPlaceholderParam].(bool)
	require.True(t, ok, "user placeholder parameter must be a boolean")
	assert.True(t, requiresUserPlaceholder,
		"positions subscription should retain the deprecated user placeholder")
	require.Equal(t, asset.CoinMarginedFutures, positionsSubscription.Asset,
		"account-wide subscription must retain its futures asset")
	closeSubscription := accountSubscriptions[futuresAutoPositionCloseChannel]
	require.NotNil(t, closeSubscription, "position closes subscription must be generated")
	assert.Equal(t, "20011", closeSubscription.Params["user"],
		"position closes subscription should use the futures account user ID")
	accountSubscriptionList := make(subscription.List, 0, len(accountSubscriptions))
	for _, sub := range accountSubscriptions {
		accountSubscriptionList = append(accountSubscriptionList, sub)
	}
	payloads, err := e.generateFuturesPayload(t.Context(), subscribeEvent, accountSubscriptionList)
	require.NoError(t, err, "generateFuturesPayload must generate account-wide payloads")
	require.Len(t, payloads, len(authenticatedChannels),
		"account-wide subscriptions must each generate one payload")
	expectedPayloads := map[string][]string{
		futuresOrdersChannel:            {"20011", allFuturesContracts},
		futuresUserTradesChannel:        {"20011", allFuturesContracts},
		futuresLiquidatesChannel:        {"20011", allFuturesContracts},
		futuresAutoDeleveragesChannel:   {"20011", allFuturesContracts},
		futuresAutoPositionCloseChannel: {"20011", allFuturesContracts},
		futuresBalancesChannel:          {"20011"},
		futuresReduceRiskLimitsChannel:  {"20011", allFuturesContracts},
		futuresPositionsChannel:         {"", allFuturesContracts},
		futuresAutoOrdersChannel:        {allFuturesContracts},
	}
	for i := range payloads {
		assert.Equalf(t, expectedPayloads[payloads[i].Channel], payloads[i].Payload,
			"%s payload should match Gate's documented parameters", payloads[i].Channel)
		require.NotNilf(t, payloads[i].Auth, "%s payload must be authenticated", payloads[i].Channel)
	}
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, nil, true),
		"StorePairs must clear enabled coin-margined futures pairs")
	subs, err = e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err, "Enabled asset without pairs must not tear down other Gate connections")
	require.Empty(t, subs, "Enabled asset without pairs must return no subscriptions")
	require.NoError(t, e.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "Disabled asset must not error")
	require.Empty(t, subs, "Disabled asset must return no pairs")
}

func TestGenerateFuturesDefaultSubscriptionsAccountIDCache(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"user":20011}`))
		assert.NoError(t, err, "Mock futures account response should be written")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for endpoint := range ex.API.Endpoints.GetURLMap() {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint, server.URL+"/"), "SetRunningURL must not error")
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	err := ex.cacheFuturesUserID(t.Context(), "key", currency.USDT)
	require.NoError(t, err, "Initial account lookup must populate the futures user ID cache")
	subs, err := ex.GenerateFuturesDefaultSubscriptions(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "Subscription generation must reuse the matching cached user ID")
	privateChannels := map[string]bool{
		futuresOrdersChannel:            false,
		futuresUserTradesChannel:        false,
		futuresLiquidatesChannel:        false,
		futuresAutoDeleveragesChannel:   false,
		futuresAutoPositionCloseChannel: false,
		futuresBalancesChannel:          false,
		futuresReduceRiskLimitsChannel:  false,
		futuresPositionsChannel:         false,
		futuresAutoOrdersChannel:        false,
	}
	for _, sub := range subs {
		if _, ok := privateChannels[sub.Channel]; ok {
			privateChannels[sub.Channel] = true
		}
	}
	for channel, generated := range privateChannels {
		assert.Truef(t, generated, "%s should remain generated from the cached account ID", channel)
	}

	_, err = ex.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err, "A user ID cached for another settlement account must not affect generation")
	ex.SetCredentials(&accounts.Credentials{Key: "rotated-key", Secret: "secret"})
	_, err = ex.GenerateFuturesDefaultSubscriptions(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "A user ID cached for another API key must not affect generation")
}

func TestGenerateFuturesDefaultSubscriptionsColdCache(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	var accountRequests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		accountRequests.Add(1)
		_, err := w.Write([]byte(`{"user":20011}`))
		assert.NoError(t, err, "Mock futures account response should be written")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for endpoint := range ex.API.Endpoints.GetURLMap() {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint, server.URL+"/"), "SetRunningURL must not error")
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	subs, err := ex.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err, "Missing cached account ID must not prevent futures subscription generation")
	assert.Zero(t, accountRequests.Load(), "subscription generation should not perform an account request")

	authenticatedChannels := map[string]bool{
		futuresOrdersChannel:            false,
		futuresUserTradesChannel:        false,
		futuresLiquidatesChannel:        false,
		futuresAutoDeleveragesChannel:   false,
		futuresAutoPositionCloseChannel: false,
		futuresBalancesChannel:          false,
		futuresReduceRiskLimitsChannel:  false,
		futuresPositionsChannel:         false,
		futuresAutoOrdersChannel:        false,
	}
	for _, sub := range subs {
		if _, ok := authenticatedChannels[sub.Channel]; ok {
			authenticatedChannels[sub.Channel] = true
		}
	}
	assert.True(t, authenticatedChannels[futuresPositionsChannel], "Positions subscription should not require a user ID")
	assert.True(t, authenticatedChannels[futuresAutoOrdersChannel], "Auto-orders subscription should not require a user ID")
	for channel, generated := range authenticatedChannels {
		if channel == futuresPositionsChannel || channel == futuresAutoOrdersChannel {
			continue
		}
		assert.Falsef(t, generated, "%s should be skipped without a valid user ID", channel)
	}
}

func TestGenerateFuturesDefaultSubscriptionsMissingCredentials(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	subs, err := ex.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err, "Missing credentials must degrade to public futures subscriptions")
	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticated websocket support should be disabled")
	for _, sub := range subs {
		assert.NotContains(t, []string{
			futuresOrdersChannel,
			futuresUserTradesChannel,
			futuresLiquidatesChannel,
			futuresAutoDeleveragesChannel,
			futuresAutoPositionCloseChannel,
			futuresBalancesChannel,
			futuresReduceRiskLimitsChannel,
			futuresPositionsChannel,
			futuresAutoOrdersChannel,
		}, sub.Channel, "private channel should not be generated without credentials")
	}
}

func TestPreConnectWiring(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")

	var accountRequests atomic.Int64
	var healthy atomic.Bool
	rest := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		accountRequests.Add(1)
		if !healthy.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err := w.Write([]byte(`{"user":20011}`))
		assert.NoError(t, err, "Mock futures account response should be written")
	}))
	require.NoError(t, ex.SetHTTPClient(rest.Client()), "SetHTTPClient must not error")
	for endpoint := range ex.API.Endpoints.GetURLMap() {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint, rest.URL+"/"), "SetRunningURL must not error")
	}

	// Refuses the upgrade, so Connect fails locally after preparation has run.
	ws := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	ws.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(ws.URL, "http")), "SetAllConnectionURLs must not error")

	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	require.Error(t, ex.Websocket.Connect(t.Context()), "Connect must fail against the rejecting mock")
	assert.Equal(t, int64(2), accountRequests.Load(), "Connect should run futures account preparation for both settlement accounts")
	assert.Empty(t, ex.getFuturesUserID("key", currency.USDT), "a failed lookup should leave the cache cold")

	healthy.Store(true)
	require.Error(t, ex.Websocket.Connect(t.Context()), "Connect must fail against the rejecting mock")
	assert.Equal(t, int64(4), accountRequests.Load(), "a later connection attempt should retry the failed lookup")
	assert.Equal(t, "20011", ex.getFuturesUserID("key", currency.USDT), "a successful retry should populate the cache")

	require.Error(t, ex.Websocket.Connect(t.Context()), "Connect must fail against the rejecting mock")
	assert.Equal(t, int64(4), accountRequests.Load(), "a warm cache should not perform further account requests")
}

func TestPrepareFuturesUserIDsLookupFailure(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	var accountRequests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		accountRequests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for endpoint := range ex.API.Endpoints.GetURLMap() {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint, server.URL+"/"), "SetRunningURL must not error")
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	ex.prepareFuturesUserIDs(t.Context())

	assert.Equal(t, int64(2), accountRequests.Load(), "preparation should try each enabled futures settlement account")
	assert.Empty(t, ex.getFuturesUserID("key", currency.USDT), "failed preparation should leave the USDT account ID uncached")
	assert.Empty(t, ex.getFuturesUserID("key", currency.BTC), "failed preparation should leave the BTC account ID uncached")
}

func TestProcessFuturesOrdersPushData(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		incoming string
		status   order.Status
	}{
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"open","stp_id":"123456","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Open},
	}
	processed, err := e.processFuturesOrdersPushData([]byte(testCases[0].incoming), asset.CoinMarginedFutures)
	require.NoError(t, err)
	require.Len(t, processed, 1, "futures order update must contain one order")
	assert.Equal(t, testCases[0].status, processed[0].Status, "futures order status should be open")

	incomingWithoutTIF := strings.Replace(testCases[0].incoming, `,"tif":"gtc"`, "", 1)
	processed, err = e.processFuturesOrdersPushData([]byte(incomingWithoutTIF), asset.CoinMarginedFutures)
	require.NoError(t, err, "futures order update without TIF must use Gate's default")
	require.Len(t, processed, 1, "futures order update without TIF must contain one order")
	assert.Equal(t, order.GoodTillCancel, processed[0].TimeInForce, "futures order without TIF should default to GTC")
}

func TestProcessFinishedFuturesOrdersPushData(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		incoming string
		status   order.Status
	}{
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"filled","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Filled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"cancelled","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"liquidated","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Liquidated},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"ioc","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"auto_deleveraged","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.AutoDeleverage},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"reduce_only","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"position_closed","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Closed},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"stp","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.STP},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			processed, err := e.processFuturesOrdersPushData([]byte(tc.incoming), asset.CoinMarginedFutures)
			require.NoError(t, err)
			require.Len(t, processed, 1, "futures order update must contain one order")
			for i := range processed {
				assert.Equal(t, tc.status.String(), processed[i].Status.String())
				assert.Equal(t, order.Long, processed[i].Side, "positive futures order size should map to long")
				assert.Equal(t, 1.0, processed[i].Amount, "futures order amount should be absolute")
				assert.Equal(t, order.Limit, processed[i].Type, "priced GTC futures order should be a limit order")
				assert.Equal(t, order.GoodTillCancel, processed[i].TimeInForce, "GTC futures order should preserve time in force")
				assert.Equal(t, 40000.4, processed[i].AverageExecutedPrice, "futures order should preserve its fill price")
			}
		})
	}

	t.Run("reduce out among other orders", func(t *testing.T) {
		t.Parallel()
		order1 := `{"contract":"BTC_USD","create_time":1628736847,"fill_price":0,"finish_as":"reduce_out","finish_time":1628736848,"id":4872461,"is_reduce_only":true,"left":2,"price":40000.4,"size":-2,"status":"finished","text":"-","tif":"gtc","user":"110xxxxx"}`
		order2 := `{"contract":"BTC_USD","create_time":1628736847,"fill_price":0,"finish_as":"","id":4872462,"left":1,"price":39000,"size":1,"status":"open","text":"-","tif":"gtc","user":"110xxxxx"}`
		incoming := `{"channel":"futures.orders","event":"update","time":1541505434,"result":[` + order1 + `,` + order2 + `]}`
		processed, err := e.processFuturesOrdersPushData([]byte(incoming), asset.CoinMarginedFutures)
		require.NoError(t, err, "processFuturesOrdersPushData must accept an order finished as reduce_out")
		require.Len(t, processed, 2, "processFuturesOrdersPushData must keep every order of the push")
		assert.Equal(t, order.Cancelled, processed[0].Status, "an order finished as reduce_out should be cancelled")
		assert.Equal(t, order.Open, processed[1].Status, "an open order should stay open")
	})

	t.Run("finish_as spellings Gate's futures websocket page lists", func(t *testing.T) {
		t.Parallel()
		order1 := `{"contract":"BTC_USD","create_time":1628736847,"finish_as":"position_close","finish_time":1628736848,"id":4872463,"left":2,"price":40000.4,"size":-2,"status":"finished","text":"-","tif":"gtc","user":"110xxxxx"}`
		order2 := `{"contract":"BTC_USD","create_time":1628736847,"finish_as":"auto_deleveraging","finish_time":1628736848,"id":4872464,"left":0,"price":40000.4,"size":3,"status":"finished","text":"auto_deleveraging","tif":"gtc","user":"110xxxxx"}`
		order3 := `{"contract":"BTC_USD","create_time":1628736847,"finish_as":"filled","finish_time":1628736848,"id":4872465,"left":0,"price":39000,"size":1,"status":"finished","text":"-","tif":"gtc","user":"110xxxxx"}`
		incoming := `{"channel":"futures.orders","event":"update","time":1541505434,"result":[` + order1 + `,` + order2 + `,` + order3 + `]}`
		processed, err := e.processFuturesOrdersPushData([]byte(incoming), asset.CoinMarginedFutures)
		require.NoError(t, err, "processFuturesOrdersPushData must accept both spellings Gate documents for each finish_as value")
		require.Len(t, processed, 3, "processFuturesOrdersPushData must keep every order of the push")
		assert.Equal(t, order.Closed, processed[0].Status, "an order finished as position_close should be closed")
		assert.Equal(t, order.AutoDeleverage, processed[1].Status, "an order finished as auto_deleveraging should be auto-deleveraged")
		assert.Equal(t, order.Filled, processed[2].Status, "a filled order in the same push should be filled")
	})

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		_, err := e.processFuturesOrdersPushData([]byte(`{`), asset.CoinMarginedFutures)
		assert.Error(t, err, "processFuturesOrdersPushData should reject malformed JSON")
	})

	t.Run("unknown time in force", func(t *testing.T) {
		t.Parallel()
		incoming := strings.Replace(testCases[0].incoming, `"tif":"gtc"`, `"tif":"gtd"`, 1)
		_, err := e.processFuturesOrdersPushData([]byte(incoming), asset.CoinMarginedFutures)
		assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "processFuturesOrdersPushData should reject an unknown time in force")
	})

	t.Run("finished without finish_as", func(t *testing.T) {
		t.Parallel()
		incoming := strings.Replace(testCases[0].incoming, `"finish_as":"filled"`, `"finish_as":""`, 1)
		_, err := e.processFuturesOrdersPushData([]byte(incoming), asset.CoinMarginedFutures)
		assert.ErrorContains(t, err, `"FINISHED"`, "processFuturesOrdersPushData should not guess how a finished order finished")
	})

	t.Run("negative size", func(t *testing.T) {
		t.Parallel()
		incoming := strings.Replace(testCases[0].incoming, `"size":1`, `"size":-1`, 1)
		processed, err := e.processFuturesOrdersPushData([]byte(incoming), asset.CoinMarginedFutures)
		require.NoError(t, err)
		require.Len(t, processed, 1, "futures order update must contain one order")
		assert.Equal(t, order.Short, processed[0].Side, "negative futures order size should map to short")
		assert.Equal(t, 1.0, processed[0].Amount, "futures order amount should be absolute")
		assert.Equal(t, 1.0, processed[0].ExecutedAmount, "executed futures order amount should be absolute")
	})

	t.Run("market order", func(t *testing.T) {
		t.Parallel()
		incoming := strings.NewReplacer(`"price":40000.4`, `"price":0`, `"tif":"gtc"`, `"tif":"ioc"`).Replace(testCases[1].incoming)
		processed, err := e.processFuturesOrdersPushData([]byte(incoming), asset.USDTMarginedFutures)
		require.NoError(t, err)
		require.Len(t, processed, 1, "futures order update must contain one order")
		assert.Equal(t, order.Market, processed[0].Type, "zero-price IOC futures order should be a market order")
		assert.Equal(t, order.ImmediateOrCancel, processed[0].TimeInForce, "IOC futures order should preserve time in force")
		assert.Zero(t, processed[0].Price, "market order request price should remain zero")
		assert.Equal(t, 40000.4, processed[0].AverageExecutedPrice, "market order should preserve its fill price")
	})
}

// Captured from live GateIO websocket data on 2026-09-03.
// Treat these payloads as semi-trusted until independently confirmed.
func TestFuturesLiveCapturedPayloads(t *testing.T) {
	t.Parallel()

	t.Run("orders", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			name             string
			payload          string
			direction        order.Side
			amount           float64
			price            float64
			averageFillPrice float64
			timeInForce      order.TimeInForce
			orderType        order.Type
			reduceOnly       bool
		}{
			{
				name:      "reduce-only IOC close",
				payload:   `{"time":1788387249,"time_ms":1788387249995,"channel":"futures.orders","event":"update","result":[{"create_time":1788387249,"create_time_ms":1788387249995,"fill_price":0.242073333333,"finish_as":"filled","iceberg":"0","id":294985777215626819,"id_string":"294985777215626819","is_close":false,"is_liq":false,"left":"0","mkfr":0.0002,"is_reduce_only":true,"status":"finished","tkfr":0.00048,"price":0,"refu":0,"refr":0,"size":"21","text":"apiv4-ws","tif":"ioc","finish_time":1788387249,"finish_time_ms":1788387249995,"user":"12870774","contract":"RAVE_USDT","stop_profit_price":"","stop_loss_price":"","amend_text":"-","biz_info":"-","stp_act":"-","stp_id":"0","update_id":1,"is_voucher":false,"update_time":1788387249995,"fee":0.024400992,"point_fee":0,"role":"taker","bbo":"-","market_order_slip_ratio":"0.025","pos_margin_mode":"isolated","leverage":"1"}]}`,
				direction: order.Long, orderType: order.Market, amount: 21, averageFillPrice: 0.242073333333,
				timeInForce: order.ImmediateOrCancel, reduceOnly: true,
			},
			{
				name:      "short FOK open",
				payload:   `{"time":1788387300,"time_ms":1788387300728,"channel":"futures.orders","event":"update","result":[{"create_time":1788387300,"create_time_ms":1788387300727,"fill_price":60.97,"finish_as":"filled","iceberg":"0","id":296956100605608003,"id_string":"296956100605608003","is_close":false,"is_liq":false,"left":"0","mkfr":0.0002,"is_reduce_only":false,"status":"finished","tkfr":0.00048,"price":60.91,"refu":0,"refr":0,"size":"-3","text":"apiv4-ws","tif":"fok","finish_time":1788387300,"finish_time_ms":1788387300727,"user":"12870774","contract":"SLVON_USDT","stop_profit_price":"","stop_loss_price":"","amend_text":"-","biz_info":"-","stp_act":"-","stp_id":"0","update_id":1,"is_voucher":false,"update_time":1788387300727,"fee":0.00877968,"point_fee":0,"role":"taker","bbo":"-","market_order_slip_ratio":"0","pos_margin_mode":"isolated","leverage":"1"}]}`,
				direction: order.Short, orderType: order.Limit, amount: 3, price: 60.91, averageFillPrice: 60.97,
				timeInForce: order.FillOrKill,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				processed, err := e.processFuturesOrdersPushData([]byte(tt.payload), asset.USDTMarginedFutures)
				require.NoError(t, err, "processFuturesOrdersPushData must process captured live data")
				require.Len(t, processed, 1, "captured order payload must contain one order")
				assert.Equal(t, order.Filled, processed[0].Status, "captured order status should be filled")
				assert.Equal(t, tt.orderType, processed[0].Type, "captured order type should follow the request price")
				assert.Equal(t, tt.direction, processed[0].Side, "captured order direction should follow signed size")
				assert.Equal(t, tt.amount, processed[0].Amount, "captured order amount should be absolute")
				assert.Equal(t, tt.amount, processed[0].ExecutedAmount, "captured order should be fully executed")
				assert.Zero(t, processed[0].RemainingAmount, "captured order should have no remaining amount")
				assert.Equal(t, tt.price, processed[0].Price, "captured order price should be preserved")
				assert.Equal(t, tt.averageFillPrice, processed[0].AverageExecutedPrice, "captured fill price should be preserved")
				assert.Equal(t, tt.timeInForce, processed[0].TimeInForce, "captured time in force should be preserved")
				assert.Equal(t, tt.reduceOnly, processed[0].ReduceOnly, "captured reduce-only flag should be preserved")
			})
		}
	})

	t.Run("user trade", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			name      string
			payload   string
			direction order.Side
			amount    float64
			tradeID   string
			orderID   string
			timestamp time.Time
		}{
			{
				name: "long", direction: order.Long, amount: 13.4,
				tradeID: "27640330", orderID: "294985777215626819", timestamp: time.UnixMilli(1788387249995),
				payload: `{"time":1788387249,"time_ms":1788387249995,"channel":"futures.usertrades","event":"update","result":[{"id":"27640330","order_id":"294985777215626819","contract":"RAVE_USDT","create_time":1788387249,"create_time_ms":1788387249995,"size":"13.4","role":"taker","price":"0.242","text":"apiv4-ws","fee":0.01556544,"point_fee":0,"amend_text":"-","biz_info":"-","close_size":"13.4"}]}`,
			},
			{
				name: "short", direction: order.Short, amount: 6,
				tradeID: "1762690", orderID: "296956100605606387", timestamp: time.UnixMilli(1788387276248),
				payload: `{"time":1788387276,"time_ms":1788387276249,"channel":"futures.usertrades","event":"update","result":[{"id":"1762690","order_id":"296956100605606387","contract":"SLVON_USDT","create_time":1788387276,"create_time_ms":1788387276248,"size":"-6","role":"taker","price":"61.13","text":"apiv4-ws","fee":0.01760544,"point_fee":0,"amend_text":"-","biz_info":"-","close_size":"0"}]}`,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
				ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
				require.NoError(t, ex.processFuturesUserTrades([]byte(tt.payload), asset.USDTMarginedFutures),
					"processFuturesUserTrades must process captured live data")
				message := <-ex.Websocket.DataHandler.C
				fills, ok := message.Data.([]fill.Data)
				require.True(t, ok, "captured user trade must emit fills")
				require.Len(t, fills, 1, "captured user trade must contain one fill")
				assert.Equal(t, tt.direction, fills[0].Side, "captured trade direction should follow signed size")
				assert.Equal(t, tt.amount, fills[0].Amount, "captured trade amount should be absolute")
				assert.Equal(t, tt.tradeID, fills[0].TradeID, "captured trade ID should be preserved")
				assert.Equal(t, tt.orderID, fills[0].OrderID, "captured order ID should be preserved")
				assert.Equal(t, tt.timestamp, fills[0].Timestamp, "captured trade timestamp should preserve millisecond precision")
			})
		}
	})

	t.Run("position", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures,
			currency.Pairs{currency.NewPair(currency.NewCode("RAVE"), currency.USDT)}, false),
			"StorePairs must register the captured contract")
		payload := []byte(`{"time":1788387249,"time_ms":1788387249995,"channel":"futures.positions","event":"update","result":[{"time":1788387249,"time_ms":1788387249995,"size":"-37","user":"12870774","leverage_max":25,"mode":"single","update_id":4815,"cross_leverage_limit":20,"liq_price":0.4942,"maintenance_rate":0.8,"risk_limit":10000000,"last_close_pnl":0.00672032,"entry_price":0.252077758621,"leverage":1,"realised_pnl":2.0063498704,"contract":"RAVE_USDT","history_point":0,"margin":93.40867384569,"realised_point":0,"history_pnl":-55.48692517441,"pos_margin_mode":"isolated","lever":"1"}]}`)
		require.NoError(t, ex.processFuturesPositionsNotification(t.Context(), payload, asset.USDTMarginedFutures),
			"processFuturesPositionsNotification must process captured live data")
		message := <-ex.Websocket.DataHandler.C
		positions, ok := message.Data.([]futures.Position)
		require.True(t, ok, "captured position must emit canonical positions")
		require.Len(t, positions, 1, "captured position payload must contain one position")
		assert.Equal(t, order.Short, positions[0].LatestDirection, "negative captured position size should map to short")
		assert.Equal(t, "37", positions[0].LatestSize.String(), "captured position size should be absolute")
		assert.Equal(t, int64(4815), positions[0].UpdateID, "captured position update ID should be preserved")
		assert.True(t, positions[0].RealisedPNL.Equal(decimal.MustFromFloat(2.0063498704)), "captured realised PNL should be preserved")
	})
}
