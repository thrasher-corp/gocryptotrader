package audit

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
)

// Event inserts a new audit event to database
func Event(id, msgtype, message string) {
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
		var tempEvent = modelSQLite.AuditEvent{
			Type:       msgtype,
			Identifier: id,
			Message:    message,
		}
		err = tempEvent.Insert(ctx, tx, boil.Blacklist("created_at"))
	} else {
		var tempEvent = modelPSQL.AuditEvent{
			Type:       msgtype,
			Identifier: id,
			Message:    message,
		}
		err = tempEvent.Insert(ctx, tx, boil.Blacklist("created_at"))
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

// GetEvent () returns list of order events matching query
func GetEvent(starTime, endTime, order string, limit int64) (interface{}, error) {
	if database.DB.SQL == nil {
		return nil, errors.New("database is nil")
	}
	boil.DebugMode = true

	query := qm.Where("created_at BETWEEN ? and ?", starTime, endTime)
	orderby := qm.OrderBy("id desc")

	fmt.Println(query)

	ctx := context.Background()
	if repository.GetSQLDialect() == database.DBSQLite3 {
		events, err := modelSQLite.AuditEvents().All(ctx, database.DB.SQL)
		if err != nil {
			return nil, err
		}
		return events, nil
	}
	events, err := modelPSQL.AuditEvents(query, orderby).All(ctx, database.DB.SQL)
	if err != nil {
		return nil, err
	}
	return events, nil
}
