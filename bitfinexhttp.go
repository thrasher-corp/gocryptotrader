package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	BITFINEX_API_URL              = "https://api.bitfinex.com/v1/"
	BITFINEX_API_VERSION          = "1"
	BITFINEX_TICKER               = "pubticker/"
	BITFINEX_STATS                = "stats/"
	BITFINEX_LENDBOOK             = "lendbook/"
	BITFINEX_ORDERBOOK            = "book/"
	BITFINEX_TRADES               = "trades/"
	BITFINEX_LENDS                = "lends/"
	BITFINEX_SYMBOLS              = "symbols/"
	BITFINEX_SYMBOLS_DETAILS      = "symbols_details/"
	BITFINEX_ACCOUNT_INFO         = "account_infos"
	BITFINEX_DEPOSIT              = "deposit/new"
	BITFINEX_ORDER_NEW            = "order/new"
	BITFINEX_ORDER_NEW_MULTI      = "order/new/multi"
	BITFINEX_ORDER_CANCEL         = "order/cancel"
	BITFINEX_ORDER_CANCEL_MULTI   = "order/cancel/multi"
	BITFINEX_ORDER_CANCEL_ALL     = "order/cancel/all"
	BITFINEX_ORDER_CANCEL_REPLACE = "order/cancel/replace"
	BITFINEX_ORDER_STATUS         = "order/status"
	BITFINEX_ORDERS               = "orders"
	BITFINEX_POSITIONS            = "positions"
	BITFINEX_CLAIM_POSITION       = "position/claim"
	BITFINEX_HISTORY              = "history"
	BITFINEX_HISTORY_MOVEMENTS    = "history/movements"
	BITFINEX_TRADE_HISTORY        = "mytrades"
	BITFINEX_OFFER_NEW            = "offer/new"
	BITFINEX_OFFER_CANCEL         = "offer/cancel"
	BITFINEX_OFFER_STATUS         = "offer/status"
	BITFINEX_OFFERS               = "offers"
	BITFINEX_MARGIN_ACTIVE_FUNDS  = "taken_funds"
	BITFINEX_MARGIN_TOTAL_FUNDS   = "total_taken_funds"
	BITFINEX_MARGIN_CLOSE         = "funding/close"
	BITFINEX_BALANCES             = "balances"
	BITFINEX_MARGIN_INFO          = "margin_infos"
	BITFINEX_TRANSFER             = "transfer"
	BITFINEX_WITHDRAWAL           = "withdrawal"
)

type BitfinexStats struct {
	Period int64
	Volume float64 `json:",string"`
}

type BitfinexTicker struct {
	Mid       float64 `json:",string"`
	Bid       float64 `json:",string"`
	Ask       float64 `json:",string"`
	Last      float64 `json:"Last_price,string"`
	Low       float64 `json:",string"`
	High      float64 `json:",string"`
	Volume    float64 `json:",string"`
	Timestamp string
}

type MarginLimits struct {
	On_Pair           string
	InitialMargin     float64 `json:"initial_margin,string"`
	MarginRequirement float64 `json:"margin_requirement,string"`
	TradableBalance   float64 `json:"tradable_balance,string"`
}

type BitfinexMarginInfo struct {
	MarginBalance     float64        `json:"margin_balance,string"`
	TradableBalance   float64        `json:"tradable_balance,string"`
	UnrealizedPL      int64          `json:"unrealized_pl"`
	UnrealizedSwap    int64          `json:"unrealized_swap"`
	NetValue          float64        `json:"net_value,string"`
	RequiredMargin    int64          `json:"required_margin"`
	Leverage          float64        `json:"leverage,string"`
	MarginRequirement float64        `json:"margin_requirement,string"`
	MarginLimits      []MarginLimits `json:"margin_limits"`
	Message           string
}

