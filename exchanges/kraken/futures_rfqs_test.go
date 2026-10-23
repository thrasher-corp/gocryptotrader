package kraken

import (
	"math"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	futuresTestSingleRFQUID = "50582b83-991c-4e8e-9033-2e44e99904ee"
	futuresTestSpreadRFQUID = "de302c7f-e952-449a-8be3-d2adfcef1d5c"
	futuresTestOwnRFQUID    = "6e2f9b14-3a7d-4c58-b0e1-5d9a2c7f8e63"
	futuresTestOfferUID     = "0b8e1f37-5c2a-4d6e-a8f1-9e3c7b2d4a61"
)

var (
	futuresTestCall85000 = currency.NewPairWithDelimiter("OF", "XBTUSD_261030_85000_C", currency.UnderscoreDelimiter)
	futuresTestCall90000 = currency.NewPairWithDelimiter("OF", "XBTUSD_261225_90000_C", currency.UnderscoreDelimiter)
	futuresTestPut100000 = currency.NewPairWithDelimiter("OF", "XBTUSD_261225_100000_P", currency.UnderscoreDelimiter)

	futuresTestSpreadBidSide = []FuturesRFQLegPrice{{Symbol: "OF_XBTUSD_261225_90000_C", Price: 2835.5}, {Symbol: "OF_XBTUSD_261225_100000_P", Price: 18720}}
	futuresTestSpreadAskSide = []FuturesRFQLegPrice{{Symbol: "OF_XBTUSD_261225_90000_C", Price: 2862}, {Symbol: "OF_XBTUSD_261225_100000_P", Price: 18680.25}}
)

// futuresRequireRFQRefusal requires err to be the APIError an RFQ request Kraken refused for reason returns
func futuresRequireRFQRefusal(t *testing.T, err error, reason string) {
	t.Helper()
	require.ErrorIs(t, err, errAPIResponse, "a refused RFQ request must return an API error")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "a refused RFQ request must return an APIError")
	assert.Equal(t, &APIError{Errors: []string{reason}}, apiErr, "APIError should hold the refusal reason")
}

