package gemini

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
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
	geminiVolume             = "notionalvolume"

	// gemini limit rates
	geminiAuthRate   = 600
	geminiUnauthRate = 120

	// Too many requests returns this
	geminiRateError = "429"

	// Assigned API key roles on creation
	geminiRoleTrader      = "trader"
	geminiRoleFundManager = "fundmanager"
)

// Gemini is the overarching type across the Gemini package, create multiple
// instances with differing APIkeys for segregation of roles for authenticated
// requests & sessions by appending new sessions to the Session map using
// AddSession, if sandbox test is needed append a new session with with the same
// API keys and change the IsSandbox variable to true.
type Gemini struct {
	AuthenticatedWebsocketConn *wshandler.WebsocketConnection
	exchange.Base
	Role              string
	RequiresHeartBeat bool
}

// SetDefaults sets package defaults for gemini exchange
func (g *Gemini) SetDefaults() {
	g.Name = "Gemini"
	g.Enabled = false
	g.Verbose = false
	g.RESTPollingDelay = 10
	g.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithAPIPermission |
		exchange.AutoWithdrawCryptoWithSetup |
		exchange.WithdrawFiatViaWebsiteOnly
	g.RequestCurrencyPairFormat.Delimiter = ""
	g.RequestCurrencyPairFormat.Uppercase = true
	g.ConfigCurrencyPairFormat.Delimiter = ""
	g.ConfigCurrencyPairFormat.Uppercase = true
	g.AssetTypes = []string{ticker.Spot}
	g.SupportsAutoPairUpdating = true
	g.SupportsRESTTickerBatching = false
	g.Requester = request.New(g.Name,
		request.NewRateLimit(time.Second, geminiAuthRate),
		request.NewRateLimit(time.Second, geminiUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	g.APIUrlDefault = geminiAPIURL
	g.APIUrl = g.APIUrlDefault
	g.Websocket = wshandler.New()
	g.Websocket.Functionality = wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported |
		wshandler.WebsocketSequenceNumberSupported
	g.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	g.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	g.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets exchange configuration parameters
func (g *Gemini) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		g.SetEnabled(false)
	} else {
		g.Enabled = true
		g.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		g.AuthenticatedWebsocketAPISupport = exch.AuthenticatedWebsocketAPISupport
		g.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		g.SetHTTPClientTimeout(exch.HTTPTimeout)
		g.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		g.RESTPollingDelay = exch.RESTPollingDelay
		g.Verbose = exch.Verbose
		g.HTTPDebugging = exch.HTTPDebugging
		g.BaseCurrencies = exch.BaseCurrencies
		g.AvailablePairs = exch.AvailablePairs
		g.EnabledPairs = exch.EnabledPairs
		g.WebsocketURL = geminiWebsocketEndpoint
		err := g.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = g.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = g.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = g.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		if exch.UseSandbox {
			g.APIUrl = geminiSandboxAPIURL
			g.WebsocketURL = geminiWebsocketSandboxEndpoint
		}
		err = g.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = g.Websocket.Setup(g.WsConnect,
			nil,
			nil,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			g.WebsocketURL,
			g.WebsocketURL,
			exch.AuthenticatedWebsocketAPISupport)
		if err != nil {
			log.Fatal(err)
		}
		responseCheckTimeout = exch.WebsocketResponseCheckTimeout
		responseMaxLimit = exch.WebsocketResponseMaxLimit
		g.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			true,
			true,
			false,
			false,
			exch.Name)
	}
}

// GetSymbols returns all available symbols for trading
func (g *Gemini) GetSymbols() ([]string, error) {
	var symbols []string
	path := fmt.Sprintf("%s/v%s/%s", g.APIUrl, geminiAPIVersion, geminiSymbols)

	return symbols, g.SendHTTPRequest(path, &symbols)
}

