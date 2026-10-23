package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AccountSummaryInformationResponse holds the VIP level and the sub-account counts and limits of the account
type AccountSummaryInformationResponse struct {
	Level                 uint64 `json:"level"`
	SubQuantity           uint64 `json:"subQuantity"`
	SpotSubQuantity       uint64 `json:"spotSubQuantity"`
	MarginSubQuantity     uint64 `json:"marginSubQuantity"`
	FuturesSubQuantity    uint64 `json:"futuresSubQuantity"`
	OptionSubQuantity     uint64 `json:"optionSubQuantity"`
	MaxSubQuantity        uint64 `json:"maxSubQuantity"` // MaxDefaultSubQuantity + MaxSpotSubQuantity
	MaxDefaultSubQuantity uint64 `json:"maxDefaultSubQuantity"`
	MaxSpotSubQuantity    uint64 `json:"maxSpotSubQuantity"`
	MaxMarginSubQuantity  uint64 `json:"maxMarginSubQuantity"`
	MaxFuturesSubQuantity uint64 `json:"maxFuturesSubQuantity"`
	MaxOptionSubQuantity  uint64 `json:"maxOptionSubQuantity"`
}

// APIKeyInfoResponse holds the details of an API key
type APIKeyInfoResponse struct {
	UID                   uint64     `json:"uid"`
	ParentUID             uint64     `json:"parentUid"` // Zero for a master account key
	Region                string     `json:"region"`    // KYC region as a two-letter country code
	KYCStatus             uint64     `json:"kycStatus"` // 1 passed, 0 not passed
	SubName               string     `json:"subName"`
	Remark                string     `json:"remark"`
	APIKey                string     `json:"apiKey"`
	APIVersion            uint64     `json:"apiVersion"`
	Permission            string     `json:"permission"`  // Comma-separated, e.g. General,Spot,Futures
	IPWhitelist           string     `json:"ipWhitelist"` // Comma-separated
	IsMaster              bool       `json:"isMaster"`
	CreatedAt             types.Time `json:"createdAt"`
	ExpiredAt             types.Time `json:"expiredAt"` // Zero when the key does not expire
	ThirdPartyApplication string     `json:"thirdPartyApp"`
	SiteType              string     `json:"siteType"`
}

// AccountDetailResponse holds the balance of a single spot account
type AccountDetailResponse struct {
	Currency  currency.Code `json:"currency"`
	Balance   types.Number  `json:"balance"`
	Available types.Number  `json:"available"`
	Holds     types.Number  `json:"holds"`
}

// CrossMarginAccountsDetailResponse holds the cross margin account
type CrossMarginAccountsDetailResponse struct {
	TotalAssetOfQuoteCurrency     types.Number        `json:"totalAssetOfQuoteCurrency"`
	TotalLiabilityOfQuoteCurrency types.Number        `json:"totalLiabilityOfQuoteCurrency"`
	DebtRatio                     types.Number        `json:"debtRatio"`
	Status                        string              `json:"status"` // EFFECTIVE, BANKRUPTCY, LIQUIDATION, REPAY or BORROW
	Accounts                      []MarginAssetDetail `json:"accounts"`
}

// MarginAssetDetail holds a currency's balance in a cross or isolated margin account
type MarginAssetDetail struct {
	Currency           currency.Code `json:"currency"`
	Total              types.Number  `json:"total"`
	Available          types.Number  `json:"available"`
	Hold               types.Number  `json:"hold"`
	Liability          types.Number  `json:"liability"`
	MaxBorrowSize      types.Number  `json:"maxBorrowSize"`
	BorrowEnabled      bool          `json:"borrowEnabled"`
	TransferInEnabled  bool          `json:"transferInEnabled"`
	LiabilityPrincipal types.Number  `json:"liabilityPrincipal"`
	LiabilityInterest  types.Number  `json:"liabilityInterest"`
}

// IsolatedMarginAccountDetailRequest holds the filters for GetIsolatedMarginAccountDetail
type IsolatedMarginAccountDetailRequest struct {
	Symbol        string
	QuoteCurrency currency.Code // USDT (default), KCS or BTC
	QueryType     string        // ISOLATED, ISOLATED_V2 (being phased out, same meaning) or ALL
}

// IsolatedMarginAccountDetailResponse holds the isolated margin accounts
type IsolatedMarginAccountDetailResponse struct {
	TotalAssetOfQuoteCurrency     types.Number                `json:"totalAssetOfQuoteCurrency"`
	TotalLiabilityOfQuoteCurrency types.Number                `json:"totalLiabilityOfQuoteCurrency"`
	Timestamp                     types.Time                  `json:"timestamp"`
	Assets                        []IsolatedMarginAssetDetail `json:"assets"`
}

// IsolatedMarginAssetDetail holds the isolated margin account of a symbol
type IsolatedMarginAssetDetail struct {
	Symbol     string            `json:"symbol"`
	Status     string            `json:"status"` // EFFECTIVE, BANKRUPTCY, LIQUIDATION, REPAY or BORROW
	DebtRatio  types.Number      `json:"debtRatio"`
	BaseAsset  MarginAssetDetail `json:"baseAsset"`
	QuoteAsset MarginAssetDetail `json:"quoteAsset"`
}

