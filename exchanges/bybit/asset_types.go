package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FundingAccountTransactionHistoryRequest holds the Funding Account Transaction History filters; StartTime and EndTime
// go together and span at most 7 days
type FundingAccountTransactionHistoryRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// FundingAccountTransactionHistoryResponse is a page of Funding wallet transactions
type FundingAccountTransactionHistoryResponse struct {
	NextPageCursor string                      `json:"nextPageCursor"`
	List           []FundingAccountTransaction `json:"list"`
}

// FundingAccountTransaction is one Funding wallet transaction
type FundingAccountTransaction struct {
	MemberID string        `json:"memberId"`
	Currency currency.Code `json:"currency"`
	// IODirection is I for a credit and O for a debit
	IODirection       string       `json:"ioDirection"`
	TransactionAmount types.Number `json:"txnAmt"`
	// AfterAmount is the balance after the transaction
	AfterAmount types.Number `json:"afterAmt"`
	CreateTime  types.Time   `json:"createTime"`
	// ShowBusinessType and Description are localisation keys; the English fields hold their English text
	ShowBusinessType        string `json:"showBusiType"`
	ShowBusinessTypeEnglish string `json:"showBusiTypeEn"`
	Description             string `json:"description"`
	DescriptionEnglish      string `json:"descriptionEn"`
	// CurrencyCursor is Bybit's currcCursor, documented only as a unique ID for dropping duplicates; the docs never say
	// what currc stands for, and the record's currency is the likeliest reading
	CurrencyCursor string `json:"currcCursor"`
}

// DeliveryRecordRequest holds the Get Delivery Record filters; Category is required
type DeliveryRecordRequest struct {
	Category  string
	Symbol    string
	StartTime time.Time
	EndTime   time.Time
	// ExpiryDate filters by expiry date, taken in UTC; the API rejects dates more than 2 years ago
	ExpiryDate time.Time
	Limit      uint64
	Cursor     string
}

// DeliveryRecordResponse is a page of delivery records
type DeliveryRecordResponse struct {
	Category       string           `json:"category"`
	List           []DeliveryRecord `json:"list"`
	NextPageCursor string           `json:"nextPageCursor"`
}

// DeliveryRecord is the delivery of one futures or option position
type DeliveryRecord struct {
	DeliveryTime  types.Time   `json:"deliveryTime"`
	Symbol        string       `json:"symbol"`
	Side          string       `json:"side"`
	Position      types.Number `json:"position"`
	EntryPrice    types.Number `json:"entryPrice"`
	DeliveryPrice types.Number `json:"deliveryPrice"`
	Strike        types.Number `json:"strike"`
	Fee           types.Number `json:"fee"`
	// DeliveryRealisedPNL is the realised profit and loss of the delivery
	DeliveryRealisedPNL types.Number `json:"deliveryRpl"`
}

