package okx

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *OrderBookResponse) UnmarshalJSON(data []byte) error {
	type Alias OrderBookResponse
	chil := &struct {
		*Alias
		GenerationTimeStamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	if chil.GenerationTimeStamp > 0 {
		a.GenerationTimeStamp = time.UnixMilli(chil.GenerationTimeStamp)
	}
	return nil
}

// UnmarshalJSON decerialize the timestamp information to TakerVolume.
func (a *TakerVolume) UnmarshalJSON(data []byte) error {
	type Alias TakerVolume
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes the integer timestamp to local time instance.
func (a *TradeResponse) UnmarshalJSON(data []byte) error {
	type Alias TradeResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *TradingVolumdIn24HR) UnmarshalJSON(data []byte) error {
	type Alias TradingVolumdIn24HR
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *OracleSmartContractResponse) UnmarshalJSON(data []byte) error {
	type Alias OracleSmartContractResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"timestamp,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *IndexComponent) UnmarshalJSON(data []byte) error {
	type Alias IndexComponent
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

// NumbersOnlyRegexp for checking the value is numberics only
var NumbersOnlyRegexp = regexp.MustCompile(`^\d*$`)

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *Instrument) UnmarshalJSON(data []byte) error {
	type Alias Instrument
	chil := &struct {
		*Alias
		ListTime       string `json:"listTime"`
		ExpTime        string `json:"expTime"`
		InstrumentType string `json:"instType"`

		MaxLeverage      string `json:"lever"`
		TickSize         string `json:"tickSz"`
		LotSize          string `json:"lotSz"`
		MinimumOrderSize string `json:"minSz"`
	}{
		Alias: (*Alias)(a),
	}

	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	var val float64
	if val, er = strconv.ParseFloat(chil.MaxLeverage, 64); er == nil {
		a.MaxLeverage = val
	}
	if val, er = strconv.ParseFloat(chil.TickSize, 64); er == nil {
		a.TickSize = val
	}
	if val, er = strconv.ParseFloat(chil.LotSize, 64); er == nil {
		a.LotSize = val
	}
	if val, er = strconv.ParseFloat(chil.MinimumOrderSize, 64); er == nil {
		a.MinimumOrderSize = val
	}
	if NumbersOnlyRegexp.MatchString(chil.ListTime) {
		if val, er := strconv.Atoi(chil.ListTime); er == nil {
			a.ListTime = time.UnixMilli(int64(val))
		}
	}
	if NumbersOnlyRegexp.MatchString(chil.ExpTime) {
		if val, er := strconv.Atoi(chil.ExpTime); er == nil {
			a.ExpTime = time.UnixMilli(int64(val))
		}
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualContract
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes the json obeject to the MarginLendRationItem.
func (a *MarginLendRatioItem) UnmarshalJSON(data []byte) error {
	type Alie MarginLendRatioItem
	chil := &struct {
		*Alie
		Timestamp int64 `json:"ts,string"`
	}{
		Alie: (*Alie)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes the json obeject to the DeliveryHistoryResponse
func (a *DeliveryHistory) UnmarshalJSON(data []byte) error {
	type Alias DeliveryHistory
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

// UnmarshalJSON decoder for OpenInterestResponse instance.
func (a *OpenInterest) UnmarshalJSON(data []byte) error {
	type Alias OpenInterest
	chil := &struct {
		*Alias
		Timestamp      int64  `json:"ts,string"`
		InstrumentType string `json:"instType"`
	}{Alias: (*Alias)(a)}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *FundingRateResponse) UnmarshalJSON(data []byte) error {
	type Alias FundingRateResponse
	chil := &struct {
		*Alias
		FundingTime     int64  `json:"fundingTime,string"`
		NextFundingTime string `json:"nextFundingTime"`
		InstrumentType  string `json:"instType"`
		FundingRate     string `json:"fundingRate"`
		NextFundingRate string `json:"nextFundingRate"`
	}{
		Alias: (*Alias)(a),
	}
	if val, er := strconv.ParseFloat(chil.FundingRate, 64); er == nil {
		a.FundingRate = val
	}
	if val, er := strconv.ParseFloat(chil.NextFundingRate, 64); er == nil {
		a.NextFundingRate = val
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.FundingTime > 0 {
		a.FundingTime = time.UnixMilli(chil.FundingTime)
	}
	if NumbersOnlyRegexp.MatchString(chil.NextFundingTime) {
		if val, er := strconv.Atoi(chil.NextFundingTime); er == nil {
			a.NextFundingTime = time.UnixMilli(int64(val))
		}
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *LimitPriceResponse) UnmarshalJSON(data []byte) error {
	type Alias LimitPriceResponse
	chil := &struct {
		*Alias
		Timestamp      int64  `json:"ts,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerialize the account and position response.
func (a *TickerResponse) UnmarshalJSON(data []byte) error {
	type Alias TickerResponse
	chil := &struct {
		*Alias
		InstrumentType           string `json:"instType"`
		TickerDataGenerationTime int64  `json:"ts,string"`
		LastTradePrice           string `json:"last"`
		LastTradeSize            string `json:"lastSz"`
		BestAskPrice             string `json:"askPx"`
		BestAskSize              string `json:"askSz"`
		BidPrice                 string `json:"bidPx"`
		BidSize                  string `json:"bidSz"`
		Open24H                  string `json:"open24h"`
		High24H                  string `json:"high24h"`
		Low24H                   string `json:"low24h"`
		VolCcy24H                string `json:"volCcy24h"`
		Vol24H                   string `json:"vol24h"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	var val float64
	var er error
	if chil.LastTradePrice != "" {
		val, er = strconv.ParseFloat(chil.LastTradePrice, 64)
		if er == nil {
			a.LastTradePrice = val
		}
	}
	if chil.LastTradeSize != "" {
		val, er = strconv.ParseFloat(chil.LastTradeSize, 64)
		if er == nil {
			a.LastTradeSize = val
		}
	}
	if chil.BestAskPrice != "" {
		val, er = strconv.ParseFloat(chil.BestAskPrice, 64)
		if er == nil {
			a.BestAskPrice = val
		}
	}
	if chil.BestAskSize != "" {
		val, er = strconv.ParseFloat(chil.BestAskSize, 64)
		if er == nil {
			a.BestAskSize = val
		}
	}
	if chil.BidPrice != "" {
		val, er = strconv.ParseFloat(chil.BidPrice, 64)
		if er == nil {
			a.BidPrice = val
		}
	}
	if chil.BidSize != "" {
		val, er = strconv.ParseFloat(chil.BidSize, 64)
		if er == nil {
			a.BidSize = val
		}
	}
	if chil.Open24H != "" {
		val, er = strconv.ParseFloat(chil.Open24H, 64)
		if er == nil {
			a.Open24H = val
		}
	}
	if chil.High24H != "" {
		val, er = strconv.ParseFloat(chil.High24H, 64)
		if er == nil {
			a.High24H = val
		}
	}
	if chil.Low24H != "" {
		val, er = strconv.ParseFloat(chil.Low24H, 64)
		if er == nil {
			a.Low24H = val
		}
	}
	if chil.VolCcy24H != "" {
		val, er = strconv.ParseFloat(chil.VolCcy24H, 64)
		if er == nil {
			a.VolCcy24H = val
		}
	}
	if chil.Vol24H != "" {
		val, er = strconv.ParseFloat(chil.Vol24H, 64)
		if er == nil {
			a.Vol24H = val
		}
	}
	if chil.TickerDataGenerationTime > 0 {
		a.TickerDataGenerationTime = time.UnixMilli(chil.TickerDataGenerationTime)
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *OptionMarketDataResponse) UnmarshalJSON(data []byte) error {
	type Alias OptionMarketDataResponse
	chil := &struct {
		*Alias
		Timestamp      int64  `json:"ts,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, asset item, and timestamp information.
func (a *DeliveryEstimatedPrice) UnmarshalJSON(data []byte) error {
	type Alias DeliveryEstimatedPrice
	chil := &struct {
		*Alias
		Timestamp      int64  `json:"ts,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *ServerTime) UnmarshalJSON(data []byte) error {
	type Alias ServerTime
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *LiquidationOrderDetailItem) UnmarshalJSON(data []byte) error {
	type Alias LiquidationOrderDetailItem
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

// UnmarshalJSON custom Unmarshaler to convert the Instrument type string to an asset.Item instance.
func (a *LiquidationOrder) UnmarshalJSON(data []byte) error {
	type Alias LiquidationOrder
	chil := &struct {
		*Alias
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	default:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON unmarshals the timestamp for mark price data
func (a *MarkPrice) UnmarshalJSON(data []byte) error {
	type Alias MarkPrice
	chil := &struct {
		*Alias
		Timestamp      int64  `json:"ts,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *InsuranceFundInformationDetail) UnmarshalJSON(data []byte) error {
	type Alias InsuranceFundInformationDetail
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *OrderDetail) UnmarshalJSON(data []byte) error {
	type Alias OrderDetail
	chil := &struct {
		*Alias
		Side           string `json:"side"`
		UpdateTime     int64  `json:"uTime,string"`
		CreationTime   int64  `json:"cTime,string"`
		InstrumentType string `json:"instType"`

		Leverage     string `json:"lever"`
		RebateAmount string `json:"rebate"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	var er error
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	a.Side, er = order.StringToOrderSide(chil.Side)
	if er != nil {
		return er
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	val, er := strconv.ParseFloat(chil.Leverage, 64)
	if er == nil {
		a.Leverage = val
	}
	if val, er := strconv.ParseFloat(chil.RebateAmount, 64); er == nil {
		a.RebateAmount = val
	}
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *PendingOrderItem) UnmarshalJSON(data []byte) error {
	type Alias PendingOrderItem
	chil := &struct {
		*Alias
		Side           string `json:"side"`
		UpdateTime     int64  `json:"uTime,string"`
		CreationTime   int64  `json:"cTime,string"`
		InstrumentType string `json:"instType"`
		//
		AccumulatedFillSize string `json:"accFillSz"`
		AveragePrice        string `json:"avgPx"`
		FeeCurrency         string `json:"feeCcy"`
		LastFilledSize      string `json:"fillSz"`
		Leverage            string `json:"lever"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	var er error
	a.Side, er = order.StringToOrderSide(chil.Side)
	if er != nil {
		return er
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	if val, er := strconv.ParseFloat(chil.AccumulatedFillSize, 64); er == nil {
		a.AccumulatedFillSize = val
	}
	if val, er := strconv.ParseFloat(chil.AveragePrice, 64); er == nil {
		a.AveragePrice = val
	}
	if val, er := strconv.ParseFloat(chil.FeeCurrency, 64); er == nil {
		a.FeeCurrency = val
	}
	if val, er := strconv.ParseFloat(chil.LastFilledSize, 64); er == nil {
		a.LastFilledSize = val
	}
	if val, er := strconv.ParseFloat(chil.Leverage, 64); er == nil {
		a.Leverage = val
	}
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *TransactionDetail) UnmarshalJSON(data []byte) error {
	type Alias TransactionDetail
	chil := &struct {
		*Alias
		Timestamp      int64  `json:"ts,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *AlgoOrderResponse) UnmarshalJSON(data []byte) error {
	type Alias AlgoOrderResponse
	chil := &struct {
		*Alias
		CreationTime   int64  `json:"cTime,string"`
		TriggerTime    int64  `json:"triggerTime,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	a.TriggerTime = time.UnixMilli(chil.TriggerTime)
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *AccountAssetValuation) UnmarshalJSON(data []byte) error {
	type Alias AccountAssetValuation
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *AssetBillDetail) UnmarshalJSON(data []byte) error {
	type Alias AssetBillDetail
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON to unmarshal the timestamp information to the struct.
func (a *LightningDepositItem) UnmarshalJSON(data []byte) error {
	type Alias LightningDepositItem
	chil := &struct {
		*Alias
		CreationTime int64 `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *DepositHistoryResponseItem) UnmarshalJSON(data []byte) error {
	type Alias DepositHistoryResponseItem
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *LightningWithdrawalResponse) UnmarshalJSON(data []byte) error {
	type Alias LightningWithdrawalResponse
	chil := &struct {
		*Alias
		CreationTime int64 `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *WithdrawalHistoryResponse) UnmarshalJSON(data []byte) error {
	type Alias WithdrawalHistoryResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *LendingHistory) UnmarshalJSON(data []byte) error {
	type Alias LendingHistory
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *EstimateQuoteResponse) UnmarshalJSON(data []byte) error {
	type Alias EstimateQuoteResponse
	chil := &struct {
		*Alias
		QuoteTime int64 `json:"quoteTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.QuoteTime = time.UnixMilli(chil.QuoteTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *ConvertHistory) UnmarshalJSON(data []byte) error {
	type Alias ConvertHistory
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *AccountDetail) UnmarshalJSON(data []byte) error {
	type Alias AccountDetail
	chil := &struct {
		*Alias
		UpdateTime int64 `json:"uTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	chil := &struct {
		*Alias
		UpdateTime int64 `json:"uTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *ConvertTradeResponse) UnmarshalJSON(data []byte) error {
	type Alias ConvertTradeResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *PositionData) UnmarshalJSON(data []byte) error {
	type Alias PositionData
	chil := &struct {
		*Alias
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	default:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *AccountPosition) UnmarshalJSON(data []byte) error {
	type Alias AccountPosition
	chil := &struct {
		*Alias
		CreationTime   int64  `json:"cTime,string"`
		UpdatedTime    int64  `json:"uTime,string"` // Latest time position was adjusted,
		InstrumentType string `json:"instType"`
		PushTime       string `json:"pTime"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	a.UpdatedTime = time.UnixMilli(chil.UpdatedTime)
	if chil.PushTime != "" {
		val, er := strconv.ParseUint(chil.PushTime, 10, 64)
		if er != nil {
			return er
		}
		a.PushTime = time.UnixMilli(int64(val))
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON deserialises the JSON info, asset item instance, and including the timestamp
func (a *AccountPositionHistory) UnmarshalJSON(data []byte) error {
	type Alias AccountPositionHistory
	chil := &struct {
		*Alias
		CreationTime   int64  `json:"cTime,string"`
		UpdateTime     int64  `json:"uTime,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerialize the account and position response, and timestamp information.
func (a *AccountAndPositionRisk) UnmarshalJSON(data []byte) error {
	type Alias AccountAndPositionRisk
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *BillsDetailResponse) UnmarshalJSON(data []byte) error {
	type Alias BillsDetailResponse
	chil := &struct {
		*Alias
		Timestamp      int64  `json:"ts,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *TradeFeeRate) UnmarshalJSON(data []byte) error {
	type Alias TradeFeeRate
	chil := &struct {
		*Alias
		Timestamp      int64  `json:"ts,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	default:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *InterestAccruedData) UnmarshalJSON(data []byte) error {
	type Alias InterestAccruedData
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerialize the account and position response.
func (a *AccountRiskState) UnmarshalJSON(data []byte) error {
	type Alias AccountRiskState
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *BorrowRepayHistory) UnmarshalJSON(data []byte) error {
	type Alias BorrowRepayHistory
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerialize the account and position response.
func (a *BorrowInterestAndLimitResponse) UnmarshalJSON(data []byte) error {
	type Alias BorrowInterestAndLimitResponse
	chil := &struct {
		*Alias
		NextDiscountTime int64 `json:"nextDiscountTime,string"`
		NextInterestTime int64 `json:"nextInterestTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.NextDiscountTime = time.UnixMilli(chil.NextDiscountTime)
	a.NextInterestTime = time.UnixMilli(chil.NextInterestTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *PositionBuilderData) UnmarshalJSON(data []byte) error {
	type Alias PositionBuilderData
	chil := &struct {
		*Alias
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	default:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *PositionBuilderResponse) UnmarshalJSON(data []byte) error {
	type Alias PositionBuilderResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *TimestampResponse) UnmarshalJSON(data []byte) error {
	type Alias TimestampResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *ExecuteQuoteResponse) UnmarshalJSON(data []byte) error {
	type Alias ExecuteQuoteResponse
	chil := &struct {
		*Alias
		CreationTime int64 `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *RfqTradeResponse) UnmarshalJSON(data []byte) error {
	type Alias RfqTradeResponse
	chil := &struct {
		*Alias
		CreationTime int64 `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *PublicTradesResponse) UnmarshalJSON(data []byte) error {
	type Alias PublicTradesResponse
	chil := &struct {
		*Alias
		CreationTime int64 `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *SubaccountInfo) UnmarshalJSON(data []byte) error {
	type Alias SubaccountInfo
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *SubaccountBalanceDetail) UnmarshalJSON(data []byte) error {
	type Alias SubaccountBalanceDetail
	chil := &struct {
		*Alias
		UpdateTime int64 `json:"uTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *SubaccountBillItem) UnmarshalJSON(data []byte) error {
	type Alias SubaccountBillItem
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *SubaccountBalanceResponse) UnmarshalJSON(data []byte) error {
	type Alias SubaccountBalanceResponse
	chil := &struct {
		*Alias
		UpdateTime int64 `json:"uTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *BlockTicker) UnmarshalJSON(data []byte) error {
	type Alias BlockTicker
	chil := &struct {
		*Alias
		Timestamp      int64  `json:"ts,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *IndexTicker) UnmarshalJSON(data []byte) error {
	type Alias IndexTicker
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *LongShortRatio) UnmarshalJSON(data []byte) error {
	type Alias LongShortRatio
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *OpenInterestVolume) UnmarshalJSON(data []byte) error {
	type Alias OpenInterestVolume
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *OpenInterestVolumeRatio) UnmarshalJSON(data []byte) error {
	type Alias OpenInterestVolumeRatio
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *BlockTrade) UnmarshalJSON(data []byte) error {
	type Alias BlockTrade
	chil := &struct {
		*Alias
		Timestamp int64  `json:"ts,string"`
		Side      string `json:"side"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	switch {
	case strings.EqualFold(chil.Side, "buy"):
		a.Side = order.Buy
	case strings.EqualFold(chil.Side, "sell"):
		a.Side = order.Sell
	default:
		a.Side = order.UnknownSide
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *UnitConvertResponse) UnmarshalJSON(data []byte) error {
	type Alias UnitConvertResponse
	chil := &struct {
		*Alias
		ConvertType int `json:"type,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	switch chil.ConvertType {
	case 1:
		a.ConvertType = CurrencyConvertType(1)
	case 2:
		a.ConvertType = CurrencyConvertType(2)
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *GreeksItem) UnmarshalJSON(data []byte) error {
	type Alias GreeksItem
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *GridAlgoSuborder) UnmarshalJSON(data []byte) error {
	type Alias GridAlgoSuborder
	chil := &struct {
		*Alias
		InstrumentType string `json:"instType"`
		UpdateTime     int64  `json:"uTime,string"`
		CreationTime   int64  `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	default:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *GridAlgoOrderResponse) UnmarshalJSON(data []byte) error {
	type Alias GridAlgoOrderResponse
	chil := &struct {
		*Alias
		UpdateTime     int64  `json:"uTime,string"`
		CreationTime   int64  `json:"cTime,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *AlgoOrderPosition) UnmarshalJSON(data []byte) error {
	type Alias AlgoOrderPosition
	chil := &struct {
		*Alias
		UpdateTime     int64  `json:"uTime,string"`
		CreationTime   int64  `json:"cTime,string"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	switch strings.ToUpper(chil.InstrumentType) {
	case okxInstTypeSwap:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeSpot:
		a.InstrumentType = asset.Spot
	case okxInstTypeFutures:
		a.InstrumentType = asset.Futures
	case okxInstTypeOption:
		a.InstrumentType = asset.Option
	case okxInstTypeContract:
		a.InstrumentType = asset.PerpetualSwap
	case okxInstTypeMargin:
		a.InstrumentType = asset.Margin
	case okxInstTypeANY:
		a.InstrumentType = asset.Empty
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *SystemStatusResponse) UnmarshalJSON(data []byte) error {
	type Alias SystemStatusResponse
	chil := &struct {
		*Alias
		Begin    int64  `json:"begin,string"`
		End      int64  `json:"end,string"`
		PushTime string `json:"ts"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Begin = time.UnixMilli(chil.Begin)
	a.End = time.UnixMilli(chil.End)
	if ts, er := strconv.ParseInt(chil.PushTime, 10, 64); er == nil && ts > 0 {
		a.PushTime = time.UnixMilli(ts)
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *QuoteLeg) UnmarshalJSON(data []byte) error {
	type Alias QuoteLeg
	chil := &struct {
		*Alias
		Side string `json:"side"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	chil.Side = strings.ToLower(chil.Side)
	if chil.Side == "buy" {
		a.Side = order.Buy
	} else {
		a.Side = order.Sell
	}
	return nil
}

// MarshalJSON serialized QuoteLeg instance into bytes
func (a *QuoteLeg) MarshalJSON() ([]byte, error) {
	type Alias QuoteLeg
	chil := &struct {
		*Alias
		Side string `json:"side"`
	}{
		Alias: (*Alias)(a),
	}
	if a.Side == order.Buy {
		chil.Side = "buy"
	} else {
		chil.Side = "sell"
	}
	return json.Marshal(chil)
}

// MarshalJSON serialized CreateQuoteParams instance into bytes
func (a *CreateQuoteParams) MarshalJSON() ([]byte, error) {
	type Alias CreateQuoteParams
	chil := &struct {
		*Alias
		QuoteSide string `json:"quoteSide"`
	}{
		Alias: (*Alias)(a),
	}
	if a.QuoteSide == order.Buy {
		chil.QuoteSide = "buy"
	} else {
		chil.QuoteSide = "sell"
	}
	return json.Marshal(chil)
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *RFQResponse) UnmarshalJSON(data []byte) error {
	type Alias RFQResponse
	chil := &struct {
		*Alias
		CreateTime int64 `json:"cTime,string"`
		UpdateTime int64 `json:"uTime,string"`
		ValidUntil int64 `json:"validUntil,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreateTime = time.UnixMilli(chil.CreateTime)
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	a.ValidUntil = time.UnixMilli(chil.ValidUntil)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *QuoteResponse) UnmarshalJSON(data []byte) error {
	type Alias QuoteResponse
	chil := &struct {
		*Alias
		CreationTime int64 `json:"cTime,string"`
		UpdateTime   int64 `json:"uTime,string"`
		ValidUntil   int64 `json:"validUntil,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	a.ValidUntil = time.UnixMilli(chil.ValidUntil)
	return nil
}

// MarshalJSON serializes the WebsocketLoginData object
func (a *WebsocketLoginData) MarshalJSON() ([]byte, error) {
	type Alias WebsocketLoginData
	return json.Marshal(struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Timestamp: a.Timestamp.UTC().Unix(),
		Alias:     (*Alias)(a),
	})
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *WebsocketLoginData) UnmarshalJSON(data []byte) error {
	type Alias WebsocketLoginData
	chil := &struct {
		*Alias
		Timestamp int64 `json:"timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *WSTradeData) UnmarshalJSON(data []byte) error {
	type Alias WSTradeData
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, &chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *BalanceData) UnmarshalJSON(data []byte) error {
	type Alias BalanceData
	chil := &struct {
		*Alias
		UpdateTime int64 `json:"uTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, &chil); er != nil {
		return er
	}
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *BalanceAndPositionData) UnmarshalJSON(data []byte) error {
	type Alias BalanceAndPositionData
	chil := &struct {
		*Alias
		PushTime string `json:"pTime"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, &chil); er != nil {
		return er
	}
	if val, er := strconv.ParseInt(chil.PushTime, 10, 64); er != nil {
		a.PushTime = time.UnixMilli(val)
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *PositionDataDetail) UnmarshalJSON(data []byte) error {
	type Alias PositionDataDetail
	chil := &struct {
		*Alias
		UpdateTime int64 `json:"uTIme,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *WsAlgoOrderDetail) UnmarshalJSON(data []byte) error {
	type Alias WsAlgoOrderDetail
	chil := &struct {
		*Alias
		TriggerTime  int64 `json:"triggerTime,string"`
		CreationTime int64 `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, &chil); er != nil {
		return er
	}
	a.TriggerTime = time.UnixMilli(chil.TriggerTime)
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *WsAdvancedAlgoOrderDetail) UnmarshalJSON(data []byte) error {
	type Alias WsAdvancedAlgoOrderDetail
	chil := &struct {
		*Alias
		CreationTime string `json:"cTime"`
		TriggerTime  string `json:"triggerTime"`
		PushTime     string `json:"pTime"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if val, er := strconv.ParseInt(chil.CreationTime, 10, 64); er != nil {
		a.CreationTime = time.UnixMilli(val)
	}
	if val, er := strconv.ParseInt(chil.TriggerTime, 10, 64); er != nil {
		a.TriggerTime = time.UnixMilli(val)
	}
	if val, er := strconv.ParseInt(chil.PushTime, 10, 64); er != nil {
		a.PushTime = time.UnixMilli(val)
	}
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *WsGreekData) UnmarshalJSON(data []byte) error {
	type Alias WsGreekData
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *WsQuoteData) UnmarshalJSON(data []byte) error {
	type Alias WsQuoteData
	chil := &struct {
		*Alias
		ValidUntil   int64 `json:"validUntil,string"`
		UpdatedTime  int64 `json:"uTime,string"`
		CreationTime int64 `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.ValidUntil = time.UnixMilli(chil.ValidUntil)
	a.UpdatedTime = time.UnixMilli(chil.UpdatedTime)
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *WsBlocTradeResponse) UnmarshalJSON(data []byte) error {
	type Alias WsBlocTradeResponse
	chil := &struct {
		*Alias
		CreationTime int64 `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *SpotGridAlgoData) UnmarshalJSON(data []byte) error {
	type Alias SpotGridAlgoData
	chil := &struct {
		*Alias
		TriggerTime  int64 `json:"triggerTime,string"`
		CreationTime int64 `json:"cTime,string"`
		PushTime     int64 `json:"pTime,string"`
		UpdateTime   int64 `json:"uTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.TriggerTime = time.UnixMilli(chil.TriggerTime)
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	a.PushTime = time.UnixMilli(chil.PushTime)
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *ContractGridAlgoOrder) UnmarshalJSON(data []byte) error {
	type Alias ContractGridAlgoOrder
	chil := &struct {
		*Alias
		CreationTime int64 `json:"cTime,string"`
		PushTime     int64 `json:"pTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	a.PushTime = time.UnixMilli(chil.PushTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *GridPositionData) UnmarshalJSON(data []byte) error {
	type Alias GridPositionData
	chil := &struct {
		*Alias
		PushTime     int64 `json:"pTime,string"`
		UpdateTime   int64 `json:"uTime,string"`
		CreationTime int64 `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.PushTime = time.UnixMilli(chil.PushTime)
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *WsOrderBookData) UnmarshalJSON(data []byte) error {
	type Alias WsOrderBookData
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON decerializes JSON, and timestamp information.
func (a *GridSubOrderData) UnmarshalJSON(data []byte) error {
	type Alias GridSubOrderData
	chil := &struct {
		*Alias
		PushTime   int64 `json:"pTime,string"`
		UpdateTime int64 `json:"uTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.PushTime = time.UnixMilli(chil.PushTime)
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	return nil
}
