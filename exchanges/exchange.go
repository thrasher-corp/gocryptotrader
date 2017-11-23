package exchange

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	warningBase64DecryptSecretKeyFailed = "WARNING -- Exchange %s unable to base64 decode secret key.. Disabling Authenticated API support."

	// WarningAuthenticatedRequestWithoutCredentialsSet error message for authenticated request without credentials set
	WarningAuthenticatedRequestWithoutCredentialsSet = "WARNING -- Exchange %s authenticated HTTP request called but not supported due to unset/default API keys."
	// ErrExchangeNotFound is a constant for an error message
	ErrExchangeNotFound = "Exchange not found in dataset."
)

// AccountInfo is a Generic type to hold each exchange's holdings in
// all enabled currencies
type AccountInfo struct {
	ExchangeName string
	Currencies   []AccountCurrencyInfo
}

// AccountCurrencyInfo is a sub type to store currency name and value
type AccountCurrencyInfo struct {
	CurrencyName string
	TotalValue   float64
	Hold         float64
}

// Base stores the individual exchange information
type Base struct {
	Name                        string
	Enabled                     bool
	Verbose                     bool
	Websocket                   bool
	RESTPollingDelay            time.Duration
	AuthenticatedAPISupport     bool
	APISecret, APIKey, ClientID string
	Nonce                       nonce.Nonce
	TakerFee, MakerFee, Fee     float64
	BaseCurrencies              []string
	AvailablePairs              []string
	EnabledPairs                []string
	AssetTypes                  []string
	WebsocketURL                string
	APIUrl                      string
	RequestCurrencyPairFormat   config.CurrencyPairFormatConfig
	ConfigCurrencyPairFormat    config.CurrencyPairFormatConfig
}

// IBotExchange enforces standard functions for all exchanges supported in
// GoCryptoTrader
type IBotExchange interface {
	Setup(exch config.ExchangeConfig)
	Start()
	SetDefaults()
	GetName() string
	IsEnabled() bool
	GetTickerPrice(currency pair.CurrencyPair, assetType string) (ticker.Price, error)
	UpdateTicker(currency pair.CurrencyPair, assetType string) (ticker.Price, error)
	GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error)
	UpdateOrderbook(currency pair.CurrencyPair, assetType string) (orderbook.Base, error)
	GetEnabledCurrencies() []pair.CurrencyPair
	GetExchangeAccountInfo() (AccountInfo, error)
	GetAuthenticatedAPISupport() bool
}

// SetAssetTypes checks the exchange asset types (whether it supports SPOT,
// Binary or Futures) and sets it to a default setting if it doesn't exist
func (e *Base) SetAssetTypes() error {
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(e.Name)
	if err != nil {
		return err
	}

	update := false
	if exch.AssetTypes == "" {
		exch.AssetTypes = common.JoinStrings(e.AssetTypes, ",")
		update = true
	} else {
		e.AssetTypes = common.SplitStrings(exch.AssetTypes, ",")
	}

	if update {
		return cfg.UpdateExchangeConfig(exch)
	}

	return nil
}

// GetExchangeAssetTypes returns the asset types the exchange supports (SPOT,
// binary, futures)
func GetExchangeAssetTypes(exchName string) ([]string, error) {
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	return common.SplitStrings(exch.AssetTypes, ","), nil
}

// SetCurrencyPairFormat checks the exchange request and config currency pair
// formats and sets it to a default setting if it doesn't exist
func (e *Base) SetCurrencyPairFormat() error {
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(e.Name)
	if err != nil {
		return err
	}

	update := false
	if exch.RequestCurrencyPairFormat == nil {
		exch.RequestCurrencyPairFormat = &config.CurrencyPairFormatConfig{
			Delimiter: e.RequestCurrencyPairFormat.Delimiter,
			Uppercase: e.RequestCurrencyPairFormat.Uppercase,
			Separator: e.RequestCurrencyPairFormat.Separator,
			Index:     e.RequestCurrencyPairFormat.Index,
		}
		update = true
	} else {
		e.RequestCurrencyPairFormat = *exch.RequestCurrencyPairFormat
	}

	if exch.ConfigCurrencyPairFormat == nil {
		exch.ConfigCurrencyPairFormat = &config.CurrencyPairFormatConfig{
			Delimiter: e.ConfigCurrencyPairFormat.Delimiter,
			Uppercase: e.ConfigCurrencyPairFormat.Uppercase,
			Separator: e.ConfigCurrencyPairFormat.Separator,
			Index:     e.ConfigCurrencyPairFormat.Index,
		}
		update = true
	} else {
		e.ConfigCurrencyPairFormat = *exch.ConfigCurrencyPairFormat
	}

	if update {
		return cfg.UpdateExchangeConfig(exch)
	}
	return nil
}

