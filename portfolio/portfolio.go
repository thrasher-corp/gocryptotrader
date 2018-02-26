package portfolio

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	cryptoIDAPIURL = "https://chainz.cryptoid.info"

	etherchainAPIURL          = "https://etherchain.org/api"
	etherchainAccountMultiple = "account/multiple"

	ethplorerAPIURL      = "https://api.ethplorer.io"
	ethplorerAddressInfo = "getAddressInfo"

	// PortfolioAddressExchange is a label for an exchange address
	PortfolioAddressExchange = "Exchange"
	// PortfolioAddressPersonal is a label for a personal/offline address
	PortfolioAddressPersonal = "Personal"
)

// Portfolio is variable store holding an array of portfolioAddress
var Portfolio Base

// Base holds the portfolio base addresses
type Base struct {
	Addresses []Address
}

// Address sub type holding address information for portfolio
type Address struct {
	Address     string
	CoinType    string
	Balance     float64
	Description string
}

// EtherchainBalanceResponse holds JSON incoming and outgoing data for
// Etherchain
type EtherchainBalanceResponse struct {
	Status int `json:"status"`
	Data   []struct {
		Address   string      `json:"address"`
		Balance   float64     `json:"balance"`
		Nonce     interface{} `json:"nonce"`
		Code      string      `json:"code"`
		Name      interface{} `json:"name"`
		Storage   interface{} `json:"storage"`
		FirstSeen interface{} `json:"firstSeen"`
	} `json:"data"`
}

