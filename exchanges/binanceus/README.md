# GoCryptoTrader package Binanceus

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/binanceus)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This binanceus package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Binanceus Exchange

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
var b exchange.IBotExchange

for i := range bot.Exchanges {
    if bot.Exchanges[i].GetName() == "Binanceus" {
        b = bot.Exchanges[i]
    }
}

// Public calls - wrapper functions

// Fetches current ticker information
tick, err := b.UpdateTicker(...)
if err != nil {
    // Handle error
}

// Fetches current orderbook information
ob, err := b.UpdateOrderbook(...)
if err != nil {
    // Handle error
}

// Private calls - wrapper functions - make sure your APIKEY and APISECRET are
// set and AuthenticatedAPISupport is set to true

// Fetches current account information
accountInfo, err := b.UpdateAccountBalances(ctx, asset.Spot)
if err != nil {
    // Handle error
}
```

+ If enabled via individually importing package, rudimentary example below:

```go
// Public calls

// Fetches the order book
book, err := e.GetOrderBook(ctx, &binanceus.OrderBookRequest{Symbol: currency.NewBTCUSDT(), Limit: 100})
if err != nil {
    // Handle error
}

// Private calls - make sure your APIKEY and APISECRET are set and
// AuthenticatedAPISupport is set to true

// Fetches the account's information and balances
account, err := e.GetAccount(ctx)
if err != nil {
    // Handle error
}

// Places an order
placed, err := e.NewOrder(ctx, &binanceus.NewOrderRequest{
    Symbol:      currency.NewBTCUSDT(),
    Side:        binanceus.SideBuy,
    Type:        binanceus.OrderTypeLimit,
    TimeInForce: binanceus.TimeInForceGoodTillCancel,
    Price:       50000,
    Quantity:    0.001,
})
if err != nil {
    // Handle error
}
```

### How to do Websocket public/private calls

+ The market data streams connect to the `WebsocketSpotURL` endpoint with the configured subscriptions of tickers,
trades, candles and order books, whose diff depth streams are kept in sync with REST snapshots.

+ The WebSocket API connects to the `WebsocketTradeURL` endpoint when authenticated websocket support is enabled. It
subscribes the configured account's User Data Stream, which streams balance and order updates, and the wrapper places
and cancels orders over it while it is connected. Its requests are methods of `WebsocketAPI()`, named after their
REST forms:

```go
book, err := e.WebsocketAPI().GetOrderBook(ctx, &binanceus.OrderBookRequest{Symbol: currency.NewBTCUSDT(), Limit: 5})
if err != nil {
    // Handle error
}
```

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
