package okx

import (
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SystemStatusResponse holds a system maintenance event
type SystemStatusResponse struct {
	Title               string     `json:"title"`
	State               string     `json:"state"`
	Begin               types.Time `json:"begin"`
	End                 types.Time `json:"end"`
	PreOpenBegin        types.Time `json:"preOpenBegin"`
	Hyperlink           string     `json:"href"`
	ServiceType         string     `json:"serviceType"`
	System              string     `json:"system"`
	ScheduleDescription string     `json:"scheDesc"`
	MaintenanceType     string     `json:"maintType"`
	Environment         string     `json:"env"`
	// PushTime is sent by the websocket status channel only
	PushTime types.Time `json:"ts"`
}
