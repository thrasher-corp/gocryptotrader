package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestSignAgreement(t *testing.T) {
	t.Parallel()
	err := e.SignAgreement(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SignAgreement must reject a nil request")
	err = e.SignAgreement(t.Context(), &SignAgreementRequest{})
	require.ErrorIs(t, err, errAgreementCategoryRequired, "SignAgreement must reject a missing category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.SignAgreement(t.Context(), &SignAgreementRequest{CategoryV2: 2}), "SignAgreement must not error with categoryV2")
	require.NoError(t, e.SignAgreement(t.Context(), &SignAgreementRequest{Category: 3}), "SignAgreement must not error with category")
}

func TestCreateNewSubUserID(t *testing.T) {
	t.Parallel()
	_, err := e.CreateNewSubUserID(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateNewSubUserID must reject a nil request")
	_, err = e.CreateNewSubUserID(t.Context(), &CreateSubUserRequest{MemberType: 1})
	require.ErrorIs(t, err, errMissingUsername, "CreateNewSubUserID must reject a missing username")
	_, err = e.CreateNewSubUserID(t.Context(), &CreateSubUserRequest{Username: "gctsub01"})
	require.ErrorIs(t, err, errInvalidMemberType, "CreateNewSubUserID must reject a missing member type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateNewSubUserID(t.Context(), &CreateSubUserRequest{Username: "gctsub01", Password: "Example1234", MemberType: 1, Switch: true, Note: "test"})
	require.NoError(t, err, "CreateNewSubUserID must not error")
	if mockTests {
		exp := &CreateSubUserResponse{
			UID:        "100000002",
			Username:   "gctsub01",
			MemberType: 1,
			Status:     1,
			Remark:     "test",
		}
		assert.Equal(t, exp, result, "CreateNewSubUserID should decode every field")
		return
	}
	assert.NotEmpty(t, result.UID, "CreateNewSubUserID should return the new sub UID")
}

func TestCreateSubUIDAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubUIDAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSubUIDAPIKey must reject a nil request")
	_, err = e.CreateSubUIDAPIKey(t.Context(), &CreateSubUIDAPIKeyRequest{})
	require.ErrorIs(t, err, errMissingUserID, "CreateSubUIDAPIKey must reject a missing sub UID")
	_, err = e.CreateSubUIDAPIKey(t.Context(), &CreateSubUIDAPIKeyRequest{SubUID: 100000002})
	require.ErrorIs(t, err, errAPIKeyPermissionsRequired, "CreateSubUIDAPIKey must reject missing permissions")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSubUIDAPIKey(t.Context(), &CreateSubUIDAPIKeyRequest{
		SubUID:   100000002,
		Note:     "gct test",
		ReadOnly: true,
		IPs:      []string{"192.0.2.1", "192.0.2.2"},
		Permissions: CreateSubUIDAPIKeyPermissionsRequest{
			Wallet: []string{"AccountTransfer"},
			Spot:   []string{"SpotTrade"},
		},
	})
	require.NoError(t, err, "CreateSubUIDAPIKey must not error")
	if mockTests {
		exp := &CreateSubUIDAPIKeyResponse{
			ID:       "16651283",
			Note:     "gct test",
			APIKey:   "mock-sub-api-key",
			ReadOnly: true,
			Secret:   "mock-sub-api-secret",
			Permissions: SubAPIKeyPermissions{
				ContractTrade:     []string{},
				Spot:              []string{"SpotTrade"},
				Wallet:            []string{"AccountTransfer"},
				Options:           []string{},
				Derivatives:       []string{},
				Exchange:          []string{},
				Earn:              []string{},
				BlockTrade:        []string{},
				Affiliate:         []string{},
				FiatP2P:           []string{},
				FiatConvertBroker: []string{},
				NFT:               []string{},
				CopyTrading:       []string{},
			},
		}
		assert.Equal(t, exp, result, "CreateSubUIDAPIKey should decode every field")
		return
	}
	assert.NotEmpty(t, result.APIKey, "CreateSubUIDAPIKey should return the new API key")
}

func TestGetSubUIDList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubUIDList(t.Context())
	require.NoError(t, err, "GetSubUIDList must not error")
	if mockTests {
		exp := &SubUIDListResponse{
			SubMembers: []SubMember{
				{
					UID:         "100000002",
					Username:    "gctsub01",
					MemberType:  1,
					Status:      1,
					AccountMode: 5,
					Remark:      "test",
				},
				{
					UID:         "100000003",
					Username:    "gctsub02",
					MemberType:  6,
					Status:      4,
					AccountMode: 6,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSubUIDList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubUIDList should return a result")
}

func TestGetSubUIDListUnlimited(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubUIDListUnlimited(t.Context(), 2, "300")
	require.NoError(t, err, "GetSubUIDListUnlimited must not error")
	if mockTests {
		exp := &SubUIDListUnlimitedResponse{
			SubMembers: []SubMember{
				{
					UID:         "100000004",
					Username:    "gctsub03",
					MemberType:  4,
					Status:      1,
					AccountMode: 5,
					Remark:      "copy trading",
				},
				{
					UID:         "100000005",
					Username:    "gctsub04",
					MemberType:  16,
					Status:      2,
					AccountMode: 6,
				},
			},
			NextCursor: "302",
		}
		assert.Equal(t, exp, result, "GetSubUIDListUnlimited should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubUIDListUnlimited should return a result")
}

func TestGetFundCustodialSubAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFundCustodialSubAccounts(t.Context(), 2, "342")
	require.NoError(t, err, "GetFundCustodialSubAccounts must not error")
	if mockTests {
		exp := &FundCustodialSubAccountsResponse{
			SubMembers: []SubMember{
				{
					UID:         "100000006",
					Username:    "Private_Wealth_Management",
					MemberType:  12,
					Status:      1,
					AccountMode: 3,
					Remark:      "earn fund",
				},
				{
					UID:         "100000007",
					Username:    "Private_Wealth_Management",
					MemberType:  12,
					Status:      1,
					AccountMode: 1,
					Remark:      "earn fund",
				},
			},
			NextCursor: "344",
		}
		assert.Equal(t, exp, result, "GetFundCustodialSubAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFundCustodialSubAccounts should return a result")
}

func TestFreezeSubUID(t *testing.T) {
	t.Parallel()
	err := e.FreezeSubUID(t.Context(), 0, true)
	require.ErrorIs(t, err, errMissingUserID, "FreezeSubUID must reject a missing sub UID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.FreezeSubUID(t.Context(), 100000002, true), "FreezeSubUID must not error")
}

func TestGetAPIKeyInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAPIKeyInformation(t.Context())
	require.NoError(t, err, "GetAPIKeyInformation must not error")
	if mockTests {
		exp := &APIKeyInformationResponse{
			ID:       "2208369",
			Note:     "gct test",
			APIKey:   "mock-api-key",
			ReadOnly: true,
			Secret:   "******",
			Permissions: APIKeyInformationPermissions{
				ContractTrade:     []string{"Order", "Position"},
				Spot:              []string{"SpotTrade"},
				Wallet:            []string{"AccountTransfer", "SubMemberTransfer"},
				Options:           []string{"OptionsTrade"},
				Derivatives:       []string{"DerivativesTrade"},
				Exchange:          []string{"ExchangeHistory"},
				Earn:              []string{"Earn"},
				FiatP2P:           []string{"FiatP2POrder", "Advertising"},
				FiatBitPay:        []string{"FaitPayOrder"},
				FiatConvertBroker: []string{"FiatConvertBrokerOrder"},
				BitCard:           []string{"BitCard"},
				ByXPost:           []string{"ByXPost"},
				Affiliate:         []string{},
				BlockTrade:        []string{"BlockTrade"},
				NFT:               []string{},
				CopyTrading:       []string{},
				FiatGlobalPay:     []string{},
			},
			IPs:                 []string{"192.0.2.1", "192.0.2.2"},
			Type:                1,
			DeadlineDay:         83,
			ExpiredAt:           time.Date(2026, 12, 31, 3, 20, 45, 0, time.UTC),
			CreatedAt:           time.Date(2025, 10, 13, 3, 20, 45, 0, time.UTC),
			UnifiedTradeAccount: true,
			UserID:              100000001,
			InviterID:           100000009,
			VIPLevel:            "PRO-1",
			MarketMakerLevel:    "0",
			AffiliateID:         100000010,
			RSAPublicKey:        "mock-rsa-public-key",
			IsFIXAPI:            true,
			IsMaster:            true,
			ParentUID:           "0",
			KYCLevel:            "LEVEL_1",
			KYCRegion:           "AUS",
			Unified:             true,
			UserIDInt64:         "100000001",
			InviterIDInt64:      "100000009",
			AffiliateIDInt64:    "100000010",
		}
		assert.Equal(t, exp, result, "GetAPIKeyInformation should decode every field")
		return
	}
	assert.NotEmpty(t, result.APIKey, "GetAPIKeyInformation should return the API key")
}

func TestGetSubAccountAllAPIKeys(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAllAPIKeys(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountAllAPIKeys must reject a nil request")
	_, err = e.GetSubAccountAllAPIKeys(t.Context(), &SubAccountAllAPIKeysRequest{Limit: 1})
	require.ErrorIs(t, err, errMemberIDRequired, "GetSubAccountAllAPIKeys must reject a missing sub UID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountAllAPIKeys(t.Context(), &SubAccountAllAPIKeysRequest{SubMemberID: "100000002", Limit: 1, Cursor: "24828208"})
	require.NoError(t, err, "GetSubAccountAllAPIKeys must not error")
	if mockTests {
		exp := &SubAccountAllAPIKeysResponse{
			Result: []SubAccountAPIKey{
				{
					ID:        "24828209",
					IPs:       []string{"*"},
					APIKey:    "mock-sub-api-key",
					Note:      "UTA",
					Status:    3,
					ExpiredAt: time.Date(2023, 12, 1, 2, 36, 6, 0, time.UTC),
					CreatedAt: time.Date(2023, 8, 25, 6, 42, 39, 0, time.UTC),
					Type:      1,
					Permissions: SubAccountAPIKeyPermissions{
						ContractTrade: []string{"Order", "Position"},
						Spot:          []string{"SpotTrade"},
						Wallet:        []string{"AccountTransfer", "SubMemberTransferList"},
						Options:       []string{"OptionsTrade"},
						Derivatives:   []string{"DerivativesTrade"},
						Exchange:      []string{"ExchangeHistory"},
						Earn:          []string{"Earn"},
						Affiliate:     []string{},
						BlockTrade:    []string{},
						NFT:           []string{},
						CopyTrading:   []string{},
					},
					Secret:      "******",
					ReadOnly:    true,
					DeadlineDay: 21,
					Flag:        "hmac",
				},
			},
			NextPageCursor: "24828209",
		}
		assert.Equal(t, exp, result, "GetSubAccountAllAPIKeys should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountAllAPIKeys should return a result")
}

func TestGetUIDWalletType(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUIDWalletType(t.Context(), "100000002", "100000003")
	require.NoError(t, err, "GetUIDWalletType must not error")
	if mockTests {
		exp := &UIDWalletTypeResponse{
			Accounts: []UIDWalletType{
				{
					UID:         "100000001",
					AccountType: []string{"UNIFIED", "FUND"},
				},
				{
					UID:         "100000002",
					AccountType: []string{"UNIFIED", "FUND"},
				},
				{
					UID:         "100000003",
					AccountType: []string{"FUND"},
				},
			},
		}
		assert.Equal(t, exp, result, "GetUIDWalletType should decode every field")
		return
	}
	assert.NotEmpty(t, result.Accounts, "GetUIDWalletType should return the master account")
}

func TestModifyMasterAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyMasterAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifyMasterAPIKey must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	readOnly := true
	result, err := e.ModifyMasterAPIKey(t.Context(), &ModifyMasterAPIKeyRequest{
		ReadOnly: &readOnly,
		Permissions: ModifyMasterAPIKeyPermissionsRequest{
			ContractTrade: []string{"Order", "Position"},
			Spot:          []string{"SpotTrade"},
			Wallet:        []string{"AccountTransfer", "SubMemberTransfer"},
			Options:       []string{"OptionsTrade"},
			Exchange:      []string{"ExchangeHistory"},
			Earn:          []string{"Earn"},
			Derivatives:   []string{"DerivativesTrade"},
			BlockTrade:    []string{},
		},
	})
	require.NoError(t, err, "ModifyMasterAPIKey must not error")
	if mockTests {
		exp := &ModifyMasterAPIKeyResponse{
			ID:       "13770661",
			Note:     "gct test",
			APIKey:   "mock-api-key",
			ReadOnly: true,
			Secret:   "******",
			Permissions: MasterAPIKeyPermissions{
				ContractTrade:     []string{"Order", "Position"},
				Spot:              []string{"SpotTrade"},
				Wallet:            []string{"AccountTransfer", "SubMemberTransfer"},
				Options:           []string{"OptionsTrade"},
				Derivatives:       []string{"DerivativesTrade"},
				BlockTrade:        []string{},
				Exchange:          []string{"ExchangeHistory"},
				Earn:              []string{"Earn"},
				Affiliate:         []string{},
				FiatP2P:           []string{"FiatP2POrder"},
				FiatBitPay:        []string{"FaitPayOrder"},
				FiatConvertBroker: []string{"FiatConvertBrokerOrder"},
				NFT:               []string{},
				CopyTrading:       []string{},
			},
			IPs: []string{"192.0.2.1"},
		}
		assert.Equal(t, exp, result, "ModifyMasterAPIKey should decode every field")
		return
	}
	assert.True(t, result.ReadOnly.Bool(), "ModifyMasterAPIKey should make the key read only")
}

func TestModifySubAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.ModifySubAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifySubAPIKey must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	readOnly, readWrite := true, false
	for _, tc := range []struct {
		name string
		req  *ModifySubAPIKeyRequest
		exp  *ModifySubAPIKeyResponse
	}{
		{
			name: "read write",
			req: &ModifySubAPIKeyRequest{
				APIKey:   "mock-sub-api-key",
				ReadOnly: &readWrite,
				IPs:      []string{"192.0.2.3"},
				Permissions: ModifySubAPIKeyPermissionsRequest{
					ContractTrade: []string{},
					Spot:          []string{"SpotTrade"},
					Wallet:        []string{"AccountTransfer"},
				},
			},
			exp: &ModifySubAPIKeyResponse{
				ID:     "16651472",
				Note:   "gct test",
				APIKey: "mock-sub-api-key",
				Secret: "******",
				Permissions: SubAPIKeyPermissions{
					ContractTrade:     []string{},
					Spot:              []string{"SpotTrade"},
					Wallet:            []string{"AccountTransfer"},
					Options:           []string{"OptionsTrade"},
					Derivatives:       []string{"DerivativesTrade"},
					Exchange:          []string{"ExchangeHistory"},
					Earn:              []string{"Earn"},
					BlockTrade:        []string{},
					Affiliate:         []string{},
					FiatP2P:           []string{},
					FiatConvertBroker: []string{},
					NFT:               []string{},
					CopyTrading:       []string{},
				},
				IPs: []string{"192.0.2.3"},
			},
		},
		{
			name: "read only",
			req:  &ModifySubAPIKeyRequest{APIKey: "mock-sub-api-key", ReadOnly: &readOnly},
			exp: &ModifySubAPIKeyResponse{
				ID:       "16651472",
				Note:     "gct test",
				APIKey:   "mock-sub-api-key",
				ReadOnly: true,
				Permissions: SubAPIKeyPermissions{
					ContractTrade:     []string{},
					Spot:              []string{"SpotTrade"},
					Wallet:            []string{"AccountTransfer"},
					Options:           []string{"OptionsTrade"},
					Derivatives:       []string{"DerivativesTrade"},
					Exchange:          []string{"ExchangeHistory"},
					Earn:              []string{"Earn"},
					BlockTrade:        []string{},
					Affiliate:         []string{},
					FiatP2P:           []string{},
					FiatConvertBroker: []string{},
					NFT:               []string{},
					CopyTrading:       []string{},
				},
				IPs: []string{"192.0.2.3"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ModifySubAPIKey(t.Context(), tc.req)
			require.NoError(t, err, "ModifySubAPIKey must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ModifySubAPIKey should decode every field")
				return
			}
			assert.Equal(t, *tc.req.ReadOnly, result.ReadOnly.Bool(), "ModifySubAPIKey should set the key's read only setting")
		})
	}
}

func TestAPIKeyPermissionsRequestBody(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		body map[string]any
		exp  map[string]any
	}{
		{
			name: "create sub UID API key",
			body: (&CreateSubUIDAPIKeyPermissionsRequest{
				ContractTrade: []string{"Order", "Position"},
				Spot:          []string{"SpotTrade"},
				Options:       []string{"OptionsTrade"},
				Wallet:        []string{"AccountTransfer", "SubMemberTransferList"},
				Exchange:      []string{"ExchangeHistory"},
				Earn:          []string{},
			}).body(),
			exp: map[string]any{
				"ContractTrade": []string{"Order", "Position"},
				"Spot":          []string{"SpotTrade"},
				"Options":       []string{"OptionsTrade"},
				"Wallet":        []string{"AccountTransfer", "SubMemberTransferList"},
				"Exchange":      []string{"ExchangeHistory"},
				"Earn":          []string{},
			},
		},
		{
			name: "modify master API key",
			body: (&ModifyMasterAPIKeyPermissionsRequest{
				ContractTrade:     []string{"Order", "Position"},
				Spot:              []string{"SpotTrade"},
				Wallet:            []string{"AccountTransfer", "SubMemberTransfer"},
				Options:           []string{"OptionsTrade"},
				Exchange:          []string{"ExchangeHistory"},
				Earn:              []string{"Earn"},
				FiatP2P:           []string{"FiatP2POrder", "Advertising"},
				FiatBitPay:        []string{"FaitPayOrder"},
				FiatConvertBroker: []string{"FiatConvertBrokerOrder"},
				BitCard:           []string{"BitCard"},
				ByXPost:           []string{"ByXPost"},
				Affiliate:         []string{"Affiliate"},
				Derivatives:       []string{"DerivativesTrade"},
				BlockTrade:        []string{},
			}).body(),
			exp: map[string]any{
				"ContractTrade":     []string{"Order", "Position"},
				"Spot":              []string{"SpotTrade"},
				"Wallet":            []string{"AccountTransfer", "SubMemberTransfer"},
				"Options":           []string{"OptionsTrade"},
				"Exchange":          []string{"ExchangeHistory"},
				"Earn":              []string{"Earn"},
				"FiatP2P":           []string{"FiatP2POrder", "Advertising"},
				"FiatBitPay":        []string{"FaitPayOrder"},
				"FiatConvertBroker": []string{"FiatConvertBrokerOrder"},
				"BitCard":           []string{"BitCard"},
				"ByXPost":           []string{"ByXPost"},
				"Affiliate":         []string{"Affiliate"},
				"Derivatives":       []string{"DerivativesTrade"},
				"BlockTrade":        []string{},
			},
		},
		{
			name: "modify sub API key",
			body: (&ModifySubAPIKeyPermissionsRequest{
				ContractTrade: []string{"Order", "Position"},
				Spot:          []string{"SpotTrade"},
				Wallet:        []string{"AccountTransfer", "SubMemberTransferList"},
				Options:       []string{"OptionsTrade"},
				Derivatives:   []string{"DerivativesTrade"},
				Exchange:      []string{"ExchangeHistory"},
				Earn:          []string{},
			}).body(),
			exp: map[string]any{
				"ContractTrade": []string{"Order", "Position"},
				"Spot":          []string{"SpotTrade"},
				"Wallet":        []string{"AccountTransfer", "SubMemberTransferList"},
				"Options":       []string{"OptionsTrade"},
				"Derivatives":   []string{"DerivativesTrade"},
				"Exchange":      []string{"ExchangeHistory"},
				"Earn":          []string{},
			},
		},
		{
			name: "nil lists are not sent",
			body: (&ModifySubAPIKeyPermissionsRequest{}).body(),
			exp:  map[string]any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, tc.body, "body should send every permission list that is not nil")
		})
	}
}

func TestBoolToInt(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint64(1), boolToInt(true), "boolToInt should return 1 for true")
	assert.Equal(t, uint64(0), boolToInt(false), "boolToInt should return 0 for false")
}

func TestAddPermission(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		list []string
		exp  map[string]any
	}{
		{name: "nil list", exp: map[string]any{}},
		{name: "empty list", list: []string{}, exp: map[string]any{"Wallet": []string{}}},
		{name: "list", list: []string{"AccountTransfer", "SubMemberTransfer"}, exp: map[string]any{"Wallet": []string{"AccountTransfer", "SubMemberTransfer"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			permissions := map[string]any{}
			addPermission(permissions, "Wallet", tc.list)
			assert.Equal(t, tc.exp, permissions, "addPermission should add a list that is not nil, even an empty one")
		})
	}
}

func TestDeleteSubUID(t *testing.T) {
	t.Parallel()
	err := e.DeleteSubUID(t.Context(), "")
	require.ErrorIs(t, err, errMemberIDRequired, "DeleteSubUID must reject a missing sub UID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.DeleteSubUID(t.Context(), "100000002"), "DeleteSubUID must not error")
}

func TestDeleteMasterAPIKey(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.DeleteMasterAPIKey(t.Context()), "DeleteMasterAPIKey must not error")
}

func TestDeleteSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.DeleteSubAccountAPIKey(t.Context(), "mock-sub-api-key"), "DeleteSubAccountAPIKey must not error")
}

func TestGetFriendReferrals(t *testing.T) {
	t.Parallel()
	_, err := e.GetFriendReferrals(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFriendReferrals must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFriendReferrals(t.Context(), &FriendReferralsRequest{Status: "1", Size: 2, Cursor: "6867"})
	require.NoError(t, err, "GetFriendReferrals must not error")
	if mockTests {
		exp := &FriendReferralsResponse{
			NextCursor: "6863",
			Records: []FriendReferral{
				{
					ID:         "6866",
					InviteeUID: "100000011",
					Status:     1,
					CreatedAt:  types.Time(time.Unix(1681206247, 0)),
					UpdatedAt:  types.Time(time.Unix(1681306247, 0)),
				},
				{
					ID:         "6865",
					InviteeUID: "100000012",
					Status:     1,
					CreatedAt:  types.Time(time.Unix(1681192249, 0)),
					UpdatedAt:  types.Time(time.Unix(1681292248, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetFriendReferrals should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFriendReferrals should return a result")
}

func TestGetReferralCode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetReferralCode(t.Context())
	require.NoError(t, err, "GetReferralCode must not error")
	if mockTests {
		exp := &ReferralCodeResponse{
			ReferralCodes: []ReferralCode{
				{
					ReferralCode: "AFFIL001",
					ReferralLink: "https://www.bybit.com/en/invite/?ref=AFFIL001",
					Scene:        1,
				},
				{
					ReferralCode: "FRIEND01",
					ReferralLink: "https://www.bybit.com/ru-RU/invite/?ref=FRIEND01",
					Scene:        2,
				},
			},
		}
		assert.Equal(t, exp, result, "GetReferralCode should decode every field")
		return
	}
	assert.NotNil(t, result, "GetReferralCode should return a result")
}
