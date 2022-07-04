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

// UnmarshalJSON
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
var NumbersOnlyRegexp = regexp.MustCompile("^[0-9]*$")

// UnmarshalJSON
func (a *Instrument) UnmarshalJSON(data []byte) error {
	type Alias Instrument
	chil := &struct {
		*Alias
		ListTime       string `json:"listTime"`
		ExpTime        string `json:"expTime"`
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}

	if er := json.Unmarshal(data, chil); er != nil {
		return er
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
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
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
func (a *OpenInterestResponse) UnmarshalJSON(data []byte) error {
	type Alias OpenInterestResponse
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

func (a *FundingRateResponse) UnmarshalJSON(data []byte) error {
	type Alias FundingRateResponse
	chil := &struct {
		*Alias
		FundingTime     int64  `json:"fundingTime,string"`
		NextFundingTime string `json:"nextFundingTime"`
		InstrumentType  string `json:"instType"`
	}{
		Alias: (*Alias)(a),
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

// UnmarshalJSON decerialize the account and position response.
func (a *MarketDataResponse) UnmarshalJSON(data []byte) error {
	type Alias MarketDataResponse
	chil := &struct {
		*Alias
		InstrumentType           string `json:"instType"`
		TickerDataGenerationTime int64  `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.TickerDataGenerationTime > 0 {
		a.TickerDataGenerationTime = time.UnixMilli(chil.TickerDataGenerationTime)
	}
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

// UnmarshalJSON
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

// UnmarshalJSON decerialize the timestamp.
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	default:
		a.InstrumentType = asset.Item("ANY")
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
		return nil
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

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

func (a *OrderDetail) UnmarshalJSON(data []byte) error {
	type Alias OrderDetail
	chil := &struct {
		*Alias
		Side           string `json:"side"`
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
	a.Side = order.ParseOrderSideString(chil.Side)
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

func (a *PendingOrderItem) UnmarshalJSON(data []byte) error {
	type Alias PendingOrderItem
	chil := &struct {
		*Alias
		Side           string `json:"side"`
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
	a.Side = order.ParseOrderSideString(chil.Side)
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

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

// UnmarshalJSON a custom unmarshaling function implementing the Unmarshaler interface to safely unmarshal the incomming messages.
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

// UnmarshalJSON a custom unmarshaling function to convert unix creation time n millisecond to built in golang time.Time instance.
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

// WithdrawalHistoryResponse a custom function to unmarshal timestamp json
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

// UnmsrhalJSON convert timestamp unix miliseconds to builtin time.
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

// UnmarshalJSON the unmarshal support method to convert the
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

// UnmarshalJSON convert timestamp unix millisecond to built in Time object
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

// UnmarshalJSON custome unmarshal method to convert the update time to built in time.Time instance.
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

// UnmarshalJSON custome unmarshal method to convert the update time to built in time.Time instance.
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

// UnmarshalJSON custom Unmarshaler to convert the Instrument type string to an asset.Item instance.
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	default:
		a.InstrumentType = asset.Item("ANY")
	}
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp (creation time and update time).
func (a *AccountPosition) UnmarshalJSON(data []byte) error {
	type Alias AccountPosition
	chil := &struct {
		*Alias
		CreationTime   int64  `json:"cTime,string"`
		UpdatedTime    int64  `json:"uTime,string"` // Latest time position was adjusted,
		InstrumentType string `json:"instType"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	a.UpdatedTime = time.UnixMilli(chil.UpdatedTime)
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

// UnmarshalJSON decerialize the account and position response and custom Unmarshaler to convert the Instrument type string to an asset.Item instance.
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	default:
		a.InstrumentType = asset.Item("ANY")
	}
	return nil
}

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON custom unmarshaler to convert the Instrument type string to an asset.Item instance.
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	default:
		a.InstrumentType = asset.Item("ANY")
	}
	return nil
}

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

// UnmarshalJSON decerialize the timestamp.
func (a *OKXIndexTickerResponse) UnmarshalJSON(data []byte) error {
	type Alias OKXIndexTickerResponse
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

// UnmarshalJSON decerialize the timestamp.
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

// UnmarshalJSON decerialize the timestamp.
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

// UnmarshalJSON decerialize the timestamp.
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

// UnmarshalJSON decerialize the account and position response.
func (a *BlockTrade) UnmarshalJSON(data []byte) error {
	type Alias BlockTrade
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

// UnmarshalJSON decerialize the timestamp.
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

// UnmarshalJSON custom Unmarshaler to convert the Instrument type string to an asset.Item instance.
func (a *GridAlgoSuborders) UnmarshalJSON(data []byte) error {
	type Alias GridAlgoSuborders
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	default:
		a.InstrumentType = asset.Item("ANY")
	}
	return nil
}

// UnmarshalJSON decerialize the account and position response.
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

// UnmarshalJSON decerialize the account and position response.
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
	chil.InstrumentType = strings.ToUpper(strings.Trim(chil.InstrumentType, " "))
	switch chil.InstrumentType {
	case "SWAP":
		a.InstrumentType = asset.PerpetualSwap
	case "SPOT":
		a.InstrumentType = asset.Spot
	case "FUTURES":
		a.InstrumentType = asset.Futures
	case "OPTION":
		a.InstrumentType = asset.Option
	case "CONTRACT":
		a.InstrumentType = asset.PerpetualSwap
	case "MARGIN":
		a.InstrumentType = asset.Margin
	case "ANY":
		a.InstrumentType = asset.Item("")
	}
	return nil
}

// UnmarshalJSON decerialize the account and position response.
func (a *SystemStatusResponse) UnmarshalJSON(data []byte) error {
	type Alias SystemStatusResponse
	chil := &struct {
		*Alias
		Begin int64 `json:"begin,string"`
		End   int64 `json:"end,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	a.Begin = time.UnixMilli(chil.Begin)
	a.End = time.UnixMilli(chil.End)
	return nil
}

// UnmarshalJSON decerialize the account and position response.
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

// UnmarshalJSON decerialize the account and position response.
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
