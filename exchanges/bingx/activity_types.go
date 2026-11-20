package bingx

import "time"

// ActivityListRequest holds the parameters of Query Activity List and Query Public Activity List; the filters combine
type ActivityListRequest struct {
	// ActivityName matches part of an activity's title
	ActivityName string
	// Status is ongoing when zero
	Status ActivityStatus
	// TagURLKey is an activity type tag's URL key, such as highReward, spot, trading_tournament or draw, not its name
	TagURLKey string
	// PageIndex counts from 1, and is 1 when zero
	PageIndex uint64
	// PageSize is 10 when zero, and at most 100
	PageSize uint64
}

// ActivityListResponse holds a page of activities
type ActivityListResponse struct {
	// Total is the number of matching activities, not the number on the page
	Total                 uint64     `json:"total"`
	ActivityCenterInfoVOs []Activity `json:"activityCenterInfoVos"`
}

// Activity is a BingX activity, such as a trading competition; BingX sends null for an unavailable value
type Activity struct {
	ActivityID  uint64        `json:"activityId"`
	ActivityURL string        `json:"activityUrl"`
	Title       string        `json:"title"`
	Tags        []ActivityTag `json:"tags"`
	// BeginTime and EndTime carry the +08:00 offset BingX sends them in
	BeginTime              time.Time `json:"beginTime"`
	EndTime                time.Time `json:"endTime"`
	ActivityCenterPCImage  string    `json:"activityCenterPcImg"`
	ActivityCenterAppImage string    `json:"activityCenterAppImg"`
}

// ActivityTag is an activity's display tag
type ActivityTag struct {
	Name string `json:"name"`
}
