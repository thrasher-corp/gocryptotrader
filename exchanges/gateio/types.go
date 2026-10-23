package gateio

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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

	// FuturesContractResponse.Type: a direct contract is worth QuantoMultiplier of its base currency,
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
	Code      int64        `json:"code"` // 0 on success; Gate answers -1 when its service is busy
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

// errorReply holds the label and message Gate answers a failed public request with
type errorReply struct {
	Label   string `json:"label"`
	Message string `json:"message"`
}

// codedErrorReply holds the label, code and message Gate answers a failed request with
type codedErrorReply struct {
	Label   string `json:"label"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TickerResponse holds detail ticker information for a currency pair
type TickerResponse struct {
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

// OrderbookResponse stores the orderbook data
type OrderbookResponse struct {
	ID           int64           `json:"id"` // int64 to match the order book API it feeds
	ResponseTime types.Time      `json:"current"`
	UpdateTime   types.Time      `json:"update"`
	Bids         OrderbookLevels `json:"bids"`
	Asks         OrderbookLevels `json:"asks"`
}

// OrderbookOfLendingLoan represents order book of lending loans
type OrderbookOfLendingLoan struct {
	Rate   types.Number `json:"rate"`
	Amount types.Number `json:"amount"`
	Days   uint64       `json:"days"`
}

// TradingHistoryItemResponse represents a futures or delivery market trade, or one of the user's own fills
type TradingHistoryItemResponse struct {
	ID uint64 `json:"id"`
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

// FuturesCandlestickResponse represents futures candlestick data
type FuturesCandlestickResponse struct {
	Timestamp    types.Time   `json:"t"`
	Volume       types.Number `json:"v"`
	ClosePrice   types.Number `json:"c"`
	HighestPrice types.Number `json:"h"`
	LowestPrice  types.Number `json:"l"`
	OpenPrice    types.Number `json:"o"`
	QuoteVolume  types.Number `json:"sum"`
	Name         string       `json:"n,omitempty"`
}

// FuturesTickerResponse represents futures ticker data.
type FuturesTickerResponse struct {
	Contract         currency.Pair `json:"contract"`
	ChangePercentage types.Number  `json:"change_percentage"`
	Last             types.Number  `json:"last"`
	Low24Hour        types.Number  `json:"low_24h"`
	High24Hour       types.Number  `json:"high_24h"`
	TotalSize        types.Number  `json:"total_size"`
	Volume24Hour     types.Number  `json:"volume_24h"`
	// Deprecated: use the base, quote or settlement currency volumes
	Volume24HourBTC types.Number `json:"volume_24h_btc"`
	// Deprecated: use the base, quote or settlement currency volumes
	Volume24HourUSD    types.Number `json:"volume_24h_usd"`
	Volume24HourBase   types.Number `json:"volume_24h_base"`
	Volume24HourQuote  types.Number `json:"volume_24h_quote"`
	Volume24HourSettle types.Number `json:"volume_24h_settle"`
	QuantoMultiplier   types.Number `json:"quanto_multiplier"`
	MarkPrice          types.Number `json:"mark_price"`
	FundingRate        types.Number `json:"funding_rate"`
	// Deprecated: use FundingRate
	FundingRateIndicative types.Number `json:"funding_rate_indicative"`
	IndexPrice            types.Number `json:"index_price"`
	LowestAsk             types.Number `json:"lowest_ask"`
	LowestAskSize         types.Number `json:"lowest_size"`
	HighestBid            types.Number `json:"highest_bid"`
	HighestBidSize        types.Number `json:"highest_size"`
	ChangeUTC0            types.Number `json:"change_utc0"`
	ChangeUTC8            types.Number `json:"change_utc8"`
	ChangePrice           types.Number `json:"change_price"`
	ChangeUTC0Price       types.Number `json:"change_utc0_price"`
	ChangeUTC8Price       types.Number `json:"change_utc8_price"`
}

// InsuranceBalanceResponse represents futures insurance balance item.
type InsuranceBalanceResponse struct {
	Timestamp types.Time   `json:"t"`
	Balance   types.Number `json:"b"`
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

// TransferCurrencyRequest represents currency transfer.
type TransferCurrencyRequest struct {
	Currency     currency.Code `json:"currency"`
	From         asset.Item    `json:"from"`
	To           asset.Item    `json:"to"`
	Amount       types.Number  `json:"amount"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Settle       currency.Code `json:"settle"`
}

