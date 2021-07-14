# GoCryptoTrader ADD NEW EXCHANGE

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)

This exchanges package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## How to add a new exchange

This document is from a perspective of adding a new exchange called FTX to the codebase:

### Run the [exchange templating tool](../cmd/exchange_template/) which will create a base exchange package based on the features the exchange supports

#### Linux/OSX
GoCryptoTrader is built using [Go Modules](https://github.com/golang/go/wiki/Modules) and requires Go 1.11 or above
Using Go Modules you now clone this repository **outside** your GOPATH

```bash
git clone https://github.com/thrasher-corp/gocryptotrader.git
cd gocryptotrader/cmd/exchange_template
go run exchange_template.go -name FTX -ws -rest
```

#### Windows

```bash
git clone https://github.com/thrasher-corp/gocryptotrader.git
cd gocryptotrader\cmd\exchange_template
go run exchange_template.go -name FTX -ws -rest
```

### Add exchange struct to [config_example.json](../config_example.json), [configtest.json](../testdata/configtest.json):

Find out which asset types are supported by the exchange and add them to the pairs struct (spot is enabled by default)

#### If main config path is unknown the following function can be used:
```go
config.GetDefaultFilePath()
```

```js
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
    "pairs": {
     "futures": {
      "assetEnabled": true,
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
      "assetEnabled": true,
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

#### Configs can be updated automatically by running the following command:

Check to make sure that the command does not override the NTP client and encrypt config default settings:

```bash
go build && gocryptotrader.exe --config=config_example.json
```

### Add the currency pair format structs in ftx_wrapper.go:

#### Futures currency support:

Similar to the configs, spot support is inbuilt but other asset types will need to be manually supported

```go
	spot := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		},
	}
	futures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
	}

	err := f.StoreAssetPairFormat(asset.Spot, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = f.StoreAssetPairFormat(asset.Futures, futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
```

### Document the addition of the new exchange (FTX exchange is used as an example below):

Yes means supported, No means not yet implemented and NA means protocol unsupported

#### Add exchange to the root [readme](../README.md) file:
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
| Bittrex | Yes | Yes | NA |
| BTCMarkets | Yes | No       | NA  |
| BTSE | Yes | Yes | NA |
| COINUT | Yes | Yes | NA |
| Exmo | Yes | NA | NA |
| FTX | Yes | Yes | No | // <-------- new exchange
| CoinbasePro | Yes | Yes | No|
| Coinbene | Yes | No | No |
| GateIO | Yes | Yes | NA |
| Gemini | Yes | Yes | No |
| HitBTC | Yes | Yes | No |
| Huobi.Pro | Yes | Yes | NA |
| ItBit | Yes | NA | No |
| Kraken | Yes | Yes | NA |
| Lbank | Yes | No | NA |
| LocalBitcoins | Yes | NA | NA |
| OKCoin International | Yes | Yes | No |
| OKEX | Yes | Yes | No |
| Poloniex | Yes | Yes | NA |
| Yobit | Yes | NA | NA |
| ZB.COM | Yes | Yes | NA |
```

#### Add exchange to the list of [supported exchanges](../exchanges/support.go):
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
	"ftx", // <-------- new exchange
	"gateio",
	"gemini",
	"hitbtc",
	"huobi",
	"itbit",
	"kraken",
	"lbank",
	"localbitcoins",
	"okcoin international",
	"okex",
	"poloniex",
	"yobit",
    "zb",
```

#### Increment the default number of supported exchanges in [config/config_test.go](../config/config_test.go):
```go
func TestGetEnabledExchanges(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"TestGetEnabledExchanges. LoadConfig Error: %s", err.Error(),
		)
	}

	exchanges := cfg.GetEnabledExchanges()
	if len(exchanges) != defaultEnabledExchanges { // modify the value of defaultEnabledExchanges at the top of the config_test.go file to match the total count of exchanges
		t.Error(
			"TestGetEnabledExchanges. Enabled exchanges value mismatch",
		)
	}

	if !common.StringDataCompare(exchanges, "Bitfinex") {
		t.Error(
			"TestGetEnabledExchanges. Expected exchange Bitfinex not found",
		)
	}
}
```

#### Increment the number of supported exchanges in [the gctscript exchange wrapper test file](../gctscript/wrappers/gct/exchange/exchange_test.go):
```go
func TestExchange_Exchanges(t *testing.T) {
	t.Parallel()
	x := exchangeTest.Exchanges(false)
	y := len(x)
	expected := 28 // modify this value to match the total count of exchanges
	if y != expected {
    	t.Fatalf("expected %v received %v", expected , y)
	}
}
```

#### Setup and run the [documentation tool](../cmd/documentation):

- Create a new file named *exchangename*.tmpl
- Copy contents of template from another exchange example here being Exmo
- Replace names and variables as shown:

```go
{{define "exchanges exmo" -}} // exmo -> ftx
{{template "header" .}}
## Exmo Exchange

