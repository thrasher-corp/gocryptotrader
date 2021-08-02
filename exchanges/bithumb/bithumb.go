package bithumb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	apiURL = "https://api.bithumb.com"

	noError = "0000"

	publicTicker             = "/public/ticker/"
	publicOrderBook          = "/public/orderbook/"
	publicTransactionHistory = "/public/transaction_history/"
	publicCandleStick        = "/public/candlestick/"

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

// Bithumb is the overarching type across the Bithumb package
type Bithumb struct {
	exchange.Base
}

// GetTradablePairs returns a list of tradable currencies
func (b *Bithumb) GetTradablePairs() ([]string, error) {
	result, err := b.GetAllTickers()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range result {
		currencies = append(currencies, x)
	}
	return currencies, nil
}

// GetTicker returns ticker information
//
// symbol e.g. "btc"
func (b *Bithumb) GetTicker(symbol string) (Ticker, error) {
	var response TickerResponse
	err := b.SendHTTPRequest(exchange.RestSpot, publicTicker+strings.ToUpper(symbol), &response)
	if err != nil {
		return response.Data, err
	}

	if response.Status != noError {
		return response.Data, errors.New(response.Message)
	}

	return response.Data, nil
}

// GetAllTickers returns all ticker information
func (b *Bithumb) GetAllTickers() (map[string]Ticker, error) {
	var response TickersResponse
	err := b.SendHTTPRequest(exchange.RestSpot, publicTicker+"all", &response)
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
func (b *Bithumb) GetOrderBook(symbol string) (Orderbook, error) {
	response := Orderbook{}
	err := b.SendHTTPRequest(exchange.RestSpot, publicOrderBook+strings.ToUpper(symbol), &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}

	return response, nil
}

// GetTransactionHistory returns recent transactions
//
// symbol e.g. "btc"
func (b *Bithumb) GetTransactionHistory(symbol string) (TransactionHistory, error) {
	response := TransactionHistory{}
	path := publicTransactionHistory +
		strings.ToUpper(symbol)

	err := b.SendHTTPRequest(exchange.RestSpot, path, &response)
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
func (b *Bithumb) GetAccountInformation(orderCurrency, paymentCurrency string) (Account, error) {
	var response Account
	if orderCurrency == "" {
		return response, errors.New("order currency must be set")
	}

	val := url.Values{}
	val.Add("order_currency", orderCurrency)
	if paymentCurrency != "" { // optional param, default is KRW
		val.Add("payment_currency", paymentCurrency)
	}

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateAccInfo, val, &response)
}

// GetAccountBalance returns customer wallet information
func (b *Bithumb) GetAccountBalance(c string) (FullBalance, error) {
	var response Balance
	var fullBalance = FullBalance{
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

	err := b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateAccBalance, vals, &response)
	if err != nil {
		return fullBalance, err
	}

	// Added due to increasing of the usuable currencies on exchange, usually
	// without notificatation, so we dont need to update structs later on
	for tag, datum := range response.Data {
		splitTag := strings.Split(tag, "_")
		c := splitTag[len(splitTag)-1]
		var val float64
		if reflect.TypeOf(datum).String() != "float64" {
			val, err = strconv.ParseFloat(datum.(string), 64)
			if err != nil {
				return fullBalance, err
			}
		} else {
			val = datum.(float64)
		}

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
			return fullBalance, fmt.Errorf("getaccountbalance error tag name %s unhandled",
				splitTag)
		}
	}

	return fullBalance, nil
}

// GetWalletAddress returns customer wallet address
//
// currency e.g. btc, ltc or "", will default to btc without currency specified
func (b *Bithumb) GetWalletAddress(currency string) (WalletAddressRes, error) {
	response := WalletAddressRes{}
	params := url.Values{}
	params.Set("currency", strings.ToUpper(currency))

	err := b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateWalletAdd, params, &response)
	if err != nil {
		return response, err
	}

	if response.Data.WalletAddress == "" {
		return response,
			fmt.Errorf("deposit address needs to be created via the Bithumb website before retrieval for currency %s",
				currency)
	}

	return response, nil
}

