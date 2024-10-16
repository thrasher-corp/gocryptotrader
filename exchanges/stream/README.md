# GoCryptoTrader Exchange Stream Package

This package is part of the GoCryptoTrader project and is responsible for handling exchange streaming data.

## Overview

The `stream` package uses Gorilla Websocket and provides functionalities to connect to various cryptocurrency exchanges and handle real-time data streams.

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
Here is a basic example of how to setup the `stream` package for websocket:

```go
package main

import (
    "github.com/thrasher-corp/gocryptotrader/exchanges/stream"
    exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
    "github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

type Exchange struct {
    exchange.Base
}

// In the exchange wrapper this will set up the initial pointer field provided by exchange.Base
func (e *Exchange) SetDefault() {
    e.Websocket = stream.NewWebsocket()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// In the exchange wrapper this is the original setup pattern for the websocket services 
func (e *Exchange) Setup(exch *config.Exchange) error {
    // This sets up global connection, sub, unsub and generate subscriptions for each connection defined below.
    if err := e.Websocket.Setup(&stream.WebsocketSetup{
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
	if err := ok.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		URL:                  connectionURLString,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exchangeWebsocketResponseMaxLimit,
		RateLimit:            request.NewRateLimitWithWeight(time.Second, 2, 1),
	}); err != nil {
		return err
	}

    // This is a private websocket connection 
	return ok.Websocket.SetupNewConnection(&stream.ConnectionSetup{
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
    if err := e.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:               exch,
		Features:                     &e.Features.Supports.WebsocketCapabilities,
		FillsFeed:                    e.Features.Enabled.FillsFeed,
		TradeFeed:                    e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement: true,
	})
	if err != nil {
		return err
	}
	// Spot connection
	err = g.Websocket.SetupNewConnection(&stream.ConnectionSetup{
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
	})
	if err != nil {
		return err
	}
	// Futures connection - USDT margined
	err = g.Websocket.SetupNewConnection(&stream.ConnectionSetup{
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
	})
	if err != nil {
		return err
	}
}
```