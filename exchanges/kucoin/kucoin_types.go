package kucoin

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Trade type values for Spot, Isolated Margin, and Cross Margin accounts
const (
	SpotTradeType           = "TRADE"
	IsolatedMarginTradeType = "MARGIN_ISOLATED_TRADE"
	CrossMarginTradeType    = "MARGIN_TRADE"
)

var (
	validPeriods = []string{
		"1min", "3min", "5min", "15min", "30min", "1hour", "2hour", "4hour", "6hour", "8hour", "12hour", "1day", "1week",
	}

	errInvalidResponseReceiver    = errors.New("invalid response receiver")
	errInvalidStopPriceType       = errors.New("stopPriceType is required")
	errMalformedData              = errors.New("malformed data")
	errNoDepositAddress           = errors.New("no deposit address found")
	errAddressRequired            = errors.New("address is required")
	errMultipleDepositAddress     = errors.New("multiple deposit addresses")
	errInvalidResultInterface     = errors.New("result interface has to be pointer")
	errInvalidSubAccountName      = errors.New("invalid sub-account name")
	errRemarkIsRequired           = errors.New("remark with a 24 characters max-length is required")
	errAPIKeyRequired             = errors.New("account API key is required")
	errInvalidPassPhraseInstance  = errors.New("invalid passphrase string")
	errNoValidResponseFromServer  = errors.New("no valid response from server")
	errMissingOrderbookSequence   = errors.New("missing orderbook sequence")
	errSizeOrFundIsRequired       = errors.New("at least one required among size and funds")
	errInvalidLeverage            = errors.New("invalid leverage value")
	errAccountTypeMissing         = errors.New("account type is required")
	errTransferTypeMissing        = errors.New("transfer type is required")
	errTradeTypeMissing           = errors.New("trade type is missing")
	errTimeInForceRequired        = errors.New("time in force is required")
	errInvalidMsgType             = errors.New("message type field not valid")
	errMissingPurchaseOrderNumber = errors.New("missing purchase order number")
	errMissingInterestRate        = errors.New("interest rate is required")
	errAccountIDMissing           = errors.New("account ID is required")
	errQueryDateIsRequired        = errors.New("query date is required")
	errOffsetIsRequired           = errors.New("offset is required")
	errProductIDMissing           = errors.New("product ID is missing")
	errStatusMissing              = errors.New("status is missing")
	errInvalidPeriod              = errors.New("invalid period")
	errTransferDirectionRequired  = errors.New("transfer direction cannot be empty")
	errSubUserIDRequired          = errors.New("sub-user ID is required")
	errPageSizeRequired           = errors.New("pageSize is required")
	errCurrentPageRequired        = errors.New("current page value is required")
	errTimeoutRequired            = errors.New("timeout value required")
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

// GetError checks and returns an error if it is supplied
func (e Error) GetError() error {
	code, err := strconv.ParseInt(e.Code, 10, 64)
	if err != nil {
		return err
	}
	switch code {
	case 200:
		return nil
	case 200000:
		if e.Msg == "" {
			return nil
		}
	}
	return fmt.Errorf("code: %s message: %s", e.Code, e.Msg)
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
	Sequence    string     `json:"sequence"`
	BestAsk     float64    `json:"bestAsk,string"`
	Size        float64    `json:"size,string"`
	Price       float64    `json:"price,string"`
	BestBidSize float64    `json:"bestBidSize,string"`
	BestBid     float64    `json:"bestBid,string"`
	BestAskSize float64    `json:"bestAskSize,string"`
	Time        types.Time `json:"time"`
}

// TickerInfoBase represents base price ticker details
type TickerInfoBase struct {
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
	BestBidSize      float64 `json:"bestBidSize,string"`
	BestAskSize      float64 `json:"bestAskSize,string"`
}

// TickerInfo stores ticker information
type TickerInfo struct {
	TickerInfoBase
	SymbolName string `json:"symbolName"`
}

// Stats24hrs stores 24 hrs statistics
type Stats24hrs struct {
	TickerInfoBase
	Time types.Time `json:"time"`
}

// Orderbook stores the orderbook data
type Orderbook struct {
	Sequence int64
	Bids     []orderbook.Level
	Asks     []orderbook.Level
	Time     time.Time
}

type orderbookResponse struct {
	Asks     [][2]types.Number `json:"asks"`
	Bids     [][2]types.Number `json:"bids"`
	Time     types.Time        `json:"time"`
	Sequence string            `json:"sequence"`
}

// Trade stores trade data
type Trade struct {
	Sequence string     `json:"sequence"`
	Price    float64    `json:"price,string"`
	Size     float64    `json:"size,string"`
	Side     string     `json:"side"`
	Time     types.Time `json:"time"`
}

// Kline stores kline data
type Kline struct {
	StartTime types.Time
	Open      types.Number
	Close     types.Number
	High      types.Number
	Low       types.Number
	Volume    types.Number // Transaction volume
	Amount    types.Number // Transaction amount
}

// UnmarshalJSON deserilizes kline data from a JSON array into Kline fields.
func (k *Kline) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&k.StartTime, &k.Open, &k.Close, &k.High, &k.Low, &k.Volume, &k.Amount})
}

// CurrencyBase represents currency code response details
type CurrencyBase struct {
	Currency        string `json:"currency"` // a unique currency code that will never change
	Name            string `json:"name"`     // will change after renaming
	FullName        string `json:"fullName"`
	Precision       int64  `json:"precision"`
	Confirms        int64  `json:"confirms"`
	ContractAddress string `json:"contractAddress"`
	IsMarginEnabled bool   `json:"isMarginEnabled"`
	IsDebitEnabled  bool   `json:"isDebitEnabled"`
}

// Currency stores currency data
type Currency struct {
	CurrencyBase
	WithdrawalMinSize float64 `json:"withdrawalMinSize,string"`
	WithdrawalMinFee  float64 `json:"withdrawalMinFee,string"`
	IsWithdrawEnabled bool    `json:"isWithdrawEnabled"`
	IsDepositEnabled  bool    `json:"isDepositEnabled"`
}

// Chain stores blockchain data
type Chain struct {
	ChainName         string       `json:"chainName"`
	Confirms          int64        `json:"confirms"`
	ContractAddress   string       `json:"contractAddress"`
	WithdrawalMinSize float64      `json:"withdrawalMinSize,string"`
	WithdrawalMinFee  float64      `json:"withdrawalMinFee,string"`
	IsWithdrawEnabled bool         `json:"isWithdrawEnabled"`
	IsDepositEnabled  bool         `json:"isDepositEnabled"`
	DepositMinSize    any          `json:"depositMinSize"`
	WithdrawFeeRate   types.Number `json:"withdrawFeeRate"`
	PreConfirms       int64        `json:"preConfirms"`
	ChainID           string       `json:"chainId"`
}

// CurrencyDetail stores currency details
type CurrencyDetail struct {
	CurrencyBase
	Chains []Chain `json:"chains"`
}

// LeveragedTokenInfo represents a leveraged token information
type LeveragedTokenInfo struct {
	Currency              string       `json:"currency"`
	NetAsset              float64      `json:"netAsset"`
	IssuedSize            types.Number `json:"issuedSize"`
	TargetLeverage        types.Number `json:"targetLeverage"`
	ActualLeverage        types.Number `json:"actualLeverage"`
	AssetsUnderManagement string       `json:"assetsUnderManagement"`
	Basket                string       `json:"basket"`
}

// MarkPrice stores mark price data
type MarkPrice struct {
	Symbol      string     `json:"symbol"`
	Granularity int64      `json:"granularity"`
	TimePoint   types.Time `json:"timePoint"`
	Value       float64    `json:"value"`
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
	AvailableBalance float64 `json:"availableBalance,string"`
	Currency         string  `json:"currency"`
	HoldBalance      float64 `json:"holdBalance,string"`
	Liability        float64 `json:"liability,string"`
	MaxBorrowSize    float64 `json:"maxBorrowSize,string"`
	TotalBalance     float64 `json:"totalBalance,string"`
}

// MarginAccounts stores margin accounts data
type MarginAccounts struct {
	Accounts  []MarginAccount `json:"accounts"`
	DebtRatio float64         `json:"debtRatio,string"`
}

// CrossMarginRiskLimitCurrencyConfig currency configuration of cross margin accounts
type CrossMarginRiskLimitCurrencyConfig struct {
	Timestamp         types.Time   `json:"timestamp"`
	Currency          string       `json:"currency"`
	BorrowMaxAmount   types.Number `json:"borrowMaxAmount"`
	BuyMaxAmount      types.Number `json:"buyMaxAmount"`
	HoldMaxAmount     types.Number `json:"holdMaxAmount"`
	BorrowCoefficient string       `json:"borrowCoefficient"`
	MarginCoefficient string       `json:"marginCoefficient"`
	Precision         float64      `json:"precision"`
	BorrowMinAmount   types.Number `json:"borrowMinAmount"`
	BorrowMinUnit     string       `json:"borrowMinUnit"`
	BorrowEnabled     bool         `json:"borrowEnabled"`
}

// IsolatedMarginRiskLimitCurrencyConfig represents a currency configuration of isolated margin account
type IsolatedMarginRiskLimitCurrencyConfig struct {
	Timestamp              types.Time   `json:"timestamp"`
	Symbol                 string       `json:"symbol"`
	BaseMaxBorrowAmount    types.Number `json:"baseMaxBorrowAmount"`
	QuoteMaxBorrowAmount   types.Number `json:"quoteMaxBorrowAmount"`
	BaseMaxBuyAmount       types.Number `json:"baseMaxBuyAmount"`
	QuoteMaxBuyAmount      types.Number `json:"quoteMaxBuyAmount"`
	BaseMaxHoldAmount      types.Number `json:"baseMaxHoldAmount"`
	QuoteMaxHoldAmount     types.Number `json:"quoteMaxHoldAmount"`
	BasePrecision          int64        `json:"basePrecision"`
	QuotePrecision         int64        `json:"quotePrecision"`
	BaseBorrowCoefficient  types.Number `json:"baseBorrowCoefficient"`
	QuoteBorrowCoefficient types.Number `json:"quoteBorrowCoefficient"`
	BaseMarginCoefficient  types.Number `json:"baseMarginCoefficient"`
	QuoteMarginCoefficient types.Number `json:"quoteMarginCoefficient"`
	BaseBorrowMinAmount    string       `json:"baseBorrowMinAmount"`
	BaseBorrowMinUnit      string       `json:"baseBorrowMinUnit"`
	QuoteBorrowMinAmount   types.Number `json:"quoteBorrowMinAmount"`
	QuoteBorrowMinUnit     types.Number `json:"quoteBorrowMinUnit"`
	BaseBorrowEnabled      bool         `json:"baseBorrowEnabled"`
	QuoteBorrowEnabled     bool         `json:"quoteBorrowEnabled"`
}

// MarginRiskLimit stores margin risk limit
type MarginRiskLimit struct {
	Currency            string  `json:"currency"`
	MaximumBorrowAmount float64 `json:"borrowMaxAmount,string"`
	MaxumumBuyAmount    float64 `json:"buyMaxAmount,string"`
	MaximumHoldAmount   float64 `json:"holdMaxAmount,string"`
	Precision           int64   `json:"precision"`
}

// MarginBorrowParam represents a margin borrow parameter
type MarginBorrowParam struct {
	Currency    currency.Code `json:"currency"`
	Size        float64       `json:"size"`
	IsIsolated  bool          `json:"isisolated"`
	Symbol      currency.Pair `json:"symbol"`
	TimeInForce string        `json:"timeInForce"`
}

// RepayParam represents a repayment parameter for cross and isolated margin orders
type RepayParam struct {
	Currency   currency.Code `json:"currency"`
	Size       float64       `json:"size"`
	IsIsolated bool          `json:"isisolated"`
	Symbol     currency.Pair `json:"symbol"`
}

