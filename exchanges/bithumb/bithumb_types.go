package bithumb

import (
	"encoding/json"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
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
	OrderID         string      `json:"order_id"`
	OrderCurrency   string      `json:"order_currency"`
	OrderDate       bithumbTime `json:"order_date"`
	PaymentCurrency string      `json:"payment_currency"`
	Type            string      `json:"type"`
	Status          string      `json:"status"`
	Units           float64     `json:"units,string"`
	UnitsRemaining  float64     `json:"units_remaining,string"`
	Price           float64     `json:"price,string"`
	Fee             float64     `json:"fee,string"`
	Total           float64     `json:"total,string"`
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
var bankTransferFees = []fee.Transfer{
	{Currency: currency.KRW, BankTransfer: bank.WireTransfer, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
}

// defaultTransferFees the large list of predefined fees. Prone to change.
var defaultTransferFees = []fee.Transfer{
	{Currency: currency.BTC, Deposit: fee.ConvertWithAmount(0.001, 0, 0.005), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.ETH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.DASH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.LTC, Deposit: fee.ConvertWithAmount(0.01, 0, 0.3), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.ETC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.XRP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.BCH, Deposit: fee.ConvertWithAmount(0.001, 0, 0.03), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.XMR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.05)},
	{Currency: currency.ZEC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.QTUM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.05)},
	{Currency: currency.BTG, Deposit: fee.ConvertWithAmount(0.001, 0, 0.15), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.ICX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.1)},
	{Currency: currency.TRX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.ELF, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.OMG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.7)},
	{Currency: currency.KNC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.7)},
	{Currency: currency.GLM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(95)},
	{Currency: currency.ZIL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.WAXP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.POWR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(73)},
	{Currency: currency.LRC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
	{Currency: currency.STEEM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.STRAX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.2)},
	{Currency: currency.ZRX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
	{Currency: currency.REP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.3)},
	{Currency: currency.XEM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
	{Currency: currency.SNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(73)},
	{Currency: currency.ADA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.5)},
	{Currency: currency.CTXC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.BAT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.5)},
	{Currency: currency.WTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.7)},
	{Currency: currency.THETA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.5)},
	{Currency: currency.LOOM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.WAVES, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.TRUE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
	{Currency: currency.LINK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.4)},
	{Currency: currency.ENJ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.4)},
	{Currency: currency.VET, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.MTL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.7)},
	{Currency: currency.IOST, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.TMTG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
	{Currency: currency.QKC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.HDAC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
	{Currency: currency.AMO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
	{Currency: currency.BSV, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.ORBS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(77)},
	{Currency: currency.TFUEL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.VALOR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
	{Currency: currency.CON, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500)},
	{Currency: currency.ANKR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(85)},
	{Currency: currency.MIX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
	{Currency: currency.CRO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.FX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
	{Currency: currency.CHR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(36)},
	{Currency: currency.MBL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
	{Currency: currency.MXC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(230)},
	{Currency: currency.FCT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(64)},
	{Currency: currency.TRV, Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
	{Currency: currency.DAD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(36)},
	{Currency: currency.WOM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(48)},
	{Currency: currency.SOC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(150)},
	{Currency: currency.EM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.BOA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
	{Currency: currency.FLETA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(580)},
	{Currency: currency.SXP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.7)},
	{Currency: currency.COS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(310)},
	{Currency: currency.APIX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.EL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(540)},
	{Currency: currency.BASIC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500)},
	{Currency: currency.HIV, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
	{Currency: currency.XPR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(80)},
	{Currency: currency.VRA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(950)},
	{Currency: currency.FIT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2300)},
	{Currency: currency.EGG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
	{Currency: currency.BORA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(54)},
	{Currency: currency.ARPA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.APM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
	{Currency: currency.CKB, Deposit: fee.ConvertWithAmount(61.99999999, 0, 61.99999999), Withdrawal: fee.Convert(1)},
	{Currency: currency.AERGO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
	{Currency: currency.ANW, Deposit: fee.Convert(0), Withdrawal: fee.Convert(67)},
	{Currency: currency.CENNZ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6)},
	{Currency: currency.EVZ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
	{Currency: currency.CYCLUB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(540)},
	{Currency: currency.SRM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.QTCON, Deposit: fee.Convert(0), Withdrawal: fee.Convert(180)},
	{Currency: currency.UNI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.4)},
	{Currency: currency.YFI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0003)},
	{Currency: currency.UMA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.AAVE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.04)},
	{Currency: currency.COMP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.03)},
	{Currency: currency.REN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(23)},
	{Currency: currency.BAL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.9)},
	{Currency: currency.RSR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(640)},
	{Currency: currency.NMR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.4)},
	{Currency: currency.RLC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
	{Currency: currency.UOS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
	{Currency: currency.SAND, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.STPT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
	{Currency: currency.GOM2, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
	{Currency: currency.RINGX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(87)},
	{Currency: currency.BEL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.5)},
	{Currency: currency.OBSR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.ORC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.6)},
	{Currency: currency.POLA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(46)},
	{Currency: currency.AWO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(850)},
	{Currency: currency.ADP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
	{Currency: currency.DVI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
	{Currency: currency.DRM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(64)},
	{Currency: currency.IBP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(80)},
	{Currency: currency.GHX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
	{Currency: currency.MIR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.6)},
	{Currency: currency.MVC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(80)},
	{Currency: currency.BLY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(180)},
	{Currency: currency.WOZX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8.9)},
	{Currency: currency.ANV, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
	{Currency: currency.GRT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.3)},
	{Currency: currency.MM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
	{Currency: currency.BIOT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
	{Currency: currency.XNO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(38)},
	{Currency: currency.SNX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.7)},
	{Currency: currency.RAI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.8)},
	{Currency: currency.COLA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
	{Currency: currency.NU, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.OXT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(18)},
	{Currency: currency.LINA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.ASTA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.MAP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.AQT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
	{Currency: currency.WIKEN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(80)},
	{Currency: currency.CTSI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
	{Currency: currency.MANA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
	{Currency: currency.LPT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.4)},
	{Currency: currency.MKR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.004)},
	{Currency: currency.SUSHI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.8)},
	{Currency: currency.ASM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
	{Currency: currency.PUNDIX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.CELR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(270)},
	{Currency: currency.LF, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.8)},
	{Currency: currency.ARW, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.6)},
	{Currency: currency.MSB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.4)},
	{Currency: currency.RLY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(16)},
	{Currency: currency.OCEAN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
	{Currency: currency.BFC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
	{Currency: currency.ALICE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.CAKE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.8)},
	{Currency: currency.BNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.6)},
	{Currency: currency.BNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.6)},
	{Currency: currency.CHZ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.AXS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.3)},
	{Currency: currency.DAI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
	{Currency: currency.MATIC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
	{Currency: currency.BAKE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.2)},
	{Currency: currency.VELO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
	{Currency: currency.BCD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.02)},
	{Currency: currency.XLM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.01)},
	{Currency: currency.GXC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.5)},
	{Currency: currency.BTT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.VSYS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.IPX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.WICC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
	{Currency: currency.ONT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.LUNA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.AION, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.5)},
	{Currency: currency.META, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.KLAY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.ONG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.ALGO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.1)},
	{Currency: currency.JST, Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
	{Currency: currency.XTZ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.3)},
	{Currency: currency.MLK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
	{Currency: currency.WEMIX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
	{Currency: currency.DOT, Deposit: fee.ConvertWithAmount(1.9999999999, 0, 1.9999999999), Withdrawal: fee.Convert(.15)},
	{Currency: currency.ATOM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.1)},
	{Currency: currency.SSX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
	{Currency: currency.TEMCO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
	{Currency: currency.HIBS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
	{Currency: currency.BURGER, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.7)},
	{Currency: currency.DOGE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.KSM, Deposit: fee.ConvertWithAmount(0.00999999, 0, 0.00999999), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.CTK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.XYM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.BNB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.001)},
	{Currency: currency.SUN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.XEC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(30000)},
	{Currency: currency.PCI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
	{Currency: currency.SOL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(.03)},
	{Currency: currency.LN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
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
