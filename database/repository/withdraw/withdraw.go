package withdraw

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/gocryptotrader/withdraw"
	"github.com/thrasher-corp/sqlboiler/boil"
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
