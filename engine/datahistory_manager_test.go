package engine

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/repository/candle"
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
		dbConn: dbInst,
	}
	dbCM.started.Store(true)
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
	m.started.Store(false)
	if m.IsRunning() {
		t.Error("expected false")
	}
	m.started.Store(true)
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
	m.started.Store(false)
	err := m.Start(t.Context())
	assert.NoError(t, err)

	err = m.Start(t.Context())
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted)

	m = nil
	err = m.Start(t.Context())
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

	t.Run("conversion job rounding to no conversion interval", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		start := time.Now().Add(-time.Hour).Truncate(kline.FiveMin.Duration())
		job := &DataHistoryJob{
			Nickname:           "TestUpsertJobShortConversion",
			Exchange:           testExchange,
			Asset:              asset.Spot,
			Pair:               currency.NewBTCUSD(),
			StartDate:          start,
			EndDate:            start.Add(2 * time.Minute),
			Interval:           kline.OneMin,
			ConversionInterval: kline.FiveMin,
			DataType:           dataHistoryConvertCandlesDataType,
		}
		err := mgr.UpsertJob(job, false)
		assert.ErrorIs(t, err, common.ErrStartEqualsEnd, "UpsertJob should refuse a conversion job whose dates round to one conversion interval start")
	})
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

	m.started.Store(false)
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

	t.Run("missing job is not found", func(t *testing.T) {
		t.Parallel()
		missing, _ := createDHM(t)
		missing.jobDB = missingJobDB{}
		_, err := missing.GetByNickname("missing", false)
		assert.ErrorIs(t, err, errJobNotFound, "GetByNickname should report a missing job as not found")
	})

	t.Run("missing job with full details is not found", func(t *testing.T) {
		t.Parallel()
		missing, _ := createDHM(t)
		missing.jobDB = missingJobDB{}
		_, err := missing.GetByNickname("missing", true)
		assert.ErrorIs(t, err, errJobNotFound, "GetByNickname should report a missing job as not found")
	})

	t.Run("database failure is not a missing job", func(t *testing.T) {
		t.Parallel()
		failing, _ := createDHM(t)
		failing.jobDB = failingJobDB{}
		_, err := failing.GetByNickname("failing", false)
		assert.ErrorIs(t, err, errExpectedTestError, "GetByNickname should return the database's error")
		assert.NotErrorIs(t, err, errJobNotFound, "GetByNickname should not report a database failure as a missing job")
	})

	t.Run("database failure with full details is not a missing job", func(t *testing.T) {
		t.Parallel()
		failing, _ := createDHM(t)
		failing.jobDB = failingJobDB{}
		_, err := failing.GetByNickname("failing", true)
		assert.ErrorIs(t, err, errExpectedTestError, "GetByNickname should return the database's error")
		assert.NotErrorIs(t, err, errJobNotFound, "GetByNickname should not report a database failure as a missing job")
	})

	m.started.Store(false)
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

	t.Run("missing job is not found", func(t *testing.T) {
		t.Parallel()
		missing, _ := createDHM(t)
		missing.jobDB = missingJobDB{}
		_, err := missing.GetByID(dhj.ID)
		assert.ErrorIs(t, err, errJobNotFound, "GetByID should report a missing job as not found")
	})

	t.Run("database failure is not a missing job", func(t *testing.T) {
		t.Parallel()
		failing, _ := createDHM(t)
		failing.jobDB = failingJobDB{}
		_, err := failing.GetByID(dhj.ID)
		assert.ErrorIs(t, err, errExpectedTestError, "GetByID should return the database's error")
		assert.NotErrorIs(t, err, errJobNotFound, "GetByID should not report a database failure as a missing job")
	})

	m.started.Store(false)
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

	m.started.Store(false)
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

	m.started.Store(false)
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

	newJob := func(dataType dataHistoryDataType) *DataHistoryJob {
		return &DataHistoryJob{
			Nickname:  "TestValidateJob",
			Exchange:  testExchange,
			Asset:     asset.Spot,
			Pair:      currency.NewBTCUSD(),
			StartDate: time.Now().Add(-time.Hour * 60),
			EndDate:   time.Now().Add(-time.Minute),
			Interval:  kline.OneHour,
			DataType:  dataType,
		}
	}

	t.Run("convert candles needs a source interval", func(t *testing.T) {
		t.Parallel()
		job := newJob(dataHistoryConvertCandlesDataType)
		job.Interval = 0
		job.ConversionInterval = kline.OneDay
		assert.ErrorIs(t, m.validateJob(job), kline.ErrUnsupportedInterval, "validateJob should reject a convert candles job without an interval")
		job.Interval = kline.OneHour
		assert.NoError(t, m.validateJob(job), "validateJob should accept a convert candles job with both intervals")
	})

	t.Run("decimal place comparison limit is 18", func(t *testing.T) {
		t.Parallel()
		job := newJob(dataHistoryCandleValidationDataType)
		job.DecimalPlaceComparison = 18
		assert.NoError(t, m.validateJob(job), "validateJob should accept 18 decimal places")
		job.DecimalPlaceComparison = 19
		assert.ErrorIs(t, m.validateJob(job), errInvalidDecimalPlaces, "validateJob should reject 19 decimal places, as ten to that power overflows an int64")
		job.DecimalPlaceComparison = 20
		assert.ErrorIs(t, m.validateJob(job), errInvalidDecimalPlaces, "validateJob should reject 20 decimal places")
	})

	t.Run("secondary source validation limits", func(t *testing.T) {
		t.Parallel()
		job := newJob(dataHistoryCandleValidationSecondarySourceType)
		job.SecondaryExchangeSource = testExchange
		job.RequestSizeLimit = defaultDataHistoryRequestSizeLimit + 1
		require.NoError(t, m.validateJob(job), "validateJob must not error")
		assert.Equal(t, defaultDataHistoryRequestSizeLimit, job.RequestSizeLimit, "request size limit should be capped")
		assert.Zero(t, job.DecimalPlaceComparison, "a secondary source job should keep a zero decimal place comparison")
		job.DecimalPlaceComparison = maxDecimalPlaceComparison + 1
		assert.ErrorIs(t, m.validateJob(job), errInvalidDecimalPlaces, "validateJob should reject a decimal place comparison an int64 cannot hold")
	})

	t.Run("primary source decimal place comparison default", func(t *testing.T) {
		t.Parallel()
		job := newJob(dataHistoryCandleValidationDataType)
		require.NoError(t, m.validateJob(job), "validateJob must not error")
		assert.Equal(t, defaultDecimalPlaceComparison, job.DecimalPlaceComparison, "a primary source job should default a zero decimal place comparison")
	})

	t.Run("range with too many intervals", func(t *testing.T) {
		t.Parallel()
		job := newJob(dataHistoryCandleDataType)
		job.Interval = kline.OneMin
		job.StartDate = time.Now().Add(-time.Minute * (maxDataHistoryJobIntervals + 60))
		assert.ErrorIs(t, m.validateJob(job), errRangeTooLarge, "validateJob should reject a job spanning too many intervals")
	})

	t.Run("range at the job limit", func(t *testing.T) {
		t.Parallel()
		job := newJob(dataHistoryCandleDataType)
		job.Interval = kline.FiveMin
		job.EndDate = time.Now().Add(-time.Hour).Truncate(time.Minute)
		job.StartDate = job.EndDate.Add(-1_000_000 * 5 * time.Minute)
		assert.NoError(t, m.validateJob(job), "validateJob should accept a job of 1,000,000 intervals")
		job.StartDate = job.StartDate.Add(-5 * time.Minute)
		assert.ErrorIs(t, m.validateJob(job), errRangeTooLarge, "validateJob should reject a job of 1,000,001 intervals")
	})

	t.Run("conversion dates within whole conversion intervals", func(t *testing.T) {
		t.Parallel()
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		for _, tc := range []struct {
			name               string
			from, to           time.Duration
			wantStart, wantEnd time.Duration
			wantErr            error
		}{
			{name: "aligned dates", to: 15 * time.Minute, wantEnd: 15 * time.Minute},
			{name: "start before and end after a midpoint", from: 2 * time.Minute, to: 13 * time.Minute, wantStart: 5 * time.Minute, wantEnd: 10 * time.Minute},
			{name: "start after and end before a midpoint", from: 3 * time.Minute, to: 12 * time.Minute, wantStart: 5 * time.Minute, wantEnd: 10 * time.Minute},
			{name: "dates seconds from conversion interval starts", from: 20 * time.Second, to: 10*time.Minute - 20*time.Second, wantEnd: 10 * time.Minute},
			{name: "dates within one conversion interval", from: time.Minute, to: 4 * time.Minute, wantErr: common.ErrStartAfterEnd},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				job := newJob(dataHistoryConvertCandlesDataType)
				job.Interval = kline.OneMin
				job.ConversionInterval = kline.FiveMin
				job.StartDate = start.Add(tc.from)
				job.EndDate = start.Add(tc.to)
				err := m.validateJob(job)
				if tc.wantErr != nil {
					assert.ErrorIs(t, err, tc.wantErr, "validateJob should reject dates holding no whole conversion interval")
					return
				}
				require.NoError(t, err, "validateJob must not error")
				assert.Equal(t, start.Add(tc.wantStart), job.StartDate, "validateJob should start a conversion at its first whole conversion interval")
				assert.Equal(t, start.Add(tc.wantEnd), job.EndDate, "validateJob should end a conversion with its last whole conversion interval")
			})
		}
	})

	t.Run("candle job dates kept off its conversion interval", func(t *testing.T) {
		t.Parallel()
		start := time.Date(2025, 1, 1, 0, 2, 0, 0, time.UTC)
		job := newJob(dataHistoryCandleDataType)
		job.Interval = kline.OneMin
		job.ConversionInterval = kline.FiveMin
		job.StartDate = start
		job.EndDate = start.Add(11 * time.Minute)
		require.NoError(t, m.validateJob(job), "validateJob must not error")
		assert.Equal(t, start, job.StartDate, "validateJob should keep a candle job's start off its conversion interval")
		assert.Equal(t, start.Add(11*time.Minute), job.EndDate, "validateJob should keep a candle job's end off its conversion interval")
	})

	t.Run("conversion range at the job limit", func(t *testing.T) {
		t.Parallel()
		job := newJob(dataHistoryConvertCandlesDataType)
		job.Interval = kline.FiveMin
		job.ConversionInterval = kline.OneMin
		job.EndDate = time.Now().Add(-time.Hour).Truncate(time.Minute)
		job.StartDate = job.EndDate.Add(-1_000_000 * time.Minute)
		assert.NoError(t, m.validateJob(job), "validateJob should accept a job of 1,000,000 conversion intervals")
		// validateJob rounds the dates to the five-minute interval, so the range can only grow by five conversion intervals
		job.StartDate = job.StartDate.Add(-5 * time.Minute)
		assert.ErrorIs(t, m.validateJob(job), errRangeTooLarge, "validateJob should reject a job of 1,000,005 conversion intervals")
	})

	t.Run("sub-second range with too many intervals", func(t *testing.T) {
		t.Parallel()
		job := newJob(dataHistoryConvertTradesDataType)
		job.Interval = kline.Interval(time.Nanosecond)
		job.StartDate = time.Now().Add(-time.Hour).Truncate(time.Second)
		job.EndDate = job.StartDate.Add(900 * time.Millisecond)
		assert.ErrorIs(t, m.validateJob(job), errIntervalNotWholeSeconds, "validateJob should refuse an interval of part of a second before counting its intervals")
	})

	t.Run("conversion interval with too many intervals", func(t *testing.T) {
		t.Parallel()
		job := newJob(dataHistoryConvertCandlesDataType)
		job.StartDate = job.EndDate.Add(-300 * time.Hour)
		job.ConversionInterval = kline.Interval(time.Second)
		assert.ErrorIs(t, m.validateJob(job), errRangeTooLarge, "validateJob should count the intervals the job's date ranges are built with")
		job.ConversionInterval = kline.OneDay
		assert.NoError(t, m.validateJob(job), "validateJob should accept a conversion interval within the limit")
	})

	for _, tc := range []struct {
		name       string
		interval   kline.Interval
		conversion kline.Interval
	}{
		{name: "conversion interval of part of a second", interval: kline.OneMin, conversion: kline.Interval(90 * time.Second / 60)},
		{name: "source interval of part of a second", interval: kline.Interval(1500 * time.Millisecond), conversion: kline.Interval(3 * time.Second)},
		{name: "calendar conversion interval", interval: kline.OneDay, conversion: kline.OneMonth},
		{name: "calendar source interval", interval: kline.OneMonth, conversion: kline.Interval(60 * 24 * time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			job := newJob(dataHistoryConvertCandlesDataType)
			job.Interval = tc.interval
			job.ConversionInterval = tc.conversion
			assert.ErrorIs(t, m.validateJob(job), kline.ErrUnsupportedInterval, "validateJob should reject a conversion between intervals that are not fixed whole seconds")
		})
	}

	for _, tc := range []struct {
		name     string
		dataType dataHistoryDataType
		interval kline.Interval
	}{
		{name: "validation interval of part of a second", dataType: dataHistoryCandleValidationDataType, interval: kline.HundredMilliseconds},
		{name: "secondary source validation interval of part of a second", dataType: dataHistoryCandleValidationSecondarySourceType, interval: kline.HundredMilliseconds},
		{name: "candle interval of part of a second", dataType: dataHistoryCandleDataType, interval: kline.HundredMilliseconds},
		{name: "trade conversion interval of part of a second", dataType: dataHistoryConvertTradesDataType, interval: kline.HundredMilliseconds},
		{name: "trade interval with part of a second", dataType: dataHistoryTradeDataType, interval: kline.Interval(60500 * time.Millisecond)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mgr, _ := createDHM(t)
			exch, err := mgr.exchangeManager.GetExchangeByName(testExchange)
			require.NoError(t, err, "GetExchangeByName must not error")
			// Listed by the exchange, over ten intervals within the interval limit, so that only the whole-second rule can refuse it
			exch.GetBase().Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: tc.interval})
			job := newJob(tc.dataType)
			job.Interval = tc.interval
			job.StartDate = job.EndDate.Add(-10 * tc.interval.Duration())
			job.SecondaryExchangeSource = testExchange
			assert.ErrorIs(t, mgr.validateJob(job), errIntervalNotWholeSeconds, "validateJob should refuse an interval of part of a second")
		})
	}

	t.Run("trade conversion interval of whole seconds", func(t *testing.T) {
		t.Parallel()
		job := newJob(dataHistoryConvertTradesDataType)
		job.Interval = kline.Interval(90 * time.Second)
		assert.NoError(t, m.validateJob(job), "validateJob should accept a trade conversion interval of whole seconds")
	})
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

	m.started.Store(false)
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
	m.started.Store(false)
	_, err = m.PrepareJobs()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	_, err = m.PrepareJobs()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestRangeInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		dataType dataHistoryDataType
		want     kline.Interval
	}{
		{dataType: dataHistoryCandleDataType, want: kline.OneMin},
		{dataType: dataHistoryTradeDataType, want: kline.OneMin},
		{dataType: dataHistoryConvertTradesDataType, want: kline.OneMin},
		{dataType: dataHistoryConvertCandlesDataType, want: kline.FiveMin},
		{dataType: dataHistoryCandleValidationDataType, want: kline.OneMin},
		{dataType: dataHistoryCandleValidationSecondarySourceType, want: kline.OneMin},
	} {
		t.Run(tc.dataType.String(), func(t *testing.T) {
			t.Parallel()
			job := &DataHistoryJob{Interval: kline.OneMin, ConversionInterval: kline.FiveMin, DataType: tc.dataType}
			assert.Equal(t, tc.want, job.rangeInterval(), "rangeInterval should return the interval the ranges are built with")
		})
	}
}

