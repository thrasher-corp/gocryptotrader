package okx

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// Trade Modes

	// TradeModeCross trade mode, cross
	TradeModeCross = "cross"
	// TradeModeIsolated trade mode, isolated
	TradeModeIsolated = "isolated"
	// TradeModeCash trade mode, cash
	TradeModeCash = "cash"

	// Algo Order Types

	// AlgoOrdTypeGrid algo order type,grid
	AlgoOrdTypeGrid = "grid"
	// AlgoOrdTypeContractGrid algo order type, contract_grid
	AlgoOrdTypeContractGrid = "contract_grid"

	// Position Side for placing order
	positionSideLong  = "long"
	positionSideShort = "short"
	positionSideNet   = "net"
)

// order types, margin balance types, and instrument types constants
const (
	orderLimit                            = "limit"
	orderMarket                           = "market"
	orderPostOnly                         = "post_only"
	orderFOK                              = "fok"
	orderIOC                              = "ioc"
	orderOptimalLimitIOC                  = "optimal_limit_ioc"
	orderConditional                      = "conditional"
	orderMoveOrderStop                    = "move_order_stop"
	orderChase                            = "chase"
	orderTWAP                             = "twap"
	orderTrigger                          = "trigger"
	orderMarketMakerProtectionAndPostOnly = "mmp_and_post_only"
	orderMarketMakerProtection            = "mmp"
	orderOCO                              = "oco"

	// represents a margin balance type
	marginBalanceReduce = "reduce"
	marginBalanceAdd    = "add"

	// Instrument Types ( Asset Types )

	instTypeFutures  = "FUTURES"   // instrument type "futures"
	instTypeANY      = "ANY"       // instrument type ""
	instTypeSpot     = "SPOT"      // instrument type "spot"
	instTypeSwap     = "SWAP"      // instrument type "swap"
	instTypeOption   = "OPTION"    // instrument type "option"
	instTypeMargin   = "MARGIN"    // instrument type "margin"
	instTypeContract = "CONTRACTS" // instrument type "contract"

	operationSubscribe   = "subscribe"
	operationUnsubscribe = "unsubscribe"
	operationLogin       = "login"
)

var (
	errIndexComponentNotFound               = errors.New("unable to fetch index components")
	errLimitValueExceedsMaxOf100            = errors.New("limit value exceeds the maximum value 100")
	errMissingInstrumentID                  = errors.New("missing instrument ID")
	errEitherInstIDOrCcyIsRequired          = errors.New("either parameter instId or ccy is required")
	errInvalidTradeMode                     = errors.New("unacceptable required argument, trade mode")
	errMissingExpiryTimeParameter           = errors.New("missing expiry date parameter")
	errInvalidTradeModeValue                = errors.New("invalid trade mode value")
	errCurrencyQuantityTypeRequired         = errors.New("only base_ccy and quote_ccy quantity types are supported")
	errInvalidNewSizeOrPriceInformation     = errors.New("invalid new size or price information")
	errSizeOrPriceIsRequired                = errors.New("either size or price is required")
	errInvalidPriceLimit                    = errors.New("invalid price limit value")
	errMissingIntervalValue                 = errors.New("missing interval value")
	errMissingSizeLimit                     = errors.New("missing required parameter 'szLimit'")
	errMissingEitherAlgoIDOrState           = errors.New("either algo ID or order state is required")
	errAlgoIDRequired                       = errors.New("algo ID is required")
	errMissingValidWithdrawalID             = errors.New("missing valid withdrawal ID")
	errInstrumentFamilyRequired             = errors.New("instrument family is required")
	errCountdownTimeoutRequired             = errors.New("countdown timeout is required")
	errInstrumentIDorFamilyRequired         = errors.New("either instrument ID or instrument family is required")
	errInvalidQuantityLimit                 = errors.New("invalid quantity limit")
	errInvalidInstrumentType                = errors.New("invalid instrument type")
	errMissingValidGreeksType               = errors.New("missing valid greeks type")
	errMissingIsolatedMarginTradingSetting  = errors.New("missing isolated margin trading setting, isolated margin trading settings automatic:Auto transfers autonomy:Manual transfers")
	errInvalidCounterParties                = errors.New("missing counter parties")
	errMissingRFQIDOrQuoteID                = errors.New("either RFQ ID or Quote ID is missing")
	errMissingRFQID                         = errors.New("error missing rfq ID")
	errMissingLegs                          = errors.New("missing legs")
	errMissingSizeOfQuote                   = errors.New("missing size of quote leg")
	errMissingLegsQuotePrice                = errors.New("error missing quote price")
	errInvalidLoanAllocationValue           = errors.New("invalid loan allocation value, must be between 0 to 100")
	errInvalidSubaccount                    = errors.New("invalid sub-account type")
	errMissingAlgoOrderType                 = errors.New("missing algo order type 'grid': Spot grid, \"contract_grid\": Contract grid")
	errInvalidGridQuantity                  = errors.New("invalid grid quantity (grid number)")
	errRunTypeRequired                      = errors.New("runType is required; possible values are 1: Arithmetic, 2: Geometric")
	errMissingRequiredArgumentDirection     = errors.New("missing required argument, direction")
	errInvalidLeverage                      = errors.New("invalid leverage value")
	errMissingValidStopType                 = errors.New("invalid grid order stop type, only values are \"1\" and \"2\" ")
	errMissingSubOrderType                  = errors.New("missing sub order type")
	errMissingQuantity                      = errors.New("invalid quantity to buy or sell")
	errAddressRequired                      = errors.New("address is required")
	errMaxRFQOrdersToCancel                 = errors.New("no more than 100 RFQ cancel order parameter is allowed")
	errInvalidUnderlying                    = errors.New("invalid underlying")
	errInstrumentFamilyOrUnderlyingRequired = errors.New("either underlying or instrument family is required")
	errMissingRequiredParameter             = errors.New("missing required parameter")
	errMissingMakerInstrumentSettings       = errors.New("missing maker instrument settings")
	errInvalidSubAccountName                = errors.New("invalid sub-account name")
	errInvalidAPIKey                        = errors.New("invalid api key")
	errInvalidMarginTypeAdjust              = errors.New("invalid margin type adjust, only 'add' and 'reduce' are allowed")
	errInvalidAlgoOrderType                 = errors.New("invalid algo order type")
	errInvalidIPAddress                     = errors.New("invalid ip address")
	errInvalidAPIKeyPermission              = errors.New("invalid API Key permission")
	errInvalidDuration                      = errors.New("invalid grid contract duration, only '7D', '30D', and '180D' are allowed")
	errInvalidProtocolType                  = errors.New("invalid protocol type, only 'staking' and 'defi' allowed")
	errExceedLimit                          = errors.New("limit exceeded")
	errOnlyThreeMonthsSupported             = errors.New("only three months of trade data retrieval supported")
	errOnlyOneResponseExpected              = errors.New("one response item expected")
	errStrategyNameRequired                 = errors.New("strategy name required")
	errRecurringDayRequired                 = errors.New("recurring day is required")
	errRecurringBuyTimeRequired             = errors.New("recurring buy time, the value range is an integer with value between 0 and 23")
	errSubPositionIDRequired                = errors.New("sub position ID is required")
	errUserIDRequired                       = errors.New("uid is required")
	errSubPositionCloseTypeRequired         = errors.New("sub position close type")
	errUniqueCodeRequired                   = errors.New("unique code is required")
	errLastDaysRequired                     = errors.New("last days required")
	errCopyInstrumentIDTypeRequired         = errors.New("copy instrument ID type is required")
	errInvalidChecksum                      = errors.New("invalid checksum")
	errInvalidPositionMode                  = errors.New("invalid position mode")
	errLendingTermIsRequired                = errors.New("lending term is required")
	errRateRequired                         = errors.New("lending rate is required")
	errQuarterValueRequired                 = errors.New("quarter is required")
	errYearRequired                         = errors.New("year is required")
	errBorrowTypeRequired                   = errors.New("borrow type is required")
	errMaxRateRequired                      = errors.New("max rate is required")
	errLendingSideRequired                  = errors.New("lending side is required")
	errPaymentMethodRequired                = errors.New("payment method required")
	errIDNotSet                             = errors.New("ID is not set")
	errMonthNameRequired                    = errors.New("month name is required")
	errPriceTrackingNotSet                  = errors.New("price tracking value not set")
	errInvoiceTextMissing                   = errors.New("missing invoice text")
	errFeeTypeUnsupported                   = errors.New("fee type is not supported")
)

// testNetKey this key is designed for using the testnet endpoints
// setting context.WithValue(ctx, testNetKey("testnet"), useTestNet)
// will ensure the appropriate headers are sent to OKx to use the testnet
type testNetKey string

var testNetVal = testNetKey("testnet")

type candlestickState int8

// candlestickState represents index candlestick states.
const (
	StateUncompleted candlestickState = iota
	StateCompleted
)

// PremiumInfo represents data on premiums for the past 6 months.
type PremiumInfo struct {
	InstrumentID string     `json:"instId"`
	Premium      string     `json:"premium"`
	Timestamp    types.Time `json:"ts"`
}

// TickerResponse represents the detailed data from the market ticker endpoint.
type TickerResponse struct {
	InstrumentType string        `json:"instType"`
	InstrumentID   currency.Pair `json:"instId"`
	LastTradePrice types.Number  `json:"last"`
	LastTradeSize  types.Number  `json:"lastSz"`
	BestAskPrice   types.Number  `json:"askPx"`
	BestAskSize    types.Number  `json:"askSz"`
	BestBidPrice   types.Number  `json:"bidPx"`
	BestBidSize    types.Number  `json:"bidSz"`
	Open24H        types.Number  `json:"open24h"`
	High24H        types.Number  `json:"high24h"`
	Low24H         types.Number  `json:"low24h"`
	VolCcy24H      types.Number  `json:"volCcy24h"`
	Vol24H         types.Number  `json:"vol24h"`

	OpenPriceInUTC0          string     `json:"sodUtc0"`
	OpenPriceInUTC8          string     `json:"sodUtc8"`
	TickerDataGenerationTime types.Time `json:"ts"`
}

// IndexTicker represents data from the index ticker.
type IndexTicker struct {
	InstID    string       `json:"instId"`
	IdxPx     types.Number `json:"idxPx"`
	High24H   types.Number `json:"high24h"`
	SodUtc0   types.Number `json:"sodUtc0"`
	Open24H   types.Number `json:"open24h"`
	Low24H    types.Number `json:"low24h"`
	SodUtc8   types.Number `json:"sodUtc8"`
	Timestamp types.Time   `json:"ts"`
}

// OrderBookResponseDetail contains the ask and bid orders, structured with fields that include the timestamp of order generation.
type OrderBookResponseDetail struct {
	Asks                []OrderbookItemDetail
	Bids                []OrderbookItemDetail
	GenerationTimestamp time.Time
}

// OrderbookItemDetail represents detailed information about currency bids.
type OrderbookItemDetail struct {
	DepthPrice        types.Number
	Amount            types.Number
	LiquidationOrders types.Number
	NumberOfOrders    types.Number
}

// UnmarshalJSON deserializes byte data into OrderbookItemDetail instance
func (o *OrderbookItemDetail) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[4]any{&o.DepthPrice, &o.Amount, &o.LiquidationOrders, &o.NumberOfOrders})
}

// CandlestickHistoryItem retrieves historical candlestick charts for the index or mark price from recent years.
type CandlestickHistoryItem struct {
	Timestamp    types.Time
	OpenPrice    types.Number
	HighestPrice types.Number
	LowestPrice  types.Number
	ClosePrice   types.Number
	Confirm      candlestickState
}

// UnmarshalJSON converts the data slice into a CandlestickHistoryItem instance.
func (c *CandlestickHistoryItem) UnmarshalJSON(data []byte) error {
	var state string
	if err := json.Unmarshal(data, &[6]any{&c.Timestamp, &c.OpenPrice, &c.HighestPrice, &c.LowestPrice, &c.ClosePrice, &state}); err != nil {
		return err
	}
	if state == "1" {
		c.Confirm = StateCompleted
	} else {
		c.Confirm = StateUncompleted
	}
	return nil
}

// CandleStick stores candlestick price data.
type CandleStick struct {
	OpenTime         types.Time
	OpenPrice        types.Number
	HighestPrice     types.Number
	LowestPrice      types.Number
	ClosePrice       types.Number
	Volume           types.Number
	QuoteAssetVolume types.Number
}

// UnmarshalJSON deserializes slice of data into Candlestick structure
func (c *CandleStick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&c.OpenTime, &c.OpenPrice, &c.HighestPrice, &c.LowestPrice, &c.ClosePrice, &c.Volume, &c.QuoteAssetVolume})
}

// TradeResponse represents the recent transaction instance
type TradeResponse struct {
	InstrumentID string       `json:"instId"`
	TradeID      string       `json:"tradeId"`
	Price        types.Number `json:"px"`
	Quantity     types.Number `json:"sz"`
	Side         order.Side   `json:"side"`
	Timestamp    types.Time   `json:"ts"`
	Count        string       `json:"count"`
}

// InstrumentFamilyTrade represents transaction information of instrument.
// instrument family, e.g. BTC-USD Applicable to OPTION
type InstrumentFamilyTrade struct {
	Vol24H    types.Number `json:"vol24h"`
	TradeInfo []struct {
		InstrumentID string       `json:"instId"`
		TradeID      string       `json:"tradeId"`
		Side         string       `json:"side"`
		Size         types.Number `json:"sz"`
		Price        types.Number `json:"px"`
		Timestamp    types.Time   `json:"ts"`
	} `json:"tradeInfo"`
	OptionType string `json:"optType"`
}

// OptionTrade holds option trade item
type OptionTrade struct {
	FillVolume       types.Number `json:"fillVol"`
	ForwardPrice     types.Number `json:"fwdPx"`
	IndexPrice       types.Number `json:"idxPx"`
	MarkPrice        types.Number `json:"markPx"`
	Price            types.Number `json:"px"`
	Size             types.Number `json:"sz"`
	InstrumentFamily string       `json:"instFamily"`
	InstrumentID     string       `json:"instId"`
	OptionType       string       `json:"optType"`
	Side             string       `json:"side"`
	TradeID          string       `json:"tradeId"`
	Timestamp        types.Time   `json:"ts"`
}

// TradingVolumeIn24HR response model
type TradingVolumeIn24HR struct {
	BlockVolumeInCNY   types.Number `json:"blockVolCny"`
	BlockVolumeInUSD   types.Number `json:"blockVolUsd"`
	TradingVolumeInUSD types.Number `json:"volUsd"`
	TradingVolumeInCny types.Number `json:"volCny"`
	Timestamp          types.Time   `json:"ts"`
}

// OracleSmartContractResponse represents the cryptocurrency price signed using the Open Oracle smart contract.
type OracleSmartContractResponse struct {
	Messages   []string          `json:"messages"`
	Prices     map[string]string `json:"prices"`
	Signatures []string          `json:"signatures"`
	Timestamp  types.Time        `json:"timestamp"`
}

// UsdCnyExchangeRate the exchange rate for converting from USD to CNV
type UsdCnyExchangeRate struct {
	UsdCny types.Number `json:"usdCny"`
}

// IndexComponent represents index component data on the market
type IndexComponent struct {
	Index      string               `json:"index"`
	Components []IndexComponentItem `json:"components"`
	Last       types.Number         `json:"last"`
	Timestamp  types.Time           `json:"ts"`
}

// IndexComponentItem an item representing the index component item
type IndexComponentItem struct {
	Symbol          string `json:"symbol"`
	SymbolPairPrice string `json:"symbolPx"`
	Weights         string `json:"wgt"`
	ConvertToPrice  string `json:"cnvPx"`
	ExchangeName    string `json:"exch"`
}

// InstrumentsFetchParams request params for requesting list of instruments
type InstrumentsFetchParams struct {
	InstrumentType   string // Mandatory
	Underlying       string // Optional
	InstrumentFamily string
	InstrumentID     string // Optional
}

// Instrument  representing an instrument with open contract
type Instrument struct {
	InstrumentType                  string        `json:"instType"`
	InstrumentID                    currency.Pair `json:"instId"`
	InstrumentFamily                string        `json:"instFamily"`
	Underlying                      string        `json:"uly"`
	Category                        string        `json:"category"`
	BaseCurrency                    string        `json:"baseCcy"`
	QuoteCurrency                   string        `json:"quoteCcy"`
	SettlementCurrency              string        `json:"settleCcy"`
	ContractValue                   types.Number  `json:"ctVal"`
	ContractMultiplier              types.Number  `json:"ctMult"`
	ContractValueCurrency           string        `json:"ctValCcy"`
	OptionType                      string        `json:"optType"`
	StrikePrice                     types.Number  `json:"stk"`
	ListTime                        types.Time    `json:"listTime"`
	ExpTime                         types.Time    `json:"expTime"`
	MaxLeverage                     types.Number  `json:"lever"`
	TickSize                        types.Number  `json:"tickSz"`
	LotSize                         types.Number  `json:"lotSz"`
	MinimumOrderSize                types.Number  `json:"minSz"`
	ContractType                    string        `json:"ctType"`
	Alias                           string        `json:"alias"`
	State                           string        `json:"state"`
	MaxQuantityOfSpotLimitOrder     types.Number  `json:"maxLmtSz"`
	MaxQuantityOfMarketLimitOrder   types.Number  `json:"maxMktSz"`
	MaxQuantityOfSpotTwapLimitOrder types.Number  `json:"maxTwapSz"`
	MaxSpotIcebergSize              types.Number  `json:"maxIcebergSz"`
	MaxTriggerSize                  types.Number  `json:"maxTriggerSz"`
	MaxStopSize                     types.Number  `json:"maxStopSz"`
}

// DeliveryHistoryDetail holds instrument ID and delivery price information detail
type DeliveryHistoryDetail struct {
	Type          string       `json:"type"`
	InstrumentID  string       `json:"insId"`
	DeliveryPrice types.Number `json:"px"`
}

// DeliveryHistory represents list of delivery history detail items and timestamp information
type DeliveryHistory struct {
	Timestamp types.Time              `json:"ts"`
	Details   []DeliveryHistoryDetail `json:"details"`
}

// OpenInterest Retrieve the total open interest for contracts on OKX
type OpenInterest struct {
	InstrumentType       asset.Item   `json:"instType"`
	InstrumentID         string       `json:"instId"`
	OpenInterest         types.Number `json:"oi"`
	OpenInterestCurrency types.Number `json:"oiCcy"`
	Timestamp            types.Time   `json:"ts"`
}

// FundingRateResponse response data for the Funding Rate for an instruction type
type FundingRateResponse struct {
	InstrumentType               string       `json:"instType"`
	InstrumentID                 string       `json:"instId"`
	FundingRateMethod            string       `json:"method"`
	FundingRate                  types.Number `json:"fundingRate"`
	NextFundingRate              types.Number `json:"nextFundingRate"`
	FundingTime                  types.Time   `json:"fundingTime"`
	NextFundingTime              types.Time   `json:"nextFundingTime"`
	MinFundingRate               types.Number `json:"minFundingRate"`
	MaxFundingRate               types.Number `json:"maxFundingRate"`
	SettlementStateOfFundingRate string       `json:"settState"`
	SettlementFundingRate        types.Number `json:"settFundingRate"`
	Premium                      string       `json:"premium"`
	Timestamp                    types.Time   `json:"ts"`
}

// LimitPriceResponse hold an information for
type LimitPriceResponse struct {
	InstrumentType string       `json:"instType"`
	InstrumentID   string       `json:"instId"`
	BuyLimit       types.Number `json:"buyLmt"`
	SellLimit      types.Number `json:"sellLmt"`
	Timestamp      types.Time   `json:"ts"`
}

// OptionMarketDataResponse holds response data for option market data
type OptionMarketDataResponse struct {
	InstrumentType string       `json:"instType"`
	InstrumentID   string       `json:"instId"`
	Underlying     string       `json:"uly"`
	Delta          types.Number `json:"delta"`
	Gamma          types.Number `json:"gamma"`
	Theta          types.Number `json:"theta"`
	Vega           types.Number `json:"vega"`
	DeltaBS        types.Number `json:"deltaBS"`
	GammaBS        types.Number `json:"gammaBS"`
	ThetaBS        types.Number `json:"thetaBS"`
	VegaBS         types.Number `json:"vegaBS"`
	RealVol        types.Number `json:"realVol"`
	BidVolatility  types.Number `json:"bidVol"`
	AskVolatility  types.Number `json:"askVol"`
	MarkVolatility types.Number `json:"markVol"`
	Leverage       types.Number `json:"lever"`
	ForwardPrice   types.Number `json:"fwdPx"`
	Timestamp      types.Time   `json:"ts"`
}

// DeliveryEstimatedPrice holds an estimated delivery or exercise price response
type DeliveryEstimatedPrice struct {
	InstrumentType         string       `json:"instType"`
	InstrumentID           string       `json:"instId"`
	EstimatedDeliveryPrice types.Number `json:"settlePx"`
	Timestamp              types.Time   `json:"ts"`
}

// DiscountRate represents the discount rate amount, currency, and other discount related information
type DiscountRate struct {
	Amount            types.Number           `json:"amt"`
	Currency          string                 `json:"ccy"`
	DiscountRateLevel string                 `json:"discountLv"`
	MinDiscountRate   types.Number           `json:"minDiscountRate"`
	DiscountInfo      []DiscountRateInfoItem `json:"discountInfo"`
}

// DiscountRateInfoItem represents discount info list item for discount rate response
type DiscountRateInfoItem struct {
	DiscountRate           types.Number `json:"discountRate"`
	MaxAmount              types.Number `json:"maxAmt"`
	MinAmount              types.Number `json:"minAmt"`
	Tiers                  string       `json:"tier"`
	LiqPenaltyRate         types.Number `json:"liqPenaltyRate"`
	DiscountCurrencyEquity types.Number `json:"disCcyEq"`
}

// LiquidationOrderRequestParams holds information to request liquidation orders
type LiquidationOrderRequestParams struct {
	InstrumentType string
	MarginMode     string // values are either isolated or crossed
	InstrumentID   string
	Currency       currency.Code
	Underlying     string
	Alias          string
	State          string
	Before         time.Time
	After          time.Time
	Limit          int64
}

// LiquidationOrder represents liquidation order item detailed information
type LiquidationOrder struct {
	Details          []LiquidationOrderDetailItem `json:"details"`
	InstrumentID     string                       `json:"instId"`
	InstrumentType   string                       `json:"instType"`
	TotalLoss        types.Number                 `json:"totalLoss"`
	Underlying       string                       `json:"uly"`
	InstrumentFamily string                       `json:"instFamily"`
}

// LiquidationOrderDetailItem represents the detail information of liquidation order
type LiquidationOrderDetailItem struct {
	BankruptcyLoss        string       `json:"bkLoss"`
	BankruptcyPrice       types.Number `json:"bkPx"`
	Currency              string       `json:"ccy"`
	PositionSide          string       `json:"posSide"`
	Side                  string       `json:"side"` // May be empty
	QuantityOfLiquidation types.Number `json:"sz"`
	Timestamp             types.Time   `json:"ts"`
}

// MarkPrice represents a mark price information for a single instrument ID
type MarkPrice struct {
	InstrumentType string       `json:"instType"`
	InstrumentID   string       `json:"instId"`
	MarkPrice      types.Number `json:"markPx"`
	Timestamp      types.Time   `json:"ts"`
}

// PositionTiers represents position tier detailed information
type PositionTiers struct {
	BaseMaxLoan                  string       `json:"baseMaxLoan"`
	InitialMarginRequirement     string       `json:"imr"`
	InstrumentID                 string       `json:"instId"`
	MaximumLeverage              string       `json:"maxLever"`
	MaximumSize                  types.Number `json:"maxSz"`
	MinSize                      types.Number `json:"minSz"`
	MaintenanceMarginRequirement string       `json:"mmr"`
	OptionalMarginFactor         string       `json:"optMgnFactor"`
	QuoteMaxLoan                 string       `json:"quoteMaxLoan"`
	Tier                         string       `json:"tier"`
	Underlying                   string       `json:"uly"`
}

// InterestRateLoanQuotaBasic holds the basic Currency, loan,and interest rate information
type InterestRateLoanQuotaBasic struct {
	Currency     string       `json:"ccy"`
	LoanQuota    string       `json:"quota"`
	InterestRate types.Number `json:"rate"`
}

// InterestRateLoanQuotaItem holds the basic Currency, loan,interest rate, and other level and VIP related information
type InterestRateLoanQuotaItem struct {
	Basic   []InterestRateLoanQuotaBasic `json:"basic"`
	VIP     []InterestAndLoanDetail      `json:"vip"`
	Regular []InterestAndLoanDetail      `json:"regular"`
}

// InterestAndLoanDetail represents an interest rate and loan quota information
type InterestAndLoanDetail struct {
	InterestRateDiscount types.Number `json:"irDiscount"`
	LoanQuotaCoefficient types.Number `json:"loanQuotaCoef"`
	UserLevel            string       `json:"level"`
}

// VIPInterestRateAndLoanQuotaInformation holds interest rate and loan quoata information for VIP users
type VIPInterestRateAndLoanQuotaInformation struct {
	InterestRateLoanQuotaBasic
	LevelList []struct {
		Level     string       `json:"level"`
		LoanQuota types.Number `json:"loanQuota"`
	} `json:"levelList"`
}

// InsuranceFundInformationRequestParams insurance fund balance information
type InsuranceFundInformationRequestParams struct {
	InstrumentType   string        `json:"instType"`
	InsuranceType    string        `json:"type"` //  Type values allowed are `liquidation_balance_deposit, bankruptcy_loss, and platform_revenue`
	Underlying       string        `json:"uly"`
	InstrumentFamily string        `json:"instFamily"`
	Currency         currency.Code `json:"ccy"`
	Before           time.Time     `json:"before"`
	After            time.Time     `json:"after"`
	Limit            int64         `json:"limit"`
}

// InsuranceFundInformation holds insurance fund information data
type InsuranceFundInformation struct {
	Details          []InsuranceFundInformationDetail `json:"details"`
	InstrumentFamily string                           `json:"instFamily"`
	InstrumentType   string                           `json:"instType"`
	Total            types.Number                     `json:"total"`
}

// InsuranceFundInformationDetail represents an Insurance fund information item for a
// single currency and type
type InsuranceFundInformationDetail struct {
	Timestamp                    types.Time   `json:"ts"`
	Amount                       types.Number `json:"amt"`
	Balance                      types.Number `json:"balance"`
	Currency                     string       `json:"ccy"`
	InsuranceType                string       `json:"type"`
	MaxBalance                   types.Number `json:"maxBal"`
	MaxBalTimestamp              types.Time   `json:"maxBalTs"`
	RealTimeInsuranceDeclineRate types.Number `json:"decRate"`
	ADLType                      string       `json:"adlType"`
}

// SupportedCoinsData holds information about currencies supported by the trading data endpoints
type SupportedCoinsData struct {
	Contract       []string `json:"contract"`
	TradingOptions []string `json:"option"`
	Spot           []string `json:"spot"`
}

// TakerVolume represents taker volume information with creation timestamp
type TakerVolume struct {
	Timestamp  types.Time
	SellVolume types.Number
	BuyVolume  types.Number
}

// UnmarshalJSON deserializes a slice of data into TakerVolume
func (t *TakerVolume) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[3]any{&t.Timestamp, &t.SellVolume, &t.BuyVolume})
}

// MarginLendRatioItem represents margin lend ration information and creation timestamp
type MarginLendRatioItem struct {
	Timestamp       types.Time
	MarginLendRatio types.Number
}

// UnmarshalJSON deserializes a slice of data into MarginLendRatio
func (m *MarginLendRatioItem) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[2]any{&m.Timestamp, &m.MarginLendRatio})
}

// LongShortRatio represents the ratio of users with net long vs net short positions for futures and perpetual swaps
type LongShortRatio struct {
	Timestamp       types.Time
	MarginLendRatio types.Number
}

// UnmarshalJSON deserializes a slice of data into LongShortRatio
func (l *LongShortRatio) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[2]any{&l.Timestamp, &l.MarginLendRatio})
}

// OpenInterestVolume represents open interest and trading volume item for currencies of futures and perpetual swaps
type OpenInterestVolume struct {
	Timestamp    types.Time
	OpenInterest types.Number
	Volume       types.Number
}

// UnmarshalJSON deserializes json data into OpenInterestVolume struct
func (p *OpenInterestVolume) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[3]any{&p.Timestamp, &p.OpenInterest, &p.Volume})
}

// OpenInterestVolumeRatio represents open interest and trading volume ratio for currencies of futures and perpetual swaps
type OpenInterestVolumeRatio struct {
	Timestamp         types.Time
	OpenInterestRatio types.Number
	VolumeRatio       types.Number
}

// UnmarshalJSON deserializes json data into OpenInterestVolumeRatio
func (o *OpenInterestVolumeRatio) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[3]any{&o.Timestamp, &o.OpenInterestRatio, &o.VolumeRatio})
}

// ExpiryOpenInterestAndVolume represents  open interest and trading volume of calls and puts for each upcoming expiration
type ExpiryOpenInterestAndVolume struct {
	Timestamp        types.Time
	ExpiryTime       time.Time
	CallOpenInterest types.Number
	PutOpenInterest  types.Number
	CallVolume       types.Number
	PutVolume        types.Number
}

