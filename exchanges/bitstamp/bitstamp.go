package bitstamp

import (
	"bytes"
	"context"
	"encoding/json"
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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bitstampAPIURL                = "https://www.bitstamp.net/api"
	bitstampAPIVersion            = "2"
	bitstampAPITicker             = "ticker"
	bitstampAPITickerHourly       = "ticker_hour"
	bitstampAPIOrderbook          = "order_book"
	bitstampAPITransactions       = "transactions"
	bitstampAPIEURUSD             = "eur_usd"
	bitstampAPIBalance            = "balance"
	bitstampAPIUserTransactions   = "user_transactions"
	bitstampAPIOpenOrders         = "open_orders"
	bitstampAPIOrderStatus        = "order_status"
	bitstampAPICancelOrder        = "cancel_order"
	bitstampAPICancelAllOrders    = "cancel_all_orders"
	bitstampAPIMarket             = "market"
	bitstampAPIWithdrawalRequests = "withdrawal_requests"
	bitstampAPIOpenWithdrawal     = "withdrawal/open"
	bitstampAPIBitcoinWithdrawal  = "bitcoin_withdrawal"
	bitstampAPILTCWithdrawal      = "ltc_withdrawal"
	bitstampAPIETHWithdrawal      = "eth_withdrawal"
	bitstampAPIBCHWithdrawal      = "bch_withdrawal"
	bitstampAPIBitcoinDeposit     = "bitcoin_deposit_address"
	bitstampAPILitecoinDeposit    = "ltc_address"
	bitstampAPIEthereumDeposit    = "eth_address"
	bitstampAPIBitcoinCashDeposit = "bch_address"
	bitstampAPIUnconfirmedBitcoin = "unconfirmed_btc"
	bitstampAPITransferToMain     = "transfer-to-main"
	bitstampAPITransferFromMain   = "transfer-from-main"
	bitstampAPIXrpWithdrawal      = "xrp_withdrawal"
	bitstampAPIXrpDeposit         = "xrp_address"
	bitstampAPIReturnType         = "string"
	bitstampAPITradingPairsInfo   = "trading-pairs-info"
	bitstampOHLC                  = "ohlc"

	bitstampRateInterval = time.Minute * 10
	bitstampRequestRate  = 8000
	bitstampTimeLayout   = "2006-1-2 15:04:05"
)

// Bitstamp is the overarching type across the bitstamp package
type Bitstamp struct {
	exchange.Base
}

// GetTicker returns ticker information
func (b *Bitstamp) GetTicker(ctx context.Context, currency string, hourly bool) (*Ticker, error) {
	response := Ticker{}
	tickerEndpoint := bitstampAPITicker

	if hourly {
		tickerEndpoint = bitstampAPITickerHourly
	}
	path := "/v" + bitstampAPIVersion + "/" + tickerEndpoint + "/" + strings.ToLower(currency) + "/"
	return &response, b.SendHTTPRequest(ctx, exchange.RestSpot, path, &response)
}

// GetOrderbook Returns a JSON dictionary with "bids" and "asks". Each is a list
// of open orders and each order is represented as a list holding the price and
// the amount.
func (b *Bitstamp) GetOrderbook(ctx context.Context, currency string) (Orderbook, error) {
	type response struct {
		Timestamp int64      `json:"timestamp,string"`
		Bids      [][]string `json:"bids"`
		Asks      [][]string `json:"asks"`
	}
	resp := response{}
	path := "/v" + bitstampAPIVersion + "/" + bitstampAPIOrderbook + "/" + strings.ToLower(currency) + "/"
	err := b.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
	if err != nil {
		return Orderbook{}, err
	}

	orderbook := Orderbook{}
	orderbook.Timestamp = resp.Timestamp

	for _, x := range resp.Bids {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Error(log.ExchangeSys, err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Error(log.ExchangeSys, err)
			continue
		}
		orderbook.Bids = append(orderbook.Bids, OrderbookBase{price, amount})
	}

	for _, x := range resp.Asks {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Error(log.ExchangeSys, err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Error(log.ExchangeSys, err)
			continue
		}
		orderbook.Asks = append(orderbook.Asks, OrderbookBase{price, amount})
	}

	return orderbook, nil
}

// GetTradingPairs returns a list of trading pairs which Bitstamp
// currently supports
func (b *Bitstamp) GetTradingPairs(ctx context.Context) ([]TradingPair, error) {
	var result []TradingPair
	path := "/v" + bitstampAPIVersion + "/" + bitstampAPITradingPairsInfo
	return result, b.SendHTTPRequest(ctx, exchange.RestSpot, path, &result)
}

