package gdax

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	gdaxAPIURL                  = "https://api.gdax.com/"
	gdaxSandboxAPIURL           = "https://public.sandbox.gdax.com"
	gdaxAPIVersion              = "0"
	gdaxProducts                = "products"
	gdaxOrderbook               = "book"
	gdaxTicker                  = "ticker"
	gdaxTrades                  = "trades"
	gdaxHistory                 = "candles"
	gdaxStats                   = "stats"
	gdaxCurrencies              = "currencies"
	gdaxAccounts                = "accounts"
	gdaxLedger                  = "ledger"
	gdaxHolds                   = "holds"
	gdaxOrders                  = "orders"
	gdaxFills                   = "fills"
	gdaxTransfers               = "transfers"
	gdaxReports                 = "reports"
	gdaxTime                    = "time"
	gdaxMarginTransfer          = "profiles/margin-transfer"
	gdaxFunding                 = "funding"
	gdaxFundingRepay            = "funding/repay"
	gdaxPosition                = "position"
	gdaxPositionClose           = "position/close"
	gdaxPaymentMethod           = "payment-methods"
	gdaxPaymentMethodDeposit    = "deposits/payment-method"
	gdaxDepositCoinbase         = "deposits/coinbase-account"
	gdaxWithdrawalPaymentMethod = "withdrawals/payment-method"
	gdaxWithdrawalCoinbase      = "withdrawals/coinbase"
	gdaxWithdrawalCrypto        = "withdrawals/crypto"
	gdaxCoinbaseAccounts        = "coinbase-accounts"
	gdaxTrailingVolume          = "users/self/trailing-volume"
)

var sometin []string

// GDAX is the overarching type across the GDAX package
type GDAX struct {
	exchange.Base
}

// SetDefaults sets default values for the exchange
func (g *GDAX) SetDefaults() {
	g.Name = "GDAX"
	g.Enabled = false
	g.Verbose = false
	g.TakerFee = 0.25
	g.MakerFee = 0
	g.Websocket = false
	g.RESTPollingDelay = 10
	g.RequestCurrencyPairFormat.Delimiter = "-"
	g.RequestCurrencyPairFormat.Uppercase = true
	g.ConfigCurrencyPairFormat.Delimiter = ""
	g.ConfigCurrencyPairFormat.Uppercase = true
	g.AssetTypes = []string{ticker.Spot}
	g.APIUrl = gdaxAPIURL
}

// Setup initialises the exchange parameters with the current configuration
func (g *GDAX) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		g.SetEnabled(false)
	} else {
		g.Enabled = true
		g.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		g.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, true)
		g.RESTPollingDelay = exch.RESTPollingDelay
		g.Verbose = exch.Verbose
		g.Websocket = exch.Websocket
		g.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		g.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		g.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		if exch.UseSandbox {
			g.APIUrl = gdaxSandboxAPIURL
		}
		err := g.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = g.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns the current fee for the exchange
func (g *GDAX) GetFee(maker bool) float64 {
	if maker {
		return g.MakerFee
	}
	return g.TakerFee
}

// GetProducts returns supported currency pairs on the exchange with specific
// information about the pair
func (g *GDAX) GetProducts() ([]Product, error) {
	products := []Product{}

	return products,
		common.SendHTTPGetRequest(g.APIUrl+gdaxProducts, true, g.Verbose, &products)
}

