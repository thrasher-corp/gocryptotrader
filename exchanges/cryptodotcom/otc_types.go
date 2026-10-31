package cryptodotcom

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// OTC settlement arrangements: IMMEDIATE for an account that cannot settle later, T1 for one that can
const (
	OTCSettlementArrangementImmediate = "IMMEDIATE"
	OTCSettlementArrangementT1        = "T1"
)

// OTCDealTypeQuoteRequest is the type of a deal on a quote, the only deal type, which the deal methods send
const OTCDealTypeQuoteRequest = "QUOTE_REQUEST"

// OTC quote request statuses: a quote request is ACTIVE while it can be quoted and dealt on, until its duration passes
// and it is COMPLETED, or it is REJECTED
const (
	OTCQuoteRequestStatusNew       = "NEW"
	OTCQuoteRequestStatusActive    = "ACTIVE"
	OTCQuoteRequestStatusCompleted = "COMPLETED"
	OTCQuoteRequestStatusRejected  = "REJECTED"
)

// OTC quote statuses: a quote is ACTIVE until its time to live passes and it is EXPIRED, unless its liquidity provider
// cancels it first
const (
	OTCQuoteStatusActive    = "ACTIVE"
	OTCQuoteStatusExpired   = "EXPIRED"
	OTCQuoteStatusCancelled = "CANCELLED"
)

// OTC deal statuses: Crypto.com accepts a deal, its liquidity provider confirms it by executing it, and it is then
// settled, unless it is rejected, such as for an expired quote or an insufficient balance
const (
	OTCDealStatusAccepted  = "ACCEPTED"
	OTCDealStatusConfirmed = "CONFIRMED"
	OTCDealStatusSettled   = "SETTLED"
	OTCDealStatusRejected  = "REJECTED"
)

// OTCLegTypePrice is the type Crypto.com reports for a priced leg of a quote, quote request or deal
const OTCLegTypePrice = "PRICE"

// OTCQuoteRequest requests quotes on one or more legs. ClientQuoteRequestID must be unique among the active quote
// requests, of letters, digits, underscores and hyphens. FirmQuote asks for quotes whose prices are guaranteed.
// Duration is how long the quote request stays active, 5, 10 or 30 seconds, or 1, 5 or 10 minutes, and 10 seconds when
// unset; QuoteTimeToLive is the least time each quote stays valid, 4 or 5 seconds, and 4 when unset
type OTCQuoteRequest struct {
	ClientQuoteRequestID  string
	FirmQuote             bool
	SettlementArrangement string
	Duration              time.Duration
	QuoteTimeToLive       time.Duration
	Legs                  []OTCQuoteRequestLeg
}

// otcQuoteRequestParams is the params of a request-quote request, its durations in milliseconds
type otcQuoteRequestParams struct {
	ClientQuoteRequestID  string               `json:"cl_quote_req_id"`
	FirmQuote             bool                 `json:"firm_quote,omitempty"`
	SettlementArrangement string               `json:"settlement_arrangement,omitempty"`
	Duration              uint64               `json:"duration,omitempty,string"`
	QuoteTimeToLive       uint64               `json:"quote_ttl,omitempty,string"`
	Legs                  []OTCQuoteRequestLeg `json:"leg_list"`
}

// OTCQuoteRequestLeg is a leg of a quote request: an instrument to buy or sell, by quantity or by notional but not both.
// Type is only reported
type OTCQuoteRequestLeg struct {
	InstrumentName string       `json:"instrument_name"`
	Side           string       `json:"side"`
	Quantity       types.Number `json:"quantity,omitempty"`
	Notional       types.Number `json:"notional,omitempty"`
	Type           string       `json:"type,omitempty"`
}

// OTCQuoteRequestResponse is a quote request; request-quote reports only its IDs, and the user.otc_qr.requests channel
// only its IDs, status and reason
type OTCQuoteRequestResponse struct {
	QuoteRequestID        string               `json:"quote_req_id"`
	ClientQuoteRequestID  string               `json:"cl_quote_req_id"`
	FirmQuote             bool                 `json:"firm_quote"`
	SettlementArrangement string               `json:"settlement_arrangement"`
	Status                string               `json:"status"`
	Reason                string               `json:"reason"`
	RequestTime           types.Time           `json:"request_time_ns"`
	Legs                  []OTCQuoteRequestLeg `json:"leg_list"`
}

