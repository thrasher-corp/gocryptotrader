package portfolio

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/log"
	"golang.org/x/time/rate"
)

const (
	cryptoIDAPIURL = "https://chainz.cryptoid.info"
	xrpScanAPIURL  = "https://api.xrpscan.com/api/v1/account/"

	ethplorerAPIURL      = "https://api.ethplorer.io"
	ethplorerAddressInfo = "getAddressInfo"

	// ExchangeAddress is a label for an exchange address
	ExchangeAddress = "Exchange"
	// PersonalAddress is a label for a personal/offline address
	PersonalAddress = "Personal"

	defaultAPIKey   = "Key"
	defaultInterval = 10 * time.Second
)

var (
	errProviderNotFound        = errors.New("provider not found")
	errProviderNotEnabled      = errors.New("provider not enabled")
	errProviderAPIKeyNotSet    = errors.New("provider API key not set")
	errPortfolioItemNotFound   = errors.New("portfolio item not found")
	errNoPortfolioItemsToWatch = errors.New("no portfolio items to watch")
)

// GetEthereumAddressBalance fetches Ethereum address balance for a given address
func (b *Base) GetEthereumAddressBalance(ctx context.Context, address string) (float64, error) {
	if err := common.IsValidCryptoAddress(address, "eth"); err != nil {
		return 0, err
	}

	apiKey := "freekey"
	if p, ok := b.Providers.GetProvider("ethplorer"); ok && p.APIKey != "" {
		apiKey = p.APIKey
	}

	urlPath := ethplorerAPIURL + "/" + ethplorerAddressInfo + "/" + address + "?apiKey=" + apiKey

	contents, err := common.SendHTTPRequest(ctx, http.MethodGet, urlPath, nil, nil, b.Verbose)
	if err != nil {
		return 0, err
	}

	var result EthplorerResponse
	if err := json.Unmarshal(contents, &result); err != nil {
		return 0, err
	}

	return result.ETH.Balance, nil
}

// GetCryptoIDAddressBalance fetches the address balance for a specified cryptocurrency
func (b *Base) GetCryptoIDAddressBalance(ctx context.Context, address string, coinType currency.Code) (float64, error) {
	if err := common.IsValidCryptoAddress(address, coinType.String()); err != nil {
		return 0, err
	}

	p, ok := b.Providers.GetProvider("cryptoid")
	if !ok {
		return 0, fmt.Errorf("cryptoid: %w", errProviderNotFound)
	}

	if p.APIKey == "" || p.APIKey == defaultAPIKey {
		return 0, fmt.Errorf("cryptoid: %w", errProviderAPIKeyNotSet)
	}

	b.cryptoIDLimiterOnce.Do(func() {
		b.cryptoIDLimiter = rate.NewLimiter(rate.Every(10*time.Second), 1)
	})

	if err := b.cryptoIDLimiter.Wait(ctx); err != nil {
		return 0, fmt.Errorf("rate limiter wait error: %w", err)
	}

	urlPath := cryptoIDAPIURL + "/" + coinType.Lower().String() + "/api.dws?q=getbalance&a=" + address + "&key=" + p.APIKey

	contents, err := common.SendHTTPRequest(ctx, http.MethodGet, urlPath, nil, nil, b.Verbose)
	if err != nil {
		return 0, err
	}

	var result float64
	return result, json.Unmarshal(contents, &result)
}

// GetRippleAddressBalance returns the value for a ripple address
func (b *Base) GetRippleAddressBalance(ctx context.Context, address string) (float64, error) {
	contents, err := common.SendHTTPRequest(ctx, http.MethodGet, xrpScanAPIURL+address, nil, nil, b.Verbose)
	if err != nil {
		return 0, err
	}

	var result XRPScanAccount
	if err := json.Unmarshal(contents, &result); err != nil {
		return 0, err
	}

	return result.XRPBalance, nil
}

// GetAddressBalance accesses the portfolio base and returns the balance by passed
// in address, coin type and description
func (b *Base) GetAddressBalance(address, description string, coinType currency.Code) (float64, bool) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	idx := slices.IndexFunc(b.Addresses, func(a Address) bool {
		return a.Address == address && a.Description == description && a.CoinType.Equal(coinType)
	})
	if idx == -1 {
		return 0, false
	}
	return b.Addresses[idx].Balance, true
}

// ExchangeExists checks to see if an exchange exists in the portfolio base
func (b *Base) ExchangeExists(exchangeName string) bool {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	return slices.ContainsFunc(b.Addresses, func(a Address) bool {
		return a.Address == exchangeName && a.Description == ExchangeAddress
	})
}

