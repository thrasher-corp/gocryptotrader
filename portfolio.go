package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	BLOCKR_API_URL         = "blockr.io/api"
	BLOCKR_API_VERSION     = "1"
	BLOCKR_ADDRESS_BALANCE = "address/balance"

	ETHERCHAIN_API_URL          = "https://etherchain.org/api"
	ETHERCHAIN_ACCOUNT_MULTIPLE = "account/multiple"
)

type PortfolioAddress struct {
	Address  string
	CoinType string
	Balance  float64
}

type Portfolio struct {
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

func GetAddressBalance(address string) (float64, bool) {
	for _, x := range bot.config.Portfolio.Addresses {
		if x.Address == address {
			return x.Balance, true
		}
	}
	return 0, false
}

func AddressExists(address string) bool {
	for _, x := range bot.config.Portfolio.Addresses {
		if x.Address == address {
			return true
		}
	}
	return false
}

func UpdateAddressBalance(address string, amount float64) {
	for x, _ := range bot.config.Portfolio.Addresses {
		if bot.config.Portfolio.Addresses[x].Address == address {
			bot.config.Portfolio.Addresses[x].Balance = amount
		}
	}
}

func UpdatePortfolio(addresses []string, coinType string) bool {
	if coinType == "ETH" {
		result, err := GetEthereumBalance(addresses)
		if err != nil {
			return false
		}

		for _, x := range result.Data {
			if !AddressExists(x.Address) {
				bot.config.Portfolio.Addresses = append(bot.config.Portfolio.Addresses, config.PortfolioAddressConfig{Address: x.Address, CoinType: coinType, Balance: x.Balance / common.WEI_PER_ETHER})
			} else {
				UpdateAddressBalance(x.Address, x.Balance)
			}
		}
		return true
	}
	if len(addresses) > 1 {
		result, err := GetBlockrAddressMulti(addresses, coinType)
		if err != nil {
			return false
		}
		for _, x := range result.Data {
			if !AddressExists(x.Address) {
				bot.config.Portfolio.Addresses = append(bot.config.Portfolio.Addresses, config.PortfolioAddressConfig{Address: x.Address, CoinType: coinType, Balance: x.Balance})
			} else {
				UpdateAddressBalance(x.Address, x.Balance)
			}
		}
	} else {
		result, err := GetBlockrBalanceSingle(addresses[0], coinType)
		if err != nil {
			return false
		}
		if !AddressExists(result.Data.Address) {
			bot.config.Portfolio.Addresses = append(bot.config.Portfolio.Addresses, config.PortfolioAddressConfig{Address: result.Data.Address, CoinType: coinType, Balance: result.Data.Balance})
		} else {
			UpdateAddressBalance(result.Data.Address, result.Data.Balance)
		}
	}
	return true
}

func GetPortfolioSummary(coinFilter string) map[string]float64 {
	result := make(map[string]float64)
	for _, x := range bot.config.Portfolio.Addresses {
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

func GetPortfolioGroupedCoin() map[string][]string {
	result := make(map[string][]string)
	for _, x := range bot.config.Portfolio.Addresses {
		result[x.CoinType] = append(result[x.CoinType], x.Address)
	}
	return result
}

func StartPortfolioWatcher() {
	addrCount := len(bot.config.Portfolio.Addresses)
	log.Printf("PortfolioWatcher started: Have %d address(es) in portfolio.\n", addrCount)
	for {
		data := GetPortfolioGroupedCoin()
		for key, value := range data {
			success := UpdatePortfolio(value, key)
			if success {
				log.Printf("PortfolioWatcher: Successfully updated address balance for %s address(es) %s\n", key, value)
			}
		}
		time.Sleep(time.Minute * 10)
	}
}