// GetTicker returns information about recent trading activity for the symbol
func (g *Gemini) GetTicker(currencyPair string) (Ticker, error) {

	type TickerResponse struct {
		Ask     float64 `json:"ask,string"`
		Bid     float64 `json:"bid,string"`
		Last    float64 `json:"last,string"`
		Volume  map[string]interface{}
		Message string `json:"message"`
	}

	ticker := Ticker{}
	resp := TickerResponse{}
	path := fmt.Sprintf("%s/v%s/%s/%s", g.APIUrl, geminiAPIVersion, geminiTicker, currencyPair)

	err := g.SendHTTPRequest(path, &resp)
	if err != nil {
		return ticker, err
	}

	if resp.Message != "" {
		return ticker, errors.New(resp.Message)
	}

	ticker.Ask = resp.Ask
	ticker.Bid = resp.Bid
	ticker.Last = resp.Last

	ticker.Volume.Currency, _ = strconv.ParseFloat(resp.Volume[currencyPair[0:3]].(string), 64)

	if common.StringContains(currencyPair, "USD") {
		ticker.Volume.USD, _ = strconv.ParseFloat(resp.Volume["USD"].(string), 64)
	} else {
		if resp.Volume["ETH"] != nil {
			ticker.Volume.ETH, _ = strconv.ParseFloat(resp.Volume["ETH"].(string), 64)
		}

		if resp.Volume["BTC"] != nil {
			ticker.Volume.BTC, _ = strconv.ParseFloat(resp.Volume["BTC"].(string), 64)
		}
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
	path := common.EncodeURLValues(
		fmt.Sprintf("%s/v%s/%s/%s",
			g.APIUrl,
			geminiAPIVersion,
			geminiOrderbook,
			currencyPair),
		params)

	var orderbook Orderbook
	return orderbook, g.SendHTTPRequest(path, &orderbook)
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
	var trades []Trade

	return trades, g.SendHTTPRequest(path, &trades)
}

// GetAuction returns auction information
func (g *Gemini) GetAuction(currencyPair string) (Auction, error) {
	path := fmt.Sprintf("%s/v%s/%s/%s", g.APIUrl, geminiAPIVersion, geminiAuction, currencyPair)
	auction := Auction{}

	return auction, g.SendHTTPRequest(path, &auction)
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
	var auctionHist []AuctionHistory

	return auctionHist, g.SendHTTPRequest(path, &auctionHist)
}

// NewOrder Only limit orders are supported through the API at present.
// returns order ID if successful
func (g *Gemini) NewOrder(symbol string, amount, price float64, side, orderType string) (int64, error) {
	req := make(map[string]interface{})
	req["symbol"] = symbol
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	req["side"] = side
	req["type"] = orderType

	response := Order{}
	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiOrderNew, req, &response)
	if err != nil {
		return 0, err
	}
	return response.OrderID, nil
}

// CancelExistingOrder will cancel an order. If the order is already canceled, the
// message will succeed but have no effect.
func (g *Gemini) CancelExistingOrder(orderID int64) (Order, error) {
	req := make(map[string]interface{})
	req["order_id"] = orderID

	response := Order{}
	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiOrderCancel, req, &response)
	if err != nil {
		return Order{}, err
	}
	if response.Message != "" {
		return response, errors.New(response.Message)
	}

	return response, nil
}

// CancelExistingOrders will cancel all outstanding orders created by all
// sessions owned by this account, including interactive orders placed through
// the UI. If sessions = true will only cancel the order that is called on this
// session asssociated with the APIKEY
func (g *Gemini) CancelExistingOrders(cancelBySession bool) (OrderResult, error) {
	path := geminiOrderCancelAll
	if cancelBySession {
		path = geminiOrderCancelSession
	}

	var response OrderResult
	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, path, nil, &response)
	if err != nil {
		return response, err
	}
	if response.Message != "" {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// GetOrderStatus returns the status for an order
func (g *Gemini) GetOrderStatus(orderID int64) (Order, error) {
	req := make(map[string]interface{})
	req["order_id"] = orderID

	response := Order{}

	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiOrderStatus, req, &response)
	if err != nil {
		return response, err
	}

	if response.Message != "" {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// GetOrders returns active orders in the market
func (g *Gemini) GetOrders() ([]Order, error) {
	var response interface{}

	type orders struct {
		orders []Order
	}

	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiOrders, nil, &response)
	if err != nil {
		return nil, err
	}

	switch r := response.(type) {
	case orders:
		return r.orders, nil
	default:
		return []Order{}, nil
	}
}

// GetTradeHistory returns an array of trades that have been on the exchange
//
// currencyPair - example "btcusd"
// timestamp - [optional] Only return trades on or after this timestamp.
func (g *Gemini) GetTradeHistory(currencyPair string, timestamp int64) ([]TradeHistory, error) {
	var response []TradeHistory
	req := make(map[string]interface{})
	req["symbol"] = currencyPair

	if timestamp > 0 {
		req["timestamp"] = timestamp
	}

	return response,
		g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiMyTrades, req, &response)
}

// GetNotionalVolume returns  the volume in price currency that has been traded across all pairs over a period of 30 days
func (g *Gemini) GetNotionalVolume() (NotionalVolume, error) {
	response := NotionalVolume{}

	return response,
		g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiVolume, nil, &response)
}

