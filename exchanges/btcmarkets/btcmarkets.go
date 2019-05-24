package btcmarkets

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	btcMarketsAPIURL            = "https://api.btcmarkets.net"
	btcMarketsAPIVersion        = "0"
	btcMarketsAccountBalance    = "/account/balance"
	btcMarketsTradingFee        = "/account/%s/%s/tradingfee"
	btcMarketsOrderCreate       = "/order/create"
	btcMarketsOrderCancel       = "/order/cancel"
	btcMarketsOrderHistory      = "/order/history"
	btcMarketsOrderOpen         = "/order/open"
	btcMarketsOrderTradeHistory = "/order/trade/history"
	btcMarketsOrderDetail       = "/order/detail"
	btcMarketsWithdrawCrypto    = "/fundtransfer/withdrawCrypto"
	btcMarketsWithdrawAud       = "/fundtransfer/withdrawEFT"

	// Status Values
	orderStatusNew                = "New"
	orderStatusPlaced             = "Placed"
	orderStatusFailed             = "Failed"
	orderStatusError              = "Error"
	orderStatusCancelled          = "Cancelled"
	orderStatusPartiallyCancelled = "Partially Cancelled"
	orderStatusFullyMatched       = "Fully Matched"
	orderStatusPartiallyMatched   = "Partially Matched"

	btcmarketsAuthLimit   = 10
	btcmarketsUnauthLimit = 25
)

// BTCMarkets is the overarching type across the BTCMarkets package
type BTCMarkets struct {
	exchange.Base
	Ticker map[string]Ticker
}

// SetDefaults sets basic defaults
func (b *BTCMarkets) SetDefaults() {
	b.Name = "BTC Markets"
	b.Enabled = false
	b.Fee = 0.85
	b.Verbose = false
	b.RESTPollingDelay = 10
	b.Ticker = make(map[string]Ticker)
	b.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.AutoWithdrawFiat
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.RequestCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Delimiter = "-"
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.AssetTypes = []string{ticker.Spot}
	b.SupportsAutoPairUpdating = true
	b.SupportsRESTTickerBatching = false
	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second*10, btcmarketsAuthLimit),
		request.NewRateLimit(time.Second*10, btcmarketsUnauthLimit),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	b.APIUrlDefault = btcMarketsAPIURL
	b.APIUrl = b.APIUrlDefault
	b.Websocket = wshandler.New()
}

// Setup takes in an exchange configuration and sets all parameters
func (b *BTCMarkets) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, "", true)
		b.SetHTTPClientTimeout(exch.HTTPTimeout)
		b.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.HTTPDebugging = exch.HTTPDebugging
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

// GetMarkets returns the BTCMarkets instruments
func (b *BTCMarkets) GetMarkets() ([]Market, error) {
	type marketsResp struct {
		Response
		Markets []Market `json:"markets"`
	}

	var resp marketsResp
	path := fmt.Sprintf("%s/v2/market/active", b.APIUrl)

	err := b.SendHTTPRequest(path, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("%s unable to get markets: %s", b.Name, resp.ErrorMessage)
	}

	return resp.Markets, nil
}

// GetTicker returns a ticker
// symbol - example "btc" or "ltc"
func (b *BTCMarkets) GetTicker(firstPair, secondPair string) (Ticker, error) {
	tick := Ticker{}
	path := fmt.Sprintf("%s/market/%s/%s/tick",
		b.APIUrl,
		common.StringToUpper(firstPair),
		common.StringToUpper(secondPair))

	return tick, b.SendHTTPRequest(path, &tick)
}

// GetOrderbook returns current orderbook
// symbol - example "btc" or "ltc"
func (b *BTCMarkets) GetOrderbook(firstPair, secondPair string) (Orderbook, error) {
	orderbook := Orderbook{}
	path := fmt.Sprintf("%s/market/%s/%s/orderbook",
		b.APIUrl,
		common.StringToUpper(firstPair),
		common.StringToUpper(secondPair))

	return orderbook, b.SendHTTPRequest(path, &orderbook)
}

// GetTrades returns executed trades on the exchange
// symbol - example "btc" or "ltc"
// values - optional paramater "since" example values.Set(since, "59868345231")
func (b *BTCMarkets) GetTrades(firstPair, secondPair string, values url.Values) ([]Trade, error) {
	var trades []Trade
	path := common.EncodeURLValues(fmt.Sprintf("%s/market/%s/%s/trades",
		b.APIUrl, common.StringToUpper(firstPair),
		common.StringToUpper(secondPair)), values)

	return trades, b.SendHTTPRequest(path, &trades)
}

