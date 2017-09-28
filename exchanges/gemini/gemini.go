package gemini

import (
	"errors"
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
	GEMINI_API_URL     = "https://api.gemini.com"
	GEMINI_API_VERSION = "1"

	GEMINI_SYMBOLS              = "symbols"
	GEMINI_TICKER               = "pubticker"
	GEMINI_AUCTION              = "auction"
	GEMINI_AUCTION_HISTORY      = "history"
	GEMINI_ORDERBOOK            = "book"
	GEMINI_TRADES               = "trades"
	GEMINI_ORDERS               = "orders"
	GEMINI_ORDER_NEW            = "order/new"
	GEMINI_ORDER_CANCEL         = "order/cancel"
	GEMINI_ORDER_CANCEL_SESSION = "order/cancel/session"
	GEMINI_ORDER_CANCEL_ALL     = "order/cancel/all"
	GEMINI_ORDER_STATUS         = "order/status"
	GEMINI_MYTRADES             = "mytrades"
	GEMINI_BALANCES             = "balances"
	GEMINI_HEARTBEAT            = "heartbeat"
)

type Gemini struct {
	exchange.Base
}

func (g *Gemini) SetDefaults() {
	g.Name = "Gemini"
	g.Enabled = false
	g.Verbose = false
	g.Websocket = false
	g.RESTPollingDelay = 10
	g.RequestCurrencyPairFormat.Delimiter = ""
	g.RequestCurrencyPairFormat.Uppercase = true
	g.ConfigCurrencyPairFormat.Delimiter = ""
	g.ConfigCurrencyPairFormat.Uppercase = true
	g.AssetTypes = []string{ticker.Spot}
}

func (g *Gemini) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		g.SetEnabled(false)
	} else {
		g.Enabled = true
		g.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		g.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		g.RESTPollingDelay = exch.RESTPollingDelay
		g.Verbose = exch.Verbose
		g.Websocket = exch.Websocket
		g.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		g.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		g.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := g.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = g.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (g *Gemini) GetTicker(currency string) (GeminiTicker, error) {

	type TickerResponse struct {
		Ask    float64 `json:"ask,string"`
		Bid    float64 `json:"bid,string"`
		Last   float64 `json:"last,string"`
		Volume map[string]interface{}
	}

	ticker := GeminiTicker{}
	resp := TickerResponse{}
	path := fmt.Sprintf("%s/v%s/%s/%s", GEMINI_API_URL, GEMINI_API_VERSION, GEMINI_TICKER, currency)

	err := common.SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return ticker, err
	}

	ticker.Ask = resp.Ask
	ticker.Bid = resp.Bid
	ticker.Last = resp.Last

	ticker.Volume.Currency, _ = strconv.ParseFloat(resp.Volume[currency[0:3]].(string), 64)
	ticker.Volume.USD, _ = strconv.ParseFloat(resp.Volume["USD"].(string), 64)

	time, _ := resp.Volume["timestamp"].(float64)
	ticker.Volume.Timestamp = int64(time)

	return ticker, nil
}

func (g *Gemini) GetSymbols() ([]string, error) {
	symbols := []string{}
	path := fmt.Sprintf("%s/v%s/%s", GEMINI_API_URL, GEMINI_API_VERSION, GEMINI_SYMBOLS)
	err := common.SendHTTPGetRequest(path, true, &symbols)
	if err != nil {
		return nil, err
	}
	return symbols, nil
}

func (g *Gemini) GetAuction(currency string) (GeminiAuction, error) {
	path := fmt.Sprintf("%s/v%s/%s/%s", GEMINI_API_URL, GEMINI_API_VERSION, GEMINI_AUCTION, currency)
	auction := GeminiAuction{}
	err := common.SendHTTPGetRequest(path, true, &auction)
	if err != nil {
		return auction, err
	}
	return auction, nil
}

func (g *Gemini) GetAuctionHistory(currency string, params url.Values) ([]GeminiAuctionHistory, error) {
	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s/%s/%s", GEMINI_API_URL, GEMINI_API_VERSION, GEMINI_AUCTION, currency, GEMINI_AUCTION_HISTORY), params)
	auctionHist := []GeminiAuctionHistory{}
	err := common.SendHTTPGetRequest(path, true, &auctionHist)
	if err != nil {
		return nil, err
	}
	return auctionHist, nil
}

