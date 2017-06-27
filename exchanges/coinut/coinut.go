package coinut

import (
	"bytes"
	"errors"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
)

const (
	COINUT_API_URL          = "https://api.coinut.com"
	COINUT_API_VERISON      = "1"
	COINUT_INSTRUMENTS      = "inst_list"
	COINUT_TICKER           = "inst_tick"
	COINUT_ORDERBOOK        = "inst_order_book"
	COINUT_TRADES           = "inst_trade"
	COINUT_BALANCE          = "user_balance"
	COINUT_ORDER            = "new_order"
	COINUT_ORDERS           = "new_orders"
	COINUT_ORDERS_OPEN      = "user_open_orders"
	COINUT_ORDER_CANCEL     = "cancel_order"
	COINUT_ORDERS_CANCEL    = "cancel_orders"
	COINUT_TRADE_HISTORY    = "trade_history"
	COINUT_INDEX_TICKER     = "index_tick"
	COINUT_OPTION_CHAIN     = "option_chain"
	COINUT_POSITION_HISTORY = "position_history"
	COINUT_POSITION_OPEN    = "user_open_positions"
)

type COINUT struct {
	exchange.ExchangeBase
	WebsocketConn *websocket.Conn
	InstrumentMap map[string]int
}

func (c *COINUT) SetDefaults() {
	c.Name = "COINUT"
	c.Enabled = false
	c.Verbose = false
	c.TakerFee = 0.1 //spot
	c.MakerFee = 0
	c.Verbose = false
	c.Websocket = false
	c.RESTPollingDelay = 10
}

func (c *COINUT) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		c.SetEnabled(false)
	} else {
		c.Enabled = true
		c.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		c.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, true)
		c.RESTPollingDelay = exch.RESTPollingDelay
		c.Verbose = exch.Verbose
		c.Websocket = exch.Websocket
		c.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		c.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		c.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

