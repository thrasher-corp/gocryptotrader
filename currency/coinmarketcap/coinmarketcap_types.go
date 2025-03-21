package coinmarketcap

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Coinmarketcap account plan bitmasks, url and endpoint consts
const (
	Basic uint8 = 1 << iota
	Hobbyist
	Startup
	Standard
	Professional
	Enterprise

	baseURL    = "https://pro-api.coinmarketcap.com"
	sandboxURL = "https://sandbox-api.coinmarketcap.com"
	version    = "/v1/"

	endpointCryptocurrencyInfo               = "cryptocurrency/info"
	endpointCryptocurrencyMap                = "cryptocurrency/map"
	endpointCryptocurrencyHistoricalListings = "cryptocurrency/listings/historical"
	endpointCryptocurrencyLatestListings     = "cryptocurrency/listings/latest"
	endpointCryptocurrencyMarketPairs        = "cryptocurrency/market-pairs/latest"
	endpointOHLCVHistorical                  = "cryptocurrency/ohlcv/historical"
	endpointOHLCVLatest                      = "cryptocurrency/ohlcv/latest"
	endpointGetMarketQuotesHistorical        = "cryptocurrency/quotes/historical"
	endpointGetMarketQuotesLatest            = "cryptocurrency/quotes/latest"
	endpointExchangeInfo                     = "exchange/info"
	endpointExchangeMap                      = "exchange/map"
	endpointExchangeMarketPairsLatest        = "exchange/market-pairs/latest"
	endpointExchangeMarketQuoteHistorical    = "exchange/quotes/historical"
	endpointExchangeMarketQuoteLatest        = "exchange/quotes/latest"
	endpointGlobalQuoteHistorical            = "global-metrics/quotes/historical"
	endpointGlobalQuoteLatest                = "global-metrics/quotes/latest"
	endpointPriceConversion                  = "tools/price-conversion"

	defaultTimeOut = time.Second * 15

	// BASIC, HOBBYIST STARTUP tier rate limits
	RateInterval     = time.Minute
	BasicRequestRate = 30

	// STANDARD tier rate limit
	StandardRequestRate = 60

	// PROFESSIONAL tier rate limit
	ProfessionalRequestRate = 90

	// ENTERPRISE tier rate limit - Can be extended checkout agreement
	EnterpriseRequestRate = 120
)

// Coinmarketcap is the overarching type across this package
type Coinmarketcap struct {
	Verbose    bool
	Enabled    bool
	Name       string
	APIkey     string
	APIUrl     string
	APIVersion string
	Plan       uint8
	Requester  *request.Requester
}

// Settings defines the current settings from configuration file
type Settings struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Verbose     bool   `json:"verbose"`
	APIKey      string `json:"apiKey"`
	AccountPlan string `json:"accountPlan"`
}

