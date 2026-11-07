package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Portfolio types, which select portfolios in ListPortfolios and identify them in responses
const (
	// PortfolioTypeDefault is the account's default portfolio, the only one US derivatives trade in
	PortfolioTypeDefault = "DEFAULT"
	// PortfolioTypeConsumer is a further spot portfolio, such as one CreatePortfolio creates
	PortfolioTypeConsumer = "CONSUMER"
	// PortfolioTypeINTX is the international derivatives portfolio, whose perpetuals moved to the Global Derivatives
	// gateway on 1 October 2026
	PortfolioTypeINTX = "INTX"
)

// Portfolio account types, the kind of account a portfolio position is held in
const (
	PortfolioAccountTypeWallet                     = "ACCOUNT_TYPE_WALLET"
	PortfolioAccountTypeFiat                       = "ACCOUNT_TYPE_FIAT"
	PortfolioAccountTypeVault                      = "ACCOUNT_TYPE_VAULT"
	PortfolioAccountTypeCollateral                 = "ACCOUNT_TYPE_COLLATERAL"
	PortfolioAccountTypeDeFiYield                  = "ACCOUNT_TYPE_DEFI_YIELD"
	PortfolioAccountTypeLoanManagement             = "ACCOUNT_TYPE_LOAN_MANAGEMENT"
	PortfolioAccountTypeMultisig                   = "ACCOUNT_TYPE_MULTISIG"
	PortfolioAccountTypeMultisigVault              = "ACCOUNT_TYPE_MULTISIG_VAULT"
	PortfolioAccountTypeDerivativesTransfer        = "ACCOUNT_TYPE_DERIVATIVES_TRANSFER"
	PortfolioAccountTypeDerivativesEquity          = "ACCOUNT_TYPE_DERIVATIVES_EQUITY"
	PortfolioAccountTypeStakedFunds                = "ACCOUNT_TYPE_STAKED_FUNDS"
	PortfolioAccountTypePerpetualFutures           = "ACCOUNT_TYPE_PERP_FUTURES"
	PortfolioAccountTypePerpetualFuturesIsolated   = "ACCOUNT_TYPE_PERP_FUTURES_ISOLATED"
	PortfolioAccountTypeLentFunds                  = "ACCOUNT_TYPE_LENT_FUNDS"
	PortfolioAccountTypePerpetualFuturesCollateral = "ACCOUNT_TYPE_PERP_FUTURES_COLLATERAL"
	PortfolioAccountTypeDerivativesCash            = "ACCOUNT_TYPE_DERIVATIVES_CASH"
	PortfolioAccountTypeClawbackDebt               = "ACCOUNT_TYPE_CLAWBACK_DEBT"
	PortfolioAccountTypeSuspenseGeofence           = "ACCOUNT_TYPE_SUSPENSE_GEOFENCE"
	PortfolioAccountTypeRetailDeFiLend             = "ACCOUNT_TYPE_RETAIL_DEFI_LEND"
	PortfolioAccountTypeDeFiBorrowCollateral       = "ACCOUNT_TYPE_DEFI_BORROW_COLLATERAL"
	PortfolioAccountTypeFiatSavings                = "ACCOUNT_TYPE_FIAT_SAVINGS"
	PortfolioAccountTypeRaiseInvestments           = "ACCOUNT_TYPE_RAISE_INVESTMENTS"
	PortfolioAccountTypeCCMEquity                  = "ACCOUNT_TYPE_CCM_EQUITY"
	PortfolioAccountTypePredictionMarketsTransfer  = "ACCOUNT_TYPE_PREDICTION_MARKETS_TRANSFER"
	PortfolioAccountTypePredictionMarketsEquity    = "ACCOUNT_TYPE_PREDICTION_MARKETS_EQUITY"
	PortfolioAccountTypePredictionMarketsCFM       = "ACCOUNT_TYPE_PREDICTION_MARKETS_CFM"
	PortfolioAccountTypeCreditCardCollateral       = "ACCOUNT_TYPE_CREDIT_CARD_COLLATERAL" //nolint:gosec // An account type, which G101 mistakes for a credential as Credit holds cred
)

