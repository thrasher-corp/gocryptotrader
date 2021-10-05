package btse

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
)

const (
	// Default order type is good till cancel (or filled)
	goodTillCancel = "GTC"

	orderInserted  = 2
	orderCancelled = 6
)

// FundingHistoryData stores funding history data
type FundingHistoryData struct {
	Time   int64   `json:"time"`
	Rate   float64 `json:"rate"`
	Symbol string  `json:"symbol"`
}

// MarketSummary response data
type MarketSummary []struct {
	Symbol              string   `json:"symbol"`
	Last                float64  `json:"last"`
	LowestAsk           float64  `json:"lowestAsk"`
	HighestBid          float64  `json:"highestBid"`
	PercentageChange    float64  `json:"percentageChange"`
	Volume              float64  `json:"volume"`
	High24Hr            float64  `json:"high24Hr"`
	Low24Hr             float64  `json:"low24Hr"`
	Base                string   `json:"base"`
	Quote               string   `json:"quote"`
	Active              bool     `json:"active"`
	Size                float64  `json:"size"`
	MinValidPrice       float64  `json:"minValidPrice"`
	MinPriceIncrement   float64  `json:"minPriceIncrement"`
	MinOrderSize        float64  `json:"minOrderSize"`
	MaxOrderSize        float64  `json:"maxOrderSize"`
	MinSizeIncrement    float64  `json:"minSizeIncrement"`
	OpenInterest        float64  `json:"openInterest"`
	OpenInterestUSD     float64  `json:"openInterestUSD"`
	ContractStart       int64    `json:"contractStart"`
	ContractEnd         int64    `json:"contractEnd"`
	TimeBasedContract   bool     `json:"timeBasedContract"`
	OpenTime            int64    `json:"openTime"`
	CloseTime           int64    `json:"closeTime"`
	StartMatching       int64    `json:"startMatching"`
	InactiveTime        int64    `json:"inactiveTime"`
	FundingRate         float64  `json:"fundingRate"`
	ContractSize        float64  `json:"contractSize"`
	MaxPosition         int64    `json:"maxPosition"`
	MinRiskLimit        int      `json:"minRiskLimit"`
	MaxRiskLimit        int      `json:"maxRiskLimit"`
	AvailableSettlement []string `json:"availableSettlement"`
	Futures             bool     `json:"futures"`
}

// OHLCV holds Open, High Low, Close, Volume data for set symbol
type OHLCV [][]float64

// Price stores last price for requested symbol
type Price []struct {
	IndexPrice float64 `json:"indexPrice"`
	LastPrice  float64 `json:"lastPrice"`
	MarkPrice  float64 `json:"markPrice"`
	Symbol     string  `json:"symbol"`
}

// SpotMarket stores market data
type SpotMarket struct {
	Symbol            string  `json:"symbol"`
	ID                string  `json:"id"`
	BaseCurrency      string  `json:"base_currency"`
	QuoteCurrency     string  `json:"quote_currency"`
	BaseMinSize       float64 `json:"base_min_size"`
	BaseMaxSize       float64 `json:"base_max_size"`
	BaseIncrementSize float64 `json:"base_increment_size"`
	QuoteMinPrice     float64 `json:"quote_min_price"`
	QuoteIncrement    float64 `json:"quote_increment"`
	Status            string  `json:"status"`
}

// FuturesMarket stores market data
type FuturesMarket struct {
	Symbol              string   `json:"symbol"`
	Last                float64  `json:"last"`
	LowestAsk           float64  `json:"lowestAsk"`
	HighestBid          float64  `json:"highestBid"`
	OpenInterest        float64  `json:"openInterest"`
	OpenInterestUSD     float64  `json:"openInterestUSD"`
	PercentageChange    float64  `json:"percentageChange"`
	Volume              float64  `json:"volume"`
	High24Hr            float64  `json:"high24Hr"`
	Low24Hr             float64  `json:"low24Hr"`
	Base                string   `json:"base"`
	Quote               string   `json:"quote"`
	ContractStart       int64    `json:"contractStart"`
	ContractEnd         int64    `json:"contractEnd"`
	Active              bool     `json:"active"`
	TimeBasedContract   bool     `json:"timeBasedContract"`
	OpenTime            int64    `json:"openTime"`
	CloseTime           int64    `json:"closeTime"`
	StartMatching       int64    `json:"startMatching"`
	InactiveTime        int64    `json:"inactiveTime"`
	FundingRate         float64  `json:"fundingRate"`
	ContractSize        float64  `json:"contractSize"`
	MaxPosition         int64    `json:"maxPosition"`
	MinValidPrice       float64  `json:"minValidPrice"`
	MinPriceIncrement   float64  `json:"minPriceIncrement"`
	MinOrderSize        int32    `json:"minOrderSize"`
	MaxOrderSize        int32    `json:"maxOrderSize"`
	MinRiskLimit        int32    `json:"minRiskLimit"`
	MaxRiskLimit        int32    `json:"maxRiskLimit"`
	MinSizeIncrement    float64  `json:"minSizeIncrement"`
	AvailableSettlement []string `json:"availableSettlement"`
}

