package kucoin

type SymbolInfo struct {
	Symbol          string  `json:"symbol"`
	Name            string  `name`
	BaseCurrency    string  `baseCurrency`
	QuoteCurrency   string  `quoteCurrency`
	FeeCurrency     string  `feeCurrency`
	Market          string  `market`
	BaseMinSize     float64 `baseMinSize,string`
	QuoteMinSize    float64 `quoteMinSize,string`
	BaseMaxSize     float64 `baseMaxSize,string`
	QuoteMaxSize    float64 `quoteMaxSize,string`
	BaseIncrement   float64 `baseIncrement,string`
	QuoteIncrement  float64 `quoteIncrement,string`
	PriceIncrement  float64 `priceIncrement,string`
	PriceLimitRate  float64 `priceLimitRate,string`
	MinFunds        float64 `minFunds,string`
	IsMarginEnabled bool    `isMarginEnabled`
	EnableTrading   bool    `enableTrading`
}
