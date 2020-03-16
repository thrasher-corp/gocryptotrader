package btse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// BTSE is the overarching type across this package
type BTSE struct {
	exchange.Base
	WebsocketConn *wshandler.WebsocketConnection
}

const (
	btseAPIURL  = "https://api.btse.com"
	btseAPIPath = "/spot/v2/"

	// Public endpoints
	btseMarketOverview = "market_summary"
	btseMarkets        = "markets"
	btseOrderbook      = "orderbook"
	btseTrades         = "trades"
	btseTicker         = "ticker"
	btseStats          = "stats"
	btseTime           = "time"

	// Authenticated endpoints
	btseAccount       = "account"
	btseOrder         = "order"
	btsePendingOrders = "pending"
	btseDeleteOrder   = "deleteOrder"
	btseFills         = "fills"
	btseTimeLayout    = "2006-01-02 15:04:04"
)

// GetMarketsSummary stores market summary data
func (b *BTSE) GetMarketsSummary() (*HighLevelMarketData, error) {
	var m HighLevelMarketData
	return &m, b.SendHTTPRequest(http.MethodGet, btseMarketOverview, &m)
}

// GetMarkets returns a list of markets available on BTSE
func (b *BTSE) GetMarkets() ([]Market, error) {
	var m []Market
	return m, b.SendHTTPRequest(http.MethodGet, btseMarkets, &m)
}

// FetchOrderBook gets orderbook data for a given pair
func (b *BTSE) FetchOrderBook(symbol string) (*Orderbook, error) {
	var o Orderbook
	endpoint := fmt.Sprintf("%s/%s", btseOrderbook, symbol)
	return &o, b.SendHTTPRequest(http.MethodGet, endpoint, &o)
}

// GetTrades returns a list of trades for the specified symbol
func (b *BTSE) GetTrades(symbol string) ([]Trade, error) {
	var t []Trade
	endpoint := fmt.Sprintf("%s/%s", btseTrades, symbol)
	return t, b.SendHTTPRequest(http.MethodGet, endpoint, &t)
}

// GetTicker returns the ticker for a specified symbol
func (b *BTSE) GetTicker(symbol string) (*Ticker, error) {
	var t Ticker
	endpoint := fmt.Sprintf("%s/%s", btseTicker, symbol)
	err := b.SendHTTPRequest(http.MethodGet, endpoint, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetMarketStatistics gets market statistics for a specificed market
func (b *BTSE) GetMarketStatistics(symbol string) (*MarketStatistics, error) {
	var m MarketStatistics
	endpoint := fmt.Sprintf("%s/%s", btseStats, symbol)
	return &m, b.SendHTTPRequest(http.MethodGet, endpoint, &m)
}

// GetServerTime returns the exchanges server time
func (b *BTSE) GetServerTime() (*ServerTime, error) {
	var s ServerTime
	return &s, b.SendHTTPRequest(http.MethodGet, btseTime, &s)
}

// GetAccountBalance returns the users account balance
func (b *BTSE) GetAccountBalance() ([]CurrencyBalance, error) {
	var a []CurrencyBalance
	return a, b.SendAuthenticatedHTTPRequest(http.MethodGet, btseAccount, nil, &a)
}

// CreateOrder creates an order
func (b *BTSE) CreateOrder(amount, price float64, side, orderType, symbol, timeInForce, tag string) (*string, error) {
	req := make(map[string]interface{})
	req["amount"] = amount
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
	return &r.ID, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseOrder, req, &r)
}

// GetOrders returns all pending orders
func (b *BTSE) GetOrders(symbol string) ([]OpenOrder, error) {
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	var o []OpenOrder
	return o, b.SendAuthenticatedHTTPRequest(http.MethodGet, btsePendingOrders, req, &o)
}

// CancelExistingOrder cancels an order
func (b *BTSE) CancelExistingOrder(orderID, symbol string) (*CancelOrder, error) {
	var c CancelOrder
	req := make(map[string]interface{})
	req["order_id"] = orderID
	req["symbol"] = symbol
	return &c, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseDeleteOrder, req, &c)
}

// GetFills gets all filled orders
func (b *BTSE) GetFills(orderID, symbol, before, after, limit, username string) ([]FilledOrder, error) {
	if orderID != "" && symbol != "" {
		return nil, errors.New("orderID and symbol cannot co-exist in the same query")
	} else if orderID == "" && symbol == "" {
		return nil, errors.New("orderID OR symbol must be set")
	}

	req := make(map[string]interface{})
	if orderID != "" {
		req["order_id"] = orderID
	}

	if symbol != "" {
		req["symbol"] = symbol
	}

	if before != "" {
		req["before"] = before
	}

	if after != "" {
		req["after"] = after
	}

	if limit != "" {
		req["limit"] = limit
	}
	if username != "" {
		req["username"] = username
	}

	var o []FilledOrder
	return o, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseFills, req, &o)
}

// SendHTTPRequest sends an HTTP request to the desired endpoint
func (b *BTSE) SendHTTPRequest(method, endpoint string, result interface{}) error {
	return b.SendPayload(&request.Item{
		Method:        method,
		Path:          b.API.Endpoints.URL + btseAPIPath + endpoint,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to the desired endpoint
func (b *BTSE) SendAuthenticatedHTTPRequest(method, endpoint string, req map[string]interface{}, result interface{}) error {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			b.Name)
	}
	path := btseAPIPath + endpoint
	headers := make(map[string]string)
	headers["btse-api"] = b.API.Credentials.Key
	nonce := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	headers["btse-nonce"] = nonce
	var body io.Reader
	var hmac []byte
	var payload []byte
	if len(req) != 0 {
		var err error
		payload, err = json.Marshal(req)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
		hmac = crypto.GetHMAC(
			crypto.HashSHA512_384,
			[]byte((path + nonce + string(payload))),
			[]byte(b.API.Credentials.Secret),
		)
	} else {
		hmac = crypto.GetHMAC(
			crypto.HashSHA512_384,
			[]byte((path + nonce)),
			[]byte(b.API.Credentials.Secret),
		)
	}
	headers["btse-sign"] = crypto.HexEncodeToString(hmac)
	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Sending %s request to URL %s with params %s\n",
			b.Name, method, path, string(payload))
	}

	return b.SendPayload(&request.Item{
		Method:        method,
		Path:          b.API.Endpoints.URL + path,
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
		fee = calculateTradingFee(feeBuilder.IsMaker) * feeBuilder.Amount * feeBuilder.PurchasePrice
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

// calculateTradingFee BTSE has fee tiers, but does not disclose them via API,
// so the largest fee has to be assumed
func calculateTradingFee(isMaker bool) float64 {
	fee := 0.00050
	if !isMaker {
		fee = 0.001
	}
	return fee
}

func parseOrderTime(timeStr string) (time.Time, error) {
	return time.Parse(btseTimeLayout, timeStr)
}
