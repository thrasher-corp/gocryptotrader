package gateio

import (
	"context"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// GetAnnouncementArticles retrieves Gate's announcement articles, filtered and paged by arg when it is set
func (e *Exchange) GetAnnouncementArticles(ctx context.Context, arg *AnnouncementArticlesRequest) (*AnnouncementArticleListResponse, error) {
	var resp gateioAPIResponse[*AnnouncementArticleListResponse]
	if err := e.sendPublicPostRequest(ctx, publicAnnouncementArticlesEPL, "ann/list_article", announcementArticlesBody(arg), &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data, nil
}

// announcementArticlesBody converts the request into the body Gate expects, which carries most numbers as strings and
// only needs the pinned and empty content flags when they differ from their defaults
func announcementArticlesBody(arg *AnnouncementArticlesRequest) *announcementArticlesRequestBody {
	body := &announcementArticlesRequestBody{}
	if arg == nil {
		return body
	}
	body.TitleQuery = arg.TitleQuery
	body.Tags = arg.Tags
	body.CategoryName = arg.CategoryName
	body.Language = arg.Language
	if arg.Page > 0 {
		body.Page = strconv.FormatUint(arg.Page, 10)
	}
	if arg.Size > 0 {
		body.Size = strconv.FormatUint(arg.Size, 10)
	}
	if arg.RecentDays > 0 {
		body.Timer = strconv.FormatUint(arg.RecentDays, 10)
	}
	if arg.CategoryLevel > 0 {
		body.CategoryLevel = strconv.FormatUint(arg.CategoryLevel, 10)
	}
	if arg.SubWebsiteID > 0 {
		body.SubWebsiteID = strconv.FormatUint(arg.SubWebsiteID, 10)
	}
	if arg.ExcludePinned {
		body.Pinned = new(uint8)
	}
	if !arg.UpdatedAfter.IsZero() {
		body.UpdateAfter = unixSeconds(arg.UpdatedAfter)
	}
	if arg.IncludeEmptyContent {
		body.FilterEmptyContent = new(uint8)
	}
	return body
}
