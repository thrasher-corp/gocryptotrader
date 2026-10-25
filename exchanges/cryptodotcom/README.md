# GoCryptoTrader package Cryptodotcom

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/cryptodotcom)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This cryptodotcom package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Crypto.com Exchange

### Current Features

+ REST Support
+ Websocket Support

### Supported Assets

+ Spot
+ Margin, trading spot pairs on the unified margin account
+ Perpetual swaps, such as BTCUSD-PERP
+ Futures, such as BTCUSD-261030

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
var c exchange.IBotExchange

for i := range bot.Exchanges {
    if bot.Exchanges[i].GetName() == "Cryptodotcom" {
        c = bot.Exchanges[i]
    }
}

// Public calls - wrapper functions

// Fetches current ticker information
tick, err := c.UpdateTicker(...)
if err != nil {
    // Handle error
}

// Fetches current orderbook information
ob, err := c.UpdateOrderbook(...)
if err != nil {
    // Handle error
}

// Private calls - wrapper functions - make sure your APIKEY and APISECRET are
// set and AuthenticatedAPISupport is set to true

// Fetches current account balances
balances, err := c.UpdateAccountBalances(...)
if err != nil {
    // Handle error
}
```

+ If enabled via individually importing package, rudimentary example below:

```go
// Public calls

// Fetches the instruments Crypto.com trades
instruments, err := c.GetInstruments(...)
if err != nil {
    // Handle error
}

// Fetches an instrument's order book
ob, err := c.GetOrderbook(...)
if err != nil {
    // Handle error
}

// Private calls - make sure your APIKEY and APISECRET are set and
// AuthenticatedAPISupport is set to true

// Fetches the account's balances
balances, err := c.GetUserBalance(...)
if err != nil {
    // Handle error
}

// Places an order and returns its order ID
resp, err := c.CreateOrder(...)
if err != nil {
    // Handle error
}
```

### How to do Websocket public/private calls

Crypto.com's market data, its user channels and its user API are served on separate websocket connections. Methods
Crypto.com serves over the user API, such as creating and cancelling orders, are on the `WebsocketAPI` type under
their REST names:

```go
// Places an order over the websocket and returns its order ID
resp, err := c.WebsocketAPI().CreateOrder(...)
if err != nil {
    // Handle error
}
```

### Subscriptions

Default Public Subscriptions:

+ Ticker for every asset
+ Orderbook, 50 levels with deltas every 100 milliseconds, for every asset
+ All trades for every asset
+ One minute candles for every asset

Default Authenticated Subscriptions, each covering every instrument:

+ My Orders
+ Advanced orders
+ My Trades, through the fills feed when it is enabled
+ My Account balances
+ Positions

Index price, mark price, settlement price, funding rate and estimated funding rate channels are also supported, and
are left to configuration as each subscription of a pair counts towards a connection's limit of 500. So are the
transactions, account risk, position balance and OTC quote request, quote and deal channels.

Subscriptions are subject to enabled assets and pairs.

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
