package bitstamp

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	bitstampAPIURL                = "https://www.bitstamp.net/api"
	tradeBaseURL                  = "https://www.bitstamp.net/trade/"
	bitstampAPIVersion            = "2"
	bitstampAPITicker             = "ticker"
	bitstampAPITickerHourly       = "ticker_hour"
	bitstampAPIOrderbook          = "order_book"
	bitstampAPITransactions       = "transactions"
	bitstampAPIEURUSD             = "eur_usd"
	bitstampAPITradingFees        = "fees/trading"
	bitstampAPIBalance            = "balance"
	bitstampAPIUserTransactions   = "user_transactions"
	bitstampAPIOHLC               = "ohlc"
	bitstampAPIOpenOrders         = "open_orders"
	bitstampAPIOrderStatus        = "order_status"
	bitstampAPICancelOrder        = "cancel_order"
	bitstampAPICancelAllOrders    = "cancel_all_orders"
	bitstampAPIMarket             = "market"
	bitstampAPIWithdrawalRequests = "withdrawal_requests"
	bitstampAPIOpenWithdrawal     = "withdrawal/open"
	bitstampAPIUnconfirmedBitcoin = "unconfirmed_btc"
	bitstampAPITransferToMain     = "transfer-to-main"
	bitstampAPITransferFromMain   = "transfer-from-main"
	bitstampAPIReturnType         = "string"
	bitstampAPITradingPairsInfo   = "trading-pairs-info"
	bitstampAPIWSAuthToken        = "websockets_token"
	bitstampAPIWSTrades           = "live_trades"
	bitstampAPIWSOrders           = "live_orders"
	bitstampAPIWSOrderbook        = "order_book"
	bitstampAPIWSMyOrders         = "my_orders"
	bitstampAPIWSMyTrades         = "my_trades"

	bitstampRateInterval = time.Minute * 10
	bitstampRequestRate  = 8000
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Bitstamp
type Exchange struct {
	exchange.Base
}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		tradingFee, err := e.getTradingFee(ctx, feeBuilder)
		if err != nil {
			return 0, fmt.Errorf("error getting trading fee: %w", err)
		}
		fee = tradingFee
	case exchange.CryptocurrencyDepositFee:
		fee = 0
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.Amount)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.Amount)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// getTradingFee returns a trading fee based on a currency
func (e *Exchange) getTradingFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	tradingFees, err := e.GetAccountTradingFee(ctx, feeBuilder.Pair)
	if err != nil {
		return 0, err
	}
	fees := tradingFees.Fees
	fee := fees.Taker
	if feeBuilder.IsMaker {
		fee = fees.Maker
	}
	return fee / 100 * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
}

// GetAccountTradingFee returns a TradingFee for a pair
func (e *Exchange) GetAccountTradingFee(ctx context.Context, pair currency.Pair) (TradingFees, error) {
	path := bitstampAPITradingFees + "/" + strings.ToLower(pair.String())

	var resp TradingFees
	if pair.IsEmpty() {
		return resp, currency.ErrCurrencyPairEmpty
	}
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, path, true, nil, &resp)

	return resp, err
}

// GetAccountTradingFees returns a slice of TradingFee
func (e *Exchange) GetAccountTradingFees(ctx context.Context) ([]TradingFees, error) {
	path := bitstampAPITradingFees
	var resp []TradingFees
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, path, true, nil, &resp)
	return resp, err
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.0025 * price * amount
}

// getInternationalBankWithdrawalFee returns international withdrawal fee
func getInternationalBankWithdrawalFee(amount float64) float64 {
	fee := amount * 0.0009

	if fee < 15 {
		return 15
	}
	return fee
}

// getInternationalBankDepositFee returns international deposit fee
func getInternationalBankDepositFee(amount float64) float64 {
	fee := amount * 0.0005

	if fee < 7.5 {
		return 7.5
	}
	if fee > 300 {
		return 300
	}
	return fee
}

// GetTicker returns ticker information
func (e *Exchange) GetTicker(ctx context.Context, symbol string, hourly bool) (*Ticker, error) {
	response := Ticker{}
	tickerEndpoint := bitstampAPITicker
	if hourly {
		tickerEndpoint = bitstampAPITickerHourly
	}
	path := "/v" + bitstampAPIVersion + "/" + tickerEndpoint + "/" + strings.ToLower(symbol) + "/"
	return &response, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &response)
}

