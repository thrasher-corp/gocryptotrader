package bittrex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	bittrexAPIURL              = "https://bittrex.com/api/v1.1"
	bittrexAPIVersion          = "v1.1"
	bittrexMaxOpenOrders       = 500
	bittrexMaxOrderCountPerDay = 200000

	// Returned messages from Bittrex API
	bittrexAddressGenerating      = "ADDRESS_GENERATING"
	bittrexErrorMarketNotProvided = "MARKET_NOT_PROVIDED"
	bittrexErrorInvalidMarket     = "INVALID_MARKET"
	bittrexErrorAPIKeyInvalid     = "APIKEY_INVALID"
	bittrexErrorInvalidPermission = "INVALID_PERMISSION"

	// Public requests
	bittrexAPIGetMarkets         = "public/getmarkets"
	bittrexAPIGetCurrencies      = "public/getcurrencies"
	bittrexAPIGetTicker          = "public/getticker"
	bittrexAPIGetMarketSummaries = "public/getmarketsummaries"
	bittrexAPIGetMarketSummary   = "public/getmarketsummary"
	bittrexAPIGetOrderbook       = "public/getorderbook"
	bittrexAPIGetMarketHistory   = "public/getmarkethistory"

	// Market requests
	bittrexAPIBuyLimit      = "market/buylimit"
	bittrexAPISellLimit     = "market/selllimit"
	bittrexAPICancel        = "market/cancel"
	bittrexAPIGetOpenOrders = "market/getopenorders"

	// Account requests
	bittrexAPIGetBalances          = "account/getbalances"
	bittrexAPIGetBalance           = "account/getbalance"
	bittrexAPIGetDepositAddress    = "account/getdepositaddress"
	bittrexAPIWithdraw             = "account/withdraw"
	bittrexAPIGetOrder             = "account/getorder"
	bittrexAPIGetOrderHistory      = "account/getorderhistory"
	bittrexAPIGetWithdrawalHistory = "account/getwithdrawalhistory"
	bittrexAPIGetDepositHistory    = "account/getdeposithistory"

	bittrexRateInterval = time.Minute
	bittrexRequestRate  = 60
	bittrexTimeLayout   = "2006-01-02T15:04:05"
)

// Bittrex is the overaching type across the bittrex methods
type Bittrex struct {
	exchange.Base
}

// GetMarkets is used to get the open and available trading markets at Bittrex
// along with other meta data.
func (b *Bittrex) GetMarkets() (Market, error) {
	var markets Market
	path := fmt.Sprintf("%s/%s/", b.API.Endpoints.URL, bittrexAPIGetMarkets)

	if err := b.SendHTTPRequest(path, &markets); err != nil {
		return markets, err
	}

	if !markets.Success {
		return markets, errors.New(markets.Message)
	}
	return markets, nil
}

// GetCurrencies is used to get all supported currencies at Bittrex
func (b *Bittrex) GetCurrencies() (Currency, error) {
	var currencies Currency
	path := fmt.Sprintf("%s/%s/", b.API.Endpoints.URL, bittrexAPIGetCurrencies)

	if err := b.SendHTTPRequest(path, &currencies); err != nil {
		return currencies, err
	}

	if !currencies.Success {
		return currencies, errors.New(currencies.Message)
	}
	return currencies, nil
}

// GetTicker sends a public get request and returns current ticker information
// on the supplied currency. Example currency input param "btc-ltc".
func (b *Bittrex) GetTicker(currencyPair string) (Ticker, error) {
	tick := Ticker{}
	path := fmt.Sprintf("%s/%s?market=%s", b.API.Endpoints.URL, bittrexAPIGetTicker,
		strings.ToUpper(currencyPair),
	)

	if err := b.SendHTTPRequest(path, &tick); err != nil {
		return tick, err
	}

	if !tick.Success {
		return tick, errors.New(tick.Message)
	}
	return tick, nil
}

// GetMarketSummaries is used to get the last 24 hour summary of all active
// exchanges
func (b *Bittrex) GetMarketSummaries() (MarketSummary, error) {
	var summaries MarketSummary
	path := fmt.Sprintf("%s/%s/", b.API.Endpoints.URL, bittrexAPIGetMarketSummaries)

	if err := b.SendHTTPRequest(path, &summaries); err != nil {
		return summaries, err
	}

	if !summaries.Success {
		return summaries, errors.New(summaries.Message)
	}
	return summaries, nil
}

// GetMarketSummary is used to get the last 24 hour summary of all active
// exchanges by currency pair (btc-ltc).
func (b *Bittrex) GetMarketSummary(currencyPair string) (MarketSummary, error) {
	var summary MarketSummary
	path := fmt.Sprintf("%s/%s?market=%s", b.API.Endpoints.URL,
		bittrexAPIGetMarketSummary, strings.ToLower(currencyPair),
	)

	if err := b.SendHTTPRequest(path, &summary); err != nil {
		return summary, err
	}

	if !summary.Success {
		return summary, errors.New(summary.Message)
	}
	return summary, nil
}