// UnmarshalJSON deserializes slice of data into ExpiryOpenInterestAndVolume structure
func (e *ExpiryOpenInterestAndVolume) UnmarshalJSON(data []byte) error {
	var expiryTimeString string
	err := json.Unmarshal(data, &[6]any{&e.Timestamp, &expiryTimeString, &e.CallOpenInterest, &e.PutOpenInterest, &e.CallVolume, &e.PutVolume})
	if err != nil {
		return err
	}
	if expiryTimeString != "" && len(expiryTimeString) == 8 {
		year, err := strconv.ParseInt(expiryTimeString[0:4], 10, 64)
		if err != nil {
			return err
		}
		month, err := strconv.ParseInt(expiryTimeString[4:6], 10, 64)
		if err != nil {
			return err
		}
		var months string
		var days string
		if month <= 9 {
			months = "0" + strconv.FormatInt(month, 10)
		} else {
			months = strconv.FormatInt(month, 10)
		}
		day, err := strconv.ParseInt(expiryTimeString[6:], 10, 64)
		if err != nil {
			return err
		}
		if day <= 9 {
			days = "0" + strconv.FormatInt(day, 10)
		} else {
			days = strconv.FormatInt(day, 10)
		}
		e.ExpiryTime, err = time.Parse("2006-01-02", strconv.FormatInt(year, 10)+"-"+months+"-"+days)
		if err != nil {
			return err
		}
	}
	return nil
}

// StrikeOpenInterestAndVolume represents open interest and volume for both buyers and sellers of calls and puts
type StrikeOpenInterestAndVolume struct {
	Timestamp        types.Time
	Strike           types.Number
	CallOpenInterest types.Number
	PutOpenInterest  types.Number
	CallVolume       types.Number
	PutVolume        types.Number
}

// UnmarshalJSON deserializes slice of byte data into StrikeOpenInterestAndVolume
func (s *StrikeOpenInterestAndVolume) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[6]any{&s.Timestamp, &s.Strike, &s.CallOpenInterest, &s.PutOpenInterest, &s.CallVolume, &s.PutVolume})
}

// CurrencyTakerFlow holds the taker volume information for a single currency
type CurrencyTakerFlow struct {
	Timestamp       types.Time
	CallBuyVolume   types.Number
	CallSellVolume  types.Number
	PutBuyVolume    types.Number
	PutSellVolume   types.Number
	CallBlockVolume types.Number
	PutBlockVolume  types.Number
}

// UnmarshalJSON deserializes a slice of byte data into CurrencyTakerFlow
func (c *CurrencyTakerFlow) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&c.Timestamp, &c.CallBuyVolume, &c.CallSellVolume, &c.PutBuyVolume, &c.PutSellVolume, &c.CallBlockVolume, &c.PutBlockVolume})
}

// PlaceOrderRequestParam requesting parameter for placing an order
type PlaceOrderRequestParam struct {
	AssetType     asset.Item `json:"-"`
	InstrumentID  string     `json:"instId"`
	TradeMode     string     `json:"tdMode"` // cash isolated
	ClientOrderID string     `json:"clOrdId,omitempty"`
	Currency      string     `json:"ccy,omitempty"` // Only applicable to cross MARGIN orders in Single-currency margin.
	OrderTag      string     `json:"tag,omitempty"`
	Side          string     `json:"side"`
	PositionSide  string     `json:"posSide,omitempty"` // long/short only for FUTURES and SWAP
	OrderType     string     `json:"ordType"`           // Time in force for the order
	Amount        float64    `json:"sz,string"`
	Price         float64    `json:"px,string,omitempty"` // Only applicable to limit,post_only,fok,ioc,mmp,mmp_and_post_only order.
	// Options orders
	PlaceOptionsOrder                    string `json:"pxUsd,omitempty"` // Place options orders in USD
	PlaceOptionsOrderOnImpliedVolatility string `json:"pxVol,omitempty"` // Place options orders based on implied volatility, where 1 represents 100%

	ReduceOnly              bool   `json:"reduceOnly,string,omitempty"`
	TargetCurrency          string `json:"tgtCcy,omitempty"`  // values base_ccy and quote_ccy for spot market orders
	SelfTradePreventionMode string `json:"stpMode,omitempty"` // Default to cancel maker, `cancel_maker`,`cancel_taker`, `cancel_both``
	// Added in the websocket requests
	BanAmend bool `json:"banAmend,omitempty"` // Whether the SPOT Market Order size can be amended by the system.
}

// Validate validates the PlaceOrderRequestParam
func (arg *PlaceOrderRequestParam) Validate() error {
	if arg == nil {
		return fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return errMissingInstrumentID
	}
	if arg.AssetType == asset.Spot || arg.AssetType == asset.Margin || arg.AssetType == asset.Empty {
		arg.Side = strings.ToLower(arg.Side)
		if arg.Side != order.Buy.Lower() && arg.Side != order.Sell.Lower() {
			return fmt.Errorf("%w %s", order.ErrSideIsInvalid, arg.Side)
		}
	}
	if !slices.Contains([]string{"", TradeModeCross, TradeModeIsolated, TradeModeCash}, arg.TradeMode) {
		return fmt.Errorf("%w %s", errInvalidTradeModeValue, arg.TradeMode)
	}
	if arg.AssetType == asset.Futures || arg.AssetType == asset.PerpetualSwap {
		arg.PositionSide = strings.ToLower(arg.PositionSide)
		if !slices.Contains([]string{"long", "short"}, arg.PositionSide) {
			return fmt.Errorf("%w: %q, 'long' or 'short' supported", order.ErrSideIsInvalid, arg.PositionSide)
		}
	}
	arg.OrderType = strings.ToLower(arg.OrderType)
	if !slices.Contains([]string{orderMarket, orderLimit, orderPostOnly, orderFOK, orderIOC, orderOptimalLimitIOC, "mmp", "mmp_and_post_only"}, arg.OrderType) {
		return fmt.Errorf("%w: '%v'", order.ErrTypeIsInvalid, arg.OrderType)
	}
	if arg.Amount <= 0 {
		return order.ErrAmountBelowMin
	}
	if !slices.Contains([]string{"", "base_ccy", "quote_ccy"}, arg.TargetCurrency) {
		return errCurrencyQuantityTypeRequired
	}
	return nil
}

// OrderData response message for place, cancel, and amend an order requests.
type OrderData struct {
	OrderID       string     `json:"ordId"`
	RequestID     string     `json:"reqId"`
	ClientOrderID string     `json:"clOrdId"`
	Tag           string     `json:"tag"`
	StatusCode    int64      `json:"sCode,string"` // Anything above 0 is an error with an attached message
	StatusMessage string     `json:"sMsg"`
	Timestamp     types.Time `json:"ts"`
}

func (o *OrderData) Error() error {
	return getStatusError(o.StatusCode, o.StatusMessage)
}

// ResponseResult holds responses having a status result value
type ResponseResult struct {
	Result        bool   `json:"result"`
	StatusCode    int64  `json:"sCode,string"`
	StatusMessage string `json:"sMsg"`
}

func (r *ResponseResult) Error() error {
	return getStatusError(r.StatusCode, r.StatusMessage)
}

// CancelOrderRequestParam represents order parameters to cancel an order
type CancelOrderRequestParam struct {
	InstrumentID  string `json:"instId"`
	OrderID       string `json:"ordId"`
	ClientOrderID string `json:"clOrdId,omitempty"`
}

// CancelMassReqParam holds MMP batch cancel request parameters
type CancelMassReqParam struct {
	InstrumentType   string `json:"instType"`
	InstrumentFamily string `json:"instFamily"`
}

// AmendOrderRequestParams represents amend order requesting parameters
type AmendOrderRequestParams struct {
	InstrumentID    string  `json:"instId"`
	CancelOnFail    bool    `json:"cxlOnFail,omitempty"`
	OrderID         string  `json:"ordId,omitempty"`
	ClientOrderID   string  `json:"clOrdId,omitempty"`
	ClientRequestID string  `json:"reqId,omitempty"`
	NewQuantity     float64 `json:"newSz,omitempty,string"`
	NewPrice        float64 `json:"newPx,omitempty,string"`

	// Modify options orders using USD prices
	// Only applicable to options.
	// When modifying options orders, users can only fill in one of the following: newPx, newPxUsd, or newPxVol.
	NewPriceInUSD float64 `json:"newPxUsd,omitempty,string"`

	NewPriceVolatility float64       `json:"newPxVol,omitempty"` // Modify options orders based on implied volatility, where 1 represents 100%. Only applicable to options.
	AttachAlgoOrders   []AlgoOrdInfo `json:"attachAlgoOrds,omitempty"`
}

// AlgoOrdInfo represents TP/SL info attached when placing an order
type AlgoOrdInfo struct {
	AttachAlgoID                   string  `json:"attachAlgoId,omitempty"`
	AttachAlgoClientOrderID        string  `json:"attachAlgoClOrdId,omitempty"`
	NewTakeProfitTriggerPrice      float64 `json:"newTpTriggerPx,omitempty"`
	NewTakeProfitOrderPrice        float64 `json:"newTpOrdPx,omitempty"`
	NewTakeProfitOrderKind         string  `json:"newTpOrdKind,omitempty"` // possible values are 'condition' and 'limit'
	NewStopLossTriggerPrice        float64 `json:"newSlTriggerPx,omitempty"`
	NewStopLossOrderPrice          float64 `json:"newSlOrdPx,omitempty"`
	NewTakkeProfitTriggerPriceType string  `json:"newTpTriggerPxType,omitempty"`
	NewStopLossTriggerPriceType    string  `json:"newSlTriggerPxType,omitempty"` // possible values are 'last', 'index', and 'mark'
	NewSize                        float64 `json:"sz,omitempty"`
	AmendPriceOnTriggerType        string  `json:"amendPxOnTriggerType,omitempty"` // Whether to enable Cost-price SL. Only applicable to SL order of split TPs. '0': disable, the default value '1': Enable
}

// ClosePositionsRequestParams input parameters for close position endpoints
type ClosePositionsRequestParams struct {
	InstrumentID          string `json:"instId"` // REQUIRED
	PositionSide          string `json:"posSide"`
	MarginMode            string `json:"mgnMode"` // cross or isolated
	Currency              string `json:"ccy"`
	AutomaticallyCanceled bool   `json:"autoCxl"`
	ClientID              string `json:"clOrdId,omitempty"`
	Tag                   string `json:"tag,omitempty"`
}

// ClosePositionResponse response data for close position
type ClosePositionResponse struct {
	InstrumentID string `json:"instId"`
	PositionSide string `json:"posSide"`
}

// OrderDetailRequestParam payload data to request order detail
type OrderDetailRequestParam struct {
	InstrumentID  string `json:"instId"`
	OrderID       string `json:"ordId"`
	ClientOrderID string `json:"clOrdId"`
}

// OrderDetail holds detailed information about an order.
type OrderDetail struct {
	InstrumentType             string       `json:"instType"`
	InstrumentID               string       `json:"instId"`
	Currency                   string       `json:"ccy"`
	OrderID                    string       `json:"ordId"`
	ClientOrderID              string       `json:"clOrdId"`
	Tag                        string       `json:"tag"`
	ProfitAndLoss              types.Number `json:"pnl"`
	OrderType                  string       `json:"ordType"`
	Side                       order.Side   `json:"side"`
	PositionSide               string       `json:"posSide"`
	TradeMode                  string       `json:"tdMode"`
	TradeID                    string       `json:"tradeId"`
	FillTime                   types.Time   `json:"fillTime"`
	Source                     string       `json:"source"`
	State                      string       `json:"state"`
	TakeProfitTriggerPriceType string       `json:"tpTriggerPxType"`
	StopLossTriggerPriceType   string       `json:"slTriggerPxType"`
	StopLossOrderPrice         types.Number `json:"slOrdPx"`
	RebateCurrency             string       `json:"rebateCcy"`
	QuantityType               string       `json:"tgtCcy"`   // base_ccy and quote_ccy
	Category                   string       `json:"category"` // normal, twap, adl, full_liquidation, partial_liquidation, delivery, ddh
	AccumulatedFillSize        types.Number `json:"accFillSz"`
	FillPrice                  types.Number `json:"fillPx"`
	FillSize                   types.Number `json:"fillSz"`
	RebateAmount               types.Number `json:"rebate"`
	FeeCurrency                string       `json:"feeCcy"`
	TransactionFee             types.Number `json:"fee"`
	AveragePrice               types.Number `json:"avgPx"`
	Leverage                   types.Number `json:"lever"`
	Price                      types.Number `json:"px"`
	Size                       types.Number `json:"sz"`
	TakeProfitTriggerPrice     types.Number `json:"tpTriggerPx"`
	TakeProfitOrderPrice       types.Number `json:"tpOrdPx"`
	StopLossTriggerPrice       types.Number `json:"slTriggerPx"`
	UpdateTime                 types.Time   `json:"uTime"`
	CreationTime               types.Time   `json:"cTime"`
	AlgoClOrdID                string       `json:"algoClOrdId"`
	AlgoID                     string       `json:"algoId"`
	AttachAlgoClOrdID          string       `json:"attachAlgoClOrdId"`
	AttachAlgoOrds             []any        `json:"attachAlgoOrds"`
	CancelSource               string       `json:"cancelSource"`
	CancelSourceReason         string       `json:"cancelSourceReason"`
	IsTakeProfitLimit          string       `json:"isTpLimit"`
	LinkedAlgoOrd              struct {
		AlgoID string `json:"algoId"`
	} `json:"linkedAlgoOrd"`
	PriceType               string       `json:"pxType"`
	PriceVolume             types.Number `json:"pxVol"`
	PriceUSD                types.Number `json:"pxUsd"`
	QuickMgnType            string       `json:"quickMgnType"`
	ReduceOnly              bool         `json:"reduceOnly,string,omitempty"`
	SelfTradePreventionID   string       `json:"stpId"`
	SelfTradePreventionMode string       `json:"stpMode"`
}

// OrderListRequestParams represents order list requesting parameters
type OrderListRequestParams struct {
	InstrumentType string    `json:"instType"` // SPOT , MARGIN, SWAP, FUTURES , OPTIONS
	Underlying     string    `json:"uly"`
	InstrumentID   string    `json:"instId"`
	OrderType      string    `json:"orderType"`
	State          string    `json:"state"`            // live, partially_filled
	Before         string    `json:"before,omitempty"` // used for order IDs
	After          string    `json:"after,omitempty"`  // used for order IDs
	Start          time.Time `json:"begin"`
	End            time.Time `json:"end"`
	Limit          int64     `json:"limit,omitempty"`
}

// OrderHistoryRequestParams holds parameters to request order data history of last 7 days
type OrderHistoryRequestParams struct {
	OrderListRequestParams
	Category string `json:"category"` // twap, adl, full_liquidation, partial_liquidation, delivery, ddh
}

// PendingOrderItem represents a pending order Item in pending orders list
type PendingOrderItem struct {
	AccumulatedFillSize        types.Number  `json:"accFillSz"`
	AveragePrice               types.Number  `json:"avgPx"`
	CreationTime               types.Time    `json:"cTime"`
	Category                   string        `json:"category"`
	Currency                   string        `json:"ccy"`
	ClientOrderID              string        `json:"clOrdId"`
	Fee                        types.Number  `json:"fee"`
	FeeCurrency                currency.Code `json:"feeCcy"`
	LastFilledPrice            types.Number  `json:"fillPx"`
	LastFilledSize             types.Number  `json:"fillSz"`
	FillTime                   types.Time    `json:"fillTime"`
	InstrumentID               string        `json:"instId"`
	InstrumentType             string        `json:"instType"`
	Leverage                   types.Number  `json:"lever"`
	OrderID                    string        `json:"ordId"`
	OrderType                  string        `json:"ordType"`
	ProfitAndLoss              types.Number  `json:"pnl"`
	PositionSide               string        `json:"posSide"`
	RebateAmount               types.Number  `json:"rebate"`
	RebateCurrency             string        `json:"rebateCcy"`
	Side                       order.Side    `json:"side"`
	StopLossOrdPrice           types.Number  `json:"slOrdPx"`
	StopLossTriggerPrice       types.Number  `json:"slTriggerPx"`
	StopLossTriggerPriceType   string        `json:"slTriggerPxType"`
	State                      string        `json:"state"`
	Price                      types.Number  `json:"px"`
	Size                       types.Number  `json:"sz"`
	Tag                        string        `json:"tag"`
	SizeType                   string        `json:"tgtCcy"`
	TradeMode                  string        `json:"tdMode"`
	Source                     string        `json:"source"`
	TakeProfitOrdPrice         types.Number  `json:"tpOrdPx"`
	TakeProfitTriggerPrice     types.Number  `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string        `json:"tpTriggerPxType"`
	TradeID                    string        `json:"tradeId"`
	UpdateTime                 types.Time    `json:"uTime"`
}

// TransactionDetailRequestParams retrieve recently-filled transaction details in the last 3 day
type TransactionDetailRequestParams struct {
	InstrumentType string    `json:"instType"` // SPOT , MARGIN, SWAP, FUTURES , option
	Underlying     string    `json:"uly"`
	InstrumentID   string    `json:"instId"`
	OrderID        string    `json:"ordId"`
	OrderType      string    `json:"orderType"`
	SubType        string    `json:"subType,omitempty"`
	After          string    `json:"after"`  // after billid
	Before         string    `json:"before"` // before billid
	Begin          time.Time `json:"begin"`
	End            time.Time `json:"end"`
	Limit          int64     `json:"limit"`
}

// FillArchiveParam transaction detail param for 2 year
type FillArchiveParam struct {
	Year    int64  `json:"year,string"`
	Quarter string `json:"quarter"`
}

// ArchiveReference holds recently-filled transaction details archive link and timestamp information
type ArchiveReference struct {
	FileHref  string     `json:"fileHref"`
	State     string     `json:"state"`
	Timestamp types.Time `json:"ts"`
}

// TransactionDetail holds recently-filled transaction detail data
type TransactionDetail struct {
	InstrumentType           string       `json:"instType"`
	InstrumentID             string       `json:"instId"`
	TradeID                  string       `json:"tradeId"`
	OrderID                  string       `json:"ordId"`
	ClientOrderID            string       `json:"clOrdId"`
	TransactionType          string       `json:"subType"`
	BillID                   string       `json:"billId"`
	Tag                      string       `json:"tag"`
	FillPrice                types.Number `json:"fillPx"`
	FillSize                 types.Number `json:"fillSz"`
	FillIndexPrice           types.Number `json:"fillIdxPx"`
	FillProfitAndLoss        types.Number `json:"fillPnl"`
	FillPriceVolatility      types.Number `json:"fillPxVol"`
	FillPriceUSD             types.Number `json:"fillPxUsd"`
	MarkVolatilityWhenFilled types.Number `json:"fillMarkVol"`
	ForwardPriceWhenFilled   types.Number `json:"fillFwdPx"`
	MarkPriceWhenFilled      types.Number `json:"fillMarkPx"`
	Side                     order.Side   `json:"side"`
	PositionSide             string       `json:"posSide"`
	ExecType                 string       `json:"execType"`
	FeeCurrency              string       `json:"feeCcy"`
	Fee                      types.Number `json:"fee"`
	FillTime                 types.Time   `json:"fillTime"`
	Timestamp                types.Time   `json:"ts"`
}

// AlgoOrderParams holds algo order information
type AlgoOrderParams struct {
	InstrumentID      string  `json:"instId"` // Required
	TradeMode         string  `json:"tdMode"` // Required
	Currency          string  `json:"ccy,omitempty"`
	Side              string  `json:"side"` // Required
	PositionSide      string  `json:"posSide,omitempty"`
	OrderType         string  `json:"ordType"`   // Required
	Size              float64 `json:"sz,string"` // Required
	ReduceOnly        bool    `json:"reduceOnly,omitempty"`
	OrderTag          string  `json:"tag,omitempty"`
	QuantityType      string  `json:"tgtCcy,omitempty"`
	AlgoClientOrderID string  `json:"algoClOrdId,omitempty"`

	// Place Stop Order params
	TakeProfitOrderPrice       float64 `json:"tpOrdPx,string,omitempty"`
	TakeProfitTriggerPrice     float64 `json:"tpTriggerPx,string,omitempty"`
	TakeProfitTriggerPriceType string  `json:"tpTriggerPxType,omitempty"`
	StopLossTriggerPrice       float64 `json:"slTriggerPx,string,omitempty"`
	StopLossOrderPrice         float64 `json:"slOrdPx,string,omitempty"`
	StopLossTriggerPriceType   string  `json:"slTriggerPxType,omitempty"`
	CancelOnClosePosition      bool    `json:"cxlOnClosePos,omitempty"`

	// For trigger and trailing stop order
	CallbackRatio          float64 `json:"callbackRatio,omitempty,string"`
	ActivePrice            float64 `json:"activePx,omitempty,string"`
	CallbackSpreadVariance float64 `json:"callbackSpread,omitempty,string"`

	// trigger algo orders params.
	// notice: Trigger orders are not available in the net mode of futures and perpetual swaps
	TriggerPrice     float64 `json:"triggerPx,string,omitempty"`
	OrderPrice       float64 `json:"orderPx,string,omitempty"` // if the price i -1, then the order will be executed on the market.
	TriggerPriceType string  `json:"triggerPxType,omitempty"`  // last, index, and mark

	PriceVariance float64 `json:"pxVar,omitempty,string"`    // Optional
	PriceSpread   float64 `json:"pxSpread,omitempty,string"` // Optional
	SizeLimit     float64 `json:"szLimit,string,omitempty"`  // Required
	LimitPrice    float64 `json:"pxLimit,string,omitempty"`  // Required

	// TWAPOrder
	TimeInterval kline.Interval `json:"interval,omitempty"` // Required

	// Chase order
	ChaseType     string  `json:"chaseType,omitempty"` // Possible values: "distance" and "ratio"
	ChaseValue    float64 `json:"chaseVal,omitempty,string"`
	MaxChaseType  string  `json:"maxChaseType,omitempty"`
	MaxChaseValue float64 `json:"maxChaseVal,omitempty,string"`
}

// AlgoOrder algo order requests response
type AlgoOrder struct {
	AlgoID            string `json:"algoId"`
	StatusCode        int64  `json:"sCode,string"`
	StatusMessage     string `json:"sMsg"`
	ClientOrderID     string `json:"clOrdId"`
	AlgoClientOrderID string `json:"algoClOrdId"`
	Tag               string `json:"tag"`
}

// AmendAlgoOrderParam request parameter to amend an algo order
type AmendAlgoOrderParam struct {
	InstrumentID              string  `json:"instId"`
	AlgoID                    string  `json:"algoId,omitempty"`
	ClientSuppliedAlgoOrderID string  `json:"algoClOrdId,omitempty"`
	CancelOrderWhenFail       bool    `json:"cxlOnFail,omitempty"` // Whether the order needs to be automatically canceled when the order amendment fails Valid options: false or true, the default is false.
	RequestID                 string  `json:"reqId,omitempty"`
	NewSize                   float64 `json:"newSz,omitempty,string"`

	// Take Profit Stop Loss Orders
	NewTakeProfitTriggerPrice     float64 `json:"newTpTriggerPx,omitempty,string"`
	NewTakeProfitOrderPrice       float64 `json:"newTpOrdPx,omitempty,string"`
	NewStopLossTriggerPrice       float64 `json:"newSlTriggerPx,omitempty,string"`
	NewStopLossOrderPrice         float64 `json:"newSlOrdPx,omitempty,string"`  // Stop-loss order price If the price is -1, stop-loss will be executed at the market price.
	NewTakeProfitTriggerPriceType string  `json:"newTpTriggerPxType,omitempty"` // Take-profit trigger price type'last': last price 'index': index price 'mark': mark price
	NewStopLossTriggerPriceType   string  `json:"newSlTriggerPxType,omitempty"` // Stop-loss trigger price type 'last': last price  'index': index price  'mark': mark price

	// Trigger Order parameters
	NewTriggerPrice     float64 `json:"newTriggerPx,omitempty"`
	NewOrderPrice       float64 `json:"newOrdPx,omitempty"`
	NewTriggerPriceType string  `json:"newTriggerPxType,omitempty"`

	AttachAlgoOrders []SubTPSLParams `json:"attachAlgoOrds,omitempty"`
}

// SubTPSLParams represents take-profit and stop-loss price parameters to be used by algo orders
type SubTPSLParams struct {
	NewTakeProfitTriggerPrice     float64 `json:"newTpTriggerPx,omitempty,string"`
	NewTakeProfitOrderPrice       float64 `json:"newTpOrdPx,omitempty,string"`
	NewStopLossTriggerPrice       float64 `json:"newSlTriggerPx,omitempty,string"`
	NewStopLossOrderPrice         float64 `json:"newSlOrdPx,omitempty,string"`  // Stop-loss order price If the price is -1, stop-loss will be executed at the market price.
	NewTakeProfitTriggerPriceType string  `json:"newTpTriggerPxType,omitempty"` // Take-profit trigger price type'last': last price 'index': index price 'mark': mark price
	NewStopLossTriggerPriceType   string  `json:"newSlTriggerPxType,omitempty"` // Stop-loss trigger price type 'last': last price  'index': index price  'mark': mark price
}

// AmendAlgoResponse holds response information of amending an algo order
type AmendAlgoResponse struct {
	AlgoClientOrderID string `json:"algoClOrdId"`
	AlgoID            string `json:"algoId"`
	ReqID             string `json:"reqId"`
	StatusCode        string `json:"sCode"`
	StatusMessage     string `json:"sMsg"`
}

// AlgoOrderDetail represents an algo order detail
type AlgoOrderDetail struct {
	InstrumentType          string       `json:"instType"`
	InstrumentID            string       `json:"instId"`
	OrderID                 string       `json:"ordId"`
	OrderIDList             []string     `json:"ordIdList"`
	Currency                string       `json:"ccy"`
	ClientOrderID           string       `json:"clOrdId"`
	AlgoID                  string       `json:"algoId"`
	AttachAlgoOrds          []string     `json:"attachAlgoOrds"`
	Size                    types.Number `json:"sz"`
	CloseFraction           string       `json:"closeFraction"`
	OrderType               string       `json:"ordType"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"posSide"`
	TradeMode               string       `json:"tdMode"`
	TargetCurrency          string       `json:"tgtCcy"`
	State                   string       `json:"state"`
	Leverage                types.Number `json:"lever"`
	TpTriggerPrice          types.Number `json:"tpTriggerPx"`
	TpTriggerPriceType      string       `json:"tpTriggerPxType"`
	TpOrdPrice              types.Number `json:"tpOrdPx"`
	SlTriggerPrice          types.Number `json:"slTriggerPx"`
	SlTriggerPriceType      string       `json:"slTriggerPxType"`
	TriggerPrice            types.Number `json:"triggerPx"`
	TriggerPriceType        string       `json:"triggerPxType"`
	OrderPrice              types.Number `json:"ordPx"`
	ActualSize              types.Number `json:"actualSz"`
	ActualPrice             types.Number `json:"actualPx"`
	ActualSide              string       `json:"actualSide"`
	PriceVar                string       `json:"pxVar"`
	PriceSpread             types.Number `json:"pxSpread"`
	PriceLimit              types.Number `json:"pxLimit"`
	SizeLimit               types.Number `json:"szLimit"`
	Tag                     string       `json:"tag"`
	TimeInterval            string       `json:"timeInterval"`
	CallbackRatio           types.Number `json:"callbackRatio"`
	CallbackSpread          string       `json:"callbackSpread"`
	ActivePrice             types.Number `json:"activePx"`
	MoveTriggerPrice        types.Number `json:"moveTriggerPx"`
	ReduceOnly              string       `json:"reduceOnly"`
	TriggerTime             types.Time   `json:"triggerTime"`
	Last                    types.Number `json:"last"` // Last filled price while placing
	FailCode                string       `json:"failCode"`
	AlgoClOrdID             string       `json:"algoClOrdId"`
	AmendPriceOnTriggerType string       `json:"amendPxOnTriggerType"`
	CreationTime            types.Time   `json:"cTime"`
}

// AlgoOrderCancelParams algo order request parameter
type AlgoOrderCancelParams struct {
	AlgoOrderID  string `json:"algoId"`
	InstrumentID string `json:"instId"`
}

// AlgoOrderResponse holds algo order information
type AlgoOrderResponse struct {
	InstrumentType             string       `json:"instType"`
	InstrumentID               string       `json:"instId"`
	OrderID                    string       `json:"ordId"`
	Currency                   string       `json:"ccy"`
	AlgoOrderID                string       `json:"algoId"`
	Quantity                   types.Number `json:"sz"`
	OrderType                  string       `json:"ordType"`
	Side                       order.Side   `json:"side"`
	PositionSide               string       `json:"posSide"`
	TradeMode                  string       `json:"tdMode"`
	QuantityType               string       `json:"tgtCcy"`
	State                      string       `json:"state"`
	Lever                      types.Number `json:"lever"`
	TakeProfitTriggerPrice     types.Number `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string       `json:"tpTriggerPxType"`
	TakeProfitOrdPrice         types.Number `json:"tpOrdPx"`
	StopLossTriggerPriceType   string       `json:"slTriggerPxType"`
	StopLossTriggerPrice       types.Number `json:"slTriggerPx"`
	TriggerPrice               types.Number `json:"triggerPx"`
	TriggerPriceType           string       `json:"triggerPxType"`
	OrderPrice                 types.Number `json:"ordPx"`
	ActualSize                 types.Number `json:"actualSz"`
	ActualPrice                types.Number `json:"actualPx"`
	ActualSide                 string       `json:"actualSide"`
	PriceVar                   types.Number `json:"pxVar"`
	PriceSpread                types.Number `json:"pxSpread"`
	PriceLimit                 types.Number `json:"pxLimit"`
	SizeLimit                  types.Number `json:"szLimit"`
	TimeInterval               string       `json:"timeInterval"`
	TriggerTime                types.Time   `json:"triggerTime"`
	CallbackRatio              types.Number `json:"callbackRatio"`
	CallbackSpread             string       `json:"callbackSpread"`
	ActivePrice                types.Number `json:"activePx"`
	MoveTriggerPrice           types.Number `json:"moveTriggerPx"`
	CreationTime               types.Time   `json:"cTime"`
}

