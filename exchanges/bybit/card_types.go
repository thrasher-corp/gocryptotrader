package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CardAssetRecordsRequest holds the filters of Query Asset Records. Type is required unless TransactionID or
// OrderNumber is set
type CardAssetRecordsRequest struct {
	// StatusCode is 0 for pending, 1 for cleared or 2 for declined transactions
	StatusCode string
	Limit      uint64
	Page       uint64
	// PAN4 is the last 2 or 4 digits of the card number
	PAN4            string
	CreateBeginTime time.Time
	CreateEndTime   time.Time
	// MerchantName matches merchant names fuzzily
	MerchantName string
	// Type is a query type such as SIDE_QUERY_AUTH_ALL or SIDE_QUERY_FINANCIAL_ALL
	Type          string
	TransactionID string
	CardToken     string
	OrderNumber   string
}

// CardAssetRecordsResponse holds a page of card transactions
type CardAssetRecordsResponse struct {
	PageSize   uint64            `json:"pageSize"`
	PageNumber uint64            `json:"pageNo"`
	TotalCount uint64            `json:"totalCount"`
	Data       []CardAssetRecord `json:"data"`
}

// CardAssetRecord is a card transaction. TradeStatus, Side and Status are numeric codes the Query Asset Records page
// lists
type CardAssetRecord struct {
	PAN4                string        `json:"pan4"`
	PAN6                string        `json:"pan6"` // First 6 digits of the card number (BIN)
	TradeStatus         string        `json:"tradeStatus"`
	Side                string        `json:"side"`
	BasicAmount         types.Number  `json:"basicAmount"` // Total amount
	BasicCurrency       currency.Code `json:"basicCurrency"`
	TransactionAmount   types.Number  `json:"transactionAmount"` // Amount before tax
	TransactionCurrency currency.Code `json:"transactionCurrency"`
	TransactionCreate   types.Time    `json:"txnCreate"` // Creation time
	MerchantCountry     string        `json:"merchCountry"`
	MerchantCity        string        `json:"merchCity"`
	MerchantName        string        `json:"merchName"`
	TransactionID       string        `json:"txnId"`
	// DeclinedReason is only set on declined transactions
	DeclinedReason            string       `json:"declinedReason"`
	TotalFees                 types.Number `json:"totalFees"`
	UID                       uint64       `json:"uid"`
	TransactionCurrencyAmount types.Number `json:"transactionCurrencyAmount"` // Fiat transaction amount
	FXPad                     types.Number `json:"fxPad"`                     // Fee
	// InterchangeFee is the card network's fee, which the user does not pay
	InterchangeFee        types.Number  `json:"interchangeFee"`
	BillAmount            types.Number  `json:"billAmount"`
	PaidAmount            types.Number  `json:"paidAmount"` // In the paid currency
	PaidCurrency          currency.Code `json:"paidCurrency"`
	BonusAmount           types.Number  `json:"bonusAmount"` // Trial bonus
	ForeignTransactionFee types.Number  `json:"foreignTransactionFee"`
	TotalTax              types.Number  `json:"totalTax"`
	PaidFiat              types.Number  `json:"paidFiat"`
	WithdrawalFee         types.Number  `json:"withdrawalFee"`
	Status                string        `json:"status"`
	OrderNumber           string        `json:"orderNo"`
	MCCCode               string        `json:"mccCode"` // Merchant category code
	// MerchantCategoryDescription describes the merchant category code
	MerchantCategoryDescription string `json:"merchCategoryDesc"`
}

// CardPointBalanceResponse holds the card reward point balance
type CardPointBalanceResponse struct {
	AccountID string `json:"accountId"`
	// AvailablePoint and PendingPoint are documented as strings but sent as numbers in the docs' example
	AvailablePoint   types.Number `json:"availablePoint"`
	PendingPoint     types.Number `json:"pendingPoint"` // Frozen points
	Status           string       `json:"status"`
	UpdateTime       types.Time   `json:"updateTime"`
	SettlementPeriod uint64       `json:"settlementPeriod"`
}

// CardPointRecordsRequest holds the filters of Query Point Records
type CardPointRecordsRequest struct {
	Type       string
	PageSize   uint64
	PageNumber uint64
	StartTime  time.Time
	EndTime    time.Time
	OutOrderID string
	// BusinessID is the point order ID
	BusinessID            string
	BusinessTransactionID string
	// Side is 1 for points earned or 2 for points deducted
	Side string
}

// CardPointRecordsResponse holds a page of card reward point records
type CardPointRecordsResponse struct {
	PageSize   uint64            `json:"pageSize"`
	PageNumber uint64            `json:"pageNo"`
	TotalCount uint64            `json:"totalCount"`
	Data       []CardPointRecord `json:"data"`
}

