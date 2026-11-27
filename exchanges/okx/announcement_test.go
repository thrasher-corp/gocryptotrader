package okx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := e.GetAnnouncements(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	for _, tc := range []struct {
		name string
		req  *AnnouncementsRequest
		exp  *AnnouncementsResponse
	}{
		{
			name: "every announcement",
			req:  &AnnouncementsRequest{},
			exp: &AnnouncementsResponse{
				TotalPage: types.Number(94),
				Details: []Announcement{
					{
						Title:               "How do I update Android System WebView?",
						AnnouncementType:    "announcements-others",
						BusinessPublishTime: types.Time(time.UnixMilli(1795017600000)),
						PublishTime:         types.Time(time.UnixMilli(1794938814998)),
						URL:                 "https://www.okx.com/en-au/help/how-do-i-update-android-system-webview",
					},
					{
						Title:               "OKX will launch AERO/USDC for spot trading",
						AnnouncementType:    "announcements-new-listings",
						BusinessPublishTime: types.Time(time.UnixMilli(1794387600000)),
						PublishTime:         types.Time(time.UnixMilli(1794389409337)),
						URL:                 "https://www.okx.com/en-au/help/okx-will-launch-aero-usdc-for-spot-trading",
					},
				},
			},
		},
		{
			name: "a type's second page",
			req:  &AnnouncementsRequest{AnnouncementType: "announcements-new-listings", Page: 2},
			exp: &AnnouncementsResponse{
				TotalPage: types.Number(23),
				Details: []Announcement{
					{
						Title:               "OKX to list TAO/USDT (Bittensor) for spot trading",
						AnnouncementType:    "announcements-new-listings",
						BusinessPublishTime: types.Time(time.UnixMilli(1782788400000)),
						PublishTime:         types.Time(time.UnixMilli(1782788410038)),
						URL:                 "https://www.okx.com/en-au/help/okx-to-list-tao-usdt-bittensor-for-spot-trading",
					},
					{
						Title:               "OKX will launch TAO/USDⓈ for spot trading",
						AnnouncementType:    "announcements-new-listings",
						BusinessPublishTime: types.Time(time.UnixMilli(1782788400000)),
						PublishTime:         types.Time(time.UnixMilli(1782788404917)),
						URL:                 "https://www.okx.com/en-au/help/okx-will-launch-tao-usds-for-spot-trading",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAnnouncements(contextGenerate(), tc.req)
			require.NoError(t, err, "GetAnnouncements must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAnnouncements should decode every field")
			} else {
				assert.NotNil(t, result, "GetAnnouncements should return a page")
			}
		})
	}
}

func TestGetAnnouncementTypes(t *testing.T) {
	t.Parallel()
	result, err := e.GetAnnouncementTypes(contextGenerate())
	require.NoError(t, err, "GetAnnouncementTypes must not error")
	if mockTests {
		exp := []AnnouncementTypeResponse{
			{AnnouncementType: "announcements-new-listings", AnnouncementTypeDescription: "New listings"},
			{AnnouncementType: "announcements-delistings", AnnouncementTypeDescription: "Delistings"},
			{AnnouncementType: "trading-updates-us-aus", AnnouncementTypeDescription: "Trading updates"},
		}
		assert.Equal(t, exp, result, "GetAnnouncementTypes should decode every field")
	}
	// No live test of the contents because announcement-types returns nothing to US based GitHub Actions
}
