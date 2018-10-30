package btcc

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	btccAuthRate   = 0
	btccUnauthRate = 0
)

// BTCC is the main overaching type across the BTCC package
// NOTE this package is websocket connection dependant, the REST endpoints have
// been dropped
type BTCC struct {
	exchange.Base
	Conn *websocket.Conn
}

// SetDefaults sets default values for the exchange
func (b *BTCC) SetDefaults() {
	b.Name = "BTCC"
	b.Enabled = false
	b.Fee = 0
	b.Verbose = false
	b.RESTPollingDelay = 10
	b.APIWithdrawPermissions = exchange.NoAPIWithdrawalMethods
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.RequestCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.AssetTypes = []string{ticker.Spot}
	b.SupportsAutoPairUpdating = true
	b.SupportsRESTTickerBatching = false
	b.SupportsRESTAPI = false
	b.SupportsWebsocketAPI = true
	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, btccAuthRate),
		request.NewRateLimit(time.Second, btccUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	b.WebsocketInit()
}

// Setup is run on startup to setup exchange with config values
func (b *BTCC) Setup(exch config.ExchangeConfig) {
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
		b.Websocket.SetEnabled(exch.Websocket)
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
		err = b.WebsocketSetup(b.WsConnect,
			exch.Name,
			exch.Websocket,
			btccSocketioAddress,
			exch.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns an estimate of fee based on type of transaction
func (b *BTCC) GetFee(feeBuilder exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.FirstCurrency)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.CurrencyItem, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

func getCryptocurrencyWithdrawalFee(currency string) float64 {
	return WithdrawalFees[currency]
}

func getInternationalBankWithdrawalFee(currency string, amount float64) float64 {
	var fee float64

	fee = WithdrawalFees[currency] * amount
	return fee
}
