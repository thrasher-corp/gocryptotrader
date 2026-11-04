package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// GetExchangeBrokerRebateRequest holds the Exchange Broker Get Broker Rebate parameters, all of them required
type GetExchangeBrokerRebateRequest struct {
	// Begin and End are sent as UTC days and may be at most six months apart
	Begin time.Time
	End   time.Time
	// TradeType is 1 (spot) or 2 (futures)
	TradeType string
}

// GetExchangeBrokerRebateV3Request holds the Get Broker RebateV3 parameters, all of them required
type GetExchangeBrokerRebateV3Request struct {
	// Begin and End are sent as UTC days
	Begin time.Time
	End   time.Time
	// TradeType is SPOT or FUTURE
	TradeType string
}

// SubmitExchangeBrokerKYCRequest holds the KYC information Submit KYC sends for an Exchange Broker sub-account. Every
// field is required except FrontPhoto, which a bvn identity may omit, and BackendPhoto, which a passport or bvn
// identity may omit
type SubmitExchangeBrokerKYCRequest struct {
	ClientUID string
	FirstName string
	LastName  string
	// IssueCountry is the ISO country code of the identity document
	IssueCountry string
	// BirthDate and ExpireDate are sent as UTC days; an identity document without an expiry date uses 2099-01-01
	BirthDate time.Time
	// IdentityType is idcard, drivinglicense, passport or bvn
	IdentityType   string
	IdentityNumber string
	ExpireDate     time.Time
	// FrontPhoto, BackendPhoto and FacePhoto are base64 images of at most 2MB, preferably PNG or JPG; the identity
	// photos must be of the original document, not a scanned copy, and FacePhoto is a selfie
	FrontPhoto   string
	BackendPhoto string
	FacePhoto    string
}

// ExchangeBrokerKYCStatusResponse is the KYC status of an Exchange Broker sub-account
type ExchangeBrokerKYCStatusResponse struct {
	ClientUID uint64 `json:"clientUid"`
	// Status is NONE (not submitted), PROCESS (in progress), PASS or REJECT
	Status       string `json:"status"`
	RejectReason string `json:"rejectReason"`
}

// ExchangeBrokerKYCStatusListResponse holds a page of the KYC statuses of Exchange Broker sub-accounts
type ExchangeBrokerKYCStatusListResponse struct {
	CurrentPage uint64                            `json:"currentPage"`
	PageSize    uint64                            `json:"pageSize"`
	TotalNumber uint64                            `json:"totalNum"`
	TotalPage   uint64                            `json:"totalPage"`
	Items       []ExchangeBrokerKYCStatusResponse `json:"items"`
}

// GetExchangeBrokerInfoRequest holds the Get Broker Info parameters, all of them required
type GetExchangeBrokerInfoRequest struct {
	// Begin and End are sent as UTC days and may be at most six months apart
	Begin time.Time
	End   time.Time
	// TradeType is 1 (spot) or 2 (futures)
	TradeType string
}

// ExchangeBrokerInfoResponse holds the Exchange Broker's level and sub-account quota
type ExchangeBrokerInfoResponse struct {
	AccountSize uint64 `json:"accountSize"`
	// MaxAccountSize is zero when the API returns null, which means there is no limit
	MaxAccountSize uint64 `json:"maxAccountSize"`
	Level          uint64 `json:"level"`
}

// AddExchangeBrokerSubAccountResponse is the sub-account Add sub-account created
type AddExchangeBrokerSubAccountResponse ExchangeBrokerSubAccountResponse

// GetExchangeBrokerSubAccountsRequest holds the optional Get sub-account parameters
type GetExchangeBrokerSubAccountsRequest struct {
	UID         string
	CurrentPage uint64
	// PageSize is at most 100; the API defaults to 20
	PageSize uint64
}

// ExchangeBrokerSubAccountsResponse holds a page of the sub-accounts an Exchange Broker created
type ExchangeBrokerSubAccountsResponse struct {
	CurrentPage uint64                             `json:"currentPage"`
	PageSize    uint64                             `json:"pageSize"`
	TotalNumber uint64                             `json:"totalNum"`
	TotalPage   uint64                             `json:"totalPage"`
	Items       []ExchangeBrokerSubAccountResponse `json:"items"`
}

// ExchangeBrokerSubAccountResponse is a sub-account an Exchange Broker created
type ExchangeBrokerSubAccountResponse struct {
	AccountName string     `json:"accountName"`
	UID         string     `json:"uid"`
	CreatedAt   types.Time `json:"createdAt"`
	// Level is the sub-account's VIP level
	Level uint64 `json:"level"`
}

// AddExchangeBrokerSubAccountAPIRequest holds the Add sub-account API parameters; UID, Passphrase and Label are
// required
type AddExchangeBrokerSubAccountAPIRequest struct {
	UID        string
	Passphrase string
	// IPWhitelist holds up to 20 IP addresses; it is required when Permissions includes Transfer
	IPWhitelist []string
	// Permissions are General, Spot, Margin, Futures, InnerTransfer, Transfer or Earn; the API defaults to General
	Permissions []string
	// Label is a remark of 4 to 32 characters
	Label string
}

