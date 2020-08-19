package candle

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
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

// Series returns candle data
func Series(exchangeName, base, quote string, interval int64, asset string, start, end time.Time) (out Item, err error) {
	if exchangeName == "" || base == "" || quote == "" || asset == "" || interval <= 0 {
		return out, errInvalidInput
	}

	queries := []qm.QueryMod{
		qm.Where("base = ?", base),
		qm.Where("quote = ?", quote),
		qm.Where("interval = ?", interval),
		qm.Where("asset = ?", asset),
		qm.Where("timestamp between ? and ?", start, end),
	}

	exchangeUUID, errS := exchange.UUIDByName(exchangeName)
	if errS != nil {
		return out, errS
	}
	queries = append(queries, qm.Where("exchange_id = ?", exchangeUUID.String()))

	if repository.GetSQLDialect() == database.DBSQLite3 {
		retCandle, errC := modelSQLite.Candles(queries...).All(context.Background(), database.DB.SQL)
		if errC != nil {
			return out, errC
		}
		for x := range retCandle {
			t, errT := time.Parse(time.RFC3339, retCandle[x].Timestamp)
			if errT != nil {
				return out, errT
			}
			out.Candles = append(out.Candles, Candle{
				Timestamp: t,
				Open:      retCandle[x].Open,
				High:      retCandle[x].High,
				Low:       retCandle[x].Low,
				Close:     retCandle[x].Close,
				Volume:    retCandle[x].Volume,
			})
		}
	} else {
		retCandle, errC := modelPSQL.Candles(queries...).All(context.Background(), database.DB.SQL)
		if errC != nil {
			return out, errC
		}

		for x := range retCandle {
			out.Candles = append(out.Candles, Candle{
				Timestamp: retCandle[x].Timestamp,
				Open:      retCandle[x].Open,
				High:      retCandle[x].High,
				Low:       retCandle[x].Low,
				Close:     retCandle[x].Close,
				Volume:    retCandle[x].Volume,
			})
		}
	}
	if len(out.Candles) < 1 {
		return out, fmt.Errorf(errNoCandleDataFound, exchangeName, base, quote, interval, asset)
	}

	out.ExchangeID = exchangeName
	out.Interval = interval
	out.Base = base
	out.Quote = quote
	out.Asset = asset
	return out, err
}

// Insert series of candles
func Insert(in *Item) (uint64, error) {
	if database.DB.SQL == nil {
		return 0, database.ErrDatabaseSupportDisabled
	}

	ctx := context.Background()
	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	var totalInserted uint64
	if repository.GetSQLDialect() == database.DBSQLite3 {
		totalInserted, err = insertSQLite(ctx, tx, in)
	} else {
		totalInserted, err = insertPostgresSQL(ctx, tx, in)
	}
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		errRB := tx.Rollback()
		if errRB != nil {
			log.Errorln(log.DatabaseMgr, errRB)
		}
		return 0, err
	}
	return totalInserted, nil
}

func insertSQLite(ctx context.Context, tx *sql.Tx, in *Item) (uint64, error) {
	var totalInserted uint64
	for x := range in.Candles {
		var tempCandle = modelSQLite.Candle{
			ExchangeID: null.NewString(in.ExchangeID, true),
			Base:       in.Base,
			Quote:      in.Quote,
			Interval:   strconv.FormatInt(in.Interval, 10),
			Asset:      in.Asset,
			Timestamp:  in.Candles[x].Timestamp.Format(time.RFC3339),
			Open:       in.Candles[x].Open,
			High:       in.Candles[x].High,
			Low:        in.Candles[x].Low,
			Close:      in.Candles[x].Close,
			Volume:     in.Candles[x].Volume,
		}
		tempUUID, err := uuid.NewV4()
		if err != nil {
			return 0, err
		}
		tempCandle.ID = tempUUID.String()
		err = tempCandle.Insert(ctx, tx, boil.Infer())
		if err != nil {
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorln(log.DatabaseMgr, errRB)
			}
			return 0, err
		}
		if totalInserted < math.MaxUint64 {
			totalInserted++
		}
	}
	return totalInserted, nil
}

func insertPostgresSQL(ctx context.Context, tx *sql.Tx, in *Item) (uint64, error) {
	var totalInserted uint64
	for x := range in.Candles {
		var tempCandle = modelPSQL.Candle{
			ExchangeNameID: in.ExchangeID,
			Base:           in.Base,
			Quote:          in.Quote,
			Interval:       in.Interval,
			Asset:          in.Asset,
			Timestamp:      in.Candles[x].Timestamp,
			Open:           in.Candles[x].Open,
			High:           in.Candles[x].High,
			Low:            in.Candles[x].Low,
			Close:          in.Candles[x].Close,
			Volume:         in.Candles[x].Volume,
		}
		err := tempCandle.Upsert(ctx, tx, true, []string{"timestamp", "exchange_id", "base", "quote", "interval", "asset"}, boil.Infer(), boil.Infer())
		if err != nil {
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorln(log.DatabaseMgr, errRB)
			}
			return 0, err
		}
		if totalInserted < math.MaxUint64 {
			totalInserted++
		}
	}
	return totalInserted, nil
}

// InsertFromCSV load a CSV list of candle data and insert into database
func InsertFromCSV(exchangeName, base, quote string, interval int64, asset, file string) (uint64, error) {
	csvFile, err := os.Open(file)
	if err != nil {
		return 0, err
	}

	defer func() {
		err = csvFile.Close()
		if err != nil {
			log.Errorln(log.Global, err)
		}
	}()

	csvData := csv.NewReader(csvFile)

	exchangeUUID, err := exchange.UUIDByName(exchangeName)
	if err != nil {
		return 0, err
	}

	tempCandle := &Item{
		ExchangeID: exchangeUUID.String(),
		Base:       base,
		Quote:      quote,
		Interval:   interval,
		Asset:      asset,
	}

	for {
		row, errCSV := csvData.Read()
		if errCSV != nil {
			if errCSV == io.EOF {
				break
			}
			return 0, errCSV
		}

		tempTick := Candle{}
		v, errParse := strconv.ParseInt(row[0], 10, 32)
		if errParse != nil {
			return 0, errParse
		}
		tempTick.Timestamp = time.Unix(v, 0).UTC()
		if tempTick.Timestamp.IsZero() {
			err = fmt.Errorf("invalid timestamp received on row %v", row)
			break
		}

		tempTick.Volume, err = strconv.ParseFloat(row[1], 64)
		if err != nil {
			break
		}

		tempTick.Open, err = strconv.ParseFloat(row[2], 64)
		if err != nil {
			break
		}

		tempTick.High, err = strconv.ParseFloat(row[3], 64)
		if err != nil {
			break
		}

		tempTick.Low, err = strconv.ParseFloat(row[4], 64)
		if err != nil {
			break
		}

		tempTick.Close, err = strconv.ParseFloat(row[5], 64)
		if err != nil {
			break
		}
		tempCandle.Candles = append(tempCandle.Candles, tempTick)
	}
	if err != nil {
		return 0, err
	}

	return Insert(tempCandle)
}
