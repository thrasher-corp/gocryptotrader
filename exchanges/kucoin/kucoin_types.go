package kucoin

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var (
	validPeriods = []string{
		"1min", "3min", "5min", "15min", "30min", "1hour", "2hour", "4hour", "6hour", "8hour", "12hour", "1day", "1week",
	}
)

// UnmarshalTo acts as interface to exchange API response
type UnmarshalTo interface {
	GetError() error
}

// Error defines all error information for each request
type Error struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// GetError checks and returns an error if it is supplied.
func (e Error) GetError() error {
	code, err := strconv.ParseInt(e.Code, 10, 64)
	if err != nil {
		return err

	}

	switch code {
	case 200000:
		return nil
	default:
		return errors.New(e.Msg)
	}
}

// kucoinTimeMilliSec provides an internal conversion helper
type kucoinTimeMilliSec time.Time

// UnmarshalJSON is custom type json unmarshaller for kucoinTimeMilliSec
func (k *kucoinTimeMilliSec) UnmarshalJSON(data []byte) error {
	var timestamp int64
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	*k = kucoinTimeMilliSec(time.UnixMilli(timestamp))
	return nil
}

// Time returns a time.Time object
func (k kucoinTimeMilliSec) Time() time.Time {
	return time.Time(k)
}

// kucoinTimeMilliSecStr provides an internal conversion helper
type kucoinTimeMilliSecStr time.Time

// UnmarshalJSON is custom type json unmarshaller for kucoinTimeMilliSecStr
func (k *kucoinTimeMilliSecStr) UnmarshalJSON(data []byte) error {
	var timestamp string
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}

	t, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return err
	}
	*k = kucoinTimeMilliSecStr(time.UnixMilli(t))
	return nil
}

// Time returns a time.Time object
func (k kucoinTimeMilliSecStr) Time() time.Time {
	return time.Time(k)
}

// kucoinTimeNanoSec provides an internal conversion helper
type kucoinTimeNanoSec time.Time

// UnmarshalJSON is custom type json unmarshaller for kucoinTimeNanoSec
func (k *kucoinTimeNanoSec) UnmarshalJSON(data []byte) error {
	var timestamp int64
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	*k = kucoinTimeNanoSec(time.Unix(0, timestamp))
	return nil
}

// Time returns a time.Time object
func (k kucoinTimeNanoSec) Time() time.Time {
	return time.Time(k)
}

type SymbolInfo struct {
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name"`
	BaseCurrency    string  `json:"baseCurrency"`
	QuoteCurrency   string  `json:"quoteCurrency"`
	FeeCurrency     string  `json:"feeCurrency"`
	Market          string  `json:"market"`
	BaseMinSize     float64 `json:"baseMinSize,string"`
	QuoteMinSize    float64 `json:"quoteMinSize,string"`
	BaseMaxSize     float64 `json:"baseMaxSize,string"`
	QuoteMaxSize    float64 `json:"quoteMaxSize,string"`
	BaseIncrement   float64 `json:"baseIncrement,string"`
	QuoteIncrement  float64 `json:"quoteIncrement,string"`
	PriceIncrement  float64 `json:"priceIncrement,string"`
	PriceLimitRate  float64 `json:"priceLimitRate,string"`
	MinFunds        float64 `json:"minFunds,string"`
	IsMarginEnabled bool    `json:"isMarginEnabled"`
	EnableTrading   bool    `json:"enableTrading"`
}

type Ticker struct {
	Sequence    string  `json:"sequence"`
	BestAsk     float64 `json:"bestAsk,string"`
	Size        float64 `json:"size,string"`
	Price       float64 `json:"price,string"`
	BestBidSize float64 `json:"bestBidSize,string"`
	BestBid     float64 `json:"bestBid,string"`
	BestAskSize float64 `json:"bestAskSize,string"`
	Time        uint64  `json:"time"`
}

