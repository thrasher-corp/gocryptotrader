package gdax

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
)

const (
	GDAX_API_URL     = "https://api.gdax.com/"
	GDAX_API_VERISON = "0"
	GDAX_PRODUCTS    = "products"
	GDAX_ORDERBOOK   = "book"
	GDAX_TICKER      = "ticker"
	GDAX_TRADES      = "trades"
	GDAX_HISTORY     = "candles"
	GDAX_STATS       = "stats"
	GDAX_CURRENCIES  = "currencies"
	GDAX_ACCOUNTS    = "accounts"
	GDAX_LEDGER      = "ledger"
	GDAX_HOLDS       = "holds"
	GDAX_ORDERS      = "orders"
	GDAX_FILLS       = "fills"
	GDAX_TRANSFERS   = "transfers"
	GDAX_REPORTS     = "reports"
)

type GDAX struct {
	exchange.ExchangeBase
}

func (g *GDAX) SetDefaults() {
	g.Name = "GDAX"
	g.Enabled = false
	g.Verbose = false
	g.TakerFee = 0.25
	g.MakerFee = 0
	g.Verbose = false
	g.Websocket = false
	g.RESTPollingDelay = 10
}

func (g *GDAX) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		g.SetEnabled(false)
	} else {
		g.Enabled = true
		g.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		g.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, true)
		g.RESTPollingDelay = exch.RESTPollingDelay
		g.Verbose = exch.Verbose
		g.Websocket = exch.Websocket
		g.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		g.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		g.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

func (g *GDAX) GetFee(maker bool) float64 {
	if maker {
		return g.MakerFee
	} else {
		return g.TakerFee
	}
}

func (g *GDAX) GetProducts() ([]GDAXProduct, error) {
	products := []GDAXProduct{}
	err := common.SendHTTPGetRequest(GDAX_API_URL+GDAX_PRODUCTS, true, &products)

	if err != nil {
		return nil, err
	}

	return products, nil
}

