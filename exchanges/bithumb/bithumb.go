package bithumb

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	apiURL       = "https://api.bithumb.com"
	tradeBaseURL = "https://www.bithumb.com/react/trade/order/"

	noError = "0000"

	publicTicker             = "/public/ticker/"
	publicOrderBook          = "/public/orderbook/"
	publicTransactionHistory = "/public/transaction_history/"
	publicCandleStick        = "/public/candlestick/"
	publicAssetStatus        = "/public/assetsstatus/"

	privateAccInfo     = "/info/account"
	privateAccBalance  = "/info/balance"
	privateWalletAdd   = "/info/wallet_address"
	privateTicker      = "/info/ticker"
	privateOrders      = "/info/orders"
	privateUserTrans   = "/info/user_transactions"
	privatePlaceTrade  = "/trade/place"
	privateOrderDetail = "/info/order_detail"
	privateCancelTrade = "/trade/cancel"
	privateBTCWithdraw = "/trade/btc_withdrawal"
	privateKRWDeposit  = "/trade/krw_deposit"
	privateKRWWithdraw = "/trade/krw_withdrawal"
	privateMarketBuy   = "/trade/market_buy"
	privateMarketSell  = "/trade/market_sell"
)

var errSymbolIsEmpty = errors.New("symbol cannot be empty")

// Bithumb is the overarching type across the Bithumb package
type Bithumb struct {
	exchange.Base
	location *time.Location
	obm      orderbookManager
}

// GetTradablePairs returns a list of tradable currencies
func (b *Bithumb) GetTradablePairs(ctx context.Context) ([]string, error) {
	result, err := b.GetAllTickers(ctx)
	if err != nil {
		return nil, err
	}

	currencies := make([]string, 0, len(result))
	for x := range result {
		currencies = append(currencies, x)
	}
	return currencies, nil
}

// GetTicker returns ticker information
//
// symbol e.g. "btc"
func (b *Bithumb) GetTicker(ctx context.Context, symbol string) (Ticker, error) {
	var response TickerResponse
	err := b.SendHTTPRequest(ctx, exchange.RestSpot, publicTicker+strings.ToUpper(symbol), &response)
	if err != nil {
		return response.Data, err
	}

	if response.Status != noError {
		return response.Data, errors.New(response.Message)
	}

	return response.Data, nil
}

// GetAllTickers returns all ticker information
func (b *Bithumb) GetAllTickers(ctx context.Context) (map[string]Ticker, error) {
	var response TickersResponse
	err := b.SendHTTPRequest(ctx, exchange.RestSpot, publicTicker+"all", &response)
	if err != nil {
		return nil, err
	}

	if response.Status != noError {
		return nil, errors.New(response.Message)
	}

	result := make(map[string]Ticker)
	for k, v := range response.Data {
		if k == "date" {
			continue
		}
		var newTicker Ticker
		err := json.Unmarshal(v, &newTicker)
		if err != nil {
			return nil, err
		}
		result[k] = newTicker
	}
	return result, nil
}

// GetOrderBook returns current orderbook
//
// symbol e.g. "btc"
func (b *Bithumb) GetOrderBook(ctx context.Context, symbol string) (*Orderbook, error) {
	response := Orderbook{}
	err := b.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderBook+strings.ToUpper(symbol), &response)
	if err != nil {
		return nil, err
	}

	if response.Status != noError {
		return nil, errors.New(response.Message)
	}

	return &response, nil
}

// GetAssetStatus returns the withdrawal and deposit status for the symbol
func (b *Bithumb) GetAssetStatus(ctx context.Context, symbol string) (*Status, error) {
	if symbol == "" {
		return nil, errSymbolIsEmpty
	}
	var response Status
	err := b.SendHTTPRequest(ctx, exchange.RestSpot, publicAssetStatus+strings.ToUpper(symbol), &response)
	if err != nil {
		return nil, err
	}

	if response.Status != noError {
		return nil, errors.New(response.Message)
	}

	return &response, nil
}

// GetAssetStatusAll returns the withdrawal and deposit status for all symbols
func (b *Bithumb) GetAssetStatusAll(ctx context.Context) (*StatusAll, error) {
	var response StatusAll
	err := b.SendHTTPRequest(ctx, exchange.RestSpot, publicAssetStatus+"ALL", &response)
	if err != nil {
		return nil, err
	}

	if response.Status != noError {
		return nil, errors.New(response.Message)
	}

	return &response, nil
}

// GetTransactionHistory returns recent transactions
//
// symbol e.g. "btc"
func (b *Bithumb) GetTransactionHistory(ctx context.Context, symbol string) (TransactionHistory, error) {
	response := TransactionHistory{}
	path := publicTransactionHistory +
		strings.ToUpper(symbol)

	err := b.SendHTTPRequest(ctx, exchange.RestSpot, path, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}

	return response, nil
}

