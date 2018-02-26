package poloniex

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
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	poloniexAPIURL               = "https://poloniex.com"
	poloniexAPITradingEndpoint   = "tradingApi"
	poloniexAPIVersion           = "1"
	poloniexBalances             = "returnBalances"
	poloniexBalancesComplete     = "returnCompleteBalances"
	poloniexDepositAddresses     = "returnDepositAddresses"
	poloniexGenerateNewAddress   = "generateNewAddress"
	poloniexDepositsWithdrawals  = "returnDepositsWithdrawals"
	poloniexOrders               = "returnOpenOrders"
	poloniexTradeHistory         = "returnTradeHistory"
	poloniexOrderBuy             = "buy"
	poloniexOrderSell            = "sell"
	poloniexOrderCancel          = "cancelOrder"
	poloniexOrderMove            = "moveOrder"
	poloniexWithdraw             = "withdraw"
	poloniexFeeInfo              = "returnFeeInfo"
	poloniexAvailableBalances    = "returnAvailableAccountBalances"
	poloniexTradableBalances     = "returnTradableBalances"
	poloniexTransferBalance      = "transferBalance"
	poloniexMarginAccountSummary = "returnMarginAccountSummary"
	poloniexMarginBuy            = "marginBuy"
	poloniexMarginSell           = "marginSell"
	poloniexMarginPosition       = "getMarginPosition"
	poloniexMarginPositionClose  = "closeMarginPosition"
	poloniexCreateLoanOffer      = "createLoanOffer"
	poloniexCancelLoanOffer      = "cancelLoanOffer"
	poloniexOpenLoanOffers       = "returnOpenLoanOffers"
	poloniexActiveLoans          = "returnActiveLoans"
	poloniexLendingHistory       = "returnLendingHistory"
	poloniexAutoRenew            = "toggleAutoRenew"
)

// Poloniex is the overarching type across the poloniex package
type Poloniex struct {
	exchange.Base
}

// SetDefaults sets default settings for poloniex
func (p *Poloniex) SetDefaults() {
	p.Name = "Poloniex"
	p.Enabled = false
	p.Fee = 0
	p.Verbose = false
	p.Websocket = false
	p.RESTPollingDelay = 10
	p.RequestCurrencyPairFormat.Delimiter = "_"
	p.RequestCurrencyPairFormat.Uppercase = true
	p.ConfigCurrencyPairFormat.Delimiter = "_"
	p.ConfigCurrencyPairFormat.Uppercase = true
	p.AssetTypes = []string{ticker.Spot}
}