#### Current Features

+ REST Support // if websocket or fix are supported, add that in too
```

```go
var e exchange.IBotExchange // We name the exchange.IBotExchange variable after the first character of the exchange, eg f for FTX. e -> f

for i := range bot.Exchanges {
  if bot.Exchanges[i].GetName() == "Exmo" { // Exmo -> FTX
    e = bot.Exchanges[i] // e -> f
  }
}

// Public calls - wrapper functions

// Fetches current ticker information
tick, err := e.FetchTicker() // e -> f 
if err != nil {
  // Handle error
}

// Fetches current orderbook information
ob, err := e.FetchOrderbook() // e -> f (do so for the rest of the functions too)
if err != nil {
  // Handle error
}
```

- Run documentation.go to generate readme file for the exchange:
```bash
cd gocryptotrader\cmd\documentation
go run documentation.go
```

This will generate a readme file for the exchange which can be found in the new exchange's folder

### Create functions supported by the exchange:

#### Requester functions:

```go
// SendHTTPRequest sends an unauthenticated HTTP request
func (f *FTX) SendHTTPRequest(path string, result interface{}) error {
	return f.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          path,
		Result:        result,
		Verbose:       f.Verbose,
		HTTPDebugging: f.HTTPDebugging,
		HTTPRecording: f.HTTPRecording,
	})
}
```

#### Unauthenticated Functions:

https://docs.ftx.com/#get-markets

Create a type struct in types.go for the response type shown on the documentation website:

For efficiency, a JSON to Golang converter can be used: https://mholt.github.io/json-to-go/.
However, great care must be taken as to the values which are autogenerated. The JSON converter tool will default to whatever type it detects, but ultimately conversions to a more useful variable type would be better. For example, price and quantity on some exchange API's provide these as strings. Internally, it would be better if they're converted to the more useful float64 var type.

```go
// MarketData stores market data
type MarketData struct {
	Name           string  `json:"name"`
	BaseCurrency   string  `json:"baseCurrency"`
	QuoteCurrency  string  `json:"quoteCurrency"`
	MarketType     string  `json:"type"`
	Underlying     string  `json:"underlying"`
	Enabled        bool    `json:"enabled"`
	Ask            float64 `json:"ask"`
	Bid            float64 `json:"bid"`
	Last           float64 `json:"last"`
	PriceIncrement float64 `json:"priceIncrement"`
	SizeIncrement  float64 `json:"sizeIncrement"`
}
```

Create new consts to define endpoint strings, they are created at the top of ftx.go file:
```go
const (
	ftxAPIURL = "https://ftx.com/api"

	// Public endpoints
	getMarkets           = "/markets"
	getMarket            = "/markets/"
	getOrderbook         = "/markets/%s/orderbook?depth=%s"
	getTrades            = "/markets/%s/trades?"
	getHistoricalData    = "/markets/%s/candles?"
	getFutures           = "/futures"
	getFuture            = "/futures/"
	getFutureStats       = "/futures/%s/stats"
	getFundingRates      = "/funding_rates"
  	getAllWallegetAllWalletBalances = "/wallet/all_balances"
  ```

Create a get function in ftx.go file and unmarshall the data in the created type:
```go
// GetMarkets gets market data
func (f *FTX) GetMarkets() (Markets, error) {
	var resp Markets
	return resp, f.SendHTTPRequest(ftxAPIURL+getMarkets, &resp)
}
```

Create a test function in ftx_test.go to see if the data is received and unmarshalled correctly
```go
const(
	spotPair = "FTT/BTC"
)

func TestGetMarket(t *testing.T) {
	t.Parallel() // adding t.Parralel() is preferred as it allows tests to run simultaneously, speeding up package test time
	f.Verbose = true // used for more detailed output
	a, err := f.GetMarket(spotPair) // spotPair is just a const so it can be reused in other tests too
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}
```
Verbose can be set to true to see the data received if there are errors unmarshalling
Once testing is done remove verbose, variable a and t.Log(a) since they produce unnecessary output when GCT is run
```go
_, err := f.GetMarket(spotPair)
```

Ensure each endpoint is implemented and has an associated test to improve test coverage and increase confidence

#### Authenticated functions:

Authenticated request function is created based on the way the exchange documentation specifies: https://docs.ftx.com/#authentication
```go
// SendAuthHTTPRequest sends an authenticated request
func (f *FTX) SendAuthHTTPRequest(method, path string, data, result interface{}) error {
	ts := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	var body io.Reader
	var hmac, payload []byte
	var err error
	if data != nil {
		payload, err = json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
		sigPayload := ts + method + "/api" + path + string(payload)
		hmac = crypto.GetHMAC(crypto.HashSHA256, []byte(sigPayload), []byte(f.API.Credentials.Secret))
	} else {
		sigPayload := ts + method + "/api" + path
		hmac = crypto.GetHMAC(crypto.HashSHA256, []byte(sigPayload), []byte(f.API.Credentials.Secret))
	}
	headers := make(map[string]string)
	headers["FTX-KEY"] = f.API.Credentials.Key
	headers["FTX-SIGN"] = crypto.HexEncodeToString(hmac)
	headers["FTX-TS"] = ts
	headers["Content-Type"] = "application/json"
	return f.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          ftxAPIURL + path,
		Headers:       headers,
		Body:          body,
		Result:        result,
		AuthRequest:   true,
		Verbose:       f.Verbose,
		HTTPDebugging: f.HTTPDebugging,
		HTTPRecording: f.HTTPRecording,
	})
}
```

To test authenticated functions, you must have an account with API keys and SendAuthHTTPRequest must be implemented.

HTTP Mocking framework can also be added for the exchange. For reference, please see the [HTTP mock](../testdata/http_mock) package.

Create authenticated functions and test along the way similar to the functions above:

https://docs.ftx.com/#get-account-information:

```go
// GetAccountInfo gets account info
func (f *FTX) GetAccountInfo() (AccountData, error) {
	var resp AccountData
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getAccountInfo, nil, &resp)
}
```

Get Request params for authenticated requests are sent through url.Values{}:

https://docs.ftx.com/#get-withdrawal-history:

```go
// GetTriggerOrderHistory gets trigger orders that are currently open
func (f *FTX) GetTriggerOrderHistory(marketName string, startTime, endTime time.Time, side, orderType, limit string) (TriggerOrderHistory, error) {
	var resp TriggerOrderHistory
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getTriggerOrderHistory+params.Encode(), nil, &resp)
}
```

https://docs.ftx.com/#place-order


Structs for unmarshalling the data are made exactly the same way as the previous functions.

```go
type OrderData struct {
	CreatedAt     time.Time `json:"createdAt"`
	FilledSize    float64   `json:"filledSize"`
	Future        string    `json:"future"`
	ID            int64     `json:"id"`
	Market        string    `json:"market"`
	Price         float64   `json:"price"`
	AvgFillPrice  float64   `json:"avgFillPrice"`
	RemainingSize float64   `json:"remainingSize"`
	Side          string    `json:"side"`
	Size          float64   `json:"size"`
	Status        string    `json:"status"`
	OrderType     string    `json:"type"`
	ReduceOnly    bool      `json:"reduceOnly"`
	IOC           bool      `json:"ioc"`
	PostOnly      bool      `json:"postOnly"`
	ClientID      string    `json:"clientId"`
}

