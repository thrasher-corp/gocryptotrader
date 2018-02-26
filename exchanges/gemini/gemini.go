package gemini

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	geminiAPIURL        = "https://api.gemini.com"
	geminiSandboxAPIURL = "https://api.sandbox.gemini.com"
	geminiAPIVersion    = "1"

	geminiSymbols            = "symbols"
	geminiTicker             = "pubticker"
	geminiAuction            = "auction"
	geminiAuctionHistory     = "history"
	geminiOrderbook          = "book"
	geminiTrades             = "trades"
	geminiOrders             = "orders"
	geminiOrderNew           = "order/new"
	geminiOrderCancel        = "order/cancel"
	geminiOrderCancelSession = "order/cancel/session"
	geminiOrderCancelAll     = "order/cancel/all"
	geminiOrderStatus        = "order/status"
	geminiMyTrades           = "mytrades"
	geminiBalances           = "balances"
	geminiTradeVolume        = "tradevolume"
	geminiDeposit            = "deposit"
	geminiNewAddress         = "newAddress"
	geminiWithdraw           = "withdraw/"
	geminiHeartbeat          = "heartbeat"

	// rate limits per minute
	geminiPublicRate  = 120
	geminiPrivateRate = 600

	// rates limits per second
	geminiPublicRateSec  = 1
	geminiPrivateRateSec = 5

	// Too many requests returns this
	geminiRateError = "429"

	// Assigned API key roles on creation
	geminiRoleTrader      = "trader"
	geminiRoleFundManager = "fundmanager"
)

var (
	// Session manager
	Session map[int]*Gemini
)

// Gemini is the overarching type across the Gemini package, create multiple
// instances with differing APIkeys for segregation of roles for authenticated
// requests & sessions by appending new sessions to the Session map using
// AddSession, if sandbox test is needed append a new session with with the same
// API keys and change the IsSandbox variable to true.
type Gemini struct {
	exchange.Base
	Role              string
	RequiresHeartBeat bool
}

// AddSession adds a new session to the gemini base
func AddSession(g *Gemini, sessionID int, apiKey, apiSecret, role string, needsHeartbeat, isSandbox bool) error {
	if Session == nil {
		Session = make(map[int]*Gemini)
	}

	_, ok := Session[sessionID]
	if ok {
		return errors.New("sessionID already being used")
	}

	g.APIKey = apiKey
	g.APISecret = apiSecret
	g.Role = role
	g.RequiresHeartBeat = needsHeartbeat
	g.APIUrl = geminiAPIURL

	if isSandbox {
		g.APIUrl = geminiSandboxAPIURL
	}

	Session[sessionID] = g

	return nil
}

// SetDefaults sets package defaults for gemini exchange
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

// Setup sets exchange configuration parameters
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
		if exch.UseSandbox {
			g.APIUrl = geminiSandboxAPIURL
		} else {
			g.APIUrl = geminiAPIURL
		}
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

// GetSymbols returns all available symbols for trading
func (g *Gemini) GetSymbols() ([]string, error) {
	symbols := []string{}
	path := fmt.Sprintf("%s/v%s/%s", g.APIUrl, geminiAPIVersion, geminiSymbols)

	return symbols, common.SendHTTPGetRequest(path, true, g.Verbose, &symbols)
}

// GetTicker returns information about recent trading activity for the symbol
func (g *Gemini) GetTicker(currencyPair string) (Ticker, error) {

	type TickerResponse struct {
		Ask    float64 `json:"ask,string"`
		Bid    float64 `json:"bid,string"`
		Last   float64 `json:"last,string"`
		Volume map[string]interface{}
	}

	ticker := Ticker{}
	resp := TickerResponse{}
	path := fmt.Sprintf("%s/v%s/%s/%s", g.APIUrl, geminiAPIVersion, geminiTicker, currencyPair)

	err := common.SendHTTPGetRequest(path, true, g.Verbose, &resp)
	if err != nil {
		return ticker, err
	}

	ticker.Ask = resp.Ask
	ticker.Bid = resp.Bid
	ticker.Last = resp.Last

	ticker.Volume.Currency, _ = strconv.ParseFloat(resp.Volume[currencyPair[0:3]].(string), 64)

	if common.StringContains(currencyPair, "USD") {
		ticker.Volume.USD, _ = strconv.ParseFloat(resp.Volume["USD"].(string), 64)
	} else {
		ticker.Volume.ETH, _ = strconv.ParseFloat(resp.Volume["ETH"].(string), 64)
		ticker.Volume.BTC, _ = strconv.ParseFloat(resp.Volume["BTC"].(string), 64)
	}

	time, _ := resp.Volume["timestamp"].(float64)
	ticker.Volume.Timestamp = int64(time)

	return ticker, nil
}

