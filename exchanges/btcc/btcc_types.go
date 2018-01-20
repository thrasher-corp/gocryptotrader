package btcc

// Response is the generalized response type
type Response struct {
	Ticker Ticker `json:"ticker"`
}

// Ticker holds basic ticker information
type Ticker struct {
	BidPrice           float64 `json:"BidPrice"`
	AskPrice           float64 `json:"AskPrice"`
	Open               float64 `json:"Open"`
	High               float64 `json:"High"`
	Low                float64 `json:"Low"`
	Last               float64 `json:"Last"`
	LastQuantity       float64 `json:"LastQuantity"`
	PrevCls            float64 `json:"PrevCls"`
	Volume             float64 `json:"Volume"`
	Volume24H          float64 `json:"Volume24H"`
	Timestamp          int64   `json:"Timestamp"`
	ExecutionLimitDown float64 `json:"ExecutionLimitDown"`
	ExecutionLimitUp   float64 `json:"ExecutionLimitUp"`
}

// Trade holds executed trade data
type Trade struct {
	ID        int64   `json:"Id"`
	Timestamp int64   `json:"Timestamp"`
	Price     float64 `json:"Price"`
	Quantity  float64 `json:"Quantity"`
	Side      string  `json:"Side"`
}

// Orderbook holds orderbook data
type Orderbook struct {
	Bids [][]float64 `json:"bids"`
	Asks [][]float64 `json:"asks"`
	Date int64       `json:"date"`
}

// Profile holds profile information
type Profile struct {
	Username             string
	TradePasswordEnabled bool    `json:"trade_password_enabled,bool"`
	OTPEnabled           bool    `json:"otp_enabled,bool"`
	TradeFee             float64 `json:"trade_fee"`
	TradeFeeCNYLTC       float64 `json:"trade_fee_cnyltc"`
	TradeFeeBTCLTC       float64 `json:"trade_fee_btcltc"`
	DailyBTCLimit        float64 `json:"daily_btc_limit"`
	DailyLTCLimit        float64 `json:"daily_ltc_limit"`
	BTCDespoitAddress    string  `json:"btc_despoit_address"`
	BTCWithdrawalAddress string  `json:"btc_withdrawal_address"`
	LTCDepositAddress    string  `json:"ltc_deposit_address"`
	LTCWithdrawalAddress string  `json:"ltc_withdrawal_request"`
	APIKeyPermission     int64   `json:"api_key_permission"`
}

// CurrencyGeneric holds currency information
type CurrencyGeneric struct {
	Currency      string
	Symbol        string
	Amount        string
	AmountInt     int64   `json:"amount_integer"`
	AmountDecimal float64 `json:"amount_decimal"`
}

// Order holds order information
type Order struct {
	ID         int64
	Type       string
	Price      float64
	Currency   string
	Amount     float64
	AmountOrig float64 `json:"amount_original"`
	Date       int64
	Status     string
	Detail     OrderDetail
}

// OrderDetail holds order detail information
type OrderDetail struct {
	Dateline int64
	Price    float64
	Amount   float64
}

// Withdrawal holds withdrawal transaction information
type Withdrawal struct {
	ID          int64
	Address     string
	Currency    string
	Amount      float64
	Date        int64
	Transaction string
	Status      string
}

// Deposit holds deposit address information
type Deposit struct {
	ID       int64
	Address  string
	Currency string
	Amount   float64
	Date     int64
	Status   string
}

// BidAsk holds bid and ask information
type BidAsk struct {
	Price  float64
	Amount float64
}

// Depth holds order book depth
type Depth struct {
	Bid []BidAsk
	Ask []BidAsk
}

// Transaction holds transaction information
type Transaction struct {
	ID        int64
	Type      string
	BTCAmount float64 `json:"btc_amount"`
	LTCAmount float64 `json:"ltc_amount"`
	CNYAmount float64 `json:"cny_amount"`
	Date      int64
}

// IcebergOrder holds iceberg lettuce
type IcebergOrder struct {
	ID              int64
	Type            string
	Price           float64
	Market          string
	Amount          float64
	AmountOrig      float64 `json:"amount_original"`
	DisclosedAmount float64 `json:"disclosed_amount"`
	Variance        float64
	Date            int64
	Status          string
}

// StopOrder holds stop order information
type StopOrder struct {
	ID          int64
	Type        string
	StopPrice   float64 `json:"stop_price"`
	TrailingAmt float64 `json:"trailing_amount"`
	TrailingPct float64 `json:"trailing_percentage"`
	Price       float64
	Market      string
	Amount      float64
	Date        int64
	Status      string
	OrderID     int64 `json:"order_id"`
}

// WebsocketOrder holds websocket order information
type WebsocketOrder struct {
	Price       float64 `json:"price"`
	TotalAmount float64 `json:"totalamount"`
	Type        string  `json:"type"`
}

// WebsocketGroupOrder holds websocket group order book information
type WebsocketGroupOrder struct {
	Asks   []WebsocketOrder `json:"ask"`
	Bids   []WebsocketOrder `json:"bid"`
	Market string           `json:"market"`
}

// WebsocketTrade holds websocket trade information
type WebsocketTrade struct {
	Amount  float64 `json:"amount"`
	Date    float64 `json:"date"`
	Market  string  `json:"market"`
	Price   float64 `json:"price"`
	TradeID float64 `json:"trade_id"`
	Type    string  `json:"type"`
}

// WebsocketTicker holds websocket ticker information
type WebsocketTicker struct {
	Buy       float64 `json:"buy"`
	Date      float64 `json:"date"`
	High      float64 `json:"high"`
	Last      float64 `json:"last"`
	Low       float64 `json:"low"`
	Market    string  `json:"market"`
	Open      float64 `json:"open"`
	PrevClose float64 `json:"prev_close"`
	Sell      float64 `json:"sell"`
	Volume    float64 `json:"vol"`
	Vwap      float64 `json:"vwap"`
}
