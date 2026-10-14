package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// recordedSubAccount is the sub-account the recorded sub-account fixtures return
var recordedSubAccount = &SubAccount{
	Remark:     "test sub account",
	LoginName:  "test_sub_account_001",
	Email:      "test@example.com",
	State:      1,
	Type:       1,
	UserID:     12345678,
	CreateTime: types.Time(time.Unix(1682899200, 0)),
}

// recordedSubAccountAPIKey is the API key the recorded sub-account key fixtures return
var recordedSubAccountAPIKey = &SubAccountAPIKey{
	UserID: 12345678,
	Mode:   1,
	Name:   "test_api_key",
	Permissions: []*SubAccountKeyPerm{
		{Name: "wallet"}, {Name: "spot"}, {Name: "futures"}, {Name: "delivery"}, {Name: "earn"}, {Name: "options"},
	},
	IPWhitelist: []string{},
	Key:         "test-api-key-001",
	State:       1,
	CreatedAt:   types.Time(time.Unix(1682899200, 0)),
	UpdatedAt:   types.Time(time.Unix(1682899200, 0)),
}

func TestListSubAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, subAccountType := range []int64{-1, 1} {
		result, err := e.ListSubAccounts(t.Context(), subAccountType)
		require.NoErrorf(t, err, "ListSubAccounts must not error for type %d", subAccountType)
		if mockTests {
			assert.Equalf(t, []*SubAccount{recordedSubAccount}, result, "ListSubAccounts should decode every field for type %d", subAccountType)
			continue
		}
		assert.NotNilf(t, result, "ListSubAccounts should return sub-accounts for type %d", subAccountType)
	}
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSubAccount must reject a nil request")

	_, err = e.CreateSubAccount(t.Context(), &CreateSubAccountRequest{})
	require.ErrorIs(t, err, errInvalidSubAccount, "CreateSubAccount must reject an empty login name")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSubAccount(t.Context(), &CreateSubAccountRequest{
		LoginName: "test_sub_account_001",
	})
	require.NoError(t, err, "CreateSubAccount must not error")
	if mockTests {
		assert.Equal(t, recordedSubAccount, result, "CreateSubAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateSubAccount should return the sub-account")
}

func TestGetSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccount(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidSubAccountUserID, "GetSubAccount must reject an empty user ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccount(t.Context(), 12345678)
	require.NoError(t, err, "GetSubAccount must not error")
	if mockTests {
		assert.Equal(t, recordedSubAccount, result, "GetSubAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccount should return the sub-account")
}

func TestListSubAccountAPIKeys(t *testing.T) {
	t.Parallel()
	_, err := e.ListSubAccountAPIKeys(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidSubAccountUserID, "ListSubAccountAPIKeys must reject an empty user ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ListSubAccountAPIKeys(t.Context(), 12345678)
	require.NoError(t, err, "ListSubAccountAPIKeys must not error")
	if mockTests {
		assert.Equal(t, []*SubAccountAPIKey{recordedSubAccountAPIKey}, result, "ListSubAccountAPIKeys should decode every field")
		return
	}
	assert.NotNil(t, result, "ListSubAccountAPIKeys should return keys")
}

func TestCreateSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubAccountAPIKey(t.Context(), 0, &SubAccountKeyRequest{})
	require.ErrorIs(t, err, errInvalidSubAccountUserID, "CreateSubAccountAPIKey must reject an empty user ID")

	_, err = e.CreateSubAccountAPIKey(t.Context(), 12345678, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSubAccountAPIKey must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSubAccountAPIKey(t.Context(), 12345678, &SubAccountKeyRequest{
		Name: "test_key",
	})
	require.NoError(t, err, "CreateSubAccountAPIKey must not error")
	if mockTests {
		exp := &SubAccountAPIKey{
			UserID:      12345678,
			Mode:        1,
			Name:        "test_key",
			Permissions: []*SubAccountKeyPerm{{Name: "wallet"}, {Name: "spot"}},
			IPWhitelist: []string{},
			Key:         "test-api-key-001",
			Secret:      "test-api-secret-001",
			State:       1,
			CreatedAt:   types.Time(time.Unix(1682899200, 0)),
			UpdatedAt:   types.Time(time.Unix(1682899200, 0)),
		}
		assert.Equal(t, exp, result, "CreateSubAccountAPIKey should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateSubAccountAPIKey should return the key")
}

func TestGetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAPIKey(t.Context(), 0, "test-api-key-001")
	require.ErrorIs(t, err, errInvalidSubAccountUserID, "GetSubAccountAPIKey must reject an empty user ID")

	_, err = e.GetSubAccountAPIKey(t.Context(), 12345678, "")
	require.ErrorIs(t, err, errMissingAPIKey, "GetSubAccountAPIKey must reject an empty key")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountAPIKey(t.Context(), 12345678, "test-api-key-001")
	require.NoError(t, err, "GetSubAccountAPIKey must not error")
	if mockTests {
		assert.Equal(t, recordedSubAccountAPIKey, result, "GetSubAccountAPIKey should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountAPIKey should return the key")
}

func TestUpdateSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	err := e.UpdateSubAccountAPIKey(t.Context(), 0, "test-api-key-001", &SubAccountKeyRequest{})
	require.ErrorIs(t, err, errInvalidSubAccountUserID, "UpdateSubAccountAPIKey must reject an empty user ID")

	err = e.UpdateSubAccountAPIKey(t.Context(), 12345678, "", &SubAccountKeyRequest{})
	require.ErrorIs(t, err, errMissingAPIKey, "UpdateSubAccountAPIKey must reject an empty key")

	err = e.UpdateSubAccountAPIKey(t.Context(), 12345678, "test-api-key-001", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateSubAccountAPIKey must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.UpdateSubAccountAPIKey(t.Context(), 12345678, "test-api-key-001", &SubAccountKeyRequest{
		Name: "updated_key",
		Permissions: []*SubAccountKeyPerm{
			{Name: "wallet", ReadOnly: true},
		},
	})
	require.NoError(t, err, "UpdateSubAccountAPIKey must not error")
}

func TestDeleteSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	err := e.DeleteSubAccountAPIKey(t.Context(), 0, "test-api-key-001")
	require.ErrorIs(t, err, errInvalidSubAccountUserID, "DeleteSubAccountAPIKey must reject an empty user ID")

	err = e.DeleteSubAccountAPIKey(t.Context(), 12345678, "")
	require.ErrorIs(t, err, errMissingAPIKey, "DeleteSubAccountAPIKey must reject an empty key")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.DeleteSubAccountAPIKey(t.Context(), 12345678, "test-api-key-001")
	require.NoError(t, err, "DeleteSubAccountAPIKey must not error")
}

func TestLockSubAccount(t *testing.T) {
	t.Parallel()
	err := e.LockSubAccount(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidSubAccountUserID, "LockSubAccount must reject an empty user ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.LockSubAccount(t.Context(), 12345678)
	require.NoError(t, err, "LockSubAccount must not error")
}

func TestUnlockSubAccount(t *testing.T) {
	t.Parallel()
	err := e.UnlockSubAccount(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidSubAccountUserID, "UnlockSubAccount must reject an empty user ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.UnlockSubAccount(t.Context(), 12345678)
	require.NoError(t, err, "UnlockSubAccount must not error")
}

func TestGetSubAccountMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountMode(t.Context())
	require.NoError(t, err, "GetSubAccountMode must not error")
	if mockTests {
		assert.Equal(t, []*SubAccountMode{{UserID: 12345678, IsUnified: true, Mode: "multi_currency"}}, result, "GetSubAccountMode should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountMode should return modes")
}
