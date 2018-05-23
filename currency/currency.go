package currency

import (
	"errors"
	"log"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-/gocryptotrader/currency/pair"
)

const (
	maxCurrencyPairsPerRequest = 350
	// DefaultCurrencies has the default minimum of FIAT values
	DefaultCurrencies = "USD,AUD,EUR,CNY"
	// DefaultCryptoCurrencies has the default minimum of crytpocurrency values
	DefaultCryptoCurrencies = "BTC,LTC,ETH,DOGE,DASH,XRP,XMR"
)

// Manager is the overarching type across this package
type Manager struct {
	FXRates map[string]float64

	FiatCurrencies   []string
	Cryptocurrencies []string

	MainFxProvider      string
	BackUpFxProviders   []string
	Verbose             bool
	FXProviders         *forexprovider.ForexProviders
	PairFormat          *config.CurrencyPairFormatConfig
	FiatDisplayCurrency string
	sync.Mutex
}

// NewManager takes in a CurrencyConfig and sets up relational currency provider
// packages then returns a pointer to a currency manager
func NewManager(currencyConfig config.CurrencyConfig) *Manager {
	var cryptocurrencies []string

	m := new(Manager)

	if currencyConfig.Cryptocurrencies == "" {
		cryptocurrencies = common.SplitStrings(DefaultCryptoCurrencies, ",")
	} else {
		cryptocurrencies = common.SplitStrings(currencyConfig.Cryptocurrencies, ",")
	}

	m.Cryptocurrencies = cryptocurrencies
	m.FiatCurrencies = common.SplitStrings(DefaultCurrencies, ",")
	m.PairFormat = currencyConfig.CurrencyPairFormat
	m.FXProviders = forexprovider.StartFXService(currencyConfig.ForexProviders)
	m.FXRates = make(map[string]float64)

	return m
}

func (m *Manager) StartDefault(exchangeConfigurations []config.ExchangeConfig) error {
	log.Println("Retrieving enabled configuration currencies...")
	err := m.RetrieveConfigCurrencyPairs(exchangeConfigurations, true)
	if err != nil {
		return err
	}
	log.Println("Supported fiat: ", m.GetEnabledFiatCurrencies())
	log.Println("Supported crypto: ", m.GetEnabledCryptocurrencies())

	return m.SeedCurrencyData()
}

func (m *Manager) GetEnabledFiatCurrencies() []string {
	return m.FiatCurrencies
}

func (m *Manager) GetEnabledCryptocurrencies() []string {
	return m.Cryptocurrencies
}

// SeedCurrencyData returns rates correlated with suported currencies
func (m *Manager) SeedCurrencyData() error {
	m.Lock()
	defer m.Unlock()

	newRates, err := m.FXProviders.GetCurrencyData("", "")
	if err != nil {
		return err
	}

	for key, value := range newRates {
		// if len(key) < 5 {
		// 	rates[base+key] = value
		// 	continue
		// }
		m.FXRates[key] = value
	}
	return nil
}

func (m *Manager) GetExchangeRates() map[string]float64 {
	m.Lock()
	defer m.Unlock()
	return m.FXRates
}

// IsDefaultCurrency checks if the currency passed in matches the default fiat
// currency
func (m *Manager) IsDefaultCurrency(currency string) bool {
	m.Lock()
	defer m.Unlock()
	defaultCurrencies := common.SplitStrings(DefaultCurrencies, ",")
	return common.StringDataCompare(defaultCurrencies, common.StringToUpper(currency))
}

// IsDefaultCryptocurrency checks if the currency passed in matches the default
// cryptocurrency
func (m *Manager) IsDefaultCryptocurrency(currency string) bool {
	m.Lock()
	defer m.Unlock()
	cryptoCurrencies := common.SplitStrings(DefaultCryptoCurrencies, ",")
	return common.StringDataCompare(cryptoCurrencies, common.StringToUpper(currency))
}

// IsFiatCurrency checks if the currency passed is an enabled fiat currency
func (m *Manager) IsFiatCurrency(currency string) bool {
	m.Lock()
	defer m.Unlock()
	if len(m.FiatCurrencies) == 0 {
		log.Fatal("Currency IsFiatCurrency() error BaseCurrencies string variable not populated")
	}
	return common.StringDataCompare(m.FiatCurrencies, common.StringToUpper(currency))
}

// IsCryptocurrency checks if the currency passed is an enabled CRYPTO currency.
func (m *Manager) IsCryptocurrency(currency string) bool {
	m.Lock()
	defer m.Unlock()
	if len(m.Cryptocurrencies) == 0 {
		log.Fatal("Currency IsCryptocurrency() CryptoCurrencies string variable not populated")
	}
	return common.StringDataCompare(m.Cryptocurrencies, common.StringToUpper(currency))
}

// IsCryptoPair checks to see if the pair is a crypto pair e.g. BTCLTC
func (m *Manager) IsCryptoPair(p pair.CurrencyPair) bool {
	m.Lock()
	defer m.Unlock()
	return m.IsCryptocurrency(p.FirstCurrency.String()) &&
		m.IsCryptocurrency(p.SecondCurrency.String())
}

