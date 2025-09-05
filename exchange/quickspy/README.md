# GoCryptoTrader package Quickspy

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchange/quickspy)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This quickspy package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Overview

The `quickspy` package provides a means to quickly request and receive data for an individual exchange, asset and currency pair. For the times you just really want to get some data without fussing about with configs - setting currency pairs or API keys - or `SetDefaults()` functions.

## Features

- Quickly creates an exchange with only the selected asset and currency pair enabled
- Supports a range of focus data types allowing for a more tailored approach to data retrieval
- Supports both REST and Websocket data retrieval methods
- Supports both public and authenticated data retrieval methods


### Focus Data Types
| Type | Supports REST | Supports Websocket | Futures Only | Requires Authentication |
| ---- | ------------- | ------------------ | ------------ | ----------------------- |
| Orderbook | Yes | Yes | No | No |
| Ticker | Yes | Yes | No | No |
| Trades | Yes | Yes | No | No |
| Funding Rates | Yes | Yes | Yes | No |
| Klines | Yes | Yes | No | No |
| Account Info | Yes | Yes | No | Yes |
| Open Interest | Yes | Yes | Yes | No |
| Active Orders | Yes | Yes | No | Yes |
| Order Execution Limits | Yes | No | No | No |
| Contract Info | Yes | No | Yes | No |
| URL | No | Yes | No | No |

## Usage

There are multiple ways to utilise a quickspy. See `/cmd/quickspy` for a basic way of establishing a single purpose quickspy that subscribes to data and prints it to console.

### Utilise QuickestSpy to fetch and print the latest ticker data
```go
func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	q, err := quickspy.NewQuickestSpy(ctx, "Binance", asset.Spot, currency.NewBTCUSDT(), quickspy.TickerFocusType, nil)
	if err != nil {
		log.Fatal(err)
	}
	if err := q.WaitForInitialData(ctx, quickspy.TickerFocusType); err != nil {
		log.Fatal(err)
	}
	d, err := q.LatestData(quickspy.TickerFocusType)
	if err != nil {
		log.Fatal(err)
	}
	b, err := json.MarshalIndent(d, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", b)
}
```

### Stream gateio websocket orderbook
```go
func main() {
	focusData := quickspy.NewFocusData(quickspy.OrderbookFocusType, false, true, time.Second)
	focusList := []*quickspy.FocusData{focusData}
	k := &quickspy.CredentialsKey{
		ExchangeAssetPair: key.NewExchangeAssetPair("gateio", asset.Spot, currency.NewBTCUSDT()),
	}
	q, err := quickspy.NewQuickSpy(
		nil,
		k,
		focusList)
	if err != nil {
		log.Fatal(err)
	}
	go streamData(&focusData)
	_ = signaler.WaitForInterrupt()
	q.Shutdown()
}

func streamData(fd *quickspy.FocusData) {
	for {
		fmt.Printf("%+v\n", <-fd.Stream)
	}
}

```

### View JSON data
``` go
func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	focusData := quickspy.NewFocusData(quickspy.TickerFocusType, false, true, time.Second)
	focusList := []*quickspy.FocusData{focusData}
	k := &quickspy.CredentialsKey{
		ExchangeAssetPair: key.NewExchangeAssetPair("binance", asset.Spot, currency.NewBTCUSDT()),
	}
	q, err := quickspy.NewQuickSpy(
		ctx,
		k,
		focusList)
	if err != nil {
		log.Fatal(err)
	}
	if err := q.WaitForInitialData(ctx, quickspy.TickerFocusType); err != nil {
		log.Fatal(err)
	}
	d, err := q.DumpJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", d)
}

```



## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
