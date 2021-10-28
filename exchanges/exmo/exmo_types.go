package exmo

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
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

// PairInformation defines the full pair information
type PairInformation struct {
	Name                    string  `json:"name"`
	BuyPrice                float64 `json:"buy_price,string"`
	SellPrice               float64 `json:"sell_price,string"`
	LastTradePrice          float64 `json:"last_trade_price,string"`
	TickerUpdated           int64   `json:"ticker_updated,string"`
	IsFairPrice             bool    `json:"is_fair_price"`
	MaximumPricePrecision   float64 `json:"max_price_precision"`
	MinimumOrderQuantity    float64 `json:"min_order_quantity,string"`
	MaximumOrderQuantity    float64 `json:"max_order_quantity,string"`
	MinimumOrderPrice       float64 `json:"min_order_price,string"`
	MaximumOrderPrice       float64 `json:"max_order_price,string"`
	MaximumPositionQuantity float64 `json:"max_position_quantity,string"`
	TradeTakerFee           float64 `json:"trade_taker_fee,string"`
	TradeMakerFee           float64 `json:"trade_maker_fee,string"`
	LiquidationFee          float64 `json:"liquidation_fee,string"`
	MaxLeverage             float64 `json:"max_leverage,string"`
	DefaultLeverage         float64 `json:"default_leverage,string"`
	LiquidationLevel        float64 `json:"liquidation_level,string"`
	MarginCallLevel         float64 `json:"margin_call_level,string"`
	Position                float64 `json:"position"`
	Updated                 int64   `json:"updated,string"`
}

// transferFees the large list of predefined withdrawal and deposit fees.
// Prone to change.
var transferFees = map[asset.Item]map[currency.Code]fee.Transfer{
	asset.Spot: {
		currency.EXM:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
		currency.ADA:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.ALGO:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.ATOM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.05)},
		currency.BCH:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
		currency.BTC:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0005)},
		currency.BTCV:    {Deposit: fee.Convert(0)},
		currency.BTG:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
		currency.BTT:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
		currency.CHZ:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
		currency.CRON:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.5)},
		currency.DAI:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7)},
		currency.DASH:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.002)},
		currency.DOGE:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.ETC:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.ETH:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.004)},
		currency.GAS:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.GNY:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.GUSD:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7)},
		currency.HAI:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
		currency.HB:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4000)},
		currency.HP:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
		currency.IQN:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
		currency.LINK:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.3)},
		currency.LTC:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.MKR:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.002)},
		currency.MNC:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4000)},
		currency.NEO:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.OMG:     {Deposit: fee.Convert(.1), Withdrawal: fee.Convert(1)},
		currency.ONE:     {Withdrawal: fee.Convert(1)},
		currency.ONG:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
		currency.ONT:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.PRQ:     {Withdrawal: fee.Convert(20)},
		currency.QTUM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.ROOBEE:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
		currency.SMART:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.5)},
		currency.TONCOIN: {Withdrawal: fee.Convert(0.1)},
		currency.TRX:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.UNI:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.3)},
		currency.USDC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
		currency.USDT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)}, // TODO: TRC20 Chain
		currency.VLX:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.WAVES:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
		currency.WXT:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)}, // TODO: ERC20 Chain
		currency.XEM:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
		currency.XLM:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.XRP:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.02)},
		currency.XTZ:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
		currency.XYM:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.5)},
		currency.YFI:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0002)},
		currency.ZEC:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
		currency.ZRC:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(28)},
	},
}

// transferBank the large list of predefined withdrawal and deposit fees for
// fiat. Prone to change.
var transferBank = map[bank.Transfer]map[currency.Code]fee.Transfer{
	bank.Payeer: {
		currency.USD: {Deposit: fee.Convert(0.0249), IsPercentage: true},
		currency.RUB: {Withdrawal: fee.Convert(0.0199), IsPercentage: true},
	},
	bank.SEPA: {
		currency.EUR: {Deposit: fee.Convert(1), Withdrawal: fee.Convert(1)},
	},
	bank.AdvCash: {
		currency.USD: {Withdrawal: fee.Convert(0.0499), Deposit: fee.Convert(0.0049), IsPercentage: true},
		currency.RUB: {Withdrawal: fee.Convert(0), Deposit: fee.Convert(0.0399), IsPercentage: true},
		currency.KZT: {Withdrawal: fee.Convert(0.0099), Deposit: fee.Convert(0.0149), IsPercentage: true},
	},
	bank.ExmoGiftCard: {
		currency.USD: {Deposit: fee.Convert(0)},
		currency.EUR: {Deposit: fee.Convert(0)},
		currency.RUB: {Deposit: fee.Convert(0)},
		currency.PLN: {Deposit: fee.Convert(0)},
		currency.UAH: {Deposit: fee.Convert(0)},
		currency.KZT: {Deposit: fee.Convert(0)},
	},
	bank.VisaMastercard: {
		currency.USD: {Deposit: fee.Convert(0.0299), IsPercentage: true}, // + 35 cents??
		currency.UAH: {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0249), IsPercentage: true},
	},
	bank.Terminal: {
		currency.UAH: {Deposit: fee.Convert(0.026), IsPercentage: true},
	},
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
