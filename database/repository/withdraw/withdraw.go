package withdraw

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/gocryptotrader/withdraw"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
)

func Event(req *withdraw.Response) {
	if database.DB.SQL == nil {
		return
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event transaction being failed: %v", err)
		return
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		newUUID, errUUID := uuid.NewV4()
		if errUUID != nil {
			log.Errorf(log.DatabaseMgr, "Failed to generate UUID: %v", err)
			_ = tx.Rollback()
			return
		}

		var tempEvent = modelSQLite.WithdrawalHistory{
			ID:           newUUID.String(),
			ExchangeID:   req.ExchangeID,
			Status:       req.Status,
			Currency:     req.RequestDetails.Currency.String(),
			Amount:       req.RequestDetails.Amount,
			WithdrawType: int64(req.RequestDetails.Type),
			CreatedAt:    time.Now().String(),
		}
		if req.RequestDetails.Description != "" {
			tempEvent.Description.SetValid(req.RequestDetails.Description)
		}
		err = tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
			_ = tx.Rollback()
			return
		}

		if req.RequestDetails.Type == withdraw.Fiat {
			fiatEvent := &modelSQLite.WithdrawalFiat{
				BankName:          req.RequestDetails.Fiat.BankName,
				BankAddress:       req.RequestDetails.Fiat.BankAddress,
				BankAccountName:   req.RequestDetails.Fiat.BankName,
				BankAccountNumber: req.RequestDetails.Fiat.BankAccountNumber,
				BSB:               req.RequestDetails.Fiat.BSB,
				SwiftCode:         req.RequestDetails.Fiat.SwiftCode,
				Iban:              req.RequestDetails.Fiat.IBAN,
				BankCode:          req.RequestDetails.Fiat.BankCode,
			}
			err = tempEvent.AddWithdrawalFiats(ctx, tx, true, fiatEvent)
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
				_ = tx.Rollback()
				return
			}
		}
		if req.RequestDetails.Type == withdraw.Crypto {
			cryptoEvent := &modelSQLite.WithdrawalCrypto{
				Address: req.RequestDetails.Crypto.Address,
				Fee:     req.RequestDetails.Crypto.FeeAmount,
			}
			if req.RequestDetails.Crypto.AddressTag != "" {
				cryptoEvent.AddressTag.SetValid(req.RequestDetails.Crypto.AddressTag)
			}
			err = tempEvent.AddWithdrawalCryptos(ctx, tx, true, cryptoEvent)
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
				_ = tx.Rollback()
				return
			}
		}
	} else {
		var tempEvent = modelPSQL.WithdrawalHistory{
			ExchangeID:   req.ExchangeID,
			Status:       req.Status,
			Currency:     req.RequestDetails.Currency.String(),
			Amount:       req.RequestDetails.Amount,
			WithdrawType: int(req.RequestDetails.Type),
			CreatedAt:    time.Now(),
		}
		if req.RequestDetails.Description != "" {
			tempEvent.Description.SetValid(req.RequestDetails.Description)
		}
		err = tempEvent.Insert(ctx, tx, boil.Infer())

		if req.RequestDetails.Type == withdraw.Fiat {
			fiatEvent := &modelPSQL.WithdrawalFiat{
				BankName:          req.RequestDetails.Fiat.BankName,
				BankAddress:       req.RequestDetails.Fiat.BankAddress,
				BankAccountName:   req.RequestDetails.Fiat.BankName,
				BankAccountNumber: req.RequestDetails.Fiat.BankAccountNumber,
				BSB:               req.RequestDetails.Fiat.BSB,
				SwiftCode:         req.RequestDetails.Fiat.SwiftCode,
				Iban:              req.RequestDetails.Fiat.IBAN,
				BankCode:          req.RequestDetails.Fiat.BankCode,
			}
			err = tempEvent.AddWithdrawalFiatWithdrawalFiats(ctx, tx, true, fiatEvent)
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
				_ = tx.Rollback()
				return
			}
		}
		if req.RequestDetails.Type == withdraw.Crypto {
			cryptoEvent := &modelPSQL.WithdrawalCrypto{
				Address: req.RequestDetails.Crypto.Address,
				Fee:     req.RequestDetails.Crypto.FeeAmount,
			}
			if req.RequestDetails.Crypto.AddressTag != "" {
				cryptoEvent.AddressTag.SetValid(req.RequestDetails.Crypto.AddressTag)
			}
			err = tempEvent.AddWithdrawalCryptoWithdrawalCryptos(ctx, tx, true, cryptoEvent)
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
				_ = tx.Rollback()
				return
			}
		}
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

func EventByUUID(id string) (*withdraw.Response, error) {
	if database.DB.SQL == nil {
		return nil, errors.New("database is nil")
	}

	ctx := context.Background()
	if repository.GetSQLDialect() == database.DBSQLite3 {
	} else {
		x, err := modelPSQL.WithdrawalHistories(qm.InnerJoin("withdrawal_fiat wf on withdrawal_history.id = wf.withdrawal_fiat_id"), qm.Where("id = ?", id)).All(ctx, database.DB.SQL)
		if err != nil {
			return nil, err
		}
		fmt.Println(x)
	}
	return nil, nil
}
