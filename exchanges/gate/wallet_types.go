package gate

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CurrencyChainResponse currency chain detail.
type CurrencyChainResponse struct {
	Chain               string        `json:"chain"`
	ChineseChainName    string        `json:"name_cn"`
	ChainName           string        `json:"name_en"`
	ContractAddress     string        `json:"contract_address"`
	Disabled            types.Boolean `json:"is_disabled"`
	DepositDisabled     types.Boolean `json:"is_deposit_disabled"`
	WithdrawDisabled    types.Boolean `json:"is_withdraw_disabled"`
	WithdrawalPrecision types.Number  `json:"decimal"`
	RequiresTag         types.Boolean `json:"is_tag"` // Whether deposits and withdrawals need a tag
}

// ConvertibleSmallBalanceResponse holds a small balance that can be converted into GT
type ConvertibleSmallBalanceResponse struct {
	Currency         currency.Code `json:"currency"`
	AvailableBalance types.Number  `json:"available_balance"`
	EstimatedAsBTC   types.Number  `json:"estimated_as_btc"`
	ConvertibleToGT  types.Number  `json:"convertible_to_gt"`
}

// convertSmallBalancesRequest holds the currencies to convert to GT, or every small balance when IsAll is set
type convertSmallBalancesRequest struct {
	Currency []string `json:"currency"`
	IsAll    bool     `json:"is_all"`
}

// SmallCurrencyBalanceResponse holds a small currency balance detail
type SmallCurrencyBalanceResponse struct {
	ID         string        `json:"id"`
	CreateTime types.Time    `json:"create_time"`
	Currency   currency.Code `json:"currency"`
	Amount     types.Number  `json:"amount"`
	GTAmount   types.Number  `json:"gt_amount"`
}

// WithdrawalRecordsRequest holds query parameters for retrieving withdrawal records
type WithdrawalRecordsRequest struct {
	Currency currency.Code
	// WithdrawalID is a withdrawal record ID starting with "w"; Gate ignores the time range when it is set
	WithdrawalID string
	// AssetClass selects the main zone (SPOT) or the innovation zone (PILOT)
	AssetClass        string
	WithdrawalOrderID string
	From              time.Time
	To                time.Time
	Limit             uint64
	Offset            uint64
}

// UIDTransferHistoryRequest holds query parameters for retrieving UID transfer records
type UIDTransferHistoryRequest struct {
	ID     uint64
	From   time.Time
	To     time.Time
	Limit  uint64
	Offset uint64
	// TransactionType selects withdraw (Gate's default) or deposit records
	TransactionType string
}

// UIDTransferRecordResponse holds a UID transfer record
type UIDTransferRecordResponse struct {
	ID              uint64        `json:"id"`
	InitiatorUserID uint64        `json:"push_uid"`
	RecipientUserID uint64        `json:"receive_uid"`
	Currency        currency.Code `json:"currency"`
	Amount          types.Number  `json:"amount"`
	CreateTime      types.Time    `json:"create_time"`
	Status          string        `json:"status"`
	Message         string        `json:"message"`
	TransactionType string        `json:"transaction_type"`
}

// CurrencyDepositAddressInfoResponse represents a crypto deposit address
type CurrencyDepositAddressInfoResponse struct {
	Currency            currency.Code            `json:"currency"`
	Address             string                   `json:"address"`
	MinDepositAmount    types.Number             `json:"min_deposit_amount"`
	MultichainAddresses []*MultiChainAddressItem `json:"multichain_addresses"`
}

// MultiChainAddressItem represents a multi-chain address item
type MultiChainAddressItem struct {
	Chain        string `json:"chain"`
	Address      string `json:"address"`
	PaymentID    string `json:"payment_id"`   // Tag or memo some currencies require when depositing
	PaymentName  string `json:"payment_name"` // Kind of PaymentID, either Tag or Memo
	ObtainFailed uint64 `json:"obtain_failed"`
	MinConfirms  uint64 `json:"min_confirms"`
}

// DepositRecordResponse represents deposit record item
type DepositRecordResponse struct {
	ID            string        `json:"id"`
	Timestamp     types.Time    `json:"timestamp"`
	Currency      currency.Code `json:"currency"`
	Address       string        `json:"address"`
	TransactionID string        `json:"txid"`
	Amount        types.Number  `json:"amount"`
	Memo          string        `json:"memo"`
	Status        string        `json:"status"`
	RefundStatus  string        `json:"refund_status"`
	Chain         string        `json:"chain"`
	Fee           types.Number  `json:"fee"`
}

