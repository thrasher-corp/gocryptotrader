package kucoin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var (
	validPeriods = []string{
		"1min", "3min", "5min", "15min", "30min", "1hour", "2hour", "4hour", "6hour", "8hour", "12hour", "1day", "1week",
	}

	errInvalidResponseReceiver   = errors.New("invalid response receiver")
	errInvalidPrice              = errors.New("invalid price")
	errInvalidSize               = errors.New("invalid size")
	errMalformedData             = errors.New("malformed data")
	errNoDepositAddress          = errors.New("no deposit address found")
	errMultipleDepositAddress    = errors.New("multiple deposit addresses")
	errInvalidResultInterface    = errors.New("result interface has to be pointer")
	errInvalidSubAccountName     = errors.New("invalid sub-account name")
	errInvalidPassPhraseInstance = errors.New("invalid passphrase string")
	errNoValidResponseFromServer = errors.New("no valud response from server")
	errMissingOrderbookSequence  = errors.New("missing orderbook sequence")
)

var offlineTradeFee = map[currency.Code]float64{
	currency.BTC:   0.0005,
	currency.ETH:   0.005,
	currency.BNB:   0.01,
	currency.USDT:  25,
	currency.SOL:   0.01,
	currency.ADA:   1.000,
	currency.XRP:   0.5,
	currency.DOT:   0.1,
	currency.USDC:  20,
	currency.DOGE:  20,
	currency.AVAX:  0.01,
	currency.SHIB:  600000,
	currency.LUNA:  0.15,
	currency.LTC:   0.001,
	currency.CRO:   50,
	currency.UNI:   1.2,
	currency.BUSD:  1,
	currency.LINK:  1,
	currency.MATIC: 10,
	currency.ALGO:  0.1,
	currency.BCH:   0.01,
	currency.VET:   30,
	currency.XLM:   0.02,
	currency.ICP:   0.0005,
	currency.AXS:   0.6,
	currency.EGLD:  0.005,
	currency.TRX:   1.5,
	currency.FTT:   0.35,
	currency.UST:   4,
	currency.MANA:  10,
	currency.THETA: 0.2,
	currency.ETC:   0.01,
	currency.FIL:   0.01,
	currency.ATOM:  0.01,
	currency.DAI:   6,
	currency.APE:   1.5,
	currency.HBAR:  3,
	currency.NEAR:  0.01,
	currency.FTM:   30,
	currency.XTZ:   0.2,
	currency.XCN:   100,
	currency.HNT:   0.05,
	currency.XMR:   0.001,
	currency.GRT:   60,
	currency.EOS:   0.2,
	currency.FLOW:  0.05,
	currency.KLAY:  0.5,
	currency.SAND:  10,
	currency.CAKE:  0.05,
	currency.AAVE:  0.2,
	currency.LRC:   20,
	currency.XEC:   5000,
	currency.KSM:   0.01,
	currency.ONE:   100,
	currency.MKR:   0.0075,
	currency.KDA:   0.5,
	currency.BSV:   0.01,
	currency.BTT:   300000,
	currency.NEO:   0,
	currency.RUNE:  0.05,
	currency.USDD:  1,
	currency.QNT:   0.04,
	currency.CHZ:   4,
	currency.STX:   1.5,
	currency.ZEC:   0.005,
	currency.WAVES: 0.002,
	currency.AR:    0.02,
	currency.AMP:   2100,
	currency.DASH:  0.002,
	currency.KCS:   0.75,
	currency.CELO:  0.1,
	currency.COMP:  0.15,
	currency.TFUEL: 5,
	currency.CRV:   8.5,
	currency.XEM:   4,
	currency.BAT:   25,
	currency.HT:    0.1,
	currency.IMX:   10,
	currency.QTUM:  0.01,
	currency.DCR:   0.01,
	currency.ICX:   1,
	currency.OMG:   3,
	currency.TUSD:  15,
	currency.RVN:   2,
	currency.ROSE:  0.1,
	currency.ZEN:   0.002,
	currency.ZIL:   10,
	currency.SUSHI: 5,
	currency.AUDIO: 24,
	currency.LPT:   0.85,
	currency.XDC:   2,
	currency.SCRT:  0.25,
	currency.UMA:   3.5,
	currency.VLX:   10,
	currency.ANKR:  275,
	currency.GMT:   0.5,
	currency.PERP:  7.5,
	currency.TEL:   5500,
	currency.SNX:   6,
}

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
		return fmt.Errorf("code: %s message: %s", e.Code, e.Msg)
	}
}

// kucoinTimeMilliSec provides an internal conversion helper
type kucoinTimeMilliSec int64

// Time returns a time.Time object
func (k kucoinTimeMilliSec) Time() time.Time {
	return time.UnixMilli(int64(k))
}

// kucoinTimeMilliSecStr provides an internal conversion helper
type kucoinTimeMilliSecStr time.Time

// Time returns a time.Time object
func (k kucoinTimeMilliSecStr) Time() time.Time {
	return time.Time(k)
}

// kucoinTimeNanoSec provides an internal conversion helper
type kucoinTimeNanoSec time.Time

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
	Sequence int64
	Bids     []orderbook.Item
	Asks     []orderbook.Item
	Time     time.Time
}

