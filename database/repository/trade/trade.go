package trade

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Insert saves trade data to the database
func Insert(trades ...trade.Data) error {
	ctx, tx, err := initialDBSetup()
	if err != nil {
		return fmt.Errorf("trade.Insert initialDBSetup %w", err)
	}
	defer func() {
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "trade.Insert tx.Rollback %w", err)
			}
		}
	}()

	if repository.GetSQLDialect() == database.DBSQLite3 {
		err = insertSQLite(ctx, tx, trades...)
		if err != nil {
			return fmt.Errorf("trade.Insert insertSQLite %w", err)
		}
	} else {
		err = insertPostgres(ctx, tx, trades...)
		if err != nil {
			return fmt.Errorf("trade.Insert insertPostgres %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("trade.Insert Commit %w", err)
	}
	return nil
}


func insertSQLite(ctx context.Context, tx *sql.Tx, trades ...trade.Data) error {
	// get all exchanges
	var err error
	exchangeID, _ := uuid.NewV4()
	for i := range trades {
		var tempEvent = modelSQLite.Trade{
			ExchangeID: exchangeID.String(),
			Currency:   trades[i].CurrencyPair.String(),
			Asset:      trades[i].AssetType.String(),
			Event:      trades[i].EventType.String(),
			Price:      trades[i].Price,
			Amount:     trades[i].Amount,
			Side:       trades[i].Side.String(),
		}

		err = tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func insertPostgres(ctx context.Context, tx *sql.Tx, trades ...trade.Data) error {
	// get all exchanges
	var err error
	exchangeID, _ := uuid.NewV4()
	for i := range trades {
		var tempEvent = modelPSQL.Trade{
			Currency:   trades[i].CurrencyPair.String(),
			Asset:      trades[i].AssetType.String(),
			Event:      trades[i].EventType.String(),
			Price:      trades[i].Price,
			Amount:     trades[i].Amount,
			Side:       trades[i].Side.String(),
		}
		if exchangeID.String() != "" {
			tempEvent.ExchangeID.SetValid(exchangeID.String())
		}

		err = tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

// GetByUUID returns a trade by its unique ID
func GetByUUID(uuid string) (td trade.Data, err error) {
	var ctx context.Context
	ctx, _, err = initialDBSetup()
	if err != nil {
		return td, fmt.Errorf("trade.Insert initialDBSetup %w", err)
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		td, err = getByUUIDSQLite(ctx, uuid)
		if err != nil {
			return td, fmt.Errorf("trade.Get getByUUIDSQLite %w", err)
		}
	} else {
		td, err = getByUUIDPostgres(ctx, uuid)
		if err != nil {
			return td, fmt.Errorf("trade.Get getByUUIDPostgres %w", err)
		}
	}

	return td, nil
}

func getByUUIDSQLite(ctx context.Context, uuid string) (td trade.Data, err error) {
	query := modelSQLite.Trades(qm.Where("id = ?", uuid))
	var result *modelSQLite.Trade
	result, err = query.One(ctx, database.DB.SQL)
	if err != nil {
		return td, err
	}

	td = resultToTradeData(
		result.Side,
		result.Event,
		result.Asset,
		result.Currency,
		result.ExchangeID,
		result.Amount,
		result.Price,
		result.Timestamp,
	)
	return td, nil
}

func getByUUIDPostgres(ctx context.Context, uuid string) (td trade.Data, err error) {
	query := modelPSQL.Trades(qm.Where("id = ?", uuid))
	var result *modelPSQL.Trade
	result, err = query.One(ctx, database.DB.SQL)
	if err != nil {
		return td, err
	}

	td = resultToTradeData(
		result.Side,
		result.Event,
		result.Asset,
		result.Currency,
		"",
		result.Amount,
		result.Price,
		float64(result.Timestamp),
	)
	return td, nil
}

// SelectByExchangeBetweenRange returns all trades by an exchange in a date range
func SelectByExchangeBetweenRange(exchangeName string, startDate, endDate int64) (td []trade.Data, err error) {
	var ctx context.Context
	ctx, _, err = initialDBSetup()
	if err != nil {
		return td, fmt.Errorf("trade.Insert initialDBSetup %w", err)
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		td, err = selectSQLite(ctx, exchangeName ,startDate, endDate)
		if err != nil {
			return td, fmt.Errorf("trade.SelectByExchangeBetweenRange selectSQLite %w", err)
		}
	} else {
		td, err = selectPostgres(ctx, exchangeName ,startDate, endDate)
		if err != nil {
			return td, fmt.Errorf("trade.SelectByExchangeBetweenRange selectPostgres %w", err)
		}
	}

	return td, nil
}

func selectSQLite(ctx context.Context, exchangeName string, startDate, endDate int64) (td []trade.Data, err error) {
	wheres := map[string]interface{}{
		"exchange": exchangeName,
	}
	q := generateQuery(wheres, startDate, endDate, -1)
	query := modelSQLite.Trades(q...)
	var result []*modelSQLite.Trade
	result, err = query.All(ctx, database.DB.SQL)
	if err != nil {
		return td, err
	}
	for i := range result {
		td = append(td, resultToTradeData(
			result[i].Side,
			result[i].Event,
			result[i].Asset,
			result[i].Currency,
			result[i].ExchangeID,
			result[i].Amount,
			result[i].Price,
			result[i].Timestamp),
		)
	}
	return td, nil
}

func selectPostgres(ctx context.Context, exchangeName string, startDate, endDate int64) (td []trade.Data, err error) {
	wheres := map[string]interface{}{
		"exchange": exchangeName,
	}
	q := generateQuery(wheres, startDate, endDate, -1)
	query := modelPSQL.Trades(q...)
	var result []*modelPSQL.Trade
	result, err = query.All(ctx, database.DB.SQL)
	if err != nil {
		return td, err
	}
	for i := range result {
		td = append(td, resultToTradeData(
			result[i].Side,
			result[i].Event,
			result[i].Asset,
			result[i].Currency,
			"",
			result[i].Amount,
			result[i].Price,
			float64(result[i].Timestamp)),
		)
	}
	return td, nil
}

// DeleteTradeData will remove trades from the database using trade.Data
func DeleteTradeData(trades ...trade.Data) error {
	ctx, tx, err := initialDBSetup()
	if err != nil {
		return fmt.Errorf("trade.DeleteTradeData initialDBSetup %w", err)
	}
	defer func() {
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "trade.DeleteTradeData tx.Rollback %w", err)
			}
		}
	}()

	if repository.GetSQLDialect() == database.DBSQLite3 {
		err = deleteTradeDataSQLite(ctx, database.DB.SQL, trades...)
		if err != nil {
			return fmt.Errorf("trade.DeleteTradeData deleteTradeDataSQLite %w", err)
		}
	} else {
		err = deleteTradeDataPostgres(ctx, database.DB.SQL, trades...)
		if err != nil {
			return fmt.Errorf("trade.DeleteTradeData deleteTradeDataPostgres %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("trade.DeleteTradeData Commit %w", err)
	}
	return nil
}

func deleteTradeDataSQLite(ctx context.Context, bbb *sql.DB, trades ...trade.Data) error {
	var tradeids []interface{}
	for i := range trades {
		tradeids = append(tradeids, trades[i].ID.String())
	}
	query := modelPSQL.Trades(qm.WhereIn(`id in ?`, tradeids...))
	_, err := query.DeleteAll(ctx, bbb)
	return err
}

func deleteTradeDataPostgres(ctx context.Context, bbb *sql.DB, trades ...trade.Data) error {
	var tradeids []interface{}
	for i := range trades {
		tradeids = append(tradeids, trades[i].ID.String())
	}
	query := modelSQLite.Trades(qm.WhereIn(`id in ?`, tradeids...))
	_, err := query.DeleteAll(ctx, bbb)
	return err
}


func initialDBSetup() (context.Context, *sql.Tx, error) {
	if database.DB.SQL == nil {
		return nil, nil, errors.New("trade.Insert nil db")
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("BeginTx %w", err)
	}
	return ctx, tx, nil
}

func resultToTradeData(side, event, assetType, curr, exchangeName string, amount, price,  timestamp float64) (td trade.Data) {
	s, err := order.StringToOrderSide(side)
	if err != nil {
		// ?
	}
	e, err := order.StringToOrderType(event)
	if err != nil {
		// ?
	}
	a := asset.Spot
	if asset.IsValid(asset.Item(assetType)) {
		a = asset.Item(assetType)
	}
	cp, err := currency.NewPairFromString(curr)
	if err != nil {
		// ?
	}
	td.Side = s
	td.Amount = amount
	td.Price = price
	td.EventType = e
	td.AssetType = a
	td.CurrencyPair = cp
	td.Exchange = exchangeName
	td.Timestamp = time.Unix(int64(timestamp), 0)

	return td
}

func generateQuery(clauses map[string]interface{}, start, end int64, limit int) []qm.QueryMod {
	query := []qm.QueryMod{
		qm.Limit(limit),
		qm.Where("timestamp BETWEEN ? AND ?", start, end),
	}
	for k, v := range clauses {
		query = append(query, qm.Where(k + ` = ?`, v))
	}
	return query
}
