# GoCryptoTrader package Websocket

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchange/websocket)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This websocket package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Overview

The `websocket` package provides methods to manage connections and subscriptions for exchange websockets.

## Features

- Handle real-time market data streams
- Unified interface for managing data streams
- Multi-connection management - a system that can be used to manage multiple connections to the same exchange
- Connection monitoring - a system that can be used to monitor the health of the websocket connections. This can be used to check if the connection is still alive and if it is not, it will attempt to reconnect
- Traffic monitoring - will reconnect if no message is sent for a period of time defined in your config
- Subscription management - a system that can be used to manage subscriptions to various data streams
- Rate limiting - a system that can be used to rate limit the number of requests sent to the exchange
- Message ID generation - a system that can be used to generate message IDs for websocket requests
- Websocket message response matching - can be used to match websocket responses to the requests that were sent

## Usage

### Default single websocket connection

Example setup for the `websocket` package connection:

```go
package main

import (
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

type Exchange struct {
	exchange.Base
}

// In the exchange wrapper this will set up the initial pointer field provided by exchange.Base
func (e *Exchange) SetDefault() {
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// In the exchange wrapper this is the original setup pattern for the websocket services
func (e *Exchange) Setup(exch *config.Exchange) error {
	// This sets up global connection, sub, unsub and generate subscriptions for each connection defined below.
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		DefaultURL:                             connectionURLString,
		RunningURL:                             connectionURLString,
		Connector:                              e.WsConnect,
		Subscriber:                             e.Subscribe,
		Unsubscriber:                           e.Unsubscribe,
		GenerateSubscriptions:                  e.GenerateDefaultSubscriptions,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		MaxWebsocketSubscriptionsPerConnection: 240,
		OrderbookBufferConfig: buffer.Config{ Checksum: e.CalculateUpdateOrderbookChecksum },
	}); err != nil {
		return err
	}

	// This is a public websocket connection
	if err := ok.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  connectionURLString,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exchangeWebsocketResponseMaxLimit,
		RateLimit:            request.NewRateLimitWithWeight(time.Second, 2, 1),
	}); err != nil {
		return err
	}

	// This is a private websocket connection
	return ok.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  privateConnectionURLString,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exchangeWebsocketResponseMaxLimit,
		Authenticated:        true,
		RateLimit:            request.NewRateLimitWithWeight(time.Second, 2, 1),
	})
}
```

### Multiple websocket connections
 The example below provides the now optional multi connection management system which allows for more connections
 to be maintained and established based off URL, connections types, asset types etc.
```go
func (e *Exchange) Setup(exch *config.Exchange) error {
	// This sets up global connection, sub, unsub and generate subscriptions for each connection defined below.
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exch,
		Features:                     &e.Features.Supports.WebsocketCapabilities,
		FillsFeed:                    e.Features.Enabled.FillsFeed,
		TradeFeed:                    e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement: true,
	}); err != nil {
		return err
	}
	// Spot connection
	if err := g.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                      connectionURLStringForSpot,
		RateLimit:                request.NewWeightedRateLimitByDuration(gateioWebsocketRateLimit),
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         exch.WebsocketResponseMaxLimit,
		// Custom handlers for the specific connection:
		Handler:                  e.WsHandleSpotData,
		Subscriber:               e.SpotSubscribe,
		Unsubscriber:             e.SpotUnsubscribe,
		GenerateSubscriptions:    e.GenerateDefaultSubscriptionsSpot,
		Connector:                e.WsConnectSpot,
		BespokeGenerateMessageID: e.GenerateWebsocketMessageID,
	}); err != nil {
		return err
	}
	// Futures connection - USDT margined
	if err := g.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  connectionURLStringForSpotForFutures,
		RateLimit:            request.NewWeightedRateLimitByDuration(gateioWebsocketRateLimit),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		// Custom handlers for the specific connection:
		Handler: func(ctx context.Context, incoming []byte) error {	return e.WsHandleFuturesData(ctx, incoming, asset.Futures)	},
		Subscriber:               e.FuturesSubscribe,
		Unsubscriber:             e.FuturesUnsubscribe,
		GenerateSubscriptions:    func() (subscription.List, error) { return e.GenerateFuturesDefaultSubscriptions(currency.USDT) },
		Connector:                e.WsFuturesConnect,
		BespokeGenerateMessageID: e.GenerateWebsocketMessageID,
	}); err != nil {
		return err
	}
}
```


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
