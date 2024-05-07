package btse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// BTSE is the overarching type across this package
type BTSE struct {
	exchange.Base
}

const (
	btseAPIURL         = "https://api.btse.com"
	btseSPOTPath       = "/spot"
	btseSPOTAPIPath    = "/api/v3.2/"
	btseFuturesPath    = "/futures"
	btseFuturesAPIPath = "/api/v2.1/"
	tradeBaseURL       = "https://www.btse.com/en/"
	tradeSpot          = "trading/"
	tradeFutures       = "futures/"

	// Public endpoints
	btseMarketOverview = "market_summary"
	btseMarkets        = "markets"
	btseOrderbook      = "orderbook"
	btseTrades         = "trades"
	btseTime           = "time"
	btseOHLCV          = "ohlcv"
	btsePrice          = "price"
	btseFuturesFunding = "funding_history"

	// Authenticated endpoints
	btseWallet           = "user/wallet"
	btseWalletHistory    = "user/wallet_history"
	btseWalletAddress    = "user/wallet/address"
	btseWalletWithdrawal = "user/wallet/withdraw"
	btseExchangeHistory  = "user/trade_history"
	btseUserFee          = "user/fees"
	btseOrder            = "order"
	btsePegOrder         = "order/peg"
	btsePendingOrders    = "user/open_orders"
	btseCancelAllAfter   = "order/cancelAllAfter"
)

// FetchFundingHistory gets funding history
func (b *BTSE) FetchFundingHistory(ctx context.Context, symbol string) (map[string][]FundingHistoryData, error) {
	var resp map[string][]FundingHistoryData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, btseFuturesFunding+params.Encode(), &resp, false, queryFunc)
}

// GetMarketSummary stores market summary data
func (b *BTSE) GetMarketSummary(ctx context.Context, symbol string, spot bool) (MarketSummary, error) {
	var m MarketSummary
	path := btseMarketOverview
	if symbol != "" {
		path += "?symbol=" + url.QueryEscape(symbol)
	}
	return m, b.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, &m, spot, queryFunc)
}

// FetchOrderBook gets orderbook data for a given pair
func (b *BTSE) FetchOrderBook(ctx context.Context, symbol string, group, limitBids, limitAsks int, spot bool) (*Orderbook, error) {
	var o Orderbook
	urlValues := url.Values{}
	urlValues.Add("symbol", symbol)
	if limitBids > 0 {
		urlValues.Add("limit_bids", strconv.Itoa(limitBids))
	}
	if limitAsks > 0 {
		urlValues.Add("limit_asks", strconv.Itoa(limitAsks))
	}
	if group > 0 {
		urlValues.Add("group", strconv.Itoa(group))
	}
	return &o, b.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		common.EncodeURLValues(btseOrderbook, urlValues), &o, spot, queryFunc)
}

// FetchOrderBookL2 retrieve level 2 orderbook for requested symbol and depth
func (b *BTSE) FetchOrderBookL2(ctx context.Context, symbol string, depth int) (*Orderbook, error) {
	var o Orderbook
	urlValues := url.Values{}
	urlValues.Add("symbol", symbol)
	urlValues.Add("depth", strconv.FormatInt(int64(depth), 10))
	endpoint := common.EncodeURLValues(btseOrderbook+"/L2", urlValues)
	return &o, b.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, &o, true, queryFunc)
}

// GetTrades returns a list of trades for the specified symbol
func (b *BTSE) GetTrades(ctx context.Context, symbol string, start, end time.Time, beforeSerialID, afterSerialID, count int, includeOld, spot bool) ([]Trade, error) {
	var t []Trade
	urlValues := url.Values{}
	urlValues.Add("symbol", symbol)
	if count > 0 {
		urlValues.Add("count", strconv.Itoa(count))
	}
	if !start.IsZero() {
		urlValues.Add("start", strconv.FormatInt(start.Unix(), 10))
	}
	if !end.IsZero() {
		urlValues.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return t, common.ErrStartAfterEnd
	}
	if beforeSerialID > 0 {
		urlValues.Add("beforeSerialId", strconv.Itoa(beforeSerialID))
	}
	if afterSerialID > 0 {
		urlValues.Add("afterSerialId", strconv.Itoa(afterSerialID))
	}
	if includeOld {
		urlValues.Add("includeOld", "true")
	}
	return t, b.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		common.EncodeURLValues(btseTrades, urlValues), &t, spot, queryFunc)
}

