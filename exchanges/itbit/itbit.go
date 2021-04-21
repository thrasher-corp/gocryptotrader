package itbit

import (
	"bytes"
	"context"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	itbitAPIURL         = "https://api.itbit.com/v1"
	itbitAPIVersion     = "1"
	itbitMarkets        = "markets"
	itbitOrderbook      = "order_book"
	itbitTicker         = "ticker"
	itbitWallets        = "wallets"
	itbitBalances       = "balances"
	itbitTrades         = "trades"
	itbitFundingHistory = "funding_history"
	itbitOrders         = "orders"
	itbitCryptoDeposits = "cryptocurrency_deposits"
	itbitWalletTransfer = "wallet_transfers"
)

// ItBit is the overarching type across the ItBit package
type ItBit struct {
	exchange.Base
}

// GetTicker returns ticker info for a specified market.
// currencyPair - example "XBTUSD" "XBTSGD" "XBTEUR"
func (i *ItBit) GetTicker(currencyPair string) (Ticker, error) {
	var response Ticker
	path := fmt.Sprintf("/%s/%s/%s", itbitMarkets, currencyPair, itbitTicker)

	return response, i.SendHTTPRequest(exchange.RestSpot, path, &response)
}

// GetOrderbook returns full order book for the specified market.
// currencyPair - example "XBTUSD" "XBTSGD" "XBTEUR"
func (i *ItBit) GetOrderbook(currencyPair string) (OrderbookResponse, error) {
	response := OrderbookResponse{}
	path := fmt.Sprintf("/%s/%s/%s", itbitMarkets, currencyPair, itbitOrderbook)

	return response, i.SendHTTPRequest(exchange.RestSpot, path, &response)
}

// GetTradeHistory returns recent trades for a specified market.
//
// currencyPair - example "XBTUSD" "XBTSGD" "XBTEUR"
// timestamp - matchNumber, only executions after this will be returned
func (i *ItBit) GetTradeHistory(currencyPair, tradeID string) (Trades, error) {
	response := Trades{}

	var req = itbitTrades
	if tradeID != "" {
		req += "?since=" + tradeID
	}
	path := fmt.Sprintf("/%s/%s/%s", itbitMarkets, currencyPair, req)

	return response, i.SendHTTPRequest(exchange.RestSpot, path, &response)
}

// GetWallets returns information about all wallets associated with the account.
//
// params --
// 					page - [optional] page to return example 1. default 1
//					perPage - [optional] items per page example 50, default 50 max 50
func (i *ItBit) GetWallets(params url.Values) ([]Wallet, error) {
	var resp []Wallet
	params.Set("userId", i.API.Credentials.ClientID)
	path := fmt.Sprintf("/%s?%s", itbitWallets, params.Encode())
	return resp, i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// CreateWallet creates a new wallet with a specified name.
func (i *ItBit) CreateWallet(walletName string) (Wallet, error) {
	resp := Wallet{}
	params := make(map[string]interface{})
	params["userId"] = i.API.Credentials.ClientID
	params["name"] = walletName

	err := i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, "/"+itbitWallets, params, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Description != "" {
		return resp, errors.New(resp.Description)
	}
	return resp, nil
}

// GetWallet returns wallet information by walletID
func (i *ItBit) GetWallet(walletID string) (Wallet, error) {
	resp := Wallet{}
	path := fmt.Sprintf("/%s/%s", itbitWallets, walletID)

	err := i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet, path, nil, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Description != "" {
		return resp, errors.New(resp.Description)
	}
	return resp, nil
}

// GetWalletBalance returns balance information for a specific currency in a
// wallet.
func (i *ItBit) GetWalletBalance(walletID, currency string) (Balance, error) {
	resp := Balance{}
	path := fmt.Sprintf("/%s/%s/%s/%s", itbitWallets, walletID, itbitBalances, currency)

	err := i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet, path, nil, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Description != "" {
		return resp, errors.New(resp.Description)
	}
	return resp, nil
}

// GetOrders returns active orders for itBit
// perPage defaults to & has a limit of 50
func (i *ItBit) GetOrders(walletID, symbol, status string, page, perPage int64) ([]Order, error) {
	var resp []Order
	params := make(map[string]interface{})
	params["walletID"] = walletID

	if symbol != "" {
		params["instrument"] = symbol
	}
	if status != "" {
		params["status"] = status
	}
	if page > 0 {
		params["page"] = strconv.FormatInt(page, 10)
	}
	if perPage > 0 {
		params["perPage"] = strconv.FormatInt(perPage, 10)
	}

	return resp, i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet, itbitOrders, params, &resp)
}

// GetWalletTrades returns all trades for a specified wallet.
func (i *ItBit) GetWalletTrades(walletID string, params url.Values) (Records, error) {
	resp := Records{}
	urlPath := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitTrades)
	path := common.EncodeURLValues(urlPath, params)

	err := i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet, path, nil, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Description != "" {
		return resp, errors.New(resp.Description)
	}
	return resp, nil
}

// GetFundingHistoryForWallet returns all funding history for a specified wallet.
func (i *ItBit) GetFundingHistoryForWallet(walletID string, params url.Values) (FundingRecords, error) {
	resp := FundingRecords{}
	urlPath := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitFundingHistory)
	path := common.EncodeURLValues(urlPath, params)

	err := i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet, path, nil, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Description != "" {
		return resp, errors.New(resp.Description)
	}
	return resp, nil
}

