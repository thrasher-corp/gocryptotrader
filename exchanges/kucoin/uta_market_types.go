package kucoin

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTAAnnouncementsRequest holds the parameters for GetUTAAnnouncements
type UTAAnnouncementsRequest struct {
	// Language defaults to en_US
	Language string
	// Type defaults to latest-announcements
	Type       string
	PageNumber uint64
	PageSize   uint64
	// StartTime and EndTime bound the announcements' release times; KuCoin searches the past month when they are unset
	StartTime time.Time
	EndTime   time.Time
}

// UTAAnnouncementsResponse holds a page of announcements
type UTAAnnouncementsResponse struct {
	TotalNumber uint64            `json:"totalNumber"`
	TotalPage   uint64            `json:"totalPage"`
	PageNumber  uint64            `json:"pageNumber"`
	PageSize    uint64            `json:"pageSize"`
	List        []UTAAnnouncement `json:"list"`
}

// UTAAnnouncement holds an announcement
type UTAAnnouncement struct {
	ID          uint64     `json:"id"`
	Title       string     `json:"title"`
	Type        []string   `json:"type"`
	Description string     `json:"description"`
	ReleaseTime types.Time `json:"releaseTime"`
	Language    string     `json:"language"`
	URL         string     `json:"url"`
}

// UTACurrencyResponse holds a currency's configuration and the chains it moves on. Get Currencies returns one per
// currency
type UTACurrencyResponse struct {
	Currency  currency.Code `json:"currency"`
	Name      string        `json:"name"`
	FullName  string        `json:"fullName"`
	Precision uint64        `json:"precision"`
	// Confirmations and ContractAddress are undocumented and null at currency level; each chain carries its own
	Confirmations   uint64 `json:"confirms"`
	ContractAddress string `json:"contractAddress"`
	// IsMarginEnabled and IsDebitEnabled are undocumented; the docs' changelog says they were removed, but KuCoin still
	// sends them
	IsMarginEnabled bool               `json:"isMarginEnabled"`
	IsDebitEnabled  bool               `json:"isDebitEnabled"`
	List            []UTACurrencyChain `json:"list"`
}

// UTACurrencyChain holds a currency's configuration on one chain
type UTACurrencyChain struct {
	// Chain is the chain identifier other V2 asset endpoints take, such as trx; ChainName is for display, such as TRC20
	Chain     string `json:"chain"`
	ChainName string `json:"chainName"`
	// Confirmations is the number of confirmations a deposit needs; PreConfirmations is the number for preliminary
	// recognition
	Confirmations     uint64 `json:"confirms"`
	PreConfirmations  uint64 `json:"preConfirms"`
	ContractAddress   string `json:"contractAddress"`
	IsDepositEnabled  bool   `json:"isDepositEnabled"`
	IsWithdrawEnabled bool   `json:"isWithdrawEnabled"`
	IsMemoRequired    bool   `json:"isMemoRequired"`
	// AddressRegex and MemoRegex validate withdrawal addresses and memos; empty means KuCoin gives none
	AddressRegex   string       `json:"addressRegex"`
	MemoRegex      string       `json:"memoRegex"`
	DepositFeeRate types.Number `json:"depositFeeRate"`
	// DepositTierFee is documented as a string but was null on every chain checked, so its format is unknown
	DepositTierFee    string       `json:"depositTierFee"`
	FixedDepositFee   types.Number `json:"fixedDepositFee"`
	MaxDepositFee     types.Number `json:"maxDepositFee"`
	MinDepositSize    types.Number `json:"minDepositSize"`
	MaxDepositSize    types.Number `json:"maxDepositSize"`
	WithdrawFeeRate   types.Number `json:"withdrawFeeRate"`
	WithdrawPrecision uint64       `json:"withdrawPrecision"`
	MinWithdrawSize   types.Number `json:"minWithdrawSize"`
	MaxWithdrawSize   types.Number `json:"maxWithdrawSize"`
	MinWithdrawFee    types.Number `json:"minWithdrawFee"`
	MaxWithdrawFee    types.Number `json:"maxWithdrawFee"`
}

