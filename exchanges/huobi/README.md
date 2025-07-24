# GoCryptoTrader package Huobi

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/huobi)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This huobi package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Huobi Exchange

### Current Features

+ REST Support
+ Websocket Support

### How to enable

+ [Enable via configuration](https://github.com/thrasher-corp/gocryptotrader/tree/master/config#enable-exchange-via-config-example)

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
var h exchange.IBotExchange

for i := range bot.Exchanges {
	if bot.Exchanges[i].GetName() == "Huobi" {
		h = bot.Exchanges[i]
	}
}

// Public calls - wrapper functions

// Fetches current ticker information
tick, err := h.UpdateTicker(...)
if err != nil {
	// Handle error
}

// Fetches current orderbook information
ob, err := h.UpdateOrderbook(...)
if err != nil {
	// Handle error
}

// Private calls - wrapper functions - make sure your APIKEY and APISECRET are
// set and AuthenticatedAPISupport is set to true

// Fetches current account information
accountInfo, err := h.GetAccountInfo()
if err != nil {
	// Handle error
}
```

+ If enabled via individually importing package, rudimentary example below:

```go
// Public calls

// Fetches current ticker information
ticker, err := h.GetTicker()
if err != nil {
	// Handle error
}

// Fetches current orderbook information
ob, err := h.GetOrderBook()
if err != nil {
	// Handle error
}

// Private calls - make sure your APIKEY and APISECRET are set and
// AuthenticatedAPISupport is set to true

// GetUserInfo returns account info
accountInfo, err := h.GetUserInfo(...)
if err != nil {
	// Handle error
}

// Submits an order and the exchange and returns its tradeID
tradeID, err := h.Trade(...)
if err != nil {
	// Handle error
}
```

### Subscriptions

All subscriptions are for spot only.

Default Public Subscriptions:
- Ticker
- Candles ( Interval: 1min )
- Orderbook ( Level: 0 - No aggregation )
  - Configure Level: 1-5 for depth aggregation, for example:
```json
 {"enabled": true, "channel": "orderbook", "asset": "spot", "levels": 1}
```
- Trades

Default Authenticated Subscriptions:
- Account Trades
- Account Orders
- Account Updates

### Please click GoDocs chevron above to view current GoDoc information for this package

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-corp/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
