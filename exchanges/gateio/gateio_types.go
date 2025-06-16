package gateio

import (
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// Order time in force variables
	gtcTIF = "gtc" // good-'til-canceled
	iocTIF = "ioc" // immediate-or-cancel
	pocTIF = "poc" // pending-or-cancel - post only
	fokTIF = "fok" // fill-or-kill

	// Frequently used order Status
	statusOpen     = "open"
	statusLoaned   = "loaned"
	statusFinished = "finished"

	// Loan sides
	sideLend   = "lend"
	sideBorrow = "borrow"
)

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[currency.Code]float64{
	currency.BTC:      0.001,
	currency.ETH:      0.003,
	currency.USDT:     40,
	currency.USDC:     2,
	currency.BUSD:     5,
	currency.ADA:      3.8,
	currency.SOL:      0.11,
	currency.DOT:      .25,
	currency.DOGE:     29,
	currency.MATIC:    2.2,
	currency.STETH:    0.023,
	currency.DAI:      24,
	currency.SHIB:     420000,
	currency.AVAX:     0.083,
	currency.TRX:      1,
	currency.WBTC:     0.0011,
	currency.ETC:      0.051,
	currency.OKB:      1.6,
	currency.LTC:      0.002,
	currency.UNI:      3.2,
	currency.LINK:     3.2,
	currency.ATOM:     0.19,
	currency.XLM:      0.01,
	currency.XMR:      0.013,
	currency.BCH:      0.014,
	currency.ALGO:     6,
	currency.ICP:      0.25,
	currency.FLOW:     2.6,
	currency.VET:      100,
	currency.MANA:     26,
	currency.SAND:     19,
	currency.AXS:      1.3,
	currency.HBAR:     1,
	currency.XTZ:      1.1,
	currency.FRAX:     25,
	currency.QNT:      0.24,
	currency.THETA:    1.5,
	currency.AAVE:     1,
	currency.EOS:      1.5,
	currency.EGLD:     0.052,
	currency.BSV:      0.032,
	currency.TUSD:     2,
	currency.HNT:      0.21,
	currency.MKR:      0.0045,
	currency.GRT:      190,
	currency.KLAY:     6.4,
	currency.BTT:      1,
	currency.MIOTA:    0.1,
	currency.XEC:      10000,
	currency.FTM:      6,
	currency.FTM:      6,
	currency.SNX:      13,
	currency.ZEC:      0.031,
	currency.NEO:      0,
	currency.BIT:      45,
	currency.AR:       0.5,
	currency.HT:       1.1,
	currency.AMP:      7000,
	currency.CHZ:      16,
	currency.GT:       0.22,
	currency.TENSET:   0.22,
	currency.ZIL:      48,
	currency.BAT:      12,
	currency.BTG:      0.061,
	currency.GMT:      5.1,
	currency.ENJ:      42,
	currency.STX:      5,
	currency.CAKE:     0.77,
	currency.KSM:      0.032,
	currency.WAVES:    0.36,
	currency.DASH:     0.04,
	currency.LRC:      59,
	currency.CRV:      18,
	currency.FXS:      7.8,
	currency.CVX:      3.8,
	currency.CVX:      3.3,
	currency.RVN:      1,
	currency.CELO:     1.9,
	currency.CEL:      25,
	currency.QTUM:     0.47,
	currency.KAVA:     1.1,
	currency.XEM:      25,
	currency.ONEINCH:  6.6,
	currency.XAUT:     0.0028,
	currency.ROSE:     0.1,
	currency.GNO:      0.16,
	currency.GALA:     440,
	currency.NEXO:     34,
	currency.COMP:     0.47,
	currency.HOT:      2200,
	currency.DCR:      0.073,
	currency.OP:       1.1,
	currency.ENS:      1.9,
	currency.SRM:      24,
	currency.YFI:      0.0022,
	currency.TFUEL:    34,
	currency.IOST:     140,
	currency.TWT:      2.1,
	currency.IOTX:     60,
	currency.LPT:      2.2,
	currency.ZRX:      72,
	currency.SYN:      17,
	currency.ONE:      85,
	currency.SUSHI:    16,
	currency.SAFEMOON: 28000000,
	currency.IMX:      22,
	currency.JST:      170,
	currency.DYDX:     40,
	currency.GLM:      94,
	currency.LUNA:     2.7,
	currency.AUDIO:    85,
	currency.ICX:      6.5,
	currency.ANKR:     840,
	currency.ONT:      1,
	currency.NU:       43,
	currency.WAXP:     19,
	currency.SC:       450,
	currency.BAL:      4.3,
	currency.ZEN:      0.15,
	currency.SGB:      77,
	currency.SKL:      830,
	currency.EURT:     29,
	currency.UMA:      15,
	currency.XCH:      0.046,
	currency.FEI:      28,
	currency.HIVE:     3.8,
	currency.SCRT:     1.8,
	currency.ELON:     70000000,
	currency.CSPR:     62,
	currency.SLP:      5700,
	currency.MXC:      310,
	currency.NFT:      8100000,
	currency.BTCST:    0.22,
	currency.ASTR:     44,
	currency.PLA:      68,
	currency.LSK:      0.11,
	currency.FX:       16,
	currency.YGG:      5.9,
	currency.METIS:    0.1,
	currency.CKB:      450,
	currency.REN:      180,
	currency.RLY:      570,
	currency.FLUX:     10,
	currency.PROM:     3.3,
	currency.RACA:     7400,
	currency.XYO:      2500,
	currency.ACA:      7.3,
	currency.SUSD:     54,
	currency.RSR:      3900,
	currency.NEST:     1000,
	currency.ORBS:     580,
	currency.WIN:      38000,
	currency.ERG:      0.93,
	currency.SNT:      1700,
	currency.WRX:      7.4,
	currency.CHR:      120,
	currency.MED:      100,
	currency.BNT:      46,
	currency.CVC:      160,
	currency.SYS:      11,
	currency.CELR:     1300,
	currency.FLOKI:    3100000,
	currency.COTI:     240,
	currency.CFX:      0.01,
	currency.API3:     13,
	currency.PUNDIX:   68,
	currency.OGN:      130,
	currency.RAY:      5.9,
	currency.NMR:      0.29,
	currency.POWR:     100,
	currency.DENT:     24000,
	currency.VTHO:     1000,
	currency.MBOX:     4.4,
	currency.DKA:      930,
	currency.VGX:      70,
	currency.REQ:      38,
	currency.CTSI:     150,
	currency.KEEP:     39,
	currency.STRAX:    5,
	currency.STEEM:    8,
	currency.RAD:      11,
	currency.STORJ:    7.4,
	currency.MLK:      0.5,
	currency.VLX:      48,
	currency.BOBA:     49,
	currency.C98:      9.8,
	currency.INJ:      1.4,
	currency.XVS:      0.46,
	currency.MTL:      18,
	currency.FUN:      4000,
	currency.BFC:      320,
	currency.OCEAN:    190,
	currency.UOS:      15,
	currency.RENBTC:   0.0012,
	currency.MULTI:    5.5,
	currency.RBN:      97,
	currency.ILV:      0.043,
	currency.ILM:      170,
	currency.FLM:      9,
	currency.HUSD:     27,
	currency.EFI:      27,
	currency.MDX:      56,
	currency.YFII:     0.011,
	currency.ELF:      12,
	currency.MASK:     15,
	currency.SFUND:    1.4,
	currency.ACH:      320,
	currency.QKC:      180,
	currency.STMX:     3200,
	currency.ANT:      12,
	currency.TRIBE:    170,
	currency.BAND:     1.1,
	currency.MOVR:     0.14,
	currency.DODO:     150,
	currency.RLC:      28,
	currency.DOCK:     74,
	currency.NKN:      19,
	currency.OXT:      210,
	currency.IQ:       20,
	currency.UFO:      9600000,
	currency.TRB:      0.18,
	currency.REP:      4,
	currency.HERO:     1500,
	currency.AKT:      5.2,
	currency.GHST:     47,
	currency.UTK:      180,
	currency.KP3R:     0.16,
	currency.BAKE:     9.3,
	currency.BETA:     180,
	currency.AUCTION:  3.1,
	currency.PERP:     28,
	currency.BOND:     2.9,
	currency.RIDE:     10,
	currency.XVG:      550,
	currency.FET:      23,
	currency.DUSK:     34,
	currency.SSV:      2.9,
	currency.BCN:      2100,
	currency.POLS:     42,
	currency.TALK:     59,
	currency.VRA:      6000,
	currency.POND:     1900,
	currency.RGT:      2.1,
	currency.ATA:      120,
	currency.ALCX:     0.71,
	currency.AERGO:    210,
	currency.MNGO:     100,
	currency.OUSD:     32,
	currency.TOMO:     3.4,
	currency.COCOS:    2.6,
	currency.IDEX:     65,
	currency.VEGA:     12,
	currency.CUSD:     2,
	currency.TT:       1,
	currency.WNXM:     1.4,
	currency.NSBT:     0.3,
	currency.CQT:      200,
	currency.WOZX:     280,
	currency.BEL:      32,
	currency.FORTH:    4.6,
	currency.ALICE:    8.9,
	currency.KISHU:    2000000000,
	currency.ALEPH:    96,
	currency.UNFI:     3.9,
	currency.ORN:      18,
	currency.SUPER:    170,
	currency.STARL:    5300000,
	currency.BADGER:   13,
	currency.JASMY:    520,
	currency.DG:       320,
	currency.RARE:     98,
	currency.XPR:      530,
	currency.PHA:      200,
	currency.MFT:      5700,
	currency.SAMO:     410,
	currency.SFP:      7.7,
	currency.ALPACA:   11,
	currency.GAS:      0.69,
	currency.TORN:     0.95,
	currency.DNT:      920,
	currency.ANC:      44,
	currency.MLN:      0.18,
	currency.KAR:      3.4,
	currency.FARM:     0.41,
	currency.LTO:      290,
	currency.HYDRA:    0.67,
	currency.QASH:     540,
	currency.AE:       21,
	currency.LINA:     3700,
	currency.ARPA:     680,
	currency.AQT:      20,
	currency.XCAD:     3.3,
	currency.DIA:      55,
	currency.LIT:      26,
	currency.AVA:      2.9,
	currency.BZZ:      41,
	currency.AGLD:     51,
	currency.BLZ:      250,
	currency.BCD:      11,
	currency.CEUR:     2,
	currency.NOIA:     390,
	currency.FINE:     110,
	currency.ERN:      12,
	currency.RMRK:     0.57,
	currency.MIR:      120,
	currency.BTS:      170,
	currency.CHESS:    7.3,
	currency.HNS:      32,
	currency.FIO:      38,
	currency.IRIS:     83,
	currency.RFR:      8600,
	currency.RARI:     7.9,
	currency.FIDA:     9.9,
	currency.QRDO:     75,
	currency.GYEN:     1000,
	currency.SPS:      50,
	currency.KEY:      5400,
	currency.ATM:      1,
	currency.SOUL:     7.5,
	currency.PRQ:      160,
	currency.FRONT:    81,
	currency.NCT:      1400,
	currency.PSG:      0.33,
	currency.BOO:      .7,
	currency.RSV:      29,
	currency.CUDOS:    600,
	currency.NPXS:     40,
	currency.OM:       92,
	currency.ADX:      27,
	currency.AUTO:     .0087,
	currency.SAITO:    2000,
	currency.COS:      270,
	currency.VELO:     99,
	currency.FIS:      4.6,
	currency.NULS:     8.2,
	currency.UPP:      20,
	currency.XDB:      .01,
	currency.LUFFY:    140000000000,
	currency.TKO:      9.7,
	currency.KIN:      410000,
	currency.GFI:      22,
	currency.MIX:      6100,
	currency.TIME:     .014,
	currency.HOPR:     570,
	currency.BEAM:     11,
	currency.BTM:      160,
	currency.OVR:      29,
	currency.CITY:     1,
	currency.CATE:     50000000,
	currency.DEXE:     13,
	currency.ORCA:     5.1,
	currency.MDT:      150,
	currency.PNK:      1500,
	currency.QSP:      180,
	currency.DVI:      65,
	currency.DF:       610,
	currency.INV:      .24,
	currency.TABOO:    45000,
	currency.FSN:      8,
	currency.SDN:      6.1,
	currency.LON:      33,
	currency.MITH:     850,
	currency.ATLAS:    630,
	currency.LAZIO:    1.1,
	currency.MBL:      420,
	currency.PNT:      100,
	currency.WXT:      280,
	currency.NBS:      390,
	currency.WHALE:    14,
	currency.BOA:      490,
	currency.SWFTC:    11000,
	currency.JUV:      1,
	currency.MAPS:     130,
	currency.ADP:      1600,
	currency.AST:      60,
	currency.EDEN:     190,
	currency.WICC:     30,
	currency.UFT:      110,
	currency.ZKS:      380,
	currency.CREAM:    1.5,
	currency.MET:      26,
	currency.RAI:      9.4,
	currency.XAVA:     3.9,
	currency.FOR:      1000,
	currency.AVT:      18,
	currency.SOV:      53,
	currency.SOS:      78000000,
	currency.LSS:      160,
	currency.NFTX:     .13,
	currency.DEGO:     23,
	currency.DERC:     82,
	currency.CHAIN:    760,
	currency.POLIS:    9.3,
	currency.PDEX:     1.3,
	currency.SUKU:     200,
	currency.ARV:      20000,
	currency.REVV:     1300,
	currency.GO:       220,
	currency.OOE:      83,
	currency.EDG:      1300,
	currency.STEP:     120,
	currency.BORING:   480,
	currency.STC:      55,
	currency.OCC:      55,
	currency.SHFT:     84,
	currency.AIR:      79,
	currency.URUS:     1.2,
	currency.SLIM:     51,
	currency.HAI:      100,
	currency.ZCN:      120,
	currency.ABT:      53,
	currency.NWC:      140,
	currency.STAKE:    2.7,
	currency.OPUL:     60,
	currency.RBC:      340,
	currency.BAO:      230000,
	currency.TCT:      1600,
	currency.WTC:      .2,
	currency.NUM:      730,
	currency.DRGN:     1100,
	currency.POSI:     99,
	currency.TROY:     6100,
	currency.ASR:      1,
	currency.TBTC:     .0011,
	currency.GEL:      11,
	currency.GRIN:     28,
	currency.AFC:      1,
	currency.KAN:      20,
	currency.OG:       1,
	currency.XED:      340,
	currency.FEVR:     2900,
	currency.HEGIC:    510,
	currency.SBR:      810,
	currency.HAPI:     2.6,
	currency.PING:     33000,
	currency.REF:      12,
	currency.BUY:      100,
	currency.INSUR:    290,
	currency.PUSH:     79,
}

