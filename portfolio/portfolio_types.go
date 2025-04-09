package portfolio

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"golang.org/x/time/rate"
)

// Base holds the portfolio base addresses
type Base struct {
	Addresses []Address `json:"addresses"`
	Providers providers `json:"providers,omitzero"`
	Verbose   bool      `json:"verbose"`

	mtx                 sync.RWMutex
	cryptoIDLimiter     *rate.Limiter
	cryptoIDLimiterOnce sync.Once
}

// Address sub type holding address information for portfolio
type Address struct {
	Address            string
	AddressTag         string
	CoinType           currency.Code
	Balance            float64
	Description        string
	WhiteListed        bool
	ColdStorage        bool
	SupportedExchanges string
}

// EtherchainBalanceResponse holds JSON incoming and outgoing data for
// Etherchain
type EtherchainBalanceResponse struct {
	Status int `json:"status"`
	Data   []struct {
		Address   string  `json:"address"`
		Balance   float64 `json:"balance"`
		Nonce     any     `json:"nonce"`
		Code      string  `json:"code"`
		Name      any     `json:"name"`
		Storage   any     `json:"storage"`
		FirstSeen any     `json:"firstSeen"`
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
			Rate      int    `json:"rate"`
			Diff      int    `json:"diff"`
			Timestamp int64  `json:"ts"`
			Currency  string `json:"currency"`
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

// Coin stores a coin type, balance, address and percentage relative to the total
// amount.
type Coin struct {
	Coin       currency.Code `json:"coin"`
	Balance    float64       `json:"balance"`
	Address    string        `json:"address,omitempty"`
	Percentage float64       `json:"percentage,omitempty"`
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
	Totals         []Coin                                         `json:"coin_totals"`
	Offline        []Coin                                         `json:"coins_offline"`
	OfflineSummary map[currency.Code][]OfflineCoinSummary         `json:"offline_summary"`
	Online         []Coin                                         `json:"coins_online"`
	OnlineSummary  map[string]map[currency.Code]OnlineCoinSummary `json:"online_summary"`
}

// XRPScanAccount defines the return type for account data
type XRPScanAccount struct {
	Sequence                                  int     `json:"sequence"`
	XRPBalance                                float64 `json:"xrpBalance,string"`
	OwnerCount                                int     `json:"ownerCount"`
	PreviousAffectingTransactionID            string  `json:"previousAffectingTransactionID"`
	PreviousAffectingTransactionLedgerVersion int     `json:"previousAffectingTransactionLedgerVersion"`
	Settings                                  struct {
		RequireDestinationTag bool   `json:"requireDestinationTag"`
		EmailHash             string `json:"emailHash"`
		Domain                string `json:"domain"`
	} `json:"settings"`
	Account        string      `json:"account"`
	Parent         string      `json:"parent"`
	InitialBalance float64     `json:"initial_balance"`
	Inception      time.Time   `json:"inception"`
	LedgerIndex    int         `json:"ledger_index"`
	TxHash         string      `json:"tx_hash"`
	AccountName    AccountInfo `json:"accountName"`
	ParentName     AccountInfo `json:"parentName"`
	Advisory       any         `json:"advisory"`
}

// AccountInfo is a XRPScan subtype for account associations
type AccountInfo struct {
	Name        string `json:"name"`
	Description string `json:"desc"`
	Account     string `json:"account"`
	Domain      string `json:"domain"`
	Twitter     string `json:"twitter"`
	Verified    bool   `json:"verified"`
}

type provider struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"apiKey,omitempty"`
}

type providers []provider
