package bittrex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Bittrex is the overaching type across the bittrex methods
type Bittrex struct {
	exchange.Base
	WsSequenceOrders int64

	obm         *orderbookManager
	tickerCache *TickerCache
}

const (
	bittrexAPIRestURL       = "https://api.bittrex.com/v3"
	bittrexAPIDeprecatedURL = "https://bittrex.com/api/v1.1"

	// Public endpoints
	getMarkets           = "/markets"
	getMarketSummaries   = "/markets/summaries"
	getTicker            = "/markets/%s/ticker"
	getMarketSummary     = "/markets/%s/summary"
	getMarketTrades      = "/markets/%s/trades"
	getOrderbook         = "/markets/%s/orderbook?depth=%s"
	getRecentCandles     = "/markets/%s/candles/%s/%s/recent"
	getHistoricalCandles = "/markets/%s/candles/%s/%s/historical/%s"
	getCurrencies        = "/currencies"

	// Authenticated endpoints
	getBalances          = "/balances"
	getBalance           = "/balances/%s"
	getDepositAddress    = "/addresses/%s"
	getAllOpenOrders     = "/orders/open"
	getOpenOrders        = "/orders/open?marketSymbol=%s"
	getOrder             = "/orders/%s"
	getClosedOrders      = "/orders/closed?marketSymbol=%s"
	cancelOrder          = "/orders/%s"
	cancelOpenOrders     = "/orders/open"
	getClosedWithdrawals = "/withdrawals/closed"
	getOpenWithdrawals   = "/withdrawals/open"
	submitWithdrawal     = "/transfers"
	getClosedDeposits    = "/deposits/closed"
	getOpenDeposits      = "/deposits/open"
	submitOrder          = "/orders"

	// Other Consts
	ratePeriod     = time.Minute
	rateLimit      = 60
	orderbookDepth = 500 // ws uses REST snapshots and needs identical depths
)

// GetMarkets is used to get the open and available trading markets at Bittrex
// along with other meta data.
func (b *Bittrex) GetMarkets() ([]MarketData, error) {
	var resp []MarketData
	return resp, b.SendHTTPRequest(exchange.RestSpot, getMarkets, &resp, nil)
}

// GetCurrencies is used to get all supported currencies at Bittrex
func (b *Bittrex) GetCurrencies() ([]CurrencyData, error) {
	var resp []CurrencyData
	return resp, b.SendHTTPRequest(exchange.RestSpot, getCurrencies, &resp, nil)
}

// GetTicker sends a public get request and returns current ticker information
// on the supplied currency. Example currency input param "ltc-btc".
func (b *Bittrex) GetTicker(marketName string) (TickerData, error) {
	var resp TickerData
	return resp, b.SendHTTPRequest(exchange.RestSpot, fmt.Sprintf(getTicker, marketName), &resp, nil)
}

// GetMarketSummaries is used to get the last 24 hour summary of all active
// exchanges
func (b *Bittrex) GetMarketSummaries() ([]MarketSummaryData, error) {
	var resp []MarketSummaryData
	return resp, b.SendHTTPRequest(exchange.RestSpot, getMarketSummaries, &resp, nil)
}

// GetMarketSummary is used to get the last 24 hour summary of all active
// exchanges by currency pair (ltc-btc).
func (b *Bittrex) GetMarketSummary(marketName string) (MarketSummaryData, error) {
	var resp MarketSummaryData
	return resp, b.SendHTTPRequest(exchange.RestSpot, fmt.Sprintf(getMarketSummary, marketName), &resp, nil)
}

// GetOrderbook method returns current order book information by currency and depth.
// "marketSymbol" ie ltc-btc
// "depth" is either 1, 25 or 500. Server side, the depth defaults to 25.
func (b *Bittrex) GetOrderbook(marketName string, depth int64) (OrderbookData, int64, error) {
	strDepth := strconv.FormatInt(depth, 10)

	var resp OrderbookData
	var sequence int64
	resultHeader := http.Header{}
	err := b.SendHTTPRequest(exchange.RestSpot, fmt.Sprintf(getOrderbook, marketName, strDepth), &resp, &resultHeader)
	if err != nil {
		return OrderbookData{}, 0, err
	}
	sequence, err = strconv.ParseInt(resultHeader.Get("sequence"), 10, 64)
	if err != nil {
		return OrderbookData{}, 0, err
	}

	return resp, sequence, nil
}

