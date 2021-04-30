package portfolio

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/log"
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
)

var errNotEthAddress = errors.New("not an Ethereum address")

// GetEthereumBalance single or multiple address information as
// EtherchainBalanceResponse
func (b *Base) GetEthereumBalance(address string) (EthplorerResponse, error) {
	valid, _ := common.IsValidCryptoAddress(address, "eth")
	if !valid {
		return EthplorerResponse{}, errNotEthAddress
	}

	urlPath := fmt.Sprintf(
		"%s/%s/%s?apiKey=freekey", ethplorerAPIURL, ethplorerAddressInfo, address,
	)

	result := EthplorerResponse{}
	return result, common.SendHTTPGetRequest(urlPath, true, b.Verbose, &result)
}

// GetCryptoIDAddress queries CryptoID for an address balance for a
// specified cryptocurrency
func (b *Base) GetCryptoIDAddress(address string, coinType currency.Code) (float64, error) {
	ok, err := common.IsValidCryptoAddress(address, coinType.String())
	if !ok || err != nil {
		return 0, errors.New("invalid address")
	}

	var result interface{}
	url := fmt.Sprintf("%s/%s/api.dws?q=getbalance&a=%s",
		cryptoIDAPIURL,
		coinType.Lower(),
		address)

	err = common.SendHTTPGetRequest(url, true, b.Verbose, &result)
	if err != nil {
		return 0, err
	}
	return result.(float64), nil
}

// GetRippleBalance returns the value for a ripple address
func (b *Base) GetRippleBalance(address string) (float64, error) {
	var result XRPScanAccount
	err := common.SendHTTPGetRequest(xrpScanAPIURL+address, true, b.Verbose, &result)
	if err != nil {
		return 0, err
	}

	if (result == XRPScanAccount{}) {
		return 0, errors.New("no balance info returned")
	}

	return result.XRPBalance, nil
}

// GetAddressBalance acceses the portfolio base and returns the balance by passed
// in address, coin type and description
func (b *Base) GetAddressBalance(address, description string, coinType currency.Code) (float64, bool) {
	for x := range b.Addresses {
		if b.Addresses[x].Address == address &&
			b.Addresses[x].Description == description &&
			b.Addresses[x].CoinType == coinType {
			return b.Addresses[x].Balance, true
		}
	}
	return 0, false
}

// ExchangeExists checks to see if an exchange exists in the portfolio base
func (b *Base) ExchangeExists(exchangeName string) bool {
	for x := range b.Addresses {
		if b.Addresses[x].Address == exchangeName {
			return true
		}
	}
	return false
}

// AddressExists checks to see if there is an address associated with the
// portfolio base
func (b *Base) AddressExists(address string) bool {
	for x := range b.Addresses {
		if b.Addresses[x].Address == address {
			return true
		}
	}
	return false
}

// ExchangeAddressExists checks to see if there is an exchange address
// associated with the portfolio base
func (b *Base) ExchangeAddressExists(exchangeName string, coinType currency.Code) bool {
	for x := range b.Addresses {
		if b.Addresses[x].Address == exchangeName && b.Addresses[x].CoinType == coinType {
			return true
		}
	}
	return false
}

// AddExchangeAddress adds an exchange address to the portfolio base
func (b *Base) AddExchangeAddress(exchangeName string, coinType currency.Code, balance float64) {
	if b.ExchangeAddressExists(exchangeName, coinType) {
		b.UpdateExchangeAddressBalance(exchangeName, coinType, balance)
	} else {
		b.Addresses = append(
			b.Addresses, Address{Address: exchangeName, CoinType: coinType,
				Balance: balance, Description: ExchangeAddress},
		)
	}
}

// UpdateAddressBalance updates the portfolio base balance
func (b *Base) UpdateAddressBalance(address string, amount float64) {
	for x := range b.Addresses {
		if b.Addresses[x].Address == address {
			b.Addresses[x].Balance = amount
		}
	}
}

// RemoveExchangeAddress removes an exchange address from the portfolio.
func (b *Base) RemoveExchangeAddress(exchangeName string, coinType currency.Code) {
	for x := range b.Addresses {
		if b.Addresses[x].Address == exchangeName && b.Addresses[x].CoinType == coinType {
			b.Addresses = append(b.Addresses[:x], b.Addresses[x+1:]...)
			return
		}
	}
}

// UpdateExchangeAddressBalance updates the portfolio balance when checked
// against correct exchangeName and coinType.
func (b *Base) UpdateExchangeAddressBalance(exchangeName string, coinType currency.Code, balance float64) {
	for x := range b.Addresses {
		if b.Addresses[x].Address == exchangeName && b.Addresses[x].CoinType == coinType {
			b.Addresses[x].Balance = balance
		}
	}
}

