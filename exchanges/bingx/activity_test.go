package bingx

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetActivityList(t *testing.T) {
	t.Parallel()
	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetActivityList(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrNilPointer, "GetActivityList should reject a nil request")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetActivityList(t.Context(), &ActivityListRequest{
			Status:    ActivityStatusOngoing,
			TagURLKey: "trading_tournament",
			PageIndex: 1,
			PageSize:  10,
		})
		require.NoError(t, err, "GetActivityList must not error")
		require.NotNil(t, result, "GetActivityList must return a result")
		if !mockTests {
			assert.GreaterOrEqual(t, result.Total, uint64(len(result.ActivityCenterInfoVOs)), "Total should count at least the activities on the page")
			return
		}
		begin, err := time.Parse(time.RFC3339, "2026-10-13T10:00:00.000+08:00")
		require.NoError(t, err, "time.Parse must not error")
		end, err := time.Parse(time.RFC3339, "2026-10-31T23:59:59.000+08:00")
		require.NoError(t, err, "time.Parse must not error")
		exp := &ActivityListResponse{
			Total: 1,
			ActivityCenterInfoVOs: []Activity{{
				ActivityID:  12345,
				ActivityURL: "https://bingx.com/en/events/",
				Title:       "Trading Competition",
				Tags:        []ActivityTag{{Name: "Trading"}},
				BeginTime:   begin,
				EndTime:     end,
			}},
		}
		assert.Equal(t, exp, result, "GetActivityList should decode the documented example")
	})
}

func TestGetPublicActivityList(t *testing.T) {
	t.Parallel()
	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetPublicActivityList(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrNilPointer, "GetPublicActivityList should reject a nil request")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetPublicActivityList(t.Context(), &ActivityListRequest{
			Status:    ActivityStatusOngoing,
			TagURLKey: "trading_tournament",
			PageIndex: 1,
			PageSize:  10,
		})
		require.NoError(t, err, "GetPublicActivityList must not error")
		require.NotNil(t, result, "GetPublicActivityList must return a result")
		if !mockTests {
			assert.LessOrEqual(t, len(result.ActivityCenterInfoVOs), 10, "GetPublicActivityList should return at most a page of activities")
			assert.GreaterOrEqual(t, result.Total, uint64(len(result.ActivityCenterInfoVOs)), "Total should count at least the activities on the page")
			return
		}
		begin, err := time.Parse(time.RFC3339, "2026-11-11T10:00:00.000+08:00")
		require.NoError(t, err, "time.Parse must not error")
		end, err := time.Parse(time.RFC3339, "2026-12-01T23:59:59.000+08:00")
		require.NoError(t, err, "time.Parse must not error")
		exp := &ActivityListResponse{
			Total: 1,
			ActivityCenterInfoVOs: []Activity{{
				ActivityID:             154,
				ActivityURL:            "https://bingx.com/zh-tc/activity/novsuperx?ch=bingx_Eventcenter",
				Title:                  "Black Friday! Register for an airdrop and share $6,000,000!",
				Tags:                   []ActivityTag{{Name: "Trading Tournament"}, {Name: "High Rewards"}},
				BeginTime:              begin,
				EndTime:                end,
				ActivityCenterPCImage:  "https://static-app.bb-os.com/social_feed_pictures/20261106/ad33167ff8d0538cad2fbee57838ce494705f299952b8d959a16280b77e8ba61.png",
				ActivityCenterAppImage: "https://static-app.bb-os.com/social_feed_pictures/20261106/a6a6fe4055340b773d5385e60b6915f82cc9d4cb2909c94a4d222fcc14bc5268.png",
			}},
		}
		assert.Equal(t, exp, result, "GetPublicActivityList should decode the recorded reply")
	})
}

func TestActivityListParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *ActivityListRequest
		exp  url.Values
	}{
		{name: "no filters", arg: &ActivityListRequest{}, exp: url.Values{}},
		{
			name: "every filter",
			arg:  &ActivityListRequest{ActivityName: "Black Friday", Status: ActivityStatusStartingSoon, TagURLKey: "draw", PageIndex: 2, PageSize: 100},
			exp:  url.Values{"activityName": {"Black Friday"}, "status": {"1"}, "tagUrlKey": {"draw"}, "pageIndex": {"2"}, "pageSize": {"100"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, activityListParams(tc.arg), "activityListParams should return the set filters")
		})
	}
}
