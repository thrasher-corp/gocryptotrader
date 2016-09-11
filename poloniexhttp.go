package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"
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
	POLONIEX_AUTO_RENEW             = "toggleAutoRenew"
)

type Poloniex struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	AccessKey, SecretKey    string
	Fee                     float64
	BaseCurrencies          []string
	AvailablePairs          []string
	EnabledPairs            []string
}

type PoloniexTicker struct {
	Last          float64 `json:"last,string"`
	LowestAsk     float64 `json:"lowestAsk,string"`
	HighestBid    float64 `json:"highestBid,string"`
	PercentChange float64 `json:"percentChange,string"`
	BaseVolume    float64 `json:"baseVolume,string"`
	QuoteVolume   float64 `json:"quoteVolume,string"`
	IsFrozen      int     `json:"isFrozen,string"`
	High24Hr      float64 `json:"high24hr,string"`
	Low24Hr       float64 `json:"low24hr,string"`
}

func (p *Poloniex) SetDefaults() {
	p.Name = "Poloniex"
	p.Enabled = false
	p.Fee = 0
	p.Verbose = false
	p.Websocket = false
	p.RESTPollingDelay = 10
}

func (p *Poloniex) GetName() string {
	return p.Name
}

func (p *Poloniex) SetEnabled(enabled bool) {
	p.Enabled = enabled
}

func (p *Poloniex) IsEnabled() bool {
	return p.Enabled
}

func (p *Poloniex) Setup(exch Exchanges) {
	if !exch.Enabled {
		p.SetEnabled(false)
	} else {
		p.Enabled = true
		p.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		p.SetAPIKeys(exch.APIKey, exch.APISecret)
		p.RESTPollingDelay = exch.RESTPollingDelay
		p.Verbose = exch.Verbose
		p.Websocket = exch.Websocket
		p.BaseCurrencies = SplitStrings(exch.BaseCurrencies, ",")
		p.AvailablePairs = SplitStrings(exch.AvailablePairs, ",")
		p.EnabledPairs = SplitStrings(exch.EnabledPairs, ",")
	}
}

func (k *Poloniex) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (p *Poloniex) Start() {
	go p.Run()
}

func (p *Poloniex) SetAPIKeys(apiKey, apiSecret string) {
	p.AccessKey = apiKey
	p.SecretKey = apiSecret
}

func (p *Poloniex) GetFee() float64 {
	return p.Fee
}

