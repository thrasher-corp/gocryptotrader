package engine

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/utils"
)

// GetAvailableExchanges returns a list of enabled exchanges
func GetAvailableExchanges() []string {
	var enExchanges []string
	for x := range Bot.Config.Exchanges {
		if Bot.Config.Exchanges[x].Enabled {
			enExchanges = append(enExchanges, Bot.Config.Exchanges[x].Name)
		}
	}
	return enExchanges
}

// GetAllAvailablePairs returns a list of all available pairs on either enabled
// or disabled exchanges
func GetAllAvailablePairs(enabledExchangesOnly bool, assetType assets.AssetType) currency.Pairs {
	var pairList currency.Pairs
	for x := range Bot.Config.Exchanges {
		if enabledExchangesOnly && !Bot.Config.Exchanges[x].Enabled {
			continue
		}

		exchName := Bot.Config.Exchanges[x].Name
		pairs, err := Bot.Config.GetAvailablePairs(exchName, assetType)
		if err != nil {
			continue
		}

		for y := range pairs {
			if pairList.Contains(pairs[y], false) {
				continue
			}
			pairList = append(pairList, pairs[y])
		}
	}
	return pairList
}

// GetSpecificAvailablePairs returns a list of supported pairs based on specific
// parameters
func GetSpecificAvailablePairs(enabledExchangesOnly, fiatPairs, includeUSDT, cryptoPairs bool, assetType assets.AssetType) currency.Pairs {
	var pairList currency.Pairs
	supportedPairs := GetAllAvailablePairs(enabledExchangesOnly, assetType)

	for x := range supportedPairs {
		if fiatPairs {
			if supportedPairs[x].IsCryptoFiatPair() &&
				!supportedPairs[x].ContainsCurrency(currency.USDT) ||
				(includeUSDT &&
					supportedPairs[x].ContainsCurrency(currency.USDT) &&
					supportedPairs[x].IsCryptoPair()) {
				if pairList.Contains(supportedPairs[x], false) {
					continue
				}
				pairList = append(pairList, supportedPairs[x])
			}
		}
		if cryptoPairs {
			if supportedPairs[x].IsCryptoPair() {
				if pairList.Contains(supportedPairs[x], false) {
					continue
				}
				pairList = append(pairList, supportedPairs[x])
			}
		}
	}
	return pairList
}

// IsRelatablePairs checks to see if the two pairs are relatable
func IsRelatablePairs(p1, p2 currency.Pair, includeUSDT bool) bool {
	if p1.EqualIncludeReciprocal(p2) {
		return true
	}

	var relatablePairs = GetRelatableCurrencies(p1, true, includeUSDT)
	if p1.IsCryptoFiatPair() {
		for x := range relatablePairs {
			relatablePairs = append(relatablePairs,
				GetRelatableFiatCurrencies(relatablePairs[x])...)
		}
	}
	return relatablePairs.Contains(p2, false)
}

// MapCurrenciesByExchange returns a list of currency pairs mapped to an
// exchange
func MapCurrenciesByExchange(p currency.Pairs, enabledExchangesOnly bool, assetType assets.AssetType) map[string]currency.Pairs {
	currencyExchange := make(map[string]currency.Pairs)
	for x := range p {
		for y := range Bot.Config.Exchanges {
			if enabledExchangesOnly && !Bot.Config.Exchanges[y].Enabled {
				continue
			}
			exchName := Bot.Config.Exchanges[y].Name
			success, err := Bot.Config.SupportsPair(exchName, p[x], assetType)
			if err != nil || !success {
				continue
			}

			result, ok := currencyExchange[exchName]
			if !ok {
				var pairs []currency.Pair
				pairs = append(pairs, p[x])
				currencyExchange[exchName] = pairs
			} else {
				if result.Contains(p[x], false) {
					continue
				}
				result = append(result, p[x])
				currencyExchange[exchName] = result
			}
		}
	}
	return currencyExchange
}

// GetExchangeNamesByCurrency returns a list of exchanges supporting
// a currency pair based on whether the exchange is enabled or not
func GetExchangeNamesByCurrency(p currency.Pair, enabled bool, assetType assets.AssetType) []string {
	var exchanges []string
	for x := range Bot.Config.Exchanges {
		if enabled != Bot.Config.Exchanges[x].Enabled {
			continue
		}

		exchName := Bot.Config.Exchanges[x].Name
		success, err := Bot.Config.SupportsPair(exchName, p, assetType)
		if err != nil {
			continue
		}

		if success {
			exchanges = append(exchanges, exchName)
		}
	}
	return exchanges
}

