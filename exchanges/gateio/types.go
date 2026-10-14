package gateio

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
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

	// FuturesContract.Type: a direct contract is worth QuantoMultiplier of its base currency,
	// an inverse one a single unit of its quote currency
	contractTypeInverse = "inverse"
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

// gateioAPIResponse is a generic wrapper for API responses.
type gateioAPIResponse[T any] struct {
	Timestamp types.Number `json:"timestamp"`
	Method    string       `json:"method"`
	Code      int64        `json:"code"`
	Message   string       `json:"message"`
	Data      T            `json:"data"`
	Version   string       `json:"version"`
}

// AsError returns the response status as an error, or nil when the request succeeded. It is used
// by SendAuthenticatedHTTPRequest to surface API level failures carried in a 200 response body.
func (p *gateioAPIResponse[T]) AsError() error {
	if p.Code != 0 {
		return fmt.Errorf("error code: %d message: %s", p.Code, p.Message)
	}
	return nil
}

// CurrencyInfo represents currency details with permission.
// The currency level withdraw_disabled, withdraw_delayed and deposit_disabled flags are deprecated in favour of the
// per chain flags in Chains
type CurrencyInfo struct {
	Currency         string               `json:"currency"`
	Name             string               `json:"name"`
	Delisted         bool                 `json:"delisted"`
	WithdrawDisabled bool                 `json:"withdraw_disabled"`
	WithdrawDelayed  bool                 `json:"withdraw_delayed"`
	DepositDisabled  bool                 `json:"deposit_disabled"`
	TradeDisabled    bool                 `json:"trade_disabled"`
	FixedFeeRate     types.Number         `json:"fixed_rate,omitempty"`
	Chain            string               `json:"chain"`
	Chains           []*SpotCurrencyChain `json:"chains"`
	TotalSupply      string               `json:"total_supply"` // Gate reports ∞ for currencies without a supply cap
	MarketCap        types.Number         `json:"market_cap"`
	Category         []string             `json:"category"`
}

// SpotCurrencyChain holds the deposit and withdrawal status of a currency on one of its chains
type SpotCurrencyChain struct {
	Name             string `json:"name"`
	Address          string `json:"addr"`
	WithdrawDisabled bool   `json:"withdraw_disabled"`
	WithdrawDelayed  bool   `json:"withdraw_delayed"`
	DepositDisabled  bool   `json:"deposit_disabled"`
}

// CurrencyPairDetail represents a single currency pair detail.
type CurrencyPairDetail struct {
	ID                            currency.Pair   `json:"id"`
	Base                          currency.Code   `json:"base"`
	BaseName                      string          `json:"base_name"`
	Quote                         currency.Code   `json:"quote"`
	QuoteName                     string          `json:"quote_name"`
	TradeQuotes                   []currency.Code `json:"trade_quotes"` // Quote currencies accepted by a unified market; empty when unsupported
	Fee                           types.Number    `json:"fee"`          // Deprecated by Gate
	MinBaseAmount                 types.Number    `json:"min_base_amount"`
	MinQuoteAmount                types.Number    `json:"min_quote_amount"`
	MaxBaseAmount                 types.Number    `json:"max_base_amount"`
	MaxQuoteAmount                types.Number    `json:"max_quote_amount"`
	AmountPrecision               uint8           `json:"amount_precision"`
	PricePrecision                uint8           `json:"precision"`
	TradeStatus                   string          `json:"trade_status"` // e.g. "untradable", "buyable", "sellable", "tradable"
	SellStart                     types.Time      `json:"sell_start"`
	BuyStart                      types.Time      `json:"buy_start"`
	DelistingTime                 types.Time      `json:"delisting_time"`
	Type                          string          `json:"type"` // normal or premarket
	TradeURL                      string          `json:"trade_url"`
	STTag                         bool            `json:"st_tag"`
	MaximumQuoteRisePercentage    types.Number    `json:"up_rate"`
	MaximumQuoteDeclinePercentage types.Number    `json:"down_rate"`
	Slippage                      types.Number    `json:"slippage"`
	MarketOrderMaxStock           types.Number    `json:"market_order_max_stock"`
	MarketOrderMaxMoney           types.Number    `json:"market_order_max_money"`
}

// Ticker holds detail ticker information for a currency pair
type Ticker struct {
	CurrencyPair     currency.Pair `json:"currency_pair"`
	Last             types.Number  `json:"last"`
	LowestAsk        types.Number  `json:"lowest_ask"`
	LowestSize       types.Number  `json:"lowest_size"`
	HighestBid       types.Number  `json:"highest_bid"`
	HighestSize      types.Number  `json:"highest_size"`
	ChangePercentage types.Number  `json:"change_percentage"`
	ChangeUTC0       types.Number  `json:"change_utc0"`
	ChangeUTC8       types.Number  `json:"change_utc8"`
	BaseVolume       types.Number  `json:"base_volume"`
	QuoteVolume      types.Number  `json:"quote_volume"`
	High24Hour       types.Number  `json:"high_24h"`
	Low24Hour        types.Number  `json:"low_24h"`
	ETFNetValue      types.Number  `json:"etf_net_value"`
	ETFPreNetValue   types.Number  `json:"etf_pre_net_value"`
	ETFPreTimestamp  types.Time    `json:"etf_pre_timestamp"`
	ETFLeverage      types.Number  `json:"etf_leverage"`
}

// OrderbookData holds orderbook ask and bid datas.
type OrderbookData struct {
	ID      int64                            `json:"id"`
	Current types.Time                       `json:"current"` // The timestamp of the response data being generated (in milliseconds)
	Update  types.Time                       `json:"update"`  // The timestamp of when the orderbook last changed (in milliseconds)
	Asks    orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids    orderbook.LevelsArrayPriceAmount `json:"bids"`
}

// MakeOrderbook converts OrderbookData into an Orderbook
func (a *OrderbookData) MakeOrderbook() *Orderbook {
	return &Orderbook{ID: a.ID, Current: a.Current, Update: a.Update, Asks: OrderbookLevels(a.Asks.Levels()), Bids: OrderbookLevels(a.Bids.Levels())}
}

// OrderbookLevels represents a slice of orderbook levels.
type OrderbookLevels orderbook.Levels

// UnmarshalJSON implements the json.Unmarshaler interface for OrderbookLevels.
func (o *OrderbookLevels) UnmarshalJSON(data []byte) error {
	var levels []OrderbookItem
	if err := json.Unmarshal(data, &levels); err != nil {
		return err
	}

	*o = make(OrderbookLevels, len(levels))
	for x := range levels {
		(*o)[x] = orderbook.Level{
			Price:  levels[x].Price.Float64(),
			Amount: levels[x].Amount.Float64(),
		}
	}
	return nil
}

// Levels converts OrderbookLevels to orderbook.Levels.
func (o *OrderbookLevels) Levels() orderbook.Levels {
	return orderbook.Levels(*o)
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
	Bids    OrderbookLevels `json:"bids"`
	Asks    OrderbookLevels `json:"asks"`
}

// Trade represents market trade.
type Trade struct {
	ID                string        `json:"id"`
	CreateTimeSeconds types.Time    `json:"create_time"`
	CreateTime        types.Time    `json:"create_time_ms"`
	CurrencyPair      string        `json:"currency_pair"`
	OrderID           string        `json:"order_id"`
	Side              string        `json:"side"`
	Role              string        `json:"role"`
	Amount            types.Number  `json:"amount"`
	Price             types.Number  `json:"price"`
	Fee               types.Number  `json:"fee"`
	FeeCurrency       string        `json:"fee_currency"`
	PointFee          types.Number  `json:"point_fee"`
	GTFee             types.Number  `json:"gt_fee"`
	AmendText         string        `json:"amend_text"`
	SequenceID        string        `json:"sequence_id"`
	Text              string        `json:"text"`
	Deal              types.Number  `json:"deal"`
	TradeQuote        currency.Code `json:"trade_quote"`
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
	Chain               string       `json:"chain"`
	ChineseChainName    string       `json:"name_cn"`
	ChainName           string       `json:"name_en"`
	ContractAddress     string       `json:"contract_address"`
	IsDisabled          uint64       `json:"is_disabled"`          // If it is disabled. 0 means NOT being disabled
	IsDepositDisabled   uint64       `json:"is_deposit_disabled"`  // Is deposit disabled. 0 means not
	IsWithdrawDisabled  uint64       `json:"is_withdraw_disabled"` // Is withdrawal disabled. 0 means not
	WithdrawalPrecision types.Number `json:"decimal"`
	IsTag               uint64       `json:"is_tag"` // Whether deposits and withdrawals need a tag. 0 means not
}

// ConvertibleSmallBalance holds a small balance that can be converted into GT
type ConvertibleSmallBalance struct {
	Currency         currency.Code `json:"currency"`
	AvailableBalance types.Number  `json:"available_balance"`
	EstimatedAsBTC   types.Number  `json:"estimated_as_btc"`
	ConvertibleToGT  types.Number  `json:"convertible_to_gt"`
}

// SmallCurrencyBalance holds a small currency balance detail
type SmallCurrencyBalance struct {
	ID         string        `json:"id"`
	CreateTime types.Time    `json:"create_time"`
	Currency   currency.Code `json:"currency"`
	Amount     types.Number  `json:"amount"`
	GTAmount   types.Number  `json:"gt_amount"`
}

// OrderbookOfLendingLoan represents order book of lending loans
type OrderbookOfLendingLoan struct {
	Rate   types.Number `json:"rate"`
	Amount types.Number `json:"amount"`
	Days   int64        `json:"days"`
}

// FuturesContract represents futures contract detailed data.
type FuturesContract struct {
	Name                  currency.Pair `json:"name"`
	Type                  string        `json:"type"`
	QuantoMultiplier      types.Number  `json:"quanto_multiplier"`
	RefDiscountRate       types.Number  `json:"ref_discount_rate"`
	OrderPriceDeviate     types.Number  `json:"order_price_deviate"`
	MaintenanceRate       types.Number  `json:"maintenance_rate"`
	MarkType              string        `json:"mark_type"`
	LastPrice             types.Number  `json:"last_price"`
	MarkPrice             types.Number  `json:"mark_price"`
	IndexPrice            types.Number  `json:"index_price"`
	FundingRateIndicative types.Number  `json:"funding_rate_indicative"`
	MarkPriceRound        types.Number  `json:"mark_price_round"`
	FundingOffset         types.Number  `json:"funding_offset"`
	Delisting             bool          `json:"in_delisting"`
	RiskLimitBase         types.Number  `json:"risk_limit_base"`
	InterestRate          types.Number  `json:"interest_rate"`
	OrderPriceRound       types.Number  `json:"order_price_round"`
	OrderSizeMin          types.Number  `json:"order_size_min"`
	RefRebateRate         types.Number  `json:"ref_rebate_rate"`
	FundingInterval       int64         `json:"funding_interval"`
	RiskLimitStep         types.Number  `json:"risk_limit_step"`
	LeverageMin           types.Number  `json:"leverage_min"`
	LeverageMax           types.Number  `json:"leverage_max"`
	RiskLimitMax          types.Number  `json:"risk_limit_max"`
	MakerFeeRate          types.Number  `json:"maker_fee_rate"`
	TakerFeeRate          types.Number  `json:"taker_fee_rate"`
	FundingRate           types.Number  `json:"funding_rate"`
	OrderSizeMax          types.Number  `json:"order_size_max"`
	FundingNextApply      types.Time    `json:"funding_next_apply"`
	ShortUsers            types.Number  `json:"short_users"`
	ConfigChangeTime      types.Time    `json:"config_change_time"`
	TradeSize             types.Number  `json:"trade_size"`
	PositionSize          types.Number  `json:"position_size"`
	LongUsers             types.Number  `json:"long_users"`
	FundingImpactValue    types.Number  `json:"funding_impact_value"`
	OrdersLimit           types.Number  `json:"orders_limit"`
	TradeID               int64         `json:"trade_id"`
	OrderbookID           int64         `json:"orderbook_id"`
	EnableBonus           bool          `json:"enable_bonus"`
	EnableCredit          bool          `json:"enable_credit"`
	CreateTime            types.Time    `json:"create_time"`
	FundingCapRatio       types.Number  `json:"funding_cap_ratio"`
	Status                string        `json:"status"`
	LaunchTime            types.Time    `json:"launch_time"`
	DelistingTime         types.Time    `json:"delisting_time"`
	DelistedTime          types.Time    `json:"delisted_time"`
	EnableDecimal         bool          `json:"enable_decimal"`
	MarketOrderSlipRatio  types.Number  `json:"market_order_slip_ratio"`
	MarketOrderSizeMax    types.Number  `json:"market_order_size_max"`
	FundingRateLimit      types.Number  `json:"funding_rate_limit"`
	ContractType          string        `json:"contract_type"`
	EnableCircuitBreaker  bool          `json:"enable_circuit_breaker"`
	// The fields below are undocumented but returned live
	CrossLeverageDefault types.Number `json:"cross_leverage_default"`
	VoucherLeverage      types.Number `json:"voucher_leverage"`
	IsPreMarket          bool         `json:"is_pre_market"`
}

// FuturesADLRiskStatesResponse holds the auto-deleveraging risk state of every futures market of a settlement currency
type FuturesADLRiskStatesResponse struct {
	Settle string                          `json:"settle"`
	States map[string]*FuturesADLRiskState `json:"states"`
}

// FuturesADLRiskState holds a futures market's auto-deleveraging risk state
type FuturesADLRiskState struct {
	State        string     `json:"state"`
	CalculatedAt types.Time `json:"calculated_at_ms"`
}

// TradingHistoryItem represents a futures or delivery market trade, or one of the user's own fills
type TradingHistoryItem struct {
	ID int64 `json:"id"`
	// CreateTime carries millisecond precision for market trades and is the only time sent for the user's own fills
	CreateTime             types.Time   `json:"create_time"`
	CreateTimeMilliseconds types.Time   `json:"create_time_ms"`
	Contract               string       `json:"contract"`
	Text                   string       `json:"text"`
	Size                   types.Number `json:"size"`
	Price                  types.Number `json:"price"`
	// The fields below are only returned for the user's own fills
	OrderID    string       `json:"order_id"`
	CloseSize  types.Number `json:"close_size"`
	Fee        types.Number `json:"fee"`
	PointFee   types.Number `json:"point_fee"`
	Role       string       `json:"role"`
	TradeValue types.Number `json:"trade_value"`
}

// FuturesTradeByTimeRangeResponse represents one of the user's futures fills queried by time range
type FuturesTradeByTimeRangeResponse struct {
	TradeID    string       `json:"trade_id"`
	CreateTime types.Time   `json:"create_time"`
	Contract   string       `json:"contract"`
	OrderID    string       `json:"order_id"`
	Size       types.Number `json:"size"`
	CloseSize  types.Number `json:"close_size"`
	Price      types.Number `json:"price"`
	Role       string       `json:"role"`
	Text       string       `json:"text"`
	Fee        types.Number `json:"fee"`
	PointFee   types.Number `json:"point_fee"`
}

// FuturesCandlestick represents futures candlestick data
type FuturesCandlestick struct {
	Timestamp    types.Time   `json:"t"`
	Volume       types.Number `json:"v"`
	ClosePrice   types.Number `json:"c"`
	HighestPrice types.Number `json:"h"`
	LowestPrice  types.Number `json:"l"`
	OpenPrice    types.Number `json:"o"`
	Sum          types.Number `json:"sum"` // Trading volume (unit: Quote currency)
	Name         string       `json:"n,omitempty"`
}

// FuturesPremiumIndexKlineResponse represents premium index K-Line information.
type FuturesPremiumIndexKlineResponse struct {
	UnixTimestamp types.Time   `json:"t"`
	ClosePrice    types.Number `json:"c"`
	HighestPrice  types.Number `json:"h"`
	LowestPrice   types.Number `json:"l"`
	OpenPrice     types.Number `json:"o"`
}

// FuturesTicker represents futures ticker data.
type FuturesTicker struct {
	Contract              currency.Pair `json:"contract"`
	ChangePercentage      types.Number  `json:"change_percentage"`
	Last                  types.Number  `json:"last"`
	Low24Hour             types.Number  `json:"low_24h"`
	High24Hour            types.Number  `json:"high_24h"`
	TotalSize             types.Number  `json:"total_size"`
	Volume24Hour          types.Number  `json:"volume_24h"`
	Volume24HourBTC       types.Number  `json:"volume_24h_btc"`
	Volume24HourUSD       types.Number  `json:"volume_24h_usd"`
	Volume24HourBase      types.Number  `json:"volume_24h_base"`
	Volume24HourQuote     types.Number  `json:"volume_24h_quote"`
	Volume24HourSettle    types.Number  `json:"volume_24h_settle"`
	QuantoMultiplier      types.Number  `json:"quanto_multiplier"`
	MarkPrice             types.Number  `json:"mark_price"`
	FundingRate           types.Number  `json:"funding_rate"`
	FundingRateIndicative types.Number  `json:"funding_rate_indicative"`
	IndexPrice            types.Number  `json:"index_price"`
	LowestAsk             types.Number  `json:"lowest_ask"`
	LowestAskSize         types.Number  `json:"lowest_size"`
	HighestBid            types.Number  `json:"highest_bid"`
	HighestBidSize        types.Number  `json:"highest_size"`
	ChangeUTC0            types.Number  `json:"change_utc0"`
	ChangeUTC8            types.Number  `json:"change_utc8"`
	ChangePrice           types.Number  `json:"change_price"`
	ChangeUTC0Price       types.Number  `json:"change_utc0_price"`
	ChangeUTC8Price       types.Number  `json:"change_utc8_price"`
}

// FuturesFundingRate represents futures funding rate response.
type FuturesFundingRate struct {
	Timestamp types.Time   `json:"t"`
	Rate      types.Number `json:"r"`
}

// BatchFundingRatesResponse holds a contract's funding rate history from a batch query
type BatchFundingRatesResponse struct {
	Contract string                `json:"contract"`
	Data     []*FuturesFundingRate `json:"data"`
}

// InsuranceBalance represents futures insurance balance item.
type InsuranceBalance struct {
	Timestamp types.Time   `json:"t"`
	Balance   types.Number `json:"b"`
}

// ContractStat represents futures stats
type ContractStat struct {
	Time                   types.Time   `json:"time"`
	LongShortTaker         types.Number `json:"lsr_taker"`
	LongShortAccount       types.Number `json:"lsr_account"`
	LongLiquidationSize    types.Number `json:"long_liq_size"`
	ShortLiquidationSize   types.Number `json:"short_liq_size"`
	OpenInterest           types.Number `json:"open_interest"`
	ShortLiquidationUsd    types.Number `json:"short_liq_usd"`
	MarkPrice              types.Number `json:"mark_price"`
	TopLongShortSize       types.Number `json:"top_lsr_size"`
	ShortLiquidationAmount types.Number `json:"short_liq_amount"`
	LongLiquidationAmount  types.Number `json:"long_liq_amount"`
	LastFundingRate        types.Number `json:"last_funding_rate"`
	OpenInterestUSD        types.Number `json:"open_interest_usd"`
	TopLongShortAccount    types.Number `json:"top_lsr_account"`
	LongLiquidationUSD     types.Number `json:"long_liq_usd"`
	TopLongSize            types.Number `json:"top_long_size"`
	TopShortSize           types.Number `json:"top_short_size"`
	ShortLiquidationUSDNew types.Number `json:"short_liq_usd_new"`
	LongLiquidationUSDNew  types.Number `json:"long_liq_usd_new"`
	TopLongAccount         types.Number `json:"top_long_account"`
	TopShortAccount        types.Number `json:"top_short_account"`
	LongTakerSize          types.Number `json:"long_taker_size"`
	ShortTakerSize         types.Number `json:"short_taker_size"`
	LongUsers              types.Number `json:"long_users"`
	ShortUsers             types.Number `json:"short_users"`
}

// IndexConstituent represents index constituents
type IndexConstituent struct {
	Index        string                     `json:"index"`
	Constituents []*IndexConstituentSources `json:"constituents"`
}

// IndexConstituentSources holds the symbols an exchange contributes to an index
type IndexConstituentSources struct {
	Exchange string   `json:"exchange"`
	Symbols  []string `json:"symbols"`
}

// LiquidationHistory represents a public futures liquidation order
type LiquidationHistory struct {
	Time       types.Time   `json:"time"`
	Contract   string       `json:"contract"`
	Size       types.Number `json:"size"`
	OrderSize  types.Number `json:"order_size"`
	OrderPrice types.Number `json:"order_price"`
	FillPrice  types.Number `json:"fill_price"`
	// RemainingAmount is documented as a reserved field with no current meaning
	RemainingAmount types.Number `json:"left"`
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
	SettlePriceInterval uint64       `json:"settle_price_interval"`
	SettlePriceDuration uint64       `json:"settle_price_duration"`
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
	OrderSizeMin        types.Number `json:"order_size_min"`
	OrderSizeMax        types.Number `json:"order_size_max"`
	OrdersLimit         uint64       `json:"orders_limit"`
	OrderbookID         uint64       `json:"orderbook_id"`
	TradeID             uint64       `json:"trade_id"`
	TradeSize           types.Number `json:"trade_size"`
	PositionSize        types.Number `json:"position_size"`
	ExpireTime          types.Time   `json:"expire_time"`
	ConfigChangeTime    types.Time   `json:"config_change_time"`
	InDelisting         bool         `json:"in_delisting"`
}

// OptionUnderlying represents an option underlying and its index price.
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
	IndexPrice        types.Number `json:"index_price"`
	OrderbookID       uint64       `json:"orderbook_id"`
	TradeID           uint64       `json:"trade_id"`
	TradeSize         types.Number `json:"trade_size"`
	PositionSize      types.Number `json:"position_size"`
	Underlying        string       `json:"underlying"`
	UnderlyingPrice   types.Number `json:"underlying_price"`
	Multiplier        types.Number `json:"multiplier"`
	OrderPriceRound   types.Number `json:"order_price_round"`
	MarkPriceRound    types.Number `json:"mark_price_round"`
	MakerFeeRate      types.Number `json:"maker_fee_rate"`
	TakerFeeRate      types.Number `json:"taker_fee_rate"`
	PriceLimitFeeRate types.Number `json:"price_limit_fee_rate"`
	RefDiscountRate   types.Number `json:"ref_discount_rate"`
	RefRebateRate     types.Number `json:"ref_rebate_rate"`
	OrderPriceDeviate types.Number `json:"order_price_deviate"`
	OrderSizeMin      types.Number `json:"order_size_min"`
	OrderSizeMax      types.Number `json:"order_size_max"`
	OrdersLimit       uint64       `json:"orders_limit"`
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

// FlashSwapCurrencyPair represents a supported flash swap pair.
type FlashSwapCurrencyPair struct {
	CurrencyPair  currency.Pair `json:"currency_pair"`
	SellCurrency  currency.Code `json:"sell_currency"`
	BuyCurrency   currency.Code `json:"buy_currency"`
	SellMinAmount types.Number  `json:"sell_min_amount"`
	SellMaxAmount types.Number  `json:"sell_max_amount"`
	BuyMinAmount  types.Number  `json:"buy_min_amount"`
	BuyMaxAmount  types.Number  `json:"buy_max_amount"`
}

// MyOptionSettlement represents option private settlement
type MyOptionSettlement struct {
	Size         types.Number  `json:"size"`
	SettleProfit types.Number  `json:"settle_profit"`
	Contract     currency.Pair `json:"contract"`
	StrikePrice  types.Number  `json:"strike_price"`
	Time         types.Time    `json:"time"`
	SettlePrice  types.Number  `json:"settle_price"`
	Underlying   string        `json:"underlying"`
	RealisedPNL  types.Number  `json:"realised_pnl"`
	Fee          types.Number  `json:"fee"`
}

