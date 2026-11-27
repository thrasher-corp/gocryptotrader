package deribit

import (
	"errors"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestWebsocketHandleOrders(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	exp := []order.Detail{{
		Exchange:        ex.Name,
		OrderID:         "5",
		Pair:            currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter),
		AssetType:       asset.Futures,
		Side:            order.Buy,
		Type:            order.Market,
		Status:          order.Open,
		TimeInForce:     order.GoodTillCancel,
		Price:           10502.52,
		Amount:          200,
		RemainingAmount: 200,
		Date:            time.UnixMilli(1581507423789),
		LastUpdated:     time.UnixMilli(1581507423789),
	}}
	require.NoError(t, ex.wsHandleOrders(t.Context(), "user.orders.BTC-PERPETUAL.raw", notification("user.orders.BTC-PERPETUAL.raw", testWebsocketOrder)), "wsHandleOrders must not error for a raw channel's order")
	assert.Equal(t, exp, relayed(t, ex), "wsHandleOrders should relay a raw channel's order")
	require.NoError(t, ex.wsHandleOrders(t.Context(), "user.orders.future.BTC.100ms", notification("user.orders.future.BTC.100ms", "["+testWebsocketOrder+"]")), "wsHandleOrders must not error for an interval channel's orders")
	assert.Equal(t, exp, relayed(t, ex), "wsHandleOrders should relay an interval channel's orders")

	err := ex.wsHandleOrders(t.Context(), "user.orders.future.any.100ms", notification("user.orders.future.any.100ms", `[{"instrument_name":"BTC","direction":"buy"},`+testWebsocketOrder+`]`))
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "wsHandleOrders should report an order it cannot convert")
	assert.Equal(t, exp, relayed(t, ex), "wsHandleOrders should still relay the orders it converts")
	err = ex.wsHandleOrders(t.Context(), "user.orders.future.any.100ms", notification("user.orders.future.any.100ms", `[{"instrument_name":"BTC"}]`))
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "wsHandleOrders should report an order it cannot convert")
	assertNothingRelayed(t, ex)
	assert.ErrorIs(t, ex.wsHandleOrders(t.Context(), "user.orders.BTC-PERPETUAL.raw", notification("user.orders.BTC-PERPETUAL.raw", `[]`)), errMalformedNotification, "wsHandleOrders should reject a raw channel's list")
	assert.ErrorIs(t, ex.wsHandleOrders(t.Context(), "user.orders.future.BTC.100ms", notification("user.orders.future.BTC.100ms", `{}`)), errMalformedNotification, "wsHandleOrders should reject an interval channel's single order")
}

func TestRelayOrders(t *testing.T) {
	t.Parallel()
	var placed OrderResponse
	require.NoError(t, json.Unmarshal([]byte(testWebsocketOrder), &placed), "Unmarshal must not error")
	second := placed
	second.OrderID, second.Amount = "7", 20
	// A sell option order with a lower ID and a price of its own, so a relay that sorts, drops an asset, keeps one order
	// an asset or copies one order's fields to another fails
	option := placed
	option.OrderID, option.InstrumentName, option.Direction, option.Price, option.Amount = "4", "BTC-25DEC26-104000-C", DirectionSell, OrderPrice{Value: 0.0525}, 20
	unconvertible := OrderResponse{OrderID: "6", InstrumentName: "BTC", Direction: DirectionBuy}
	for _, tc := range []struct {
		name    string
		orders  []OrderResponse
		relayed []OrderResponse
		err     error
	}{
		{name: "an order converted", orders: []OrderResponse{placed}, relayed: []OrderResponse{placed}},
		{name: "two orders converted", orders: []OrderResponse{placed, second}, relayed: []OrderResponse{placed, second}},
		{name: "three orders of two assets converted", orders: []OrderResponse{placed, option, second}, relayed: []OrderResponse{placed, option, second}},
		{name: "an order it cannot convert", orders: []OrderResponse{unconvertible, placed}, relayed: []OrderResponse{placed}, err: errUnsupportedInstrumentFormat},
		{name: "an order it cannot convert between two", orders: []OrderResponse{placed, unconvertible, second}, relayed: []OrderResponse{placed, second}, err: errUnsupportedInstrumentFormat},
		{name: "no order converted", orders: []OrderResponse{unconvertible}, err: errUnsupportedInstrumentFormat},
		{name: "no orders"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			assert.ErrorIs(t, ex.relayOrders(t.Context(), tc.orders), tc.err, "relayOrders should return the errors of the orders it cannot convert")
			if len(tc.relayed) > 0 {
				exp := make([]order.Detail, 0, len(tc.relayed))
				for i := range tc.relayed {
					detail, err := ex.orderDetail(&tc.relayed[i])
					require.NoError(t, err, "orderDetail must not error")
					exp = append(exp, *detail)
				}
				assert.Equal(t, exp, relayed(t, ex), "relayOrders should relay every order it converts, in order, at once")
			}
			assertNothingRelayed(t, ex)
		})
	}

	t.Run("data handler full", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		var refused error
		ex.Websocket.DataHandler, refused = fullRelay(t)
		assert.ErrorIs(t, ex.relayOrders(t.Context(), []OrderResponse{placed}), refused, "relayOrders should return the data handler's failure")
	})
}

