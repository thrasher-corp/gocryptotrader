package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSmallBalanceCoins(t *testing.T) {
	t.Parallel()
	_, err := e.GetSmallBalanceCoins(t.Context(), "", currency.XRP)
	require.ErrorIs(t, err, errMissingAccountType, "GetSmallBalanceCoins must reject an empty account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSmallBalanceCoins(t.Context(), "eb_convert_uta", currency.XRP)
	require.NoError(t, err, "GetSmallBalanceCoins must not error")
	if mockTests {
		// The conversion and fee fields are reserved and empty here; RequestSmallBalanceQuote's fixture fills them
		exp := &SmallBalanceCoinsResponse{
			SmallAssetCoins: []SmallBalanceCoin{
				{
					FromCoin:         currency.XRP,
					SupportConvert:   1,
					AvailableBalance: 0.0002,
					BaseValue:        0.00036554008,
				},
			},
			SupportToCoins: []currency.Code{
				currency.NewCode("MNT"),
				currency.USDT,
				currency.USDC,
			},
		}
		assert.Equal(t, exp, result, "GetSmallBalanceCoins should decode every field")
		return
	}
	assert.NotEmpty(t, result.SupportToCoins, "GetSmallBalanceCoins should return the coins to convert to")
}

func TestRequestSmallBalanceQuote(t *testing.T) {
	t.Parallel()
	_, err := e.RequestSmallBalanceQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestSmallBalanceQuote must reject a nil request")
	arg := &SmallBalanceQuoteRequest{}
	_, err = e.RequestSmallBalanceQuote(t.Context(), arg)
	require.ErrorIs(t, err, errMissingAccountType, "RequestSmallBalanceQuote must reject an empty account type")
	arg.AccountType = "eb_convert_uta"
	_, err = e.RequestSmallBalanceQuote(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RequestSmallBalanceQuote must reject an empty coin list")
	arg.FromCoins = []currency.Code{currency.XRP, currency.EMPTYCODE}
	_, err = e.RequestSmallBalanceQuote(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RequestSmallBalanceQuote must reject an empty coin in the list")
	arg.FromCoins = []currency.Code{currency.XRP, currency.SOL}
	_, err = e.RequestSmallBalanceQuote(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RequestSmallBalanceQuote must reject an empty to coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.ToCoin = currency.USDC
	result, err := e.RequestSmallBalanceQuote(t.Context(), arg)
	require.NoError(t, err, "RequestSmallBalanceQuote must not error")
	if mockTests {
		exp := &SmallBalanceQuoteResponse{
			QuoteID: "1010075157602510902217555968",
			Result: SmallBalanceQuote{
				QuoteCreateTime: types.Time(time.UnixMilli(1766126593232)),
				QuoteExpireTime: types.Time(time.UnixMilli(1766126623231)),
				ExchangeCoins: []SmallBalanceCoin{
					{
						FromCoin:         currency.XRP,
						SupportConvert:   1,
						AvailableBalance: 0.0002,
						BaseValue:        0.00024536,
						ToCoin:           currency.USDC,
						ToAmount:         0.000359866676661744,
						ExchangeRate:     1.79933338330872,
						FeeInfo: &SmallBalanceFeeInfo{
							FeeCoin: currency.USDC,
							Amount:  0.000007344217891056,
							FeeRate: 0.02,
						},
						TaxFeeInfo: &SmallBalanceTaxFeeInfo{
							TotalAmount: 0.0000001,
							FeeCoin:     currency.USDC,
							TaxFeeItems: []json.RawMessage{},
						},
					},
					{
						FromCoin:         currency.SOL,
						SupportConvert:   1,
						AvailableBalance: 0.000003,
						BaseValue:        0.00036837,
						ToCoin:           currency.USDC,
						ToAmount:         0.00035721396701649,
						ExchangeRate:     119.07132233883026,
						FeeInfo: &SmallBalanceFeeInfo{
							FeeCoin: currency.USDC,
							Amount:  0.00000729008095952,
							FeeRate: 0.02,
						},
						TaxFeeInfo: &SmallBalanceTaxFeeInfo{
							TaxFeeItems: []json.RawMessage{},
						},
					},
				},
				TotalFeeInfo: SmallBalanceFeeInfo{
					FeeCoin: currency.USDC,
					Amount:  0.000014634298850576,
					FeeRate: 0.02,
				},
				TotalTaxFeeInfo: SmallBalanceTaxFeeInfo{
					TotalAmount: 0.0000001,
					FeeCoin:     currency.USDC,
					TaxFeeItems: []json.RawMessage{},
				},
			},
		}
		assert.Equal(t, exp, result, "RequestSmallBalanceQuote should decode every field")
		return
	}
	assert.NotEmpty(t, result.QuoteID, "RequestSmallBalanceQuote should return a quote")
}

func TestConfirmSmallBalanceQuote(t *testing.T) {
	t.Parallel()
	_, err := e.ConfirmSmallBalanceQuote(t.Context(), "")
	require.ErrorIs(t, err, errQuoteIDRequired, "ConfirmSmallBalanceQuote must reject an empty quote ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ConfirmSmallBalanceQuote(t.Context(), "1010075157602517596339322880")
	require.NoError(t, err, "ConfirmSmallBalanceQuote must not error")
	if mockTests {
		exp := &SmallBalanceConfirmQuoteResponse{
			QuoteID:               "1010075157602517596339322880",
			ExchangeTransactionID: "1010075157602517596339322880",
			SubmitTime:            types.Time(time.UnixMilli(1766128195512)),
			Status:                "processing",
			Message:               "submitted",
		}
		assert.Equal(t, exp, result, "ConfirmSmallBalanceQuote should decode every field")
		return
	}
	assert.NotEmpty(t, result.Status, "ConfirmSmallBalanceQuote should return the status")
}

func TestGetSmallBalanceExchangeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSmallBalanceExchangeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSmallBalanceExchangeHistory must reject a nil request")
	_, err = e.GetSmallBalanceExchangeHistory(t.Context(), &SmallBalanceExchangeHistoryRequest{StartTime: time.UnixMilli(1766128195000), EndTime: time.UnixMilli(1766100000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSmallBalanceExchangeHistory must reject a reversed window")

	arg := &SmallBalanceExchangeHistoryRequest{AccountType: "eb_convert_uta"}
	if mockTests {
		// The quote ID names the fixture's conversion, which a live account does not have
		arg.QuoteID = "1010075157602517596339322880"
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSmallBalanceExchangeHistory(t.Context(), arg)
	require.NoError(t, err, "GetSmallBalanceExchangeHistory must not error")
	t.Run("second page of a window", func(t *testing.T) {
		t.Parallel()
		// The cursor is a page number, and the size the page's length
		page, err := e.GetSmallBalanceExchangeHistory(t.Context(), &SmallBalanceExchangeHistoryRequest{
			AccountType: "eb_convert_uta",
			StartTime:   time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			EndTime:     time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC),
			Cursor:      2,
			Size:        1,
		})
		require.NoError(t, err, "GetSmallBalanceExchangeHistory must not error")
		if mockTests {
			exp := &SmallBalanceExchangeHistoryResponse{
				Cursor:     2,
				Size:       1,
				LastPage:   2,
				TotalCount: 2,
				Records: []SmallBalanceExchangeRecord{
					{
						AccountType:           "eb_convert_uta",
						ExchangeTransactionID: "1010075157602517596339320001",
						ToCoin:                currency.USDT,
						ToAmount:              0.84035,
						SubRecords: []SmallBalanceExchangeSubRecord{
							{
								FromCoin:   currency.DOGE,
								FromAmount: 3.5,
								ToCoin:     currency.USDT,
								ToAmount:   0.84035,
								FeeCoin:    currency.USDT,
								FeeAmount:  0.01715,
								Status:     "success",
								TaxFeeInfo: SmallBalanceTaxFeeInfo{
									TaxFeeItems: []json.RawMessage{},
								},
							},
						},
						Status:         "success",
						CreatedAt:      types.Time(time.UnixMilli(1788400000000)),
						ExchangeSource: "small_asset_uta",
						FeeCoin:        currency.USDT,
						TotalFeeAmount: 0.01715,
						TotalTaxFeeInfo: SmallBalanceTaxFeeInfo{
							TaxFeeItems: []json.RawMessage{},
						},
					},
				},
			}
			assert.Equal(t, exp, page, "GetSmallBalanceExchangeHistory should send the window and page and decode the page")
			return
		}
		assert.NotNil(t, page, "GetSmallBalanceExchangeHistory should return a page")
	})
	if mockTests {
		exp := &SmallBalanceExchangeHistoryResponse{
			Cursor:     1,
			Size:       50,
			LastPage:   1,
			TotalCount: 1,
			Records: []SmallBalanceExchangeRecord{
				{
					AccountType:           "eb_convert_uta",
					ExchangeTransactionID: "1010075157602517596339322880",
					ToCoin:                currency.USDC,
					ToAmount:              0.000728325793503221,
					SubRecords: []SmallBalanceExchangeSubRecord{
						{
							FromCoin:   currency.SOL,
							FromAmount: 0.000003,
							ToCoin:     currency.USDC,
							ToAmount:   0.000363439538230885,
							FeeCoin:    currency.USDC,
							FeeAmount:  0.000007417133433283,
							Status:     "success",
							TaxFeeInfo: SmallBalanceTaxFeeInfo{
								TotalAmount: 0.0000001,
								FeeCoin:     currency.USDC,
								TaxFeeItems: []json.RawMessage{},
							},
						},
						{
							FromCoin:   currency.XRP,
							FromAmount: 0.0002,
							ToCoin:     currency.USDC,
							ToAmount:   0.000364886255272336,
							FeeCoin:    currency.USDC,
							FeeAmount:  0.000007446658270864,
							Status:     "success",
							TaxFeeInfo: SmallBalanceTaxFeeInfo{
								TaxFeeItems: []json.RawMessage{},
							},
						},
					},
					Status:         "success",
					CreatedAt:      types.Time(time.Unix(1766128195, 0)),
					ExchangeSource: "small_asset_uta",
					FeeCoin:        currency.USDC,
					TotalFeeAmount: 0.000014863791704147,
					TotalTaxFeeInfo: SmallBalanceTaxFeeInfo{
						TotalAmount: 0.0000001,
						FeeCoin:     currency.USDC,
						TaxFeeItems: []json.RawMessage{},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetSmallBalanceExchangeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSmallBalanceExchangeHistory should return a page")
}
