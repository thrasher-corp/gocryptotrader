package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// BrokerSubAccountAPIPermissionRequest holds a sub-account API key's permissions; every permission is sent, so false
// revokes it
type BrokerSubAccountAPIPermissionRequest struct {
	SubAccountID     string
	SubAccountAPIKey string
	CanTrade         bool
	MarginTrade      bool
	FuturesTrade     bool
}

// BrokerSubAccountAPIPermissionResponse is a sub-account API key's permissions after a change
type BrokerSubAccountAPIPermissionResponse struct {
	SubAccountID string `json:"subaccountId"`
	APIKey       string `json:"apikey"`
	CanTrade     bool   `json:"canTrade"`
	MarginTrade  bool   `json:"marginTrade"`
	FuturesTrade bool   `json:"futuresTrade"`
}

// BrokerSubAccount is a link sub-account and its commissions
type BrokerSubAccount struct {
	SubAccountID    string  `json:"subaccountId"`
	Email           string  `json:"email"`
	Tag             string  `json:"tag"`
	MakerCommission float64 `json:"makerCommission"`
	TakerCommission float64 `json:"takerCommission"`
	// MarginMakerCommission and MarginTakerCommission are -1 while margin is disabled
	MarginMakerCommission float64    `json:"marginMakerCommission"`
	MarginTakerCommission float64    `json:"marginTakerCommission"`
	CreateTime            types.Time `json:"createTime"`
}

// CreateBrokerSubAccountResponse is a created link sub-account
type CreateBrokerSubAccountResponse struct {
	SubAccountID string `json:"subaccountId"`
	Email        string `json:"email"`
	Tag          string `json:"tag"`
}

// BrokerSubAccountAPIKeyRequest holds the parameters of a sub-account API key creation
type BrokerSubAccountAPIKeyRequest struct {
	SubAccountID string
	// PublicKey makes the key an asymmetric Ed25519 or RSA key; an HMAC key is created without it
	PublicKey string
	// CanTrade, the spot trading permission, is always sent
	CanTrade bool
	// MarginTrade and FuturesTrade need margin and futures enabled on the sub-account first
	MarginTrade  bool
	FuturesTrade bool
}

// BrokerSubAccountAPIKeyResponse is a created sub-account API key; SecretKey is returned once, for HMAC keys only
type BrokerSubAccountAPIKeyResponse struct {
	SubAccountID string `json:"subaccountId"`
	APIKey       string `json:"apiKey"`
	SecretKey    string `json:"secretKey"`
	CanTrade     bool   `json:"canTrade"`
	MarginTrade  bool   `json:"marginTrade"`
	FuturesTrade bool   `json:"futuresTrade"`
}

// BrokerSubAccountIPRestrictionResponse is a sub-account API key's trusted IPs after a deletion
type BrokerSubAccountIPRestrictionResponse struct {
	SubAccountID string     `json:"subaccountId"`
	APIKey       string     `json:"apikey"`
	IPList       []string   `json:"ipList"`
	UpdateTime   types.Time `json:"updateTime"`
}

// BrokerEnableFuturesResponse is a sub-account's futures status after enabling futures
type BrokerEnableFuturesResponse struct {
	SubAccountID  string     `json:"subaccountId"`
	EnableFutures bool       `json:"enableFutures"`
	UpdateTime    types.Time `json:"updateTime"`
}

// BrokerBNBBurnMarginInterestResponse is whether a sub-account pays margin interest in BNB
type BrokerBNBBurnMarginInterestResponse struct {
	SubAccountID    uint64 `json:"subAccountId"`
	InterestBNBBurn bool   `json:"interestBNBBurn"`
}

// BrokerBNBBurnSpotResponse is whether a sub-account pays spot and margin trading fees in BNB
type BrokerBNBBurnSpotResponse struct {
	SubAccountID uint64 `json:"subAccountId"`
	SpotBNBBurn  bool   `json:"spotBNBBurn"`
}

// BrokerUniversalTransferPermissionResponse is whether a sub-account API key may make universal transfers
type BrokerUniversalTransferPermissionResponse struct {
	SubAccountID         string `json:"subaccountId"`
	APIKey               string `json:"apikey"`
	CanUniversalTransfer bool   `json:"canUniversalTransfer"`
}

