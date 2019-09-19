package audit

import (
	"context"

	"github.com/volatiletech/sqlboiler/boil"

	"github.com/thrasher-corp/gocryptotrader/database"
	models "github.com/thrasher-corp/gocryptotrader/database/models/sqlite"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Event inserts a new audit event to database
func Event(id, msgtype, message string) {
	if database.DB.SQL == nil {
		return
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	var tempEvent = models.AuditEvent{
		Type:       msgtype,
		Identifier: id,
		Message:    message,
	}

	boil.DebugMode = true

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.Global, "transaction begin failed: %v", err)
		return
	}

	err = tempEvent.Insert(ctx, tx, boil.Blacklist("created_at"))
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