// EthplorerResponse holds JSON address data for Ethplorer
type EthplorerResponse struct {
	Address string `json:"address"`
	ETH     struct {
		Balance  float64 `json:"balance"`
		TotalIn  float64 `json:"totalIn"`
		TotalOut float64 `json:"totalOut"`
	} `json:"ETH"`
	CountTxs     int `json:"countTxs"`
	ContractInfo struct {
		CreatorAddress  string `json:"creatorAddress"`
		TransactionHash string `json:"transactionHash"`
		Timestamp       int    `json:"timestamp"`
	} `json:"contractInfo"`
	TokenInfo struct {
		Address        string `json:"address"`
		Name           string `json:"name"`
		Decimals       int    `json:"decimals"`
		Symbol         string `json:"symbol"`
		TotalSupply    string `json:"totalSupply"`
		Owner          string `json:"owner"`
		LastUpdated    int    `json:"lastUpdated"`
		TotalIn        int64  `json:"totalIn"`
		TotalOut       int64  `json:"totalOut"`
		IssuancesCount int    `json:"issuancesCount"`
		HoldersCount   int    `json:"holdersCount"`
		Image          string `json:"image"`
		Description    string `json:"description"`
		Price          struct {
			Rate     int    `json:"rate"`
			Diff     int    `json:"diff"`
			Ts       int    `json:"ts"`
			Currency string `json:"currency"`
		} `json:"price"`
	} `json:"tokenInfo"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ExchangeAccountInfo : Generic type to hold each exchange's holdings in all
// enabled currencies
type ExchangeAccountInfo struct {
	ExchangeName string
	Currencies   []ExchangeAccountCurrencyInfo
}

// ExchangeAccountCurrencyInfo : Sub type to store currency name and value
type ExchangeAccountCurrencyInfo struct {
	CurrencyName string
	TotalValue   float64
	Hold         float64
}

// GetEthereumBalance single or multiple address information as
// EtherchainBalanceResponse
func GetEthereumBalance(address string) (EthplorerResponse, error) {
	valid, _ := common.IsValidCryptoAddress(address, "eth")
	if !valid {
		return EthplorerResponse{}, errors.New("Not an ethereum address")
	}

	url := fmt.Sprintf(
		"%s/%s/%s?apiKey=freekey", ethplorerAPIURL, ethplorerAddressInfo, address,
	)
	result := EthplorerResponse{}
	err := common.SendHTTPGetRequest(url, true, false, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetCryptoIDAddress queries CryptoID for an address balance for a
// specified cryptocurrency
func GetCryptoIDAddress(address string, coinType string) (float64, error) {
	ok, err := common.IsValidCryptoAddress(address, coinType)
	if !ok || err != nil {
		return 0, errors.New("invalid address")
	}

	var result interface{}
	url := fmt.Sprintf("%s/%s/api.dws?q=getbalance&a=%s", cryptoIDAPIURL, common.StringToLower(coinType), address)
	err = common.SendHTTPGetRequest(url, true, false, &result)
	if err != nil {
		return 0, err
	}
	return result.(float64), nil
}

// GetAddressBalance acceses the portfolio base and returns the balance by passed
// in address, coin type and description
func (p *Base) GetAddressBalance(address, coinType, description string) (float64, bool) {
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
func (p *Base) ExchangeAddressExists(exchangeName, coinType string) bool {
	for x := range p.Addresses {
		if p.Addresses[x].Address == exchangeName && p.Addresses[x].CoinType == coinType {
			return true
		}
	}
	return false
}

// AddExchangeAddress adds an exchange address to the portfolio base
func (p *Base) AddExchangeAddress(exchangeName, coinType string, balance float64) {
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
func (p *Base) RemoveExchangeAddress(exchangeName, coinType string) {
	for x := range p.Addresses {
		if p.Addresses[x].Address == exchangeName && p.Addresses[x].CoinType == coinType {
			p.Addresses = append(p.Addresses[:x], p.Addresses[x+1:]...)
			return
		}
	}
}

// UpdateExchangeAddressBalance updates the portfolio balance when checked
// against correct exchangeName and coinType.
func (p *Base) UpdateExchangeAddressBalance(exchangeName, coinType string, balance float64) {
	for x := range p.Addresses {
		if p.Addresses[x].Address == exchangeName && p.Addresses[x].CoinType == coinType {
			p.Addresses[x].Balance = balance
		}
	}
}

// AddAddress adds an address to the portfolio base
func (p *Base) AddAddress(address, coinType, description string, balance float64) {
	if description == PortfolioAddressExchange {
		p.AddExchangeAddress(address, coinType, balance)
		return
	}
	if !p.AddressExists(address) {
		p.Addresses = append(
			p.Addresses, Address{Address: address, CoinType: coinType,
				Balance: balance, Description: description},
		)
	} else {
		if balance <= 0 {
			p.RemoveAddress(address, coinType, description)
		} else {
			p.UpdateAddressBalance(address, balance)
		}
	}
}

// RemoveAddress removes an address when checked against the correct address and
// coinType
func (p *Base) RemoveAddress(address, coinType, description string) {
	for x := range p.Addresses {
		if p.Addresses[x].Address == address && p.Addresses[x].CoinType == coinType && p.Addresses[x].Description == description {
			p.Addresses = append(p.Addresses[:x], p.Addresses[x+1:]...)
			return
		}
	}
}

// UpdatePortfolio adds to the portfolio addresses by coin type
func (p *Base) UpdatePortfolio(addresses []string, coinType string) bool {
	if common.StringContains(common.JoinStrings(addresses, ","), PortfolioAddressExchange) || common.StringContains(common.JoinStrings(addresses, ","), PortfolioAddressPersonal) {
		return true
	}

	errors := 0
	if coinType == "ETH" {
		for x := range addresses {
			result, err := GetEthereumBalance(addresses[x])
			if err != nil {
				errors++
				continue
			}

			if result.Error.Message != "" {
				errors++
				continue
			}
			p.AddAddress(addresses[x], coinType, PortfolioAddressPersonal, result.ETH.Balance)
		}
		if errors > 0 {
			return false
		}
		return true
	}
	for x := range addresses {
		result, err := GetCryptoIDAddress(addresses[x], coinType)
		if err != nil {
			return false
		}
		p.AddAddress(addresses[x], coinType, PortfolioAddressPersonal, result)
	}
	return true
}

// GetPortfolioByExchange returns currency portfolio amount by exchange
func (p *Base) GetPortfolioByExchange(exchangeName string) map[string]float64 {
	result := make(map[string]float64)
	for x := range p.Addresses {
		if common.StringContains(p.Addresses[x].Address, exchangeName) {
			result[p.Addresses[x].CoinType] = p.Addresses[x].Balance
		}
	}
	return result
}

// GetExchangePortfolio returns current portfolio base information
func (p *Base) GetExchangePortfolio() map[string]float64 {
	result := make(map[string]float64)
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
func (p *Base) GetPersonalPortfolio() map[string]float64 {
	result := make(map[string]float64)
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
func getPercentage(input map[string]float64, target string, totals map[string]float64) float64 {
	subtotal, _ := input[target]
	total, _ := totals[target]
	percentage := (subtotal / total) * 100 / 1
	return percentage
}

// getPercentage returns the percentage a specific value of a target coin amount
// against the total coin amount.
func getPercentageSpecific(input float64, target string, totals map[string]float64) float64 {
	total, _ := totals[target]
	percentage := (input / total) * 100 / 1
	return percentage
}

// Coin stores a coin type, balance, address and percentage relative to the total
// amount.
type Coin struct {
	Coin       string  `json:"coin"`
	Balance    float64 `json:"balance"`
	Address    string  `json:"address,omitempty"`
	Percentage float64 `json:"percentage,omitempty"`
}

// OfflineCoinSummary stores a coin types address, balance and percentage
// relative to the total amount.
type OfflineCoinSummary struct {
	Address    string  `json:"address"`
	Balance    float64 `json:"balance"`
	Percentage float64 `json:"percentage,omitempty"`
}

// OnlineCoinSummary stores a coin types balance and percentage relative to the
// total amount.
type OnlineCoinSummary struct {
	Balance    float64 `json:"balance"`
	Percentage float64 `json:"percentage,omitempty"`
}

// Summary Stores the entire portfolio summary
type Summary struct {
	Totals         []Coin                                  `json:"coin_totals"`
	Offline        []Coin                                  `json:"coins_offline"`
	OfflineSummary map[string][]OfflineCoinSummary         `json:"offline_summary"`
	Online         []Coin                                  `json:"coins_online"`
	OnlineSummary  map[string]map[string]OnlineCoinSummary `json:"online_summary"`
}

// GetPortfolioSummary returns the complete portfolio summary, showing
// coin totals, offline and online summaries with their relative percentages.
func (p *Base) GetPortfolioSummary() Summary {
	personalHoldings := p.GetPersonalPortfolio()
	exchangeHoldings := p.GetExchangePortfolio()
	totalCoins := make(map[string]float64)

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

	exchangeSummary := make(map[string]map[string]OnlineCoinSummary)
	for x := range portfolioExchanges {
		exchgName := portfolioExchanges[x]
		result := p.GetPortfolioByExchange(exchgName)

		coinSummary := make(map[string]OnlineCoinSummary)
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

	offlineSummary := make(map[string][]OfflineCoinSummary)
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
func (p *Base) GetPortfolioGroupedCoin() map[string][]string {
	result := make(map[string][]string)
	for _, x := range p.Addresses {
		if common.StringContains(x.Description, PortfolioAddressExchange) {
			continue
		}
		result[x.CoinType] = append(result[x.CoinType], x.Address)
	}
	return result
}

// SeedPortfolio appends a portfolio base object with another base portfolio
// addresses
func (p *Base) SeedPortfolio(port Base) {
	p.Addresses = port.Addresses
}

// StartPortfolioWatcher observes the portfolio object
func StartPortfolioWatcher() {
	addrCount := len(Portfolio.Addresses)
	log.Printf(
		"PortfolioWatcher started: Have %d entries in portfolio.\n", addrCount,
	)
	for {
		data := Portfolio.GetPortfolioGroupedCoin()
		for key, value := range data {
			success := Portfolio.UpdatePortfolio(value, key)
			if success {
				log.Printf(
					"PortfolioWatcher: Successfully updated address balance for %s address(es) %s\n",
					key, value,
				)
			}
		}
		time.Sleep(time.Minute * 10)
	}
}

// GetPortfolio returns a pointer to the portfolio base
func GetPortfolio() *Base {
	return &Portfolio
}