// Trade stores trade data
type Trade struct {
	SerialID int64   `json:"serialId"`
	Symbol   string  `json:"symbol"`
	Price    float64 `json:"price"`
	Amount   float64 `json:"size"`
	Time     int64   `json:"timestamp"`
	Side     string  `json:"side"`
	Type     string  `json:"type"`
}

// QuoteData stores quote data
type QuoteData struct {
	Price float64 `json:"price,string"`
	Size  float64 `json:"size,string"`
}

// Orderbook stores orderbook info
type Orderbook struct {
	BuyQuote  []QuoteData `json:"buyQuote"`
	SellQuote []QuoteData `json:"sellQuote"`
	Symbol    string      `json:"symbol"`
	Timestamp int64       `json:"timestamp"`
}

// Ticker stores the ticker data
type Ticker struct {
	Price  float64 `json:"price,string"`
	Size   float64 `json:"size,string"`
	Bid    float64 `json:"bid,string"`
	Ask    float64 `json:"ask,string"`
	Volume float64 `json:"volume,string"`
	Time   string  `json:"time"`
}

// MarketStatistics stores market statistics for a particular product
type MarketStatistics struct {
	Open   float64   `json:"open,string"`
	Low    float64   `json:"low,string"`
	High   float64   `json:"high,string"`
	Close  float64   `json:"close,string"`
	Volume float64   `json:"volume,string"`
	Time   time.Time `json:"time"`
}

// ServerTime stores the server time data
type ServerTime struct {
	ISO   time.Time `json:"iso"`
	Epoch int64     `json:"epoch"`
}

// CurrencyBalance stores the account info data
type CurrencyBalance struct {
	Currency  string  `json:"currency"`
	Total     float64 `json:"total"`
	Available float64 `json:"available"`
}

// AccountFees stores fee for each currency pair
type AccountFees struct {
	MakerFee float64 `json:"makerFee"`
	Symbol   string  `json:"symbol"`
	TakerFee float64 `json:"takerFee"`
}

// TradeHistory stores user trades for exchange
type TradeHistory []struct {
	Base         string  `json:"base"`
	ClOrderID    string  `json:"clOrderID"`
	FeeAmount    float64 `json:"feeAmount"`
	FeeCurrency  string  `json:"feeCurrency"`
	FilledPrice  float64 `json:"filledPrice"`
	FilledSize   float64 `json:"filledSize"`
	OrderID      string  `json:"orderId"`
	OrderType    int     `json:"orderType"`
	Price        float64 `json:"price"`
	Quote        string  `json:"quote"`
	RealizedPnl  float64 `json:"realizedPnl"`
	SerialID     int64   `json:"serialId"`
	Side         string  `json:"side"`
	Size         float64 `json:"size"`
	Symbol       string  `json:"symbol"`
	Timestamp    string  `json:"timestamp"`
	Total        float64 `json:"total"`
	TradeID      string  `json:"tradeId"`
	TriggerPrice float64 `json:"triggerPrice"`
	TriggerType  int     `json:"triggerType"`
	Username     string  `json:"username"`
	Wallet       string  `json:"wallet"`
}

// WalletHistory stores account funding history
type WalletHistory []struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	Fees        float64 `json:"fees"`
	OrderID     string  `json:"orderId"`
	Status      string  `json:"status"`
	Timestamp   int64   `json:"timestamp"`
	Type        string  `json:"type"`
	Username    string  `json:"username"`
	Wallet      string  `json:"wallet"`
}

// WalletAddress stores address for crypto deposit's
type WalletAddress []struct {
	Address string `json:"address"`
	Created int    `json:"created"`
}

// WithdrawalResponse response received when submitting a crypto withdrawal request
type WithdrawalResponse struct {
	WithdrawID string `json:"withdraw_id"`
}