// GetAuthenticatedAPISupport returns whether the exchange supports
// authenticated API requests
func (e *Base) GetAuthenticatedAPISupport() bool {
	return e.AuthenticatedAPISupport
}

// GetName is a method that returns the name of the exchange base
func (e *Base) GetName() string {
	return e.Name
}

// GetEnabledCurrencies is a method that returns the enabled currency pairs of
// the exchange base
func (e *Base) GetEnabledCurrencies() []pair.CurrencyPair {
	var pairs []pair.CurrencyPair
	for x := range e.EnabledPairs {
		var currencyPair pair.CurrencyPair
		if e.RequestCurrencyPairFormat.Delimiter != "" {
			if e.ConfigCurrencyPairFormat.Delimiter != "" {
				if e.ConfigCurrencyPairFormat.Delimiter == e.RequestCurrencyPairFormat.Delimiter {
					currencyPair = pair.NewCurrencyPairDelimiter(e.EnabledPairs[x],
						e.RequestCurrencyPairFormat.Delimiter)
				} else {
					currencyPair = pair.NewCurrencyPairDelimiter(e.EnabledPairs[x],
						e.ConfigCurrencyPairFormat.Delimiter)
					currencyPair.Delimiter = "-"
				}
			} else {
				if e.ConfigCurrencyPairFormat.Index != "" {
					currencyPair = pair.NewCurrencyPairFromIndex(e.EnabledPairs[x],
						e.ConfigCurrencyPairFormat.Index)
				} else {
					currencyPair = pair.NewCurrencyPair(e.EnabledPairs[x][0:3],
						e.EnabledPairs[x][3:])
				}
			}
		} else {
			if e.ConfigCurrencyPairFormat.Delimiter != "" {
				currencyPair = pair.NewCurrencyPairDelimiter(e.EnabledPairs[x],
					e.ConfigCurrencyPairFormat.Delimiter)
			} else {
				if e.ConfigCurrencyPairFormat.Index != "" {
					currencyPair = pair.NewCurrencyPairFromIndex(e.EnabledPairs[x],
						e.ConfigCurrencyPairFormat.Index)
				} else {
					currencyPair = pair.NewCurrencyPair(e.EnabledPairs[x][0:3],
						e.EnabledPairs[x][3:])
				}
			}
		}
		pairs = append(pairs, currencyPair)
	}
	return pairs
}

// GetAvailableCurrencies is a method that returns the available currency pairs
// of the exchange base
func (e *Base) GetAvailableCurrencies() []pair.CurrencyPair {
	var pairs []pair.CurrencyPair
	for x := range e.AvailablePairs {
		var currencyPair pair.CurrencyPair
		if e.RequestCurrencyPairFormat.Delimiter != "" {
			if e.ConfigCurrencyPairFormat.Delimiter != "" {
				if e.ConfigCurrencyPairFormat.Delimiter == e.RequestCurrencyPairFormat.Delimiter {
					currencyPair = pair.NewCurrencyPairDelimiter(e.AvailablePairs[x],
						e.RequestCurrencyPairFormat.Delimiter)
				} else {
					currencyPair = pair.NewCurrencyPairDelimiter(e.AvailablePairs[x],
						e.ConfigCurrencyPairFormat.Delimiter)
					currencyPair.Delimiter = "-"
				}
			} else {
				if e.ConfigCurrencyPairFormat.Index != "" {
					currencyPair = pair.NewCurrencyPairFromIndex(e.AvailablePairs[x],
						e.ConfigCurrencyPairFormat.Index)
				} else {
					currencyPair = pair.NewCurrencyPair(e.AvailablePairs[x][0:3],
						e.AvailablePairs[x][3:])
				}
			}
		} else {
			if e.ConfigCurrencyPairFormat.Delimiter != "" {
				currencyPair = pair.NewCurrencyPairDelimiter(e.AvailablePairs[x],
					e.ConfigCurrencyPairFormat.Delimiter)
			} else {
				if e.ConfigCurrencyPairFormat.Index != "" {
					currencyPair = pair.NewCurrencyPairFromIndex(e.AvailablePairs[x],
						e.ConfigCurrencyPairFormat.Index)
				} else {
					currencyPair = pair.NewCurrencyPair(e.AvailablePairs[x][0:3],
						e.AvailablePairs[x][3:])
				}
			}
		}
		pairs = append(pairs, currencyPair)
	}
	return pairs
}

