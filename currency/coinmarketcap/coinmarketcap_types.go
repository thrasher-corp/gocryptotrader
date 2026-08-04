package coinmarketcap

import (
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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

	endpointCryptocurrencyInfo            = "v2/cryptocurrency/info"
	endpointCryptocurrencyMap             = "v1/cryptocurrency/map"
	endpointCryptocurrencyLatestListings  = "v3/cryptocurrency/listings/latest"
	endpointCryptocurrencyMarketPairs     = "v2/cryptocurrency/market-pairs/latest"
	endpointOHLCVHistorical               = "v2/cryptocurrency/ohlcv/historical"
	endpointOHLCVLatest                   = "v2/cryptocurrency/ohlcv/latest"
	endpointGetMarketQuotesHistorical     = "v3/cryptocurrency/quotes/historical"
	endpointGetMarketQuotesLatest         = "v3/cryptocurrency/quotes/latest"
	endpointExchangeInfo                  = "v1/exchange/info"
	endpointExchangeMap                   = "v1/exchange/map"
	endpointExchangeMarketPairsLatest     = "v1/exchange/market-pairs/latest"
	endpointExchangeMarketQuoteHistorical = "v1/exchange/quotes/historical"
	endpointExchangeMarketQuoteLatest     = "v1/exchange/quotes/latest"
	endpointGlobalQuoteHistorical         = "v1/global-metrics/quotes/historical"
	endpointGlobalQuoteLatest             = "v1/global-metrics/quotes/latest"
	endpointPriceConversion               = "v2/tools/price-conversion"

	defaultTimeOut = time.Second * 15

	rateInterval            = time.Minute // BASIC, HOBBYIST STARTUP tier rate limits
	basicRequestRate        = 30
	standardRequestRate     = 60  // STANDARD tier rate limit
	professionalRequestRate = 90  // PROFESSIONAL tier rate limit
	enterpriseRequestRate   = 120 // ENTERPRISE tier rate limit - Can be extended checkout agreement
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
	Timestamp    string       `json:"timestamp"`
	ErrorCode    APIErrorCode `json:"error_code"`
	ErrorMessage string       `json:"error_message"`
	Elapsed      int64        `json:"elapsed"`
	CreditCount  int64        `json:"credit_count"`
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
	ID       int64           `json:"id"`
	Name     string          `json:"name"`
	Symbol   string          `json:"symbol"`
	Category string          `json:"category"`
	Slug     string          `json:"slug"`
	Logo     string          `json:"logo"`
	Tags     json.RawMessage `json:"tags"`
	Platform json.RawMessage `json:"platform"`
	Urls     struct {
		Website      []string        `json:"website"`
		Explorer     []string        `json:"explorer"`
		SourceCode   []string        `json:"source_code"`
		MessageBoard []string        `json:"message_board"`
		Chat         json.RawMessage `json:"chat"`
		Announcement json.RawMessage `json:"announcement"`
		Reddit       []string        `json:"reddit"`
		Twitter      []string        `json:"twitter"`
	} `json:"urls"`
}

// CryptoCurrencyMap defines a cryptocurrency struct
type CryptoCurrencyMap struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	Symbol              string    `json:"symbol"`
	Slug                string    `json:"slug"`
	IsActive            int64     `json:"is_active"`
	FirstHistoricalData time.Time `json:"first_historical_data"`
	LastHistoricalData  time.Time `json:"last_historical_data"`
	Platform            struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		Symbol       string `json:"symbol"`
		Slug         string `json:"slug"`
		TokenAddress string `json:"token_address"`
	} `json:"platform"`
}

// CryptocurrencyHistoricalListings defines a historical listing data
type CryptocurrencyHistoricalListings struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Symbol            string    `json:"symbol"`
	Slug              string    `json:"slug"`
	CmcRank           int64     `json:"cmc_rank"`
	NumMarketPairs    int64     `json:"num_market_pairs"`
	CirculatingSupply float64   `json:"circulating_supply"`
	TotalSupply       float64   `json:"total_supply"`
	MaxSupply         float64   `json:"max_supply"`
	LastUpdated       time.Time `json:"last_updated"`
	Quote             QuoteMap  `json:"quote"`
}

// CryptocurrencyLatestListings defines latest cryptocurrency listing data
type CryptocurrencyLatestListings struct {
	ID                int64           `json:"id"`
	Name              string          `json:"name"`
	Symbol            string          `json:"symbol"`
	Slug              string          `json:"slug"`
	CmcRank           int64           `json:"cmc_rank"`
	NumMarketPairs    int64           `json:"num_market_pairs"`
	CirculatingSupply float64         `json:"circulating_supply"`
	TotalSupply       float64         `json:"total_supply"`
	MaxSupply         float64         `json:"max_supply"`
	LastUpdated       time.Time       `json:"last_updated"`
	DateAdded         time.Time       `json:"date_added"`
	Tags              json.RawMessage `json:"tags"`
	Platform          json.RawMessage `json:"platform"`
	Quote             QuoteMap        `json:"quote"`
}

