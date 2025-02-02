# GoCryptoTrader Backtester: Trackingcurrencies package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/funding/trackingcurrencies)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This trackingcurrencies package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Trackingcurrencies package overview

### What does the tracking currencies package do?
The tracking currencies package is responsible breaking up a user's strategy currencies into pairs with a USD equivalent pair in order to track strategy performance against a singular currency. For example, you are wanting to backtest on Binance using XRP/DOGE, the tracking currencies will also retrieve XRP/BUSD and DOGE/BUSD pair data for use in calculating how much a currency is worth at every candle point.

### What if the exchange does not support USD?
The tracking currencies package will check supported currencies against a list of USD equivalent USD backed stablecoins. So if your select exchange only supports BUSD or USDT based pairs, then the GoCryptoTrader Backtester will break up config pairs into the equivalent. See below list for currently supported stablecoin equivalency

| Currency |
|----------|
|USD       |
|USDT      |
|BUSD      |
|USDC      |
|DAI       |
|TUSD      |
|ZUSD      |
|PAX       |

### How do I disable this?
If you need to disable this functionality, for example, you are using Live, Database or CSV based trade data, then under `strategy-settings` in your config, set `disable-usd-tracking` to `true`

### Can I supply my own list of equivalent currencies instead of USD?
This is currently not supported. If this is a feature you would like to have, please raise an issue on GitHub or in our Slack channel

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