// FuturesAccountOverviewResponse holds the futures account of a settlement currency
type FuturesAccountOverviewResponse struct {
	AccountEquity     float64       `json:"accountEquity"` // MarginBalance + UnrealisedPNL
	UnrealisedPNL     float64       `json:"unrealisedPNL"`
	MarginBalance     float64       `json:"marginBalance"` // PositionMargin + OrderMargin + FrozenFunds + AvailableBalance - UnrealisedPNL
	PositionMargin    float64       `json:"positionMargin"`
	OrderMargin       float64       `json:"orderMargin"`
	FrozenFunds       float64       `json:"frozenFunds"`      // Frozen for transfers out
	AvailableBalance  float64       `json:"availableBalance"` // Transferable amount
	Currency          currency.Code `json:"currency"`
	RiskRatio         float64       `json:"riskRatio"` // The account is liquidated at 1 (100%) or more
	MaxWithdrawAmount float64       `json:"maxWithdrawAmount"`
	AvailableMargin   float64       `json:"availableMargin"` // Cross margin available
}

// AccountLedgersRequest holds the filters for GetAccountLedgers
type AccountLedgersRequest struct {
	Currencies   []currency.Code // At most 10; every currency when empty
	Direction    string          // in or out
	BusinessType string          // e.g. DEPOSIT, WITHDRAW, TRANSFER, SUB_TRANSFER, TRADE_EXCHANGE, MARGIN_EXCHANGE
	StartAt      time.Time
	EndAt        time.Time
	CurrentPage  uint64
	PageSize     uint64 // 10 to 500
}

// AccountLedgersResponse holds a page of account ledger entries
type AccountLedgersResponse struct {
	CurrentPage uint64          `json:"currentPage"`
	PageSize    uint64          `json:"pageSize"`
	TotalNumber uint64          `json:"totalNum"`
	TotalPage   uint64          `json:"totalPage"`
	Items       []AccountLedger `json:"items"`
}

// AccountLedger holds an account ledger entry
type AccountLedger struct {
	ID           string        `json:"id"`
	Currency     currency.Code `json:"currency"`
	Amount       types.Number  `json:"amount"` // Including fees
	Fee          types.Number  `json:"fee"`
	Balance      types.Number  `json:"balance"` // Deprecated by KuCoin and carries no meaning
	AccountType  string        `json:"accountType"`
	BusinessType string        `json:"bizType"`
	Direction    string        `json:"direction"`
	CreatedAt    types.Time    `json:"createdAt"`
	Context      string        `json:"context"` // JSON with the order ID, trade ID and symbol of trades
}

// HFAccountLedgersRequest holds the filters for GetAccountLedgersHFTrade and GetAccountLedgerHFMargin
type HFAccountLedgersRequest struct {
	Currencies   []currency.Code // At most 10; every currency when empty
	Direction    string          // in or out
	BusinessType string          // The trade and margin ledgers take different values
	LastID       uint64          // Returns entries before this ID
	Limit        uint64          // Default 100, at most 200
	StartAt      time.Time
	EndAt        time.Time
}

// HFAccountLedgerResponse holds a high-frequency trading account ledger entry
type HFAccountLedgerResponse struct {
	ID           string        `json:"id"`
	Currency     currency.Code `json:"currency"`
	Amount       types.Number  `json:"amount"`
	Fee          types.Number  `json:"fee"`
	Tax          types.Number  `json:"tax"`
	Balance      types.Number  `json:"balance"`
	AccountType  string        `json:"accountType"`
	BusinessType string        `json:"bizType"`
	Direction    string        `json:"direction"`
	CreatedAt    types.Time    `json:"createdAt"`
	Context      string        `json:"context"` // JSON with the order ID, trade ID and symbol of trades
}

// HFMarginAccountLedgerResponse holds a high-frequency margin account ledger entry
type HFMarginAccountLedgerResponse struct {
	ID           uint64        `json:"id"`
	Currency     currency.Code `json:"currency"`
	Amount       types.Number  `json:"amount"`
	Fee          types.Number  `json:"fee"`
	Balance      types.Number  `json:"balance"`
	AccountType  string        `json:"accountType"` // MARGIN_V2 or ISOLATED_V2, moving to MARGIN or ISOLATED
	BusinessType string        `json:"bizType"`
	Direction    string        `json:"direction"`
	CreatedAt    types.Time    `json:"createdAt"`
	Context      string        `json:"context"` // JSON with the order ID, trade ID and symbol of trades
	Tax          types.Number  `json:"tax"`
}

// FuturesAccountLedgersRequest holds the filters for GetFuturesAccountLedgers
type FuturesAccountLedgersRequest struct {
	Currency currency.Code // XBT or USDT
	Type     string        // RealizedPNL, Deposit, Withdrawal, Transferin or TransferOut
	Offset   uint64
	Forward  *bool // KuCoin looks forward unless this is false
	MaxCount uint64
	StartAt  time.Time
	EndAt    time.Time
}

// FuturesAccountLedgersResponse holds a page of futures ledger entries
type FuturesAccountLedgersResponse struct {
	DataList []FuturesAccountLedger `json:"dataList"`
	HasMore  bool                   `json:"hasMore"`
}

// FuturesAccountLedger holds a futures ledger entry
type FuturesAccountLedger struct {
	Time          types.Time    `json:"time"`
	Type          string        `json:"type"`
	Amount        float64       `json:"amount"`
	Fee           float64       `json:"fee"`
	AccountEquity float64       `json:"accountEquity"`
	Status        string        `json:"status"` // Completed or Pending
	Remark        string        `json:"remark"`
	Offset        uint64        `json:"offset"`
	Currency      currency.Code `json:"currency"`
}
