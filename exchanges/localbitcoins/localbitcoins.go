package localbitcoins

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
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	LOCALBITCOINS_API_URL             = "https://localbitcoins.com"
	LOCALBITCOINS_API_TICKER          = "/bitcoinaverage/ticker-all-currencies/"
	LOCALBITCOINS_API_BITCOINCHARTS   = "/bitcoincharts/"
	LOCALBITCOINS_API_PINCODE         = "pincode/"
	LOCALBITCOINS_API_WALLET          = "wallet/"
	LOCALBITCOINS_API_MYSELF          = "myself/"
	LOCALBITCOINS_API_WALLET_BALANCE  = "wallet-balance/"
	LOCALBITCOINS_API_WALLET_SEND     = "wallet-send/"
	LOCALBITCOINS_API_WALLET_SEND_PIN = "wallet-send-pin/"
	LOCALBITCOINS_API_WALLET_ADDRESS  = "wallet-addr/"
)

type LocalBitcoins struct {
	exchange.Base
}

func (l *LocalBitcoins) SetDefaults() {
	l.Name = "LocalBitcoins"
	l.Enabled = false
	l.Verbose = false
	l.Verbose = false
	l.Websocket = false
	l.RESTPollingDelay = 10
	l.RequestCurrencyPairFormat.Delimiter = ""
	l.RequestCurrencyPairFormat.Uppercase = true
	l.ConfigCurrencyPairFormat.Delimiter = ""
	l.ConfigCurrencyPairFormat.Uppercase = true
	l.AssetTypes = []string{ticker.Spot}
}

