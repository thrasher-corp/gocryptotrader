package anx

type ANXOrder struct {
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

type ANXOrderResponse struct {
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

type ANXTickerComponent struct {
	Currency     string `json:"currency"`
	Display      string `json:"display"`
	DisplayShort string `json:"display_short"`
	Value        string `json:"value"`
}

type ANXTicker struct {
	Result string `json:"result"`
	Data   struct {
		High       ANXTickerComponent `json:"high"`
		Low        ANXTickerComponent `json:"low"`
		Avg        ANXTickerComponent `json:"avg"`
		Vwap       ANXTickerComponent `json:"vwap"`
		Vol        ANXTickerComponent `json:"vol"`
		Last       ANXTickerComponent `json:"last"`
		Buy        ANXTickerComponent `json:"buy"`
		Sell       ANXTickerComponent `json:"sell"`
		Now        string             `json:"now"`
		UpdateTime string             `json:"dataUpdateTime"`
	} `json:"data"`
}
