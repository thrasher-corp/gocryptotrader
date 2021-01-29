# GoCryptoTrader Backtester: Backtester package

<img src="https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://travis-ci.org/thrasher-corp/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-corp/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This backtester package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)


# GoCryptoTrader Backtester
An event-driven backtesting tool to test and iterate trading strategies using historical or custom data.

## Features
- Works with all GoCryptoTrader exchanges that support trade/candle retrieval
- API data retrieval
- CSV data import
- Database data import
- Proof of concept live data running
- Can run strategies against multiple cryptocurrencies
- Can run strategies that can assess multiple currencies simultaneously to make complex decisions
- Dollar cost strategy implementation
- RSI strategy implementation
- Strategy customisation via config `.strat` files
- Report generation
- Portfolio manager to help size orders based on config rules, risk and candle volume
- Order manager to place orders with customisable slippage estimator
- Helpful statistics to help determine whether a strategy was effective
- Compliance manager to keep snapshots of every transaction and their changes at every interval

## How does it work?
- The application will load a `.strat` config file as specified at runtime
- The `.strat` config file will contain
  - Start & end dates
  - The strategy to run
  - The candle interval
  - Where the data is to be sourced ([API](https://github.com/gloriousCode/gocryptotrader/tree/backscratcher/backtester/data/kline/api), [CSV](https://github.com/gloriousCode/gocryptotrader/tree/backscratcher/backtester/data/kline/csv), [database](https://github.com/gloriousCode/gocryptotrader/tree/backscratcher/backtester/data/kline/database), [live](https://github.com/gloriousCode/gocryptotrader/tree/backscratcher/backtester/data/kline/live))
  - Whether to use trade or candle data ([readme](https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/data/kline/README.md))
  - A nickname for the strategy (to help differentiate between runs/configs using the same strategy)
  - The currency/currencies to use
  - The exchange(s) to run against
  - See [readme](https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/config/README.md) for a breakdown of all config features
- The GoCryptoTrader Backtester will retrieve the data specified in the config ([readme](https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/backtest/README.md))
- The data is converted into candles and each candle is streamed as a data event.
- The data event is analysed by the strategy which will output a purchasing signal such as `BUY`, `SELL` or `DONOTHING` ([readme](https://github.com/gloriousCode/gocryptotrader/tree/backscratcher/backtester/eventtypes/signal))
- The purchase signal is then processed by the portfolio manager ([readme](https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/eventhandlers/portfolio/README.md)) which will size the order ([readme](https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/eventhandlers/portfolio/size/README.md)) and assess risk ([readme](https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/eventhandlers/portfolio/risk/README.md)) before sending it to the exchange
- The exchange order event handler will size to the candle data and run a slippage estimator ([readme](https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/eventhandlers/exchange/slippage/README.md) and place the order ([readme](https://github.com/gloriousCode/gocryptotrader/tree/backscratcher/backtester/eventhandlers/exchange))
- Upon an order being placed, the order is snapshot for analysis in both the statistics package ([readme](https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/eventhandlers/statistics/README.md)) and the report package ([readme](https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/report/README.md))


# Cool story, how do I use it?
To run the application using the provided dollar cost average strategy, simply run `go run .` from `gocryptotrader/backtester`. An output of the results will be put in the `results` folder

# How does it work technically?
- The readmes linked in the "How does it work" covers the main parts of the application.
  - If you are still unsure, please raise an issue, ask a question in our Slack or open a pull request
- Here is an overview
![workflow](https://user-images.githubusercontent.com/9261323/104982257-61d97900-5a5e-11eb-930e-3b431d6e6bab.png)


# Important notes
- This application is not considered production ready and you may experience issues
  - If you encounter any issues, you can raise them in our Slack channel or via Github issues
- **Past performance is no guarantee of future results**
- While an experimental feature, it is **not** recommended to **ever** use live trading and real orders
- **Past performance is no guarantee of future results**



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
