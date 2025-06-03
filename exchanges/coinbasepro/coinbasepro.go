package coinbasepro

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	coinbaseproAPIURL                  = "https://api.pro.coinbase.com/"
	coinbaseproSandboxAPIURL           = "https://api-public.sandbox.pro.coinbase.com/"
	tradeBaseURL                       = "https://www.coinbase.com/advanced-trade/spot/"
	coinbaseproAPIVersion              = "0"
	coinbaseproProducts                = "products"
	coinbaseproOrderbook               = "book"
	coinbaseproTicker                  = "ticker"
	coinbaseproTrades                  = "trades"
	coinbaseproHistory                 = "candles"
	coinbaseproStats                   = "stats"
	coinbaseproCurrencies              = "currencies"
	coinbaseproAccounts                = "accounts"
	coinbaseproLedger                  = "ledger"
	coinbaseproHolds                   = "holds"
	coinbaseproOrders                  = "orders"
	coinbaseproFills                   = "fills"
	coinbaseproTransfers               = "transfers"
	coinbaseproReports                 = "reports"
	coinbaseproTime                    = "time"
	coinbaseproMarginTransfer          = "profiles/margin-transfer"
	coinbaseproPosition                = "position"
	coinbaseproPositionClose           = "position/close"
	coinbaseproPaymentMethod           = "payment-methods"
	coinbaseproPaymentMethodDeposit    = "deposits/payment-method"
	coinbaseproDepositCoinbase         = "deposits/coinbase-account"
	coinbaseproWithdrawalPaymentMethod = "withdrawals/payment-method"
	coinbaseproWithdrawalCoinbase      = "withdrawals/coinbase"
	coinbaseproWithdrawalCrypto        = "withdrawals/crypto"
	coinbaseproCoinbaseAccounts        = "coinbase-accounts"
	coinbaseproTrailingVolume          = "users/self/trailing-volume"
)

// CoinbasePro is the overarching type across the coinbasepro package
type CoinbasePro struct {
	exchange.Base
}

// GetProducts returns supported currency pairs on the exchange with specific
// information about the pair
func (c *CoinbasePro) GetProducts(ctx context.Context) ([]Product, error) {
	var products []Product

	return products, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseproProducts, &products)
}

// GetOrderbook returns orderbook by currency pair and level
func (c *CoinbasePro) GetOrderbook(ctx context.Context, symbol string, level int) (any, error) {
	orderbook := OrderbookResponse{}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproProducts, symbol, coinbaseproOrderbook)
	if level > 0 {
		levelStr := strconv.Itoa(level)
		path = fmt.Sprintf("%s/%s/%s?level=%s", coinbaseproProducts, symbol, coinbaseproOrderbook, levelStr)
	}

	if err := c.SendHTTPRequest(ctx, exchange.RestSpot, path, &orderbook); err != nil {
		return nil, err
	}

	if level == 3 {
		ob := OrderbookL3{
			Sequence: orderbook.Sequence,
			Bids:     make([]OrderL3, len(orderbook.Bids)),
			Asks:     make([]OrderL3, len(orderbook.Asks)),
		}
		ob.Sequence = orderbook.Sequence
		for x := range orderbook.Asks {
			ob.Asks[x].Price = orderbook.Asks[x][0].Float64()
			ob.Asks[x].Amount = orderbook.Asks[x][1].Float64()
			ob.Asks[x].OrderID = orderbook.Asks[x][2].String()
		}
		for x := range orderbook.Bids {
			ob.Bids[x].Price = orderbook.Bids[x][0].Float64()
			ob.Bids[x].Amount = orderbook.Bids[x][1].Float64()
			ob.Bids[x].OrderID = orderbook.Bids[x][2].String()
		}
		return ob, nil
	}
	ob := OrderbookL1L2{
		Sequence: orderbook.Sequence,
		Bids:     make([]OrderL1L2, len(orderbook.Bids)),
		Asks:     make([]OrderL1L2, len(orderbook.Asks)),
	}
	for x := range orderbook.Asks {
		ob.Asks[x].Price = orderbook.Asks[x][0].Float64()
		ob.Asks[x].Amount = orderbook.Asks[x][1].Float64()
		ob.Asks[x].NumOrders = orderbook.Asks[x][2].Float64()
	}
	for x := range orderbook.Bids {
		ob.Bids[x].Price = orderbook.Bids[x][0].Float64()
		ob.Bids[x].Amount = orderbook.Bids[x][1].Float64()
		ob.Bids[x].NumOrders = orderbook.Bids[x][2].Float64()
	}
	return ob, nil
}