// AddressExists checks to see if there is an address associated with the portfolio base
func (b *Base) AddressExists(address string) bool {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	return slices.ContainsFunc(b.Addresses, func(a Address) bool {
		return a.Address == address
	})
}

// ExchangeAddressCoinExists checks to see if there is an exchange address
// associated with the portfolio base
func (b *Base) ExchangeAddressCoinExists(exchangeName string, coinType currency.Code) bool {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	return slices.ContainsFunc(b.Addresses, func(a Address) bool {
		return a.Address == exchangeName && a.CoinType.Equal(coinType) && a.Description == ExchangeAddress
	})
}

// AddExchangeAddress adds an exchange address to the portfolio base
func (b *Base) AddExchangeAddress(exchangeName string, coinType currency.Code, balance float64) {
	if b.ExchangeAddressCoinExists(exchangeName, coinType) {
		b.UpdateExchangeAddressBalance(exchangeName, coinType, balance)
		return
	}

	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.Addresses = append(b.Addresses, Address{
		Address:     exchangeName,
		CoinType:    coinType,
		Balance:     balance,
		Description: ExchangeAddress,
	})
}

// UpdateAddressBalance updates the portfolio base balance.
func (b *Base) UpdateAddressBalance(address string, amount float64) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	for x := range b.Addresses {
		if b.Addresses[x].Address == address {
			b.Addresses[x].Balance = amount
		}
	}
}

// RemoveExchangeAddress removes an exchange address from the portfolio.
func (b *Base) RemoveExchangeAddress(exchangeName string, coinType currency.Code) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.Addresses = slices.Clip(slices.DeleteFunc(b.Addresses, func(a Address) bool {
		return a.Address == exchangeName && a.CoinType.Equal(coinType)
	}))
}

// UpdateExchangeAddressBalance updates the portfolio balance when checked against correct exchangeName and coinType.
func (b *Base) UpdateExchangeAddressBalance(exchangeName string, coinType currency.Code, balance float64) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	for x := range b.Addresses {
		if b.Addresses[x].Address == exchangeName && b.Addresses[x].CoinType.Equal(coinType) {
			b.Addresses[x].Balance = balance
		}
	}
}

// AddAddress adds an address to the portfolio base or updates its balance if it already exists.
func (b *Base) AddAddress(address, description string, coinType currency.Code, balance float64) error {
	if address == "" {
		return common.ErrAddressIsEmptyOrInvalid
	}

	if coinType.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}

	if description == ExchangeAddress {
		b.AddExchangeAddress(address, coinType, balance)
		return nil
	}

	if !b.AddressExists(address) {
		b.mtx.Lock()
		defer b.mtx.Unlock()

		b.Addresses = append(b.Addresses, Address{
			Address:     address,
			CoinType:    coinType,
			Balance:     balance,
			Description: description,
		})
		return nil
	}

	b.UpdateAddressBalance(address, balance)
	return nil
}

// RemoveAddress removes an address when checked against the correct address and
// coinType
func (b *Base) RemoveAddress(address, description string, coinType currency.Code) error {
	if address == "" {
		return common.ErrAddressIsEmptyOrInvalid
	}

	if coinType.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}

	b.mtx.Lock()
	defer b.mtx.Unlock()

	idx := slices.IndexFunc(b.Addresses, func(a Address) bool {
		return a.Address == address && a.CoinType.Equal(coinType) && a.Description == description
	})
	if idx == -1 {
		return errPortfolioItemNotFound
	}
	b.Addresses = slices.Clip(slices.Delete(b.Addresses, idx, idx+1))
	return nil
}