func TestDropStaleResults(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	result := func(from, to time.Duration, status dataHistoryStatus) DataHistoryJobResult {
		return DataHistoryJobResult{IntervalStartDate: start.Add(from), IntervalEndDate: start.Add(to), Status: status}
	}
	newJob := func(t *testing.T, results map[int64][]DataHistoryJobResult) *DataHistoryJob {
		t.Helper()
		h, err := kline.CalculateCandleDateRanges(start, start.Add(25*time.Minute), kline.FiveMin, 2)
		require.NoError(t, err, "CalculateCandleDateRanges must not error")
		require.Len(t, h.Ranges, 3, "the job must have three ranges")
		for i := range h.Ranges[0].Intervals {
			h.Ranges[0].Intervals[i].HasData = true
		}
		h.Ranges[1].Intervals[0].HasData = true
		return &DataHistoryJob{rangeHolder: h, Results: results}
	}

	t.Run("stale results", func(t *testing.T) {
		t.Parallel()
		job := newJob(t, map[int64][]DataHistoryJobResult{
			start.Unix():                       {result(0, 5*time.Minute, dataHistoryStatusFailed), result(0, 10*time.Minute, dataHistoryStatusComplete)},
			start.Add(2 * time.Minute).Unix():  {result(2*time.Minute, 4*time.Minute, dataHistoryStatusComplete)},
			start.Add(10 * time.Minute).Unix(): {result(10*time.Minute, 12*time.Minute, dataHistoryIntervalIssuesFound), result(10*time.Minute, 20*time.Minute, dataHistoryStatusFailed), result(10*time.Minute, 20*time.Minute, dataHistoryIntervalIssuesFound), result(10*time.Minute, 20*time.Minute, dataHistoryStatusComplete)},
			start.Add(20 * time.Minute).Unix(): {result(20*time.Minute, 25*time.Minute, dataHistoryStatusComplete)},
		})
		job.dropStaleResults()
		assert.Equal(t, map[int64][]DataHistoryJobResult{
			start.Unix():                       {result(0, 10*time.Minute, dataHistoryStatusComplete)},
			start.Add(2 * time.Minute).Unix():  {result(2*time.Minute, 4*time.Minute, dataHistoryStatusComplete)},
			start.Add(10 * time.Minute).Unix(): {result(10*time.Minute, 20*time.Minute, dataHistoryStatusFailed), result(10*time.Minute, 20*time.Minute, dataHistoryIntervalIssuesFound)},
		}, job.Results, "dropStaleResults should drop results for other batches, and completions of batches missing candles")
	})

	t.Run("result for a range starting within the same second", func(t *testing.T) {
		t.Parallel()
		job := newJob(t, map[int64][]DataHistoryJobResult{
			start.Unix(): {result(500*time.Millisecond, 10*time.Minute, dataHistoryStatusComplete), result(0, 10*time.Minute, dataHistoryStatusFailed)},
		})
		job.dropStaleResults()
		assert.Equal(t, map[int64][]DataHistoryJobResult{
			start.Unix(): {result(0, 10*time.Minute, dataHistoryStatusFailed)},
		}, job.Results, "dropStaleResults should drop a result for a range starting elsewhere within the same second")
	})

	t.Run("no results", func(t *testing.T) {
		t.Parallel()
		job := newJob(t, nil)
		assert.NotPanics(t, job.dropStaleResults, "dropStaleResults should accept a job without results")
		assert.Nil(t, job.Results, "dropStaleResults should leave a job without results as it was")
	})
}

func TestPadSourceCandles(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(offset time.Duration, value float64) kline.Candle {
		return kline.Candle{Time: start.Add(offset), Open: value, High: value + 3, Low: value - 1, Close: value + 1, Volume: value}
	}
	empty := func(offset time.Duration) kline.Candle { return kline.Candle{Time: start.Add(offset)} }
	for _, tc := range []struct {
		name        string
		candles     []kline.Candle
		end         time.Duration
		interval    kline.Interval
		wantPadded  []kline.Candle
		wantPresent []bool
	}{
		{name: "every interval", candles: []kline.Candle{at(0, 1), at(time.Minute, 2), at(2*time.Minute, 3)}, end: 3 * time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{at(0, 1), at(time.Minute, 2), at(2*time.Minute, 3)}, wantPresent: []bool{true, true, true}},
		{name: "leading gap", candles: []kline.Candle{at(time.Minute, 2), at(2*time.Minute, 3)}, end: 3 * time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{empty(0), at(time.Minute, 2), at(2*time.Minute, 3)}, wantPresent: []bool{false, true, true}},
		{name: "interior gap", candles: []kline.Candle{at(0, 1), at(2*time.Minute, 3)}, end: 3 * time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{at(0, 1), empty(time.Minute), at(2*time.Minute, 3)}, wantPresent: []bool{true, false, true}},
		{name: "candle away from an interval start", candles: []kline.Candle{at(0, 1), at(90*time.Second, 2), at(2*time.Minute, 3)}, end: 3 * time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{at(0, 1), at(time.Minute, 2), at(2*time.Minute, 3)}, wantPresent: []bool{true, true, true}},
		{name: "two candles in one interval", candles: []kline.Candle{at(0, 1), at(30*time.Second, 9), at(time.Minute, 2)}, end: 2 * time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{at(0, 1), at(time.Minute, 2)}, wantPresent: []bool{true, true}},
		{name: "candle without data", candles: []kline.Candle{at(0, 1), empty(time.Minute), at(2*time.Minute, 3)}, end: 3 * time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{at(0, 1), empty(time.Minute), at(2*time.Minute, 3)}, wantPresent: []bool{true, false, true}},
		{name: "candle without data before one with", candles: []kline.Candle{empty(0), at(30*time.Second, 2)}, end: time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{at(0, 2)}, wantPresent: []bool{true}},
		{name: "candle with nothing above zero", candles: []kline.Candle{{Time: start, Open: -1, High: -1, Low: -1, Close: -1}}, end: time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{empty(0)}, wantPresent: []bool{false}},
		{name: "candle without data away from an interval start", candles: []kline.Candle{at(0, 1), empty(90 * time.Second), at(2*time.Minute, 3)}, end: 3 * time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{at(0, 1), empty(time.Minute), at(2*time.Minute, 3)}, wantPresent: []bool{true, false, true}},
		{name: "price without volume", candles: []kline.Candle{{Time: start, Open: 1, High: 1, Low: 1, Close: 1}}, end: time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{{Time: start, Open: 1, High: 1, Low: 1, Close: 1}}, wantPresent: []bool{true}},
		{name: "candles outside the range", candles: []kline.Candle{at(-time.Minute, 9), at(time.Minute, 2), at(3*time.Minute, 9)}, end: 3 * time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{empty(0), at(time.Minute, 2), empty(2 * time.Minute)}, wantPresent: []bool{false, true, false}},
		{name: "no candles", end: 2 * time.Minute, interval: kline.OneMin, wantPadded: []kline.Candle{empty(0), empty(time.Minute)}, wantPresent: []bool{false, false}},
		{name: "candle in a partial final interval", candles: []kline.Candle{at(0, 1), at(time.Minute, 2), at(2*time.Minute+15*time.Second, 9)}, end: 2*time.Minute + 30*time.Second, interval: kline.OneMin, wantPadded: []kline.Candle{at(0, 1), at(time.Minute, 2)}, wantPresent: []bool{true, true}},
		{name: "no interval", candles: []kline.Candle{at(time.Minute, 2)}, end: 3 * time.Minute},
		{name: "no range", candles: []kline.Candle{at(0, 1)}, interval: kline.OneMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			padded, present := padSourceCandles(tc.candles, start, start.Add(tc.end), tc.interval)
			assert.Equal(t, tc.wantPadded, padded, "padSourceCandles should return one candle for each interval in the range")
			assert.Equal(t, tc.wantPresent, present, "padSourceCandles should report which intervals had a candle")
		})
	}
}

func TestCandleRuns(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(offsets ...time.Duration) []kline.Candle {
		candles := make([]kline.Candle, len(offsets))
		for i, offset := range offsets {
			candles[i] = kline.Candle{Time: start.Add(offset)}
		}
		return candles
	}
	times := func(offsets ...time.Duration) []time.Time {
		run := make([]time.Time, len(offsets))
		for i, offset := range offsets {
			run[i] = start.Add(offset)
		}
		return run
	}
	for _, tc := range []struct {
		name     string
		interval kline.Interval
		end      time.Time
		candles  []kline.Candle
		want     [][]time.Time
	}{
		{name: "every interval", interval: kline.FiveMin, end: start.Add(15 * time.Minute), candles: at(0, 5*time.Minute, 10*time.Minute), want: [][]time.Time{times(0, 5*time.Minute, 10*time.Minute)}},
		{name: "candles away from interval starts", interval: kline.FiveMin, end: start.Add(15 * time.Minute), candles: at(time.Minute, 9*time.Minute, 14*time.Minute), want: [][]time.Time{times(time.Minute, 9*time.Minute, 14*time.Minute)}},
		{name: "unordered candles", interval: kline.FiveMin, end: start.Add(15 * time.Minute), candles: at(10*time.Minute, 0, 5*time.Minute), want: [][]time.Time{times(0, 5*time.Minute, 10*time.Minute)}},
		{name: "no candles", interval: kline.FiveMin, end: start.Add(15 * time.Minute)},
		{name: "first interval missing", interval: kline.FiveMin, end: start.Add(15 * time.Minute), candles: at(5*time.Minute, 10*time.Minute), want: [][]time.Time{times(5*time.Minute, 10*time.Minute)}},
		{name: "middle interval missing", interval: kline.FiveMin, end: start.Add(15 * time.Minute), candles: at(0, 10*time.Minute), want: [][]time.Time{times(0), times(10 * time.Minute)}},
		{name: "last interval missing", interval: kline.FiveMin, end: start.Add(15 * time.Minute), candles: at(0, 5*time.Minute), want: [][]time.Time{times(0, 5*time.Minute)}},
		{name: "two candles in one interval", interval: kline.FiveMin, end: start.Add(15 * time.Minute), candles: at(0, 2*time.Minute, 10*time.Minute), want: [][]time.Time{times(0), times(10 * time.Minute)}},
		{name: "candles outside the intervals", interval: kline.FiveMin, end: start.Add(15 * time.Minute), candles: at(-time.Minute, 15*time.Minute)},
		{name: "zero time", interval: kline.FiveMin, end: start.Add(15 * time.Minute), candles: append([]kline.Candle{{}}, at(5*time.Minute)...), want: [][]time.Time{times(5 * time.Minute)}},
		{name: "calendar months", interval: kline.OneMonth, end: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), candles: at(0, 30*24*time.Hour), want: [][]time.Time{times(0)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h, err := kline.CalculateCandleDateRanges(start, tc.end, tc.interval, 2)
			require.NoError(t, err, "CalculateCandleDateRanges must not error")
			var got [][]time.Time
			for _, run := range candleRuns(h, tc.candles) {
				runTimes := make([]time.Time, len(run))
				for i := range run {
					runTimes[i] = run[i].Time
				}
				got = append(got, runTimes)
			}
			assert.Equal(t, tc.want, got, "candleRuns should return the first candle of each interval in runs of consecutive intervals")
		})
	}
}

func TestIntervalsWithoutData(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name        string
		hasData     []bool
		wantMissing int
	}{
		{name: "no data", hasData: []bool{false, false, false}, wantMissing: 3},
		{name: "every interval", hasData: []bool{true, true, true}},
		{name: "one interval in a later range", hasData: []bool{true, true, false}, wantMissing: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h, err := kline.CalculateCandleDateRanges(start, start.Add(15*time.Minute), kline.FiveMin, 2)
			require.NoError(t, err, "CalculateCandleDateRanges must not error")
			var i int
			for x := range h.Ranges {
				for y := range h.Ranges[x].Intervals {
					h.Ranges[x].Intervals[y].HasData = tc.hasData[i]
					i++
				}
			}
			missing, total := intervalsWithoutData(h)
			assert.Equal(t, tc.wantMissing, missing, "intervalsWithoutData should count the intervals without data")
			assert.Equal(t, 3, total, "intervalsWithoutData should count every interval")
		})
	}
}

func TestRangeResult(t *testing.T) {
	t.Parallel()
	result := func(status dataHistoryStatus) DataHistoryJobResult { return DataHistoryJobResult{Status: status} }
	for _, tc := range []struct {
		name         string
		results      []DataHistoryJobResult
		wantDecided  bool
		wantComplete bool
	}{
		{name: "no results"},
		{name: "failed", results: []DataHistoryJobResult{result(dataHistoryStatusFailed)}},
		{name: "complete after a failure", results: []DataHistoryJobResult{result(dataHistoryStatusFailed), result(dataHistoryStatusComplete)}, wantDecided: true, wantComplete: true},
		{name: "complete before a failure", results: []DataHistoryJobResult{result(dataHistoryStatusComplete), result(dataHistoryStatusFailed)}, wantDecided: true, wantComplete: true},
		{name: "issues found after a failure", results: []DataHistoryJobResult{result(dataHistoryStatusFailed), result(dataHistoryIntervalIssuesFound)}, wantDecided: true},
		{name: "complete after issues found", results: []DataHistoryJobResult{result(dataHistoryIntervalIssuesFound), result(dataHistoryStatusComplete)}, wantDecided: true, wantComplete: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			decided, complete := rangeResult(tc.results)
			assert.Equal(t, tc.wantDecided, decided, "rangeResult should report whether the results decide the range")
			assert.Equal(t, tc.wantComplete, complete, "rangeResult should report whether the range completed")
		})
	}
}

func TestDataHistoryJobResultCoversRange(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	sydney := time.FixedZone("AEDT", 11*60*60)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		want       bool
	}{
		{name: "the range's bounds", start: start, end: end, want: true},
		{name: "the range's bounds in another time zone", start: start.In(sydney), end: end.In(sydney), want: true},
		{name: "a later start within the same second", start: start.Add(500 * time.Millisecond), end: end},
		{name: "an earlier end", start: start, end: end.Add(-time.Minute)},
		{name: "a later end", start: start, end: end.Add(time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := DataHistoryJobResult{IntervalStartDate: tc.start, IntervalEndDate: tc.end}
			assert.Equal(t, tc.want, r.coversRange(start, end), "coversRange should report only a result recorded for the range")
		})
	}
}

