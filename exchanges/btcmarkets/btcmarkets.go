package btcmarkets

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errInvalidAmount = errors.New("cannot be less than or equal to zero")
	errIDRequired    = errors.New("id is required")
)

const (
	btcMarketsAPIURL     = "https://api.btcmarkets.net"
	tradeBaseURL         = "https://app.btcmarkets.net/buy-sell?market="
	btcMarketsAPIVersion = "/v3"

	// UnAuthenticated EPs
	btcMarketsAllMarkets         = "/markets"
	btcMarketsGetTicker          = "/ticker"
	btcMarketsGetTrades          = "/trades?"
	btcMarketOrderBook           = "/orderbook?"
	btcMarketsCandles            = "/candles?"
	btcMarketsTickers            = "tickers?"
	btcMarketsMultipleOrderbooks = "orderbooks?"
	btcMarketsGetTime            = "/time"
	btcMarketsWithdrawalFees     = "/withdrawal-fees"
	btcMarketsUnauthPath         = btcMarketsAPIURL + btcMarketsAPIVersion + btcMarketsAllMarkets

	// Authenticated EPs
	btcMarketsAccountBalance = "/accounts/me/balances"
	btcMarketsTradingFees    = "/accounts/me/trading-fees"
	btcMarketsTransactions   = "/accounts/me/transactions"
	btcMarketsOrders         = "/orders"
	btcMarketsTradeHistory   = "/trades"
	btcMarketsWithdrawals    = "/withdrawals"
	btcMarketsDeposits       = "/deposits"
	btcMarketsTransfers      = "/transfers"
	btcMarketsAddresses      = "/addresses"
	btcMarketsAssets         = "/assets"
	btcMarketsReports        = "/reports"
	btcMarketsBatchOrders    = "/batchorders"

	orderFailed             = "Failed"
	orderPartiallyCancelled = "Partially Cancelled"
	orderCancelled          = "Cancelled"
	orderFullyMatched       = "Fully Matched"
	orderPartiallyMatched   = "Partially Matched"
	orderPlaced             = "Placed"
	orderAccepted           = "Accepted"

	ask = "ask"

	// order types
	limit      = "Limit"
	market     = "Market"
	stopLimit  = "Stop Limit"
	stop       = "Stop"
	takeProfit = "Take Profit"

	// order sides
	askSide = "Ask"
	bidSide = "Bid"

	subscribe         = "subscribe"
	fundChange        = "fundChange"
	orderChange       = "orderChange"
	heartbeat         = "heartbeat"
	tick              = "tick"
	wsOrderbookUpdate = "orderbookUpdate"
	tradeEndPoint     = "trade"

	// Subscription management when connection and subscription established
	addSubscription    = "addSubscription"
	removeSubscription = "removeSubscription"
	clientType         = "api"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with BTC Markets
type Exchange struct {
	exchange.Base
}

// GetMarkets returns the BTCMarkets instruments
func (e *Exchange) GetMarkets(ctx context.Context) ([]Market, error) {
	var resp []Market
	return resp, e.SendHTTPRequest(ctx, btcMarketsUnauthPath, &resp)
}

// GetTicker returns a ticker
// symbol - example "btc" or "ltc"
func (e *Exchange) GetTicker(ctx context.Context, marketID string) (Ticker, error) {
	var tick Ticker
	return tick, e.SendHTTPRequest(ctx, btcMarketsUnauthPath+"/"+marketID+btcMarketsGetTicker, &tick)
}

// GetTrades returns executed trades on the exchange
func (e *Exchange) GetTrades(ctx context.Context, marketID string, before, after, limit int64) ([]Trade, error) {
	if (before > 0) && (after >= 0) {
		return nil, errors.New("BTCMarkets only supports either before or after, not both")
	}
	var trades []Trade
	params := url.Values{}
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after > 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return trades, e.SendHTTPRequest(ctx, btcMarketsUnauthPath+"/"+marketID+btcMarketsGetTrades+params.Encode(),
		&trades)
}

// GetOrderbook returns current orderbook.
// levels are:
// 0 - Returns the top bids and ask orders only.
// 1 - Returns top 50 bids and asks.
// 2 - Returns full orderbook. WARNING: This is cached every 10 seconds.
func (e *Exchange) GetOrderbook(ctx context.Context, marketID string, level int64) (*Orderbook, error) {
	params := url.Values{}
	if level != 0 {
		params.Set("level", strconv.FormatInt(level, 10))
	}

	var resp tempOrderbook
	if err := e.SendHTTPRequest(ctx, btcMarketsUnauthPath+"/"+marketID+btcMarketOrderBook+params.Encode(), &resp); err != nil {
		return nil, err
	}

	return &Orderbook{MarketID: resp.MarketID, SnapshotID: resp.SnapshotID, Bids: resp.Bids.Levels(), Asks: resp.Asks.Levels()}, nil
}

// GetMarketCandles gets candles for specified currency pair
func (e *Exchange) GetMarketCandles(ctx context.Context, marketID, timeWindow string, from, to time.Time, before, after, limit int64) (out []CandleResponse, err error) {
	if (before > 0) && (after >= 0) {
		return out, errors.New("BTCMarkets only supports either before or after, not both")
	}
	params := url.Values{}
	params.Set("timeWindow", timeWindow)

	if from.After(to) && !to.IsZero() {
		return out, errors.New("start time cannot be after end time")
	}
	if !from.IsZero() {
		params.Set("from", from.UTC().Format(time.RFC3339))
	}
	if !to.IsZero() {
		params.Set("to", to.UTC().Format(time.RFC3339))
	}
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after >= 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return out, e.SendHTTPRequest(ctx, btcMarketsUnauthPath+"/"+marketID+btcMarketsCandles+params.Encode(), &out)
}

// GetTickers gets multiple tickers
func (e *Exchange) GetTickers(ctx context.Context, marketIDs currency.Pairs) ([]Ticker, error) {
	var tickers []Ticker
	params := url.Values{}
	pFmt, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	for x := range marketIDs {
		params.Add("marketId", pFmt.Format(marketIDs[x]))
	}
	return tickers, e.SendHTTPRequest(ctx, btcMarketsUnauthPath+"/"+btcMarketsTickers+params.Encode(),
		&tickers)
}

// GetMultipleOrderbooks gets orderbooks
func (e *Exchange) GetMultipleOrderbooks(ctx context.Context, marketIDs []string) ([]Orderbook, error) {
	params := url.Values{}
	for x := range marketIDs {
		params.Add("marketId", marketIDs[x])
	}

	var resp []tempOrderbook
	if err := e.SendHTTPRequest(ctx, btcMarketsUnauthPath+"/"+btcMarketsMultipleOrderbooks+params.Encode(), &resp); err != nil {
		return nil, err
	}

	orderbooks := make([]Orderbook, len(marketIDs))
	for i := range resp {
		orderbooks[i] = Orderbook{
			MarketID:   resp[i].MarketID,
			SnapshotID: resp[i].SnapshotID,
			Asks:       resp[i].Asks.Levels(),
			Bids:       resp[i].Bids.Levels(),
		}
	}
	return orderbooks, nil
}

// GetCurrentServerTime gets time from btcmarkets
func (e *Exchange) GetCurrentServerTime(ctx context.Context) (time.Time, error) {
	var resp TimeResp
	return resp.Time, e.SendHTTPRequest(ctx, btcMarketsAPIURL+btcMarketsAPIVersion+btcMarketsGetTime,
		&resp)
}

// GetAccountBalance returns the full account balance
func (e *Exchange) GetAccountBalance(ctx context.Context) ([]AccountData, error) {
	var resp []AccountData
	return resp,
		e.SendAuthenticatedRequest(ctx, http.MethodGet,
			btcMarketsAccountBalance,
			nil,
			&resp,
			request.Auth)
}

// GetTradingFees returns trading fees for all pairs based on trading activity
func (e *Exchange) GetTradingFees(ctx context.Context) (TradingFeeResponse, error) {
	var resp TradingFeeResponse
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		btcMarketsTradingFees,
		nil,
		&resp,
		request.Auth)
}

