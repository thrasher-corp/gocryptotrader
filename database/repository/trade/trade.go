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
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
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

	ctx := context.Background()
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

func insertSQLite(ctx context.Context, tx *sql.Tx, trades ...Data) error {
	for i := range trades {
		if trades[i].ID == "" {
			freshUUID, err := uuid.NewV4()
			if err != nil {
				return err
			}
			trades[i].ID = freshUUID.String()
		}
		var tempEvent = modelSQLite.Trade{
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
		var tempEvent = modelPSQL.Trade{
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
func GetByUUID(uuid string) (td Data, err error) {
	if repository.GetSQLDialect() == database.DBSQLite3 || repository.GetSQLDialect() == database.DBSQLite {
		td, err = getByUUIDSQLite(uuid)
		if err != nil {
			return td, fmt.Errorf("trade.Get getByUUIDSQLite %w", err)
		}
	} else {
		td, err = getByUUIDPostgres(uuid)
		if err != nil {
			return td, fmt.Errorf("trade.Get getByUUIDPostgres %w", err)
		}
	}

	return td, nil
}

func getByUUIDSQLite(uuid string) (Data, error) {
	var td Data
	var ts time.Time
	query := modelSQLite.Trades(qm.Where("id = ?", uuid))
	result, err := query.One(context.Background(), database.DB.SQL)
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

func getByUUIDPostgres(uuid string) (td Data, err error) {
	query := modelPSQL.Trades(qm.Where("id = ?", uuid))
	var result *modelPSQL.Trade
	result, err = query.One(context.Background(), database.DB.SQL)
	if err != nil {
		return td, err
	}

	td = Data{
		ID:        result.ID,
		Timestamp: result.Timestamp,
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
	wheres := map[string]interface{}{
		"exchange_name_id": exchangeUUID,
		"asset":            strings.ToLower(assetType),
		"base":             strings.ToUpper(base),
		"quote":            strings.ToUpper(quote),
	}
	q := generateQuery(wheres, startDate, endDate)
	query := modelSQLite.Trades(q...)
	var result []*modelSQLite.Trade
	result, err = query.All(context.Background(), database.DB.SQL)
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
	wheres := map[string]interface{}{
		"exchange_name_id": exchangeUUID,
		"asset":            strings.ToLower(assetType),
		"base":             strings.ToUpper(base),
		"quote":            strings.ToUpper(quote),
	}
	q := generateQuery(wheres, startDate, endDate)
	query := modelPSQL.Trades(q...)
	var result []*modelPSQL.Trade
	result, err = query.All(context.Background(), database.DB.SQL)
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
	ctx := context.Background()
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
		err = deleteTradesSQLite(context.Background(), tx, trades...)
	} else {
		err = deleteTradesPostgres(context.Background(), tx, trades...)
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func deleteTradesSQLite(ctx context.Context, tx *sql.Tx, trades ...Data) error {
	var tradeIDs []interface{}
	for i := range trades {
		tradeIDs = append(tradeIDs, trades[i].ID)
	}
	query := modelSQLite.Trades(qm.WhereIn(`id in ?`, tradeIDs...))
	_, err := query.DeleteAll(ctx, tx)
	return err
}

func deleteTradesPostgres(ctx context.Context, tx *sql.Tx, trades ...Data) error {
	var tradeIDs []interface{}
	for i := range trades {
		tradeIDs = append(tradeIDs, trades[i].ID)
	}
	query := modelPSQL.Trades(qm.WhereIn(`id in ?`, tradeIDs...))
	_, err := query.DeleteAll(ctx, tx)
	return err
}

func generateQuery(clauses map[string]interface{}, start, end time.Time) []qm.QueryMod {
	query := []qm.QueryMod{
		qm.Where("timestamp BETWEEN ? AND ?", start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339)),
	}
	for k, v := range clauses {
		query = append(query, qm.Where(k+` = ?`, v))
	}
	return query
}