type tickerInfoBase struct {
	Symbol           string  `json:"symbol"`
	Buy              float64 `json:"buy,string"`
	Sell             float64 `json:"sell,string"`
	ChangeRate       float64 `json:"changeRate,string"`
	ChangePrice      float64 `json:"changePrice,string"`
	High             float64 `json:"high,string"`
	Low              float64 `json:"low,string"`
	Volume           float64 `json:"vol,string"`
	VolumeValue      float64 `json:"volValue,string"`
	Last             float64 `json:"last,string"`
	AveragePrice     float64 `json:"averagePrice,string"`
	TakerFeeRate     float64 `json:"takerFeeRate,string"`
	MakerFeeRate     float64 `json:"makerFeeRate,string"`
	TakerCoefficient float64 `json:"takerCoefficient,string"`
	MakerCoefficient float64 `json:"makerCoefficient,string"`
}

type TickerInfo struct {
	tickerInfoBase
	SymbolName string `json:"symbolName"`
}

type Stats24hrs struct {
	tickerInfoBase
	Time uint64 `json:"time"`
}

// Orderbook stores the orderbook data
type Orderbook struct {
	Bids []orderbook.Item
	Asks []orderbook.Item
	Time time.Time
}

type orderbookResponse struct {
	Data struct {
		Asks     [][2]string        `json:"asks"`
		Bids     [][2]string        `json:"bids"`
		Time     kucoinTimeMilliSec `json:"time"`
		Sequence string             `json:"sequence"`
	} `json:"result"`
	Error
}

type Trade struct {
	Sequence string            `json:"sequence"`
	Price    float64           `json:"price,string"`
	Size     float64           `json:"size,string"`
	Side     string            `json:"side"`
	Time     kucoinTimeNanoSec `json:"time"`
}

type Kline struct {
	StartTime time.Time
	Open      float64
	Close     float64
	High      float64
	Low       float64
	Volume    float64 // Transaction volume
	Amount    float64 // Transaction amount
}

type currencyBase struct {
	Currency        string `json:"currency"` // a unique currency code that will never change
	Name            string `json:"name"`     // will change after renaming
	Fullname        string `json:"fullName"`
	Precision       int64  `json:"precision"`
	Confirms        int64  `json:"confirms"`
	ContractAddress string `json:"contractAddress"`
	IsMarginEnabled bool   `json:"isMarginEnabled"`
	IsDebitEnabled  bool   `json:"isDebitEnabled"`
}

type Currency struct {
	currencyBase
	WithdrawalMinSize float64 `json:"withdrawalMinSize,string"`
	WithdrawalMinFee  float64 `json:"withdrawalMinFee,string"`
	IsWithdrawEnabled bool    `json:"isWithdrawEnabled"`
	IsDepositEnabled  bool    `json:"isDepositEnabled"`
}

type Chain struct {
	Name              string  `json:"chainName"`
	Confirms          int64   `json:"confirms"`
	ContractAddress   string  `json:"contractAddress"`
	WithdrawalMinSize float64 `json:"withdrawalMinSize,string"`
	WithdrawalMinFee  float64 `json:"withdrawalMinFee,string"`
	IsWithdrawEnabled bool    `json:"isWithdrawEnabled"`
	IsDepositEnabled  bool    `json:"isDepositEnabled"`
}

type CurrencyDetail struct {
	currencyBase
	Chains []Chain `json:"chains"`
}

type MarkPrice struct {
	Symbol      string             `json:"symbol"`
	Granularity int64              `json:"granularity"`
	TimePoint   kucoinTimeMilliSec `json:"timePoint"`
	Value       float64            `json:"value"`
}

type MarginConfiguration struct {
	CurrencyList     []string `json:"currencyList"`
	WarningDebtRatio float64  `json:"warningDebtRatio,string"`
	LiqDebtRatio     float64  `json:"liqDebtRatio,string"`
	MaxLeverage      float64  `json:"maxLeverage"`
}

type MarginAccount struct {
	CurrencyList  float64 `json:"availableBalance,string"`
	Currency      string  `json:"currency"`
	HoldBalance   float64 `json:"holdBalance,string"`
	Liability     float64 `json:"liability,string"`
	MaxBorrowSize float64 `json:"maxBorrowSize,string"`
	TotalBalance  float64 `json:"totalBalance,string"`
}

type MarginAccounts struct {
	Accounts  []MarginAccount `json:"accounts"`
	DebtRatio float64         `json:"debtRatio,string"`
}