// GetOHLCV retrieve and return OHLCV candle data for requested symbol
func (b *BTSE) GetOHLCV(ctx context.Context, symbol string, start, end time.Time, resolution int, a asset.Item) (OHLCV, error) {
	var o OHLCV
	urlValues := url.Values{}
	urlValues.Add("symbol", symbol)

	if !start.IsZero() && !end.IsZero() {
		if start.After(end) {
			return o, common.ErrStartAfterEnd
		}
		urlValues.Add("start", strconv.FormatInt(start.Unix(), 10))
		urlValues.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	var res = 60
	if resolution != 0 {
		res = resolution
	}
	urlValues.Add("resolution", strconv.FormatInt(int64(res), 10))
	endpoint := common.EncodeURLValues(btseOHLCV, urlValues)

	return o, b.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, &o, a == asset.Spot, queryFunc)
}

// GetPrice get current price for requested symbol
func (b *BTSE) GetPrice(ctx context.Context, symbol string) (Price, error) {
	var p Price
	path := btsePrice + "?symbol=" + url.QueryEscape(symbol)
	return p, b.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, &p, true, queryFunc)
}

// GetCurrentServerTime returns the exchanges server time
func (b *BTSE) GetCurrentServerTime(ctx context.Context) (*ServerTime, error) {
	var s ServerTime
	return &s, b.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, btseTime, &s, true, queryFunc)
}

// GetWalletInformation returns the users account balance
func (b *BTSE) GetWalletInformation(ctx context.Context) ([]CurrencyBalance, error) {
	var a []CurrencyBalance
	return a, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, btseWallet, true, nil, nil, &a, queryFunc)
}

// GetFeeInformation retrieve fee's (maker/taker) for requested symbol
func (b *BTSE) GetFeeInformation(ctx context.Context, symbol string) ([]AccountFees, error) {
	var resp []AccountFees
	urlValues := url.Values{}
	if symbol != "" {
		urlValues.Add("symbol", symbol)
	}
	return resp, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, btseUserFee, true, urlValues, nil, &resp, queryFunc)
}

// GetWalletHistory returns the users account balance
func (b *BTSE) GetWalletHistory(ctx context.Context, symbol string, start, end time.Time, count int) (WalletHistory, error) {
	var resp WalletHistory

	urlValues := url.Values{}
	if symbol != "" {
		urlValues.Add("symbol", symbol)
	}
	if !start.IsZero() && !end.IsZero() {
		if start.After(end) || end.Before(start) {
			return resp, common.ErrStartAfterEnd
		}
		urlValues.Add("start", strconv.FormatInt(start.Unix(), 10))
		urlValues.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	if count > 0 {
		urlValues.Add("count", strconv.Itoa(count))
	}
	return resp, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, btseWalletHistory, true, urlValues, nil, &resp, queryFunc)
}

// GetWalletAddress returns the users account balance
func (b *BTSE) GetWalletAddress(ctx context.Context, currency string) (WalletAddress, error) {
	var resp WalletAddress

	urlValues := url.Values{}
	if currency != "" {
		urlValues.Add("currency", currency)
	}

	return resp, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, btseWalletAddress, true, urlValues, nil, &resp, queryFunc)
}

// CreateWalletAddress create new deposit address for requested currency
func (b *BTSE) CreateWalletAddress(ctx context.Context, currency string) (WalletAddress, error) {
	var resp WalletAddress
	req := make(map[string]interface{}, 1)
	req["currency"] = currency
	err := b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, btseWalletAddress, true, nil, req, &resp, queryFunc)
	if err != nil {
		errResp := ErrorResponse{}
		errResponseStr := strings.Split(err.Error(), "raw response: ")
		err := json.Unmarshal([]byte(errResponseStr[1]), &errResp)
		if err != nil {
			return resp, err
		}
		if errResp.ErrorCode == 3528 {
			walletAddress := strings.Split(errResp.Message, "BADREQUEST: ")
			return WalletAddress{
				{
					Address: walletAddress[1],
				},
			}, nil
		}
		return resp, err
	}

	return resp, nil
}

