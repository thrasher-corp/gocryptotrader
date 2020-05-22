# GoCryptoTrader package Exchanges

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">

[![Build Status](https://travis-ci.org/thrasher-corp/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-corp/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)

This exchanges package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Current Features for exchanges

+ This package is used to connect and query data from supported exchanges.

+ Please checkout individual exchange README for more information on
implementation

#### How to add a new exchange

+ 1) run exchange_template.go which automatically creates files n inbuilt functions

###### Linux/OSX
GoCryptoTrader is built using [Go Modules](https://github.com/golang/go/wiki/Modules) and requires Go 1.11 or above
Using Go Modules you now clone this repository **outside** your GOPATH

```bash
git clone https://github.com/thrasher-corp/gocryptotrader.git
cd cmd
cd exchange_template
go build
go run exchange_template.go -name Bitmex -ws -rest
```

###### Windows

```bash
git clone https://github.com/thrasher-corp/gocryptotrader.git
cd gocryptotrader\cmd\exchange_template
go build
go run exchange_template.go -name Bitmex -ws -rest
```

+ 2) add exchange struct to config_example.json, configtest.json (in testdata) & to the main config

###### If main config path is unknown the following function can be used:
```go
config.GetDefaultFilePath()
```

```go
  {
   "name": "FTX",
   "enabled": true,
   "verbose": false,
   "httpTimeout": 15000000000,
   "websocketResponseCheckTimeout": 30000000,
   "websocketResponseMaxLimit": 7000000000,
   "websocketTrafficTimeout": 30000000000,
   "websocketOrderbookBufferLimit": 5,
   "baseCurrencies": "USD",
   "currencyPairs": {
    "requestFormat": {
     "uppercase": false,
     "delimiter": "_"
    },
    "configFormat": {
     "uppercase": true,
     "delimiter": "_"
    },
    "useGlobalFormat": true,
    "assetTypes": [
      "spot",
      "futures"
     ],
     "pairs": {
      "futures": {
       "enabled": "BTC-PERP",
       "available": "BTC-PERP",
       "requestFormat": {
        "uppercase": true,
        "delimiter": "-"
       },
       "configFormat": {
        "uppercase": true,
        "delimiter": "-"
       }
      },
      "spot": {
       "enabled": "BTC/USD",
       "available": "BTC/USD",
       "requestFormat": {
        "uppercase": true,
        "delimiter": "/"
       },
       "configFormat": {
        "uppercase": true,
        "delimiter": "/"
       }
      }
     }
    },
   "api": {
    "authenticatedSupport": false,
    "authenticatedWebsocketApiSupport": false,
    "endpoints": {
     "url": "NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API",
     "urlSecondary": "NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API",
     "websocketURL": "NON_DEFAULT_HTTP_LINK_TO_WEBSOCKET_EXCHANGE_API"
    },
    "credentials": {
     "key": "Key",
     "secret": "Secret"
    },
    "credentialsValidator": {
     "requiresKey": true,
     "requiresSecret": true
    }
   },
   "features": {
    "supports": {
     "restAPI": true,
     "restCapabilities": {
      "tickerBatching": true,
      "autoPairUpdates": true
     },
     "websocketAPI": true,
     "websocketCapabilities": {}
    },
    "enabled": {
     "autoPairUpdates": true,
     "websocketAPI": false
    }
   },
   "bankAccounts": [
    {
     "enabled": false,
     "bankName": "",
     "bankAddress": "",
     "bankPostalCode": "",
     "bankPostalCity": "",
     "bankCountry": "",
     "accountName": "",
     "accountNumber": "",
     "swiftCode": "",
     "iban": "",
     "supportedCurrencies": ""
    }
   ]
  },
```

###### Available pairs will be automatically filled in the configs when wrapper functions are filled out and gocryptotrader is run with the new exchange enabled

+ 3) Add the new exchange to the following files (FTX exchange is used as an example below):