// PlaceOrder stores data of placed orders
type PlaceOrder struct {
	Success bool      `json:"success"`
	Result  OrderData `json:"result"`
}
```

For `POST` or `DELETE` requests, params are sent through a map[string]interface{}:

```go
// Order places an order
func (f *FTX) Order(marketName, side, orderType, reduceOnly, ioc, postOnly, clientID string, price, size float64) (PlaceOrder, error) {
	req := make(map[string]interface{})
	req["market"] = marketName
	req["side"] = side
	req["price"] = price
	req["type"] = orderType
	req["size"] = size
	if reduceOnly != "" {
		req["reduceOnly"] = reduceOnly
	}
	if ioc != "" {
		req["ioc"] = ioc
	}
	if postOnly != "" {
		req["postOnly"] = postOnly
	}
	if clientID != "" {
		req["clientID"] = clientID
	}
	var resp PlaceOrder
	return resp, f.SendAuthHTTPRequest(http.MethodPost, placeOrder, req, &resp)
}
```

### Implementing wrapper functions:

Wrapper functions are the interface in which the GoCryptoTrader engine communicates with an exchange for receiving data and sending requests. A breakdown of all API functions can be found [here](../exchanges/interfaces.go).
The exchanges may not support all the functionality in the wrapper, so fill out the ones that are supported as shown in the examples below:

Unsupported Example:

```go
// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (f *FTX) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	var resp *withdraw.ExchangeResponse
	return resp, common.ErrFunctionNotSupported
}
```

Supported Examples:

```go
// FetchTradablePairs returns a list of the exchanges tradable pairs
func (f *FTX) FetchTradablePairs(a asset.Item) ([]string, error) {
	if !f.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, f.Name)
	}
	markets, err := f.GetMarkets()
	if err != nil {
		return nil, err
	}
	var pairs []string
	switch a {
	case asset.Spot:
		for x := range markets.Result {
			if markets.Result[x].MarketType == spotString {
				pairs = append(pairs, markets.Result[x].Name)
			}
		}
	case asset.Futures:
		for x := range markets.Result {
			if markets.Result[x].MarketType == futuresString {
				pairs = append(pairs, markets.Result[x].Name)
			}
		}
	}
	return pairs, nil
}
```

Wrapper functions on most exchanges are written in similar ways so other exchanges can be used as a reference.

Many helper functions defined in [exchange.go](../exchanges/exchange.go) can be useful when implementing wrapper functions. See examples below:

```go
f.FormatExchangeCurrency(p, a) // Formats the currency pair to the style accepted by the exchange. p is the currency pair & a is the asset type

