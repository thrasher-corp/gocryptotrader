package gateio

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// AnnouncementArticlesRequest holds the filters and paging for listing announcement articles
type AnnouncementArticlesRequest struct {
	TitleQuery string
	Page       uint64
	Size       uint64
	Tags       string
	// RecentDays limits the articles to those from the last N days
	RecentDays    uint64
	CategoryName  string
	CategoryLevel uint64 // 1 or 2
	// SubWebsiteID selects a subsite, such as 177 for Turkey; zero selects the main site
	SubWebsiteID uint64
	// ExcludePinned leaves out pinned articles, which Gate includes by default
	ExcludePinned bool
	UpdatedAfter  time.Time
	Language      string // Language code, such as "cn"
	// IncludeEmptyContent keeps articles without content in the requested language, which Gate excludes by default
	IncludeEmptyContent bool
}

// announcementArticlesRequestBody is the request body of the announcement article list endpoint
type announcementArticlesRequestBody struct {
	TitleQuery         string `json:"title_query,omitempty"`
	Page               string `json:"page,omitempty"`
	Size               string `json:"size,omitempty"`
	Tags               string `json:"tags,omitempty"`
	Timer              string `json:"timer,omitempty"`
	CategoryName       string `json:"cate_name,omitempty"`
	CategoryLevel      string `json:"cate_level,omitempty"`
	SubWebsiteID       string `json:"sub_website_id,omitempty"`
	Pinned             *uint8 `json:"pinned,omitempty"`
	UpdateAfter        uint64 `json:"update_after,omitempty"`
	Language           string `json:"lang,omitempty"`
	FilterEmptyContent *uint8 `json:"filter_empty_content,omitempty"`
}

// AnnouncementArticleListResponse holds a page of announcement articles
type AnnouncementArticleListResponse struct {
	Total uint64                 `json:"total"`
	List  []*AnnouncementArticle `json:"list"`
}

// AnnouncementArticle holds an announcement article summary
type AnnouncementArticle struct {
	ID          uint64 `json:"id"`
	Title       string `json:"title"`
	Brief       string `json:"brief"`
	Created     string `json:"created"` // Relative to now, such as 1 days, or a date such as 2026-10-09 UTC once older
	Updated     string `json:"updated"` // Relative to now, such as 1 days, or a date such as 2026-10-09 UTC once older
	ReleaseDate Date   `json:"release_time"`
	// ReleaseTime, CreateTime, UpdateTime and URL are returned by the live API but not documented
	ReleaseTime      types.Time                     `json:"release_timestamp"`
	CreateTime       types.Time                     `json:"created_t"`
	UpdateTime       types.Time                     `json:"updated_t"`
	URL              string                         `json:"url"`
	Views            uint64                         `json:"views"`
	Author           string                         `json:"author"`
	AuthorID         uint64                         `json:"author_id"`
	Tags             string                         `json:"tags"`
	Category         string                         `json:"cate"`
	PinnedStatus     uint64                         `json:"is_top"`
	CategoryID       uint64                         `json:"cate_id"`
	Source           string                         `json:"source"`
	ContentLanguages []*AnnouncementArticleLanguage `json:"content_lang_list"`
}

// AnnouncementArticleLanguage holds a language an announcement article is published in
type AnnouncementArticleLanguage struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