// fullRelay returns a relay whose buffer is full, so every send to it fails, and the error such a send wraps
func fullRelay(t *testing.T) (*stream.Relay, error) {
	t.Helper()
	full := stream.NewRelay(1)
	require.NoError(t, full.Send(t.Context(), "filler"), "Send must fill the relay")
	refused := errors.Unwrap(full.Send(t.Context(), "refused"))
	require.Error(t, refused, "Send must refuse a message to a full relay")
	return full, refused
}

// testWebsocketUserChangesTwice is testWebsocketUserChanges with a second order, filled in a second trade
var testWebsocketUserChangesTwice = strings.Replace(strings.Replace(testWebsocketUserChanges,
	`"direction":"sell","amount":10}],"positions"`,
	`"direction":"sell","amount":10},{"trade_seq":866639,"trade_id":"1430915","timestamp":1605780344032,"tick_direction":1,"state":"filled","reduce_only":false,"price":17391,"post_only":false,"order_type":"market","order_id":"3398017","mark_price":17391,"liquidity":"T","instrument_name":"BTC-PERPETUAL","index_price":17501.88,"fee_currency":"BTC","fee":3.2e-07,"direction":"sell","amount":20}],"positions"`, 1),
	`"average_price":17391,"api":false,"amount":10}]`,
	`"average_price":17391,"api":false,"amount":10},{"web":true,"time_in_force":"good_til_cancelled","replaced":false,"reduce_only":false,"price":15665.5,"post_only":false,"order_type":"market","order_state":"filled","order_id":"3398017","last_update_timestamp":1605780344032,"label":"","is_rebalance":false,"is_liquidation":false,"instrument_name":"BTC-PERPETUAL","filled_amount":20,"direction":"sell","creation_timestamp":1605780344032,"average_price":17391,"api":false,"amount":20}]`, 1)

// testWebsocketUserChangesWithOption is testWebsocketUserChangesTwice with a third order, a buy option order with a lower
// ID and prices of its own, filled in a third trade
var testWebsocketUserChangesWithOption = strings.Replace(strings.Replace(testWebsocketUserChangesTwice,
	`"direction":"sell","amount":20}],"positions"`,
	`"direction":"sell","amount":20},{"trade_seq":1201,"trade_id":"1430913","timestamp":1605780344032,"tick_direction":1,"state":"filled","reduce_only":false,"price":0.0525,"post_only":false,"order_type":"market","order_id":"3398015","mark_price":0.0525,"liquidity":"T","instrument_name":"BTC-25DEC26-104000-C","index_price":17501.88,"fee_currency":"BTC","fee":3.2e-07,"direction":"buy","amount":20}],"positions"`, 1),
	`"average_price":17391,"api":false,"amount":20}]`,
	`"average_price":17391,"api":false,"amount":20},{"web":true,"time_in_force":"good_til_cancelled","replaced":false,"reduce_only":false,"price":0.05,"post_only":false,"order_type":"market","order_state":"filled","order_id":"3398015","last_update_timestamp":1605780344032,"label":"","is_rebalance":false,"is_liquidation":false,"instrument_name":"BTC-25DEC26-104000-C","filled_amount":20,"direction":"buy","creation_timestamp":1605780344032,"average_price":0.0525,"api":false,"amount":20}]`, 1)