// OpenOrder stores an open order info
type OpenOrder struct {
	AverageFillPrice             float64 `json:"averageFillPrice"`
	CancelDuration               int64   `json:"cancelDuration"`
	ClOrderID                    string  `json:"clOrderID"`
	FillSize                     float64 `json:"fillSize"`
	FilledSize                   float64 `json:"filledSize"`
	OrderID                      string  `json:"orderID"`
	OrderState                   string  `json:"orderState"`
	OrderType                    int     `json:"orderType"`
	OrderValue                   float64 `json:"orderValue"`
	PegPriceDeviation            float64 `json:"pegPriceDeviation"`
	PegPriceMax                  float64 `json:"pegPriceMax"`
	PegPriceMin                  float64 `json:"pegPriceMin"`
	Price                        float64 `json:"price"`
	Side                         string  `json:"side"`
	Size                         float64 `json:"size"`
	Symbol                       string  `json:"symbol"`
	Timestamp                    int64   `json:"timestamp"`
	TrailValue                   float64 `json:"trailValue"`
	TriggerOrder                 bool    `json:"triggerOrder"`
	TriggerOrderType             int     `json:"triggerOrderType"`
	TriggerOriginalPrice         float64 `json:"triggerOriginalPrice"`
	TriggerPrice                 float64 `json:"triggerPrice"`
	TriggerStopPrice             float64 `json:"triggerStopPrice"`
	TriggerTrailingStopDeviation float64 `json:"triggerTrailingStopDeviation"`
	Triggered                    bool    `json:"triggered"`
}

// CancelOrder stores slice of orders
type CancelOrder []Order

// Order stores information for a single order
type Order struct {
	AverageFillPrice float64 `json:"averageFillPrice"`
	ClOrderID        string  `json:"clOrderID"`
	Deviation        float64 `json:"deviation"`
	FillSize         float64 `json:"fillSize"`
	Message          string  `json:"message"`
	OrderID          string  `json:"orderID"`
	OrderType        int     `json:"orderType"`
	Price            float64 `json:"price"`
	Side             string  `json:"side"`
	Size             float64 `json:"size"`
	Status           int     `json:"status"`
	Stealth          float64 `json:"stealth"`
	StopPrice        float64 `json:"stopPrice"`
	Symbol           string  `json:"symbol"`
	Timestamp        int64   `json:"timestamp"`
	Trigger          bool    `json:"trigger"`
	TriggerPrice     float64 `json:"triggerPrice"`
}

type wsSub struct {
	Operation string   `json:"op"`
	Arguments []string `json:"args"`
}

type wsQuoteData struct {
	Total string `json:"cumulativeTotal"`
	Price string `json:"price"`
	Size  string `json:"size"`
}

type wsOBData struct {
	Currency  string        `json:"currency"`
	BuyQuote  []wsQuoteData `json:"buyQuote"`
	SellQuote []wsQuoteData `json:"sellQuote"`
}

type wsOrderBook struct {
	Topic string   `json:"topic"`
	Data  wsOBData `json:"data"`
}

type wsTradeData struct {
	Amount          float64 `json:"amount"`
	Gain            int64   `json:"gain"`
	Newest          int64   `json:"newest"`
	Price           float64 `json:"price"`
	ID              int64   `json:"serialId"`
	TransactionTime int64   `json:"transactionUnixTime"`
}

type wsTradeHistory struct {
	Topic string        `json:"topic"`
	Data  []wsTradeData `json:"data"`
}

type wsNotification struct {
	Topic string          `json:"topic"`
	Data  []wsOrderUpdate `json:"data"`
}

type wsOrderUpdate struct {
	OrderID           string  `json:"orderID"`
	OrderMode         string  `json:"orderMode"`
	OrderType         string  `json:"orderType"`
	PegPriceDeviation string  `json:"pegPriceDeviation"`
	Price             float64 `json:"price,string"`
	Size              float64 `json:"size,string"`
	Status            string  `json:"status"`
	Stealth           string  `json:"stealth"`
	Symbol            string  `json:"symbol"`
	Timestamp         int64   `json:"timestamp,string"`
	TriggerPrice      float64 `json:"triggerPrice,string"`
	Type              string  `json:"type"`
}

// ErrorResponse contains errors received from API
type ErrorResponse struct {
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
	Status    int    `json:"status"`
}

