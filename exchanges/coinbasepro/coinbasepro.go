package coinbasepro

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	coinbaseproAPIURL                  = "https://api.pro.coinbase.com/"
	coinbaseproSandboxAPIURL           = "https://public.sandbox.pro.coinbase.com"
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
	coinbaseproFunding                 = "funding"
	coinbaseproFundingRepay            = "funding/repay"
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

	coinbaseproAuthRate   = 5
	coinbaseproUnauthRate = 3
)

// CoinbasePro is the overarching type across the coinbasepro package
type CoinbasePro struct {
	exchange.Base
}

// SetDefaults sets default values for the exchange
func (c *CoinbasePro) SetDefaults() {
	c.Name = "CoinbasePro"
	c.Enabled = false
	c.Verbose = false
	c.TakerFee = 0.25
	c.MakerFee = 0
	c.Websocket = false
	c.RESTPollingDelay = 10
	c.RequestCurrencyPairFormat.Delimiter = "-"
	c.RequestCurrencyPairFormat.Uppercase = true
	c.ConfigCurrencyPairFormat.Delimiter = ""
	c.ConfigCurrencyPairFormat.Uppercase = true
	c.AssetTypes = []string{ticker.Spot}
	c.APIUrl = coinbaseproAPIURL
	c.SupportsAutoPairUpdating = true
	c.SupportsRESTTickerBatching = false
	c.Requester = request.New(c.Name, request.NewRateLimit(time.Second, coinbaseproAuthRate), request.NewRateLimit(time.Second, coinbaseproUnauthRate), common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
}

// Setup initialises the exchange parameters with the current configuration
func (c *CoinbasePro) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		c.SetEnabled(false)
	} else {
		c.Enabled = true
		c.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		c.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, true)
		c.SetHTTPClientTimeout(exch.HTTPTimeout)
		c.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		c.RESTPollingDelay = exch.RESTPollingDelay
		c.Verbose = exch.Verbose
		c.Websocket = exch.Websocket
		c.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		c.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		c.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		if exch.UseSandbox {
			c.APIUrl = coinbaseproSandboxAPIURL
		}
		err := c.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = c.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = c.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns the current fee for the exchange
func (c *CoinbasePro) GetFee(maker bool) float64 {
	if maker {
		return c.MakerFee
	}
	return c.TakerFee
}

// GetProducts returns supported currency pairs on the exchange with specific
// information about the pair
func (c *CoinbasePro) GetProducts() ([]Product, error) {
	products := []Product{}

	return products, c.SendHTTPRequest(c.APIUrl+coinbaseproProducts, &products)
}

