# GoCryptoTrader package Trade

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/trade)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This trade package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Current Features for trade

+ The trade package contains a processor for both REST and websocket trade history processing
  + Its primary purpose is to collect trade data from multiple sources and save it to the database's trade table
  + If you do not have database enabled, then trades will not be processed

### Requirements to save a trade to the database
+ Database has to be enabled
+ Under `config.json`, under your selected exchange, enable the field `saveTradeData`
  + This will enable trade processing to occur for that specific exchange
  + This can also be done via gRPC under the `SetExchangeTradeProcessing` command

### Usage
+ To send trade data to be processed, use the following example:
```
err := trade.AddTradesToBuffer(b.Name, trade.Data{
    Exchange:     b.Name,
    TID:          strconv.FormatInt(tradeData[i].TID, 10),
    CurrencyPair: p,
    AssetType:    assetType,
    Side:         side,
    Price:        tradeData[i].Price,
    Amount:       tradeData[i].Amount,
    Timestamp:    tradeTS,
})
```
_b in this context is an `IBotExchange` implemented struct_

### Rules
+ If the trade processor has not started, it will automatically start upon being sent trade data.
+ The processor will add all received trades to a buffer
+ After 15 seconds, the trade processor will parse and save all trades on the buffer to the trade table
  + This is to save on constant writing to the database. Trade data, especially when received via websocket would cause massive issues on the round trip of saving data for every trade
+ If the processor has not received any trades in that 15 second timeframe, it will shut down.
  + Sending trade data to it later will automatically start it up again


## Exchange Support Table

| Exchange | Recent Trades via REST | Live trade updates via Websocket | Trade history via REST |
|----------|------|-----------|-----|
| Alphapoint | No  | No        | No  |
| Binance| Yes  | Yes        | Yes  |
| Bitfinex | Yes  | Yes        | Yes  |
| Bitflyer | Yes  | No      | No  |
| Bithumb | Yes  | NA       | No  |
| BitMEX | Yes | Yes | Yes |
| Bitstamp | Yes  | Yes       | No  |
| Bittrex | Yes | Yes | No |
| BTCMarkets | Yes | Yes       | No  |
| BTSE | Yes | Yes | No |
| Coinbene | Yes | Yes | No |
| CoinbasePro | Yes | Yes | No|
| COINUT | Yes | Yes | No |
| Exmo | Yes | NA | No |
| FTX | Yes | Yes | Yes |
| GateIO | Yes | Yes | No |
| Gemini | Yes | Yes | Yes |
| HitBTC | Yes | Yes | Yes |
| Huobi.Pro | Yes | Yes | No |
| ItBit | Yes | NA | No |
| Kraken | Yes | Yes | No |
| Lbank | Yes | No | Yes |
| LocalBitcoins | Yes | NA | No |
| OKCoin International | Yes | Yes | No |
| OKEX | Yes | Yes | No |
| Poloniex | Yes | Yes | Yes |
| Yobit | Yes | NA | No |
| ZB.COM | Yes | Yes | No |


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
