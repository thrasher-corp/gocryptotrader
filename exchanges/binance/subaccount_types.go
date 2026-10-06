package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CreateVirtualSubAccountResponse holds the email of a created virtual sub-account
type CreateVirtualSubAccountResponse struct {
	Email string `json:"email"`
}

// EnableFuturesSubAccountResponse holds a sub-account's futures status after enabling futures
type EnableFuturesSubAccountResponse struct {
	Email            string `json:"email"`
	IsFuturesEnabled bool   `json:"isFuturesEnabled"`
}

// EnableOptionsForSubAccountResponse holds a sub-account's options status after enabling options
type EnableOptionsForSubAccountResponse struct {
	Email             string `json:"email"`
	IsEOptionsEnabled bool   `json:"isEOptionsEnabled"`
}

// SubAccountFuturesPositionRisk holds a USDⓈ-M futures position of a sub-account
type SubAccountFuturesPositionRisk struct {
	EntryPrice       types.Number `json:"entryPrice"`
	Leverage         types.Number `json:"leverage"`
	MaxNotional      types.Number `json:"maxNotional"`
	LiquidationPrice types.Number `json:"liquidationPrice"`
	MarkPrice        types.Number `json:"markPrice"`
	PositionAmount   types.Number `json:"positionAmount"`
	Symbol           string       `json:"symbol"`
	UnrealisedProfit types.Number `json:"unrealizedProfit"`
}

// SubAccountFuturesPositionRiskV2Response holds a sub-account's futures positions; which list is set depends on the
// futures type queried
type SubAccountFuturesPositionRiskV2Response struct {
	FuturePositionRiskVos   []SubAccountFuturesPositionRisk  `json:"futurePositionRiskVos"`
	DeliveryPositionRiskVos []SubAccountDeliveryPositionRisk `json:"deliveryPositionRiskVos"`
}

// SubAccountDeliveryPositionRisk holds a COIN-M futures position of a sub-account
type SubAccountDeliveryPositionRisk struct {
	EntryPrice       types.Number  `json:"entryPrice"`
	MarkPrice        types.Number  `json:"markPrice"`
	Leverage         types.Number  `json:"leverage"`
	Isolated         types.Boolean `json:"isolated"` // Arrives as a quoted boolean
	IsolatedWallet   types.Number  `json:"isolatedWallet"`
	IsolatedMargin   types.Number  `json:"isolatedMargin"`
	IsAutoAddMargin  types.Boolean `json:"isAutoAddMargin"` // Arrives as a quoted boolean
	PositionSide     string        `json:"positionSide"`
	PositionAmount   types.Number  `json:"positionAmount"`
	Symbol           string        `json:"symbol"`
	UnrealisedProfit types.Number  `json:"unrealizedProfit"`
}

// SubAccountStatus holds a sub-account's margin and futures status
type SubAccountStatus struct {
	Email            string     `json:"email"`
	IsSubUserEnabled bool       `json:"isSubUserEnabled"`
	IsUserActive     bool       `json:"isUserActive"`
	InsertTime       types.Time `json:"insertTime"`
	IsMarginEnabled  bool       `json:"isMarginEnabled"`
	IsFutureEnabled  bool       `json:"isFutureEnabled"`
	Mobile           uint64     `json:"mobile"`
}

// SubAccountListRequest holds the parameters for a sub-account list query
type SubAccountListRequest struct {
	Email    string
	IsFreeze *bool // Filters by frozen status when set
	Page     uint64
	Limit    uint64 // Up to 200, default 1
}

// SubAccountListResponse holds sub-accounts
type SubAccountListResponse struct {
	SubAccounts []SubAccount `json:"subAccounts"`
	Success     bool         `json:"success"` // Undocumented, but present in live responses
}