// GetExchangeFormatCurrencySeperator returns whether or not a specific
// exchange contains a separator used for API requests
func GetExchangeFormatCurrencySeperator(exchName string) bool {
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(exchName)
	if err != nil {
		return false
	}

	if exch.RequestCurrencyPairFormat.Separator != "" {
		return true
	}
	return false
}

// GetAndFormatExchangeCurrencies returns a pair.CurrencyItem string containing
// the exchanges formatted currency pairs
func GetAndFormatExchangeCurrencies(exchName string, pairs []pair.CurrencyPair) (pair.CurrencyItem, error) {
	var currencyItems pair.CurrencyItem
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(exchName)
	if err != nil {
		return currencyItems, err
	}

	for x := range pairs {
		currencyItems += FormatExchangeCurrency(exchName, pairs[x])
		if x == len(pairs)-1 {
			continue
		}
		currencyItems += pair.CurrencyItem(exch.RequestCurrencyPairFormat.Separator)
	}
	return currencyItems, nil
}

// FormatExchangeCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func FormatExchangeCurrency(exchName string, p pair.CurrencyPair) pair.CurrencyItem {
	cfg := config.GetConfig()
	exch, _ := cfg.GetExchangeConfig(exchName)

	return p.Display(exch.RequestCurrencyPairFormat.Delimiter,
		exch.RequestCurrencyPairFormat.Uppercase)
}

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func FormatCurrency(p pair.CurrencyPair) pair.CurrencyItem {
	cfg := config.GetConfig()
	return p.Display(cfg.CurrencyPairFormat.Delimiter,
		cfg.CurrencyPairFormat.Uppercase)
}

// SetEnabled is a method that sets if the exchange is enabled
func (e *Base) SetEnabled(enabled bool) {
	e.Enabled = enabled
}

// IsEnabled is a method that returns if the current exchange is enabled
func (e *Base) IsEnabled() bool {
	return e.Enabled
}

// SetAPIKeys is a method that sets the current API keys for the exchange
func (e *Base) SetAPIKeys(APIKey, APISecret, ClientID string, b64Decode bool) {
	if !e.AuthenticatedAPISupport {
		return
	}

	e.APIKey = APIKey
	e.ClientID = ClientID

	if b64Decode {
		result, err := common.Base64Decode(APISecret)
		if err != nil {
			e.AuthenticatedAPISupport = false
			log.Printf(warningBase64DecryptSecretKeyFailed, e.Name)
		}
		e.APISecret = string(result)
	} else {
		e.APISecret = APISecret
	}
}

// UpdateEnabledCurrencies is a method that sets new pairs to the current
// exchange. Setting force to true upgrades the enabled currencies
func (e *Base) UpdateEnabledCurrencies(exchangeProducts []string, force bool) error {
	exchangeProducts = common.SplitStrings(common.StringToUpper(common.JoinStrings(exchangeProducts, ",")), ",")
	diff := common.StringSliceDifference(e.EnabledPairs, exchangeProducts)
	if force || len(diff) > 0 {
		cfg := config.GetConfig()
		exch, err := cfg.GetExchangeConfig(e.Name)
		if err != nil {
			return err
		}

		if force {
			log.Printf("%s forced update of enabled pairs.", e.Name)
		} else {
			log.Printf("%s Updating available pairs. Difference: %s.\n", e.Name, diff)
		}
		exch.EnabledPairs = common.JoinStrings(exchangeProducts, ",")
		e.EnabledPairs = exchangeProducts
		return cfg.UpdateExchangeConfig(exch)
	}
	return nil
}

// UpdateAvailableCurrencies is a method that sets new pairs to the current
// exchange. Setting force to true upgrades the available currencies
func (e *Base) UpdateAvailableCurrencies(exchangeProducts []string, force bool) error {
	exchangeProducts = common.SplitStrings(common.StringToUpper(common.JoinStrings(exchangeProducts, ",")), ",")
	diff := common.StringSliceDifference(e.AvailablePairs, exchangeProducts)
	if force || len(diff) > 0 {
		cfg := config.GetConfig()
		exch, err := cfg.GetExchangeConfig(e.Name)
		if err != nil {
			return err
		}

		if force {
			log.Printf("%s forced update of available pairs.", e.Name)
		} else {
			log.Printf("%s Updating available pairs. Difference: %s.\n", e.Name, diff)
		}
		exch.AvailablePairs = common.JoinStrings(exchangeProducts, ",")
		e.AvailablePairs = exchangeProducts
		return cfg.UpdateExchangeConfig(exch)
	}
	return nil
}
