# GoCryptoTrader package Bitstamp

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This bitstamp package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Bitstamp Exchange

### Current Features

+ REST Support for the Bitstamp v2 API, including every documented endpoint
+ Websocket Support for orderbooks, detail orderbooks, trades, funding rates, announcements and private order, trade, order trade, self trade, settlement, liquidation and token settlement channels
+ Private order events missed during a reconnection or a server restart are recovered from the REST API
+ The `diff_order_book` channel is not supported, as its documented reconciliation with a REST snapshot keeps levels of orders created and deleted between broadcasts, which leaves the book crossed
+ Spot and perpetual contract assets; perpetual contract pairs are quoted in `USD-PERP`, for example `BTC/USD-PERP`
+ Perpetual contract support for contract details, funding rates, open interest, leverage, positions and collateral

### Authentication

+ Private endpoints use Bitstamp's v2 header authentication and only require an API key and secret
+ Sub account requests are sent when the credentials' sub account is set
+ Private websocket channels use a short lived token fetched from the REST API when subscribing

### Testing

+ Tests run against recorded mock responses in `testdata/http.json` by default
+ Live tests are enabled with the `mock_test_off` build tag, for example `go test -tags=mock_test_off ./exchanges/bitstamp/...`
+ Authenticated live tests require credentials in `bitstamp_live_test.go`, and order placement tests also require `canManipulateRealOrders` to be enabled in `bitstamp_test.go`
+ Setting `useTestNet` in `bitstamp_test.go` runs the live REST tests against the Bitstamp sandbox, which requires sandbox credentials; the sandbox does not provide a websocket server

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
    if bot.Exchanges[i].GetName() == "Bitstamp" {
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
accountInfo, err := b.GetAccountInfo()
if err != nil {
    // Handle error
}
```

+ If enabled via individually importing package, rudimentary example below:

```go
// Public calls

// Fetches current ticker information
ticker, err := b.GetTicker()
if err != nil {
    // Handle error
}

// Fetches current orderbook information
ob, err := b.GetOrderBook()
if err != nil {
    // Handle error
}

// Private calls - make sure your APIKEY and APISECRET are set and
// AuthenticatedAPISupport is set to true

// GetUserInfo returns account info
accountInfo, err := b.GetUserInfo(...)
if err != nil {
    // Handle error
}

// Submits an order to the exchange and returns its tradeID
tradeID, err := b.Trade(...)
if err != nil {
    // Handle error
}
```

### How to do Websocket public/private calls

```go
    // Exchanges will be abstracted out in further updates and examples will be
    // supplied then
```

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