// Portfolio position sides, of a portfolio's perpetual and futures positions
const (
	PortfolioPositionSideLong  = "FUTURES_POSITION_SIDE_LONG"
	PortfolioPositionSideShort = "FUTURES_POSITION_SIDE_SHORT"
)

// Portfolio margin types, of a portfolio's perpetual positions
const (
	PortfolioMarginTypeCross    = "MARGIN_TYPE_CROSS"
	PortfolioMarginTypeIsolated = "MARGIN_TYPE_ISOLATED"
)

// Portfolio product types, the kind of product a portfolio's unified position is in
const (
	PortfolioProductTypeAll               = "PRODUCT_TYPE_ALL"
	PortfolioProductTypeSpot              = "PRODUCT_TYPE_SPOT"
	PortfolioProductTypeFCMFutures        = "PRODUCT_TYPE_FCM_FUTURES"
	PortfolioProductTypePerpetualFutures  = "PRODUCT_TYPE_PERP_FUTURES"
	PortfolioProductTypeINTXSpot          = "PRODUCT_TYPE_INTX_SPOT"
	PortfolioProductTypeDEX               = "PRODUCT_TYPE_DEX"
	PortfolioProductTypeEquity            = "PRODUCT_TYPE_EQUITY"
	PortfolioProductTypePredictionMarkets = "PRODUCT_TYPE_PREDICTION_MARKETS"
	PortfolioProductTypeCompositeCrypto   = "PRODUCT_TYPE_COMPOSITE_CRYPTO"
)

// Portfolio prediction market position sides
const (
	PortfolioPredictionMarketSideLong  = "PREDICTION_MARKETS_POSITION_SIDE_LONG"
	PortfolioPredictionMarketSideShort = "PREDICTION_MARKETS_POSITION_SIDE_SHORT"
)

// Portfolio identifies a portfolio. A CDP API key acts for the portfolios chosen when it was created, and Coinbase
// refuses a request for any other with PERMISSION_DENIED
type Portfolio struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
	// Type is one of the PortfolioType values
	Type string `json:"type"`
	// Deleted is set for a deleted portfolio, which stays viewable, read only
	Deleted bool `json:"deleted"`
	// SubaccountUUID is undocumented; the sandbox sends it empty
	SubaccountUUID string `json:"subaccount_uuid"`
}

// PortfoliosResponse holds a list of portfolios
type PortfoliosResponse struct {
	Portfolios []Portfolio `json:"portfolios"`
}

// PortfolioResponse holds a created or edited portfolio
type PortfolioResponse struct {
	Portfolio *Portfolio `json:"portfolio"`
}

// PortfolioNameWireRequest is the body of a request that names a portfolio
type PortfolioNameWireRequest struct {
	Name string `json:"name"`
}

// EditPortfolioRequest renames a portfolio
type EditPortfolioRequest struct {
	PortfolioUUID string
	Name          string
}

// PortfolioBreakdownRequest selects a portfolio's breakdown
type PortfolioBreakdownRequest struct {
	PortfolioUUID string
	// Currency, such as USD, is the currency the breakdown's fiat values are in
	Currency currency.Code
}

// PortfolioBreakdownResponse holds a portfolio's breakdown
type PortfolioBreakdownResponse struct {
	Breakdown *PortfolioBreakdown `json:"breakdown"`
}

// PortfolioBreakdown is a portfolio's balances and its positions of every product type
type PortfolioBreakdown struct {
	Portfolio         Portfolio               `json:"portfolio"`
	PortfolioBalances PortfolioBalances       `json:"portfolio_balances"`
	SpotPositions     []PortfolioSpotPosition `json:"spot_positions"`
	// PerpetualPositions are INTX perpetual positions, which moved to the Global Derivatives gateway on 1 October
	// 2026
	PerpetualPositions         []PortfolioPerpetualPosition `json:"perp_positions"`
	FuturesPositions           []PortfolioFuturesPosition   `json:"futures_positions"`
	PredictionMarketsPositions []PortfolioUnifiedPosition   `json:"prediction_markets_positions"`
	EquityPositions            []PortfolioEquityPosition    `json:"equity_positions"`
}

