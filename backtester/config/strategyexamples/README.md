# GoCryptoTrader Backtester: Strategyexamples package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/config/strategyexamples)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This strategyexamples package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Strategyexamples package overview

### Current Config Examples

| Config | Description |
| --- | ------ |
| dca-api-candles.strat | A simple dollar cost average strategy which makes a purchase on every candle |
| dca-api-candles-multiple-currencies.strat| The same DCA strategy, but applied to multiple currencies |
| dca-api-candles-simultaneous-processing.strat | The same DCA strategy, but uses simultaneous signal processing |
| dca-api-candles-exchange-level-funding.strat| The same DCA strategy, but utilises simultaneous signal processing and a shared pool of funding against multiple currencies |
| dca-api-trades.strat| The same DCA strategy, but sources its candle data from trades |
| dca-candles-live.strat| The same DCA strategy, but utilises live data instead of old data |
| dca-csv-candles.strat | The same DCA strategy, but uses a CSV to source candle data |
| dca-database-candles.strat | The same DCA strategy, but uses a database to retrieve candle data |
| rsi-api-candles.strat | Runs a strategy using rsi figures to make buy or sell orders based on market figures |
| t2b2-api-candles-exchange-funding.strat | Runs a more complex strategy using simultaneous signal processing, exchange level funding and MFI values to make buy or sell signals based on the two strongest and weakest MFI values |
| binance-cash-and-carry.strat | Executes a cash and carry trade on Binance, buying BTC-USD while shorting the long dated futures contract. Is not currently implemented |
| binance-live-cash-and-carry.strat | Executes a cash and carry trade on Binance using realtime 15 second candles, buying BTC-USD while shorting the long dated futures contract. Is not currently implemented |

### Want to make your own configs?
Use the provided config builder under `/backtester/config/configbuilder` or modify tests under `/backtester/config/config_test.go` to generates strategy files quickly

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
