# GoCryptoTrader package Kraken

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/kraken)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This kraken package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Kraken Exchange

### Current Features

+ REST Support
+ Websocket Support

### How to enable

+ [Enable via configuration](../../config/README.md#enable-exchange-via-config-example)

+ Individual package example below:

```go
    // Exchanges will be abstracted out in further updates and examples will be
    // supplied then
```

### How to do REST public/private calls

+ If enabled via "configuration".json file the exchange will be added to the
IBotExchange array in the ```go var bot Bot``` and you will only be able to use
the wrapper interface functions for accessing exchange data. View routines.go
for an example of integration usage with GoCryptoTrader. Rudimentary example
below:

main.go

```go
var k exchange.IBotExchange

for i := range bot.Exchanges {
    if bot.Exchanges[i].GetName() == "Kraken" {
        k = bot.Exchanges[i]
    }
}

// Public calls - wrapper functions

// Fetches current ticker information
tick, err := k.UpdateTicker(ctx, pair, asset.Spot)
if err != nil {
    // Handle error
}

// Fetches current orderbook information
ob, err := k.UpdateOrderbook(ctx, pair, asset.Spot)
if err != nil {
    // Handle error
}

// Private calls - wrapper functions - make sure your APIKEY and APISECRET are
// set and AuthenticatedAPISupport is set to true

// Fetches current account balances
balances, err := k.UpdateAccountBalances(ctx, asset.Spot)
if err != nil {
    // Handle error
}
```

+ If enabled via individually importing package, rudimentary example below:

```go
// Public calls

// Fetches current ticker information
tickers, err := k.GetTickerInformation(ctx, &kraken.TickerInformationRequest{Pairs: currency.Pairs{pair}})
if err != nil {
    // Handle error
}

// Fetches current orderbook information
books, err := k.GetOrderBook(ctx, &kraken.OrderBookRequest{Pair: pair})
if err != nil {
    // Handle error
}

// Private calls - make sure your APIKEY and APISECRET are set and
// AuthenticatedAPISupport is set to true

// Fetches each asset's balance, with the amount held by open orders
balances, err := k.GetExtendedBalance(ctx, nil)
if err != nil {
    // Handle error
}

// Submits an order to the exchange and returns its transaction ID
order, err := k.AddOrder(ctx, &kraken.AddOrderRequest{
    Pair: pair,
    Order: kraken.OrderParameters{
        OrderType: "limit",
        Side:      "buy",
        Volume:    0.001,
        Price:     kraken.OrderPrice{Value: 50000},
    },
})
if err != nil {
    // Handle error
}
```

+ Derivatives (futures) calls are named after their section, such as GetFuturesTickers, SendFuturesOrder,
GetFuturesAccounts and the Derivatives History API's GetFuturesOrderEvents.

### How to do Websocket public/private calls

+ The spot websocket uses Kraken's websocket v2 API over three connections: market data subscriptions stream on the
public connection; account subscriptions stream, and orders are placed, on the private connection; and level 3 order
books stream on their own connection. The private and level 3 connections connect when their subscriptions are
configured, authenticated websocket support is enabled and credentials are set. Spot tickers carry no opening price
of the day, so they leave `Open` unset.

+ The futures websocket streams the tickers, order books and trades of futures subscriptions, and, when
authenticated websocket support is enabled and credentials are set, the account's open orders, fills and balances
with the `myOrders`, `myTrades` and `myWallet` channels. Futures order books stream whole, so futures subscriptions
take no `levels`.

+ Kraken's spot balance messages carry neither the amount on hold nor the credit it counts as available, so each one
the `myWallet` channel streams refreshes the stored spot balances over REST, at most once every five seconds, and the
balances are read a second time after a ledger update, as REST can lag it. An order the private connection reports
finished after fills is likewise read over REST for its fee, as execution reports carry each fill's fee alone.
These reads are retried only while the websocket is connected, or still connecting.

+ The futures balances feed names single-collateral margin accounts differently from REST, so one is stored only
under the REST account it matches, such as `fi_xbtusd` for `F-XBT:USD`, once `UpdateAccountBalances` has stored that
account; other margin accounts follow REST.

+ A level 3 order book lists each order rather than the total at each price. It is subscribed to with the `allOrders`
channel at a depth of 10, 100 or 1000 price levels:

```json
{"enabled": true, "channel": "allOrders", "asset": "spot", "levels": 100}
```

+ With the `orderbook` channel, the stored order book holds the total at each price. With the `allOrders` channel, it
holds a level for each order, so a price appears once for each order resting at it, and the level 3 book's snapshots
and updates are also sent to the data handler as `WebsocketLevel3Book`s with their order IDs. As a pair has one stored
order book, it can be subscribed to at one depth of either the `orderbook` or the `allOrders` channel.

+ Websocket v2 streams one candle interval for each pair on a connection, so a pair's candles can be subscribed to at
one interval: Kraken rejects a subscription to a second.

+ Default subscriptions are configured in the exchange's `features.subscriptions` config. Upgrading a config that
holds the previous default subscriptions adds the futures subscriptions to them; other subscriptions are kept as they
are.

+ Orders can be managed over the private connection. The wrapper's spot order functions use it while it is connected
and holds a subscription, and REST otherwise, such as when Kraken refuses the websocket token:

```go
// Places an order over the private websocket connection
resp, err := k.WebsocketAddOrder(ctx, &kraken.WebsocketAddOrderRequest{
    Pair: pair,
    Order: kraken.WebsocketOrder{
        OrderType:  "limit",
        Side:       "buy",
        Quantity:   0.001,
        LimitPrice: 50000,
    },
})
if err != nil {
    // Handle error
}
```

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
