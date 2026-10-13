package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := e.GetAnnouncements(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAnnouncements must reject a nil request")
	_, err = e.GetAnnouncements(t.Context(), &AnnouncementsRequest{})
	require.ErrorIs(t, err, errLocaleRequired, "GetAnnouncements must reject an empty locale")

	result, err := e.GetAnnouncements(t.Context(), &AnnouncementsRequest{Locale: "en-US", Type: "latest_bybit_news", Tag: "Spot", Page: 1, Limit: 1})
	require.NoError(t, err, "GetAnnouncements must not error")
	if mockTests {
		exp := &AnnouncementsResponse{
			Total: 331,
			List: []Announcement{
				{
					Title:       "Bybit AI Now Supports Main Account Operations",
					Description: "Bybit AI Now Supports Main Account Operations",
					Type: AnnouncementType{
						Title: "Latest Bybit News",
						Key:   "latest_bybit_news",
					},
					Tags:               []string{"Spot", "Latest Updates", "Upgrades"},
					URL:                "https://announcements.bybit.com/en-US/article/bybit-ai-now-supports-main-account-operations--art91af7ea2483b/",
					DateTimestamp:      types.Time(time.UnixMilli(1790568000000)),
					StartDateTimestamp: types.Time(time.UnixMilli(1790568000000)),
					EndDateTimestamp:   types.Time(time.UnixMilli(1798675200000)),
					PublishTime:        types.Time(time.UnixMilli(1790676068000)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetAnnouncements should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetAnnouncements should return announcements")
}

func TestRequestDemoTradingFunds(t *testing.T) {
	t.Parallel()
	err := e.RequestDemoTradingFunds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestDemoTradingFunds must reject a nil request")

	if !mockTests {
		t.Skip("demo trading funds need demo account credentials and the demo trading host")
	}
	// The demo trading host has no default URL, so the mock serves it on the sandbox URL key
	ex := testInstance(t)
	mockURL, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "GetURL must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSandbox.String(), mockURL), "SetRunningURL must not error")
	err = ex.RequestDemoTradingFunds(t.Context(), &DemoTradingFundsRequest{
		AdjustType: 1,
		UTADemoApplyMoney: []DemoApplyMoney{
			{Coin: currency.USDT, Amount: 109},
			{Coin: currency.ETH, Amount: 1},
		},
	})
	require.NoError(t, err, "RequestDemoTradingFunds must not error")
}

func TestCreateDemoAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateDemoAccount(t.Context())
	require.NoError(t, err, "CreateDemoAccount must not error")
	if mockTests {
		assert.Equal(t, &CreateDemoAccountResponse{SubMemberID: "100000001"}, result, "CreateDemoAccount should decode every field")
		return
	}
	assert.NotEmpty(t, result.SubMemberID, "CreateDemoAccount should return the demo account ID")
}

func TestGetSystemStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemStatus(t.Context(), "4d95b2a0-587f-11f0-bcc9-56f28c94d6ea", "completed")
	require.NoError(t, err, "GetSystemStatus must not error")
	if mockTests {
		exp := &SystemStatusResponse{
			List: []SystemStatus{
				{
					ID:           "4d95b2a0-587f-11f0-bcc9-56f28c94d6ea",
					Title:        "t06",
					State:        "completed",
					Begin:        types.Time(time.UnixMilli(1751596902000)),
					End:          types.Time(time.UnixMilli(1751597011000)),
					Href:         "https://announcements.bybit.com/en/article/system-maintenance-4d95b2a0",
					ServiceTypes: []uint64{2, 3, 4, 5},
					Product:      []uint64{1, 2},
					UIDSuffix:    []uint64{3, 7},
					MaintainType: 1,
					Environment:  1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSystemStatus should decode every field")
		return
	}
	// Bybit lists an incident only while it is announced, so a live list is usually empty
	assert.NotNil(t, result, "GetSystemStatus should return a result")
}
