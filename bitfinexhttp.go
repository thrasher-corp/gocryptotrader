package main

import (
	"fmt"
	"log"
	"encoding/json"
	"errors"
	"strings"
	"strconv"
	"time"
)

const (
	BITFINEX_API_URL = "https://api.bitfinex.com/v1/"
	BITFINEX_API_VERSION = "1"
	BITFINEX_TICKER = "pubticker/"
	BITFINEX_STATS = "stats/"
	BITFINEX_ORDERBOOK = "book/"
	BITFINEX_TRADES = "trades/"
	BITFINEX_SYMBOLS = "symbols/"
	BITFINEX_SYMBOLS_DETAILS = "symbols_details/"
	BITFINEX_DEPOSIT = "deposit/new"
	BITFINEX_ORDER_NEW = "order/new"
	BITFINEX_ORDER_CANCEL = "order/cancel"
	BITFINEX_ORDER_CANCEL_MULTI = "order/cancel/multi"
	BITFINEX_ORDER_CANCEL_ALL = "order/cancel/all"
	BITFINEX_ORDER_STATUS = "order/status"
	BITFINEX_ORDERS = "orders"
	BITFINEX_POSITIONS = "positions"
	BITFINEX_CLAIM_POSITION = "position/claim"
	BITFINEX_HISTORY = "history"
	BITFINEX_TRADE_HISTORY = "mytrades"
	BITFINEX_OFFER_NEW = "offer/new"
	BITFINEX_OFFER_CANCEL = "offer/cancel"
	BITFINEX_OFFER_STATUS = "offer/status"
	BITFINEX_OFFERS = "offers"
	BITFINEX_CREDITS = "credits"
	BITFINEX_SWAP_ACTIVE = "taken_swaps"
	BITFINEX_SWAP_CLOSE = "swap/close"
	BITFINEX_BALANCES = "balances"
	BITFINEX_ACCOUNT_INFO = "account_infos"
	BITFINEX_MARGIN_INFO = "margin_infos"
)

type BitfinexStats struct {
	Period int64
	Volume float64 `json:",string"`
}

type BitfinexTicker struct {
	Mid float64 `json:",string"`
	Bid float64 `json:",string"`
	Ask float64 `json:",string"`
	Last float64 `json:"Last_price,string"`
	Low float64 `json:",string"`
	High float64 `json:",string"`
	Volume float64 `json:",string"`
	Timestamp string
}

type MarginLimits struct {
	On_Pair string
	InitialMargin float64 `json:"initial_margin,string"`
	MarginRequirement float64 `json:"margin_requirement,string"`
	TradableBalance float64 `json:"tradable_balance,string"`
}

type BitfinexMarginInfo struct {
	MarginBalance float64 `json:"margin_balance,string"`
	TradableBalance float64 `json:"tradable_balance,string"`
	UnrealizedPL int64 `json:"unrealized_pl"`
	UnrealizedSwap int64 `json:"unrealized_swap"`
	NetValue float64 `json:"net_value,string"`
	RequiredMargin int64 `json:"required_margin"`
	Leverage float64 `json:"leverage,string"`
	MarginRequirement float64 `json:"margin_requirement,string"`
	MarginLimits []MarginLimits `json:"margin_limits"`
	Message string
}

type BitfinexActiveOrder struct {
	ID int64
	Symbol string
	Exchange string
	Price float64 `json:"Price,string"`
	Avg_Execution_Price float64 `json:"Price,string"`
	Side string
	Type string
	Timestamp string
	Is_Live bool
	Is_Cancelled bool
	Was_Forced bool
	OriginalAmount float64 `json:"original_amount,string"`
	RemainingAmount float64 `json:"remaining_amount,string"`
	ExecutedAmount float64 `json:"executed_amount,string"`
}

type BitfinexBalance struct {
	Type string
	Currency string
	Amount string
	Available string
}

type BitfinexOffer struct {
	Currency string
	Rate float64
	Period int64
	Direction string
	Type string
	Timestamp time.Time
	Is_Live bool
	Is_Cancelled bool
	Executed_Amount float64
	Remaining_Amount float64
	Original_Amount float64
}

type BookStructure struct {
	Price, Amount, Timestamp string
}

type BitfinexFee struct {
	Currency string
	TakerFees float64
	MakerFees float64
}

type BitfinexOrderbook struct {
	Bids []BookStructure
	Asks []BookStructure
}

type TradeStructure struct {
	Timestamp, Tid int64
	Price, Amount, Exchange, Type string
}

type SymbolsDetails struct {
	Pair, Initial_margin, Minimum_margin, Maximum_order_size, Minimum_order_size, Expiration string
	Price_precision int
}