// BorrowAndRepaymentOrderResp stores borrow order response
type BorrowAndRepaymentOrderResp struct {
	OrderNo     string       `json:"orderNo"`
	ActualSize  float64      `json:"actualSize"`
	Symbol      string       `json:"symbol"`
	Currency    string       `json:"currency"`
	Size        types.Number `json:"size"`
	Principal   types.Number `json:"principal"`
	Interest    types.Number `json:"interest"`
	Status      string       `json:"status"`
	CreatedTime types.Time   `json:"createdTime"`
}

// BorrowRepayDetailResponse a full response of borrow and repay order
type BorrowRepayDetailResponse struct {
	CurrentPage int64                   `json:"currentPage"`
	PageSize    int64                   `json:"pageSize"`
	TotalNum    int64                   `json:"totalNum"`
	TotalPage   int64                   `json:"totalPage"`
	Items       []BorrowRepayDetailItem `json:"items"`
}

// BorrowRepayDetailItem represents a borrow and repay order detail
type BorrowRepayDetailItem struct {
	OrderNo     string       `json:"orderNo"`
	Symbol      string       `json:"symbol"`
	Currency    string       `json:"currency"`
	Size        float64      `json:"size"`
	Principal   types.Number `json:"principal"`
	Interest    types.Number `json:"interest"`
	ActualSize  float64      `json:"actualSize"`
	Status      string       `json:"status"`
	CreatedTime types.Time   `json:"createdTime"`
}

// BorrowOrder stores borrow order
type BorrowOrder struct {
	OrderID   string                 `json:"orderId"`
	Currency  string                 `json:"currency"`
	Size      float64                `json:"size,string"`
	Filled    float64                `json:"filled"`
	MatchList []BorrowOrderMatchItem `json:"matchList"`
	Status    string                 `json:"status"`
}

// BorrowOrderMatchItem represents a borrow order match item detail
type BorrowOrderMatchItem struct {
	TradeID      string     `json:"tradeId"`
	Currency     string     `json:"currency"`
	DailyIntRate float64    `json:"dailyIntRate,string"`
	Size         float64    `json:"size,string"`
	Term         int64      `json:"term"`
	Timestamp    types.Time `json:"timestamp"`
}

type baseRecord struct {
	TradeID      string  `json:"tradeId"`
	Currency     string  `json:"currency"`
	DailyIntRate float64 `json:"dailyIntRate,string"`
	Principal    float64 `json:"principal,string"`
	RepaidSize   float64 `json:"repaidSize,string"`
	Term         int64   `json:"term"`
}

// RepaidRecordsResponse stores list of repaid record details
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
	Interest  float64    `json:"interest,string"`
	RepayTime types.Time `json:"repayTime"`
}

// LendOrder stores lend order
type LendOrder struct {
	OrderID      string     `json:"orderId"`
	Currency     string     `json:"currency"`
	Size         float64    `json:"size,string"`
	FilledSize   float64    `json:"filledSize,string"`
	DailyIntRate float64    `json:"dailyIntRate,string"`
	Term         int64      `json:"term"`
	CreatedAt    types.Time `json:"createdAt"`
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

// AssetInfo holds asset information for an instrument
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
	LoanID            string     `json:"loanId"`
	Symbol            string     `json:"symbol"`
	Currency          string     `json:"currency"`
	PrincipalTotal    float64    `json:"principalTotal,string"`
	InterestBalance   float64    `json:"interestBalance,string"`
	CreatedAt         types.Time `json:"createdAt"`
	Period            int64      `json:"period"`
	RepaidSize        float64    `json:"repaidSize,string"`
	DailyInterestRate float64    `json:"dailyInterestRate,string"`
}

// ServiceStatus represents a service status message
type ServiceStatus struct {
	Status  string `json:"status"`
	Message string `json:"msg"`
}

// PlaceHFParam represents a place HF order parameters
type PlaceHFParam struct {
	ClientOrderID       string        `json:"clientOid,omitempty"`
	Symbol              currency.Pair `json:"symbol"`
	OrderType           string        `json:"type"`
	Side                string        `json:"side"`
	SelfTradePrevention string        `json:"stp,omitempty"`
	OrderTags           string        `json:"tags,omitempty"`
	Remark              string        `json:"remark,omitempty"`

	// Additional 'limit' order parameters
	Price       float64 `json:"price,string,omitempty"`
	Size        float64 `json:"size,string,omitempty"`
	TimeInForce string  `json:"timeInForce"`
	CancelAfter int64   `json:"cancelAfter"`
	PostOnly    bool    `json:"postOnly"`
	Hidden      bool    `json:"hidden"`
	Iceberg     bool    `json:"iceberg"`
	VisibleSize float64 `json:"visibleSize"`

	// Additional 'market' parameters
	Funds string `json:"funds"`
}

// ModifyHFOrderParam represents a modify high frequency order parameter
type ModifyHFOrderParam struct {
	Symbol        currency.Pair `json:"symbol"`
	ClientOrderID string        `json:"clientOid,omitempty"`
	OrderID       string        `json:"orderId,omitempty"`
	NewPrice      float64       `json:"newPrice,omitempty,string"`
	NewSize       float64       `json:"newSize,omitempty,string"`
}

// SyncCancelHFOrderResp represents a cancel sync high frequency order
type SyncCancelHFOrderResp struct {
	OrderID      string       `json:"orderId"`
	OriginSize   types.Number `json:"originSize"`
	OriginFunds  string       `json:"originFunds"`
	DealSize     types.Number `json:"dealSize"`
	RemainSize   types.Number `json:"remainSize"`
	CanceledSize types.Number `json:"canceledSize"`
	Status       string       `json:"status"`
}

// PlaceOrderResp represents a place order response
type PlaceOrderResp struct {
	OrderID string `json:"orderId"`
	Success bool   `json:"success"`
}

// SyncPlaceHFOrderResp represents a request parameter for high frequency sync orders
type SyncPlaceHFOrderResp struct {
	OrderID       string       `json:"orderId"`
	OrderTime     types.Time   `json:"orderTime"`
	OriginSize    types.Number `json:"originSize"`
	OriginFunds   string       `json:"originFunds"`
	DealSize      types.Number `json:"dealSize"`
	DealFunds     string       `json:"dealFunds"`
	RemainSize    types.Number `json:"remainSize"`
	RemainFunds   types.Number `json:"remainFunds"`
	CanceledSize  string       `json:"canceledSize"`
	CanceledFunds string       `json:"canceledFunds"`
	Status        string       `json:"status"`
	MatchTime     types.Time   `json:"matchTime"`
}

// CancelOrderByNumberResponse represents response for canceling an order by number
type CancelOrderByNumberResponse struct {
	OrderID    string `json:"orderId"`
	CancelSize string `json:"cancelSize"`
}

// CancelAllHFOrdersResponse represents a response for cancelling all high-frequency orders
type CancelAllHFOrdersResponse struct {
	SucceedSymbols []string                        `json:"succeedSymbols"`
	FailedSymbols  []FailedHFOrderCancellationInfo `json:"failedSymbols"`
}

// FailedHFOrderCancellationInfo represents a failed order cancellation information
type FailedHFOrderCancellationInfo struct {
	Symbol string `json:"symbol"`
	Error  string `json:"error"`
}

// CompletedHFOrder represents a completed HF orders list
type CompletedHFOrder struct {
	LastID int64         `json:"lastId"`
	Items  []OrderDetail `json:"items"`
}

// AutoCancelHFOrderResponse represents an auto cancel HF order response
type AutoCancelHFOrderResponse struct {
	Timeout     int64      `json:"timeout"`
	Symbols     string     `json:"symbols"`
	CurrentTime types.Time `json:"currentTime"`
	TriggerTime types.Time `json:"triggerTime"`
}

// HFOrderFills represents an HF order list
type HFOrderFills struct {
	Items  []Fill `json:"items"`
	LastID int64  `json:"lastId"`
}

// PlaceOrderParams represents a batch place order parameters
type PlaceOrderParams struct {
	OrderList []PlaceHFParam `json:"orderList"`
}

// CompletedRepaymentRecord represents repayment records of isolated margin positions
type CompletedRepaymentRecord struct {
	baseRepaymentRecord
	RepayFinishAt types.Time `json:"repayFinishAt"`
}

// PostMarginOrderResp represents response data for placing margin orders
type PostMarginOrderResp struct {
	OrderID     string  `json:"orderId"`
	BorrowSize  float64 `json:"borrowSize"`
	LoanApplyID string  `json:"loanApplyId"`
}

// OrderRequest represents place order request parameters
type OrderRequest struct {
	ClientOID           string  `json:"clientOid"`
	Symbol              string  `json:"symbol"`
	Side                string  `json:"side"`
	Type                string  `json:"type,omitempty"`             // optional
	Remark              string  `json:"remark,omitempty"`           // optional
	Stop                string  `json:"stop,omitempty"`             // optional
	StopPrice           float64 `json:"stopPrice,string,omitempty"` // optional
	SelfTradePrevention string  `json:"stp,omitempty"`              // optional
	Price               float64 `json:"price,string,omitempty"`
	Size                float64 `json:"size,string,omitempty"`
	TimeInForce         string  `json:"timeInForce,omitempty"` // optional
	CancelAfter         int64   `json:"cancelAfter,omitempty"` // optional
	PostOnly            bool    `json:"postOnly,omitempty"`    // optional
	Hidden              bool    `json:"hidden,omitempty"`      // optional
	Iceberg             bool    `json:"iceberg,omitempty"`     // optional
	VisibleSize         string  `json:"visibleSize,omitempty"` // optional
}

// PostBulkOrderResp response data for submitting a bulk order
type PostBulkOrderResp struct {
	OrderRequest
	ID      string `json:"id"`
	Channel string `json:"channel"`
	Status  string `json:"status"`
	FailMsg string `json:"failMsg"`
}

// OrdersListResponse represents an order list response
type OrdersListResponse struct {
	CurrentPage int64         `json:"currentPage"`
	PageSize    int64         `json:"pageSize"`
	TotalNum    int64         `json:"totalNum"`
	TotalPage   int64         `json:"totalPage"`
	Items       []OrderDetail `json:"items"`
}

// OrderDetail represents order detail
type OrderDetail struct {
	ID                  string       `json:"id"`
	ClientOID           string       `json:"clientOid"`
	Symbol              string       `json:"symbol"`
	Side                string       `json:"side"`
	Type                string       `json:"type"`          // optional
	Remark              string       `json:"remark"`        // optional
	Stop                string       `json:"stop"`          // optional
	StopPrice           types.Number `json:"stopPrice"`     // optional
	SelfTradePrevention string       `json:"stp,omitempty"` // optional
	Price               types.Number `json:"price"`
	Size                types.Number `json:"size"`
	TimeInForce         string       `json:"timeInForce"` // optional
	CancelAfter         int64        `json:"cancelAfter"` // optional
	PostOnly            bool         `json:"postOnly"`    // optional
	Hidden              bool         `json:"hidden"`      // optional
	Iceberg             bool         `json:"iceberg"`     // optional
	VisibleSize         types.Number `json:"visibleSize"` // optional
	Channel             string       `json:"channel"`
	OperationType       string       `json:"opType"` // operation type: DEAL
	Funds               string       `json:"funds"`
	DealFunds           string       `json:"dealFunds"`
	DealSize            types.Number `json:"dealSize"`
	Fee                 types.Number `json:"fee"`
	FeeCurrency         string       `json:"feeCurrency"`
	StopTriggered       bool         `json:"stopTriggered"`
	Tags                string       `json:"tags"`
	IsActive            bool         `json:"isActive"`
	CancelExist         bool         `json:"cancelExist"`
	CreatedAt           types.Time   `json:"createdAt"`
	TradeType           string       `json:"tradeType"`

	// Used by HF Orders
	Active        bool       `json:"active"`
	InOrderBook   bool       `json:"inOrderBook"`
	LastUpdatedAt types.Time `json:"lastUpdatedAt"`

	// Added for HF Spot orders
	CancelledSize  types.Number `json:"cancelledSize"`
	CancelledFunds string       `json:"cancelledFunds"`
	RemainSize     types.Number `json:"remainSize"`
	RemainFunds    string       `json:"remainFunds"`
}

