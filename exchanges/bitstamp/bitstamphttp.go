package bitstamp

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
)

const (
	BITSTAMP_API_URL                 = "https://www.bitstamp.net/api"
	BITSTAMP_API_VERSION             = "2"
	BITSTAMP_API_TICKER              = "ticker"
	BITSTAMP_API_TICKER_HOURLY       = "ticker_hour"
	BITSTAMP_API_ORDERBOOK           = "order_book"
	BITSTAMP_API_TRANSACTIONS        = "transactions"
	BITSTAMP_API_EURUSD              = "eur_usd"
	BITSTAMP_API_BALANCE             = "balance"
	BITSTAMP_API_USER_TRANSACTIONS   = "user_transactions"
	BITSTAMP_API_OPEN_ORDERS         = "open_orders"
	BITSTAMP_API_ORDER_STATUS        = "order_status"
	BITSTAMP_API_CANCEL_ORDER        = "cancel_order"
	BITSTAMP_API_CANCEL_ALL_ORDERS   = "cancel_all_orders"
	BITSTAMP_API_BUY                 = "buy"
	BITSTAMP_API_SELL                = "sell"
	BITSTAMP_API_MARKET              = "market"
	BITSTAMP_API_WITHDRAWAL_REQUESTS = "withdrawal_requests"
	BITSTAMP_API_BITCOIN_WITHDRAWAL  = "bitcoin_withdrawal"
	BITSTAMP_API_BITCOIN_DEPOSIT     = "bitcoin_deposit_address"
	BITSTAMP_API_UNCONFIRMED_BITCOIN = "unconfirmed_btc"
	BITSTAMP_API_RIPPLE_WITHDRAWAL   = "ripple_withdrawal"
	BITSTAMP_API_RIPPLE_DESPOIT      = "ripple_address"
	BITSTAMP_API_TRANSFER_TO_MAIN    = "transfer-to-main"
	BITSTAMP_API_TRANSFER_FROM_MAIN  = "transfer-from-main"
	BITSTAMP_API_XRP_WITHDRAWAL      = "xrp_withdrawal"
	BITSTAMP_API_XRP_DESPOIT         = "xrp_address"
)

type Bitstamp struct {
	exchange.ExchangeBase
	Balance BitstampBalances
}

func (b *Bitstamp) SetDefaults() {
	b.Name = "Bitstamp"
	b.Enabled = false
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
}

func (b *Bitstamp) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, false)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

func (b *Bitstamp) GetFee(currency string) float64 {
	switch currency {
	case "BTCUSD":
		return b.Balance.BTCUSDFee
	case "BTCEUR":
		return b.Balance.BTCEURFee
	case "XRPEUR":
		return b.Balance.XRPEURFee
	case "XRPUSD":
		return b.Balance.XRPUSDFee
	case "EURUSD":
		return b.Balance.EURUSDFee
	default:
		return 0
	}
}

func (b *Bitstamp) GetTicker(currency string, hourly bool) (BitstampTicker, error) {
	tickerEndpoint := BITSTAMP_API_TICKER
	if hourly {
		tickerEndpoint = BITSTAMP_API_TICKER_HOURLY
	}

	path := fmt.Sprintf("%s/v%s/%s/%s/", BITSTAMP_API_URL, BITSTAMP_API_VERSION, tickerEndpoint, common.StringToLower(currency))
	ticker := BitstampTicker{}

	err := common.SendHTTPGetRequest(path, true, &ticker)

	if err != nil {
		return ticker, err
	}

	return ticker, nil
}

func (b *Bitstamp) GetOrderbook(currency string) (BitstampOrderbook, error) {
	type response struct {
		Timestamp int64 `json:"timestamp,string"`
		Bids      [][]string
		Asks      [][]string
	}

	resp := response{}
	path := fmt.Sprintf("%s/v%s/%s/%s/", BITSTAMP_API_URL, BITSTAMP_API_VERSION, BITSTAMP_API_ORDERBOOK, common.StringToLower(currency))
	err := common.SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return BitstampOrderbook{}, err
	}

	orderbook := BitstampOrderbook{}
	orderbook.Timestamp = resp.Timestamp

	for _, x := range resp.Bids {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		orderbook.Bids = append(orderbook.Bids, BitstampOrderbookBase{price, amount})
	}

	for _, x := range resp.Asks {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		orderbook.Asks = append(orderbook.Asks, BitstampOrderbookBase{price, amount})
	}

	return orderbook, nil
}