// PlaceOrder places a new order
func (i *ItBit) PlaceOrder(walletID, side, orderType, currency string, amount, price float64, instrument, clientRef string) (Order, error) {
	resp := Order{}
	path := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitOrders)

	params := make(map[string]interface{})
	params["side"] = side
	params["type"] = orderType
	params["currency"] = currency
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	params["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	params["instrument"] = instrument

	if clientRef != "" {
		params["clientOrderIdentifier"] = clientRef
	}

	err := i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Description != "" {
		return resp, errors.New(resp.Description)
	}
	return resp, nil
}

// GetOrder returns an order by id.
func (i *ItBit) GetOrder(walletID string, params url.Values) (Order, error) {
	resp := Order{}
	urlPath := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitOrders)
	path := common.EncodeURLValues(urlPath, params)

	err := i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet, path, nil, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Description != "" {
		return resp, errors.New(resp.Description)
	}
	return resp, nil
}

// CancelExistingOrder cancels and open order. *This is not a guarantee that the
// order has been cancelled!*
func (i *ItBit) CancelExistingOrder(walletID, orderID string) error {
	path := fmt.Sprintf("/%s/%s/%s/%s", itbitWallets, walletID, itbitOrders, orderID)

	return i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodDelete, path, nil, nil)
}

// GetCryptoDepositAddress returns a deposit address to send cryptocurrency to.
func (i *ItBit) GetCryptoDepositAddress(walletID, currency string) (CryptoCurrencyDeposit, error) {
	resp := CryptoCurrencyDeposit{}
	path := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitCryptoDeposits)
	params := make(map[string]interface{})
	params["currency"] = currency

	err := i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Description != "" {
		return resp, errors.New(resp.Description)
	}
	return resp, nil
}

// WalletTransfer transfers funds between wallets.
func (i *ItBit) WalletTransfer(walletID, sourceWallet, destWallet string, amount float64, currency string) (WalletTransfer, error) {
	resp := WalletTransfer{}
	path := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitWalletTransfer)

	params := make(map[string]interface{})
	params["sourceWalletId"] = sourceWallet
	params["destinationWalletId"] = destWallet
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	params["currencyCode"] = currency

	err := i.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Description != "" {
		return resp, errors.New(resp.Description)
	}
	return resp, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (i *ItBit) SendHTTPRequest(ep exchange.URL, path string, result interface{}) error {
	endpoint, err := i.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	return i.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       i.Verbose,
		HTTPDebugging: i.HTTPDebugging,
		HTTPRecording: i.HTTPRecording,
	})
}

// SendAuthenticatedHTTPRequest sends an authenticated request to itBit
func (i *ItBit) SendAuthenticatedHTTPRequest(ep exchange.URL, method, path string, params map[string]interface{}, result interface{}) error {
	if !i.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", i.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	endpoint, err := i.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	req := make(map[string]interface{})
	urlPath := endpoint + path

	for key, value := range params {
		req[key] = value
	}

	PayloadJSON := []byte("")

	if params != nil {
		PayloadJSON, err = json.Marshal(req)
		if err != nil {
			return err
		}

		if i.Verbose {
			log.Debugf(log.ExchangeSys, "Request JSON: %s\n", PayloadJSON)
		}
	}

	n := i.Requester.GetNonce(true).String()
	timestamp := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	message, err := json.Marshal([]string{method, urlPath, string(PayloadJSON), n, timestamp})
	if err != nil {
		return err
	}

	hash := crypto.GetSHA256([]byte(n + string(message)))
	hmac := crypto.GetHMAC(crypto.HashSHA512, []byte(urlPath+string(hash)), []byte(i.API.Credentials.Secret))
	signature := crypto.Base64Encode(hmac)

	headers := make(map[string]string)
	headers["Authorization"] = i.API.Credentials.ClientID + ":" + signature
	headers["X-Auth-Timestamp"] = timestamp
	headers["X-Auth-Nonce"] = n
	headers["Content-Type"] = "application/json"

	var intermediary json.RawMessage

	errCheck := struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
		RequestID   string `json:"requestId"`
	}{}

	err = i.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          urlPath,
		Headers:       headers,
		Body:          bytes.NewBuffer(PayloadJSON),
		Result:        &intermediary,
		AuthRequest:   true,
		NonceEnabled:  true,
		Verbose:       i.Verbose,
		HTTPDebugging: i.HTTPDebugging,
		HTTPRecording: i.HTTPRecording,
	})
	if err != nil {
		return err
	}

	err = json.Unmarshal(intermediary, &errCheck)
	if err == nil {
		if errCheck.Code != 0 || errCheck.Description != "" {
			return fmt.Errorf("sendAuthRequest error code: %d description: %s",
				errCheck.Code,
				errCheck.Description)
		}
	}

	return json.Unmarshal(intermediary, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (i *ItBit) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency, feeBuilder.BankTransactionType)
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
	return 0.0035 * price * amount
}

func calculateTradingFee(purchasePrice, amount float64, isMaker bool) float64 {
	// TODO: Itbit has volume discounts, but not API endpoint to get the exact volume numbers
	// When support is added, this needs to be updated to calculate the accurate volume fee
	feePercent := 0.0035
	if isMaker {
		feePercent = -0.0003
	}
	return feePercent * purchasePrice * amount
}

func getInternationalBankWithdrawalFee(c currency.Code, bankTransactionType exchange.InternationalBankTransactionType) float64 {
	var fee float64
	if (bankTransactionType == exchange.Swift ||
		bankTransactionType == exchange.WireTransfer) &&
		c == currency.USD {
		fee = 40
	} else if (bankTransactionType == exchange.SEPA ||
		bankTransactionType == exchange.WireTransfer) &&
		c == currency.EUR {
		fee = 1
	}
	return fee
}