func TestDataHistoryJobRangeResults(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)
	own := []DataHistoryJobResult{
		{IntervalStartDate: start, IntervalEndDate: end, Status: dataHistoryStatusFailed},
		{IntervalStartDate: start, IntervalEndDate: end, Status: dataHistoryStatusComplete},
	}
	shorter := DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: start.Add(time.Hour), Status: dataHistoryStatusComplete}
	recorded := []DataHistoryJobResult{own[0], shorter, own[1]}
	job := &DataHistoryJob{Results: map[int64][]DataHistoryJobResult{start.Unix(): slices.Clone(recorded)}}
	assert.Equal(t, own, job.rangeResults(start, end), "rangeResults should return only the results recorded for the range, in order")
	assert.Equal(t, recorded, job.Results[start.Unix()], "rangeResults should leave the job's results as recorded")
	assert.Empty(t, job.rangeResults(end, end.Add(time.Hour)), "rangeResults should return nothing for a range without results")
}

func TestDataHistoryJobMarkRangeIssuesFound(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)
	failed := DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: end, Status: dataHistoryStatusFailed}
	completed := DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: end, Status: dataHistoryStatusComplete}
	shorter := DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: start.Add(time.Hour), Status: dataHistoryStatusFailed}
	job := &DataHistoryJob{Results: map[int64][]DataHistoryJobResult{start.Unix(): {failed, shorter, completed}}}
	job.markRangeIssuesFound(start, end)
	issues := failed
	issues.Status = dataHistoryIntervalIssuesFound
	assert.Equal(t, []DataHistoryJobResult{issues, shorter, issues}, job.Results[start.Unix()], "markRangeIssuesFound should mark every result recorded for the range and leave the others")
}

func TestSetHasDataByCandleTime(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	withData := func(offset time.Duration) kline.Candle {
		return kline.Candle{Time: start.Add(offset), Open: 1, High: 1, Low: 1, Close: 1, Volume: 1}
	}
	for _, tc := range []struct {
		name    string
		candles []kline.Candle
		want    []bool
	}{
		{name: "no candles", want: []bool{false, false, false}},
		{name: "every interval", candles: []kline.Candle{withData(0), withData(5 * time.Minute), withData(10 * time.Minute)}, want: []bool{true, true, true}},
		{name: "gap before later batches", candles: []kline.Candle{withData(5 * time.Minute), withData(10 * time.Minute)}, want: []bool{false, true, true}},
		{name: "candle away from an interval start", candles: []kline.Candle{withData(2 * time.Minute)}, want: []bool{false, false, false}},
		{name: "candle a nanosecond after an interval start", candles: []kline.Candle{withData(time.Nanosecond)}, want: []bool{false, false, false}},
		{name: "starts shared by candles with and without data", candles: []kline.Candle{{Time: start}, withData(0), withData(5 * time.Minute), {Time: start.Add(5 * time.Minute)}}, want: []bool{true, true, false}},
		{name: "candle after the last interval", candles: []kline.Candle{withData(15 * time.Minute)}, want: []bool{false, false, false}},
		{name: "candle without data", candles: []kline.Candle{{Time: start, Open: -1, High: -1, Low: -1, Close: -1}}, want: []bool{false, false, false}},
		{name: "not a number", candles: []kline.Candle{{Time: start, Open: math.NaN(), High: -1, Low: -1, Close: -1}}, want: []bool{true, false, false}},
		{name: "open only", candles: []kline.Candle{{Time: start, Open: 1}}, want: []bool{true, false, false}},
		{name: "high only", candles: []kline.Candle{{Time: start, High: 1}}, want: []bool{true, false, false}},
		{name: "low only", candles: []kline.Candle{{Time: start, Low: 1}}, want: []bool{true, false, false}},
		{name: "close only", candles: []kline.Candle{{Time: start, Close: 1}}, want: []bool{true, false, false}},
		{name: "volume only", candles: []kline.Candle{{Time: start, Volume: 1}}, want: []bool{true, false, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h, err := kline.CalculateCandleDateRanges(start, start.Add(15*time.Minute), kline.FiveMin, 2)
			require.NoError(t, err, "CalculateCandleDateRanges must not error")
			setHasDataByCandleTime(h.Ranges, tc.candles)
			var got []bool
			for _, r := range h.Ranges {
				for _, interval := range r.Intervals {
					got = append(got, interval.HasData)
				}
			}
			assert.Equal(t, tc.want, got, "each interval's HasData should match its saved candle")
		})
	}

	t.Run("only the ranges given", func(t *testing.T) {
		t.Parallel()
		h, err := kline.CalculateCandleDateRanges(start, start.Add(20*time.Minute), kline.FiveMin, 1)
		require.NoError(t, err, "CalculateCandleDateRanges must not error")
		h.Ranges[1].Intervals[0].HasData = true
		h.Ranges[2].Intervals[0].HasData = true
		setHasDataByCandleTime(h.Ranges[1:], []kline.Candle{withData(0), {Time: start.Add(5 * time.Minute)}, withData(15 * time.Minute)})
		assert.Equal(t, [][]bool{{false}, {false}, {true}, {true}}, rangesHasData(h), "setHasDataByCandleTime should set each interval of the ranges given that a candle starts and leave the rest as they were")
	})
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

	t.Run("no existing candle data for primary interval", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		calls := 0
		var gotInterval kline.Interval
		mgr.candleLoader = func(
			exchangeName string,
			pair currency.Pair,
			assetType asset.Item,
			interval kline.Interval,
			start time.Time,
			end time.Time,
		) (*kline.Item, error) {
			_ = exchangeName
			_ = pair
			_ = assetType
			_ = start
			_ = end
			calls++
			gotInterval = interval
			return nil, candle.ErrNoCandleDataFound
		}
		tt := time.Now().Truncate(kline.OneHour.Duration())
		job := &DataHistoryJob{
			Nickname:           "TestCompareJobsToDataNoPrimaryCandleData",
			Exchange:           testExchange,
			Asset:              asset.Spot,
			Pair:               currency.NewBTCUSD(),
			StartDate:          tt.Add(-time.Minute * 5),
			EndDate:            tt,
			Interval:           kline.OneMin,
			ConversionInterval: kline.FiveMin,
		}
		// ErrNoCandleDataFound is recoverable and should mark intervals as missing.
		err := mgr.compareJobsToData(job)
		require.NoError(t, err, "ErrNoCandleDataFound must be non-fatal for primary interval")
		require.Equal(t, 1, calls)
		require.Equal(t, kline.OneMin, gotInterval)
		require.NotNil(t, job.rangeHolder)
		require.NotEmpty(t, job.rangeHolder.Ranges)
		for i := range job.rangeHolder.Ranges {
			for j := range job.rangeHolder.Ranges[i].Intervals {
				assert.False(t, job.rangeHolder.Ranges[i].Intervals[j].HasData)
			}
		}
	})

	t.Run("no existing candle data for conversion interval", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		calls := 0
		var gotInterval kline.Interval
		mgr.candleLoader = func(
			exchangeName string,
			pair currency.Pair,
			assetType asset.Item,
			interval kline.Interval,
			start time.Time,
			end time.Time,
		) (*kline.Item, error) {
			_ = exchangeName
			_ = pair
			_ = assetType
			_ = start
			_ = end
			calls++
			gotInterval = interval
			return nil, candle.ErrNoCandleDataFound
		}
		tt := time.Now().Truncate(kline.OneHour.Duration())
		job := &DataHistoryJob{
			Nickname:           "TestCompareJobsToDataNoConversionCandleData",
			Exchange:           testExchange,
			Asset:              asset.Spot,
			Pair:               currency.NewBTCUSD(),
			StartDate:          tt.Add(-time.Minute * 5),
			EndDate:            tt,
			Interval:           kline.OneMin,
			DataType:           dataHistoryConvertCandlesDataType,
			ConversionInterval: kline.FiveMin,
		}
		err := mgr.compareJobsToData(job)
		require.NoError(t, err, "ErrNoCandleDataFound must be non-fatal for conversion interval")
		require.Equal(t, 1, calls)
		require.Equal(t, kline.FiveMin, gotInterval)
		require.NotNil(t, job.rangeHolder)
		require.NotEmpty(t, job.rangeHolder.Ranges)
		for i := range job.rangeHolder.Ranges {
			for j := range job.rangeHolder.Ranges[i].Intervals {
				assert.False(t, job.rangeHolder.Ranges[i].Intervals[j].HasData)
			}
		}
	})

	t.Run("resume after saving converted candles", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		job := &DataHistoryJob{
			StartDate:          start,
			EndDate:            start.Add(15 * time.Minute),
			Interval:           kline.OneMin,
			ConversionInterval: kline.FiveMin,
			DataType:           dataHistoryConvertCandlesDataType,
			RequestSizeLimit:   2,
		}
		var saved *kline.Item
		mgr.candleLoader = func(_ string, _ currency.Pair, _ asset.Item, interval kline.Interval, from, until time.Time) (*kline.Item, error) {
			if interval == job.ConversionInterval {
				return saved, nil
			}
			return &kline.Item{Interval: interval, Candles: generateCandles(int(until.Sub(from)/interval.Duration()), from, interval)}, nil
		}
		mgr.candleSaver = func(item *kline.Item, _ bool) (uint64, error) {
			saved = item
			return uint64(len(item.Candles)), nil
		}
		result, err := mgr.convertCandleData(job, start, start.Add(10*time.Minute))
		require.NoError(t, err, "convertCandleData must not error")
		require.Equal(t, dataHistoryStatusComplete, result.Status, "convertCandleData must complete the first batch")
		require.Len(t, saved.Candles, 2, "convertCandleData must save two five-minute candles")
		require.NoError(t, mgr.compareJobsToData(job), "compareJobsToData must prepare the partly converted job")
		require.Len(t, job.rangeHolder.Ranges, 2, "the job must have two batches")
		for _, interval := range job.rangeHolder.Ranges[0].Intervals {
			assert.True(t, interval.HasData, "each saved candle should be recognised")
		}
		pending := job.rangeHolder.Ranges[1]
		require.Len(t, pending.Intervals, 1, "one candle must remain")
		assert.Equal(t, start.Add(10*time.Minute), pending.Start.Time, "the final batch should start after the saved candles")
		assert.False(t, pending.Intervals[0].HasData, "the final candle should be pending")
	})

	t.Run("resume after a failed first batch", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		job := &DataHistoryJob{
			StartDate:          start,
			EndDate:            start.Add(15 * time.Minute),
			Interval:           kline.OneMin,
			ConversionInterval: kline.FiveMin,
			DataType:           dataHistoryConvertCandlesDataType,
			RequestSizeLimit:   1,
		}
		source := generateCandles(10, start.Add(5*time.Minute), kline.OneMin)
		var saved []kline.Candle
		mgr.candleLoader = func(_ string, _ currency.Pair, _ asset.Item, interval kline.Interval, from, until time.Time) (*kline.Item, error) {
			if interval == job.ConversionInterval {
				return loadSavedCandles(saved, interval, from, until)
			}
			return loadSavedCandles(source, interval, from, until)
		}
		mgr.candleSaver = func(item *kline.Item, _ bool) (uint64, error) {
			saved = append(saved, item.Candles...)
			return uint64(len(item.Candles)), nil
		}
		require.NoError(t, mgr.compareJobsToData(job), "compareJobsToData must prepare the new job")
		statuses := make([]dataHistoryStatus, 0, len(job.rangeHolder.Ranges))
		for _, r := range job.rangeHolder.Ranges {
			result, err := mgr.convertCandleData(job, r.Start.Time, r.End.Time)
			require.NoError(t, err, "convertCandleData must not error")
			statuses = append(statuses, result.Status)
		}
		require.Equal(t, []dataHistoryStatus{dataHistoryStatusFailed, dataHistoryStatusComplete, dataHistoryStatusComplete}, statuses, "only the batch before the source candles must fail")
		require.NoError(t, mgr.compareJobsToData(job), "compareJobsToData must prepare a job whose first batch failed")
		var hasData []bool
		for _, r := range job.rangeHolder.Ranges {
			for _, interval := range r.Intervals {
				hasData = append(hasData, interval.HasData)
			}
		}
		assert.Equal(t, []bool{false, true, true}, hasData, "only the converted batches should be recognised")
	})

	t.Run("resume after results for shorter batches", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		job := &DataHistoryJob{
			StartDate:          start,
			EndDate:            start.Add(25 * time.Minute),
			Interval:           kline.OneMin,
			ConversionInterval: kline.FiveMin,
			DataType:           dataHistoryConvertCandlesDataType,
			RequestSizeLimit:   5,
			RunBatchLimit:      1,
			MaxRetryAttempts:   1,
			Results: map[int64][]DataHistoryJobResult{
				start.Unix(): {{IntervalStartDate: start, IntervalEndDate: start.Add(5 * time.Minute), Status: dataHistoryStatusComplete}},
			},
		}
		source := generateCandles(25, start, kline.OneMin)
		saved := generateCandles(1, start, kline.FiveMin)
		mgr.candleLoader = func(_ string, _ currency.Pair, _ asset.Item, interval kline.Interval, from, until time.Time) (*kline.Item, error) {
			if interval == job.ConversionInterval {
				return loadSavedCandles(saved, interval, from, until)
			}
			return loadSavedCandles(source, interval, from, until)
		}
		mgr.candleSaver = func(item *kline.Item, _ bool) (uint64, error) {
			saved = append(saved, item.Candles...)
			return uint64(len(item.Candles)), nil
		}
		require.NoError(t, mgr.compareJobsToData(job), "compareJobsToData must prepare the job")
		assert.Empty(t, job.Results, "compareJobsToData should set aside the result of the shorter batch")
		require.NoError(t, mgr.runDataJob(t.Context(), job, nil), "runDataJob must not error")
		converted := make([]int64, 0, len(saved)-1)
		for _, c := range saved[1:] {
			converted = append(converted, c.Time.Unix())
		}
		assert.Equal(t, []int64{start.Unix(), start.Add(5 * time.Minute).Unix(), start.Add(10 * time.Minute).Unix(), start.Add(15 * time.Minute).Unix(), start.Add(20 * time.Minute).Unix()}, converted, "runDataJob should convert the whole batch")
		assert.Equal(t, dataHistoryStatusComplete, job.Status, "runDataJob should complete the job once its batch converts")
	})

	for _, tc := range []struct {
		name         string
		resultEnd    time.Duration
		resultStatus dataHistoryStatus
		wantHasData  []bool
	}{
		{name: "trade range with its own completion", resultEnd: 3 * time.Hour, resultStatus: dataHistoryStatusComplete, wantHasData: []bool{true, true, true}},
		{name: "trade range beside a completion for a shorter range", resultEnd: time.Hour, resultStatus: dataHistoryStatusComplete, wantHasData: []bool{false, false, false}},
		{name: "trade range whose own attempts found issues", resultEnd: 3 * time.Hour, resultStatus: dataHistoryIntervalIssuesFound, wantHasData: []bool{false, false, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mgr, _ := createDHM(t)
			start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			job := &DataHistoryJob{
				StartDate:        start,
				EndDate:          start.Add(3 * time.Hour),
				Interval:         kline.OneHour,
				DataType:         dataHistoryTradeDataType,
				RequestSizeLimit: 3,
				Results: map[int64][]DataHistoryJobResult{start.Unix(): {
					{IntervalStartDate: start, IntervalEndDate: start.Add(tc.resultEnd), Status: tc.resultStatus},
				}},
			}
			require.NoError(t, mgr.compareJobsToData(job), "compareJobsToData must not error")
			var hasData []bool
			for _, r := range job.rangeHolder.Ranges {
				for _, interval := range r.Intervals {
					hasData = append(hasData, interval.HasData)
				}
			}
			assert.Equal(t, tc.wantHasData, hasData, "compareJobsToData should mark a trade range only from a completion recorded for it")
		})
	}

	t.Run("candle series with a gap", func(t *testing.T) {
		t.Parallel()
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		gap := generateCandles(2, start.Add(5*time.Minute), kline.FiveMin)
		for _, tc := range []struct {
			name  string
			saved []kline.Candle
		}{
			{name: "first interval missing", saved: gap},
			{name: "candle away from the first interval start", saved: slices.Concat(generateCandles(1, start.Add(2*time.Minute), kline.FiveMin), gap)},
		} {
			for _, dataType := range []dataHistoryDataType{dataHistoryCandleDataType, dataHistoryCandleValidationDataType, dataHistoryCandleValidationSecondarySourceType, dataHistoryConvertTradesDataType, dataHistoryConvertCandlesDataType} {
				for _, overwrite := range []bool{false, true} {
					name := tc.name + "/" + dataType.String() + "/keep existing data"
					if overwrite {
						name = tc.name + "/" + dataType.String() + "/overwrite existing data"
					}
					t.Run(name, func(t *testing.T) {
						t.Parallel()
						mgr, _ := createDHM(t)
						mgr.candleLoader = func(_ string, _ currency.Pair, _ asset.Item, interval kline.Interval, from, until time.Time) (*kline.Item, error) {
							return loadSavedCandles(tc.saved, interval, from, until)
						}
						job := &DataHistoryJob{
							StartDate:             start,
							EndDate:               start.Add(15 * time.Minute),
							Interval:              kline.FiveMin,
							ConversionInterval:    kline.FiveMin,
							DataType:              dataType,
							RequestSizeLimit:      2,
							OverwriteExistingData: overwrite,
						}
						if dataType == dataHistoryConvertCandlesDataType {
							job.Interval = kline.OneMin
						}
						require.NoError(t, mgr.compareJobsToData(job), "compareJobsToData must prepare a job whose candle series has a gap")
						var hasData []bool
						for _, r := range job.rangeHolder.Ranges {
							for _, interval := range r.Intervals {
								hasData = append(hasData, interval.HasData)
							}
						}
						assert.Equal(t, []bool{false, true, true}, hasData, "only the intervals a saved candle starts should have data")
					})
				}
			}
		}
	})

	t.Run("start between conversion intervals", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		job := &DataHistoryJob{
			StartDate:          start.Add(time.Minute),
			EndDate:            start.Add(16 * time.Minute),
			Interval:           kline.OneMin,
			ConversionInterval: kline.FiveMin,
			DataType:           dataHistoryConvertCandlesDataType,
			RequestSizeLimit:   1,
		}
		saved := generateCandles(3, start, kline.FiveMin)
		mgr.candleLoader = func(_ string, _ currency.Pair, _ asset.Item, interval kline.Interval, from, until time.Time) (*kline.Item, error) {
			return loadSavedCandles(saved, interval, from, until)
		}
		require.NoError(t, mgr.compareJobsToData(job), "compareJobsToData must prepare the job")
		require.True(t, job.rangeHolder.Start.Time.Equal(start), "the ranges must start on the conversion interval")
		var hasData []bool
		for _, r := range job.rangeHolder.Ranges {
			for _, interval := range r.Intervals {
				hasData = append(hasData, interval.HasData)
			}
		}
		assert.Equal(t, []bool{true, true, true}, hasData, "each saved candle should be recognised")
	})

	for _, dataType := range []dataHistoryDataType{
		dataHistoryCandleDataType, dataHistoryCandleValidationDataType, dataHistoryCandleValidationSecondarySourceType,
		dataHistoryConvertTradesDataType, dataHistoryConvertCandlesDataType,
	} {
		t.Run("each job's own stored candles for "+dataType.String(), func(t *testing.T) {
			t.Parallel()
			mgr, _ := createDHM(t)
			start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			// BTC/USD has its second hour stored and BTC/USDT its first
			mgr.candleLoader = func(_ string, pair currency.Pair, _ asset.Item, interval kline.Interval, from, until time.Time) (*kline.Item, error) {
				hour := 1
				if pair.Equal(currency.NewBTCUSDT()) {
					hour = 0
				}
				return loadSavedCandles(generateCandles(1, start.Add(time.Duration(hour)*time.Hour), interval), interval, from, until)
			}
			newJob := func(pair currency.Pair) *DataHistoryJob {
				return &DataHistoryJob{
					Nickname: pair.String(), Exchange: testExchange, Asset: asset.Spot, Pair: pair, StartDate: start, EndDate: start.Add(2 * time.Hour),
					Interval: kline.OneHour, ConversionInterval: kline.OneHour, DataType: dataType, RequestSizeLimit: 1,
				}
			}
			btcusd, btcusdt := newJob(currency.NewBTCUSD()), newJob(currency.NewBTCUSDT())
			require.NoError(t, mgr.compareJobsToData(btcusd, btcusdt), "compareJobsToData must not error")
			assert.Equal(t, [][]bool{{false}, {true}}, rangesHasData(btcusd.rangeHolder), "compareJobsToData should mark the first job's own stored candles")
			assert.Equal(t, [][]bool{{true}, {false}}, rangesHasData(btcusdt.rangeHolder), "compareJobsToData should mark the second job's own stored candles")
		})
	}

	m.started.Store(false)
	err = m.compareJobsToData(dhj)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	err = m.compareJobsToData(dhj)
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

