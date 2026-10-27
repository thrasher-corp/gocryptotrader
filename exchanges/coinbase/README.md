# GoCryptoTrader package Coinbase

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/coinbase)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This coinbase package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Coinbase Exchange

### Current Features

+ Spot and US derivatives through the Advanced Trade API, and perpetual contracts through Coinbase's Global Derivatives
  gateway, which Deribit runs for eligible accounts outside the US
+ Every endpoint of the Advanced Trade REST API, grouped as its docs are: accounts, products, public market data,
  orders, portfolios, US derivatives, fees, convert, payment methods and data
+ The Coinbase App API (v2), which holds an account's wallets, addresses, transactions, deposits and withdrawals, and
  Coinbase Exchange's currency endpoints, which list each currency's networks
+ Every JSON-RPC method of the Global Derivatives gateway, over HTTP and its websocket: market data, combo books,
  trading, account management, session management and subaccount transfers
+ Advanced Trade's market data and user websockets, and the gateway's public streams and private connection, on
  separate connections, decoding every channel into typed payloads; tickers, order books, candles, trades, orders and
  fills also update GoCryptoTrader's stores and feeds

### How to enable

+ [Enable via configuration](../../config/README.md#enable-exchange-via-config-example)

### Credentials

Coinbase authenticates with Coinbase Developer Platform (CDP) API keys: the key's name, such as
`organizations/{org_id}/apiKeys/{key_id}`, as the Key, and its private key as the Secret, an ECDSA key as PEM or an
Ed25519 key as PEM or base64. Each request carries a JWT signed with the private key, valid for two minutes, so the
secret is never sent. The gateway exchanges a JWT for an access token, which its private HTTP methods carry and which
is replaced before it expires; its private websocket connection authenticates with a JWT and renews its session.

### Assets

+ `spot` trades Advanced Trade's spot products, such as `BTC-USD`. A USDC product other than `USDT-USDC` and
  `EURC-USDC` trades on its USD product's book, as its alias, which Coinbase streams under the USD product's ID, so both
  pairs' tickers and books update from one subscription.
+ `futures` trades the US derivatives Coinbase Financial Markets lists, such as `BIT-30OCT26-CDE`: dated monthly
  contracts, and perpetual style contracts, such as `BIP-20DEC30-CDE`, which expire years out and pay funding hourly.
  Their amounts are in contracts. A contract is tradable once its month opens; until then Coinbase lists it as view
  only, which the available pairs leave out.
+ `usdcmarginedfutures` trades the gateway's USDC perpetuals, such as `BTC_USDC-PERPETUAL`, linear and margined in
  USDC, and `coinmarginedfutures` its inverse perpetuals, `BTC-PERPETUAL` and `ETH-PERPETUAL`, margined in their base
  currency, whose amounts are in USD. Coinbase's web app trades the USDC perpetuals alone, so `GetCurrencyTradeURL`
  has no trade page for the inverse perpetuals. The gateway's dated futures, options, spot and combos trade through the
  `GlobalDerivatives` and `WebsocketAPI` methods only.
+ The gateway's assets are disabled by default, as the gateway serves eligible accounts outside the US, though a
  configuration without currency pairs, such as `GetDefaultConfig` builds, enables every asset, as it does for every
  exchange. While one is enabled, a gateway connection that cannot be dialled takes the Coinbase websocket down with
  it, as the websocket manager has no optional connections; a disabled asset whose pairs fail to load keeps the pairs
  it has.

### Orders

+ `SubmitOrder` places Advanced Trade orders: market, limit, stop limit and bracket orders; fill or kill, immediate or
  cancel, post only and good till date orders, which a good till time or good till day order with an end time is; quote
  sized market and limit orders. A time in force Advanced Trade has no configuration for, a reduce only order, and a
  stop limit or bracket order sized in the quote currency, which Coinbase sizes in the base currency, are refused.
  Coinbase requires a client order ID, so an order without one is given a UUID, which the response returns.
  `RFQDisabled` turns smart order routing off. Coinbase's response only acknowledges an order, so it is reported new;
  `RetrieveFees` reads the order after `RetrieveFeeDelay` for its state and fees.
+ A gateway order is placed through the gateway's websocket where the wrapper may use it, and otherwise over HTTP. It
  takes limit, market, stop limit, stop market, take profit, take profit market and trailing stop orders, whose trigger
  price type sets their trigger; a trailing stop takes its distance from the `TrackingValue`. A post only order is
  rejected rather than repriced when it would take liquidity, and isolated margin places an order in the isolated margin
  subaccount the gateway binds to its instrument. Leverage is an instrument's, which `SetLeverage` sets. Its orders take
  no end time, and the response reports the amount the gateway accepted, which it can reduce for a reduce only order.
  A market order is reported as one though the gateway holds it as a limit order, and a trigger order that has fired is
  reported closed, as the order it placed carries on under its own ID.
+ `ModifyOrder` changes an Advanced Trade limit order's price and size, which Coinbase requires together, so the
  order's own fill in for those left unset; it changes a gateway order by its ID, or by its label in the pair. The
  gateway takes an edit's flags afresh, so the order's amount, price, trigger price and reduce only flag are kept where
  the change leaves them unset, as are its post only flags unless it asks for another post only setting. A time in
  force the edit cannot apply is refused: Advanced Trade edits price and size alone, and the gateway can make an order
  post only or not, but not change how long it rests. The order's own time in force, which the order manager passes
  with every modification, is accepted, a good till day one for the good till time order Advanced Trade made of it.
+ `CancelOrder`, `CancelBatchOrders`, `GetOrderInfo` and `ModifyOrder` find a gateway order in the main account, or in
  the isolated margin subaccount holding it. Advanced Trade cancels asynchronously, so `CancelBatchOrders` and
  `CancelAllOrders` report an order pending cancellation once Coinbase submits its cancel request, and an order named
  twice keeps the status of the cancel that went through. A batch that fails returns the statuses reported so far with
  its error, the gateway's orders being cancelled still. `CancelAllOrders` lists an asset's open orders and cancels
  them in batches, those of the type asked for alone. The gateway cancels an instrument's orders together, in
  both margin scopes, unless the request selects them by side, margin type, time in force or a single stop or take
  profit type, when they are listed and cancelled one at a time.
+ `GetOrderInfo` returns an order with its fills, less those a reversal undoes, `GetActiveOrders` an asset's live
  orders, Advanced Trade's pending and queued ones among them, and `GetOrderHistory` its filled, cancelled, expired and
  failed orders, by margin type and time in force too, a good till day filter matching the good till time orders
  Advanced Trade made of them. Advanced Trade's live orders come from one list of every live status, as an order moving
  between statuses would be missing from separate lists; were Coinbase to refuse the list, as its earlier documentation
  said, the open orders are listed after the others, which finds an order accepted or edited between the two lists but
  can miss one whose edit or cancellation is queued between them. An order's last update is its latest fill or accepted
  edit. The gateway's
  include the isolated margin subaccounts' orders. Its history and an order's trades come from both its recent store,
  which keeps orders for 30 minutes and trades for a day, and its historical store, which indexes them after a short
  delay. The gateway reports an order's state until 30 minutes after it closes, so `GetOrderInfo` finds an older order
  in its pair's history, a page at a time. Where the EU's strong customer authentication answers an order or fill list
  with a proof token challenge, which only the account holder can meet, the list reports an error rather than no orders.

### Derivatives

+ `GetFuturesContractDetails` returns the US derivatives, view only ones among them, and the gateway's perpetuals, with
  each perpetual style contract's latest funding rate.
+ `GetLatestFundingRates` returns a perpetual style US derivative's hourly rate, as its product reports it, or a gateway
  perpetual's latest hourly funding record. `GetHistoricalFundingRates` returns a gateway perpetual's hourly records;
  Coinbase keeps no funding history for US derivatives. Neither venue predicts the next rate or reports payments.
+ `GetOpenInterest` returns a US derivative's open interest in contracts, and a gateway perpetual's in its base
  currency, or USD for an inverse perpetual.
+ `GetFuturesPositionSummary` returns a US derivative's position, its sizes in contracts, or a gateway perpetual's cross
  margin position, its unrealised profit since the position opened and its realised profit the session's. `GetLeverage`
  and `SetLeverage` act on a gateway perpetual's cross or isolated margin; US derivatives take no leverage.

### Balances

`UpdateAccountBalances` returns the accounts an asset trades from: the spot accounts for spot and the US derivatives
accounts for futures, of the portfolio the API key belongs to, leaving out vaults, which cannot trade. Coinbase's
available balance leaves out what is held. The gateway's account summaries back both perpetual assets; a summary's free
amount is what can be withdrawn, which leaves out margin held, session profit and reserved balances. The two assets
split the gateway account's currencies, so each balance is counted once: bitcoin and ether, which margin the inverse
perpetuals, under `coinmarginedfutures`, and the rest, USDC among them, under `usdcmarginedfutures`. Each active
isolated margin subaccount is a sub-account of its own, by its ID, cleared once the gateway recycles it, as its
collateral returns to the main account.

### Deposits and withdrawals

Deposits and withdrawals go through the Coinbase App API's accounts. `GetAvailableTransferChains` returns the networks
Coinbase Exchange moves a currency on, such as `ethereum` and `solana` for USDC.

+ `GetDepositAddress` returns the newest address of the account ID given, or of the currency's primary account, on the
  network given, or else the currency's default network, which Coinbase Exchange names, creating one when there is
  none.
+ `WithdrawCryptocurrencyFunds` sends from the request's wallet ID, or the currency's primary account, with the
  request's travel rule details. `WithdrawFiatFunds` withdraws to the payment method its bank names, and commits the
  withdrawal at once.
+ `GetAccountFundingHistory` and `GetWithdrawalsHistory` list each account's transactions in full, a page at a time, as
  the App API filters them by no type; on an account with many trades that takes many requests. A send that credited
  the account, as Coinbase's sends of either direction did before it restricted the type, is a deposit.

### Market data

+ `UpdateTickers` updates every enabled pair's ticker: from Advanced Trade's product list, which carries no best bid or
  ask, with each pair's top of book fetched beside it, or from the gateway's book summaries, one request a currency its
  perpetuals settle in. `UpdateTicker` returns a product's best bid and ask. Advanced Trade's websocket tickers carry no
  quote volume, and its batched tickers no best bid or ask, so a pair keeps the ones another source stored, for a
  minute after that source sent them.
+ `GetRecentTrades` and `GetHistoricTrades` return each trade's taker side; Advanced Trade reports the maker's.
  Advanced Trade's history pages back a second at a time, and the gateway serves its last 24 hours of trades.
+ Candles cover 1, 5, 15 and 30 minutes, 1, 2, 4 and 6 hours and a day. The gateway aligns its longer candles to its
  08:00 UTC daily settlement and has no 4 hour candles, so its 4 hour, 6 hour and daily candles are built from 2 hour
  candles.
+ `GetFeeByType` estimates spot fees, at the account's tier or offline at the lowest tier's. US derivatives are charged
  by the contract and the gateway's perpetuals at their own tier, so their estimates are refused.

### Websocket

Market data and the user channels go over separate Advanced Trade connections, and the gateway's channels over its
public streams and its private connection. Advanced Trade's connections each subscribe to heartbeats first, which keep
them open, then one channel a request, as its acknowledgements name no request; a request that times out closes the
connection, as its late acknowledgement would be taken for the next request's. A connection carries at most 30
subscriptions, as Coinbase holds at most 30 level2 products a connection, and more connections open as needed. Each
message's sequence number is checked; after a gap, the books are resubscribed for new snapshots, and the account
channels for a fresh view of the account. A resubscription that fails closes the connection too. A closed connection
has the manager reconnect every Coinbase connection, the gateway's included, and subscribe each afresh, so a request
that times out on one connection resubscribes every channel. The standard channels use these channels: tickers
`ticker`, candles `candles`, trades `market_trades`, order books `level2` and the account `user`; `status`,
`ticker_batch` and `futures_balance_summary` take their own names.

+ The account channel, and `futures_balance_summary`, subscribe once, without an asset, as they cover every product.
  It reports no order's size, so each order takes the size it was placed, edited or last listed with; an order placed
  elsewhere is looked up, and relayed again with its size. A filled order with no base size, such as one first
  reported filled or one sized in the quote currency alone, takes what it filled as its size; one first reported
  cancelled, expired or failed has none, as its report doesn't say which currency its remaining quantity is in. An open
  order sized in the base currency that an edit can resize, a limit, stop limit, bracket or take profit/stop loss
  order, is looked up too when its filled and remaining quantities give another size, so an edit made elsewhere is
  followed, and takes what it filled as its size when it fills beyond it. An order Coinbase marks filled leaving dust
  keeps its size. The record keeps the size an order it held closed with for at least an hour, and a later report of
  the order's close with no size recorded for it, such as a repeated one, or one after a list read it closed, takes
  that size, or for a filled order what it executed where that is larger. `ModifyOrder` records the size it gives an
  order once Coinbase answers. Should a lookup started by the edit's own report list the order as it was before the
  edit, and end before that answer, the order is relayed with its old size until its next report.
+ On the gateway, tickers subscribe `ticker`, order books `book` and trades `trades`, aggregated over 100ms by default,
  `agg2` for an interval of a second or `raw` for `kline.Raw`, and candles `chart.trades` at their interval, of up to 2
  hours, as the gateway aligns longer ones to its daily settlement. Any of the gateway's channels can be subscribed by
  name, such as `deribit_price_index`. Its private channels, such as `user.orders`, `user.trades` and `user.portfolio`,
  are not in the default subscriptions: the gateway refuses them to an account without its derivatives, and the private
  connection is only dialled for them. It renews its session before the session expires; while a renewal fails, the
  wrapper's private calls go over HTTP, and a session that lapses reconnects the websocket. A session acts for its API
  key's account alone, so a call carrying another key's credentials goes over HTTP.

### Rate limits

Advanced Trade's authenticated requests are limited to 27 a second, under the 30 a second it allowed, and its public
ones to 10 a second; the Coinbase App API to 10,000 an hour; Coinbase Exchange to 10 a second. The gateway counts its
credits as Deribit does: 20 requests a second, 5 matching engine requests a second, 10 subscriptions every 3 seconds,
and 1 a second for its costliest methods. The Advanced Trade websocket takes 8 connections and 8 messages a second.

### How to do REST public/private calls

When enabled through configuration, Coinbase is available through the `exchange.IBotExchange` interface:

```go
var c exchange.IBotExchange

for i := range bot.Exchanges {
	if bot.Exchanges[i].GetName() == "Coinbase" {
		c = bot.Exchanges[i]
	}
}

// Public calls
tick, err := c.UpdateTicker(ctx, currency.NewBTCUSD(), asset.Spot)
if err != nil {
	// Handle error
}

book, err := c.UpdateOrderbook(ctx, currency.NewPairWithDelimiter("BTC", "USDC-PERPETUAL", "_"), asset.USDCMarginedFutures)
if err != nil {
	// Handle error
}

// Private calls, which need Key and Secret
balances, err := c.UpdateAccountBalances(ctx, asset.Spot)
if err != nil {
	// Handle error
}
```

The rest of the API is available through the package's own methods: Advanced Trade and Coinbase Exchange's on the
exchange, the Coinbase App API's on `App`, and the gateway's on `GlobalDerivatives` over HTTP:

```go
cb, ok := c.(*coinbase.Exchange)
if !ok {
	// Handle error
}

products, err := cb.ListPublicProducts(ctx, &coinbase.ListProductsRequest{ProductType: coinbase.ProductTypeFuture})
if err != nil {
	// Handle error
}

accounts, err := cb.App().ListAccounts(ctx, &coinbase.AppListAccountsRequest{})
if err != nil {
	// Handle error
}

instruments, err := cb.GlobalDerivatives().GetInstruments(ctx, &coinbase.DerivativesInstrumentsRequest{
	Currency: currency.USDC,
	Kind:     coinbase.DerivativesKindFuture,
})
if err != nil {
	// Handle error
}
```

### How to do Websocket public/private calls

Channels are subscribed as the Websocket section describes. `WebsocketAPI` calls the gateway's methods over its
private connection, taking the same arguments as their `GlobalDerivatives` forms:

```go
placed, err := cb.WebsocketAPI().Buy(ctx, &coinbase.DerivativesOrderRequest{
	InstrumentName: "BTC_USDC-PERPETUAL",
	Amount:         0.001,
	Type:           coinbase.DerivativesOrderTypeLimit,
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