// CurrencyInfo represents currency details with permission.
type CurrencyInfo struct {
	Currency         string       `json:"currency"`
	Delisted         bool         `json:"delisted"`
	WithdrawDisabled bool         `json:"withdraw_disabled"`
	WithdrawDelayed  bool         `json:"withdraw_delayed"`
	DepositDisabled  bool         `json:"deposit_disabled"`
	TradeDisabled    bool         `json:"trade_disabled"`
	FixedFeeRate     types.Number `json:"fixed_rate,omitempty"`
	Chain            string       `json:"chain"`
}

// CurrencyPairDetail represents a single currency pair detail.
type CurrencyPairDetail struct {
	ID              string       `json:"id"`
	Base            string       `json:"base"`
	Quote           string       `json:"quote"`
	TradingFee      types.Number `json:"fee"`
	MinBaseAmount   types.Number `json:"min_base_amount"`
	MinQuoteAmount  types.Number `json:"min_quote_amount"`
	AmountPrecision float64      `json:"amount_precision"` // Amount scale
	Precision       float64      `json:"precision"`        // Price scale
	TradeStatus     string       `json:"trade_status"`
	SellStart       float64      `json:"sell_start"`
	BuyStart        float64      `json:"buy_start"`
}

// Ticker holds detail ticker information for a currency pair
type Ticker struct {
	CurrencyPair     string       `json:"currency_pair"`
	Last             types.Number `json:"last"`
	LowestAsk        types.Number `json:"lowest_ask"`
	HighestBid       types.Number `json:"highest_bid"`
	ChangePercentage string       `json:"change_percentage"`
	ChangeUtc0       string       `json:"change_utc0"`
	ChangeUtc8       string       `json:"change_utc8"`
	BaseVolume       types.Number `json:"base_volume"`
	QuoteVolume      types.Number `json:"quote_volume"`
	High24H          types.Number `json:"high_24h"`
	Low24H           types.Number `json:"low_24h"`
	EtfNetValue      string       `json:"etf_net_value"`
	EtfPreNetValue   string       `json:"etf_pre_net_value"`
	EtfPreTimestamp  types.Time   `json:"etf_pre_timestamp"`
	EtfLeverage      types.Number `json:"etf_leverage"`
}

// OrderbookData holds orderbook ask and bid datas.
type OrderbookData struct {
	ID      int64             `json:"id"`
	Current types.Time        `json:"current"` // The timestamp of the response data being generated (in milliseconds)
	Update  types.Time        `json:"update"`  // The timestamp of when the orderbook last changed (in milliseconds)
	Asks    [][2]types.Number `json:"asks"`
	Bids    [][2]types.Number `json:"bids"`
}

// MakeOrderbook converts OrderbookData into an Orderbook
func (a *OrderbookData) MakeOrderbook() *Orderbook {
	asks := make([]OrderbookItem, len(a.Asks))
	for x := range a.Asks {
		asks[x].Price = a.Asks[x][0]
		asks[x].Amount = a.Asks[x][1]
	}
	bids := make([]OrderbookItem, len(a.Bids))
	for x := range a.Bids {
		bids[x].Price = a.Bids[x][0]
		bids[x].Amount = a.Bids[x][1]
	}
	return &Orderbook{ID: a.ID, Current: a.Current, Update: a.Update, Asks: asks, Bids: bids}
}

// OrderbookItem stores an orderbook item
type OrderbookItem struct {
	Price  types.Number `json:"p"`
	Amount types.Number `json:"s"`
}

// Orderbook stores the orderbook data
type Orderbook struct {
	ID      int64           `json:"id"`
	Current types.Time      `json:"current"` // The timestamp of the response data being generated (in milliseconds)
	Update  types.Time      `json:"update"`  // The timestamp of when the orderbook last changed (in milliseconds)
	Bids    []OrderbookItem `json:"bids"`
	Asks    []OrderbookItem `json:"asks"`
}

// Trade represents market trade.
type Trade struct {
	ID          int64        `json:"id,string"`
	CreateTime  types.Time   `json:"create_time_ms"`
	OrderID     string       `json:"order_id"`
	Side        string       `json:"side"`
	Role        string       `json:"role"`
	Amount      types.Number `json:"amount"`
	Price       types.Number `json:"price"`
	Fee         types.Number `json:"fee"`
	FeeCurrency string       `json:"fee_currency"`
	PointFee    string       `json:"point_fee"`
	GtFee       string       `json:"gt_fee"`
}

// Candlestick represents candlestick data point detail.
type Candlestick struct {
	Timestamp      types.Time
	QuoteCcyVolume types.Number
	ClosePrice     types.Number
	HighestPrice   types.Number
	LowestPrice    types.Number
	OpenPrice      types.Number
	BaseCcyAmount  types.Number
	WindowClosed   bool
}

// UnmarshalJSON parses kline data from a JSON array into Candlestick fields.
func (c *Candlestick) UnmarshalJSON(data []byte) error {
	var windowClosed string
	err := json.Unmarshal(data, &[8]any{&c.Timestamp, &c.QuoteCcyVolume, &c.ClosePrice, &c.HighestPrice, &c.LowestPrice, &c.OpenPrice, &c.BaseCcyAmount, &windowClosed})
	if err != nil {
		return err
	}
	c.WindowClosed, err = strconv.ParseBool(windowClosed)
	return err
}

// CurrencyChain currency chain detail.
type CurrencyChain struct {
	Chain              string `json:"chain"`
	ChineseChainName   string `json:"name_cn"`
	ChainName          string `json:"name_en"`
	IsDisabled         int64  `json:"is_disabled"`          // If it is disabled. 0 means NOT being disabled
	IsDepositDisabled  int64  `json:"is_deposit_disabled"`  // Is deposit disabled. 0 means not
	IsWithdrawDisabled int64  `json:"is_withdraw_disabled"` // Is withdrawal disabled. 0 means not
}

// MarginCurrencyPairInfo represents margin currency pair detailed info.
type MarginCurrencyPairInfo struct {
	ID             string       `json:"id"`
	Base           string       `json:"base"`
	Quote          string       `json:"quote"`
	Leverage       float64      `json:"leverage"`
	MinBaseAmount  types.Number `json:"min_base_amount"`
	MinQuoteAmount types.Number `json:"min_quote_amount"`
	MaxQuoteAmount types.Number `json:"max_quote_amount"`
	Status         int32        `json:"status"`
}

// OrderbookOfLendingLoan represents order book of lending loans
type OrderbookOfLendingLoan struct {
	Rate   types.Number `json:"rate"`
	Amount types.Number `json:"amount"`
	Days   int64        `json:"days"`
}

// FuturesContract represents futures contract detailed data.
type FuturesContract struct {
	Name                  string       `json:"name"`
	Type                  string       `json:"type"`
	QuantoMultiplier      types.Number `json:"quanto_multiplier"`
	RefDiscountRate       types.Number `json:"ref_discount_rate"`
	OrderPriceDeviate     string       `json:"order_price_deviate"`
	MaintenanceRate       types.Number `json:"maintenance_rate"`
	MarkType              string       `json:"mark_type"`
	LastPrice             types.Number `json:"last_price"`
	MarkPrice             types.Number `json:"mark_price"`
	IndexPrice            types.Number `json:"index_price"`
	FundingRateIndicative types.Number `json:"funding_rate_indicative"`
	MarkPriceRound        types.Number `json:"mark_price_round"`
	FundingOffset         int64        `json:"funding_offset"`
	InDelisting           bool         `json:"in_delisting"`
	RiskLimitBase         string       `json:"risk_limit_base"`
	InterestRate          string       `json:"interest_rate"`
	OrderPriceRound       string       `json:"order_price_round"`
	OrderSizeMin          int64        `json:"order_size_min"`
	RefRebateRate         string       `json:"ref_rebate_rate"`
	FundingInterval       int64        `json:"funding_interval"`
	RiskLimitStep         string       `json:"risk_limit_step"`
	LeverageMin           types.Number `json:"leverage_min"`
	LeverageMax           types.Number `json:"leverage_max"`
	RiskLimitMax          string       `json:"risk_limit_max"`
	MakerFeeRate          types.Number `json:"maker_fee_rate"`
	TakerFeeRate          types.Number `json:"taker_fee_rate"`
	FundingRate           types.Number `json:"funding_rate"`
	OrderSizeMax          int64        `json:"order_size_max"`
	FundingNextApply      types.Time   `json:"funding_next_apply"`
	ConfigChangeTime      types.Time   `json:"config_change_time"`
	ShortUsers            int64        `json:"short_users"`
	TradeSize             int64        `json:"trade_size"`
	PositionSize          int64        `json:"position_size"`
	LongUsers             int64        `json:"long_users"`
	FundingImpactValue    string       `json:"funding_impact_value"`
	OrdersLimit           int64        `json:"orders_limit"`
	TradeID               int64        `json:"trade_id"`
	OrderbookID           int64        `json:"orderbook_id"`
	EnableBonus           bool         `json:"enable_bonus"`
	EnableCredit          bool         `json:"enable_credit"`
	CreateTime            types.Time   `json:"create_time"`
	FundingCapRatio       types.Number `json:"funding_cap_ratio"`
	VoucherLeverage       types.Number `json:"voucher_leverage"`
}

// TradingHistoryItem represents futures trading history item.
type TradingHistoryItem struct {
	ID         int64        `json:"id"`
	CreateTime types.Time   `json:"create_time_ms"`
	Contract   string       `json:"contract"`
	Text       string       `json:"text"`
	Size       float64      `json:"size"`
	Price      types.Number `json:"price"`
	// Added for Derived market trade history data
	Fee      types.Number `json:"fee"`
	PointFee types.Number `json:"point_fee"`
	Role     string       `json:"role"`
}

// FuturesCandlestick represents futures candlestick data
type FuturesCandlestick struct {
	Timestamp    types.Time   `json:"t"`
	Volume       float64      `json:"v"`
	ClosePrice   types.Number `json:"c"`
	HighestPrice types.Number `json:"h"`
	LowestPrice  types.Number `json:"l"`
	OpenPrice    types.Number `json:"o"`
	Sum          types.Number `json:"sum"` // Trading volume (unit: Quote currency)
	Name         string       `json:"n,omitempty"`
}

// FuturesPremiumIndexKLineResponse represents premium index K-Line information.
type FuturesPremiumIndexKLineResponse struct {
	UnixTimestamp types.Time   `json:"t"`
	ClosePrice    types.Number `json:"c"`
	HighestPrice  types.Number `json:"h"`
	LowestPrice   types.Number `json:"l"`
	OpenPrice     types.Number `json:"o"`
}

// FuturesTicker represents futures ticker data.
type FuturesTicker struct {
	Contract              string       `json:"contract"`
	ChangePercentage      string       `json:"change_percentage"`
	Last                  types.Number `json:"last"`
	Low24H                types.Number `json:"low_24h"`
	High24H               types.Number `json:"high_24h"`
	TotalSize             types.Number `json:"total_size"`
	Volume24H             types.Number `json:"volume_24h"`
	Volume24HBtc          types.Number `json:"volume_24h_btc"`
	Volume24HUsd          types.Number `json:"volume_24h_usd"`
	Volume24HBase         types.Number `json:"volume_24h_base"`
	Volume24HQuote        types.Number `json:"volume_24h_quote"`
	Volume24HSettle       types.Number `json:"volume_24h_settle"`
	MarkPrice             types.Number `json:"mark_price"`
	FundingRate           types.Number `json:"funding_rate"`
	FundingRateIndicative string       `json:"funding_rate_indicative"`
	IndexPrice            types.Number `json:"index_price"`
}

// FuturesFundingRate represents futures funding rate response.
type FuturesFundingRate struct {
	Timestamp types.Time   `json:"t"`
	Rate      types.Number `json:"r"`
}

// InsuranceBalance represents futures insurance balance item.
type InsuranceBalance struct {
	Timestamp types.Time `json:"t"`
	Balance   float64    `json:"b"`
}

// ContractStat represents futures stats
type ContractStat struct {
	Time                   types.Time `json:"time"`
	LongShortTaker         float64    `json:"lsr_taker"`
	LongShortAccount       float64    `json:"lsr_account"`
	LongLiqSize            float64    `json:"long_liq_size"`
	ShortLiquidationSize   float64    `json:"short_liq_size"`
	OpenInterest           float64    `json:"open_interest"`
	ShortLiquidationUsd    float64    `json:"short_liq_usd"`
	MarkPrice              float64    `json:"mark_price"`
	TopLongShortSize       float64    `json:"top_lsr_size"`
	ShortLiquidationAmount float64    `json:"short_liq_amount"`
	LongLiquidiationAmount float64    `json:"long_liq_amount"`
	OpenInterestUsd        float64    `json:"open_interest_usd"`
	TopLongShortAccount    float64    `json:"top_lsr_account"`
	LongLiquidationUSD     float64    `json:"long_liq_usd"`
}

// IndexConstituent represents index constituents
type IndexConstituent struct {
	Index        string `json:"index"`
	Constituents []struct {
		Exchange string   `json:"exchange"`
		Symbols  []string `json:"symbols"`
	} `json:"constituents"`
}