// BrokerBNBBurnStatusResponse is whether a sub-account pays trading fees and margin interest in BNB
type BrokerBNBBurnStatusResponse struct {
	SubAccountID    uint64 `json:"subAccountId"`
	SpotBNBBurn     bool   `json:"spotBNBBurn"`
	InterestBNBBurn bool   `json:"interestBNBBurn"`
}

// LinkAccountInformationResponse holds a link account's commission bounds and sub-account allowance
type LinkAccountInformationResponse struct {
	MaxMakerCommission    float64 `json:"maxMakerCommission"`
	MinMakerCommission    float64 `json:"minMakerCommission"`
	MaxTakerCommission    float64 `json:"maxTakerCommission"`
	MinTakerCommission    float64 `json:"minTakerCommission"`
	SubAccountQuantity    uint64  `json:"subAccountQty"`
	MaxSubAccountQuantity uint64  `json:"maxSubAccountQty"`
}

// BrokerIPRestrictionUpdateRequest holds the parameters of a sub-account API key IP restriction update
type BrokerIPRestrictionUpdateRequest struct {
	SubAccountID     string
	SubAccountAPIKey string
	// Status is 1 for unrestricted access or 2 to restrict access to trusted IPs
	Status uint64
	// IPAddress is a comma separated list of trusted IPs to add
	IPAddress string
}

// BrokerIPRestrictionUpdateResponse is a sub-account API key's IP restriction after an update
type BrokerIPRestrictionUpdateResponse struct {
	// Status is 1 for unrestricted access or 2 for access from trusted IPs only
	Status     string     `json:"status"`
	IPList     []string   `json:"ipList"`
	UpdateTime types.Time `json:"updateTime"`
	APIKey     string     `json:"apiKey"`
}

// BrokerSubAccountDepositHistoryRequest holds the parameters of Get Sub Account Deposit History
type BrokerSubAccountDepositHistoryRequest struct {
	SubAccountID string
	Coin         currency.Code
	// Status is 0 pending, 6 credited but cannot withdraw or 1 success; it is a string so 0 can be sent, and every
	// status is returned when it is empty
	Status string
	// StartTime and EndTime are at most 7 days apart
	StartTime time.Time
	EndTime   time.Time
	// Limit is at most 500, the default
	Limit  uint64
	Offset uint64
}

// BrokerSubAccountDeposit is a link sub-account deposit
type BrokerSubAccountDeposit struct {
	DepositID     uint64        `json:"depositId"`
	SubAccountID  string        `json:"subAccountId"`
	Address       string        `json:"address"`
	AddressTag    string        `json:"addressTag"`
	Amount        types.Number  `json:"amount"`
	Coin          currency.Code `json:"coin"`
	InsertTime    types.Time    `json:"insertTime"`
	TransferType  uint64        `json:"transferType"`
	Network       string        `json:"network"`
	Status        uint64        `json:"status"`
	TransactionID string        `json:"txId"`
	SourceAddress string        `json:"sourceAddress"`
	// ConfirmTimes is the confirmations so far and required, such as 12/12
	ConfirmTimes     string `json:"confirmTimes"`
	SelfReturnStatus uint64 `json:"selfReturnStatus"`
	// TravelRuleStatus is 0 when the funds are ready to use and 1 when the travel rule needs more deposit information
	TravelRuleStatus uint64 `json:"travelRuleStatus"`
}

// BrokerFuturesAssetInfoRequest holds the parameters of Query Sub Account Futures Asset Info (V3)
type BrokerFuturesAssetInfoRequest struct {
	SubAccountID string
	// CoinMargined selects COIN-M futures (futuresType 2) instead of USDT-M futures (futuresType 1)
	CoinMargined bool
	Page         uint64
	// Size is at most 20 and defaults to 10
	Size uint64
}

// BrokerFuturesAssetInfoResponse holds link sub-accounts' futures assets
type BrokerFuturesAssetInfoResponse struct {
	Data      []BrokerFuturesAsset `json:"data"`
	Timestamp types.Time           `json:"timestamp"`
}

