package deribit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
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

func TestWebsocketHandlePortfolio(t *testing.T) {
	t.Parallel()
	// Deribit's documented user.portfolio example
	const channel = "user.portfolio.btc"
	const btcPortfolio = `{"delta_total_map":{"btc_usd":31.594397699},"margin_balance":302.62675921,"session_upl":0.05341555,"options_gamma_map":{"btc_usd":1e-05},"options_value":-0.0079,"available_withdrawal_funds":301.35426172,"maintenance_margin":0.8854841,"available_funds":301.38036328,"balance":302.60065765,"equity":302.6188592,"fee_balance":0,"currency":"BTC","portfolio_margining_enabled":false,"margin_model":"segregated_sm","initial_margin":1.24639592,"spot_reserve":0,"delta_total":31.602298,"additional_reserve":0}`
	cashBalance, withdrawable := 302.60065765, 301.35426172
	expBalance := accounts.Balance{Currency: currency.BTC, Total: cashBalance, Hold: cashBalance - withdrawable, Free: withdrawable}

	t.Run("nothing saved", func(t *testing.T) {
		t.Parallel()
		ex := newAccountTestExchange(t)
		require.NoError(t, ex.wsHandleNotification(t.Context(), channel, notification(channel, btcPortfolio)), "wsHandleNotification must not error for a portfolio")
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

	t.Run("saved under futures", func(t *testing.T) {
		t.Parallel()
		ex := newAccountTestExchange(t, &accounts.SubAccount{AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.BTC: {Total: 1, Free: 1}}})
		require.NoError(t, ex.wsHandlePortfolio(t.Context(), notification(channel, btcPortfolio)), "wsHandlePortfolio must not error")
		assert.IsType(t, &AccountSummary{}, relayed(t, ex), "wsHandlePortfolio should relay the summary")
		balance, err := savedBalance(t, ex, asset.Futures, currency.BTC)
		require.NoError(t, err, "the balance must be saved under futures")
		assert.Equal(t, expBalance, balance, "wsHandlePortfolio should update the balance where it is saved")
		_, err = savedBalance(t, ex, asset.Spot, currency.BTC)
		assert.ErrorIs(t, err, accounts.ErrNoBalances, "wsHandlePortfolio should not save another copy, which asset.All totals would count again")
	})

	t.Run("saved under two assets", func(t *testing.T) {
		t.Parallel()
		ex := newAccountTestExchange(
			t,
			&accounts.SubAccount{AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.BTC: {Total: 1, Free: 1}}},
			&accounts.SubAccount{AssetType: asset.Options, Balances: accounts.CurrencyBalances{currency.BTC: {Total: 1, Free: 1}}},
		)
		require.NoError(t, ex.wsHandlePortfolio(t.Context(), notification(channel, btcPortfolio)), "wsHandlePortfolio must not error")
		for _, a := range []asset.Item{asset.Futures, asset.Options} {
			balance, err := savedBalance(t, ex, a, currency.BTC)
			require.NoErrorf(t, err, "the balance must be saved under %s", a)
			assert.Equalf(t, expBalance, balance, "wsHandlePortfolio should update the balance saved under %s", a)
		}
		_, err := savedBalance(t, ex, asset.Spot, currency.BTC)
		assert.ErrorIs(t, err, accounts.ErrNoBalances, "wsHandlePortfolio should not save a copy under spot")
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
}

func TestMainAccountAssets(t *testing.T) {
	t.Parallel()
	// Save writes to the balances it is given, so each sub-account needs its own
	balances := func() accounts.CurrencyBalances { return accounts.CurrencyBalances{currency.BTC: {Total: 1}} }
	for _, tc := range []struct {
		name  string
		saved []*accounts.SubAccount
		exp   asset.Items
	}{
		{name: "nothing saved", exp: asset.Items{asset.Spot}},
		{name: "main account", saved: []*accounts.SubAccount{{AssetType: asset.Futures, Balances: balances()}, {AssetType: asset.Options, Balances: balances()}}, exp: asset.Items{asset.Futures, asset.Options}},
		{name: "subaccount", saved: []*accounts.SubAccount{{ID: "sub", AssetType: asset.Futures, Balances: balances()}}, exp: asset.Items{asset.Spot}},
		{name: "main account and subaccount", saved: []*accounts.SubAccount{{AssetType: asset.Options, Balances: balances()}, {ID: "sub", AssetType: asset.Futures, Balances: balances()}}, exp: asset.Items{asset.Options}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assets, err := newAccountTestExchange(t, tc.saved...).mainAccountAssets(t.Context())
			require.NoError(t, err, "mainAccountAssets must not error")
			assert.ElementsMatch(t, tc.exp, assets, "mainAccountAssets should return the assets the main account's balances are saved under")
		})
	}

	ex := newAccountTestExchange(t)
	ex.SkipAuthCheck = false
	ex.SetCredentials(&accounts.Credentials{})
	_, err := ex.mainAccountAssets(t.Context())
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "mainAccountAssets should need credentials")
	ex = newAccountTestExchange(t)
	ex.Accounts = nil
	_, err = ex.mainAccountAssets(t.Context())
	assert.ErrorIs(t, err, common.ErrNilPointer, "mainAccountAssets should return an account store failure")
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
