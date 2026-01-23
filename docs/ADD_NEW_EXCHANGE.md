# GoCryptoTrader ADD NEW EXCHANGE

<img src="/docs/assets/page-logo.png" width="350px" height="350px" hspace="70" alt="GoCryptoTrader project logo">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## How to add a new exchange

This document is from a perspective of adding a new exchange called Binance to the codebase:

### Run the [exchange templating tool](../cmd/exchange_template/) which will create a base exchange package based on the features the exchange supports

#### Linux/macOS

GoCryptoTrader is built using [Go Modules](https://go.dev/wiki/Modules) and requires Go 1.11 or above
Using Go Modules you now clone this repository **outside** your GOPATH

```console
git clone https://github.com/thrasher-corp/gocryptotrader.git
cd gocryptotrader/cmd/exchange_template
go run exchange_template.go -name Binance -ws -rest
```

#### Windows

```console
git clone https://github.com/thrasher-corp/gocryptotrader.git
cd gocryptotrader\cmd\exchange_template
go run exchange_template.go -name Binance -ws -rest
```

### Add exchange struct to [config_example.json](../config_example.json), [configtest.json](../testdata/configtest.json)

Find out which asset types are supported by the exchange and add them to the pairs struct (spot is enabled by default)

If main config path is unknown the following function can be used:

```go
config.GetDefaultFilePath()
```

```js
  {
   "name": "Binance",
   "enabled": true,
   "verbose": false,
   "currencyPairs": {
    "bypassConfigFormatUpgrades": false,
    "requestFormat": {
     "uppercase": true
    },
    "configFormat": {
     "uppercase": true,
     "delimiter": "-"
    },
    "useGlobalFormat": true,
    "pairs": {
     "spot": {
      "assetEnabled": true,
      "enabled": "BTC-USDT",
      "available": "BTC-USDT,BNB-BTC,NEO-BTC,QTUM-ETH,ETH-BTC"
     }
    }
   },
  },
```

#### Configs can be updated automatically by running the following command

Check to make sure that the command does not override the NTP client and encrypt config default settings:

```console
go build && gocryptotrader.exe --config=config_example.json
```

### Add the currency pair format structs in wrapper.go

#### Futures currency support

Similar to the configs, spot support is inbuilt but other asset types will need to be manually supported

```go
    fmt1 := currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: "_"},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: "_"},
	}

	fmt2 := currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: "_"},
	}

	if err := e.SetAssetPairStore(asset.Spot, fmt1); err != nil {
		log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, asset.Spot, err)
	}
	if err := e.SetAssetPairStore(asset.Futures, fmt2); err != nil {
		log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, asset.Futures, err)
	}
```

### Document the addition of the new exchange (Binance exchange is used as an example below)

**Yes** means supported, **No** means not yet implemented and **NA** means protocol unsupported by the exchange

#### Add exchange to the [root README template](/cmd/documentation/root_templates/root_readme.tmpl) file

```go
| Exchange | REST API | Websocket API | FIX API |
|----------|------|-----------|-----|
| Binance| Yes  | Yes        | NA  | // <-------- new exchange
| Bitfinex | Yes  | Yes        | NA  |
| Bitflyer | Yes  | No      | NA  |
| Bithumb | Yes  | NA       | NA  |
| BitMEX | Yes | Yes | NA |
| Bitstamp | Yes  | Yes       | No  |
| BTCMarkets | Yes | No       | NA  |
| BTSE | Yes | Yes | NA |
| Bybit | Yes | Yes | NA |
| COINUT | Yes | Yes | NA |
| Deribit | Yes | Yes | NA |
| Exmo | Yes | NA | NA |
| Coinbase | Yes | Yes | No|
| GateIO | Yes | Yes | NA |
| Gemini | Yes | Yes | No |
| HitBTC | Yes | Yes | No |
| Huobi.Pro | Yes | Yes | NA |
| Kraken | Yes | Yes | NA |
| Kucoin | Yes | Yes | No |
| Lbank | Yes | No | NA |
| Okx | Yes | Yes | NA |
| Poloniex | Yes | Yes | NA |
| Yobit | Yes | NA | NA |
```

#### Add exchange to the list of [supported exchanges](../exchanges/support.go)

```go
var Exchanges = []string{
    "binance", // <-------- new exchange
    "bitfinex",
    "bitflyer",
    "bithumb",
    "bitmex",
    "bitstamp",
    "btc markets",
    "btse",
    "bybit",
    "coinbase",
    "coinut",
    "deribit",
    "exmo",
    "gateio",
    "gemini",
    "hitbtc",
    "huobi",
    "kraken",
    "kucoin",
    "lbank",
    "okx",
    "poloniex",
    "yobit",
```

#### Setup and run the [documentation tool](../cmd/documentation)

- Create a new file named *exchangename*.tmpl
- Copy contents of template from another exchange example here being Exmo
- Replace names and variables as shown:

```go
{{define "exchanges exmo" -}} // exmo -> binance
{{template "header" .}}
## Exmo Exchange

#### Current Features

+ REST Support // if websocket or fix are supported, add that in too
```

```go
var e exchange.IBotExchange

for i := range bot.Exchanges {
  if bot.Exchanges[i].GetName() == "Exmo" { // Exmo -> Binance
    e = bot.Exchanges[i]
  }
}

// Public calls - wrapper functions

pair := currency.NewBTCUSD()

// Fetches current ticker information
tick, err := e.GetCachedTicker(context.Background(), pair, asset.Spot)
if err != nil {
  // Handle error
}

// Fetches current orderbook information
ob, err := e.GetCachedOrderbook(context.Background(), pair, asset.Spot)
if err != nil {
  // Handle error
}
```

- Run documentation.go to generate readme file for the exchange:

```console
cd gocryptotrader\cmd\documentation
go run documentation.go
```

This will generate a readme file for the exchange which can be found in the new exchange's folder

### Code Consistency Guidelines

Please refer to our [coding guidelines](/docs/CODING_GUIDELINES.md).

### Create functions supported by the exchange

#### Requester functions

```go
// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, path string, result any) error {
    // This is used to generate the *http.Request, used in conjunction with the
    // generate functionality below. 
    item := &request.Item{  
        Method:                 http.MethodGet,
        Path:                   path,
        Result:                 result,
        Verbose:                e.Verbose,
        HTTPDebugging:          e.HTTPDebugging,
        HTTPRecording:          e.HTTPRecording,
        HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
    }

    // Request function that closes over the above request.Item values, which
    // executes on every attempt after rate limiting. 
    generate := func() (*request.Item, error) { return item, nil }

    endpoint := request.Unset // Used in conjunction with the rate limiting 
    // system defined in the exchange package to slow down outbound requests
    // depending on each individual endpoint. 
    return e.SendPayload(ctx, endpoint, generate, request.UnauthenticatedRequest)
}
```

#### Public Functions

[Binance Spot REST API reference link](https://developers.binance.com/docs/binance-spot-api-docs/rest-api/general-endpoints).

Create a type struct in `types.go` for the response type, based on the above documentation.

For efficiency, a [JSON to Golang converter can be used](https://mholt.github.io/json-to-go/).
However, great care must be taken as to the values which are autogenerated. The JSON converter tool will default to whatever type it detects, but ultimately conversions to a more useful variable type would be better. For example, price and quantity on some exchange API's provide these as strings. Internally, it would be better if they're converted to the more useful float64 var type.

```go
// ExchangeInfo holds the full exchange information type
type ExchangeInfo struct {
    Code       int        `json:"code"`
    Msg        string     `json:"msg"`
    Timezone   string     `json:"timezone"`
    ServerTime types.Time `json:"serverTime"`
    RateLimits []*struct {
        RateLimitType string `json:"rateLimitType"`
        Interval      string `json:"interval"`
        Limit         int    `json:"limit"`
    } `json:"rateLimits"`
    ExchangeFilters any `json:"exchangeFilters"`
    Symbols         []*struct {
        Symbol                     string        `json:"symbol"`
        Status                     string        `json:"status"`
        BaseAsset                  string        `json:"baseAsset"`
        BaseAssetPrecision         int           `json:"baseAssetPrecision"`
        QuoteAsset                 string        `json:"quoteAsset"`
        QuotePrecision             int           `json:"quotePrecision"`
        OrderTypes                 []string      `json:"orderTypes"`
        IcebergAllowed             bool          `json:"icebergAllowed"`
        OCOAllowed                 bool          `json:"ocoAllowed"`
        QuoteOrderQtyMarketAllowed bool          `json:"quoteOrderQtyMarketAllowed"`
        IsSpotTradingAllowed       bool          `json:"isSpotTradingAllowed"`
        IsMarginTradingAllowed     bool          `json:"isMarginTradingAllowed"`
        Filters                    []*filterData `json:"filters"`
        Permissions                []string      `json:"permissions"`
        PermissionSets             [][]string    `json:"permissionSets"`
    } `json:"symbols"`
}
```

Modify existing constants or create new ones to define the API URL paths, as appropriate:

```go
    apiURL = "https://api.binance.com"
```

Create a get function in the `rest.go` file and unmarshal the data in the created type:

```go
// GetExchangeInfo returns exchange information. Check types for more
// information
func (e *Exchange) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
    var resp *ExchangeInfo
    return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, exchangeInfo, spotExchangeInfo, &resp)
}
```

Create a test function in `rest_test.go` to see if the data is received and unmarshalled correctly

```go
func TestGetExchangeInfo(t *testing.T) {
    t.Parallel() // adding t.Parallel() is preferred as it allows tests to run simultaneously, speeding up package test time
    e.Verbose = true
    result, err := e.GetExchangeInfo(t.Context())
    require.NoError(t, err)
    t.Log(result)
    assert.NotNil(t, result)
}
```

Set `Verbose` to `true` to view received data during unmarshalling errors.
After testing, remove `Verbose`, the result variable, and `t.Log(result)`, or replace the log with `assert.NotNil(t, result)` to avoid unnecessary output when running GCT.
Alternatively you can use `request.WithVerbose(t.Context())` as the `context` param to achieve the same result.

```go
    result, err := e.GetExchangeInfo(t.Context())
    require.NoError(t, err)
    assert.NotNil(t, result)
```

Ensure each endpoint is implemented and has an associated test to improve test coverage and increase confidence

#### Message IDs

* e.MessageID() to get a UUIDv7 if the exchange supports unique string IDs
* e.MessageSequence() to get a simple integer ID if uniqueness is not critical
* Otherwise override MessageID with a suitable alternative

#### Authenticated functions

Authenticated request function is created based on the way the exchange documentation specifies. For example, see the [Binance Spot API - Endpoint Security Types](https://developers.binance.com/docs/binance-spot-api-docs/rest-api/endpoint-security-type).

```go
// SendAuthHTTPRequest sends an authenticated request
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result any) error {
    // A potential example below of closing over authenticated variables which may 
    // be required to regenerate on every request between each attempt after rate
    // limiting. This is for when signatures are based on timestamps/nonces that are 
    // within time receive windows. NOTE: This is not always necessary and the above
    // SendHTTPRequest example may suffice.

    // Fetches credentials, this can either use a context set credential or if
    // not found, will default to the config.json exchange specific credentials.
    creds, err := e.GetCredentials(ctx)
    if err != nil {
        return err
    }

    endpointPath, err := e.API.Endpoints.GetURL(ePath)
    if err != nil {
        return err
    }

    if params == nil {
        params = url.Values{}
    }

    if params.Get("recvWindow") == "" {
        params.Set("recvWindow", strconv.FormatInt(defaultRecvWindow.Milliseconds(), 10))
    }

    interim := json.RawMessage{}
    err = e.SendPayload(ctx, f, func() (*request.Item, error) {
        params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
        hmacSigned, err := crypto.GetHMAC(crypto.HashSHA256, []byte(params.Encode()), []byte(creds.Secret))
        if err != nil {
            return nil, err
        }
        headers := make(map[string]string)
        headers["X-MBX-APIKEY"] = creds.Key
        fullPath := common.EncodeURLValues(endpointPath+path, params) + "&signature=" + hex.EncodeToString(hmacSigned)
        return &request.Item{
            Method:                 method,
            Path:                   fullPath,
            Headers:                headers,
            Result:                 &interim,
            Verbose:                e.Verbose,
            HTTPDebugging:          e.HTTPDebugging,
            HTTPRecording:          e.HTTPRecording,
            HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
        }, nil
    }, request.AuthenticatedRequest)
    if err != nil {
        return err
    }
    errCap := struct {
        Success bool   `json:"success"`
        Message string `json:"msg"`
        Code    int64  `json:"code"`
    }{}

    if err := json.Unmarshal(interim, &errCap); err == nil {
        if !errCap.Success && errCap.Message != "" && errCap.Code != 200 {
            return errors.New(errCap.Message)
        }
    }
    if result == nil {
        return nil
    }
    return json.Unmarshal(interim, result)
}
```

To test authenticated functions, you must have an account with API keys and `SendAuthHTTPRequest` must be implemented.

An HTTP mocking framework can also be added for the exchange. For reference, please see the [HTTP mock package](../testdata/http_mock).

Create authenticated functions and test along the way, similar to the functions above.  

See the [Binance Spot REST API - Account Endpoints](https://developers.binance.com/docs/binance-spot-api-docs/rest-api/account-endpoints) for details.

```go
// GetAccount returns binance user accounts
func (e *Exchange) GetAccount(ctx context.Context) (*Account, error) {
    type response struct {
        Response
        Account
    }

    var resp response
    return &resp.Account,  e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, accountInfo, url.Values{}, spotAccountInformationRate, &resp)
}
```

`GET` request params for authenticated requests are sent through `url.Values{}`:

```go
// QueryOrder returns information on a past order
func (e *Exchange) QueryOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID uint64) (*OrderResponse, error) {
    symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
    if err != nil {
        return resp, err
    }
    params := url.Values{}
    params.Set("symbol", symbolValue)
    if origClientOrderID != "" {
        params.Set("origClientOrderId", origClientOrderID)
    }
    if orderID != 0 {
        params.Set("orderId", strconv.FormatUint(orderID, 10))
    }

    var resp *OrderResponse
    return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, orderEndpoint, params, spotOrderQueryRate, &resp)
}
```

Structs for unmarshalling the data are made exactly the same way as the previous functions.  
See the [Binance Spot REST API - Trading Endpoints](https://developers.binance.com/docs/binance-spot-api-docs/rest-api/trading-endpoints) for details.

```go
// OrderResponse is the return structured response from the exchange
type OrderResponse struct {
    Code            int        `json:"code"`
    Msg             string     `json:"msg"`
    Symbol          string     `json:"symbol"`
    OrderID         int64      `json:"orderId"`
    ClientOrderID   string     `json:"clientOrderId"`
    TransactionTime types.Time `json:"transactTime"`
    Price           float64    `json:"price,string"`
    OrigQty         float64    `json:"origQty,string"`
    ExecutedQty     float64    `json:"executedQty,string"`
    CumulativeQuoteQty float64 `json:"cummulativeQuoteQty,string"`
    Status             string  `json:"status"`
    TimeInForce        string  `json:"timeInForce"`
    Type               string  `json:"type"`
    Side               string  `json:"side"`
    Fills              []struct {
        Price           float64 `json:"price,string"`
        Qty             float64 `json:"qty,string"`
        Commission      float64 `json:"commission,string"`
        CommissionAsset string  `json:"commissionAsset"`
    } `json:"fills"`
}
```

For `POST` or `DELETE` requests, params are sent through a query params:

```go
// NewOrder sends a new test order to Binance
func (e *Exchange) NewOrder(ctx context.Context, o *NewOrderRequest) (*OrderResponse, error) {
    symbol, err := e.FormatSymbol(o.Symbol, asset.Spot)
    if err != nil {
        return err
    }
    params := url.Values{}
    params.Set("symbol", symbol)
    params.Set("side", o.Side)
    params.Set("type", string(o.TradeType))
    if o.QuoteOrderQty > 0 {
        params.Set("quoteOrderQty", strconv.FormatFloat(o.QuoteOrderQty, 'f', -1, 64))
    } else {
        params.Set("quantity", strconv.FormatFloat(o.Quantity, 'f', -1, 64))
    }
    if o.TradeType == BinanceRequestParamsOrderLimit {
        params.Set("price", strconv.FormatFloat(o.Price, 'f', -1, 64))
    }
    if o.TimeInForce != "" {
        params.Set("timeInForce", o.TimeInForce)
    }

    if o.NewClientOrderID != "" {
        params.Set("newClientOrderId", o.NewClientOrderID)
    }

    if o.StopPrice != 0 {
        params.Set("stopPrice", strconv.FormatFloat(o.StopPrice, 'f', -1, 64))
    }

    if o.IcebergQty != 0 {
        params.Set("icebergQty", strconv.FormatFloat(o.IcebergQty, 'f', -1, 64))
    }

    if o.NewOrderRespType != "" {
        params.Set("newOrderRespType", o.NewOrderRespType)
    }
    var resp *OrderResponse
    return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, orderEndpoint, params, spotOrderRate, resp)
}
```

### Implementing wrapper functions

Wrapper functions are the interface in which the GoCryptoTrader engine communicates with an exchange for receiving data and sending requests. See the [exchanges/interfaces.go Go interface file](../exchanges/interfaces.go) for a full list of API methods.
The exchanges may not support all the functionality in the wrapper, so fill out the ones that are supported as shown in the examples below:

Unsupported Example:

```go
// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
    return nil, common.ErrFunctionNotSupported
}
```

Supported Examples:

```go
// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
    if !e.SupportsAsset(a) {
        return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
    }
    tradingStatus := "TRADING"
    var pairs []currency.Pair
    switch a {
    case asset.Spot, asset.Margin:
        info, err := e.GetExchangeInfo(ctx)
        if err != nil {
            return nil, err
        }
        pairs = make([]currency.Pair, 0, len(info.Symbols))
        for x := range info.Symbols {
            if info.Symbols[x].Status != tradingStatus {
                continue
            }
            pair, err := currency.NewPairFromStrings(info.Symbols[x].BaseAsset,
                info.Symbols[x].QuoteAsset)
            if err != nil {
                return nil, err
            }
            if a == asset.Spot && info.Symbols[x].IsSpotTradingAllowed {
                pairs = append(pairs, pair)
            }
            if a == asset.Margin && info.Symbols[x].IsMarginTradingAllowed {
                pairs = append(pairs, pair)
            }
        }
    case asset.CoinMarginedFutures:
        cInfo, err := e.FuturesExchangeInfo(ctx)
        if err != nil {
            return nil, err
        }
        pairs = make([]currency.Pair, 0, len(cInfo.Symbols))
        for z := range cInfo.Symbols {
            if cInfo.Symbols[z].ContractStatus != tradingStatus {
                continue
            }
            pair, err := currency.NewPairFromString(cInfo.Symbols[z].Symbol)
            if err != nil {
                return nil, err
            }
            pairs = append(pairs, pair)
        }
    }
    return pairs, nil
}
```

Wrapper functions on most exchanges are written in similar ways so other exchanges can be used as a reference.

Many helper functions defined in [exchange.go](../exchanges/exchange.go) can be useful when implementing wrapper functions. See examples below:

```go
e.FormatExchangeCurrency(p, a) // Formats the currency pair to the style accepted by the exchange. p is the currency pair & a is the asset type

e.SupportsAsset(a) // Checks if an asset type is supported by the bot

e.GetPairAssetType(p) // Returns the asset type of currency pair p
```

The currency package contains many helper functions to format and process currency pairs. See [currency](../currency/README.md).

### Websocket addition if exchange supports it

#### Websocket Setup

- Set the websocket url in `websocket.go` that is provided in the documentation:

```go
    binanceDefaultWebsocketURL = "wss://stream.binance.com:9443/stream"
```

#### Complete WsConnect function

```go
// WsConnect initiates a websocket connection
func (e *Exchange) WsConnect() error {
    ctx := context.TODO()
    if !e.Websocket.IsEnabled() || !e.IsEnabled() {
        return websocket.ErrWebsocketNotEnabled
    }

    dialer := gws.Dialer{
        HandshakeTimeout: e.Config.HTTPTimeout
        Proxy:            http.ProxyFromEnvironment
    }

    if e.Websocket.CanUseAuthenticatedEndpoints() {
        listenKey, err := e.GetWsAuthStreamKey(ctx)
        if err != nil {
            e.Websocket.SetCanUseAuthenticatedEndpoints(false)
            log.Errorf(log.ExchangeSys, "%v unable to connect to authenticated Websocket. Error: %s", e.Name, err)
        } else {
            // cleans on failed connection
            clean := strings.Split(b.Websocket.GetWebsocketURL(), "?streams=")
            authPayload := clean[0] + "?streams=" + listenKey
            if err := e.Websocket.SetWebsocketURL(authPayload, false, false); err != nil {
                return err
            }
        }
    }

    if err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{}); err != nil {
        return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", e.Name, err)
    }

    if e.Websocket.CanUseAuthenticatedEndpoints() {
        // Start a goroutine to keep the WebSocket auth key alive
        // for accessing authenticated endpoints.
        go e.KeepAuthKeyAlive(ctx)
    }

    e.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
        UseGorillaHandler: true,
        MessageType:       gws.PongMessage,
        Delay:             pingDelay,
    })

    e.Websocket.Wg.Add(1)
    go e.wsReadData()

    e.setupOrderbookManager(ctx)
    return nil
}
```

- Create the authentication function based on the [Binance Spot WebSocket Authentication Requests](https://developers.binance.com/docs/binance-spot-api-docs/websocket-api/authentication-requests) documentation.

```go
// KeepAuthKeyAlive will continuously send messages to
// keep the WS auth key active
func (e *Exchange) KeepAuthKeyAlive(ctx context.Context) {
    defer e.Websocket.Wg.Done()
    for {
        select {
        case <-e.Websocket.ShutdownC:
            return
        case <-time.After(time.Minute * 30):
            if err := e.MaintainWsAuthStreamKey(ctx); err != nil {
                if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
                    log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
                }
                log.Warnf(log.ExchangeSys, "%s %s: Unable to renew auth websocket token, may experience shutdown", e.Name, e.Websocket.Conn.GetURL())
            }
        }
    }
}
```

- Create function to generate default subscriptions:

```go
// generateSubscriptions generates default subscription
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
    for _, s := range e.Features.Subscriptions {
        if s.Asset == asset.Empty {
            // Handle backwards compatibility with config without assets, all binance subs are spot
            s.Asset = asset.Spot
        }
    }
    return e.Features.Subscriptions.ExpandTemplates(b)
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

- Create subscribe function with the data provided by the [Binance Spot Websocket API documentation](https://developers.binance.com/docs/binance-spot-api-docs/websocket-api/request-format)

- Create a struct required to subscribe to channels:

```go
// WsPayload defines the payload through the websocket connection
type WsPayload struct {
    ID     int64    `json:"id"`
    Method string   `json:"method"`
    Params []string `json:"params"`
}
```

- Create the subscription function:

```go
// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(channelsToSubscribe subscription.List) error {
    // For subscriptions we try to batch as much as possible to limit the amount
    // of connection usage but sometimes this is not supported on the exchange API.
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
            // Ensures our outbound currency pair is formatted correctly, sometimes our configuration format is different from what our request format needs to be.
            formattedPair, err := e.FormatExchangeCurrency(channelsToSubscribe[i].Pair, channelsToSubscribe[i].Asset)
            if err != nil {
                errs = append(errs, err)
                continue channels
            }
            sub.Market = formattedPair.String()
        }
        err := e.Websocket.Conn.SendJSONMessage(sub)
        if err != nil {
            errs = append(errs, err)
            continue
        }
        // When we have a successful subscription, we can alert our internal management system of the success.
        e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, channelsToSubscribe[i])
    }
    return errs
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

#### Handle websocket data

- Trades and order events are handled by populating an order.Detail
  struct by [the following rules](./WS_ORDER_EVENTS.md).

- Function to read data received from websocket:

```go
// wsReadData gets and passes on websocket messages for processing
func (e *Exchange) wsReadData() {
    defer e.Websocket.Wg.Done()
    for {
        select {
        case <-e.Websocket.ShutdownC:
            return
        default:
            resp := e.Websocket.Conn.ReadMessage()
            if resp.Raw == nil {
                return
            }
            if err := e.wsHandleData(ctx, resp.Raw); err != nil {
                if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
                    log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
                }
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
    Data         any `json:"data"`
}
```

- Unmarshall the raw data into the main type:

```go
    var result map[string]any
    if err := json.Unmarshal(respRaw, &result); err != nil {
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
        if err := json.Unmarshal(respRaw, &resultData);err != nil {
            return err
        }
        return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
            ExchangeName: e.Name,
            Bid:          resultData.Ticker.Bid,
            Ask:          resultData.Ticker.Ask,
            Last:         resultData.Ticker.Last,
            LastUpdated:  resultData.Ticker.Time,
            Pair:         p,
            AssetType:    a,
        })
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
            return e.Websocket.DataHandler.Send(ctx, resultData.FillsData)
```

- Data Handling can be tested offline similar to the following example:

```go
func TestParsingWSOrdersData(t *testing.T) {
    t.Parallel()
    sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
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
    err := e.wsHandleData(data)
    assert.NoError(t, err)
}
```

- Create types given in the documentation to unmarshall the streamed data from the [Binance Spot WebSocket Streams - Trade Streams](https://developers.binance.com/docs/binance-spot-api-docs/web-socket-streams#trade-streams) documentation.

```go
// TradeStream holds the trade stream data
type TradeStream struct {
    EventType      string       `json:"e"`
    EventTime      types.Time   `json:"E"`
    Symbol         string       `json:"s"`
    TradeID        int64        `json:"t"`
    Price          types.Number `json:"p"`
    Quantity       types.Number `json:"q"`
    BuyerOrderID   int64        `json:"b"`
    SellerOrderID  int64        `json:"a"`
    TimeStamp      types.Time   `json:"T"`
    IsBuyerMaker   bool         `json:"m"`
    BestMatchPrice bool         `json:"M"`
}
```

- Create an unsubscribe function if the exchange has the functionality:

```go
// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(channelsToUnsubscribe subscription.List) error {
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
            formattedPair, err := e.FormatExchangeCurrency(channelsToUnsubscribe[i].Pair, channelsToUnsubscribe[i].Asset)
            if err != nil {
                errs = append(errs, err)
                continue channels
            }
            unSub.Market = formattedPair.String()
        }
        err := e.Websocket.Conn.SendJSONMessage(unSub)
        if err != nil {
            errs = append(errs, err)
            continue
        }
        // When we have a successful unsubscription, we can alert our internal management system of the success.
        e.Websocket.RemoveSubscriptions(e.Websocket.Conn, channelsToUnsubscribe[i])
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
func (e *Exchange) Setup(exch *config.Exchange) error {
    err := exch.Validate()
    if err != nil {
        return err
    }
    if !exch.Enabled {
        e.SetEnabled(false)
        return nil
    }
    err = e.SetupDefaults(exch)
    if err != nil {
        return err
    }

    // Websocket details setup below
    err = e.Websocket.Setup(&websocket.ManagerSetup{
        ExchangeConfig:         exch,
        // DefaultURL defines the default endpoint in the event a rollback is 
        // needed via gctcli.
        DefaultURL:             binanceWSURL, 
        RunningURL:             exch.API.Endpoints.WebsocketURL,
        // Connector function outlined above.
        Connector:              e.WsConnect, 
        // Subscriber function outlined above.
        Subscriber:             e.Subscribe, 
        // Unsubscriber function outlined above.
        UnSubscriber:           e.Unsubscribe,
        // GenerateSubscriptions function outlined above. 
        GenerateSubscriptions:  e.generateSubscriptions, 
        // Defines the capabilities of the websocket outlined in supported 
        // features struct. This allows the websocket connection to be flushed 
        // appropriately if we have a pair/asset enable/disable change. This is 
        // outlined below.
        Features:               &e.Features.Supports.WebsocketCapabilities, 

        // Orderbook buffer specific variables for processing orderbook updates 
        // via websocket feed: 
        // SortBuffer            bool 
        // SortBufferByUpdateIDs bool 
        // UpdateEntriesByID     bool 
    })
    if err != nil {
        return err
    }
    // Sets up a new connection for the websocket, there are two separate connections denoted by the ConnectionSetup struct auth bool.
    return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
        ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
        ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
        // RateLimit            int64  rudimentary rate limit that sleeps connection in milliseconds before sending designated payload
        // Authenticated        bool  sets if the connection is dedicated for an authenticated websocket stream which can be accessed from the Websocket field variable AuthConn e.g. e.Websocket.AuthConn
    })
}
```

Below are the features supported by Binance API protocol:

```go
  e.Features = exchange.Features{
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

Initially the functions `return nil` or `common.ErrNotYetImplemented`.

```go
// AuthenticateWebsocket sends an authentication message to the websocket
func (e *Exchange) AuthenticateWebsocket(ctx context.Context) error {
    return e.WsAuth(ctx)
}
```

## Live testing

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
- `disable` to ensure correct disabling of pair(s) and associated subscriptions.
- `enable` to ensure correct enabling of pair(s) and associated subscriptions.
- `enableall` to ensure correct enabling of all pairs for an asset type and associated subscriptions.
- `disableall` to ensure correct disabling of all pairs for an asset type and associated unsubscriptions.

## Open a PR

Submitting a PR is easy and all are welcome additions to the public repository. Submit via github.com/thrasher-corp/gocryptotrader or contact our team via slack for more information.