// OrderSizeLimit holds accepted minimum, maximum, and size increment when submitting new orders
type OrderSizeLimit struct {
	MinOrderSize     float64
	MaxOrderSize     float64
	MinSizeIncrement float64
}

// orderSizeLimitMap map of OrderSizeLimit per currency
var orderSizeLimitMap sync.Map

// WsSubscriptionAcknowledgement contains successful subscription messages
type WsSubscriptionAcknowledgement struct {
	Channel []string `json:"channel"`
	Event   string   `json:"event"`
}

// WsLoginAcknowledgement contains whether authentication was successful
type WsLoginAcknowledgement struct {
	Event   string `json:"event"`
	Success bool   `json:"success"`
}

// transferFees defines exchange crypto currency transfer fees, subject to
// change.
// NOTE: https://www.btse.com/en/deposit-withdrawal-fees
var transferFees = map[asset.Item]map[currency.Code]fee.Transfer{
	asset.Spot: {
		currency.AAVE: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.4003), Withdrawal: fee.Convert(0.1003)},
		currency.ADA:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(8), Withdrawal: fee.Convert(1)},
		currency.ATOM: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(.02), Withdrawal: fee.Convert(.01)},
		currency.BAL:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2.889), Withdrawal: fee.Convert(1.389)},
		currency.BAND: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5.7), Withdrawal: fee.Convert(2.85)},
		currency.BCB:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.1)},
		currency.BNB:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(.008), Withdrawal: fee.Convert(.0005)},
		currency.BNT:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(15.885), Withdrawal: fee.Convert(7.885)},
		currency.BRZ:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(23), Withdrawal: fee.Convert(22)},
		currency.BTC:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.001), Withdrawal: fee.Convert(0.0005)},
		currency.BTSE: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(12.461), Withdrawal: fee.Convert(2.461)},
		currency.BUSD: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(50), Withdrawal: fee.Convert(25)},
		// currency.BUSD:{Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(10), Withdrawal: fee.Convert(.5)}, // TODO: ADD IN NETWORK HANDLING BEP20
		currency.COMP: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1.01243), Withdrawal: fee.Convert(0.01243)},
		currency.CRV:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(20), Withdrawal: fee.Convert(10)},
		currency.DAI:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(39.97), Withdrawal: fee.Convert(29.97)},
		currency.DOGE: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(0.82)},
		currency.DOT:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(0.1)},
		currency.ETH:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.042118), Withdrawal: fee.Convert(0.002118)},
		// currency.ETH:{Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.02), Withdrawal: fee.Convert(0.01)}, // TODO: TRC20
		currency.FIL:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.01), Withdrawal: fee.Convert(0.001)},
		currency.FLY:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(168), Withdrawal: fee.Convert(118)},
		currency.FRM:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(100), Withdrawal: fee.Convert(80)},
		currency.FTT:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.5)},
		currency.HT:    {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5.718), Withdrawal: fee.Convert(3.718)},
		currency.HXRO:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(150), Withdrawal: fee.Convert(50)},
		currency.JST:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(350), Withdrawal: fee.Convert(250)},
		currency.LEO:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(19.22), Withdrawal: fee.Convert(10.22)},
		currency.LINK:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2.138), Withdrawal: fee.Convert(1.138)},
		currency.LTC:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.002), Withdrawal: fee.Convert(0.001)},
		currency.MATIC: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.2), Withdrawal: fee.Convert(0.1)},
		currency.MBM:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(300), Withdrawal: fee.Convert(200)},
		currency.MKR:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.02), Withdrawal: fee.Convert(0.01)},
		currency.PAX:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(39.99), Withdrawal: fee.Convert(29.99)},
		currency.PHNX:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(150), Withdrawal: fee.Convert(140)},
		currency.SFI:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.1), Withdrawal: fee.Convert(0.001)},
		currency.SHIB:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(3953154), Withdrawal: fee.Convert(2305154)},
		currency.STAKE: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2.979), Withdrawal: fee.Convert(1.979)},
		currency.SUSHI: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5.892), Withdrawal: fee.Convert(2.892)},
		currency.SWRV:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5), Withdrawal: fee.Convert(4)},
		currency.TRX:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(1)},
		currency.TRYB:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(1.4)},
		currency.TUSD:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(40.97), Withdrawal: fee.Convert(29.97)},
		currency.UNI:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(3.202), Withdrawal: fee.Convert(1.202)},
		currency.USDC:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(40.97), Withdrawal: fee.Convert(29.97)},
		currency.USDP:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.1)},
		// currency.USDT:{Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(10), Withdrawal: fee.Convert(1)}, TODO: TRC20
		currency.USDT: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(59.96), Withdrawal: fee.Convert(29.96)},
		currency.WAUD: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(13)},
		currency.WCAD: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(12)},
		currency.WCHF: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(9)},
		currency.WEUR: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(8)},
		currency.WGBP: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(7)},
		currency.WHKD: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(77)},
		currency.WINR: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(729)},
		currency.WJPY: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(1050)},
		currency.WMYR: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(40)},
		currency.WOO:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(133.29), Withdrawal: fee.Convert(33.29)},
		currency.WSGD: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(13)},
		currency.WUSD: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(10)},
		currency.WXMR: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(0.06)},
		currency.XAUT: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.02303), Withdrawal: fee.Convert(0.01703)},
		currency.XMR:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.002), Withdrawal: fee.Convert(0.001)},
		currency.XRP:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.25)},
		currency.YFI:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.0014953), Withdrawal: fee.Convert(0.0009953)},
	},
}

