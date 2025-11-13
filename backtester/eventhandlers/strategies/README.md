# GoCryptoTrader Backtester: Strategies package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This strategies package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

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

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
