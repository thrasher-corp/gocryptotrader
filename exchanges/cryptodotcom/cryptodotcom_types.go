package cryptodotcom

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errInvalidOrderCancellationScope   = errors.New("invalid order cancellation scope, only ACCOUNT or CONNECTION is supported")
	errInvalidResponseFromServer       = errors.New("invalid response from server")
	errInstrumentNameOrOrderIDRequired = errors.New("either InstrumentName or OrderID is required")
	errTriggerPriceRequired            = errors.New("trigger price is required")
	errSubAccountAddressRequired       = errors.New("sub-account address is required")
	errRequestedDataTypesRequired      = errors.New("requested data types are required")
	errQuoteIDRequired                 = errors.New("missing quote ID")
	errAddressRequired                 = errors.New("address is required")
	errInvalidRate                     = errors.New("invalid rate")
	errInvalidSlippageToleraceBPs      = errors.New("invalid slippage tolerance base points")
	errValuationTypeUnset              = errors.New("valuation type unset")
	errAccountIDMissing                = errors.New("account id is required")
	errContingencyTypeRequired         = errors.New("contingency type is required")
	errPriceBelowMin                   = errors.New("price below min")
	errSTPInstructionIsRequired        = errors.New("self-trade-prevention instruction (stpInstruction) is missing")
	errPaymentNetworkIsMissing         = errors.New("payment network is missing") // possible payment networks: https://exchange-docs.crypto.com/exchange/v1/rest-ws/index.html#payment-networks-and-currencies
)

const (
	tifGTC      = "GOOD_TILL_CANCEL"
	tifPOSTONLY = "POST_ONLY"
	tifIOC      = "IMMEDIATE_OR_CANCEL"
	tifFOK      = "FILL_OR_KILL"
)

