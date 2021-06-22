package btcmarkets

import (
	"bytes"
	"context"
	"encoding/json"
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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	btcMarketsAPIURL     = "https://api.btcmarkets.net"
	btcMarketsAPIVersion = "/v3"

	// UnAuthenticated EPs
	btcMarketsAllMarkets         = "/markets/"
	btcMarketsGetTicker          = "/ticker/"
	btcMarketsGetTrades          = "/trades?"
	btcMarketOrderBooks          = "/orderbook?"
	btcMarketsCandles            = "/candles?"
	btcMarketsTickers            = "tickers?"
	btcMarketsMultipleOrderbooks = "/orderbooks?"
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

	ask        = "ask"
	limit      = "Limit"
	market     = "Market"
	stopLimit  = "Stop Limit"
	stop       = "Stop"
	takeProfit = "Take Profit"

	subscribe     = "subscribe"
	fundChange    = "fundChange"
	orderChange   = "orderChange"
	heartbeat     = "heartbeat"
	tick          = "tick"
	wsOB          = "orderbookUpdate"
	tradeEndPoint = "trade"
)

// BTCMarkets is the overarching type across the BTCMarkets package
type BTCMarkets struct {
	exchange.Base
}

// GetMarkets returns the BTCMarkets instruments
func (b *BTCMarkets) GetMarkets() ([]Market, error) {
	var resp []Market
	return resp, b.SendHTTPRequest(btcMarketsUnauthPath, &resp)
}

// GetTicker returns a ticker
// symbol - example "btc" or "ltc"
func (b *BTCMarkets) GetTicker(marketID string) (Ticker, error) {
	var tick Ticker
	return tick, b.SendHTTPRequest(btcMarketsUnauthPath+marketID+btcMarketsGetTicker, &tick)
}

// GetTrades returns executed trades on the exchange
func (b *BTCMarkets) GetTrades(marketID string, before, after, limit int64) ([]Trade, error) {
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
	return trades, b.SendHTTPRequest(btcMarketsUnauthPath+marketID+btcMarketsGetTrades+params.Encode(),
		&trades)
}

// GetOrderbook returns current orderbook
func (b *BTCMarkets) GetOrderbook(marketID string, level int64) (Orderbook, error) {
	var orderbook Orderbook
	var temp tempOrderbook
	params := url.Values{}
	if level != 0 {
		params.Set("level", strconv.FormatInt(level, 10))
	}
	err := b.SendHTTPRequest(btcMarketsUnauthPath+marketID+btcMarketOrderBooks+params.Encode(),
		&temp)
	if err != nil {
		return orderbook, err
	}

	orderbook.MarketID = temp.MarketID
	orderbook.SnapshotID = temp.SnapshotID
	for x := range temp.Asks {
		price, err := strconv.ParseFloat(temp.Asks[x][0], 64)
		if err != nil {
			return orderbook, err
		}
		amount, err := strconv.ParseFloat(temp.Asks[x][1], 64)
		if err != nil {
			return orderbook, err
		}
		orderbook.Asks = append(orderbook.Asks, OBData{
			Price:  price,
			Volume: amount,
		})
	}
	for a := range temp.Bids {
		price, err := strconv.ParseFloat(temp.Bids[a][0], 64)
		if err != nil {
			return orderbook, err
		}
		amount, err := strconv.ParseFloat(temp.Bids[a][1], 64)
		if err != nil {
			return orderbook, err
		}
		orderbook.Bids = append(orderbook.Bids, OBData{
			Price:  price,
			Volume: amount,
		})
	}
	return orderbook, nil
}

// GetMarketCandles gets candles for specified currency pair
func (b *BTCMarkets) GetMarketCandles(marketID, timeWindow string, from, to time.Time, before, after, limit int64) (out CandleResponse, err error) {
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
	return out, b.SendHTTPRequest(btcMarketsUnauthPath+marketID+btcMarketsCandles+params.Encode(), &out)
}

// GetTickers gets multiple tickers
func (b *BTCMarkets) GetTickers(marketIDs currency.Pairs) ([]Ticker, error) {
	var tickers []Ticker
	params := url.Values{}
	for x := range marketIDs {
		params.Add("marketId", marketIDs[x].String())
	}
	return tickers, b.SendHTTPRequest(btcMarketsUnauthPath+btcMarketsTickers+params.Encode(),
		&tickers)
}