// CardPointRecord is a card reward point record
type CardPointRecord struct {
	OutOrderID string     `json:"outOrderId"` // External order ID
	Point      uint64     `json:"point"`
	Side       string     `json:"side"`
	Type       string     `json:"type"`
	SubType    string     `json:"subType"`
	CreateTime types.Time `json:"createTime"`
	UpdateTime types.Time `json:"updateTime"`
	// BusinessID is the point order ID
	BusinessID            string `json:"bizId"`
	BusinessTransactionID string `json:"bizTxnId"`
	// TransactionDate is a formatted date and time such as 2023-01-01 12:00:00, in a time zone the docs do not state
	TransactionDate             string        `json:"transactionDate"`
	TransactionID               string        `json:"transactionId"`
	TransactionAmount           types.Number  `json:"transactionAmount"` // Paid with fiat
	BasicCurrency               currency.Code `json:"basicCurrency"`
	MerchantCategoryDescription string        `json:"merchCategoryDesc"`
	MerchantName                string        `json:"merchName"`
	MerchantCountry             string        `json:"merchCountry"`
	MerchantCity                string        `json:"merchCity"`
	PAN4                        string        `json:"pan4"`
	PayFiatAmount               types.Number  `json:"payFiatAmount"`
	TransactionCurrencyAmount   types.Number  `json:"transactionCurrencyAmount"` // Paid with crypto
}

// CardTierInfoResponse holds the card reward tier and spending limit
type CardTierInfoResponse struct {
	// UsedLimit is points used multiplied by 0.002, in USD to 2 decimal places
	UsedLimit types.Number `json:"usedLimit"`
	Limit     types.Number `json:"limit"`
	// Unit is the limit's unit; the docs describe it as USD and show the code 1
	Unit         string `json:"unit"`
	Tier         string `json:"tier"`
	AutoCashback bool   `json:"autoCashback"`
}

// CardMallItemListRequest holds the filters of Query Mall Item List
type CardMallItemListRequest struct {
	PageNumber uint64
	PageSize   uint64
	// ItemType is 1 for virtual items or 2 for physical items
	ItemType uint64
	// ItemBusinessType is 1 for points or 2 for currency
	ItemBusinessType uint64
	// OrderBy sorts by 1 priority, 2 listing time or 3 price
	OrderBy uint64
	// Ascending sets the sort direction; nil leaves Bybit's default
	Ascending *bool
	// Source is 0 for the default list or 1 for the VIP item list
	Source uint64
}

// CardMallItemListResponse holds a page of card reward mall items
type CardMallItemListResponse struct {
	PageNumber uint64         `json:"pageNo"`
	PageSize   uint64         `json:"pageSize"`
	TotalCount uint64         `json:"totalCount"`
	Data       []CardMallItem `json:"data"`
}

// CardMallItem is an item of the card reward mall
type CardMallItem struct {
	ItemID        string        `json:"itemId"`
	ItemName      string        `json:"itemName"`
	Priority      uint64        `json:"priority"`
	OnTime        types.Time    `json:"onTime"`  // Listing time
	OffTime       types.Time    `json:"offTime"` // Delisting time
	Price         types.Number  `json:"price"`
	DiscountPrice types.Number  `json:"discountPrice"`
	TotalNumber   uint64        `json:"totalNum"`
	RedeemNumber  uint64        `json:"redeemNum"`
	PicturePath   string        `json:"picPath"`
	Currency      currency.Code `json:"currency"`
	// CurrencyType, ItemType and ItemBusinessType are numeric codes for FIAT, CRYPTO or POINT; VIRTUAL or ENTITY; and
	// POINTS or CURRENCY
	CurrencyType     uint64 `json:"currencyType"`
	ItemType         uint64 `json:"itemType"`
	ItemBusinessType uint64 `json:"itemBizType"`
}

// CardCashbackDetailResponse holds the cashback of a card reward order
type CardCashbackDetailResponse struct {
	Points       types.Number  `json:"points"`
	Amount       types.Number  `json:"amt"`
	Currency     currency.Code `json:"ccy"`
	CurrencyType string        `json:"ccyType"`
	// CreateTime is a formatted date and time such as 2023-01-01 12:00:00, in a time zone the docs do not state
	CreateTime            string `json:"createTime"`
	BusinessTransactionID string `json:"bizTxnId"`
	SourceID              uint64 `json:"sourceId"`   // Coupon or voucher ID
	SourceCode            string `json:"sourceCode"` // External order ID
	// OrderStatus and OrderSubStatus are undocumented status codes; signed, since the card API uses -1 for an initial
	// status elsewhere
	OrderStatus        int64  `json:"orderStatus"`
	OrderSubStatus     int64  `json:"orderSubStatus"`
	OrderShowStatus    string `json:"orderShowStatus"`
	FailedBusinessCode string `json:"failedBizCode"`
}