// PortfolioBalances are a portfolio's total balances
type PortfolioBalances struct {
	TotalBalance               Amount `json:"total_balance"`
	TotalFuturesBalance        Amount `json:"total_futures_balance"`
	TotalCashEquivalentBalance Amount `json:"total_cash_equivalent_balance"`
	TotalCryptoBalance         Amount `json:"total_crypto_balance"`
	FuturesUnrealisedPNL       Amount `json:"futures_unrealized_pnl"`
	PerpetualUnrealisedPNL     Amount `json:"perp_unrealized_pnl"`
	TotalEquitiesBalance       Amount `json:"total_equities_balance"`
}

// PortfolioSpotPosition is a portfolio's holding of an asset; its fiat values are in the breakdown's currency and its
// crypto values in the asset
type PortfolioSpotPosition struct {
	Asset                currency.Code `json:"asset"`
	AccountUUID          string        `json:"account_uuid"`
	TotalBalanceFiat     float64       `json:"total_balance_fiat"`
	TotalBalanceCrypto   float64       `json:"total_balance_crypto"`
	AvailableToTradeFiat float64       `json:"available_to_trade_fiat"`
	// Allocation is the position's share of the portfolio
	Allocation                float64 `json:"allocation"`
	CostBasis                 Amount  `json:"cost_basis"`
	AssetImageURL             string  `json:"asset_img_url"`
	IsCash                    bool    `json:"is_cash"`
	AverageEntryPrice         Amount  `json:"average_entry_price"`
	AssetUUID                 string  `json:"asset_uuid"`
	AvailableToTradeCrypto    float64 `json:"available_to_trade_crypto"`
	UnrealisedPNL             float64 `json:"unrealized_pnl"`
	AvailableToTransferFiat   float64 `json:"available_to_transfer_fiat"`
	AvailableToTransferCrypto float64 `json:"available_to_transfer_crypto"`
	AssetColour               string  `json:"asset_color"`
	// AccountType is one of the PortfolioAccountType values
	AccountType           string  `json:"account_type"`
	FundingPNL            float64 `json:"funding_pnl"`
	AvailableToSendFiat   float64 `json:"available_to_send_fiat"`
	AvailableToSendCrypto float64 `json:"available_to_send_crypto"`
}

// PortfolioPerpetualPosition is a portfolio's INTX perpetual position
type PortfolioPerpetualPosition struct {
	ProductID     string `json:"product_id"`
	ProductUUID   string `json:"product_uuid"`
	Symbol        string `json:"symbol"`
	AssetImageURL string `json:"asset_image_url"`
	// VWAP is the position's volume weighted average price
	VWAP PortfolioBalancePair `json:"vwap"`
	// PositionSide is one of the PortfolioPositionSide values
	PositionSide              string               `json:"position_side"`
	NetSize                   types.Number         `json:"net_size"`
	BuyOrderSize              types.Number         `json:"buy_order_size"`
	SellOrderSize             types.Number         `json:"sell_order_size"`
	InitialMarginContribution types.Number         `json:"im_contribution"`
	UnrealisedPNL             PortfolioBalancePair `json:"unrealized_pnl"`
	MarkPrice                 PortfolioBalancePair `json:"mark_price"`
	LiquidationPrice          PortfolioBalancePair `json:"liquidation_price"`
	Leverage                  types.Number         `json:"leverage"`
	InitialMarginNotional     PortfolioBalancePair `json:"im_notional"`
	MaintenanceMarginNotional PortfolioBalancePair `json:"mm_notional"`
	PositionNotional          PortfolioBalancePair `json:"position_notional"`
	// MarginType is one of the PortfolioMarginType values
	MarginType            string       `json:"margin_type"`
	LiquidationBuffer     types.Number `json:"liquidation_buffer"`
	LiquidationPercentage types.Number `json:"liquidation_percentage"`
	AssetColour           string       `json:"asset_color"`
}

// PortfolioBalancePair is a value in the user's native currency and in the position's own currency
type PortfolioBalancePair struct {
	UserNativeCurrency Amount `json:"userNativeCurrency"`
	RawCurrency        Amount `json:"rawCurrency"`
}

