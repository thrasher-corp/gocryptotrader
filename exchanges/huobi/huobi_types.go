package huobi

import "github.com/thrasher-/gocryptotrader/currency"

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
	ID     int64   `json:"id"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Amount float64 `json:"amount"`
	Vol    float64 `json:"vol"`
	Count  int     `json:"count"`
}

// CancelOpenOrdersBatch stores open order batch response data
type CancelOpenOrdersBatch struct {
	Data struct {
		FailedCount  int `json:"failed-count"`
		NextID       int `json:"next-id"`
		SuccessCount int `json:"success-count"`
	} `json:"data"`
	Status       string `json:"status"`
	ErrorMessage string `json:"err-msg"`
}

// DetailMerged stores the ticker detail merged data
type DetailMerged struct {
	Detail
	Version int       `json:"version"`
	Ask     []float64 `json:"ask"`
	Bid     []float64 `json:"bid"`
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
	ID         int64       `json:"id"`
	Timetstamp int64       `json:"ts"`
	Bids       [][]float64 `json:"bids"`
	Asks       [][]float64 `json:"asks"`
}

// Trade stores the trade data
type Trade struct {
	ID        float64 `json:"id"`
	Price     float64 `json:"price"`
	Amount    float64 `json:"amount"`
	Direction string  `json:"direction"`
	Timestamp int64   `json:"ts"`
}

// TradeHistory stores the the trade history data
type TradeHistory struct {
	ID        int64   `json:"id"`
	Timestamp int64   `json:"ts"`
	Trades    []Trade `json:"data"`
}

// Detail stores the ticker detail data
type Detail struct {
	Amount    float64 `json:"amount"`
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Timestamp int64   `json:"timestamp"`
	ID        int     `json:"id"`
	Count     int     `json:"count"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"vol"`
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
	State  string `json:"state"`
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
	Currency string  `json:"currency"`
	Type     string  `json:"type"`
	Balance  float64 `json:"balance,string"`
}

// AggregatedBalance stores balances of all the sub-account
type AggregatedBalance struct {
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance,string"`
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
	ID               int     `json:"id"`
	Symbol           string  `json:"symbol"`
	AccountID        float64 `json:"account-id"`
	Amount           float64 `json:"amount,string"`
	Price            float64 `json:"price,string"`
	CreatedAt        int64   `json:"created-at"`
	Type             string  `json:"type"`
	FieldAmount      float64 `json:"field-amount,string"`
	FieldCashAmount  float64 `json:"field-cash-amount,string"`
	Fieldees         float64 `json:"field-fees,string"`
	FilledAmount     float64 `json:"filled-amount,string"`
	FilledCashAmount float64 `json:"filled-cash-amount,string"`
	FilledFees       float64 `json:"filled-fees,string"`
	FinishedAt       int64   `json:"finished-at"`
	UserID           int     `json:"user-id"`
	Source           string  `json:"source"`
	State            string  `json:"state"`
	CanceledAt       int     `json:"canceled-at"`
	Exchange         string  `json:"exchange"`
	Batch            string  `json:"batch"`
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
	AccountID int                           `json:"account-id,string"` // Account ID, obtained using the accounts method. Curency trades use the accountid of the ‘spot’ account; for loan asset transactions, please use the accountid of the ‘margin’ account.
	Amount    float64                       `json:"amount"`            // The limit price indicates the quantity of the order, the market price indicates how much to buy when the order is paid, and the market price indicates how much the coin is sold when the order is sold.
	Price     float64                       `json:"price"`             // Order price, market price does not use  this parameter
	Source    string                        `json:"source"`            // Order source, api: API call, margin-api: loan asset transaction
	Symbol    string                        `json:"symbol"`            // The symbol to use; example btcusdt, bccbtc......
	Type      SpotNewOrderRequestParamsType `json:"type"`              // 订单类型, buy-market: 市价买, sell-market: 市价卖, buy-limit: 限价买, sell-limit: 限价卖
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

// WsRequest defines a request data structure
type WsRequest struct {
	Topic             string `json:"req,omitempty"`
	Subscribe         string `json:"sub,omitempty"`
	Unsubscribe       string `json:"unsub,omitempty"`
	ClientGeneratedID string `json:"id,omitempty"`
}

