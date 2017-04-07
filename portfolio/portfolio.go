package portfolio

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	BLOCKR_API_URL         = "blockr.io/api"
	BLOCKR_API_VERSION     = "1"
	BLOCKR_ADDRESS_BALANCE = "address/balance"

	ETHERCHAIN_API_URL          = "https://etherchain.org/api"
	ETHERCHAIN_ACCOUNT_MULTIPLE = "account/multiple"
	PORTFOLIO_ADDRESS_EXCHANGE  = "Exchange"
	PORTFOLIO_ADDRESS_PERSONAL  = "Personal"
)

var Portfolio PortfolioBase

type PortfolioAddress struct {
	Address      string
	CoinType     string
	Balance      float64
	Decscription string
}

type PortfolioBase struct {
	Addresses []PortfolioAddress
}

type BlockrAddress struct {
	Address         string  `json:"address"`
	Balance         float64 `json:"balance"`
	BalanceMultisig float64 `json:"balance_multisig"`
}

type BlockrAddressBalanceSingle struct {
	Status  string        `json:"status"`
	Data    BlockrAddress `json:"data"`
	Code    int           `json:"code"`
	Message string        `json:"message"`
}

type BlockrAddressBalanceMulti struct {
	Status  string          `json:"status"`
	Data    []BlockrAddress `json:"data"`
	Code    int             `json:"code"`
	Message string          `json:"message"`
}

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

//ExchangeAccountInfo : Generic type to hold each exchange's holdings in all enabled currencies
type ExchangeAccountInfo struct {
	ExchangeName string
	Currencies   []ExchangeAccountCurrencyInfo
}

//ExchangeAccountCurrencyInfo : Sub type to store currency name and value
type ExchangeAccountCurrencyInfo struct {
	CurrencyName string
	TotalValue   float64
	Hold         float64
}

func GetEthereumBalance(address []string) (EtherchainBalanceResponse, error) {
	addresses := common.JoinStrings(address, ",")
	url := fmt.Sprintf("%s/%s/%s", ETHERCHAIN_API_URL, ETHERCHAIN_ACCOUNT_MULTIPLE, addresses)
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

func GetBlockrBalanceSingle(address string, coinType string) (BlockrAddressBalanceSingle, error) {
	url := fmt.Sprintf("https://%s.%s/v%s/%s/%s", common.StringToLower(coinType), BLOCKR_API_URL, BLOCKR_API_VERSION, BLOCKR_ADDRESS_BALANCE, address)
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

func GetBlockrAddressMulti(addresses []string, coinType string) (BlockrAddressBalanceMulti, error) {
	addressesStr := common.JoinStrings(addresses, ",")
	url := fmt.Sprintf("https://%s.%s/v%s/%s/%s", common.StringToLower(coinType), BLOCKR_API_URL, BLOCKR_API_VERSION, BLOCKR_ADDRESS_BALANCE, addressesStr)
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

func (p *PortfolioBase) GetAddressBalance(address string) (float64, bool) {
	for _, x := range p.Addresses {
		if x.Address == address {
			return x.Balance, true
		}
	}
	return 0, false
}

func (p *PortfolioBase) ExchangeExists(exchangeName string) bool {
	for _, x := range p.Addresses {
		if x.Address == exchangeName {
			return true
		}
	}
	return false
}

func (p *PortfolioBase) AddressExists(address string) bool {
	for _, x := range p.Addresses {
		if x.Address == address {
			return true
		}
	}
	return false
}

func (p *PortfolioBase) ExchangeAddressExists(exchangeName, coinType string) bool {
	for _, x := range p.Addresses {
		if x.Address == exchangeName && x.CoinType == coinType {
			return true
		}
	}
	return false
}

func (p *PortfolioBase) UpdateAddressBalance(address string, amount float64) {
	for x, _ := range p.Addresses {
		if p.Addresses[x].Address == address {
			p.Addresses[x].Balance = amount
		}
	}
}

func (p *PortfolioBase) UpdateExchangeAddressBalance(exchangeName, coinType string, balance float64) {
	for x, _ := range p.Addresses {
		if p.Addresses[x].Address == exchangeName && p.Addresses[x].CoinType == coinType {
			p.Addresses[x].Balance = balance
		}
	}
}

func (p *PortfolioBase) AddAddress(address, coinType, description string, balance float64) {
	if !p.AddressExists(address) {
		p.Addresses = append(p.Addresses, PortfolioAddress{Address: address, CoinType: coinType, Balance: balance, Decscription: description})
	} else {
		p.UpdateAddressBalance(address, balance)
	}
}

func (p *PortfolioBase) UpdatePortfolio(addresses []string, coinType string) bool {
	if common.StringContains(common.JoinStrings(addresses, ","), PORTFOLIO_ADDRESS_EXCHANGE) || common.StringContains(common.JoinStrings(addresses, ","), PORTFOLIO_ADDRESS_PERSONAL) {
		return true
	}

	if coinType == "ETH" {
		result, err := GetEthereumBalance(addresses)
		if err != nil {
			return false
		}

		for _, x := range result.Data {
			p.AddAddress(x.Address, coinType, PORTFOLIO_ADDRESS_PERSONAL, x.Balance)
		}
		return true
	}
	if len(addresses) > 1 {
		result, err := GetBlockrAddressMulti(addresses, coinType)
		if err != nil {
			return false
		}
		for _, x := range result.Data {
			p.AddAddress(x.Address, coinType, PORTFOLIO_ADDRESS_PERSONAL, x.Balance)
		}
	} else {
		result, err := GetBlockrBalanceSingle(addresses[0], coinType)
		if err != nil {
			return false
		}
		p.AddAddress(addresses[0], coinType, PORTFOLIO_ADDRESS_PERSONAL, result.Data.Balance)
	}
	return true
}

func (p *PortfolioBase) GetExchangePortfolio() map[string]float64 {
	result := make(map[string]float64)
	for _, x := range p.Addresses {
		if x.Decscription != PORTFOLIO_ADDRESS_EXCHANGE {
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

func (p *PortfolioBase) GetPersonalPortfolio() map[string]float64 {
	result := make(map[string]float64)
	for _, x := range p.Addresses {
		if x.Decscription == PORTFOLIO_ADDRESS_EXCHANGE {
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

func (p *PortfolioBase) GetPortfolioSummary(coinFilter string) map[string]float64 {
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

func (p *PortfolioBase) GetPortfolioGroupedCoin() map[string][]string {
	result := make(map[string][]string)
	for _, x := range p.Addresses {
		if common.StringContains(x.Decscription, PORTFOLIO_ADDRESS_EXCHANGE) || common.StringContains(x.Decscription, PORTFOLIO_ADDRESS_PERSONAL) {
			continue
		}
		result[x.CoinType] = append(result[x.CoinType], x.Address)
	}
	return result
}

func (p *PortfolioBase) SeedPortfolio(port PortfolioBase) {
	p.Addresses = port.Addresses
}

func StartPortfolioWatcher() {
	addrCount := len(Portfolio.Addresses)
	log.Printf("PortfolioWatcher started: Have %d entries in portfolio.\n", addrCount)
	for {
		data := Portfolio.GetPortfolioGroupedCoin()
		for key, value := range data {
			success := Portfolio.UpdatePortfolio(value, key)
			if success {
				log.Printf("PortfolioWatcher: Successfully updated address balance for %s address(es) %s\n", key, value)
			}
		}
		time.Sleep(time.Minute * 10)
	}
}

func GetPortfolio() *PortfolioBase {
	return &Portfolio
}
