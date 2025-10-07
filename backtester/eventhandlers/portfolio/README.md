# GoCryptoTrader Backtester: Portfolio package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This portfolio package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Portfolio package overview

The portfolio is one of the most critical packages in the GoCryptoTrader Backtester. It is responsible for making sure that all orders, simulated or otherwise are within all defined risk and sizing rules defined in the config.
The portfolio receives three kinds of events to be processed: `OnSignal`, `OnFill` and `Update`

The following steps are taken for the `OnSignal` function:
- Retrieve previous iteration's holdings data
- If a buy order signal is received, ensure there are enough funds
- If a sell order signal is received, ensure there are any holdings to sell
- If any other direction, return
- The portfolio manager will then size the order according to the exchange asset currency pair's settings along with the portfolio manager's own sizing rules
  - In the event that the order is to large, the sizing package will reduce the order until it fits that limit, inclusive of fees.
  - When an order is sized under the limits, an order event cannot be raised an no order will be submitted by the exchange
  - The portfolio manager's sizing rules override any CurrencySettings' rules if the sizing is outside the portfolio manager's
- The portfolio manager will then assess the risk of the order, it will compare existing holdings and ensures that if an order is placed, it will not go beyond risk rules as defined in the config
- If the risk is too high, the order signal will be changed to `CouldNotBuy`, `CouldNotSell` or `DoNothing`
- If the order is deemed appropriate, the order event will be returned and appended to the event queue for the exchange event handler to run and place the order

The following steps are taken for the `OnFill` function:
- Previous holdings are retrieved and amended with new order information.
  - The stats for the exchange asset currency pair will be updated to reflect the order and pricing
- The order will be added to the compliance manager for analysis in future events or the statistics package

The following steps are taken for the `Update` function:
- The `Update` function is called when orders are not placed, this allows for the portfolio manager to still keep track of pricing and holding statistics, while not needing to process any orders

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
