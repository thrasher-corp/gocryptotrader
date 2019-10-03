package orderdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
)

// Insert inserts a new order to the database
func Insert(exchOrderID, clientID, exchName, pair, asset, orderType, side, status string, amount, price float64) {
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
		var order = &modelSQLite.OrderEvent{
			ExchangeOrderID: exchOrderID,
			ClientID:        clientID,
			Exchange:        exchName,
			CurrencyPair:    pair,
			AssetType:       asset,
			OrderType:       orderType,
			OrderSide:       side,
			OrderStatus:     status,
			Amount:          amount,
			Price:           price,
		}
		err = order.Insert(ctx, tx, boil.Blacklist("updated_at", "created_at"))
	} else {
		var order = &modelPSQL.OrderEvent{
			ExchangeOrderID: exchOrderID,
			ClientID:        clientID,
			Exchange:        exchName,
			CurrencyPair:    pair,
			AssetType:       asset,
			OrderType:       orderType,
			OrderSide:       side,
			OrderStatus:     status,
			Amount:          amount,
			Price:           price,
		}
		err = order.Insert(ctx, tx, boil.Blacklist("updated_at", "created_at"))
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

// GetAllOpenOrders by client ID
func GetAllOpenOrders(clientID uuid.UUID) ([]order.Detail, error) {
	var details []order.Detail

	if database.DB.SQL == nil {
		return nil, errors.New("no connection to database")
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	if repository.GetSQLDialect() == database.DBSQLite3 {
		orders, err := modelSQLite.OrderEvents(
			qm.Select("client_id = ?", clientID.String()),
			qm.Where("order_status = ?", order.Active),
			qm.Or("order_status = ?", order.PartiallyFilled)).All(ctx, database.DB.SQL)
		if err != nil {
			return nil, err
		}

		fmt.Println("SQLITE3 ORDERS:", orders)

		for i := range orders {
			details = append(details, order.Detail{
				Exchange: orders[i].Exchange,
			})
		}

		return nil, nil
	}

	orders, err := modelPSQL.OrderEvents(
		qm.Select("client_id = ?", clientID.String()),
		qm.Where("order_status = ?", order.Active),
		qm.Or("order_status = ?", order.PartiallyFilled)).All(ctx, database.DB.SQL)
	if err != nil {
		return nil, err
	}

	fmt.Println("POSTGRES ORDERS:", orders)

	for i := range orders {
		details = append(details, order.Detail{
			Exchange: orders[i].Exchange,
		})
	}

	return nil, nil
}