// Setup sets user exchange configuration settings
func (p *Poloniex) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		p.SetEnabled(false)
	} else {
		p.Enabled = true
		p.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		p.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		p.RESTPollingDelay = exch.RESTPollingDelay
		p.Verbose = exch.Verbose
		p.Websocket = exch.Websocket
		p.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		p.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		p.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := p.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = p.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns the fee for poloniex
func (p *Poloniex) GetFee() float64 {
	return p.Fee
}

// GetTicker returns current ticker information
func (p *Poloniex) GetTicker() (map[string]Ticker, error) {
	type response struct {
		Data map[string]Ticker
	}

	resp := response{}
	path := fmt.Sprintf("%s/public?command=returnTicker", poloniexAPIURL)

	return resp.Data, common.SendHTTPGetRequest(path, true, p.Verbose, &resp.Data)
}

// GetVolume returns a list of currencies with associated volume
func (p *Poloniex) GetVolume() (interface{}, error) {
	var resp interface{}
	path := fmt.Sprintf("%s/public?command=return24hVolume", poloniexAPIURL)

	return resp, common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
}

// GetOrderbook returns the full orderbook from poloniex
func (p *Poloniex) GetOrderbook(currencyPair string, depth int) (OrderbookAll, error) {
	vals := url.Values{}

	if depth != 0 {
		vals.Set("depth", strconv.Itoa(depth))
	}

	oba := OrderbookAll{Data: make(map[string]Orderbook)}
	if currencyPair != "" {
		vals.Set("currencyPair", currencyPair)
		resp := OrderbookResponse{}
		path := fmt.Sprintf("%s/public?command=returnOrderBook&%s", poloniexAPIURL, vals.Encode())
		err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
		if err != nil {
			return oba, err
		}
		if len(resp.Error) != 0 {
			log.Println(resp.Error)
			return oba, fmt.Errorf("Poloniex GetOrderbook() error: %s", resp.Error)
		}
		ob := Orderbook{}
		for x := range resp.Asks {
			data := resp.Asks[x]
			price, err := strconv.ParseFloat(data[0].(string), 64)
			if err != nil {
				return oba, err
			}
			amount := data[1].(float64)
			ob.Asks = append(ob.Asks, OrderbookItem{Price: price, Amount: amount})
		}

		for x := range resp.Bids {
			data := resp.Bids[x]
			price, err := strconv.ParseFloat(data[0].(string), 64)
			if err != nil {
				return oba, err
			}
			amount := data[1].(float64)
			ob.Bids = append(ob.Bids, OrderbookItem{Price: price, Amount: amount})
		}
		oba.Data[currencyPair] = Orderbook{Bids: ob.Bids, Asks: ob.Asks}
	} else {
		vals.Set("currencyPair", "all")
		resp := OrderbookResponseAll{}
		path := fmt.Sprintf("%s/public?command=returnOrderBook&%s", poloniexAPIURL, vals.Encode())
		err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp.Data)
		if err != nil {
			return oba, err
		}
		for currency, orderbook := range resp.Data {
			ob := Orderbook{}
			for x := range orderbook.Asks {
				data := orderbook.Asks[x]
				price, err := strconv.ParseFloat(data[0].(string), 64)
				if err != nil {
					return oba, err
				}
				amount := data[1].(float64)
				ob.Asks = append(ob.Asks, OrderbookItem{Price: price, Amount: amount})
			}

			for x := range orderbook.Bids {
				data := orderbook.Bids[x]
				price, err := strconv.ParseFloat(data[0].(string), 64)
				if err != nil {
					return oba, err
				}
				amount := data[1].(float64)
				ob.Bids = append(ob.Bids, OrderbookItem{Price: price, Amount: amount})
			}
			oba.Data[currency] = Orderbook{Bids: ob.Bids, Asks: ob.Asks}
		}
	}
	return oba, nil
}

// GetTradeHistory returns trades history from poloniex
func (p *Poloniex) GetTradeHistory(currencyPair, start, end string) ([]TradeHistory, error) {
	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if start != "" {
		vals.Set("start", start)
	}

	if end != "" {
		vals.Set("end", end)
	}

	resp := []TradeHistory{}
	path := fmt.Sprintf("%s/public?command=returnTradeHistory&%s", poloniexAPIURL, vals.Encode())

	return resp, common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
}

