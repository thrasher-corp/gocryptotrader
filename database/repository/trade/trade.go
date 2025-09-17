package trade

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
)

// Insert saves trade data to the database
func Insert(trades ...Data) error {
	for i := range trades {
		if trades[i].ExchangeNameID == "" && trades[i].Exchange != "" {
			exchangeUUID, err := exchange.UUIDByName(trades[i].Exchange)
			if err != nil {
				return err
			}
			trades[i].ExchangeNameID = exchangeUUID.String()
		} else if trades[i].ExchangeNameID == "" && trades[i].Exchange == "" {
			return errors.New("exchange name/uuid not set, cannot insert")
		}
	}

	ctx := context.TODO()
	ctx = boil.SkipTimestamps(ctx)

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginTx %w", err)
	}
	defer func() {
		if err != nil {
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorf(log.DatabaseMgr, "Insert tx.Rollback %v", errRB)
			}
		}
	}()

	if repository.GetSQLDialect() == database.DBSQLite3 || repository.GetSQLDialect() == database.DBSQLite {
		err = insertSQLite(ctx, tx, trades...)
	} else {
		err = insertPostgres(ctx, tx, trades...)
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

// VerifyTradeInIntervals will query for ONE trade within each kline interval and verify if data exists
// if it does, it will set the range holder property "HasData" to true
func VerifyTradeInIntervals(exchangeName, assetType, base, quote string, irh *kline.IntervalRangeHolder) error {
	ctx := context.TODO()
	ctx = boil.SkipTimestamps(ctx)

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginTx %w", err)
	}
	defer func() {
		if err != nil {
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorf(log.DatabaseMgr, "Insert tx.Rollback %v", errRB)
			}
		}
	}()

	if repository.GetSQLDialect() == database.DBSQLite3 || repository.GetSQLDialect() == database.DBSQLite {
		err = verifyTradeInIntervalsSqlite(ctx, tx, exchangeName, assetType, base, quote, irh)
	} else {
		err = verifyTradeInIntervalsPostgres(ctx, tx, exchangeName, assetType, base, quote, irh)
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func verifyTradeInIntervalsSqlite(ctx context.Context, tx *sql.Tx, exchangeName, assetType, base, quote string, irh *kline.IntervalRangeHolder) error {
	exch, err := sqlite3.Exchanges(qm.Where("name = ?", exchangeName)).One(ctx, tx)
	if err != nil {
		return err
	}
	for i := range irh.Ranges {
		for j := range irh.Ranges[i].Intervals {
			result, err := sqlite3.Trades(qm.Where("exchange_name_id = ? AND asset = ? AND base = ? AND quote = ? AND timestamp between ? AND ?",
				exch.ID,
				assetType,
				base,
				quote,
				irh.Ranges[i].Intervals[j].Start.Time.UTC().Format(time.RFC3339),
				irh.Ranges[i].Intervals[j].End.Time.UTC().Format(time.RFC3339))).One(ctx, tx)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if result != nil {
				irh.Ranges[i].Intervals[j].HasData = true
			}
		}
	}

	return nil
}

func verifyTradeInIntervalsPostgres(ctx context.Context, tx *sql.Tx, exchangeName, assetType, base, quote string, irh *kline.IntervalRangeHolder) error {
	exch, err := postgres.Exchanges(qm.Where("name = ?", exchangeName)).One(ctx, tx)
	if err != nil {
		return err
	}
	for i := range irh.Ranges {
		for j := range irh.Ranges[i].Intervals {
			result, err := postgres.Trades(qm.Where("exchange_name_id = ? AND asset = ? AND base = ? AND quote = ? AND timestamp between ? AND ?",
				exch.ID,
				assetType,
				base,
				quote,
				irh.Ranges[i].Intervals[j].Start.Time.UTC(),
				irh.Ranges[i].Intervals[j].End.Time.UTC())).One(ctx, tx)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if result != nil {
				irh.Ranges[i].Intervals[j].HasData = true
			}
		}
	}

	return nil
}

func insertSQLite(ctx context.Context, tx *sql.Tx, trades ...Data) error {
	for i := range trades {
		if trades[i].ID == "" {
			freshUUID, err := uuid.NewV4()
			if err != nil {
				return err
			}
			trades[i].ID = freshUUID.String()
		}
		tempEvent := sqlite3.Trade{
			ID:             trades[i].ID,
			ExchangeNameID: trades[i].ExchangeNameID,
			Base:           strings.ToUpper(trades[i].Base),
			Quote:          strings.ToUpper(trades[i].Quote),
			Asset:          strings.ToLower(trades[i].AssetType),
			Price:          trades[i].Price,
			Amount:         trades[i].Amount,
			Timestamp:      trades[i].Timestamp.UTC().Format(time.RFC3339),
		}
		if trades[i].Side != "" {
			tempEvent.Side.SetValid(strings.ToUpper(trades[i].Side))
		}
		if trades[i].TID != "" {
			tempEvent.Tid.SetValid(trades[i].TID)
		}
		err := tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func insertPostgres(ctx context.Context, tx *sql.Tx, trades ...Data) error {
	var err error
	for i := range trades {
		if trades[i].ID == "" {
			var freshUUID uuid.UUID
			freshUUID, err = uuid.NewV4()
			if err != nil {
				return err
			}
			trades[i].ID = freshUUID.String()
		}
		tempEvent := postgres.Trade{
			ExchangeNameID: trades[i].ExchangeNameID,
			Base:           strings.ToUpper(trades[i].Base),
			Quote:          strings.ToUpper(trades[i].Quote),
			Asset:          strings.ToLower(trades[i].AssetType),
			Price:          trades[i].Price,
			Amount:         trades[i].Amount,
			Timestamp:      trades[i].Timestamp.UTC(),
			ID:             trades[i].ID,
		}
		if trades[i].Side != "" {
			tempEvent.Side.SetValid(strings.ToUpper(trades[i].Side))
		}
		if trades[i].TID != "" {
			tempEvent.Tid.SetValid(trades[i].TID)
		}

		err = tempEvent.Upsert(ctx, tx, false, nil, boil.Infer(), boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

// GetByUUID returns a trade by its unique ID
func GetByUUID(u string) (td Data, err error) {
	if repository.GetSQLDialect() == database.DBSQLite3 || repository.GetSQLDialect() == database.DBSQLite {
		td, err = getByUUIDSQLite(u)
		if err != nil {
			return td, fmt.Errorf("trade.Get getByUUIDSQLite %w", err)
		}
	} else {
		td, err = getByUUIDPostgres(u)
		if err != nil {
			return td, fmt.Errorf("trade.Get getByUUIDPostgres %w", err)
		}
	}

	return td, nil
}

func getByUUIDSQLite(u string) (Data, error) {
	var td Data
	var ts time.Time
	query := sqlite3.Trades(qm.Where("id = ?", u))
	result, err := query.One(context.TODO(), database.DB.SQL)
	if err != nil {
		return td, err
	}
	ts, err = time.Parse(time.RFC3339, result.Timestamp)
	if err != nil {
		return td, err
	}

	td = Data{
		ID:        result.ID,
		Exchange:  result.ExchangeNameID,
		Base:      strings.ToUpper(result.Base),
		Quote:     strings.ToUpper(result.Quote),
		AssetType: strings.ToLower(result.Asset),
		Price:     result.Price,
		Amount:    result.Amount,
		Timestamp: ts,
	}
	if result.Side.Valid {
		td.Side = result.Side.String
	}
	return td, nil
}

func getByUUIDPostgres(u string) (td Data, err error) {
	query := postgres.Trades(qm.Where("id = ?", u))
	var result *postgres.Trade
	result, err = query.One(context.TODO(), database.DB.SQL)
	if err != nil {
		return td, err
	}

	td = Data{
		ID:        result.ID,
		Timestamp: result.Timestamp.UTC(),
		Exchange:  result.ExchangeNameID,
		Base:      strings.ToUpper(result.Base),
		Quote:     strings.ToUpper(result.Quote),
		AssetType: strings.ToLower(result.Asset),
		Price:     result.Price,
		Amount:    result.Amount,
	}
	if result.Side.Valid {
		td.Side = result.Side.String
	}
	return td, nil
}

// GetInRange returns all trades by an exchange in a date range
func GetInRange(exchangeName, assetType, base, quote string, startDate, endDate time.Time) (td []Data, err error) {
	if repository.GetSQLDialect() == database.DBSQLite3 || repository.GetSQLDialect() == database.DBSQLite {
		td, err = getInRangeSQLite(exchangeName, assetType, base, quote, startDate, endDate)
		if err != nil {
			return td, fmt.Errorf("trade.GetByExchangeInRange getInRangeSQLite %w", err)
		}
	} else {
		td, err = getInRangePostgres(exchangeName, assetType, base, quote, startDate, endDate)
		if err != nil {
			return td, fmt.Errorf("trade.GetByExchangeInRange getInRangePostgres %w", err)
		}
	}

	return td, nil
}

func getInRangeSQLite(exchangeName, assetType, base, quote string, startDate, endDate time.Time) (td []Data, err error) {
	var exchangeUUID uuid.UUID
	exchangeUUID, err = exchange.UUIDByName(exchangeName)
	if err != nil {
		return nil, err
	}
	wheres := map[string]any{
		"exchange_name_id": exchangeUUID,
		"asset":            strings.ToLower(assetType),
		"base":             strings.ToUpper(base),
		"quote":            strings.ToUpper(quote),
	}
	q := generateQuery(wheres, startDate, endDate, true)
	query := sqlite3.Trades(q...)
	var result []*sqlite3.Trade
	result, err = query.All(context.TODO(), database.DB.SQL)
	if err != nil {
		return td, err
	}
	for i := range result {
		ts, err := time.Parse(time.RFC3339, result[i].Timestamp)
		if err != nil {
			return td, err
		}
		t := Data{
			ID:        result[i].ID,
			Timestamp: ts,
			Exchange:  strings.ToLower(exchangeName),
			Base:      strings.ToUpper(result[i].Base),
			Quote:     strings.ToUpper(result[i].Quote),
			AssetType: strings.ToLower(result[i].Asset),
			Price:     result[i].Price,
			Amount:    result[i].Amount,
		}
		if result[i].Side.Valid {
			t.Side = result[i].Side.String
		}
		td = append(td, t)
	}
	return td, nil
}

func getInRangePostgres(exchangeName, assetType, base, quote string, startDate, endDate time.Time) (td []Data, err error) {
	var exchangeUUID uuid.UUID
	exchangeUUID, err = exchange.UUIDByName(exchangeName)
	if err != nil {
		return nil, err
	}
	wheres := map[string]any{
		"exchange_name_id": exchangeUUID,
		"asset":            strings.ToLower(assetType),
		"base":             strings.ToUpper(base),
		"quote":            strings.ToUpper(quote),
	}

	q := generateQuery(wheres, startDate, endDate, false)
	query := postgres.Trades(q...)
	var result []*postgres.Trade
	result, err = query.All(context.TODO(), database.DB.SQL)
	if err != nil {
		return td, err
	}
	for i := range result {
		t := Data{
			ID:        result[i].ID,
			Timestamp: result[i].Timestamp,
			Exchange:  strings.ToLower(exchangeName),
			Base:      strings.ToUpper(result[i].Base),
			Quote:     strings.ToUpper(result[i].Quote),
			AssetType: strings.ToLower(result[i].Asset),
			Price:     result[i].Price,
			Amount:    result[i].Amount,
		}
		if result[i].Side.Valid {
			t.Side = result[i].Side.String
		}
		td = append(td, t)
	}
	return td, nil
}

// DeleteTrades will remove trades from the database using trade.Data
func DeleteTrades(trades ...Data) error {
	ctx := context.TODO()
	ctx = boil.SkipTimestamps(ctx)

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginTx %w", err)
	}
	defer func() {
		if err != nil {
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorf(log.DatabaseMgr, "DeleteTrades tx.Rollback %v", errRB)
			}
		}
	}()
	if repository.GetSQLDialect() == database.DBSQLite3 || repository.GetSQLDialect() == database.DBSQLite {
		err = deleteTradesSQLite(context.TODO(), tx, trades...)
	} else {
		err = deleteTradesPostgres(context.TODO(), tx, trades...)
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func deleteTradesSQLite(ctx context.Context, tx *sql.Tx, trades ...Data) error {
	tradeIDs := make([]any, len(trades))
	for i := range trades {
		tradeIDs[i] = trades[i].ID
	}
	query := sqlite3.Trades(qm.WhereIn(`id in ?`, tradeIDs...))
	_, err := query.DeleteAll(ctx, tx)
	return err
}

func deleteTradesPostgres(ctx context.Context, tx *sql.Tx, trades ...Data) error {
	tradeIDs := make([]any, len(trades))
	for i := range trades {
		tradeIDs[i] = trades[i].ID
	}
	query := postgres.Trades(qm.WhereIn(`id in ?`, tradeIDs...))
	_, err := query.DeleteAll(ctx, tx)
	return err
}

func generateQuery(clauses map[string]any, start, end time.Time, isSQLite bool) []qm.QueryMod {
	query := []qm.QueryMod{
		qm.OrderBy("timestamp"),
	}
	if isSQLite {
		query = append(query, qm.Where("timestamp BETWEEN ? AND ?", start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339)))
	} else {
		query = append(query, qm.Where("timestamp BETWEEN ? AND ?", start.UTC(), end.UTC()))
	}
	for k, v := range clauses {
		query = append(query, qm.Where(k+` = ?`, v))
	}

	return query
}
