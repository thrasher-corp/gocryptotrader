package mexc

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Futures contract states
const (
	FuturesContractStateEnabled   = 0
	FuturesContractStateDelivery  = 1
	FuturesContractStateDelivered = 2
	FuturesContractStateOffline   = 3
	FuturesContractStatePaused    = 4
)

// Futures contract types
const (
	FuturesTypePerpetual = 1
	FuturesTypeDelivery  = 2
)

// FuturesID is an identifier which the futures API sends as a JSON string on some replies and as a bare number on
// others. A bare number is read as an unsigned integer, since order IDs above 2^53 lose digits as a float64
type FuturesID string

// UnmarshalJSON decodes an identifier from a JSON string or number. The venue sends an ID of zero where there is
// none, such as the order of an untriggered plan order, so zero and null decode as an empty ID
func (f *FuturesID) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*f = ""
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s == "0" {
			s = ""
		}
		*f = FuturesID(s)
		return nil
	}
	id, err := strconv.ParseUint(string(data), 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %q", errInvalidFuturesID, data)
	}
	*f = ""
	if id != 0 {
		*f = FuturesID(strconv.FormatUint(id, 10))
	}
	return nil
}

// String returns the identifier
func (f FuturesID) String() string {
	return string(f)
}

// oneOrMany decodes a reply which is one object when a single symbol is asked for and a list otherwise
type oneOrMany[T any] []T

// UnmarshalJSON decodes a JSON object or list
func (o *oneOrMany[T]) UnmarshalJSON(data []byte) error {
	if data = bytes.TrimSpace(data); len(data) > 0 && data[0] == '{' {
		var v T
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*o = oneOrMany[T]{v}
		return nil
	}
	var v []T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = v
	return nil
}

// pagedList decodes a reply documented as a page and shown as a bare list: it reads the page's resultList or the
// list itself
type pagedList[T any] []T

// UnmarshalJSON decodes a page or a JSON list
func (p *pagedList[T]) UnmarshalJSON(data []byte) error {
	if data = bytes.TrimSpace(data); len(data) > 0 && data[0] == '{' {
		var page FuturesPageResponse[T]
		if err := json.Unmarshal(data, &page); err != nil {
			return err
		}
		*p = page.ResultList
		return nil
	}
	var v []T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*p = v
	return nil
}

// FuturesPageResponse is a page of a paged futures reply
type FuturesPageResponse[T any] struct {
	PageSize    uint64 `json:"pageSize"`
	TotalCount  uint64 `json:"totalCount"`
	TotalPage   uint64 `json:"totalPage"`
	CurrentPage uint64 `json:"currentPage"`
	ResultList  []T    `json:"resultList"`
}

