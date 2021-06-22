package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
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
	gateioTrades          = "tradeHistory"
	gateioTickers         = "tickers"
	gateioOrderbook       = "orderBook"

	gateioGenerateAddress = "New address is being generated for you, please wait a moment and refresh this page. "
)

// Gateio is the overarching type across this package
type Gateio struct {
	exchange.Base
}

// GetSymbols returns all supported symbols
func (g *Gateio) GetSymbols() ([]string, error) {
	var result []string
	urlPath := fmt.Sprintf("/%s/%s", gateioAPIVersion, gateioSymbol)
	err := g.SendHTTPRequest(exchange.RestSpotSupplementary, urlPath, &result)
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

	urlPath := fmt.Sprintf("/%s/%s", gateioAPIVersion, gateioMarketInfo)
	var res response
	var result MarketInfoResponse
	err := g.SendHTTPRequest(exchange.RestSpotSupplementary, urlPath, &res)
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
	urlPath := fmt.Sprintf("/%s/%s/%s", gateioAPIVersion, gateioTicker, symbol)
	var res TickerResponse
	return res, g.SendHTTPRequest(exchange.RestSpotSupplementary, urlPath, &res)
}

// GetTickers returns tickers for all symbols
func (g *Gateio) GetTickers() (map[string]TickerResponse, error) {
	urlPath := fmt.Sprintf("/%s/%s", gateioAPIVersion, gateioTickers)
	resp := make(map[string]TickerResponse)
	err := g.SendHTTPRequest(exchange.RestSpotSupplementary, urlPath, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetTrades returns trades for symbols
func (g *Gateio) GetTrades(symbol string) (TradeHistory, error) {
	urlPath := fmt.Sprintf("/%s/%s/%s", gateioAPIVersion, gateioTrades, symbol)
	var resp TradeHistory
	err := g.SendHTTPRequest(exchange.RestSpotSupplementary, urlPath, &resp)
	if err != nil {
		return TradeHistory{}, err
	}
	return resp, nil
}

// GetOrderbook returns the orderbook data for a suppled symbol
func (g *Gateio) GetOrderbook(symbol string) (Orderbook, error) {
	urlPath := fmt.Sprintf("/%s/%s/%s", gateioAPIVersion, gateioOrderbook, symbol)
	var resp OrderbookResponse
	err := g.SendHTTPRequest(exchange.RestSpotSupplementary, urlPath, &resp)
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
func (g *Gateio) GetSpotKline(arg KlinesRequestParams) (kline.Item, error) {
	urlPath := fmt.Sprintf("/%s/%s/%s?group_sec=%s&range_hour=%d",
		gateioAPIVersion,
		gateioKline,
		arg.Symbol,
		arg.GroupSec,
		arg.HourSize)

	var rawKlines map[string]interface{}
	err := g.SendHTTPRequest(exchange.RestSpotSupplementary, urlPath, &rawKlines)
	if err != nil {
		return kline.Item{}, err
	}

	result := kline.Item{
		Exchange: g.Name,
	}

	if rawKlines == nil || rawKlines["data"] == nil {
		return kline.Item{}, errors.New("rawKlines is nil")
	}

	rawKlineDatasString, _ := json.Marshal(rawKlines["data"].([]interface{}))
	var rawKlineDatas [][]interface{}
	if err := json.Unmarshal(rawKlineDatasString, &rawKlineDatas); err != nil {
		return kline.Item{}, fmt.Errorf("rawKlines unmarshal failed. Err: %s", err)
	}

	for _, k := range rawKlineDatas {
		otString, err := strconv.ParseFloat(k[0].(string), 64)
		if err != nil {
			return kline.Item{}, err
		}
		ot, err := convert.TimeFromUnixTimestampFloat(otString)
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.OpenTime. Err: %s", err)
		}
		_vol, err := convert.FloatFromString(k[1])
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.Volume. Err: %s", err)
		}
		_close, err := convert.FloatFromString(k[2])
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.Close. Err: %s", err)
		}
		_high, err := convert.FloatFromString(k[3])
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.High. Err: %s", err)
		}
		_low, err := convert.FloatFromString(k[4])
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.Low. Err: %s", err)
		}
		_open, err := convert.FloatFromString(k[5])
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.Open. Err: %s", err)
		}
		result.Candles = append(result.Candles, kline.Candle{
			Time:   ot,
			Volume: _vol,
			Close:  _close,
			High:   _high,
			Low:    _low,
			Open:   _open,
		})
	}
	return result, nil
}

// GetBalances obtains the users account balance
func (g *Gateio) GetBalances() (BalancesResponse, error) {
	var result BalancesResponse

	return result,
		g.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, gateioBalances, "", &result)
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
	return result, g.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, urlPath, params, &result)
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
	err := g.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, gateioCancelOrder, params, &result)
	if err != nil {
		return false, err
	}
	if !result.Result {
		return false, fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return true, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (g *Gateio) SendHTTPRequest(ep exchange.URL, path string, result interface{}) error {
	endpoint, err := g.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	return g.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       g.Verbose,
		HTTPDebugging: g.HTTPDebugging,
		HTTPRecording: g.HTTPRecording,
	})
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
	err := g.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, gateioCancelAllOrders, params, &result)
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

	err := g.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, gateioOpenOrders, params, &result)
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

	err := g.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, gateioTradeHistory, params, &result)
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
	return crypto.GetHMAC(crypto.HashSHA512, []byte(message),
		[]byte(g.API.Credentials.Secret))
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the Gateio API
// To use this you must setup an APIKey and APISecret from the exchange
func (g *Gateio) SendAuthenticatedHTTPRequest(ep exchange.URL, method, endpoint, param string, result interface{}) error {
	if !g.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", g.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	ePoint, err := g.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	headers["key"] = g.API.Credentials.Key

	hmac := g.GenerateSignature(param)
	headers["sign"] = crypto.HexEncodeToString(hmac)

	urlPath := fmt.Sprintf("%s/%s/%s", ePoint, gateioAPIVersion, endpoint)

	var intermidiary json.RawMessage
	err = g.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          urlPath,
		Headers:       headers,
		Body:          strings.NewReader(param),
		Result:        &intermidiary,
		AuthRequest:   true,
		Verbose:       g.Verbose,
		HTTPDebugging: g.HTTPDebugging,
		HTTPRecording: g.HTTPRecording,
	})
	if err != nil {
		return err
	}

	errCap := struct {
		Result  bool   `json:"result,string"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{}

	if err := json.Unmarshal(intermidiary, &errCap); err == nil {
		if !errCap.Result {
			return fmt.Errorf("%s auth request error, code: %d message: %s",
				g.Name,
				errCap.Code,
				errCap.Message)
		}
	}

	return json.Unmarshal(intermidiary, result)
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
func (g *Gateio) WithdrawCrypto(currency, address string, amount float64) (*withdraw.ExchangeResponse, error) {
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
	err := g.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, gateioWithdraw, params, &result)
	if err != nil {
		return nil, err
	}
	if !result.Result {
		return nil, fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return &withdraw.ExchangeResponse{
		Status: result.Message,
	}, nil
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

	err := g.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, gateioDepositAddress, params, &result)
	if err != nil {
		return "", err
	}

	if !result.Result {
		return "", fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return result.Address, nil
}