// GetMultipleOrderbooks gets orderbooks
func (b *BTCMarkets) GetMultipleOrderbooks(marketIDs []string) ([]Orderbook, error) {
	var orderbooks []Orderbook
	var temp []tempOrderbook
	var tempOB Orderbook
	params := url.Values{}
	for x := range marketIDs {
		params.Add("marketId", marketIDs[x])
	}
	err := b.SendHTTPRequest(btcMarketsUnauthPath+btcMarketsMultipleOrderbooks+params.Encode(),
		&temp)
	if err != nil {
		return orderbooks, err
	}
	for i := range temp {
		var price, volume float64
		tempOB.MarketID = temp[i].MarketID
		tempOB.SnapshotID = temp[i].SnapshotID
		for a := range temp[i].Asks {
			volume, err = strconv.ParseFloat(temp[i].Asks[a][1], 64)
			if err != nil {
				return orderbooks, err
			}
			price, err = strconv.ParseFloat(temp[i].Asks[a][0], 64)
			if err != nil {
				return orderbooks, err
			}
			tempOB.Asks = append(tempOB.Asks, OBData{Price: price, Volume: volume})
		}
		for y := range temp[i].Bids {
			volume, err = strconv.ParseFloat(temp[i].Bids[y][1], 64)
			if err != nil {
				return orderbooks, err
			}
			price, err = strconv.ParseFloat(temp[i].Bids[y][0], 64)
			if err != nil {
				return orderbooks, err
			}
			tempOB.Bids = append(tempOB.Bids, OBData{Price: price, Volume: volume})
		}
		orderbooks = append(orderbooks, tempOB)
	}
	return orderbooks, nil
}

// GetServerTime gets time from btcmarkets
func (b *BTCMarkets) GetServerTime() (time.Time, error) {
	var resp TimeResp
	return resp.Time, b.SendHTTPRequest(btcMarketsAPIURL+btcMarketsAPIVersion+btcMarketsGetTime,
		&resp)
}

// GetAccountBalance returns the full account balance
func (b *BTCMarkets) GetAccountBalance() ([]AccountData, error) {
	var resp []AccountData
	return resp,
		b.SendAuthenticatedRequest(http.MethodGet,
			btcMarketsAccountBalance,
			nil,
			&resp,
			request.Auth)
}

// GetTradingFees returns trading fees for all pairs based on trading activity
func (b *BTCMarkets) GetTradingFees() (TradingFeeResponse, error) {
	var resp TradingFeeResponse
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsTradingFees,
		nil,
		&resp,
		request.Auth)
}

// GetTradeHistory returns trade history
func (b *BTCMarkets) GetTradeHistory(marketID, orderID string, before, after, limit int64) ([]TradeHistoryData, error) {
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
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		common.EncodeURLValues(btcMarketsTradeHistory, params),
		nil,
		&resp,
		request.Auth)
}

// GetTradeByID returns the singular trade of the ID given
func (b *BTCMarkets) GetTradeByID(id string) (TradeHistoryData, error) {
	var resp TradeHistoryData
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsTradeHistory+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// NewOrder requests a new order and returns an ID
func (b *BTCMarkets) NewOrder(marketID string, price, amount float64, orderType, side string, triggerPrice,
	targetAmount float64, timeInForce string, postOnly bool, selfTrade, clientOrderID string) (OrderData, error) {
	var resp OrderData
	req := make(map[string]interface{})
	req["marketId"] = marketID
	req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
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
	req["postOnly"] = postOnly
	if selfTrade != "" {
		req["selfTrade"] = selfTrade
	}
	if clientOrderID != "" {
		req["clientOrderID"] = clientOrderID
	}
	return resp, b.SendAuthenticatedRequest(http.MethodPost,
		btcMarketsOrders,
		req,
		&resp,
		orderFunc)
}

// GetOrders returns current order information on the exchange
func (b *BTCMarkets) GetOrders(marketID string, before, after, limit int64, openOnly bool) ([]OrderData, error) {
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
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		common.EncodeURLValues(btcMarketsOrders, params),
		nil,
		&resp,
		request.Auth)
}

// CancelAllOpenOrdersByPairs cancels all open orders unless pairs are specified
func (b *BTCMarkets) CancelAllOpenOrdersByPairs(marketIDs []string) ([]CancelOrderResp, error) {
	var resp []CancelOrderResp
	req := make(map[string]interface{})
	if len(marketIDs) > 0 {
		var strTemp strings.Builder
		for x := range marketIDs {
			strTemp.WriteString("marketId=" + marketIDs[x] + "&")
		}
		req["marketId"] = strTemp.String()[:strTemp.Len()-1]
	}
	return resp, b.SendAuthenticatedRequest(http.MethodDelete,
		btcMarketsOrders,
		req,
		&resp,
		request.Auth)
}