func (p *Poloniex) Run() {
	if p.Verbose {
		log.Printf("%s Websocket: %s (url: %s).\n", p.GetName(), IsEnabled(p.Websocket), POLONIEX_WEBSOCKET_ADDRESS)
		log.Printf("%s polling delay: %ds.\n", p.GetName(), p.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", p.GetName(), len(p.EnabledPairs), p.EnabledPairs)
	}

	if p.Websocket {
		go p.WebsocketClient()
	}

	for p.Enabled {
		for _, x := range p.EnabledPairs {
			currency := x
			go func() {
				ticker, err := p.GetTicker()
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("Poloniex %s Last %f High %f Low %f Volume %f\n", currency, ticker[currency].Last, ticker[currency].High24Hr, ticker[currency].Low24Hr, ticker[currency].QuoteVolume)
				//AddExchangeInfo(p.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * p.RESTPollingDelay)
	}
}

func (p *Poloniex) GetTicker() (map[string]PoloniexTicker, error) {
	type response struct {
		Data map[string]PoloniexTicker
	}

	resp := response{}
	path := fmt.Sprintf("%s/public?command=returnTicker", POLONIEX_API_URL)
	err := SendHTTPGetRequest(path, true, &resp.Data)

	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

func (p *Poloniex) GetTickerPrice(currency string) TickerPrice {
	var tickerPrice TickerPrice
	ticker, err := p.GetTicker()
	if err != nil {
		log.Println(err)
		return tickerPrice
	}
	tickerPrice.CryptoCurrency = currency
	tickerPrice.Ask = ticker[currency].Last
	tickerPrice.Bid = ticker[currency].HighestBid
	tickerPrice.High = ticker[currency].HighestBid
	tickerPrice.Last = ticker[currency].Last
	tickerPrice.Low = ticker[currency].LowestAsk
	tickerPrice.Volume = ticker[currency].BaseVolume

	return tickerPrice
}

func (p *Poloniex) GetVolume() (interface{}, error) {
	var resp interface{}
	path := fmt.Sprintf("%s/public?command=return24hVolume", POLONIEX_API_URL)
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return resp, err
	}
	return resp, nil
}

type PoloniexOrderbook struct {
	Asks     [][]interface{} `json:"asks"`
	Bids     [][]interface{} `json:"bids"`
	IsFrozen string          `json:"isFrozen"`
}

//TO-DO: add support for individual pair depth fetching
func (p *Poloniex) GetOrderbook(currencyPair string, depth int) (map[string]PoloniexOrderbook, error) {
	type Response struct {
		Data map[string]PoloniexOrderbook
	}

	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if depth != 0 {
		vals.Set("depth", strconv.Itoa(depth))
	}

	resp := Response{}
	path := fmt.Sprintf("%s/public?command=returnOrderBook&%s", POLONIEX_API_URL, vals.Encode())
	err := SendHTTPGetRequest(path, true, &resp.Data)

	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

type PoloniexTradeHistory struct {
	GlobalTradeID int64   `json:"globalTradeID"`
	TradeID       int64   `json:"tradeID"`
	Date          string  `json:"date"`
	Type          string  `json:"type"`
	Rate          float64 `json:"rate,string"`
	Amount        float64 `json:"amount,string"`
	Total         float64 `json:"total,string"`
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
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

type PoloniexChartData struct {
	Date            int     `json:"date"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	Open            float64 `json:"open"`
	Close           float64 `json:"close"`
	Volume          float64 `json:"volume"`
	QuoteVolume     float64 `json:"quoteVolume"`
	WeightedAverage float64 `json:"weightedAverage"`
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
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

type PoloniexCurrencies struct {
	Name               string      `json:"name"`
	MaxDailyWithdrawal string      `json:"maxDailyWithdrawal"`
	TxFee              float64     `json:"txFee,string"`
	MinConfirmations   int         `json:"minConf"`
	DepositAddresses   interface{} `json:"depositAddress"`
	Disabled           int         `json:"disabled"`
	Delisted           int         `json:"delisted"`
	Frozen             int         `json:"frozen"`
}

func (p *Poloniex) GetCurrencies() (map[string]PoloniexCurrencies, error) {
	type Response struct {
		Data map[string]PoloniexCurrencies
	}
	resp := Response{}
	path := fmt.Sprintf("%s/public?command=returnCurrencies", POLONIEX_API_URL)
	err := SendHTTPGetRequest(path, true, &resp.Data)

	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

type PoloniexLoanOrder struct {
	Rate     float64 `json:"rate,string"`
	Amount   float64 `json:"amount,string"`
	RangeMin int     `json:"rangeMin"`
	RangeMax int     `json:"rangeMax"`
}

type PoloniexLoanOrders struct {
	Offers  []PoloniexLoanOrder `json:"offers"`
	Demands []PoloniexLoanOrder `json:"demands"`
}

func (p *Poloniex) GetLoanOrders(currency string) (PoloniexLoanOrders, error) {
	resp := PoloniexLoanOrders{}
	path := fmt.Sprintf("%s/public?command=returnLoanOrders&currency=%s", POLONIEX_API_URL, currency)
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return resp, err
	}
	return resp, nil
}

type PoloniexBalance struct {
	Currency map[string]float64
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

type PoloniexCompleteBalance struct {
	Available float64
	OnOrders  float64
	BTCValue  float64
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

type PoloniexDepositAddresses struct {
	Addresses map[string]string
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

type PoloniexDepositsWithdrawals struct {
	Deposits []struct {
		Currency      string    `json:"currency"`
		Address       string    `json:"address"`
		Amount        float64   `json:"amount,string"`
		Confirmations int       `json:"confirmations"`
		TransactionID string    `json:"txid"`
		Timestamp     time.Time `json:"timestamp"`
		Status        string    `json:"string"`
	} `json:"deposits"`
	Withdrawals []struct {
		WithdrawalNumber int64     `json:"withdrawalNumber"`
		Currency         string    `json:"currency"`
		Address          string    `json:"address"`
		Amount           float64   `json:"amount,string"`
		Confirmations    int       `json:"confirmations"`
		TransactionID    string    `json:"txid"`
		Timestamp        time.Time `json:"timestamp"`
		Status           string    `json:"string"`
		IPAddress        string    `json:"ipAddress"`
	} `json:"withdrawals"`
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

type PoloniexOrder struct {
	OrderNumber int64   `json:"orderNumber,string"`
	Type        string  `json:"type"`
	Rate        float64 `json:"rate,string"`
	Amount      float64 `json:"amount,string"`
	Total       float64 `json:"total,string"`
	Date        string  `json:"date"`
	Margin      float64 `json:"margin"`
}

type PoloniexOpenOrdersResponseAll struct {
	Data map[string][]PoloniexOrder
}

type PoloniexOpenOrdersResponse struct {
	Data []PoloniexOrder
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

type PoloniexAuthentictedTradeHistory struct {
	GlobalTradeID int64   `json:"globalTradeID"`
	TradeID       int64   `json:"tradeID,string"`
	Date          string  `json:"data,string"`
	Rate          float64 `json:"rate,string"`
	Amount        float64 `json:"amount,string"`
	Total         float64 `json:"total,string"`
	Fee           float64 `json:"fee,string"`
	OrderNumber   int64   `json:"orderNumber,string"`
	Type          string  `json:"type"`
	Category      string  `json:"category"`
}

type PoloniexAuthenticatedTradeHistoryAll struct {
	Data map[string][]PoloniexOrder
}

type PoloniexAuthenticatedTradeHistoryResponse struct {
	Data []PoloniexOrder
}

func (p *Poloniex) GetAuthenticatedTradeHistory(currency, start, end string) (interface{}, error) {
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	}

	if end != "" {
		values.Set("end", end)
	}

	if currency != "" {
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

type PoloniexResultingTrades struct {
	Amount  float64 `json:"amount,string"`
	Date    string  `json:"date"`
	Rate    float64 `json:"rate,string"`
	Total   float64 `json:"total,string"`
	TradeID int64   `json:"tradeID,string"`
	Type    string  `json:"type,string"`
}

type PoloniexOrderResponse struct {
	OrderNumber int64                     `json:"orderNumber,string"`
	Trades      []PoloniexResultingTrades `json:"resultingTrades"`
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

type PoloniexGenericResponse struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
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

type PoloniexMoveOrderResponse struct {
	Success     int                                  `json:"success"`
	Error       string                               `json:"error"`
	OrderNumber int64                                `json:"orderNumber,string"`
	Trades      map[string][]PoloniexResultingTrades `json:"resultingTrades"`
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

type PoloniexWithdraw struct {
	Response string `json:"response"`
	Error    string `json:"error"`
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

type PoloniexFee struct {
	MakerFee        float64 `json:"makerFee,string"`
	TakerFee        float64 `json:"takerFee,string"`
	ThirtyDayVolume float64 `json:"thirtyDayVolume,string"`
	NextTier        float64 `json:"nextTier,string"`
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

type PoloniexMargin struct {
	TotalValue    float64 `json:"totalValue,string"`
	ProfitLoss    float64 `json:"pl,string"`
	LendingFees   float64 `json:"lendingFees,string"`
	NetValue      float64 `json:"netValue,string"`
	BorrowedValue float64 `json:"totalBorrowedValue,string"`
	CurrentMargin float64 `json:"currentMargin,string"`
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

type PoloniexMarginPosition struct {
	Amount            float64 `json:"amount,string"`
	Total             float64 `json:"total,string"`
	BasePrice         float64 `json:"basePrice,string"`
	LiquidiationPrice float64 `json:"liquidiationPrice"`
	ProfitLoss        float64 `json:"pl,string"`
	LendingFees       float64 `json:"lendingFees,string"`
	Type              string  `json:"type"`
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

type PoloniexLoanOffer struct {
	ID        int64   `json:"id"`
	Rate      float64 `json:"rate,string"`
	Amount    float64 `json:"amount,string"`
	Duration  int     `json:"duration"`
	AutoRenew bool    `json:"autoRenew,int"`
	Date      string  `json:"date"`
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

type PoloniexActiveLoans struct {
	Provided []PoloniexLoanOffer `json:"provided"`
	Used     []PoloniexLoanOffer `json:"used"`
}

func (p *Poloniex) GetActiveLoans() (PoloniexActiveLoans, error) {
	result := PoloniexActiveLoans{}
	err := p.SendAuthenticatedHTTPRequest("POST", POLONIEX_ACTIVE_LOANS, url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
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
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	headers["Key"] = p.AccessKey

	nonce := time.Now().UnixNano()
	nonceStr := strconv.FormatInt(nonce, 10)

	values.Set("nonce", nonceStr)
	values.Set("command", endpoint)

	hmac := GetHMAC(HASH_SHA512, []byte(values.Encode()), []byte(p.SecretKey))
	headers["Sign"] = HexEncodeToString(hmac)

	path := fmt.Sprintf("%s/%s", POLONIEX_API_URL, POLONIEX_API_TRADING_ENDPOINT)
	resp, err := SendHTTPRequest(method, path, headers, bytes.NewBufferString(values.Encode()))

	if err != nil {
		return err
	}

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}
	return nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Poloniex exchange
func (e *Poloniex) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	accountBalance, err := e.GetBalances()
	if err != nil {
		return response, err
	}
	currencies := e.GetEnabledCurrencies()
	for i := 0; i < len(currencies); i++ {
		var exchangeCurrency ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = currencies[i]
		exchangeCurrency.TotalValue = accountBalance.Currency[currencies[i]]
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}
