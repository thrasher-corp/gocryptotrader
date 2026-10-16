package deribit

import (
	"slices"
	"sync"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// websocketVenue records the requests the mock of Deribit's websocket receives; live tests share Deribit's websocket,
// and get a nil venue, which records nothing
type websocketVenue struct {
	mu       sync.Mutex
	requests []websocketVenueRequest
}

// websocketVenueRequest is a JSON-RPC request as Deribit's websocket receives it
type websocketVenueRequest struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// received returns the requests received for a method, in arrival order
func (v *websocketVenue) received(method string) []websocketVenueRequest {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var requests []websocketVenueRequest
	for i := range v.requests {
		if v.requests[i].Method == method {
			requests = append(requests, v.requests[i])
		}
	}
	return requests
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
			channel: "quote.BTC-PERPETUAL",
			data:    `{"timestamp":1792107433899,"instrument_name":"BTC-PERPETUAL","best_ask_price":82141.5,"best_bid_price":82141.0,"best_ask_amount":59940.0,"best_bid_amount":6.69e4}`,
			exp:     &WebsocketQuote{Timestamp: types.Time(time.UnixMilli(1792107433899)), InstrumentName: "BTC-PERPETUAL", BestBidPrice: 82141, BestBidAmount: 66900, BestAskPrice: 82141.5, BestAskAmount: 59940},
		},
		{
			channel: "deribit_price_index.btc_usd",
			data:    `{"timestamp":1792107433668,"price":82119.87,"index_name":"btc_usd"}`,
			exp:     &WebsocketPriceIndex{Timestamp: types.Time(time.UnixMilli(1792107433668)), IndexName: "btc_usd", Price: 82119.87},
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
			channel: "user.portfolio.btc",
			data:    `{"delta_total_map":{"btc_usd":31.594397699},"margin_balance":302.62675921,"session_upl":0.05341555,"options_gamma_map":{"btc_usd":1e-05},"options_value":-0.0079,"available_withdrawal_funds":301.35426172,"maintenance_margin":0.8854841,"available_funds":301.38036328,"balance":302.60065765,"equity":302.6188592,"fee_balance":0,"currency":"BTC","portfolio_margining_enabled":false,"margin_model":"segregated_sm","initial_margin":1.24639592,"spot_reserve":0,"delta_total":31.602298,"additional_reserve":0}`,
			exp: &AccountSummary{
				Currency:                    currency.BTC,
				Balance:                     302.60065765,
				Equity:                      302.6188592,
				MarginBalance:               302.62675921,
				AvailableFunds:              301.38036328,
				AvailableWithdrawalFunds:    301.35426172,
				InitialMargin:               1.24639592,
				MaintenanceMargin:           0.8854841,
				SessionUnrealisedProfitLoss: 0.05341555,
				OptionsValue:                -0.0079,
				OptionsGammaByIndex:         map[string]float64{"btc_usd": 1e-05},
				DeltaTotal:                  31.602298,
				DeltaTotalByIndex:           map[string]float64{"btc_usd": 31.594397699},
				MarginModel:                 "segregated_sm",
			},
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
		{sub: &subscription.Subscription{Channel: ChannelTrades, Interval: kline.HundredMilliseconds}, exp: []string{"trades.any.any.100ms"}},
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
		{sub: &subscription.Subscription{Channel: ChannelTrades, Interval: kline.OneMin}, err: kline.ErrUnsupportedInterval},
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

func TestSubscribe(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	public := &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "eth_usd"}}
	require.NoError(t, ex.Subscribe(subscription.List{public}), "Subscribe must not error")
	held := ex.Websocket.GetSubscriptions()
	require.Len(t, held, 1, "Subscribe must store the acknowledged subscription")
	assert.Equal(t, "deribit_price_index.eth_usd", held[0].QualifiedChannel, "Subscribe should store the qualified channel")
	if mockTests {
		requests := venue.received("public/subscribe")
		require.Len(t, requests, 1, "Subscribe must send a public/subscribe request")
		assert.JSONEq(t, `{"channels":["deribit_price_index.eth_usd"]}`, string(requests[0].Params), "Subscribe should send the qualified channel")
	}
	require.NoError(t, ex.Unsubscribe(held), "Unsubscribe must not error")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "Unsubscribe should remove the acknowledged subscription")

	assert.ErrorIs(t, ex.Subscribe(subscription.List{{Channel: "bogus"}}), subscription.ErrNotSupported, "Subscribe should reject an unknown channel")
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	skipLiveWithoutCredentials(t)
	private := &subscription.Subscription{Channel: ChannelUserLock, Authenticated: true}
	require.NoError(t, ex.Subscribe(subscription.List{private}), "Subscribe must not error for a private channel")
	held := ex.Websocket.GetSubscriptions()
	require.Len(t, held, 1, "Subscribe must store the private subscription")
	require.NoError(t, ex.Unsubscribe(held), "Unsubscribe must not error for a private channel")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "Unsubscribe should remove the private subscription")
	if mockTests {
		assert.Len(t, venue.received("private/subscribe"), 1, "Subscribe should subscribe a private channel through private/subscribe")
		assert.Len(t, venue.received("private/unsubscribe"), 1, "Unsubscribe should unsubscribe a private channel through private/unsubscribe")
	}
}

func TestHandleSubscription(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	require.NoError(t, ex.handleSubscription(t.Context(), true, false, nil), "handleSubscription must not error without subscriptions")
	err := ex.handleSubscription(t.Context(), true, false, subscription.List{{Channel: ChannelPriceIndex}})
	assert.ErrorIs(t, err, errChannelParameter, "handleSubscription should return a qualification failure")

	if !mockTests {
		return
	}
	subs := subscription.List{
		{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}},
		{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": unacknowledgedChannel}},
	}
	err = ex.handleSubscription(t.Context(), true, false, subs)
	assert.ErrorIs(t, err, errUnacknowledgedSubscription, "handleSubscription should report a channel Deribit does not acknowledge")
	held := ex.Websocket.GetSubscriptions()
	require.Len(t, held, 1, "handleSubscription must store only the acknowledged subscription")
	assert.Equal(t, "deribit_price_index.btc_usd", held[0].QualifiedChannel, "handleSubscription should store the acknowledged channel")

	_, err = newTestExchange(t).WebsocketAPI().Subscribe(t.Context(), []string{"deribit_price_index.btc_usd"})
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Subscribe must error without a connection")
	disconnected := newTestExchange(t)
	err = disconnected.handleSubscription(t.Context(), true, false, subscription.List{{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "handleSubscription should return the request failure")
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
