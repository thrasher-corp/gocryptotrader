package currency

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency/coinmarketcap"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider"
)

// CurrencyFileUpdateDelay defines the rate at which the currency.json file is
// updated
const (
	DefaultCurrencyFileDelay    = 168 * time.Hour
	DefaultForeignExchangeDelay = 1 * time.Minute
	DefaultStorageFile          = "currency.json"
)

// storage is an overarching type that keeps track of and updates currency,
// currency exchange rates and pairs
var storage Storage

// Storage contains the loaded storage currencies supported on available crypto
// or fiat marketplaces
// NOTE: All internal currencies are upper case
type Storage struct {
	// FiatCurrencies defines the running fiat currencies in the currency
	// storage
	fiatCurrencies Currencies
	// Cryptocurrencies defines the running cryptocurrencies in the currency
	// storage
	cryptocurrencies Currencies
	// CurrencyCodes is a full basket of currencies either crypto, fiat, ico or
	// contract being tracked by the currency storage system
	currencyCodes BaseCodes
	// Main converting currency
	baseCurrency Code
	// FXRates defines a protected conversion rate map
	fxRates ConversionRates
	// DefaultBaseCurrency is the base currency used for conversion
	defaultBaseCurrency Code
	// DefaultFiatCurrencies has the default minimum of FIAT values
	defaultFiatCurrencies Currencies
	// DefaultCryptoCurrencies has the default minimum of crytpocurrency values
	defaultCryptoCurrencies Currencies
	// FiatExchangeMarkets defines an interface to access FX data for fiat
	// currency rates
	fiatExchangeMarkets *forexprovider.ForexProviders
	// CurrencyAnalysis defines a full market analysis suite to receive and
	// define different fiat currencies, cryptocurrencies and markets
	currencyAnalysis *coinmarketcap.Coinmarketcap
	// Path defines the main folder to dump and find currency JSON
	path string
	// Update delay variables
	currencyFileUpdateDelay    time.Duration
	foreignExchangeUpdateDelay time.Duration
	mtx                        sync.Mutex
	wg                         sync.WaitGroup
	shutdown                   chan struct{}
	updaterRunning             bool
	Verbose                    bool
}