// GetTransactions returns transaction information
// value paramater ["time"] = "minute", "hour", "day" will collate your
// response into time intervals.
func (b *Bitstamp) GetTransactions(ctx context.Context, currencyPair, timePeriod string) ([]Transactions, error) {
	var transactions []Transactions
	requestURL := "/v" + bitstampAPIVersion + "/" + bitstampAPITransactions + "/" + strings.ToLower(currencyPair) + "/"
	if timePeriod != "" {
		requestURL += "?time=" + url.QueryEscape(timePeriod)
	}
	return transactions, b.SendHTTPRequest(ctx, exchange.RestSpot, requestURL, &transactions)
}

// GetEURUSDConversionRate returns the conversion rate between Euro and USD
func (b *Bitstamp) GetEURUSDConversionRate(ctx context.Context) (EURUSDConversionRate, error) {
	rate := EURUSDConversionRate{}
	path := "/" + bitstampAPIEURUSD
	return rate, b.SendHTTPRequest(ctx, exchange.RestSpot, path, &rate)
}

// GetBalance returns full balance of currency held on the exchange
// This is done to keep items in order because maps have randomized access.
func (b *Bitstamp) GetBalance(ctx context.Context) (Balances, error) {
	var balance json.RawMessage
	err := b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIBalance, true, nil, &balance)
	if err != nil {
		return nil, err
	}

	data := string(balance)
	data = strings.Trim(data, "{")
	data = strings.Trim(data, "}")
	run := strings.Split(data, ",")

	var balances = map[currency.Code]*Balance{}
	var ID currency.Code
	for x := 0; x < len(run); x++ {
		run[x] = strings.ReplaceAll(run[x], "\"", "")
		run[x] = strings.ReplaceAll(run[x], " ", "")

		element := strings.Split(run[x], ":")
		serviceID := strings.Split(element[0], "_")
		var quote currency.Code
		if ID.IsEmpty() {
			ID = currency.NewCode(serviceID[0])
		} else {

			translations, ok := stableTranslation[ID.Item]
			if !ok && !strings.Contains(serviceID[0], ID.String()) {
				ID = currency.NewCode(serviceID[0])
			} else {
				for y := range translations {
					if strings.Contains(serviceID[0], translations[y].String()) {
						ID = currency.NewCode(serviceID[0])
						break
					}
				}
				for y := range translations {
					if strings.Contains(serviceID[0], translations[y].String()) {
						quote = translations[y]
						break
					}
				}
			}
			if quote.IsEmpty() {
				cleaned := strings.Replace(serviceID[0], ID.String(), "", 1)
				quote = currency.NewCode(cleaned)
			}
		}

		balance, ok := balances[ID]
		if !ok {
			balance = &Balance{TransactionFees: make(map[currency.Code]float64)}
			balances[ID] = balance
		}

		value, err := strconv.ParseFloat(element[1], 64)
		if err != nil {
			return nil, err
		}

		switch serviceID[1] {
		case "available":
			balance.Available = value
		case "balance":
			balance.Balance = value
		case "reserved":
			balance.Reserved = value
		case "withdrawal":
			balance.WithdrawalFee = value
		case "fee":
			balance.TransactionFees[quote] = value
		}
	}
	return balances, nil
}

var stableTranslation = map[*currency.Item][]currency.Code{
	currency.USD.Item: {currency.USDT.Lower(), currency.USDC.Lower()},
	currency.EUR.Item: {currency.EURT.Lower()},
}

// GetUserTransactions returns an array of transactions
func (b *Bitstamp) GetUserTransactions(ctx context.Context, currencyPair string) ([]UserTransactions, error) {
	type Response struct {
		Date          string      `json:"datetime"`
		TransactionID int64       `json:"id"`
		Type          int         `json:"type,string"`
		USD           interface{} `json:"usd"`
		EUR           interface{} `json:"eur"`
		XRP           interface{} `json:"xrp"`
		BTC           interface{} `json:"btc"`
		BTCUSD        interface{} `json:"btc_usd"`
		Fee           float64     `json:"fee,string"`
		OrderID       int64       `json:"order_id"`
	}
	var response []Response

	if currencyPair == "" {
		if err := b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIUserTransactions,
			true,
			url.Values{},
			&response); err != nil {
			return nil, err
		}
	} else {
		if err := b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIUserTransactions+"/"+currencyPair,
			true,
			url.Values{},
			&response); err != nil {
			return nil, err
		}
	}

	processNumber := func(i interface{}) float64 {
		switch t := i.(type) {
		case float64:
			return t
		case string:
			amt, _ := strconv.ParseFloat(t, 64)
			return amt
		default:
			return 0
		}
	}

	var transactions []UserTransactions
	for x := range response {
		tx := UserTransactions{}
		tx.Date = response[x].Date
		tx.TransactionID = response[x].TransactionID
		tx.Type = response[x].Type
		tx.EUR = processNumber(response[x].EUR)
		tx.XRP = processNumber(response[x].XRP)
		tx.USD = processNumber(response[x].USD)
		tx.BTC = processNumber(response[x].BTC)
		tx.BTCUSD = processNumber(response[x].BTCUSD)
		tx.Fee = response[x].Fee
		tx.OrderID = response[x].OrderID
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// GetOpenOrders returns all open orders on the exchange
func (b *Bitstamp) GetOpenOrders(ctx context.Context, currencyPair string) ([]Order, error) {
	var resp []Order
	path := bitstampAPIOpenOrders + "/" + strings.ToLower(currencyPair)
	return resp, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, path, true, nil, &resp)
}