// GetOrderbook Returns a JSON dictionary with "bids" and "asks". Each is a list
// of open orders and each order is represented as a list holding the price and
// the amount.
func (e *Exchange) GetOrderbook(ctx context.Context, symbol string) (*Orderbook, error) {
	type response struct {
		Timestamp types.Time                       `json:"timestamp"`
		Bids      orderbook.LevelsArrayPriceAmount `json:"bids"`
		Asks      orderbook.LevelsArrayPriceAmount `json:"asks"`
	}

	path := "/v" + bitstampAPIVersion + "/" + bitstampAPIOrderbook + "/" + strings.ToLower(symbol) + "/"
	var resp response
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp); err != nil {
		return nil, err
	}

	return &Orderbook{Timestamp: resp.Timestamp.Time(), Bids: resp.Bids.Levels(), Asks: resp.Asks.Levels()}, nil
}

// GetTradingPairs returns a list of trading pairs which Bitstamp
// currently supports
func (e *Exchange) GetTradingPairs(ctx context.Context) ([]TradingPair, error) {
	var result []TradingPair
	path := "/v" + bitstampAPIVersion + "/" + bitstampAPITradingPairsInfo
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &result)
}

// GetTransactions returns transaction information
// value parameter ["time"] = "minute", "hour", "day" will collate your
// response into time intervals.
func (e *Exchange) GetTransactions(ctx context.Context, currencyPair, timePeriod string) ([]Transactions, error) {
	var transactions []Transactions
	requestURL := "/v" + bitstampAPIVersion + "/" + bitstampAPITransactions + "/" + strings.ToLower(currencyPair) + "/"
	if timePeriod != "" {
		requestURL += "?time=" + url.QueryEscape(timePeriod)
	}
	return transactions, e.SendHTTPRequest(ctx, exchange.RestSpot, requestURL, &transactions)
}

// GetEURUSDConversionRate returns the conversion rate between Euro and USD
func (e *Exchange) GetEURUSDConversionRate(ctx context.Context) (EURUSDConversionRate, error) {
	rate := EURUSDConversionRate{}
	path := "/" + bitstampAPIEURUSD
	return rate, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &rate)
}

// GetBalance returns full balance of currency held on the exchange
func (e *Exchange) GetBalance(ctx context.Context) (Balances, error) {
	var balance map[string]types.Number
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIBalance, true, nil, &balance)
	if err != nil {
		return nil, err
	}
	currs := []string{}
	for k := range balance {
		if strings.HasSuffix(k, "_balance") {
			curr, _, _ := strings.Cut(k, "_")
			currs = append(currs, curr)
		}
	}

	balances := make(map[string]Balance)
	for _, curr := range currs {
		currBalance := Balance{
			Available:     balance[curr+"_available"].Float64(),
			Balance:       balance[curr+"_balance"].Float64(),
			Reserved:      balance[curr+"_reserved"].Float64(),
			WithdrawalFee: balance[curr+"_withdrawal_fee"].Float64(),
		}
		balances[strings.ToUpper(curr)] = currBalance
	}
	return balances, nil
}

// GetUserTransactions returns an array of transactions
func (e *Exchange) GetUserTransactions(ctx context.Context, currencyPair string) ([]UserTransactions, error) {
	var resp []UserTransactions
	var err error
	if currencyPair == "" {
		err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIUserTransactions, true, url.Values{}, &resp)
	} else {
		err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIUserTransactions+"/"+currencyPair, true, url.Values{}, &resp)
	}
	return resp, err
}

// GetOpenOrders returns all open orders on the exchange
func (e *Exchange) GetOpenOrders(ctx context.Context, currencyPair string) ([]Order, error) {
	var resp []Order
	path := bitstampAPIOpenOrders + "/" + strings.ToLower(currencyPair)
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, path, true, nil, &resp)
}

// GetOrderStatus returns an the status of an order by its ID
func (e *Exchange) GetOrderStatus(ctx context.Context, orderID int64) (OrderStatus, error) {
	resp := OrderStatus{}
	req := url.Values{}
	req.Add("id", strconv.FormatInt(orderID, 10))

	return resp,
		e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIOrderStatus, false, req, &resp)
}

