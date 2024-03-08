package bitget

import "time"

// AnnResp holds information on announcements, returned by QueryAnnouncements
type AnnResp struct {
	Data []struct {
		AnnID    string    `json:"annId"`
		AnnTitle string    `json:"annTitle"`
		AnnDesc  string    `json:"annDesc"`
		CTime    time.Time `json:"cTime"`
		Language string    `json:"language"`
		AnnURL   string    `json:"annUrl"`
	} `json:"data"`
}