func (l *LocalBitcoins) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		l.SetEnabled(false)
	} else {
		l.Enabled = true
		l.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		l.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		l.RESTPollingDelay = exch.RESTPollingDelay
		l.Verbose = exch.Verbose
		l.Websocket = exch.Websocket
		l.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		l.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		l.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := l.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (l *LocalBitcoins) GetFee(maker bool) float64 {
	if maker {
		return l.MakerFee
	} else {
		return l.TakerFee
	}
}

func (l *LocalBitcoins) GetTicker() (map[string]LocalBitcoinsTicker, error) {
	result := make(map[string]LocalBitcoinsTicker)
	err := common.SendHTTPGetRequest(LOCALBITCOINS_API_URL+LOCALBITCOINS_API_TICKER, true, l.Verbose, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (l *LocalBitcoins) GetTrades(currency string, values url.Values) ([]LocalBitcoinsTrade, error) {
	path := common.EncodeURLValues(fmt.Sprintf("%s/%s/trades.json", LOCALBITCOINS_API_URL+LOCALBITCOINS_API_BITCOINCHARTS, currency), values)
	result := []LocalBitcoinsTrade{}
	err := common.SendHTTPGetRequest(path, true, l.Verbose, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (l *LocalBitcoins) GetOrderbook(currency string) (LocalBitcoinsOrderbook, error) {
	type response struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}

	path := fmt.Sprintf("%s/%s/orderbook.json", LOCALBITCOINS_API_URL+LOCALBITCOINS_API_BITCOINCHARTS, currency)
	resp := response{}
	err := common.SendHTTPGetRequest(path, true, l.Verbose, &resp)

	if err != nil {
		return LocalBitcoinsOrderbook{}, err
	}

	orderbook := LocalBitcoinsOrderbook{}

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
		orderbook.Bids = append(orderbook.Bids, LocalBitcoinsOrderbookStructure{price, amount})
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
		orderbook.Asks = append(orderbook.Asks, LocalBitcoinsOrderbookStructure{price, amount})
	}

	return orderbook, nil
}

func (l *LocalBitcoins) GetAccountInfo(username string, self bool) (LocalBitcoinsAccountInfo, error) {
	type response struct {
		Data LocalBitcoinsAccountInfo `json:"data"`
	}
	resp := response{}

	if self {
		err := l.SendAuthenticatedHTTPRequest("GET", LOCALBITCOINS_API_MYSELF, nil, &resp)

		if err != nil {
			return resp.Data, err
		}
	} else {
		path := fmt.Sprintf("%s/api/account_info/%s/", LOCALBITCOINS_API_URL, username)
		err := common.SendHTTPGetRequest(path, true, l.Verbose, &resp)

		if err != nil {
			return resp.Data, err
		}
	}

	return resp.Data, nil
}

func (l *LocalBitcoins) CheckPincode(pin int) (bool, error) {
	type response struct {
		Data struct {
			PinOK bool `json:"pincode_ok"`
		} `json:"data"`
	}
	resp := response{}
	values := url.Values{}
	values.Set("pincode", strconv.Itoa(pin))
	err := l.SendAuthenticatedHTTPRequest("POST", LOCALBITCOINS_API_PINCODE, values, &resp)

	if err != nil {
		return false, err
	}

	if !resp.Data.PinOK {
		return false, errors.New("Pin invalid.")
	}

	return true, nil
}

func (l *LocalBitcoins) GetWalletInfo() (LocalBitcoinsWalletInfo, error) {
	type response struct {
		Data LocalBitcoinsWalletInfo `json:"data"`
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("GET", LOCALBITCOINS_API_WALLET, nil, &resp)

	if err != nil {
		return LocalBitcoinsWalletInfo{}, err
	}

	if resp.Data.Message != "OK" {
		return LocalBitcoinsWalletInfo{}, errors.New("Unable to fetch wallet info.")
	}

	return resp.Data, nil
}

func (l *LocalBitcoins) GetWalletBalance() (LocalBitcoinsWalletBalanceInfo, error) {
	type response struct {
		Data LocalBitcoinsWalletBalanceInfo `json:"data"`
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("GET", LOCALBITCOINS_API_WALLET_BALANCE, nil, &resp)

	if err != nil {
		return LocalBitcoinsWalletBalanceInfo{}, err
	}

	if resp.Data.Message != "OK" {
		return LocalBitcoinsWalletBalanceInfo{}, errors.New("Unable to fetch wallet balance.")
	}

	return resp.Data, nil
}

func (l *LocalBitcoins) WalletSend(address string, amount float64, pin int) (bool, error) {
	values := url.Values{}
	values.Set("address", address)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	path := LOCALBITCOINS_API_WALLET_SEND

	if pin > 0 {
		values.Set("pincode", strconv.Itoa(pin))
		path = LOCALBITCOINS_API_WALLET_SEND_PIN
	}

	type response struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}

	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("POST", path, values, &resp)
	if err != nil {
		return false, err
	}

	if resp.Data.Message != "Money is being sent" {
		return false, errors.New("Unable to send Bitcoins.")
	}

	return true, nil
}

func (l *LocalBitcoins) GetWalletAddress() (string, error) {
	type response struct {
		Data struct {
			Message string `json:"message"`
			Address string `json:"address"`
		}
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("POST", LOCALBITCOINS_API_WALLET_ADDRESS, nil, &resp)
	if err != nil {
		return "", err
	}

	if resp.Data.Message != "OK!" {
		return "", errors.New("Unable to fetch wallet address.")
	}

	return resp.Data.Address, nil
}

func (l *LocalBitcoins) SendAuthenticatedHTTPRequest(method, path string, values url.Values, result interface{}) (err error) {
	if !l.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, l.Name)
	}

	if l.Nonce.Get() == 0 {
		l.Nonce.Set(time.Now().UnixNano())
	} else {
		l.Nonce.Inc()
	}

	payload := ""
	path = "/api/" + path

	if len(values) > 0 {
		payload = values.Encode()
	}

	message := l.Nonce.String() + l.APIKey + path + payload
	hmac := common.GetHMAC(common.HashSHA256, []byte(message), []byte(l.APISecret))
	headers := make(map[string]string)
	headers["Apiauth-Key"] = l.APIKey
	headers["Apiauth-Nonce"] = l.Nonce.String()
	headers["Apiauth-Signature"] = common.StringToUpper(common.HexEncodeToString(hmac))
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest(method, LOCALBITCOINS_API_URL+path, headers, bytes.NewBuffer([]byte(payload)))

	if l.Verbose {
		log.Printf("Received raw: \n%s\n", resp)
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("unable to JSON Unmarshal response")
	}

	return nil
}