// GetLastTransaction returns customer last transaction
func (b *Bithumb) GetLastTransaction() (LastTransactionTicker, error) {
	response := LastTransactionTicker{}

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateTicker, nil, &response)
}

// GetOrders returns order list
//
// orderID: order number registered for purchase/sales
// transactionType: transaction type(bid : purchase, ask : sell)
// count: Value : 1 ~1000 (default : 100)
// after: YYYY-MM-DD hh:mm:ss's UNIX Timestamp
// (2014-11-28 16:40:01 = 1417160401000)
func (b *Bithumb) GetOrders(orderID, transactionType, count, after, currency string) (Orders, error) {
	response := Orders{}
	params := url.Values{}

	if currency == "" {
		return response, errors.New("order currency is required")
	}

	params.Set("order_currency", strings.ToUpper(currency))

	if len(orderID) > 0 {
		params.Set("order_id", orderID)
	}

	if len(transactionType) > 0 {
		params.Set("type", transactionType)
	}

	if len(count) > 0 {
		params.Set("count", count)
	}

	if len(after) > 0 {
		params.Set("after", after)
	}

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateOrders, params, &response)
}

// GetUserTransactions returns customer transactions
func (b *Bithumb) GetUserTransactions() (UserTransactions, error) {
	response := UserTransactions{}

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateUserTrans, nil, &response)
}

// PlaceTrade executes a trade order
//
// orderCurrency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
// transactionType: Transaction type(bid : purchase, ask : sales)
// units: Order quantity
// price: Transaction amount per currency
func (b *Bithumb) PlaceTrade(orderCurrency, transactionType string, units float64, price int64) (OrderPlace, error) {
	response := OrderPlace{}

	params := url.Values{}
	params.Set("order_currency", strings.ToUpper(orderCurrency))
	params.Set("Payment_currency", "KRW")
	params.Set("type", strings.ToUpper(transactionType))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))
	params.Set("price", strconv.FormatInt(price, 10))

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privatePlaceTrade, params, &response)
}

// ModifyTrade modifies an order already on the exchange books
func (b *Bithumb) ModifyTrade(orderID, orderCurrency, transactionType string, units float64, price int64) (OrderPlace, error) {
	response := OrderPlace{}

	params := url.Values{}
	params.Set("order_currency", strings.ToUpper(orderCurrency))
	params.Set("Payment_currency", "KRW")
	params.Set("type", strings.ToUpper(transactionType))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))
	params.Set("price", strconv.FormatInt(price, 10))
	params.Set("order_id", orderID)

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privatePlaceTrade, params, &response)
}

// GetOrderDetails returns specific order details
//
// orderID: Order number registered for purchase/sales
// transactionType: Transaction type(bid : purchase, ask : sales)
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
func (b *Bithumb) GetOrderDetails(orderID, transactionType, currency string) (OrderDetails, error) {
	response := OrderDetails{}

	params := url.Values{}
	params.Set("order_id", strings.ToUpper(orderID))
	params.Set("type", transactionType)
	params.Set("currency", strings.ToUpper(currency))

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateOrderDetail, params, &response)
}

// CancelTrade cancels a customer purchase/sales transaction
// transactionType: Transaction type(bid : purchase, ask : sales)
// orderID: Order number registered for purchase/sales
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
func (b *Bithumb) CancelTrade(transactionType, orderID, currency string) (ActionStatus, error) {
	response := ActionStatus{}

	params := url.Values{}
	params.Set("order_id", strings.ToUpper(orderID))
	params.Set("type", strings.ToUpper(transactionType))
	params.Set("currency", strings.ToUpper(currency))

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateCancelTrade, nil, &response)
}

// WithdrawCrypto withdraws a customer currency to an address
//
// address: Currency withdrawing address
// destination: Currency withdrawal Destination Tag (when withdraw XRP) OR
// Currency withdrawal Payment Id (when withdraw XMR)
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM
// (default value: BTC)
// units: Quantity to withdraw currency
func (b *Bithumb) WithdrawCrypto(address, destination, currency string, units float64) (ActionStatus, error) {
	response := ActionStatus{}

	params := url.Values{}
	params.Set("address", address)
	if len(destination) > 0 {
		params.Set("destination", destination)
	}
	params.Set("currency", strings.ToUpper(currency))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateBTCWithdraw, params, &response)
}

