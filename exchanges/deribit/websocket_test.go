package deribit

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// websocketVenueRequest is a JSON-RPC request as Deribit's websocket receives it
type websocketVenueRequest struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func TestSendWebsocketRequest(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	var result types.Time
	require.NoError(t, ex.SendWebsocketRequest(t.Context(), nonMatchingEPL, "public/get_time", nil, &result), "SendWebsocketRequest must not error")
	assert.NotZero(t, result, "SendWebsocketRequest should decode the result")
	if mockTests {
		requests := venue.received("public/get_time")
		require.Len(t, requests, 1, "SendWebsocketRequest must send one request")
		assert.NotZero(t, requests[0].ID, "SendWebsocketRequest should send a request ID to match the response with")
	}

	err := ex.SendWebsocketRequest(t.Context(), nonMatchingEPL, "public/test", map[string]string{"expected_result": testExpectedResultException}, nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SendWebsocketRequest must return the error object Deribit sends")
	assert.Equal(t, int64(11094), apiErr.Code, "SendWebsocketRequest should decode the error code")

	unauthenticated := newTestExchange(t)
	unauthenticated.Websocket.SetCanUseAuthenticatedEndpoints(false)
	err = unauthenticated.SendWebsocketRequest(t.Context(), nonMatchingEPL, "private/get_positions", nil, nil)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "SendWebsocketRequest should refuse a private method on an unauthenticated connection")

	err = newTestExchange(t).SendWebsocketRequest(t.Context(), nonMatchingEPL, "public/get_time", nil, &result)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "SendWebsocketRequest should error without a connection")
}

// unacknowledgedChannel marks a channel the mock websocket does not acknowledge subscribing or unsubscribing
const unacknowledgedChannel = "unacknowledged"

// relayed returns the next payload an exchange's data handler holds, failing the test when it holds none
func relayed(t *testing.T, ex *Exchange) any {
	t.Helper()
	select {
	case msg := <-ex.Websocket.DataHandler.C:
		return msg.Data
	default:
		require.Fail(t, "the data handler must hold a payload")
		return nil
	}
}

// assertNothingRelayed checks an exchange's data handler holds no payload
func assertNothingRelayed(t *testing.T, ex *Exchange) {
	t.Helper()
	select {
	case msg := <-ex.Websocket.DataHandler.C:
		assert.Failf(t, "the data handler should hold no payload", "got %T: %v", msg.Data, msg.Data)
	default:
	}
}

// notification returns a subscription notification of a channel carrying data
func notification(channel, data string) []byte {
	return []byte(`{"jsonrpc":"2.0","method":"subscription","params":{"channel":"` + channel + `","data":` + data + `}}`)
}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	assert.True(t, ex.websocketDialled(), "WebsocketConnect should dial the websocket")
	if mockTests {
		heartbeats := venue.received("public/set_heartbeat")
		require.NotEmpty(t, heartbeats, "WebsocketConnect must set the heartbeat")
		assert.JSONEq(t, `{"interval":15}`, string(heartbeats[0].Params), "WebsocketConnect should ask for a heartbeat every 15 seconds")
		assert.Len(t, venue.received("public/auth"), 1, "WebsocketConnect should authenticate a connection with credentials")
	}

	disabled := newTestExchange(t)
	require.NoError(t, disabled.Websocket.Disable(), "Disable must not error")
	assert.ErrorIs(t, disabled.WebsocketConnect(), websocket.ErrWebsocketNotEnabled, "WebsocketConnect should refuse a disabled websocket")

	if !mockTests {
		return
	}
	_, unauthenticatedVenue := newUnauthenticatedWebsocketTestExchange(t)
	assert.Empty(t, unauthenticatedVenue.received("public/auth"), "WebsocketConnect should not authenticate without authenticated websocket support")

	// Connecting without the manager leaves no connection monitor to redial, and waiting for each connection's reader to
	// stop keeps it from marking the next connection closed
	url, reloginVenue := startWebsocketVenue(t)
	relogin := newTestExchange(t)
	require.NoError(t, relogin.Websocket.SetWebsocketURL(url, false, false), "SetWebsocketURL must not error")
	relogin.API.AuthenticatedWebsocketSupport = true
	for _, tc := range []struct {
		clientID      string
		authenticated bool
	}{
		{testClientID, true},
		{"other-client-id", false},
		{testClientID, true},
	} {
		relogin.SetCredentials(&accounts.Credentials{Key: tc.clientID, Secret: testClientSecret})
		require.NoError(t, relogin.WebsocketConnect(), "WebsocketConnect must connect, whether or not its login succeeds")
		assert.Equalf(t, tc.authenticated, relogin.Websocket.CanUseAuthenticatedEndpoints(), "WebsocketConnect should leave the connection of client ID %s authenticated only when its login succeeds", tc.clientID)
		require.NoError(t, relogin.Websocket.Conn.Shutdown(), "Shutdown must close the connection")
		relogin.Websocket.Wg.Wait()
	}
	assert.Len(t, reloginVenue.received("public/auth"), 3, "WebsocketConnect should authenticate every connection, a connection after a failed login included")
}

func TestWebsocketLogin(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, venue := newWebsocketTestExchange(t)
	require.NoError(t, ex.wsLogin(t.Context()), "wsLogin must not error")
	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "wsLogin should allow authenticated requests")
	if mockTests {
		requests := venue.received("public/auth")
		require.Len(t, requests, 2, "the connection and wsLogin must each authenticate")
		var params map[string]any
		require.NoError(t, json.Unmarshal(requests[1].Params, &params), "the authentication params must decode")
		assert.Equal(t, GrantTypeClientSignature, params["grant_type"], "wsLogin should authenticate with a client signature")
		assert.Equal(t, testClientID, params["client_id"], "wsLogin should send the client ID")
		assert.NotContains(t, params, "client_secret", "wsLogin should not send the client secret")
	}

	unsupported := newTestExchange(t)
	unsupported.API.AuthenticatedWebsocketSupport = false
	assert.ErrorIs(t, unsupported.wsLogin(t.Context()), exchange.ErrAuthenticationSupportNotEnabled, "wsLogin should need websocket authentication support")

	if !mockTests {
		return
	}
	// The mock websocket authenticates only the test client ID's signature
	rejected, _ := newWebsocketTestExchange(t)
	rejected.SetCredentials(&accounts.Credentials{Key: "other-client-id", Secret: testClientSecret})
	var apiErr *APIError
	require.ErrorAs(t, rejected.wsLogin(t.Context()), &apiErr, "wsLogin must return the authentication failure")
	assert.Equal(t, int64(13004), apiErr.Code, "wsLogin should return Deribit's invalid credentials error")
	assert.False(t, rejected.Websocket.CanUseAuthenticatedEndpoints(), "wsLogin should stop authenticated requests after a failure")
}

func TestWebsocketDialled(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	assert.True(t, ex.websocketDialled(), "websocketDialled should report a dialled connection")
	assert.False(t, newTestExchange(t).websocketDialled(), "websocketDialled should not report an undialled connection")
}

// readerConnection serves queued messages to the websocket reader, then the empty message a closed connection reads
type readerConnection struct {
	websocket.Connection
	messages []websocket.Response
}

func (r *readerConnection) ReadMessage() websocket.Response {
	if len(r.messages) == 0 {
		return websocket.Response{}
	}
	message := r.messages[0]
	r.messages = r.messages[1:]
	return message
}

func (r *readerConnection) GetURL() string {
	return "reader"
}

func TestWebsocketReadData(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.Websocket.Conn = &readerConnection{messages: []websocket.Response{
		{Raw: notification(ChannelPriceIndex+".btc_usd", `{"timestamp":1792107433668,"price":82119.87,"index_name":"btc_usd"}`)},
		{Raw: []byte(`not json`)},
	}}
	ex.Websocket.Wg.Add(1)
	ex.wsReadData(t.Context())
	assert.IsType(t, &WebsocketPriceIndex{}, relayed(t, ex), "wsReadData should pass a message to its handler")
	err, ok := relayed(t, ex).(error)
	require.True(t, ok, "wsReadData must relay a handling error")
	assert.ErrorContains(t, err, "could not parse websocket data", "wsReadData should relay the handling error")
	assertNothingRelayed(t, ex)
}

func TestWebsocketHandleData(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	assert.ErrorContains(t, ex.wsHandleData(t.Context(), []byte(`{`)), "could not parse websocket data", "wsHandleData should reject malformed JSON")
	assert.Error(t, ex.wsHandleData(t.Context(), []byte(`{"jsonrpc":"2.0","id":987654321,"result":"ok"}`)), "wsHandleData should error for a response no request awaits")

	require.NoError(t, ex.wsHandleData(t.Context(), []byte(`{"jsonrpc":"2.0","method":"heartbeat","params":{"type":"heartbeat"}}`)), "wsHandleData must not error for a heartbeat")
	assertNothingRelayed(t, ex)

	require.NoError(t, ex.wsHandleData(t.Context(), notification(ChannelPriceIndex+".btc_usd", `{"timestamp":1792107433668,"price":82119.87,"index_name":"btc_usd"}`)), "wsHandleData must not error for a notification")
	assert.IsType(t, &WebsocketPriceIndex{}, relayed(t, ex), "wsHandleData should pass a notification to its channel's handler")

	require.NoError(t, ex.wsHandleData(t.Context(), []byte(`{"jsonrpc":"2.0","method":"bogus"}`)), "wsHandleData must not error for an unknown method")
	assert.IsType(t, websocket.UnhandledMessageWarning{}, relayed(t, ex), "wsHandleData should relay an unknown method as unhandled")

	if !mockTests {
		return
	}
	connected, venue := newWebsocketTestExchange(t)
	require.NoError(t, connected.wsHandleData(t.Context(), []byte(`{"jsonrpc":"2.0","method":"heartbeat","params":{"type":"test_request"}}`)), "wsHandleData must not error for a test request")
	assert.Eventually(t, func() bool {
		return len(venue.received("public/test")) == 1
	}, 5*time.Second, 10*time.Millisecond, "wsHandleData should answer a test request with a public/test request")
}