type BitfinexOrder struct {
	ID                    int64
	Symbol                string
	Exchange              string
	Price                 float64 `json:"price,string"`
	AverageExecutionPrice float64 `json:"avg_execution_price,string"`
	Side                  string
	Type                  string
	Timestamp             string
	IsLive                bool    `json:"is_live"`
	IsCancelled           bool    `json:"is_cancelled"`
	IsHidden              bool    `json:"is_hidden"`
	WasForced             bool    `json:"was_forced"`
	OriginalAmount        float64 `json:"original_amount,string"`
	RemainingAmount       float64 `json:"remaining_amount,string"`
	ExecutedAmount        float64 `json:"executed_amount,string"`
	OrderID               int64   `json:"order_id"`
}

type BitfinexPlaceOrder struct {
	Symbol   string  `json:"symbol"`
	Amount   float64 `json:"amount,string"`
	Price    float64 `json:"price,string"`
	Exchange string  `json:"exchange"`
	Side     string  `json:"side"`
	Type     string  `json:"type"`
}

type BitfinexBalance struct {
	Type      string
	Currency  string
	Amount    float64 `json:"amount,string"`
	Available float64 `json:"available,string"`
}

type BitfinexOffer struct {
	ID              int64
	Currency        string
	Rate            float64 `json:"rate,string"`
	Period          int64
	Direction       string
	Timestamp       string
	Type            string
	IsLive          bool    `json:"is_live"`
	IsCancelled     bool    `json:"is_cancelled"`
	OriginalAmount  float64 `json:"original_amount,string"`
	RemainingAmount float64 `json:"remaining_amount,string"`
	ExecutedAmount  float64 `json:"remaining_amount,string"`
}

type BookStructure struct {
	Price, Amount, Timestamp string
}

type BitfinexFee struct {
	Currency  string
	TakerFees float64
	MakerFees float64
}

type BitfinexOrderbook struct {
	Bids []BookStructure
	Asks []BookStructure
}

type BitfinexTradeStructure struct {
	Timestamp, Tid                int64
	Price, Amount, Exchange, Type string
}

type BitfinexSymbolDetails struct {
	Pair             string  `json:"pair"`
	PricePrecision   int     `json:"price_precision"`
	InitialMargin    float64 `json:"initial_margin,string"`
	MinimumMargin    float64 `json:"minimum_margin,string"`
	MaximumOrderSize float64 `json:"maximum_order_size,string"`
	MinimumOrderSize float64 `json:"minimum_order_size,string"`
	Expiration       string  `json:"expiration"`
}

type Bitfinex struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	APIKey, APISecret       string
	ActiveOrders            []BitfinexOrder
	BaseCurrencies          []string
	AvailablePairs          []string
	EnabledPairs            []string
	WebsocketConn           *websocket.Conn
	WebsocketSubdChannels   map[int]BitfinexWebsocketChanInfo
}

func (b *Bitfinex) SetDefaults() {
	b.Name = "Bitfinex"
	b.Enabled = false
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
	b.WebsocketSubdChannels = make(map[int]BitfinexWebsocketChanInfo)
}

func (b *Bitfinex) GetName() string {
	return b.Name
}

func (b *Bitfinex) Setup(exch Exchanges) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.BaseCurrencies = SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = SplitStrings(exch.EnabledPairs, ",")
	}
}

func (b *Bitfinex) Start() {
	go b.Run()
}

func (b *Bitfinex) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *Bitfinex) IsEnabled() bool {
	return b.Enabled
}

func (b *Bitfinex) SetAPIKeys(apiKey, apiSecret string) {
	b.APIKey = apiKey
	b.APISecret = apiSecret
}

