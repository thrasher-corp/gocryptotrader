package okgroup

import "encoding/json"
import "github.com/thrasher-/gocryptotrader/currency/symbol"

// SpotInstrument stores the spot instrument info
type SpotInstrument struct {
	BaseCurrency   string  `json:"base_currency"`
	BaseIncrement  float64 `json:"base_increment,string"`
	BaseMinSize    float64 `json:"base_min_size,string"`
	InstrumentID   string  `json:"instrument_id"`
	MinSize        float64 `json:"min_size,string"`
	ProductID      string  `json:"product_id"`
	QuoteCurrency  string  `json:"quote_currency"`
	QuoteIncrement float64 `json:"quote_increment,string"`
	SizeIncrement  float64 `json:"size_increment,string"`
	TickSize       float64 `json:"tick_size,string"`
}

// ContractPrice holds date and ticker price price for contracts.
type ContractPrice struct {
	Date   string `json:"date"`
	Ticker struct {
		Buy        float64 `json:"buy"`
		ContractID float64 `json:"contract_id"`
		High       float64 `json:"high"`
		Low        float64 `json:"low"`
		Last       float64 `json:"last"`
		Sell       float64 `json:"sell"`
		UnitAmount float64 `json:"unit_amount"`
		Vol        float64 `json:"vol"`
	} `json:"ticker"`
	Result bool        `json:"result"`
	Error  interface{} `json:"error_code"`
}

// MultiStreamData contains raw data from okex
type MultiStreamData struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

// TokenOrdersResponse is returned after a request for all Token Orders
type TokenOrdersResponse struct {
	Result bool         `json:"result"`
	Orders []TokenOrder `json:"orders"`
}

// TokenOrder is the individual order details returned from TokenOrderResponse
type TokenOrder struct {
	Amount     float64 `json:"amount"`
	AvgPrice   int64   `json:"avg_price"`
	DealAmount int64   `json:"deal_amount"`
	OrderID    int64   `json:"order_id"`
	Price      int64   `json:"price"`
	Status     int64   `json:"status"`
	Symbol     string  `json:"symbol"`
	Type       string  `json:"type"`
}

// TickerStreamData contains ticker stream data from okex
type TickerStreamData struct {
	Buy       string  `json:"buy"`
	Change    string  `json:"change"`
	High      string  `json:"high"`
	Low       string  `json:"low"`
	Last      string  `json:"last"`
	Sell      string  `json:"sell"`
	DayLow    string  `json:"dayLow"`
	DayHigh   string  `json:"dayHigh"`
	Timestamp float64 `json:"timestamp"`
	Vol       string  `json:"vol"`
}

// DealsStreamData defines Deals data
type DealsStreamData = [][]string

// KlineStreamData defines kline data
type KlineStreamData = [][]string

// DepthStreamData defines orderbook depth
type DepthStreamData struct {
	Asks      [][]string `json:"asks"`
	Bids      [][]string `json:"bids"`
	Timestamp float64    `json:"timestamp"`
}

// ContractDepth response depth
type ContractDepth struct {
	Asks   []interface{} `json:"asks"`
	Bids   []interface{} `json:"bids"`
	Result bool          `json:"result"`
	Error  interface{}   `json:"error_code"`
}

// ActualContractDepth better manipulated structure to return
type ActualContractDepth struct {
	Asks []struct {
		Price  float64
		Volume float64
	}
	Bids []struct {
		Price  float64
		Volume float64
	}
}

// ActualContractTradeHistory holds contract trade history
type ActualContractTradeHistory struct {
	Amount   float64 `json:"amount"`
	DateInMS float64 `json:"date_ms"`
	Date     float64 `json:"date"`
	Price    float64 `json:"price"`
	TID      float64 `json:"tid"`
	Type     string  `json:"buy"`
}