// WsResponse defines a response from the websocket connection when there
// is an error
type WsResponse struct {
	TS           int64       `json:"ts"`
	Status       string      `json:"status"`
	ErrorCode    interface{} `json:"err-code"`
	ErrorMessage string      `json:"err-msg"`
	Ping         int64       `json:"ping"`
	Channel      string      `json:"ch"`
	Subscribed   string      `json:"subbed"`
}

// WsHeartBeat defines a heartbeat request
type WsHeartBeat struct {
	ClientNonce int64 `json:"ping"`
}

// WsDepth defines market depth websocket response
type WsDepth struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		Bids      []interface{} `json:"bids"`
		Asks      []interface{} `json:"asks"`
		Timestamp int64         `json:"ts"`
		Version   int64         `json:"version"`
	} `json:"tick"`
}

// WsKline defines market kline websocket response
type WsKline struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		ID     int64   `json:"id"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
		Volume float64 `json:"vol"`
		Count  int64   `json:"count"`
	}
}

// WsTrade defines market trade websocket response
type WsTrade struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		ID        int64 `json:"id"`
		Timestamp int64 `json:"ts"`
		Data      []struct {
			Amount    float64 `json:"amount"`
			Timestamp int64   `json:"ts"`
			ID        float64 `json:"id"`
			Price     float64 `json:"price"`
			Direction string  `json:"direction"`
		} `json:"data"`
	}
}

// WsAuthenticationRequest data for login
type WsAuthenticationRequest struct {
	Op               string `json:"op"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
}

// WsMessage defines read data from the websocket connection
type WsMessage struct {
	Raw []byte
	URL string
}

// WsAuthenticatedSubscriptionRequest request for subscription on authenticated connection
type WsAuthenticatedSubscriptionRequest struct {
	Op               string `json:"op"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
	Topic            string `json:"topic"`
}

// WsAuthenticatedAccountsListRequest request for account list authenticated connection
type WsAuthenticatedAccountsListRequest struct {
	Op               string        `json:"op"`
	AccessKeyID      string        `json:"AccessKeyId"`
	SignatureMethod  string        `json:"SignatureMethod"`
	SignatureVersion string        `json:"SignatureVersion"`
	Timestamp        string        `json:"Timestamp"`
	Signature        string        `json:"Signature"`
	Topic            string        `json:"topic"`
	Symbol           currency.Pair `json:"symbol"`
}

// WsAuthenticatedOrderDetailsRequest request for order details authenticated connection
type WsAuthenticatedOrderDetailsRequest struct {
	Op               string `json:"op"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
	Topic            string `json:"topic"`
	OrderID          string `json:"order-id"`
}

// WsAuthenticatedOrdersListRequest request for orderslist authenticated connection
type WsAuthenticatedOrdersListRequest struct {
	Op               string        `json:"op"`
	AccessKeyID      string        `json:"AccessKeyId"`
	SignatureMethod  string        `json:"SignatureMethod"`
	SignatureVersion string        `json:"SignatureVersion"`
	Timestamp        string        `json:"Timestamp"`
	Signature        string        `json:"Signature"`
	Topic            string        `json:"topic"`
	States           string        `json:"states"`
	AccountID        int64         `json:"account-id"`
	Symbol           currency.Pair `json:"symbol"`
}

// WsAuthenticatedDataResponse response from authenticated connection
type WsAuthenticatedDataResponse struct {
	Op           string `json:"op,omitempty"`
	Ts           int64  `json:"ts,omitempty"`
	Topic        string `json:"topic,omitempty"`
	ErrorCode    int64  `json:"err-code,omitempty"`
	ErrorMessage string `json:"err-msg,omitempty"`
	Ping         int64  `json:"ping,omitempty"`
	CID          string `json:"cid,omitempty"`
}

// WsAuthenticatedAccountsResponse response from Accounts authenticated subscription
type WsAuthenticatedAccountsResponse struct {
	WsAuthenticatedDataResponse
	Data WsAuthenticatedAccountsResponseData `json:"data"`
}

// WsAuthenticatedAccountsResponseData account data
type WsAuthenticatedAccountsResponseData struct {
	Event string                                    `json:"event"`
	List  []WsAuthenticatedAccountsResponseDataList `json:"list"`
}

// WsAuthenticatedAccountsResponseDataList detailed account data
type WsAuthenticatedAccountsResponseDataList struct {
	AccountID int64   `json:"account-id"`
	Currency  string  `json:"currency"`
	Type      string  `json:"type"`
	Balance   float64 `json:"balance,string"`
}