// rangesHasData returns the HasData of each range's intervals
func rangesHasData(h *kline.IntervalRangeHolder) [][]bool {
	hasData := make([][]bool, len(h.Ranges))
	for i := range h.Ranges {
		for _, interval := range h.Ranges[i].Intervals {
			hasData[i] = append(hasData[i], interval.HasData)
		}
	}
	return hasData
}

// loadSavedCandles returns the candles between two times inclusively, as the candle repository's query does
func loadSavedCandles(candles []kline.Candle, interval kline.Interval, from, until time.Time) (*kline.Item, error) {
	var found []kline.Candle
	for i := range candles {
		if !candles[i].Time.Before(from) && !candles[i].Time.After(until) {
			found = append(found, candles[i])
		}
	}
	if len(found) == 0 {
		return nil, candle.ErrNoCandleDataFound
	}
	return &kline.Item{Interval: interval, Candles: found}, nil
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
			err = m.runJob(t.Context(), tc)
			assert.ErrorIs(t, err, errJobInvalid)

			rh := tc.rangeHolder
			tc.Status = dataHistoryStatusActive
			tc.rangeHolder = nil
			err = m.runJob(t.Context(), tc)
			assert.ErrorIs(t, err, errJobInvalid)

			tc.rangeHolder = rh
			err = m.runJob(t.Context(), tc)
			assert.NoError(t, err)
		})
	}
	var badM *DataHistoryManager
	err := badM.runJob(t.Context(), nil)
	assert.ErrorIs(t, err, ErrNilSubsystem)

	badM = &DataHistoryManager{}
	err = badM.runJob(t.Context(), nil)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)
}

func TestRunJobSkipsPersistenceOnCancelledContext(t *testing.T) {
	t.Parallel()
	m, dbJob := createDHM(t)
	m.tradeSaver = dataHistoryTradeSaver
	m.candleSaver = dataHistoryCandleSaver
	m.tradeLoader = dataHistoryTraderLoader

	tt := time.Now().Truncate(kline.OneMin.Duration())
	tc := &DataHistoryJob{
		Nickname:  "TestRunJobSkipsPersistenceOnCancelledContext",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		StartDate: tt.Add(-kline.OneMin.Duration()),
		EndDate:   tt,
		Interval:  kline.OneMin,
		DataType:  dataHistoryCandleDataType,
	}
	require.NoError(t, m.UpsertJob(tc, false))
	tc.Status = dataHistoryStatusActive

	jobService := &countingDataHistoryJobService{job: dbJob}
	resultService := &countingDataHistoryJobResultService{}
	m.jobDB = jobService
	m.jobResultDB = resultService

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := m.runJob(ctx, tc)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, jobService.upsertCalls)
	assert.Equal(t, 0, resultService.upsertCalls)
}

func TestRunJobsReturnsNilOnCancelledContext(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := m.runJobs(ctx)
	require.NoError(t, err)
}