// SubAccount holds a sub-account of the master account
type SubAccount struct {
	SubUserID                   uint64     `json:"subUserId"`
	Email                       string     `json:"email"`
	Remark                      string     `json:"remark"`
	IsFreeze                    bool       `json:"isFreeze"`
	CreateTime                  types.Time `json:"createTime"`
	IsManagedSubAccount         bool       `json:"isManagedSubAccount"`
	IsAssetManagementSubAccount bool       `json:"isAssetManagementSubAccount"`
}

// SubAccountTransactionStatisticsResponse holds a sub-account's trading volumes of the last 30 days
type SubAccountTransactionStatisticsResponse struct {
	Recent30DayBTCTotal         types.Number          `json:"recent30BtcTotal"`
	Recent30DayBTCFuturesTotal  types.Number          `json:"recent30BtcFuturesTotal"`
	Recent30DayBTCMarginTotal   types.Number          `json:"recent30BtcMarginTotal"`
	Recent30DayBUSDTotal        types.Number          `json:"recent30BusdTotal"`
	Recent30DayBUSDFuturesTotal types.Number          `json:"recent30BusdFuturesTotal"`
	Recent30DayBUSDMarginTotal  types.Number          `json:"recent30BusdMarginTotal"`
	TradeInfoVos                []SubAccountTradeInfo `json:"tradeInfoVos"`
}

// SubAccountTradeInfo holds a sub-account's trading volumes of one day
type SubAccountTradeInfo struct {
	UserID      uint64     `json:"userId"`
	BTC         float64    `json:"btc"`
	BTCFutures  float64    `json:"btcFutures"`
	BTCMargin   float64    `json:"btcMargin"`
	BUSD        float64    `json:"busd"`
	BUSDFutures float64    `json:"busdFutures"`
	BUSDMargin  float64    `json:"busdMargin"`
	Date        types.Time `json:"date"`
}

// SubAccountAPIIPRestrictionRequest holds the parameters for adding an IP restriction to a sub-account API key
type SubAccountAPIIPRestrictionRequest struct {
	Email                string
	SubAccountAPIKey     string
	RestrictToTrustedIPs bool // Sends IP restriction status 2 when true and 1, unrestricted, when false
	IPAddresses          []string
}

// AddSubAccountAPIIPRestrictionResponse holds a sub-account API key's IP restriction after an update
type AddSubAccountAPIIPRestrictionResponse struct {
	Status     string     `json:"status"` // 1 unrestricted, 2 restricted to trusted IPs
	IPList     []string   `json:"ipList"`
	UpdateTime types.Time `json:"updateTime"`
	APIKey     string     `json:"apiKey"`
}

// SubAccountAPIIPRestrictionResponse holds a sub-account API key's IP restriction
type SubAccountAPIIPRestrictionResponse struct {
	IPRestrict types.Boolean `json:"ipRestrict"` // Arrives as a quoted boolean
	IPList     []string      `json:"ipList"`
	UpdateTime types.Time    `json:"updateTime"`
	APIKey     string        `json:"apiKey"`
}

// SubAccountWalletTransferRequest holds the parameters for a transfer between a sub-account's spot and futures or margin
// accounts
type SubAccountWalletTransferRequest struct {
	Email  string
	Asset  currency.Code
	Amount float64
	Type   uint64 // Transfer direction; see FuturesTransferSubAccount and MarginTransferForSubAccount
}

// SubAccountTransferResponse holds the transaction ID of a sub-account transfer
type SubAccountTransferResponse struct {
	TransactionID string `json:"txnId"`
}

