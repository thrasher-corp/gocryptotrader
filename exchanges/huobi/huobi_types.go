package huobi

import "github.com/thrasher-/gocryptotrader/decimal"

// Response stores the Huobi response information
type Response struct {
	Status       string `json:"status"`
	Channel      string `json:"ch"`
	Timestamp    int64  `json:"ts"`
	ErrorCode    string `json:"err-code"`
	ErrorMessage string `json:"err-msg"`
}

// KlineItem stores a kline item
type KlineItem struct {
	ID     int             `json:"id"`
	Open   decimal.Decimal `json:"open"`
	Close  decimal.Decimal `json:"close"`
	Low    decimal.Decimal `json:"low"`
	High   decimal.Decimal `json:"high"`
	Amount decimal.Decimal `json:"amount"`
	Vol    decimal.Decimal `json:"vol"`
	Count  int             `json:"count"`
}

// DetailMerged stores the ticker detail merged data
type DetailMerged struct {
	Detail
	Version int               `json:"version"`
	Ask     []decimal.Decimal `json:"ask"`
	Bid     []decimal.Decimal `json:"bid"`
}

// OrderBookDataRequestParamsType var for request param types
type OrderBookDataRequestParamsType string

// vars for OrderBookDataRequestParamsTypes
var (
	OrderBookDataRequestParamsTypeNone  = OrderBookDataRequestParamsType("")
	OrderBookDataRequestParamsTypeStep0 = OrderBookDataRequestParamsType("step0")
	OrderBookDataRequestParamsTypeStep1 = OrderBookDataRequestParamsType("step1")
	OrderBookDataRequestParamsTypeStep2 = OrderBookDataRequestParamsType("step2")
	OrderBookDataRequestParamsTypeStep3 = OrderBookDataRequestParamsType("step3")
	OrderBookDataRequestParamsTypeStep4 = OrderBookDataRequestParamsType("step4")
	OrderBookDataRequestParamsTypeStep5 = OrderBookDataRequestParamsType("step5")
)

// OrderBookDataRequestParams represents Klines request data.
type OrderBookDataRequestParams struct {
	Symbol string                         `json:"symbol"` // Required; example LTCBTC,BTCUSDT
	Type   OrderBookDataRequestParamsType `json:"type"`   // step0, step1, step2, step3, step4, step5 (combined depth 0-5); when step0, no depth is merged
}

// Orderbook stores the orderbook data
type Orderbook struct {
	ID         int64               `json:"id"`
	Timetstamp int64               `json:"ts"`
	Bids       [][]decimal.Decimal `json:"bids"`
	Asks       [][]decimal.Decimal `json:"asks"`
}

// Trade stores the trade data
type Trade struct {
	ID        decimal.Decimal `json:"id"`
	Price     decimal.Decimal `json:"price"`
	Amount    decimal.Decimal `json:"amount"`
	Direction string          `json:"direction"`
	Timestamp int64           `json:"ts"`
}

// TradeHistory stores the the trade history data
type TradeHistory struct {
	ID        int64   `json:"id"`
	Timestamp int64   `json:"ts"`
	Trades    []Trade `json:"data"`
}

// Detail stores the ticker detail data
type Detail struct {
	Amount    decimal.Decimal `json:"amount"`
	Open      decimal.Decimal `json:"open"`
	Close     decimal.Decimal `json:"close"`
	High      decimal.Decimal `json:"high"`
	Timestamp int64           `json:"timestamp"`
	ID        int             `json:"id"`
	Count     int             `json:"count"`
	Low       decimal.Decimal `json:"low"`
	Volume    decimal.Decimal `json:"vol"`
}

// Symbol stores the symbol data
type Symbol struct {
	BaseCurrency    string `json:"base-currency"`
	QuoteCurrency   string `json:"quote-currency"`
	PricePrecision  int    `json:"price-precision"`
	AmountPrecision int    `json:"amount-precision"`
	SymbolPartition string `json:"symbol-partition"`
}

// Account stores the account data
type Account struct {
	ID     int64  `json:"id"`
	Type   string `json:"type"`
	State  string `json:"working"`
	UserID int64  `json:"user-id"`
}

// AccountBalance stores the user all account balance
type AccountBalance struct {
	ID                    int64                  `json:"id"`
	Type                  string                 `json:"type"`
	State                 string                 `json:"state"`
	AccountBalanceDetails []AccountBalanceDetail `json:"list"`
}

// AccountBalanceDetail stores the user account balance
type AccountBalanceDetail struct {
	Currency string          `json:"currency"`
	Type     string          `json:"type"`
	Balance  decimal.Decimal `json:"balance,string"`
}

// CancelOrderBatch stores the cancel order batch data
type CancelOrderBatch struct {
	Success []string `json:"success"`
	Failed  []struct {
		OrderID      int64  `json:"order-id,string"`
		ErrorCode    string `json:"err-code"`
		ErrorMessage string `json:"err-msg"`
	} `json:"failed"`
}

