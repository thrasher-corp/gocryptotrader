package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// BrokerWhitelistIP holds a broker client's dedicated hostname and the IP addresses whitelisted for it
type BrokerWhitelistIP struct {
	// Name is the client's name
	Name string `json:"name"`
	// EndpointDomain is the hostname dedicated to the client
	EndpointDomain string   `json:"endpointDomain"`
	IPList         []string `json:"ipList"`
}

// GetBrokerIPChangeLogRequest holds the filters for listing changes to a broker's whitelisted IP addresses
type GetBrokerIPChangeLogRequest struct {
	StartTime time.Time
	EndTime   time.Time
	// Limit is from 1 to 50 and defaults to 20
	Limit  uint64
	Cursor string
}

// BrokerIPChangeLogResponse holds a page of changes to a broker's whitelisted IP addresses
type BrokerIPChangeLogResponse struct {
	List           []BrokerIPChange `json:"list"`
	NextPageCursor string           `json:"nextPageCursor"`
}

// BrokerIPChange holds a change to a broker's whitelisted IP addresses
type BrokerIPChange struct {
	// OperationType is ADD, REMOVE, ENABLE or DISABLE
	OperationType string `json:"operationType"`
	// IPAddress may be a CIDR range
	IPAddress string `json:"ipAddress"`
	// DomainType is GW for the HTTP hostname or WS for the websocket hostname
	DomainType   string    `json:"domainType"`
	ChangeReason string    `json:"changeReason"`
	CreatedAt    time.Time `json:"createdAt"`
}

// GetExchangeBrokerEarningRequest holds the filters for an exchange broker's earnings
type GetExchangeBrokerEarningRequest struct {
	// BusinessType is SPOT, DERIVATIVES, OPTIONS, CONVERT or FIAT_CONVERT
	BusinessType string
	// Begin and End are UTC dates, sent together or not at all; both days are included, and without them the last 7
	// days are returned
	Begin time.Time
	End   time.Time
	// UID filters by a subaccount; every subaccount is returned when empty
	UID string
	// Limit is from 1 to 1000 and defaults to 1000
	Limit  uint64
	Cursor string
}

// ExchangeBrokerEarningResponse holds an exchange broker's earnings, totalled per category and detailed per trade
type ExchangeBrokerEarningResponse struct {
	TotalEarningCategory ExchangeBrokerEarningCategories `json:"totalEarningCat"`
	Details              []ExchangeBrokerEarningDetail   `json:"details"`
	NextPageCursor       string                          `json:"nextPageCursor"`
}

// ExchangeBrokerEarningCategories holds an exchange broker's total earnings per business category; a category without
// rebates is empty
type ExchangeBrokerEarningCategories struct {
	Spot        []BrokerCoinEarning `json:"spot"`
	Derivatives []BrokerCoinEarning `json:"derivatives"`
	Options     []BrokerCoinEarning `json:"options"`
	Convert     []BrokerCoinEarning `json:"convert"`
	FiatConvert []BrokerCoinEarning `json:"fiatConvert"`
	// Total sums every category
	Total []BrokerCoinEarning `json:"total"`
}

// BrokerCoinEarning holds a broker's rebate in one coin
type BrokerCoinEarning struct {
	Coin    currency.Code `json:"coin"`
	Earning types.Number  `json:"earning"`
}

// ExchangeBrokerEarningDetail holds the rebate an exchange broker earned on one trade of a subaccount
type ExchangeBrokerEarningDetail struct {
	UserID string `json:"userId"`
	// BusinessType is SPOT, DERIVATIVES, OPTIONS, CONVERT or FIAT_CONVERT
	BusinessType string        `json:"bizType"`
	Symbol       string        `json:"symbol"`
	Coin         currency.Code `json:"coin"`
	Earning      types.Number  `json:"earning"`
	// MarkupEarning and BaseFeeEarning are the parts of Earning from the markup and base fee rates
	MarkupEarning  types.Number `json:"markupEarning"`
	BaseFeeEarning types.Number `json:"baseFeeEarning"`
	OrderID        string       `json:"orderId"`
	ExecutionID    string       `json:"execId"`
	ExecutionTime  types.Time   `json:"execTime"`
}

