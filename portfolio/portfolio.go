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

	// PortfolioAddressExchange is a label for an exchange address
	PortfolioAddressExchange = "Exchange"
	// PortfolioAddressPersonal is a label for a personal/offline address
	PortfolioAddressPersonal = "Personal"
)

// Portfolio is variable store holding an array of portfolioAddress
var Portfolio Base

// Verbose allows for debug output when sending an http request
var Verbose bool

// GetEthereumBalance single or multiple address information as
// EtherchainBalanceResponse
func GetEthereumBalance(address string) (EthplorerResponse, error) {
	valid, _ := common.IsValidCryptoAddress(address, "eth")
	if !valid {
		return EthplorerResponse{}, errors.New("not an Ethereum address")
	}

	urlPath := fmt.Sprintf(
		"%s/%s/%s?apiKey=freekey", ethplorerAPIURL, ethplorerAddressInfo, address,
	)

	result := EthplorerResponse{}
	return result, common.SendHTTPGetRequest(urlPath, true, Verbose, &result)
}

// GetCryptoIDAddress queries CryptoID for an address balance for a
// specified cryptocurrency
func GetCryptoIDAddress(address string, coinType currency.Code) (float64, error) {
	ok, err := common.IsValidCryptoAddress(address, coinType.String())
	if !ok || err != nil {
		return 0, errors.New("invalid address")
	}

	var result interface{}
	url := fmt.Sprintf("%s/%s/api.dws?q=getbalance&a=%s",
		cryptoIDAPIURL,
		coinType.Lower(),
		address)

	err = common.SendHTTPGetRequest(url, true, Verbose, &result)
	if err != nil {
		return 0, err
	}
	return result.(float64), nil
}