// GetAccountInformation returns account information based on the desired
// order/payment currencies
func (b *Bithumb) GetAccountInformation(ctx context.Context, orderCurrency, paymentCurrency string) (Account, error) {
	var response Account
	if orderCurrency == "" {
		return response, errSymbolIsEmpty
	}

	val := url.Values{}
	val.Add("order_currency", orderCurrency)
	if paymentCurrency != "" { // optional param, default is KRW
		val.Add("payment_currency", paymentCurrency)
	}

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateAccInfo, val, &response)
}

// GetAccountBalance returns customer wallet information
func (b *Bithumb) GetAccountBalance(ctx context.Context, c string) (FullBalance, error) {
	var response Balance
	fullBalance := FullBalance{
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
	}

	vals := url.Values{}
	if c != "" {
		vals.Set("currency", c)
	}

	err := b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateAccBalance, vals, &response)
	if err != nil {
		return fullBalance, err
	}

	// Added due to increasing of the usable currencies on exchange, usually
	// without notification, so we dont need to update structs later on
	for tag, datum := range response.Data {
		splitTag := strings.Split(tag, "_")
		if len(splitTag) < 2 {
			return fullBalance, fmt.Errorf("unhandled tag format: %q", splitTag)
		}

		c := splitTag[len(splitTag)-1]
		val := datum.Float64()

		switch splitTag[0] {
		case "available":
			fullBalance.Available[c] = val
		case "in":
			fullBalance.InUse[c] = val
		case "total":
			fullBalance.Total[c] = val
		case "misu":
			fullBalance.Misu[c] = val
		case "xcoin":
			fullBalance.Xcoin[c] = val
		default:
			return fullBalance, fmt.Errorf("getaccountbalance error tag name %s unhandled", splitTag)
		}
	}

	return fullBalance, nil
}

// GetWalletAddress returns customer wallet address
//
// currency e.g. btc, ltc or "", will default to btc without currency specified
func (b *Bithumb) GetWalletAddress(ctx context.Context, curr currency.Code) (WalletAddressRes, error) {
	response := WalletAddressRes{}
	params := url.Values{}
	params.Set("currency", curr.Upper().String())

	err := b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateWalletAdd, params, &response)
	if err != nil {
		return response, err
	}

	if response.Data.WalletAddress == "" {
		return response,
			fmt.Errorf("deposit address needs to be created via the Bithumb website before retrieval for currency %s",
				curr.String())
	}

	var address, tag string
	switch curr {
	case currency.XRP:
		splitStr := "&dt="
		if !strings.Contains(response.Data.WalletAddress, splitStr) {
			return response, errors.New("unable to parse XRP deposit address")
		}
		splitter := strings.Split(response.Data.WalletAddress, splitStr)
		address, tag = splitter[0], splitter[1]
	case currency.XLM, currency.BNB:
		splitStr := "&memo="
		if !strings.Contains(response.Data.WalletAddress, splitStr) {
			return response, fmt.Errorf("unable to parse %s deposit address", curr.String())
		}
		splitter := strings.Split(response.Data.WalletAddress, splitStr)
		address, tag = splitter[0], splitter[1]
	}

	if tag != "" {
		response.Data.WalletAddress = address
		response.Data.Tag = tag
	}

	return response, nil
}

// GetLastTransaction returns customer last transaction
func (b *Bithumb) GetLastTransaction(ctx context.Context) (LastTransactionTicker, error) {
	response := LastTransactionTicker{}

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateTicker, nil, &response)
}

// GetOrders returns order list
//
// orderID: order number registered for purchase/sales
// transactionType: transaction type(bid : purchase, ask : sell)
// count: Value : 1 ~1000 (default : 100)
// after: YYYY-MM-DD hh:mm:ss's UNIX Timestamp
// (2014-11-28 16:40:01 = 1417160401000)
func (b *Bithumb) GetOrders(ctx context.Context, orderID, transactionType string, count int64, after time.Time, orderCurrency, paymentCurrency currency.Code) (Orders, error) {
	response := Orders{}
	params := url.Values{}

	if orderCurrency.IsEmpty() {
		return response, currency.ErrCurrencyCodeEmpty
	}
	if !paymentCurrency.IsEmpty() {
		params.Set("payment_currency", paymentCurrency.Upper().String())
	}

	params.Set("order_currency", orderCurrency.Upper().String())

	if orderID != "" {
		params.Set("order_id", orderID)
	}

	if transactionType != "" {
		params.Set("type", transactionType)
	}

	if count > 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}

	if !after.IsZero() {
		params.Set("after", after.Format(time.DateTime))
	}

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateOrders, params, &response)
}

