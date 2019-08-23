package bithumb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	apiURL = "https://api.bithumb.com"

	noError = "0000"

	// Public API
	requestsPerSecondPublicAPI = 20

	publicTicker             = "/public/ticker/"
	publicOrderBook          = "/public/orderbook/"
	publicTransactionHistory = "/public/transaction_history/"

	// Private API
	requestsPerSecondPrivateAPI = 10

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

	bithumbAuthRate   = 10
	bithumbUnauthRate = 20
)

// Bithumb is the overarching type across the Bithumb package
type Bithumb struct {
	exchange.Base
}

// SetDefaults sets the basic defaults for Bithumb
func (b *Bithumb) SetDefaults() {
	b.Name = "Bithumb"
	b.Enabled = false
	b.Verbose = false
	b.RESTPollingDelay = 10
	b.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.AutoWithdrawFiat
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.RequestCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Index = "KRW"
	b.AssetTypes = []string{ticker.Spot}
	b.SupportsAutoPairUpdating = true
	b.SupportsRESTTickerBatching = true
	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, bithumbAuthRate),
		request.NewRateLimit(time.Second, bithumbUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	b.APIUrlDefault = apiURL
	b.APIUrl = b.APIUrlDefault
	b.Websocket = wshandler.New()
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bithumb) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		b.SetHTTPClientTimeout(exch.HTTPTimeout)
		b.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.HTTPDebugging = exch.HTTPDebugging
		b.Websocket.SetWsStatusAndConnection(exch.Websocket)
		b.BaseCurrencies = exch.BaseCurrencies
		b.AvailablePairs = exch.AvailablePairs
		b.EnabledPairs = exch.EnabledPairs
		err := b.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
	}
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
	path := fmt.Sprintf("%s%s%s",
		b.APIUrl,
		publicTicker,
		strings.ToUpper(symbol))

	err := b.SendHTTPRequest(path, &response)
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
	path := fmt.Sprintf("%s%s%s", b.APIUrl, publicTicker, "all")

	err := b.SendHTTPRequest(path, &response)
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
		err := common.JSONDecode(v, &newTicker)
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
	path := fmt.Sprintf("%s%s%s", b.APIUrl, publicOrderBook, common.StringToUpper(symbol))

	err := b.SendHTTPRequest(path, &response)
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
	path := fmt.Sprintf("%s%s%s", b.APIUrl, publicTransactionHistory, common.StringToUpper(symbol))

	err := b.SendHTTPRequest(path, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}

	return response, nil
}

// GetAccountInformation returns account information by singular currency
func (b *Bithumb) GetAccountInformation(currency string) (Account, error) {
	response := Account{}

	val := url.Values{}
	if currency != "" {
		val.Set("currency", currency)
	}

	return response,
		b.SendAuthenticatedHTTPRequest(privateAccInfo, val, &response)
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

	err := b.SendAuthenticatedHTTPRequest(privateAccBalance, vals, &response)
	if err != nil {
		return fullBalance, err
	}

	// Added due to increasing of the usuable currencies on exchange, usually
	// without notificatation, so we dont need to update structs later on
	for tag, datum := range response.Data {
		splitTag := common.SplitStrings(tag, "_")
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
	params.Set("currency", common.StringToUpper(currency))

	err := b.SendAuthenticatedHTTPRequest(privateWalletAdd, params, &response)
	if err != nil {
		return response, err
	}

	if response.Data.WalletAddress == "" {
		return response,
			fmt.Errorf("deposit address needs to be created via the Bithumb website before retreival for currency %s",
				currency)
	}

	return response, nil
}

// GetLastTransaction returns customer last transaction
func (b *Bithumb) GetLastTransaction() (LastTransactionTicker, error) {
	response := LastTransactionTicker{}

	return response,
		b.SendAuthenticatedHTTPRequest(privateTicker, nil, &response)
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

	if len(currency) > 0 {
		params.Set("currency", common.StringToUpper(currency))
	}

	return response,
		b.SendAuthenticatedHTTPRequest(privateOrders, params, &response)
}

// GetUserTransactions returns customer transactions
func (b *Bithumb) GetUserTransactions() (UserTransactions, error) {
	response := UserTransactions{}

	return response,
		b.SendAuthenticatedHTTPRequest(privateUserTrans, nil, &response)
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
	params.Set("order_currency", common.StringToUpper(orderCurrency))
	params.Set("Payment_currency", "KRW")
	params.Set("type", common.StringToUpper(transactionType))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))
	params.Set("price", strconv.FormatInt(price, 10))

	return response,
		b.SendAuthenticatedHTTPRequest(privatePlaceTrade, params, &response)
}