// LiquidationHistory represents  liquidation history for a specifies settle.
type LiquidationHistory struct {
	Time             types.Time   `json:"time"`
	Contract         string       `json:"contract"`
	Size             int64        `json:"size"`
	Leverage         string       `json:"leverage"`
	Margin           string       `json:"margin"`
	EntryPrice       types.Number `json:"entry_price"`
	LiquidationPrice types.Number `json:"liq_price"`
	MarkPrice        types.Number `json:"mark_price"`
	OrderID          int64        `json:"order_id"`
	OrderPrice       types.Number `json:"order_price"`
	FillPrice        types.Number `json:"fill_price"`
	Left             int64        `json:"left"`
}

// DeliveryContract represents a delivery contract instance detail.
type DeliveryContract struct {
	Name                string       `json:"name"`
	Underlying          string       `json:"underlying"`
	Cycle               string       `json:"cycle"`
	Type                string       `json:"type"`
	QuantoMultiplier    types.Number `json:"quanto_multiplier"`
	MarkType            string       `json:"mark_type"`
	LastPrice           types.Number `json:"last_price"`
	MarkPrice           types.Number `json:"mark_price"`
	IndexPrice          types.Number `json:"index_price"`
	BasisRate           types.Number `json:"basis_rate"`
	BasisValue          types.Number `json:"basis_value"`
	BasisImpactValue    types.Number `json:"basis_impact_value"`
	SettlePrice         types.Number `json:"settle_price"`
	SettlePriceInterval int64        `json:"settle_price_interval"`
	SettlePriceDuration int64        `json:"settle_price_duration"`
	SettleFeeRate       types.Number `json:"settle_fee_rate"`
	OrderPriceRound     types.Number `json:"order_price_round"`
	MarkPriceRound      types.Number `json:"mark_price_round"`
	LeverageMin         types.Number `json:"leverage_min"`
	LeverageMax         types.Number `json:"leverage_max"`
	MaintenanceRate     types.Number `json:"maintenance_rate"`
	RiskLimitBase       types.Number `json:"risk_limit_base"`
	RiskLimitStep       types.Number `json:"risk_limit_step"`
	RiskLimitMax        types.Number `json:"risk_limit_max"`
	MakerFeeRate        types.Number `json:"maker_fee_rate"`
	TakerFeeRate        types.Number `json:"taker_fee_rate"`
	RefDiscountRate     types.Number `json:"ref_discount_rate"`
	RefRebateRate       types.Number `json:"ref_rebate_rate"`
	OrderPriceDeviate   types.Number `json:"order_price_deviate"`
	OrderSizeMin        int64        `json:"order_size_min"`
	OrderSizeMax        int64        `json:"order_size_max"`
	OrdersLimit         int64        `json:"orders_limit"`
	OrderbookID         int64        `json:"orderbook_id"`
	TradeID             int64        `json:"trade_id"`
	TradeSize           int64        `json:"trade_size"`
	PositionSize        int64        `json:"position_size"`
	ExpireTime          types.Time   `json:"expire_time"`
	ConfigChangeTime    types.Time   `json:"config_change_time"`
	InDelisting         bool         `json:"in_delisting"`
}

// OptionUnderlying represents option underlying and it's index price.
type OptionUnderlying struct {
	Name       string       `json:"name"`
	IndexPrice types.Number `json:"index_price"`
	IndexTime  types.Time   `json:"index_time"`
}

// OptionContract represents an option contract detail.
type OptionContract struct {
	Name              string       `json:"name"`
	Tag               string       `json:"tag"`
	IsCall            bool         `json:"is_call"`
	StrikePrice       types.Number `json:"strike_price"`
	LastPrice         types.Number `json:"last_price"`
	MarkPrice         types.Number `json:"mark_price"`
	OrderbookID       int64        `json:"orderbook_id"`
	TradeID           int64        `json:"trade_id"`
	TradeSize         int64        `json:"trade_size"`
	PositionSize      int64        `json:"position_size"`
	Underlying        string       `json:"underlying"`
	UnderlyingPrice   types.Number `json:"underlying_price"`
	Multiplier        string       `json:"multiplier"`
	OrderPriceRound   string       `json:"order_price_round"`
	MarkPriceRound    string       `json:"mark_price_round"`
	MakerFeeRate      string       `json:"maker_fee_rate"`
	TakerFeeRate      string       `json:"taker_fee_rate"`
	PriceLimitFeeRate string       `json:"price_limit_fee_rate"`
	RefDiscountRate   string       `json:"ref_discount_rate"`
	RefRebateRate     string       `json:"ref_rebate_rate"`
	OrderPriceDeviate string       `json:"order_price_deviate"`
	OrderSizeMin      int64        `json:"order_size_min"`
	OrderSizeMax      int64        `json:"order_size_max"`
	OrdersLimit       int64        `json:"orders_limit"`
	CreateTime        types.Time   `json:"create_time"`
	ExpirationTime    types.Time   `json:"expiration_time"`
}

// OptionSettlement list settlement history
type OptionSettlement struct {
	Timestamp   types.Time   `json:"time"`
	Profit      types.Number `json:"profit"`
	Fee         types.Number `json:"fee"`
	SettlePrice types.Number `json:"settle_price"`
	Contract    string       `json:"contract"`
	StrikePrice types.Number `json:"strike_price"`
}

// SwapCurrencies represents Flash Swap supported currencies
type SwapCurrencies struct {
	Currency  string       `json:"currency"`
	MinAmount types.Number `json:"min_amount"`
	MaxAmount types.Number `json:"max_amount"`
	Swappable []string     `json:"swappable"`
}

// MyOptionSettlement represents option private settlement
type MyOptionSettlement struct {
	Size         float64      `json:"size"`
	SettleProfit types.Number `json:"settle_profit"`
	Contract     string       `json:"contract"`
	StrikePrice  types.Number `json:"strike_price"`
	Time         types.Time   `json:"time"`
	SettlePrice  types.Number `json:"settle_price"`
	Underlying   string       `json:"underlying"`
	RealisedPnl  string       `json:"realised_pnl"`
	Fee          types.Number `json:"fee"`
}

// OptionsTicker represents  tickers of options contracts
type OptionsTicker struct {
	Name                  currency.Pair `json:"name"`
	LastPrice             types.Number  `json:"last_price"`
	MarkPrice             types.Number  `json:"mark_price"`
	PositionSize          float64       `json:"position_size"`
	Ask1Size              float64       `json:"ask1_size"`
	Ask1Price             types.Number  `json:"ask1_price"`
	Bid1Size              float64       `json:"bid1_size"`
	Bid1Price             types.Number  `json:"bid1_price"`
	Vega                  string        `json:"vega"`
	Theta                 string        `json:"theta"`
	Rho                   string        `json:"rho"`
	Gamma                 string        `json:"gamma"`
	Delta                 string        `json:"delta"`
	MarkImpliedVolatility types.Number  `json:"mark_iv"`
	BidImpliedVolatility  types.Number  `json:"bid_iv"`
	AskImpliedVolatility  types.Number  `json:"ask_iv"`
	Leverage              types.Number  `json:"leverage"`

	// Added fields for the websocket
	IndexPrice types.Number `json:"index_price"`
}

// OptionsUnderlyingTicker represents underlying ticker
type OptionsUnderlyingTicker struct {
	TradePut   float64      `json:"trade_put"`
	TradeCall  float64      `json:"trade_call"`
	IndexPrice types.Number `json:"index_price"`
}

// OptionAccount represents option account.
type OptionAccount struct {
	User          int64        `json:"user"`
	Currency      string       `json:"currency"`
	ShortEnabled  bool         `json:"short_enabled"`
	Total         types.Number `json:"total"`
	UnrealisedPnl string       `json:"unrealised_pnl"`
	InitMargin    string       `json:"init_margin"`
	MaintMargin   string       `json:"maint_margin"`
	OrderMargin   string       `json:"order_margin"`
	Available     types.Number `json:"available"`
	Point         string       `json:"point"`
}

// AccountBook represents account changing history item
type AccountBook struct {
	ChangeTime    types.Time   `json:"time"`
	AccountChange types.Number `json:"change"`
	Balance       types.Number `json:"balance"`
	CustomText    string       `json:"text"`
	ChangingType  string       `json:"type"`
}

// UsersPositionForUnderlying represents user's position for specified underlying.
type UsersPositionForUnderlying struct {
	User          int64        `json:"user"`
	Contract      string       `json:"contract"`
	Size          int64        `json:"size"`
	EntryPrice    types.Number `json:"entry_price"`
	RealisedPnl   types.Number `json:"realised_pnl"`
	MarkPrice     types.Number `json:"mark_price"`
	UnrealisedPnl types.Number `json:"unrealised_pnl"`
	PendingOrders int64        `json:"pending_orders"`
	CloseOrder    struct {
		ID    int64        `json:"id"`
		Price types.Number `json:"price"`
		IsLiq bool         `json:"is_liq"`
	} `json:"close_order"`
}

// ContractClosePosition represents user's liquidation history
type ContractClosePosition struct {
	PositionCloseTime types.Time   `json:"time"`
	Pnl               types.Number `json:"pnl"`
	SettleSize        string       `json:"settle_size"`
	Side              string       `json:"side"` // Position side, long or short
	FuturesContract   string       `json:"contract"`
	CloseOrderText    string       `json:"text"`
}

// OptionOrderParam represents option order request body
type OptionOrderParam struct {
	OrderSize   float64      `json:"size"`              // Order size. Specify positive number to make a bid, and negative number to ask
	Iceberg     float64      `json:"iceberg,omitempty"` // Display size for iceberg order. 0 for non-iceberg. Note that you will have to pay the taker fee for the hidden size
	Contract    string       `json:"contract"`
	Text        string       `json:"text,omitempty"`
	TimeInForce string       `json:"tif,omitempty"`
	Price       types.Number `json:"price,omitempty"`
	// Close Set as true to close the position, with size set to 0
	Close      bool `json:"close,omitempty"`
	ReduceOnly bool `json:"reduce_only,omitempty"`
}

// OptionOrderResponse represents option order response detail
type OptionOrderResponse struct {
	Status               string       `json:"status"`
	Size                 float64      `json:"size"`
	OptionOrderID        int64        `json:"id"`
	Iceberg              int64        `json:"iceberg"`
	IsOrderLiquidation   bool         `json:"is_liq"`
	IsOrderPositionClose bool         `json:"is_close"`
	Contract             string       `json:"contract"`
	Text                 string       `json:"text"`
	FillPrice            types.Number `json:"fill_price"`
	FinishAs             string       `json:"finish_as"` //  finish_as 	filled, cancelled, liquidated, ioc, auto_deleveraged, reduce_only, position_closed, reduce_out
	Left                 float64      `json:"left"`
	TimeInForce          string       `json:"tif"`
	IsReduceOnly         bool         `json:"is_reduce_only"`
	CreateTime           types.Time   `json:"create_time"`
	FinishTime           types.Time   `json:"finish_time"`
	Price                types.Number `json:"price"`

	TakerFee        types.Number `json:"tkrf"`
	MakerFee        types.Number `json:"mkrf"`
	ReferenceUserID string       `json:"refu"`
}

// OptionTradingHistory list personal trading history
type OptionTradingHistory struct {
	ID              int64        `json:"id"`
	UnderlyingPrice types.Number `json:"underlying_price"`
	Size            float64      `json:"size"`
	Contract        string       `json:"contract"`
	TradeRole       string       `json:"role"`
	CreateTime      types.Time   `json:"create_time"`
	OrderID         int64        `json:"order_id"`
	Price           types.Number `json:"price"`
}

// WithdrawalResponse represents withdrawal response
type WithdrawalResponse struct {
	ID                string       `json:"id"`
	Timestamp         types.Time   `json:"timestamp"`
	Currency          string       `json:"currency"`
	WithdrawalAddress string       `json:"address"`
	TransactionID     string       `json:"txid"`
	Amount            types.Number `json:"amount"`
	Memo              string       `json:"memo"`
	Status            string       `json:"status"`
	Chain             string       `json:"chain"`
	Fee               types.Number `json:"fee"`
}

// WithdrawalRequestParam represents currency withdrawal request param.
type WithdrawalRequestParam struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount"`
	Chain    string        `json:"chain,omitempty"`

	// Optional parameters
	Address string `json:"address,omitempty"`
	Memo    string `json:"memo,omitempty"`
}

// CurrencyDepositAddressInfo represents a crypto deposit address
type CurrencyDepositAddressInfo struct {
	Currency            string                  `json:"currency"`
	Address             string                  `json:"address"`
	MultichainAddresses []MultiChainAddressItem `json:"multichain_addresses"`
}

// MultiChainAddressItem represents a multi-chain address item
type MultiChainAddressItem struct {
	Chain        string `json:"chain"`
	Address      string `json:"address"`
	PaymentID    string `json:"payment_id"`
	PaymentName  string `json:"payment_name"`
	ObtainFailed int64  `json:"obtain_failed"`
}

// DepositRecord represents deposit record item
type DepositRecord struct {
	ID            string       `json:"id"`
	Timestamp     types.Time   `json:"timestamp"`
	Currency      string       `json:"currency"`
	Address       string       `json:"address"`
	TransactionID string       `json:"txid"`
	Amount        types.Number `json:"amount"`
	Memo          string       `json:"memo"`
	Status        string       `json:"status"`
	Chain         string       `json:"chain"`
	Fee           types.Number `json:"fee"`
}

