package script

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/volatiletech/null"
)

func Event(id uuid.UUID, name, path, hash null.String, executionType, status string, time time.Time) {
	if database.DB.SQL == nil {
		return
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)
	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.Global, "Event transaction begin failed: %v", err)
		return
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		var tempEvent = modelSQLite.ScriptEvent{
			ScriptID:        id.String(),
			ScriptName:      name,
			ScriptPath:      path,
			ScriptHash:      hash,
			ExecutionType:   executionType,
			ExecutionTime:   time.UTC().String(),
			ExecutionStatus: status,
		}
		err = tempEvent.Insert(ctx, tx, boil.Infer())
	} else {
		var tempEvent = modelPSQL.ScriptEvent{
			ScriptID:        id.String(),
			ScriptName:      name,
			ScriptPath:      path,
			ScriptHash:      hash,
			ExecutionType:   executionType,
			ExecutionTime:   time.UTC(),
			ExecutionStatus: status,
		}
		err = tempEvent.Insert(ctx, tx, boil.Infer())
	}

	if err != nil {
		log.Errorf(log.Global, "Event insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.Global, "Event Transaction rollback failed: %v", err)
		}
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf(log.Global, "Event Transaction commit failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.Global, "Event Transaction rollback failed: %v", err)
		}
		return
	}
}
