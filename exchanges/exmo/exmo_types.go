package exmo

import "github.com/shopspring/decimal"

// Trades holds trade data
type Trades struct {
	TradeID  int64           `json:"trade_id"`
	Type     string          `json:"string"`
	Quantity decimal.Decimal `json:"quantity,string"`
	Price    decimal.Decimal `json:"price,string"`
	Amount   decimal.Decimal `json:"amount,string"`
	Date     int64           `json:"date"`
}

// Orderbook holds the orderbook data
type Orderbook struct {
	AskQuantity decimal.Decimal `json:"ask_quantity,string"`
	AskAmount   decimal.Decimal `json:"ask_amount,string"`
	AskTop      decimal.Decimal `json:"ask_top,string"`
	BidQuantity decimal.Decimal `json:"bid_quantity,string"`
	BidTop      decimal.Decimal `json:"bid_top,string"`
	Ask         [][]string      `json:"ask"`
	Bid         [][]string      `json:"bid"`
}

// Ticker holds the ticker data
type Ticker struct {
	Buy           decimal.Decimal `json:"buy_price,string"`
	Sell          decimal.Decimal `json:"sell_price,string"`
	Last          decimal.Decimal `json:"last_trade,string"`
	High          decimal.Decimal `json:"high,string"`
	Low           decimal.Decimal `json:"low,string"`
	Average       decimal.Decimal `json:"average,string"`
	Volume        decimal.Decimal `json:"vol,string"`
	VolumeCurrent decimal.Decimal `json:"vol_curr,string"`
	Updated       int64           `json:"updated"`
}

// PairSettings holds the pair settings
type PairSettings struct {
	MinQuantity decimal.Decimal `json:"min_quantity,string"`
	MaxQuantity decimal.Decimal `json:"max_quantity,string"`
	MinPrice    decimal.Decimal `json:"min_price,string"`
	MaxPrice    decimal.Decimal `json:"max_price,string"`
	MaxAmount   decimal.Decimal `json:"max_amount,string"`
	MinAmount   decimal.Decimal `json:"min_amount,string"`
}

// AuthResponse stores the auth response
type AuthResponse struct {
	Result bool   `json:"bool"`
	Error  string `json:"error"`
}

// UserInfo stores the user info
type UserInfo struct {
	AuthResponse
	UID        int               `json:"uid"`
	ServerDate int               `json:"server_date"`
	Balances   map[string]string `json:"balances"`
	Reserved   map[string]string `json:"reserved"`
}

// OpenOrders stores the order info
type OpenOrders struct {
	OrderID  int64           `json:"order_id,string"`
	Created  int64           `json:"created,string"`
	Type     string          `json:"type"`
	Pair     string          `json:"pair"`
	Price    decimal.Decimal `json:"price,string"`
	Quantity decimal.Decimal `json:"quantity,string"`
	Amount   decimal.Decimal `json:"amount,string"`
}

// UserTrades stores the users trade info
type UserTrades struct {
	TradeID  int64           `json:"trade_id"`
	Date     int64           `json:"date"`
	Type     string          `json:"type"`
	Pair     string          `json:"pair"`
	OrderID  int64           `json:"order_id"`
	Quantity decimal.Decimal `json:"quantity"`
	Price    decimal.Decimal `json:"price"`
	Amount   decimal.Decimal `json:"amount"`
}

// CancelledOrder stores cancelled order data
type CancelledOrder struct {
	Date     int64           `json:"date"`
	OrderID  int64           `json:"order_id,string"`
	Type     string          `json:"type"`
	Pair     string          `json:"pair"`
	Price    decimal.Decimal `json:"price,string"`
	Quantity decimal.Decimal `json:"quantity,string"`
	Amount   decimal.Decimal `json:"amount,string"`
}

// OrderTrades stores order trade information
type OrderTrades struct {
	Type        string          `json:"type"`
	InCurrency  string          `json:"in_currency"`
	InAmount    decimal.Decimal `json:"in_amount,string"`
	OutCurrency string          `json:"out_currency"`
	OutAmount   decimal.Decimal `json:"out_amount,string"`
	Trades      []UserTrades    `json:"trades"`
}

// RequiredAmount stores the calculation for buying a certain amount of currency
// for a particular currency
type RequiredAmount struct {
	Quantity decimal.Decimal `json:"quantity,string"`
	Amount   decimal.Decimal `json:"amount,string"`
	AvgPrice decimal.Decimal `json:"avg_price,string"`
}

// ExcodeCreate stores the excode create coupon info
type ExcodeCreate struct {
	TaskID   int64             `json:"task_id"`
	Code     string            `json:"code"`
	Amount   decimal.Decimal   `json:"amount,string"`
	Currency string            `json:"currency"`
	Balances map[string]string `json:"balances"`
}

// ExcodeLoad stores the excode load coupon info
type ExcodeLoad struct {
	TaskID   int64             `json:"task_id"`
	Amount   decimal.Decimal   `json:"amount,string"`
	Currency string            `json:"currency"`
	Balances map[string]string `json:"balances"`
}

// WalletHistory stores the users wallet history
type WalletHistory struct {
	Begin   int64 `json:"begin,string"`
	End     int64 `json:"end,string"`
	History []struct {
		Timestamp int64           `json:"dt"`
		Type      string          `json:"string"`
		Currency  string          `json:"curr"`
		Status    string          `json:"status"`
		Provider  string          `json:"provider"`
		Amount    decimal.Decimal `json:"amount,string"`
		Account   string          `json:"account,string"`
	}
}