type Bitfinex struct {
	Name string
	Enabled bool
	Verbose bool
	Websocket bool
	PollingDelay time.Duration
	APIKey, APISecret string
	Ticker BitfinexTicker
	Stats []BitfinexStats
	Orderbook BitfinexOrderbook
	Trades []TradeStructure
	SymbolsDetails []SymbolsDetails
	Fees []BitfinexFee
	ActiveOrders []BitfinexActiveOrder
	AccountBalance []BitfinexBalance
}

func (b *Bitfinex) SetDefaults() {
	b.Name = "Bitfinex"
	b.Enabled = true
	b.Verbose = false
	b.Websocket = false
	b.PollingDelay = 10
}

func (b *Bitfinex) GetName() (string) {
	return b.Name
}

func (b *Bitfinex) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *Bitfinex) IsEnabled() (bool) {
	return b.Enabled
}

func (b *Bitfinex) SetAPIKeys(apiKey, apiSecret string) {
	b.APIKey = apiKey
	b.APISecret = apiSecret
}

func (b *Bitfinex) Run() {
	if b.Verbose {
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.PollingDelay)
	}

	b.GetAccountFeeInfo()
	b.GetAccountBalance()

	for b.Enabled {
		go func() {
			BitfinexLTC := b.GetTicker("ltcusd")
			log.Printf("Bitfinex LTC: Last %f High %f Low %f Volume %f\n", BitfinexLTC.Last, BitfinexLTC.High, BitfinexLTC.Low, BitfinexLTC.Volume)
			AddExchangeInfo(b.GetName(), "LTC", BitfinexLTC.Last, BitfinexLTC.Volume)
		}()

		go func() {
			BitfinexBTC := b.GetTicker("btcusd")
			log.Printf("Bitfinex BTC: Last %f High %f Low %f Volume %f\n", BitfinexBTC.Last, BitfinexBTC.High, BitfinexBTC.Low, BitfinexBTC.Volume)
			AddExchangeInfo(b.GetName(), "BTC", BitfinexBTC.Last, BitfinexBTC.Volume)
		}()
		time.Sleep(time.Second * b.PollingDelay)
	}
}

func (b *Bitfinex) GetFee(maker bool, symbol string) (float64, error) {
	for _, i := range b.Fees {
		if symbol == i.Currency {
			if maker {
				return i.MakerFees, nil
			} else {
				return i.TakerFees, nil
			}
		}
	}
	return 0, errors.New("Unable to find specified currency.")
}

func (b *Bitfinex) GetAccountBalance() (bool, error) {
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_BALANCES, nil, &b.AccountBalance)

	if err != nil {
		log.Println(err)
		return false, err
	}
	return true, nil
}

func (b *Bitfinex) GetAccountFeeInfo() (bool, error) {
	type Fee struct {
		Pairs string `json:"pairs"`
		MakerFees string `json:"maker_fees"`
		TakerFees string `json:"taker_fees"`
	}

	type Response struct {
		Data []map[string][]Fee
	}

	var resp Response
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ACCOUNT_INFO, nil, &resp.Data)

	if err != nil {
		return false, err
	}

	Fees := []BitfinexFee{}

	for _, i := range resp.Data[0]["fees"] {
		var bfxFee BitfinexFee
		bfxFee.Currency = i.Pairs
		bfxFee.MakerFees, _ = strconv.ParseFloat(i.MakerFees, 64)
		bfxFee.TakerFees, _ = strconv.ParseFloat(i.TakerFees, 64)
		Fees = append(Fees, bfxFee)
	}

	b.Fees = Fees
	return true, nil
}

func (b *Bitfinex) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	request := make(map[string]interface{})
	request["request"] = fmt.Sprintf("/v%s/%s", BITFINEX_API_VERSION, path)
	request["nonce"] = strconv.FormatInt(time.Now().UnixNano(), 10)

	if params != nil {
		for key, value:= range params {
			request[key] = value
		}
	}

	PayloadJson, err := json.Marshal(request)

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

	resp, err := SendHTTPRequest(method, BITFINEX_API_URL + path, headers, strings.NewReader(""))

	if b.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}
	
	err = json.Unmarshal([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}
	
	return nil
}

func (b *Bitfinex) GetTicker(symbol string) (BitfinexTicker) {
	err := SendHTTPGetRequest(BITFINEX_API_URL + BITFINEX_TICKER + symbol, true, &b.Ticker)
	if err != nil {
		log.Println(err)
		return BitfinexTicker{}
	}
	return b.Ticker
}

