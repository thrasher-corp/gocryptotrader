package bybit

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	p2pOrderID         = "1900004704665923584"
	p2pAdvertisementID = "1898988222063644672"
	p2pMakerUID        = "100000003"
	p2pUserMaskID      = "s00000000000000000000000000000001"
	p2pChatSessionID   = "aes_encrypted_session_id"
)

func TestGetP2POnlineAds(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2POnlineAds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2POnlineAds must reject a nil request")
	r := &P2POnlineAdsRequest{}
	_, err = e.GetP2POnlineAds(t.Context(), r)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetP2POnlineAds must require a token ID")
	r.TokenID = currency.USDT
	_, err = e.GetP2POnlineAds(t.Context(), r)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetP2POnlineAds must require a currency ID")
	r.CurrencyID = currency.EUR
	_, err = e.GetP2POnlineAds(t.Context(), r)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetP2POnlineAds must require a side")
	r.Side = "1"
	r.Page = 1
	r.Size = 10

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2POnlineAds(t.Context(), r)
	require.NoError(t, err, "GetP2POnlineAds must not error")
	if mockTests {
		exp := &P2POnlineAdsResponse{
			Count: 2,
			Items: []P2POnlineAd{
				{
					ID:               "1899658238346616832",
					AccountID:        "100000004",
					UserID:           preciseNumber(t, "100000003"),
					NickName:         "TestMaker",
					TokenID:          currency.USDT,
					TokenName:        "USDT",
					CurrencyID:       currency.EUR,
					Side:             1,
					Price:            0.93,
					LastQuantity:     9000,
					Quantity:         10000,
					FrozenQuantity:   500,
					ExecutedQuantity: 500,
					MinAmount:        200,
					MaxAmount:        9300,
					Remark:           "Bank transfer only",
					Status:           10,
					CreateDate:       types.Time(time.Unix(1741748793, 0)),
					Payments: []string{
						"14",
					},
					OrderNumber:       12,
					FinishNumber:      11,
					RecentOrderNumber: 5,
					RecentExecuteRate: 91.6,
					Fee:               0.5,
					IsOnline:          true,
					LastLogoutTime:    types.Time(time.Unix(1741749194, 0)),
					Blocked:           "N",
					MakerContact:      true,
					SymbolInfo: P2PSymbolInfo{
						ID:                    "13",
						ExchangeID:            "301",
						OrgID:                 "9001",
						TokenID:               currency.USDT,
						CurrencyID:            currency.EUR,
						Status:                1,
						LowerLimitAlarm:       90,
						UpperLimitAlarm:       110,
						ItemDownRange:         70,
						ItemUpRange:           130,
						CurrencyMinQuote:      2,
						CurrencyMaxQuote:      46500,
						CurrencyLowerMaxQuote: 2,
						TokenMinQuote:         1,
						TokenMaxQuote:         50000,
						KYCCurrencyLimit:      900,
						ItemSideLimit:         3,
						BuyFeeRate:            0.001,
						SellFeeRate:           0.0015,
						OrderAutoCancelMinute: 15,
						OrderFinishMinute:     10,
						TradeSide:             9,
						Currency: P2PSymbolCurrency{
							ID:         "14",
							ExchangeID: "1",
							OrgID:      "9001",
							CurrencyID: currency.EUR,
							Scale:      3,
						},
						Token: P2PSymbolToken{
							ID:         "1",
							ExchangeID: "1",
							OrgID:      "9001",
							TokenID:    currency.USDT,
							Scale:      4,
							Sequence:   1,
						},
						BuyAd: P2PSymbolAdConfig{
							PaymentPeriods: []uint64{
								15,
								30,
							},
						},
						SellAd: P2PSymbolAdConfig{
							PaymentPeriods: []uint64{
								15,
								30,
								45,
							},
						},
					},
					TradingPreferenceSet: P2PTradingPreferenceSet{
						HasUnPostAd:               true,
						IsKYC:                     true,
						IsEmail:                   true,
						IsMobile:                  true,
						HasRegisterTime:           true,
						RegisterTimeThreshold:     15,
						OrderFinishNumberDay30:    60,
						CompleteRateDay30:         95,
						NationalLimit:             "MYS",
						HasOrderFinishNumberDay30: true,
						HasCompleteRateDay30:      true,
						HasNationalLimit:          true,
					},
					Version:      2,
					AuthStatus:   1,
					Recommend:    true,
					RecommendTag: "FEATURED",
					AuthTag: []string{
						"BA",
						"VA",
					},
					UserType:      "ORG",
					ItemType:      "ORIGIN",
					PaymentPeriod: 15,
				},
				{
					ID:           "1899659847717838848",
					AccountID:    "100000004",
					UserID:       preciseNumber(t, "100000003"),
					NickName:     "TestMaker",
					TokenID:      currency.USDT,
					CurrencyID:   currency.EUR,
					Side:         1,
					PriceType:    1,
					Price:        0.92,
					Premium:      101.5,
					LastQuantity: 20000,
					Quantity:     20000,
					MinAmount:    20,
					MaxAmount:    18400,
					Status:       10,
					CreateDate:   types.Time(time.Unix(1741749177, 0)),
					Payments: []string{
						"377",
						"14",
					},
					Blocked: "N",
					SymbolInfo: P2PSymbolInfo{
						ID:                    "13",
						ExchangeID:            "301",
						OrgID:                 "9001",
						TokenID:               currency.USDT,
						CurrencyID:            currency.EUR,
						Status:                1,
						LowerLimitAlarm:       90,
						UpperLimitAlarm:       110,
						ItemDownRange:         70,
						ItemUpRange:           130,
						CurrencyMinQuote:      2,
						CurrencyMaxQuote:      46500,
						CurrencyLowerMaxQuote: 2,
						TokenMinQuote:         1,
						TokenMaxQuote:         50000,
						KYCCurrencyLimit:      900,
						ItemSideLimit:         3,
						OrderAutoCancelMinute: 15,
						OrderFinishMinute:     10,
						TradeSide:             9,
						Currency: P2PSymbolCurrency{
							ID:         "14",
							ExchangeID: "1",
							OrgID:      "9001",
							CurrencyID: currency.EUR,
							Scale:      3,
						},
						Token: P2PSymbolToken{
							ID:         "1",
							ExchangeID: "1",
							OrgID:      "9001",
							TokenID:    currency.USDT,
							Scale:      4,
							Sequence:   1,
						},
					},
					TradingPreferenceSet: P2PTradingPreferenceSet{
						IsKYC: true,
					},
					AuthStatus: 2,
					AuthTag: []string{
						"GA",
					},
					UserType:      "PERSONAL",
					ItemType:      "BULK",
					PaymentPeriod: 30,
				},
			},
		}
		assert.Equal(t, exp, result, "GetP2POnlineAds should decode every field")
		return
	}
	assert.NotNil(t, result.Items, "GetP2POnlineAds should return ads")
}

