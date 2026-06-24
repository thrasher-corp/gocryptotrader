package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestListSubAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.ListSubAccounts(t.Context(), -1)
	require.NoError(t, err)

	_, err = e.ListSubAccounts(t.Context(), 1)
	require.NoError(t, err)
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubAccount(t.Context(), &CreateSubAccountRequest{})
	require.ErrorIs(t, err, errInvalidSubAccount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateSubAccount(t.Context(), &CreateSubAccountRequest{
		LoginName: "test_sub_account_001",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccount(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetSubAccount(t.Context(), 12345678)
	require.NoError(t, err)
}

func TestListSubAccountAPIKeys(t *testing.T) {
	t.Parallel()
	_, err := e.ListSubAccountAPIKeys(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ListSubAccountAPIKeys(t.Context(), 12345678)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestCreateSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubAccountAPIKey(t.Context(), 0, &SubAccountKeyRequest{})
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateSubAccountAPIKey(t.Context(), 12345678, &SubAccountKeyRequest{
		Name: "test_key",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAPIKey(t.Context(), 0, "test-api-key-001")
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	_, err = e.GetSubAccountAPIKey(t.Context(), 12345678, "")
	require.ErrorIs(t, err, errMissingAPIKey)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetSubAccountAPIKey(t.Context(), 12345678, "test-api-key-001")
	require.NoError(t, err)
}

func TestUpdateSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	err := e.UpdateSubAccountAPIKey(t.Context(), 0, "test-api-key-001", &SubAccountKeyRequest{})
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	err = e.UpdateSubAccountAPIKey(t.Context(), 12345678, "", &SubAccountKeyRequest{})
	require.ErrorIs(t, err, errMissingAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.UpdateSubAccountAPIKey(t.Context(), 12345678, "test-api-key-001", &SubAccountKeyRequest{
		Name: "updated_key",
		Permissions: []*SubAccountKeyPerm{
			{Name: "wallet", ReadOnly: true},
		},
	})
	require.NoError(t, err)
}

func TestDeleteSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	err := e.DeleteSubAccountAPIKey(t.Context(), 0, "test-api-key-001")
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	err = e.DeleteSubAccountAPIKey(t.Context(), 12345678, "")
	require.ErrorIs(t, err, errMissingAPIKey)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.DeleteSubAccountAPIKey(t.Context(), 12345678, "test-api-key-001")
	require.NoError(t, err)
}

func TestLockSubAccount(t *testing.T) {
	t.Parallel()
	err := e.LockSubAccount(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.LockSubAccount(t.Context(), 12345678)
	require.NoError(t, err)
}

func TestUnlockSubAccount(t *testing.T) {
	t.Parallel()
	err := e.UnlockSubAccount(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.UnlockSubAccount(t.Context(), 12345678)
	require.NoError(t, err)
}

func TestGetSubAccountMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountMode(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}
