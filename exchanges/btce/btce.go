package btce

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	BTCE_API_PUBLIC_URL      = "https://btc-e.com/api"
	BTCE_API_PRIVATE_URL     = "https://btc-e.com/tapi"
	BTCE_API_PUBLIC_VERSION  = "3"
	BTCE_API_PRIVATE_VERSION = "1"
	BTCE_INFO                = "info"
	BTCE_TICKER              = "ticker"
	BTCE_DEPTH               = "depth"
	BTCE_TRADES              = "trades"
	BTCE_ACCOUNT_INFO        = "getInfo"
	BTCE_TRADE               = "Trade"
	BTCE_ACTIVE_ORDERS       = "ActiveOrders"
	BTCE_ORDER_INFO          = "OrderInfo"
	BTCE_CANCEL_ORDER        = "CancelOrder"
	BTCE_TRADE_HISTORY       = "TradeHistory"
	BTCE_TRANSACTION_HISTORY = "TransHistory"
	BTCE_WITHDRAW_COIN       = "WithdrawCoin"
	BTCE_CREATE_COUPON       = "CreateCoupon"
	BTCE_REDEEM_COUPON       = "RedeemCoupon"
)

type BTCE struct {
	exchange.Base
	Ticker map[string]BTCeTicker
}

func (b *BTCE) SetDefaults() {
	b.Name = "BTCE"
	b.Enabled = false
	b.Fee = 0.2
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
	b.Ticker = make(map[string]BTCeTicker)
	b.RequestCurrencyPairFormat.Delimiter = "_"
	b.RequestCurrencyPairFormat.Uppercase = false
	b.RequestCurrencyPairFormat.Separator = "-"
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.AssetTypes = []string{ticker.Spot}
}

