# GoCryptoTrader Unified API

<img src="/docs/assets/page-logo.png" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)

A cryptocurrency trading bot supporting multiple exchanges written in Golang.

**Please note that this bot is under development and is not ready for production!**

## Community

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Unified API

GoCryptoTrader supports a unified API for dealing with exchanges. Each exchange
has its own wrapper file which maps the exchanges own RESTful endpoints into a
standardised way for bot and standalone application usage.

A full breakdown of all the supported wrapper funcs can be found [here.](https://github.com/thrasher-corp/gocryptotrader/blob/master/exchanges/interfaces.go#L21)
Please note that these change on a regular basis as GoCryptoTrader is undergoing
rapid development.

Each exchange supports public API endpoints which don't require any authentication
(fetching ticker, orderbook, trade data) and also private API endpoints (which
require authentication). Some examples include submitting, cancelling and fetching
open orders). To use the authenticated API endpoints, you'll need to set your API
credentials in either the `config.json` file or when you initialise an exchange in
your application, and also have the appropriate key permissions set for the exchange.
Each exchange has a credentials validator which ensures that the API credentials
supplied meet the requirements to make an authenticated request.

## Public API Ticker Example

```go
    var b bitstamp.Bitstamp
    b.SetDefaults()
    ticker, err := b.GetCachedTicker(context.Background(), currency.NewBTCUSD(), asset.Spot)
    if err != nil {
        // Handle error
    }
    fmt.Println(ticker.Last)
```

## Private API Submit Order Example

```go
    var b bitstamp.Bitstamp
    b.SetDefaults()

    // Set default keys 
    b.API.SetKey("your_key") 
    b.API.SetSecret("your_secret") 
    b.API.SetClientID("your_clientid")
    b.API.SetPEMKey("your_PEM_key")
    b.API.SetSubAccount("your_specific_subaccount")

    // Set client/strategy/subsystem specific credentials that will override
    // default credentials.
    // Make a standard context and add credentials to it by using exchange 
    // package helper function DeployCredentialsToContext
    ctx := context.Background() 
    ctx = exchange.DeployCredentialsToContext(ctx, &exchange.Credentials{
        Key:        "your_key",
        Secret:     "your_secret",
        ClientID:   "your_clientid",
        PEMKey:     "your_PEM_key",
        SubAccount: "your_specific_subaccount",
    })


    o := &order.Submit{
        Exchange:  b.Name, // or method GetName() if exchange.IBotInterface
        Pair:      currency.NewBTCUSD(),
        Side:      order.Sell,
        Type:      order.Limit,
        Price:     1000000,
        Amount:    0.1,
        AssetType: asset.Spot,
    }

    // Context will be intercepted when sending an authenticated HTTP request. 
    resp, err := b.SubmitOrder(ctx, o)
    if err != nil {
        // Handle error
    }
    fmt.Println(resp.OrderID)
```
