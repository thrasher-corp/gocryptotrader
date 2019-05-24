package gateio

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	gateioTradeURL   = "https://api.gateio.io"
	gateioMarketURL  = "https://data.gateio.io"
	gateioAPIVersion = "api2/1"

	gateioSymbol          = "pairs"
	gateioMarketInfo      = "marketinfo"
	gateioKline           = "candlestick2"
	gateioOrder           = "private"
	gateioBalances        = "private/balances"
	gateioCancelOrder     = "private/cancelOrder"
	gateioCancelAllOrders = "private/cancelAllOrders"
	gateioWithdraw        = "private/withdraw"
	gateioOpenOrders      = "private/openOrders"
	gateioTradeHistory    = "private/tradeHistory"
	gateioDepositAddress  = "private/depositAddress"
	gateioTicker          = "ticker"
	gateioTickers         = "tickers"
	gateioOrderbook       = "orderBook"

	gateioAuthRate   = 100
	gateioUnauthRate = 100

	gateioGenerateAddress = "New address is being generated for you, please wait a moment and refresh this page. "
)

// Gateio is the overarching type across this package
type Gateio struct {
	WebsocketConn *wshandler.WebsocketConnection
	exchange.Base
}

// SetDefaults sets default values for the exchange
func (g *Gateio) SetDefaults() {
	g.Name = "GateIO"
	g.Enabled = false
	g.Verbose = false
	g.RESTPollingDelay = 10
	g.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	g.RequestCurrencyPairFormat.Delimiter = "_"
	g.RequestCurrencyPairFormat.Uppercase = false
	g.ConfigCurrencyPairFormat.Delimiter = "_"
	g.ConfigCurrencyPairFormat.Uppercase = true
	g.AssetTypes = []string{ticker.Spot}
	g.SupportsAutoPairUpdating = true
	g.SupportsRESTTickerBatching = true
	g.Requester = request.New(g.Name,
		request.NewRateLimit(time.Second*10, gateioAuthRate),
		request.NewRateLimit(time.Second*10, gateioUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	g.APIUrlDefault = gateioTradeURL
	g.APIUrl = g.APIUrlDefault
	g.APIUrlSecondaryDefault = gateioMarketURL
	g.APIUrlSecondary = g.APIUrlSecondaryDefault
	g.Websocket = wshandler.New()
	g.Websocket.Functionality = wshandler.WebsocketTickerSupported |
		wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketKlineSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported |
		wshandler.WebsocketMessageCorrelationSupported
	g.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	g.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	g.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user configuration
func (g *Gateio) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		g.SetEnabled(false)
	} else {
		g.Enabled = true
		g.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		g.AuthenticatedWebsocketAPISupport = exch.AuthenticatedWebsocketAPISupport
		g.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		g.APIAuthPEMKey = exch.APIAuthPEMKey
		g.SetHTTPClientTimeout(exch.HTTPTimeout)
		g.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		g.RESTPollingDelay = exch.RESTPollingDelay
		g.Verbose = exch.Verbose
		g.BaseCurrencies = exch.BaseCurrencies
		g.AvailablePairs = exch.AvailablePairs
		g.EnabledPairs = exch.EnabledPairs
		g.WebsocketURL = gateioWebsocketEndpoint
		g.HTTPDebugging = exch.HTTPDebugging
		err := g.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = g.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = g.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = g.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = g.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = g.Websocket.Setup(g.WsConnect,
			g.Subscribe,
			g.Unsubscribe,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			gateioWebsocketEndpoint,
			exch.WebsocketURL,
			exch.AuthenticatedWebsocketAPISupport)
		if err != nil {
			log.Fatal(err)
		}
		g.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         g.Name,
			URL:                  g.Websocket.GetWebsocketURL(),
			ProxyURL:             g.Websocket.GetProxyAddress(),
			Verbose:              g.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
			RateLimit:            gateioWebsocketRateLimit,
		}
		g.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			true,
			false,
			false,
			false,
			exch.Name)
	}
}

// GetSymbols returns all supported symbols
func (g *Gateio) GetSymbols() ([]string, error) {
	var result []string

	urlPath := fmt.Sprintf("%s/%s/%s", g.APIUrlSecondary, gateioAPIVersion, gateioSymbol)

	err := g.SendHTTPRequest(urlPath, &result)
	if err != nil {
		return nil, nil
	}
	return result, err
}