// NewOrder requests a new order and returns an ID
// currency - example "AUD"
// instrument - example "BTC"
// price - example 13000000000 (i.e x 100000000)
// amount - example 100000000 (i.e x 100000000)
// orderside - example "Bid" or "Ask"
// orderType - example "limit"
// clientReq - example "abc-cdf-1000"
func (b *BTCMarkets) NewOrder(currency, instrument string, price, amount float64, orderSide, orderType, clientReq string) (int64, error) {
	newPrice := int64(price * float64(common.SatoshisPerBTC))
	newVolume := int64(amount * float64(common.SatoshisPerBTC))

	order := OrderToGo{
		Currency:        common.StringToUpper(currency),
		Instrument:      common.StringToUpper(instrument),
		Price:           newPrice,
		Volume:          newVolume,
		OrderSide:       orderSide,
		OrderType:       orderType,
		ClientRequestID: clientReq,
	}

	resp := Response{}

	err := b.SendAuthenticatedRequest(http.MethodPost, btcMarketsOrderCreate, order, &resp)
	if err != nil {
		return 0, err
	}

	if !resp.Success {
		return 0, fmt.Errorf("%s Unable to place order. Error message: %s", b.GetName(), resp.ErrorMessage)
	}
	return int64(resp.ID), nil
}

// CancelExistingOrder cancels an order by its ID
// orderID - id for order example "1337"
func (b *BTCMarkets) CancelExistingOrder(orderID []int64) ([]ResponseDetails, error) {
	resp := Response{}
	type CancelOrder struct {
		OrderIDs []int64 `json:"orderIds"`
	}
	orders := CancelOrder{}
	orders.OrderIDs = append(orders.OrderIDs, orderID...)

	err := b.SendAuthenticatedRequest(http.MethodPost, btcMarketsOrderCancel, orders, &resp)
	if err != nil {
		return resp.Responses, err
	}

	if !resp.Success {
		return resp.Responses, fmt.Errorf("%s Unable to cancel order. Error message: %s", b.GetName(), resp.ErrorMessage)
	}

	return resp.Responses, nil
}

// GetOrders returns current order information on the exchange
// currency - example "AUD"
// instrument - example "BTC"
// limit - example "10"
// since - since a time example "33434568724"
// historic - if false just normal Orders open
func (b *BTCMarkets) GetOrders(currency, instrument string, limit, since int64, historic bool) ([]Order, error) {
	req := make(map[string]interface{})

	if currency != "" {
		req["currency"] = common.StringToUpper(currency)
	}
	if instrument != "" {
		req["instrument"] = common.StringToUpper(instrument)
	}
	if limit != 0 {
		req["limit"] = limit
	}
	if since != 0 {
		req["since"] = since
	}

	path := btcMarketsOrderOpen
	if historic {
		path = btcMarketsOrderHistory
	}

	resp := Response{}

	err := b.SendAuthenticatedRequest(http.MethodPost, path, req, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.ErrorMessage)
	}

	for i := range resp.Orders {
		resp.Orders[i].Price /= common.SatoshisPerBTC
		resp.Orders[i].OpenVolume /= common.SatoshisPerBTC
		resp.Orders[i].Volume /= common.SatoshisPerBTC

		for j := range resp.Orders[i].Trades {
			resp.Orders[i].Trades[j].Fee /= common.SatoshisPerBTC
			resp.Orders[i].Trades[j].Price /= common.SatoshisPerBTC
			resp.Orders[i].Trades[j].Volume /= common.SatoshisPerBTC
		}
	}

	return resp.Orders, nil
}

// GetOpenOrders returns all open orders
func (b *BTCMarkets) GetOpenOrders() ([]Order, error) {
	type marketsResp struct {
		Response
	}
	req := make(map[string]interface{})
	var resp marketsResp
	path := fmt.Sprintf("/v2/order/open")

	err := b.SendAuthenticatedRequest(http.MethodGet, path, req, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.ErrorMessage)
	}

	for i := range resp.Orders {
		resp.Orders[i].Price /= common.SatoshisPerBTC
		resp.Orders[i].OpenVolume /= common.SatoshisPerBTC
		resp.Orders[i].Volume /= common.SatoshisPerBTC

		for j := range resp.Orders[i].Trades {
			resp.Orders[i].Trades[j].Fee /= common.SatoshisPerBTC
			resp.Orders[i].Trades[j].Price /= common.SatoshisPerBTC
			resp.Orders[i].Trades[j].Volume /= common.SatoshisPerBTC
		}
	}

	return resp.Orders, nil
}

// GetOrderDetail returns order information an a specific order
// orderID - example "1337"
func (b *BTCMarkets) GetOrderDetail(orderID []int64) ([]Order, error) {
	type OrderDetail struct {
		OrderIDs []int64 `json:"orderIds"`
	}
	orders := OrderDetail{}
	orders.OrderIDs = append(orders.OrderIDs, orderID...)

	resp := Response{}

	err := b.SendAuthenticatedRequest(http.MethodPost, btcMarketsOrderDetail, orders, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.ErrorMessage)
	}

	for i := range resp.Orders {
		resp.Orders[i].Price /= common.SatoshisPerBTC
		resp.Orders[i].OpenVolume /= common.SatoshisPerBTC
		resp.Orders[i].Volume /= common.SatoshisPerBTC

		for x := range resp.Orders[i].Trades {
			resp.Orders[i].Trades[x].Fee /= common.SatoshisPerBTC
			resp.Orders[i].Trades[x].Price /= common.SatoshisPerBTC
			resp.Orders[i].Trades[x].Volume /= common.SatoshisPerBTC
		}
	}
	return resp.Orders, nil
}