// GetTradeVolume returns a multi-arrayed volume response
func (g *Gemini) GetTradeVolume() ([][]TradeVolume, error) {
	var response [][]TradeVolume

	return response,
		g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiTradeVolume, nil, &response)
}

// GetBalances returns available balances in the supported currencies
func (g *Gemini) GetBalances() ([]Balance, error) {
	var response []Balance

	return response,
		g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiBalances, nil, &response)
}

// GetCryptoDepositAddress returns a deposit address
func (g *Gemini) GetCryptoDepositAddress(depositAddlabel, currency string) (DepositAddress, error) {
	response := DepositAddress{}
	req := make(map[string]interface{})

	if len(depositAddlabel) > 0 {
		req["label"] = depositAddlabel
	}

	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiDeposit+"/"+currency+"/"+geminiNewAddress, req, &response)
	if err != nil {
		return response, err
	}
	if response.Message != "" {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// WithdrawCrypto withdraws crypto currency to a whitelisted address
func (g *Gemini) WithdrawCrypto(address, currency string, amount float64) (WithdrawalAddress, error) {
	response := WithdrawalAddress{}
	req := make(map[string]interface{})
	req["address"] = address
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)

	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiWithdraw+common.StringToLower(currency), req, &response)
	if err != nil {
		return response, err
	}
	if response.Message != "" {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// PostHeartbeat sends a maintenance heartbeat to the exchange for all heartbeat
// maintaned sessions
func (g *Gemini) PostHeartbeat() (string, error) {
	type Response struct {
		Result  string `json:"result"`
		Message string `json:"message"`
	}
	response := Response{}

	err := g.SendAuthenticatedHTTPRequest(http.MethodPost, geminiHeartbeat, nil, &response)
	if err != nil {
		return response.Result, err
	}
	if response.Message != "" {
		return response.Result, errors.New(response.Message)
	}
	return response.Result, nil
}

// SendHTTPRequest sends an unauthenticated request
func (g *Gemini) SendHTTPRequest(path string, result interface{}) error {
	return g.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		result,
		false,
		false,
		g.Verbose,
		g.HTTPDebugging,
		g.HTTPRecording)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to the
// exchange and returns an error
func (g *Gemini) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	if !g.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, g.Name)
	}

	req := make(map[string]interface{})
	req["request"] = fmt.Sprintf("/v%s/%s", geminiAPIVersion, path)
	req["nonce"] = g.Requester.GetNonce(true).String()

	for key, value := range params {
		req[key] = value
	}

	PayloadJSON, err := common.JSONEncode(req)
	if err != nil {
		return errors.New("sendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	if g.Verbose {
		log.Debugf("Request JSON: %s", PayloadJSON)
	}

	PayloadBase64 := common.Base64Encode(PayloadJSON)
	hmac := common.GetHMAC(common.HashSHA512_384, []byte(PayloadBase64), []byte(g.APISecret))

	headers := make(map[string]string)
	headers["Content-Length"] = "0"
	headers["Content-Type"] = "text/plain"
	headers["X-GEMINI-APIKEY"] = g.APIKey
	headers["X-GEMINI-PAYLOAD"] = PayloadBase64
	headers["X-GEMINI-SIGNATURE"] = common.HexEncodeToString(hmac)
	headers["Cache-Control"] = "no-cache"

	return g.SendPayload(method,
		g.APIUrl+"/v1/"+path,
		headers,
		nil,
		result,
		true,
		false,
		g.Verbose,
		g.HTTPDebugging,
		g.HTTPRecording)
}

// GetFee returns an estimate of fee based on type of transaction
func (g *Gemini) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		notionVolume, err := g.GetNotionalVolume()
		if err != nil {
			return 0, err
		}
		fee = calculateTradingFee(&notionVolume, feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		// TODO: no free transactions after 10; Need database to know how many trades have been done
		// Could do via trade history, but would require analysis of response and dates to determine level of fee
	case exchange.InternationalBankWithdrawalFee:
		fee = 0
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.01 * price * amount
}

func calculateTradingFee(notionVolume *NotionalVolume, purchasePrice, amount float64, isMaker bool) float64 {
	var volumeFee float64
	if isMaker {
		volumeFee = (float64(notionVolume.APIMakerFeeBPS) / 10000)
	} else {
		volumeFee = (float64(notionVolume.APITakerFeeBPS) / 10000)
	}

	return volumeFee * amount * purchasePrice
}