// CurrencyResponse represents a currency item detail response data
type CurrencyResponse struct {
	Name                string       `json:"name"`        // Chinese name of currency
	CanDeposit          bool         `json:"canDep"`      // Availability to deposit from chain. false: not available true: available
	CanInternalTransfer bool         `json:"canInternal"` // Availability to internal transfer.
	CanWithdraw         bool         `json:"canWd"`       // Availability to withdraw to chain.
	Currency            string       `json:"ccy"`         //
	Chain               string       `json:"chain"`       //
	LogoLink            string       `json:"logoLink"`    // Logo link of currency
	MainNet             bool         `json:"mainNet"`     // If current chain is main net then return true, otherwise return false
	MaxFee              types.Number `json:"maxFee"`      // Minimum withdrawal fee
	MaxWithdrawal       types.Number `json:"maxWd"`       // Minimum amount of currency withdrawal in a single transaction
	MinFee              types.Number `json:"minFee"`      // Minimum withdrawal fee
	MinWithdrawal       types.Number `json:"minWd"`       // Minimum amount of currency withdrawal in a single transaction
	UsedWithdrawalQuota types.Number `json:"usedWdQuota"` // Amount of currency withdrawal used in the past 24 hours, unit in BTC
	WithdrawalQuota     types.Number `json:"wdQuota"`     // Minimum amount of currency withdrawal in a single transaction
	WithdrawalTickSize  types.Number `json:"wdTickSz"`    // Withdrawal precision, indicating the number of digits after the decimal point
}

// AssetBalance represents account owner asset balance
type AssetBalance struct {
	Currency      string       `json:"ccy"`
	AvailBal      types.Number `json:"availBal"`
	Balance       types.Number `json:"bal"`
	FrozenBalance types.Number `json:"frozenBal"`
}

// NonTradableAsset holds non-tradable asset detail
type NonTradableAsset struct {
	Balance          types.Number  `json:"bal"`
	CanWithdraw      bool          `json:"canWd"`
	Currency         string        `json:"ccy"`
	Chain            string        `json:"chain"`
	CtAddr           string        `json:"ctAddr"`
	LogoLink         string        `json:"logoLink"`
	Name             string        `json:"name"`
	NeedTag          bool          `json:"needTag"`
	WithdrawAll      bool          `json:"wdAll"`
	FeeCurrency      currency.Code `json:"feeCcy"`
	Fee              types.Number  `json:"fee"`
	MinWithdrawal    types.Number  `json:"minWd"`
	WithdrawTickSize types.Number  `json:"wdTickSz"`
	BurningFeeRate   types.Number  `json:"burningFeeRate"`
}

// AccountAssetValuation represents view account asset valuation data
type AccountAssetValuation struct {
	Details struct {
		Classic types.Number `json:"classic"`
		Earn    types.Number `json:"earn"`
		Funding types.Number `json:"funding"`
		Trading types.Number `json:"trading"`
	} `json:"details"`
	TotalBalance types.Number `json:"totalBal"`
	Timestamp    types.Time   `json:"ts"`
}

// FundingTransferRequestInput represents funding account request input
type FundingTransferRequestInput struct {
	Currency               currency.Code `json:"ccy"`
	TransferType           int64         `json:"type,string"`
	Amount                 float64       `json:"amt,string"`
	RemittingAccountType   string        `json:"from"` // "6": Funding account, "18": Trading account
	BeneficiaryAccountType string        `json:"to"`
	SubAccount             string        `json:"subAcct"`
	LoanTransfer           bool          `json:"loanTrans,string"`
	OmitPositionRisk       bool          `json:"omitPosRisk,omitempty,string"`
	ClientID               string        `json:"clientId"` // Client-supplied ID A combination of case-sensitive alphanumerics, all numbers, or all letters of up to 32 characters.
}

// FundingTransferResponse represents funding transfer and trading account transfer response
type FundingTransferResponse struct {
	TransferID string       `json:"transId"`
	Currency   string       `json:"ccy"`
	ClientID   string       `json:"clientId"`
	From       types.Number `json:"from"`
	Amount     types.Number `json:"amt"`
	To         types.Number `json:"to"`
}

// TransferFundRateResponse represents funding transfer rate response
type TransferFundRateResponse struct {
	Amount         types.Number `json:"amt"`
	Currency       string       `json:"ccy"`
	ClientID       string       `json:"clientId"`
	From           string       `json:"from"`
	InstrumentID   string       `json:"instId"`
	State          string       `json:"state"`
	SubAccount     string       `json:"subAcct"`
	To             string       `json:"to"`
	ToInstrumentID string       `json:"toInstId"`
	TransferID     string       `json:"transId"`
	Type           types.Number `json:"type"`
}

// AssetBillDetail represents  the billing record
type AssetBillDetail struct {
	BillID         string       `json:"billId"`
	Currency       string       `json:"ccy"`
	ClientID       string       `json:"clientId"`
	BalanceChange  types.Number `json:"balChg"`
	AccountBalance types.Number `json:"bal"`
	Type           types.Number `json:"type"`
	Timestamp      types.Time   `json:"ts"`
}

// LightningDepositItem for creating an invoice
type LightningDepositItem struct {
	CreationTime types.Time `json:"cTime"`
	Invoice      string     `json:"invoice"`
}

// CurrencyDepositResponseItem represents the deposit address information item
type CurrencyDepositResponseItem struct {
	Tag                      string            `json:"tag"`
	Chain                    string            `json:"chain"`
	ContractAddress          string            `json:"ctAddr"`
	Currency                 string            `json:"ccy"`
	ToBeneficiaryAccount     string            `json:"to"`
	Address                  string            `json:"addr"`
	Selected                 bool              `json:"selected"`
	Memo                     string            `json:"memo"`
	DepositAddressAttachment map[string]string `json:"addrEx"`
	PaymentID                string            `json:"pmtId"`
	VerifiedName             string            `json:"verifiedName"`
}

// DepositHistoryResponseItem deposit history response item
type DepositHistoryResponseItem struct {
	Amount              types.Number `json:"amt"`
	TransactionID       string       `json:"txId"` // Hash record of the deposit
	Currency            string       `json:"ccy"`
	Chain               string       `json:"chain"`
	From                string       `json:"from"`
	ToDepositAddress    string       `json:"to"`
	Timestamp           types.Time   `json:"ts"`
	State               types.Number `json:"state"`
	DepositID           string       `json:"depId"`
	AreaCodeFrom        string       `json:"areaCodeFrom"`
	FromWithdrawalID    string       `json:"fromWdId"`
	ActualDepBlkConfirm string       `json:"actualDepBlkConfirm"`
}

// WithdrawalInput represents request parameters for cryptocurrency withdrawal
type WithdrawalInput struct {
	Currency              currency.Code                   `json:"ccy"`
	Amount                float64                         `json:"amt,string"`
	TransactionFee        float64                         `json:"fee,string"`
	WithdrawalDestination string                          `json:"dest"`
	ChainName             string                          `json:"chain"`
	ToAddress             string                          `json:"toAddr"`
	ClientID              string                          `json:"clientId"`
	AreaCode              string                          `json:"areaCode,omitempty"`
	RecipientInformation  *WithdrawalRecipientInformation `json:"rcvrInfo,omitempty"`
}

// WithdrawalRecipientInformation represents a recipient information for withdrawal
type WithdrawalRecipientInformation struct {
	WalletType                 string `json:"walletType,omitempty"`
	ExchangeID                 string `json:"exchId,omitempty"`
	ReceiverFirstName          string `json:"rcvrFirstName,omitempty"`
	ReceiverLastName           string `json:"rcvrLastName,omitempty"`
	ReceiverCountry            string `json:"rcvrCountry,omitempty"`
	ReceiverCountrySubDivision string `json:"rcvrCountrySubDivision,omitempty"`
	ReceiverTownName           string `json:"rcvrTownName,omitempty"`
	ReceiverStreetName         string `json:"rcvrStreetName,omitempty"`
}

// WithdrawalResponse cryptocurrency withdrawal response
type WithdrawalResponse struct {
	Amount       types.Number `json:"amt"`
	WithdrawalID string       `json:"wdId"`
	Currency     string       `json:"ccy"`
	ClientID     string       `json:"clientId"`
	Chain        string       `json:"chain"`
}

// LightningWithdrawalRequestInput to request Lightning Withdrawal requests
type LightningWithdrawalRequestInput struct {
	Currency currency.Code `json:"ccy"`     // REQUIRED Token symbol. Currently only BTC is supported.
	Invoice  string        `json:"invoice"` // REQUIRED Invoice text
	Memo     string        `json:"memo"`    // Lightning withdrawal memo
}

// LightningWithdrawalResponse response item for holding lightning withdrawal requests
type LightningWithdrawalResponse struct {
	WithdrawalID string     `json:"wdId"`
	CreationTime types.Time `json:"cTime"`
}

// WithdrawalHistoryResponse represents the withdrawal response history
type WithdrawalHistoryResponse struct {
	Currency             string       `json:"ccy"`
	ChainName            string       `json:"chain"`
	NonTradableAsset     bool         `json:"nonTradableAsset"`
	Amount               types.Number `json:"amt"`
	Timestamp            types.Time   `json:"ts"`
	FromRemittingAddress string       `json:"from"`
	ToReceivingAddress   string       `json:"to"`
	AreaCodeFrom         string       `json:"areaCodeFrom"`
	AreaCodeTo           string       `json:"areaCodeTo"`
	Tag                  string       `json:"tag"`
	WithdrawalFee        types.Number `json:"fee"`
	FeeCurrency          string       `json:"feeCcy"`
	Memo                 string       `json:"memo"`
	AddrEx               string       `json:"addrEx"`
	ClientID             string       `json:"clientId"`
	TransactionID        string       `json:"txId"` // Hash record of the withdrawal. This parameter will not be returned for internal transfers.
	StateOfWithdrawal    string       `json:"state"`
	WithdrawalID         string       `json:"wdId"`
	PaymentID            string       `json:"pmtId"`
}

// DepositWithdrawStatus holds deposit withdraw status info
type DepositWithdrawStatus struct {
	WithdrawID      string     `json:"wdId"`
	TransactionID   string     `json:"txId"`
	State           string     `json:"state"`
	EstCompleteTime types.Time `json:"estCompleteTime"`
}

// ExchangeInfo represents exchange information
type ExchangeInfo struct {
	ExchID       string `json:"exchId"`
	ExchangeName string `json:"exchName"`
}

// SmallAssetConvertResponse represents a response of converting a small asset to OKB
type SmallAssetConvertResponse struct {
	Details []struct {
		Amount        types.Number `json:"amt"`    // Quantity of currency assets before conversion
		Currency      string       `json:"ccy"`    //
		ConvertAmount types.Number `json:"cnvAmt"` // Quantity of OKB after conversion
		ConversionFee types.Number `json:"fee"`    // Fee for conversion, unit in OKB
	} `json:"details"`
	TotalConvertAmount types.Number `json:"totalCnvAmt"` // Total quantity of OKB after conversion
}

// SavingBalanceResponse holds the response data for a savings balance.
type SavingBalanceResponse struct {
	Currency      string       `json:"ccy"`
	Earnings      types.Number `json:"earnings"`
	RedemptAmount types.Number `json:"redemptAmt"`
	Rate          types.Number `json:"rate"`
	Amount        types.Number `json:"amt"`
	LoanAmount    types.Number `json:"loanAmt"`
	PendingAmount types.Number `json:"pendingAmt"`
}

// SavingsPurchaseRedemptionInput input json to SavingPurchase Post method
type SavingsPurchaseRedemptionInput struct {
	Currency   currency.Code `json:"ccy"`         // REQUIRED:
	Amount     float64       `json:"amt,string"`  // REQUIRED: purchase or redemption amount
	ActionType string        `json:"side"`        // REQUIRED: action type 'purchase' or 'redemption'
	Rate       float64       `json:"rate,string"` // REQUIRED:
}

// SavingsPurchaseRedemptionResponse formats the JSON response for the SavingPurchase or SavingRedemption POST methods
type SavingsPurchaseRedemptionResponse struct {
	Currency   string       `json:"ccy"`
	ActionType string       `json:"side"`
	Account    string       `json:"acct"` // '6': Funding account '18': Trading account
	Amount     types.Number `json:"amt"`
	Rate       types.Number `json:"rate"`
}

// LendingRate represents the response containing the lending rate.
type LendingRate struct {
	Currency currency.Code `json:"ccy"`
	Rate     types.Number  `json:"rate"`
}

// LendingHistory holds lending history responses
type LendingHistory struct {
	Currency  string       `json:"ccy"`
	Amount    types.Number `json:"amt"`
	Earnings  types.Number `json:"earnings"`
	Rate      types.Number `json:"rate"`
	Timestamp types.Time   `json:"ts"`
}

// PublicBorrowInfo holds a currency's borrow info
type PublicBorrowInfo struct {
	Currency         string       `json:"ccy"`
	AverageAmount    types.Number `json:"avgAmt"`
	AverageAmountUSD types.Number `json:"avgAmtUsd"`
	AverageRate      types.Number `json:"avgRate"`
	PreviousRate     types.Number `json:"preRate"`
	EstimatedRate    types.Number `json:"estRate"`
}

// PublicBorrowHistory holds a currencies borrow history
type PublicBorrowHistory struct {
	Amount    types.Number `json:"amt"`
	Currency  string       `json:"ccy"`
	Rate      types.Number `json:"rate"`
	Timestamp types.Time   `json:"ts"`
}

// ConvertCurrency represents currency conversion detailed data
type ConvertCurrency struct {
	Currency string       `json:"currency"`
	Min      types.Number `json:"min"`
	Max      types.Number `json:"max"`
}

// ConvertCurrencyPair holds information related to conversion between two pairs
type ConvertCurrencyPair struct {
	InstrumentID     string       `json:"instId"`
	BaseCurrency     string       `json:"baseCcy"`
	BaseCurrencyMax  types.Number `json:"baseCcyMax"`
	BaseCurrencyMin  types.Number `json:"baseCcyMin"`
	QuoteCurrency    string       `json:"quoteCcy,omitempty"`
	QuoteCurrencyMax types.Number `json:"quoteCcyMax"`
	QuoteCurrencyMin types.Number `json:"quoteCcyMin"`
}

// EstimateQuoteRequestInput represents estimate quote request parameters
type EstimateQuoteRequestInput struct {
	BaseCurrency         currency.Code `json:"baseCcy,omitzero"`
	QuoteCurrency        currency.Code `json:"quoteCcy,omitzero"`
	Side                 string        `json:"side,omitempty"`
	RFQAmount            float64       `json:"rfqSz,omitempty"`
	RFQSzCurrency        string        `json:"rfqSzCcy,omitempty"`
	ClientRequestOrderID string        `json:"clQReqId,string,omitempty"`
	Tag                  string        `json:"tag,omitempty"`
}

// EstimateQuoteResponse represents estimate quote response data
type EstimateQuoteResponse struct {
	BaseCurrency    string       `json:"baseCcy"`
	BaseSize        types.Number `json:"baseSz"`
	ClientRequestID string       `json:"clQReqId"`
	ConvertPrice    types.Number `json:"cnvtPx"`
	OrigRFQSize     types.Number `json:"origRfqSz"`
	QuoteCurrency   string       `json:"quoteCcy"`
	QuoteID         string       `json:"quoteId"`
	QuoteSize       types.Number `json:"quoteSz"`
	QuoteTime       types.Time   `json:"quoteTime"`
	RFQSize         types.Number `json:"rfqSz"`
	RFQSizeCurrency string       `json:"rfqSzCcy"`
	Side            order.Side   `json:"side"`
	TTLMs           string       `json:"ttlMs"` // Validity period of quotation in milliseconds
}

// ConvertTradeInput represents convert trade request input
type ConvertTradeInput struct {
	BaseCurrency  string        `json:"baseCcy"`
	QuoteCurrency string        `json:"quoteCcy"`
	Side          string        `json:"side"`
	Size          float64       `json:"sz,string"`
	SizeCurrency  currency.Code `json:"szCcy"`
	QuoteID       string        `json:"quoteId"`
	ClientOrderID string        `json:"clTReqId,omitempty"`
	Tag           string        `json:"tag,omitempty"`
}

// ConvertTradeResponse represents convert trade response
type ConvertTradeResponse struct {
	BaseCurrency  string       `json:"baseCcy"`
	ClientOrderID string       `json:"clTReqId"`
	FillBaseSize  types.Number `json:"fillBaseSz"`
	FillPrice     types.Number `json:"fillPx"`
	FillQuoteSize types.Number `json:"fillQuoteSz"`
	InstrumentID  string       `json:"instId"`
	QuoteCurrency string       `json:"quoteCcy"`
	QuoteID       string       `json:"quoteId"`
	Side          order.Side   `json:"side"`
	State         string       `json:"state"`
	TradeID       string       `json:"tradeId"`
	Size          types.Number `json:"sz"`
	SizeCurrency  string       `json:"szCcy"`
	Timestamp     types.Time   `json:"ts"`
}

// ConvertHistory holds convert trade history response
type ConvertHistory struct {
	InstrumentID    string       `json:"instId"`
	Side            order.Side   `json:"side"`
	FillPrice       types.Number `json:"fillPx"`
	BaseCurrency    string       `json:"baseCcy"`
	QuoteCurrency   string       `json:"quoteCcy"`
	FillBaseSize    types.Number `json:"fillBaseSz"`
	State           string       `json:"state"`
	TradeID         string       `json:"tradeId"`
	FillQuoteSize   types.Number `json:"fillQuoteSz"`
	ClientRequestID string       `json:"clTReqId"`
	Timestamp       types.Time   `json:"ts"`
}

// Account holds currency account balance and related information
type Account struct {
	AdjustedEquity               types.Number    `json:"adjEq"`
	Details                      []AccountDetail `json:"details"`
	InitialMarginRequirement     types.Number    `json:"imr"` // Frozen equity for open positions and pending orders in USD level Applicable to Multi-currency margin and Portfolio margin
	IsolatedMarginEquity         types.Number    `json:"isoEq"`
	MgnRatio                     types.Number    `json:"mgnRatio"`
	MaintenanceMarginRequirement types.Number    `json:"mmr"` // Maintenance margin requirement in USD level Applicable to Multi-currency margin and Portfolio margin
	BorrowFrozen                 string          `json:"borrowFroz"`
	NotionalUsd                  types.Number    `json:"notionalUsd"`
	OrdFroz                      types.Number    `json:"ordFroz"` // Margin frozen for pending orders in USD level Applicable to Multi-currency margin and Portfolio margin
	TotalEquity                  types.Number    `json:"totalEq"` // Total Equity in USD level
	UpdateTime                   types.Time      `json:"uTime"`   // UpdateTime
}

// AccountDetail account detail information
type AccountDetail struct {
	Currency                  currency.Code `json:"ccy"`
	EquityOfCurrency          types.Number  `json:"eq"`
	CashBalance               types.Number  `json:"cashBal"` // Cash Balance
	UpdateTime                types.Time    `json:"uTime"`
	IsoEquity                 types.Number  `json:"isoEq"`
	AvailableEquity           types.Number  `json:"availEq"`
	DiscountEquity            types.Number  `json:"disEq"`
	FixedBalance              types.Number  `json:"fixedBal"`
	AvailableBalance          types.Number  `json:"availBal"`
	MarginFrozenForOpenOrders types.Number  `json:"ordFrozen"`

	CrossLiab             types.Number `json:"crossLiab"`
	EquityUsd             types.Number `json:"eqUsd"`
	FrozenBalance         types.Number `json:"frozenBal"`
	Interest              types.Number `json:"interest"`
	IsolatedLiabilities   types.Number `json:"isoLiab"`
	IsoUpl                types.Number `json:"isoUpl"` // Isolated unrealized profit and loss of the currency applicable to Single-currency margin and Multi-currency margin and Portfolio margin
	LiabilitiesOfCurrency types.Number `json:"liab"`
	MaxLoan               types.Number `json:"maxLoan"`
	MarginRatio           types.Number `json:"mgnRatio"`      // Equity of the currency
	NotionalLever         types.Number `json:"notionalLever"` // Leverage of the currency applicable to Single-currency margin
	Twap                  types.Number `json:"twap"`
	UPL                   types.Number `json:"upl"` // unrealized profit & loss of all margin and derivatives positions of currency.
	UPLLiabilities        types.Number `json:"uplLiab"`
	StrategyEquity        types.Number `json:"stgyEq"`  // strategy equity
	TotalEquity           types.Number `json:"totalEq"` // Total equity in USD level. Appears unused
	RewardBalance         types.Number `json:"rewardBal"`
	InitialMarginRate     types.Number `json:"imr"`
	MMR                   types.Number `json:"mmr"` // ross maintenance margin requirement at the currency level. Applicable to Spot and futures mode and when there is cross position
	SpotInUseAmount       types.Number `json:"spotInUseAmt"`
	ClientSpotInUseAmount types.Number `json:"clSpotInUseAmt"`
	MaxSpotInUseAmount    types.Number `json:"maxSpotInUse"`
	SpotIsolatedBalance   types.Number `json:"spotIsoBal"`
	SmarkSyncEquity       types.Number `json:"smtSyncEq"`
	SpotCopyTradingEquity types.Number `json:"spotCopyTradingEq"`
	SpotBalance           types.Number `json:"spotBal"`
	OpenAvgPrice          types.Number `json:"openAvgPx"`
	AccAvgPrice           types.Number `json:"accAvgPx"`
	SpotUPL               types.Number `json:"spotUpl"`
	SpotUplRatio          types.Number `json:"spotUplRatio"`
	TotalPNL              types.Number `json:"totalPnl"`
	TotalPNLRatio         types.Number `json:"totalPnlRatio"`
}

// AccountPosition account position
type AccountPosition struct {
	AutoDeleveraging             string        `json:"adl"`      // Auto-deleveraging (ADL) indicator Divided into 5 levels, from 1 to 5, the smaller the number, the weaker the adl intensity.
	AvailablePosition            string        `json:"availPos"` // Position that can be closed Only applicable to MARGIN, FUTURES/SWAP in the long-short mode, OPTION in Simple and isolated OPTION in margin Account.
	AveragePrice                 types.Number  `json:"avgPx"`
	CreationTime                 types.Time    `json:"cTime"`
	Currency                     currency.Code `json:"ccy"`
	DeltaBS                      string        `json:"deltaBS"` // delta：Black-Scholes Greeks in dollars,only applicable to OPTION
	DeltaPA                      string        `json:"deltaPA"` // delta：Greeks in coins,only applicable to OPTION
	GammaBS                      string        `json:"gammaBS"` // gamma：Black-Scholes Greeks in dollars,only applicable to OPTION
	GammaPA                      string        `json:"gammaPA"` // gamma：Greeks in coins,only applicable to OPTION
	InitialMarginRequirement     types.Number  `json:"imr"`     // Initial margin requirement, only applicable to cross.
	InstrumentID                 string        `json:"instId"`
	InstrumentType               asset.Item    `json:"instType"`
	Interest                     types.Number  `json:"interest"`
	USDPrice                     types.Number  `json:"usdPx"`
	LastTradePrice               types.Number  `json:"last"`
	Leverage                     types.Number  `json:"lever"`   // Leverage, not applicable to OPTION seller
	Liabilities                  types.Number  `json:"liab"`    // Liabilities, only applicable to MARGIN.
	LiabilitiesCurrency          string        `json:"liabCcy"` // Liabilities currency, only applicable to MARGIN.
	LiquidationPrice             types.Number  `json:"liqPx"`   // Estimated liquidation price Not applicable to OPTION
	MarkPrice                    types.Number  `json:"markPx"`
	Margin                       types.Number  `json:"margin"`
	MarginMode                   string        `json:"mgnMode"`
	MarginRatio                  types.Number  `json:"mgnRatio"`
	MaintenanceMarginRequirement types.Number  `json:"mmr"`         // Maintenance margin requirement in USD level Applicable to Multi-currency margin and Portfolio margin
	NotionalUsd                  types.Number  `json:"notionalUsd"` // Quality of Positions -- usd
	OptionValue                  types.Number  `json:"optVal"`      // Option Value, only application to position.
	QuantityOfPosition           types.Number  `json:"pos"`         // Quantity of positions,In the mode of autonomous transfer from position to position, after the deposit is transferred, a position with pos of 0 will be generated
	PositionCurrency             string        `json:"posCcy"`
	PositionID                   string        `json:"posId"`
	PositionSide                 string        `json:"posSide"`
	ThetaBS                      string        `json:"thetaBS"` // theta：Black-Scholes Greeks in dollars,only applicable to OPTION
	ThetaPA                      string        `json:"thetaPA"` // theta：Greeks in coins,only applicable to OPTION
	TradeID                      string        `json:"tradeId"`
	UpdatedTime                  types.Time    `json:"uTime"`    // Latest time position was adjusted,
	UPNL                         types.Number  `json:"upl"`      // Unrealized profit and loss
	UPLRatio                     types.Number  `json:"uplRatio"` // Unrealized profit and loss ratio
	VegaBS                       string        `json:"vegaBS"`   // vega：Black-Scholes Greeks in dollars,only applicable to OPTION
	VegaPA                       string        `json:"vegaPA"`   // vega：Greeks in coins,only applicable to OPTION

	// PushTime added feature in the websocket push data.

	PushTime types.Time `json:"pTime"` // The time when the account position data is pushed.
}

// AccountPositionHistory hold account position history
type AccountPositionHistory struct {
	InstrumentType    string       `json:"instType"`
	InstrumentID      string       `json:"instId"`
	ManagementMode    string       `json:"mgnMode"`
	Type              string       `json:"type"`
	CreationTime      types.Time   `json:"cTime"`
	UpdateTime        types.Time   `json:"uTime"`
	OpenAveragePrice  string       `json:"openAvgPx"`
	CloseAveragePrice types.Number `json:"closeAvgPx"`

	Positions                types.Number `json:"pos"`
	BaseBalance              types.Number `json:"baseBal"`
	QuoteBalance             types.Number `json:"quoteBal"`
	BaseBorrowed             types.Number `json:"baseBorrowed"`
	BaseInterest             types.Number `json:"baseInterest"`
	QuoteBorrowed            types.Number `json:"quoteBorrowed"`
	QuoteInterest            types.Number `json:"quoteInterest"`
	PositionCurrency         string       `json:"posCcy"`
	AvailablePositions       string       `json:"availPos"`
	AveragePrice             types.Number `json:"avgPx"`
	MarkPrice                types.Number `json:"markPx"`
	UPL                      types.Number `json:"upl"`
	UPLRatio                 types.Number `json:"uplRatio"`
	UPLLastPrice             types.Number `json:"uplLastPx"`
	UPLRatioLastPrice        types.Number `json:"uplRatioLastPx"`
	Leverage                 string       `json:"lever"`
	LiquidiationPrice        types.Number `json:"liqPx"`
	InitialMarginRequirement types.Number `json:"imr"`
	Margin                   string       `json:"margin"`
	MarginRatio              types.Number `json:"mgnRatio"`
	MMR                      types.Number `json:"mmr"`
	Liabilities              types.Number `json:"liab"`
	LiabilitiesCurrency      string       `json:"liabCcy"`
	Interest                 types.Number `json:"interest"`
	TradeID                  string       `json:"tradeId"`
	OptionValue              string       `json:"optVal"`
	PendingCloseOrdLiabVal   types.Number `json:"pendingCloseOrdLiabVal"`
	NotionalUSD              string       `json:"notionalUsd"`
	ADL                      string       `json:"adl"`
	LastTradedPrice          types.Number `json:"last"`
	IndexPrice               types.Number `json:"idxPx"`
	USDPrice                 string       `json:"usdPx"`
	BreakevenPrice           string       `json:"bePx"`
	DeltaBS                  string       `json:"deltaBS"`
	DeltaPA                  string       `json:"deltaPA"`
	GammaBS                  string       `json:"gammaBS"`
	ThetaBS                  string       `json:"thetaBS"`
	ThetaPA                  string       `json:"thetaPA"`
	VegaBS                   string       `json:"vegaBS"`
	VegaPA                   string       `json:"vegaPA"`
	SpotInUseAmount          types.Number `json:"spotInUseAmt"`
	SpotInUseCurrency        string       `json:"spotInUseCcy"`
	ClientSpotInUseAmount    types.Number `json:"clSpotInUseAmt"`
	BizRefID                 string       `json:"bizRefId"`
	BizRefType               string       `json:"bizRefType"`
	ProfitAndLoss            types.Number `json:"pnl"`
	Fee                      types.Number `json:"fee"`
	LiqPenalty               types.Number `json:"liqPenalty"`
	CloseOrderAlgo           types.Number `json:"closeOrderAlgo"`

	Currency           string       `json:"ccy"`
	CloseTotalPosition types.Number `json:"closeTotalPos"`
	OpenMaxPosition    types.Number `json:"openMaxPos"`
	ProfitAndLossRatio types.Number `json:"pnlRatio"`
	PositionID         string       `json:"posId"`
	PositionSide       string       `json:"posSide"`
	TriggerPrice       types.Number `json:"triggerPx"`
	Underlying         string       `json:"uly"`
}

// AccountBalanceData represents currency account balance
type AccountBalanceData struct {
	Currency       string       `json:"ccy"`
	DiscountEquity types.Number `json:"disEq"` // discount equity of the currency in USD level.
	Equity         types.Number `json:"eq"`    // Equity of the currency
}