func (b *Bitstamp) GetTransactions(currency string, values url.Values) ([]BitstampTransactions, error) {
	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s/%s/", BITSTAMP_API_URL, BITSTAMP_API_VERSION, BITSTAMP_API_TRANSACTIONS, common.StringToLower(currency)), values)
	transactions := []BitstampTransactions{}
	err := common.SendHTTPGetRequest(path, true, &transactions)
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func (b *Bitstamp) GetEURUSDConversionRate() (BitstampEURUSDConversionRate, error) {
	rate := BitstampEURUSDConversionRate{}
	path := fmt.Sprintf("%s/%s", BITSTAMP_API_URL, BITSTAMP_API_EURUSD)
	err := common.SendHTTPGetRequest(path, true, &rate)

	if err != nil {
		return rate, err
	}
	return rate, nil
}

func (b *Bitstamp) GetBalance() (BitstampBalances, error) {
	balance := BitstampBalances{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BALANCE, true, url.Values{}, &balance)

	if err != nil {
		return balance, err
	}
	return balance, nil
}

func (b *Bitstamp) GetUserTransactions(values url.Values) ([]BitstampUserTransactions, error) {
	type Response struct {
		Date    string      `json:"datetime"`
		TransID int64       `json:"id"`
		Type    int         `json:"type,string"`
		USD     interface{} `json:"usd"`
		EUR     float64     `json:"eur"`
		XRP     float64     `json:"xrp"`
		BTC     interface{} `json:"btc"`
		BTCUSD  interface{} `json:"btc_usd"`
		Fee     float64     `json:"fee,string"`
		OrderID int64       `json:"order_id"`
	}

	response := []Response{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_USER_TRANSACTIONS, true, values, &response)

	if err != nil {
		return nil, err
	}

	transactions := []BitstampUserTransactions{}

	for _, y := range response {
		tx := BitstampUserTransactions{}
		tx.Date = y.Date
		tx.TransID = y.TransID
		tx.Type = y.Type

		/* Hack due to inconsistent JSON values... */
		varType := reflect.TypeOf(y.USD).String()
		if varType == "string" {
			tx.USD, _ = strconv.ParseFloat(y.USD.(string), 64)
		} else {
			tx.USD = y.USD.(float64)
		}

		tx.EUR = y.EUR
		tx.XRP = y.XRP

		varType = reflect.TypeOf(y.BTC).String()
		if varType == "string" {
			tx.BTC, _ = strconv.ParseFloat(y.BTC.(string), 64)
		} else {
			tx.BTC = y.BTC.(float64)
		}

		varType = reflect.TypeOf(y.BTCUSD).String()
		if varType == "string" {
			tx.BTCUSD, _ = strconv.ParseFloat(y.BTCUSD.(string), 64)
		} else {
			tx.BTCUSD = y.BTCUSD.(float64)
		}

		tx.Fee = y.Fee
		tx.OrderID = y.OrderID
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func (b *Bitstamp) GetOpenOrders(currency string) ([]BitstampOrder, error) {
	resp := []BitstampOrder{}
	path := fmt.Sprintf("%s/%s", BITSTAMP_API_OPEN_ORDERS, common.StringToLower(currency))
	err := b.SendAuthenticatedHTTPRequest(path, true, nil, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (b *Bitstamp) GetOrderStatus(OrderID int64) (BitstampOrderStatus, error) {
	var req = url.Values{}
	req.Add("id", strconv.FormatInt(OrderID, 10))
	resp := BitstampOrderStatus{}

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ORDER, false, req, &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (b *Bitstamp) CancelOrder(OrderID int64) (bool, error) {
	var req = url.Values{}
	result := false
	req.Add("id", strconv.FormatInt(OrderID, 10))

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ORDER, true, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *Bitstamp) CancelAllOrders() (bool, error) {
	result := false
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ALL_ORDERS, false, nil, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *Bitstamp) PlaceOrder(currency string, price float64, amount float64, buy, market bool) (BitstampOrder, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("price", strconv.FormatFloat(price, 'f', -1, 64))
	response := BitstampOrder{}
	orderType := BITSTAMP_API_BUY
	path := ""

	if !buy {
		orderType = BITSTAMP_API_SELL
	}

	path = fmt.Sprintf("%s/%s", orderType, common.StringToLower(currency))

	if market {
		path = fmt.Sprintf("%s/%s/%s", orderType, BITSTAMP_API_MARKET, common.StringToLower(currency))
	}

	err := b.SendAuthenticatedHTTPRequest(path, true, req, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

func (b *Bitstamp) GetWithdrawalRequests(values url.Values) ([]BitstampWithdrawalRequests, error) {
	resp := []BitstampWithdrawalRequests{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_WITHDRAWAL_REQUESTS, false, values, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (b *Bitstamp) BitcoinWithdrawal(amount float64, address string, instant bool) (string, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	if instant {
		req.Add("instant", "1")
	} else {
		req.Add("instant", "0")
	}

	type response struct {
		ID string `json:"id"`
	}

	resp := response{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BITCOIN_WITHDRAWAL, false, req, &resp)

	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (b *Bitstamp) GetBitcoinDepositAddress() (string, error) {
	address := ""
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BITCOIN_DEPOSIT, false, url.Values{}, &address)

	if err != nil {
		return address, err
	}
	return address, nil
}

func (b *Bitstamp) GetUnconfirmedBitcoinDeposits() ([]BitstampUnconfirmedBTCTransactions, error) {
	response := []BitstampUnconfirmedBTCTransactions{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_UNCONFIRMED_BITCOIN, false, nil, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (b *Bitstamp) RippleWithdrawal(amount float64, address, currency string) (bool, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)
	req.Add("currency", currency)

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_RIPPLE_WITHDRAWAL, false, req, nil)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (b *Bitstamp) GetRippleDepositAddress() (string, error) {
	type response struct {
		Address string
	}

	resp := response{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_RIPPLE_DESPOIT, false, nil, &resp)

	if err != nil {
		return "", err
	}

	return resp.Address, nil
}

func (b *Bitstamp) TransferAccountBalance(amount float64, currency, subAccount string, toMain bool) (bool, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("currency", currency)
	req.Add("subAccount", subAccount)

	path := BITSTAMP_API_TRANSFER_TO_MAIN
	if !toMain {
		path = BITSTAMP_API_TRANSFER_FROM_MAIN
	}

	err := b.SendAuthenticatedHTTPRequest(path, true, req, nil)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (b *Bitstamp) XRPWithdrawal(amount float64, address, destTag string) (string, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)
	if destTag != "" {
		req.Add("destination_tag", destTag)
	}

	type response struct {
		ID string `json:"id"`
	}

	resp := response{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_XRP_WITHDRAWAL, true, req, &resp)

	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (b *Bitstamp) GetXRPDepositAddress() (BitstampXRPDepositResponse, error) {
	resp := BitstampXRPDepositResponse{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_XRP_DESPOIT, true, nil, &resp)

	if err != nil {
		return BitstampXRPDepositResponse{}, err
	}

	return resp, nil
}

func (b *Bitstamp) SendAuthenticatedHTTPRequest(path string, v2 bool, values url.Values, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)

	if values == nil {
		values = url.Values{}
	}

	values.Set("key", b.APIKey)
	values.Set("nonce", nonce)
	hmac := common.GetHMAC(common.HASH_SHA256, []byte(nonce+b.ClientID+b.APIKey), []byte(b.APISecret))
	values.Set("signature", common.StringToUpper(common.HexEncodeToString(hmac)))

	if v2 {
		path = fmt.Sprintf("%s/v%s/%s/", BITSTAMP_API_URL, BITSTAMP_API_VERSION, path)
	} else {
		path = fmt.Sprintf("%s/%s/", BITSTAMP_API_URL, path)
	}

	if b.Verbose {
		log.Println("Sending POST request to " + path)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest("POST", path, headers, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Recieved raw: %s\n", resp)
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
