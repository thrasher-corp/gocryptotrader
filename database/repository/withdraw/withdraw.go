package withdraw

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/sqlboiler/boil"
)

// Event store new Withdrawal event
func Event(res *withdraw.Response) {
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
		err = addSQLiteEvent(ctx, tx, res)
	} else {
		err = addPSQLEvent(ctx, tx, res)
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

func addPSQLEvent(ctx context.Context, tx *sql.Tx, res *withdraw.Response) (err error) {
	var tempEvent = modelPSQL.WithdrawalHistory{
		Exchange:     res.Exchange.Name,
		ExchangeID:   res.Exchange.ID,
		Status:       res.Exchange.Status,
		Currency:     res.RequestDetails.Currency.String(),
		Amount:       res.RequestDetails.Amount,
		WithdrawType: int(res.RequestDetails.Type),
		CreatedAt:    time.Now(),
	}
	if res.RequestDetails.Description != "" {
		tempEvent.Description.SetValid(res.RequestDetails.Description)
	}
	err = tempEvent.Insert(ctx, tx, boil.Infer())
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
		_ = tx.Rollback()
		return
	}
	if res.RequestDetails.Type == withdraw.Fiat {
		fiatEvent := &modelPSQL.WithdrawalFiat{
			BankName:          res.RequestDetails.Fiat.Bank.BankName,
			BankAddress:       res.RequestDetails.Fiat.Bank.BankAddress,
			BankAccountName:   res.RequestDetails.Fiat.Bank.AccountName,
			BankAccountNumber: res.RequestDetails.Fiat.Bank.AccountNumber,
			BSB:               res.RequestDetails.Fiat.Bank.BSBNumber,
			SwiftCode:         res.RequestDetails.Fiat.Bank.SWIFTCode,
			Iban:              res.RequestDetails.Fiat.Bank.IBAN,
		}
		err = tempEvent.SetWithdrawalFiatWithdrawalFiats(ctx, tx, true, fiatEvent)
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
			_ = tx.Rollback()
			return
		}
	}
	if res.RequestDetails.Type == withdraw.Crypto {
		cryptoEvent := &modelPSQL.WithdrawalCrypto{
			Address: res.RequestDetails.Crypto.Address,
			Fee:     res.RequestDetails.Crypto.FeeAmount,
		}
		if res.RequestDetails.Crypto.AddressTag != "" {
			cryptoEvent.AddressTag.SetValid(res.RequestDetails.Crypto.AddressTag)
		}
		err = tempEvent.AddWithdrawalCryptoWithdrawalCryptos(ctx, tx, true, cryptoEvent)
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
			_ = tx.Rollback()
			return
		}
	}
	realID, _ := uuid.FromString(tempEvent.ID)
	res.ID = realID

	return nil
}

func addSQLiteEvent(ctx context.Context, tx *sql.Tx, res *withdraw.Response) (err error) {
	newUUID, errUUID := uuid.NewV4()
	if errUUID != nil {
		log.Errorf(log.DatabaseMgr, "Failed to generate UUID: %v", err)
		_ = tx.Rollback()
		return
	}

	var tempEvent = modelSQLite.WithdrawalHistory{
		ID:           newUUID.String(),
		Exchange:     res.Exchange.Name,
		ExchangeID:   res.Exchange.ID,
		Status:       res.Exchange.Status,
		Currency:     res.RequestDetails.Currency.String(),
		Amount:       res.RequestDetails.Amount,
		WithdrawType: int64(res.RequestDetails.Type),
		CreatedAt:    time.Now().String(),
	}
	if res.RequestDetails.Description != "" {
		tempEvent.Description.SetValid(res.RequestDetails.Description)
	}
	err = tempEvent.Insert(ctx, tx, boil.Infer())
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
		_ = tx.Rollback()
		return
	}

	if res.RequestDetails.Type == withdraw.Fiat {
		fiatEvent := &modelSQLite.WithdrawalFiat{
			BankName:          res.RequestDetails.Fiat.Bank.BankName,
			BankAddress:       res.RequestDetails.Fiat.Bank.BankAddress,
			BankAccountName:   res.RequestDetails.Fiat.Bank.AccountName,
			BankAccountNumber: res.RequestDetails.Fiat.Bank.AccountNumber,
			BSB:               res.RequestDetails.Fiat.Bank.BSBNumber,
			SwiftCode:         res.RequestDetails.Fiat.Bank.SWIFTCode,
			Iban:              res.RequestDetails.Fiat.Bank.IBAN,
		}
		err = tempEvent.AddWithdrawalFiats(ctx, tx, true, fiatEvent)
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
			_ = tx.Rollback()
			return
		}
	}
	if res.RequestDetails.Type == withdraw.Crypto {
		cryptoEvent := &modelSQLite.WithdrawalCrypto{
			Address: res.RequestDetails.Crypto.Address,
			Fee:     res.RequestDetails.Crypto.FeeAmount,
		}
		if res.RequestDetails.Crypto.AddressTag != "" {
			cryptoEvent.AddressTag.SetValid(res.RequestDetails.Crypto.AddressTag)
		}
		err = tempEvent.AddWithdrawalCryptos(ctx, tx, true, cryptoEvent)
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
			_ = tx.Rollback()
			return
		}
	}
	res.ID = newUUID
	return nil
}

// EventByUUID return requested withdraw information by ID
func EventByUUID(id string) (*withdraw.Response, error) {
	if database.DB.SQL == nil {
		return nil, errors.New("database is nil")
	}

	var resp = &withdraw.Response{}
	var ctx = context.Background()

	v, err := modelPSQL.FindWithdrawalHistory(ctx, database.DB.SQL, id, "*")
	if err != nil {
		return nil, err
	}

	newUUID, _ := uuid.FromString(v.ID)
	resp.ID = newUUID
	resp.Exchange = new(withdraw.ExchangeResponse)
	resp.Exchange.ID = v.ExchangeID
	resp.Exchange.Name = v.Exchange
	resp.Exchange.Status = v.Status
	resp.RequestDetails = new(withdraw.Request)
	resp.RequestDetails = &withdraw.Request{
		Currency:    currency.Code{},
		Description: v.Description.String,
		Amount:      v.Amount,
		Type:        withdraw.RequestType(v.WithdrawType),
	}
	resp.CreatedAt = v.CreatedAt
	resp.UpdatedAt = v.UpdatedAt

	if withdraw.RequestType(v.WithdrawType) == withdraw.Crypto {
		resp.RequestDetails.Crypto = new(withdraw.CryptoRequest)
		x, err := v.WithdrawalCryptoWithdrawalCryptos().One(ctx, database.DB.SQL)
		if err != nil {
			return nil, err
		}
		resp.RequestDetails.Crypto.Address = x.Address
		resp.RequestDetails.Crypto.AddressTag = x.AddressTag.String
		resp.RequestDetails.Crypto.FeeAmount = x.Fee
	} else {
		resp.RequestDetails.Fiat = new(withdraw.FiatRequest)
		x, err := v.WithdrawalFiatWithdrawalFiats().One(ctx, database.DB.SQL)
		if err != nil {
			return nil, err
		}
		resp.RequestDetails.Fiat.Bank = new(banking.Account)
		resp.RequestDetails.Fiat.Bank.BSBNumber = x.BSB
	}

	return resp, nil
}