// GetRelatableCryptocurrencies returns a list of currency pairs if it can find
// any relatable currencies (e.g ETHBTC -> ETHLTC -> ETHUSDT -> ETHREP)
func GetRelatableCryptocurrencies(p currency.Pair) currency.Pairs {
	var pairs currency.Pairs
	cryptocurrencies := currency.GetCryptocurrencies()

	for x := range cryptocurrencies {
		newPair := currency.NewPair(p.Base, cryptocurrencies[x])
		if newPair.IsInvalid() {
			continue
		}

		if newPair.Base.Upper() == p.Base.Upper() &&
			newPair.Quote.Upper() == p.Quote.Upper() {
			continue
		}

		if pairs.Contains(newPair, false) {
			continue
		}
		pairs = append(pairs, newPair)
	}
	return pairs
}

// GetRelatableFiatCurrencies returns a list of currency pairs if it can find
// any relatable currencies (e.g ETHUSD -> ETHAUD -> ETHGBP -> ETHJPY)
func GetRelatableFiatCurrencies(p currency.Pair) currency.Pairs {
	var pairs currency.Pairs
	fiatCurrencies := currency.GetFiatCurrencies()

	for x := range fiatCurrencies {
		newPair := currency.NewPair(p.Base, fiatCurrencies[x])
		if newPair.Base.Upper() == newPair.Quote.Upper() {
			continue
		}

		if newPair.Base.Upper() == p.Base.Upper() &&
			newPair.Quote.Upper() == p.Quote.Upper() {
			continue
		}

		if pairs.Contains(newPair, false) {
			continue
		}
		pairs = append(pairs, newPair)
	}
	return pairs
}

// GetRelatableCurrencies returns a list of currency pairs if it can find
// any relatable currencies (e.g BTCUSD -> BTC USDT -> XBT USDT -> XBT USD)
// incOrig includes the supplied pair if desired
func GetRelatableCurrencies(p currency.Pair, incOrig, incUSDT bool) currency.Pairs {
	var pairs currency.Pairs

	addPair := func(p currency.Pair) {
		if pairs.Contains(p, true) {
			return
		}
		pairs = append(pairs, p)
	}

	buildPairs := func(p currency.Pair, incOrig bool) {
		if incOrig {
			addPair(p)
		}

		first, ok := currency.GetTranslation(p.Base)
		if ok {
			addPair(currency.NewPair(first, p.Quote))

			var second currency.Code
			second, ok = currency.GetTranslation(p.Quote)
			if ok {
				addPair(currency.NewPair(first, second))
			}
		}

		second, ok := currency.GetTranslation(p.Quote)
		if ok {
			addPair(currency.NewPair(p.Base, second))
		}
	}

	buildPairs(p, incOrig)
	buildPairs(p.Swap(), incOrig)

	if !incUSDT {
		pairs = pairs.RemovePairsByFilter(currency.USDT)
	}

	return pairs
}

// GetSpecificOrderbook returns a specific orderbook given the currency,
// exchangeName and assetType
func GetSpecificOrderbook(p currency.Pair, exchangeName string, assetType assets.AssetType) (orderbook.Base, error) {
	var specificOrderbook orderbook.Base
	var err error
	for x := range Bot.Exchanges {
		if Bot.Exchanges[x] != nil {
			if Bot.Exchanges[x].GetName() == exchangeName {
				specificOrderbook, err = Bot.Exchanges[x].FetchOrderbook(
					p,
					assetType,
				)
				break
			}
		}
	}
	return specificOrderbook, err
}

// GetSpecificTicker returns a specific ticker given the currency,
// exchangeName and assetType
func GetSpecificTicker(p currency.Pair, exchangeName string, assetType assets.AssetType) (ticker.Price, error) {
	var specificTicker ticker.Price
	var err error
	for x := range Bot.Exchanges {
		if Bot.Exchanges[x] != nil {
			if Bot.Exchanges[x].GetName() == exchangeName {
				specificTicker, err = Bot.Exchanges[x].FetchTicker(
					p,
					assetType,
				)
				break
			}
		}
	}
	return specificTicker, err
}

