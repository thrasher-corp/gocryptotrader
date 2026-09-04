package coinmarketcap

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Coinmarketcap account plan bitmasks, url and endpoint consts
const (
	Basic uint8 = 1 << iota
	Builder
	Startup
	Growth
	Professional
	Enterprise
)

const (
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

	// CoinMarketCap publishes current plan limits at https://coinmarketcap.com/api/pricing/.
	rateInterval            = time.Minute
	basicRequestRate        = 50
	builderRequestRate      = 300
	startupRequestRate      = 600
	growthRequestRate       = 750
	professionalRequestRate = 1200
	enterpriseRequestRate   = 1600
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
	Notice       string       `json:"notice"`
}

// Currency defines a generic sub type to capture currency data
type Currency struct {
	Price                     float64   `json:"price"`
	Volume24Hour              float64   `json:"volume_24h"`
	Volume24HourAdjusted      float64   `json:"volume_24h_adjusted"`
	Volume7Day                float64   `json:"volume_7d"`
	Volume30Day               float64   `json:"volume_30d"`
	PercentChange1Hour        float64   `json:"percent_change_1h"`
	PercentChangeVolume24Hour float64   `json:"percent_change_volume_24h"`
	PercentChangeVolume7Day   float64   `json:"percent_change_volume_7d"`
	PercentChangeVolume30Day  float64   `json:"percent_change_volume_30d"`
	MarketCap                 float64   `json:"market_cap"`
	TotalMarketCap            float64   `json:"total_market_cap"`
	LastUpdated               time.Time `json:"last_updated"`
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

// CryptocurrencyLatestListings defines the shared V3 response element used by
// the latest cryptocurrency listings and quotes endpoints.
type CryptocurrencyLatestListings struct {
	ID                            int64                        `json:"id"`
	Name                          string                       `json:"name"`
	Symbol                        string                       `json:"symbol"`
	Slug                          string                       `json:"slug"`
	Platform                      json.RawMessage              `json:"platform"`
	Quote                         CryptocurrencyLatestQuoteMap `json:"quote"`
	Tags                          json.RawMessage              `json:"tags"`
	IsActive                      int64                        `json:"is_active"`
	InfiniteSupply                bool                         `json:"infinite_supply"`
	IsMarketCapIncludedInCalc     int64                        `json:"is_market_cap_included_in_calc"`
	IsFiat                        int64                        `json:"is_fiat"`
	CirculatingSupply             float64                      `json:"circulating_supply"`
	TotalSupply                   float64                      `json:"total_supply"`
	MaxSupply                     float64                      `json:"max_supply"`
	DateAdded                     time.Time                    `json:"date_added"`
	NumMarketPairs                int64                        `json:"num_market_pairs"`
	CmcRank                       int64                        `json:"cmc_rank"`
	LastUpdated                   time.Time                    `json:"last_updated"`
	TVLRatio                      float64                      `json:"tvl_ratio"`
	SelfReportedCirculatingSupply float64                      `json:"self_reported_circulating_supply"`
	SelfReportedMarketCap         float64                      `json:"self_reported_market_cap"`
	UnlockedCirculatingSupply     float64                      `json:"unlocked_circulating_supply"`
	UnlockedMarketCap             float64                      `json:"unlocked_market_cap"`
	// MintedMarketCap is documented on the listings/latest response.
	MintedMarketCap float64 `json:"minted_market_cap"`
}

// CryptocurrencyLatestQuotes defines latest cryptocurrency quotation data
// using the same V3 response element as CryptocurrencyLatestListings.
type CryptocurrencyLatestQuotes []CryptocurrencyLatestListings

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
				Price             float64   `json:"price"`
				Volume24HourBase  float64   `json:"volume_24h_base"`
				Volume24HourQuote float64   `json:"volume_24h_quote"`
				LastUpdated       time.Time `json:"last_updated"`
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

// CryptocurrencyQuote defines a V3 cryptocurrency quote in one conversion
// currency.
type CryptocurrencyQuote struct {
	ID                     int64     `json:"id"`
	Symbol                 string    `json:"symbol"`
	Price                  float64   `json:"price"`
	Volume24Hour           float64   `json:"volume_24h"`
	CEXVolume24Hour        float64   `json:"cex_volume_24h"`
	DEXVolume24Hour        float64   `json:"dex_volume_24h"`
	Volume24HourReported   float64   `json:"volume_24h_reported"`
	Volume7Day             float64   `json:"volume_7d"`
	Volume7DayReported     float64   `json:"volume_7d_reported"`
	Volume30Day            float64   `json:"volume_30d"`
	Volume30DayReported    float64   `json:"volume_30d_reported"`
	VolumeChange24Hour     float64   `json:"volume_change_24h"`
	PercentChange1Hour     float64   `json:"percent_change_1h"`
	PercentChange24Hour    float64   `json:"percent_change_24h"`
	PercentChange7Day      float64   `json:"percent_change_7d"`
	PercentChange30Day     float64   `json:"percent_change_30d"`
	PercentChange60Day     float64   `json:"percent_change_60d"`
	PercentChange90Day     float64   `json:"percent_change_90d"`
	MarketCap              float64   `json:"market_cap"`
	MarketCapDominance     float64   `json:"market_cap_dominance"`
	FullyDilutedMarketCap  float64   `json:"fully_diluted_market_cap"`
	MintedMarketCap        float64   `json:"minted_market_cap"`
	TVL                    float64   `json:"tvl"`
	MarketCapByTotalSupply float64   `json:"market_cap_by_total_supply"`
	LastUpdated            time.Time `json:"last_updated"`
}

// CryptocurrencyLatestQuoteMap captures V3 quote values by conversion symbol.
type CryptocurrencyLatestQuoteMap map[string]CryptocurrencyQuote

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
				Price             float64   `json:"price"`
				Volume24HourBase  float64   `json:"volume_24h_base"`
				Volume24HourQuote float64   `json:"volume_24h_quote"`
				LastUpdated       time.Time `json:"last_updated"`
			} `json:"exchange_reported"`
			USD Currency `json:"USD"`
		} `json:"quote"`
	} `json:"market_pairs"`
}