// SubAccountFuturesAccountResponse holds the detail of a sub-account's futures account
type SubAccountFuturesAccountResponse struct {
	Email                       string                   `json:"email"`
	Asset                       currency.Code            `json:"asset"`
	Assets                      []SubAccountFuturesAsset `json:"assets"`
	CanDeposit                  bool                     `json:"canDeposit"`
	CanTrade                    bool                     `json:"canTrade"`
	CanWithdraw                 bool                     `json:"canWithdraw"`
	FeeTier                     uint64                   `json:"feeTier"`
	MaxWithdrawAmount           types.Number             `json:"maxWithdrawAmount"`
	TotalInitialMargin          types.Number             `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number             `json:"totalMaintenanceMargin"`
	TotalMarginBalance          types.Number             `json:"totalMarginBalance"`
	TotalOpenOrderInitialMargin types.Number             `json:"totalOpenOrderInitialMargin"`
	TotalPositionInitialMargin  types.Number             `json:"totalPositionInitialMargin"`
	TotalUnrealisedProfit       types.Number             `json:"totalUnrealizedProfit"`
	TotalWalletBalance          types.Number             `json:"totalWalletBalance"`
	UpdateTime                  types.Time               `json:"updateTime"`
}

// SubAccountFuturesAsset holds a sub-account's futures balance of one asset
type SubAccountFuturesAsset struct {
	Asset                  currency.Code `json:"asset"`
	InitialMargin          types.Number  `json:"initialMargin"`
	MaintenanceMargin      types.Number  `json:"maintenanceMargin"`
	MarginBalance          types.Number  `json:"marginBalance"`
	MaxWithdrawAmount      types.Number  `json:"maxWithdrawAmount"`
	OpenOrderInitialMargin types.Number  `json:"openOrderInitialMargin"`
	PositionInitialMargin  types.Number  `json:"positionInitialMargin"`
	UnrealisedProfit       types.Number  `json:"unrealizedProfit"`
	WalletBalance          types.Number  `json:"walletBalance"`
}

// SubAccountFuturesAccountV2Response holds the detail of a sub-account's futures account; which account is set depends
// on the futures type queried
type SubAccountFuturesAccountV2Response struct {
	FutureAccountResponse   *SubAccountFutureAccount   `json:"futureAccountResp"`
	DeliveryAccountResponse *SubAccountDeliveryAccount `json:"deliveryAccountResp"`
}

// SubAccountFutureAccount holds the detail of a sub-account's USDⓈ-M futures account
type SubAccountFutureAccount struct {
	Email                       string                   `json:"email"`
	Assets                      []SubAccountFuturesAsset `json:"assets"`
	CanDeposit                  bool                     `json:"canDeposit"`
	CanTrade                    bool                     `json:"canTrade"`
	CanWithdraw                 bool                     `json:"canWithdraw"`
	FeeTier                     uint64                   `json:"feeTier"`
	MaxWithdrawAmount           types.Number             `json:"maxWithdrawAmount"`
	TotalInitialMargin          types.Number             `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number             `json:"totalMaintenanceMargin"`
	TotalMarginBalance          types.Number             `json:"totalMarginBalance"`
	TotalOpenOrderInitialMargin types.Number             `json:"totalOpenOrderInitialMargin"`
	TotalPositionInitialMargin  types.Number             `json:"totalPositionInitialMargin"`
	TotalUnrealisedProfit       types.Number             `json:"totalUnrealizedProfit"`
	TotalWalletBalance          types.Number             `json:"totalWalletBalance"`
	UpdateTime                  types.Time               `json:"updateTime"`
}

// SubAccountDeliveryAccount holds the detail of a sub-account's COIN-M futures account
type SubAccountDeliveryAccount struct {
	Email       string                   `json:"email"`
	Assets      []SubAccountFuturesAsset `json:"assets"`
	CanDeposit  bool                     `json:"canDeposit"`
	CanTrade    bool                     `json:"canTrade"`
	CanWithdraw bool                     `json:"canWithdraw"`
	FeeTier     uint64                   `json:"feeTier"`
	UpdateTime  types.Time               `json:"updateTime"`
}