// testUserChangesAddedOrders returns the orders testWebsocketUserChangesWithOption adds after the documented one, as
// order details
func testUserChangesAddedOrders(exchangeName string) []order.Detail {
	second := testUserChangesOrders(exchangeName)[0]
	second.OrderID, second.Amount, second.ExecutedAmount = "3398017", 20, 20
	return []order.Detail{second, {
		Exchange:             exchangeName,
		OrderID:              "3398015",
		Pair:                 currency.NewPairWithDelimiter("BTC", "25DEC26-104000-C", currency.DashDelimiter),
		AssetType:            asset.Options,
		Side:                 order.Buy,
		Type:                 order.Market,
		Status:               order.Filled,
		TimeInForce:          order.GoodTillCancel,
		Price:                0.05,
		Amount:               20,
		ExecutedAmount:       20,
		AverageExecutedPrice: 0.0525,
		Date:                 time.UnixMilli(1605780344032),
		LastUpdated:          time.UnixMilli(1605780344032),
	}}
}

// testUserChangesAddedFills returns the trades testWebsocketUserChangesWithOption adds after the documented one, as fills
func testUserChangesAddedFills(exchangeName string) []fill.Data {
	second := testUserChangesFills(exchangeName)[0]
	second.ID, second.TradeID, second.OrderID, second.Amount = "1430915", "1430915", "3398017", 20
	return []fill.Data{second, {
		ID:           "1430913",
		TradeID:      "1430913",
		Timestamp:    time.UnixMilli(1605780344032),
		Exchange:     exchangeName,
		AssetType:    asset.Options,
		CurrencyPair: currency.NewPairWithDelimiter("BTC", "25DEC26-104000-C", currency.DashDelimiter),
		Side:         order.Buy,
		OrderID:      "3398015",
		Price:        0.0525,
		Amount:       20,
	}}
}

func TestWebsocketHandleUserChanges(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		data       string
		fillsFeed  bool
		err        error
		expOrders  bool
		expFills   bool
		expChanges bool
		// added counts the orders and trades the notification adds after the documented ones
		added int
	}{
		{name: "fills feed on", data: testWebsocketUserChanges, fillsFeed: true, expOrders: true, expFills: true, expChanges: true},
		{name: "fills feed off", data: testWebsocketUserChanges, expOrders: true, expChanges: true},
		{
			name:       "an order it cannot convert",
			data:       strings.Replace(testWebsocketUserChanges, `"orders":[`, `"orders":[{"order_id":"6","instrument_name":"BTC","direction":"sell"},`, 1),
			fillsFeed:  true,
			err:        errUnsupportedInstrumentFormat,
			expOrders:  true,
			expFills:   true,
			expChanges: true,
		},
		{
			name:       "a trade it cannot convert",
			data:       strings.Replace(testWebsocketUserChanges, `"trades":[`, `"trades":[{"trade_id":"7","instrument_name":"BTC","direction":"sell"},`, 1),
			fillsFeed:  true,
			err:        errUnsupportedInstrumentFormat,
			expOrders:  true,
			expFills:   true,
			expChanges: true,
		},
		{name: "two orders and two trades", data: testWebsocketUserChangesTwice, fillsFeed: true, expOrders: true, expFills: true, expChanges: true, added: 1},
		{name: "three orders and three trades of two assets", data: testWebsocketUserChangesWithOption, fillsFeed: true, expOrders: true, expFills: true, expChanges: true, added: 2},
		{name: "no orders or trades", data: `{"instrument_name":"BTC-PERPETUAL","orders":[],"trades":[],"positions":[]}`, fillsFeed: true, expChanges: true},
		{name: "malformed", data: `[]`, fillsFeed: true, err: errMalformedNotification},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.Features.Enabled.FillsFeed = tc.fillsFeed
			ex.Websocket.Fills.Setup(tc.fillsFeed, ex.Websocket.DataHandler)
			err := ex.wsHandleUserChanges(t.Context(), notification("user.changes.BTC-PERPETUAL.raw", tc.data))
			assert.ErrorIs(t, err, tc.err, "wsHandleUserChanges should return the errors of what it cannot convert or decode")
			if tc.expOrders {
				exp := append(testUserChangesOrders(ex.Name), testUserChangesAddedOrders(ex.Name)[:tc.added]...)
				assert.Equal(t, exp, relayed(t, ex), "wsHandleUserChanges should first relay the orders it converts, in order")
			}
			if tc.expFills {
				exp := append(testUserChangesFills(ex.Name), testUserChangesAddedFills(ex.Name)[:tc.added]...)
				assert.Equal(t, exp, relayed(t, ex), "wsHandleUserChanges should then pass the trades it converts to the fills feed, in order")
			}
			if tc.expChanges {
				exp := new(WebsocketUserChanges)
				require.NoError(t, json.Unmarshal([]byte(tc.data), exp), "Unmarshal must not error")
				assert.Equal(t, exp, relayed(t, ex), "wsHandleUserChanges should relay the notification last")
			}
			assertNothingRelayed(t, ex)
		})
	}

	t.Run("data handler full", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		var refused error
		ex.Websocket.DataHandler, refused = fullRelay(t)
		err := ex.wsHandleUserChanges(t.Context(), notification("user.changes.BTC-PERPETUAL.raw", `{"instrument_name":"BTC-PERPETUAL","orders":[],"trades":[],"positions":[]}`))
		assert.ErrorIs(t, err, refused, "wsHandleUserChanges should return the data handler's failure to relay the notification")
	})
}