// MarshalJSON sends the source and destination as Gate account names, which differ from asset.Item strings for
// futures assets, and the settlement currency in lower case as Gate reports it
func (t *TransferCurrencyRequest) MarshalJSON() ([]byte, error) {
	from, err := walletAccountType(t.From)
	if err != nil {
		return nil, fmt.Errorf("%w: source account", err)
	}
	to, err := walletAccountType(t.To)
	if err != nil {
		return nil, fmt.Errorf("%w: destination account", err)
	}
	type Alias TransferCurrencyRequest
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

// TriggerTimeResponse holds the time at which a cancel countdown expires
type TriggerTimeResponse struct {
	TriggerTime types.Time `json:"triggerTime"`
}

// TriggerOrderIDResponse represents order creation ID response.
type TriggerOrderIDResponse struct {
	ID       uint64 `json:"id"`
	IDString string `json:"id_string"` // ID as a string, which avoids precision loss outside Go
}

// CurrencyAndAmountResponse represents request parameters for repayment
type CurrencyAndAmountResponse struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount"`
}

// FuturesAccountResponse represents futures account detail
type FuturesAccountResponse struct {
	User          uint64        `json:"user"`
	Currency      currency.Code `json:"currency"`
	Total         types.Number  `json:"total"`
	UnrealisedPNL types.Number  `json:"unrealised_pnl"`
	// Deprecated: Gate has deprecated it
	PositionMargin        types.Number `json:"position_margin"`
	OrderMargin           types.Number `json:"order_margin"`
	Available             types.Number `json:"available"`
	Point                 types.Number `json:"point"`
	Bonus                 types.Number `json:"bonus"`
	EnabledCredit         bool         `json:"enable_credit"`
	InDualMode            bool         `json:"in_dual_mode"`
	UpdateTime            types.Time   `json:"update_time"`
	UpdateID              uint64       `json:"update_id"`
	PositionInitialMargin types.Number `json:"position_initial_margin"`
	MaintenanceMargin     types.Number `json:"maintenance_margin"`
	MarginMode            int64        `json:"margin_mode"` // 0: classic futures account or classic spot margin mode of a unified account, 1: multi-currency margin, 2: portfolio margin, 3: single-currency margin
	// Deprecated: Gate has deprecated it
	EnabledEvolvedClassic      bool         `json:"enable_evolved_classic"`
	CrossOrderMargin           types.Number `json:"cross_order_margin"`
	CrossInitialMargin         types.Number `json:"cross_initial_margin"`
	CrossMaintenanceMargin     types.Number `json:"cross_maintenance_margin"`
	CrossUnrealisedPNL         types.Number `json:"cross_unrealised_pnl"`
	CrossAvailable             types.Number `json:"cross_available"`
	CrossMarginBalance         types.Number `json:"cross_margin_balance"`
	CrossMaintenanceMarginRate types.Number `json:"cross_mmr"`
	CrossInitialMarginRate     types.Number `json:"cross_imr"`
	IsolatedPositionMargin     types.Number `json:"isolated_position_margin"`
	// Deprecated: Gate has deprecated it
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

// AccountBookItemResponse represents account book item
type AccountBookItemResponse struct {
	Time     types.Time   `json:"time"`
	Change   types.Number `json:"change"`
	Balance  types.Number `json:"balance"`
	Text     string       `json:"text"`
	Type     string       `json:"type"`
	Contract string       `json:"contract"`
	TradeID  string       `json:"trade_id"`
	ID       string       `json:"id"`
}

// PositionResponse represents futures position
type PositionResponse struct {
	User                       uint64        `json:"user"`
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
	UpdateID                   uint64        `json:"update_id"`
	TradeMaxSize               types.Number  `json:"trade_max_size"`
	RiskLimitTable             string        `json:"risk_limit_table"`
	AverageMaintenanceRate     types.Number  `json:"average_maintenance_rate"`
	PositionID                 uint64        `json:"pid"`
	VoucherSize                types.Number  `json:"voucher_size"`
	VoucherMargin              types.Number  `json:"voucher_margin"`
	VoucherID                  uint64        `json:"voucher_id"`
	TradeLongSize              types.Number  `json:"trade_long_size"`
	TradeShortSize             types.Number  `json:"trade_short_size"`
	PositionMarginMode         string        `json:"pos_margin_mode"`
	PositionLeverage           types.Number  `json:"lever"` // Indicates the current leverage of the position, applicable to both isolated and cross margin, gradually replacing the current leverage and cross_leverage_limit
}

// CloseOrder represents close order information
type CloseOrder struct {
	ID            uint64       `json:"id"`
	Price         types.Number `json:"price"`
	IsLiquidation bool         `json:"is_liq"`
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

// FuturesOrderResponse represents future order response
type FuturesOrderResponse struct {
	ID uint64 `json:"id"`
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
	ReferenceUserID           uint64        `json:"refu"`
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

// LiquidationHistoryItemResponse liquidation history item
type LiquidationHistoryItemResponse struct {
	Time             types.Time    `json:"time"`
	Contract         currency.Pair `json:"contract"`
	Size             types.Number  `json:"size"`
	Leverage         types.Number  `json:"leverage"`
	Margin           types.Number  `json:"margin"`
	EntryPrice       types.Number  `json:"entry_price"`
	MarkPrice        types.Number  `json:"mark_price"`
	OrderPrice       types.Number  `json:"order_price"`
	FillPrice        types.Number  `json:"fill_price"`
	LiquidationPrice types.Number  `json:"liq_price"`
	OrderID          uint64        `json:"order_id"`
	RemainingAmount  types.Number  `json:"left"`
}

// FuturesPriceTriggeredOrderRequest represents a creates a price triggered order
type FuturesPriceTriggeredOrderRequest struct {
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
	StrategyType int64                 `json:"strategy_type,omitempty"` // How the order will be triggered 0: by price, which means the order will be triggered if price condition is satisfied 1: by price gap, which means the order will be triggered if gap of recent two prices of specified price_type are satisfied. Only 0 is supported currently
	PriceType    uint64                `json:"price_type,omitempty"`
	Price        types.Number          `json:"price,omitempty"`
	Rule         uint64                `json:"rule,omitempty"`
	ExpireAfter  types.DurationSeconds `json:"expiration,omitempty"` // Whole seconds to wait for the condition to trigger before cancelling the order
}

// PriceTriggeredOrderResponse represents a future triggered price order response
type PriceTriggeredOrderResponse struct {
	Initial            PriceTriggeredOrderInitial `json:"initial"`
	Trigger            PriceTriggeredOrderTrigger `json:"trigger"`
	ID                 uint64                     `json:"id"`
	IDString           string                     `json:"id_string"`
	User               uint64                     `json:"user"`
	CreateTime         types.Time                 `json:"create_time"`
	FinishTime         types.Time                 `json:"finish_time"`
	TradeID            uint64                     `json:"trade_id"`
	Status             string                     `json:"status"`
	FinishAs           string                     `json:"finish_as"`
	Reason             string                     `json:"reason"`
	OrderType          string                     `json:"order_type"`
	MatchingOrderID    uint64                     `json:"me_order_id"`
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
	StrategyType uint64                `json:"strategy_type"`
	PriceType    uint64                `json:"price_type"`
	Price        types.Number          `json:"price"`
	Rule         uint64                `json:"rule"`
	ExpireAfter  types.DurationSeconds `json:"expiration"`
}

// WebsocketChannelRequest represents general structure for websocket requests
type WebsocketChannelRequest struct {
	Time    uint64              `json:"time,omitempty"`
	ID      int64               `json:"id,omitempty"` // int64, the type MessageSequence returns and replies are matched by
	Channel string              `json:"channel,omitempty"`
	Event   string              `json:"event,omitempty"`
	Payload []string            `json:"payload,omitempty"`
	Auth    *WebsocketAuthInput `json:"auth,omitempty"`
}

// WebsocketAuthInput represents the authentication information
type WebsocketAuthInput struct {
	Method string `json:"method,omitempty"`
	Key    string `json:"KEY,omitempty"`
	Sign   string `json:"SIGN,omitempty"`
}

// WebsocketEventResponse represents websocket incoming subscription, unsubscription, and update response
type WebsocketEventResponse struct {
	Time    types.Time            `json:"time"`
	ID      int64                 `json:"id"` // int64 like the request's ID
	Channel string                `json:"channel"`
	Event   string                `json:"event"`
	Result  *WebsocketEventResult `json:"result"`
	Error   *WebsocketEventError  `json:"error"`
}

// WebsocketEventResult holds the status of a subscription or unsubscription
type WebsocketEventResult struct {
	Status string `json:"status"`
}

// WebsocketEventError holds the error a refused subscription or unsubscription carries
type WebsocketEventError struct {
	Code    uint64 `json:"code"`
	Message string `json:"message"`
}

// WebsocketPush holds a websocket push, whose result carries the channel's updates
type WebsocketPush[T any] struct {
	Time    types.Time `json:"time"`
	Channel string     `json:"channel"`
	Event   string     `json:"event"`
	Result  T          `json:"result"`
}

// WebsocketMessage represents generalised websocket push data from the server.
type WebsocketMessage struct {
	ID        int64           `json:"id"` // int64 like the request's ID
	Time      time.Time       `json:"time"`
	Channel   string          `json:"channel"`
	Event     string          `json:"event"`
	Result    json.RawMessage `json:"result"`
	RequestID string          `json:"request_id"`
}

// WebsocketTicker websocket ticker information.
type WebsocketTicker struct {
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

// WebsocketTrade represents a websocket push data response for a trade
type WebsocketTrade struct {
	ID           uint64        `json:"id"`
	CreateTime   types.Time    `json:"create_time_ms"`
	Side         string        `json:"side"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Amount       types.Number  `json:"amount"`
	Price        types.Number  `json:"price"`
}

// WebsocketCandlesticks represents the candlestick data for spot, margin and cross margin trades pushed through the websocket channel.
type WebsocketCandlesticks struct {
	Timestamp          types.Time   `json:"t"`
	TotalVolume        types.Number `json:"v"`
	ClosePrice         types.Number `json:"c"`
	HighestPrice       types.Number `json:"h"`
	LowestPrice        types.Number `json:"l"`
	OpenPrice          types.Number `json:"o"`
	NameOfSubscription string       `json:"n"`
}

// WebsocketOrderbookTicker represents the websocket orderbook best bid or best ask push data
type WebsocketOrderbookTicker struct {
	UpdateTime    types.Time    `json:"t"`
	UpdateOrderID uint64        `json:"u"`
	Pair          currency.Pair `json:"s"`
	BestBidPrice  types.Number  `json:"b"`
	BestBidAmount types.Number  `json:"B"`
	BestAskPrice  types.Number  `json:"a"`
	BestAskAmount types.Number  `json:"A"`
}

// WebsocketOrderbookUpdate represents websocket orderbook update push data
type WebsocketOrderbookUpdate struct {
	UpdateTime    types.Time                       `json:"t"`
	Pair          currency.Pair                    `json:"s"`
	FirstUpdateID int64                            `json:"U"` // First update order book id in this event since last update
	LastUpdateID  int64                            `json:"u"` // int64 to match the order book API it feeds
	Bids          orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks          orderbook.LevelsArrayPriceAmount `json:"a"`
}

// WebsocketOrderbookUpdateWithSnapshot represents websocket orderbook update push data
type WebsocketOrderbookUpdateWithSnapshot struct {
	UpdateTime    types.Time                       `json:"t"`
	Full          bool                             `json:"full"`
	Channel       string                           `json:"s"`
	FirstUpdateID int64                            `json:"U"` // int64 to match the order book API it feeds
	LastUpdateID  int64                            `json:"u"` // int64 to match the order book API it feeds
	Bids          orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks          orderbook.LevelsArrayPriceAmount `json:"a"`
}

// WebsocketOrderbookSnapshot represents a websocket orderbook snapshot push data
type WebsocketOrderbookSnapshot struct {
	UpdateTime   types.Time                       `json:"t"`
	LastUpdateID int64                            `json:"lastUpdateId"` // int64 to match the order book API it feeds
	CurrencyPair currency.Pair                    `json:"s"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// WebsocketSpotOrder represents an order push data through the websocket channel.
type WebsocketSpotOrder struct {
	ID                 string        `json:"id,omitempty"`
	User               uint64        `json:"user"`
	Text               string        `json:"text,omitempty"`
	Succeeded          bool          `json:"succeeded,omitempty"`
	Label              string        `json:"label,omitempty"`
	Message            string        `json:"message,omitempty"`
	CurrencyPair       currency.Pair `json:"currency_pair,omitzero"`
	TradeQuote         currency.Code `json:"money,omitzero"` // The quote currency a unified order book order trades in
	Type               string        `json:"type,omitempty"`
	Account            string        `json:"account,omitempty"`
	Side               string        `json:"side,omitempty"`
	Amount             types.Number  `json:"amount,omitempty"`
	Price              types.Number  `json:"price,omitempty"`
	TimeInForce        string        `json:"time_in_force,omitempty"`
	Iceberg            string        `json:"iceberg,omitempty"`
	Left               types.Number  `json:"left,omitempty"`
	FilledTotal        types.Number  `json:"filled_total,omitempty"`
	FilledAmount       types.Number  `json:"filled_amount,omitempty"`
	AverageFillPrice   types.Number  `json:"avg_deal_price,omitempty"`
	Fee                types.Number  `json:"fee,omitempty"`
	FeeCurrency        currency.Code `json:"fee_currency,omitzero"`
	PointFee           types.Number  `json:"point_fee,omitempty"`
	GTFee              types.Number  `json:"gt_fee,omitempty"`
	GTDiscount         bool          `json:"gt_discount,omitempty"`
	RebatedFee         types.Number  `json:"rebated_fee,omitempty"`
	RebatedFeeCurrency currency.Code `json:"rebated_fee_currency,omitzero"`
	Event              string        `json:"event"`
	FinishAs           string        `json:"finish_as,omitempty"`
	CreateTime         types.Time    `json:"create_time_ms,omitzero"`
	UpdateTime         types.Time    `json:"update_time_ms,omitzero"`
}

// WebsocketUserPersonalTrade represents a user's personal trade pushed through the websocket connection.
type WebsocketUserPersonalTrade struct {
	ID           uint64        `json:"id"`
	UserID       uint64        `json:"user_id"`
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

// WebsocketSpotBalance represents a spot balance.
type WebsocketSpotBalance struct {
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

// WebsocketMarginBalance represents margin account balance push data
type WebsocketMarginBalance struct {
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

// WebsocketFundingBalance represents funding balance push data.
type WebsocketFundingBalance struct {
	Timestamp types.Time    `json:"timestamp_ms"`
	User      string        `json:"user"`
	Currency  currency.Code `json:"currency"`
	Change    types.Number  `json:"change"`
	Freeze    types.Number  `json:"freeze"`
	Lent      types.Number  `json:"lent"`
}

// WebsocketCrossMarginBalance represents a cross margin balance detail
type WebsocketCrossMarginBalance struct {
	Timestamp types.Time    `json:"timestamp_ms"`
	User      string        `json:"user"`
	Currency  currency.Code `json:"currency"`
	Change    types.Number  `json:"change"`
	Total     types.Number  `json:"total"`
	Available types.Number  `json:"available"`
}

// WebsocketCrossMarginLoan represents a cross margin loan push data
type WebsocketCrossMarginLoan struct {
	Timestamp types.Time    `json:"timestamp"`
	User      string        `json:"user"`
	Currency  currency.Code `json:"currency"`
	Change    types.Number  `json:"change"`
	Total     types.Number  `json:"total"`
	Available types.Number  `json:"available"`
	Borrowed  types.Number  `json:"borrowed"`
	Interest  types.Number  `json:"interest"`
}

// WebsocketFutureTicker represents a futures push data.
type WebsocketFutureTicker struct {
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

// WebsocketFuturesTrades represents a list of trades push data
type WebsocketFuturesTrades struct {
	Size       types.Number  `json:"size"`
	ID         uint64        `json:"id"`
	CreateTime types.Time    `json:"create_time_ms"`
	Price      types.Number  `json:"price"`
	Contract   currency.Pair `json:"contract"`
	IsInternal bool          `json:"is_internal"`
}

// WebsocketFuturesOrderbookTicker represents the orderbook ticker push data
type WebsocketFuturesOrderbookTicker struct {
	Timestamp     types.Time   `json:"t"`
	UpdateID      uint64       `json:"u"`
	CurrencyPair  string       `json:"s"`
	BestBidPrice  types.Number `json:"b"`
	BestBidAmount types.Number `json:"B"`
	BestAskPrice  types.Number `json:"a"`
	BestAskAmount types.Number `json:"A"`
}

// WebsocketFuturesAndOptionsOrderbookUpdate represents futures and options account orderbook update push data
type WebsocketFuturesAndOptionsOrderbookUpdate struct {
	Timestamp      types.Time    `json:"t"`
	ContractName   currency.Pair `json:"s"`
	FirstUpdatedID int64         `json:"U"` // int64 to match the order book API it feeds
	LastUpdatedID  int64         `json:"u"` // int64 to match the order book API it feeds
	Bids           []Level       `json:"b"`
	Asks           []Level       `json:"a"`
}

// Level represents a level of orderbook data
type Level struct {
	Price types.Number `json:"p"`
	Size  types.Number `json:"s"`
}

// WebsocketFuturesOrderbookSnapshot represents a futures orderbook snapshot push data
type WebsocketFuturesOrderbookSnapshot struct {
	Timestamp   types.Time    `json:"t"`
	Contract    currency.Pair `json:"contract"`
	OrderbookID uint64        `json:"id"`
	Asks        []Level       `json:"asks"`
	Bids        []Level       `json:"bids"`
}

// WebsocketFuturesOrderbookUpdateEvent represents futures orderbook push data with the event 'update'
type WebsocketFuturesOrderbookUpdateEvent struct {
	Price        types.Number `json:"p"`
	Amount       types.Number `json:"s"`
	CurrencyPair string       `json:"c"`
	ID           uint64       `json:"id"`
}

// WebsocketFuturesUserTrade represents a futures account user trade push data
type WebsocketFuturesUserTrade struct {
	ID         string        `json:"id"`
	CreateTime types.Time    `json:"create_time_ms"`
	Contract   currency.Pair `json:"contract"`
	OrderID    string        `json:"order_id"`
	Size       types.Number  `json:"size"`
	Price      types.Number  `json:"price"`
	Role       string        `json:"role"`
	Text       string        `json:"text"`
	Fee        types.Number  `json:"fee"`
	PointFee   types.Number  `json:"point_fee"`
}

// WebsocketFuturesLiquidationNotification represents a liquidation notification push data
type WebsocketFuturesLiquidationNotification struct {
	EntryPrice       types.Number  `json:"entry_price"`
	FillPrice        types.Number  `json:"fill_price"`
	RemainingAmount  types.Number  `json:"left"`
	Leverage         types.Number  `json:"leverage"`
	LiquidationPrice types.Number  `json:"liq_price"`
	Margin           types.Number  `json:"margin"`
	MarkPrice        types.Number  `json:"mark_price"`
	OrderID          uint64        `json:"order_id"`
	OrderPrice       types.Number  `json:"order_price"`
	Size             types.Number  `json:"size"`
	Time             types.Time    `json:"time_ms"`
	Contract         currency.Pair `json:"contract"`
	User             string        `json:"user"`
}

// WebsocketFuturesAutoDeleveragesNotification represents futures auto deleverages push data
type WebsocketFuturesAutoDeleveragesNotification struct {
	EntryPrice   types.Number  `json:"entry_price"`
	FillPrice    types.Number  `json:"fill_price"`
	PositionSize types.Number  `json:"position_size"`
	TradeSize    types.Number  `json:"trade_size"`
	Time         types.Time    `json:"time_ms"`
	Contract     currency.Pair `json:"contract"`
	User         string        `json:"user"`
}

// WebsocketPositionClose represents a close position futures push data
type WebsocketPositionClose struct {
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

// WebsocketBalance represents a options and futures balance push data
type WebsocketBalance struct {
	Balance  types.Number  `json:"balance"`
	Change   types.Number  `json:"change"`
	Currency currency.Code `json:"currency"`
	Text     string        `json:"text"`
	Time     types.Time    `json:"time_ms"`
	Type     string        `json:"type"`
	User     string        `json:"user"`
}

// WebsocketFuturesReduceRiskLimitNotification represents a futures reduced risk limit push data
type WebsocketFuturesReduceRiskLimitNotification struct {
	CancelOrders     uint64       `json:"cancel_orders"`
	Contract         string       `json:"contract"`
	LeverageMax      uint64       `json:"leverage_max"`
	LiquidationPrice types.Number `json:"liq_price"`
	MaintenanceRate  types.Number `json:"maintenance_rate"`
	RiskLimit        uint64       `json:"risk_limit"`
	Time             types.Time   `json:"time_ms"`
	User             string       `json:"user"`
}

// WebsocketFuturesPosition represents futures notify positions update.
type WebsocketFuturesPosition struct {
	Contract           string       `json:"contract"`
	CrossLeverageLimit types.Number `json:"cross_leverage_limit"`
	EntryPrice         types.Number `json:"entry_price"`
	HistoryPNL         types.Number `json:"history_pnl"`
	HistoryPoint       types.Number `json:"history_point"`
	LastClosePNL       types.Number `json:"last_close_pnl"`
	Leverage           types.Number `json:"leverage"`
	LeverageMax        types.Number `json:"leverage_max"`
	LiquidationPrice   types.Number `json:"liq_price"`
	MaintenanceRate    types.Number `json:"maintenance_rate"`
	Margin             types.Number `json:"margin"`
	Mode               string       `json:"mode"`
	RealisedPNL        types.Number `json:"realised_pnl"`
	RealisedPoint      types.Number `json:"realised_point"`
	RiskLimit          types.Number `json:"risk_limit"`
	Size               types.Number `json:"size"`
	TimeSeconds        types.Time   `json:"time"`
	Time               types.Time   `json:"time_ms"`
	UpdateID           int64        `json:"update_id"` // int64 to match the futures position API it feeds
	User               string       `json:"user"`
	PositionMarginMode string       `json:"pos_margin_mode"`
	PositionLeverage   types.Number `json:"lever"`
}

// WebsocketFuturesAutoOrder represents an auto order push data.
type WebsocketFuturesAutoOrder struct {
	User        uint64                               `json:"user"`
	Trigger     PriceTriggeredOrderTrigger           `json:"trigger"`
	Initial     WebsocketFuturesAutoOrderInitial     `json:"initial"`
	ID          uint64                               `json:"id"`
	TradeID     uint64                               `json:"trade_id"`
	Status      string                               `json:"status"`
	Reason      string                               `json:"reason"`
	CreateTime  types.Time                           `json:"create_time"`
	Name        string                               `json:"name"`
	IsStopOrder bool                                 `json:"is_stop_order"`
	StopTrigger WebsocketFuturesAutoOrderStopTrigger `json:"stop_trigger"`
}

// WebsocketFuturesAutoOrderInitial holds the order an auto order places once triggered
type WebsocketFuturesAutoOrderInitial struct {
	Contract     string       `json:"contract"`
	Size         types.Number `json:"size"`
	Price        types.Number `json:"price"`
	TimeInForce  string       `json:"tif"`
	Text         string       `json:"text"`
	Iceberg      types.Number `json:"iceberg"`
	IsClose      bool         `json:"is_close"`
	IsReduceOnly bool         `json:"is_reduce_only"`
}

// WebsocketFuturesAutoOrderStopTrigger holds the take-profit or stop-loss condition of an auto order
type WebsocketFuturesAutoOrderStopTrigger struct {
	Rule         uint64       `json:"rule"`
	TriggerPrice types.Number `json:"trigger_price"`
	OrderPrice   types.Number `json:"order_price"`
}

// WebsocketOptionUnderlyingTicker represents options underlying ticker push data
type WebsocketOptionUnderlyingTicker struct {
	TradePut   uint64       `json:"trade_put"`
	TradeCall  uint64       `json:"trade_call"`
	IndexPrice types.Number `json:"index_price"`
	Name       string       `json:"name"`
}

// WebsocketOptionsTrades represents options trades for websocket push data.
type WebsocketOptionsTrades struct {
	ID         uint64        `json:"id"`
	CreateTime types.Time    `json:"create_time_ms"`
	Contract   currency.Pair `json:"contract"`
	Size       types.Number  `json:"size"`
	Price      types.Number  `json:"price"`
	Underlying string        `json:"underlying"`
	IsCall     bool          `json:"is_call"` // added in underlying trades
}

// WebsocketOptionsUnderlyingPrice represents the underlying price.
type WebsocketOptionsUnderlyingPrice struct {
	Underlying string       `json:"underlying"`
	Price      types.Number `json:"price"`
	UpdateTime types.Time   `json:"time_ms"`
}

// WebsocketOptionsMarkPrice represents options mark price push data.
type WebsocketOptionsMarkPrice struct {
	Contract   currency.Pair `json:"contract"`
	Price      types.Number  `json:"price"`
	UpdateTime types.Time    `json:"time_ms"`
}

// WebsocketOptionsSettlement represents a options settlement push data.
type WebsocketOptionsSettlement struct {
	Contract     currency.Pair `json:"contract"`
	OrderbookID  uint64        `json:"orderbook_id"`
	PositionSize types.Number  `json:"position_size"`
	Profit       types.Number  `json:"profit"`
	SettlePrice  types.Number  `json:"settle_price"`
	StrikePrice  types.Number  `json:"strike_price"`
	Tag          string        `json:"tag"`
	TradeID      uint64        `json:"trade_id"`
	TradeSize    types.Number  `json:"trade_size"`
	Underlying   string        `json:"underlying"`
	UpdateTime   types.Time    `json:"time_ms"`
}

// WebsocketOptionsContract represents an option contract push data.
type WebsocketOptionsContract struct {
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

// WebsocketOptionsContractCandlestick represents an options contract candlestick push data.
type WebsocketOptionsContractCandlestick struct {
	Timestamp          types.Time   `json:"t"`
	TotalVolume        types.Number `json:"v"`
	ClosePrice         types.Number `json:"c"`
	HighestPrice       types.Number `json:"h"`
	LowestPrice        types.Number `json:"l"`
	OpenPrice          types.Number `json:"o"`
	Amount             types.Number `json:"a"`
	NameOfSubscription string       `json:"n"` // the format of <interval string>_<currency pair>
}

// WebsocketOptionsOrderbookTicker represents options orderbook ticker push data.
type WebsocketOptionsOrderbookTicker struct {
	UpdateTimestamp types.Time   `json:"t"`
	UpdateID        uint64       `json:"u"`
	ContractName    string       `json:"s"`
	BidPrice        types.Number `json:"b"`
	BidSize         types.Number `json:"B"`
	AskPrice        types.Number `json:"a"`
	AskSize         types.Number `json:"A"`
}

// WebsocketOptionsOrderbookSnapshot represents the options orderbook snapshot push data.
type WebsocketOptionsOrderbookSnapshot struct {
	ID        uint64         `json:"id"`
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

// WebsocketOptionsOrder represents options order push data.
type WebsocketOptionsOrder struct {
	ID              uint64        `json:"id"`
	Contract        currency.Pair `json:"contract"`
	CreateTime      types.Time    `json:"create_time"`
	FillPrice       types.Number  `json:"fill_price"`
	FinishAs        string        `json:"finish_as"`
	Iceberg         types.Number  `json:"iceberg"`
	IsClose         bool          `json:"is_close"`
	IsLiquidation   bool          `json:"is_liq"`
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

// WebsocketOptionsUserTrade represents user's personal trades of option account.
type WebsocketOptionsUserTrade struct {
	ID         string        `json:"id"`
	Underlying string        `json:"underlying"`
	OrderID    string        `json:"order"`
	Contract   currency.Pair `json:"contract"`
	CreateTime types.Time    `json:"create_time_ms"`
	Price      types.Number  `json:"price"`
	Role       string        `json:"role"`
	Size       types.Number  `json:"size"`
}

// WebsocketOptionsLiquidates represents the liquidates push data of option account.
type WebsocketOptionsLiquidates struct {
	User        string       `json:"user"`
	InitMargin  types.Number `json:"init_margin"`
	MaintMargin types.Number `json:"maint_margin"`
	OrderMargin types.Number `json:"order_margin"`
	Time        types.Time   `json:"time_ms"`
}

// WebsocketOptionsUserSettlement represents user's personal settlements push data of options account.
type WebsocketOptionsUserSettlement struct {
	User         string        `json:"user"`
	Contract     currency.Pair `json:"contract"`
	RealisedPNL  types.Number  `json:"realised_pnl"`
	SettlePrice  types.Number  `json:"settle_price"`
	SettleProfit types.Number  `json:"settle_profit"`
	Size         types.Number  `json:"size"`
	StrikePrice  types.Number  `json:"strike_price"`
	Underlying   string        `json:"underlying"`
	SettleTime   types.Time    `json:"time_ms"`
}

// WebsocketOptionsPosition represents positions push data for options account.
type WebsocketOptionsPosition struct {
	EntryPrice  types.Number `json:"entry_price"`
	RealisedPNL types.Number `json:"realised_pnl"`
	Size        types.Number `json:"size"`
	Contract    string       `json:"contract"`
	User        string       `json:"user"`
	UpdateTime  types.Time   `json:"time_ms"`
}

// InterestDeductionRecordResponse holds an interest deduction record
type InterestDeductionRecordResponse struct {
	Status       uint64        `json:"status"` // 0: failed, 1: succeeded
	CurrencyPair currency.Pair `json:"currency_pair"`
	Currency     currency.Code `json:"currency"`
	ActualRate   types.Number  `json:"actual_rate"`
	Interest     types.Number  `json:"interest"`
	LoanType     string        `json:"type"`
	CreateTime   types.Time    `json:"create_time"`
}

// DisplayNumber is a number Gate formats for display, such as $0.04, 50% or $1,234.56. It keeps the text as Gate sent
// it, which marshalling writes back, and the number without its signs and separators for Float64
type DisplayNumber struct {
	Text  string
	Value float64
}

// UnmarshalJSON implements json.Unmarshaler. Text holding no readable number, such as "--", gives a zero Value rather
// than failing the whole reply, as the text stays available
func (d *DisplayNumber) UnmarshalJSON(data []byte) error {
	var text string
	switch {
	case string(data) == "null":
	case len(data) != 0 && data[0] == '"':
		if err := json.Unmarshal(data, &text); err != nil {
			return fmt.Errorf("%w: %s", errInvalidDisplayNumber, data)
		}
	default:
		if _, err := strconv.ParseFloat(string(data), 64); err != nil {
			return fmt.Errorf("%w: %s", errInvalidDisplayNumber, data)
		}
		text = string(data)
	}
	*d = DisplayNumber{Text: text, Value: displayValue(text)}
	return nil
}

// displayValue returns the number in Gate's display text, dropping the signs around it and any thousands separators
func displayValue(text string) float64 {
	start := strings.IndexFunc(text, func(r rune) bool { return r >= '0' && r <= '9' || r == '.' })
	end := strings.LastIndexFunc(text, func(r rune) bool { return r >= '0' && r <= '9' }) + 1
	if start < 0 || end <= start {
		return 0
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(text[start:end], ",", ""), 64)
	if err != nil {
		return 0
	}
	if strings.Contains(text[:start], "-") {
		return -v
	}
	return v
}

// MarshalJSON implements json.Marshaler, writing the text as Gate sent it
func (d DisplayNumber) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// Float64 returns the number without its signs, so 50% returns 50
func (d DisplayNumber) Float64() float64 {
	return d.Value
}

// String returns the number as Gate displays it, or the bare number when there is no text
func (d DisplayNumber) String() string {
	if d.Text == "" && d.Value != 0 {
		return strconv.FormatFloat(d.Value, 'f', -1, 64)
	}
	return d.Text
}

// Date is a calendar date, which Gate formats as 2006-01-02 or 2006/01/02, held as midnight UTC
type Date time.Time

// UnmarshalJSON implements json.Unmarshaler
func (d *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("%w: %s", errInvalidDate, data)
	}
	if s == "" {
		*d = Date{}
		return nil
	}
	for _, layout := range []string{time.DateOnly, "2006/01/02"} {
		if t, err := time.Parse(layout, s); err == nil {
			*d = Date(t)
			return nil
		}
	}
	return fmt.Errorf("%w: %q", errInvalidDate, s)
}

// MarshalJSON implements json.Marshaler, writing the date as 2006-01-02, or an empty string when unset
func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time().IsZero() {
		return []byte(`""`), nil
	}
	return json.Marshal(d.Time().Format(time.DateOnly))
}

// Time returns the date as a time.Time at midnight UTC
func (d Date) Time() time.Time {
	return time.Time(d)
}

// OrderIDResponse represents an order ID response
type OrderIDResponse struct {
	OrderID uint64 `json:"order_id"`
}

// websocketResponse is a generic wrapper for all futures websocket responses.
type websocketResponse[T any] struct {
	Time    types.Time `json:"time"`
	Channel string     `json:"channel"`
	Event   string     `json:"event"`
	Result  T          `json:"result"`
}