// SubAccountMarginAccountDetailResponse holds the detail of a sub-account's margin account
type SubAccountMarginAccountDetailResponse struct {
	Email                    string                           `json:"email"`
	MarginLevel              types.Number                     `json:"marginLevel"`
	TotalAssetOfBTC          types.Number                     `json:"totalAssetOfBtc"`
	TotalLiabilityOfBTC      types.Number                     `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC       types.Number                     `json:"totalNetAssetOfBtc"`
	MarginTradeCoefficientVo SubAccountMarginTradeCoefficient `json:"marginTradeCoeffVo"`
	MarginUserAssetVoList    []MarginAssetBalance             `json:"marginUserAssetVoList"`
}

// SubAccountMarginTradeCoefficient holds the margin ratios of a sub-account's margin account
type SubAccountMarginTradeCoefficient struct {
	ForceLiquidationBar types.Number `json:"forceLiquidationBar"` // Liquidation margin ratio
	MarginCallBar       types.Number `json:"marginCallBar"`       // Margin call margin ratio
	NormalBar           types.Number `json:"normalBar"`           // Initial margin ratio
}

// SubAccountDepositAddressRequest holds the parameters for a sub-account deposit address query
type SubAccountDepositAddressRequest struct {
	Email   string
	Coin    currency.Code
	Network string  // The coin's default network when empty
	Amount  float64 // Required for the LIGHTNING network
}

// SubAccountDepositAddressResponse holds a sub-account deposit address
type SubAccountDepositAddressResponse struct {
	Address string        `json:"address"`
	Coin    currency.Code `json:"coin"`
	Tag     string        `json:"tag"`
	URL     string        `json:"url"`
}

// SubAccountDepositHistoryRequest holds the parameters for a sub-account deposit history query
type SubAccountDepositHistoryRequest struct {
	Email         string
	IncludeSource bool // Adds each deposit's source address
	Coin          currency.Code
	Status        *uint64 // 0 pending, 1 success, 6 credited but cannot withdraw, 7 wrong deposit, 8 waiting user confirmation
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64 // Up to 200
	Offset        uint64
	TxID          string
}

// SubAccountDepositRecord holds one deposit of a sub-account
type SubAccountDepositRecord struct {
	ID            string        `json:"id"`
	Amount        types.Number  `json:"amount"`
	Coin          currency.Code `json:"coin"`
	Network       string        `json:"network"`
	Status        uint64        `json:"status"`
	Address       string        `json:"address"`
	AddressTag    string        `json:"addressTag"`
	TxID          string        `json:"txId"`
	InsertTime    types.Time    `json:"insertTime"`
	TransferType  uint64        `json:"transferType"`
	ConfirmTimes  string        `json:"confirmTimes"`
	UnlockConfirm uint64        `json:"unlockConfirm"`
	WalletType    uint64        `json:"walletType"`
	SourceAddress string        `json:"sourceAddress"` // Only present when the source is requested
}

// SubAccountFuturesAccountSummaryResponse holds the summary of the sub-accounts' futures accounts, valued in USD
type SubAccountFuturesAccountSummaryResponse struct {
	TotalInitialMargin          types.Number               `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number               `json:"totalMaintenanceMargin"`
	TotalMarginBalance          types.Number               `json:"totalMarginBalance"`
	TotalOpenOrderInitialMargin types.Number               `json:"totalOpenOrderInitialMargin"`
	TotalPositionInitialMargin  types.Number               `json:"totalPositionInitialMargin"`
	TotalUnrealisedProfit       types.Number               `json:"totalUnrealizedProfit"`
	TotalWalletBalance          types.Number               `json:"totalWalletBalance"`
	Asset                       currency.Code              `json:"asset"`
	SubAccountList              []SubAccountFuturesSummary `json:"subAccountList"`
}

