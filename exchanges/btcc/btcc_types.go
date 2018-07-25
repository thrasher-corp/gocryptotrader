package btcc

import "github.com/kempeng/gocryptotrader/decimal"

// Response is the generalized response type
type Response struct {
	Ticker Ticker `json:"ticker"`
}

// Ticker holds basic ticker information
type Ticker struct {
	BidPrice           decimal.Decimal `json:"BidPrice"`
	AskPrice           decimal.Decimal `json:"AskPrice"`
	Open               decimal.Decimal `json:"Open"`
	High               decimal.Decimal `json:"High"`
	Low                decimal.Decimal `json:"Low"`
	Last               decimal.Decimal `json:"Last"`
	LastQuantity       decimal.Decimal `json:"LastQuantity"`
	PrevCls            decimal.Decimal `json:"PrevCls"`
	Volume             decimal.Decimal `json:"Volume"`
	Volume24H          decimal.Decimal `json:"Volume24H"`
	Timestamp          int64           `json:"Timestamp"`
	ExecutionLimitDown decimal.Decimal `json:"ExecutionLimitDown"`
	ExecutionLimitUp   decimal.Decimal `json:"ExecutionLimitUp"`
}

// Trade holds executed trade data
type Trade struct {
	ID        int64           `json:"Id"`
	Timestamp int64           `json:"Timestamp"`
	Price     decimal.Decimal `json:"Price"`
	Quantity  decimal.Decimal `json:"Quantity"`
	Side      string          `json:"Side"`
}

// Orderbook holds orderbook data
type Orderbook struct {
	Bids [][]decimal.Decimal `json:"bids"`
	Asks [][]decimal.Decimal `json:"asks"`
	Date int64               `json:"date"`
}

// Profile holds profile information
type Profile struct {
	Username             string
	TradePasswordEnabled bool            `json:"trade_password_enabled,bool"`
	OTPEnabled           bool            `json:"otp_enabled,bool"`
	TradeFee             decimal.Decimal `json:"trade_fee"`
	TradeFeeCNYLTC       decimal.Decimal `json:"trade_fee_cnyltc"`
	TradeFeeBTCLTC       decimal.Decimal `json:"trade_fee_btcltc"`
	DailyBTCLimit        decimal.Decimal `json:"daily_btc_limit"`
	DailyLTCLimit        decimal.Decimal `json:"daily_ltc_limit"`
	BTCDespoitAddress    string          `json:"btc_despoit_address"`
	BTCWithdrawalAddress string          `json:"btc_withdrawal_address"`
	LTCDepositAddress    string          `json:"ltc_deposit_address"`
	LTCWithdrawalAddress string          `json:"ltc_withdrawal_request"`
	APIKeyPermission     int64           `json:"api_key_permission"`
}

// CurrencyGeneric holds currency information
type CurrencyGeneric struct {
	Currency      string
	Symbol        string
	Amount        string
	AmountInt     int64           `json:"amount_integer"`
	AmountDecimal decimal.Decimal `json:"amount_decimal"`
}

// Order holds order information
type Order struct {
	ID         int64
	Type       string
	Price      decimal.Decimal
	Currency   string
	Amount     decimal.Decimal
	AmountOrig decimal.Decimal `json:"amount_original"`
	Date       int64
	Status     string
	Detail     OrderDetail
}

// OrderDetail holds order detail information
type OrderDetail struct {
	Dateline int64
	Price    decimal.Decimal
	Amount   decimal.Decimal
}

// Withdrawal holds withdrawal transaction information
type Withdrawal struct {
	ID          int64
	Address     string
	Currency    string
	Amount      decimal.Decimal
	Date        int64
	Transaction string
	Status      string
}

// Deposit holds deposit address information
type Deposit struct {
	ID       int64
	Address  string
	Currency string
	Amount   decimal.Decimal
	Date     int64
	Status   string
}

// BidAsk holds bid and ask information
type BidAsk struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
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
	BTCAmount decimal.Decimal `json:"btc_amount"`
	LTCAmount decimal.Decimal `json:"ltc_amount"`
	CNYAmount decimal.Decimal `json:"cny_amount"`
	Date      int64
}

// IcebergOrder holds iceberg lettuce
type IcebergOrder struct {
	ID              int64
	Type            string
	Price           decimal.Decimal
	Market          string
	Amount          decimal.Decimal
	AmountOrig      decimal.Decimal `json:"amount_original"`
	DisclosedAmount decimal.Decimal `json:"disclosed_amount"`
	Variance        decimal.Decimal
	Date            int64
	Status          string
}

// StopOrder holds stop order information
type StopOrder struct {
	ID          int64
	Type        string
	StopPrice   decimal.Decimal `json:"stop_price"`
	TrailingAmt decimal.Decimal `json:"trailing_amount"`
	TrailingPct decimal.Decimal `json:"trailing_percentage"`
	Price       decimal.Decimal
	Market      string
	Amount      decimal.Decimal
	Date        int64
	Status      string
	OrderID     int64 `json:"order_id"`
}

// WebsocketOrder holds websocket order information
type WebsocketOrder struct {
	Price       decimal.Decimal `json:"price"`
	TotalAmount decimal.Decimal `json:"totalamount"`
	Type        string          `json:"type"`
}

// WebsocketGroupOrder holds websocket group order book information
type WebsocketGroupOrder struct {
	Asks   []WebsocketOrder `json:"ask"`
	Bids   []WebsocketOrder `json:"bid"`
	Market string           `json:"market"`
}

// WebsocketTrade holds websocket trade information
type WebsocketTrade struct {
	Amount  decimal.Decimal `json:"amount"`
	Date    float64         `json:"date"`
	Market  string          `json:"market"`
	Price   decimal.Decimal `json:"price"`
	TradeID float64         `json:"trade_id"`
	Type    string          `json:"type"`
}

// WebsocketTicker holds websocket ticker information
type WebsocketTicker struct {
	Buy       decimal.Decimal `json:"buy"`
	Date      decimal.Decimal `json:"date"`
	High      decimal.Decimal `json:"high"`
	Last      decimal.Decimal `json:"last"`
	Low       decimal.Decimal `json:"low"`
	Market    string          `json:"market"`
	Open      decimal.Decimal `json:"open"`
	PrevClose decimal.Decimal `json:"prev_close"`
	Sell      decimal.Decimal `json:"sell"`
	Volume    decimal.Decimal `json:"vol"`
	Vwap      decimal.Decimal `json:"vwap"`
}
