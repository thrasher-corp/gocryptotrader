package bitstamp

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// TransactionType is the type of user transaction
type TransactionType uint8

// Sort orders
const (
	SortAscending  = "asc"
	SortDescending = "desc"
)

// Travel Rule identity document types
const (
	IDTypeDriversLicence = "drivers_licence"
	IDTypeIDCard         = "id_card"
	IDTypePassport       = "passport"
	IDTypeMisc           = "misc"
)

// PaginationRequest holds the paging parameters of list endpoints
type PaginationRequest struct {
	Page    uint64
	PerPage uint64
}

// CustomerInfo holds Travel Rule details for a retail or corporate customer
type CustomerInfo struct {
	RetailInfo    *RetailInfo    `json:"retail_info,omitempty"`
	CorporateInfo *CorporateInfo `json:"corporate_info,omitempty"`
}

// RetailInfo holds Travel Rule details for a retail customer
// FirstName and LastName are required; DateOfBirth is sent as a date only
type RetailInfo struct {
	FirstName            string
	LastName             string
	DateOfBirth          time.Time
	PlaceOfBirth         string
	IDType               string
	IDNumber             string
	StreetAndHouseNumber string
	City                 string
	PostalCode           string
	Country              string
}

// CorporateInfo holds Travel Rule details for a corporate customer
// CompanyName is required
type CorporateInfo struct {
	CompanyName string `json:"company_name"`
	Address     string `json:"address,omitempty"`
	Country     string `json:"country,omitempty"`
	LEI         string `json:"lei,omitempty"`
	CompanyID   string `json:"company_id,omitempty"`
}

// errorResponse holds the fields Bitstamp uses to report a failed request
// Each field is decoded when formatted since reasons may be strings, lists or objects mapping fields to messages and
// codes may be strings or numbers
type errorResponse struct {
	Status              json.RawMessage `json:"status"`
	Reason              json.RawMessage `json:"reason"`
	Code                json.RawMessage `json:"code"`
	Error               json.RawMessage `json:"error"`
	Message             json.RawMessage `json:"message"`
	Field               json.RawMessage `json:"field"`
	ResponseCode        json.RawMessage `json:"response_code"`
	ResponseExplanation json.RawMessage `json:"response_explanation"`
	Errors              json.RawMessage `json:"errors"`
	// OrderID names the order to check when an order request timed out before the order was confirmed
	OrderID json.RawMessage `json:"order_id"`
}

// OrderSide decodes an order side sent as 0 for buy or 1 for sell, either quoted or bare
type OrderSide order.Side

// EnabledStatus decodes the availability of trading, deposits or withdrawals, which the API sends as "Enabled" or
// "Disabled", as whether it is "Enabled". Any other status decodes as not enabled, as the API's string did when compared
type EnabledStatus bool

// DateTime decodes a date and time sent either in RFC 3339 form or in Bitstamp's "2006-01-02 15:04:05" form, with or
// without fractional seconds, since the API documents both forms as date-time values; null and an empty string decode as
// the zero time
type DateTime time.Time