// USDCSessionSettlementRequest holds the Get USDC Session Settlement filters; Category is required
type USDCSessionSettlementRequest struct {
	Category  string
	Symbol    string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// USDCSessionSettlementResponse is a page of USDC session settlement records
type USDCSessionSettlementResponse struct {
	Category       string                  `json:"category"`
	List           []USDCSessionSettlement `json:"list"`
	NextPageCursor string                  `json:"nextPageCursor"`
}

// USDCSessionSettlement is the settlement of one USDC position for a session
type USDCSessionSettlement struct {
	Symbol string       `json:"symbol"`
	Side   string       `json:"side"`
	Size   types.Number `json:"size"`
	// SessionAveragePrice is the settlement price
	SessionAveragePrice types.Number `json:"sessionAvgPrice"`
	MarkPrice           types.Number `json:"markPrice"`
	RealisedPNL         types.Number `json:"realisedPnl"`
	CreatedTime         types.Time   `json:"createdTime"`
}

// CoinExchangeRecordsRequest holds the Get Coin Exchange Records filters
type CoinExchangeRecordsRequest struct {
	FromCoin currency.Code
	ToCoin   currency.Code
	Limit    uint64
	Cursor   string
}

// CoinExchangeRecordsResponse is a page of coin exchange records
type CoinExchangeRecordsResponse struct {
	NextPageCursor string               `json:"nextPageCursor"`
	OrderBody      []CoinExchangeRecord `json:"orderBody"`
}

// CoinExchangeRecord is one coin exchange
type CoinExchangeRecord struct {
	FromCoin              currency.Code `json:"fromCoin"`
	FromAmount            types.Number  `json:"fromAmount"`
	ToCoin                currency.Code `json:"toCoin"`
	ToAmount              types.Number  `json:"toAmount"`
	ExchangeRate          types.Number  `json:"exchangeRate"`
	CreatedTime           types.Time    `json:"createdTime"`
	ExchangeTransactionID string        `json:"exchangeTxId"`
}

// CoinInfoResponse lists coins and the chains each can move on
type CoinInfoResponse struct {
	Rows []CoinInfo `json:"rows"`
}

// CoinInfo describes one coin and its chains
type CoinInfo struct {
	Name   string        `json:"name"`
	Coin   currency.Code `json:"coin"`
	Chains []CoinChain   `json:"chains"`
	// RemainAmount is deprecated and should be ignored
	RemainAmount types.Number `json:"remainAmount"`
}

// CoinChain describes depositing and withdrawing a coin on one chain
type CoinChain struct {
	Chain     string `json:"chain"`
	ChainType string `json:"chainType"`
	// Confirmation is the number of confirmations after which a deposit is credited
	Confirmation types.Number `json:"confirmation"`
	// WithdrawFee is empty, decoding as 0, when the coin cannot be withdrawn
	WithdrawFee types.Number `json:"withdrawFee"`
	DepositMin  types.Number `json:"depositMin"`
	WithdrawMin types.Number `json:"withdrawMin"`
	// MinAccuracy is the precision of deposits and withdrawals
	MinAccuracy types.Number `json:"minAccuracy"`
	// ChainDeposit and ChainWithdraw are 1 when the chain is open and 0 when suspended. Some chains report an empty
	// status, which a boolean type would reject, failing the whole list
	ChainDeposit  string `json:"chainDeposit"`
	ChainWithdraw string `json:"chainWithdraw"`
	// WithdrawPercentageFee is a fraction: 0.022 means 2.2%
	WithdrawPercentageFee types.Number `json:"withdrawPercentageFee"`
	// ContractAddress is empty when the coin has no contract on the chain
	ContractAddress string `json:"contractAddress"`
	// SafeConfirmNumber is the number of confirmations after which a deposit can be withdrawn in full
	SafeConfirmNumber types.Number `json:"safeConfirmNumber"`
	// WithdrawMax is the largest withdrawal per transaction on the chain; -1 means no limit
	WithdrawMax types.Number `json:"withdrawMax"`
}

// SubUIDResponse lists the sub UIDs under the master UID
type SubUIDResponse struct {
	SubMemberIDs []string `json:"subMemberIds"`
	// TransferableSubMemberIDs lists the sub UIDs with universal transfer enabled
	TransferableSubMemberIDs []string `json:"transferableSubMemberIds"`
}

// PortfolioMarginInfoResponse is the portfolio margin account's wallet and its P&L ranges per base coin
type PortfolioMarginInfoResponse struct {
	Wallet        PortfolioMarginWallet          `json:"wallet"`
	AssetPNLRange []PortfolioMarginAssetPNLRange `json:"assetPnlRange"`
}

// PortfolioMarginWallet is the portfolio margin account's wallet balance and margin
type PortfolioMarginWallet struct {
	Equity                       types.Number `json:"equity"`
	CashBalance                  types.Number `json:"cashBalance"`
	MarginBalance                types.Number `json:"marginBalance"`
	AvailableBalance             types.Number `json:"availableBalance"`
	AccountInitialMargin         types.Number `json:"accountIM"`
	AccountMaintenanceMargin     types.Number `json:"accountMM"`
	AccountMaintenanceMarginRate types.Number `json:"accountMMRate"`
	AccountInitialMarginRate     types.Number `json:"accountIMRate"`
}

// PortfolioMarginAssetPNLRange holds the P&L ranges of one base coin's positions
type PortfolioMarginAssetPNLRange struct {
	BaseCoin                      currency.Code                              `json:"baseCoin"`
	TotalPNLRanges                PortfolioMarginTotalPNLRanges              `json:"totalPnlRanges"`
	PerpetualPositionPNLRanges    []PortfolioMarginPerpetualPositionPNLRange `json:"perpPositionPnlRanges"`
	OptionExpiryDatePNLRanges     []PortfolioMarginOptionExpiryDatePNLRange  `json:"optionExpiryDatePnlRanges"`
	Contingency                   PortfolioMarginContingency                 `json:"contingency"`
	Asset                         PortfolioMarginAsset                       `json:"asset"`
	MaxLossPriceMove              types.Number                               `json:"maxLossPriceMove"`
	MaxLossImpliedVolatilityShock types.Number                               `json:"maxLossIvShock"`
	TotalClosePositionFee         types.Number                               `json:"totalClosePzFee"`
	SpotHedgeInfo                 PortfolioMarginSpotHedgeInfo               `json:"spotHedgeInfo"`
	// MaxLossImpliedVolatilityShockList holds the IV shock of each price scale scenario
	MaxLossImpliedVolatilityShockList []types.Number `json:"maxLossIvShockList"`
}

// PortfolioMarginTotalPNLRanges holds a base coin's P&L ranges per contract type
type PortfolioMarginTotalPNLRanges struct {
	All       PortfolioMarginPNLRanges `json:"ALL"`
	Perpetual PortfolioMarginPNLRanges `json:"PERPETUAL"`
	Option    PortfolioMarginPNLRanges `json:"OPTION"`
}

// PortfolioMarginPNLRanges holds the P&L range data points of one contract type
type PortfolioMarginPNLRanges struct {
	PNLRanges []PortfolioMarginPNLRange `json:"pnlRanges"`
}

// PortfolioMarginPNLRange is the P&L at one price scale; options carry one value per IV shock scenario
type PortfolioMarginPNLRange struct {
	// PriceScale is a fractional price change: -0.1 means -10%
	PriceScale types.Number   `json:"priceScale"`
	PNLs       []types.Number `json:"pnls"`
}

// PortfolioMarginPerpetualPositionPNLRange holds the P&L ranges of one perpetual position
type PortfolioMarginPerpetualPositionPNLRange struct {
	SymbolName          string                    `json:"symbolName"`
	Position            types.Number              `json:"position"`
	PNLRanges           []PortfolioMarginPNLRange `json:"pnlRanges"`
	SessionAveragePrice types.Number              `json:"sessionAvgPrice"`
	MarkPrice           types.Number              `json:"markPrice"`
	OrderSize           types.Number              `json:"orderSize"`
	ContractType        uint64                    `json:"contractType"`
	SettleCoin          currency.Code             `json:"settleCoin"`
	SymbolAlias         string                    `json:"symbolAlias"`
}

// PortfolioMarginOptionExpiryDatePNLRange holds the P&L ranges of the option positions expiring on one date
type PortfolioMarginOptionExpiryDatePNLRange struct {
	// ExpiryDateRepresentation is the expiry date, such as 25SEP26
	ExpiryDateRepresentation string                                  `json:"expiryDateRepresentation"`
	PNLRanges                []PortfolioMarginPNLRange               `json:"pnlRanges"`
	OptionPositionPNLRanges  []PortfolioMarginOptionPositionPNLRange `json:"optionPositionPnlRanges"`
}

// PortfolioMarginOptionPositionPNLRange holds the P&L ranges of one option position
type PortfolioMarginOptionPositionPNLRange struct {
	SymbolName          string                    `json:"symbolName"`
	Position            types.Number              `json:"position"`
	PNLRanges           []PortfolioMarginPNLRange `json:"pnlRanges"`
	SessionAveragePrice types.Number              `json:"sessionAvgPrice"`
	MarkPrice           types.Number              `json:"markPrice"`
	OrderSize           types.Number              `json:"orderSize"`
	ContractType        uint64                    `json:"contractType"`
	SettleCoin          currency.Code             `json:"settleCoin"`
}

// PortfolioMarginContingency is a base coin's contingency margin
type PortfolioMarginContingency struct {
	OptionContingency      types.Number `json:"optionContingency"`
	FutureDeltaContingency types.Number `json:"futureDeltaContingency"`
	OptionVegaContingency  types.Number `json:"optionVegaContingency"`
	// ContingencyComponents is the total of the contingency components
	ContingencyComponents types.Number `json:"contingencyComponents"`
	USDTUSDCContingency   types.Number `json:"usdtUsdcContingency"`
	FutureContingency     types.Number `json:"futureContingency"`
}

// PortfolioMarginAsset is a base coin's margin
type PortfolioMarginAsset struct {
	Coin                   currency.Code `json:"coin"`
	AssetInitialMargin     types.Number  `json:"assetIM"`
	AssetMaintenanceMargin types.Number  `json:"assetMM"`
}

// PortfolioMarginSpotHedgeInfo describes the spot holdings hedging a base coin's positions
type PortfolioMarginSpotHedgeInfo struct {
	HedgeSpotSize types.Number              `json:"hedgeSpotSize"`
	WalletBalance types.Number              `json:"walletBalance"`
	USDIndexPrice types.Number              `json:"usdIndexPrice"`
	PNLRanges     []PortfolioMarginPNLRange `json:"pnlRanges"`
}

// TotalMembersAssetsResponse is the total value of the master account and its sub-accounts
type TotalMembersAssetsResponse struct {
	Total types.Number `json:"total"`
	// QuoteTotal is Total valued in the requested coin
	QuoteTotal types.Number `json:"quoteTotal"`
	// Stat is 0 when every downstream query succeeded, otherwise an error code
	Stat int64          `json:"stat"`
	List []MemberAssets `json:"list"`
}

// MemberAssets is the value of one member account
type MemberAssets struct {
	UID uint64 `json:"uid"`
	// IsMaster is only returned for the master account
	IsMaster bool `json:"isM"`
	// Type is the sub-account type, only returned for sub-accounts: 1 normal, 2 MT4, 3 trading bot, 4 copy trading
	// leader, 5 copy trading follower, 6 custodial, 7 fund custodial, 8 demo, 9 Copy Pro, 10 MT5 follow, 11 Earn vendor,
	// 12 Earn fund
	Type uint64 `json:"type"`
	// Stat is 0 when every downstream query succeeded, otherwise an error code
	Stat int64 `json:"stat"`
	// OriginalBalance is valued in USD
	OriginalBalance types.Number `json:"origb"`
	// QuoteBalance is valued in the requested coin
	QuoteBalance types.Number          `json:"quoteb"`
	Items        []MemberAccountAssets `json:"items"`
}

// MemberAccountAssets is the value of one account type of a member account
type MemberAccountAssets struct {
	// Type is the account type, such as ACCOUNT_TYPE_FUND or ACCOUNT_TYPE_UNIFIED
	Type string `json:"type"`
	// OriginalBalance is valued in USD
	OriginalBalance types.Number `json:"origb"`
	// QuoteBalance is valued in the requested coin
	QuoteBalance types.Number `json:"quoteb"`
	// Stat is 0 when every downstream query succeeded, otherwise an error code
	Stat int64 `json:"stat"`
}

// SingleCoinBalanceRequest holds the Get Single Coin Balance parameters; AccountType and Coin are required, and
// WithLTVTransferSafeAmount needs ToAccountType
type SingleCoinBalanceRequest struct {
	// MemberID is required to query a sub UID's balance with the master API key
	MemberID string
	// ToMemberID is required to query the amount transferable to another UID
	ToMemberID  string
	AccountType string
	// ToAccountType is required to query the amount transferable to another account type
	ToAccountType string
	Coin          currency.Code
	WithBonus     bool
	// WithTransferSafeAmount queries the amount not held back by delayed withdrawal
	WithTransferSafeAmount bool
	// WithLTVTransferSafeAmount queries the amount an institutional loan account can transfer at its risk level
	WithLTVTransferSafeAmount bool
}

// SingleCoinBalanceResponse is one coin's balance in one account
type SingleCoinBalanceResponse struct {
	AccountType  string            `json:"accountType"`
	BusinessType uint64            `json:"bizType"`
	AccountID    string            `json:"accountId"`
	MemberID     string            `json:"memberId"`
	Balance      SingleCoinBalance `json:"balance"`
}

// SingleCoinBalance is one coin's balance, with the optional safe amounts empty, decoding as 0, when not queried
type SingleCoinBalance struct {
	Coin            currency.Code `json:"coin"`
	WalletBalance   types.Number  `json:"walletBalance"`
	TransferBalance types.Number  `json:"transferBalance"`
	Bonus           types.Number  `json:"bonus"`
	// TransferSafeAmount is the amount that can be transferred
	TransferSafeAmount types.Number `json:"transferSafeAmount"`
	// LTVTransferSafeAmount is the amount an institutional loan account can transfer
	LTVTransferSafeAmount types.Number `json:"ltvTransferSafeAmount"`
}

// AllCoinsBalanceRequest holds the Get All Coins Balance parameters; AccountType is required, and the Unified account
// needs Coins, at most 10 of them
type AllCoinsBalanceRequest struct {
	// MemberID is required to query a sub-account's balance with the master API key
	MemberID    string
	AccountType string
	Coins       []currency.Code
	WithBonus   bool
}

// AllCoinsBalanceResponse lists the coin balances of one account
type AllCoinsBalanceResponse struct {
	AccountType string        `json:"accountType"`
	MemberID    string        `json:"memberId"`
	Balance     []CoinBalance `json:"balance"`
}

// CoinBalance is one coin's balance
type CoinBalance struct {
	Coin            currency.Code `json:"coin"`
	WalletBalance   types.Number  `json:"walletBalance"`
	TransferBalance types.Number  `json:"transferBalance"`
	Bonus           types.Number  `json:"bonus"`
}

// WithdrawableAmountResponse is a coin's withdrawable amount per wallet
type WithdrawableAmountResponse struct {
	// LimitAmountUSD is the amount frozen by risk control, in USD
	LimitAmountUSD     types.Number               `json:"limitAmountUsd"`
	WithdrawableAmount WithdrawableAmountByWallet `json:"withdrawableAmount"`
}

// WithdrawableAmountByWallet holds a coin's withdrawable amount per wallet; a wallet that cannot withdraw the coin, or
// a removed Spot wallet, is nil
type WithdrawableAmountByWallet struct {
	Spot *WalletWithdrawableAmount `json:"SPOT"`
	Fund *WalletWithdrawableAmount `json:"FUND"`
	UTA  *WalletWithdrawableAmount `json:"UTA"`
	Earn *WalletWithdrawableAmount `json:"EARN"`
}

// WalletWithdrawableAmount is a coin's withdrawable amount in one wallet
type WalletWithdrawableAmount struct {
	Coin               currency.Code `json:"coin"`
	WithdrawableAmount types.Number  `json:"withdrawableAmount"`
	AvailableBalance   types.Number  `json:"availableBalance"`
}

// AssetOverviewRequest holds the Asset Overview filters
type AssetOverviewRequest struct {
	// MemberID is required to query a sub-account with the master API key
	MemberID string
	// ValuationCurrency is the fiat currency to value holdings in, USD when empty
	ValuationCurrency currency.Code
	// AccountType limits the overview to one account type, such as FundingAccount or UnifiedTradingAccount
	AccountType string
}

// AssetOverviewResponse is the total equity and holdings of each account type
type AssetOverviewResponse struct {
	TotalEquity types.Number           `json:"totalEquity"`
	List        []AssetOverviewAccount `json:"list"`
}

// AssetOverviewAccount is the equity of one account type; accounts with product sub-categories, such as Earn or the
// Unified account, break holdings down in Categories instead of CoinDetail
type AssetOverviewAccount struct {
	AccountType       string                  `json:"accountType"`
	TotalEquity       types.Number            `json:"totalEquity"`
	ValuationCurrency currency.Code           `json:"valuationCurrency"`
	SnapshotTime      types.Time              `json:"snapshotTime"`
	CoinDetail        []AssetOverviewCoin     `json:"coinDetail"`
	Categories        []AssetOverviewCategory `json:"categories"`
}

// AssetOverviewCoin is the equity of one coin in an account
type AssetOverviewCoin struct {
	Coin   currency.Code `json:"coin"`
	Equity types.Number  `json:"equity"`
}

// AssetOverviewCategory is the equity of one product sub-category of an account
type AssetOverviewCategory struct {
	Category   string                      `json:"category"`
	Equity     types.Number                `json:"equity"`
	CoinDetail []AssetOverviewCategoryCoin `json:"coinDetail"`
}

// AssetOverviewCategoryCoin is the equity of one coin in a sub-category
type AssetOverviewCategoryCoin struct {
	Coin   currency.Code `json:"coin"`
	Equity types.Number  `json:"equity"`
	// ExtendedMap is only returned for the Alpha account's farm category
	ExtendedMap *AssetOverviewExtendedMap `json:"extMap"`
}

// AssetOverviewExtendedMap describes an Alpha farm liquidity position
type AssetOverviewExtendedMap struct {
	PriceUpper types.Number  `json:"priceUpper"`
	PriceLower types.Number  `json:"priceLower"`
	EquityUnit currency.Code `json:"equityUnit"`
}