// TransferCurrencyParam represents currency transfer.
type TransferCurrencyParam struct {
	Currency     currency.Code `json:"currency"`
	From         string        `json:"from"`
	To           string        `json:"to"`
	Amount       types.Number  `json:"amount"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Settle       string        `json:"settle"`
}

// TransactionIDResponse represents transaction ID
type TransactionIDResponse struct {
	TransactionID int64 `json:"tx_id"`
}

// SubAccountTransferParam represents currency subaccount transfer request param
type SubAccountTransferParam struct {
	Currency       currency.Code `json:"currency"`
	SubAccount     string        `json:"sub_account"`
	Direction      string        `json:"direction"`
	Amount         types.Number  `json:"amount"`
	SubAccountType string        `json:"sub_account_type"`
}

// SubAccountTransferResponse represents transfer records between main and sub accounts
type SubAccountTransferResponse struct {
	MainAccountUserID string       `json:"uid"`
	Timestamp         types.Time   `json:"timest"`
	Source            string       `json:"source"`
	Currency          string       `json:"currency"`
	SubAccount        string       `json:"sub_account"`
	TransferDirection string       `json:"direction"`
	Amount            types.Number `json:"amount"`
	SubAccountType    string       `json:"sub_account_type"`
}

// WithdrawalStatus represents currency withdrawal status
type WithdrawalStatus struct {
	Currency               string            `json:"currency"`
	CurrencyName           string            `json:"name"`
	CurrencyNameChinese    string            `json:"name_cn"`
	Deposit                types.Number      `json:"deposit"`
	WithdrawPercent        string            `json:"withdraw_percent"`
	FixedWithdrawalFee     types.Number      `json:"withdraw_fix"`
	WithdrawDayLimit       types.Number      `json:"withdraw_day_limit"`
	WithdrawDayLimitRemain types.Number      `json:"withdraw_day_limit_remain"`
	WithdrawAmountMini     types.Number      `json:"withdraw_amount_mini"`
	WithdrawEachTimeLimit  types.Number      `json:"withdraw_eachtime_limit"`
	WithdrawFixOnChains    map[string]string `json:"withdraw_fix_on_chains"`
	AdditionalProperties   string            `json:"additionalProperties"`
}

// FuturesSubAccountBalance represents sub account balance for specific sub account and several currencies
type FuturesSubAccountBalance struct {
	UserID    int64 `json:"uid,string"`
	Available struct {
		Total                     types.Number `json:"total"`
		UnrealisedProfitAndLoss   string       `json:"unrealised_pnl"`
		PositionMargin            string       `json:"position_margin"`
		OrderMargin               string       `json:"order_margin"`
		TotalAvailable            types.Number `json:"available"`
		PointAmount               float64      `json:"point"`
		SettleCurrency            string       `json:"currency"`
		InDualMode                bool         `json:"in_dual_mode"`
		EnableCredit              bool         `json:"enable_credit"`
		PositionInitialMargin     string       `json:"position_initial_margin"` // applicable to the portfolio margin account model
		MaintenanceMarginPosition string       `json:"maintenance_margin"`
		PerpetualContractBonus    string       `json:"bonus"`
		StatisticalData           struct {
			TotalDNW         types.Number `json:"dnw"` // total amount of deposit and withdraw
			ProfitAndLoss    types.Number `json:"pnl"` // total amount of trading profit and loss
			TotalAmountOfFee types.Number `json:"fee"`
			ReferrerRebates  types.Number `json:"refr"` // total amount of referrer rebates
			Fund             types.Number `json:"fund"` // total amount of funding costs
			PointDNW         types.Number `json:"point_dnw"`
			PoointFee        types.Number `json:"point_fee"`
			PointRefr        types.Number `json:"point_refr"`
			BonusDNW         types.Number `json:"bonus_dnw"`
			BonusOffset      types.Number `json:"bonus_offset"`
		} `json:"history"`
	} `json:"available"`
}

// SubAccountMarginBalance represents sub account margin balance for specific sub account and several currencies
type SubAccountMarginBalance struct {
	UID       string `json:"uid"`
	Available []struct {
		CurrencyPair string                `json:"currency_pair"`
		Locked       bool                  `json:"locked"`
		Risk         string                `json:"risk"`
		Base         MarginCurrencyBalance `json:"base"`
		Quote        MarginCurrencyBalance `json:"quote"`
	} `json:"available"`
}

// MarginCurrencyBalance represents a currency balance detail information.
type MarginCurrencyBalance struct {
	Currency       string       `json:"currency"`
	Available      types.Number `json:"available"`
	Locked         types.Number `json:"locked"`
	BorrowedAmount types.Number `json:"borrowed"`
	UnpairInterest types.Number `json:"interest"`
}

// MarginAccountItem margin account item
type MarginAccountItem struct {
	Locked       bool                      `json:"locked"`
	CurrencyPair string                    `json:"currency_pair"`
	Risk         string                    `json:"risk"`
	Base         AccountBalanceInformation `json:"base"`
	Quote        AccountBalanceInformation `json:"quote"`
}

// AccountBalanceInformation represents currency account balance information.
type AccountBalanceInformation struct {
	Available    types.Number `json:"available"`
	Borrowed     types.Number `json:"borrowed"`
	Interest     types.Number `json:"interest"`
	Currency     string       `json:"currency"`
	LockedAmount types.Number `json:"locked"`
}

// MarginAccountBalanceChangeInfo represents margin account balance
type MarginAccountBalanceChangeInfo struct {
	ID            string     `json:"id"`
	Time          types.Time `json:"time_ms"`
	Currency      string     `json:"currency"`
	CurrencyPair  string     `json:"currency_pair"`
	AmountChanged string     `json:"change"`
	Balance       string     `json:"balance"`
}

// MarginFundingAccountItem represents funding account list item.
type MarginFundingAccountItem struct {
	Currency     string       `json:"currency"`
	Available    types.Number `json:"available"`
	LockedAmount types.Number `json:"locked"`
	Lent         string       `json:"lent"`       // Outstanding loan amount yet to be repaid
	TotalLent    string       `json:"total_lent"` // Amount used for lending. total_lent = lent + locked
}

// MarginLoanRequestParam represents margin lend or borrow request param
type MarginLoanRequestParam struct {
	Side         string        `json:"side"`
	Currency     currency.Code `json:"currency"`
	Rate         types.Number  `json:"rate,omitempty"`
	Amount       types.Number  `json:"amount,omitempty"`
	Days         int64         `json:"days,omitempty"`
	AutoRenew    bool          `json:"auto_renew,omitempty"`
	CurrencyPair currency.Pair `json:"currency_pair,omitzero"`
	FeeRate      types.Number  `json:"fee_rate,omitempty"`
	OrigID       string        `json:"orig_id,omitempty"`
	Text         string        `json:"text,omitempty"`
}

// MarginLoanResponse represents lending or borrow response.
type MarginLoanResponse struct {
	ID             string       `json:"id"`
	OrigID         string       `json:"orig_id,omitempty"`
	Side           string       `json:"side"`
	Currency       string       `json:"currency"`
	Amount         types.Number `json:"amount"`
	Rate           types.Number `json:"rate"`
	Days           int64        `json:"days,omitempty"`
	AutoRenew      bool         `json:"auto_renew,omitempty"`
	CurrencyPair   string       `json:"currency_pair,omitempty"`
	FeeRate        types.Number `json:"fee_rate"`
	Text           string       `json:"text,omitempty"`
	CreateTime     types.Time   `json:"create_time"`
	ExpireTime     types.Time   `json:"expire_time"`
	Status         string       `json:"status"`
	Left           types.Number `json:"left"`
	Repaid         types.Number `json:"repaid"`
	PaidInterest   types.Number `json:"paid_interest"`
	UnpaidInterest types.Number `json:"unpaid_interest"`
}

// SubAccountCrossMarginInfo represents subaccount's cross_margin account info
type SubAccountCrossMarginInfo struct {
	UID       string `json:"uid"`
	Available struct {
		UserID                     int64                         `json:"user_id"`
		Locked                     bool                          `json:"locked"`
		Total                      types.Number                  `json:"total"`
		Borrowed                   types.Number                  `json:"borrowed"`
		Interest                   types.Number                  `json:"interest"` // Total unpaid interests in USDT, i.e., the sum of all currencies' interest*price*discount
		BorrowedNet                string                        `json:"borrowed_net"`
		TotalNetAssets             types.Number                  `json:"net"`
		Leverage                   types.Number                  `json:"leverage"`
		Risk                       string                        `json:"risk"`
		TotalInitialMargin         types.Number                  `json:"total_initial_margin"`
		TotalMarginBalance         types.Number                  `json:"total_margin_balance"`
		TotalMaintenanceMargin     types.Number                  `json:"total_maintenance_margin"`
		TotalInitialMarginRate     types.Number                  `json:"total_initial_margin_rate"`
		TotalMaintenanceMarginRate types.Number                  `json:"total_maintenance_margin_rate"`
		TotalAvailableMargin       types.Number                  `json:"total_available_margin"`
		CurrencyBalances           map[string]CrossMarginBalance `json:"balances"`
	} `json:"available"`
}

// CrossMarginBalance represents cross-margin currency balance detail
type CrossMarginBalance struct {
	Available           types.Number `json:"available"`
	Freeze              types.Number `json:"freeze"`
	Borrowed            types.Number `json:"borrowed"`
	Interest            types.Number `json:"interest"`
	Total               string       `json:"total"`
	BorrowedNet         string       `json:"borrowed_net"`
	TotalNetAssetInUSDT string       `json:"net"`
	PositionLeverage    string       `json:"leverage"`
	Risk                string       `json:"risk"` // Risk percentage; Liquidation is triggered when this falls below required margin. Calculation: total / (borrowed+interest)
}

// WalletSavedAddress represents currency saved address
type WalletSavedAddress struct {
	Currency string `json:"currency"`
	Chain    string `json:"chain"`
	Address  string `json:"address"`
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Verified string `json:"verified"` // Whether to pass the verification 0-unverified, 1-verified
}

// PersonalTradingFee represents personal trading fee for specific currency pair
type PersonalTradingFee struct {
	UserID           int64        `json:"user_id"`
	TakerFee         types.Number `json:"taker_fee"`
	MakerFee         types.Number `json:"maker_fee"`
	GtDiscount       bool         `json:"gt_discount"`
	GtTakerFee       types.Number `json:"gt_taker_fee"`
	GtMakerFee       types.Number `json:"gt_maker_fee"`
	LoanFee          types.Number `json:"loan_fee"`
	PointType        string       `json:"point_type"`
	FuturesTakerFee  types.Number `json:"futures_taker_fee"`
	FuturesMakerFee  types.Number `json:"futures_maker_fee"`
	DeliveryTakerFee types.Number `json:"delivery_taker_fee"`
	DeliveryMakerFee types.Number `json:"delivery_maker_fee"`
	DebitFee         float64      `json:"debit_fee"`
}

// UsersAllAccountBalance represents user all account balances.
type UsersAllAccountBalance struct {
	Details map[string]CurrencyBalanceAmount `json:"details"`
	Total   CurrencyBalanceAmount            `json:"total"`
}

// CurrencyBalanceAmount represents currency and its amount.
type CurrencyBalanceAmount struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

// SpotTradingFeeRate user trading fee rates
type SpotTradingFeeRate struct {
	UserID          int64        `json:"user_id"`
	TakerFee        types.Number `json:"taker_fee"`
	MakerFee        types.Number `json:"maker_fee"`
	GtDiscount      bool         `json:"gt_discount"`
	GtTakerFee      types.Number `json:"gt_taker_fee"`
	GtMakerFee      types.Number `json:"gt_maker_fee"`
	FuturesTakerFee types.Number `json:"futures_taker_fee"`
	FuturesMakerFee types.Number `json:"futures_maker_fee"`
	LoanFee         types.Number `json:"loan_fee"`
	PointType       string       `json:"point_type"`
}

// SpotAccount represents spot account
type SpotAccount struct {
	Currency  string       `json:"currency"`
	Available types.Number `json:"available"`
	Locked    types.Number `json:"locked"`
}

// CreateOrderRequest represents a single order creation param.
type CreateOrderRequest struct {
	Text                      string        `json:"text,omitempty"`
	CurrencyPair              currency.Pair `json:"currency_pair,omitzero"`
	Type                      string        `json:"type,omitempty"`
	Account                   string        `json:"account,omitempty"`
	Side                      string        `json:"side,omitempty"`
	Iceberg                   string        `json:"iceberg,omitempty"`
	Amount                    types.Number  `json:"amount,omitempty"`
	Price                     types.Number  `json:"price,omitempty"`
	TimeInForce               string        `json:"time_in_force,omitempty"`
	AutoBorrow                bool          `json:"auto_borrow,omitempty"`
	AutoRepay                 bool          `json:"auto_repay,omitempty"`
	SelfTradePreventionAction string        `json:"stp_act,omitempty"`
	// ActionMode specifies the processing mode for an order request, determining the fields returned in the response.
	// Valid only during the request and omitted from the response. Options:
	// - ACK: Asynchronous mode, returns only key order fields
	// - RESULT: Excludes clearing information
	// - FULL: Full mode (default)
	ActionMode string `json:"action_mode,omitempty"`
}

// SpotOrder represents create order response.
type SpotOrder struct {
	OrderID            string       `json:"id,omitempty"`
	Text               string       `json:"text,omitempty"`
	Succeeded          bool         `json:"succeeded"`
	ErrorLabel         string       `json:"label,omitempty"`
	Message            string       `json:"message,omitempty"`
	CreateTime         types.Time   `json:"create_time_ms,omitzero"`
	UpdateTime         types.Time   `json:"update_time_ms,omitzero"`
	CurrencyPair       string       `json:"currency_pair,omitempty"`
	Status             string       `json:"status,omitempty"`
	Type               string       `json:"type,omitempty"`
	Account            string       `json:"account,omitempty"`
	Side               string       `json:"side,omitempty"`
	Amount             types.Number `json:"amount,omitempty"`
	Price              types.Number `json:"price,omitempty"`
	TimeInForce        string       `json:"time_in_force,omitempty"`
	Iceberg            string       `json:"iceberg,omitempty"`
	AutoRepay          bool         `json:"auto_repay"`
	AutoBorrow         bool         `json:"auto_borrow"`
	Left               types.Number `json:"left"`
	AverageFillPrice   types.Number `json:"avg_deal_price"`
	FeeDeducted        types.Number `json:"fee"`
	FeeCurrency        string       `json:"fee_currency"`
	FillPrice          types.Number `json:"fill_price"`   // Total filled in quote currency. Deprecated in favor of filled_total
	FilledTotal        types.Number `json:"filled_total"` // Total filled in quote currency
	PointFee           types.Number `json:"point_fee"`
	GtFee              string       `json:"gt_fee,omitempty"`
	GtDiscount         bool         `json:"gt_discount"`
	GtMakerFee         types.Number `json:"gt_maker_fee"`
	GtTakerFee         types.Number `json:"gt_taker_fee"`
	RebatedFee         types.Number `json:"rebated_fee"`
	RebatedFeeCurrency string       `json:"rebated_fee_currency"`
}

// SpotOrdersDetail represents list of orders for specific currency pair
type SpotOrdersDetail struct {
	CurrencyPair string      `json:"currency_pair"`
	Total        float64     `json:"total"`
	Orders       []SpotOrder `json:"orders"`
}

// ClosePositionRequestParam represents close position when cross currency is disable.
type ClosePositionRequestParam struct {
	Text         string        `json:"text"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Amount       types.Number  `json:"amount"`
	Price        types.Number  `json:"price"`
}