// CandleStickData holds candlestick data
type CandleStickData struct {
	Timestamp float64 `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	Amount    float64 `json:"amount"`
}

// Info holds individual information
type Info struct {
	AccountRights float64 `json:"account_rights"`
	KeepDeposit   float64 `json:"keep_deposit"`
	ProfitReal    float64 `json:"profit_real"`
	ProfitUnreal  float64 `json:"profit_unreal"`
	RiskRate      float64 `json:"risk_rate"`
}

// UserInfo holds a collection of user data
type UserInfo struct {
	Info struct {
		BTC Info `json:"btc"`
		LTC Info `json:"ltc"`
	} `json:"info"`
	Result bool `json:"result"`
}

// HoldData is a sub type for FuturePosition
type HoldData struct {
	BuyAmount      float64 `json:"buy_amount"`
	BuyAvailable   float64 `json:"buy_available"`
	BuyPriceAvg    float64 `json:"buy_price_avg"`
	BuyPriceCost   float64 `json:"buy_price_cost"`
	BuyProfitReal  float64 `json:"buy_profit_real"`
	ContractID     float64 `json:"contract_id"`
	ContractType   string  `json:"contract_type"`
	CreateDate     int     `json:"create_date"`
	LeverRate      float64 `json:"lever_rate"`
	SellAmount     float64 `json:"sell_amount"`
	SellAvailable  float64 `json:"sell_available"`
	SellPriceAvg   float64 `json:"sell_price_avg"`
	SellPriceCost  float64 `json:"sell_price_cost"`
	SellProfitReal float64 `json:"sell_profit_real"`
	Symbol         string  `json:"symbol"`
}

// FuturePosition contains an array of holding types
type FuturePosition struct {
	ForceLiquidationPrice float64    `json:"force_liqu_price"`
	Holding               []HoldData `json:"holding"`
}

// FutureTradeHistory will contain futures trade data
type FutureTradeHistory struct {
	Amount float64 `json:"amount"`
	Date   int     `json:"date"`
	Price  float64 `json:"price"`
	TID    float64 `json:"tid"`
	Type   string  `json:"type"`
}

// SpotPrice holds date and ticker price price for contracts.
type SpotPrice struct {
	Date   string `json:"date"`
	Ticker struct {
		Buy        float64 `json:"buy,string"`
		ContractID float64 `json:"contract_id"`
		High       float64 `json:"high,string"`
		Low        float64 `json:"low,string"`
		Last       float64 `json:"last,string"`
		Sell       float64 `json:"sell,string"`
		UnitAmount float64 `json:"unit_amount,string"`
		Vol        float64 `json:"vol,string"`
	} `json:"ticker"`
	Result bool        `json:"result"`
	Error  interface{} `json:"error_code"`
}

// SpotDepth response depth
type SpotDepth struct {
	Asks   []interface{} `json:"asks"`
	Bids   []interface{} `json:"bids"`
	Result bool          `json:"result"`
	Error  interface{}   `json:"error_code"`
}

// ActualSpotDepthRequestParams represents Klines request data.
type ActualSpotDepthRequestParams struct {
	Symbol string `json:"symbol"` // Symbol; example ltc_btc
	Size   int    `json:"size"`   // value: 1-200
}

// ActualSpotDepth better manipulated structure to return
type ActualSpotDepth struct {
	Asks []struct {
		Price  float64
		Volume float64
	}
	Bids []struct {
		Price  float64
		Volume float64
	}
}

// ActualSpotTradeHistoryRequestParams represents Klines request data.
type ActualSpotTradeHistoryRequestParams struct {
	Symbol string `json:"symbol"` // Symbol; example ltc_btc
	Since  int    `json:"since"`  // TID; transaction record ID (return data does not include the current TID value, returning up to 600 items)
}

// ActualSpotTradeHistory holds contract trade history
type ActualSpotTradeHistory struct {
	Amount   float64 `json:"amount"`
	DateInMS float64 `json:"date_ms"`
	Date     float64 `json:"date"`
	Price    float64 `json:"price"`
	TID      float64 `json:"tid"`
	Type     string  `json:"buy"`
}

// SpotUserInfo holds the spot user info
type SpotUserInfo struct {
	Result bool                                    `json:"result"`
	Info   map[string]map[string]map[string]string `json:"info"`
}

// SpotNewOrderRequestParams holds the params for making a new spot order
type SpotNewOrderRequestParams struct {
	Amount float64                 `json:"amount"` // Order quantity
	Price  float64                 `json:"price"`  // Order price
	Symbol string                  `json:"symbol"` // Symbol; example btc_usdt, eth_btc......
	Type   SpotNewOrderRequestType `json:"type"`   // Order type (see below)
}

// SpotNewOrderRequestType order type
type SpotNewOrderRequestType string

var (
	// SpotNewOrderRequestTypeBuy buy order
	SpotNewOrderRequestTypeBuy = SpotNewOrderRequestType("buy")

	// SpotNewOrderRequestTypeSell sell order
	SpotNewOrderRequestTypeSell = SpotNewOrderRequestType("sell")

	// SpotNewOrderRequestTypeBuyMarket buy market order
	SpotNewOrderRequestTypeBuyMarket = SpotNewOrderRequestType("buy_market")

	// SpotNewOrderRequestTypeSellMarket sell market order
	SpotNewOrderRequestTypeSellMarket = SpotNewOrderRequestType("sell_market")
)

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol string       // Symbol; example btcusdt, bccbtc......
	Type   TimeInterval // Kline data time interval; 1min, 5min, 15min......
	Size   int          // Size; [1-2000]
	Since  int64        // Since timestamp, return data after the specified timestamp (for example, 1417536000000)
}

// TimeInterval represents interval enum.
type TimeInterval string

// vars for time intervals
var (
	TimeIntervalMinute         = TimeInterval("1min")
	TimeIntervalThreeMinutes   = TimeInterval("3min")
	TimeIntervalFiveMinutes    = TimeInterval("5min")
	TimeIntervalFifteenMinutes = TimeInterval("15min")
	TimeIntervalThirtyMinutes  = TimeInterval("30min")
	TimeIntervalHour           = TimeInterval("1hour")
	TimeIntervalFourHours      = TimeInterval("4hour")
	TimeIntervalSixHours       = TimeInterval("6hour")
	TimeIntervalTwelveHours    = TimeInterval("12hour")
	TimeIntervalDay            = TimeInterval("1day")
	TimeIntervalThreeDays      = TimeInterval("3day")
	TimeIntervalWeek           = TimeInterval("1week")
)

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change, using highest value
var WithdrawalFees = map[string]float64{
	symbol.ZRX:   10,
	symbol.ACE:   2.2,
	symbol.ACT:   0.01,
	symbol.AAC:   5,
	symbol.AE:    1,
	symbol.AIDOC: 17,
	symbol.AST:   8,
	symbol.SOC:   20,
	symbol.ABT:   3,
	symbol.ARK:   0.1,
	symbol.ATL:   1.5,
	symbol.AVT:   1,
	symbol.BNT:   1,
	symbol.BKX:   3,
	symbol.BEC:   4,
	symbol.BTC:   0.0005,
	symbol.BCH:   0.0001,
	symbol.BCD:   0.02,
	symbol.BTG:   0.001,
	symbol.VEE:   100,
	symbol.BRD:   1.5,
	symbol.CTR:   7,
	symbol.LINK:  10,
	symbol.CAG:   2,
	symbol.CHAT:  10,
	symbol.CVC:   10,
	symbol.CIC:   150,
	symbol.CBT:   10,
	symbol.CAN:   3,
	symbol.CMT:   10,
	symbol.DADI:  10,
	symbol.DASH:  0.002,
	symbol.DAT:   2,
	symbol.MANA:  20,
	symbol.DCR:   0.03,
	symbol.DPY:   0.8,
	symbol.DENT:  100,
	symbol.DGD:   0.2,
	symbol.DNT:   20,
	symbol.EDO:   2,
	symbol.DNA:   3,
	symbol.ENG:   5,
	symbol.ENJ:   20,
	symbol.ETH:   0.01,
	symbol.ETC:   0.001,
	symbol.LEND:  10,
	symbol.EVX:   1.5,
	symbol.XUC:   5.8,
	symbol.FAIR:  15,
	symbol.FIRST: 6,
	symbol.FUN:   40,
	symbol.GTC:   40,
	symbol.GNX:   8,
	symbol.GTO:   10,
	symbol.GSC:   20,
	symbol.GNT:   5,
	symbol.HMC:   40,
	symbol.HOT:   10,
	symbol.ICN:   2,
	symbol.INS:   2.5,
	symbol.INT:   10,
	symbol.IOST:  100,
	symbol.ITC:   2,
	symbol.IPC:   2.5,
	symbol.KNC:   2,
	symbol.LA:    3,
	symbol.LEV:   20,
	symbol.LIGHT: 100,
	symbol.LSK:   0.4,
	symbol.LTC:   0.001,
	symbol.LRC:   7,
	symbol.MAG:   34,
	symbol.MKR:   0.002,
	symbol.MTL:   0.5,
	symbol.AMM:   5,
	symbol.MITH:  20,
	symbol.MDA:   2,
	symbol.MOF:   5,
	symbol.MCO:   0.2,
	symbol.MTH:   35,
	symbol.NGC:   1.5,
	symbol.NANO:  0.2,
	symbol.NULS:  2,
	symbol.OAX:   6,
	symbol.OF:    600,
	symbol.OKB:   0,
	symbol.MOT:   1.5,
	symbol.OMG:   0.1,
	symbol.RNT:   13,
	symbol.POE:   30,
	symbol.PPT:   0.2,
	symbol.PST:   10,
	symbol.PRA:   4,
	symbol.QTUM:  0.01,
	symbol.QUN:   20,
	symbol.QVT:   10,
	symbol.RDN:   0.3,
	symbol.READ:  20,
	symbol.RCT:   15,
	symbol.RFR:   200,
	symbol.REF:   0.2,
	symbol.REN:   50,
	symbol.REQ:   15,
	symbol.R:     2,
	symbol.RCN:   20,
	symbol.XRP:   0.15,
	symbol.SALT:  0.5,
	symbol.SAN:   1,
	symbol.KEY:   50,
	symbol.SSC:   8,
	symbol.SHOW:  150,
	symbol.SC:    200,
	symbol.OST:   3,
	symbol.SNGLS: 20,
	symbol.SMT:   8,
	symbol.SNM:   20,
	symbol.SPF:   5,
	symbol.SNT:   50,
	symbol.STORJ: 2,
	symbol.SUB:   4,
	symbol.SNC:   10,
	symbol.SWFTC: 350,
	symbol.PAY:   0.5,
	symbol.USDT:  2,
	symbol.TRA:   500,
	symbol.THETA: 20,
	symbol.TNB:   40,
	symbol.TCT:   50,
	symbol.TOPC:  20,
	symbol.TIO:   2.5,
	symbol.TRIO:  200,
	symbol.TRUE:  4,
	symbol.UCT:   10,
	symbol.UGC:   12,
	symbol.UKG:   2.5,
	symbol.UTK:   3,
	symbol.VIB:   6,
	symbol.VIU:   40,
	symbol.WTC:   0.4,
	symbol.WFEE:  500,
	symbol.WRC:   48,
	symbol.YEE:   70,
	symbol.YOYOW: 10,
	symbol.ZEC:   0.001,
	symbol.ZEN:   0.07,
	symbol.ZIL:   20,
	symbol.ZIP:   1000,
}

// FullBalance defines a structured return type with balance data
type FullBalance struct {
	Available float64
	Currency  string
	Hold      float64
}

// Balance defines returned balance data
type Balance struct {
	Info struct {
		Funds struct {
			Free  map[string]string `json:"free"`
			Holds map[string]string `json:"holds"`
		} `json:"funds"`
	} `json:"info"`
}

// WithdrawalResponse is a response type for withdrawal
type WithdrawalResponse struct {
	WithdrawID int  `json:"withdraw_id"`
	Result     bool `json:"result"`
}
