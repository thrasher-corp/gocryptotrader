package bybit

import "github.com/thrasher-corp/gocryptotrader/types"

// WebsocketSystemStatusUpdate is a push from the system.status topic (System Status)
type WebsocketSystemStatusUpdate struct {
	Topic     string                  `json:"topic"`
	Timestamp types.Time              `json:"ts"`
	Data      []WebsocketSystemStatus `json:"data"`
}

// WebsocketSystemStatus is a platform maintenance or service incident notice
type WebsocketSystemStatus struct {
	ID    string     `json:"id"`
	Title string     `json:"title"`
	State string     `json:"state"`
	Begin types.Time `json:"begin"`
	// End is the expected end time until the maintenance completes, and the actual end time afterwards
	End  types.Time `json:"end"`
	Href string     `json:"href"`
	// ServiceTypes and Product hold enum values: services 1 trading, 2 HTTP trading, 3 websocket trading, 4 private
	// websocket stream, 5 market data; products 1 futures, 2 spot, 3 option, 4 spread
	ServiceTypes []uint64 `json:"serviceTypes"`
	Product      []uint64 `json:"product"`
	// UIDSuffix lists the UID tail numbers affected
	UIDSuffix []uint64 `json:"uidSuffix"`
	// MaintainType and Environment are documented as strings but sent as bare numbers: maintenance 1 planned, 2 temporary,
	// 3 incident; environment 1 production, 2 production demo
	MaintainType types.Number `json:"maintainType"`
	Environment  types.Number `json:"env"`
}
