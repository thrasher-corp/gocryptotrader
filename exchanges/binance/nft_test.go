package binance

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

func TestGetNFTTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetNFTTransactionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetNFTTransactionHistory must reject a nil request")
	startTime, endTime := getTime()
	arg := &NFTTransactionHistoryRequest{OrderType: 5, StartTime: endTime, EndTime: startTime, Limit: 10, Page: 40}
	_, err = e.GetNFTTransactionHistory(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "GetNFTTransactionHistory must reject an unknown order type")
	arg.OrderType = 1
	_, err = e.GetNFTTransactionHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetNFTTransactionHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetNFTTransactionHistory(t.Context(), arg)
	require.NoError(t, err, "GetNFTTransactionHistory must not error")
	if mockTests {
		exp := &NFTTransactionHistoryResponse{
			Total: 1,
			List: []NFTTransaction{
				{
					OrderNumber:   "1_470502070600699904",
					Tokens:        []NFTToken{{Network: "BSC", TokenID: "216000000496", ContractAddress: "MYSTERY_BOX0000087"}},
					TradeTime:     types.Time(time.UnixMilli(1744150000000)),
					TradeAmount:   19.6,
					TradeCurrency: currency.BNB,
				},
			},
		}
		assert.Equal(t, exp, result, "GetNFTTransactionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetNFTTransactionHistory should return a result")
}

func TestGetNFTDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetNFTDepositHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetNFTDepositHistory must reject a nil request")
	startTime, endTime := getTime()
	arg := &NFTHistoryRequest{StartTime: endTime, EndTime: startTime, Limit: 10, Page: 40}
	_, err = e.GetNFTDepositHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetNFTDepositHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetNFTDepositHistory(t.Context(), arg)
	require.NoError(t, err, "GetNFTDepositHistory must not error")
	if mockTests {
		exp := &NFTDepositHistoryResponse{
			Total: 1,
			List: []NFTDeposit{
				{
					Network:         "ETH",
					TransactionID:   "0x5759dfe9983a4c7619bce9bc736bb6c26f804091753bf66fa91e7cd5cfeebafd",
					ContractAdrress: "0xe507c961ee127d4439977a61af39c34eafee0dc6",
					TokenID:         "10014",
					Timestamp:       types.Time(time.UnixMilli(1744150000000)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetNFTDepositHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetNFTDepositHistory should return a result")
}

func TestGetNFTWithdrawalHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetNFTWithdrawalHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetNFTWithdrawalHistory must reject a nil request")
	startTime, endTime := getTime()
	arg := &NFTHistoryRequest{StartTime: endTime, EndTime: startTime, Limit: 10, Page: 40}
	_, err = e.GetNFTWithdrawalHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetNFTWithdrawalHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetNFTWithdrawalHistory(t.Context(), arg)
	require.NoError(t, err, "GetNFTWithdrawalHistory must not error")
	if mockTests {
		exp := &NFTWithdrawalHistoryResponse{
			Total: 178,
			List: []NFTWithdrawal{
				{
					Network:         "ETH",
					TransactionID:   "0x2be5eed31d787fdb4880bc631c8e76bdfb6150e137f5cf1732e0416ea206f57f",
					ContractAdrress: "0xe507c961ee127d4439977a61af39c34eafee0dc6",
					TokenID:         "1000001247",
					Timestamp:       types.Time(time.UnixMilli(1744150000000)),
					Fee:             0.1,
					FeeAsset:        currency.ETH,
				},
			},
		}
		assert.Equal(t, exp, result, "GetNFTWithdrawalHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetNFTWithdrawalHistory should return a result")
}

func TestGetNFTAsset(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetNFTAsset(t.Context(), 10, 20)
	require.NoError(t, err, "GetNFTAsset must not error")
	if mockTests {
		exp := &NFTAssetsResponse{
			Total: 347,
			List:  []NFTToken{{Network: "BSC", ContractAddress: "REGULAR11234567891779", TokenID: "100900000017"}},
		}
		assert.Equal(t, exp, result, "GetNFTAsset should decode every field")
		return
	}
	assert.NotNil(t, result, "GetNFTAsset should return a result")
}