// Status defines a response status JSON struct that is received with every
// HTTP request
type Status struct {
	Timestamp    string `json:"timestamp"`
	ErrorCode    int64  `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Elapsed      int64  `json:"elapsed"`
	CreditCount  int64  `json:"credit_count"`
}

// Currency defines a generic sub type to capture currency data
type Currency struct {
	Price                  float64   `json:"price"`
	Volume24H              float64   `json:"volume_24h"`
	Volume24HAdjusted      float64   `json:"volume_24h_adjusted"`
	Volume7D               float64   `json:"volume_7d"`
	Volume30D              float64   `json:"volume_30d"`
	PercentChange1H        float64   `json:"percent_change_1h"`
	PercentChangeVolume24H float64   `json:"percent_change_volume_24h"`
	PercentChangeVolume7D  float64   `json:"percent_change_volume_7d"`
	PercentChangeVolume30D float64   `json:"percent_change_volume_30d"`
	MarketCap              float64   `json:"market_cap"`
	TotalMarketCap         float64   `json:"total_market_cap"`
	LastUpdated            time.Time `json:"last_updated"`
}

// OHLC defines a generic sub type for OHLC currency data
type OHLC struct {
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// CryptoCurrencyInfo defines cryptocurrency information
type CryptoCurrencyInfo map[string]struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Symbol   string   `json:"symbol"`
	Category string   `json:"category"`
	Slug     string   `json:"slug"`
	Logo     string   `json:"logo"`
	Tags     []string `json:"tags"`
	Platform any      `json:"platform"`
	Urls     struct {
		Website      []string `json:"website"`
		Explorer     []string `json:"explorer"`
		SourceCode   []string `json:"source_code"`
		MessageBoard []string `json:"message_board"`
		Chat         []any    `json:"chat"`
		Announcement []any    `json:"announcement"`
		Reddit       []string `json:"reddit"`
		Twitter      []string `json:"twitter"`
	} `json:"urls"`
}

// CryptoCurrencyMap defines a cryptocurrency struct
type CryptoCurrencyMap struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	Symbol              string    `json:"symbol"`
	Slug                string    `json:"slug"`
	IsActive            int       `json:"is_active"`
	FirstHistoricalData time.Time `json:"first_historical_data"`
	LastHistoricalData  time.Time `json:"last_historical_data"`
	Platform            struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		Symbol       string `json:"symbol"`
		Slug         string `json:"slug"`
		TokenAddress string `json:"token_address"`
	} `json:"platform"`
}

// CryptocurrencyHistoricalListings defines a historical listing data
type CryptocurrencyHistoricalListings struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Symbol            string    `json:"symbol"`
	Slug              string    `json:"slug"`
	CmcRank           int       `json:"cmc_rank"`
	NumMarketPairs    int       `json:"num_market_pairs"`
	CirculatingSupply float64   `json:"circulating_supply"`
	TotalSupply       float64   `json:"total_supply"`
	MaxSupply         float64   `json:"max_supply"`
	LastUpdated       time.Time `json:"last_updated"`
	Quote             struct {
		USD Currency `json:"USD"`
		BTC Currency `json:"BTC"`
	} `json:"quote"`
}

// CryptocurrencyLatestListings defines latest cryptocurrency listing data
type CryptocurrencyLatestListings struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Symbol            string    `json:"symbol"`
	Slug              string    `json:"slug"`
	CmcRank           int       `json:"cmc_rank"`
	NumMarketPairs    int       `json:"num_market_pairs"`
	CirculatingSupply float64   `json:"circulating_supply"`
	TotalSupply       float64   `json:"total_supply"`
	MaxSupply         float64   `json:"max_supply"`
	LastUpdated       time.Time `json:"last_updated"`
	DateAdded         time.Time `json:"date_added"`
	Tags              []string  `json:"tags"`
	Platform          any       `json:"platform"`
	Quote             struct {
		USD Currency `json:"USD"`
		BTC Currency `json:"BTC"`
	} `json:"quote"`
}

// CryptocurrencyLatestMarketPairs defines the latest cryptocurrency pairs
type CryptocurrencyLatestMarketPairs struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	NumMarketPairs int    `json:"num_market_pairs"`
	MarketPairs    []struct {
		Exchange struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"exchange"`
		MarketPair     string `json:"market_pair"`
		MarketPairBase struct {
			CurrencyID     int    `json:"currency_id"`
			CurrencySymbol string `json:"currency_symbol"`
			CurrencyType   string `json:"currency_type"`
		} `json:"market_pair_base"`
		MarketPairQuote struct {
			CurrencyID     int    `json:"currency_id"`
			CurrencySymbol string `json:"currency_symbol"`
			CurrencyType   string `json:"currency_type"`
		} `json:"market_pair_quote"`
		Quote struct {
			ExchangeReported struct {
				Price          float64   `json:"price"`
				Volume24HBase  float64   `json:"volume_24h_base"`
				Volume24HQuote float64   `json:"volume_24h_quote"`
				LastUpdated    time.Time `json:"last_updated"`
			} `json:"exchange_reported"`
			USD Currency `json:"USD"`
		} `json:"quote"`
	} `json:"market_pairs"`
}

// CryptocurrencyOHLCHistorical defines open high low close historical data
type CryptocurrencyOHLCHistorical struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Quotes []struct {
		TimeOpen  time.Time `json:"time_open"`
		TimeClose time.Time `json:"time_close"`
		Quote     struct {
			USD OHLC `json:"USD"`
		} `json:"quote"`
	} `json:"quotes"`
}

// CryptocurrencyOHLCLatest defines open high low close latest data
type CryptocurrencyOHLCLatest map[string]struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Symbol      string    `json:"symbol"`
	LastUpdated time.Time `json:"last_updated"`
	TimeOpen    time.Time `json:"time_open"`
	TimeClose   any       `json:"time_close"`
	Quote       struct {
		USD OHLC `json:"USD"`
	} `json:"quote"`
}

// CryptocurrencyLatestQuotes defines latest cryptocurrency quotation data
type CryptocurrencyLatestQuotes map[string]struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Symbol            string    `json:"symbol"`
	Slug              string    `json:"slug"`
	CirculatingSupply float64   `json:"circulating_supply"`
	TotalSupply       float64   `json:"total_supply"`
	MaxSupply         float64   `json:"max_supply"`
	DateAdded         time.Time `json:"date_added"`
	NumMarketPairs    int       `json:"num_market_pairs"`
	CmcRank           int       `json:"cmc_rank"`
	LastUpdated       time.Time `json:"last_updated"`
	Tags              []string  `json:"tags"`
	Platform          any       `json:"platform"`
	Quote             struct {
		USD Currency `json:"USD"`
	} `json:"quote"`
}

