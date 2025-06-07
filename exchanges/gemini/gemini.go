package gemini

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	geminiAPIURL        = "https://api.gemini.com"
	geminiSandboxAPIURL = "https://api.sandbox.gemini.com"
	geminiAPIVersion    = "1"
	tradeBaseURL        = "https://exchange.gemini.com/trade/"

	geminiSymbols            = "symbols"
	geminiSymbolDetails      = "symbols/details"
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
	geminiTransfers          = "transfers"
)

// Gemini is the overarching type across the Gemini package, create multiple
// instances with differing APIkeys for segregation of roles for authenticated
// requests & sessions by appending new sessions to the Session map using
// AddSession. If sandbox test is needed, append a new session with the same
// API keys and change the IsSandbox variable to true.
type Gemini struct {
	exchange.Base
}

// GetSymbols returns all available symbols for trading
func (g *Gemini) GetSymbols(ctx context.Context) ([]string, error) {
	var symbols []string
	path := fmt.Sprintf("/v%s/%s", geminiAPIVersion, geminiSymbols)
	return symbols, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &symbols)
}

// GetSymbolDetails returns extra symbol details
// use symbol "all" to get everything
func (g *Gemini) GetSymbolDetails(ctx context.Context, symbol string) ([]SymbolDetails, error) {
	if symbol == "all" {
		var details []SymbolDetails
		return details, g.SendHTTPRequest(ctx, exchange.RestSpot, "/v"+geminiAPIVersion+"/"+geminiSymbolDetails+"/"+symbol, &details)
	}
	var details SymbolDetails
	err := g.SendHTTPRequest(ctx, exchange.RestSpot, "/v"+geminiAPIVersion+"/"+geminiSymbolDetails+"/"+symbol, &details)
	if err != nil {
		return nil, err
	}
	return []SymbolDetails{details}, nil
}

// GetTicker returns information about recent trading activity for the symbol
func (g *Gemini) GetTicker(ctx context.Context, currencyPair string) (TickerV2, error) {
	ticker := TickerV2{}
	path := "/v2/ticker/" + currencyPair
	err := g.SendHTTPRequest(ctx, exchange.RestSpot, path, &ticker)
	if err != nil {
		return ticker, err
	}
	if ticker.Result == "error" {
		return ticker, fmt.Errorf("%v %v %v",
			g.Name,
			ticker.Reason,
			ticker.Message)
	}

	return ticker, nil
}

// GetOrderbook returns the current order book, as two arrays, one of bids, and
// one of asks
//
// params - limit_bids or limit_asks [OPTIONAL] default 50, 0 returns all Values
// Type is an integer ie "params.Set("limit_asks", 30)"
func (g *Gemini) GetOrderbook(ctx context.Context, currencyPair string, params url.Values) (*Orderbook, error) {
	path := common.EncodeURLValues(
		fmt.Sprintf("/v%s/%s/%s",
			geminiAPIVersion,
			geminiOrderbook,
			currencyPair),
		params)

	var orderbook Orderbook
	return &orderbook, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &orderbook)
}

// GetTrades return the trades that have executed since the specified timestamp.
// Timestamps are either seconds or milliseconds since the epoch (1970-01-01).
//
// currencyPair - example "btcusd"
// params --
// since, timestamp [optional]
// limit_trades	integer	Optional. The maximum number of trades to return.
// include_breaks	boolean	Optional. Whether to display broken trades. False by
// default. Can be '1' or 'true' to activate
func (g *Gemini) GetTrades(ctx context.Context, currencyPair string, since, limit int64, includeBreaks bool) ([]Trade, error) {
	params := url.Values{}
	if since > 0 {
		params.Add("since", strconv.FormatInt(since, 10))
	}
	if limit > 0 {
		params.Add("limit_trades", strconv.FormatInt(limit, 10))
	}
	if includeBreaks {
		params.Add("include_breaks", strconv.FormatBool(true))
	}
	path := common.EncodeURLValues(fmt.Sprintf("/v%s/%s/%s", geminiAPIVersion, geminiTrades, currencyPair), params)
	var trades []Trade

	return trades, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &trades)
}

