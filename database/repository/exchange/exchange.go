package exchange

import (
	"context"
	"database/sql"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/cache"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
)

// One returns one exchange by Name
func One(in string) (Details, error) {
	return one(in, "name")
}

// OneByUUID returns one exchange by UUID
func OneByUUID(in uuid.UUID) (Details, error) {
	return one(in.String(), "id")
}

// one returns one exchange by clause
func one(in, clause string) (out Details, err error) {
	if database.DB.SQL == nil {
		return out, database.ErrDatabaseSupportDisabled
	}

	whereQM := qm.Where(clause+"= ?", in)
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
		out.UUID, errS = uuid.FromString(ret.ID)
		if errS != nil {
			return out, errS
		}
	}

	return out, err
}

// Insert writes a single entry into database
func Insert(in Details) error {
	if database.DB.SQL == nil {
		return database.ErrDatabaseSupportDisabled
	}

	ctx := context.Background()
	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		err = insertSQLite(ctx, tx, []Details{in})
	} else {
		err = insertPostgresql(ctx, tx, []Details{in})
	}

	if err != nil {
		errRB := tx.Rollback()
		if errRB != nil {
			return errRB
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		errRB := tx.Rollback()
		if errRB != nil {
			log.Errorln(log.DatabaseMgr, errRB)
		}
		return err
	}

	return nil
}

// InsertMany writes multiple entries into database
func InsertMany(in []Details) error {
	if database.DB.SQL == nil {
		return database.ErrDatabaseSupportDisabled
	}

	ctx := context.Background()
	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		err = insertSQLite(ctx, tx, in)
	} else {
		err = insertPostgresql(ctx, tx, in)
	}

	if err != nil {
		errRB := tx.Rollback()
		if errRB != nil {
			return errRB
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		errRB := tx.Rollback()
		if errRB != nil {
			log.Errorln(log.DatabaseMgr, errRB)
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
			errRB := tx.Rollback()
			if errRB != nil {
				return errRB
			}
			return err
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
			errRB := tx.Rollback()
			if errRB != nil {
				return errRB
			}
			return
		}
	}
	return nil
}

// UUIDByName returns UUID of exchange
func UUIDByName(in string) (uuid.UUID, error) {
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

// ResetExchangeCache
func ResetExchangeCache() {
	exchangeCache = cache.New(5)
}
