package cryptodotcom

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestCreateWithdrawal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *WithdrawalRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &WithdrawalRequest{Amount: 1, Address: "a"}, err: currency.ErrCurrencyCodeEmpty},
		{req: &WithdrawalRequest{Currency: currency.BTC, Address: "a"}, err: errInvalidAmount},
		{req: &WithdrawalRequest{Currency: currency.BTC, Amount: -1, Address: "a"}, err: errInvalidAmount},
		{req: &WithdrawalRequest{Currency: currency.BTC, Amount: 1}, err: errAddressRequired},
	} {
		_, err := e.CreateWithdrawal(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "CreateWithdrawal must refuse %+v", tc.req)
	}

	ex, sent := newSignedRecorder(t, "private/create-withdrawal", `{"id":"1"}`)
	_, err := ex.CreateWithdrawal(t.Context(), &WithdrawalRequest{ClientWithdrawalID: "c1", Currency: currency.NewCode("xrp"), Amount: 25.5, Address: "rAddress", AddressTag: "1234567890", NetworkID: "XRP"})
	require.NoError(t, err, "CreateWithdrawal must not error with every param")
	_, err = ex.CreateWithdrawal(t.Context(), &WithdrawalRequest{Currency: currency.BTC, Amount: 0.00000001, Address: "bAddress"})
	require.NoError(t, err, "CreateWithdrawal must not error with only the required params")
	params := sent()
	require.Len(t, params, 2, "CreateWithdrawal must send two requests")
	assert.JSONEq(t, `{"client_wid":"c1","currency":"XRP","amount":"25.5","address":"rAddress","address_tag":"1234567890","network_id":"XRP"}`, params[0], "CreateWithdrawal should send every param, the currency in upper case")
	assert.JSONEq(t, `{"currency":"BTC","amount":"0.00000001","address":"bAddress"}`, params[1], "CreateWithdrawal should send only the required params when the others are unset")

	req := &WithdrawalRequest{ClientWithdrawalID: "my_withdrawal_002", Currency: currency.BTC, Amount: 1, Address: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf", NetworkID: "BTC"}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/create-withdrawal","code":0,"result":null}`).CreateWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateWithdrawal should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateWithdrawal should return the request's failure")
	bare, err := newResponseServerExchange(t, `{"id":-1,"method":"private/create-withdrawal","code":0,"result":{"id":2220,"amount":1,"fee":0.0004,"symbol":"BTC","address":"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf","client_wid":"my_withdrawal_002","create_time":1607063412000,"network_id":null}}`).CreateWithdrawal(t.Context(), req)
	require.NoError(t, err, "CreateWithdrawal must not error for a bare withdrawal ID")
	assert.Equal(t, &WithdrawalResponse{
		ID:                 "2220",
		ClientWithdrawalID: "my_withdrawal_002",
		Symbol:             currency.BTC,
		Amount:             preciseNumber("1"),
		Fee:                preciseNumber("0.0004"),
		Address:            "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf",
		CreateTime:         types.Time(time.UnixMilli(1607063412000)),
	}, bare, "CreateWithdrawal should decode a reply with a bare withdrawal ID")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		// Live, a small amount goes to GoCryptoTrader's donation address, which the API key must have whitelisted
		req = &WithdrawalRequest{Currency: currency.BTC, Amount: 0.001, Address: core.BitcoinDonationAddress, NetworkID: "BTC"}
	}
	result, err := e.CreateWithdrawal(t.Context(), req)
	require.NoError(t, err, "CreateWithdrawal must not error")
	if mockTests {
		exp := &WithdrawalResponse{
			ID:                 "2220",
			ClientWithdrawalID: "my_withdrawal_002",
			Symbol:             currency.BTC,
			Amount:             preciseNumber("1"),
			Fee:                preciseNumber("0.0004"),
			Address:            "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf",
			CreateTime:         types.Time(time.UnixMilli(1607063412000)),
			NetworkID:          "BTC",
		}
		assert.Equal(t, exp, result, "CreateWithdrawal should decode every field")
	} else {
		assert.NotEmpty(t, result.ID, "CreateWithdrawal should return the withdrawal's ID")
	}
}

func TestGetCurrencyNetworks(t *testing.T) {
	t.Parallel()
	ex, sent := newSignedRecorder(t, "private/get-currency-networks", `{"currency_map":{}}`)
	_, err := ex.GetCurrencyNetworks(t.Context())
	require.NoError(t, err, "GetCurrencyNetworks must not error")
	params := sent()
	require.Len(t, params, 1, "GetCurrencyNetworks must send one request")
	assert.JSONEq(t, `{}`, params[0], "GetCurrencyNetworks should send empty params")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-currency-networks","code":0,"result":null}`).GetCurrencyNetworks(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetCurrencyNetworks should error for a null result")
	_, err = newUnavailableServerExchange(t).GetCurrencyNetworks(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCurrencyNetworks should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetCurrencyNetworks(t.Context())
	require.NoError(t, err, "GetCurrencyNetworks must not error")
	if mockTests {
		exp := &CurrencyNetworksResponse{
			UpdateTime: types.Time(time.UnixMilli(1641151604000)),
			Currencies: map[currency.Code]WalletCurrency{
				currency.NewCode("AGLD"): {
					FullName: "Adventure Gold",
					Networks: []CurrencyNetwork{
						{NetworkID: "ETH", WithdrawalEnabled: true, MinimumWithdrawalAmount: preciseNumber("10.0"), DepositEnabled: true, ConfirmationsRequired: 12},
					},
				},
				currency.MATIC: {
					FullName:       "Polygon",
					DefaultNetwork: "ETH",
					Networks: []CurrencyNetwork{
						{NetworkID: "BNB", WithdrawalFee: preciseNumber("0.80000000"), WithdrawalEnabled: true, MinimumWithdrawalAmount: preciseNumber("1.6"), DepositEnabled: true},
						{NetworkID: "ETH", WithdrawalFee: preciseNumber("20.00000000"), WithdrawalEnabled: true, MinimumWithdrawalAmount: preciseNumber("40.0"), DepositEnabled: true},
						{NetworkID: "MATIC", WithdrawalFee: preciseNumber("0.08000000"), MinimumWithdrawalAmount: preciseNumber("0.16"), DepositEnabled: true},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetCurrencyNetworks should decode every field")
	} else {
		assert.NotEmpty(t, result.Currencies, "GetCurrencyNetworks should return each currency's networks")
	}
}

func TestGetDepositAddresses(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddresses(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddresses must refuse an empty currency")

	ex, sent := newSignedRecorder(t, "private/get-deposit-address", `{"deposit_address_list":[]}`)
	_, err = ex.GetDepositAddresses(t.Context(), currency.NewCode("cro"))
	require.NoError(t, err, "GetDepositAddresses must not error")
	params := sent()
	require.Len(t, params, 1, "GetDepositAddresses must send one request")
	assert.JSONEq(t, `{"currency":"CRO"}`, params[0], "GetDepositAddresses should send the currency in upper case")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-deposit-address","code":0,"result":null}`).GetDepositAddresses(t.Context(), currency.CRO)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetDepositAddresses should error for a null result")
	_, err = newUnavailableServerExchange(t).GetDepositAddresses(t.Context(), currency.CRO)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDepositAddresses should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetDepositAddresses(t.Context(), currency.CRO)
	require.NoError(t, err, "GetDepositAddresses must not error")
	if mockTests {
		exp := []DepositAddressResponse{
			{
				ID:         "12345",
				Currency:   currency.CRO,
				Network:    "CRO",
				Address:    "cro1q9vfn0fjtyfvx3h8k2rwwgvxnz9k4j6m3y5pth",
				CreateTime: types.Time(time.UnixMilli(1615886328000)),
				Status:     DepositAddressStatusActive,
			},
			{
				ID:         "12346",
				Currency:   currency.CRO,
				Network:    "ETH",
				Address:    "0x1234567890abcdef1234567890abcdef12345678",
				CreateTime: types.Time(time.UnixMilli(1615886332000)),
				Status:     DepositAddressStatusInactive,
			},
		}
		assert.Equal(t, exp, result, "GetDepositAddresses should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].Address, "GetDepositAddresses should return address %d", i)
		}
	}
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *WalletHistoryRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &WalletHistoryRequest{StartTime: time.UnixMilli(1607100000000), EndTime: time.UnixMilli(1607000000000)}, err: common.ErrStartAfterEnd},
		{req: &WalletHistoryRequest{StartTime: time.UnixMilli(-1), EndTime: time.UnixMilli(1607000000000)}, err: errTimeBeforeUnixEpoch},
		{req: &WalletHistoryRequest{EndTime: time.UnixMilli(-1)}, err: errTimeBeforeUnixEpoch},
		{req: &WalletHistoryRequest{PageSize: maxWalletHistoryPageSize + 1}, err: errInvalidCount},
	} {
		_, err := e.GetDepositHistory(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetDepositHistory must refuse %+v", tc.req)
	}

	ex, sent := newSignedRecorder(t, "private/get-deposit-history", `{"deposit_list":[]}`)
	_, err := ex.GetDepositHistory(t.Context(), &WalletHistoryRequest{Currency: currency.NewCode("xrp"), StartTime: time.UnixMilli(1587846300000), EndTime: time.UnixMilli(1587846358253), PageSize: 2, Page: 1, Status: DepositStatusNotArrived})
	require.NoError(t, err, "GetDepositHistory must not error with every param")
	_, err = ex.GetDepositHistory(t.Context(), &WalletHistoryRequest{})
	require.NoError(t, err, "GetDepositHistory must not error without params")
	params := sent()
	require.Len(t, params, 2, "GetDepositHistory must send two requests")
	assert.JSONEq(t, `{"currency":"XRP","start_ts":"1587846300000","end_ts":"1587846358253","page_size":"2","page":"1","status":"0"}`, params[0], "GetDepositHistory should send every param, the not arrived status among them")
	assert.JSONEq(t, `{}`, params[1], "GetDepositHistory should send no params when none are set")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-deposit-history","code":0,"result":null}`).GetDepositHistory(t.Context(), &WalletHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetDepositHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetDepositHistory(t.Context(), &WalletHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDepositHistory should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetDepositHistory(t.Context(), &WalletHistoryRequest{Currency: currency.XRP, StartTime: time.UnixMilli(1607000000000), EndTime: time.UnixMilli(1607100000000), PageSize: 2, Status: DepositStatusArrived})
	require.NoError(t, err, "GetDepositHistory must not error")
	if mockTests {
		exp := []DepositResponse{{
			ID:            "2220",
			Currency:      currency.XRP,
			Amount:        preciseNumber("100"),
			Fee:           preciseNumber("1.0"),
			Address:       "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf?1234567890",
			CreateTime:    types.Time(time.UnixMilli(1607063412000)),
			UpdateTime:    types.Time(time.UnixMilli(1607063460000)),
			Status:        DepositStatusArrived,
			TransactionID: "C53ECF838647FA5A4C780377025FEC7999AB4182590510CA461444B207AB74A9",
		}}
		assert.Equal(t, exp, result, "GetDepositHistory should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].ID, "GetDepositHistory should return deposit %d's ID", i)
		}
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *WalletHistoryRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &WalletHistoryRequest{StartTime: time.UnixMilli(1607100000000), EndTime: time.UnixMilli(1607000000000)}, err: common.ErrStartAfterEnd},
		{req: &WalletHistoryRequest{StartTime: time.UnixMilli(-1), EndTime: time.UnixMilli(1607000000000)}, err: errTimeBeforeUnixEpoch},
		{req: &WalletHistoryRequest{EndTime: time.UnixMilli(-1)}, err: errTimeBeforeUnixEpoch},
	} {
		_, err := e.GetWithdrawalHistory(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetWithdrawalHistory must refuse %+v", tc.req)
	}

	ex, sent := newSignedRecorder(t, "private/get-withdrawal-history", `{"withdrawal_list":[]}`)
	_, err := ex.GetWithdrawalHistory(t.Context(), &WalletHistoryRequest{Currency: currency.NewCode("xrp"), StartTime: time.UnixMilli(1587846300000), EndTime: time.UnixMilli(1587846358253), PageSize: 2, Page: 1, Status: WithdrawalStatusPending})
	require.NoError(t, err, "GetWithdrawalHistory must not error with every param")
	_, err = ex.GetWithdrawalHistory(t.Context(), &WalletHistoryRequest{})
	require.NoError(t, err, "GetWithdrawalHistory must not error without params")
	params := sent()
	require.Len(t, params, 2, "GetWithdrawalHistory must send two requests")
	assert.JSONEq(t, `{"currency":"XRP","start_ts":"1587846300000","end_ts":"1587846358253","page_size":"2","page":"1","status":"0"}`, params[0], "GetWithdrawalHistory should send every param, the pending status among them")
	assert.JSONEq(t, `{}`, params[1], "GetWithdrawalHistory should send no params when none are set")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-withdrawal-history","code":0,"result":null}`).GetWithdrawalHistory(t.Context(), &WalletHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetWithdrawalHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetWithdrawalHistory(t.Context(), &WalletHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetWithdrawalHistory should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetWithdrawalHistory(t.Context(), &WalletHistoryRequest{Currency: currency.XRP, StartTime: time.UnixMilli(1607000000000), EndTime: time.UnixMilli(1607100000000), PageSize: 2, Status: WithdrawalStatusProcessing})
	require.NoError(t, err, "GetWithdrawalHistory must not error")
	if mockTests {
		exp := []WithdrawalResponse{{
			ID:                 "2220",
			ClientWithdrawalID: "my_withdrawal_002",
			Currency:           currency.XRP,
			Amount:             preciseNumber("100"),
			Fee:                preciseNumber("1.0"),
			Address:            "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf?1234567890",
			CreateTime:         types.Time(time.UnixMilli(1607063412000)),
			UpdateTime:         types.Time(time.UnixMilli(1607063460000)),
			Status:             WithdrawalStatusProcessing,
		}}
		assert.Equal(t, exp, result, "GetWithdrawalHistory should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].ID, "GetWithdrawalHistory should return withdrawal %d's ID", i)
		}
	}
}