// PositionData holds account position data
type PositionData struct {
	BaseBalance        types.Number `json:"baseBal"`
	Currency           string       `json:"ccy"`
	InstrumentID       string       `json:"instId"`
	InstrumentType     string       `json:"instType"`
	ManagementMode     string       `json:"mgnMode"`
	NotionalCurrency   string       `json:"notionalCcy"`
	NotionalUSD        types.Number `json:"notionalUsd"`
	Position           string       `json:"pos"`
	PositionedCurrency string       `json:"posCcy"`
	PositionedID       string       `json:"posId"`
	PositionedSide     string       `json:"posSide"`
	QuoteBalance       types.Number `json:"quoteBal"`
}

// AccountAndPositionRisk holds information
type AccountAndPositionRisk struct {
	AdjEq              string               `json:"adjEq"`
	AccountBalanceData []AccountBalanceData `json:"balData"`
	PosData            []PositionData       `json:"posData"`
	Timestamp          types.Time           `json:"ts"`
}

// BillsDetailQueryParameter represents bills detail query parameter
type BillsDetailQueryParameter struct {
	InstrumentType string // Instrument type "SPOT" "MARGIN" "SWAP" "FUTURES" "OPTION"
	InstrumentID   string
	Currency       currency.Code
	MarginMode     string // Margin mode "isolated" "cross"
	ContractType   string // Contract type "linear" & "inverse" Only applicable to FUTURES/SWAP
	BillType       uint64 // Bill type 1: Transfer 2: Trade 3: Delivery 4: Auto token conversion 5: Liquidation 6: Margin transfer 7: Interest deduction 8: Funding fee 9: ADL 10: Clawback 11: System token conversion 12: Strategy transfer 13: ddh
	BillSubType    int64  // allowed bill subtype values are [ 1,2,3,4,5,6,9,11,12,14,160,161,162,110,111,118,119,100,101,102,103,104,105,106,110,125,126,127,128,131,132,170,171,172,112,113,117,173,174,200,201,202,203 ], link: https://www.okx.com/docs-v5/en/#rest-api-account-get-bills-details-last-7-days
	After          string
	Before         string
	BeginTime      time.Time
	EndTime        time.Time
	Limit          int64
}

// BillsDetailResp represents response for applying for bill-details
type BillsDetailResp struct {
	Result    string     `json:"result"`
	Timestamp types.Time `json:"ts"`
}

// BillsArchiveInfo represents a bill archive information
type BillsArchiveInfo struct {
	FileHref  string     `json:"fileHref"`
	State     string     `json:"state"`
	Timestamp types.Time `json:"ts"`
}

// BillsDetailResponse represents account bills information
type BillsDetailResponse struct {
	Balance                    types.Number `json:"bal"`
	BalanceChange              types.Number `json:"balChg"`
	BillID                     string       `json:"billId"`
	Currency                   string       `json:"ccy"`
	ExecType                   string       `json:"execType"` // Order flow type, T：taker M：maker
	Fee                        types.Number `json:"fee"`      // Fee Negative number represents the user transaction fee charged by the platform. Positive number represents rebate.
	From                       string       `json:"from"`     // The remitting account 6: FUNDING 18: Trading account When bill type is not transfer, the field returns "".
	InstrumentID               string       `json:"instId"`
	InstrumentType             asset.Item   `json:"instType"`
	MarginMode                 string       `json:"mgnMode"`
	Notes                      string       `json:"notes"` // notes When bill type is not transfer, the field returns "".
	OrderID                    string       `json:"ordId"`
	ProfitAndLoss              types.Number `json:"pnl"`
	PositionLevelBalance       types.Number `json:"posBal"`
	PositionLevelBalanceChange types.Number `json:"posBalChg"`
	SubType                    string       `json:"subType"`
	Price                      types.Number `json:"px"`
	Interest                   types.Number `json:"interest"`
	Tag                        string       `json:"tag"`
	FillTime                   types.Time   `json:"fillTime"`
	TradeID                    string       `json:"tradeId"`
	ClientOrdID                string       `json:"clOrdId"`
	FillIdxPrice               types.Number `json:"fillIdxPx"`
	FillMarkPrice              types.Number `json:"fillMarkPx"`
	FillPxVolume               types.Number `json:"fillPxVol"`
	FillPxUSD                  types.Number `json:"fillPxUsd"`
	FillMarkVolume             types.Number `json:"fillMarkVol"`
	FillFwdPrice               types.Number `json:"fillFwdPx"`
	Size                       types.Number `json:"sz"`
	To                         string       `json:"to"`
	Timestamp                  types.Time   `json:"ts"`
	Type                       string       `json:"type"`
}

// AccountConfigurationResponse represents account configuration response
type AccountConfigurationResponse struct {
	UID                            string       `json:"uid"`
	MainUID                        string       `json:"mainUid"`
	AccountSelfTradePreventionMode string       `json:"acctStpMode"`
	AccountLevel                   types.Number `json:"acctLv"`     // 1: Simple 2: Single-currency margin 3: Multi-currency margin 4：Portfolio margin
	AutoLoan                       bool         `json:"autoLoan"`   // Whether to borrow coins automatically true: borrow coins automatically false: not borrow coins automatically
	ContractIsolatedMode           string       `json:"ctIsoMode"`  // Contract isolated margin trading settings automatic：Auto transfers autonomy：Manual transfers
	GreeksType                     string       `json:"greeksType"` // Current display type of Greeks PA: Greeks in coins BS: Black-Scholes Greeks in dollars
	Level                          string       `json:"level"`      // The user level of the current real trading volume on the platform, e.g lv1
	LevelTemporary                 string       `json:"levelTmp"`
	MarginIsolatedMode             string       `json:"mgnIsoMode"` // Margin isolated margin trading settings automatic：Auto transfers autonomy：Manual transfers
	PositionMode                   string       `json:"posMode"`
	RoleType                       types.Number `json:"roleType"` // 0: General user 1: Leading trader 2: Copy trader
	TraderInsts                    []string     `json:"traderInsts"`
	SpotRoleType                   types.Number `json:"spotRoleType"` // SPOT copy trading role type. 0: General user；1: Leading trader；2: Copy trader
	SpotTraderInsts                []string     `json:"spotTraderInsts"`
	OptionalTradingAuth            types.Number `json:"opAuth"` // Whether the optional trading was activated 0: not activated 1: activated
	KYCLevel                       types.Number `json:"kycLv"`
	Label                          string       `json:"label"`
	IP                             string       `json:"ip"`
	Permission                     string       `json:"perm"`
	LiquidationGear                types.Number `json:"liquidationGear"`
	EnableSpotBorrow               bool         `json:"enableSpotBorrow"`
	SpotBorrowAutoRepay            bool         `json:"spotBorrowAutoRepay"`
	Type                           types.Number `json:"type"` // 0: Main account 1: Standard sub-account 2: Managed trading sub-account 5: Custody trading sub-account - Copper 9: Managed trading sub-account - Copper 12: Custody trading sub-account - Komainu
}

// PositionMode represents position mode response
type PositionMode struct {
	PositionMode string `json:"posMode"` // "long_short_mode": long/short, only applicable to FUTURES/SWAP "net_mode": net
}

// SetLeverageInput represents set leverage request input
type SetLeverageInput struct {
	Leverage     float64       `json:"lever,string"`     // set leverage for isolated
	MarginMode   string        `json:"mgnMode"`          // Margin Mode "cross" and "isolated"
	InstrumentID string        `json:"instId,omitempty"` // Optional:
	Currency     currency.Code `json:"ccy,omitzero"`     // Optional:
	PositionSide string        `json:"posSide,omitempty"`

	AssetType asset.Item `json:"-"`
}

// SetLeverageResponse represents set leverage response
type SetLeverageResponse struct {
	Leverage     types.Number `json:"lever"`
	MarginMode   string       `json:"mgnMode"` // Margin Mode "cross" and "isolated"
	InstrumentID string       `json:"instId"`
	PositionSide string       `json:"posSide"` // "long", "short", and "net"
}

// MaximumBuyAndSell get maximum buy , sell amount or open amount
type MaximumBuyAndSell struct {
	Currency     string       `json:"ccy"`
	InstrumentID string       `json:"instId"`
	MaximumBuy   types.Number `json:"maxBuy"`
	MaximumSell  types.Number `json:"maxSell"`
}

// MaximumTradableAmount represents get maximum tradable amount response
type MaximumTradableAmount struct {
	InstrumentID string `json:"instId"`
	AvailBuy     string `json:"availBuy"`
	AvailSell    string `json:"availSell"`
}

// IncreaseDecreaseMarginInput represents increase or decrease the margin of the isolated position
type IncreaseDecreaseMarginInput struct {
	InstrumentID      string  `json:"instId"`
	PositionSide      string  `json:"posSide"`
	MarginBalanceType string  `json:"type"`
	Amount            float64 `json:"amt,string"`
	Currency          string  `json:"ccy"`
}

// IncreaseDecreaseMargin represents increase or decrease the margin of the isolated position response
type IncreaseDecreaseMargin struct {
	Amount       types.Number `json:"amt"`
	Currency     string       `json:"ccy"`
	InstrumentID string       `json:"instId"`
	Leverage     types.Number `json:"leverage"`
	PositionSide string       `json:"posSide"`
	Type         string       `json:"type"`
}

// LeverageResponse instrument ID leverage response
type LeverageResponse struct {
	InstrumentID string       `json:"instId"`
	MarginMode   string       `json:"mgnMode"`
	PositionSide string       `json:"posSide"`
	Leverage     types.Number `json:"lever"`
	Currency     string       `json:"ccy"`
}

// LeverageEstimatedInfo leverage estimated info response
type LeverageEstimatedInfo struct {
	EstimatedAvailQuoteTrans string       `json:"estAvailQuoteTrans"`
	EstimatedAvailTrans      string       `json:"estAvailTrans"`
	EstimatedLiqPrice        types.Number `json:"estLiqPx"`
	EstimatedMaxAmount       types.Number `json:"estMaxAmt"`
	EstimatedMargin          types.Number `json:"estMgn"`
	EstimatedQuoteMaxAmount  types.Number `json:"estQuoteMaxAmt"`
	EstimatedQuoteMargin     types.Number `json:"estQuoteMgn"`
	ExistOrd                 bool         `json:"existOrd"` // Whether there is pending orders
	MaxLeverage              types.Number `json:"maxLever"`
	MinLeverage              types.Number `json:"minLever"`
}

// MaximumLoanInstrument represents maximum loan of an instrument ID
type MaximumLoanInstrument struct {
	InstrumentID   string       `json:"instId"`
	MarginMode     string       `json:"mgnMode"`
	MarginCurrency string       `json:"mgnCcy"`
	MaxLoan        types.Number `json:"maxLoan"`
	Currency       string       `json:"ccy"`
	Side           order.Side   `json:"side"`
}

// TradeFeeRate holds trade fee rate information for a given instrument type
type TradeFeeRate struct {
	Category         string         `json:"category"`
	DeliveryFeeRate  types.Number   `json:"delivery"`
	Exercise         string         `json:"exercise"`
	InstrumentType   asset.Item     `json:"instType"`
	FeeRateLevel     string         `json:"level"`
	FeeRateMaker     types.Number   `json:"maker"`
	FeeRateTaker     types.Number   `json:"taker"`
	Timestamp        types.Time     `json:"ts"`
	FeeRateMakerUSDT types.Number   `json:"makerU"`
	FeeRateTakerUSDT types.Number   `json:"takerU"`
	FeeRateMakerUSDC types.Number   `json:"makerUSDC"`
	FeeRateTakerUSDC types.Number   `json:"takerUSDC"`
	RuleType         string         `json:"ruleType"`
	Fiat             []FiatItemInfo `json:"fiat"`
}

// FiatItemInfo represents fiat currency with taker and maker fee details
type FiatItemInfo struct {
	Currency string       `json:"ccy"`
	Taker    types.Number `json:"taker"`
	Maker    types.Number `json:"maker"`
}

// InterestAccruedData represents interest rate accrued response
type InterestAccruedData struct {
	Currency     string       `json:"ccy"`
	InstrumentID string       `json:"instId"`
	Interest     types.Number `json:"interest"`
	InterestRate types.Number `json:"interestRate"` // Interest rate in an hour.
	Liability    types.Number `json:"liab"`
	MarginMode   string       `json:"mgnMode"` //  	Margin mode "cross" "isolated"
	Timestamp    types.Time   `json:"ts"`
	LoanType     string       `json:"type"`
}

// VIPInterestData holds interest accrued/deducted data
type VIPInterestData struct {
	OrderID      string       `json:"ordId"`
	Currency     string       `json:"ccy"`
	Interest     types.Number `json:"interest"`
	InterestRate types.Number `json:"interestRate"`
	Liability    types.Number `json:"liab"`
	Timestamp    types.Time   `json:"ts"`
}

// VIPLoanOrder holds VIP loan items
type VIPLoanOrder struct {
	OrderID         string       `json:"ordId"`
	Currency        string       `json:"ccy"`
	State           string       `json:"state"`
	BorrowAmount    types.Number `json:"borrowAmt"`
	CurrentRate     types.Number `json:"curRate"`
	DueAmount       types.Number `json:"dueAmt"`
	NextRefreshTime types.Time   `json:"nextRefreshTime"`
	OriginalRate    types.Number `json:"origRate"`
	RepayAmount     types.Number `json:"repayAmt"`
	Timestamp       types.Time   `json:"ts"`
}

// VIPLoanOrderDetail holds vip loan order detail
type VIPLoanOrderDetail struct {
	Amount     types.Number `json:"amt"`
	Currency   string       `json:"ccy"`
	FailReason string       `json:"failReason"`
	Rate       types.Number `json:"rate"`
	Timestamp  types.Time   `json:"ts"`
	Type       string       `json:"type"` // Operation Type: 1:Borrow 2:Repayment 3:System Repayment 4:Interest Rate Refresh
}

// InterestRateResponse represents interest rate response
type InterestRateResponse struct {
	InterestRate types.Number `json:"interestRate"`
	Currency     string       `json:"ccy"`
}

// GreeksType represents for greeks type response
type GreeksType struct {
	GreeksType string `json:"greeksType"` // Display type of Greeks. PA: Greeks in coins BS: Black-Scholes Greeks in dollars
}

// IsolatedMode represents Isolated margin trading settings
type IsolatedMode struct {
	IsoMode        string `json:"isoMode"` // "automatic":Auto transfers "autonomy":Manual transfers
	InstrumentType string `json:"type"`    // Instrument type "MARGIN" "CONTRACTS"
}

// BorrowAndRepay manual holds manual borrow and repay in quick margin mode
type BorrowAndRepay struct {
	Amount       float64       `json:"amt,string"`
	InstrumentID string        `json:"instId"`
	LoanCcy      currency.Code `json:"ccy"`
	Side         string        `json:"side"` // possible values: 'borrow' and 'repay'
}

// BorrowRepayHistoryItem holds borrow or repay history item information
type BorrowRepayHistoryItem struct {
	InstrumentID    string       `json:"instId"`
	Currency        string       `json:"ccy"`
	Side            string       `json:"side"`
	AccBorrowAmount types.Number `json:"accBorrowed"`
	Amount          types.Number `json:"amt"`
	RefID           string       `json:"refId"`
	Timestamp       types.Time   `json:"ts"`
}

// MaximumWithdrawal represents maximum withdrawal amount query response
type MaximumWithdrawal struct {
	Currency                string       `json:"ccy"`
	MaximumWithdrawal       types.Number `json:"maxWd"`   // Max withdrawal (not allowing borrowed crypto transfer out under Multi-currency margin)
	MaximumWithdrawalEx     types.Number `json:"maxWdEx"` // Max withdrawal (allowing borrowed crypto transfer out under Multi-currency margin)
	SpotOffsetMaxWithdrawal types.Number `json:"spotOffsetMaxWd"`
	SpotOffsetMaxWdEx       types.Number `json:"spotOffsetMaxWdEx"`
}

// AccountRiskState represents account risk state
type AccountRiskState struct {
	IsTheAccountAtRisk string     `json:"atRisk"`
	AtRiskIdx          []any      `json:"atRiskIdx"` // derivatives risk unit list
	AtRiskMgn          []any      `json:"atRiskMgn"` // margin risk unit list
	Timestamp          types.Time `json:"ts"`
}

// LoanBorrowAndReplayInput represents currency VIP borrow or repay request params
type LoanBorrowAndReplayInput struct {
	Currency currency.Code `json:"ccy"`
	Side     string        `json:"side,omitempty"`
	Amount   float64       `json:"amt,string,omitempty"`
}

// LoanBorrowAndReplay loans borrow and repay
type LoanBorrowAndReplay struct {
	Amount        types.Number `json:"amt"`
	AvailableLoan types.Number `json:"availLoan"`
	Currency      string       `json:"ccy"`
	LoanQuota     types.Number `json:"loanQuota"`
	PosLoan       string       `json:"posLoan"`
	Side          string       `json:"side"` // borrow or repay
	UsedLoan      string       `json:"usedLoan"`
}

// BorrowRepayHistory represents borrow and repay history item data
type BorrowRepayHistory struct {
	Currency   string     `json:"ccy"`
	TradedLoan string     `json:"tradedLoan"`
	Timestamp  types.Time `json:"ts"`
	Type       string     `json:"type"`
	UsedLoan   string     `json:"usedLoan"`
}

// BorrowInterestAndLimitResponse represents borrow interest and limit rate for different loan type
type BorrowInterestAndLimitResponse struct {
	Debt             string       `json:"debt"`
	Interest         string       `json:"interest"`
	NextDiscountTime types.Time   `json:"nextDiscountTime"`
	NextInterestTime types.Time   `json:"nextInterestTime"`
	LoanAllocation   types.Number `json:"loanAlloc"`
	Records          []struct {
		AvailLoan           types.Number       `json:"availLoan"`
		Currency            string             `json:"ccy"`
		Interest            types.Number       `json:"interest"`
		LoanQuota           types.Number       `json:"loanQuota"`
		PosLoan             types.Number       `json:"posLoan"` // Frozen amount for current account Only applicable to VIP loans
		Rate                types.Number       `json:"rate"`
		SurplusLimit        types.Number       `json:"surplusLmt"`
		SurplusLimitDetails SurplusLimitDetail `json:"surplusLmtDetails"`
		UsedLmt             types.Number       `json:"usedLmt"`
		UsedLoan            types.Number       `json:"usedLoan"`
	} `json:"records"`
}

// SurplusLimitDetail represents details of available amount across all sub-accounts. The value of surplusLmt is the minimum value within this array
type SurplusLimitDetail struct {
	AllAcctRemainingQuota string `json:"allAcctRemainingQuota"`
	CurAcctRemainingQuota string `json:"curAcctRemainingQuota"`
	PlatRemainingQuota    string `json:"platRemainingQuota"`
}

// FixedLoanBorrowLimitInformation represents a fixed loan borrow information
type FixedLoanBorrowLimitInformation struct {
	TotalBorrowLimit     types.Number `json:"totalBorrowLmt"`
	TotalAvailableBorrow types.Number `json:"totalAvailBorrow"`
	Borrowed             types.Number `json:"borrowed"`
	UsedAmount           types.Number `json:"used"`
	AvailRepay           string       `json:"availRepay"`
	Details              []struct {
		Borrowed    types.Number `json:"borrowed"`
		AvailBorrow types.Number `json:"availBorrow"`
		Currency    string       `json:"ccy"`
		MinBorrow   types.Number `json:"minBorrow"`
		Used        types.Number `json:"used"`
		Term        string       `json:"term"`
	} `json:"details"`
	Timestamp types.Time `json:"ts"`
}

// FixedLoanBorrowQuote represents a fixed loan quote details
type FixedLoanBorrowQuote struct {
	Currency        string       `json:"ccy"`
	Term            string       `json:"term"`
	EstAvailBorrow  types.Number `json:"estAvailBorrow"`
	EstRate         types.Number `json:"estRate"`
	EstInterest     types.Number `json:"estInterest"`
	PenaltyInterest types.Number `json:"penaltyInterest"`
	Timestamp       types.Time   `json:"ts"`
}

// PositionItem represents current position of the user
type PositionItem struct {
	Position     string `json:"pos"`
	InstrumentID string `json:"instId"`
}

// PositionBuilderInput represents request parameter for position builder item
type PositionBuilderInput struct {
	InstrumentType         string         `json:"instType,omitempty"`
	InstrumentID           string         `json:"instId,omitempty"`
	ImportExistingPosition bool           `json:"inclRealPos,omitempty"` // "true"：Import existing positions and hedge with simulated ones "false"：Only use simulated positions The default is true
	ListOfPositions        []PositionItem `json:"simPos,omitempty"`
	PositionsCount         uint64         `json:"pos,omitempty"`
}

// PositionBuilderResponse represents a position builder endpoint response
type PositionBuilderResponse struct {
	InitialMarginRequirement     string                `json:"imr"` // Initial margin requirement of riskUnit dimension
	MaintenanceMarginRequirement string                `json:"mmr"` // Maintenance margin requirement of riskUnit dimension
	SpotAndVolumeMovement        string                `json:"mr1"`
	ThetaDecay                   string                `json:"mr2"`
	VegaTermStructure            string                `json:"mr3"`
	BasicRisk                    string                `json:"mr4"`
	InterestRateRisk             string                `json:"mr5"`
	ExtremeMarketMove            string                `json:"mr6"`
	TransactionCostAndSlippage   string                `json:"mr7"`
	PositionData                 []PositionBuilderData `json:"posData"` // List of positions
	RiskUnit                     string                `json:"riskUnit"`
	Timestamp                    types.Time            `json:"ts"`
}

// PositionBuilderData represent a position item
type PositionBuilderData struct {
	Delta              string       `json:"delta"`
	Gamma              string       `json:"gamma"`
	InstrumentID       string       `json:"instId"`
	InstrumentType     string       `json:"instType"`
	NotionalUSD        types.Number `json:"notionalUsd"` // Quantity of positions usd
	QuantityOfPosition types.Number `json:"pos"`         // Quantity of positions
	Theta              string       `json:"theta"`       // Sensitivity of option price to remaining maturity
	Vega               string       `json:"vega"`        // Sensitivity of option price to implied volatility
}

// GreeksItem represents greeks response
type GreeksItem struct {
	ThetaBS   string     `json:"thetaBS"`
	ThetaPA   string     `json:"thetaPA"`
	DeltaBS   string     `json:"deltaBS"`
	DeltaPA   string     `json:"deltaPA"`
	GammaBS   string     `json:"gammaBS"`
	GammaPA   string     `json:"gammaPA"`
	VegaBS    string     `json:"vegaBS"`
	VegaPA    string     `json:"vegaPA"`
	Currency  string     `json:"ccy"`
	Timestamp types.Time `json:"ts"`
}

// CounterpartiesResponse represents
type CounterpartiesResponse struct {
	TraderName string `json:"traderName"`
	TraderCode string `json:"traderCode"`
	Type       string `json:"type"`
}

// RFQOrderLeg represents RFQ Order responses leg
type RFQOrderLeg struct {
	Size         types.Number `json:"sz"`
	Side         string       `json:"side"`
	InstrumentID string       `json:"instId"`
	TgtCurrency  string       `json:"tgtCcy,omitempty"`
}

// CreateRFQInput RFQ create method input
type CreateRFQInput struct {
	Anonymous      bool          `json:"anonymous"`
	CounterParties []string      `json:"counterparties"`
	ClientRFQID    string        `json:"clRfqId"`
	Legs           []RFQOrderLeg `json:"legs"`
}

// CancelRFQRequestParam represents cancel RFQ order request params
type CancelRFQRequestParam struct {
	RFQID       string `json:"rfqId,omitempty"`
	ClientRFQID string `json:"clRfqId,omitempty"`
}

// CancelRFQRequestsParam represents cancel multiple RFQ orders request params
type CancelRFQRequestsParam struct {
	RFQIDs       []string `json:"rfqIds"`
	ClientRFQIDs []string `json:"clRfqIds"`
}

// CancelRFQResponse represents cancel RFQ orders response
type CancelRFQResponse struct {
	RFQID         string `json:"rfqId"`
	ClientRFQID   string `json:"clRfqId"`
	StatusCode    string `json:"sCode"`
	StatusMessage string `json:"sMsg"`
}

// MMPStatusResponse holds MMP reset status response
type MMPStatusResponse struct {
	Result bool `json:"result"`
}

// MMPConfig holds request/response structure to set Market Maker Protection (MMP)
type MMPConfig struct {
	InstrumentFamily string  `json:"instFamily"`
	TimeInterval     int64   `json:"timeInterval,string"`
	FrozenInterval   int64   `json:"frozenInterval,string"` // Frozen period (ms). "0" means the trade will remain frozen until you request "Reset MMP Status" to unfrozen
	QuantityLimit    float64 `json:"qtyLimit,string"`
}

// MMPConfigDetail holds MMP config details
type MMPConfigDetail struct {
	FrozenInterval   types.Number `json:"frozenInterval"`
	InstrumentFamily string       `json:"instFamily"`
	MMPFrozen        bool         `json:"mmpFrozen"`
	MMPFrozenUntil   string       `json:"mmpFrozenUntil"`
	QuantityLimit    types.Number `json:"qtyLimit"`
	TimeInterval     int64        `json:"timeInterval"`
}

// ExecuteQuoteParams represents Execute quote request params
type ExecuteQuoteParams struct {
	RFQID   string `json:"rfqId,omitempty"`
	QuoteID string `json:"quoteId,omitempty"`
}

// ExecuteQuoteResponse represents execute quote response
type ExecuteQuoteResponse struct {
	BlockTradedID   string     `json:"blockTdId"`
	RFQID           string     `json:"rfqId"`
	ClientRFQID     string     `json:"clRfqId"`
	QuoteID         string     `json:"quoteId"`
	ClientQuoteID   string     `json:"clQuoteId"`
	TraderCode      string     `json:"tTraderCode"`
	MakerTraderCode string     `json:"mTraderCode"`
	CreationTime    types.Time `json:"cTime"`
	Legs            []OrderLeg `json:"legs"`
}

// QuoteProduct represents products which makers want to quote and receive RFQs for
type QuoteProduct struct {
	InstrumentType string `json:"instType,omitempty"`
	IncludeALL     bool   `json:"includeALL"`
	Data           []struct {
		Underlying     string       `json:"uly"`
		MaxBlockSize   types.Number `json:"maxBlockSz"`
		MakerPriceBand types.Number `json:"makerPxBand"`
	} `json:"data"`
	InstrumentType0 string `json:"instType:,omitempty"`
}

// OrderLeg represents legs information for both websocket and REST available Quote information
type OrderLeg struct {
	Price          types.Number `json:"px"`
	Size           types.Number `json:"sz"`
	InstrumentID   string       `json:"instId"`
	Side           string       `json:"side"`
	TargetCurrency string       `json:"tgtCcy"`

	// available in REST only
	Fee         types.Number `json:"fee"`
	FeeCurrency string       `json:"feeCcy"`
	TradeID     string       `json:"tradeId"`
}

// CreateQuoteParams holds information related to create quote
type CreateQuoteParams struct {
	RFQID         string     `json:"rfqId"`
	ClientQuoteID string     `json:"clQuoteId"`
	QuoteSide     string     `json:"quoteSide"`
	Legs          []QuoteLeg `json:"legs"`
}

// QuoteLeg the legs of the Quote
type QuoteLeg struct {
	Price          float64    `json:"px,string"`
	SizeOfQuoteLeg float64    `json:"sz,string"`
	InstrumentID   string     `json:"instId"`
	Side           order.Side `json:"side"`

	// TargetCurrency represents target currency
	TargetCurrency string `json:"tgtCcy,omitempty"`
}

// QuoteResponse holds create quote response variables
type QuoteResponse struct {
	CreationTime  types.Time `json:"cTime"`
	UpdateTime    types.Time `json:"uTime"`
	ValidUntil    types.Time `json:"validUntil"`
	QuoteID       string     `json:"quoteId"`
	ClientQuoteID string     `json:"clQuoteId"`
	RFQID         string     `json:"rfqId"`
	QuoteSide     string     `json:"quoteSide"`
	ClientRFQID   string     `json:"clRfqId"`
	TraderCode    string     `json:"traderCode"`
	State         string     `json:"state"`
	Legs          []QuoteLeg `json:"legs"`
}

// CancelQuoteRequestParams represents cancel quote request params
type CancelQuoteRequestParams struct {
	QuoteID       string `json:"quoteId,omitempty"`
	ClientQuoteID string `json:"clQuoteId,omitempty"`
}

// CancelQuotesRequestParams represents cancel multiple quotes request params
type CancelQuotesRequestParams struct {
	QuoteIDs       []string `json:"quoteIds,omitempty"`
	ClientQuoteIDs []string `json:"clQuoteIds,omitempty"`
}

// CancelQuoteResponse represents cancel quote response
type CancelQuoteResponse struct {
	QuoteID       string `json:"quoteId"`
	ClientQuoteID string `json:"clQuoteId"`
	StatusCode    string `json:"sCode"`
	StatusMessage string `json:"sMsg"`
}

// RFQRequestParams represents get RFQ orders param
type RFQRequestParams struct {
	RFQID       string
	ClientRFQID string
	State       string
	BeginningID string
	EndID       string
	Limit       int64
}

// RFQResponse RFQ response detail
type RFQResponse struct {
	CreateTime     types.Time `json:"cTime"`
	UpdateTime     types.Time `json:"uTime"`
	ValidUntil     types.Time `json:"validUntil"`
	TraderCode     string     `json:"traderCode"`
	RFQID          string     `json:"rfqId"`
	ClientRFQID    string     `json:"clRfqId"`
	State          string     `json:"state"`
	Counterparties []string   `json:"counterparties"`
	Legs           []struct {
		InstrumentID string       `json:"instId"`
		Size         types.Number `json:"sz"`
		Side         string       `json:"side"`
		TgtCurrency  string       `json:"tgtCcy"`
	} `json:"legs"`
}