// WalletWithdrawal submit request to withdraw crypto currency
func (b *BTSE) WalletWithdrawal(ctx context.Context, currency, address, tag, amount string) (WithdrawalResponse, error) {
	var resp WithdrawalResponse
	req := make(map[string]interface{}, 4)
	req["currency"] = currency
	req["address"] = address
	req["tag"] = tag
	req["amount"] = amount
	return resp, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, btseWalletWithdrawal, true, nil, req, &resp, queryFunc)
}

// CreateOrder creates an order
func (b *BTSE) CreateOrder(ctx context.Context, clOrderID string, deviation float64, postOnly bool, price float64, side string, size, stealth, stopPrice float64, symbol, timeInForce string, trailValue, triggerPrice float64, txType, orderType string) ([]Order, error) {
	req := make(map[string]interface{})
	if clOrderID != "" {
		req["clOrderID"] = clOrderID
	}
	if deviation > 0.0 {
		req["deviation"] = deviation
	}
	if postOnly {
		req["postOnly"] = postOnly
	}
	if price > 0.0 {
		req["price"] = price
	}
	if side != "" {
		req["side"] = side
	}
	if size > 0.0 {
		req["size"] = size
	}
	if stealth > 0.0 {
		req["stealth"] = stealth
	}
	if stopPrice > 0.0 {
		req["stopPrice"] = stopPrice
	}
	if symbol != "" {
		req["symbol"] = symbol
	}
	if timeInForce != "" {
		req["time_in_force"] = timeInForce
	}
	if trailValue > 0.0 {
		req["trailValue"] = trailValue
	}
	if triggerPrice > 0.0 {
		req["triggerPrice"] = triggerPrice
	}
	if txType != "" {
		req["txType"] = txType
	}
	if orderType != "" {
		req["type"] = orderType
	}

	var r []Order
	return r, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, btseOrder, true, url.Values{}, req, &r, orderFunc)
}

// GetOrders returns all pending orders
func (b *BTSE) GetOrders(ctx context.Context, symbol, orderID, clOrderID string) ([]OpenOrder, error) {
	req := url.Values{}
	if orderID != "" {
		req.Add("orderID", orderID)
	}
	req.Add("symbol", symbol)
	if clOrderID != "" {
		req.Add("clOrderID", clOrderID)
	}
	var o []OpenOrder
	return o, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, btsePendingOrders, true, req, nil, &o, orderFunc)
}

// CancelExistingOrder cancels an order
func (b *BTSE) CancelExistingOrder(ctx context.Context, orderID, symbol, clOrderID string) (CancelOrder, error) {
	var c CancelOrder
	req := url.Values{}
	if orderID != "" {
		req.Add("orderID", orderID)
	}
	req.Add("symbol", symbol)
	if clOrderID != "" {
		req.Add("clOrderID", clOrderID)
	}

	return c, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, btseOrder, true, req, nil, &c, orderFunc)
}

// CancelAllAfter cancels all orders after timeout
func (b *BTSE) CancelAllAfter(ctx context.Context, timeout int) error {
	req := make(map[string]interface{})
	req["timeout"] = timeout
	return b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, btseCancelAllAfter, true, url.Values{}, req, nil, orderFunc)
}