// FuturesContractResponse holds a futures contract's details
type FuturesContractResponse struct {
	Symbol             string        `json:"symbol"`
	DisplayName        string        `json:"displayName"`
	DisplayNameEnglish string        `json:"displayNameEn"`
	PositionOpenType   uint8         `json:"positionOpenType"`
	BaseCoin           currency.Code `json:"baseCoin"`
	QuoteCoin          currency.Code `json:"quoteCoin"`
	BaseCoinName       string        `json:"baseCoinName"`
	QuoteCoinName      string        `json:"quoteCoinName"`
	FutureType         uint8         `json:"futureType"`
	SettleCoin         currency.Code `json:"settleCoin"`
	// ContractSize is the size of one contract: in the base currency for a linear contract and in the quote
	// currency for an inverse one
	ContractSize                     float64 `json:"contractSize"`
	MinLeverage                      uint64  `json:"minLeverage"`
	MaxLeverage                      uint64  `json:"maxLeverage"`
	CountryConfigContractMaxLeverage uint64  `json:"countryConfigContractMaxLeverage"`
	PriceScale                       uint64  `json:"priceScale"`
	VolumeScale                      uint64  `json:"volScale"`
	AmountScale                      uint64  `json:"amountScale"`
	PriceUnit                        float64 `json:"priceUnit"`
	VolumeUnit                       float64 `json:"volUnit"`
	MinVolume                        float64 `json:"minVol"`
	MaxVolume                        float64 `json:"maxVol"`
	BidLimitPriceRate                float64 `json:"bidLimitPriceRate"`
	AskLimitPriceRate                float64 `json:"askLimitPriceRate"`
	TakerFeeRate                     float64 `json:"takerFeeRate"`
	MakerFeeRate                     float64 `json:"makerFeeRate"`
	MaintenanceMarginRate            float64 `json:"maintenanceMarginRate"`
	InitialMarginRate                float64 `json:"initialMarginRate"`
	RiskBaseVolume                   float64 `json:"riskBaseVol"`
	RiskIncrementVolume              float64 `json:"riskIncrVol"`
	// RiskLongShortSwitch is 1 when long and short positions have separate risk limits
	RiskLongShortSwitch                uint8    `json:"riskLongShortSwitch"`
	RiskIncrementMaintenanceMarginRate float64  `json:"riskIncrMmr"`
	RiskIncrementInitialMarginRate     float64  `json:"riskIncrImr"`
	RiskLevelLimit                     uint64   `json:"riskLevelLimit"`
	PriceCoefficientVariation          float64  `json:"priceCoefficientVariation"`
	IndexOrigin                        []string `json:"indexOrigin"`
	State                              uint8    `json:"state"`
	IsNew                              bool     `json:"isNew"`
	IsHot                              bool     `json:"isHot"`
	IsHidden                           bool     `json:"isHidden"`
	ConceptPlate                       []string `json:"conceptPlate"`
	// ConceptPlateID is signed, since the venue lists -1 among some contracts' sector IDs
	ConceptPlateID             []int64                `json:"conceptPlateId"`
	RiskLimitType              string                 `json:"riskLimitType"`
	MaxNumberOfOrders          []uint64               `json:"maxNumOrders"`
	MarketOrderMaxLevel        uint64                 `json:"marketOrderMaxLevel"`
	MarketOrderPriceLimitRate1 float64                `json:"marketOrderPriceLimitRate1"`
	MarketOrderPriceLimitRate2 float64                `json:"marketOrderPriceLimitRate2"`
	TriggerProtect             float64                `json:"triggerProtect"`
	Appraisal                  uint8                  `json:"appraisal"`
	ShowAppraisalCountdown     uint8                  `json:"showAppraisalCountdown"`
	AutomaticDelivery          uint8                  `json:"automaticDelivery"`
	APIAllowed                 bool                   `json:"apiAllowed"`
	DepthStepList              []types.Number         `json:"depthStepList"`
	LimitMaxVolume             float64                `json:"limitMaxVol"`
	Threshold                  uint64                 `json:"threshold"`
	BaseCoinIconURL            string                 `json:"baseCoinIconUrl"`
	ID                         uint64                 `json:"id"`
	SettleCoinID               string                 `json:"vid"`
	BaseCoinID                 string                 `json:"baseCoinId"`
	CreateTime                 types.Time             `json:"createTime"`
	OpeningTime                types.Time             `json:"openingTime"`
	OpeningCountdownOption     uint8                  `json:"openingCountdownOption"`
	ShowBeforeOpen             bool                   `json:"showBeforeOpen"`
	IsMaxLeverage              bool                   `json:"isMaxLeverage"`
	IsZeroFeeRate              bool                   `json:"isZeroFeeRate"`
	RiskLimitMode              string                 `json:"riskLimitMode"`
	IsZeroFeeSymbol            bool                   `json:"isZeroFeeSymbol"`
	RiskLimitCustom            []FuturesRiskLimitTier `json:"riskLimitCustom"`
	LiquidationFeeRate         float64                `json:"liquidationFeeRate"`
	FeeRateMode                string                 `json:"feeRateMode"`
	TieredFeeRates             []FuturesTieredFeeRate `json:"tieredFeeRates"`
	// PairType is 1 for a normal pair and 2 for a suspended one
	PairType                    uint8      `json:"type"`
	StopOnlyFair                bool       `json:"stopOnlyFair"`
	PreMarket                   bool       `json:"preMarket"`
	TypeLabel                   uint8      `json:"typeLabel"`
	FullName                    string     `json:"fn"`
	FeeRateType                 string     `json:"feeRateType"`
	TagIDList                   []uint64   `json:"tagIdList"`
	DeliveryTime                types.Time `json:"deliveryTime"`
	RegularMaxLeverage          uint64     `json:"regularMaxLeverage"`
	TemporaryMaxLeverage        uint64     `json:"tempMaxLeverage"`
	TemporaryMaxLeverageLimited bool       `json:"tempMaxLeverageLimited"`
	TemporaryMaxLeverageReason  string     `json:"tempMaxLeverageReason"`
}

// IsInverse reports whether the contract is coin margined: its size is counted in the quote currency and it settles
// in the base currency
func (c *FuturesContractResponse) IsInverse() bool {
	return !c.SettleCoin.IsEmpty() && c.SettleCoin.Equal(c.BaseCoin) && !c.SettleCoin.Equal(c.QuoteCoin)
}

