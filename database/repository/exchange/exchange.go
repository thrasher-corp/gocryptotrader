package exchange

import (
	"context"
	"database/sql"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
)

type Details struct {
	Name string
}

func One(in string) (*modelPSQL.Exchange,error) {
	if database.DB.SQL == nil {
		return nil, database.ErrDatabaseSupportDisabled
	}

	return modelPSQL.Exchanges(qm.Where("name = ?", in)).One(context.Background(), database.DB.SQL)
}

func OneByUUID(in uuid.UUID) (*modelPSQL.Exchange, error) {
	if database.DB.SQL == nil {
		return nil, database.ErrDatabaseSupportDisabled
	}

	return modelPSQL.FindExchange(context.Background(), database.DB.SQL,
		in.String())
}

func Insert(in Details) {
	if database.DB.SQL == nil {
		return
	}

	ctx := boil.SkipTimestamps(context.Background())
	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Insert transaction being failed: %v", err)
		return
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		err = insertSQLite(ctx, tx, []Details{in})
	} else {
		err = insertPostgresql(ctx, tx, []Details{in})
	}

	if err != nil {
		log.Errorf(log.DatabaseMgr, "Insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Insert Transaction rollback failed: %v", err)
		}
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Insert Transaction commit failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Insert Transaction rollback failed: %v", err)
		}
		return
	}
}

func InsertMany(in []Details) error {
	if database.DB.SQL == nil {
		return database.ErrDatabaseSupportDisabled
	}

	ctx := boil.SkipTimestamps(context.Background())
	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Insert transaction being failed: %v", err)
		return err
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		err = insertSQLite(ctx, tx, in)
	} else {
		err = insertPostgresql(ctx, tx, in)
	}

	if err != nil {
		log.Errorf(log.DatabaseMgr, "Insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Insert Transaction rollback failed: %v", err)
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Insert Transaction commit failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Insert Transaction rollback failed: %v", err)
		}
		return err
	}
	return nil
}

func insertSQLite(ctx context.Context, tx *sql.Tx, in []Details) (err error) {
	return nil
}

func insertPostgresql(ctx context.Context, tx *sql.Tx, in []Details) (err error) {
	for x := range in {
		var tempInsert = modelPSQL.Exchange{
			Name: in[x].Name,
		}

		err = tempInsert.Upsert(ctx, tx, true, []string{"name"}, boil.Infer(), boil.Infer())
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", errRB)
			}
			return
		}
	}
	return nil
}