func TestWebsocketAnswerHeartbeat(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	ex.wsAnswerHeartbeat(t.Context())
	if mockTests {
		requests := venue.received("public/test")
		require.Len(t, requests, 1, "wsAnswerHeartbeat must send a public/test request")
		assert.Empty(t, requests[0].Params, "wsAnswerHeartbeat should send no params")
	}
	// Without a connection the failure is logged, as nothing awaits the answer
	newTestExchange(t).wsAnswerHeartbeat(t.Context())
}

func TestWebsocketHandleNotification(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	// Each payload is trimmed from a live notification, except the private channels', which follow Deribit's documented
	// examples, and the rare announcements, instrument.creation and block_rfq.trades, which carry the data of their
	// public method recorded live
	for _, tc := range []struct {
		channel string
		data    string
		exp     any
	}{
		{
			channel: "perpetual.BTC-PERPETUAL.100ms",
			data:    `{"index_price":82119.87,"interest":3.70194509563518e-05,"timestamp":1792107433899}`,
			exp:     &WebsocketPerpetualInterest{Timestamp: types.Time(time.UnixMilli(1792107433899)), IndexPrice: 82119.87, Interest: 3.70194509563518e-05},
		},
		{
			channel: "deribit_price_index.btc_usd",
			data:    `{"timestamp":1792107433668,"price":82119.87,"index_name":"btc_usd"}`,
			exp:     &WebsocketPriceIndex{Timestamp: types.Time(time.UnixMilli(1792107433668)), IndexName: "btc_usd", Price: 82119.87},
		},
		{
			channel: "deribit_price_index.sol_usdc",
			data:    `{"timestamp":1792483554699,"price":103.8935,"index_name":"sol_usdc","mda_source":"primary","mda_version":1792160739946}`,
			exp:     &WebsocketPriceIndex{Timestamp: types.Time(time.UnixMilli(1792483554699)), IndexName: "sol_usdc", Price: 103.8935, MDASource: "primary", MDAVersion: 1792160739946},
		},
		{
			channel: "deribit_price_ranking.btc_usd",
			data:    `[{"enabled":false,"timestamp":1792107431589,"identifier":"binance","price":82134.55,"weight":0.0,"original_price":82134.55},{"enabled":true,"timestamp":1792107433000,"identifier":"coinbase","price":82120.08,"weight":20.0,"original_price":82120.08}]`,
			exp: &[]WebsocketPriceRanking{
				{Identifier: "binance", Timestamp: types.Time(time.UnixMilli(1792107431589)), OriginalPrice: 82134.55, Price: 82134.55},
				{Identifier: "coinbase", Enabled: true, Timestamp: types.Time(time.UnixMilli(1792107433000)), OriginalPrice: 82120.08, Price: 82120.08, Weight: 20},
			},
		},
		{
			channel: "deribit_price_statistics.btc_usd",
			data:    `{"index_name":"btc_usd","fast_market":false,"change24h":636.83,"high24h":82540.38,"low24h":79699.63}`,
			exp:     &WebsocketPriceStatistics{IndexName: "btc_usd", Low24Hour: 79699.63, High24Hour: 82540.38, Change24Hour: 636.83},
		},
		{
			channel: "deribit_volatility_index.btc_usd",
			data:    `{"timestamp":1792107470023,"index_name":"btc_usd","volatility":38.76}`,
			exp:     &WebsocketVolatilityIndex{Timestamp: types.Time(time.UnixMilli(1792107470023)), IndexName: "btc_usd", Volatility: 38.76},
		},
		{
			channel: "markprice.options.btc_usd",
			data:    `[{"timestamp":1792107433944,"iv":0.3037,"instrument_name":"BTC-16OCT26-83000-P","mark_price":0.0113}]`,
			exp:     &[]WebsocketMarkPriceOption{{Timestamp: types.Time(time.UnixMilli(1792107433944)), InstrumentName: "BTC-16OCT26-83000-P", MarkPrice: 0.0113, ImpliedVolatility: 0.3037}},
		},
		{
			channel: "estimated_expiration_price.btc_usd",
			data:    `{"seconds":30164,"timestamp":1792107436016,"price":82119.57,"is_estimated":false}`,
			exp:     &WebsocketEstimatedExpirationPrice{Timestamp: types.Time(time.UnixMilli(1792107436016)), Seconds: 30164, Price: 82119.57},
		},
		{
			channel: "instrument.state.any.any",
			data:    `{"timestamp":1792107544433,"state":"inactive","instrument_name":"BTC-CCAL-18OCT26_16OCT26-83000"}`,
			exp:     &WebsocketInstrumentState{Timestamp: types.Time(time.UnixMilli(1792107544433)), InstrumentName: "BTC-CCAL-18OCT26_16OCT26-83000", State: "inactive"},
		},
		{
			channel: "platform_state",
			data:    `{"locked":false,"price_index":"arb_usdc","reasons":[]}`,
			exp:     &WebsocketPlatformState{PriceIndex: "arb_usdc", Reasons: json.RawMessage(`[]`)},
		},
		{
			channel: "platform_state.public_methods_state",
			data:    `{"allow_unauthenticated_public_requests":true}`,
			exp:     &WebsocketPublicMethodsState{AllowUnauthenticatedPublicRequests: true},
		},
		{
			channel: "user.lock",
			data:    `{"currency":"ALL","locked":true}`,
			exp:     &WebsocketUserLock{Currency: currency.NewCode("ALL"), Locked: true},
		},
		{
			channel: "user.liquidation",
			data:    `{"state":"liquidation_started","user_id":12345,"isolated":false,"currency":"BTC","margin_balance":0.5,"maintenance_margin":0.6,"initial_margin":0.8,"close_out_margin":0.3,"equity":0.5,"timestamp":1711468632693}`,
			exp:     &WebsocketUserLiquidation{State: "liquidation_started", UserID: 12345, Currency: currency.BTC, MarginBalance: 0.5, MaintenanceMargin: 0.6, InitialMargin: 0.8, CloseOutMargin: 0.3, Equity: 0.5, Timestamp: types.Time(time.UnixMilli(1711468632693))},
		},
		{
			channel: "user.isolated.liquidation",
			data:    `{"state":"adl_transfer","user_id":12346,"isolated":true,"currency":"ETH","instrument_name":"ETH-PERPETUAL","direction":"sell","amount":100,"price":2500.5,"commission":0.0001,"timestamp":1711468632694}`,
			exp:     &WebsocketUserLiquidation{State: "adl_transfer", UserID: 12346, Isolated: true, Currency: currency.ETH, InstrumentName: "ETH-PERPETUAL", Direction: DirectionSell, Amount: 100, Price: 2500.5, Commission: 0.0001, Timestamp: types.Time(time.UnixMilli(1711468632694))},
		},
		{
			channel: "user.mmp_trigger.eth_usdc",
			data:    `{"frozen_until":0,"index_name":"eth_usdc","mmp_group":"MassQuoteBot7"}`,
			exp:     &WebsocketMMPTrigger{IndexName: "eth_usdc", MMPGroup: "MassQuoteBot7"},
		},
		{
			channel: "announcements",
			data:    `{"action":"new","body":"Deribit will launch USDC-settled equity perpetuals","confirmation":false,"id":1791813857051,"important":true,"publication_timestamp":1791713204677,"title":"New Equity & ETF Perpetuals. Coming to Deribit."}`,
			exp: &WebsocketAnnouncement{
				ID:                   1791813857051,
				Title:                "New Equity & ETF Perpetuals. Coming to Deribit.",
				Body:                 "Deribit will launch USDC-settled equity perpetuals",
				Important:            true,
				PublicationTimestamp: types.Time(time.UnixMilli(1791713204677)),
				Action:               "new",
			},
		},
		{
			channel: "instrument.creation.any.any",
			data:    `{"base_currency":"BTC","block_trade_commission":0.0003,"block_trade_min_trade_amount":25,"block_trade_tick_size":0.0001,"contract_size":1,"counter_currency":"USD","creation_timestamp":1787817624000,"expiration_timestamp":1795766400000,"index_id":1000033,"instrument_id":683239,"instrument_name":"BTC-27NOV26-90000-C","instrument_type":"reversed","is_active":true,"kind":"option","lot_size":1,"maker_commission":0.0003,"min_trade_amount":0.1,"option_type":"call","price_index":"btc_usd","product_group":"BTC","quote_currency":"BTC","settlement_currency":"BTC","settlement_period":"month","state":"open","strike":90000,"taker_commission":0.0003,"tick_size":0.0001,"tick_size_steps":[{"above_price":0.005,"tick_size":0.0005}],"underlying_type":"crypto","timestamp":1787817624000}`,
			exp: &WebsocketInstrumentCreation{
				InstrumentName:           "BTC-27NOV26-90000-C",
				InstrumentID:             683239,
				Kind:                     KindOption,
				InstrumentType:           "reversed",
				OptionType:               "call",
				Strike:                   90000,
				State:                    "open",
				IsActive:                 true,
				BaseCurrency:             currency.BTC,
				QuoteCurrency:            currency.BTC,
				CounterCurrency:          currency.USD,
				SettlementCurrency:       currency.BTC,
				SettlementPeriod:         "month",
				PriceIndex:               "btc_usd",
				IndexID:                  1000033,
				ProductGroup:             "BTC",
				UnderlyingType:           "crypto",
				CreationTimestamp:        types.Time(time.UnixMilli(1787817624000)),
				ExpirationTimestamp:      types.Time(time.UnixMilli(1795766400000)),
				TickSize:                 0.0001,
				TickSizeSteps:            []TickSizeStep{{AbovePrice: 0.005, TickSize: 0.0005}},
				ContractSize:             1,
				MinTradeAmount:           0.1,
				LotSize:                  1,
				MakerCommission:          0.0003,
				TakerCommission:          0.0003,
				BlockTradeCommission:     0.0003,
				BlockTradeTickSize:       0.0001,
				BlockTradeMinTradeAmount: 25,
				Timestamp:                types.Time(time.UnixMilli(1787817624000)),
			},
		},
		{
			channel: "block_rfq.trades.btc",
			data:    `{"amount":50,"combo_id":"BTC-REV-27NOV26-82000","direction":"buy","hedge":{"amount":4100000,"direction":"sell","instrument_name":"BTC-PERPETUAL","price":82000},"id":57976,"index_prices":{"btc_usd":82060.01},"legs":[{"direction":"buy","instrument_name":"BTC-27NOV26-82000-C","price":0.054,"ratio":1},{"direction":"sell","instrument_name":"BTC-27NOV26-82000-P","price":0.0472,"ratio":1}],"mark_price":0.007482373,"timestamp":1792082270445,"trades":[{"amount":50,"direction":"buy","hedge_amount":4100000,"price":0.0068}]}`,
			exp: &BlockRFQTrade{
				ID:        57976,
				Timestamp: types.Time(time.UnixMilli(1792082270445)),
				Direction: DirectionBuy,
				Amount:    50,
				MarkPrice: 0.007482373,
				Legs: []BlockRFQLeg{
					{InstrumentName: "BTC-27NOV26-82000-C", Direction: DirectionBuy, Ratio: 1, Price: 0.054},
					{InstrumentName: "BTC-27NOV26-82000-P", Direction: DirectionSell, Ratio: 1, Price: 0.0472},
				},
				ComboID:     "BTC-REV-27NOV26-82000",
				Hedge:       &BlockRFQHedge{InstrumentName: "BTC-PERPETUAL", Direction: DirectionSell, Price: 82000, Amount: 4100000},
				IndexPrices: map[string]float64{"btc_usd": 82060.01},
				Trades:      []BlockRFQFill{{Direction: DirectionBuy, Price: 0.0068, Amount: 50, HedgeAmount: 4100000}},
			},
		},
		{
			channel: "user.access_log",
			data:    `{"timestamp":1632488963633,"log":"success","ip":"8.9.10.11","id":243343,"country":"China","city":"Pekin"}`,
			exp:     &AccessLogEntry{ID: 243343, Timestamp: types.Time(time.UnixMilli(1632488963633)), IP: "8.9.10.11", Country: "China", City: "Pekin", Log: "success"},
		},
		{
			channel: "user.lsp",
			data:    `{"type":"transfer","instrument_name":"BTC-PERPETUAL","direction":"sell","amount":5000,"price":63481.75,"currency":"BTC","success":true,"starbase_match_id":987654321,"timestamp":1750000400000}`,
			exp:     &WebsocketLSPEvent{Type: "transfer", InstrumentName: "BTC-PERPETUAL", Currency: currency.BTC, Direction: DirectionSell, Amount: 5000, Price: 63481.75, Success: true, StarbaseMatchID: 987654321, Timestamp: types.Time(time.UnixMilli(1750000400000))},
		},
		{
			channel: "user.changes.BTC-PERPETUAL.raw",
			data:    `{"trades":[{"trade_seq":866638,"trade_id":"1430914","timestamp":1605780344032,"tick_direction":1,"state":"filled","reduce_only":false,"profit_loss":4.898e-05,"price":17391,"post_only":false,"order_type":"market","order_id":"3398016","matching_id":null,"mark_price":17391,"liquidity":"T","instrument_name":"BTC-PERPETUAL","index_price":17501.88,"fee_currency":"BTC","fee":1.6e-07,"direction":"sell","amount":10}],"positions":[{"total_profit_loss":1.69711368,"size_currency":10.646886321,"size":185160,"settlement_price":16025.83,"realized_profit_loss":0.012454598,"realized_funding":0.01235663,"mark_price":17391,"maintenance_margin":0.234575865,"leverage":33.0,"kind":"future","interest_value":1.7362511643080387,"instrument_name":"BTC-PERPETUAL","initial_margin":0.319750953,"index_price":17501.88,"floating_profit_loss":0.906961435,"direction":"buy","delta":10.646886321,"average_price":15000,"user_id":1234}],"orders":[{"web":true,"time_in_force":"good_til_cancelled","replaced":false,"reduce_only":false,"price":15665.5,"post_only":false,"order_type":"market","order_state":"filled","order_id":"3398016","last_update_timestamp":1605780344032,"label":"","is_rebalance":false,"is_liquidation":false,"instrument_name":"BTC-PERPETUAL","filled_amount":10,"direction":"sell","creation_timestamp":1605780344032,"average_price":17391,"api":false,"amount":10}],"instrument_name":"BTC-PERPETUAL"}`,
			exp: &WebsocketUserChanges{
				InstrumentName: "BTC-PERPETUAL",
				Orders: []Order{{
					OrderID:             "3398016",
					OrderState:          "filled",
					OrderType:           "market",
					TimeInForce:         "good_til_cancelled",
					InstrumentName:      "BTC-PERPETUAL",
					CreationTimestamp:   types.Time(time.UnixMilli(1605780344032)),
					LastUpdateTimestamp: types.Time(time.UnixMilli(1605780344032)),
					Direction:           DirectionSell,
					Price:               OrderPrice{Value: 15665.5},
					Web:                 true,
					Amount:              10,
					FilledAmount:        10,
					AveragePrice:        17391,
				}},
				Trades: []UserTrade{{
					TradeID:        "1430914",
					TradeSequence:  866638,
					InstrumentName: "BTC-PERPETUAL",
					Timestamp:      types.Time(time.UnixMilli(1605780344032)),
					OrderType:      "market",
					OrderID:        "3398016",
					Direction:      DirectionSell,
					TickDirection:  1,
					IndexPrice:     17501.88,
					Price:          17391,
					Amount:         10,
					Liquidity:      "T",
					Fee:            1.6e-07,
					FeeCurrency:    currency.BTC,
					State:          "filled",
					ProfitLoss:     4.898e-05,
					MarkPrice:      17391,
				}},
				Positions: []Position{{
					InstrumentName:     "BTC-PERPETUAL",
					Kind:               KindFuture,
					UserID:             1234,
					Direction:          DirectionBuy,
					Size:               185160,
					SizeCurrency:       10.646886321,
					AveragePrice:       15000,
					MarkPrice:          17391,
					IndexPrice:         17501.88,
					SettlementPrice:    16025.83,
					Delta:              10.646886321,
					InitialMargin:      0.319750953,
					MaintenanceMargin:  0.234575865,
					Leverage:           33,
					TotalProfitLoss:    1.69711368,
					FloatingProfitLoss: 0.906961435,
					RealisedProfitLoss: 0.012454598,
					RealisedFunding:    0.01235663,
					InterestValue:      1.7362511643080387,
				}},
			},
		},
		{
			channel: "block_trade_confirmations",
			data:    `{"nonce":"bt-jdqv98","role":"maker","user_id":7,"state":{"value":"rejected","timestamp":1711468632693},"trades":[{"instrument_name":"BTC-PERPETUAL","price":70246.66,"direction":"buy","amount":10}],"timestamp":1711468468131}`,
			exp: &PendingBlockTrade{
				Timestamp: types.Time(time.UnixMilli(1711468468131)),
				Nonce:     "bt-jdqv98",
				Trades:    []BlockTradeLeg{{InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy, Price: 70246.66, Amount: 10}},
				Role:      "maker",
				UserID:    7,
				State:     PendingBlockTradeState{Value: "rejected", Timestamp: types.Time(time.UnixMilli(1711468632693))},
			},
		},
		{
			channel: "block_rfq.maker.btc",
			data:    `{"state":"open","combo_id":"BTC-18NOV24-82000-C","legs":[{"direction":"buy","instrument_name":"BTC-18NOV24-82000-C","ratio":1}],"amount":25,"role":"maker","expiration_timestamp":1731664976443,"block_rfq_id":722,"creation_timestamp":1731664676443,"taker_rating":"1-2","disclosed":true,"taker":"TAKER1"}`,
			exp: &BlockRFQ{
				BlockRFQID:          722,
				CreationTimestamp:   types.Time(time.UnixMilli(1731664676443)),
				ExpirationTimestamp: types.Time(time.UnixMilli(1731664976443)),
				Role:                "maker",
				State:               "open",
				TakerRating:         "1-2",
				Amount:              25,
				Legs:                []BlockRFQLeg{{InstrumentName: "BTC-18NOV24-82000-C", Direction: DirectionBuy, Ratio: 1}},
				ComboID:             "BTC-18NOV24-82000-C",
				Disclosed:           true,
				Taker:               "TAKER1",
			},
		},
		{
			channel: "block_rfq.taker.btc",
			data:    `{"label":"example","state":"open","amount":10000,"role":"taker","bids":[{"amount":10000,"price":291664.14,"makers":["ANONYMOUS"],"last_update_timestamp":1740047910507,"execution_instruction":"any_part_of"}],"asks":[],"combo_id":null,"legs":[{"direction":"buy","instrument_name":"BTC-21FEB25","ratio":1},{"direction":"buy","instrument_name":"BTC-PERPETUAL","ratio":1}],"min_trade_amount":10,"makers":["MAKER1","MAKER2"],"creation_timestamp":1740047910438,"block_rfq_id":321,"expiration_timestamp":1740048210438,"taker_rating":"1-2","disclosed":true,"taker":"TAKER1"}`,
			exp: &BlockRFQ{
				BlockRFQID:          321,
				CreationTimestamp:   types.Time(time.UnixMilli(1740047910438)),
				ExpirationTimestamp: types.Time(time.UnixMilli(1740048210438)),
				Role:                "taker",
				State:               "open",
				TakerRating:         "1-2",
				Makers:              []string{"MAKER1", "MAKER2"},
				Amount:              10000,
				MinTradeAmount:      10,
				Asks:                []BlockRFQPriceLevel{},
				Bids:                []BlockRFQPriceLevel{{Makers: []string{"ANONYMOUS"}, Price: 291664.14, Amount: 10000, LastUpdateTimestamp: types.Time(time.UnixMilli(1740047910507)), ExecutionInstruction: "any_part_of"}},
				Legs:                []BlockRFQLeg{{InstrumentName: "BTC-21FEB25", Direction: DirectionBuy, Ratio: 1}, {InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy, Ratio: 1}},
				Label:               "example",
				Disclosed:           true,
				Taker:               "TAKER1",
			},
		},
		{
			channel: "block_rfq.maker.quotes.btc",
			data:    `[{"label":"example_quote","price":10,"direction":"buy","legs":[{"price":10,"direction":"buy","instrument_name":"BTC-16NOV24-82000-C","ratio":1}],"amount":25,"block_rfq_id":724,"replaced":false,"filled_amount":0,"last_update_timestamp":1731665928291,"creation_timestamp":1731665928291,"block_rfq_quote_id":1301,"quote_state":"open"}]`,
			exp: &WebsocketBlockRFQQuotes{{
				BlockRFQQuoteID:     1301,
				BlockRFQID:          724,
				CreationTimestamp:   types.Time(time.UnixMilli(1731665928291)),
				LastUpdateTimestamp: types.Time(time.UnixMilli(1731665928291)),
				QuoteState:          "open",
				Price:               10,
				Amount:              25,
				Direction:           DirectionBuy,
				Legs:                []BlockRFQLeg{{InstrumentName: "BTC-16NOV24-82000-C", Direction: DirectionBuy, Ratio: 1, Price: 10}},
				Label:               "example_quote",
			}},
		},
	} {
		require.NoErrorf(t, ex.wsHandleNotification(t.Context(), tc.channel, notification(tc.channel, tc.data)), "wsHandleNotification must not error for %s", tc.channel)
		assert.Equalf(t, tc.exp, relayed(t, ex), "wsHandleNotification should relay %s's data", tc.channel)
	}

	require.NoError(t, ex.wsHandleNotification(t.Context(), "bogus.channel", notification("bogus.channel", `{}`)), "wsHandleNotification must not error for an unknown channel")
	assert.IsType(t, websocket.UnhandledMessageWarning{}, relayed(t, ex), "wsHandleNotification should relay an unknown channel as unhandled")
}