// GetMarketHistory retrieves the latest trades that have occurred for a specific market
func (b *Bittrex) GetMarketHistory(currency string) ([]TradeData, error) {
	var resp []TradeData
	return resp, b.SendHTTPRequest(exchange.RestSpot, fmt.Sprintf(getMarketTrades, currency), &resp, nil)
}

// Order places an order
func (b *Bittrex) Order(marketName, side, orderType string, timeInForce TimeInForce, price, amount, ceiling float64) (OrderData, error) {
	req := make(map[string]interface{})
	req["marketSymbol"] = marketName
	req["direction"] = side
	req["type"] = orderType
	req["quantity"] = strconv.FormatFloat(amount, 'f', -1, 64)
	if orderType == "CEILING_LIMIT" || orderType == "CEILING_MARKET" {
		req["ceiling"] = strconv.FormatFloat(ceiling, 'f', -1, 64)
	}
	if orderType == "LIMIT" {
		req["limit"] = strconv.FormatFloat(price, 'f', -1, 64)
	}
	if timeInForce != "" {
		req["timeInForce"] = timeInForce
	} else {
		req["timeInForce"] = GoodTilCancelled
	}
	var resp OrderData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, submitOrder, nil, req, &resp, nil)
}

// GetOpenOrders returns all orders that you currently have opened.
// A specific market can be requested for example "ltc-btc"
func (b *Bittrex) GetOpenOrders(marketName string) ([]OrderData, int64, error) {
	var path string
	if marketName == "" || marketName == " " {
		path = getAllOpenOrders
	} else {
		path = fmt.Sprintf(getOpenOrders, marketName)
	}
	var resp []OrderData
	var sequence int64
	resultHeader := http.Header{}
	err := b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, &resultHeader)
	if err != nil {
		return nil, 0, err
	}
	sequence, err = strconv.ParseInt(resultHeader.Get("sequence"), 10, 64)
	if err != nil {
		return nil, 0, err
	}
	return resp, sequence, err
}

// CancelExistingOrder is used to cancel a buy or sell order.
func (b *Bittrex) CancelExistingOrder(uuid string) (OrderData, error) {
	var resp OrderData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodDelete, fmt.Sprintf(cancelOrder, uuid), nil, nil, &resp, nil)
}

// CancelOpenOrders is used to cancel all open orders for a specific market
// Or cancel all orders for all markets if the parameter `markets` is set to ""
func (b *Bittrex) CancelOpenOrders(market string) ([]BulkCancelResultData, error) {
	var resp []BulkCancelResultData

	params := url.Values{}
	if market != "" {
		params.Set("marketSymbol", market)
	}

	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodDelete, cancelOpenOrders, params, nil, &resp, nil)
}

// GetRecentCandles retrieves recent candles;
// Interval: MINUTE_1, MINUTE_5, HOUR_1, or DAY_1
// Type: TRADE or MIDPOINT
func (b *Bittrex) GetRecentCandles(marketName, candleInterval, candleType string) ([]CandleData, error) {
	var resp []CandleData

	return resp, b.SendHTTPRequest(exchange.RestSpot, fmt.Sprintf(getRecentCandles, marketName, candleType, candleInterval), &resp, nil)
}

