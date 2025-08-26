package kraken

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
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

var errInvalidBatchOrderType = errors.New("invalid batch order type")

// GetFuturesOrderbook gets orderbook data for futures
func (e *Exchange) GetFuturesOrderbook(ctx context.Context, symbol currency.Pair) (*FuturesOrderbookData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp FuturesOrderbookData
	return &resp, e.SendHTTPRequest(ctx, exchange.RestFutures, futuresOrderbook+"?"+params.Encode(), &resp)
}

// GetFuturesCharts returns candle data for kraken futures
func (e *Exchange) GetFuturesCharts(ctx context.Context, resolution, tickType string, symbol currency.Pair, to, from time.Time) (*FuturesCandles, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	reqStr := futuresCandles + tickType + "/" + symbolValue + "/" + resolution
	if len(params) > 0 {
		reqStr += "?" + params.Encode()
	}
	var resp FuturesCandles
	return &resp, e.SendHTTPRequest(ctx, exchange.RestFuturesSupplementary, reqStr, &resp)
}

// GetFuturesTrades returns public trade data for kraken futures
func (e *Exchange) GetFuturesTrades(ctx context.Context, symbol currency.Pair, to, from time.Time) (*FuturesPublicTrades, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if !to.IsZero() {
		params.Set("since", strconv.FormatInt(to.Unix(), 10))
	}
	if !from.IsZero() {
		params.Set("before", strconv.FormatInt(from.Unix(), 10))
	}
	var resp FuturesPublicTrades
	return &resp, e.SendHTTPRequest(ctx, exchange.RestFuturesSupplementary, futuresPublicTrades+"/"+symbolValue+"/executions?"+params.Encode(), &resp)
}

// GetInstruments gets a list of futures markets and their data
func (e *Exchange) GetInstruments(ctx context.Context) (FuturesInstrumentData, error) {
	var resp FuturesInstrumentData
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, futuresInstruments, &resp)
}

// GetFuturesTickers gets a list of futures tickers and their data
func (e *Exchange) GetFuturesTickers(ctx context.Context) (FuturesTickersData, error) {
	var resp FuturesTickersData
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, futuresTickers, &resp)
}

// GetFuturesTickerBySymbol returns futures ticker data by symbol
func (e *Exchange) GetFuturesTickerBySymbol(ctx context.Context, symbol string) (FuturesTickerData, error) {
	var resp FuturesTickerData
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, futuresTickers+"/"+symbol, &resp)
}

// GetFuturesTradeHistory gets public trade history data for futures
func (e *Exchange) GetFuturesTradeHistory(ctx context.Context, symbol currency.Pair, lastTime time.Time) (FuturesTradeHistoryData, error) {
	var resp FuturesTradeHistoryData
	params := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !lastTime.IsZero() {
		params.Set("lastTime", lastTime.Format("2006-01-02T15:04:05.070Z"))
	}
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, futuresTradeHistory+"?"+params.Encode(), &resp)
}

// FuturesBatchOrder places a batch order for futures
func (e *Exchange) FuturesBatchOrder(ctx context.Context, data []PlaceBatchOrderData) (BatchOrderData, error) {
	var resp BatchOrderData
	for x := range data {
		unformattedPair, err := currency.NewPairFromString(data[x].Symbol)
		if err != nil {
			return resp, err
		}
		formattedPair, err := e.FormatExchangeCurrency(unformattedPair, asset.Futures)
		if err != nil {
			return resp, err
		}
		if !slices.Contains(validBatchOrderType, data[x].PlaceOrderType) {
			return resp, fmt.Errorf("%s %w",
				data[x].PlaceOrderType,
				errInvalidBatchOrderType)
		}
		data[x].Symbol = formattedPair.String()
	}

	req := make(map[string]any)
	req["batchOrder"] = data

	jsonData, err := json.Marshal(req)
	if err != nil {
		return resp, err
	}

	params := url.Values{}
	params.Set("json", string(jsonData))
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodPost, futuresBatchOrder, params, &resp)
}

// FuturesEditOrder edits a futures order
func (e *Exchange) FuturesEditOrder(ctx context.Context, orderID, clientOrderID string, size, limitPrice, stopPrice float64) (FuturesAccountsData, error) {
	var resp FuturesAccountsData
	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if clientOrderID != "" {
		params.Set("cliOrderId", clientOrderID)
	}
	params.Set("size", strconv.FormatFloat(size, 'f', -1, 64))
	params.Set("limitPrice", strconv.FormatFloat(limitPrice, 'f', -1, 64))
	params.Set("stopPrice", strconv.FormatFloat(stopPrice, 'f', -1, 64))
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodPost, futuresEditOrder, params, &resp)
}