// ExchangeLatestQuote defines a latest exchange quotation.
type ExchangeLatestQuote struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	NumMarketPairs int64     `json:"num_market_pairs"`
	LastUpdated    time.Time `json:"last_updated"`
	Quote          QuoteMap  `json:"quote"`
}

// ExchangeLatestQuotes defines latest exchange quotations.
type ExchangeLatestQuotes struct {
	Binance struct {
		ID             int64     `json:"id"`
		Name           string    `json:"name"`
		Slug           string    `json:"slug"`
		NumMarketPairs int64     `json:"num_market_pairs"`
		LastUpdated    time.Time `json:"last_updated"`
		Quote          QuoteMap  `json:"quote"`
	} `json:"binance"`
	Exchanges map[string]ExchangeLatestQuote `json:"-"`
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
	ID          int64     `json:"id"`
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

// UnmarshalJSON handles legacy quote payloads that may be either an object map
// or an array of object maps. Both forms merge with a populated receiver, while
// null clears it.
func (q *QuoteMap) UnmarshalJSON(data []byte) error {
	var quotes map[string]Currency
	if *q != nil {
		quotes = make(map[string]Currency, len(*q))
		maps.Copy(quotes, *q)
	}
	if err := json.Unmarshal(data, &quotes); err == nil {
		*q = quotes
		return nil
	}
	var arr []map[string]Currency
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("%w: quote collection: %w", common.ErrInvalidResponse, err)
	}
	for i := range arr {
		if quotes == nil && len(arr[i]) != 0 {
			quotes = make(map[string]Currency, len(arr[i]))
		}
		maps.Copy(quotes, arr[i])
	}
	*q = quotes
	return nil
}

// UnmarshalJSON handles V3 quote payloads that may be either an object map or
// array.
func (q *CryptocurrencyLatestQuoteMap) UnmarshalJSON(data []byte) error {
	var quotes map[string]CryptocurrencyQuote
	if err := json.Unmarshal(data, &quotes); err == nil {
		*q = quotes
		return nil
	}
	var arr []CryptocurrencyQuote
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("%w: cryptocurrency quote collection: %w", common.ErrInvalidResponse, err)
	}
	quotes = make(CryptocurrencyLatestQuoteMap, len(arr))
	for i := range arr {
		symbol := arr[i].Symbol
		if symbol == "" || strings.TrimSpace(symbol) != symbol {
			return fmt.Errorf("%w: cryptocurrency quote at index %d has an invalid symbol", common.ErrInvalidResponse, i)
		}
		if _, exists := quotes[symbol]; exists {
			return fmt.Errorf("%w: duplicate cryptocurrency quote symbol %q", common.ErrInvalidResponse, symbol)
		}
		quotes[symbol] = arr[i]
	}
	*q = quotes
	return nil
}