// CancelExistingOrder cancels order by ID
func (e *Exchange) CancelExistingOrder(ctx context.Context, orderID int64) (CancelOrder, error) {
	req := url.Values{}
	req.Add("id", strconv.FormatInt(orderID, 10))

	var result CancelOrder
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPICancelOrder, true, req, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

// CancelAllExistingOrders cancels all open orders on the exchange
func (e *Exchange) CancelAllExistingOrders(ctx context.Context) (bool, error) {
	result := false

	return result,
		e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPICancelAllOrders, false, nil, &result)
}

// PlaceOrder places an order on the exchange.
func (e *Exchange) PlaceOrder(ctx context.Context, currencyPair string, price, amount float64, buy, market bool) (Order, error) {
	req := url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("price", strconv.FormatFloat(price, 'f', -1, 64))
	response := Order{}
	orderType := order.Buy.Lower()

	if !buy {
		orderType = order.Sell.Lower()
	}

	var path string
	if market {
		path = orderType + "/" + bitstampAPIMarket + "/" + strings.ToLower(currencyPair)
	} else {
		path = orderType + "/" + strings.ToLower(currencyPair)
	}

	return response,
		e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, path, true, req, &response)
}

// GetWithdrawalRequests returns withdrawal requests for the account
// timedelta - positive integer with max value 50000000 which returns requests
// from number of seconds ago to now.
func (e *Exchange) GetWithdrawalRequests(ctx context.Context, timedelta int64) ([]WithdrawalRequests, error) {
	var resp []WithdrawalRequests
	if timedelta > 50000000 || timedelta < 0 {
		return resp, errors.New("time delta exceeded, max: 50000000 min: 0")
	}

	value := url.Values{}
	value.Set("timedelta", strconv.FormatInt(timedelta, 10))

	if timedelta == 0 {
		value = url.Values{}
	}

	return resp,
		e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIWithdrawalRequests, false, value, &resp)
}

// CryptoWithdrawal withdraws a cryptocurrency into a supplied wallet, returns ID
// amount - The amount you want withdrawn
// address - The wallet address of the cryptocurrency
// symbol - the type of crypto ie "ltc", "btc", "eth"
// destTag - only for XRP  default to ""
func (e *Exchange) CryptoWithdrawal(ctx context.Context, amount float64, address, symbol, destTag string) (*CryptoWithdrawalResponse, error) {
	req := url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	var endpoint string
	switch strings.ToUpper(symbol) {
	case currency.XLM.String():
		if destTag != "" {
			req.Add("memo_id", destTag)
		}
	case currency.XRP.String():
		if destTag != "" {
			req.Add("destination_tag", destTag)
		}
	}

	var resp CryptoWithdrawalResponse
	endpoint = strings.ToLower(symbol) + "_withdrawal"
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, endpoint, true, req, &resp)
}

// OpenBankWithdrawal Opens a bank withdrawal request (SEPA or international)
func (e *Exchange) OpenBankWithdrawal(ctx context.Context, req *OpenBankWithdrawalRequest) (FIATWithdrawalResponse, error) {
	v := url.Values{}
	v.Add("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	v.Add("account_currency", req.Currency.String())
	v.Add("name", req.Name)
	v.Add("iban", req.IBAN)
	v.Add("bic", req.BIC)
	v.Add("address", req.Address)
	v.Add("postal_code", req.PostalCode)
	v.Add("city", req.City)
	v.Add("country", req.Country)
	v.Add("type", req.WithdrawalType)
	v.Add("comment", req.Comment)
	resp := FIATWithdrawalResponse{}
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIOpenWithdrawal, true, v, &resp)
}

// OpenInternationalBankWithdrawal Opens a bank withdrawal request (international)
func (e *Exchange) OpenInternationalBankWithdrawal(ctx context.Context, req *OpenBankWithdrawalRequest) (FIATWithdrawalResponse, error) {
	v := url.Values{}
	v.Add("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	v.Add("account_currency", req.Currency.String())
	v.Add("name", req.Name)
	v.Add("iban", req.IBAN)
	v.Add("bic", req.BIC)
	v.Add("address", req.Address)
	v.Add("postal_code", req.PostalCode)
	v.Add("city", req.City)
	v.Add("country", req.Country)
	v.Add("type", req.WithdrawalType)
	v.Add("comment", req.Comment)
	v.Add("currency", req.InternationalCurrency)
	v.Add("bank_name", req.BankName)
	v.Add("bank_address", req.BankAddress)
	v.Add("bank_postal_code", req.BankPostalCode)
	v.Add("bank_city", req.BankCity)
	v.Add("bank_country", req.BankCountry)
	resp := FIATWithdrawalResponse{}
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIOpenWithdrawal, true, v, &resp)
}

