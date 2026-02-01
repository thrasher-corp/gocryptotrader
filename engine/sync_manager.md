# GoCryptoTrader package Sync Manager

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/engine/sync_manager)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This sync_manager package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for Sync Manager
+ The currency pair syncer subsystem is used to keep all trades, tickers and orderbooks up to date for all enabled exchange asset currency pairs
+ It can sync data via a websocket connection or REST and will switch between them if there has been no updates
+ In order to modify the behaviour of the currency pair syncer subsystem, you can change runtime parameters as detailed below:

| Config | Description | Example |
| ------ | ----------- | ------- |
| syncmanager | Determines whether the subsystem is enabled | `true` |
| tickersync |  Enables ticker syncing for all enabled exchanges |   `true`|
| orderbooksync | Enables orderbook syncing for all enabled exchanges |  `true` |
| tradesync | Enables trade syncing for all enabled exchanges |  `true` |
| syncworkers | The amount of workers (goroutines) to use for syncing exchange data | `15` |
| synccontinuously | Whether to sync exchange data continuously (ticker, orderbook and trades) | `true` |
| synctimeout | The amount of time in golang `time.Duration` format before the syncer will switch from one protocol to the other (e.g. from REST to websocket) | `15000000000` |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