// FetchOrder finds order based on the provided id
func (b *BTCMarkets) FetchOrder(id string) (OrderData, error) {
	var resp OrderData
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsOrders+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// RemoveOrder removes a given order
func (b *BTCMarkets) RemoveOrder(id string) (CancelOrderResp, error) {
	var resp CancelOrderResp
	return resp, b.SendAuthenticatedRequest(http.MethodDelete,
		btcMarketsOrders+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// ListWithdrawals lists the withdrawal history
func (b *BTCMarkets) ListWithdrawals(before, after, limit int64) ([]TransferData, error) {
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
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		common.EncodeURLValues(btcMarketsWithdrawals, params),
		nil,
		&resp,
		request.Auth)
}

// GetWithdrawal gets withdrawawl info for a given id
func (b *BTCMarkets) GetWithdrawal(id string) (TransferData, error) {
	var resp TransferData
	if id == "" {
		return resp, errors.New("id cannot be an empty string")
	}
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsWithdrawals+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// ListDeposits lists the deposit history
func (b *BTCMarkets) ListDeposits(before, after, limit int64) ([]TransferData, error) {
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
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		common.EncodeURLValues(btcMarketsDeposits, params),
		nil,
		&resp,
		request.Auth)
}

// GetDeposit gets deposit info for a given ID
func (b *BTCMarkets) GetDeposit(id string) (TransferData, error) {
	var resp TransferData
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsDeposits+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// ListTransfers lists the past asset transfers
func (b *BTCMarkets) ListTransfers(before, after, limit int64) ([]TransferData, error) {
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
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		common.EncodeURLValues(btcMarketsTransfers, params),
		nil,
		&resp,
		request.Auth)
}

// GetTransfer gets asset transfer info for a given ID
func (b *BTCMarkets) GetTransfer(id string) (TransferData, error) {
	var resp TransferData
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsTransfers+"/"+id,
		nil,
		&resp,
		request.Auth)
}

// FetchDepositAddress gets deposit address for the given asset
func (b *BTCMarkets) FetchDepositAddress(assetName string, before, after, limit int64) (DepositAddress, error) {
	var resp DepositAddress
	if (before > 0) && (after >= 0) {
		return resp, errors.New("BTCMarkets only supports either before or after, not both")
	}
	params := url.Values{}
	params.Set("assetName", assetName)
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after >= 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		common.EncodeURLValues(btcMarketsAddresses, params),
		nil,
		&resp,
		request.Auth)
}

// GetWithdrawalFees gets withdrawal fees for all assets
func (b *BTCMarkets) GetWithdrawalFees() ([]WithdrawalFeeData, error) {
	var resp []WithdrawalFeeData
	return resp, b.SendHTTPRequest(btcMarketsAPIURL+btcMarketsAPIVersion+btcMarketsWithdrawalFees,
		&resp)
}

// ListAssets lists all available assets
func (b *BTCMarkets) ListAssets() ([]AssetData, error) {
	var resp []AssetData
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsAssets,
		nil,
		&resp,
		request.Auth)
}

// GetTransactions gets trading fees
func (b *BTCMarkets) GetTransactions(assetName string, before, after, limit int64) ([]TransactionData, error) {
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
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		common.EncodeURLValues(btcMarketsTransactions, params),
		nil,
		&resp,
		request.Auth)
}

// CreateNewReport creates a new report
func (b *BTCMarkets) CreateNewReport(reportType, format string) (CreateReportResp, error) {
	var resp CreateReportResp
	req := make(map[string]interface{})
	req["type"] = reportType
	req["format"] = format
	return resp, b.SendAuthenticatedRequest(http.MethodPost,
		btcMarketsReports,
		req,
		&resp,
		newReportFunc)
}

// GetReport finds details bout a past report
func (b *BTCMarkets) GetReport(reportID string) (ReportData, error) {
	var resp ReportData
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsReports+"/"+reportID,
		nil,
		&resp,
		request.Auth)
}

// RequestWithdraw requests withdrawals
func (b *BTCMarkets) RequestWithdraw(assetName string, amount float64,
	toAddress, accountName, accountNumber, bsbNumber, bankName string) (TransferData, error) {
	var resp TransferData
	req := make(map[string]interface{})
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
	return resp, b.SendAuthenticatedRequest(http.MethodPost,
		btcMarketsWithdrawals,
		req,
		&resp,
		withdrawFunc)
}