// CryptocurrencyLatestMarketPairs defines the latest cryptocurrency pairs
type CryptocurrencyLatestMarketPairs struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	NumMarketPairs int64  `json:"num_market_pairs"`
	MarketPairs    []struct {
		Exchange struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"exchange"`
		MarketPair     string `json:"market_pair"`
		MarketPairBase struct {
			CurrencyID     int64  `json:"currency_id"`
			CurrencySymbol string `json:"currency_symbol"`
			CurrencyType   string `json:"currency_type"`
		} `json:"market_pair_base"`
		MarketPairQuote struct {
			CurrencyID     int64  `json:"currency_id"`
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
	ID     int64  `json:"id"`
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
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Symbol      string          `json:"symbol"`
	LastUpdated time.Time       `json:"last_updated"`
	TimeOpen    time.Time       `json:"time_open"`
	TimeClose   json.RawMessage `json:"time_close"`
	Quote       struct {
		USD OHLC `json:"USD"`
	} `json:"quote"`
}

// CryptocurrencyLatestQuotes defines latest cryptocurrency quotation data.
type CryptocurrencyLatestQuotes []struct {
	ID                int64           `json:"id"`
	Name              string          `json:"name"`
	Symbol            string          `json:"symbol"`
	Slug              string          `json:"slug"`
	CirculatingSupply float64         `json:"circulating_supply"`
	TotalSupply       float64         `json:"total_supply"`
	MaxSupply         float64         `json:"max_supply"`
	DateAdded         time.Time       `json:"date_added"`
	NumMarketPairs    int64           `json:"num_market_pairs"`
	CmcRank           int64           `json:"cmc_rank"`
	LastUpdated       time.Time       `json:"last_updated"`
	Tags              json.RawMessage `json:"tags"`
	Platform          json.RawMessage `json:"platform"`
	Quote             QuoteMap        `json:"quote"`
}

// CryptocurrencyHistoricalQuotes defines historical cryptocurrency quotation
// data
type CryptocurrencyHistoricalQuotes struct {
	ID     int64  `json:"id"`
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
		Website []string        `json:"website"`
		Twitter []string        `json:"twitter"`
		Blog    json.RawMessage `json:"blog"`
		Chat    []string        `json:"chat"`
		Fee     []string        `json:"fee"`
	} `json:"urls"`
	Logo string `json:"logo"`
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ExchangeMap defines a data for an exchange
type ExchangeMap struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	IsActive            int64     `json:"is_active"`
	FirstHistoricalData time.Time `json:"first_historical_data"`
	LastHistoricalData  time.Time `json:"last_historical_data"`
}

// ExchangeHistoricalListings defines historical exchange listings
type ExchangeHistoricalListings struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	CmcRank        int64     `json:"cmc_rank"`
	NumMarketPairs int64     `json:"num_market_pairs"`
	Timestamp      time.Time `json:"timestamp"`
	Quote          struct {
		USD Currency `json:"USD"`
	} `json:"quote"`
}

// ExchangeLatestListings defines latest exchange listings
type ExchangeLatestListings struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	NumMarketPairs int64     `json:"num_market_pairs"`
	LastUpdated    time.Time `json:"last_updated"`
	Quote          struct {
		USD Currency `json:"USD"`
	} `json:"quote"`
}

// ExchangeLatestMarketPairs defines latest market pairs
type ExchangeLatestMarketPairs struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	NumMarketPairs int64  `json:"num_market_pairs"`
	MarketPairs    []struct {
		MarketPair     string `json:"market_pair"`
		MarketPairBase struct {
			CurrencyID     int64  `json:"currency_id"`
			CurrencySymbol string `json:"currency_symbol"`
			CurrencyType   string `json:"currency_type"`
		} `json:"market_pair_base"`
		MarketPairQuote struct {
			CurrencyID     int64  `json:"currency_id"`
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
		ID             int64     `json:"id"`
		Name           string    `json:"name"`
		Slug           string    `json:"slug"`
		NumMarketPairs int64     `json:"num_market_pairs"`
		LastUpdated    time.Time `json:"last_updated"`
		Quote          QuoteMap  `json:"quote"`
	} `json:"binance"`
}

// ExchangeHistoricalQuotes defines historical exchange quotations
type ExchangeHistoricalQuotes struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Quotes []struct {
		Timestamp      time.Time `json:"timestamp"`
		Quote          QuoteMap  `json:"quote"`
		NumMarketPairs int64     `json:"num_market_pairs"`
	} `json:"quotes"`
}

// GlobalMeticLatestQuotes defines latest global metric quotations
type GlobalMeticLatestQuotes struct {
	BtcDominance           float64   `json:"btc_dominance"`
	EthDominance           float64   `json:"eth_dominance"`
	ActiveCryptocurrencies int64     `json:"active_cryptocurrencies"`
	ActiveMarketPairs      int64     `json:"active_market_pairs"`
	ActiveExchanges        int64     `json:"active_exchanges"`
	LastUpdated            time.Time `json:"last_updated"`
	Quote                  QuoteMap  `json:"quote"`
}

// GlobalMeticHistoricalQuotes defines historical global metric quotations
type GlobalMeticHistoricalQuotes struct {
	Quotes []struct {
		Timestamp    time.Time `json:"timestamp"`
		BtcDominance float64   `json:"btc_dominance"`
		Quote        QuoteMap  `json:"quote"`
	} `json:"quotes"`
}

// PriceConversion defines price conversion data
type PriceConversion struct {
	Symbol      string    `json:"symbol"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Amount      float64   `json:"amount"`
	LastUpdated time.Time `json:"last_updated"`
	Quote       QuoteMap  `json:"quote"`
}

// QuoteMap captures quote values for all returned conversion symbols.
type QuoteMap map[string]Currency

// APIErrorCode supports status error code decoding from either number or string.
type APIErrorCode int64

// UnmarshalJSON decodes error code from number or quoted string.
func (c *APIErrorCode) UnmarshalJSON(data []byte) error {
	var num int64
	if err := json.Unmarshal(data, &num); err == nil {
		*c = APIErrorCode(num)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse error code: %w", err)
	}
	*c = APIErrorCode(parsed)
	return nil
}

// UnmarshalJSON handles quote payloads that may be either an object map or array.
func (q *QuoteMap) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, (*map[string]Currency)(q)); err == nil {
		return nil
	}
	var arr []map[string]Currency
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	for i := range arr {
		maps.Copy(*q, arr[i])
	}
	return nil
}
