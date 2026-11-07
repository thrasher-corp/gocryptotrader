package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AppFiatCurrencyResponse is a fiat currency the Coinbase App knows of, by its ISO 4217 code where it has one
type AppFiatCurrencyResponse struct {
	ID          currency.Code `json:"id"`
	Name        string        `json:"name"`
	MinimumSize types.Number  `json:"min_size"`
}

// AppCurrencyResponse is a currency as the Coinbase App describes it: a cryptocurrency Get Cryptocurrencies lists, or
// an account's currency
type AppCurrencyResponse struct {
	AssetID string        `json:"asset_id"`
	Code    currency.Code `json:"code"`
	Name    string        `json:"name"`
	Colour  string        `json:"color"`
	// SortIndex orders currencies as the Coinbase App lists them
	SortIndex uint64 `json:"sort_index"`
	// Exponent is how many decimal places the currency's amounts have
	Exponent uint64 `json:"exponent"`
	Type     string `json:"type"`
	// AddressRegex matches the currency's addresses on its default network
	AddressRegex string `json:"address_regex"`
	// DestinationTagName and DestinationTagRegex name and match the destination tag or memo a currency's sends take,
	// for the few currencies that take one; Get Cryptocurrencies sends them undocumented
	DestinationTagName  string `json:"destination_tag_name"`
	DestinationTagRegex string `json:"destination_tag_regex"`
	// Slug is the currency's name in Coinbase's URLs, which accounts' currencies hold
	Slug string `json:"slug"`
	// Rewards is undocumented and has no documented fields, so it is kept as sent
	Rewards json.RawMessage `json:"rewards"`
}

// AppExchangeRatesResponse holds what one unit of a base currency is worth in each currency Coinbase prices
type AppExchangeRatesResponse struct {
	Currency currency.Code                  `json:"currency"`
	Rates    map[currency.Code]types.Number `json:"rates"`
}

// AppPriceResponse is the price of one unit of Base in Currency
type AppPriceResponse struct {
	Amount types.Number `json:"amount"`
	// Base is undocumented
	Base     currency.Code `json:"base"`
	Currency currency.Code `json:"currency"`
}

// AppSpotPriceRequest selects a pair's spot price, now or on a past day
type AppSpotPriceRequest struct {
	Pair currency.Pair
	// Date selects a past UTC day's spot price; the current one when it is zero
	Date time.Time
}

// AppServerTimeResponse is the Coinbase App API's clock
type AppServerTimeResponse struct {
	ISO   time.Time  `json:"iso"`
	Epoch types.Time `json:"epoch"`
}
