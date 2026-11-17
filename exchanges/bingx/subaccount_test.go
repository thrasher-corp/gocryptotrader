package bingx

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSubAccount must reject a nil request")
	_, err = e.CreateSubAccount(t.Context(), &CreateSubAccountRequest{Note: "demo"})
	require.ErrorIs(t, err, errSubAccountStringRequired, "CreateSubAccount must reject an empty username")

	t.Run("JSON body without a note", func(t *testing.T) {
		t.Parallel()
		ex := accountWalletJSONBodyExchange(t, "/openApi/subAccount/v1/create", `{"subAccountString":"abc123456"}`,
			`{"code":0,"timestamp":1702288510557,"data":{"subUid":16777654,"subAccountString":"abc123456","note":"demo"}}`)
		result, err := ex.CreateSubAccount(t.Context(), &CreateSubAccountRequest{SubAccountString: "abc123456"})
		require.NoError(t, err, "CreateSubAccount must not error")
		assert.Equal(t, &CreateSubAccountResponse{SubUID: 16777654, SubAccountString: "abc123456", Note: "demo"}, result, "CreateSubAccount should decode every field")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.CreateSubAccount(t.Context(), &CreateSubAccountRequest{SubAccountString: "abc123456", Note: "demo"})
		require.NoError(t, err, "CreateSubAccount must not error")
		require.NotNil(t, result, "CreateSubAccount must return a result")
		if mockTests {
			assert.Equal(t, &CreateSubAccountResponse{SubUID: 16777654, SubAccountString: "abc123456"}, result, "CreateSubAccount should decode the documented example")
		}
	})
}

func TestGetAPIKeyPermissions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAPIKeyPermissions(t.Context())
	require.NoError(t, err, "GetAPIKeyPermissions must not error")
	require.NotNil(t, result, "GetAPIKeyPermissions must return a result")
	if mockTests {
		exp := &APIKeyPermissionsResponse{
			Permissions: []APIKeyPermission{APIKeyPermissionSpotTrading, APIKeyPermissionRead},
			IPAddresses: []string{},
			Note:        "demo",
		}
		assert.Equal(t, exp, result, "GetAPIKeyPermissions should decode the documented example")
	}
}

func TestGetAccountUID(t *testing.T) {
	t.Parallel()
	t.Run("no UID", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":0,"timestamp":1702558965648,"data":{}}`)
		})
		_, err := ex.GetAccountUID(t.Context())
		assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccountUID should reject a reply without a UID")
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":100413,"msg":"Incorrect apiKey"}`)
		})
		result, err := ex.GetAccountUID(t.Context())
		assert.ErrorIs(t, err, errAPIResponse, "GetAccountUID should return the API error")
		assert.Nil(t, result, "GetAccountUID should not return a result with an error")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAccountUID(t.Context())
		require.NoError(t, err, "GetAccountUID must not error")
		require.NotNil(t, result, "GetAccountUID must return a result")
		if mockTests {
			assert.Equal(t, &AccountUIDResponse{UID: 16844999, AccountStatus: AccountStatusNormal}, result, "GetAccountUID should decode every field")
		}
	})
}

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SubAccountListRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no page", req: &SubAccountListRequest{Limit: 100}, err: errSubAccountPageRequired},
		{name: "no limit", req: &SubAccountListRequest{Page: 1}, err: errSubAccountPageSizeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSubAccountList(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetSubAccountList should return the validation error")
		})
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetSubAccountList(t.Context(), &SubAccountListRequest{
			SubUID:           16477999,
			SubAccountString: "abc123456",
			IsFeeze:          new(false),
			Page:             1,
			Limit:            100,
		})
		require.NoError(t, err, "GetSubAccountList must not error")
		require.NotNil(t, result, "GetSubAccountList must return a result")
		if mockTests {
			exp := &SubAccountListResponse{
				Result: []SubAccount{
					{SubUID: 16477999, SubAccountString: "abc123456", CreateTime: types.Time(time.UnixMilli(1700847351000))},
				},
				PageID: 1,
				Total:  1,
			}
			assert.Equal(t, exp, result, "GetSubAccountList should decode every field")
		}
	})
}

func TestGetSubAccountAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAssets(t.Context(), 0)
	require.ErrorIs(t, err, errSubAccountUIDRequired, "GetSubAccountAssets must reject an empty sub-account UID")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountAssets(t.Context(), 16477999)
	require.NoError(t, err, "GetSubAccountAssets must not error")
	require.NotNil(t, result, "GetSubAccountAssets must return a result")
	if mockTests {
		assert.Equal(t, &AssetBalancesResponse{Balances: []AssetBalance{{Asset: currency.ETH, Free: 0.0068}}}, result, "GetSubAccountAssets should decode every field")
	}
}

func TestCreateSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CreateSubAccountAPIKeyRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no sub-account UID", req: &CreateSubAccountAPIKeyRequest{Note: "abc6798", Permissions: []APIKeyPermission{APIKeyPermissionRead}}, err: errSubAccountUIDRequired},
		{name: "no note", req: &CreateSubAccountAPIKeyRequest{SubUID: 14189999, Permissions: []APIKeyPermission{APIKeyPermissionRead}}, err: errAPIKeyNoteRequired},
		{name: "no permissions", req: &CreateSubAccountAPIKeyRequest{SubUID: 14189999, Note: "abc6798"}, err: errAPIKeyPermissionsRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CreateSubAccountAPIKey(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "CreateSubAccountAPIKey should return the validation error")
		})
	}

	t.Run("JSON body without an IP whitelist", func(t *testing.T) {
		t.Parallel()
		ex := accountWalletJSONBodyExchange(t, "/openApi/subAccount/v1/apiKey/create", `{"subUid":14189999,"note":"abc6798","permissions":[2,3]}`,
			`{"code":0,"timestamp":1701526872165,"data":{"note":"abc6798","apiKey":"kRaent****jg","apiSecret":"2b****Og","permissions":[2,3],"ipAddresses":["8.8.8.8"]}}`)
		result, err := ex.CreateSubAccountAPIKey(t.Context(), &CreateSubAccountAPIKeyRequest{
			SubUID:      14189999,
			Note:        "abc6798",
			Permissions: []APIKeyPermission{APIKeyPermissionRead, APIKeyPermissionProfessionalContractTrading},
		})
		require.NoError(t, err, "CreateSubAccountAPIKey must not error")
		exp := &CreateSubAccountAPIKeyResponse{
			Note:        "abc6798",
			APIKey:      "kRaent****jg",
			APISecret:   "2b****Og",
			Permissions: []APIKeyPermission{APIKeyPermissionRead, APIKeyPermissionProfessionalContractTrading},
			IPAddresses: []string{"8.8.8.8"},
		}
		assert.Equal(t, exp, result, "CreateSubAccountAPIKey should decode every field")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.CreateSubAccountAPIKey(t.Context(), &CreateSubAccountAPIKeyRequest{
			SubUID:      14189999,
			Note:        "abc6798",
			Permissions: []APIKeyPermission{APIKeyPermissionProfessionalContractTrading},
			IPAddresses: []string{"8.8.8.8"},
		})
		require.NoError(t, err, "CreateSubAccountAPIKey must not error")
		require.NotNil(t, result, "CreateSubAccountAPIKey must return a result")
		if mockTests {
			exp := &CreateSubAccountAPIKeyResponse{
				Note:        "abc6798",
				APIKey:      "kRaent****jg",
				APISecret:   "2b****Og",
				Permissions: []APIKeyPermission{},
				IPAddresses: []string{},
			}
			assert.Equal(t, exp, result, "CreateSubAccountAPIKey should decode the documented example")
		}
	})
}

