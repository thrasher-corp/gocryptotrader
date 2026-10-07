package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AccountAPITradingStatusResponse holds the Account API Trading Status
type AccountAPITradingStatusResponse struct {
	Data AccountAPITradingStatus `json:"data"`
}

// AccountAPITradingStatus holds whether the quantitative trading rules have locked the account's API trading
type AccountAPITradingStatus struct {
	IsLocked           bool                       `json:"isLocked"`
	PlannedRecoverTime types.Time                 `json:"plannedRecoverTime"` // Zero unless API trading is locked
	TriggerCondition   TradingAPITriggerCondition `json:"triggerCondition"`
	UpdateTime         types.Time                 `json:"updateTime"`
}

// TradingAPITriggerCondition holds the order counts that lock an account's API trading
type TradingAPITriggerCondition struct {
	GCR  uint64 `json:"GCR"`  // Number of GTC orders
	IFER uint64 `json:"IFER"` // Number of FOK/IOC orders
	UFR  uint64 `json:"UFR"`  // Number of orders
}

// AccountInfoResponse holds the Account info
type AccountInfoResponse struct {
	VIPLevel                       uint64 `json:"vipLevel"`
	IsMarginEnabled                bool   `json:"isMarginEnabled"`
	IsFutureEnabled                bool   `json:"isFutureEnabled"`
	IsOptionsEnabled               bool   `json:"isOptionsEnabled"`
	IsPortfolioMarginRetailEnabled bool   `json:"isPortfolioMarginRetailEnabled"`
}

// AccountStatusResponse holds the Account Status
type AccountStatusResponse struct {
	Data string `json:"data"`
}

// AccountSnapshotRequest holds the parameters for an account snapshot query
type AccountSnapshotRequest struct {
	Type      string // SPOT, MARGIN or FUTURES
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 7 to 30, default 7
}

// AccountSnapshotResponse holds daily account snapshots
type AccountSnapshotResponse struct {
	Code        int64             `json:"code"`
	Message     string            `json:"msg"`
	SnapshotVos []AccountSnapshot `json:"snapshotVos"`
}

// AccountSnapshot holds one daily snapshot of a spot, margin or futures account
type AccountSnapshot struct {
	Data       AccountSnapshotData `json:"data"`
	Type       string              `json:"type"`
	UpdateTime types.Time          `json:"updateTime"`
}

