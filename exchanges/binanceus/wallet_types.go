package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CoinInfoResponse holds a coin's balances and its deposit and withdrawal networks
type CoinInfoResponse struct {
	Coin              currency.Code `json:"coin"`
	DepositAllEnable  bool          `json:"depositAllEnable"`
	WithdrawAllEnable bool          `json:"withdrawAllEnable"`
	Name              string        `json:"name"`
	Free              types.Number  `json:"free"`
	Locked            types.Number  `json:"locked"`
	Freeze            types.Number  `json:"freeze"`
	Withdrawing       types.Number  `json:"withdrawing"`
	IPOing            types.Number  `json:"ipoing"`
	IPOable           types.Number  `json:"ipoable"`
	Storage           types.Number  `json:"storage"`
	IsLegalMoney      bool          `json:"isLegalMoney"`
	Trading           bool          `json:"trading"`
	NetworkList       []CoinNetwork `json:"networkList"`
}

// CoinNetwork holds a coin's deposit and withdrawal details on one network
type CoinNetwork struct {
	Network                 string        `json:"network"`
	Coin                    currency.Code `json:"coin"`
	WithdrawIntegerMultiple types.Number  `json:"withdrawIntegerMultiple"`
	IsDefault               bool          `json:"isDefault"`
	DepositEnable           bool          `json:"depositEnable"`
	WithdrawEnable          bool          `json:"withdrawEnable"`
	DepositDescription      string        `json:"depositDesc"`
	WithdrawDescription     string        `json:"withdrawDesc"`
	SpecialTips             string        `json:"specialTips"`
	Name                    string        `json:"name"`
	ResetAddressStatus      bool          `json:"resetAddressStatus"`
	AddressRegex            string        `json:"addressRegex"`
	MemoRegex               string        `json:"memoRegex"`
	WithdrawFee             types.Number  `json:"withdrawFee"`
	WithdrawMinimum         types.Number  `json:"withdrawMin"`
	WithdrawMaximum         types.Number  `json:"withdrawMax"`
	MinimumConfirm          uint64        `json:"minConfirm"`
	UnlockConfirm           uint64        `json:"unLockConfirm"`
}

// WithdrawFiatRequest holds the parameters of a fiat withdrawal. Binance.US picks the payment rail from the payment
// method, so a withdrawal takes no payment channel
type WithdrawFiatRequest struct {
	PaymentMethod PaymentMethod
	// PaymentAccount is the linked account paid: the BitGo payment account for BITGO, the bank account's external ID
	// for INTRABANK_TRANSFER, and for WIRE either the 32 hexadecimal character payment method ID or the bank account
	// number of a linked wire payment method
	PaymentAccount string
	FiatCurrency   currency.Code // USD when empty
	Amount         float64       // At least 100 for WIRE
}

// WithdrawFiatResponse holds a submitted fiat withdrawal
type WithdrawFiatResponse struct {
	OrderID      string        `json:"orderId"`
	ChannelCode  string        `json:"channelCode"` // The payment rail, such as BITGO, or WAB for INTRABANK_TRANSFER
	CurrencyCode currency.Code `json:"currencyCode"`
	Amount       types.Number  `json:"amount"`
	OrderStatus  string        `json:"orderStatus"`
}

// WithdrawRequest holds the parameters of a crypto withdrawal
type WithdrawRequest struct {
	Coin            currency.Code
	Network         string // Such as ERC20 or BEP20, which the address must suit
	WithdrawOrderID string // The client's ID for the withdrawal
	Address         string
	AddressTag      string // The memo some coins, such as XRP, need to identify the recipient at the address
	Amount          float64
}

// WithdrawResponse holds the ID of a submitted crypto withdrawal
type WithdrawResponse struct {
	ID string `json:"id"`
}

// WithdrawalHistoryRequest holds the parameters of a crypto withdrawal history request
type WithdrawalHistoryRequest struct {
	Coin            currency.Code
	WithdrawOrderID string
	// Status filters on 0 email sent, 1 cancelled, 2 awaiting approval, 3 rejected, 4 processing, 5 failure or 6
	// completed
	Status    *uint64
	StartTime time.Time // 90 days ago when zero
	EndTime   time.Time // Now when zero
	Offset    uint64
	Limit     uint64 // 1000 when zero, at most 1000
}