// CancelOrderByIDParam represents cancel order by id request param.
type CancelOrderByIDParam struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	ID           string        `json:"id"`
}

// CancelOrderByIDResponse represents calcel order response when deleted by id.
type CancelOrderByIDResponse struct {
	CurrencyPair string `json:"currency_pair"`
	OrderID      string `json:"id"`
	Succeeded    bool   `json:"succeeded"`
	Label        string `json:"label"`
	Message      string `json:"message"`
	Account      string `json:"account"`
}

// SpotPersonalTradeHistory represents personal trading history.
type SpotPersonalTradeHistory struct {
	TradeID      string       `json:"id"`
	CreateTime   types.Time   `json:"create_time_ms"`
	CurrencyPair string       `json:"currency_pair"`
	OrderID      string       `json:"order_id"`
	Side         string       `json:"side"`
	Role         string       `json:"role"`
	Amount       types.Number `json:"amount"`
	Price        types.Number `json:"price"`
	Fee          types.Number `json:"fee"`
	FeeCurrency  string       `json:"fee_currency"`
	PointFee     string       `json:"point_fee"`
	GtFee        string       `json:"gt_fee"`
}

// CountdownCancelOrderParam represents countdown cancel order params
type CountdownCancelOrderParam struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	Timeout      int64         `json:"timeout"` // timeout: Countdown time, in seconds At least 5 seconds, 0 means cancel the countdown
}

// TriggerTimeResponse represents trigger types.Timeas a response for countdown candle order response
type TriggerTimeResponse struct {
	TriggerTime types.Time `json:"trigger_time"`
}

// PriceTriggeredOrderParam represents price triggered order request.
type PriceTriggeredOrderParam struct {
	Trigger TriggerPriceInfo `json:"trigger"`
	Put     PutOrderData     `json:"put"`
	Market  currency.Pair    `json:"market"`
}

// TriggerPriceInfo represents a trigger price and related information for Price triggered order
type TriggerPriceInfo struct {
	Price      types.Number `json:"price"`
	Rule       string       `json:"rule"`
	Expiration int64        `json:"expiration"`
}

// PutOrderData represents order detail for price triggered order request
type PutOrderData struct {
	Type        string       `json:"type"`
	Side        string       `json:"side"`
	Price       types.Number `json:"price"`
	Amount      types.Number `json:"amount"`
	Account     string       `json:"account"`
	TimeInForce string       `json:"time_in_force,omitempty"`
}

// OrderID represents order creation ID response.
type OrderID struct {
	ID int64 `json:"id"`
}

// SpotPriceTriggeredOrder represents spot price triggered order response data.
type SpotPriceTriggeredOrder struct {
	Trigger      TriggerPriceInfo `json:"trigger"`
	Put          PutOrderData     `json:"put"`
	AutoOrderID  int64            `json:"id"`
	UserID       int64            `json:"user"`
	CreationTime types.Time       `json:"ctime"`
	FireTime     types.Time       `json:"ftime"`
	FiredOrderID int64            `json:"fired_order_id"`
	Status       string           `json:"status"`
	Reason       string           `json:"reason"`
	Market       string           `json:"market"`
}

// ModifyLoanRequestParam represents request parameters for modify loan request
type ModifyLoanRequestParam struct {
	Currency     currency.Code `json:"currency"`
	Side         string        `json:"side"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	AutoRenew    bool          `json:"auto_renew"`
	LoanID       string        `json:"loan_id"`
}

// RepayLoanRequestParam represents loan repay request parameters
type RepayLoanRequestParam struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	Currency     currency.Code `json:"currency"`
	Mode         string        `json:"mode"`
	Amount       types.Number  `json:"amount"`
}

// LoanRepaymentRecord represents loan repayment history record item.
type LoanRepaymentRecord struct {
	ID         string     `json:"id"`
	CreateTime types.Time `json:"create_time"`
	Principal  string     `json:"principal"`
	Interest   string     `json:"interest"`
}

// LoanRecord represents loan repayment specific record
type LoanRecord struct {
	ID             string       `json:"id"`
	LoanID         string       `json:"loan_id"`
	CreateTime     types.Time   `json:"create_time"`
	ExpireTime     types.Time   `json:"expire_time"`
	Status         string       `json:"status"`
	BorrowUserID   string       `json:"borrow_user_id"`
	Currency       string       `json:"currency"`
	Rate           types.Number `json:"rate"`
	Amount         types.Number `json:"amount"`
	Days           int64        `json:"days"`
	AutoRenew      bool         `json:"auto_renew"`
	Repaid         types.Number `json:"repaid"`
	PaidInterest   types.Number `json:"paid_interest"`
	UnpaidInterest types.Number `json:"unpaid_interest"`
}

// OnOffStatus represents on or off status response status
type OnOffStatus struct {
	Status string `json:"status"`
}

// MaxTransferAndLoanAmount represents the maximum amount to transfer, borrow, or lend for specific currency and currency pair
type MaxTransferAndLoanAmount struct {
	Currency     string       `json:"currency"`
	CurrencyPair string       `json:"currency_pair"`
	Amount       types.Number `json:"amount"`
}

// CrossMarginCurrencies represents a currency supported by cross margin
type CrossMarginCurrencies struct {
	Name                 string       `json:"name"`
	Rate                 types.Number `json:"rate"`
	CurrencyPrecision    types.Number `json:"prec"`
	Discount             string       `json:"discount"`
	MinBorrowAmount      types.Number `json:"min_borrow_amount"`
	UserMaxBorrowAmount  types.Number `json:"user_max_borrow_amount"`
	TotalMaxBorrowAmount types.Number `json:"total_max_borrow_amount"`
	Price                types.Number `json:"price"` // Price change between this currency and USDT
	Status               int64        `json:"status"`
}

// CrossMarginCurrencyBalance represents the currency detailed balance information for cross margin
type CrossMarginCurrencyBalance struct {
	Available types.Number `json:"available"`
	Freeze    types.Number `json:"freeze"`
	Borrowed  types.Number `json:"borrowed"`
	Interest  types.Number `json:"interest"`
}

// CrossMarginAccount represents the account detail for cross margin account balance
type CrossMarginAccount struct {
	UserID                      int64                                 `json:"user_id"`
	Locked                      bool                                  `json:"locked"`
	Balances                    map[string]CrossMarginCurrencyBalance `json:"balances"`
	Total                       types.Number                          `json:"total"`
	Borrowed                    types.Number                          `json:"borrowed"`
	Interest                    types.Number                          `json:"interest"`
	Risk                        types.Number                          `json:"risk"`
	TotalInitialMargin          string                                `json:"total_initial_margin"`
	TotalMarginBalance          types.Number                          `json:"total_margin_balance"`
	TotalMaintenanceMargin      types.Number                          `json:"total_maintenance_margin"`
	TotalInitialMarginRate      types.Number                          `json:"total_initial_margin_rate"`
	TotalMaintenanceMarginRate  types.Number                          `json:"total_maintenance_margin_rate"`
	TotalAvailableMargin        types.Number                          `json:"total_available_margin"`
	TotalPortfolioMarginAccount types.Number                          `json:"portfolio_margin_total"`
}

// CrossMarginAccountHistoryItem represents a cross margin account change history item
type CrossMarginAccountHistoryItem struct {
	ID       string       `json:"id"`
	Time     types.Time   `json:"time"`
	Currency string       `json:"currency"` // Currency changed
	Change   string       `json:"change"`
	Balance  types.Number `json:"balance"`
	Type     string       `json:"type"`
}

// CrossMarginBorrowLoanParams represents a cross margin borrow loan parameters
type CrossMarginBorrowLoanParams struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount"`
	Text     string        `json:"text"`
}

// CrossMarginLoanResponse represents a cross margin borrow loan response
type CrossMarginLoanResponse struct {
	ID             string       `json:"id"`
	CreateTime     types.Time   `json:"create_time"`
	UpdateTime     types.Time   `json:"update_time"`
	Currency       string       `json:"currency"`
	Amount         types.Number `json:"amount"`
	Text           string       `json:"text"`
	Status         int64        `json:"status"`
	Repaid         string       `json:"repaid"`
	RepaidInterest types.Number `json:"repaid_interest"`
	UnpaidInterest types.Number `json:"unpaid_interest"`
}

// CurrencyAndAmount represents request parameters for repayment
type CurrencyAndAmount struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount"`
}

// RepaymentHistoryItem represents an item in a repayment history.
type RepaymentHistoryItem struct {
	ID         string       `json:"id"`
	CreateTime types.Time   `json:"create_time"`
	LoanID     string       `json:"loan_id"`
	Currency   string       `json:"currency"`
	Principal  types.Number `json:"principal"`
	Interest   types.Number `json:"interest"`
}

// FlashSwapOrderParams represents create flash swap order request parameters.
type FlashSwapOrderParams struct {
	PreviewID    string        `json:"preview_id"`
	SellCurrency currency.Code `json:"sell_currency"`
	SellAmount   types.Number  `json:"sell_amount,omitempty"`
	BuyCurrency  currency.Code `json:"buy_currency"`
	BuyAmount    types.Number  `json:"buy_amount,omitempty"`
}

// FlashSwapOrderResponse represents create flash swap order response
type FlashSwapOrderResponse struct {
	ID           int64        `json:"id"`
	CreateTime   types.Time   `json:"create_time"`
	UpdateTime   types.Time   `json:"update_time"`
	UserID       int64        `json:"user_id"`
	SellCurrency string       `json:"sell_currency"`
	SellAmount   types.Number `json:"sell_amount"`
	BuyCurrency  string       `json:"buy_currency"`
	BuyAmount    types.Number `json:"buy_amount"`
	Price        types.Number `json:"price"`
	Status       int64        `json:"status"`
}

// InitFlashSwapOrderPreviewResponse represents the order preview for flash order
type InitFlashSwapOrderPreviewResponse struct {
	PreviewID    string       `json:"preview_id"`
	SellCurrency string       `json:"sell_currency"`
	SellAmount   types.Number `json:"sell_amount"`
	BuyCurrency  string       `json:"buy_currency"`
	BuyAmount    types.Number `json:"buy_amount"`
	Price        types.Number `json:"price"`
}

// FuturesAccount represents futures account detail
type FuturesAccount struct {
	User                   int64        `json:"user"`
	Currency               string       `json:"currency"`
	Total                  types.Number `json:"total"` // total = position_margin + order_margin + available
	UnrealisedPnl          types.Number `json:"unrealised_pnl"`
	PositionMargin         types.Number `json:"position_margin"`
	OrderMargin            types.Number `json:"order_margin"` // Order margin of unfinished orders
	Available              types.Number `json:"available"`    // The available balance for transferring or trading
	Point                  types.Number `json:"point"`
	Bonus                  string       `json:"bonus"`
	EnabledCredit          bool         `json:"enable_credit"`
	InDualMode             bool         `json:"in_dual_mode"` // Whether dual mode is enabled
	UpdateTime             types.Time   `json:"update_time"`
	UpdateID               int64        `json:"update_id"`
	PositionInitialMargine types.Number `json:"position_initial_margin"` // applicable to the portfolio margin account model
	MaintenanceMargin      types.Number `json:"maintenance_margin"`
	MarginMode             int64        `json:"margin_mode"` // Margin mode: 1-cross margin, 2-isolated margin, 3-portfolio margin
	EnabledEvolvedClassic  bool         `json:"enable_evolved_classic"`
	CrossInitialMargin     types.Number `json:"cross_initial_margin"`
	CrossUnrealisedPnl     types.Number `json:"cross_unrealised_pnl"`
	IsolatedPositionMargin types.Number `json:"isolated_position_margin"`
	History                struct {
		DepositAndWithdrawal string       `json:"dnw"`  // total amount of deposit and withdraw
		ProfitAndLoss        types.Number `json:"pnl"`  // total amount of trading profit and loss
		Fee                  types.Number `json:"fee"`  // total amount of fee
		Refr                 types.Number `json:"refr"` // total amount of referrer rebates
		Fund                 types.Number `json:"fund"`
		PointDNW             types.Number `json:"point_dnw"` // total amount of point deposit and withdraw
		PointFee             types.Number `json:"point_fee"` // total amount of point fee
		PointRefr            types.Number `json:"point_refr"`
		BonusDNW             types.Number `json:"bonus_dnw"`    // total amount of perpetual contract bonus transfer
		BonusOffset          types.Number `json:"bonus_offset"` // total amount of perpetual contract bonus deduction
		CrossSettle          types.Number `json:"cross_settle"`
	} `json:"history"`
}