// GetHistoricalCandles retrieves recent candles
// Type: TRADE or MIDPOINT
func (b *Bittrex) GetHistoricalCandles(marketName, candleInterval, candleType string, year, month, day int) ([]CandleData, error) {
	var resp []CandleData

	var start string
	switch candleInterval {
	case "MINUTE_1", "MINUTE_5":
		// Retrieve full day
		start = fmt.Sprintf("%d/%d/%d", year, month, day)
	case "HOUR_1":
		// Retrieve full month
		start = fmt.Sprintf("%d/%d", year, month)
	case "DAY_1":
		// Retrieve full year
		start = fmt.Sprintf("%d", year)
	default:
		return resp, fmt.Errorf("invalid interval %v, not supported", candleInterval)
	}

	return resp, b.SendHTTPRequest(exchange.RestSpot, fmt.Sprintf(getHistoricalCandles, marketName, candleType, candleInterval, start), &resp, nil)
}

// GetBalances is used to retrieve all balances from your account
func (b *Bittrex) GetBalances() ([]BalanceData, error) {
	var resp []BalanceData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getBalances, nil, nil, &resp, nil)
}

// GetAccountBalanceByCurrency is used to retrieve the balance from your account
// for a specific currency. ie. "btc" or "ltc"
func (b *Bittrex) GetAccountBalanceByCurrency(currency string) (BalanceData, error) {
	var resp BalanceData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, fmt.Sprintf(getBalance, currency), nil, nil, &resp, nil)
}

// GetCryptoDepositAddress is used to retrieve an address for a specific currency
func (b *Bittrex) GetCryptoDepositAddress(currency string) (AddressData, error) {
	var resp AddressData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, fmt.Sprintf(getDepositAddress, currency), nil, nil, &resp, nil)
}

// Withdraw is used to withdraw funds from your account.
func (b *Bittrex) Withdraw(currency, paymentID, address string, quantity float64) (WithdrawalData, error) {
	req := make(map[string]interface{})
	req["currencySymbol"] = currency
	req["quantity"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	req["cryptoAddress"] = address
	if len(paymentID) > 0 {
		req["cryptoAddressTag"] = paymentID
	}
	var resp WithdrawalData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, submitWithdrawal, nil, req, &resp, nil)
}

// GetOrder is used to retrieve a single order by UUID.
func (b *Bittrex) GetOrder(uuid string) (OrderData, error) {
	var resp OrderData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, fmt.Sprintf(getOrder, uuid), nil, nil, &resp, nil)
}

// GetOrderHistoryForCurrency is used to retrieve your order history. If marketName
// is omitted it will return the entire order History.
func (b *Bittrex) GetOrderHistoryForCurrency(currency string) ([]OrderData, error) {
	var resp []OrderData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, fmt.Sprintf(getClosedOrders, currency), nil, nil, &resp, nil)
}

// GetClosedWithdrawals is used to retrieve your withdrawal history.
func (b *Bittrex) GetClosedWithdrawals() ([]WithdrawalData, error) {
	var resp []WithdrawalData

	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getClosedWithdrawals, nil, nil, &resp, nil)
}

// GetClosedWithdrawalsForCurrency is used to retrieve your withdrawal history for the specified currency.
func (b *Bittrex) GetClosedWithdrawalsForCurrency(currency string) ([]WithdrawalData, error) {
	var resp []WithdrawalData

	params := url.Values{}
	params.Set("currencySymbol", currency)

	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getClosedWithdrawals, params, nil, &resp, nil)
}

// GetOpenWithdrawals is used to retrieve your withdrawal history. If currency
// omitted it will return the entire history
func (b *Bittrex) GetOpenWithdrawals() ([]WithdrawalData, error) {
	var resp []WithdrawalData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getOpenWithdrawals, nil, nil, &resp, nil)
}

// GetClosedDeposits is used to retrieve your deposit history.
func (b *Bittrex) GetClosedDeposits() ([]DepositData, error) {
	var resp []DepositData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getClosedDeposits, nil, nil, &resp, nil)
}

// GetClosedDepositsForCurrency is used to retrieve your deposit history for the specified currency
func (b *Bittrex) GetClosedDepositsForCurrency(currency string) ([]DepositData, error) {
	var resp []DepositData

	params := url.Values{}
	params.Set("currencySymbol", currency)

	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getClosedDeposits, params, nil, &resp, nil)
}

