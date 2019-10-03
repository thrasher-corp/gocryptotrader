package client

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/volatiletech/null"
)

// Insert inserts a new client to the database
func Insert(userName, password, email, otp string, enabled bool) {
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
		var client = &modelSQLite.Client{
			UserName:        userName,
			Password:        password,
			Email:           email,
			OneTimePassword: null.StringFrom(otp),
			Enabled:         enabled,
		}
		err = client.Insert(ctx, tx, boil.Blacklist("updated_at", "created_at"))
	} else {
		var client = &modelPSQL.Client{
			UserName:        userName,
			Password:        password,
			Email:           email,
			OneTimePassword: null.StringFrom(otp),
			Enabled:         enabled,
		}
		err = client.Insert(ctx, tx, boil.Blacklist("updated_at", "created_at"))
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

func Update() {}

func Disable() {}

func Enable() {}

func Validate() {}

func OTPValidate() {}