func TestPostP2PAd(t *testing.T) {
	t.Parallel()
	_, err := e.PostP2PAd(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PostP2PAd must reject a nil request")
	r := &P2PPostAdRequest{}
	_, err = e.PostP2PAd(t.Context(), r)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PostP2PAd must require a token ID")
	r.TokenID = currency.USDT
	_, err = e.PostP2PAd(t.Context(), r)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PostP2PAd must require a currency ID")
	r.CurrencyID = currency.EUR
	_, err = e.PostP2PAd(t.Context(), r)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PostP2PAd must require a side")
	r.Side = "0"
	_, err = e.PostP2PAd(t.Context(), r)
	require.ErrorIs(t, err, errPriceTypeRequired, "PostP2PAd must require a price type")
	r.PriceType = "0"
	_, err = e.PostP2PAd(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PostP2PAd must require a positive min amount")
	r.MinAmount = 20
	_, err = e.PostP2PAd(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PostP2PAd must require a positive max amount")
	r.MaxAmount = 45000
	_, err = e.PostP2PAd(t.Context(), r)
	require.ErrorIs(t, err, errPaymentIDsRequired, "PostP2PAd must require a payment method ID")
	r.PaymentIDs = []string{"7110"}
	_, err = e.PostP2PAd(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PostP2PAd must require a positive quantity")
	r.Quantity = 20000
	_, err = e.PostP2PAd(t.Context(), r)
	require.ErrorIs(t, err, errPaymentPeriodRequired, "PostP2PAd must require a payment period")
	r.PaymentPeriod = 15
	_, err = e.PostP2PAd(t.Context(), r)
	require.ErrorIs(t, err, errItemTypeRequired, "PostP2PAd must require an item type")
	r.ItemType = "ORIGIN"
	r.Price = 0.92
	r.Remark = "test"
	r.TradingPreferenceSet = &P2PTradingPreferenceSetRequest{
		IsKYC:                     true,
		OrderFinishNumberDay30:    60,
		CompleteRateDay30:         95,
		HasOrderFinishNumberDay30: true,
		HasCompleteRateDay30:      true,
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PostP2PAd(t.Context(), r)
	require.NoError(t, err, "PostP2PAd must not error")
	if mockTests {
		exp := &P2PPostAdResponse{
			ItemID:            "1899659847717838848",
			SecurityRiskToken: "risk-token-0001",
			RiskTokenType:     "challenge",
			RiskVersion:       "1",
			NeedSecurityRisk:  true,
		}
		assert.Equal(t, exp, result, "PostP2PAd should decode every field")
		return
	}
	assert.NotEmpty(t, result.ItemID, "PostP2PAd should return the advertisement ID")
}

func TestP2POptionalDecimal(t *testing.T) {
	t.Parallel()
	assert.Empty(t, p2pOptionalDecimal(0), "p2pOptionalDecimal should send zero as an empty string")
	assert.Equal(t, "101.5", p2pOptionalDecimal(101.5), "p2pOptionalDecimal should format a decimal")
}

func TestP2PTradingPreferenceSetRequestPayload(t *testing.T) {
	t.Parallel()
	var unset *P2PTradingPreferenceSetRequest
	assert.Equal(t, map[string]string{}, unset.payload(), "payload should send an empty object for a nil set")
	p := &P2PTradingPreferenceSetRequest{
		HasUnPostAd:               true,
		IsKYC:                     true,
		IsEmail:                   true,
		IsMobile:                  true,
		HasRegisterTime:           true,
		RegisterTimeThreshold:     15,
		OrderFinishNumberDay30:    60,
		CompleteRateDay30:         95.5,
		NationalLimit:             "MYS",
		HasOrderFinishNumberDay30: true,
		HasCompleteRateDay30:      true,
		HasNationalLimit:          true,
	}
	exp := map[string]string{
		"hasUnPostAd":               "1",
		"isKyc":                     "1",
		"isEmail":                   "1",
		"isMobile":                  "1",
		"hasRegisterTime":           "1",
		"registerTimeThreshold":     "15",
		"orderFinishNumberDay30":    "60",
		"completeRateDay30":         "95.5",
		"nationalLimit":             "MYS",
		"hasOrderFinishNumberDay30": "1",
		"hasCompleteRateDay30":      "1",
		"hasNationalLimit":          "1",
	}
	assert.Equal(t, exp, p.payload(), "payload should send every preference as a string")
	exp = map[string]string{
		"hasUnPostAd":               "0",
		"isKyc":                     "0",
		"isEmail":                   "0",
		"isMobile":                  "0",
		"hasRegisterTime":           "0",
		"registerTimeThreshold":     "0",
		"orderFinishNumberDay30":    "0",
		"completeRateDay30":         "0",
		"nationalLimit":             "",
		"hasOrderFinishNumberDay30": "0",
		"hasCompleteRateDay30":      "0",
		"hasNationalLimit":          "0",
	}
	assert.Equal(t, exp, new(P2PTradingPreferenceSetRequest).payload(), "payload should send unset preferences as not required")
}

func TestP2PFlag(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1", p2pFlag(true), "p2pFlag should send true as required")
	assert.Equal(t, "0", p2pFlag(false), "p2pFlag should send false as not required")
}

func TestRemoveP2PAd(t *testing.T) {
	t.Parallel()
	err := e.RemoveP2PAd(t.Context(), "")
	require.ErrorIs(t, err, errAdvertisementIDRequired, "RemoveP2PAd must require an advertisement ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.RemoveP2PAd(t.Context(), "1899667660027793408"), "RemoveP2PAd should not error")
}

func TestUpdateP2PAd(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateP2PAd(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateP2PAd must reject a nil request")
	r := &P2PUpdateAdRequest{}
	_, err = e.UpdateP2PAd(t.Context(), r)
	require.ErrorIs(t, err, errAdvertisementIDRequired, "UpdateP2PAd must require an advertisement ID")
	r.ID = p2pAdvertisementID
	_, err = e.UpdateP2PAd(t.Context(), r)
	require.ErrorIs(t, err, errPriceTypeRequired, "UpdateP2PAd must require a price type")
	r.PriceType = "1"
	_, err = e.UpdateP2PAd(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "UpdateP2PAd must require a positive min amount")
	r.MinAmount = 25
	_, err = e.UpdateP2PAd(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "UpdateP2PAd must require a positive max amount")
	r.MaxAmount = 5000
	_, err = e.UpdateP2PAd(t.Context(), r)
	require.ErrorIs(t, err, errPaymentIDsRequired, "UpdateP2PAd must require a payment method ID")
	r.PaymentIDs = []string{"7110"}
	_, err = e.UpdateP2PAd(t.Context(), r)
	require.ErrorIs(t, err, errActionTypeRequired, "UpdateP2PAd must require an action type")
	r.ActionType = "MODIFY"
	_, err = e.UpdateP2PAd(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "UpdateP2PAd must require a positive quantity")
	r.Quantity = 1000
	_, err = e.UpdateP2PAd(t.Context(), r)
	require.ErrorIs(t, err, errPaymentPeriodRequired, "UpdateP2PAd must require a payment period")
	r.PaymentPeriod = 15
	r.Premium = 101.5

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateP2PAd(t.Context(), r)
	require.NoError(t, err, "UpdateP2PAd must not error")
	if mockTests {
		exp := &P2PUpdateAdResponse{
			SecurityRiskToken: "risk-token-0002",
			RiskTokenType:     "challenge",
			RiskVersion:       "2",
			NeedSecurityRisk:  true,
		}
		assert.Equal(t, exp, result, "UpdateP2PAd should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateP2PAd should return a result")
}

func TestGetP2PMyAds(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PMyAds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2PMyAds must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PMyAds(t.Context(), &P2PMyAdsRequest{
		ItemID:     p2pAdvertisementID,
		Status:     "2",
		Side:       "1",
		TokenID:    currency.USDT,
		Page:       1,
		Size:       10,
		CurrencyID: currency.EUR,
	})
	require.NoError(t, err, "GetP2PMyAds must not error")
	if mockTests {
		exp := &P2PMyAdsResponse{
			Count: 1,
			Items: []P2PMyAd{
				{
					ID:               "1898988222063644672",
					AccountID:        "100000002",
					UserID:           "100000001",
					NickName:         "TestUser",
					TokenID:          currency.USDT,
					TokenName:        "USDT",
					CurrencyID:       currency.EUR,
					Side:             1,
					PriceType:        1,
					Price:            0.91,
					Premium:          101.5,
					LastQuantity:     4000,
					Quantity:         5000,
					FrozenQuantity:   600,
					ExecutedQuantity: 400,
					MinAmount:        10,
					MaxAmount:        5000,
					Remark:           "Bank transfer only",
					Status:           10,
					CreateDate:       types.Time(time.Unix(1741589049, 0)),
					Payments: []string{
						"14",
						"377",
					},
					OrderNumber:       8,
					FinishNumber:      7,
					RecentOrderNumber: 3,
					RecentExecuteRate: 87.5,
					Fee:               0.2,
					IsOnline:          true,
					LastLogoutTime:    types.Time(time.Unix(1741589000, 0)),
					SymbolInfo: P2PSymbolInfo{
						ID:                    "13",
						ExchangeID:            "301",
						OrgID:                 "9001",
						TokenID:               currency.USDT,
						CurrencyID:            currency.EUR,
						Status:                1,
						LowerLimitAlarm:       90,
						UpperLimitAlarm:       110,
						ItemDownRange:         70,
						ItemUpRange:           130,
						CurrencyMinQuote:      2,
						CurrencyMaxQuote:      46500,
						CurrencyLowerMaxQuote: 2,
						TokenMinQuote:         1,
						TokenMaxQuote:         50000,
						KYCCurrencyLimit:      900,
						ItemSideLimit:         3,
						BuyFeeRate:            0.001,
						SellFeeRate:           0.0015,
						OrderAutoCancelMinute: 15,
						OrderFinishMinute:     10,
						TradeSide:             9,
						Currency: P2PSymbolCurrency{
							ID:         "14",
							ExchangeID: "1",
							OrgID:      "9001",
							CurrencyID: currency.EUR,
							Scale:      3,
						},
						Token: P2PSymbolToken{
							ID:         "1",
							ExchangeID: "1",
							OrgID:      "9001",
							TokenID:    currency.USDT,
							Scale:      4,
							Sequence:   1,
						},
						BuyAd: P2PSymbolAdConfig{
							PaymentPeriods: []uint64{
								15,
								30,
							},
						},
						SellAd: P2PSymbolAdConfig{
							PaymentPeriods: []uint64{
								15,
								30,
								45,
							},
						},
					},
					TradingPreferenceSet: P2PTradingPreferenceSet{
						HasUnPostAd:               true,
						IsKYC:                     true,
						IsEmail:                   true,
						IsMobile:                  true,
						HasRegisterTime:           true,
						RegisterTimeThreshold:     15,
						OrderFinishNumberDay30:    60,
						CompleteRateDay30:         95,
						NationalLimit:             "MYS",
						HasOrderFinishNumberDay30: true,
						HasCompleteRateDay30:      true,
						HasNationalLimit:          true,
					},
					PaymentTerms: []P2PAdPaymentTerm{
						{
							ID:                     "7110",
							RealName:               "TEST USER",
							PaymentType:            14,
							BankName:               "Example Bank",
							BranchName:             "Example Branch",
							AccountNo:              "000123456789",
							QRCode:                 "https://example.com/qr/7110.png",
							Visible:                1,
							PayMessage:             "Pay within 15 minutes",
							FirstName:              "TEST",
							LastName:               "USER",
							SecondLastName:         "SAMPLE",
							Clabe:                  "000000000000000000",
							DebitCardNumber:        "0000000000000000",
							Mobile:                 "+0000000000",
							BusinessName:           "Example Business",
							Concept:                "P2P",
							PaymentExt1:            "ext1",
							PaymentExt2:            "ext2",
							PaymentExt3:            "ext3",
							PaymentExt4:            "ext4",
							PaymentExt5:            "ext5",
							PaymentExt6:            "ext6",
							PaymentTemplateVersion: 1,
							PaymentConfig: P2PAdPaymentConfig{
								PaymentType:         14,
								PaymentName:         "Bank Transfer",
								PaymentDialect:      "payment_field_bank_transfer",
								PaymentTemplateItem: json.RawMessage(`[]`),
							},
							RealNameVerified: true,
						},
						{
							ID:          "-1",
							PaymentType: 377,
							PaymentConfig: P2PAdPaymentConfig{
								PaymentType:         377,
								PaymentName:         "Balance",
								PaymentDialect:      "payment_field_balance",
								PaymentTemplateItem: json.RawMessage(`[]`),
							},
						},
					},
					Version:       2,
					UpdateDate:    types.Time(time.Unix(1741753781, 0)),
					FeeRate:       0.001,
					PaymentPeriod: 15,
					ItemType:      "ORIGIN",
					HiddenReason:  "Insufficient balance",
				},
			},
			HiddenFlag: true,
		}
		assert.Equal(t, exp, result, "GetP2PMyAds should decode every field")
		return
	}
	assert.NotNil(t, result.Items, "GetP2PMyAds should return ads")
}

func TestGetP2PMyAdDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PMyAdDetails(t.Context(), "")
	require.ErrorIs(t, err, errAdvertisementIDRequired, "GetP2PMyAdDetails must require an advertisement ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PMyAdDetails(t.Context(), p2pAdvertisementID)
	require.NoError(t, err, "GetP2PMyAdDetails must not error")
	if mockTests {
		exp := &P2PMyAdDetailsResponse{
			ID:               "1898988222063644672",
			AccountID:        "100000002",
			UserID:           "100000001",
			NickName:         "TestUser",
			TokenID:          currency.USDT,
			TokenName:        "USDT",
			CurrencyID:       currency.EUR,
			Side:             1,
			PriceType:        1,
			Price:            0.91,
			Premium:          101.5,
			LastQuantity:     4000,
			Quantity:         5000,
			FrozenQuantity:   600,
			ExecutedQuantity: 400,
			MinAmount:        10,
			MaxAmount:        5000,
			Remark:           "Bank transfer only",
			Status:           10,
			CreateDate:       types.Time(time.Unix(1741589049, 0)),
			Payments: []string{
				"14",
				"377",
			},
			OrderNumber:       8,
			FinishNumber:      7,
			RecentOrderNumber: 3,
			RecentExecuteRate: 87.5,
			Fee:               0.2,
			IsOnline:          true,
			LastLogoutTime:    types.Time(time.Unix(1741589000, 0)),
			SymbolInfo: P2PSymbolInfo{
				ID:                    "13",
				ExchangeID:            "301",
				OrgID:                 "9001",
				TokenID:               currency.USDT,
				CurrencyID:            currency.EUR,
				Status:                1,
				LowerLimitAlarm:       90,
				UpperLimitAlarm:       110,
				ItemDownRange:         70,
				ItemUpRange:           130,
				CurrencyMinQuote:      2,
				CurrencyMaxQuote:      46500,
				CurrencyLowerMaxQuote: 2,
				TokenMinQuote:         1,
				TokenMaxQuote:         50000,
				KYCCurrencyLimit:      900,
				ItemSideLimit:         3,
				BuyFeeRate:            0.001,
				SellFeeRate:           0.0015,
				OrderAutoCancelMinute: 15,
				OrderFinishMinute:     10,
				TradeSide:             9,
				Currency: P2PSymbolCurrency{
					ID:         "14",
					ExchangeID: "1",
					OrgID:      "9001",
					CurrencyID: currency.EUR,
					Scale:      3,
				},
				Token: P2PSymbolToken{
					ID:         "1",
					ExchangeID: "1",
					OrgID:      "9001",
					TokenID:    currency.USDT,
					Scale:      4,
					Sequence:   1,
				},
				BuyAd: P2PSymbolAdConfig{
					PaymentPeriods: []uint64{
						15,
						30,
					},
				},
				SellAd: P2PSymbolAdConfig{
					PaymentPeriods: []uint64{
						15,
						30,
						45,
					},
				},
			},
			TradingPreferenceSet: P2PTradingPreferenceSet{
				HasUnPostAd:               true,
				IsKYC:                     true,
				IsEmail:                   true,
				IsMobile:                  true,
				HasRegisterTime:           true,
				RegisterTimeThreshold:     15,
				OrderFinishNumberDay30:    60,
				CompleteRateDay30:         95,
				NationalLimit:             "MYS",
				HasOrderFinishNumberDay30: true,
				HasCompleteRateDay30:      true,
				HasNationalLimit:          true,
			},
			PaymentTerms: []P2PAdPaymentTerm{
				{
					ID:                     "7110",
					RealName:               "TEST USER",
					PaymentType:            14,
					BankName:               "Example Bank",
					BranchName:             "Example Branch",
					AccountNo:              "000123456789",
					QRCode:                 "https://example.com/qr/7110.png",
					Visible:                1,
					PayMessage:             "Pay within 15 minutes",
					FirstName:              "TEST",
					LastName:               "USER",
					SecondLastName:         "SAMPLE",
					Clabe:                  "000000000000000000",
					DebitCardNumber:        "0000000000000000",
					Mobile:                 "+0000000000",
					BusinessName:           "Example Business",
					Concept:                "P2P",
					PaymentExt1:            "ext1",
					PaymentExt2:            "ext2",
					PaymentExt3:            "ext3",
					PaymentExt4:            "ext4",
					PaymentExt5:            "ext5",
					PaymentExt6:            "ext6",
					PaymentTemplateVersion: 1,
					PaymentConfig: P2PAdPaymentConfig{
						PaymentType:         14,
						PaymentName:         "Bank Transfer",
						PaymentDialect:      "payment_field_bank_transfer",
						PaymentTemplateItem: json.RawMessage(`[]`),
					},
					RealNameVerified: true,
				},
				{
					ID:          "-1",
					PaymentType: 377,
					PaymentConfig: P2PAdPaymentConfig{
						PaymentType:         377,
						PaymentName:         "Balance",
						PaymentDialect:      "payment_field_balance",
						PaymentTemplateItem: json.RawMessage(`[]`),
					},
				},
			},
			Version:       2,
			UpdateDate:    types.Time(time.Unix(1741753781, 0)),
			FeeRate:       0.001,
			PaymentPeriod: 15,
			ItemType:      "ORIGIN",
		}
		assert.Equal(t, exp, result, "GetP2PMyAdDetails should decode every field")
		return
	}
	assert.NotEmpty(t, result.ID, "GetP2PMyAdDetails should return the advertisement ID")
}

func TestGetAllP2POrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllP2POrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllP2POrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllP2POrders(t.Context(), &P2POrderListRequest{
		Page:      1,
		Size:      10,
		BeginTime: time.UnixMilli(1741651200000),
		EndTime:   time.UnixMilli(1741910400000),
		TokenID:   currency.USDT,
		Side:      "1",
	})
	require.NoError(t, err, "GetAllP2POrders must not error")
	if mockTests {
		exp := &P2PAllOrdersResponse{
			Count: 2,
			Items: []P2POrder{
				{
					ID:                     "1899742990873296896",
					Side:                   1,
					TokenID:                currency.USDT,
					OrderType:              "ORIGIN",
					Amount:                 46,
					CurrencyID:             currency.EUR,
					Price:                  0.92,
					NotifyTokenQuantity:    50,
					NotifyTokenID:          currency.USDT,
					Fee:                    0.05,
					TargetNickName:         "TestMaker",
					TargetUserID:           "100000003",
					Status:                 30,
					SelfUnreadMessageCount: 1,
					CreateDate:             types.Time(time.Unix(1741831397, 0)),
					AppealLastSeconds:      3600,
					UserID:                 "100000001",
					SellerRealName:         "TEST SELLER",
					BuyerRealName:          "TEST BUYER",
					JudgeInfo: P2PJudgeInfo{
						AutoJudgeUnlockTime: types.Time(time.Unix(1741917797, 0)),
						DissentResult:       "PASS",
						PreDissent:          "BUYER",
						PostDissent:         "SELLER",
					},
					UnreadMessageCount: 1,
					Extension: P2POrderExtension{
						IsDelayWithdraw: true,
						DelayTime:       86400,
						StartTime:       types.Time(time.Unix(1741831397, 0)),
					},
					BulkOrderFlag: true,
				},
				{
					ID:                  "1899742990873296897",
					Side:                1,
					TokenID:             currency.USDT,
					OrderType:           "ORIGIN",
					Amount:              46,
					CurrencyID:          currency.EUR,
					Price:               0.92,
					NotifyTokenQuantity: 50,
					NotifyTokenID:       currency.USDT,
					Fee:                 0.05,
					TargetNickName:      "TestMaker",
					TargetUserID:        "100000003",
					Status:              10,
					CreateDate:          types.Time(time.Unix(1741831397, 0)),
					TransferLastSeconds: 871,
					UserID:              "100000001",
					SellerRealName:      "TEST SELLER",
					BuyerRealName:       "TEST BUYER",
				},
			},
		}
		assert.Equal(t, exp, result, "GetAllP2POrders should decode every field")
		return
	}
	assert.NotNil(t, result.Items, "GetAllP2POrders should return orders")
}

func TestP2POrderListBody(t *testing.T) {
	t.Parallel()
	_, err := p2pOrderListBody(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "p2pOrderListBody must reject a nil request")
	r := &P2POrderListRequest{}
	_, err = p2pOrderListBody(r)
	require.ErrorIs(t, err, errPageIndexRequired, "p2pOrderListBody must require a page")
	r.Page = 2
	_, err = p2pOrderListBody(r)
	require.ErrorIs(t, err, errPageLimitRequired, "p2pOrderListBody must require a page size")
	r.Size = 30
	r.BeginTime, r.EndTime = time.UnixMilli(1741910400000), time.UnixMilli(1741651200000)
	_, err = p2pOrderListBody(r)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "p2pOrderListBody must reject a reversed window")
	r.BeginTime, r.EndTime = r.EndTime, r.BeginTime
	r.Side = "Buy"
	_, err = p2pOrderListBody(r)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "p2pOrderListBody must reject a side that is not a number")
	r.Side = "0"
	r.Status = 20
	r.TokenID = currency.NewCode("usdt")

	body, err := p2pOrderListBody(r)
	require.NoError(t, err, "p2pOrderListBody must not error")
	exp := map[string]any{
		"page":      uint64(2),
		"size":      uint64(30),
		"status":    uint64(20),
		"beginTime": "1741651200000",
		"endTime":   "1741910400000",
		"tokenId":   "USDT",
		"side":      uint64(0),
	}
	assert.Equal(t, exp, body, "p2pOrderListBody should send side as an integer and times as millisecond strings")
}

func TestGetP2POrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2POrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetP2POrderDetail must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2POrderDetail(t.Context(), p2pOrderID)
	require.NoError(t, err, "GetP2POrderDetail must not error")
	if mockTests {
		exp := &P2POrderDetailResponse{
			ID:                       "1900004704665923584",
			Side:                     1,
			ItemID:                   "1899659847717838848",
			AccountID:                "100000002",
			UserID:                   "100000001",
			NickName:                 "TestUser",
			MakerUserID:              "100000003",
			TargetAccountID:          "100000004",
			TargetUserID:             "100000003",
			TargetNickName:           "TestMaker",
			TargetUserMaskID:         "s00000000000000000000000000000001",
			TargetFirstName:          "TEST",
			TargetSecondName:         "MAKER",
			TargetUserAuthStatus:     2,
			TargetConnectInformation: "test@example.com",
			PayerRealName:            "TEST BUYER",
			SellerRealName:           "TEST SELLER",
			BuyerRealName:            "TEST BUYER",
			TokenID:                  currency.USDT,
			TokenName:                "USDT",
			CurrencyID:               currency.EUR,
			Price:                    0.92,
			Quantity:                 50,
			Amount:                   46,
			PayCode:                  "826102",
			PaymentType:              62,
			TransferDate:             types.Time(time.Unix(1741831497, 0)),
			Status:                   20,
			CreateDate:               types.Time(time.Unix(1741831397, 0)),
			PaymentTermList: []P2POrderPaymentTerm{
				{
					ID:                     "7110",
					RealName:               "TEST USER",
					PaymentType:            14,
					BankName:               "Example Bank",
					BranchName:             "Example Branch",
					AccountNo:              "000123456789",
					QRCode:                 "https://example.com/qr/7110.png",
					Visible:                1,
					PayMessage:             "Pay within 15 minutes",
					FirstName:              "TEST",
					LastName:               "USER",
					SecondLastName:         "SAMPLE",
					Clabe:                  "000000000000000000",
					DebitCardNumber:        "0000000000000000",
					Mobile:                 "+0000000000",
					BusinessName:           "Example Business",
					Concept:                "P2P",
					Online:                 "0",
					PaymentExt1:            "ext1",
					PaymentExt2:            "ext2",
					PaymentExt3:            "ext3",
					PaymentExt4:            "ext4",
					PaymentExt5:            "ext5",
					PaymentExt6:            "ext6",
					PaymentTemplateVersion: 1,
					PaymentConfigVO: P2PPaymentConfigVO{
						PaymentType: 14,
						CheckType:   7,
						Sort:        1,
						PaymentName: "Bank Transfer",
						AddTips:     "Add your bank account",
						ItemTips:    "Use your own account",
						Items: []P2PPaymentConfigItem{
							{
								View:        true,
								Name:        "realName",
								Label:       "Name",
								Placeholder: "Please Enter Name",
								Type:        "text",
								MaxLength:   50,
								Required:    true,
							},
							{
								Name:        "branchName",
								Label:       "Bank Branch",
								Placeholder: "Please Enter Bank Branch",
								Type:        "text",
								MaxLength:   100,
							},
						},
					},
					RUPaymentPrompt: true,
				},
				{
					ID:          "-1",
					PaymentType: 377,
					Online:      "1",
					PaymentConfigVO: P2PPaymentConfigVO{
						PaymentType: 377,
						PaymentName: "Balance",
						Online:      1,
						Items:       []P2PPaymentConfigItem{},
					},
				},
			},
			Remark:              "test",
			TransferLastSeconds: 201,
			RecentOrderNumber:   4,
			RecentExecuteRate:   75,
			AppealLastSeconds:   1800,
			AppealContent:       "Payment not received",
			AppealType:          2,
			AppealNickName:      "TestMaker",
			CanAppeal:           "Y",
			TotalAppealTimes:    4,
			AppealedTimes:       1,
			PaymentTermResult: P2POrderPaymentTerm{
				ID:                     "8120",
				RealName:               "TEST USER",
				PaymentType:            62,
				BankName:               "Example Wallet",
				BranchName:             "Example Branch",
				AccountNo:              "000123456789",
				QRCode:                 "https://example.com/qr/7110.png",
				Visible:                1,
				PayMessage:             "Pay within 15 minutes",
				FirstName:              "TEST",
				LastName:               "USER",
				SecondLastName:         "SAMPLE",
				Clabe:                  "000000000000000000",
				DebitCardNumber:        "0000000000000000",
				Mobile:                 "+0000000000",
				BusinessName:           "Example Business",
				Concept:                "P2P",
				Online:                 "1",
				PaymentExt1:            "ext1",
				PaymentExt2:            "ext2",
				PaymentExt3:            "ext3",
				PaymentExt4:            "ext4",
				PaymentExt5:            "ext5",
				PaymentExt6:            "ext6",
				PaymentTemplateVersion: 1,
				PaymentConfigVO: P2PPaymentConfigVO{
					PaymentType: 62,
					CheckType:   7,
					Sort:        1,
					PaymentName: "E-Wallet",
					AddTips:     "Add your wallet",
					ItemTips:    "Use your own wallet",
					Online:      1,
					Items: []P2PPaymentConfigItem{
						{
							View:        true,
							Name:        "realName",
							Label:       "Name",
							Placeholder: "Please Enter Name",
							Type:        "text",
							MaxLength:   50,
							Required:    true,
						},
						{
							Name:        "branchName",
							Label:       "Bank Branch",
							Placeholder: "Please Enter Bank Branch",
							Type:        "text",
							MaxLength:   100,
						},
					},
				},
				RUPaymentPrompt: true,
			},
			OrderFinishMinute: 600000,
			ConfirmedPayTerm: P2POrderPaymentTerm{
				ID:                     "8120",
				RealName:               "TEST USER",
				PaymentType:            62,
				BankName:               "Example Wallet",
				BranchName:             "Example Branch",
				AccountNo:              "000123456789",
				QRCode:                 "https://example.com/qr/7110.png",
				Visible:                1,
				PayMessage:             "Pay within 15 minutes",
				FirstName:              "TEST",
				LastName:               "USER",
				SecondLastName:         "SAMPLE",
				Clabe:                  "000000000000000000",
				DebitCardNumber:        "0000000000000000",
				Mobile:                 "+0000000000",
				BusinessName:           "Example Business",
				Concept:                "P2P",
				Online:                 "1",
				PaymentExt1:            "ext1",
				PaymentExt2:            "ext2",
				PaymentExt3:            "ext3",
				PaymentExt4:            "ext4",
				PaymentExt5:            "ext5",
				PaymentExt6:            "ext6",
				PaymentTemplateVersion: 1,
				PaymentConfigVO: P2PPaymentConfigVO{
					PaymentType: 62,
					CheckType:   7,
					Sort:        1,
					PaymentName: "E-Wallet",
					AddTips:     "Add your wallet",
					ItemTips:    "Use your own wallet",
					Online:      1,
					Items: []P2PPaymentConfigItem{
						{
							View:        true,
							Name:        "realName",
							Label:       "Name",
							Placeholder: "Please Enter Name",
							Type:        "text",
							MaxLength:   50,
							Required:    true,
						},
						{
							Name:        "branchName",
							Label:       "Bank Branch",
							Placeholder: "Please Enter Bank Branch",
							Type:        "text",
							MaxLength:   100,
						},
					},
				},
				RUPaymentPrompt: true,
			},
			MakerFee:           0.05,
			TakerFee:           0.02,
			Fee:                0.05,
			ShowContact:        true,
			ContactInfo:        json.RawMessage(`[]`),
			TokenBalance:       74241.51120610707,
			FiatBalance:        4657.78,
			UnreadMessageCount: 2,
			UpdateDate:         types.Time(time.Unix(1741831497, 0)),
			Extension: P2POrderExtension{
				IsDelayWithdraw: true,
				DelayTime:       86400,
				StartTime:       types.Time(time.Unix(1741831397, 0)),
			},
			SelfUnreadMessageCount: 1,
			JudgeType:              "AUTO",
			CanReport:              true,
			CanReportDisagree:      true,
			CanReportType: []string{
				"REPORT_SCAM",
			},
			CanReportDisagreeType: []string{
				"REPORT_DISAGREE",
			},
			AppraiseStatus: "CAN_APPRAISE",
			AppraiseInfo: P2PAppraiseInfo{
				Anonymous:       "Y",
				AppraiseContent: "Fast and friendly",
				AppraiseID:      "5001",
				AppraiseType:    "GOOD",
				ModifyFlag:      "Y",
				UpdateDate:      types.Time(time.Unix(1741832000, 0)),
			},
			CanReportDisagreeTypes: []string{
				"REPORT_DISAGREE",
			},
			CanReportTypes: []string{
				"REPORT_SCAM",
			},
			OrderType:                      "ORIGIN",
			MiddleToken:                    currency.USDT,
			BeforePrice:                    0.91,
			BeforeQuantity:                 49,
			BeforeToken:                    currency.USDC,
			Alternative:                    "USDC",
			AppealUserID:                   "100000003",
			NotifyTokenID:                  currency.USDT,
			NotifyTokenQuantity:            50,
			CancelResponsible:              "BUYER",
			ChainType:                      "TRX",
			ChainAddress:                   "TExampleAddress000000000000000000000",
			TradeHashCode:                  "0xexamplehash",
			EstimatedGasFee:                1,
			GasFeeTokenID:                  currency.TRX,
			TradingFeeTokenID:              currency.USDT,
			OnChainInfo:                    "confirmed",
			TransactionID:                  "3000835348",
			DisplayRefund:                  1,
			ChainWithdrawLastSeconds:       201,
			ChainTransferLastSeconds:       201,
			OrderSource:                    "P2P",
			CancelReason:                   "Buyer requested",
			NeedBuyerExamineCancel:         true,
			SellerCancelExamineRemainTime:  600,
			NeedSellerExamineCancel:        true,
			CouponCurrencyAmount:           1,
			TotalCurrencyAmount:            46,
			UsedCoupon:                     true,
			CouponTokenID:                  currency.USDT,
			CouponQuantity:                 1,
			CompletedOrderAppealCount:      1,
			TotalCompletedOrderAppealCount: 10,
			RealOrderStatus:                20,
			AppealVersion:                  1,
			JudgeInfo: P2PJudgeInfo{
				AutoJudgeUnlockTime: types.Time(time.Unix(1741917797, 0)),
				DissentResult:       "PASS",
				PreDissent:          "BUYER",
				PostDissent:         "SELLER",
			},
			HelpType:         "NEVER",
			AppealFlowStatus: "PROCESSING",
			AppealSubStatus:  "WAIT_EVIDENCE",
			BulkOrderFlag:    true,
			TargetUserType:   "ORG",
			TargetUserDisplays: []string{
				"DISPLAY_MAKER_BULK",
			},
			AppealProcessChangeFlag: true,
			AppealNegotiationNode:   1,
		}
		assert.Equal(t, exp, result, "GetP2POrderDetail should decode every field")
		return
	}
	assert.NotEmpty(t, result.ID, "GetP2POrderDetail should return the order ID")
}

func TestGetP2PPendingOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PPendingOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2PPendingOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PPendingOrders(t.Context(), &P2POrderListRequest{Page: 1, Size: 10, Status: 10})
	require.NoError(t, err, "GetP2PPendingOrders must not error")
	if mockTests {
		exp := &P2PPendingOrdersResponse{
			Count: 2,
			Items: []P2PPendingOrder{
				{
					ID:                     "1900004704665923584",
					Side:                   1,
					TokenID:                currency.USDT,
					OrderType:              "ORIGIN",
					Amount:                 46,
					CurrencyID:             currency.EUR,
					Price:                  0.92,
					NotifyTokenQuantity:    50,
					NotifyTokenID:          currency.USDT,
					Fee:                    0.05,
					TargetNickName:         "TestMaker",
					TargetUserID:           "100000003",
					TargetUserMaskID:       "s00000000000000000000000000000001",
					Status:                 10,
					SelfUnreadMessageCount: 2,
					CreateDate:             types.Time(time.Unix(1741831397, 0)),
					TransferLastSeconds:    871,
					UserID:                 "100000001",
					SellerRealName:         "TEST SELLER",
					BuyerRealName:          "TEST BUYER",
					UnreadMessageCount:     2,
				},
				{
					ID:                  "1900004704665923585",
					TokenID:             currency.USDT,
					OrderType:           "ORIGIN",
					Amount:              46,
					CurrencyID:          currency.EUR,
					Price:               0.92,
					NotifyTokenQuantity: 50,
					NotifyTokenID:       currency.USDT,
					Fee:                 0.05,
					TargetNickName:      "TestMaker",
					TargetUserID:        "100000003",
					TargetUserMaskID:    "s00000000000000000000000000000002",
					Status:              10,
					CreateDate:          types.Time(time.Unix(1741831397, 0)),
					TransferLastSeconds: 600,
					AppealLastSeconds:   1800,
					UserID:              "100000001",
					SellerRealName:      "TEST SELLER",
					BuyerRealName:       "TEST BUYER",
					JudgeInfo: P2PJudgeInfo{
						AutoJudgeUnlockTime: types.Time(time.Unix(1741917797, 0)),
						DissentResult:       "PASS",
						PreDissent:          "BUYER",
						PostDissent:         "SELLER",
					},
					Extension: P2POrderExtension{
						IsDelayWithdraw: true,
						DelayTime:       86400,
						StartTime:       types.Time(time.Unix(1741831397, 0)),
					},
					BulkOrderFlag: true,
				},
			},
		}
		assert.Equal(t, exp, result, "GetP2PPendingOrders should decode every field")
		return
	}
	assert.NotNil(t, result.Items, "GetP2PPendingOrders should return orders")
}

func TestMarkP2POrderAsPaid(t *testing.T) {
	t.Parallel()
	err := e.MarkP2POrderAsPaid(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "MarkP2POrderAsPaid must reject a nil request")
	r := &P2PMarkOrderAsPaidRequest{}
	err = e.MarkP2POrderAsPaid(t.Context(), r)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "MarkP2POrderAsPaid must require an order ID")
	r.OrderID = p2pOrderID
	err = e.MarkP2POrderAsPaid(t.Context(), r)
	require.ErrorIs(t, err, errPaymentTypeRequired, "MarkP2POrderAsPaid must require a payment type")
	r.PaymentType = "14"
	err = e.MarkP2POrderAsPaid(t.Context(), r)
	require.ErrorIs(t, err, errPaymentIDRequired, "MarkP2POrderAsPaid must require a payment method ID")
	r.PaymentID = "7110"

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.MarkP2POrderAsPaid(t.Context(), r), "MarkP2POrderAsPaid should not error")
}

func TestReleaseP2PAssets(t *testing.T) {
	t.Parallel()
	err := e.ReleaseP2PAssets(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ReleaseP2PAssets must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.ReleaseP2PAssets(t.Context(), p2pOrderID), "ReleaseP2PAssets should not error")
}

func TestReviewP2PSellerCancelOrderApply(t *testing.T) {
	t.Parallel()
	err := e.ReviewP2PSellerCancelOrderApply(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ReviewP2PSellerCancelOrderApply must reject a nil request")
	r := &P2PReviewSellerCancelOrderApplyRequest{}
	err = e.ReviewP2PSellerCancelOrderApply(t.Context(), r)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ReviewP2PSellerCancelOrderApply must require an order ID")
	r.OrderID = p2pOrderID
	err = e.ReviewP2PSellerCancelOrderApply(t.Context(), r)
	require.ErrorIs(t, err, errExamineResultRequired, "ReviewP2PSellerCancelOrderApply must require an examine result")
	r.ExamineResult = "REJECT"
	err = e.ReviewP2PSellerCancelOrderApply(t.Context(), r)
	require.ErrorIs(t, err, errRejectReasonRequired, "ReviewP2PSellerCancelOrderApply must require a reason to reject")
	r.RejectReason = "buyerRefuseOrderCancelReason_haveMadePayment"
	err = e.ReviewP2PSellerCancelOrderApply(t.Context(), r)
	require.ErrorIs(t, err, errRejectProofsRequired, "ReviewP2PSellerCancelOrderApply must require proofs to reject")
	r.RejectProofs = []string{"https://example.com/proof1.jpg", "https://example.com/proof2.jpg"}
	r.RejectRemark = "I have made the payment."

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.ReviewP2PSellerCancelOrderApply(t.Context(), r), "ReviewP2PSellerCancelOrderApply should not error")
}

func TestGetP2PAccountInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PAccountInformation(t.Context())
	require.NoError(t, err, "GetP2PAccountInformation must not error")
	if mockTests {
		exp := &P2PUserInfoResponse{
			NickName:             "TestUser",
			DefaultNickName:      true,
			WhiteFlag:            1,
			ContactConfig:        true,
			IsOnline:             true,
			Email:                "t***@example.com",
			Mobile:               "+00****0000",
			KYCLevel:             2,
			LastLogoutTime:       types.Time(time.Unix(1741246087, 0)),
			RecentRate:           98.5,
			TotalFinishCount:     120,
			TotalFinishSellCount: 70,
			TotalFinishBuyCount:  50,
			RecentFinishCount:    12,
			AverageReleaseTime:   2.5,
			AverageTransferTime:  4.1,
			AccountCreateDays:    698,
			FirstTradeDays:       360,
			RealName:             "TEST USER",
			RecentTradeAmount:    12500.5,
			TotalTradeAmount:     250000,
			ExecuteNumber:        118,
			OrderNumber:          120,
			HasUnPostAd:          true,
			RegisterTime:         types.Time(time.Unix(1681266699, 0)),
			AuthStatus:           1,
			KYCCountryCode:       "MYS",
			Blocked:              "N",
			GoodAppraiseRate:     99.1,
			GoodAppraiseCount:    110,
			BadAppraiseCount:     1,
			AccountID:            preciseNumber(t, "100000002"),
			PaymentCount:         2,
			ContactCount:         1,
			RealNameMask:         "TEST ***R",
			VIPLevel:             1,
			VIPProfit: []string{
				"P2P_AUTO_JUDGE",
			},
			UserTag: []string{
				"RU_PAYMENT_PROMPT",
			},
			UserCancelCountLimit:      1000,
			PaymentRealNameUneditable: true,
			LostRoleAffected:          true,
			UserCurrentPrivilege: []string{
				"OPEN_API_ACCESS",
				"DISPLAY_MAKER_BULK",
				"POST_EDIT_AD",
				"CREATE_ORDER_BULK",
				"POST_AD_BULK",
				"POST_VERIFICATION_ORDER_AD",
				"CAN_APPLY_OPEN_API",
			},
			UserType:        "PERSONAL",
			UserID:          "100000001",
			RealNameEnglish: "TEST USER",
			CanSubOnline:    true,
			CurrentPrivilegeInfo: []P2PPrivilegeInfo{
				{
					Name: "OPEN_API_ACCESS",
					Data: "NONE",
				},
				{
					Name: "POST_EDIT_AD",
					Data: "1",
				},
			},
			OpenAPISwitch: 1,
		}
		assert.Equal(t, exp, result, "GetP2PAccountInformation should decode every field")
		return
	}
	assert.NotEmpty(t, result.UserID, "GetP2PAccountInformation should return the user ID")
}

