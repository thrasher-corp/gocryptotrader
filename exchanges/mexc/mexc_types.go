package mexc

import (
	"encoding/json"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// ExchangeConfig represents rules and symbols of an exchange
type ExchangeConfig struct {
	Timezone        string         `json:"timezone"`
	ServerTime      types.Time     `json:"serverTime"`
	RateLimits      []any          `json:"rateLimits"`
	ExchangeFilters []any          `json:"exchangeFilters"`
	Symbols         []SymbolDetail `json:"symbols"`
}

// SymbolDetail represents a symbol detail.
type SymbolDetail struct {
	Symbol                     string       `json:"symbol"`
	Status                     string       `json:"status"`
	BaseAsset                  string       `json:"baseAsset"`
	BaseAssetPrecision         int          `json:"baseAssetPrecision"`
	QuoteAsset                 string       `json:"quoteAsset"`
	QuotePrecision             int          `json:"quotePrecision"`
	QuoteAssetPrecision        int          `json:"quoteAssetPrecision"`
	BaseCommissionPrecision    int          `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision   int          `json:"quoteCommissionPrecision"`
	OrderTypes                 []string     `json:"orderTypes"`
	IsSpotTradingAllowed       bool         `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed     bool         `json:"isMarginTradingAllowed"`
	QuoteAmountPrecision       string       `json:"quoteAmountPrecision"`
	BaseSizePrecision          string       `json:"baseSizePrecision"`
	Permissions                []string     `json:"permissions"`
	Filters                    []any        `json:"filters"`
	MaxQuoteAmount             string       `json:"maxQuoteAmount"`
	MakerCommission            types.Number `json:"makerCommission"`
	TakerCommission            types.Number `json:"takerCommission"`
	QuoteAmountPrecisionMarket types.Number `json:"quoteAmountPrecisionMarket"`
	MaxQuoteAmountMarket       types.Number `json:"maxQuoteAmountMarket"`
	FullName                   string       `json:"fullName"`
	TradeSideType              int64        `json:"tradeSideType"`
}

// Orderbook represents a symbol orderbook detail
type Orderbook struct {
	LastUpdateID int64             `json:"lastUpdateId"`
	Bids         [][2]types.Number `json:"bids"`
	Asks         [][2]types.Number `json:"asks"`
	Timestamp    types.Time        `json:"timestamp"`
}

// TradeDetail represents a trade detail
type TradeDetail struct {
	ID           any          `json:"id"`
	Price        types.Number `json:"price"`
	Quantity     types.Number `json:"qty"`
	QuoteQty     types.Number `json:"quoteQty"`
	Time         types.Time   `json:"time"`
	IsBuyerMaker bool         `json:"isBuyerMaker"`
	IsBestMatch  bool         `json:"isBestMatch"`
	TradeType    string       `json:"tradeType"`
}

// AggregatedTradeDetail represents an aggregated trade detail
type AggregatedTradeDetail struct {
	AggregatedTradeID any          `json:"a"`
	FirstTradeID      any          `json:"f"`
	LastTradeID       any          `json:"l"`
	Price             types.Number `json:"p"`
	Quantity          types.Number `json:"q"`
	Timestamp         types.Time   `json:"T"`
	MakerBuyer        bool         `json:"m"` // Was the buyer the maker?
	MathBestPrice     bool         `json:"M"` // Was the trade the best price match?
}

// CandlestickData represents a candlestick data for a symbol
type CandlestickData struct {
	OpenTime         types.Time
	OpenPrice        types.Number
	HighPrice        types.Number
	LowPrice         types.Number
	ClosePrice       types.Number
	Volume           types.Number
	CloseTime        types.Time
	QuoteAssetVolume types.Number
}

// UnmarshalJSON deserializes byte data into a CandlestickData instance
func (c *CandlestickData) UnmarshalJSON(data []byte) error {
	target := [8]any{&c.OpenTime, &c.OpenPrice, &c.HighPrice, &c.LowPrice, &c.ClosePrice, &c.Volume, &c.CloseTime, &c.QuoteAssetVolume}
	return json.Unmarshal(data, &target)
}

// SymbolAveragePrice represents a symbol average price detail
type SymbolAveragePrice struct {
	Mins  int64        `json:"mins"`
	Price types.Number `json:"price"`
}

// TickerData represents a ticker data for a symbol
type TickerData struct {
	Symbol             string       `json:"symbol"`
	PriceChange        types.Number `json:"priceChange"`
	PriceChangePercent types.Number `json:"priceChangePercent"`
	PrevClosePrice     types.Number `json:"prevClosePrice"`
	LastPrice          types.Number `json:"lastPrice"`
	BidPrice           types.Number `json:"bidPrice"`
	BidQty             types.Number `json:"bidQty"`
	AskPrice           types.Number `json:"askPrice"`
	AskQty             types.Number `json:"askQty"`
	OpenPrice          types.Number `json:"openPrice"`
	HighPrice          types.Number `json:"highPrice"`
	LowPrice           types.Number `json:"lowPrice"`
	Volume             types.Number `json:"volume"`
	QuoteVolume        types.Number `json:"quoteVolume"`
	OpenTime           types.Time   `json:"openTime"`
	CloseTime          types.Time   `json:"closeTime"`
	Count              any          `json:"count"`
}

// TickerList represents list of ticker data
type TickerList []TickerData

// UnmarshalJSON deserializes byte data into TickerList
func (t *TickerList) UnmarshalJSON(data []byte) error {
	tickers := []TickerData{}
	err := json.Unmarshal(data, &tickers)
	if err != nil {
		var val *TickerData
		err = json.Unmarshal(data, &val)
		if err != nil {
			return err
		}
		tickers = []TickerData{*val}
	}
	*t = tickers
	return nil
}

// SymbolPriceTicker represents a symbol price ticker info
type SymbolPriceTicker struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
}

