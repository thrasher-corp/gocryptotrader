package bithumb

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	apiURL = "https://api.bithumb.com"

	noError = "0000"

	// Public API
	requestsPerSecondPublicAPI = 20

	publicTicker            = "/public/ticker/"
	publicOrderBook         = "/public/orderbook/"
	publicRecentTransaction = "/public/recent_transactions/"

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
	b.Websocket = false
	b.RESTPollingDelay = 10
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.RequestCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Index = "KRW"
	b.AssetTypes = []string{ticker.Spot}
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bithumb) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := b.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetTicker returns ticker information
//
// symbol e.g. "btc"
func (b *Bithumb) GetTicker(symbol string) (Ticker, error) {
	response := Ticker{}
	path := fmt.Sprintf("%s%s%s", apiURL, publicTicker, common.StringToUpper(symbol))

	return response, common.SendHTTPGetRequest(path, true, b.Verbose, &response)
}

// GetOrderBook returns current orderbook
//
// symbol e.g. "btc"
func (b *Bithumb) GetOrderBook(symbol string) (Orderbook, error) {
	response := Orderbook{}
	path := fmt.Sprintf("%s%s%s", apiURL, publicOrderBook, common.StringToUpper(symbol))

	return response, common.SendHTTPGetRequest(path, true, b.Verbose, &response)
}

// GetRecentTransactions returns recent transactions
//
// symbol e.g. "btc"
func (b *Bithumb) GetRecentTransactions(symbol string) (RecentTransactions, error) {
	response := RecentTransactions{}
	path := fmt.Sprintf("%s%s%s", apiURL, publicRecentTransaction, common.StringToUpper(symbol))

	return response, common.SendHTTPGetRequest(path, true, b.Verbose, &response)
}

// GetAccountInfo returns account information
func (b *Bithumb) GetAccountInfo() (Account, error) {
	response := Account{}

	err := b.SendAuthenticatedHTTPRequest(privateAccInfo, nil, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// GetAccountBalance returns customer wallet information
func (b *Bithumb) GetAccountBalance() (Balance, error) {
	response := Balance{}

	err := b.SendAuthenticatedHTTPRequest(privateAccBalance, nil, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
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

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// GetLastTransaction returns customer last transaction
func (b *Bithumb) GetLastTransaction() (LastTransactionTicker, error) {
	response := LastTransactionTicker{}

	err := b.SendAuthenticatedHTTPRequest(privateTicker, nil, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
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
	params.Set("order_id", orderID)
	params.Set("type", transactionType)
	params.Set("count", count)
	params.Set("after", after)
	params.Set("currency", common.StringToUpper(currency))

	err := b.SendAuthenticatedHTTPRequest(privateOrders, params, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// GetUserTransactions returns customer transactions
func (b *Bithumb) GetUserTransactions() (UserTransactions, error) {
	response := UserTransactions{}

	err := b.SendAuthenticatedHTTPRequest(privateUserTrans, nil, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
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

	err := b.SendAuthenticatedHTTPRequest(privatePlaceTrade, params, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
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
	params.Set("type", common.StringToUpper(transactionType))
	params.Set("currency", common.StringToUpper(currency))

	err := b.SendAuthenticatedHTTPRequest(privateOrderDetail, params, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
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

	err := b.SendAuthenticatedHTTPRequest(privateCancelTrade, nil, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
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
	params.Set("destination", destination)
	params.Set("currency", common.StringToUpper(currency))
	params.Set("units", strconv.FormatFloat(units, 'f', -1, 64))

	err := b.SendAuthenticatedHTTPRequest(privateBTCWithdraw, params, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// RequestKRWDepositDetails returns Bithumb banking details for deposit
// information
func (b *Bithumb) RequestKRWDepositDetails() (KRWDeposit, error) {
	response := KRWDeposit{}

	err := b.SendAuthenticatedHTTPRequest(privateKRWDeposit, nil, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
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

	err := b.SendAuthenticatedHTTPRequest(privateKRWWithdraw, params, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
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

	err := b.SendAuthenticatedHTTPRequest(privateMarketBuy, params, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
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

	err := b.SendAuthenticatedHTTPRequest(privateMarketSell, params, &response)
	if err != nil {
		return response, err
	}

	if response.Status != noError {
		return response, errors.New(response.Message)
	}
	return response, nil
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to bithumb
func (b *Bithumb) SendAuthenticatedHTTPRequest(path string, params url.Values, result interface{}) error {
	if !b.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}

	if params == nil {
		params = url.Values{}
	}

	if b.Nonce.Get() == 0 {
		b.Nonce.Set(time.Now().UnixNano() / int64(time.Millisecond))
	} else {
		b.Nonce.Inc()
	}

	params.Set("endpoint", path)
	payload := params.Encode()
	hmacPayload := path + string(0) + payload + string(0) + b.Nonce.String()
	hmac := common.GetHMAC(common.HashSHA512, []byte(hmacPayload), []byte(b.APISecret))
	hmacStr := common.HexEncodeToString(hmac)

	headers := make(map[string]string)
	headers["Api-Key"] = b.APIKey
	headers["Api-Sign"] = common.Base64Encode([]byte(hmacStr))
	headers["Api-Nonce"] = b.Nonce.String()
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest(
		"POST", apiURL+path, headers, bytes.NewBufferString(payload),
	)
	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Received raw: \n%s\n", resp)
	}

	if err = common.JSONDecode([]byte(resp), &result); err != nil {
		return errors.New("sendAuthenticatedHTTPRequest: Unable to JSON Unmarshal response." + err.Error())
	}
	return nil
}