// RequestKRWDepositDetails returns Bithumb banking details for deposit
// information
func (b *Bithumb) RequestKRWDepositDetails() (KRWDeposit, error) {
	response := KRWDeposit{}

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateKRWDeposit, nil, &response)
}

// RequestKRWWithdraw allows a customer KRW withdrawal request
//
// bank: Bankcode with bank name e.g. (bankcode)_(bankname)
// account: Withdrawing bank account number
// price: 	Withdrawing amount
func (b *Bithumb) RequestKRWWithdraw(bank, account string, price int64) (ActionStatus, error) {
	response := ActionStatus{}

	params := url.Values{}
	params.Set("bank", bank)
	params.Set("account", account)
	params.Set("price", strconv.FormatInt(price, 10))

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateKRWWithdraw, params, &response)
}

// MarketBuyOrder initiates a buy order through available order books
//
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
// units: Order quantity
func (b *Bithumb) MarketBuyOrder(pair currency.Pair, units float64) (MarketBuy, error) {
	response := MarketBuy{}

	params := url.Values{}
	params.Set("order_currency", strings.ToUpper(pair.Base.String()))
	params.Set("payment_currency", strings.ToUpper(pair.Quote.String()))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateMarketBuy, params, &response)
}

// MarketSellOrder initiates a sell order through available order books
//
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
// units: Order quantity
func (b *Bithumb) MarketSellOrder(pair currency.Pair, units float64) (MarketSell, error) {
	response := MarketSell{}

	params := url.Values{}
	params.Set("order_currency", strings.ToUpper(pair.Base.String()))
	params.Set("payment_currency", strings.ToUpper(pair.Quote.String()))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))

	return response,
		b.SendAuthenticatedHTTPRequest(exchange.RestSpot, privateMarketSell, params, &response)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *Bithumb) SendHTTPRequest(ep exchange.URL, path string, result interface{}) error {
	endpoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	return b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to bithumb
func (b *Bithumb) SendAuthenticatedHTTPRequest(ep exchange.URL, path string, params url.Values, result interface{}) error {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", b.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	endpoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	if params == nil {
		params = url.Values{}
	}

	// This is time window sensitive
	tnMS := time.Now().UnixNano() / int64(time.Millisecond)
	n := strconv.FormatInt(tnMS, 10)

	params.Set("endpoint", path)
	payload := params.Encode()
	hmacPayload := path + string('\x00') + payload + string('\x00') + n
	hmac := crypto.GetHMAC(crypto.HashSHA512,
		[]byte(hmacPayload),
		[]byte(b.API.Credentials.Secret))
	hmacStr := crypto.HexEncodeToString(hmac)

	headers := make(map[string]string)
	headers["Api-Key"] = b.API.Credentials.Key
	headers["Api-Sign"] = crypto.Base64Encode([]byte(hmacStr))
	headers["Api-Nonce"] = n
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	var intermediary json.RawMessage

	errCapture := struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{}

	err = b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodPost,
		Path:          endpoint + path,
		Headers:       headers,
		Body:          bytes.NewBufferString(payload),
		Result:        &intermediary,
		AuthRequest:   true,
		NonceEnabled:  true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
		Endpoint:      request.Auth})
	if err != nil {
		return err
	}

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
func (b *Bithumb) GetCandleStick(symbol, interval string) (resp OHLCVResponse, err error) {
	path := publicCandleStick + symbol + "/" + interval
	err = b.SendHTTPRequest(exchange.RestSpot, path, &resp)
	return
}

// FetchExchangeLimits fetches spot order execution limits
func (b *Bithumb) FetchExchangeLimits() ([]order.MinMaxLevel, error) {
	ticks, err := b.GetAllTickers()
	if err != nil {
		return nil, err
	}

	var limits []order.MinMaxLevel
	for code, data := range ticks {
		c := currency.NewCode(code)
		cp := currency.NewPair(c, currency.KRW)
		if err != nil {
			return nil, err
		}

		limits = append(limits, order.MinMaxLevel{
			Pair:      cp,
			Asset:     asset.Spot,
			MinAmount: getAmountMinimum(data.ClosingPrice),
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