// OTCInstrumentResponse is an instrument quotes can be requested on, with its price and quantity precision
type OTCInstrumentResponse struct {
	InstrumentName   string        `json:"instrument_name"`
	BaseCurrency     currency.Code `json:"base_currency"`
	QuoteCurrency    currency.Code `json:"quote_currency"`
	Type             string        `json:"type"`
	PriceTickSize    types.Number  `json:"price_tick_size"`
	QuoteDecimals    uint64        `json:"quote_decimals"`
	QuantityTickSize types.Number  `json:"qty_tick_size"`
	QuantityDecimals uint64        `json:"qty_decimals"`
}

// otcInstrumentListResponse is the result of get-otc-instruments
type otcInstrumentListResponse struct {
	InstrumentList []OTCInstrumentResponse `json:"instrument_list"`
}

// OTCQuoteRequestHistoryRequest filters the open quote requests and the quote request history: by quote request ID or
// client quote request ID, and by a time window, the start inclusive and the end exclusive, that defaults to the day
// before the end time, which defaults to now. Limit is the page size, 20 by default and at most 200
type OTCQuoteRequestHistoryRequest struct {
	ClientQuoteRequestID string
	QuoteRequestID       string
	StartTime            time.Time
	EndTime              time.Time
	Limit                uint64
}

// otcQuoteRequestHistoryParams is the params of a quote request query, its times in Unix nanoseconds
type otcQuoteRequestHistoryParams struct {
	ClientQuoteRequestID string `json:"cl_quote_req_id,omitempty"`
	QuoteRequestID       string `json:"quote_req_id,omitempty"`
	StartTime            uint64 `json:"start_time,omitempty,string"`
	EndTime              uint64 `json:"end_time,omitempty,string"`
	Limit                uint64 `json:"limit,omitempty"`
}

// otcQuoteRequestListResponse is the result of a quote request query; its count is the list's length, so it is left out
type otcQuoteRequestListResponse struct {
	QuoteRequestList []OTCQuoteRequestResponse `json:"quote_request_list"`
}

// OTCQuoteHistoryRequest filters the open quotes and the quote history: by quote ID or liquidity provider quote ID,
// either of which overrides the time window, or by a time window as OTCQuoteRequestHistoryRequest's
type OTCQuoteHistoryRequest struct {
	QuoteID                  string
	LiquidityProviderQuoteID string
	StartTime                time.Time
	EndTime                  time.Time
	Limit                    uint64
}

// otcQuoteHistoryParams is the params of a quote query, its times in Unix nanoseconds
type otcQuoteHistoryParams struct {
	QuoteID                  string `json:"quote_id,omitempty"`
	LiquidityProviderQuoteID string `json:"lp_quote_id,omitempty"`
	StartTime                uint64 `json:"start_time,omitempty,string"`
	EndTime                  uint64 `json:"end_time,omitempty,string"`
	Limit                    uint64 `json:"limit,omitempty"`
}

// OTCQuoteResponse is a liquidity provider's quote for a quote request, as the quote queries and the user.otc_qr.quotes
// channel report it; the queries leave out the client quote request ID, and the channel the liquidity provider's quote
// ID
type OTCQuoteResponse struct {
	QuoteID                  string        `json:"quote_id"`
	LiquidityProviderQuoteID string        `json:"lp_quote_id"`
	QuoteRequestID           string        `json:"quote_req_id"`
	ClientQuoteRequestID     string        `json:"cl_quote_req_id"`
	Status                   string        `json:"status"`
	Reason                   string        `json:"reason"`
	ResponseTime             types.Time    `json:"response_time_ns"`
	ExpiryTime               types.Time    `json:"expiry_time_ns"`
	Legs                     []OTCQuoteLeg `json:"leg_list"`
}

