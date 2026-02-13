package engine

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjob"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

func TestSetupDataHistoryManager(t *testing.T) {
	t.Parallel()
	_, err := SetupDataHistoryManager(nil, nil, nil)
	assert.ErrorIs(t, err, errNilExchangeManager)

	_, err = SetupDataHistoryManager(NewExchangeManager(), nil, nil)
	assert.ErrorIs(t, err, errNilDatabaseConnectionManager)

	_, err = SetupDataHistoryManager(NewExchangeManager(), &DatabaseConnectionManager{}, nil)
	assert.ErrorIs(t, err, errNilConfig)

	_, err = SetupDataHistoryManager(NewExchangeManager(), &DatabaseConnectionManager{}, &config.DataHistoryManager{})
	assert.ErrorIs(t, err, database.ErrNilInstance)

	dbInst := &database.Instance{}
	err = dbInst.SetConfig(&database.Config{Enabled: true})
	assert.NoError(t, err)

	dbInst.SetConnected(true)
	dbCM := &DatabaseConnectionManager{
		dbConn:  dbInst,
		started: 1,
	}
	err = dbInst.SetSQLiteConnection(&sql.DB{})
	assert.NoError(t, err)

	m, err := SetupDataHistoryManager(NewExchangeManager(), dbCM, &config.DataHistoryManager{})
	assert.NoError(t, err)

	if m == nil {
		t.Fatal("expected manager")
	}
}

func TestDataHistoryManagerIsRunning(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	m.started = 0
	if m.IsRunning() {
		t.Error("expected false")
	}
	m.started = 1
	if !m.IsRunning() {
		t.Error("expected true")
	}
	m = nil
	if m.IsRunning() {
		t.Error("expected false")
	}
}