// GetClosedDepositsPaginated is used to retrieve your deposit history.
// The maximum page size is 200 and it defaults to 100.
// PreviousPageToken is the unique identifier of the item that the resulting
// query result should end before, in the sort order of the given endpoint. Used
// for traversing a paginated set in the reverse direction.
func (b *Bittrex) GetClosedDepositsPaginated(pageSize int, previousPageTokenOptional ...string) ([]DepositData, error) {
	var resp []DepositData

	params := url.Values{}
	params.Set("pageSize", strconv.Itoa(pageSize))

	if len(previousPageTokenOptional) > 0 {
		params.Set("previousPageToken", previousPageTokenOptional[0])
	}

	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getClosedDeposits, params, nil, &resp, nil)
}

// GetOpenDeposits is used to retrieve your open deposits.
func (b *Bittrex) GetOpenDeposits() ([]DepositData, error) {
	var resp []DepositData
	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getOpenDeposits, nil, nil, &resp, nil)
}

// GetOpenDepositsForCurrency is used to retrieve your open deposits for the specified currency
func (b *Bittrex) GetOpenDepositsForCurrency(currency string) ([]DepositData, error) {
	var resp []DepositData

	params := url.Values{}
	params.Set("currencySymbol", currency)

	return resp, b.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getOpenDeposits, params, nil, &resp, nil)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *Bittrex) SendHTTPRequest(ep exchange.URL, path string, result interface{}, resultHeader *http.Header) error {
	endpoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	requestItem := request.Item{
		Method:         http.MethodGet,
		Path:           endpoint + path,
		Result:         result,
		Verbose:        b.Verbose,
		HTTPDebugging:  b.HTTPDebugging,
		HTTPRecording:  b.HTTPRecording,
		HeaderResponse: resultHeader,
	}
	return b.SendPayload(context.Background(), &requestItem)
}

// SendAuthHTTPRequest sends an authenticated request
func (b *Bittrex) SendAuthHTTPRequest(ep exchange.URL, method, action string, params url.Values, data, result interface{}, resultHeader *http.Header) error {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", b.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	endpoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	ts := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)

	path := common.EncodeURLValues(action, params)

	var body io.Reader
	var hmac, payload []byte
	var contentHash string
	if data == nil {
		payload = []byte("")
	} else {
		var err error
		payload, err = json.Marshal(data)
		if err != nil {
			return err
		}
	}
	body = bytes.NewBuffer(payload)
	contentHash = crypto.HexEncodeToString(crypto.GetSHA512(payload))
	sigPayload := ts + endpoint + path + method + contentHash
	hmac = crypto.GetHMAC(crypto.HashSHA512, []byte(sigPayload), []byte(b.API.Credentials.Secret))

	headers := make(map[string]string)
	headers["Api-Key"] = b.API.Credentials.Key
	headers["Api-Timestamp"] = ts
	headers["Api-Content-Hash"] = contentHash
	headers["Api-Signature"] = crypto.HexEncodeToString(hmac)
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json"
	return b.SendPayload(context.Background(), &request.Item{
		Method:         method,
		Path:           endpoint + path,
		Headers:        headers,
		Body:           body,
		Result:         result,
		AuthRequest:    true,
		Verbose:        b.Verbose,
		HTTPDebugging:  b.HTTPDebugging,
		HTTPRecording:  b.HTTPRecording,
		HeaderResponse: resultHeader,
	})
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Bittrex) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	var err error

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	case exchange.CryptocurrencyWithdrawalFee:
		fee, err = b.GetWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.OfflineTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, err
}

// GetWithdrawalFee returns the fee for withdrawing from the exchange
func (b *Bittrex) GetWithdrawalFee(c currency.Code) (float64, error) {
	var fee float64

	currencies, err := b.GetCurrencies()
	if err != nil {
		return 0, err
	}
	for i := range currencies {
		if currencies[i].Symbol == c.String() {
			fee = currencies[i].TxFee
		}
	}
	return fee, nil
}

// calculateTradingFee returns the fee for trading any currency on Bittrex
func calculateTradingFee(price, amount float64) float64 {
	return 0.0025 * price * amount
}