// GetUserTransactions returns customer transactions
func (b *Bithumb) GetUserTransactions(ctx context.Context, offset, count, searchType int64, orderCurrency, paymentCurrency currency.Code) (UserTransactions, error) {
	params := url.Values{}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if count > 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if searchType > 0 {
		params.Set("searchGb", strconv.FormatInt(searchType, 10))
	}
	if !orderCurrency.IsEmpty() {
		params.Set("order_currency", orderCurrency.String())
	}
	if !paymentCurrency.IsEmpty() {
		params.Set("payment_currency", paymentCurrency.String())
	}
	var response UserTransactions

	return response, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUserTrans, params, &response)
}

// PlaceTrade executes a trade order
//
// orderCurrency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
// transactionType: Transaction type(bid : purchase, ask : sales)
// units: Order quantity
// price: Transaction amount per currency
func (b *Bithumb) PlaceTrade(ctx context.Context, orderCurrency, transactionType string, units float64, price int64) (OrderPlace, error) {
	response := OrderPlace{}

	params := url.Values{}
	params.Set("order_currency", strings.ToUpper(orderCurrency))
	params.Set("payment_currency", "KRW")
	params.Set("type", strings.ToUpper(transactionType))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))
	params.Set("price", strconv.FormatInt(price, 10))

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privatePlaceTrade, params, &response)
}

// GetOrderDetails returns specific order details
//
// orderID: Order number registered for purchase/sales
// transactionType: Transaction type(bid : purchase, ask : sales)
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
func (b *Bithumb) GetOrderDetails(ctx context.Context, orderID, transactionType, currency string) (OrderDetails, error) {
	response := OrderDetails{}

	params := url.Values{}
	params.Set("order_id", strings.ToUpper(orderID))
	params.Set("type", transactionType)
	params.Set("currency", strings.ToUpper(currency))

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateOrderDetail, params, &response)
}

// CancelTrade cancels a customer purchase/sales transaction
// transactionType: Transaction type(bid : purchase, ask : sales)
// orderID: Order number registered for purchase/sales
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
func (b *Bithumb) CancelTrade(ctx context.Context, transactionType, orderID, currency string) (ActionStatus, error) {
	response := ActionStatus{}

	params := url.Values{}
	params.Set("order_id", strings.ToUpper(orderID))
	params.Set("type", strings.ToUpper(transactionType))
	params.Set("currency", strings.ToUpper(currency))

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateCancelTrade, nil, &response)
}

// WithdrawCrypto withdraws a customer currency to an address
//
// address: Currency withdrawing address
// destination: Currency withdrawal Destination Tag (when withdraw XRP) OR
// Currency withdrawal Payment ID (when withdraw XMR)
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM
// (default value: BTC)
// units: Quantity to withdraw currency
func (b *Bithumb) WithdrawCrypto(ctx context.Context, address, destination, currency string, units float64) (ActionStatus, error) {
	response := ActionStatus{}

	params := url.Values{}
	params.Set("address", address)
	if destination != "" {
		params.Set("destination", destination)
	}
	params.Set("currency", strings.ToUpper(currency))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateBTCWithdraw, params, &response)
}

// RequestKRWDepositDetails returns Bithumb banking details for deposit
// information
func (b *Bithumb) RequestKRWDepositDetails(ctx context.Context) (KRWDeposit, error) {
	response := KRWDeposit{}

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateKRWDeposit, nil, &response)
}

// RequestKRWWithdraw allows a customer KRW withdrawal request
//
// bank: Bankcode with bank name e.g. (bankcode)_(bankname)
// account: Withdrawing bank account number
// price: 	Withdrawing amount
func (b *Bithumb) RequestKRWWithdraw(ctx context.Context, bank, account string, price int64) (ActionStatus, error) {
	response := ActionStatus{}

	params := url.Values{}
	params.Set("bank", bank)
	params.Set("account", account)
	params.Set("price", strconv.FormatInt(price, 10))

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateKRWWithdraw, params, &response)
}

// MarketBuyOrder initiates a buy order through available order books
//
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
// units: Order quantity
func (b *Bithumb) MarketBuyOrder(ctx context.Context, pair currency.Pair, units float64) (MarketBuy, error) {
	response := MarketBuy{}

	params := url.Values{}
	params.Set("order_currency", strings.ToUpper(pair.Base.String()))
	params.Set("payment_currency", strings.ToUpper(pair.Quote.String()))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateMarketBuy, params, &response)
}

// MarketSellOrder initiates a sell order through available order books
//
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
// units: Order quantity
func (b *Bithumb) MarketSellOrder(ctx context.Context, pair currency.Pair, units float64) (MarketSell, error) {
	response := MarketSell{}

	params := url.Values{}
	params.Set("order_currency", strings.ToUpper(pair.Base.String()))
	params.Set("payment_currency", strings.ToUpper(pair.Quote.String()))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateMarketSell, params, &response)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *Bithumb) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	}
	return b.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to bithumb