// GetTradeHistory returns trade history
func (e *Exchange) GetTradeHistory(ctx context.Context, marketID, orderID string, before, after, limit int64) ([]TradeHistoryData, error) {
	if (before > 0) && (after >= 0) {
		return nil, errors.New("BTCMarkets only supports either before or after, not both")
	}
	var resp []TradeHistoryData
	params := url.Values{}
	if marketID != "" {
		params.Set("marketId", marketID)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after >= 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		common.EncodeURLValues(btcMarketsTradeHistory, params),
		nil,
		&resp,
		request.Auth)
}

// GetTradeByID returns the singular trade of the ID given
func (e *Exchange) GetTradeByID(ctx context.Context, id string) (TradeHistoryData, error) {
	var resp TradeHistoryData
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		btcMarketsTradeHistory+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// formatOrderType conforms order type to the exchange acceptable order type
// strings
func (e *Exchange) formatOrderType(o order.Type) (string, error) {
	switch o {
	case order.Limit:
		return limit, nil
	case order.Market:
		return market, nil
	case order.StopLimit:
		return stopLimit, nil
	case order.Stop:
		return stop, nil
	case order.TakeProfit:
		return takeProfit, nil
	default:
		return "", fmt.Errorf("%s %s %w", e.Name, o, order.ErrTypeIsInvalid)
	}
}

// formatOrderSide conforms order side to the exchange acceptable order side
// strings
func (e *Exchange) formatOrderSide(o order.Side) (string, error) {
	switch o {
	case order.Ask:
		return askSide, nil
	case order.Bid:
		return bidSide, nil
	default:
		return "", fmt.Errorf("%s %s %w", e.Name, o, order.ErrSideIsInvalid)
	}
}

// getTimeInForce returns a string depending on the options in order.Submit
func (e *Exchange) getTimeInForce(s *order.Submit) string {
	if s.TimeInForce.Is(order.ImmediateOrCancel) || s.TimeInForce.Is(order.FillOrKill) {
		return s.TimeInForce.String()
	}
	return "" // GTC (good till cancelled, default value)
}

// NewOrder requests a new order and returns an ID
func (e *Exchange) NewOrder(ctx context.Context, price, amount, triggerPrice, targetAmount float64, marketID, orderType, side, timeInForce, selfTrade, clientOrderID string, postOnly bool) (OrderData, error) {
	req := make(map[string]any)
	req["marketId"] = marketID
	if price != 0 {
		req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	}
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["type"] = orderType
	req["side"] = side
	if orderType == stopLimit || orderType == takeProfit || orderType == stop {
		req["triggerPrice"] = strconv.FormatFloat(triggerPrice, 'f', -1, 64)
	}
	if targetAmount > 0 {
		req["targetAmount"] = strconv.FormatFloat(targetAmount, 'f', -1, 64)
	}
	if timeInForce != "" {
		req["timeInForce"] = timeInForce
	}
	if postOnly {
		req["postOnly"] = postOnly
	}
	if selfTrade != "" {
		req["selfTrade"] = selfTrade
	}
	if clientOrderID != "" {
		req["clientOrderID"] = clientOrderID
	}
	var resp OrderData
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodPost,
		btcMarketsOrders,
		req,
		&resp,
		orderFunc)
}

