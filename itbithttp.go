package main

import (
	"net/http"
	"strconv"
	"crypto/sha512"
	"errors"
	"strings"
	"time"
	"encoding/json"
	"io/ioutil"
	"log"
)

const (
	ITBIT_API_URL = "https://api.itbit.com/v1/"
)

type ItBit struct {
	Name string
	Enabled bool
	Verbose bool
	ClientKey, APISecret, UserID string
	MakerFee, TakerFee float64
}

type ItBitTicker struct {
	Pair string
	Bid float64 `json:",string"`
	BidAmt float64 `json:",string"`
	Ask float64 `json:",string"`
	AskAmt float64 `json:",string"`
	LastPrice float64 `json:",string"`
	LastAmt float64 `json:",string"`
	Volume24h float64 `json:",string"`
	VolumeToday float64 `json:",string"`
	High24h float64 `json:",string"`
	Low24h float64 `json:",string"`
	HighToday float64 `json:",string"`
	LowToday float64 `json:",string"`
	OpenToday float64 `json:",string"`
	VwapToday float64 `json:",string"`
	Vwap24h float64 `json:",string"`
	ServertimeUTC string
}

func (i *ItBit) SetDefaults() {
	i.Name = "ITBIT"
	i.Enabled = true
	i.MakerFee = -0.10
	i.TakerFee = 0.50
	i.Verbose = false
}

func (i *ItBit) GetName() (string) {
	return i.Name
}

func (i *ItBit) SetEnabled(enabled bool) {
	i.Enabled = enabled
}

func (i *ItBit) IsEnabled() (bool) {
	return i.Enabled
}

func (i *ItBit) SetAPIKeys(apiKey, apiSecret string) {
	i.ClientKey = apiKey
	i.APISecret = apiSecret
}

func (i *ItBit) GetFee(maker bool) (float64) {
	if maker {
		return i.MakerFee
	} else {
		return i.TakerFee
	}
}

func (i *ItBit) GetTicker(currency string) (ItBitTicker) {
	path := ITBIT_API_URL + "/markets/" + currency + "/ticker"
	var itbitTicker ItBitTicker
	err := SendHTTPRequest(path, true, &itbitTicker)
	if err != nil {
		log.Println(err)
		return ItBitTicker{}
	}
	return itbitTicker
}

func (i *ItBit) GetOrderbook(currency string) (bool) {
	path := ITBIT_API_URL + "/markets/" + currency + "/orders"
	err := SendHTTPRequest(path , true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (i *ItBit) GetTradeHistory(currency, timestamp string) (bool) {
	req := "/trades?since=" + timestamp
	err := SendHTTPRequest(ITBIT_API_URL + "markets/" + currency + req, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (i *ItBit) GetWallets(page int64, perPage int64, userID string) {
	path := ITBIT_API_URL + "wallets/"
	params := make(map[string]interface{})
	params["page"] = strconv.FormatInt(page, 10)
	params["perPage"] = strconv.FormatInt(perPage, 10)
	params["userID"] = userID

	err := i.SendAuthenticatedHTTPRequest("GET", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWallet(walletID string) {
	path := ITBIT_API_URL + "/wallets/" + walletID
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletBalance(walletID, currency string) {
	path := ITBIT_API_URL + "/wallets/ " + walletID +  "/balances/" + currency
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletTrades(walletID string, page int64, perPage int64, rangeEnd int64, rangeStart int64) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/trades"
	params := make(map[string]interface{})
	params["page"] = strconv.FormatInt(page, 10)
	params["perPage"] = strconv.FormatInt(perPage, 10)
	params["rangeEnd"] = strconv.FormatInt(page, 10)
	params["rangeStart"] = strconv.FormatInt(perPage, 10)

	err := i.SendAuthenticatedHTTPRequest("GET", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletOrders(walletID string, instrument string, page int64, perPage int64, status string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/orders"
	params := make(map[string]interface{})
	params["instrument"] = instrument
	params["page"] = strconv.FormatInt(page, 10)
	params["perPage"] = strconv.FormatInt(perPage, 10)
	params["status"] = status

	err := i.SendAuthenticatedHTTPRequest("GET", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) PlaceWalletOrder(walletID, side, orderType, currency string, amount, price float64, instrument string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/orders"
	params := make(map[string]interface{})
	params["side"] = side
	params["type"] = orderType
	params["currency"] = currency
	params["amount"] = strconv.FormatFloat(amount, 'f', 8, 64)
	params["price"] = strconv.FormatFloat(price, 'f', 2, 64)
	params["instrument"] = instrument

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletOrder(walletID, orderID string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/orders/" + orderID
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) CancelWalletOrder(walletID, orderID string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/orders/" + orderID
	err := i.SendAuthenticatedHTTPRequest("DELETE", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) PlaceWithdrawalRequest(walletID, currency, address string, amount float64) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/cryptocurrency_withdrawals"
	params := make(map[string]interface{})
	params["currency"] = currency
	params["amount"] = strconv.FormatFloat(amount, 'f', 8, 64)
	params["address"] = address

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetDepositAddress(walletID, currency string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/cryptocurrency_deposits"
	params := make(map[string]interface{})
	params["currency"] = currency

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) WalletTransfer(walletID, sourceWallet, destWallet string, amount float64, currency string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/wallet_transfers"
	params := make(map[string]interface{})
	params["sourceWalletId"] = sourceWallet
	params["destinationWalletId"] = destWallet
	params["amount"] = strconv.FormatFloat(amount, 'f', 8, 64)
	params["currencyCode"] = currency

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) SendAuthenticatedHTTPRequest(method string, path string, params map[string]interface{}) (err error) {
	request := make(map[string]interface{})
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	request["nonce"] = nonce
	request["timestamp"] = nonce

	if params != nil {
		for key, value:= range params {
			request[key] = value
		}
	}

	PayloadJson, err := json.Marshal(request)	
	
	if i.Verbose {
		log.Printf("Request JSON: %s\n", PayloadJson)
	}

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	hmac := GetHMAC(sha512.New, []byte(nonce + string(PayloadJson)), []byte(i.APISecret))
	signature := Base64Encode([]byte(HexEncodeToString(hmac)))
	req, err := http.NewRequest(method, path, strings.NewReader(""))

	req.Header.Add("Authorization", i.ClientKey + ":" + signature)
	req.Header.Add("X-Auth-Timestamp", nonce)
	req.Header.Add("X-Auth-Nonce", nonce)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to send request")
	}

	contents, _ := ioutil.ReadAll(resp.Body)

	if i.Verbose {
		log.Printf("Recieved raw: %s\n", string(contents))
	}
	
	resp.Body.Close()
	return nil
}