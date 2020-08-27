package trade

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"

	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Insert saves trade data to the database
func Insert(trades ...Data) error {
	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("BeginTx %w", err)
	}
	defer func() {
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "trade.Insert tx.Rollback %v", err)
			}
		}
	}()
	if repository.GetSQLDialect() == database.DBSQLite3 {
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
	// get all exchanges
	var err error
	for i := range trades {
		var tempEvent = modelSQLite.Trade{
			ID:         trades[i].ID,
			ExchangeID: trades[i].Exchange,
			Currency:   trades[i].CurrencyPair,
			Asset:      trades[i].AssetType,
			Price:      trades[i].Price,
			Amount:     trades[i].Amount,
			Side:       trades[i].Side,
			Timestamp:  float64(trades[i].Timestamp),
		}
		if trades[i].TID != "" {
			tempEvent.Tid.SetValid(trades[i].TID)
		}
		err = tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func insertPostgres(ctx context.Context, tx *sql.Tx, trades ...Data) error {
	// get all exchanges
	var err error
	exchangeID, _ := uuid.NewV4()
	for i := range trades {
		var tempEvent = modelPSQL.Trade{
			Currency:  trades[i].CurrencyPair,
			Asset:     trades[i].AssetType,
			Price:     trades[i].Price,
			Amount:    trades[i].Amount,
			Side:      trades[i].Side,
			Timestamp: trades[i].Timestamp,
			ID:        trades[i].ID,
		}
		if trades[i].TID != "" {
			tempEvent.Tid.SetValid(trades[i].TID)
		}
		if exchangeID.String() != "" {
			//tempEvent.ExchangeID.SetValid(trades[i].Exchange)
		}

		err = tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

// GetByUUID returns a trade by its unique ID
func GetByUUID(uuid string) (td Data, err error) {
	if repository.GetSQLDialect() == database.DBSQLite3 {
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

func getByUUIDSQLite(uuid string) (td Data, err error) {
	query := modelSQLite.Trades(qm.Where("id = ?", uuid))
	result, err := query.One(context.Background(), database.DB.SQL)
	if err != nil {
		return td, err
	}

	td = Data{
		ID:           result.ID,
		Timestamp:    int64(result.Timestamp),
		Exchange:     result.ExchangeID,
		CurrencyPair: result.Currency,
		AssetType:    result.Asset,
		Price:        result.Price,
		Amount:       result.Amount,
		Side:         result.Side,
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
		ID:           result.ID,
		Timestamp:    result.Timestamp,
		Exchange:     result.ExchangeID.String,
		CurrencyPair: result.Currency,
		AssetType:    result.Asset,
		Price:        result.Price,
		Amount:       result.Amount,
		Side:         result.Side,
	}
	return td, nil
}

// GetByExchangeInRange returns all trades by an exchange in a date range
func GetByExchangeInRange(exchangeName string, startDate, endDate int64) (td []Data, err error) {
	if repository.GetSQLDialect() == database.DBSQLite3 {
		td, err = getByExchangeInRangeSQLite(exchangeName, startDate, endDate)
		if err != nil {
			return td, fmt.Errorf("trade.GetByExchangeInRange getByExchangeInRangeSQLite %w", err)
		}
	} else {
		td, err = getByExchangeInRangePostgres(exchangeName, startDate, endDate)
		if err != nil {
			return td, fmt.Errorf("trade.GetByExchangeInRange getByExchangeInRangePostgres %w", err)
		}
	}

	return td, nil
}

func getByExchangeInRangeSQLite(exchangeName string, startDate, endDate int64) (td []Data, err error) {
	wheres := map[string]interface{}{
		"exchange_id": exchangeName,
	}
	q := generateQuery(wheres, startDate, endDate, 50000)
	query := modelSQLite.Trades(q...)
	var result []*modelSQLite.Trade
	result, err = query.All(context.Background(), database.DB.SQL)
	if err != nil {
		return td, err
	}
	for i := range result {
		td = append(td, Data{
			ID:           result[i].ID,
			Timestamp:    int64(result[i].Timestamp),
			Exchange:     result[i].ExchangeID,
			CurrencyPair: result[i].Currency,
			AssetType:    result[i].Asset,
			Price:        result[i].Price,
			Amount:       result[i].Amount,
			Side:         result[i].Side,
		})
	}
	return td, nil
}

func getByExchangeInRangePostgres(exchangeName string, startDate, endDate int64) (td []Data, err error) {
	wheres := map[string]interface{}{
		"exchange_id": exchangeName,
	}
	q := generateQuery(wheres, startDate, endDate, -1)
	query := modelPSQL.Trades(q...)
	var result []*modelPSQL.Trade
	result, err = query.All(context.Background(), database.DB.SQL)
	if err != nil {
		return td, err
	}
	for i := range result {
		td = append(td, Data{
			ID:           result[i].ID,
			Timestamp:    result[i].Timestamp,
			Exchange:     result[i].ExchangeID.String,
			CurrencyPair: result[i].Currency,
			AssetType:    result[i].Asset,
			Price:        result[i].Price,
			Amount:       result[i].Amount,
			Side:         result[i].Side,
		})
	}
	return td, nil
}

// DeleteTrades will remove trades from the database using trade.Data
func DeleteTrades(trades ...Data) error {
	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("BeginTx %w", err)
	}
	defer func() {
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "trade.Insert tx.Rollback %v", err)
			}
		}
	}()
	if repository.GetSQLDialect() == database.DBSQLite3 {
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
	var tradeids []interface{}
	for i := range trades {
		tradeids = append(tradeids, trades[i].ID)
	}
	query := modelSQLite.Trades(qm.WhereIn(`id in ?`, tradeids...))
	_, err := query.DeleteAll(ctx, tx)
	return err
}

func deleteTradesPostgres(ctx context.Context, tx *sql.Tx, trades ...Data) error {
	var tradeids []interface{}
	for i := range trades {
		tradeids = append(tradeids, trades[i].ID)
	}
	query := modelPSQL.Trades(qm.WhereIn(`id in ?`, tradeids...))
	_, err := query.DeleteAll(ctx, tx)
	return err
}

func generateQuery(clauses map[string]interface{}, start, end int64, limit int) []qm.QueryMod {
	query := []qm.QueryMod{
		qm.Limit(limit),
		qm.Where("timestamp BETWEEN ? AND ?", start, end),
	}
	for k, v := range clauses {
		query = append(query, qm.Where(k+` = ?`, v))
	}
	return query
}