// AddExchangeBrokerSubAccountAPIResponse is the API key Add sub-account API created, with its secret
type AddExchangeBrokerSubAccountAPIResponse struct {
	UID         string     `json:"uid"`
	Label       string     `json:"label"`
	APIKey      string     `json:"apiKey"`
	SecretKey   string     `json:"secretKey"`
	APIVersion  uint64     `json:"apiVersion"`
	Permissions []string   `json:"permissions"`
	IPWhitelist []string   `json:"ipWhitelist"`
	CreatedAt   types.Time `json:"createdAt"`
}

// ExchangeBrokerSubAccountAPIResponse is an API key of an Exchange Broker sub-account
type ExchangeBrokerSubAccountAPIResponse struct {
	UID         string     `json:"uid"`
	Label       string     `json:"label"`
	APIKey      string     `json:"apiKey"`
	APIVersion  uint64     `json:"apiVersion"`
	Permissions []string   `json:"permissions"`
	IPWhitelist []string   `json:"ipWhitelist"`
	CreatedAt   types.Time `json:"createdAt"`
}

// ModifyExchangeBrokerSubAccountAPIRequest holds the Modify sub-account API parameters; UID, APIKey and Label are
// required
type ModifyExchangeBrokerSubAccountAPIRequest struct {
	UID    string
	APIKey string
	// IPWhitelist holds up to 20 IP addresses; it is required when Permissions includes Transfer
	IPWhitelist []string
	// Permissions are General, Spot, Margin, Futures, InnerTransfer, Transfer or Earn; the API defaults to General
	Permissions []string
	// Label is a remark of 4 to 32 characters
	Label string
}

// ModifyExchangeBrokerSubAccountAPIResponse is the API key Modify sub-account API changed
type ModifyExchangeBrokerSubAccountAPIResponse ExchangeBrokerSubAccountAPIResponse

// ExchangeBrokerTransferRequest holds the Exchange Broker Transfer parameters, all of them required
type ExchangeBrokerTransferRequest struct {
	Currency currency.Code
	Amount   float64
	// Direction is OUT (from the broker account to the sub-account) or IN (from the sub-account to the broker
	// account)
	Direction string
	// AccountType and SpecialAccountType, of the broker account and the sub-account, are MAIN (funding) or TRADE
	// (spot)
	AccountType string
	// SpecialUID is the UID of a sub-account the broker created
	SpecialUID         string
	SpecialAccountType string
	// ClientOrderID is a unique ID of at most 128 bits, preferably a UUID
	ClientOrderID string
}

// ExchangeBrokerTransferResponse identifies an Exchange Broker transfer
type ExchangeBrokerTransferResponse struct {
	OrderID string `json:"orderId"`
}

// ExchangeBrokerTransferHistoryResponse is a transfer of an Exchange Broker or one of its sub-accounts
type ExchangeBrokerTransferHistoryResponse struct {
	OrderID  string        `json:"orderId"`
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount"`
	FromUID  uint64        `json:"fromUid"`
	// FromAccountType and ToAccountType are MAIN, TRADE, CONTRACT, MARGIN or ISOLATED
	FromAccountType string `json:"fromAccountType"`
	// FromAccountTag and ToAccountTag are the trading pair of an ISOLATED account
	FromAccountTag string `json:"fromAccountTag"`
	ToUID          uint64 `json:"toUid"`
	ToAccountType  string `json:"toAccountType"`
	ToAccountTag   string `json:"toAccountTag"`
	// Status is PROCESSING, SUCCESS or FAILURE
	Status    string     `json:"status"`
	Reason    string     `json:"reason"`
	CreatedAt types.Time `json:"createdAt"`
}

// GetExchangeBrokerDepositListRequest holds the optional Exchange Broker Get Deposit List parameters
type GetExchangeBrokerDepositListRequest struct {
	Currency currency.Code
	// Status is PROCESSING, SUCCESS, FAILURE, PRE_SUCCESS, WAIT_TRM_MGT, TRM_MGT_REJECTED, ROLLBACKING, ROLLBACK,
	// WAIT_RISK_MGT or RISK_MGT_REJECTED
	Status         string
	Hash           string
	StartTimestamp time.Time
	EndTimestamp   time.Time
	// Limit is at most 1000, which is the API's default
	Limit uint64
}

// ExchangeBrokerDepositResponse is a deposit to an Exchange Broker sub-account
type ExchangeBrokerDepositResponse struct {
	// UID is typed as a number in the Get Deposit List schema and as an integer in the Get Deposit Detail schema; both
	// examples are integers
	UID     uint64 `json:"uid"`
	Hash    string `json:"hash"`
	Address string `json:"address"`
	// Memo is the address remark, empty when the address has none
	Memo       string        `json:"memo"`
	Amount     types.Number  `json:"amount"`
	Fee        types.Number  `json:"fee"`
	Currency   currency.Code `json:"currency"`
	IsInner    bool          `json:"isInner"`
	WalletTxID string        `json:"walletTxId"`
	// Status is one of the GetExchangeBrokerDepositListRequest Status values
	Status       string     `json:"status"`
	Remark       string     `json:"remark"`
	Chain        string     `json:"chain"`
	CreatedAt    types.Time `json:"createdAt"`
	UpdatedAt    types.Time `json:"updatedAt"`
	StatusRemark string     `json:"statusRemark"`
	// PreConfirms is the minimum number of confirmations before the balance is credited
	PreConfirms uint64 `json:"preConfirms"`
	// Confirms is the number of confirmations that unlocks the balance
	Confirms        uint64 `json:"confirms"`
	CurrentConfirms uint64 `json:"currentConfirms"`
}