// FuturesSendOrder sends a futures order
func (e *Exchange) FuturesSendOrder(ctx context.Context, orderType order.Type, symbol currency.Pair, side, triggerSignal, clientOrderID, reduceOnly string, tif order.TimeInForce, size, limitPrice, stopPrice float64) (FuturesSendOrderData, error) {
	var resp FuturesSendOrderData
	oType, ok := validOrderTypes[orderType]
	if !ok {
		return resp, errors.New("invalid orderType")
	}

	if oType != "mkt" {
		if tif.Is(order.PostOnly) {
			oType = "post"
		} else if tif.Is(order.ImmediateOrCancel) {
			oType = "ioc"
		}
	}

	params := url.Values{}
	params.Set("orderType", oType)
	symbolValue, err := e.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !slices.Contains(validSide, side) {
		return resp, errors.New("invalid side")
	}
	params.Set("side", side)
	if triggerSignal != "" {
		if !slices.Contains(validTriggerSignal, triggerSignal) {
			return resp, errors.New("invalid triggerSignal")
		}
		params.Set("triggerSignal", triggerSignal)
	}
	if clientOrderID != "" {
		params.Set("cliOrdId", clientOrderID)
	}
	if reduceOnly != "" {
		if !slices.Contains(validReduceOnly, reduceOnly) {
			return resp, errors.New("invalid reduceOnly")
		}
		params.Set("reduceOnly", reduceOnly)
	}
	params.Set("size", strconv.FormatFloat(size, 'f', -1, 64))
	params.Set("limitPrice", strconv.FormatFloat(limitPrice, 'f', -1, 64))
	if stopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(stopPrice, 'f', -1, 64))
	}
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodPost, futuresSendOrder, params, &resp)
}

// FuturesCancelOrder cancels an order
func (e *Exchange) FuturesCancelOrder(ctx context.Context, orderID, clientOrderID string) (FuturesCancelOrderData, error) {
	var resp FuturesCancelOrderData
	params := url.Values{}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if clientOrderID != "" {
		params.Set("cliOrdId", clientOrderID)
	}
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodPost, futuresCancelOrder, params, &resp)
}

// FuturesGetFills gets order fills for futures
func (e *Exchange) FuturesGetFills(ctx context.Context, lastFillTime time.Time) (FuturesFillsData, error) {
	var resp FuturesFillsData
	params := url.Values{}
	if !lastFillTime.IsZero() {
		params.Set("lastFillTime", lastFillTime.UTC().Format("2006-01-02T15:04:05.999Z"))
	}
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodGet, futuresOrderFills, params, &resp)
}

// FuturesTransfer transfers funds between accounts
func (e *Exchange) FuturesTransfer(ctx context.Context, fromAccount, toAccount, unit string, amount float64) (FuturesTransferData, error) {
	var resp FuturesTransferData
	req := make(map[string]any)
	req["fromAccount"] = fromAccount
	req["toAccount"] = toAccount
	req["unit"] = unit
	req["amount"] = amount
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodPost, futuresTransfer, nil, &resp)
}

// FuturesGetOpenPositions gets futures platform's notifications
func (e *Exchange) FuturesGetOpenPositions(ctx context.Context) (FuturesOpenPositions, error) {
	var resp FuturesOpenPositions
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodGet, futuresOpenPositions, nil, &resp)
}

// FuturesNotifications gets futures notifications
func (e *Exchange) FuturesNotifications(ctx context.Context) (FuturesNotificationData, error) {
	var resp FuturesNotificationData
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodGet, futuresNotifications, nil, &resp)
}

// FuturesCancelAllOrders cancels all futures orders for a given symbol or all symbols
func (e *Exchange) FuturesCancelAllOrders(ctx context.Context, symbol currency.Pair) (CancelAllOrdersData, error) {
	var resp CancelAllOrdersData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodPost, futuresCancelAllOrders, params, &resp)
}