// GetOrders returns current order information on the exchange
func (e *Exchange) GetOrders(ctx context.Context, marketID string, before, after, limit int64, openOnly bool) ([]OrderData, error) {
	if (before > 0) && (after >= 0) {
		return nil, errors.New("BTCMarkets only supports either before or after, not both")
	}
	var resp []OrderData
	params := url.Values{}
	if marketID != "" {
		params.Set("marketId", marketID)
	}
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after >= 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if openOnly {
		params.Set("status", "open")
	}
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		common.EncodeURLValues(btcMarketsOrders, params),
		nil,
		&resp,
		request.Auth)
}

// CancelAllOpenOrdersByPairs cancels all open orders unless pairs are specified
func (e *Exchange) CancelAllOpenOrdersByPairs(ctx context.Context, marketIDs []string) ([]CancelOrderResp, error) {
	var resp []CancelOrderResp
	req := make(map[string]any)
	if len(marketIDs) > 0 {
		var strTemp strings.Builder
		for x := range marketIDs {
			strTemp.WriteString("marketId=" + marketIDs[x] + "&")
		}
		req["marketId"] = strTemp.String()[:strTemp.Len()-1]
	}
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodDelete,
		btcMarketsOrders,
		req,
		&resp,
		request.Auth)
}

// FetchOrder finds order based on the provided id
func (e *Exchange) FetchOrder(ctx context.Context, id string) (*OrderData, error) {
	var resp *OrderData
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		btcMarketsOrders+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// RemoveOrder removes a given order
func (e *Exchange) RemoveOrder(ctx context.Context, id string) (CancelOrderResp, error) {
	var resp CancelOrderResp
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodDelete,
		btcMarketsOrders+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// ReplaceOrder cancels an order and then places a new order.
func (e *Exchange) ReplaceOrder(ctx context.Context, id, clientOrderID string, price, amount float64) (*OrderData, error) {
	if price <= 0 {
		return nil, fmt.Errorf("price %w", errInvalidAmount)
	}

	if amount <= 0 {
		return nil, fmt.Errorf("amount %w", errInvalidAmount)
	}

	if id == "" {
		return nil, errIDRequired
	}

	req := make(map[string]any, 3)
	req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	if clientOrderID != "" {
		req["clientOrderId"] = clientOrderID
	}

	var resp *OrderData
	return resp, e.SendAuthenticatedRequest(ctx,
		http.MethodPut,
		btcMarketsOrders+"/"+id,
		req,
		&resp,
		request.Auth)
}

// ListWithdrawals lists the withdrawal history
func (e *Exchange) ListWithdrawals(ctx context.Context, before, after, limit int64) ([]TransferData, error) {
	if (before > 0) && (after >= 0) {
		return nil, errors.New("BTCMarkets only supports either before or after, not both")
	}
	var resp []TransferData
	params := url.Values{}
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after >= 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		common.EncodeURLValues(btcMarketsWithdrawals, params),
		nil,
		&resp,
		request.Auth)
}

// GetWithdrawal gets withdrawawl info for a given id
func (e *Exchange) GetWithdrawal(ctx context.Context, id string) (TransferData, error) {
	var resp TransferData
	if id == "" {
		return resp, errors.New("id cannot be an empty string")
	}
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		btcMarketsWithdrawals+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// ListDeposits lists the deposit history
func (e *Exchange) ListDeposits(ctx context.Context, before, after, limit int64) ([]TransferData, error) {
	if (before > 0) && (after >= 0) {
		return nil, errors.New("BTCMarkets only supports either before or after, not both")
	}
	var resp []TransferData
	params := url.Values{}
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after >= 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		common.EncodeURLValues(btcMarketsDeposits, params),
		nil,
		&resp,
		request.Auth)
}

// GetDeposit gets deposit info for a given ID
func (e *Exchange) GetDeposit(ctx context.Context, id string) (TransferData, error) {
	var resp TransferData
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		btcMarketsDeposits+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// ListTransfers lists the past asset transfers
func (e *Exchange) ListTransfers(ctx context.Context, before, after, limit int64) ([]TransferData, error) {
	if (before > 0) && (after >= 0) {
		return nil, errors.New("BTCMarkets only supports either before or after, not both")
	}
	var resp []TransferData
	params := url.Values{}
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after >= 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		common.EncodeURLValues(btcMarketsTransfers, params),
		nil,
		&resp,
		request.Auth)
}

// GetTransfer gets asset transfer info for a given ID
func (e *Exchange) GetTransfer(ctx context.Context, id string) (TransferData, error) {
	var resp TransferData
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		btcMarketsTransfers+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// FetchDepositAddress gets deposit address for the given asset
func (e *Exchange) FetchDepositAddress(ctx context.Context, curr currency.Code, before, after, limit int64) (*DepositAddress, error) {
	var resp DepositAddress
	if (before > 0) && (after >= 0) {
		return nil, errors.New("BTCMarkets only supports either before or after, not both")
	}
	params := url.Values{}
	params.Set("assetName", curr.Upper().String())
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after >= 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if err := e.SendAuthenticatedRequest(ctx,
		http.MethodGet,
		common.EncodeURLValues(btcMarketsAddresses, params),
		nil,
		&resp,
		request.Auth); err != nil {
		return nil, err
	}
	if curr.Equal(currency.XRP) {
		splitStr := "?dt="
		if !strings.Contains(resp.Address, splitStr) {
			return nil, errors.New("unable to find split string for XRP")
		}
		splitter := strings.Split(resp.Address, splitStr)
		resp.Address = splitter[0]
		resp.Tag = splitter[1]
	}
	return &resp, nil
}

// GetWithdrawalFees gets withdrawal fees for all assets
func (e *Exchange) GetWithdrawalFees(ctx context.Context) ([]WithdrawalFeeData, error) {
	var resp []WithdrawalFeeData
	return resp, e.SendHTTPRequest(ctx, btcMarketsAPIURL+btcMarketsAPIVersion+btcMarketsWithdrawalFees,
		&resp)
}

// ListAssets lists all available assets
func (e *Exchange) ListAssets(ctx context.Context) ([]AssetData, error) {
	var resp []AssetData
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		btcMarketsAssets,
		nil,
		&resp,
		request.Auth)
}

// GetTransactions gets trading fees
func (e *Exchange) GetTransactions(ctx context.Context, assetName string, before, after, limit int64) ([]TransactionData, error) {
	if (before > 0) && (after >= 0) {
		return nil, errors.New("BTCMarkets only supports either before or after, not both")
	}
	var resp []TransactionData
	params := url.Values{}
	if assetName != "" {
		params.Set("assetName", assetName)
	}
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after >= 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		common.EncodeURLValues(btcMarketsTransactions, params),
		nil,
		&resp,
		request.Auth)
}