func TestRunJobSkipsPersistenceWhenStoppedDuringRun(t *testing.T) {
	t.Parallel()
	m, dbJob := createDHM(t)
	m.tradeSaver = dataHistoryTradeSaver
	m.tradeLoader = dataHistoryTraderLoader
	m.candleSaver = func(i *kline.Item, _ bool) (uint64, error) {
		m.started.Store(false)
		if i == nil {
			return 0, nil
		}
		return uint64(len(i.Candles)), nil
	}

	tt := time.Now().Truncate(kline.OneHour.Duration())
	tc := &DataHistoryJob{
		Nickname:           "TestRunJobSkipsPersistenceWhenStoppedDuringRun",
		Exchange:           testExchange,
		Asset:              asset.Spot,
		Pair:               currency.NewBTCUSDT(),
		StartDate:          tt.Add(-kline.OneHour.Duration()),
		EndDate:            tt,
		Interval:           kline.FifteenMin,
		DataType:           dataHistoryConvertCandlesDataType,
		ConversionInterval: kline.OneHour,
	}
	require.NoError(t, m.UpsertJob(tc, false))
	tc.Status = dataHistoryStatusActive

	jobService := &countingDataHistoryJobService{job: dbJob}
	resultService := &countingDataHistoryJobResultService{}
	m.jobDB = jobService
	m.jobResultDB = resultService

	err := m.runJob(t.Context(), tc)
	if err != nil {
		require.ErrorIs(t, err, ErrSubSystemNotStarted)
	}
	assert.Equal(t, 0, jobService.upsertCalls)
	assert.Equal(t, 0, resultService.upsertCalls)
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

	m.started.Store(false)
	_, err = m.GenerateJobSummary("TestGenerateJobSummary")
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	_, err = m.GenerateJobSummary("TestGenerateJobSummary")
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestRunJobs(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	err := m.runJobs(t.Context())
	assert.NoError(t, err)

	m.started.Store(false)
	err = m.runJobs(t.Context())
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	err = m.runJobs(t.Context())
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestConverters(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	id := uuid.NewV4()

	id2 := uuid.NewV4()

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
		exchangeManager:            em,
		candleLoader:               dataHistoryCandleLoader,
		interval:                   time.NewTicker(time.Minute),
		verbose:                    true,
		maxResultInsertions:        defaultMaxResultInsertions,
	}
	m.started.Store(true)
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
	_, err := m.processCandleData(t.Context(), nil, nil, time.Time{}, time.Time{}, 0)
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
	_, err = m.processCandleData(t.Context(), j, nil, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	assert.NoError(t, err)

	exch.SetDefaults()
	fakeExchange := dhmExchange{
		IBotExchange: exch,
	}
	_, err = m.processCandleData(t.Context(), j, exch, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)

	m.candleSaver = dataHistoryCandleSaver
	j.rangeHolder, err = kline.CalculateCandleDateRanges(j.StartDate, j.EndDate, j.Interval, 1337)
	if err != nil {
		t.Error(err)
	}
	r, err := m.processCandleData(t.Context(), j, fakeExchange, j.StartDate, j.EndDate, 0)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusComplete {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusComplete)
	}
	r, err = m.processCandleData(t.Context(), j, exch, j.StartDate, j.EndDate, 0)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	withData := func(hour int) kline.Candle {
		return kline.Candle{Time: start.Add(time.Duration(hour) * time.Hour), Open: 1, High: 1, Low: 1, Close: 1, Volume: 1}
	}
	for _, tc := range []struct {
		name        string
		stored      [][]bool
		reply       []kline.Candle
		wantStatus  dataHistoryStatus
		wantResult  string
		wantHasData [][]bool
	}{
		{name: "range after the job's first", stored: [][]bool{{false, false}, {false, false}, {false, false}}, reply: []kline.Candle{withData(2), withData(3)}, wantStatus: dataHistoryStatusComplete, wantHasData: [][]bool{{false, false}, {true, true}, {false, false}}},
		{name: "reply with candles beyond the range", stored: [][]bool{{false, false}, {false, false}, {false, false}}, reply: []kline.Candle{withData(0), withData(2), withData(3), withData(4)}, wantStatus: dataHistoryStatusComplete, wantHasData: [][]bool{{false, false}, {true, true}, {false, false}}},
		{name: "interval the reply leaves out", stored: [][]bool{{false, false}, {false, false}, {false, false}}, reply: []kline.Candle{withData(2)}, wantStatus: dataHistoryStatusFailed, wantResult: "missing data from 2025-01-01 03:00:00 UTC - 2025-01-01 04:00:00 UTC. ", wantHasData: [][]bool{{false, false}, {true, false}, {false, false}}},
		{name: "first interval the reply leaves out", stored: [][]bool{{false, false}, {false, false}, {false, false}}, reply: []kline.Candle{withData(3)}, wantStatus: dataHistoryStatusFailed, wantResult: "missing data from 2025-01-01 02:00:00 UTC - 2025-01-01 03:00:00 UTC. ", wantHasData: [][]bool{{false, false}, {false, true}, {false, false}}},
		{name: "stored interval the reply leaves out", stored: [][]bool{{false, false}, {false, true}, {false, false}}, reply: []kline.Candle{withData(2)}, wantStatus: dataHistoryStatusComplete, wantHasData: [][]bool{{false, false}, {true, true}, {false, false}}},
		{name: "stored interval the reply has without data", stored: [][]bool{{false, false}, {false, true}, {false, false}}, reply: []kline.Candle{withData(2), {Time: start.Add(3 * time.Hour)}}, wantStatus: dataHistoryStatusFailed, wantResult: "missing data from 2025-01-01 03:00:00 UTC - 2025-01-01 04:00:00 UTC. ", wantHasData: [][]bool{{false, false}, {true, false}, {false, false}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, _ := createDHM(t)
			m.candleSaver = dataHistoryCandleSaver
			job := &DataHistoryJob{Exchange: testExchange, Asset: asset.Spot, Pair: currency.NewBTCUSD(), StartDate: start, EndDate: start.Add(6 * time.Hour), Interval: kline.OneHour}
			var err error
			job.rangeHolder, err = kline.CalculateCandleDateRanges(job.StartDate, job.EndDate, job.Interval, 2)
			require.NoError(t, err, "CalculateCandleDateRanges must not error")
			for i, stored := range tc.stored {
				for x := range stored {
					job.rangeHolder.Ranges[i].Intervals[x].HasData = stored[x]
				}
			}
			var requested []time.Time
			exch := candleExchange{candles: func(requestStart time.Time) []kline.Candle {
				requested = append(requested, requestStart)
				return slices.Clone(tc.reply)
			}}
			r, err := m.processCandleData(t.Context(), job, exch, start.Add(2*time.Hour), start.Add(4*time.Hour), 1)
			require.NoError(t, err, "processCandleData must not error for a range after the job's first")
			assert.Equal(t, []time.Time{start.Add(2 * time.Hour)}, requested, "processCandleData should request the range given")
			assert.Equal(t, tc.wantStatus, r.Status, "processCandleData should judge the range by its own intervals")
			assert.Equal(t, tc.wantResult, r.Result, "processCandleData should name each of the range's intervals without data")
			assert.Equal(t, tc.wantHasData, rangesHasData(job.rangeHolder), "processCandleData should set its own range's intervals from the candles starting them and leave the rest as they were")
		})
	}
}

func TestProcessTradeData(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	_, err := m.processTradeData(t.Context(), nil, nil, time.Time{}, time.Time{}, 0)
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
	_, err = m.processTradeData(t.Context(), j, nil, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	assert.NoError(t, err)

	exch.SetDefaults()
	fakeExchange := dhmExchange{
		IBotExchange: exch,
	}
	_, err = m.processTradeData(t.Context(), j, exch, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)

	j.rangeHolder, err = kline.CalculateCandleDateRanges(j.StartDate, j.EndDate, j.Interval, 1337)
	if err != nil {
		t.Error(err)
	}
	m.tradeSaver = dataHistoryTradeSaver
	r, err := m.processTradeData(t.Context(), j, fakeExchange, j.StartDate, j.EndDate, 0)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}
	r, err = m.processTradeData(t.Context(), j, exch, j.StartDate, j.EndDate, 0)
	assert.NoError(t, err)

	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	// inFirstHour answers a request with two trades in its first hour, as dhmExchange does
	inFirstHour := func(requestStart, _ time.Time) []time.Time {
		return []time.Time{requestStart.Add(time.Minute), requestStart.Add(2 * time.Minute)}
	}
	for _, tc := range []struct {
		name         string
		requestHours uint64
		trades       func(requestStart, requestEnd time.Time) []time.Time
		wantStatus   dataHistoryStatus
		wantResult   string
		wantHasData  [][]bool
		wantSaved    []time.Time
	}{
		{name: "range after the job's first", requestHours: 1, trades: inFirstHour, wantStatus: dataHistoryStatusComplete, wantHasData: [][]bool{{false}, {true}, {false}, {false}}, wantSaved: []time.Time{start.Add(61 * time.Minute), start.Add(62 * time.Minute)}},
		{name: "range after the job's first without trades in an interval", requestHours: 2, trades: inFirstHour, wantStatus: dataHistoryStatusFailed, wantResult: "missing data from 2025-01-01 03:00:00 UTC - 2025-01-01 04:00:00 UTC. ", wantHasData: [][]bool{{false, false}, {true, false}}, wantSaved: []time.Time{start.Add(121 * time.Minute), start.Add(122 * time.Minute)}},
		{
			// The range processed is the job's last, so its end is the job's end
			name:         "trades outside the range",
			requestHours: 2,
			trades: func(requestStart, requestEnd time.Time) []time.Time {
				return []time.Time{requestEnd.In(time.FixedZone("UTC+10", 10*60*60)), requestStart, requestStart.Add(-time.Nanosecond), requestEnd.Add(time.Minute), requestEnd.Add(-time.Nanosecond)}
			},
			wantStatus:  dataHistoryStatusComplete,
			wantHasData: [][]bool{{false, false}, {true, true}},
			wantSaved:   []time.Time{start.Add(2 * time.Hour), start.Add(4*time.Hour - time.Nanosecond)},
		},
		{
			name:         "trades only outside the range",
			requestHours: 1,
			trades: func(requestStart, requestEnd time.Time) []time.Time {
				return []time.Time{requestStart.Add(-time.Minute), requestEnd}
			},
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "could not convert candles to trades: no trades supplied. ",
			wantHasData: [][]bool{{false}, {false}, {false}, {false}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, _ := createDHM(t)
			var saved []time.Time
			m.tradeSaver = func(trades ...trade.Data) error {
				for i := range trades {
					saved = append(saved, trades[i].Timestamp)
				}
				return nil
			}
			job := &DataHistoryJob{Exchange: testExchange, Asset: asset.Spot, Pair: currency.NewBTCUSD(), StartDate: start, EndDate: start.Add(4 * time.Hour), Interval: kline.OneHour}
			var err error
			job.rangeHolder, err = kline.CalculateCandleDateRanges(job.StartDate, job.EndDate, job.Interval, tc.requestHours)
			require.NoError(t, err, "CalculateCandleDateRanges must not error")
			second := job.rangeHolder.Ranges[1]
			r, err := m.processTradeData(t.Context(), job, tradeExchange{trades: tc.trades}, second.Start.Time, second.End.Time, 1)
			require.NoError(t, err, "processTradeData must not error for a range after the job's first")
			assert.Equal(t, tc.wantStatus, r.Status, "processTradeData should judge the range by its own intervals")
			assert.Equal(t, tc.wantResult, r.Result, "processTradeData should name each of the range's intervals without trades")
			assert.Equal(t, tc.wantHasData, rangesHasData(job.rangeHolder), "processTradeData should set only its own range's intervals")
			assert.Equal(t, tc.wantSaved, saved, "processTradeData should save only the trades within the range")
		})
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

func TestConvertCandleData(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	m.candleSaver = dataHistoryCandleSaver
	_, err := m.convertCandleData(nil, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errNilJob)

	today := time.Now().UTC().Truncate(kline.OneDay.Duration())
	j := &DataHistoryJob{
		Nickname:           "",
		Exchange:           testExchange,
		Asset:              asset.Spot,
		Pair:               currency.NewBTCUSDT(),
		StartDate:          today.Add(-kline.OneDay.Duration()),
		EndDate:            today,
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

	t.Run("saves the converted candles", func(t *testing.T) {
		t.Parallel()
		m, _ := createDHM(t)
		var saved *kline.Item
		m.candleSaver = func(candles *kline.Item, _ bool) (uint64, error) {
			saved = candles
			return 0, nil
		}
		job := *j
		_, err := m.convertCandleData(&job, job.StartDate, job.EndDate)
		require.NoError(t, err, "convertCandleData must not error")
		require.NotNil(t, saved, "convertCandleData must save candles")
		assert.Equal(t, kline.OneDay, saved.Interval, "saved candles should be at the conversion interval")
	})

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	candleAt := func(offset time.Duration, price, volume float64) kline.Candle {
		return kline.Candle{Time: start.Add(offset), Open: price, High: price, Low: price, Close: price, Volume: volume}
	}
	minutes := func(from, to time.Duration, price, volume float64) []kline.Candle {
		var candles []kline.Candle
		for offset := from; offset < to; offset += time.Minute {
			candles = append(candles, candleAt(offset, price, volume))
		}
		return candles
	}
	rising := func(from, to time.Duration) []kline.Candle {
		var candles []kline.Candle
		for offset := from; offset < to; offset += time.Minute {
			k := float64(offset / time.Minute)
			candles = append(candles, kline.Candle{Time: start.Add(offset), Open: 10 + k, High: 20 + k, Low: 5 + k, Close: 15 + k, Volume: 1 + k})
		}
		return candles
	}
	type ohlcv [5]float64
	for _, tc := range []struct {
		name        string
		source      []kline.Candle
		existing    []kline.Candle
		overwrite   bool
		begin       time.Duration
		end         time.Duration
		wantStatus  dataHistoryStatus
		wantResult  string
		wantSaves   [][]time.Time
		wantTargets map[time.Duration]ohlcv
		conversion  kline.Interval
		loadFails   bool
		sourceFails bool
		saveFails   bool
		stopOnRead  bool
		wantErr     error
	}{
		{
			name:        "source candles cover the batch",
			source:      rising(0, 10*time.Minute),
			end:         10 * time.Minute,
			wantStatus:  dataHistoryStatusComplete,
			wantSaves:   [][]time.Time{{start, start.Add(5 * time.Minute)}},
			wantTargets: map[time.Duration]ohlcv{0: {10, 24, 5, 19, 15}, 5 * time.Minute: {15, 29, 10, 24, 40}},
		},
		{
			name:        "source candles cover part of the batch",
			source:      minutes(0, 5*time.Minute, 1337, 1337),
			end:         25 * time.Minute,
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "missing 4 of 5 converted candles",
			wantSaves:   [][]time.Time{{start}},
			wantTargets: map[time.Duration]ohlcv{0: {1337, 1337, 1337, 1337, 6685}},
		},
		{
			name:        "source candles without volume",
			source:      append(minutes(0, 5*time.Minute, 1, 0), minutes(5*time.Minute, 10*time.Minute, 1337, 1337)...),
			end:         10 * time.Minute,
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "missing 1 of 2 converted candles",
			wantSaves:   [][]time.Time{{start.Add(5 * time.Minute)}},
			wantTargets: map[time.Duration]ohlcv{5 * time.Minute: {1337, 1337, 1337, 1337, 6685}},
		},
		{
			name:        "source candles starting after the batch start",
			source:      append(append(minutes(time.Minute, 5*time.Minute, 1, 1), minutes(5*time.Minute, 10*time.Minute, 2, 2)...), candleAt(10*time.Minute, 900, 900)),
			existing:    []kline.Candle{candleAt(10*time.Minute, 777, 777)},
			overwrite:   true,
			end:         10 * time.Minute,
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "missing 1 of 2 converted candles",
			wantSaves:   [][]time.Time{{start.Add(5 * time.Minute)}},
			wantTargets: map[time.Duration]ohlcv{5 * time.Minute: {2, 2, 2, 2, 10}, 10 * time.Minute: {777, 777, 777, 777, 777}},
		},
		{
			name:        "interior source gap",
			source:      append(minutes(0, 2*time.Minute, 1, 1), minutes(3*time.Minute, 10*time.Minute, 1, 1)...),
			end:         10 * time.Minute,
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "missing 1 of 2 converted candles",
			wantSaves:   [][]time.Time{{start.Add(5 * time.Minute)}},
			wantTargets: map[time.Duration]ohlcv{5 * time.Minute: {1, 1, 1, 1, 5}},
		},
		{
			name:        "source candle without data in the batch",
			source:      slices.Concat(minutes(0, 2*time.Minute, 1, 1), []kline.Candle{{Time: start.Add(2 * time.Minute)}}, minutes(3*time.Minute, 10*time.Minute, 1, 1)),
			end:         10 * time.Minute,
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "missing 1 of 2 converted candles",
			wantSaves:   [][]time.Time{{start.Add(5 * time.Minute)}},
			wantTargets: map[time.Duration]ohlcv{5 * time.Minute: {1, 1, 1, 1, 5}},
		},
		{
			name:        "source candle without data under a saved converted candle",
			source:      slices.Concat(minutes(0, 2*time.Minute, 1, 1), []kline.Candle{{Time: start.Add(2 * time.Minute)}}, minutes(3*time.Minute, 10*time.Minute, 1, 1)),
			existing:    []kline.Candle{candleAt(0, 777, 777)},
			overwrite:   true,
			end:         10 * time.Minute,
			wantStatus:  dataHistoryStatusComplete,
			wantSaves:   [][]time.Time{{start.Add(5 * time.Minute)}},
			wantTargets: map[time.Duration]ohlcv{0: {777, 777, 777, 777, 777}, 5 * time.Minute: {1, 1, 1, 1, 5}},
		},
		{
			name:        "source candle with a price but no volume",
			source:      slices.Concat(minutes(0, 2*time.Minute, 1, 1), []kline.Candle{candleAt(2*time.Minute, 1, 0)}, minutes(3*time.Minute, 5*time.Minute, 1, 1)),
			end:         5 * time.Minute,
			wantStatus:  dataHistoryStatusComplete,
			wantSaves:   [][]time.Time{{start}},
			wantTargets: map[time.Duration]ohlcv{0: {1, 1, 1, 1, 4}},
		},
		{
			name:        "source candle away from its interval start",
			source:      append(append(minutes(0, 3*time.Minute, 1, 1), candleAt(3*time.Minute+30*time.Second, 1, 1)), minutes(4*time.Minute, 10*time.Minute, 1, 1)...),
			end:         10 * time.Minute,
			wantStatus:  dataHistoryStatusComplete,
			wantSaves:   [][]time.Time{{start, start.Add(5 * time.Minute)}},
			wantTargets: map[time.Duration]ohlcv{0: {1, 1, 1, 1, 5}, 5 * time.Minute: {1, 1, 1, 1, 5}},
		},
		{
			name:        "interval without a converted candle between two with",
			source:      append(append(minutes(0, 5*time.Minute, 1337, 1337), minutes(5*time.Minute, 10*time.Minute, 1, 0)...), minutes(10*time.Minute, 15*time.Minute, 1337, 1337)...),
			existing:    []kline.Candle{candleAt(5*time.Minute, 777, 777)},
			overwrite:   true,
			end:         15 * time.Minute,
			wantStatus:  dataHistoryStatusComplete,
			wantSaves:   [][]time.Time{{start}, {start.Add(10 * time.Minute)}},
			wantTargets: map[time.Duration]ohlcv{0: {1337, 1337, 1337, 1337, 6685}, 5 * time.Minute: {777, 777, 777, 777, 777}, 10 * time.Minute: {1337, 1337, 1337, 1337, 6685}},
		},
		{
			name:        "converted candle another job saved",
			source:      minutes(5*time.Minute, 10*time.Minute, 1337, 1337),
			existing:    []kline.Candle{candleAt(0, 777, 777)},
			end:         10 * time.Minute,
			wantStatus:  dataHistoryStatusComplete,
			wantSaves:   [][]time.Time{{start.Add(5 * time.Minute)}},
			wantTargets: map[time.Duration]ohlcv{0: {777, 777, 777, 777, 777}, 5 * time.Minute: {1337, 1337, 1337, 1337, 6685}},
		},
		{
			name:        "stored candle without data kept",
			source:      minutes(0, 10*time.Minute, 1337, 1337),
			existing:    []kline.Candle{candleAt(0, 0, 0)},
			end:         10 * time.Minute,
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "missing 1 of 2 converted candles",
			wantSaves:   [][]time.Time{{start, start.Add(5 * time.Minute)}},
			wantTargets: map[time.Duration]ohlcv{0: {}, 5 * time.Minute: {1337, 1337, 1337, 1337, 6685}},
		},
		{
			name:        "no converted candles",
			source:      minutes(0, 10*time.Minute, 1, 0),
			end:         10 * time.Minute,
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "missing 2 of 2 converted candles",
			wantTargets: map[time.Duration]ohlcv{},
		},
		{
			name:      "saved candles cannot be read",
			source:    minutes(0, 10*time.Minute, 1337, 1337),
			end:       10 * time.Minute,
			loadFails: true,
			wantErr:   errExpectedTestError,
			wantSaves: [][]time.Time{{start, start.Add(5 * time.Minute)}},
		},
		{
			name:        "source candles cannot be read",
			source:      minutes(0, 10*time.Minute, 1337, 1337),
			end:         10 * time.Minute,
			sourceFails: true,
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "could not get candles",
			wantTargets: map[time.Duration]ohlcv{},
		},
		{
			name:        "conversion interval not a whole number of source intervals",
			source:      minutes(0, 10*time.Minute, 1337, 1337),
			end:         9 * time.Minute,
			conversion:  kline.Interval(90 * time.Second),
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "could not convert candles",
			wantTargets: map[time.Duration]ohlcv{},
		},
		{
			name:        "saving the converted candles fails",
			source:      minutes(0, 10*time.Minute, 1337, 1337),
			end:         10 * time.Minute,
			saveFails:   true,
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "could not save results",
			wantTargets: map[time.Duration]ohlcv{},
		},
		{
			name:       "manager stopped during the conversion",
			source:     minutes(0, 10*time.Minute, 1337, 1337),
			end:        10 * time.Minute,
			stopOnRead: true,
			wantErr:    ErrSubSystemNotStarted,
		},
		{
			name:        "batch rounding to no conversion interval",
			source:      minutes(3*time.Minute, 8*time.Minute, 1337, 1337),
			begin:       3 * time.Minute,
			end:         7 * time.Minute,
			wantStatus:  dataHistoryStatusFailed,
			wantResult:  "could not calculate converted candle intervals",
			wantTargets: map[time.Duration]ohlcv{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, _ := createDHM(t)
			store := &candleStore{}
			if len(tc.existing) > 0 {
				_, err := store.save(&kline.Item{Interval: kline.FiveMin, Candles: tc.existing}, false)
				require.NoError(t, err, "storing the existing converted candles must not error")
				store.saves = nil
			}
			m.candleLoader = func(exchangeName string, pair currency.Pair, a asset.Item, interval kline.Interval, from, until time.Time) (*kline.Item, error) {
				if interval == kline.OneMin {
					if tc.stopOnRead {
						m.started.Store(false)
					}
					if tc.sourceFails {
						return nil, errExpectedTestError
					}
					// Source candles are served as given, as rows imported from a file keep their own times
					return loadSavedCandles(tc.source, interval, from, until)
				}
				if tc.loadFails {
					return nil, errExpectedTestError
				}
				return store.load(exchangeName, pair, a, interval, from, until)
			}
			m.candleSaver = func(item *kline.Item, overwrite bool) (uint64, error) {
				if tc.saveFails {
					return 0, errExpectedTestError
				}
				return store.save(item, overwrite)
			}
			job := DataHistoryJob{
				Exchange:              testExchange,
				Asset:                 asset.Spot,
				Pair:                  currency.NewBTCUSDT(),
				Interval:              kline.OneMin,
				ConversionInterval:    kline.FiveMin,
				OverwriteExistingData: tc.overwrite,
			}
			if tc.conversion != 0 {
				job.ConversionInterval = tc.conversion
			}
			r, err := m.convertCandleData(&job, start.Add(tc.begin), start.Add(tc.end))
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "convertCandleData should return the error")
				assert.Nil(t, r, "convertCandleData should return no result with the error")
				assert.Equal(t, tc.wantSaves, store.saves, "convertCandleData should save only before the error")
				return
			}
			require.NoError(t, err, "convertCandleData must not error")
			assert.Equal(t, tc.wantStatus, r.Status, "convertCandleData should record whether the store holds every converted candle")
			if tc.wantResult == "" {
				assert.Empty(t, r.Result, "convertCandleData should give no reason for a complete batch")
			} else {
				assert.Contains(t, r.Result, tc.wantResult, "convertCandleData should give the reason the batch failed")
			}
			assert.Equal(t, tc.wantSaves, store.saves, "convertCandleData should save converted candles within the batch in runs of consecutive intervals")
			targets := make(map[time.Duration]ohlcv, len(store.candles[kline.FiveMin]))
			for _, c := range store.candles[kline.FiveMin] {
				targets[c.Time.Sub(start)] = ohlcv{c.Open, c.High, c.Low, c.Close, c.Volume}
			}
			assert.Equal(t, tc.wantTargets, targets, "convertCandleData should store each converted candle from its own interval's source candles")
		})
	}
}

// candleStore keeps candles by interval and aligned start time, as the candle repository does: an overwriting save
// first deletes every stored candle between its first and last, and other saves keep a stored candle in place
type candleStore struct {
	candles map[kline.Interval]map[int64]kline.Candle
	saves   [][]time.Time
}

func (s *candleStore) save(item *kline.Item, overwrite bool) (uint64, error) {
	if s.candles == nil {
		s.candles = make(map[kline.Interval]map[int64]kline.Candle)
	}
	stored := s.candles[item.Interval]
	if stored == nil {
		stored = make(map[int64]kline.Candle)
		s.candles[item.Interval] = stored
	}
	times := make([]time.Time, len(item.Candles))
	for i := range item.Candles {
		times[i] = item.Candles[i].Time.Truncate(item.Interval.Duration())
	}
	if overwrite && len(times) > 0 {
		for key := range stored {
			if t := time.Unix(0, key); !t.Before(times[0]) && !t.After(times[len(times)-1]) {
				delete(stored, key)
			}
		}
	}
	for i := range item.Candles {
		if _, ok := stored[times[i].UnixNano()]; ok {
			continue
		}
		c := item.Candles[i]
		c.Time = times[i]
		stored[times[i].UnixNano()] = c
	}
	s.saves = append(s.saves, times)
	return uint64(len(item.Candles)), nil
}

func (s *candleStore) load(_ string, _ currency.Pair, _ asset.Item, interval kline.Interval, from, until time.Time) (*kline.Item, error) {
	candles := make([]kline.Candle, 0, len(s.candles[interval]))
	for _, c := range s.candles[interval] {
		candles = append(candles, c)
	}
	slices.SortFunc(candles, func(a, b kline.Candle) int { return a.Time.Compare(b.Time) })
	return loadSavedCandles(candles, interval, from, until)
}

func TestValidationIntervalLength(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		interval         kline.Interval
		requestSizeLimit uint64
		want             time.Duration
		wantErr          error
	}{
		{name: "length fits", interval: kline.OneHour, requestSizeLimit: 500, want: 500 * time.Hour},
		{name: "longest length", interval: 1, requestSizeLimit: math.MaxInt64, want: math.MaxInt64},
		{name: "longest interval once", interval: math.MaxInt64, requestSizeLimit: 1, want: math.MaxInt64},
		{name: "largest product that fits", interval: 2, requestSizeLimit: math.MaxInt64 / 2, want: math.MaxInt64 - 1},
		{name: "one nanosecond too long", interval: 2, requestSizeLimit: math.MaxInt64/2 + 1, wantErr: kline.ErrInvalidInterval},
		{name: "product wraps to a negative duration", interval: kline.OneHour, requestSizeLimit: 2_562_048, wantErr: kline.ErrInvalidInterval},
		{name: "product wraps to a positive duration", interval: 1<<62 + 1, requestSizeLimit: 4, wantErr: kline.ErrInvalidInterval},
		{name: "limit beyond a duration", interval: 1, requestSizeLimit: math.MaxInt64 + 1, wantErr: kline.ErrInvalidInterval},
		{name: "limit beyond a duration wraps to a positive duration", interval: 2, requestSizeLimit: math.MaxInt64 + 2, wantErr: kline.ErrInvalidInterval},
		{name: "negative interval with a limit beyond a duration", interval: -kline.OneHour, requestSizeLimit: math.MaxUint64 - 1, wantErr: kline.ErrInvalidInterval},
		{name: "negative interval", interval: -kline.OneHour, requestSizeLimit: 2, wantErr: kline.ErrInvalidInterval},
		{name: "zero interval", requestSizeLimit: 2, wantErr: kline.ErrInvalidInterval},
		{name: "zero limit", interval: kline.OneHour, wantErr: kline.ErrInvalidInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := validationIntervalLength(tc.interval, tc.requestSizeLimit)
			require.ErrorIs(t, err, tc.wantErr, "validationIntervalLength must return the expected error")
			assert.Equal(t, tc.want, got, "validationIntervalLength should return the expected length")
		})
	}
}

