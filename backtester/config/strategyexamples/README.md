# GoCryptoTrader Backtester: Strategyexamples package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/config/strategyexamples)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This strategyexamples package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

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

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
