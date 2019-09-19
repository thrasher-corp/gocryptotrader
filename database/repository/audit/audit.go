package audit

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Event inserts a new audit event to database
func Event(id, msgtype, message string) {
	if database.DB.SQL == nil {
		return
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	var tempEvent interface{}

	if repository.GetSQLDialect() == "sqlite" {
		tempEvent = modelSQLite.AuditEvent{
			Type:       msgtype,
			Identifier: id,
			Message:    message,
		}
	} else {
		tempEvent = modelPSQL.AuditEvent{
			Type:       msgtype,
			Identifier: id,
			Message:    message,
		}
	}

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.Global, "transaction begin failed: %v", err)
		return
	}

	if repository.GetSQLDialect() == "sqlite" {
		err = tempEvent.Insert(ctx, tx, boil.Blacklist("created_at"))
	} else {

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