// OptionsTicker represents tickers of options contracts
type OptionsTicker struct {
	Name                  currency.Pair `json:"name"`
	LastPrice             types.Number  `json:"last_price"`
	MarkPrice             types.Number  `json:"mark_price"`
	IndexPrice            types.Number  `json:"index_price"`
	PositionSize          types.Number  `json:"position_size"`
	Ask1Size              types.Number  `json:"ask1_size"`
	Ask1Price             types.Number  `json:"ask1_price"`
	Bid1Size              types.Number  `json:"bid1_size"`
	Bid1Price             types.Number  `json:"bid1_price"`
	Vega                  types.Number  `json:"vega"`
	Theta                 types.Number  `json:"theta"`
	Rho                   types.Number  `json:"rho"`
	Gamma                 types.Number  `json:"gamma"`
	Delta                 types.Number  `json:"delta"`
	MarkImpliedVolatility types.Number  `json:"mark_iv"`
	BidImpliedVolatility  types.Number  `json:"bid_iv"`
	AskImpliedVolatility  types.Number  `json:"ask_iv"`
	Leverage              types.Number  `json:"leverage"`
}

// OptionsUnderlyingTicker represents underlying ticker
type OptionsUnderlyingTicker struct {
	TradePut   uint64       `json:"trade_put"`
	TradeCall  uint64       `json:"trade_call"`
	IndexPrice types.Number `json:"index_price"`
}

// OptionTradeResponse represents an options market trade
type OptionTradeResponse struct {
	ID         int64         `json:"id"`
	CreateTime types.Time    `json:"create_time"`
	Contract   currency.Pair `json:"contract"`
	Size       types.Number  `json:"size"`
	Price      types.Number  `json:"price"`
}

// OptionAccount represents an option account.
type OptionAccount struct {
	UserID                       int64         `json:"user"`
	Total                        types.Number  `json:"total"`
	PositionValue                types.Number  `json:"position_value"`
	Equity                       types.Number  `json:"equity"`
	ShortEnabled                 bool          `json:"short_enabled"`
	MarketMakerProtectionEnabled bool          `json:"mmp_enabled"`
	LiquidationTriggered         bool          `json:"liq_triggered"`
	MarginMode                   uint8         `json:"margin_mode"`
	UnrealisedPNL                types.Number  `json:"unrealised_pnl"`
	InitialMargin                types.Number  `json:"init_margin"`
	MaintenanceMargin            types.Number  `json:"maint_margin"`
	OrderMargin                  types.Number  `json:"order_margin"`
	AskOrderMargin               types.Number  `json:"ask_order_margin"`
	BidOrderMargin               types.Number  `json:"bid_order_margin"`
	Available                    types.Number  `json:"available"`
	Point                        types.Number  `json:"point"`
	Currency                     currency.Code `json:"currency"`
	MaxOrderLimit                uint64        `json:"orders_limit"`
	PositionNotionalLimit        uint64        `json:"position_notional_limit"`
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
	User                  uint64        `json:"user"`
	Underlying            string        `json:"underlying"`
	UnderlyingPrice       types.Number  `json:"underlying_price"`
	Contract              currency.Pair `json:"contract"`
	Size                  int64         `json:"size"`
	EntryPrice            types.Number  `json:"entry_price"`
	MarkPrice             types.Number  `json:"mark_price"`
	MarkImpliedVolatility types.Number  `json:"mark_iv"`
	RealisedPNL           types.Number  `json:"realised_pnl"`
	UnrealisedPNL         types.Number  `json:"unrealised_pnl"`
	PendingOrders         uint64        `json:"pending_orders"`
	CloseOrder            CloseOrder    `json:"close_order"`
	Delta                 types.Number  `json:"delta"`
	Gamma                 types.Number  `json:"gamma"`
	Vega                  types.Number  `json:"vega"`
	Theta                 types.Number  `json:"theta"`
}

// ContractClosePosition represents user's liquidation history
type ContractClosePosition struct {
	PositionCloseTime types.Time   `json:"time"`
	Contract          string       `json:"contract"`
	Side              string       `json:"side"` // Position side, long or short
	PNL               types.Number `json:"pnl"`
	CloseOrderText    string       `json:"text"`
	SettleSize        types.Number `json:"settle_size"`
}

// OptionOrderRequest represents option order request body
type OptionOrderRequest struct {
	Contract              currency.Pair `json:"contract"`
	OrderSize             types.Number  `json:"size"`              // Positive to buy, negative to sell, zero with Close set to close the position
	Iceberg               int64         `json:"iceberg,omitempty"` // Display size for iceberg order. 0 for non-iceberg. Note that you will have to pay the taker fee for the hidden size
	Price                 types.Number  `json:"price"`             // Zero with an IOC time in force places a market order
	Close                 bool          `json:"close,omitempty"`
	ReduceOnly            bool          `json:"reduce_only,omitempty"`
	MarketMakerProtection bool          `json:"mmp,omitempty"`
	TimeInForce           string        `json:"tif,omitempty"`
	Text                  string        `json:"text,omitempty"`
}

// MarshalJSON sends the size as a JSON integer and the price as a decimal string, since types.Number
// marshals both as strings and zero as "", whereas Gate expects a zero size to close a position and
// a "0" price for a market order
func (o OptionOrderRequest) MarshalJSON() ([]byte, error) { //nolint:gocritic // a value receiver keeps the encoding when a value rather than a pointer is marshalled
	type alias OptionOrderRequest
	return json.Marshal(struct {
		*alias
		OrderSize float64 `json:"size"`
		Price     string  `json:"price"`
	}{
		alias:     (*alias)(&o),
		OrderSize: o.OrderSize.Float64(),
		Price:     strconv.FormatFloat(o.Price.Float64(), 'f', -1, 64),
	})
}

// OptionOrderResponse represents option order response detail
type OptionOrderResponse struct {
	OptionOrderID           int64        `json:"id"`
	User                    uint64       `json:"user"`
	CreateTime              types.Time   `json:"create_time"`
	FinishTime              types.Time   `json:"finish_time"`
	FinishAs                string       `json:"finish_as"`
	Status                  string       `json:"status"`
	Contract                string       `json:"contract"`
	Size                    types.Number `json:"size"`
	Iceberg                 int64        `json:"iceberg"`
	Price                   types.Number `json:"price"`
	IsOrderPositionClose    bool         `json:"is_close"`
	IsReduceOnly            bool         `json:"is_reduce_only"`
	IsOrderLiquidation      bool         `json:"is_liq"`
	IsMarketMakerProtection bool         `json:"is_mmp"`
	TimeInForce             string       `json:"tif"`
	RemainingAmount         types.Number `json:"left"`
	FillPrice               types.Number `json:"fill_price"`
	Text                    string       `json:"text"`
	TakerFeeRate            types.Number `json:"tkfr"`
	MakerFeeRate            types.Number `json:"mkfr"`
	ReferenceUserID         uint64       `json:"refu"`
	ReferrerRebate          types.Number `json:"refr"`
}

// AmendOptionOrderRequest represents a request to amend the price and size of an open options order
type AmendOptionOrderRequest struct {
	Contract currency.Pair `json:"contract"`
	Price    types.Number  `json:"price"`
	Size     int64         `json:"size"`
}

// CountdownCancelOptionOrdersRequest represents a request to set or clear the options order countdown, scoped
// to a contract or underlying when either is set
type CountdownCancelOptionOrdersRequest struct {
	Timeout    time.Duration // Whole seconds of at least 5; zero clears the countdown
	Contract   currency.Pair
	Underlying string
}

// MarshalJSON sends the timeout in whole seconds and omits an empty contract or underlying, which the
// json omitempty option cannot do for a currency.Pair
func (c CountdownCancelOptionOrdersRequest) MarshalJSON() ([]byte, error) {
	var contract string
	if !c.Contract.IsEmpty() {
		contract = c.Contract.String()
	}
	return json.Marshal(struct {
		Timeout    int64  `json:"timeout"`
		Contract   string `json:"contract,omitempty"`
		Underlying string `json:"underlying,omitempty"`
	}{
		Timeout:    int64(c.Timeout / time.Second),
		Contract:   contract,
		Underlying: c.Underlying,
	})
}

// OptionsMarketMakerProtectionRequest represents market maker protection settings for an options underlying
type OptionsMarketMakerProtectionRequest struct {
	Underlying    string
	Window        time.Duration // Up to 5 seconds in whole milliseconds; zero disables market maker protection
	FrozenPeriod  time.Duration // Whole milliseconds; zero keeps trading frozen until a reset
	QuantityLimit types.Number
	DeltaLimit    types.Number
}

// MarshalJSON sends the window and frozen period in milliseconds, keeping a zero value since each one
// carries meaning on the wire
func (o OptionsMarketMakerProtectionRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Underlying    string       `json:"underlying"`
		Window        int64        `json:"window"`
		FrozenPeriod  int64        `json:"frozen_period"`
		QuantityLimit types.Number `json:"qty_limit"`
		DeltaLimit    types.Number `json:"delta_limit"`
	}{
		Underlying:    o.Underlying,
		Window:        o.Window.Milliseconds(),
		FrozenPeriod:  o.FrozenPeriod.Milliseconds(),
		QuantityLimit: o.QuantityLimit,
		DeltaLimit:    o.DeltaLimit,
	})
}

// OptionsMarketMakerProtectionResponse represents the market maker protection settings and state of an options
// underlying
type OptionsMarketMakerProtectionResponse struct {
	Underlying    string        `json:"underlying"`
	Window        time.Duration `json:"-"`
	FrozenPeriod  time.Duration `json:"-"`
	QuantityLimit types.Number  `json:"qty_limit"`
	DeltaLimit    types.Number  `json:"delta_limit"`
	TriggerTime   types.Time    `json:"trigger_time_ms"`
	FrozenUntil   types.Time    `json:"frozen_until_ms"`
}

// UnmarshalJSON decodes the window and frozen period, which Gate sends in milliseconds
func (o *OptionsMarketMakerProtectionResponse) UnmarshalJSON(data []byte) error {
	type alias OptionsMarketMakerProtectionResponse
	aux := struct {
		*alias
		Window       int64 `json:"window"`
		FrozenPeriod int64 `json:"frozen_period"`
	}{alias: (*alias)(o)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	o.Window = time.Duration(aux.Window) * time.Millisecond
	o.FrozenPeriod = time.Duration(aux.FrozenPeriod) * time.Millisecond
	return nil
}

// OptionTradingHistory list personal trading history
type OptionTradingHistory struct {
	ID              uint64        `json:"id"`
	UnderlyingPrice types.Number  `json:"underlying_price"`
	Size            types.Number  `json:"size"`
	Contract        currency.Pair `json:"contract"`
	TradeRole       string        `json:"role"`
	CreateTime      types.Time    `json:"create_time"`
	OrderID         uint64        `json:"order_id"`
	Price           types.Number  `json:"price"`
}

// WithdrawalResponse represents a withdrawal record, as returned when withdrawing, cancelling a withdrawal or listing
// withdrawal records
type WithdrawalResponse struct {
	ID                string        `json:"id"`
	WithdrawalID      string        `json:"withdraw_id"`
	WithdrawalOrderID string        `json:"withdraw_order_id"`
	Timestamp         types.Time    `json:"timestamp"`
	FinalisedTime     types.Time    `json:"timestamp2"` // Cancellation time for CANCEL records and success time for DONE records
	Currency          currency.Code `json:"currency"`
	WithdrawalAddress string        `json:"address"`
	TransactionID     string        `json:"txid"`
	BlockNumber       types.Number  `json:"block_number"`
	Amount            types.Number  `json:"amount"`
	Fee               types.Number  `json:"fee"`
	Memo              string        `json:"memo"`
	Status            string        `json:"status"`
	FailReason        string        `json:"fail_reason"`
	BusinessType      string        `json:"type"`
	Chain             string        `json:"chain"`
	AssetClass        string        `json:"asset_class"`
}

// WithdrawalRequest represents currency withdrawal request param.
type WithdrawalRequest struct {
	Currency        currency.Code `json:"currency"`
	Amount          types.Number  `json:"amount"`
	Chain           string        `json:"chain"`
	AssetClass      string        `json:"asset_class,omitempty"`
	WithdrawID      string        `json:"withdraw_id,omitempty"`
	WithdrawOrderID string        `json:"withdraw_order_id,omitempty"`
	Address         string        `json:"address,omitempty"`
	Memo            string        `json:"memo,omitempty"`
}

// WithdrawalRecordsRequest holds query parameters for retrieving withdrawal records
type WithdrawalRecordsRequest struct {
	Currency currency.Code
	// WithdrawalID is a withdrawal record ID starting with "w"; Gate ignores the time range when it is set
	WithdrawalID string
	// AssetClass selects the main zone (SPOT) or the innovation zone (PILOT)
	AssetClass        string
	WithdrawalOrderID string
	From              time.Time
	To                time.Time
	Limit             uint64
	Offset            uint64
}

// UIDTransferRequest holds request parameters for transferring funds to another main account by UID
type UIDTransferRequest struct {
	RecipientUserID uint64        `json:"receive_uid"`
	Currency        currency.Code `json:"currency"`
	Amount          types.Number  `json:"amount"`
}

// UIDTransferResponse holds the order ID of an accepted UID transfer
type UIDTransferResponse struct {
	ID string `json:"id"`
}

// UIDTransferHistoryRequest holds query parameters for retrieving UID transfer records
type UIDTransferHistoryRequest struct {
	ID     uint64
	From   time.Time
	To     time.Time
	Limit  uint64
	Offset uint64
	// TransactionType selects withdraw (Gate's default) or deposit records
	TransactionType string
}

// UIDTransferRecord holds a UID transfer record
type UIDTransferRecord struct {
	ID              uint64        `json:"id"`
	InitiatorUserID uint64        `json:"push_uid"`
	RecipientUserID uint64        `json:"receive_uid"`
	Currency        currency.Code `json:"currency"`
	Amount          types.Number  `json:"amount"`
	CreateTime      types.Time    `json:"create_time"`
	Status          string        `json:"status"`
	Message         string        `json:"message"`
	TransactionType string        `json:"transaction_type"`
}

// CurrencyDepositAddressInfo represents a crypto deposit address
type CurrencyDepositAddressInfo struct {
	Currency            currency.Code            `json:"currency"`
	Address             string                   `json:"address"`
	MinDepositAmount    types.Number             `json:"min_deposit_amount"`
	MultichainAddresses []*MultiChainAddressItem `json:"multichain_addresses"`
}

// MultiChainAddressItem represents a multi-chain address item
type MultiChainAddressItem struct {
	Chain        string `json:"chain"`
	Address      string `json:"address"`
	PaymentID    string `json:"payment_id"`   // Tag or memo some currencies require when depositing
	PaymentName  string `json:"payment_name"` // Kind of PaymentID, either Tag or Memo
	ObtainFailed uint64 `json:"obtain_failed"`
	MinConfirms  uint64 `json:"min_confirms"`
}

// DepositRecord represents deposit record item
type DepositRecord struct {
	ID            string        `json:"id"`
	Timestamp     types.Time    `json:"timestamp"`
	Currency      currency.Code `json:"currency"`
	Address       string        `json:"address"`
	TransactionID string        `json:"txid"`
	Amount        types.Number  `json:"amount"`
	Memo          string        `json:"memo"`
	Status        string        `json:"status"`
	RefundStatus  string        `json:"refund_status"`
	Chain         string        `json:"chain"`
	Fee           types.Number  `json:"fee"`
}

// TransferCurrencyParam represents currency transfer.
type TransferCurrencyParam struct {
	Currency     currency.Code `json:"currency"`
	From         asset.Item    `json:"from"`
	To           asset.Item    `json:"to"`
	Amount       types.Number  `json:"amount"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Settle       currency.Code `json:"settle"`
}

// MarshalJSON sends the source and destination as Gate account names, which differ from asset.Item strings for
// futures assets, and the settlement currency in lower case as Gate reports it
func (t *TransferCurrencyParam) MarshalJSON() ([]byte, error) {
	from, err := walletAccountType(t.From)
	if err != nil {
		return nil, fmt.Errorf("%w: source account", err)
	}
	to, err := walletAccountType(t.To)
	if err != nil {
		return nil, fmt.Errorf("%w: destination account", err)
	}
	type Alias TransferCurrencyParam
	return json.Marshal(&struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Settle string `json:"settle"`
		*Alias
	}{
		From:   from,
		To:     to,
		Settle: t.Settle.Lower().String(),
		Alias:  (*Alias)(t),
	})
}

// TransactionIDResponse represents transaction ID
type TransactionIDResponse struct {
	TransactionID uint64 `json:"tx_id"`
}

// TradingAccountTransferResponse holds a transfer between the user's trading accounts
type TradingAccountTransferResponse struct {
	TransactionID string        `json:"tx_id"`
	Status        string        `json:"status"`
	Currency      currency.Code `json:"currency"`
	Amount        types.Number  `json:"amount"`
	FromAccount   string        `json:"from_account"`
	ToAccount     string        `json:"to_account"`
	Settle        currency.Code `json:"settle"`        // Set for futures, delivery and options transfers
	CurrencyPair  currency.Pair `json:"currency_pair"` // Set for margin transfers
}

// SubAccountTransferParam represents currency subaccount transfer request param
type SubAccountTransferParam struct {
	Currency       currency.Code `json:"currency"`
	SubAccount     string        `json:"sub_account"`
	Direction      string        `json:"direction"`
	Amount         types.Number  `json:"amount"`
	SubAccountType string        `json:"sub_account_type,omitempty"`
	ClientOrderID  string        `json:"client_order_id,omitempty"`
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
	ClientOrderID     string       `json:"client_order_id"`
	Status            string       `json:"status"`
}

// WithdrawalStatus represents currency withdrawal status
type WithdrawalStatus struct {
	Currency               currency.Code `json:"currency"`
	CurrencyName           string        `json:"name"`
	CurrencyNameChinese    string        `json:"name_cn"`
	DepositFee             types.Number  `json:"deposit"`
	WithdrawPercent        string        `json:"withdraw_percent"` // Formatted as a percentage, such as "0%"
	FixedWithdrawalFee     types.Number  `json:"withdraw_fix"`
	WithdrawDayLimit       types.Number  `json:"withdraw_day_limit"`
	WithdrawDayLimitRemain types.Number  `json:"withdraw_day_limit_remain"`
	WithdrawAmountMini     types.Number  `json:"withdraw_amount_mini"`
	WithdrawEachTimeLimit  types.Number  `json:"withdraw_eachtime_limit"`
	// WithdrawFixOnChains holds the fixed withdrawal fee keyed by chain
	WithdrawFixOnChains map[string]types.Number `json:"withdraw_fix_on_chains"`
	// WithdrawPercentOnChains holds the percentage withdrawal fee keyed by chain, formatted like WithdrawPercent
	WithdrawPercentOnChains map[string]string `json:"withdraw_percent_on_chains"`
}

// SubAccountBalances holds a sub-account's spot balances
type SubAccountBalances struct {
	UserID    string                         `json:"uid"`
	Available map[currency.Code]types.Number `json:"available"`
	Locking   map[currency.Code]types.Number `json:"locking"`
}

// AccountMargin represents margin account for specific user and several currencies
type AccountMargin struct {
	User                          uint64              `json:"user"`
	UpdateTime                    types.Time          `json:"update_time"`
	UpdateID                      uint64              `json:"update_id"`
	Total                         types.Number        `json:"total"`
	UnrealisedPNL                 types.Number        `json:"unrealised_pnl"`
	PositionMargin                types.Number        `json:"position_margin"`
	OrderMargin                   types.Number        `json:"order_margin"`
	Available                     types.Number        `json:"available"`
	Point                         types.Number        `json:"point"`
	Currency                      currency.Code       `json:"currency"`
	InDualMode                    bool                `json:"in_dual_mode"`
	EnableCredit                  bool                `json:"enable_credit"`
	PositionInitialMargin         types.Number        `json:"position_initial_margin"`
	MaintenanceMargin             types.Number        `json:"maintenance_margin"`
	Bonus                         types.Number        `json:"bonus"`
	EnableEvolvedClassic          bool                `json:"enable_evolved_classic"` // Deprecated.
	CrossOrderMargin              types.Number        `json:"cross_order_margin"`
	CrossInitialMargin            types.Number        `json:"cross_initial_margin"`
	CrossMaintenanceMargin        types.Number        `json:"cross_maintenance_margin"`
	CrossUnrealisedPNL            types.Number        `json:"cross_unrealised_pnl"`
	CrossAvailable                types.Number        `json:"cross_available"`
	CrossMarginBalance            types.Number        `json:"cross_margin_balance"`
	CrossMaintenanceMarginRate    types.Number        `json:"cross_mmr"`
	CrossInitialMarginRate        types.Number        `json:"cross_imr"`
	CrossVirtualUnrealisedPNL     types.Number        `json:"cross_virtual_unrealised_pnl"`
	IsolatedPositionMargin        types.Number        `json:"isolated_position_margin"`
	EnableNewDualMode             bool                `json:"enable_new_dual_mode"` // Deprecated.
	MarginMode                    uint8               `json:"margin_mode"`
	MarginModeName                string              `json:"margin_mode_name"`
	EnableTieredMaintenanceMargin bool                `json:"enable_tiered_mm"`
	EnableDualPlus                bool                `json:"enable_dual_plus"`
	PositionMode                  string              `json:"position_mode"`
	PositionVoucherTotal          types.Number        `json:"position_voucher_total"`
	History                       MarginHistoryTotals `json:"history"`
}

// MarginHistoryTotals represents margin history totals
type MarginHistoryTotals struct {
	DepositAndWithdrawal      types.Number `json:"dnw"`
	PNL                       types.Number `json:"pnl"`
	Fee                       types.Number `json:"fee"`
	ReferrerRebates           types.Number `json:"refr"`
	FundingCost               types.Number `json:"fund"`
	PointDepositAndWithdrawal types.Number `json:"point_dnw"`
	PointFee                  types.Number `json:"point_fee"`
	PointReferrerRebates      types.Number `json:"point_refr"`
	BonusDepositAndWithdrawal types.Number `json:"bonus_dnw"`
	BonusOffset               types.Number `json:"bonus_offset"`
	CrossSettle               types.Number `json:"cross_settle"`
}

// FuturesSubAccountBalance holds a sub-account's perpetual futures balances
type FuturesSubAccountBalance struct {
	UserID string `json:"uid"`
	// Available is keyed by the lower case settlement currency, such as "usdt"
	Available map[string]*AccountMargin `json:"available"`
}

// SubAccountMarginBalance holds a sub-account's isolated margin balances
type SubAccountMarginBalance struct {
	UserID    string                         `json:"uid"`
	Available []*SubAccountMarginPairBalance `json:"available"`
}

// SubAccountMarginPairBalance holds a sub-account's isolated margin balances for one currency pair
type SubAccountMarginPairBalance struct {
	CurrencyPair          currency.Pair         `json:"currency_pair"`
	AccountType           string                `json:"account_type"` // mmr for a maintenance margin rate account or inactive when the market is not activated
	Leverage              types.Number          `json:"leverage"`
	Locked                bool                  `json:"locked"`
	Risk                  types.Number          `json:"risk"` // Deprecated
	MaintenanceMarginRate types.Number          `json:"mmr"`
	Base                  MarginCurrencyBalance `json:"base"`
	Quote                 MarginCurrencyBalance `json:"quote"`
}

// TransferStatus holds an asset transfer status
type TransferStatus struct {
	TransactionID string `json:"tx_id"`
	Status        string `json:"status"`
}

// MarginCurrencyBalance represents a currency balance detail information.
type MarginCurrencyBalance struct {
	Currency       currency.Code `json:"currency"`
	Available      types.Number  `json:"available"`
	Locked         types.Number  `json:"locked"`
	BorrowedAmount types.Number  `json:"borrowed"`
	UnpaidInterest types.Number  `json:"interest"`
}

// SubAccountCrossMarginInfo represents subaccount's cross_margin account info
type SubAccountCrossMarginInfo struct {
	UserID    string                       `json:"uid"`
	Available *SubAccountCrossMarginDetail `json:"available"`
}

// SubAccountCrossMarginDetail represents a sub-account cross margin detail
type SubAccountCrossMarginDetail struct {
	UserID                     uint64                        `json:"user_id"` // 0 when the sub-account has not opened a cross margin account
	Locked                     bool                          `json:"locked"`
	Total                      types.Number                  `json:"total"`
	Borrowed                   types.Number                  `json:"borrowed"`
	Interest                   types.Number                  `json:"interest"` // Total unpaid interests in USDT, i.e., the sum of all currencies' interest*price*discount
	BorrowedNet                types.Number                  `json:"borrowed_net"`
	TotalNetAssets             types.Number                  `json:"net"`
	Leverage                   types.Number                  `json:"leverage"`
	Risk                       types.Number                  `json:"risk"`
	TotalInitialMargin         types.Number                  `json:"total_initial_margin"`
	TotalMarginBalance         types.Number                  `json:"total_margin_balance"`
	TotalMaintenanceMargin     types.Number                  `json:"total_maintenance_margin"`
	TotalInitialMarginRate     types.Number                  `json:"total_initial_margin_rate"`
	TotalMaintenanceMarginRate types.Number                  `json:"total_maintenance_margin_rate"`
	TotalAvailableMargin       types.Number                  `json:"total_available_margin"`
	CurrencyBalances           map[string]CrossMarginBalance `json:"balances"`
}

// CrossMarginBalance represents cross-margin currency balance detail
type CrossMarginBalance struct {
	Available           types.Number `json:"available"`
	Freeze              types.Number `json:"freeze"`
	Borrowed            types.Number `json:"borrowed"`
	Interest            types.Number `json:"interest"`
	Total               types.Number `json:"total"`
	BorrowedNet         types.Number `json:"borrowed_net"`
	TotalNetAssetInUSDT types.Number `json:"net"`
	PositionLeverage    types.Number `json:"leverage"`
	Risk                types.Number `json:"risk"` // Risk percentage; Liquidation is triggered when this falls below required margin. Calculation: total / (borrowed+interest)
}

// SavedAddressesRequest holds query parameters for retrieving saved withdrawal addresses
type SavedAddressesRequest struct {
	Currency currency.Code
	Chain    string
	// Verified filters by address type: "1" for verified addresses, "0" for normal addresses and empty for both
	Verified string
	Limit    uint64
	Page     uint64
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
	UserID              uint64       `json:"user_id"`
	TakerFee            types.Number `json:"taker_fee"`
	MakerFee            types.Number `json:"maker_fee"`
	RPIMakerFee         types.Number `json:"rpi_maker_fee"`
	GTDiscount          bool         `json:"gt_discount"`
	GTTakerFee          types.Number `json:"gt_taker_fee"`
	GTMakerFee          types.Number `json:"gt_maker_fee"`
	LoanFee             types.Number `json:"loan_fee"`
	PointType           string       `json:"point_type"`
	FuturesTakerFee     types.Number `json:"futures_taker_fee"`
	FuturesMakerFee     types.Number `json:"futures_maker_fee"`
	FuturesRPIMakerFee  types.Number `json:"futures_rpi_maker_fee"`
	DeliveryTakerFee    types.Number `json:"delivery_taker_fee"`
	DeliveryMakerFee    types.Number `json:"delivery_maker_fee"`
	FeeDeductionType    uint64       `json:"debit_fee"` // 1 - GT deduction, 2 - Point card deduction, 3 - VIP rates
	RPIMarketMakerLevel uint64       `json:"rpi_mm"`
}

// UsersAllAccountBalance represents user all account balances.
type UsersAllAccountBalance struct {
	Details map[string]CurrencyBalanceAmount `json:"details"`
	Total   CurrencyBalanceAmount            `json:"total"`
}

// CurrencyBalanceAmount represents currency and its amount.
type CurrencyBalanceAmount struct {
	Currency      currency.Code `json:"currency"`
	Amount        types.Number  `json:"amount"`
	UnrealisedPNL types.Number  `json:"unrealised_pnl"` // Only reported for futures, options, delivery and total balances
	Borrowed      types.Number  `json:"borrowed"`       // Only reported for margin and cross margin balances
}

// SpotTradingFeeRate holds the account's spot fee rates for a currency pair
type SpotTradingFeeRate struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	UserID       uint64        `json:"user_id"`
	TakerFee     types.Number  `json:"taker_fee"`
	MakerFee     types.Number  `json:"maker_fee"`
	RPIMakerFee  types.Number  `json:"rpi_maker_fee"` // Retail price improvement market maker fee rate
	GTDiscount   bool          `json:"gt_discount"`
	GTTakerFee   types.Number  `json:"gt_taker_fee"`
	GTMakerFee   types.Number  `json:"gt_maker_fee"`
	LoanFee      types.Number  `json:"loan_fee"`
	PointType    string        `json:"point_type"`
	DebitFee     uint64        `json:"debit_fee"` // 1 GT deduction, 2 point card deduction, 3 VIP rates
	RPIMMLevel   uint64        `json:"rpi_mm"`
}

// SpotAccount represents spot account
type SpotAccount struct {
	Currency  currency.Code `json:"currency"`
	Available types.Number  `json:"available"`
	Locked    types.Number  `json:"locked"`
	UpdateID  uint64        `json:"update_id"`
}

// GetSpotAccountBookRequest holds the filters for retrieving the spot account book
type GetSpotAccountBookRequest struct {
	Currency currency.Code
	From     time.Time
	To       time.Time
	Page     uint64
	Limit    uint64
	Type     string
	Code     string // Account change code, which takes priority over Type
}

// SpotAccountBookResponse holds a single spot account balance change
type SpotAccountBookResponse struct {
	ID       string        `json:"id"`
	Time     types.Time    `json:"time"`
	Currency currency.Code `json:"currency"`
	Change   types.Number  `json:"change"`
	Balance  types.Number  `json:"balance"`
	Type     string        `json:"type"` // Deprecated by Gate in favour of Code
	Code     string        `json:"code"`
	Text     string        `json:"text"`
}

// CreateOrderRequest represents a single order creation param.
type CreateOrderRequest struct {
	Text                      string        `json:"text"`
	CurrencyPair              currency.Pair `json:"currency_pair,omitzero"`
	TradeQuote                currency.Code `json:"trade_quote,omitzero"` // Quote currency actually traded; unified markets only
	Type                      string        `json:"type,omitempty"`
	Account                   asset.Item    `json:"account,omitempty"`
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
	// Slippage caps a market order's price deviation from the latest price as a ratio, e.g. 0.03 for 3%
	Slippage   types.Number   `json:"slippage,omitempty"`
	StopProfit *SpotOrderTPSL `json:"stop_profit,omitempty"`
	StopLoss   *SpotOrderTPSL `json:"stop_loss,omitempty"`
}

// SpotOrderTPSL holds the take profit or stop loss attached to a limit order
// When amending an order, an empty non-nil value cancels it while nil leaves it unchanged
type SpotOrderTPSL struct {
	TriggerPrice types.Number `json:"trigger_price,omitempty"`
	OrderPrice   types.Number `json:"order_price,omitempty"`
}

// SpotOrder holds a spot order, or one order's result within a batch or cancel all response
type SpotOrder struct {
	OrderID            string         `json:"id,omitempty"`
	BatchOrderID       string         `json:"order_id,omitempty"` // Only returned by batch create and amend responses
	Text               string         `json:"text,omitempty"`
	AmendText          string         `json:"amend_text,omitempty"`
	Succeeded          bool           `json:"succeeded"`
	ErrorLabel         string         `json:"label,omitempty"`
	Message            string         `json:"message,omitempty"`
	CreateTimeSeconds  types.Time     `json:"create_time,omitzero"`
	UpdateTimeSeconds  types.Time     `json:"update_time,omitzero"`
	CreateTime         types.Time     `json:"create_time_ms,omitzero"`
	UpdateTime         types.Time     `json:"update_time_ms,omitzero"`
	CurrencyPair       string         `json:"currency_pair,omitempty"`
	TradeQuote         currency.Code  `json:"trade_quote,omitzero"`
	Status             string         `json:"status,omitempty"`
	Type               string         `json:"type,omitempty"`
	Account            string         `json:"account,omitempty"`
	Side               string         `json:"side,omitempty"`
	Amount             types.Number   `json:"amount,omitempty"`
	Price              types.Number   `json:"price,omitempty"`
	TimeInForce        string         `json:"time_in_force,omitempty"`
	Iceberg            types.Number   `json:"iceberg,omitempty"`
	AutoRepay          bool           `json:"auto_repay"`
	AutoBorrow         bool           `json:"auto_borrow"`
	RemainingAmount    types.Number   `json:"left"`
	FilledAmount       types.Number   `json:"filled_amount"`
	AverageFillPrice   types.Number   `json:"avg_deal_price"`
	FeeDeducted        types.Number   `json:"fee"`
	FeeCurrency        string         `json:"fee_currency"`
	FillPrice          types.Number   `json:"fill_price"`   // Total filled in quote currency. Deprecated in favour of filled_total
	FilledTotal        types.Number   `json:"filled_total"` // Total filled in quote currency
	PointFee           types.Number   `json:"point_fee"`
	GTFee              types.Number   `json:"gt_fee,omitempty"`
	GTDiscount         bool           `json:"gt_discount"`
	GTMakerFee         types.Number   `json:"gt_maker_fee"`
	GTTakerFee         types.Number   `json:"gt_taker_fee"`
	RebatedFee         types.Number   `json:"rebated_fee"`
	RebatedFeeCurrency currency.Code  `json:"rebated_fee_currency"`
	STPID              uint64         `json:"stp_id"`
	STPAction          string         `json:"stp_act"`
	FinishAs           string         `json:"finish_as"`
	Slippage           types.Number   `json:"slippage"`
	StopProfit         *SpotOrderTPSL `json:"stop_profit"`
	StopLoss           *SpotOrderTPSL `json:"stop_loss"`
}

// SpotOrdersDetail represents list of orders for specific currency pair
type SpotOrdersDetail struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	Total        uint64        `json:"total"`
	Orders       []*SpotOrder  `json:"orders"`
}

// GetSpotOrdersRequest holds the filters for listing spot orders
type GetSpotOrdersRequest struct {
	CurrencyPair currency.Pair // Required for open orders
	Status       string        // open or finished
	Page         uint64
	Limit        uint64     // At most 100 for open orders
	Account      string     // spot, margin, cross_margin or unified
	From         time.Time  // Finished orders only, matched against the order finish time
	To           time.Time  // Finished orders only, matched against the order finish time
	Side         order.Side // Finished orders only
}

// ClosePositionRequestParam holds an order closing a position whose cross-currency trading is disabled
type ClosePositionRequestParam struct {
	Text         string        `json:"text"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Amount       types.Number  `json:"amount"`
	Price        types.Number  `json:"price"`
	ActionMode   string        `json:"action_mode,omitempty"` // ACK, RESULT or FULL, which is the default
}

// CancelAllSpotOrdersRequest holds the filters for cancelling all open spot orders
type CancelAllSpotOrdersRequest struct {
	CurrencyPair currency.Pair // Every currency pair when empty
	Side         order.Side    // Both sides when unset
	Account      string        // spot, margin, cross_margin or unified; unified accounts must set unified
	TradeQuote   currency.Code // Quote currency to cancel in a unified market; every quote currency when empty
	ActionMode   string        // ACK, RESULT or FULL, which is the default
}

// CancelOrderByIDParam represents cancel order by id request param.
type CancelOrderByIDParam struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	ID           string        `json:"id"`
	Account      string        `json:"account,omitempty"`     // Must be unified when cancelling with a unified account API key
	ActionMode   string        `json:"action_mode,omitempty"` // ACK, RESULT or FULL, which is the default
}

