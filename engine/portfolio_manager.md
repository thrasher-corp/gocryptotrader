# GoCryptoTrader package Portfolio Manager

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/engine/portfolio_manager)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This portfolio_manager package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for Portfolio Manager
+ The portfolio manager subsystem is used to synchronise and monitor wallet addresses
+ It can read addresses specified in your config file
+ If you have set API keys for an enabled exchange and enabled `authenticatedSupport`, it will store your exchange addresses
+ In order to modify the behaviour of the portfolio manager subsystem, you can edit the following inside your config file under `portfolioAddresses`:

### portfolioAddresses

| Config | Description | Example |
| ------ | ----------- | ------- |
| Verbose | Enabling this will output more detailed logs to your logging output  |  `false` |
| addresses | An array of portfolio wallet addresses to monitor, see below table |   |

### addresses

| Config | Description | Example |
| ------ | ----------- | ------- |
| Address | The wallet address  |  `bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc` |
| CoinType | The coin for the wallet address | `BTC` |
| Balance | The balance of the wallet |   |
| Description | A customisable description  | `My secret billion stash`  |
| WhiteListed | Determines whether GoCryptoTrader withdraw manager subsystem can make withdrawals from this address | `true` |
| ColdStorage | Describes whether the wallet address is a cold storage wallet eg Ledger | `false`  |
| SupportedExchanges | A comma delimited string of which exchanges are allowed to interact with this wallet | `"Binance"`  |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