func TestNotificationChannelName(t *testing.T) {
	t.Parallel()
	for channel, exp := range map[string]string{
		"book.BTC-PERPETUAL.100ms":                ChannelBook,
		"book.BTC-PERPETUAL.none.10.100ms":        ChannelBook,
		"trades.future.BTC.100ms":                 ChannelTrades,
		"user.trades.BTC-PERPETUAL.raw":           ChannelUserTrades,
		"user.combo_trades.future.any.100ms":      ChannelUserComboTrades,
		"chart.trades.BTC-PERPETUAL.1":            ChannelChartTrades,
		"block_rfq.maker.quotes.any":              ChannelBlockRFQMakerQuotes,
		"block_rfq.maker.BTC":                     ChannelBlockRFQMaker,
		"platform_state":                          ChannelPlatformState,
		"platform_state.public_methods_state":     ChannelPlatformPublicMethodsState,
		"user.isolated.liquidation":               ChannelUserIsolatedLiquidation,
		"user.liquidation":                        ChannelUserLiquidation,
		"block_trade_confirmations":               ChannelBlockTradeConfirmations,
		"block_trade_confirmations.BTC":           ChannelBlockTradeConfirmations,
		"incremental_ticker.BTC-PERPETUAL":        ChannelIncrementalTicker,
		"ticker.BTC-PERPETUAL.100ms":              ChannelTicker,
		"deribit_price_index.btc_usd":             ChannelPriceIndex,
		"tickers.BTC-PERPETUAL.100ms":             "",
		"bookkeeping":                             "",
		"user":                                    "",
		"block_rfq":                               "",
		"markprice.options.btc_usd":               ChannelMarkPriceOptions,
		"instrument.creation.option.BTC":          ChannelInstrumentCreation,
		"user.orders.BTC-PERPETUAL.raw":           ChannelUserOrders,
		"user.changes.BTC-PERPETUAL.100ms":        ChannelUserChanges,
		"user.portfolio.any":                      ChannelUserPortfolio,
		"user.access_log":                         ChannelUserAccessLog,
		"estimated_expiration_price.btc_usd":      ChannelEstimatedExpirationPrice,
		"perpetual.BTC-PERPETUAL.100ms":           ChannelPerpetual,
		"quote.BTC-PERPETUAL":                     ChannelQuote,
		"announcements":                           ChannelAnnouncements,
		"user.lsp":                                ChannelUserLSP,
		"user.lock":                               ChannelUserLock,
		"user.mmp_trigger.btc_usd":                ChannelUserMMPTrigger,
		"deribit_price_ranking.btc_usd":           ChannelPriceRanking,
		"deribit_price_statistics.btc_usd":        ChannelPriceStatistics,
		"deribit_volatility_index.btc_usd":        ChannelVolatilityIndex,
		"block_rfq.taker.any":                     ChannelBlockRFQTaker,
		"block_rfq.trades.any":                    ChannelBlockRFQTrades,
		"instrument.state.future.any":             ChannelInstrumentState,
		"user.trades.option.BTC.100ms":            ChannelUserTrades,
		"user.orders.future.BTC.raw":              ChannelUserOrders,
		"user.combo_trades.BTC-FS-30OCT26_PERP.1": ChannelUserComboTrades,
	} {
		assert.Equalf(t, exp, notificationChannelName(channel), "notificationChannelName should match %s", channel)
	}
	assert.Len(t, websocketChannelsByLength, len(websocketChannels), "websocketChannelsByLength should hold every channel")
}