// WithdrawalRecordResponse holds a crypto withdrawal
type WithdrawalRecordResponse struct {
	ID             string        `json:"id"`
	Amount         types.Number  `json:"amount"`
	TransactionFee types.Number  `json:"transactionFee"`
	Coin           currency.Code `json:"coin"`
	Status         uint64        `json:"status"`
	Address        string        `json:"address"`
	// ApplyTime is decoded as UTC, as Binance.US does not document its time zone
	ApplyTime    types.DateTime `json:"applyTime"`
	Network      string         `json:"network"`
	TransferType uint64         `json:"transferType"`
}

// FiatHistoryRequest holds the optional filters of a fiat withdrawal or deposit history request. A range with both
// times set must span more than 0 and less than 90 days
type FiatHistoryRequest struct {
	FiatCurrency   currency.Code
	OrderID        string
	Offset         uint64
	PaymentChannel string // The payment rail, such as BITGO or WAB
	PaymentMethod  PaymentMethod
	StartTime      time.Time // 90 days ago when zero
	EndTime        time.Time // Now when zero
}

// FiatHistoryResponse holds fiat withdrawals or deposits
type FiatHistoryResponse struct {
	AssetLogRecordList []FiatAssetLogRecord `json:"assetLogRecordList"`
}

// FiatAssetLogRecord holds a fiat withdrawal or deposit
type FiatAssetLogRecord struct {
	OrderID        string        `json:"orderId"`
	PaymentAccount string        `json:"paymentAccount"`
	PaymentChannel string        `json:"paymentChannel"`
	PaymentMethod  PaymentMethod `json:"paymentMethod"`
	OrderStatus    string        `json:"orderStatus"`
	FiatCurrency   currency.Code `json:"fiatCurrency"` // Documented for deposits only
	Amount         types.Number  `json:"amount"`
	TransactionFee types.Number  `json:"transactionFee"`
	PlatformFee    types.Number  `json:"platformFee"`
}

// DepositAddressResponse holds a deposit address of the account or of a sub-account
type DepositAddressResponse struct {
	Coin    currency.Code `json:"coin"`
	Address string        `json:"address"`
	Tag     string        `json:"tag"` // The memo some coins need to identify the account at the address
	URL     string        `json:"url"` // The address on a blockchain explorer
}

// DepositHistoryRequest holds the parameters of a crypto deposit history request
type DepositHistoryRequest struct {
	Coin      currency.Code
	Status    *uint64   // Filters on 0 pending, 1 success or 6 credited but cannot withdraw
	StartTime time.Time // 90 days ago when zero
	EndTime   time.Time // Now when zero
	Offset    uint64
	Limit     uint64 // 1000 when zero, at most 1000
}

// DepositRecordResponse holds a crypto deposit of the account or of a sub-account
type DepositRecordResponse struct {
	Amount        types.Number  `json:"amount"`
	Coin          currency.Code `json:"coin"`
	Network       string        `json:"network"`
	Status        uint64        `json:"status"`
	Address       string        `json:"address"`
	AddressTag    string        `json:"addressTag"`
	TransactionID string        `json:"txId"`
	InsertTime    types.Time    `json:"insertTime"`
	TransferType  uint64        `json:"transferType"`
	ConfirmTimes  string        `json:"confirmTimes"` // Confirmations, such as 12/12
}

// SubAccountDepositAddressRequest holds the parameters of a sub-account deposit address request
type SubAccountDepositAddressRequest struct {
	Email   string
	Coin    currency.Code
	Network string // The coin's default network when empty
}

// SubAccountDepositHistoryRequest holds the parameters of a sub-account deposit history request
type SubAccountDepositHistoryRequest struct {
	Email     string
	Coin      currency.Code
	Status    *uint64 // Filters on 0 pending, 1 success or 6 credited but cannot withdraw
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Offset    uint64
}