func (b *Bitfinex) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		go b.WebsocketClient()
	}

	exchangeProducts, err := b.GetSymbols()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", b.GetName())
	} else {
		exchangeProducts = SplitStrings(StringToUpper(JoinStrings(exchangeProducts, ",")), ",")
		diff := StringSliceDifference(b.AvailablePairs, exchangeProducts)
		if len(diff) > 0 {
			exch, err := GetExchangeConfig(b.Name)
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("%s Updating available pairs. Difference: %s.\n", b.Name, diff)
				exch.AvailablePairs = JoinStrings(exchangeProducts, ",")
				UpdateExchangeConfig(exch)
			}
		}
	}

	for b.Enabled {
		for _, x := range b.EnabledPairs {
			currency := x
			go func() {
				ticker, err := b.GetTicker(currency, nil)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("Bitfinex %s Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				AddExchangeInfo(b.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *Bitfinex) GetTicker(symbol string, values url.Values) (BitfinexTicker, error) {
	path := EncodeURLValues(BITFINEX_API_URL+BITFINEX_TICKER+symbol, values)
	response := BitfinexTicker{}
	err := SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return response, err
	}
	return response, nil
}

type BitfinexLendbookBidAsk struct {
	Rate            float64 `json:"rate,string"`
	Amount          float64 `json:"amount,string"`
	Period          int     `json:"period"`
	Timestamp       string  `json:"timestamp"`
	FlashReturnRate string  `json:"frr"`
}

type BitfinexLendbook struct {
	Bids []BitfinexLendbookBidAsk `json:"bids"`
	Asks []BitfinexLendbookBidAsk `json:"asks"`
}

func (b *Bitfinex) GetStats(symbol string) (BitfinexStats, error) {
	response := BitfinexStats{}
	err := SendHTTPGetRequest(BITFINEX_API_URL+BITFINEX_STATS+symbol, true, &response)
	if err != nil {
		return response, err
	}
	return response, nil
}

func (b *Bitfinex) GetLendbook(symbol string, values url.Values) (BitfinexLendbook, error) {
	path := EncodeURLValues(BITFINEX_API_URL+BITFINEX_LENDBOOK+symbol, values)
	response := BitfinexLendbook{}
	err := SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return response, err
	}
	return response, nil
}

func (b *Bitfinex) GetOrderbook(symbol string, values url.Values) (BitfinexOrderbook, error) {
	path := EncodeURLValues(BITFINEX_API_URL+BITFINEX_ORDERBOOK+symbol, values)
	response := BitfinexOrderbook{}
	err := SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return response, err
	}
	return response, nil
}