func TestDecodeNotification(t *testing.T) {
	t.Parallel()
	var data WebsocketPriceIndex
	require.NoError(t, decodeNotification(notification("deribit_price_index.btc_usd", `{"timestamp":1792107433668,"price":82119.87,"index_name":"btc_usd"}`), &data), "decodeNotification must not error")
	assert.Equal(t, WebsocketPriceIndex{Timestamp: types.Time(time.UnixMilli(1792107433668)), IndexName: "btc_usd", Price: 82119.87}, data, "decodeNotification should decode the notification's data")
	assert.ErrorIs(t, decodeNotification(notification("deribit_price_index.btc_usd", `{"price":"cheap"}`), &data), errMalformedNotification, "decodeNotification should reject data of the wrong type")
}

func TestWebsocketRelay(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, wsRelay[WebsocketVolatilityIndex](t.Context(), ex, notification("deribit_volatility_index.btc_usd", `{"timestamp":1792107470023,"index_name":"btc_usd","volatility":38.76}`)), "wsRelay must not error")
	assert.Equal(t, &WebsocketVolatilityIndex{Timestamp: types.Time(time.UnixMilli(1792107470023)), IndexName: "btc_usd", Volatility: 38.76}, relayed(t, ex), "wsRelay should relay a pointer to the decoded data")
	assert.ErrorIs(t, wsRelay[WebsocketVolatilityIndex](t.Context(), ex, notification("deribit_volatility_index.btc_usd", `[]`)), errMalformedNotification, "wsRelay should return a decoding failure")
	assertNothingRelayed(t, ex)
}

// testWebsocketOrder is an order as a user.orders notification sends it, following Deribit's documented example
const testWebsocketOrder = `{"time_in_force":"good_til_cancelled","replaced":false,"reduce_only":false,"price":10502.52,"post_only":false,"original_order_type":"market","order_type":"limit","order_state":"open","order_id":"5","max_show":200,"last_update_timestamp":1581507423789,"label":"","is_rebalance":false,"is_liquidation":false,"instrument_name":"BTC-PERPETUAL","filled_amount":0,"direction":"buy","creation_timestamp":1581507423789,"average_price":0,"api":false,"amount":200}`

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	// Deribit routes these enabled spot pairs to Coinbase Exchange, and refuses their trades and candles
	routed := []string{"BTC_USDC", "ETH_USDC"}
	for _, name := range routed {
		require.Truef(t, ex.isCoinbaseRoutedSpot(name), "generateSubscriptions must load %s as routed to Coinbase Exchange", name)
	}
	exp := subscription.List{}
	for _, s := range ex.Features.Subscriptions {
		for _, a := range ex.GetAssetTypes(true) {
			if !ex.IsAssetWebsocketSupported(a) {
				continue
			}
			pairs, err := ex.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
			for i, p := range pairs {
				sub := s.Clone()
				sub.Asset = a
				sub.Pairs = pairs[i : i+1]
				name, err := websocketChannelName(sub)
				require.NoError(t, err, "websocketChannelName must not error")
				if a == asset.Spot && (name == ChannelTrades || name == ChannelChartTrades) && slices.Contains(routed, formatPairString(a, p)) {
					continue
				}
				interval := intervalHundredMS
				if name == ChannelChartTrades {
					interval = "1D"
				}
				sub.QualifiedChannel = name + "." + formatPairString(a, p) + "." + interval
				exp = append(exp, sub)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	subs, err = ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error unauthenticated")
	assert.False(t, slices.ContainsFunc(subs, func(s *subscription.Subscription) bool { return s.Authenticated }), "generateSubscriptions should leave out private channels without authentication")

	// spotSubscriptions enables only a Coinbase-routed and a matched spot pair, and subscribes the channels given
	spotSubscriptions := func(t *testing.T, ex *Exchange, channels ...string) {
		t.Helper()
		setWrapperTestPairs(t, ex, asset.Spot, currency.NewPairWithDelimiter("BTC", "USDC", currency.UnderscoreDelimiter), currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter))
		for _, a := range ex.GetAssetTypes(false) {
			if a != asset.Spot {
				require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error for %s", a)
			}
		}
		ex.Features.Subscriptions = subscription.List{}
		for _, channel := range channels {
			interval := kline.HundredMilliseconds
			if channel == subscription.CandlesChannel {
				interval = kline.OneDay
			}
			ex.Features.Subscriptions = append(ex.Features.Subscriptions, &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: channel, Interval: interval})
		}
	}
	// spotInstruments is a public/get_instruments response with a Coinbase-routed and a matched spot pair
	const spotInstruments = `{"jsonrpc":"2.0","result":[{"instrument_name":"BTC_USDC","kind":"spot","is_active":true,"is_cbe_routed":true,"is_csr":true},{"instrument_name":"ETH_BTC","kind":"spot","is_active":true}]}`
	streamed := []string{"ticker.BTC_USDC.100ms", "ticker.ETH_BTC.100ms", "trades.ETH_BTC.100ms", "chart.trades.ETH_BTC.1D"}

	t.Run("routed spot loaded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperServerExchange(t, func(method string, _ json.RawMessage) string {
			assert.Failf(t, "unexpected request", "generateSubscriptions should not request %s once the routed spot is loaded", method)
			return spotInstruments
		})
		spotSubscriptions(t, ex, subscription.TickerChannel, subscription.AllTradesChannel, subscription.CandlesChannel)
		_, err := ex.tradablePairs(asset.Spot, []Instrument{{InstrumentName: "BTC_USDC", IsActive: true, IsCoinbaseExchangeRouted: true}})
		require.NoError(t, err, "tradablePairs must not error")
		subs, err := ex.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not error")
		assert.ElementsMatch(t, streamed, subs.QualifiedChannels(), "generateSubscriptions should leave out the trades and candles of Coinbase-routed spot")
	})

	t.Run("routed spot unloaded", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var requests []string
		ex := newWrapperServerExchange(t, func(method string, params json.RawMessage) string {
			mu.Lock()
			requests = append(requests, method+" "+string(params))
			mu.Unlock()
			return spotInstruments
		})
		spotSubscriptions(t, ex, subscription.TickerChannel, subscription.AllTradesChannel, subscription.CandlesChannel)
		subs, err := ex.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not error")
		assert.ElementsMatch(t, streamed, subs.QualifiedChannels(), "generateSubscriptions should load the spot instruments to leave out the trades and candles of Coinbase-routed spot")
		mu.Lock()
		assert.Equal(t, []string{`public/get_instruments {"currency":"any","kind":"spot"}`}, requests, "generateSubscriptions should request the spot instruments once")
		mu.Unlock()
		_, err = ex.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not error again")
		mu.Lock()
		assert.Len(t, requests, 1, "generateSubscriptions should not request the spot instruments again once loaded")
		mu.Unlock()
	})

	t.Run("spot trades by kind", func(t *testing.T) {
		t.Parallel()
		var requests atomic.Int32
		ex := newWrapperServerExchange(t, func(string, json.RawMessage) string {
			requests.Add(1)
			return spotInstruments
		})
		ex.Features.Subscriptions = subscription.List{
			{Enabled: true, Channel: subscription.AllTradesChannel, Interval: kline.HundredMilliseconds, Params: map[string]any{"kind": "spot"}},
			{Enabled: true, Channel: subscription.AllTradesChannel, Interval: kline.HundredMilliseconds, Params: map[string]any{"kind": "spot", "currency": "ETH"}},
			{Enabled: true, Channel: subscription.AllTradesChannel, Interval: kline.HundredMilliseconds, Params: map[string]any{"kind": "future"}},
		}
		subs, err := ex.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not error")
		assert.ElementsMatch(t, []string{"trades.spot.ETH.100ms", "trades.future.any.100ms"}, subs.QualifiedChannels(), "generateSubscriptions should leave out spot trades by kind of every currency, which Deribit refuses while any spot is routed")
		assert.Equal(t, int32(1), requests.Load(), "generateSubscriptions should load the spot instruments for spot trades by kind")
	})

	t.Run("routed spot not needed", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperServerExchange(t, func(method string, _ json.RawMessage) string {
			assert.Failf(t, "unexpected request", "generateSubscriptions should not request %s without spot trades or candles to subscribe", method)
			return spotInstruments
		})
		spotSubscriptions(t, ex, subscription.TickerChannel, subscription.OrderbookChannel)
		subs, err := ex.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not error")
		assert.ElementsMatch(t, []string{"ticker.BTC_USDC.100ms", "ticker.ETH_BTC.100ms", "book.BTC_USDC.100ms", "book.ETH_BTC.100ms"}, subs.QualifiedChannels(), "generateSubscriptions should subscribe every pair's ticker and book")
		assert.Nil(t, ex.coinbaseRoutedSpot, "generateSubscriptions should leave the routed spot unloaded")
	})

	t.Run("loading fails", func(t *testing.T) {
		t.Parallel()
		ex := newUnavailableServerExchange(t)
		spotSubscriptions(t, ex, subscription.AllTradesChannel)
		_, err := ex.generateSubscriptions()
		assert.ErrorIs(t, err, request.ErrBadStatus, "generateSubscriptions should return the failure to load the spot instruments")
		assert.Nil(t, ex.coinbaseRoutedSpot, "generateSubscriptions should leave the routed spot unloaded, for the next generation to load")
	})
}