// GetRippleBalance returns the value for a ripple address
func GetRippleBalance(address string) (float64, error) {
	var result XRPScanAccount
	err := common.SendHTTPGetRequest(xrpScanAPIURL+address, true, Verbose, &result)
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
func (p *Base) GetAddressBalance(address, description string, coinType currency.Code) (float64, bool) {
	for x := range p.Addresses {
		if p.Addresses[x].Address == address &&
			p.Addresses[x].Description == description &&
			p.Addresses[x].CoinType == coinType {
			return p.Addresses[x].Balance, true
		}
	}
	return 0, false
}

// ExchangeExists checks to see if an exchange exists in the portfolio base
func (p *Base) ExchangeExists(exchangeName string) bool {
	for x := range p.Addresses {
		if p.Addresses[x].Address == exchangeName {
			return true
		}
	}
	return false
}

// AddressExists checks to see if there is an address associated with the
// portfolio base
func (p *Base) AddressExists(address string) bool {
	for x := range p.Addresses {
		if p.Addresses[x].Address == address {
			return true
		}
	}
	return false
}

// ExchangeAddressExists checks to see if there is an exchange address
// associated with the portfolio base
func (p *Base) ExchangeAddressExists(exchangeName string, coinType currency.Code) bool {
	for x := range p.Addresses {
		if p.Addresses[x].Address == exchangeName && p.Addresses[x].CoinType == coinType {
			return true
		}
	}
	return false
}

// AddExchangeAddress adds an exchange address to the portfolio base
func (p *Base) AddExchangeAddress(exchangeName string, coinType currency.Code, balance float64) {
	if p.ExchangeAddressExists(exchangeName, coinType) {
		p.UpdateExchangeAddressBalance(exchangeName, coinType, balance)
	} else {
		p.Addresses = append(
			p.Addresses, Address{Address: exchangeName, CoinType: coinType,
				Balance: balance, Description: PortfolioAddressExchange},
		)
	}
}

// UpdateAddressBalance updates the portfolio base balance
func (p *Base) UpdateAddressBalance(address string, amount float64) {
	for x := range p.Addresses {
		if p.Addresses[x].Address == address {
			p.Addresses[x].Balance = amount
		}
	}
}

// RemoveExchangeAddress removes an exchange address from the portfolio.
func (p *Base) RemoveExchangeAddress(exchangeName string, coinType currency.Code) {
	for x := range p.Addresses {
		if p.Addresses[x].Address == exchangeName && p.Addresses[x].CoinType == coinType {
			p.Addresses = append(p.Addresses[:x], p.Addresses[x+1:]...)
			return
		}
	}
}

// UpdateExchangeAddressBalance updates the portfolio balance when checked
// against correct exchangeName and coinType.
func (p *Base) UpdateExchangeAddressBalance(exchangeName string, coinType currency.Code, balance float64) {
	for x := range p.Addresses {
		if p.Addresses[x].Address == exchangeName && p.Addresses[x].CoinType == coinType {
			p.Addresses[x].Balance = balance
		}
	}
}

// AddAddress adds an address to the portfolio base
func (p *Base) AddAddress(address, description string, coinType currency.Code, balance float64) error {
	if address == "" {
		return errors.New("address is empty")
	}

	if coinType.String() == "" {
		return errors.New("coin type is empty")
	}

	if description == PortfolioAddressExchange {
		p.AddExchangeAddress(address, coinType, balance)
	}
	if !p.AddressExists(address) {
		p.Addresses = append(
			p.Addresses, Address{Address: address, CoinType: coinType,
				Balance: balance, Description: description},
		)
	} else {
		if balance <= 0 {
			p.RemoveAddress(address, description, coinType)
		} else {
			p.UpdateAddressBalance(address, balance)
		}
	}
	return nil
}

// RemoveAddress removes an address when checked against the correct address and
// coinType
func (p *Base) RemoveAddress(address, description string, coinType currency.Code) error {
	if address == "" {
		return errors.New("address is empty")
	}

	if coinType.String() == "" {
		return errors.New("coin type is empty")
	}

	for x := range p.Addresses {
		if p.Addresses[x].Address == address &&
			p.Addresses[x].CoinType == coinType &&
			p.Addresses[x].Description == description {
			p.Addresses = append(p.Addresses[:x], p.Addresses[x+1:]...)
			return nil
		}
	}

	return errors.New("portfolio item does not exist")
}

// UpdatePortfolio adds to the portfolio addresses by coin type
func (p *Base) UpdatePortfolio(addresses []string, coinType currency.Code) error {
	if strings.Contains(strings.Join(addresses, ","), PortfolioAddressExchange) ||
		strings.Contains(strings.Join(addresses, ","), PortfolioAddressPersonal) {
		return nil
	}

	switch coinType {
	case currency.ETH:
		for x := range addresses {
			result, err := GetEthereumBalance(addresses[x])
			if err != nil {
				return err
			}

			if result.Error.Message != "" {
				return errors.New(result.Error.Message)
			}

			err = p.AddAddress(addresses[x],
				PortfolioAddressPersonal,
				coinType,
				result.ETH.Balance)
			if err != nil {
				return err
			}
		}
	case currency.XRP:
		for x := range addresses {
			result, err := GetRippleBalance(addresses[x])
			if err != nil {
				return err
			}
			err = p.AddAddress(addresses[x],
				PortfolioAddressPersonal,
				coinType,
				result)
			if err != nil {
				return err
			}
		}
	default:
		for x := range addresses {
			result, err := GetCryptoIDAddress(addresses[x], coinType)
			if err != nil {
				return err
			}
			err = p.AddAddress(addresses[x],
				PortfolioAddressPersonal,
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
func (p *Base) GetPortfolioByExchange(exchangeName string) map[currency.Code]float64 {
	result := make(map[currency.Code]float64)
	for x := range p.Addresses {
		if strings.Contains(p.Addresses[x].Address, exchangeName) {
			result[p.Addresses[x].CoinType] = p.Addresses[x].Balance
		}
	}
	return result
}

// GetExchangePortfolio returns current portfolio base information
func (p *Base) GetExchangePortfolio() map[currency.Code]float64 {
	result := make(map[currency.Code]float64)
	for _, x := range p.Addresses {
		if x.Description != PortfolioAddressExchange {
			continue
		}
		balance, ok := result[x.CoinType]
		if !ok {
			result[x.CoinType] = x.Balance
		} else {
			result[x.CoinType] = x.Balance + balance
		}
	}
	return result
}

// GetPersonalPortfolio returns current portfolio base information
func (p *Base) GetPersonalPortfolio() map[currency.Code]float64 {
	result := make(map[currency.Code]float64)
	for _, x := range p.Addresses {
		if x.Description == PortfolioAddressExchange {
			continue
		}
		balance, ok := result[x.CoinType]
		if !ok {
			result[x.CoinType] = x.Balance
		} else {
			result[x.CoinType] = x.Balance + balance
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
func (p *Base) GetPortfolioSummary() Summary {
	personalHoldings := p.GetPersonalPortfolio()
	exchangeHoldings := p.GetExchangePortfolio()
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
	for _, x := range p.Addresses {
		if x.Description == PortfolioAddressExchange {
			if !common.StringDataCompare(portfolioExchanges, x.Address) {
				portfolioExchanges = append(portfolioExchanges, x.Address)
			}
		}
	}

	exchangeSummary := make(map[string]map[currency.Code]OnlineCoinSummary)
	for x := range portfolioExchanges {
		exchgName := portfolioExchanges[x]
		result := p.GetPortfolioByExchange(exchgName)

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
	for _, x := range p.Addresses {
		if x.Description != PortfolioAddressExchange {
			coinSummary := OfflineCoinSummary{
				Address: x.Address,
				Balance: x.Balance,
				Percentage: getPercentageSpecific(x.Balance, x.CoinType,
					totalCoins),
			}
			result, ok := offlineSummary[x.CoinType]
			if !ok {
				offlineSummary[x.CoinType] = append(offlineSummary[x.CoinType],
					coinSummary)
			} else {
				result = append(result, coinSummary)
				offlineSummary[x.CoinType] = result
			}
		}
	}
	portfolioOutput.OfflineSummary = offlineSummary
	return portfolioOutput
}

// GetPortfolioGroupedCoin returns portfolio base information grouped by coin
func (p *Base) GetPortfolioGroupedCoin() map[currency.Code][]string {
	result := make(map[currency.Code][]string)
	for _, x := range p.Addresses {
		if strings.Contains(x.Description, PortfolioAddressExchange) {
			continue
		}
		result[x.CoinType] = append(result[x.CoinType], x.Address)
	}
	return result
}

// Seed appends a portfolio base object with another base portfolio
// addresses
func (p *Base) Seed(port Base) {
	p.Addresses = port.Addresses
}

// StartPortfolioWatcher observes the portfolio object
func StartPortfolioWatcher() {
	addrCount := len(Portfolio.Addresses)
	log.Debugf(log.PortfolioMgr,
		"PortfolioWatcher started: Have %d entries in portfolio.\n", addrCount,
	)
	for {
		data := Portfolio.GetPortfolioGroupedCoin()
		for key, value := range data {
			err := Portfolio.UpdatePortfolio(value, key)
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

// GetPortfolio returns a pointer to the portfolio base
func GetPortfolio() *Base {
	return &Portfolio
}

// IsExchangeSupported checks if exchange is supported by portfolio address
func IsExchangeSupported(exchange, address string) (ret bool) {
	for x := range Portfolio.Addresses {
		if Portfolio.Addresses[x].Address != address {
			continue
		}
		exchangeList := strings.Split(Portfolio.Addresses[x].SupportedExchanges, ",")
		return common.StringDataContainsInsensitive(exchangeList, exchange)
	}
	return
}

// IsColdStorage checks if address is a cold storage wallet
func IsColdStorage(address string) (ret bool) {
	for x := range Portfolio.Addresses {
		if Portfolio.Addresses[x].Address != address {
			continue
		}
		return Portfolio.Addresses[x].ColdStorage
	}
	return
}

// IsWhiteListed checks if address is whitelisted for withdraw transfers
func IsWhiteListed(address string) (ret bool) {
	for x := range Portfolio.Addresses {
		if Portfolio.Addresses[x].Address != address {
			continue
		}
		return Portfolio.Addresses[x].WhiteListed
	}
	return
}