// CancelOrderByIDResponse holds the result of cancelling one order by ID
type CancelOrderByIDResponse struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	OrderID      string        `json:"id"`
	Text         string        `json:"text"`
	Succeeded    bool          `json:"succeeded"`
	Label        string        `json:"label"`
	Message      string        `json:"message"`
	Account      string        `json:"account"` // Deprecated by Gate
}

// AmendSpotOrderRequest holds the changes to an open spot order; either Amount or Price must be set
type AmendSpotOrderRequest struct {
	CurrencyPair currency.Pair  `json:"currency_pair"`
	Account      string         `json:"account,omitempty"` // spot, margin, cross_margin or unified
	Amount       types.Number   `json:"amount,omitempty"`
	Price        types.Number   `json:"price,omitempty"`
	AmendText    string         `json:"amend_text,omitempty"`
	ActionMode   string         `json:"action_mode,omitempty"` // ACK, RESULT or FULL, which is the default
	StopProfit   *SpotOrderTPSL `json:"stop_profit,omitempty"`
	StopLoss     *SpotOrderTPSL `json:"stop_loss,omitempty"`
}

// BatchAmendSpotOrderRequest holds the changes to one order of a batch amendment
type BatchAmendSpotOrderRequest struct {
	OrderID string `json:"order_id"` // Order ID or custom text ID
	AmendSpotOrderRequest
}

// GetMySpotTradingHistoryRequest holds the filters for retrieving the account's spot trades
type GetMySpotTradingHistoryRequest struct {
	CurrencyPair currency.Pair // Required when OrderID is set
	OrderID      string
	Page         uint64
	Limit        uint64 // At most 1000
	From         time.Time
	To           time.Time
}

// SpotPersonalTradeHistory represents personal trading history.
type SpotPersonalTradeHistory struct {
	TradeID           string        `json:"id"`
	CreateTimeSeconds types.Time    `json:"create_time"`
	CreateTime        types.Time    `json:"create_time_ms"`
	CurrencyPair      currency.Pair `json:"currency_pair"`
	OrderID           string        `json:"order_id"`
	Side              string        `json:"side"`
	Role              string        `json:"role"`
	Amount            types.Number  `json:"amount"`
	Price             types.Number  `json:"price"`
	Fee               types.Number  `json:"fee"`
	FeeCurrency       currency.Code `json:"fee_currency"`
	PointFee          types.Number  `json:"point_fee"`
	GTFee             types.Number  `json:"gt_fee"`
	AmendText         string        `json:"amend_text"`
	SequenceID        string        `json:"sequence_id"`
	Text              string        `json:"text"`
	Deal              types.Number  `json:"deal"`
	TradeQuote        currency.Code `json:"trade_quote"`
}

// CountdownCancelOrderParam represents countdown cancel order params
type CountdownCancelOrderParam struct {
	CurrencyPair currency.Pair `json:"currency_pair,omitzero"` // Every currency pair when empty
	Timeout      uint64        `json:"timeout"`                // Countdown in seconds of at least 5, or 0 to stop the countdown
}

// TriggerTimeResponse holds the time at which a cancel countdown expires
type TriggerTimeResponse struct {
	TriggerTime types.Time `json:"triggerTime"`
}

// GetSpotInsuranceHistoryRequest holds the filters for retrieving spot insurance fund history
type GetSpotInsuranceHistoryRequest struct {
	Business string // margin or unified
	Currency currency.Code
	From     time.Time
	To       time.Time
	Page     uint64
	Limit    uint64
}

// SpotInsuranceHistoryResponse holds an insurance fund balance at a point in time
type SpotInsuranceHistoryResponse struct {
	Currency currency.Code `json:"currency"`
	Balance  types.Number  `json:"balance"`
	Time     types.Time    `json:"time"`
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
	Expiration uint64       `json:"expiration,omitempty"` // Seconds to wait for the trigger before cancelling
}

// PutOrderData represents order detail for price triggered order request
type PutOrderData struct {
	Type        string       `json:"type,omitempty"` // limit, the default, or market
	Side        string       `json:"side"`
	Price       types.Number `json:"price"`
	Amount      types.Number `json:"amount"`
	Account     string       `json:"account"` // normal, margin or unified
	TimeInForce string       `json:"time_in_force"`
	AutoBorrow  bool         `json:"auto_borrow,omitempty"`
	AutoRepay   bool         `json:"auto_repay,omitempty"`
	Text        string       `json:"text,omitempty"`
}

// OrderID represents order creation ID response.
type OrderID struct {
	ID       int64  `json:"id"`
	IDString string `json:"id_string"` // ID as a string, which avoids precision loss outside Go
}

// SpotPriceTriggeredOrder represents spot price triggered order response data.
type SpotPriceTriggeredOrder struct {
	Trigger      TriggerPriceInfo `json:"trigger"`
	Put          PutOrderData     `json:"put"`
	AutoOrderID  uint64           `json:"id"`
	UserID       uint64           `json:"user"`
	CreationTime types.Time       `json:"ctime"`
	FireTime     types.Time       `json:"ftime"`
	FiredOrderID uint64           `json:"fired_order_id"`
	Status       string           `json:"status"`
	Reason       string           `json:"reason"`
	Market       currency.Pair    `json:"market"`
}

// SpotPOVOrderRequest holds a spot percentage of volume (POV) order
type SpotPOVOrderRequest struct {
	CurrencyPair      currency.Pair `json:"currency_pair"`
	Side              string        `json:"side"`
	Amount            types.Number  `json:"amount"`
	ParticipationRate uint64        `json:"participation_rate"` // Target share of market volume in percent: 5, 10, 20 or 40
	TimeToLive        string        `json:"ttl"`                // 1h, 6h, 12h, 1d, 2d, 3d, 4d, 5d, 6d or 7d
	LimitPrice        types.Number  `json:"limit_price,omitempty"`
	TriggerPrice      types.Number  `json:"trigger_price,omitempty"`
	Text              string        `json:"text,omitempty"`
}

// GetSpotPOVOrdersRequest holds the filters for listing spot POV orders
type GetSpotPOVOrdersRequest struct {
	CurrencyPair currency.Pair
	Status       string // open or finished
	Side         order.Side
	Page         uint64 // At most 100
	Limit        uint64
}

// SpotPOVOrderResponse holds a spot POV order
type SpotPOVOrderResponse struct {
	ID                string        `json:"id"`
	CurrencyPair      currency.Pair `json:"currency_pair"`
	Side              string        `json:"side"`
	Amount            types.Number  `json:"amount"`
	ParticipationRate uint64        `json:"participation_rate"`
	TimeToLive        string        `json:"ttl"`
	LimitPrice        types.Number  `json:"limit_price"`
	TriggerPrice      types.Number  `json:"trigger_price"`
	Status            string        `json:"status"`
	TerminatedAs      string        `json:"terminated_as"`
	StartTime         types.Time    `json:"start_time_ms"`
	EndTime           types.Time    `json:"end_time_ms"`
	ExpireTime        types.Time    `json:"expire_time_ms"`
	CreateTime        types.Time    `json:"create_time_ms"`
	UpdateTime        types.Time    `json:"update_time_ms"`
	Text              string        `json:"text"`
}

// CrossMarginCurrencies represents a currency supported by cross margin
type CrossMarginCurrencies struct {
	Name                 currency.Code `json:"name"`
	Rate                 types.Number  `json:"rate"`
	CurrencyPrecision    types.Number  `json:"prec"`
	Discount             types.Number  `json:"discount"`
	MinBorrowAmount      types.Number  `json:"min_borrow_amount"`
	UserMaxBorrowAmount  types.Number  `json:"user_max_borrow_amount"`
	TotalMaxBorrowAmount types.Number  `json:"total_max_borrow_amount"`
	Price                types.Number  `json:"price"` // Price change between this currency and USDT
	Loanable             bool          `json:"loanable"`
	Status               uint64        `json:"status"`
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
	UserID                      uint64                                `json:"user_id"`
	Locked                      bool                                  `json:"locked"`
	Balances                    map[string]CrossMarginCurrencyBalance `json:"balances"`
	Total                       types.Number                          `json:"total"`
	Borrowed                    types.Number                          `json:"borrowed"`
	Interest                    types.Number                          `json:"interest"`
	Risk                        types.Number                          `json:"risk"`
	TotalInitialMargin          types.Number                          `json:"total_initial_margin"`
	TotalMarginBalance          types.Number                          `json:"total_margin_balance"`
	TotalMaintenanceMargin      types.Number                          `json:"total_maintenance_margin"`
	TotalInitialMarginRate      types.Number                          `json:"total_initial_margin_rate"`
	TotalMaintenanceMarginRate  types.Number                          `json:"total_maintenance_margin_rate"`
	TotalAvailableMargin        types.Number                          `json:"total_available_margin"`
	TotalPortfolioMarginAccount types.Number                          `json:"portfolio_margin_total"`
}

// CrossMarginAccountHistoryItem represents a cross margin account change history item
type CrossMarginAccountHistoryItem struct {
	ID       string        `json:"id"`
	Time     types.Time    `json:"time"`
	Currency currency.Code `json:"currency"` // Currency changed
	Change   types.Number  `json:"change"`
	Balance  types.Number  `json:"balance"`
	Type     string        `json:"type"`
}

// CrossMarginBorrowLoanParams represents a cross margin borrow loan parameters
type CrossMarginBorrowLoanParams struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount"`
	Text     string        `json:"text,omitempty"`
}

// CrossMarginLoanResponse represents a cross margin borrow loan response
type CrossMarginLoanResponse struct {
	ID             string        `json:"id"`
	CreateTime     types.Time    `json:"create_time"`
	UpdateTime     types.Time    `json:"update_time"`
	Currency       currency.Code `json:"currency"`
	Amount         types.Number  `json:"amount"`
	Text           string        `json:"text"`
	Status         uint64        `json:"status"` // 1: failed, 2: borrowed, 3: repaid
	Repaid         types.Number  `json:"repaid"`
	RepaidInterest types.Number  `json:"repaid_interest"`
	UnpaidInterest types.Number  `json:"unpaid_interest"`
}

// CurrencyAndAmount represents request parameters for repayment
type CurrencyAndAmount struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount"`
}

// FlashSwapOrderParams represents create flash swap order request parameters.
type FlashSwapOrderParams struct {
	PreviewID    string        `json:"preview_id"`
	SellCurrency currency.Code `json:"sell_currency"`
	SellAmount   types.Number  `json:"sell_amount,omitempty"`
	BuyCurrency  currency.Code `json:"buy_currency"`
	BuyAmount    types.Number  `json:"buy_amount,omitempty"`
}

// FlashSwapPreviewRequest holds a flash swap to preview; set either SellAmount or BuyAmount
type FlashSwapPreviewRequest struct {
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
	PreviewID    string        `json:"preview_id"`
	SellCurrency currency.Code `json:"sell_currency"`
	SellAmount   types.Number  `json:"sell_amount"`
	BuyCurrency  currency.Code `json:"buy_currency"`
	BuyAmount    types.Number  `json:"buy_amount"`
	Price        types.Number  `json:"price"`
}

// DualInvestmentProductListRequest holds dual investment product list filters
type DualInvestmentProductListRequest struct {
	PlanID        uint64
	Coin          currency.Code
	Type          string // put to buy low or call to sell high
	QuoteCurrency currency.Code
	Sort          string // apy, short-period or multiple
	Page          uint64
	PageSize      uint64
}

