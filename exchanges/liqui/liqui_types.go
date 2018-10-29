package liqui

import "github.com/thrasher-/gocryptotrader/currency/symbol"

// Info holds the current pair information as well as server time
type Info struct {
	ServerTime int64               `json:"server_time"`
	Pairs      map[string]PairData `json:"pairs"`
	Success    int                 `json:"success"`
	Error      string              `json:"error"`
}

// PairData is a sub-type for Info
type PairData struct {
	DecimalPlaces int     `json:"decimal_places"`
	MinPrice      float64 `json:"min_price"`
	MaxPrice      float64 `json:"max_price"`
	MinAmount     float64 `json:"min_amount"`
	Hidden        int     `json:"hidden"`
	Fee           float64 `json:"fee"`
}

// Ticker contains ticker information
type Ticker struct {
	High           float64
	Low            float64
	Avg            float64
	Vol            float64
	VolumeCurrency float64
	Last           float64
	Buy            float64
	Sell           float64
	Updated        int64
}

// Orderbook references both ask and bid sides
type Orderbook struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

// Trades contains trade information
type Trades struct {
	Type      string  `json:"type"`
	Price     float64 `json:"bid"`
	Amount    float64 `json:"amount"`
	TID       int64   `json:"tid"`
	Timestamp int64   `json:"timestamp"`
}

// AccountInfo contains full account details information
type AccountInfo struct {
	Funds  map[string]float64 `json:"funds"`
	Rights struct {
		Info     bool `json:"info"`
		Trade    bool `json:"trade"`
		Withdraw bool `json:"withdraw"`
	} `json:"rights"`
	ServerTime       float64 `json:"server_time"`
	TransactionCount int     `json:"transaction_count"`
	OpenOrders       int     `json:"open_orders"`
	Success          int     `json:"success"`
	Error            string  `json:"error"`
}

// ActiveOrders holds active order information
type ActiveOrders struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"sell"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"time_created"`
	Status           int     `json:"status"`
	Success          int     `json:"success"`
	Error            string  `json:"error"`
}

// OrderInfo holds specific order information
type OrderInfo struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"sell"`
	StartAmount      float64 `json:"start_amount"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"time_created"`
	Status           int     `json:"status"`
	Success          int     `json:"success"`
	Error            string  `json:"error"`
}

// CancelOrder holds cancelled order information
type CancelOrder struct {
	OrderID float64            `json:"order_id"`
	Funds   map[string]float64 `json:"funds"`
	Success int                `json:"success"`
	Error   string             `json:"error"`
}

// Trade holds trading information
type Trade struct {
	Received float64            `json:"received"`
	Remains  float64            `json:"remains"`
	OrderID  float64            `json:"order_id"`
	Funds    map[string]float64 `json:"funds"`
	Success  int                `json:"success"`
	Error    string             `json:"error"`
}

// TradeHistory contains trade history data
type TradeHistory struct {
	Pair      string  `json:"pair"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Rate      float64 `json:"rate"`
	OrderID   float64 `json:"order_id"`
	MyOrder   int     `json:"is_your_order"`
	Timestamp float64 `json:"timestamp"`
	Success   int     `json:"success"`
	Error     string  `json:"error"`
}

// Response is a generalized return type
type Response struct {
	Return  interface{} `json:"return"`
	Success int         `json:"success"`
	Error   string      `json:"error"`
}

// WithdrawCoins shows the amount of coins withdrawn from liqui not yet available
type WithdrawCoins struct {
	TID        int64              `json:"tId"`
	AmountSent float64            `json:"amountSent"`
	Funds      map[string]float64 `json:"funds"`
	Success    int                `json:"success"`
	Error      string             `json:"error"`
}

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[string]float64{
	symbol.ZRX:   5,
	symbol.ADX:   10,
	symbol.AE:    2,
	symbol.AION:  5,
	symbol.AST:   25,
	symbol.ANT:   2,
	symbol.REP:   0.15,
	symbol.BNT:   1.5,
	symbol.BAT:   20,
	symbol.BTC:   0.001,
	symbol.BCH:   0.007,
	symbol.BMC:   7,
	symbol.BCAP:  2,
	symbol.TIME:  0.5,
	symbol.CVC:   15,
	symbol.CFI:   100,
	symbol.CLN:   100,
	symbol.DASH:  0.003,
	symbol.MANA:  40,
	symbol.DGD:   0.05,
	symbol.DNT:   100,
	symbol.EDG:   15,
	symbol.ENG:   3,
	symbol.ENJ:   50,
	symbol.EOS:   0.5,
	symbol.ETH:   0.01,
	symbol.FIRST: 5,
	symbol.GNO:   0.1,
	symbol.GNT:   15,
	symbol.GOLOS: 0.01,
	symbol.GBG:   0.01,
	symbol.HMQ:   20,
	symbol.ICN:   7,
	symbol.RLC:   5,
	symbol.INCNT: 1,
	symbol.IND:   70,
	symbol.INS:   7,
	symbol.KNC:   4,
	symbol.LDC:   1000,
	symbol.LTC:   0.01,
	symbol.LUN:   0.5,
	symbol.GUP:   40,
	symbol.MLN:   0.2,
	symbol.MGO:   20,
	symbol.MCO:   0.7,
	symbol.MYST:  20,
	symbol.NEU:   7,
	symbol.NET:   10,
	symbol.OAX:   10,
	symbol.OMG:   0.5,
	symbol.PTOY:  50,
	symbol.PLU:   0.5,
	symbol.PRO:   7,
	symbol.QTUM:  0.2,
	symbol.QRL:   10,
	symbol.REN:   100,
	symbol.REQ:   50,
	symbol.ROUND: 100,
	symbol.SALT:  4,
	symbol.SAN:   4,
	symbol.SNGLS: 80,
	symbol.AGI:   50,
	symbol.SRN:   30,
	symbol.SNM:   25,
	symbol.XID:   50,
	symbol.SNT:   50,
	symbol.STEEM: 0.01,
	symbol.SBD:   0.01,
	symbol.STORJ: 7,
	symbol.STX:   20,
	symbol.TAAS:  2,
	symbol.PAY:   5,
	symbol.USDT:  20,
	symbol.TNT:   100,
	symbol.TKN:   5,
	symbol.TRX:   100,
	symbol.VEN:   3,
	symbol.VSL:   30,
	symbol.WAVES: 0.01,
	symbol.WPR:   100,
	symbol.TRST:  100,
	symbol.WINGS: 20,
	symbol.XXX:   0.01,
	symbol.XZC:   0.01,
}
