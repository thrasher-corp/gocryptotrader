package gateio

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
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