// UTATickerResponse holds the 24 hour tickers of a product type
type UTATickerResponse struct {
	TradeType string      `json:"tradeType"`
	Timestamp types.Time  `json:"ts"`
	List      []UTATicker `json:"list"`
}

// UTATicker holds a symbol's 24 hour ticker
type UTATicker struct {
	Symbol string `json:"symbol"`
	// Name is the spot symbol's display name, which can differ from Symbol after KuCoin renames a currency
	Name         string       `json:"name"`
	BestBidPrice types.Number `json:"bestBidPrice"`
	BestBidSize  types.Number `json:"bestBidSize"`
	BestAskPrice types.Number `json:"bestAskPrice"`
	BestAskSize  types.Number `json:"bestAskSize"`
	// Open, High and Low cover the past 24 hours
	Open types.Number `json:"open"`
	High types.Number `json:"high"`
	Low  types.Number `json:"low"`
	// BaseVolume and QuoteVolume add up the maker and taker volume of each trade
	BaseVolume  types.Number `json:"baseVolume"`
	QuoteVolume types.Number `json:"quoteVolume"`
	LastPrice   types.Number `json:"lastPrice"`
	// Size is the size of the last trade
	Size        types.Number `json:"size"`
	PriceChange types.Number `json:"priceChange"`
	// PriceChangePercent is a ratio for spot, where 0.0113 is 1.13%, but a percentage for futures, where 1.1475 is
	// 1.1475%
	PriceChangePercent types.Number `json:"priceChangePercent"`
	IndexPrice         types.Number `json:"indexPrice"`
	MarkPrice          types.Number `json:"markPrice"`
	// Sequence, Side, TradeID and Timestamp describe the last trade; KuCoin sends them, undocumented, for futures only
	Sequence  uint64     `json:"sequence"`
	Side      string     `json:"side"`
	TradeID   string     `json:"tradeId"`
	Timestamp types.Time `json:"ts"`
}

// UTASymbolResponse holds the trading rules of a product type's symbols
type UTASymbolResponse struct {
	TradeType string      `json:"tradeType"`
	List      []UTASymbol `json:"list"`
}

