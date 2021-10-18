package bithumb

import (
	"encoding/json"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
)

// Ticker holds ticker data
type Ticker struct {
	OpeningPrice              float64 `json:"opening_price,string"`
	ClosingPrice              float64 `json:"closing_price,string"`
	MinPrice                  float64 `json:"min_price,string"`
	MaxPrice                  float64 `json:"max_price,string"`
	UnitsTraded               float64 `json:"units_traded,string"`
	AccumulatedTradeValue     float64 `json:"acc_trade_value,string"`
	PreviousClosingPrice      float64 `json:"prev_closing_price,string"`
	UnitsTraded24Hr           float64 `json:"units_traded_24H,string"`
	AccumulatedTradeValue24hr float64 `json:"acc_trade_value_24H,string"`
	Fluctuate24Hr             float64 `json:"fluctate_24H,string"`
	FluctuateRate24hr         float64 `json:"fluctate_rate_24H,string"`
	Date                      int64   `json:"date,string"`
}

// TickerResponse holds the standard ticker response
type TickerResponse struct {
	Status  string `json:"status"`
	Data    Ticker `json:"data"`
	Message string `json:"message"`
}

// TickersResponse holds the standard ticker response
type TickersResponse struct {
	Status  string                     `json:"status"`
	Data    map[string]json.RawMessage `json:"data"`
	Message string                     `json:"message"`
}

// Orderbook holds full range of order book information
type Orderbook struct {
	Status string `json:"status"`
	Data   struct {
		Timestamp       int64  `json:"timestamp,string"`
		OrderCurrency   string `json:"order_currency"`
		PaymentCurrency string `json:"payment_currency"`
		Bids            []struct {
			Quantity float64 `json:"quantity,string"`
			Price    float64 `json:"price,string"`
		} `json:"bids"`
		Asks []struct {
			Quantity float64 `json:"quantity,string"`
			Price    float64 `json:"price,string"`
		} `json:"asks"`
	} `json:"data"`
	Message string `json:"message"`
}

