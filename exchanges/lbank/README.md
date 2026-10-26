# GoCryptoTrader package Lbank

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/lbank)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This lbank package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Lbank Exchange

### Current Features

+ Spot markets, including leveraged tokens, and the public data of USDT margined perpetual contracts
+ Every endpoint of LBank's v2 spot REST API: common information, wallet, market data, spot trading and the subscribe
  keys of the account websocket channels
+ The four public endpoints of LBank's contract API: server time, instruments, market data and order books. LBank
  serves contract trading and accounts to approved API users only, so those endpoints aren't implemented
+ Websocket tickers, order books, trades and candles, and the account's order and balance updates, which also update
  GoCryptoTrader's stores and feeds

### How to enable

+ [Enable via configuration](../../config/README.md#enable-exchange-via-config-example)

### Credentials

`Key` is the API key and `Secret` its secret. LBank issues HmacSHA256 secrets and RSA private keys, which it shows as
base64 PKCS #8 starting with `MI`; a PEM private key works too. Requests are signed with the kind of secret given, so the
secret is never sent. A secret that can't sign requests, such as an RSA key that doesn't parse, turns authenticated
support off when the exchange is set up.

### Orders

+ `SubmitOrder` places spot limit orders, good till cancelled, post only, immediate or cancel or fill or kill, and spot
  market orders. A market buy spends its `QuoteAmount` and a market sell sells its `Amount`. Contracts can't be traded.
+ `ModifyOrder` isn't supported, as LBank doesn't amend orders.
+ `CancelOrder` cancels an order by its ID or client order ID, and `CancelAllOrders` cancels a pair's open orders.
  LBank's current API cancels no list of orders, so `CancelBatchOrders` isn't supported.
+ `GetOrderInfo` returns an order in any state, and needs the order's pair.
+ `GetActiveOrders` and `GetOrderHistory` return the orders of the requested pairs, or of the enabled spot pairs when
  none are requested. LBank's history holds the cancelled and filled orders placed in the last 24 hours.

### Balances and withdrawals

`UpdateAccountBalances` returns the spot balances, each one's hold being the amount LBank has locked.
`GetAccountFundingHistory` and `GetWithdrawalsHistory` return the deposits and withdrawals of the last 90 days, as LBank
does by default.

`GetDepositAddress` returns a coin's address on the chain given, or on its default chain, with its memo as the tag.
LBank requires a withdrawal to carry its chain's fee, so `WithdrawCryptocurrencyFunds` looks the fee up from the coins
the account holds when the request gives none. A withdrawal with `InternalTransfer` set goes to another LBank account.

### Fees

`GetFeeByType` estimates a trade's fee at the account's maker or taker rate for the pair, and at LBank's standard rate of
0.1% without valid credentials. It estimates a withdrawal's fee on the first chain LBank lists for the coin.

### Market data

+ `UpdateTickers` updates spot tickers from LBank's tickers and its leveraged token tickers, which it serves apart, and
  contract tickers from the contracts' market data. `UpdateTicker` asks for a leveraged token's ticker when LBank's
  standard endpoint refuses the pair.
+ Order books are fetched 200 levels deep. LBank sends a contract's order book without a time, so it is dated when it
  arrives.
+ `GetRecentTrades` returns the latest 500 trades and `GetHistoricTrades` pages forward from the start. LBank's
  `isBuyerMaker` is true when the buyer took liquidity, so such a trade is a buy.
+ Candles of 1 minute to 4 hours are fetched directly. LBank starts its day, week and month candles at midnight UTC+8,
  so longer candles are built from 4 hour candles. LBank serves no trades or candles for contracts.
+ `GetLatestFundingRates` returns each contract's rate for the current funding interval, of 1, 4 or 8 hours, dated an
  interval before the next funding time.

### Websocket

Market data channels are spread over connections of up to 200 subscriptions each. The standard channels use these
LBank channels: tickers `tick`, order books `depth`, trades `trade` and candles `kbar`. Order books take the
subscription's levels, defaulting to 100, the deepest LBank documents, and LBank sends each update as a full
snapshot. Candles take 1, 5, 15 or 30 minute, 1 or 4 hour, day, week or month intervals; day and longer candles start
at midnight UTC+8, and weeks on Sunday. LBank acknowledges no subscription.

When authenticated, a separate connection carries the account's orders, `orderUpdate`, and balances, `assetUpdate`. It
creates a subscribe key of its own with its first account subscription, so a key LBank refuses fails only the account
subscriptions, which the next connect or flush retries, and leaves the market data streaming. The key is refreshed every
30 minutes, retrying each minute after a failed refresh, and closed when the connection ends. A key left unrefreshed for
the hour it lasts is replaced, and the account channels are subscribed to again with the new key.

Each connection pings LBank every 20 seconds and answers LBank's pings. LBank's websocket errors carry the client's IP
address, which is removed before an error is reported.

### Rate limits

Requests are limited to LBank's documented rates: 500 order placements and cancellations and 200 other requests every 10
seconds. The contract API documents no limit, so it is held to 200 requests every 10 seconds.

### How to do REST public/private calls

When enabled through configuration, LBank is available through the `exchange.IBotExchange` interface:

```go
var l exchange.IBotExchange

for i := range bot.Exchanges {
	if bot.Exchanges[i].GetName() == "Lbank" {
		l = bot.Exchanges[i]
	}
}

// Public calls
tick, err := l.UpdateTicker(ctx, pair, asset.Spot)
if err != nil {
	// Handle error
}

book, err := l.UpdateOrderbook(ctx, pair, asset.Spot)
if err != nil {
	// Handle error
}

// Private calls, which need Key and Secret
balances, err := l.UpdateAccountBalances(ctx, asset.Spot)
if err != nil {
	// Handle error
}
```

The rest of the API is available through the package's own methods:

```go
lb, ok := l.(*lbank.Exchange)
if !ok {
	// Handle error
}

fees, err := lb.GetTradingFeeRates(ctx, pair)
if err != nil {
	// Handle error
}
```

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
