# GoCryptoTrader package Quickdata

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchange/quickdata)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This quickdata package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Overview

This package provides a means to quickly request and receive data for an individual exchange, asset and currency pair. For the times you just really want to get some data without config and minimal setup.

## Features

- Creates an exchange with only the selected asset and currency pair enabled
- Supports a range of focused data types
- Supports both REST and Websocket data retrieval methods
- Supports both public and authenticated data retrieval methods
- Three types of QuickData implementations:
  - `QuickData` - supports multiple focus data types, has the finest level of control over data retrieval method and frequency
  - `QuickerData` - supports a single focus data type, prioritises websocket and allows for control over the quickData instance
  - `QuickestData` - supports a single focus data type, prioritises websocket and returns a chan of data for the caller to consume


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
| URL | Yes | No | No | No |

## Context

All quickData implementations require a context to run. The context must have a deadline, whether from SIGINT, cancel function or deadline. This allows for graceful shutdown of the instance.
The context can also be used to provide API credentials for authenticated data retrieval methods.
See Usage section below.

## Usage

There are multiple ways to utilise a quickData. See `/cmd/quickData` for a basic way of establishing a single purpose quickData that subscribes to data and prints it to console.

### QuickData with two focus types

```go
func main() {
	isOnceOff := false
	useWebsocket := true
	tickerFocusType := quickdata.NewFocusData(quickdata.TickerFocusType, isOnceOff, useWebsocket, time.Second)
	orderbookFocusType := quickdata.NewFocusData(quickdata.OrderBookFocusType, isOnceOff, useWebsocket, time.Second)
	focusTypes := []*quickdata.FocusData{tickerFocusType, orderbookFocusType}

	k := key.NewExchangeAssetPair("Binance", asset.Spot, currency.NewBTCUSDT())
	qs, err := quickdata.NewQuickData(&k, focusTypes)
	if err != nil {
		log.Fatalf("could not create quickData instance: %v", err)
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
	defer cancel()
	if err := qs.Run(ctx); err != nil {
		log.Fatalf("could not run quickData instance: %v", err)
	}
	fmt.Printf("%+v", <-tickerFocusType.Stream)
	fmt.Printf("%+v", <-orderbookFocusType.Stream)
}
```

### QuickerData for account info focus type with credentials provided by context

```go
func main() {
	k := key.NewExchangeAssetPair("Binance", asset.Spot, currency.NewBTCUSDT())
	qs, err := quickdata.NewQuickerData(&k, quickdata.AccountHoldingsFocusType)
	if err != nil {
		log.Fatalf("could not create quickData instance: %v", err)
	}

	credentials := &account.Credentials{
		Key:    "abc",
		Secret: "123",
	}
	ctx := account.DeployCredentialsToContext(context.Background(), credentials)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := qs.Run(ctx); err != nil {
		log.Fatalf("could not run quickData instance: %v", err)
	}

	if err := qs.WaitForInitialDataWithTimeout(ctx, quickdata.AccountHoldingsFocusType, time.Second*5); err != nil {
		log.Fatalf("could not get initial data: %v", err)
	}
	data, err := qs.LatestData(quickdata.AccountHoldingsFocusType)
	if err != nil {
		log.Fatalf("could not get latest data: %v", err)
	}
	log.Printf("latest data: %+v", data)
}
```

### QuickestData to stream ticker data as fast as possible

```go
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	k := key.NewExchangeAssetPair("Binance", asset.Spot, currency.NewBTCUSDT())
	qs, err := quickdata.NewQuickestData(ctx, &k, quickdata.TickerFocusType)
	if err != nil {
		log.Fatalf("could not create quickData instance: %v", err)
	}
	if err := parseData(ctx, qs); err != nil {
		log.Fatalf("could not parse data: %v", err)
	}
}

func parseData(ctx context.Context, c <-chan any) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data := <-c:
			log.Printf("%+v", data)
		}
	}
}

```




## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
