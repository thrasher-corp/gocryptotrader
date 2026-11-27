package okx

import (
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AnnouncementsRequest holds the filters of the announcements
type AnnouncementsRequest struct {
	AnnouncementType string
	Page             uint64
}

// AnnouncementsResponse holds a page of announcements
type AnnouncementsResponse struct {
	TotalPage types.Number   `json:"totalPage"`
	Details   []Announcement `json:"details"`
}

// Announcement holds an announcement
type Announcement struct {
	Title            string `json:"title"`
	AnnouncementType string `json:"annType"`
	// BusinessPublishTime is the time the announcement page displays
	BusinessPublishTime types.Time `json:"businessPTime"`
	// PublishTime is when the announcement was first published, which OKX can report around 5 minutes late
	PublishTime types.Time `json:"pTime"`
	URL         string     `json:"url"`
}

// AnnouncementTypeResponse holds an announcement type and its description
type AnnouncementTypeResponse struct {
	AnnouncementType            string `json:"annType"`
	AnnouncementTypeDescription string `json:"annTypeDesc"`
}