// BrokerFuturesAsset is a link sub-account's futures assets; the USDT valuations are returned for COIN-M futures
type BrokerFuturesAsset struct {
	SubAccountID                string        `json:"subAccountId"`
	TotalInitialMargin          types.Number  `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number  `json:"totalMaintenanceMargin"`
	TotalWalletBalance          types.Number  `json:"totalWalletBalance"`
	TotalUnrealizedProfit       types.Number  `json:"totalUnrealizedProfit"`
	TotalMarginBalance          types.Number  `json:"totalMarginBalance"`
	TotalPositionInitialMargin  types.Number  `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin types.Number  `json:"totalOpenOrderInitialMargin"`
	FuturesEnable               bool          `json:"futuresEnable"`
	Asset                       currency.Code `json:"asset"`
	TotalWalletBalanceOfUSDT    types.Number  `json:"totalWalletBalanceOfUsdt"`
	TotalUnrealizedProfitOfUSDT types.Number  `json:"totalUnrealizedProfitOfUsdt"`
	TotalMarginBalanceOfUSDT    types.Number  `json:"totalMarginBalanceOfUsdt"`
}

// BrokerMarginAssetInfoResponse holds link sub-accounts' margin assets
type BrokerMarginAssetInfoResponse struct {
	Data      []BrokerMarginAsset `json:"data"`
	Timestamp types.Time          `json:"timestamp"`
}