// GetOrderbook method returns current order book information by currency, type
// & depth.
// "Currency Pair" ie btc-ltc
// "Category" either "buy", "sell" or "both"; for ease of use and reduced
// complexity this function is set to "both"
// "Depth" max depth is 50 but you can literally set it any integer you want and
// it returns full depth. So depth default is 50.
func (b *Bittrex) GetOrderbook(currencyPair string) (OrderBooks, error) {
	var orderbooks OrderBooks
	path := fmt.Sprintf("%s/%s?market=%s&type=both&depth=50", b.API.Endpoints.URL,
		bittrexAPIGetOrderbook, strings.ToUpper(currencyPair),
	)

	if err := b.SendHTTPRequest(path, &orderbooks); err != nil {
		return orderbooks, err
	}

	if !orderbooks.Success {
		return orderbooks, errors.New(orderbooks.Message)
	}
	return orderbooks, nil
}

// GetMarketHistory retrieves the latest trades that have occurred for a specific
// market
func (b *Bittrex) GetMarketHistory(currencyPair string) (MarketHistory, error) {
	var marketHistoriae MarketHistory
	path := fmt.Sprintf("%s/%s?market=%s", b.API.Endpoints.URL,
		bittrexAPIGetMarketHistory, strings.ToUpper(currencyPair),
	)

	if err := b.SendHTTPRequest(path, &marketHistoriae); err != nil {
		return marketHistoriae, err
	}

	if !marketHistoriae.Success {
		return marketHistoriae, errors.New(marketHistoriae.Message)
	}
	return marketHistoriae, nil
}

// PlaceBuyLimit is used to place a buy order in a specific market. Use buylimit
// to place limit orders. Make sure you have the proper permissions set on your
// API keys for this call to work.
// "Currency" ie "btc-ltc"
// "Quantity" is the amount to purchase
// "Rate" is the rate at which to purchase
func (b *Bittrex) PlaceBuyLimit(currencyPair string, quantity, rate float64) (UUID, error) {
	var id UUID
	values := url.Values{}
	values.Set("market", currencyPair)
	values.Set("quantity", strconv.FormatFloat(quantity, 'E', -1, 64))
	values.Set("rate", strconv.FormatFloat(rate, 'E', -1, 64))
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPIBuyLimit)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &id); err != nil {
		return id, err
	}

	if !id.Success {
		return id, errors.New(id.Message)
	}
	return id, nil
}

// PlaceSellLimit is used to place a sell order in a specific market. Use
// selllimit to place limit orders. Make sure you have the proper permissions
// set on your API keys for this call to work.
// "Currency" ie "btc-ltc"
// "Quantity" is the amount to purchase
// "Rate" is the rate at which to purchase
func (b *Bittrex) PlaceSellLimit(currencyPair string, quantity, rate float64) (UUID, error) {
	var id UUID
	values := url.Values{}
	values.Set("market", currencyPair)
	values.Set("quantity", strconv.FormatFloat(quantity, 'E', -1, 64))
	values.Set("rate", strconv.FormatFloat(rate, 'E', -1, 64))
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPISellLimit)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &id); err != nil {
		return id, err
	}

	if !id.Success {
		return id, errors.New(id.Message)
	}
	return id, nil
}

// GetOpenOrders returns all orders that you currently have opened.
// A specific market can be requested for example "btc-ltc"
func (b *Bittrex) GetOpenOrders(currencyPair string) (Order, error) {
	var orders Order
	values := url.Values{}
	if !(currencyPair == "" || currencyPair == " ") {
		values.Set("market", currencyPair)
	}
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPIGetOpenOrders)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &orders); err != nil {
		return orders, err
	}

	if !orders.Success {
		return orders, errors.New(orders.Message)
	}
	return orders, nil
}

// CancelExistingOrder is used to cancel a buy or sell order.
func (b *Bittrex) CancelExistingOrder(uuid string) (Balances, error) {
	var balances Balances
	values := url.Values{}
	values.Set("uuid", uuid)
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPICancel)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &balances); err != nil {
		return balances, err
	}

	if !balances.Success {
		return balances, errors.New(balances.Message)
	}
	return balances, nil
}

// GetAccountBalances is used to retrieve all balances from your account
func (b *Bittrex) GetAccountBalances() (Balances, error) {
	var balances Balances
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPIGetBalances)

	if err := b.SendAuthenticatedHTTPRequest(path, url.Values{}, &balances); err != nil {
		return balances, err
	}

	if !balances.Success {
		return balances, errors.New(balances.Message)
	}
	return balances, nil
}

// GetAccountBalanceByCurrency is used to retrieve the balance from your account
// for a specific currency. ie. "btc" or "ltc"
func (b *Bittrex) GetAccountBalanceByCurrency(currency string) (Balance, error) {
	var balance Balance
	values := url.Values{}
	values.Set("currency", currency)
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPIGetBalance)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &balance); err != nil {
		return balance, err
	}

	if !balance.Success {
		return balance, errors.New(balance.Message)
	}
	return balance, nil
}

