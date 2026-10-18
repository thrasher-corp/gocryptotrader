package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Web3TradeQuoteRequest holds the parameters for Get Trade Quote
type Web3TradeQuoteRequest struct {
	// TradeType is 1 for a purchase (payment token to on-chain token) or 2 for a redeem (on-chain token to payment
	// token)
	TradeType uint64
	// FromTokenCode is a CEX_ payment token code for a purchase, or a DEX_ on-chain token code for a redeem
	FromTokenCode   string
	FromTokenAmount float64
	ToTokenCode     string
	// QuoteMode is 0 for auto (the default), 1 for price priority or 2 for success rate priority
	QuoteMode uint64
}

// Web3TradeQuoteResponse is a quote for an on-chain purchase or redeem trade. QuoteData, CorrectingCode and Gas must
// be passed unchanged to Execute Purchase or Execute Redeem before ExpireTime
type Web3TradeQuoteResponse struct {
	TradeType          uint64       `json:"tradeType"`
	FromTokenCode      string       `json:"fromTokenCode"`
	FromTokenAmount    types.Number `json:"fromTokenAmount"`
	FromTokenAmountUSD types.Number `json:"fromTokenAmountUsd"`
	ToTokenCode        string       `json:"toTokenCode"`
	ToTokenAmount      types.Number `json:"toTokenAmount"`
	ToTokenAmountUSD   types.Number `json:"toTokenAmountUsd"`
	MinToTokenAmount   types.Number `json:"minToTokenAmount"`
	// Slippage is a decimal fraction: 0.005 is 0.5%
	Slippage types.Number `json:"slippage"`
	// Gas is the estimated gas fee in the chain's native token. It stays the string Bybit sent, since Execute Purchase
	// and Execute Redeem require it unchanged
	Gas             string                    `json:"gas"`
	GasUSD          types.Number              `json:"gasUsd"`
	PlatformFee     types.Number              `json:"platformFee"`
	PlatformFeeUSD  types.Number              `json:"platformFeeUsd"`
	SwapRate        types.Number              `json:"swapRate"`
	LossRate        types.Number              `json:"lossRate"`
	QuoteData       string                    `json:"quoteData"`
	CorrectingCode  string                    `json:"correctingCode"`
	QuoteMode       uint64                    `json:"quoteMode"`
	QuoteDataID     string                    `json:"quoteDataId"`
	ExpireTime      types.Time                `json:"expireTime"`
	Timestamp       types.Time                `json:"timestamp"`
	ChargeAmount    types.Number              `json:"chargeAmount"`
	ModeEstimations []Web3QuoteModeEstimation `json:"modeEstimations"`
}

// Web3QuoteModeEstimation is the estimate a quote would have under another quote mode
type Web3QuoteModeEstimation struct {
	QuoteMode         uint64       `json:"quoteMode"`
	EstimatedGas      types.Number `json:"estimatedGas"`
	EstimatedGasUSD   types.Number `json:"estimatedGasUsd"`
	EstimatedSlippage types.Number `json:"estimatedSlippage"`
}

// Web3TradeRequest holds the parameters for Execute Purchase and Execute Redeem. FromTokenCode, FromTokenAmount,
// ToTokenCode and QuoteMode must match the Get Trade Quote request, and QuoteData, Gas and CorrectingCode its response
type Web3TradeRequest struct {
	FromTokenCode   string
	FromTokenAmount float64
	ToTokenCode     string
	// Slippage is the tolerated slippage as a decimal fraction: 0.01 is 1%
	Slippage  float64
	QuoteData string
	// Gas is the quote's Web3TradeQuoteResponse.Gas, unchanged
	Gas            string
	QuoteMode      uint64
	CorrectingCode string
	Tenant         string
}

// Web3OrderResponse holds the order number Bybit assigns to an on-chain trade, LP redemption or prediction order
type Web3OrderResponse struct {
	OrderNumber string `json:"orderNo"`
}

// Web3PaymentToken is a payment token usable for on-chain trading
type Web3PaymentToken struct {
	// TokenCode is the token's CEX_ code, used in quote and execution requests
	TokenCode         string        `json:"tokenCode"`
	Symbol            currency.Code `json:"symbol"`
	TokenDecimals     uint64        `json:"tokenDecimals"`
	TokenIconURLDay   string        `json:"tokenIconUrlDay"`
	TokenIconURLNight string        `json:"tokenIconUrlNight"`
	// Limit is the maximum trading amount for the payment token
	Limit         types.Number `json:"limit"`
	SupportChains []string     `json:"supportChains"`
}

// Web3OrderListRequest holds the parameters for Get Order List (Web3 trade)
type Web3OrderListRequest struct {
	// TradeType is 1 for purchases or 2 for redeems; 0 returns both
	TradeType uint64
	TokenCode string
	// OrderStatus filters by status: 1 processing, 2 success, 3 failed
	OrderStatus []uint64
	// Days is the number of days to look back, up to 90; 0 uses the 90 day default
	Days uint64
	// Limit is the number of results per page, from 1 to 100
	Limit uint64
	// PageIndex is the 1-based page number
	PageIndex uint64
	// Direction is the pagination direction: prev or next
	Direction string
}

