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
	POLONIEX_API_URL                = "https://poloniex.com"
	POLONIEX_API_TRADING_ENDPOINT   = "tradingApi"
	POLONIEX_API_VERSION            = "1"
	POLONIEX_BALANCES               = "returnBalances"
	POLONIEX_BALANCES_COMPLETE      = "returnCompleteBalances"
	POLONIEX_DEPOSIT_ADDRESSES      = "returnDepositAddresses"
	POLONIEX_GENERATE_NEW_ADDRESS   = "generateNewAddress"
	POLONIEX_DEPOSITS_WITHDRAWALS   = "returnDepositsWithdrawals"
	POLONIEX_ORDERS                 = "returnOpenOrders"
	POLONIEX_TRADE_HISTORY          = "returnTradeHistory"
	POLONIEX_ORDER_BUY              = "buy"
	POLONIEX_ORDER_SELL             = "sell"
	POLONIEX_ORDER_CANCEL           = "cancelOrder"
	POLONIEX_ORDER_MOVE             = "moveOrder"
	POLONIEX_WITHDRAW               = "withdraw"
	POLONIEX_FEE_INFO               = "returnFeeInfo"
	POLONIEX_AVAILABLE_BALANCES     = "returnAvailableAccountBalances"
	POLONIEX_TRADABLE_BALANCES      = "returnTradableBalances"
	POLONIEX_TRANSFER_BALANCE       = "transferBalance"
	POLONIEX_MARGIN_ACCOUNT_SUMMARY = "returnMarginAccountSummary"
	POLONIEX_MARGIN_BUY             = "marginBuy"
	POLONIEX_MARGIN_SELL            = "marginSell"
	POLONIEX_MARGIN_POSITION        = "getMarginPosition"
	POLONIEX_MARGIN_POSITION_CLOSE  = "closeMarginPosition"
	POLONIEX_CREATE_LOAN_OFFER      = "createLoanOffer"
	POLONIEX_CANCEL_LOAN_OFFER      = "cancelLoanOffer"
	POLONIEX_OPEN_LOAN_OFFERS       = "returnOpenLoanOffers"
	POLONIEX_ACTIVE_LOANS           = "returnActiveLoans"
	POLONIEX_LENDING_HISTORY        = "returnLendingHistory"
	POLONIEX_AUTO_RENEW             = "toggleAutoRenew"
)

type Poloniex struct {
	exchange.Base
}

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

func (p *Poloniex) GetFee() float64 {
	return p.Fee
}

func (p *Poloniex) GetTicker() (map[string]PoloniexTicker, error) {
	type response struct {
		Data map[string]PoloniexTicker
	}

	resp := response{}
	path := fmt.Sprintf("%s/public?command=returnTicker", POLONIEX_API_URL)
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp.Data)

	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

func (p *Poloniex) GetVolume() (interface{}, error) {
	var resp interface{}
	path := fmt.Sprintf("%s/public?command=return24hVolume", POLONIEX_API_URL)
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)

	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (p *Poloniex) GetOrderbook(currencyPair string, depth int) (PoloniexOrderbook, error) {
	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if depth != 0 {
		vals.Set("depth", strconv.Itoa(depth))
	}

	resp := PoloniexOrderbookResponse{}
	path := fmt.Sprintf("%s/public?command=returnOrderBook&%s", POLONIEX_API_URL, vals.Encode())
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)

	if err != nil {
		return PoloniexOrderbook{}, err
	}

	ob := PoloniexOrderbook{}
	for x := range resp.Asks {
		data := resp.Asks[x]
		price, err := strconv.ParseFloat(data[0].(string), 64)
		if err != nil {
			return ob, err
		}
		amount := data[1].(float64)
		ob.Asks = append(ob.Asks, PoloniexOrderbookItem{Price: price, Amount: amount})
	}

	for x := range resp.Bids {
		data := resp.Bids[x]
		price, err := strconv.ParseFloat(data[0].(string), 64)
		if err != nil {
			return ob, err
		}
		amount := data[1].(float64)
		ob.Bids = append(ob.Bids, PoloniexOrderbookItem{Price: price, Amount: amount})
	}
	return ob, nil
}