// GetMarketInfo returns information about all trading pairs, including
// transaction fee, minimum order quantity, price accuracy and so on
func (g *Gateio) GetMarketInfo() (MarketInfoResponse, error) {
	type response struct {
		Result string        `json:"result"`
		Pairs  []interface{} `json:"pairs"`
	}

	urlPath := fmt.Sprintf("%s/%s/%s", g.APIUrlSecondary, gateioAPIVersion, gateioMarketInfo)

	var res response
	var result MarketInfoResponse
	err := g.SendHTTPRequest(urlPath, &res)
	if err != nil {
		return result, err
	}

	result.Result = res.Result
	for _, v := range res.Pairs {
		item := v.(map[string]interface{})
		for itemk, itemv := range item {
			pairv := itemv.(map[string]interface{})
			result.Pairs = append(result.Pairs, MarketInfoPairsResponse{
				Symbol:        itemk,
				DecimalPlaces: pairv["decimal_places"].(float64),
				MinAmount:     pairv["min_amount"].(float64),
				Fee:           pairv["fee"].(float64),
			})
		}
	}
	return result, nil
}

// GetLatestSpotPrice returns latest spot price of symbol
// updated every 10 seconds
//
// symbol: string of currency pair
func (g *Gateio) GetLatestSpotPrice(symbol string) (float64, error) {
	res, err := g.GetTicker(symbol)
	if err != nil {
		return 0, err
	}

	return res.Last, nil
}

// GetTicker returns a ticker for the supplied symbol
// updated every 10 seconds
func (g *Gateio) GetTicker(symbol string) (TickerResponse, error) {
	urlPath := fmt.Sprintf("%s/%s/%s/%s", g.APIUrlSecondary, gateioAPIVersion, gateioTicker, symbol)
	var res TickerResponse
	return res, g.SendHTTPRequest(urlPath, &res)
}