type orderbookResponse struct {
	Asks     [][2]string        `json:"asks"`
	Bids     [][2]string        `json:"bids"`
	Time     kucoinTimeMilliSec `json:"time"`
	Sequence string             `json:"sequence"`
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

// PostBorrowOrderResp stores borrow order response
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

// OutstandingRecordResponse represents outstanding record detail.
type OutstandingRecordResponse struct {
	CurrentPage int64               `json:"currentPage"`
	PageSize    int64               `json:"pageSize"`
	TotalNumber int64               `json:"totalNum"`
	TotalPage   int64               `json:"totalPage"`
	Items       []OutstandingRecord `json:"items"` // lists
}

// OutstandingRecord stores outstanding record
type OutstandingRecord struct {
	baseRecord
	AccruedInterest float64               `json:"accruedInterest,string"`
	Liability       float64               `json:"liability,string"`
	MaturityTime    kucoinTimeMilliSecStr `json:"maturityTime"`
	CreatedAt       kucoinTimeMilliSecStr `json:"createdAt"`
}

// RepaidRecordsResponse stores list of repaid record details.
type RepaidRecordsResponse struct {
	CurrentPage int64          `json:"currentPage"`
	PageSize    int64          `json:"pageSize"`
	TotalNumber int64          `json:"totalNum"`
	TotalPage   int64          `json:"totalPage"`
	Items       []RepaidRecord `json:"items"`
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

// IsolatedMarginPairConfig current isolated margin trading pair configuration
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

// AssetInfo holds asset information for an instrument.
type AssetInfo struct {
	Symbol     string    `json:"symbol"`
	Status     string    `json:"status"`
	DebtRatio  float64   `json:"debtRatio,string"`
	BaseAsset  baseAsset `json:"baseAsset"`
	QuoteAsset baseAsset `json:"quoteAsset"`
}

// IsolatedMarginAccountInfo holds isolated margin accounts of the current user
type IsolatedMarginAccountInfo struct {
	TotalConversionBalance     float64     `json:"totalConversionBalance,string"`
	LiabilityConversionBalance float64     `json:"liabilityConversionBalance,string"`
	Assets                     []AssetInfo `json:"assets"`
}

type baseRepaymentRecord struct {
	LoanID            string             `json:"loanId"`
	Symbol            string             `json:"symbol"`
	Currency          string             `json:"currency"`
	PrincipalTotal    float64            `json:"principalTotal,string"`
	InterestBalance   float64            `json:"interestBalance,string"`
	CreatedAt         kucoinTimeMilliSec `json:"createdAt"`
	Period            int64              `json:"period"`
	RepaidSize        float64            `json:"repaidSize,string"`
	DailyInterestRate float64            `json:"dailyInterestRate,string"`
}

// OutstandingRepaymentRecordsResponse represents an outstanding repayment records of isolated margin positions list
type OutstandingRepaymentRecordsResponse struct {
	CurrentPage int64                        `json:"currentPage"`
	PageSize    int64                        `json:"pageSize"`
	TotalNum    int64                        `json:"totalNum"`
	TotalPage   int64                        `json:"totalPage"`
	Items       []OutstandingRepaymentRecord `json:"items"`
}

// OutstandingRepaymentRecord represents an outstanding repayment records of isolated margin positions
type OutstandingRepaymentRecord struct {
	baseRepaymentRecord
	LiabilityBalance float64 `json:"liabilityBalance,string"`
	MaturityTime     int64   `json:"maturityTime"`
}

// ServiceStatus represents a service status message.
type ServiceStatus struct {
	Status  string `json:"status"`
	Message string `json:"msg"`
}

// CompletedRepaymentRecordsResponse represents a completed payment records list.
type CompletedRepaymentRecordsResponse struct {
	CurrentPage int64                      `json:"currentPage"`
	PageSize    int64                      `json:"pageSize"`
	TotalNum    int64                      `json:"totalNum"`
	TotalPage   int64                      `json:"totalPage"`
	Items       []CompletedRepaymentRecord `json:"items"`
}

// CompletedRepaymentRecord represents repayment records of isolated margin positions
type CompletedRepaymentRecord struct {
	baseRepaymentRecord
	RepayFinishAt kucoinTimeMilliSec `json:"repayFinishAt"`
}

// PostMarginOrderResp represents response data for placing margin orders
type PostMarginOrderResp struct {
	OrderID     string  `json:"orderId"`
	BorrowSize  float64 `json:"borrowSize"`
	LoanApplyID string  `json:"loanApplyId"`
}

// OrderRequest represents place order request parameters
type OrderRequest struct {
	ClientOID   string  `json:"clientOid"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	Type        string  `json:"type,omitempty"`      // optional
	Remark      string  `json:"remark,omitempty"`    // optional
	Stop        string  `json:"stop,omitempty"`      // optional
	StopPrice   string  `json:"stopPrice,omitempty"` // optional
	STP         string  `json:"stp,omitempty"`       // optional
	Price       float64 `json:"price,string,omitempty"`
	Size        float64 `json:"size,string,omitempty"`
	TimeInForce string  `json:"timeInForce,omitempty"` // optional
	CancelAfter int64   `json:"cancelAfter,omitempty"` // optional
	PostOnly    bool    `json:"postOnly,omitempty"`    // optional
	Hidden      bool    `json:"hidden,omitempty"`      // optional
	Iceberg     bool    `json:"iceberg,omitempty"`     // optional
	VisibleSize string  `json:"visibleSize,omitempty"` // optional
}

// PostBulkOrderResp response data for submitting a bulk order
type PostBulkOrderResp struct {
	OrderRequest
	Channel string `json:"channel"`
	ID      string `json:"id"`
	Status  string `json:"status"`
	FailMsg string `json:"failMsg"`
}

// OrdersListResponse represents an order list response.
type OrdersListResponse struct {
	CurrentPage int64         `json:"currentPage"`
	PageSize    int64         `json:"pageSize"`
	TotalNum    int64         `json:"totalNum"`
	TotalPage   int64         `json:"totalPage"`
	Items       []OrderDetail `json:"items"`
}

// OrderDetail represents order detail
type OrderDetail struct {
	OrderRequest
	Channel       string             `json:"channel"`
	ID            string             `json:"id"`
	OpType        string             `json:"opType"` // operation type: DEAL
	Funds         string             `json:"funds"`
	DealFunds     string             `json:"dealFunds"`
	DealSize      float64            `json:"dealSize,string"`
	Fee           float64            `json:"fee,string"`
	FeeCurrency   string             `json:"feeCurrency"`
	StopTriggered bool               `json:"stopTriggered"`
	Tags          string             `json:"tags"`
	IsActive      bool               `json:"isActive"`
	CancelExist   bool               `json:"cancelExist"`
	CreatedAt     kucoinTimeMilliSec `json:"createdAt"`
	TradeType     string             `json:"tradeType"`
}

// ListFills represents fills response list detail.
type ListFills struct {
	CurrentPage int64  `json:"currentPage"`
	PageSize    int64  `json:"pageSize"`
	TotalNumber int64  `json:"totalNum"`
	TotalPage   int64  `json:"totalPage"`
	Items       []Fill `json:"items"`
}

// Fill represents order fills for margin and spot orders.
type Fill struct {
	Symbol         string             `json:"symbol"`
	TradeID        string             `json:"tradeId"`
	OrderID        string             `json:"orderId"`
	CounterOrderID string             `json:"counterOrderId"`
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

// StopOrderListResponse represents a list of spot orders details.
type StopOrderListResponse struct {
	CurrentPage int64       `json:"currentPage"`
	PageSize    int64       `json:"pageSize"`
	TotalNumber int64       `json:"totalNum"`
	TotalPage   int64       `json:"totalPage"`
	Items       []StopOrder `json:"items"`
}

// StopOrder holds a stop order detail
type StopOrder struct {
	OrderRequest
	ID              string             `json:"id"`
	UserID          string             `json:"userId"`
	Status          string             `json:"status"`
	Funds           float64            `json:"funds,string"`
	Channel         string             `json:"channel"`
	Tags            string             `json:"tags"`
	DomainID        string             `json:"domainId"`
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

// AccountInfo represents account information
type AccountInfo struct {
	baseAccount
	ID   string `json:"id"`
	Type string `json:"type"`
}

// LedgerInfo represents account ledger information.
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

// MainAccountInfo represents main account detailed information.
type MainAccountInfo struct {
	baseAccount
	BaseCurrency      string  `json:"baseCurrency"`
	BaseCurrencyPrice float64 `json:"baseCurrencyPrice,string"`
	BaseAmount        float64 `json:"baseAmount,string"`
}

// AccountSummaryInformation represents account summary information detail.
type AccountSummaryInformation struct {
	Level             int64 `json:"level"`
	SubQuantity       int64 `json:"subQuantity"`
	SubQuantityByType struct {
		GeneralSubQuantity int64 `json:"generalSubQuantity"`
		MarginSubQuantity  int64 `json:"marginSubQuantity"`
		FuturesSubQuantity int64 `json:"futuresSubQuantity"`
	} `json:"subQuantityByType"`
	MaxSubQuantity       int64 `json:"maxSubQuantity"`
	MaxSubQuantityByType struct {
		MaxDefaultSubQuantity int64 `json:"maxDefaultSubQuantity"`
		MaxGeneralSubQuantity int64 `json:"maxGeneralSubQuantity"`
		MaxMarginSubQuantity  int64 `json:"maxMarginSubQuantity"`
		MaxFuturesSubQuantity int64 `json:"maxFuturesSubQuantity"`
	} `json:"maxSubQuantityByType"`
}

// SubAccountsResponse represents a sub-accounts items response instance.
type SubAccountsResponse struct {
	CurrentPage int64            `json:"currentPage"`
	PageSize    int64            `json:"pageSize"`
	TotalNumber int64            `json:"totalNum"`
	TotalPage   int64            `json:"totalPage"`
	Items       []SubAccountInfo `json:"items"`
}

// SubAccountInfo holds subaccount data for main, spot(trade), and margin accounts.
type SubAccountInfo struct {
	SubUserID      string            `json:"subUserId"`
	SubName        string            `json:"subName"`
	MainAccounts   []MainAccountInfo `json:"mainAccounts"`
	TradeAccounts  []MainAccountInfo `json:"tradeAccounts"`
	MarginAccounts []MainAccountInfo `json:"marginAccounts"`
}

// TransferableBalanceInfo represents transferable balance information
type TransferableBalanceInfo struct {
	baseAccount
	Transferable float64 `json:"transferable,string"`
}

// DepositAddress represents deposit address information for Spot and Margin trading.
type DepositAddress struct {
	Address         string `json:"address"`
	Memo            string `json:"memo"`
	Chain           string `json:"chain"`
	ContractAddress string `json:"contractAddress"` // missing in case of futures
}

type baseDeposit struct {
	Currency   string  `json:"currency"`
	Amount     float64 `json:"amount"`
	WalletTxID string  `json:"walletTxId"`
	IsInner    bool    `json:"isInner"`
	Status     string  `json:"status"`
}

// DepositResponse represents a detailed response for list of deposit.
type DepositResponse struct {
	CurrentPage int64     `json:"currentPage"`
	PageSize    int64     `json:"pageSize"`
	TotalNum    int64     `json:"totalNum"`
	TotalPage   int64     `json:"totalPage"`
	Items       []Deposit `json:"items"`
}

// Deposit represents deposit address and detail and timestamp information.
type Deposit struct {
	baseDeposit
	Address   string  `json:"address"`
	Memo      string  `json:"memo"`
	Fee       float64 `json:"fee"`
	Remark    string  `json:"remark"`
	CreatedAt kucoinTimeMilliSec
	UpdatedAt kucoinTimeMilliSec
}

// HistoricalDepositWithdrawalResponse represents deposit and withdrawal funding items details.
type HistoricalDepositWithdrawalResponse struct {
	CurrentPage int64                         `json:"currentPage"`
	PageSize    int64                         `json:"pageSize"`
	TotalNum    int64                         `json:"totalNum"`
	TotalPage   int64                         `json:"totalPage"`
	Items       []HistoricalDepositWithdrawal `json:"items"`
}

// HistoricalDepositWithdrawal represents deposit and withdrawal funding item.
type HistoricalDepositWithdrawal struct {
	baseDeposit
	CreatedAt kucoinTimeMilliSec `json:"createAt"`
}

// WithdrawalsResponse represents a withdrawals list of items details.
type WithdrawalsResponse struct {
	CurrentPage int64        `json:"currentPage"`
	PageSize    int64        `json:"pageSize"`
	TotalNum    int64        `json:"totalNum"`
	TotalPage   int64        `json:"totalPage"`
	Items       []Withdrawal `json:"items"`
}

// Withdrawal represents withdrawal funding information.
type Withdrawal struct {
	Deposit
	ID string `json:"id"`
}

// WithdrawalQuota represents withdrawal quota detail information.
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

// Fees represents taker and maker fee information a symbol.
type Fees struct {
	Symbol       string  `json:"symbol"`
	TakerFeeRate float64 `json:"takerFeeRate,string"`
	MakerFeeRate float64 `json:"makerFeeRate,string"`
}

// WSInstanceServers response connection token and websocket instance server information.
type WSInstanceServers struct {
	Token           string           `json:"token"`
	InstanceServers []InstanceServer `json:"instanceServers"`
}

// InstanceServer represents a single websocket instance server information.
type InstanceServer struct {
	Endpoint     string `json:"endpoint"`
	Encrypt      bool   `json:"encrypt"`
	Protocol     string `json:"protocol"`
	PingInterval int64  `json:"pingInterval"`
	PingTimeout  int64  `json:"pingTimeout"`
}

// WSConnMessages represents response messages ping, pong, and welcome message structures.
type WSConnMessages struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// WsSubscriptionInput represents a subscription information structure.
type WsSubscriptionInput struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Topic          string `json:"topic"`
	PrivateChannel bool   `json:"privateChannel"`
	Response       bool   `json:"response,omitempty"`
}

// WsPushData represents a push data from a server.
type WsPushData struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Topic       string          `json:"topic"`
	UserID      string          `json:"userId"`
	Subject     string          `json:"subject"`
	ChannelType string          `json:"channelType,omitempty"`
	Data        json.RawMessage `json:"data"`
}

// WsTicker represents a ticker push data from server.
type WsTicker struct {
	Sequence    string  `json:"sequence"`
	BestAsk     float64 `json:"bestAsk,string"`
	Size        float64 `json:"size,string"`
	BestBidSize float64 `json:"bestBidSize,string"`
	Price       float64 `json:"price,string"`
	BestAskSize float64 `json:"bestAskSize,string"`
	BestBid     float64 `json:"bestBid,string"`
}

// WsSpotTicker represents a spot ticker push data.
type WsSpotTicker struct {
	Trading         bool    `json:"trading"`
	Symbol          string  `json:"symbol"`
	Buy             float64 `json:"buy"`
	Sell            float64 `json:"sell"`
	Sort            int64   `json:"sort"`
	VolValue        float64 `json:"volValue"`
	BaseCurrency    string  `json:"baseCurrency"`
	Market          string  `json:"market"`
	QuoteCurrency   string  `json:"quoteCurrency"`
	SymbolCode      string  `json:"symbolCode"`
	Datetime        int64   `json:"datetime"`
	High            float64 `json:"high"`
	Vol             float64 `json:"vol"`
	Low             float64 `json:"low"`
	ChangePrice     float64 `json:"changePrice"`
	ChangeRate      float64 `json:"changeRate"`
	LastTradedPrice float64 `json:"lastTradedPrice"`
	Board           float64 `json:"board"`
	Mark            float64 `json:"mark"`
}

// WsOrderbook represents orderbook information.
type WsOrderbook struct {
	Changes struct {
		Asks [][3]string `json:"asks"`
		Bids [][3]string `json:"bids"`
	} `json:"changes"`
	SequenceEnd   int64              `json:"sequenceEnd"`
	SequenceStart int64              `json:"sequenceStart"`
	Symbol        string             `json:"symbol"`
	TimeMS        kucoinTimeMilliSec `json:"time"`
}

// WsLevel2Orderbook represents orderbook information.
type WsLevel2Orderbook struct {
	Asks   [][2]string        `json:"asks"`
	Bids   [][2]string        `json:"bids"`
	Symbol string             `json:"symbol"`
	TimeMS kucoinTimeMilliSec `json:"time"`
}

// WsCandlestickData represents candlestick information push data for a symbol.
type WsCandlestickData struct {
	Symbol  string    `json:"symbol"`
	Candles [7]string `json:"candles"`
	Time    int64     `json:"time"`
}

// WsCandlestick represents candlestick information push data for a symbol.
type WsCandlestick struct {
	Symbol  string `json:"symbol"`
	Candles struct {
		StartTime         time.Time
		OpenPrice         float64
		ClosePrice        float64
		HighPrice         float64
		LowPrice          float64
		TransactionVolume float64
		TransactionAmount float64
	} `json:"candles"`
	Time time.Time `json:"time"`
}

func (a *WsCandlestickData) getCandlestickData() (*WsCandlestick, error) {
	cand := &WsCandlestick{
		Symbol: a.Symbol,
		Time:   time.UnixMilli(a.Time),
	}
	timeStamp, err := strconv.ParseInt(a.Candles[0], 10, 64)
	if err != nil {
		return nil, err
	}
	cand.Candles.StartTime = time.UnixMilli(timeStamp)
	cand.Candles.OpenPrice, err = strconv.ParseFloat(a.Candles[1], 64)
	if err != nil {
		return nil, err
	}
	cand.Candles.ClosePrice, err = strconv.ParseFloat(a.Candles[2], 64)
	if err != nil {
		return nil, err
	}
	cand.Candles.HighPrice, err = strconv.ParseFloat(a.Candles[3], 64)
	if err != nil {
		return nil, err
	}
	cand.Candles.LowPrice, err = strconv.ParseFloat(a.Candles[4], 64)
	if err != nil {
		return nil, err
	}
	cand.Candles.TransactionVolume, err = strconv.ParseFloat(a.Candles[5], 64)
	if err != nil {
		return nil, err
	}
	cand.Candles.TransactionAmount, err = strconv.ParseFloat(a.Candles[6], 64)
	if err != nil {
		return nil, err
	}
	return cand, nil
}

// WsTrade represents a trade push data.
type WsTrade struct {
	Sequence     string  `json:"sequence"`
	Type         string  `json:"type"`
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	Price        float64 `json:"price,string"`
	Size         float64 `json:"size,string"`
	TradeID      string  `json:"tradeId"`
	TakerOrderID string  `json:"takerOrderId"`
	MakerOrderID string  `json:"makerOrderId"`
	Time         int64   `json:"time,string"`
}

// WsPriceIndicator represents index price or mark price indicator push data.
type WsPriceIndicator struct {
	Symbol      string  `json:"symbol"`
	Granularity float64 `json:"granularity"`
	Timestamp   int64   `json:"timestamp"`
	Value       float64 `json:"value"`
}

// WsMarginFundingBook represents order book changes on margin.
type WsMarginFundingBook struct {
	Sequence           int64             `json:"sequence"`
	Currency           string            `json:"currency"`
	DailyInterestRate  float64           `json:"dailyIntRate"`
	AnnualInterestRate float64           `json:"annualIntRate"`
	Term               int64             `json:"term"`
	Size               float64           `json:"size"`
	Side               string            `json:"side"`
	Timestamp          kucoinTimeNanoSec `json:"ts"` // In Nanosecond

}

// WsTradeOrder represents a private trade order push data.
type WsTradeOrder struct {
	Symbol     string            `json:"symbol"`
	OrderType  string            `json:"orderType"`
	Side       string            `json:"side"`
	OrderID    string            `json:"orderId"`
	Type       string            `json:"type"`
	OrderTime  kucoinTimeNanoSec `json:"orderTime"`
	Size       float64           `json:"size,string"`
	FilledSize float64           `json:"filledSize,string"`
	Price      float64           `json:"price,string"`
	ClientOid  string            `json:"clientOid"`
	RemainSize float64           `json:"remainSize,string"`
	Status     string            `json:"status"`
	Timestamp  kucoinTimeNanoSec `json:"ts"`
	Liquidity  string            `json:"liquidity,omitempty"`
	MatchPrice string            `json:"matchPrice,omitempty"`
	MatchSize  string            `json:"matchSize,omitempty"`
	TradeID    string            `json:"tradeId,omitempty"`
	OldSize    string            `json:"oldSize,omitempty"`
}

// WsAccountBalance represents a Account Balance push data.
type WsAccountBalance struct {
	Total           float64 `json:"total,string"`
	Available       float64 `json:"available,string"`
	AvailableChange float64 `json:"availableChange,string"`
	Currency        string  `json:"currency"`
	Hold            float64 `json:"hold,string"`
	HoldChange      float64 `json:"holdChange,string"`
	RelationEvent   string  `json:"relationEvent"`
	RelationEventID string  `json:"relationEventId"`
	RelationContext struct {
		Symbol  string `json:"symbol"`
		TradeID string `json:"tradeId"`
		OrderID string `json:"orderId"`
	} `json:"relationContext"`
	Time kucoinTimeMilliSecStr `json:"time"`
}

// WsDebtRatioChange represents a push data
type WsDebtRatioChange struct {
	DebtRatio float64            `json:"debtRatio"`
	TotalDebt string             `json:"totalDebt"`
	DebtList  map[string]string  `json:"debtList"`
	Timestamp kucoinTimeMilliSec `json:"timestamp"`
}

// WsPositionStatus represents a position status push data.
type WsPositionStatus struct {
	Type        string             `json:"type"`
	TimestampMS kucoinTimeMilliSec `json:"timestamp"`
}

// WsMarginTradeOrderEntersEvent represents a push data to the lenders
// when the order enters the order book or when the order is executed.
type WsMarginTradeOrderEntersEvent struct {
	Currency     string            `json:"currency"`
	OrderID      string            `json:"orderId"`      // Trade ID
	DailyIntRate float64           `json:"dailyIntRate"` // Daily interest rate.
	Term         int64             `json:"term"`         // Term (Unit: Day)
	Size         float64           `json:"size"`         // Size
	LentSize     float64           `json:"lentSize"`     // Size executed -- filled when the subject is order.update
	Side         string            `json:"side"`         // Lend or borrow. Currently, only "Lend" is available
	Timestamp    kucoinTimeNanoSec `json:"ts"`           // Timestamp (nanosecond)
}

// WsMarginTradeOrderDoneEvent represents a push message to the lenders when the order is completed.
type WsMarginTradeOrderDoneEvent struct {
	Currency  string            `json:"currency"`
	OrderID   string            `json:"orderId"`
	Reason    string            `json:"reason"`
	Side      string            `json:"side"`
	Timestamp kucoinTimeNanoSec `json:"ts"`
}

// WsStopOrder represents a stop order.
// When a stop order is received by the system, you will receive a message with "open" type.
// It means that this order entered the system and waited to be triggered.
type WsStopOrder struct {
	CreatedAt      kucoinTimeNanoSec `json:"createdAt"`
	OrderID        string            `json:"orderId"`
	OrderPrice     float64           `json:"orderPrice,string"`
	OrderType      string            `json:"orderType"`
	Side           string            `json:"side"`
	Size           float64           `json:"size,string"`
	Stop           string            `json:"stop"`
	StopPrice      float64           `json:"stopPrice,string"`
	Symbol         string            `json:"symbol"`
	TradeType      string            `json:"tradeType"`
	TriggerSuccess bool              `json:"triggerSuccess"`
	Timestamp      kucoinTimeNanoSec `json:"ts"`
	Type           string            `json:"type"`
}

// WsFuturesTicker represents a futures ticker push data.
type WsFuturesTicker struct {
	Symbol       string               `json:"symbol"`
	Sequence     int64                `json:"sequence"`
	Side         string               `json:"side"`
	FilledPrice  float64              `json:"price"`
	FilledSize   float64              `json:"size"`
	TradeID      string               `json:"tradeId"`
	BestBidSize  float64              `json:"bestBidSize"`
	BestBidPrice kucoinAmbiguousFloat `json:"bestBidPrice"`
	BestAskPrice kucoinAmbiguousFloat `json:"bestAskPrice"`
	BestAskSize  float64              `json:"bestAskSize"`
	FilledTime   kucoinTimeNanoSec    `json:"ts"`
}

// WsFuturesOrderbokInfo represents Level 2 order book information.
type WsFuturesOrderbokInfo struct {
	Sequence  int64              `json:"sequence"`
	Change    string             `json:"change"`
	Timestamp kucoinTimeMilliSec `json:"timestamp"`
}

// WsFuturesExecutionData represents execution data for symbol.
type WsFuturesExecutionData struct {
	Symbol           string            `json:"symbol"`
	Sequence         int64             `json:"sequence"`
	Side             string            `json:"side"`
	FilledQuantity   float64           `json:"matchSize"` // Filled quantity
	UnfilledQuantity float64           `json:"size"`
	FilledPrice      float64           `json:"price"`
	TakerOrderID     string            `json:"takerOrderId"`
	Time             kucoinTimeNanoSec `json:"time"`
	MakerOrderID     string            `json:"makerOrderId"`
	TradeID          string            `json:"tradeId"`
}

// WsOrderbookLevel5 represents an orderbook push data with depth level 5.
type WsOrderbookLevel5 struct {
	Asks      []orderbook.Item `json:"asks"`
	Bids      []orderbook.Item `json:"bids"`
	Timestamp time.Time        `json:"ts"`
}

// WsFundingRate represents the funding rate push data information through the websocket channel.
type WsFundingRate struct {
	Symbol      string  `json:"symbol"`
	Granularity int     `json:"granularity"`
	FundingRate float64 `json:"fundingRate"`
	Timestamp   int64   `json:"timestamp"`
}

// WsFuturesMarkPriceAndIndexPrice represents mark price and index price information.
type WsFuturesMarkPriceAndIndexPrice struct {
	Symbol      string  `json:"symbol"`
	Granularity int     `json:"granularity"`
	IndexPrice  float64 `json:"indexPrice"`
	MarkPrice   float64 `json:"markPrice"`
	Timestamp   int64   `json:"timestamp"`
}

// WsFuturesFundingBegin represents the Start Funding Fee Settlement.
type WsFuturesFundingBegin struct {
	Subject     string             `json:"subject"`
	Symbol      string             `json:"symbol"`
	FundingTime int64              `json:"fundingTime"`
	FundingRate float64            `json:"fundingRate"`
	Timestamp   kucoinTimeMilliSec `json:"timestamp"`
}

// WsFuturesTransactionStatisticsTimeEvent represents transaction statistics data.
type WsFuturesTransactionStatisticsTimeEvent struct {
	Symbol                   string            `json:"symbol"`
	Volume24H                float64           `json:"volume"`
	Turnover24H              float64           `json:"turnover"`
	LastPrice                int               `json:"lastPrice"`
	PriceChangePercentage24H float64           `json:"priceChgPct"`
	SnapshotTime             kucoinTimeNanoSec `json:"ts"`
}

// WsFuturesTradeOrder represents trade order information according to the market.
type WsFuturesTradeOrder struct {
	OrderID          string            `json:"orderId"`
	Symbol           string            `json:"symbol"`
	Type             string            `json:"type"`       // Message Type: "open", "match", "filled", "canceled", "update"
	Status           string            `json:"status"`     // Order Status: "match", "open", "done"
	MatchSize        string            `json:"matchSize"`  // Match Size (when the type is "match")
	MatchPrice       string            `json:"matchPrice"` // Match Price (when the type is "match")
	OrderType        string            `json:"orderType"`  // Order Type, "market" indicates market order, "limit" indicates limit order
	Side             string            `json:"side"`       // Trading direction,include buy and sell
	OrderPrice       float64           `json:"price,string"`
	OrderSize        float64           `json:"size,string"`
	RemainSize       float64           `json:"remainSize,string"`
	FilledSize       float64           `json:"filledSize,string"`   // Remaining Size for Trading
	CanceledSize     float64           `json:"canceledSize,string"` // In the update message, the Size of order reduced
	TradeID          string            `json:"tradeId"`             // Trade ID (when the type is "match")
	ClientOid        string            `json:"clientOid"`           // Client supplied order id.
	OrderTime        kucoinTimeNanoSec `json:"orderTime"`
	OldSize          string            `json:"oldSize "`  // Size Before Update (when the type is "update")
	TradingDirection string            `json:"liquidity"` // Liquidity, Trading direction, buy or sell in taker
	Timestamp        kucoinTimeNanoSec `json:"ts"`
}

// WsStopOrderLifecycleEvent represents futures stop order lifecycle event.
type WsStopOrderLifecycleEvent struct {
	OrderID        string             `json:"orderId"`
	Symbol         string             `json:"symbol"`
	Type           string             `json:"type"`
	OrderType      string             `json:"orderType"`
	Side           string             `json:"side"`
	Size           float64            `json:"size,string"`
	OrderPrice     float64            `json:"orderPrice,string"`
	Stop           string             `json:"stop"`
	StopPrice      float64            `json:"stopPrice,string"`
	StopPriceType  string             `json:"stopPriceType"`
	TriggerSuccess bool               `json:"triggerSuccess"`
	Error          string             `json:"error"`
	CreatedAt      kucoinTimeMilliSec `json:"createdAt"`
	Timestamp      kucoinTimeMilliSec `json:"ts"`
}

// WsFuturesOrderMarginEvent represents an order margin account balance event.
type WsFuturesOrderMarginEvent struct {
	OrderMargin float64            `json:"orderMargin"`
	Currency    string             `json:"currency"`
	Timestamp   kucoinTimeMilliSec `json:"timestamp"`
}

// WsFuturesAvailableBalance represents an available balance push data for futures account.
type WsFuturesAvailableBalance struct {
	AvailableBalance float64            `json:"availableBalance"`
	HoldBalance      float64            `json:"holdBalance"`
	Currency         string             `json:"currency"`
	Timestamp        kucoinTimeMilliSec `json:"timestamp"`
}

// WsFuturesWithdrawalAmountAndTransferOutAmountEvent represents Withdrawal Amount & Transfer-Out Amount Event push data.
type WsFuturesWithdrawalAmountAndTransferOutAmountEvent struct {
	WithdrawHold float64            `json:"withdrawHold"` // Current frozen amount for withdrawal
	Currency     string             `json:"currency"`
	Timestamp    kucoinTimeMilliSec `json:"timestamp"`
}

// WsFuturesPosition represents futures account position change event.
type WsFuturesPosition struct {
	RealisedGrossPnl  float64            `json:"realisedGrossPnl"` // Accumulated realised profit and loss
	Symbol            string             `json:"symbol"`
	CrossMode         bool               `json:"crossMode"`        // Cross mode or not
	LiquidationPrice  float64            `json:"liquidationPrice"` // Liquidation price
	PosLoss           float64            `json:"posLoss"`          // Manually added margin amount
	AvgEntryPrice     float64            `json:"avgEntryPrice"`    // Average entry price
	UnrealisedPnl     float64            `json:"unrealisedPnl"`    // Unrealised profit and loss
	MarkPrice         float64            `json:"markPrice"`        // Mark price
	PosMargin         float64            `json:"posMargin"`        // Position margin
	AutoDeposit       bool               `json:"autoDeposit"`      // Auto deposit margin or not
	RiskLimit         float64            `json:"riskLimit"`
	UnrealisedCost    float64            `json:"unrealisedCost"`    // Unrealised value
	PosComm           float64            `json:"posComm"`           // Bankruptcy cost
	PosMaint          float64            `json:"posMaint"`          // Maintenance margin
	PosCost           float64            `json:"posCost"`           // Position value
	MaintMarginReq    float64            `json:"maintMarginReq"`    // Maintenance margin rate
	BankruptPrice     float64            `json:"bankruptPrice"`     // Bankruptcy price
	RealisedCost      float64            `json:"realisedCost"`      // Currently accumulated realised position value
	MarkValue         float64            `json:"markValue"`         // Mark value
	PosInit           float64            `json:"posInit"`           // Position margin
	RealisedPnl       float64            `json:"realisedPnl"`       // Realised profit and loss
	MaintMargin       float64            `json:"maintMargin"`       // Position margin
	RealLeverage      float64            `json:"realLeverage"`      // Leverage of the order
	ChangeReason      string             `json:"changeReason"`      // changeReason:marginChange、positionChange、liquidation、autoAppendMarginStatusChange、adl
	CurrentCost       float64            `json:"currentCost"`       // Current position value
	OpeningTimestamp  kucoinTimeMilliSec `json:"openingTimestamp"`  // Open time
	CurrentQty        float64            `json:"currentQty"`        // Current position
	DelevPercentage   float64            `json:"delevPercentage"`   // ADL ranking percentile
	CurrentComm       float64            `json:"currentComm"`       // Current commission
	RealisedGrossCost float64            `json:"realisedGrossCost"` // Accumulated realised gross profit value
	IsOpen            bool               `json:"isOpen"`            // Opened position or not
	PosCross          float64            `json:"posCross"`          // Manually added margin
	CurrentTimestamp  kucoinTimeMilliSec `json:"currentTimestamp"`  // Current timestamp
	UnrealisedRoePcnt float64            `json:"unrealisedRoePcnt"` // Rate of return on investment
	UnrealisedPnlPcnt float64            `json:"unrealisedPnlPcnt"` // Position profit and loss ratio
	SettleCurrency    string             `json:"settleCurrency"`    // Currency used to clear and settle the trades
}

// WsFuturesMarkPricePositionChanges represents futures account position change caused by mark price.
type WsFuturesMarkPricePositionChanges struct {
	MarkPrice         float64            `json:"markPrice"`         // Mark price
	MarkValue         float64            `json:"markValue"`         // Mark value
	MaintMargin       float64            `json:"maintMargin"`       // Position margin
	RealLeverage      float64            `json:"realLeverage"`      // Leverage of the order
	UnrealisedPnl     float64            `json:"unrealisedPnl"`     // Unrealised profit and lost
	UnrealisedRoePcnt float64            `json:"unrealisedRoePcnt"` // Rate of return on investment
	UnrealisedPnlPcnt float64            `json:"unrealisedPnlPcnt"` // Position profit and loss ratio
	DelevPercentage   float64            `json:"delevPercentage"`   // ADL ranking percentile
	CurrentTimestamp  kucoinTimeMilliSec `json:"currentTimestamp"`  // Current timestamp
	SettleCurrency    string             `json:"settleCurrency"`    // Currency used to clear and settle the trades
}

// WsFuturesPositionFundingSettlement represents futures account position funding settlement push data.
type WsFuturesPositionFundingSettlement struct {
	PositionSize     float64            `json:"qty"`
	MarkPrice        float64            `json:"markPrice"`
	FundingRate      float64            `json:"fundingRate"`
	FundingFee       float64            `json:"fundingFee"`
	FundingTime      kucoinTimeMilliSec `json:"fundingTime"`
	CurrentTimestamp kucoinTimeNanoSec  `json:"ts"`
	SettleCurrency   string             `json:"settleCurrency"`
}

// IsolatedMarginBorrowing represents response data for initiating isolated margin borrowing.
type IsolatedMarginBorrowing struct {
	OrderID    string  `json:"orderId"`
	Currency   string  `json:"currency"`
	ActualSize float64 `json:"actualSize,string"`
}

// Response represents response model and implements UnmarshalTo interface.
type Response struct {
	Data interface{} `json:"data"`
	Error
}

// CancelOrderResponse represents cancel order response model.
type CancelOrderResponse struct {
	CancelledOrderID string `json:"cancelledOrderId"`
	ClientOID        string `json:"clientOid"`
	Error
}

// AccountLedgerResponse represents the account ledger response detailed information
type AccountLedgerResponse struct {
	CurrentPage int64        `json:"currentPage"`
	PageSize    int64        `json:"pageSize"`
	TotalNum    int64        `json:"totalNum"`
	TotalPage   int64        `json:"totalPage"`
	Items       []LedgerInfo `json:"items"`
}

// SpotAPISubAccountParams parameters for Spot APIs for sub-accounts
type SpotAPISubAccountParams struct {
	SubAccountName string `json:"subName"`
	Passphrase     string `json:"passphrase"`
	Remark         string `json:"remark"`
	Permission     string `json:"permission,omitempty"`    // Permissions(Only "General" and "Trade" permissions can be set, such as "General, Trade". The default is "General")
	IPWhitelist    string `json:"ipWhitelist,omitempty"`   // IP whitelist(You may add up to 20 IPs. Use a halfwidth comma to each IP)
	Expire         int    `json:"expire,string,omitempty"` // API expiration time; Never expire(default)-1，30Day30，90Day90，180Day180，360Day360
}

// SubAccountResponse represents the sub-user detail.
type SubAccountResponse struct {
	CurrentPage int64        `json:"currentPage"`
	PageSize    int64        `json:"pageSize"`
	TotalNum    int64        `json:"totalNum"`
	TotalPage   int64        `json:"totalPage"`
	Items       []SubAccount `json:"items"`
}

// SubAccount represents sub-user
type SubAccount struct {
	UserID    string             `json:"userId"`
	SubName   string             `json:"subName"`
	Type      int                `json:"type"` //type:1-rebot  or type:0-nomal
	Remarks   string             `json:"remarks"`
	UID       int                `json:"uid,omitempty"`
	Status    int                `json:"status,omitempty"`
	Access    string             `json:"access,omitempty"`
	CreatedAt kucoinTimeMilliSec `json:"createdAt,omitempty"`
}

// SpotAPISubAccount represents a Spot APIs for sub-accounts.
type SpotAPISubAccount struct {
	SubName     string             `json:"subName"`
	Remark      string             `json:"remark"`
	APIKey      string             `json:"apiKey"`
	APISecret   string             `json:"apiSecret"`
	Passphrase  string             `json:"passphrase"`
	Permission  string             `json:"permission"`
	IPWhitelist string             `json:"ipWhitelist"`
	CreatedAt   kucoinTimeMilliSec `json:"createdAt,omitempty"`
}

// DeleteSubAccountResponse represents delete sub-account response.
type DeleteSubAccountResponse struct {
	SubAccountName string `json:"subName"`
	APIKey         string `json:"apiKey"`
}

// ConnectionMessage represents a connection and subscription status message.
type ConnectionMessage struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}
