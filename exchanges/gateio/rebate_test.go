package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetAgencyTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAgencyTransactionHistory(t.Context(), &RebateTransactionHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAgencyTransactionHistory(t.Context(), &RebateTransactionHistoryRequest{CurrencyPair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), From: startTime, To: endTime, Limit: 10})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAgencyCommissionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAgencyCommissionHistory(t.Context(), &RebateCommissionHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAgencyCommissionHistory(t.Context(), &RebateCommissionHistoryRequest{Currency: currency.USDT, CommissionType: 1, From: startTime, To: endTime, Limit: 10})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPartnerTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPartnerTransactionHistory(t.Context(), &RebateTransactionHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPartnerTransactionHistory(t.Context(), &RebateTransactionHistoryRequest{From: startTime, To: endTime, Limit: 10})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPartnerCommissionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPartnerCommissionHistory(t.Context(), &RebateCommissionHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPartnerCommissionHistory(t.Context(), &RebateCommissionHistoryRequest{Currency: currency.USDT, UserID: 10100011213, From: startTime, To: endTime, Limit: 10})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPartnerSubordinateList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPartnerSubordinateList(t.Context(), &PartnerSubordinateListRequest{UserID: 12312312, Limit: 10, Offset: 100})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerCommissionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBrokerCommissionHistory(t.Context(), &RebateBrokerHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerCommissionHistory(t.Context(), &RebateBrokerHistoryRequest{UserID: 12312312, From: startTime, To: endTime, Limit: 1, Offset: 100})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBrokerTransactionHistory(t.Context(), &RebateBrokerHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerTransactionHistory(t.Context(), &RebateBrokerHistoryRequest{UserID: 12312312, From: startTime, To: endTime, Limit: 1, Offset: 100})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserRebateInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetUserRebateInformation(t.Context())
	require.NoError(t, err)
}

func TestGetUserSubordinateRelationship(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserSubordinateRelationship(t.Context(), nil)
	require.ErrorIs(t, err, errUserIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetUserSubordinateRelationship(t.Context(), []string{"12342", "21312312312"})
	require.NoError(t, err)
}
