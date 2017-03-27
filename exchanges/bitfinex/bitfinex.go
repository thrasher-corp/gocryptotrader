package bitfinex

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
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

type Bitfinex struct {
	exchange.ExchangeBase
	WebsocketConn         *websocket.Conn
	WebsocketSubdChannels map[int]BitfinexWebsocketChanInfo
}

func (b *Bitfinex) SetDefaults() {
	b.Name = "Bitfinex"
	b.Enabled = false
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
	b.WebsocketSubdChannels = make(map[int]BitfinexWebsocketChanInfo)
}

func (b *Bitfinex) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

func (b *Bitfinex) GetTicker(symbol string, values url.Values) (BitfinexTicker, error) {
	path := common.EncodeURLValues(BITFINEX_API_URL+BITFINEX_TICKER+symbol, values)
	response := BitfinexTicker{}
	err := common.SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return response, err
	}
	return response, nil
}

func (b *Bitfinex) GetStats(symbol string) ([]BitfinexStats, error) {
	response := []BitfinexStats{}
	err := common.SendHTTPGetRequest(BITFINEX_API_URL+BITFINEX_STATS+symbol, true, &response)
	if err != nil {
		return response, err
	}
	return response, nil
}

func (b *Bitfinex) GetLendbook(symbol string, values url.Values) (BitfinexLendbook, error) {
	if len(symbol) == 6 {
		symbol = symbol[:3]
	}
	path := common.EncodeURLValues(BITFINEX_API_URL+BITFINEX_LENDBOOK+symbol, values)
	response := BitfinexLendbook{}
	err := common.SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return response, err
	}
	return response, nil
}

func (b *Bitfinex) GetOrderbook(symbol string, values url.Values) (BitfinexOrderbook, error) {
	path := common.EncodeURLValues(BITFINEX_API_URL+BITFINEX_ORDERBOOK+symbol, values)
	response := BitfinexOrderbook{}
	err := common.SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return response, err
	}
	return response, nil
}

func (b *Bitfinex) GetTrades(symbol string, values url.Values) ([]BitfinexTradeStructure, error) {
	path := common.EncodeURLValues(BITFINEX_API_URL+BITFINEX_TRADES+symbol, values)
	response := []BitfinexTradeStructure{}
	err := common.SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Bitfinex) GetLends(symbol string, values url.Values) ([]BitfinexLends, error) {
	path := common.EncodeURLValues(BITFINEX_API_URL+BITFINEX_LENDS+symbol, values)
	response := []BitfinexLends{}
	err := common.SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Bitfinex) GetSymbols() ([]string, error) {
	products := []string{}
	err := common.SendHTTPGetRequest(BITFINEX_API_URL+BITFINEX_SYMBOLS, true, &products)
	if err != nil {
		return nil, err
	}
	return products, nil
}

func (b *Bitfinex) GetSymbolsDetails() ([]BitfinexSymbolDetails, error) {
	response := []BitfinexSymbolDetails{}
	err := common.SendHTTPGetRequest(BITFINEX_API_URL+BITFINEX_SYMBOLS_DETAILS, true, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Bitfinex) GetAccountInfo() ([]BitfinexAccountInfo, error) {
	response := []BitfinexAccountInfo{}
	err := b.SendAuthenticatedHTTPRequest("POST", BITFINEX_ACCOUNT_INFO, nil, &response)

	if err != nil {
		return response, err
	}
	return response, nil
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
	if len(b.APIKey) == 0 {
		return errors.New("SendAuthenticatedHTTPRequest: Invalid API key")
	}

	request := make(map[string]interface{})
	request["request"] = fmt.Sprintf("/v%s/%s", BITFINEX_API_VERSION, path)
	request["nonce"] = strconv.FormatInt(time.Now().UnixNano(), 10)

	if params != nil {
		for key, value := range params {
			request[key] = value
		}
	}

	PayloadJson, err := common.JSONEncode(request)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	if b.Verbose {
		log.Printf("Request JSON: %s\n", PayloadJson)
	}

	PayloadBase64 := common.Base64Encode(PayloadJson)
	hmac := common.GetHMAC(common.HASH_SHA512_384, []byte(PayloadBase64), []byte(b.APISecret))
	headers := make(map[string]string)
	headers["X-BFX-APIKEY"] = b.APIKey
	headers["X-BFX-PAYLOAD"] = PayloadBase64
	headers["X-BFX-SIGNATURE"] = common.HexEncodeToString(hmac)

	resp, err := common.SendHTTPRequest(method, BITFINEX_API_URL+path, headers, strings.NewReader(""))
	if err != nil {
		return err
	}

	if strings.Contains(resp, "message") {
		return errors.New("SendAuthenticatedHTTPRequest: " + resp[11:])
	}

	if b.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON Unmarshal response.")
	}

	return nil
}
