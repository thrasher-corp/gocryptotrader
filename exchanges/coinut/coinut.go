package coinut

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
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
)

// COINUT is the overarching type across the coinut package
type COINUT struct {
	exchange.Base
	WebsocketConn *websocket.Conn
	InstrumentMap map[string]int
}

// SetDefaults sets current default values
func (c *COINUT) SetDefaults() {
	c.Name = "COINUT"
	c.Enabled = false
	c.Verbose = false
	c.TakerFee = 0.1 //spot
	c.MakerFee = 0
	c.Verbose = false
	c.Websocket = false
	c.RESTPollingDelay = 10
	c.RequestCurrencyPairFormat.Delimiter = ""
	c.RequestCurrencyPairFormat.Uppercase = true
	c.ConfigCurrencyPairFormat.Delimiter = ""
	c.ConfigCurrencyPairFormat.Uppercase = true
	c.AssetTypes = []string{ticker.Spot}
}

// Setup sets the current exchange configuration
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
		err := c.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = c.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetInstruments returns instruments
func (c *COINUT) GetInstruments() (Instruments, error) {
	var result Instruments
	params := make(map[string]interface{})
	params["sec_type"] = "SPOT"
	err := c.SendHTTPRequest(coinutInstruments, params, false, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetInstrumentTicker returns a ticker for a specific instrument
func (c *COINUT) GetInstrumentTicker(instrumentID int) (Ticker, error) {
	var result Ticker
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	err := c.SendHTTPRequest(coinutTicker, params, false, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetInstrumentOrderbook returns the orderbooks for a specific instrument
func (c *COINUT) GetInstrumentOrderbook(instrumentID, limit int) (Orderbook, error) {
	var result Orderbook
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	if limit > 0 {
		params["top_n"] = limit
	}
	err := c.SendHTTPRequest(coinutOrderbook, params, false, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetTrades returns trade information
func (c *COINUT) GetTrades(instrumentID int) (Trades, error) {
	var result Trades
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	err := c.SendHTTPRequest(coinutTrades, params, false, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetUserBalance returns the full user balance
func (c *COINUT) GetUserBalance() (UserBalance, error) {
	result := UserBalance{}
	err := c.SendHTTPRequest(coinutBalance, nil, true, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// NewOrder places a new order on the exchange
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

	err := c.SendHTTPRequest(coinutOrder, params, true, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// NewOrders places multiple orders on the exchange
func (c *COINUT) NewOrders(orders []Order) ([]OrdersBase, error) {
	var result OrdersResponse
	params := make(map[string]interface{})
	params["orders"] = orders
	err := c.SendHTTPRequest(coinutOrders, params, true, &result.Data)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetOpenOrders returns a list of open order and relevant information
func (c *COINUT) GetOpenOrders(instrumentID int) ([]OrdersResponse, error) {
	var result []OrdersResponse
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	err := c.SendHTTPRequest(coinutOrdersOpen, params, true, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CancelOrder cancels a specific order and returns if it was actioned
func (c *COINUT) CancelOrder(instrumentID, orderID int) (bool, error) {
	var result GenericResponse
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID
	params["order_id"] = orderID
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
	params["entries"] = orders
	err := c.SendHTTPRequest(coinutOrdersCancel, params, true, &result)
	if err != nil {
		return result, err
	}
	return result, nil
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
	err := c.SendHTTPRequest(coinutTradeHistory, params, true, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetIndexTicker returns the index ticker for an asset
func (c *COINUT) GetIndexTicker(asset string) (IndexTicker, error) {
	var result IndexTicker
	params := make(map[string]interface{})
	params["asset"] = asset
	err := c.SendHTTPRequest(coinutIndexTicker, params, false, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetDerivativeInstruments returns a list of derivative instruments
func (c *COINUT) GetDerivativeInstruments(secType string) (interface{}, error) {
	var result interface{} //to-do
	params := make(map[string]interface{})
	params["sec_type"] = secType
	err := c.SendHTTPRequest(coinutInstruments, params, false, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetOptionChain returns option chain
func (c *COINUT) GetOptionChain(asset, secType string, expiry int64) (OptionChainResponse, error) {
	var result OptionChainResponse
	params := make(map[string]interface{})
	params["asset"] = asset
	params["sec_type"] = secType
	err := c.SendHTTPRequest(coinutOptionChain, params, false, &result)
	if err != nil {
		return result, err
	}
	return result, nil
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
	err := c.SendHTTPRequest(coinutPositionHistory, params, true, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetOpenPositions returns all your current opened positions
func (c *COINUT) GetOpenPositions(instrumentID int) ([]OpenPosition, error) {
	type Response struct {
		Positions []OpenPosition `json:"positions"`
	}
	var result Response
	params := make(map[string]interface{})
	params["inst_id"] = instrumentID

	err := c.SendHTTPRequest(coinutPositionOpen, params, true, &result)
	if err != nil {
		return result.Positions, err
	}
	return result.Positions, nil
}

//to-do: user position update via websocket

// SendHTTPRequest sends an authenticated HTTP request
func (c *COINUT) SendHTTPRequest(apiRequest string, params map[string]interface{}, authenticated bool, result interface{}) (err error) {
	if !c.AuthenticatedAPISupport && authenticated {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, c.Name)
	}

	if c.Nonce.Get() == 0 {
		c.Nonce.Set(time.Now().Unix())
	} else {
		c.Nonce.Inc()
	}

	if params == nil {
		params = map[string]interface{}{}
	}
	params["nonce"] = c.Nonce.Get()
	params["request"] = apiRequest

	payload, err := common.JSONEncode(params)
	if err != nil {
		return errors.New("SenddHTTPRequest: Unable to JSON request")
	}

	if c.Verbose {
		log.Printf("Request JSON: %s\n", payload)
	}

	headers := make(map[string]string)
	if authenticated {
		headers["X-USER"] = c.ClientID
		hmac := common.GetHMAC(common.HashSHA256, []byte(payload), []byte(c.APIKey))
		headers["X-SIGNATURE"] = common.HexEncodeToString(hmac)
	}
	headers["Content-Type"] = "application/json"

	resp, err := common.SendHTTPRequest("POST", coinutAPIURL, headers, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	if c.Verbose {
		log.Printf("Received raw: \n%s", resp)
	}

	genResp := GenericResponse{}
	err = common.JSONDecode([]byte(resp), &genResp)
	if err != nil {
		return errors.New("unable to JSON Unmarshal generic response")
	}

	if genResp.Status[0] != "OK" {
		return errors.New("status is not OK")
	}

	err = common.JSONDecode([]byte(resp), &result)
	if err != nil {
		return errors.New("unable to JSON Unmarshal response")
	}

	return nil
}
