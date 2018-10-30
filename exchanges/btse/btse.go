package btse

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// BTSE is the overarching type across this package
type BTSE struct {
	exchange.Base
	WebsocketConn *websocket.Conn
}

const (
	btseAPIURL     = "https://api.btse.com/v1/restapi"
	btseAPIVersion = "1"

	// Public endpoints
	btseMarkets = "markets"
	btseTrades  = "trades"
	btseTicker  = "ticker"
	btseStats   = "stats"
	btseTime    = "time"

	// Authenticated endpoints
	btseAccount       = "account"
	btseOrder         = "order"
	btsePendingOrders = "pending"
	btseDeleteOrder   = "deleteOrder"
	btseDeleteOrders  = "deleteOrders"
	btseFills         = "fills"
)

// GetMarkets returns a list of markets available on BTSE
func (b *BTSE) GetMarkets() (*Markets, error) {
	var m Markets
	return &m, b.SendHTTPRequest(http.MethodGet, btseMarkets, &m)
}

// GetTrades returns a list of trades for the specified symbol
func (b *BTSE) GetTrades(symbol string) (*Trades, error) {
	var t Trades
	endpoint := fmt.Sprintf("%s/%s", btseTrades, symbol)
	return &t, b.SendHTTPRequest(http.MethodGet, endpoint, &t)

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
func (b *BTSE) GetAccountBalance() (*AccountBalance, error) {
	var a AccountBalance
	return &a, b.SendAuthenticatedHTTPRequest(http.MethodGet, btseAccount, nil, &a)
}

// CreateOrder creates an order
func (b *BTSE) CreateOrder(amount, price float64, side, orderType, symbol, timeInForce, tag string) (*string, error) {
	req := make(map[string]interface{})
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	req["side"] = side
	req["type"] = orderType
	req["product_id"] = symbol

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
func (b *BTSE) GetOrders(productID string) (*OpenOrders, error) {
	req := make(map[string]interface{})
	if productID != "" {
		req["product_id"] = productID
	}
	var o OpenOrders
	return &o, b.SendAuthenticatedHTTPRequest(http.MethodGet, btsePendingOrders, req, &o)
}

// CancelExistingOrder cancels an order
func (b *BTSE) CancelExistingOrder(orderID, productID string) (*CancelOrder, error) {
	var c CancelOrder
	req := make(map[string]interface{})
	req["order_id"] = orderID
	req["product_id"] = productID
	return &c, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseDeleteOrder, req, &c)
}

// CancelOrders cancels all orders
// productID optional. If product ID is sent, all orders of that specified market
// will be cancelled. If not specified, all orders of all markets will be cancelled
func (b *BTSE) CancelOrders(productID string) (*CancelOrder, error) {
	var c CancelOrder
	req := make(map[string]interface{})
	if productID != "" {
		req["product_id"] = productID
	}
	return &c, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseDeleteOrders, req, &c)
}

// GetFills gets all filled orders
func (b *BTSE) GetFills(orderID, productID, before, after, limit string) (*FilledOrders, error) {
	if orderID != "" && productID != "" {
		return nil, errors.New("orderID and productID cannot co-exist in the same query")
	} else if orderID == "" && productID == "" {
		return nil, errors.New("orderID OR productID must be set")
	}

	req := make(map[string]interface{})
	if orderID != "" {
		req["order_id"] = orderID
	}

	if productID != "" {
		req["product_id"] = productID
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

	var o FilledOrders
	return &o, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseFills, req, &o)
}

// SendHTTPRequest sends an HTTP request to the desired endpoint
func (b *BTSE) SendHTTPRequest(method, endpoint string, result interface{}) error {
	p := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, endpoint)
	return b.SendPayload(method, p, nil, nil, &result, false, false, b.Verbose)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to the desired endpoint
func (b *BTSE) SendAuthenticatedHTTPRequest(method, endpoint string, req map[string]interface{}, result interface{}) error {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			b.Name)
	}

	payload, err := common.JSONEncode(req)
	if err != nil {
		return errors.New("sendAuthenticatedAPIRequest: unable to JSON request")
	}

	headers := make(map[string]string)
	headers["API-KEY"] = b.API.Credentials.Key
	headers["API-PASSPHRASE"] = b.API.Credentials.Secret
	if len(payload) > 0 {
		headers["Content-Type"] = "application/json"
	}

	p := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, endpoint)
	if b.Verbose {
		log.Debugf("Sending %s request to URL %s with params %s\n", method, p, string(payload))
	}
	return b.SendPayload(method, p, headers, strings.NewReader(string(payload)),
		&result, true, false, b.Verbose)
}

// GetFee returns an estimate of fee based on type of transaction
func (b *BTSE) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		if feeBuilder.Pair.Base.Match(currency.BTC) {
			fee = 0.0005
		} else if feeBuilder.Pair.Base.Match(currency.USDT) {
			fee = 5
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
	return 0.0015 * price * amount
}

// getInternationalBankDepositFee returns international deposit fee
// Only when the initial deposit amount is less than $1000 or equivalent,
// BTSE will charge a small fee (0.25% or $3 USD equivalent, whichever is greater).
// The small deposit fee is charged in whatever currency it comes in.
func getInternationalBankDepositFee(amount float64) float64 {
	var fee float64
	if amount <= 1000 {
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
	fee := amount * 0.001

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
		fee = 0.0015
	}
	return fee
}

func parseOrderTime(timeStr string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:04", timeStr)
	return t
}