// TradingAccountTransferResponse holds a transfer between the user's trading accounts
type TradingAccountTransferResponse struct {
	TransactionID string        `json:"tx_id"`
	Status        string        `json:"status"`
	Currency      currency.Code `json:"currency"`
	Amount        types.Number  `json:"amount"`
	FromAccount   string        `json:"from_account"`
	ToAccount     string        `json:"to_account"`
	Settle        currency.Code `json:"settle"`        // Set for futures, delivery and options transfers
	CurrencyPair  currency.Pair `json:"currency_pair"` // Set for margin transfers
}

// SubAccountTransferRequest represents currency subaccount transfer request param
type SubAccountTransferRequest struct {
	Currency       currency.Code `json:"currency"`
	SubAccount     string        `json:"sub_account"`
	Direction      string        `json:"direction"`
	Amount         types.Number  `json:"amount"`
	SubAccountType string        `json:"sub_account_type,omitempty"`
	ClientOrderID  string        `json:"client_order_id,omitempty"`
}

// SubAccountTransferResponse represents transfer records between main and sub accounts
type SubAccountTransferResponse struct {
	MainAccountUserID string        `json:"uid"`
	Timestamp         types.Time    `json:"timest"`
	Source            string        `json:"source"`
	Currency          currency.Code `json:"currency"`
	SubAccount        string        `json:"sub_account"`
	TransferDirection string        `json:"direction"`
	Amount            types.Number  `json:"amount"`
	SubAccountType    string        `json:"sub_account_type"`
	ClientOrderID     string        `json:"client_order_id"`
	Status            string        `json:"status"`
}

// WithdrawalStatusResponse represents currency withdrawal status
type WithdrawalStatusResponse struct {
	Currency               currency.Code `json:"currency"`
	CurrencyName           string        `json:"name"`
	CurrencyNameChinese    string        `json:"name_cn"`
	DepositFee             types.Number  `json:"deposit"`
	WithdrawPercentage     DisplayNumber `json:"withdraw_percent"`
	FixedWithdrawalFee     types.Number  `json:"withdraw_fix"`
	WithdrawDayLimit       types.Number  `json:"withdraw_day_limit"`
	WithdrawDayLimitRemain types.Number  `json:"withdraw_day_limit_remain"`
	WithdrawAmountMini     types.Number  `json:"withdraw_amount_mini"`
	WithdrawEachTimeLimit  types.Number  `json:"withdraw_eachtime_limit"`
	// WithdrawFixOnChains holds the fixed withdrawal fee keyed by chain
	WithdrawFixOnChains map[string]types.Number `json:"withdraw_fix_on_chains"`
	// WithdrawPercentageOnChains holds the percentage withdrawal fee keyed by chain
	WithdrawPercentageOnChains map[string]DisplayNumber `json:"withdraw_percent_on_chains"`
}

// SubAccountBalancesResponse holds a sub-account's spot balances
type SubAccountBalancesResponse struct {
	UserID    string                         `json:"uid"`
	Available map[currency.Code]types.Number `json:"available"`
	Locking   map[currency.Code]types.Number `json:"locking"`
}

// AccountMargin represents margin account for specific user and several currencies
type AccountMargin struct {
	User          uint64       `json:"user"`
	UpdateTime    types.Time   `json:"update_time"`
	UpdateID      uint64       `json:"update_id"`
	Total         types.Number `json:"total"`
	UnrealisedPNL types.Number `json:"unrealised_pnl"`
	// Deprecated: Gate has deprecated it
	PositionMargin        types.Number  `json:"position_margin"`
	OrderMargin           types.Number  `json:"order_margin"`
	Available             types.Number  `json:"available"`
	Point                 types.Number  `json:"point"`
	Currency              currency.Code `json:"currency"`
	InDualMode            bool          `json:"in_dual_mode"`
	EnableCredit          bool          `json:"enable_credit"`
	PositionInitialMargin types.Number  `json:"position_initial_margin"`
	MaintenanceMargin     types.Number  `json:"maintenance_margin"`
	Bonus                 types.Number  `json:"bonus"`
	// Deprecated: Gate has deprecated it
	EnableEvolvedClassic       bool         `json:"enable_evolved_classic"`
	CrossOrderMargin           types.Number `json:"cross_order_margin"`
	CrossInitialMargin         types.Number `json:"cross_initial_margin"`
	CrossMaintenanceMargin     types.Number `json:"cross_maintenance_margin"`
	CrossUnrealisedPNL         types.Number `json:"cross_unrealised_pnl"`
	CrossAvailable             types.Number `json:"cross_available"`
	CrossMarginBalance         types.Number `json:"cross_margin_balance"`
	CrossMaintenanceMarginRate types.Number `json:"cross_mmr"`
	CrossInitialMarginRate     types.Number `json:"cross_imr"`
	CrossVirtualUnrealisedPNL  types.Number `json:"cross_virtual_unrealised_pnl"`
	IsolatedPositionMargin     types.Number `json:"isolated_position_margin"`
	// Deprecated: Gate has deprecated it
	EnableNewDualMode             bool                `json:"enable_new_dual_mode"`
	MarginMode                    uint8               `json:"margin_mode"`
	MarginModeName                string              `json:"margin_mode_name"`
	EnableTieredMaintenanceMargin bool                `json:"enable_tiered_mm"`
	EnableDualPlus                bool                `json:"enable_dual_plus"`
	PositionMode                  string              `json:"position_mode"`
	PositionVoucherTotal          types.Number        `json:"position_voucher_total"`
	History                       MarginHistoryTotals `json:"history"`
}

