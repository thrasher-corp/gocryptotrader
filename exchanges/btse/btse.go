package btse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// BTSE is the overarching type across this package
type BTSE struct {
	exchange.Base
}

const (
	btseAPIURL         = "https://api.btse.com"
	btseSPOTPath       = "/spot"
	btseSPOTAPIPath    = "/api/v3.2/"
	btseFuturesPath    = "/futures"
	btseFuturesAPIPath = "/api/v2.1/"

	// Public endpoints
	btseMarketOverview = "market_summary"
	btseOrderbook      = "orderbook"
	btseTrades         = "trades"
	btseTime           = "time"
	btseOHLCV          = "ohlcv"

	// Authenticated endpoints
	btseWallet           = "user/wallet"
	btseWalletHistory    = "user/wallet_history"
	btseWalletAddress    = "user/wallet/address"
	btseWalletWithdrawal = "user/wallet/withdraw"
	btseExchangeHistory  = "user/trade_history"
	btseUserFee          = "user/fees"
	btseOrder            = "order"
	btsePendingOrders    = "user/open_orders"
	btseCancelAllAfter   = "order/cancelAllAfter"

	btseTimeLayout = "2006-01-02 15:04:05"
)

// GetMarketsSummary stores market summary data
func (b *BTSE) GetMarketsSummary(symbol string, spot bool) (MarketSummary, error) {
	var m MarketSummary
	path := btseMarketOverview
	if symbol != "" {
		path += "?symbol=" + url.QueryEscape(symbol)
	}
	return m, b.SendHTTPRequest(http.MethodGet, path, &m, spot, queryFunc)
}

// // GetFuturesMarkets returns a list of futures markets available on BTSEx
// func (b *BTSE) GetFuturesMarkets() ([]FuturesMarket, error) {
// 	var m []FuturesMarket
// 	return m, b.SendHTTPRequest(http.MethodGet, btseMarketOverview, &m, false, queryFunc)
// }

// FetchOrderBook gets orderbook data for a given pair
func (b *BTSE) FetchOrderBook(symbol string, group, limitBids, limitAsks int, spot bool) (*Orderbook, error) {
	var o Orderbook
	urlValues := url.Values{}
	urlValues.Add("symbol", symbol)
	if limitBids > 0 {
		urlValues.Add("limit_bids", strconv.Itoa(limitBids))
	}
	if limitAsks > 0 {
		urlValues.Add("limit_asks", strconv.Itoa(limitAsks))
	}
	if group > 0 {
		urlValues.Add("group", strconv.Itoa(group))
	}
	return &o, b.SendHTTPRequest(http.MethodGet,
		common.EncodeURLValues(btseOrderbook, urlValues), &o, spot, queryFunc)
}

// FetchOrderBookL2 retrieve level 2 orderbook for requested symbol and depth
func (b *BTSE) FetchOrderBookL2(symbol string, depth int) (*Orderbook, error) {
	var o Orderbook
	urlValues := url.Values{}
	urlValues.Add("symbol", symbol)
	urlValues.Add("depth", strconv.FormatInt(int64(depth), 10))
	endpoint := common.EncodeURLValues(btseOrderbook+"/L2", urlValues)
	return &o, b.SendHTTPRequest(http.MethodGet, endpoint, &o, true, queryFunc)
}

