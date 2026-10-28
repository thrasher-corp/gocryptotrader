package lbank

import (
	"bytes"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SystemStatusResponse holds LBank's system status
type SystemStatusResponse struct {
	// Status is "1" while LBank operates normally and "0" during maintenance
	Status string `json:"status"`
}

// CoinInformationResponse holds the account's balance of a coin and the chains it can be deposited and withdrawn on
type CoinInformationResponse struct {
	Coin            currency.Code `json:"coin"`
	TotalAmount     types.Number  `json:"assetAmt"`
	AvailableAmount types.Number  `json:"usableAmt"`
	FrozenAmount    types.Number  `json:"freezeAmt"`
	Networks        []CoinNetwork `json:"networkList"`
}

// CoinNetwork holds a coin's deposit and withdrawal settings on one chain
type CoinNetwork struct {
	IsDefault bool          `json:"isDefault"`
	Coin      currency.Code `json:"coin"`
	Name      string        `json:"name"`
	Network   string        `json:"network"`
	// FeeType is 1 for a fixed fee, 2 for a rate and 3 for a fixed fee plus a rate
	FeeType           uint64        `json:"type"`
	WithdrawFee       types.Number  `json:"withdrawFee"`
	WithdrawFeeRate   types.Number  `json:"withdrawFeeRate"`
	FeeAssetCode      currency.Code `json:"feeAssetCode"`
	MinimumWithdrawal types.Number  `json:"withdrawMin"`
	MinimumTransfer   types.Number  `json:"minLimit"`
	MinimumDeposit    types.Number  `json:"minDeposit"`
}

// WithdrawRequest holds a withdrawal's parameters
type WithdrawRequest struct {
	// Address is the destination address, or the LBank account an internal transfer goes to
	Address string
	Coin    currency.Code
	Amount  float64
	// Fee must match the chain's withdrawal fee, which GetAllCoinInformation and GetAssetConfigs return
	Fee float64
	// NetworkName is the chain to withdraw on, which defaults to the coin's default chain
	NetworkName string
	Memo        string
	// Mark is a note kept with the withdrawal
	Mark string
	// AddressName adds the address to the coin's address book under this name
	AddressName      string
	WithdrawOrderID  string
	InternalTransfer bool
}

// WithdrawResponse holds a submitted withdrawal
type WithdrawResponse struct {
	WithdrawID uint64       `json:"withdrawId"`
	Fee        types.Number `json:"fee"`
}

// DepositHistoryRequest holds a deposit history request's filters, each optional
type DepositHistoryRequest struct {
	// Status is 1 applying, 2 successful, 3 failed, 4 cancelled or 5 transfer; zero returns every status
	Status    uint64
	Coin      currency.Code
	StartTime time.Time
	EndTime   time.Time
}

// DepositHistoryResponse holds a page of deposits
type DepositHistoryResponse struct {
	Total       uint64          `json:"total"`
	PageLength  uint64          `json:"page_length"`
	CurrentPage uint64          `json:"current_page"`
	Deposits    []DepositRecord `json:"depositOrders"`
}

// UnmarshalJSON decodes a page of deposits. LBank's docs show the deposits as a bare list while captured replies hold
// them in a page object, so both forms are accepted
func (d *DepositHistoryResponse) UnmarshalJSON(data []byte) error {
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
		*d = DepositHistoryResponse{}
		return json.Unmarshal(data, &d.Deposits)
	}
	type page DepositHistoryResponse
	return json.Unmarshal(data, (*page)(d))
}

// DepositRecord holds a deposit
type DepositRecord struct {
	Coin          currency.Code `json:"coin"`
	Amount        types.Number  `json:"amount"`
	Address       string        `json:"address"`
	NetworkName   string        `json:"networkName"`
	TransactionID string        `json:"txId"`
	// Status is "1" applying, "2" successful, "3" failed, "4" cancelled or "5" transfer
	Status     string     `json:"status"`
	InsertTime types.Time `json:"insertTime"`
}

// WithdrawalHistoryRequest holds a withdrawal history request's filters, each optional
type WithdrawalHistoryRequest struct {
	// Status is 1 applying, 2 cancelled, 3 failed or 4 completed; zero returns every status
	Status          uint64
	Coin            currency.Code
	WithdrawOrderID string
	StartTime       time.Time
	EndTime         time.Time
}

// WithdrawalHistoryResponse holds a page of withdrawals
type WithdrawalHistoryResponse struct {
	Total       uint64             `json:"total"`
	PageLength  uint64             `json:"page_length"`
	CurrentPage uint64             `json:"current_page"`
	Withdrawals []WithdrawalRecord `json:"withdraws"`
}

// UnmarshalJSON decodes a page of withdrawals. LBank's docs show the withdrawals as a bare list while captured replies
// hold them in a page object, so both forms are accepted
func (w *WithdrawalHistoryResponse) UnmarshalJSON(data []byte) error {
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
		*w = WithdrawalHistoryResponse{}
		return json.Unmarshal(data, &w.Withdrawals)
	}
	type page WithdrawalHistoryResponse
	return json.Unmarshal(data, (*page)(w))
}

// WithdrawalRecord holds a withdrawal
type WithdrawalRecord struct {
	ID uint64 `json:"id"`
	// Coin is sent under LBank's misspelt "coid"
	Coin            currency.Code `json:"coid"`
	Amount          types.Number  `json:"amount"`
	Fee             types.Number  `json:"fee"`
	FeeAssetCode    currency.Code `json:"feeAssetCode"`
	Address         string        `json:"address"`
	NetworkName     string        `json:"networkName"`
	TransferType    string        `json:"transferType"`
	TransactionID   string        `json:"txId"`
	WithdrawOrderID string        `json:"withdrawOrderId"`
	// Status is "1" applying, "2" cancelled, "3" failed or "4" completed
	Status    string     `json:"status"`
	ApplyTime types.Time `json:"applyTime"`
}

// DepositAddressResponse holds a deposit address
type DepositAddressResponse struct {
	Coin        currency.Code `json:"coin"`
	NetworkName string        `json:"networkName"`
	Address     string        `json:"address"`
	Memo        string        `json:"memo"`
}

// AssetDetailResponse holds a coin's deposit and withdrawal status and withdrawal fee
type AssetDetailResponse struct {
	CanDeposit            bool         `json:"depositStatus"`
	CanWithdraw           bool         `json:"withdrawStatus"`
	CanTransfer           bool         `json:"stationDrawStatus"`
	WithdrawFee           types.Number `json:"withdrawFee"`
	MinimumWithdrawAmount types.Number `json:"minWithdrawAmount"`
}

// TradingFeeRateResponse holds the account's fee rates on a pair
type TradingFeeRateResponse struct {
	Symbol string `json:"symbol"`
	// MakerCommissionPercent and TakerCommissionPercent are percentages: "0.10" is a fee of 0.1%
	MakerCommissionPercent types.Number `json:"makerCommission"`
	TakerCommissionPercent types.Number `json:"takerCommission"`
}

// APIRestrictionsResponse holds an API key's permissions
type APIRestrictionsResponse struct {
	ReadingEnabled     bool       `json:"enableReading"`
	SpotTradingEnabled bool       `json:"enableSpotTrading"`
	WithdrawalsEnabled bool       `json:"enableWithdrawals"`
	IPRestricted       bool       `json:"ipRestrict"`
	CreateTime         types.Time `json:"createTime"`
}