// QuoteRequestParams request params
type QuoteRequestParams struct {
	RFQID         string
	ClientRFQID   string
	QuoteID       string
	ClientQuoteID string
	State         string
	BeginID       string
	EndID         string
	Limit         int64
}

// RFQTradesRequestParams represents RFQ trades request param
type RFQTradesRequestParams struct {
	RFQID         string
	ClientRFQID   string
	QuoteID       string
	BlockTradeID  string
	ClientQuoteID string
	State         string
	BeginID       string
	EndID         string
	Limit         int64
}

// RFQTradeResponse RFQ trade response
type RFQTradeResponse struct {
	RFQID           string        `json:"rfqId"`
	ClientRFQID     string        `json:"clRfqId"`
	QuoteID         string        `json:"quoteId"`
	ClientQuoteID   string        `json:"clQuoteId"`
	BlockTradeID    string        `json:"blockTdId"`
	Legs            []RFQTradeLeg `json:"legs"`
	CreationTime    types.Time    `json:"cTime"`
	TakerTraderCode string        `json:"tTraderCode"`
	MakerTraderCode string        `json:"mTraderCode"`
}

// RFQTradeLeg RFQ trade response leg
type RFQTradeLeg struct {
	InstrumentID string       `json:"instId"`
	Side         string       `json:"side"`
	Size         string       `json:"sz"`
	Price        types.Number `json:"px"`
	TradeID      string       `json:"tradeId"`

	Fee         types.Number `json:"fee"`
	FeeCurrency string       `json:"feeCcy"`
}

// PublicTradesResponse represents data will be pushed whenever there is a block trade
type PublicTradesResponse struct {
	BlockTradeID string        `json:"blockTdId"`
	CreationTime types.Time    `json:"cTime"`
	Legs         []RFQTradeLeg `json:"legs"`
}

// SubaccountInfo represents subaccount information detail
type SubaccountInfo struct {
	Enable          bool       `json:"enable"`
	SubAccountName  string     `json:"subAcct"`
	SubaccountType  string     `json:"type"` // sub-account note
	SubaccountLabel string     `json:"label"`
	MobileNumber    string     `json:"mobile"`      // Mobile number that linked with the sub-account.
	GoogleAuth      bool       `json:"gAuth"`       // If the sub-account switches on the Google Authenticator for login authentication.
	CanTransferOut  bool       `json:"canTransOut"` // If can transfer out, false: can not transfer out, true: can transfer.
	Timestamp       types.Time `json:"ts"`
}

// SubaccountBalanceDetail represents subaccount balance detail
type SubaccountBalanceDetail struct {
	AvailableBalance               types.Number `json:"availBal"`
	AvailableEquity                types.Number `json:"availEq"`
	CashBalance                    types.Number `json:"cashBal"`
	Currency                       string       `json:"ccy"`
	CrossLiability                 types.Number `json:"crossLiab"`
	DiscountEquity                 types.Number `json:"disEq"`
	Equity                         types.Number `json:"eq"`
	EquityUSD                      types.Number `json:"eqUsd"`
	FrozenBalance                  types.Number `json:"frozenBal"`
	Interest                       types.Number `json:"interest"`
	IsoEquity                      string       `json:"isoEq"`
	IsolatedLiabilities            types.Number `json:"isoLiab"`
	LiabilitiesOfCurrency          string       `json:"liab"`
	MaxLoan                        types.Number `json:"maxLoan"`
	MarginRatio                    types.Number `json:"mgnRatio"`
	NotionalLeverage               string       `json:"notionalLever"`
	OrdFrozen                      string       `json:"ordFrozen"`
	Twap                           string       `json:"twap"`
	UpdateTime                     types.Time   `json:"uTime"`
	UnrealizedProfitAndLoss        types.Number `json:"upl"`
	UnrealizedProfitAndLiabilities string       `json:"uplLiab"`
	FixedBalance                   types.Number `json:"fixedBal"`
	BorrowFroz                     types.Number `json:"borrowFroz"`
	SpotISOBalance                 types.Number `json:"spotIsoBal"`
	SMTSyncEquity                  types.Number `json:"smtSyncEq"`
}

// SubaccountBalanceResponse represents subaccount balance response
type SubaccountBalanceResponse struct {
	AdjustedEffectiveEquity      string                    `json:"adjEq"`
	Details                      []SubaccountBalanceDetail `json:"details"`
	Imr                          string                    `json:"imr"`
	IsolatedMarginEquity         string                    `json:"isoEq"`
	MarginRatio                  types.Number              `json:"mgnRatio"`
	MaintenanceMarginRequirement types.Number              `json:"mmr"`
	NotionalUSD                  types.Number              `json:"notionalUsd"`
	OrdFroz                      types.Number              `json:"ordFroz"`
	TotalEq                      types.Number              `json:"totalEq"`
	UpdateTime                   types.Time                `json:"uTime"`
	BorrowFroz                   types.Number              `json:"borrowFroz"`
	UPL                          types.Number              `json:"upl"`
}

// FundingBalance holds function balance
type FundingBalance struct {
	AvailableBalance types.Number `json:"availBal"`
	Balance          types.Number `json:"bal"`
	Currency         string       `json:"ccy"`
	FrozenBalance    types.Number `json:"frozenBal"`
}

// SubAccountMaximumWithdrawal holds sub-account maximum withdrawal information
type SubAccountMaximumWithdrawal struct {
	Currency          string       `json:"ccy"`
	MaxWd             types.Number `json:"maxWd"`
	MaxWdEx           types.Number `json:"maxWdEx"`
	SpotOffsetMaxWd   types.Number `json:"spotOffsetMaxWd"`
	SpotOffsetMaxWdEx types.Number `json:"spotOffsetMaxWdEx"`
}

// SubaccountBillItem represents subaccount balance bill item
type SubaccountBillItem struct {
	BillID                 string       `json:"billId"`
	Type                   string       `json:"type"`
	AccountCurrencyBalance string       `json:"ccy"`
	Amount                 types.Number `json:"amt"`
	SubAccount             string       `json:"subAcct"`
	Timestamp              types.Time   `json:"ts"`
}

// SubAccountTransfer holds sub-account transfer instance
type SubAccountTransfer struct {
	BillID     string       `json:"billId"`
	Type       string       `json:"type"`
	Currency   string       `json:"ccy"`
	SubAccount string       `json:"subAcct"`
	SubUID     string       `json:"subUid"`
	Amount     types.Number `json:"amt"`
	Timestamp  types.Time   `json:"ts"`
}

// SubAccountAssetTransferParams represents subaccount asset transfer request parameters
type SubAccountAssetTransferParams struct {
	Currency         currency.Code `json:"ccy"`            // {REQUIRED}
	Amount           float64       `json:"amt,string"`     // {REQUIRED}
	From             int64         `json:"from,string"`    // {REQUIRED} 6:Funding Account 18:Trading account
	To               int64         `json:"to,string"`      // {REQUIRED} 6:Funding Account 18:Trading account
	FromSubAccount   string        `json:"fromSubAccount"` // {REQUIRED} subaccount name.
	ToSubAccount     string        `json:"toSubAccount"`   // {REQUIRED} destination sub-account
	LoanTransfer     bool          `json:"loanTrans,omitempty"`
	OmitPositionRisk bool          `json:"omitPosRisk,omitempty"`
}

// TransferIDInfo represents master account transfer between subaccount
type TransferIDInfo struct {
	TransferID string `json:"transId"`
}

// PermissionOfTransfer represents subaccount transfer information and it's permission
type PermissionOfTransfer struct {
	SubAcct     string `json:"subAcct"`
	CanTransOut bool   `json:"canTransOut"`
}

// SubaccountName represents single subaccount name
type SubaccountName struct {
	SubaccountName string `json:"subAcct"`
}

// SubAccountLoanAllocationParam holds parameter for VIP sub-account loan allocation
type SubAccountLoanAllocationParam struct {
	Enable bool                              `json:"enable"`
	Alloc  []subAccountVIPLoanAllocationInfo `json:"alloc"`
}

type subAccountVIPLoanAllocationInfo struct {
	SubAcct   string  `json:"subAcct"`
	LoanAlloc float64 `json:"loanAlloc,string"`
}

// SubAccounBorrowInterestAndLimit represents sub-account borrow interest and limit
type SubAccounBorrowInterestAndLimit struct {
	SubAcct          string       `json:"subAcct"`
	Debt             types.Number `json:"debt"`
	Interest         types.Number `json:"interest"`
	NextDiscountTime types.Time   `json:"nextDiscountTime"`
	NextInterestTime types.Time   `json:"nextInterestTime"`
	LoanAlloc        types.Number `json:"loanAlloc"`
	Records          []struct {
		AvailLoan         types.Number `json:"availLoan"`
		Currency          string       `json:"ccy"`
		Interest          types.Number `json:"interest"`
		LoanQuota         types.Number `json:"loanQuota"`
		PosLoan           string       `json:"posLoan"`
		Rate              types.Number `json:"rate"`
		SurplusLmt        string       `json:"surplusLmt"`
		SurplusLmtDetails struct {
			AllAcctRemainingQuota types.Number `json:"allAcctRemainingQuota"`
			CurAcctRemainingQuota types.Number `json:"curAcctRemainingQuota"`
			PlatRemainingQuota    types.Number `json:"platRemainingQuota"`
		} `json:"surplusLmtDetails"`
		UsedLmt  types.Number `json:"usedLmt"`
		UsedLoan types.Number `json:"usedLoan"`
	} `json:"records"`
}

// GridAlgoOrder represents grid algo order
type GridAlgoOrder struct {
	InstrumentID string  `json:"instId"`
	AlgoOrdType  string  `json:"algoOrdType"`
	MaxPrice     float64 `json:"maxPx,string"`
	MinPrice     float64 `json:"minPx,string"`
	GridQuantity float64 `json:"gridNum,string"`
	GridType     string  `json:"runType"` // "1": Arithmetic, "2": Geometric Default is Arithmetic

	// Spot Grid Order
	QuoteSize float64 `json:"quoteSz,string"` // Invest amount for quote currency Either "instId" or "ccy" is required
	BaseSize  float64 `json:"baseSz,string"`  // Invest amount for base currency Either "instId" or "ccy" is required

	// Contract Grid Order
	BasePosition bool    `json:"basePos"` // Whether or not open a position when strategy actives Default is false Neutral contract grid should omit the parameter
	Size         float64 `json:"sz,string"`
	Direction    string  `json:"direction"`
	Leverage     string  `json:"lever"`
}

// GridAlgoOrderIDResponse represents grid algo order
type GridAlgoOrderIDResponse struct {
	AlgoOrderID   string `json:"algoId"`
	StatusCode    int64  `json:"sCode,string"`
	StatusMessage string `json:"sMsg"`
}

// StopGridAlgoOrderParam holds stop grid algo order parameter
type StopGridAlgoOrderParam struct {
	AlgoID        string `json:"algoId"`
	InstrumentID  string `json:"instId"`
	StopType      string `json:"stopType"`
	AlgoOrderType string `json:"algoOrdType"`
}

// ClosePositionParams holds close position parameters
type ClosePositionParams struct {
	AlgoID                  string  `json:"algoId"`
	MarketCloseAllPositions bool    `json:"mktClose"` // true: Market close all position, false：Close part of position
	Size                    float64 `json:"sz,omitempty,string"`
	Price                   float64 `json:"px,omitempty,string"`
}

// ClosePositionContractGridResponse holds contract grid close position response data
type ClosePositionContractGridResponse struct {
	AlgoClientOrderID string `json:"algoClOrdId"`
	AlgoID            string `json:"algoId"`
	OrderID           string `json:"ordId"`
	Tag               string `json:"tag"`
}

// CancelClosePositionOrder holds close position order parameter cancellation parameter
type CancelClosePositionOrder struct {
	AlgoID  string `json:"algoId"`
	OrderID string `json:"ordId"` // Close position order ID
}

// TriggeredGridAlgoOrderInfo holds grid algo order info
type TriggeredGridAlgoOrderInfo struct {
	AlgoClientOrderID string `json:"algoClOrdId"`
	AlgoID            string `json:"algoId"`
}

// GridAlgoOrderAmend represents amend algo order response
type GridAlgoOrderAmend struct {
	AlgoID                 string       `json:"algoId"`
	InstrumentID           string       `json:"instId"`
	StopLossTriggerPrice   types.Number `json:"slTriggerPx"`
	TakeProfitTriggerPrice types.Number `json:"tpTriggerPx"`
}

// StopGridAlgoOrderRequest represents stop grid algo order request parameter
type StopGridAlgoOrderRequest struct {
	AlgoID        string `json:"algoId"`
	InstrumentID  string `json:"instId"`
	StopType      uint64 `json:"stopType,string"` // Spot grid "1": Sell base currency "2": Keep base currency | Contract grid "1": Market Close All positions "2": Keep positions
	AlgoOrderType string `json:"algoOrdType"`
}

// GridAlgoOrderResponse is a complete information of grid algo order item response
type GridAlgoOrderResponse struct {
	ActualLever               string       `json:"actualLever"`
	AlgoID                    string       `json:"algoId"`
	AlgoOrderType             string       `json:"algoOrdType"`
	ArbitrageNumber           string       `json:"arbitrageNum"`
	BasePosition              bool         `json:"basePos"`
	BaseSize                  types.Number `json:"baseSz"`
	CancelType                string       `json:"cancelType"`
	Direction                 string       `json:"direction"`
	FloatProfit               types.Number `json:"floatProfit"`
	GridQuantity              types.Number `json:"gridNum"`
	GridProfit                string       `json:"gridProfit"`
	InstrumentID              string       `json:"instId"`
	InstrumentType            string       `json:"instType"`
	Investment                string       `json:"investment"`
	Leverage                  string       `json:"lever"`
	EstimatedLiquidationPrice types.Number `json:"liqPx"`
	MaximumPrice              types.Number `json:"maxPx"`
	MinimumPrice              types.Number `json:"minPx"`
	ProfitAndLossRatio        types.Number `json:"pnlRatio"`
	QuoteSize                 types.Number `json:"quoteSz"`
	RunType                   string       `json:"runType"`
	StopLossTriggerPrice      types.Number `json:"slTriggerPx"`
	State                     string       `json:"state"`
	StopResult                string       `json:"stopResult,omitempty"`
	StopType                  string       `json:"stopType"`
	Size                      types.Number `json:"sz"`
	Tag                       string       `json:"tag"`
	TotalProfitAndLoss        types.Number `json:"totalPnl"`
	TakeProfitTriggerPrice    types.Number `json:"tpTriggerPx"`
	CreationTime              types.Time   `json:"cTime"`
	UpdateTime                types.Time   `json:"uTime"`
	Underlying                string       `json:"uly"`

	// Added in Detail

	EquityOfStrength    string       `json:"eq,omitempty"`
	PerMaxProfitRate    types.Number `json:"perMaxProfitRate,omitempty"`
	PerMinProfitRate    types.Number `json:"perMinProfitRate,omitempty"`
	Profit              types.Number `json:"profit,omitempty"`
	Runpx               string       `json:"runpx,omitempty"`
	SingleAmt           types.Number `json:"singleAmt,omitempty"`
	TotalAnnualizedRate types.Number `json:"totalAnnualizedRate,omitempty"`
	TradeNumber         string       `json:"tradeNum,omitempty"`

	// Suborders Detail

	AnnualizedRate types.Number `json:"annualizedRate,omitempty"`
	CurBaseSize    types.Number `json:"curBaseSz,omitempty"`
	CurQuoteSize   types.Number `json:"curQuoteSz,omitempty"`
}

// AlgoOrderPosition represents algo order position detailed data
type AlgoOrderPosition struct {
	AutoDecreasingLine           string       `json:"adl"`
	AlgoID                       string       `json:"algoId"`
	AveragePrice                 types.Number `json:"avgPx"`
	Currency                     string       `json:"ccy"`
	InitialMarginRequirement     string       `json:"imr"`
	InstrumentID                 string       `json:"instId"`
	InstrumentType               string       `json:"instType"`
	LastTradedPrice              types.Number `json:"last"`
	Leverage                     types.Number `json:"lever"`
	LiquidationPrice             types.Number `json:"liqPx"`
	MarkPrice                    types.Number `json:"markPx"`
	MarginMode                   string       `json:"mgnMode"`
	MarginRatio                  types.Number `json:"mgnRatio"`
	MaintenanceMarginRequirement string       `json:"mmr"`
	NotionalUSD                  types.Number `json:"notionalUsd"`
	QuantityPosition             types.Number `json:"pos"`
	PositionSide                 string       `json:"posSide"`
	UnrealizedProfitAndLoss      types.Number `json:"upl"`
	UnrealizedProfitAndLossRatio types.Number `json:"uplRatio"`
	UpdateTime                   types.Time   `json:"uTime"`
	CreationTime                 types.Time   `json:"cTime"`
}

// AlgoOrderWithdrawalProfit algo withdrawal order profit info
type AlgoOrderWithdrawalProfit struct {
	AlgoID         string `json:"algoId"`
	WithdrawProfit string `json:"profit"`
}

// SystemStatusResponse represents the system status and other details
type SystemStatusResponse struct {
	Title               string     `json:"title"`
	State               string     `json:"state"`
	Begin               types.Time `json:"begin"` // Begin time of system maintenance,
	End                 types.Time `json:"end"`   // Time of resuming trading totally.
	Href                string     `json:"href"`  // Hyperlink for system maintenance details
	ServiceType         string     `json:"serviceType"`
	System              string     `json:"system"`
	ScheduleDescription string     `json:"scheDesc"`
	PreOpenBegin        string     `json:"preOpenBegin"`
	MaintenanceType     string     `json:"maintType"`
	Environment         string     `json:"env"` // Environment '1': Production Trading '2': Demo Trading

	// PushTime timestamp information when the data is pushed
	PushTime types.Time `json:"ts"`
}

// BlockTicker holds block trading information
type BlockTicker struct {
	InstrumentType           string       `json:"instType"`
	InstrumentID             string       `json:"instId"`
	TradingVolumeInCCY24Hour types.Number `json:"volCcy24h"`
	TradingVolumeInUSD24Hour types.Number `json:"vol24h"`
	Timestamp                types.Time   `json:"ts"`
}

// BlockTrade represents a block trade
type BlockTrade struct {
	InstrumentID   string       `json:"instId"`
	TradeID        string       `json:"tradeId"`
	Price          types.Number `json:"px"`
	Size           types.Number `json:"sz"`
	Side           order.Side   `json:"side"`
	FillVolatility types.Number `json:"fillVol"`
	ForwardPrice   types.Number `json:"fwdPx"`
	IndexPrice     types.Number `json:"idxPx"`
	MarkPrice      types.Number `json:"markPx"`
	Timestamp      types.Time   `json:"ts"`
}

// SpreadOrderParam holds parameters for spread orders
type SpreadOrderParam struct {
	InstrumentID  string  `json:"instId"`
	SpreadID      string  `json:"sprdId,omitempty"`
	ClientOrderID string  `json:"clOrdId,omitempty"`
	Side          string  `json:"side"`    // Order side, buy sell
	OrderType     string  `json:"ordType"` // Order type  'limit': Limit order  'post_only': Post-only order 'ioc': Immediate-or-cancel order
	Size          float64 `json:"sz,string"`
	Price         float64 `json:"px,string"`
	Tag           string  `json:"tag,omitempty"`
}

// Validate checks if the parameters are valid
func (arg *SpreadOrderParam) Validate() error {
	if arg == nil {
		return fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.SpreadID == "" {
		return fmt.Errorf("%w, spread ID missing", errMissingInstrumentID)
	}
	if arg.OrderType == "" {
		return fmt.Errorf("%w spread order type is required", order.ErrTypeIsInvalid)
	}
	if arg.Size <= 0 {
		return order.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return order.ErrPriceBelowMin
	}
	arg.Side = strings.ToLower(arg.Side)
	switch arg.Side {
	case order.Buy.Lower(), order.Sell.Lower():
	default:
		return fmt.Errorf("%w %s", order.ErrSideIsInvalid, arg.Side)
	}
	return nil
}

// SpreadOrderResponse represents a spread create order response
type SpreadOrderResponse struct {
	StatusCode    int64  `json:"sCode,string"` // Anything above 0 is an error with an attached message
	StatusMessage string `json:"sMsg"`
	ClientOrderID string `json:"clOrdId"`
	OrderID       string `json:"ordId"`
	Tag           string `json:"tag"`

	// Added when amending spread order through websocket
	RequestID string `json:"reqId"`
}

func (arg *SpreadOrderResponse) Error() error {
	return getStatusError(arg.StatusCode, arg.StatusMessage)
}

// AmendSpreadOrderParam holds amend parameters for spread order
type AmendSpreadOrderParam struct {
	OrderID       string  `json:"ordId"`
	ClientOrderID string  `json:"clOrdId"`
	RequestID     string  `json:"reqId"`
	NewSize       float64 `json:"newSz,omitempty,string"`
	NewPrice      float64 `json:"newPx,omitempty,string"`
}

// SpreadOrder holds spread order details
type SpreadOrder struct {
	TradeID           string       `json:"tradeId"`
	InstrumentID      string       `json:"instId"`
	OrderID           string       `json:"ordId"`
	SpreadID          string       `json:"sprdId"`
	ClientOrderID     string       `json:"clOrdId"`
	Tag               string       `json:"tag"`
	Price             types.Number `json:"px"`
	Size              types.Number `json:"sz"`
	OrderType         string       `json:"ordType"`
	Side              string       `json:"side"`
	FillSize          types.Number `json:"fillSz"`
	FillPrice         types.Number `json:"fillPx"`
	AccFillSize       types.Number `json:"accFillSz"`
	PendingFillSize   types.Number `json:"pendingFillSz"`
	PendingSettleSize types.Number `json:"pendingSettleSz"`
	CanceledSize      types.Number `json:"canceledSz"`
	State             string       `json:"state"`
	AveragePrice      types.Number `json:"avgPx"`
	CancelSource      string       `json:"cancelSource"`
	UpdateTime        types.Time   `json:"uTime"`
	CreationTime      types.Time   `json:"cTime"`
}

// SpreadTrade holds spread trade transaction instance
type SpreadTrade struct {
	SpreadID      string       `json:"sprdId"`
	TradeID       string       `json:"tradeId"`
	OrderID       string       `json:"ordId"`
	ClientOrderID string       `json:"clOrdId"`
	Tag           string       `json:"tag"`
	FillPrice     types.Number `json:"fillPx"`
	FillSize      types.Number `json:"fillSz"`
	State         string       `json:"state"`
	Side          string       `json:"side"`
	ExecType      string       `json:"execType"`
	Timestamp     types.Time   `json:"ts"`
	Legs          []struct {
		InstrumentID string       `json:"instId"`
		Price        types.Number `json:"px"`
		Size         types.Number `json:"sz"`
		Side         string       `json:"side"`
		Fee          types.Number `json:"fee"`
		FeeCurrency  string       `json:"feeCcy"`
		TradeID      string       `json:"tradeId"`
	} `json:"legs"`
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// SpreadInstrument retrieve all available spreads based on the request parameters
type SpreadInstrument struct {
	SpreadID      currency.Pair `json:"sprdId"`
	SpreadType    string        `json:"sprdType"`
	State         string        `json:"state"`
	BaseCurrency  string        `json:"baseCcy"`
	SizeCurrency  string        `json:"szCcy"`
	QuoteCurrency string        `json:"quoteCcy"`
	TickSize      types.Number  `json:"tickSz"`
	MinSize       types.Number  `json:"minSz"`
	LotSize       types.Number  `json:"lotSz"`
	ListTime      types.Time    `json:"listTime"`
	Legs          []struct {
		InstrumentID string `json:"instId"`
		Side         string `json:"side"`
	} `json:"legs"`
	ExpTime    types.Time `json:"expTime"`
	UpdateTime types.Time `json:"uTime"`
}

// SpreadOrderbook holds spread orderbook information
type SpreadOrderbook struct {
	// Asks and Bids are [3]string; price, quantity, and # number of orders at the price
	Asks      [][]types.Number `json:"asks"`
	Bids      [][]types.Number `json:"bids"`
	Timestamp types.Time       `json:"ts"`
}

// SpreadTicker represents a ticker instance
type SpreadTicker struct {
	SpreadID     string       `json:"sprdId"`
	Last         types.Number `json:"last"`
	LastSize     types.Number `json:"lastSz"`
	AskPrice     types.Number `json:"askPx"`
	AskSize      types.Number `json:"askSz"`
	BidPrice     types.Number `json:"bidPx"`
	BidSize      types.Number `json:"bidSz"`
	Open24Hour   types.Number `json:"open24h"`
	High24Hour   types.Number `json:"high24h"`
	Low24Hour    types.Number `json:"low24h"`
	Volume24Hour types.Number `json:"vol24h"`
	Timestamp    types.Time   `json:"ts"`
}

// SpreadPublicTradeItem represents publicly available trade order instance
type SpreadPublicTradeItem struct {
	SprdID    string       `json:"sprdId"`
	Side      string       `json:"side"`
	Size      types.Number `json:"sz"`
	Price     types.Number `json:"px"`
	TradeID   string       `json:"tradeId"`
	Timestamp types.Time   `json:"ts"`
}

// SpreadCandlestick represents a candlestick instance
type SpreadCandlestick struct {
	Timestamp types.Time
	Open      types.Number
	High      types.Number
	Low       types.Number
	Close     types.Number
	Volume    types.Number
	Confirm   types.Number
}

// UnmarshalJSON unmarshals the JSON data into a SpreadCandlestick struct
func (s *SpreadCandlestick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&s.Timestamp, &s.Open, &s.High, &s.Low, &s.Close, &s.Volume, &s.Confirm})
}

// UnitConvertResponse unit convert response
type UnitConvertResponse struct {
	InstrumentID string       `json:"instId"`
	Price        types.Number `json:"px"`
	Size         types.Number `json:"sz"`
	ConvertType  types.Number `json:"type"`
	Unit         string       `json:"unit"`
}

// OptionTickBand holds option band information
type OptionTickBand struct {
	InstrumentType   string `json:"instType"`
	InstrumentFamily string `json:"instFamily"`
	TickBand         []struct {
		MinPrice types.Number `json:"minPx"`
		MaxPrice types.Number `json:"maxPx"`
		TickSize types.Number `json:"tickSz"`
	} `json:"tickBand"`
}

// Websocket Models

// WebsocketEventRequest contains event data for a websocket channel
type WebsocketEventRequest struct {
	Operation string               `json:"op"`   // 1--subscribe 2--unsubscribe 3--login
	Arguments []WebsocketLoginData `json:"args"` // args: the value is the channel name, which can be one or more channels
}

// WebsocketLoginData represents the websocket login data input json data
type WebsocketLoginData struct {
	APIKey     string `json:"apiKey"`
	Passphrase string `json:"passphrase"`
	Timestamp  int64  `json:"timestamp,string"`
	Sign       string `json:"sign"`
}

// SubscriptionInfo holds the channel and instrument IDs
type SubscriptionInfo struct {
	Channel          string        `json:"channel,omitempty"`
	InstrumentID     currency.Pair `json:"instId,omitzero"`
	InstrumentFamily string        `json:"instFamily,omitempty"`
	InstrumentType   string        `json:"instType,omitempty"`
	Underlying       string        `json:"uly,omitempty"`
	UID              string        `json:"uid,omitempty"` // user identifier

	// For Algo Orders
	AlgoID   string `json:"algoId,omitempty"`
	Currency string `json:"ccy,omitempty"`
	SpreadID string `json:"sprdId,omitempty"`
}

// WSSubscriptionInformationList websocket subscription and unsubscription operation inputs
type WSSubscriptionInformationList struct {
	Operation string             `json:"op"`
	Arguments []SubscriptionInfo `json:"args"`
}

// SpreadOrderInfo holds spread order response information
type SpreadOrderInfo struct {
	ClientOrderID string `json:"clOrdId"`
	OrderID       string `json:"ordId"`
	Tag           string `json:"tag"`
	StatusCode    string `json:"sCode"`
	StatusMessage string `json:"sMsg"`
}

type wsIncomingData struct {
	Event      string           `json:"event"`
	Argument   SubscriptionInfo `json:"arg"`
	StatusCode string           `json:"code"`
	Message    string           `json:"msg"`

	// For Websocket Trading Endpoints websocket responses
	ID        string          `json:"id"`
	Operation string          `json:"op"`
	Data      json.RawMessage `json:"data"`
}

// WSInstrumentResponse represents websocket instruments push message
type WSInstrumentResponse struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []Instrument     `json:"data"`
}

// WSTickerResponse represents websocket ticker response
type WSTickerResponse struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []TickerResponse `json:"data"`
}

// WSOpenInterestResponse represents an open interest instance
type WSOpenInterestResponse struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []OpenInterest   `json:"data"`
}

// WsOperationInput for all websocket request inputs
type WsOperationInput struct {
	ID        string `json:"id"`
	Operation string `json:"op"`
	Arguments any    `json:"args"`
}

// WsOrderActionResponse holds websocket response Amendment request
type WsOrderActionResponse struct {
	ID        string      `json:"id"`
	Operation string      `json:"op"`
	Data      []OrderData `json:"data"`
	Code      string      `json:"code"`
	Msg       string      `json:"msg"`
}

// SubscriptionOperationInput represents the account channel input data
type SubscriptionOperationInput struct {
	Operation string             `json:"op"`
	Arguments []SubscriptionInfo `json:"args"`
}