func (g *Gemini) GetOrderbook(currency string, params url.Values) (GeminiOrderbook, error) {
	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s/%s", GEMINI_API_URL, GEMINI_API_VERSION, GEMINI_ORDERBOOK, currency), params)
	orderbook := GeminiOrderbook{}
	err := common.SendHTTPGetRequest(path, true, &orderbook)
	if err != nil {
		return GeminiOrderbook{}, err
	}

	return orderbook, nil
}

func (g *Gemini) GetTrades(currency string, params url.Values) ([]GeminiTrade, error) {
	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s/%s", GEMINI_API_URL, GEMINI_API_VERSION, GEMINI_TRADES, currency), params)
	trades := []GeminiTrade{}
	err := common.SendHTTPGetRequest(path, true, &trades)
	if err != nil {
		return []GeminiTrade{}, err
	}

	return trades, nil
}

func (g *Gemini) NewOrder(symbol string, amount, price float64, side, orderType string) (int64, error) {
	request := make(map[string]interface{})
	request["symbol"] = symbol
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	request["side"] = side
	request["type"] = orderType

	response := GeminiOrder{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_ORDER_NEW, request, &response)
	if err != nil {
		return 0, err
	}
	return response.OrderID, nil
}

func (g *Gemini) CancelOrder(OrderID int64) (GeminiOrder, error) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID

	response := GeminiOrder{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_ORDER_CANCEL, request, &response)
	if err != nil {
		return GeminiOrder{}, err
	}
	return response, nil
}

func (g *Gemini) CancelOrders(sessions bool) ([]GeminiOrderResult, error) {
	response := []GeminiOrderResult{}
	path := GEMINI_ORDER_CANCEL_ALL
	if sessions {
		path = GEMINI_ORDER_CANCEL_SESSION
	}
	err := g.SendAuthenticatedHTTPRequest("POST", path, nil, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (g *Gemini) GetOrderStatus(orderID int64) (GeminiOrder, error) {
	request := make(map[string]interface{})
	request["order_id"] = orderID

	response := GeminiOrder{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_ORDER_STATUS, request, &response)
	if err != nil {
		return GeminiOrder{}, err
	}
	return response, nil
}

func (g *Gemini) GetOrders() ([]GeminiOrder, error) {
	response := []GeminiOrder{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_ORDERS, nil, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (g *Gemini) GetTradeHistory(symbol string, timestamp int64) ([]GeminiTradeHistory, error) {
	request := make(map[string]interface{})
	request["symbol"] = symbol
	request["timestamp"] = timestamp

	response := []GeminiTradeHistory{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_MYTRADES, request, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (g *Gemini) GetBalances() ([]GeminiBalance, error) {
	response := []GeminiBalance{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_BALANCES, nil, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (g *Gemini) PostHeartbeat() (bool, error) {
	type Response struct {
		Result bool `json:"result"`
	}

	response := Response{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_HEARTBEAT, nil, &response)
	if err != nil {
		return false, err
	}

	return response.Result, nil
}

func (g *Gemini) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	if !g.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, g.Name)
	}

	if g.Nonce.Get() == 0 {
		g.Nonce.Set(time.Now().UnixNano())
	} else {
		g.Nonce.Inc()
	}

	request := make(map[string]interface{})
	request["request"] = fmt.Sprintf("/v%s/%s", GEMINI_API_VERSION, path)
	request["nonce"] = g.Nonce.Get()

	if params != nil {
		for key, value := range params {
			request[key] = value
		}
	}

	PayloadJSON, err := common.JSONEncode(request)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	if g.Verbose {
		log.Printf("Request JSON: %s\n", PayloadJSON)
	}

	PayloadBase64 := common.Base64Encode(PayloadJSON)
	hmac := common.GetHMAC(common.HashSHA512_384, []byte(PayloadBase64), []byte(g.APISecret))
	headers := make(map[string]string)
	headers["X-GEMINI-APIKEY"] = g.APIKey
	headers["X-GEMINI-PAYLOAD"] = PayloadBase64
	headers["X-GEMINI-SIGNATURE"] = common.HexEncodeToString(hmac)

	resp, err := common.SendHTTPRequest(method, GEMINI_API_URL+path, headers, strings.NewReader(""))

	if g.Verbose {
		log.Printf("Received raw: \n%s\n", resp)
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("unable to JSON Unmarshal response")
	}

	return nil
}
