package coinut

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	coinutAPIURL          = "https://api.coinut.com"
	coinutAPIVersion      = "1"
	coinutInstruments     = "inst_list"
	coinutTicker          = "inst_tick"
	coinutOrderbook       = "inst_order_book"
	coinutTrades          = "inst_trade"
	coinutBalance         = "user_balance"
	coinutOrder           = "new_order"
	coinutOrders          = "new_orders"
	coinutOrdersOpen      = "user_open_orders"
	coinutOrderCancel     = "cancel_order"
	coinutOrdersCancel    = "cancel_orders"
	coinutTradeHistory    = "trade_history"
	coinutIndexTicker     = "index_tick"
	coinutOptionChain     = "option_chain"
	coinutPositionHistory = "position_history"
	coinutPositionOpen    = "user_open_positions"

	coinutAuthRate   = 0
	coinutUnauthRate = 0

	coinutStatusOK = "OK"
)

// COINUT is the overarching type across the coinut package
type COINUT struct {
	exchange.Base
	WebsocketConn *wshandler.WebsocketConnection
	InstrumentMap map[string]int
}

// SetDefaults sets current default values
func (c *COINUT) SetDefaults() {
	c.Name = "COINUT"
	c.Enabled = false
	c.Verbose = false
	c.TakerFee = 0.1 // spot
	c.MakerFee = 0
	c.Verbose = false
	c.RESTPollingDelay = 10
	c.APIWithdrawPermissions = exchange.WithdrawCryptoViaWebsiteOnly |
		exchange.WithdrawFiatViaWebsiteOnly
	c.RequestCurrencyPairFormat.Delimiter = ""
	c.RequestCurrencyPairFormat.Uppercase = true
	c.ConfigCurrencyPairFormat.Delimiter = ""
	c.ConfigCurrencyPairFormat.Uppercase = true
	c.AssetTypes = []string{ticker.Spot}
	c.SupportsAutoPairUpdating = true
	c.SupportsRESTTickerBatching = false
	c.Requester = request.New(c.Name,
		request.NewRateLimit(time.Second, coinutAuthRate),
		request.NewRateLimit(time.Second, coinutUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	c.APIUrlDefault = coinutAPIURL
	c.APIUrl = c.APIUrlDefault
	c.Websocket = wshandler.New()
	c.Websocket.Functionality = wshandler.WebsocketTickerSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported |
		wshandler.WebsocketSubmitOrderSupported |
		wshandler.WebsocketCancelOrderSupported |
		wshandler.WebsocketMessageCorrelationSupported
	c.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	c.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	c.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets the current exchange configuration
func (c *COINUT) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		c.SetEnabled(false)
	} else {
		c.Enabled = true
		c.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		c.AuthenticatedWebsocketAPISupport = exch.AuthenticatedWebsocketAPISupport
		c.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, false)
		c.SetHTTPClientTimeout(exch.HTTPTimeout)
		c.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		c.RESTPollingDelay = exch.RESTPollingDelay
		c.Verbose = exch.Verbose
		c.HTTPDebugging = exch.HTTPDebugging
		c.Websocket.SetWsStatusAndConnection(exch.Websocket)
		c.BaseCurrencies = exch.BaseCurrencies
		c.AvailablePairs = exch.AvailablePairs
		c.EnabledPairs = exch.EnabledPairs
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
		err = c.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = c.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = c.Websocket.Setup(c.WsConnect,
			c.Subscribe,
			c.Unsubscribe,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			coinutWebsocketURL,
			exch.WebsocketURL,
			exch.AuthenticatedWebsocketAPISupport)
		if err != nil {
			log.Fatal(err)
		}
		c.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         c.Name,
			URL:                  c.Websocket.GetWebsocketURL(),
			ProxyURL:             c.Websocket.GetProxyAddress(),
			Verbose:              c.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
			RateLimit:            coinutWebsocketRateLimit,
		}
		c.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			true,
			true,
			true,
			false,
			exch.Name)
	}
}

// GetInstruments returns instruments
func (c *COINUT) GetInstruments() (Instruments, error) {
	var result Instruments
	params := make(map[string]interface{})
	params["sec_type"] = orderbook.Spot

	return result, c.SendHTTPRequest(coinutInstruments, params, false, &result)
}

// GetInstrumentTicker returns a ticker for a specific instrument
func (c *COINUT) GetInstrumentTicker(instrumentID int) (Ticker, error) {
	var result Ticker
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID

	return result, c.SendHTTPRequest(coinutTicker, params, false, &result)
}

