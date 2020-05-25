package candle

// import (
// 	"context"
// 	"database/sql"
//
// 	"github.com/thrasher-corp/gocryptotrader/common"
// 	"github.com/thrasher-corp/gocryptotrader/database"
// 	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
// 	"github.com/thrasher-corp/gocryptotrader/database/repository"
// 	"github.com/thrasher-corp/gocryptotrader/log"
// 	"github.com/thrasher-corp/sqlboiler/boil"
// )
//
// func One() error {
// 	return nil
// }
//
// func Series() error {
// 	return nil
// }
//
// func Insert() error {
// 	return nil
// }
//
// func InsertMany(in []modelPSQL.Candle) error {
// 	if database.DB.SQL == nil {
// 		return database.ErrDatabaseSupportDisabled
// 	}
//
// 	ctx := boil.SkipTimestamps(context.Background())
// 	tx, err := database.DB.SQL.BeginTx(ctx, nil)
// 	if err != nil {
// 		log.Errorf(log.DatabaseMgr, "Insert transaction being failed: %v", err)
// 		return err
// 	}
//
// 	if repository.GetSQLDialect() == database.DBSQLite3 {
// 		err = insertSQLite(ctx, tx, in)
// 	} else {
// 		err = insertPostgresSQL(ctx, tx, in)
// 	}
//
// 	err = tx.Commit()
// 	if err != nil {
// 		log.Errorf(log.DatabaseMgr, "Insert Transaction commit failed: %v", err)
// 		err = tx.Rollback()
// 		if err != nil {
// 			log.Errorf(log.DatabaseMgr, "Insert Transaction rollback failed: %v", err)
// 		}
// 		return err
// 	}
//
// 	return nil
// }
//
// func insertSQLite(ctx context.Context, tx *sql.Tx, in []modelPSQL.Candle) (err error) {
// 	return common.ErrNotYetImplemented
// }
//
// func insertPostgresSQL(ctx context.Context, tx *sql.Tx, in []modelPSQL.Candle) error {
// 	for  x := range in {
// 		var  tempCandle = in[x]
//
// 		err := tempCandle.Upsert(ctx, tx, true, []string{"exchange"}, boil.Infer(), boil.Infer())
// 		if err != nil {
// 			log.Errorf(log.DatabaseMgr, "Candle Insert failed: %v", err)
// 			errRB := tx.Rollback()
// 			if errRB != nil {
// 				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", errRB)
// 			}
// 			return err
// 		}
// 	}
// 	return nil
// }