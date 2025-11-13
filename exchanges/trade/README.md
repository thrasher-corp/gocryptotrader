# GoCryptoTrader package Trade

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/trade)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This trade package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

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
| Binance.US | Yes  | Yes        | NA  | 
| Binance| Yes  | Yes        | Yes  |
| Bitfinex | Yes  | Yes        | Yes  |
| Bitflyer | Yes  | No      | No  |
| Bithumb | Yes  | Yes       | No  |
| BitMEX | Yes | Yes | Yes |
| Bitstamp | Yes  | Yes       | No  |
| BTCMarkets | Yes | Yes       | No  |
| BTSE | Yes | Yes | No |
| Bybit | Yes | Yes | Yes |
| Coinbase | Yes | Yes | No|
| COINUT | Yes | Yes | No |
| Deribit | Yes | Yes | Yes |
| Exmo | Yes | NA | No |
| GateIO | Yes | Yes | No |
| Gemini | Yes | Yes | Yes |
| HitBTC | Yes | Yes | Yes |
| Huobi.Pro | Yes | Yes | No |
| Kraken | Yes | Yes | No |
| Kucoin | Yes | No | Yes |
| Lbank | Yes | No | Yes |
| Okx | Yes | Yes | Yes |
| Poloniex | Yes | Yes | Yes |
| Yobit | Yes | NA | No |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
