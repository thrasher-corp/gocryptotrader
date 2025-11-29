# GoCryptoTrader package Okx

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/okx)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This okx package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Okx Exchange

### Current Features

+ REST Support
+ Websocket Support

### How to enable

 [Enable via configuration](https://github.com/thrasher-corp/gocryptotrader/tree/master/config#enable-exchange-via-config-example)

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
var ok exchange.IBotExchange

for i := range bot.Exchanges {
	if bot.Exchanges[i].GetName() == "Okx" {
		y = bot.Exchanges[i]
	}
}

// Public calls - wrapper functions

// Fetches current ticker information
tick, err := ok.UpdateTicker(...)
if err != nil {
	// Handle error
}

// Fetches current orderbook information
ob, err := ok.UpdateOrderbook(...)
if err != nil {
	// Handle error
}

// Private calls - wrapper functions - make sure your APIKEY, APISECRET, and API_CLIENT_ID are
// set and AuthenticatedAPISupport is set to true

// Fetches current account information
accountInfo, err := ok.GetAccountInfo()
if err != nil {
	// Handle error
}
```

+ If enabled via individually importing package, rudimentary example below:

```go
// Public calls

// Fetches current ticker information
ticker, err := ok.GetTicker()
if err != nil {
	// Handle error
}

// Fetches current orderbook information
ob, err := ok.GetOrderBook()
if err != nil {
	// Handle error
}

// Fetches historic trade data within the timeframe provided
tradeDatas, err := ok.GetHistoricTrades(...)
if err != nil {
	// Handle error
}


// Returns an estimate of fee based on the type of transaction
fee, err := ok.GetFeeByType(...)
if err != nil {
	// Handle error
}

// Private calls - make sure your APIKEY and APISECRET are set and
// AuthenticatedAPISupport is set to true

// Submits an order to the exchange and returns its tradeID
orderID, err := ok.SubmitOrder(...)
if err != nil {
	// Handle error
}

// ModifyOrder modifies an existing order
updatedOrder, err := ok.ModifyOrder(...)
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

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