// DualInvestmentPlan holds an earn dual investment plan detail
type DualInvestmentPlan struct {
	ID               uint64        `json:"id"`
	InstrumentName   string        `json:"instrument_name"`
	Type             string        `json:"type"`
	InvestCurrency   currency.Code `json:"invest_currency"`
	ExerciseCurrency currency.Code `json:"exercise_currency"`
	ExercisePrice    float64       `json:"exercise_price"`
	DeliveryTime     types.Time    `json:"delivery_time"`
	APYDisplay       types.Number  `json:"apy_display"`
	MinAmount        types.Number  `json:"min_amount"`
	StartTime        types.Time    `json:"start_time"`
	EndTime          types.Time    `json:"end_time"`
	Status           string        `json:"status"`
}

// DualInvestmentOrderListRequest holds dual investment order list filters
type DualInvestmentOrderListRequest struct {
	From   time.Time // Settlement time
	To     time.Time // Settlement time
	Type   string    // put to buy low or call to sell high
	Status string    // HOLD, REPAY, PROCESSING, SETTLEMENT_PROCESSING or ALL
	Coin   currency.Code
	Page   uint64
	Limit  uint64
}

// DualInvestmentOrderDetail holds a dual investment order item detail
type DualInvestmentOrderDetail struct {
	ID                 uint64        `json:"id"`
	PlanID             uint64        `json:"plan_id"`
	InvestAmount       types.Number  `json:"invest_amount"`
	SettlementAmount   types.Number  `json:"settlement_amount"`
	CreateTime         types.Time    `json:"create_time"`
	CompleteTime       types.Time    `json:"complete_time"`
	Status             string        `json:"status"`
	InvestCurrency     currency.Code `json:"invest_currency"`
	ExerciseCurrency   currency.Code `json:"exercise_currency"`
	SettlementCurrency currency.Code `json:"settlement_currency"`
	ExercisePrice      types.Number  `json:"exercise_price"`
	SettlementPrice    types.Number  `json:"settlement_price"`
	DeliveryTime       types.Time    `json:"delivery_time"`
	APYDisplay         types.Number  `json:"apy_display"`
	APYSettlement      types.Number  `json:"apy_settlement"`
	Text               string        `json:"text"`
}

// DualCurrencyEarningAsset represents the user's dual investment assets and interest
type DualCurrencyEarningAsset struct {
	UserAssetUSDT         types.Number `json:"user_asset_usdt"`
	UserAssetBTC          types.Number `json:"user_asset_btc"`
	UserTotalInterestUSDT types.Number `json:"user_total_interest_usdt"`
	UserTotalInterestBTC  types.Number `json:"user_total_interest_btc"`
}

// DualCurrencyEarlyRedemption represents a dual investment order's early redemption preview
type DualCurrencyEarlyRedemption struct {
	CreateTime          types.Time    `json:"create_timest"`
	DeliveryTime        types.Time    `json:"delivery_timest"`
	ExercisePrice       types.Number  `json:"exercise_price"`
	InvestAmount        types.Number  `json:"invest_amount"`
	InvestCurrency      currency.Code `json:"invest_currency"`
	Name                string        `json:"name"`
	OrderID             uint64        `json:"order_id"`
	RequestID           string        `json:"req_id"`
	RefundServiceCharge float64       `json:"refund_service_charge"`
	SettlePrice         types.Number  `json:"settle_price"`
	SettlementAmount    types.Number  `json:"settlement_amount"`
	SettlementCurrency  currency.Code `json:"settlement_currency"`
	SettlementInterest  types.Number  `json:"settlement_interest"`
	SettlementPrincipal types.Number  `json:"settlement_principle"`
	Type                string        `json:"type"`
	MoneyBackTime       types.Time    `json:"money_back_timest"`
}

// DualInvestmentOrderRequest holds a dual investment order request
type DualInvestmentOrderRequest struct {
	PlanID string       `json:"plan_id"`
	Amount types.Number `json:"amount"`
	Text   string       `json:"text,omitempty"` // Must start with t-
}

// DualModifyOrderReinvestRequest holds a dual investment order reinvestment change
type DualModifyOrderReinvestRequest struct {
	OrderID uint64 `json:"order_id"`
	// Status is 0 to switch reinvestment off and 1 to switch it on; it is always sent as 0 is a valid status
	Status                uint8  `json:"status"`
	EffectiveTimeDuration uint64 `json:"effective_time_duration,omitempty"` // In seconds, defaulting to 86400
}

// DualRecommendedProject holds a recommended dual investment product
type DualRecommendedProject struct {
	ID               uint64        `json:"id"`
	Category         uint64        `json:"category"`
	Type             string        `json:"type"`
	InvestCurrency   currency.Code `json:"invest_currency"`
	ExerciseCurrency currency.Code `json:"exercise_currency"`
	APYDisplay       types.Number  `json:"apy_display"`
	ExercisePrice    types.Number  `json:"exercise_price"`
	DeliveryTime     types.Time    `json:"delivery_timest"`
	MinAmount        types.Number  `json:"min_amount"`
	MaxAmount        types.Number  `json:"max_amount"`
	MinCopies        uint64        `json:"min_copies"`
	MaxCopies        uint64        `json:"max_copies"`
	InvestDays       uint64        `json:"invest_days"`
	InvestHours      types.Number  `json:"invest_hours"`
}

// AutoInvestCoinItem holds a currency supported by auto invest
type AutoInvestCoinItem struct {
	Key          currency.Code `json:"key"`
	Value        string        `json:"value"`
	AssetIconURL string        `json:"asset_icon_url"`
	Sort         uint64        `json:"sort"`
}

// AutoInvestMinAmountRequest holds parameters for querying the minimum investment amount
type AutoInvestMinAmountRequest struct {
	Money currency.Code         `json:"money"`
	Items []*AutoInvestPlanItem `json:"items"`
}

// AutoInvestMinAmountResponse holds the minimum investment amount response
type AutoInvestMinAmountResponse struct {
	MinAmount types.Number `json:"min_amount"`
}

// AutoInvestPlanExecutionRecord holds a single plan execution record item
type AutoInvestPlanExecutionRecord struct {
	ID            uint64        `json:"id"`
	Type          string        `json:"type"`
	Money         currency.Code `json:"money"`
	Asset         currency.Code `json:"asset"`
	UserID        uint64        `json:"user_id"`
	PlanID        uint64        `json:"plan_id"`
	PlanVersion   uint64        `json:"plan_version"`
	Amount        types.Number  `json:"amount"`
	CreateTime    types.Time    `json:"create_time"`
	UpdateTime    types.Time    `json:"update_time"`
	Status        string        `json:"status"`
	StatusType    uint64        `json:"status_type"`
	Side          uint64        `json:"side"` // 2 buy, anything else sell
	StatusMessage string        `json:"status_message"`
	Detail        string        `json:"detail"`
}

// AutoInvestPlanExecutionRecordsResponse holds the paginated response for plan execution records
type AutoInvestPlanExecutionRecordsResponse struct {
	Page      uint64                           `json:"page"`
	PageSize  uint64                           `json:"page_size"`
	TotalPage uint64                           `json:"total_page"`
	Total     uint64                           `json:"total"`
	List      []*AutoInvestPlanExecutionRecord `json:"list"`
}

// AutoInvestOrderItem holds an auto invest order detail item
type AutoInvestOrderItem struct {
	ID           uint64        `json:"id"`
	Type         string        `json:"type"`
	Amount       types.Number  `json:"amount"`
	PlanID       uint64        `json:"plan_id"`
	Side         uint64        `json:"side"`
	Asset        currency.Code `json:"asset"`
	RecordID     uint64        `json:"record_id"`
	TotalMoney   types.Number  `json:"total_money"`
	Market       currency.Pair `json:"market"`
	Price        types.Number  `json:"price"`
	CreateTime   types.Time    `json:"create_time"`
	Total        types.Number  `json:"total"`
	FundFlow     string        `json:"fund_flow"`
	ErrorCode    int64         `json:"error_code"`
	ErrorMessage string        `json:"error_msg"`
	Status       uint64        `json:"status"`
}

// AutoInvestConfigItem holds an investment currency configuration item
type AutoInvestConfigItem struct {
	Coin     currency.Code `json:"coin"`
	MaxLimit types.Number  `json:"max_limit"`
}

// AutoInvestPlanDetails holds auto invest plan details
type AutoInvestPlanDetails struct {
	ID         uint64                     `json:"id"`
	Version    uint64                     `json:"version"`
	Name       string                     `json:"name"`
	CreateTime types.Time                 `json:"create_time"`
	UpdateTime types.Time                 `json:"update_time"`
	UserID     uint64                     `json:"user_id"`
	Money      currency.Code              `json:"money"`
	Amount     types.Number               `json:"amount"`
	PeriodType string                     `json:"period_type"`
	PeriodDay  uint64                     `json:"period_day"`
	PeriodHour uint64                     `json:"period_hour"`
	Portfolio  []*AutoInvestPortfolioItem `json:"portfolio"`
	NextTime   types.Time                 `json:"next_time"`
	Period     uint64                     `json:"period"`
	FundSource string                     `json:"fund_source"`
	FundFlow   string                     `json:"fund_flow"`
}

// AutoInvestPortfolioItem holds a currency of an auto invest plan's portfolio
type AutoInvestPortfolioItem struct {
	Asset                currency.Code `json:"asset"`
	Ratio                types.Number  `json:"ratio"`
	CumulativeInvestment types.Number  `json:"cum_invest"`
	CumulativeHolding    types.Number  `json:"cum_hold"`
	CumulativeRedemption types.Number  `json:"cum_redeem"`
	AveragePrice         types.Number  `json:"avg_price"`
	RedeemStatus         uint64        `json:"redeem_status"`
	LendAmount           types.Number  `json:"lend_amount"`
}

// AutoInvestPlanListResponse holds the paginated response for auto invest plan list
type AutoInvestPlanListResponse struct {
	Page       uint64                   `json:"page"`
	PageSize   uint64                   `json:"page_size"`
	PageCount  uint64                   `json:"page_count"`
	TotalCount uint64                   `json:"total_count"`
	List       []*AutoInvestPlanDetails `json:"list"`
}

// AutoInvestPlanItem holds a single asset allocation item for plan creation
type AutoInvestPlanItem struct {
	Asset currency.Code `json:"asset"`
	Ratio types.Number  `json:"ratio"` // Share of the portfolio; every item's ratio must total 100
}

// CreateAutoInvestPlanRequest holds parameters for creating an auto invest plan
type CreateAutoInvestPlanRequest struct {
	PlanName        string `json:"plan_name,omitempty"`
	PlanDescription string `json:"plan_des,omitempty"`
	PlanPeriodType  string `json:"plan_period_type"` // daily, weekly, biweekly, monthly, hourly or 4-hourly
	// PlanPeriodDay is the day of the month (1-30) for monthly plans or the weekday (1-7 from Monday) for weekly and
	// biweekly plans; it is always sent as the API requires it even when the period type ignores it
	PlanPeriodDay uint64 `json:"plan_period_day"`
	// PlanPeriodHour is the hour from 0 to 23; it is always sent as 0 is a valid hour
	PlanPeriodHour uint64                `json:"plan_period_hour"`
	PlanMoney      currency.Code         `json:"plan_money"` // USDT or BTC
	PlanAmount     types.Number          `json:"plan_amount"`
	Type           uint8                 `json:"type,omitempty"` // 0 normal creation, 1 quick investment
	Items          []*AutoInvestPlanItem `json:"items"`
	FundSource     string                `json:"fund_source,omitempty"` // spot or earn, defaulting to spot
	FundFlow       string                `json:"fund_flow,omitempty"`   // auto_invest or earn, defaulting to auto_invest
}

// AutoInvestPlanResponse represents an auto-invest plan response.
type AutoInvestPlanResponse struct {
	ID         uint64        `json:"id"`
	Amount     types.Number  `json:"amount"`
	Money      currency.Code `json:"money"`
	NextTime   types.Time    `json:"next_time"`
	PeriodType string        `json:"period_type"`
	PeriodDay  uint64        `json:"period_day"`
	PeriodHour uint64        `json:"period_hour"`
	FundFlow   string        `json:"fund_flow"`
	FundSource string        `json:"fund_source"`
}

// AutoInvestPlanUpdateRequest holds parameters for updating an auto invest plan
type AutoInvestPlanUpdateRequest struct {
	PlanID     uint64 `json:"plan_id"`
	FundSource string `json:"fund_source,omitempty"` // spot or earn, defaulting to spot
	FundFlow   string `json:"fund_flow,omitempty"`   // auto_invest or earn, defaulting to auto_invest
}

// AutoInvestPlanStopRequest holds parameters for stopping an auto invest plan
type AutoInvestPlanStopRequest struct {
	PlanID uint64 `json:"plan_id"`
}

// AutoInvestPlanAddPositionRequest holds parameters for adding a position to an auto invest plan
type AutoInvestPlanAddPositionRequest struct {
	PlanID uint64       `json:"plan_id"`
	Amount types.Number `json:"amount"`
}

// FixedTermProduct holds a fixed-term earn product
type FixedTermProduct struct {
	ID                uint64        `json:"id"`
	Name              string        `json:"name"`
	Asset             currency.Code `json:"asset"`
	LockUpPeriodDays  uint64        `json:"lock_up_period"`
	MinLendAmount     types.Number  `json:"min_lend_amount"`
	UserMaxLendAmount types.Number  `json:"user_max_lend_amount"`
	TotalLendAmount   types.Number  `json:"total_lend_amount"`
	YearRate          types.Number  `json:"year_rate"`
	Type              uint8         `json:"type"` // 1 regular, 2 VIP
	PreRedeem         types.Boolean `json:"pre_redeem"`
	Reinvest          types.Boolean `json:"reinvest"`
	RedeemAccount     uint8         `json:"redeem_account"` // Whether fixed-to-flexible conversion is supported
	MinVIP            uint8         `json:"min_vip"`
	MaxVIP            uint8         `json:"max_vip"`
	Status            uint8         `json:"status"` // 1 unlisted, 2 listed, 3 delisted
	CreateTime        time.Time     `json:"create_time"`
	UserMaxLendVolume types.Number  `json:"user_max_lend_volume"`
	UserTotalAmount   types.Number  `json:"user_total_amount"`
	SaleStatus        uint8         `json:"sale_status"` // 1 on sale, 2 sold out
}

// FixedTermProductsResponse holds a page of fixed-term earn products
type FixedTermProductsResponse struct {
	List  []*FixedTermProduct `json:"list"`
	Total uint64              `json:"total"`
}

// FixedTermProductSimple holds a fixed-term earn product from a single currency's product list
type FixedTermProductSimple struct {
	ID               uint64        `json:"id"`
	Asset            currency.Code `json:"asset"`
	LockUpPeriodDays uint64        `json:"lock_up_period"`
	YearRate         types.Number  `json:"year_rate"`
	Type             uint8         `json:"type"` // 1 regular, 2 VIP
	PreRedeem        types.Boolean `json:"pre_redeem"`
	Reinvest         types.Boolean `json:"reinvest"`
	SimpleEarn       types.Boolean `json:"simple_earn"` // Whether fixed-to-flexible conversion is supported
	// RedeemAccount is returned by the live API in place of the documented simple_earn
	RedeemAccount uint8 `json:"redeem_account"`
	MinVIP        uint8 `json:"min_vip"`
	MaxVIP        uint8 `json:"max_vip"`
	SaleStatus    uint8 `json:"sale_status"` // 1 on sale, 2 sold out
}

// FixedTermProductsByAssetResponse holds a single currency's fixed-term earn products
type FixedTermProductsByAssetResponse struct {
	List []*FixedTermProductSimple `json:"list"`
}

// FixedTermSubscriptionRequest represents a fixed-term earn product subscription
type FixedTermSubscriptionRequest struct {
	ProductID uint64       `json:"product_id"`
	Amount    types.Number `json:"amount"`
	YearRate  types.Number `json:"year_rate,omitempty"`
	// ReinvestStatus is 1 to renew automatically at maturity; it is always sent so 0 explicitly disables renewal
	ReinvestStatus    uint8  `json:"reinvest_status"`
	RedeemAccountType uint8  `json:"redeem_account_type,omitempty"` // 1 spot account
	FinancialRateID   uint64 `json:"financial_rate_id,omitempty"`   // Interest rate boost coupon to apply
	SubBusiness       uint64 `json:"sub_business,omitempty"`
}

// FixedTermBusinessFilter narrows fixed-term earn queries to a business line
type FixedTermBusinessFilter struct {
	Business    uint8  `json:"business"` // 1 regular, 2 VIP
	SubBusiness uint64 `json:"sub_business"`
}

// FixedTermSubscriptionOrdersRequest holds fixed-term earn subscription order filters
type FixedTermSubscriptionOrdersRequest struct {
	ProductID      uint64
	OrderID        uint64
	Asset          currency.Code
	OrderType      uint8 // 1 current orders, 2 historical orders
	Page           uint64
	Limit          uint64
	SubBusiness    uint64
	BusinessFilter []FixedTermBusinessFilter
}

// FixedTermLendOrder holds a fixed-term earn subscription order
type FixedTermLendOrder struct {
	ID                   uint64                `json:"id"`
	Business             uint8                 `json:"business"` // 1 regular, 2 VIP
	OrderID              uint64                `json:"order_id"`
	UserID               uint64                `json:"user_id"`
	Asset                currency.Code         `json:"asset"`
	ProductID            uint64                `json:"product_id"`
	LockUpPeriodDays     uint64                `json:"lock_up_period"`
	Principal            types.Number          `json:"principal"`
	YearRate             types.Number          `json:"year_rate"`
	ProductType          uint8                 `json:"product_type"` // 1 regular, 2 VIP
	Interest             types.Number          `json:"interest"`
	Status               uint8                 `json:"status"` // 1 holding, 2 redeemed, 3 matured, 4 settled
	ReinvestStatus       uint8                 `json:"reinvest_status"`
	RedeemAccountType    uint8                 `json:"redeem_account_type"`
	OriginOrder          string                `json:"origin_order"` // Previous order of an automatically renewed order
	RedeemType           uint8                 `json:"redeem_type"`  // 1 early redemption, 2 redemption at maturity
	RedeemTime           time.Time             `json:"redeem_time"`
	FinishTime           time.Time             `json:"finish_time"`
	CreateTime           time.Time             `json:"create_time"`
	YearRatePercent      types.Number          `json:"year_rate_perent"`
	TotalYearRatePercent types.Number          `json:"total_year_rate_percent"`
	TotalInterest        types.Number          `json:"total_interest"`
	ProductInfo          *FixedTermProductInfo `json:"product_info"`
	BonusInfo            *FixedTermBonusInfo   `json:"bonus_info"`
	CouponInfo           *FixedTermCouponInfo  `json:"coupon_info"`
	RedeemAt             types.Time            `json:"redeem_at"`
	FinishAt             types.Time            `json:"finish_at"`
	CreateAt             types.Time            `json:"create_at"`
	Icon                 string                `json:"icon"`
}

// FixedTermProductInfo holds a fixed-term earn subscription order's product configuration
type FixedTermProductInfo struct {
	PreRedeem     types.Boolean `json:"pre_redeem"`
	Reinvest      types.Boolean `json:"reinvest"`
	RedeemAccount uint8         `json:"redeem_account"`
	MinVIP        uint8         `json:"min_vip"`
	MaxVIP        uint8         `json:"max_vip"`
}

// FixedTermBonusInfo holds a fixed-term earn bonus reward campaign
type FixedTermBonusInfo struct {
	ID                    uint64                `json:"id"`
	ProductID             uint64                `json:"product_id"`
	Asset                 currency.Code         `json:"asset"`
	BonusAsset            currency.Code         `json:"bonus_asset"`
	KYCLimit              string                `json:"kyc_limit"` // Comma separated KYC levels
	LadderAPR             []*FixedTermLadderAPR `json:"ladder_apr"`
	TotalBonusAmount      types.Number          `json:"total_bonus_amount"`
	UserTotalBonusAmount  types.Number          `json:"user_total_bonus_amount"`
	Status                uint8                 `json:"status"` // 1 unlisted, 2 listed, 3 delisted
	StartTime             time.Time             `json:"start_time"`
	EndTime               time.Time             `json:"end_time"`
	CreateTime            time.Time             `json:"create_time"`
	StartAt               types.Time            `json:"start_at"`
	EndAt                 types.Time            `json:"end_at"`
	TotalIssuedAmount     types.Number          `json:"total_issued_amount"`
	UserTotalIssuedAmount types.Number          `json:"user_total_issued_amount"`
	BonusAssetPrice       types.Number          `json:"bonus_asset_price"`
	ProductAssetPrice     types.Number          `json:"product_asset_price"`
	ProductYearRate       types.Number          `json:"product_year_rate"`
}

// FixedTermLadderAPR holds a tier of a fixed-term earn bonus reward campaign
type FixedTermLadderAPR struct {
	APR   types.Number `json:"apr"`
	Left  types.Number `json:"left"`
	Right types.Number `json:"right"`
}

// FixedTermCouponInfo holds a fixed-term earn interest rate boost coupon
type FixedTermCouponInfo struct {
	ID              uint64        `json:"id"`
	Business        uint8         `json:"business"`
	UserID          uint64        `json:"user_id"`
	Asset           currency.Code `json:"asset"`
	OrderID         uint64        `json:"order_id"`
	FinancialRateID uint64        `json:"financial_rate_id"`
	BuyLimitLow     types.Number  `json:"buy_limit_low"`
	BuyLimitHigh    types.Number  `json:"buy_limit_high"`
	RateDays        uint64        `json:"rate_day"`
	RateRatio       types.Number  `json:"rate_ratio"`
	CouponDays      uint64        `json:"coupon_days"`
	CouponPrincipal types.Number  `json:"coupon_principal"`
	CouponYearRate  types.Number  `json:"coupon_year_rate"`
	CouponInterest  types.Number  `json:"coupon_interest"`
	Status          uint8         `json:"status"` // 1 active, 2 settled
	FinishTime      time.Time     `json:"finish_time"`
	CreateTime      time.Time     `json:"create_time"`
}

// FixedTermSubscriptionOrdersResponse holds a page of fixed-term earn subscription orders
type FixedTermSubscriptionOrdersResponse struct {
	List  []*FixedTermLendOrder `json:"list"`
	Total uint64                `json:"total"`
}

// FixedTermHistoryRequest holds fixed-term earn history filters
type FixedTermHistoryRequest struct {
	ProductID      uint64
	OrderID        string
	Asset          currency.Code
	Type           uint8 // 1 subscription, 2 redemption, 3 interest, 4 bonus reward
	Page           uint64
	Limit          uint64
	StartTime      time.Time
	EndTime        time.Time
	SubBusiness    uint64
	BusinessFilter []FixedTermBusinessFilter
}

// FixedTermHistoryRecord holds a fixed-term earn history record
type FixedTermHistoryRecord struct {
	ID               uint64        `json:"id"`
	OrderID          uint64        `json:"order_id"`
	UserID           uint64        `json:"user_id"`
	Asset            currency.Code `json:"asset"`
	UniqueTime       string        `json:"uniq_time"` // Date the record belongs to
	BonusID          uint64        `json:"bonus_id"`
	ProductID        uint64        `json:"product_id"`
	BonusAsset       currency.Code `json:"bonus_asset"`
	TotalPrincipal   types.Number  `json:"total_principal"`
	Amount           types.Number  `json:"amount"`
	AssetPrice       types.Number  `json:"asset_price"`
	Status           uint64        `json:"status"`
	Detail           string        `json:"detail"`
	CreateTime       time.Time     `json:"create_time"`
	CreateAt         types.Time    `json:"create_at"`
	LockUpPeriodDays uint64        `json:"lock_up_period"`
}