func (b *BTCE) Setup(exch config.ExchangeConfig) {
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

func (b *BTCE) GetFee() float64 {
	return b.Fee
}

func (b *BTCE) GetInfo() (BTCEInfo, error) {
	req := fmt.Sprintf("%s/%s/%s/", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_INFO)
	resp := BTCEInfo{}
	err := common.SendHTTPGetRequest(req, true, &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (b *BTCE) GetTicker(symbol string) (map[string]BTCeTicker, error) {
	type Response struct {
		Data map[string]BTCeTicker
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_TICKER, symbol)
	err := common.SendHTTPGetRequest(req, true, &response.Data)

	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (b *BTCE) GetDepth(symbol string) (BTCEOrderbook, error) {
	type Response struct {
		Data map[string]BTCEOrderbook
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_DEPTH, symbol)

	err := common.SendHTTPGetRequest(req, true, &response.Data)
	if err != nil {
		return BTCEOrderbook{}, err
	}

	depth := response.Data[symbol]
	return depth, nil
}

func (b *BTCE) GetTrades(symbol string) ([]BTCETrades, error) {
	type Response struct {
		Data map[string][]BTCETrades
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_TRADES, symbol)

	err := common.SendHTTPGetRequest(req, true, &response.Data)
	if err != nil {
		return []BTCETrades{}, err
	}

	trades := response.Data[symbol]
	return trades, nil
}

func (b *BTCE) GetAccountInfo() (BTCEAccountInfo, error) {
	var result BTCEAccountInfo
	err := b.SendAuthenticatedHTTPRequest(BTCE_ACCOUNT_INFO, url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *BTCE) GetActiveOrders(pair string) (map[string]BTCEActiveOrders, error) {
	req := url.Values{}
	req.Add("pair", pair)

	var result map[string]BTCEActiveOrders
	err := b.SendAuthenticatedHTTPRequest(BTCE_ACTIVE_ORDERS, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *BTCE) GetOrderInfo(OrderID int64) (map[string]BTCEOrderInfo, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	var result map[string]BTCEOrderInfo
	err := b.SendAuthenticatedHTTPRequest(BTCE_ORDER_INFO, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *BTCE) CancelOrder(OrderID int64) (bool, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	var result BTCECancelOrder
	err := b.SendAuthenticatedHTTPRequest(BTCE_CANCEL_ORDER, req, &result)

	if err != nil {
		return false, err
	}

	return true, nil
}

//to-do: convert orderid to int64
func (b *BTCE) Trade(pair, orderType string, amount, price float64) (float64, error) {
	req := url.Values{}
	req.Add("pair", pair)
	req.Add("type", orderType)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("rate", strconv.FormatFloat(price, 'f', -1, 64))

	var result BTCETrade
	err := b.SendAuthenticatedHTTPRequest(BTCE_TRADE, req, &result)

	if err != nil {
		return 0, err
	}

	return result.OrderID, nil
}

func (b *BTCE) GetTransactionHistory(TIDFrom, Count, TIDEnd int64, order, since, end string) (map[string]BTCETransHistory, error) {
	req := url.Values{}
	req.Add("from", strconv.FormatInt(TIDFrom, 10))
	req.Add("count", strconv.FormatInt(Count, 10))
	req.Add("from_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("end_id", strconv.FormatInt(TIDEnd, 10))
	req.Add("order", order)
	req.Add("since", since)
	req.Add("end", end)

	var result map[string]BTCETransHistory
	err := b.SendAuthenticatedHTTPRequest(BTCE_TRANSACTION_HISTORY, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *BTCE) GetTradeHistory(TIDFrom, Count, TIDEnd int64, order, since, end, pair string) (map[string]BTCETradeHistory, error) {
	req := url.Values{}

	req.Add("from", strconv.FormatInt(TIDFrom, 10))
	req.Add("count", strconv.FormatInt(Count, 10))
	req.Add("from_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("end_id", strconv.FormatInt(TIDEnd, 10))
	req.Add("order", order)
	req.Add("since", since)
	req.Add("end", end)
	req.Add("pair", pair)

	var result map[string]BTCETradeHistory
	err := b.SendAuthenticatedHTTPRequest(BTCE_TRADE_HISTORY, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *BTCE) WithdrawCoins(coin string, amount float64, address string) (BTCEWithdrawCoins, error) {
	req := url.Values{}

	req.Add("coinName", coin)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	var result BTCEWithdrawCoins
	err := b.SendAuthenticatedHTTPRequest(BTCE_WITHDRAW_COIN, req, &result)

	if err != nil {
		return result, err
	}
	return result, nil
}

func (b *BTCE) CreateCoupon(currency string, amount float64) (BTCECreateCoupon, error) {
	req := url.Values{}

	req.Add("currency", currency)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	var result BTCECreateCoupon
	err := b.SendAuthenticatedHTTPRequest(BTCE_CREATE_COUPON, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *BTCE) RedeemCoupon(coupon string) (BTCERedeemCoupon, error) {
	req := url.Values{}

	req.Add("coupon", coupon)

	var result BTCERedeemCoupon
	err := b.SendAuthenticatedHTTPRequest(BTCE_REDEEM_COUPON, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *BTCE) SendAuthenticatedHTTPRequest(method string, values url.Values, result interface{}) (err error) {
	if !b.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}

	if b.Nonce.Get() == 0 {
		b.Nonce.Set(time.Now().Unix())
	} else {
		b.Nonce.Inc()
	}
	values.Set("nonce", b.Nonce.String())
	values.Set("method", method)

	encoded := values.Encode()
	hmac := common.GetHMAC(common.HashSHA512, []byte(encoded), []byte(b.APISecret))

	if b.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", BTCE_API_PRIVATE_URL, method, encoded)
	}

	headers := make(map[string]string)
	headers["Key"] = b.APIKey
	headers["Sign"] = common.HexEncodeToString(hmac)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest("POST", BTCE_API_PRIVATE_URL, headers, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	response := BTCEResponse{}
	err = common.JSONDecode([]byte(resp), &response)

	if err != nil {
		return err
	}

	if response.Success != 1 {
		return errors.New(response.Error)
	}

	JSONEncoded, err := common.JSONEncode(response.Return)

	if err != nil {
		return err
	}

	err = common.JSONDecode(JSONEncoded, &result)

	if err != nil {
		return err
	}
	return nil
}
