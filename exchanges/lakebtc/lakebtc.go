package lakebtc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	lakeBTCAPIURL              = "https://api.lakebtc.com/api_v2"
	lakeBTCAPIVersion          = "2"
	lakeBTCTicker              = "ticker"
	lakeBTCOrderbook           = "bcorderbook"
	lakeBTCTrades              = "bctrades"
	lakeBTCGetAccountInfo      = "getAccountInfo"
	lakeBTCBuyOrder            = "buyOrder"
	lakeBTCSellOrder           = "sellOrder"
	lakeBTCOpenOrders          = "openOrders"
	lakeBTCGetOrders           = "getOrders"
	lakeBTCCancelOrder         = "cancelOrders"
	lakeBTCGetTrades           = "getTrades"
	lakeBTCGetExternalAccounts = "getExternalAccounts"
	lakeBTCCreateWithdraw      = "createWithdraw"
)

// LakeBTC is the overarching type across the LakeBTC package
type LakeBTC struct {
	exchange.Base
	WebsocketConn
}

// GetTicker returns the current ticker from lakeBTC
func (l *LakeBTC) GetTicker() (map[string]Ticker, error) {
	response := make(map[string]TickerResponse)
	path := fmt.Sprintf("%s/%s", l.API.Endpoints.URL, lakeBTCTicker)

	err := l.SendHTTPRequest(path, &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]Ticker)

	for k, v := range response {
		var tick Ticker
		key := strings.ToUpper(k)
		if v.Ask != nil {
			tick.Ask, err = strconv.ParseFloat(v.Ask.(string), 64)
			if err != nil {
				return nil, err
			}
		}
		if v.Bid != nil {
			tick.Bid, err = strconv.ParseFloat(v.Bid.(string), 64)
			if err != nil {
				return nil, err
			}
		}
		if v.High != nil {
			tick.High, err = strconv.ParseFloat(v.High.(string), 64)
			if err != nil {
				return nil, err
			}
		}
		if v.Last != nil {
			tick.Last, err = strconv.ParseFloat(v.Last.(string), 64)
			if err != nil {
				return nil, err
			}
		}
		if v.Low != nil {
			tick.Low, err = strconv.ParseFloat(v.Low.(string), 64)
			if err != nil {
				return nil, err
			}
		}
		if v.Volume != nil {
			tick.Volume, err = strconv.ParseFloat(v.Volume.(string), 64)
			if err != nil {
				return nil, err
			}
		}
		result[key] = tick
	}
	return result, nil
}

// GetOrderBook returns the order book from LakeBTC
func (l *LakeBTC) GetOrderBook(currency string) (Orderbook, error) {
	type Response struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}
	path := fmt.Sprintf("%s/%s?symbol=%s", l.API.Endpoints.URL, lakeBTCOrderbook, strings.ToLower(currency))
	resp := Response{}
	err := l.SendHTTPRequest(path, &resp)
	if err != nil {
		return Orderbook{}, err
	}
	orderbook := Orderbook{}

	for _, x := range resp.Bids {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Error(log.ExchangeSys, err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Error(log.ExchangeSys, err)
			continue
		}
		orderbook.Bids = append(orderbook.Bids, OrderbookStructure{price, amount})
	}

	for _, x := range resp.Asks {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Error(log.ExchangeSys, err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Error(log.ExchangeSys, err)
			continue
		}
		orderbook.Asks = append(orderbook.Asks, OrderbookStructure{price, amount})
	}
	return orderbook, nil
}

// GetTradeHistory returns the trade history for a given currency pair
func (l *LakeBTC) GetTradeHistory(currency string) ([]TradeHistory, error) {
	path := fmt.Sprintf("%s/%s?symbol=%s", l.API.Endpoints.URL, lakeBTCTrades, strings.ToLower(currency))
	var resp []TradeHistory
	return resp, l.SendHTTPRequest(path, &resp)
}

// GetAccountInformation returns your current account information
func (l *LakeBTC) GetAccountInformation() (AccountInfo, error) {
	resp := AccountInfo{}
	return resp, l.SendAuthenticatedHTTPRequest(lakeBTCGetAccountInfo, "", &resp)
}

// Trade executes an order on the exchange and returns trade inforamtion or an
// error
func (l *LakeBTC) Trade(isBuyOrder bool, amount, price float64, currency string) (Trade, error) {
	resp := Trade{}
	params := strconv.FormatFloat(price, 'f', -1, 64) + "," + strconv.FormatFloat(amount, 'f', -1, 64) + "," + currency

	if isBuyOrder {
		if err := l.SendAuthenticatedHTTPRequest(lakeBTCBuyOrder, params, &resp); err != nil {
			return resp, err
		}
	} else {
		if err := l.SendAuthenticatedHTTPRequest(lakeBTCSellOrder, params, &resp); err != nil {
			return resp, err
		}
	}

	if resp.Result != "order received" {
		return resp, fmt.Errorf("unexpected result: %s", resp.Result)
	}

	return resp, nil
}

// GetOpenOrders returns all open orders associated with your account
func (l *LakeBTC) GetOpenOrders() ([]OpenOrders, error) {
	var orders []OpenOrders

	return orders, l.SendAuthenticatedHTTPRequest(lakeBTCOpenOrders, "", &orders)
}