// FixedTermHistoryResponse holds a page of fixed-term earn history records
type FixedTermHistoryResponse struct {
	List  []*FixedTermHistoryRecord `json:"list"`
	Total uint64                    `json:"total"`
}

// FuturesAccount represents futures account detail
type FuturesAccount struct {
	User                          int64                 `json:"user"`
	Currency                      currency.Code         `json:"currency"`
	Total                         types.Number          `json:"total"`
	UnrealisedPNL                 types.Number          `json:"unrealised_pnl"`
	PositionMargin                types.Number          `json:"position_margin"`
	OrderMargin                   types.Number          `json:"order_margin"`
	Available                     types.Number          `json:"available"`
	Point                         types.Number          `json:"point"`
	Bonus                         types.Number          `json:"bonus"`
	EnabledCredit                 bool                  `json:"enable_credit"`
	InDualMode                    bool                  `json:"in_dual_mode"`
	UpdateTime                    types.Time            `json:"update_time"`
	UpdateID                      int64                 `json:"update_id"`
	PositionInitialMargin         types.Number          `json:"position_initial_margin"`
	MaintenanceMargin             types.Number          `json:"maintenance_margin"`
	MarginMode                    int64                 `json:"margin_mode"` // 0: classic futures account or classic spot margin mode of a unified account, 1: multi-currency margin, 2: portfolio margin, 3: single-currency margin
	EnabledEvolvedClassic         bool                  `json:"enable_evolved_classic"`
	CrossOrderMargin              types.Number          `json:"cross_order_margin"`
	CrossInitialMargin            types.Number          `json:"cross_initial_margin"`
	CrossMaintenanceMargin        types.Number          `json:"cross_maintenance_margin"`
	CrossUnrealisedPNL            types.Number          `json:"cross_unrealised_pnl"`
	CrossAvailable                types.Number          `json:"cross_available"`
	CrossMarginBalance            types.Number          `json:"cross_margin_balance"`
	CrossMaintenanceMarginRate    types.Number          `json:"cross_mmr"`
	CrossInitialMarginRate        types.Number          `json:"cross_imr"`
	IsolatedPositionMargin        types.Number          `json:"isolated_position_margin"`
	EnableNewDualMode             bool                  `json:"enable_new_dual_mode"`
	EnableTieredMaintenanceMargin bool                  `json:"enable_tiered_mm"`
	EnableDualPlus                bool                  `json:"enable_dual_plus"`
	PositionMode                  string                `json:"position_mode"`
	History                       FuturesAccountHistory `json:"history"`
}

// FuturesAccountHistory holds a futures account's cumulative fund flows
type FuturesAccountHistory struct {
	DepositAndWithdrawal      types.Number `json:"dnw"`
	ProfitAndLoss             types.Number `json:"pnl"`
	Fee                       types.Number `json:"fee"`
	ReferrerRebate            types.Number `json:"refr"`
	Funding                   types.Number `json:"fund"`
	PointDepositAndWithdrawal types.Number `json:"point_dnw"`
	PointFee                  types.Number `json:"point_fee"`
	PointReferrerRebate       types.Number `json:"point_refr"`
	BonusDepositAndWithdrawal types.Number `json:"bonus_dnw"`
	BonusOffset               types.Number `json:"bonus_offset"`
	CrossSettle               types.Number `json:"cross_settle"`
}

// AccountBookItem represents account book item
type AccountBookItem struct {
	Time     types.Time   `json:"time"`
	Change   types.Number `json:"change"`
	Balance  types.Number `json:"balance"`
	Text     string       `json:"text"`
	Type     string       `json:"type"`
	Contract string       `json:"contract"`
	TradeID  string       `json:"trade_id"`
	ID       string       `json:"id"`
}

// Position represents futures position
type Position struct {
	User                       int64         `json:"user"`
	Contract                   currency.Pair `json:"contract"`
	Size                       types.Number  `json:"size"` // Denotes long or short position, positive for long, negative for short
	HedgeStatus                string        `json:"hedge_status"`
	HedgedSize                 types.Number  `json:"hedged_size"`
	UnhedgedSize               types.Number  `json:"unhedged_size"`
	Leverage                   types.Number  `json:"leverage"`
	RiskLimit                  types.Number  `json:"risk_limit"`
	LeverageMax                types.Number  `json:"leverage_max"`
	MaintenanceRate            types.Number  `json:"maintenance_rate"`
	Value                      types.Number  `json:"value"`
	Margin                     types.Number  `json:"margin"`
	EntryPrice                 types.Number  `json:"entry_price"`
	LiquidationPrice           types.Number  `json:"liq_price"`
	MarkPrice                  types.Number  `json:"mark_price"`
	InitialMargin              types.Number  `json:"initial_margin"`
	MaintenanceMargin          types.Number  `json:"maintenance_margin"`
	UnrealisedPNL              types.Number  `json:"unrealised_pnl"`
	RealisedPNL                types.Number  `json:"realised_pnl"`
	RealisedPNLPosition        types.Number  `json:"pnl_pnl"`
	RealisedPNLFundingFee      types.Number  `json:"pnl_fund"`
	RealisedPNLTransactionFees types.Number  `json:"pnl_fee"`
	HistoryPNL                 types.Number  `json:"history_pnl"`
	LastClosePNL               types.Number  `json:"last_close_pnl"`
	RealisedPNLPoint           types.Number  `json:"realised_point"`
	RealisedPNLHistoryPoint    types.Number  `json:"history_point"`
	ADLRanking                 uint8         `json:"adl_ranking"` // Ranking of auto deleveraging, a total of 1-5 grades, 1 is the highest, 5 is the lowest, and 6 is the special case when there is no position held or in liquidation
	PendingOrders              uint64        `json:"pending_orders"`
	CloseOrder                 CloseOrder    `json:"close_order"`
	Mode                       string        `json:"mode"`
	CrossLeverageLimit         types.Number  `json:"cross_leverage_limit"`
	UpdateTime                 types.Time    `json:"update_time"`
	OpenTime                   types.Time    `json:"open_time"`
	UpdateID                   int64         `json:"update_id"`
	TradeMaxSize               types.Number  `json:"trade_max_size"`
	RiskLimitTable             string        `json:"risk_limit_table"`
	AverageMaintenanceRate     types.Number  `json:"average_maintenance_rate"`
	PositionID                 int64         `json:"pid"`
	VoucherSize                types.Number  `json:"voucher_size"`
	VoucherMargin              types.Number  `json:"voucher_margin"`
	VoucherID                  int64         `json:"voucher_id"`
	TradeLongSize              types.Number  `json:"trade_long_size"`
	TradeShortSize             types.Number  `json:"trade_short_size"`
	PositionMarginMode         string        `json:"pos_margin_mode"`
	PositionLeverage           types.Number  `json:"lever"` // Indicates the current leverage of the position, applicable to both isolated and cross margin, gradually replacing the current leverage and cross_leverage_limit
}

// positionReply decodes a position reply, which the docs define as an object but which has also been handled as a
// single element array
type positionReply struct {
	Position *Position
}

// UnmarshalJSON implements json.Unmarshaler
func (p *positionReply) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] != '[' {
		p.Position = new(Position)
		return json.Unmarshal(data, p.Position)
	}
	var positions []*Position
	if err := json.Unmarshal(data, &positions); err != nil {
		return err
	}
	if len(positions) != 1 {
		return fmt.Errorf("%w: expected one position, received %d", common.ErrInvalidResponse, len(positions))
	}
	p.Position = positions[0]
	return nil
}

// CloseOrder represents close order information
type CloseOrder struct {
	ID            int64        `json:"id"`
	Price         types.Number `json:"price"`
	IsLiquidation bool         `json:"is_liq"`
}

// FuturesHistoricalPositionResponse represents a historical futures position snapshot queried by time range
type FuturesHistoricalPositionResponse struct {
	Contract                currency.Pair `json:"contract"`
	Size                    types.Number  `json:"size"`
	Leverage                types.Number  `json:"leverage"`
	RiskLimit               types.Number  `json:"risk_limit"`
	LeverageMax             types.Number  `json:"leverage_max"`
	MaintenanceRate         types.Number  `json:"maintenance_rate"`
	Margin                  types.Number  `json:"margin"`
	LiquidationPrice        types.Number  `json:"liq_price"`
	RealisedPNL             types.Number  `json:"realised_pnl"`
	HistoryPNL              types.Number  `json:"history_pnl"`
	LastClosePNL            types.Number  `json:"last_close_pnl"`
	RealisedPNLPoint        types.Number  `json:"realised_point"`
	RealisedPNLHistoryPoint types.Number  `json:"history_point"`
	Mode                    string        `json:"mode"`
	CrossLeverageLimit      types.Number  `json:"cross_leverage_limit"`
	EntryPrice              types.Number  `json:"entry_price"`
	Time                    types.Time    `json:"time"`
}

// FuturesLeverageResponse holds a contract's leverage for a position margin mode and side
type FuturesLeverageResponse struct {
	Leverage types.Number `json:"Lever"`
}

// FuturesPositionMarginModeRequest holds a request to switch a futures position between isolated and cross margin
type FuturesPositionMarginModeRequest struct {
	Mode     string        `json:"mode"` // ISOLATED or CROSS
	Contract currency.Pair `json:"contract"`
}

// FuturesOrderCreateParams represents future order creation parameters
type FuturesOrderCreateParams struct {
	Contract                  currency.Pair `json:"contract"`
	Size                      types.Number  `json:"size"` // positive long, negative short, zero to close a position
	Iceberg                   types.Number  `json:"iceberg"`
	Price                     types.Number  `json:"price"` // zero with an ioc time in force places a market order
	TimeInForce               string        `json:"tif"`
	Text                      string        `json:"text,omitempty"`  // errors when empty; Either populated or omitted
	ClosePosition             bool          `json:"close,omitempty"` // Size needs to be zero if true
	ReduceOnly                bool          `json:"reduce_only,omitempty"`
	AutoSize                  string        `json:"auto_size,omitempty"` // either close_long or close_short, requires zero in size field
	Settle                    currency.Code `json:"-"`                   // Used in URL. REST Calls only.
	SelfTradePreventionAction string        `json:"stp_act,omitempty"`
	PositionID                int64         `json:"pid,omitempty"`
	MarketOrderSlipRatio      types.Number  `json:"market_order_slip_ratio,omitempty"`
	PositionMarginMode        string        `json:"pos_margin_mode,omitempty"`
	ActionMode                string        `json:"action_mode,omitempty"`
	TakeProfitTriggerPrice    types.Number  `json:"tpsl_tp_trigger_price,omitempty"`
	StopLossTriggerPrice      types.Number  `json:"tpsl_sl_trigger_price,omitempty"`
}

// MarshalJSON sends size, iceberg and price as decimal strings even when zero, since Gate reads a zero price
// with an ioc time in force as a market order and a zero size as a close order, whereas types.Number marshals
// zero as an empty string
func (c FuturesOrderCreateParams) MarshalJSON() ([]byte, error) { //nolint:gocritic // hugeParam: a value receiver lets json call this for unaddressable values, such as orders marshalled by value
	type alias FuturesOrderCreateParams
	return json.Marshal(&struct {
		Contract currency.Pair `json:"contract"`
		Size     string        `json:"size"`
		Iceberg  string        `json:"iceberg"`
		Price    string        `json:"price"`
		*alias
	}{
		Contract: c.Contract,
		Size:     strconv.FormatFloat(c.Size.Float64(), 'f', -1, 64),
		Iceberg:  strconv.FormatFloat(c.Iceberg.Float64(), 'f', -1, 64),
		Price:    strconv.FormatFloat(c.Price.Float64(), 'f', -1, 64),
		alias:    (*alias)(&c),
	})
}

// DeliveryOrderCreateParams represents delivery order creation parameters.
type DeliveryOrderCreateParams struct {
	Contract                  currency.Pair `json:"contract"`
	Size                      types.Number  `json:"size"`    // positive long, negative short
	Iceberg                   int64         `json:"iceberg"` // required; can be zero
	Price                     types.Number  `json:"price"`   // NOTE: Market orders require string "0"
	TimeInForce               string        `json:"tif"`
	Text                      string        `json:"text,omitempty"`  // errors when empty; Either populated or omitted
	ClosePosition             bool          `json:"close,omitempty"` // Size needs to be zero if true
	ReduceOnly                bool          `json:"reduce_only,omitempty"`
	AutoSize                  string        `json:"auto_size,omitempty"` // either close_long or close_short, requires zero in size field
	Settle                    currency.Code `json:"-"`                   // Used in URL. REST Calls only.
	SelfTradePreventionAction string        `json:"stp_act,omitempty"`
}

// MarshalJSON sends the size as a JSON integer and the price as a decimal string, since types.Number
// marshals both as strings and zero as "", whereas Gate expects a zero size to close a position and
// a "0" price for a market order
func (c DeliveryOrderCreateParams) MarshalJSON() ([]byte, error) { //nolint:gocritic // a value receiver keeps the encoding when a value rather than a pointer is marshalled
	type alias DeliveryOrderCreateParams
	return json.Marshal(struct {
		*alias
		Size  float64 `json:"size"`
		Price string  `json:"price"`
	}{
		alias: (*alias)(&c),
		Size:  c.Size.Float64(),
		Price: strconv.FormatFloat(c.Price.Float64(), 'f', -1, 64),
	})
}

// NumericString holds a value Gate sends as either a JSON string or a bare JSON integer, depending on the endpoint
// or transport. Keeping the text avoids the float64 rounding of identifiers beyond 2^53.
type NumericString string

// UnmarshalJSON implements json.Unmarshaler
func (n *NumericString) UnmarshalJSON(data []byte) error {
	switch {
	case len(data) == 0 || string(data) == "null":
		*n = ""
	case data[0] == '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*n = NumericString(s)
	default:
		if _, err := strconv.ParseInt(string(data), 10, 64); err != nil {
			return fmt.Errorf("%w: %s", errInvalidNumericString, data)
		}
		*n = NumericString(data)
	}
	return nil
}

// String returns the value's text
func (n NumericString) String() string {
	return string(n)
}

// FuturesOrder represents future order response
type FuturesOrder struct {
	ID int64 `json:"id"`
	// User is a JSON number over REST but a string over websocket, both of which decode into this type
	User                      NumericString `json:"user"`
	Contract                  currency.Pair `json:"contract"`
	CreateTime                types.Time    `json:"create_time"`
	UpdateTime                types.Time    `json:"update_time"`
	Size                      types.Number  `json:"size"`
	Iceberg                   types.Number  `json:"iceberg"`
	RemainingAmount           types.Number  `json:"left"` // Size left to be traded
	Price                     types.Number  `json:"price"`
	FillPrice                 types.Number  `json:"fill_price"` // Fill price of the order. total filled in quote currency.
	MakerFeeRate              types.Number  `json:"mkfr"`
	TakerFeeRate              types.Number  `json:"tkfr"`
	TimeInForce               string        `json:"tif"`
	ReferenceUserID           int64         `json:"refu"`
	IsReduceOnly              bool          `json:"is_reduce_only"`
	IsClose                   bool          `json:"is_close"`
	IsOrderForLiquidation     bool          `json:"is_liq"`
	Text                      string        `json:"text"`
	Status                    string        `json:"status"`
	FinishTime                types.Time    `json:"finish_time"`
	FinishAs                  string        `json:"finish_as"`
	SelfTradePreventionID     types.Number  `json:"stp_id"`
	SelfTradePreventionAction string        `json:"stp_act"`
	AmendText                 string        `json:"amend_text"`
	OrderValue                types.Number  `json:"order_value"`
	TradeValue                types.Number  `json:"trade_value"`
	MarketOrderSlipRatio      types.Number  `json:"market_order_slip_ratio"`
	PositionMarginMode        string        `json:"pos_margin_mode"`
	ActionMode                string        `json:"action_mode"`
	TakeProfitTriggerPrice    types.Number  `json:"tpsl_tp_trigger_price"`
	StopLossTriggerPrice      types.Number  `json:"tpsl_sl_trigger_price"`
}

// BatchFuturesOrderResponse represents one order's outcome in a batch futures order placement or amendment
type BatchFuturesOrderResponse struct {
	Succeeded bool   `json:"succeeded"`
	Label     string `json:"label"`
	Detail    string `json:"detail"`
	FuturesOrder
}

// CancelFuturesOrdersRequest holds the filters for cancelling all open futures orders
type CancelFuturesOrdersRequest struct {
	Contract          currency.Pair // Cancels the open orders of every contract when empty
	Side              string        // bid or ask; both are cancelled when empty
	ActionMode        string        // ACK, RESULT or FULL
	ExcludeReduceOnly bool
	Text              string
}

// AmendFuturesOrderParam represents amend futures order parameter
type AmendFuturesOrderParam struct {
	Size       types.Number `json:"size,omitempty"` // keeps the order's sign; a size at or below the filled size cancels the order
	Price      types.Number `json:"price,omitempty"`
	AmendText  string       `json:"amend_text,omitempty"`
	Text       string       `json:"text,omitempty"`
	ActionMode string       `json:"action_mode,omitempty"`
}

// BatchAmendFuturesOrderRequest holds one order's amendment in a batch futures order amendment
type BatchAmendFuturesOrderRequest struct {
	OrderID    uint64       `json:"order_id,omitempty"`
	Text       string       `json:"text,omitempty"`
	Size       types.Number `json:"size,omitempty"`
	Price      types.Number `json:"price,omitempty"`
	AmendText  string       `json:"amend_text,omitempty"`
	ActionMode string       `json:"action_mode,omitempty"`
}

// CancelFuturesOrderResultResponse represents one order's outcome in a futures batch cancellation by order ID
type CancelFuturesOrderResultResponse struct {
	UserID    int64  `json:"user_id"`
	ID        string `json:"id"`
	Succeeded bool   `json:"succeeded"`
	Message   string `json:"message"`
}

// PositionCloseHistoryResponse represents a close position history detail
type PositionCloseHistoryResponse struct {
	Time                  types.Time   `json:"time"`
	ProfitAndLoss         types.Number `json:"pnl"`
	ProfitAndLossPosition types.Number `json:"pnl_pnl"`
	ProfitAndLossFunding  types.Number `json:"pnl_fund"`
	ProfitAndLossFee      types.Number `json:"pnl_fee"`
	Side                  string       `json:"side"`
	Contract              string       `json:"contract"`
	Text                  string       `json:"text"`
	MaxSize               types.Number `json:"max_size"`
	AccumulatedClosedSize types.Number `json:"accum_size"`
	FirstOpenTime         types.Time   `json:"first_open_time"`
	LongAveragePrice      types.Number `json:"long_price"`  // the opening average price of a long position, or the closing one of a short
	ShortAveragePrice     types.Number `json:"short_price"` // the closing average price of a long position, or the opening one of a short
}

// LiquidationHistoryItem liquidation history item
type LiquidationHistoryItem struct {
	Time            types.Time    `json:"time"`
	Contract        currency.Pair `json:"contract"`
	Size            types.Number  `json:"size"`
	Leverage        types.Number  `json:"leverage"`
	Margin          types.Number  `json:"margin"`
	EntryPrice      types.Number  `json:"entry_price"`
	MarkPrice       types.Number  `json:"mark_price"`
	OrderPrice      types.Number  `json:"order_price"`
	FillPrice       types.Number  `json:"fill_price"`
	LiqPrice        types.Number  `json:"liq_price"`
	OrderID         int64         `json:"order_id"`
	RemainingAmount types.Number  `json:"left"`
}

// FuturesAutoDeleverageResponse represents an auto-deleveraging of one of the user's futures positions
type FuturesAutoDeleverageResponse struct {
	Time               types.Time    `json:"time"`
	User               int64         `json:"user"`
	OrderID            int64         `json:"order_id"`
	Contract           currency.Pair `json:"contract"`
	Leverage           types.Number  `json:"leverage"`
	CrossLeverageLimit types.Number  `json:"cross_leverage_limit"`
	EntryPrice         types.Number  `json:"entry_price"`
	FillPrice          types.Number  `json:"fill_price"`
	TradeSize          types.Number  `json:"trade_size"`
	PositionSize       types.Number  `json:"position_size"`
}

// CountdownParams represents query parameters for countdown cancel order
type CountdownParams struct {
	Timeout  int64         `json:"timeout"` // In seconds; at least 5, or 0 to cancel the countdown
	Contract currency.Pair `json:"contract,omitzero"`
}

// FuturesTradingFeeResponse holds a futures contract's trading fee rates
type FuturesTradingFeeResponse struct {
	TakerFee types.Number `json:"taker_fee"`
	MakerFee types.Number `json:"maker_fee"`
}

// FuturesBBOOrderRequest holds a futures order priced at a level of the order book
type FuturesBBOOrderRequest struct {
	Contract                  currency.Pair `json:"contract"`
	Size                      int64         `json:"size"`      // positive buys, negative sells, zero closes a position
	Direction                 string        `json:"direction"` // sell prices against the bids, buy against the asks
	Iceberg                   int64         `json:"iceberg,omitempty"`
	Level                     uint64        `json:"level"` // order book level to price at, from 1 to 20
	ClosePosition             bool          `json:"close,omitempty"`
	ReduceOnly                bool          `json:"reduce_only,omitempty"`
	TimeInForce               string        `json:"tif,omitempty"`
	Text                      string        `json:"text,omitempty"`
	AutoSize                  string        `json:"auto_size,omitempty"`
	SelfTradePreventionAction string        `json:"stp_act,omitempty"`
	PositionID                int64         `json:"pid,omitempty"`
}

// CreateTrailOrderRequest holds a futures trailing stop order creation request
type CreateTrailOrderRequest struct {
	Contract           currency.Pair `json:"contract"`
	Amount             types.Number  `json:"amount"` // positive buys, negative sells
	ActivationPrice    types.Number  `json:"activation_price,omitempty"`
	IsGreaterThanEqual bool          `json:"is_gte"`
	PriceType          uint8         `json:"price_type,omitempty"`   // 1: latest price, 2: index price, 3: mark price
	PriceOffset        string        `json:"price_offset,omitempty"` // a callback ratio such as 0.1% or a price distance such as 0.1
	ReduceOnly         bool          `json:"reduce_only,omitempty"`
	PositionRelated    bool          `json:"position_related,omitempty"`
	Text               string        `json:"text,omitempty"`
	PositionMarginMode string        `json:"pos_margin_mode,omitempty"`
	PositionMode       string        `json:"position_mode,omitempty"`
}

// CreateTrailOrderResponse holds the ID of a newly created trail order
type CreateTrailOrderResponse struct {
	ID NumericString `json:"id"`
}

// StopTrailOrderRequest identifies a trail order to stop by its ID or custom text
type StopTrailOrderRequest struct {
	ID   uint64 `json:"id,omitempty"`
	Text string `json:"text,omitempty"`
}

// StopAllTrailOrdersRequest holds the filters for stopping trail orders in batch
type StopAllTrailOrdersRequest struct {
	Contract        currency.Pair `json:"contract,omitzero"`
	RelatedPosition uint8         `json:"related_position,omitempty"` // 1: long, 2: short
}

