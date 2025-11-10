# GoCryptoTrader Backtester: Slippage package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange/slippage)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This slippage package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Slippage package overview

Slippage refers to the difference between the expected price of a trade and the price at which the trade is executed. Slippage is used here to simulate what would occur if trading was live as no perfect conditions exist for placing orders.
Slippage is calculated in two ways in the GoCryptoTrader Backtester

### If `RealOrders` is `true`
- The orderbook is frequently requested during live cycle candle retrieval
- When the order is being calculated in the `ExecuteOrder` eventhandler, it will use the orderbook to simulate placing the order and adjust the order price

### If `RealOrders` is `false`
- The `min-slippage-percent` and `max-slippage-percent` values for the specific exchange, asset and currency pair will be used as bounds to simulate an orderbook using a random number
  - If it is a buy order, it will raise the price by a random percentage between the two values
  - If the order is a sell order, it will reduce the price by a random percentage between the two values

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
