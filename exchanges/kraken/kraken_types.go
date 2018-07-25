package kraken

import "github.com/kempeng/gocryptotrader/decimal"

// GeneralResponse is a generalized response type
type GeneralResponse struct {
	Result map[string]interface{} `json:"result"`
	Error  []interface{}          `json:"error"`
}

// AssetPairs holds asset pair information
type AssetPairs struct {
	Altname           string              `json:"altname"`
	AclassBase        string              `json:"aclass_base"`
	Base              string              `json:"base"`
	AclassQuote       string              `json:"aclass_quote"`
	Quote             string              `json:"quote"`
	Lot               string              `json:"lot"`
	PairDecimals      int                 `json:"pair_decimals"`
	LotDecimals       int                 `json:"lot_decimals"`
	LotMultiplier     int                 `json:"lot_multiplier"`
	LeverageBuy       []int               `json:"leverage_buy"`
	LeverageSell      []int               `json:"leverage_sell"`
	Fees              [][]decimal.Decimal `json:"fees"`
	FeesMaker         [][]decimal.Decimal `json:"fees_maker"`
	FeeVolumeCurrency string              `json:"fee_volume_currency"`
	MarginCall        int                 `json:"margin_call"`
	MarginStop        int                 `json:"margin_stop"`
}

// Ticker is a standard ticker type
type Ticker struct {
	Ask    decimal.Decimal
	Bid    decimal.Decimal
	Last   decimal.Decimal
	Volume decimal.Decimal
	VWAP   decimal.Decimal
	Trades int64
	Low    decimal.Decimal
	High   decimal.Decimal
	Open   decimal.Decimal
}

// TickerResponse holds ticker information before its put into the Ticker struct
type TickerResponse struct {
	Ask    []string `json:"a"`
	Bid    []string `json:"b"`
	Last   []string `json:"c"`
	Volume []string `json:"v"`
	VWAP   []string `json:"p"`
	Trades []int64  `json:"t"`
	Low    []string `json:"l"`
	High   []string `json:"h"`
	Open   string   `json:"o"`
}

// OpenHighLowClose contains ticker event information
type OpenHighLowClose struct {
	Time   decimal.Decimal
	Open   decimal.Decimal
	High   decimal.Decimal
	Low    decimal.Decimal
	Close  decimal.Decimal
	Vwap   decimal.Decimal
	Volume decimal.Decimal
	Count  decimal.Decimal
}

// RecentTrades holds recent trade data
type RecentTrades struct {
	Price         decimal.Decimal
	Volume        decimal.Decimal
	Time          decimal.Decimal
	BuyOrSell     string
	MarketOrLimit string
	Miscellaneous interface{}
}

// OrderbookBase stores the orderbook price and amount data
type OrderbookBase struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
}

// Orderbook stores the bids and asks orderbook data
type Orderbook struct {
	Bids []OrderbookBase
	Asks []OrderbookBase
}

// Spread holds the spread between trades
type Spread struct {
	Time decimal.Decimal
	Bid  decimal.Decimal
	Ask  decimal.Decimal
}