// TrailOrder represents a futures trailing stop order
type TrailOrder struct {
	ID                 NumericString `json:"id"`
	UserID             NumericString `json:"user_id"`
	User               NumericString `json:"user"`
	Contract           currency.Pair `json:"contract"`
	Settle             string        `json:"settle"`
	Amount             types.Number  `json:"amount"`
	IsGreaterThanEqual bool          `json:"is_gte"`
	ActivationPrice    types.Number  `json:"activation_price"`
	PriceType          uint8         `json:"price_type"`
	PriceOffset        string        `json:"price_offset"`
	Text               string        `json:"text"`
	ReduceOnly         bool          `json:"reduce_only"`
	PositionRelated    bool          `json:"position_related"`
	CreatedAt          types.Time    `json:"created_at"`
	ActivatedAt        types.Time    `json:"activated_at"`
	FinishedAt         types.Time    `json:"finished_at"`
	CreateTime         types.Time    `json:"create_time"`
	ActiveTime         types.Time    `json:"active_time"`
	FinishTime         types.Time    `json:"finish_time"`
	Reason             string        `json:"reason"`
	SuborderText       string        `json:"suborder_text"`
	IsDualMode         bool          `json:"is_dual_mode"`
	TriggerPrice       types.Number  `json:"trigger_price"`
	SuborderID         NumericString `json:"suborder_id"`
	SideLabel          string        `json:"side_label"`
	OriginalStatus     int64         `json:"original_status"`
	Status             string        `json:"status"`
	PositionSideOutput string        `json:"position_side_output"`
	UpdatedAt          types.Time    `json:"updated_at"`
	ExtremumPrice      types.Number  `json:"extremum_price"`
	StatusCode         string        `json:"status_code"`
	CreatedAtPrecise   types.Time    `json:"created_at_precise"`
	FinishedAtPrecise  types.Time    `json:"finished_at_precise"`
	ActivatedAtPrecise types.Time    `json:"activated_at_precise"`
	StatusLabel        string        `json:"status_label"`
	PositionMarginMode string        `json:"pos_margin_mode"`
	PositionMode       string        `json:"position_mode"`
	ErrorLabel         string        `json:"error_label"`
	Leverage           types.Number  `json:"leverage"`
}

// trailOrderReply decodes a single trail order reply, which the schema nests under "order" while the documented
// examples send the order itself
type trailOrderReply struct {
	Order *TrailOrder
}

// UnmarshalJSON implements json.Unmarshaler
func (t *trailOrderReply) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var nested struct {
		Order *TrailOrder `json:"order"`
	}
	if err := json.Unmarshal(data, &nested); err != nil {
		return err
	}
	if nested.Order != nil {
		t.Order = nested.Order
		return nil
	}
	t.Order = new(TrailOrder)
	return json.Unmarshal(data, t.Order)
}

// GetTrailOrdersRequest holds the filters for listing futures trail orders
type GetTrailOrdersRequest struct {
	Contract        currency.Pair
	IsFinished      bool
	StartAt         time.Time
	EndAt           time.Time
	PageNumber      uint64
	PageSize        uint64
	SortBy          uint8 // 1: creation time, 2: end time
	HideCancelled   bool
	RelatedPosition uint8 // 1: long, 2: short
	SortByTrigger   bool
	ReduceOnly      uint8 // 1: yes, 2: no
	Side            uint8 // 1: long, 2: short
}

// UpdateTrailOrderRequest holds a futures trail order amendment, where empty values leave a parameter unchanged
type UpdateTrailOrderRequest struct {
	ID                 uint64       `json:"id"`
	Amount             types.Number `json:"amount,omitempty"`
	ActivationPrice    string       `json:"activation_price,omitempty"` // "0" triggers immediately
	IsGreaterThanEqual string       `json:"is_gte_str,omitempty"`       // "true" or "false"
	PriceType          uint8        `json:"price_type,omitempty"`
	PriceOffset        string       `json:"price_offset,omitempty"`
}

// TrailOrderChange represents a creation or modification of a futures trail order
type TrailOrderChange struct {
	UpdatedAt          types.Time   `json:"updated_at"`
	Amount             types.Number `json:"amount"`
	IsGreaterThanEqual bool         `json:"is_gte"`
	ActivationPrice    types.Number `json:"activation_price"`
	PriceType          uint8        `json:"price_type"`
	PriceOffset        string       `json:"price_offset"`
	IsCreate           bool         `json:"is_create"`
}

// CreateChaseOrderRequest represents a chase limit order creation request.
type CreateChaseOrderRequest struct {
	Contract           currency.Pair `json:"contract"`
	Amount             types.Number  `json:"amount"`                 // positive buys, negative sells
	PriceLimit         types.Number  `json:"price_limit"`            // the furthest price to chase to; zero sets no limit
	OffsetLimit        types.Number  `json:"offset_limit,omitempty"` // the furthest distance to chase from the best price; exclusive of PriceLimit
	ReduceOnly         bool          `json:"reduce_only,omitempty"`
	Text               string        `json:"text,omitempty"`
	IsDualMode         bool          `json:"is_dual_mode,omitempty"`
	PriceType          int64         `json:"price_type,omitempty"`     // 1: best bid or ask, 2: a distance from the best bid or ask
	PriceGapType       int64         `json:"price_gap_type,omitempty"` // 1: absolute, 2: percentage; used with a PriceType of 2
	PriceGapValue      types.Number  `json:"price_gap_value,omitempty"`
	PositionMarginMode string        `json:"pos_margin_mode,omitempty"`
	PositionMode       string        `json:"position_mode,omitempty"`
	Settle             currency.Code `json:"-"`
}

// MarshalJSON sends the price limit as a decimal string even when zero, since Gate requires the field and reads
// "0" as no limit, whereas types.Number marshals zero as an empty string
func (c CreateChaseOrderRequest) MarshalJSON() ([]byte, error) { //nolint:gocritic // hugeParam: a value receiver lets json call this for unaddressable values, such as requests marshalled by value
	type alias CreateChaseOrderRequest
	return json.Marshal(&struct {
		Contract   currency.Pair `json:"contract"`
		Amount     types.Number  `json:"amount"`
		PriceLimit string        `json:"price_limit"`
		*alias
	}{
		Contract:   c.Contract,
		Amount:     c.Amount,
		PriceLimit: strconv.FormatFloat(c.PriceLimit.Float64(), 'f', -1, 64),
		alias:      (*alias)(&c),
	})
}

// CreateChaseOrderResponse holds the ID of a newly created chase order
type CreateChaseOrderResponse struct {
	ID NumericString `json:"id"`
}

// StopChaseOrderRequest identifies a running chase order to stop by its ID or custom text
type StopChaseOrderRequest struct {
	ID   string `json:"id,omitempty"`
	Text string `json:"text,omitempty"`
}

// StopAllChaseOrdersRequest holds the filters for stopping chase orders in batch
type StopAllChaseOrdersRequest struct {
	Contract           currency.Pair `json:"contract,omitzero"`
	PositionMarginMode string        `json:"pos_margin_mode,omitempty"`
}

// GetChaseOrdersRequest holds the filters for listing futures chase orders
type GetChaseOrdersRequest struct {
	Contract      currency.Pair
	IsFinished    bool
	StartAt       time.Time // required when IsFinished is set
	EndAt         time.Time // required when IsFinished is set
	PageNumber    uint64
	PageSize      uint64 // 1 to 100
	SortBy        uint8  // 1: creation time, 2: finish time
	HideCancelled bool
	ReduceOnly    uint8 // 1: reduce only orders, 2: other orders
	Side          uint8 // 1: long, 2: short
}

// ChaseOrder represents a chase limit order detail or list item.
type ChaseOrder struct {
	ID                 NumericString `json:"id"`
	User               NumericString `json:"user"`
	Contract           currency.Pair `json:"contract"`
	Settle             string        `json:"settle"`
	Amount             types.Number  `json:"amount"`
	PriceLimit         types.Number  `json:"price_limit"`
	ReduceOnly         bool          `json:"reduce_only"`
	Text               string        `json:"text"`
	CreateTime         types.Time    `json:"create_time"`
	FinishTime         types.Time    `json:"finish_time"`
	OriginalStatus     int64         `json:"original_status"`
	Status             string        `json:"status"`
	Reason             string        `json:"reason"`
	FillAmount         types.Number  `json:"fill_amount"`
	AverageFillPrice   types.Number  `json:"average_fill_price"`
	SuborderID         NumericString `json:"suborder_id"`
	IsDualMode         bool          `json:"is_dual_mode"`
	SideLabel          string        `json:"side_label"`
	PositionSideOutput string        `json:"position_side_output"`
	ChasePrice         types.Number  `json:"chase_price"`
	IntervalSeconds    uint64        `json:"interval_sec"`
	UpdatedAt          types.Time    `json:"updated_at"`
	SuborderPrice      types.Number  `json:"suborder_price"`
	SuborderOngoing    bool          `json:"suborder_ongoing"`
	SuborderFinishAs   string        `json:"suborder_finish_as"`
	PriceType          int64         `json:"price_type"`
	PriceGapType       NumericString `json:"price_gap_type"`
	PriceGapValue      types.Number  `json:"price_gap_value"`
	StatusCode         string        `json:"status_code"`
	CreateTimePrecise  types.Time    `json:"create_time_precise"`
	FinishTimePrecise  types.Time    `json:"finish_time_precise"`
	PositionMarginMode string        `json:"pos_margin_mode"`
	PositionMode       string        `json:"position_mode"`
	Leverage           types.Number  `json:"leverage"`
	ErrorLabel         string        `json:"error_label"`
}

// FuturesPriceTriggeredOrderParam represents a creates a price triggered order
type FuturesPriceTriggeredOrderParam struct {
	Initial            FuturesInitial `json:"initial"`
	Trigger            FuturesTrigger `json:"trigger"`
	OrderType          string         `json:"order_type,omitempty"`
	PositionMarginMode string         `json:"pos_margin_mode,omitempty"`
}

// FuturesInitial represents a price triggered order initial parameters
type FuturesInitial struct {
	Contract    currency.Pair `json:"contract"`
	Size        int64         `json:"size"`             // Order size. Positive size means to buy, while negative one means to sell. Set to 0 to close the position
	Amount      types.Number  `json:"amount,omitempty"` // A decimal size, which takes precedence over Size
	Price       types.Number  `json:"price"`            // Order price. Set to 0 to use market price
	Close       bool          `json:"close,omitempty"`
	TimeInForce string        `json:"tif,omitempty"`
	Text        string        `json:"text,omitempty"`
	ReduceOnly  bool          `json:"reduce_only,omitempty"`
	AutoSize    string        `json:"auto_size,omitempty"`
}

// MarshalJSON sends the price as a decimal string even when zero, since Gate reads a zero price as a market
// order, whereas types.Number marshals zero as an empty string
func (f FuturesInitial) MarshalJSON() ([]byte, error) { //nolint:gocritic // hugeParam: a value receiver lets json call this for unaddressable values, such as orders marshalled by value
	type alias FuturesInitial
	return json.Marshal(&struct {
		Contract currency.Pair `json:"contract"`
		Size     int64         `json:"size"`
		Amount   types.Number  `json:"amount,omitempty"`
		Price    string        `json:"price"`
		*alias
	}{
		Contract: f.Contract,
		Size:     f.Size,
		Amount:   f.Amount,
		Price:    strconv.FormatFloat(f.Price.Float64(), 'f', -1, 64),
		alias:    (*alias)(&f),
	})
}

// FuturesTrigger represents a price triggered order trigger parameter
type FuturesTrigger struct {
	StrategyType int64        `json:"strategy_type,omitempty"` // How the order will be triggered 0: by price, which means the order will be triggered if price condition is satisfied 1: by price gap, which means the order will be triggered if gap of recent two prices of specified price_type are satisfied. Only 0 is supported currently
	PriceType    int64        `json:"price_type,omitempty"`
	Price        types.Number `json:"price,omitempty"`
	Rule         int64        `json:"rule,omitempty"`
	Expiration   int64        `json:"expiration,omitempty"` // how long(in seconds) to wait for the condition to be triggered before cancelling the order
}

// PriceTriggeredOrder represents a future triggered price order response
type PriceTriggeredOrder struct {
	Initial            PriceTriggeredOrderInitial `json:"initial"`
	Trigger            PriceTriggeredOrderTrigger `json:"trigger"`
	ID                 int64                      `json:"id"`
	IDString           string                     `json:"id_string"`
	User               int64                      `json:"user"`
	CreateTime         types.Time                 `json:"create_time"`
	FinishTime         types.Time                 `json:"finish_time"`
	TradeID            int64                      `json:"trade_id"`
	Status             string                     `json:"status"`
	FinishAs           string                     `json:"finish_as"`
	Reason             string                     `json:"reason"`
	OrderType          string                     `json:"order_type"`
	MatchingOrderID    int64                      `json:"me_order_id"`
	PositionMarginMode string                     `json:"pos_margin_mode"`
}

// PriceTriggeredOrderInitial holds the order a price triggered order places once triggered
type PriceTriggeredOrderInitial struct {
	Contract     currency.Pair `json:"contract"`
	Size         types.Number  `json:"size"`
	Amount       types.Number  `json:"amount"`
	Price        types.Number  `json:"price"`
	TimeInForce  string        `json:"tif"`
	Text         string        `json:"text"`
	ReduceOnly   bool          `json:"reduce_only"`
	IsReduceOnly bool          `json:"is_reduce_only"`
	IsClose      bool          `json:"is_close"`
}

// PriceTriggeredOrderTrigger holds the condition that triggers a price triggered order
type PriceTriggeredOrderTrigger struct {
	StrategyType int64        `json:"strategy_type"`
	PriceType    int64        `json:"price_type"`
	Price        types.Number `json:"price"`
	Rule         int64        `json:"rule"`
	Expiration   int64        `json:"expiration"`
}

// AmendFuturesPriceTriggeredOrderRequest holds a futures price triggered order amendment
type AmendFuturesPriceTriggeredOrderRequest struct {
	OrderID      uint64       `json:"order_id"`
	Size         *int64       `json:"size,omitempty"`   // 0 fully closes the position
	Amount       types.Number `json:"amount,omitempty"` // A decimal size, which takes precedence over Size
	Price        *float64     `json:"price,omitempty"`  // 0 places a market order once triggered
	TriggerPrice types.Number `json:"trigger_price,omitempty"`
	PriceType    *int64       `json:"price_type,omitempty"` // 0: latest trade price, 1: mark price, 2: index price
	AutoSize     string       `json:"auto_size,omitempty"`
	Close        bool         `json:"close,omitempty"`
}

// MarshalJSON sends the price as the documented decimal string, keeping an explicit zero so that a market
// order can be requested
func (a AmendFuturesPriceTriggeredOrderRequest) MarshalJSON() ([]byte, error) {
	type alias AmendFuturesPriceTriggeredOrderRequest
	var price *string
	if a.Price != nil {
		formatted := strconv.FormatFloat(*a.Price, 'f', -1, 64)
		price = &formatted
	}
	return json.Marshal(&struct {
		OrderID uint64       `json:"order_id"`
		Size    *int64       `json:"size,omitempty"`
		Amount  types.Number `json:"amount,omitempty"`
		Price   *string      `json:"price,omitempty"`
		*alias
	}{
		OrderID: a.OrderID,
		Size:    a.Size,
		Amount:  a.Amount,
		Price:   price,
		alias:   (*alias)(&a),
	})
}

// SettlementHistoryItem represents a settlement history item
type SettlementHistoryItem struct {
	Time        types.Time    `json:"time"`
	Contract    currency.Pair `json:"contract"`
	Size        int64         `json:"size"`
	Leverage    types.Number  `json:"leverage"`
	Margin      types.Number  `json:"margin"`
	EntryPrice  types.Number  `json:"entry_price"`
	SettlePrice types.Number  `json:"settle_price"`
	Profit      types.Number  `json:"profit"`
	Fee         types.Number  `json:"fee"`
}

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
	} `json:"error"`
}

// WSResponse represents generalised websocket push data from the server.
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
	High24Hour       types.Number  `json:"high_24h"`
	Low24Hour        types.Number  `json:"low_24h"`
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
	UpdateTime    types.Time                       `json:"t"`
	Pair          currency.Pair                    `json:"s"`
	FirstUpdateID int64                            `json:"U"` // First update order book id in this event since last update
	LastUpdateID  int64                            `json:"u"`
	Bids          orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks          orderbook.LevelsArrayPriceAmount `json:"a"`
}

// WsOrderbookUpdateWithSnapshot represents websocket orderbook update push data
type WsOrderbookUpdateWithSnapshot struct {
	UpdateTime    types.Time                       `json:"t"`
	Full          bool                             `json:"full"`
	Channel       string                           `json:"s"`
	FirstUpdateID int64                            `json:"U"`
	LastUpdateID  int64                            `json:"u"`
	Bids          orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks          orderbook.LevelsArrayPriceAmount `json:"a"`
}