func TestIsSpotKindTrades(t *testing.T) {
	t.Parallel()
	spot := currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		exp  bool
	}{
		{name: "spot trades", sub: &subscription.Subscription{Channel: subscription.AllTradesChannel, Params: map[string]any{"kind": "spot"}}, exp: true},
		{name: "spot trades named by channel", sub: &subscription.Subscription{Channel: ChannelTrades, Params: map[string]any{"kind": "SPOT", "currency": "BTC"}}, exp: true},
		{name: "future trades", sub: &subscription.Subscription{Channel: ChannelTrades, Params: map[string]any{"kind": "future"}}},
		{name: "trades without a kind", sub: &subscription.Subscription{Channel: ChannelTrades}},
		{name: "a spot pair's trades", sub: &subscription.Subscription{Asset: asset.Spot, Channel: ChannelTrades, Pairs: currency.Pairs{spot}, Params: map[string]any{"kind": "spot"}}},
		{name: "spot instrument states", sub: &subscription.Subscription{Channel: ChannelInstrumentState, Params: map[string]any{"kind": "spot"}}},
		{name: "an unknown channel", sub: &subscription.Subscription{Channel: "bogus", Params: map[string]any{"kind": "spot"}}},
		{name: "a malformed kind", sub: &subscription.Subscription{Channel: ChannelTrades, Params: map[string]any{"kind": 1}}},
	} {
		assert.Equalf(t, tc.exp, isSpotKindTrades(tc.sub), "isSpotKindTrades should report %s correctly", tc.name)
	}
}

func TestIsRoutedSpotSensitive(t *testing.T) {
	t.Parallel()
	spot := currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter)
	assert.True(t, isRoutedSpotSensitive(&subscription.Subscription{Asset: asset.Spot, Channel: ChannelChartTrades, Pairs: currency.Pairs{spot}}), "isRoutedSpotSensitive should report a spot pair's candles")
	assert.True(t, isRoutedSpotSensitive(&subscription.Subscription{Channel: ChannelTrades, Params: map[string]any{"kind": "spot"}}), "isRoutedSpotSensitive should report spot trades by kind")
	assert.False(t, isRoutedSpotSensitive(&subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spot}}), "isRoutedSpotSensitive should not report a spot pair's ticker")
}

func TestExpandSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	subs, err := ex.expandSubscriptions(subscription.List{
		{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}},
		{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "eth_usd"}},
	})
	require.NoError(t, err, "expandSubscriptions must not error")
	require.Len(t, subs, 2, "expandSubscriptions must qualify each subscription")
	for _, s := range subs {
		assert.Equalf(t, qualifiedChannelKey{s}, s.EnsureKeyed(), "expandSubscriptions should key %s by its qualified channel", s.QualifiedChannel)
	}
	_, err = ex.expandSubscriptions(subscription.List{{Channel: ChannelPriceIndex}})
	assert.ErrorIs(t, err, errChannelParameter, "expandSubscriptions should return a qualification failure")

	// The store reads a held subscription's key without its lock while it compares subscriptions, so expanding a held
	// subscription again must leave its key alone, which the race detector checks
	store := subscription.NewStore()
	require.NoError(t, store.Add(subs[0]), "Add must not error")
	compare, err := ex.expandSubscriptions(subscription.List{{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}})
	require.NoError(t, err, "expandSubscriptions must not error")
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			_, _ = ex.expandSubscriptions(subscription.List{subs[0]})
		}
	})
	wg.Go(func() {
		for range 100 {
			store.Diff(compare)
		}
	})
	wg.Wait()
	assert.Equal(t, qualifiedChannelKey{subs[0]}, subs[0].EnsureKeyed(), "expandSubscriptions should leave a held subscription keyed by its qualified channel")

	keyedByAnother := &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "eth_usd"}, QualifiedChannel: "deribit_price_index.eth_usd"}
	keyedByAnother.SetKey(qualifiedChannelKey{subs[0]})
	_, err = ex.expandSubscriptions(subscription.List{keyedByAnother})
	require.NoError(t, err, "expandSubscriptions must not error")
	assert.Equal(t, qualifiedChannelKey{keyedByAnother}, keyedByAnother.EnsureKeyed(), "expandSubscriptions should key a subscription keyed by another one by its own qualified channel")
}

func TestQualifiedChannelKey(t *testing.T) {
	t.Parallel()
	btc := &subscription.Subscription{Channel: ChannelPriceIndex, QualifiedChannel: "deribit_price_index.btc_usd"}
	key := qualifiedChannelKey{btc}
	assert.True(t, key.Match(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "deribit_price_index.btc_usd"}}), "Match should match the same qualified channel")
	assert.False(t, key.Match(qualifiedChannelKey{&subscription.Subscription{Channel: ChannelPriceIndex, QualifiedChannel: "deribit_price_index.eth_usd"}}), "Match should not match a channel differing by a parameter")
	assert.False(t, key.Match(nil), "Match should not match a nil key")
	assert.False(t, key.Match(qualifiedChannelKey{}), "Match should not match a key without a subscription")
	assert.Same(t, btc, key.GetSubscription(), "GetSubscription should return the key's subscription")
	assert.Equal(t, "deribit_price_index.btc_usd", key.String(), "String should return the qualified channel")
	assert.Equal(t, "Uninitialised qualifiedChannelKey", qualifiedChannelKey{}.String(), "String should describe a key without a subscription")
}

