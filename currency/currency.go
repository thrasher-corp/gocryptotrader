package currency

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/coinmarketcap"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	// DefaultBaseCurrency is the base currency used for conversion
	DefaultBaseCurrency = USD
)

// Manager is the overarching type across this package
var (
	// DefaultCurrencies has the default minimum of FIAT values
	DefaultCurrencies = Currencies{USD, AUD, EUR, CNY}

	// DefaultCryptoCurrencies has the default minimum of crytpocurrency values
	DefaultCryptoCurrencies = Currencies{BTC, LTC, ETH, DOGE, DASH, XRP, XMR}

	FXRates map[string]float64

	FiatCurrencies   Currencies
	CryptoCurrencies Currencies

	BaseCurrency Code
	FXProviders  *forexprovider.ForexProviders

	CryptocurrencyProvider *coinmarketcap.Coinmarketcap
	TotalCryptocurrencies  []Data
	TotalExchanges         []Data
)

// Currencies define a range of supported currency codes
type Currencies []Code

// NewCurrencyListFromCurrencies returns a Currencies object from strings
// NOTE: need to add lookup table
func NewCurrencyListFromCurrencies(currencies []string) Currencies {
	var list Currencies
	for _, c := range currencies {
		if c == "" {
			continue
		}
		list = append(list, Code(c))
	}
	return list
}

// String returns an array of currency strings
func (c Currencies) String() []string {
	var list []string
	for _, d := range c {
		list = append(list, d.String())
	}
	return list
}

// Join returns a comma serparated string
func (c Currencies) Join() string {
	return common.JoinStrings(c.String(), ",")
}

// UnmarshalJSON comforms type to the umarshaler interface
func (c *Currencies) UnmarshalJSON(d []byte) error {
	var configCurrencies string
	err := common.JSONDecode(d, &configCurrencies)
	if err != nil {
		return err
	}

	var allTheCurrencies Currencies
	for _, data := range common.SplitStrings(configCurrencies, ",") {
		allTheCurrencies = append(allTheCurrencies, Code(data))
	}

	*c = allTheCurrencies
	return nil
}

// MarshalJSON conforms type to the marshaler interface
func (c Currencies) MarshalJSON() ([]byte, error) {
	return common.JSONEncode(c.Join())
}

// SetDefaults sets the default currency provider and settings for
// currency conversion used outside of the bot setting
func SetDefaults() {
	FXRates = make(map[string]float64)
	BaseCurrency = DefaultBaseCurrency

	FXProviders = forexprovider.NewDefaultFXProvider()
	err := SeedCurrencyData(DefaultCurrencies.Join())
	if err != nil {
		log.Errorf("Failed to seed currency data. Err: %s", err)
		return
	}
}

// SeedCurrencyData returns rates correlated with suported currencies
func SeedCurrencyData(currencies string) error {
	if FXRates == nil {
		FXRates = make(map[string]float64)
	}

	if FXProviders == nil {
		FXProviders = forexprovider.NewDefaultFXProvider()
	}

	newRates, err := FXProviders.GetCurrencyData(BaseCurrency.String(), currencies)
	if err != nil {
		return err
	}

	for key, value := range newRates {
		FXRates[key] = value
	}

	return nil
}

// GetExchangeRates returns the currency exchange rates
func GetExchangeRates() map[string]float64 {
	return FXRates
}

// IsDefaultCurrency checks if the currency passed in matches the default fiat
// currency
func IsDefaultCurrency(currency string) bool {
	return common.StringDataCompare(DefaultCurrencies.String(),
		common.StringToUpper(currency))
}

// IsDefaultCryptocurrency checks if the currency passed in matches the default
// cryptocurrency
func IsDefaultCryptocurrency(currency string) bool {
	return common.StringDataCompare(DefaultCryptoCurrencies.String(),
		common.StringToUpper(currency))
}

// IsFiatCurrency checks if the currency passed is an enabled fiat currency
func IsFiatCurrency(currency string) bool {
	return common.StringDataCompare(FiatCurrencies.String(),
		common.StringToUpper(currency))
}

// IsCryptocurrency checks if the currency passed is an enabled CRYPTO currency.
func IsCryptocurrency(currency string) bool {
	return common.StringDataCompare(CryptoCurrencies.String(),
		common.StringToUpper(currency))
}

// IsCryptoPair checks to see if the pair is a crypto pair e.g. BTCLTC
func IsCryptoPair(p Pair) bool {
	return IsCryptocurrency(p.Base.String()) &&
		IsCryptocurrency(p.Quote.String())
}

// IsCryptoFiatPair checks to see if the pair is a crypto fiat pair e.g. BTCUSD
func IsCryptoFiatPair(p Pair) bool {
	return IsCryptocurrency(p.Base.String()) &&
		!IsCryptocurrency(p.Quote.String()) ||
		!IsCryptocurrency(p.Base.String()) &&
			IsCryptocurrency(p.Quote.String())
}

// IsFiatPair checks to see if the pair is a fiat pair e.g. EURUSD
func IsFiatPair(p Pair) bool {
	return IsFiatCurrency(p.Base.String()) &&
		IsFiatCurrency(p.Quote.String())
}

