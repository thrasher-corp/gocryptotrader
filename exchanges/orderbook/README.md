# GoCryptoTrader package Orderbook

<img src="https://github.com/thrasher-/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://travis-ci.org/thrasher-/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-/gocryptotrader/exchanges/orderbook)
[![Coverage Status](http://codecov.io/github/thrasher-/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-/gocryptotrader)


This orderbook package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progresss on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://gocryptotrader.herokuapp.com/)

## Current Features for orderbook

+ This package facilitates orderbook generation.
+ Attaches methods to an orderbook
  - To Return total Bids
  - To Return total Asks
  - Update orderbooks
+ Gets a loaded orderbook by exchange, asset type and currency pair.

+ This package is primarily used in conjunction with but not limited to the
exchange interface system set by exchange wrapper orderbook functions in
"exchange"_wrapper.go.

Examples below:

```go
ob, err := yobitExchange.FetchOrderbook()
if err != nil {
  // Handle error
}

// Find total asks which also returns total orderbook value
totalAsks, totalOrderbookVal := ob.CalculateTotalAsks()
```

+ or if you have a routine setting an exchange orderbook you can access it via
the package itself.

```go
ob, err := orderbook.GetOrderbook(...)
if err != nil {
  // Handle error
}
```

### Please click GoDocs chevron above to view current GoDoc information for this package

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Donations

<img src="https://github.com/thrasher-/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB***