// GetCryptoDepositAddress returns a depositing address by crypto.
// c - example "btc", "ltc", "eth", "xrp" or "bch"
func (e *Exchange) GetCryptoDepositAddress(ctx context.Context, c currency.Code) (*DepositAddress, error) {
	path := c.Lower().String() + "_address"
	var resp DepositAddress
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, path, true, nil, &resp)
}

// GetUnconfirmedBitcoinDeposits returns unconfirmed transactions
func (e *Exchange) GetUnconfirmedBitcoinDeposits(ctx context.Context) ([]UnconfirmedBTCTransactions, error) {
	var response []UnconfirmedBTCTransactions

	return response,
		e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIUnconfirmedBitcoin, false, nil, &response)
}

// OHLC returns OHLCV data for step (interval)
func (e *Exchange) OHLC(ctx context.Context, symbol string, start, end time.Time, step, limit string) (resp OHLCResponse, err error) {
	v := url.Values{}
	v.Add("limit", limit)
	v.Add("step", step)

	if start.After(end) && !end.IsZero() {
		return resp, errors.New("start time cannot be after end time")
	}
	if !start.IsZero() {
		v.Add("start", strconv.FormatInt(start.Unix(), 10))
	}
	if !end.IsZero() {
		v.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v"+bitstampAPIVersion+"/"+bitstampAPIOHLC+"/"+symbol, v), &resp)
}

// TransferAccountBalance transfers funds from either a main or sub account
// amount - to transfers
// currency - which currency to transfer
// subaccount - name of account
// toMain - bool either to or from account
func (e *Exchange) TransferAccountBalance(ctx context.Context, amount float64, ccy, subAccount string, toMain bool) error {
	req := url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("currency", ccy)

	if subAccount == "" {
		return errors.New("missing subAccount parameter")
	}

	req.Add("subAccount", subAccount)

	var path string
	if toMain {
		path = bitstampAPITransferToMain
	} else {
		path = bitstampAPITransferFromMain
	}

	var resp any

	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, path, true, req, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpoint + path,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}
	return e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated request
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, path string, v2 bool, values url.Values, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	if values == nil {
		values = url.Values{}
	}

	interim := json.RawMessage{}
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		n := e.Requester.GetNonce(nonce.UnixNano).String()

		values.Set("key", creds.Key)
		values.Set("nonce", n)

		hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(n+creds.ClientID+creds.Key), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		values.Set("signature", strings.ToUpper(hex.EncodeToString(hmac)))

		var fullPath string
		if v2 {
			fullPath = endpoint + "/v" + bitstampAPIVersion + "/" + path + "/"
		} else {
			fullPath = endpoint + "/" + path + "/"
		}

		headers := make(map[string]string)
		headers["Content-Type"] = "application/x-www-form-urlencoded"

		encodedValues := values.Encode()
		readerValues := bytes.NewBufferString(encodedValues)

		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   fullPath,
			Headers:                headers,
			Body:                   readerValues,
			Result:                 &interim,
			NonceEnabled:           true,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}

	errCap := struct {
		Error  string `json:"error"`  // v1 errors
		Status string `json:"status"` // v2 errors
		Reason any    `json:"reason"` // v2 errors
	}{}
	if err := json.Unmarshal(interim, &errCap); err == nil {
		if errCap.Error != "" || errCap.Status == errStr {
			if errCap.Error != "" { // v1 errors
				return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, errCap.Error)
			}
			switch data := errCap.Reason.(type) { // v2 errors
			case map[string]any:
				var details strings.Builder
				for k, v := range data {
					details.WriteString(fmt.Sprintf("%s: %v", k, v))
				}
				return errors.New(details.String())
			case string:
				return errors.New(data)
			default:
				return errors.New(errCap.Status)
			}
		}
	}
	return json.Unmarshal(interim, result)
}

func filterOrderbookZeroBidPrice(ob *orderbook.Book) {
	if len(ob.Bids) == 0 || ob.Bids[len(ob.Bids)-1].Price != 0 {
		return
	}

	ob.Bids = ob.Bids[0 : len(ob.Bids)-1]
}