// AccountSnapshotData holds a snapshot's balances; which fields are set depends on the snapshot type: balances for spot,
// margin level, liabilities and user assets for margin, assets and positions for futures
type AccountSnapshotData struct {
	Balances            []AccountSnapshotBalance      `json:"balances"`
	TotalAssetOfBTC     types.Number                  `json:"totalAssetOfBtc"`
	MarginLevel         types.Number                  `json:"marginLevel"`
	TotalLiabilityOfBTC types.Number                  `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC  types.Number                  `json:"totalNetAssetOfBtc"`
	UserAssets          []MarginAssetBalance          `json:"userAssets"`
	Assets              []AccountSnapshotFuturesAsset `json:"assets"`
	Position            []AccountSnapshotPosition     `json:"position"`
}

// AccountSnapshotBalance holds a spot asset balance in an account snapshot
type AccountSnapshotBalance struct {
	Asset  currency.Code `json:"asset"`
	Free   types.Number  `json:"free"`
	Locked types.Number  `json:"locked"`
}

// MarginAssetBalance holds a margin account's balance of one asset
type MarginAssetBalance struct {
	Asset    currency.Code `json:"asset"`
	Borrowed types.Number  `json:"borrowed"`
	Free     types.Number  `json:"free"`
	Interest types.Number  `json:"interest"`
	Locked   types.Number  `json:"locked"`
	NetAsset types.Number  `json:"netAsset"`
}

// AccountSnapshotFuturesAsset holds a futures asset balance in an account snapshot
type AccountSnapshotFuturesAsset struct {
	Asset         currency.Code `json:"asset"`
	MarginBalance types.Number  `json:"marginBalance"`
	WalletBalance types.Number  `json:"walletBalance"`
}

// AccountSnapshotPosition holds a futures position in an account snapshot
type AccountSnapshotPosition struct {
	EntryPrice       types.Number `json:"entryPrice"`
	MarkPrice        types.Number `json:"markPrice"`
	PositionAmount   types.Number `json:"positionAmt"`
	Symbol           string       `json:"symbol"`
	UnrealisedProfit types.Number `json:"unRealizedProfit"`
}

// APIKeyPermissionResponse holds the permissions of the API key
type APIKeyPermissionResponse struct {
	IPRestrict                   bool       `json:"ipRestrict"`
	CreateTime                   types.Time `json:"createTime"`
	EnableReading                bool       `json:"enableReading"`
	EnableWithdrawals            bool       `json:"enableWithdrawals"`
	EnableInternalTransfer       bool       `json:"enableInternalTransfer"`
	EnableMargin                 bool       `json:"enableMargin"`
	EnableFutures                bool       `json:"enableFutures"`
	PermitsUniversalTransfer     bool       `json:"permitsUniversalTransfer"`
	EnableVanillaOptions         bool       `json:"enableVanillaOptions"`
	EnableFixAPITrade            bool       `json:"enableFixApiTrade"`
	EnableFixReadOnly            bool       `json:"enableFixReadOnly"`
	EnableSpotAndMarginTrading   bool       `json:"enableSpotAndMarginTrading"`
	EnablePortfolioMarginTrading bool       `json:"enablePortfolioMarginTrading"`
}

// AssetDetail holds the deposit and withdrawal details of an asset
type AssetDetail struct {
	MinimumWithdrawAmount types.Number `json:"minWithdrawAmount"`
	DepositStatus         bool         `json:"depositStatus"`
	WithdrawFee           types.Number `json:"withdrawFee"` // Quoted in live responses, although the documentation shows a bare number
	WithdrawStatus        bool         `json:"withdrawStatus"`
	DepositTip            string       `json:"depositTip"` // Only present when deposits are suspended
}

// AssetDividendRecordRequest holds the parameters for an asset dividend record query
type AssetDividendRecordRequest struct {
	Asset     currency.Code
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // Up to 500, default 20
}

// AssetDividendRecordResponse holds asset dividend records
type AssetDividendRecordResponse struct {
	Rows  []AssetDividend `json:"rows"`
	Total uint64          `json:"total"`
}

// AssetDividend holds one asset dividend distribution
type AssetDividend struct {
	ID            uint64        `json:"id"`
	Amount        types.Number  `json:"amount"`
	Asset         currency.Code `json:"asset"`
	DividendTime  types.Time    `json:"divTime"`
	EnglishInfo   string        `json:"enInfo"`
	TransactionID uint64        `json:"tranId"`
	Direction     int64         `json:"direction"`
}

// DustLogResponse holds the dust conversion log
type DustLogResponse struct {
	Total              uint64            `json:"total"`
	UserAssetDribblets []DustLogDribblet `json:"userAssetDribblets"`
}

// DustLogDribblet holds one dust conversion
type DustLogDribblet struct {
	OperateTime              types.Time              `json:"operateTime"`
	TotalTransferedAmount    types.Number            `json:"totalTransferedAmount"`
	TotalServiceChargeAmount types.Number            `json:"totalServiceChargeAmount"`
	TransactionID            uint64                  `json:"transId"`
	UserAssetDribbletDetails []DustLogDribbletDetail `json:"userAssetDribbletDetails"`
}

// DustLogDribbletDetail holds the conversion of one asset within a dust conversion
type DustLogDribbletDetail struct {
	TransactionID       uint64        `json:"transId"`
	ServiceChargeAmount types.Number  `json:"serviceChargeAmount"`
	Amount              types.Number  `json:"amount"`
	OperateTime         types.Time    `json:"operateTime"`
	TransferedAmount    types.Number  `json:"transferedAmount"`
	FromAsset           currency.Code `json:"fromAsset"`
	TargetAsset         currency.Code `json:"targetAsset"`
}

// DustTransferResponse holds the result of a dust transfer
type DustTransferResponse struct {
	TotalServiceCharge types.Number         `json:"totalServiceCharge"`
	TotalTransfered    types.Number         `json:"totalTransfered"`
	TransferResult     []DustTransferResult `json:"transferResult"`
}

// DustTransferResult holds the conversion of one asset within a dust transfer
type DustTransferResult struct {
	Amount              types.Number  `json:"amount"`
	FromAsset           currency.Code `json:"fromAsset"`
	OperateTime         types.Time    `json:"operateTime"`
	ServiceChargeAmount types.Number  `json:"serviceChargeAmount"`
	TransactionID       uint64        `json:"tranId"`
	TransferedAmount    types.Number  `json:"transferedAmount"`
}

// FundingAsset holds a funding wallet asset balance
type FundingAsset struct {
	Asset        currency.Code `json:"asset"`
	Free         types.Number  `json:"free"`
	Locked       types.Number  `json:"locked"`
	Freeze       types.Number  `json:"freeze"`
	Withdrawing  types.Number  `json:"withdrawing"`
	BTCValuation types.Number  `json:"btcValuation"`
}

// AssetsConvertibleToBNBResponse holds the assets that can be converted into BNB
type AssetsConvertibleToBNBResponse struct {
	Details            []AssetConvertibleToBNB `json:"details"`
	TotalTransferBTC   types.Number            `json:"totalTransferBtc"`
	TotalTransferBNB   types.Number            `json:"totalTransferBNB"`
	DribbletPercentage types.Number            `json:"dribbletPercentage"`
}

// AssetConvertibleToBNB holds an asset balance that can be converted into BNB
type AssetConvertibleToBNB struct {
	Asset            currency.Code `json:"asset"`
	AssetFullName    string        `json:"assetFullName"`
	AmountFree       types.Number  `json:"amountFree"`
	ToBTC            types.Number  `json:"toBTC"`
	ToBNB            types.Number  `json:"toBNB"`
	ToBNBOffExchange types.Number  `json:"toBNBOffExchange"`
	Exchange         types.Number  `json:"exchange"`
}

// CloudMiningHistoryRequest holds the parameters for a cloud-mining payment and refund history query
type CloudMiningHistoryRequest struct {
	TransactionID       uint64
	ClientTransactionID string
	Asset               currency.Code
	StartTime           time.Time
	EndTime             time.Time
	Current             uint64
	Size                uint64 // Up to 100, default 10
}

// CloudMiningHistoryResponse holds cloud-mining payments and refunds
type CloudMiningHistoryResponse struct {
	Total uint64                `json:"total"`
	Rows  []CloudMiningTransfer `json:"rows"`
}

// CloudMiningTransfer holds one cloud-mining payment or refund
type CloudMiningTransfer struct {
	CreateTime    types.Time    `json:"createTime"`
	TransactionID uint64        `json:"tranId"`
	Type          uint64        `json:"type"` // 248 payment, 249 refund
	Asset         currency.Code `json:"asset"`
	Amount        types.Number  `json:"amount"`
	Status        string        `json:"status"`
}

// UserDelegationHistoryRequest holds the parameters for a user delegation history query
type UserDelegationHistoryRequest struct {
	Email     string
	StartTime time.Time
	EndTime   time.Time
	Type      string // DELEGATE or UNDELEGATE
	Asset     currency.Code
	Current   uint64
	Size      uint64 // Up to 100, default 10
}

// UserDelegationHistoryResponse holds delegation records
type UserDelegationHistoryResponse struct {
	Total uint64           `json:"total"`
	Rows  []UserDelegation `json:"rows"`
}

// UserDelegation holds one delegation or undelegation
type UserDelegation struct {
	ClientTransactionID string        `json:"clientTranId"`
	TransferType        string        `json:"transferType"`
	Asset               currency.Code `json:"asset"`
	Amount              types.Number  `json:"amount"`
	Time                types.Time    `json:"time"`
}

// UserUniversalTransferHistoryRequest holds the parameters for a user universal transfer history query
type UserUniversalTransferHistoryRequest struct {
	Type       string
	StartTime  time.Time
	EndTime    time.Time
	Current    uint64
	Size       uint64        // Up to 100, default 10
	FromSymbol currency.Pair // Isolated margin symbol, required for ISOLATEDMARGIN_MARGIN and ISOLATEDMARGIN_ISOLATEDMARGIN
	ToSymbol   currency.Pair // Isolated margin symbol, required for MARGIN_ISOLATEDMARGIN and ISOLATEDMARGIN_ISOLATEDMARGIN
}

// UserUniversalTransferHistoryResponse holds user universal transfers
type UserUniversalTransferHistoryResponse struct {
	Total uint64                        `json:"total"`
	Rows  []UserUniversalTransferRecord `json:"rows"`
}

// UserUniversalTransferRecord holds one user universal transfer
type UserUniversalTransferRecord struct {
	Asset         currency.Code `json:"asset"`
	Amount        types.Number  `json:"amount"`
	Type          string        `json:"type"`
	Status        string        `json:"status"`
	TransactionID uint64        `json:"tranId"`
	Timestamp     types.Time    `json:"timestamp"`
}

// UserUniversalTransferRequest holds the parameters for a user universal transfer
type UserUniversalTransferRequest struct {
	Type       string // Transfer type such as MAIN_UMFUTURE or FUNDING_MAIN
	Asset      currency.Code
	Amount     float64
	FromSymbol currency.Pair // Isolated margin symbol, required for ISOLATEDMARGIN_MARGIN and ISOLATEDMARGIN_ISOLATEDMARGIN
	ToSymbol   currency.Pair // Isolated margin symbol, required for MARGIN_ISOLATEDMARGIN and ISOLATEDMARGIN_ISOLATEDMARGIN
}

// UserUniversalTransferResponse holds the result of a user universal transfer
type UserUniversalTransferResponse struct {
	TransactionID uint64 `json:"tranId"`
}

// UserWalletBalance holds the balance of one wallet
type UserWalletBalance struct {
	Activate      bool                 `json:"activate"`
	Balance       types.Number         `json:"balance"`
	WalletName    string               `json:"walletName"`
	AssetBalances []WalletAssetBalance `json:"assetBalances"` // Only present when the balance detail is requested
}

// WalletAssetBalance holds a wallet's balance of one asset
type WalletAssetBalance struct {
	Asset        currency.Code `json:"asset"`
	AssetName    string        `json:"assetName"`
	Free         types.Number  `json:"free"`
	Locked       types.Number  `json:"locked"`
	Freeze       types.Number  `json:"freeze"`
	Withdrawing  types.Number  `json:"withdrawing"`
	BTCValuation types.Number  `json:"btcValuation"`
}

// TradeFee holds the trading fee rates of a symbol
type TradeFee struct {
	Symbol          string       `json:"symbol"`
	MakerCommission types.Number `json:"makerCommission"`
	TakerCommission types.Number `json:"takerCommission"`
}

// UserAsset holds the positive balance of one asset
type UserAsset struct {
	Asset        currency.Code `json:"asset"`
	Free         types.Number  `json:"free"`
	Locked       types.Number  `json:"locked"`
	Freeze       types.Number  `json:"freeze"`
	Withdrawing  types.Number  `json:"withdrawing"`
	IPOable      types.Number  `json:"ipoable"`
	BTCValuation types.Number  `json:"btcValuation"`
}

// CoinInfo holds a coin's balances and its deposit and withdrawal networks
type CoinInfo struct {
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
	DepositDescription      string        `json:"depositDesc"`  // Only set when deposits are disabled
	WithdrawDescription     string        `json:"withdrawDesc"` // Only set when withdrawals are disabled
	SpecialTips             string        `json:"specialTips"`
	SpecialWithdrawTips     string        `json:"specialWithdrawTips"`
	Name                    string        `json:"name"`
	ResetAddressStatus      bool          `json:"resetAddressStatus"`
	AddressRegex            string        `json:"addressRegex"`
	MemoRegex               string        `json:"memoRegex"`
	WithdrawFee             types.Number  `json:"withdrawFee"`
	WithdrawMinimum         types.Number  `json:"withdrawMin"`
	WithdrawMaximum         types.Number  `json:"withdrawMax"`
	WithdrawInternalMinimum types.Number  `json:"withdrawInternalMin"`
	DepositDust             types.Number  `json:"depositDust"` // Minimum creditable deposit amount
	MinimumConfirm          uint64        `json:"minConfirm"`
	UnlockConfirm           uint64        `json:"unLockConfirm"`
	SameAddress             bool          `json:"sameAddress"` // Superseded by WithdrawTag, which carries the same value
	WithdrawTag             bool          `json:"withdrawTag"` // Whether a withdrawal needs a memo or tag
	EstimatedArrivalTime    uint64        `json:"estimatedArrivalTime"`
	Busy                    bool          `json:"busy"`
	ContractAddressURL      string        `json:"contractAddressUrl"`
	ContractAddress         string        `json:"contractAddress"`
	Denomination            uint64        `json:"denomination"`
}

// DepositAddressResponse holds a deposit address
type DepositAddressResponse struct {
	Address   string        `json:"address"`
	Coin      currency.Code `json:"coin"`
	IsDefault uint64        `json:"isDefault"` // Undocumented, but present in live responses: 1 for the default address
	Tag       string        `json:"tag"`
	URL       string        `json:"url"`
}

// DepositHistoryRequest holds the parameters for a deposit history query
type DepositHistoryRequest struct {
	IncludeSource bool // Adds each deposit's source address
	Coin          currency.Code
	Status        *uint64 // 0 pending, 1 success, 2 rejected, 6 credited but cannot withdraw, 7 wrong deposit, 8 waiting user confirm
	StartTime     time.Time
	EndTime       time.Time
	Offset        uint64
	Limit         uint64 // Up to 1000, default 1000
	TxID          string
}

// DepositRecord holds one deposit
type DepositRecord struct {
	ID               string        `json:"id"`
	Amount           types.Number  `json:"amount"`
	Coin             currency.Code `json:"coin"`
	Network          string        `json:"network"`
	Status           uint64        `json:"status"`
	Address          string        `json:"address"`
	AddressTag       string        `json:"addressTag"`
	TxID             string        `json:"txId"`
	InsertTime       types.Time    `json:"insertTime"`
	CompleteTime     types.Time    `json:"completeTime"`
	TransferType     uint64        `json:"transferType"`
	ConfirmTimes     string        `json:"confirmTimes"`
	UnlockConfirm    uint64        `json:"unlockConfirm"`
	WalletType       uint64        `json:"walletType"`
	TravelRuleStatus uint64        `json:"travelRuleStatus"` // 1 when travel rule deposit information is still required
	SourceAddress    string        `json:"sourceAddress"`    // Only present when the source is requested
}

// DepositAddressWithNetwork holds a deposit address of a coin on a network
type DepositAddressWithNetwork struct {
	Coin      currency.Code `json:"coin"`
	Address   string        `json:"address"`
	Tag       string        `json:"tag"`
	IsDefault uint64        `json:"isDefault"` // 1 for the default address
}

// WithdrawAddress holds an address book entry
type WithdrawAddress struct {
	Address     string        `json:"address"`
	AddressTag  string        `json:"addressTag"`
	Coin        currency.Code `json:"coin"`
	Name        string        `json:"name"`
	Network     string        `json:"network"`
	Origin      string        `json:"origin"`     // The address source the user entered when the origin type is others
	OriginType  string        `json:"originType"` // Address source type
	WhiteStatus bool          `json:"whiteStatus"`
}

// DepositCreditApplyRequest holds the parameters for a one click arrival deposit apply
type DepositCreditApplyRequest struct {
	DepositID    uint64 // Takes priority over TxID
	TxID         string
	SubAccountID string
	SubUserID    uint64
}

// DepositCreditApplyResponse holds the result of a one click arrival deposit apply
type DepositCreditApplyResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    bool   `json:"data"`
	Success bool   `json:"success"`
}

// WithdrawRequest holds the parameters for a withdrawal
type WithdrawRequest struct {
	Coin               currency.Code
	WithdrawOrderID    string // Client withdrawal ID, which WithdrawHistory can filter by
	Network            string
	Address            string
	AddressTag         string // Must be empty for networks without memo or tag support
	Amount             float64
	TransactionFeeFlag bool // For internal transfers, charges the fee to the destination account
	Name               string
	WalletType         *uint64 // 0 spot, 1 funding; the selected wallet when nil
}

// WithdrawResponse holds the ID of a submitted withdrawal
type WithdrawResponse struct {
	ID string `json:"id"`
}

// WithdrawHistoryRequest holds the parameters for a withdrawal history query
type WithdrawHistoryRequest struct {
	Coin            currency.Code
	WithdrawOrderID string
	Status          *uint64 // 0 email sent, 2 awaiting approval, 3 rejected, 4 processing, 6 completed
	Offset          uint64
	Limit           uint64   // Up to 1000, default 1000
	IDList          []string // Withdrawal IDs, up to 45
	StartTime       time.Time
	EndTime         time.Time
}

// WithdrawalRecord holds one withdrawal
type WithdrawalRecord struct {
	ID              string         `json:"id"`
	Amount          types.Number   `json:"amount"`
	TransactionFee  types.Number   `json:"transactionFee"`
	Coin            currency.Code  `json:"coin"`
	Status          uint64         `json:"status"`
	Address         string         `json:"address"`
	TxID            string         `json:"txId"`
	ApplyTime       types.DateTime `json:"applyTime"`
	Network         string         `json:"network"`
	TransferType    uint64         `json:"transferType"`
	WithdrawOrderID string         `json:"withdrawOrderId"`
	Info            string         `json:"info"` // Reason for a failed withdrawal
	ConfirmNumber   uint64         `json:"confirmNo"`
	WalletType      uint64         `json:"walletType"`
	TxKey           string         `json:"txKey"`
	CompleteTime    types.DateTime `json:"completeTime"`
}

// DelistSchedule holds the spot symbols delisted at one time
type DelistSchedule struct {
	DelistTime types.Time `json:"delistTime"`
	Symbols    []string   `json:"symbols"`
}

// SystemStatusResponse holds the system status
type SystemStatusResponse struct {
	Status  uint64 `json:"status"` // 0 normal, 1 system maintenance
	Message string `json:"msg"`
}

// LocalEntityDepositHistoryRequest holds the parameters for a travel rule deposit history query
type LocalEntityDepositHistoryRequest struct {
	TravelRuleRecordIDs  []uint64
	TxIDs                []string
	TransactionIDs       []uint64 // Wallet transaction IDs
	Network              string
	Coin                 currency.Code
	TravelRuleStatus     *uint64 // 0 completed, 1 pending, 2 failed
	PendingQuestionnaire bool    // Only returns deposits awaiting a questionnaire
	StartTime            time.Time
	EndTime              time.Time
	Offset               uint64
	Limit                uint64 // Up to 1000, default 1000
}

// LocalEntityDepositRecord holds one deposit of a local entity that requires the travel rule
type LocalEntityDepositRecord struct {
	TravelRuleRecordID   uint64        `json:"trId"`
	TransactionID        uint64        `json:"tranId"`
	Amount               types.Number  `json:"amount"`
	Coin                 currency.Code `json:"coin"`
	Network              string        `json:"network"`
	DepositStatus        uint64        `json:"depositStatus"`
	TravelRuleStatus     uint64        `json:"travelRuleStatus"`
	TravelRuleStatusV2   string        `json:"travelRuleStatusV2"` // Combined travel rule and sanctions screening status
	Address              string        `json:"address"`
	AddressTag           string        `json:"addressTag"`
	TxID                 string        `json:"txId"`
	InsertTime           types.Time    `json:"insertTime"`
	CompleteTime         types.Time    `json:"completeTime"`
	TransferType         uint64        `json:"transferType"`
	ConfirmTimes         string        `json:"confirmTimes"`
	RequireQuestionnaire bool          `json:"requireQuestionnaire"`
	Questionnaire        string        `json:"questionnaire"` // JSON encoded questionnaire answers
}

// DepositQuestionnaireResponse holds the result of a deposit questionnaire submission
type DepositQuestionnaireResponse struct {
	TravelRuleRecordID uint64 `json:"trId"`
	Accepted           bool   `json:"accepted"`
	Info               string `json:"info"`
}

// VASP holds a virtual asset service provider of the travel rule
type VASP struct {
	VASPName   string `json:"vaspName"`
	VASPCode   string `json:"vaspCode"`
	Identifier string `json:"identifier"` // Value for a questionnaire's vasp field
}

// LocalEntityWithdrawHistoryRequest holds the parameters for a travel rule withdrawal history query
type LocalEntityWithdrawHistoryRequest struct {
	TravelRuleRecordIDs []uint64
	TxIDs               []string
	WithdrawOrderID     string
	Network             string
	Coin                currency.Code
	TravelRuleStatus    *uint64 // 0 completed, 1 pending, 2 failed
	Offset              uint64
	Limit               uint64 // Up to 1000, default 1000
	StartTime           time.Time
	EndTime             time.Time
}

// LocalEntityWithdrawalRecord holds one withdrawal of a local entity that requires the travel rule
type LocalEntityWithdrawalRecord struct {
	ID                 string         `json:"id"`
	TravelRuleRecordID uint64         `json:"trId"`
	Amount             types.Number   `json:"amount"`
	TransactionFee     types.Number   `json:"transactionFee"`
	Coin               currency.Code  `json:"coin"`
	WithdrawalStatus   uint64         `json:"withdrawalStatus"`
	TravelRuleStatus   uint64         `json:"travelRuleStatus"`
	Address            string         `json:"address"`
	TxID               string         `json:"txId"`
	ApplyTime          types.DateTime `json:"applyTime"`
	Network            string         `json:"network"`
	TransferType       uint64         `json:"transferType"`
	WithdrawOrderID    string         `json:"withdrawOrderId"`
	Info               string         `json:"info"` // Reason for a failed withdrawal
	ConfirmNumber      uint64         `json:"confirmNo"`
	WalletType         uint64         `json:"walletType"`
	TxKey              string         `json:"txKey"`
	Questionnaire      string         `json:"questionnaire"` // JSON encoded questionnaire answers
	CompleteTime       types.DateTime `json:"completeTime"`
}

// HistoricalDataDownloadLinkRequest holds the parameters for a futures tick-level order book download link query
type HistoricalDataDownloadLinkRequest struct {
	Symbol    currency.Pair
	DataType  string // T_DEPTH for tick-level order book data, S_DEPTH for order book snapshots
	StartTime time.Time
	EndTime   time.Time
}

// HistoricalDataDownloadLinkResponse holds futures tick-level order book download links
type HistoricalDataDownloadLinkResponse struct {
	Data []HistoricalDataDownloadLink `json:"data"`
}

// HistoricalDataDownloadLink holds the download link of one day's order book data
type HistoricalDataDownloadLink struct {
	Day string `json:"day"`
	URL string `json:"url"`
}
