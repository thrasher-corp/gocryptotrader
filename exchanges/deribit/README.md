# GoCryptoTrader package Deribit

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/deribit)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This deribit package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Deribit Exchange

### Current Features

+ Spot, futures, options, future combo and option combo markets
+ Every method of Deribit's JSON-RPC API: market data, trading, combo books, account management, wallet, block trades,
  Block RFQ, the Liquidity Support Program, supporting methods, authentication, session management and subscription
  management
+ Each method is available over REST, and under the same name over the websocket through `WebsocketAPI()`, which also
  holds the methods Deribit serves only over the websocket: heartbeats, cancel on disconnect, subscriptions, hello and
  logout
+ Every websocket notification channel, decoded into typed payloads; tickers, orderbooks, candles, trades, account
  orders and account trades also update GoCryptoTrader's stores and feeds
+ Security key (two-factor authentication) confirmation of withdrawals, API key management and other sensitive requests

### How to enable

+ [Enable via configuration](../../config/README.md#enable-exchange-via-config-example)

### Credentials

`Key` is an API key's client ID and `Secret` its client secret. REST requests are signed with Deribit's
`deri-hmac-sha256` scheme, and the websocket authenticates with a client signature, so the secret is never sent.

### Testnet

Deribit's testnet takes credentials of its own. To use it, set the `RestFuturesURL` endpoint to
`https://test.deribit.com` and the `WebsocketSpotURL` endpoint to `wss://test.deribit.com/ws/api/v2`.

### Security keys

On an account with two-factor authentication, Deribit answers withdrawals, API key management and other sensitive
requests with a security key challenge instead of carrying them out. Methods return it as a
`*deribit.SecurityKeyChallengeError`, which wraps `deribit.ErrSecurityKeyAuthorisationRequired`. Sending the request
again within a minute, with a context from `deribit.WithSecurityKeyAuthorisation` holding the challenge and the current
code of the two-factor authentication app, confirms it:

```go
withdrawal, err := db.Withdraw(ctx, req)
var challenge *deribit.SecurityKeyChallengeError
if errors.As(err, &challenge) {
	withdrawal, err = db.Withdraw(deribit.WithSecurityKeyAuthorisation(ctx, challenge.Challenge, code), req)
}
if err != nil {
	// Handle error
}
```

Deribit accepts each code once, so a context holding one should only send the challenged request again.

### Orders

+ Amounts are in Deribit's units: USD for perpetuals and inverse futures, and the base currency for options, linear
  futures and spot.
+ `SubmitOrder` places limit, market, stop limit, stop market, take profit limit, take profit market and trailing stop
  orders. Trigger orders trigger on the last price unless `TriggerPriceType` selects the index or mark price, and a
  trailing stop needs a `TrackingMode` of `order.Distance` with its distance as the `TrackingValue`.
+ Post only orders are rejected rather than repriced when they would take liquidity, as spot routed to Coinbase Exchange
  requires.
+ `ModifyOrder` edits an order identified by its ID, or by its label in a pair. Deribit requires an amount with every
  edit, so when the modification leaves it unset, the order is fetched first and keeps its amount.
+ `CancelBatchOrders` cancels one order a request, as Deribit cancels no list of orders, and reports each order's state
  or error. `CancelAllOrders` cancels a pair's orders, or every order of the asset when the pair is empty.
+ `GetActiveOrders` returns the open orders of the requested pairs, or of every pair of the asset when none are
  requested. `GetOrderHistory` returns the latest 1000 orders of each requested pair, including those older than two
  days and those that never filled.

### Balances

Deribit's balances back every kind of instrument, so `UpdateAccountBalances` saves them under the asset requested. A
balance's total is the currency's cash balance, its free amount what can be withdrawn, and its hold the rest, which
covers the margin in use, positive session profit and reserved balances.

`GetAccountFundingHistory` returns the latest 1000 deposits and withdrawals of each currency available to the account,
and `GetWithdrawalsHistory` the latest 1000 withdrawals of a currency, or of each one when the currency is empty.

### Market data

+ `UpdateTickers` updates the tickers of an asset's enabled pairs from Deribit's book summaries, one request per currency
  they are summarised under: a spot pair's base currency, or a derivative's settlement currency, which is USDC for linear
  instruments such as `ETH_USDC-PERPETUAL`. Summaries leave out the best bid and ask sizes and the index price, which
  `UpdateTicker` returns.
+ Deribit serves no trade tape for the spot it routes to Coinbase Exchange, such as `BTC_USDC`, so `GetRecentTrades` and
  `GetHistoricTrades` fail for it, and its trades and candles are left out of subscriptions once its pairs are fetched.
+ A combo's spread can be quoted at a zero price, which an orderbook cannot hold, so such levels are left out.
+ Candles of 1 minute to 2 hours are fetched directly and longer ones are built from them, as Deribit aligns its 3 hour
  and longer candles to 08:00 UTC.

### Websocket

The standard channels use these Deribit channels: tickers `ticker`, orderbooks `book`, candles `chart.trades`, trades
`trades`, account orders `user.orders`, account trades `user.trades` and account balances `user.portfolio`. Any other
channel is subscribed by using its name, such as `deribit_price_index` or `user.changes`, as the subscription's
channel. An instrument's channels are subscribed for each enabled pair of the subscription's asset, and the other
channels take their options from the subscription's parameters:

+ `kind` and `currency` select the instruments of a channel such as `trades` subscribed without an asset, and default
  to `any`
+ `index_name`, such as `btc_usd`, selects an index channel's index, and is required
+ `currency` selects a currency channel's currency, such as for `user.portfolio`, and defaults to `any`
+ `group`, with 1, 10 or 20 levels, subscribes a grouped orderbook, whose prices it groups, such as by `none` or `5`

Intervals of 100ms, 1s (Deribit's `agg2`) or, when authenticated, raw set how often a channel notifies, and candles take
their resolution from the interval. An orderbook change that does not follow the last one applied invalidates the book
and resubscribes it, so Deribit sends a new snapshot.

The connection asks Deribit for a heartbeat every 15 seconds and answers each one.

### Rate limits

Requests are limited to Deribit's default credits: 20 non-matching engine and 5 matching engine requests a second, and
the lower rates of the methods Deribit limits separately. Subaccounts with more trading volume are allowed more.

### How to do REST public/private calls

When enabled through configuration, Deribit is available through the `exchange.IBotExchange` interface:

```go
var d exchange.IBotExchange

for i := range bot.Exchanges {
	if bot.Exchanges[i].GetName() == "Deribit" {
		d = bot.Exchanges[i]
	}
}

// Public calls
tick, err := d.UpdateTicker(ctx, pair, asset.Futures)
if err != nil {
	// Handle error
}

book, err := d.UpdateOrderbook(ctx, pair, asset.Futures)
if err != nil {
	// Handle error
}

// Private calls, which need Key and Secret
balances, err := d.UpdateAccountBalances(ctx, asset.Futures)
if err != nil {
	// Handle error
}
```

The rest of the API is available through the package's own methods:

```go
db, ok := d.(*deribit.Exchange)
if !ok {
	// Handle error
}

instruments, err := db.GetInstruments(ctx, &deribit.InstrumentsRequest{
	Currency: currency.BTC,
	Kind:     deribit.KindOption,
})
if err != nil {
	// Handle error
}
```

### How to do Websocket public/private calls

Channels are subscribed as the Websocket section describes. `WebsocketAPI()` sends the methods over the websocket:

```go
placed, err := db.WebsocketAPI().Buy(ctx, &deribit.OrderRequest{
	InstrumentName: "BTC-PERPETUAL",
	Amount:         10,
	Type:           deribit.OrderTypeLimit,
	Price:          price,
})
if err != nil {
	// Handle error
}
```

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