// GetTickers returns tickers for all symbols
func (g *Gateio) GetTickers() (map[string]TickerResponse, error) {
	urlPath := fmt.Sprintf("%s/%s/%s", g.APIUrlSecondary, gateioAPIVersion, gateioTickers)

	resp := make(map[string]TickerResponse)
	err := g.SendHTTPRequest(urlPath, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOrderbook returns the orderbook data for a suppled symbol
func (g *Gateio) GetOrderbook(symbol string) (Orderbook, error) {
	urlPath := fmt.Sprintf("%s/%s/%s/%s", g.APIUrlSecondary, gateioAPIVersion, gateioOrderbook, symbol)

	var resp OrderbookResponse
	err := g.SendHTTPRequest(urlPath, &resp)
	if err != nil {
		return Orderbook{}, err
	}

	if resp.Result != "true" {
		return Orderbook{}, errors.New("result was not true")
	}

	var ob Orderbook

	if len(resp.Asks) == 0 {
		return ob, errors.New("asks are empty")
	}

	// Asks are in reverse order
	for x := len(resp.Asks) - 1; x != 0; x-- {
		data := resp.Asks[x]

		price, err := strconv.ParseFloat(data[0], 64)
		if err != nil {
			continue
		}

		amount, err := strconv.ParseFloat(data[1], 64)
		if err != nil {
			continue
		}

		ob.Asks = append(ob.Asks, OrderbookItem{Price: price, Amount: amount})
	}

	if len(resp.Bids) == 0 {
		return ob, errors.New("bids are empty")
	}

	for x := range resp.Bids {
		data := resp.Bids[x]

		price, err := strconv.ParseFloat(data[0], 64)
		if err != nil {
			continue
		}

		amount, err := strconv.ParseFloat(data[1], 64)
		if err != nil {
			continue
		}

		ob.Bids = append(ob.Bids, OrderbookItem{Price: price, Amount: amount})
	}

	ob.Result = resp.Result
	ob.Elapsed = resp.Elapsed
	return ob, nil
}

// GetSpotKline returns kline data for the most recent time period
func (g *Gateio) GetSpotKline(arg KlinesRequestParams) ([]*KLineResponse, error) {
	urlPath := fmt.Sprintf("%s/%s/%s/%s?group_sec=%d&range_hour=%d",
		g.APIUrlSecondary,
		gateioAPIVersion,
		gateioKline,
		arg.Symbol,
		arg.GroupSec,
		arg.HourSize)

	var rawKlines map[string]interface{}
	err := g.SendHTTPRequest(urlPath, &rawKlines)
	if err != nil {
		return nil, err
	}

	var result []*KLineResponse
	if rawKlines == nil || rawKlines["data"] == nil {
		return nil, fmt.Errorf("rawKlines is nil. Err: %s", err)
	}

	rawKlineDatasString, _ := json.Marshal(rawKlines["data"].([]interface{}))
	var rawKlineDatas [][]interface{}
	if err := json.Unmarshal(rawKlineDatasString, &rawKlineDatas); err != nil {
		return nil, fmt.Errorf("rawKlines unmarshal failed. Err: %s", err)
	}

	for _, k := range rawKlineDatas {
		otString, _ := strconv.ParseFloat(k[0].(string), 64)
		ot, err := common.TimeFromUnixTimestampFloat(otString)
		if err != nil {
			return nil, fmt.Errorf("cannot parse Kline.OpenTime. Err: %s", err)
		}
		_vol, err := common.FloatFromString(k[1])
		if err != nil {
			return nil, fmt.Errorf("cannot parse Kline.Volume. Err: %s", err)
		}
		_id, err := common.FloatFromString(k[0])
		if err != nil {
			return nil, fmt.Errorf("cannot parse Kline.Id. Err: %s", err)
		}
		_close, err := common.FloatFromString(k[2])
		if err != nil {
			return nil, fmt.Errorf("cannot parse Kline.Close. Err: %s", err)
		}
		_high, err := common.FloatFromString(k[3])
		if err != nil {
			return nil, fmt.Errorf("cannot parse Kline.High. Err: %s", err)
		}
		_low, err := common.FloatFromString(k[4])
		if err != nil {
			return nil, fmt.Errorf("cannot parse Kline.Low. Err: %s", err)
		}
		_open, err := common.FloatFromString(k[5])
		if err != nil {
			return nil, fmt.Errorf("cannot parse Kline.Open. Err: %s", err)
		}
		result = append(result, &KLineResponse{
			ID:        _id,
			KlineTime: ot,
			Volume:    _vol,
			Close:     _close,
			High:      _high,
			Low:       _low,
			Open:      _open,
		})
	}
	return result, nil
}

// GetBalances obtains the users account balance
func (g *Gateio) GetBalances() (BalancesResponse, error) {
	var result BalancesResponse

	return result,
		g.SendAuthenticatedHTTPRequest(http.MethodPost, gateioBalances, "", &result)
}

// SpotNewOrder places a new order
func (g *Gateio) SpotNewOrder(arg SpotNewOrderRequestParams) (SpotNewOrderResponse, error) {
	var result SpotNewOrderResponse

	// Be sure to use the correct price precision before calling this
	params := fmt.Sprintf("currencyPair=%s&rate=%s&amount=%s",
		arg.Symbol,
		strconv.FormatFloat(arg.Price, 'f', -1, 64),
		strconv.FormatFloat(arg.Amount, 'f', -1, 64),
	)

	urlPath := fmt.Sprintf("%s/%s", gateioOrder, arg.Type)
	return result, g.SendAuthenticatedHTTPRequest(http.MethodPost, urlPath, params, &result)
}

// CancelExistingOrder cancels an order given the supplied orderID and symbol
// orderID order ID number
// symbol trade pair (ltc_btc)
func (g *Gateio) CancelExistingOrder(orderID int64, symbol string) (bool, error) {
	type response struct {
		Result  bool   `json:"result"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	var result response
	// Be sure to use the correct price precision before calling this
	params := fmt.Sprintf("orderNumber=%d&currencyPair=%s",
		orderID,
		symbol,
	)
	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, gateioCancelOrder, params, &result)
	if err != nil {
		return false, err
	}
	if !result.Result {
		return false, fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return true, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (g *Gateio) SendHTTPRequest(path string, result interface{}) error {
	return g.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		result,
		false,
		false,
		g.Verbose,
		g.HTTPDebugging,
		g.HTTPRecording)
}

// CancelAllExistingOrders all orders for a given symbol and side
// orderType (0: sell,1: buy,-1: unlimited)
func (g *Gateio) CancelAllExistingOrders(orderType int64, symbol string) error {
	type response struct {
		Result  bool   `json:"result"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	var result response
	params := fmt.Sprintf("type=%d&currencyPair=%s",
		orderType,
		symbol,
	)
	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, gateioCancelAllOrders, params, &result)
	if err != nil {
		return err
	}

	if !result.Result {
		return fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return nil
}

// GetOpenOrders retrieves all open orders with an optional symbol filter
func (g *Gateio) GetOpenOrders(symbol string) (OpenOrdersResponse, error) {
	var params string
	var result OpenOrdersResponse

	if symbol != "" {
		params = fmt.Sprintf("currencyPair=%s", symbol)
	}

	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, gateioOpenOrders, params, &result)
	if err != nil {
		return result, err
	}

	if result.Code > 0 {
		return result, fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return result, nil
}

// GetTradeHistory retrieves all orders with an optional symbol filter
func (g *Gateio) GetTradeHistory(symbol string) (TradHistoryResponse, error) {
	var params string
	var result TradHistoryResponse
	params = fmt.Sprintf("currencyPair=%s", symbol)

	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, gateioTradeHistory, params, &result)
	if err != nil {
		return result, err
	}

	if result.Code > 0 {
		return result, fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return result, nil
}

// GenerateSignature returns hash for authenticated requests
func (g *Gateio) GenerateSignature(message string) []byte {
	return common.GetHMAC(common.HashSHA512, []byte(message), []byte(g.APISecret))
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the Gateio API
// To use this you must setup an APIKey and APISecret from the exchange
func (g *Gateio) SendAuthenticatedHTTPRequest(method, endpoint, param string, result interface{}) error {
	if !g.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			g.Name)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	headers["key"] = g.APIKey

	hmac := g.GenerateSignature(param)
	headers["sign"] = common.HexEncodeToString(hmac)

	urlPath := fmt.Sprintf("%s/%s/%s", g.APIUrl, gateioAPIVersion, endpoint)

	var intermidiary json.RawMessage

	err := g.SendPayload(method,
		urlPath,
		headers,
		strings.NewReader(param),
		&intermidiary,
		true,
		false,
		g.Verbose,
		g.HTTPDebugging,
		g.HTTPRecording)
	if err != nil {
		return err
	}

	errCap := struct {
		Result  bool   `json:"result,string"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{}

	if err := common.JSONDecode(intermidiary, &errCap); err == nil {
		if !errCap.Result {
			return fmt.Errorf("%s auth request error, code: %d message: %s",
				g.Name,
				errCap.Code,
				errCap.Message)
		}
	}

	return common.JSONDecode(intermidiary, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (g *Gateio) GetFee(feeBuilder *exchange.FeeBuilder) (fee float64, err error) {
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feePairs, err := g.GetMarketInfo()
		if err != nil {
			return 0, err
		}

		currencyPair := feeBuilder.Pair.Base.String() +
			feeBuilder.Pair.Delimiter +
			feeBuilder.Pair.Quote.String()

		var feeForPair float64
		for _, i := range feePairs.Pairs {
			if strings.EqualFold(currencyPair, i.Symbol) {
				feeForPair = i.Fee
			}
		}

		if feeForPair == 0 {
			return 0, fmt.Errorf("currency '%s' failed to find fee data",
				currencyPair)
		}

		fee = calculateTradingFee(feeForPair,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount)

	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}

	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.002 * price * amount
}

func calculateTradingFee(feeForPair, purchasePrice, amount float64) float64 {
	return (feeForPair / 100) * purchasePrice * amount
}

func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

// WithdrawCrypto withdraws cryptocurrency to your selected wallet
func (g *Gateio) WithdrawCrypto(currency, address string, amount float64) (string, error) {
	type response struct {
		Result  bool   `json:"result"`
		Message string `json:"message"`
		Code    int    `json:"code"`
	}

	var result response
	params := fmt.Sprintf("currency=%v&amount=%v&address=%v",
		currency,
		address,
		amount,
	)
	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, gateioWithdraw, params, &result)
	if err != nil {
		return "", err
	}
	if !result.Result {
		return "", fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return result.Message, nil
}

// GetCryptoDepositAddress returns a deposit address for a cryptocurrency
func (g *Gateio) GetCryptoDepositAddress(currency string) (string, error) {
	type response struct {
		Result  bool   `json:"result,string"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Address string `json:"addr"`
	}

	var result response
	params := fmt.Sprintf("currency=%s",
		currency)

	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, gateioDepositAddress, params, &result)
	if err != nil {
		return "", err
	}

	if !result.Result {
		return "", fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return result.Address, nil
}