// GetAuction returns auction information
func (g *Gemini) GetAuction(ctx context.Context, currencyPair string) (Auction, error) {
	path := fmt.Sprintf("/v%s/%s/%s", geminiAPIVersion, geminiAuction, currencyPair)
	auction := Auction{}

	return auction, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &auction)
}

// GetAuctionHistory returns the auction events, optionally including
// publications of indicative prices, since the specific timestamp.
//
// currencyPair - example "btcusd"
// params -- [optional]
//
//	since - [timestamp] Only returns auction events after the specified
//
// timestamp.
//
//	limit_auction_results - [integer] The maximum number of auction
//
// events to return.
//
//	include_indicative - [bool] Whether to include publication of
//
// indicative prices and quantities.
func (g *Gemini) GetAuctionHistory(ctx context.Context, currencyPair string, params url.Values) ([]AuctionHistory, error) {
	path := common.EncodeURLValues(fmt.Sprintf("/v%s/%s/%s/%s", geminiAPIVersion, geminiAuction, currencyPair, geminiAuctionHistory), params)
	var auctionHist []AuctionHistory
	return auctionHist, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &auctionHist)
}

// NewOrder Only limit orders are supported through the API at present.
// returns order ID if successful
func (g *Gemini) NewOrder(ctx context.Context, symbol string, amount, price float64, side, orderType string) (int64, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	req["side"] = side
	req["type"] = orderType

	response := Order{}
	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiOrderNew, req, &response)
	if err != nil {
		return 0, err
	}
	return response.OrderID, nil
}

// Transfers returns transfer history ie withdrawals and deposits
func (g *Gemini) Transfers(ctx context.Context, curr currency.Code, start time.Time, limit int64, account string, showCompletedDeposit bool) ([]TransferResponse, error) {
	req := make(map[string]any)
	if !curr.IsEmpty() {
		req["symbol"] = curr.String()
	}
	if !start.IsZero() {
		req["timestamp"] = start.Unix()
	}
	if limit > 0 {
		req["limit"] = limit
	}
	if account != "" {
		req["account"] = account
	}
	if showCompletedDeposit {
		req["showCompletedDeposit"] = showCompletedDeposit
	}

	var response []TransferResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiTransfers, req, &response)
}

// CancelExistingOrder will cancel an order. If the order is already canceled, the
// message will succeed but have no effect.
func (g *Gemini) CancelExistingOrder(ctx context.Context, orderID int64) (Order, error) {
	req := make(map[string]any)
	req["order_id"] = orderID

	response := Order{}
	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiOrderCancel, req, &response)
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
// session associated with the APIKEY
func (g *Gemini) CancelExistingOrders(ctx context.Context, cancelBySession bool) (OrderResult, error) {
	path := geminiOrderCancelAll
	if cancelBySession {
		path = geminiOrderCancelSession
	}

	var response OrderResult
	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, &response)
	if err != nil {
		return response, err
	}
	if response.Message != "" {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// GetOrderStatus returns the status for an order
func (g *Gemini) GetOrderStatus(ctx context.Context, orderID int64) (Order, error) {
	req := make(map[string]any)
	req["order_id"] = orderID

	response := Order{}

	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiOrderStatus, req, &response)
	if err != nil {
		return response, err
	}

	if response.Message != "" {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// GetOrders returns active orders in the market
func (g *Gemini) GetOrders(ctx context.Context) ([]Order, error) {
	var response any

	type orders struct {
		orders []Order
	}

	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiOrders, nil, &response)
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
func (g *Gemini) GetTradeHistory(ctx context.Context, currencyPair string, timestamp int64) ([]TradeHistory, error) {
	var response []TradeHistory
	req := make(map[string]any)
	req["symbol"] = currencyPair

	if timestamp > 0 {
		req["timestamp"] = timestamp
	}

	return response,
		g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiMyTrades, req, &response)
}

// GetNotionalVolume returns  the volume in price currency that has been traded across all pairs over a period of 30 days
func (g *Gemini) GetNotionalVolume(ctx context.Context) (NotionalVolume, error) {
	response := NotionalVolume{}

	return response,
		g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiVolume, nil, &response)
}

