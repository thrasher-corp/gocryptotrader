# GoCryptoTrader package Lbank

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/lbank)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This lbank package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Lbank Exchange

### Current Features

+ REST Support

### How to enable

+ [Enable via configuration](https://githul.com/thrasher-corp/gocryptotrader/tree/master/config#enable-exchange-via-config-example)

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
var l exchange.IBotExchange

for i := range Bot.Exchanges {
	if Bot.Exchanges[i].GetName() == "Lbank" {
		l = Bot.Exchanges[i]
	}
}

// Public calls - wrapper functions

// Fetches current ticker information
tick, err := l.UpdateTicker(...)
if err != nil {
	// Handle error
}

// Fetches current orderbook information
ob, err := l.UpdateOrderbook(...)
if err != nil {
	// Handle error
}

// Private calls - wrapper functions - make sure your APIKEY and APISECRET are
// set and AuthenticatedAPISupport is set to true

// Fetches current account information
accountInfo, err := l.GetAccountInfo()
if err != nil {
	// Handle error
}
```

+ If enabled via individually importing package, rudimentary example below:

```go
// Public calls

// Fetches current ticker information
ticker, err := l.GetTicker()
if err != nil {
	// Handle error
}

// Fetches current orderbook information
ob, err := l.GetOrderBook()
if err != nil {
	// Handle error
}

// Private calls - make sure your APIKEY and APISECRET are set and
// AuthenticatedAPISupport is set to true

// GetUserInfo returns account info
accountInfo, err := l.GetUserInfo(...)
if err != nil {
	// Handle error
}

// Submits an order to the exchange and returns its tradeID
tradeID, err := l.Trade(...)
if err != nil {
	// Handle error
}
```

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