// UnmarshalJSON decodes a quote, taking its expiry time from expire_time_ns, an alias Crypto.com sends in its place, when
// expiry_time_ns is missing
func (q *OTCQuoteResponse) UnmarshalJSON(data []byte) error {
	type Alias OTCQuoteResponse
	aux := struct {
		*Alias
		ExpireTime types.Time `json:"expire_time_ns"`
	}{Alias: (*Alias)(q)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if q.ExpiryTime.Time().IsZero() {
		q.ExpiryTime = aux.ExpireTime
	}
	return nil
}

// OTCQuoteLeg is a leg of a quote: its instrument, quantity or notional, and the prices it is quoted at; Type is
// OTCLegTypePrice, and only set when the leg is priced
type OTCQuoteLeg struct {
	InstrumentName string       `json:"instrument_name"`
	Quantity       types.Number `json:"quantity"`
	Notional       types.Number `json:"notional"`
	Bid            types.Number `json:"bid"`
	Ask            types.Number `json:"ask"`
	Type           string       `json:"type"`
}

// otcQuoteListResponse is the result of a quote query; its count is the list's length, so it is left out
type otcQuoteListResponse struct {
	QuoteList []OTCQuoteResponse `json:"quote_list"`
}

// OTCDealRequest requests a deal on a quote. ClientDealID must be unique per deal, of letters, digits, underscores and
// hyphens; each leg is priced at the quote's bid or ask, with the instrument and the quantity or notional of the quote
// request's leg
type OTCDealRequest struct {
	ClientDealID   string
	QuoteID        string
	QuoteRequestID string
	Legs           []OTCDealLegRequest
}

// otcDealParams is the params of a request-deal request
type otcDealParams struct {
	DealType       string              `json:"deal_type"`
	ClientDealID   string              `json:"cl_deal_id"`
	QuoteID        string              `json:"quote_id"`
	QuoteRequestID string              `json:"quote_req_id"`
	Legs           []OTCDealLegRequest `json:"leg_list"`
}

// OTCDealLegRequest is a leg of a deal to request: an instrument bought or sold at a price, by quantity or by notional
// but not both
type OTCDealLegRequest struct {
	InstrumentName string       `json:"instrument_name"`
	Side           string       `json:"side"`
	Price          types.Number `json:"price"`
	Quantity       types.Number `json:"quantity,omitempty"`
	Notional       types.Number `json:"notional,omitempty"`
}

// OTCDealLeg is a leg of a deal as Crypto.com reports it, with its executed quantity and notional, and its type
type OTCDealLeg struct {
	InstrumentName   string              `json:"instrument_name"`
	Side             string              `json:"side"`
	Price            types.PreciseNumber `json:"price"`
	Quantity         types.PreciseNumber `json:"quantity"`
	Notional         types.PreciseNumber `json:"notional"`
	ExecutedQuantity types.PreciseNumber `json:"executed_quantity"`
	ExecutedNotional types.PreciseNumber `json:"executed_notional"`
	Type             string              `json:"type"`
}

// OTCDealResponse is a deal, as request-deal, the deal queries and the user.otc.deals channel report it; request-deal
// and the channel leave out its settlement arrangement and its create and settle times, and the queries its reason. A
// deal settled immediately has no settle time, and one settled early shows when in its update time
type OTCDealResponse struct {
	DealID                string       `json:"deal_id"`
	ClientDealID          string       `json:"cl_deal_id"`
	QuoteID               string       `json:"quote_id"`
	QuoteRequestID        string       `json:"quote_req_id"`
	DealType              string       `json:"deal_type"`
	DealStatus            string       `json:"deal_status"`
	SettlementArrangement string       `json:"settlement_arrangement"`
	Reason                string       `json:"reason"`
	CreateTime            types.Time   `json:"create_time_ns"`
	UpdateTime            types.Time   `json:"update_time_ns"`
	SettleTime            types.Time   `json:"settle_time_ns"`
	Legs                  []OTCDealLeg `json:"leg_list"`
}

// OTCDealHistoryRequest filters the open deals and the deal history: by deal ID, which overrides the time window, or by
// a time window as OTCQuoteRequestHistoryRequest's
type OTCDealHistoryRequest struct {
	DealID    string
	StartTime time.Time
	EndTime   time.Time
}

// otcDealHistoryParams is the params of a deal query, its times in Unix nanoseconds
type otcDealHistoryParams struct {
	DealType  string `json:"deal_type"`
	DealID    string `json:"deal_id,omitempty"`
	StartTime uint64 `json:"start_time,omitempty,string"`
	EndTime   uint64 `json:"end_time,omitempty,string"`
}

// otcWindow is an OTC query's time window in Unix nanoseconds, which Crypto.com recommends for paging accurately
type otcWindow struct {
	start, end uint64
}

// otcDealListResponse is the result of a deal query
type otcDealListResponse struct {
	DealList []OTCDealResponse `json:"deal_list"`
}

// OTCSettleLaterLimitResponse is an account's limit on deals settled later, and how much of it is used
type OTCSettleLaterLimitResponse struct {
	AccountID       string       `json:"account_id"`
	ConfiguredLimit types.Number `json:"configured_limit"`
	UsedLimit       types.Number `json:"used_limit"`
}

// OTCUnsettledAmountResponse is the amount of a currency pending settlement on a date; a negative amount is owed
type OTCUnsettledAmountResponse struct {
	SettlementDate time.Time
	Currency       currency.Code
	Amount         types.PreciseNumber
}

// otcUnsettledAmountsResponse is the result of get-unsettled-amounts: the amounts pending settlement keyed by
// settlement date, in Unix nanoseconds, then by currency
type otcUnsettledAmountsResponse struct {
	UnsettledCurrencies map[string]map[currency.Code]types.PreciseNumber `json:"unsettled_ccys"`
}