func TestGetP2PCounterpartyUserInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PCounterpartyUserInfo(t.Context(), "", p2pOrderID)
	require.ErrorIs(t, err, errOriginalUIDRequired, "GetP2PCounterpartyUserInfo must require a counterparty UID")
	_, err = e.GetP2PCounterpartyUserInfo(t.Context(), p2pMakerUID, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetP2PCounterpartyUserInfo must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PCounterpartyUserInfo(t.Context(), p2pMakerUID, p2pOrderID)
	require.NoError(t, err, "GetP2PCounterpartyUserInfo must not error")
	if mockTests {
		exp := &P2PUserInfoResponse{
			NickName:             "TestMaker",
			DefaultNickName:      true,
			WhiteFlag:            1,
			ContactConfig:        true,
			IsOnline:             true,
			Email:                "m***@example.com",
			Mobile:               "+00****0000",
			KYCLevel:             2,
			LastLogoutTime:       types.Time(time.Unix(1741246087, 0)),
			RecentRate:           98.5,
			TotalFinishCount:     120,
			TotalFinishSellCount: 70,
			TotalFinishBuyCount:  50,
			RecentFinishCount:    12,
			AverageReleaseTime:   2.5,
			AverageTransferTime:  4.1,
			AccountCreateDays:    698,
			FirstTradeDays:       360,
			RealName:             "TEST USER",
			RecentTradeAmount:    12500.5,
			TotalTradeAmount:     250000,
			ExecuteNumber:        118,
			OrderNumber:          120,
			HasUnPostAd:          true,
			RegisterTime:         types.Time(time.Unix(1681266699, 0)),
			AuthStatus:           1,
			KYCCountryCode:       "MYS",
			Blocked:              "N",
			GoodAppraiseRate:     99.1,
			GoodAppraiseCount:    110,
			BadAppraiseCount:     1,
			AccountID:            preciseNumber(t, "100000004"),
			PaymentCount:         2,
			ContactCount:         1,
			RealNameMask:         "TEST ***R",
			VIPLevel:             1,
			VIPProfit: []string{
				"P2P_AUTO_JUDGE",
			},
			UserTag: []string{
				"RU_PAYMENT_PROMPT",
			},
			UserCancelCountLimit:      1000,
			PaymentRealNameUneditable: true,
			LostRoleAffected:          true,
			UserCurrentPrivilege: []string{
				"OPEN_API_ACCESS",
				"DISPLAY_MAKER_BULK",
				"POST_EDIT_AD",
				"CREATE_ORDER_BULK",
				"POST_AD_BULK",
				"POST_VERIFICATION_ORDER_AD",
				"CAN_APPLY_OPEN_API",
			},
			UserType:        "PERSONAL",
			UserID:          "100000003",
			RealNameEnglish: "TEST USER",
			CanSubOnline:    true,
			CurrentPrivilegeInfo: []P2PPrivilegeInfo{
				{
					Name: "OPEN_API_ACCESS",
					Data: "NONE",
				},
				{
					Name: "POST_EDIT_AD",
					Data: "1",
				},
			},
			OpenAPISwitch: 1,
		}
		assert.Equal(t, exp, result, "GetP2PCounterpartyUserInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.NickName, "GetP2PCounterpartyUserInfo should return the nickname")
}

func TestGetP2PUserPayment(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PUserPayment(t.Context())
	require.NoError(t, err, "GetP2PUserPayment must not error")
	if mockTests {
		exp := []P2PUserPaymentResponse{
			{
				ID:          "-1",
				PaymentType: 377,
				Online:      "1",
				PaymentConfigVO: P2PPaymentConfigVO{
					PaymentType: 377,
					PaymentName: "Balance",
					Online:      1,
					Items:       []P2PPaymentConfigItem{},
				},
				Channel: "bybit",
				CurrencyBalance: []currency.Code{
					currency.NewCode("ARS"),
					currency.EUR,
					currency.USD,
				},
			},
			{
				ID:                        "7110",
				RealName:                  "TEST USER",
				PaymentType:               14,
				BankName:                  "Example Bank",
				BranchName:                "Example Branch",
				AccountNo:                 "000123456789",
				QRCode:                    "https://example.com/qr/7110.png",
				Visible:                   1,
				PayMessage:                "Pay within 15 minutes",
				FirstName:                 "TEST",
				LastName:                  "USER",
				SecondLastName:            "SAMPLE",
				Clabe:                     "000000000000000000",
				DebitCardNumber:           "0000000000000000",
				Mobile:                    "+0000000000",
				BusinessName:              "Example Business",
				Concept:                   "P2P",
				Online:                    "0",
				CountNo:                   "0001",
				PaymentExt1:               "ext1",
				PaymentExt2:               "ext2",
				PaymentExt3:               "ext3",
				PaymentExt4:               "ext4",
				PaymentExt5:               "ext5",
				PaymentExt6:               "ext6",
				PaymentTemplateVersion:    1,
				HasPaymentTemplateChanged: true,
				PaymentConfigVO: P2PPaymentConfigVO{
					PaymentType: 14,
					CheckType:   7,
					Sort:        1,
					PaymentName: "Bank Transfer",
					AddTips:     "Add your bank account",
					ItemTips:    "Use your own account",
					Items: []P2PPaymentConfigItem{
						{
							View:        true,
							Name:        "realName",
							Label:       "Name",
							Placeholder: "Please Enter Name",
							Type:        "text",
							MaxLength:   50,
							Required:    true,
						},
						{
							Name:        "branchName",
							Label:       "Bank Branch",
							Placeholder: "Please Enter Bank Branch",
							Type:        "text",
							MaxLength:   100,
						},
					},
				},
				RealNameVerified: true,
				Channel:          "bybit",
				CurrencyBalance:  []currency.Code{},
			},
		}
		assert.Equal(t, exp, result, "GetP2PUserPayment should decode every field")
		return
	}
	assert.NotNil(t, result, "GetP2PUserPayment should return payment methods")
}

func TestGetP2PChatSessionList(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PChatSessionList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2PChatSessionList must reject a nil request")
	_, err = e.GetP2PChatSessionList(t.Context(), &P2PChatSessionListRequest{})
	require.ErrorIs(t, err, errPageLimitRequired, "GetP2PChatSessionList must require a page size")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PChatSessionList(t.Context(), &P2PChatSessionListRequest{Size: 10, ReadStatus: 2, Type: "SINGLE"})
	require.NoError(t, err, "GetP2PChatSessionList must not error")
	if mockTests {
		exp := &P2PChatSessionListResponse{
			ChatSession: []P2PChatSession{
				{
					SessionName: "TestMaker",
					SessionID:   "aes_encrypted_session_id",
					Type:        "SINGLE",
					ID:          preciseNumber(t, "128"),
					UnreadCount: 5,
				},
				{
					SessionName: "AnotherMaker",
					SessionID:   "aes_encrypted_session_id_2",
					Type:        "SINGLE",
					ID:          preciseNumber(t, "96"),
				},
			},
		}
		assert.Equal(t, exp, result, "GetP2PChatSessionList should decode every field")
		return
	}
	assert.NotNil(t, result.ChatSession, "GetP2PChatSessionList should return sessions")
}

func TestGetP2PChatSessionID(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PChatSessionID(t.Context(), "")
	require.ErrorIs(t, err, errUserMaskIDRequired, "GetP2PChatSessionID must require a user mask ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PChatSessionID(t.Context(), p2pUserMaskID)
	require.NoError(t, err, "GetP2PChatSessionID must not error")
	if mockTests {
		exp := &P2PChatSessionIDResponse{
			SessionID: "aes_encrypted_session_id",
		}
		assert.Equal(t, exp, result, "GetP2PChatSessionID should decode every field")
		return
	}
	assert.NotEmpty(t, result.SessionID, "GetP2PChatSessionID should return a session ID")
}

// p2pTestPNG is the start of a PNG file, enough for its content type to be detected
var p2pTestPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")

func TestUploadP2PChatFile(t *testing.T) {
	t.Parallel()
	_, err := e.UploadP2PChatFile(t.Context(), "", p2pTestPNG)
	require.ErrorIs(t, err, errUploadFileNameRequired, "UploadP2PChatFile must require a file name")
	_, err = e.UploadP2PChatFile(t.Context(), "proof.png", nil)
	require.ErrorIs(t, err, errUploadFileEmpty, "UploadP2PChatFile must reject an empty file")
	unreachable := newTransportTestExchange(t, func(http.ResponseWriter, *http.Request) {
		assert.Fail(t, "UploadP2PChatFile should not send a request it cannot sign or address")
	})
	unreachable.SetCredentials(&accounts.Credentials{})
	_, err = unreachable.UploadP2PChatFile(t.Context(), "proof.png", p2pTestPNG)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "UploadP2PChatFile must require credentials")
	unreachable.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	unreachable.API.Endpoints = unreachable.NewEndpoints()
	_, err = unreachable.UploadP2PChatFile(t.Context(), "proof.png", p2pTestPNG)
	require.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "UploadP2PChatFile must require a REST URL")

	ex := e
	if mockTests {
		// The recorded-response mock server cannot match a multipart body, so a local handler checks the request
		// and answers with the documented response
		ex = newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "the upload should be a POST")
			assert.Equal(t, "/v5/p2p/oss/upload_file", r.URL.Path, "the upload should reach the upload path")
			assert.Empty(t, r.URL.RawQuery, "the upload should not carry a query string")
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the upload should not error")
			mac := hmac.New(sha256.New, []byte("secret"))
			_, err = mac.Write(append([]byte(r.Header.Get("X-BAPI-TIMESTAMP")+"key"+defaultRecvWindow), body...))
			assert.NoError(t, err, "hashing the upload should not error")
			assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), r.Header.Get("X-BAPI-SIGN"), "the upload should sign its multipart body")
			assert.Equal(t, "key", r.Header.Get("X-BAPI-API-KEY"), "the upload should carry the API key")
			assert.Equal(t, defaultRecvWindow, r.Header.Get("X-BAPI-RECV-WINDOW"), "the upload should carry the receive window")
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			assert.NoError(t, err, "the upload content type should parse")
			assert.Equal(t, "multipart/form-data", mediaType, "the upload should be a multipart form")
			part, err := multipart.NewReader(bytes.NewReader(body), params["boundary"]).NextPart()
			if assert.NoError(t, err, "the upload should hold a part") {
				assert.Equal(t, "upload_file", part.FormName(), "the upload should send the file as upload_file")
				assert.Equal(t, "proof.png", part.FileName(), "the upload should send the file name")
			}
			_, err = fmt.Fprint(w, `{"result":{"uploadId":null,"type":"IMAGE","url":"/fiat/p2p/oss/showObj/otc/9001/proof.png?e=1742570854&token=placeholder"},"token":null,"ret_code":0,"ret_msg":"成功"}`)
			assert.NoError(t, err, "writing the response should not error")
		})
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := ex.UploadP2PChatFile(t.Context(), "proof.png", p2pTestPNG)
	require.NoError(t, err, "UploadP2PChatFile must not error")
	if mockTests {
		exp := &P2PChatFileResponse{
			Type: "IMAGE",
			URL:  "/fiat/p2p/oss/showObj/otc/9001/proof.png?e=1742570854&token=placeholder",
		}
		assert.Equal(t, exp, result, "UploadP2PChatFile should decode every field")
		return
	}
	assert.NotEmpty(t, result.URL, "UploadP2PChatFile should return the file URL")
}