// GetTicker returns ticker by currency pair
// currencyPair - example "BTC-USD"
func (c *CoinbasePro) GetTicker(ctx context.Context, currencyPair string) (Ticker, error) {
	tick := Ticker{}
	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproTicker)
	return tick, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &tick)
}

// GetTrades listd the latest trades for a product
// currencyPair - example "BTC-USD"
func (c *CoinbasePro) GetTrades(ctx context.Context, currencyPair string) ([]Trade, error) {
	var trades []Trade
	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproTrades)
	return trades, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &trades)
}

// GetHistoricRates returns historic rates for a product. Rates are returned in
// grouped buckets based on requested granularity.
func (c *CoinbasePro) GetHistoricRates(ctx context.Context, currencyPair, start, end string, granularity int64) ([]History, error) {
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	} else {
		values.Set("start", "")
	}

	if end != "" {
		values.Set("end", end)
	} else {
		values.Set("end", "")
	}

	allowedGranularities := []int64{60, 300, 900, 3600, 21600, 86400}
	if !slices.Contains(allowedGranularities, granularity) {
		return nil, errors.New("Invalid granularity value: " + strconv.FormatInt(granularity, 10) + ". Allowed values are {60, 300, 900, 3600, 21600, 86400}")
	}
	if granularity > 0 {
		values.Set("granularity", strconv.FormatInt(granularity, 10))
	}

	var resp []History
	path := common.EncodeURLValues(
		fmt.Sprintf("%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproHistory),
		values)
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetStats returns a 24 hr stat for the product. Volume is in base currency
// units. open, high, low are in quote currency units.
func (c *CoinbasePro) GetStats(ctx context.Context, currencyPair string) (Stats, error) {
	stats := Stats{}
	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproStats)

	return stats, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &stats)
}

// GetCurrencies returns a list of supported currency on the exchange
// Warning: Not all currencies may be currently in use for tradinc.
func (c *CoinbasePro) GetCurrencies(ctx context.Context) ([]Currency, error) {
	var currencies []Currency

	return currencies, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseproCurrencies, &currencies)
}

// GetCurrentServerTime returns the API server time
func (c *CoinbasePro) GetCurrentServerTime(ctx context.Context) (ServerTime, error) {
	serverTime := ServerTime{}
	return serverTime, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseproTime, &serverTime)
}

// GetAccounts returns a list of trading accounts associated with the APIKEYS
func (c *CoinbasePro) GetAccounts(ctx context.Context) ([]AccountResponse, error) {
	var resp []AccountResponse

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproAccounts, nil, &resp)
}