// SubAccountFuturesSummary holds the summary of one sub-account's futures account
type SubAccountFuturesSummary struct {
	Email                       string        `json:"email"`
	TotalInitialMargin          types.Number  `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number  `json:"totalMaintenanceMargin"`
	TotalMarginBalance          types.Number  `json:"totalMarginBalance"`
	TotalOpenOrderInitialMargin types.Number  `json:"totalOpenOrderInitialMargin"`
	TotalPositionInitialMargin  types.Number  `json:"totalPositionInitialMargin"`
	TotalUnrealisedProfit       types.Number  `json:"totalUnrealizedProfit"`
	TotalWalletBalance          types.Number  `json:"totalWalletBalance"`
	Asset                       currency.Code `json:"asset"`
}

// SubAccountFuturesAccountSummaryV2Response holds the summary of the sub-accounts' futures accounts; which summary is
// set depends on the futures type queried
type SubAccountFuturesAccountSummaryV2Response struct {
	FutureAccountSummaryResponse   *SubAccountFuturesAccountSummaryResponse `json:"futureAccountSummaryResp"`
	DeliveryAccountSummaryResponse *SubAccountDeliveryAccountSummary        `json:"deliveryAccountSummaryResp"`
}

// SubAccountDeliveryAccountSummary holds the summary of the sub-accounts' COIN-M futures accounts, valued in BTC
type SubAccountDeliveryAccountSummary struct {
	TotalMarginBalanceOfBTC    types.Number                `json:"totalMarginBalanceOfBTC"`
	TotalUnrealisedProfitOfBTC types.Number                `json:"totalUnrealizedProfitOfBTC"`
	TotalWalletBalanceOfBTC    types.Number                `json:"totalWalletBalanceOfBTC"`
	Asset                      currency.Code               `json:"asset"`
	SubAccountList             []SubAccountDeliverySummary `json:"subAccountList"`
}

// SubAccountDeliverySummary holds the summary of one sub-account's COIN-M futures account
type SubAccountDeliverySummary struct {
	Email                 string        `json:"email"`
	TotalMarginBalance    types.Number  `json:"totalMarginBalance"`
	TotalUnrealisedProfit types.Number  `json:"totalUnrealizedProfit"`
	TotalWalletBalance    types.Number  `json:"totalWalletBalance"`
	Asset                 currency.Code `json:"asset"`
}

// SubAccountMarginAccountSummaryResponse holds the summary of the sub-accounts' margin accounts
type SubAccountMarginAccountSummaryResponse struct {
	TotalAssetOfBTC     types.Number              `json:"totalAssetOfBtc"`
	TotalLiabilityOfBTC types.Number              `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC  types.Number              `json:"totalNetAssetOfBtc"`
	SubAccountList      []SubAccountMarginSummary `json:"subAccountList"`
}

// SubAccountMarginSummary holds the summary of one sub-account's margin account
type SubAccountMarginSummary struct {
	Email               string       `json:"email"`
	TotalAssetOfBTC     types.Number `json:"totalAssetOfBtc"`
	TotalLiabilityOfBTC types.Number `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC  types.Number `json:"totalNetAssetOfBtc"`
}

// SubAccountAssetsResponse holds a sub-account's asset balances
type SubAccountAssetsResponse struct {
	Balances []SubAccountBalance `json:"balances"`
}

// SubAccountBalance holds a sub-account's balance of one asset. The balances arrive as bare numbers from the V3
// endpoint and quoted from V4
type SubAccountBalance struct {
	Freeze      types.Number  `json:"freeze"`
	Withdrawing types.Number  `json:"withdrawing"`
	Asset       currency.Code `json:"asset"`
	Free        types.Number  `json:"free"`
	Locked      types.Number  `json:"locked"`
}

// SubAccountFuturesAssetTransferHistoryRequest holds the parameters for a sub-account futures asset transfer history
// query
type SubAccountFuturesAssetTransferHistoryRequest struct {
	Email       string
	FuturesType uint64    // 1 USDⓈ-M, 2 COIN-M
	StartTime   time.Time // Cannot be more than a month ago
	EndTime     time.Time
	Page        uint64
	Limit       uint64 // Up to 500, default 50
}

// SubAccountFuturesAssetTransferHistoryResponse holds a sub-account's futures asset transfers
type SubAccountFuturesAssetTransferHistoryResponse struct {
	Success     bool                             `json:"success"`
	FuturesType uint64                           `json:"futuresType"`
	Transfers   []SubAccountFuturesAssetTransfer `json:"transfers"`
}

// SubAccountFuturesAssetTransfer holds one futures asset transfer between the master account and sub-accounts
type SubAccountFuturesAssetTransfer struct {
	From          string        `json:"from"`
	To            string        `json:"to"`
	Asset         currency.Code `json:"asset"`
	Quantity      types.Number  `json:"qty"`
	TransactionID uint64        `json:"tranId"`
	Time          types.Time    `json:"time"`
}

// SubAccountFuturesAssetTransferRequest holds the parameters for a sub-account futures asset transfer
type SubAccountFuturesAssetTransferRequest struct {
	FromEmail   string
	ToEmail     string
	FuturesType uint64 // 1 USDⓈ-M, 2 COIN-M
	Asset       currency.Code
	Amount      float64
}

// SubAccountFuturesAssetTransferResponse holds the result of a sub-account futures asset transfer
type SubAccountFuturesAssetTransferResponse struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"txnId"`
}

