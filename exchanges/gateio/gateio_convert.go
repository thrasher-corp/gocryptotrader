package gateio

import (
	"encoding/json"
	"math"
	"strconv"
	"time"
)

// UnmarshalJSON decerializes json, and timestamp information.
func (a *CrossMarginAccountHistoryItem) UnmarshalJSON(data []byte) error {
	type Alias CrossMarginAccountHistoryItem
	chil := &struct {
		*Alias
		Time int64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Time = time.Unix(chil.Time, 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *DeliveryTradingHistory) UnmarshalJSON(data []byte) error {
	type Alias DeliveryTradingHistory
	chil := &struct {
		*Alias
		CreateTime float64 `json:"create_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(int64(math.Round(chil.CreateTime)), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *FlashSwapOrderResponse) UnmarshalJSON(data []byte) error {
	type Alias FlashSwapOrderResponse
	chil := &struct {
		*Alias
		CreateTime int64 `json:"create_time"`
		UpdateTime int64 `json:"update_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.UnixMilli(chil.CreateTime)
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *RepaymentHistoryItem) UnmarshalJSON(data []byte) error {
	type Alias RepaymentHistoryItem
	chil := &struct {
		*Alias
		CreateTime int64 `json:"create_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(chil.CreateTime, 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *TriggerTimeResponse) UnmarshalJSON(data []byte) error {
	type Alias TriggerTimeResponse
	chil := &struct {
		*Alias
		TriggerTime int64 `json:"trigger_time,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.TriggerTime = time.Unix(chil.TriggerTime, 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *LoanRepaymentRecord) UnmarshalJSON(data []byte) error {
	type Alias LoanRepaymentRecord
	chil := &struct {
		*Alias
		CreateTime int64 `json:"create_time,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(chil.CreateTime, 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *CrossMarginLoanResponse) UnmarshalJSON(data []byte) error {
	type Alias CrossMarginLoanResponse
	chil := &struct {
		*Alias
		CreateTime int64 `json:"create_time"`
		UpdateTime int64 `json:"update_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(chil.CreateTime, 0)
	a.UpdateTime = time.Unix(chil.UpdateTime, 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *LoanRecord) UnmarshalJSON(data []byte) error {
	type Alias LoanRecord
	chil := &struct {
		*Alias
		CreateTime int64 `json:"create_time,string"`
		ExpireTime int64 `json:"expire_time,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(chil.CreateTime, 0)
	a.ExpireTime = time.Unix(chil.ExpireTime, 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *SpotPriceTriggeredOrder) UnmarshalJSON(data []byte) error {
	type Alias SpotPriceTriggeredOrder
	chil := &struct {
		*Alias
		CreationTime int64 `json:"ctime"`
		FireTime     int64 `json:"ftime"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreationTime = time.Unix(chil.CreationTime, 0)
	a.FireTime = time.Unix(chil.FireTime, 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *SpotPersonalTradeHistory) UnmarshalJSON(data []byte) error {
	type Alias SpotPersonalTradeHistory
	chil := &struct {
		*Alias
		CreateTime   float64 `json:"create_time,string"`
		CreateTimeMs float64 `json:"create_time_ms,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(int64(chil.CreateTime), 0)
	a.CreateTimeMs = time.UnixMilli(int64(chil.CreateTime))
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *SubAccountTransferResponse) UnmarshalJSON(data []byte) error {
	type Alias SubAccountTransferResponse
	chil := &struct {
		*Alias
		Timestamp float64 `json:"timest,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Timestamp = time.Unix(int64(chil.Timestamp), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *WithdrawalResponse) UnmarshalJSON(data []byte) error {
	type Alias WithdrawalResponse
	chil := &struct {
		*Alias
		Timestamp float64 `json:"timestamp,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Timestamp = time.Unix(int64(chil.Timestamp), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *OptionTradingHistory) UnmarshalJSON(data []byte) error {
	type Alias OptionTradingHistory
	chil := &struct {
		*Alias
		CreateTime float64 `json:"create_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(int64(chil.CreateTime), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *SettlementHistoryItem) UnmarshalJSON(data []byte) error {
	type Alias SettlementHistoryItem
	chil := &struct {
		*Alias
		Time float64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Time = time.Unix(int64(chil.Time), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *OptionOrderResponse) UnmarshalJSON(data []byte) error {
	type Alias OptionOrderResponse
	chil := &struct {
		*Alias
		CreateTime float64 `json:"create_time"`
		FinishTime float64 `json:"finish_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(int64(chil.CreateTime), 0)
	a.FinishTime = time.Unix(int64(chil.FinishTime), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *ContractClosePosition) UnmarshalJSON(data []byte) error {
	type Alias ContractClosePosition
	chil := &struct {
		*Alias
		PositionCloseTime float64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.PositionCloseTime = time.Unix(int64(chil.PositionCloseTime), 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *AccountBook) UnmarshalJSON(data []byte) error {
	type Alias AccountBook
	chil := &struct {
		*Alias
		ChangeTime float64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.ChangeTime = time.Unix(int64(chil.ChangeTime), 0)
	return nil
}

// UnmarshalJSON decerializes json data into CurrencyPairDetail
func (a *CurrencyPairDetail) UnmarshalJSON(data []byte) error {
	type Alias CurrencyPairDetail
	chil := struct {
		*Alias
		Fee            string `json:"fee"`
		MinBaseAmount  string `json:"min_base_amount"`
		MinQuoteAmount string `json:"min_quote_amount"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, &chil)
	if err != nil {
		return err
	}
	if chil.Fee != "" {
		a.Fee, err = strconv.ParseFloat(chil.Fee, 64)
		if err != nil {
			return err
		}
	}
	if chil.MinBaseAmount != "" {
		a.MinBaseAmount, err = strconv.ParseFloat(chil.MinBaseAmount, 64)
		if err != nil {
			return err
		}
	}
	if chil.MinQuoteAmount != "" {
		a.MinQuoteAmount, err = strconv.ParseFloat(chil.MinQuoteAmount, 64)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information
func (a *Ticker) UnmarshalJSON(data []byte) error {
	type Alias Ticker
	child := &struct {
		*Alias
		BaseVolume  string `json:"base_volume"`
		QuoteVolume string `json:"quote_volume"`
		High24H     string `json:"high_24h"`
		Low24H      string `json:"low_24h"`
		Last        string `json:"last"`

		LowestAsk       string `json:"lowest_ask"`
		HighestBid      string `json:"highest_bid"`
		EtfLeverage     string `json:"etf_leverage"`
		EtfPreTimestamp int64  `json:"etf_pre_timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, child)
	if err != nil {
		return err
	}
	if child.BaseVolume != "" {
		if a.BaseVolume, err = strconv.ParseFloat(child.BaseVolume, 64); err != nil {
			return err
		}
	}
	if child.QuoteVolume != "" {
		if a.QuoteVolume, err = strconv.ParseFloat(child.QuoteVolume, 64); err != nil {
			return err
		}
	}
	if child.High24H != "" {
		if a.High24H, err = strconv.ParseFloat(child.High24H, 64); err != nil {
			return err
		}
	}
	if child.Low24H != "" {
		if a.Low24H, err = strconv.ParseFloat(child.Low24H, 64); err != nil {
			return err
		}
	}
	if child.LowestAsk != "" {
		if a.LowestAsk, err = strconv.ParseFloat(child.LowestAsk, 64); err != nil {
			return err
		}
	}
	if child.HighestBid != "" {
		if a.HighestBid, err = strconv.ParseFloat(child.HighestBid, 64); err != nil {
			return err
		}
	}
	if child.EtfLeverage != "" {
		if a.EtfLeverage, err = strconv.ParseFloat(child.EtfLeverage, 64); err != nil {
			return err
		}
	}
	if child.Last != "" {
		if a.Last, err = strconv.ParseFloat(child.Last, 64); err != nil {
			return err
		}
	}
	a.EtfPreTimestamp = time.Unix(child.EtfPreTimestamp, 0)
	return nil
}

// UnmarshalJSON to decerialize the timestamp information to golang time.Time instance
func (a *OrderbookData) UnmarshalJSON(data []byte) error {
	type Alias OrderbookData
	chil := &struct {
		*Alias
		Current float64 `json:"current"`
		Update  float64 `json:"update"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Current = time.Unix(int64(math.Round(chil.Current)), 0)
	a.Update = time.Unix(int64(math.Round(chil.Update)), 0)
	return nil
}

// UnmarshalJSON to decerialize timestamp information and create OrderbookItem instance from the list of asks and bids data.
func (a *OptionsTicker) UnmarshalJSON(data []byte) error {
	type Alias OptionsTicker
	chil := &struct {
		*Alias
		LastPrice string `json:"last_price"`
		MarkPrice string `json:"mark_price"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	if chil.LastPrice != "" {
		val, err := strconv.ParseFloat(chil.LastPrice, 64)
		if err != nil {
			return err
		}
		a.LastPrice = val
	}
	if chil.MarkPrice != "" {
		val, err := strconv.ParseFloat(chil.MarkPrice, 64)
		if err != nil {
			return err
		}
		a.MarkPrice = val
	}
	return nil
}

// UnmarshalJSON to decerialize timestamp information and create OrderbookItem instance from the list of asks and bids data.
func (a *Orderbook) UnmarshalJSON(data []byte) error {
	type Alias Orderbook
	type askorbid struct {
		Price string  `json:"p"`
		Size  float64 `json:"s"`
	}
	chil := &struct {
		*Alias
		Current float64    `json:"current"`
		Update  float64    `json:"update"`
		Bids    []askorbid `json:"asks"`
		Asks    []askorbid `json:"bids"`
	}{}
	if err := json.Unmarshal(data, &chil); err != nil {
		return err
	}
	a.Current = time.Unix(int64(chil.Current), 0)
	a.Update = time.Unix(int64(chil.Update), 0)
	asks := make([]OrderbookItem, len(chil.Asks))
	bids := make([]OrderbookItem, len(chil.Bids))
	for x := range chil.Asks {
		val, err := strconv.ParseFloat(chil.Asks[x].Price, 64)
		if err != nil {
			return err
		}
		asks[x] = OrderbookItem{
			Price:  val,
			Amount: chil.Asks[x].Size,
		}
	}
	for x := range chil.Bids {
		val, err := strconv.ParseFloat(chil.Bids[x].Price, 64)
		if err != nil {
			return err
		}
		bids[x] = OrderbookItem{
			Price:  val,
			Amount: chil.Bids[x].Size,
		}
	}
	a.Asks = asks
	a.Bids = bids
	return nil
}

// UnmarshalJSON decerializes the unix seconds, and milliseconds timestamp information to builtin time.Time.
func (a *Trade) UnmarshalJSON(data []byte) error {
	type Alias Trade
	chil := &struct {
		*Alias
		TradingTime  int64   `json:"create_time,string"`
		CreateTimeMs float64 `json:"create_time_ms,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.TradingTime = time.Unix(chil.TradingTime, 0)
	a.CreateTimeMs = time.UnixMilli(int64(math.Round(chil.CreateTimeMs)))
	return nil
}

// UnmarshalJSON to decerialize timestamp information to built-int golang time.Time instance.
func (a *FuturesContract) UnmarshalJSON(data []byte) error {
	type Alias FuturesContract
	chil := &struct {
		*Alias
		FundingNextApply int64 `json:"funding_next_apply"`
		ConfigChangeTime int64 `json:"config_change_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.FundingNextApply = time.Unix(chil.FundingNextApply, 0)
	a.ConfigChangeTime = time.Unix(chil.ConfigChangeTime, 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *TradingHistoryItem) UnmarshalJSON(data []byte) error {
	type Alias TradingHistoryItem
	chil := &struct {
		*Alias
		CreateTime float64 `json:"create_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(int64(chil.CreateTime), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *FuturesCandlestick) UnmarshalJSON(data []byte) error {
	type Alias FuturesCandlestick
	chil := &struct {
		*Alias
		Timestamp float64 `json:"t"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Timestamp = time.Unix(int64(chil.Timestamp), 10)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *FuturesFundingRate) UnmarshalJSON(data []byte) error {
	type Alias FuturesFundingRate
	chil := &struct {
		*Alias
		Timestamp float64 `json:"t"`
		Rate      string  `json:"r"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	if chil.Rate != "" {
		if a.Rate, err = strconv.ParseFloat(chil.Rate, 64); err != nil {
			return err
		}
	}
	a.Timestamp = time.Unix(int64(chil.Timestamp), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *InsuranceBalance) UnmarshalJSON(data []byte) error {
	type Alias InsuranceBalance
	chil := &struct {
		*Alias
		Timestamp float64 `json:"t"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Timestamp = time.Unix(int64(chil.Timestamp), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *ContractStat) UnmarshalJSON(data []byte) error {
	type Alias ContractStat
	chil := &struct {
		*Alias
		Time float64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Time = time.Unix(int64(chil.Time), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp.
func (a *LiquidationHistory) UnmarshalJSON(data []byte) error {
	type Alias LiquidationHistory
	chil := &struct {
		*Alias
		Time float64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Time = time.Unix(int64(chil.Time), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *DeliveryContract) UnmarshalJSON(data []byte) error {
	type Alias DeliveryContract
	chil := &struct {
		*Alias
		ExpireTime       float64 `json:"expire_time"`
		ConfigChangeTime float64 `json:"config_change_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.ExpireTime = time.Unix(int64(chil.ExpireTime), 0)
	a.ConfigChangeTime = time.Unix(int64(chil.ConfigChangeTime), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *OptionContract) UnmarshalJSON(data []byte) error {
	type Alias OptionContract
	chil := &struct {
		*Alias
		CreateTime     float64 `json:"create_time"`
		ExpirationTime float64 `json:"expiration_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(int64(chil.CreateTime), 0)
	a.ExpirationTime = time.Unix(int64(chil.ExpirationTime), 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *OptionSettlement) UnmarshalJSON(data []byte) error {
	type Alias OptionSettlement
	chil := &struct {
		*Alias
		Time int64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Time = time.Unix(chil.Time, 10)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *PositionCloseHistoryResponse) UnmarshalJSON(data []byte) error {
	type Alias PositionCloseHistoryResponse
	chil := &struct {
		*Alias
		Time int64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Time = time.Unix(chil.Time, 10)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *LiquidationHistoryItem) UnmarshalJSON(data []byte) error {
	type Alias LiquidationHistoryItem
	chil := &struct {
		*Alias
		Time int64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Time = time.Unix(chil.Time, 10)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *SpotOrder) UnmarshalJSON(data []byte) error {
	type Alias SpotOrder
	chil := &struct {
		*Alias
		CreateTime   int64  `json:"create_time,string"`
		UpdateTime   int64  `json:"update_time,string"`
		CreateTimeMs int64  `json:"create_time_ms"`
		UpdateTimeMs int64  `json:"update_time_ms"`
		Left         string `json:"left"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	if chil.Left != "" {
		if a.Left, err = strconv.ParseFloat(chil.Left, 64); err != nil {
			return err
		}
	}
	a.CreateTime = time.Unix(chil.CreateTime, 0)
	a.UpdateTime = time.Unix(chil.UpdateTime, 0)
	a.CreateTimeMs = time.UnixMilli(chil.CreateTimeMs)
	a.UpdateTimeMs = time.UnixMilli(chil.UpdateTimeMs)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information.
func (a *WsSpotOrder) UnmarshalJSON(data []byte) error {
	type Alias WsSpotOrder
	chil := &struct {
		*Alias
		CreateTime   int64   `json:"create_time,string"`
		UpdateTime   int64   `json:"update_time,string"`
		CreateTimeMs float64 `json:"create_time_ms,string"`
		UpdateTimeMs float64 `json:"update_time_ms,string"`
		Left         string  `json:"left"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	if chil.Left != "" {
		if a.Left, err = strconv.ParseFloat(chil.Left, 64); err != nil {
			return err
		}
	}
	a.CreateTime = time.Unix(chil.CreateTime, 0)
	a.UpdateTime = time.Unix(chil.UpdateTime, 0)
	a.CreateTimeMs = time.UnixMilli(int64(chil.CreateTimeMs))
	a.UpdateTimeMs = time.UnixMilli(int64(chil.UpdateTimeMs))
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information
func (a *MarginAccountBalanceChangeInfo) UnmarshalJSON(data []byte) error {
	type Alias MarginAccountBalanceChangeInfo
	chil := &struct {
		*Alias
		Time   int64 `json:"time,string"`
		TimeMs int64 `json:"time_ms"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Time = time.Unix(chil.Time, 0)
	a.TimeMs = time.UnixMilli(chil.TimeMs)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information
func (a *AccountBookItem) UnmarshalJSON(data []byte) error {
	type Alias AccountBookItem
	chil := &struct {
		*Alias
		Time int64 `json:"time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Time = time.Unix(chil.Time, 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information
func (a *Order) UnmarshalJSON(data []byte) error {
	type Alias Order
	chil := &struct {
		*Alias
		FinishTime int64 `json:"finish_time"`
		CreateTime int64 `json:"create_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.FinishTime = time.Unix(chil.FinishTime, 0)
	a.CreateTime = time.Unix(chil.CreateTime, 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information
func (a *PriceTriggeredOrder) UnmarshalJSON(data []byte) error {
	type Alias PriceTriggeredOrder
	chil := &struct {
		*Alias
		CreateTime int64 `json:"create_time"`
		FinishTime int64 `json:"finish_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(chil.CreateTime, 0)
	a.FinishTime = time.Unix(chil.FinishTime, 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information
func (a *SubAccount) UnmarshalJSON(data []byte) error {
	type Alias SubAccount
	chil := &struct {
		*Alias
		CreateTime int64 `json:"create_time"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Unix(chil.CreateTime, 0)
	return nil
}

// UnmarshalJSON decerializes json, and timestamp information
func (a *WsOptionsTrades) UnmarshalJSON(data []byte) error {
	type Alias WsOptionsTrades
	chil := &struct {
		*Alias
		CreateTime   int64 `json:"create_time"`
		CreateTimeMs int64 `json:"create_time_ms"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.UnixMilli(func() int64 {
		if chil.CreateTimeMs != 0 {
			return chil.CreateTimeMs
		}
		return chil.CreateTime
	}())
	return nil
}
