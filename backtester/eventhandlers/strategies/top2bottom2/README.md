# GoCryptoTrader Backtester: Top2bottom2 package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/top2bottom2)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This top2bottom2 package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Top 2 Bottom 2 package overview

The Top 2 Bottom 2 (T2B2) strategy utilises [the gct-ta MFI package](https://github.com/thrasher-corp/gct-ta) to analyse market signals and selects the top and bottom two currencies based on MFI value.
It is a basic example strategy to highlight how the backtester can perform more complex data event signal processing

This strategy *requires* at least 4 exchange currency settings to determine the 4 signals to process
This strategy *requires* `SimultaneousSignalProcessing` aka [use-simultaneous-signal-processing](/backtester/config/README.md).
This strategy does support strategy customisation in the following ways:

| Field | Description |  Example |
| --- | ------- | --- |
|mfi-high| The upper bounds of MFI that when met, will trigger a Sell signal | 70 |
|mfi-low| The lower bounds of MFI that when met, will trigger a Buy signal | 30 |
|mfi-period| The consecutive candle periods used in order to generate a value. All values less than this number cannot output a buy or sell signal | 14 |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
