package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AnnouncementsRequest holds the parameters for GetAnnouncements
type AnnouncementsRequest struct {
	// Locale is the announcement language, such as en-US; required
	Locale string
	Type   string
	Tag    string
	Page   uint64
	Limit  uint64
}

// AnnouncementsResponse holds a page of announcements
type AnnouncementsResponse struct {
	Total uint64         `json:"total"`
	List  []Announcement `json:"list"`
}

// Announcement holds a Bybit announcement
type Announcement struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Type        AnnouncementType `json:"type"`
	Tags        []string         `json:"tags"`
	URL         string           `json:"url"`
	// DateTimestamp is the time the author gave the announcement
	DateTimestamp types.Time `json:"dateTimestamp"`
	// StartDateTimestamp and EndDateTimestamp bound the event an announcement of type latest_activities describes.
	// The docs table spells their keys startDataTimestamp and endDataTimestamp, but Bybit sends these spellings
	StartDateTimestamp types.Time `json:"startDateTimestamp"`
	EndDateTimestamp   types.Time `json:"endDateTimestamp"`
	PublishTime        types.Time `json:"publishTime"`
}

// AnnouncementType holds an announcement's type
type AnnouncementType struct {
	Title string `json:"title"`
	Key   string `json:"key"`
}

// DemoTradingFundsRequest holds the parameters for RequestDemoTradingFunds
type DemoTradingFundsRequest struct {
	// AdjustType is 0 (the default) to add demo funds or 1 to reduce them
	AdjustType        uint64
	UTADemoApplyMoney []DemoApplyMoney
}

// DemoApplyMoney holds a coin and the amount of it to add or reduce. Each request is capped per coin, at 15 BTC,
// 200 ETH, 100000 USDT and 100000 USDC
type DemoApplyMoney struct {
	Coin   currency.Code
	Amount float64
}

// CreateDemoAccountResponse holds the ID of a demo trading account
type CreateDemoAccountResponse struct {
	SubMemberID string `json:"subMemberId"`
}

// SystemStatusResponse holds platform maintenance and service incidents
type SystemStatusResponse struct {
	List []SystemStatus `json:"list"`
}

// SystemStatus holds a platform maintenance or service incident
type SystemStatus struct {
	ID    string     `json:"id"`
	Title string     `json:"title"`
	State string     `json:"state"`
	Begin types.Time `json:"begin"`
	// End is the expected end time until the maintenance completes, then the actual end time
	End          types.Time `json:"end"`
	Href         string     `json:"href"`
	ServiceTypes []uint64   `json:"serviceTypes"`
	Product      []uint64   `json:"product"`
	UIDSuffix    []uint64   `json:"uidSuffix"`
	// MaintainType and Environment are documented as strings but sent as bare numbers in the docs' examples, so
	// they decode either form
	MaintainType types.Number `json:"maintainType"`
	Environment  types.Number `json:"env"`
}
