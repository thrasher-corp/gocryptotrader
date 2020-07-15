package exchange

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
)

// One returns one exchange by Name
func One(in string) (out Details, err error) {
	if database.DB.SQL == nil {
		return out, database.ErrDatabaseSupportDisabled
	}

	whereQM := qm.Where("name = ?", in)
	if repository.GetSQLDialect() == database.DBSQLite3 {
		ret, errS := modelSQLite.Exchanges(whereQM).One(context.Background(), database.DB.SQL)
		if errS != nil {
			return out, errS
		}
		out.Name = ret.Name
		out.UUID, errS = uuid.FromString(ret.ID)
		if errS != nil {
			return out, errS
		}
	} else {
		ret, errS := modelPSQL.Exchanges(whereQM).One(context.Background(), database.DB.SQL)
		if errS != nil {
			return out, errS
		}
		out.Name = ret.Name
		out.UUID, _ = uuid.FromString(ret.ID)
	}

	return out, err
}

// OneByUUID returns one exchange by UUID
func OneByUUID(in uuid.UUID) (*modelPSQL.Exchange, error) {
	if database.DB.SQL == nil {
		return nil, database.ErrDatabaseSupportDisabled
	}

	return modelPSQL.FindExchange(context.Background(), database.DB.SQL,
		in.String())
}

// Insert writes a single entry into database
func Insert(in Details) {
	if database.DB.SQL == nil {
		return
	}

	ctx := boil.SkipTimestamps(context.Background())
	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Insert transaction failed: %v", err)
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

// InsertMany writes multiple entries into database
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
	for x := range in {
		tempUUID, errUUID := uuid.NewV4()
		if errUUID != nil {
			return errUUID
		}
		var tempInsert = modelSQLite.Exchange{
			Name: in[x].Name,
			ID:   tempUUID.String(),
		}

		err = tempInsert.Insert(ctx, tx, boil.Infer())
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Exchange Insert failed: %v", err)
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", errRB)
			}
			return
		}
	}

	return nil
}

func insertPostgresql(ctx context.Context, tx *sql.Tx, in []Details) (err error) {
	for x := range in {
		var tempInsert = modelPSQL.Exchange{
			Name: in[x].Name,
		}

		err = tempInsert.Upsert(ctx, tx, true, []string{"name"}, boil.Infer(), boil.Infer())
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Exchange Insert failed: %v", err)
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", errRB)
			}
			return
		}
	}
	return nil
}

// UUIDByName returns UUID of exchange
func UUIDByName(in string) (uuid.UUID, error) {
	fmt.Println("UUIDByName() Entered")
	defer func() {
		fmt.Println("UUIDByName() Exit")
	}()
	v := exchangeCache.Get(in)
	if v != nil {
		return v.(uuid.UUID), nil
	}

	ret, err := One(in)
	if err != nil {
		return uuid.UUID{}, err
	}

	exchangeCache.Add(in, ret.UUID)
	return ret.UUID, nil
}