// MarginHistoryTotals represents margin history totals
type MarginHistoryTotals struct {
	DepositAndWithdrawal      types.Number `json:"dnw"`
	PNL                       types.Number `json:"pnl"`
	Fee                       types.Number `json:"fee"`
	ReferrerRebates           types.Number `json:"refr"`
	FundingCost               types.Number `json:"fund"`
	PointDepositAndWithdrawal types.Number `json:"point_dnw"`
	PointFee                  types.Number `json:"point_fee"`
	PointReferrerRebates      types.Number `json:"point_refr"`
	BonusDepositAndWithdrawal types.Number `json:"bonus_dnw"`
	BonusOffset               types.Number `json:"bonus_offset"`
	CrossSettle               types.Number `json:"cross_settle"`
}

// FuturesSubAccountBalanceResponse holds a sub-account's perpetual futures balances
type FuturesSubAccountBalanceResponse struct {
	UserID string `json:"uid"`
	// Available is keyed by the lower case settlement currency, such as "usdt"
	Available map[string]*AccountMargin `json:"available"`
}

// SubAccountMarginBalanceResponse holds a sub-account's isolated margin balances
type SubAccountMarginBalanceResponse struct {
	UserID    string                         `json:"uid"`
	Available []*SubAccountMarginPairBalance `json:"available"`
}

// SubAccountMarginPairBalance holds a sub-account's isolated margin balances for one currency pair
type SubAccountMarginPairBalance struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	AccountType  string        `json:"account_type"` // mmr for a maintenance margin rate account or inactive when the market is not activated
	Leverage     types.Number  `json:"leverage"`
	Locked       bool          `json:"locked"`
	// Deprecated: use MaintenanceMarginRate
	Risk                  types.Number          `json:"risk"`
	MaintenanceMarginRate types.Number          `json:"mmr"`
	Base                  MarginCurrencyBalance `json:"base"`
	Quote                 MarginCurrencyBalance `json:"quote"`
}

// TransferStatusResponse holds an asset transfer status
type TransferStatusResponse struct {
	TransactionID string `json:"tx_id"`
	Status        string `json:"status"`
}

// MarginCurrencyBalance represents a currency balance detail information.
type MarginCurrencyBalance struct {
	Currency       currency.Code `json:"currency"`
	Available      types.Number  `json:"available"`
	Locked         types.Number  `json:"locked"`
	BorrowedAmount types.Number  `json:"borrowed"`
	UnpaidInterest types.Number  `json:"interest"`
}

// SubAccountCrossMarginInfoResponse represents subaccount's cross_margin account info
type SubAccountCrossMarginInfoResponse struct {
	UserID    string                       `json:"uid"`
	Available *SubAccountCrossMarginDetail `json:"available"`
}

// SubAccountCrossMarginDetail represents a sub-account cross margin detail
type SubAccountCrossMarginDetail struct {
	UserID                     uint64                        `json:"user_id"` // 0 when the sub-account has not opened a cross margin account
	Locked                     bool                          `json:"locked"`
	Total                      types.Number                  `json:"total"`
	Borrowed                   types.Number                  `json:"borrowed"`
	UnpaidInterestUSDT         types.Number                  `json:"interest"` // The sum of every currency's interest x price x discount
	BorrowedNet                types.Number                  `json:"borrowed_net"`
	TotalNetAssets             types.Number                  `json:"net"`
	Leverage                   types.Number                  `json:"leverage"`
	Risk                       types.Number                  `json:"risk"`
	TotalInitialMargin         types.Number                  `json:"total_initial_margin"`
	TotalMarginBalance         types.Number                  `json:"total_margin_balance"`
	TotalMaintenanceMargin     types.Number                  `json:"total_maintenance_margin"`
	TotalInitialMarginRate     types.Number                  `json:"total_initial_margin_rate"`
	TotalMaintenanceMarginRate types.Number                  `json:"total_maintenance_margin_rate"`
	TotalAvailableMargin       types.Number                  `json:"total_available_margin"`
	CurrencyBalances           map[string]CrossMarginBalance `json:"balances"`
}