// FuturesRiskLimitTier is a custom risk limit tier of a contract
type FuturesRiskLimitTier struct {
	Level                 uint64  `json:"level"`
	MaxVolume             float64 `json:"maxVol"`
	MaintenanceMarginRate float64 `json:"mmr"`
	InitialMarginRate     float64 `json:"imr"`
	MaxLeverage           uint64  `json:"maxLeverage"`
}

// FuturesTieredFeeRate is a band of a tiered fee schedule
type FuturesTieredFeeRate struct {
	TakerFeeRate        float64 `json:"takerFeeRate"`
	MakerFeeRate        float64 `json:"makerFeeRate"`
	MinTieredDealAmount float64 `json:"minTieredDealAmount"`
	MaxTieredDealAmount float64 `json:"maxTieredDealAmount"`
	RealTakerFee        float64 `json:"realTakerFee"`
	RealMakerFee        float64 `json:"realMakerFee"`
}

// FuturesOrderbookLevels decodes futures order book levels sent as [price, contracts, order count]
type FuturesOrderbookLevels orderbook.Levels

// UnmarshalJSON decodes the levels of an order book side
func (l *FuturesOrderbookLevels) UnmarshalJSON(data []byte) error {
	var levels [][]float64
	if err := json.Unmarshal(data, &levels); err != nil {
		return err
	}
	*l = make(FuturesOrderbookLevels, len(levels))
	for i, level := range levels {
		if len(level) < 2 {
			return fmt.Errorf("%w: %v", errInvalidOrderbookLevel, level)
		}
		(*l)[i].Price = level[0]
		(*l)[i].Amount = level[1]
		if len(level) > 2 {
			(*l)[i].OrderCount = int64(level[2])
		}
	}
	return nil
}

// Levels returns the levels as order book levels
func (l FuturesOrderbookLevels) Levels() orderbook.Levels {
	return orderbook.Levels(l)
}

// FuturesOrderbookResponse holds a futures order book; amounts are in contracts
type FuturesOrderbookResponse struct {
	Asks               FuturesOrderbookLevels `json:"asks"`
	Bids               FuturesOrderbookLevels `json:"bids"`
	Version            uint64                 `json:"version"`
	Timestamp          types.Time             `json:"timestamp"`
	MatchingEngineTime types.Time             `json:"cts"`
}

// FuturesDepthSnapshotResponse holds one of the latest order book changes; amounts are in contracts
type FuturesDepthSnapshotResponse struct {
	Asks               FuturesOrderbookLevels `json:"asks"`
	Bids               FuturesOrderbookLevels `json:"bids"`
	Version            uint64                 `json:"version"`
	MatchingEngineTime types.Time             `json:"cts"`
}

// FuturesIndexPriceResponse holds a contract's index price
type FuturesIndexPriceResponse struct {
	Symbol     string     `json:"symbol"`
	IndexPrice float64    `json:"indexPrice"`
	Timestamp  types.Time `json:"timestamp"`
}

// FuturesFairPriceResponse holds a contract's fair price
type FuturesFairPriceResponse struct {
	Symbol    string     `json:"symbol"`
	FairPrice float64    `json:"fairPrice"`
	Timestamp types.Time `json:"timestamp"`
}

// FuturesFundingRateResponse holds a contract's current funding rate, settled at NextSettleTime
type FuturesFundingRateResponse struct {
	Symbol         string  `json:"symbol"`
	FundingRate    float64 `json:"fundingRate"`
	MaxFundingRate float64 `json:"maxFundingRate"`
	MinFundingRate float64 `json:"minFundingRate"`
	// CollectCycle is the funding interval in hours
	CollectCycle   uint64     `json:"collectCycle"`
	NextSettleTime types.Time `json:"nextSettleTime"`
	Timestamp      types.Time `json:"timestamp"`
	IndexPrice     float64    `json:"idxPrice"`
	FairPrice      float64    `json:"fairPrice"`
}

// FuturesCandlesRequest holds the parameters of a futures candle request
type FuturesCandlesRequest struct {
	Symbol   currency.Pair
	Interval kline.Interval
	// StartTime and EndTime are sent in whole seconds; either may be left unset
	StartTime time.Time
	EndTime   time.Time
}