// CreateNewReport creates a new report
func (e *Exchange) CreateNewReport(ctx context.Context, reportType, format string) (CreateReportResp, error) {
	var resp CreateReportResp
	req := make(map[string]any)
	req["type"] = reportType
	req["format"] = format
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodPost,
		btcMarketsReports,
		req,
		&resp,
		newReportFunc)
}

// GetReport finds details bout a past report
func (e *Exchange) GetReport(ctx context.Context, reportID string) (ReportData, error) {
	var resp ReportData
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		btcMarketsReports+"/"+reportID,
		nil,
		&resp,
		request.Auth)
}

// RequestWithdraw requests withdrawals
func (e *Exchange) RequestWithdraw(ctx context.Context, assetName string, amount float64,
	toAddress, accountName, accountNumber, bsbNumber, bankName string,
) (TransferData, error) {
	var resp TransferData
	req := make(map[string]any)
	req["assetName"] = assetName
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	if assetName != "AUD" {
		req["toAddress"] = toAddress
	} else {
		if accountName != "" {
			req["accountName"] = accountName
		}
		if accountNumber != "" {
			req["accountNumber"] = accountNumber
		}
		if bsbNumber != "" {
			req["bsbNumber"] = bsbNumber
		}
		if bankName != "" {
			req["bankName"] = bankName
		}
	}
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodPost,
		btcMarketsWithdrawals,
		req,
		&resp,
		withdrawFunc)
}