// SymbolPriceTickers represent list of symbol price tickers
type SymbolPriceTickers []SymbolPriceTicker

func (t *SymbolPriceTickers) UnmarshalJSON(data []byte) error {
	tickers := []SymbolPriceTicker{}
	err := json.Unmarshal(data, &tickers)
	if err != nil {
		var val *SymbolPriceTicker
		err = json.Unmarshal(data, &val)
		if err != nil {
			return err
		}
		tickers = []SymbolPriceTicker{*val}
	}
	*t = tickers
	return nil
}

// SymbolOrderbookTicker represents a symbol orderbook ticker detail
type SymbolOrderbookTicker struct {
	Symbol   string `json:"symbol"`
	BidPrice string `json:"bidPrice"`
	BidQty   string `json:"bidQty"`
	AskPrice string `json:"askPrice"`
	AskQty   string `json:"askQty"`
}

// SymbolOrderbookTickerList represents a list of symbols orderbook ticker detail
type SymbolOrderbookTickerList []SymbolOrderbookTicker

func (t *SymbolOrderbookTickerList) UnmarshalJSON(data []byte) error {
	tickers := []SymbolOrderbookTicker{}
	err := json.Unmarshal(data, &tickers)
	if err != nil {
		var val *SymbolOrderbookTicker
		err = json.Unmarshal(data, &val)
		if err != nil {
			return err
		}
		tickers = []SymbolOrderbookTicker{*val}
	}
	*t = tickers
	return nil
}

// SubAccountCreationResponse represents a sub-account creation response.
type SubAccountCreationResponse struct {
	SubAccount string `json:"subAccount"`
	Note       string `json:"note"`
}

// SubAccounts represents list of sub-accounts and sub-account detail
type SubAccounts struct {
	SubAccounts []struct {
		SubAccount string     `json:"subAccount"`
		IsFreeze   bool       `json:"isFreeze"`
		CreateTime types.Time `json:"createTime"`
		UID        string     `json:"uid"`
	} `json:"subAccounts"`
}

// SubAccountAPIDetail represents a sub-account API key detail
type SubAccountAPIDetail struct {
	SubAccount  string     `json:"subAccount"`
	Note        string     `json:"note"`
	APIKey      string     `json:"apiKey"`
	SecretKey   string     `json:"secretKey"`
	Permissions string     `json:"permissions"`
	IP          string     `json:"ip"`
	CreatTime   types.Time `json:"creatTime"`
}

// SubAccountsAPIs represents a sub-account API keys detail
type SubAccountsAPIs struct {
	SubAccount []SubAccountAPIDetail `json:"subAccount"`
}
