package kraken

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestGetPreTradeData(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreTradeData(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPreTradeData must reject an empty pair")
	for _, pair := range partialTestPairs {
		_, err = e.GetPreTradeData(t.Context(), pair)
		require.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "GetPreTradeData must reject %q, which lacks a base or a quote", pair)
	}

	result, err := e.GetPreTradeData(t.Context(), currency.NewPair(currency.ETH, currency.XBT))
	require.NoError(t, err, "GetPreTradeData must not error")
	if mockTests {
		exp := &PreTradeDataResponse{
			Symbol:            "ETHXBT",
			Description:       "Ethereum / Bitcoin",
			BaseAsset:         currency.ETH,
			BaseNotation:      "UNIT",
			BaseDTICode:       "1STM49Z4L",
			BaseDTIShortName:  "ETH",
			QuoteAsset:        currency.XBT,
			QuoteNotation:     "UNIT",
			QuoteDTICode:      "V15WLZJMF",
			QuoteDTIShortName: "BTC,XBT",
			Venue:             "PGSL",
			System:            "CLOB",
			Bids: []PreTradePriceLevel{
				{
					Side:            "BUY",
					Price:           0.030296,
					Quantity:        0.36263383,
					OrderCount:      3,
					SubmissionTime:  time.Date(2026, 10, 9, 0, 20, 12, 896619358, time.UTC),
					PublicationTime: time.Date(2026, 10, 9, 0, 20, 13, 896619358, time.UTC),
				},
			},
			Asks: []PreTradePriceLevel{
				{
					Side:            "SELL",
					Price:           0.030297,
					Quantity:        0.00100152,
					OrderCount:      1,
					SubmissionTime:  time.Date(2026, 10, 9, 0, 18, 11, 798474318, time.UTC),
					PublicationTime: time.Date(2026, 10, 9, 0, 18, 12, 798474318, time.UTC),
				},
			},
		}
		assert.Equal(t, exp, result, "GetPreTradeData should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "GetPreTradeData should return bids")
}

func TestGetPostTradeData(t *testing.T) {
	t.Parallel()
	_, err := e.GetPostTradeData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPostTradeData must reject a nil request")
	_, err = e.GetPostTradeData(t.Context(), &PostTradeDataRequest{Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetPostTradeData must reject a count above 1000")
	from := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	to := from.Add(10 * time.Minute)
	_, err = e.GetPostTradeData(t.Context(), &PostTradeDataRequest{From: to, To: from})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPostTradeData must reject a reversed window")
	_, err = e.GetPostTradeData(t.Context(), &PostTradeDataRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPostTradeData must reject an empty pair, which Kraken refuses")
	for _, pair := range partialTestPairs {
		_, err = e.GetPostTradeData(t.Context(), &PostTradeDataRequest{Pair: pair})
		require.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "GetPostTradeData must reject %q, which lacks a base or a quote", pair)
	}
	_, err = new(Exchange).GetPostTradeData(t.Context(), &PostTradeDataRequest{Pair: spotTestPair})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetPostTradeData must return the error formatting the pair")

	t.Run("no trades", func(t *testing.T) {
		t.Parallel()
		ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/0/public/PostTrade", r.URL.Path, "GetPostTradeData should call Post-Trade Data")
			assert.Equal(t, url.Values{"count": {"3"}, "from_ts": {"2030-01-01T00:00:00Z"}, "symbol": {"XBTUSD"}}, r.URL.Query(), "GetPostTradeData should send the pair, start and count")
			_, err := w.Write([]byte(`{"error":[],"result":{"last_ts":"","count":0,"trades":[]}}`))
			assert.NoError(t, err, "Write should not error")
		})
		result, err := ex.GetPostTradeData(t.Context(), &PostTradeDataRequest{Pair: spotTestPair, From: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), Count: 3})
		require.NoError(t, err, "GetPostTradeData must not error")
		assert.Equal(t, &PostTradeDataResponse{Trades: []PostTradeData{}}, result, "GetPostTradeData should decode a page holding no trades")
	})

	result, err := e.GetPostTradeData(t.Context(), &PostTradeDataRequest{Pair: currency.NewPair(currency.ETH, currency.XBT), From: from, To: to, Count: 1})
	require.NoError(t, err, "GetPostTradeData must not error")
	if mockTests {
		exp := &PostTradeDataResponse{
			LastTime: time.Date(2026, 10, 9, 0, 7, 53, 340172214, time.UTC),
			Count:    1,
			Trades: []PostTradeData{
				{
					TradeID:           "OJUSHU-KIV2Z-DAOLPU",
					Price:             0.030286,
					Quantity:          0.3254176,
					Symbol:            "ETHXBT",
					Description:       "Ethereum / Bitcoin",
					BaseAsset:         currency.ETH,
					BaseNotation:      "UNIT",
					BaseDTICode:       "1STM49Z4L",
					BaseDTIShortName:  "ETH",
					QuoteAsset:        currency.XBT,
					QuoteNotation:     "UNIT",
					QuoteDTICode:      "V15WLZJMF",
					QuoteDTIShortName: "BTC,XBT",
					TradeVenue:        "PGSL",
					TradeTime:         time.Date(2026, 10, 9, 0, 7, 53, 340172214, time.UTC),
					PublicationVenue:  "PGSL",
					PublicationTime:   time.Date(2026, 10, 9, 0, 7, 53, 340172215, time.UTC),
				},
			},
		}
		assert.Equal(t, exp, result, "GetPostTradeData should decode every field")
		return
	}
	assert.LessOrEqual(t, result.Count, uint64(1), "GetPostTradeData should return at most the requested count")
}

func TestPostTradeDataResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	decoded := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		prev time.Time
		data string
		exp  PostTradeDataResponse
	}{
		{
			name: "page holding trades",
			data: `{"last_ts":"2026-10-09T00:07:53.340172214Z","count":1,"trades":[{"trade_id":"OJUSHU-KIV2Z-DAOLPU"}]}`,
			exp:  PostTradeDataResponse{LastTime: time.Date(2026, 10, 9, 0, 7, 53, 340172214, time.UTC), Count: 1, Trades: []PostTradeData{{TradeID: "OJUSHU-KIV2Z-DAOLPU"}}},
		},
		{name: "page holding no trades", data: `{"last_ts":"","count":0,"trades":[]}`, exp: PostTradeDataResponse{Trades: []PostTradeData{}}},
		{name: "empty last time after a decoded one", prev: decoded, data: `{"last_ts":""}`},
		{name: "absent last time", prev: decoded, data: `{"count":0}`, exp: PostTradeDataResponse{LastTime: decoded}},
		{name: "null last time", prev: decoded, data: `{"last_ts":null}`, exp: PostTradeDataResponse{LastTime: decoded}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := PostTradeDataResponse{LastTime: tc.prev}
			require.NoError(t, json.Unmarshal([]byte(tc.data), &got), "Unmarshal must not error")
			assert.Equal(t, tc.exp, got, "Unmarshal should decode the post-trade data")
		})
	}
	var got PostTradeDataResponse
	var parseErr *time.ParseError
	assert.ErrorAs(t, json.Unmarshal([]byte(`{"last_ts":"2026-10-09"}`), &got), &parseErr, "Unmarshal should reject a last time it cannot parse")
	assert.Error(t, json.Unmarshal([]byte(`{"last_ts":1791504473}`), &got), "Unmarshal should reject a last time that is not a string")
}