f.SupportsAsset(a) // Checks if an asset type is supported by the bot

f.GetPairAssetType(p) // Returns the asset type of currency pair p
```

The currency package contains many helper functions to format and process currency pairs. See [currency](../currency/README.md).

### Websocket addition if exchange supports it:

#### Websocket Setup:

- Set the websocket url in ftx_websocket.go that is provided in the documentation:

```go
	ftxWSURL          = "wss://ftx.com/ws/"
```

#### Complete WsConnect function:

```go
// WsConnect connects to a websocket feed
func (f *FTX) WsConnect() error {
	if !f.Websocket.IsEnabled() || !f.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := f.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	// Can set up custom ping handler per websocket connection.
	f.Websocket.Conn.SetupPingHandler(wshandler.WebsocketPingHandler{
		MessageType: websocket.PingMessage,
		Delay:       ftxWebsocketTimer,
	})
	if f.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", f.Name)
	}
	// This reader routine is called prior to initiating a subscription for
	// efficient processing.
	go f.wsReadData()
	if f.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err = f.WsAuth()
		if err != nil {
			f.Websocket.DataHandler <- err
			f.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	// Generates the default subscription set, based off enabled pairs.
	subs, err := f.GenerateDefaultSubscriptions()
	if err != nil {
		return err
	}
	// Finally subscribes to each individual channel.
	return f.Websocket.SubscribeToChannels(subs)
}
```

- Create function to generate default subscriptions:

```go
// GenerateDefaultSubscriptions generates default subscription
func (f *FTX) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	subscriptions = append(subscriptions, stream.ChannelSubscription{
		Channel: wsMarkets,
	})
	// Ranges over available channels, pairs and asset types to produce a full
	// subscription list.
	var channels = []string{wsTicker, wsTrades, wsOrderbook}
	assets := f.GetAssetTypes()
	for a := range assets {
		pairs, err := f.GetEnabledPairs(assets[a])
		if err != nil {
			return nil, err
		}
		for z := range pairs {
			newPair := currency.NewPairWithDelimiter(pairs[z].Base.String(),
				pairs[z].Quote.String(),
				"-")
			for x := range channels {
				subscriptions = append(subscriptions,
					stream.ChannelSubscription{
						Channel:  channels[x],
						Currency: newPair,
						Asset:    assets[a],
					})
			}
		}
	}
	// Appends authenticated channels to the subscription list
	if f.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		var authchan = []string{wsOrders, wsFills}
		for x := range authchan {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: authchan[x],
			})
		}
	}
	return subscriptions, nil
}
```

- To receive data from websocket, a subscription needs to be made with one or more of the available channels:

- Set channel names as consts for ease of use:

```go
	wsTicker          = "ticker"
	wsTrades          = "trades"
	wsOrderbook       = "orderbook"
	wsMarkets         = "markets"
	wsFills           = "fills"
	wsOrders          = "orders"
	wsUpdate          = "update"
	wsPartial         = "partial"