// ExchangeBrokerAccountInfoResponse holds an exchange broker's subaccount quota and rebate rates
type ExchangeBrokerAccountInfoResponse struct {
	SubAccountQuantity    types.Number                      `json:"subAcctQty"`
	MaxSubAccountQuantity types.Number                      `json:"maxSubAcctQty"`
	BaseFeeRebateRate     ExchangeBrokerBaseFeeRebateRate   `json:"baseFeeRebateRate"`
	MarkupFeeRebateRate   ExchangeBrokerMarkupFeeRebateRate `json:"markupFeeRebateRate"`
	Timestamp             types.Time                        `json:"ts"`
}

// ExchangeBrokerBaseFeeRebateRate holds an exchange broker's base fee rebate percentages, such as "10.0%"
type ExchangeBrokerBaseFeeRebateRate struct {
	Spot        string `json:"spot"`
	Derivatives string `json:"derivatives"`
}

// ExchangeBrokerMarkupFeeRebateRate holds an exchange broker's markup fee rebate percentages, such as "6.00%"
type ExchangeBrokerMarkupFeeRebateRate struct {
	Spot        string `json:"spot"`
	Derivatives string `json:"derivatives"`
	Convert     string `json:"convert"`
}

// GetBrokerSubAccountDepositRecordsRequest holds the filters for an exchange broker's subaccount deposits
type GetBrokerSubAccountDepositRecordsRequest struct {
	// ID is the deposit's internal ID and takes precedence over the other filters
	ID string
	// TransactionID cannot find deposits made before 1 January 2024
	TransactionID string
	SubMemberID   string
	Coin          currency.Code
	// StartTime and EndTime must be less than 30 days apart and default to the last 30 days; Bybit applies them to the
	// second
	StartTime time.Time
	EndTime   time.Time
	// Limit is from 1 to 50 and defaults to 50
	Limit  uint64
	Cursor string
}

// BrokerSubAccountDepositRecordsResponse holds a page of an exchange broker's subaccount deposits, newest first
type BrokerSubAccountDepositRecordsResponse struct {
	Rows           []BrokerSubAccountDeposit `json:"rows"`
	NextPageCursor string                    `json:"nextPageCursor"`
}

// BrokerSubAccountDeposit holds a deposit to one of an exchange broker's subaccounts
type BrokerSubAccountDeposit struct {
	ID            string        `json:"id"`
	SubMemberID   string        `json:"subMemberId"`
	Coin          currency.Code `json:"coin"`
	Chain         string        `json:"chain"`
	Amount        types.Number  `json:"amount"`
	TransactionID string        `json:"txID"`
	// Status is the deposit status, such as 3 success or 4 failed
	Status           uint64       `json:"status"`
	ToAddress        string       `json:"toAddress"`
	Tag              string       `json:"tag"`
	DepositFee       types.Number `json:"depositFee"`
	SuccessAt        types.Time   `json:"successAt"`
	Confirmations    types.Number `json:"confirmations"`
	TransactionIndex types.Number `json:"txIndex"`
	BlockHash        string       `json:"blockHash"`
	// BatchReleaseLimit is the coin's deposit limit on the chain; -1 means no limit
	BatchReleaseLimit types.Number `json:"batchReleaseLimit"`
	// DepositType is 0 normal, 10 daily limit reached, 20 held for AML review or 50 blocked by a restriction
	DepositType string `json:"depositType"`
	// FromAddress is set only for on-chain deposits from a single address
	FromAddress string `json:"fromAddress"`
	// TaxDepositRecordsID and TaxStatus serve Bybit EU (Austria) tax reporting: TaxStatus is 0 not required, 1 pending
	// or 2 completed
	TaxDepositRecordsID string `json:"taxDepositRecordsId"`
	TaxStatus           uint64 `json:"taxStatus"`
	// TravelRuleStatus is 0 approved, 1 counterparty information needed, 2 under review or 3 rejected
	TravelRuleStatus uint64 `json:"travelRuleStatus"`
}

// BrokerRateLimitRequest holds the rate limit to set for one business type of some accounts
type BrokerRateLimitRequest struct {
	// UIDs are a master account's subaccounts, or the calling subaccount itself
	UIDs []string
	// BusinessType is SPOT, DERIVATIVES or OPTIONS
	BusinessType string
	// Rate is in requests per second
	Rate uint64
}

// SetBrokerRateLimitResponse holds the outcome of each rate limit set
type SetBrokerRateLimitResponse struct {
	Result []BrokerRateLimitResult `json:"result"`
}