// GetOrderStatus returns an the status of an order by its ID
func (b *Bitstamp) GetOrderStatus(ctx context.Context, orderID int64) (OrderStatus, error) {
	resp := OrderStatus{}
	req := url.Values{}
	req.Add("id", strconv.FormatInt(orderID, 10))

	return resp,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIOrderStatus, false, req, &resp)
}

// CancelExistingOrder cancels order by ID
func (b *Bitstamp) CancelExistingOrder(ctx context.Context, orderID int64) (CancelOrder, error) {
	var req = url.Values{}
	req.Add("id", strconv.FormatInt(orderID, 10))

	var result CancelOrder
	err := b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPICancelOrder, true, req, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

// CancelAllExistingOrders cancels all open orders on the exchange
func (b *Bitstamp) CancelAllExistingOrders(ctx context.Context) (bool, error) {
	result := false

	return result,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPICancelAllOrders, false, nil, &result)
}

// PlaceOrder places an order on the exchange.
func (b *Bitstamp) PlaceOrder(ctx context.Context, currencyPair string, price, amount float64, buy, market bool) (Order, error) {
	var req = url.Values{}
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
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, path, true, req, &response)
}

// GetWithdrawalRequests returns withdrawal requests for the account
// timedelta - positive integer with max value 50000000 which returns requests
// from number of seconds ago to now.
func (b *Bitstamp) GetWithdrawalRequests(ctx context.Context, timedelta int64) ([]WithdrawalRequests, error) {
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
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIWithdrawalRequests, false, value, &resp)
}

// CryptoWithdrawal withdraws a cryptocurrency into a supplied wallet, returns ID
// amount - The amount you want withdrawn
// address - The wallet address of the cryptocurrency
// symbol - the type of crypto ie "ltc", "btc", "eth"
// destTag - only for XRP  default to ""
// instant - only for bitcoins
func (b *Bitstamp) CryptoWithdrawal(ctx context.Context, amount float64, address, symbol, destTag string, instant bool) (CryptoWithdrawalResponse, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)
	resp := CryptoWithdrawalResponse{}
	var endpoint string

	switch strings.ToLower(symbol) {
	case currency.BTC.Lower().String():
		if instant {
			req.Add("instant", "1")
		} else {
			req.Add("instant", "0")
		}
		endpoint = bitstampAPIBitcoinWithdrawal
	case currency.LTC.Lower().String():
		endpoint = bitstampAPILTCWithdrawal
	case currency.ETH.Lower().String():
		endpoint = bitstampAPIETHWithdrawal
	case currency.XRP.Lower().String():
		if destTag != "" {
			req.Add("destination_tag", destTag)
		}
		endpoint = bitstampAPIXrpWithdrawal
	case currency.BCH.Lower().String():
		endpoint = bitstampAPIBCHWithdrawal
	default:
		return resp, errors.New("incorrect symbol")
	}

	return resp, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, endpoint, false, req, &resp)
}

// OpenBankWithdrawal Opens a bank withdrawal request (SEPA or international)
func (b *Bitstamp) OpenBankWithdrawal(ctx context.Context, amount float64, currency,
	name, iban, bic, address, postalCode, city, country,
	comment, withdrawalType string) (FIATWithdrawalResponse, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("account_currency", currency)
	req.Add("name", name)
	req.Add("iban", iban)
	req.Add("bic", bic)
	req.Add("address", address)
	req.Add("postal_code", postalCode)
	req.Add("city", city)
	req.Add("country", country)
	req.Add("type", withdrawalType)
	req.Add("comment", comment)

	resp := FIATWithdrawalResponse{}
	return resp, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIOpenWithdrawal, true, req, &resp)
}

// OpenInternationalBankWithdrawal Opens a bank withdrawal request (international)
func (b *Bitstamp) OpenInternationalBankWithdrawal(ctx context.Context, amount float64, currency,
	name, iban, bic, address, postalCode, city, country,
	bankName, bankAddress, bankPostCode, bankCity, bankCountry, internationalCurrency,
	comment, withdrawalType string) (FIATWithdrawalResponse, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("account_currency", currency)
	req.Add("name", name)
	req.Add("iban", iban)
	req.Add("bic", bic)
	req.Add("address", address)
	req.Add("postal_code", postalCode)
	req.Add("city", city)
	req.Add("country", country)
	req.Add("type", withdrawalType)
	req.Add("comment", comment)
	req.Add("currency", internationalCurrency)
	req.Add("bank_name", bankName)
	req.Add("bank_address", bankAddress)
	req.Add("bank_postal_code", bankPostCode)
	req.Add("bank_city", bankCity)
	req.Add("bank_country", bankCountry)

	resp := FIATWithdrawalResponse{}
	return resp, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIOpenWithdrawal, true, req, &resp)
}