func TestRunValidationJob(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	start := time.Now().Add(-time.Hour * 24)
	for _, tc := range []struct {
		name             string
		interval         kline.Interval
		requestSizeLimit uint64
	}{
		{name: "length wraps to a negative duration", interval: kline.OneHour, requestSizeLimit: 2_562_048},
		{name: "length wraps to a positive duration", interval: 1<<62 + 1, requestSizeLimit: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			job := &DataHistoryJob{
				Nickname:  "TestRunValidationJob",
				StartDate: start,
				// Short enough to step through quickly were a wrapped positive length accepted
				EndDate:          start.Add(40 * time.Nanosecond),
				Interval:         tc.interval,
				RequestSizeLimit: tc.requestSizeLimit,
			}
			assert.ErrorIs(t, m.runValidationJob(t.Context(), job, nil), kline.ErrInvalidInterval, "runValidationJob should reject an interval length that overflows")
		})
	}

	// rangeResults returns results with the given statuses for the range from start to end, as validation records them
	rangeResults := func(start, end time.Time, statuses ...dataHistoryStatus) []DataHistoryJobResult {
		results := make([]DataHistoryJobResult, len(statuses))
		for i := range statuses {
			results[i] = DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: end, Status: statuses[i]}
		}
		return results
	}
	hourResults := func(start time.Time, statuses ...dataHistoryStatus) []DataHistoryJobResult {
		return rangeResults(start, start.Add(time.Hour), statuses...)
	}

	t.Run("interval retried after a failure completes", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		job := &DataHistoryJob{
			Nickname:         "TestRunValidationJobRetried",
			StartDate:        from,
			EndDate:          from.Add(2 * time.Hour),
			Interval:         kline.OneHour,
			RequestSizeLimit: 1,
			// No batch processes, leaving only the completion check to run
			RunBatchLimit:    0,
			MaxRetryAttempts: 3,
			Results: map[int64][]DataHistoryJobResult{
				from.Unix():                hourResults(from, dataHistoryStatusFailed, dataHistoryStatusComplete),
				from.Add(time.Hour).Unix(): hourResults(from.Add(time.Hour), dataHistoryStatusFailed, dataHistoryStatusComplete),
			},
		}
		require.NoError(t, mgr.runValidationJob(t.Context(), job, nil), "runValidationJob must not error")
		assert.Equal(t, dataHistoryStatusComplete, job.Status, "runValidationJob should complete a job whose intervals each completed on a retry")
	})

	t.Run("interval with issues found", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		job := &DataHistoryJob{
			Nickname:         "TestRunValidationJobIssuesFound",
			StartDate:        from,
			EndDate:          from.Add(2 * time.Hour),
			Interval:         kline.OneHour,
			RequestSizeLimit: 1,
			// No batch processes, leaving only the completion check to run
			RunBatchLimit:    0,
			MaxRetryAttempts: 3,
			Results: map[int64][]DataHistoryJobResult{
				from.Unix():                hourResults(from, dataHistoryStatusComplete),
				from.Add(time.Hour).Unix(): hourResults(from.Add(time.Hour), dataHistoryStatusFailed, dataHistoryIntervalIssuesFound),
			},
		}
		require.NoError(t, mgr.runValidationJob(t.Context(), job, nil), "runValidationJob must not error")
		assert.Equal(t, dataHistoryIntervalIssuesFound, job.Status, "runValidationJob should finish a job with an interval's issues found")
	})

	newValidationJob := func(from time.Time, runBatchLimit uint64, results map[int64][]DataHistoryJobResult) *DataHistoryJob {
		return &DataHistoryJob{
			Nickname:         "TestRunValidationJobIntervals",
			Exchange:         testExchange,
			Asset:            asset.Spot,
			Pair:             currency.NewBTCUSD(),
			StartDate:        from,
			EndDate:          from.Add(3 * time.Hour),
			Interval:         kline.OneHour,
			RequestSizeLimit: 1,
			RunBatchLimit:    runBatchLimit,
			MaxRetryAttempts: 3,
			Results:          results,
		}
	}
	matchingCandles := func(t *testing.T, mgr *DataHistoryManager) (exchange.IBotExchange, *[]time.Time) {
		t.Helper()
		candle := func(at time.Time) []kline.Candle {
			return []kline.Candle{{Time: at, Open: 1, High: 2, Low: 1, Close: 2, Volume: 1}}
		}
		var validated []time.Time
		mgr.candleSaver = dataHistoryCandleSaver
		mgr.candleLoader = func(_ string, _ currency.Pair, _ asset.Item, interval kline.Interval, from, _ time.Time) (*kline.Item, error) {
			validated = append(validated, from.UTC())
			return &kline.Item{Interval: interval, Candles: candle(from)}, nil
		}
		return candleExchange{candles: candle}, &validated
	}

	t.Run("each cycle validates the next intervals", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		exch, validated := matchingCandles(t, mgr)
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		job := newValidationJob(from, 1, make(map[int64][]DataHistoryJobResult))
		for range 3 {
			require.NoError(t, mgr.runValidationJob(t.Context(), job, exch), "runValidationJob must not error")
		}
		assert.Equal(t, []time.Time{from, from.Add(time.Hour), from.Add(2 * time.Hour)}, *validated, "runValidationJob should validate each interval once, in order")
		assert.Equal(t, dataHistoryStatusComplete, job.Status, "runValidationJob should complete a job once each interval validates")
	})

	t.Run("completion on the last allowed attempt kept", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		// No batch processes, leaving only the retry limit and completion checks to run
		job := newValidationJob(from, 0, map[int64][]DataHistoryJobResult{
			from.Unix(): hourResults(from, dataHistoryStatusFailed, dataHistoryStatusFailed, dataHistoryStatusComplete),
		})
		require.NoError(t, mgr.runValidationJob(t.Context(), job, nil), "runValidationJob must not error")
		assert.Equal(t, hourResults(from, dataHistoryStatusFailed, dataHistoryStatusFailed, dataHistoryStatusComplete), job.Results[from.Unix()], "runValidationJob should keep a completion on the last allowed attempt")
	})

	t.Run("interval out of attempts marked and another retried", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		exch, validated := matchingCandles(t, mgr)
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		job := newValidationJob(from, 1, map[int64][]DataHistoryJobResult{
			from.Unix():                    hourResults(from, dataHistoryStatusFailed, dataHistoryStatusFailed, dataHistoryStatusFailed),
			from.Add(time.Hour).Unix():     hourResults(from.Add(time.Hour), dataHistoryStatusFailed, dataHistoryStatusFailed),
			from.Add(2 * time.Hour).Unix(): hourResults(from.Add(2*time.Hour), dataHistoryStatusComplete),
		})
		require.NoError(t, mgr.runValidationJob(t.Context(), job, exch), "runValidationJob must not error")
		assert.Equal(t, []time.Time{from.Add(time.Hour)}, *validated, "runValidationJob should retry only the interval with attempts left")
		assert.Equal(t, hourResults(from, dataHistoryIntervalIssuesFound, dataHistoryIntervalIssuesFound, dataHistoryIntervalIssuesFound), job.Results[from.Unix()], "runValidationJob should mark an interval out of attempts as having issues")
		assert.Equal(t, dataHistoryIntervalIssuesFound, job.Status, "runValidationJob should finish a job with an interval out of attempts")
	})

	t.Run("interval past a lowered retry limit not retried", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		exch, validated := matchingCandles(t, mgr)
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		// Two attempts that found issues, recorded before an edit lowered the job's retry limit to one
		job := newValidationJob(from, 1, map[int64][]DataHistoryJobResult{
			from.Unix():                    hourResults(from, dataHistoryIntervalIssuesFound, dataHistoryIntervalIssuesFound),
			from.Add(time.Hour).Unix():     hourResults(from.Add(time.Hour), dataHistoryStatusComplete),
			from.Add(2 * time.Hour).Unix(): hourResults(from.Add(2*time.Hour), dataHistoryStatusComplete),
		})
		job.MaxRetryAttempts = 1
		require.NoError(t, mgr.runValidationJob(t.Context(), job, exch), "runValidationJob must not error")
		assert.Empty(t, *validated, "runValidationJob should not retry an interval past its retry limit")
		assert.Equal(t, dataHistoryIntervalIssuesFound, job.Status, "runValidationJob should finish a job with an interval past its retry limit")
	})

	t.Run("results from an earlier request size count only for unchanged ranges", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		exch, validated := matchingCandles(t, mgr)
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		// Recorded while the job requested an hour at a time, before an edit had it request two
		earlier := map[int64][]DataHistoryJobResult{
			from.Unix():                    hourResults(from, dataHistoryStatusFailed, dataHistoryStatusFailed, dataHistoryStatusComplete),
			from.Add(time.Hour).Unix():     hourResults(from.Add(time.Hour), dataHistoryStatusComplete),
			from.Add(2 * time.Hour).Unix(): hourResults(from.Add(2*time.Hour), dataHistoryStatusComplete),
		}
		results := make(map[int64][]DataHistoryJobResult, len(earlier))
		for key, recorded := range earlier {
			results[key] = slices.Clone(recorded)
		}
		job := newValidationJob(from, 1, results)
		job.RequestSizeLimit = 2
		require.NoError(t, mgr.runValidationJob(t.Context(), job, exch), "runValidationJob must not error")
		assert.Equal(t, []time.Time{from}, *validated, "runValidationJob should validate the range whose earlier results cover another range, as with no attempts used")
		require.Len(t, job.Results[from.Unix()], 4, "runValidationJob must add one result beside the earlier ones")
		assert.Equal(t, earlier[from.Unix()], job.Results[from.Unix()][:3], "runValidationJob should leave the earlier results as recorded")
		assert.WithinDuration(t, from.Add(2*time.Hour), job.Results[from.Unix()][3].IntervalEndDate, 0, "runValidationJob should record a result covering the first range")
		assert.Equal(t, earlier[from.Add(time.Hour).Unix()], job.Results[from.Add(time.Hour).Unix()], "runValidationJob should leave results under an earlier range's start as recorded")
		assert.Equal(t, earlier[from.Add(2*time.Hour).Unix()], job.Results[from.Add(2*time.Hour).Unix()], "runValidationJob should keep the result covering the last range, which ends with the job")
		assert.Equal(t, dataHistoryStatusComplete, job.Status, "runValidationJob should complete the job once a result covers each range")
	})

	t.Run("range out of attempts marks only its own results", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		// No batch processes, leaving only the retry limit and completion checks to run
		// Beside the range's own failures, completions for the first hour and for ranges starting or ending within a
		// second of its bounds, as earlier request sizes and intervals record them
		others := slices.Concat(hourResults(from, dataHistoryStatusComplete),
			rangeResults(from.Add(500*time.Millisecond), from.Add(2*time.Hour), dataHistoryStatusComplete),
			rangeResults(from, from.Add(2*time.Hour+500*time.Millisecond), dataHistoryStatusComplete))
		job := newValidationJob(from, 0, map[int64][]DataHistoryJobResult{
			from.Unix():                    slices.Concat(others, rangeResults(from, from.Add(2*time.Hour), dataHistoryStatusFailed, dataHistoryStatusFailed, dataHistoryStatusFailed)),
			from.Add(2 * time.Hour).Unix(): hourResults(from.Add(2*time.Hour), dataHistoryStatusComplete),
		})
		job.RequestSizeLimit = 2
		require.NoError(t, mgr.runValidationJob(t.Context(), job, nil), "runValidationJob must not error")
		want := slices.Concat(others, rangeResults(from, from.Add(2*time.Hour), dataHistoryIntervalIssuesFound, dataHistoryIntervalIssuesFound, dataHistoryIntervalIssuesFound))
		assert.Equal(t, want, job.Results[from.Unix()], "runValidationJob should mark only the results covering the range out of attempts")
		assert.Equal(t, dataHistoryIntervalIssuesFound, job.Status, "runValidationJob should finish a job with a range out of attempts")
	})

	t.Run("last range out of attempts marks its own results", func(t *testing.T) {
		t.Parallel()
		mgr, _ := createDHM(t)
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		// No batch processes, leaving only the retry limit and completion checks to run
		job := newValidationJob(from, 0, map[int64][]DataHistoryJobResult{
			from.Unix():                    rangeResults(from, from.Add(2*time.Hour), dataHistoryStatusComplete),
			from.Add(2 * time.Hour).Unix(): hourResults(from.Add(2*time.Hour), dataHistoryStatusFailed, dataHistoryStatusFailed, dataHistoryStatusFailed),
		})
		job.RequestSizeLimit = 2
		require.NoError(t, mgr.runValidationJob(t.Context(), job, nil), "runValidationJob must not error")
		assert.Equal(t, hourResults(from.Add(2*time.Hour), dataHistoryIntervalIssuesFound, dataHistoryIntervalIssuesFound, dataHistoryIntervalIssuesFound), job.Results[from.Add(2*time.Hour).Unix()], "runValidationJob should mark the results of a last range ending with the job")
		assert.Equal(t, dataHistoryIntervalIssuesFound, job.Status, "runValidationJob should finish a job whose last range is out of attempts")
	})
}

