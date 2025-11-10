# GoCryptoTrader OHLCV support

<img src="/docs/assets/page-logo.png" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)

This exchanges package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Wrapper Methods

Candle retrieval is handled by two methods 


GetHistoricCandles which makes a single request to the exchange and follows all exchange limitations
```go
func (b *base) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}
```

GetHistoricCandlesExtended that will make multiple requests to an exchange if the requested periods are outside exchange limits
```go
func (b *base) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}
```

both methods return kline.Item{} 

```go
// Item holds all the relevant information for internal kline elements
type Item struct {
	Exchange string
	Pair     currency.Pair
	Asset    asset.Item
	Interval Interval
	Candles  []Candle
}

// Candle holds historic rate information.
type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}
```

### DBSeed helper

A helper tool [cmd/dbseed](../cmd/dbseed/README.md) has been created for assisting with candle data migration 

## Exchange status
| Exchange       | Supported   | 
|----------------|-------------|
| Binance.US	 | Y		   | 
| Binance        | Y           | 
| Bitfinex       | Y           | 
| Bitflyer       |             | 
| Bithumb        | Y           | 
| Bitmex         |             |        
| Bitstamp       | Y           | 
| BTC Markets    | Y           | 
| BTSE           | Y           |      
| Bybit          | Y           | 
| Coinbase Pro   | Y           |
| Coinut         |             |
| Deribit        | Y            |         
| Exmo           |             |
| GateIO         | Y           |
| Gemini         |             |
| HitBTC         | Y           |     
| Huobi          | Y           |              
| Kraken         | Y           |
| Kucoin         | Y           |                 
| lBank          | Y           |                
| Okx            | Y           |         
| Poloniex       | Y           |          
| Yobit          |             |           