// IsCryptoFiatPair checks to see if the pair is a crypto fiat pair e.g. BTCUSD
func (m *Manager) IsCryptoFiatPair(p pair.CurrencyPair) bool {
	m.Lock()
	defer m.Unlock()
	return m.IsCryptocurrency(p.FirstCurrency.String()) &&
		!m.IsCryptocurrency(p.SecondCurrency.String()) ||
		!m.IsCryptocurrency(p.FirstCurrency.String()) &&
			m.IsCryptocurrency(p.SecondCurrency.String())
}

// IsFiatPair checks to see if the pair is a fiat pair e.g. EURUSD
func (m *Manager) IsFiatPair(p pair.CurrencyPair) bool {
	m.Lock()
	defer m.Unlock()
	return m.IsFiatCurrency(p.FirstCurrency.String()) &&
		m.IsFiatCurrency(p.SecondCurrency.String())
}

// Update updates the local crypto currency or base currency store
func (m *Manager) Update(input []string, cryptos bool) {
	m.Lock()
	defer m.Unlock()
	for x := range input {
		if cryptos {
			if !common.StringDataCompare(m.Cryptocurrencies, input[x]) {
				m.Cryptocurrencies = append(m.Cryptocurrencies, common.StringToUpper(input[x]))
			}
		} else {
			if !common.StringDataCompare(m.FiatCurrencies, input[x]) {
				m.FiatCurrencies = append(m.FiatCurrencies, common.StringToUpper(input[x]))
			}
		}
	}
}

// ConvertCurrency for example converts $1 USD to the equivalent Japanese Yen
// or vice versa.
func (m *Manager) ConvertCurrency(amount float64, from, to string) (float64, error) {
	m.Lock()
	defer m.Unlock()
	from = common.StringToUpper(from)
	to = common.StringToUpper(to)

	if from == to {
		return amount, nil
	}

	if from == "RUR" {
		from = "RUB"
	}

	if to == "RUR" {
		to = "RUB"
	}

	conversionRate, ok := m.FXRates[from+to]
	if !ok {
		conversionRate, ok = m.FXRates[to+from]
		if !ok {
			return 0, errors.New("cannot find currencypair")
		}
		newConversionRate := 1 / conversionRate
		return amount * newConversionRate, nil
	}
	return amount * conversionRate, nil
}

// RetrieveConfigCurrencyPairs splits, assigns and verifies enabled currency
// pairs either cryptoCurrencies or fiatCurrencies
func (m *Manager) RetrieveConfigCurrencyPairs(exchanges []config.ExchangeConfig, enabledOnly bool) error {
	cryptoCurrencies := m.Cryptocurrencies
	fiatCurrencies := common.SplitStrings(DefaultCurrencies, ",")

	for x := range exchanges {
		if !exchanges[x].Enabled && enabledOnly {
			continue
		}

		baseCurrencies := common.SplitStrings(exchanges[x].BaseCurrencies, ",")
		for y := range baseCurrencies {
			if !common.StringDataCompare(fiatCurrencies, common.StringToUpper(baseCurrencies[y])) {
				fiatCurrencies = append(fiatCurrencies, common.StringToUpper(baseCurrencies[y]))
			}
		}
	}

	for x := range exchanges {
		var pairs []pair.CurrencyPair

		if !exchanges[x].Enabled && enabledOnly {
			pairs = func(c config.ExchangeConfig) []pair.CurrencyPair {
				return pair.FormatPairs(common.SplitStrings(c.EnabledPairs, ","),
					c.ConfigCurrencyPairFormat.Delimiter,
					c.ConfigCurrencyPairFormat.Index)
			}(exchanges[x])
		} else {
			pairs = func(c config.ExchangeConfig) []pair.CurrencyPair {
				return pair.FormatPairs(common.SplitStrings(c.AvailablePairs, ","),
					c.ConfigCurrencyPairFormat.Delimiter,
					c.ConfigCurrencyPairFormat.Index)
			}(exchanges[x])
		}

		for y := range pairs {
			if !common.StringDataCompare(fiatCurrencies, pairs[y].FirstCurrency.Upper().String()) &&
				!common.StringDataCompare(cryptoCurrencies, pairs[y].FirstCurrency.Upper().String()) {
				cryptoCurrencies = append(cryptoCurrencies, pairs[y].FirstCurrency.Upper().String())
			}

			if !common.StringDataCompare(fiatCurrencies, pairs[y].SecondCurrency.Upper().String()) &&
				!common.StringDataCompare(cryptoCurrencies, pairs[y].SecondCurrency.Upper().String()) {
				cryptoCurrencies = append(cryptoCurrencies, pairs[y].SecondCurrency.Upper().String())
			}
		}
	}

	m.Update(fiatCurrencies, false)
	m.Update(cryptoCurrencies, true)
	return nil
}
