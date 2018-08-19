package portfolio

import "github.com/shopspring/decimal"

// Base holds the portfolio base addresses
type Base struct {
	Addresses []Address
}

// Address sub type holding address information for portfolio
type Address struct {
	Address     string
	CoinType    string
	Balance     decimal.Decimal
	Description string
}

// EtherchainBalanceResponse holds JSON incoming and outgoing data for
// Etherchain
type EtherchainBalanceResponse struct {
	Status int `json:"status"`
	Data   []struct {
		Address   string          `json:"address"`
		Balance   decimal.Decimal `json:"balance"`
		Nonce     interface{}     `json:"nonce"`
		Code      string          `json:"code"`
		Name      interface{}     `json:"name"`
		Storage   interface{}     `json:"storage"`
		FirstSeen interface{}     `json:"firstSeen"`
	} `json:"data"`
}

// EthplorerResponse holds JSON address data for Ethplorer
type EthplorerResponse struct {
	Address string `json:"address"`
	ETH     struct {
		Balance  decimal.Decimal `json:"balance"`
		TotalIn  decimal.Decimal `json:"totalIn"`
		TotalOut decimal.Decimal `json:"totalOut"`
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
	TotalValue   decimal.Decimal
	Hold         decimal.Decimal
}

// Coin stores a coin type, balance, address and percentage relative to the total
// amount.
type Coin struct {
	Coin       string          `json:"coin"`
	Balance    decimal.Decimal `json:"balance"`
	Address    string          `json:"address,omitempty"`
	Percentage decimal.Decimal `json:"percentage,omitempty"`
}

// OfflineCoinSummary stores a coin types address, balance and percentage
// relative to the total amount.
type OfflineCoinSummary struct {
	Address    string          `json:"address"`
	Balance    decimal.Decimal `json:"balance"`
	Percentage decimal.Decimal `json:"percentage,omitempty"`
}

// OnlineCoinSummary stores a coin types balance and percentage relative to the
// total amount.
type OnlineCoinSummary struct {
	Balance    decimal.Decimal `json:"balance"`
	Percentage decimal.Decimal `json:"percentage,omitempty"`
}

// Summary Stores the entire portfolio summary
type Summary struct {
	Totals         []Coin                                  `json:"coin_totals"`
	Offline        []Coin                                  `json:"coins_offline"`
	OfflineSummary map[string][]OfflineCoinSummary         `json:"offline_summary"`
	Online         []Coin                                  `json:"coins_online"`
	OnlineSummary  map[string]map[string]OnlineCoinSummary `json:"online_summary"`
}
