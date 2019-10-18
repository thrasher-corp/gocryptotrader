package btcmarkets

import (
	"bytes"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
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
	btcMarketsTickers            = "/tickers?"
	btcMarketsMultipleOrderbooks = "/orderbooks?"
	btcMarketsGetTime            = "/time"
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
	btcMarketsWithdrawalFees = "/withdrawal-fees"
	btcMarketsAssets         = "/assets"
	btcMarketsReports        = "/reports"
	btcMarketsBatchOrders    = "/batchorders"

	btcmarketsAuthLimit   = 10
	btcmarketsUnauthLimit = 25
)

// BTCMarkets is the overarching type across the BTCMarkets package
type BTCMarkets struct {
	exchange.Base
	WebsocketConn *wshandler.WebsocketConnection
}

// GetMarkets returns the BTCMarkets instruments
func (b *BTCMarkets) GetMarkets() ([]Market, error) {
	var resp []Market
	path := btcMarketsAPIURL + btcMarketsAPIVersion + btcMarketsAllMarkets
	return resp, b.SendHTTPRequest(path, &resp)
}

// GetTicker returns a ticker
// symbol - example "btc" or "ltc"
func (b *BTCMarkets) GetTicker(marketID string) (Ticker, error) {
	var tick Ticker
	path := btcMarketsUnauthPath + marketID + btcMarketsGetTicker
	return tick, b.SendHTTPRequest(path, &tick)
}

// GetTrades returns executed trades on the exchange
func (b *BTCMarkets) GetTrades(marketID string, before, after, limit int64) ([]Trade, error) {
	var trades []Trade
	params := url.Values{}
	if before != 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after != 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := btcMarketsUnauthPath + marketID + btcMarketsGetTrades + params.Encode()
	return trades, b.SendHTTPRequest(path, &trades)
}