// UTASymbol holds a spot symbol's or a futures contract's trading rules; each product type fills its own fields
type UTASymbol struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	// BaseCurrency can be a futures contract's base, such as XBT
	BaseCurrency  currency.Code `json:"baseCurrency"`
	QuoteCurrency currency.Code `json:"quoteCurrency"`
	Market        string        `json:"market"`
	FeeCurrency   currency.Code `json:"feeCurrency"`
	// TradingStatus is TradingEnabled or TradingDisabled for spot, and Init, Open, PrepareSettled, BeingSettled,
	// Settled, Paused, Closed or CancelOnly for futures
	TradingStatus       string       `json:"tradingStatus"`
	PriceLimitRatio     types.Number `json:"priceLimitRatio"`
	FeeCategory         string       `json:"feeCategory"`
	MakerFeeCoefficient types.Number `json:"makerFeeCoefficient"`
	TakerFeeCoefficient types.Number `json:"takerFeeCoefficient"`
	// ST reports whether the spot symbol is under special treatment
	ST                bool         `json:"st"`
	MinBaseOrderSize  types.Number `json:"minBaseOrderSize"`
	MaxBaseOrderSize  types.Number `json:"maxBaseOrderSize"`
	BaseOrderStep     types.Number `json:"baseOrderStep"`
	MinQuoteOrderSize types.Number `json:"minQuoteOrderSize"`
	MaxQuoteOrderSize types.Number `json:"maxQuoteOrderSize"`
	QuoteOrderStep    types.Number `json:"quoteOrderStep"`
	TickSize          types.Number `json:"tickSize"`
	MinFunds          types.Number `json:"minFunds"`
	// The call auction times arrive as bare numbers, although the docs type them as strings
	CallAuctionIsEnabled            bool         `json:"callauctionIsEnabled"`
	CallAuctionPriceFloor           types.Number `json:"callauctionPriceFloor"`
	CallAuctionPriceCeiling         types.Number `json:"callauctionPriceCeiling"`
	CallAuctionFirstStageStartTime  types.Time   `json:"callauctionFirstStageStartTime"`
	CallAuctionSecondStageStartTime types.Time   `json:"callauctionSecondStageStartTime"`
	CallAuctionThirdStageStartTime  types.Time   `json:"callauctionThirdStageStartTime"`
	// TradingStartTime arrives as a bare number for spot and as a quoted one for futures
	TradingStartTime    types.Time    `json:"tradingStartTime"`
	MaxLeverage         types.Number  `json:"maxLeverage"`
	BuyLimit            types.Number  `json:"buyLimit"`
	DisplayBaseCurrency currency.Code `json:"displayBaseCurrency"`
	ExpiryTime          types.Time    `json:"expiryTime"`
	IsInverse           bool          `json:"isInverse"`
	LotSize             types.Number  `json:"lotSize"`
	MaxMarketOrderSize  types.Number  `json:"maxMarketOrderSize"`
	MakerFeeRate        types.Number  `json:"makerFeeRate"`
	MarketStage         string        `json:"marketStage"`
	// MarketType is deprecated in favour of AssetClass and SubMarketType
	MarketType string `json:"marketType"`
	AssetClass string `json:"assetClass"`
	// SubMarketType is the stock market, such as US.STOCK, when AssetClass is STOCK
	SubMarketType        string        `json:"subMarketType"`
	MaxPrice             types.Number  `json:"maxPrice"`
	PreMarketToPerpDate  types.Time    `json:"preMarketToPerpDate"`
	UnitSize             types.Number  `json:"unitSize"`
	IndexSourceExchanges []string      `json:"indexSourceExchanges"`
	TakerFeeRate         types.Number  `json:"takerFeeRate"`
	SettlementFeeRate    types.Number  `json:"settlementFeeRate"`
	SettlementTime       types.Time    `json:"settlementTime"`
	SettlementCurrency   currency.Code `json:"settlementCurrency"`
	SellLimit            types.Number  `json:"sellLimit"`
	IndexPriceTickSize   types.Number  `json:"indexPriceTickSize"`
	// ContractType is 0 for a perpetual contract and 1 for a delivery contract
	ContractType string `json:"contractType"`
	// K, M, F, MMRLimit and MMRLeverageConstant are undocumented futures margin parameters, which classic futures
	// contracts also carry
	K                   types.Number `json:"k"`
	M                   types.Number `json:"m"`
	F                   types.Number `json:"f"`
	MMRLimit            types.Number `json:"mmrLimit"`
	MMRLeverageConstant types.Number `json:"mmrLevConstant"`
}

// UTAKlinesRequest holds the parameters for GetUTAKlines
type UTAKlinesRequest struct {
	// Symbol and TradeType are required
	Symbol    string
	TradeType string
	// KlineType is TRADE, the default, MARK_PRICE, INDEX_PRICE or PREMIUM_INDEX; spot supports TRADE only
	KlineType string
	// Interval is required: 1min, 3min, 5min, 15min, 30min, 1hour, 2hour, 4hour, 6hour, 8hour, 12hour, 1day, 3day,
	// 1week or 1month; futures supports neither 6hour nor 3day
	Interval string
	StartAt  time.Time
	EndAt    time.Time
}

// UTAKlinesResponse holds a symbol's candles, newest first
type UTAKlinesResponse struct {
	TradeType string `json:"tradeType"`
	// KlineType is undocumented and sent for futures only
	KlineType string     `json:"klineType"`
	Symbol    string     `json:"symbol"`
	List      []UTAKline `json:"list"`
}