// OrderbookDetail public order book detail.
type OrderbookDetail struct {
	Depth int64 `json:"depth"`
	Data  []struct {
		Asks [][3]types.Number `json:"asks"`
		Bids [][3]types.Number `json:"bids"`
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
	EndTime types.Time `json:"t"` // this represents Start Time for websocket push data.
	Open    float64    `json:"o,string"`
	High    float64    `json:"h,string"`
	Low     float64    `json:"l,string"`
	Close   float64    `json:"c,string"`
	Volume  float64    `json:"v,string"`
}

// WsCandlestickItem represents candlestick (k-line data history) item pushed through the websocket connection.
type WsCandlestickItem struct {
	CandlestickItem
	// Added for websocket push data
	UpdateTime types.Time `json:"ut"` // this represents Update Time for websocket push data.
}

// OTCBook represents an orderbook data for OTC instrument.
type OTCBook struct {
	Asks [][5]types.Number `json:"asks"` // Price, Total Size, Number of Orders in the level, Expiry Time, Unique ID
	Bids [][5]types.Number `json:"bids"` // Price, Total Size, Number of Orders in the level, Expiry Time, Unique ID
}

// TickersResponse represents a list of tickers.
type TickersResponse struct {
	Data []TickerItem `json:"data"`
}

// InstrumentValuation represents a particular instrument valuation.
type InstrumentValuation struct {
	Data []struct {
		Value     types.Number `json:"v"`
		Timestamp types.Time   `json:"t"`
	} `json:"data"`
	InstrumentName string `json:"instrument_name"`
}

// TickerItem represents a ticker item.
type TickerItem struct {
	InstrumentName       string     `json:"i"`
	HighestTradePrice    SafeNumber `json:"h"`  // Price of the 24h highest trade
	LowestTradePrice     SafeNumber `json:"l"`  // Price of the 24h lowest trade, null if there weren't any trades
	LatestTradePrice     SafeNumber `json:"a"`  // The price of the latest trade, null if there weren't any trades
	TradedVolume         SafeNumber `json:"v"`  // The total 24h traded volume
	TradedVolumeInUSD24H SafeNumber `json:"vv"` // The total 24h traded volume value (in USD)
	OpenInterest         string     `json:"oi"`
	PriceChange24H       SafeNumber `json:"c"` // 24-hour price change, null if there weren't any trades
	BestBidPrice         SafeNumber `json:"b"` // The current best bid price, null if there aren't any bids
	BestAskPrice         SafeNumber `json:"k"` // The current best ask price, null if there aren't any asks
	TradeTimestamp       types.Time `json:"t"`

	// Added for websocket push data.
	BestBidSize SafeNumber `json:"bs"`
	BestAskSize SafeNumber `json:"ks"`
}

// TradesResponse represents public trades for a particular instrument.
type TradesResponse struct {
	Data []TradeItem `json:"data"`
}

// TradeItem represents a public trade item.
type TradeItem struct {
	Side             string     `json:"s"`
	TradePrice       SafeNumber `json:"p"`
	TradeQuantity    SafeNumber `json:"q"`
	TradeTimestamp   types.Time `json:"t"`
	TradeID          string     `json:"d"`
	InstrumentName   string     `json:"i"`
	TradeTimestampNs types.Time `json:"tn"`
	TradeMatchID     string     `json:"m"`
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

// CurrencyNetworkResponse retrieves the symbol network mapping.
type CurrencyNetworkResponse struct {
	UpdateTime  types.Time `json:"update_time"`
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
	ConfirmationRequired int64   `json:"confirmation_required"`
}

// WithdrawalResponse represents a list of withdrawal notifications.
type WithdrawalResponse struct {
	WithdrawalList []WithdrawalItem `json:"withdrawal_list"`
}

// WithdrawalItem represents a withdrawal instance item.
type WithdrawalItem struct {
	Currency           string     `json:"currency"`
	Fee                float64    `json:"fee"`
	ID                 string     `json:"id"`
	UpdateTime         types.Time `json:"update_time"`
	Amount             float64    `json:"amount"`
	Address            string     `json:"address"`
	Status             string     `json:"status"`
	TransactionID      string     `json:"txid"`
	NetworkID          string     `json:"network_id"`
	Symbol             string     `json:"symbol"`
	ClientWithdrawalID string     `json:"client_wid"` // client generated withdrawal id.
	CreateTime         types.Time `json:"create_time"`
}

// DepositResponse represents accounts list of deposit funds.
type DepositResponse struct {
	DepositList []DepositItem `json:"deposit_list"`
}

// DepositItem represents accounts deposit item
type DepositItem struct {
	Currency   string     `json:"currency"`
	Fee        float64    `json:"fee"`
	ID         string     `json:"id"`
	CreateTime types.Time `json:"create_time"`
	UpdateTime types.Time `json:"update_time"`
	Amount     float64    `json:"amount"`
	Address    string     `json:"address"`
	Status     string     `json:"status"`
}

// DepositAddresses represents a list of deposit address.
type DepositAddresses struct {
	DepositAddressList []DepositAddress `json:"deposit_address_list"`
}

// ExportRequestResponse represents a response after creating an instrument export request.
type ExportRequestResponse struct {
	ID              int64  `json:"id"`
	Status          string `json:"status"`
	ClientRequestID string `json:"client_request_id"`
}

// ExportRequests represents a list of export requests
type ExportRequests struct {
	UserBatchList []struct {
		ID              string     `json:"id"`
		StartTime       types.Time `json:"start_ts"`
		EndTime         types.Time `json:"end_ts"`
		InstrumentNames []string   `json:"instrument_names"`
		RequestedData   []string   `json:"requested_data"`
		ClientRequestID string     `json:"client_request_id"`
		Status          string     `json:"status"`
	} `json:"user_batch_list"`
}

// DepositAddress represents a single deposit address item.
type DepositAddress struct {
	Currency   string     `json:"currency"`
	CreateTime types.Time `json:"create_time"`
	ID         string     `json:"id"`
	Address    string     `json:"address"`
	Status     string     `json:"status"`
	Network    string     `json:"network"`
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
	Stake     float64 `json:"stake"`
	Currency  string  `json:"currency"`
}

// CreateOrderResponse represents a response for a new BUY or SELL order on the Exchange.
type CreateOrderResponse struct {
	OrderID   string `json:"order_id"`
	ClientOid string `json:"client_oid"`
}

// PersonalTrades represents a personal trade list response.
type PersonalTrades struct {
	TradeList []PersonalTradeItem `json:"trade_list"`
}

// PersonalTradeItem represents a personal trade item instance.
type PersonalTradeItem struct {
	Side           string     `json:"side"`
	InstrumentName string     `json:"instrument_name"`
	Fee            float64    `json:"fee"`
	TradeID        string     `json:"trade_id"`
	CreateTime     types.Time `json:"create_time"`
	TradedPrice    float64    `json:"traded_price"`
	TradedQuantity float64    `json:"traded_quantity"`
	FeeCurrency    string     `json:"fee_currency"`
	OrderID        string     `json:"order_id"`
}

// OrderAndTrades represents an order detail.
type OrderAndTrades struct {
	TradeList []struct {
		Side           string     `json:"side"`
		InstrumentName string     `json:"instrument_name"`
		Fee            float64    `json:"fee"`
		TradeID        string     `json:"trade_id"`
		CreateTime     types.Time `json:"create_time"`
		TradedPrice    float64    `json:"traded_price"`
		TradedQuantity float64    `json:"traded_quantity"`
		FeeCurrency    string     `json:"fee_currency"`
		OrderID        string     `json:"order_id"`
	} `json:"trade_list"`
	OrderInfo OrderItem `json:"order_info"`
}

// OrderItem represents order instance detail information.
type OrderItem struct {
	Status             string            `json:"status"`
	Side               string            `json:"side"`
	OrderID            string            `json:"order_id"`
	ClientOid          string            `json:"client_oid"`
	CreateTime         types.Time        `json:"create_time"`
	UpdateTime         types.Time        `json:"update_time"`
	Type               string            `json:"type"`
	InstrumentName     string            `json:"instrument_name"`
	CumulativeQuantity float64           `json:"cumulative_quantity"`
	CumulativeValue    float64           `json:"cumulative_value"`
	AvgPrice           float64           `json:"avg_price"`
	FeeCurrency        string            `json:"fee_currency"`
	TimeInForce        order.TimeInForce `json:"time_in_force"`
	ExecInst           string            `json:"exec_inst"`
	Price              float64           `json:"price"`
	Quantity           float64           `json:"quantity"`
}

// PersonalOrdersResponse represents a personal order.
type PersonalOrdersResponse struct {
	Count     int64       `json:"count,omitempty"`
	OrderList []OrderItem `json:"order_list"`
}

// OrderParam represents a create order request parameter.
type OrderParam struct {
	Symbol                        string     `json:"instrument_name"`
	Side                          order.Side `json:"side"`
	OrderType                     order.Type `json:"type"`
	Price                         float64    `json:"price"`
	Quantity                      float64    `json:"quantity"`
	Notional                      float64    `json:"notional"`
	ClientOrderID                 string     `json:"client_oid"`
	TimeInForce                   string     `json:"time_in_force"`
	PostOnly                      bool       `json:"exec_inst"`
	TriggerPrice                  float64    `json:"ref_price,omitempty"`
	TriggerPriceType              string     `json:"ref_price_type,omitempty"`
	SpotMargin                    asset.Item `json:"spot_margin,omitempty"`
	SelfTradePreventionScope      asset.Item `json:"stp_scope,omitempty"`
	SelfTradePreventionInstrument string     `json:"stp_inst"`
	SelfTradePreventionID         string     `json:"stp_id"`
	FeeInstrumentName             string     `json:"fee_instrument_name,omitempty"`
}

// AmendOrderParam holds order updating request parameters
type AmendOrderParam struct {
	ClientOrderID         string  `json:"client_oid,omitempty"`
	OrderID               string  `json:"order_id,omitempty"`
	OriginalClientOrderID string  `json:"orig_client_oid,omitempty"`
	NewPrice              float64 `json:"new_price"`
	NewQuantity           float64 `json:"new_quantity"`
}

func (arg *OrderParam) getCreateParamMap() (map[string]interface{}, error) {
	if arg == nil {
		return nil, fmt.Errorf("%w, CreateOrderParam can not be nil", common.ErrNilPointer)
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side != order.Sell && arg.Side != order.Buy {
		return nil, fmt.Errorf("%w, side: %s", order.ErrSideIsInvalid, arg.Side)
	}
	switch arg.OrderType {
	case order.Limit, order.StopLimit, order.TakeProfitLimit:
		if arg.Price <= 0 { // Unit price
			return nil, fmt.Errorf("%w, price must be non-zero positive decimal value", order.ErrPriceMustBeSetIfLimitOrder)
		}
		if arg.Quantity <= 0 {
			return nil, order.ErrAmountIsInvalid
		}
		switch arg.OrderType {
		case order.StopLimit, order.TakeProfitLimit:
			if arg.TriggerPrice <= 0 {
				return nil, fmt.Errorf("%w for Order Type: %v", errTriggerPriceRequired, arg.OrderType)
			}
		}
	case order.Market:
		if arg.Side == order.Buy {
			if arg.Notional <= 0 && arg.Quantity <= 0 {
				return nil, fmt.Errorf("%w, either notional or quantity must be non-zero value for order type: %v and order side: %v", order.ErrAmountMustBeSet, arg.OrderType, arg.Side)
			}
		} else {
			if arg.Quantity <= 0 {
				return nil, fmt.Errorf("%w order type: %v and order side: %v", order.ErrAmountIsInvalid, arg.OrderType, arg.Side)
			}
		}
	case order.Stop, order.TakeProfit:
		if arg.Side == order.Sell {
			if arg.Quantity <= 0 {
				return nil, fmt.Errorf("%w order type: %v and order side: %v", order.ErrAmountIsInvalid, arg.OrderType, arg.Side)
			}
		} else {
			if arg.Notional <= 0 {
				return nil, fmt.Errorf("%w, notional must be non-zero positive decimal value for order type: %v", order.ErrAmountMustBeSet, arg.OrderType)
			}
		}
		if arg.TriggerPrice <= 0 {
			return nil, fmt.Errorf("%w for Order Type: %s", errTriggerPriceRequired, arg.OrderType)
		}
	default:
		return nil, fmt.Errorf("%w, unsupported order type: %v", order.ErrTypeIsInvalid, arg.OrderType)
	}
	params := make(map[string]interface{})
	params["instrument_name"] = arg.Symbol
	params["side"] = arg.Side.String()
	params["type"] = OrderTypeToString(arg.OrderType)
	params["price"] = arg.Price
	if arg.Quantity > 0 {
		params["quantity"] = arg.Quantity
	}
	if arg.Notional > 0 {
		params["notional"] = arg.Notional
	}
	if arg.ClientOrderID != "" {
		params["client_oid"] = arg.ClientOrderID
	}
	arg.TimeInForce = strings.ToUpper(arg.TimeInForce)
	if arg.TimeInForce != "" {
		params["time_in_force"] = arg.TimeInForce
	}
	if arg.PostOnly {
		if arg.TimeInForce != tifGTC {
			return nil, errors.New("execution type POST_ONLY is only allowed with GOOD_TILL_CANCEL time-in-force")
		}
		params["exec_inst"] = []string{tifPOSTONLY}
	}
	if arg.TriggerPrice > 0 {
		params["trigger_price"] = arg.TriggerPrice
	}
	return params, nil
}

// CancelOrderParam represents the parameters to cancel an existing order.
type CancelOrderParam struct {
	InstrumentName string
	OrderID        string
}

// OrderCreationResponse represents list of order creation result information.
type OrderCreationResponse struct {
	ResultList []OrderCreationResultItem `json:"result_list"`
}

// CancelOrdersResponse represents list of cancel orders response.
type CancelOrdersResponse struct {
	ResultList []struct {
		Index int64 `json:"index"`
		Code  int64 `json:"code"`
	} `json:"result_list"`
}

// OrderCreationResultItem represents order creation result Item.
// This represents single order information.
type OrderCreationResultItem struct {
	Index     int64  `json:"index"`
	Code      int64  `json:"code"`
	OrderID   string `json:"order_id"`
	ClientOid string `json:"client_oid"`
}

// AccountResponse represents main and sub account detail information
type AccountResponse struct {
	MasterAccount  AccountInfo   `json:"master_account"`
	SubAccountList []AccountInfo `json:"sub_account_list"`
}

// AccountInfo represents the account information.
type AccountInfo struct {
	UUID              string     `json:"uuid"`
	MasterAccountUUID string     `json:"master_account_uuid"`
	MarginAccountUUID string     `json:"margin_account_uuid"`
	Enabled           bool       `json:"enabled"`
	Tradable          bool       `json:"tradable"`
	Name              string     `json:"name"`
	Email             string     `json:"email"`
	MobileNumber      string     `json:"mobile_number"`
	CountryCode       string     `json:"country_code"`
	Address           string     `json:"address"`
	MarginAccess      string     `json:"margin_access"`
	DerivativesAccess string     `json:"derivatives_access"`
	CreateTime        types.Time `json:"create_time"`
	UpdateTime        types.Time `json:"update_time"`
	TwoFaEnabled      bool       `json:"two_fa_enabled"`
	KycLevel          string     `json:"kyc_level"`
	Suspended         bool       `json:"suspended"`
	Terminated        bool       `json:"terminated"`
	Label             string     `json:"label"`
}

// TransactionResponse represents a transaction response.
type TransactionResponse struct {
	Data []TransactionItem `json:"data"`
}

// TransactionItem represents a transaction instance.
type TransactionItem struct {
	OrderID             string     `json:"order_id,omitempty"`
	AccountID           string     `json:"account_id"`
	TradeMatchID        string     `json:"trade_match_id"`
	TradeID             string     `json:"trade_id,omitempty"`
	EventDate           string     `json:"event_date"` // format 2021-02-18
	JournalType         string     `json:"journal_type"`
	JournalID           string     `json:"journal_id"`
	TransactionCost     SafeNumber `json:"transaction_cost"`
	TransactionQuantity SafeNumber `json:"transaction_qty"`
	RealizedPnl         SafeNumber `json:"realized_pnl"`
	EventTimestampMs    types.Time `json:"event_timestamp_ms"` // Event timestamp in milliseconds
	EventTimestampNs    types.Time `json:"event_timestamp_ns"` // Event timestamp in nanoseconds
	ClientOrderID       string     `json:"client_oid"`
	TakerSide           string     `json:"taker_side"`
	Side                string     `json:"side,omitempty"`
	InstrumentName      string     `json:"instrument_name"`
}

// InstrumentFeeRate represents instruments fee rate detail
type InstrumentFeeRate struct {
	InstrumentName        string       `json:"instrument_name"`
	EffectiveMakerRateBps types.Number `json:"effective_maker_rate_bps"`
	EffectiveTakerRateBps types.Number `json:"effective_taker_rate_bps"`
}

// OTCTrade represents an OTC trade.
type OTCTrade struct {
	AccountUUID         string     `json:"account_uuid"`
	RequestsPerMinute   int64      `json:"requests_per_minute"`
	MaxTradeValueUSD    SafeNumber `json:"max_trade_value_usd"`
	MinTradeValueUSD    SafeNumber `json:"min_trade_value_usd"`
	AcceptOtcTcDatetime types.Time `json:"accept_otc_tc_datetime"`
}

// OTCInstrumentsResponse represents an OTC instruments instance.
type OTCInstrumentsResponse struct {
	InstrumentList []OTCInstrument `json:"instrument_list"`
}

// OTCInstrument represents an OTC instrument.
type OTCInstrument struct {
	InstrumentName               string  `json:"instrument_name"`
	BaseCurrency                 string  `json:"base_currency"`
	QuoteCurrency                string  `json:"quote_currency"`
	BaseCurrencyDecimals         float64 `json:"base_currency_decimals"`
	QuoteCurrencyDecimals        float64 `json:"quote_currency_decimals"`
	BaseCurrencyDisplayDecimals  float64 `json:"base_currency_display_decimals"`
	QuoteCurrencyDisplayDecimals float64 `json:"quote_currency_display_decimals"`
	Tradable                     bool    `json:"tradable"`
}

// OTCQuoteResponse represents quote to buy or sell with either base currency or quote currency.
type OTCQuoteResponse struct {
	QuoteID           string     `json:"quote_id"`
	QuoteStatus       string     `json:"quote_status"`
	QuoteDirection    string     `json:"quote_direction"`
	BaseCurrency      string     `json:"base_currency"`
	QuoteCurrency     string     `json:"quote_currency"`
	BaseCurrencySize  SafeNumber `json:"base_currency_size"`
	QuoteCurrencySize SafeNumber `json:"quote_currency_size"`
	QuoteBuy          SafeNumber `json:"quote_buy"`
	QuoteBuyQuantity  SafeNumber `json:"quote_buy_quantity"`
	QuoteBuyValue     SafeNumber `json:"quote_buy_value"`
	QuoteSell         SafeNumber `json:"quote_sell"`
	QuoteSellQuantity SafeNumber `json:"quote_sell_quantity"`
	QuoteSellValue    SafeNumber `json:"quote_sell_value"`
	QuoteDuration     int64      `json:"quote_duration"`
	QuoteTime         types.Time `json:"quote_time"`
	QuoteExpiryTime   types.Time `json:"quote_expiry_time"`
}

// AcceptQuoteResponse represents response param for accepting quote.
type AcceptQuoteResponse struct {
	QuoteID           string     `json:"quote_id"`
	QuoteStatus       string     `json:"quote_status"`
	TradeDirection    string     `json:"trade_direction"`
	QuoteDirection    string     `json:"quote_direction"`
	BaseCurrency      string     `json:"base_currency"`
	QuoteCurrency     string     `json:"quote_currency"`
	BaseCurrencySize  SafeNumber `json:"base_currency_size"`
	QuoteCurrencySize SafeNumber `json:"quote_currency_size"`
	QuoteBuy          SafeNumber `json:"quote_buy"`
	QuoteSell         SafeNumber `json:"quote_sell"`
	QuoteDuration     int64      `json:"quote_duration"`
	QuoteTime         types.Time `json:"quote_time"`
	QuoteExpiryTime   types.Time `json:"quote_expiry_time"`
	TradePrice        SafeNumber `json:"trade_price"`
	TradeQuantity     SafeNumber `json:"trade_quantity"`
	TradedValue       SafeNumber `json:"trade_value"`
	TradeTime         types.Time `json:"trade_time"`
}

// QuoteHistoryResponse represents a quote history instance.
type QuoteHistoryResponse struct {
	Count     int64 `json:"count"`
	QuoteList []struct {
		QuoteID           string     `json:"quote_id"`
		QuoteStatus       string     `json:"quote_status"`
		QuoteDirection    string     `json:"quote_direction"`
		BaseCurrency      string     `json:"base_currency"`
		QuoteCurrency     string     `json:"quote_currency"`
		BaseCurrencySize  float64    `json:"base_currency_size"`
		QuoteCurrencySize SafeNumber `json:"quote_currency_size"`
		QuoteBuy          SafeNumber `json:"quote_buy"`
		QuoteSell         SafeNumber `json:"quote_sell"`
		QuoteDuration     int64      `json:"quote_duration"`
		QuoteTime         types.Time `json:"quote_time"`
		QuoteExpiryTime   types.Time `json:"quote_expiry_time"`
		TradeDirection    string     `json:"trade_direction"`
		TradePrice        float64    `json:"trade_price"`
		TradeQuantity     float64    `json:"trade_quantity"`
		TradeValue        float64    `json:"trade_value"`
		TradeTime         types.Time `json:"trade_time"`
	} `json:"quote_list"`
}

// OTCTradeHistoryResponse represents an OTC trade history response.
type OTCTradeHistoryResponse struct {
	Count     int64          `json:"count"`
	TradeList []OTCTradeItem `json:"trade_list"`
}

// OTCOrderResponse represents an OTC order response.
type OTCOrderResponse struct {
	ClientOid       string       `json:"client_oid"`
	OrderID         string       `json:"order_id"`
	Status          string       `json:"status"` // FILLED, REJECTED, UNSETTLED, PENDING
	InstrumentName  string       `json:"instrument_name"`
	Side            string       `json:"side"`
	Price           types.Number `json:"price"`
	Quantity        types.Number `json:"quantity"`
	Value           string       `json:"value"`
	CreateTime      types.Time   `json:"create_time"`
	RejectionReason string       `json:"reject_reason"`
}

// OTCTradeItem represents an OTC trade item detail.
type OTCTradeItem struct {
	QuoteID           string     `json:"quote_id"`
	QuoteStatus       string     `json:"quote_status"`
	QuoteDirection    string     `json:"quote_direction"`
	BaseCurrency      string     `json:"base_currency"`
	QuoteCurrency     string     `json:"quote_currency"`
	BaseCurrencySize  SafeNumber `json:"base_currency_size"`
	QuoteCurrencySize SafeNumber `json:"quote_currency_size"`
	QuoteBuy          string     `json:"quote_buy"`
	QuoteSell         string     `json:"quote_sell"`
	QuoteDuration     int64      `json:"quote_duration"`
	QuoteTime         types.Time `json:"quote_time"`
	QuoteExpiryTime   types.Time `json:"quote_expiry_time"`
	TradeDirection    string     `json:"trade_direction"`
	TradePrice        SafeNumber `json:"trade_price"`
	TradeQuantity     SafeNumber `json:"trade_quantity"`
	TradeValue        SafeNumber `json:"trade_value"`
	TradeTime         types.Time `json:"trade_time"`
}

// SubscriptionPayload represents a subscription payload
type SubscriptionPayload struct {
	ID            int64               `json:"id"`
	Method        string              `json:"method"`
	Params        map[string][]string `json:"params"`
	Nonce         int64               `json:"nonce"`
	Authenticated bool                `json:"-"`
}

// SubscriptionResponse represents a websocket subscription response.
type SubscriptionResponse struct {
	ID     int64     `json:"id"`
	Code   int64     `json:"code,omitempty"`
	Method string    `json:"method"`
	Result *WsResult `json:"result,omitempty"`
}

// SubscriptionInput represents a public/heartbead response
type SubscriptionInput struct {
	ID     int64  `json:"id"`
	Code   int64  `json:"code,omitempty"`
	Method string `json:"method"`
}

// SubscriptionRawData represents a subscription response raw data.
type SubscriptionRawData struct {
	Data          []byte
	Authenticated bool
}

// WsResult represents a subscriptions response result
type WsResult struct {
	Channel        string          `json:"channel,omitempty"`
	Subscription   string          `json:"subscription,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`
	InstrumentName string          `json:"instrument_name,omitempty"`
	Depth          int64           `json:"depth,omitempty"`    // for orderbooks
	Interval       string          `json:"interval,omitempty"` // for candlestick data.
	Timestamp      types.Time      `json:"t"`                  // Timestamp of book publish (milliseconds since the Unix epoch)
}

// UserOrder represents a user orderbook object.
type UserOrder struct {
	Status                     string       `json:"status"`
	Side                       string       `json:"side"`
	Price                      types.Number `json:"price"`
	Quantity                   types.Number `json:"quantity"`
	OrderID                    string       `json:"order_id"`
	ClientOrderID              string       `json:"client_oid"`
	CreateTime                 types.Time   `json:"create_time"`
	UpdateTime                 types.Time   `json:"update_time"`
	Type                       string       `json:"type"`
	InstrumentName             string       `json:"instrument_name"`
	CumulativeExecutedQuantity types.Number `json:"cumulative_quantity"`
	CumulativeExecutedValue    types.Number `json:"cumulative_value"`
	AvgPrice                   types.Number `json:"avg_price"`
	FeeCurrency                string       `json:"fee_currency"`
	TimeInForce                string       `json:"time_in_force"`
	AccountID                  string       `json:"account_id"`
	OrderType                  string       `json:"order_type"`
	ExecInst                   []string     `json:"exec_inst"`
	LimitPrice                 types.Number `json:"limit_price"`
	OrderValue                 types.Number `json:"order_value"`
	MakerFeeRate               types.Number `json:"maker_fee_rate"`
	TakerFeeRate               types.Number `json:"taker_fee_rate"`
	CumulativeFee              types.Number `json:"cumulative_fee"`
	UpdateUserID               string       `json:"update_user_id"`
	OrderDate                  string       `json:"order_date"`
	FeeInstrumentName          string       `json:"fee_instrument_name"`
	CreateTimeNs               types.Time   `json:"create_time_ns"`
}

// UserTrade represents a user trade instance.
type UserTrade struct {
	TradeID           string       `json:"trade_id"`
	OrderID           string       `json:"order_id"`
	Side              string       `json:"side"`
	InstrumentName    string       `json:"instrument_name"`
	Fee               types.Number `json:"fee"`
	CreateTime        types.Time   `json:"create_time"`
	TradedPrice       types.Number `json:"traded_price"`
	TradedQuantity    types.Number `json:"traded_quantity"`
	FeeCurrency       string       `json:"fee_currency"`
	AccountID         string       `json:"account_id"`
	EventDate         string       `json:"event_date"`
	JournalType       string       `json:"journal_type"`
	Fees              string       `json:"fees"`
	TradeMatchID      string       `json:"trade_match_id"`
	ClientOrderID     string       `json:"client_oid"`
	TakerSide         string       `json:"taker_side"`
	FeeInstrumentName string       `json:"fee_instrument_name"`
	CreateTimeNs      types.Time   `json:"create_time_ns"`
}

// WsOrderbook represents an orderbook websocket push data.
type WsOrderbook struct {
	Asks                [][3]types.Number `json:"asks"`
	Bids                [][3]types.Number `json:"bids"`
	PushTime            types.Time        `json:"t"`
	OrderbookUpdateTime types.Time        `json:"tt"`
	UpdateSequence      int64             `json:"u"`
	Cs                  int64             `json:"cs"`
}

// WsRequestPayload represents authentication and request sending payload
type WsRequestPayload struct {
	ID        int64          `json:"id"`
	Method    string         `json:"method"`
	APIKey    string         `json:"api_key,omitempty"`
	Signature string         `json:"sig,omitempty"`
	Nonce     int64          `json:"nonce,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
}

// RespData represents a generalized object structure of responses.
type RespData struct {
	ID            int64       `json:"id"`
	Method        string      `json:"method"`
	Code          int64       `json:"code"`
	Message       string      `json:"message"`
	DetailCode    string      `json:"detail_code"`
	DetailMessage string      `json:"detail_message"`
	Result        interface{} `json:"result"`
}

// WSRespData represents a generalized object structure of websocket responses.
type WSRespData struct {
	ID            int64       `json:"id"`
	Method        string      `json:"method"`
	Code          int64       `json:"code"`
	Message       string      `json:"message"`
	DetailCode    string      `json:"detail_code"`
	DetailMessage string      `json:"detail_message"`
	Result        interface{} `json:"result"`
}

// AllInstruments holds all instruments detail.
type AllInstruments struct {
	Instruments []InstrumentDetail `json:"data"`
}

// InstrumentDetail holds a trading instrument detail
type InstrumentDetail struct {
	Symbol            string       `json:"symbol"`
	InstrumentType    string       `json:"inst_type"`
	DisplayName       string       `json:"display_name"`
	BaseCcy           string       `json:"base_ccy"`
	QuoteCcy          string       `json:"quote_ccy"`
	QuoteDecimals     int          `json:"quote_decimals"`
	QuantityDecimals  int          `json:"quantity_decimals"`
	PriceTickSize     types.Number `json:"price_tick_size"`
	QtyTickSize       types.Number `json:"qty_tick_size"`
	MaxLeverage       types.Number `json:"max_leverage"`
	Tradable          bool         `json:"tradable"`
	ExpiryTimestampMs types.Time   `json:"expiry_timestamp_ms"`
	BetaProduct       bool         `json:"beta_product"`
	UnderlyingSymbol  string       `json:"underlying_symbol,omitempty"`
	ContractSize      types.Number `json:"contract_size,omitempty"`
	MarginBuyEnabled  bool         `json:"margin_buy_enabled"`
	MarginSellEnabled bool         `json:"margin_sell_enabled"`
}

// StakingConversionRate represents staked token and liquid staking token
type StakingConversionRate struct {
	InstrumentName string       `json:"instrument_name"`
	ConversionRate types.Number `json:"conversion_rate"`
}

// StakingResp represents a staking order response
type StakingResp struct {
	StakingID               string       `json:"staking_id"`
	InstrumentName          string       `json:"instrument_name"`
	Status                  string       `json:"status"`
	UnderlyingInstName      string       `json:"underlying_inst_name"`
	PreStakeChargeRateInBps string       `json:"pre_stake_charge_rate_in_bps"`
	Quantity                types.Number `json:"quantity"`
	PreStakeCharge          types.Number `json:"pre_stake_charge"`
	Reason                  string       `json:"reason"`
}

// StakingPosition represents a staking position for a user/token
type StakingPosition struct {
	Data []struct {
		InstrumentName          string       `json:"instrument_name"`
		UnderlyingInstName      string       `json:"underlying_inst_name"`
		StakedQuantity          types.Number `json:"staked_quantity"`
		PendingStakedQuantity   types.Number `json:"pending_staked_quantity"`
		PendingUnstakedQuantity types.Number `json:"pending_unstaked_quantity"`
		RewardEligibleQuantity  types.Number `json:"reward_eligible_quantity"`
	} `json:"data"`
}

// StakingInstrumentsResponse represents a list of staking instruments list
type StakingInstrumentsResponse struct {
	Data []StakingInstrumentDetail `json:"data"`
}

// StakingInstrumentDetail represents a staking instrument
type StakingInstrumentDetail struct {
	InstrumentName          string       `json:"instrument_name"`
	UnderlyingInstName      string       `json:"underlying_inst_name"`
	RewardInstName          string       `json:"reward_inst_name"`
	OutOfStock              bool         `json:"out_of_stock"`
	BlockUnstake            bool         `json:"block_unstake"`
	EstRewards              string       `json:"est_rewards"`
	AprY                    string       `json:"apr_y"`
	MinStakeAmt             types.Number `json:"min_stake_amt"`
	RewardFrequency         string       `json:"reward_frequency"`
	LockUpPeriod            string       `json:"lock_up_period"`
	IsCompoundReward        bool         `json:"is_compound_reward"`
	PreStakeChargeEnable    bool         `json:"pre_stake_charge_enable"`
	PreStakeChargeRateInBps string       `json:"pre_stake_charge_rate_in_bps"`
	IsRestaked              bool         `json:"is_restaked"`
	AdditionalRewards       []any        `json:"additional_rewards"`
}

// StakingRequestsResponse represents a stake/unstake requests list
type StakingRequestsResponse struct {
	Data []StakingDetail `json:"data"`
}

// StakingDetail represents a stake/unstake request detail
type StakingDetail struct {
	InstrumentName     string       `json:"instrument_name"`
	UnderlyingInstName string       `json:"underlying_inst_name"`
	CycleID            string       `json:"cycle_id"`
	StakingID          string       `json:"staking_id"`
	Status             string       `json:"status"`
	Account            string       `json:"account"`
	Quantity           types.Number `json:"quantity"`
	Side               string       `json:"side"`
	CreateTimestampMs  types.Time   `json:"create_timestamp_ms"`
}

// StakingRewardHistory represents a staking reward history
type StakingRewardHistory struct {
	Data []StakingRewardDetail `json:"data"`
}

// StakingRewardDetail represents a staking reward detail information.
type StakingRewardDetail struct {
	StakingInstName    string       `json:"staking_inst_name"`
	UnderlyingInstName string       `json:"underlying_inst_name"`
	RewardInstName     string       `json:"reward_inst_name"`
	RewardQuantity     types.Number `json:"reward_quantity"`
	StakedBalance      types.Number `json:"staked_balance"`
	EventTimestampMs   types.Time   `json:"event_timestamp_ms"`
}

// StakingTokenConversionResponse represents a staking token conversion response detail
type StakingTokenConversionResponse struct {
	FromInstrumentName   string       `json:"from_instrument_name"`
	ToInstrumentName     string       `json:"to_instrument_name"`
	ExpectedRate         types.Number `json:"expected_rate"`
	FromQuantity         types.Number `json:"from_quantity"`
	SlippageToleranceBps types.Number `json:"slippage_tolerance_bps"`
	ConvertID            int64        `json:"convert_id"`
	Reason               string       `json:"reason"`
}

// StakingConvertsHistory represents a list of open staking requests
type StakingConvertsHistory struct {
	Data []StakingConvertRequestDetail `json:"data"`
}

// StakingConvertRequestDetail represents an open staking request detail
type StakingConvertRequestDetail struct {
	FromInstrumentName   string       `json:"from_instrument_name"`
	ToInstrumentName     string       `json:"to_instrument_name"`
	ExpectedRate         types.Number `json:"expected_rate"`
	FromQuantity         types.Number `json:"from_quantity"`
	SlippageToleranceBps string       `json:"slippage_tolerance_bps"`
	ActualRate           types.Number `json:"actual_rate"`
	ToQuantity           types.Number `json:"to_quantity"`
	ConvertID            int64        `json:"convert_id"`
	Status               string       `json:"status"`
	CreateTimestampMs    types.Time   `json:"create_timestamp_ms"`
}

// FeeRate holds fee rate information for spot and derivatives maker and taker rates.
type FeeRate struct {
	SpotTier                   types.Number `json:"spot_tier"`
	DerivTier                  types.Number `json:"deriv_tier"`
	EffectiveSpotMakerRateBps  types.Number `json:"effective_spot_maker_rate_bps"`
	EffectiveSpotTakerRateBps  types.Number `json:"effective_spot_taker_rate_bps"`
	EffectiveDerivMakerRateBps types.Number `json:"effective_deriv_maker_rate_bps"`
	EffectiveDerivTakerRateBps types.Number `json:"effective_deriv_taker_rate_bps"`
}

// SmartCrossMarginRiskParameter holds risk parameter settings for smart cross margin
type SmartCrossMarginRiskParameter struct {
	DefaultMaxProductLeverageForSpot    types.Number `json:"default_max_product_leverage_for_spot"`
	DefaultMaxProductLeverageForPerps   types.Number `json:"default_max_product_leverage_for_perps"`
	DefaultMaxProductLeverageForFutures types.Number `json:"default_max_product_leverage_for_futures"`
	DefaultUnitMarginRate               types.Number `json:"default_unit_margin_rate"`
	DefaultCollateralCap                types.Number `json:"default_collateral_cap"`
	UpdateTimestampMs                   int64        `json:"update_timestamp_ms"`
	BaseCurrencyConfig                  []struct {
		InstrumentName            string       `json:"instrument_name"`
		MinimumHaircut            string       `json:"minimum_haircut,omitempty"`
		UnitMarginRate            types.Number `json:"unit_margin_rate,omitempty"`
		OrderLimit                types.Number `json:"order_limit"`
		CollateralCapNotional     string       `json:"collateral_cap_notional,omitempty"`
		MaxProductLeverageForSpot string       `json:"max_product_leverage_for_spot,omitempty"`
		MaxShortSellLimit         types.Number `json:"max_short_sell_limit,omitempty"`
	} `json:"base_currency_config"`
}

// UserAccountBalanceDetail holds user's account balance detail
type UserAccountBalanceDetail struct {
	Data []UserPositionBalanceDetail `json:"data"`
}

// UserPositionBalanceDetail holds user's position balance valuation detail.
type UserPositionBalanceDetail struct {
	TotalAvailableBalance     types.Number      `json:"total_available_balance"`
	TotalMarginBalance        types.Number      `json:"total_margin_balance"`
	TotalInitialMargin        types.Number      `json:"total_initial_margin"`
	TotalPositionIm           string            `json:"total_position_im"`
	TotalHaircut              string            `json:"total_haircut"`
	TotalMaintenanceMargin    types.Number      `json:"total_maintenance_margin"`
	TotalPositionCost         types.Number      `json:"total_position_cost"`
	TotalCashBalance          types.Number      `json:"total_cash_balance"`
	TotalCollateralValue      types.Number      `json:"total_collateral_value"`
	TotalSessionUnrealizedPnl types.Number      `json:"total_session_unrealized_pnl"`
	InstrumentName            string            `json:"instrument_name"`
	TotalSessionRealizedPnl   string            `json:"total_session_realized_pnl"`
	IsLiquidating             bool              `json:"is_liquidating"`
	TotalEffectiveLeverage    string            `json:"total_effective_leverage"`
	PositionLimit             types.Number      `json:"position_limit"`
	UsedPositionLimit         types.Number      `json:"used_position_limit"`
	PositionBalances          []PositionBalance `json:"position_balances"`
}

// PositionBalance holds user's position balance detail.
type PositionBalance struct {
	InstrumentName       string       `json:"instrument_name"`
	Quantity             types.Number `json:"quantity"`
	MarketValue          types.Number `json:"market_value"`
	CollateralEligible   string       `json:"collateral_eligible"`
	Haircut              string       `json:"haircut"`
	CollateralAmount     types.Number `json:"collateral_amount"`
	MaxWithdrawalBalance types.Number `json:"max_withdrawal_balance"`
	ReservedQty          types.Number `json:"reserved_qty"`
}

// UserBalanceHistory holds a brief information of user balance history
type UserBalanceHistory struct {
	InstrumentName string `json:"instrument_name"`
	Data           []struct {
		Time             types.Time   `json:"t"`
		TotalCashBalance types.Number `json:"c"`
	} `json:"data"`
}

// SubAccountBalances holds list of sub-account balance
type SubAccountBalances struct {
	Data []SubAccountBalance `json:"data"`
}

// SubAccountBalance represents a sub-account balance detail
type SubAccountBalance struct {
	Account                   string                      `json:"account"`
	InstrumentName            string                      `json:"instrument_name"`
	TotalAvailableBalance     types.Number                `json:"total_available_balance"`
	TotalMarginBalance        types.Number                `json:"total_margin_balance"`
	TotalInitialMargin        types.Number                `json:"total_initial_margin"`
	TotalMaintenanceMargin    types.Number                `json:"total_maintenance_margin"`
	TotalPositionCost         types.Number                `json:"total_position_cost"`
	TotalCashBalance          types.Number                `json:"total_cash_balance"`
	TotalCollateralValue      types.Number                `json:"total_collateral_value"`
	TotalSessionUnrealizedPnl types.Number                `json:"total_session_unrealized_pnl"`
	TotalSessionRealizedPnl   types.Number                `json:"total_session_realized_pnl"`
	TotalEffectiveLeverage    types.Number                `json:"total_effective_leverage"`
	PositionLimit             types.Number                `json:"position_limit"`
	UsedPositionLimit         types.Number                `json:"used_position_limit"`
	IsLiquidating             bool                        `json:"is_liquidating"`
	PositionBalances          []SubAccountPositionBalance `json:"position_balances"`
}

// SubAccountPositionBalance holds sub-account's position balance detail
type SubAccountPositionBalance struct {
	InstrumentName       string       `json:"instrument_name"`
	Quantity             types.Number `json:"quantity"`
	MarketValue          string       `json:"market_value"`
	CollateralEligible   string       `json:"collateral_eligible"`
	Haircut              string       `json:"haircut"`
	CollateralAmount     types.Number `json:"collateral_amount"`
	MaxWithdrawalBalance types.Number `json:"max_withdrawal_balance"`
}

// UsersPositions holds user's positions
type UsersPositions struct {
	Data []UserPosition `json:"data"`
}

// UserPosition holds a brief of user's position information
type UserPosition struct {
	InstrumentType       string       `json:"type"`
	AccountID            string       `json:"account_id"`
	Quantity             types.Number `json:"quantity"`
	Cost                 types.Number `json:"cost"`
	OpenPositionPnl      types.Number `json:"open_position_pnl"`
	OpenPosCost          types.Number `json:"open_pos_cost"`
	SessionPnl           types.Number `json:"session_pnl"`
	UpdateTimestampMs    types.Time   `json:"update_timestamp_ms"`
	InstrumentName       string       `json:"instrument_name"`
	LiquidationPrice     types.Number `json:"liquidation_price"`
	SessionUnrealizedPnl types.Number `json:"session_unrealized_pnl"`
	PosInitialMargin     types.Number `json:"pos_initial_margin"`
	PosMaintenanceMargin types.Number `json:"pos_maintenance_margin"`
	MarketValue          types.Number `json:"market_value"`
	MarkPrice            types.Number `json:"mark_price"`
	TargetLeverage       types.Number `json:"target_leverage"`
}

// WsUserPositionBalance holds user's asset and position balance details.
type WsUserPositionBalance struct {
	Balances  []UserAssetBalance `json:"balances"`
	Positions []UserPosition     `json:"positions"`
}

// UserAssetBalance holds user asset balance detail.
type UserAssetBalance struct {
	CurrencyName string       `json:"instrument_name"`
	Quantity     types.Number `json:"quantity"`
}

// InstrumentTrades holds list of executed trades of an instrument
type InstrumentTrades struct {
	Data []TradeDetail `json:"data"`
}

// AccountSetting holds a self-trade-prevention account setting details
type AccountSetting struct {
	Leverage int64  `json:"leverage"`
	STPID    int64  `json:"stp_id"`
	STPScope string `json:"stp_scope"`
	STPInst  string `json:"stp_inst"`
}

// TradeDetail holds instrument's executed trade detail
type TradeDetail struct {
	AccountID         string       `json:"account_id"`
	EventDate         string       `json:"event_date"`
	JournalType       string       `json:"journal_type"`
	TradedQuantity    types.Number `json:"traded_quantity"`
	TradedPrice       types.Number `json:"traded_price"`
	Fees              types.Number `json:"fees"`
	OrderID           string       `json:"order_id"`
	TradeID           string       `json:"trade_id"`
	TradeMatchID      string       `json:"trade_match_id"`
	ClientOrderID     string       `json:"client_oid"`
	TakerSide         string       `json:"taker_side"`
	Side              string       `json:"side"`
	InstrumentName    string       `json:"instrument_name"`
	FeeInstrumentName string       `json:"fee_instrument_name"`
	CreateTime        types.Time   `json:"create_time"`
	CreateTimeNs      types.Time   `json:"create_time_ns"`
}

// OrderIDsDetail holds order id and client supplied order id
type OrderIDsDetail struct {
	ClientOid string `json:"client_oid"`
	OrderID   string `json:"order_id"`
}

// OrdersDetail holds list of detailed order
type OrdersDetail struct {
	Data []OrderDetail `json:"data"`
}

// OrderDetail holds order detail
type OrderDetail struct {
	AccountID          string            `json:"account_id"`
	OrderID            string            `json:"order_id"`
	ClientOrderID      string            `json:"client_oid"`
	OrderType          string            `json:"type"`
	TimeInForce        order.TimeInForce `json:"time_in_force"`
	Side               string            `json:"side"`
	ExecInst           []any             `json:"exec_inst"`
	Quantity           types.Number      `json:"quantity"`
	Price              types.Number      `json:"price,omitempty"`
	OrderValue         types.Number      `json:"order_value"`
	AvgPrice           types.Number      `json:"avg_price"`
	TriggerPrice       types.Number      `json:"trigger_price"`
	CumulativeQuantity types.Number      `json:"cumulative_quantity"`
	CumulativeValue    types.Number      `json:"cumulative_value"`
	CumulativeFee      types.Number      `json:"cumulative_fee"`
	Status             string            `json:"status"`
	UpdateUserID       string            `json:"update_user_id"`
	OrderDate          string            `json:"order_date"`
	InstrumentName     string            `json:"instrument_name"`
	FeeInstrumentName  string            `json:"fee_instrument_name"`
	ListID             string            `json:"list_id"`
	ContingencyType    string            `json:"contingency_type"`
	TriggerPriceType   string            `json:"trigger_price_type"`
	CreateTime         types.Time        `json:"create_time"`
	CreateTimeNs       types.Time        `json:"create_time_ns"`
	UpdateTime         types.Time        `json:"update_time"`
}

// ValueAndTimestamp holds value, and timestamp information
type ValueAndTimestamp struct {
	Value     types.Number `json:"v"`
	Timestamp types.Time   `json:"t"`
}

// UserBalanceDetail holds user balance detail specific to an instrument/currency
type UserBalanceDetail struct {
	TotalAvailableBalance     types.Number `json:"total_available_balance"`
	TotalMarginBalance        types.Number `json:"total_margin_balance"`
	TotalInitialMargin        types.Number `json:"total_initial_margin"`
	TotalMaintenanceMargin    types.Number `json:"total_maintenance_margin"`
	TotalPositionCost         types.Number `json:"total_position_cost"`
	TotalCashBalance          types.Number `json:"total_cash_balance"`
	TotalCollateralValue      types.Number `json:"total_collateral_value"`
	TotalSessionUnrealizedPnl types.Number `json:"total_session_unrealized_pnl"`
	InstrumentName            string       `json:"instrument_name"`
	TotalSessionRealizedPnl   types.Number `json:"total_session_realized_pnl"`
	IsLiquidating             bool         `json:"is_liquidating"`
	TotalEffectiveLeverage    types.Number `json:"total_effective_leverage"`
	PositionLimit             types.Number `json:"position_limit"`
	UsedPositionLimit         types.Number `json:"used_position_limit"`
	PositionBalances          []struct {
		InstrumentName       string       `json:"instrument_name"`
		Quantity             types.Number `json:"quantity"`
		MarketValue          types.Number `json:"market_value"`
		CollateralAmount     types.Number `json:"collateral_amount"`
		CollateralWeight     types.Number `json:"collateral_weight"`
		MaxWithdrawalBalance types.Number `json:"max_withdrawal_balance"`
	} `json:"position_balances"`
}

// UserAccountRisk holds position and balance snapshot for the user on a regular basis
type UserAccountRisk struct {
	Currency                  string               `json:"instrument_name"`
	TotalAvailableBalance     types.Number         `json:"total_available_balance"`
	TotalCashBalance          types.Number         `json:"total_cash_balance"`
	TotalInitialMargin        types.Number         `json:"total_initial_margin"`
	TotalMaintenanceMargin    types.Number         `json:"total_maintenance_margin"`
	TotalPositionCost         types.Number         `json:"total_position_cost"`
	TotalSessionUnrealizedPnl types.Number         `json:"total_session_unrealized_pnl"`
	TotalMarginBalance        types.Number         `json:"total_margin_balance"`
	TotalSessionRealizedPnl   types.Number         `json:"total_session_realized_pnl"`
	TotalEffectiveLeverage    types.Number         `json:"total_effective_leverage"`
	PositionLimit             types.Number         `json:"position_limit"`
	UsedPositionLimit         types.Number         `json:"used_position_limit"`
	IsLiquidating             bool                 `json:"is_liquidating"`
	TotalBorrow               string               `json:"total_borrow"`
	MarginScore               types.Number         `json:"margin_score"`
	Balances                  []AccountRiskBalance `json:"balances"`
	Positions                 []UserPosition       `json:"positions"`
	TotalCollateralValue      types.Number         `json:"total_collateral_value"`
}

// AccountRiskBalance holds user's account risk balance detail.
type AccountRiskBalance struct {
	Currency             string       `json:"instrument_name"`
	Quantity             types.Number `json:"quantity"`
	ReservedQty          types.Number `json:"reserved_qty"`
	MarketValue          types.Number `json:"market_value"`
	CollateralAmount     types.Number `json:"collateral_amount"`
	CollateralWeight     types.Number `json:"collateral_weight"`
	MaxWithdrawalBalance types.Number `json:"max_withdrawal_balance"`
	HourlyInterestRate   types.Number `json:"hourly_interest_rate"`
}

// ExpiredSettlementPrice holds expired settlement price detail of instruments
type ExpiredSettlementPrice struct {
	Data []struct {
		Symbol     string       `json:"i"`
		ExpiryTime types.Time   `json:"x"`
		Value      types.Number `json:"v"`
		Timestamp  types.Time   `json:"t"`
	} `json:"data"`
}

// Announcements holds exchange announcements
type Announcements struct {
	Data []struct {
		ID             string     `json:"id"`
		Category       string     `json:"category"`
		ProductType    string     `json:"product_type"`
		AnnouncedAt    types.Time `json:"announced_at"`
		Title          string     `json:"title"`
		Content        string     `json:"content"`
		InstrumentName string     `json:"instrument_name"`
		ImpactedParams struct {
			SpotTradingImpacted       string `json:"spot_trading_impacted"`
			DerivativeTradingImpacted string `json:"derivative_trading_impacted"`
			MarginTradingImpacted     string `json:"margin_trading_impacted"`
			OtcTradingImpacted        string `json:"otc_trading_impacted"`
			ConvertImpacted           string `json:"convert_impacted"`
			StakingImpacted           string `json:"staking_impacted"`
			TradingBotImpacted        string `json:"trading_bot_impacted"`
			CryptoWalletImpacted      string `json:"crypto_wallet_impacted"`
			FiatWalletImpacted        string `json:"fiat_wallet_impacted"`
			LoginImpacted             string `json:"login_impacted"`
		} `json:"impacted_params"`
		StartTime types.Time `json:"start_time"`
		EndTime   types.Time `json:"end_time"`
	} `json:"data"`
}

// FiatDepositInfoDetail holds fiat deposit info
type FiatDepositInfoDetail struct {
	DepositInfoList []DepositInfoItem `json:"deposit_info_list"`
}

// DepositInfoItem holds fiat deposit info
type DepositInfoItem struct {
	PaymentNetwork string `json:"payment_network"`
	Currency       string `json:"currency"`
	BankDetails    struct {
		AccountHolderName  string `json:"account_holder_name"`
		BankName           string `json:"bank_name"`
		BankAddress        string `json:"bank_address"`
		BankCountry        string `json:"bank_country"`
		RoutingNumber      string `json:"routing_number"`
		AccountNumber      string `json:"account_number"`
		RecipientName      string `json:"recipient_name"`
		RecipientAddress   string `json:"recipient_address"`
		BicCode            string `json:"bic_code"`
		IbanCode           string `json:"iban_code"`
		ReferenceCode      string `json:"reference_code"`
		SortCode           string `json:"sort_code"`
		CubixPartnerName   string `json:"cubix_partner_name"`
		CubixAccountName   string `json:"cubix_account_name"`
		CubixAccountID     string `json:"cubix_account_id"`
		CubixAccountNumber string `json:"cubix_account_number"`
		Meta               string `json:"meta"`
	} `json:"bank_details"`
}

// FiatDepositHistoryParam holds a fiat deposit history request parameter
type FiatDepositHistoryParam struct {
	Page            int    `json:"page"`
	PageSize        int    `json:"page_size"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	PaymentNetworks string `json:"payment_networks"`
}

// FiatDepositHistory holds a fiat deposit history detail
type FiatDepositHistory struct {
	TransactionHistoryList []FiatTransactionItem `json:"transaction_history_list"`
	Page                   int64                 `json:"page"`
	PageSize               int64                 `json:"page_size"`
}

// FiatTransactionItem hods a fiat transaction item
type FiatTransactionItem struct {
	ID             string       `json:"id"`
	AccountID      string       `json:"account_id"`
	Currency       string       `json:"currency"`
	Amount         types.Number `json:"amount"`
	AmountInUsd    types.Number `json:"amount_in_usd"`
	FeeCurrency    string       `json:"fee_currency"`
	FeeAmount      types.Number `json:"fee_amount"`
	FeeAmountInUSD types.Number `json:"fee_amount_in_usd"`
	PaymentNetwork string       `json:"payment_network"`
	Status         string       `json:"status"`
	CreatedAt      types.Time   `json:"created_at"`
	UpdatedAt      types.Time   `json:"updated_at"`
	CompletedAt    types.Time   `json:"completed_at"`
	Sender         struct {
		AccountIdentifierValue string `json:"account_identifier_value"`
	} `json:"sender"`
	Beneficiary string `json:"beneficiary"`
}

// FiatWithdrawalHistory hods fiat withdrawal transaction history
type FiatWithdrawalHistory struct {
	Page                   int `json:"page"`
	PageSize               int `json:"page_size"`
	TransactionHistoryList []struct {
		ID             string       `json:"id"`
		AccountID      string       `json:"account_id"`
		Currency       string       `json:"currency"`
		Amount         types.Number `json:"amount"`
		AmountInUSD    types.Number `json:"amount_in_usd"`
		FeeCurrency    string       `json:"fee_currency"`
		FeeAmount      string       `json:"fee_amount"`
		FeeAmountInUSD types.Number `json:"fee_amount_in_usd"`
		PaymentNetwork string       `json:"payment_network"`
		Status         string       `json:"status"`
		CreatedAt      types.Time   `json:"created_at"`
		UpdatedAt      types.Time   `json:"updated_at"`
		CompletedAt    types.Time   `json:"completed_at"`
		Sender         string       `json:"sender"`
		Beneficiary    struct {
			AccountIdentifierValue string `json:"account_identifier_value"`
		} `json:"beneficiary"`
	} `json:"transaction_history_list"`
}

// FiatCreateWithdrawl hods fiat withdrawal params
type FiatCreateWithdrawl struct {
	AccountID              string  `json:"account_id"`
	Amount                 float64 `json:"amount,omitempty,string"`
	Currency               string  `json:"currency"`
	PaymentNetwork         string  `json:"payment_network"`
	AuthorizationID        string  `json:"authorization_id,omitempty"`
	BeneficiaryID          string  `json:"beneficiary_id,omitempty"`
	AccountIdentifierValue string  `json:"account_identifier_value,omitempty"`
	BankIdentifierValue    string  `json:"bank_identifier_value,omitempty"`
	IntermediateBank       struct {
		BankIdentifierType  string `json:"bank_identifier_type,omitempty"`
		BankIdentifierValue string `json:"bank_identifier_value,omitempty"`
		BankName            string `json:"bank_name,omitempty"`
		Address1            string `json:"address_1,omitempty"`
		Address2            string `json:"address_2,omitempty"`
	} `json:"intermediate_bank,omitempty"`
}

// FiatWithdrawalResponse holds a fiat currency withdrawal response
type FiatWithdrawalResponse struct {
	ID                     string  `json:"id"`
	AccountID              string  `json:"account_id"`
	Status                 string  `json:"status"`
	PaymentNetwork         string  `json:"payment_network"`
	Currency               string  `json:"currency"`
	Amount                 float64 `json:"amount"`
	AmountInUsd            float64 `json:"amount_in_usd"`
	FeeCurrency            string  `json:"fee_currency"`
	FeeAmount              float64 `json:"fee_amount"`
	BeneficiaryID          string  `json:"beneficiary_id"`
	AuthorizationID        string  `json:"authorization_id"`
	BankIdentifierValue    string  `json:"bank_identifier_value,omitempty"`
	AccountIdentifierValue string  `json:"account_identifier_value,omitempty"`
}

// FiatWithdrawalQuota holds fiat transaction quota information
type FiatWithdrawalQuota struct {
	Details FiatWithdrawalQuotaDetail `json:"details"`
}

// FiatWithdrawalQuotaDetail holds a fiat withdrawal quota detail
type FiatWithdrawalQuotaDetail struct {
	Currency                                  string            `json:"currency"`
	PaymentNetwork                            string            `json:"payment_network"`
	NetworkName                               string            `json:"network_name"`
	MonthlyQuotaInUSD                         CurrencyAndAmount `json:"monthly_quota_in_usd"`
	DailyQuotaInUSD                           CurrencyAndAmount `json:"daily_quota_in_usd"`
	UsedMonthlyQuotaInUSD                     CurrencyAndAmount `json:"used_monthly_quota_in_usd"`
	UsedDailyQuotaInUSD                       CurrencyAndAmount `json:"used_daily_quota_in_usd"`
	RemainingMonthlyQuotaInUSD                CurrencyAndAmount `json:"remaining_monthly_quota_in_usd"`
	RemainingDailyQuotaInUSD                  CurrencyAndAmount `json:"remaining_daily_quota_in_usd"`
	MinimumWithdrawalAmountInUSD              CurrencyAndAmount `json:"minimum_withdrawal_amount_in_usd"`
	MonthlyQuota                              CurrencyAndAmount `json:"monthly_quota"`
	DailyQuota                                CurrencyAndAmount `json:"daily_quota"`
	UsedMonthlyQuota                          CurrencyAndAmount `json:"used_monthly_quota"`
	UsedDailyQuota                            CurrencyAndAmount `json:"used_daily_quota"`
	RemainingMonthlyQuota                     CurrencyAndAmount `json:"remaining_monthly_quota"`
	RemainingDailyQuota                       CurrencyAndAmount `json:"remaining_daily_quota"`
	MinimumWithdrawalAmount                   CurrencyAndAmount `json:"minimum_withdrawal_amount"`
	CurrencyDailyQuota                        CurrencyAndAmount `json:"currency_daily_quota"`
	CurrencyMonthlyQuota                      CurrencyAndAmount `json:"currency_monthly_quota"`
	CurrencyUsedDailyQuota                    CurrencyAndAmount `json:"currency_used_daily_quota"`
	CurrencyUsedMonthlyQuota                  CurrencyAndAmount `json:"currency_used_monthly_quota"`
	CurrencyRemainingDailyQuota               CurrencyAndAmount `json:"currency_remaining_daily_quota"`
	CurrencyRemainingMonthlyQuota             CurrencyAndAmount `json:"currency_remaining_monthly_quota"`
	TransactionsPerDay                        int64             `json:"transactions_per_day"`
	TransactionsPerMonth                      int64             `json:"transactions_per_month"`
	TransactionsDailyCount                    int64             `json:"transactions_daily_count"`
	TransactionsMonthlyCount                  int64             `json:"transactions_monthly_count"`
	RemainingTransactionsDailyCount           int64             `json:"remaining_transactions_daily_count"`
	RemainingTransactionsMonthlyCount         int64             `json:"remaining_transactions_monthly_count"`
	CurrencyTransactionsPerDay                int64             `json:"currency_transactions_per_day"`
	CurrencyTransactionsPerMonth              int64             `json:"currency_transactions_per_month"`
	CurrencyTransactionsDailyCount            int64             `json:"currency_transactions_daily_count"`
	CurrencyTransactionsMonthlyCount          int64             `json:"currency_transactions_monthly_count"`
	CurrencyRemainingTransactionsDailyCount   int64             `json:"currency_remaining_transactions_daily_count"`
	CurrencyRemainingTransactionsMonthlyCount int64             `json:"currency_remaining_transactions_monthly_count"`
}

// CurrencyAndAmount holds a currency and amount detail
type CurrencyAndAmount struct {
	Currency string       `json:"currency"`
	Amount   types.Number `json:"amount"`
}

// FiatTransactionLimit holds transaction limits for a specific payment network
type FiatTransactionLimit struct {
	Deposit FiatDepositLimit `json:"deposit"`
	Payment FiatPaymentLimit `json:"payment"`
}

// FiatDepositLimit holds a fiat deposit transaction limit detail
type FiatDepositLimit struct {
	MinDepositAmount                   CurrencyAndAmount `json:"min_deposit_amount"`
	DailyMaxDepositAmount              CurrencyAndAmount `json:"daily_max_deposit_amount"`
	MonthlyMaxDepositAmount            CurrencyAndAmount `json:"monthly_max_deposit_amount"`
	CurrencyDailyMaxDepositAmount      CurrencyAndAmount `json:"currency_daily_max_deposit_amount"`
	CurrencyMonthlyMaxDepositAmount    CurrencyAndAmount `json:"currency_monthly_max_deposit_amount"`
	DailyQuota                         int64             `json:"daily_quota"`
	MonthlyQuota                       int64             `json:"monthly_quota"`
	DailyTransactionCount              int64             `json:"daily_transaction_count"`
	MonthlyTransactionCount            int64             `json:"monthly_transaction_count"`
	CurrencyDailyTransactionCount      int64             `json:"currency_daily_transaction_count"`
	CurrencyMonthlyTransactionCount    int64             `json:"currency_monthly_transaction_count"`
	FeeAmount                          CurrencyAndAmount `json:"fee_amount"`
	DailyMaxTransactionCount           int64             `json:"daily_max_transaction_count"`
	MonthlyMaxTransactionCount         int64             `json:"monthly_max_transaction_count"`
	CurrencyDailyMaxTransactionCount   int64             `json:"currency_daily_max_transaction_count"`
	CurrencyMonthlyMaxTransactionCount int64             `json:"currency_monthly_max_transaction_count"`
}

// FiatPaymentLimit holds a fiat payment transaction detail
type FiatPaymentLimit struct {
	Name                  string `json:"name"`
	FullName              string `json:"full_name"`
	ReviewTimeDescription string `json:"review_time_description"`
	ReviewTime            struct {
		Min  types.Number `json:"min"`
		Max  types.Number `json:"max"`
		Unit string       `json:"unit"`
	} `json:"review_time"`
	BankTransferTimeDescription string `json:"bank_transfer_time_description"`
	BankTransferTime            struct {
		Min  types.Number `json:"min"`
		Max  types.Number `json:"max"`
		Unit string       `json:"unit"`
	} `json:"bank_transfer_time"`
	MinPaymentAmount                   CurrencyAndAmount `json:"min_payment_amount"`
	DailyMaxPaymentAmount              CurrencyAndAmount `json:"daily_max_payment_amount"`
	MonthlyMaxPaymentAmount            CurrencyAndAmount `json:"monthly_max_payment_amount"`
	AutoApproveMaxPaymentAmount        CurrencyAndAmount `json:"auto_approve_max_payment_amount"`
	CurrencyDailyMaxPaymentAmount      CurrencyAndAmount `json:"currency_daily_max_payment_amount"`
	CurrencyMonthlyMaxPaymentAmount    CurrencyAndAmount `json:"currency_monthly_max_payment_amount"`
	FeeAmount                          CurrencyAndAmount `json:"fee_amount"`
	RefundFeeAmount                    CurrencyAndAmount `json:"refund_fee_amount"`
	DailyTransactionCount              int64             `json:"daily_transaction_count"`
	MonthlyTransactionCount            int64             `json:"monthly_transaction_count"`
	CurrencyDailyTransactionCount      int64             `json:"currency_daily_transaction_count"`
	CurrencyMonthlyTransactionCount    int64             `json:"currency_monthly_transaction_count"`
	DailyMaxTransactionCount           int64             `json:"daily_max_transaction_count"`
	MonthlyMaxTransactionCount         int64             `json:"monthly_max_transaction_count"`
	CurrencyDailyMaxTransactionCount   int64             `json:"currency_daily_max_transaction_count"`
	CurrencyMonthlyMaxTransactionCount int64             `json:"currency_monthly_max_transaction_count"`
}

// FiatBankAccounts holds a fiat bank accounts list
type FiatBankAccounts struct {
	BankAccountsList []FiatBankAccountDetail `json:"bank_accounts_list"`
}

// FiatBankAccountDetail holds a fiat bank account detail
type FiatBankAccountDetail struct {
	ID                            string   `json:"id"`
	Ok                            string   `json:"ok"`
	Code                          string   `json:"code"`
	Message                       string   `json:"message"`
	AccountID                     string   `json:"account_id"`
	UserUUID                      string   `json:"user_uuid"`
	Status                        string   `json:"status"`
	BankName                      string   `json:"bank_name"`
	BankCity                      string   `json:"bank_city"`
	BankCountry                   string   `json:"bank_country"`
	BankIdentifierType            string   `json:"bank_identifier_type"`
	BankIdentifierValue           string   `json:"bank_identifier_value"`
	BankAccountHolderName         string   `json:"bank_account_holder_name"`
	AccountIdentifierType         string   `json:"account_identifier_type"`
	AccountIdentifierValue        string   `json:"account_identifier_value"`
	AccountHolderName             string   `json:"account_holder_name"`
	AccountType                   string   `json:"account_type"`
	Currency                      string   `json:"currency"`
	VerifiedBy                    string   `json:"verified_by"`
	Reason                        string   `json:"reason"`
	SupportedPaymentNetworks      []string `json:"supported_payment_networks"`
	WithdrawalPaymentNetworks     string   `json:"withdrawal_payment_networks"`
	PaymentNetworkIdentifierValue string   `json:"payment_network_identifier_value"`
	BankDetails                   struct {
		InstitutionalStreetAddress string `json:"institutional_street_address"`
		BankStreetAddress          string `json:"bank_street_address"`
		AccountHolderAddress       string `json:"account_holder_address"`
		BankAccountAddress         string `json:"bank_account_address"`
		IntermediateBank           string `json:"intermediate_bank"`
		IntermediateBankOptions    string `json:"intermediate_bank_options"`
	} `json:"bank_details"`
	CreatedAt types.Time `json:"created_at"`
	UpdatedAt types.Time `json:"updated_at"`
}