func TestRunDataJob(t *testing.T) {
	t.Parallel()

	t.Run("range retried after a failure completes", func(t *testing.T) {
		t.Parallel()
		m, _ := createDHM(t)
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		h, err := kline.CalculateCandleDateRanges(start, start.Add(10*time.Minute), kline.FiveMin, 2)
		require.NoError(t, err, "CalculateCandleDateRanges must not error")
		for i := range h.Ranges[0].Intervals {
			h.Ranges[0].Intervals[i].HasData = true
		}
		end := start.Add(10 * time.Minute)
		job := &DataHistoryJob{
			Nickname:         "TestRunDataJobRetried",
			DataType:         dataHistoryConvertCandlesDataType,
			RunBatchLimit:    1,
			MaxRetryAttempts: 3,
			rangeHolder:      h,
			Results: map[int64][]DataHistoryJobResult{start.Unix(): {
				{IntervalStartDate: start, IntervalEndDate: end, Status: dataHistoryStatusFailed},
				{IntervalStartDate: start, IntervalEndDate: end, Status: dataHistoryStatusComplete},
			}},
		}
		require.NoError(t, m.runDataJob(t.Context(), job, nil), "runDataJob must not error")
		assert.Equal(t, dataHistoryStatusComplete, job.Status, "runDataJob should complete a job whose range completed on a retry")
	})

	t.Run("stored range with only a failed result", func(t *testing.T) {
		t.Parallel()
		m, _ := createDHM(t)
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		h, err := kline.CalculateCandleDateRanges(start, start.Add(10*time.Minute), kline.FiveMin, 2)
		require.NoError(t, err, "CalculateCandleDateRanges must not error")
		for i := range h.Ranges[0].Intervals {
			h.Ranges[0].Intervals[i].HasData = true
		}
		failed := DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: start.Add(10 * time.Minute), Status: dataHistoryStatusFailed}
		job := &DataHistoryJob{
			Nickname:         "TestRunDataJobStoredAfterFailure",
			DataType:         dataHistoryConvertCandlesDataType,
			RunBatchLimit:    1,
			MaxRetryAttempts: 3,
			rangeHolder:      h,
			Results:          map[int64][]DataHistoryJobResult{start.Unix(): {failed}},
		}
		require.NoError(t, m.runDataJob(t.Context(), job, nil), "runDataJob must not error")
		assert.Equal(t, dataHistoryStatusComplete, job.Status, "runDataJob should complete a job whose failed range has every candle stored")
		results := job.Results[start.Unix()]
		require.Len(t, results, 2, "runDataJob must record a completion beside the failure")
		assert.Equal(t, failed, results[0], "runDataJob should keep the failure")
		assert.Equal(t, dataHistoryStatusComplete, results[1].Status, "runDataJob should record the range as complete")
	})

	for _, tc := range []struct {
		name       string
		stored     bool
		wantStatus dataHistoryStatus
	}{
		{name: "range with issues found", wantStatus: dataHistoryIntervalIssuesFound},
		{name: "stored range with issues found", stored: true, wantStatus: dataHistoryStatusComplete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, _ := createDHM(t)
			start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			h, err := kline.CalculateCandleDateRanges(start, start.Add(10*time.Minute), kline.FiveMin, 1)
			require.NoError(t, err, "CalculateCandleDateRanges must not error")
			h.Ranges[0].Intervals[0].HasData = true
			h.Ranges[1].Intervals[0].HasData = tc.stored
			job := &DataHistoryJob{
				Nickname:         "TestRunDataJobIssuesFound",
				DataType:         dataHistoryConvertCandlesDataType,
				RunBatchLimit:    1,
				MaxRetryAttempts: 3,
				rangeHolder:      h,
				Results: map[int64][]DataHistoryJobResult{
					start.Unix(): {{IntervalStartDate: start, IntervalEndDate: start.Add(5 * time.Minute), Status: dataHistoryStatusComplete}},
					start.Add(5 * time.Minute).Unix(): {
						{IntervalStartDate: start.Add(5 * time.Minute), IntervalEndDate: start.Add(10 * time.Minute), Status: dataHistoryStatusFailed},
						{IntervalStartDate: start.Add(5 * time.Minute), IntervalEndDate: start.Add(10 * time.Minute), Status: dataHistoryIntervalIssuesFound},
					},
				},
			}
			require.NoError(t, m.runDataJob(t.Context(), job, nil), "runDataJob must not error")
			assert.Equal(t, tc.wantStatus, job.Status, "runDataJob should finish with issues found only while a range lacks data")
		})
	}

	// candleJob returns a candle job over three hours requesting the hours given at a time, whose store holds the hours
	// given, beside the results given, and the exchange it requests candles from
	candleJob := func(t *testing.T, requestHours uint64, storedHours []int, fetchErr error, results ...DataHistoryJobResult) (*DataHistoryManager, *DataHistoryJob, *rangeRequestExchange) {
		t.Helper()
		m, _ := createDHM(t)
		m.candleSaver = dataHistoryCandleSaver
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		stored := make([]kline.Candle, 0, len(storedHours))
		for _, hour := range storedHours {
			stored = append(stored, generateCandles(1, start.Add(time.Duration(hour)*time.Hour), kline.OneHour)...)
		}
		m.candleLoader = func(_ string, _ currency.Pair, _ asset.Item, interval kline.Interval, from, until time.Time) (*kline.Item, error) {
			return loadSavedCandles(stored, interval, from, until)
		}
		job := &DataHistoryJob{
			Nickname:         "TestRunDataJobEdited",
			Exchange:         testExchange,
			Asset:            asset.Spot,
			Pair:             currency.NewBTCUSD(),
			StartDate:        start,
			EndDate:          start.Add(3 * time.Hour),
			Interval:         kline.OneHour,
			DataType:         dataHistoryCandleDataType,
			RequestSizeLimit: requestHours,
			RunBatchLimit:    1,
			MaxRetryAttempts: 3,
			Results:          map[int64][]DataHistoryJobResult{start.Unix(): results},
		}
		require.NoError(t, m.compareJobsToData(job), "compareJobsToData must not error")
		return m, job, &rangeRequestExchange{err: fetchErr}
	}
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	// Recorded before an edit changed how many hours the job requests at a time
	hourComplete := DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: start.Add(time.Hour), Status: dataHistoryStatusComplete}
	threeHourFailed := DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: start.Add(3 * time.Hour), Status: dataHistoryStatusFailed}

	for _, tc := range []struct {
		name         string
		requestHours uint64
		storedHours  []int
		fetchErr     error
		results      []DataHistoryJobResult
		wantRequests []string
		wantStatus   dataHistoryStatus
	}{
		{name: "widened range beside a result for a shorter range", requestHours: 3, storedHours: []int{0, 2}, results: []DataHistoryJobResult{hourComplete}, wantRequests: []string{"00:00-03:00"}, wantStatus: dataHistoryStatusComplete},
		{name: "widened range failing beside a result for a shorter range", requestHours: 3, storedHours: []int{0, 2}, fetchErr: errExpectedTestError, results: []DataHistoryJobResult{hourComplete}, wantRequests: []string{"00:00-03:00"}, wantStatus: dataHistoryStatusActive},
		{name: "widened range stored beside a result for a shorter range", requestHours: 3, storedHours: []int{0, 1, 2}, results: []DataHistoryJobResult{hourComplete}, wantStatus: dataHistoryStatusComplete},
		{name: "narrowed range beside failures for a longer range", requestHours: 1, storedHours: []int{1, 2}, results: []DataHistoryJobResult{threeHourFailed, threeHourFailed, threeHourFailed}, wantRequests: []string{"00:00-01:00"}, wantStatus: dataHistoryStatusComplete},
		{name: "range after the first fetched between stored ranges", requestHours: 1, storedHours: []int{0, 2}, results: []DataHistoryJobResult{hourComplete}, wantRequests: []string{"01:00-02:00"}, wantStatus: dataHistoryStatusComplete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, job, exch := candleJob(t, tc.requestHours, tc.storedHours, tc.fetchErr, slices.Clone(tc.results)...)
			require.NoError(t, m.runDataJob(t.Context(), job, exch), "runDataJob must not error")
			assert.Equal(t, tc.wantRequests, exch.requests, "runDataJob should request a range while its own results leave it undecided")
			assert.Equal(t, tc.results, job.Results[start.Unix()][:len(tc.results)], "runDataJob should leave the results recorded for other ranges as they were")
			assert.Equal(t, tc.wantStatus, job.Status, "runDataJob should decide a range from the results recorded for it")
		})
	}

	t.Run("range out of attempts marks only its own results", func(t *testing.T) {
		t.Parallel()
		failed := DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: start.Add(3 * time.Hour), Status: dataHistoryStatusFailed}
		m, job, exch := candleJob(t, 3, []int{0, 2}, nil, hourComplete, failed, failed, failed)
		require.NoError(t, m.runDataJob(t.Context(), job, exch), "runDataJob must not error")
		assert.Empty(t, exch.requests, "runDataJob should not request a range out of attempts")
		issues := failed
		issues.Status = dataHistoryIntervalIssuesFound
		assert.Equal(t, []DataHistoryJobResult{hourComplete, issues, issues, issues}, job.Results[start.Unix()], "runDataJob should mark only the results recorded for the range out of attempts")
		assert.Equal(t, dataHistoryStatusFailed, job.Status, "runDataJob should fail a job whose only range is out of attempts")
	})

	t.Run("range out of attempts marks its own completion too", func(t *testing.T) {
		t.Parallel()
		failed := DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: start.Add(3 * time.Hour), Status: dataHistoryStatusFailed}
		completed := DataHistoryJobResult{IntervalStartDate: start, IntervalEndDate: start.Add(3 * time.Hour), Status: dataHistoryStatusComplete}
		// The completion was recorded before an edit lowered the retry limit to the range's failures
		m, job, exch := candleJob(t, 3, []int{0, 2}, nil, failed, completed)
		job.MaxRetryAttempts = 1
		require.NoError(t, m.runDataJob(t.Context(), job, exch), "runDataJob must not error")
		assert.Empty(t, exch.requests, "runDataJob should not request a range out of attempts")
		issues := failed
		issues.Status = dataHistoryIntervalIssuesFound
		assert.Equal(t, []DataHistoryJobResult{issues, issues}, job.Results[start.Unix()], "runDataJob should mark every result recorded for a range out of attempts, its completion included")
		assert.Equal(t, dataHistoryStatusFailed, job.Status, "runDataJob should fail a job whose only range is out of attempts")
	})

	t.Run("conversion job whose dates fall within conversion intervals", func(t *testing.T) {
		t.Parallel()
		m, _ := createDHM(t)
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		store := &candleStore{}
		// No source candles from beyond the job's dates, which the conversion must not need
		_, err := store.save(&kline.Item{Interval: kline.OneMin, Candles: generateCandles(11, start.Add(2*time.Minute), kline.OneMin)}, false)
		require.NoError(t, err, "storing the source candles must not error")
		m.candleLoader = store.load
		m.candleSaver = store.save
		job := &DataHistoryJob{
			Nickname:           "TestRunDataJobConversionDates",
			Exchange:           testExchange,
			Asset:              asset.Spot,
			Pair:               currency.NewBTCUSD(),
			StartDate:          start.Add(2 * time.Minute),
			EndDate:            start.Add(13 * time.Minute),
			Interval:           kline.OneMin,
			ConversionInterval: kline.FiveMin,
			DataType:           dataHistoryConvertCandlesDataType,
			RequestSizeLimit:   1,
			RunBatchLimit:      3,
			MaxRetryAttempts:   3,
		}
		require.NoError(t, m.validateJob(job), "validateJob must not error")
		require.NoError(t, m.compareJobsToData(job), "compareJobsToData must not error")
		require.NoError(t, m.runDataJob(t.Context(), job, nil), "runDataJob must not error")
		assert.Equal(t, dataHistoryStatusComplete, job.Status, "runDataJob should complete a conversion of every whole conversion interval within the job's dates")
		converted := make([]int64, 0, len(store.candles[kline.FiveMin]))
		for key := range store.candles[kline.FiveMin] {
			converted = append(converted, time.Unix(0, key).Unix())
		}
		assert.Equal(t, []int64{start.Add(5 * time.Minute).Unix()}, converted, "runDataJob should save the one conversion interval within the job's dates")
	})

	t.Run("conversion job resumes batch by batch", func(t *testing.T) {
		t.Parallel()
		m, _ := createDHM(t)
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		store := &candleStore{}
		_, err := store.save(&kline.Item{Interval: kline.OneMin, Candles: generateCandles(15, start, kline.OneMin)}, false)
		require.NoError(t, err, "storing the source candles must not error")
		m.candleLoader = store.load
		m.candleSaver = store.save
		job := &DataHistoryJob{
			Nickname:           "TestRunDataJobConversion",
			StartDate:          start,
			EndDate:            start.Add(15 * time.Minute),
			Interval:           kline.OneMin,
			ConversionInterval: kline.FiveMin,
			DataType:           dataHistoryConvertCandlesDataType,
			RequestSizeLimit:   1,
			RunBatchLimit:      1,
			MaxRetryAttempts:   1,
			Results:            make(map[int64][]DataHistoryJobResult),
		}
		var cycles int
		for ; cycles < 6 && job.Status == dataHistoryStatusActive; cycles++ {
			require.NoError(t, m.compareJobsToData(job), "compareJobsToData must not error")
			require.NoError(t, m.runDataJob(t.Context(), job, nil), "runDataJob must not error")
		}
		assert.Equal(t, 3, cycles, "runDataJob should convert one batch a cycle")
		assert.Equal(t, dataHistoryStatusComplete, job.Status, "runDataJob should complete the job once every batch converts")
		converted := make([]int64, 0, len(store.candles[kline.FiveMin]))
		for key := range store.candles[kline.FiveMin] {
			converted = append(converted, time.Unix(0, key).Unix())
		}
		slices.Sort(converted)
		assert.Equal(t, []int64{start.Unix(), start.Add(5 * time.Minute).Unix(), start.Add(10 * time.Minute).Unix()}, converted, "runDataJob should save every converted candle")
	})
}