func TestP2PChatFileBody(t *testing.T) {
	t.Parallel()
	_, _, err := p2pChatFileBody("", p2pTestPNG)
	require.ErrorIs(t, err, errUploadFileNameRequired, "p2pChatFileBody must require a file name")
	_, _, err = p2pChatFileBody("proof.png", nil)
	require.ErrorIs(t, err, errUploadFileEmpty, "p2pChatFileBody must reject an empty file")

	body, contentType, err := p2pChatFileBody(`proof "1".png`, p2pTestPNG)
	require.NoError(t, err, "p2pChatFileBody must not error")
	mediaType, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err, "the content type must parse")
	assert.Equal(t, "multipart/form-data", mediaType, "p2pChatFileBody should build a multipart form")
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	part, err := reader.NextPart()
	require.NoError(t, err, "the body must hold a part")
	assert.Equal(t, "upload_file", part.FormName(), "p2pChatFileBody should send the file as upload_file")
	assert.Equal(t, `proof "1".png`, part.FileName(), "p2pChatFileBody should escape the file name")
	assert.Equal(t, "image/png", part.Header.Get("Content-Type"), "p2pChatFileBody should detect the file's content type")
	content, err := io.ReadAll(part)
	require.NoError(t, err, "reading the part must not error")
	assert.Equal(t, p2pTestPNG, content, "p2pChatFileBody should send the file unchanged")
	_, err = reader.NextPart()
	assert.ErrorIs(t, err, io.EOF, "p2pChatFileBody should send a single part")
}