// WsAccountChannelPushData holds the websocket push data following the subscription
type WsAccountChannelPushData struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []Account        `json:"data,omitempty"`
}

// WsPositionResponse represents pushed position data through the websocket channel
type WsPositionResponse struct {
	Argument  SubscriptionInfo  `json:"arg"`
	Arguments []AccountPosition `json:"data"`
}

// PositionDataDetail position data information for the websocket push data
type PositionDataDetail struct {
	PositionID       string       `json:"posId"`
	TradeID          string       `json:"tradeId"`
	InstrumentID     string       `json:"instId"`
	InstrumentType   string       `json:"instType"`
	MarginMode       string       `json:"mgnMode"`
	PositionSide     string       `json:"posSide"`
	Position         string       `json:"pos"`
	Currency         string       `json:"ccy"`
	PositionCurrency string       `json:"posCcy"`
	AveragePrice     types.Number `json:"avgPx"`
	UpdateTime       types.Time   `json:"uTIme"`
}

// BalanceData represents currency and it's Cash balance with the update time
type BalanceData struct {
	Currency    string       `json:"ccy"`
	CashBalance types.Number `json:"cashBal"`
	UpdateTime  types.Time   `json:"uTime"`
}

// BalanceAndPositionData represents balance and position data with the push time
type BalanceAndPositionData struct {
	PushTime     types.Time           `json:"pTime"`
	EventType    string               `json:"eventType"`
	BalanceData  []BalanceData        `json:"balData"`
	PositionData []PositionDataDetail `json:"posData"`
}

// WsBalanceAndPosition websocket push data for lis of BalanceAndPosition information
type WsBalanceAndPosition struct {
	Argument SubscriptionInfo         `json:"arg"`
	Data     []BalanceAndPositionData `json:"data"`
}

// WsOrder represents a websocket order
type WsOrder struct {
	PendingOrderItem
	AmendResult     string       `json:"amendResult"`
	Code            string       `json:"code"`
	ExecType        string       `json:"execType"`
	FillFee         types.Number `json:"fillFee"`
	FillFeeCurrency string       `json:"fillFeeCcy"`
	FillNotionalUsd types.Number `json:"fillNotionalUsd"`
	Msg             string       `json:"msg"`
	NotionalUSD     types.Number `json:"notionalUsd"`
	ReduceOnly      bool         `json:"reduceOnly,string"`
	RequestID       string       `json:"reqId"`
}

// WsOrderResponse holds order list push data through the websocket connection
type WsOrderResponse struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsOrder        `json:"data"`
}

// WsAlgoOrder algo order detailed data
type WsAlgoOrder struct {
	Argument SubscriptionInfo    `json:"arg"`
	Data     []WsAlgoOrderDetail `json:"data"`
}