// Web3OrderListResponse is a page of on-chain trade orders
type Web3OrderListResponse struct {
	Total     uint64      `json:"total"`
	PageIndex uint64      `json:"pageIndex"`
	Orders    []Web3Order `json:"orders"`
}

// Web3Order is an on-chain purchase or redeem order
type Web3Order struct {
	// OrderType is 1 for a market order or 2 for a limit order
	OrderType uint64 `json:"orderType"`
	// TradeType is 1 for a purchase or 2 for a redeem
	TradeType   uint64 `json:"tradeType"`
	OrderNumber string `json:"orderNo"`
	// OrderStatus is 1 processing, 2 success or 3 failed
	OrderStatus           uint64        `json:"orderStatus"`
	FromTokenCode         string        `json:"fromTokenCode"`
	FromTokenAmount       types.Number  `json:"fromTokenAmount"`
	FromTokenSymbol       currency.Code `json:"fromTokenSymbol"`
	FromTokenDecimals     uint64        `json:"fromTokenDecimals"`
	FromTokenIconURLDay   string        `json:"fromTokenIconUrlDay"`
	FromTokenIconURLNight string        `json:"fromTokenIconUrlNight"`
	FromChainCode         string        `json:"fromChainCode"`
	FromChainIconURL      string        `json:"fromChainIconUrl"`
	ToTokenCode           string        `json:"toTokenCode"`
	// ToTokenAmount is the amount received, set once the order completes
	ToTokenAmount       types.Number  `json:"toTokenAmount"`
	ToTokenSymbol       currency.Code `json:"toTokenSymbol"`
	ToTokenDecimals     uint64        `json:"toTokenDecimals"`
	ToTokenIconURLDay   string        `json:"toTokenIconUrlDay"`
	ToTokenIconURLNight string        `json:"toTokenIconUrlNight"`
	ToChainCode         string        `json:"toChainCode"`
	ToChainIconURL      string        `json:"toChainIconUrl"`
	GasTokenSymbol      currency.Code `json:"gasTokenSymbol"`
	GasOnchain          types.Number  `json:"gasOnchain"`
	// GasUSD and PlatformFeeUSD are null while the order is processing
	GasUSD         types.Number `json:"gasUsd"`
	PlatformFee    types.Number `json:"platformFee"`
	PlatformFeeUSD types.Number `json:"platformFeeUsd"`
	QuoteMode      uint64       `json:"quoteMode"`
	CreateTime     types.Time   `json:"createTime"`
	ExecutionTime  types.Time   `json:"executionTime"`
	// FailureReasonCode is set when OrderStatus is 3, for example ERR103 for an insufficient balance
	FailureReasonCode     string       `json:"failureReasonCode"`
	Source                string       `json:"source"`
	SwapRate              types.Number `json:"swapRate"`
	ActualFromTokenAmount types.Number `json:"actualFromTokenAmount"`
}

// Web3BizToken is an on-chain token available for trading
type Web3BizToken struct {
	// TokenCode is the token's DEX_ code, used in quote and execution requests
	TokenCode         string        `json:"tokenCode"`
	ChainCode         string        `json:"chainCode"`
	ChainIconURL      string        `json:"chainIconUrl"`
	TokenAddress      string        `json:"tokenAddress"`
	Symbol            currency.Code `json:"symbol"`
	TokenDecimals     uint64        `json:"tokenDecimals"`
	TokenIconURLDay   string        `json:"tokenIconUrlDay"`
	TokenIconURLNight string        `json:"tokenIconUrlNight"`
	CreateTime        types.Time    `json:"createTime"`
	CreateTimeOnchain types.Time    `json:"createTimeOnchain"`
	// RiskFlag is set when Bybit identified a risk; warn the user before trading
	RiskFlag types.Boolean `json:"riskFlag"`
	// MinOrderQuantity and MaxOrderQuantity are in payment token units. They are documented as JSON numbers here and
	// as strings by Get Token Details, so either is accepted
	MinOrderQuantity types.Number `json:"minOrderQuantity"`
	MaxOrderQuantity types.Number `json:"maxOrderQuantity"`
	TokenTags        []uint64     `json:"tokenTags"`
	PayTokenCodes    []string     `json:"payTokenCodes"`
}

// Web3TokenAddress identifies an on-chain token by its chain and contract address
type Web3TokenAddress struct {
	ChainCode    string
	TokenAddress string
}

// Web3TokenPriceListResponse holds on-chain token market data
type Web3TokenPriceListResponse struct {
	TokenPriceInfoList []Web3TokenPriceInfo `json:"tokenPriceInfoList"`
}