// bankTransferFees defines bank transfer fees between exchange and bank. Subject
// to change.
// NOTE: https://support.btse.com/en/support/solutions/articles/43000588188#Fiat
var bankTransferFees = map[fee.BankTransaction]map[currency.Code]fee.Transfer{
	fee.Swift: {
		currency.USD: {Deposit: fee.Convert(0), Withdrawal: getWithdrawalUSD(currency.USD)},
		currency.EUR: {Deposit: fee.Convert(0), Withdrawal: getWithdrawalUSD(currency.EUR)},
		currency.GBP: {Deposit: fee.Convert(0), Withdrawal: getWithdrawalUSD(currency.GBP)},
		currency.HKD: {Deposit: fee.Convert(0), Withdrawal: getWithdrawalUSD(currency.HKD)},
		currency.SGD: {Deposit: fee.Convert(0), Withdrawal: getWithdrawalUSD(currency.SGD)},
		currency.JPY: {Deposit: fee.Convert(0), Withdrawal: getWithdrawalUSD(currency.JPY)},
		currency.AUD: {Deposit: fee.Convert(0), Withdrawal: getWithdrawalUSD(currency.AUD)},
		currency.AED: {Deposit: fee.Convert(0), Withdrawal: getWithdrawalUSD(currency.AED)},
		currency.CAD: {Deposit: fee.Convert(0), Withdrawal: getWithdrawalUSD(currency.CAD)},
	},
	fee.FasterPaymentService: {
		currency.GBP: {Deposit: fee.Convert(0), Withdrawal: getWithdrawal(currency.GBP, 0.0009)},
	},
	fee.SEPA: {
		currency.EUR: {Deposit: fee.Convert(0), Withdrawal: getWithdrawal(currency.EUR, 0.001)},
	},
}

func getWithdrawalUSD(c currency.Code) fee.Value {
	return &WithdrawalUSDValuedMinimumCharge{
		Code:           c,
		MinimumInUSD:   decimal.NewFromFloat(100),   // $100 USD value.
		PercentageRate: decimal.NewFromFloat(0.001), // 0.1% fee
		MinimumCharge:  decimal.NewFromFloat(25),    // $25 USD value
	}
}

// WithdrawalUSDValuedMinimumCharge defines a value structure that implements
// the fee.Value interface.
// This is a proof of concept. This relates also to the minimum charge in USD
type WithdrawalUSDValuedMinimumCharge struct {
	Code           currency.Code   `json:"-"`
	MinimumInUSD   decimal.Decimal `json:"minimumInUSD"`
	PercentageRate decimal.Decimal `json:"percentageRate"`
	MinimumCharge  decimal.Decimal `json:"minimumCharge"`
}

var errBelowMinimumAmount = errors.New("amount is less than minimum amount")