// WsAlgoOrderDetail algo order response pushed through the websocket conn
type WsAlgoOrderDetail struct {
	InstrumentType             string       `json:"instType"`
	InstrumentID               string       `json:"instId"`
	OrderID                    string       `json:"ordId"`
	Currency                   string       `json:"ccy"`
	AlgoID                     string       `json:"algoId"`
	Price                      types.Number `json:"px"`
	Size                       types.Number `json:"sz"`
	TradeMode                  string       `json:"tdMode"`
	TargetCurrency             string       `json:"tgtCcy"`
	NotionalUsd                string       `json:"notionalUsd"`
	OrderType                  string       `json:"ordType"`
	Side                       order.Side   `json:"side"`
	PositionSide               string       `json:"posSide"`
	State                      string       `json:"state"`
	Leverage                   string       `json:"lever"`
	TakeProfitTriggerPrice     types.Number `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string       `json:"tpTriggerPxType"`
	TakeProfitOrdPrice         types.Number `json:"tpOrdPx"`
	StopLossTriggerPrice       types.Number `json:"slTriggerPx"`
	StopLossTriggerPriceType   string       `json:"slTriggerPxType"`
	TriggerPrice               types.Number `json:"triggerPx"`
	TriggerPriceType           string       `json:"triggerPxType"`
	OrderPrice                 types.Number `json:"ordPx"`
	ActualSize                 types.Number `json:"actualSz"`
	ActualPrice                types.Number `json:"actualPx"`
	Tag                        string       `json:"tag"`
	ActualSide                 string       `json:"actualSide"`
	TriggerTime                types.Time   `json:"triggerTime"`
	CreationTime               types.Time   `json:"cTime"`
}

// WsAdvancedAlgoOrder advanced algo order response
type WsAdvancedAlgoOrder struct {
	Argument SubscriptionInfo            `json:"arg"`
	Data     []WsAdvancedAlgoOrderDetail `json:"data"`
}

// WsAdvancedAlgoOrderDetail advanced algo order response pushed through the websocket conn
type WsAdvancedAlgoOrderDetail struct {
	ActualPrice            types.Number `json:"actualPx"`
	ActualSide             string       `json:"actualSide"`
	ActualSize             types.Number `json:"actualSz"`
	AlgoID                 string       `json:"algoId"`
	Currency               string       `json:"ccy"`
	Count                  string       `json:"count"`
	InstrumentID           string       `json:"instId"`
	InstrumentType         string       `json:"instType"`
	Leverage               types.Number `json:"lever"`
	NotionalUSD            types.Number `json:"notionalUsd"`
	OrderPrice             types.Number `json:"ordPx"`
	OrdType                string       `json:"ordType"`
	PositionSide           string       `json:"posSide"`
	PriceLimit             types.Number `json:"pxLimit"`
	PriceSpread            types.Number `json:"pxSpread"`
	PriceVariation         string       `json:"pxVar"`
	Side                   order.Side   `json:"side"`
	StopLossOrderPrice     types.Number `json:"slOrdPx"`
	StopLossTriggerPrice   types.Number `json:"slTriggerPx"`
	State                  string       `json:"state"`
	Size                   types.Number `json:"sz"`
	SizeLimit              types.Number `json:"szLimit"`
	TradeMode              string       `json:"tdMode"`
	TimeInterval           string       `json:"timeInterval"`
	TakeProfitOrderPrice   types.Number `json:"tpOrdPx"`
	TakeProfitTriggerPrice types.Number `json:"tpTriggerPx"`
	Tag                    string       `json:"tag"`
	TriggerPrice           types.Number `json:"triggerPx"`
	CallbackRatio          types.Number `json:"callbackRatio"`
	CallbackSpread         string       `json:"callbackSpread"`
	ActivePrice            types.Number `json:"activePx"`
	MoveTriggerPrice       types.Number `json:"moveTriggerPx"`
	CreationTime           types.Time   `json:"cTime"`
	PushTime               types.Time   `json:"pTime"`
	TriggerTime            types.Time   `json:"triggerTime"`
}

// WsGreeks greeks push data with the subscription info through websocket channel
type WsGreeks struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsGreekData    `json:"data"`
}

// WsGreekData greeks push data through websocket channel
type WsGreekData struct {
	ThetaBS   string     `json:"thetaBS"`
	ThetaPA   string     `json:"thetaPA"`
	DeltaBS   string     `json:"deltaBS"`
	DeltaPA   string     `json:"deltaPA"`
	GammaBS   string     `json:"gammaBS"`
	GammaPA   string     `json:"gammaPA"`
	VegaBS    string     `json:"vegaBS"`
	VegaPA    string     `json:"vegaPA"`
	Currency  string     `json:"ccy"`
	Timestamp types.Time `json:"ts"`
}

// WsRFQ represents websocket push data for "rfqs" subscription
type WsRFQ struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsRFQData      `json:"data"`
}

// WsRFQData represents rfq order response data streamed through the websocket channel
type WsRFQData struct {
	CreationTime   time.Time     `json:"cTime"`
	UpdateTime     time.Time     `json:"uTime"`
	TraderCode     string        `json:"traderCode"`
	RFQID          string        `json:"rfqId"`
	ClientRFQID    string        `json:"clRfqId"`
	State          string        `json:"state"`
	ValidUntil     string        `json:"validUntil"`
	Counterparties []string      `json:"counterparties"`
	Legs           []RFQOrderLeg `json:"legs"`
}

// WsQuote represents websocket push data for "quotes" subscription
type WsQuote struct {
	Arguments SubscriptionInfo `json:"arg"`
	Data      []WsQuoteData    `json:"data"`
}

// WsQuoteData represents a single quote order information
type WsQuoteData struct {
	ValidUntil    types.Time `json:"validUntil"`
	UpdatedTime   types.Time `json:"uTime"`
	CreationTime  types.Time `json:"cTime"`
	Legs          []OrderLeg `json:"legs"`
	QuoteID       string     `json:"quoteId"`
	RFQID         string     `json:"rfqId"`
	TraderCode    string     `json:"traderCode"`
	QuoteSide     string     `json:"quoteSide"`
	State         string     `json:"state"`
	ClientQuoteID string     `json:"clQuoteId"`
}

// WsStructureBlocTrade represents websocket push data for "struc-block-trades" subscription
type WsStructureBlocTrade struct {
	Argument SubscriptionInfo       `json:"arg"`
	Data     []WsBlockTradeResponse `json:"data"`
}

// WsBlockTradeResponse represents a structure block order information
type WsBlockTradeResponse struct {
	CreationTime    types.Time `json:"cTime"`
	RFQID           string     `json:"rfqId"`
	ClientRFQID     string     `json:"clRfqId"`
	QuoteID         string     `json:"quoteId"`
	ClientQuoteID   string     `json:"clQuoteId"`
	BlockTradeID    string     `json:"blockTdId"`
	TakerTraderCode string     `json:"tTraderCode"`
	MakerTraderCode string     `json:"mTraderCode"`
	Legs            []OrderLeg `json:"legs"`
}

// WsSpotGridAlgoOrder represents websocket push data for "struc-block-trades" subscription
type WsSpotGridAlgoOrder struct {
	Argument SubscriptionInfo   `json:"arg"`
	Data     []SpotGridAlgoData `json:"data"`
}

// SpotGridAlgoData represents spot grid algo orders
type SpotGridAlgoData struct {
	AlgoID          string       `json:"algoId"`
	AlgoOrderType   string       `json:"algoOrdType"`
	AnnualizedRate  types.Number `json:"annualizedRate"`
	ArbitrageNumber types.Number `json:"arbitrageNum"`
	BaseSize        types.Number `json:"baseSz"`
	// Algo order stop reason 0: None 1: Manual stop 2: Take profit
	// 3: Stop loss 4: Risk control 5: delivery
	CancelType           string       `json:"cancelType"`
	CurBaseSize          types.Number `json:"curBaseSz"`
	CurQuoteSize         types.Number `json:"curQuoteSz"`
	FloatProfit          types.Number `json:"floatProfit"`
	GridNumber           string       `json:"gridNum"`
	GridProfit           types.Number `json:"gridProfit"`
	InstrumentID         string       `json:"instId"`
	InstrumentType       string       `json:"instType"`
	Investment           types.Number `json:"investment"`
	MaximumPrice         types.Number `json:"maxPx"`
	MinimumPrice         types.Number `json:"minPx"`
	PerMaximumProfitRate types.Number `json:"perMaxProfitRate"`
	PerMinimumProfitRate types.Number `json:"perMinProfitRate"`
	ProfitAndLossRatio   types.Number `json:"pnlRatio"`
	QuoteSize            types.Number `json:"quoteSz"`
	RunPrice             types.Number `json:"runPx"`
	RunType              string       `json:"runType"`
	SingleAmount         types.Number `json:"singleAmt"`
	StopLossTriggerPrice types.Number `json:"slTriggerPx"`
	State                string       `json:"state"`
	// Stop result of spot grid
	// 0: default, 1: Successful selling of currency at market price,
	// -1: Failed to sell currency at market price
	StopResult string `json:"stopResult"`
	// Stop type Spot grid 1: Sell base currency 2: Keep base currency
	// Contract grid 1: Market Close All positions 2: Keep positions
	StopType               string       `json:"stopType"`
	TotalAnnualizedRate    types.Number `json:"totalAnnualizedRate"`
	TotalProfitAndLoss     types.Number `json:"totalPnl"`
	TakeProfitTriggerPrice types.Number `json:"tpTriggerPx"`
	TradeNum               types.Number `json:"tradeNum"`
	TriggerTime            types.Time   `json:"triggerTime"`
	CreationTime           types.Time   `json:"cTime"`
	PushTime               types.Time   `json:"pTime"`
	UpdateTime             types.Time   `json:"uTime"`
}

// WsContractGridAlgoOrder represents websocket push data for "grid-orders-contract" subscription
type WsContractGridAlgoOrder struct {
	Argument SubscriptionInfo        `json:"arg"`
	Data     []ContractGridAlgoOrder `json:"data"`
}

// ContractGridAlgoOrder represents contract grid algo order
type ContractGridAlgoOrder struct {
	ActualLever            string       `json:"actualLever"`
	AlgoID                 string       `json:"algoId"`
	AlgoOrderType          string       `json:"algoOrdType"`
	AnnualizedRate         types.Number `json:"annualizedRate"`
	ArbitrageNumber        types.Number `json:"arbitrageNum"`
	BasePosition           bool         `json:"basePos"`
	CancelType             string       `json:"cancelType"`
	Direction              string       `json:"direction"`
	Equity                 types.Number `json:"eq"`
	FloatProfit            types.Number `json:"floatProfit"`
	GridQuantity           types.Number `json:"gridNum"`
	GridProfit             types.Number `json:"gridProfit"`
	InstrumentID           string       `json:"instId"`
	InstrumentType         string       `json:"instType"`
	Investment             string       `json:"investment"`
	Leverage               string       `json:"lever"`
	LiqPrice               types.Number `json:"liqPx"`
	MaxPrice               types.Number `json:"maxPx"`
	MinPrice               types.Number `json:"minPx"`
	CreationTime           types.Time   `json:"cTime"`
	PushTime               types.Time   `json:"pTime"`
	PerMaxProfitRate       types.Number `json:"perMaxProfitRate"`
	PerMinProfitRate       types.Number `json:"perMinProfitRate"`
	ProfitAndLossRatio     types.Number `json:"pnlRatio"`
	RunPrice               types.Number `json:"runPx"`
	RunType                string       `json:"runType"`
	SingleAmount           types.Number `json:"singleAmt"`
	SlTriggerPrice         types.Number `json:"slTriggerPx"`
	State                  string       `json:"state"`
	StopType               string       `json:"stopType"`
	Size                   types.Number `json:"sz"`
	Tag                    string       `json:"tag"`
	TotalAnnualizedRate    string       `json:"totalAnnualizedRate"`
	TotalProfitAndLoss     types.Number `json:"totalPnl"`
	TakeProfitTriggerPrice types.Number `json:"tpTriggerPx"`
	TradeNumber            string       `json:"tradeNum"`
	TriggerTime            types.Time   `json:"triggerTime"`
	UpdateTime             types.Time   `json:"uTime"`
	Underlying             string       `json:"uly"`
}

// WsGridSubOrderData to retrieve grid sub orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing order
type WsGridSubOrderData struct {
	Argument SubscriptionInfo   `json:"arg"`
	Data     []GridSubOrderData `json:"data"`
}

// GridSubOrderData represents a single sub order detailed info
type GridSubOrderData struct {
	AccumulatedFillSize types.Number `json:"accFillSz"`
	AlgoID              string       `json:"algoId"`
	AlgoOrderType       string       `json:"algoOrdType"`
	AveragePrice        types.Number `json:"avgPx"`
	CreationTime        types.Time   `json:"cTime"`
	ContractValue       string       `json:"ctVal"`
	Fee                 types.Number `json:"fee"`
	FeeCurrency         string       `json:"feeCcy"`
	GroupID             string       `json:"groupId"`
	InstrumentID        string       `json:"instId"`
	InstrumentType      string       `json:"instType"`
	Leverage            types.Number `json:"lever"`
	OrderID             string       `json:"ordId"`
	OrderType           string       `json:"ordType"`
	PushTime            types.Time   `json:"pTime"`
	ProfitAndLoss       types.Number `json:"pnl"`
	PositionSide        string       `json:"posSide"`
	Price               types.Number `json:"px"`
	Side                order.Side   `json:"side"`
	State               string       `json:"state"`
	Size                types.Number `json:"sz"`
	Tag                 string       `json:"tag"`
	TradeMode           string       `json:"tdMode"`
	UpdateTime          types.Time   `json:"uTime"`
}

// WsTradeOrder represents a trade push data response as a result subscription to "trades" channel
type WsTradeOrder struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []TradeResponse  `json:"data"`
}

// WsMarkPrice represents an estimated mark price push data as a result of subscription to "mark-price" channel
type WsMarkPrice struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []MarkPrice      `json:"data"`
}

// WsDeliveryEstimatedPrice represents an estimated delivery/exercise price push data as a result of subscription to "estimated-price" channel
type WsDeliveryEstimatedPrice struct {
	Argument SubscriptionInfo         `json:"arg"`
	Data     []DeliveryEstimatedPrice `json:"data"`
}

// CandlestickMarkPrice represents candlestick mark price push data as a result of  subscription to "mark-price-candle*" channel
type CandlestickMarkPrice struct {
	Timestamp    time.Time `json:"ts"`
	OpenPrice    float64   `json:"o"`
	HighestPrice float64   `json:"h"`
	LowestPrice  float64   `json:"l"`
	ClosePrice   float64   `json:"s"`
}

// WsOrderBook order book represents order book push data which is returned as a result of subscription to "books*" channel
type WsOrderBook struct {
	Argument SubscriptionInfo  `json:"arg"`
	Action   string            `json:"action"`
	Data     []WsOrderBookData `json:"data"`
}

// WsOptionTrades represents option trade data
type WsOptionTrades struct {
	Arg  SubscriptionInfo `json:"arg"`
	Data []PublicTrade    `json:"data"`
}

// PublicTrade represents public trade item for option, block, and others
type PublicTrade struct {
	FillVolume       types.Number `json:"fillVol"`
	ForwardPrice     types.Number `json:"fwdPx"`
	IndexPrice       types.Number `json:"idxPx"`
	InstrumentFamily string       `json:"instFamily"`
	InstrumentID     string       `json:"instId"`
	MarkPrice        types.Number `json:"markPx"`
	OptionType       string       `json:"optType"`
	Price            types.Number `json:"px"`
	Side             string       `json:"side"`
	Size             types.Number `json:"sz"`
	TradeID          string       `json:"tradeId"`
	Timestamp        types.Time   `json:"ts"`
}

// WsOrderBookData represents a book order push data
type WsOrderBookData struct {
	Asks      [][4]types.Number `json:"asks"`
	Bids      [][4]types.Number `json:"bids"`
	Timestamp types.Time        `json:"ts"`
	Checksum  int32             `json:"checksum,omitempty"`
}

// WsOptionSummary represents option summary
type WsOptionSummary struct {
	Argument SubscriptionInfo           `json:"arg"`
	Data     []OptionMarketDataResponse `json:"data"`
}

// WsFundingRate represents websocket push data funding rate response
type WsFundingRate struct {
	Argument SubscriptionInfo      `json:"arg"`
	Data     []FundingRateResponse `json:"data"`
}

// WsIndexTicker represents websocket push data index ticker response
type WsIndexTicker struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []IndexTicker    `json:"data"`
}

// WsSystemStatusResponse represents websocket push data system status push data
type WsSystemStatusResponse struct {
	Argument SubscriptionInfo       `json:"arg"`
	Data     []SystemStatusResponse `json:"data"`
}

// WsPublicTradesResponse represents websocket push data of structured block trades as a result of subscription to "public-struc-block-trades"
type WsPublicTradesResponse struct {
	Argument SubscriptionInfo       `json:"arg"`
	Data     []PublicTradesResponse `json:"data"`
}

// WsBlockTicker represents websocket push data as a result of subscription to channel "block-tickers"
type WsBlockTicker struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []BlockTicker    `json:"data"`
}

// PublicBlockTrades holds public block trades
type PublicBlockTrades struct {
	Arg  SubscriptionInfo `json:"arg"`
	Data []PublicTrade    `json:"data"`
}

// PMLimitationResponse represents portfolio margin mode limitation for specific underlying
type PMLimitationResponse struct {
	MaximumSize      types.Number `json:"maxSz"`
	PositionType     string       `json:"postType"`
	Underlying       string       `json:"uly"`
	InstrumentFamily string       `json:"instFamily"`
}

// RiskOffsetType represents risk offset type value
type RiskOffsetType struct {
	Type string `json:"type"`
}

// AutoLoan holds auto loan information
type AutoLoan struct {
	AutoLoan bool `json:"autoLoan"`
}

// AccountMode holds account mode
type AccountMode struct {
	IsoMode string `json:"isoMode"`
}

// EasyConvertDetail represents easy convert currencies list and their detail
type EasyConvertDetail struct {
	FromData   []EasyConvertFromData `json:"fromData"`
	ToCurrency []string              `json:"toCcy"`
}

// EasyConvertFromData represents convert currency from detail
type EasyConvertFromData struct {
	FromAmount   types.Number `json:"fromAmt"`
	FromCurrency string       `json:"fromCcy"`
}

// PlaceEasyConvertParam represents easy convert request params
type PlaceEasyConvertParam struct {
	FromCurrency []string `json:"fromCcy"`
	ToCurrency   string   `json:"toCcy"`
	Source       string   `json:"source,omitempty"`
}

// EasyConvertItem represents easy convert place order response
type EasyConvertItem struct {
	FilFromSize  types.Number `json:"fillFromSz"`
	FillToSize   types.Number `json:"fillToSz"`
	FromCurrency string       `json:"fromCcy"`
	Status       string       `json:"status"`
	ToCurrency   string       `json:"toCcy"`
	UpdateTime   types.Time   `json:"uTime"`
	Account      string       `json:"acct"`
}

// TradeOneClickRepayParam represents click one repay param
type TradeOneClickRepayParam struct {
	DebtCurrency  []string `json:"debtCcy"`
	RepayCurrency string   `json:"repayCcy"`
}

// CurrencyOneClickRepay represents the currency used for one-click repayment.
type CurrencyOneClickRepay struct {
	DebtCurrency  string       `json:"debtCcy"`
	FillFromSize  types.Number `json:"fillFromSz"`
	FillRepaySize types.Number `json:"fillRepaySz"`
	FillDebtSize  types.Number `json:"fillDebtSz"`
	FillToSize    types.Number `json:"fillToSz"`
	RepayCurrency string       `json:"repayCcy"`
	Status        string       `json:"status"`
	UpdateTime    types.Time   `json:"uTime"`
}

// CancelMMPResponse holds the result of a cancel MMP response.
type CancelMMPResponse struct {
	Result bool `json:"result"`
}

// CancelResponse represents a pending order cancellation response
type CancelResponse struct {
	TriggerTime types.Time `json:"triggerTime"` // The time the cancellation is triggered. triggerTime=0 means Cancel All After is disabled.
	Tag         string     `json:"tag"`
	Timestamp   types.Time `json:"ts"` // The time the request is sent.
}

// SetQuoteProductParam represents set quote product request param
type SetQuoteProductParam struct {
	InstrumentType string                   `json:"instType"`
	Data           []MakerInstrumentSetting `json:"data"`
}

// MakerInstrumentSetting represents set quote product setting info
type MakerInstrumentSetting struct {
	Underlying     string       `json:"uly"`
	InstrumentID   string       `json:"instId"`
	MaxBlockSize   types.Number `json:"maxBlockSz"`
	MakerPriceBand types.Number `json:"makerPxBand"`
}

// SetQuoteProductsResult represents set quote products result
type SetQuoteProductsResult struct {
	Result bool `json:"result"`
}

// SubAccountAPIKeyParam represents Reset the APIKey of a sub-account request param
type SubAccountAPIKeyParam struct {
	SubAccountName   string   `json:"subAcct"`         // Sub-account name
	APIKey           string   `json:"apiKey"`          // Sub-accountAPI public key
	Label            string   `json:"label,omitempty"` // Sub-account APIKey label
	APIKeyPermission string   `json:"perm,omitempty"`  // Sub-account APIKey permissions
	IP               string   `json:"ip,omitempty"`    // Sub-account APIKey linked IP addresses, separate with commas if more than
	Permissions      []string `json:"-"`
}

// SubAccountAPIKeyResponse represents sub-account api key reset response
type SubAccountAPIKeyResponse struct {
	IP               string     `json:"ip"`
	SubAccountName   string     `json:"subAcct"`
	APIKey           string     `json:"apiKey"`
	Label            string     `json:"label"`
	APIKeyPermission string     `json:"perm"`
	Timestamp        types.Time `json:"ts"`
}

// MarginBalanceParam represents compute margin balance request param
type MarginBalanceParam struct {
	AlgoID                  string  `json:"algoId"`
	AdjustMarginBalanceType string  `json:"type"`
	Amount                  float64 `json:"amt,string"`               // Adjust margin balance amount Either amt or percent is required.
	Percentage              float64 `json:"percent,string,omitempty"` // Adjust margin balance percentage, used In Adjusting margin balance
}

// ComputeMarginBalance represents compute margin amount request response
type ComputeMarginBalance struct {
	Leverage      types.Number `json:"lever"`
	MaximumAmount types.Number `json:"maxAmt"`
}

// AdjustMarginBalanceResponse represents algo ID for response for margin balance adjust request
type AdjustMarginBalanceResponse struct {
	AlgoID string `json:"algoId"`
}

// GridAIParameterResponse represents gri AI parameter response
type GridAIParameterResponse struct {
	AlgoOrderType        string       `json:"algoOrdType"`
	AnnualizedRate       string       `json:"annualizedRate"`
	Currency             string       `json:"ccy"`
	Direction            string       `json:"direction"`
	Duration             string       `json:"duration"`
	GridNum              string       `json:"gridNum"`
	InstrumentID         string       `json:"instId"`
	Leverage             types.Number `json:"lever"`
	MaximumPrice         types.Number `json:"maxPx"`
	MinimumInvestment    types.Number `json:"minInvestment"`
	MinimumPrice         types.Number `json:"minPx"`
	PerMaximumProfitRate types.Number `json:"perMaxProfitRate"`
	PerMinimumProfitRate types.Number `json:"perMinProfitRate"`
	RunType              string       `json:"runType"`
}

// InvestmentData holds investment data parameter
type InvestmentData struct {
	Amount   float64       `json:"amt,string"`
	Currency currency.Code `json:"ccy"`
}

// ComputeInvestmentDataParam holds parameter values for computing investment data
type ComputeInvestmentDataParam struct {
	InstrumentID    string           `json:"instId"`
	AlgoOrderType   string           `json:"algoOrdType"` // Algo order type 'grid': Spot grid 'contract_grid': Contract grid
	GridNumber      float64          `json:"gridNum,string"`
	Direction       string           `json:"direction,omitempty"` // Contract grid type 'long','short', 'neutral' Only applicable to contract grid
	MaxPrice        float64          `json:"maxPx,string"`
	MinPrice        float64          `json:"minPx,string"`
	RunType         string           `json:"runType"` // Grid type 1: Arithmetic, 2: Geometric
	Leverage        float64          `json:"lever,omitempty,string"`
	BasePosition    bool             `json:"basePos,omitempty"`
	InvestmentType  string           `json:"investmentType,omitempty"`
	TriggerStrategy string           `json:"triggerStrategy,omitempty"` // TriggerStrategy possible values are 'instant', 'price', 'rsi'
	InvestmentData  []InvestmentData `json:"investmentData,omitempty"`
}

// InvestmentResult holds investment response
type InvestmentResult struct {
	MinInvestmentData []InvestmentData `json:"minInvestmentData"`
	SingleAmount      types.Number     `json:"singleAmt"`
}

// RSIBacktestingResponse holds response for relative strength index(RSI) backtesting
type RSIBacktestingResponse struct {
	TriggerNumber string `json:"triggerNum"`
}

// SignalBotOrderDetail holds detail of signal bot order
type SignalBotOrderDetail struct {
	AlgoID               string       `json:"algoId"`
	ClientSuppliedAlgoID string       `json:"algoClOrdId"`
	AlgoOrderType        string       `json:"algoOrdType"`
	InstrumentType       string       `json:"instType"`
	InstrumentIDs        []string     `json:"instIds"`
	CreationTime         types.Time   `json:"cTime"`
	UpdateTime           types.Time   `json:"uTime"`
	State                string       `json:"state"`
	CancelType           string       `json:"cancelType"`
	TotalPNL             types.Number `json:"totalPnl"`
	ProfitAndLossRatio   types.Number `json:"pnlRatio"`
	TotalEq              types.Number `json:"totalEq"`
	FloatPNL             types.Number `json:"floatPnl"`
	FrozenBalance        types.Number `json:"frozenBal"`
	AvailableBalance     types.Number `json:"availBal"`
	Lever                types.Number `json:"lever"`
	InvestAmount         types.Number `json:"investAmt"`
	SubOrdType           string       `json:"subOrdType"`
	Ratio                types.Number `json:"ratio"`
	EntrySettingParam    struct {
		AllowMultipleEntry bool         `json:"allowMultipleEntry"`
		Amount             types.Number `json:"amt"`
		EntryType          string       `json:"entryType"`
		Ratio              types.Number `json:"ratio"`
	} `json:"entrySettingParam"`
	ExitSettingParam struct {
		StopLossPercentage   types.Number `json:"slPct"`
		TakeProfitPercentage types.Number `json:"tpPct"`
		TakeProfitSlType     string       `json:"tpSlType"`
	} `json:"exitSettingParam"`
	SignalChanID     string `json:"signalChanId"`
	SignalChanName   string `json:"signalChanName"`
	SignalSourceType string `json:"signalSourceType"`

	TotalPnlRatio types.Number `json:"totalPnlRatio"`
	RealizedPnl   types.Number `json:"realizedPnl"`
}

// SignalBotPosition holds signal bot position information
type SignalBotPosition struct {
	AutoDecreaseLine             string       `json:"adl"`
	AlgoClientOrderID            string       `json:"algoClOrdId"`
	AlgoID                       string       `json:"algoId"`
	AveragePrice                 types.Number `json:"avgPx"`
	CreationTime                 types.Time   `json:"cTime"`
	Currency                     string       `json:"ccy"`
	InitialMarginRequirement     string       `json:"imr"`
	InstrumentID                 string       `json:"instId"`
	InstrumentType               string       `json:"instType"`
	Last                         types.Number `json:"last"`
	Lever                        types.Number `json:"lever"`
	LiquidationPrice             types.Number `json:"liqPx"`
	MarkPrice                    types.Number `json:"markPx"`
	MarginMode                   string       `json:"mgnMode"`
	MgnRatio                     types.Number `json:"mgnRatio"` // Margin mode 'cross' 'isolated'
	MaintenanceMarginRequirement string       `json:"mmr"`
	NotionalUSD                  string       `json:"notionalUsd"`
	Position                     string       `json:"pos"`
	PositionSide                 string       `json:"posSide"` // Position side 'net'
	UpdateTime                   types.Time   `json:"uTime"`
	UnrealizedProfitAndLoss      string       `json:"upl"`
	UplRatio                     types.Number `json:"uplRatio"` // Unrealized profit and loss ratio
}

// SubOrder holds signal bot sub orders
type SubOrder struct {
	AccountFillSize   types.Number `json:"accFillSz"`
	AlgoClientOrderID string       `json:"algoClOrdId"`
	AlgoID            string       `json:"algoId"`
	AlgoOrdType       string       `json:"algoOrdType"`
	AveragePrice      types.Number `json:"avgPx"`
	CreationTime      types.Time   `json:"cTime"`
	Currency          string       `json:"ccy"`
	ClientOrderID     string       `json:"clOrdId"`
	CtVal             string       `json:"ctVal"`
	Fee               types.Number `json:"fee"`
	FeeCurrency       string       `json:"feeCcy"`
	InstrumentID      string       `json:"instId"`
	InstrumentType    string       `json:"instType"`
	Leverage          types.Number `json:"lever"`
	OrderID           string       `json:"ordId"`
	OrderType         string       `json:"ordType"`
	ProfitAndLoss     types.Number `json:"pnl"`
	PositionSide      string       `json:"posSide"`
	Price             types.Number `json:"px"`
	Side              string       `json:"side"`
	State             string       `json:"state"`
	Size              types.Number `json:"sz"`
	Tag               string       `json:"tag"`
	TdMode            string       `json:"tdMode"`
	UpdateTime        types.Time   `json:"uTime"`
}

// SignalBotEventHistory holds history information for signal bot
type SignalBotEventHistory struct {
	AlertMsg            time.Time  `json:"alertMsg"`
	AlgoID              string     `json:"algoId"`
	EventCreationTime   types.Time `json:"eventCtime"`
	EventProcessMessage string     `json:"eventProcessMsg"`
	EventStatus         string     `json:"eventStatus"`
	EventUtime          types.Time `json:"eventUtime"`
	EventType           string     `json:"eventType"`
	TriggeredOrdData    []struct {
		ClientOrderID string `json:"clOrdId"`
	} `json:"triggeredOrdData"`
}

// PlaceRecurringBuyOrderParam holds parameters for placing recurring order
type PlaceRecurringBuyOrderParam struct {
	Tag                       string              `json:"tag"`
	ClientSuppliedAlgoOrderID string              `json:"algoClOrdId"`
	StrategyName              string              `json:"stgyName"` // Custom name for trading bot
	Amount                    float64             `json:"amt,string"`
	RecurringList             []RecurringListItem `json:"recurringList"`
	Period                    string              `json:"period"` // Period 'monthly' 'weekly' 'daily'

	// Recurring buy date
	// When the period is monthly, the value range is an integer of [1,28]
	// When the period is weekly, the value range is an integer of [1,7]
	// When the period is daily, the value is 1
	RecurringDay       string `json:"recurringDay"`
	RecurringTime      int64  `json:"recurringTime,string"` // Recurring buy time, the value range is an integer of [0,23]
	TimeZone           string `json:"timeZone"`
	TradeMode          string `json:"tdMode"` // Trading mode Margin mode: 'cross' Non-Margin mode: 'cash'
	InvestmentCurrency string `json:"investmentCcy"`
}

// RecurringListItem holds recurring list item
type RecurringListItem struct {
	Currency currency.Code `json:"ccy"`
	Ratio    float64       `json:"ratio,string"`
}

// RecurringListItemDetailed holds a detailed instance of recurring list item
type RecurringListItemDetailed struct {
	AveragePrice types.Number `json:"avgPx"`
	Currency     string       `json:"ccy"`
	Profit       types.Number `json:"profit"`
	Price        types.Number `json:"px"`
	Ratio        types.Number `json:"ratio"`
	TotalAmount  types.Number `json:"totalAmt"`
}

// RecurringOrderResponse holds recurring order response
type RecurringOrderResponse struct {
	AlgoID            string `json:"algoId"`
	AlgoClientOrderID string `json:"algoClOrdId"`
	StatusCode        string `json:"sCode"`
	StatusMessage     string `json:"sMsg"`
}

// AmendRecurringOrderParam holds recurring order params
type AmendRecurringOrderParam struct {
	AlgoID       string `json:"algoId"`
	StrategyName string `json:"stgyName"`
}

// StopRecurringBuyOrder stop recurring order
type StopRecurringBuyOrder struct {
	AlgoID string `json:"algoId"`
}

// RecurringOrderItem holds recurring order info
type RecurringOrderItem struct {
	AlgoClOrdID        string              `json:"algoClOrdId"`
	AlgoID             string              `json:"algoId"`
	AlgoOrdType        string              `json:"algoOrdType"`
	Amount             types.Number        `json:"amt"`
	CreationTime       types.Time          `json:"cTime"`
	Cycles             string              `json:"cycles"`
	InstrumentType     string              `json:"instType"`
	InvestmentAmount   types.Number        `json:"investmentAmt"`
	InvestmentCurrency string              `json:"investmentCcy"`
	MarketCap          string              `json:"mktCap"`
	Period             string              `json:"period"`
	ProfitAndLossRatio types.Number        `json:"pnlRatio"`
	RecurringDay       string              `json:"recurringDay"`
	RecurringList      []RecurringListItem `json:"recurringList"`
	RecurringTime      string              `json:"recurringTime"`
	State              string              `json:"state"`
	StgyName           string              `json:"stgyName"`
	Tag                string              `json:"tag"`
	TimeZone           string              `json:"timeZone"`
	TotalAnnRate       types.Number        `json:"totalAnnRate"`
	TotalPnl           types.Number        `json:"totalPnl"`
	UpdateTime         types.Time          `json:"uTime"`
}

// RecurringOrderDeail holds detailed information about recurring order
type RecurringOrderDeail struct {
	RecurringListItem
	RecurringList []RecurringListItemDetailed `json:"recurringList"`
}

// RecurringBuySubOrder holds recurring buy sub order detail
type RecurringBuySubOrder struct {
	AccFillSize     types.Number `json:"accFillSz"`
	AlgoClientOrdID string       `json:"algoClOrdId"`
	AlgoID          string       `json:"algoId"`
	AlgoOrderType   string       `json:"algoOrdType"`
	AveragePrice    types.Number `json:"avgPx"`
	CreationTime    types.Time   `json:"cTime"`
	Fee             types.Number `json:"fee"`
	FeeCurrency     string       `json:"feeCcy"`
	InstrumentID    string       `json:"instId"`
	InstrumentType  string       `json:"instType"`
	OrderID         string       `json:"ordId"`
	OrderType       string       `json:"ordType"`
	Price           types.Number `json:"px"`
	Side            string       `json:"side"`
	State           string       `json:"state"`
	Size            types.Number `json:"sz"`
	Tag             string       `json:"tag"`
	TradeMode       string       `json:"tdMode"`
	UpdateTime      types.Time   `json:"uTime"`
}

// PositionInfo represents a positions detail
type PositionInfo struct {
	InstrumentType    string       `json:"instType"`
	InstrumentID      string       `json:"instId"`
	AlgoID            string       `json:"algoId"`
	Lever             types.Number `json:"lever"`
	MarginMode        string       `json:"mgnMode"`
	OpenAvgPrice      types.Number `json:"openAvgPx"`
	OpenOrderID       string       `json:"openOrdId"`
	OpenTime          types.Time   `json:"openTime"`
	PositionSide      string       `json:"posSide"`
	SlTriggerPrice    types.Number `json:"slTriggerPx"`
	SubPos            string       `json:"subPos"`
	SubPosID          string       `json:"subPosId"`
	TpTriggerPrice    types.Number `json:"tpTriggerPx"`
	CloseAveragePrice types.Number `json:"closeAvgPx"`
	CloseTime         types.Time   `json:"closeTime"`
}

// TPSLOrderParam holds Take profit and stop loss order parameters
type TPSLOrderParam struct {
	InstrumentType         string  `json:"instType"`
	SubPositionID          string  `json:"subPosId"`
	TakeProfitTriggerPrice float64 `json:"tpTriggerPx,omitempty,string"`
	StopLossTriggerPrice   float64 `json:"slTriggerPx,omitempty,string"`

	TakeProfitOrderPrice float64 `json:"tpOrdPx,omitempty,string"`
	StopLossOrderPrice   float64 `json:"slOrdPx,omitempty,string"`

	TakePofitTriggerPriceType string `json:"tpTriggerPriceType,omitempty,string"` // last: last price, 'index': index price 'mark': mark price Default is 'last'
	StopLossTriggerPriceType  string `jsonL:"slTriggerPxType,omitempty,string"`   // Stop-loss trigger price type 'last': last price 'index': index price 'mark': mark price Default is 'last'
	SubPositionType           string `json:"subPosType,omitempty,string"`         // 'lead': lead trading, the default value 'copy': copy trading
	Tag                       string `json:"tag,omitempty,string"`
}

// PositionIDInfo holds place positions information
type PositionIDInfo struct {
	SubPosID string `json:"subPosId"`
	Tag      string `json:"tag"`
}

// CloseLeadingPositionParam request parameter for closing leading position
type CloseLeadingPositionParam struct {
	InstrumentType string `json:"instType"`
	SubPositionID  string `json:"subPosId"`
	Tag            string `json:"tag"`
}

// LeadingInstrumentItem represents leading instrument info and it's status
type LeadingInstrumentItem struct {
	Enabled      bool   `json:"enabled"`
	InstrumentID string `json:"instId"`
}

// ProfitSharingItem holds profit sharing information
type ProfitSharingItem struct {
	Currency            string       `json:"ccy"`
	NickName            string       `json:"nickName"`
	ProfitSharingAmount types.Number `json:"profitSharingAmt"`
	ProfitSharingID     string       `json:"profitSharingId"`
	InstrumentType      string       `json:"instType"`
	Timestamp           types.Time   `json:"ts"`
}

// TotalProfitSharing holds information about total amount of profit shared since joining the platform
type TotalProfitSharing struct {
	Currency                 string       `json:"ccy"`
	InstrumentType           string       `json:"instType"`
	TotalProfitSharingAmount types.Number `json:"totalProfitSharingAmt"`
}

// Offer represents an investment offer information for different 'staking' and 'defi' protocols
type Offer struct {
	Currency        string            `json:"ccy"`
	ProductID       string            `json:"productId"`
	Protocol        string            `json:"protocol"`
	ProtocolType    string            `json:"protocolType"`
	EarningCurrency []string          `json:"earningCcy"`
	Term            string            `json:"term"`
	Apy             types.Number      `json:"apy"`
	EarlyRedeem     bool              `json:"earlyRedeem"`
	InvestData      []OfferInvestData `json:"investData"`
	EarningData     []struct {
		Currency    string `json:"ccy"`
		EarningType string `json:"earningType"`
	} `json:"earningData"`
	State                    string       `json:"state"`
	FastRedemptionDailyLimit types.Number `json:"fastRedemptionDailyLimit"`
}

// OfferInvestData represents currencies invest data information for an offer
type OfferInvestData struct {
	Currency      string       `json:"ccy"`
	Balance       types.Number `json:"bal"`
	MinimumAmount types.Number `json:"minAmt"`
	MaximumAmount types.Number `json:"maxAmt"`
}

// PurchaseRequestParam represents purchase request param specific product
type PurchaseRequestParam struct {
	ProductID  string                   `json:"productId"`
	Term       int64                    `json:"term,string,omitempty"`
	InvestData []PurchaseInvestDataItem `json:"investData"`
}

// PurchaseInvestDataItem represents purchase invest data information having the currency and amount information
type PurchaseInvestDataItem struct {
	Currency currency.Code `json:"ccy"`
	Amount   float64       `json:"amt,string"`
}

// OrderIDResponse represents purchase order ID
type OrderIDResponse struct {
	OrderID string `json:"orderId"`
	Tag     string `json:"tag"` // Optional to most ID responses
}

// CancelPurchaseOrRedemptionResponse represents a response for canceling a purchase or redemption
type CancelPurchaseOrRedemptionResponse struct {
	OrderIDResponse
	Tag string `json:"tag"`
}

// RedeemRequestParam represents redeem request input param
type RedeemRequestParam struct {
	OrderID          string `json:"ordId"`
	ProtocolType     string `json:"protocolType"`
	AllowEarlyRedeem bool   `json:"allowEarlyRedeem"`
}

// CancelFundingParam cancel purchase or redemption request
type CancelFundingParam struct {
	OrderID      string `json:"ordId"`
	ProtocolType string `json:"protocolType"`
}

// ProductInfo represents ETH staking information
type ProductInfo struct {
	FastRedemptionDailyLimit types.Number `json:"fastRedemptionDailyLimit"`
}

// ActiveFundingOrder represents active purchase orders
type ActiveFundingOrder struct {
	OrderID      string `json:"ordId"`
	State        string `json:"state"`
	Currency     string `json:"ccy"`
	Protocol     string `json:"protocol"`
	ProtocolType string `json:"protocolType"`
	Term         string `json:"term"`
	Apy          string `json:"apy"`
	InvestData   []struct {
		Currency string       `json:"ccy"`
		Amount   types.Number `json:"amt"`
	} `json:"investData"`
	EarningData []struct {
		Currency    string       `json:"ccy"`
		EarningType string       `json:"earningType"`
		Earnings    types.Number `json:"earnings"`
	} `json:"earningData"`
	PurchasedTime      types.Time `json:"purchasedTime"`
	FastRedemptionData []struct {
		Currency        string       `json:"ccy"`
		RedeemingAmount types.Number `json:"redeemingAmt"`
	} `json:"fastRedemptionData"`
	EstimatedRedemptionSettlementTime types.Time `json:"estSettlementTime"`
	CancelRedemptionDeadline          types.Time `json:"cancelRedemptionDeadline"`
	Tag                               string     `json:"tag"`
}

// BETHAssetsBalance balance is a snapshot summarized all BETH assets
type BETHAssetsBalance struct {
	Currency              string       `json:"ccy"`
	Amount                types.Number `json:"amt"`
	LatestInterestAccrual types.Number `json:"latestInterestAccrual"`
	TotalInterestAccrual  types.Number `json:"totalInterestAccrual"`
	Timestamp             types.Time   `json:"ts"`
}

// PurchaseRedeemHistory holds purchase and redeem history
type PurchaseRedeemHistory struct {
	Amt              types.Number `json:"amt"`
	CompletedTime    types.Time   `json:"completedTime"`
	EstCompletedTime types.Time   `json:"estCompletedTime"`
	RequestTime      types.Time   `json:"requestTime"`
	Status           string       `json:"status"`
	Type             string       `json:"type"`
}

// APYItem holds annual percentage yield record
type APYItem struct {
	Rate      types.Number `json:"rate"`
	Timestamp types.Time   `json:"ts"`
}

// WsOrderbook5 stores the orderbook data for orderbook 5 websocket
type WsOrderbook5 struct {
	Argument struct {
		Channel      string        `json:"channel"`
		InstrumentID currency.Pair `json:"instId"`
	} `json:"arg"`
	Data []Book5Data `json:"data"`
}

// Book5Data stores the orderbook data for orderbook 5 websocket
type Book5Data struct {
	Asks         [][4]types.Number `json:"asks"`
	Bids         [][4]types.Number `json:"bids"`
	InstrumentID string            `json:"instId"`
	Timestamp    types.Time        `json:"ts"`
	SequenceID   int64             `json:"seqId"`
}

// WsSpreadOrder represents spread order detail
type WsSpreadOrder struct {
	SpreadID          string       `json:"sprdId"`
	OrderID           string       `json:"ordId"`
	ClientOrderID     string       `json:"clOrdId"`
	Tag               string       `json:"tag"`
	Price             types.Number `json:"px"`
	Size              types.Number `json:"sz"`
	OrderType         string       `json:"ordType"`
	Side              string       `json:"side"`
	FillSize          types.Number `json:"fillSz"`
	FillPrice         types.Number `json:"fillPx"`
	TradeID           string       `json:"tradeId"`
	AccFillSize       types.Number `json:"accFillSz"`
	PendingFillSize   types.Number `json:"pendingFillSz"`
	PendingSettleSize types.Number `json:"pendingSettleSz"`
	CanceledSize      types.Number `json:"canceledSz"`
	State             string       `json:"state"`
	AveragePrice      types.Number `json:"avgPx"`
	CancelSource      string       `json:"cancelSource"`
	UpdateTime        types.Time   `json:"uTime"`
	CreationTime      types.Time   `json:"cTime"`
	Code              string       `json:"code"`
	Msg               string       `json:"msg"`
}

// WsSpreadOrderTrade trade of an order
type WsSpreadOrderTrade struct {
	Argument struct {
		Channel  string `json:"channel"`
		SpreadID string `json:"sprdId"`
		UID      string `json:"uid"`
	} `json:"arg"`
	Data []struct {
		SpreadID      string       `json:"sprdId"`
		TradeID       string       `json:"tradeId"`
		OrderID       string       `json:"ordId"`
		ClientOrderID string       `json:"clOrdId"`
		Tag           string       `json:"tag"`
		FillPrice     types.Number `json:"fillPx"`
		FillSize      types.Number `json:"fillSz"`
		State         string       `json:"state"`
		Side          string       `json:"side"`
		ExecType      string       `json:"execType"`
		Timestamp     types.Time   `json:"ts"`
		Legs          []struct {
			InstrumentID string       `json:"instId"`
			Price        types.Number `json:"px"`
			Size         types.Number `json:"sz"`
			Side         string       `json:"side"`
			Fee          types.Number `json:"fee"`
			FeeCurrency  string       `json:"feeCcy"`
			TradeID      string       `json:"tradeId"`
		} `json:"legs"`
		Code string `json:"code"`
		Msg  string `json:"msg"`
	} `json:"data"`
}

// WsSpreadOrderbook holds spread orderbook data
type WsSpreadOrderbook struct {
	Arg struct {
		Channel  string `json:"channel"`
		SpreadID string `json:"sprdId"`
	} `json:"arg"`
	Data []struct {
		Asks      [][3]types.Number `json:"asks"`
		Bids      [][3]types.Number `json:"bids"`
		Timestamp types.Time        `json:"ts"`
	} `json:"data"`
}

// WsSpreadPushData holds push data
type WsSpreadPushData struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     any              `json:"data"`
}

// WsSpreadPublicTicker holds spread public ticker data
type WsSpreadPublicTicker struct {
	SpreadID  string       `json:"sprdId"`
	Last      types.Number `json:"last"`
	LastSize  types.Number `json:"lastSz"`
	AskPrice  types.Number `json:"askPx"`
	AskSize   types.Number `json:"askSz"`
	BidPrice  types.Number `json:"bidPx"`
	BidSize   types.Number `json:"bidSz"`
	Timestamp types.Time   `json:"ts"`
}

// WsSpreadPublicTrade holds trades data from sprd-public-trades
type WsSpreadPublicTrade struct {
	SpreadID  string       `json:"sprdId"`
	Side      string       `json:"side"`
	Size      types.Number `json:"sz"`
	Price     types.Number `json:"px"`
	TradeID   string       `json:"tradeId"`
	Timestamp types.Time   `json:"ts"`
}

// ExtractSpreadOrder extracts WsSpreadOrderbookData from a WsSpreadOrderbook instance
func (a *WsSpreadOrderbook) ExtractSpreadOrder() (*WsSpreadOrderbookData, error) {
	resp := &WsSpreadOrderbookData{
		Argument: SubscriptionInfo{
			SpreadID: a.Arg.SpreadID,
			Channel:  a.Arg.Channel,
		},
		Data: make([]WsSpreadOrderbookItem, len(a.Data)),
	}
	for x := range a.Data {
		resp.Data[x].Timestamp = a.Data[x].Timestamp.Time()
		resp.Data[x].Asks = make([]orderbook.Level, len(a.Data[x].Asks))
		resp.Data[x].Bids = make([]orderbook.Level, len(a.Data[x].Bids))

		for as := range a.Data[x].Asks {
			resp.Data[x].Asks[as].Price = a.Data[x].Asks[as][0].Float64()
			resp.Data[x].Asks[as].Amount = a.Data[x].Asks[as][1].Float64()
			resp.Data[x].Asks[as].OrderCount = a.Data[x].Asks[as][2].Int64()
		}
		for as := range a.Data[x].Bids {
			resp.Data[x].Bids[as].Price = a.Data[x].Bids[as][0].Float64()
			resp.Data[x].Bids[as].Amount = a.Data[x].Bids[as][1].Float64()
			resp.Data[x].Bids[as].OrderCount = a.Data[x].Bids[as][2].Int64()
		}
	}
	return resp, nil
}

// WsSpreadOrderbookItem represents an orderbook asks and bids details
type WsSpreadOrderbookItem struct {
	Asks      []orderbook.Level
	Bids      []orderbook.Level
	Timestamp time.Time
}

// WsSpreadOrderbookData represents orderbook response for spread instruments
type WsSpreadOrderbookData struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsSpreadOrderbookItem
}

// AffilateInviteesDetail represents affiliate invitee's detail
type AffilateInviteesDetail struct {
	InviteeLevel             types.Number `json:"inviteeLv"`
	JoinTime                 types.Time   `json:"joinTime"`
	InviteeRebateRate        types.Number `json:"inviteeRebateRate"`
	TotalCommission          types.Number `json:"totalCommission"`
	FirstTradeTime           types.Time   `json:"firstTradeTime"`
	Level                    string       `json:"level"`
	DepositAmount            types.Number `json:"depAmt"`
	AccumulatedTradingVolume types.Number `json:"volMonth"`
	AccumulatedFee           types.Number `json:"accFee"`
	KYCTime                  types.Time   `json:"kycTime"`
	Region                   string       `json:"region"`
	AffiliateCode            string       `json:"affiliateCode"`
}

// AffilateRebateInfo represents rebate information
type AffilateRebateInfo struct {
	Result bool   `json:"result"`
	Type   string `json:"type"`
}

// WsDepositInfo represents a deposit information
type WsDepositInfo struct {
	ActualDepBulkConfirm string       `json:"actualDepBlkConfirm"`
	Amount               types.Number `json:"amt"`
	AreaCodeFrom         string       `json:"areaCodeFrom"`
	Currency             string       `json:"ccy"`
	Chain                string       `json:"chain"`
	DepositID            string       `json:"depId"`
	From                 string       `json:"from"`
	FromWdID             string       `json:"fromWdId"` // Internal transfer initiator's withdrawal ID
	PushTime             types.Time   `json:"pTime"`
	State                string       `json:"state"`
	SubAccount           string       `json:"subAcct"`
	To                   string       `json:"to"`
	Timestamp            types.Time   `json:"ts"`
	TransactionID        string       `json:"txId"`
	UID                  string       `json:"uid"`
}

// WsWithdrawlInfo represents push notification is triggered when a withdrawal is initiated or the withdrawal status changes
type WsWithdrawlInfo struct {
	AddrEx           any          `json:"addrEx"`
	Amount           types.Number `json:"amt"`
	AreaCodeFrom     string       `json:"areaCodeFrom"`
	AreaCodeTo       string       `json:"areaCodeTo"`
	Currency         string       `json:"ccy"`
	Chain            string       `json:"chain"`
	ClientID         string       `json:"clientId"`
	Fee              types.Number `json:"fee"`
	FeeCurrency      string       `json:"feeCcy"`
	From             string       `json:"from"`
	Memo             string       `json:"memo"`
	NonTradableAsset bool         `json:"nonTradableAsset"`
	PushTime         types.Time   `json:"pTime"`
	PmtID            string       `json:"pmtId"`
	State            string       `json:"state"`
	SubAcct          string       `json:"subAcct"`
	Tag              string       `json:"tag"`
	To               string       `json:"to"`
	Timestamp        types.Time   `json:"ts"`
	TransactionID    string       `json:"txId"`
	UID              string       `json:"uid"`
	WithdrawalID     string       `json:"wdId"`
}

// RecurringBuyOrder represents a recurring buy order instance
type RecurringBuyOrder struct {
	AlgoClOrdID        string       `json:"algoClOrdId"`
	AlgoID             string       `json:"algoId"`
	AlgoOrderType      string       `json:"algoOrdType"`
	Amount             types.Number `json:"amt"`
	CreationTime       types.Time   `json:"cTime"`
	Cycles             string       `json:"cycles"`
	InstrumentType     string       `json:"instType"`
	InvestmentAmount   types.Number `json:"investmentAmt"`
	InvestmentCurrency string       `json:"investmentCcy"`
	MarketCap          string       `json:"mktCap"`
	NextInvestTime     types.Time   `json:"nextInvestTime"`
	PushTime           types.Time   `json:"pTime"`
	Period             string       `json:"period"`
	ProfitAndLossRatio types.Number `json:"pnlRatio"`
	RecurringDay       string       `json:"recurringDay"`
	RecurringHour      string       `json:"recurringHour"`
	RecurringList      []struct {
		AveragePrice types.Number `json:"avgPx"`
		Currency     string       `json:"ccy"`
		Profit       string       `json:"profit"`
		Price        types.Number `json:"px"`
		Ratio        types.Number `json:"ratio"`
		TotalAmount  types.Number `json:"totalAmt"`
	} `json:"recurringList"`
	RecurringTime string     `json:"recurringTime"`
	State         string     `json:"state"`
	StrategyName  string     `json:"stgyName"`
	Tag           string     `json:"tag"`
	TimeZone      string     `json:"timeZone"`
	TotalAnnRate  string     `json:"totalAnnRate"`
	TotalPnl      string     `json:"totalPnl"`
	UpdateTime    types.Time `json:"uTime"`
}

// ADLWarning represents auto-deleveraging warning
type ADLWarning struct {
	Arg  SubscriptionInfo `json:"arg"`
	Data []struct {
		DecRate          string       `json:"decRate"`
		MaxBal           string       `json:"maxBal"`
		AdlRecRate       types.Number `json:"adlRecRate"`
		AdlRecBal        types.Number `json:"adlRecBal"`
		Balance          types.Number `json:"bal"`
		InstrumentType   string       `json:"instType"`
		AdlRate          types.Number `json:"adlRate"`
		InstrumentFamily string       `json:"instFamily"`
		MaxBalTimestamp  types.Time   `json:"maxBalTs"`
		AdlType          string       `json:"adlType"`
		State            string       `json:"state"`
		AdlBalance       types.Number `json:"adlBal"`
		Timestamp        types.Time   `json:"ts"`
	} `json:"data"`
}

// EconomicCalendar represents macro-economic calendar data
type EconomicCalendar struct {
	Actual      string     `json:"actual"`
	CalendarID  string     `json:"calendarId"`
	Date        types.Time `json:"date"`
	Region      string     `json:"region"`
	Category    string     `json:"category"`
	Event       string     `json:"event"`
	RefDate     types.Time `json:"refDate"`
	Previous    string     `json:"previous"`
	Forecast    string     `json:"forecast"`
	Importance  string     `json:"importance"`
	PrevInitial string     `json:"prevInitial"`
	Currency    string     `json:"ccy"`
	Unit        string     `json:"unit"`
	Timestamp   types.Time `json:"ts"`
	DateSpan    string     `json:"dateSpan"`
	UpdateTime  types.Time `json:"uTime"`
}

// EconomicCalendarResponse represents response for economic calendar
type EconomicCalendarResponse struct {
	Arg  SubscriptionInfo   `json:"arg"`
	Data []EconomicCalendar `json:"data"`
}

// CopyTradingNotification holds copy-trading notifications
type CopyTradingNotification struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []struct {
		AveragePrice        types.Number `json:"avgPx"`
		Currency            string       `json:"ccy"`
		CopyTotalAmount     types.Number `json:"copyTotalAmt"`
		InfoType            string       `json:"infoType"`
		InstrumentID        string       `json:"instId"`
		InstrumentType      string       `json:"instType"`
		Leverage            types.Number `json:"lever"`
		MaxLeadTraderNumber string       `json:"maxLeadTraderNum"`
		MinNotional         types.Number `json:"minNotional"`
		PositionSide        string       `json:"posSide"`
		RmThreshold         string       `json:"rmThold"` // Lead trader can remove copy trader if balance of copy trader less than this value.
		Side                string       `json:"side"`
		StopLossTotalAmount types.Number `json:"slTotalAmt"`
		SlippageRatio       types.Number `json:"slippageRatio"`
		SubPosID            string       `json:"subPosId"`
		UniqueCode          string       `json:"uniqueCode"`
	} `json:"data"`
}

// FirstCopySettings holds parameters first copy settings for the certain lead trader
type FirstCopySettings struct {
	InstrumentType       string  `json:"instType,omitempty"`
	InstrumentID         string  `json:"instId"` // Instrument ID. If there are multiple instruments, separate them with commas. Maximum of 200 instruments can be selected
	UniqueCode           string  `json:"uniqueCode"`
	CopyMarginMode       string  `json:"copyMgnMode,omitempty"` // Copy margin mode 'cross': cross 'isolated': isolated 'copy'
	CopyInstrumentIDType string  `json:"copyInstIdType"`        // Copy contract type set 'custom': custom by instId which is required；'copy': Keep your contracts consistent with this trader
	CopyMode             string  `json:"copyMode,omitempty"`    // Possible values: 'fixed_amount', 'ratio_copy', 'copyRatio', and 'fixed_amount'
	CopyRatio            float64 `json:"copyRatio,string"`
	CopyAmount           float64 `json:"copyAmt,string,omitempty"`
	CopyTotalAmount      float64 `json:"copyTotalAmt,string"`
	SubPosCloseType      string  `json:"subPosCloseType"`
	TakeProfitRatio      float64 `json:"tpRatio,string,omitempty"`
	StopLossRatio        float64 `json:"slRatio,string,omitempty"`
	StopLossTotalAmount  float64 `json:"slTotalAmt,string,omitempty"`
}

// StopCopyingParameter holds stop copying request parameter
type StopCopyingParameter struct {
	InstrumentType       string `json:"instType,omitempty"`
	UniqueCode           string `json:"uniqueCode"`
	SubPositionCloseType string `json:"subPosCloseType"`
}

// CopySetting represents a copy setting response
type CopySetting struct {
	Currency             string       `json:"ccy"`
	CopyState            string       `json:"copyState"`
	CopyMarginMode       string       `json:"copyMgnMode"`
	SubPositionCloseType string       `json:"subPosCloseType"`
	StopLossTotalAmount  types.Number `json:"slTotalAmt"`
	CopyAmount           types.Number `json:"copyAmt"`
	CopyInstrumentIDType string       `json:"copyInstIdType"`
	CopyMode             string       `json:"copyMode"`
	CopyRatio            types.Number `json:"copyRatio"`
	CopyTotalAmount      types.Number `json:"copyTotalAmt"`
	InstrumentIDs        []struct {
		Enabled      string `json:"enabled"`
		InstrumentID string `json:"instId"`
	} `json:"instIds"`
	StopLossRatio   types.Number `json:"slRatio"`
	TakeProfitRatio types.Number `json:"tpRatio"`
}

// Leverages holds batch leverage info
type Leverages struct {
	LeadTraderLevers []LeverageInfo `json:"leadTraderLevers"`
	MyLevers         []LeverageInfo `json:"myLevers"`
	InstrumentID     string         `json:"instId"`
	MarginMode       string         `json:"mgnMode"`
}

// LeverageInfo holds leverage information
type LeverageInfo struct {
	Leverage     types.Number `json:"lever"`
	PositionSide string       `json:"posSide"`
}

// SetMultipleLeverageResponse represents multiple leverage response
type SetMultipleLeverageResponse struct {
	FailInstrumentID string `json:"failInstId"`
	Result           string `json:"result"`
	SuccInstrumentID string `json:"succInstId"`
}

// SetLeveragesParam sets leverage parameter
type SetLeveragesParam struct {
	MarginMode   string `json:"mgnMode"`
	Leverage     int64  `json:"lever,string"`
	InstrumentID string `json:"instId,omitempty"` // Instrument ID. If there are multiple instruments, separate them with commas. Maximum of 200 instruments can be selected
}

// CopyTradingLeadTrader represents a lead trader information
type CopyTradingLeadTrader struct {
	BeginCopyTime           types.Time   `json:"beginCopyTime"`
	Currency                string       `json:"ccy"`
	CopyTotalAmount         types.Number `json:"copyTotalAmt"`
	CopyTotalProfitAndLoss  types.Number `json:"copyTotalPnl"`
	LeadMode                string       `json:"leadMode"`
	Margin                  types.Number `json:"margin"`
	NickName                string       `json:"nickName"`
	PortLink                string       `json:"portLink"`
	ProfitSharingRatio      types.Number `json:"profitSharingRatio"`
	TodayProfitAndLoss      types.Number `json:"todayPnl"`
	UniqueCode              string       `json:"uniqueCode"`
	UnrealizedProfitAndLoss types.Number `json:"upl"`
	CopyMode                string       `json:"copyMode"`
	CopyNum                 string       `json:"copyNum"`
	CopyRatio               types.Number `json:"copyRatio"`
	CopyRelID               string       `json:"copyRelId"`
	CopyState               string       `json:"copyState"`
}

// LeadTraderRanksRequest represents lead trader ranks request parameters
type LeadTraderRanksRequest struct {
	InstrumentType           string  // Instrument type e.g 'SWAP'. The default value is 'SWAP'
	SortType                 string  // Overview, the default value. pnl: profit and loss, aum: assets under management, win_ratio: win ratio,pnl_ratio: pnl ratio, current_copy_trader_pnl: current copy trader pnl
	HasVacancy               bool    // false: include all lead traders (default), with or without vacancies; true: include only those with vacancies
	MinLeadDays              uint64  // 1: 7 days. 2: 30 days. 3: 90 days. 4: 180 days
	MinAssets                float64 // Minimum assets in USDT
	MaxAssets                float64 // Maximum assets in USDT
	MinAssetsUnderManagement float64 // Minimum assets under management in USDT
	MaxAssetsUnderManagement float64 // Maximum assets under management in USDT
	DataVersion              uint64  // It is 14 numbers. e.g. 20231010182400 used for pagination. A new version will be generated every 10 minutes. Only last 5 versions are stored. The default is latest version
	Page                     uint64  // Page number for pagination
	Limit                    uint64  // Number of results per request. The maximum is 20; the default is 10
}

// LeadTradersRank represents lead traders rank info
type LeadTradersRank struct {
	DataVer string `json:"dataVer"`
	Ranks   []struct {
		AccCopyTraderNum string       `json:"accCopyTraderNum"`
		Aum              string       `json:"aum"`
		Currency         string       `json:"ccy"`
		CopyState        string       `json:"copyState"`
		CopyTraderNum    string       `json:"copyTraderNum"`
		LeadDays         string       `json:"leadDays"`
		MaxCopyTraderNum string       `json:"maxCopyTraderNum"`
		NickName         string       `json:"nickName"`
		Pnl              types.Number `json:"pnl"`
		PnlRatio         types.Number `json:"pnlRatio"`
		PnlRatios        []struct {
			BeginTimestamp types.Time   `json:"beginTs"`
			PnlRatio       types.Number `json:"pnlRatio"`
		} `json:"pnlRatios"`
		PortLink    string       `json:"portLink"`
		TraderInsts []string     `json:"traderInsts"`
		UniqueCode  string       `json:"uniqueCode"`
		WinRatio    types.Number `json:"winRatio"`
	} `json:"ranks"`
	TotalPage string `json:"totalPage"`
}

// TraderWeeklyProfitAndLoss represents lead trader weekly pnl
type TraderWeeklyProfitAndLoss struct {
	BeginTimestamp     types.Time   `json:"beginTs"`
	ProfitAndLoss      types.Number `json:"pnl"`
	ProfitAndLossRatio types.Number `json:"pnlRatio"`
}

// LeadTraderStat represents lead trader performance info
type LeadTraderStat struct {
	AvgSubPosNotional types.Number `json:"avgSubPosNotional"`
	Currency          string       `json:"ccy"`
	CurCopyTraderPnl  types.Number `json:"curCopyTraderPnl"`
	InvestAmount      types.Number `json:"investAmt"`
	LossDays          string       `json:"lossDays"`
	ProfitDays        string       `json:"profitDays"`
	WinRatio          types.Number `json:"winRatio"`
}

// LeadTraderCurrencyPreference holds public preference currency
type LeadTraderCurrencyPreference struct {
	Currency string       `json:"ccy"`
	Ratio    types.Number `json:"ratio"`
}

// LeadTraderCurrentLeadPosition holds leading positions of lead trader
type LeadTraderCurrentLeadPosition struct {
	Currency       string       `json:"ccy"`
	InstrumentID   string       `json:"instId"`
	InstrumentType string       `json:"instType"`
	Lever          types.Number `json:"lever"`
	Margin         types.Number `json:"margin"`
	MarkPrice      types.Number `json:"markPx"`
	MarginMode     string       `json:"mgnMode"`
	OpenAvgPrice   types.Number `json:"openAvgPx"`
	OpenTime       types.Time   `json:"openTime"`
	PositionSide   string       `json:"posSide"`
	SubPos         string       `json:"subPos"`
	SubPosID       string       `json:"subPosId"`
	UniqueCode     string       `json:"uniqueCode"`
	UPL            types.Number `json:"upl"`
	UPLRatio       types.Number `json:"uplRatio"`
}

// LeadPosition holds lead trader completed leading position
type LeadPosition struct {
	Currency           string       `json:"ccy"`
	CloseAveragePrice  types.Number `json:"closeAvgPx"`
	CloseTime          types.Time   `json:"closeTime"`
	InstrumentID       string       `json:"instId"`
	InstrumentType     string       `json:"instType"`
	Leverage           types.Number `json:"lever"`
	Margin             types.Number `json:"margin"`
	MarginMode         string       `json:"mgnMode"`
	OpenAveragePrice   types.Number `json:"openAvgPx"`
	OpenTime           types.Time   `json:"openTime"`
	ProfitAndLoss      string       `json:"pnl"`
	ProfitAndLossRatio types.Number `json:"pnlRatio"`
	PositionSide       string       `json:"posSide"`
	SubPosition        string       `json:"subPos"`
	SubPositionID      string       `json:"subPosId"`
	UniqueCode         string       `json:"uniqueCode"`
}

// LendingOrderParam represents a lending order request parameters
type LendingOrderParam struct {
	Currency    currency.Code `json:"ccy"`
	Amount      float64       `json:"amt,omitempty,string"`
	Rate        float64       `json:"rate,omitempty,string"`
	Term        string        `json:"term"`
	AutoRenewal bool          `json:"autoRenewal,omitempty"`
}

// LendingOrderResponse represents an order ID response after placing a lending order
type LendingOrderResponse []struct {
	OrderID string `json:"ordId"`
}

// LendingOrderDetail represents a lending order detail
type LendingOrderDetail struct {
	OrderID       string       `json:"ordId"`
	Amount        types.Number `json:"amt"`
	AutoRenewal   bool         `json:"autoRenewal"`
	Currency      string       `json:"ccy"`
	EarningAmount types.Number `json:"earningAmt"`
	PendingAmount types.Number `json:"pendingAmt"`
	Rate          types.Number `json:"rate"`
	State         string       `json:"state"`
	Term          string       `json:"term"`
	TotalInterest string       `json:"totalInterest"`
	CreationTime  types.Time   `json:"cTime"`
	UpdateTime    types.Time   `json:"uTime"`
	SettledTime   types.Time   `json:"settledTime"`
	StartTime     types.Time   `json:"startTime"`
}

// LendingSubOrder represents a lending sub-order detail
type LendingSubOrder struct {
	AccruedInterest        string       `json:"accruedInterest"`
	Amount                 types.Number `json:"amt"`
	Currency               string       `json:"ccy"`
	EarlyTerminatedPenalty string       `json:"earlyTerminatedPenalty"`
	ExpiryTime             types.Time   `json:"expiryTime"`
	FinalSettlementTime    types.Time   `json:"finalSettlementTime"`
	OrderID                string       `json:"ordId"`
	OverdueInterest        string       `json:"overdueInterest"`
	Rate                   string       `json:"rate"`
	SettledTime            types.Time   `json:"settledTime"`
	State                  string       `json:"state"`
	SubOrdID               string       `json:"subOrdId"`
	Term                   string       `json:"term"`
	TotalInterest          string       `json:"totalInterest"`
	CreationTime           types.Time   `json:"cTime"`
	UpdateTime             types.Time   `json:"uTime"`
}

// PublicLendingOffer represents a lending offer detail
type PublicLendingOffer struct {
	Currency         string       `json:"ccy"`
	LendQuota        string       `json:"lendQuota"`
	MinLendingAmount types.Number `json:"minLend"`
	Rate             types.Number `json:"rate"`
	Term             string       `json:"term"`
}

// LendingAPIHistoryItem represents a lending API history item
type LendingAPIHistoryItem struct {
	Currency  string       `json:"ccy"`
	Rate      types.Number `json:"rate"`
	Timestamp types.Time   `json:"ts"`
}

// LendingVolume represents a lending volume detail for a specific currency
type LendingVolume struct {
	Currency      string       `json:"ccy"`
	PendingVol    types.Number `json:"pendingVol"`
	RateRangeFrom string       `json:"rateRangeFrom"`
	RateRangeTo   string       `json:"rateRangeTo"`
	Term          string       `json:"term"`
}

// SpreadOrderCancellationResponse represents a spread order cancellation response
type SpreadOrderCancellationResponse struct {
	TriggerTime types.Time `json:"triggerTime"`
	Timestamp   types.Time `json:"ts"`
}

// ContractTakerVolume represents a contract taker sell and buy volume
type ContractTakerVolume struct {
	Timestamp       types.Time
	TakerSellVolume types.Number
	TakerBuyVolume  types.Number
}

// UnmarshalJSON deserializes a slice data into ContractTakerVolume
func (c *ContractTakerVolume) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[3]any{&c.Timestamp, &c.TakerSellVolume, &c.TakerBuyVolume})
}

// ContractOpenInterestHistoryItem represents an open interest information for contract
type ContractOpenInterestHistoryItem struct {
	Timestamp              types.Time
	OpenInterestInContract types.Number
	OpenInterestInCurrency types.Number
	OpenInterestInUSD      types.Number
}

// UnmarshalJSON deserializes slice data into ContractOpenInterestHistoryItem instance
func (c *ContractOpenInterestHistoryItem) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[4]any{&c.Timestamp, &c.OpenInterestInContract, &c.OpenInterestInCurrency, &c.OpenInterestInUSD})
}

// TopTraderContractsLongShortRatio represents the timestamp and ratio information of top traders long and short accounts/positions
type TopTraderContractsLongShortRatio struct {
	Timestamp types.Time
	Ratio     types.Number
}

// UnmarshalJSON deserializes slice data into TopTraderContractsLongShortRatio instance
func (t *TopTraderContractsLongShortRatio) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[2]any{&t.Timestamp, &t.Ratio})
}

// AccountInstrument represents an account instrument
type AccountInstrument struct {
	BaseCurrency        string       `json:"baseCcy"`
	ContractMultiplier  string       `json:"ctMult"`
	ContractType        string       `json:"ctType"`
	ContractValue       string       `json:"ctVal"`
	ContractValCurrency string       `json:"ctValCcy"`
	ExpiryTime          types.Time   `json:"expTime"`
	InstrumentType      string       `json:"instType"`
	InstrumentID        string       `json:"instId"`
	InstFamily          string       `json:"instFamily"`
	MaxLeverage         string       `json:"lever"`
	ListTime            types.Time   `json:"listTime"`
	LotSz               types.Number `json:"lotSz"` // If it is a derivatives contract, the value is the number of contracts. If it is SPOT/MARGIN, the value is the quantity in base currency.
	MaxIcebergSz        types.Number `json:"maxIcebergSz"`
	MaxLimitAmount      types.Number `json:"maxLmtAmt"`
	MaxLimitSize        types.Number `json:"maxLmtSz"`
	MaxMktAmount        types.Number `json:"maxMktAmt"`
	MaxMktSize          types.Number `json:"maxMktSz"`
	MaxStopSize         types.Number `json:"maxStopSz"`
	MaxTriggerSz        types.Number `json:"maxTriggerSz"`
	MaxTwapSize         types.Number `json:"maxTwapSz"`
	MinSize             types.Number `json:"minSz"`
	OptionType          string       `json:"optType"`
	QuoteCurrency       string       `json:"quoteCcy"`
	SettleCurrency      string       `json:"settleCcy"`
	State               string       `json:"state"`
	StrikePrice         string       `json:"stk"`
	TickSize            types.Number `json:"tickSz"`
	Underlying          string       `json:"uly"`
	RuleType            string       `json:"ruleType"`
}

// ReduceLiabilities represents a response after reducing liabilities
type ReduceLiabilities struct {
	OrderID      string `json:"ordId"`
	PendingRepay bool   `json:"pendingRepay"`
}

// FixedLoanBorrowOrderDetail represents a borrow order detail
type FixedLoanBorrowOrderDetail struct {
	OrderID                   string       `json:"ordId"`
	AccruedInterest           string       `json:"accruedInterest"`
	ActualBorrowAmount        types.Number `json:"actualBorrowAmt"`
	CreateTime                types.Time   `json:"cTime"`
	Currency                  string       `json:"ccy"`
	CurRate                   types.Number `json:"curRate"`
	DeadlinePenaltyInterest   types.Number `json:"deadlinePenaltyInterest"`
	EarlyRepayPenaltyInterest types.Number `json:"earlyRepayPenaltyInterest"`
	ExpiryTime                types.Time   `json:"expiryTime"`
	FailedReason              string       `json:"failedReason"`
	ForceRepayTime            types.Time   `json:"forceRepayTime"`
	OverduePenaltyInterest    types.Number `json:"overduePenaltyInterest"`
	PotentialPenaltyInterest  types.Number `json:"potentialPenaltyInterest"`
	Reborrow                  bool         `json:"reborrow"`
	ReborrowRate              types.Number `json:"reborrowRate"`
	ReqBorrowAmount           types.Number `json:"reqBorrowAmt"`
	SettleReason              string       `json:"settleReason"`
	State                     string       `json:"state"`
	Term                      string       `json:"term"`
	UpdateTime                types.Time   `json:"uTime"`
}

// BorrowOrRepay represents a borrow and repay operation response
type BorrowOrRepay struct {
	Currency string       `json:"ccy"`
	Side     string       `json:"side"`
	Amount   types.Number `json:"amt"`
}

// AutoRepay represents an auto-repay request and response
type AutoRepay struct {
	AutoRepay bool `json:"autoRepay"`
}

// BorrowRepayItem represents a borrow/repay history
type BorrowRepayItem struct {
	AccBorrowed string       `json:"accBorrowed"`
	Amount      types.Number `json:"amt"`
	Currency    string       `json:"ccy"`
	Timestamp   types.Time   `json:"ts"`
	EventType   string       `json:"type"`
}

// PositionBuilderParam represents a position builder parameters
type PositionBuilderParam struct {
	InclRealPosAndEq bool                `json:"inclRealPosAndEq"`
	SimPos           []SimulatedPosition `json:"simPos"`
	SimAsset         []SimulatedAsset    `json:"simAsset"`
	SpotOffsetType   string              `json:"spotOffsetType"`
	GreeksType       string              `json:"greeksType"`
}

// SimulatedPosition represents a simulated position detail of a new position builder
type SimulatedPosition struct {
	Position     string `json:"pos"`
	InstrumentID string `json:"instId"`
}

// SimulatedAsset represents a simulated asset detail
type SimulatedAsset struct {
	Currency string       `json:"ccy"`
	Amount   types.Number `json:"amt"`
}

// PositionBuilderDetail represents details of portfolio margin information for virtual position/assets or current position of the user
type PositionBuilderDetail struct {
	Assets []struct {
		AvailEq   types.Number `json:"availEq"`
		BorrowIMR types.Number `json:"borrowImr"`
		BorrowMMR types.Number `json:"borrowMmr"`
		Currency  string       `json:"ccy"`
		SpotInUse string       `json:"spotInUse"`
	} `json:"assets"`
	BorrowMMR    string       `json:"borrowMmr"`
	DerivMMR     string       `json:"derivMmr"`
	Equity       string       `json:"eq"`
	MarginRatio  types.Number `json:"marginRatio"`
	RiskUnitData []struct {
		Delta          string `json:"delta"`
		Gamma          string `json:"gamma"`
		IMR            string `json:"imr"`
		IndexUsd       string `json:"indexUsd"`
		Mmr            string `json:"mmr"`
		Mr1            string `json:"mr1"`
		Mr1FinalResult struct {
			PNL       types.Number `json:"pnl"`
			SpotShock string       `json:"spotShock"`
			VolShock  string       `json:"volShock"`
		} `json:"mr1FinalResult"`
		Mr1Scenarios struct {
			VolSame      map[string]string `json:"volSame"`
			VolShockDown map[string]string `json:"volShockDown"`
			VolShockUp   map[string]string `json:"volShockUp"`
		} `json:"mr1Scenarios"`
		Mr2            string `json:"mr2"`
		Mr3            string `json:"mr3"`
		Mr4            string `json:"mr4"`
		Mr5            string `json:"mr5"`
		Mr6            string `json:"mr6"`
		Mr6FinalResult struct {
			PNL       types.Number `json:"pnl"`
			SpotShock string       `json:"spotShock"`
		} `json:"mr6FinalResult"`
		Mr7        string `json:"mr7"`
		Portfolios []struct {
			Amount         types.Number `json:"amt"`
			Delta          types.Number `json:"delta"`
			Gamma          types.Number `json:"gamma"`
			InstrumentID   string       `json:"instId"`
			InstrumentType string       `json:"instType"`
			IsRealPos      bool         `json:"isRealPos"`
			NotionalUsd    string       `json:"notionalUsd"`
			Theta          string       `json:"theta"`
			Vega           string       `json:"vega"`
		} `json:"portfolios"`
		RiskUnit string `json:"riskUnit"`
		Theta    string `json:"theta"`
		Vega     string `json:"vega"`
	} `json:"riskUnitData"`
	TotalImr  types.Number `json:"totalImr"`
	TotalMmr  types.Number `json:"totalMmr"`
	Timestamp types.Time   `json:"ts"`
}

// RiskOffsetAmount represents risk offset amount
type RiskOffsetAmount struct {
	Currency              string       `json:"ccy"`
	ClientSpotInUseAmount types.Number `json:"clSpotInUseAmt"`
}

// AccountRateLimit represents an account rate limit details
type AccountRateLimit struct {
	AccRateLimit     types.Number `json:"accRateLimit"`
	FillRatio        types.Number `json:"fillRatio"`
	MainFillRatio    types.Number `json:"mainFillRatio"`
	NextAccRateLimit types.Number `json:"nextAccRateLimit"`
	Timestamp        types.Time   `json:"ts"`
}

// OrderPreCheckParams represents an order pre-check parameters
type OrderPreCheckParams struct {
	InstrumentID     string          `json:"instId"`
	TradeMode        string          `json:"tdMode"`
	ClientOrderID    string          `json:"clOrdId"`
	Side             string          `json:"side"`
	PositionSide     string          `json:"posSide"`
	OrderType        string          `json:"ordType"`
	Size             float64         `json:"sz,omitempty"`
	Price            float64         `json:"px,omitempty"`
	ReduceOnly       bool            `json:"reduceOnly,omitempty"`
	TargetCurrency   string          `json:"tgtCcy,omitempty"`
	AttachAlgoOrders []AlgoOrderInfo `json:"attachAlgoOrds,omitempty"`
}

// AlgoOrderInfo represents an algo order info
type AlgoOrderInfo struct {
	AttachAlgoClientOrderID  string       `json:"attachAlgoClOrdId,omitempty"`
	TPTriggerPrice           types.Number `json:"tpTriggerPx,omitempty"`
	TPOrderPrice             types.Number `json:"tpOrdPx,omitempty"`
	TPOrderKind              string       `json:"tpOrdKind,omitempty"`
	StopLossTriggerPrice     types.Number `json:"slTriggerPx,omitempty"`
	StopLossOrderPrice       types.Number `json:"slOrdPx,omitempty"`
	TPTriggerPriceType       string       `json:"tpTriggerPxType,omitempty"`
	StopLossTriggerPriceType string       `json:"slTriggerPxType,omitempty"`
	Size                     types.Number `json:"sz,omitempty"`
}

// OrderPreCheckResponse represents an order pre-checks response of account information for placing orders
type OrderPreCheckResponse struct {
	AdjEq                      types.Number `json:"adjEq"`
	AdjEqChg                   types.Number `json:"adjEqChg"`
	AvailBal                   types.Number `json:"availBal"`
	AvailBalChg                types.Number `json:"availBalChg"`
	IMR                        types.Number `json:"imr"`
	IMRChg                     types.Number `json:"imrChg"`
	Liab                       types.Number `json:"liab"`
	LiabChg                    types.Number `json:"liabChg"`
	LiabChgCurrency            string       `json:"liabChgCcy"`
	LiquidiationPrice          types.Number `json:"liqPx"`
	LiquidiationPriceDiff      string       `json:"liqPxDiff"`
	LiquidiationPriceDiffRatio types.Number `json:"liqPxDiffRatio"`
	MgnRatio                   types.Number `json:"mgnRatio"`
	MgnRatioChg                types.Number `json:"mgnRatioChg"`
	MMR                        types.Number `json:"mmr"`
	MMRChange                  types.Number `json:"mmrChg"`
	PosBalance                 types.Number `json:"posBal"`
	PosBalChange               types.Number `json:"posBalChg"`
	Type                       string       `json:"type"`
}

// AnnouncementDetail represents an exchange's announcement detail
type AnnouncementDetail struct {
	Details []struct {
		AnnouncementType string     `json:"annType"`
		PushTime         types.Time `json:"pTime"`
		Title            string     `json:"title"`
		URL              string     `json:"url"`
	} `json:"details"`
	TotalPage types.Number `json:"totalPage"`
}

// AnnouncementTypeInfo represents an announcement type sample and it's description
type AnnouncementTypeInfo struct {
	AnnouncementType     string `json:"annType"`
	AnnouncementTypeDesc string `json:"annTypeDesc"`
}

// FiatOrderDetail represents a fiat deposit/withdrawal order detail
type FiatOrderDetail struct {
	CreatTime       types.Time   `json:"cTime"`
	UpdateTime      types.Time   `json:"uTime"`
	OrdID           string       `json:"ordId"`
	PaymentMethod   string       `json:"paymentMethod"`
	PaymentAcctID   string       `json:"paymentAcctId"`
	Amount          types.Number `json:"amt"`
	Fee             types.Number `json:"fee"`
	Currency        string       `json:"ccy"`
	State           string       `json:"state"`
	ClientID        string       `json:"clientId"`
	PaymentMethodID string       `json:"paymentMethodId,omitempty"`
}

// OrderIDAndState represents an orderID and state information
type OrderIDAndState struct {
	OrderID string `json:"ordId"`
	State   string `json:"state"`
}

// FiatWithdrawalPaymentMethods represents a detailed information about fiat asset withdrawal payment methods and accounts
type FiatWithdrawalPaymentMethods struct {
	Currency      string       `json:"ccy"`
	PaymentMethod string       `json:"paymentMethod"`
	FeeRate       types.Number `json:"feeRate"`
	MinFee        types.Number `json:"minFee"`
	Limits        struct {
		DailyLimit            types.Number `json:"dailyLimit"`
		DailyLimitRemaining   types.Number `json:"dailyLimitRemaining"`
		WeeklyLimit           types.Number `json:"weeklyLimit"`
		WeeklyLimitRemaining  types.Number `json:"weeklyLimitRemaining"`
		MonthlyLimit          types.Number `json:"monthlyLimit"`
		MonthlyLimitRemaining types.Number `json:"monthlyLimitRemaining"`
		MaxAmount             types.Number `json:"maxAmt"`
		MinAmount             types.Number `json:"minAmt"`
		LifetimeLimit         types.Number `json:"lifetimeLimit"`
	} `json:"limits"`
	Accounts []struct {
		PaymentAcctID string `json:"paymentAcctId"`
		AccountNumber string `json:"acctNum"`
		RecipientName string `json:"recipientName"`
		BankName      string `json:"bankName"`
		BankCode      string `json:"bankCode"`
		State         string `json:"state"`
	} `json:"accounts"`
}

// FiatDepositPaymentMethods represents a fiat deposit payment methods
type FiatDepositPaymentMethods struct {
	Currency      string       `json:"ccy"`
	PaymentMethod string       `json:"paymentMethod"`
	FeeRate       types.Number `json:"feeRate"`
	MinFee        types.Number `json:"minFee"`
	Limits        struct {
		DailyLimit            types.Number `json:"dailyLimit"`
		DailyLimitRemaining   types.Number `json:"dailyLimitRemaining"`
		WeeklyLimit           types.Number `json:"weeklyLimit"`
		WeeklyLimitRemaining  types.Number `json:"weeklyLimitRemaining"`
		MonthlyLimit          types.Number `json:"monthlyLimit"`
		MonthlyLimitRemaining types.Number `json:"monthlyLimitRemaining"`
		MaxAmount             types.Number `json:"maxAmt"`
		MinAmount             types.Number `json:"minAmt"`
		LifetimeLimit         types.Number `json:"lifetimeLimit"`
	} `json:"limits"`
	Accounts []struct {
		PaymentAcctID string `json:"paymentAcctId"`
		AccountNumber string `json:"acctNum"`
		RecipientName string `json:"recipientName"`
		BankName      string `json:"bankName"`
		BankCode      string `json:"bankCode"`
		State         string `json:"state"`
	} `json:"accounts"`
}

// MonthlyStatement represents the information and download link for a monthly statement document.
type MonthlyStatement struct {
	FileHref  string     `json:"fileHref"`
	State     string     `json:"state"`
	Timestamp types.Time `json:"ts"`
}

type tsResp struct {
	Timestamp types.Time `json:"ts"`
}

type withdrawData struct {
	WithdrawalID string `json:"wdId"`
}