// newAccountTestExchange returns a test exchange with placeholder credentials, which its account store saves balances
// under, and the main account's balances saved under each of the sub-accounts given
func newAccountTestExchange(t *testing.T, saved ...*accounts.SubAccount) *Exchange {
	t.Helper()
	ex := newTestExchange(t)
	setTestCredentials(ex, &accounts.Credentials{Key: testClientID, Secret: testClientSecret})
	if len(saved) > 0 {
		require.NoError(t, ex.Accounts.Save(t.Context(), saved, true), "Save must not error")
	}
	return ex
}

// savedBalance returns a balance the account store holds for the main account, without the arrival time it is stamped
// with
func savedBalance(t *testing.T, ex *Exchange, a asset.Item, c currency.Code) (accounts.Balance, error) {
	t.Helper()
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	balance, err := ex.Accounts.GetBalance("", creds, a, c)
	balance.UpdatedAt = time.Time{}
	return balance, err
}

func TestWebsocketHandlePortfolio(t *testing.T) { //nolint:tparallel // Its failed publication case starts the process-wide dispatcher, so it runs serially while the other cases wait
	// Deribit's documented user.portfolio example
	const channel = "user.portfolio.btc"
	const btcPortfolio = `{"delta_total_map":{"btc_usd":31.594397699},"margin_balance":302.62675921,"session_upl":0.05341555,"options_gamma_map":{"btc_usd":1e-05},"options_value":-0.0079,"available_withdrawal_funds":301.35426172,"maintenance_margin":0.8854841,"available_funds":301.38036328,"balance":302.60065765,"equity":302.6188592,"fee_balance":0,"currency":"BTC","portfolio_margining_enabled":false,"margin_model":"segregated_sm","initial_margin":1.24639592,"spot_reserve":0,"delta_total":31.602298,"additional_reserve":0}`
	cashBalance, withdrawable := 302.60065765, 301.35426172
	expBalance := accounts.Balance{Currency: currency.BTC, Total: cashBalance, Hold: cashBalance - withdrawable, Free: withdrawable}

	t.Run("nothing saved", func(t *testing.T) {
		t.Parallel()
		ex := newAccountTestExchange(t)
		require.NoError(t, ex.wsHandleNotification(t.Context(), nil, channel, notification(channel, btcPortfolio)), "wsHandleNotification must not error for a portfolio")
		exp := &AccountSummary{
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
		}
		assert.Equal(t, exp, relayed(t, ex), "wsHandlePortfolio should relay the summary")
		balance, err := savedBalance(t, ex, asset.Spot, currency.BTC)
		require.NoError(t, err, "the balance must be saved under spot")
		assert.Equal(t, expBalance, balance, "wsHandlePortfolio should save the cash balance as the total, the withdrawable funds as free and the rest as held")
		saved, err := ex.Accounts.SubAccounts(nil, asset.All)
		require.NoError(t, err, "SubAccounts must not error")
		assert.Len(t, saved, 1, "wsHandlePortfolio should save the balance under spot alone")
	})

	t.Run("saved under other assets", func(t *testing.T) {
		t.Parallel()
		other := accounts.Balance{Currency: currency.BTC, Total: 1, Free: 1}
		ex := newAccountTestExchange(
			t,
			&accounts.SubAccount{AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.BTC: other}},
			&accounts.SubAccount{AssetType: asset.Options, Balances: accounts.CurrencyBalances{currency.BTC: other}},
		)
		require.NoError(t, ex.wsHandlePortfolio(t.Context(), notification(channel, btcPortfolio)), "wsHandlePortfolio must not error")
		assert.IsType(t, &AccountSummary{}, relayed(t, ex), "wsHandlePortfolio should relay the summary")
		balance, err := savedBalance(t, ex, asset.Spot, currency.BTC)
		require.NoError(t, err, "the balance must be saved under spot")
		assert.Equal(t, expBalance, balance, "wsHandlePortfolio should save the balance under spot, where UpdateAccountBalances saves it")
		for _, a := range []asset.Item{asset.Futures, asset.Options} {
			balance, err := savedBalance(t, ex, a, currency.BTC)
			require.NoErrorf(t, err, "the balance saved under %s must still be saved", a)
			assert.Equalf(t, other, balance, "wsHandlePortfolio should leave the balance saved under %s as it is", a)
		}
	})

	t.Run("saved under futures alone", func(t *testing.T) {
		t.Parallel()
		other := accounts.Balance{Currency: currency.BTC, Total: 1, Free: 1}
		ex := newAccountTestExchange(t, &accounts.SubAccount{AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.BTC: other}})
		require.NoError(t, ex.wsHandlePortfolio(t.Context(), notification(channel, btcPortfolio)), "wsHandlePortfolio must not error")
		assert.IsType(t, &AccountSummary{}, relayed(t, ex), "wsHandlePortfolio should relay the summary")
		balance, err := savedBalance(t, ex, asset.Spot, currency.BTC)
		require.NoError(t, err, "the balance must be saved under spot")
		assert.Equal(t, expBalance, balance, "wsHandlePortfolio should save the balance under spot, where UpdateAccountBalances saves it")
		balance, err = savedBalance(t, ex, asset.Futures, currency.BTC)
		require.NoError(t, err, "the balance saved under futures must still be saved")
		assert.Equal(t, other, balance, "wsHandlePortfolio should leave the balance saved under futures as it is")
	})

	t.Run("after a balance update for another asset", func(t *testing.T) {
		t.Parallel()
		ex := newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"summaries":[{"currency":"BTC","balance":1,"available_withdrawal_funds":1}]}}`)
		_, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		require.NoError(t, ex.wsHandlePortfolio(t.Context(), notification(channel, btcPortfolio)), "wsHandlePortfolio must not error")
		saved, err := ex.Accounts.SubAccounts(nil, asset.All)
		require.NoError(t, err, "SubAccounts must not error")
		require.Len(t, saved, 1, "the account store must hold the balances once")
		assert.Equal(t, asset.Spot, saved[0].AssetType, "the account store should hold the balances under spot")
		totals, err := ex.Accounts.CurrencyBalances(nil, asset.All)
		require.NoError(t, err, "CurrencyBalances must not error")
		assert.Equal(t, cashBalance, totals[currency.BTC].Total, "the account store should total the updated balance once")
	})

	t.Run("negative cash balance", func(t *testing.T) {
		t.Parallel()
		ex := newAccountTestExchange(t)
		require.NoError(t, ex.wsHandlePortfolio(t.Context(), notification(channel, `{"currency":"BTC","balance":-2.5,"available_withdrawal_funds":0}`)), "wsHandlePortfolio must not error")
		balance, err := savedBalance(t, ex, asset.Spot, currency.BTC)
		require.NoError(t, err, "the balance must be saved under spot")
		assert.Equal(t, accounts.Balance{Currency: currency.BTC, Total: -2.5}, balance, "wsHandlePortfolio should hold nothing of a negative cash balance")
	})

	t.Run("currency not saved", func(t *testing.T) {
		t.Parallel()
		eth := accounts.Balance{Currency: currency.ETH, Total: 100, Free: 100}
		ex := newAccountTestExchange(t, &accounts.SubAccount{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.ETH: eth}})
		require.NoError(t, ex.wsHandlePortfolio(t.Context(), notification(channel, btcPortfolio)), "wsHandlePortfolio must not error")
		assert.IsType(t, &AccountSummary{}, relayed(t, ex), "wsHandlePortfolio should relay the summary")
		balance, err := savedBalance(t, ex, asset.Spot, currency.BTC)
		require.NoError(t, err, "the balance must be saved under spot")
		assert.Equal(t, expBalance, balance, "wsHandlePortfolio should save a currency the account store has not held")
		balance, err = savedBalance(t, ex, asset.Spot, currency.ETH)
		require.NoError(t, err, "the other currency's balance must still be saved")
		assert.Equal(t, eth, balance, "wsHandlePortfolio should leave the other currencies' balances as they are")
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex := newAccountTestExchange(t)
		assert.ErrorIs(t, ex.wsHandlePortfolio(t.Context(), notification(channel, `[]`)), errMalformedNotification, "wsHandlePortfolio should reject a malformed portfolio")
		assert.ErrorIs(t, ex.wsHandlePortfolio(t.Context(), notification(channel, `{"balance":1}`)), currency.ErrCurrencyCodeEmpty, "wsHandlePortfolio should return a balance it cannot save")
		ex.SkipAuthCheck = false
		ex.SetCredentials(&accounts.Credentials{})
		assert.ErrorIs(t, ex.wsHandlePortfolio(t.Context(), notification(channel, btcPortfolio)), exchange.ErrCredentialsAreEmpty, "wsHandlePortfolio should need credentials to save the balance under")
		assertNothingRelayed(t, ex)
	})

	t.Run("save whose publication failed", func(t *testing.T) {
		// Serial and on one processor: the process-wide dispatcher it starts has one job slot, which other tests'
		// publications would share, and its one worker must not take a job off the filled slot before the notification's
		// balance is published
		procs := runtime.GOMAXPROCS(1)
		t.Cleanup(func() { runtime.GOMAXPROCS(procs) })
		require.False(t, dispatch.IsRunning(), "the dispatcher must not be running yet")
		require.NoError(t, dispatch.Start(1, 1), "dispatch.Start must not error")
		t.Cleanup(func() { assert.NoError(t, dispatch.Stop(), "dispatch.Stop should not error") })
		mux := dispatch.GetNewMux(nil)
		filler, err := mux.GetID()
		require.NoError(t, err, "GetID must not error")
		fillerPipe, err := mux.Subscribe(filler)
		require.NoError(t, err, "Subscribe must not error")
		t.Cleanup(func() { assert.NoError(t, fillerPipe.Release(), "Release should not error") })
		fillJobSlot := func() {
			t.Helper()
			var err error
			for i := 0; i < 1000 && err == nil; i++ {
				err = mux.Publish("filler", filler)
			}
			require.ErrorContains(t, err, "dispatcher jobs at limit", "the dispatcher's job slot must fill")
		}
		// drain waits until the dispatcher has taken every job published so far off its slot
		drain := func() {
			t.Helper()
			deadline := time.After(5 * time.Second)
			for mux.Publish("drained", filler) != nil {
				select {
				case <-deadline:
					require.FailNow(t, "the dispatcher must free its job slot")
				default:
					runtime.Gosched()
				}
			}
			for {
				select {
				case published := <-fillerPipe.Channel():
					if published == "drained" {
						return
					}
				case <-deadline:
					require.FailNow(t, "the dispatcher must relay the jobs published")
				}
			}
		}
		type update struct {
			subAccounts accounts.SubAccounts
			err         error
		}
		// startUpdate returns an exchange whose balance update has sent its request, which waits for the reply sent on the
		// channel returned, and the channel the update returns on
		startUpdate := func() (*Exchange, chan<- string, <-chan update) {
			t.Helper()
			arrived, replies := make(chan struct{}, 1), make(chan string, 1)
			ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case arrived <- struct{}{}:
				default:
				}
				select {
				case reply := <-replies:
					_, err := w.Write([]byte(reply))
					assert.NoError(t, err, "Writing the response should not error")
				case <-r.Context().Done():
				case <-time.After(5 * time.Second):
					assert.Fail(t, "the test should release the reply")
				}
			}))
			updated := make(chan update, 1)
			go func() {
				subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
				updated <- update{subAccounts, err}
			}()
			select {
			case <-arrived:
			case <-time.After(5 * time.Second):
				require.FailNow(t, "the update's request must arrive")
			}
			return ex, replies, updated
		}
		received := func(updated <-chan update) update {
			t.Helper()
			select {
			case u := <-updated:
				return u
			case <-time.After(5 * time.Second):
				require.FailNow(t, "UpdateAccountBalances must return")
				return update{}
			}
		}
		const reply = `{"jsonrpc":"2.0","result":{"summaries":[{"currency":"ETH","balance":2,"available_withdrawal_funds":2},{"currency":"BTC","balance":4,"available_withdrawal_funds":4}]}}`
		var (
			ex        *Exchange
			replies   chan<- string
			updated   <-chan update
			notifyErr error
		)
		// The worker runs once the test waits, so it can take a job off the filled slot before the notification is
		// published only if the runtime preempts the test; the case then starts again on another exchange
		for range 5 {
			ex, replies, updated = startUpdate()
			fillJobSlot()
			if notifyErr = ex.wsHandlePortfolio(t.Context(), notification("user.portfolio.eth", `{"currency":"ETH","balance":3,"available_withdrawal_funds":3}`)); notifyErr != nil {
				break
			}
			drain()
			replies <- reply
			require.NoError(t, received(updated).err, "UpdateAccountBalances must not error")
		}
		require.ErrorContains(t, notifyErr, "dispatcher jobs at limit", "the notification's publication must fail on the filled job slot")
		balance, err := savedBalance(t, ex, asset.Spot, currency.ETH)
		require.NoError(t, err, "the notified balance must be saved")
		require.Equal(t, 3.0, balance.Total, "the notification must save its balance though its publication fails")
		events, err := ex.SubscribeAccountBalances()
		require.NoError(t, err, "SubscribeAccountBalances must not error")
		t.Cleanup(func() { assert.NoError(t, events.Release(), "Release should not error") })
		drain()
		replies <- reply
		u := received(updated)
		require.NoError(t, u.err, "UpdateAccountBalances must not error")
		require.Len(t, u.subAccounts, 1, "UpdateAccountBalances must return one subaccount")
		assert.Equal(t, asset.Futures, u.subAccounts[0].AssetType, "UpdateAccountBalances should label the balances with the asset requested")
		var published *accounts.SubAccount
		select {
		case event := <-events.Channel():
			var ok bool
			published, ok = event.(*accounts.SubAccount)
			require.Truef(t, ok, "the account store must publish a subaccount, not %T", event)
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the account store must publish the update's save")
		}
		assert.Equal(t, asset.Spot, published.AssetType, "the account store should publish the balances saved under spot")
		for c, exp := range map[currency.Code]float64{currency.ETH: 3, currency.BTC: 4} {
			assert.Equalf(t, exp, u.subAccounts[0].Balances[c].Total, "UpdateAccountBalances should return the %s balance, keeping the notified one", c)
			assert.Equalf(t, exp, published.Balances[c].Total, "the account store should publish the %s balance, keeping the notified one", c)
			balance, err := savedBalance(t, ex, asset.Spot, c)
			require.NoErrorf(t, err, "the %s balance must be saved", c)
			assert.Equalf(t, exp, balance.Total, "UpdateAccountBalances should keep the %s balance notified after its request over its reply", c)
		}
	})
}

func TestWebsocketHandleUserTrades(t *testing.T) {
	t.Parallel()
	// Following Deribit's documented user.trades example
	const trades = `[{"trade_seq":30289432,"trade_id":"48079254","timestamp":1590484156350,"tick_direction":0,"state":"filled","reduce_only":false,"price":8954,"post_only":false,"order_type":"market","order_id":"4008965646","matching_id":null,"mark_price":8952.86,"liquidity":"T","instrument_name":"BTC-PERPETUAL","index_price":8956.73,"fee_currency":"BTC","fee":0.00000168,"direction":"sell","amount":20}]`
	ex := newTestExchange(t)
	ex.Features.Enabled.FillsFeed = true
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	require.NoError(t, ex.wsHandleUserTrades(t.Context(), notification("user.trades.BTC-PERPETUAL.100ms", trades)), "wsHandleUserTrades must not error")
	exp := []fill.Data{{
		ID:           "48079254",
		TradeID:      "48079254",
		Timestamp:    time.UnixMilli(1590484156350),
		Exchange:     ex.Name,
		AssetType:    asset.Futures,
		CurrencyPair: currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter),
		Side:         order.Sell,
		OrderID:      "4008965646",
		Price:        8954,
		Amount:       20,
	}}
	assert.Equal(t, exp, relayed(t, ex), "wsHandleUserTrades should relay the account's trades as fills")

	err := ex.wsHandleUserTrades(t.Context(), notification("user.trades.future.any.100ms", `[{"instrument_name":"BTC","direction":"sell"}]`))
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "wsHandleUserTrades should report a trade it cannot convert")
	assertNothingRelayed(t, ex)
	assert.ErrorIs(t, ex.wsHandleUserTrades(t.Context(), notification("user.trades.future.any.100ms", `{}`)), errMalformedNotification, "wsHandleUserTrades should reject malformed trades")

	disabled := newTestExchange(t)
	disabled.Features.Enabled.FillsFeed = false
	require.NoError(t, disabled.wsHandleUserTrades(t.Context(), notification("user.trades.BTC-PERPETUAL.100ms", trades)), "wsHandleUserTrades must not error with the fills feed off")
	assertNothingRelayed(t, disabled)
}

func TestUpdateFills(t *testing.T) {
	t.Parallel()
	documented := testUserChanges().Trades[0]
	second := documented
	second.TradeID, second.TradeSequence, second.Amount = "1430915", 866639, 20
	// A buy option trade with a lower ID and a price of its own, so a feed that sorts, drops an asset, keeps one trade an
	// asset or copies one trade's fields to another fails
	option := documented
	option.TradeID, option.InstrumentName, option.Direction, option.Price, option.Amount = "1430913", "BTC-25DEC26-104000-C", DirectionBuy, 0.0525, 20
	unconvertible := UserTradeResponse{TradeID: "7", InstrumentName: "BTC", Direction: DirectionSell}
	for _, tc := range []struct {
		name      string
		trades    []UserTradeResponse
		fillsFeed bool
		relayed   []UserTradeResponse
		err       error
	}{
		{name: "a trade converted", trades: []UserTradeResponse{documented}, fillsFeed: true, relayed: []UserTradeResponse{documented}},
		{name: "two trades converted", trades: []UserTradeResponse{documented, second}, fillsFeed: true, relayed: []UserTradeResponse{documented, second}},
		{name: "three trades of two assets converted", trades: []UserTradeResponse{documented, option, second}, fillsFeed: true, relayed: []UserTradeResponse{documented, option, second}},
		{name: "a trade it cannot convert", trades: []UserTradeResponse{unconvertible, documented}, fillsFeed: true, relayed: []UserTradeResponse{documented}, err: errUnsupportedInstrumentFormat},
		{name: "a trade it cannot convert between two", trades: []UserTradeResponse{documented, unconvertible, second}, fillsFeed: true, relayed: []UserTradeResponse{documented, second}, err: errUnsupportedInstrumentFormat},
		{name: "no trade converted", trades: []UserTradeResponse{unconvertible}, fillsFeed: true, err: errUnsupportedInstrumentFormat},
		{name: "no trades", fillsFeed: true},
		{name: "fills feed off", trades: []UserTradeResponse{unconvertible, documented, second}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.Features.Enabled.FillsFeed = tc.fillsFeed
			ex.Websocket.Fills.Setup(tc.fillsFeed, ex.Websocket.DataHandler)
			assert.ErrorIs(t, ex.updateFills(tc.trades), tc.err, "updateFills should return the errors of the trades it cannot convert")
			if len(tc.relayed) > 0 {
				exp := make([]fill.Data, 0, len(tc.relayed))
				for i := range tc.relayed {
					data, err := ex.userTradeFill(&tc.relayed[i])
					require.NoError(t, err, "userTradeFill must not error")
					exp = append(exp, *data)
				}
				assert.Equal(t, exp, relayed(t, ex), "updateFills should pass every trade it converts, in order, to the fills feed at once")
			}
			assertNothingRelayed(t, ex)
		})
	}

	t.Run("fills feed full", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		ex.Features.Enabled.FillsFeed = true
		full, refused := fullRelay(t)
		ex.Websocket.Fills.Setup(true, full)
		assert.ErrorIs(t, ex.updateFills([]UserTradeResponse{documented}), refused, "updateFills should return the fills feed's failure")
	})
}