// GetCryptoDepositAddress returns a depositing address by crypto
// crypto - example "btc", "ltc", "eth", "xrp" or "bch"
func (b *Bitstamp) GetCryptoDepositAddress(ctx context.Context, crypto currency.Code) (string, error) {
	var resp string
	v2Resp := struct {
		Address string `json:"address"`
	}{}

	switch crypto {
	case currency.BTC:
		return resp,
			b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIBitcoinDeposit, false, nil, &resp)

	case currency.LTC:
		return v2Resp.Address,
			b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPILitecoinDeposit, true, nil, &v2Resp)

	case currency.ETH:
		return v2Resp.Address,
			b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIEthereumDeposit, true, nil, &v2Resp)

	case currency.XRP:
		return v2Resp.Address,
			b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIXrpDeposit, true, nil, &v2Resp)

	case currency.BCH:
		return v2Resp.Address,
			b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIBitcoinCashDeposit, true, nil, &v2Resp)

	default:
		return resp, fmt.Errorf("unsupported cryptocurrency string %s", crypto)
	}
}

// GetUnconfirmedBitcoinDeposits returns unconfirmed transactions
func (b *Bitstamp) GetUnconfirmedBitcoinDeposits(ctx context.Context) ([]UnconfirmedBTCTransactions, error) {
	var response []UnconfirmedBTCTransactions

	return response,
		b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIUnconfirmedBitcoin, false, nil, &response)
}

// OHLC returns OHLCV data for step (interval)
func (b *Bitstamp) OHLC(ctx context.Context, currency string, start, end time.Time, step, limit string) (resp OHLCResponse, err error) {
	var v = url.Values{}
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
	return resp, b.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v"+bitstampAPIVersion+"/"+bitstampOHLC+"/"+currency, v), &resp)
}

// TransferAccountBalance transfers funds from either a main or sub account
// amount - to transfers
// currency - which currency to transfer
// subaccount - name of account
// toMain - bool either to or from account
func (b *Bitstamp) TransferAccountBalance(ctx context.Context, amount float64, currency, subAccount string, toMain bool) error {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("currency", currency)

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

	var resp interface{}

	return b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, path, true, req, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *Bitstamp) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result interface{}) error {
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
	})
}

// SendAuthenticatedHTTPRequest sends an authenticated request
func (b *Bitstamp) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, path string, v2 bool, values url.Values, result interface{}) error {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", b.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	endpoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	if values == nil {
		values = url.Values{}
	}

	interim := json.RawMessage{}
	err = b.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		n := b.Requester.GetNonce(true).String()

		values.Set("key", b.API.Credentials.Key)
		values.Set("nonce", n)

		var hmac []byte
		hmac, err = crypto.GetHMAC(crypto.HashSHA256,
			[]byte(n+b.API.Credentials.ClientID+b.API.Credentials.Key),
			[]byte(b.API.Credentials.Secret))
		if err != nil {
			return nil, err
		}

		values.Set("signature", strings.ToUpper(crypto.HexEncodeToString(hmac)))

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
			Method:        http.MethodPost,
			Path:          fullPath,
			Headers:       headers,
			Body:          readerValues,
			Result:        &interim,
			AuthRequest:   true,
			NonceEnabled:  true,
			Verbose:       b.Verbose,
			HTTPDebugging: b.HTTPDebugging,
			HTTPRecording: b.HTTPRecording,
		}, nil
	})
	if err != nil {
		return err
	}

	errCap := struct {
		Error  string      `json:"error"`
		Status string      `json:"status"`
		Reason interface{} `json:"reason"`
	}{}
	if err := json.Unmarshal(interim, &errCap); err == nil {
		if errCap.Error != "" {
			return errors.New(errCap.Error)
		}
		if data, ok := errCap.Reason.(map[string][]string); ok {
			var details strings.Builder
			for x := range data {
				details.WriteString(strings.Join(data[x], ""))
			}
			return errors.New(details.String())
		}

		if data, ok := errCap.Reason.(string); ok {
			return errors.New(data)
		}

		if errCap.Status != "" {
			return errors.New(errCap.Status)
		}
	}

	return json.Unmarshal(interim, result)
}

func parseTime(dateTime string) (time.Time, error) {
	return time.Parse(bitstampTimeLayout, dateTime)
}
