package yobit

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	apiPublicURL                  = "https://yobit.net/api"
	apiPrivateURL                 = "https://yobit.net/tapi"
	tradeBaseURL                  = "https://yobit.net/en/trade/"
	apiPublicVersion              = "3"
	publicInfo                    = "info"
	publicTicker                  = "ticker"
	publicDepth                   = "depth"
	publicTrades                  = "trades"
	privateAccountInfo            = "getInfo"
	privateTrade                  = "Trade"
	privateActiveOrders           = "ActiveOrders"
	privateOrderInfo              = "OrderInfo"
	privateCancelOrder            = "CancelOrder"
	privateTradeHistory           = "TradeHistory"
	privateGetDepositAddress      = "GetDepositAddress"
	privateWithdrawCoinsToAddress = "WithdrawCoinsToAddress"
	privateCreateCoupon           = "CreateYobicode"
	privateRedeemCoupon           = "RedeemYobicode"
)

// Yobit is the overarching type across the Yobit package
type Yobit struct {
	exchange.Base
}

// GetInfo returns the Yobit info
func (y *Yobit) GetInfo(ctx context.Context) (Info, error) {
	resp := Info{}
	path := fmt.Sprintf("/%s/%s/", apiPublicVersion, publicInfo)

	return resp, y.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetTicker returns a ticker for a specific currency
func (y *Yobit) GetTicker(ctx context.Context, symbol string) (map[string]Ticker, error) {
	type Response struct {
		Data map[string]Ticker
	}

	response := Response{}
	path := fmt.Sprintf("/%s/%s/%s", apiPublicVersion, publicTicker, symbol)

	return response.Data, y.SendHTTPRequest(ctx, exchange.RestSpot, path, &response.Data)
}

// GetDepth returns the depth for a specific currency
func (y *Yobit) GetDepth(ctx context.Context, symbol string) (Orderbook, error) {
	type Response struct {
		Data map[string]Orderbook
	}

	response := Response{}
	path := fmt.Sprintf("/%s/%s/%s", apiPublicVersion, publicDepth, symbol)

	return response.Data[symbol],
		y.SendHTTPRequest(ctx, exchange.RestSpot, path, &response.Data)
}

// GetTrades returns the trades for a specific currency
func (y *Yobit) GetTrades(ctx context.Context, symbol string) ([]Trade, error) {
	type respDataHolder struct {
		Data map[string][]Trade
	}

	var dataHolder respDataHolder
	path := "/" + apiPublicVersion + "/" + publicTrades + "/" + symbol
	err := y.SendHTTPRequest(ctx, exchange.RestSpot, path, &dataHolder.Data)
	if err != nil {
		return nil, err
	}

	if tr, ok := dataHolder.Data[symbol]; ok {
		return tr, nil
	}
	return nil, nil
}

// GetAccountInformation returns a users account info
func (y *Yobit) GetAccountInformation(ctx context.Context) (AccountInfo, error) {
	result := AccountInfo{}

	err := y.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, privateAccountInfo, url.Values{}, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, fmt.Errorf("%w %v", request.ErrAuthRequestFailed, result.Error)
	}
	return result, nil
}

// Trade places an order and returns the order ID if successful or an error
func (y *Yobit) Trade(ctx context.Context, pair, orderType string, amount, price float64) (int64, error) {
	req := url.Values{}
	req.Add("pair", pair)
	req.Add("type", strings.ToLower(orderType))
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("rate", strconv.FormatFloat(price, 'f', -1, 64))

	result := TradeOrderResponse{}

	err := y.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, privateTrade, req, &result)
	if err != nil {
		return int64(result.OrderID), err
	}
	if result.Error != "" {
		return -1, fmt.Errorf("%w %v", request.ErrAuthRequestFailed, result.Error)
	}
	return int64(result.OrderID), nil
}

// GetOpenOrders returns the active orders for a specific currency
func (y *Yobit) GetOpenOrders(ctx context.Context, pair string) (map[string]ActiveOrders, error) {
	req := url.Values{}
	req.Add("pair", pair)

	result := map[string]ActiveOrders{}

	return result, y.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, privateActiveOrders, req, &result)
}

// GetOrderInformation returns the order info for a specific order ID
func (y *Yobit) GetOrderInformation(ctx context.Context, orderID int64) (map[string]OrderInfo, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(orderID, 10))

	result := map[string]OrderInfo{}

	return result, y.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, privateOrderInfo, req, &result)
}

// CancelExistingOrder cancels an order for a specific order ID
func (y *Yobit) CancelExistingOrder(ctx context.Context, orderID int64) error {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(orderID, 10))

	result := CancelOrder{}

	err := y.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, privateCancelOrder, req, &result)
	if err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, result.Error)
	}
	return nil
}

// GetTradeHistory returns the trade history
func (y *Yobit) GetTradeHistory(ctx context.Context, tidFrom, count, tidEnd, since, end int64, order, pair string) (map[string]TradeHistory, error) {
	req := url.Values{}
	req.Add("from", strconv.FormatInt(tidFrom, 10))
	req.Add("count", strconv.FormatInt(count, 10))
	req.Add("from_id", strconv.FormatInt(tidFrom, 10))
	req.Add("end_id", strconv.FormatInt(tidEnd, 10))
	req.Add("order", order)
	req.Add("since", strconv.FormatInt(since, 10))
	req.Add("end", strconv.FormatInt(end, 10))
	req.Add("pair", pair)

	result := TradeHistoryResponse{}

	err := y.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, privateTradeHistory, req, &result)
	if err != nil {
		return nil, err
	}
	if result.Success == 0 {
		return nil, fmt.Errorf("%w %v", request.ErrAuthRequestFailed, result.Error)
	}

	return result.Data, nil
}