// GetAccountBalance returns the full account balance
func (b *BTCMarkets) GetAccountBalance() ([]AccountBalance, error) {
	var balance []AccountBalance

	err := b.SendAuthenticatedRequest(http.MethodGet, btcMarketsAccountBalance, nil, &balance)
	if err != nil {
		return nil, err
	}

	// All values are returned in Satoshis, even for fiat currencies.
	for i := range balance {
		balance[i].Balance /= common.SatoshisPerBTC
		balance[i].PendingFunds /= common.SatoshisPerBTC
	}
	return balance, nil
}

// GetTradingFee returns the account's trading fee for a currency pair
func (b *BTCMarkets) GetTradingFee(base, quote currency.Code) (TradingFee, error) {
	var tradingFee TradingFee
	path := fmt.Sprintf(btcMarketsTradingFee, base, quote)
	return tradingFee, b.SendAuthenticatedRequest(http.MethodGet, path, nil, &tradingFee)
}

// WithdrawCrypto withdraws cryptocurrency into a designated address
func (b *BTCMarkets) WithdrawCrypto(amount float64, currency, address string) (string, error) {
	newAmount := int64(amount * float64(common.SatoshisPerBTC))

	req := WithdrawRequestCrypto{
		Amount:   newAmount,
		Currency: common.StringToUpper(currency),
		Address:  address,
	}

	resp := Response{}
	err := b.SendAuthenticatedRequest(http.MethodPost,
		btcMarketsWithdrawCrypto,
		req,
		&resp)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", errors.New(resp.ErrorMessage)
	}

	return resp.Status, nil
}

// WithdrawAUD withdraws AUD into a designated bank address
// Does not return a TxID!
func (b *BTCMarkets) WithdrawAUD(accountName, accountNumber, bankName, bsbNumber string, amount float64) (string, error) {
	newAmount := int64(amount * float64(common.SatoshisPerBTC))

	req := WithdrawRequestAUD{
		AccountName:   accountName,
		AccountNumber: accountNumber,
		BankName:      bankName,
		BSBNumber:     bsbNumber,
		Amount:        newAmount,
		Currency:      "AUD",
	}

	resp := Response{}
	err := b.SendAuthenticatedRequest(http.MethodPost, btcMarketsWithdrawAud,
		req,
		&resp)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", errors.New(resp.ErrorMessage)
	}

	return resp.Status, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *BTCMarkets) SendHTTPRequest(path string, result interface{}) error {
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

// SendAuthenticatedRequest sends an authenticated HTTP request
func (b *BTCMarkets) SendAuthenticatedRequest(reqType, path string, data, result interface{}) (err error) {
	if !b.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			b.Name)
	}

	n := b.Requester.GetNonce(true).String()[0:13]

	var req string
	payload := []byte("")

	if data != nil {
		payload, err = common.JSONEncode(data)
		if err != nil {
			return err
		}
		req = path + "\n" + n + "\n" + string(payload)
	} else {
		req = path + "\n" + n + "\n"
	}

	hmac := common.GetHMAC(common.HashSHA512,
		[]byte(req), []byte(b.APISecret))

	if b.Verbose {
		log.Debugf("Sending %s request to URL %s with params %s\n",
			reqType,
			b.APIUrl+path,
			req)
	}

	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Accept-Charset"] = "UTF-8"
	headers["Content-Type"] = "application/json"
	headers["apikey"] = b.APIKey
	headers["timestamp"] = n
	headers["signature"] = common.Base64Encode(hmac)

	return b.SendPayload(reqType,
		b.APIUrl+path,
		headers,
		bytes.NewBuffer(payload),
		result,
		true,
		true,
		b.Verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
}

// GetFee returns an estimate of fee based on type of transaction
func (b *BTCMarkets) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		tradingFee, err := b.GetTradingFee(feeBuilder.Pair.Base,
			feeBuilder.Pair.Quote)
		if err != nil {
			return 0, err
		}

		fee = calculateTradingFee(tradingFee,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount)

	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency)
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
	return 0.0085 * price * amount
}

func calculateTradingFee(tradingFee TradingFee, purchasePrice, amount float64) (fee float64) {
	fee = tradingFee.TradingFeeRate / 100000000
	return fee * amount * purchasePrice
}

func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

func getInternationalBankWithdrawalFee(c currency.Code) float64 {
	var fee float64

	if c == currency.AUD {
		fee = 0
	}
	return fee
}