func TestGetAPIKeyInformation(t *testing.T) {
	t.Parallel()
	_, err := e.GetAPIKeyInformation(t.Context(), 0, "zF*******zQ")
	require.ErrorIs(t, err, errAccountUIDRequired, "GetAPIKeyInformation must reject an empty UID")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAPIKeyInformation(t.Context(), 16789999, "zF*******zQ")
	require.NoError(t, err, "GetAPIKeyInformation must not error")
	require.NotNil(t, result, "GetAPIKeyInformation must return a result")
	if mockTests {
		exp := &APIKeyInformationResponse{
			APIInfos: []APIKeyInformation{
				{
					APIKey: "zF*******zQ",
					Note:   "note****",
					Permissions: []APIKeyPermission{
						APIKeyPermissionSpotTrading,
						APIKeyPermissionRead,
						APIKeyPermissionProfessionalContractTrading,
						APIKeyPermissionUniversalTransfer,
						APIKeyPermissionSubAccountInternalTransfer,
					},
					IPAddresses: []string{""},
					CreateTime:  types.Time(time.UnixMilli(1702289687211)),
					UpdateTime:  types.Time(time.UnixMilli(1702289687000)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetAPIKeyInformation should decode every field")
	}
}

func TestEditSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *EditSubAccountAPIKeyRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no sub-account UID", req: &EditSubAccountAPIKeyRequest{APIKey: "CK***g", Note: "note", Permissions: []APIKeyPermission{APIKeyPermissionRead}}, err: errSubAccountUIDRequired},
		{name: "no API key", req: &EditSubAccountAPIKeyRequest{SubUID: 16259999, Note: "note", Permissions: []APIKeyPermission{APIKeyPermissionRead}}, err: errAPIKeyRequired},
		{name: "no note", req: &EditSubAccountAPIKeyRequest{SubUID: 16259999, APIKey: "CK***g", Permissions: []APIKeyPermission{APIKeyPermissionRead}}, err: errAPIKeyNoteRequired},
		{name: "no permissions", req: &EditSubAccountAPIKeyRequest{SubUID: 16259999, APIKey: "CK***g", Note: "note"}, err: errAPIKeyPermissionsRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.EditSubAccountAPIKey(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "EditSubAccountAPIKey should return the validation error")
		})
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		permissions := []APIKeyPermission{
			APIKeyPermissionSpotTrading,
			APIKeyPermissionRead,
			APIKeyPermissionProfessionalContractTrading,
			APIKeyPermissionUniversalTransfer,
			APIKeyPermission(6),
			APIKeyPermissionSubAccountInternalTransfer,
		}
		result, err := e.EditSubAccountAPIKey(t.Context(), &EditSubAccountAPIKeyRequest{
			SubUID:      16259999,
			APIKey:      "CK***g",
			Note:        "note",
			Permissions: permissions,
			IPAddresses: []string{"51.**.**.172", "51.**.**.135"},
		})
		require.NoError(t, err, "EditSubAccountAPIKey must not error")
		require.NotNil(t, result, "EditSubAccountAPIKey must return a result")
		if mockTests {
			exp := &EditSubAccountAPIKeyResponse{
				Note:        "note",
				Permissions: permissions,
				IPAddresses: []string{"51.**.**.172", "51.**.**.135"},
			}
			assert.Equal(t, exp, result, "EditSubAccountAPIKey should decode every field")
		}
	})
}

func TestDeleteSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *DeleteSubAccountAPIKeyRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no sub-account UID", req: &DeleteSubAccountAPIKeyRequest{APIKey: "2W****mA"}, err: errSubAccountUIDRequired},
		{name: "no API key", req: &DeleteSubAccountAPIKeyRequest{SubUID: 14189999}, err: errAPIKeyRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, e.DeleteSubAccountAPIKey(t.Context(), tc.req), tc.err, "DeleteSubAccountAPIKey should return the validation error")
		})
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		assert.NoError(t, e.DeleteSubAccountAPIKey(t.Context(), &DeleteSubAccountAPIKeyRequest{SubUID: 14189999, APIKey: "2W****mA"}), "DeleteSubAccountAPIKey should not error")
	})
}

func TestUpdateSubAccountStatus(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateSubAccountStatus(t.Context(), nil), common.ErrNilPointer, "UpdateSubAccountStatus must reject a nil request")
	require.ErrorIs(t, e.UpdateSubAccountStatus(t.Context(), &UpdateSubAccountStatusRequest{Freeze: true}), errSubAccountUIDRequired, "UpdateSubAccountStatus must reject an empty sub-account UID")

	t.Run("unfreeze is sent", func(t *testing.T) {
		t.Parallel()
		ex := accountWalletJSONBodyExchange(t, "/openApi/subAccount/v1/updateStatus", `{"subUid":16477999,"freeze":false}`, `{"code":0,"timestamp":1702520269455}`)
		assert.NoError(t, ex.UpdateSubAccountStatus(t.Context(), &UpdateSubAccountStatusRequest{SubUID: 16477999}), "UpdateSubAccountStatus should not error")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		assert.NoError(t, e.UpdateSubAccountStatus(t.Context(), &UpdateSubAccountStatusRequest{SubUID: 16477999, Freeze: true}), "UpdateSubAccountStatus should not error")
	})
}

func TestAuthoriseSubAccountInternalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.AuthoriseSubAccountInternalTransfer(t.Context(), nil, true)
	require.ErrorIs(t, err, errSubAccountUIDRequired, "AuthoriseSubAccountInternalTransfer must reject an empty list")
	_, err = e.AuthoriseSubAccountInternalTransfer(t.Context(), []uint64{16789999, 0}, true)
	require.ErrorIs(t, err, errSubAccountUIDRequired, "AuthoriseSubAccountInternalTransfer must reject an empty UID")

	t.Run("API error", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":100400,"msg":"invalid subUids"}`)
		})
		result, err := ex.AuthoriseSubAccountInternalTransfer(t.Context(), []uint64{16789999}, false)
		assert.ErrorIs(t, err, errAPIResponse, "AuthoriseSubAccountInternalTransfer should return the API error")
		assert.False(t, result, "AuthoriseSubAccountInternalTransfer should not report success with an error")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.AuthoriseSubAccountInternalTransfer(t.Context(), []uint64{16789999, 16789998}, true)
		require.NoError(t, err, "AuthoriseSubAccountInternalTransfer must not error")
		assert.True(t, result, "AuthoriseSubAccountInternalTransfer should return the documented result")
	})
}

func TestSubAccountInternalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountInternalTransfer(t.Context(), &InternalTransferRequest{Coin: currency.USDT, UserAccountType: UserAccountTypeUID, UserAccount: "16689999", WalletType: WalletTypeFund})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubAccountInternalTransfer must return the request's validation error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	result, err := e.SubAccountInternalTransfer(t.Context(), &InternalTransferRequest{
		Coin:             currency.NewCode("usdt"),
		UserAccountType:  UserAccountTypeUID,
		UserAccount:      "16689999",
		Amount:           20,
		WalletType:       WalletTypeFund,
		TransferClientID: "abc123",
	})
	require.NoError(t, err, "SubAccountInternalTransfer must not error")
	require.NotNil(t, result, "SubAccountInternalTransfer must return a result")
	if mockTests {
		assert.Equal(t, &InternalTransferResponse{ID: "12*******12"}, result, "SubAccountInternalTransfer should decode the documented example")
	}
}

func TestCreateSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CreateSubAccountDepositAddressRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty coin", req: &CreateSubAccountDepositAddressRequest{SubUID: 16769999, Network: "TRC20", WalletType: WalletTypeFund}, err: currency.ErrCurrencyCodeEmpty},
		{name: "no sub-account UID", req: &CreateSubAccountDepositAddressRequest{Coin: currency.USDT, Network: "TRC20", WalletType: WalletTypeFund}, err: errSubAccountUIDRequired},
		{name: "no network", req: &CreateSubAccountDepositAddressRequest{Coin: currency.USDT, SubUID: 16769999, WalletType: WalletTypeFund}, err: errDepositNetworkRequired},
		{name: "no wallet type", req: &CreateSubAccountDepositAddressRequest{Coin: currency.USDT, SubUID: 16769999, Network: "TRC20"}, err: errWalletTypeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CreateSubAccountDepositAddress(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "CreateSubAccountDepositAddress should return the validation error")
		})
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.CreateSubAccountDepositAddress(t.Context(), &CreateSubAccountDepositAddressRequest{Coin: currency.USDT, SubUID: 16769999, Network: "TRC20", WalletType: WalletTypeFund})
		require.NoError(t, err, "CreateSubAccountDepositAddress must not error")
		require.NotNil(t, result, "CreateSubAccountDepositAddress must return a result")
		if mockTests {
			exp := &CreateSubAccountDepositAddressResponse{
				Address:   "T********D",
				Coin:      currency.USDT,
				Network:   "TRC20",
				Status:    DepositAddressStatusActivated,
				Timestamp: types.Time(time.UnixMilli(1702520565000)),
			}
			assert.Equal(t, exp, result, "CreateSubAccountDepositAddress should decode every field")
		}
	})
}

func TestGetSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SubAccountDepositAddressRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty coin", req: &SubAccountDepositAddressRequest{SubUID: 16239999}, err: currency.ErrCurrencyCodeEmpty},
		{name: "no sub-account UID", req: &SubAccountDepositAddressRequest{Coin: currency.USDT}, err: errSubAccountUIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSubAccountDepositAddress(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetSubAccountDepositAddress should return the validation error")
		})
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetSubAccountDepositAddress(t.Context(), &SubAccountDepositAddressRequest{Coin: currency.USDT, SubUID: 16239999, Offset: 1, Limit: 100})
		require.NoError(t, err, "GetSubAccountDepositAddress must not error")
		require.NotNil(t, result, "GetSubAccountDepositAddress must return a result")
		if mockTests {
			assert.Equal(t, &DepositAddressListResponse{Data: []DepositAddress{}}, result, "GetSubAccountDepositAddress should decode the documented empty page")
		}
	})
}

func TestGetSubAccountDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountDepositRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountDepositRecords must reject a nil request")
	_, err = e.GetSubAccountDepositRecords(t.Context(), &SubAccountDepositRecordsRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubAccountDepositRecords must reject a reversed time range")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountDepositRecords(t.Context(), &SubAccountDepositRecordsRequest{
		Coin:      currency.USDT,
		SubUID:    16789999,
		Status:    new(DepositStatusCompleted),
		StartTime: time.UnixMilli(1702350000000),
		EndTime:   time.UnixMilli(1702436064748),
		Offset:    1,
		Limit:     100,
		TxID:      "abc123",
	})
	require.NoError(t, err, "GetSubAccountDepositRecords must not error")
	require.NotNil(t, result, "GetSubAccountDepositRecords must return a result")
	if mockTests {
		assert.Equal(t, &SubAccountDepositRecordsResponse{Data: []SubAccountDepositRecord{}}, result, "GetSubAccountDepositRecords should decode the documented empty page")
	}
}

func TestGetSubAccountInternalTransferRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountInternalTransferRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountInternalTransferRecords must return the request's validation error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountInternalTransferRecords(t.Context(), &InternalTransferRecordsRequest{
		Coin:             currency.BTC,
		TransferClientID: "abc123",
		StartTime:        time.UnixMilli(1701519898118),
		EndTime:          time.UnixMilli(1702383898118),
		Offset:           1,
		Limit:            100,
	})
	require.NoError(t, err, "GetSubAccountInternalTransferRecords must not error")
	require.NotNil(t, result, "GetSubAccountInternalTransferRecords must return a result")
	if mockTests {
		exp := &InternalTransferRecordsResponse{
			Data: []InternalTransferRecord{
				{
					ID:         1251111922229444400,
					Coin:       currency.BTC,
					Receiver:   1128763679,
					Amount:     0.0072366,
					Status:     WithdrawStatusCompleted,
					FromUID:    1128763678,
					RecordType: InternalTransferRecordTypeOut,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetSubAccountInternalTransferRecords should decode every field")
	}
}

func TestGetSubMotherAccountTransferHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SubMotherAccountTransferHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no UID", req: &SubMotherAccountTransferHistoryRequest{PageID: 1}, err: errAccountUIDRequired},
		{name: "reversed time range", req: &SubMotherAccountTransferHistoryRequest{UID: 213342, StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSubMotherAccountTransferHistory(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetSubMotherAccountTransferHistory should return the validation error")
		})
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetSubMotherAccountTransferHistory(t.Context(), &SubMotherAccountTransferHistoryRequest{
			UID:           213342,
			Type:          SubMotherAccountTransferTypeMainCapitalToSubCapital,
			TransactionID: "1051323896482406240336",
			StartTime:     time.UnixMilli(1719496046943),
			EndTime:       time.UnixMilli(1719596046943),
			PageID:        1,
			PagingSize:    10,
		})
		require.NoError(t, err, "GetSubMotherAccountTransferHistory must not error")
		require.NotNil(t, result, "GetSubMotherAccountTransferHistory must return a result")
		if mockTests {
			exp := &SubMotherAccountTransferHistoryResponse{
				Total: 1,
				Rows: []SubMotherAccountTransfer{
					{
						Asset:         currency.NewCode("VST"),
						Amount:        100000,
						Type:          SubMotherAccountTransferTypeMainCapitalToSubCapital,
						Status:        "CONFIRMED",
						TransactionID: "1051323896482406240336",
						Timestamp:     types.Time(time.UnixMilli(1702252271000)),
						FromUID:       2332424,
						ToUID:         123244,
					},
				},
			}
			assert.Equal(t, exp, result, "GetSubMotherAccountTransferHistory should decode every field")
		}
	})
}