// GetAccount returns information for a single account. Use this endpoint when
// account_id is known
func (c *CoinbasePro) GetAccount(ctx context.Context, accountID string) (AccountResponse, error) {
	resp := AccountResponse{}
	path := fmt.Sprintf("%s/%s", coinbaseproAccounts, accountID)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetAccountHistory returns a list of account activity. Account activity either
// increases or decreases your account balance. Items are paginated and sorted
// latest first.
func (c *CoinbasePro) GetAccountHistory(ctx context.Context, accountID string) ([]AccountLedgerResponse, error) {
	var resp []AccountLedgerResponse
	path := fmt.Sprintf("%s/%s/%s", coinbaseproAccounts, accountID, coinbaseproLedger)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetHolds returns the holds that are placed on an account for any active
// orders or pending withdraw requests. As an order is filled, the hold amount
// is updated. If an order is canceled, any remaining hold is removed. For a
// withdraw, once it is completed, the hold is removed.
func (c *CoinbasePro) GetHolds(ctx context.Context, accountID string) ([]AccountHolds, error) {
	var resp []AccountHolds
	path := fmt.Sprintf("%s/%s/%s", coinbaseproAccounts, accountID, coinbaseproHolds)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// PlaceLimitOrder places a new limit order. Orders can only be placed if the
// account has sufficient funds. Once an order is placed, account funds
// will be put on hold for the duration of the order. How much and which funds
// are put on hold depends on the order type and parameters specified.
//
// GENERAL PARAMS
// clientRef - [optional] Order ID selected by you to identify your order
// side - 	buy or sell
// productID - A valid product id
// stp - [optional] Self-trade prevention flag
//
// LIMIT ORDER PARAMS
// price - Price per bitcoin
// amount - Amount of BTC to buy or sell
// timeInforce - [optional] GTC, GTT, IOC, or FOK (default is GTC)
// cancelAfter - [optional] min, hour, day * Requires time_in_force to be GTT
// postOnly - [optional] Post only flag Invalid when time_in_force is IOC or FOK
func (c *CoinbasePro) PlaceLimitOrder(ctx context.Context, clientRef string, price, amount float64, side, timeInforce, cancelAfter, productID, stp string, postOnly bool) (string, error) {
	req := make(map[string]any)
	req["type"] = order.Limit.Lower()
	req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	req["size"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["side"] = side
	req["product_id"] = productID
	if cancelAfter != "" {
		req["cancel_after"] = cancelAfter
	}
	if timeInforce != "" {
		req["time_in_force"] = timeInforce
	}
	if clientRef != "" {
		req["client_oid"] = clientRef
	}
	if stp != "" {
		req["stp"] = stp
	}
	if postOnly {
		req["post_only"] = postOnly
	}
	resp := GeneralizedOrderResponse{}
	return resp.ID, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproOrders, req, &resp)
}

// PlaceMarketOrder places a new market order.
// Orders can only be placed if the account has sufficient funds. Once an order
// is placed, account funds will be put on hold for the duration of the order.
// How much and which funds are put on hold depends on the order type and
// parameters specified.
//
// GENERAL PARAMS
// clientRef - [optional] Order ID selected by you to identify your order
// side - 	buy or sell
// productID - A valid product id
// stp - [optional] Self-trade prevention flag
//
// MARKET ORDER PARAMS
// size - [optional]* Desired amount in BTC
// funds	[optional]* Desired amount of quote currency to use
// * One of size or funds is required.
func (c *CoinbasePro) PlaceMarketOrder(ctx context.Context, clientRef string, size, funds float64, side, productID, stp string) (string, error) {
	resp := GeneralizedOrderResponse{}
	req := make(map[string]any)
	req["side"] = side
	req["product_id"] = productID
	req["type"] = order.Market.Lower()

	if size != 0 {
		req["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	}
	if funds != 0 {
		req["funds"] = strconv.FormatFloat(funds, 'f', -1, 64)
	}
	if clientRef != "" {
		req["client_oid"] = clientRef
	}
	if stp != "" {
		req["stp"] = stp
	}

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproOrders, req, &resp)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// PlaceMarginOrder places a new market order.
// Orders can only be placed if the account has sufficient funds. Once an order
// is placed, account funds will be put on hold for the duration of the order.
// How much and which funds are put on hold depends on the order type and
// parameters specified.
//
// GENERAL PARAMS
// clientRef - [optional] Order ID selected by you to identify your order
// side - 	buy or sell
// productID - A valid product id
// stp - [optional] Self-trade prevention flag
//
// MARGIN ORDER PARAMS
// size - [optional]* Desired amount in BTC
// funds - [optional]* Desired amount of quote currency to use
func (c *CoinbasePro) PlaceMarginOrder(ctx context.Context, clientRef string, size, funds float64, side, productID, stp string) (string, error) {
	resp := GeneralizedOrderResponse{}
	req := make(map[string]any)
	req["side"] = side
	req["product_id"] = productID
	req["type"] = "margin"

	if size != 0 {
		req["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	}
	if funds != 0 {
		req["funds"] = strconv.FormatFloat(funds, 'f', -1, 64)
	}
	if clientRef != "" {
		req["client_oid"] = clientRef
	}
	if stp != "" {
		req["stp"] = stp
	}

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproOrders, req, &resp)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// CancelExistingOrder cancels order by orderID
func (c *CoinbasePro) CancelExistingOrder(ctx context.Context, orderID string) error {
	path := fmt.Sprintf("%s/%s", coinbaseproOrders, orderID)

	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil)
}

// CancelAllExistingOrders cancels all open orders on the exchange and returns
// and array of order IDs
// currencyPair - [optional] all orders for a currencyPair string will be
// canceled
func (c *CoinbasePro) CancelAllExistingOrders(ctx context.Context, currencyPair string) ([]string, error) {
	var resp []string
	req := make(map[string]any)

	if currencyPair != "" {
		req["product_id"] = currencyPair
	}
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, coinbaseproOrders, req, &resp)
}

// GetOrders lists current open orders. Only open or un-settled orders are
// returned. As soon as an order is no longer open and settled, it will no
// longer appear in the default request.
// status - can be a range of "open", "pending", "done" or "active"
// currencyPair - [optional] for example "BTC-USD"
func (c *CoinbasePro) GetOrders(ctx context.Context, status []string, currencyPair string) ([]GeneralizedOrderResponse, error) {
	var resp []GeneralizedOrderResponse
	params := url.Values{}

	for _, individualStatus := range status {
		params.Add("status", individualStatus)
	}
	if currencyPair != "" {
		params.Set("product_id", currencyPair)
	}

	path := common.EncodeURLValues(coinbaseproOrders, params)
	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetOrder returns a single order by order id.
func (c *CoinbasePro) GetOrder(ctx context.Context, orderID string) (GeneralizedOrderResponse, error) {
	resp := GeneralizedOrderResponse{}
	path := fmt.Sprintf("%s/%s", coinbaseproOrders, orderID)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetFills returns a list of recent fills
func (c *CoinbasePro) GetFills(ctx context.Context, orderID, currencyPair string) ([]FillResponse, error) {
	var resp []FillResponse
	params := url.Values{}

	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if currencyPair != "" {
		params.Set("product_id", currencyPair)
	}
	if params.Get("order_id") == "" && params.Get("product_id") == "" {
		return resp, errors.New("no parameters set")
	}

	path := common.EncodeURLValues(coinbaseproFills, params)
	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// MarginTransfer sends funds between a standard/default profile and a margin
// profile.
// A deposit will transfer funds from the default profile into the margin
// profile. A withdraw will transfer funds from the margin profile to the
// default profile. Withdraws will fail if they would set your margin ratio
// below the initial margin ratio requirement.
//
// amount - the amount to transfer between the default and margin profile
// transferType - either "deposit" or "withdraw"
// profileID - The id of the margin profile to deposit or withdraw from
// currency - currency to transfer, currently on "BTC" or "USD"
func (c *CoinbasePro) MarginTransfer(ctx context.Context, amount float64, transferType, profileID, currency string) (MarginTransfer, error) {
	resp := MarginTransfer{}
	req := make(map[string]any)
	req["type"] = transferType
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["currency"] = currency
	req["margin_profile_id"] = profileID

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproMarginTransfer, req, &resp)
}

// GetPosition returns an overview of account profile.
func (c *CoinbasePro) GetPosition(ctx context.Context) (AccountOverview, error) {
	resp := AccountOverview{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproPosition, nil, &resp)
}

// ClosePosition closes a position and allowing you to repay position as well
// repayOnly -  allows the position to be repaid
func (c *CoinbasePro) ClosePosition(ctx context.Context, repayOnly bool) (AccountOverview, error) {
	resp := AccountOverview{}
	req := make(map[string]any)
	req["repay_only"] = repayOnly

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproPositionClose, req, &resp)
}

// GetPayMethods returns a full list of payment methods
func (c *CoinbasePro) GetPayMethods(ctx context.Context) ([]PaymentMethod, error) {
	var resp []PaymentMethod

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproPaymentMethod, nil, &resp)
}

// DepositViaPaymentMethod deposits funds from a payment method. See the Payment
// Methods section for retrieving your payment methods.
//
// amount - The amount to deposit
// currency - The type of currency
// paymentID - ID of the payment method
func (c *CoinbasePro) DepositViaPaymentMethod(ctx context.Context, amount float64, currency, paymentID string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]any)
	req["amount"] = amount
	req["currency"] = currency
	req["payment_method_id"] = paymentID

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproPaymentMethodDeposit, req, &resp)
}

// DepositViaCoinbase deposits funds from a coinbase account. Move funds between
// a Coinbase account and coinbasepro trading account within daily limits. Moving
// funds between Coinbase and coinbasepro is instant and free. See the Coinbase
// Accounts section for retrieving your Coinbase accounts.
//
// amount - The amount to deposit
// currency - The type of currency
// accountID - ID of the coinbase account
func (c *CoinbasePro) DepositViaCoinbase(ctx context.Context, amount float64, currency, accountID string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]any)
	req["amount"] = amount
	req["currency"] = currency
	req["coinbase_account_id"] = accountID

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproDepositCoinbase, req, &resp)
}