// ListFills represents fills response list detail
type ListFills struct {
	CurrentPage int64  `json:"currentPage"`
	PageSize    int64  `json:"pageSize"`
	TotalNumber int64  `json:"totalNum"`
	TotalPage   int64  `json:"totalPage"`
	Items       []Fill `json:"items"`
}

// Fill represents order fills for margin and spot orders
type Fill struct {
	Symbol         string     `json:"symbol"`
	TradeID        string     `json:"tradeId"`
	OrderID        string     `json:"orderId"`
	CounterOrderID string     `json:"counterOrderId"`
	Side           string     `json:"side"`
	Liquidity      string     `json:"liquidity"`
	ForceTaker     bool       `json:"forceTaker"`
	Price          float64    `json:"price,string"`
	Size           float64    `json:"size,string"`
	Funds          float64    `json:"funds,string"`
	Fee            float64    `json:"fee,string"`
	FeeRate        float64    `json:"feeRate,string"`
	FeeCurrency    string     `json:"feeCurrency"`
	Stop           string     `json:"stop"`
	OrderType      string     `json:"type"`
	CreatedAt      types.Time `json:"createdAt"`
	TradeType      string     `json:"tradeType"`

	// Used by HF orders
	ID int64 `json:"id"`
}

// StopOrderListResponse represents a list of spot orders details
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
	ID              string     `json:"id"`
	UserID          string     `json:"userId"`
	Status          string     `json:"status"`
	Funds           float64    `json:"funds,string"`
	Channel         string     `json:"channel"`
	Tags            string     `json:"tags"`
	DomainID        string     `json:"domainId"`
	TradeSource     string     `json:"tradeSource"`
	TradeType       string     `json:"tradeType"`
	FeeCurrency     string     `json:"feeCurrency"`
	TakerFeeRate    string     `json:"takerFeeRate"`
	MakerFeeRate    string     `json:"makerFeeRate"`
	CreatedAt       types.Time `json:"createdAt"`
	OrderTime       types.Time `json:"orderTime"`
	StopTriggerTime types.Time `json:"stopTriggerTime"`
}

// AccountInfo represents account information
type AccountInfo struct {
	ID          string       `json:"id"`
	Currency    string       `json:"currency"`
	AccountType string       `json:"type"` // Account type:，main、trade、trade_hf、margin
	Balance     types.Number `json:"balance"`
	Available   types.Number `json:"available"`
	Holds       types.Number `json:"holds"`
}

// CrossMarginAccountDetail represents a cross-margin account details
type CrossMarginAccountDetail struct {
	Timestamp   types.Time                   `json:"timestamp"`
	CurrentPage int64                        `json:"currentPage"`
	PageSize    int64                        `json:"pageSize"`
	TotalNum    int64                        `json:"totalNum"`
	TotalPage   int64                        `json:"totalPage"`
	Items       []CrossMarginStatusAndAssets `json:"items"`
}

// CrossMarginStatusAndAssets represents a cross-margin status and assets with similar status
type CrossMarginStatusAndAssets struct {
	TotalLiabilityOfQuoteCurrency string              `json:"totalLiabilityOfQuoteCurrency"`
	TotalAssetOfQuoteCurrency     string              `json:"totalAssetOfQuoteCurrency"`
	DebtRatio                     types.Number        `json:"debtRatio"`
	Status                        string              `json:"status"`
	Assets                        []MarginAssetDetail `json:"assets"`
}

// IsolatedMarginAccountDetail represents an isolated-margin account detail
type IsolatedMarginAccountDetail struct {
	TotalAssetOfQuoteCurrency     string                      `json:"totalAssetOfQuoteCurrency"`
	TotalLiabilityOfQuoteCurrency string                      `json:"totalLiabilityOfQuoteCurrency"`
	Timestamp                     types.Time                  `json:"timestamp"`
	Assets                        []IsolatedMarginAssetDetail `json:"assets"`
}

// IsolatedMarginAssetDetail represents an isolated margin asset detail
type IsolatedMarginAssetDetail struct {
	Symbol     string            `json:"symbol"`
	Status     string            `json:"status"`
	DebtRatio  types.Number      `json:"debtRatio"`
	BaseAsset  MarginAssetDetail `json:"baseAsset"`
	QuoteAsset MarginAssetDetail `json:"quoteAsset"`
}

// MarginAssetDetail represents an asset detailed information
type MarginAssetDetail struct {
	Currency        string       `json:"currency"`
	BorrowEnabled   bool         `json:"borrowEnabled"`
	RepayEnabled    bool         `json:"repayEnabled"`
	TransferEnabled bool         `json:"transferEnabled"`
	Borrowed        types.Number `json:"borrowed"`
	TotalAsset      types.Number `json:"totalAsset"`
	Available       types.Number `json:"available"`
	Hold            types.Number `json:"hold"`
	MaxBorrowSize   types.Number `json:"maxBorrowSize"`
	Liability       string       `json:"liability"`
	Total           types.Number `json:"total"`
}

// FuturesAccountOverview represents a futures account detail
type FuturesAccountOverview struct {
	AccountEquity    float64 `json:"accountEquity"`
	UnrealisedPNL    float64 `json:"unrealisedPNL"`
	MarginBalance    float64 `json:"marginBalance"`
	PositionMargin   float64 `json:"positionMargin"`
	OrderMargin      float64 `json:"orderMargin"`
	FrozenFunds      float64 `json:"frozenFunds"`
	AvailableBalance float64 `json:"availableBalance"`
	Currency         string  `json:"currency"`
}

// AccountBalance represents an account balance detail
type AccountBalance struct {
	Currency          string       `json:"currency"`
	Balance           types.Number `json:"balance"`
	Available         types.Number `json:"available"`
	Holds             types.Number `json:"holds"`
	BaseCurrency      string       `json:"baseCurrency"`
	BaseCurrencyPrice types.Number `json:"baseCurrencyPrice"`
	BaseAmount        types.Number `json:"baseAmount"`
}

// SubAccounts represents subacounts detail and balances
type SubAccounts struct {
	SubUserID      string           `json:"subUserId"`
	SubName        string           `json:"subName"`
	MainAccounts   []AccountBalance `json:"mainAccounts"`
	TradeAccounts  []AccountBalance `json:"tradeAccounts"`
	MarginAccounts []AccountBalance `json:"marginAccounts"`
}

// FuturesSubAccountBalance represents a subaccount balances for futures trading
type FuturesSubAccountBalance struct {
	Summary struct {
		AccountEquityTotal    float64 `json:"accountEquityTotal"`
		UnrealisedPNLTotal    float64 `json:"unrealisedPNLTotal"`
		MarginBalanceTotal    float64 `json:"marginBalanceTotal"`
		PositionMarginTotal   float64 `json:"positionMarginTotal"`
		OrderMarginTotal      float64 `json:"orderMarginTotal"`
		FrozenFundsTotal      float64 `json:"frozenFundsTotal"`
		AvailableBalanceTotal float64 `json:"availableBalanceTotal"`
		Currency              string  `json:"currency"`
	} `json:"summary"`
	Accounts []FuturesSubAccountBalanceDetail `json:"accounts"`
}

// FuturesSubAccountBalanceDetail represents a futures sub-account balance detail
type FuturesSubAccountBalanceDetail struct {
	AccountName      string  `json:"accountName"`
	AccountEquity    float64 `json:"accountEquity"`
	UnrealisedPNL    float64 `json:"unrealisedPNL"`
	MarginBalance    float64 `json:"marginBalance"`
	PositionMargin   float64 `json:"positionMargin"`
	OrderMargin      float64 `json:"orderMargin"`
	FrozenFunds      float64 `json:"frozenFunds"`
	AvailableBalance float64 `json:"availableBalance"`
	Currency         string  `json:"currency"`
}

// LedgerInfo represents account ledger information
type LedgerInfo struct {
	ID          string       `json:"id"`
	Currency    string       `json:"currency"`
	Amount      types.Number `json:"amount"`
	Fee         types.Number `json:"fee"`
	Balance     types.Number `json:"balance"`
	AccountType string       `json:"accountType"`
	BizType     string       `json:"bizType"`
	Direction   string       `json:"direction"`
	CreatedAt   types.Time   `json:"createdAt"`
	Context     string       `json:"context"`
}

// FuturesLedgerInfo represents account ledger information for futures trading
type FuturesLedgerInfo struct {
	HasMore  bool                        `json:"hasMore"`
	DataList []FuturesLedgerInfoDataItem `json:"dataList"`
}

// FuturesLedgerInfoDataItem represents a futures ledger info data item
type FuturesLedgerInfoDataItem struct {
	Time          types.Time `json:"time"`
	Type          string     `json:"type"`
	Amount        float64    `json:"amount"`
	Fee           float64    `json:"fee"`
	AccountEquity float64    `json:"accountEquity"`
	Status        string     `json:"status"`
	Remark        string     `json:"remark"`
	Offset        int64      `json:"offset"`
	Currency      string     `json:"currency"`
}

// AccountCurrencyInfo represents main account detailed information
type AccountCurrencyInfo struct {
	AccountInfo
	BaseCurrency      string  `json:"baseCurrency"`
	BaseCurrencyPrice float64 `json:"baseCurrencyPrice,string"`
	BaseAmount        float64 `json:"baseAmount,string"`
}

// AccountSummaryInformation represents account summary information detail
type AccountSummaryInformation struct {
	Level                 float64 `json:"level"`
	SubQuantity           float64 `json:"subQuantity"`
	MaxSubQuantity        float64 `json:"maxSubQuantity"`
	SpotSubQuantity       float64 `json:"spotSubQuantity"`
	MarginSubQuantity     float64 `json:"marginSubQuantity"`
	FuturesSubQuantity    float64 `json:"futuresSubQuantity"`
	MaxSpotSubQuantity    float64 `json:"maxSpotSubQuantity"`
	MaxMarginSubQuantity  float64 `json:"maxMarginSubQuantity"`
	MaxFuturesSubQuantity float64 `json:"maxFuturesSubQuantity"`
	MaxDefaultSubQuantity float64 `json:"maxDefaultSubQuantity"`
}

// SubAccountsResponse represents a sub-accounts items response instance
type SubAccountsResponse struct {
	CurrentPage int64            `json:"currentPage"`
	PageSize    int64            `json:"pageSize"`
	TotalNumber int64            `json:"totalNum"`
	TotalPage   int64            `json:"totalPage"`
	Items       []SubAccountInfo `json:"items"`
}

// SubAccountInfo holds subaccount data for main, spot(trade), and margin accounts
type SubAccountInfo struct {
	SubUserID      string                `json:"subUserId"`
	SubName        string                `json:"subName"`
	MainAccounts   []AccountCurrencyInfo `json:"mainAccounts"`
	TradeAccounts  []AccountCurrencyInfo `json:"tradeAccounts"`
	MarginAccounts []AccountCurrencyInfo `json:"marginAccounts"`
}

// SubAccountsBalanceV2 represents a sub-account balance detail through the V2 API
type SubAccountsBalanceV2 struct {
	CurrentPage int64                     `json:"currentPage"`
	PageSize    int64                     `json:"pageSize"`
	TotalNum    int64                     `json:"totalNum"`
	TotalPage   int64                     `json:"totalPage"`
	Items       []SubAccountBalanceDetail `json:"items"`
}

// SubAccountBalanceDetail represents a sub-account balance detail
type SubAccountBalanceDetail struct {
	SubUserID    string                            `json:"subUserId"`
	SubName      string                            `json:"subName"`
	MainAccounts []SubAccountCurrencyBalanceDetail `json:"mainAccounts"`
}

// SubAccountCurrencyBalanceDetail represents a sub-account currency balance detail
type SubAccountCurrencyBalanceDetail struct {
	Currency          string       `json:"currency"`
	BaseCurrency      string       `json:"baseCurrency"`
	Balance           types.Number `json:"balance"`
	Available         types.Number `json:"available"`
	Holds             types.Number `json:"holds"`
	BaseCurrencyPrice types.Number `json:"baseCurrencyPrice"`
	BaseAmount        types.Number `json:"baseAmount"`
}