func TestSendP2PChatMessage(t *testing.T) {
	t.Parallel()
	err := e.SendP2PChatMessage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SendP2PChatMessage must reject a nil request")
	r := &P2PSendChatMessageRequest{}
	err = e.SendP2PChatMessage(t.Context(), r)
	require.ErrorIs(t, err, errChatMessageRequired, "SendP2PChatMessage must require a message")
	r.Message = "Hello, I have paid."
	err = e.SendP2PChatMessage(t.Context(), r)
	require.ErrorIs(t, err, errChatContentTypeRequired, "SendP2PChatMessage must require a content type")
	r.ContentType = "str"
	err = e.SendP2PChatMessage(t.Context(), r)
	require.ErrorIs(t, err, errSessionIDRequired, "SendP2PChatMessage must require a session ID")
	r.SessionID = p2pChatSessionID
	err = e.SendP2PChatMessage(t.Context(), r)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "SendP2PChatMessage must require an order ID")
	r.OrderID = p2pOrderID

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.SendP2PChatMessage(t.Context(), r), "SendP2PChatMessage should not error")
}

func TestGetP2PChatMessageList(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PChatMessageList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2PChatMessageList must reject a nil request")
	r := &P2PChatMessageListRequest{}
	_, err = e.GetP2PChatMessageList(t.Context(), r)
	require.ErrorIs(t, err, errPageLimitRequired, "GetP2PChatMessageList must require a page size")
	r.Limit = 10
	_, err = e.GetP2PChatMessageList(t.Context(), r)
	require.ErrorIs(t, err, errSessionIDRequired, "GetP2PChatMessageList must require a session ID")
	r.SessionID = p2pChatSessionID

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PChatMessageList(t.Context(), r)
	require.NoError(t, err, "GetP2PChatMessageList must not error")
	if mockTests {
		exp := &P2PChatMessageListResponse{
			Messages: []P2PChatMessage{
				{
					ID:               preciseNumber(t, "1920001234567890"),
					SendUserNickName: "TestUser",
					ContentType:      "str",
					Message:          "{\"content\":\"Hello, I have paid.\",\"msgCode\":\"\",\"msgType\":\"\",\"fileName\":\"\",\"size\":\"\"}",
					CreateDate:       types.Time(time.Unix(1741832094, 0)),
				},
				{
					ID:               preciseNumber(t, "1920001234567889"),
					SendUserNickName: "TestMaker",
					ContentType:      "pic",
					Message:          "{\"content\":\"/fiat/p2p/oss/showObj/otc/9001/payment-proof.png\",\"msgCode\":\"\",\"msgType\":\"\",\"fileName\":\"payment-proof.png\",\"size\":\"102400\"}",
					CreateDate:       types.Time(time.Unix(1741831997, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetP2PChatMessageList should decode every field")
		return
	}
	assert.NotNil(t, result.Messages, "GetP2PChatMessageList should return messages")
}

func TestP2PChatMessageContent(t *testing.T) {
	t.Parallel()
	m := &P2PChatMessage{
		ID:      preciseNumber(t, "1920001234567889"),
		Message: `{"content":"/fiat/p2p/oss/showObj/otc/9001/payment-proof.png","msgCode":"1","msgType":"2","fileName":"payment-proof.png","size":"102400"}`,
	}
	content, err := m.MessageContent()
	require.NoError(t, err, "MessageContent must not error")
	exp := &P2PChatMessageContent{
		Content:     "/fiat/p2p/oss/showObj/otc/9001/payment-proof.png",
		MessageCode: "1",
		MessageType: "2",
		FileName:    "payment-proof.png",
		Size:        102400,
	}
	assert.Equal(t, exp, content, "MessageContent should decode every field")

	m.Message = "null"
	_, err = m.MessageContent()
	assert.ErrorIs(t, err, common.ErrNoResponse, "MessageContent should reject a message without content")

	m.Message = "Hello"
	_, err = m.MessageContent()
	assert.Error(t, err, "MessageContent should reject a message that is not JSON")
}

func TestP2PFieldsDecodeEitherForm(t *testing.T) {
	t.Parallel()
	// The first form of each is the parameter table's, the second the recorded example's
	requireDecodesEachForm(t, P2POnlineAd{UserID: preciseNumber(t, "100000003"), Side: 1, RecentOrderNumber: 5},
		`{"userId":100000003,"side":"1","recentOrderNum":"5"}`, `{"userId":"100000003","side":1,"recentOrderNum":5}`)
	requireDecodesEachForm(t, P2PUserInfoResponse{AccountID: preciseNumber(t, "100000002")}, `{"accountId":100000002}`, `{"accountId":"100000002"}`)
	requireDecodesEachForm(t, P2PChatSession{ID: preciseNumber(t, "128")}, `{"id":128}`, `{"id":"128"}`)
	requireDecodesEachForm(t, P2PChatMessage{ID: preciseNumber(t, "1920001234567890123")}, `{"id":1920001234567890123}`, `{"id":"1920001234567890123"}`)
}

// requireDecodesEachForm decodes every form into a T and compares each with exp
func requireDecodesEachForm[T any](t *testing.T, exp T, forms ...string) {
	t.Helper()
	for _, data := range forms {
		var got T
		require.NoErrorf(t, json.Unmarshal([]byte(data), &got), "Unmarshal must decode %s", data)
		assert.Equalf(t, exp, got, "Unmarshal should decode %s", data)
	}
}