// GetOrderbook returns current orderbook
func (b *BTCMarkets) GetOrderbook(marketID string, level int64) (Orderbook, error) {
	var orderbook Orderbook
	var temp tempOrderbook
	params := url.Values{}
	if level != 0 {
		params.Set("level", strconv.FormatInt(level, 10))
	}
	path := btcMarketsUnauthPath + marketID + btcMarketOrderBooks + params.Encode()
	err := b.SendHTTPRequest(path, &temp)
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
func (b *BTCMarkets) GetMarketCandles(marketID, timeWindow, from, to string, before, after, limit int64) ([]MarketCandle, error) {
	var marketCandles []MarketCandle
	var temp [][]interface{}
	params := url.Values{}
	if timeWindow != "" {
		params.Set("timeWindow", timeWindow)
	}
	if from != "" {
		params.Set("from", from)
	}
	if to != "" {
		params.Set("to", to)
	}
	if before != 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after != 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := btcMarketsUnauthPath + marketID + btcMarketsCandles + params.Encode()
	err := b.SendHTTPRequest(path, &temp)
	if err != nil {
		return marketCandles, err
	}
	var tempData MarketCandle
	for x := range temp {
		tempData.Time = temp[x][0].(string)
		tempData.Open, err = strconv.ParseFloat(temp[x][1].(string), 64)
		if err != nil {
			return marketCandles, err
		}
		tempData.High, err = strconv.ParseFloat(temp[x][2].(string), 64)
		if err != nil {
			return marketCandles, err
		}
		tempData.Low, err = strconv.ParseFloat(temp[x][3].(string), 64)
		if err != nil {
			return marketCandles, err
		}
		tempData.Close, err = strconv.ParseFloat(temp[x][4].(string), 64)
		if err != nil {
			return marketCandles, err
		}
		tempData.Volume, err = strconv.ParseFloat(temp[x][5].(string), 64)
		if err != nil {
			return marketCandles, err
		}
		marketCandles = append(marketCandles, tempData)
	}
	return marketCandles, nil
}

// GetTickers gets multiple tickers
func (b *BTCMarkets) GetTickers(marketIDs string) ([]Ticker, error) {
	var tickers []Ticker
	arrayMarkets := strings.Split(marketIDs, ",")
	params := url.Values{}
	for x := range arrayMarkets {
		params.Add("marketId", arrayMarkets[x])
	}
	path := btcMarketsUnauthPath + btcMarketsTickers + params.Encode()
	return tickers, b.SendHTTPRequest(path, &tickers)
}

// GetMultipleOrderbooks gets orderbooks
func (b *BTCMarkets) GetMultipleOrderbooks(marketIDs string) ([]Orderbook, error) {
	var orderbooks []Orderbook
	var temp []tempOrderbook
	var tempOB Orderbook
	arrayMarkets := strings.Split(marketIDs, ",")
	params := url.Values{}
	for x := range arrayMarkets {
		params.Add("marketId", arrayMarkets[x])
	}
	path := btcMarketsUnauthPath + btcMarketsMultipleOrderbooks + params.Encode()
	err := b.SendHTTPRequest(path, &temp)
	if err != nil {
		return orderbooks, err
	}
	for i := range temp {
		var obData OBData
		tempOB.MarketID = temp[i].MarketID
		tempOB.SnapshotID = temp[i].SnapshotID
		for a := range temp[i].Asks {
			obData.Price, err = strconv.ParseFloat(temp[i].Asks[a][0], 64)
			if err != nil {
				return orderbooks, err
			}
			obData.Volume, err = strconv.ParseFloat(temp[i].Asks[a][1], 64)
			if err != nil {
				return orderbooks, err
			}
			tempOB.Asks = append(tempOB.Asks, obData)
		}
		for y := range temp[i].Bids {
			obData.Price, err = strconv.ParseFloat(temp[i].Bids[y][0], 64)
			if err != nil {
				return orderbooks, err
			}
			obData.Volume, err = strconv.ParseFloat(temp[i].Bids[y][1], 64)
			if err != nil {
				return orderbooks, err
			}
			tempOB.Bids = append(tempOB.Bids, obData)
		}
		orderbooks = append(orderbooks, tempOB)
	}
	return orderbooks, nil
}

// GetServerTime gets time from btcmarkets
func (b *BTCMarkets) GetServerTime() (string, time.Time, error) {
	var tempResp TimeResp
	var t time.Time
	path := btcMarketsAPIURL + btcMarketsAPIVersion + btcMarketsGetTime
	err := b.SendHTTPRequest(path, &tempResp)
	if err != nil {
		return tempResp.Time, t, err
	}
	t, err = time.Parse(time.RFC3339Nano, tempResp.Time)
	if err != nil {
		return tempResp.Time, t, err
	}
	return tempResp.Time, t, nil
}

// GetAccountBalance returns the full account balance
func (b *BTCMarkets) GetAccountBalance() ([]AccountData, error) {
	var resp []AccountData
	return resp,
		b.SendAuthenticatedRequest(http.MethodGet,
			btcMarketsAccountBalance,
			nil,
			&resp,
			nil)
}

// GetTradingFees returns trading fees for all pairs based on trading activity
func (b *BTCMarkets) GetTradingFees() (TradingFeeResponse, error) {
	var resp TradingFeeResponse
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsTradingFees,
		nil,
		&resp,
		nil)
}