func TestGetSubMotherAccountTransferableAmount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SubMotherAccountTransferableAmountRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no from UID", req: &SubMotherAccountTransferableAmountRequest{FromAccountType: WalletTypeStandardFutures, ToUID: 25316652, ToAccountType: WalletTypeFund}, err: errAccountUIDRequired},
		{name: "no from account type", req: &SubMotherAccountTransferableAmountRequest{FromUID: 25472377, ToUID: 25316652, ToAccountType: WalletTypeFund}, err: errWalletTypeRequired},
		{name: "no to UID", req: &SubMotherAccountTransferableAmountRequest{FromUID: 25472377, FromAccountType: WalletTypeStandardFutures, ToAccountType: WalletTypeFund}, err: errAccountUIDRequired},
		{name: "no to account type", req: &SubMotherAccountTransferableAmountRequest{FromUID: 25472377, FromAccountType: WalletTypeStandardFutures, ToUID: 25316652}, err: errWalletTypeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSubMotherAccountTransferableAmount(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetSubMotherAccountTransferableAmount should return the validation error")
		})
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetSubMotherAccountTransferableAmount(t.Context(), &SubMotherAccountTransferableAmountRequest{
			FromUID:         25472377,
			FromAccountType: WalletTypeStandardFutures,
			ToUID:           25316652,
			ToAccountType:   WalletTypeFund,
		})
		require.NoError(t, err, "GetSubMotherAccountTransferableAmount must not error")
		require.NotNil(t, result, "GetSubMotherAccountTransferableAmount must return a result")
		if mockTests {
			exp := &SubMotherAccountTransferableAmountResponse{
				Coins: []SubMotherAccountTransferableCoin{
					{
						ID:              4,
						Name:            currency.USDT,
						ShowName:        "Tether",
						Icon:            CoinIcon{ID: 4, URI: "https://static-app.teststar.cc/icon/USDT.png"},
						FiatSymbol:      "$",
						AvailableAmount: 79.02345678,
						USDTRate:        1,
					},
				},
			}
			assert.Equal(t, exp, result, "GetSubMotherAccountTransferableAmount should decode every field")
		}
	})
}