// ExchangeBrokerDepositDetailResponse is a deposit to an Exchange Broker sub-account
type ExchangeBrokerDepositDetailResponse ExchangeBrokerDepositResponse

// ExchangeBrokerWithdrawDetailResponse is a withdrawal from an Exchange Broker sub-account
type ExchangeBrokerWithdrawDetailResponse struct {
	ID         string        `json:"id"`
	Chain      string        `json:"chain"`
	WalletTxID string        `json:"walletTxId"`
	UID        uint64        `json:"uid"`
	UpdatedAt  types.Time    `json:"updatedAt"`
	Amount     types.Number  `json:"amount"`
	Memo       string        `json:"memo"`
	Fee        types.Number  `json:"fee"`
	Address    string        `json:"address"`
	Remark     string        `json:"remark"`
	IsInner    bool          `json:"isInner"`
	Currency   currency.Code `json:"currency"`
	// Status is PROCESSING, WALLET_PROCESSING, REVIEW, SUCCESS or FAILURE
	Status               string     `json:"status"`
	CreatedAt            types.Time `json:"createdAt"`
	FailureReason        string     `json:"failureReason"`
	FailureReasonMessage string     `json:"failureReasonMsg"`
}

// SetExchangeBrokerMarkupFeeRequest holds the Set Markup Fee parameters; TradeType, EffectiveType and both mark-ups
// are required. Setting both mark-ups to zero cancels the mark-up
type SetExchangeBrokerMarkupFeeRequest struct {
	// SubUIDs lists up to 20 sub-account UIDs, used when EffectiveType is UID_SPECIFIC
	SubUIDs []uint64
	// EffectAt defaults to the next day's 00:10 UTC+8
	EffectAt time.Time
	// TradeType is SPOT or FUTURES
	TradeType string
	// MakerMarkUp and TakerMarkUp are absolute mark-ups between 0 and 0.03 for spot, or between 0 and 0.001 for
	// futures. For spot the taker mark-up must be at least the maker mark-up; for futures the maker mark-up may
	// exceed the taker mark-up by at most 0.025%
	MakerMarkUp float64
	TakerMarkUp float64
	// EffectiveType is DEFAULT (every sub-account) or UID_SPECIFIC
	EffectiveType string
}

// SetExchangeBrokerMarkupFeeResponse is the mark-up Set Markup Fee configured
type SetExchangeBrokerMarkupFeeResponse struct {
	SubUIDs  []uint64   `json:"subUids"`
	EffectAt types.Time `json:"effectAt"`
	// CreatedAt is when the request was received
	CreatedAt     types.Time   `json:"createdAt"`
	TradeType     string       `json:"tradeType"`
	MakerMarkUp   types.Number `json:"makerMarkUp"`
	TakerMarkUp   types.Number `json:"takerMarkUp"`
	EffectiveType string       `json:"effectiveType"`
}

// ExchangeBrokerMarkupFeeResponse holds the Exchange Broker's default mark-up and those of the sub-accounts queried
type ExchangeBrokerMarkupFeeResponse struct {
	DefaultMarkUp ExchangeBrokerDefaultMarkUp `json:"defaultMarkUp"`
	// Items is sorted by UID and empty when no valid sub-account UID was queried
	Items []ExchangeBrokerSubAccountMarkUp `json:"items"`
}

// ExchangeBrokerDefaultMarkUp is an Exchange Broker's default mark-up; a mark-up that is not set is zero, as the API
// returns an empty string
type ExchangeBrokerDefaultMarkUp struct {
	UpdatedAt          types.Time   `json:"updatedAt"`
	SpotMakerMarkUp    types.Number `json:"spotMakerMarkUp"`
	SpotTakerMarkUp    types.Number `json:"spotTakerMarkUp"`
	FuturesMakerMarkUp types.Number `json:"futuresMakerMarkUp"`
	FuturesTakerMarkUp types.Number `json:"futuresTakerMarkUp"`
}

// ExchangeBrokerSubAccountMarkUp is a sub-account's mark-up; a mark-up that is not set is zero, as the API returns an
// empty string
type ExchangeBrokerSubAccountMarkUp struct {
	UID                uint64       `json:"uid"`
	UpdatedAt          types.Time   `json:"updatedAt"`
	SpotMakerMarkUp    types.Number `json:"spotMakerMarkUp"`
	SpotTakerMarkUp    types.Number `json:"spotTakerMarkUp"`
	FuturesMakerMarkUp types.Number `json:"futuresMakerMarkUp"`
	FuturesTakerMarkUp types.Number `json:"futuresTakerMarkUp"`
}