// WithdrawViaPaymentMethod withdraws funds to a payment method
//
// amount - The amount to withdraw
// currency - The type of currency
// paymentID - ID of the payment method
func (c *CoinbasePro) WithdrawViaPaymentMethod(ctx context.Context, amount float64, currency, paymentID string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]any)
	req["amount"] = amount
	req["currency"] = currency
	req["payment_method_id"] = paymentID

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalPaymentMethod, req, &resp)
}

// /////////////////////// NO ROUTE FOUND ERROR ////////////////////////////////
// WithdrawViaCoinbase withdraws funds to a coinbase account.
//
// amount - The amount to withdraw
// currency - The type of currency
// accountID - 	ID of the coinbase account
// func (c *CoinbasePro) WithdrawViaCoinbase(amount float64, currency, accountID string) (DepositWithdrawalInfo, error) {
// 	resp := DepositWithdrawalInfo{}
// 	req := make(map[string]any)
// 	req["amount"] = amount
// 	req["currency"] = currency
// 	req["coinbase_account_id"] = accountID
//
// 	return resp,
// 		c.SendAuthenticatedHTTPRequest(ctx,http.MethodPost, coinbaseproWithdrawalCoinbase, req, &resp)
// }

// WithdrawCrypto withdraws funds to a crypto address
//
// amount - The amount to withdraw
// currency - The type of currency
// cryptoAddress - 	A crypto address of the recipient
func (c *CoinbasePro) WithdrawCrypto(ctx context.Context, amount float64, currency, cryptoAddress string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]any)
	req["amount"] = amount
	req["currency"] = currency
	req["crypto_address"] = cryptoAddress

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalCrypto, req, &resp)
}