// GetChartData returns chart data for a specific currency pair
func (p *Poloniex) GetChartData(currencyPair, start, end, period string) ([]ChartData, error) {
	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if start != "" {
		vals.Set("start", start)
	}

	if end != "" {
		vals.Set("end", end)
	}

	if period != "" {
		vals.Set("period", period)
	}

	resp := []ChartData{}
	path := fmt.Sprintf("%s/public?command=returnChartData&%s", poloniexAPIURL, vals.Encode())

	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetCurrencies returns information about currencies
func (p *Poloniex) GetCurrencies() (map[string]Currencies, error) {
	type Response struct {
		Data map[string]Currencies
	}
	resp := Response{}
	path := fmt.Sprintf("%s/public?command=returnCurrencies", poloniexAPIURL)

	return resp.Data, common.SendHTTPGetRequest(path, true, p.Verbose, &resp.Data)
}

// GetExchangeCurrencies returns a list of currencies using the GetTicker API
// as the GetExchangeCurrencies information doesn't return currency pair information
func (p *Poloniex) GetExchangeCurrencies() ([]string, error) {
	response, err := p.GetTicker()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range response {
		currencies = append(currencies, x)
	}

	return currencies, nil
}

// GetLoanOrders returns the list of loan offers and demands for a given
// currency, specified by the "currency" GET parameter.
func (p *Poloniex) GetLoanOrders(currency string) (LoanOrders, error) {
	resp := LoanOrders{}
	path := fmt.Sprintf("%s/public?command=returnLoanOrders&currency=%s", poloniexAPIURL, currency)

	return resp, common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
}

// GetBalances returns balances for your account.
func (p *Poloniex) GetBalances() (Balance, error) {
	var result interface{}
	err := p.SendAuthenticatedHTTPRequest("POST", poloniexBalances, url.Values{}, &result)

	if err != nil {
		return Balance{}, err
	}

	data := result.(map[string]interface{})
	balance := Balance{}
	balance.Currency = make(map[string]float64)

	for x, y := range data {
		balance.Currency[x], _ = strconv.ParseFloat(y.(string), 64)
	}

	return balance, nil
}

// GetCompleteBalances returns complete balances from your account.
func (p *Poloniex) GetCompleteBalances() (CompleteBalances, error) {
	var result interface{}
	err := p.SendAuthenticatedHTTPRequest("POST", poloniexBalancesComplete, url.Values{}, &result)

	if err != nil {
		return CompleteBalances{}, err
	}

	data := result.(map[string]interface{})
	balance := CompleteBalances{}
	balance.Currency = make(map[string]CompleteBalance)

	for x, y := range data {
		dataVals := y.(map[string]interface{})
		balancesData := CompleteBalance{}
		balancesData.Available, _ = strconv.ParseFloat(dataVals["available"].(string), 64)
		balancesData.OnOrders, _ = strconv.ParseFloat(dataVals["onOrders"].(string), 64)
		balancesData.BTCValue, _ = strconv.ParseFloat(dataVals["btcValue"].(string), 64)
		balance.Currency[x] = balancesData
	}

	return balance, nil
}

// GetDepositAddresses returns deposit addresses for all enabled cryptos.
func (p *Poloniex) GetDepositAddresses() (DepositAddresses, error) {
	var result interface{}
	addresses := DepositAddresses{}
	err := p.SendAuthenticatedHTTPRequest("POST", poloniexDepositAddresses, url.Values{}, &result)

	if err != nil {
		return addresses, err
	}

	addresses.Addresses = make(map[string]string)
	data := result.(map[string]interface{})
	for x, y := range data {
		addresses.Addresses[x] = y.(string)
	}

	return addresses, nil
}

// GenerateNewAddress generates a new address for a currency
func (p *Poloniex) GenerateNewAddress(currency string) (string, error) {
	type Response struct {
		Success  int
		Error    string
		Response string
	}
	resp := Response{}
	values := url.Values{}
	values.Set("currency", currency)

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexGenerateNewAddress, values, &resp)

	if err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}

	return resp.Response, nil
}

// GetDepositsWithdrawals returns a list of deposits and withdrawals
func (p *Poloniex) GetDepositsWithdrawals(start, end string) (DepositsWithdrawals, error) {
	resp := DepositsWithdrawals{}
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	} else {
		values.Set("start", "0")
	}

	if end != "" {
		values.Set("end", end)
	} else {
		values.Set("end", strconv.FormatInt(time.Now().Unix(), 10))
	}

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexDepositsWithdrawals, values, &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

// GetOpenOrders returns current unfilled opened orders
func (p *Poloniex) GetOpenOrders(currency string) (interface{}, error) {
	values := url.Values{}

	if currency != "" {
		values.Set("currencyPair", currency)
		result := OpenOrdersResponse{}

		err := p.SendAuthenticatedHTTPRequest("POST", poloniexOrders, values, &result.Data)
		if err != nil {
			return result, err
		}

		return result, nil
	}
	values.Set("currencyPair", "all")
	result := OpenOrdersResponseAll{}

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexOrders, values, &result.Data)
	if err != nil {
		return result, err
	}

	return result, nil
}

// GetAuthenticatedTradeHistory returns account trade history
func (p *Poloniex) GetAuthenticatedTradeHistory(currency, start, end, limit string) (interface{}, error) {
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	}

	if limit != "" {
		values.Set("limit", limit)
	}

	if end != "" {
		values.Set("end", end)
	}

	if currency != "" && currency != "all" {
		values.Set("currencyPair", currency)
		result := AuthenticatedTradeHistoryResponse{}

		err := p.SendAuthenticatedHTTPRequest("POST", poloniexTradeHistory, values, &result.Data)
		if err != nil {
			return result, err
		}

		return result, nil
	}
	values.Set("currencyPair", "all")
	result := AuthenticatedTradeHistoryAll{}

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexTradeHistory, values, &result.Data)
	if err != nil {
		return result, err
	}

	return result, nil
}

