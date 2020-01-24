package withdraw

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
			Exchange: 	  req.Exchange.Name,
			ExchangeID:   req.Exchange.ID,
			Status:       req.Exchange.Status,
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
			Exchange:     req.Exchange.Name,
			ExchangeID:   req.Exchange.ID,
			Status:       req.Exchange.Status,
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
	resp := &withdraw.Response{}
	if repository.GetSQLDialect() == database.DBSQLite3 {
	} else {
		query := qm.Where("id = ?", id)
		v, err := modelPSQL.WithdrawalHistories(query).One(ctx, database.DB.SQL)
		if err != nil {
			return nil, err
		}
		newUUID,_ := uuid.FromString(v.ID)
		resp.ID = newUUID
		resp.Exchange = new(withdraw.ExchangeResponse)
		resp.Exchange.ID = v.ExchangeID
		resp.Exchange.Name = v.Exchange
		resp.Exchange.Status = v.Status
		resp.RequestDetails = new(withdraw.Request)
		resp.RequestDetails = &withdraw.Request{
			Currency:        currency.Code{},
			Description:     v.Description.String,
			Amount:          v.Amount,
			Type:            withdraw.RequestType(v.WithdrawType),
		}
		resp.CreatedAt = v.CreatedAt
		resp.UpdatedAt = v.UpdatedAt
	}
	return resp, nil
}