// GetTradeVolume returns a multi-arrayed volume response
func (g *Gemini) GetTradeVolume(ctx context.Context) ([][]TradeVolume, error) {
	var response [][]TradeVolume

	return response,
		g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiTradeVolume, nil, &response)
}

// GetBalances returns available balances in the supported currencies
func (g *Gemini) GetBalances(ctx context.Context) ([]Balance, error) {
	var response []Balance

	return response,
		g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiBalances, nil, &response)
}

// GetCryptoDepositAddress returns a deposit address
func (g *Gemini) GetCryptoDepositAddress(ctx context.Context, depositAddlabel, currency string) (DepositAddress, error) {
	response := DepositAddress{}
	req := make(map[string]any)

	if depositAddlabel != "" {
		req["label"] = depositAddlabel
	}

	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiDeposit+"/"+currency+"/"+geminiNewAddress, req, &response)
	if err != nil {
		return response, err
	}
	if response.Message != "" {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// WithdrawCrypto withdraws crypto currency to a whitelisted address
func (g *Gemini) WithdrawCrypto(ctx context.Context, address, currency string, amount float64) (WithdrawalAddress, error) {
	response := WithdrawalAddress{}
	req := make(map[string]any)
	req["address"] = address
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)

	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiWithdraw+strings.ToLower(currency), req, &response)
	if err != nil {
		return response, err
	}
	if response.Message != "" {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// PostHeartbeat sends a maintenance heartbeat to the exchange for all heartbeat
// maintained sessions
func (g *Gemini) PostHeartbeat(ctx context.Context) (string, error) {
	type Response struct {
		Result  string `json:"result"`
		Message string `json:"message"`
	}
	response := Response{}

	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, geminiHeartbeat, nil, &response)
	if err != nil {
		return response.Result, err
	}
	if response.Message != "" {
		return response.Result, errors.New(response.Message)
	}
	return response.Result, nil
}

// SendHTTPRequest sends an unauthenticated request
func (g *Gemini) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := g.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       g.Verbose,
		HTTPDebugging: g.HTTPDebugging,
		HTTPRecording: g.HTTPRecording,
	}

	return g.SendPayload(ctx, request.UnAuth, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to the
// exchange and returns an error
func (g *Gemini) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, params map[string]any, result any) (err error) {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}

	endpoint, err := g.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	return g.SendPayload(ctx, request.Auth, func() (*request.Item, error) {
		req := make(map[string]any)
		req["request"] = fmt.Sprintf("/v%s/%s", geminiAPIVersion, path)
		req["nonce"] = g.Requester.GetNonce(nonce.UnixNano).String()

		maps.Copy(req, params)

		payload, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}

		payloadB64 := base64.StdEncoding.EncodeToString(payload)
		hmac, err := crypto.GetHMAC(crypto.HashSHA512_384, []byte(payloadB64), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		headers["Content-Length"] = "0"
		headers["Content-Type"] = "text/plain"
		headers["X-GEMINI-APIKEY"] = creds.Key
		headers["X-GEMINI-PAYLOAD"] = payloadB64
		headers["X-GEMINI-SIGNATURE"] = hex.EncodeToString(hmac)
		headers["Cache-Control"] = "no-cache"

		return &request.Item{
			Method:        method,
			Path:          endpoint + "/v1/" + path,
			Headers:       headers,
			Result:        result,
			NonceEnabled:  true,
			Verbose:       g.Verbose,
			HTTPDebugging: g.HTTPDebugging,
			HTTPRecording: g.HTTPRecording,
		}, nil
	}, request.AuthenticatedRequest)
}

// GetFee returns an estimate of fee based on type of transaction
func (g *Gemini) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		notionVolume, err := g.GetNotionalVolume(ctx)
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
