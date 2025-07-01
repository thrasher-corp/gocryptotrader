package exmo

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Trades holds trade data
type Trades struct {
	TradeID  int64      `json:"trade_id"`
	Type     string     `json:"type"`
	Quantity float64    `json:"quantity,string"`
	Price    float64    `json:"price,string"`
	Amount   float64    `json:"amount,string"`
	Date     types.Time `json:"date"`
	Pair     string     `json:"pair"`
}

// Orderbook holds the orderbook data
type Orderbook struct {
	AskQuantity float64           `json:"ask_quantity,string"`
	AskAmount   float64           `json:"ask_amount,string"`
	AskTop      float64           `json:"ask_top,string"`
	BidQuantity float64           `json:"bid_quantity,string"`
	BidTop      float64           `json:"bid_top,string"`
	Asks        [][3]types.Number `json:"ask"`
	Bids        [][3]types.Number `json:"bid"`
}

// Ticker holds the ticker data
type Ticker struct {
	Buy           float64    `json:"buy_price,string"`
	Sell          float64    `json:"sell_price,string"`
	Last          float64    `json:"last_trade,string"`
	High          float64    `json:"high,string"`
	Low           float64    `json:"low,string"`
	Average       float64    `json:"average,string"`
	Volume        float64    `json:"vol,string"`
	VolumeCurrent float64    `json:"vol_curr,string"`
	Updated       types.Time `json:"updated"`
}

// PairSettings holds the pair settings
type PairSettings struct {
	MinQuantity float64 `json:"min_quantity,string"`
	MaxQuantity float64 `json:"max_quantity,string"`
	MinPrice    float64 `json:"min_price,string"`
	MaxPrice    float64 `json:"max_price,string"`
	MaxAmount   float64 `json:"max_amount,string"`
	MinAmount   float64 `json:"min_amount,string"`
}

// AuthResponse stores the auth response
type AuthResponse struct {
	Result bool   `json:"bool"`
	Error  string `json:"error"`
}

// UserInfo stores the user info
type UserInfo struct {
	AuthResponse
	UID        int                     `json:"uid"`
	ServerDate int                     `json:"server_date"`
	Balances   map[string]types.Number `json:"balances"`
	Reserved   map[string]types.Number `json:"reserved"`
}

// OpenOrders stores the order info
type OpenOrders struct {
	OrderID  int64      `json:"order_id,string"`
	Created  types.Time `json:"created"`
	Type     string     `json:"type"`
	Pair     string     `json:"pair"`
	Price    float64    `json:"price,string"`
	Quantity float64    `json:"quantity,string"`
	Amount   float64    `json:"amount,string"`
}

// UserTrades stores the users trade info
type UserTrades struct {
	TradeID  int64      `json:"trade_id"`
	Date     types.Time `json:"date"`
	Type     string     `json:"type"`
	Pair     string     `json:"pair"`
	OrderID  int64      `json:"order_id"`
	Quantity float64    `json:"quantity"`
	Price    float64    `json:"price"`
	Amount   float64    `json:"amount"`
}

// CancelledOrder stores cancelled order data
type CancelledOrder struct {
	Date     int64   `json:"date"`
	OrderID  int64   `json:"order_id,string"`
	Type     string  `json:"type"`
	Pair     string  `json:"pair"`
	Price    float64 `json:"price,string"`
	Quantity float64 `json:"quantity,string"`
	Amount   float64 `json:"amount,string"`
}

// OrderTrades stores order trade information
type OrderTrades struct {
	Type        string       `json:"type"`
	InCurrency  string       `json:"in_currency"`
	InAmount    float64      `json:"in_amount,string"`
	OutCurrency string       `json:"out_currency"`
	OutAmount   float64      `json:"out_amount,string"`
	Trades      []UserTrades `json:"trades"`
}

// RequiredAmount stores the calculation for buying a certain amount of currency
// for a particular currency
type RequiredAmount struct {
	Quantity float64 `json:"quantity,string"`
	Amount   float64 `json:"amount,string"`
	AvgPrice float64 `json:"avg_price,string"`
}

// ExcodeCreate stores the excode create coupon info
type ExcodeCreate struct {
	TaskID   int64             `json:"task_id"`
	Code     string            `json:"code"`
	Amount   float64           `json:"amount,string"`
	Currency string            `json:"currency"`
	Balances map[string]string `json:"balances"`
}

// ExcodeLoad stores the excode load coupon info
type ExcodeLoad struct {
	TaskID   int64             `json:"task_id"`
	Amount   float64           `json:"amount,string"`
	Currency string            `json:"currency"`
	Balances map[string]string `json:"balances"`
}

// WalletHistory stores the users wallet history
type WalletHistory struct {
	Begin   int64 `json:"begin,string"`
	End     int64 `json:"end,string"`
	History []struct {
		Timestamp types.Time `json:"dt"`
		Type      string     `json:"type"`
		Currency  string     `json:"curr"`
		Status    string     `json:"status"`
		Provider  string     `json:"provider"`
		Amount    float64    `json:"amount,string"`
		Account   string     `json:"account"`
		TXID      string     `json:"txid"`
	}
}

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[currency.Code]float64{
	currency.BTC:   0.0005,
	currency.LTC:   0.01,
	currency.DOGE:  1,
	currency.DASH:  0.01,
	currency.ETH:   0.01,
	currency.WAVES: 0.001,
	currency.ZEC:   0.001,
	currency.USDT:  5,
	currency.XMR:   0.05,
	currency.XRP:   0.02,
	currency.KICK:  50,
	currency.ETC:   0.01,
	currency.BCH:   0.001,
	currency.BTG:   0.001,
	currency.HBZ:   65,
	currency.BTCZ:  5,
	currency.DXT:   20,
	currency.STQ:   100,
	currency.XLM:   0.001,
	currency.OMG:   0.5,
	currency.TRX:   1,
	currency.ADA:   1,
	currency.INK:   50,
	currency.ZRX:   1,
	currency.GNT:   1,
}

// CryptoPaymentProvider stores the cryptocurrency transfer settings
type CryptoPaymentProvider struct {
	Type                  string  `json:"type"`
	Name                  string  `json:"name"`
	CurrencyName          string  `json:"currency_name"`
	Min                   float64 `json:"min,string"`
	Max                   float64 `json:"max,string"`
	Enabled               bool    `json:"enabled"`
	Comment               string  `json:"comment"`
	CommissionDescription string  `json:"commission_desc"`
	CurrencyConfirmations uint16  `json:"currency_confirmations"`
}
