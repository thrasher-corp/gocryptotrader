package cryptodotcom

import (
	"errors"
)

var (
	errSymbolIsRequired              = errors.New("symbol is required")
	errInvalidOrderCancellationScope = errors.New("invalid order cancellation scope, only ACCOUNT or CONNECTION is supported")
	errInvalidCurrency               = errors.New("invalid currency")
	errInvalidAmount                 = errors.New("amount has to be greater than zero")
)

// InstrumentsResponse represents instruments response.
type InstrumentsResponse struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Code   int    `json:"code"`
	Result struct {
		Instruments []Instrument `json:"instruments"`
	} `json:"result"`
}

// Instrument represents an details.
type Instrument struct {
	InstrumentName          string  `json:"instrument_name"`
	QuoteCurrency           string  `json:"quote_currency"`
	BaseCurrency            string  `json:"base_currency"`
	PriceDecimals           int     `json:"price_decimals"`
	QuantityDecimals        int     `json:"quantity_decimals"`
	MarginTradingEnabled    bool    `json:"margin_trading_enabled"`
	MarginTradingEnabled5X  bool    `json:"margin_trading_enabled_5x"`
	MarginTradingEnabled10X bool    `json:"margin_trading_enabled_10x"`
	MaxQuantity             string  `json:"max_quantity"`
	MinQuantity             string  `json:"min_quantity"`
	MaxPrice                float64 `json:"max_price,string"`
	MinPrice                float64 `json:"min_price,string"`
	LastUpdateDate          int64   `json:"last_update_date"`
	QuantityTickSize        float64 `json:"quantity_tick_size,string"`
	PriceTickSize           float64 `json:"price_tick_size,string"`
}

// OrderbookDetail public order book detail.
type OrderbookDetail struct {
	Depth int `json:"depth"`
	Data  []struct {
		Asks [][3]string `json:"asks"`
		Bids [][3]string `json:"bids"`
	} `json:"data"`
	InstrumentName string `json:"instrument_name"`
}

// CandlestickDetail candlesticks (k-line data history).
type CandlestickDetail struct {
	InstrumentName string            `json:"instrument_name"`
	Interval       string            `json:"interval"`
	Data           []CandlestickItem `json:"data"`
}

// CandlestickItem candlesticks (k-line data history) item.
type CandlestickItem struct {
	EndTime cryptoDotComMilliSec `json:"t"`
	Open    float64              `json:"o,string"`
	High    float64              `json:"h,string"`
	Low     float64              `json:"l,string"`
	Close   float64              `json:"c,string"`
	Volume  float64              `json:"v,string"`
}

// TickersResponse represents a list of tickers.
type TickersResponse struct {
	Data []TickerItem `json:"data"`
}

// TickerItem represents a ticker item.
type TickerItem struct {
	HighestTradePrice string               `json:"h"` // Price of the 24h highest trade
	LowestTradePrice  string               `json:"l"` // Price of the 24h lowest trade, null if there weren't any trades
	LatestTradePrice  string               `json:"a"` // The price of the latest trade, null if there weren't any trades
	InstrumentName    string               `json:"i"`
	TradedVolume      string               `json:"v"`  // The total 24h traded volume
	TradedVolumeInUSD string               `json:"vv"` // The total 24h traded volume value (in USD)
	OpenInterest      string               `json:"oi"`
	PriceChange       string               `json:"c"` // 24-hour price change, null if there weren't any trades
	BidPriceChange    string               `json:"b"` // The current best bid price, null if there aren't any bids
	BestAskPrice      string               `json:"k"` // The current best ask price, null if there aren't any asks
	TradeTimestamp    cryptoDotComMilliSec `json:"t"`
}

// TradesResponse represents public trades for a particular instrument.
type TradesResponse struct {
	Data []TradeItem `json:"data"`
}