// PortfolioFuturesPosition is a portfolio's US derivatives position
type PortfolioFuturesPosition struct {
	ProductID    string       `json:"product_id"`
	ContractSize types.Number `json:"contract_size"`
	// Side is one of the PortfolioPositionSide values
	Side string `json:"side"`
	// Amount is the position's size in contracts
	Amount            types.Number  `json:"amount"`
	AverageEntryPrice types.Number  `json:"avg_entry_price"`
	CurrentPrice      types.Number  `json:"current_price"`
	UnrealisedPNL     types.Number  `json:"unrealized_pnl"`
	Expiry            time.Time     `json:"expiry"`
	UnderlyingAsset   currency.Code `json:"underlying_asset"`
	AssetImageURL     string        `json:"asset_img_url"`
	ProductName       string        `json:"product_name"`
	Venue             string        `json:"venue"`
	NotionalValue     types.Number  `json:"notional_value"`
	AssetColour       string        `json:"asset_color"`
	LastTradedAt      time.Time     `json:"last_traded_at"`
	RollDate          time.Time     `json:"roll_date"`
	PriceIncrement    types.Number  `json:"price_increment"`
	// PricePrecision is how many decimal places the price takes
	PricePrecision uint64 `json:"price_precision"`
}

// PortfolioUnifiedPosition is a portfolio's position in a product of any type; only prediction market positions are
// reported this way
type PortfolioUnifiedPosition struct {
	// ProductType is one of the PortfolioProductType values
	ProductType string `json:"product_type"`
	// CBRN identifies the product as a Coinbase resource name
	CBRN              string                             `json:"cbrn"`
	PredictionMarkets *PortfolioPredictionMarketPosition `json:"prediction_markets"`
}

// PortfolioPredictionMarketPosition is a portfolio's prediction market position
type PortfolioPredictionMarketPosition struct {
	// Side is one of the PortfolioPredictionMarketSide values
	Side              string       `json:"side"`
	LastTradedAt      time.Time    `json:"last_traded_at"`
	AverageEntryPrice Amount       `json:"average_entry_price"`
	CurrentPrice      Amount       `json:"current_price"`
	UnrealisedPNL     Amount       `json:"unrealized_pnl"`
	NotionalValue     Amount       `json:"notional_value"`
	CostBasis         Amount       `json:"cost_basis"`
	ContractsOwned    types.Number `json:"contracts_owned"`
	QuoteCBRN         string       `json:"quote_cbrn"`
}

// PortfolioEquityPosition is a portfolio's holding of a stock or ETF
type PortfolioEquityPosition struct {
	// CBRN identifies the equity as a Coinbase resource name
	CBRN                   string `json:"cbrn"`
	AccountUUID            string `json:"account_uuid"`
	TotalBalanceFiat       Amount `json:"total_balance_fiat"`
	TotalBalanceEquity     Amount `json:"total_balance_equity"`
	AvailableToTradeEquity Amount `json:"available_to_trade_equity"`
	AvailableToTradeFiat   Amount `json:"available_to_trade_fiat"`
	// AccountType is one of the PortfolioAccountType values
	AccountType string `json:"account_type"`
	// Allocation is the position's share of the portfolio
	Allocation        float64 `json:"allocation"`
	CostBasis         Amount  `json:"cost_basis"`
	AverageEntryPrice Amount  `json:"average_entry_price"`
	UnrealisedPNL     Amount  `json:"unrealized_pnl"`
}

// MovePortfolioFundsRequest moves an amount of a currency from one portfolio to another
type MovePortfolioFundsRequest struct {
	SourcePortfolioUUID string
	TargetPortfolioUUID string
	Currency            currency.Code
	Amount              float64
}

// MovePortfolioFundsWireRequest is MovePortfolioFunds' body
type MovePortfolioFundsWireRequest struct {
	Funds               PortfolioFunds `json:"funds"`
	SourcePortfolioUUID string         `json:"source_portfolio_uuid"`
	TargetPortfolioUUID string         `json:"target_portfolio_uuid"`
}

// PortfolioFunds is an amount of a currency moved between portfolios
type PortfolioFunds struct {
	Value string `json:"value"`
	// Currency stays a string, the code as Advanced Trade takes it
	Currency string `json:"currency"`
}

// MovePortfolioFundsResponse names the portfolios funds moved between
type MovePortfolioFundsResponse struct {
	SourcePortfolioUUID string `json:"source_portfolio_uuid"`
	TargetPortfolioUUID string `json:"target_portfolio_uuid"`
}