// IndexOrderPeg create peg order that will track a certain percentage above/below the index price
func (b *BTSE) IndexOrderPeg(ctx context.Context, clOrderID string, deviation float64, postOnly bool, price float64, side string, size, stealth, stopPrice float64, symbol, timeInForce string, trailValue, triggerPrice float64, txType, orderType string) ([]Order, error) {
	var o []Order
	req := make(map[string]interface{})
	if clOrderID != "" {
		req["clOrderID"] = clOrderID
	}
	if deviation > 0.0 {
		req["deviation"] = deviation
	}
	if postOnly {
		req["postOnly"] = postOnly
	}
	if price > 0.0 {
		req["price"] = price
	}
	if side != "" {
		req["side"] = side
	}
	if size > 0.0 {
		req["size"] = size
	}
	if stealth > 0.0 {
		req["stealth"] = stealth
	}
	if stopPrice > 0.0 {
		req["stopPrice"] = stopPrice
	}
	if symbol != "" {
		req["symbol"] = symbol
	}
	if timeInForce != "" {
		req["time_in_force"] = timeInForce
	}
	if trailValue > 0.0 {
		req["trailValue"] = trailValue
	}
	if triggerPrice > 0.0 {
		req["triggerPrice"] = triggerPrice
	}
	if txType != "" {
		req["txType"] = txType
	}
	if orderType != "" {
		req["type"] = orderType
	}

	return o, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, btsePegOrder, true, url.Values{}, req, nil, orderFunc)
}

// TradeHistory returns previous trades on exchange
func (b *BTSE) TradeHistory(ctx context.Context, symbol string, start, end time.Time, beforeSerialID, afterSerialID, count int, includeOld bool, clOrderID, orderID string) (TradeHistory, error) {
	var resp TradeHistory
	urlValues := url.Values{}
	if symbol != "" {
		urlValues.Add("symbol", symbol)
	}
	if !start.IsZero() && !end.IsZero() {
		if start.After(end) || end.Before(start) {
			return resp, errors.New("start and end must both be valid")
		}
		urlValues.Add("start", strconv.FormatInt(start.Unix(), 10))
		urlValues.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	if beforeSerialID > 0 {
		urlValues.Add("beforeSerialId", strconv.Itoa(beforeSerialID))
	}
	if afterSerialID > 0 {
		urlValues.Add("afterSerialId", strconv.Itoa(afterSerialID))
	}
	if includeOld {
		urlValues.Add("includeOld", "true")
	}
	if count > 0 {
		urlValues.Add("count", strconv.Itoa(count))
	}
	if clOrderID != "" {
		urlValues.Add("clOrderId", clOrderID)
	}
	if orderID != "" {
		urlValues.Add("orderID", orderID)
	}
	return resp, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, btseExchangeHistory, true, urlValues, nil, &resp, queryFunc)
}

// SendHTTPRequest sends an HTTP request to the desired endpoint
func (b *BTSE) SendHTTPRequest(ctx context.Context, ep exchange.URL, method, endpoint string, result interface{}, spotEndpoint bool, f request.EndpointLimit) error {
	ePoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	p := btseSPOTPath + btseSPOTAPIPath
	if !spotEndpoint {
		p = btseFuturesPath + btseFuturesAPIPath
	}
	item := &request.Item{
		Method:        method,
		Path:          ePoint + p + endpoint,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	}
	return b.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to the desired endpoint
func (b *BTSE) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, endpoint string, isSpot bool, values url.Values, req map[string]interface{}, result interface{}, f request.EndpointLimit) error {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}

	ePoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	newRequest := func() (*request.Item, error) {
		// The concatenation is done this way because BTSE expect endpoint+nonce or endpoint+nonce+body
		// when signing the data but the full path of the request  is /spot/api/v3.2/<endpoint>
		// its messy but it works and supports futures as well
		host := ePoint
		var expandedEndpoint string
		if isSpot {
			host += btseSPOTPath + btseSPOTAPIPath + endpoint
			expandedEndpoint = btseSPOTAPIPath + endpoint
		} else {
			host += btseFuturesPath + btseFuturesAPIPath
			expandedEndpoint = endpoint + btseFuturesAPIPath
		}

		var hmac []byte
		var body io.Reader
		nonce := strconv.FormatInt(time.Now().UnixMilli(), 10)
		headers := map[string]string{
			"btse-api":   creds.Key,
			"btse-nonce": nonce,
		}
		if req != nil {
			var reqPayload []byte
			reqPayload, err = json.Marshal(req)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(reqPayload)
			hmac, err = crypto.GetHMAC(
				crypto.HashSHA512_384,
				[]byte(expandedEndpoint+nonce+string(reqPayload)),
				[]byte(creds.Secret),
			)
			if err != nil {
				return nil, err
			}
			headers["Content-Type"] = "application/json"
		} else {
			hmac, err = crypto.GetHMAC(
				crypto.HashSHA512_384,
				[]byte(expandedEndpoint+nonce),
				[]byte(creds.Secret),
			)
			if err != nil {
				return nil, err
			}
			if len(values) > 0 {
				host += "?" + values.Encode()
			}
		}
		headers["btse-sign"] = crypto.HexEncodeToString(hmac)

		return &request.Item{
			Method:        method,
			Path:          host,
			Headers:       headers,
			Body:          body,
			Result:        result,
			Verbose:       b.Verbose,
			HTTPDebugging: b.HTTPDebugging,
			HTTPRecording: b.HTTPRecording,
		}, nil
	}
	return b.SendPayload(ctx, f, newRequest, request.AuthenticatedRequest)
}

