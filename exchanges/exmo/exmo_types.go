package exmo

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
)

// Trades holds trade data
type Trades struct {
	TradeID  int64   `json:"trade_id"`
	Type     string  `json:"type"`
	Quantity float64 `json:"quantity,string"`
	Price    float64 `json:"price,string"`
	Amount   float64 `json:"amount,string"`
	Date     int64   `json:"date"`
	Pair     string  `json:"pair"`
}

// Orderbook holds the orderbook data
type Orderbook struct {
	AskQuantity float64    `json:"ask_quantity,string"`
	AskAmount   float64    `json:"ask_amount,string"`
	AskTop      float64    `json:"ask_top,string"`
	BidQuantity float64    `json:"bid_quantity,string"`
	BidTop      float64    `json:"bid_top,string"`
	Ask         [][]string `json:"ask"`
	Bid         [][]string `json:"bid"`
}

// Ticker holds the ticker data
type Ticker struct {
	Buy           float64 `json:"buy_price,string"`
	Sell          float64 `json:"sell_price,string"`
	Last          float64 `json:"last_trade,string"`
	High          float64 `json:"high,string"`
	Low           float64 `json:"low,string"`
	Average       float64 `json:"average,string"`
	Volume        float64 `json:"vol,string"`
	VolumeCurrent float64 `json:"vol_curr,string"`
	Updated       int64   `json:"updated"`
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
	UID        int               `json:"uid"`
	ServerDate int               `json:"server_date"`
	Balances   map[string]string `json:"balances"`
	Reserved   map[string]string `json:"reserved"`
}

// OpenOrders stores the order info
type OpenOrders struct {
	OrderID  int64   `json:"order_id,string"`
	Created  int64   `json:"created,string"`
	Type     string  `json:"type"`
	Pair     string  `json:"pair"`
	Price    float64 `json:"price,string"`
	Quantity float64 `json:"quantity,string"`
	Amount   float64 `json:"amount,string"`
}

// UserTrades stores the users trade info
type UserTrades struct {
	TradeID  int64   `json:"trade_id"`
	Date     int64   `json:"date"`
	Type     string  `json:"type"`
	Pair     string  `json:"pair"`
	OrderID  int64   `json:"order_id"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	Amount   float64 `json:"amount"`
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
		Timestamp int64   `json:"dt"`
		Type      string  `json:"string"`
		Currency  string  `json:"curr"`
		Status    string  `json:"status"`
		Provider  string  `json:"provider"`
		Amount    float64 `json:"amount,string"`
		Account   string  `json:"account,string"`
	}
}

// withdrawFees the large list of predefined withdrawal and deposit fees.
// Prone to change.
var withdrawFees = map[asset.Item]map[currency.Code]fee.Transfer{
	asset.Spot: {
		currency.BTC:   {Withdrawal: 0.0005},
		currency.LTC:   {Withdrawal: 0.01},
		currency.DOGE:  {Withdrawal: 1},
		currency.DASH:  {Withdrawal: 0.01},
		currency.ETH:   {Withdrawal: 0.01},
		currency.WAVES: {Withdrawal: 0.001},
		currency.ZEC:   {Withdrawal: 0.001},
		currency.USDT:  {Withdrawal: 5},
		currency.XMR:   {Withdrawal: 0.05},
		currency.XRP:   {Withdrawal: 0.02},
		currency.KICK:  {Withdrawal: 50},
		currency.ETC:   {Withdrawal: 0.01},
		currency.BCH:   {Withdrawal: 0.001},
		currency.BTG:   {Withdrawal: 0.001},
		currency.HBZ:   {Withdrawal: 65},
		currency.BTCZ:  {Withdrawal: 5},
		currency.DXT:   {Withdrawal: 20},
		currency.STQ:   {Withdrawal: 100},
		currency.XLM:   {Withdrawal: 0.001},
		currency.OMG:   {Withdrawal: 0.5},
		currency.TRX:   {Withdrawal: 1},
		currency.ADA:   {Withdrawal: 1},
		currency.INK:   {Withdrawal: 50},
		currency.ZRX:   {Withdrawal: 1},
		currency.GNT:   {Withdrawal: 1},
	},
}

var transferBank = map[fee.InternationalBankTransaction]map[currency.Code]fee.Transfer{
	exchange.WireTransfer: {
		currency.RUB: {Withdrawal: 3200, Deposit: 1600, IsPercentage: true}, // This doesn't seem like a percentage val???
		currency.PLN: {Withdrawal: 125, Deposit: 30, IsPercentage: true},    // Or this?
		currency.TRY: {Withdrawal: 0, Deposit: 0, IsPercentage: true},
	},
	exchange.PerfectMoney: {
		currency.USD: {Withdrawal: 0.01, IsPercentage: true},
		currency.EUR: {Withdrawal: 0.0195, IsPercentage: true},
	},
	exchange.Neteller: {
		currency.USD: {Withdrawal: 0.0195, Deposit: 0.035, IsPercentage: true}, // Also has an addition of .29 ??
		currency.EUR: {Withdrawal: 0.0195, Deposit: 0.035, IsPercentage: true}, // Also has an addition of .25 ??
	},
	exchange.AdvCash: {
		currency.USD: {Withdrawal: 0.0295, Deposit: 0.0295, IsPercentage: true},
		currency.EUR: {Withdrawal: 0.03, Deposit: 0.01, IsPercentage: true},
		currency.RUB: {Withdrawal: 0.0195, Deposit: 0.0495, IsPercentage: true},
		currency.UAH: {Withdrawal: 0.0495, Deposit: 0.01, IsPercentage: true},
	},
	exchange.Payeer: {
		currency.USD: {Withdrawal: 0.0395, Deposit: 0.0195, IsPercentage: true},
		currency.EUR: {Withdrawal: 0.01, Deposit: 0.0295, IsPercentage: true},
		currency.RUB: {Withdrawal: 0.0595, Deposit: 0.0345, IsPercentage: true},
	},
	exchange.Skrill: {
		currency.USD: {Withdrawal: 0.0145, Deposit: 0.0495, IsPercentage: true}, // Also has an addition of .36 ??
		currency.EUR: {Withdrawal: 0.03, Deposit: 0.0295, IsPercentage: true},   // Also has an addition of .29 ??
		currency.PLN: {Withdrawal: 0, Deposit: 0.035, IsPercentage: true},       // Also has an addition of 1.21 ??
		currency.TRY: {Withdrawal: 0, Deposit: 0, IsPercentage: true},
	},
	exchange.VisaMastercard: {
		currency.USD: {Withdrawal: 0.06, IsPercentage: true},
		currency.EUR: {Withdrawal: 0.06, IsPercentage: true},
		currency.PLN: {Withdrawal: 0.06, IsPercentage: true},
	},
}
