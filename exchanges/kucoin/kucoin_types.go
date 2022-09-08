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
	case 200000, 200:
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

// SymbolInfo stores symbol information
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

// Ticker stores ticker data
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

// TickerInfo stores ticker information
type TickerInfo struct {
	tickerInfoBase
	SymbolName string `json:"symbolName"`
}

// Stats24hrs stores 24 hrs statistics
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

// Trade stores trade data
type Trade struct {
	Sequence string            `json:"sequence"`
	Price    float64           `json:"price,string"`
	Size     float64           `json:"size,string"`
	Side     string            `json:"side"`
	Time     kucoinTimeNanoSec `json:"time"`
}

// Kline stores kline data
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

// Currency stores currency data
type Currency struct {
	currencyBase
	WithdrawalMinSize float64 `json:"withdrawalMinSize,string"`
	WithdrawalMinFee  float64 `json:"withdrawalMinFee,string"`
	IsWithdrawEnabled bool    `json:"isWithdrawEnabled"`
	IsDepositEnabled  bool    `json:"isDepositEnabled"`
}

// Chain stores blockchain data
type Chain struct {
	Name              string  `json:"chainName"`
	Confirms          int64   `json:"confirms"`
	ContractAddress   string  `json:"contractAddress"`
	WithdrawalMinSize float64 `json:"withdrawalMinSize,string"`
	WithdrawalMinFee  float64 `json:"withdrawalMinFee,string"`
	IsWithdrawEnabled bool    `json:"isWithdrawEnabled"`
	IsDepositEnabled  bool    `json:"isDepositEnabled"`
}

// CurrencyDetail stores currency details
type CurrencyDetail struct {
	currencyBase
	Chains []Chain `json:"chains"`
}

// MarkPrice stores mark price data
type MarkPrice struct {
	Symbol      string             `json:"symbol"`
	Granularity int64              `json:"granularity"`
	TimePoint   kucoinTimeMilliSec `json:"timePoint"`
	Value       float64            `json:"value"`
}

// MarginConfiguration stores margin configuration
type MarginConfiguration struct {
	CurrencyList     []string `json:"currencyList"`
	WarningDebtRatio float64  `json:"warningDebtRatio,string"`
	LiqDebtRatio     float64  `json:"liqDebtRatio,string"`
	MaxLeverage      float64  `json:"maxLeverage"`
}

// MarginAccount stores margin account data
type MarginAccount struct {
	CurrencyList  float64 `json:"availableBalance,string"`
	Currency      string  `json:"currency"`
	HoldBalance   float64 `json:"holdBalance,string"`
	Liability     float64 `json:"liability,string"`
	MaxBorrowSize float64 `json:"maxBorrowSize,string"`
	TotalBalance  float64 `json:"totalBalance,string"`
}

// MarginAccounts stores margin accounts data
type MarginAccounts struct {
	Accounts  []MarginAccount `json:"accounts"`
	DebtRatio float64         `json:"debtRatio,string"`
}

// MarginRiskLimit stores margin risk limit
type MarginRiskLimit struct {
	Currency        string  `json:"currency"`
	BorrowMaxAmount float64 `json:"borrowMaxAmount,string"`
	BuyMaxAmount    float64 `json:"buyMaxAmount,string"`
	Precision       int64   `json:"precision"`
}

// PostBorrowOrderResp stores borrow order resposne
type PostBorrowOrderResp struct {
	OrderID  string `json:"orderId"`
	Currency string `json:"currency"`
}

// BorrowOrder stores borrow order
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

// OutstandingRecord stores outstanding record
type OutstandingRecord struct {
	baseRecord
	AccruedInterest float64               `json:"accruedInterest,string"`
	Liability       float64               `json:"liability,string"`
	MaturityTime    kucoinTimeMilliSecStr `json:"maturityTime"`
	CreatedAt       kucoinTimeMilliSecStr `json:"createdAt"`
}

