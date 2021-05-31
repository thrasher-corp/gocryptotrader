package kraken

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetFuturesOrderbook gets orderbook data for futures
func (k *Kraken) GetFuturesOrderbook(symbol currency.Pair) (FuturesOrderbookData, error) {
	var resp FuturesOrderbookData
	params := url.Values{}
	symbolValue, err := k.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	return resp, k.SendHTTPRequest(exchange.RestFutures, futuresOrderbook+"?"+params.Encode(), &resp)
}

// GetFuturesMarkets gets a list of futures markets and their data
func (k *Kraken) GetFuturesMarkets() (FuturesInstrumentData, error) {
	var resp FuturesInstrumentData
	return resp, k.SendHTTPRequest(exchange.RestFutures, futuresInstruments, &resp)
}

// GetFuturesTickers gets a list of futures tickers and their data
func (k *Kraken) GetFuturesTickers() (FuturesTickerData, error) {
	var resp FuturesTickerData
	return resp, k.SendHTTPRequest(exchange.RestFutures, futuresTickers, &resp)
}

// GetFuturesTradeHistory gets public trade history data for futures
func (k *Kraken) GetFuturesTradeHistory(symbol currency.Pair, lastTime time.Time) (FuturesTradeHistoryData, error) {
	var resp FuturesTradeHistoryData
	params := url.Values{}
	symbolValue, err := k.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !lastTime.IsZero() {
		params.Set("lastTime", lastTime.Format("2006-01-02T15:04:05.070Z"))
	}
	return resp, k.SendHTTPRequest(exchange.RestFutures, futuresTradeHistory+"?"+params.Encode(), &resp)
}

// FuturesBatchOrder places a batch order for futures
func (k *Kraken) FuturesBatchOrder(data []PlaceBatchOrderData) (FuturesAccountsData, error) {
	var resp FuturesAccountsData
	for x := range data {
		unformattedPair, err := currency.NewPairFromString(data[x].Symbol)
		if err != nil {
			return resp, err
		}
		formattedPair, err := k.FormatExchangeCurrency(unformattedPair, asset.Futures)
		if err != nil {
			return resp, err
		}
		data[x].Symbol = formattedPair.String()
	}
	req := make(map[string]interface{})
	req["batchOrder"] = data
	return resp, k.SendFuturesAuthRequest(http.MethodPost, futuresBatchOrder, nil, req, &resp)
}

// FuturesEditOrder edits a futures order
func (k *Kraken) FuturesEditOrder(orderID, clientOrderID string, size, limitPrice, stopPrice float64) (FuturesAccountsData, error) {
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
	return resp, k.SendFuturesAuthRequest(http.MethodPost, futuresEditOrder, params, nil, &resp)
}

// FuturesSendOrder sends a futures order
func (k *Kraken) FuturesSendOrder(orderType order.Type, symbol currency.Pair, side, triggerSignal, clientOrderID, reduceOnly string,
	size, limitPrice, stopPrice float64) (FuturesSendOrderData, error) {
	var resp FuturesSendOrderData
	oType, ok := validOrderTypes[orderType]
	if !ok {
		return resp, errors.New("invalid orderType")
	}
	params := url.Values{}
	params.Set("orderType", oType)
	symbolValue, err := k.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(validSide, side) {
		return resp, errors.New("invalid side")
	}
	params.Set("side", side)
	if triggerSignal != "" {
		if !common.StringDataCompare(validTriggerSignal, triggerSignal) {
			return resp, errors.New("invalid triggerSignal")
		}
		params.Set("triggerSignal", triggerSignal)
	}
	if clientOrderID != "" {
		params.Set("cliOrdId", clientOrderID)
	}
	if reduceOnly != "" {
		if !common.StringDataCompare(validReduceOnly, reduceOnly) {
			return resp, errors.New("invalid reduceOnly")
		}
		params.Set("reduceOnly", reduceOnly)
	}
	params.Set("size", strconv.FormatFloat(size, 'f', -1, 64))
	params.Set("limitPrice", strconv.FormatFloat(limitPrice, 'f', -1, 64))
	if stopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(stopPrice, 'f', -1, 64))
	}
	return resp, k.SendFuturesAuthRequest(http.MethodPost, futuresSendOrder, params, nil, &resp)
}

// FuturesCancelOrder cancels an order
func (k *Kraken) FuturesCancelOrder(orderID, clientOrderID string) (FuturesCancelOrderData, error) {
	var resp FuturesCancelOrderData
	params := url.Values{}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if clientOrderID != "" {
		params.Set("cliOrdId", clientOrderID)
	}
	return resp, k.SendFuturesAuthRequest(http.MethodPost, futuresCancelOrder, params, nil, &resp)
}

// FuturesGetFills gets order fills for futures
func (k *Kraken) FuturesGetFills(lastFillTime time.Time) (FuturesFillsData, error) {
	var resp FuturesFillsData
	params := url.Values{}
	if !lastFillTime.IsZero() {
		params.Set("lastFillTime", lastFillTime.UTC().Format("2006-01-02T15:04:05.999Z"))
	}
	return resp, k.SendFuturesAuthRequest(http.MethodGet, futuresOrderFills, params, nil, &resp)
}

// FuturesTransfer transfers funds between accounts
func (k *Kraken) FuturesTransfer(fromAccount, toAccount, unit string, amount float64) (FuturesTransferData, error) {
	var resp FuturesTransferData
	req := make(map[string]interface{})
	req["fromAccount"] = fromAccount
	req["toAccount"] = toAccount
	req["unit"] = unit
	req["amount"] = amount
	return resp, k.SendFuturesAuthRequest(http.MethodPost, futuresTransfer, nil, nil, &resp)
}

