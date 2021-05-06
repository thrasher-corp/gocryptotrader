# GoCryptoTrader Backtester: Strategies package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This strategies package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Strategies package overview

Strategies are programmed instruction sets which act upon pricing data. After data has been loaded into the GoCryptoTrader, each tick is passed through your loaded strategy and is analysed in either the `OnSignal` function or the `OnSignals` function.

### Creating strategies
The level customisation allowed in a strategy is extensive. They are required to be written in Golang.
The strategy must adhere to the interface `strategies.Handler` by implementing the function signature `OnSignal(d data.Handler, _ portfolio.Handler) (signal.Event, error)`. The `data.Handler` allows you to access the current pricing information as well as all previous intervals. You can use this to feed any Technical Analysis package to create strategies based on market movements such as RSI (see `./strategies/rsi/rsi.go`). Strategies can also access the portfolio manager on signal(s) which allows analysis of existing holdings value, current orders and positions of other currencies in order to make complex decisions.
When outputting the `signal.Event`, you are not dictating the price of an order, but rather signalling to the portfolio manager what ideally should occur. These options are to buy, sell or do nothing. Additional signals are to flag missing data, handled via checking `d.HasDataAtTime(d.Latest().GetTime()` to prevent any issues from occurring down the line.
Additionally, you can utilise the `AppendWhy()` function to help understand what went into make a signalling decision when reviewing the results.

### What does Simultaneous Signal Processing mean?
GoCryptoTrader Backtester config files may contain multiple `ExchangeSettings` which defined exchange, asset and currency pairs to iterate through a period of time.

If there are multiple entries to `ExchangeSettings` and SimultaneousProcessing is disabled, then each individual exchange, asset and currency pair candle event is evaluated individually and does not know about other exchange, asset and currency pair data events. It is a way to test a singular strategy against multiple assets simultaneously. But it isn't defined as Simultaneous Processing
Simultaneous Signal Processing is a setting which allows multiple `ExchangeSettings` data events for a candle event to be considered simultaneously. This means that you can check if the price of BTC-USDT is 5% greater on Binance than it is on Kraken and choose to make signal a BUY event for Kraken and not Binance.

It allows for complex strategical decisions to be made when you consider the scope of the entire market at a given time, rather than in a vacuum when SimultaneousSignalProcessing is disabled.

### Loading strategies
Each strategy has a unique name and is to be added to the function `getStrategies()` in order to be recognised.

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