// TransferableBalanceInfo represents transferable balance information
type TransferableBalanceInfo struct {
	AccountInfo
	Transferable float64 `json:"transferable,string"`
}

// FundTransferFuturesParam holds parameter values for internal transfer
type FundTransferFuturesParam struct {
	Amount             float64       `json:"amount"`
	Currency           currency.Code `json:"currency"`
	RecieveAccountType string        `json:"recAccountType"` // possible values are: MAIN and TRADE
}

// FundTransferToFuturesParam holds request parameters to transfer funds to futures account
type FundTransferToFuturesParam struct {
	Amount             float64       `json:"amount"`
	Currency           currency.Code `json:"currency"`
	PaymentAccountType string        `json:"payAccountType"` // Payment account type, including MAIN,TRADE
}

// InnerTransferToMainAndTradeResponse represents a detailed response after transferring fund to main and trade accounts
type InnerTransferToMainAndTradeResponse struct {
	ApplyID        string       `json:"applyId"`
	BizNo          string       `json:"bizNo"`
	PayAccountType string       `json:"payAccountType"`
	PayTag         string       `json:"payTag"`
	Remark         string       `json:"remark"`
	RecAccountType string       `json:"recAccountType"`
	RecTag         string       `json:"recTag"`
	RecRemark      string       `json:"recRemark"`
	RecSystem      string       `json:"recSystem"`
	Status         string       `json:"status"`
	Currency       string       `json:"currency"`
	Amount         types.Number `json:"amount"`
	Fee            types.Number `json:"fee"`
	SerialNumber   int64        `json:"sn"`
	Reason         string       `json:"reason"`
	CreatedAt      types.Time   `json:"createdAt"`
	UpdatedAt      types.Time   `json:"updatedAt"`
}

// FundTransferToFuturesResponse response struct after transferring fund to Futures account
type FundTransferToFuturesResponse struct {
	Code    string `json:"code"`
	Message string `json:"msg"`
	Retry   bool   `json:"retry"`
	Success bool   `json:"success"`
}

// FuturesTransferOutResponse represents a list of transfer out instance
type FuturesTransferOutResponse struct {
	CurrentPage int64              `json:"currentPage"`
	PageSize    int64              `json:"pageSize"`
	TotalNum    int64              `json:"totalNum"`
	TotalPage   int64              `json:"totalPage"`
	Items       []FundTransferInfo `json:"items"`
}

// FundTransferInfo represents a fund transfer instance information
type FundTransferInfo struct {
	ApplyID   string       `json:"applyId"`
	Currency  string       `json:"currency"`
	RecRemark string       `json:"recRemark"`
	RecSystem string       `json:"recSystem"`
	Status    string       `json:"status"`
	Reason    string       `json:"reason"`
	Offset    int64        `json:"offset"`
	Remark    string       `json:"remark"`
	Amount    types.Number `json:"amount"`
	CreatedAt types.Time   `json:"createdAt"`
}

// DepositAddressParams represents a deposit address creation parameters
type DepositAddressParams struct {
	Currency currency.Code `json:"currency"`
	Chain    string        `json:"chain,omitempty"`
}

// DepositAddress represents deposit address information for Spot and Margin trading
type DepositAddress struct {
	Address string `json:"address"`
	Memo    string `json:"memo"`
	Chain   string `json:"chain"`

	// TODO: to be removed if not used by other endpoints
	ContractAddress string `json:"contractAddress"` // missing in case of futures
}

type baseDeposit struct {
	Currency   string `json:"currency"`
	WalletTxID string `json:"walletTxId"`
	IsInner    bool   `json:"isInner"`
	Status     string `json:"status"`
}

// DepositResponse represents a detailed response for list of deposit
type DepositResponse struct {
	CurrentPage int64     `json:"currentPage"`
	PageSize    int64     `json:"pageSize"`
	TotalNum    int64     `json:"totalNum"`
	TotalPage   int64     `json:"totalPage"`
	Items       []Deposit `json:"items"`
}

// Deposit represents deposit address and detail and timestamp information
type Deposit struct {
	baseDeposit
	Amount    float64    `json:"amount,string"`
	Address   string     `json:"address"`
	Memo      string     `json:"memo"`
	Fee       float64    `json:"fee,string"`
	Remark    string     `json:"remark"`
	CreatedAt types.Time `json:"createdAt"`
	UpdatedAt types.Time `json:"updatedAt"`
	Chain     string     `json:"chain"`
}

// HistoricalDepositWithdrawalResponse represents deposit and withdrawal funding items details
type HistoricalDepositWithdrawalResponse struct {
	CurrentPage int64                         `json:"currentPage"`
	PageSize    int64                         `json:"pageSize"`
	TotalNum    int64                         `json:"totalNum"`
	TotalPage   int64                         `json:"totalPage"`
	Items       []HistoricalDepositWithdrawal `json:"items"`
}

// HistoricalDepositWithdrawal represents deposit and withdrawal funding item
type HistoricalDepositWithdrawal struct {
	baseDeposit
	Amount    float64    `json:"amount,string"`
	CreatedAt types.Time `json:"createAt"`
}

// WithdrawalsResponse represents a withdrawals list of items details
type WithdrawalsResponse struct {
	CurrentPage int64        `json:"currentPage"`
	PageSize    int64        `json:"pageSize"`
	TotalNum    int64        `json:"totalNum"`
	TotalPage   int64        `json:"totalPage"`
	Items       []Withdrawal `json:"items"`
}

// Withdrawal represents withdrawal funding information
type Withdrawal struct {
	Deposit
	ID string `json:"id"`
}

// WithdrawalQuota represents withdrawal quota detail information
type WithdrawalQuota struct {
	Currency                 string       `json:"currency"`
	LimitBTCAmount           float64      `json:"limitBTCAmount,string"`
	UsedBTCAmount            float64      `json:"usedBTCAmount,string"`
	RemainAmount             float64      `json:"remainAmount,string"`
	AvailableAmount          float64      `json:"availableAmount,string"`
	WithdrawMinFee           float64      `json:"withdrawMinFee,string"`
	InnerWithdrawMinFee      float64      `json:"innerWithdrawMinFee,string"`
	WithdrawMinSize          float64      `json:"withdrawMinSize,string"`
	IsWithdrawEnabled        bool         `json:"isWithdrawEnabled"`
	Precision                int64        `json:"precision"`
	Chain                    string       `json:"chain"`
	QuotaCurrency            string       `json:"quotaCurrency"`
	LimitQuotaCurrencyAmount types.Number `json:"limitQuotaCurrencyAmount"`
	Reason                   string       `json:"reason"`
	UsedQuotaCurrencyAmount  string       `json:"usedQuotaCurrencyAmount"`
}

// Fees represents taker and maker fee information a symbol
type Fees struct {
	Symbol       string  `json:"symbol"`
	TakerFeeRate float64 `json:"takerFeeRate,string"`
	MakerFeeRate float64 `json:"makerFeeRate,string"`
}

// LendingCurrencyInfo represents a lending currency information
type LendingCurrencyInfo struct {
	Currency           string       `json:"currency"`
	PurchaseEnable     bool         `json:"purchaseEnable"`
	RedeemEnable       bool         `json:"redeemEnable"`
	Increment          string       `json:"increment"`
	MinPurchaseSize    types.Number `json:"minPurchaseSize"`
	MinInterestRate    types.Number `json:"minInterestRate"`
	MaxInterestRate    types.Number `json:"maxInterestRate"`
	InterestIncrement  string       `json:"interestIncrement"`
	MaxPurchaseSize    types.Number `json:"maxPurchaseSize"`
	MarketInterestRate types.Number `json:"marketInterestRate"`
	AutoPurchaseEnable bool         `json:"autoPurchaseEnable"`
}

// InterestRate represents a currency interest rate
type InterestRate struct {
	Time               types.Time   `json:"time"`
	MarketInterestRate types.Number `json:"marketInterestRate"`
}

// OrderNumberResponse represents a response for margin trading lending and redemption
type OrderNumberResponse struct {
	OrderNo string `json:"orderNo"`
}

// ModifySubscriptionOrderResponse represents a modify subscription order response
type ModifySubscriptionOrderResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Msg     string `json:"msg"`
	Retry   bool   `json:"retry"`
}

// RedemptionOrdersResponse represents a response for querying list of redemption orders
type RedemptionOrdersResponse struct {
	CurrentPage int64             `json:"currentPage"`
	PageSize    int64             `json:"pageSize"`
	TotalNum    int64             `json:"totalNum"`
	TotalPage   int64             `json:"totalPage"`
	Items       []RedemptionOrder `json:"items"`
}

// RedemptionOrder represents a single redemption order instance
type RedemptionOrder struct {
	Currency        string       `json:"currency"`
	PurchaseOrderNo string       `json:"purchaseOrderNo"`
	RedeemOrderNo   string       `json:"redeemOrderNo"`
	RedeemAmount    types.Number `json:"redeemAmount"`
	ReceiptAmount   types.Number `json:"receiptAmount"`
	ApplyTime       types.Time   `json:"applyTime"`
	Status          string       `json:"status"`
}

// PurchaseSubscriptionOrdersResponse represents list of purchase subscription orders response
type PurchaseSubscriptionOrdersResponse struct {
	CurrentPage int64                              `json:"currentPage"`
	PageSize    int64                              `json:"pageSize"`
	TotalNum    int64                              `json:"totalNum"`
	TotalPage   int64                              `json:"totalPage"`
	Items       []PurchaseSubscriptionResponseItem `json:"items"`
}

// PurchaseSubscriptionResponseItem represents a purchase order subscription response single item
type PurchaseSubscriptionResponseItem struct {
	Currency        string       `json:"currency"`
	PurchaseOrderNo string       `json:"purchaseOrderNo"`
	PurchaseAmount  types.Number `json:"purchaseAmount"`
	MatchSize       types.Number `json:"matchSize"`
	RedeemSize      types.Number `json:"redeemSize"`
	RedeemAmount    types.Number `json:"redeemAmount"`
	LendAmount      types.Number `json:"lendAmount"`
	InterestRate    types.Number `json:"interestRate"`
	IncomeAmount    types.Number `json:"incomeAmount"`
	IncomeSize      types.Number `json:"incomeSize"`
	ApplyTime       types.Time   `json:"applyTime"`
	Status          string       `json:"status"`
}

// WSInstanceServers response connection token and websocket instance server information
type WSInstanceServers struct {
	Token           string           `json:"token"`
	InstanceServers []InstanceServer `json:"instanceServers"`
}

// InstanceServer represents a single websocket instance server information
type InstanceServer struct {
	Endpoint     string `json:"endpoint"`
	Encrypt      bool   `json:"encrypt"`
	Protocol     string `json:"protocol"`
	PingInterval int64  `json:"pingInterval"`
	PingTimeout  int64  `json:"pingTimeout"`
}

// WSConnMessages represents response messages ping, pong, and welcome message structures
type WSConnMessages struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// WsSubscriptionInput represents a subscription information structure
type WsSubscriptionInput struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Topic          string `json:"topic"`
	PrivateChannel bool   `json:"privateChannel"`
	Response       bool   `json:"response,omitempty"`
}

// WsPushData represents a push data from a server
type WsPushData struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Topic       string          `json:"topic"`
	UserID      string          `json:"userId"`
	Subject     string          `json:"subject"`
	ChannelType string          `json:"channelType"`
	Data        json.RawMessage `json:"data"`
}

// WsTicker represents a ticker push data from server
type WsTicker struct {
	Sequence    string     `json:"sequence"`
	BestAsk     float64    `json:"bestAsk,string"`
	Size        float64    `json:"size,string"`
	BestBidSize float64    `json:"bestBidSize,string"`
	Price       float64    `json:"price,string"`
	BestAskSize float64    `json:"bestAskSize,string"`
	BestBid     float64    `json:"bestBid,string"`
	Timestamp   types.Time `json:"time"`
}