// GetOrderbook returns the current order book, as two arrays, one of bids, and
// one of asks
//
// params - limit_bids or limit_asks [OPTIONAL] default 50, 0 returns all Values
// Type is an integer ie "params.Set("limit_asks", 30)"
func (g *Gemini) GetOrderbook(currencyPair string, params url.Values) (Orderbook, error) {
	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s/%s", g.APIUrl, geminiAPIVersion, geminiOrderbook, currencyPair), params)
	orderbook := Orderbook{}

	return orderbook, common.SendHTTPGetRequest(path, true, g.Verbose, &orderbook)
}

// GetTrades eturn the trades that have executed since the specified timestamp.
// Timestamps are either seconds or milliseconds since the epoch (1970-01-01).
//
// currencyPair - example "btcusd"
// params --
// since, timestamp [optional]
// limit_trades	integer	Optional. The maximum number of trades to return.
// include_breaks	boolean	Optional. Whether to display broken trades. False by
// default. Can be '1' or 'true' to activate
func (g *Gemini) GetTrades(currencyPair string, params url.Values) ([]Trade, error) {
	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s/%s", g.APIUrl, geminiAPIVersion, geminiTrades, currencyPair), params)
	trades := []Trade{}

	return trades, common.SendHTTPGetRequest(path, true, g.Verbose, &trades)
}

// GetAuction returns auction information
func (g *Gemini) GetAuction(currencyPair string) (Auction, error) {
	path := fmt.Sprintf("%s/v%s/%s/%s", g.APIUrl, geminiAPIVersion, geminiAuction, currencyPair)
	auction := Auction{}

	return auction, common.SendHTTPGetRequest(path, true, g.Verbose, &auction)
}

// GetAuctionHistory returns the auction events, optionally including
// publications of indicative prices, since the specific timestamp.
//
// currencyPair - example "btcusd"
// params -- [optional]
//          since - [timestamp] Only returns auction events after the specified
// timestamp.
//          limit_auction_results - [integer] The maximum number of auction
// events to return.
//          include_indicative - [bool] Whether to include publication of
// indicative prices and quantities.
func (g *Gemini) GetAuctionHistory(currencyPair string, params url.Values) ([]AuctionHistory, error) {
	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s/%s/%s", g.APIUrl, geminiAPIVersion, geminiAuction, currencyPair, geminiAuctionHistory), params)
	auctionHist := []AuctionHistory{}

	return auctionHist, common.SendHTTPGetRequest(path, true, g.Verbose, &auctionHist)
}

func (g *Gemini) isCorrectSession(role string) error {
	if g.Role != role {
		return errors.New("incorrect role for APIKEY cannot use this function")
	}
	return nil
}

// NewOrder Only limit orders are supported through the API at present.
// returns order ID if successful
func (g *Gemini) NewOrder(symbol string, amount, price float64, side, orderType string) (int64, error) {
	if err := g.isCorrectSession(geminiRoleTrader); err != nil {
		return 0, err
	}

	request := make(map[string]interface{})
	request["symbol"] = symbol
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	request["side"] = side
	request["type"] = orderType

	response := Order{}
	err := g.SendAuthenticatedHTTPRequest("POST", geminiOrderNew, request, &response)
	if err != nil {
		return 0, err
	}
	return response.OrderID, nil
}

// CancelOrder will cancel an order. If the order is already canceled, the
// message will succeed but have no effect.
func (g *Gemini) CancelOrder(OrderID int64) (Order, error) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID

	response := Order{}
	err := g.SendAuthenticatedHTTPRequest("POST", geminiOrderCancel, request, &response)
	if err != nil {
		return Order{}, err
	}
	return response, nil
}