// PlaceOrder places a new order on the exchange
func (p *Poloniex) PlaceOrder(currency string, rate, amount float64, immediate, fillOrKill, buy bool) (OrderResponse, error) {
	result := OrderResponse{}
	values := url.Values{}

	var orderType string
	if buy {
		orderType = poloniexOrderBuy
	} else {
		orderType = poloniexOrderSell
	}

	values.Set("currencyPair", currency)
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	if immediate {
		values.Set("immediateOrCancel", "1")
	}

	if fillOrKill {
		values.Set("fillOrKill", "1")
	}

	err := p.SendAuthenticatedHTTPRequest("POST", orderType, values, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

// CancelOrder cancels and order by orderID
func (p *Poloniex) CancelOrder(orderID int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexOrderCancel, values, &result)

	if err != nil {
		return false, err
	}

	if result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// MoveOrder moves an order
func (p *Poloniex) MoveOrder(orderID int64, rate, amount float64) (MoveOrderResponse, error) {
	result := MoveOrderResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))

	if amount != 0 {
		values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexOrderMove, values, &result)

	if err != nil {
		return result, err
	}

	if result.Success != 1 {
		return result, errors.New(result.Error)
	}

	return result, nil
}

// Withdraw withdraws a currency to a specific delegated address
func (p *Poloniex) Withdraw(currency, address string, amount float64) (bool, error) {
	result := Withdraw{}
	values := url.Values{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("address", address)

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexWithdraw, values, &result)

	if err != nil {
		return false, err
	}

	if result.Error != "" {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetFeeInfo returns fee information
func (p *Poloniex) GetFeeInfo() (Fee, error) {
	result := Fee{}

	return result, p.SendAuthenticatedHTTPRequest("POST", poloniexFeeInfo, url.Values{}, &result)
}

// GetTradableBalances returns tradable balances
func (p *Poloniex) GetTradableBalances() (map[string]map[string]float64, error) {
	type Response struct {
		Data map[string]map[string]interface{}
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexTradableBalances, url.Values{}, &result.Data)

	if err != nil {
		return nil, err
	}

	balances := make(map[string]map[string]float64)

	for x, y := range result.Data {
		balances[x] = make(map[string]float64)
		for z, w := range y {
			balances[x][z], _ = strconv.ParseFloat(w.(string), 64)
		}
	}

	return balances, nil
}

// TransferBalance transfers balances between your accounts
func (p *Poloniex) TransferBalance(currency, from, to string, amount float64) (bool, error) {
	values := url.Values{}
	result := GenericResponse{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("fromAccount", from)
	values.Set("toAccount", to)

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexTransferBalance, values, &result)

	if err != nil {
		return false, err
	}

	if result.Error != "" && result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetMarginAccountSummary returns a summary on your margin accounts
func (p *Poloniex) GetMarginAccountSummary() (Margin, error) {
	result := Margin{}
	err := p.SendAuthenticatedHTTPRequest("POST", poloniexMarginAccountSummary, url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

// PlaceMarginOrder places a margin order
func (p *Poloniex) PlaceMarginOrder(currency string, rate, amount, lendingRate float64, buy bool) (OrderResponse, error) {
	result := OrderResponse{}
	values := url.Values{}

	var orderType string
	if buy {
		orderType = poloniexMarginBuy
	} else {
		orderType = poloniexMarginSell
	}

	values.Set("currencyPair", currency)
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	if lendingRate != 0 {
		values.Set("lendingRate", strconv.FormatFloat(lendingRate, 'f', -1, 64))
	}

	err := p.SendAuthenticatedHTTPRequest("POST", orderType, values, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

// GetMarginPosition returns a position on a margin order
func (p *Poloniex) GetMarginPosition(currency string) (interface{}, error) {
	values := url.Values{}

	if currency != "" && currency != "all" {
		values.Set("currencyPair", currency)
		result := MarginPosition{}

		err := p.SendAuthenticatedHTTPRequest("POST", poloniexMarginPosition, values, &result)
		if err != nil {
			return result, err
		}

		return result, nil
	}
	values.Set("currencyPair", "all")

	type Response struct {
		Data map[string]MarginPosition
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexMarginPosition, values, &result.Data)
	if err != nil {
		return result, err
	}

	return result, nil
}

// CloseMarginPosition closes a current margin position
func (p *Poloniex) CloseMarginPosition(currency string) (bool, error) {
	values := url.Values{}
	values.Set("currencyPair", currency)
	result := GenericResponse{}

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexMarginPositionClose, values, &result)

	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// CreateLoanOffer places a loan offer on the exchange
func (p *Poloniex) CreateLoanOffer(currency string, amount, rate float64, duration int, autoRenew bool) (int64, error) {
	values := url.Values{}
	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("duration", strconv.Itoa(duration))

	if autoRenew {
		values.Set("autoRenew", "1")
	} else {
		values.Set("autoRenew", "0")
	}

	values.Set("lendingRate", strconv.FormatFloat(rate, 'f', -1, 64))

	type Response struct {
		Success int    `json:"success"`
		Error   string `json:"error"`
		OrderID int64  `json:"orderID"`
	}

	result := Response{}

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexCreateLoanOffer, values, &result)

	if err != nil {
		return 0, err
	}

	if result.Success == 0 {
		return 0, errors.New(result.Error)
	}

	return result.OrderID, nil
}

// CancelLoanOffer cancels a loan offer order
func (p *Poloniex) CancelLoanOffer(orderNumber int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}
	values.Set("orderID", strconv.FormatInt(orderNumber, 10))

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexCancelLoanOffer, values, &result)

	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetOpenLoanOffers returns all open loan offers
func (p *Poloniex) GetOpenLoanOffers() (map[string][]LoanOffer, error) {
	type Response struct {
		Data map[string][]LoanOffer
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexOpenLoanOffers, url.Values{}, &result.Data)

	if err != nil {
		return nil, err
	}

	if result.Data == nil {
		return nil, errors.New("there are no open loan offers")
	}

	return result.Data, nil
}

// GetActiveLoans returns active loans
func (p *Poloniex) GetActiveLoans() (ActiveLoans, error) {
	result := ActiveLoans{}
	err := p.SendAuthenticatedHTTPRequest("POST", poloniexActiveLoans, url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

// GetLendingHistory returns lending history for the account
func (p *Poloniex) GetLendingHistory(start, end string) ([]LendingHistory, error) {
	vals := url.Values{}

	if start != "" {
		vals.Set("start", start)
	}

	if end != "" {
		vals.Set("end", end)
	}

	resp := []LendingHistory{}
	err := p.SendAuthenticatedHTTPRequest("POST", poloniexLendingHistory, vals, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ToggleAutoRenew allows for the autorenew of a contract
func (p *Poloniex) ToggleAutoRenew(orderNumber int64) (bool, error) {
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderNumber, 10))
	result := GenericResponse{}

	err := p.SendAuthenticatedHTTPRequest("POST", poloniexAutoRenew, values, &result)

	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (p *Poloniex) SendAuthenticatedHTTPRequest(method, endpoint string, values url.Values, result interface{}) error {
	if !p.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, p.Name)
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	headers["Key"] = p.APIKey

	if p.Nonce.Get() == 0 {
		p.Nonce.Set(time.Now().UnixNano())
	} else {
		p.Nonce.Inc()
	}
	values.Set("nonce", p.Nonce.String())
	values.Set("command", endpoint)

	hmac := common.GetHMAC(common.HashSHA512, []byte(values.Encode()), []byte(p.APISecret))
	headers["Sign"] = common.HexEncodeToString(hmac)

	path := fmt.Sprintf("%s/%s", poloniexAPIURL, poloniexAPITradingEndpoint)

	resp, err := common.SendHTTPRequest(method, path, headers, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}

	return common.JSONDecode([]byte(resp), &result)
}