// GetTrades returns a list of trades for the specified symbol
func (b *BTSE) GetTrades(symbol string, start, end time.Time, count int) ([]Trade, error) {
	var t []Trade
	urlValues := url.Values{}
	urlValues.Add("symbol", symbol)
	if count > 0 {
		urlValues.Add("count", strconv.Itoa(count))
	}
	if !start.IsZero() && !end.IsZero() {
		if start.After(end) {
			return t, errors.New("start cannot be after end time")
		}
		urlValues.Add("start", strconv.FormatInt(start.Unix(), 10))
		urlValues.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	return t, b.SendHTTPRequest(http.MethodGet,
		common.EncodeURLValues(btseTrades, urlValues), &t, true, queryFunc)
}

// OHLCV retrieve and return OHLCV candle data for requested symbol
func (b *BTSE) OHLCV(symbol string, start, end time.Time, resolution int) (OHLCV, error) {
	var o OHLCV
	urlValues := url.Values{}
	urlValues.Add("symbol", symbol)

	if !start.IsZero() && !end.IsZero() {
		if start.After(end) {
			return o, errors.New("start cannot be after end time")
		}
		urlValues.Add("start", strconv.FormatInt(start.Unix(), 10))
		urlValues.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	var res = 60
	if resolution != 0 {
		res = resolution
	}
	urlValues.Add("resolution", strconv.FormatInt(int64(res), 10))
	endpoint := common.EncodeURLValues(btseOHLCV, urlValues)
	return o, b.SendHTTPRequest(http.MethodGet, endpoint, &o, true, queryFunc)
}

// GetServerTime returns the exchanges server time
func (b *BTSE) GetServerTime() (*ServerTime, error) {
	var s ServerTime
	return &s, b.SendHTTPRequest(http.MethodGet, btseTime, &s, true, queryFunc)
}

// GetWalletInformation returns the users account balance
func (b *BTSE) GetWalletInformation() ([]CurrencyBalance, error) {
	var a []CurrencyBalance
	return a, b.SendAuthenticatedHTTPRequest(http.MethodGet, btseWallet, true, nil, nil, &a, queryFunc)
}

// GetFeeInformation retrieve fee's (maker/taker) for requested symbol
func (b *BTSE) GetFeeInformation(symbol string) ([]AccountFees, error) {
	var resp []AccountFees
	urlValues := url.Values{}
	if symbol != "" {
		urlValues.Add("symbol", symbol)
	}
	return resp, b.SendAuthenticatedHTTPRequest(http.MethodGet, btseUserFee, true, urlValues, nil, &resp, queryFunc)
}

// GetWalletHistory returns the users account balance
func (b *BTSE) GetWalletHistory(symbol string, start, end time.Time, count int) (WalletHistory, error) {
	var resp WalletHistory

	urlValues := url.Values{}
	if symbol != "" {
		urlValues.Add("symbol", symbol)
	}
	if !start.IsZero() && !end.IsZero() {
		if start.After(end) || end.Before(start) {
			return resp, errors.New("start cannot be after end time")
		}
		urlValues.Add("start", strconv.FormatInt(start.Unix(), 10))
		urlValues.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	if count > 0 {
		urlValues.Add("count", strconv.Itoa(count))
	}
	return resp, b.SendAuthenticatedHTTPRequest(http.MethodGet, btseWalletHistory, true, urlValues, nil, &resp, queryFunc)
}

// GetWalletAddress returns the users account balance
func (b *BTSE) GetWalletAddress(currency string) (WalletAddress, error) {
	var resp WalletAddress

	urlValues := url.Values{}
	if currency != "" {
		urlValues.Add("currency", currency)
	}

	return resp, b.SendAuthenticatedHTTPRequest(http.MethodGet, btseWalletAddress, true, urlValues, nil, &resp, queryFunc)
}

// CreateWalletAddress create new deposit address for requested currency
func (b *BTSE) CreateWalletAddress(currency string) (WalletAddress, error) {
	var resp WalletAddress
	req := make(map[string]interface{}, 1)
	req["currency"] = currency
	err := b.SendAuthenticatedHTTPRequest(http.MethodPost, btseWalletAddress, true, nil, req, &resp, queryFunc)
	if err != nil {
		errResp := ErrorResponse{}
		errResponseStr := strings.Split(err.Error(), "raw response: ")
		err := json.Unmarshal([]byte(errResponseStr[1]), &errResp)
		if err != nil {
			return resp, err
		}
		if errResp.ErrorCode == 3528 {
			walletAddress := strings.Split(errResp.Message, "BADREQUEST: ")
			return WalletAddress{
				{
					Address: walletAddress[1],
				},
			}, nil
		}
		return resp, err
	}

	return resp, nil
}

// WalletWithdrawal submit request to withdraw crypto currency
func (b *BTSE) WalletWithdrawal(currency, address, tag, amount string) (WithdrawalResponse, error) {
	var resp WithdrawalResponse
	req := make(map[string]interface{}, 4)
	req["currency"] = currency
	req["address"] = address
	req["tag"] = tag
	req["amount"] = amount
	return resp, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseWalletWithdrawal, true, nil, req, &resp, queryFunc)
}

// CreateOrder creates an order
func (b *BTSE) CreateOrder(size, price float64, side, orderType, symbol, timeInForce, tag string) (*string, error) {
	req := make(map[string]interface{})
	req["size"] = size
	req["price"] = price
	if side != "" {
		req["side"] = side
	}
	if orderType != "" {
		req["type"] = orderType
	}
	if symbol != "" {
		req["symbol"] = symbol
	}
	if timeInForce != "" {
		req["time_in_force"] = timeInForce
	}
	if tag != "" {
		req["tag"] = tag
	}

	type orderResp struct {
		ID string `json:"id"`
	}

	var r orderResp
	return &r.ID, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseOrder, true, url.Values{}, req, &r, orderFunc)
}

// GetOrders returns all pending orders
func (b *BTSE) GetOrders(symbol, orderID, clOrderID string) ([]OpenOrder, error) {
	req := url.Values{}
	if orderID != "" {
		req.Add("orderID", orderID)
	}
	req.Add("symbol", symbol)
	if clOrderID != "" {
		req.Add("clOrderID", clOrderID)
	}
	var o []OpenOrder
	return o, b.SendAuthenticatedHTTPRequest(http.MethodGet, btsePendingOrders, true, req, nil, &o, orderFunc)
}

// CancelExistingOrder cancels an order
func (b *BTSE) CancelExistingOrder(orderID, symbol, clOrderID string) (CancelOrder, error) {
	var c CancelOrder
	req := url.Values{}
	if orderID != "" {
		req.Add("orderID", orderID)
	}
	req.Add("symbol", symbol)
	if clOrderID != "" {
		req.Add("clOrderID", clOrderID)
	}

	return c, b.SendAuthenticatedHTTPRequest(http.MethodDelete, btseOrder, true, req, nil, &c, orderFunc)
}

// CancelAllAfter cancels all orders after timeout
func (b *BTSE) CancelAllAfter(timeout int) error {
	req := make(map[string]interface{})
	req["timeout"] = timeout
	return b.SendAuthenticatedHTTPRequest(http.MethodPost, btseCancelAllAfter, true, url.Values{}, req, nil, orderFunc)
}

// TradeHistory returns previous trades on exchange
func (b *BTSE) TradeHistory(symbol, orderID string, start, end time.Time, count int) (TradeHistory, error) {
	var resp TradeHistory

	urlValues := url.Values{}
	if symbol != "" {
		urlValues.Add("symbol", symbol)
	}
	if orderID != "" {
		urlValues.Add("orderID", orderID)
	}
	urlValues.Add("count", strconv.Itoa(count))

	if !start.IsZero() && !end.IsZero() {
		if start.After(end) || end.Before(start) {
			return resp, errors.New("start and end must both be valid")
		}
		urlValues.Add("start", strconv.FormatInt(start.Unix(), 10))
		urlValues.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	return resp, b.SendAuthenticatedHTTPRequest(http.MethodGet, btseExchangeHistory, true, urlValues, nil, &resp, queryFunc)
}

// SendHTTPRequest sends an HTTP request to the desired endpoint
func (b *BTSE) SendHTTPRequest(method, endpoint string, result interface{}, spotEndpoint bool, f request.EndpointLimit) error {
	p := btseSPOTPath + btseSPOTAPIPath
	if !spotEndpoint {
		p = btseFuturesPath + btseFuturesAPIPath
	}
	return b.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          b.API.Endpoints.URL + p + endpoint,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to the desired endpoint
func (b *BTSE) SendAuthenticatedHTTPRequest(method, endpoint string, isSpot bool, values url.Values, req map[string]interface{}, result interface{}, f request.EndpointLimit) error {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			b.Name)
	}

	// The concatenation is done this way because BTSE expect endpoint+nonce or endpoint+nonce+body
	// when signing the data but the full path of the request  is /spot/api/v3.2/<endpoint>
	// its messy but it works and supports futures as well
	host := b.API.Endpoints.URL
	if isSpot {
		host += btseSPOTPath + btseSPOTAPIPath + endpoint
		endpoint = btseSPOTAPIPath + endpoint
	} else {
		host += btseFuturesPath + btseFuturesAPIPath
		endpoint += btseFuturesAPIPath
	}
	var hmac []byte
	var body io.Reader
	nonce := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	headers := map[string]string{
		"btse-api":   b.API.Credentials.Key,
		"btse-nonce": nonce,
	}
	if req != nil {
		reqPayload, err := json.Marshal(req)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(reqPayload)
		hmac = crypto.GetHMAC(
			crypto.HashSHA512_384,
			[]byte((endpoint + nonce + string(reqPayload))),
			[]byte(b.API.Credentials.Secret),
		)
		headers["Content-Type"] = "application/json"
	} else {
		hmac = crypto.GetHMAC(
			crypto.HashSHA512_384,
			[]byte((endpoint + nonce)),
			[]byte(b.API.Credentials.Secret),
		)
		if len(values) > 0 {
			host += "?" + values.Encode()
		}
	}
	headers["btse-sign"] = crypto.HexEncodeToString(hmac)

	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Sending %s request to URL %s",
			b.Name, method, endpoint)
	}

	return b.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          host,
		Headers:       headers,
		Body:          body,
		Result:        result,
		AuthRequest:   true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}

