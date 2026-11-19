package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// AccountResponse holds the account's information and balances
type AccountResponse struct {
	MakerCommissionInBasisPoints  int64           `json:"makerCommission"`  // signed, as a rebate is a negative commission
	TakerCommissionInBasisPoints  int64           `json:"takerCommission"`  // signed, as a rebate is a negative commission
	BuyerCommissionInBasisPoints  int64           `json:"buyerCommission"`  // signed, as a rebate is a negative commission
	SellerCommissionInBasisPoints int64           `json:"sellerCommission"` // signed, as a rebate is a negative commission
	CommissionRates               CommissionRates `json:"commissionRates"`
	CanTrade                      bool            `json:"canTrade"`
	CanWithdraw                   bool            `json:"canWithdraw"`
	CanDeposit                    bool            `json:"canDeposit"`
	Brokered                      bool            `json:"brokered"`
	RequireSelfTradePrevention    bool            `json:"requireSelfTradePrevention"`
	UpdateTime                    types.Time      `json:"updateTime"`
	AccountType                   string          `json:"accountType"`
	Balances                      []Balance       `json:"balances"`
	Permissions                   []Permission    `json:"permissions"`
	UID                           uint64          `json:"uid"`
}

// CommissionRates holds maker, taker, buyer and seller commission rates
type CommissionRates struct {
	Maker  types.Number `json:"maker"`
	Taker  types.Number `json:"taker"`
	Buyer  types.Number `json:"buyer"`
	Seller types.Number `json:"seller"`
}

// Balance holds an asset's free and locked balance
type Balance struct {
	Asset  currency.Code   `json:"asset"`
	Free   decimal.Decimal `json:"free"`
	Locked decimal.Decimal `json:"locked"`
}

// AccountStatusResponse holds the account's status. A suspended account's message says why and its objects hold the
// details, such as the minutes until trading is reactivated
type AccountStatusResponse struct {
	Message string   `json:"msg"`
	Success bool     `json:"success"`
	Objects []string `json:"objs"`
}

// APITradingStatusResponse holds the account's API trading status
type APITradingStatusResponse struct {
	Success bool             `json:"success"`
	Status  APITradingStatus `json:"status"`
}

// APITradingStatus holds whether API trading is locked and the order ratios that lock it
type APITradingStatus struct {
	IsLocked bool `json:"isLocked"`
	// PlannedRecoverTime is when locked API trading is to be restored
	PlannedRecoverTime types.Time                    `json:"plannedRecoverTime"`
	TriggerCondition   APITradingTriggerCondition    `json:"triggerCondition"`
	Indicators         map[string][]TradingIndicator `json:"indicators"` // Keyed by symbol, updated every 30 seconds
	UpdateTime         types.Time                    `json:"updateTime"`
}

// APITradingTriggerCondition holds the order counts each ratio is measured over before it can lock API trading
type APITradingTriggerCondition struct {
	GTCCancellationRatio    uint64 `json:"GCR"`  // Number of GTC orders
	IOCOrFOKExpirationRatio uint64 `json:"IFER"` // Number of FOK and IOC orders
	UnfilledRatio           uint64 `json:"UFR"`  // Number of orders
}

// TradingIndicator holds an order ratio of a symbol: its current value and the value that locks API trading
type TradingIndicator struct {
	Indicator    string  `json:"i"` // UFR, IFER or GCR
	Count        uint64  `json:"c"`
	CurrentValue float64 `json:"v"`
	TriggerValue float64 `json:"t"`
}

// AssetDistributionHistoryRequest holds the optional filters of an asset distribution history request
type AssetDistributionHistoryRequest struct {
	Asset currency.Code
	// Category is the distribution category, such as Market Maker Rebate, MM Streaks Rebate, API Partner Rebate or
	// airdrop
	Category  string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 20 when zero, at most 500
}

// AssetDistributionHistoryResponse holds asset distributions and their total number
type AssetDistributionHistoryResponse struct {
	Rows  []AssetDistribution `json:"rows"`
	Total uint64              `json:"total"`
}

// AssetDistribution holds an asset distribution
type AssetDistribution struct {
	Amount           types.Number  `json:"amount"`
	Asset            currency.Code `json:"asset"`
	DistributionTime types.Time    `json:"divTime"`
	Category         string        `json:"category"`
	TransactionID    uint64        `json:"tranId"`
}

// TradeFeeResponse holds a symbol's maker and taker commission rates
type TradeFeeResponse struct {
	Symbol          string       `json:"symbol"`
	MakerCommission types.Number `json:"makerCommission"`
	TakerCommission types.Number `json:"takerCommission"`
}

// AccountCommissionResponse holds the account's commission rates for a symbol
type AccountCommissionResponse struct {
	Symbol             string             `json:"symbol"`
	StandardCommission CommissionRates    `json:"standardCommission"`
	TaxCommission      CommissionRates    `json:"taxCommission"`
	Discount           CommissionDiscount `json:"discount"`
}

// CommissionDiscount holds the discount on standard commissions paid in the discount asset
type CommissionDiscount struct {
	EnabledForAccount bool          `json:"enabledForAccount"`
	EnabledForSymbol  bool          `json:"enabledForSymbol"`
	DiscountAsset     currency.Code `json:"discountAsset"`
	Discount          types.Number  `json:"discount"` // Rate the standard commission is reduced by
}

// TradeVolumeResponse holds the account's trading volume over the past 30 days
type TradeVolumeResponse struct {
	Past30DaysTradingVolume float64 `json:"past30DaysTradingVolume"`
}

// MyFiltersResponse holds the filters that apply to the account on a symbol; asset filters, such as MAX_ASSET, are
// returned only here
type MyFiltersResponse struct {
	ExchangeFilters []ExchangeFilter `json:"exchangeFilters"`
	SymbolFilters   []SymbolFilter   `json:"symbolFilters"`
	AssetFilters    []AssetFilter    `json:"assetFilters"`
}