// CryptocurrencyHistoricalQuotes defines historical cryptocurrency quotation
// data
type CryptocurrencyHistoricalQuotes struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Quotes []struct {
		Timestamp time.Time `json:"timestamp"`
		Quote     struct {
			USD Currency `json:"USD"`
		} `json:"quote"`
	} `json:"quotes"`
}

// ExchangeInfo defines exchange information
type ExchangeInfo map[string]struct {
	Urls struct {
		Website []string `json:"website"`
		Twitter []string `json:"twitter"`
		Blog    []any    `json:"blog"`
		Chat    []string `json:"chat"`
		Fee     []string `json:"fee"`
	} `json:"urls"`
	Logo string `json:"logo"`
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ExchangeMap defines a data for an exchange
type ExchangeMap struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	IsActive            int       `json:"is_active"`
	FirstHistoricalData time.Time `json:"first_historical_data"`
	LastHistoricalData  time.Time `json:"last_historical_data"`
}

// ExchangeHistoricalListings defines historical exchange listings
type ExchangeHistoricalListings struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	CmcRank        int       `json:"cmc_rank"`
	NumMarketPairs int       `json:"num_market_pairs"`
	Timestamp      time.Time `json:"timestamp"`
	Quote          struct {
		USD Currency `json:"USD"`
	} `json:"quote"`
}

// ExchangeLatestListings defines latest exchange listings
type ExchangeLatestListings struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	NumMarketPairs int       `json:"num_market_pairs"`
	LastUpdated    time.Time `json:"last_updated"`
	Quote          struct {
		USD Currency `json:"USD"`
	} `json:"quote"`
}

// ExchangeLatestMarketPairs defines latest market pairs
type ExchangeLatestMarketPairs struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	NumMarketPairs int    `json:"num_market_pairs"`
	MarketPairs    []struct {
		MarketPair     string `json:"market_pair"`
		MarketPairBase struct {
			CurrencyID     int    `json:"currency_id"`
			CurrencySymbol string `json:"currency_symbol"`
			CurrencyType   string `json:"currency_type"`
		} `json:"market_pair_base"`
		MarketPairQuote struct {
			CurrencyID     int    `json:"currency_id"`
			CurrencySymbol string `json:"currency_symbol"`
			CurrencyType   string `json:"currency_type"`
		} `json:"market_pair_quote"`
		Quote struct {
			ExchangeReported struct {
				Price          float64   `json:"price"`
				Volume24HBase  float64   `json:"volume_24h_base"`
				Volume24HQuote float64   `json:"volume_24h_quote"`
				LastUpdated    time.Time `json:"last_updated"`
			} `json:"exchange_reported"`
			USD Currency `json:"USD"`
		} `json:"quote"`
	} `json:"market_pairs"`
}

// ExchangeLatestQuotes defines latest exchange quotations
type ExchangeLatestQuotes struct {
	Binance struct {
		ID             int       `json:"id"`
		Name           string    `json:"name"`
		Slug           string    `json:"slug"`
		NumMarketPairs int       `json:"num_market_pairs"`
		LastUpdated    time.Time `json:"last_updated"`
		Quote          struct {
			USD Currency `json:"USD"`
		} `json:"quote"`
	} `json:"binance"`
}

// ExchangeHistoricalQuotes defines historical exchange quotations
type ExchangeHistoricalQuotes struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Quotes []struct {
		Timestamp time.Time `json:"timestamp"`
		Quote     struct {
			USD Currency `json:"USD"`
		} `json:"quote"`
		NumMarketPairs int `json:"num_market_pairs"`
	} `json:"quotes"`
}

// GlobalMeticLatestQuotes defines latest global metric quotations
type GlobalMeticLatestQuotes struct {
	BtcDominance           float64   `json:"btc_dominance"`
	EthDominance           float64   `json:"eth_dominance"`
	ActiveCryptocurrencies int       `json:"active_cryptocurrencies"`
	ActiveMarketPairs      int       `json:"active_market_pairs"`
	ActiveExchanges        int       `json:"active_exchanges"`
	LastUpdated            time.Time `json:"last_updated"`
	Quote                  struct {
		USD Currency `json:"USD"`
	} `json:"quote"`
}

// GlobalMeticHistoricalQuotes defines historical global metric quotations
type GlobalMeticHistoricalQuotes struct {
	Quotes []struct {
		Timestamp    time.Time `json:"timestamp"`
		BtcDominance float64   `json:"btc_dominance"`
		Quote        struct {
			USD Currency `json:"USD"`
		} `json:"quote"`
	} `json:"quotes"`
}

// PriceConversion defines price conversion data
type PriceConversion struct {
	Symbol      string    `json:"symbol"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Amount      float64   `json:"amount"`
	LastUpdated time.Time `json:"last_updated"`
	Quote       struct {
		GBP Currency `json:"GBP"`
		LTC Currency `json:"LTC"`
		USD Currency `json:"USD"`
	} `json:"quote"`
}