func TestValidateCandles(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	m.candleSaver = dataHistoryCandleSaver
	_, err := m.validateCandles(t.Context(), nil, nil, time.Time{}, time.Time{})
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
	_, err = m.validateCandles(t.Context(), j, nil, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	assert.NoError(t, err)

	exch.SetDefaults()
	fakeExchange := dhmExchange{
		IBotExchange: exch,
	}
	_, err = m.validateCandles(t.Context(), j, exch, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)

	j.rangeHolder, err = kline.CalculateCandleDateRanges(j.StartDate, j.EndDate, j.Interval, 1337)
	if err != nil {
		t.Error(err)
	}
	r, err := m.validateCandles(t.Context(), j, fakeExchange, j.StartDate, j.EndDate)
	assert.NoError(t, err)

	if r.Status != dataHistoryIntervalIssuesFound {
		t.Errorf("received %v expected %v", r.Status, dataHistoryIntervalIssuesFound)
	}
	r, err = m.validateCandles(t.Context(), j, exch, j.StartDate, j.EndDate)
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

	jID := uuid.NewV4()

	j.ID = jID.String()
	j.PrerequisiteJobID = ""
	j.PrerequisiteJobNickname = ""
	err = m.SetJobRelationship("", "123")
	assert.NoError(t, err)

	err = m.SetJobRelationship("", "")
	assert.ErrorIs(t, err, errNicknameUnset)

	m.started.Store(false)
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

	m.started.Store(false)
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

	dhm.started.Store(true)
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

type countingDataHistoryJobService struct {
	dataHistoryJobService
	upsertCalls int
}

type countingDataHistoryJobResultService struct {
	dataHistoryJobResultService
	upsertCalls int
}

var (
	jobID     = "00a434e2-8502-4d6b-865f-e4243fd8b5a7"
	startDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)
	endDate   = time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
)

func (d dataHistoryJobService) Upsert(_ ...*datahistoryjob.DataHistoryJob) error {
	return nil
}

func (d *countingDataHistoryJobService) Upsert(_ ...*datahistoryjob.DataHistoryJob) error {
	d.upsertCalls++
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

// jobDBLoadingResultsByNickname leaves a job's results out of GetByID and GetByNickName and returns them from
// GetJobAndAllResults only for the job's nickname, as the database does
type jobDBLoadingResultsByNickname struct {
	datahistoryjob.IDBService
	nickname string
}

func (j jobDBLoadingResultsByNickname) GetByID(id string) (*datahistoryjob.DataHistoryJob, error) {
	return withoutJobResults(j.IDBService.GetByID(id))
}

func (j jobDBLoadingResultsByNickname) GetByNickName(nickname string) (*datahistoryjob.DataHistoryJob, error) {
	return withoutJobResults(j.IDBService.GetByNickName(nickname))
}

func (j jobDBLoadingResultsByNickname) GetJobAndAllResults(nickname string) (*datahistoryjob.DataHistoryJob, error) {
	if !strings.EqualFold(nickname, j.nickname) {
		return nil, sql.ErrNoRows
	}
	return j.IDBService.GetJobAndAllResults(nickname)
}

func withoutJobResults(job *datahistoryjob.DataHistoryJob, err error) (*datahistoryjob.DataHistoryJob, error) {
	if err != nil {
		return nil, err
	}
	without := *job
	without.Results = nil
	return &without, nil
}

func TestJobDBLoadingResultsByNickname(t *testing.T) {
	t.Parallel()
	newStore := func() (jobDBLoadingResultsByNickname, *datahistoryjob.DataHistoryJob) {
		job := &datahistoryjob.DataHistoryJob{
			Nickname: "TestJobDBLoadingResultsByNickname",
			Results:  []*datahistoryjobresult.DataHistoryJobResult{{ID: jobID, JobID: jobID}},
		}
		return jobDBLoadingResultsByNickname{IDBService: dataHistoryJobService{job: job}, nickname: job.Nickname}, job
	}

	t.Run("lookups by ID and nickname leave the results out", func(t *testing.T) {
		t.Parallel()
		store, job := newStore()
		byID, err := store.GetByID(jobID)
		require.NoError(t, err, "GetByID must not error")
		assert.Empty(t, byID.Results, "GetByID should leave the results out")
		byNickname, err := store.GetByNickName(job.Nickname)
		require.NoError(t, err, "GetByNickName must not error")
		assert.Empty(t, byNickname.Results, "GetByNickName should leave the results out")
		assert.Len(t, job.Results, 1, "the stored job should keep its results")
	})

	t.Run("full lookup by the job's nickname loads the results", func(t *testing.T) {
		t.Parallel()
		store, job := newStore()
		full, err := store.GetJobAndAllResults(strings.ToUpper(job.Nickname))
		require.NoError(t, err, "GetJobAndAllResults must not error")
		assert.Len(t, full.Results, 1, "GetJobAndAllResults should load the job's results")
	})

	t.Run("full lookup by another nickname finds nothing", func(t *testing.T) {
		t.Parallel()
		store, _ := newStore()
		_, err := store.GetJobAndAllResults("")
		assert.ErrorIs(t, err, sql.ErrNoRows, "GetJobAndAllResults should find nothing for another nickname")
	})

	t.Run("lookup errors are returned", func(t *testing.T) {
		t.Parallel()
		store := jobDBLoadingResultsByNickname{IDBService: failingJobDB{}, nickname: "failing"}
		for _, tc := range []struct {
			name   string
			lookup func() (*datahistoryjob.DataHistoryJob, error)
		}{
			{name: "GetByID", lookup: func() (*datahistoryjob.DataHistoryJob, error) { return store.GetByID(jobID) }},
			{name: "GetByNickName", lookup: func() (*datahistoryjob.DataHistoryJob, error) { return store.GetByNickName("failing") }},
			{name: "GetJobAndAllResults", lookup: func() (*datahistoryjob.DataHistoryJob, error) { return store.GetJobAndAllResults("failing") }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				job, err := tc.lookup()
				assert.ErrorIs(t, err, errExpectedTestError, "the lookup should return the store's error")
				assert.Nil(t, job, "the lookup should return no job with the error")
			})
		}
	})
}

// jobDBNicknameTakenOver answers a lookup by nickname with another job, as if it had replaced the job by that nickname
type jobDBNicknameTakenOver struct {
	datahistoryjob.IDBService
	id string
}

func (j jobDBNicknameTakenOver) GetJobAndAllResults(nickname string) (*datahistoryjob.DataHistoryJob, error) {
	job, err := j.IDBService.GetJobAndAllResults(nickname)
	if err != nil {
		return nil, err
	}
	other := *job
	other.ID = j.id
	return &other, nil
}

// failingJobDB fails every lookup
type failingJobDB struct {
	datahistoryjob.IDBService
}

func (failingJobDB) GetByID(string) (*datahistoryjob.DataHistoryJob, error) {
	return nil, errExpectedTestError
}

func (failingJobDB) GetByNickName(string) (*datahistoryjob.DataHistoryJob, error) {
	return nil, errExpectedTestError
}

func (failingJobDB) GetJobAndAllResults(string) (*datahistoryjob.DataHistoryJob, error) {
	return nil, errExpectedTestError
}

// missingJobDB finds no job, as the database answers a lookup for a nickname or ID it does not hold
type missingJobDB struct {
	datahistoryjob.IDBService
}

func (missingJobDB) GetByID(string) (*datahistoryjob.DataHistoryJob, error) {
	return nil, sql.ErrNoRows
}

func (missingJobDB) GetByNickName(string) (*datahistoryjob.DataHistoryJob, error) {
	return nil, sql.ErrNoRows
}

func (missingJobDB) GetJobAndAllResults(string) (*datahistoryjob.DataHistoryJob, error) {
	return nil, sql.ErrNoRows
}

func (d dataHistoryJobResultService) Upsert(_ ...*datahistoryjobresult.DataHistoryJobResult) error {
	return nil
}

func (d *countingDataHistoryJobResultService) Upsert(_ ...*datahistoryjobresult.DataHistoryJobResult) error {
	d.upsertCalls++
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

// rangeRequestExchange answers each candle request with a candle for every interval requested, or the error given, and
// records the ranges requested
type rangeRequestExchange struct {
	exchange.IBotExchange
	err      error
	requests []string
}

func (e *rangeRequestExchange) GetHistoricCandlesExtended(_ context.Context, p currency.Pair, a asset.Item, interval kline.Interval, timeStart, timeEnd time.Time) (*kline.Item, error) {
	e.requests = append(e.requests, timeStart.UTC().Format("15:04")+"-"+timeEnd.UTC().Format("15:04"))
	if e.err != nil {
		return nil, e.err
	}
	return &kline.Item{Exchange: testExchange, Pair: p, Asset: a, Interval: interval, Candles: generateCandles(int(timeEnd.Sub(timeStart)/interval.Duration()), timeStart, interval)}, nil
}

// candleExchange answers each candle request with the candles given for the request's start
type candleExchange struct {
	exchange.IBotExchange
	candles func(time.Time) []kline.Candle
}

func (e candleExchange) GetHistoricCandlesExtended(_ context.Context, p currency.Pair, a asset.Item, interval kline.Interval, timeStart, _ time.Time) (*kline.Item, error) {
	return &kline.Item{Exchange: testExchange, Pair: p, Asset: a, Interval: interval, Candles: e.candles(timeStart)}, nil
}

// tradeExchange answers each trade request with a trade at each of the times given for the request's dates
type tradeExchange struct {
	exchange.IBotExchange
	trades func(requestStart, requestEnd time.Time) []time.Time
}

func (e tradeExchange) GetHistoricTrades(_ context.Context, p currency.Pair, a asset.Item, startTime, endTime time.Time) ([]trade.Data, error) {
	times := e.trades(startTime, endTime)
	trades := make([]trade.Data, len(times))
	for i := range times {
		trades[i] = trade.Data{Exchange: testExchange, CurrencyPair: p, AssetType: a, Side: order.Buy, Price: 1337, Amount: 1, Timestamp: times[i]}
	}
	return trades, nil
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
