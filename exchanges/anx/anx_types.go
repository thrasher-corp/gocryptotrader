package anx

// Order holds order information
type Order struct {
	OrderType                      string `json:"orderType"`
	BuyTradedCurrency              bool   `json:"buyTradedCurrency"`
	TradedCurrency                 string `json:"tradedCurrency"`
	SettlementCurrency             string `json:"settlementCurrency"`
	TradedCurrencyAmount           string `json:"tradedCurrencyAmount"`
	SettlementCurrencyAmount       string `json:"settlementCurrencyAmount"`
	LimitPriceInSettlementCurrency string `json:"limitPriceInSettlementCurrency"`
	ReplaceExistingOrderUUID       string `json:"replaceExistingOrderUuid"`
	ReplaceOnlyIfActive            bool   `json:"replaceOnlyIfActive"`
}

// OrderResponse holds order response data
type OrderResponse struct {
	BuyTradedCurrency              bool   `json:"buyTradedCurrency"`
	ExecutedAverageRate            string `json:"executedAverageRate"`
	LimitPriceInSettlementCurrency string `json:"limitPriceInSettlementCurrency"`
	OrderID                        string `json:"orderId"`
	OrderStatus                    string `json:"orderStatus"`
	OrderType                      string `json:"orderType"`
	ReplaceExistingOrderUUID       string `json:"replaceExistingOrderId"`
	SettlementCurrency             string `json:"settlementCurrency"`
	SettlementCurrencyAmount       string `json:"settlementCurrencyAmount"`
	SettlementCurrencyOutstanding  string `json:"settlementCurrencyOutstanding"`
	Timestamp                      int64  `json:"timestamp"`
	TradedCurrency                 string `json:"tradedCurrency"`
	TradedCurrencyAmount           string `json:"tradedCurrencyAmount"`
	TradedCurrencyOutstanding      string `json:"tradedCurrencyOutstanding"`
}

// TickerComponent is a sub-type for ticker
type TickerComponent struct {
	Currency     string `json:"currency"`
	Display      string `json:"display"`
	DisplayShort string `json:"display_short"`
	Value        string `json:"value"`
}

// Ticker contains ticker data
type Ticker struct {
	Result string `json:"result"`
	Data   struct {
		High       TickerComponent `json:"high"`
		Low        TickerComponent `json:"low"`
		Avg        TickerComponent `json:"avg"`
		Vwap       TickerComponent `json:"vwap"`
		Vol        TickerComponent `json:"vol"`
		Last       TickerComponent `json:"last"`
		Buy        TickerComponent `json:"buy"`
		Sell       TickerComponent `json:"sell"`
		Now        string          `json:"now"`
		UpdateTime string          `json:"dataUpdateTime"`
	} `json:"data"`
}

// DepthItem contains depth information
type DepthItem struct {
	Price     float64 `json:"price,string"`
	PriceInt  float64 `json:"price_int,string"`
	Amount    float64 `json:"amount,string"`
	AmountInt int64   `json:"amount_int,string"`
}

// Depth contains full depth information
type Depth struct {
	Result string `json:"result"`
	Data   struct {
		Now            string      `json:"now"`
		DataUpdateTime string      `json:"dataUpdateTime"`
		Asks           []DepthItem `json:"asks"`
		Bids           []DepthItem `json:"bids"`
	} `json:"data"`
}