// RepaidRecord stores repaid record
type RepaidRecord struct {
	baseRecord
	Interest  float64               `json:"interest,string"`
	RepayTime kucoinTimeMilliSecStr `json:"repayTime"`
}

// LendOrder stores lend order
type LendOrder struct {
	OrderID      string                `json:"orderId"`
	Currency     string                `json:"currency"`
	Size         float64               `json:"size,string"`
	FilledSize   float64               `json:"filledSize,string"`
	DailyIntRate float64               `json:"dailyIntRate,string"`
	Term         int64                 `json:"term"`
	CreatedAt    kucoinTimeMilliSecStr `json:"createdAt"`
}

// LendOrderHistory stores lend order history
type LendOrderHistory struct {
	LendOrder
	Status string `json:"status"`
}

// UnsettleLendOrder stores unsettle lend order
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

// SettleLendOrder stores  settled lend order
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

// LendRecord stores lend record
type LendRecord struct {
	Currency        string  `json:"currency"`
	Outstanding     float64 `json:"outstanding,string"`
	FilledSize      float64 `json:"filledSize,string"`
	AccruedInterest float64 `json:"accruedInterest,string"`
	RealizedProfit  float64 `json:"realizedProfit,string"`
	IsAutoLend      bool    `json:"isAutoLend"`
}

// LendMarketData stores lend market data
type LendMarketData struct {
	DailyIntRate float64 `json:"dailyIntRate,string"`
	Term         int64   `json:"term"`
	Size         float64 `json:"size,string"`
}

// MarginTradeData stores margin trade data
type MarginTradeData struct {
	TradeID      string            `json:"tradeId"`
	Currency     string            `json:"currency"`
	Size         float64           `json:"size,string"`
	DailyIntRate float64           `json:"dailyIntRate,string"`
	Term         int64             `json:"term"`
	Timestamp    kucoinTimeNanoSec `json:"timestamp"`
}

type IsolatedMarginPairConfig struct {
	Symbol                string  `json:"symbol"`
	SymbolName            string  `json:"symbolName"`
	BaseCurrency          string  `json:"baseCurrency"`
	QuoteCurrency         string  `json:"quoteCurrency"`
	MaxLeverage           int64   `json:"maxLeverage"`
	LiquidationDebtRatio  float64 `json:"flDebtRatio,string"`
	TradeEnable           bool    `json:"tradeEnable"`
	AutoRenewMaxDebtRatio float64 `json:"autoRenewMaxDebtRatio,string"`
	BaseBorrowEnable      bool    `json:"baseBorrowEnable"`
	QuoteBorrowEnable     bool    `json:"quoteBorrowEnable"`
	BaseTransferInEnable  bool    `json:"baseTransferInEnable"`
	QuoteTransferInEnable bool    `json:"quoteTransferInEnable"`
}

type baseAsset struct {
	Currency         string  `json:"currency"`
	TotalBalance     float64 `json:"totalBalance,string"`
	HoldBalance      float64 `json:"holdBalance,string"`
	AvailableBalance float64 `json:"availableBalance,string"`
	Liability        float64 `json:"liability,string"`
	Interest         float64 `json:"interest,string"`
	BorrowableAmount float64 `json:"borrowableAmount,string"`
}

type AssetInfo struct {
	Symbol     string    `json:"symbol"`
	Status     string    `json:"status"`
	DebtRatio  float64   `json:"debtRatio,string"`
	BaseAsset  baseAsset `json:"baseAsset"`
	QuoteAsset baseAsset `json:"quoteAsset"`
}

type IsolatedMarginAccountInfo struct {
	TotalConversionBalance     float64     `json:"totalConversionBalance,string"`
	LiabilityConversionBalance float64     `json:"liabilityConversionBalance,string"`
	Assets                     []AssetInfo `json:"assets"`
}

type baseRepaymentRecord struct {
	LoanID            string  `json:"loanId"`
	Symbol            string  `json:"symbol"`
	Currency          string  `json:"currency"`
	PrincipalTotal    float64 `json:"principalTotal,string"`
	InterestBalance   float64 `json:"interestBalance,string"`
	CreatedAt         int64   `json:"createdAt"`
	Period            int64   `json:"period"`
	RepaidSize        float64 `json:"repaidSize,string"`
	DailyInterestRate float64 `json:"dailyInterestRate,string"`
}

