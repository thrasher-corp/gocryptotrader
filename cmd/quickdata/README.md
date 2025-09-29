# GoCryptoTrader package Quickdata

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/cmd/quickdata)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This quickdata package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)


## Current Features for quickdata
quickdata fetches targeted data from an exchange with minimal setup. It supports both public and authenticated data retrieval methods.

## Usage
```go
go run . --exchange="Binance" --asset="Spot" --pair="BTCUSDT" -data=Ticker
```

```go
go run . --exchange="binance" --asset="spot" --pair="btc-usdt" --data="orders" --apiKey="abc" --apiSecret="123"
```

### Supported Flags:
```go
go run . --help
```

### Data support

| Data Type | Accepted --data flags |
| --------- | -------------- |
| Ticker | "ticker", "tick" |
| Orderbook | "orderbook", "order_book", "ob", "book" |
| Kline | "kline", "candles", "candle", "ohlc" |
| Trades | "trades", "trade" |
| Open Interest | "openinterest", "oi" |
| Funding Rate | "fundingrate", "funding" |
| Account Info | "accountholdings", "account", "holdings", "balances" |
| Active Orders | "activeorders", "orders" |
| Order Execution Limits | "orderexecution", "executionlimits", "limits" |
| URL | "url", "tradeurl", "trade_url" |
| Contract Info | "contract" |


## Further Reading
For more  details about QuickData, see the [quickData package documentation](/exchange/quickData/README.md).

## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