```

- Create subscribe function with the data provided by the exchange documentation:

https://docs.ftx.com/#request-process

- Create a struct required to subscribe to channels:

```go
// WsSub has the data used to subscribe to a channel
type WsSub struct {
	Channel   string `json:"channel,omitempty"`
	Market    string `json:"market,omitempty"`
	Operation string `json:"op,omitempty"`
}
```

- Create the subscription function:

```go
// Subscribe sends a websocket message to receive data from the channel
func (f *FTX) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	// For subscriptions we try to batch as much as possible to limit the amount
	// of connection usage but sometimes this is not supported on the exchange 
	// API.
	var errs common.Errors // This is an array of errors useful in the event that one channel subscription errors but we can subscribe to the next iteration.
channels:
	for i := range channelsToSubscribe {
		// Type we declared above to send via our websocket connection.
		var sub WsSub
		sub.Channel = channelsToSubscribe[i].Channel
		sub.Operation = subscribe

		switch channelsToSubscribe[i].Channel {
		case wsFills, wsOrders, wsMarkets:
		// Authenticated wsFills && wsOrders or wsMarkets which is a channel subscription for the full set of tradable markets do not need a currency pair association. 
		default:
			a, err := f.GetPairAssetType(channelsToSubscribe[i].Currency)
			if err != nil {
				errs = append(errs, err)
				continue channels
			}
			// Ensures our outbound currency pair is formatted correctly, sometimes our configuration format is different from what our request format needs to be.
			formattedPair, err := f.FormatExchangeCurrency(channelsToSubscribe[i].Currency, a)
			if err != nil {
				errs = append(errs, err)
				continue channels
			}
			sub.Market = formattedPair.String()
		}
		err := f.Websocket.Conn.SendJSONMessage(sub)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		// When we have a successful subscription, we can alert our internal management system of the success.
		f.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}
