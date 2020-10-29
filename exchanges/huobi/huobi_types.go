package huobi

// Response stores the Huobi response information
type Response struct {
	Status       string `json:"status"`
	Channel      string `json:"ch"`
	Timestamp    int64  `json:"ts"`
	ErrorCode    string `json:"err-code"`
	ErrorMessage string `json:"err-msg"`
}

// ResponseV2 stores the Huobi generic response info
type ResponseV2 struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

// KlineItem stores a kline item
type KlineItem struct {
	ID     int64   `json:"id"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Amount float64 `json:"amount"`
	Volume float64 `json:"vol"`
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
	Version int64     `json:"version"`
	Ask     []float64 `json:"ask"`
	Bid     []float64 `json:"bid"`
}

// Tickers contain all tickers
type Tickers struct {
	Data []Ticker `json:"data"`
}

// Ticker latest ticker data
type Ticker struct {
	Amount float64 `json:"amount"`
	Close  float64 `json:"close"`
	Count  int64   `json:"count"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Open   float64 `json:"open"`
	Symbol string  `json:"symbol"`
	Volume float64 `json:"vol"`
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
	TradeID   float64 `json:"trade-id"`
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
	ID        int64   `json:"id"`
	Count     int     `json:"count"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"vol"`
}

// Symbol stores the symbol data
type Symbol struct {
	BaseCurrency       string  `json:"base-currency"`
	QuoteCurrency      string  `json:"quote-currency"`
	PricePrecision     int     `json:"price-precision"`
	AmountPrecision    int     `json:"amount-precision"`
	SymbolPartition    string  `json:"symbol-partition"`
	Innovation         string  `json:"innovation"`
	State              string  `json:"state"`
	ValuePrecision     int     `json:"value-precision"`
	MinimumOrderAmount float64 `json:"min-order-amt"`
	MaximumOrderAmount float64 `json:"max-order-amt"`
	MinimumOrderValue  float64 `json:"min-order-value"`
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
	ID               int64   `json:"id"`
	Symbol           string  `json:"symbol"`
	AccountID        int64   `json:"account-id"`
	Amount           float64 `json:"amount,string"`
	Price            float64 `json:"price,string"`
	CreatedAt        int64   `json:"created-at"`
	Type             string  `json:"type"`
	FieldAmount      float64 `json:"field-amount,string"`
	FieldCashAmount  float64 `json:"field-cash-amount,string"`
	FilledAmount     float64 `json:"filled-amount,string"`
	FilledCashAmount float64 `json:"filled-cash-amount,string"`
	FilledFees       float64 `json:"filled-fees,string"`
	FinishedAt       int64   `json:"finished-at"`
	UserID           int64   `json:"user-id"`
	Source           string  `json:"source"`
	State            string  `json:"state"`
	CanceledAt       int64   `json:"canceled-at"`
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

// DepositAddress stores the users deposit address info
type DepositAddress struct {
	Currency   string `json:"currency"`
	Address    string `json:"address"`
	AddressTag string `json:"addressTag"`
	Chain      string `json:"chain"`
}

// ChainQuota stores the users currency chain quota
type ChainQuota struct {
	Chain                         string  `json:"chain"`
	MaxWithdrawAmount             float64 `json:"maxWithdrawAmt,string"`
	WithdrawQuotaPerDay           float64 `json:"withdrawQuotaPerDay,string"`
	RemainingWithdrawQuotaPerDay  float64 `json:"remainWithdrawQuotaPerDay,string"`
	WithdrawQuotaPerYear          float64 `json:"withdrawQuotaPerYear,string"`
	RemainingWithdrawQuotaPerYear float64 `json:"remainWithdrawQuotaPerYear,string"`
	WithdrawQuotaTotal            float64 `json:"withdrawQuotaTotal,string"`
	RemainingWithdrawQuotaTotal   float64 `json:"remainWithdrawQuotaTotal,string"`
}

// WithdrawQuota stores the users withdraw quotas
type WithdrawQuota struct {
	Currency string       `json:"currency"`
	Chains   []ChainQuota `json:"chains"`
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
	Symbol string // Symbol to be used; example btcusdt, bccbtc......
	Period string // Kline time interval; 1min, 5min, 15min......
	Size   int    // Size; [1-2000]
}

// WsRequest defines a request data structure
type WsRequest struct {
	Topic       string `json:"req,omitempty"`
	Subscribe   string `json:"sub,omitempty"`
	Unsubscribe string `json:"unsub,omitempty"`
	ClientID    int64  `json:"cid,string,omitempty"`
}

// WsResponse defines a response from the websocket connection when there
// is an error
type WsResponse struct {
	Op     string `json:"op"`
	TS     int64  `json:"ts"`
	Status string `json:"status"`
	// ErrorCode returns either an integer or a string
	ErrorCode    interface{} `json:"err-code"`
	ErrorMessage string      `json:"err-msg"`
	Ping         int64       `json:"ping"`
	Channel      string      `json:"ch"`
	Rep          string      `json:"rep"`
	Topic        string      `json:"topic"`
	Subscribed   string      `json:"subbed"`
	UnSubscribed string      `json:"unsubbed"`
	ClientID     int64       `json:"cid,string"`
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
		Bids      [][]interface{} `json:"bids"`
		Asks      [][]interface{} `json:"asks"`
		Timestamp int64           `json:"ts"`
		Version   int64           `json:"version"`
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
	} `json:"tick"`
}