// AccountBookItem represents account book item
type AccountBookItem struct {
	Time    types.Time   `json:"time"`
	Change  types.Number `json:"change"`
	Balance types.Number `json:"balance"`
	Text    string       `json:"text"`
	Type    string       `json:"type"`
}

// Position represents futures position
type Position struct {
	User                       int64        `json:"user"`
	Contract                   string       `json:"contract"`
	Size                       float64      `json:"size"` // Denotes long or short position, positive for long, negative for short
	Leverage                   types.Number `json:"leverage"`
	RiskLimit                  types.Number `json:"risk_limit"`
	LeverageMax                types.Number `json:"leverage_max"`
	MaintenanceRate            types.Number `json:"maintenance_rate"`
	Value                      types.Number `json:"value"`
	Margin                     types.Number `json:"margin"`
	EntryPrice                 types.Number `json:"entry_price"`
	LiquidationPrice           types.Number `json:"liq_price"`
	MarkPrice                  types.Number `json:"mark_price"`
	InitialMargin              types.Number `json:"initial_margin"`
	MaintenanceMargin          types.Number `json:"maintenance_margin"`
	UnrealisedPNL              types.Number `json:"unrealised_pnl"`
	RealisedPnl                types.Number `json:"realised_pnl"`
	RealisedPNLPosition        types.Number `json:"pnl_pnl"`
	RealisedPNLFundingFee      types.Number `json:"pnl_fund"`
	RealisedPNLTransactionFees types.Number `json:"pnl_fee"`
	HistoryPNL                 types.Number `json:"history_pnl"`
	LastClosePNL               types.Number `json:"last_close_pnl"`
	RealisedPNLPoint           types.Number `json:"realised_point"`
	RealisedPNLHistoryPoint    types.Number `json:"history_point"`
	ADLRanking                 int64        `json:"adl_ranking"` // Ranking of auto deleveraging, a total of 1-5 grades, 1 is the highest, 5 is the lowest, and 6 is the special case when there is no position held or in liquidation
	PendingOrders              int64        `json:"pending_orders"`
	CloseOrder                 struct {
		ID    int64        `json:"id"`
		Price types.Number `json:"price"`
		IsLiq bool         `json:"is_liq"`
	} `json:"close_order"`
	Mode               string       `json:"mode"`
	CrossLeverageLimit types.Number `json:"cross_leverage_limit"`
	UpdateTime         types.Time   `json:"update_time"`
	OpenTime           types.Time   `json:"open_time"`
	UpdateID           int64        `json:"update_id"`
	TradeMaxSize       types.Number `json:"trade_max_size"`
}

// DualModeResponse represents  dual mode enable or disable
type DualModeResponse struct {
	User           int64        `json:"user"`
	Currency       string       `json:"currency"`
	Total          string       `json:"total"`
	UnrealisedPnl  types.Number `json:"unrealised_pnl"`
	PositionMargin types.Number `json:"position_margin"`
	OrderMargin    string       `json:"order_margin"`
	Available      string       `json:"available"`
	Point          string       `json:"point"`
	Bonus          string       `json:"bonus"`
	InDualMode     bool         `json:"in_dual_mode"`
	History        struct {
		DepositAndWithdrawal types.Number `json:"dnw"` // total amount of deposit and withdraw
		ProfitAndLoss        types.Number `json:"pnl"` // total amount of trading profit and loss
		Fee                  types.Number `json:"fee"`
		Refr                 types.Number `json:"refr"`
		Fund                 types.Number `json:"fund"`
		PointDnw             types.Number `json:"point_dnw"`
		PointFee             types.Number `json:"point_fee"`
		PointRefr            types.Number `json:"point_refr"`
		BonusDnw             types.Number `json:"bonus_dnw"`
		BonusOffset          types.Number `json:"bonus_offset"`
	} `json:"history"`
}

// ContractOrderCreateParams represents future order creation parameters
type ContractOrderCreateParams struct {
	Contract                  currency.Pair `json:"contract"`
	Size                      float64       `json:"size"`    // positive long, negative short
	Iceberg                   int64         `json:"iceberg"` // required; can be zero
	Price                     string        `json:"price"`   // NOTE: Market orders require string "0"
	TimeInForce               string        `json:"tif"`
	Text                      string        `json:"text,omitempty"`  // errors when empty; Either populated or omitted
	ClosePosition             bool          `json:"close,omitempty"` // Size needs to be zero if true
	ReduceOnly                bool          `json:"reduce_only,omitempty"`
	AutoSize                  string        `json:"auto_size,omitempty"` // either close_long or close_short, requires zero in size field
	Settle                    currency.Code `json:"-"`                   // Used in URL. REST Calls only.
	SelfTradePreventionAction string        `json:"stp_act,omitempty"`
}

// Order represents future order response
type Order struct {
	ID                    int64        `json:"id"`
	User                  int64        `json:"user"`
	Contract              string       `json:"contract"`
	CreateTime            types.Time   `json:"create_time"`
	Size                  float64      `json:"size"`
	Iceberg               int64        `json:"iceberg"`
	RemainingAmount       float64      `json:"left"` // Size left to be traded
	OrderPrice            types.Number `json:"price"`
	FillPrice             types.Number `json:"fill_price"` // Fill price of the order. total filled in quote currency.
	MakerFee              string       `json:"mkfr"`
	TakerFee              string       `json:"tkfr"`
	TimeInForce           string       `json:"tif"`
	ReferenceUserID       int64        `json:"refu"`
	IsReduceOnly          bool         `json:"is_reduce_only"`
	IsClose               bool         `json:"is_close"`
	IsOrderForLiquidation bool         `json:"is_liq"`
	Text                  string       `json:"text"`
	Status                string       `json:"status"`
	FinishTime            types.Time   `json:"finish_time"`
	FinishAs              string       `json:"finish_as"`
}

// AmendFuturesOrderParam represents amend futures order parameter
type AmendFuturesOrderParam struct {
	Size  types.Number `json:"size"`
	Price types.Number `json:"price"`
}

// PositionCloseHistoryResponse represents a close position history detail
type PositionCloseHistoryResponse struct {
	Time          types.Time   `json:"time"`
	ProfitAndLoss types.Number `json:"pnl"`
	Side          string       `json:"side"`
	Contract      string       `json:"contract"`
	Text          string       `json:"text"`
}

// LiquidationHistoryItem liquidation history item
type LiquidationHistoryItem struct {
	Time       types.Time   `json:"time"`
	Contract   string       `json:"contract"`
	Size       int64        `json:"size"`
	Leverage   types.Number `json:"leverage"`
	Margin     string       `json:"margin"`
	EntryPrice types.Number `json:"entry_price"`
	MarkPrice  types.Number `json:"mark_price"`
	OrderPrice types.Number `json:"order_price"`
	FillPrice  types.Number `json:"fill_price"`
	LiqPrice   types.Number `json:"liq_price"`
	OrderID    int64        `json:"order_id"`
	Left       int64        `json:"left"`
}

// CountdownParams represents query parameters for countdown cancel order
type CountdownParams struct {
	Timeout  int64         `json:"timeout"` // In Seconds
	Contract currency.Pair `json:"contract"`
}

// FuturesPriceTriggeredOrderParam represents a creates a price triggered order
type FuturesPriceTriggeredOrderParam struct {
	Initial   FuturesInitial `json:"initial"`
	Trigger   FuturesTrigger `json:"trigger"`
	OrderType string         `json:"order_type,omitempty"`
}

// FuturesInitial represents a price triggered order initial parameters
type FuturesInitial struct {
	Contract    currency.Pair `json:"contract"`
	Size        int64         `json:"size"`  // Order size. Positive size means to buy, while negative one means to sell. Set to 0 to close the position
	Price       types.Number  `json:"price"` // Order price. Set to 0 to use market price
	Close       bool          `json:"close,omitempty"`
	TimeInForce string        `json:"tif,omitempty"`
	Text        string        `json:"text,omitempty"`
	ReduceOnly  bool          `json:"reduce_only,omitempty"`
	AutoSize    string        `json:"auto_size,omitempty"`
}

// FuturesTrigger represents a price triggered order trigger parameter
type FuturesTrigger struct {
	StrategyType int64        `json:"strategy_type,omitempty"` // How the order will be triggered 0: by price, which means the order will be triggered if price condition is satisfied 1: by price gap, which means the order will be triggered if gap of recent two prices of specified price_type are satisfied. Only 0 is supported currently
	PriceType    int64        `json:"price_type,omitempty"`
	Price        types.Number `json:"price,omitempty"`
	Rule         int64        `json:"rule,omitempty"`
	Expiration   int64        `json:"expiration,omitempty"` // how long(in seconds) to wait for the condition to be triggered before cancelling the order
	OrderType    string       `json:"order_type,omitempty"`
}

// PriceTriggeredOrder represents a future triggered price order response
type PriceTriggeredOrder struct {
	Initial struct {
		Contract string       `json:"contract"`
		Size     float64      `json:"size"`
		Price    types.Number `json:"price"`
	} `json:"initial"`
	Trigger struct {
		StrategyType int64        `json:"strategy_type"`
		PriceType    int64        `json:"price_type"`
		Price        types.Number `json:"price"`
		Rule         int64        `json:"rule"`
		Expiration   int64        `json:"expiration"`
	} `json:"trigger"`
	ID         int64      `json:"id"`
	User       int64      `json:"user"`
	CreateTime types.Time `json:"create_time"`
	FinishTime types.Time `json:"finish_time"`
	TradeID    int64      `json:"trade_id"`
	Status     string     `json:"status"`
	FinishAs   string     `json:"finish_as"`
	Reason     string     `json:"reason"`
	OrderType  string     `json:"order_type"`
}

// SettlementHistoryItem represents a settlement history item
type SettlementHistoryItem struct {
	Time        types.Time   `json:"time"`
	Contract    string       `json:"contract"`
	Size        int64        `json:"size"`
	Leverage    string       `json:"leverage"`
	Margin      string       `json:"margin"`
	EntryPrice  types.Number `json:"entry_price"`
	SettlePrice types.Number `json:"settle_price"`
	Profit      types.Number `json:"profit"`
	Fee         types.Number `json:"fee"`
}

// SubAccountParams represents subaccount creation parameters
type SubAccountParams struct {
	LoginName string `json:"login_name"`
	Remark    string `json:"remark,omitempty"`
	Email     string `json:"email,omitempty"`    // The sub-account's password.
	Password  string `json:"password,omitempty"` // The sub-account's email address.
}

// SubAccount represents a subaccount response
type SubAccount struct {
	Remark          string     `json:"remark"`     // custom text
	LoginName       string     `json:"login_name"` // SubAccount login name
	Password        string     `json:"password"`   // The sub-account's password
	SubAccountEmail string     `json:"email"`      // The sub-account's email
	UserID          int64      `json:"user_id"`
	State           int64      `json:"state"`
	CreateTime      types.Time `json:"create_time"`
}

// **************************************************************************************************

// WsInput represents general structure for websocket requests
type WsInput struct {
	Time    int64        `json:"time,omitempty"`
	ID      int64        `json:"id,omitempty"`
	Channel string       `json:"channel,omitempty"`
	Event   string       `json:"event,omitempty"`
	Payload []string     `json:"payload,omitempty"`
	Auth    *WsAuthInput `json:"auth,omitempty"`
}

// WsAuthInput represents the authentication information
type WsAuthInput struct {
	Method string `json:"method,omitempty"`
	Key    string `json:"KEY,omitempty"`
	Sign   string `json:"SIGN,omitempty"`
}

// WsEventResponse represents websocket incoming subscription, unsubscription, and update response
type WsEventResponse struct {
	Time    types.Time `json:"time"`
	ID      int64      `json:"id"`
	Channel string     `json:"channel"`
	Event   string     `json:"event"`
	Result  *struct {
		Status string `json:"status"`
	} `json:"result"`
	Error *struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	}
}

// WSResponse represents generalized websocket push data from the server.
type WSResponse struct {
	ID        int64           `json:"id"`
	Time      time.Time       `json:"time"`
	Channel   string          `json:"channel"`
	Event     string          `json:"event"`
	Result    json.RawMessage `json:"result"`
	RequestID string          `json:"request_id"`
}

// WsTicker websocket ticker information.
type WsTicker struct {
	CurrencyPair     currency.Pair `json:"currency_pair"`
	Last             types.Number  `json:"last"`
	LowestAsk        types.Number  `json:"lowest_ask"`
	HighestBid       types.Number  `json:"highest_bid"`
	ChangePercentage types.Number  `json:"change_percentage"`
	BaseVolume       types.Number  `json:"base_volume"`
	QuoteVolume      types.Number  `json:"quote_volume"`
	High24H          types.Number  `json:"high_24h"`
	Low24H           types.Number  `json:"low_24h"`
}