// WsSnapshot represents a spot ticker push data
type WsSnapshot struct {
	Sequence types.Number     `json:"sequence"`
	Data     WsSnapshotDetail `json:"data"`
}

// WsSnapshotDetail represents the detail of a spot ticker data
// This represents all websocket ticker information pushed as a result of subscription to /market/snapshot:{symbol}, and /market/snapshot:{currency,market}
type WsSnapshotDetail struct {
	AveragePrice     float64    `json:"averagePrice"`
	BaseCurrency     string     `json:"baseCurrency"`
	Board            int64      `json:"board"`
	Buy              float64    `json:"buy"`
	ChangePrice      float64    `json:"changePrice"`
	ChangeRate       float64    `json:"changeRate"`
	Close            float64    `json:"close"`
	Datetime         types.Time `json:"datetime"`
	High             float64    `json:"high"`
	LastTradedPrice  float64    `json:"lastTradedPrice"`
	Low              float64    `json:"low"`
	MakerCoefficient float64    `json:"makerCoefficient"`
	MakerFeeRate     float64    `json:"makerFeeRate"`
	MarginTrade      bool       `json:"marginTrade"`
	Mark             float64    `json:"mark"`
	Market           string     `json:"market"`
	Markets          []string   `json:"markets"`
	Open             float64    `json:"open"`
	QuoteCurrency    string     `json:"quoteCurrency"`
	Sell             float64    `json:"sell"`
	Sort             int64      `json:"sort"`
	Symbol           string     `json:"symbol"`
	SymbolCode       string     `json:"symbolCode"`
	TakerCoefficient float64    `json:"takerCoefficient"`
	TakerFeeRate     float64    `json:"takerFeeRate"`
	Trading          bool       `json:"trading"`
	Vol              float64    `json:"vol"`
	VolValue         float64    `json:"volValue"`
}

// WsOrderbook represents orderbook information
type WsOrderbook struct {
	Changes       OrderbookChanges `json:"changes"`
	SequenceEnd   int64            `json:"sequenceEnd"`
	SequenceStart int64            `json:"sequenceStart"`
	Symbol        string           `json:"symbol"`
	TimeMS        types.Time       `json:"time"`
}

// OrderbookChanges represents orderbook ask and bid changes
type OrderbookChanges struct {
	Asks [][]types.Number `json:"asks"`
	Bids [][]types.Number `json:"bids"`
}

// WsCandlestick represents candlestick information push data for a symbol
type WsCandlestick struct {
	Symbol  string       `json:"symbol"`
	Candles wsCandleItem `json:"candles"`
	Time    types.Time   `json:"time"`
}

type wsCandleItem struct {
	StartTime         types.Time
	OpenPrice         types.Number
	ClosePrice        types.Number
	HighPrice         types.Number
	LowPrice          types.Number
	TransactionVolume types.Number
	TransactionAmount types.Number
}

// UnmarshalJSON unmarshals data into a wsCandleItem
func (a *wsCandleItem) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&a.StartTime, &a.OpenPrice, &a.ClosePrice, &a.HighPrice, &a.LowPrice, &a.TransactionVolume, &a.TransactionAmount})
}

// WsTrade represents a trade push data
type WsTrade struct {
	Sequence     string     `json:"sequence"`
	Type         string     `json:"type"`
	Symbol       string     `json:"symbol"`
	Side         string     `json:"side"`
	Price        float64    `json:"price,string"`
	Size         float64    `json:"size,string"`
	TradeID      string     `json:"tradeId"`
	TakerOrderID string     `json:"takerOrderId"`
	MakerOrderID string     `json:"makerOrderId"`
	Time         types.Time `json:"time"`
}

// WsPriceIndicator represents index price or mark price indicator push data
type WsPriceIndicator struct {
	Symbol      string     `json:"symbol"`
	Granularity float64    `json:"granularity"`
	Timestamp   types.Time `json:"timestamp"`
	Value       float64    `json:"value"`
}

// WsTradeOrder represents a private trade order push data
type WsTradeOrder struct {
	Symbol     string       `json:"symbol"`
	OrderType  string       `json:"orderType"`
	Side       string       `json:"side"`
	OrderID    string       `json:"orderId"`
	Type       string       `json:"type"`
	OrderTime  types.Time   `json:"orderTime"`
	Size       float64      `json:"size,string"`
	FilledSize float64      `json:"filledSize,string"`
	Price      float64      `json:"price,string"`
	ClientOid  string       `json:"clientOid"`
	RemainSize float64      `json:"remainSize,string"`
	Status     string       `json:"status"`
	Timestamp  types.Time   `json:"ts"`
	Liquidity  string       `json:"liquidity"`
	MatchPrice types.Number `json:"matchPrice"`
	MatchSize  types.Number `json:"matchSize"`
	OldSize    types.Number `json:"oldSize"`
	TradeID    string       `json:"tradeId"`
}

// WsAccountBalance represents a Account Balance push data
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
	Time types.Time `json:"time"`
}

// WsDebtRatioChange represents a push data
type WsDebtRatioChange struct {
	DebtRatio float64           `json:"debtRatio"`
	TotalDebt float64           `json:"totalDebt,string"`
	DebtList  map[string]string `json:"debtList"`
	Timestamp types.Time        `json:"timestamp"`
}

// WsPositionStatus represents a position status push data
type WsPositionStatus struct {
	Type        string     `json:"type"`
	TimestampMS types.Time `json:"timestamp"`
}

// WsMarginTradeOrderEntersEvent represents a push data to the lenders
// when the order enters the order book or when the order is executed
type WsMarginTradeOrderEntersEvent struct {
	Currency     string     `json:"currency"`
	OrderID      string     `json:"orderId"`      // Trade ID
	DailyIntRate float64    `json:"dailyIntRate"` // Daily interest rate
	Term         int64      `json:"term"`         // Term (Unit: Day)
	Size         float64    `json:"size"`         // Size
	LentSize     float64    `json:"lentSize"`     // Size executed -- filled when the subject is order.update
	Side         string     `json:"side"`         // Lend or borrow. Currently, only "Lend" is available
	Timestamp    types.Time `json:"ts"`           // Timestamp (nanosecond)
}

// WsMarginTradeOrderDoneEvent represents a push message to the lenders when the order is completed
type WsMarginTradeOrderDoneEvent struct {
	Currency  string     `json:"currency"`
	OrderID   string     `json:"orderId"`
	Reason    string     `json:"reason"`
	Side      string     `json:"side"`
	Timestamp types.Time `json:"ts"`
}

// WsFuturesKline represents a futures kline data
type WsFuturesKline struct {
	Symbol  string          `json:"symbol"`
	Candles [7]types.Number `json:"candles"` // Start Time, Open Price, Close Price, High Price, Low Price, Transaction Volume, and Transaction Amount respectively
	Time    types.Time      `json:"time"`
}

// WsStopOrder represents a stop order
// When a stop order is received by the system, you will receive a message with "open" type
// It means that this order entered the system and waited to be triggered
type WsStopOrder struct {
	CreatedAt      types.Time `json:"createdAt"`
	OrderID        string     `json:"orderId"`
	OrderPrice     float64    `json:"orderPrice,string"`
	OrderType      string     `json:"orderType"`
	Side           string     `json:"side"`
	Size           float64    `json:"size,string"`
	Stop           string     `json:"stop"`
	StopPrice      float64    `json:"stopPrice,string"`
	Symbol         string     `json:"symbol"`
	TradeType      string     `json:"tradeType"`
	TriggerSuccess bool       `json:"triggerSuccess"`
	Timestamp      types.Time `json:"ts"`
	Type           string     `json:"type"`
}

// WsFuturesTicker represents a futures ticker push data
type WsFuturesTicker struct {
	Symbol       string       `json:"symbol"`
	Sequence     int64        `json:"sequence"`
	Side         string       `json:"side"`
	FilledPrice  types.Number `json:"price"`
	FilledSize   types.Number `json:"size"`
	TradeID      string       `json:"tradeId"`
	BestBidSize  types.Number `json:"bestBidSize"`
	BestBidPrice types.Number `json:"bestBidPrice"`
	BestAskPrice types.Number `json:"bestAskPrice"`
	BestAskSize  types.Number `json:"bestAskSize"`
	FilledTime   types.Time   `json:"ts"`
}

// WsFuturesOrderbookInfo represents Level 2 order book information
type WsFuturesOrderbookInfo struct {
	Sequence  int64      `json:"sequence"`
	Change    string     `json:"change"`
	Timestamp types.Time `json:"timestamp"`
}

// WsFuturesExecutionData represents execution data for symbol
type WsFuturesExecutionData struct {
	Sequence         int64      `json:"sequence"`
	FilledQuantity   float64    `json:"matchSize"` // Filled quantity
	UnfilledQuantity float64    `json:"size"`
	FilledPrice      float64    `json:"price"`
	TradeID          string     `json:"tradeId"`
	MakerUserID      string     `json:"makerUserId"`
	Symbol           string     `json:"symbol"`
	Side             string     `json:"side"`
	TakerOrderID     string     `json:"takerOrderId"`
	MakerOrderID     string     `json:"makerOrderId"`
	TakerUserID      string     `json:"takerUserId"`
	Timestamp        types.Time `json:"ts"`
}

// WsOrderbookLevel5 represents an orderbook push data with depth level 5
type WsOrderbookLevel5 struct {
	Sequence      int64             `json:"sequence"`
	Asks          []orderbook.Level `json:"asks"`
	Bids          []orderbook.Level `json:"bids"`
	PushTimestamp types.Time        `json:"ts"`
	Timestamp     types.Time        `json:"timestamp"`
}

// WsOrderbookLevel5Response represents a response data for an orderbook push data with depth level 5
type WsOrderbookLevel5Response struct {
	Sequence      int64             `json:"sequence"`
	Bids          [][2]types.Number `json:"bids"`
	Asks          [][2]types.Number `json:"asks"`
	PushTimestamp types.Time        `json:"ts"`
	Timestamp     types.Time        `json:"timestamp"`
}

// ExtractOrderbookItems returns WsOrderbookLevel5 instance from WsOrderbookLevel5Response
func (a *WsOrderbookLevel5Response) ExtractOrderbookItems() *WsOrderbookLevel5 {
	resp := WsOrderbookLevel5{
		Timestamp:     a.Timestamp,
		Sequence:      a.Sequence,
		PushTimestamp: a.PushTimestamp,
	}
	resp.Asks = make([]orderbook.Level, len(a.Asks))
	for x := range a.Asks {
		resp.Asks[x] = orderbook.Level{
			Price:  a.Asks[x][0].Float64(),
			Amount: a.Asks[x][1].Float64(),
		}
	}
	resp.Bids = make([]orderbook.Level, len(a.Bids))
	for x := range a.Bids {
		resp.Bids[x] = orderbook.Level{
			Price:  a.Bids[x][0].Float64(),
			Amount: a.Bids[x][1].Float64(),
		}
	}
	return &resp
}

// WsFundingRate represents the funding rate push data information through the websocket channel
type WsFundingRate struct {
	Symbol      string     `json:"symbol"`
	Granularity int64      `json:"granularity"`
	FundingRate float64    `json:"fundingRate"`
	Timestamp   types.Time `json:"timestamp"`
}

// WsFuturesMarkPriceAndIndexPrice represents mark price and index price information
type WsFuturesMarkPriceAndIndexPrice struct {
	Symbol      string     `json:"symbol"`
	Granularity int64      `json:"granularity"`
	IndexPrice  float64    `json:"indexPrice"`
	MarkPrice   float64    `json:"markPrice"`
	Timestamp   types.Time `json:"timestamp"`
}

// WsFuturesFundingBegin represents the Start Funding Fee Settlement
type WsFuturesFundingBegin struct {
	Subject     string     `json:"subject"`
	Symbol      string     `json:"symbol"`
	FundingTime types.Time `json:"fundingTime"`
	FundingRate float64    `json:"fundingRate"`
	Timestamp   types.Time `json:"timestamp"`
}