// UTAKline holds a candle, which KuCoin sends as an array
type UTAKline struct {
	StartTime  types.Time
	OpenPrice  types.Number
	HighPrice  types.Number
	LowPrice   types.Number
	ClosePrice types.Number
	// Volume is in the base currency for spot and in contracts for futures
	Volume types.Number
	// Amount is the value traded, in the quote currency
	Amount types.Number
}

// UnmarshalJSON decodes a candle from its array form
func (k *UTAKline) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&k.StartTime, &k.OpenPrice, &k.HighPrice, &k.LowPrice, &k.ClosePrice, &k.Volume, &k.Amount})
}

// UTAOrderbookRequest holds the parameters for GetUTAOrderbook
type UTAOrderbookRequest struct {
	// TradeType, Symbol and Limit are required; Limit is 20, 100 or FULL
	TradeType string
	Symbol    string
	Limit     string
	// RPIFilter asks for each level's RPI order size to be split out; it applies to futures
	RPIFilter bool
}

// UTAOrderbookResponse holds an orderbook snapshot
type UTAOrderbookResponse struct {
	TradeType string              `json:"tradeType"`
	Symbol    string              `json:"symbol"`
	Sequence  uint64              `json:"sequence"`
	Bids      []UTAOrderbookLevel `json:"bids"`
	Asks      []UTAOrderbookLevel `json:"asks"`
}

// UTAOrderbookLevel holds a price level, which KuCoin sends as [price, size], or as [price, non-RPI size, RPI size] when
// the request sets rpiFilter
type UTAOrderbookLevel struct {
	Price types.Number
	// Size is the level's size, or the size of its non-RPI orders when the request sets RPIFilter
	Size types.Number
	// RPISize is the size of the level's Retail Price Improvement orders; KuCoin sends it only when the request sets
	// RPIFilter
	RPISize types.Number
}

// UnmarshalJSON decodes a price level from its array form
func (l *UTAOrderbookLevel) UnmarshalJSON(data []byte) error {
	var level []types.Number
	if err := json.Unmarshal(data, &level); err != nil {
		return err
	}
	switch len(level) {
	case 2:
		l.Price, l.Size, l.RPISize = level[0], level[1], 0
	case 3:
		l.Price, l.Size, l.RPISize = level[0], level[1], level[2]
	default:
		return fmt.Errorf("%w: %s", errUTAOrderbookLevelLength, data)
	}
	return nil
}

// UTAIndexPriceRequest holds the parameters for GetUTAIndexPrice
type UTAIndexPriceRequest struct {
	// Symbol is required
	Symbol   string
	StartAt  time.Time
	EndAt    time.Time
	PageSize uint64
	// LastID is the previous page's LastID, which pages back to older records
	LastID uint64
}

// UTAIndexPriceResponse holds a page of a futures contract's index prices, newest first
type UTAIndexPriceResponse struct {
	Items  []UTAIndexPrice `json:"items"`
	LastID uint64          `json:"lastId"`
}

// UTAIndexPrice holds a futures contract's index price and the exchange prices it weighs
type UTAIndexPrice struct {
	Symbol string `json:"symbol"`
	// Granularity is the interval between data points in milliseconds
	Granularity       uint64                       `json:"granularity"`
	Timestamp         types.Time                   `json:"ts"`
	IndexPrice        types.Number                 `json:"indexPrice"`
	DecompositionList []UTAIndexPriceDecomposition `json:"decompositionList"`
}

// UTAIndexPriceDecomposition holds an exchange's price and its weight in an index price
type UTAIndexPriceDecomposition struct {
	Exchange string       `json:"exchange"`
	Price    types.Number `json:"price"`
	Weight   types.Number `json:"weight"`
}

// UTATradesResponse holds a symbol's latest public trades
type UTATradesResponse struct {
	TradeType string     `json:"tradeType"`
	List      []UTATrade `json:"list"`
}

// UTATrade holds a public trade
type UTATrade struct {
	Sequence uint64       `json:"sequence"`
	TradeID  string       `json:"tradeId"`
	Price    types.Number `json:"price"`
	Size     types.Number `json:"size"`
	// Side is the taker's side
	Side      string     `json:"side"`
	Timestamp types.Time `json:"ts"`
	// IsRPITrade reports whether a Retail Price Improvement order filled; futures only
	IsRPITrade bool `json:"isRpiTrade"`
}