// OrderInfo stores the order info
type OrderInfo struct {
	ID              int    `json:"id"`
	Symbol          string `json:"symbol"`
	AccountID       int    `json:"account-id"`
	Amount          string `json:"amount"`
	Price           string `json:"price"`
	CreatedAt       int64  `json:"created-at"`
	Type            string `json:"type"`
	FieldAmount     string `json:"field-amount"`
	FieldCashAmount string `json:"field-cash-amount"`
	FieldFees       string `json:"field-fees"`
	FinishedAt      int64  `json:"finished-at"`
	UserID          int    `json:"user-id"`
	Source          string `json:"source"`
	State           string `json:"state"`
	CanceledAt      int    `json:"canceled-at"`
	Exchange        string `json:"exchange"`
	Batch           string `json:"batch"`
}

// OrderMatchInfo stores the order match info
type OrderMatchInfo struct {
	ID           int    `json:"id"`
	OrderID      int    `json:"order-id"`
	MatchID      int    `json:"match-id"`
	Symbol       string `json:"symbol"`
	Type         string `json:"type"`
	Source       string `json:"source"`
	Price        string `json:"price"`
	FilledAmount string `json:"filled-amount"`
	FilledFees   string `json:"filled-fees"`
	CreatedAt    int64  `json:"created-at"`
}

// MarginOrder stores the margin order info
type MarginOrder struct {
	Currency        string `json:"currency"`
	Symbol          string `json:"symbol"`
	AccruedAt       int64  `json:"accrued-at"`
	LoanAmount      string `json:"loan-amount"`
	LoanBalance     string `json:"loan-balance"`
	InterestBalance string `json:"interest-balance"`
	CreatedAt       int64  `json:"created-at"`
	InterestAmount  string `json:"interest-amount"`
	InterestRate    string `json:"interest-rate"`
	AccountID       int    `json:"account-id"`
	UserID          int    `json:"user-id"`
	UpdatedAt       int64  `json:"updated-at"`
	ID              int    `json:"id"`
	State           string `json:"state"`
}

// MarginAccountBalance stores the margin account balance info
type MarginAccountBalance struct {
	ID       int              `json:"id"`
	Type     string           `json:"type"`
	State    string           `json:"state"`
	Symbol   string           `json:"symbol"`
	FlPrice  string           `json:"fl-price"`
	FlType   string           `json:"fl-type"`
	RiskRate string           `json:"risk-rate"`
	List     []AccountBalance `json:"list"`
}

// SpotNewOrderRequestParams holds the params required to place
// an order
type SpotNewOrderRequestParams struct {
	AccountID int                           `json:"account-id"` // Account ID, obtained using the accounts method. Curency trades use the accountid of the ‘spot’ account; for loan asset transactions, please use the accountid of the ‘margin’ account.
	Amount    decimal.Decimal               `json:"amount"`     // The limit price indicates the quantity of the order, the market price indicates how much to buy when the order is paid, and the market price indicates how much the coin is sold when the order is sold.
	Price     decimal.Decimal               `json:"price"`      // Order price, market price does not use  this parameter
	Source    string                        `json:"source"`     // Order source, api: API call, margin-api: loan asset transaction
	Symbol    string                        `json:"symbol"`     // The symbol to use; example btcusdt, bccbtc......
	Type      SpotNewOrderRequestParamsType `json:"type"`       // 订单类型, buy-market: 市价买, sell-market: 市价卖, buy-limit: 限价买, sell-limit: 限价卖
}

// SpotNewOrderRequestParamsType order type
type SpotNewOrderRequestParamsType string

var (
	// SpotNewOrderRequestTypeBuyMarket buy market order
	SpotNewOrderRequestTypeBuyMarket = SpotNewOrderRequestParamsType("buy-market")

	// SpotNewOrderRequestTypeSellMarket sell market order
	SpotNewOrderRequestTypeSellMarket = SpotNewOrderRequestParamsType("sell-market")

	// SpotNewOrderRequestTypeBuyLimit buy limit order
	SpotNewOrderRequestTypeBuyLimit = SpotNewOrderRequestParamsType("buy-limit")

	// SpotNewOrderRequestTypeSellLimit sell lmit order
	SpotNewOrderRequestTypeSellLimit = SpotNewOrderRequestParamsType("sell-limit")
)

//-----------

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol string       // Symbol to be used; example btcusdt, bccbtc......
	Period TimeInterval // Kline time interval; 1min, 5min, 15min......
	Size   int          // Size; [1-2000]
}

// TimeInterval base type
type TimeInterval string

// TimeInterval vars
var (
	TimeIntervalMinute         = TimeInterval("1min")
	TimeIntervalFiveMinutes    = TimeInterval("5min")
	TimeIntervalFifteenMinutes = TimeInterval("15min")
	TimeIntervalThirtyMinutes  = TimeInterval("30min")
	TimeIntervalHour           = TimeInterval("60min")
	TimeIntervalDay            = TimeInterval("1day")
	TimeIntervalWeek           = TimeInterval("1week")
	TimeIntervalMohth          = TimeInterval("1mon")
	TimeIntervalYear           = TimeInterval("1year")
)