// GetCoinbaseAccounts returns a list of coinbase accounts
func (c *CoinbasePro) GetCoinbaseAccounts(ctx context.Context) ([]CoinbaseAccounts, error) {
	var resp []CoinbaseAccounts

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproCoinbaseAccounts, nil, &resp)
}

// GetReport returns batches of historic information about your account in
// various human and machine readable forms.
//
// reportType - "fills" or "account"
// startDate - Starting date for the report (inclusive)
// endDate - Ending date for the report (inclusive)
// currencyPair - ID of the product to generate a fills report for.
// E.c. BTC-USD. *Required* if type is fills
// accountID - ID of the account to generate an account report for. *Required*
// if type is account
// format - 	pdf or csv (default is pdf)
// email - [optional] Email address to send the report to
func (c *CoinbasePro) GetReport(ctx context.Context, reportType, startDate, endDate, currencyPair, accountID, format, email string) (Report, error) {
	resp := Report{}
	req := make(map[string]any)
	req["type"] = reportType
	req["start_date"] = startDate
	req["end_date"] = endDate
	req["format"] = "pdf"

	if currencyPair != "" {
		req["product_id"] = currencyPair
	}
	if accountID != "" {
		req["account_id"] = accountID
	}
	if format == "csv" {
		req["format"] = format
	}
	if email != "" {
		req["email"] = email
	}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproReports, req, &resp)
}