// GetCollatedExchangeAccountInfoByCoin collates individual exchange account
// information and turns into into a map string of
// exchange.AccountCurrencyInfo
func GetCollatedExchangeAccountInfoByCoin(exchAccounts []exchange.AccountInfo) map[currency.Code]exchange.AccountCurrencyInfo {
	result := make(map[currency.Code]exchange.AccountCurrencyInfo)
	for _, accounts := range exchAccounts {
		for _, account := range accounts.Accounts {
			for _, accountCurrencyInfo := range account.Currencies {
				currencyName := accountCurrencyInfo.CurrencyName
				avail := accountCurrencyInfo.TotalValue
				onHold := accountCurrencyInfo.Hold

				info, ok := result[currencyName]
				if !ok {
					accountInfo := exchange.AccountCurrencyInfo{CurrencyName: currencyName, Hold: onHold, TotalValue: avail}
					result[currencyName] = accountInfo
				} else {
					info.Hold += onHold
					info.TotalValue += avail
					result[currencyName] = info
				}
			}
		}
	}
	return result
}

// GetAccountCurrencyInfoByExchangeName returns info for an exchange
func GetAccountCurrencyInfoByExchangeName(accounts []exchange.AccountInfo, exchangeName string) (exchange.AccountInfo, error) {
	for i := 0; i < len(accounts); i++ {
		if accounts[i].Exchange == exchangeName {
			return accounts[i], nil
		}
	}
	return exchange.AccountInfo{}, ErrExchangeNotFound
}

// GetExchangeHighestPriceByCurrencyPair returns the exchange with the highest
// price for a given currency pair and asset type
func GetExchangeHighestPriceByCurrencyPair(p currency.Pair, assetType assets.AssetType) (string, error) {
	result := stats.SortExchangesByPrice(p, assetType, true)
	if len(result) == 0 {
		return "", fmt.Errorf("no stats for supplied currency pair and asset type")
	}

	return result[0].Exchange, nil
}

// GetExchangeLowestPriceByCurrencyPair returns the exchange with the lowest
// price for a given currency pair and asset type
func GetExchangeLowestPriceByCurrencyPair(p currency.Pair, assetType assets.AssetType) (string, error) {
	result := stats.SortExchangesByPrice(p, assetType, false)
	if len(result) == 0 {
		return "", fmt.Errorf("no stats for supplied currency pair and asset type")
	}

	return result[0].Exchange, nil
}

// SeedExchangeAccountInfo seeds account info
func SeedExchangeAccountInfo(data []exchange.AccountInfo) {
	if len(data) == 0 {
		return
	}

	port := portfolio.GetPortfolio()

	for _, exchangeData := range data {
		exchangeName := exchangeData.Exchange

		var currencies []exchange.AccountCurrencyInfo
		for _, account := range exchangeData.Accounts {
			for _, info := range account.Currencies {

				var update bool
				for i := range currencies {
					if info.CurrencyName == currencies[i].CurrencyName {
						currencies[i].Hold += info.Hold
						currencies[i].TotalValue += info.TotalValue
						update = true
					}
				}

				if update {
					continue
				}

				currencies = append(currencies, exchange.AccountCurrencyInfo{
					CurrencyName: info.CurrencyName,
					TotalValue:   info.TotalValue,
					Hold:         info.Hold,
				})
			}
		}

		for _, total := range currencies {
			currencyName := total.CurrencyName
			total := total.TotalValue

			if !port.ExchangeAddressExists(exchangeName, currencyName) {
				if total <= 0 {
					continue
				}

				log.Debugf("Portfolio: Adding new exchange address: %s, %s, %f, %s\n",
					exchangeName,
					currencyName,
					total,
					portfolio.PortfolioAddressExchange)

				port.Addresses = append(
					port.Addresses,
					portfolio.Address{Address: exchangeName,
						CoinType:    currencyName,
						Balance:     total,
						Description: portfolio.PortfolioAddressExchange})

			} else {
				if total <= 0 {
					log.Debugf("Portfolio: Removing %s %s entry.\n",
						exchangeName,
						currencyName)

					port.RemoveExchangeAddress(exchangeName, currencyName)
				} else {
					balance, ok := port.GetAddressBalance(exchangeName,
						portfolio.PortfolioAddressExchange,
						currencyName)

					if !ok {
						continue
					}

					if balance != total {
						log.Debugf("Portfolio: Updating %s %s entry with balance %f.\n",
							exchangeName,
							currencyName,
							total)

						port.UpdateExchangeAddressBalance(exchangeName,
							currencyName,
							total)
					}
				}
			}
		}
	}
}

