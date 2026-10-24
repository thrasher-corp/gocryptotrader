# GoCryptoTrader package Mexc

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/mexc)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This mexc package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## MEXC Exchange

### Current Features

+ REST Support
+ Websocket Support
+ Spot and perpetual futures assets; futures pairs are the venue's contract symbols, for example `BTC_USDT` for a USDT margined contract and `BTC_USD` for a coin margined one
+ Futures support for contract details, funding rates, open interest, leverage, positions and position margin

### Futures

+ Order, order book and trade amounts are in contracts, as the venue counts them; a contract's size is in its details, the `Multiplier` of `GetFuturesContractDetails`
+ Ticker volumes, candle volumes and open interest are restated in currency with each contract's size
+ An order without a margin type is isolated, and an isolated order opening a position without leverage takes the account's leverage for that side of the contract
+ A reduce-only order names the position it closes, and takes that position's margin type when it names none
+ `GetHistoricTrades` is not supported for futures, since the venue serves only the latest trades, which `GetRecentTrades` returns

### Subscriptions

Default Public Subscriptions:

+ Orderbook for spot and futures
+ Candles for spot and futures
+ Ticker for spot and futures
+ All trades for spot and futures

Default Authenticated Subscriptions:

+ Account Balance events for spot and futures
+ Order events for spot and futures
+ Trade fill events for spot and futures
+ Position events for futures

Subscriptions are subject to enabled assets and pairs.

### Testing

+ Tests run against recorded mock responses in `testdata/http.json` by default
+ Live tests are enabled with the `mock_test_off` build tag, for example `go test -tags=mock_test_off ./exchanges/mexc/...`
+ Authenticated live tests require credentials in `rest_test.go` or the `MEXC_API_KEY` and `MEXC_API_SECRET` environment variables, and order placement tests also require `canManipulateRealOrders` to be enabled in `rest_test.go`

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
