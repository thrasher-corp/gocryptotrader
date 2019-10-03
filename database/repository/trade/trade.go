package trade

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/sqlboiler/boil"
)

// Insert inserts a new order to the database
func Insert(tradeID, exchangeName, base, quote, side string, orderID int64, volume, price, fee, tax float64, executedAt time.Time) {
	if database.DB.SQL == nil {
		return
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.Global, "transaction begin failed: %v", err)
		return
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		var trade = &modelSQLite.TradeEvent{
			TradeID:       tradeID,
			OrderID:       orderID,
			Exchange:      exchangeName,
			BaseCurrency:  base,
			QuoteCurrency: quote,
			Side:          side,
			Volume:        volume,
			Price:         price,
			Fee:           fee,
			Tax:           tax,
			ExecutedAt:    executedAt.UTC().String(),
		}
		err = trade.Insert(ctx, tx, boil.Blacklist("updated_at", "created_at"))
	} else {
		var trade = &modelPSQL.TradeEvent{
			TradeID:       tradeID,
			OrderID:       orderID,
			Exchange:      exchangeName,
			BaseCurrency:  base,
			QuoteCurrency: quote,
			Side:          side,
			Volume:        volume,
			Price:         price,
			Fee:           fee,
			Tax:           tax,
			ExecutedAt:    executedAt,
		}
		err = trade.Insert(ctx, tx, boil.Blacklist("updated_at", "created_at"))
	}

	if err != nil {
		log.Errorf(log.Global, "insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.Global, "transaction rollback failed: %v", err)
		}
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf(log.Global, "transaction commit failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.Global, "transaction rollback failed: %v", err)
		}
		return
	}
}