func (b *Bitfinex) GetStats(symbol string) (bool) {
	err := SendHTTPGetRequest(BITFINEX_API_URL + BITFINEX_STATS + symbol, true, &b.Stats)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (b *Bitfinex) GetOrderbook(symbol string) (bool) {
	err := SendHTTPGetRequest(BITFINEX_API_URL + BITFINEX_ORDERBOOK + symbol, true, &b.Orderbook)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (b *Bitfinex) GetTrades(symbol string) (bool) {
	err := SendHTTPGetRequest(BITFINEX_API_URL + BITFINEX_TRADES + symbol, true, &b.Trades)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (b *Bitfinex) GetSymbols() (bool) {
	err := SendHTTPGetRequest(BITFINEX_API_URL + BITFINEX_SYMBOLS, false, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (b *Bitfinex) GetSymbolsDetails() (bool) {
	err := SendHTTPGetRequest(BITFINEX_API_URL + BITFINEX_SYMBOLS_DETAILS, false, &b.SymbolsDetails)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (b *Bitfinex) NewDeposit(Symbol, Method, Wallet string) {
	request := make(map[string]interface{})
	request["currency"] = Symbol
	request["method"] = Method
	request["wallet_name"] = Wallet

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_DEPOSIT, request, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) NewOrder(Symbol string, Amount float64, Price float64, Buy bool, Type string, Hidden bool) {
	request := make(map[string]interface{})
	request["symbol"] = Symbol
	request["amount"] = fmt.Sprintf("%.8f", Amount)
	request["price"] = fmt.Sprintf("%.5f", Price)
	request["exchange"] = "bitfinex"

	if Buy {
		request["side"] = "buy"
	} else {
		request["side"] = "sell"
	}

	//request["is_hidden"] - currently not implemented
	request["type"] = Type
	
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_NEW, request, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) CancelOrder(OrderID int64) (bool) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_CANCEL, request, nil)

	if err != nil {
		log.Println(err)
		return false
	}

	return true
}

func (b *Bitfinex) CancelMultiplateOrders(OrderIDs []int64) {
	request := make(map[string]interface{})
	request["order_ids"] = OrderIDs

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_CANCEL_MULTI, request, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) CancelAllOrders() {
	err := b.SendAuthenticatedHTTPRequest("GET", BITFINEX_ORDER_CANCEL_ALL, nil, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) ReplaceOrder(OrderID int) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID
}

func (b *Bitfinex) GetOrderStatus(OrderID int64) (BitfinexActiveOrder) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID
	orderStatus := BitfinexActiveOrder{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_STATUS, request, &orderStatus)

	if err != nil {
		log.Println(err)
		return BitfinexActiveOrder{}
	}

	return orderStatus
}

func (b *Bitfinex) GetActiveOrders() (bool) {
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDERS, nil, &b.ActiveOrders)

	if err != nil {
		log.Println(err)
		return false
	}

	log.Printf("Bitfinex active orders: %d\n", len(b.ActiveOrders))
	return true
}

func (b *Bitfinex) GetActivePositions() {
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_POSITIONS, nil, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) ClaimPosition(PositionID int) {
	request := make(map[string]interface{})
	request["position_id"] = PositionID

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_CLAIM_POSITION, nil, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) GetBalanceHistory(symbol string, timeSince time.Time, timeUntil time.Time, limit int, wallet string) {
	request := make(map[string]interface{})
	request["currency"] = symbol
	request["since"] = timeSince
	request["until"] = timeUntil

	if limit > 0 {
		request["limit"] = limit
	}
	
	if len(wallet) > 0 {
		request["wallet"] = wallet
	}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_HISTORY, request, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) GetTradeHistory(symbol string, timestamp time.Time, limit int) {
	request := make(map[string]interface{})
	request["currency"] = symbol
	request["timestamp"] = timestamp

	if (limit > 0) {
		request["limit_trades"] = limit
	}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_TRADE_HISTORY, nil, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) NewOffer(symbol string, amount, rate float64, period int64, direction string) (int64) {
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

func (b *Bitfinex) CancelOffer(OfferID int64) {
	request := make(map[string]interface{})
	request["offer_id"] = OfferID

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_OFFER_CANCEL, request, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) GetOfferStatus(OfferID int64) {
	request := make(map[string]interface{})
	request["offer_id"] = OfferID
	offer := BitfinexOffer{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ORDER_STATUS, request, &offer)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) GetActiveOffers() {
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_OFFERS, nil, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) GetActiveCredits() {
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_CREDITS, nil, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) GetActiveSwaps() {
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_SWAP_ACTIVE, nil, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) CloseSwap(SwapID int64) {
	request := make(map[string]interface{})
	request["swap_id"] = SwapID

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_SWAP_CLOSE, request, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitfinex) GetMarginInfo() {
	type Response struct {
		Data []BitfinexMarginInfo
	}

	response := Response{}

	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_MARGIN_INFO, nil, &response.Data)

	if err != nil {
		log.Println(err)
	}
}