type MarginRiskLimit struct {
	Currency        string  `json:"currency"`
	BorrowMaxAmount float64 `json:"borrowMaxAmount,string"`
	BuyMaxAmount    float64 `json:"buyMaxAmount,string"`
	Precision       int64   `json:"precision"`
}

type PostBorrowOrderResp struct {
	OrderID  string `json:"orderId"`
	Currency string `json:"currency"`
}

type BorrowOrder struct {
	OrderID   string  `json:"orderId"`
	Currency  string  `json:"currency"`
	Size      float64 `json:"size,string"`
	Filled    float64 `json:"filled"`
	MatchList []struct {
		Currency     string                `json:"currency"`
		DailyIntRate float64               `json:"dailyIntRate,string"`
		Size         float64               `json:"size,string"`
		Term         int64                 `json:"term"`
		Timestamp    kucoinTimeMilliSecStr `json:"timestamp"`
		TradeID      string                `json:"tradeId"`
	} `json:"matchList"`
	Status string `json:"status"`
}

type baseRecord struct {
	TradeID      string  `json:"tradeId"`
	Currency     string  `json:"currency"`
	DailyIntRate float64 `json:"dailyIntRate,string"`
	Principal    float64 `json:"principal,string"`
	RepaidSize   float64 `json:"repaidSize,string"`
	Term         int64   `json:"term"`
}

type OutstandingRecord struct {
	baseRecord
	AccruedInterest float64               `json:"accruedInterest,string"`
	Liability       float64               `json:"liability,string"`
	MaturityTime    kucoinTimeMilliSecStr `json:"maturityTime"`
	CreatedAt       kucoinTimeMilliSecStr `json:"createdAt"`
}

type RepaidRecord struct {
	baseRecord
	Interest  float64               `json:"interest,string"`
	RepayTime kucoinTimeMilliSecStr `json:"repayTime"`
}

type LendOrder struct {
	OrderID      string                `json:"orderId"`
	Currency     string                `json:"currency"`
	Size         float64               `json:"size,string"`
	FilledSize   float64               `json:"filledSize,string"`
	DailyIntRate float64               `json:"dailyIntRate,string"`
	Term         int64                 `json:"term"`
	CreatedAt    kucoinTimeMilliSecStr `json:"createdAt"`
}

type LendOrderHistory struct {
	LendOrder
	Status string `json:"status"`
}

type UnsettleLendOrder struct {
	TradeID         string                `json:"tradeId"`
	Currency        string                `json:"currency"`
	Size            float64               `json:"size,string"`
	AccruedInterest float64               `json:"accruedInterest,string"`
	Repaid          float64               `json:"repaid,string"`
	DailyIntRate    float64               `json:"dailyIntRate,string"`
	Term            int64                 `json:"term"`
	MaturityTime    kucoinTimeMilliSecStr `json:"maturityTime"`
}

type SettleLendOrder struct {
	TradeID      string             `json:"tradeId"`
	Currency     string             `json:"currency"`
	Size         float64            `json:"size,string"`
	Interest     float64            `json:"interest,string"`
	Repaid       float64            `json:"repaid,string"`
	DailyIntRate float64            `json:"dailyIntRate,string"`
	Term         int64              `json:"term"`
	SettledAt    kucoinTimeMilliSec `json:"settledAt"`
	Note         string             `json:"note"`
}

type LendRecord struct {
	Currency        string  `json:"currency"`
	Outstanding     float64 `json:"outstanding"`
	FilledSize      float64 `json:"filledSize,string"`
	AccruedInterest float64 `json:"accruedInterest,string"`
	RealizedProfit  float64 `json:"realizedProfit,string"`
	IsAutoLend      bool    `json:"isAutoLend"`
}

type LendMarketData struct {
	DailyIntRate float64 `json:"dailyIntRate,string"`
	Term         int64   `json:"term"`
	Size         float64 `json:"size,string"`
}

type MarginTradeData struct {
	TradeID      string            `json:"tradeId"`
	Currency     string            `json:"currency"`
	Size         float64           `json:"size,string"`
	DailyIntRate float64           `json:"dailyIntRate,string"`
	Term         int64             `json:"term"`
	Timestamp    kucoinTimeNanoSec `json:"timestamp"`
}