// WsOrderbookSnapshot represents a websocket orderbook snapshot push data
type WsOrderbookSnapshot struct {
	UpdateTime   types.Time                       `json:"t"`
	LastUpdateID int64                            `json:"lastUpdateId"`
	CurrencyPair currency.Pair                    `json:"s"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
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
	PointFee           types.Number  `json:"point_fee,omitempty"`
	GTFee              types.Number  `json:"gt_fee,omitempty"`
	GTDiscount         bool          `json:"gt_discount,omitempty"`
	RebatedFee         types.Number  `json:"rebated_fee,omitempty"`
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
	GTFee        types.Number  `json:"gt_fee"`
	Text         string        `json:"text"`
}

// WsSpotBalance represents a spot balance.
type WsSpotBalance struct {
	Timestamp    types.Time    `json:"timestamp_ms"`
	User         string        `json:"user"`
	Currency     currency.Code `json:"currency"`
	Change       types.Number  `json:"change"`
	Total        types.Number  `json:"total"`
	Available    types.Number  `json:"available"`
	Freeze       types.Number  `json:"freeze"`
	FreezeChange types.Number  `json:"freeze_change"`
	ChangeType   string        `json:"change_type"` // e.g. "order-create", "order-match"
}

// WsMarginBalance represents margin account balance push data
type WsMarginBalance struct {
	Timestamp    types.Time    `json:"timestamp_ms"`
	User         string        `json:"user"`
	CurrencyPair string        `json:"currency_pair"`
	Currency     currency.Code `json:"currency"`
	Change       types.Number  `json:"change"`
	Available    types.Number  `json:"available"`
	Freeze       types.Number  `json:"freeze"`
	Borrowed     types.Number  `json:"borrowed"`
	Interest     types.Number  `json:"interest"`
}

// WsFundingBalance represents funding balance push data.
type WsFundingBalance struct {
	Timestamp types.Time    `json:"timestamp_ms"`
	User      string        `json:"user"`
	Currency  currency.Code `json:"currency"`
	Change    types.Number  `json:"change"`
	Freeze    types.Number  `json:"freeze"`
	Lent      types.Number  `json:"lent"`
}

// WsCrossMarginBalance represents a cross margin balance detail
type WsCrossMarginBalance struct {
	Timestamp types.Time    `json:"timestamp_ms"`
	User      string        `json:"user"`
	Currency  currency.Code `json:"currency"`
	Change    types.Number  `json:"change"`
	Total     types.Number  `json:"total"`
	Available types.Number  `json:"available"`
}

// WsCrossMarginLoan represents a cross margin loan push data
type WsCrossMarginLoan struct {
	Timestamp types.Time    `json:"timestamp"`
	User      string        `json:"user"`
	Currency  currency.Code `json:"currency"`
	Change    types.Number  `json:"change"`
	Total     types.Number  `json:"total"`
	Available types.Number  `json:"available"`
	Borrowed  types.Number  `json:"borrowed"`
	Interest  types.Number  `json:"interest"`
}

// WsFutureTicker represents a futures push data.
type WsFutureTicker struct {
	Contract              currency.Pair `json:"contract"`
	Last                  types.Number  `json:"last"`
	ChangePercentage      types.Number  `json:"change_percentage"`
	FundingRate           types.Number  `json:"funding_rate"`
	FundingRateIndicative types.Number  `json:"funding_rate_indicative"`
	MarkPrice             types.Number  `json:"mark_price"`
	IndexPrice            types.Number  `json:"index_price"`
	TotalSize             types.Number  `json:"total_size"`
	Volume24Hour          types.Number  `json:"volume_24h"`
	Volume24HourBTC       types.Number  `json:"volume_24h_btc"`
	Volume24HourUSD       types.Number  `json:"volume_24h_usd"`
	QuantoBaseRate        types.Number  `json:"quanto_base_rate"`
	Volume24HourQuote     types.Number  `json:"volume_24h_quote"`
	Volume24HourSettle    string        `json:"volume_24h_settle"`
	Volume24HourBase      types.Number  `json:"volume_24h_base"`
	Low24Hour             types.Number  `json:"low_24h"`
	High24Hour            types.Number  `json:"high_24h"`
}

// WsFuturesTrades represents a list of trades push data
type WsFuturesTrades struct {
	Size       types.Number  `json:"size"`
	ID         int64         `json:"id"`
	CreateTime types.Time    `json:"create_time_ms"`
	Price      types.Number  `json:"price"`
	Contract   currency.Pair `json:"contract"`
	IsInternal bool          `json:"is_internal"` //
}

// WsFuturesOrderbookTicker represents the orderbook ticker push data
type WsFuturesOrderbookTicker struct {
	Timestamp     types.Time   `json:"t"`
	UpdateID      int64        `json:"u"`
	CurrencyPair  string       `json:"s"`
	BestBidPrice  types.Number `json:"b"`
	BestBidAmount types.Number `json:"B"`
	BestAskPrice  types.Number `json:"a"`
	BestAskAmount types.Number `json:"A"`
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
	Size  types.Number `json:"s"`
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
	Amount       types.Number `json:"s"`
	CurrencyPair string       `json:"c"`
	ID           int64        `json:"id"`
}

// WsFuturesUserTrade represents a futures account user trade push data
type WsFuturesUserTrade struct {
	ID         string        `json:"id"`
	CreateTime types.Time    `json:"create_time_ms"`
	Contract   currency.Pair `json:"contract"`
	OrderID    string        `json:"order_id"`
	Size       types.Number  `json:"size"`
	Price      types.Number  `json:"price"`
	Role       string        `json:"role"`
	Text       string        `json:"text"`
	Fee        types.Number  `json:"fee"`
	PointFee   int64         `json:"point_fee"`
}

// WsFuturesLiquidationNotification represents a liquidation notification push data
type WsFuturesLiquidationNotification struct {
	EntryPrice        types.Number  `json:"entry_price"`
	FillPrice         types.Number  `json:"fill_price"`
	Left              types.Number  `json:"left"`
	Leverage          types.Number  `json:"leverage"`
	LiquidiationPrice types.Number  `json:"liq_price"`
	Margin            types.Number  `json:"margin"`
	MarkPrice         types.Number  `json:"mark_price"`
	OrderID           int64         `json:"order_id"`
	OrderPrice        types.Number  `json:"order_price"`
	Size              types.Number  `json:"size"`
	Time              types.Time    `json:"time_ms"`
	Contract          currency.Pair `json:"contract"`
	User              string        `json:"user"`
}

// WsFuturesAutoDeleveragesNotification represents futures auto deleverages push data
type WsFuturesAutoDeleveragesNotification struct {
	EntryPrice   types.Number  `json:"entry_price"`
	FillPrice    types.Number  `json:"fill_price"`
	PositionSize int64         `json:"position_size"`
	TradeSize    int64         `json:"trade_size"`
	Time         types.Time    `json:"time_ms"`
	Contract     currency.Pair `json:"contract"`
	User         string        `json:"user"`
}

// WsPositionClose represents a close position futures push data
type WsPositionClose struct {
	Contract      currency.Pair `json:"contract"`
	ProfitAndLoss types.Number  `json:"pnl,omitempty"`
	Side          string        `json:"side"`
	Text          string        `json:"text"`
	Time          types.Time    `json:"time_ms"`
	User          string        `json:"user"`

	// Added in options close position push datas
	SettleSize types.Number `json:"settle_size,omitempty"`
	Underlying string       `json:"underlying,omitempty"`
}

// WsBalance represents a options and futures balance push data
type WsBalance struct {
	Balance  types.Number  `json:"balance"`
	Change   types.Number  `json:"change"`
	Currency currency.Code `json:"currency"`
	Text     string        `json:"text"`
	Time     types.Time    `json:"time_ms"`
	Type     string        `json:"type"`
	User     string        `json:"user"`
}

// WsFuturesReduceRiskLimitNotification represents a futures reduced risk limit push data
type WsFuturesReduceRiskLimitNotification struct {
	CancelOrders    int64        `json:"cancel_orders"`
	Contract        string       `json:"contract"`
	LeverageMax     int64        `json:"leverage_max"`
	LiqPrice        types.Number `json:"liq_price"`
	MaintenanceRate types.Number `json:"maintenance_rate"`
	RiskLimit       int64        `json:"risk_limit"`
	Time            types.Time   `json:"time_ms"`
	User            string       `json:"user"`
}

// WsFuturesPosition represents futures notify positions update.
type WsFuturesPosition struct {
	Contract           string       `json:"contract"`
	CrossLeverageLimit types.Number `json:"cross_leverage_limit"`
	EntryPrice         types.Number `json:"entry_price"`
	HistoryPnl         types.Number `json:"history_pnl"`
	HistoryPoint       int64        `json:"history_point"`
	LastClosePnl       types.Number `json:"last_close_pnl"`
	Leverage           types.Number `json:"leverage"`
	LeverageMax        types.Number `json:"leverage_max"`
	LiqPrice           types.Number `json:"liq_price"`
	MaintenanceRate    types.Number `json:"maintenance_rate"`
	Margin             types.Number `json:"margin"`
	Mode               string       `json:"mode"`
	RealisedPNL        types.Number `json:"realised_pnl"`
	RealisedPoint      types.Number `json:"realised_point"`
	RiskLimit          types.Number `json:"risk_limit"`
	Size               types.Number `json:"size"`
	TimeSeconds        types.Time   `json:"time"`
	Time               types.Time   `json:"time_ms"`
	UpdateID           int64        `json:"update_id"`
	User               string       `json:"user"`
	PositionMarginMode string       `json:"pos_margin_mode"`
	PositionLeverage   types.Number `json:"lever"`
}

// WsFuturesAutoOrder represents an auto order push data.
type WsFuturesAutoOrder struct {
	User    int64 `json:"user"`
	Trigger struct {
		StrategyType int64        `json:"strategy_type"`
		PriceType    int64        `json:"price_type"`
		Price        types.Number `json:"price"`
		Rule         int64        `json:"rule"`
		Expiration   int64        `json:"expiration"`
	} `json:"trigger"`
	Initial struct {
		Contract     string       `json:"contract"`
		Size         types.Number `json:"size"`
		Price        types.Number `json:"price"`
		TimeInForce  string       `json:"tif"`
		Text         string       `json:"text"`
		Iceberg      types.Number `json:"iceberg"`
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
		Rule         int64        `json:"rule"`
		TriggerPrice types.Number `json:"trigger_price"`
		OrderPrice   types.Number `json:"order_price"`
	} `json:"stop_trigger"`
}

// WsOptionUnderlyingTicker represents options underlying ticker push data
type WsOptionUnderlyingTicker struct {
	TradePut   int64        `json:"trade_put"`
	TradeCall  int64        `json:"trade_call"`
	IndexPrice types.Number `json:"index_price"`
	Name       string       `json:"name"`
}

// WsOptionsTrades represents options trades for websocket push data.
type WsOptionsTrades struct {
	ID         int64         `json:"id"`
	CreateTime types.Time    `json:"create_time_ms"`
	Contract   currency.Pair `json:"contract"`
	Size       types.Number  `json:"size"`
	Price      types.Number  `json:"price"`
	Underlying string        `json:"underlying"`
	IsCall     bool          `json:"is_call"` // added in underlying trades
}

// WsOptionsUnderlyingPrice represents the underlying price.
type WsOptionsUnderlyingPrice struct {
	Underlying string       `json:"underlying"`
	Price      types.Number `json:"price"`
	UpdateTime types.Time   `json:"time_ms"`
}

// WsOptionsMarkPrice represents options mark price push data.
type WsOptionsMarkPrice struct {
	Contract   currency.Pair `json:"contract"`
	Price      types.Number  `json:"price"`
	UpdateTime types.Time    `json:"time_ms"`
}

// WsOptionsSettlement represents a options settlement push data.
type WsOptionsSettlement struct {
	Contract     currency.Pair `json:"contract"`
	OrderbookID  int64         `json:"orderbook_id"`
	PositionSize types.Number  `json:"position_size"`
	Profit       types.Number  `json:"profit"`
	SettlePrice  types.Number  `json:"settle_price"`
	StrikePrice  types.Number  `json:"strike_price"`
	Tag          string        `json:"tag"`
	TradeID      int64         `json:"trade_id"`
	TradeSize    types.Number  `json:"trade_size"`
	Underlying   string        `json:"underlying"`
	UpdateTime   types.Time    `json:"time_ms"`
}

// WsOptionsContract represents an option contract push data.
type WsOptionsContract struct {
	Contract          currency.Pair `json:"contract"`
	CreateTime        types.Time    `json:"create_time"`
	ExpirationTime    types.Time    `json:"expiration_time"`
	InitMarginHigh    types.Number  `json:"init_margin_high"`
	InitMarginLow     types.Number  `json:"init_margin_low"`
	IsCall            bool          `json:"is_call"`
	MaintMarginBase   types.Number  `json:"maint_margin_base"`
	MakerFeeRate      types.Number  `json:"maker_fee_rate"`
	MarkPriceRound    types.Number  `json:"mark_price_round"`
	MinBalanceShort   types.Number  `json:"min_balance_short"`
	MinOrderMargin    types.Number  `json:"min_order_margin"`
	Multiplier        types.Number  `json:"multiplier"`
	OrderPriceDeviate types.Number  `json:"order_price_deviate"`
	OrderPriceRound   types.Number  `json:"order_price_round"`
	OrderSizeMax      types.Number  `json:"order_size_max"`
	OrderSizeMin      types.Number  `json:"order_size_min"`
	OrdersLimit       types.Number  `json:"orders_limit"`
	RefDiscountRate   types.Number  `json:"ref_discount_rate"`
	RefRebateRate     types.Number  `json:"ref_rebate_rate"`
	StrikePrice       types.Number  `json:"strike_price"`
	Tag               string        `json:"tag"`
	TakerFeeRate      types.Number  `json:"taker_fee_rate"`
	Underlying        string        `json:"underlying"`
	Time              types.Time    `json:"time_ms"`
}

// WsOptionsContractCandlestick represents an options contract candlestick push data.
type WsOptionsContractCandlestick struct {
	Timestamp          types.Time   `json:"t"`
	TotalVolume        types.Number `json:"v"`
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
	BidSize         types.Number `json:"B"`
	AskPrice        types.Number `json:"a"`
	AskSize         types.Number `json:"A"`
}

// WsOptionsOrderbookSnapshot represents the options orderbook snapshot push data.
type WsOptionsOrderbookSnapshot struct {
	ID        int64          `json:"id"`
	Timestamp types.Time     `json:"t"`
	Contract  currency.Pair  `json:"contract"`
	Asks      []PriceAndSize `json:"asks"`
	Bids      []PriceAndSize `json:"bids"`
}

// PriceAndSize represents a price and size detail
type PriceAndSize struct {
	Price types.Number `json:"p"`
	Size  types.Number `json:"s"`
}

// WsOptionsOrder represents options order push data.
type WsOptionsOrder struct {
	ID              int64         `json:"id"`
	Contract        currency.Pair `json:"contract"`
	CreateTime      types.Time    `json:"create_time"`
	FillPrice       types.Number  `json:"fill_price"`
	FinishAs        string        `json:"finish_as"`
	Iceberg         types.Number  `json:"iceberg"`
	IsClose         bool          `json:"is_close"`
	IsLiq           bool          `json:"is_liq"`
	IsReduceOnly    bool          `json:"is_reduce_only"`
	RemainingAmount types.Number  `json:"left"`
	Mkfr            types.Number  `json:"mkfr"`
	Price           types.Number  `json:"price"`
	Refr            types.Number  `json:"refr"`
	Refu            types.Number  `json:"refu"`
	Size            types.Number  `json:"size"`
	Status          string        `json:"status"`
	Text            string        `json:"text"`
	Tif             string        `json:"tif"`
	Tkfr            types.Number  `json:"tkfr"`
	Underlying      string        `json:"underlying"`
	User            string        `json:"user"`
	CreationTime    types.Time    `json:"time_ms"`
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
	Size       types.Number  `json:"size"`
}

// WsOptionsLiquidates represents the liquidates push data of option account.
type WsOptionsLiquidates struct {
	User        string       `json:"user"`
	InitMargin  types.Number `json:"init_margin"`
	MaintMargin types.Number `json:"maint_margin"`
	OrderMargin types.Number `json:"order_margin"`
	Time        types.Time   `json:"time_ms"`
}

// WsOptionsUserSettlement represents user's personal settlements push data of options account.
type WsOptionsUserSettlement struct {
	User         string        `json:"user"`
	Contract     currency.Pair `json:"contract"`
	RealisedPnl  types.Number  `json:"realised_pnl"`
	SettlePrice  types.Number  `json:"settle_price"`
	SettleProfit types.Number  `json:"settle_profit"`
	Size         types.Number  `json:"size"`
	StrikePrice  types.Number  `json:"strike_price"`
	Underlying   string        `json:"underlying"`
	SettleTime   types.Time    `json:"time_ms"`
}

// WsOptionsPosition represents positions push data for options account.
type WsOptionsPosition struct {
	EntryPrice  types.Number `json:"entry_price"`
	RealisedPnl types.Number `json:"realised_pnl"`
	Size        types.Number `json:"size"`
	Contract    string       `json:"contract"`
	User        string       `json:"user"`
	UpdateTime  types.Time   `json:"time_ms"`
}

// InterSubAccountTransferParams represents parameters to transfer funds between sub-accounts.
type InterSubAccountTransferParams struct {
	Currency                currency.Code `json:"currency"`
	SubAccountFromUserID    string        `json:"sub_account_from"`
	SubAccountFromAssetType asset.Item    `json:"sub_account_from_type"` // Spot, perpetual futures or delivery futures
	SubAccountToUserID      string        `json:"sub_account_to"`
	SubAccountToAssetType   asset.Item    `json:"sub_account_to_type"` // Spot, perpetual futures or delivery futures
	Amount                  types.Number  `json:"amount"`
}

// MarshalJSON sends the sub-account account types as Gate account names, which differ from asset.Item strings for
// futures assets
func (p *InterSubAccountTransferParams) MarshalJSON() ([]byte, error) {
	from, err := walletAccountType(p.SubAccountFromAssetType)
	if err != nil {
		return nil, fmt.Errorf("%w: source sub-account", err)
	}
	to, err := walletAccountType(p.SubAccountToAssetType)
	if err != nil {
		return nil, fmt.Errorf("%w: destination sub-account", err)
	}
	type Alias InterSubAccountTransferParams
	return json.Marshal(&struct {
		SubAccountFromAssetType string `json:"sub_account_from_type"`
		SubAccountToAssetType   string `json:"sub_account_to_type"`
		*Alias
	}{
		SubAccountFromAssetType: from,
		SubAccountToAssetType:   to,
		Alias:                   (*Alias)(p),
	})
}

// BalanceDetails represents a user's balance details
type BalanceDetails struct {
	Available                  types.Number `json:"available"`
	Freeze                     types.Number `json:"freeze"`
	Borrowed                   types.Number `json:"borrowed"`
	NegativeLiabilities        types.Number `json:"negative_liab"`
	FuturesPositionLiabilities types.Number `json:"futures_pos_liab"`
	Equity                     types.Number `json:"equity"`
	TotalFreeze                types.Number `json:"total_freeze"`
	TotalLiabilities           types.Number `json:"total_liab"`
	SpotInUse                  types.Number `json:"spot_in_use"`
	Funding                    types.Number `json:"funding"`
	FundingVersion             types.Number `json:"funding_version"`
	CrossBalance               types.Number `json:"cross_balance"`
	IsolatedBalance            types.Number `json:"iso_balance"`
	InitialMargin              types.Number `json:"im"`
	MaintenanceMargin          types.Number `json:"mm"`
	InitialMarginRate          types.Number `json:"imr"`
	MaintenanceMarginRate      types.Number `json:"mmr"`
	MarginBalance              types.Number `json:"margin_balance"`
	AvailableMargin            types.Number `json:"available_margin"`
	EnabledCollateral          bool         `json:"enabled_collateral"`
	BalanceVersion             uint64       `json:"balance_version"`
}

// UnifiedUserAccount represents a unified user account
type UnifiedUserAccount struct {
	Mode                           string                           `json:"mode"`
	UserID                         uint64                           `json:"user_id"`
	RefreshTime                    types.Time                       `json:"refresh_time"`
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
	OptionsOrderLoss               types.Number                     `json:"options_order_loss"`
	SpotHedge                      bool                             `json:"spot_hedge"`
	UseFunding                     bool                             `json:"use_funding"`
	IsAllCollateral                bool                             `json:"is_all_collateral"`
}

// AccountDetails represents account detail information
type AccountDetails struct {
	IPWhitelist     []string          `json:"ip_whitelist"`
	CurrencyPairs   []currency.Pair   `json:"currency_pairs"`
	UserID          uint64            `json:"user_id"`
	VIPTier         uint64            `json:"tier"`
	Key             AccountKeyDetails `json:"key"`
	CopyTradingRole uint64            `json:"copy_trading_role"` // 0 - Normal user, 1 - Copy trading leader, 2 - Copy trading follower, 3 - Both leader and follower
}

// AccountKeyDetails holds the mode of the API key in use
type AccountKeyDetails struct {
	Mode uint64 `json:"mode"` // 1 - Classic mode, 2 - Legacy unified mode
}

// AccountMainKey holds a main account API key
type AccountMainKey struct {
	State         uint64               `json:"state"` // 1 - Normal, 2 - Locked, 3 - Frozen
	Mode          uint64               `json:"mode"`  // 1 - Classic mode, 2 - Legacy unified mode
	Name          string               `json:"name"`
	CurrencyPairs []currency.Pair      `json:"currency_pairs"`
	UserID        uint64               `json:"user_id"`
	IPWhitelist   []string             `json:"ip_whitelist"`
	Permissions   []*SubAccountKeyPerm `json:"perms"`
	Key           string               `json:"key"` // Masked API key
	CreatedAt     types.Time           `json:"created_at"`
	// Gate documents updated_at but its example responses send update_at, so both are decoded
	UpdatedAt  types.Time `json:"updated_at"`
	UpdateAt   types.Time `json:"update_at"`
	LastAccess types.Time `json:"last_access"`
}

// UserTransactionRateLimitInfo represents user transaction rate limit information
type UserTransactionRateLimitInfo struct {
	Type      string       `json:"type"`
	Tier      types.Number `json:"tier"`
	Ratio     types.Number `json:"ratio"`
	MainRatio types.Number `json:"main_ratio"`
	UpdatedAt types.Time   `json:"updated_at"`
}

// STPUserGroup holds self-trade prevention user group
type STPUserGroup struct {
	ID         uint64     `json:"id"`
	Name       string     `json:"name"`
	CreatorID  uint64     `json:"creator_id"`
	CreateTime types.Time `json:"create_time"`
}

// STPUserGroupMember holds STP user group member detail
type STPUserGroupMember struct {
	UserID     uint64     `json:"user_id"`
	STPID      uint64     `json:"stp_id"`
	CreateTime types.Time `json:"create_time"`
}

// BorrowOrRepayRequest holds a request parameter for asset borrow and repay requests
type BorrowOrRepayRequest struct {
	Currency  currency.Code `json:"currency"`
	Type      string        `json:"type"` // Type: borrow - "borrow", repay - "repay"
	Amount    types.Number  `json:"amount"`
	RepaidAll bool          `json:"repaid_all,omitempty"` // Repays the whole loan, overriding Amount; repay only
	Text      string        `json:"text,omitempty"`
}

// BorrowOrRepayResponse holds the transaction created by a unified account borrow or repayment
type BorrowOrRepayResponse struct {
	TransactionID uint64 `json:"tran_id"`
}

// MultiCollateralLoanOrderRequest holds a multi-collateral loan order request
type MultiCollateralLoanOrderRequest struct {
	OrderID              string              `json:"order_id,omitempty"`
	OrderType            string              `json:"order_type,omitempty"` // current or fixed, defaulting to current
	FixedType            string              `json:"fixed_type,omitempty"` // 7d or 30d, required for fixed rate orders
	FixedRate            types.Number        `json:"fixed_rate,omitempty"` // Required for fixed rate orders
	AutoRenew            bool                `json:"auto_renew,omitempty"` // Fixed rate orders only
	AutoRepay            bool                `json:"auto_repay,omitempty"` // Fixed rate orders only
	BorrowCurrency       currency.Code       `json:"borrow_currency"`
	BorrowAmount         types.Number        `json:"borrow_amount"`
	CollateralCurrencies []CurrencyAndAmount `json:"collateral_currencies,omitempty"`
}

// MultiCollateralLoanOrderDetail holds a multi-collateral loan order
type MultiCollateralLoanOrderDetail struct {
	OrderID                 string                    `json:"order_id"`
	OrderType               string                    `json:"order_type"`
	FixedType               string                    `json:"fixed_type"`
	FixedRate               types.Number              `json:"fixed_rate"`
	ExpireTime              types.Time                `json:"expire_time"`
	AutoRenew               bool                      `json:"auto_renew"`
	AutoRepay               bool                      `json:"auto_repay"`
	CurrentLTV              types.Number              `json:"current_ltv"`
	Status                  string                    `json:"status"`
	BorrowTime              types.Time                `json:"borrow_time"`
	TotalLeftRepayUSDT      types.Number              `json:"total_left_repay_usdt"`
	TotalLeftCollateralUSDT types.Number              `json:"total_left_collateral_usdt"`
	BorrowCurrencies        []*BorrowCurrencyInfo     `json:"borrow_currencies"`
	CollateralCurrencies    []*CollateralCurrencyInfo `json:"collateral_currencies"`
}

// BorrowCurrencyInfo holds a multi-collateral loan order's outstanding borrowed currency
type BorrowCurrencyInfo struct {
	Currency           currency.Code `json:"currency"`
	IndexPrice         types.Number  `json:"index_price"`
	LeftRepayPrincipal types.Number  `json:"left_repay_principal"`
	LeftRepayInterest  types.Number  `json:"left_repay_interest"`
	LeftRepayUSDT      types.Number  `json:"left_repay_usdt"`
}

// CollateralCurrencyInfo holds a multi-collateral loan order's remaining collateral currency
type CollateralCurrencyInfo struct {
	Currency           currency.Code `json:"currency"`
	IndexPrice         types.Number  `json:"index_price"`
	LeftCollateral     types.Number  `json:"left_collateral"`
	LeftCollateralUSDT types.Number  `json:"left_collateral_usdt"`
}

// MultiCollateralLoanRepaymentRequest holds a multi-collateral loan repayment request
type MultiCollateralLoanRepaymentRequest struct {
	OrderID    uint64                                  `json:"order_id"`
	RepayItems []LoanRepaymentMultiCollateralAssetItem `json:"repay_items"`
}

// LoanRepaymentMultiCollateralAssetItem holds a currency to repay on a multi-collateral loan
type LoanRepaymentMultiCollateralAssetItem struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount,omitempty"` // Required for partial repayments
	// RepaidAll is true for a full repayment; it is always sent as the API requires it
	RepaidAll bool `json:"repaid_all"`
}

// MultiCollateralLoanRepayment holds a multi-collateral loan repayment result
type MultiCollateralLoanRepayment struct {
	OrderID          uint64                           `json:"order_id"`
	RepaidCurrencies []*MultiCollateralRepaidCurrency `json:"repaid_currencies"`
}

// MultiCollateralRepaidCurrency holds a currency's multi-collateral loan repayment result
type MultiCollateralRepaidCurrency struct {
	Succeeded       bool          `json:"succeeded"`
	Label           string        `json:"label"`   // Error label of a failed repayment
	Message         string        `json:"message"` // Error description of a failed repayment
	Currency        currency.Code `json:"currency"`
	RepaidPrincipal types.Number  `json:"repaid_principal"`
	RepaidInterest  types.Number  `json:"repaid_interest"`
}

// MultiCollateralRepaymentRecordsRequest holds multi-collateral loan repayment record filters
type MultiCollateralRepaymentRecordsRequest struct {
	Type           string // repay or liquidate
	BorrowCurrency currency.Code
	Page           uint64
	Limit          uint64
	From           time.Time
	To             time.Time
}

// MultiCurrencyCollateralRepayment holds a multi-collateral loan repayment record
type MultiCurrencyCollateralRepayment struct {
	OrderID               uint64                       `json:"order_id"`
	RecordID              uint64                       `json:"record_id"`
	InitLTV               types.Number                 `json:"init_ltv"`
	BeforeLTV             types.Number                 `json:"before_ltv"`
	AfterLTV              types.Number                 `json:"after_ltv"`
	BorrowTime            types.Time                   `json:"borrow_time"`
	RepayTime             types.Time                   `json:"repay_time"`
	BorrowCurrencies      []*CurrencyDetail            `json:"borrow_currencies"`
	CollateralCurrencies  []*CurrencyDetail            `json:"collateral_currencies"`
	RepaidCurrencies      []*RepayRecordRepaidCurrency `json:"repaid_currencies"`
	TotalInterestList     []*RepayRecordTotalInterest  `json:"total_interest_list"`
	LeftRepayInterestList []*CurrencyDetail            `json:"left_repay_interest_list"`
}

// RepayRecordRepaidCurrency holds a currency repaid by a multi-collateral loan repayment
type RepayRecordRepaidCurrency struct {
	Currency         currency.Code `json:"currency"`
	IndexPrice       types.Number  `json:"index_price"`
	RepaidAmount     types.Number  `json:"repaid_amount"`
	RepaidPrincipal  types.Number  `json:"repaid_principal"`
	RepaidInterest   types.Number  `json:"repaid_interest"`
	RepaidAmountUSDT types.Number  `json:"repaid_amount_usdt"`
}

// RepayRecordTotalInterest holds a currency's total interest on a multi-collateral loan
type RepayRecordTotalInterest struct {
	Currency   currency.Code `json:"currency"`
	IndexPrice types.Number  `json:"index_price"`
	Amount     types.Number  `json:"amount"`
	AmountUSDT types.Number  `json:"amount_usdt"`
}

// AddOrWithdrawCollateralRequest holds a request to add or withdraw multi-collateral loan collateral
type AddOrWithdrawCollateralRequest struct {
	OrderID       uint64              `json:"order_id"`
	OperationType string              `json:"type"` // append to add collateral or redeem to withdraw it
	Collaterals   []CurrencyAndAmount `json:"collaterals,omitempty"`
}