// UTAPositionTiersRequest holds the parameters for GetUTAPositionTiers
type UTAPositionTiersRequest struct {
	// TradeType, MarginMode, Data and AccountType are required. TradeType is MARGIN or FUTURES, MarginMode is CROSS and
	// AccountType is UNIFIED
	TradeType string
	// Currencies, up to 10, are required when Data is BORROW
	Currencies []currency.Code
	MarginMode string
	// Data is BORROW, for margin borrowing tiers, or RISK_LIMIT, for futures risk limit tiers
	Data        string
	AccountType string
	// Symbols are required when Data is RISK_LIMIT
	Symbols []string
}

// UTAPositionTier holds a margin borrowing tier or a futures risk limit tier
type UTAPositionTier struct {
	// Currency is set for margin tiers and Symbol for futures tiers
	Currency    currency.Code `json:"currency"`
	Symbol      string        `json:"symbol"`
	Tier        uint64        `json:"tier"`
	MinSize     types.Number  `json:"minSize"`
	MaxSize     types.Number  `json:"maxSize"`
	MaxLeverage types.Number  `json:"maxLeverage"`
	// InitialMarginRate is undocumented and sent for futures only
	InitialMarginRate  types.Number `json:"initialMarginRate"`
	MaintainMarginRate types.Number `json:"maintainMarginRate"`
	// QuickCalculationOffset is the futures maintenance margin deduction: the maintenance margin is the notional value
	// times MaintainMarginRate, less QuickCalculationOffset
	QuickCalculationOffset types.Number `json:"quickCalOffset"`
}

// UTACollateralRatio holds a currency's collateral discount tiers
type UTACollateralRatio struct {
	Currency          currency.Code            `json:"currency"`
	CDRConfigurations []UTACollateralRatioTier `json:"cdrConfigs"`
}

// UTACollateralRatioTier holds a collateral discount tier, which applies to equity, in the currency, between Min and Max
type UTACollateralRatioTier struct {
	// Tier ranks the tiers; a smaller number is a higher tier
	Tier uint64       `json:"tier"`
	Min  types.Number `json:"min"`
	Max  types.Number `json:"max"`
	// CDR is the cross-asset discount rate
	CDR types.Number `json:"cdr"`
	// CDREquity is the most equity the tier converts at its discount rate
	CDREquity types.Number `json:"cdrEquity"`
}

// UTABorrowableCurrency holds a currency the unified account can borrow
type UTABorrowableCurrency struct {
	Currency currency.Code `json:"currency"`
}

// UTACurrentFundingRate holds a futures contract's current funding rate
type UTACurrentFundingRate struct {
	Symbol          string       `json:"symbol"`
	NextFundingRate types.Number `json:"nextFundingRate"`
	// FundingTime is the next settlement time
	FundingTime      types.Time   `json:"fundingTime"`
	FundingRateCap   types.Number `json:"fundingRateCap"`
	FundingRateFloor types.Number `json:"fundingRateFloor"`
	// CurrentGranularity and NewGranularity are settlement intervals in milliseconds; NewGranularity applies from
	// NewGranularityStartTime
	CurrentGranularity      uint64     `json:"currentGranularity"`
	NewGranularity          uint64     `json:"newGranularity"`
	NewGranularityStartTime types.Time `json:"newGranularityStartTime"`
}

// UTAFundingRateHistoryResponse holds a futures contract's funding rates at each settlement, newest first
type UTAFundingRateHistoryResponse struct {
	Symbol string                   `json:"symbol"`
	List   []UTAHistoricFundingRate `json:"list"`
}

// UTAHistoricFundingRate holds the funding rate of a settlement
type UTAHistoricFundingRate struct {
	FundingRate types.Number `json:"fundingRate"`
	Timestamp   types.Time   `json:"ts"`
}

