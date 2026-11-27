package kraken

import "time"

// FuturesNotificationsResponse holds the platform's notifications
type FuturesNotificationsResponse struct {
	Notifications []FuturesNotification `json:"notifications"`
	ServerTime    time.Time             `json:"serverTime"`
}

// FuturesNotification is a platform notification. A high priority maintenance notification means downtime from its
// EffectiveTime
type FuturesNotification struct {
	EffectiveTime time.Time `json:"effectiveTime"`
	Note          string    `json:"note"`
	// Priority is low, medium or high
	Priority string `json:"priority"`
	// Type is new_feature, bug_fix, settlement, general, maintenance or market
	Type string `json:"type"`
	// ExpectedDowntimeMinutes is 0 when no downtime is expected
	ExpectedDowntimeMinutes uint64 `json:"expectedDowntimeMinutes"`
}