// Web3TokenPriceInfo is an on-chain token's market data in USD
type Web3TokenPriceInfo struct {
	ChainCode    string       `json:"chainCode"`
	TokenAddress string       `json:"tokenAddress"`
	Price        types.Number `json:"price"`
	// Change24Hour is a decimal fraction: 0.15 is +15%
	Change24Hour types.Number `json:"change24h"`
	Volume24Hour types.Number `json:"vol24h"`
	MarketCap    types.Number `json:"marketCap"`
	Liquidity    types.Number `json:"liquidity"`
	Holders      types.Number `json:"holders"`
}

// Web3TokenDetailsResponse holds an on-chain token's project details
type Web3TokenDetailsResponse struct {
	TokenCode           string        `json:"tokenCode"`
	ChainCode           string        `json:"chainCode"`
	ChainIconURL        string        `json:"chainIconUrl"`
	TokenAddress        string        `json:"tokenAddress"`
	Symbol              currency.Code `json:"symbol"`
	TokenDecimals       uint64        `json:"tokenDecimals"`
	TokenIconURLDay     string        `json:"tokenIconUrlDay"`
	TokenIconURLNight   string        `json:"tokenIconUrlNight"`
	MinOrderQuantity    types.Number  `json:"minOrderQuantity"`
	MaxOrderQuantity    types.Number  `json:"maxOrderQuantity"`
	MaxPositionQuantity types.Number  `json:"maxPositionQuantity"`
	TokenDescription    string        `json:"tokenDesc"`
	XURL                string        `json:"xUrl"`
	OfficialURL         string        `json:"officialUrl"`
	WhitePaperURL       string        `json:"whitePaperUrl"`
	// TokenTag is 0 for no tag, 1 for new token sniping or 2 for on-chain hot token
	TokenTag          uint64        `json:"tokenTag"`
	RiskFlag          types.Boolean `json:"riskFlag"`
	CreateTimeOnchain types.Time    `json:"createTimeOnchain"`
	// Status is 0 not listed, 1 listed, 2 delisting, 3 in delivery or 4 delisted
	Status    uint64   `json:"status"`
	TokenTags []uint64 `json:"tokenTags"`
	// ShowMessage asks for Content, and the link when LinkName and LinkAddress are set, to be shown to the user
	ShowMessage types.Boolean `json:"showMessage"`
	Content     string        `json:"content"`
	LinkName    string        `json:"linkName"`
	LinkAddress string        `json:"linkAddress"`
}

// Web3AssetListResponse holds the on-chain token portfolio
type Web3AssetListResponse struct {
	TotalAssetUSD types.Number `json:"totalAssetUsd"`
	AssetList     []Web3Asset  `json:"assetList"`
}

// Web3Asset is an on-chain token holding
type Web3Asset struct {
	ChainCode         string        `json:"chainCode"`
	ChainIconURL      string        `json:"chainIconUrl"`
	TokenAddress      string        `json:"tokenAddress"`
	TokenCode         string        `json:"tokenCode"`
	TokenSymbol       currency.Code `json:"tokenSymbol"`
	TokenDecimals     uint64        `json:"tokenDecimals"`
	TokenIconURLDay   string        `json:"tokenIconUrlDay"`
	TokenIconURLNight string        `json:"tokenIconUrlNight"`
	TokenAmount       types.Number  `json:"tokenAmount"`
	TokenAmountUSD    types.Number  `json:"tokenAmountUsd"`
	// TradeFlag is set when the token can be sold with Execute Redeem
	TradeFlag types.Boolean `json:"tradeFlag"`
	// PNL, PNLRatio, CostPrice and CostTotalValue are null when there is no cost basis
	PNL            types.Number `json:"pnl"`
	PNLRatio       types.Number `json:"pnlRatio"`
	CostPrice      types.Number `json:"costPrice"`
	LastPrice      types.Number `json:"lastPrice"`
	CostTotalValue types.Number `json:"costTotalValue"`
	// AssetStatus is 0 running, 1 delisting soon or 2 delisted
	AssetStatus          uint64     `json:"assetStatus"`
	AnnouncementURL      string     `json:"announcementUrl"`
	EstimatedOfflineTime types.Time `json:"estimatedOfflineTime"`
	DelistingTime        types.Time `json:"delistingTime"`
}

// Web3AssetDetailResponse holds a single on-chain token holding; AssetList is empty when the token is not held
type Web3AssetDetailResponse struct {
	AssetList []Web3Asset `json:"assetList"`
}

// Web3LPStakeRequest holds the parameters for Execute LP Stake. The upper and lower bounds take either a range or a
// price, not both
type Web3LPStakeRequest struct {
	// PositionID adds liquidity to an existing position; 0 creates a new position
	PositionID     uint64
	PoolAddress    string
	PayTokenAmount float64
	PayTokenCode   string
	RangeUpper     float64
	RangeLower     float64
	PriceUpper     float64
	PriceLower     float64
}

// Web3LPStakeResponse holds the position and order an LP stake created
type Web3LPStakeResponse struct {
	PositionID  uint64 `json:"positionId"`
	OrderNumber string `json:"orderNo"`
}