// TransactionHistory holds history of completed transaction data
type TransactionHistory struct {
	Status string `json:"status"`
	Data   []struct {
		ContNumber      int64   `json:"cont_no,string"`
		TransactionDate string  `json:"transaction_date"`
		Type            string  `json:"type"`
		UnitsTraded     float64 `json:"units_traded,string"`
		Price           float64 `json:"price,string"`
		Total           float64 `json:"total,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// Account holds account details
type Account struct {
	Status string `json:"status"`
	Data   struct {
		Created   int64   `json:"created,string"`
		AccountID string  `json:"account_id"`
		TradeFee  float64 `json:"trade_fee,string"`
		Balance   float64 `json:"balance,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// Balance holds balance details
type Balance struct {
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data"`
	Message string                 `json:"message"`
}

// WalletAddressRes contains wallet address information
type WalletAddressRes struct {
	Status string `json:"status"`
	Data   struct {
		WalletAddress string `json:"wallet_address"`
		Tag           string // custom field we populate
		Currency      string `json:"currency"`
	} `json:"data"`
	Message string `json:"message"`
}

// LastTransactionTicker holds customer last transaction information
type LastTransactionTicker struct {
	Status string `json:"status"`
	Data   struct {
		OpeningPrice float64 `json:"opening_price,string"`
		ClosingPrice float64 `json:"closing_price,string"`
		MinPrice     float64 `json:"min_price,string"`
		MaxPrice     float64 `json:"max_price,string"`
		AveragePrice float64 `json:"average_price,string"`
		UnitsTraded  float64 `json:"units_traded,string"`
		Volume1Day   float64 `json:"volume_1day,string"`
		Volume7Day   float64 `json:"volume_7day,string"`
		BuyPrice     int64   `json:"buy_price,string"`
		SellPrice    int64   `json:"sell_price,string"`
		Date         int64   `json:"date,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// Orders contains information about your current orders
type Orders struct {
	Status  string      `json:"status"`
	Data    []OrderData `json:"data"`
	Message string      `json:"message"`
}

// OrderData contains all individual order details
type OrderData struct {
	OrderID         string  `json:"order_id"`
	OrderCurrency   string  `json:"order_currency"`
	OrderDate       int64   `json:"order_date"`
	PaymentCurrency string  `json:"payment_currency"`
	Type            string  `json:"type"`
	Status          string  `json:"status"`
	Units           float64 `json:"units,string"`
	UnitsRemaining  float64 `json:"units_remaining,string"`
	Price           float64 `json:"price,string"`
	Fee             float64 `json:"fee,string"`
	Total           float64 `json:"total,string"`
	DateCompleted   int64   `json:"date_completed"`
}

// UserTransactions holds users full transaction list
type UserTransactions struct {
	Status string `json:"status"`
	Data   []struct {
		Search       string  `json:"search"`
		TransferDate int64   `json:"transfer_date"`
		Units        string  `json:"units"`
		Price        float64 `json:"price,string"`
		BTC1KRW      float64 `json:"btc1krw,string"`
		Fee          string  `json:"fee"`
		BTCRemain    float64 `json:"btc_remain,string"`
		KRWRemain    float64 `json:"krw_remain,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// OrderPlace contains order information
type OrderPlace struct {
	Status string `json:"status"`
	Data   []struct {
		ContID string  `json:"cont_id"`
		Units  float64 `json:"units,string"`
		Price  float64 `json:"price,string"`
		Total  float64 `json:"total,string"`
		Fee    float64 `json:"fee,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// OrderDetails contains specific order information
type OrderDetails struct {
	Status string `json:"status"`
	Data   []struct {
		TransactionDate int64   `json:"transaction_date,string"`
		Type            string  `json:"type"`
		OrderCurrency   string  `json:"order_currency"`
		PaymentCurrency string  `json:"payment_currency"`
		UnitsTraded     float64 `json:"units_traded,string"`
		Price           float64 `json:"price,string"`
		Total           float64 `json:"total,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// ActionStatus holds the return status
type ActionStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// KRWDeposit resp type for a KRW deposit
type KRWDeposit struct {
	Status   string `json:"status"`
	Account  string `json:"account"`
	Bank     string `json:"bank"`
	BankUser string `json:"BankUser"`
	Message  string `json:"message"`
}

// MarketBuy holds market buy order information
type MarketBuy struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
	Data    []struct {
		ContID string  `json:"cont_id"`
		Units  float64 `json:"units,string"`
		Price  float64 `json:"price,string"`
		Total  float64 `json:"total,string"`
		Fee    float64 `json:"fee,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// MarketSell holds market buy order information
type MarketSell struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
	Data    []struct {
		ContID string  `json:"cont_id"`
		Units  float64 `json:"units,string"`
		Price  float64 `json:"price,string"`
		Total  float64 `json:"total,string"`
		Fee    float64 `json:"fee,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// bankTransferFees predefined banking transfer fees. Prone to change.
var bankTransferFees = map[fee.BankTransaction]map[currency.Code]fee.Transfer{
	fee.WireTransfer: {
		currency.KRW: {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
	},
}

// transferFees the large list of predefined fees. Prone to change.
var transferFees = map[asset.Item]map[currency.Code]fee.Transfer{
	asset.Spot: {
		currency.BTC:    {Deposit: fee.ConvertWithAmount(0.001, 0, 0.005), Withdrawal: fee.Convert(0.001)},
		currency.ETH:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.DASH:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.LTC:    {Deposit: fee.ConvertWithAmount(0.01, 0, 0.3), Withdrawal: fee.Convert(0.01)},
		currency.ETC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.XRP:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.BCH:    {Deposit: fee.ConvertWithAmount(0.001, 0, 0.03), Withdrawal: fee.Convert(0.001)},
		currency.XMR:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.05)},
		currency.ZEC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
		currency.QTUM:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.05)},
		currency.BTG:    {Deposit: fee.ConvertWithAmount(0.001, 0, 0.15), Withdrawal: fee.Convert(0.001)},
		currency.ICX:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.1)},
		currency.TRX:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.ELF:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.OMG:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.7)},
		currency.KNC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.7)},
		currency.GLM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(95)},
		currency.ZIL:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
		currency.WAXP:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
		currency.POWR:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(73)},
		currency.LRC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
		currency.STEEM:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.STRAX:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.2)},
		currency.ZRX:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
		currency.REP:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.3)},
		currency.XEM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
		currency.SNT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(73)},
		currency.ADA:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.5)},
		currency.CTXC:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.BAT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.5)},
		currency.WTC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.7)},
		currency.THETA:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.5)},
		currency.LOOM:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.WAVES:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.TRUE:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
		currency.LINK:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.4)},
		currency.ENJ:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.4)},
		currency.VET:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
		currency.MTL:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.7)},
		currency.IOST:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.TMTG:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
		currency.QKC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.HDAC:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
		currency.AMO:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
		currency.BSV:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
		currency.ORBS:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(77)},
		currency.TFUEL:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.VALOR:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
		currency.CON:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500)},
		currency.ANKR:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(85)},
		currency.MIX:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
		currency.CRO:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.FX:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
		currency.CHR:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(36)},
		currency.MBL:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
		currency.MXC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(230)},
		currency.FCT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(64)},
		currency.TRV:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
		currency.DAD:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(36)},
		currency.WOM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(48)},
		currency.SOC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(150)},
		currency.EM:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
		currency.BOA:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
		currency.FLETA:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(580)},
		currency.SXP:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.7)},
		currency.COS:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(310)},
		currency.APIX:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.EL:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(540)},
		currency.BASIC:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500)},
		currency.HIV:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
		currency.XPR:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(80)},
		currency.VRA:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(950)},
		currency.FIT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2300)},
		currency.EGG:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
		currency.BORA:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(54)},
		currency.ARPA:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.APM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
		currency.CKB:    {Deposit: fee.ConvertWithAmount(61.99999999, 0, 61.99999999), Withdrawal: fee.Convert(1)},
		currency.AERGO:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
		currency.ANW:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(67)},
		currency.CENNZ:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6)},
		currency.EVZ:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
		currency.CYCLUB: {Deposit: fee.Convert(0), Withdrawal: fee.Convert(540)},
		currency.SRM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.QTCON:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(180)},
		currency.UNI:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.4)},
		currency.YFI:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0003)},
		currency.UMA:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.AAVE:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.04)},
		currency.COMP:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.03)},
		currency.REN:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(23)},
		currency.BAL:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.9)},
		currency.RSR:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(640)},
		currency.NMR:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.4)},
		currency.RLC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
		currency.UOS:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
		currency.SAND:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.STPT:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
		currency.GOM2:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
		currency.RINGX:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(87)},
		currency.BEL:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.5)},
		currency.OBSR:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.ORC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.6)},
		currency.POLA:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(46)},
		currency.AWO:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(850)},
		currency.ADP:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
		currency.DVI:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
		currency.DRM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(64)},
		currency.IBP:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(80)},
		currency.GHX:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
		currency.MIR:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.6)},
		currency.MVC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(80)},
		currency.BLY:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(180)},
		currency.WOZX:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8.9)},
		currency.ANV:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
		currency.GRT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.3)},
		currency.MM:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
		currency.BIOT:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
		currency.XNO:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(38)},
		currency.SNX:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.7)},
		currency.RAI:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.8)},
		currency.COLA:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
		currency.NU:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.OXT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(18)},
		currency.LINA:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.ASTA:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.MAP:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.AQT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
		currency.WIKEN:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(80)},
		currency.CTSI:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
		currency.MANA:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
		currency.LPT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.4)},
		currency.MKR:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.004)},
		currency.SUSHI:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.8)},
		currency.ASM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
		currency.PUNDIX: {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.CELR:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(270)},
		currency.LF:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.8)},
		currency.ARW:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.6)},
		currency.MSB:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.4)},
		currency.RLY:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(16)},
		currency.OCEAN:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
		currency.BFC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
		currency.ALICE:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.CAKE:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.8)},
		currency.BNT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.6)},
		currency.BNT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.6)},
		currency.CHZ:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.AXS:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.3)},
		currency.DAI:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
		currency.MATIC:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
		currency.BAKE:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.2)},
		currency.VELO:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
		currency.BCD:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.02)},
		currency.XLM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.01)},
		currency.GXC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.5)},
		currency.BTT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
		currency.VSYS:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.IPX:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.WICC:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
		currency.ONT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.LUNA:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.AION:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.5)},
		currency.META:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.KLAY:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.ONG:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
		currency.ALGO:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.1)},
		currency.JST:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
		currency.XTZ:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.3)},
		currency.MLK:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
		currency.WEMIX:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
		currency.DOT:    {Deposit: fee.ConvertWithAmount(1.9999999999, 0, 1.9999999999), Withdrawal: fee.Convert(.15)},
		currency.ATOM:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.1)},
		currency.SSX:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
		currency.TEMCO:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
		currency.HIBS:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
		currency.BURGER: {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.7)},
		currency.DOGE:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.KSM:    {Deposit: fee.ConvertWithAmount(0.00999999, 0, 0.00999999), Withdrawal: fee.Convert(0.01)},
		currency.CTK:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.XYM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.BNB:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.001)},
		currency.SUN:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.XEC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(30000)},
		currency.PCI:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
		currency.SOL:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(.03)},
		currency.LN:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	},
}

// FullBalance defines a return type with full balance data
type FullBalance struct {
	InUse     map[string]float64
	Misu      map[string]float64
	Total     map[string]float64
	Xcoin     map[string]float64
	Available map[string]float64
}

// OHLCVResponse holds returned kline data
type OHLCVResponse struct {
	Status string           `json:"status"`
	Data   [][6]interface{} `json:"data"`
}

// Status defines the current exchange allowance to deposit or withdraw a
// currency
type Status struct {
	Status string `json:"status"`
	Data   struct {
		DepositStatus    int64 `json:"deposit_status"`
		WithdrawalStatus int64 `json:"withdrawal_status"`
	} `json:"data"`
	Message string `json:"message"`
}

// StatusAll defines the current exchange allowance to deposit or withdraw a
// currency
type StatusAll struct {
	Status string `json:"status"`
	Data   map[string]struct {
		DepositStatus    int64 `json:"deposit_status"`
		WithdrawalStatus int64 `json:"withdrawal_status"`
	} `json:"data"`
	Message string `json:"message"`
}