// UpdatePortfolio adds to the portfolio addresses by coin type
func (b *Base) UpdatePortfolio(ctx context.Context, addresses []string, coinType currency.Code) error {
	if slices.ContainsFunc(addresses, func(a string) bool {
		return a == PersonalAddress || a == ExchangeAddress
	}) {
		return nil
	}

	var providerName string
	var getBalance func(ctx context.Context, address string) (float64, error)

	switch coinType {
	case currency.ETH:
		providerName = "Ethplorer"
		getBalance = b.GetEthereumAddressBalance
	case currency.XRP:
		providerName = "XRPScan"
		getBalance = b.GetRippleAddressBalance
	case currency.BTC, currency.LTC:
		providerName = "CryptoID"
		getBalance = func(ctx context.Context, address string) (float64, error) {
			return b.GetCryptoIDAddressBalance(ctx, address, coinType)
		}
	default:
		return fmt.Errorf("%w: %s", currency.ErrCurrencyNotSupported, coinType)
	}

	p, ok := b.Providers.GetProvider(providerName)
	if !ok {
		return fmt.Errorf("%w: %s", errProviderNotFound, providerName)
	}

	if !p.Enabled {
		return fmt.Errorf("%w: %s", errProviderNotEnabled, providerName)
	}

	if p.Name == "CryptoID" && (p.APIKey == "" || p.APIKey == defaultAPIKey) {
		return fmt.Errorf("%w: %s", errProviderAPIKeyNotSet, providerName)
	}

	var errs error
	for x := range addresses {
		balance, err := getBalance(ctx, addresses[x])
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("error getting balance for %s: %w", addresses[x], err))
			continue
		}

		if err := b.AddAddress(addresses[x], PersonalAddress, coinType, balance); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("error adding address %s: %w", addresses[x], err))
		}
	}
	return errs
}

// GetPortfolioByExchange returns currency portfolio amount by exchange
func (b *Base) GetPortfolioByExchange(exchangeName string) map[currency.Code]float64 {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	result := make(map[currency.Code]float64)
	for x := range b.Addresses {
		if strings.Contains(b.Addresses[x].Address, exchangeName) {
			result[b.Addresses[x].CoinType] = b.Addresses[x].Balance
		}
	}
	return result
}

// GetExchangePortfolio returns current portfolio base information
func (b *Base) GetExchangePortfolio() map[currency.Code]float64 {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	result := make(map[currency.Code]float64)
	for i := range b.Addresses {
		if b.Addresses[i].Description != ExchangeAddress {
			continue
		}
		balance, ok := result[b.Addresses[i].CoinType]
		if !ok {
			result[b.Addresses[i].CoinType] = b.Addresses[i].Balance
		} else {
			result[b.Addresses[i].CoinType] = b.Addresses[i].Balance + balance
		}
	}
	return result
}

// GetPersonalPortfolio returns current portfolio base information
func (b *Base) GetPersonalPortfolio() map[currency.Code]float64 {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	result := make(map[currency.Code]float64)
	for i := range b.Addresses {
		if strings.EqualFold(b.Addresses[i].Description, ExchangeAddress) {
			continue
		}
		balance, ok := result[b.Addresses[i].CoinType]
		if !ok {
			result[b.Addresses[i].CoinType] = b.Addresses[i].Balance
		} else {
			result[b.Addresses[i].CoinType] = b.Addresses[i].Balance + balance
		}
	}
	return result
}

// getPercentage returns the percentage of the target coin amount against the
// total coin amount.
func getPercentage(input map[currency.Code]float64, target currency.Code, totals map[currency.Code]float64) float64 {
	subtotal := input[target]
	total := totals[target]
	percentage := (subtotal / total) * 100 / 1
	return percentage
}

// getPercentageSpecific returns the percentage a specific value of a target coin amount
// against the total coin amount.
func getPercentageSpecific(input float64, target currency.Code, totals map[currency.Code]float64) float64 {
	total := totals[target]
	percentage := (input / total) * 100 / 1
	return percentage
}