// Web3LPRedeemRequest holds the parameters for Execute LP Redeem
type Web3LPRedeemRequest struct {
	PositionID  uint64
	PoolAddress string
	// DercRatio is the share of the position to redeem, from 0 to 1: 0.25 redeems 25% and 1 closes the position
	DercRatio        float64
	ReceiveTokenCode string
}

// Web3LPOrderListRequest holds the parameters for Get LP Order List
type Web3LPOrderListRequest struct {
	// OrderType is 1 for stakes or 2 for redeems; 0 returns both
	OrderType uint64
	TokenCode string
	// OrderStatus filters by status: 1 processing, 2 success, 3 failed
	OrderStatus []uint64
	// Days is the number of days to look back, up to 365; 0 uses the 90 day default
	Days uint64
	// Limit is the number of results per page, up to 100; 0 uses the default of 20
	Limit uint64
	// PageIndex is the 1-based page number; 0 uses the first page
	PageIndex   uint64
	PoolAddress string
}

// Web3LPOrderListResponse is a page of LP stake and redeem orders
type Web3LPOrderListResponse struct {
	Total     uint64        `json:"total"`
	PageIndex uint64        `json:"pageIndex"`
	Orders    []Web3LPOrder `json:"orders"`
}

// Web3LPOrder is an LP stake or redeem order
type Web3LPOrder struct {
	// OrderType is 1 for a stake or 2 for a redeem
	OrderType   uint64 `json:"orderType"`
	OrderNumber string `json:"orderNo"`
	// OrderStatus is 1 processing, 2 success or 3 failed
	OrderStatus       uint64        `json:"orderStatus"`
	PoolAddress       string        `json:"poolAddress"`
	PoolName          string        `json:"poolName"`
	PositionID        uint64        `json:"positionId"`
	TokenCode         string        `json:"tokenCode"`
	TokenSymbol       currency.Code `json:"tokenSymbol"`
	TokenIconURLDay   string        `json:"tokenIconUrlDay"`
	TokenIconURLNight string        `json:"tokenIconUrlNight"`
	Amount            types.Number  `json:"amount"`
	ChainCode         string        `json:"chainCode"`
	ChainIconURL      string        `json:"chainIconUrl"`
	GasTokenSymbol    currency.Code `json:"gasTokenSymbol"`
	GasOnchain        types.Number  `json:"gasOnchain"`
	GasUSD            types.Number  `json:"gasUsd"`
	PlatformFee       types.Number  `json:"platformFee"`
	PlatformFeeUSD    types.Number  `json:"platformFeeUsd"`
	CreateTime        types.Time    `json:"createTime"`
	ExecutionTime     types.Time    `json:"executionTime"`
	// FailureReason is set when OrderStatus is 3
	FailureReason string `json:"failureReason"`
	// DercRatio is the share of the position a redeem order withdrew
	DercRatio types.Number `json:"dercRatio"`
}

// Web3LPPayTokenListResponse holds the payment tokens accepted for LP staking
type Web3LPPayTokenListResponse struct {
	Tokens []Web3LPPayToken `json:"tokens"`
}

// Web3LPPayToken is a payment token accepted for LP staking, with the user's available balance
type Web3LPPayToken struct {
	TokenCode         string        `json:"tokenCode"`
	TokenSymbol       currency.Code `json:"tokenSymbol"`
	ChainCode         string        `json:"chainCode"`
	ChainIconURL      string        `json:"chainIconUrl"`
	Decimals          uint64        `json:"decimals"`
	AvailableBalance  types.Number  `json:"availableBalance"`
	TokenIconURLDay   string        `json:"tokenIconUrlDay"`
	TokenIconURLNight string        `json:"tokenIconUrlNight"`
	MinStakeAmount    types.Number  `json:"minStakeAmount"`
	MaxStakeAmount    types.Number  `json:"maxStakeAmount"`
}

// Web3LPPayTokenPriceResponse holds payment token prices
type Web3LPPayTokenPriceResponse struct {
	Prices []Web3LPPayTokenPrice `json:"prices"`
}

// Web3LPPayTokenPrice is a payment token's price in USD
type Web3LPPayTokenPrice struct {
	TokenCode   string        `json:"tokenCode"`
	TokenSymbol currency.Code `json:"tokenSymbol"`
	PriceUSD    types.Number  `json:"priceUsd"`
	ChainCode   string        `json:"chainCode"`
	UpdateTime  types.Time    `json:"updateTime"`
}