func TestSubMotherAccountAssetTransfer(t *testing.T) {
	t.Parallel()
	valid := SubMotherAccountAssetTransferRequest{
		AssetName:       currency.NewCode("usdt"),
		TransferAmount:  1.1,
		FromUID:         25472377,
		FromType:        SubAccountTypeSubAccount,
		FromAccountType: WalletTypeStandardFutures,
		ToUID:           25316652,
		ToType:          SubAccountTypeParentAccount,
		ToAccountType:   WalletTypeFund,
		Remark:          "rebalance",
	}
	for _, tc := range []struct {
		name   string
		modify func(*SubMotherAccountAssetTransferRequest)
		err    error
	}{
		{name: "empty asset", modify: func(r *SubMotherAccountAssetTransferRequest) { r.AssetName = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "zero amount", modify: func(r *SubMotherAccountAssetTransferRequest) { r.TransferAmount = 0 }, err: limits.ErrAmountBelowMin},
		{name: "no from UID", modify: func(r *SubMotherAccountAssetTransferRequest) { r.FromUID = 0 }, err: errAccountUIDRequired},
		{name: "no from type", modify: func(r *SubMotherAccountAssetTransferRequest) { r.FromType = 0 }, err: errSubAccountTypeRequired},
		{name: "no from account type", modify: func(r *SubMotherAccountAssetTransferRequest) { r.FromAccountType = 0 }, err: errWalletTypeRequired},
		{name: "no to UID", modify: func(r *SubMotherAccountAssetTransferRequest) { r.ToUID = 0 }, err: errAccountUIDRequired},
		{name: "no to type", modify: func(r *SubMotherAccountAssetTransferRequest) { r.ToType = 0 }, err: errSubAccountTypeRequired},
		{name: "no to account type", modify: func(r *SubMotherAccountAssetTransferRequest) { r.ToAccountType = 0 }, err: errWalletTypeRequired},
		{name: "no remark", modify: func(r *SubMotherAccountAssetTransferRequest) { r.Remark = "" }, err: errTransferRemarkRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := valid
			tc.modify(&req)
			_, err := e.SubMotherAccountAssetTransfer(t.Context(), &req)
			assert.ErrorIs(t, err, tc.err, "SubMotherAccountAssetTransfer should return the validation error")
		})
	}

	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		_, err := e.SubMotherAccountAssetTransfer(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrNilPointer, "SubMotherAccountAssetTransfer should reject a nil request")
	})

	t.Run("reply without a nested envelope", func(t *testing.T) {
		t.Parallel()
		ex := accountWalletJSONBodyExchange(t, "/openApi/account/transfer/v1/subAccount/transferAsset",
			`{"assetName":"USDT","transferAmount":1.1,"fromUid":25472377,"fromType":2,"fromAccountType":2,"toUid":25316652,"toType":1,"toAccountType":1,"remark":"rebalance"}`,
			`{"code":0,"timestamp":1719494258281,"data":{"tranId":"1051323892566796963873"}}`)
		req := valid
		result, err := ex.SubMotherAccountAssetTransfer(t.Context(), &req)
		require.NoError(t, err, "SubMotherAccountAssetTransfer must not error")
		assert.Equal(t, &SubMotherAccountAssetTransferResponse{TransactionID: "1051323892566796963873"}, result, "SubMotherAccountAssetTransfer should decode an unnested reply")
		assert.Equal(t, "usdt", req.AssetName.String(), "the request's asset should be left as it was")
	})

	t.Run("error in the nested envelope", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":0,"timestamp":1719494258281,"data":{"code":100400,"msg":"insufficient balance","timestamp":1719495091669}}`)
		})
		req := valid
		result, err := ex.SubMotherAccountAssetTransfer(t.Context(), &req)
		require.ErrorIs(t, err, errAPIResponse, "SubMotherAccountAssetTransfer must return the nested error")
		assert.ErrorContains(t, err, "code 100400 msg insufficient balance", "the error should carry the nested code and message")
		assert.Nil(t, result, "SubMotherAccountAssetTransfer should not return a result with an error")
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":100001,"msg":"signature verification failed"}`)
		})
		req := valid
		result, err := ex.SubMotherAccountAssetTransfer(t.Context(), &req)
		assert.ErrorIs(t, err, errAPIResponse, "SubMotherAccountAssetTransfer should return the API error")
		assert.Nil(t, result, "SubMotherAccountAssetTransfer should not return a result with an error")
	})

	t.Run("nested envelope without data", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":0,"timestamp":1719494258281,"data":{"code":0,"timestamp":1719495091669,"data":null}}`)
		})
		req := valid
		_, err := ex.SubMotherAccountAssetTransfer(t.Context(), &req)
		assert.ErrorIs(t, err, common.ErrNoResponse, "SubMotherAccountAssetTransfer should report a nested envelope without data")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		req := valid
		result, err := e.SubMotherAccountAssetTransfer(t.Context(), &req)
		require.NoError(t, err, "SubMotherAccountAssetTransfer must not error")
		require.NotNil(t, result, "SubMotherAccountAssetTransfer must return a result")
		if mockTests {
			assert.Equal(t, &SubMotherAccountAssetTransferResponse{TransactionID: "1051323892566796963873"}, result, "SubMotherAccountAssetTransfer should decode the documented nested reply")
		}
	})
}

func TestGetSubAccountAssetOverview(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SubAccountAssetOverviewRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no page index", req: &SubAccountAssetOverviewRequest{PageSize: 10}, err: errSubAccountPageRequired},
		{name: "no page size", req: &SubAccountAssetOverviewRequest{PageIndex: 1}, err: errSubAccountPageSizeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSubAccountAssetOverview(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetSubAccountAssetOverview should return the validation error")
		})
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetSubAccountAssetOverview(t.Context(), &SubAccountAssetOverviewRequest{SubUID: 25316652, AccountType: AccountTypeSpot, PageIndex: 1, PageSize: 10})
		require.NoError(t, err, "GetSubAccountAssetOverview must not error")
		require.NotNil(t, result, "GetSubAccountAssetOverview must return a result")
		if mockTests {
			exp := &SubAccountAssetOverviewResponse{
				Result: []SubAccountAssetOverview{
					{SubUID: 16477999, AccountBalances: []AccountBalanceOverview{{AccountType: AccountTypeSpot, USDTBalance: 100}}},
				},
				PageID: 1,
				Total:  1,
			}
			assert.Equal(t, exp, result, "GetSubAccountAssetOverview should decode every field")
		}
	})
}