func (c *COINUT) GetInstruments() (CoinutInstruments, error) {
	var result CoinutInstruments
	params := make(map[string]interface{})
	params["sec_type"] = "SPOT"
	err := c.SendAuthenticatedHTTPRequest(COINUT_INSTRUMENTS, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) GetInstrumentTicker(instrumentID int) (CoinutTicker, error) {
	var result CoinutTicker
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	err := c.SendAuthenticatedHTTPRequest(COINUT_TICKER, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) GetInstrumentOrderbook(instrumentID, limit int) (CoinutOrderbook, error) {
	var result CoinutOrderbook
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	if limit > 0 {
		params["top_n"] = limit
	}
	err := c.SendAuthenticatedHTTPRequest(COINUT_ORDERBOOK, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) GetTrades(instrumentID int) (CoinutTrades, error) {
	var result CoinutTrades
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	err := c.SendAuthenticatedHTTPRequest(COINUT_TRADES, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) GetUserBalance() (CoinutUserBalance, error) {
	result := CoinutUserBalance{}
	err := c.SendAuthenticatedHTTPRequest(COINUT_BALANCE, nil, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) NewOrder(instrumentID int, quantity, price float64, buy bool, orderID uint32) (interface{}, error) {
	var result interface{}
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	params["price"] = price
	params["qty"] = quantity
	params["side"] = "BUY"
	if !buy {
		params["side"] = "SELL"
	}
	params["client_ord_id"] = orderID

	err := c.SendAuthenticatedHTTPRequest(COINUT_ORDER, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) NewOrders(orders []CoinutOrder) ([]CoinutOrdersBase, error) {
	var result CoinutOrdersResponse
	params := make(map[string]interface{})
	params["orders"] = orders
	err := c.SendAuthenticatedHTTPRequest(COINUT_ORDERS, params, &result.Data)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *COINUT) GetOpenOrders(instrumentID int) ([]CoinutOrdersResponse, error) {
	var result []CoinutOrdersResponse
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	err := c.SendAuthenticatedHTTPRequest(COINUT_ORDERS_OPEN, params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *COINUT) CancelOrder(instrumentID, orderID int) (bool, error) {
	var result CoinutGenericResponse
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	params["order_id"] = orderID
	err := c.SendAuthenticatedHTTPRequest(COINUT_ORDERS_CANCEL, params, &result)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *COINUT) CancelOrders(orders []CoinutCancelOrders) (CoinutCancelOrdersResponse, error) {
	var result CoinutCancelOrdersResponse
	params := make(map[string]interface{})
	params["entries"] = orders
	err := c.SendAuthenticatedHTTPRequest(COINUT_ORDERS_CANCEL, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) GetTradeHistory(instrumentID, start, limit int) (CoinutTradeHistory, error) {
	var result CoinutTradeHistory
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	if start >= 0 && start <= 100 {
		params["start"] = start
	}
	if limit >= 0 && start <= 100 {
		params["limit"] = limit
	}
	err := c.SendAuthenticatedHTTPRequest(COINUT_TRADE_HISTORY, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) GetIndexTicker(asset string) (CoinutIndexTicker, error) {
	var result CoinutIndexTicker
	params := make(map[string]interface{})
	params["asset"] = asset
	err := c.SendAuthenticatedHTTPRequest(COINUT_INDEX_TICKER, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) GetDerivativeInstruments(secType string) (interface{}, error) {
	var result interface{} //to-do
	params := make(map[string]interface{})
	params["sec_type"] = secType
	err := c.SendAuthenticatedHTTPRequest(COINUT_INSTRUMENTS, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) GetOptionChain(asset, secType string, expiry int64) (CoinutOptionChainResponse, error) {
	var result CoinutOptionChainResponse
	params := make(map[string]interface{})
	params["asset"] = asset
	params["sec_type"] = secType
	err := c.SendAuthenticatedHTTPRequest(COINUT_OPTION_CHAIN, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) GetPositionHistory(secType string, start, limit int) (CoinutPositionHistory, error) {
	var result CoinutPositionHistory
	params := make(map[string]interface{})
	params["sec_type"] = secType
	if start >= 0 {
		params["start"] = start
	}
	if limit >= 0 {
		params["limit"] = limit
	}
	err := c.SendAuthenticatedHTTPRequest(COINUT_POSITION_HISTORY, params, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *COINUT) GetOpenPosition(instrumentID int) ([]CoinutOpenPosition, error) {
	type Response struct {
		Positions []CoinutOpenPosition `json:"positions"`
	}
	var result Response
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID

	err := c.SendAuthenticatedHTTPRequest(COINUT_POSITION_OPEN, params, &result)
	if err != nil {
		return result.Positions, err
	}
	return result.Positions, nil
}

//to-do: user position update via websocket

func (c *COINUT) SendAuthenticatedHTTPRequest(apiRequest string, params map[string]interface{}, result interface{}) (err error) {
	timestamp := time.Now().Unix()
	payload := []byte("")

	if params == nil {
		params = map[string]interface{}{}
	}
	params["nonce"] = timestamp
	params["request"] = apiRequest

	payload, err = common.JSONEncode(params)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	if c.Verbose {
		log.Printf("Request JSON: %s\n", payload)
	}

	hmac := common.GetHMAC(common.HASH_SHA256, []byte(payload), []byte(c.APIKey))
	headers := make(map[string]string)
	headers["X-USER"] = c.ClientID
	headers["X-SIGNATURE"] = common.HexEncodeToString(hmac)
	headers["Content-Type"] = "application/json"

	resp, err := common.SendHTTPRequest("POST", COINUT_API_URL, headers, bytes.NewBuffer(payload))

	if c.Verbose {
		log.Printf("Recieved raw: \n%s", resp)
	}

	genResp := CoinutGenericResponse{}

	err = common.JSONDecode([]byte(resp), &genResp)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal generic response.")
	}

	if genResp.Status[0] != "OK" {
		return errors.New("Status is not OK.")
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