```

- Test subscriptions and check to see if data is received from websocket:

Run gocryptotrader with the following settings enabled in config

```go
     "websocketAPI": true,
     "websocketCapabilities": {}
    },
    "enabled": {
     "autoPairUpdates": true,
	 "websocketAPI": true // <- Change this to true if it is false
```

#### Handle websocket data:

- Function to read data received from websocket:

```go
// wsReadData gets and passes on websocket messages for processing
func (f *FTX) wsReadData() {
	f.Websocket.Wg.Add(1)
	defer f.Websocket.Wg.Done()

	for {
		select {
		case <-f.Websocket.ShutdownC:
			return
		default:
			resp := f.Websocket.Conn.ReadMessage()
			if resp.Raw == nil {
				return
			}

			err := f.wsHandleData(resp.Raw)
			if err != nil {
				f.Websocket.DataHandler <- err
			}
		}
	}
}
```

- Simple Examples of data handling:

1. Create the main struct used for unmarshalling data

2. Unmarshall the data into the overarching result type

```go
// WsResponseData stores basic ws response data on being subscribed to a channel successfully
type WsResponseData struct {
	ResponseType string      `json:"type"`
	Channel      string      `json:"channel"`
	Market       string      `json:"market"`
	Data         interface{} `json:"data"`
}
```

- Unmarshall the raw data into the main type:

```go
	var result map[string]interface{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
  }
```

Using switch cases and types created earlier, unmarshall the data into the more specific structs.
There are some built in structs in wshandler which are used to store the websocket data such as wshandler.TradeData or wshandler.KlineData.
If a suitable struct does not exist in wshandler, wrapper types are the next preference to store the data such as in the market channel example given below:

```go
	switch result["channel"] {
	case wsTicker:
		var resultData WsTickerDataStore
		err = json.Unmarshal(respRaw, &resultData)
		if err != nil {
			return err
		}
		f.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: f.Name,
			Bid:          resultData.Ticker.Bid,
			Ask:          resultData.Ticker.Ask,
			Last:         resultData.Ticker.Last,
			LastUpdated:  timestampFromFloat64(resultData.Ticker.Time),
			Pair:         p,
			AssetType:    a,
	  }
```

If neither of those provide a suitable struct to store the data in, the data can just be passed onto wshandler without any further changes:

```go
		case wsFills:
			var resultData WsFillsDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
      f.Websocket.DataHandler <- resultData.FillsData
```

- Data Handling can be tested offline similar to the following example:

```go
func TestParsingWSOrdersData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	data := []byte(`{
		"channel": "orders",
		"data": {
		  "id": 24852229,
		  "clientId": null,
		  "market": "BTC-PERP",
		  "type": "limit",
		  "side": "buy",
		  "size": 42353.0,
		  "price": 0.2977,
		  "reduceOnly": false,
		  "ioc": false,
		  "postOnly": false,
		  "status": "closed",
		  "filledSize": 0.0,
		  "remainingSize": 0.0,
		  "avgFillPrice": 0.2978
		},
		"type": "update"
	  }`)
	err := f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
}
```

- Create types given in the documentation to unmarshall the streamed data:

https://docs.ftx.com/#fills-2

```go
// WsFills stores websocket fills' data
type WsFills struct {
	Fee       float64   `json:"fee"`
	FeeRate   float64   `json:"feeRate"`
	Future    string    `json:"future"`
	ID        int64     `json:"id"`
	Liquidity string    `json:"liquidity"`
	Market    string    `json:"market"`
	OrderID   int64     `json:"int64"`
	TradeID   int64     `json:"tradeID"`
	Price     float64   `json:"price"`
	Side      string    `json:"side"`
	Size      float64   `json:"size"`
	Time      time.Time `json:"time"`
	OrderType string    `json:"orderType"`
}

// WsFillsDataStore stores ws fills' data
type WsFillsDataStore struct {
	Channel     string  `json:"channel"`
	MessageType string  `json:"type"`
	FillsData   WsFills `json:"fills"`
}
```

- Create the authentication function based on specifications provided in the documentation:

https://docs.ftx.com/#private-channels

```go
// WsAuth sends an authentication message to receive auth data
func (f *FTX) WsAuth() error {
	intNonce := time.Now().UnixNano() / 1000000
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte(strNonce+"websocket_login"),
		[]byte(f.API.Credentials.Secret),
	)
	sign := crypto.HexEncodeToString(hmac)
	req := Authenticate{Operation: "login",
		Args: AuthenticationData{
			Key:  f.API.Credentials.Key,
			Sign: sign,
			Time: intNonce,
		},
	}
	return f.Websocket.Conn.SendJSONMessage(req)
}
```

- Create an unsubscribe function if the exchange has the functionality:

```go
// Unsubscribe sends a websocket message to stop receiving data from the channel
func (f *FTX) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	// As with subscribing we want to batch as much as possible, but sometimes this cannot be achieved due to API shortfalls. 
	var errs common.Errors
channels:
	for i := range channelsToUnsubscribe {
		var unSub WsSub
		unSub.Operation = unsubscribe
		unSub.Channel = channelsToUnsubscribe[i].Channel
		switch channelsToUnsubscribe[i].Channel {
		case wsFills, wsOrders, wsMarkets:
		default:
			a, err := f.GetPairAssetType(channelsToUnsubscribe[i].Currency)
			if err != nil {
				errs = append(errs, err)
				continue channels
			}

			formattedPair, err := f.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, a)
			if err != nil {
				errs = append(errs, err)
				continue channels
			}
			unSub.Market = formattedPair.String()
		}
		err := f.Websocket.Conn.SendJSONMessage(unSub)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		// When we have a successful unsubscription, we can alert our internal management system of the success.
		f.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}