func (g *GDAX) GetOrderbook(symbol string, level int) (interface{}, error) {
	orderbook := GDAXOrderbookResponse{}
	path := ""
	if level > 0 {
		levelStr := strconv.Itoa(level)
		path = fmt.Sprintf("%s/%s/%s?level=%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_ORDERBOOK, levelStr)
	} else {
		path = fmt.Sprintf("%s/%s/%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_ORDERBOOK)
	}

	err := common.SendHTTPGetRequest(path, true, &orderbook)
	if err != nil {
		return nil, err
	}

	if level == 3 {
		ob := GDAXOrderbookL3{}
		ob.Sequence = orderbook.Sequence
		for _, x := range orderbook.Asks {
			price, err := strconv.ParseFloat((x[0].(string)), 64)
			if err != nil {
				continue
			}
			amount, err := strconv.ParseFloat((x[1].(string)), 64)
			if err != nil {
				continue
			}

			order := make([]GDAXOrderL3, 1)
			order[0].Price = price
			order[0].Amount = amount
			order[0].OrderID = x[2].(string)
			ob.Asks = append(ob.Asks, order)
		}
		for _, x := range orderbook.Bids {
			price, err := strconv.ParseFloat((x[0].(string)), 64)
			if err != nil {
				continue
			}
			amount, err := strconv.ParseFloat((x[1].(string)), 64)
			if err != nil {
				continue
			}

			order := make([]GDAXOrderL3, 1)
			order[0].Price = price
			order[0].Amount = amount
			order[0].OrderID = x[2].(string)
			ob.Bids = append(ob.Bids, order)
		}
		return ob, nil
	} else {
		ob := GDAXOrderbookL1L2{}
		ob.Sequence = orderbook.Sequence
		for _, x := range orderbook.Asks {
			price, err := strconv.ParseFloat((x[0].(string)), 64)
			if err != nil {
				continue
			}
			amount, err := strconv.ParseFloat((x[1].(string)), 64)
			if err != nil {
				continue
			}

			order := make([]GDAXOrderL1L2, 1)
			order[0].Price = price
			order[0].Amount = amount
			order[0].NumOrders = x[2].(float64)
			ob.Asks = append(ob.Asks, order)
		}
		for _, x := range orderbook.Bids {
			price, err := strconv.ParseFloat((x[0].(string)), 64)
			if err != nil {
				continue
			}
			amount, err := strconv.ParseFloat((x[1].(string)), 64)
			if err != nil {
				continue
			}

			order := make([]GDAXOrderL1L2, 1)
			order[0].Price = price
			order[0].Amount = amount
			order[0].NumOrders = x[2].(float64)
			ob.Bids = append(ob.Bids, order)
		}
		return ob, nil
	}
}

func (g *GDAX) GetTicker(symbol string) (GDAXTicker, error) {
	ticker := GDAXTicker{}
	path := fmt.Sprintf("%s/%s/%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_TICKER)
	err := common.SendHTTPGetRequest(path, true, &ticker)

	if err != nil {
		return ticker, err
	}
	return ticker, nil
}

func (g *GDAX) GetTrades(symbol string) ([]GDAXTrade, error) {
	trades := []GDAXTrade{}
	path := fmt.Sprintf("%s/%s/%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_TRADES)
	err := common.SendHTTPGetRequest(path, true, &trades)

	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (g *GDAX) GetHistoricRates(symbol string, start, end, granularity int64) ([]GDAXHistory, error) {
	history := []GDAXHistory{}
	values := url.Values{}

	if start > 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if end > 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	if granularity > 0 {
		values.Set("granularity", strconv.FormatInt(granularity, 10))
	}

	path := common.EncodeURLValues(fmt.Sprintf("%s/%s/%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_HISTORY), values)
	err := common.SendHTTPGetRequest(path, true, &history)

	if err != nil {
		return nil, err
	}
	return history, nil
}

func (g *GDAX) GetStats(symbol string) (GDAXStats, error) {
	stats := GDAXStats{}
	path := fmt.Sprintf("%s/%s/%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_STATS)
	err := common.SendHTTPGetRequest(path, true, &stats)

	if err != nil {
		return stats, err
	}
	return stats, nil
}

func (g *GDAX) GetCurrencies() ([]GDAXCurrency, error) {
	currencies := []GDAXCurrency{}
	err := common.SendHTTPGetRequest(GDAX_API_URL+GDAX_CURRENCIES, true, &currencies)

	if err != nil {
		return nil, err
	}
	return currencies, nil
}

func (g *GDAX) GetAccounts() ([]GDAXAccountResponse, error) {
	resp := []GDAXAccountResponse{}
	err := g.SendAuthenticatedHTTPRequest("GET", GDAX_API_URL+GDAX_ACCOUNTS, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (g *GDAX) GetAccount(account string) (GDAXAccountResponse, error) {
	resp := GDAXAccountResponse{}
	path := fmt.Sprintf("%s/%s", GDAX_ACCOUNTS, account)
	err := g.SendAuthenticatedHTTPRequest("GET", GDAX_API_URL+path, nil, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (g *GDAX) GetAccountHistory(accountID string) ([]GDAXAccountLedgerResponse, error) {
	resp := []GDAXAccountLedgerResponse{}
	path := fmt.Sprintf("%s/%s/%s", GDAX_ACCOUNTS, accountID, GDAX_LEDGER)
	err := g.SendAuthenticatedHTTPRequest("GET", GDAX_API_URL+path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (g *GDAX) GetHolds(accountID string) ([]GDAXAccountHolds, error) {
	resp := []GDAXAccountHolds{}
	path := fmt.Sprintf("%s/%s/%s", GDAX_ACCOUNTS, accountID, GDAX_HOLDS)
	err := g.SendAuthenticatedHTTPRequest("GET", GDAX_API_URL+path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (g *GDAX) PlaceOrder(clientRef string, price, amount float64, side string, productID, stp string) (string, error) {
	request := make(map[string]interface{})

	if clientRef != "" {
		request["client_oid"] = clientRef
	}

	request["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	request["size"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["side"] = side
	request["product_id"] = productID

	if stp != "" {
		request["stp"] = stp
	}

	type OrderResponse struct {
		ID string `json:"id"`
	}

	resp := OrderResponse{}
	err := g.SendAuthenticatedHTTPRequest("POST", GDAX_API_URL+GDAX_ORDERS, request, &resp)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (g *GDAX) CancelOrder(orderID string) error {
	path := fmt.Sprintf("%s/%s", GDAX_ORDERS, orderID)
	err := g.SendAuthenticatedHTTPRequest("DELETE", GDAX_API_URL+path, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (g *GDAX) GetOrders(params url.Values) ([]GDAXOrdersResponse, error) {
	path := common.EncodeURLValues(GDAX_API_URL+GDAX_ORDERS, params)
	resp := []GDAXOrdersResponse{}
	err := g.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (g *GDAX) GetOrder(orderID string) (GDAXOrderResponse, error) {
	path := fmt.Sprintf("%s/%s", GDAX_ORDERS, orderID)
	resp := GDAXOrderResponse{}
	err := g.SendAuthenticatedHTTPRequest("GET", GDAX_API_URL+path, nil, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (g *GDAX) GetFills(params url.Values) ([]GDAXFillResponse, error) {
	path := common.EncodeURLValues(GDAX_API_URL+GDAX_FILLS, params)
	resp := []GDAXFillResponse{}
	err := g.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (g *GDAX) Transfer(transferType string, amount float64, accountID string) error {
	request := make(map[string]interface{})
	request["type"] = transferType
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["GDAX_account_id"] = accountID

	err := g.SendAuthenticatedHTTPRequest("POST", GDAX_API_URL+GDAX_TRANSFERS, request, nil)
	if err != nil {
		return err
	}
	return nil
}

func (g *GDAX) GetReport(reportType, startDate, endDate string) (GDAXReportResponse, error) {
	request := make(map[string]interface{})
	request["type"] = reportType
	request["start_date"] = startDate
	request["end_date"] = endDate

	resp := GDAXReportResponse{}
	err := g.SendAuthenticatedHTTPRequest("POST", GDAX_API_URL+GDAX_REPORTS, request, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (g *GDAX) GetReportStatus(reportID string) (GDAXReportResponse, error) {
	path := fmt.Sprintf("%s/%s", GDAX_REPORTS, reportID)
	resp := GDAXReportResponse{}
	err := g.SendAuthenticatedHTTPRequest("POST", GDAX_API_URL+path, nil, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (g *GDAX) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
	payload := []byte("")

	if params != nil {
		payload, err = common.JSONEncode(params)

		if err != nil {
			return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
		}

		if g.Verbose {
			log.Printf("Request JSON: %s\n", payload)
		}
	}

	message := timestamp + method + path + string(payload)
	hmac := common.GetHMAC(common.HASH_SHA256, []byte(message), []byte(g.APISecret))
	headers := make(map[string]string)
	headers["CB-ACCESS-SIGN"] = common.Base64Encode([]byte(hmac))
	headers["CB-ACCESS-TIMESTAMP"] = timestamp
	headers["CB-ACCESS-KEY"] = g.APIKey
	headers["CB-ACCESS-PASSPHRASE"] = g.ClientID
	headers["Content-Type"] = "application/json"

	resp, err := common.SendHTTPRequest(method, GDAX_API_URL+path, headers, bytes.NewBuffer(payload))

	if g.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