// GetOrderbook returns orderbook by currency pair and level
func (c *CoinbasePro) GetOrderbook(symbol string, level int) (interface{}, error) {
	orderbook := OrderbookResponse{}

	path := fmt.Sprintf("%s/%s/%s", c.APIUrl+coinbaseproProducts, symbol, coinbaseproOrderbook)
	if level > 0 {
		levelStr := strconv.Itoa(level)
		path = fmt.Sprintf("%s/%s/%s?level=%s", c.APIUrl+coinbaseproProducts, symbol, coinbaseproOrderbook, levelStr)
	}

	if err := c.SendHTTPRequest(path, &orderbook); err != nil {
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
func (c *CoinbasePro) GetTicker(currencyPair string) (Ticker, error) {
	ticker := Ticker{}
	path := fmt.Sprintf(
		"%s/%s/%s", c.APIUrl+coinbaseproProducts, currencyPair, coinbaseproTicker)

	return ticker, c.SendHTTPRequest(path, &ticker)
}

// GetTrades listd the latest trades for a product
// currencyPair - example "BTC-USD"
func (c *CoinbasePro) GetTrades(currencyPair string) ([]Trade, error) {
	trades := []Trade{}
	path := fmt.Sprintf(
		"%s/%s/%s", c.APIUrl+coinbaseproProducts, currencyPair, coinbaseproTrades)

	return trades, c.SendHTTPRequest(path, &trades)
}

// GetHistoricRates returns historic rates for a product. Rates are returned in
// grouped buckets based on requested granularity.
func (c *CoinbasePro) GetHistoricRates(currencyPair string, start, end, granularity int64) ([]History, error) {
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
		fmt.Sprintf("%s/%s/%s", c.APIUrl+coinbaseproProducts, currencyPair, coinbaseproHistory),
		values)

	if err := c.SendHTTPRequest(path, &resp); err != nil {
		return history, err
	}

	for _, single := range resp {
		var s History
		a, _ := single[0].(float64)
		s.Time = int64(a)
		b, _ := single[1].(float64)
		s.Low = b
		c, _ := single[2].(float64)
		s.High = c
		d, _ := single[3].(float64)
		s.Open = d
		e, _ := single[4].(float64)
		s.Close = e
		f, _ := single[5].(float64)
		s.Volume = f
		history = append(history, s)
	}

	return history, nil
}

// GetStats returns a 24 hr stat for the product. Volume is in base currency
// units. open, high, low are in quote currency units.
func (c *CoinbasePro) GetStats(currencyPair string) (Stats, error) {
	stats := Stats{}
	path := fmt.Sprintf(
		"%s/%s/%s", c.APIUrl+coinbaseproProducts, currencyPair, coinbaseproStats)

	return stats, c.SendHTTPRequest(path, &stats)
}

// GetCurrencies returns a list of supported currency on the exchange
// Warning: Not all currencies may be currently in use for tradinc.
func (c *CoinbasePro) GetCurrencies() ([]Currency, error) {
	currencies := []Currency{}

	return currencies, c.SendHTTPRequest(c.APIUrl+coinbaseproCurrencies, &currencies)
}

// GetServerTime returns the API server time
func (c *CoinbasePro) GetServerTime() (ServerTime, error) {
	serverTime := ServerTime{}

	return serverTime, c.SendHTTPRequest(c.APIUrl+coinbaseproTime, &serverTime)
}

// GetAccounts returns a list of trading accounts associated with the APIKEYS
func (c *CoinbasePro) GetAccounts() ([]AccountResponse, error) {
	resp := []AccountResponse{}

	return resp,
		c.SendAuthenticatedHTTPRequest("GET", coinbaseproAccounts, nil, &resp)
}

// GetAccount returns information for a single account. Use this endpoint when
// account_id is known
func (c *CoinbasePro) GetAccount(accountID string) (AccountResponse, error) {
	resp := AccountResponse{}
	path := fmt.Sprintf("%s/%s", coinbaseproAccounts, accountID)

	return resp, c.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetAccountHistory returns a list of account activity. Account activity either
// increases or decreases your account balance. Items are paginated and sorted
// latest first.
func (c *CoinbasePro) GetAccountHistory(accountID string) ([]AccountLedgerResponse, error) {
	resp := []AccountLedgerResponse{}
	path := fmt.Sprintf("%s/%s/%s", coinbaseproAccounts, accountID, coinbaseproLedger)

	return resp, c.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetHolds returns the holds that are placed on an account for any active
// orders or pending withdraw requests. As an order is filled, the hold amount
// is updated. If an order is canceled, any remaining hold is removed. For a
// withdraw, once it is completed, the hold is removed.
func (c *CoinbasePro) GetHolds(accountID string) ([]AccountHolds, error) {
	resp := []AccountHolds{}
	path := fmt.Sprintf("%s/%s/%s", coinbaseproAccounts, accountID, coinbaseproHolds)

	return resp, c.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
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
func (c *CoinbasePro) PlaceLimitOrder(clientRef string, price, amount float64, side, timeInforce, cancelAfter, productID, stp string, postOnly bool) (string, error) {
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

	err := c.SendAuthenticatedHTTPRequest("POST", coinbaseproOrders, request, &resp)
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
func (c *CoinbasePro) PlaceMarketOrder(clientRef string, size, funds float64, side string, productID, stp string) (string, error) {
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

	err := c.SendAuthenticatedHTTPRequest("POST", coinbaseproOrders, request, &resp)
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
func (c *CoinbasePro) PlaceMarginOrder(clientRef string, size, funds float64, side string, productID, stp string) (string, error) {
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

	err := c.SendAuthenticatedHTTPRequest("POST", coinbaseproOrders, request, &resp)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// CancelOrder cancels order by orderID
func (c *CoinbasePro) CancelOrder(orderID string) error {
	path := fmt.Sprintf("%s/%s", coinbaseproOrders, orderID)

	return c.SendAuthenticatedHTTPRequest("DELETE", path, nil, nil)
}

// CancelAllOrders cancels all open orders on the exchange and returns and array
// of order IDs
// currencyPair - [optional] all orders for a currencyPair string will be
// canceled
func (c *CoinbasePro) CancelAllOrders(currencyPair string) ([]string, error) {
	var resp []string
	request := make(map[string]interface{})

	if len(currencyPair) != 0 {
		request["product_id"] = currencyPair
	}
	return resp, c.SendAuthenticatedHTTPRequest("DELETE", coinbaseproOrders, request, &resp)
}

// GetOrders lists current open orders. Only open or un-settled orders are
// returned. As soon as an order is no longer open and settled, it will no
// longer appear in the default request.
// status - can be a range of "open", "pending", "done" or "active"
// currencyPair - [optional] for example "BTC-USD"
func (c *CoinbasePro) GetOrders(status []string, currencyPair string) ([]GeneralizedOrderResponse, error) {
	resp := []GeneralizedOrderResponse{}
	params := url.Values{}

	for _, individualStatus := range status {
		params.Add("status", individualStatus)
	}
	if len(currencyPair) != 0 {
		params.Set("product_id", currencyPair)
	}

	path := common.EncodeURLValues(c.APIUrl+coinbaseproOrders, params)
	path = common.GetURIPath(path)

	return resp,
		c.SendAuthenticatedHTTPRequest("GET", path[1:], nil, &resp)
}

// GetOrder returns a single order by order id.
func (c *CoinbasePro) GetOrder(orderID string) (GeneralizedOrderResponse, error) {
	resp := GeneralizedOrderResponse{}
	path := fmt.Sprintf("%s/%s", coinbaseproOrders, orderID)

	return resp, c.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetFills returns a list of recent fills
func (c *CoinbasePro) GetFills(orderID, currencyPair string) ([]FillResponse, error) {
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

	path := common.EncodeURLValues(c.APIUrl+coinbaseproFills, params)
	uri := common.GetURIPath(path)

	return resp,
		c.SendAuthenticatedHTTPRequest("GET", uri[1:], nil, &resp)
}

// GetFundingRecords every order placed with a margin profile that draws funding
// will create a funding record.
//
// status - "outstanding", "settled", or "rejected"
func (c *CoinbasePro) GetFundingRecords(status string) ([]Funding, error) {
	resp := []Funding{}
	params := url.Values{}
	params.Set("status", status)

	path := common.EncodeURLValues(c.APIUrl+coinbaseproFunding, params)
	uri := common.GetURIPath(path)

	return resp,
		c.SendAuthenticatedHTTPRequest("GET", uri[1:], nil, &resp)
}

////////////////////////// Not receiving reply from server /////////////////
// RepayFunding repays the older funding records first
//
// amount - amount of currency to repay
// currency - currency, example USD
// func (c *CoinbasePro) RepayFunding(amount, currency string) (Funding, error) {
// 	resp := Funding{}
// 	params := make(map[string]interface{})
// 	params["amount"] = amount
// 	params["currency"] = currency
//
// 	return resp,
// 		c.SendAuthenticatedHTTPRequest("POST", coinbaseproFundingRepay, params, &resp)
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
func (c *CoinbasePro) MarginTransfer(amount float64, transferType, profileID, currency string) (MarginTransfer, error) {
	resp := MarginTransfer{}
	request := make(map[string]interface{})
	request["type"] = transferType
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["currency"] = currency
	request["margin_profile_id"] = profileID

	return resp,
		c.SendAuthenticatedHTTPRequest("POST", coinbaseproMarginTransfer, request, &resp)
}

// GetPosition returns an overview of account profile.
func (c *CoinbasePro) GetPosition() (AccountOverview, error) {
	resp := AccountOverview{}

	return resp,
		c.SendAuthenticatedHTTPRequest("GET", coinbaseproPosition, nil, &resp)
}

// ClosePosition closes a position and allowing you to repay position as well
// repayOnly -  allows the position to be repaid
func (c *CoinbasePro) ClosePosition(repayOnly bool) (AccountOverview, error) {
	resp := AccountOverview{}
	request := make(map[string]interface{})
	request["repay_only"] = repayOnly

	return resp,
		c.SendAuthenticatedHTTPRequest("POST", coinbaseproPositionClose, request, &resp)
}

// GetPayMethods returns a full list of payment methods
func (c *CoinbasePro) GetPayMethods() ([]PaymentMethod, error) {
	resp := []PaymentMethod{}

	return resp,
		c.SendAuthenticatedHTTPRequest("GET", coinbaseproPaymentMethod, nil, &resp)
}

// DepositViaPaymentMethod deposits funds from a payment method. See the Payment
// Methods section for retrieving your payment methods.
//
// amount - The amount to deposit
// currency - The type of currency
// paymentID - ID of the payment method
func (c *CoinbasePro) DepositViaPaymentMethod(amount float64, currency, paymentID string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]interface{})
	req["amount"] = amount
	req["currency"] = currency
	req["payment_method_id"] = paymentID

	return resp,
		c.SendAuthenticatedHTTPRequest("POST", coinbaseproPaymentMethodDeposit, req, &resp)
}

// DepositViaCoinbase deposits funds from a coinbase account. Move funds between
// a Coinbase account and coinbasepro trading account within daily limits. Moving
// funds between Coinbase and coinbasepro is instant and free. See the Coinbase
// Accounts section for retrieving your Coinbase accounts.
//
// amount - The amount to deposit
// currency - The type of currency
// accountID - ID of the coinbase account
func (c *CoinbasePro) DepositViaCoinbase(amount float64, currency, accountID string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]interface{})
	req["amount"] = amount
	req["currency"] = currency
	req["coinbase_account_id"] = accountID

	return resp,
		c.SendAuthenticatedHTTPRequest("POST", coinbaseproDepositCoinbase, req, &resp)
}

// WithdrawViaPaymentMethod withdraws funds to a payment method
//
// amount - The amount to withdraw
// currency - The type of currency
// paymentID - ID of the payment method
func (c *CoinbasePro) WithdrawViaPaymentMethod(amount float64, currency, paymentID string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]interface{})
	req["amount"] = amount
	req["currency"] = currency
	req["payment_method_id"] = paymentID

	return resp,
		c.SendAuthenticatedHTTPRequest("POST", coinbaseproWithdrawalPaymentMethod, req, &resp)
}

///////////////////////// NO ROUTE FOUND ERROR ////////////////////////////////
// WithdrawViaCoinbase withdraws funds to a coinbase account.
//
// amount - The amount to withdraw
// currency - The type of currency
// accountID - 	ID of the coinbase account
// func (c *CoinbasePro) WithdrawViaCoinbase(amount float64, currency, accountID string) (DepositWithdrawalInfo, error) {
// 	resp := DepositWithdrawalInfo{}
// 	req := make(map[string]interface{})
// 	req["amount"] = amount
// 	req["currency"] = currency
// 	req["coinbase_account_id"] = accountID
//
// 	return resp,
// 		c.SendAuthenticatedHTTPRequest("POST", coinbaseproWithdrawalCoinbase, req, &resp)
// }

// WithdrawCrypto withdraws funds to a crypto address
//
// amount - The amount to withdraw
// currency - The type of currency
// cryptoAddress - 	A crypto address of the recipient
func (c *CoinbasePro) WithdrawCrypto(amount float64, currency, cryptoAddress string) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := make(map[string]interface{})
	req["amount"] = amount
	req["currency"] = currency
	req["crypto_address"] = cryptoAddress

	return resp,
		c.SendAuthenticatedHTTPRequest("POST", coinbaseproWithdrawalCrypto, req, &resp)
}

// GetCoinbaseAccounts returns a list of coinbase accounts
func (c *CoinbasePro) GetCoinbaseAccounts() ([]CoinbaseAccounts, error) {
	resp := []CoinbaseAccounts{}

	return resp,
		c.SendAuthenticatedHTTPRequest("GET", coinbaseproCoinbaseAccounts, nil, &resp)
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
func (c *CoinbasePro) GetReport(reportType, startDate, endDate, currencyPair, accountID, format, email string) (Report, error) {
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
		c.SendAuthenticatedHTTPRequest("POST", coinbaseproReports, request, &resp)
}

// GetReportStatus once a report request has been accepted for processing, the
// status is available by polling the report resource endpoint.
func (c *CoinbasePro) GetReportStatus(reportID string) (Report, error) {
	resp := Report{}
	path := fmt.Sprintf("%s/%s", coinbaseproReports, reportID)

	return resp, c.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetTrailingVolume this request will return your 30-day trailing volume for
// all products.
func (c *CoinbasePro) GetTrailingVolume() ([]Volume, error) {
	resp := []Volume{}

	return resp,
		c.SendAuthenticatedHTTPRequest("GET", coinbaseproTrailingVolume, nil, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (c *CoinbasePro) SendHTTPRequest(path string, result interface{}) error {
	return c.SendPayload("GET", path, nil, nil, result, false, c.Verbose)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP reque
func (c *CoinbasePro) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	if !c.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, c.Name)
	}

	payload := []byte("")

	if params != nil {
		payload, err = common.JSONEncode(params)
		if err != nil {
			return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
		}

		if c.Verbose {
			log.Printf("Request JSON: %s\n", payload)
		}
	}

	nonce := c.Nonce.GetValue(c.Name, false).String()
	message := nonce + method + "/" + path + string(payload)
	hmac := common.GetHMAC(common.HashSHA256, []byte(message), []byte(c.APISecret))
	headers := make(map[string]string)
	headers["CB-ACCESS-SIGN"] = common.Base64Encode([]byte(hmac))
	headers["CB-ACCESS-TIMESTAMP"] = nonce
	headers["CB-ACCESS-KEY"] = c.APIKey
	headers["CB-ACCESS-PASSPHRASE"] = c.ClientID
	headers["Content-Type"] = "application/json"

	return c.SendPayload(method, c.APIUrl+path, headers, bytes.NewBuffer(payload), result, true, c.Verbose)
}