// WsFuturesTransactionStatisticsTimeEvent represents transaction statistics data
type WsFuturesTransactionStatisticsTimeEvent struct {
	Symbol                   string     `json:"symbol"`
	Volume24H                float64    `json:"volume"`
	Turnover24H              float64    `json:"turnover"`
	LastPrice                int64      `json:"lastPrice"`
	PriceChangePercentage24H float64    `json:"priceChgPct"`
	SnapshotTime             types.Time `json:"ts"`
}

// WsFuturesTradeOrder represents trade order information according to the market
type WsFuturesTradeOrder struct {
	OrderID          string     `json:"orderId"`
	Symbol           string     `json:"symbol"`
	Type             string     `json:"type"`       // Message Type: "open", "match", "filled", "canceled", "update"
	Status           string     `json:"status"`     // Order Status: "match", "open", "done"
	MatchSize        string     `json:"matchSize"`  // Match Size (when the type is "match")
	MatchPrice       string     `json:"matchPrice"` // Match Price (when the type is "match")
	OrderType        string     `json:"orderType"`  // Order Type, "market" indicates market order, "limit" indicates limit order
	Side             string     `json:"side"`       // Trading direction,include buy and sell
	OrderPrice       float64    `json:"price,string"`
	OrderSize        float64    `json:"size,string"`
	RemainSize       float64    `json:"remainSize,string"`
	FilledSize       float64    `json:"filledSize,string"`   // Remaining Size for Trading
	CanceledSize     float64    `json:"canceledSize,string"` // In the update message, the Size of order reduced
	TradeID          string     `json:"tradeId"`             // Trade ID (when the type is "match")
	ClientOid        string     `json:"clientOid"`           // Client supplied order id
	OrderTime        types.Time `json:"orderTime"`
	OldSize          string     `json:"oldSize "`  // Size Before Update (when the type is "update")
	TradingDirection string     `json:"liquidity"` // Liquidity, Trading direction, buy or sell in taker
	Timestamp        types.Time `json:"ts"`
}

// WsStopOrderLifecycleEvent represents futures stop order lifecycle event
type WsStopOrderLifecycleEvent struct {
	OrderID        string     `json:"orderId"`
	Symbol         string     `json:"symbol"`
	Type           string     `json:"type"`
	OrderType      string     `json:"orderType"`
	Side           string     `json:"side"`
	Size           float64    `json:"size,string"`
	OrderPrice     float64    `json:"orderPrice,string"`
	Stop           string     `json:"stop"`
	StopPrice      float64    `json:"stopPrice,string"`
	StopPriceType  string     `json:"stopPriceType"`
	TriggerSuccess bool       `json:"triggerSuccess"`
	Error          string     `json:"error"`
	CreatedAt      types.Time `json:"createdAt"`
	Timestamp      types.Time `json:"ts"`
}

// WsFuturesOrderMarginEvent represents an order margin account balance event
type WsFuturesOrderMarginEvent struct {
	OrderMargin float64    `json:"orderMargin"`
	Currency    string     `json:"currency"`
	Timestamp   types.Time `json:"timestamp"`
}

// WsFuturesAvailableBalance represents an available balance push data for futures account
type WsFuturesAvailableBalance struct {
	AvailableBalance float64    `json:"availableBalance"`
	HoldBalance      float64    `json:"holdBalance"`
	Currency         string     `json:"currency"`
	Timestamp        types.Time `json:"timestamp"`
}

// WsFuturesWithdrawalAmountAndTransferOutAmountEvent represents Withdrawal Amount & Transfer-Out Amount Event push data
type WsFuturesWithdrawalAmountAndTransferOutAmountEvent struct {
	WithdrawHold float64    `json:"withdrawHold"` // Current frozen amount for withdrawal
	Currency     string     `json:"currency"`
	Timestamp    types.Time `json:"timestamp"`
}

// WsFuturesPosition represents futures account position change event
type WsFuturesPosition struct {
	RealisedGrossPnl  float64    `json:"realisedGrossPnl"` // Accumulated realised profit and loss
	Symbol            string     `json:"symbol"`
	CrossMode         bool       `json:"crossMode"`        // Cross mode or not
	LiquidationPrice  float64    `json:"liquidationPrice"` // Liquidation price
	PosLoss           float64    `json:"posLoss"`          // Manually added margin amount
	AvgEntryPrice     float64    `json:"avgEntryPrice"`    // Average entry price
	UnrealisedPnl     float64    `json:"unrealisedPnl"`    // Unrealised profit and loss
	MarkPrice         float64    `json:"markPrice"`        // Mark price
	PosMargin         float64    `json:"posMargin"`        // Position margin
	AutoDeposit       bool       `json:"autoDeposit"`      // Auto deposit margin or not
	RiskLimit         float64    `json:"riskLimit"`
	UnrealisedCost    float64    `json:"unrealisedCost"`    // Unrealised value
	PosComm           float64    `json:"posComm"`           // Bankruptcy cost
	PosMaint          float64    `json:"posMaint"`          // Maintenance margin
	PosCost           float64    `json:"posCost"`           // Position value
	MaintMarginReq    float64    `json:"maintMarginReq"`    // Maintenance margin rate
	BankruptPrice     float64    `json:"bankruptPrice"`     // Bankruptcy price
	RealisedCost      float64    `json:"realisedCost"`      // Currently accumulated realised position value
	MarkValue         float64    `json:"markValue"`         // Mark value
	PosInit           float64    `json:"posInit"`           // Position margin
	RealisedPnl       float64    `json:"realisedPnl"`       // Realised profit and loss
	MaintMargin       float64    `json:"maintMargin"`       // Position margin
	RealLeverage      float64    `json:"realLeverage"`      // Leverage of the order
	ChangeReason      string     `json:"changeReason"`      // changeReason:marginChange、positionChange、liquidation、autoAppendMarginStatusChange、adl
	CurrentCost       float64    `json:"currentCost"`       // Current position value
	OpeningTimestamp  types.Time `json:"openingTimestamp"`  // Open time
	CurrentQty        float64    `json:"currentQty"`        // Current position
	DelevPercentage   float64    `json:"delevPercentage"`   // ADL ranking percentile
	CurrentComm       float64    `json:"currentComm"`       // Current commission
	RealisedGrossCost float64    `json:"realisedGrossCost"` // Accumulated realised gross profit value
	IsOpen            bool       `json:"isOpen"`            // Opened position or not
	PosCross          float64    `json:"posCross"`          // Manually added margin
	CurrentTimestamp  types.Time `json:"currentTimestamp"`  // Current timestamp
	UnrealisedRoePcnt float64    `json:"unrealisedRoePcnt"` // Rate of return on investment
	UnrealisedPnlPcnt float64    `json:"unrealisedPnlPcnt"` // Position profit and loss ratio
	SettleCurrency    string     `json:"settleCurrency"`    // Currency used to clear and settle the trades
}

// WsFuturesMarkPricePositionChanges represents futures account position change caused by mark price
type WsFuturesMarkPricePositionChanges struct {
	MarkPrice         float64    `json:"markPrice"`         // Mark price
	MarkValue         float64    `json:"markValue"`         // Mark value
	MaintMargin       float64    `json:"maintMargin"`       // Position margin
	RealLeverage      float64    `json:"realLeverage"`      // Leverage of the order
	UnrealisedPnl     float64    `json:"unrealisedPnl"`     // Unrealised profit and lost
	UnrealisedRoePcnt float64    `json:"unrealisedRoePcnt"` // Rate of return on investment
	UnrealisedPnlPcnt float64    `json:"unrealisedPnlPcnt"` // Position profit and loss ratio
	DelevPercentage   float64    `json:"delevPercentage"`   // ADL ranking percentile
	CurrentTimestamp  types.Time `json:"currentTimestamp"`  // Current timestamp
	SettleCurrency    string     `json:"settleCurrency"`    // Currency used to clear and settle the trades
}

// WsFuturesPositionFundingSettlement represents futures account position funding settlement push data
type WsFuturesPositionFundingSettlement struct {
	PositionSize     float64    `json:"qty"`
	MarkPrice        float64    `json:"markPrice"`
	FundingRate      float64    `json:"fundingRate"`
	FundingFee       float64    `json:"fundingFee"`
	FundingTime      types.Time `json:"fundingTime"`
	CurrentTimestamp types.Time `json:"ts"`
	SettleCurrency   string     `json:"settleCurrency"`
}

// IsolatedMarginBorrowing represents response data for initiating isolated margin borrowing
type IsolatedMarginBorrowing struct {
	OrderID    string  `json:"orderId"`
	Currency   string  `json:"currency"`
	ActualSize float64 `json:"actualSize,string"`
}

// Response represents response model and implements UnmarshalTo interface
type Response struct {
	Data any `json:"data"`
	Error
}

// CancelOrderResponse represents cancel order response model
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
	Permission     string `json:"permission,omitempty"`    // Permissions(Only General、Spot、Futures、Margin permissions can be set, such as "General, Trade". The default is "General")
	IPWhitelist    string `json:"ipWhitelist,omitempty"`   // IP whitelist(You may add up to 20 IPs. Use a halfwidth comma to each IP)
	Expire         int64  `json:"expire,string,omitempty"` // API expiration time; Never expire(default)-1，30Day30，90Day90，180Day180，360Day360

	// used when modifying sub-account API key
	APIKey string `json:"apiKey"`
}

// SubAccountResponse represents the sub-user detail
type SubAccountResponse struct {
	CurrentPage int64        `json:"currentPage"`
	PageSize    int64        `json:"pageSize"`
	TotalNum    int64        `json:"totalNum"`
	TotalPage   int64        `json:"totalPage"`
	Items       []SubAccount `json:"items"`
}

// SubAccount represents sub-user
type SubAccount struct {
	UserID    string     `json:"userId"`
	SubName   string     `json:"subName"`
	Type      int64      `json:"type"` // type:1-robot  or type:0-nomal
	Remarks   string     `json:"remarks"`
	UID       int64      `json:"uid"`
	Status    int64      `json:"status"`
	Access    string     `json:"access"`
	CreatedAt types.Time `json:"createdAt"`
}

// SubAccountV2Response represents a sub-account detailed response of V2
type SubAccountV2Response struct {
	CurrentPage int64        `json:"currentPage"`
	PageSize    int64        `json:"pageSize"`
	TotalNum    int64        `json:"totalNum"`
	TotalPage   int64        `json:"totalPage"`
	Items       []SubAccount `json:"items"`
}

// SubAccountCreatedResponse represents the sub-account response
type SubAccountCreatedResponse struct {
	UID     int64  `json:"uid"`
	SubName string `json:"subName"`
	Remarks string `json:"remarks"`
	Access  string `json:"access"`
}

// SpotAPISubAccount represents a Spot APIs for sub-accounts
type SpotAPISubAccount struct {
	APIKey      string `json:"apiKey"`
	IPWhitelist string `json:"ipWhitelist"`
	Permission  string `json:"permission"`
	SubName     string `json:"subName"`

	Remark    string     `json:"remark"`
	CreatedAt types.Time `json:"createdAt"`

	// TODO: to be removed if not used any other place
	APISecret  string `json:"apiSecret"`
	Passphrase string `json:"passphrase"`
	APIVersion string `json:"apiVersion"`
}

// DeleteSubAccountResponse represents delete sub-account response
type DeleteSubAccountResponse struct {
	SubAccountName string `json:"subName"`
	APIKey         string `json:"apiKey"`
}

// ConnectionMessage represents a connection and subscription status message
type ConnectionMessage struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// TickersResponse represents list of tickers and update timestamp information
type TickersResponse struct {
	Time    types.Time   `json:"time"`
	Tickers []TickerInfo `json:"ticker"`
}

// FundingInterestRateResponse represents a funding interest rate list response information
type FundingInterestRateResponse struct {
	List    []FuturesInterestRate `json:"dataList"`
	HasMore bool                  `json:"hasMore"`
}

// FuturesIndexResponse represents a response data for futures indexes
type FuturesIndexResponse struct {
	List    []FuturesIndex `json:"dataList"`
	HasMore bool           `json:"hasMore"`
}