// GetPortfolioSummary returns the complete portfolio summary, showing
// coin totals, offline and online summaries with their relative percentages.
func (b *Base) GetPortfolioSummary() Summary {
	personalHoldings := b.GetPersonalPortfolio()
	exchangeHoldings := b.GetExchangePortfolio()
	totalCoins := maps.Clone(personalHoldings)

	for x, y := range exchangeHoldings {
		balance, ok := totalCoins[x]
		if !ok {
			totalCoins[x] = y
		} else {
			totalCoins[x] = y + balance
		}
	}

	var portfolioOutput Summary
	for x, y := range totalCoins {
		coins := Coin{Coin: x, Balance: y}
		portfolioOutput.Totals = append(portfolioOutput.Totals, coins)
	}

	for x, y := range personalHoldings {
		coins := Coin{
			Coin:       x,
			Balance:    y,
			Percentage: getPercentage(personalHoldings, x, totalCoins),
		}
		portfolioOutput.Offline = append(portfolioOutput.Offline, coins)
	}

	for x, y := range exchangeHoldings {
		coins := Coin{
			Coin:       x,
			Balance:    y,
			Percentage: getPercentage(exchangeHoldings, x, totalCoins),
		}
		portfolioOutput.Online = append(portfolioOutput.Online, coins)
	}

	var portfolioExchanges []string
	for i := range b.Addresses {
		if strings.EqualFold(b.Addresses[i].Description, ExchangeAddress) {
			if !slices.Contains(portfolioExchanges, b.Addresses[i].Address) {
				portfolioExchanges = append(portfolioExchanges, b.Addresses[i].Address)
			}
		}
	}

	exchangeSummary := make(map[string]map[currency.Code]OnlineCoinSummary)
	for x := range portfolioExchanges {
		exchgName := portfolioExchanges[x]
		result := b.GetPortfolioByExchange(exchgName)

		coinSummary := make(map[currency.Code]OnlineCoinSummary)
		for y, z := range result {
			coinSum := OnlineCoinSummary{
				Balance:    z,
				Percentage: getPercentageSpecific(z, y, totalCoins),
			}
			coinSummary[y] = coinSum
		}
		exchangeSummary[exchgName] = coinSummary
	}
	portfolioOutput.OnlineSummary = exchangeSummary

	offlineSummary := make(map[currency.Code][]OfflineCoinSummary)
	for i := range b.Addresses {
		if !strings.EqualFold(b.Addresses[i].Description, ExchangeAddress) {
			coinSummary := OfflineCoinSummary{
				Address: b.Addresses[i].Address,
				Balance: b.Addresses[i].Balance,
				Percentage: getPercentageSpecific(b.Addresses[i].Balance, b.Addresses[i].CoinType,
					totalCoins),
			}
			result, ok := offlineSummary[b.Addresses[i].CoinType]
			if !ok {
				offlineSummary[b.Addresses[i].CoinType] = append(offlineSummary[b.Addresses[i].CoinType],
					coinSummary)
			} else {
				result = append(result, coinSummary)
				offlineSummary[b.Addresses[i].CoinType] = result
			}
		}
	}
	portfolioOutput.OfflineSummary = offlineSummary
	return portfolioOutput
}

// GetPortfolioAddressesGroupedByCoin returns portfolio addresses grouped by coin
func (b *Base) GetPortfolioAddressesGroupedByCoin() map[currency.Code][]string {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	result := make(map[currency.Code][]string)
	for i := range b.Addresses {
		if strings.EqualFold(b.Addresses[i].Description, ExchangeAddress) {
			continue
		}
		result[b.Addresses[i].CoinType] = append(result[b.Addresses[i].CoinType], b.Addresses[i].Address)
	}
	return result
}

// StartPortfolioWatcher observes the portfolio object
func (b *Base) StartPortfolioWatcher(ctx context.Context, interval time.Duration) error {
	if len(b.Addresses) == 0 {
		return errNoPortfolioItemsToWatch
	}

	if interval <= 0 {
		interval = defaultInterval
	}

	log.Infof(log.PortfolioMgr, "PortfolioWatcher started: Have %d entries in portfolio.\n", len(b.Addresses))

	updatePortfolio := func() {
		for key, value := range b.GetPortfolioAddressesGroupedByCoin() {
			if err := b.UpdatePortfolio(ctx, value, key); err != nil {
				log.Errorf(log.PortfolioMgr, "PortfolioWatcher: UpdatePortfolio error: %s for currency %s", err, key)
				continue
			}
			log.Debugf(log.PortfolioMgr, "PortfolioWatcher: Successfully updated address balance for %s", key)
		}
	}

	updatePortfolio()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debugf(log.PortfolioMgr, "PortfolioWatcher stopped: context cancelled")
			return ctx.Err()
		case <-ticker.C:
			updatePortfolio()
		}
	}
}

// IsExchangeSupported checks if exchange is supported by portfolio address
func (b *Base) IsExchangeSupported(exchange, address string) bool {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	return slices.ContainsFunc(b.Addresses, func(a Address) bool {
		if a.Address != address {
			return false
		}
		exchangeList := strings.Split(a.SupportedExchanges, ",")
		return common.StringSliceContainsInsensitive(exchangeList, exchange)
	})
}

// IsColdStorage checks if address is a cold storage wallet
func (b *Base) IsColdStorage(address string) bool {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	return slices.ContainsFunc(b.Addresses, func(a Address) bool {
		return a.Address == address && a.ColdStorage
	})
}

// IsWhiteListed checks if address is whitelisted for withdraw transfers
func (b *Base) IsWhiteListed(address string) bool {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	return slices.ContainsFunc(b.Addresses, func(a Address) bool {
		return a.Address == address && a.WhiteListed
	})
}

// GetProvider returns a provider by name
func (p providers) GetProvider(name string) (provider, bool) {
	for _, provider := range p {
		if strings.EqualFold(provider.Name, name) {
			return provider, true
		}
	}
	return provider{}, false
}