// AddAddress adds an address to the portfolio base
func (b *Base) AddAddress(address, description string, coinType currency.Code, balance float64) error {
	if address == "" {
		return errors.New("address is empty")
	}

	if coinType.String() == "" {
		return errors.New("coin type is empty")
	}

	if description == ExchangeAddress {
		b.AddExchangeAddress(address, coinType, balance)
	}
	if !b.AddressExists(address) {
		b.Addresses = append(
			b.Addresses, Address{Address: address, CoinType: coinType,
				Balance: balance, Description: description},
		)
	} else {
		if balance <= 0 {
			err := b.RemoveAddress(address, description, coinType)
			if err != nil {
				return err
			}
		} else {
			b.UpdateAddressBalance(address, balance)
		}
	}
	return nil
}

// RemoveAddress removes an address when checked against the correct address and
// coinType
func (b *Base) RemoveAddress(address, description string, coinType currency.Code) error {
	if address == "" {
		return errors.New("address is empty")
	}

	if coinType.String() == "" {
		return errors.New("coin type is empty")
	}

	for x := range b.Addresses {
		if b.Addresses[x].Address == address &&
			b.Addresses[x].CoinType == coinType &&
			b.Addresses[x].Description == description {
			b.Addresses = append(b.Addresses[:x], b.Addresses[x+1:]...)
			return nil
		}
	}

	return errors.New("portfolio item does not exist")
}

// UpdatePortfolio adds to the portfolio addresses by coin type
func (b *Base) UpdatePortfolio(addresses []string, coinType currency.Code) error {
	if strings.Contains(strings.Join(addresses, ","), ExchangeAddress) ||
		strings.Contains(strings.Join(addresses, ","), PersonalAddress) {
		return nil
	}

	switch coinType {
	case currency.ETH:
		for x := range addresses {
			result, err := b.GetEthereumBalance(addresses[x])
			if err != nil {
				return err
			}

			if result.Error.Message != "" {
				return errors.New(result.Error.Message)
			}

			err = b.AddAddress(addresses[x],
				PersonalAddress,
				coinType,
				result.ETH.Balance)
			if err != nil {
				return err
			}
		}
	case currency.XRP:
		for x := range addresses {
			result, err := b.GetRippleBalance(addresses[x])
			if err != nil {
				return err
			}
			err = b.AddAddress(addresses[x],
				PersonalAddress,
				coinType,
				result)
			if err != nil {
				return err
			}
		}
	default:
		for x := range addresses {
			result, err := b.GetCryptoIDAddress(addresses[x], coinType)
			if err != nil {
				return err
			}
			err = b.AddAddress(addresses[x],
				PersonalAddress,
				coinType,
				result)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetPortfolioByExchange returns currency portfolio amount by exchange
func (b *Base) GetPortfolioByExchange(exchangeName string) map[currency.Code]float64 {
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
	totalCoins := make(map[currency.Code]float64)

	for x, y := range personalHoldings {
		totalCoins[x] = y
	}

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
			if !common.StringDataCompare(portfolioExchanges, b.Addresses[i].Address) {
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

// GetPortfolioGroupedCoin returns portfolio base information grouped by coin
func (b *Base) GetPortfolioGroupedCoin() map[currency.Code][]string {
	result := make(map[currency.Code][]string)
	for i := range b.Addresses {
		if strings.EqualFold(b.Addresses[i].Description, ExchangeAddress) {
			continue
		}
		result[b.Addresses[i].CoinType] = append(result[b.Addresses[i].CoinType], b.Addresses[i].Address)
	}
	return result
}

// Seed appends a portfolio base object with another base portfolio
// addresses
func (b *Base) Seed(port Base) {
	b.Addresses = port.Addresses
}

// StartPortfolioWatcher observes the portfolio object
func (b *Base) StartPortfolioWatcher() {
	addrCount := len(b.Addresses)
	log.Debugf(log.PortfolioMgr,
		"PortfolioWatcher started: Have %d entries in portfolio.\n", addrCount,
	)
	for {
		data := b.GetPortfolioGroupedCoin()
		for key, value := range data {
			err := b.UpdatePortfolio(value, key)
			if err != nil {
				log.Errorf(log.PortfolioMgr,
					"PortfolioWatcher error %s for currency %s, val %v\n",
					err,
					key,
					value)
				continue
			}

			log.Debugf(log.PortfolioMgr,
				"PortfolioWatcher: Successfully updated address balance for %s address(es) %s\n",
				key,
				value)
		}
		time.Sleep(time.Minute * 10)
	}
}

// IsExchangeSupported checks if exchange is supported by portfolio address
func (b *Base) IsExchangeSupported(exchange, address string) (ret bool) {
	for x := range b.Addresses {
		if b.Addresses[x].Address != address {
			continue
		}
		exchangeList := strings.Split(b.Addresses[x].SupportedExchanges, ",")
		return common.StringDataContainsInsensitive(exchangeList, exchange)
	}
	return
}

// IsColdStorage checks if address is a cold storage wallet
func (b *Base) IsColdStorage(address string) bool {
	for x := range b.Addresses {
		if b.Addresses[x].Address != address {
			continue
		}
		return b.Addresses[x].ColdStorage
	}
	return false
}

// IsWhiteListed checks if address is whitelisted for withdraw transfers
func (b *Base) IsWhiteListed(address string) bool {
	for x := range b.Addresses {
		if b.Addresses[x].Address != address {
			continue
		}
		return b.Addresses[x].WhiteListed
	}
	return false
}
