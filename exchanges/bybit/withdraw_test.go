package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetWithdrawalAddressList(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalAddressList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWithdrawalAddressList must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawalAddressList(t.Context(), &WithdrawalAddressListRequest{Chain: "ETH"})
	require.NoError(t, err, "GetWithdrawalAddressList must not error")
	t.Run("universal addresses", func(t *testing.T) {
		t.Parallel()
		// A chain with coin baseCoin returns only the chain's universal addresses
		next, err := e.GetWithdrawalAddressList(t.Context(), &WithdrawalAddressListRequest{Coin: "baseCoin", Chain: "ETH", Limit: 10, Cursor: result.NextPageCursor})
		require.NoError(t, err, "GetWithdrawalAddressList must not error")
		if mockTests {
			exp := &WithdrawalAddressListResponse{
				Rows: []WithdrawalAddress{
					{
						Coin:      "baseCoin",
						Chain:     "ETH",
						Address:   "0xE3De6d711e0951d34777b5Cd93c827F822ee8514",
						Remark:    "Cold wallet",
						Verified:  true,
						CreatedAt: types.Time(time.Unix(1760900000, 0)),
					},
				},
			}
			assert.Equal(t, exp, next, "GetWithdrawalAddressList should send the coin, chain, limit and cursor and decode the page")
			return
		}
		assert.NotNil(t, next, "GetWithdrawalAddressList should return a page")
	})
	t.Run("internal transfer addresses", func(t *testing.T) {
		t.Parallel()
		// Bybit ignores the coin and chain filters for internal transfer addresses
		internal, err := e.GetWithdrawalAddressList(t.Context(), &WithdrawalAddressListRequest{AddressType: 1})
		require.NoError(t, err, "GetWithdrawalAddressList must not error")
		if mockTests {
			exp := &WithdrawalAddressListResponse{
				Rows: []WithdrawalAddress{
					{
						Address:     "100000002",
						Remark:      "Sub-account",
						AddressType: 1,
						Verified:    true,
						CreatedAt:   types.Time(time.Unix(1760800000, 0)),
					},
				},
			}
			assert.Equal(t, exp, internal, "GetWithdrawalAddressList should send the address type and decode the internal transfer addresses")
			return
		}
		assert.NotNil(t, internal, "GetWithdrawalAddressList should return a page")
	})
	if mockTests {
		exp := &WithdrawalAddressListResponse{
			Rows: []WithdrawalAddress{
				{
					Coin:      "USDT",
					Chain:     "ETH",
					Address:   "0x48101adb67d426cb15e46be5f1d9f6ab25f311ea",
					CreatedAt: types.Time(time.Unix(1760951195, 0)),
				},
				{
					Coin:        "baseCoin",
					Chain:       "ETH",
					Address:     "0x48101adb67d426cb15e46be5f1d9f6ab25f311ea",
					Tag:         "memo1",
					Remark:      "Universal Address",
					Status:      1,
					AddressType: 2,
					Verified:    true,
					CreatedAt:   types.Time(time.Unix(1760951332, 0)),
				},
			},
			NextPageCursor: "eyJtaW5JRCI6MTA1MDgsIm1heElEIjoxMDUwOX0=",
		}
		assert.Equal(t, exp, result, "GetWithdrawalAddressList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetWithdrawalAddressList should return a page")
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWithdrawalRecords must reject a nil request")
	_, err = e.GetWithdrawalRecords(t.Context(), &WithdrawalRecordsRequest{StartTime: time.UnixMilli(1742738305000), EndTime: time.UnixMilli(1742700000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWithdrawalRecords must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawalRecords(t.Context(), &WithdrawalRecordsRequest{Coin: currency.LTC, Limit: 10})
	require.NoError(t, err, "GetWithdrawalRecords must not error")
	t.Run("every filter", func(t *testing.T) {
		t.Parallel()
		// Withdraw type 2 asks for both on-chain and off-chain withdrawals
		next, err := e.GetWithdrawalRecords(t.Context(), &WithdrawalRecordsRequest{
			WithdrawID:    "131700001",
			TransactionID: "6b1d0f3e2a4c5b7d9e8f0a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f",
			Coin:          currency.NewCode("ltc"),
			WithdrawType:  2,
			StartTime:     time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			EndTime:       time.Date(2026, time.September, 8, 0, 0, 0, 0, time.UTC),
			Limit:         10,
			Cursor:        result.NextPageCursor,
		})
		require.NoError(t, err, "GetWithdrawalRecords must not error")
		if mockTests {
			exp := &WithdrawalRecordsResponse{
				Rows: []WithdrawalRecord{
					{
						TransactionID: "6b1d0f3e2a4c5b7d9e8f0a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f",
						Coin:          currency.LTC,
						Chain:         "LTC",
						Amount:        3,
						WithdrawFee:   0.001,
						Status:        "success",
						ToAddress:     "XXXXXX",
						CreateTime:    types.Time(time.UnixMilli(1788400000000)),
						UpdateTime:    types.Time(time.UnixMilli(1788400600000)),
						WithdrawID:    "131700001",
					},
				},
			}
			assert.Equal(t, exp, next, "GetWithdrawalRecords should send every filter and decode the matching page")
			return
		}
		assert.NotNil(t, next, "GetWithdrawalRecords should return a page")
	})
	if mockTests {
		exp := &WithdrawalRecordsResponse{
			Rows: []WithdrawalRecord{
				{
					TransactionID: "9f4c1e2b3a4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7",
					Coin:          currency.LTC,
					Chain:         "LTC",
					Amount:        2.5,
					WithdrawFee:   0.001,
					Status:        "success",
					ToAddress:     "XXXXXX",
					CreateTime:    types.Time(time.Unix(1742738305, 0)),
					UpdateTime:    types.Time(time.Unix(1742738340, 0)),
					WithdrawID:    "131629076",
					Tax:           0.0125,
					TaxRate:       0.005,
					TaxType:       "TDS",
				},
				{
					Coin:         currency.LTC,
					Chain:        "LTC",
					Amount:       1,
					Status:       "CancelByUser",
					ToAddress:    "100000002",
					Tag:          "memo1",
					CreateTime:   types.Time(time.Unix(1742729329, 0)),
					UpdateTime:   types.Time(time.Unix(1742729437, 0)),
					WithdrawID:   "131603458",
					WithdrawType: 1,
					Fee:          0.0001,
				},
			},
			NextPageCursor: "eyJtaW5JRCI6MTMxNjAzNDU4LCJtYXhJRCI6MTMxNjI5MDc2fQ==",
		}
		assert.Equal(t, exp, result, "GetWithdrawalRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetWithdrawalRecords should return a page")
}

func TestGetAvailableVASPs(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAvailableVASPs(t.Context())
	require.NoError(t, err, "GetAvailableVASPs must not error")
	if mockTests {
		exp := &AvailableVASPsResponse{
			VASP: []VASP{
				{
					VASPEntityID: "basic-finance",
					VASPName:     "Basic-finance",
				},
				{
					VASPEntityID: "korbit",
					VASPName:     "korbit",
				},
				{
					VASPEntityID: "others",
					VASPName:     "Others (including Upbit)",
				},
			},
		}
		assert.Equal(t, exp, result, "GetAvailableVASPs should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAvailableVASPs should return the list")
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawCurrency must reject a nil request")
	arg := &WithdrawalRequest{}
	_, err = e.WithdrawCurrency(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "WithdrawCurrency must reject an empty coin")
	arg.Coin = currency.LTC
	_, err = e.WithdrawCurrency(t.Context(), arg)
	require.ErrorIs(t, err, errMissingChainInformation, "WithdrawCurrency must reject an empty chain for an on-chain withdrawal")
	arg.ForceChain = forceChainUID
	_, err = e.WithdrawCurrency(t.Context(), arg)
	require.ErrorIs(t, err, errMissingAddressInfo, "WithdrawCurrency must accept an empty chain for a UID withdrawal and reject an empty address")
	arg.ForceChain = 1
	arg.Chain = "LTC"
	arg.Address = "234234234"
	_, err = e.WithdrawCurrency(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "WithdrawCurrency must reject a zero amount")
	arg.Amount = 0.1
	_, err = e.WithdrawCurrency(t.Context(), arg)
	require.ErrorIs(t, err, errMissingAccountType, "WithdrawCurrency must reject an empty account type")
	arg.AccountType = "FUND"
	arg.Questionnaire = &TravelRuleQuestionnaire{WalletType: 1}
	_, err = e.WithdrawCurrency(t.Context(), arg)
	require.ErrorIs(t, err, errLegalTypeRequired, "WithdrawCurrency must reject a questionnaire without a legal type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	isSelfWallet := true
	arg.Tag = "memo1"
	arg.FeeType = 1
	arg.RequestID = "test00001"
	arg.TransactionPurpose = "Long-term holding for retirement savings purpose"
	arg.Questionnaire = &TravelRuleQuestionnaire{
		WalletType:         1,
		LegalType:          "individual",
		IsSelfWallet:       &isSelfWallet,
		FirstName:          "Ahmet",
		LastName:           "Yilmaz",
		TransactionPurpose: "Long-term holding for retirement savings purpose",
		POIType:            "ID_CARD",
		POINumber:          "12345678901",
		POIIssuingCountry:  "TUR",
		POIExpiredDate:     time.Date(2030, time.June, 15, 0, 0, 0, 0, time.UTC),
	}
	arg.Beneficiary = &WithdrawalBeneficiary{
		TransactionPurpose:      "Payment for consulting services rendered",
		RepresentativeFirstName: "Min-jun",
		RepresentativeLastName:  "Kim",
		VASPEntityID:            "others",
		Name:                    "Example Co., Ltd.",
		LegalType:               "company",
		WalletType:              "1",
		UnhostedWalletType:      "1",
		POINumber:               "BIN1234567890",
		POIType:                 "OTHER",
		POIIssuingCountry:       "KAZ",
		POIExpiredDate:          time.Date(2099, time.December, 31, 0, 0, 0, 0, time.UTC),
		AddressCountry:          "ARE",
		AddressState:            "Dubai",
		AddressCity:             "Dubai",
		AddressBuilding:         "Example Tower",
		AddressStreet:           "1 Example Street",
		AddressPostalCode:       "00000",
		DateOfBirth:             "1990-02-15",
		PlaceOfBirth:            "Dubai",
	}
	result, err := e.WithdrawCurrency(t.Context(), arg)
	require.NoError(t, err, "WithdrawCurrency must not error")
	if mockTests {
		exp := &WithdrawalResponse{
			ID: "10195",
		}
		assert.Equal(t, exp, result, "WithdrawCurrency should decode every field")
		return
	}
	assert.NotEmpty(t, result.ID, "WithdrawCurrency should return the withdrawal ID")
}

func TestWithdrawalBeneficiaryPayload(t *testing.T) {
	t.Parallel()
	got := (&WithdrawalBeneficiary{
		TransactionPurpose:      "Payment for consulting services rendered",
		RepresentativeFirstName: "Min-jun",
		RepresentativeLastName:  "Kim",
		VASPEntityID:            "others",
		Name:                    "Example Co., Ltd.",
		LegalType:               "company",
		WalletType:              "1",
		UnhostedWalletType:      "1",
		POINumber:               "BIN1234567890",
		POIType:                 "OTHER",
		POIIssuingCountry:       "KAZ",
		POIExpiredDate:          time.Date(2099, time.December, 31, 0, 0, 0, 0, time.UTC),
		AddressCountry:          "ARE",
		AddressState:            "Dubai",
		AddressCity:             "Dubai",
		AddressBuilding:         "Example Tower",
		AddressStreet:           "1 Example Street",
		AddressPostalCode:       "00000",
		DateOfBirth:             "1990-02-15",
		PlaceOfBirth:            "Dubai",
	}).payload()
	exp := map[string]any{
		"beneficiaryTransactionPurpose":      "Payment for consulting services rendered",
		"beneficiaryRepresentativeFirstName": "Min-jun",
		"beneficiaryRepresentativeLastName":  "Kim",
		"vaspEntityId":                       "others",
		"beneficiaryName":                    "Example Co., Ltd.",
		"beneficiaryLegalType":               "company",
		"beneficiaryWalletType":              "1",
		"beneficiaryUnhostedWalletType":      "1",
		"beneficiaryPoiNumber":               "BIN1234567890",
		"beneficiaryPoiType":                 "OTHER",
		"beneficiaryPoiIssuingCountry":       "KAZ",
		"beneficiaryPoiExpiredDate":          "2099-12-31",
		"beneficiaryAddressCountry":          "ARE",
		"beneficiaryAddressState":            "Dubai",
		"beneficiaryAddressCity":             "Dubai",
		"beneficiaryAddressBuilding":         "Example Tower",
		"beneficiaryAddressStreet":           "1 Example Street",
		"beneficiaryAddressPostalCode":       "00000",
		"beneficiaryDateOfBirth":             "1990-02-15",
		"beneficiaryPlaceOfBirth":            "Dubai",
	}
	assert.Equal(t, exp, got, "payload should send every field set under its documented key")
	assert.Empty(t, (&WithdrawalBeneficiary{}).payload(), "payload should omit unset fields")
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawal(t.Context(), "")
	require.ErrorIs(t, err, errMissingWithdrawalID, "CancelWithdrawal must reject an empty withdrawal ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelWithdrawal(t.Context(), "12314")
	require.NoError(t, err, "CancelWithdrawal must not error")
	if mockTests {
		exp := &StatusResponse{
			Status: true,
		}
		assert.Equal(t, exp, result, "CancelWithdrawal should decode every field")
		return
	}
	assert.True(t, result.Status.Bool(), "CancelWithdrawal should succeed")
}

func TestTravelRuleQuestionnaireEncode(t *testing.T) {
	t.Parallel()
	_, err := (&TravelRuleQuestionnaire{FirstName: "Rahul"}).encode()
	require.ErrorIs(t, err, errLegalTypeRequired, "encode must reject a questionnaire without a legal type")

	isSelfWallet := false
	got, err := (&TravelRuleQuestionnaire{
		LegalType:               "company",
		VASPCode:                "EXAMPLE_VASP",
		IsSelfWallet:            &isSelfWallet,
		FirstName:               "John",
		LastName:                "Smith",
		CompanyName:             "Example LLC",
		TransactionPurpose:      "Settlement of an invoice for services",
		POIType:                 "OTHER",
		POINumber:               "BIN1234567890",
		POIIssuingCountry:       "KAZ",
		POIExpiredDate:          time.Date(2099, time.December, 31, 0, 0, 0, 0, time.UTC),
		RepresentativeFirstName: "Min-jun",
		RepresentativeLastName:  "Kim",
	}).encode()
	require.NoError(t, err, "encode must not error")
	exp := `{"companyName":"Example LLC","firstName":"John","isSelfWallet":false,"lastName":"Smith","legalType":"company",` +
		`"poiExpiredDate":"2099-12-31","poiIssuingCountry":"KAZ","poiNumber":"BIN1234567890","poiType":"OTHER",` +
		`"representativeFirstName":"Min-jun","representativeLastName":"Kim","transactionPurpose":"Settlement of an invoice for services",` +
		`"vaspCode":"EXAMPLE_VASP","walletType":0}`
	assert.Equal(t, exp, got, "encode should send every field set, the custodial wallet type and an explicit false")
}