```

- Complete websocket setup in wrapper:

Add websocket functionality if supported to Setup:

```go
// Setup takes in the supplied exchange configuration details and sets params
func (f *FTX) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		f.SetEnabled(false)
		return nil
	}

	err := f.SetupDefaults(exch)
	if err != nil {
		return err
	}

	// Websocket details setup below
	err = f.Websocket.Setup(&stream.WebsocketSetup{
		Enabled:                          exch.Features.Enabled.Websocket,
		Verbose:                          exch.Verbose,
		AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
		WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
		DefaultURL:                       ftxWSURL, // Default ws endpoint so we can roll back via CLI if needed.
		ExchangeName:                     exch.Name, // Sets websocket name to the exchange name.
		RunningURL:                       exch.API.Endpoints.WebsocketURL,
		Connector:                        f.WsConnect, // Connector function outlined above.
		Subscriber:                       f.Subscribe, // Subscriber function outlined above.
		UnSubscriber:                     f.Unsubscribe, // Unsubscriber function outlined above.
		GenerateSubscriptions:            f.GenerateDefaultSubscriptions, // GenerateDefaultSubscriptions function outlined above.
		Features:                         &f.Features.Supports.WebsocketCapabilities, // Defines the capabilities of the websocket outlined in supported features struct. This allows the websocket connection to be flushed appropriately if we have a pair/asset enable/disable change. This is outlined below.

		// Orderbook buffer specific variables for processing orderbook updates via websocket feed. 
		OrderbookBufferLimit:             exch.OrderbookConfig.WebsocketBufferLimit,
		// Other orderbook buffer vars:
		// BufferEnabled         bool 
		// SortBuffer            bool 
		// SortBufferByUpdateIDs bool 
		// UpdateEntriesByID     bool 
	})
	if err != nil {
		return err
	}
	// Sets up a new connection for the websocket, there are two separate connections denoted by the ConnectionSetup struct auth bool.
	return f.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		// RateLimit            int64  rudimentary rate limit that sleeps connection in milliseconds before sending designated payload
		// Authenticated        bool  sets if the connection is dedicated for an authenticated websocket stream which can be accessed from the Websocket field variable AuthConn e.g. f.Websocket.AuthConn
	})
}
```

Below are the features supported by FTX API protocol:

  ```go
  f.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:      true,
				KlineFetching:       true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				OrderbookFetching: true,
				TradeFetching:     true,
				Subscribe:         true,
				Unsubscribe:       true,
				GetOrders:         true,
				GetOrder:          true,
			},
			WithdrawPermissions: exchange.NoAPIWithdrawalMethods,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}
```

- Link websocket to wrapper functions:

Initially the functions return nil or common.ErrNotYetImplemented

```go
// AuthenticateWebsocket sends an authentication message to the websocket
func (f *FTX) AuthenticateWebsocket() error {
	return f.WsAuth()
}
```


## Last but not least - Live testing

### Live testing websocket via [gctcli](../cmd/gctcli/main.go)

Please test all `websocket` commands below whilst a GoCryptoTrader instance is running and with the exchange websocket setting enabled:

- `getinfo` to ensure fetching websocket information is possible (that the websocket connection is enabled, connected and is running).
- `disable/enable` to ensure disabling/enabling a websocket connection disconnects/connects accordingly.
- `getsubs` to ensure the subscriptions are in sync with the exchange's config settings or by manual subscriptions added/removed via `gctcli`.
- `setproxy` to ensure that a proxy can be set and resets the websocket connection accordingly.
- `seturl` to ensure that a new websocket URL can be set in the event of an API endpoint change whilst an instance of GoCryptoTrader is already running.   

Please test all `pair` commands to disable and enable different assets types to witness subscriptions and unsubscriptions:

- `get` to ensure correct enabled and disabled pairs for a supported asset type.
- `disableasset` to ensure disabling of entire asset class and associated unsubscriptions.
- `enableasset` to ensure correct enabling of entire asset class and associated subscriptions.
- `disable` to ensure correct disabling of pair(s) and and associated unsubscriptions.
- `enable` to ensure correct enabling of pair(s) and associated subscriptions.
- `enableall` to ensure correct enabling of all pairs for an asset type and associated subscriptions.
- `disableall` to ensure correct disabling of all pairs for an asset type and associated unsubscriptions.

## Open a PR

Submitting a PR is easy and all are welcome additions to the public repository. Submit via github.com/thrasher-corp/gocryptotrader or contact our team via slack for more information. 