// BrokerMarginAsset is a link sub-account's margin assets valued in BTC
type BrokerMarginAsset struct {
	MarginEnable        bool         `json:"marginEnable"`
	SubAccountID        string       `json:"subAccountId"`
	TotalAssetOfBTC     types.Number `json:"totalAssetOfBtc"`
	TotalLiabilityOfBTC types.Number `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC  types.Number `json:"totalNetAssetOfBtc"`
	MarginLevel         types.Number `json:"marginLevel"`
}

// BrokerSpotAssetInfoResponse holds link sub-accounts' spot balances
type BrokerSpotAssetInfoResponse struct {
	Data      []BrokerSpotAsset `json:"data"`
	Timestamp types.Time        `json:"timestamp"`
}

// BrokerSpotAsset is a link sub-account's spot balance valued in BTC
type BrokerSpotAsset struct {
	SubAccountID      string       `json:"subAccountId"`
	TotalBalanceOfBTC types.Number `json:"totalBalanceOfBtc"`
}

// BrokerFuturesTransferHistoryRequest holds the parameters of Query Sub Account Transfer History（FUTURES）
type BrokerFuturesTransferHistoryRequest struct {
	SubAccountID string
	// CoinMargined selects COIN-M futures (futuresType 2) instead of USDT-M futures (futuresType 1)
	CoinMargined     bool
	ClientTransferID string
	StartTime        time.Time
	EndTime          time.Time
	Page             uint64
	// Limit is at most 500 and defaults to 50
	Limit uint64
}

// BrokerFuturesTransferHistoryResponse holds a link sub-account's futures transfers
type BrokerFuturesTransferHistoryResponse struct {
	Success     bool                    `json:"success"`
	FuturesType uint64                  `json:"futuresType"`
	Transfers   []BrokerFuturesTransfer `json:"transfers"`
}

// BrokerFuturesTransfer is a futures transfer between link accounts
type BrokerFuturesTransfer struct {
	From             string        `json:"from"`
	To               string        `json:"to"`
	Asset            currency.Code `json:"asset"`
	Quantity         types.Number  `json:"qty"`
	TransferID       string        `json:"tranId"`
	ClientTransferID string        `json:"clientTranId"`
	Time             types.Time    `json:"time"`
	FromID           string        `json:"fromId"`
	ToID             string        `json:"toId"`
}

// BrokerFuturesTransferRequest holds the parameters of a futures transfer between link accounts
type BrokerFuturesTransferRequest struct {
	// FromID and ToID are sub-account IDs; an empty one is the master account
	FromID string
	ToID   string
	// CoinMargined selects COIN-M futures (futuresType 2) instead of USDT-M futures (futuresType 1)
	CoinMargined bool
	Asset        currency.Code
	Amount       float64
	// ClientTransferID is at most 32 characters
	ClientTransferID string
}

// BrokerFuturesTransferResponse is a futures transfer between link accounts
type BrokerFuturesTransferResponse struct {
	Success          bool   `json:"success"`
	TransactionID    string `json:"txnId"`
	ClientTransferID string `json:"clientTranId"`
}

// BrokerSpotTransferHistoryRequest holds the parameters of Query Sub Account Transfer History（SPOT）
type BrokerSpotTransferHistoryRequest struct {
	// FromID or ToID is required
	FromID           string
	ToID             string
	ClientTransferID string
	// ShowAllStatus includes FAILURE transfers alongside INIT, PROCESS and SUCCESS
	ShowAllStatus bool
	StartTime     time.Time
	EndTime       time.Time
	Page          uint64
	// Limit is at most 500, the default
	Limit uint64
}

// BrokerSpotTransfer is a spot transfer between link accounts
type BrokerSpotTransfer struct {
	FromID           string        `json:"fromId"`
	ToID             string        `json:"toId"`
	Asset            currency.Code `json:"asset"`
	Quantity         types.Number  `json:"qty"`
	Time             types.Time    `json:"time"`
	TransactionID    string        `json:"txnId"`
	ClientTransferID string        `json:"clientTranId"`
	Status           string        `json:"status"`
}

// BrokerSpotTransferRequest holds the parameters of a spot transfer between link accounts
type BrokerSpotTransferRequest struct {
	// FromID and ToID are sub-account IDs; an empty one is the master account
	FromID string
	ToID   string
	Asset  currency.Code
	Amount float64
	// ClientTransferID must be unique and at most 32 characters
	ClientTransferID string
}

// BrokerSpotTransferResponse is a spot transfer between link accounts
type BrokerSpotTransferResponse struct {
	TransactionID    string `json:"txnId"`
	ClientTransferID string `json:"clientTranId"`
}

// BrokerUniversalTransferHistoryRequest holds the parameters of Query Universal Transfer History
type BrokerUniversalTransferHistoryRequest struct {
	// FromID or ToID is required
	FromID           string
	ToID             string
	ClientTransferID string
	StartTime        time.Time
	EndTime          time.Time
	Page             uint64
	// Limit is at most 500, the default
	Limit uint64
	// ShowAllStatus includes FAILURE transfers alongside INIT, PROCESS and SUCCESS
	ShowAllStatus bool
}

// BrokerUniversalTransfer is a universal transfer between link accounts; the master account's ID is never returned
type BrokerUniversalTransfer struct {
	FromID           string        `json:"fromId"`
	ToID             string        `json:"toId"`
	Asset            currency.Code `json:"asset"`
	Quantity         types.Number  `json:"qty"`
	Time             types.Time    `json:"time"`
	Status           string        `json:"status"`
	TransactionID    string        `json:"txnId"`
	ClientTransferID string        `json:"clientTranId"`
	FromAccountType  string        `json:"fromAccountType"`
	ToAccountType    string        `json:"toAccountType"`
}

// BrokerUniversalTransferRequest holds the parameters of a universal transfer between link accounts
type BrokerUniversalTransferRequest struct {
	// FromID and ToID are sub-account IDs; an empty one is the master account
	FromID string
	ToID   string
	// FromAccountType and ToAccountType are SPOT, USDT_FUTURE or COIN_FUTURE
	FromAccountType string
	ToAccountType   string
	// ClientTransferID must be unique and at most 32 characters
	ClientTransferID string
	Asset            currency.Code
	Amount           float64
}

// BrokerUniversalTransferResponse is a universal transfer between link accounts
type BrokerUniversalTransferResponse struct {
	TransactionID    uint64 `json:"txnId"`
	ClientTransferID string `json:"clientTranId"`
}

// BrokerCoinFuturesCommission is a link sub-account's COIN-M futures commission on a pair, where 100 is 0.01%
type BrokerCoinFuturesCommission struct {
	SubAccountID    uint64 `json:"subAccountId"`
	Pair            string `json:"pair"`
	MakerCommission int64  `json:"makerCommission"`
	TakerCommission int64  `json:"takerCommission"`
}

// BrokerCoinFuturesCommissionAdjustmentRequest holds a link sub-account's COIN-M futures commission adjustment, where 100 is
// 0.01%
type BrokerCoinFuturesCommissionAdjustmentRequest struct {
	SubAccountID string
	// Pair is a COIN-M pair such as BTCUSD
	Pair string
	// MakerAdjustment is between 0 and 100, and TakerAdjustment between 0 and 500; both are always sent
	MakerAdjustment uint64
	TakerAdjustment uint64
}

// BrokerCoinFuturesCommissionAdjustmentResponse is a link sub-account's COIN-M futures commission after an adjustment, where
// 100 is 0.01%
type BrokerCoinFuturesCommissionAdjustmentResponse struct {
	SubAccountID    uint64 `json:"subAccountId"`
	Pair            string `json:"pair"`
	MakerAdjustment uint64 `json:"makerAdjustment"`
	TakerAdjustment uint64 `json:"takerAdjustment"`
	MakerCommission int64  `json:"makerCommission"`
	TakerCommission int64  `json:"takerCommission"`
}

// BrokerCommissionRequest holds a link sub-account's spot and margin commissions as decimal fractions between
// 0.001 and 0.002
type BrokerCommissionRequest struct {
	SubAccountID    string
	MakerCommission float64
	TakerCommission float64
	// MarginMakerCommission and MarginTakerCommission default to the spot commissions
	MarginMakerCommission float64
	MarginTakerCommission float64
}

// BrokerCommissionResponse is a link sub-account's spot and margin commissions after a change
type BrokerCommissionResponse struct {
	SubAccountID          string  `json:"subAccountId"`
	MakerCommission       float64 `json:"makerCommission"`
	TakerCommission       float64 `json:"takerCommission"`
	MarginMakerCommission float64 `json:"marginMakerCommission"`
	MarginTakerCommission float64 `json:"marginTakerCommission"`
}

// BrokerUSDTFuturesCommission is a link sub-account's USDT-M futures commission on a symbol, where 100 is 0.01%
type BrokerUSDTFuturesCommission struct {
	SubAccountID    uint64 `json:"subAccountId"`
	Symbol          string `json:"symbol"`
	MakerCommission int64  `json:"makerCommission"`
	TakerCommission int64  `json:"takerCommission"`
}

// BrokerUSDTFuturesCommissionAdjustmentRequest holds a link sub-account's USDT-M futures commission adjustment, where 100 is
// 0.01%
type BrokerUSDTFuturesCommissionAdjustmentRequest struct {
	SubAccountID string
	Symbol       currency.Pair
	// MakerAdjustment is between 0 and 200, and TakerAdjustment between 0 and 400; both are always sent
	MakerAdjustment uint64
	TakerAdjustment uint64
}

// BrokerUSDTFuturesCommissionAdjustmentResponse is a link sub-account's USDT-M futures commission after an adjustment, where
// 100 is 0.01%
type BrokerUSDTFuturesCommissionAdjustmentResponse struct {
	SubAccountID    uint64 `json:"subAccountId"`
	Symbol          string `json:"symbol"`
	MakerAdjustment uint64 `json:"makerAdjustment"`
	TakerAdjustment uint64 `json:"takerAdjustment"`
	MakerCommission int64  `json:"makerCommission"`
	TakerCommission int64  `json:"takerCommission"`
}

// BrokerSpotCommissionRebateRequest holds the parameters of Query Spot Commission Rebate Recent Record
type BrokerSpotCommissionRebateRequest struct {
	SubAccountID string
	// StartTime and EndTime are at most 7 days apart
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	// Size is at most 500, the default
	Size uint64
}

// BrokerCommissionRebate is a commission rebate a link account earned
type BrokerCommissionRebate struct {
	SubAccountID string        `json:"subaccountId"`
	Income       types.Number  `json:"income"`
	Asset        currency.Code `json:"asset"`
	Symbol       string        `json:"symbol"`
	TradeID      uint64        `json:"tradeId"`
	Time         types.Time    `json:"time"`
	Status       uint64        `json:"status"`
}

// BrokerFuturesCommissionRebateRequest holds the parameters of Query Broker Futures Commission Rebate Record
type BrokerFuturesCommissionRebateRequest struct {
	// CoinMargined selects COIN-M futures (futuresType 2) instead of USDT-M futures (futuresType 1)
	CoinMargined bool
	// StartTime and EndTime are required
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	// Size is at most 100 and defaults to 10
	Size uint64
	// FilterResult filters out rebates not from the account's own sub-accounts
	FilterResult bool
}
