package portfolio

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	blockrAPIURL         = "blockr.io/api"
	blockrAPIVersion     = "1"
	blockrAddressBalance = "address/balance"

	etherchainAPIURL          = "https://etherchain.org/api"
	etherchainAccountMultiple = "account/multiple"
	// PortfolioAddressExchange holds the current portfolio address
	PortfolioAddressExchange = "Exchange"
	portfolioAddressPersonal = "Personal"
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

// BlockrAddress holds JSON incoming and outgoing data for BLOCKR with address
// information
type BlockrAddress struct {
	Address         string  `json:"address"`
	Balance         float64 `json:"balance"`
	BalanceMultisig float64 `json:"balance_multisig"`
}

// BlockrAddressBalanceSingle holds JSON incoming and outgoing data for BLOCKR
// with address balance information
type BlockrAddressBalanceSingle struct {
	Status  string        `json:"status"`
	Data    BlockrAddress `json:"data"`
	Code    int           `json:"code"`
	Message string        `json:"message"`
}

// BlockrAddressBalanceMulti holds JSON incoming and outgoing data for BLOCKR
// with address balance information for multiple wallets
type BlockrAddressBalanceMulti struct {
	Status  string          `json:"status"`
	Data    []BlockrAddress `json:"data"`
	Code    int             `json:"code"`
	Message string          `json:"message"`
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
func GetEthereumBalance(address []string) (EtherchainBalanceResponse, error) {
	for _, add := range address {
		valid, _ := common.IsValidCryptoAddress(add, "eth")
		if !valid {
			return EtherchainBalanceResponse{}, errors.New("Not an ethereum address")
		}
	}

	addresses := common.JoinStrings(address, ",")
	url := fmt.Sprintf(
		"%s/%s/%s", etherchainAPIURL, etherchainAccountMultiple, addresses,
	)
	result := EtherchainBalanceResponse{}
	err := common.SendHTTPGetRequest(url, true, &result)
	if err != nil {
		return result, err
	}
	if result.Status != 1 {
		return result, errors.New("Status was not 1")
	}
	return result, nil
}

// GetBlockrBalanceSingle queries Blockr for an address balance for either a
// LTC or a BTC single address
func GetBlockrBalanceSingle(address string, coinType string) (BlockrAddressBalanceSingle, error) {
	valid, _ := common.IsValidCryptoAddress(address, coinType)
	if !valid {
		return BlockrAddressBalanceSingle{}, fmt.Errorf(
			"Not a %s address", common.StringToUpper(coinType),
		)
	}

	url := fmt.Sprintf(
		"https://%s.%s/v%s/%s/%s", common.StringToLower(coinType), blockrAPIURL,
		blockrAPIVersion, blockrAddressBalance, address,
	)
	result := BlockrAddressBalanceSingle{}
	err := common.SendHTTPGetRequest(url, true, &result)
	if err != nil {
		return result, err
	}
	if result.Status != "success" {
		return result, errors.New(result.Message)
	}
	return result, nil
}

// GetBlockrAddressMulti queries Blockr for an address balance for either a LTC
// or a BTC multiple addresses
func GetBlockrAddressMulti(addresses []string, coinType string) (BlockrAddressBalanceMulti, error) {
	for _, add := range addresses {
		valid, _ := common.IsValidCryptoAddress(add, coinType)
		if !valid {
			return BlockrAddressBalanceMulti{}, fmt.Errorf(
				"Not a %s address", common.StringToUpper(coinType),
			)
		}
	}
	addressesStr := common.JoinStrings(addresses, ",")
	url := fmt.Sprintf(
		"https://%s.%s/v%s/%s/%s", common.StringToLower(coinType), blockrAPIURL,
		blockrAPIVersion, blockrAddressBalance, addressesStr,
	)
	result := BlockrAddressBalanceMulti{}
	err := common.SendHTTPGetRequest(url, true, &result)
	if err != nil {
		return result, err
	}
	if result.Status != "success" {
		return result, errors.New(result.Message)
	}
	return result, nil
}

// GetAddressBalance acceses the portfolio base and returns the balance by passed
// in address
func (p *Base) GetAddressBalance(address string) (float64, bool) {
	for _, x := range p.Addresses {
		if x.Address == address {
			return x.Balance, true
		}
	}
	return 0, false
}

// ExchangeExists checks to see if an exchange exists in the portfolio base
func (p *Base) ExchangeExists(exchangeName string) bool {
	for _, x := range p.Addresses {
		if x.Address == exchangeName {
			return true
		}
	}
	return false
}

// AddressExists checks to see if there is an address associated with the
// portfolio base
func (p *Base) AddressExists(address string) bool {
	for _, x := range p.Addresses {
		if x.Address == address {
			return true
		}
	}
	return false
}

// ExchangeAddressExists checks to see if there is an exchange address
// associated with the portfolio base
func (p *Base) ExchangeAddressExists(exchangeName, coinType string) bool {
	for _, x := range p.Addresses {
		if x.Address == exchangeName && x.CoinType == coinType {
			return true
		}
	}
	return false
}

// UpdateAddressBalance updates the portfolio base balance
func (p *Base) UpdateAddressBalance(address string, amount float64) {
	for x := range p.Addresses {
		if p.Addresses[x].Address == address {
			p.Addresses[x].Balance = amount
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
	if !p.AddressExists(address) {
		p.Addresses = append(
			p.Addresses, Address{Address: address, CoinType: coinType,
				Balance: balance, Description: description},
		)
	} else {
		p.UpdateAddressBalance(address, balance)
	}
}

// UpdatePortfolio adds to the portfolio addresses by coin type
func (p *Base) UpdatePortfolio(addresses []string, coinType string) bool {
	if common.StringContains(common.JoinStrings(addresses, ","), PortfolioAddressExchange) || common.StringContains(common.JoinStrings(addresses, ","), portfolioAddressPersonal) {
		return true
	}

	if coinType == "ETH" {
		result, err := GetEthereumBalance(addresses)
		if err != nil {
			return false
		}

		for _, x := range result.Data {
			p.AddAddress(x.Address, coinType, portfolioAddressPersonal, x.Balance)
		}
		return true
	}
	if len(addresses) > 1 {
		result, err := GetBlockrAddressMulti(addresses, coinType)
		if err != nil {
			return false
		}
		for _, x := range result.Data {
			p.AddAddress(x.Address, coinType, portfolioAddressPersonal, x.Balance)
		}
	} else {
		result, err := GetBlockrBalanceSingle(addresses[0], coinType)
		if err != nil {
			return false
		}
		p.AddAddress(
			addresses[0], coinType, portfolioAddressPersonal, result.Data.Balance,
		)
	}
	return true
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

// GetPortfolioSummary rpoves a summary for your portfolio base
func (p *Base) GetPortfolioSummary(coinFilter string) map[string]float64 {
	result := make(map[string]float64)
	for _, x := range p.Addresses {
		if coinFilter != "" && coinFilter != x.CoinType {
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

// GetPortfolioGroupedCoin returns portfolio base information grouped by coin
func (p *Base) GetPortfolioGroupedCoin() map[string][]string {
	result := make(map[string][]string)
	for _, x := range p.Addresses {
		if common.StringContains(x.Description, PortfolioAddressExchange) || common.StringContains(x.Description, portfolioAddressPersonal) {
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
