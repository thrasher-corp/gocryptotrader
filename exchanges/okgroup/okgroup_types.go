package okgroup

import "encoding/json"
import "github.com/thrasher-/gocryptotrader/currency/symbol"

// CurrencyResponse contains currency details from a GetCurrencies request
type CurrencyResponse struct {
	CanDeposit    int64   `json:"can_deposit"`
	CanWithdraw   int64   `json:"can_withdraw"`
	Currency      string  `json:"currency"`
	MinWithdrawal float64 `json:"min_withdrawal"`
	Name          string  `json:"name"`
}

// WalletInformationResponse contains wallet details from a GetWalletInformation request
type WalletInformationResponse struct {
	Available float64 `json:"available"`
	Balance   float64 `json:"balance"`
	Currency  string  `json:"currency"`
	Hold      float64 `json:"hold"`
}

// FundTransferRequest used to request a fund transfer
type FundTransferRequest struct {
	Currency     string  `json:"currency"`
	Amount       float64 `json:"amount"`
	From         int64   `json:"from"`
	To           int64   `json:"to"`
	SubAccountID string  `json:"sub_account,omitempty"`
	InstrumentID int64   `json:"instrument_id,omitempty"`
}

// FundTransferResponse the response after a FundTransferRequest
type FundTransferResponse struct {
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	From       int     `json:"from"`
	Result     bool    `json:"result"`
	To         int     `json:"to"`
	TransferID int     `json:"transfer_id"`
}

// WithdrawRequest used to request a withdrawal
type WithdrawRequest struct {
	Amount      int     `json:"amount"`
	Currency    string  `json:"currency"`
	Destination int     `json:"destination"`
	Fee         float64 `json:"fee"`
	ToAddress   string  `json:"to_address"`
	TradePwd    string  `json:"trade_pwd"`
}

// WithdrawResponse the response after a WithdrawRequest
type WithdrawResponse struct {
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Result       bool    `json:"result"`
	WithdrawalID int     `json:"withdrawal_id"`
}

// WithdrawalFeeResponse the response after requesting withdrawal fees
type WithdrawalFeeResponse struct {
	Available float64 `json:"available"`
	Balance   float64 `json:"balance"`
	Currency  string  `json:"currency"`
	Hold      float64 `json:"hold"`
}

// WithdrawalHistoryResponse the response after requesting withdrawal history
type WithdrawalHistoryResponse struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Fee       string  `json:"fee"`
	From      string  `json:"from"`
	Status    int     `json:"status"`
	Timestamp string  `json:"timestamp"`
	To        string  `json:"to"`
	Txid      string  `json:"txid"`
	PaymentID string  `json:"payment_id"`
	Tag       string  `json:"tag"`
}

// GetBillDetailsRequest used in GetBillDetails
type GetBillDetailsRequest struct {
	Currency string
	Type     int64
	From     int64
	To       int64
	Limit    int64
}

// GetBillDetailsResponse contains bill details from a GetBillDetailsRequest request
type GetBillDetailsResponse struct {
	Amount    float64 `json:"amount"`
	Balance   int     `json:"balance"`
	Currency  string  `json:"currency"`
	Fee       int     `json:"fee"`
	LedgerID  int     `json:"ledger_id"`
	Timestamp string  `json:"timestamp"`
	Typename  string  `json:"typename"`
}

// OrderStatus Holds OKGroup order status values
var OrderStatus = map[int]string{
	-3: "pending cancel",
	-2: "cancelled",
	-1: "failed",
	0:  "pending",
	1:  "sending",
	2:  "sent",
	3:  "email confirmation",
	4:  "manual confirmation",
	5:  "awaiting identity confirmation",
}

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
// MultiStreamData contains raw data from okex
type MultiStreamData struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
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