// SubAccountSpotAssetsSummaryResponse holds the BTC valued spot asset summary of the master account and its
// sub-accounts
type SubAccountSpotAssetsSummaryResponse struct {
	TotalCount                uint64                   `json:"totalCount"`
	MasterAccountTotalAsset   types.Number             `json:"masterAccountTotalAsset"`
	SpotSubUserAssetBTCVoList []SubAccountSpotAssetBTC `json:"spotSubUserAssetBtcVoList"`
}

// SubAccountSpotAssetBTC holds the BTC valued spot assets of one sub-account
type SubAccountSpotAssetBTC struct {
	Email      string       `json:"email"`
	TotalAsset types.Number `json:"totalAsset"`
}

// SubAccountSpotAssetTransferHistoryRequest holds the parameters for a sub-account spot asset transfer history query
type SubAccountSpotAssetTransferHistoryRequest struct {
	FromEmail string // Cannot be set with ToEmail
	ToEmail   string
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	Limit     uint64 // Up to 200
}

// SubAccountSpotAssetTransfer holds one spot asset transfer between the master account and sub-accounts
type SubAccountSpotAssetTransfer struct {
	From          string        `json:"from"`
	To            string        `json:"to"`
	Asset         currency.Code `json:"asset"`
	Quantity      types.Number  `json:"qty"`
	Status        string        `json:"status"`
	TransactionID uint64        `json:"tranId"`
	Time          types.Time    `json:"time"`
}

// SubAccountUniversalTransferHistoryRequest holds the parameters for a universal transfer history query
type SubAccountUniversalTransferHistoryRequest struct {
	FromEmail           string // Cannot be set with ToEmail
	ToEmail             string
	ClientTransactionID string
	StartTime           time.Time // The window must be shorter than 7 days and defaults to the last 7 days
	EndTime             time.Time
	Page                uint64
	Limit               uint64 // Up to 500, default 500
}

// SubAccountUniversalTransferHistoryResponse holds universal transfers between the master account and sub-accounts
type SubAccountUniversalTransferHistoryResponse struct {
	Result     []SubAccountUniversalTransfer `json:"result"`
	TotalCount uint64                        `json:"totalCount"`
}

// SubAccountUniversalTransfer holds one universal transfer between the master account and sub-accounts
type SubAccountUniversalTransfer struct {
	TransactionID       uint64        `json:"tranId"`
	FromEmail           string        `json:"fromEmail"`
	ToEmail             string        `json:"toEmail"`
	Asset               currency.Code `json:"asset"`
	Amount              types.Number  `json:"amount"`
	CreateTimeStamp     types.Time    `json:"createTimeStamp"`
	FromAccountType     string        `json:"fromAccountType"`
	ToAccountType       string        `json:"toAccountType"`
	Status              string        `json:"status"`
	ClientTransactionID string        `json:"clientTranId"`
}