// FuturesGetOpenPositions gets futures platform's notifications
func (k *Kraken) FuturesGetOpenPositions() (FuturesOpenPositions, error) {
	var resp FuturesOpenPositions
	return resp, k.SendFuturesAuthRequest(http.MethodGet, futuresOpenPositions, nil, nil, &resp)
}

// FuturesNotifications gets futures notifications
func (k *Kraken) FuturesNotifications() (FuturesNotificationData, error) {
	var resp FuturesNotificationData
	return resp, k.SendFuturesAuthRequest(http.MethodGet, futuresNotifications, nil, nil, &resp)
}

// FuturesCancelAllOrders cancels all futures orders for a given symbol or all symbols
func (k *Kraken) FuturesCancelAllOrders(symbol currency.Pair) (CancelAllOrdersData, error) {
	var resp CancelAllOrdersData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := k.FormatSymbol(symbol, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	return resp, k.SendFuturesAuthRequest(http.MethodPost, futuresCancelAllOrders, params, nil, &resp)
}

// FuturesCancelAllOrdersAfter cancels all futures orders for all symbols after a period of time (timeout measured in seconds)
func (k *Kraken) FuturesCancelAllOrdersAfter(timeout int64) (CancelOrdersAfterData, error) {
	var resp CancelOrdersAfterData
	params := url.Values{}
	params.Set("timeout", strconv.FormatInt(timeout, 10))
	return resp, k.SendFuturesAuthRequest(http.MethodPost, futuresCancelOrdersAfter, params, nil, &resp)
}

// FuturesOpenOrders gets all futures open orders
func (k *Kraken) FuturesOpenOrders() (FuturesOpenOrdersData, error) {
	var resp FuturesOpenOrdersData
	return resp, k.SendFuturesAuthRequest(http.MethodGet, futuresOpenOrders, nil, nil, &resp)
}

// FuturesRecentOrders gets recent futures orders for a symbol or all symbols
func (k *Kraken) FuturesRecentOrders(symbol currency.Pair) (FuturesRecentOrdersData, error) {
	var resp FuturesRecentOrdersData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := k.FormatSymbol(symbol, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	return resp, k.SendFuturesAuthRequest(http.MethodGet, futuresRecentOrders, nil, nil, &resp)
}

// FuturesWithdrawToSpotWallet withdraws currencies from futures wallet to spot wallet
func (k *Kraken) FuturesWithdrawToSpotWallet(currency string, amount float64) (GenericResponse, error) {
	var resp GenericResponse
	params := url.Values{}
	params.Set("currency", currency)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return resp, k.SendFuturesAuthRequest(http.MethodPost, futuresWithdraw, params, nil, &resp)
}

// FuturesGetTransfers withdraws currencies from futures wallet to spot wallet
func (k *Kraken) FuturesGetTransfers(lastTransferTime time.Time) (GenericResponse, error) {
	var resp GenericResponse
	params := url.Values{}
	if !lastTransferTime.IsZero() {
		params.Set("lastTransferTime", lastTransferTime.UTC().Format(time.RFC3339))
	}
	return resp, k.SendFuturesAuthRequest(http.MethodGet, futuresTransfers, params, nil, &resp)
}

// GetFuturesAccountData gets account data for futures
func (k *Kraken) GetFuturesAccountData() (FuturesAccountsData, error) {
	var resp FuturesAccountsData
	return resp, k.SendFuturesAuthRequest(http.MethodGet, futuresAccountData, nil, nil, &resp)
}

func (k *Kraken) signFuturesRequest(endpoint, nonce, data string) string {
	message := data + nonce + endpoint
	hash := crypto.GetSHA256([]byte(message))
	hc := crypto.GetHMAC(crypto.HashSHA512, hash, []byte(k.API.Credentials.Secret))
	return base64.StdEncoding.EncodeToString(hc)
}

// SendFuturesAuthRequest will send an auth req
func (k *Kraken) SendFuturesAuthRequest(method, path string, postData url.Values, data map[string]interface{}, result interface{}) error {
	if !k.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", k.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	if postData == nil {
		postData = url.Values{}
	}
	nonce := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	reqData := ""
	if len(data) > 0 {
		temp, err := json.Marshal(data)
		if err != nil {
			return err
		}
		postData.Add("json", string(temp))
		reqData = "json=" + string(temp)
	}
	sig := k.signFuturesRequest(path, nonce, reqData)
	headers := map[string]string{
		"APIKey":  k.API.Credentials.Key,
		"Authent": sig,
		"Nonce":   nonce,
	}
	interim := json.RawMessage{}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
	defer cancel()
	err := k.SendPayload(ctx, &request.Item{
		Method:        method,
		Path:          futuresURL + common.EncodeURLValues(path, postData),
		Headers:       headers,
		Result:        &interim,
		AuthRequest:   true,
		Verbose:       k.Verbose,
		HTTPDebugging: k.HTTPDebugging,
		HTTPRecording: k.HTTPRecording,
	})
	if err != nil {
		return err
	}
	var errCap AuthErrorData
	if err := json.Unmarshal(interim, &errCap); err == nil {
		if errCap.Result != "success" && errCap.Error != "" {
			return errors.New(errCap.Error)
		}
	}
	return json.Unmarshal(interim, result)
}