// BatchPlaceCancelOrders places and cancels batch orders
func (e *Exchange) BatchPlaceCancelOrders(ctx context.Context, cancelOrders []CancelBatch, placeOrders []PlaceBatch) (*BatchPlaceCancelResponse, error) {
	numActions := len(cancelOrders) + len(placeOrders)
	if numActions > 4 {
		return nil, errors.New("BTCMarkets can only handle 4 orders at a time")
	}

	orderRequests := make([]any, numActions)
	for x := range cancelOrders {
		orderRequests[x] = CancelOrderMethod{CancelOrder: cancelOrders[x]}
	}
	for y := range placeOrders {
		if placeOrders[y].ClientOrderID == "" {
			return nil, errors.New("placeorders must have ClientOrderID filled")
		}
		orderRequests[y] = PlaceOrderMethod{PlaceOrder: placeOrders[y]}
	}
	var resp BatchPlaceCancelResponse
	return &resp, e.SendAuthenticatedRequest(ctx, http.MethodPost,
		btcMarketsBatchOrders,
		orderRequests,
		&resp,
		batchFunc)
}

// GetBatchTrades gets batch trades
func (e *Exchange) GetBatchTrades(ctx context.Context, ids []string) (BatchTradeResponse, error) {
	var resp BatchTradeResponse
	if len(ids) > 50 {
		return resp, errors.New("batchtrades can only handle 50 ids at a time")
	}
	marketIDs := strings.Join(ids, ",")
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodGet,
		btcMarketsBatchOrders+"/"+marketIDs,
		nil,
		&resp,
		request.Auth)
}