func TestIsSpotTradesOrCandles(t *testing.T) {
	t.Parallel()
	spot := currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter)
	perp := currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp bool
	}{
		{sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{spot}}, exp: true},
		{sub: &subscription.Subscription{Asset: asset.Spot, Channel: ChannelTrades, Pairs: currency.Pairs{spot}}, exp: true},
		{sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.CandlesChannel, Pairs: currency.Pairs{spot}}, exp: true},
		{sub: &subscription.Subscription{Asset: asset.Spot, Channel: ChannelChartTrades, Pairs: currency.Pairs{spot}}, exp: true},
		{sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spot}}},
		{sub: &subscription.Subscription{Asset: asset.Futures, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{perp}}},
		{sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{spot, perp}}},
		{sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.AllTradesChannel}},
		{sub: &subscription.Subscription{Asset: asset.Spot, Channel: "bogus", Pairs: currency.Pairs{spot}}},
	} {
		assert.Equalf(t, tc.exp, isSpotTradesOrCandles(tc.sub), "isSpotTradesOrCandles should report %s %s of %d pairs correctly", tc.sub.Asset, tc.sub.Channel, len(tc.sub.Pairs))
	}
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	perp := currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)
	spot := currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp []string
	}{
		{sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{perp}, Interval: kline.HundredMilliseconds}, exp: []string{"ticker.BTC-PERPETUAL.100ms"}},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spot}, Interval: kline.ThousandMilliseconds, Levels: 10, Params: map[string]any{"group": float64(5)}}, exp: []string{"book.ETH_BTC.5.10.agg2"}},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Futures, Pairs: currency.Pairs{perp}, Interval: kline.FifteenMin}, exp: []string{"chart.trades.BTC-PERPETUAL.15"}},
		{sub: &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true, Interval: kline.Raw, Params: map[string]any{"kind": KindFuture, "currency": "btc"}}, exp: []string{"user.orders.future.BTC.raw"}},
		{sub: &subscription.Subscription{Channel: ChannelTrades, Interval: kline.HundredMilliseconds, Params: map[string]any{"kind": KindFuture}}, exp: []string{"trades.future.any.100ms"}},
		{sub: &subscription.Subscription{Channel: ChannelIncrementalTicker, Asset: asset.Futures, Pairs: currency.Pairs{perp}}, exp: []string{"incremental_ticker.BTC-PERPETUAL"}},
		{sub: &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}, exp: []string{"deribit_price_index.btc_usd"}},
		{sub: &subscription.Subscription{Channel: subscription.MyAccountChannel, Authenticated: true}, exp: []string{"user.portfolio.any"}},
		{sub: &subscription.Subscription{Channel: ChannelBlockTradeConfirmations, Authenticated: true}, exp: []string{"block_trade_confirmations"}},
		{sub: &subscription.Subscription{Channel: ChannelBlockTradeConfirmations, Authenticated: true, Params: map[string]any{"currency": "eth"}}, exp: []string{"block_trade_confirmations.ETH"}},
		{sub: &subscription.Subscription{Channel: ChannelPlatformState}, exp: []string{"platform_state"}},
		{sub: &subscription.Subscription{Channel: ChannelInstrumentState, Params: map[string]any{"kind": KindOption}}, exp: []string{"instrument.state.option.any"}},
	} {
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		subs, err := subscription.List{tc.sub}.ExpandTemplates(ex)
		require.NoErrorf(t, err, "ExpandTemplates must not error for %s", tc.sub.Channel)
		assert.Equalf(t, tc.exp, subs.QualifiedChannels(), "GetSubscriptionTemplate should qualify %s", tc.sub.Channel)
	}

	_, err := subscription.List{{Channel: ChannelQuote}}.ExpandTemplates(ex)
	assert.ErrorIs(t, err, errChannelNeedsInstrument, "GetSubscriptionTemplate should need pairs for an instrument channel")
	_, err = subscription.List{{Channel: ChannelTrades, Interval: kline.HundredMilliseconds}}.ExpandTemplates(ex)
	assert.ErrorIs(t, err, errChannelParameter, "GetSubscriptionTemplate should need a kind for trades without an asset")
	_, err = subscription.List{{Channel: "bogus"}}.ExpandTemplates(ex)
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "GetSubscriptionTemplate should reject an unknown channel")
}

func TestWebsocketChannelName(t *testing.T) {
	t.Parallel()
	for channel, exp := range map[string]string{
		subscription.TickerChannel:    ChannelTicker,
		subscription.OrderbookChannel: ChannelBook,
		subscription.AllTradesChannel: ChannelTrades,
		subscription.CandlesChannel:   ChannelChartTrades,
		subscription.MyOrdersChannel:  ChannelUserOrders,
		subscription.MyTradesChannel:  ChannelUserTrades,
		subscription.MyAccountChannel: ChannelUserPortfolio,
		ChannelUserLock:               ChannelUserLock,
	} {
		name, err := websocketChannelName(&subscription.Subscription{Channel: channel})
		require.NoErrorf(t, err, "websocketChannelName must not error for %s", channel)
		assert.Equalf(t, exp, name, "websocketChannelName should name %s's channel", channel)
	}
	_, err := websocketChannelName(&subscription.Subscription{Channel: "wibble"})
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "websocketChannelName should reject an unknown channel")
	_, err = websocketChannelName(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "websocketChannelName should reject a nil subscription")
}

func TestQualifyInstrumentChannel(t *testing.T) {
	t.Parallel()
	perp := currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp string
		err error
	}{
		{sub: &subscription.Subscription{Channel: ChannelBook, Interval: kline.HundredMilliseconds}, exp: "book.BTC-PERPETUAL.100ms"},
		{sub: &subscription.Subscription{Channel: ChannelBook, Interval: kline.HundredMilliseconds, Levels: 20}, exp: "book.BTC-PERPETUAL.none.20.100ms"},
		{sub: &subscription.Subscription{Channel: ChannelBook, Interval: kline.HundredMilliseconds, Levels: 5}, err: subscription.ErrInvalidLevel},
		{sub: &subscription.Subscription{Channel: ChannelBook, Interval: kline.HundredMilliseconds, Levels: 1, Params: map[string]any{"group": "3"}}, err: errChannelParameter},
		{sub: &subscription.Subscription{Channel: ChannelBook, Interval: kline.Raw}, err: errChannelNeedsAuthentication},
		{sub: &subscription.Subscription{Channel: ChannelBook, Interval: kline.Raw, Authenticated: true}, exp: "book.BTC-PERPETUAL.raw"},
		{sub: &subscription.Subscription{Channel: ChannelUserChanges, Interval: kline.HundredMilliseconds}, err: errChannelNeedsAuthentication},
		{sub: &subscription.Subscription{Channel: ChannelQuote}, exp: "quote.BTC-PERPETUAL"},
		{sub: &subscription.Subscription{Channel: "bogus"}, err: subscription.ErrNotSupported},
	} {
		qualified, err := qualifyInstrumentChannel(tc.sub, asset.Futures, perp)
		require.ErrorIsf(t, err, tc.err, "qualifyInstrumentChannel must return the expected error for %s", tc.sub.Channel)
		assert.Equalf(t, tc.exp, qualified, "qualifyInstrumentChannel should qualify %s", tc.sub.Channel)
	}
}

func TestQualifyChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp string
		err error
	}{
		{sub: &subscription.Subscription{Channel: ChannelUserTrades, Authenticated: true, Interval: kline.HundredMilliseconds, Params: map[string]any{"kind": KindOption, "currency": "BTC"}}, exp: "user.trades.option.BTC.100ms"},
		{sub: &subscription.Subscription{Channel: ChannelInstrumentCreation}, exp: "instrument.creation.any.any"},
		{sub: &subscription.Subscription{Channel: ChannelInstrumentCreation, Params: map[string]any{"kind": 5}}, err: errChannelParameter},
		{sub: &subscription.Subscription{Channel: ChannelInstrumentCreation, Params: map[string]any{"currency": 5}}, err: errChannelParameter},
		{sub: &subscription.Subscription{Channel: ChannelTicker, Interval: kline.HundredMilliseconds}, err: errChannelNeedsInstrument},
		{sub: &subscription.Subscription{Channel: ChannelMarkPriceOptions, Params: map[string]any{"index_name": "eth_usd"}}, exp: "markprice.options.eth_usd"},
		{sub: &subscription.Subscription{Channel: ChannelMarkPriceOptions}, err: errChannelParameter},
		{sub: &subscription.Subscription{Channel: ChannelBlockRFQTrades, Params: map[string]any{"currency": currency.USDC}}, exp: "block_rfq.trades.USDC"},
		{sub: &subscription.Subscription{Channel: ChannelBlockRFQTrades, Params: map[string]any{"currency": true}}, err: errChannelParameter},
		{sub: &subscription.Subscription{Channel: ChannelBlockTradeConfirmations, Authenticated: true, Params: map[string]any{"currency": []string{}}}, err: errChannelParameter},
		{sub: &subscription.Subscription{Channel: ChannelBlockRFQMaker, Params: map[string]any{"currency": "BTC"}}, err: errChannelNeedsAuthentication},
		{sub: &subscription.Subscription{Channel: ChannelAnnouncements}, exp: "announcements"},
		{sub: &subscription.Subscription{Channel: ChannelTrades, Interval: kline.HundredMilliseconds, Params: map[string]any{"kind": KindOption, "currency": "btc"}}, exp: "trades.option.BTC.100ms"},
		{sub: &subscription.Subscription{Channel: ChannelTrades, Interval: kline.HundredMilliseconds}, err: errChannelParameter},
		{sub: &subscription.Subscription{Channel: ChannelTrades, Interval: kline.HundredMilliseconds, Params: map[string]any{"kind": channelAny}}, err: errChannelParameter},
		{sub: &subscription.Subscription{Channel: ChannelTrades, Interval: kline.HundredMilliseconds, Params: map[string]any{"kind": "ANY", "currency": "BTC"}}, err: errChannelParameter},
		{sub: &subscription.Subscription{Channel: ChannelTrades, Interval: kline.OneMin, Params: map[string]any{"kind": KindFuture}}, err: kline.ErrUnsupportedInterval},
		{sub: &subscription.Subscription{Channel: "bogus"}, err: subscription.ErrNotSupported},
	} {
		qualified, err := qualifyChannel(tc.sub)
		require.ErrorIsf(t, err, tc.err, "qualifyChannel must return the expected error for %s", tc.sub.Channel)
		assert.Equalf(t, tc.exp, qualified, "qualifyChannel should qualify %s", tc.sub.Channel)
	}
}

func TestValidateChannelSubscription(t *testing.T) {
	t.Parallel()
	name, channel, err := validateChannelSubscription(&subscription.Subscription{Channel: subscription.MyTradesChannel, Authenticated: true})
	require.NoError(t, err, "validateChannelSubscription must not error for an authenticated private channel")
	assert.Equal(t, ChannelUserTrades, name, "validateChannelSubscription should return the Deribit channel")
	assert.Equal(t, websocketChannels[ChannelUserTrades], channel, "validateChannelSubscription should return the channel's description")
	_, _, err = validateChannelSubscription(&subscription.Subscription{Channel: subscription.MyTradesChannel})
	assert.ErrorIs(t, err, errChannelNeedsAuthentication, "validateChannelSubscription should need a private channel to be authenticated")
	_, _, err = validateChannelSubscription(&subscription.Subscription{Channel: "bogus"})
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "validateChannelSubscription should reject an unknown channel")
}

func TestAppendChannelInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		sub     *subscription.Subscription
		channel websocketChannel
		exp     string
		err     error
	}{
		{sub: &subscription.Subscription{Interval: kline.HundredMilliseconds}, channel: websocketChannel{interval: true}, exp: "c.100ms"},
		{sub: &subscription.Subscription{Interval: kline.ThousandMilliseconds}, channel: websocketChannel{interval: true}, exp: "c.agg2"},
		{sub: &subscription.Subscription{Interval: kline.Raw, Authenticated: true}, channel: websocketChannel{interval: true}, exp: "c.raw"},
		{sub: &subscription.Subscription{Interval: kline.Raw}, channel: websocketChannel{interval: true}, err: errChannelNeedsAuthentication},
		{sub: &subscription.Subscription{}, channel: websocketChannel{interval: true}, err: kline.ErrUnsupportedInterval},
		{sub: &subscription.Subscription{Interval: kline.OneHour}, channel: websocketChannel{resolution: true}, exp: "c.60"},
		{sub: &subscription.Subscription{Interval: kline.HundredMilliseconds}, channel: websocketChannel{resolution: true}, err: kline.ErrUnsupportedInterval},
		{sub: &subscription.Subscription{Interval: kline.HundredMilliseconds}, exp: "c"},
	} {
		qualified, err := appendChannelInterval("c", tc.sub, &tc.channel)
		require.ErrorIsf(t, err, tc.err, "appendChannelInterval must return the expected error for %s", tc.sub.Interval)
		assert.Equalf(t, tc.exp, qualified, "appendChannelInterval should append %s", tc.sub.Interval)
	}
}

func TestChartResolution(t *testing.T) {
	t.Parallel()
	for interval, exp := range map[kline.Interval]string{
		kline.OneMin: "1", kline.ThreeMin: "3", kline.FiveMin: "5", kline.TenMin: "10", kline.FifteenMin: "15", kline.ThirtyMin: "30",
		kline.OneHour: "60", kline.TwoHour: "120", kline.ThreeHour: "180", kline.SixHour: "360", kline.TwelveHour: "720", kline.OneDay: "1D",
	} {
		resolution, err := chartResolution(interval)
		require.NoErrorf(t, err, "chartResolution must not error for %s", interval)
		assert.Equalf(t, exp, resolution, "chartResolution should convert %s", interval)
	}
	for _, interval := range []kline.Interval{kline.HundredMilliseconds, kline.FourHour, kline.OneWeek} {
		_, err := chartResolution(interval)
		assert.ErrorIsf(t, err, kline.ErrUnsupportedInterval, "chartResolution should reject %s", interval)
	}
}

func TestBookGroup(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		params map[string]any
		exp    string
		err    error
	}{
		{exp: "none"},
		{params: map[string]any{"group": "none"}, exp: "none"},
		{params: map[string]any{"group": "250"}, exp: "250"},
		{params: map[string]any{"group": float64(25)}, exp: "25"},
		{params: map[string]any{"group": 2}, exp: "2"},
		{params: map[string]any{"group": float64(2.5)}, err: errChannelParameter},
		{params: map[string]any{"group": "4"}, err: errChannelParameter},
		{params: map[string]any{"group": true}, err: errChannelParameter},
	} {
		group, err := bookGroup(tc.params)
		require.ErrorIsf(t, err, tc.err, "bookGroup must return the expected error for %v", tc.params)
		assert.Equalf(t, tc.exp, group, "bookGroup should return the group for %v", tc.params)
	}
}

func TestChannelParameter(t *testing.T) {
	t.Parallel()
	value, err := channelParameter(map[string]any{"kind": KindSpot}, "kind", channelAny)
	require.NoError(t, err, "channelParameter must not error for a set parameter")
	assert.Equal(t, KindSpot, value, "channelParameter should return a set parameter")
	value, err = channelParameter(nil, "kind", channelAny)
	require.NoError(t, err, "channelParameter must not error for an unset parameter with a default")
	assert.Equal(t, channelAny, value, "channelParameter should return the default for an unset parameter")
	for _, params := range []map[string]any{nil, {"index_name": ""}, {"index_name": "btc.usd"}, {"index_name": 1}} {
		_, err = channelParameter(params, "index_name", "")
		assert.ErrorIsf(t, err, errChannelParameter, "channelParameter should reject %v", params)
	}
}

func TestChannelCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		params   map[string]any
		required bool
		exp      string
		err      error
	}{
		{required: true, exp: channelAny},
		{exp: ""},
		{params: map[string]any{"currency": "ANY"}, exp: channelAny},
		{params: map[string]any{"currency": "btc"}, exp: "BTC"},
		{params: map[string]any{"currency": currency.NewCode("usdc")}, exp: "USDC"},
		{params: map[string]any{"currency": ""}, err: errChannelParameter},
		{params: map[string]any{"currency": "btc.usd"}, err: errChannelParameter},
		{params: map[string]any{"currency": 1}, err: errChannelParameter},
	} {
		ccy, err := channelCurrency(tc.params, tc.required)
		require.ErrorIsf(t, err, tc.err, "channelCurrency must return the expected error for %v", tc.params)
		assert.Equalf(t, tc.exp, ccy, "channelCurrency should return the currency for %v", tc.params)
	}
}

// heldSubscription returns the subscription an exchange holds for a qualified channel, or nil; the live tests share one
// connection, so each checks only the channels it subscribes, which no other test subscribes
func heldSubscription(ex *Exchange, channel string) *subscription.Subscription {
	for _, s := range ex.Websocket.GetSubscriptions() {
		if s.QualifiedChannel == channel {
			return s
		}
	}
	return nil
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	const channel = "deribit_price_index.sol_usdc"
	public := &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "sol_usdc"}}
	require.NoError(t, ex.Subscribe(subscription.List{public}), "Subscribe must not error")
	held := heldSubscription(ex, channel)
	require.NotNil(t, held, "Subscribe must store the acknowledged subscription under its qualified channel")
	if mockTests {
		requests := venue.received("public/subscribe")
		require.Len(t, requests, 1, "Subscribe must send a public/subscribe request")
		assert.JSONEq(t, `{"channels":["deribit_price_index.sol_usdc"]}`, string(requests[0].Params), "Subscribe should send the qualified channel")
	}
	require.NoError(t, ex.Unsubscribe(subscription.List{held}), "Unsubscribe must not error")
	assert.Nil(t, heldSubscription(ex, channel), "Unsubscribe should remove the acknowledged subscription")

	assert.ErrorIs(t, ex.Subscribe(subscription.List{{Channel: "bogus"}}), subscription.ErrNotSupported, "Subscribe should reject an unknown channel")
	assert.ErrorIs(t, ex.Subscribe(subscription.List{{Channel: ChannelPriceIndex}}), errChannelParameter, "Subscribe should return a qualification failure")

	if !mockTests {
		return
	}
	perp := currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)
	plainBook := func() *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{perp}, Interval: kline.HundredMilliseconds}
	}
	groupedBook := func() *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{perp}, Interval: kline.HundredMilliseconds, Levels: 10}
	}

	t.Run("channels differing by a parameter", func(t *testing.T) {
		t.Parallel()
		ex, _ := newWebsocketTestExchange(t)
		btc := &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}
		eth := &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "eth_usd"}}
		require.NoError(t, ex.Subscribe(subscription.List{btc, eth}), "Subscribe must not error for two indexes")
		assert.ElementsMatch(t, []string{"deribit_price_index.btc_usd", "deribit_price_index.eth_usd"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "Subscribe should hold each index's channel")
		require.NoError(t, ex.Unsubscribe(subscription.List{heldSubscription(ex, "deribit_price_index.eth_usd")}), "Unsubscribe must not error")
		assert.Equal(t, []string{"deribit_price_index.btc_usd"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "Unsubscribe should leave the other index's channel held")
	})

	t.Run("book channels requested together", func(t *testing.T) {
		t.Parallel()
		ex, venue := newWebsocketTestExchange(t)
		rawBook := plainBook()
		rawBook.Interval = kline.Raw
		rawBook.Authenticated = true
		tickerSub := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{perp}, Interval: kline.HundredMilliseconds}
		err := ex.Subscribe(subscription.List{groupedBook(), rawBook, tickerSub})
		assert.ErrorIs(t, err, subscription.ErrExclusiveSubscription, "Subscribe should refuse a public and a private book channel of one instrument")
		requests := venue.received("public/subscribe")
		require.Len(t, requests, 1, "Subscribe must still subscribe the other channels")
		assert.JSONEq(t, `{"channels":["ticker.BTC-PERPETUAL.100ms"]}`, string(requests[0].Params), "Subscribe should send neither book channel")
		assert.Empty(t, venue.received("private/subscribe"), "Subscribe should send nothing for the refused raw book")
		assert.Equal(t, []string{"ticker.BTC-PERPETUAL.100ms"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "Subscribe should hold neither book channel")
	})

	t.Run("book channel held", func(t *testing.T) {
		t.Parallel()
		ex, venue := newWebsocketTestExchange(t)
		require.NoError(t, ex.Subscribe(subscription.List{plainBook()}), "Subscribe must not error for a plain book")
		err := ex.Subscribe(subscription.List{groupedBook()})
		assert.ErrorIs(t, err, subscription.ErrExclusiveSubscription, "Subscribe should refuse a grouped book while the instrument's plain book is held")
		assert.Len(t, venue.received("public/subscribe"), 1, "Subscribe should send nothing for a refused book channel")
		assert.Equal(t, []string{"book.BTC-PERPETUAL.100ms"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "Subscribe should hold only the book channel held before")
	})

	t.Run("book channels requested at once", func(t *testing.T) {
		t.Parallel()
		ex, venue := newWebsocketTestExchange(t)
		venue.delayNext("public/subscribe", 200*time.Millisecond)
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for i, book := range []*subscription.Subscription{plainBook(), groupedBook()} {
			wg.Go(func() { errs[i] = ex.Subscribe(subscription.List{book}) })
		}
		wg.Wait()
		refused := slices.IndexFunc(errs, func(err error) bool { return errors.Is(err, subscription.ErrExclusiveSubscription) })
		require.NotEqual(t, -1, refused, "Subscribe must refuse one of two book channels requested at once")
		assert.NoError(t, errs[1-refused], "Subscribe should subscribe the other book channel")
		assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "Subscribe should hold one book channel of the instrument")
		assert.Len(t, venue.received("public/subscribe"), 1, "Subscribe should send only the book channel it holds")
	})

	t.Run("unsubscribe during a subscribe", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			access string
			sub    *subscription.Subscription
		}{
			{"public", &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}},
			{"private", &subscription.Subscription{Channel: ChannelUserLiquidation, Authenticated: true}},
		} {
			t.Run(tc.access, func(t *testing.T) {
				t.Parallel()
				ex, venue := newWebsocketTestExchange(t)
				method := tc.access + "/subscribe"
				venue.delayNext(method, 150*time.Millisecond)
				subscribed := make(chan error, 1)
				go func() { subscribed <- ex.Subscribe(subscription.List{tc.sub}) }()
				require.Eventually(t, func() bool { return len(venue.received(method)) == 1 }, 5*time.Second, time.Millisecond, "Subscribe must send its request")
				assert.NoError(t, ex.Unsubscribe(subscription.List{tc.sub}), "Unsubscribe should wait for the subscribe under way, then remove its subscription")
				assert.NoError(t, <-subscribed, "Subscribe should not error")
				assert.Empty(t, ex.Websocket.GetSubscriptions(), "Unsubscribe should leave no subscription held")
			})
		}
	})
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	skipLiveWithoutCredentials(t)
	const channel = "user.liquidation"
	private := &subscription.Subscription{Channel: ChannelUserLiquidation, Authenticated: true}
	require.NoError(t, ex.Subscribe(subscription.List{private}), "Subscribe must not error for a private channel")
	held := heldSubscription(ex, channel)
	require.NotNil(t, held, "Subscribe must store the private subscription under its qualified channel")
	require.NoError(t, ex.Unsubscribe(subscription.List{held}), "Unsubscribe must not error for a private channel")
	assert.Nil(t, heldSubscription(ex, channel), "Unsubscribe should remove the private subscription")
	if mockTests {
		assert.Len(t, venue.received("private/subscribe"), 1, "Subscribe should subscribe a private channel through private/subscribe")
		assert.Len(t, venue.received("private/unsubscribe"), 1, "Unsubscribe should unsubscribe a private channel through private/unsubscribe")
	}
}