// UTAFuturesOpenInterestRequest holds the parameters for GetUTAFuturesOpenInterest
type UTAFuturesOpenInterestRequest struct {
	// Symbols, up to 10, select real time open interest, of every contract when empty; with Interval set, exactly one
	// symbol is required
	Symbols []string
	// Interval selects historic open interest: 5min, 15min, 30min, 1hour or 4hour, kept for 7 days, or 1day, kept for
	// 70 days. StartAt, EndAt and PageSize apply to historic open interest only
	Interval string
	StartAt  time.Time
	EndAt    time.Time
	PageSize uint64
}

// UTAFuturesOpenInterest holds a futures contract's open interest
type UTAFuturesOpenInterest struct {
	// Symbol is undocumented and sent for real time open interest only
	Symbol string `json:"symbol"`
	// OpenInterest is in contracts
	OpenInterest types.Number `json:"openInterest"`
	Timestamp    types.Time   `json:"ts"`
}

// UTAInterestRateIndexRequest holds the parameters for GetUTAInterestRateIndex
type UTAInterestRateIndexRequest struct {
	// Symbol is required
	Symbol  string
	StartAt time.Time
	EndAt   time.Time
	// LastID is the previous page's LastID, which pages back to older records
	LastID uint64
	// PageSize defaults to 20
	PageSize uint64
}

// UTAInterestRateIndexResponse holds a page of a futures contract's interest rates, newest first
type UTAInterestRateIndexResponse struct {
	Items  []UTAInterestRate `json:"items"`
	LastID uint64            `json:"lastId"`
}

// UTAInterestRate holds a futures contract's interest rate
type UTAInterestRate struct {
	Symbol       string       `json:"symbol"`
	InterestRate types.Number `json:"interestRate"`
	Timestamp    types.Time   `json:"ts"`
}

// UTAPlatform24HourMarketStatisticsResponse holds the platform's 24 hour spot and futures turnover
type UTAPlatform24HourMarketStatisticsResponse struct {
	Spot    UTAMarketTurnover `json:"spot"`
	Futures UTAMarketTurnover `json:"futures"`
}

// UTAMarketTurnover holds a market's 24 hour turnover
type UTAMarketTurnover struct {
	// TurnoverOf24Hour is in USD
	TurnoverOf24Hour types.Number `json:"turnoverOf24h"`
}

// UTACallAuctionInfoResponse holds a spot symbol's call auction estimates; every field is null while the symbol is not
// in a call auction
type UTACallAuctionInfoResponse struct {
	Symbol                  string       `json:"symbol"`
	EstimatedPrice          types.Number `json:"estimatedPrice"`
	EstimatedSize           types.Number `json:"estimatedSize"`
	SellOrderRangeLowPrice  types.Number `json:"sellOrderRangeLowPrice"`
	SellOrderRangeHighPrice types.Number `json:"sellOrderRangeHighPrice"`
	BuyOrderRangeLowPrice   types.Number `json:"buyOrderRangeLowPrice"`
	BuyOrderRangeHighPrice  types.Number `json:"buyOrderRangeHighPrice"`
	// Time is when the price was estimated
	Time types.Time `json:"time"`
}

// UTAThirdPartyCustodyCurrency holds a settlement currency a custodian supports
type UTAThirdPartyCustodyCurrency struct {
	Custodian string        `json:"custodian"`
	Currency  currency.Code `json:"currency"`
	Precision uint64        `json:"precision"`
}

// UTAServiceStatusResponse holds a product type's service status
type UTAServiceStatusResponse struct {
	TradeType string `json:"tradeType"`
	// ServerStatus is open, close, for maintenance, or cancelonly, which allows cancelling orders but not placing them
	ServerStatus string `json:"serverStatus"`
	Message      string `json:"msg"`
}

// UTAKYCRegion holds a KYC region
type UTAKYCRegion struct {
	Code        string `json:"code"`
	EnglishName string `json:"enName"`
}