// GetInstrumentOrderbook returns the orderbooks for a specific instrument
func (c *COINUT) GetInstrumentOrderbook(instrumentID, limit int) (Orderbook, error) {
	var result Orderbook
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	if limit > 0 {
		params["top_n"] = limit
	}

	return result, c.SendHTTPRequest(coinutOrderbook, params, false, &result)
}

// GetTrades returns trade information
func (c *COINUT) GetTrades(instrumentID int) (Trades, error) {
	var result Trades
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID

	return result, c.SendHTTPRequest(coinutTrades, params, false, &result)
}

// GetUserBalance returns the full user balance
func (c *COINUT) GetUserBalance() (UserBalance, error) {
	result := UserBalance{}

	return result, c.SendHTTPRequest(coinutBalance, nil, true, &result)
}

// NewOrder places a new order on the exchange
func (c *COINUT) NewOrder(instrumentID int, quantity, price float64, buy bool, orderID uint32) (interface{}, error) {
	var result interface{}
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	if price > 0 {
		params["price"] = fmt.Sprintf("%v", price)
	}
	params["qty"] = fmt.Sprintf("%v", quantity)
	params["side"] = "BUY"
	if !buy {
		params["side"] = "SELL"
	}
	params["client_ord_id"] = orderID

	err := c.SendHTTPRequest(coinutOrder, params, true, &result)
	if _, ok := result.(OrderRejectResponse); ok {
		return result.(OrderRejectResponse), err
	}
	if _, ok := result.(OrderFilledResponse); ok {
		return result.(OrderFilledResponse), err
	}
	if _, ok := result.(OrdersBase); ok {
		return result.(OrdersBase), err
	}
	return result, err
}

// NewOrders places multiple orders on the exchange
func (c *COINUT) NewOrders(orders []Order) ([]OrdersBase, error) {
	var result OrdersResponse
	params := make(map[string]interface{})
	params["orders"] = orders

	return result.Data, c.SendHTTPRequest(coinutOrders, params, true, &result.Data)
}

// GetOpenOrders returns a list of open order and relevant information
func (c *COINUT) GetOpenOrders(instrumentID int) (GetOpenOrdersResponse, error) {
	var result GetOpenOrdersResponse
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID

	return result, c.SendHTTPRequest(coinutOrdersOpen, params, true, &result)
}

// CancelExistingOrder cancels a specific order and returns if it was actioned
func (c *COINUT) CancelExistingOrder(instrumentID, orderID int) (bool, error) {
	var result GenericResponse
	params := make(map[string]interface{})
	type Request struct {
		InstrumentID int `json:"inst_id"`
		OrderID      int `json:"order_id"`
	}

	var entry = Request{
		InstrumentID: instrumentID,
		OrderID:      orderID,
	}

	entries := []Request{entry}
	params["entries"] = entries

	err := c.SendHTTPRequest(coinutOrdersCancel, params, true, &result)
	if err != nil {
		return false, err
	}
	return true, nil
}

// CancelOrders cancels multiple orders
func (c *COINUT) CancelOrders(orders []CancelOrders) (CancelOrdersResponse, error) {
	var result CancelOrdersResponse
	params := make(map[string]interface{})
	type Request struct {
		InstrumentID int `json:"inst_id"`
		OrderID      int `json:"order_id"`
	}

	var entries []CancelOrders
	entries = append(entries, orders...)
	params["entries"] = entries

	return result, c.SendHTTPRequest(coinutOrdersCancel, params, true, &result)
}

// GetTradeHistory returns trade history for a specific instrument.
func (c *COINUT) GetTradeHistory(instrumentID, start, limit int) (TradeHistory, error) {
	var result TradeHistory
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	if start >= 0 && start <= 100 {
		params["start"] = start
	}
	if limit >= 0 && start <= 100 {
		params["limit"] = limit
	}

	return result, c.SendHTTPRequest(coinutTradeHistory, params, true, &result)
}

// GetIndexTicker returns the index ticker for an asset
func (c *COINUT) GetIndexTicker(asset string) (IndexTicker, error) {
	var result IndexTicker
	params := make(map[string]interface{})
	params["asset"] = asset

	return result, c.SendHTTPRequest(coinutIndexTicker, params, false, &result)
}

// GetDerivativeInstruments returns a list of derivative instruments
func (c *COINUT) GetDerivativeInstruments(secType string) (interface{}, error) {
	var result interface{} // to-do
	params := make(map[string]interface{})
	params["sec_type"] = secType

	return result, c.SendHTTPRequest(coinutInstruments, params, false, &result)
}