// Update updates the local crypto currency or base currency store
func Update(input []string, cryptos bool) {
	for x := range input {
		if cryptos {
			if !common.StringDataCompare(CryptoCurrencies.String(), input[x]) {
				CryptoCurrencies = append(CryptoCurrencies,
					Code(common.StringToUpper(input[x])))
			}
		} else {
			if !common.StringDataCompare(FiatCurrencies.String(), input[x]) {
				FiatCurrencies = append(FiatCurrencies,
					Code(common.StringToUpper(input[x])))
			}
		}
	}
}

func extractBaseCurrency() string {
	for k := range FXRates {
		return k[0:3]
	}
	return ""
}

// ConvertCurrency for example converts $1 USD to the equivalent Japanese Yen
// or vice versa.
func ConvertCurrency(amount float64, from, to string) (float64, error) {
	if FXProviders == nil {
		SetDefaults()
	}

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

	if len(FXRates) == 0 {
		SeedCurrencyData(from + "," + to)
	}

	// Need to extract the base currency to see if we actually got it from the Forex API
	// Fixer free API sets the base currency to EUR
	baseCurr := extractBaseCurrency()

	var resultFrom float64
	var resultTo float64

	// check to see if we're converting from the base currency
	if to == baseCurr {
		var ok bool
		resultFrom, ok = FXRates[baseCurr+from]
		if !ok {
			return 0, fmt.Errorf("currency conversion failed. Unable to find %s in currency map [%s -> %s]", from, from, to)
		}
		return amount / resultFrom, nil
	}

	// Check to see if we're converting from the base currency
	if from == baseCurr {
		var ok bool
		resultTo, ok = FXRates[baseCurr+to]
		if !ok {
			return 0, fmt.Errorf("currency conversion failed. Unable to find %s in currency map [%s -> %s]", to, from, to)
		}
		return resultTo * amount, nil
	}

	// Otherwise convert to base currency, then to the target currency
	resultFrom, ok := FXRates[baseCurr+from]
	if !ok {
		return 0, fmt.Errorf("currency conversion failed. Unable to find %s in currency map [%s -> %s]", from, from, to)
	}

	converted := amount / resultFrom
	resultTo, ok = FXRates[baseCurr+to]
	if !ok {
		return 0, fmt.Errorf("currency conversion failed. Unable to find %s in currency map [%s -> %s]", to, from, to)
	}

	return converted * resultTo, nil
}

// Data defines information pertaining to exchange or a cryptocurrency from
// coinmarketcap
type Data struct {
	ID          int
	Name        string
	Symbol      string `json:",omitempty"`
	Slug        string
	Active      bool
	LastUpdated time.Time
}

// SeedCryptocurrencyMarketData seeds cryptocurrency market data
func SeedCryptocurrencyMarketData(settings coinmarketcap.Settings) error {
	if !settings.Enabled {
		return errors.New("not enabled please set in config.json with apikey and account levels")
	}

	if CryptocurrencyProvider == nil {
		err := setupCryptoProvider(settings)
		if err != nil {
			return err
		}
	}

	cryptoData, err := CryptocurrencyProvider.GetCryptocurrencyIDMap()
	if err != nil {
		return err
	}

	for x := range cryptoData {
		var active bool
		if cryptoData[x].IsActive == 1 {
			active = true
		}

		TotalCryptocurrencies = append(TotalCryptocurrencies, Data{
			ID:          cryptoData[x].ID,
			Name:        cryptoData[x].Name,
			Symbol:      cryptoData[x].Symbol,
			Slug:        cryptoData[x].Slug,
			Active:      active,
			LastUpdated: time.Now(),
		})
	}

	return nil
}

// SeedExchangeMarketData seeds exchange market data
func SeedExchangeMarketData(settings coinmarketcap.Settings) error {
	if !settings.Enabled {
		return errors.New("not enabled please set in config.json with apikey and account levels")
	}

	if CryptocurrencyProvider == nil {
		err := setupCryptoProvider(settings)
		if err != nil {
			return err
		}
	}

	exchangeData, err := CryptocurrencyProvider.GetExchangeMap(0, 0)
	if err != nil {
		return err
	}

	for _, data := range exchangeData {
		var active bool
		if data.IsActive == 1 {
			active = true
		}

		TotalExchanges = append(TotalExchanges, Data{
			ID:          data.ID,
			Name:        data.Name,
			Slug:        data.Slug,
			Active:      active,
			LastUpdated: time.Now(),
		})
	}

	return nil
}

func setupCryptoProvider(settings coinmarketcap.Settings) error {
	if settings.APIkey == "" ||
		settings.APIkey == "key" ||
		settings.AccountPlan == "" ||
		settings.AccountPlan == "accountPlan" {
		return errors.New("currencyprovider error api key or plan not set in config.json")
	}

	CryptocurrencyProvider = new(coinmarketcap.Coinmarketcap)
	CryptocurrencyProvider.SetDefaults()
	CryptocurrencyProvider.Setup(settings)

	return nil
}

// GetTotalMarketCryptocurrencies returns the total seeded market
// cryptocurrencies
func GetTotalMarketCryptocurrencies() []Data {
	return TotalCryptocurrencies
}

// GetTotalMarketExchanges returns the total seeded market exchanges
func GetTotalMarketExchanges() []Data {
	return TotalExchanges
}