// CancelOrders will cancel all outstanding orders created by all sessions owned
// by this account, including interactive orders placed through the UI. If
// sessions = true will only cancel the order that is called on this session
// asssociated with the APIKEY
func (g *Gemini) CancelOrders(CancelBySession bool) (OrderResult, error) {
	response := OrderResult{}
	path := geminiOrderCancelAll
	if CancelBySession {
		path = geminiOrderCancelSession
	}

	return response, g.SendAuthenticatedHTTPRequest("POST", path, nil, &response)
}

// GetOrderStatus returns the status for an order
func (g *Gemini) GetOrderStatus(orderID int64) (Order, error) {
	request := make(map[string]interface{})
	request["order_id"] = orderID

	response := Order{}

	return response,
		g.SendAuthenticatedHTTPRequest("POST", geminiOrderStatus, request, &response)
}

// GetOrders returns active orders in the market
func (g *Gemini) GetOrders() ([]Order, error) {
	response := []Order{}

	return response,
		g.SendAuthenticatedHTTPRequest("POST", geminiOrders, nil, &response)
}

// GetTradeHistory returns an array of trades that have been on the exchange
//
// currencyPair - example "btcusd"
// timestamp - [optional] Only return trades on or after this timestamp.
func (g *Gemini) GetTradeHistory(currencyPair string, timestamp int64) ([]TradeHistory, error) {
	response := []TradeHistory{}
	request := make(map[string]interface{})
	request["symbol"] = currencyPair

	if timestamp != 0 {
		request["timestamp"] = timestamp
	}

	return response,
		g.SendAuthenticatedHTTPRequest("POST", geminiMyTrades, request, &response)
}

// GetTradeVolume returns a multi-arrayed volume response
func (g *Gemini) GetTradeVolume() ([][]TradeVolume, error) {
	response := [][]TradeVolume{}

	return response,
		g.SendAuthenticatedHTTPRequest("POST", geminiTradeVolume, nil, &response)
}

// GetBalances returns available balances in the supported currencies
func (g *Gemini) GetBalances() ([]Balance, error) {
	response := []Balance{}

	return response,
		g.SendAuthenticatedHTTPRequest("POST", geminiBalances, nil, &response)
}

// GetDepositAddress returns a deposit address
func (g *Gemini) GetDepositAddress(depositAddlabel, currency string) (DepositAddress, error) {
	response := DepositAddress{}

	return response,
		g.SendAuthenticatedHTTPRequest("POST", geminiDeposit+"/"+currency+"/"+geminiNewAddress, nil, &response)
}

// WithdrawCrypto withdraws crypto currency to a whitelisted address
func (g *Gemini) WithdrawCrypto(address, currency string, amount float64) (WithdrawalAddress, error) {
	response := WithdrawalAddress{}
	request := make(map[string]interface{})
	request["address"] = address
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)

	return response,
		g.SendAuthenticatedHTTPRequest("POST", geminiWithdraw+currency, nil, &response)
}

// PostHeartbeat sends a maintenance heartbeat to the exchange for all heartbeat
// maintaned sessions
func (g *Gemini) PostHeartbeat() (string, error) {
	type Response struct {
		Result string `json:"result"`
	}
	response := Response{}

	return response.Result,
		g.SendAuthenticatedHTTPRequest("POST", geminiHeartbeat, nil, &response)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to the
// exchange and returns an error
func (g *Gemini) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	if !g.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, g.Name)
	}

	headers := make(map[string]string)
	request := make(map[string]interface{})
	request["request"] = fmt.Sprintf("/v%s/%s", geminiAPIVersion, path)
	request["nonce"] = g.Nonce.GetValue(g.Name, false)

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

	headers["X-GEMINI-APIKEY"] = g.APIKey
	headers["X-GEMINI-PAYLOAD"] = PayloadBase64
	headers["X-GEMINI-SIGNATURE"] = common.HexEncodeToString(hmac)

	resp, err := common.SendHTTPRequest(method, g.APIUrl+"/v1/"+path, headers, strings.NewReader(""))
	if err != nil {
		return err
	}

	if g.Verbose {
		log.Printf("Received raw: \n%s\n", resp)
	}

	captureErr := ErrorCapture{}
	if err = common.JSONDecode([]byte(resp), &captureErr); err == nil {
		if len(captureErr.Message) != 0 || len(captureErr.Result) != 0 || len(captureErr.Reason) != 0 {
			if captureErr.Result != "ok" {
				return errors.New(captureErr.Message)
			}
		}
	}

	return common.JSONDecode([]byte(resp), &result)
}