// CollateralAddOrRemoveResponse holds the result of adding or withdrawing multi-collateral loan collateral
type CollateralAddOrRemoveResponse struct {
	OrderID              uint64                       `json:"order_id"`
	CollateralCurrencies []*CollateralOperationResult `json:"collateral_currencies"`
}

// CollateralOperationResult holds a currency's collateral adjustment result
type CollateralOperationResult struct {
	Succeeded bool          `json:"succeeded"`
	Label     string        `json:"label"`   // Error label of a failed adjustment
	Message   string        `json:"message"` // Error description of a failed adjustment
	Currency  currency.Code `json:"currency"`
	Amount    types.Number  `json:"amount"` // 0 when the adjustment failed
}

// UnifiedLoansRequest holds the filters and paging of a unified account loan or loan record query
type UnifiedLoansRequest struct {
	Currency currency.Code
	Type     string // "platform" or "margin" for loans, "borrow" or "repay" for loan records
	Page     uint64
	Limit    uint64
}

// UnifiedLoanResponse holds an outstanding unified account loan
type UnifiedLoanResponse struct {
	Currency     currency.Code `json:"currency"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Amount       types.Number  `json:"amount"`
	Type         string        `json:"type"` // "platform" or "margin"
	CreateTime   types.Time    `json:"create_time"`
	UpdateTime   types.Time    `json:"update_time"`
}

// UnifiedLoanRecordResponse holds a unified account borrow or repayment record
type UnifiedLoanRecordResponse struct {
	ID            uint64        `json:"id"`
	Type          string        `json:"type"` // "borrow" or "repay"
	RepaymentType string        `json:"repayment_type"`
	BorrowType    string        `json:"borrow_type"`
	MarginMode    string        `json:"margin_mode"`
	CurrencyPair  currency.Pair `json:"currency_pair"`
	Currency      currency.Code `json:"currency"`
	Amount        types.Number  `json:"amount"`
	CreateTime    types.Time    `json:"create_time"`
}

// UnifiedInterestRecordsRequest holds the filters and paging of a unified account interest deduction record query
type UnifiedInterestRecordsRequest struct {
	Currency  currency.Code
	Type      string // "platform" or "margin", which Gate defaults to
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	Limit     uint64
}

// InterestDeductionRecord holds an interest deduction record
type InterestDeductionRecord struct {
	Status       uint64        `json:"status"` // 0: failed, 1: succeeded
	CurrencyPair currency.Pair `json:"currency_pair"`
	Currency     currency.Code `json:"currency"`
	ActualRate   types.Number  `json:"actual_rate"`
	Interest     types.Number  `json:"interest"`
	LoanType     string        `json:"type"`
	CreateTime   types.Time    `json:"create_time"`
}

// UnifiedAccountModeRequest holds a request parameter for setting a unified account mode
type UnifiedAccountModeRequest struct {
	Mode     string                             `json:"mode"`
	Settings *UnifiedAccountModeSettingsRequest `json:"settings,omitempty"`
}

// UnifiedAccountModeSettingsRequest holds the unified account mode switches to change; nil switches are left unchanged
// so that a switch can be explicitly turned off
type UnifiedAccountModeSettingsRequest struct {
	USDTFutures *bool `json:"usdt_futures,omitempty"`
	SpotHedge   *bool `json:"spot_hedge,omitempty"`
	UseFunding  *bool `json:"use_funding,omitempty"`
	Options     *bool `json:"options,omitempty"`
}

// UnifiedAccountModeResponse holds the unified account mode and its switches
type UnifiedAccountModeResponse struct {
	Mode     string                     `json:"mode"`
	Settings UnifiedAccountModeSettings `json:"settings"`
}

// UnifiedAccountModeSettings holds the unified account mode switches
type UnifiedAccountModeSettings struct {
	USDTFutures bool `json:"usdt_futures"`
	SpotHedge   bool `json:"spot_hedge"`
	UseFunding  bool `json:"use_funding"`
	Options     bool `json:"options"`
}

// TierUpperLimit is a tier's upper boundary. Gate reports the unbounded final tier as "+" or an empty string, which
// decodes to positive infinity.
type TierUpperLimit float64

// UnmarshalJSON decodes a tier upper boundary
func (t *TierUpperLimit) UnmarshalJSON(data []byte) error {
	if s := string(data); s == `"+"` || s == `""` {
		*t = TierUpperLimit(math.Inf(1))
		return nil
	}
	var limit types.Number
	if err := limit.UnmarshalJSON(data); err != nil {
		return err
	}
	*t = TierUpperLimit(limit)
	return nil
}

// MarshalJSON encodes a tier upper boundary, using Gate's "+" for an unbounded tier
func (t TierUpperLimit) MarshalJSON() ([]byte, error) {
	if math.IsInf(float64(t), 1) {
		return []byte(`"+"`), nil
	}
	return types.Number(t).MarshalJSON()
}

// Float64 returns the boundary as a float64
func (t TierUpperLimit) Float64() float64 {
	return float64(t)
}

// UnifiedAccountTieredDetail unified account tiered list
type UnifiedAccountTieredDetail struct {
	Currency      currency.Code          `json:"currency"`
	DiscountTiers []*UnifiedDiscountTier `json:"discount_tiers"`
}

// UnifiedDiscountTier holds a unified account collateral discount tier
type UnifiedDiscountTier struct {
	Tier       string         `json:"tier"`
	Discount   types.Number   `json:"discount"`
	LowerLimit types.Number   `json:"lower_limit"`
	UpperLimit TierUpperLimit `json:"upper_limit"`
	Leverage   types.Number   `json:"leverage"`
}

// UnifiedAccountLoanMargin holds unified account loan margin tiered detail
type UnifiedAccountLoanMargin struct {
	Currency    currency.Code        `json:"currency"`
	MarginTiers []*UnifiedMarginTier `json:"margin_tiers"`
}

// UnifiedMarginTier holds a unified account borrowing margin tier
type UnifiedMarginTier struct {
	Tier       string         `json:"tier"`
	MarginRate types.Number   `json:"margin_rate"`
	LowerLimit types.Number   `json:"lower_limit"`
	UpperLimit TierUpperLimit `json:"upper_limit"`
	Leverage   types.Number   `json:"leverage"`
}

// PortfolioMarginCalculatorParams holds a request parameters for a portfolio margin calculation
type PortfolioMarginCalculatorParams struct {
	SpotBalances     []*SpotBalanceParams `json:"spot_balances,omitempty"`
	SpotOrders       []*SpotOrderParams   `json:"spot_orders,omitempty"`
	FuturesPositions []*FuturesOrderInfo  `json:"futures_positions,omitempty"`
	FuturesOrders    []*FuturesOrderInfo  `json:"futures_orders,omitempty"`
	OptionsPositions []*OptionsOrderInfo  `json:"options_positions,omitempty"`
	OptionsOrders    []*OptionsOrderInfo  `json:"options_orders,omitempty"`
	SpotHedge        bool                 `json:"spot_hedge,omitempty"`
}

// SpotBalanceParams holds a spot balance parameters
type SpotBalanceParams struct {
	Currency currency.Code `json:"currency"`
	Equity   types.Number  `json:"equity"` // Balance less borrowings, negative for a net short exposure
	Freeze   types.Number  `json:"freeze,omitempty"`
}

// SpotOrderParams represents a unified account margin calculator spot order parameters
type SpotOrderParams struct {
	CurrencyPair currency.Pair `json:"currency_pairs"`
	OrderPrice   types.Number  `json:"order_price"`
	Count        types.Number  `json:"count,omitempty"` // Initial order quantity, not used in the calculation
	Left         types.Number  `json:"left"`
	Side         order.Side    `json:"type"`
}

// MarshalJSON lowercases the order side, which order.Side otherwise marshals as "BUY"/"SELL" while Gate only accepts
// "buy"/"sell"
func (s *SpotOrderParams) MarshalJSON() ([]byte, error) {
	type Alias SpotOrderParams
	return json.Marshal(&struct {
		Side string `json:"type"`
		*Alias
	}{
		Side:  s.Side.Lower(),
		Alias: (*Alias)(s),
	})
}

// FuturesOrderInfo holds a USDT perpetual position or order for the portfolio margin calculator. Sizes are signed
// contract quantities, negative for short positions and sell orders.
type FuturesOrderInfo struct {
	Contract currency.Pair `json:"contract"`
	Size     types.Number  `json:"size"`
	Left     types.Number  `json:"left,omitempty"` // Orders only
}

// OptionsOrderInfo holds an options position or order for the portfolio margin calculator. Sizes are signed contract
// quantities, negative for short positions and sell orders.
type OptionsOrderInfo struct {
	OptionsName currency.Pair `json:"options_name"`
	Size        types.Number  `json:"size"`
	Left        types.Number  `json:"left,omitempty"` // Orders only
}

// PortfolioMarginCalculationResponse holds a portfolio margin calculation response
type PortfolioMarginCalculationResponse struct {
	MaintainMarginTotal types.Number               `json:"maintain_margin_total"`
	InitialMarginTotal  types.Number               `json:"initial_margin_total"`
	CalculateTime       types.Time                 `json:"calculate_time"`
	RiskUnits           []*PortfolioMarginRiskUnit `json:"risk_unit"`
}

// PortfolioMarginRiskUnit holds the portfolio margin calculation for one underlying
type PortfolioMarginRiskUnit struct {
	Symbol         currency.Code            `json:"symbol"`
	SpotInUse      types.Number             `json:"spot_in_use"`
	MaintainMargin types.Number             `json:"maintain_margin"`
	InitialMargin  types.Number             `json:"initial_margin"`
	MarginResults  []*PortfolioMarginResult `json:"margin_result"`
	Delta          types.Number             `json:"delta"`
	Gamma          types.Number             `json:"gamma"`
	Theta          types.Number             `json:"theta"`
	Vega           types.Number             `json:"vega"`
}

// PortfolioMarginResult holds the margin requirements of one position combination of a risk unit
type PortfolioMarginResult struct {
	Type                 string            `json:"type"`
	ProfitLossRanges     []*VolatilityInfo `json:"profit_loss_ranges"`
	MaxLoss              *VolatilityInfo   `json:"max_loss"`
	StressTest           types.Number      `json:"mr1"`
	BasisSpreadRisk      types.Number      `json:"mr2"`
	VolatilitySpreadRisk types.Number      `json:"mr3"`
	OptionShortRisk      types.Number      `json:"mr4"`
}

// VolatilityInfo represents a volatility info
type VolatilityInfo struct {
	PricePercentage             types.Number `json:"price_percentage"`
	ImpliedVolatilityPercentage types.Number `json:"implied_volatility_percentage"`
	ProfitLoss                  types.Number `json:"profit_loss"`
}

// UnifiedLeverageRequest holds the leverage to apply to every currency the user borrows
type UnifiedLeverageRequest struct {
	Leverage types.Number `json:"leverage"`
}

// LeverageFailedCurrencyResponse holds a currency whose leverage could not be updated
type LeverageFailedCurrencyResponse struct {
	Currency currency.Code `json:"currency"`
	Reason   string        `json:"reason"`
}

// UnifiedLeverageSetting holds a unified account currency leverage multiplier setting.
type UnifiedLeverageSetting struct {
	Currency currency.Code `json:"currency"`
	Leverage types.Number  `json:"leverage"`
}

// UnifiedCurrencyLeverageConfig holds the maximum and minimum currency leverage configuration.
type UnifiedCurrencyLeverageConfig struct {
	CurrentLeverage          types.Number `json:"current_leverage"`
	MinLeverage              types.Number `json:"min_leverage"`
	MaxLeverage              types.Number `json:"max_leverage"`
	Debit                    types.Number `json:"debit"`
	AvailableMargin          types.Number `json:"available_margin"`
	Borrowable               types.Number `json:"borrowable"`
	ExceptLeverageBorrowable types.Number `json:"except_leverage_borrowable"`
}

// UnifiedLoanCurrency holds detail for a loan currency supported by unified account.
type UnifiedLoanCurrency struct {
	Name                 currency.Code `json:"name"`
	Precision            types.Number  `json:"prec"`
	MinBorrowAmount      types.Number  `json:"min_borrow_amount"`
	UserMaxBorrowAmount  types.Number  `json:"user_max_borrow_amount"`  // In USDT
	TotalMaxBorrowAmount types.Number  `json:"total_max_borrow_amount"` // In USDT
	LoanStatus           string        `json:"loan_status"`             // "enable" or "disable"
}

// UnifiedHistoricalLendingRatesRequest holds the filters and paging of a unified account historical lending rate query
type UnifiedHistoricalLendingRatesRequest struct {
	Currency currency.Code
	Tier     string // VIP level of the floating rate
	Page     uint64
	Limit    uint64
}

// UnifiedHistoricalLendingRate holds historical interest rate data for a currency.
type UnifiedHistoricalLendingRate struct {
	Currency   currency.Code         `json:"currency"`
	Tier       string                `json:"tier"`
	TierUpRate types.Number          `json:"tier_up_rate"`
	Rates      []*UnifiedLendingRate `json:"rates"`
}

// UnifiedLendingRate holds the lending rate of an hour
type UnifiedLendingRate struct {
	Time types.Time   `json:"time"`
	Rate types.Number `json:"rate"`
}

// UnifiedSetCollateralRequest is the request body for setting collateral currency.
type UnifiedSetCollateralRequest struct {
	CollateralType uint64          `json:"collateral_type"` // 0: all currencies, 1: the enabled currencies only
	EnableList     []currency.Code `json:"enable_list,omitempty"`
	DisableList    []currency.Code `json:"disable_list,omitempty"`
}

// UnifiedSetCollateralResponse holds the result of setting collateral currency.
type UnifiedSetCollateralResponse struct {
	IsSuccess bool `json:"is_success"`
}

// UnifiedDebtCurrency holds debt currency detail for quick repayment estimation.
type UnifiedDebtCurrency struct {
	Currency        currency.Code `json:"currency"`
	DebtAmount      types.Number  `json:"debt_amount"`
	EstimatedUSD    types.Number  `json:"estimated_usd"`
	Borrowed        types.Number  `json:"borrowed"`
	NegativeBalance types.Number  `json:"neg_balance"`
}

// UnifiedAvailableCurrency holds available currency detail for quick repayment estimation.
type UnifiedAvailableCurrency struct {
	Currency     currency.Code `json:"currency"`
	Available    types.Number  `json:"available"`
	EstimatedUSD types.Number  `json:"estimated_usd"`
}

// UnifiedQuickEstimateRepayment holds the estimated quick repayment details.
type UnifiedQuickEstimateRepayment struct {
	DebtCurrencies      []*UnifiedDebtCurrency      `json:"debt_currencies"`
	AvailableCurrencies []*UnifiedAvailableCurrency `json:"available_currencies"`
}

// UnifiedQuickRepaymentRequest is the request body for quick repayment.
type UnifiedQuickRepaymentRequest struct {
	DebtCurrencies      []currency.Code `json:"debt_currencies"`
	AvailableCurrencies []currency.Code `json:"available_currencies"`
}

// UnifiedRepaidInfo holds repaid currency detail from a quick repayment.
type UnifiedRepaidInfo struct {
	Currency currency.Code `json:"currency"`
	Repaid   types.Number  `json:"repaid"`
	Left     types.Number  `json:"left"`
}

// UnifiedUsedInfo holds used currency detail from a quick repayment.
type UnifiedUsedInfo struct {
	Currency currency.Code `json:"currency"`
	Used     types.Number  `json:"used"`
}

// UnifiedQuickRepaymentResponse holds the result of a quick repayment.
type UnifiedQuickRepaymentResponse struct {
	OrderID     string               `json:"order_id"`
	RepaidInfos []*UnifiedRepaidInfo `json:"repaid_infos"`
	UsedInfos   []*UnifiedUsedInfo   `json:"used_infos"`
}

// DeltaNeutralRequest holds a request to enable or disable the account Delta-neutral strategy mode
type DeltaNeutralRequest struct {
	Enabled bool `json:"enabled"`
}

// DeltaNeutralResponse holds whether the account Delta-neutral strategy mode is enabled
type DeltaNeutralResponse struct {
	Enabled bool `json:"enabled"`
}

// LendingCurrencyDetail holds a lending currency detail information
type LendingCurrencyDetail struct {
	Currency      currency.Code `json:"currency"`
	MinLendAmount types.Number  `json:"min_lend_amount"`
	MaxLendAmount types.Number  `json:"max_lend_amount"` // In USDT
	MaxRate       types.Number  `json:"max_rate"`        // Hourly
	MinRate       types.Number  `json:"min_rate"`        // Hourly
}

// LendingOrRedemptionRequest holds a lending or redemption request parameters
type LendingOrRedemptionRequest struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount"`
	MinRate  types.Number  `json:"min_rate,omitempty"` // Hourly minimum rate, required for lending
	Type     string        `json:"type"`               // lend or redeem
}

// LendOrderDetail represents a user's lend order item detail
type LendOrderDetail struct {
	Currency           currency.Code `json:"currency"`
	CurrentAmount      types.Number  `json:"current_amount"`
	Amount             types.Number  `json:"amount"`
	LentAmount         types.Number  `json:"lent_amount"`
	FrozenAmount       types.Number  `json:"frozen_amount"`
	MinRate            types.Number  `json:"min_rate"`
	InterestStatus     string        `json:"interest_status"`
	ReinvestLeftAmount types.Number  `json:"reinvest_left_amount"`
	CreateTime         types.Time    `json:"create_time"`
	UpdateTime         types.Time    `json:"update_time"`
}

// LendingTransactionRecordsRequest holds lending transaction record filters
type LendingTransactionRecordsRequest struct {
	Currency currency.Code
	Page     uint64
	Limit    uint64
	From     time.Time
	To       time.Time
	Type     string // lend or redeem
}

// LendingTransactionRecord holds a lending order transaction records
type LendingTransactionRecord struct {
	Type             string        `json:"type"`
	Currency         currency.Code `json:"currency"`
	Amount           types.Number  `json:"amount"`
	LastWalletAmount types.Number  `json:"last_wallet_amount"`
	LastLentAmount   types.Number  `json:"last_lent_amount"`
	LastFrozenAmount types.Number  `json:"last_frozen_amount"`
	CreateTime       types.Time    `json:"create_time"`
}

// CurrencyAndInterestIncome holds a currency and interest income detail
type CurrencyAndInterestIncome struct {
	Currency currency.Code `json:"currency"`
	Interest types.Number  `json:"interest"`
}

// UserDividendRecordsRequest holds lending interest record filters
type UserDividendRecordsRequest struct {
	Currency currency.Code
	Page     uint64
	Limit    uint64
	From     time.Time
	To       time.Time
}

// UserDividendRecords holds user's dividend records
type UserDividendRecords struct {
	Status         uint8         `json:"status"` // 0 failure, 1 success
	Currency       currency.Code `json:"currency"`
	ActualRate     types.Number  `json:"actual_rate"`
	Interest       types.Number  `json:"interest"`
	InterestStatus string        `json:"interest_status"`
	CreateTime     types.Time    `json:"create_time"`
}

// CurrencyInterestStatus holds a currency and it's interest status of user's account
type CurrencyInterestStatus struct {
	Currency       currency.Code `json:"currency"`
	InterestStatus string        `json:"interest_status"` // interest_dividend or interest_reinvest
}

// UniLoanAssetData holds a uni loan asset annualised trend chart data
type UniLoanAssetData struct {
	Time  types.Time   `json:"time"`
	Value types.Number `json:"value"`
}

// CurrencyEstimatedAnnualInterestRate holds user's account currency and estimated annualised interest create details
type CurrencyEstimatedAnnualInterestRate struct {
	Currency      currency.Code `json:"currency"`
	EstimatedRate types.Number  `json:"est_rate"` // Estimated Annualised Rate
}

// OrderIDResponse represents an order ID response
type OrderIDResponse struct {
	OrderID uint64 `json:"order_id"`
}

// MultiCollateralAdjustmentRecordsRequest holds multi-collateral loan collateral adjustment record filters
type MultiCollateralAdjustmentRecordsRequest struct {
	Page               uint64
	Limit              uint64
	From               time.Time
	To                 time.Time
	CollateralCurrency currency.Code
}

// MultiCollateralAdjustmentRecord holds a multi collateral adjustment records
type MultiCollateralAdjustmentRecord struct {
	OrderID              uint64            `json:"order_id"`
	RecordID             uint64            `json:"record_id"`
	BeforeLTV            types.Number      `json:"before_ltv"`
	AfterLTV             types.Number      `json:"after_ltv"`
	OperateTime          types.Time        `json:"operate_time"`
	BorrowCurrencies     []*CurrencyDetail `json:"borrow_currencies"`
	CollateralCurrencies []*CurrencyDetail `json:"collateral_currencies"`
}

// MultiCollateralCurrencyQuota holds currency quota information for multi-collateral
type MultiCollateralCurrencyQuota struct {
	Currency           currency.Code `json:"currency"`
	IndexPrice         types.Number  `json:"index_price"`
	MinQuota           types.Number  `json:"min_quota"`
	LeftQuota          types.Number  `json:"left_quota"`
	LeftQuotaUSDT      types.Number  `json:"left_quote_usdt"`
	LeftQuotaFixed     types.Number  `json:"left_quota_fixed"`
	LeftQuotaUSDTFixed types.Number  `json:"left_quote_usdt_fixed"`
}

// MultiCollateralSupportedCurrencies holds borrow and collateral currencies supported by multi-collateral
type MultiCollateralSupportedCurrencies struct {
	LoanCurrencies       []*MultiLoanItem       `json:"loan_currencies"`
	CollateralCurrencies []*MultiCollateralItem `json:"collateral_currencies"`
}

// MultiLoanItem represents a borrowable currency item in multi-collateral
type MultiLoanItem struct {
	Currency currency.Code `json:"currency"`
	Price    types.Number  `json:"price"`
}

// MultiCollateralItem represents a collateral currency item in multi-collateral
type MultiCollateralItem struct {
	Currency   currency.Code `json:"currency"`
	IndexPrice types.Number  `json:"index_price"`
	Discount   types.Number  `json:"discount"`
}

// MultiCollateralisationRatio holds the collateralisation ratio information for multi-collateral
type MultiCollateralisationRatio struct {
	InitLTV      types.Number `json:"init_ltv"`
	AlertLTV     types.Number `json:"alert_ltv"`
	LiquidateLTV types.Number `json:"liquidate_ltv"`
}

// CollateralFixRate holds a currency's 7-day and 30-day fixed interest rates
type CollateralFixRate struct {
	Currency   currency.Code `json:"currency"`
	Rate7Day   types.Number  `json:"rate_7d"`
	Rate30Day  types.Number  `json:"rate_30d"`
	UpdateTime types.Time    `json:"update_time"`
}

// CollateralCurrentRate holds a currency's current interest rate
type CollateralCurrentRate struct {
	Currency    currency.Code `json:"currency"`
	CurrentRate types.Number  `json:"current_rate"`
}

// CurrencyDetail holds a currency detail
type CurrencyDetail struct {
	Currency         currency.Code `json:"currency"`
	IndexPrice       types.Number  `json:"index_price"`
	BeforeAmount     types.Number  `json:"before_amount"`
	BeforeAmountUSDT types.Number  `json:"before_amount_usdt"`
	AfterAmount      types.Number  `json:"after_amount"`
	AfterAmountUSDT  types.Number  `json:"after_amount_usdt"`
}

// websocketResponse is a generic wrapper for all futures websocket responses.
type websocketResponse[T any] struct {
	Time    types.Time `json:"time"`
	Channel string     `json:"channel"`
	Event   string     `json:"event"`
	Result  T          `json:"result"`
}