// TradeItem represents a public trade item.
type TradeItem struct {
	Side           string               `json:"s"`
	TradePrice     float64              `json:"p,string"`
	TradeQuantity  float64              `json:"q,string"`
	TradeTimestamp cryptoDotComMilliSec `json:"t"`
	TradeID        string               `json:"d"`
	InstrumentName string               `json:"i"`
	DataTime       cryptoDotComMilliSec `json:"dataTime"`
}

// CancelOnDisconnectScope represents a scope of cancellation.
type CancelOnDisconnectScope struct {
	Scope string `json:"scope"`
}

// PrivateRequestParam represents a generalized private request parameter.
type PrivateRequestParam struct {
	ID        int64                  `json:"id"`
	Method    string                 `json:"method"`
	APIKey    string                 `json:"api_key,omitempty"`
	Params    map[string]interface{} `json:"params"`
	Nonce     int64                  `json:"nonce"`
	Signature string                 `json:"sig"`
}

// CurrencyNetworkResponse retrives the symbol network mapping.
type CurrencyNetworkResponse struct {
	UpdateTime  int64 `json:"update_time"`
	CurrencyMap map[string]struct {
		FullName       string              `json:"full_name"`
		DefaultNetwork string              `json:"default_network"`
		NetworkList    []NetworkListDetail `json:"network_list"`
	} `json:"currency_map"`
}

// NetworkListDetail represents a network list detail.
type NetworkListDetail struct {
	NetworkID            string  `json:"network_id"`
	WithdrawalFee        float64 `json:"withdrawal_fee"`
	WithdrawEnabled      bool    `json:"withdraw_enabled"`
	MinWithdrawalAmount  float64 `json:"min_withdrawal_amount"`
	DepositEnabled       bool    `json:"deposit_enabled"`
	ConfirmationRequired int     `json:"confirmation_required"`
}

// WithdrawalResponse represents a list of withdrawal notifications.
type WithdrawalResponse struct {
	WithdrawalList []WithdrawalItem `json:"withdrawal_list"`
}

// WithdrawalItem represents a withdrawal instance item.
type WithdrawalItem struct {
	Currency   string  `json:"currency"`
	Fee        float64 `json:"fee"`
	ID         string  `json:"id"`
	UpdateTime int64   `json:"update_time"`
	Amount     int     `json:"amount"`
	Address    string  `json:"address"`
	Status     string  `json:"status"`
	Txid       string  `json:"txid"`
	NetworkID  string  `json:"network_id"`

	Symbol             string               `json:"symbol"`
	ClientWithdrawalID string               `json:"client_wid"` // client generated withdrawal id.
	CreateTime         cryptoDotComMilliSec `json:"create_time"`
}

// DepositResponse represents accounts list of deposit funds.
type DepositResponse struct {
	DepositList []DepositItem `json:"deposit_list"`
}

// DepositItem represents accounts deposit item
type DepositItem struct {
	Currency   string  `json:"currency"`
	Fee        float64 `json:"fee"`
	CreateTime int64   `json:"create_time"`
	ID         string  `json:"id"`
	UpdateTime int64   `json:"update_time"`
	Amount     float64 `json:"amount"`
	Address    string  `json:"address"`
	Status     string  `json:"status"`
}

// DepositAddresses represents a list of deposit address.
type DepositAddresses struct {
	DepositAddressList []DepositAddress `json:"deposit_address_list"`
}

// DepositAddress represents a single deposit address item.
type DepositAddress struct {
	Currency   string               `json:"currency"`
	CreateTime cryptoDotComMilliSec `json:"create_time"`
	ID         string               `json:"id"`
	Address    string               `json:"address"`
	Status     string               `json:"status"`
	Network    string               `json:"network"`
}

// Accounts represents list of currency account.
type Accounts struct {
	Accounts []AccountItem `json:"accounts"`
}

// AccountItem represents a single currency account and balance detailed information.
type AccountItem struct {
	Balance   float64 `json:"balance"`
	Available float64 `json:"available"`
	Order     float64 `json:"order"`
	Stake     int     `json:"stake"`
	Currency  string  `json:"currency"`
}