// GetFee returns the fee based off the amount requested
func (w WithdrawalUSDValuedMinimumCharge) GetFee(amount float64) (decimal.Decimal, error) {
	amt := decimal.NewFromFloat(amount)
	potentialFee := amt.Mul(w.PercentageRate)
	if w.Code.Item == currency.USD.Item {
		if amt.LessThan(w.MinimumInUSD) {
			return decimal.Zero, errBelowMinimumAmount
		}
		if potentialFee.LessThanOrEqual(w.MinimumCharge) {
			return w.MinimumCharge, nil
		}
		return potentialFee, nil
	}
	// attempt to attain correct foreign exchange value compared to USD
	fxRate, err := currency.ConvertCurrency(1, w.Code, currency.USD)
	if err != nil {
		return decimal.Zero, err
	}

	fxRateDec := decimal.NewFromFloat(fxRate)
	valueComparedToUSD := amt.Mul(fxRateDec)
	if valueComparedToUSD.LessThan(w.MinimumInUSD) {
		return decimal.Zero, errBelowMinimumAmount
	}

	if valueComparedToUSD.LessThanOrEqual(w.MinimumCharge) {
		// Return the minimum charge in the current currency
		return w.MinimumCharge.Mul(fxRateDec), nil
	}
	return potentialFee, nil
}

// Display displays current working internal data for use in RPC outputs
func (w WithdrawalUSDValuedMinimumCharge) Display() (string, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Validate validates current values
func (w *WithdrawalUSDValuedMinimumCharge) Validate() error {
	if w.Code.IsEmpty() {
		return errors.New("currency code is empty")
	}
	if w.MinimumInUSD.LessThanOrEqual(decimal.Zero) {
		return errors.New("invalid minimum in USD")
	}
	if w.PercentageRate.LessThanOrEqual(decimal.Zero) {
		return errors.New("invalid percentage rate")
	}
	if w.MinimumCharge.LessThanOrEqual(decimal.Zero) {
		return errors.New("invalid minimum charge")
	}
	return nil
}

// LessThan implements value interface, not needed.
func (w *WithdrawalUSDValuedMinimumCharge) LessThan(_ fee.Value) (bool, error) {
	return false, errors.New("cannot compare")
}

func getWithdrawal(c currency.Code, percentageRate float64) fee.Value {
	return &Withdrawal{
		Code:           c,
		MinimumInUSD:   decimal.NewFromFloat(100), // $100 USD value.
		PercentageRate: decimal.NewFromFloat(percentageRate),
		MinimumCharge:  decimal.NewFromFloat(25), // $25 USD value
	}
}

// Withdrawal defines a value structure that implements the fee.Value interface.
// This is a proof of concept.
type Withdrawal struct {
	Code           currency.Code   `json:"-"`
	MinimumInUSD   decimal.Decimal `json:"minimumInUSD"`
	PercentageRate decimal.Decimal `json:"percentageRate"`
	MinimumCharge  decimal.Decimal `json:"minimumCharge"`
}

// GetFee returns the fee based off the amount requested
func (w Withdrawal) GetFee(amount float64) (decimal.Decimal, error) {
	amt := decimal.NewFromFloat(amount)
	potentialFee := amt.Mul(w.PercentageRate)
	if w.Code.Item == currency.USD.Item {
		if amt.LessThan(w.MinimumInUSD) {
			return decimal.Zero, errBelowMinimumAmount
		}
		if potentialFee.LessThanOrEqual(w.MinimumCharge) {
			return w.MinimumCharge, nil
		}
		return potentialFee, nil
	}
	// attempt to attain correct foreign exchange value compared to USD
	fxRate, err := currency.ConvertCurrency(1, w.Code, currency.USD)
	if err != nil {
		return decimal.Zero, err
	}

	fxRateDec := decimal.NewFromFloat(fxRate)
	valueComparedToUSD := amt.Mul(fxRateDec)
	if valueComparedToUSD.LessThan(w.MinimumInUSD) {
		return decimal.Zero, errBelowMinimumAmount
	}

	if valueComparedToUSD.LessThanOrEqual(w.MinimumCharge) {
		// Return the minimum charge in the current currency
		return w.MinimumCharge, nil
	}
	return potentialFee, nil
}

// Display displays current working internal data for use in RPC outputs
func (w Withdrawal) Display() (string, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Validate validates current values
func (w *Withdrawal) Validate() error {
	if w.Code.IsEmpty() {
		return errors.New("currency code is empty")
	}
	if w.MinimumInUSD.LessThanOrEqual(decimal.Zero) {
		return errors.New("invalid minimum in USD")
	}
	if w.PercentageRate.LessThanOrEqual(decimal.Zero) {
		return errors.New("invalid percentage rate")
	}
	if w.MinimumCharge.LessThanOrEqual(decimal.Zero) {
		return errors.New("invalid minimum charge")
	}
	return nil
}

// LessThan implements value interface, not needed.
func (w *Withdrawal) LessThan(_ fee.Value) (bool, error) {
	return false, errors.New("cannot compare")
}