func TestDataHistoryManagerStart(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	m.started = 0
	err := m.Start()
	assert.NoError(t, err)

	err = m.Start()
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted)

	m = nil
	err = m.Start()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestDataHistoryManagerStop(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	m.shutdown = make(chan struct{})
	err := m.Stop()
	assert.NoError(t, err)

	err = m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	err = m.Stop()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestUpsertJob(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	err := m.UpsertJob(nil, false)
	assert.ErrorIs(t, err, errNilJob)

	dhj := &DataHistoryJob{}
	err = m.UpsertJob(dhj, false)
	assert.ErrorIs(t, err, errNicknameUnset)

	dhj.Nickname = "test1337"
	err = m.UpsertJob(dhj, false)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	dhj.Asset = asset.Spot
	err = m.UpsertJob(dhj, false)
	assert.ErrorIs(t, err, errCurrencyPairUnset)

	dhj.Exchange = strings.ToLower(testExchange)
	dhj.Pair = currency.NewPair(currency.BTC, currency.DOGE)
	err = m.UpsertJob(dhj, false)
	assert.ErrorIs(t, err, errCurrencyNotEnabled)

	dhj.Pair = currency.NewBTCUSD()
	err = m.UpsertJob(dhj, false)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	dhj.Interval = kline.OneHour
	err = m.UpsertJob(dhj, false)
	assert.ErrorIs(t, err, common.ErrDateUnset)

	dhj.StartDate = time.Now().Add(-time.Hour)
	dhj.EndDate = time.Now()
	err = m.UpsertJob(dhj, false)
	assert.NoError(t, err)

	err = m.UpsertJob(dhj, true)
	assert.ErrorIs(t, err, errNicknameInUse)

	newJob := &DataHistoryJob{
		Nickname:                 dhj.Nickname,
		Exchange:                 testExchange,
		Asset:                    asset.Spot,
		Pair:                     currency.NewBTCUSD(),
		StartDate:                startDate,
		EndDate:                  time.Now().Add(-time.Minute),
		Interval:                 kline.FifteenMin,
		RunBatchLimit:            1338,
		RequestSizeLimit:         1337,
		DataType:                 99,
		MaxRetryAttempts:         1337,
		OverwriteExistingData:    true,
		ConversionInterval:       3,
		DecimalPlaceComparison:   5,
		SecondaryExchangeSource:  testExchange,
		IssueTolerancePercentage: 3,
		ReplaceOnIssue:           true,
		PrerequisiteJobNickname:  "hellomoto",
	}
	err = m.UpsertJob(newJob, false)
	assert.ErrorIs(t, err, errInvalidDataHistoryDataType)

	newJob.DataType = dataHistoryTradeDataType
	err = m.UpsertJob(newJob, false)
	assert.NoError(t, err)
}

func TestSetJobStatus(t *testing.T) {
	t.Parallel()
	m, j := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestSetJobStatus",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	assert.NoError(t, err)

	err = m.SetJobStatus("", "", 0)
	assert.ErrorIs(t, err, errNicknameIDUnset)

	err = m.SetJobStatus("1337", "1337", 0)
	assert.ErrorIs(t, err, errOnlyNicknameOrID)

	err = m.SetJobStatus(dhj.Nickname, "", dataHistoryStatusRemoved)
	assert.NoError(t, err)

	err = m.SetJobStatus("", dhj.ID.String(), dataHistoryStatusActive)
	assert.ErrorIs(t, err, errBadStatus)

	j.Status = int64(dataHistoryStatusActive)
	err = m.SetJobStatus("", dhj.ID.String(), dataHistoryStatusPaused)
	assert.NoError(t, err)

	err = m.SetJobStatus("", dhj.ID.String(), dataHistoryStatusFailed)
	assert.ErrorIs(t, err, errBadStatus)

	dhj.Status = dataHistoryStatusPaused
	err = m.SetJobStatus(dhj.Nickname, "", dataHistoryStatusActive)
	assert.NoError(t, err)

	dhj.Status = dataHistoryStatusRemoved
	err = m.SetJobStatus(dhj.Nickname, "", dataHistoryStatusActive)
	assert.ErrorIs(t, err, errBadStatus)

	dhj.Status = dataHistoryStatusPaused
	err = m.SetJobStatus(dhj.Nickname, "", dataHistoryStatusRemoved)
	assert.NoError(t, err)

	atomic.StoreInt32(&m.started, 0)
	err = m.SetJobStatus("", dhj.ID.String(), 0)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	err = m.SetJobStatus("", dhj.ID.String(), 0)
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestGetByNickname(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestGetByNickname",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	assert.NoError(t, err)

	_, err = m.GetByNickname(dhj.Nickname, false)
	assert.NoError(t, err)

	_, err = m.GetByNickname(dhj.Nickname, true)
	assert.NoError(t, err)

	_, err = m.GetByNickname(dhj.Nickname, false)
	assert.NoError(t, err)

	atomic.StoreInt32(&m.started, 0)
	_, err = m.GetByNickname("test123", false)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	_, err = m.GetByNickname("test123", false)
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestGetByID(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestGetByID",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	assert.NoError(t, err)

	_, err = m.GetByID(dhj.ID)
	assert.NoError(t, err)

	_, err = m.GetByID(uuid.UUID{})
	assert.ErrorIs(t, err, errEmptyID)

	_, err = m.GetByID(dhj.ID)
	assert.NoError(t, err)

	atomic.StoreInt32(&m.started, 0)
	_, err = m.GetByID(dhj.ID)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	_, err = m.GetByID(dhj.ID)
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestRetrieveJobs(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestRetrieveJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	assert.NoError(t, err)

	jobs, err := m.retrieveJobs()
	assert.NoError(t, err)

	if len(jobs) != 1 {
		t.Error("expected job")
	}

	atomic.StoreInt32(&m.started, 0)
	_, err = m.retrieveJobs()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	_, err = m.retrieveJobs()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestGetActiveJobs(t *testing.T) {
	t.Parallel()
	m, j := createDHM(t)

	dhj := &DataHistoryJob{
		Nickname:  "TestGetActiveJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	assert.NoError(t, err)

	jobs, err := m.GetActiveJobs()
	assert.NoError(t, err)

	if len(jobs) != 1 {
		t.Error("expected 1 job")
	}

	j.Status = int64(dataHistoryStatusFailed)
	jobs, err = m.GetActiveJobs()
	assert.NoError(t, err)

	if len(jobs) != 0 {
		t.Error("expected 0 jobs")
	}

	atomic.StoreInt32(&m.started, 0)
	_, err = m.GetActiveJobs()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	_, err = m.GetActiveJobs()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestValidateJob(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	err := m.validateJob(nil)
	assert.ErrorIs(t, err, errNilJob)

	dhj := &DataHistoryJob{}
	err = m.validateJob(dhj)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	dhj.Asset = asset.Spot
	err = m.validateJob(dhj)
	assert.ErrorIs(t, err, errCurrencyPairUnset)

	dhj.Exchange = testExchange
	dhj.Pair = currency.NewPair(currency.BTC, currency.XRP)
	err = m.validateJob(dhj)
	assert.ErrorIs(t, err, errCurrencyNotEnabled)

	dhj.Pair = currency.NewBTCUSD()
	err = m.validateJob(dhj)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	dhj.Interval = kline.OneMin
	err = m.validateJob(dhj)
	assert.ErrorIs(t, err, common.ErrDateUnset)

	dhj.StartDate = time.Now().Add(time.Minute)
	dhj.EndDate = time.Now().Add(time.Hour)
	err = m.validateJob(dhj)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)

	dhj.StartDate = time.Now().Add(-time.Hour * 60)
	dhj.EndDate = time.Now().Add(-time.Minute)
	err = m.validateJob(dhj)
	assert.NoError(t, err)

	dhj.DataType = dataHistoryCandleValidationDataType
	dhj.Interval = kline.OneDay
	dhj.RequestSizeLimit = 999
	err = m.validateJob(dhj)
	assert.NoError(t, err)

	dhj.DataType = dataHistoryTradeDataType
	err = m.validateJob(dhj)
	assert.NoError(t, err)

	dhj.DataType = dataHistoryCandleValidationSecondarySourceType
	err = m.validateJob(dhj)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	dhj.SecondaryExchangeSource = "lol"
	dhj.Exchange = ""
	err = m.validateJob(dhj)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)
}

func TestGetAllJobStatusBetween(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)

	dhj := &DataHistoryJob{
		Nickname:  "TestGetActiveJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	assert.NoError(t, err)

	jobs, err := m.GetAllJobStatusBetween(time.Now().Add(-time.Minute*5), time.Now().Add(time.Minute))
	assert.NoError(t, err)

	if len(jobs) != 1 {
		t.Error("expected 1 job")
	}

	_, err = m.GetAllJobStatusBetween(time.Now().Add(-time.Hour), time.Now().Add(-time.Minute*30))
	assert.NoError(t, err)

	m.started = 0
	_, err = m.GetAllJobStatusBetween(time.Now().Add(-time.Hour), time.Now().Add(-time.Minute*30))
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	_, err = m.GetAllJobStatusBetween(time.Now().Add(-time.Hour), time.Now().Add(-time.Minute*30))
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestPrepareJobs(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	jobs, err := m.PrepareJobs()
	assert.NoError(t, err)

	if len(jobs) != 1 {
		t.Errorf("expected 1 job, received %v", len(jobs))
	}
	m.started = 0
	_, err = m.PrepareJobs()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	_, err = m.PrepareJobs()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestCompareJobsToData(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	tt := time.Now().Truncate(kline.OneHour.Duration())
	dhj := &DataHistoryJob{
		Nickname:           "TestGenerateJobSummary",
		Exchange:           testExchange,
		Asset:              asset.Spot,
		Pair:               currency.NewBTCUSD(),
		StartDate:          tt.Add(-time.Minute * 5),
		EndDate:            tt,
		Interval:           kline.OneMin,
		ConversionInterval: kline.FiveMin,
	}
	err := m.compareJobsToData(dhj)
	assert.NoError(t, err)

	dhj.DataType = dataHistoryTradeDataType
	err = m.compareJobsToData(dhj)
	assert.NoError(t, err)

	dhj.DataType = 1337
	err = m.compareJobsToData(dhj)
	assert.ErrorIs(t, err, errUnknownDataType)

	dhj.DataType = dataHistoryConvertCandlesDataType
	err = m.compareJobsToData(dhj)
	assert.NoError(t, err)

	m.started = 0
	err = m.compareJobsToData(dhj)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	err = m.compareJobsToData(dhj)
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestRunJob(t *testing.T) {
	t.Parallel()
	tt := time.Now().Truncate(kline.OneHour.Duration())
	testCases := []*DataHistoryJob{
		{
			Nickname:  "TestRunJobDataHistoryCandleDataType",
			Exchange:  testExchange,
			Asset:     asset.Spot,
			Pair:      currency.NewBTCUSDT(),
			StartDate: tt.Add(-kline.FifteenMin.Duration()),
			EndDate:   tt,
			Interval:  kline.FifteenMin,
			DataType:  dataHistoryCandleDataType,
		},
		{
			Nickname:  "TestRunJobDataHistoryTradeDataType",
			Exchange:  testExchange,
			Asset:     asset.Spot,
			Pair:      currency.NewBTCUSDT(),
			StartDate: tt.Add(-kline.OneMin.Duration()),
			EndDate:   tt,
			Interval:  kline.OneMin,
			DataType:  dataHistoryTradeDataType,
		},
		{
			Nickname:           "TestRunJobDataHistoryConvertCandlesDataType",
			Exchange:           testExchange,
			Asset:              asset.Spot,
			Pair:               currency.NewBTCUSDT(),
			StartDate:          tt.Add(-kline.OneHour.Duration()),
			EndDate:            tt,
			Interval:           kline.FifteenMin,
			DataType:           dataHistoryConvertCandlesDataType,
			ConversionInterval: kline.OneHour,
		},
		{
			Nickname:           "TestRunJobDataHistoryConvertTradesDataType",
			Exchange:           testExchange,
			Asset:              asset.Spot,
			Pair:               currency.NewBTCUSDT(),
			StartDate:          tt.Add(-kline.OneHour.Duration()),
			EndDate:            tt,
			Interval:           kline.FifteenMin,
			DataType:           dataHistoryConvertTradesDataType,
			ConversionInterval: kline.OneHour,
		},
		{
			Nickname:  "TestRunJobDataHistoryCandleValidationDataType",
			Exchange:  testExchange,
			Asset:     asset.Spot,
			Pair:      currency.NewBTCUSDT(),
			StartDate: tt.Add(-kline.OneHour.Duration()),
			EndDate:   tt,
			Interval:  kline.OneHour,
			DataType:  dataHistoryCandleValidationDataType,
		},
		{
			Nickname:                "TestRunJobDataHistoryCandleSecondaryValidationDataType",
			Exchange:                testExchange,
			Asset:                   asset.Spot,
			Pair:                    currency.NewBTCUSDT(),
			StartDate:               tt.Add(-kline.OneMin.Duration()),
			EndDate:                 tt,
			Interval:                kline.OneMin,
			DataType:                dataHistoryCandleValidationSecondarySourceType,
			SecondaryExchangeSource: "Binance",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Nickname, func(t *testing.T) {
			t.Parallel()
			m, _ := createDHM(t)
			m.tradeSaver = dataHistoryTradeSaver
			m.candleSaver = dataHistoryCandleSaver
			m.tradeLoader = dataHistoryTraderLoader
			err := m.UpsertJob(tc, false)
			assert.NoError(t, err)

			tc.Status = dataHistoryIntervalIssuesFound
			err = m.runJob(tc)
			assert.ErrorIs(t, err, errJobInvalid)

			rh := tc.rangeHolder
			tc.Status = dataHistoryStatusActive
			tc.rangeHolder = nil
			err = m.runJob(tc)
			assert.ErrorIs(t, err, errJobInvalid)

			tc.rangeHolder = rh
			err = m.runJob(tc)
			assert.NoError(t, err)
		})
	}
	var badM *DataHistoryManager
	err := badM.runJob(nil)
	assert.ErrorIs(t, err, ErrNilSubsystem)

	badM = &DataHistoryManager{}
	err = badM.runJob(nil)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)
}

func TestGenerateJobSummaryTest(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestGenerateJobSummary",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	assert.NoError(t, err)

	summary, err := m.GenerateJobSummary("TestGenerateJobSummary")
	assert.NoError(t, err)

	if len(summary.ResultRanges) == 0 {
		t.Error("expected result ranges")
	}

	atomic.StoreInt32(&m.started, 0)
	_, err = m.GenerateJobSummary("TestGenerateJobSummary")
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	_, err = m.GenerateJobSummary("TestGenerateJobSummary")
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestRunJobs(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	err := m.runJobs()
	assert.NoError(t, err)

	atomic.StoreInt32(&m.started, 0)
	err = m.runJobs()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	err = m.runJobs()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestConverters(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	id, err := uuid.NewV4()
	assert.NoError(t, err)

	id2, err := uuid.NewV4()
	assert.NoError(t, err)

	dhj := &DataHistoryJob{
		ID:        id,
		Nickname:  "TestProcessJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		StartDate: time.Now().Add(-time.Hour * 24),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}

	dbJob := m.convertJobToDBModel(dhj)
	if dhj.ID.String() != dbJob.ID ||
		dhj.Nickname != dbJob.Nickname ||
		!dhj.StartDate.Equal(dbJob.StartDate) ||
		int64(dhj.Interval.Duration()) != dbJob.Interval ||
		dhj.Pair.Base.String() != dbJob.Base ||
		dhj.Pair.Quote.String() != dbJob.Quote {
		t.Error("expected matching job")
	}

	convertBack, err := m.convertDBModelToJob(dbJob)
	assert.NoError(t, err)

	if dhj.ID != convertBack.ID ||
		dhj.Nickname != convertBack.Nickname ||
		!dhj.StartDate.Equal(convertBack.StartDate) ||
		dhj.Interval != convertBack.Interval ||
		!dhj.Pair.Equal(convertBack.Pair) {
		t.Error("expected matching job")
	}

	jr := DataHistoryJobResult{
		ID:                id,
		JobID:             id2,
		IntervalStartDate: dhj.StartDate,
		IntervalEndDate:   dhj.EndDate,
		Status:            0,
		Result:            "test123",
		Date:              time.Now(),
	}
	mapperino := make(map[int64][]DataHistoryJobResult)
	mapperino[dhj.StartDate.Unix()] = append(mapperino[dhj.StartDate.Unix()], jr)
	result := m.convertJobResultToDBResult(mapperino)
	if jr.ID.String() != result[0].ID ||
		jr.JobID.String() != result[0].JobID ||
		jr.Result != result[0].Result ||
		!jr.Date.Equal(result[0].Date) ||
		!jr.IntervalStartDate.Equal(result[0].IntervalStartDate) ||
		!jr.IntervalEndDate.Equal(result[0].IntervalEndDate) ||
		jr.Status != dataHistoryStatus(result[0].Status) {
		t.Error("expected matching job")
	}

	andBackAgain, err := m.convertDBResultToJobResult(result)
	assert.NoError(t, err)

	if jr.ID != andBackAgain[dhj.StartDate.Unix()][0].ID ||
		jr.JobID != andBackAgain[dhj.StartDate.Unix()][0].JobID ||
		jr.Result != andBackAgain[dhj.StartDate.Unix()][0].Result ||
		!jr.Date.Equal(andBackAgain[dhj.StartDate.Unix()][0].Date) ||
		!jr.IntervalStartDate.Equal(andBackAgain[dhj.StartDate.Unix()][0].IntervalStartDate) ||
		!jr.IntervalEndDate.Equal(andBackAgain[dhj.StartDate.Unix()][0].IntervalEndDate) ||
		jr.Status != andBackAgain[dhj.StartDate.Unix()][0].Status {
		t.Error("expected matching job")
	}
}

func createDHM(t *testing.T) (*DataHistoryManager, *datahistoryjob.DataHistoryJob) {
	t.Helper()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err)

	cp := currency.NewBTCUSD()
	cp2 := currency.NewBTCUSDT()
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:    currency.Pairs{cp, cp2},
		Enabled:      currency.Pairs{cp, cp2},
		AssetEnabled: true,
	}
	err = em.Add(exch)
	require.NoError(t, err)

	exch2, err := em.NewExchangeByName("Binance")
	require.NoError(t, err)

	exch2.SetDefaults()
	b = exch2.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}

	err = em.Add(exch2)
	require.NoError(t, err)

	j := &datahistoryjob.DataHistoryJob{
		ID:               jobID,
		Nickname:         "datahistoryjob",
		ExchangeName:     testExchange,
		Asset:            "spot",
		Base:             "btc",
		Quote:            "usd",
		StartDate:        startDate,
		EndDate:          endDate,
		Interval:         int64(kline.OneHour.Duration()),
		RequestSizeLimit: 3,
		MaxRetryAttempts: 3,
		BatchSize:        3,
		CreatedDate:      endDate,
		Status:           0,
		Results: []*datahistoryjobresult.DataHistoryJobResult{
			{
				ID:    jobID,
				JobID: jobID,
			},
		},
	}
	m := &DataHistoryManager{
		databaseConnectionInstance: &dataBaseConnection{},
		jobDB:                      &dataHistoryJobService{job: j},
		jobResultDB:                dataHistoryJobResultService{},
		started:                    1,
		exchangeManager:            em,
		candleLoader:               dataHistoryCandleLoader,
		interval:                   time.NewTicker(time.Minute),
		verbose:                    true,
		maxResultInsertions:        defaultMaxResultInsertions,
	}
	return m, j
}

type dataBaseConnection struct{}

func (d *dataBaseConnection) IsConnected() bool {
	return false
}

func (d *dataBaseConnection) GetSQL() (*sql.DB, error) {
	return nil, errors.New("not implemented")
}

func (d *dataBaseConnection) GetConfig() *database.Config {
	return nil
}

func TestProcessCandleData(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	_, err := m.processCandleData(nil, nil, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, errNilJob)

	j := &DataHistoryJob{
		Nickname:  "",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		StartDate: time.Now().Add(-kline.OneHour.Duration() * 2).Truncate(kline.OneHour.Duration()),
		EndDate:   time.Now().Truncate(kline.OneHour.Duration()),
		Interval:  kline.OneHour,
	}
	_, err = m.processCandleData(j, nil, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	assert.NoError(t, err)

	exch.SetDefaults()
	fakeExchange := dhmExchange{
		IBotExchange: exch,
	}
	_, err = m.processCandleData(j, exch, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)

	m.candleSaver = dataHistoryCandleSaver
	j.rangeHolder, err = kline.CalculateCandleDateRanges(j.StartDate, j.EndDate, j.Interval, 1337)
	if err != nil {
		t.Error(err)
	}
	r, err := m.processCandleData(j, fakeExchange, j.StartDate, j.EndDate, 0)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusComplete {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusComplete)
	}
	r, err = m.processCandleData(j, exch, j.StartDate, j.EndDate, 0)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}
}

func TestProcessTradeData(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	_, err := m.processTradeData(nil, nil, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, errNilJob)

	j := &DataHistoryJob{
		Nickname:  "",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		StartDate: time.Now().Add(-kline.OneHour.Duration() * 2).Truncate(kline.OneHour.Duration()),
		EndDate:   time.Now().Truncate(kline.OneHour.Duration()),
		Interval:  kline.OneHour,
	}
	_, err = m.processTradeData(j, nil, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	assert.NoError(t, err)

	exch.SetDefaults()
	fakeExchange := dhmExchange{
		IBotExchange: exch,
	}
	_, err = m.processTradeData(j, exch, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)

	j.rangeHolder, err = kline.CalculateCandleDateRanges(j.StartDate, j.EndDate, j.Interval, 1337)
	if err != nil {
		t.Error(err)
	}
	m.tradeSaver = dataHistoryTradeSaver
	r, err := m.processTradeData(j, fakeExchange, j.StartDate, j.EndDate, 0)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}
	r, err = m.processTradeData(j, exch, j.StartDate, j.EndDate, 0)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}
}

func TestConvertJobTradesToCandles(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	_, err := m.convertTradesToCandles(nil, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errNilJob)

	j := &DataHistoryJob{
		Nickname:  "",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		StartDate: time.Now().Add(-kline.OneHour.Duration() * 2),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}
	_, err = m.convertTradesToCandles(j, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)

	m.tradeLoader = dataHistoryTraderLoader
	m.candleSaver = dataHistoryCandleSaver
	r, err := m.convertTradesToCandles(j, j.StartDate, j.EndDate)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusComplete {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusComplete)
	}
}

func TestUpscaleJobCandleData(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	m.candleSaver = dataHistoryCandleSaver
	_, err := m.convertCandleData(nil, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errNilJob)

	j := &DataHistoryJob{
		Nickname:           "",
		Exchange:           testExchange,
		Asset:              asset.Spot,
		Pair:               currency.NewBTCUSDT(),
		StartDate:          time.Now().Add(-kline.OneHour.Duration() * 24),
		EndDate:            time.Now(),
		Interval:           kline.OneHour,
		ConversionInterval: kline.OneDay,
	}
	_, err = m.convertCandleData(j, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)

	r, err := m.convertCandleData(j, j.StartDate, j.EndDate)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusComplete {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusComplete)
	}
}

func TestValidateCandles(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	m.candleSaver = dataHistoryCandleSaver
	_, err := m.validateCandles(nil, nil, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errNilJob)

	j := &DataHistoryJob{
		Nickname:  "",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		StartDate: time.Now().Add(-kline.OneHour.Duration() * 2),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}
	_, err = m.validateCandles(j, nil, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	assert.NoError(t, err)

	exch.SetDefaults()
	fakeExchange := dhmExchange{
		IBotExchange: exch,
	}
	_, err = m.validateCandles(j, exch, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)

	j.rangeHolder, err = kline.CalculateCandleDateRanges(j.StartDate, j.EndDate, j.Interval, 1337)
	if err != nil {
		t.Error(err)
	}
	r, err := m.validateCandles(j, fakeExchange, j.StartDate, j.EndDate)
	assert.NoError(t, err)

	if r.Status != dataHistoryIntervalIssuesFound {
		t.Errorf("received %v expected %v", r.Status, dataHistoryIntervalIssuesFound)
	}
	r, err = m.validateCandles(j, exch, j.StartDate, j.EndDate)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}
}

func TestSetJobRelationship(t *testing.T) {
	t.Parallel()
	m, j := createDHM(t)
	err := m.SetJobRelationship("test", "123")
	assert.NoError(t, err)

	jID, err := uuid.NewV4()
	assert.NoError(t, err)

	j.ID = jID.String()
	j.PrerequisiteJobID = ""
	j.PrerequisiteJobNickname = ""
	err = m.SetJobRelationship("", "123")
	assert.NoError(t, err)

	err = m.SetJobRelationship("", "")
	assert.ErrorIs(t, err, errNicknameUnset)

	m.started = 0
	err = m.SetJobRelationship("", "")
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	err = m.SetJobRelationship("", "")
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestCheckCandleIssue(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	issue, replace := m.CheckCandleIssue(nil, 0, 0, 0, "")
	if issue != errNilJob.Error() {
		t.Errorf("expected 'nil job' received %v", issue)
	}
	if replace {
		t.Errorf("expected %v received %v", false, replace)
	}

	job := &DataHistoryJob{
		IssueTolerancePercentage: 0,
		ReplaceOnIssue:           false,
		DecimalPlaceComparison:   0,
	}
	issue, replace = m.CheckCandleIssue(job, 0, 0, 0, "")
	if issue != "" {
		t.Errorf("expected 'nil job' received %v", issue)
	}
	if replace {
		t.Errorf("expected %v received %v", false, replace)
	}

	issue, replace = m.CheckCandleIssue(job, 0, 1, 2, "Open")
	if issue != "Open api: 1 db: 2 diff: 100 %" {
		t.Errorf("expected 'Open api: 1 db: 2 diff: 100 %%' received %v", issue)
	}
	if replace {
		t.Errorf("expected %v received %v", false, replace)
	}

	job.IssueTolerancePercentage = 100
	issue, replace = m.CheckCandleIssue(job, 0, 1, 1.5, "Open")
	if issue != "" {
		t.Errorf("expected 'Open api: 1 db: 2 diff: 100 %%' received %v", issue)
	}
	if replace {
		t.Errorf("expected %v received %v", false, replace)
	}

	job.IssueTolerancePercentage = 1
	job.ReplaceOnIssue = true
	issue, replace = m.CheckCandleIssue(job, 10, 1.5, 1, "Open")
	if issue != "Open api: 1.5 db: 1 diff: 50 %" {
		t.Errorf("expected 'Open api: 1.5 db: 1 diff: 50 %%' received %v", issue)
	}
	if !replace {
		t.Errorf("expected %v received %v", true, replace)
	}

	m.started = 0
	issue, replace = m.CheckCandleIssue(nil, 0, 0, 0, "")
	if issue != ErrSubSystemNotStarted.Error() {
		t.Errorf("expected %v received %v", ErrSubSystemNotStarted, issue)
	}
	if replace {
		t.Errorf("expected %v received %v", false, replace)
	}

	m = nil
	issue, replace = m.CheckCandleIssue(nil, 0, 0, 0, "")
	if issue != ErrNilSubsystem.Error() {
		t.Errorf("expected %v received %v", ErrNilSubsystem, issue)
	}
	if replace {
		t.Errorf("expected %v received %v", false, replace)
	}
}

func TestCompletionCheck(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	err := m.completeJob(nil, false, false)
	assert.ErrorIs(t, err, errNilJob)

	j := &DataHistoryJob{
		Status: dataHistoryStatusActive,
	}
	err = m.completeJob(j, false, false)
	assert.NoError(t, err)

	if j.Status != dataHistoryIntervalIssuesFound {
		t.Errorf("received %v expected %v", j.Status, dataHistoryIntervalIssuesFound)
	}

	err = m.completeJob(j, true, false)
	assert.NoError(t, err)

	if j.Status != dataHistoryStatusComplete {
		t.Errorf("received %v expected %v", j.Status, dataHistoryStatusComplete)
	}

	err = m.completeJob(j, false, true)
	assert.NoError(t, err)

	if j.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", j.Status, dataHistoryStatusFailed)
	}

	err = m.completeJob(j, true, true)
	assert.ErrorIs(t, err, errJobInvalid)
}

func TestSaveCandlesInBatches(t *testing.T) {
	t.Parallel()
	dhm := DataHistoryManager{
		candleSaver: dataHistoryCandleSaver,
	}
	err := dhm.saveCandlesInBatches(nil, nil, nil)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	dhm.started = 1
	err = dhm.saveCandlesInBatches(nil, nil, nil)
	assert.ErrorIs(t, err, errNilJob)

	job := &DataHistoryJob{}
	err = dhm.saveCandlesInBatches(job, nil, nil)
	assert.ErrorIs(t, err, errNilCandles)

	candles := &kline.Item{}
	err = dhm.saveCandlesInBatches(job, candles, nil)
	assert.ErrorIs(t, err, errNilResult)

	result := &DataHistoryJobResult{}
	err = dhm.saveCandlesInBatches(job, candles, result)
	assert.NoError(t, err)

	for i := range 10000 {
		candles.Candles = append(candles.Candles, kline.Candle{
			Volume: float64(i),
		})
	}
	dhm.maxResultInsertions = 1337
	err = dhm.saveCandlesInBatches(job, candles, result)
	assert.NoError(t, err)
}

// these structs and function implementations are used
// to override database implementations as we are not testing those
// results here. see tests in the database folder
type dataHistoryJobService struct {
	datahistoryjob.IDBService
	job *datahistoryjob.DataHistoryJob
}

type dataHistoryJobResultService struct {
	datahistoryjobresult.IDBService
}

var (
	jobID     = "00a434e2-8502-4d6b-865f-e4243fd8b5a7"
	startDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)
	endDate   = time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
)

func (d dataHistoryJobService) Upsert(_ ...*datahistoryjob.DataHistoryJob) error {
	return nil
}

func (d dataHistoryJobService) SetRelationshipByID(prereq, _ string, status int64) error {
	d.job.PrerequisiteJobID = prereq
	d.job.Status = status
	return nil
}

func (d dataHistoryJobService) SetRelationshipByNickname(prereq, _ string, status int64) error {
	d.job.PrerequisiteJobNickname = prereq
	d.job.Status = status
	return nil
}

func (d dataHistoryJobService) GetByNickName(_ string) (*datahistoryjob.DataHistoryJob, error) {
	return d.job, nil
}

func (d dataHistoryJobService) GetJobsBetween(_, _ time.Time) ([]datahistoryjob.DataHistoryJob, error) {
	return []datahistoryjob.DataHistoryJob{*d.job}, nil
}

func (d dataHistoryJobService) GetByID(id string) (*datahistoryjob.DataHistoryJob, error) {
	d.job.ID = id
	return d.job, nil
}

func (d dataHistoryJobService) GetAllIncompleteJobsAndResults() ([]datahistoryjob.DataHistoryJob, error) {
	return []datahistoryjob.DataHistoryJob{*d.job}, nil
}

func (d dataHistoryJobService) GetJobAndAllResults(nickname string) (*datahistoryjob.DataHistoryJob, error) {
	d.job.Nickname = nickname
	return d.job, nil
}

func (d dataHistoryJobService) GetRelatedUpcomingJobs(_ string) ([]*datahistoryjob.DataHistoryJob, error) {
	return []*datahistoryjob.DataHistoryJob{
		{
			Nickname: "test123",
			Status:   int64(dataHistoryStatusPaused),
		},
	}, nil
}

func (d dataHistoryJobResultService) Upsert(_ ...*datahistoryjobresult.DataHistoryJobResult) error {
	return nil
}

func (d dataHistoryJobResultService) GetByJobID(_ string) ([]datahistoryjobresult.DataHistoryJobResult, error) {
	return nil, nil
}

func (d dataHistoryJobResultService) GetJobResultsBetween(_ string, _, _ time.Time) ([]datahistoryjobresult.DataHistoryJobResult, error) {
	return nil, nil
}

func dataHistoryTraderLoader(exch, a, base, quote string, start, _ time.Time) ([]trade.Data, error) {
	cp, err := currency.NewPairFromStrings(base, quote)
	if err != nil {
		return nil, err
	}
	ai, err := asset.New(a)
	if err != nil {
		return nil, err
	}
	return []trade.Data{
		{
			Exchange:     exch,
			CurrencyPair: cp,
			AssetType:    ai,
			Side:         order.Buy,
			Price:        1337,
			Amount:       1337,
			Timestamp:    start,
		},
	}, nil
}

func dataHistoryCandleLoader(exch string, cp currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	start = start.Truncate(interval.Duration())
	end = end.Truncate(interval.Duration())
	intervals := end.Sub(start) / interval.Duration()
	candles := make([]kline.Candle, int(intervals))
	for x := range int(intervals) {
		candles[x] = kline.Candle{
			Time:   start,
			Open:   1,
			High:   10,
			Low:    1,
			Close:  4,
			Volume: 8,
		}
		start = start.Add(interval.Duration())
	}
	return &kline.Item{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
		Interval: interval,
		Candles:  candles,
	}, nil
}

func dataHistoryTradeSaver(...trade.Data) error {
	return nil
}

func dataHistoryCandleSaver(_ *kline.Item, _ bool) (uint64, error) {
	return 0, nil
}

// dhmExchange aka datahistorymanager fake exchange overrides exchange functions
// we're not testing an actual exchange's implemented functions
type dhmExchange struct {
	exchange.IBotExchange
}

func (f dhmExchange) GetHistoricCandlesExtended(_ context.Context, p currency.Pair, a asset.Item, interval kline.Interval, timeStart, _ time.Time) (*kline.Item, error) {
	return &kline.Item{
		Exchange: testExchange,
		Pair:     p,
		Asset:    a,
		Interval: interval,
		Candles: []kline.Candle{
			{
				Time:   timeStart,
				Open:   1,
				High:   2,
				Low:    3,
				Close:  4,
				Volume: 5,
			},
			{
				Time:   timeStart.Add(interval.Duration()),
				Open:   1,
				High:   2,
				Low:    3,
				Close:  4,
				Volume: 5,
			},
			{
				Time:   timeStart.Add(interval.Duration() * 2),
				Open:   1,
				High:   2,
				Low:    3,
				Close:  4,
				Volume: 5,
			},
		},
	}, nil
}

func (f dhmExchange) GetHistoricTrades(_ context.Context, p currency.Pair, a asset.Item, startTime, _ time.Time) ([]trade.Data, error) {
	return []trade.Data{
		{
			Exchange:     testExchange,
			CurrencyPair: p,
			AssetType:    a,
			Side:         order.Buy,
			Price:        1337,
			Amount:       4,
			Timestamp:    startTime.Add(time.Minute),
		},
		{
			Exchange:     testExchange,
			CurrencyPair: p,
			AssetType:    a,
			Side:         order.Buy,
			Price:        1338,
			Amount:       2,
			Timestamp:    startTime.Add(time.Minute * 2),
		},
	}, nil
}
