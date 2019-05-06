package btcc

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
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
	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, btccAuthRate),
		request.NewRateLimit(time.Second, btccUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	b.Websocket.Functionality =  
		exchange.WebsocketSubscribeSupported |
		exchange.WebsocketUnsubscribeSupported
	b.WebsocketInit()
}

// Setup is run on startup to setup exchange with config values
func (b *BTCC) Setup(exch *config.ExchangeConfig) {
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
		err = b.WebsocketSetup(b.WsConnect,
			b.Subscribe,
			b.Unsubscribe,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			btccSocketioAddress,
			exch.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns an estimate of fee based on type of transaction
func (b *BTCC) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency, feeBuilder.Amount)
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
	return 0.001 * price * amount
}

func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

func getInternationalBankWithdrawalFee(c currency.Code, amount float64) float64 {
	return WithdrawalFees[c] * amount
}
