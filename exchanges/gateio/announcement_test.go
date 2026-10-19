package gateio

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetAnnouncementArticles(t *testing.T) {
	t.Parallel()
	_, err := newFixedResponseExchange(t, "/api/v4/ann/list_article", `{"code":0,"data":null}`).GetAnnouncementArticles(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNoResults, "GetAnnouncementArticles must reject a response without data")

	result, err := e.GetAnnouncementArticles(t.Context(), &AnnouncementArticlesRequest{Page: 1, Size: 2})
	require.NoError(t, err, "GetAnnouncementArticles must not error")
	if mockTests {
		require.Len(t, result.List, 2, "GetAnnouncementArticles must return the recorded articles")
		exp := &AnnouncementArticle{
			ID:           102133,
			Title:        "Gate WCTC S9 Global Trading Competition",
			Brief:        "Gate WCTC S9 Global Trading Competition is here.",
			Created:      "3 days",
			Updated:      "3 days",
			ReleaseDate:  "2026-10-09",
			ReleaseTime:  types.Time(time.Unix(1791532603, 0)),
			CreateTime:   types.Time(time.Unix(1791532603, 0)),
			UpdateTime:   types.Time(time.Unix(1791534265, 0)),
			URL:          "/announcements/article/102133",
			Views:        2252,
			AuthorID:     3096169,
			Tags:         "Trading Competition",
			Category:     "Activities",
			PinnedStatus: 1,
			CategoryID:   32,
			Source:       "Gate.announcements",
			ContentLanguages: []*AnnouncementArticleLanguage{
				{Code: "en", Name: "English"},
				{Code: "cn", Name: "简体中文"},
			},
		}
		assert.Equal(t, uint64(500), result.Total, "GetAnnouncementArticles should decode the total")
		assert.Equal(t, exp, result.List[1], "GetAnnouncementArticles should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetAnnouncementArticles should return articles")
}

func TestGetAnnouncementArticlesRequest(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Equal(t, http.MethodPost, r.Method, "request method should be POST")
		assert.Equal(t, "/api/v4/ann/list_article", r.URL.Path, "request path should carry the API version")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "request should declare a JSON body")
		assert.Empty(t, r.Header.Get("KEY"), "request should not send an API key")
		var body map[string]string
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "request body should be a JSON object")
		response := `{"code":0,"message":"success","data":{"total":1,"list":[]},"version":"1.1.4"}`
		if body["page"] != "1" {
			response = `{"code":1,"message":"invalid page","data":null}`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "writing the response should not error")
	}))
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	result, err := ex.GetAnnouncementArticles(t.Context(), &AnnouncementArticlesRequest{Page: 1})
	require.NoError(t, err, "GetAnnouncementArticles must not error")
	assert.Equal(t, &AnnouncementArticleList{Total: 1, List: []*AnnouncementArticle{}}, result, "GetAnnouncementArticles should decode the response")

	_, err = ex.GetAnnouncementArticles(t.Context(), &AnnouncementArticlesRequest{Page: 2})
	assert.ErrorContains(t, err, "invalid page", "GetAnnouncementArticles should surface an error in the response envelope")
	assert.Equal(t, int64(2), requests.Load(), "GetAnnouncementArticles should send one request per call")
}

func TestAnnouncementArticlesBody(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(announcementArticlesBody(nil))
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{}`, string(payload), "announcementArticlesBody should send an empty body for a nil request")

	payload, err = json.Marshal(announcementArticlesBody(&AnnouncementArticlesRequest{}))
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{}`, string(payload), "announcementArticlesBody should leave the pinned and empty content defaults to Gate")

	payload, err = json.Marshal(announcementArticlesBody(&AnnouncementArticlesRequest{
		TitleQuery:          "listing",
		Page:                2,
		Size:                5,
		Tags:                "Gate",
		RecentDays:          10,
		CategoryName:        "New Listings",
		CategoryLevel:       1,
		SubWebsiteID:        177,
		ExcludePinned:       true,
		UpdatedAfter:        time.Unix(1791532603, 0),
		Language:            "cn",
		IncludeEmptyContent: true,
	}))
	require.NoError(t, err, "Marshal must not error")
	exp := `{"title_query":"listing","page":"2","size":"5","tags":"Gate","timer":"10","cate_name":"New Listings","cate_level":"1","sub_website_id":"177","pinned":0,"update_after":1791532603,"lang":"cn","filter_empty_content":0}`
	assert.JSONEq(t, exp, string(payload), "announcementArticlesBody should send every filter in Gate's format")
}