// FuturesInterestRateResponse represents a futures interest rate list response
type FuturesInterestRateResponse struct {
	List    []FuturesInterestRate `json:"dataList"`
	HasMore bool                  `json:"hasMore"`
}

// TransactionVolume represents a 24 hour transaction volume
type TransactionVolume struct {
	TurnoverOf24Hr float64 `json:"turnoverOf24h"`
}

// FuturesTransactionHistoryResponse represents a futures transaction history response
type FuturesTransactionHistoryResponse struct {
	List    []FuturesTransactionHistory `json:"dataList"`
	HasMore bool                        `json:"hasMore"`
}

// FuturesFundingHistoryResponse represents funding history response for futures account
type FuturesFundingHistoryResponse struct {
	DataList []FuturesFundingHistory `json:"dataList"`
	HasMore  bool                    `json:"hasMore"`
}

// FuturesOrderParam represents a query parameter for placing future oorder
type FuturesOrderParam struct {
	ClientOrderID string        `json:"clientOid"`
	Side          string        `json:"side"`
	Symbol        currency.Pair `json:"symbol"`
	Leverage      float64       `json:"leverage,string"`

	Size  float64 `json:"size,omitempty,string"`
	Price float64 `json:"price,string,omitempty"`

	OrderType           string  `json:"type"`
	Remark              string  `json:"remark,omitempty"`
	Stop                string  `json:"stop,omitempty"`          // Either down or up. Requires stopPrice and stopPriceType to be defined
	StopPriceType       string  `json:"stopPriceType,omitempty"` // [optional] Either TP, IP or MP, Need to be defined if stop is specified. `TP` for trade price, `MP` for Mark price, and "IP" for index price
	StopPrice           float64 `json:"stopPrice,omitempty,string"`
	ReduceOnly          bool    `json:"reduceOnly,omitempty"`
	CloseOrder          bool    `json:"closeOrder,omitempty"`
	ForceHold           bool    `json:"forceHold,omitempty"`
	SelfTradePrevention string  `json:"stp,omitempty"` // self trade prevention, CN, CO, CB. Not supported DC at the moment
	TimeInForce         string  `json:"timeInForce,omitempty"`
	VisibleSize         float64 `json:"visibleSize,omitempty,string"` // The maximum visible size of an iceberg order
	PostOnly            bool    `json:"postOnly,omitempty"`
	Hidden              bool    `json:"hidden,omitempty"`
	Iceberg             bool    `json:"iceberg,omitempty"`
}

// FuturesOrderRespItem represents a single futures order placing response in placing multiple orders
type FuturesOrderRespItem struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
	Symbol        string `json:"symbol"`
	Code          string `json:"code"`
	Msg           string `json:"msg"`
}

// SpotOrderParam represents the spot place order request parameters
type SpotOrderParam struct {
	ClientOrderID       string        `json:"clientOid"`
	Side                string        `json:"side"`
	Symbol              currency.Pair `json:"symbol"`
	OrderType           string        `json:"type,omitempty"`
	TradeType           string        `json:"tradeType,omitempty"` // [Optional] The type of trading : TRADE（Spot Trade）, MARGIN_TRADE (Margin Trade). Default is TRADE
	Remark              string        `json:"remark,omitempty"`
	SelfTradePrevention string        `json:"stp,omitempty"`         // [Optional] self trade prevention , CN, CO, CB or DC. `CN` for Cancel newest, `DC` for Decrease and Cancel, `CO` for cancel oldest, and `CB` for Cancel both
	TimeInForce         string        `json:"timeInForce,omitempty"` // [Optional] GTC, GTT, IOC, or FOK (default is GTC)
	PostOnly            bool          `json:"postOnly,omitempty"`
	Hidden              bool          `json:"hidden,omitempty"`
	Iceberg             bool          `json:"iceberg,omitempty"`
	ReduceOnly          bool          `json:"reduceOnly,omitempty"`
	CancelAfter         int64         `json:"cancelAfter,omitempty"`
	Size                float64       `json:"size,omitempty,string"`
	Price               float64       `json:"price,string,omitempty"`
	VisibleSize         float64       `json:"visibleSize,omitempty,string"`
	Funds               float64       `json:"funds,string,omitempty"`
}

// MarginOrderParam represents the margin place order request parameters
type MarginOrderParam struct {
	ClientOrderID       string        `json:"clientOid"`
	Side                string        `json:"side"`
	Symbol              currency.Pair `json:"symbol"`
	OrderType           string        `json:"type,omitempty"`
	Remark              string        `json:"remark,omitempty"`
	SelfTradePrevention string        `json:"stp,omitempty"`         // [Optional] self trade prevention , CN, CO, CB or DC. `CN` for Cancel newest, `DC` for Decrease and Cancel, `CO` for cancel oldest, and `CB` for Cancel both
	MarginModel         string        `json:"marginModel,omitempty"` // [Optional] The type of trading, including cross (cross mode) and isolated (isolated mode). It is set at cross by default
	AutoBorrow          bool          `json:"autoBorrow,omitempty"`  // [Optional] Auto-borrow to place order. The system will first borrow you funds at the optimal interest rate and then place an order for you. Currently autoBorrow parameter only supports cross mode, not isolated mode. When add this param, stop profit and stop loss are not supported
	AutoRepay           bool          `json:"autoRepay,omitempty"`
	Price               float64       `json:"price,string,omitempty"`
	Size                float64       `json:"size,omitempty,string"`
	TimeInForce         string        `json:"timeInForce,omitempty"` // [Optional] GTC, GTT, IOC, or FOK (default is GTC)
	CancelAfter         int64         `json:"cancelAfter,omitempty"` // [Optional] cancel after n seconds, requires timeInForce to be GTT
	PostOnly            bool          `json:"postOnly,omitempty"`
	Hidden              bool          `json:"hidden,omitempty"`
	Iceberg             bool          `json:"iceberg,omitempty"`
	VisibleSize         float64       `json:"visibleSize,omitempty,string"`
	Funds               float64       `json:"funds,string,omitempty"`
}

// UniversalTransferParam represents a universal transfer parameter
type UniversalTransferParam struct {
	ClientSuppliedOrderID string        `json:"clientOid"`
	Currency              currency.Code `json:"currency"`
	Amount                float64       `json:"amount"`
	FromUserID            string        `json:"fromUserId"`
	FromAccountType       string        `json:"fromAccountType"` // Account type：MAIN、TRADE、CONTRACT、MARGIN、ISOLATED、TRADE_HF、MARGIN_V2、ISOLATED_V2
	FromAccountTag        string        `json:"fromAccountTag"`  // Symbol, required when the account type is ISOLATED or ISOLATED_V2, for example: BTC-USDT
	TransferType          string        `json:"type"`            // Transfer Type: Transfer type：INTERNAL、PARENT_TO_SUB，SUB_TO_PARENT
	ToUserID              string        `json:"toUserId"`
	ToAccountType         string        `json:"toAccountType"`
	ToAccountTag          string        `json:"toAccountTag"`
}

// OCOOrderParams represents a an OCO order creation parameter
type OCOOrderParams struct {
	Symbol        currency.Pair `json:"symbol"`
	Side          string        `json:"side"`
	Price         float64       `json:"price,string"`
	Size          float64       `json:"size,string"`
	StopPrice     float64       `json:"stopPrice,string"`
	LimitPrice    float64       `json:"limitPrice,string"` // The limit order price after take-profit and stop-loss are triggered
	TradeType     string        `json:"tradeType,omitempty"`
	ClientOrderID string        `json:"clientOid"`
	Remark        string        `json:"remark,omitempty"`
}

// OCOOrderCancellationResponse represents an order cancellation response
type OCOOrderCancellationResponse struct {
	CancelledOrderIDs []string `json:"cancelledOrderIds"`
}

// OCOOrderInfo represents an order info
type OCOOrderInfo struct {
	OrderID       string     `json:"orderId"`
	Symbol        string     `json:"symbol"`
	ClientOrderID string     `json:"clientOid"`
	OrderTime     types.Time `json:"orderTime"`
	Status        string     `json:"status"`
}

// OCOOrderDetail represents an oco order detail via the order ID
type OCOOrderDetail struct {
	OrderID   string         `json:"orderId"`
	Symbol    string         `json:"symbol"`
	ClientOid string         `json:"clientOid"`
	OrderTime types.Time     `json:"orderTime"`
	Status    string         `json:"status"`
	Orders    []OCOOrderItem `json:"orders"`
}

// OCOOrderItem represents an OCO order item
type OCOOrderItem struct {
	ID        string       `json:"id"`
	Symbol    string       `json:"symbol"`
	Side      string       `json:"side"`
	Price     types.Number `json:"price"`
	StopPrice types.Number `json:"stopPrice"`
	Size      types.Number `json:"size"`
	Status    string       `json:"status"`
}

// OCOOrders represents an OCO orders list
type OCOOrders struct {
	CurrentPage int64          `json:"currentPage"`
	PageSize    int64          `json:"pageSize"`
	TotalNum    int64          `json:"totalNum"`
	TotalPage   int64          `json:"totalPage"`
	Items       []OCOOrderInfo `json:"items"`
}

// PlaceMarginHFOrderParam represents a margin HF order parameters
type PlaceMarginHFOrderParam struct {
	ClientOrderID       string        `json:"clientOid"`
	Side                string        `json:"side"`
	Symbol              currency.Pair `json:"symbol"`
	OrderType           string        `json:"type,omitempty"`
	SelfTradePrevention string        `json:"stp,omitempty"`
	IsIsolated          bool          `json:"isIsolated,omitempty"`
	AutoBorrow          bool          `json:"autoBorrow,omitempty"`
	AutoRepay           bool          `json:"autoRepay,omitempty"`
	Price               float64       `json:"price,string"`
	Size                float64       `json:"size,string"`
	TimeInForce         string        `json:"timeInForce,omitempty,string"`
	CancelAfter         int64         `json:"cancelAfter,omitempty,string"`
	PostOnly            bool          `json:"postOnly,omitempty,string"`
	Hidden              bool          `json:"hidden,omitempty,string"`
	Iceberg             bool          `json:"iceberg,omitempty,string"`
	VisibleSize         float64       `json:"visibleSize,omitempty,string"`
	Funds               string        `json:"funds,omitempty"`
}

// MarginHFOrderResponse  represents a margin HF order creation response
type MarginHFOrderResponse struct {
	OrderID     string  `json:"orderId"`
	BorrowSize  float64 `json:"borrowSize"`
	LoanApplyID string  `json:"loanApplyId"`
}

// FilledMarginHFOrdersResponse represents a filled HF margin orders
type FilledMarginHFOrdersResponse struct {
	LastID int64         `json:"lastId"`
	Items  []OrderDetail `json:"items"`
}

// HFMarginOrderTrade represents a HF margin order trade item
type HFMarginOrderTrade struct {
	ID             int64        `json:"id"`
	Symbol         string       `json:"symbol"`
	TradeID        int64        `json:"tradeId"`
	OrderID        string       `json:"orderId"`
	CounterOrderID string       `json:"counterOrderId"`
	Side           string       `json:"side"`
	Liquidity      string       `json:"liquidity"`
	ForceTaker     bool         `json:"forceTaker"`
	Price          types.Number `json:"price"`
	Size           types.Number `json:"size"`
	Funds          types.Number `json:"funds"`
	Fee            types.Number `json:"fee"`
	FeeRate        types.Number `json:"feeRate"`
	FeeCurrency    string       `json:"feeCurrency"`
	Stop           string       `json:"stop"`
	TradeType      string       `json:"tradeType"`
	Type           string       `json:"type"`
	CreatedAt      types.Time   `json:"createdAt"`
}

// HFMarginOrderTransaction represents a HF margin order transaction detail
type HFMarginOrderTransaction struct {
	Items  []HFMarginOrderTrade `json:"items"`
	LastID int64                `json:"lastId"`
}