func (b *Bithumb) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, path string, params url.Values, result any) error {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	if params == nil {
		params = url.Values{}
	}

	var intermediary json.RawMessage
	err = b.SendPayload(ctx, request.Auth, func() (*request.Item, error) {
		// This is time window sensitive
		n := strconv.FormatInt(time.Now().UnixMilli(), 10)

		params.Set("endpoint", path)

		payload := params.Encode()
		hmacPayload := path + string('\x00') + payload + string('\x00') + n

		hmac, err := crypto.GetHMAC(crypto.HashSHA512, []byte(hmacPayload), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		headers["Api-Key"] = creds.Key
		headers["Api-Sign"] = base64.StdEncoding.EncodeToString(([]byte(hex.EncodeToString(hmac))))
		headers["Api-Nonce"] = n
		headers["Content-Type"] = "application/x-www-form-urlencoded"

		return &request.Item{
			Method:        http.MethodPost,
			Path:          endpoint + path,
			Headers:       headers,
			Body:          bytes.NewBufferString(payload),
			Result:        &intermediary,
			NonceEnabled:  true,
			Verbose:       b.Verbose,
			HTTPDebugging: b.HTTPDebugging,
			HTTPRecording: b.HTTPRecording,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}

	errCapture := struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{}
	err = json.Unmarshal(intermediary, &errCapture)
	if err == nil {
		if errCapture.Status != "" && errCapture.Status != noError {
			return fmt.Errorf("sendAuthenticatedAPIRequest error code: %s message:%s",
				errCapture.Status,
				errCode[errCapture.Status])
		}
	}

	return json.Unmarshal(intermediary, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Bithumb) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	case exchange.CryptocurrencyDepositFee:
		fee = getDepositFee(feeBuilder.Pair.Base, feeBuilder.Amount)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.InternationalBankWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.FiatCurrency)
	case exchange.OfflineTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// calculateTradingFee returns fee when performing a trade
func calculateTradingFee(purchasePrice, amount float64) float64 {
	return 0.0025 * amount * purchasePrice
}

// getDepositFee returns fee on a currency when depositing small amounts to bithumb
func getDepositFee(c currency.Code, amount float64) float64 {
	var fee float64

	switch c {
	case currency.BTC:
		if amount <= 0.005 {
			fee = 0.001
		}
	case currency.LTC:
		if amount <= 0.3 {
			fee = 0.01
		}
	case currency.DASH:
		if amount <= 0.04 {
			fee = 0.01
		}
	case currency.BCH:
		if amount <= 0.03 {
			fee = 0.001
		}
	case currency.ZEC:
		if amount <= 0.02 {
			fee = 0.001
		}
	case currency.BTG:
		if amount <= 0.15 {
			fee = 0.001
		}
	}

	return fee
}

// getWithdrawalFee returns fee on a currency when withdrawing out of bithumb
func getWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

var errCode = map[string]string{
	"5100": "Bad Request",
	"5200": "Not Member",
	"5300": "Invalid Apikey",
	"5302": "Method Not Allowed",
	"5400": "Database Fail",
	"5500": "Invalid Parameter",
	"5600": "CUSTOM NOTICE (상황별 에러 메시지 출력) usually means transaction not allowed",
	"5900": "Unknown Error",
}

// GetCandleStick returns candle stick data for requested pair
func (b *Bithumb) GetCandleStick(ctx context.Context, symbol, interval string) (resp *OHLCVResponse, err error) {
	path := publicCandleStick + symbol + "/" + interval
	err = b.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
	return
}

// FetchExchangeLimits fetches spot order execution limits
func (b *Bithumb) FetchExchangeLimits(ctx context.Context) ([]order.MinMaxLevel, error) {
	ticks, err := b.GetAllTickers(ctx)
	if err != nil {
		return nil, err
	}

	limits := make([]order.MinMaxLevel, 0, len(ticks))
	for code, data := range ticks {
		limits = append(limits, order.MinMaxLevel{
			Pair:              currency.NewPair(currency.NewCode(code), currency.KRW),
			Asset:             asset.Spot,
			MinimumBaseAmount: getAmountMinimum(data.ClosingPrice),
		})
	}
	return limits, nil
}

// getAmountMinimum derives the minimum amount based on current price. This
// keeps amount in line with front end, rounded to 4 decimal places. As
// transaction policy:
// https://en.bithumb.com/customer_support/info_guide?seq=537&categorySeq=302
// Seems to not be inline with front end limits.
func getAmountMinimum(unitPrice float64) float64 {
	if unitPrice <= 0 {
		return 0
	}
	ratio := 500 / unitPrice
	pow := math.Pow(10, float64(4))
	return math.Ceil(ratio*pow) / pow // Round up our units
}