// BatchPlaceCancelOrders places and cancels batch orders
func (b *BTCMarkets) BatchPlaceCancelOrders(cancelOrders []CancelBatch, placeOrders []PlaceBatch) (BatchPlaceCancelResponse, error) {
	var resp BatchPlaceCancelResponse
	var orderRequests []interface{}
	if len(cancelOrders)+len(placeOrders) > 4 {
		return resp, errors.New("BTCMarkets can only handle 4 orders at a time")
	}
	for x := range cancelOrders {
		orderRequests = append(orderRequests, CancelOrderMethod{CancelOrder: cancelOrders[x]})
	}
	for y := range placeOrders {
		if placeOrders[y].ClientOrderID == "" {
			return resp, errors.New("placeorders must have clientorderids filled")
		}
		orderRequests = append(orderRequests, PlaceOrderMethod{PlaceOrder: placeOrders[y]})
	}
	return resp, b.SendAuthenticatedRequest(http.MethodPost,
		btcMarketsBatchOrders,
		orderRequests,
		&resp,
		batchFunc)
}

// GetBatchTrades gets batch trades
func (b *BTCMarkets) GetBatchTrades(ids []string) (BatchTradeResponse, error) {
	var resp BatchTradeResponse
	if len(ids) > 50 {
		return resp, errors.New("batchtrades can only handle 50 ids at a time")
	}
	marketIDs := strings.Join(ids, ",")
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsBatchOrders+"/"+marketIDs,
		nil,
		&resp,
		request.Auth)
}

// CancelBatch cancels given ids
func (b *BTCMarkets) CancelBatch(ids []string) (BatchCancelResponse, error) {
	var resp BatchCancelResponse
	marketIDs := strings.Join(ids, ",")
	return resp, b.SendAuthenticatedRequest(http.MethodDelete,
		btcMarketsBatchOrders+"/"+marketIDs,
		nil,
		&resp,
		batchFunc)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *BTCMarkets) SendHTTPRequest(path string, result interface{}) error {
	return b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          path,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}

// SendAuthenticatedRequest sends an authenticated HTTP request
func (b *BTCMarkets) SendAuthenticatedRequest(method, path string, data, result interface{}, f request.EndpointLimit) (err error) {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", b.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}

	now := time.Now()
	strTime := strconv.FormatInt(now.UTC().UnixNano()/1000000, 10)

	var body io.Reader
	var payload, hmac []byte
	switch data.(type) {
	case map[string]interface{}, []interface{}:
		payload, err = json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
		strMsg := method + btcMarketsAPIVersion + path + strTime + string(payload)
		hmac = crypto.GetHMAC(crypto.HashSHA512,
			[]byte(strMsg), []byte(b.API.Credentials.Secret))
	default:
		strArray := strings.Split(path, "?")
		hmac = crypto.GetHMAC(crypto.HashSHA512,
			[]byte(method+btcMarketsAPIVersion+strArray[0]+strTime),
			[]byte(b.API.Credentials.Secret))
	}

	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Accept-Charset"] = "UTF-8"
	headers["Content-Type"] = "application/json"
	headers["BM-AUTH-APIKEY"] = b.API.Credentials.Key
	headers["BM-AUTH-TIMESTAMP"] = strTime
	headers["BM-AUTH-SIGNATURE"] = crypto.Base64Encode(hmac)

	// The timestamp included with an authenticated request must be within +/- 30 seconds of the server timestamp
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(30*time.Second))
	defer cancel()
	return b.SendPayload(ctx, &request.Item{
		Method:        method,
		Path:          btcMarketsAPIURL + btcMarketsAPIVersion + path,
		Headers:       headers,
		Body:          body,
		Result:        result,
		AuthRequest:   true,
		NonceEnabled:  false,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
		Endpoint:      f,
	})
}

// GetFee returns an estimate of fee based on type of transaction
func (b *BTCMarkets) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		temp, err := b.GetTradingFees()
		if err != nil {
			return fee, err
		}
		for x := range temp.FeeByMarkets {
			p, err := currency.NewPairFromString(temp.FeeByMarkets[x].MarketID)
			if err != nil {
				return 0, err
			}
			if p == feeBuilder.Pair {
				fee = temp.FeeByMarkets[x].MakerFeeRate
				if !feeBuilder.IsMaker {
					fee = temp.FeeByMarkets[x].TakerFeeRate
				}
			}
		}
	case exchange.CryptocurrencyWithdrawalFee:
		temp, err := b.GetWithdrawalFees()
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