type OutstandingRepaymentRecord struct {
	baseRepaymentRecord
	LiabilityBalance float64 `json:"liabilityBalance,string"`
	MaturityTime     int64   `json:"maturityTime"`
}

type CompletedRepaymentRecord struct {
	baseRepaymentRecord
	RepayFinishAt int64 `json:"repayFinishAt"`
}

type PostMarginOrderResp struct {
	OrderID     string  `json:"orderId"`
	BorrowSize  float64 `json:"borrowSize"`
	LoanApplyID string  `json:"loanApplyId"`
}

type OrderRequest struct {
	ClientOID   string `json:"clientOid"`
	Symbol      string `json:"symbol"`
	Side        string `json:"side"`
	Type        string `json:"type,omitempty"`      // optional
	Remark      string `json:"remark,omitempty"`    // optional
	Stop        string `json:"stop,omitempty"`      // optional
	StopPrice   string `json:"stopPrice,omitempty"` // optional
	STP         string `json:"stp,omitempty"`       // optional
	Price       string `json:"price,omitempty"`
	Size        string `json:"size,omitempty"`
	TimeInForce string `json:"timeInForce,omitempty"` // optional
	CancelAfter int64  `json:"cancelAfter,omitempty"` // optional
	PostOnly    bool   `json:"postOnly,omitempty"`    // optional
	Hidden      bool   `json:"hidden,omitempty"`      // optional
	Iceberg     bool   `json:"iceberg,omitempty"`     // optional
	VisibleSize string `json:"visibleSize,omitempty"` // optional
}

type PostBulkOrderResp struct {
	OrderRequest
	Channel string `json:"channel"`
	ID      string `json:"id"`
	Status  string `json:"status"`
	FailMsg string `json:"failMsg"`
}

type OrderDetail struct {
	OrderRequest
	Channel       string             `json:"channel"`
	ID            string             `json:"id"`
	OpType        string             `json:"opType"` // operation type: DEAL
	Funds         string             `json:"funds"`
	DealFunds     string             `json:"dealFunds"`
	DealSize      string             `json:"dealSize"`
	Fee           string             `json:"fee"`
	FeeCurrency   string             `json:"feeCurrency"`
	StopTriggered bool               `json:"stopTriggered"`
	Tags          string             `json:"tags"`
	IsActive      bool               `json:"isActive"`
	CancelExist   bool               `json:"cancelExist"`
	CreatedAt     kucoinTimeMilliSec `json:"createdAt"`
	TradeType     string             `json:"tradeType"`
}

type Fill struct {
	Symbol         string             `json:"symbol"`
	TradeID        string             `json:"tradeId"`
	OrderID        string             `json:"orderId"`
	CounterOrderId string             `json:"counterOrderId"`
	Side           string             `json:"side"`
	Liquidity      string             `json:"liquidity"`
	ForceTaker     bool               `json:"forceTaker"`
	Price          float64            `json:"price,string"`
	Size           float64            `json:"size,string"`
	Funds          float64            `json:"funds,string"`
	Fee            float64            `json:"fee,string"`
	FeeRate        float64            `json:"feeRate,string"`
	FeeCurrency    string             `json:"feeCurrency"`
	Stop           string             `json:"stop"`
	OrderType      string             `json:"type"`
	CreatedAt      kucoinTimeMilliSec `json:"createdAt"`
	TradeType      string             `json:"tradeType"`
}