// SubAccountUniversalTransferRequest holds the parameters for a universal transfer between the master account and
// sub-accounts
type SubAccountUniversalTransferRequest struct {
	FromEmail           string // The master account when empty
	ToEmail             string // The master account when empty
	FromAccountType     string // SPOT, USDT_FUTURE, COIN_FUTURE, MARGIN, ISOLATED_MARGIN or ALPHA
	ToAccountType       string // SPOT, USDT_FUTURE, COIN_FUTURE, MARGIN, ISOLATED_MARGIN or ALPHA
	ClientTransactionID string // Must be unique
	Symbol              currency.Pair
	Asset               currency.Code
	Amount              float64
}

// SubAccountUniversalTransferResponse holds the result of a universal transfer
type SubAccountUniversalTransferResponse struct {
	TransactionID       uint64 `json:"tranId"`
	ClientTransactionID string `json:"clientTranId"`
}

// SubAccountTransferHistoryRequest holds the parameters for a sub-account's transfer history query
type SubAccountTransferHistoryRequest struct {
	Asset             currency.Code // Every asset when empty
	Type              uint64        // 1 transfer in, 2 transfer out; transfers out when unset
	StartTime         time.Time     // The last 30 days when no time window is set
	EndTime           time.Time
	Limit             uint64 // Up to 200
	ReturnFailHistory bool   // Includes failed transfers
}

// SubAccountTransferHistoryRecord holds one transfer of a sub-account
type SubAccountTransferHistoryRecord struct {
	CounterParty    string        `json:"counterParty"`
	Email           string        `json:"email"`
	Type            uint64        `json:"type"` // 1 transfer in, 2 transfer out
	Asset           currency.Code `json:"asset"`
	Quantity        types.Number  `json:"qty"`
	FromAccountType string        `json:"fromAccountType"`
	ToAccountType   string        `json:"toAccountType"`
	Status          string        `json:"status"`
	TransactionID   uint64        `json:"tranId"`
	Time            types.Time    `json:"time"`
}

// ManagedSubAccountTransferResponse holds the transaction ID of a managed sub-account deposit or withdrawal
type ManagedSubAccountTransferResponse struct {
	TransactionID uint64 `json:"tranId"`
}

// ManagedSubAccountAsset holds a managed sub-account's balance of one asset
type ManagedSubAccountAsset struct {
	Coin             currency.Code `json:"coin"`
	Name             string        `json:"name"`
	TotalBalance     types.Number  `json:"totalBalance"`
	AvailableBalance types.Number  `json:"availableBalance"`
	InOrder          types.Number  `json:"inOrder"`
	BTCValue         types.Number  `json:"btcValue"`
}

// ManagedSubAccountFuturesAssetDetailsResponse holds the futures assets and positions of a managed sub-account
type ManagedSubAccountFuturesAssetDetailsResponse struct {
	Code        string                             `json:"code"`
	Message     string                             `json:"message"`
	SnapshotVos []ManagedSubAccountFuturesSnapshot `json:"snapshotVos"`
}

// ManagedSubAccountFuturesSnapshot holds a managed sub-account's futures account
type ManagedSubAccountFuturesSnapshot struct {
	Type       string                               `json:"type"`
	UpdateTime types.Time                           `json:"updateTime"`
	Data       ManagedSubAccountFuturesSnapshotData `json:"data"`
}

// ManagedSubAccountFuturesSnapshotData holds a managed sub-account's futures assets and positions
type ManagedSubAccountFuturesSnapshotData struct {
	Assets   []ManagedSubAccountFuturesAsset    `json:"assets"`
	Position []ManagedSubAccountFuturesPosition `json:"position"`
}

// ManagedSubAccountFuturesAsset holds a managed sub-account's futures balance of one asset
type ManagedSubAccountFuturesAsset struct {
	Asset         currency.Code `json:"asset"`
	MarginBalance float64       `json:"marginBalance"`
	WalletBalance float64       `json:"walletBalance"`
}