// GetOrderbook returns orderbook by currency pair and level
func (g *GDAX) GetOrderbook(symbol string, level int) (interface{}, error) {
	orderbook := OrderbookResponse{}

	path := fmt.Sprintf("%s/%s/%s", g.APIUrl+gdaxProducts, symbol, gdaxOrderbook)
	if level > 0 {
		levelStr := strconv.Itoa(level)
		path = fmt.Sprintf("%s/%s/%s?level=%s", g.APIUrl+gdaxProducts, symbol, gdaxOrderbook, levelStr)
	}

	if err := common.SendHTTPGetRequest(path, true, g.Verbose, &orderbook); err != nil {
		return nil, err
	}

	if level == 3 {
		ob := OrderbookL3{}
		ob.Sequence = orderbook.Sequence
		for _, x := range orderbook.Asks {
			price, err := strconv.ParseFloat((x[0].(string)), 64)
			if err != nil {
				continue
			}
			amount, err := strconv.ParseFloat((x[1].(string)), 64)
			if err != nil {
				continue
			}

			ob.Asks = append(ob.Asks, OrderL3{Price: price, Amount: amount, OrderID: x[2].(string)})
		}
		for _, x := range orderbook.Bids {
			price, err := strconv.ParseFloat((x[0].(string)), 64)
			if err != nil {
				continue
			}
			amount, err := strconv.ParseFloat((x[1].(string)), 64)
			if err != nil {
				continue
			}

			ob.Bids = append(ob.Bids, OrderL3{Price: price, Amount: amount, OrderID: x[2].(string)})
		}
		return ob, nil
	}
	ob := OrderbookL1L2{}
	ob.Sequence = orderbook.Sequence
	for _, x := range orderbook.Asks {
		price, err := strconv.ParseFloat((x[0].(string)), 64)
		if err != nil {
			continue
		}
		amount, err := strconv.ParseFloat((x[1].(string)), 64)
		if err != nil {
			continue
		}

		ob.Asks = append(ob.Asks, OrderL1L2{Price: price, Amount: amount, NumOrders: x[2].(float64)})
	}
	for _, x := range orderbook.Bids {
		price, err := strconv.ParseFloat((x[0].(string)), 64)
		if err != nil {
			continue
		}
		amount, err := strconv.ParseFloat((x[1].(string)), 64)
		if err != nil {
			continue
		}

		ob.Bids = append(ob.Bids, OrderL1L2{Price: price, Amount: amount, NumOrders: x[2].(float64)})
	}
	return ob, nil
}

// GetTicker returns ticker by currency pair
// currencyPair - example "BTC-USD"
func (g *GDAX) GetTicker(currencyPair string) (Ticker, error) {
	ticker := Ticker{}
	path := fmt.Sprintf(
		"%s/%s/%s", g.APIUrl+gdaxProducts, currencyPair, gdaxTicker)

	return ticker, common.SendHTTPGetRequest(path, true, g.Verbose, &ticker)
}

// GetTrades listd the latest trades for a product
// currencyPair - example "BTC-USD"
func (g *GDAX) GetTrades(currencyPair string) ([]Trade, error) {
	trades := []Trade{}
	path := fmt.Sprintf(
		"%s/%s/%s", g.APIUrl+gdaxProducts, currencyPair, gdaxTrades)

	return trades, common.SendHTTPGetRequest(path, true, g.Verbose, &trades)
}

// GetHistoricRates returns historic rates for a product. Rates are returned in
// grouped buckets based on requested granularity.
func (g *GDAX) GetHistoricRates(currencyPair string, start, end, granularity int64) ([]History, error) {
	var resp [][]interface{}
	history := []History{}
	values := url.Values{}

	if start > 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if end > 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	if granularity > 0 {
		values.Set("granularity", strconv.FormatInt(granularity, 10))
	}

	path := common.EncodeURLValues(
		fmt.Sprintf("%s/%s/%s", g.APIUrl+gdaxProducts, currencyPair, gdaxHistory),
		values)

	if err := common.SendHTTPGetRequest(path, true, g.Verbose, &resp); err != nil {
		return history, err
	}

	for _, single := range resp {
		s := History{
			Time:   int64(single[0].(float64)),
			Low:    single[1].(float64),
			High:   single[2].(float64),
			Open:   single[3].(float64),
			Close:  single[4].(float64),
			Volume: single[5].(float64),
		}
		history = append(history, s)
	}

	return history, nil
}

// GetStats returns a 24 hr stat for the product. Volume is in base currency
// units. open, high, low are in quote currency units.
func (g *GDAX) GetStats(currencyPair string) (Stats, error) {
	stats := Stats{}
	path := fmt.Sprintf(
		"%s/%s/%s", g.APIUrl+gdaxProducts, currencyPair, gdaxStats)

	return stats, common.SendHTTPGetRequest(path, true, g.Verbose, &stats)
}