func (p *Poloniex) GetTradeHistory(currencyPair, start, end string) ([]PoloniexTradeHistory, error) {
	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if start != "" {
		vals.Set("start", start)
	}

	if end != "" {
		vals.Set("end", end)
	}

	resp := []PoloniexTradeHistory{}
	path := fmt.Sprintf("%s/public?command=returnTradeHistory&%s", POLONIEX_API_URL, vals.Encode())
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (p *Poloniex) GetChartData(currencyPair, start, end, period string) ([]PoloniexChartData, error) {
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

	resp := []PoloniexChartData{}
	path := fmt.Sprintf("%s/public?command=returnChartData&%s", POLONIEX_API_URL, vals.Encode())
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (p *Poloniex) GetCurrencies() (map[string]PoloniexCurrencies, error) {
	type Response struct {
		Data map[string]PoloniexCurrencies
	}
	resp := Response{}
	path := fmt.Sprintf("%s/public?command=returnCurrencies", POLONIEX_API_URL)
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp.Data)

	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

func (p *Poloniex) GetLoanOrders(currency string) (PoloniexLoanOrders, error) {
	resp := PoloniexLoanOrders{}
	path := fmt.Sprintf("%s/public?command=returnLoanOrders&currency=%s", POLONIEX_API_URL, currency)
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)

	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (p *Poloniex) GetBalances() (PoloniexBalance, error) {
	var result interface{}
	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_BALANCES, url.Values{}, &result)

	if err != nil {
		return PoloniexBalance{}, err
	}

	data := result.(map[string]interface{})
	balance := PoloniexBalance{}
	balance.Currency = make(map[string]float64)

	for x, y := range data {
		balance.Currency[x], _ = strconv.ParseFloat(y.(string), 64)
	}

	return balance, nil
}

type PoloniexCompleteBalances struct {
	Currency map[string]PoloniexCompleteBalance
}

func (p *Poloniex) GetCompleteBalances() (PoloniexCompleteBalances, error) {
	var result interface{}
	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_BALANCES_COMPLETE, url.Values{}, &result)

	if err != nil {
		return PoloniexCompleteBalances{}, err
	}

	data := result.(map[string]interface{})
	balance := PoloniexCompleteBalances{}
	balance.Currency = make(map[string]PoloniexCompleteBalance)

	for x, y := range data {
		dataVals := y.(map[string]interface{})
		balancesData := PoloniexCompleteBalance{}
		balancesData.Available, _ = strconv.ParseFloat(dataVals["available"].(string), 64)
		balancesData.OnOrders, _ = strconv.ParseFloat(dataVals["onOrders"].(string), 64)
		balancesData.BTCValue, _ = strconv.ParseFloat(dataVals["btcValue"].(string), 64)
		balance.Currency[x] = balancesData
	}

	return balance, nil
}

func (p *Poloniex) GetDepositAddresses() (PoloniexDepositAddresses, error) {
	var result interface{}
	addresses := PoloniexDepositAddresses{}
	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_DEPOSIT_ADDRESSES, url.Values{}, &result)

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

func (p *Poloniex) GenerateNewAddress(currency string) (string, error) {
	type Response struct {
		Success  int
		Error    string
		Response string
	}
	resp := Response{}
	values := url.Values{}
	values.Set("currency", currency)

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_GENERATE_NEW_ADDRESS, values, &resp)

	if err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}

	return resp.Response, nil
}

func (p *Poloniex) GetDepositsWithdrawals(start, end string) (PoloniexDepositsWithdrawals, error) {
	resp := PoloniexDepositsWithdrawals{}
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

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_DEPOSITS_WITHDRAWALS, values, &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (p *Poloniex) GetOpenOrders(currency string) (interface{}, error) {
	values := url.Values{}

	if currency != "" {
		values.Set("currencyPair", currency)
		result := PoloniexOpenOrdersResponse{}
		err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_ORDERS, values, &result.Data)

		if err != nil {
			return result, err
		}

		return result, nil
	} else {
		values.Set("currencyPair", "all")
		result := PoloniexOpenOrdersResponseAll{}
		err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_ORDERS, values, &result.Data)

		if err != nil {
			return result, err
		}

		return result, nil
	}
}

func (p *Poloniex) GetAuthenticatedTradeHistory(currency, start, end string) (interface{}, error) {
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	}

	if end != "" {
		values.Set("end", end)
	}

	if currency != "" && currency != "all" {
		values.Set("currencyPair", currency)
		result := PoloniexAuthenticatedTradeHistoryResponse{}
		err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_TRADE_HISTORY, values, &result.Data)

		if err != nil {
			return result, err
		}

		return result, nil
	} else {
		values.Set("currencyPair", "all")
		result := PoloniexAuthenticatedTradeHistoryAll{}
		err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_TRADE_HISTORY, values, &result.Data)

		if err != nil {
			return result, err
		}

		return result, nil
	}
}

func (p *Poloniex) PlaceOrder(currency string, rate, amount float64, immediate, fillOrKill, buy bool) (PoloniexOrderResponse, error) {
	result := PoloniexOrderResponse{}
	values := url.Values{}

	var orderType string
	if buy {
		orderType = POLONIEX_ORDER_BUY
	} else {
		orderType = POLONIEX_ORDER_SELL
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

func (p *Poloniex) CancelOrder(orderID int64) (bool, error) {
	result := PoloniexGenericResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_ORDER_CANCEL, values, &result)

	if err != nil {
		return false, err
	}

	if result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

func (p *Poloniex) MoveOrder(orderID int64, rate, amount float64) (PoloniexMoveOrderResponse, error) {
	result := PoloniexMoveOrderResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))

	if amount != 0 {
		values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_ORDER_MOVE, values, &result)

	if err != nil {
		return result, err
	}

	if result.Success != 1 {
		return result, errors.New(result.Error)
	}

	return result, nil
}

