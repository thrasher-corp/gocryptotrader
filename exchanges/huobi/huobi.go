package huobi

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	huobiAPIURL     = "https://api.huobi.com/apiv2.php"
	huobiAPIVersion = "2"
)

// HUOBI is the overarching type across this package
type HUOBI struct {
	exchange.Base
}

// SetDefaults sets default values for the exchange
func (h *HUOBI) SetDefaults() {
	h.Name = "Huobi"
	h.Enabled = false
	h.Fee = 0
	h.Verbose = false
	h.Websocket = false
	h.RESTPollingDelay = 10
	h.RequestCurrencyPairFormat.Delimiter = ""
	h.RequestCurrencyPairFormat.Uppercase = false
	h.ConfigCurrencyPairFormat.Delimiter = ""
	h.ConfigCurrencyPairFormat.Uppercase = true
	h.AssetTypes = []string{ticker.Spot}
}

// Setup sets user configuration
func (h *HUOBI) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		h.SetEnabled(false)
	} else {
		h.Enabled = true
		h.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		h.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		h.RESTPollingDelay = exch.RESTPollingDelay
		h.Verbose = exch.Verbose
		h.Websocket = exch.Websocket
		h.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		h.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		h.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := h.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = h.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns Huobi fee
func (h *HUOBI) GetFee() float64 {
	return h.Fee
}

// GetTicker returns the Huobi ticker
func (h *HUOBI) GetTicker(symbol string) (Ticker, error) {
	resp := TickerResponse{}
	path := fmt.Sprintf("https://api.huobi.com/staticmarket/ticker_%s_json.js", symbol)

	return resp.Ticker, common.SendHTTPGetRequest(path, true, h.Verbose, &resp)
}

// GetOrderBook returns the Huobi current orderbook for a currency pair
func (h *HUOBI) GetOrderBook(symbol string) (Orderbook, error) {
	path := fmt.Sprintf("https://api.huobi.com/staticmarket/depth_%s_json.js", symbol)
	resp := Orderbook{}

	return resp, common.SendHTTPGetRequest(path, true, h.Verbose, &resp)
}

// GetAccountInfo returns account information
func (h *HUOBI) GetAccountInfo() {
	err := h.SendAuthenticatedRequest("get_account_info", url.Values{})

	if err != nil {
		log.Println(err)
	}
}

// GetOrders returns full list of orders
func (h *HUOBI) GetOrders(coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("get_orders", values)

	if err != nil {
		log.Println(err)
	}
}

// GetOrderInfo returns specific info on an order
func (h *HUOBI) GetOrderInfo(orderID, coinType int) {
	values := url.Values{}
	values.Set("id", strconv.Itoa(orderID))
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("order_info", values)

	if err != nil {
		log.Println(err)
	}
}

// Trade opens a trade on the Huobi exchange
func (h *HUOBI) Trade(orderType string, coinType int, price, amount float64) {
	values := url.Values{}
	if orderType != "buy" {
		orderType = "sell"
	}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	err := h.SendAuthenticatedRequest(orderType, values)

	if err != nil {
		log.Println(err)
	}
}

// MarketTrade initiates a market trade
func (h *HUOBI) MarketTrade(orderType string, coinType int, price, amount float64) {
	values := url.Values{}
	if orderType != "buy_market" {
		orderType = "sell_market"
	}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	err := h.SendAuthenticatedRequest(orderType, values)

	if err != nil {
		log.Println(err)
	}
}

// CancelOrder cancels order by order ID
func (h *HUOBI) CancelOrder(orderID, coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("id", strconv.Itoa(orderID))
	err := h.SendAuthenticatedRequest("cancel_order", values)

	if err != nil {
		log.Println(err)
	}
}

// ModifyOrder modifies an order
func (h *HUOBI) ModifyOrder(orderType string, coinType, orderID int, price, amount float64) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("id", strconv.Itoa(orderID))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	err := h.SendAuthenticatedRequest("modify_order", values)

	if err != nil {
		log.Println(err)
	}
}

// GetNewDealOrders creates a new deal
func (h *HUOBI) GetNewDealOrders(coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("get_new_deal_orders", values)

	if err != nil {
		log.Println(err)
	}
}

// GetOrderIDByTradeID returns ORDERID by Trade ID
func (h *HUOBI) GetOrderIDByTradeID(coinType, orderID int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("trade_id", strconv.Itoa(orderID))
	err := h.SendAuthenticatedRequest("get_order_id_by_trade_id", values)

	if err != nil {
		log.Println(err)
	}
}

// SendAuthenticatedRequest sends an autheticated HTTP request to Huobi
func (h *HUOBI) SendAuthenticatedRequest(method string, v url.Values) error {
	if !h.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, h.Name)
	}

	v.Set("access_key", h.APIKey)
	v.Set("created", strconv.FormatInt(time.Now().Unix(), 10))
	v.Set("method", method)
	hash := common.GetMD5([]byte(v.Encode() + "&secret_key=" + h.APISecret))
	v.Set("sign", common.StringToLower(common.HexEncodeToString(hash)))
	encoded := v.Encode()

	if h.Verbose {
		log.Printf("Sending POST request to %s with params %s\n", huobiAPIURL, encoded)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest("POST", huobiAPIURL, headers, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	if h.Verbose {
		log.Printf("Received raw: %s\n", resp)
	}

	return nil
}