// FuturesCancelAllOrdersAfter cancels all futures orders for all symbols after a period of time (timeout measured in seconds)
func (e *Exchange) FuturesCancelAllOrdersAfter(ctx context.Context, timeout int64) (CancelOrdersAfterData, error) {
	var resp CancelOrdersAfterData
	params := url.Values{}
	params.Set("timeout", strconv.FormatInt(timeout, 10))
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodPost, futuresCancelOrdersAfter, params, &resp)
}

// FuturesOpenOrders gets all futures open orders
func (e *Exchange) FuturesOpenOrders(ctx context.Context) (FuturesOpenOrdersData, error) {
	var resp FuturesOpenOrdersData
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodGet, futuresOpenOrders, nil, &resp)
}

// FuturesRecentOrders gets recent futures orders for a symbol or all symbols
func (e *Exchange) FuturesRecentOrders(ctx context.Context, symbol currency.Pair) (FuturesRecentOrdersData, error) {
	var resp FuturesRecentOrdersData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodGet, futuresRecentOrders, nil, &resp)
}

// FuturesWithdrawToSpotWallet withdraws currencies from futures wallet to spot wallet
func (e *Exchange) FuturesWithdrawToSpotWallet(ctx context.Context, ccy string, amount float64) (GenericResponse, error) {
	var resp GenericResponse
	params := url.Values{}
	params.Set("currency", ccy)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodPost, futuresWithdraw, params, &resp)
}

// FuturesGetTransfers withdraws currencies from futures wallet to spot wallet
func (e *Exchange) FuturesGetTransfers(ctx context.Context, lastTransferTime time.Time) (GenericResponse, error) {
	var resp GenericResponse
	params := url.Values{}
	if !lastTransferTime.IsZero() {
		params.Set("lastTransferTime", lastTransferTime.UTC().Format(time.RFC3339))
	}
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodGet, futuresTransfers, params, &resp)
}

// GetFuturesAccountData gets account data for futures
func (e *Exchange) GetFuturesAccountData(ctx context.Context) (FuturesAccountsData, error) {
	var resp FuturesAccountsData
	return resp, e.SendFuturesAuthRequest(ctx, http.MethodGet, futuresAccountData, nil, &resp)
}

func (e *Exchange) signFuturesRequest(secret, endpoint, nonce, data string) (string, error) {
	shasum := sha256.Sum256([]byte(data + nonce + endpoint))
	hmac, err := crypto.GetHMAC(crypto.HashSHA512, shasum[:], []byte(secret))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hmac), nil
}

// SendFuturesAuthRequest will send an auth req
func (e *Exchange) SendFuturesAuthRequest(ctx context.Context, method, path string, data url.Values, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	if data == nil {
		data = url.Values{}
	}

	dataToSign := data.Encode()
	// when json payloads are requested, signing needs to the unendecoded data
	if data.Has("json") {
		dataToSign = "json=" + data.Get("json")
	}

	interim := json.RawMessage{}
	newRequest := func() (*request.Item, error) {
		nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
		var sig string
		sig, err = e.signFuturesRequest(creds.Secret, path, nonce, dataToSign)
		if err != nil {
			return nil, err
		}
		headers := map[string]string{
			"APIKey":  creds.Key,
			"Authent": sig,
			"Nonce":   nonce,
		}

		var futuresURL string
		futuresURL, err = e.API.Endpoints.GetURL(exchange.RestFutures)
		if err != nil {
			return nil, err
		}

		return &request.Item{
			Method:                 method,
			Path:                   futuresURL + common.EncodeURLValues(path, data),
			Headers:                headers,
			Result:                 &interim,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}

	err = e.SendPayload(ctx, request.Unset, newRequest, request.AuthenticatedRequest)

	if err == nil {
		err = getFuturesErr(interim)
	}

	if err == nil {
		err = json.Unmarshal(interim, result)
	}

	if err != nil {
		return fmt.Errorf("%w %w", request.ErrAuthRequestFailed, err)
	}

	return nil
}

func getFuturesErr(msg json.RawMessage) error {
	var resp genericFuturesResponse
	if err := json.Unmarshal(msg, &resp); err != nil {
		return err
	}

	// Result may be omitted entirely, so we don't test for == "success"
	if resp.Result != "error" {
		return nil
	}

	var errs error
	if resp.Error != "" {
		errs = errors.New(resp.Error)
	}

	for _, err := range resp.Errors {
		errs = common.AppendError(errs, errors.New(err))
	}

	if errs == nil {
		return fmt.Errorf("%w from message: %s", common.ErrUnknownError, msg)
	}

	return errs
}
