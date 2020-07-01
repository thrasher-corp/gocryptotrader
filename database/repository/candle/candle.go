package candle

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
	"github.com/volatiletech/null"
)

// Candle generic candle holder for modelPSQL & modelSQLite
type Candle struct {
	ID         string
	ExchangeID string
	Base       string
	Quote      string
	Interval   string
	Tick       []Tick
}

// Tick holds each interval
type Tick struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// Series returns candle data
func Series(exchangeName, base, quote, interval string, start, end time.Time) (out Candle, err error) {
	if exchangeName == "" || base == "" || quote == "" || interval == "" {
		err = errors.New("exchange, base , quote, interval, start & end cannot be empty")
		return
	}

	exchangeUUID, err := exchange.UUIDByName(exchangeName)
	if err != nil {
		return
	}

	uuidQM := qm.Where("exchange_id = ?", exchangeUUID.String())
	baseQM := qm.Where("base = ?", base)
	quoteQM := qm.Where("quote = ?", quote)
	intervalQM := qm.Where("interval = ?", interval)
	dateQM := qm.Where("timestamp between ? and ?", start, end)

	if repository.GetSQLDialect() == database.DBSQLite3 {
		retCandle, errC := modelSQLite.Candles(uuidQM, baseQM, quoteQM, intervalQM, dateQM).All(context.Background(), database.DB.SQL)
		if errC != nil {
			return out, errC
		}
		for x := range retCandle {
			t, errT := time.Parse(time.RFC3339, retCandle[x].Timestamp)
			if errT != nil {
				return out, errT
			}
			out.Tick = append(out.Tick, Tick{
				Timestamp: t,
				Open:      retCandle[x].Open,
				High:      retCandle[x].High,
				Low:       retCandle[x].Low,
				Close:     retCandle[x].Close,
				Volume:    retCandle[x].Volume,
			})
		}
	} else {
		retCandle, errC := modelPSQL.Candles(uuidQM, baseQM, quoteQM, intervalQM, dateQM).All(context.Background(), database.DB.SQL)
		if errC != nil {
			return out, errC
		}

		for x := range retCandle {
			out.Tick = append(out.Tick, Tick{
				Timestamp: retCandle[x].Timestamp,
				Open:      retCandle[x].Open,
				High:      retCandle[x].High,
				Low:       retCandle[x].Low,
				Close:     retCandle[x].Close,
				Volume:    retCandle[x].Volume,
			})
		}
	}

	out.ExchangeID = exchangeName
	out.Interval = interval
	out.Base = base
	out.Quote = quote

	return out, err
}

// Insert series of candles
func Insert(in *Candle) error {
	if database.DB.SQL == nil {
		return database.ErrDatabaseSupportDisabled
	}

	ctx := boil.SkipTimestamps(context.Background())
	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Insert transaction being failed: %v", err)
		return err
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		err = insertSQLite(ctx, tx, in)
	} else {
		err = insertPostgresSQL(ctx, tx, in)
	}
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Insert Transaction commit failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Insert Transaction rollback failed: %v", err)
		}
		return err
	}

	return nil
}

func insertSQLite(ctx context.Context, tx *sql.Tx, in *Candle) error {
	for x := range in.Tick {
		var tempCandle = modelSQLite.Candle{
			ExchangeID: null.NewString(in.ExchangeID, true),
			Base:       in.Base,
			Quote:      in.Quote,
			Interval:   in.Interval,
			Timestamp:  in.Tick[x].Timestamp.Format(time.RFC3339),
			Open:       in.Tick[x].Open,
			High:       in.Tick[x].High,
			Low:        in.Tick[x].Low,
			Close:      in.Tick[x].Close,
			Volume:     in.Tick[x].Volume,
		}
		tempUUID, err := uuid.NewV4()
		if err != nil {
			return err
		}
		tempCandle.ID = tempUUID.String()
		err = tempCandle.Insert(ctx, tx, boil.Infer())
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Candle Insert failed: %v", err)
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", errRB)
			}
			return err
		}
	}
	return nil
}

func insertPostgresSQL(ctx context.Context, tx *sql.Tx, in *Candle) error {
	for x := range in.Tick {
		var tempCandle = modelPSQL.Candle{
			ExchangeID: null.NewString(in.ExchangeID, true),
			Base:       in.Base,
			Quote:      in.Quote,
			Interval:   in.Interval,
			Timestamp:  in.Tick[x].Timestamp,
			Open:       in.Tick[x].Open,
			High:       in.Tick[x].High,
			Low:        in.Tick[x].Low,
			Close:      in.Tick[x].Close,
			Volume:     in.Tick[x].Volume,
		}
		err := tempCandle.Upsert(ctx, tx, true, []string{"timestamp", "exchange_id", "base", "quote", "interval"}, boil.Infer(), boil.Infer())
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Candle Insert failed: %v", err)
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", errRB)
			}
			return err
		}
	}
	return nil
}