// CrossMarginBalance represents cross-margin currency balance detail
type CrossMarginBalance struct {
	Available           types.Number `json:"available"`
	Freeze              types.Number `json:"freeze"`
	Borrowed            types.Number `json:"borrowed"`
	Interest            types.Number `json:"interest"`
	Total               types.Number `json:"total"`
	BorrowedNet         types.Number `json:"borrowed_net"`
	TotalNetAssetInUSDT types.Number `json:"net"`
	PositionLeverage    types.Number `json:"leverage"`
	Risk                types.Number `json:"risk"` // Risk percentage; Liquidation is triggered when this falls below required margin. Calculation: total / (borrowed+interest)
}

// SavedAddressesRequest holds query parameters for retrieving saved withdrawal addresses
type SavedAddressesRequest struct {
	Currency currency.Code
	Chain    string
	// Verified filters by address type: "1" for verified addresses, "0" for normal addresses and empty for both
	Verified string
	Limit    uint64
	Page     uint64
}

// WalletSavedAddressResponse represents currency saved address
type WalletSavedAddressResponse struct {
	Currency currency.Code `json:"currency"`
	Chain    string        `json:"chain"`
	Address  string        `json:"address"`
	Name     string        `json:"name"`
	Tag      string        `json:"tag"`
	Verified types.Boolean `json:"verified"`
}

// PersonalTradingFeeResponse represents personal trading fee for specific currency pair
type PersonalTradingFeeResponse struct {
	UserID                         uint64       `json:"user_id"`
	TakerFee                       types.Number `json:"taker_fee"`
	MakerFee                       types.Number `json:"maker_fee"`
	RetailPriceImprovementMakerFee types.Number `json:"rpi_maker_fee"`
	GTDiscount                     bool         `json:"gt_discount"`
	GTTakerFee                     types.Number `json:"gt_taker_fee"`
	GTMakerFee                     types.Number `json:"gt_maker_fee"`
	LoanFee                        types.Number `json:"loan_fee"`
	PointType                      string       `json:"point_type"`
	FuturesTakerFee                types.Number `json:"futures_taker_fee"`
	FuturesMakerFee                types.Number `json:"futures_maker_fee"`
	FuturesRPIMakerFee             types.Number `json:"futures_rpi_maker_fee"`
	DeliveryTakerFee               types.Number `json:"delivery_taker_fee"`
	DeliveryMakerFee               types.Number `json:"delivery_maker_fee"`
	FeeDeductionType               uint64       `json:"debit_fee"` // 1 - GT deduction, 2 - Point card deduction, 3 - VIP rates
	RPIMarketMakerLevel            uint64       `json:"rpi_mm"`
}

// UsersAllAccountBalanceResponse represents user all account balances.
type UsersAllAccountBalanceResponse struct {
	Details map[string]CurrencyBalanceAmount `json:"details"`
	Total   CurrencyBalanceAmount            `json:"total"`
}

// CurrencyBalanceAmount represents currency and its amount.
type CurrencyBalanceAmount struct {
	Currency      currency.Code `json:"currency"`
	Amount        types.Number  `json:"amount"`
	UnrealisedPNL types.Number  `json:"unrealised_pnl"` // Only reported for futures, options, delivery and total balances
	Borrowed      types.Number  `json:"borrowed"`       // Only reported for margin and cross margin balances
}

// InterSubAccountTransferRequest represents parameters to transfer funds between sub-accounts.
type InterSubAccountTransferRequest struct {
	Currency                currency.Code `json:"currency"`
	SubAccountFromUserID    string        `json:"sub_account_from"`
	SubAccountFromAssetType asset.Item    `json:"sub_account_from_type"` // Spot, perpetual futures or delivery futures
	SubAccountToUserID      string        `json:"sub_account_to"`
	SubAccountToAssetType   asset.Item    `json:"sub_account_to_type"` // Spot, perpetual futures or delivery futures
	Amount                  types.Number  `json:"amount"`
}

// MarshalJSON sends the sub-account account types as Gate account names, which differ from asset.Item strings for
// futures assets
func (p *InterSubAccountTransferRequest) MarshalJSON() ([]byte, error) {
	from, err := walletAccountType(p.SubAccountFromAssetType)
	if err != nil {
		return nil, fmt.Errorf("%w: source sub-account", err)
	}
	to, err := walletAccountType(p.SubAccountToAssetType)
	if err != nil {
		return nil, fmt.Errorf("%w: destination sub-account", err)
	}
	type Alias InterSubAccountTransferRequest
	return json.Marshal(&struct {
		SubAccountFromAssetType string `json:"sub_account_from_type"`
		SubAccountToAssetType   string `json:"sub_account_to_type"`
		*Alias
	}{
		SubAccountFromAssetType: from,
		SubAccountToAssetType:   to,
		Alias:                   (*Alias)(p),
	})
}