// WsTrade represents a websocket push data response for a trade
type WsTrade struct {
	ID           int64         `json:"id"`
	CreateTime   types.Time    `json:"create_time_ms"`
	Side         string        `json:"side"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Amount       types.Number  `json:"amount"`
	Price        types.Number  `json:"price"`
}

// WsCandlesticks represents the candlestick data for spot, margin and cross margin trades pushed through the websocket channel.
type WsCandlesticks struct {
	Timestamp          types.Time   `json:"t"`
	TotalVolume        types.Number `json:"v"`
	ClosePrice         types.Number `json:"c"`
	HighestPrice       types.Number `json:"h"`
	LowestPrice        types.Number `json:"l"`
	OpenPrice          types.Number `json:"o"`
	NameOfSubscription string       `json:"n"`
}

// WsOrderbookTickerData represents the websocket orderbook best bid or best ask push data
type WsOrderbookTickerData struct {
	UpdateTime    types.Time    `json:"t"`
	UpdateOrderID int64         `json:"u"`
	Pair          currency.Pair `json:"s"`
	BestBidPrice  types.Number  `json:"b"`
	BestBidAmount types.Number  `json:"B"`
	BestAskPrice  types.Number  `json:"a"`
	BestAskAmount types.Number  `json:"A"`
}

// WsOrderbookUpdate represents websocket orderbook update push data
type WsOrderbookUpdate struct {
	UpdateTime    types.Time        `json:"t"`
	Pair          currency.Pair     `json:"s"`
	FirstUpdateID int64             `json:"U"` // First update order book id in this event since last update
	LastUpdateID  int64             `json:"u"`
	Bids          [][2]types.Number `json:"b"`
	Asks          [][2]types.Number `json:"a"`
}

// WsOrderbookSnapshot represents a websocket orderbook snapshot push data
type WsOrderbookSnapshot struct {
	UpdateTime   types.Time        `json:"t"`
	LastUpdateID int64             `json:"lastUpdateId"`
	CurrencyPair currency.Pair     `json:"s"`
	Bids         [][2]types.Number `json:"bids"`
	Asks         [][2]types.Number `json:"asks"`
}

// WsSpotOrder represents an order push data through the websocket channel.
type WsSpotOrder struct {
	ID                 string        `json:"id,omitempty"`
	User               int64         `json:"user"`
	Text               string        `json:"text,omitempty"`
	Succeeded          bool          `json:"succeeded,omitempty"`
	Label              string        `json:"label,omitempty"`
	Message            string        `json:"message,omitempty"`
	CurrencyPair       currency.Pair `json:"currency_pair,omitzero"`
	Type               string        `json:"type,omitempty"`
	Account            string        `json:"account,omitempty"`
	Side               string        `json:"side,omitempty"`
	Amount             types.Number  `json:"amount,omitempty"`
	Price              types.Number  `json:"price,omitempty"`
	TimeInForce        string        `json:"time_in_force,omitempty"`
	Iceberg            string        `json:"iceberg,omitempty"`
	Left               types.Number  `json:"left,omitempty"`
	FilledTotal        types.Number  `json:"filled_total,omitempty"`
	Fee                types.Number  `json:"fee,omitempty"`
	FeeCurrency        string        `json:"fee_currency,omitempty"`
	PointFee           string        `json:"point_fee,omitempty"`
	GtFee              string        `json:"gt_fee,omitempty"`
	GtDiscount         bool          `json:"gt_discount,omitempty"`
	RebatedFee         string        `json:"rebated_fee,omitempty"`
	RebatedFeeCurrency string        `json:"rebated_fee_currency,omitempty"`
	Event              string        `json:"event"`
	CreateTime         types.Time    `json:"create_time_ms,omitzero"`
	UpdateTime         types.Time    `json:"update_time_ms,omitzero"`
}

// WsUserPersonalTrade represents a user's personal trade pushed through the websocket connection.
type WsUserPersonalTrade struct {
	ID           int64         `json:"id"`
	UserID       int64         `json:"user_id"`
	OrderID      string        `json:"order_id"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	CreateTime   types.Time    `json:"create_time_ms"`
	Side         string        `json:"side"`
	Amount       types.Number  `json:"amount"`
	Role         string        `json:"role"`
	Price        types.Number  `json:"price"`
	Fee          types.Number  `json:"fee"`
	PointFee     types.Number  `json:"point_fee"`
	GtFee        string        `json:"gt_fee"`
	Text         string        `json:"text"`
}

// WsSpotBalance represents a spot balance.
type WsSpotBalance struct {
	Timestamp types.Time   `json:"timestamp_ms"`
	User      string       `json:"user"`
	Currency  string       `json:"currency"`
	Change    types.Number `json:"change"`
	Total     types.Number `json:"total"`
	Available types.Number `json:"available"`
}

// WsMarginBalance represents margin account balance push data
type WsMarginBalance struct {
	Timestamp    types.Time   `json:"timestamp_ms"`
	User         string       `json:"user"`
	CurrencyPair string       `json:"currency_pair"`
	Currency     string       `json:"currency"`
	Change       types.Number `json:"change"`
	Available    types.Number `json:"available"`
	Freeze       types.Number `json:"freeze"`
	Borrowed     string       `json:"borrowed"`
	Interest     string       `json:"interest"`
}

// WsFundingBalance represents funding balance push data.
type WsFundingBalance struct {
	Timestamp types.Time `json:"timestamp_ms"`
	User      string     `json:"user"`
	Currency  string     `json:"currency"`
	Change    string     `json:"change"`
	Freeze    string     `json:"freeze"`
	Lent      string     `json:"lent"`
}

// WsCrossMarginBalance represents a cross margin balance detail
type WsCrossMarginBalance struct {
	Timestamp types.Time   `json:"timestamp_ms"`
	User      string       `json:"user"`
	Currency  string       `json:"currency"`
	Change    string       `json:"change"`
	Total     types.Number `json:"total"`
	Available types.Number `json:"available"`
}

// WsCrossMarginLoan represents a cross margin loan push data
type WsCrossMarginLoan struct {
	Timestamp types.Time   `json:"timestamp"`
	User      string       `json:"user"`
	Currency  string       `json:"currency"`
	Change    string       `json:"change"`
	Total     types.Number `json:"total"`
	Available types.Number `json:"available"`
	Borrowed  string       `json:"borrowed"`
	Interest  string       `json:"interest"`
}

// WsFutureTicker represents a futures push data.
type WsFutureTicker struct {
	Contract              currency.Pair `json:"contract"`
	Last                  types.Number  `json:"last"`
	ChangePercentage      string        `json:"change_percentage"`
	FundingRate           string        `json:"funding_rate"`
	FundingRateIndicative string        `json:"funding_rate_indicative"`
	MarkPrice             types.Number  `json:"mark_price"`
	IndexPrice            types.Number  `json:"index_price"`
	TotalSize             types.Number  `json:"total_size"`
	Volume24H             types.Number  `json:"volume_24h"`
	Volume24HBtc          types.Number  `json:"volume_24h_btc"`
	Volume24HUsd          types.Number  `json:"volume_24h_usd"`
	QuantoBaseRate        string        `json:"quanto_base_rate"`
	Volume24HQuote        types.Number  `json:"volume_24h_quote"`
	Volume24HSettle       string        `json:"volume_24h_settle"`
	Volume24HBase         types.Number  `json:"volume_24h_base"`
	Low24H                types.Number  `json:"low_24h"`
	High24H               types.Number  `json:"high_24h"`
}

// WsFuturesTrades represents  a list of trades push data
type WsFuturesTrades struct {
	Size       float64       `json:"size"`
	ID         int64         `json:"id"`
	CreateTime types.Time    `json:"create_time_ms"`
	Price      types.Number  `json:"price"`
	Contract   currency.Pair `json:"contract"`
}

// WsFuturesOrderbookTicker represents the orderbook ticker push data
type WsFuturesOrderbookTicker struct {
	Timestamp     types.Time   `json:"t"`
	UpdateID      int64        `json:"u"`
	CurrencyPair  string       `json:"s"`
	BestBidPrice  types.Number `json:"b"`
	BestBidAmount float64      `json:"B"`
	BestAskPrice  types.Number `json:"a"`
	BestAskAmount float64      `json:"A"`
}

// WsFuturesAndOptionsOrderbookUpdate represents futures and options account orderbook update push data
type WsFuturesAndOptionsOrderbookUpdate struct {
	Timestamp      types.Time    `json:"t"`
	ContractName   currency.Pair `json:"s"`
	FirstUpdatedID int64         `json:"U"`
	LastUpdatedID  int64         `json:"u"`
	Bids           []Level       `json:"b"`
	Asks           []Level       `json:"a"`
}

// Level represents a level of orderbook data
type Level struct {
	Price types.Number `json:"p"`
	Size  float64      `json:"s"`
}

// WsFuturesOrderbookSnapshot represents a futures orderbook snapshot push data
type WsFuturesOrderbookSnapshot struct {
	Timestamp   types.Time    `json:"t"`
	Contract    currency.Pair `json:"contract"`
	OrderbookID int64         `json:"id"`
	Asks        []Level       `json:"asks"`
	Bids        []Level       `json:"bids"`
}

// WsFuturesOrderbookUpdateEvent represents futures orderbook push data with the event 'update'
type WsFuturesOrderbookUpdateEvent struct {
	Price        types.Number `json:"p"`
	Amount       float64      `json:"s"`
	CurrencyPair string       `json:"c"`
	ID           int64        `json:"id"`
}

// WsFuturesOrder represents futures order
type WsFuturesOrder struct {
	Contract     currency.Pair `json:"contract"`
	CreateTime   types.Time    `json:"create_time_ms"`
	FillPrice    float64       `json:"fill_price"`
	FinishAs     string        `json:"finish_as"`
	FinishTime   types.Time    `json:"finish_time_ms"`
	Iceberg      int64         `json:"iceberg"`
	ID           int64         `json:"id"`
	IsClose      bool          `json:"is_close"`
	IsLiq        bool          `json:"is_liq"`
	IsReduceOnly bool          `json:"is_reduce_only"`
	Left         float64       `json:"left"`
	Mkfr         float64       `json:"mkfr"`
	Price        float64       `json:"price"`
	Refr         int64         `json:"refr"`
	Refu         int64         `json:"refu"`
	Size         float64       `json:"size"`
	Status       string        `json:"status"`
	Text         string        `json:"text"`
	TimeInForce  string        `json:"tif"`
	Tkfr         float64       `json:"tkfr"`
	User         string        `json:"user"`
}

// WsFuturesUserTrade represents a futures account user trade push data
type WsFuturesUserTrade struct {
	ID         string        `json:"id"`
	CreateTime types.Time    `json:"create_time_ms"`
	Contract   currency.Pair `json:"contract"`
	OrderID    string        `json:"order_id"`
	Size       float64       `json:"size"`
	Price      types.Number  `json:"price"`
	Role       string        `json:"role"`
	Text       string        `json:"text"`
	Fee        float64       `json:"fee"`
	PointFee   int64         `json:"point_fee"`
}

// WsFuturesLiquidationNotification represents a liquidation notification push data
type WsFuturesLiquidationNotification struct {
	EntryPrice int64      `json:"entry_price"`
	FillPrice  float64    `json:"fill_price"`
	Left       float64    `json:"left"`
	Leverage   float64    `json:"leverage"`
	LiqPrice   int64      `json:"liq_price"`
	Margin     float64    `json:"margin"`
	MarkPrice  int64      `json:"mark_price"`
	OrderID    int64      `json:"order_id"`
	OrderPrice float64    `json:"order_price"`
	Size       float64    `json:"size"`
	Time       types.Time `json:"time_ms"`
	Contract   string     `json:"contract"`
	User       string     `json:"user"`
}

// WsFuturesAutoDeleveragesNotification represents futures auto deleverages push data
type WsFuturesAutoDeleveragesNotification struct {
	EntryPrice   float64    `json:"entry_price"`
	FillPrice    float64    `json:"fill_price"`
	PositionSize int64      `json:"position_size"`
	TradeSize    int64      `json:"trade_size"`
	Time         types.Time `json:"time_ms"`
	Contract     string     `json:"contract"`
	User         string     `json:"user"`
}

// WsPositionClose represents a close position futures push data
type WsPositionClose struct {
	Contract      string     `json:"contract"`
	ProfitAndLoss float64    `json:"pnl,omitempty"`
	Side          string     `json:"side"`
	Text          string     `json:"text"`
	Time          types.Time `json:"time_ms"`
	User          string     `json:"user"`

	// Added in options close position push datas
	SettleSize float64 `json:"settle_size,omitempty"`
	Underlying string  `json:"underlying,omitempty"`
}

// WsBalance represents a options and futures balance push data
type WsBalance struct {
	Balance float64    `json:"balance"`
	Change  float64    `json:"change"`
	Text    string     `json:"text"`
	Time    types.Time `json:"time_ms"`
	Type    string     `json:"type"`
	User    string     `json:"user"`
}

// WsFuturesReduceRiskLimitNotification represents a futures reduced risk limit push data
type WsFuturesReduceRiskLimitNotification struct {
	CancelOrders    int64      `json:"cancel_orders"`
	Contract        string     `json:"contract"`
	LeverageMax     int64      `json:"leverage_max"`
	LiqPrice        float64    `json:"liq_price"`
	MaintenanceRate float64    `json:"maintenance_rate"`
	RiskLimit       int64      `json:"risk_limit"`
	Time            types.Time `json:"time_ms"`
	User            string     `json:"user"`
}

// WsFuturesPosition represents futures notify positions update.
type WsFuturesPosition struct {
	Contract           string     `json:"contract"`
	CrossLeverageLimit float64    `json:"cross_leverage_limit"`
	EntryPrice         float64    `json:"entry_price"`
	HistoryPnl         float64    `json:"history_pnl"`
	HistoryPoint       int64      `json:"history_point"`
	LastClosePnl       float64    `json:"last_close_pnl"`
	Leverage           float64    `json:"leverage"`
	LeverageMax        float64    `json:"leverage_max"`
	LiqPrice           float64    `json:"liq_price"`
	MaintenanceRate    float64    `json:"maintenance_rate"`
	Margin             float64    `json:"margin"`
	Mode               string     `json:"mode"`
	RealisedPnl        float64    `json:"realised_pnl"`
	RealisedPoint      float64    `json:"realised_point"`
	RiskLimit          float64    `json:"risk_limit"`
	Size               float64    `json:"size"`
	Time               types.Time `json:"time_ms"`
	User               string     `json:"user"`
}

