package kraken

// GeneralResponse is a generalized response type
type GeneralResponse struct {
	Result map[string]interface{} `json:"result"`
	Error  []interface{}          `json:"error"`
}

// AssetPairs holds asset pair information
type AssetPairs struct {
	Altname           string      `json:"altname"`
	AclassBase        string      `json:"aclass_base"`
	Base              string      `json:"base"`
	AclassQuote       string      `json:"aclass_quote"`
	Quote             string      `json:"quote"`
	Lot               string      `json:"lot"`
	PairDecimals      int         `json:"pair_decimals"`
	LotDecimals       int         `json:"lot_decimals"`
	LotMultiplier     int         `json:"lot_multiplier"`
	LeverageBuy       []int       `json:"leverage_buy"`
	LeverageSell      []int       `json:"leverage_sell"`
	Fees              [][]float64 `json:"fees"`
	FeesMaker         [][]float64 `json:"fees_maker"`
	FeeVolumeCurrency string      `json:"fee_volume_currency"`
	MarginCall        int         `json:"margin_call"`
	MarginStop        int         `json:"margin_stop"`
}

// Ticker is a standard ticker type
type Ticker struct {
	Ask    float64
	Bid    float64
	Last   float64
	Volume float64
	VWAP   float64
	Trades int64
	Low    float64
	High   float64
	Open   float64
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
	Time   float64
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Vwap   float64
	Volume float64
	Count  float64
}

// RecentTrades holds recent trade data
type RecentTrades struct {
	Price         float64
	Volume        float64
	Time          float64
	BuyOrSell     string
	MarketOrLimit string
	Miscellaneous interface{}
}

// OrderbookBase stores the orderbook price and amount data
type OrderbookBase struct {
	Price  float64
	Amount float64
}

// Orderbook stores the bids and asks orderbook data
type Orderbook struct {
	Bids []OrderbookBase
	Asks []OrderbookBase
}

// Spread holds the spread between trades
type Spread struct {
	Time float64
	Bid  float64
	Ask  float64
}

// Position holds the opened position
type Position struct {
	Ordertxid  string  `json:"ordertxid"`
	Pair       string  `json:"pair"`
	Time       float64 `json:"time"`
	SellOrBy   string  `json:"type"`
	OrderType  string  `json:"ordertype"`
	Cost       float64 `json:"cost,string"`
	Fee        float64 `json:"fee,string"`
	Vol        float64 `json:"vol,string"`
	VolClosed  float64 `json:"vol_closed,string"`
	Margin     float64 `json:"margin,string"`
	Rollovertm int64   `json:"rollovertm,string"`
	Misc       string  `json:"misc"`
	Oflags     string  `json:"oflags"`
	PosStatus  string  `json:"posstatus"`
	Net        string  `json:"net"`
	Terms      string  `json:"terms"`
}

// AddOrderResponse type
type AddOrderResponse struct {
	Description    OrderDescription `json:"descr"`
	TransactionIds []string         `json:"txid"`
}

// OrderDescription represents an orders description
type OrderDescription struct {
	Close string `json:"close"`
	Order string `json:"order"`
}

// AddOrderOptions represents the AddOrder options
type AddOrderOptions struct {
	UserRef        string
	Oflags         string
	StartTm        string
	ExpireTm       string
	CloseOrderType string
	ClosePrice     float64
	ClosePrice2    float64
	Validate       bool
}
