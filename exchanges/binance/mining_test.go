package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetAccountList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountList(t.Context(), "", "sams")
	require.ErrorIs(t, err, errTransferAlgorithmRequired, "GetAccountList must reject an empty algorithm")
	_, err = e.GetAccountList(t.Context(), "sha256", "")
	require.ErrorIs(t, err, errUsernameRequired, "GetAccountList must reject an empty mining account")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountList(t.Context(), "sha256", "sams")
	require.NoError(t, err, "GetAccountList must not error")
	if mockTests {
		exp := []MiningAccountHashrate{
			{
				Type:     "H_hashrate",
				UserName: "sams",
				List: []MiningHashrate{
					{
						Time:     types.Time(time.Unix(1744106400, 0)),
						Hashrate: 1268999325463.58,
						Reject:   0.0021,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetAccountList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountList should return a result")
}

func TestAcquiringAlgorithm(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.AcquiringAlgorithm(t.Context())
	require.NoError(t, err, "AcquiringAlgorithm must not error")
	if mockTests {
		exp := []MiningAlgorithm{
			{
				AlgorithmName: "sha256",
				AlgorithmID:   1,
				Unit:          "h/s",
			},
			{
				AlgorithmName: "Ethash",
				AlgorithmID:   2,
				PoolIndex:     1,
				Unit:          "h/s",
			},
		}
		assert.Equal(t, exp, result, "AcquiringAlgorithm should decode every field")
		return
	}
	assert.NotEmpty(t, result, "AcquiringAlgorithm should return algorithms")
}

func TestGetCoinNames(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCoinNames(t.Context())
	require.NoError(t, err, "GetCoinNames must not error")
	if mockTests {
		exp := []MiningCoin{
			{
				CoinName:      currency.BTC,
				CoinID:        1,
				AlgorithmID:   1,
				AlgorithmName: "sha256",
			},
			{
				CoinName:      currency.NewCode("BCH"),
				CoinID:        2,
				PoolIndex:     1,
				AlgorithmID:   1,
				AlgorithmName: "sha256",
			},
		}
		assert.Equal(t, exp, result, "GetCoinNames should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinNames should return coins")
}

func TestCancelHashrateResaleConfiguration(t *testing.T) {
	t.Parallel()
	_, err := e.CancelHashrateResaleConfiguration(t.Context(), 0, "sams")
	require.ErrorIs(t, err, errConfigIDRequired, "CancelHashrateResaleConfiguration must reject an empty config ID")
	_, err = e.CancelHashrateResaleConfiguration(t.Context(), 189, "")
	require.ErrorIs(t, err, errUsernameRequired, "CancelHashrateResaleConfiguration must reject an empty mining account")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelHashrateResaleConfiguration(t.Context(), 189, "sams")
	require.NoError(t, err, "CancelHashrateResaleConfiguration must not error")
	assert.True(t, result, "CancelHashrateResaleConfiguration should report the cancellation")
}

func TestGetEarningList(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarningList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEarningList must reject a nil request")
	startTime, endTime := getTime()
	arg := &MiningPaymentRequest{UserName: "sams", Coin: currency.BTC, StartDate: endTime, EndDate: startTime, PageIndex: 1, PageSize: 100}
	_, err = e.GetEarningList(t.Context(), arg)
	require.ErrorIs(t, err, errTransferAlgorithmRequired, "GetEarningList must reject an empty algorithm")
	arg.Algorithm, arg.UserName = "sha256", ""
	_, err = e.GetEarningList(t.Context(), arg)
	require.ErrorIs(t, err, errUsernameRequired, "GetEarningList must reject an empty mining account")
	arg.UserName = "sams"
	_, err = e.GetEarningList(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEarningList must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartDate, arg.EndDate = startTime, endTime
	result, err := e.GetEarningList(t.Context(), arg)
	require.NoError(t, err, "GetEarningList must not error")
	if mockTests {
		exp := &MiningEarningsResponse{
			AccountProfits: []MiningEarning{
				{
					Time:           types.Time(time.Unix(1744156800, 0)),
					Type:           31,
					HashTransfer:   200000000000,
					TransferAmount: 0.02180958,
					DayHashRate:    129129903378244,
					ProfitAmount:   8.6083060304,
					CoinName:       currency.BTC,
					Status:         2,
				},
			},
			TotalNumber: 3,
			PageSize:    100,
		}
		assert.Equal(t, exp, result, "GetEarningList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetEarningList should return a result")
}

func TestGetExtraBonusList(t *testing.T) {
	t.Parallel()
	_, err := e.GetExtraBonusList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExtraBonusList must reject a nil request")
	startTime, endTime := getTime()
	arg := &MiningPaymentRequest{UserName: "sams", Coin: currency.BTC, StartDate: endTime, EndDate: startTime, PageIndex: 1, PageSize: 10}
	_, err = e.GetExtraBonusList(t.Context(), arg)
	require.ErrorIs(t, err, errTransferAlgorithmRequired, "GetExtraBonusList must reject an empty algorithm")
	arg.Algorithm, arg.UserName = "sha256", ""
	_, err = e.GetExtraBonusList(t.Context(), arg)
	require.ErrorIs(t, err, errUsernameRequired, "GetExtraBonusList must reject an empty mining account")
	arg.UserName = "sams"
	_, err = e.GetExtraBonusList(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetExtraBonusList must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartDate, arg.EndDate = startTime, endTime
	result, err := e.GetExtraBonusList(t.Context(), arg)
	require.NoError(t, err, "GetExtraBonusList must not error")
	if mockTests {
		exp := &MiningExtraBonusResponse{
			OtherProfits: []MiningExtraBonus{
				{
					Time:         types.Time(time.Unix(1744156800, 0)),
					CoinName:     currency.BTC,
					Type:         4,
					ProfitAmount: 0.0011859,
					Status:       2,
				},
			},
			TotalNumber: 3,
			PageSize:    10,
		}
		assert.Equal(t, exp, result, "GetExtraBonusList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetExtraBonusList should return a result")
}

func TestGetHashrateResaleDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetHashrateResaleDetail(t.Context(), 0, 10, 20)
	require.ErrorIs(t, err, errConfigIDRequired, "GetHashrateResaleDetail must reject an empty config ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetHashrateResaleDetail(t.Context(), 168, 10, 20)
	require.NoError(t, err, "GetHashrateResaleDetail must not error")
	if mockTests {
		exp := &HashrateResaleDetailResponse{
			ProfitTransferDetails: []HashrateResaleProfitTransfer{
				{
					PoolUsername:   "sams",
					ToPoolUsername: "S19pro",
					AlgorithmName:  "sha256",
					HashRate:       600000000000,
					Day:            types.Time(time.Date(2025, 4, 8, 0, 0, 0, 0, time.UTC)),
					Amount:         0.2256872,
					CoinName:       currency.BTC,
				},
			},
			TotalNumber: 8,
			PageSize:    20,
		}
		assert.Equal(t, exp, result, "GetHashrateResaleDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetHashrateResaleDetail should return a result")
}

func TestGetHashrateResaleList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetHashrateResaleList(t.Context(), 10, 20)
	require.NoError(t, err, "GetHashrateResaleList must not error")
	if mockTests {
		exp := &HashrateResaleListResponse{
			ConfigDetails: []HashrateResaleConfig{
				{
					ConfigID:       168,
					PoolUsername:   "sams",
					ToPoolUsername: "S19pro",
					AlgorithmName:  "sha256",
					HashRate:       600000000000,
					StartDay:       types.Time(time.Date(2025, 4, 8, 0, 0, 0, 0, time.UTC)),
					EndDay:         types.Time(time.Date(2025, 4, 9, 0, 0, 0, 0, time.UTC)),
					Status:         1,
					Type:           1,
				},
			},
			TotalNumber: 21,
			PageSize:    20,
		}
		assert.Equal(t, exp, result, "GetHashrateResaleList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetHashrateResaleList should return a result")
}

func TestRequestHashrateResale(t *testing.T) {
	t.Parallel()
	_, err := e.RequestHashrateResale(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestHashrateResale must reject a nil request")
	arg := &HashrateResaleRequest{Algorithm: "sha256", ToPoolUser: "S19pro", HashRate: 600000000000}
	_, err = e.RequestHashrateResale(t.Context(), arg)
	require.ErrorIs(t, err, errUsernameRequired, "RequestHashrateResale must reject an empty mining account")
	arg.UserName, arg.Algorithm = "sams", ""
	_, err = e.RequestHashrateResale(t.Context(), arg)
	require.ErrorIs(t, err, errTransferAlgorithmRequired, "RequestHashrateResale must reject an empty algorithm")
	arg.Algorithm = "sha256"
	_, err = e.RequestHashrateResale(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrDateUnset, "RequestHashrateResale must reject a missing window")
	startTime, endTime := getTime()
	arg.StartDate, arg.EndDate = endTime, startTime
	_, err = e.RequestHashrateResale(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "RequestHashrateResale must reject a reversed window")
	arg.StartDate, arg.EndDate, arg.ToPoolUser = startTime, endTime, ""
	_, err = e.RequestHashrateResale(t.Context(), arg)
	require.ErrorIs(t, err, errAccountRequired, "RequestHashrateResale must reject an empty receiving account")
	arg.ToPoolUser, arg.HashRate = "S19pro", 0
	_, err = e.RequestHashrateResale(t.Context(), arg)
	require.ErrorIs(t, err, errHashRateRequired, "RequestHashrateResale must reject an empty hashrate")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.HashRate = 600000000000
	result, err := e.RequestHashrateResale(t.Context(), arg)
	require.NoError(t, err, "RequestHashrateResale must not error")
	if mockTests {
		assert.Equal(t, uint64(171), result, "RequestHashrateResale should return the config ID")
		return
	}
	assert.NotZero(t, result, "RequestHashrateResale should return a config ID")
}

func TestGetMiningAccountEarning(t *testing.T) {
	t.Parallel()
	_, err := e.GetMiningAccountEarning(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMiningAccountEarning must reject a nil request")
	startTime, endTime := getTime()
	arg := &MiningAccountEarningRequest{StartDate: endTime, EndDate: startTime, PageIndex: 1, PageSize: 100}
	_, err = e.GetMiningAccountEarning(t.Context(), arg)
	require.ErrorIs(t, err, errTransferAlgorithmRequired, "GetMiningAccountEarning must reject an empty algorithm")
	arg.Algorithm = "sha256"
	_, err = e.GetMiningAccountEarning(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMiningAccountEarning must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartDate, arg.EndDate = startTime, endTime
	result, err := e.GetMiningAccountEarning(t.Context(), arg)
	require.NoError(t, err, "GetMiningAccountEarning must not error")
	if mockTests {
		exp := &MiningAccountEarningResponse{
			AccountProfits: []MiningAccountProfit{
				{
					Time:     types.Time(time.Unix(1744156800, 0)),
					CoinName: currency.BTC,
					Type:     2,
					PUID:     59985472,
					SubName:  "sams",
					Amount:   0.09186957,
				},
			},
			TotalNumber: 3,
			PageSize:    100,
		}
		assert.Equal(t, exp, result, "GetMiningAccountEarning should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMiningAccountEarning should return a result")
}

func TestGetDetailMinerList(t *testing.T) {
	t.Parallel()
	_, err := e.GetDetailMinerList(t.Context(), "", "sams", "bhdc1.16A10404B")
	require.ErrorIs(t, err, errTransferAlgorithmRequired, "GetDetailMinerList must reject an empty algorithm")
	_, err = e.GetDetailMinerList(t.Context(), "sha256", "", "bhdc1.16A10404B")
	require.ErrorIs(t, err, errUsernameRequired, "GetDetailMinerList must reject an empty mining account")
	_, err = e.GetDetailMinerList(t.Context(), "sha256", "sams", "")
	require.ErrorIs(t, err, errNameRequired, "GetDetailMinerList must reject an empty miner name")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDetailMinerList(t.Context(), "sha256", "sams", "bhdc1.16A10404B")
	require.NoError(t, err, "GetDetailMinerList must not error")
	if mockTests {
		exp := []MinerDetail{
			{
				WorkerName: "bhdc1.16A10404B",
				Type:       "H_hashrate",
				HashrateDatas: []MinerHashrate{
					{
						Time:     types.Time(time.Unix(1744106400, 0)),
						Hashrate: 1268999325463,
						Reject:   1,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetDetailMinerList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDetailMinerList should return a result")
}

func TestGetMinersList(t *testing.T) {
	t.Parallel()
	_, err := e.GetMinersList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMinersList must reject a nil request")
	arg := &MinerListRequest{UserName: "sams", Descending: true, SortColumn: 2, WorkerStatus: 1}
	_, err = e.GetMinersList(t.Context(), arg)
	require.ErrorIs(t, err, errTransferAlgorithmRequired, "GetMinersList must reject an empty algorithm")
	arg.Algorithm, arg.UserName = "sha256", ""
	_, err = e.GetMinersList(t.Context(), arg)
	require.ErrorIs(t, err, errUsernameRequired, "GetMinersList must reject an empty mining account")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.UserName = "sams"
	result, err := e.GetMinersList(t.Context(), arg)
	require.NoError(t, err, "GetMinersList must not error")
	if mockTests {
		exp := &MinerListResponse{
			WorkerDatas: []MinerWorker{
				{
					WorkerID:      "1420554439452400131",
					WorkerName:    "2X73",
					Status:        1,
					HashRate:      13421772800000,
					DayHashRate:   12697781298013.66,
					RejectRate:    0.01,
					LastShareTime: types.Time(time.Unix(1744190000, 0)),
				},
			},
			TotalNumber: 18530,
			PageSize:    20,
		}
		assert.Equal(t, exp, result, "GetMinersList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMinersList should return a result")
}

func TestStatisticsList(t *testing.T) {
	t.Parallel()
	_, err := e.StatisticsList(t.Context(), "", "sams")
	require.ErrorIs(t, err, errTransferAlgorithmRequired, "StatisticsList must reject an empty algorithm")
	_, err = e.StatisticsList(t.Context(), "sha256", "")
	require.ErrorIs(t, err, errUsernameRequired, "StatisticsList must reject an empty mining account")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.StatisticsList(t.Context(), "sha256", "sams")
	require.NoError(t, err, "StatisticsList must not error")
	if mockTests {
		exp := &MiningStatisticsResponse{
			FifteenMinuteHashRate: 457835490067496409,
			DayHashRate:           214289268068874127.65,
			ValidNumber:           3,
			InvalidNumber:         17562,
			ProfitToday: map[currency.Code]types.Number{
				currency.NewCode("BCH"): 106.61586001,
				currency.BTC:            0.00314332,
			},
			ProfitYesterday: map[currency.Code]types.Number{
				currency.NewCode("BCH"): 106.61586001,
				currency.BTC:            0.00314332,
			},
			UserName:  "sams",
			Unit:      "h/s",
			Algorithm: "sha256",
		}
		assert.Equal(t, exp, result, "StatisticsList should decode every field")
		return
	}
	assert.NotNil(t, result, "StatisticsList should return a result")
}