// GetOptionChain returns option chain
func (c *COINUT) GetOptionChain(asset, secType string) (OptionChainResponse, error) {
	var result OptionChainResponse
	params := make(map[string]interface{})
	params["asset"] = asset
	params["sec_type"] = secType

	return result, c.SendHTTPRequest(coinutOptionChain, params, false, &result)
}

// GetPositionHistory returns position history
func (c *COINUT) GetPositionHistory(secType string, start, limit int) (PositionHistory, error) {
	var result PositionHistory
	params := make(map[string]interface{})
	params["sec_type"] = secType
	if start >= 0 {
		params["start"] = start
	}
	if limit >= 0 {
		params["limit"] = limit
	}

	return result, c.SendHTTPRequest(coinutPositionHistory, params, true, &result)
}

// GetOpenPositions returns all your current opened positions
func (c *COINUT) GetOpenPositions(instrumentID int) ([]OpenPosition, error) {
	type Response struct {
		Positions []OpenPosition `json:"positions"`
	}
	var result Response
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID

	return result.Positions,
		c.SendHTTPRequest(coinutPositionOpen, params, true, &result)
}

// to-do: user position update via websocket

// SendHTTPRequest sends either an authenticated or unauthenticated HTTP request
func (c *COINUT) SendHTTPRequest(apiRequest string, params map[string]interface{}, authenticated bool, result interface{}) (err error) {
	if !c.AuthenticatedAPISupport && authenticated {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, c.Name)
	}

	n := c.Requester.GetNonce(false)

	if params == nil {
		params = map[string]interface{}{}
	}
	params["nonce"] = n
	params["request"] = apiRequest

	payload, err := common.JSONEncode(params)
	if err != nil {
		return errors.New("sendHTTPRequest: Unable to JSON request")
	}

	if c.Verbose {
		log.Debugf("Request JSON: %s", payload)
	}

	headers := make(map[string]string)
	if authenticated {
		headers["X-USER"] = c.ClientID
		hmac := common.GetHMAC(common.HashSHA256, payload, []byte(c.APIKey))
		headers["X-SIGNATURE"] = common.HexEncodeToString(hmac)
	}
	headers["Content-Type"] = "application/json"

	var rawMsg json.RawMessage
	err = c.SendPayload(http.MethodPost,
		c.APIUrl,
		headers,
		bytes.NewBuffer(payload),
		&rawMsg,
		authenticated,
		true,
		c.Verbose,
		c.HTTPDebugging,
		c.HTTPRecording)
	if err != nil {
		return err
	}

	var genResp GenericResponse
	err = common.JSONDecode(rawMsg, &genResp)
	if err != nil {
		return err
	}

	if genResp.Status[0] != coinutStatusOK {
		return fmt.Errorf("%s SendHTTPRequest error: %s", c.Name,
			genResp.Status[0])
	}

	return common.JSONDecode(rawMsg, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (c *COINUT) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = c.calculateTradingFee(feeBuilder.Pair.Base,
			feeBuilder.Pair.Quote,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency,
			feeBuilder.Amount)
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.FiatCurrency,
			feeBuilder.Amount)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.Pair, feeBuilder.PurchasePrice, feeBuilder.Amount)
	}

	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(c currency.Pair, price, amount float64) float64 {
	if c.IsCryptoFiatPair() {
		return 0.0035 * price * amount
	}
	return 0.002 * price * amount
}

func (c *COINUT) calculateTradingFee(base, quote currency.Code, purchasePrice, amount float64, isMaker bool) float64 {
	var fee float64

	switch {
	case isMaker:
		fee = 0
	case currency.NewPair(base, quote).IsCryptoFiatPair():
		fee = 0.002
	default:
		fee = 0.001
	}

	return fee * amount * purchasePrice
}

func getInternationalBankWithdrawalFee(c currency.Code, amount float64) float64 {
	var fee float64

	switch c {
	case currency.USD:
		if amount*0.001 < 10 {
			fee = 10
		} else {
			fee = amount * 0.001
		}
	case currency.CAD:
		if amount*0.005 < 10 {
			fee = 2
		} else {
			fee = amount * 0.005
		}
	case currency.SGD:
		if amount*0.001 < 10 {
			fee = 10
		} else {
			fee = amount * 0.001
		}
	}

	return fee
}

func getInternationalBankDepositFee(c currency.Code, amount float64) float64 {
	var fee float64

	if c == currency.USD {
		if amount*0.001 < 10 {
			fee = 10
		} else {
			fee = amount * 0.001
		}
	} else if c == currency.CAD {
		if amount*0.005 < 10 {
			fee = 2
		} else {
			fee = amount * 0.005
		}
	}

	return fee
}