// GetFee returns an estimate of fee based on type of transaction
func (b *BTSE) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = b.calculateTradingFee(ctx, feeBuilder) * feeBuilder.Amount * feeBuilder.PurchasePrice
	case exchange.CryptocurrencyWithdrawalFee:
		switch feeBuilder.Pair.Base {
		case currency.USDT:
			fee = 1.08
		case currency.TUSD:
			fee = 1.09
		case currency.BTC:
			fee = 0.0005
		case currency.ETH:
			fee = 0.01
		case currency.LTC:
			fee = 0.001
		}
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.Amount)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.Amount)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.001 * price * amount
}

// getInternationalBankDepositFee returns international deposit fee
// Only when the initial deposit amount is less than $1000 or equivalent,
// BTSE will charge a small fee (0.25% or $3 USD equivalent, whichever is greater).
// The small deposit fee is charged in whatever currency it comes in.
func getInternationalBankDepositFee(amount float64) float64 {
	var fee float64
	if amount <= 100 {
		fee = amount * 0.0025
		if fee < 3 {
			return 3
		}
	}
	return fee
}

// getInternationalBankWithdrawalFee returns international withdrawal fee
// 0.1% (min25 USD)
func getInternationalBankWithdrawalFee(amount float64) float64 {
	fee := amount * 0.0009

	if fee < 25 {
		return 25
	}
	return fee
}

// calculateTradingFee return fee based on users current fee tier or default values
func (b *BTSE) calculateTradingFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) float64 {
	formattedPair, err := b.FormatExchangeCurrency(feeBuilder.Pair, asset.Spot)
	if err != nil {
		if feeBuilder.IsMaker {
			return 0.001
		}
		return 0.002
	}
	feeTiers, err := b.GetFeeInformation(ctx, formattedPair.String())
	if err != nil {
		// TODO: Return actual error, we shouldn't pivot around errors.
		if feeBuilder.IsMaker {
			return 0.001
		}
		return 0.002
	}
	if feeBuilder.IsMaker {
		return feeTiers[0].MakerFee
	}
	return feeTiers[0].TakerFee
}

func parseOrderTime(timeStr string) (time.Time, error) {
	return time.Parse(time.DateTime, timeStr)
}

// MillionPairs returns a map of symbol names which have a IsMillion equivalent
func (m *MarketSummary) MillionPairs() map[string]bool {
	pairs := map[string]bool{}
	for _, s := range *m {
		if s.Active && s.HasLiquidity() && s.IsMillions() {
			pairs[strings.TrimPrefix(s.Symbol, "M_")] = true
		}
	}
	return pairs
}

// HasLiquidity returns if a market pair has a bid or ask != 0
func (m *MarketPair) HasLiquidity() bool {
	return m.LowestAsk != 0 || m.HighestBid != 0
}

// IsMillions returns if a market pair represents a million of Base / Quote
func (m *MarketPair) IsMillions() bool {
	return strings.HasPrefix(m.Symbol, "M_")
}
