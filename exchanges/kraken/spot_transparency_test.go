package kraken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestGetPreTradeData(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreTradeData(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPreTradeData must reject an empty pair")

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
	_, err := e.GetPostTradeData(t.Context(), &PostTradeDataRequest{Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetPostTradeData must reject a count above 1000")
	from := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	to := from.Add(10 * time.Minute)
	_, err = e.GetPostTradeData(t.Context(), &PostTradeDataRequest{From: to, To: from})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPostTradeData must reject a reversed window")

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