// GetCryptocurrenciesByExchange returns a list of cryptocurrencies the exchange supports
func GetCryptocurrenciesByExchange(exchangeName string, enabledExchangesOnly, enabledPairs bool, assetType assets.AssetType) ([]string, error) {
	var cryptocurrencies []string
	for x := range Bot.Config.Exchanges {
		if Bot.Config.Exchanges[x].Name != exchangeName {
			continue
		}
		if enabledExchangesOnly && !Bot.Config.Exchanges[x].Enabled {
			continue
		}

		exchName := Bot.Config.Exchanges[x].Name
		var pairs []currency.Pair
		var err error

		if enabledPairs {
			pairs, err = Bot.Config.GetEnabledPairs(exchName, assetType)
			if err != nil {
				return nil, err
			}
		} else {
			pairs, err = Bot.Config.GetAvailablePairs(exchName, assetType)
			if err != nil {
				return nil, err
			}
		}

		for y := range pairs {
			if pairs[y].Base.IsCryptocurrency() &&
				!common.StringDataCompareInsensitive(cryptocurrencies, pairs[y].Base.String()) {
				cryptocurrencies = append(cryptocurrencies, pairs[y].Base.String())
			}

			if pairs[y].Quote.IsCryptocurrency() &&
				!common.StringDataCompareInsensitive(cryptocurrencies, pairs[y].Quote.String()) {
				cryptocurrencies = append(cryptocurrencies, pairs[y].Quote.String())
			}
		}

	}
	return cryptocurrencies, nil
}

// GetExchangeCryptocurrencyDepositAddress returns the cryptocurrency deposit address for a particular
// exchange
func GetExchangeCryptocurrencyDepositAddress(exchName string, item currency.Code) (string, error) {
	exch := GetExchangeByName(exchName)
	if exch == nil {
		return "", ErrExchangeNotFound
	}

	return exch.GetDepositAddress(item, "")
}

// GetExchangeCryptocurrencyDepositAddresses obtains an exchanges deposit cryptocurrency list
func GetExchangeCryptocurrencyDepositAddresses() map[string]map[string]string {
	result := make(map[string]map[string]string)

	for x := range Bot.Exchanges {
		if !Bot.Exchanges[x].IsEnabled() {
			continue
		}
		exchName := Bot.Exchanges[x].GetName()

		if !Bot.Exchanges[x].GetAuthenticatedAPISupport() {
			if Bot.Settings.Verbose {
				log.Debugf("GetExchangeCryptocurrencyDepositAddresses: Skippping %s due to disabled authenticated API support.", exchName)
			}
			continue
		}

		cryptoCurrencies, err := GetCryptocurrenciesByExchange(exchName, true, true, assets.AssetTypeSpot)
		if err != nil {
			log.Debugf("%s failed to get cryptocurrency deposit addresses. Err: %s", exchName, err)
			continue
		}

		cryptoAddr := make(map[string]string)
		for y := range cryptoCurrencies {
			cryptocurrency := cryptoCurrencies[y]
			depositAddr, err := Bot.Exchanges[x].GetDepositAddress(currency.NewCode(cryptocurrency), "")
			if err != nil {
				log.Debugf("%s failed to get cryptocurrency deposit addresses. Err: %s", exchName, err)
				continue
			}
			cryptoAddr[cryptocurrency] = depositAddr
		}
		result[exchName] = cryptoAddr
	}

	return result
}

// GetDepositAddressByExchange returns a deposit address for the specified exchange and cryptocurrency
// if it exists
func GetDepositAddressByExchange(exchName string, currencyItem currency.Code) string {
	for x, y := range Bot.CryptocurrencyDepositAddresses {
		if exchName == x {
			addr, ok := y[currencyItem.String()]
			if ok {
				return addr
			}
		}
	}
	return ""
}

// GetDepositAddressesByExchange returns a list of cryptocurrency addresses for the specified
// exchange if they exist
func GetDepositAddressesByExchange(exchName string) map[string]string {
	for x, y := range Bot.CryptocurrencyDepositAddresses {
		if exchName == x {
			return y
		}
	}
	return nil
}