// Web3LPPoolInfoResponse holds a liquidity pool's details
type Web3LPPoolInfoResponse struct {
	PoolAddress string       `json:"poolAddress"`
	PoolName    string       `json:"poolName"`
	APY         types.Number `json:"apy"`
	TVL         types.Number `json:"tvl"`
	// FeeRate is a decimal fraction: 0.003 is 0.3%
	FeeRate         types.Number  `json:"feeRate"`
	Token0Symbol    currency.Code `json:"token0Symbol"`
	Token0Reserve   types.Number  `json:"token0Reserve"`
	Token1Symbol    currency.Code `json:"token1Symbol"`
	Token1Reserve   types.Number  `json:"token1Reserve"`
	PriceRangeLower types.Number  `json:"priceRangeLower"`
	PriceRangeUpper types.Number  `json:"priceRangeUpper"`
	CurrentPrice    types.Number  `json:"currentPrice"`
}

// Web3LPPoolListResponse holds the available liquidity pools
type Web3LPPoolListResponse struct {
	Pools []Web3LPPool `json:"pools"`
}

// Web3LPPool is a liquidity pool
type Web3LPPool struct {
	PoolAddress string       `json:"poolAddress"`
	PoolName    string       `json:"poolName"`
	PoolTag     string       `json:"poolTag"`
	APY         types.Number `json:"apy"`
	// TVL is the total value locked in USD
	TVL                types.Number  `json:"tvl"`
	Token0Symbol       currency.Code `json:"token0Symbol"`
	Token0IconURLDay   string        `json:"token0IconUrlDay"`
	Token0IconURLNight string        `json:"token0IconUrlNight"`
	Token1Symbol       currency.Code `json:"token1Symbol"`
	Token1IconURLDay   string        `json:"token1IconUrlDay"`
	Token1IconURLNight string        `json:"token1IconUrlNight"`
	ChainCode          string        `json:"chainCode"`
	ChainIconURL       string        `json:"chainIconUrl"`
}

// Web3LPPositionListResponse holds the user's LP positions
type Web3LPPositionListResponse struct {
	Positions []Web3LPPosition `json:"positions"`
}

// Web3LPPosition is an LP position with its current valuation
type Web3LPPosition struct {
	PositionID        uint64        `json:"positionId"`
	PoolAddress       string        `json:"poolAddress"`
	PoolName          string        `json:"poolName"`
	StakedAmount      types.Number  `json:"stakedAmount"`
	StakedTokenCode   string        `json:"stakedTokenCode"`
	StakedTokenSymbol currency.Code `json:"stakedTokenSymbol"`
	CurrentValueUSD   types.Number  `json:"currentValueUsd"`
	EarnedRewardsUSD  types.Number  `json:"earnedRewardsUsd"`
	UnrealisedPNL     types.Number  `json:"unrealizedPnl"`
	RealisedPNL       types.Number  `json:"realizedPnl"`
	APY               types.Number  `json:"apy"`
	Token0Amount      types.Number  `json:"token0Amount"`
	Token0Symbol      currency.Code `json:"token0Symbol"`
	Token1Amount      types.Number  `json:"token1Amount"`
	Token1Symbol      currency.Code `json:"token1Symbol"`
	// RangeUpper and RangeLower hold the position's range or price bounds, whichever it was staked with
	RangeUpper types.Number `json:"rangeUpper"`
	RangeLower types.Number `json:"rangeLower"`
	CreateTime types.Time   `json:"createTime"`
	UpdateTime types.Time   `json:"updateTime"`
	// Status is 1 active, 2 closed or 3 processing
	Status uint64 `json:"status"`
}

// PredictionEngineStatusResponse reports whether the prediction market matching engine accepts orders
type PredictionEngineStatusResponse struct {
	Available bool `json:"available"`
}

// PredictionPaymentToken is a payment token accepted by the prediction market
type PredictionPaymentToken struct {
	// TokenCode is the code buy requests pass as the payment token, such as USDC
	TokenCode         string        `json:"tokenCode"`
	Symbol            currency.Code `json:"symbol"`
	TokenDecimals     uint64        `json:"tokenDecimals"`
	TokenIconURLDay   string        `json:"tokenIconUrlDay"`
	TokenIconURLNight string        `json:"tokenIconUrlNight"`
	SupportChains     []string      `json:"supportChains"`
}

// PredictionEventDetailRequest holds the parameters for Get Event Detail. Slug takes priority over EventID
type PredictionEventDetailRequest struct {
	EventID string
	Slug    string
	// HasMoreMarkets includes the markets of related sub-events
	HasMoreMarkets bool
}