// GetReportStatus once a report request has been accepted for processing, the
// status is available by polling the report resource endpoint.
func (c *CoinbasePro) GetReportStatus(ctx context.Context, reportID string) (Report, error) {
	resp := Report{}
	path := fmt.Sprintf("%s/%s", coinbaseproReports, reportID)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetTrailingVolume this request will return your 30-day trailing volume for
// all products.
func (c *CoinbasePro) GetTrailingVolume(ctx context.Context) ([]Volume, error) {
	var resp []Volume

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproTrailingVolume, nil, &resp)
}

// GetTransfers returns a history of withdrawal and or deposit transactions
func (c *CoinbasePro) GetTransfers(ctx context.Context, profileID, transferType string, limit int64, start, end time.Time) ([]TransferHistory, error) {
	if !start.IsZero() && !end.IsZero() {
		err := common.StartEndTimeCheck(start, end)
		if err != nil {
			return nil, err
		}
	}
	req := make(map[string]any)
	if profileID != "" {
		req["profile_id"] = profileID
	}
	if !start.IsZero() {
		req["before"] = start.Format(time.RFC3339)
	}
	if !end.IsZero() {
		req["after"] = end.Format(time.RFC3339)
	}
	if limit > 0 {
		req["limit"] = limit
	}
	if transferType != "" {
		req["type"] = transferType
	}
	var resp []TransferHistory
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproTransfers, req, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (c *CoinbasePro) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       c.Verbose,
		HTTPDebugging: c.HTTPDebugging,
		HTTPRecording: c.HTTPRecording,
	}

	return c.SendPayload(ctx, request.UnAuth, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (c *CoinbasePro) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, params map[string]any, result any) (err error) {
	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	newRequest := func() (*request.Item, error) {
		payload := []byte("")
		if params != nil {
			payload, err = json.Marshal(params)
			if err != nil {
				return nil, err
			}
		}

		n := strconv.FormatInt(time.Now().Unix(), 10)
		message := n + method + "/" + path + string(payload)

		hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		headers["CB-ACCESS-SIGN"] = base64.StdEncoding.EncodeToString(hmac)
		headers["CB-ACCESS-TIMESTAMP"] = n
		headers["CB-ACCESS-KEY"] = creds.Key
		headers["CB-ACCESS-PASSPHRASE"] = creds.ClientID
		headers["Content-Type"] = "application/json"

		return &request.Item{
			Method:        method,
			Path:          endpoint + path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        result,
			Verbose:       c.Verbose,
			HTTPDebugging: c.HTTPDebugging,
			HTTPRecording: c.HTTPRecording,
		}, nil
	}
	return c.SendPayload(ctx, request.Unset, newRequest, request.AuthenticatedRequest)
}

// GetFee returns an estimate of fee based on type of transaction
func (c *CoinbasePro) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		trailingVolume, err := c.GetTrailingVolume(ctx)
		if err != nil {
			return 0, err
		}
		fee = c.calculateTradingFee(trailingVolume,
			feeBuilder.Pair.Base,
			feeBuilder.Pair.Quote,
			feeBuilder.Pair.Delimiter,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency)
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.FiatCurrency)
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
	return 0.0025 * price * amount
}

func (c *CoinbasePro) calculateTradingFee(trailingVolume []Volume, base, quote currency.Code, delimiter string, purchasePrice, amount float64, isMaker bool) float64 {
	var fee float64
	for _, i := range trailingVolume {
		if strings.EqualFold(i.ProductID, base.String()+delimiter+quote.String()) {
			switch {
			case isMaker:
				fee = 0
			case i.Volume <= 10000000:
				fee = 0.003
			case i.Volume > 10000000 && i.Volume <= 100000000:
				fee = 0.002
			case i.Volume > 100000000:
				fee = 0.001
			}
			break
		}
	}
	return fee * amount * purchasePrice
}

func getInternationalBankWithdrawalFee(c currency.Code) float64 {
	var fee float64

	if c.Equal(currency.USD) {
		fee = 25
	} else if c.Equal(currency.EUR) {
		fee = 0.15
	}

	return fee
}

func getInternationalBankDepositFee(c currency.Code) float64 {
	var fee float64

	if c.Equal(currency.USD) {
		fee = 10
	} else if c.Equal(currency.EUR) {
		fee = 0.15
	}

	return fee
}