type StopOrder struct {
	OrderRequest
	ID              string             `json:"id"`
	UserID          string             `json:"userId"`
	Status          string             `json:"status"`
	Funds           float64            `json:"funds,string"`
	Channel         string             `json:"channel"`
	Tags            string             `json:"tags"`
	DomainId        string             `json:"domainId"`
	TradeSource     string             `json:"tradeSource"`
	TradeType       string             `json:"tradeType"`
	FeeCurrency     string             `json:"feeCurrency"`
	TakerFeeRate    string             `json:"takerFeeRate"`
	MakerFeeRate    string             `json:"makerFeeRate"`
	CreatedAt       kucoinTimeMilliSec `json:"createdAt"`
	OrderTime       kucoinTimeNanoSec  `json:"orderTime"`
	StopTriggerTime kucoinTimeMilliSec `json:"stopTriggerTime"`
}

type baseAccount struct {
	Currency  string  `json:"currency"`
	Balance   float64 `json:"balance,string"`
	Available float64 `json:"available,string"`
	Holds     float64 `json:"holds,string"`
}

type AccountInfo struct {
	baseAccount
	ID   string `json:"id"`
	Type string `json:"type"`
}

type LedgerInfo struct {
	ID          string             `json:"id"`
	Currency    string             `json:"currency"`
	Amount      float64            `json:"amount,string"`
	Fee         float64            `json:"fee,string"`
	Balance     float64            `json:"balance,string"`
	AccountType string             `json:"accountType"`
	BizType     string             `json:"bizType"`
	Direction   string             `json:"direction"`
	CreatedAt   kucoinTimeMilliSec `json:"createdAt"`
	Context     string             `json:"context"`
}

type MainAccountInfo struct {
	baseAccount
	BaseCurrency      string  `json:"baseCurrency"`
	BaseCurrencyPrice float64 `json:"baseCurrencyPrice,string"`
	BaseAmount        float64 `json:"baseAmount,string"`
}

type SubAccountInfo struct {
	SubUserID      string            `json:"subUserId"`
	SubName        string            `json:"subName"`
	MainAccounts   []MainAccountInfo `json:"mainAccounts"`
	TradeAccounts  []MainAccountInfo `json:"tradeAccounts"`
	MarginAccounts []MainAccountInfo `json:"marginAccounts"`
}

type TransferableBalanceInfo struct {
	baseAccount
	Transferable float64 `json:"transferable,string"`
}

type DepositAddress struct {
	Address         string `json:"address"`
	Memo            string `json:"memo"`
	Chain           string `json:"chain"`
	ContractAddress string `json:"contractAddress"`
}

type baseDeposit struct {
	Currency   string  `json:"currency"`
	Amount     float64 `json:"amount"`
	WalletTxID string  `json:"walletTxId"`
	IsInner    bool    `json:"isInner"`
	Status     string  `json:"status"`
}

type Deposit struct {
	baseDeposit
	Address   string  `json:"address"`
	Memo      string  `json:"memo"`
	Fee       float64 `json:"fee"`
	Remark    string  `json:"remark"`
	CreatedAt kucoinTimeMilliSec
	UpdatedAt kucoinTimeMilliSec
}

type HistoricalDepositWithdrawal struct {
	baseDeposit
	CreatedAt kucoinTimeMilliSec `json:"createAt"`
}

type Withdrawal struct {
	Deposit
	ID string `json:"id"`
}

type WithdrawalQuota struct {
	Currency            string  `json:"currency"`
	LimitBTCAmount      float64 `json:"limitBTCAmount,string"`
	UsedBTCAmount       float64 `json:"usedBTCAmount,string"`
	RemainAmount        float64 `json:"remainAmount,string"`
	AvailableAmount     float64 `json:"availableAmount,string"`
	WithdrawMinFee      float64 `json:"withdrawMinFee,string"`
	InnerWithdrawMinFee float64 `json:"innerWithdrawMinFee,string"`
	WithdrawMinSize     float64 `json:"withdrawMinSize,string"`
	IsWithdrawEnabled   bool    `json:"isWithdrawEnabled"`
	Precision           int64   `json:"precision"`
	Chain               string  `json:"chain"`
}

type Fees struct {
	Symbol       string  `json:"symbol"`
	TakerFeeRate float64 `json:"takerFeeRate,string"`
	MakerFeeRate float64 `json:"makerFeeRate,string"`
}