// ManagedSubAccountFuturesPosition holds a futures position of a managed sub-account
type ManagedSubAccountFuturesPosition struct {
	Symbol         string  `json:"symbol"`
	EntryPrice     float64 `json:"entryPrice"`
	MarkPrice      float64 `json:"markPrice"`
	PositionAmount float64 `json:"positionAmt"`
}

// ManagedSubAccountListResponse holds the investor's managed sub-accounts
type ManagedSubAccountListResponse struct {
	Total                    uint64                  `json:"total"`
	ManagerSubUserInfoVoList []ManagedSubAccountInfo `json:"managerSubUserInfoVoList"`
}

// ManagedSubAccountInfo holds a managed sub-account
type ManagedSubAccountInfo struct {
	RootUserID               uint64     `json:"rootUserId"`
	ManagersubUserID         uint64     `json:"managersubUserId"`
	BindParentUserID         uint64     `json:"bindParentUserId"`
	Email                    string     `json:"email"`
	InsertTimeStamp          types.Time `json:"insertTimeStamp"`
	BindParentEmail          string     `json:"bindParentEmail"`
	IsSubUserEnabled         bool       `json:"isSubUserEnabled"`
	IsUserActive             bool       `json:"isUserActive"`
	IsMarginEnabled          bool       `json:"isMarginEnabled"`
	IsFutureEnabled          bool       `json:"isFutureEnabled"`
	IsSignedLVTRiskAgreement bool       `json:"isSignedLVTRiskAgreement"`
}

// ManagedSubAccountMarginAssetDetailsResponse holds the margin account of a managed sub-account
type ManagedSubAccountMarginAssetDetailsResponse struct {
	MarginLevel         types.Number         `json:"marginLevel"`
	TotalAssetOfBTC     types.Number         `json:"totalAssetOfBtc"`
	TotalLiabilityOfBTC types.Number         `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC  types.Number         `json:"totalNetAssetOfBtc"`
	UserAssets          []MarginAssetBalance `json:"userAssets"`
}

// ManagedSubAccountTransferLogRequest holds the parameters for a managed sub-account transfer log query
type ManagedSubAccountTransferLogRequest struct {
	StartTime                   time.Time // The window cannot exceed half a year
	EndTime                     time.Time
	Page                        uint64
	Limit                       uint64 // Up to 500
	Transfers                   string // Transfer direction, FROM or TO
	TransferFunctionAccountType string // SPOT, MARGIN, ISOLATED_MARGIN, USDT_FUTURE or COIN_FUTURE
}

// ManagedSubAccountTransferLogResponse holds managed sub-account transfers
type ManagedSubAccountTransferLogResponse struct {
	ManagerSubTransferHistoryVos []ManagedSubAccountTransfer `json:"managerSubTransferHistoryVos"`
	Count                        uint64                      `json:"count"`
}

// ManagedSubAccountTransfer holds one transfer of a managed sub-account
type ManagedSubAccountTransfer struct {
	FromEmail       string        `json:"fromEmail"`
	FromAccountType string        `json:"fromAccountType"`
	ToEmail         string        `json:"toEmail"`
	ToAccountType   string        `json:"toAccountType"`
	Asset           currency.Code `json:"asset"`
	Amount          types.Number  `json:"amount"`
	ScheduledData   types.Time    `json:"scheduledData"`
	CreateTime      types.Time    `json:"createTime"`
	Status          string        `json:"status"`
	TransactionID   uint64        `json:"tranId"`
}

// ManagedSubAccountWithdrawRequest holds the parameters for a withdrawal from a managed sub-account
type ManagedSubAccountWithdrawRequest struct {
	FromEmail    string
	Asset        currency.Code
	Amount       float64
	TransferDate time.Time // The UTC date the withdrawal happens; immediately when zero
}