// GetTradeHistory returns trade history
func (b *BTCMarkets) GetTradeHistory(marketID, orderID, before, after, limit string) ([]TradeHistoryData, error) {
	var resp []TradeHistoryData
	req := make(map[string]interface{})
	if marketID != "" {
		req["marketId"] = marketID
	}
	if orderID != "" {
		req["orderId"] = orderID
	}
	if before != "" {
		req["before"] = before
	}
	if after != "" {
		req["after"] = after
	}
	if limit != "" {
		req["limit"] = limit
	}
	return resp, b.SendAuthenticatedRequest(http.MethodGet, btcMarketsTradeHistory, req, &resp, nil)
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
	if orderType == "Stop Limit" || orderType == "Take Profit" || orderType == "Stop" {
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
	return resp, b.SendAuthenticatedRequest(http.MethodPost, btcMarketsOrders, req, &resp, nil)
}

// GetOrders returns current order information on the exchange
func (b *BTCMarkets) GetOrders(marketID, before, after, limit, status string) ([]OrderData, error) {
	var resp []OrderData
	req := make(map[string]interface{})

	if marketID != "" {
		req["marketId"] = marketID
	}
	if before != "" {
		req["before"] = before
	}
	if after != "" {
		req["after"] = after
	}
	if limit != "" {
		req["limit"] = limit
	}
	if status != "" {
		req["status"] = status
	}

	return resp, b.SendAuthenticatedRequest(http.MethodGet, btcMarketsOrders, req, &resp, nil)
}

// CancelOpenOrders cancels all open orders unless pairs are specified
func (b *BTCMarkets) CancelOpenOrders(marketIDs string) ([]CancelOrderResp, error) {
	var resp []CancelOrderResp
	req := make(map[string]interface{})
	if marketIDs != "" {
		pairArray := strings.Split(marketIDs, ",")
		var strTemp string
		for x := range pairArray {
			strTemp += fmt.Sprintf("marketId=%s&", pairArray[x])
		}
		req["marketId"] = strTemp[:len(strTemp)-1]
	}
	return resp, b.SendAuthenticatedRequest(http.MethodDelete, btcMarketsOrders, req, &resp, nil)
}

// FetchOrder finds order based on the provided id
func (b *BTCMarkets) FetchOrder(id string) (OrderData, error) {
	var resp OrderData
	path := btcMarketsOrders + "/" + id
	return resp, b.SendAuthenticatedRequest(http.MethodGet, path, nil, &resp, nil)
}

// RemoveOrder removes a given order
func (b *BTCMarkets) RemoveOrder(id string) (CancelOrderResp, error) {
	var resp CancelOrderResp
	path := btcMarketsOrders + "/" + id
	return resp, b.SendAuthenticatedRequest(http.MethodDelete, path, nil, &resp, nil)
}

// ListWithdrawals lists the withdrawal history
func (b *BTCMarkets) ListWithdrawals() ([]TransferData, error) {
	var resp []TransferData
	return resp, b.SendAuthenticatedRequest(http.MethodGet, btcMarketsWithdrawals, nil, &resp, nil)
}

// GetWithdrawal gets withdrawawl info for a given id
func (b *BTCMarkets) GetWithdrawal(id string) (TransferData, error) {
	var resp TransferData
	path := btcMarketsWithdrawals + "/" + id
	return resp, b.SendAuthenticatedRequest(http.MethodGet, path, nil, &resp, nil)
}

// ListDeposits lists the deposit history
func (b *BTCMarkets) ListDeposits() ([]TransferData, error) {
	var resp []TransferData
	return resp, b.SendAuthenticatedRequest(http.MethodGet, btcMarketsDeposits, nil, &resp, nil)
}

// GetDeposit gets deposit info for a given ID
func (b *BTCMarkets) GetDeposit(id string) (TransferData, error) {
	var resp TransferData
	path := btcMarketsDeposits + "/" + id
	return resp, b.SendAuthenticatedRequest(http.MethodGet, path, nil, &resp, nil)
}

// ListTransfers lists the past asset transfers
func (b *BTCMarkets) ListTransfers() ([]TransferData, error) {
	var resp []TransferData
	return resp, b.SendAuthenticatedRequest(http.MethodGet, btcMarketsTransfers, nil, &resp, nil)
}

// GetTransfer gets asset transfer info for a given ID
func (b *BTCMarkets) GetTransfer(id string) (TransferData, error) {
	var resp TransferData
	path := btcMarketsTransfers + "/" + id
	return resp, b.SendAuthenticatedRequest(http.MethodGet, path, nil, &resp, nil)
}

// FetchDepositAddress gets deposit address for the given asset
func (b *BTCMarkets) FetchDepositAddress(assetName string) (DepositAddress, error) {
	var resp DepositAddress
	req := make(map[string]interface{})
	req["assetName"] = assetName
	return resp, b.SendAuthenticatedRequest(http.MethodGet,
		btcMarketsAddresses,
		req,
		&resp,
		nil)
}

// GetWithdrawalFees gets withdrawal fees for all assets
func (b *BTCMarkets) GetWithdrawalFees() ([]WithdrawalFeeData, error) {
	var resp []WithdrawalFeeData
	path := btcMarketsAPIURL + btcMarketsAPIVersion + btcMarketsWithdrawalFees
	return resp, b.SendHTTPRequest(path, &resp)
}

// ListAssets lists all available assets
func (b *BTCMarkets) ListAssets() ([]AssetData, error) {
	var resp []AssetData
	return resp, b.SendAuthenticatedRequest(http.MethodGet, btcMarketsAssets, nil, &resp, nil)
}

// GetTransactions gets trading fees
func (b *BTCMarkets) GetTransactions(assetName string) ([]TransactionData, error) {
	var resp []TransactionData
	req := make(map[string]interface{})
	req["assetName"] = assetName
	return resp, b.SendAuthenticatedRequest(http.MethodGet, btcMarketsTransactions, req, &resp, nil)
}

// CreateNewReport creates a new report
func (b *BTCMarkets) CreateNewReport(reportType, format string) (ReportData, error) {
	var resp ReportData
	req := make(map[string]interface{})
	req["type"] = reportType
	req["format"] = format
	return resp, b.SendAuthenticatedRequest(http.MethodPost, btcMarketsReports, req, &resp, nil)
}

// GetReport creates a new report
func (b *BTCMarkets) GetReport(reportID string) (ReportData, error) {
	var resp ReportData
	path := btcMarketsReports + "/" + reportID
	return resp, b.SendAuthenticatedRequest(http.MethodGet, path, nil, &resp, nil)
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
	return resp, b.SendAuthenticatedRequest(http.MethodPost, btcMarketsWithdrawals, req, &resp, nil)
}

// BatchPlaceCancelOrders places and cancels batch orders
func (b *BTCMarkets) BatchPlaceCancelOrders(cancelOrders []cancelBatch, placeOrders []placeBatch) (BatchPlaceCancelResponse, error) {
	var resp BatchPlaceCancelResponse
	var orderRequests []interface{}
	for x := range cancelOrders {
		orderRequests = append(orderRequests, cancelOrders[x])
	}
	for y := range placeOrders {
		orderRequests = append(orderRequests, placeOrders[y])
	}
	return resp, b.SendAuthenticatedRequest(http.MethodPost, btcMarketsBatchOrders, nil, &resp, orderRequests)
}

// GetBatchTrades gets batch trades
// ids MUST be comma separated with no spaces
func (b *BTCMarkets) GetBatchTrades(ids string) (BatchTradeResponse, error) {
	var resp BatchTradeResponse
	path := btcMarketsBatchOrders + "/" + ids
	return resp, b.SendAuthenticatedRequest(http.MethodGet, path, nil, &resp, nil)
}

// CancelBatchOrders cancels given ids
// ids MUST be comma separated with no spaces
func (b *BTCMarkets) CancelBatchOrders(ids string) (BatchCancelResponse, error) {
	var resp BatchCancelResponse
	path := btcMarketsBatchOrders + "/" + ids
	return resp, b.SendAuthenticatedRequest(http.MethodDelete, path, nil, &resp, nil)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *BTCMarkets) SendHTTPRequest(path string, result interface{}) error {
	return b.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		&result,
		false,
		false,
		b.Verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
}

// SendAuthenticatedRequest sends an authenticated HTTP request
func (b *BTCMarkets) SendAuthenticatedRequest(method, path string, data map[string]interface{}, result interface{}, specialCase []interface{}) (err error) {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			b.Name)
	}

	strTime := strconv.FormatInt(time.Now().UTC().UnixNano()/1000000, 10)

	var body io.Reader
	var payload, hmac []byte
	switch {
	case len(data) != 0:
		payload, err = common.JSONEncode(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
		strMsg := method + btcMarketsAPIVersion + path + strTime + string(payload)
		hmac = crypto.GetHMAC(crypto.HashSHA512,
			[]byte(strMsg), []byte(b.API.Credentials.Secret))
	case specialCase != nil:
		payload, err = common.JSONEncode(specialCase)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
		strMsg := method + btcMarketsAPIVersion + path + strTime + string(payload)
		hmac = crypto.GetHMAC(crypto.HashSHA512,
			[]byte(strMsg), []byte(b.API.Credentials.Secret))
	default:
		hmac = crypto.GetHMAC(crypto.HashSHA512,
			[]byte(method+btcMarketsAPIVersion+path+strTime+string(payload)),
			[]byte(b.API.Credentials.Secret))
	}

	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Accept-Charset"] = "UTF-8"
	headers["Content-Type"] = "application/json"
	headers["BM-AUTH-APIKEY"] = b.API.Credentials.Key
	headers["BM-AUTH-TIMESTAMP"] = strTime
	headers["BM-AUTH-SIGNATURE"] = crypto.Base64Encode(hmac)
	return b.SendPayload(method,
		btcMarketsAPIURL+btcMarketsAPIVersion+path,
		headers,
		body,
		result,
		true,
		false,
		b.Verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
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
			if currency.NewPairFromString(temp.FeeByMarkets[x].MarketID) == feeBuilder.Pair {
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
		return 0, errors.New("Internation bank withdrawals are not supported")

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
	return 0.0085 * price * amount
}
