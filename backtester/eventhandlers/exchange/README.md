# GoCryptoTrader Backtester: Exchange package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This exchange package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Exchange package overview

The exchange eventhandler is responsible for calling the `engine` package's `ordermanager` to place either a fake, or real order on the exchange via API.

The following steps are taken for the `ExecuteOrder` function:

- Calculate slippage. If the order is a sell order, it will reduce the price by a random percentage between the two values. If it is a buy order, it will raise the price by a random percentage between the two values
  - If `RealOrders` is set to `false`:
    - It will estimate the slippage based on what is in the config file under `min-slippage-percent` and `max-slippage-percent`.
    - It will be sized within the constraints of the current candles OHLCV values
    - It will generate the exchange fee based on what is stored in the config for the exchange asset currency pair
  - If `RealOrders` is set to `true`, it will use the latest orderbook data to calculate slippage by simulating the order
 - Place the order with the engine order manager
  - If `RealOrders` is set to `false` it will submit the order with no calls to the exchange's API, use no API credentials and it will always pass
  - If `RealOrders` is set to `true` it will submit the order via the exchange's API and if successful, will be stored in the order manager
 - If an order is successfully placed, a snapshot of all existing orders in the run will be captured and store for statistical purposes

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