###### root Readme.md:
```go
| Exchange | REST API | Streaming API | FIX API |
|----------|------|-----------|-----|
| Alphapoint | Yes  | Yes        | NA  |
| Binance| Yes  | Yes        | NA  |
| Bitfinex | Yes  | Yes        | NA  |
| Bitflyer | Yes  | No      | NA  |
| Bithumb | Yes  | NA       | NA  |
| BitMEX | Yes | Yes | NA |
| Bitstamp | Yes  | Yes       | No  |
| Bittrex | Yes | No | NA |
| BTCMarkets | Yes | No       | NA  |
| BTSE | Yes | Yes | NA |
| COINUT | Yes | Yes | NA |
| Exmo | Yes | NA | NA |
| FTX | Yes | Yes | No |
| CoinbasePro | Yes | Yes | No|
| Coinbene | Yes | No | No |
| GateIO | Yes | Yes | NA |
| Gemini | Yes | Yes | No |
| HitBTC | Yes | Yes | No |
| Huobi.Pro | Yes | Yes | NA |
| ItBit | Yes | NA | No |
| Kraken | Yes | Yes | NA |
| Lbank | Yes | No | NA |
| LakeBTC | Yes | No | NA |
| LocalBitcoins | Yes | NA | NA |
| OKCoin International | Yes | Yes | No |
| OKEX | Yes | Yes | No |
| Poloniex | Yes | Yes | NA |
| Yobit | Yes | NA | NA |
| ZB.COM | Yes | Yes | NA |
```

###### exchanges\support.go:
```go
var Exchanges = []string{
	"binance",
	"bitfinex",
	"bitflyer",
	"bithumb",
	"bitmex",
	"bitstamp",
	"bittrex",
	"btc markets",
	"btse",
	"coinbasepro",
	"coinbene",
	"coinut",
	"exmo",
	"ftx",
	"gateio",
	"gemini",
	"hitbtc",
	"huobi",
	"itbit",
	"kraken",
	"lakebtc",
	"lbank",
	"localbitcoins",
	"okcoin international",
	"okex",
	"poloniex",
	"yobit",
    "zb",
```

###### exchanges\exchange_test.go:
```go
func TestExchange_Exchanges(t *testing.T) {
	t.Parallel()
	x := exchangeTest.Exchanges(false)
	y := len(x)
	if y != 28 { // add 1 here (before FTX was added it was 27, so 28 now)
		t.Fatalf("expected 28 received %v", y) // add 1 here
	}
}
```

###### cmd\documentation\exchange_templates:

- Create a new file named <exchangename>.tmpl
- Copy contents of template from another exchange example here being Exmo
- Replace names and variables as shown:

```go
{{define "exchanges exmo" -}} // exmo -> ftx
{{template "header" .}}
## Exmo Exchange

### Current Features

+ REST Support

### How to enable

+ [Enable via configuration](https://github.com/thrasher-corp/gocryptotrader/tree/master/config#enable-exchange-via-config-example)

+ Individual package example below:

```go
  // Exchanges will be abstracted out in further updates and examples will be
  // supplied then
```

### How to do REST public/private calls

+ If enabled via "configuration".json file the exchange will be added to the
IBotExchange array in the ```go var bot Bot``` and you will only be able to use
the wrapper interface functions for accessing exchange data. View routines.go
for an example of integration usage with GoCryptoTrader. Rudimentary example
below:

main.go
```go
var e exchange.IBotExchange // e -> f

for i := range bot.Exchanges {
  if bot.Exchanges[i].GetName() == "Exmo" { // Exmo -> FTX
    e = bot.Exchanges[i] // e -> f
  }
}

// Public calls - wrapper functions

// Fetches current ticker information
tick, err := e.FetchTicker() // e -> f (do so for the rest of the functions too)
if err != nil {
  // Handle error
}

// Fetches current orderbook information
ob, err := e.FetchOrderbook()
if err != nil {
  // Handle error
}

// Private calls - wrapper functions - make sure your APIKEY and APISECRET are
// set and AuthenticatedAPISupport is set to true

// Fetches current account information
accountInfo, err := e.GetAccountInfo()
if err != nil {
  // Handle error
}
```

+ If enabled via individually importing package, rudimentary example below:

```go
// Public calls

// Fetches current ticker information
ticker, err := e.GetTicker()
if err != nil {
  // Handle error
}

// Fetches current orderbook information
ob, err := e.GetOrderBook()
if err != nil {
  // Handle error
}

// Private calls - make sure your APIKEY and APISECRET are set and
// AuthenticatedAPISupport is set to true

// GetUserInfo returns account info
accountInfo, err := e.GetUserInfo(...)
if err != nil {
  // Handle error
}

// Submits an order and the exchange and returns its tradeID
tradeID, err := e.Trade(...)
if err != nil {
  // Handle error
}
```

### Please click GoDocs chevron above to view current GoDoc information for this package
{{template "contributions"}}
{{template "donations" .}}
{{end}}
```