// GetOrders returns your orders
func (l *LakeBTC) GetOrders(orders []int64) ([]Orders, error) {
	var ordersStr []string
	for _, x := range orders {
		ordersStr = append(ordersStr, strconv.FormatInt(x, 10))
	}

	var resp []Orders
	return resp,
		l.SendAuthenticatedHTTPRequest(lakeBTCGetOrders, strings.Join(ordersStr, ","), &resp)
}

// CancelExistingOrder cancels an order by ID number and returns an error
func (l *LakeBTC) CancelExistingOrder(orderID int64) error {
	type Response struct {
		Result bool `json:"Result"`
	}

	resp := Response{}
	params := strconv.FormatInt(orderID, 10)
	err := l.SendAuthenticatedHTTPRequest(lakeBTCCancelOrder, params, &resp)
	if err != nil {
		return err
	}

	if !resp.Result {
		return errors.New("unable to cancel order")
	}
	return nil
}

// CancelExistingOrders cancels an order by ID number and returns an error
func (l *LakeBTC) CancelExistingOrders(orderIDs []string) error {
	type Response struct {
		Result bool `json:"Result"`
	}

	resp := Response{}
	params := strings.Join(orderIDs, ",")
	err := l.SendAuthenticatedHTTPRequest(lakeBTCCancelOrder, params, &resp)
	if err != nil {
		return err
	}

	if !resp.Result {
		return fmt.Errorf("unable to cancel order(s): %v", orderIDs)
	}
	return nil
}

// GetTrades returns trades associated with your account by timestamp
func (l *LakeBTC) GetTrades(timestamp int64) ([]AuthenticatedTradeHistory, error) {
	params := ""
	if timestamp != 0 {
		params = strconv.FormatInt(timestamp, 10)
	}

	var trades []AuthenticatedTradeHistory
	return trades, l.SendAuthenticatedHTTPRequest(lakeBTCGetTrades, params, &trades)
}

// GetExternalAccounts returns your external accounts WARNING: Only for BTC!
func (l *LakeBTC) GetExternalAccounts() ([]ExternalAccounts, error) {
	var resp []ExternalAccounts

	return resp, l.SendAuthenticatedHTTPRequest(lakeBTCGetExternalAccounts, "", &resp)
}

// CreateWithdraw allows your to withdraw to external account WARNING: Only for
// BTC!
func (l *LakeBTC) CreateWithdraw(amount float64, accountID string) (Withdraw, error) {
	resp := Withdraw{}
	params := strconv.FormatFloat(amount, 'f', -1, 64) + ",btc," + accountID

	err := l.SendAuthenticatedHTTPRequest(lakeBTCCreateWithdraw, params, &resp)
	if err != nil {
		return Withdraw{}, err
	}
	if len(resp.Error) > 0 {
		return resp, errors.New(resp.Error)
	}

	return resp, nil
}

// SendHTTPRequest sends an unauthenticated http request
func (l *LakeBTC) SendHTTPRequest(path string, result interface{}) error {
	return l.SendPayload(&request.Item{
		Method:        http.MethodGet,
		Path:          path,
		Result:        result,
		Verbose:       l.Verbose,
		HTTPDebugging: l.HTTPDebugging,
		HTTPRecording: l.HTTPRecording,
	})
}

// SendAuthenticatedHTTPRequest sends an autheticated HTTP request to a LakeBTC
func (l *LakeBTC) SendAuthenticatedHTTPRequest(method, params string, result interface{}) (err error) {
	if !l.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, l.Name)
	}

	n := l.Requester.GetNonce(true).String()

	req := fmt.Sprintf("tonce=%s&accesskey=%s&requestmethod=post&id=1&method=%s&params=%s", n, l.API.Credentials.Key, method, params)
	hmac := crypto.GetHMAC(crypto.HashSHA1, []byte(req), []byte(l.API.Credentials.Secret))

	if l.Verbose {
		log.Debugf(log.ExchangeSys, "Sending POST request to %s calling method %s with params %s\n", l.API.Endpoints.URL, method, req)
	}

	postData := make(map[string]interface{})
	postData["method"] = method
	postData["id"] = 1
	postData["params"] = strings.Split(params, ",")

	data, err := json.Marshal(postData)
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["Json-Rpc-Tonce"] = l.Nonce.String()
	headers["Authorization"] = "Basic " + crypto.Base64Encode([]byte(l.API.Credentials.Key+":"+crypto.HexEncodeToString(hmac)))
	headers["Content-Type"] = "application/json-rpc"

	return l.SendPayload(&request.Item{
		Method:        http.MethodPost,
		Path:          l.API.Endpoints.URL,
		Headers:       headers,
		Body:          strings.NewReader(string(data)),
		Result:        result,
		AuthRequest:   true,
		NonceEnabled:  true,
		Verbose:       l.Verbose,
		HTTPDebugging: l.HTTPDebugging,
		HTTPRecording: l.HTTPRecording,
	})
}

// GetFee returns an estimate of fee based on type of transaction
func (l *LakeBTC) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)
	case exchange.CyptocurrencyDepositFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.InternationalBankWithdrawalFee:
		// fees for withdrawals are dynamic. They cannot be calculated in
		// advance as they are manually performed via the website, it can only
		// be determined when submitting the request
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

func calculateTradingFee(purchasePrice, amount float64, isMaker bool) (fee float64) {
	if isMaker {
		// TODO: Volume based fee calculation
		fee = 0.0015
	} else {
		fee = 0.002
	}

	return fee * amount * purchasePrice
}

func getCryptocurrencyWithdrawalFee(c currency.Code) (fee float64) {
	if c == currency.BTC {
		fee = 0.001
	}
	return fee
}