// CancelBatch cancels given ids
func (e *Exchange) CancelBatch(ctx context.Context, ids []string) (BatchCancelResponse, error) {
	var resp BatchCancelResponse
	marketIDs := strings.Join(ids, ",")
	return resp, e.SendAuthenticatedRequest(ctx, http.MethodDelete,
		btcMarketsBatchOrders+"/"+marketIDs,
		nil,
		&resp,
		batchFunc)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, path string, result any) error {
	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   path,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}
	return e.SendPayload(ctx, request.UnAuth, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedRequest sends an authenticated HTTP request
func (e *Exchange) SendAuthenticatedRequest(ctx context.Context, method, path string, data, result any, f request.EndpointLimit) (err error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}

	newRequest := func() (*request.Item, error) {
		strTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
		var body io.Reader
		var payload, hmac []byte
		switch data.(type) {
		case map[string]any, []any:
			payload, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
			strMsg := method + btcMarketsAPIVersion + path + strTime + string(payload)
			hmac, err = crypto.GetHMAC(crypto.HashSHA512,
				[]byte(strMsg),
				[]byte(creds.Secret))
			if err != nil {
				return nil, err
			}
		default:
			strArray := strings.Split(path, "?")
			hmac, err = crypto.GetHMAC(crypto.HashSHA512,
				[]byte(method+btcMarketsAPIVersion+strArray[0]+strTime),
				[]byte(creds.Secret))
			if err != nil {
				return nil, err
			}
		}

		headers := make(map[string]string)
		headers["Accept"] = "application/json"
		headers["Accept-Charset"] = "UTF-8"
		headers["Content-Type"] = "application/json"
		headers["BM-AUTH-APIKEY"] = creds.Key
		headers["BM-AUTH-TIMESTAMP"] = strTime
		headers["BM-AUTH-SIGNATURE"] = base64.StdEncoding.EncodeToString(hmac)

		return &request.Item{
			Method:                 method,
			Path:                   btcMarketsAPIURL + btcMarketsAPIVersion + path,
			Headers:                headers,
			Body:                   body,
			Result:                 result,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}

	return e.SendPayload(ctx, f, newRequest, request.AuthenticatedRequest)
}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		temp, err := e.GetTradingFees(ctx)
		if err != nil {
			return fee, err
		}
		for x := range temp.FeeByMarkets {
			if temp.FeeByMarkets[x].MarketID.Equal(feeBuilder.Pair) {
				fee = temp.FeeByMarkets[x].MakerFeeRate
				if !feeBuilder.IsMaker {
					fee = temp.FeeByMarkets[x].TakerFeeRate
				}
			}
		}
	case exchange.CryptocurrencyWithdrawalFee:
		temp, err := e.GetWithdrawalFees(ctx)
		if err != nil {
			return fee, err
		}
		for x := range temp {
			if currency.NewCode(temp[x].AssetName) == feeBuilder.Pair.Base {
				fee = temp[x].Fee * feeBuilder.PurchasePrice * feeBuilder.Amount
			}
		}
	case exchange.InternationalBankWithdrawalFee:
		return 0, errors.New("international bank withdrawals are not supported")

	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(feeBuilder *exchange.FeeBuilder) float64 {
	switch {
	case feeBuilder.Pair.IsCryptoPair():
		return 0.002 * feeBuilder.PurchasePrice * feeBuilder.Amount
	default:
		return 0.0085 * feeBuilder.PurchasePrice * feeBuilder.Amount
	}
}
