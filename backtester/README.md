# GoCryptoTrader Backtester: Backtester package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This backtester package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)


# GoCryptoTrader Backtester
An event-driven backtesting tool to test and iterate trading strategies using historical or custom data.

## Features
- Works with all GoCryptoTrader exchanges that support trade/candle retrieval. See [candle readme](/docs/OHLCV.md) and [trade readme](/exchanges/trade/README.md) for supported exchanges
- CSV data import
- Database data import
- Proof of concept live data running
- Shopspring decimal implementation to track stats more accurately
- Can run strategies against multiple cryptocurrencies
- Can run strategies that can assess multiple currencies simultaneously to make complex decisions
- Dollar cost strategy example strategies
- RSI example strategy
- MFI example strategy
- Rules customisation via config `.strat` files
- Strategy config builder application
- Strategy customisation without requiring recompilation. For example, customising RSI high, low and length values via config `.strat` files.
- Report generation
- Portfolio manager to help size orders based on config rules, risk and candle volume
- Order manager to place orders with customisable slippage estimator
- Helpful statistics to help determine whether a strategy was effective
- Compliance manager to keep snapshots of every transaction and their changes at every interval
- Exchange level funding allows funding to be shared across multiple currency pairs and to allow for complex strategy design
- Fund transfer. At a strategy level, transfer funds between exchanges to allow for complex strategy design
- Backtesting support for futures asset types
- Example cash and carry spot futures strategy
- Long-running application as a GRPC server
- Custom strategy plugins
- Live data source trading. Traders can move their back tested strategies and use them against current live data

## Planned Features
We welcome pull requests on any feature for the Backtester! We will be especially appreciative of any contribution towards the following planned features:

| Feature | Description |
|---------|-------------|
| Perpetual futures support | Accounting for hourly funding rates in user's overall positions allows for much greater strategic depth |
| Margin borrowing support | Allowing strategies to utilise margin borrowing to have larger positions and handling borrow rate payments |
| Leverage support | Leverage is a good way to enhance profit and loss and is important to include in strategies |
| Live ticker data | A potential feature as live trading works off candle data which is only processed at intervals. Adding ticker data as a strategic source allows for faster decision making |
| Live orderbook data | Processing orders based off the latest orderbook data allows for much more accurate order placement and reduces surprise slippage |
| Enhance config-builder | Create an application that can create strategy configs in a more visual manner and execute them via GRPC to allow for faster customisation of strategies |
| Save Backtester results to database | This will allow for easier comparison of results over time |
| Backtester result comparison report | Providing an executive summary of Backtester database results |


## How does it work?
- The application will load a `.strat` config file as specified at runtime
- The `.strat` config file will contain
  - Start & end dates
  - The strategy to run
  - The candle interval
  - Where the data is to be sourced ([API](/backtester/data/kline/api/README.md), [CSV](/backtester/data/kline/csv/README.md), [database](/backtester/data/kline/database/README.md), [live](/backtester/data/kline/live/README.md))
  - Whether to use trade or candle data ([readme](/backtester/data/kline/README.md))
  - A nickname for the strategy (to help differentiate between runs/configs using the same strategy)
  - The currency/currencies to use
  - The exchange(s) to run against
  - See [readme](/backtester/config/README.md) for a breakdown of all config features
- The GoCryptoTrader Backtester will retrieve the data specified in the config ([readme](/backtester/backtest/README.md))
- The data is converted into candles and each candle is streamed as a data event.
- The data event is analysed by the strategy which will output a purchasing signal such as `BUY`, `SELL` or `DONOTHING` ([readme](/backtester/eventtypes/signal/README.md))
- The purchase signal is then processed by the portfolio manager ([readme](/backtester/eventhandlers/portfolio/README.md)) which will size the order ([readme](/backtester/eventhandlers/portfolio/size/README.md)) and assess risk ([readme](/backtester/eventhandlers/portfolio/risk/README.md)) before sending it to the exchange
- The exchange order event handler will size to the candle data and run a slippage estimator ([readme](/backtester/eventhandlers/exchange/slippage/README.md)) and place the order ([readme](/backtester/eventhandlers/exchange/README.md))
- Upon an order being placed, the order is snapshot for analysis in both the statistics package ([readme](/backtester/eventhandlers/statistics/README.md)) and the report package ([readme](/backtester/report/README.md))


# Cool story, how do I use it?
To run the application using the provided dollar cost average strategy, simply run `go run .` from `gocryptotrader/backtester`. An output of the results will be put in the `results` folder.

# How do I create my own config?
There is a config generating helper application under `/backtester/config/configbuilder` to help you create a `.strat` file. Read more about it [here](/backtester/config/configbuilder/README.md). There are also a number of tests under `/config/config_test.go` which generate configs into the `examples` folder, which if you have code knowledge, can write your own configs programmatically.

# How do I create my own strategy?
Creating strategies requires programming skills. [Here](/backtester/eventhandlers/strategies/README.md) is a readme on the subject. After reading the readmes, please review the strategies [here](/backtester/eventhandlers/strategies/) to gain an understanding on how to write your own.

# How does it work technically?
- The readmes linked in the "How does it work" covers the main parts of the application.
  - If you are still unsure, please raise an issue, ask a question in our Slack or open a pull request
- Here is an overview
![workflow](https://i.imgur.com/Kup6IA9.png)


# Important notes
- This application is not considered production ready and you may experience issues
  - If you encounter any issues, you can raise them in our Slack channel or via Github issues
- **Past performance is no guarantee of future results**
- While an experimental feature, it is **not** recommended to **ever** use live trading and real orders
- **Past performance is no guarantee of future results**

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