func (b *Bitfinex) GetTrades(symbol string, values url.Values) ([]BitfinexTradeStructure, error) {
	path := EncodeURLValues(BITFINEX_API_URL+BITFINEX_TRADES+symbol, values)
	response := []BitfinexTradeStructure{}
	err := SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

type BitfinexLends struct {
	Rate       float64 `json:"rate,string"`
	AmountLent float64 `json:"amount_lent,string"`
	AmountUsed float64 `json:"amount_used,string"`
	Timestamp  int64   `json:"timestamp"`
}

func (b *Bitfinex) GetLends(symbol string, values url.Values) ([]BitfinexLends, error) {
	path := EncodeURLValues(BITFINEX_API_URL+BITFINEX_LENDS+symbol, values)
	response := []BitfinexLends{}
	err := SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Bitfinex) GetSymbols() ([]string, error) {
	products := []string{}
	err := SendHTTPGetRequest(BITFINEX_API_URL+BITFINEX_SYMBOLS, true, &products)
	if err != nil {
		return nil, err
	}
	return products, nil
}

func (b *Bitfinex) GetSymbolsDetails() ([]BitfinexSymbolDetails, error) {
	response := []BitfinexSymbolDetails{}
	err := SendHTTPGetRequest(BITFINEX_API_URL+BITFINEX_SYMBOLS_DETAILS, true, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

type BitfinexAccountInfo struct {
	MakerFees string `json:"maker_fees"`
	TakerFees string `json:"taker_fees"`
	Fees      []struct {
		Pairs     string `json:"pairs"`
		MakerFees string `json:"maker_fees"`
		TakerFees string `json:"taker_fees"`
	} `json:"fees"`
}

func (b *Bitfinex) GetAccountInfo() ([]BitfinexAccountInfo, error) {
	response := []BitfinexAccountInfo{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ACCOUNT_INFO, nil, &response)

	if err != nil {
		log.Println(err)
	}

	log.Println(response)
	return response, nil
}

type BitfinexDepositResponse struct {
	Result   string `json:"string"`
	Method   string `json:"method"`
	Currency string `json:"currency"`
	Address  string `json:"address"`
}

func (b *Bitfinex) NewDeposit(method, walletName string, renew int) (BitfinexDepositResponse, error) {
	request := make(map[string]interface{})
	request["method"] = method
	request["wallet_name"] = walletName
	request["renew"] = renew
	response := BitfinexDepositResponse{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_DEPOSIT, request, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

func (b *Bitfinex) NewOrder(Symbol string, Amount float64, Price float64, Buy bool, Type string, Hidden bool) (BitfinexOrder, error) {
	request := make(map[string]interface{})
	request["symbol"] = Symbol
	request["amount"] = strconv.FormatFloat(Amount, 'f', -1, 64)
	request["price"] = strconv.FormatFloat(Price, 'f', -1, 64)
	request["exchange"] = "bitfinex"

	if Buy {
		request["side"] = "buy"
	} else {
		request["side"] = "sell"
	}

	request["type"] = Type
	//request["is_hidden"] = Hidden

	response := BitfinexOrder{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_NEW, request, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

type BitfinexOrderMultiResponse struct {
	Orders []BitfinexOrder `json:"order_ids"`
	Status string          `json:"status"`
}

func (b *Bitfinex) NewOrderMulti(orders []BitfinexPlaceOrder) (BitfinexOrderMultiResponse, error) {
	request := make(map[string]interface{})
	request["orders"] = orders

	response := BitfinexOrderMultiResponse{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_NEW_MULTI, request, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

func (b *Bitfinex) CancelOrder(OrderID int64) (BitfinexOrder, error) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID
	response := BitfinexOrder{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_CANCEL, request, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

type BitfinexGenericResponse struct {
	Result string `json:"result"`
}

func (b *Bitfinex) CancelMultiplateOrders(OrderIDs []int64) (string, error) {
	request := make(map[string]interface{})
	request["order_ids"] = OrderIDs
	response := BitfinexGenericResponse{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_CANCEL_MULTI, request, nil)

	if err != nil {
		return "", err
	}

	return response.Result, nil
}

func (b *Bitfinex) CancelAllOrders() (string, error) {
	response := BitfinexGenericResponse{}
	err := b.SendAuthenticatedHTTPRequest("GET", BITFINEX_ORDER_CANCEL_ALL, nil, nil)

	if err != nil {
		return "", err
	}

	return response.Result, nil
}

func (b *Bitfinex) ReplaceOrder(OrderID int64, Symbol string, Amount float64, Price float64, Buy bool, Type string, Hidden bool) (BitfinexOrder, error) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID
	request["symbol"] = Symbol
	request["amount"] = strconv.FormatFloat(Amount, 'f', -1, 64)
	request["price"] = strconv.FormatFloat(Price, 'f', -1, 64)
	request["exchange"] = "bitfinex"

	if Buy {
		request["side"] = "buy"
	} else {
		request["side"] = "sell"
	}

	request["type"] = Type
	//request["is_hidden"] = Hidden

	response := BitfinexOrder{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_CANCEL_REPLACE, request, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

func (b *Bitfinex) GetOrderStatus(OrderID int64) (BitfinexOrder, error) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID
	orderStatus := BitfinexOrder{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_STATUS, request, &orderStatus)

	if err != nil {
		return orderStatus, err
	}

	return orderStatus, err
}

func (b *Bitfinex) GetActiveOrders() ([]BitfinexOrder, error) {
	response := []BitfinexOrder{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDERS, nil, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

type BitfinexPosition struct {
	ID        int64   `json:"id"`
	Symbol    string  `json:"string"`
	Status    string  `json:"active"`
	Base      float64 `json:"base,string"`
	Amount    float64 `json:"amount,string"`
	Timestamp string  `json:"timestamp"`
	Swap      float64 `json:"swap,string"`
	PL        float64 `json:"pl,string"`
}

func (b *Bitfinex) GetActivePositions() ([]BitfinexPosition, error) {
	response := []BitfinexPosition{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_POSITIONS, nil, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (b *Bitfinex) ClaimPosition(PositionID int) (BitfinexPosition, error) {
	request := make(map[string]interface{})
	request["position_id"] = PositionID
	response := BitfinexPosition{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_CLAIM_POSITION, nil, nil)

	if err != nil {
		return BitfinexPosition{}, err
	}

	return response, nil
}

type BitfinexBalanceHistory struct {
	Currency    string  `json:"currency"`
	Amount      float64 `json:"amount,string"`
	Balance     float64 `json:"balance,string"`
	Description string  `json:"description"`
	Timestamp   string  `json:"timestamp"`
}

func (b *Bitfinex) GetBalanceHistory(symbol string, timeSince time.Time, timeUntil time.Time, limit int, wallet string) ([]BitfinexBalanceHistory, error) {
	request := make(map[string]interface{})
	request["currency"] = symbol

	if !timeSince.IsZero() {
		request["since"] = timeSince
	}

	if !timeUntil.IsZero() {
		request["until"] = timeUntil
	}

	if limit > 0 {
		request["limit"] = limit
	}

	if len(wallet) > 0 {
		request["wallet"] = wallet
	}

	response := []BitfinexBalanceHistory{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_HISTORY, request, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

type BitfinexMovementHistory struct {
	ID          int64   `json:"id"`
	Currency    string  `json:"currency"`
	Method      string  `json:"method"`
	Type        string  `json:"withdrawal"`
	Amount      float64 `json:"amount,string"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Timestamp   string  `json:"timestamp"`
}

func (b *Bitfinex) GetMovementHistory(symbol, method string, timeSince, timeUntil time.Time, limit int) ([]BitfinexMovementHistory, error) {
	request := make(map[string]interface{})
	request["currency"] = symbol

	if len(method) > 0 {
		request["method"] = method
	}

	if !timeSince.IsZero() {
		request["since"] = timeSince
	}

	if !timeUntil.IsZero() {
		request["until"] = timeUntil
	}

	if limit > 0 {
		request["limit"] = limit
	}

	response := []BitfinexMovementHistory{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_HISTORY_MOVEMENTS, request, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

type BitfinexTradeHistory struct {
	Price       float64 `json:"price,string"`
	Amount      float64 `json:"amount,string"`
	Timestamp   string  `json:"timestamp"`
	Exchange    string  `json:"exchange"`
	Type        string  `json:"type"`
	FeeCurrency string  `json:"fee_currency"`
	FeeAmount   float64 `json:"fee_amount,string"`
	TID         int64   `json:"tid"`
	OrderID     int64   `json:"order_id"`
}

func (b *Bitfinex) GetTradeHistory(symbol string, timestamp, until time.Time, limit, reverse int) ([]BitfinexTradeHistory, error) {
	request := make(map[string]interface{})
	request["currency"] = symbol
	request["timestamp"] = timestamp

	if !until.IsZero() {
		request["until"] = until
	}

	if limit > 0 {
		request["limit"] = limit
	}

	if reverse > 0 {
		request["reverse"] = reverse
	}

	response := []BitfinexTradeHistory{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_TRADE_HISTORY, request, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (b *Bitfinex) NewOffer(symbol string, amount, rate float64, period int64, direction string) int64 {
	request := make(map[string]interface{})
	request["currency"] = symbol
	request["amount"] = amount
	request["rate"] = rate
	request["period"] = period
	request["direction"] = direction

	type OfferResponse struct {
		Offer_Id int64
	}

	response := OfferResponse{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_OFFER_NEW, request, &response)

	if err != nil {
		log.Println(err)
		return 0
	}

	return response.Offer_Id
}

func (b *Bitfinex) CancelOffer(OfferID int64) (BitfinexOffer, error) {
	request := make(map[string]interface{})
	request["offer_id"] = OfferID
	response := BitfinexOffer{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_OFFER_CANCEL, request, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

func (b *Bitfinex) GetOfferStatus(OfferID int64) (BitfinexOffer, error) {
	request := make(map[string]interface{})
	request["offer_id"] = OfferID
	response := BitfinexOffer{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_STATUS, request, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

type BitfinexMarginFunds struct {
	ID         int64   `json:"id"`
	PositionID int64   `json:"position_id"`
	Currency   string  `json:"currency"`
	Rate       float64 `json:"rate,string"`
	Period     int     `json:"period"`
	Amount     float64 `json:"amount,string"`
	Timestamp  string  `json:"timestamp"`
}

func (b *Bitfinex) GetActiveOffers() ([]BitfinexOffer, error) {
	response := []BitfinexOffer{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_OFFERS, nil, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (b *Bitfinex) GetActiveMarginFunding() ([]BitfinexMarginFunds, error) {
	response := []BitfinexMarginFunds{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_MARGIN_ACTIVE_FUNDS, nil, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

type BitfinexMarginTotalTakenFunds struct {
	PositionPair string  `json:"position_pair"`
	TotalSwaps   float64 `json:"total_swaps,string"`
}

func (b *Bitfinex) GetMarginTotalTakenFunds() ([]BitfinexMarginTotalTakenFunds, error) {
	response := []BitfinexMarginTotalTakenFunds{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_MARGIN_TOTAL_FUNDS, nil, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (b *Bitfinex) CloseMarginFunding(SwapID int64) (BitfinexOffer, error) {
	request := make(map[string]interface{})
	request["swap_id"] = SwapID
	response := BitfinexOffer{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_MARGIN_CLOSE, request, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

func (b *Bitfinex) GetAccountBalance() ([]BitfinexBalance, error) {
	response := []BitfinexBalance{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_BALANCES, nil, &response)

	if err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Bitfinex) GetMarginInfo() ([]BitfinexMarginInfo, error) {
	response := []BitfinexMarginInfo{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_MARGIN_INFO, nil, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

type BitfinexWalletTransfer struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (b *Bitfinex) WalletTransfer(amount float64, currency, walletFrom, walletTo string) ([]BitfinexWalletTransfer, error) {
	request := make(map[string]interface{})
	request["amount"] = amount
	request["currency"] = currency
	request["walletfrom"] = walletFrom
	request["walletTo"] = walletTo

	response := []BitfinexWalletTransfer{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_TRANSFER, request, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

type BitfinexWithdrawal struct {
	Status       string `json:"status"`
	Message      string `json:"message"`
	WithdrawalID int64  `json:"withdrawal_id"`
}

func (b *Bitfinex) Withdrawal(withdrawType, wallet, address string, amount float64) ([]BitfinexWithdrawal, error) {
	request := make(map[string]interface{})
	request["withdrawal_type"] = withdrawType
	request["walletselected"] = wallet
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["address"] = address

	response := []BitfinexWithdrawal{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_WITHDRAWAL, request, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (b *Bitfinex) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	request := make(map[string]interface{})
	request["request"] = fmt.Sprintf("/v%s/%s", BITFINEX_API_VERSION, path)
	request["nonce"] = strconv.FormatInt(time.Now().UnixNano(), 10)

	if params != nil {
		for key, value := range params {
			request[key] = value
		}
	}

	PayloadJson, err := JSONEncode(request)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	if b.Verbose {
		log.Printf("Request JSON: %s\n", PayloadJson)
	}

	PayloadBase64 := Base64Encode(PayloadJson)
	hmac := GetHMAC(HASH_SHA512_384, []byte(PayloadBase64), []byte(b.APISecret))
	headers := make(map[string]string)
	headers["X-BFX-APIKEY"] = b.APIKey
	headers["X-BFX-PAYLOAD"] = PayloadBase64
	headers["X-BFX-SIGNATURE"] = HexEncodeToString(hmac)

	resp, err := SendHTTPRequest(method, BITFINEX_API_URL+path, headers, strings.NewReader(""))

	if b.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