// GetCurrencies returns a list of supported currency on the exchange
// Warning: Not all currencies may be currently in use for trading.
func (g *GDAX) GetCurrencies() ([]Currency, error) {
	currencies := []Currency{}

	return currencies,
		common.SendHTTPGetRequest(g.APIUrl+gdaxCurrencies, true, g.Verbose, &currencies)
}

// GetServerTime returns the API server time
func (g *GDAX) GetServerTime() (ServerTime, error) {
	serverTime := ServerTime{}

	return serverTime,
		common.SendHTTPGetRequest(g.APIUrl+gdaxTime, true, g.Verbose, &serverTime)
}

// GetAccounts returns a list of trading accounts associated with the APIKEYS
func (g *GDAX) GetAccounts() ([]AccountResponse, error) {
	resp := []AccountResponse{}

	return resp,
		g.SendAuthenticatedHTTPRequest("GET", gdaxAccounts, nil, &resp)
}

// GetAccount returns information for a single account. Use this endpoint when
// account_id is known
func (g *GDAX) GetAccount(accountID string) (AccountResponse, error) {
	resp := AccountResponse{}
	path := fmt.Sprintf("%s/%s", gdaxAccounts, accountID)

	return resp, g.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetAccountHistory returns a list of account activity. Account activity either
// increases or decreases your account balance. Items are paginated and sorted
// latest first.
func (g *GDAX) GetAccountHistory(accountID string) ([]AccountLedgerResponse, error) {
	resp := []AccountLedgerResponse{}
	path := fmt.Sprintf("%s/%s/%s", gdaxAccounts, accountID, gdaxLedger)

	return resp, g.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetHolds returns the holds that are placed on an account for any active
// orders or pending withdraw requests. As an order is filled, the hold amount
// is updated. If an order is canceled, any remaining hold is removed. For a
// withdraw, once it is completed, the hold is removed.
func (g *GDAX) GetHolds(accountID string) ([]AccountHolds, error) {
	resp := []AccountHolds{}
	path := fmt.Sprintf("%s/%s/%s", gdaxAccounts, accountID, gdaxHolds)

	return resp, g.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
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
func (g *GDAX) PlaceLimitOrder(clientRef string, price, amount float64, side, timeInforce, cancelAfter, productID, stp string, postOnly bool) (string, error) {
	resp := GeneralizedOrderResponse{}
	request := make(map[string]interface{})
	request["type"] = "limit"
	request["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	request["size"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["side"] = side
	request["product_id"] = productID

	if cancelAfter != "" {
		request["cancel_after"] = cancelAfter
	}
	if timeInforce != "" {
		request["time_in_foce"] = timeInforce
	}
	if clientRef != "" {
		request["client_oid"] = clientRef
	}
	if stp != "" {
		request["stp"] = stp
	}
	if postOnly {
		request["post_only"] = postOnly
	}

	err := g.SendAuthenticatedHTTPRequest("POST", gdaxOrders, request, &resp)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
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
func (g *GDAX) PlaceMarketOrder(clientRef string, size, funds float64, side string, productID, stp string) (string, error) {
	resp := GeneralizedOrderResponse{}
	request := make(map[string]interface{})
	request["side"] = side
	request["product_id"] = productID
	request["type"] = "market"

	if size != 0 {
		request["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	}
	if funds != 0 {
		request["funds"] = strconv.FormatFloat(funds, 'f', -1, 64)
	}
	if clientRef != "" {
		request["client_oid"] = clientRef
	}
	if stp != "" {
		request["stp"] = stp
	}

	err := g.SendAuthenticatedHTTPRequest("POST", gdaxOrders, request, &resp)
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
func (g *GDAX) PlaceMarginOrder(clientRef string, size, funds float64, side string, productID, stp string) (string, error) {
	resp := GeneralizedOrderResponse{}
	request := make(map[string]interface{})
	request["side"] = side
	request["product_id"] = productID
	request["type"] = "margin"

	if size != 0 {
		request["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	}
	if funds != 0 {
		request["funds"] = strconv.FormatFloat(funds, 'f', -1, 64)
	}
	if clientRef != "" {
		request["client_oid"] = clientRef
	}
	if stp != "" {
		request["stp"] = stp
	}

	err := g.SendAuthenticatedHTTPRequest("POST", gdaxOrders, request, &resp)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// CancelOrder cancels order by orderID
func (g *GDAX) CancelOrder(orderID string) error {
	path := fmt.Sprintf("%s/%s", gdaxOrders, orderID)

	return g.SendAuthenticatedHTTPRequest("DELETE", path, nil, nil)
}

// CancelAllOrders cancels all open orders on the exchange and returns and array
// of order IDs
// currencyPair - [optional] all orders for a currencyPair string will be
// canceled
func (g *GDAX) CancelAllOrders(currencyPair string) ([]string, error) {
	var resp []string
	request := make(map[string]interface{})

	if len(currencyPair) != 0 {
		request["product_id"] = currencyPair
	}
	return resp, g.SendAuthenticatedHTTPRequest("DELETE", gdaxOrders, request, &resp)
}

// GetOrders lists current open orders. Only open or un-settled orders are
// returned. As soon as an order is no longer open and settled, it will no
// longer appear in the default request.
// status - can be a range of "open", "pending", "done" or "active"
// currencyPair - [optional] for example "BTC-USD"
func (g *GDAX) GetOrders(status []string, currencyPair string) ([]GeneralizedOrderResponse, error) {
	resp := []GeneralizedOrderResponse{}
	params := url.Values{}

	for _, individualStatus := range status {
		params.Add("status", individualStatus)
	}
	if len(currencyPair) != 0 {
		params.Set("product_id", currencyPair)
	}

	path := common.EncodeURLValues(g.APIUrl+gdaxOrders, params)
	path = common.GetURIPath(path)

	return resp,
		g.SendAuthenticatedHTTPRequest("GET", path[1:], nil, &resp)
}

// GetOrder returns a single order by order id.
func (g *GDAX) GetOrder(orderID string) (GeneralizedOrderResponse, error) {
	resp := GeneralizedOrderResponse{}
	path := fmt.Sprintf("%s/%s", gdaxOrders, orderID)

	return resp, g.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetFills returns a list of recent fills
func (g *GDAX) GetFills(orderID, currencyPair string) ([]FillResponse, error) {
	resp := []FillResponse{}
	params := url.Values{}

	if len(orderID) != 0 {
		params.Set("order_id", orderID)
	}
	if len(currencyPair) != 0 {
		params.Set("product_id", currencyPair)
	}
	if len(params.Get("order_id")) == 0 && len(params.Get("product_id")) == 0 {
		return resp, errors.New("no parameters set")
	}

	path := common.EncodeURLValues(g.APIUrl+gdaxFills, params)
	uri := common.GetURIPath(path)

	return resp,
		g.SendAuthenticatedHTTPRequest("GET", uri[1:], nil, &resp)
}

// GetFundingRecords every order placed with a margin profile that draws funding
// will create a funding record.
//
// status - "outstanding", "settled", or "rejected"
func (g *GDAX) GetFundingRecords(status string) ([]Funding, error) {
	resp := []Funding{}
	params := url.Values{}
	params.Set("status", status)

	path := common.EncodeURLValues(g.APIUrl+gdaxFunding, params)
	uri := common.GetURIPath(path)

	return resp,
		g.SendAuthenticatedHTTPRequest("GET", uri[1:], nil, &resp)
}

////////////////////////// Not receiving reply from server /////////////////
// RepayFunding repays the older funding records first
//
// amount - amount of currency to repay
// currency - currency, example USD
// func (g *GDAX) RepayFunding(amount, currency string) (Funding, error) {
// 	resp := Funding{}
// 	params := make(map[string]interface{})
// 	params["amount"] = amount
// 	params["currency"] = currency
//
// 	return resp,
// 		g.SendAuthenticatedHTTPRequest("POST", gdaxFundingRepay, params, &resp)
// }

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
func (g *GDAX) MarginTransfer(amount float64, transferType, profileID, currency string) (MarginTransfer, error) {
	resp := MarginTransfer{}
	request := make(map[string]interface{})
	request["type"] = transferType
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["currency"] = currency
	request["margin_profile_id"] = profileID

	return resp,
		g.SendAuthenticatedHTTPRequest("POST", gdaxMarginTransfer, request, &resp)
}

// GetPosition returns an overview of account profile.
func (g *GDAX) GetPosition() (AccountOverview, error) {
	resp := AccountOverview{}

	return resp,
		g.SendAuthenticatedHTTPRequest("GET", gdaxPosition, nil, &resp)
}

// ClosePosition closes a position and allowing you to repay position as well
// repayOnly -  allows the position to be repaid
func (g *GDAX) ClosePosition(repayOnly bool) (AccountOverview, error) {
	resp := AccountOverview{}
	request := make(map[string]interface{})
	request["repay_only"] = repayOnly

	return resp,
		g.SendAuthenticatedHTTPRequest("POST", gdaxPositionClose, request, &resp)
}

// GetPayMethods returns a full list of payment methods
func (g *GDAX) GetPayMethods() ([]PaymentMethod, error) {
	resp := []PaymentMethod{}

	return resp,
		g.SendAuthenticatedHTTPRequest("GET", gdaxPaymentMethod, nil, &resp)
}

// DepositViaPaymentMethod deposits funds from a payment method. See the Payment
// Methods section for retrieving your payment methods.
//
// amount - The amount to deposit
// currency - The type of currency
// paymentID - ID of the payment method
func (g *GDAX) DepositViaPaymentMethod(amount float64, currency, paymentID string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]interface{})
	req["amount"] = amount
	req["currency"] = currency
	req["payment_method_id"] = paymentID

	return resp,
		g.SendAuthenticatedHTTPRequest("POST", gdaxPaymentMethodDeposit, req, &resp)
}

// DepositViaCoinbase deposits funds from a coinbase account. Move funds between
// a Coinbase account and GDAX trading account within daily limits. Moving
// funds between Coinbase and GDAX is instant and free. See the Coinbase
// Accounts section for retrieving your Coinbase accounts.
//
// amount - The amount to deposit
// currency - The type of currency
// accountID - ID of the coinbase account
func (g *GDAX) DepositViaCoinbase(amount float64, currency, accountID string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]interface{})
	req["amount"] = amount
	req["currency"] = currency
	req["coinbase_account_id"] = accountID

	return resp,
		g.SendAuthenticatedHTTPRequest("POST", gdaxDepositCoinbase, req, &resp)
}

// WithdrawViaPaymentMethod withdraws funds to a payment method
//
// amount - The amount to withdraw
// currency - The type of currency
// paymentID - ID of the payment method
func (g *GDAX) WithdrawViaPaymentMethod(amount float64, currency, paymentID string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]interface{})
	req["amount"] = amount
	req["currency"] = currency
	req["payment_method_id"] = paymentID

	return resp,
		g.SendAuthenticatedHTTPRequest("POST", gdaxWithdrawalPaymentMethod, req, &resp)
}

///////////////////////// NO ROUTE FOUND ERROR ////////////////////////////////
// WithdrawViaCoinbase withdraws funds to a coinbase account.
//
// amount - The amount to withdraw
// currency - The type of currency
// accountID - 	ID of the coinbase account
// func (g *GDAX) WithdrawViaCoinbase(amount float64, currency, accountID string) (DepositWithdrawalInfo, error) {
// 	resp := DepositWithdrawalInfo{}
// 	req := make(map[string]interface{})
// 	req["amount"] = amount
// 	req["currency"] = currency
// 	req["coinbase_account_id"] = accountID
//
// 	return resp,
// 		g.SendAuthenticatedHTTPRequest("POST", gdaxWithdrawalCoinbase, req, &resp)
// }

// WithdrawCrypto withdraws funds to a crypto address
//
// amount - The amount to withdraw
// currency - The type of currency
// cryptoAddress - 	A crypto address of the recipient
func (g *GDAX) WithdrawCrypto(amount float64, currency, cryptoAddress string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]interface{})
	req["amount"] = amount
	req["currency"] = currency
	req["crypto_address"] = cryptoAddress

	return resp,
		g.SendAuthenticatedHTTPRequest("POST", gdaxWithdrawalCrypto, req, &resp)
}

// GetCoinbaseAccounts returns a list of coinbase accounts
func (g *GDAX) GetCoinbaseAccounts() ([]CoinbaseAccounts, error) {
	resp := []CoinbaseAccounts{}

	return resp,
		g.SendAuthenticatedHTTPRequest("GET", gdaxCoinbaseAccounts, nil, &resp)
}

// GetReport returns batches of historic information about your account in
// various human and machine readable forms.
//
// reportType - "fills" or "account"
// startDate - Starting date for the report (inclusive)
// endDate - Ending date for the report (inclusive)
// currencyPair - ID of the product to generate a fills report for.
// E.g. BTC-USD. *Required* if type is fills
// accountID - ID of the account to generate an account report for. *Required*
// if type is account
// format - 	pdf or csv (default is pdf)
// email - [optional] Email address to send the report to
func (g *GDAX) GetReport(reportType, startDate, endDate, currencyPair, accountID, format, email string) (Report, error) {
	resp := Report{}
	request := make(map[string]interface{})
	request["type"] = reportType
	request["start_date"] = startDate
	request["end_date"] = endDate
	request["format"] = "pdf"

	if len(currencyPair) != 0 {
		request["product_id"] = currencyPair
	}
	if len(accountID) != 0 {
		request["account_id"] = accountID
	}
	if format == "csv" {
		request["format"] = format
	}
	if len(email) != 0 {
		request["email"] = email
	}

	return resp,
		g.SendAuthenticatedHTTPRequest("POST", gdaxReports, request, &resp)
}

// GetReportStatus once a report request has been accepted for processing, the
// status is available by polling the report resource endpoint.
func (g *GDAX) GetReportStatus(reportID string) (Report, error) {
	resp := Report{}
	path := fmt.Sprintf("%s/%s", gdaxReports, reportID)

	return resp, g.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetTrailingVolume this request will return your 30-day trailing volume for
// all products.
func (g *GDAX) GetTrailingVolume() ([]Volume, error) {
	resp := []Volume{}

	return resp,
		g.SendAuthenticatedHTTPRequest("GET", gdaxTrailingVolume, nil, &resp)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP reque
func (g *GDAX) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	if !g.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, g.Name)
	}

	payload := []byte("")

	if params != nil {
		payload, err = common.JSONEncode(params)
		if err != nil {
			return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
		}

		if g.Verbose {
			log.Printf("Request JSON: %s\n", payload)
		}
	}

	nonce := g.Nonce.GetValue(g.Name, false).String()
	message := nonce + method + "/" + path + string(payload)
	hmac := common.GetHMAC(common.HashSHA256, []byte(message), []byte(g.APISecret))
	headers := make(map[string]string)
	headers["CB-ACCESS-SIGN"] = common.Base64Encode([]byte(hmac))
	headers["CB-ACCESS-TIMESTAMP"] = nonce
	headers["CB-ACCESS-KEY"] = g.APIKey
	headers["CB-ACCESS-PASSPHRASE"] = g.ClientID
	headers["Content-Type"] = "application/json"

	resp, err := common.SendHTTPRequest(method, g.APIUrl+path, headers, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	if g.Verbose {
		log.Printf("Received raw: \n%s\n", resp)
	}

	type initialResponse struct {
		Message string `json:"message"`
	}
	initialCheck := initialResponse{}

	err = common.JSONDecode([]byte(resp), &initialCheck)
	if err == nil && len(initialCheck.Message) != 0 {
		return errors.New(initialCheck.Message)
	}

	return common.JSONDecode([]byte(resp), &result)
}