// ModifyTrade modifies an order already on the exchange books
func (b *Bithumb) ModifyTrade(orderID, orderCurrency, transactionType string, units float64, price int64) (OrderPlace, error) {
	response := OrderPlace{}

	params := url.Values{}
	params.Set("order_currency", common.StringToUpper(orderCurrency))
	params.Set("Payment_currency", "KRW")
	params.Set("type", common.StringToUpper(transactionType))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))
	params.Set("price", strconv.FormatInt(price, 10))
	params.Set("order_id", orderID)

	return response,
		b.SendAuthenticatedHTTPRequest(privatePlaceTrade, params, &response)
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
	params.Set("order_id", common.StringToUpper(orderID))
	params.Set("type", transactionType)
	params.Set("currency", common.StringToUpper(currency))

	return response,
		b.SendAuthenticatedHTTPRequest(privateOrderDetail, params, &response)
}

// CancelTrade cancels a customer purchase/sales transaction
// transactionType: Transaction type(bid : purchase, ask : sales)
// orderID: Order number registered for purchase/sales
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
func (b *Bithumb) CancelTrade(transactionType, orderID, currency string) (ActionStatus, error) {
	response := ActionStatus{}

	params := url.Values{}
	params.Set("order_id", common.StringToUpper(orderID))
	params.Set("type", common.StringToUpper(transactionType))
	params.Set("currency", common.StringToUpper(currency))

	return response,
		b.SendAuthenticatedHTTPRequest(privateCancelTrade, nil, &response)
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
	params.Set("currency", common.StringToUpper(currency))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))

	return response,
		b.SendAuthenticatedHTTPRequest(privateBTCWithdraw, params, &response)
}

// RequestKRWDepositDetails returns Bithumb banking details for deposit
// information
func (b *Bithumb) RequestKRWDepositDetails() (KRWDeposit, error) {
	response := KRWDeposit{}

	return response,
		b.SendAuthenticatedHTTPRequest(privateKRWDeposit, nil, &response)
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
		b.SendAuthenticatedHTTPRequest(privateKRWWithdraw, params, &response)
}

// MarketBuyOrder initiates a buy order through available order books
//
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
// units: Order quantity
func (b *Bithumb) MarketBuyOrder(currency string, units float64) (MarketBuy, error) {
	response := MarketBuy{}

	params := url.Values{}
	params.Set("currency", common.StringToUpper(currency))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))

	return response,
		b.SendAuthenticatedHTTPRequest(privateMarketBuy, params, &response)
}

// MarketSellOrder initiates a sell order through available order books
//
// currency: BTC, ETH, DASH, LTC, ETC, XRP, BCH, XMR, ZEC, QTUM, BTG, EOS
// (default value: BTC)
// units: Order quantity
func (b *Bithumb) MarketSellOrder(currency string, units float64) (MarketSell, error) {
	response := MarketSell{}

	params := url.Values{}
	params.Set("currency", common.StringToUpper(currency))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))

	return response,
		b.SendAuthenticatedHTTPRequest(privateMarketSell, params, &response)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *Bithumb) SendHTTPRequest(path string, result interface{}) error {
	return b.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		result,
		false,
		false,
		b.Verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to bithumb
func (b *Bithumb) SendAuthenticatedHTTPRequest(path string, params url.Values, result interface{}) error {
	if !b.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}

	if params == nil {
		params = url.Values{}
	}

	n := b.Requester.GetNonceMilli().String()

	params.Set("endpoint", path)
	payload := params.Encode()
	hmacPayload := path + string(0) + payload + string(0) + n
	hmac := common.GetHMAC(common.HashSHA512,
		[]byte(hmacPayload),
		[]byte(b.APISecret))
	hmacStr := common.HexEncodeToString(hmac)

	headers := make(map[string]string)
	headers["Api-Key"] = b.APIKey
	headers["Api-Sign"] = common.Base64Encode([]byte(hmacStr))
	headers["Api-Nonce"] = n
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	var intermediary json.RawMessage

	errCapture := struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{}

	err := b.SendPayload(http.MethodPost,
		b.APIUrl+path,
		headers,
		bytes.NewBufferString(payload),
		&intermediary,
		true,
		true,
		b.Verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
	if err != nil {
		return err
	}

	err = common.JSONDecode(intermediary, &errCapture)
	if err == nil {
		if errCapture.Status != "" && errCapture.Status != noError {
			return fmt.Errorf("sendAuthenticatedAPIRequest error code: %s message:%s",
				errCapture.Status,
				errCode[errCapture.Status])
		}
	}

	return common.JSONDecode(intermediary, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Bithumb) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	case exchange.CyptocurrencyDepositFee:
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