// Level2Depth5Or20 stores the orderbook data for the level 5 or level 20
// orderbook
type Level2Depth5Or20 struct {
	Asks      [][2]types.Number `json:"asks"`
	Bids      [][2]types.Number `json:"bids"`
	Timestamp types.Time        `json:"timestamp"`
}

// TradingPairFee represents actual fee information of a trading fee
type TradingPairFee struct {
	Symbol       string       `json:"symbol"`
	TakerFeeRate types.Number `json:"takerFeeRate"`
	MakerFeeRate types.Number `json:"makerFeeRate"`
}

// FuturesPositionHistory represents a position history of futures asset
type FuturesPositionHistory struct {
	CurrentPage int64                   `json:"currentPage"`
	PageSize    int64                   `json:"pageSize"`
	TotalNum    int64                   `json:"totalNum"`
	TotalPage   int64                   `json:"totalPage"`
	Items       []FuturesPositionDetail `json:"items"`
}

// FuturesPositionDetail represents a futures position detail
type FuturesPositionDetail struct {
	CloseID            string       `json:"closeId"`
	PositionID         string       `json:"positionId"`
	UID                int64        `json:"uid"`
	UserID             string       `json:"userId"`
	Symbol             string       `json:"symbol"`
	SettleCurrency     string       `json:"settleCurrency"`
	Leverage           types.Number `json:"leverage"`
	Type               string       `json:"type"`
	Side               string       `json:"side"`
	CloseSize          types.Number `json:"closeSize"`
	PNL                types.Number `json:"pnl"`
	RealisedGrossCost  types.Number `json:"realisedGrossCost"`
	WithdrawPNL        types.Number `json:"withdrawPnl"`
	ReturnOnEquityRate types.Number `json:"roe"`
	TradeFee           types.Number `json:"tradeFee"`
	FundingFee         types.Number `json:"fundingFee"`
	OpenTime           types.Time   `json:"openTime"`
	CloseTime          types.Time   `json:"closeTime"`
	OpenPrice          types.Number `json:"openPrice"`
	ClosePrice         types.Number `json:"closePrice"`
}

// SusbcribeEarn represents a subscription to earn
type SusbcribeEarn struct {
	OrderID             string `json:"orderId"`
	SubscriptionOrderID string `json:"orderTxId"`
}

// EarnRedeem represents an earn redeem by holding id
type EarnRedeem struct {
	RedemptionOrderID string       `json:"orderTxId"`
	DeliverTime       types.Time   `json:"deliverTime"`
	Status            string       `json:"status"`
	Amount            types.Number `json:"amount"`
}

// EarnRedemptionPreview represents a redemption information of a holding
type EarnRedemptionPreview struct {
	Currency              string       `json:"currency"`
	RedeemAmount          types.Number `json:"redeemAmount"`
	PenaltyInterestAmount types.Number `json:"penaltyInterestAmount"`
	RedeemPeriod          int64        `json:"redeemPeriod"`
	DeliverTime           types.Time   `json:"deliverTime"`
	ManualRedeemable      bool         `json:"manualRedeemable"`
	RedeemAll             bool         `json:"redeemAll"`
}

// EarnSavingProduct represents a saving product instance
type EarnSavingProduct struct {
	ID                   string       `json:"id"`
	Currency             string       `json:"currency"`
	Category             string       `json:"category"`
	Type                 string       `json:"type"`
	Precision            int64        `json:"precision"`
	ProductUpperLimit    string       `json:"productUpperLimit"`
	UserUpperLimit       types.Number `json:"userUpperLimit"`
	UserLowerLimit       types.Number `json:"userLowerLimit"`
	RedeemPeriod         int64        `json:"redeemPeriod"`
	LockStartTime        types.Time   `json:"lockStartTime"`
	LockEndTime          types.Time   `json:"lockEndTime"`
	ApplyStartTime       types.Time   `json:"applyStartTime"`
	ApplyEndTime         types.Time   `json:"applyEndTime"`
	ReturnRate           types.Number `json:"returnRate"`
	IncomeCurrency       string       `json:"incomeCurrency"`
	EarlyRedeemSupported int64        `json:"earlyRedeemSupported"`
	ProductRemainAmount  types.Number `json:"productRemainAmount"`
	Status               string       `json:"status"`
	RedeemType           string       `json:"redeemType"`
	IncomeReleaseType    string       `json:"incomeReleaseType"`
	InterestDate         int64        `json:"interestDate"`
	Duration             int64        `json:"duration"`
	NewUserOnly          int64        `json:"newUserOnly"`
}

// FixedIncomeEarnHoldings represents a fixed income earn holdings
type FixedIncomeEarnHoldings struct {
	TotalNum    int64                `json:"totalNum"`
	Items       []FixedIncomeHolding `json:"items"`
	CurrentPage int64                `json:"currentPage"`
	PageSize    int64                `json:"pageSize"`
	TotalPage   int64                `json:"totalPage"`
}

// FixedIncomeHolding represents a fixed income earn holding detail
type FixedIncomeHolding struct {
	OrderID              string       `json:"orderId"`
	ProductID            string       `json:"productId"`
	ProductCategory      string       `json:"productCategory"`
	ProductType          string       `json:"productType"`
	Currency             string       `json:"currency"`
	IncomeCurrency       string       `json:"incomeCurrency"`
	ReturnRate           types.Number `json:"returnRate"`
	HoldAmount           types.Number `json:"holdAmount"`
	RedeemedAmount       types.Number `json:"redeemedAmount"`
	RedeemingAmount      types.Number `json:"redeemingAmount"`
	LockStartTime        types.Time   `json:"lockStartTime"`
	LockEndTime          types.Time   `json:"lockEndTime"`
	PurchaseTime         types.Time   `json:"purchaseTime"`
	RedeemPeriod         int64        `json:"redeemPeriod"`
	Status               string       `json:"status"`
	EarlyRedeemSupported int64        `json:"earlyRedeemSupported"`
}

// EarnProduct represents a time-limited earn limited product item
type EarnProduct struct {
	ID                   string       `json:"id"`
	Currency             string       `json:"currency"`
	Category             string       `json:"category"`
	Type                 string       `json:"type"`
	Precision            float64      `json:"precision"`
	ProductUpperLimit    types.Number `json:"productUpperLimit"`
	UserUpperLimit       types.Number `json:"userUpperLimit"`
	UserLowerLimit       types.Number `json:"userLowerLimit"`
	RedeemPeriod         int64        `json:"redeemPeriod"`
	LockStartTime        types.Time   `json:"lockStartTime"`
	LockEndTime          types.Time   `json:"lockEndTime"`
	ApplyStartTime       types.Time   `json:"applyStartTime"`
	ApplyEndTime         types.Time   `json:"applyEndTime"`
	ReturnRate           types.Number `json:"returnRate"`
	IncomeCurrency       string       `json:"incomeCurrency"`
	EarlyRedeemSupported int64        `json:"earlyRedeemSupported"`
	ProductRemainAmount  types.Number `json:"productRemainAmount"`
	Status               string       `json:"status"`
	RedeemType           string       `json:"redeemType"`
	IncomeReleaseType    string       `json:"incomeReleaseType"`
	InterestDate         int64        `json:"interestDate"`
	Duration             int64        `json:"duration"`
	NewUserOnly          int64        `json:"newUserOnly"`
}

// OffExchangeFundingAndLoan represents information of off-exchange funding and loan
type OffExchangeFundingAndLoan struct {
	MasterAccountUID string                           `json:"parentUid"`
	Orders           []OffExchangeFundingAndLoanOrder `json:"orders"`
	LoanToValueRatio struct {
		TransferLtv           types.Number `json:"transferLtv"`
		OnlyClosePosLtv       types.Number `json:"onlyClosePosLtv"`
		DelayedLiquidationLtv types.Number `json:"delayedLiquidationLtv"`
		InstantLiquidationLtv types.Number `json:"instantLiquidationLtv"`
		CurrentLtv            types.Number `json:"currentLtv"`
	} `json:"ltv"`
	TotalMarginAmount    types.Number                   `json:"totalMarginAmount"`
	TransferMarginAmount types.Number                   `json:"transferMarginAmount"`
	Margins              []ExchangeFundingAndLoanMargin `json:"margins"`
}

// ExchangeFundingAndLoanMargin represents an exchange funding and loan margin
type ExchangeFundingAndLoanMargin struct {
	MarginCcy    string       `json:"marginCcy"`
	MarginQty    types.Number `json:"marginQty"`
	MarginFactor string       `json:"marginFactor"`
}

// OffExchangeFundingAndLoanOrder represents an exchange funding and loan order detail
type OffExchangeFundingAndLoanOrder struct {
	OrderID   string       `json:"orderId"`
	Currency  string       `json:"currency"`
	Principal types.Number `json:"principal"`
	Interest  types.Number `json:"interest"`
}

// VIPLendingAccounts represents VIP accounts involved in off-exchange loans
type VIPLendingAccounts struct {
	UID          string       `json:"uid"`
	MarginCcy    string       `json:"marginCcy"`
	MarginQty    types.Number `json:"marginQty"`
	MarginFactor types.Number `json:"marginFactor"`
	AccountType  string       `json:"accountType"`
	IsParent     bool         `json:"isParent"`
}

// UserRebateInfo represents a user rebate information
type UserRebateInfo struct {
	M1UID    string       `json:"m1Uid"`
	Rcode    string       `json:"rcode"`
	M2UID    string       `json:"m2Uid"`
	Amount   types.Number `json:"amount"`
	Rebate   types.Number `json:"rebate"`
	CashBack types.Number `json:"cashBack"`
	Offset   string       `json:"offset"`
}

// MarginPairConfigs querying the configuration of cross margin trading pairs
type MarginPairConfigs struct {
	Timestamp types.Time         `json:"timestamp"`
	Items     []MarginPairConfig `json:"items"`
}

// MarginActiveSymbolDetail represents an active high frequency margin symbols information
type MarginActiveSymbolDetail struct {
	SymbolSize int64    `json:"symbolSize"`
	Symbols    []string `json:"symbols"`
}

// MarginInterestRecords represents a cross/isolated margin interest records
type MarginInterestRecords struct {
	Timestamp   types.Time           `json:"timestamp"`
	CurrentPage int64                `json:"currentPage"`
	PageSize    int64                `json:"pageSize"`
	TotalNum    int64                `json:"totalNum"`
	TotalPage   int64                `json:"totalPage"`
	Items       []MarginInterestInfo `json:"items"`
}

// MarginInterestInfo represents a margin account currency interest information
type MarginInterestInfo struct {
	CreatedAt      types.Time   `json:"createdAt"`
	Currency       string       `json:"currency"`
	InterestAmount types.Number `json:"interestAmount"`
	DayRatio       types.Number `json:"dayRatio"`
}

// MarginPairConfig represents a margin pair configuration detail
type MarginPairConfig struct {
	Symbol         string       `json:"symbol"`
	Name           string       `json:"name"`
	EnableTrading  bool         `json:"enableTrading"`
	Market         string       `json:"market"`
	BaseCurrency   string       `json:"baseCurrency"`
	QuoteCurrency  string       `json:"quoteCurrency"`
	BaseIncrement  types.Number `json:"baseIncrement"`
	BaseMinSize    types.Number `json:"baseMinSize"`
	QuoteIncrement types.Number `json:"quoteIncrement"`
	QuoteMinSize   types.Number `json:"quoteMinSize"`
	BaseMaxSize    types.Number `json:"baseMaxSize"`
	QuoteMaxSize   types.Number `json:"quoteMaxSize"`
	PriceIncrement types.Number `json:"priceIncrement"`
	FeeCurrency    string       `json:"feeCurrency"`
	PriceLimitRate types.Number `json:"priceLimitRate"`
	MinFunds       types.Number `json:"minFunds"`
}

// FuturesMaxOpenPositionSize represents maximum buy/sell open positions an account could have.
type FuturesMaxOpenPositionSize struct {
	Symbol          string `json:"symbol"`
	MaxBuyOpenSize  int64  `json:"maxBuyOpenSize"`
	MaxSellOpenSize int64  `json:"maxSellOpenSize"`
}