// GetCryptoDepositAddress returns the deposit address for a specific currency
func (y *Yobit) GetCryptoDepositAddress(ctx context.Context, coin string, createNew bool) (*DepositAddress, error) {
	req := url.Values{}
	req.Add("coinName", coin)
	if createNew {
		req.Set("need_new", "1")
	}

	var result DepositAddress
	err := y.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		privateGetDepositAddress,
		req,
		&result)
	if err != nil {
		return nil, err
	}
	if result.Success != 1 {
		return nil, fmt.Errorf("%w %v", request.ErrAuthRequestFailed, result.Error)
	}

	return &result, nil
}

// WithdrawCoinsToAddress initiates a withdrawal to a specified address
func (y *Yobit) WithdrawCoinsToAddress(ctx context.Context, coin string, amount float64, address string) (WithdrawCoinsToAddress, error) {
	req := url.Values{}
	req.Add("coinName", coin)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	result := WithdrawCoinsToAddress{}

	err := y.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, privateWithdrawCoinsToAddress, req, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return WithdrawCoinsToAddress{}, fmt.Errorf("%w %v", request.ErrAuthRequestFailed, result.Error)
	}
	return result, nil
}

// CreateCoupon creates an exchange coupon for a specific currency
func (y *Yobit) CreateCoupon(ctx context.Context, currency string, amount float64) (CreateCoupon, error) {
	req := url.Values{}
	req.Add("currency", currency)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	var result CreateCoupon

	err := y.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, privateCreateCoupon, req, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// RedeemCoupon redeems an exchange coupon
func (y *Yobit) RedeemCoupon(ctx context.Context, coupon string) (RedeemCoupon, error) {
	req := url.Values{}
	req.Add("coupon", coupon)

	result := RedeemCoupon{}

	err := y.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, privateRedeemCoupon, req, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (y *Yobit) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := y.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       y.Verbose,
		HTTPDebugging: y.HTTPDebugging,
		HTTPRecording: y.HTTPRecording,
	}

	return y.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to Yobit
func (y *Yobit) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, path string, params url.Values, result any) (err error) {
	creds, err := y.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := y.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	if params == nil {
		params = url.Values{}
	}

	return y.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		n := y.Requester.GetNonce(nonce.Unix).String()

		params.Set("nonce", n)
		params.Set("method", path)

		encoded := params.Encode()
		hmac, err := crypto.GetHMAC(crypto.HashSHA512, []byte(encoded), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		headers["Key"] = creds.Key
		headers["Sign"] = hex.EncodeToString(hmac)
		headers["Content-Type"] = "application/x-www-form-urlencoded"

		return &request.Item{
			Method:        http.MethodPost,
			Path:          endpoint,
			Headers:       headers,
			Body:          strings.NewReader(encoded),
			Result:        result,
			NonceEnabled:  true,
			Verbose:       y.Verbose,
			HTTPDebugging: y.HTTPDebugging,
			HTTPRecording: y.HTTPRecording,
		}, nil
	}, request.AuthenticatedRequest)
}

// GetFee returns an estimate of fee based on type of transaction
func (y *Yobit) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.FiatCurrency,
			feeBuilder.BankTransactionType)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency,
			feeBuilder.Amount,
			feeBuilder.BankTransactionType)
	case exchange.OfflineTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(price, amount float64) (fee float64) {
	return 0.002 * price * amount
}

func getWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

func getInternationalBankWithdrawalFee(c currency.Code, amount float64, bankTransactionType exchange.InternationalBankTransactionType) float64 {
	var fee float64

	switch bankTransactionType {
	case exchange.PerfectMoney:
		if c.Equal(currency.USD) {
			fee = 0.02 * amount
		}
	case exchange.Payeer:
		switch c {
		case currency.USD:
			fee = 0.03 * amount
		case currency.RUR:
			fee = 0.006 * amount
		}
	case exchange.AdvCash:
		switch c {
		case currency.USD:
			fee = 0.04 * amount
		case currency.RUR:
			fee = 0.03 * amount
		}
	case exchange.Qiwi:
		if c.Equal(currency.RUR) {
			fee = 0.04 * amount
		}
	case exchange.Capitalist:
		if c.Equal(currency.USD) {
			fee = 0.06 * amount
		}
	}

	return fee
}

// getInternationalBankDepositFee; No real fees for yobit deposits, but want to be explicit on what each payment type supports
func getInternationalBankDepositFee(c currency.Code, bankTransactionType exchange.InternationalBankTransactionType) float64 {
	var fee float64
	switch bankTransactionType {
	case exchange.PerfectMoney:
		if c.Equal(currency.USD) {
			fee = 0
		}
	case exchange.Payeer:
		switch c {
		case currency.USD:
			fee = 0
		case currency.RUR:
			fee = 0
		}
	case exchange.AdvCash:
		switch c {
		case currency.USD:
			fee = 0
		case currency.RUR:
			fee = 0
		}
	case exchange.Qiwi:
		if c.Equal(currency.RUR) {
			fee = 0
		}
	case exchange.Capitalist:
		switch c {
		case currency.USD:
			fee = 0
		case currency.RUR:
			fee = 0
		}
	}

	return fee
}