// PredictionEventDetailResponse holds a prediction market event and its outcome markets
type PredictionEventDetailResponse struct {
	EventID     string                  `json:"eventId"`
	Slug        string                  `json:"slug"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Category    string                  `json:"category"`
	EndDate     types.Time              `json:"endDate"`
	Resolved    bool                    `json:"resolved"`
	Volume      types.Number            `json:"volume"`
	Liquidity   types.Number            `json:"liquidity"`
	Markets     []PredictionEventMarket `json:"markets"`
}

// PredictionEventMarket is an outcome market within a prediction event
type PredictionEventMarket struct {
	TokenID string `json:"tokenId"`
	Outcome string `json:"outcome"`
	// Price is the outcome's implied probability, from 0 to 1
	Price types.Number `json:"price"`
	// Side is 1 for YES or 2 for NO
	Side      uint64       `json:"side"`
	Volume    types.Number `json:"volume"`
	Liquidity types.Number `json:"liquidity"`
}

// PredictionOrderEstimateRequest holds the parameters for Get Order Estimate
type PredictionOrderEstimateRequest struct {
	TokenID string
	// Side is 1 to buy or 2 to sell
	Side    uint64
	EventID string
	// Amount is the USDC to invest when buying, or the number of shares when selling
	Amount float64
	// OrderType is 1 for fill-or-kill, the only order type phase 1 supports
	OrderType uint64
	// PayTokenCode is required when buying
	PayTokenCode string
}

// PredictionOrderEstimateResponse holds the estimated outcome of a prediction market order
type PredictionOrderEstimateResponse struct {
	AveragePrice types.Number `json:"avgPrice"`
	// EstimatedCost is the USDC to pay when buying, or the shares to sell when selling
	EstimatedCost types.Number `json:"estimatedCost"`
	// EstimatedReceive is the shares received when buying, or the USDC received when selling
	EstimatedReceive types.Number `json:"estimatedReceive"`
	// The payout if the outcome is correct, for buy orders only
	ToWin     types.Number        `json:"toWin"`
	FeeAmount types.Number        `json:"feeAmount"`
	FeeDetail PredictionFeeDetail `json:"feeDetail"`
	Slippage  types.Number        `json:"slippage"`
}

// PredictionFeeDetail breaks a prediction order's fee down by recipient, in USDC
type PredictionFeeDetail struct {
	ServerFee     types.Number `json:"serverFee"`
	PolymarketFee types.Number `json:"polymarketFee"`
}

// PredictionBuyRequest holds the parameters for Execute Buy (prediction market)
type PredictionBuyRequest struct {
	TokenID string
	// Amount is the USDC to invest
	Amount       float64
	PayTokenCode string
	// OrderType is 1 for fill-or-kill, the only order type phase 1 supports
	OrderType uint64
	// Slippage is the maximum tolerated price slippage as a decimal fraction: 0.05 is 5%
	Slippage float64
	EventID  string
}

// PredictionSellRequest holds the parameters for Execute Sell (prediction market)
type PredictionSellRequest struct {
	TokenID string
	// Size is the number of shares to sell
	Size float64
	// OrderType is 1 for fill-or-kill, the only order type phase 1 supports
	OrderType uint64
	// Slippage is the maximum tolerated price slippage as a decimal fraction: 0.05 is 5%
	Slippage    float64
	EventID     string
	ToTokenCode string
}

// PredictionOrderListRequest holds the parameters for Get Order List (prediction market)
type PredictionOrderListRequest struct {
	// Status is 1 pending, 2 filled, 3 partially filled, 4 cancelled or 5 rejected
	Status  uint64
	TokenID string
	EventID string
	// Side is 1 for buy orders or 2 for sell orders
	Side uint64
	// Days is the number of days to look back, up to 90
	Days      uint64
	Limit     uint64
	PageIndex uint64
}

// PredictionOrderListResponse is a page of prediction market orders
type PredictionOrderListResponse struct {
	Orders    []PredictionOrder `json:"orders"`
	Total     uint64            `json:"total"`
	PageIndex uint64            `json:"pageIndex"`
}

// PredictionOrder is a prediction market order
type PredictionOrder struct {
	OrderNumber string `json:"orderNo"`
	TokenID     string `json:"tokenId"`
	EventID     string `json:"eventId"`
	// Side is 1 for a buy or 2 for a sell
	Side      uint64 `json:"side"`
	OrderType uint64 `json:"orderType"`
	// OrderAmount is the USDC invested by a buy, or the shares sold by a sell
	OrderAmount types.Number `json:"orderAmount"`
	FilledSize  types.Number `json:"filledSize"`
	FilledPrice types.Number `json:"filledPrice"`
	TotalFee    types.Number `json:"totalFee"`
	// Status is 1 pending, 2 filled, 3 partially filled, 4 cancelled or 5 rejected
	Status    uint64     `json:"status"`
	CreatedAt types.Time `json:"createdAt"`
	UpdatedAt types.Time `json:"updatedAt"`
}

// PredictionOrderBook is an outcome token's order book; bids are sorted by price descending and asks ascending
type PredictionOrderBook struct {
	TokenID string                     `json:"tokenId"`
	Bids    []PredictionOrderBookLevel `json:"bids"`
	Asks    []PredictionOrderBookLevel `json:"asks"`
}

// PredictionOrderBookLevel is a price level and the shares available at it
type PredictionOrderBookLevel struct {
	Price types.Number `json:"price"`
	Size  types.Number `json:"size"`
}

// PredictionTokenPrice is an outcome token's best prices, on a 0 to 1 probability scale
type PredictionTokenPrice struct {
	TokenID   string       `json:"tokenId"`
	BestBid   types.Number `json:"bestBid"`
	BestAsk   types.Number `json:"bestAsk"`
	LastPrice types.Number `json:"lastPrice"`
	// Volume24Hour is in USDC
	Volume24Hour types.Number `json:"volume24h"`
	UpdateTime   types.Time   `json:"updateTime"`
}

// PredictionPriceHistoryResponse holds an outcome token's candles, newest first
type PredictionPriceHistoryResponse struct {
	TokenID  string             `json:"tokenId"`
	Interval string             `json:"interval"`
	List     []PredictionCandle `json:"list"`
}

// PredictionCandle is an outcome token's OHLC candle; Volume is in USDC
type PredictionCandle struct {
	OpenTime types.Time   `json:"openTime"`
	Open     types.Number `json:"open"`
	High     types.Number `json:"high"`
	Low      types.Number `json:"low"`
	Close    types.Number `json:"close"`
	Volume   types.Number `json:"volume"`
}

// PredictionPositionListRequest holds the parameters for Get Position List (prediction market)
type PredictionPositionListRequest struct {
	TokenID   string
	EventID   string
	Limit     uint64
	PageIndex uint64
}

// PredictionPositionListResponse holds the user's open prediction market positions
type PredictionPositionListResponse struct {
	Positions     []PredictionPosition `json:"positions"`
	Total         uint64               `json:"total"`
	TotalAssetUSD types.Number         `json:"totalAssetUsd"`
}

// PredictionPosition is an open prediction market position; amounts are in USDC
type PredictionPosition struct {
	PositionID   string       `json:"positionId"`
	TokenID      string       `json:"tokenId"`
	EventID      string       `json:"eventId"`
	OutcomeName  string       `json:"outcomeName"`
	Shares       types.Number `json:"shares"`
	Cost         types.Number `json:"cost"`
	AveragePrice types.Number `json:"avgPrice"`
	CurrentPrice types.Number `json:"currentPrice"`
	Value        types.Number `json:"value"`
	// UnrealisedPNLRate is a decimal fraction: 0.107 is 10.7%
	UnrealisedPNL     types.Number `json:"unrealizedPnl"`
	UnrealisedPNLRate types.Number `json:"unrealizedPnlRate"`
	CreatedAt         types.Time   `json:"createdAt"`
	// Finished is set once the position's event has ended
	Finished bool `json:"finished"`
}

// PredictionPositionHistoryRequest holds the parameters for Get Position History (prediction market)
type PredictionPositionHistoryRequest struct {
	TokenID string
	EventID string
	// Result is 1 for won, 2 for lost or 3 for positions closed before resolution
	Result uint64
	// Days is the number of days to look back, up to 90
	Days      uint64
	Limit     uint64
	PageIndex uint64
}

// PredictionPositionHistoryResponse holds the user's closed prediction market positions
type PredictionPositionHistoryResponse struct {
	Positions []PredictionClosedPosition `json:"positions"`
	Total     uint64                     `json:"total"`
}

// PredictionClosedPosition is a closed prediction market position; amounts are in USDC
type PredictionClosedPosition struct {
	PositionID      string       `json:"positionId"`
	TokenID         string       `json:"tokenId"`
	EventID         string       `json:"eventId"`
	OutcomeName     string       `json:"outcomeName"`
	Shares          types.Number `json:"shares"`
	Cost            types.Number `json:"cost"`
	AveragePrice    types.Number `json:"avgPrice"`
	ExitPrice       types.Number `json:"exitPrice"`
	RealisedPNL     types.Number `json:"realizedPnl"`
	RealisedPNLRate types.Number `json:"realizedPnlRate"`
	// Result is 1 for won, 2 for lost or 3 for closed before resolution
	Result   uint64     `json:"result"`
	ClosedAt types.Time `json:"closedAt"`
}

// PredictionPortfolioSummaryResponse summarises the user's prediction market portfolio
type PredictionPortfolioSummaryResponse struct {
	// PositionValue is the open positions' market value in USDC
	PositionValue    types.Number `json:"positionValue"`
	PositionValueUSD types.Number `json:"positionValueUsd"`
	BiggestWin       types.Number `json:"biggestWin"`
	// WinRate is a decimal fraction: 0.67 is 67%
	WinRate   types.Number `json:"winRate"`
	WinCount  uint64       `json:"winCount"`
	LossCount uint64       `json:"lossCount"`
}

// PredictionSideMarketListRequest holds the parameters for Get Side Market List
type PredictionSideMarketListRequest struct {
	EventID string
	// SortBy is volume, liquidity or endDate
	SortBy string
	// MarketType is 1 for YES/NO markets or 2 for multi-outcome markets
	MarketType uint64
	Limit      uint64
	PageIndex  uint64
}

// PredictionSideMarketListResponse is a page of an event's side markets
type PredictionSideMarketListResponse struct {
	List      []PredictionSideMarket `json:"list"`
	Total     uint64                 `json:"total"`
	PageIndex uint64                 `json:"pageIndex"`
	PageSize  uint64                 `json:"pageSize"`
}

// PredictionSideMarket is a market within a prediction event
type PredictionSideMarket struct {
	MarketID string `json:"marketId"`
	EventID  string `json:"eventId"`
	// MarketType is 1 for a YES/NO market or 2 for a multi-outcome market
	MarketType uint64                      `json:"marketType"`
	Question   string                      `json:"question"`
	Tokens     []PredictionSideMarketToken `json:"tokens"`
	// Volume24Hour is in USDC
	Volume24Hour types.Number `json:"volume24h"`
	EndDate      types.Time   `json:"endDate"`
	// Status is 1 open, 2 closed or 3 resolved
	Status uint64 `json:"status"`
}

// PredictionSideMarketToken is an outcome token of a side market
type PredictionSideMarketToken struct {
	TokenID string `json:"tokenId"`
	Outcome string `json:"outcome"`
	// Price is the outcome's implied probability, from 0 to 1
	Price types.Number `json:"price"`
	// Liquidity is in USDC
	Liquidity types.Number `json:"liquidity"`
}

// PredictionSportsMatchListRequest holds the parameters for Get Sports Match List
type PredictionSportsMatchListRequest struct {
	// EventType is 1 for the 2026 FIFA World Cup, the only event type phase 1 supports
	EventType uint64
	// Status is 1 live, 2 upcoming or 3 ended
	Status    uint64
	StartTime time.Time
	EndTime   time.Time
	// StageCode is Groups, R32, R16, QF, SF or Final
	StageCode string
	Limit     uint64
	PageIndex uint64
}

// PredictionSportsMatchListResponse is a page of sports matches
type PredictionSportsMatchListResponse struct {
	List      []PredictionSportsMatch `json:"list"`
	Total     uint64                  `json:"total"`
	PageIndex uint64                  `json:"pageIndex"`
	PageSize  uint64                  `json:"pageSize"`
}

// PredictionSportsMatch is a sports match in the prediction market
type PredictionSportsMatch struct {
	MatchID   string     `json:"matchId"`
	EventType uint64     `json:"eventType"`
	StageCode string     `json:"stageCode"`
	HomeTeam  string     `json:"homeTeam"`
	AwayTeam  string     `json:"awayTeam"`
	MatchTime types.Time `json:"matchTime"`
	// Status is 1 live, 2 upcoming or 3 ended
	Status uint64 `json:"status"`
	// EventID is the linked prediction event, when it has markets
	EventID string `json:"eventId"`
}

// PredictionSportsTimelineStagesResponse holds a sports event's tournament stages
type PredictionSportsTimelineStagesResponse struct {
	EventType uint64                  `json:"eventType"`
	Stages    []PredictionSportsStage `json:"stages"`
}

// PredictionSportsStage is a tournament stage
type PredictionSportsStage struct {
	StageCode string `json:"stageCode"`
	StageName string `json:"stageName"`
	// Status is 1 done, 2 active or 3 upcoming
	Status     uint64     `json:"status"`
	MatchCount uint64     `json:"matchCount"`
	StartTime  types.Time `json:"startTime"`
	EndTime    types.Time `json:"endTime"`
}

// PredictionSportsGroupStageDetailResponse holds a sports event's group standings and matches
type PredictionSportsGroupStageDetailResponse struct {
	Groups []PredictionSportsGroup `json:"groups"`
}

// PredictionSportsGroup is a tournament group with its standings, sorted by rank, and matches
type PredictionSportsGroup struct {
	GroupName string                       `json:"groupName"`
	Standings []PredictionSportsStanding   `json:"standings"`
	Matches   []PredictionSportsGroupMatch `json:"matches"`
}

// PredictionSportsStanding is a team's standing within a group
type PredictionSportsStanding struct {
	Rank         uint64 `json:"rank"`
	TeamName     string `json:"teamName"`
	Played       uint64 `json:"played"`
	Won          uint64 `json:"won"`
	Drawn        uint64 `json:"drawn"`
	Lost         uint64 `json:"lost"`
	GoalsFor     uint64 `json:"goalsFor"`
	GoalsAgainst uint64 `json:"goalsAgainst"`
	// GoalDifference is GoalsFor less GoalsAgainst, so it can be negative
	GoalDifference int64  `json:"goalDifference"`
	Points         uint64 `json:"points"`
}

// PredictionSportsGroupMatch is a match within a group
type PredictionSportsGroupMatch struct {
	MatchID   string     `json:"matchId"`
	HomeTeam  string     `json:"homeTeam"`
	AwayTeam  string     `json:"awayTeam"`
	MatchTime types.Time `json:"matchTime"`
	// Status is 1 live, 2 upcoming or 3 ended
	Status uint64 `json:"status"`
	// HomeScore and AwayScore are -1 before the match is played
	HomeScore int64  `json:"homeScore"`
	AwayScore int64  `json:"awayScore"`
	EventID   string `json:"eventId"`
}