// WsTick stores websocket ticker data
type WsTick struct {
	Channel   string `json:"ch"`
	Rep       string `json:"rep"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		Amount    float64 `json:"amount"`
		Close     float64 `json:"close"`
		Count     float64 `json:"count"`
		High      float64 `json:"high"`
		ID        float64 `json:"id"`
		Low       float64 `json:"low"`
		Open      float64 `json:"open"`
		Timestamp float64 `json:"ts"`
		Volume    float64 `json:"vol"`
	} `json:"tick"`
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
			TradeID   float64 `json:"tradeId"`
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
	ClientID         int64  `json:"cid,string,omitempty"`
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
	ClientID         int64  `json:"cid,string,omitempty"`
}

// WsAuthenticatedAccountsListRequest request for account list authenticated connection
type WsAuthenticatedAccountsListRequest struct {
	Op               string `json:"op"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
	Topic            string `json:"topic"`
	Symbol           string `json:"symbol"`
	ClientID         int64  `json:"cid,string,omitempty"`
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
	ClientID         int64  `json:"cid,string,omitempty"`
}

// WsAuthenticatedOrdersListRequest request for orderslist authenticated connection
type WsAuthenticatedOrdersListRequest struct {
	Op               string `json:"op"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
	Topic            string `json:"topic"`
	States           string `json:"states"`
	AccountID        int64  `json:"account-id"`
	Symbol           string `json:"symbol"`
	ClientID         int64  `json:"cid,string,omitempty"`
}

// WsAuthenticatedAccountsResponse response from Accounts authenticated subscription
type WsAuthenticatedAccountsResponse struct {
	WsResponse
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
	WsResponse
	Data WsAuthenticatedOrdersUpdateResponseData `json:"data"`
}

// WsAuthenticatedOrdersUpdateResponseData order  update data
type WsAuthenticatedOrdersUpdateResponseData struct {
	UnfilledAmount   float64 `json:"unfilled-amount,string"`
	FilledAmount     float64 `json:"filled-amount,string"`
	Price            float64 `json:"price,string"`
	OrderID          int64   `json:"order-id"`
	Symbol           string  `json:"symbol"`
	MatchID          int64   `json:"match-id"`
	FilledCashAmount float64 `json:"filled-cash-amount,string"`
	Role             string  `json:"role"`
	OrderState       string  `json:"order-state"`
	OrderType        string  `json:"order-type"`
}

// WsAuthenticatedOrdersResponse response from Orders authenticated subscription
type WsAuthenticatedOrdersResponse struct {
	WsResponse
	Data []WsAuthenticatedOrdersResponseData `json:"data"`
}

// WsOldOrderUpdate response from Orders authenticated subscription
type WsOldOrderUpdate struct {
	WsResponse
	Data WsAuthenticatedOrdersResponseData `json:"data"`
}

// WsAuthenticatedOrdersResponseData order data
type WsAuthenticatedOrdersResponseData struct {
	SeqID            int64   `json:"seq-id"`
	OrderID          int64   `json:"order-id"`
	Symbol           string  `json:"symbol"`
	AccountID        int64   `json:"account-id"`
	OrderAmount      float64 `json:"order-amount,string"`
	OrderPrice       float64 `json:"order-price,string"`
	CreatedAt        int64   `json:"created-at"`
	OrderType        string  `json:"order-type"`
	OrderSource      string  `json:"order-source"`
	OrderState       string  `json:"order-state"`
	Role             string  `json:"role"`
	Price            float64 `json:"price,string"`
	FilledAmount     float64 `json:"filled-amount,string"`
	UnfilledAmount   float64 `json:"unfilled-amount,string"`
	FilledCashAmount float64 `json:"filled-cash-amount,string"`
	FilledFees       float64 `json:"filled-fees,string"`
}

// WsAuthenticatedAccountsListResponse response from AccountsList authenticated endpoint
type WsAuthenticatedAccountsListResponse struct {
	WsResponse
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
	WsResponse
	Data []OrderInfo `json:"data"`
}

// WsAuthenticatedOrderDetailResponse response from OrderDetail authenticated endpoint
type WsAuthenticatedOrderDetailResponse struct {
	WsResponse
	Data OrderInfo `json:"data"`
}

// WsPong sent for pong messages
type WsPong struct {
	Pong int64 `json:"pong"`
}

type wsKlineResponse struct {
	Data []struct {
		Amount float64 `json:"amount"`
		Close  float64 `json:"close"`
		Count  float64 `json:"count"`
		High   float64 `json:"high"`
		ID     int64   `json:"id"`
		Low    float64 `json:"low"`
		Open   float64 `json:"open"`
		Volume float64 `json:"vol"`
	} `json:"data"`
	Rep    string `json:"rep"`
	Status string `json:"status"`
}

type authenticationPing struct {
	OP string `json:"op"`
	TS int64  `json:"ts"`
}
