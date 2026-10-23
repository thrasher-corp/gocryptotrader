package bybit

import "time"

// InstitutionWhitelistIPResponse holds a dedicated institution hostname and the IP addresses whitelisted for it
type InstitutionWhitelistIPResponse struct {
	Name           string   `json:"name"` // Client name
	EndpointDomain string   `json:"endpointDomain"`
	IPList         []string `json:"ipList"`
}

// InstitutionIPChangeLogRequest holds the filters of GetInstitutionIPChangeLog; every field is optional
type InstitutionIPChangeLogRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// InstitutionIPChangeLogResponse holds a page of institution whitelist IP changes
type InstitutionIPChangeLogResponse struct {
	List           []InstitutionIPChange `json:"list"`
	NextPageCursor string                `json:"nextPageCursor"`
}

// InstitutionIPChange holds a change to an institution whitelist IP address
type InstitutionIPChange struct {
	OperationType string    `json:"operationType"` // ADD, REMOVE, ENABLE or DISABLE
	IPAddress     string    `json:"ipAddress"`     // An address or a CIDR range
	DomainType    string    `json:"domainType"`    // GW: HTTP hostname, WS: websocket hostname
	ChangeReason  string    `json:"changeReason"`
	CreatedAt     time.Time `json:"createdAt"`
}