func TestGetFuturesOpenRFQs(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesOpenRFQs(t.Context())
	require.NoError(t, err, "GetFuturesOpenRFQs must not error")
	if mockTests {
		exp := &FuturesOpenRFQsResponse{
			RFQs: []FuturesRFQ{
				{
					UID:       futuresTestSingleRFQUID,
					Expiry:    time.Date(2026, 10, 9, 0, 37, 34, 703041771, time.UTC),
					MarkPrice: 1401.32136111116,
					Legs:      []FuturesRFQLeg{{Symbol: "OF_XBTUSD_261030_85000_C", Size: 1, MarkPrice: 1401.32136111116}},
					Status:    "open",
				},
				{
					UID:       futuresTestSpreadRFQUID,
					Expiry:    time.Date(2026, 10, 9, 0, 38, 52, 296049138, time.UTC),
					MarkPrice: -15845.37937094951,
					Legs: []FuturesRFQLeg{
						{Symbol: "OF_XBTUSD_261225_90000_C", Size: 1, MarkPrice: 2849.81228164556},
						{Symbol: "OF_XBTUSD_261225_100000_P", Size: -1, MarkPrice: 18695.19165259507},
					},
					Status: "open",
				},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 36, 2, 616000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesOpenRFQs should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesOpenRFQs should return the server time")
}

func TestGetFuturesRFQ(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesRFQ(t.Context(), "")
	require.ErrorIs(t, err, errFuturesRFQUIDEmpty, "GetFuturesRFQ must reject an empty RFQ UID")

	uid := "229be048-f3a0-493d-98d4-91152b9ef363"
	if !mockTests {
		open, err := e.GetFuturesOpenRFQs(t.Context())
		require.NoError(t, err, "GetFuturesOpenRFQs must not error")
		if len(open.RFQs) == 0 {
			t.Skip("no RFQ is open")
		}
		uid = open.RFQs[0].UID
	}
	result, err := e.GetFuturesRFQ(t.Context(), uid)
	require.NoError(t, err, "GetFuturesRFQ must not error")
	if mockTests {
		exp := &FuturesRFQResponse{
			RFQ: FuturesRFQ{
				UID:       uid,
				Expiry:    time.Date(2026, 10, 9, 1, 9, 4, 724578888, time.UTC),
				MarkPrice: -15912.07980955285,
				Legs: []FuturesRFQLeg{
					{Symbol: "OF_XBTUSD_261225_90000_C", Size: 1, MarkPrice: 2815.40170373304},
					{Symbol: "OF_XBTUSD_261225_100000_P", Size: -1, MarkPrice: 18727.48151328589},
				},
				Status: "expired",
			},
			ServerTime: time.Date(2026, 10, 9, 1, 9, 6, 977000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesRFQ should decode every field")
		return
	}
	assert.Equal(t, uid, result.RFQ.UID, "GetFuturesRFQ should return the requested RFQ")
}

func TestGetFuturesOpenRFQOffers(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOpenRFQOffers(t.Context())
	require.NoError(t, err, "GetFuturesOpenRFQOffers must not error")
	if mockTests {
		exp := &FuturesRFQOffersResponse{
			Offers: []FuturesRFQOffer{
				{
					UID:            futuresTestOfferUID,
					RFQUID:         futuresTestSingleRFQUID,
					PlacementTime:  time.Date(2026, 10, 9, 0, 36, 20, 512000000, time.UTC),
					LastUpdateTime: time.Date(2026, 10, 9, 0, 36, 41, 207000000, time.UTC),
					Bid:            new(types.Number(1395.5)),
					Ask:            new(types.Number(1408.25)),
					Status:         "open",
				},
				{
					UID:            "93d0c6a1-2e4b-4f8a-b7c5-1a6d9e3f0b28",
					RFQUID:         futuresTestSpreadRFQUID,
					PlacementTime:  time.Date(2026, 10, 9, 0, 36, 25, 44000000, time.UTC),
					LastUpdateTime: time.Date(2026, 10, 9, 0, 36, 25, 44000000, time.UTC),
					BidSide:        futuresTestSpreadBidSide,
					AskSide:        futuresTestSpreadAskSide,
					Status:         "open",
				},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 36, 44, 871000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesOpenRFQOffers should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesOpenRFQOffers should return the server time")
}

func TestGetFuturesClosedRFQOffers(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesClosedRFQOffers(t.Context())
	require.NoError(t, err, "GetFuturesClosedRFQOffers must not error")
	if mockTests {
		exp := &FuturesRFQOffersResponse{
			Offers: []FuturesRFQOffer{
				{
					UID:            "5f7a2d90-8c1e-4b3f-a6d2-7e9b0c4f1a85",
					RFQUID:         "f1c3a5e7-9b2d-4f6a-8c0e-2d4b6f8a0c1e",
					PlacementTime:  time.Date(2026, 10, 9, 0, 21, 5, 338000000, time.UTC),
					LastUpdateTime: time.Date(2026, 10, 9, 0, 22, 47, 901000000, time.UTC),
					Bid:            new(types.Number(-15890.75)),
					Ask:            new(types.Number(-15801.5)),
					Status:         "filled_ask_side",
				},
				{
					UID:            "e8b4c2a6-0d9f-4e1b-9a3c-5f7d1b8e2c04",
					RFQUID:         "a2b4c6d8-e0f1-4a3b-9c5d-7e9f1a3b5c7d",
					PlacementTime:  time.Date(2026, 10, 9, 0, 12, 16, 750000000, time.UTC),
					LastUpdateTime: time.Date(2026, 10, 9, 0, 13, 40, 120000000, time.UTC),
					BidSide:        []FuturesRFQLegPrice{{Symbol: "OF_XBTUSD_261030_85000_C", Price: 1385}},
					AskSide:        []FuturesRFQLegPrice{{Symbol: "OF_XBTUSD_261030_85000_C", Price: 1421.5}},
					Status:         "expired",
				},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 39, 30, 652000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesClosedRFQOffers should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesClosedRFQOffers should return the server time")
}

func TestPlaceFuturesRFQOffer(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceFuturesRFQOffer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceFuturesRFQOffer must reject a nil request")
	_, err = e.PlaceFuturesRFQOffer(t.Context(), &FuturesPlaceRFQOfferRequest{Bid: new(1395.5)})
	require.ErrorIs(t, err, errFuturesRFQUIDEmpty, "PlaceFuturesRFQOffer must reject an empty RFQ UID")
	_, err = e.PlaceFuturesRFQOffer(t.Context(), &FuturesPlaceRFQOfferRequest{RFQUID: futuresTestSingleRFQUID})
	require.ErrorIs(t, err, errFuturesRFQPriceEmpty, "PlaceFuturesRFQOffer must reject an offer without a price")
	for _, price := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err = e.PlaceFuturesRFQOffer(t.Context(), &FuturesPlaceRFQOfferRequest{RFQUID: futuresTestSingleRFQUID, Bid: new(price)})
		require.ErrorIsf(t, err, errFuturesRFQInvalidPrice, "PlaceFuturesRFQOffer must reject a bid of %v", price)
		_, err = e.PlaceFuturesRFQOffer(t.Context(), &FuturesPlaceRFQOfferRequest{RFQUID: futuresTestSingleRFQUID, Ask: new(price)})
		require.ErrorIsf(t, err, errFuturesRFQInvalidPrice, "PlaceFuturesRFQOffer must reject an ask of %v", price)
	}
	_, err = e.PlaceFuturesRFQOffer(t.Context(), &FuturesPlaceRFQOfferRequest{RFQUID: futuresTestSingleRFQUID, Ask: new(1408.25), BidSide: []FuturesRFQOfferLeg{{Contract: futuresTestCall85000, Price: 1395.5}}})
	require.ErrorIs(t, err, errFuturesRFQPricingConflict, "PlaceFuturesRFQOffer must reject package and per-leg prices together")
	_, err = e.PlaceFuturesRFQOffer(t.Context(), &FuturesPlaceRFQOfferRequest{RFQUID: futuresTestSingleRFQUID, BidSide: []FuturesRFQOfferLeg{{Price: 1395.5}}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceFuturesRFQOffer must reject a bid leg without a contract")
	_, err = e.PlaceFuturesRFQOffer(t.Context(), &FuturesPlaceRFQOfferRequest{RFQUID: futuresTestSingleRFQUID, AskSide: []FuturesRFQOfferLeg{{Price: 1408.25}}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceFuturesRFQOffer must reject an ask leg without a contract")
	_, err = e.PlaceFuturesRFQOffer(t.Context(), &FuturesPlaceRFQOfferRequest{RFQUID: futuresTestSingleRFQUID, AskSide: []FuturesRFQOfferLeg{{Contract: futuresTestCall85000, Price: math.NaN()}}})
	require.Error(t, err, "PlaceFuturesRFQOffer must reject a leg price JSON cannot encode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name   string
		req    *FuturesPlaceRFQOfferRequest
		exp    *FuturesPlaceRFQOfferResponse
		reason string
	}{
		{
			name: "package",
			req:  &FuturesPlaceRFQOfferRequest{RFQUID: futuresTestSingleRFQUID, Bid: new(1395.5), Ask: new(1408.25)},
			exp:  &FuturesPlaceRFQOfferResponse{OfferUID: futuresTestOfferUID, ServerTime: time.Date(2026, 10, 9, 0, 36, 20, 512000000, time.UTC)},
		},
		{
			name: "per leg",
			req: &FuturesPlaceRFQOfferRequest{
				RFQUID:  futuresTestSpreadRFQUID,
				BidSide: []FuturesRFQOfferLeg{{Contract: futuresTestCall90000, Price: 2835.5}, {Contract: futuresTestPut100000, Price: 18720}},
				AskSide: []FuturesRFQOfferLeg{{Contract: futuresTestCall90000, Price: 2862}, {Contract: futuresTestPut100000, Price: 18680.25}},
			},
			exp: &FuturesPlaceRFQOfferResponse{OfferUID: "93d0c6a1-2e4b-4f8a-b7c5-1a6d9e3f0b28", ServerTime: time.Date(2026, 10, 9, 0, 36, 25, 44000000, time.UTC)},
		},
		{
			name:   "refused",
			req:    &FuturesPlaceRFQOfferRequest{RFQUID: futuresTestSpreadRFQUID, Bid: new(0.0)},
			reason: "priceTooFarFromMarket",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceFuturesRFQOffer(t.Context(), tc.req)
			if tc.reason != "" {
				futuresRequireRFQRefusal(t, err, tc.reason)
				return
			}
			require.NoError(t, err, "PlaceFuturesRFQOffer must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceFuturesRFQOffer should decode every field")
				return
			}
			assert.NotEmpty(t, result.OfferUID, "PlaceFuturesRFQOffer should return the offer's UID")
		})
	}
}

func TestCancelFuturesRFQOffer(t *testing.T) {
	t.Parallel()
	err := e.CancelFuturesRFQOffer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelFuturesRFQOffer must reject a nil request")
	err = e.CancelFuturesRFQOffer(t.Context(), &FuturesCancelRFQOfferRequest{OfferUID: futuresTestOfferUID})
	require.ErrorIs(t, err, errFuturesRFQUIDEmpty, "CancelFuturesRFQOffer must reject an empty RFQ UID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.CancelFuturesRFQOffer(t.Context(), &FuturesCancelRFQOfferRequest{RFQUID: futuresTestSingleRFQUID, OfferUID: futuresTestOfferUID})
	require.NoError(t, err, "CancelFuturesRFQOffer must not error")
	err = e.CancelFuturesRFQOffer(t.Context(), &FuturesCancelRFQOfferRequest{RFQUID: futuresTestSpreadRFQUID})
	futuresRequireRFQRefusal(t, err, "offerNotFound")
}

func TestGetFuturesAccountOpenRFQs(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesAccountOpenRFQs(t.Context())
	require.NoError(t, err, "GetFuturesAccountOpenRFQs must not error")
	if mockTests {
		exp := &FuturesAccountOpenRFQsResponse{
			RFQs: []FuturesAccountRFQ{
				{
					UID:       futuresTestOwnRFQUID,
					Expiry:    time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC),
					MarkPrice: 1418.2754063,
					Legs:      []FuturesRFQLeg{{Symbol: "OF_XBTUSD_261030_85000_C", Size: 1, MarkPrice: 1418.2754063}},
					Status:    "open",
					BestBid:   new(1395.5),
				},
				{
					UID:       "b7d9f1a3-c5e7-4f9b-8d1f-3a5c7e9b1d2f",
					Expiry:    time.Date(2026, 10, 9, 1, 5, 30, 250000000, time.UTC),
					MarkPrice: -15838.6104129,
					Legs: []FuturesRFQLeg{
						{Symbol: "OF_XBTUSD_261225_90000_C", Size: 1, MarkPrice: 2851.0433817},
						{Symbol: "OF_XBTUSD_261225_100000_P", Size: -1, MarkPrice: 18689.6537946},
					},
					Status:  "open",
					BestBid: new(-15884.5),
					BestAsk: new(-15818.25),
					BidSide: futuresTestSpreadBidSide,
					AskSide: futuresTestSpreadAskSide,
				},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 50, 11, 742000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesAccountOpenRFQs should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesAccountOpenRFQs should return the server time")
}

func TestCreateFuturesRFQ(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFuturesRFQ(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFuturesRFQ must reject a nil request")
	_, err = e.CreateFuturesRFQ(t.Context(), &FuturesCreateRFQRequest{})
	require.ErrorIs(t, err, errFuturesRFQLegsEmpty, "CreateFuturesRFQ must reject an RFQ without legs")
	_, err = e.CreateFuturesRFQ(t.Context(), &FuturesCreateRFQRequest{Legs: make([]FuturesRFQRequestedLeg, 101)})
	require.ErrorIs(t, err, errFuturesRFQTooManyLegs, "CreateFuturesRFQ must reject more than 100 legs")
	_, err = e.CreateFuturesRFQ(t.Context(), &FuturesCreateRFQRequest{Legs: []FuturesRFQRequestedLeg{{Size: 1}}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreateFuturesRFQ must reject a leg without a contract")
	for _, pair := range partialTestPairs {
		_, err = e.CreateFuturesRFQ(t.Context(), &FuturesCreateRFQRequest{Legs: []FuturesRFQRequestedLeg{{Contract: pair, Size: 1}}})
		require.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "CreateFuturesRFQ must reject %q, which lacks a base or a quote", pair)
	}
	_, err = e.CreateFuturesRFQ(t.Context(), &FuturesCreateRFQRequest{Legs: []FuturesRFQRequestedLeg{{Contract: futuresTestCall85000}}})
	require.ErrorIs(t, err, errFuturesRFQLegSizeZero, "CreateFuturesRFQ must reject a leg without a size")
	_, err = e.CreateFuturesRFQ(t.Context(), &FuturesCreateRFQRequest{Legs: []FuturesRFQRequestedLeg{{Contract: futuresTestCall85000, Size: 1}, {Contract: futuresTestCall85000, Size: -1}}})
	require.ErrorIs(t, err, errFuturesRFQDuplicateLeg, "CreateFuturesRFQ must reject a contract in two legs")
	_, err = e.CreateFuturesRFQ(t.Context(), &FuturesCreateRFQRequest{Legs: []FuturesRFQRequestedLeg{{Contract: futuresTestCall85000, Size: math.Inf(1)}}})
	require.Error(t, err, "CreateFuturesRFQ must reject a leg size JSON cannot encode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name   string
		req    *FuturesCreateRFQRequest
		exp    *FuturesRFQUIDResponse
		reason string
	}{
		{
			name: "expiry",
			req: &FuturesCreateRFQRequest{
				Legs:   []FuturesRFQRequestedLeg{{Contract: futuresTestCall85000, Size: 1}},
				Expiry: time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC),
			},
			exp: &FuturesRFQUIDResponse{RFQUID: futuresTestOwnRFQUID, ServerTime: time.Date(2026, 10, 9, 0, 49, 58, 127000000, time.UTC)},
		},
		{
			name:   "refused",
			req:    &FuturesCreateRFQRequest{Legs: []FuturesRFQRequestedLeg{{Contract: futuresTestCall90000, Size: 1}, {Contract: futuresTestPut100000, Size: -1}}},
			reason: "tooManyOpenRfqs",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CreateFuturesRFQ(t.Context(), tc.req)
			if tc.reason != "" {
				futuresRequireRFQRefusal(t, err, tc.reason)
				return
			}
			require.NoError(t, err, "CreateFuturesRFQ must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CreateFuturesRFQ should decode every field")
				return
			}
			assert.NotEmpty(t, result.RFQUID, "CreateFuturesRFQ should return the RFQ's UID")
		})
	}
}

func TestCancelFuturesRFQ(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesRFQ(t.Context(), "")
	require.ErrorIs(t, err, errFuturesRFQUIDEmpty, "CancelFuturesRFQ must reject an empty RFQ UID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelFuturesRFQ(t.Context(), futuresTestOwnRFQUID)
	require.NoError(t, err, "CancelFuturesRFQ must not error")
	if mockTests {
		exp := &FuturesRFQUIDResponse{RFQUID: futuresTestOwnRFQUID, ServerTime: time.Date(2026, 10, 9, 0, 52, 44, 301000000, time.UTC)}
		assert.Equal(t, exp, result, "CancelFuturesRFQ should decode every field")
	}
	_, err = e.CancelFuturesRFQ(t.Context(), "0d4c3b2a-1f0e-4d9c-8b7a-6f5e4d3c2b1a")
	futuresRequireRFQRefusal(t, err, "rfqNotFound")
}

func TestAcceptFuturesRFQOffer(t *testing.T) {
	t.Parallel()
	_, err := e.AcceptFuturesRFQOffer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AcceptFuturesRFQOffer must reject a nil request")
	_, err = e.AcceptFuturesRFQOffer(t.Context(), &FuturesAcceptRFQOfferRequest{Bid: new(1395.5)})
	require.ErrorIs(t, err, errFuturesRFQUIDEmpty, "AcceptFuturesRFQOffer must reject an empty RFQ UID")
	_, err = e.AcceptFuturesRFQOffer(t.Context(), &FuturesAcceptRFQOfferRequest{RFQUID: futuresTestOwnRFQUID})
	require.ErrorIs(t, err, errFuturesRFQAcceptSide, "AcceptFuturesRFQOffer must reject an acceptance without a price")
	_, err = e.AcceptFuturesRFQOffer(t.Context(), &FuturesAcceptRFQOfferRequest{RFQUID: futuresTestOwnRFQUID, Bid: new(1395.5), Ask: new(1408.25)})
	require.ErrorIs(t, err, errFuturesRFQAcceptSide, "AcceptFuturesRFQOffer must reject accepting both sides")
	for _, price := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err = e.AcceptFuturesRFQOffer(t.Context(), &FuturesAcceptRFQOfferRequest{RFQUID: futuresTestOwnRFQUID, Bid: new(price)})
		require.ErrorIsf(t, err, errFuturesRFQInvalidPrice, "AcceptFuturesRFQOffer must reject a bid of %v", price)
		_, err = e.AcceptFuturesRFQOffer(t.Context(), &FuturesAcceptRFQOfferRequest{RFQUID: futuresTestOwnRFQUID, Ask: new(price)})
		require.ErrorIsf(t, err, errFuturesRFQInvalidPrice, "AcceptFuturesRFQOffer must reject an ask of %v", price)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AcceptFuturesRFQOffer(t.Context(), &FuturesAcceptRFQOfferRequest{RFQUID: futuresTestOwnRFQUID, Bid: new(1395.5)})
	require.NoError(t, err, "AcceptFuturesRFQOffer must not error")
	if mockTests {
		exp := &FuturesRFQUIDResponse{RFQUID: futuresTestOwnRFQUID, ServerTime: time.Date(2026, 10, 9, 0, 51, 17, 538000000, time.UTC)}
		assert.Equal(t, exp, result, "AcceptFuturesRFQOffer should decode every field")
	}
	_, err = e.AcceptFuturesRFQOffer(t.Context(), &FuturesAcceptRFQOfferRequest{RFQUID: futuresTestOwnRFQUID, Ask: new(1408.25)})
	futuresRequireRFQRefusal(t, err, "offerNoLongerValid")
}

func TestFuturesRFQRequest(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/derivatives/api/v3/rfqs/open-rfqs/refused":
			_, _ = w.Write([]byte(`{"result":"success","serverTime":"2026-10-09T00:52:46.815Z","reason":"rfqNotFound"}`))
		case "/derivatives/api/v3/rfqs/open-rfqs/array":
			_, _ = w.Write([]byte(`[]`))
		case "/derivatives/api/v3/rfqs/open-rfqs/forbidden":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"result":"error","serverTime":"2026-10-09T00:52:47.002Z","errors":[{"code":95,"message":"Account is not permitted to perform the requested trading operation"}]}`))
		default:
			_, _ = w.Write([]byte(`{"result":"success","serverTime":"2026-10-09T00:52:44.301Z","rfqUid":"` + futuresTestOwnRFQUID + `"}`))
		}
	})

	var result *FuturesRFQUIDResponse
	require.NoError(t, ex.futuresRFQRequest(t.Context(), http.MethodDelete, "/api/v3/rfqs/open-rfqs/"+futuresTestOwnRFQUID, nil, &result), "futuresRFQRequest must not error")
	exp := &FuturesRFQUIDResponse{RFQUID: futuresTestOwnRFQUID, ServerTime: time.Date(2026, 10, 9, 0, 52, 44, 301000000, time.UTC)}
	assert.Equal(t, exp, result, "futuresRFQRequest should decode the response")

	assert.NoError(t, ex.futuresRFQRequest(t.Context(), http.MethodDelete, "/api/v3/rfqs/open-rfqs/"+futuresTestOwnRFQUID, url.Values{}, nil), "futuresRFQRequest should accept a nil result")

	err := ex.futuresRFQRequest(t.Context(), http.MethodDelete, "/api/v3/rfqs/open-rfqs/refused", nil, &result)
	futuresRequireRFQRefusal(t, err, "rfqNotFound")

	err = ex.futuresRFQRequest(t.Context(), http.MethodDelete, "/api/v3/rfqs/open-rfqs/array", nil, &result)
	assert.Error(t, err, "futuresRFQRequest should reject a response that is not an object")

	err = ex.futuresRFQRequest(t.Context(), http.MethodPost, "/api/v3/rfqs/open-rfqs/forbidden", nil, &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "futuresRFQRequest must return the API error of an error status")
	assert.Equal(t, []string{"code 95: Account is not permitted to perform the requested trading operation"}, apiErr.Errors, "APIError should hold the coded error")
	assert.ErrorIs(t, err, request.ErrBadStatus, "futuresRFQRequest should keep the status error")
}

func TestFuturesRFQCheckPackagePrices(t *testing.T) {
	t.Parallel()
	assert.NoError(t, futuresRFQCheckPackagePrices(nil, nil), "futuresRFQCheckPackagePrices should accept no prices")
	for _, price := range []float64{0, -1395.5, 1408.25} {
		assert.NoErrorf(t, futuresRFQCheckPackagePrices(&price, &price), "futuresRFQCheckPackagePrices should accept a price of %v", price)
	}
	for _, price := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		assert.ErrorIsf(t, futuresRFQCheckPackagePrices(&price, nil), errFuturesRFQInvalidPrice, "futuresRFQCheckPackagePrices should reject a bid of %v", price)
		assert.ErrorIsf(t, futuresRFQCheckPackagePrices(nil, &price), errFuturesRFQInvalidPrice, "futuresRFQCheckPackagePrices should reject an ask of %v", price)
	}
}

func TestFuturesRFQLegPrices(t *testing.T) {
	t.Parallel()
	_, err := e.futuresRFQLegPrices([]FuturesRFQOfferLeg{{Contract: futuresTestPair, Price: 1}, {Price: 1}})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "futuresRFQLegPrices should reject a leg without a contract")
	for _, pair := range partialTestPairs {
		_, err = e.futuresRFQLegPrices([]FuturesRFQOfferLeg{{Contract: pair, Price: 1}})
		assert.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "futuresRFQLegPrices should reject %q, which lacks a base or a quote", pair)
	}
	prices, err := e.futuresRFQLegPrices([]FuturesRFQOfferLeg{{Contract: futuresTestPair, Price: 82000.5}, {Contract: piXBTUSDPair, Price: -10}})
	require.NoError(t, err, "futuresRFQLegPrices must not error")
	assert.JSONEq(t, `[{"tradeable":"PF_XBTUSD","price":82000.5},{"tradeable":"PI_XBTUSD","price":-10}]`, prices, "futuresRFQLegPrices should list each leg's market symbol and price")
	prices, err = e.futuresRFQLegPrices(nil)
	require.NoError(t, err, "futuresRFQLegPrices must not error without legs")
	assert.Equal(t, "[]", prices, "futuresRFQLegPrices should return an empty list without legs")
}