// GetFee returns an estimate of fee based on type of transaction
func (b *BTSE) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = b.calculateTradingFee(feeBuilder) * feeBuilder.Amount * feeBuilder.PurchasePrice
	case exchange.CryptocurrencyWithdrawalFee:
		switch feeBuilder.Pair.Base {
		case currency.USDT:
			fee = 1.08
		case currency.TUSD:
			fee = 1.09
		case currency.BTC:
			fee = 0.0005
		case currency.ETH:
			fee = 0.01
		case currency.LTC:
			fee = 0.001
		}
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.Amount)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.Amount)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.001 * price * amount
}

// getInternationalBankDepositFee returns international deposit fee
// Only when the initial deposit amount is less than $1000 or equivalent,
// BTSE will charge a small fee (0.25% or $3 USD equivalent, whichever is greater).
// The small deposit fee is charged in whatever currency it comes in.
func getInternationalBankDepositFee(amount float64) float64 {
	var fee float64
	if amount <= 100 {
		fee = amount * 0.0025
		if fee < 3 {
			return 3
		}
	}
	return fee
}

// getInternationalBankWithdrawalFee returns international withdrawal fee
// 0.1% (min25 USD)
func getInternationalBankWithdrawalFee(amount float64) float64 {
	fee := amount * 0.0009

	if fee < 25 {
		return 25
	}
	return fee
}

// calculateTradingFee return fee based on users current fee tier or default values
func (b *BTSE) calculateTradingFee(feeBuilder *exchange.FeeBuilder) float64 {
	formattedPair, err := b.FormatExchangeCurrency(feeBuilder.Pair, asset.Spot)
	if err != nil {
		if feeBuilder.IsMaker {
			return 0.001
		}
		return 0.002
	}
	feeTiers, err := b.GetFeeInformation(formattedPair.String())
	if err != nil {
		if feeBuilder.IsMaker {
			return 0.001
		}
		return 0.002
	}
	if feeBuilder.IsMaker {
		return feeTiers[0].MakerFee
	}
	return feeTiers[0].TakerFee
}

func parseOrderTime(timeStr string) (time.Time, error) {
	return time.Parse(btseTimeLayout, timeStr)
}
