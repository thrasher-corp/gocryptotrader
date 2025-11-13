# GoCryptoTrader Backtester: Binancecashandcarry package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/binancecashandcarry)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This binancecashandcarry package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Binance Cash and carry strategy overview

## Important
This strategy was initially designed for the exchange FTX. It is currently being ported to Binance. It does not work at present.

### Description
Cash and carry is a strategy which takes advantage of the difference in pricing between a long-dated futures contract and a SPOT asset.
By default, this cash and carry strategy will, upon the first data event, purchase BTC-USD SPOT asset from Binance exchange and then, once filled, raise a SHORT for BTC-20210924 FUTURES contract.
On the last event, the strategy will close the SHORT position by raising a LONG of the same contract amount, thereby netting the difference in prices

### Requirements
- At this time of writing, this strategy is only compatible with Binance
- This strategy *requires* `Simultaneous Signal Processing` aka [use-simultaneous-signal-processing](/backtester/config/README.md).
- This strategy *requires* `Exchange Level Funding` aka [use-exchange-level-funding](/backtester/config/README.md).

### Creating a strategy config
- The long-dated futures contract will need to be part of the `currency-settings` of the contract
- Funding for purchasing SPOT assets will need to be part of `funding-settings`
- See the [example config](./config/strategyexamples/binance-cash-carry.strat)

### Customisation
This strategy does support strategy customisation in the following ways:

| Field | Description |  Example |
| --- | ------- | --- |
| openShortDistancePercentage | If there is no short position open, and the difference between FUTURES and SPOT pricing goes above this this percentage threshold, raise a SHORT order of the FUTURES contract | 10 |
| closeShortDistancePercentage | If there is an open SHORT position on a FUTURES contract, and the difference in FUTURES and SPOT pricing goes below this percentage threshold, close the SHORT position | 1 |

### External Resources
- [This](https://ftxcashandcarry.com/) is a very informative site on describing what a cash and carry trade will look like

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