// GetCryptoDepositAddress is used to retrieve or generate an address for a specific
// currency. If one does not exist, the call will fail and return
// ADDRESS_GENERATING until one is available.
func (b *Bittrex) GetCryptoDepositAddress(currency string) (DepositAddress, error) {
	var address DepositAddress
	values := url.Values{}
	values.Set("currency", currency)
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPIGetDepositAddress)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &address); err != nil {
		return address, err
	}

	if !address.Success {
		return address, errors.New(address.Message)
	}
	return address, nil
}

// Withdraw is used to withdraw funds from your account.
// note: Please account for transaction fee.
func (b *Bittrex) Withdraw(currency, paymentID, address string, quantity float64) (UUID, error) {
	var id UUID
	values := url.Values{}
	values.Set("currency", currency)
	values.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	values.Set("address", address)
	if len(paymentID) > 0 {
		values.Set("paymentid", paymentID)
	}

	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPIWithdraw)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &id); err != nil {
		return id, err
	}

	if !id.Success {
		return id, errors.New(id.Message)
	}
	return id, nil
}

// GetOrder is used to retrieve a single order by UUID.
func (b *Bittrex) GetOrder(uuid string) (Order, error) {
	var order Order
	values := url.Values{}
	values.Set("uuid", uuid)
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPIGetOrder)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &order); err != nil {
		return order, err
	}

	if !order.Success {
		return order, errors.New(order.Message)
	}
	return order, nil
}

// GetOrderHistoryForCurrency is used to retrieve your order history. If currencyPair
// omitted it will return the entire order History.
func (b *Bittrex) GetOrderHistoryForCurrency(currencyPair string) (Order, error) {
	var orders Order
	values := url.Values{}

	if !(currencyPair == "" || currencyPair == " ") {
		values.Set("market", currencyPair)
	}
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPIGetOrderHistory)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &orders); err != nil {
		return orders, err
	}

	if !orders.Success {
		return orders, errors.New(orders.Message)
	}
	return orders, nil
}

// GetWithdrawalHistory is used to retrieve your withdrawal history. If currency
// omitted it will return the entire history
func (b *Bittrex) GetWithdrawalHistory(currency string) (WithdrawalHistory, error) {
	var history WithdrawalHistory
	values := url.Values{}

	if !(currency == "" || currency == " ") {
		values.Set("currency", currency)
	}
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPIGetWithdrawalHistory)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &history); err != nil {
		return history, err
	}

	if !history.Success {
		return history, errors.New(history.Message)
	}
	return history, nil
}

// GetDepositHistory is used to retrieve your deposit history. If currency is
// is omitted it will return the entire deposit history
func (b *Bittrex) GetDepositHistory(currency string) (DepositHistory, error) {
	var history DepositHistory
	values := url.Values{}

	if !(currency == "" || currency == " ") {
		values.Set("currency", currency)
	}
	path := fmt.Sprintf("%s/%s", b.API.Endpoints.URL, bittrexAPIGetDepositHistory)

	if err := b.SendAuthenticatedHTTPRequest(path, values, &history); err != nil {
		return history, err
	}

	if !history.Success {
		return history, errors.New(history.Message)
	}
	return history, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *Bittrex) SendHTTPRequest(path string, result interface{}) error {
	return b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          path,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}

// SendAuthenticatedHTTPRequest sends an authenticated http request to a desired
// path
func (b *Bittrex) SendAuthenticatedHTTPRequest(path string, values url.Values, result interface{}) (err error) {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}

	n := b.Requester.GetNonce(true).String()

	values.Set("apikey", b.API.Credentials.Key)
	values.Set("nonce", n)
	rawQuery := path + "?" + values.Encode()
	hmac := crypto.GetHMAC(
		crypto.HashSHA512, []byte(rawQuery), []byte(b.API.Credentials.Secret),
	)
	headers := make(map[string]string)
	headers["apisign"] = crypto.HexEncodeToString(hmac)

	return b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          rawQuery,
		Headers:       headers,
		Result:        result,
		AuthRequest:   true,
		NonceEnabled:  true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Bittrex) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	var err error

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	case exchange.CryptocurrencyWithdrawalFee:
		fee, err = b.GetWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.OfflineTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, err
}

// GetWithdrawalFee returns the fee for withdrawing from the exchange
func (b *Bittrex) GetWithdrawalFee(c currency.Code) (float64, error) {
	var fee float64

	currencies, err := b.GetCurrencies()
	if err != nil {
		return 0, err
	}
	for _, result := range currencies.Result {
		if result.Currency == c.String() {
			fee = result.TxFee
		}
	}
	return fee, nil
}

// calculateTradingFee returns the fee for trading any currency on Bittrex
func calculateTradingFee(price, amount float64) float64 {
	return 0.0025 * price * amount
}

func parseTime(t string) (time.Time, error) {
	return time.Parse(bittrexTimeLayout, t)
}
