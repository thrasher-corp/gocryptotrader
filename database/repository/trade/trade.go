package trade

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/sqlboiler/boil"

	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func Insert(trades ...trade.Data) error {
	if database.DB.SQL == nil {
		return errors.New("trade.Insert nil db")
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("trade.Insert BeginTx %w", err)
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

func Get() (trade.Data, error) {
	return trade.Data{}, nil
}

func getSQLite(ctx context.Context, tx *sql.Tx, trades ...trade.Data) error {
	return nil
}

func getPostgres(ctx context.Context, tx *sql.Tx, trades ...trade.Data) error {
	return nil
}


func Select() ([]trade.Data, error) {
	return nil, nil
}

func selectSQLite(ctx context.Context, tx *sql.Tx, trades ...trade.Data) error {
	return nil
}

func selectPostgres(ctx context.Context, tx *sql.Tx, trades ...trade.Data) error {
	return nil
}

func Delete() error {
	return nil
}

func deleteSQLite(ctx context.Context, tx *sql.Tx, trades ...trade.Data) error {
	return nil
}

func deletePostgres(ctx context.Context, tx *sql.Tx, trades ...trade.Data) error {
	return nil
}