// WsAuthenticatedOrdersUpdateResponse response from OrdersUpdate authenticated subscription
type WsAuthenticatedOrdersUpdateResponse struct {
	WsAuthenticatedDataResponse
	Data WsAuthenticatedOrdersUpdateResponseData `json:"data"`
}

// WsAuthenticatedOrdersUpdateResponseData order  updatedata
type WsAuthenticatedOrdersUpdateResponseData struct {
	UnfilledAmount   float64       `json:"unfilled-amount,string"`
	FilledAmount     float64       `json:"filled-amount,string"`
	Price            float64       `json:"price,string"`
	OrderID          int64         `json:"order-id"`
	Symbol           currency.Pair `json:"symbol"`
	MatchID          int64         `json:"match-id"`
	FilledCashAmount float64       `json:"filled-cash-amount,string"`
	Role             string        `json:"role"`
	OrderState       string        `json:"order-state"`
}

// WsAuthenticatedOrdersResponse response from Orders authenticated subscription
type WsAuthenticatedOrdersResponse struct {
	WsAuthenticatedDataResponse
	Data []WsAuthenticatedOrdersResponseData `json:"data"`
}

// WsAuthenticatedOrdersResponseData order data
type WsAuthenticatedOrdersResponseData struct {
	SeqID            int64         `json:"seq-id"`
	OrderID          int64         `json:"order-id"`
	Symbol           currency.Pair `json:"symbol"`
	AccountID        int64         `json:"account-id"`
	OrderAmount      float64       `json:"order-amount,string"`
	OrderPrice       float64       `json:"order-price,string"`
	CreatedAt        int64         `json:"created-at"`
	OrderType        string        `json:"order-type"`
	OrderSource      string        `json:"order-source"`
	OrderState       string        `json:"order-state"`
	Role             string        `json:"role"`
	Price            float64       `json:"price,string"`
	FilledAmount     float64       `json:"filled-amount,string"`
	UnfilledAmount   float64       `json:"unfilled-amount,string"`
	FilledCashAmount float64       `json:"filled-cash-amount,string"`
	FilledFees       float64       `json:"filled-fees,string"`
}

// WsAuthenticatedAccountsListResponse response from AccountsList authenticated endpoint
type WsAuthenticatedAccountsListResponse struct {
	WsAuthenticatedDataResponse
	Data []WsAuthenticatedAccountsListResponseData `json:"data"`
}

// WsAuthenticatedAccountsListResponseData account data
type WsAuthenticatedAccountsListResponseData struct {
	ID    int64                                         `json:"id"`
	Type  string                                        `json:"type"`
	State string                                        `json:"state"`
	List  []WsAuthenticatedAccountsListResponseDataList `json:"list"`
}

// WsAuthenticatedAccountsListResponseDataList detailed account data
type WsAuthenticatedAccountsListResponseDataList struct {
	Currency string  `json:"currency"`
	Type     string  `json:"type"`
	Balance  float64 `json:"balance,string"`
}

// WsAuthenticatedOrdersListResponse response from OrdersList authenticated endpoint
type WsAuthenticatedOrdersListResponse struct {
	WsAuthenticatedDataResponse
	Data []WsAuthenticatedOrdersListResponseData `json:"data"`
}

// WsAuthenticatedOrdersListResponseData contains order details
type WsAuthenticatedOrdersListResponseData struct {
	ID               int64         `json:"id"`
	Symbol           currency.Pair `json:"symbol"`
	AccountID        int64         `json:"account-id"`
	Amount           float64       `json:"amount,string"`
	Price            float64       `json:"price,string"`
	CreatedAt        int64         `json:"created-at"`
	Type             string        `json:"type"`
	FilledAmount     float64       `json:"filled-amount,string"`
	FilledCashAmount float64       `json:"filled-cash-amount,string"`
	FilledFees       float64       `json:"filled-fees,string"`
	FinishedAt       int64         `json:"finished-at"`
	Source           string        `json:"source"`
	State            string        `json:"state"`
	CanceledAt       int64         `json:"canceled-at"`
}

// WsAuthenticatedOrderDetailResponse response from OrderDetail authenticated endpoint
type WsAuthenticatedOrderDetailResponse struct {
	WsAuthenticatedDataResponse
	Data WsAuthenticatedOrdersListResponseData `json:"data"`
}
