# GoCryptoTrader package Bitfinex

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This bitfinex package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Bitfinex Exchange

### Current Features

+ Spot, margin and margin funding markets, and perpetual contracts: USDT margined futures, collateralised by USDt, and
  coin margined futures, collateralised by bitcoin
+ Every endpoint of Bitfinex's v2 REST API: public market data and configuration, wallets, orders, positions, margin
  funding, account actions and Thalex transfers
+ Wire withdrawals through Bitfinex's v1 API, which v2 has no equivalent of, as its withdrawal takes no bank details
+ Every public websocket channel GoCryptoTrader subscribes, with each order book checked against the checksums Bitfinex
  sends, and every account event, decoded into typed payloads; tickers, order books, candles, trades, orders, fills
  and balances also update GoCryptoTrader's stores and feeds
+ Websocket inputs for placing, updating and cancelling orders and funding offers, and for requesting calculations

### How to enable

+ [Enable via configuration](../../config/README.md#enable-exchange-via-config-example)

### Credentials

REST requests are signed with the API secret, and the websocket's account connection authenticates with a signature,
so the secret is never sent. Bitfinex requires each request with a key, over REST or the websocket, to carry a larger
nonce than the last, so a key should not be shared with another client.

### Records

Bitfinex sends most records as arrays whose positions carry the fields. Each type decodes its array by the positions its
`tuple` tags give, so `OrderResponse`, `PositionResponse`, `FundingOfferResponse` and the other records have named
fields. Write requests return a `Notification` holding the record they affected, whose `Err` reports a request Bitfinex
refused.

### Orders

+ `SubmitOrder` places spot orders from the exchange wallet, as Bitfinex's EXCHANGE order types, and margin orders from
  the margin wallet. It places limit, market, stop, stop limit and trailing stop orders; fill or kill and immediate or
  cancel limit orders; and post only, reduce only and good till time orders. A stop needs its `TriggerPrice`, which
  Bitfinex takes as its price, and a stop limit its `Price` too, the limit it places; a trailing stop takes its distance
  in price from the `TrackingValue`. Neither stop has a price of its own, and a trailing stop's `TriggerPrice` is the
  stop Bitfinex works out from its distance. A good till crossing limit order is placed post only, and a time in force
  or tracking mode Bitfinex has no order for, such as good till day without an end time, is refused rather than left
  out.
+ Client order IDs must be numbers, as Bitfinex's are.
+ `ModifyOrder` looks the order up, as Bitfinex reads an update by the order's type and takes its amount as what remains
  of the order. It moves a stop by its `TriggerPrice`, refusing a `Price` without one, such as the modify order RPC
  sends; a stop limit by its `TriggerPrice` and its `Price`, its limit; a trailing stop, which trails the market, by
  neither; and any other order by its `Price`. An amount is the order's whole amount, as GCT holds it, and is sent less
  what the order has executed, or not at all when unchanged, as Bitfinex would put back a fill made since the order was
  looked up; changing it needs the order's side, as Bitfinex signs an amount by its side. Bitfinex does not change an
  order's type, so another is refused. `ModifyOrder` leaves an order's time in force as it is and reports it as the
  order has it, as a modification carries no expiry time and the order manager sets each one's time in force to what
  it holds for the order. `UpdateOrder` and `WebsocketUpdateOrder` change an order's expiry, which an update cannot
  remove, and its flags, such as post only.
+ With the authenticated websocket connected, orders are placed, updated and cancelled through it; otherwise over REST.
+ `CancelAllOrders` cancels the open orders of the requested asset, and of its pair when one is given; a request naming
  neither, or the nil one the cancel all orders RPC sends, cancels every open order through Bitfinex's own cancellation
  of them all. `CancelBatchOrders` cancels its orders in one request. Both report the orders pending cancellation, as
  Bitfinex confirms each when the order closes.
+ `GetOrderInfo` returns an order with its trades and their fees, reading the order again when its trades show a fill
  the first read does not. `GetOrderHistory` returns the orders of the last two weeks, as Bitfinex keeps them. What an
  order has executed is read from the fills its status lists. Bitfinex cuts a status listing many fills short, keeping
  the latest, so a cut status's order is reported by its last fill, and has executed what its trades count, which
  `GetOrderInfo` reads for an order of the last 10 days, or else its original amount less what remains, which is exact
  unless the order's amount was updated, as Bitfinex keeps an order's original amount from before any update. A
  partially filled order from the history, or closed over the websocket, is reported partially filled and cancelled.

### Perpetual contracts

Bitfinex's derivatives are all perpetual contracts, on pairs whose currencies Bitfinex suffixes F0, such as
`BTCF0:USTF0`. Each trades as the futures asset its collateral, the pair's quote, margins: `usdtmarginedfutures` for
`USTF0`, Tether's USDt, and `coinmarginedfutures` for `BTCF0`, bitcoin. A pair given with another asset is refused.

+ `SubmitOrder` places a derivative order as a margin order, with the order's `Leverage`, a whole number from 1 to 100,
  or Bitfinex's default of 10 without one. Bitfinex takes leverage with each order, so `SetLeverage` is not supported.
  `ModifyOrder` sends a derivative order's own leverage, which Bitfinex needs to update it.
+ `GetLatestFundingRates` and `GetHistoricalFundingRates` return the rates Bitfinex sets every 8 hours, each from the
  premium accrued over the 8 hours before it. Bitfinex gives no predicted rate, and its ledgers record funding payments
  by currency rather than by contract, so neither is supported.
+ `GetFuturesContractDetails` returns each contract's maximum leverage, funding cap and latest funding rate, and
  `GetOpenInterest` every contract's open interest from one request.
+ `GetFuturesPositionSummary`, `GetLeverage` and `ChangePositionMargin` act on a contract's active position, which holds
  its own collateral, as an isolated margin position does. `GetFuturesPositionOrders` returns a contract's orders of the
  last two weeks.

### Balances

`UpdateAccountBalances` returns the wallet an asset trades from: the exchange wallet for spot, the margin wallet for
margin and the funding wallet for margin funding. The perpetual contracts trade from the derivatives wallet, which
Bitfinex keeps in the margin wallet as derivative currencies, such as `USTF0`. The websocket replaces them with the
wallet snapshot Bitfinex sends the account connection once it authenticates, then updates them as they change; a
snapshot holding a wallet of an unknown type or without a currency is merged into them instead, removing none. Bitfinex
sends no available balance until it has calculated it, and until then none of the balance is free; the websocket asks
Bitfinex to calculate it.

### Deposits and withdrawals

Bitfinex deposits and withdraws each currency over methods, its chains, such as `bitcoin`, or `tetheruse` and
`tetherusx` for Tether on Ethereum and Tron. `GetAvailableTransferChains` returns a currency's, and a currency with
several needs its chain given. Bitfinex names Tether's USDT `UST`, which the wrapper translates.

+ `GetDepositAddress` deposits into the funding wallet unless the account ID names `exchange` or `margin`. A currency
  needing a tag, memo or payment ID returns it as the address's tag.
+ `WithdrawCryptocurrencyFunds` and `WithdrawFiatFunds` withdraw from the exchange wallet unless the request's wallet ID
  names another.
+ A wire sends no account holder's address, which GCT's bank accounts do not hold, and gives its intermediary bank's
  IBAN, or else its account number, refusing one too long for GCT to hold exactly.
+ A cryptocurrency withdrawal's travel rule beneficiary is the account itself, or the person its beneficiary name gives,
  as first names and a last name split at the name's last space, which is how Bitfinex takes a name; its financial
  institution names the provider the withdrawal goes to.
+ `GetFeeByType` gives a cryptocurrency withdrawal the fee Bitfinex sets for its currency, or for a Tether token, which
  Bitfinex prices by chain, the fee of the token's chain. A fee request names no chain, so a token withdrawn over
  several, such as USDT, is refused.

### Market data

+ `UpdateTickers` updates every enabled pair's ticker from one request.
+ `UpdateOrderbook` fetches a pair's raw book of its 100 best orders a side, or a funding currency's offers, as the
  default websocket subscription keeps it.
+ `GetHistoricTrades` pages back from the end of the window, 10,000 trades a request. It errors should a millisecond
  hold a page of trades, which no request can page past, as do the order, deposit and withdrawal histories.
+ Funding candles cover offers of 30 days, as the default websocket subscription's do.
+ `GetHistoricCandles` and `GetHistoricCandlesExtended` make two-week candles from weekly ones, as Bitfinex's 14D
  candles, which `GetCandlesHistory` and two-week websocket candles give, open on Thursdays, where GCT's two weeks open
  on Mondays.

### Websocket

Market data goes over public connections of up to 25 channels each, five fewer than the 30 Bitfinex allows, which leaves
room for channels still being unsubscribed, and the account's events over an authenticated connection, which Bitfinex
subscribes to the account channel. The standard channels use these Bitfinex channels: tickers `ticker`, order books
`book`, candles `candles` and trades `trades`. Spot and margin trade the same markets, so a pair enabled for both has
one channel, which updates both.

+ An order book subscription's `prec` parameter sets its precision: `R0` for a book of orders, or `P0` to `P4` for
  price levels aggregated to five down to one significant figure; its levels set the length Bitfinex sends, 1, 25, 100
  or 250. Each book is checked against the checksum Bitfinex sends of its top 25 levels a side; a book that no longer
  matches is invalidated and its channel subscribed again, retrying until Bitfinex accepts, so it sends a new snapshot.
+ Candles take Bitfinex's time frames: a minute, 5, 15 or 30 minutes, an hour, 3, 6 or 12 hours, a day, a week, two
  weeks or a month; another interval is refused. Funding candles take their funding period, such as `p30`, from the
  `_period` parameter.
+ A perpetual contract's `status` channel relays its funding rate as funding data, and its status, with its mark price
  and open interest, as a `WebsocketDerivativeStatus`.
+ Each connection asks Bitfinex to stamp its events with the time they were sent, which tickers and order books are
  updated as of.
+ Account orders, fills and balances update GoCryptoTrader's stores; positions, funding offers, credits, loans, funding
  trades, margin and funding information and notifications are relayed as Bitfinex's records.

### Rate limits

Requests are limited to the rate the docs give each endpoint, from 3 to 240 a minute, and to 10 a minute where they
give none. Bitfinex blocks an IP that exceeds a limit for 60 seconds.

### How to do REST public/private calls

When enabled through configuration, Bitfinex is available through the `exchange.IBotExchange` interface:

```go
var b exchange.IBotExchange

for i := range bot.Exchanges {
	if bot.Exchanges[i].GetName() == "Bitfinex" {
		b = bot.Exchanges[i]
	}
}

// Public calls
tick, err := b.UpdateTicker(ctx, currency.NewBTCUSD(), asset.Spot)
if err != nil {
	// Handle error
}

book, err := b.UpdateOrderbook(ctx, currency.NewBTCUSD(), asset.Spot)
if err != nil {
	// Handle error
}

// Private calls, which need Key and Secret
balances, err := b.UpdateAccountBalances(ctx, asset.Spot)
if err != nil {
	// Handle error
}
```

The rest of the API is available through the package's own methods:

```go
bfx, ok := b.(*bitfinex.Exchange)
if !ok {
	// Handle error
}

candles, err := bfx.GetCandlesHistory(ctx, &bitfinex.CandlesHistoryRequest{
	Symbol:   "tBTCUSD",
	Interval: kline.OneHour,
	Limit:    24,
})
if err != nil {
	// Handle error
}
```

### How to do Websocket public/private calls

Channels are subscribed as the Websocket section describes. The `Websocket` methods send inputs over the authenticated
connection:

```go
placed, err := bfx.WebsocketPlaceOrder(ctx, &bitfinex.PlaceOrderRequest{
	Type:   bitfinex.ExchangeOrderTypePrefix + bitfinex.OrderTypeLimit,
	Symbol: "tBTCUSD",
	Amount: 0.001,
	Price:  price,
})
if err != nil {
	// Handle error
}
```

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
