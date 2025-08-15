package script

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/volatiletech/null"
)

// Event inserts a new script event into database with execution details (script name time status hash of script)
func Event(id, name, path string, data null.Bytes, executionType, status string, tm time.Time) {
	if database.DB.SQL == nil {
		return
	}

	ctx := context.TODO()
	ctx = boil.SkipTimestamps(ctx)
	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event transaction begin failed: %v", err)
		return
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		query := modelSQLite.ScriptWhere.ScriptID.EQ(id)
		f, errQry := modelSQLite.Scripts(query).Exists(ctx, tx)
		if errQry != nil {
			log.Errorf(log.DatabaseMgr, "Query failed: %v", errQry)
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Event Transaction rollback failed: %v", err)
			}
			return
		}
		tempEvent := modelSQLite.Script{}
		if !f {
			newUUID, errUUID := uuid.NewV4()
			if errUUID != nil {
				log.Errorf(log.DatabaseMgr, "Failed to generate UUID: %v", errUUID)
				_ = tx.Rollback()
				return
			}

			tempEvent.ID = newUUID.String()
			tempEvent.ScriptID = id
			tempEvent.ScriptName = name
			tempEvent.ScriptPath = path
			tempEvent.ScriptData = data
			err = tempEvent.Insert(ctx, tx, boil.Infer())
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Event insert failed: %v", err)
				err = tx.Rollback()
				if err != nil {
					log.Errorf(log.DatabaseMgr, "Event Transaction rollback failed: %v", err)
				}
				return
			}
		} else {
			tempEvent.ID = id
		}

		tempScriptExecution := &modelSQLite.ScriptExecution{
			ScriptID:        id,
			ExecutionTime:   tm.UTC().String(),
			ExecutionStatus: status,
			ExecutionType:   executionType,
		}
		err = tempEvent.AddScriptExecutions(ctx, tx, true, tempScriptExecution)
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event insert failed: %v", err)
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Event Transaction rollback failed: %v", err)
			}
			return
		}
	} else {
		tempEvent := modelPSQL.Script{
			ScriptID:   id,
			ScriptName: name,
			ScriptPath: path,
			ScriptData: data,
		}
		err = tempEvent.Upsert(ctx, tx, true, []string{"script_id"}, boil.Whitelist("last_executed_at"), boil.Infer())
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event insert failed: %v", err)
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Event Transaction rollback failed: %v", err)
			}
			return
		}

		tempScriptExecution := &modelPSQL.ScriptExecution{
			ExecutionTime:   tm.UTC(),
			ExecutionStatus: status,
			ExecutionType:   executionType,
		}

		err = tempEvent.AddScriptExecutions(ctx, tx, true, tempScriptExecution)
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event insert failed: %v", err)
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Event Transaction rollback failed: %v", err)
			}
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event Transaction commit failed: %v", err)
	}
}