func (p *Poloniex) Withdraw(currency, address string, amount float64) (bool, error) {
	result := PoloniexWithdraw{}
	values := url.Values{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("address", address)

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_WITHDRAW, values, &result)

	if err != nil {
		return false, err
	}

	if result.Error != "" {
		return false, errors.New(result.Error)
	}

	return true, nil
}

func (p *Poloniex) GetFeeInfo() (PoloniexFee, error) {
	result := PoloniexFee{}
	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_FEE_INFO, url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (p *Poloniex) GetTradableBalances() (map[string]map[string]float64, error) {
	type Response struct {
		Data map[string]map[string]interface{}
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_TRADABLE_BALANCES, url.Values{}, &result.Data)

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

func (p *Poloniex) TransferBalance(currency, from, to string, amount float64) (bool, error) {
	values := url.Values{}
	result := PoloniexGenericResponse{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("fromAccount", from)
	values.Set("toAccount", to)

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_TRANSFER_BALANCE, values, &result)

	if err != nil {
		return false, err
	}

	if result.Error != "" && result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

func (p *Poloniex) GetMarginAccountSummary() (PoloniexMargin, error) {
	result := PoloniexMargin{}
	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_MARGIN_ACCOUNT_SUMMARY, url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (p *Poloniex) PlaceMarginOrder(currency string, rate, amount, lendingRate float64, buy bool) (PoloniexOrderResponse, error) {
	result := PoloniexOrderResponse{}
	values := url.Values{}

	var orderType string
	if buy {
		orderType = POLONIEX_MARGIN_BUY
	} else {
		orderType = POLONIEX_MARGIN_SELL
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

func (p *Poloniex) GetMarginPosition(currency string) (interface{}, error) {
	values := url.Values{}

	if currency != "" && currency != "all" {
		values.Set("currencyPair", currency)
		result := PoloniexMarginPosition{}
		err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_MARGIN_POSITION, values, &result)

		if err != nil {
			return result, err
		}

		return result, nil
	} else {
		values.Set("currencyPair", "all")

		type Response struct {
			Data map[string]PoloniexMarginPosition
		}

		result := Response{}
		err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_MARGIN_POSITION, values, &result.Data)

		if err != nil {
			return result, err
		}

		return result, nil
	}
}

func (p *Poloniex) CloseMarginPosition(currency string) (bool, error) {
	values := url.Values{}
	values.Set("currencyPair", currency)
	result := PoloniexGenericResponse{}

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_MARGIN_POSITION_CLOSE, values, &result)

	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

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

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_CREATE_LOAN_OFFER, values, &result)

	if err != nil {
		return 0, err
	}

	if result.Success == 0 {
		return 0, errors.New(result.Error)
	}

	return result.OrderID, nil
}

func (p *Poloniex) CancelLoanOffer(orderNumber int64) (bool, error) {
	result := PoloniexGenericResponse{}
	values := url.Values{}
	values.Set("orderID", strconv.FormatInt(orderNumber, 10))

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_CANCEL_LOAN_OFFER, values, &result)

	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

func (p *Poloniex) GetOpenLoanOffers() (map[string][]PoloniexLoanOffer, error) {
	type Response struct {
		Data map[string][]PoloniexLoanOffer
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_OPEN_LOAN_OFFERS, url.Values{}, &result.Data)

	if err != nil {
		return nil, err
	}

	if result.Data == nil {
		return nil, errors.New("There are no open loan offers.")
	}

	return result.Data, nil
}

func (p *Poloniex) GetActiveLoans() (PoloniexActiveLoans, error) {
	result := PoloniexActiveLoans{}
	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_ACTIVE_LOANS, url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (p *Poloniex) GetLendingHistory(start, end string) ([]PoloniexLendingHistory, error) {
	vals := url.Values{}

	if start != "" {
		vals.Set("start", start)
	}

	if end != "" {
		vals.Set("end", end)
	}

	resp := []PoloniexLendingHistory{}
	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_LENDING_HISTORY, vals, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (p *Poloniex) ToggleAutoRenew(orderNumber int64) (bool, error) {
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderNumber, 10))
	result := PoloniexGenericResponse{}

	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_AUTO_RENEW, values, &result)

	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

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

	path := fmt.Sprintf("%s/%s", POLONIEX_API_URL, POLONIEX_API_TRADING_ENDPOINT)
	resp, err := common.SendHTTPRequest(method, path, headers, bytes.NewBufferString(values.Encode()))

	if err != nil {
		return err
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}
	return nil
}