// BrokerRateLimitResult holds the outcome of setting one rate limit
type BrokerRateLimitResult struct {
	// UIDs is comma separated
	UIDs         string `json:"uids"`
	BusinessType string `json:"bizType"`
	// Rate is in requests per second
	Rate    uint64 `json:"rate"`
	Success bool   `json:"success"`
	// Message describes the outcome, such as "Requested limit exceeds maximum allowed per user"
	Message string `json:"msg"`
}

// BrokerRateLimitCapResponse holds an exchange broker's rate limit caps and usage per business type; it is empty
// until the account manager has configured caps
type BrokerRateLimitCapResponse struct {
	List []BrokerRateLimitCap `json:"list"`
}

// BrokerRateLimitCap holds an exchange broker's rate limit caps and usage for one business type, in requests per
// second
type BrokerRateLimitCap struct {
	BusinessType string `json:"bizType"`
	// TotalRate is the rate allocated across the master account and its subaccounts
	TotalRate         types.Number `json:"totalRate"`
	ExchangeBrokerCap types.Number `json:"ebCap"`
	UIDCap            types.Number `json:"uidCap"`
}

// GetBrokerAllRateLimitsRequest holds the filters for listing an exchange broker's configured rate limits
type GetBrokerAllRateLimitsRequest struct {
	// Limit is from 1 to 1000 and defaults to 1000
	Limit  uint64
	Cursor string
	// UIDs may span master accounts; every subaccount is returned when empty
	UIDs []string
}

// BrokerAllRateLimitsResponse holds a page of an exchange broker's configured rate limits; accounts never configured
// use the default limits and are left out
type BrokerAllRateLimitsResponse struct {
	NextPageCursor string            `json:"nextPageCursor"`
	List           []BrokerRateLimit `json:"list"`
}

// BrokerRateLimit holds the rate limit configured for one business type of some accounts
type BrokerRateLimit struct {
	// UIDs is comma separated
	UIDs         string `json:"uids"`
	BusinessType string `json:"bizType"`
	// Rate is in requests per second
	Rate uint64 `json:"rate"`
}

// BrokerVoucherSpecResponse holds a voucher's specification
type BrokerVoucherSpecResponse struct {
	ID   string        `json:"id"`
	Coin currency.Code `json:"coin"`
	// AmountUnit is AWARD_AMOUNT_UNIT_USD or AWARD_AMOUNT_UNIT_COIN
	AmountUnit     string `json:"amountUnit"`
	ProductLine    string `json:"productLine"`
	SubProductLine string `json:"subProductLine"`
	// TotalAmount is documented as an object but arrives as a quoted number
	TotalAmount types.Number `json:"totalAmount"`
	UsedAmount  types.Number `json:"usedAmount"`
}

// IssueBrokerVoucherRequest holds a voucher to issue to a user
type IssueBrokerVoucherRequest struct {
	AccountID string
	// AwardID is the voucher ID
	AwardID string
	// SpecCode is a unique code of up to 8 characters
	SpecCode string
	// Amount takes up to 16 decimal places for spot airdrops and 4 otherwise
	Amount   float64
	BrokerID string
}

// GetBrokerIssuedVoucherRequest identifies a voucher issued to a user
type GetBrokerIssuedVoucherRequest struct {
	AccountID string
	// AwardID is the voucher ID
	AwardID  string
	SpecCode string
	// WithUsedAmount also returns how much of the voucher the user has used
	WithUsedAmount bool
}

// BrokerIssuedVoucherResponse holds a voucher issued to a user
type BrokerIssuedVoucherResponse struct {
	AccountID string       `json:"accountId"`
	AwardID   string       `json:"awardId"`
	SpecCode  string       `json:"specCode"`
	Amount    types.Number `json:"amount"`
	IsClaimed bool         `json:"isClaimed"`
	// StartAt and EndAt bound the claim window, and EffectiveAt and IneffectiveAt the voucher's validity once claimed
	StartAt       types.Time `json:"startAt"`
	EndAt         types.Time `json:"endAt"`
	EffectiveAt   types.Time `json:"effectiveAt"`
	IneffectiveAt types.Time `json:"ineffectiveAt"`
	// UsedAmount is only set when the request asks for it
	UsedAmount types.Number `json:"usedAmount"`
}