// WithdrawCryptocurrencyFundsByExchange withdraws the desired cryptocurrency and amount to a desired cryptocurrency address
func WithdrawCryptocurrencyFundsByExchange(exchName string) (string, error) {
	exch := GetExchangeByName(exchName)
	if exch == nil {
		return "", ErrExchangeNotFound
	}

	// TO-DO: FILL
	return exch.WithdrawCryptocurrencyFunds(&exchange.WithdrawRequest{})
}

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func FormatCurrency(p currency.Pair) currency.Pair {
	return p.Format(Bot.Config.Currency.CurrencyPairFormat.Delimiter,
		Bot.Config.Currency.CurrencyPairFormat.Uppercase)
}

// GetExchanges returns a list of loaded exchanges
func GetExchanges(enabled bool) []string {
	var exchanges []string
	for x := range Bot.Exchanges {
		if Bot.Exchanges[x].IsEnabled() && enabled {
			exchanges = append(exchanges, Bot.Exchanges[x].GetName())
			continue
		}
		exchanges = append(exchanges, Bot.Exchanges[x].GetName())
	}
	return exchanges
}

// GetAllActiveTickers returns all enabled exchange tickers
func GetAllActiveTickers() []EnabledExchangeCurrencies {
	var tickerData []EnabledExchangeCurrencies

	for _, exch := range Bot.Exchanges {
		if !exch.IsEnabled() {
			continue
		}

		assets := exch.GetAssetTypes()
		exchName := exch.GetName()
		var exchangeTicker EnabledExchangeCurrencies
		exchangeTicker.ExchangeName = exchName

		for y := range assets {
			currencies := exch.GetEnabledPairs(assets[y])
			for z := range currencies {
				tp, err := exch.FetchTicker(currencies[z], assets[y])
				if err != nil {
					log.Debugf("Exchange %s failed to retrieve %s ticker. Err: %s", exchName,
						currencies[z].String(),
						err)
					continue
				}
				exchangeTicker.ExchangeValues = append(exchangeTicker.ExchangeValues, tp)
			}
			tickerData = append(tickerData, exchangeTicker)
		}
	}
	return tickerData
}

// GetAllEnabledExchangeAccountInfo returns all the current enabled exchanges
func GetAllEnabledExchangeAccountInfo() AllEnabledExchangeAccounts {
	var response AllEnabledExchangeAccounts
	for _, individualBot := range Bot.Exchanges {
		if individualBot != nil && individualBot.IsEnabled() {
			if !individualBot.GetAuthenticatedAPISupport() {
				if Bot.Settings.Verbose {
					log.Debugf("GetAllEnabledExchangeAccountInfo: Skippping %s due to disabled authenticated API support.", individualBot.GetName())
				}
				continue
			}
			individualExchange, err := individualBot.GetAccountInfo()
			if err != nil {
				log.Debugf("Error encountered retrieving exchange account info for %s. Error %s",
					individualBot.GetName(), err)
				continue
			}
			response.Data = append(response.Data, individualExchange)
		}
	}
	return response
}

func checkCerts() error {
	targetDir := utils.GetTLSDir(Bot.Settings.DataDir)
	_, err := os.Stat(targetDir)
	if os.IsNotExist(err) {
		err := common.CreateDir(targetDir)
		if err != nil {
			return err
		}
		return genCert(targetDir)
	}

	log.Debugf("gRPC TLS certs directory already exists, will use them.")
	return nil
}

func genCert(targetDir string) error {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ecdsa private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 365)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %s", err)
	}

	host, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %s", err)
	}

	dnsNames := []string{host}
	if host != "localhost" {
		dnsNames = append(dnsNames, "localhost")
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"gocryptotrader"},
			CommonName:   host,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,

		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
		DNSNames: dnsNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %s", err)
	}

	certData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if certData == nil {
		return fmt.Errorf("cert data is nil")
	}

	b, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("failed to marshal ECDSA private key: %s", err)
	}

	keyData := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	if keyData == nil {
		return fmt.Errorf("key pem data is nil")
	}

	err = common.WriteFile(filepath.Join(targetDir, "key.pem"), keyData)
	if err != nil {
		return fmt.Errorf("failed to write key.pem file %s", err)
	}

	err = common.WriteFile(filepath.Join(targetDir, "cert.pem"), certData)
	if err != nil {
		return fmt.Errorf("failed to write cert.pem file %s", err)
	}

	log.Debugf("TLS key.pem and cert.pem files written to %s", targetDir)
	return nil
}