// WsFuturesAutoOrder represents an auto order push data.
type WsFuturesAutoOrder struct {
	User    int64 `json:"user"`
	Trigger struct {
		StrategyType int64  `json:"strategy_type"`
		PriceType    int64  `json:"price_type"`
		Price        string `json:"price"`
		Rule         int64  `json:"rule"`
		Expiration   int64  `json:"expiration"`
	} `json:"trigger"`
	Initial struct {
		Contract     string       `json:"contract"`
		Size         int64        `json:"size"`
		Price        types.Number `json:"price"`
		TimeInForce  string       `json:"tif"`
		Text         string       `json:"text"`
		Iceberg      int64        `json:"iceberg"`
		IsClose      bool         `json:"is_close"`
		IsReduceOnly bool         `json:"is_reduce_only"`
	} `json:"initial"`
	ID          int64      `json:"id"`
	TradeID     int64      `json:"trade_id"`
	Status      string     `json:"status"`
	Reason      string     `json:"reason"`
	CreateTime  types.Time `json:"create_time"`
	Name        string     `json:"name"`
	IsStopOrder bool       `json:"is_stop_order"`
	StopTrigger struct {
		Rule         int64  `json:"rule"`
		TriggerPrice string `json:"trigger_price"`
		OrderPrice   string `json:"order_price"`
	} `json:"stop_trigger"`
}

// WsOptionUnderlyingTicker represents options underlying ticker push data
type WsOptionUnderlyingTicker struct {
	TradePut   int64  `json:"trade_put"`
	TradeCall  int64  `json:"trade_call"`
	IndexPrice string `json:"index_price"`
	Name       string `json:"name"`
}

// WsOptionsTrades represents options trades for websocket push data.
type WsOptionsTrades struct {
	ID         int64         `json:"id"`
	CreateTime types.Time    `json:"create_time_ms"`
	Contract   currency.Pair `json:"contract"`
	Size       float64       `json:"size"`
	Price      float64       `json:"price"`
	Underlying string        `json:"underlying"`
	IsCall     bool          `json:"is_call"` // added in underlying trades
}

// WsOptionsUnderlyingPrice represents the underlying price.
type WsOptionsUnderlyingPrice struct {
	Underlying string     `json:"underlying"`
	Price      float64    `json:"price"`
	UpdateTime types.Time `json:"time_ms"`
}

// WsOptionsMarkPrice represents options mark price push data.
type WsOptionsMarkPrice struct {
	Contract   string     `json:"contract"`
	Price      float64    `json:"price"`
	UpdateTime types.Time `json:"time_ms"`
}

// WsOptionsSettlement represents a options settlement push data.
type WsOptionsSettlement struct {
	Contract     string     `json:"contract"`
	OrderbookID  int64      `json:"orderbook_id"`
	PositionSize float64    `json:"position_size"`
	Profit       float64    `json:"profit"`
	SettlePrice  float64    `json:"settle_price"`
	StrikePrice  float64    `json:"strike_price"`
	Tag          string     `json:"tag"`
	TradeID      int64      `json:"trade_id"`
	TradeSize    int64      `json:"trade_size"`
	Underlying   string     `json:"underlying"`
	UpdateTime   types.Time `json:"time_ms"`
}

// WsOptionsContract represents an option contract push data.
type WsOptionsContract struct {
	Contract          string     `json:"contract"`
	CreateTime        types.Time `json:"create_time"`
	ExpirationTime    types.Time `json:"expiration_time"`
	InitMarginHigh    float64    `json:"init_margin_high"`
	InitMarginLow     float64    `json:"init_margin_low"`
	IsCall            bool       `json:"is_call"`
	MaintMarginBase   float64    `json:"maint_margin_base"`
	MakerFeeRate      float64    `json:"maker_fee_rate"`
	MarkPriceRound    float64    `json:"mark_price_round"`
	MinBalanceShort   float64    `json:"min_balance_short"`
	MinOrderMargin    float64    `json:"min_order_margin"`
	Multiplier        float64    `json:"multiplier"`
	OrderPriceDeviate float64    `json:"order_price_deviate"`
	OrderPriceRound   float64    `json:"order_price_round"`
	OrderSizeMax      float64    `json:"order_size_max"`
	OrderSizeMin      float64    `json:"order_size_min"`
	OrdersLimit       float64    `json:"orders_limit"`
	RefDiscountRate   float64    `json:"ref_discount_rate"`
	RefRebateRate     float64    `json:"ref_rebate_rate"`
	StrikePrice       float64    `json:"strike_price"`
	Tag               string     `json:"tag"`
	TakerFeeRate      float64    `json:"taker_fee_rate"`
	Underlying        string     `json:"underlying"`
	Time              types.Time `json:"time_ms"`
}

// WsOptionsContractCandlestick represents an options contract candlestick push data.
type WsOptionsContractCandlestick struct {
	Timestamp          types.Time   `json:"t"`
	TotalVolume        float64      `json:"v"`
	ClosePrice         types.Number `json:"c"`
	HighestPrice       types.Number `json:"h"`
	LowestPrice        types.Number `json:"l"`
	OpenPrice          types.Number `json:"o"`
	Amount             types.Number `json:"a"`
	NameOfSubscription string       `json:"n"` // the format of <interval string>_<currency pair>
}

// WsOptionsOrderbookTicker represents options orderbook ticker push data.
type WsOptionsOrderbookTicker struct {
	UpdateTimestamp types.Time   `json:"t"`
	UpdateID        int64        `json:"u"`
	ContractName    string       `json:"s"`
	BidPrice        types.Number `json:"b"`
	BidSize         float64      `json:"B"`
	AskPrice        types.Number `json:"a"`
	AskSize         float64      `json:"A"`
}

// WsOptionsOrderbookSnapshot represents the options orderbook snapshot push data.
type WsOptionsOrderbookSnapshot struct {
	Timestamp types.Time    `json:"t"`
	Contract  currency.Pair `json:"contract"`
	ID        int64         `json:"id"`
	Asks      []struct {
		Price types.Number `json:"p"`
		Size  float64      `json:"s"`
	} `json:"asks"`
	Bids []struct {
		Price types.Number `json:"p"`
		Size  float64      `json:"s"`
	} `json:"bids"`
}

// WsOptionsOrder represents options order push data.
type WsOptionsOrder struct {
	ID           int64         `json:"id"`
	Contract     currency.Pair `json:"contract"`
	CreateTime   types.Time    `json:"create_time"`
	FillPrice    float64       `json:"fill_price"`
	FinishAs     string        `json:"finish_as"`
	Iceberg      float64       `json:"iceberg"`
	IsClose      bool          `json:"is_close"`
	IsLiq        bool          `json:"is_liq"`
	IsReduceOnly bool          `json:"is_reduce_only"`
	Left         float64       `json:"left"`
	Mkfr         float64       `json:"mkfr"`
	Price        float64       `json:"price"`
	Refr         float64       `json:"refr"`
	Refu         float64       `json:"refu"`
	Size         float64       `json:"size"`
	Status       string        `json:"status"`
	Text         string        `json:"text"`
	Tif          string        `json:"tif"`
	Tkfr         float64       `json:"tkfr"`
	Underlying   string        `json:"underlying"`
	User         string        `json:"user"`
	CreationTime types.Time    `json:"time_ms"`
}

// WsOptionsUserTrade represents user's personal trades of option account.
type WsOptionsUserTrade struct {
	ID         string        `json:"id"`
	Underlying string        `json:"underlying"`
	OrderID    string        `json:"order"`
	Contract   currency.Pair `json:"contract"`
	CreateTime types.Time    `json:"create_time_ms"`
	Price      types.Number  `json:"price"`
	Role       string        `json:"role"`
	Size       float64       `json:"size"`
}

// WsOptionsLiquidates represents the liquidates push data of option account.
type WsOptionsLiquidates struct {
	User        string     `json:"user"`
	InitMargin  float64    `json:"init_margin"`
	MaintMargin float64    `json:"maint_margin"`
	OrderMargin float64    `json:"order_margin"`
	Time        types.Time `json:"time_ms"`
}

// WsOptionsUserSettlement represents user's personal settlements push data of options account.
type WsOptionsUserSettlement struct {
	User         string     `json:"user"`
	Contract     string     `json:"contract"`
	RealisedPnl  float64    `json:"realised_pnl"`
	SettlePrice  float64    `json:"settle_price"`
	SettleProfit float64    `json:"settle_profit"`
	Size         float64    `json:"size"`
	StrikePrice  float64    `json:"strike_price"`
	Underlying   string     `json:"underlying"`
	SettleTime   types.Time `json:"time_ms"`
}

// WsOptionsPosition represents positions push data for options account.
type WsOptionsPosition struct {
	EntryPrice  float64    `json:"entry_price"`
	RealisedPnl float64    `json:"realised_pnl"`
	Size        float64    `json:"size"`
	Contract    string     `json:"contract"`
	User        string     `json:"user"`
	UpdateTime  types.Time `json:"time_ms"`
}

// InterSubAccountTransferParams represents parameters to transfer funds between sub-accounts.
type InterSubAccountTransferParams struct {
	Currency                currency.Code `json:"currency"` // Required
	SubAccountType          string        `json:"sub_account_type"`
	SubAccountFromUserID    string        `json:"sub_account_from"`      // Required
	SubAccountFromAssetType asset.Item    `json:"sub_account_from_type"` // Required
	SubAccountToUserID      string        `json:"sub_account_to"`        // Required
	SubAccountToAssetType   asset.Item    `json:"sub_account_to_type"`   // Required
	Amount                  types.Number  `json:"amount"`                // Required
}

// CreateAPIKeySubAccountParams represents subaccount new API key creation parameters.
type CreateAPIKeySubAccountParams struct {
	SubAccountUserID int64          `json:"user_id"`
	Body             *SubAccountKey `json:"body"`
}

// SubAccountKey represents sub-account key detail information
// this is a struct to be used for outbound requests.
type SubAccountKey struct {
	APIKeyName  string         `json:"name,omitempty"`
	Permissions []APIV4KeyPerm `json:"perms,omitempty"`
}

// APIV4KeyPerm represents an API Version 4 Key permission information
type APIV4KeyPerm struct {
	PermissionName string   `json:"name,omitempty"`
	ReadOnly       bool     `json:"read_only,omitempty"`
	IPWhitelist    []string `json:"ip_whitelist,omitempty"`
}

// CreateAPIKeyResponse represents an API key response object
type CreateAPIKeyResponse struct {
	UserID      string         `json:"user_id"`
	APIKeyName  string         `json:"name"` // API key name
	Permissions []APIV4KeyPerm `json:"perms"`
	IPWhitelist []string       `json:"ip_whitelist,omitempty"`
	APIKey      string         `json:"key"`
	Secret      string         `json:"secret"`
	State       int64          `json:"state"` // State 1 - normal 2 - locked 3 - frozen
	CreatedAt   types.Time     `json:"created_at"`
	UpdatedAt   types.Time     `json:"updated_at"`
}

// PriceAndAmount used in updating an order
type PriceAndAmount struct {
	Amount types.Number `json:"amount,omitempty"`
	Price  types.Number `json:"price,omitempty"`
}

// BalanceDetails represents a user's balance details
type BalanceDetails struct {
	Available             types.Number `json:"available"`
	Freeze                types.Number `json:"freeze"`
	Borrowed              types.Number `json:"borrowed"`
	NegativeLiabilities   types.Number `json:"negative_liab"`
	FuturesPosLiabilities types.Number `json:"futures_pos_liab"`
	Equity                types.Number `json:"equity"`
	TotalFreeze           types.Number `json:"total_freeze"`
	TotalLiabilities      types.Number `json:"total_liab"`
	SpotInUse             types.Number `json:"spot_in_use"`
	Funding               types.Number `json:"funding"`
	FundingVersion        types.Number `json:"funding_version"`
}

// UnifiedUserAccount represents a unified user account
type UnifiedUserAccount struct {
	UserID                         int64                            `json:"user_id"`
	Locked                         bool                             `json:"locked"`
	Balances                       map[currency.Code]BalanceDetails `json:"balances"`
	Total                          types.Number                     `json:"total"`
	Borrowed                       types.Number                     `json:"borrowed"`
	TotalInitialMargin             types.Number                     `json:"total_initial_margin"`
	TotalMarginBalance             types.Number                     `json:"total_margin_balance"`
	TotalMaintenanceMargin         types.Number                     `json:"total_maintenance_margin"`
	TotalInitialMarginRate         types.Number                     `json:"total_initial_margin_rate"`
	TotalMaintenanceMarginRate     types.Number                     `json:"total_maintenance_margin_rate"`
	TotalAvailableMargin           types.Number                     `json:"total_available_margin"`
	UnifiedAccountTotal            types.Number                     `json:"unified_account_total"`
	UnifiedAccountTotalLiabilities types.Number                     `json:"unified_account_total_liab"`
	UnifiedAccountTotalEquity      types.Number                     `json:"unified_account_total_equity"`
	Leverage                       types.Number                     `json:"leverage"`
	SpotOrderLoss                  types.Number                     `json:"spot_order_loss"`
	SpotHedge                      bool                             `json:"spot_hedge"`
	UseFunding                     bool                             `json:"use_funding"`
	RefreshTime                    types.Time                       `json:"refresh_time"`
}

// AccountDetails represents account detail information
type AccountDetails struct {
	IPWhitelist       []string  `json:"ip_whitelist"`
	CurrencyPairs     []string  `json:"currency_pairs"`
	UserID            int64     `json:"user_id"`
	VIPTier           int64     `json:"tier"`
	VIPTierExpireTime time.Time `json:"tier_expire_time"`
	Key               struct {
		Mode int64 `json:"mode"` // mode: 1 - classic account 2 - portfolio margin account
	} `json:"key"`
	CopyTradingRole int64 `json:"copy_trading_role"` // User role: 0 - Ordinary user 1 - Order leader 2 - Follower 3 - Order leader and follower
}

// UserTransactionRateLimitInfo represents user transaction rate limit information
type UserTransactionRateLimitInfo struct {
	Type      string       `json:"type"`
	Tier      types.Number `json:"tier"`
	Ratio     types.Number `json:"ratio"`
	MainRatio types.Number `json:"main_ratio"`
	UpdatedAt types.Time   `json:"updated_at"`
}