// FuturesCandlesResponse holds futures candles as parallel lists. Volume is in contracts and Amount in the
// contract's turnover currency; index and fair price candles carry no volume
type FuturesCandlesResponse struct {
	Time      []types.Time `json:"time"`
	Open      []float64    `json:"open"`
	Close     []float64    `json:"close"`
	High      []float64    `json:"high"`
	Low       []float64    `json:"low"`
	Volume    []float64    `json:"vol"`
	Amount    []float64    `json:"amount"`
	RealOpen  []float64    `json:"realOpen"`
	RealClose []float64    `json:"realClose"`
	RealHigh  []float64    `json:"realHigh"`
	RealLow   []float64    `json:"realLow"`
}

// FuturesTradeResponse holds a public futures trade; Volume is in contracts
type FuturesTradeResponse struct {
	Price  float64 `json:"p"`
	Volume float64 `json:"v"`
	// Side is 1 when the taker bought and 2 when it sold
	Side uint8 `json:"T"`
	// OpenFlag is 1 when both sides opened positions, 2 when neither did and 3 otherwise
	OpenFlag uint8 `json:"O"`
	// SelfTrade is 1 for a self trade and 2 otherwise
	SelfTrade          uint8      `json:"M"`
	Time               types.Time `json:"t"`
	TradeID            FuturesID  `json:"i"`
	MatchingEngineTime types.Time `json:"cts"`
}

// FuturesTickerResponse holds a contract's ticker. Volume24Hour and HoldVolume are in contracts, and Amount24Hour
// is the turnover in the quote currency for a linear contract and in the base currency for an inverse one
type FuturesTickerResponse struct {
	ContractID              uint64                   `json:"contractId"`
	Symbol                  string                   `json:"symbol"`
	LastPrice               float64                  `json:"lastPrice"`
	BestBidPrice            float64                  `json:"bid1"`
	BestAskPrice            float64                  `json:"ask1"`
	Volume24Hour            float64                  `json:"volume24"`
	Amount24Hour            float64                  `json:"amount24"`
	HoldVolume              float64                  `json:"holdVol"`
	Low24Hour               float64                  `json:"lower24Price"`
	High24Hour              float64                  `json:"high24Price"`
	PriceChangeRate         float64                  `json:"riseFallRate"`
	PriceChange             float64                  `json:"riseFallValue"`
	IndexPrice              float64                  `json:"indexPrice"`
	FairPrice               float64                  `json:"fairPrice"`
	FundingRate             float64                  `json:"fundingRate"`
	MaxBidPrice             float64                  `json:"maxBidPrice"`
	MinAskPrice             float64                  `json:"minAskPrice"`
	Timestamp               types.Time               `json:"timestamp"`
	PriceChangeRates        *FuturesPriceChangeRates `json:"riseFallRates"`
	PriceChangeRatesOfZones []float64                `json:"riseFallRatesOfTimezone"`
}

// FuturesPriceChangeRates holds a ticker's price changes over several periods
type FuturesPriceChangeRates struct {
	Zone        string  `json:"zone"`
	Rate        float64 `json:"r"`
	Value       float64 `json:"v"`
	Rate7Days   float64 `json:"r7"`
	Rate30Days  float64 `json:"r30"`
	Rate90Days  float64 `json:"r90"`
	Rate180Days float64 `json:"r180"`
	Rate365Days float64 `json:"r365"`
}

// FuturesInsuranceFundResponse holds a contract's insurance fund balance
type FuturesInsuranceFundResponse struct {
	Symbol    string        `json:"symbol"`
	Currency  currency.Code `json:"currency"`
	Available float64       `json:"available"`
	Timestamp types.Time    `json:"timestamp"`
}

// FuturesPageRequest holds the paging parameters of a paged futures request
type FuturesPageRequest struct {
	// PageNumber starts at 1; 0 asks for the first page
	PageNumber uint64
	// PageSize defaults to 20 when 0
	PageSize uint64
}

// FuturesInsuranceFundSnapshotResponse is a recorded insurance fund balance
type FuturesInsuranceFundSnapshotResponse struct {
	Symbol       string        `json:"symbol"`
	Currency     currency.Code `json:"currency"`
	Available    float64       `json:"available"`
	SnapshotTime types.Time    `json:"snapshotTime"`
}

// FuturesFundingRateHistoryResponse is a settled funding rate
type FuturesFundingRateHistoryResponse struct {
	Symbol      string     `json:"symbol"`
	FundingRate float64    `json:"fundingRate"`
	SettleTime  types.Time `json:"settleTime"`
	// CollectCycle is the funding interval in hours
	CollectCycle uint64 `json:"collectCycle"`
}