func TestHandleSubscription(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	require.NoError(t, ex.handleSubscription(t.Context(), true, false, nil), "handleSubscription must not error without subscriptions")

	disconnected := newTestExchange(t)
	indexSubs, err := disconnected.expandSubscriptions(subscription.List{{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}})
	require.NoError(t, err, "expandSubscriptions must not error")
	err = disconnected.handleSubscription(t.Context(), true, false, indexSubs)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "handleSubscription should return the request failure")

	if !mockTests {
		return
	}
	subs, err := ex.expandSubscriptions(subscription.List{
		{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}},
		{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": unacknowledgedChannel}},
	})
	require.NoError(t, err, "expandSubscriptions must not error")
	err = ex.handleSubscription(t.Context(), true, false, subs)
	assert.ErrorIs(t, err, errUnacknowledgedSubscription, "handleSubscription should report a channel Deribit does not acknowledge")
	held := ex.Websocket.GetSubscriptions()
	require.Len(t, held, 1, "handleSubscription must store only the acknowledged subscription")
	assert.Equal(t, "deribit_price_index.btc_usd", held[0].QualifiedChannel, "handleSubscription should store the acknowledged channel")
	require.NoError(t, ex.handleSubscription(t.Context(), false, false, held), "handleSubscription must not error unsubscribing")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "handleSubscription should remove the unsubscribed channel")

	_, err = newTestExchange(t).WebsocketAPI().Subscribe(t.Context(), []string{"deribit_price_index.btc_usd"})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Subscribe should error without a connection")
}

func TestRefuseBookConflicts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		held      []string
		requested []string
		allowed   []string
		refused   int
	}{
		{name: "no book channel", requested: []string{"ticker.BTC-PERPETUAL.100ms", "deribit_price_index.btc_usd"}, allowed: []string{"deribit_price_index.btc_usd", "ticker.BTC-PERPETUAL.100ms"}},
		{name: "one book channel twice", requested: []string{"book.BTC-PERPETUAL.100ms", "book.BTC-PERPETUAL.100ms"}, allowed: []string{"book.BTC-PERPETUAL.100ms", "book.BTC-PERPETUAL.100ms"}},
		{name: "book channels of two instruments", requested: []string{"book.BTC-PERPETUAL.100ms", "book.ETH-PERPETUAL.none.10.100ms"}, allowed: []string{"book.BTC-PERPETUAL.100ms", "book.ETH-PERPETUAL.none.10.100ms"}},
		{name: "plain and grouped requested", requested: []string{"book.BTC-PERPETUAL.100ms", "book.BTC-PERPETUAL.none.10.100ms", "book.ETH-PERPETUAL.100ms", "ticker.BTC-PERPETUAL.100ms"}, allowed: []string{"book.ETH-PERPETUAL.100ms", "ticker.BTC-PERPETUAL.100ms"}, refused: 2},
		{name: "two intervals requested", requested: []string{"book.BTC-PERPETUAL.100ms", "book.BTC-PERPETUAL.agg2"}, allowed: []string{}, refused: 2},
		{name: "two groupings requested", requested: []string{"book.BTC-PERPETUAL.none.10.100ms", "book.BTC-PERPETUAL.5.1.100ms"}, allowed: []string{}, refused: 2},
		{name: "grouped while plain held", held: []string{"book.BTC-PERPETUAL.100ms"}, requested: []string{"book.BTC-PERPETUAL.none.10.100ms", "book.ETH-PERPETUAL.none.10.100ms"}, allowed: []string{"book.ETH-PERPETUAL.none.10.100ms"}, refused: 1},
		{name: "plain while grouped held", held: []string{"book.BTC-PERPETUAL.none.10.100ms"}, requested: []string{"book.BTC-PERPETUAL.raw"}, allowed: []string{}, refused: 1},
		{name: "held book channel requested", held: []string{"book.BTC-PERPETUAL.100ms"}, requested: []string{"book.BTC-PERPETUAL.100ms"}, allowed: []string{"book.BTC-PERPETUAL.100ms"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			for _, channel := range tc.held {
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.Conn, &subscription.Subscription{QualifiedChannel: channel}), "AddSuccessfulSubscriptions must not error")
			}
			requested := make(subscription.List, len(tc.requested))
			for i, channel := range tc.requested {
				requested[i] = &subscription.Subscription{QualifiedChannel: channel}
			}
			allowed, err := ex.refuseBookConflicts(requested)
			assert.Equal(t, tc.allowed, allowed.QualifiedChannels(), "refuseBookConflicts should leave the conflicting book channels out")
			if tc.refused == 0 {
				assert.NoError(t, err, "refuseBookConflicts should not error without a conflict")
				return
			}
			assert.ErrorIs(t, err, subscription.ErrExclusiveSubscription, "refuseBookConflicts should refuse a second book channel of an instrument")
			refusals := 1
			var joined interface{ Unwrap() []error }
			if errors.As(err, &joined) {
				refusals = len(joined.Unwrap())
			}
			assert.Equal(t, tc.refused, refusals, "refuseBookConflicts should return an error for each channel left out")
		})
	}
}

func TestBookInstrument(t *testing.T) {
	t.Parallel()
	for channel, exp := range map[string]string{
		"book.BTC-PERPETUAL.100ms":         "BTC-PERPETUAL",
		"book.BTC-PERPETUAL.none.10.100ms": "BTC-PERPETUAL",
		"book.ETH_USDC.raw":                "ETH_USDC",
		"book.BTC-PERPETUAL":               "",
		"bookkeeping.BTC-PERPETUAL.100ms":  "",
		"ticker.BTC-PERPETUAL.100ms":       "",
		"book":                             "",
	} {
		instrument, ok := bookInstrument(channel)
		assert.Equalf(t, exp != "", ok, "bookInstrument should report whether %s is a qualified book channel", channel)
		assert.Equalf(t, exp, instrument, "bookInstrument should return %s's instrument", channel)
	}
}

func TestWebsocketBlockRFQQuotesUnmarshalJSON(t *testing.T) {
	t.Parallel()
	quote := `{"block_rfq_quote_id":1301,"block_rfq_id":724,"quote_state":"open","price":10,"amount":25,"direction":"buy"}`
	exp := WebsocketBlockRFQQuotes{{BlockRFQQuoteID: 1301, BlockRFQID: 724, QuoteState: "open", Price: 10, Amount: 25, Direction: DirectionBuy}}
	for _, data := range []string{`[` + quote + `]`, quote} {
		var quotes WebsocketBlockRFQQuotes
		require.NoErrorf(t, json.Unmarshal([]byte(data), &quotes), "Unmarshal must not error for %s", data)
		assert.Equalf(t, exp, quotes, "UnmarshalJSON should decode %s", data)
	}
	var quotes WebsocketBlockRFQQuotes
	require.NoError(t, json.Unmarshal([]byte(`null`), &quotes), "Unmarshal must not error for null")
	assert.Empty(t, quotes, "UnmarshalJSON should decode null as no quotes")
	assert.Error(t, json.Unmarshal([]byte(`"quote"`), &quotes), "UnmarshalJSON should reject a string")
}

func TestWebsocketInstrumentCreationUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var created WebsocketInstrumentCreation
	require.NoError(t, json.Unmarshal([]byte(`{"instrument_name":"BTC-PERPETUAL","kind":"future","expiration_timestamp":32503708800000,"timestamp":1671696002000}`), &created), "Unmarshal must not error")
	exp := WebsocketInstrumentCreation{
		InstrumentName:      "BTC-PERPETUAL",
		Kind:                KindFuture,
		ExpirationTimestamp: types.Time(time.UnixMilli(32503708800000)),
		Timestamp:           types.Time(time.UnixMilli(1671696002000)),
	}
	assert.Equal(t, exp, created, "UnmarshalJSON should decode the instrument and the notification's timestamp")
	assert.Error(t, json.Unmarshal([]byte(`{"instrument_name":1}`), &created), "UnmarshalJSON should reject a malformed instrument")
	assert.Error(t, json.Unmarshal([]byte(`{"timestamp":"soon"}`), &created), "UnmarshalJSON should reject a malformed timestamp")
}
