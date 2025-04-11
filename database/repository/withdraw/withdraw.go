package withdraw

import (
	"context"
	"database/sql"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	exchangeDB "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
)

// Event stores Withdrawal Response details in database
func Event(res *withdraw.Response) {
	if database.DB.SQL == nil {
		return
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	exchangeUUID, err := exchangeDB.UUIDByName(res.Exchange.Name)
	if err != nil {
		log.Errorln(log.DatabaseMgr, err)
		return
	}

	res.Exchange.Name = exchangeUUID.String()
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
		log.Errorf(log.DatabaseMgr, "Event insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Transaction rollback failed: %v", err)
		}
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event Transaction commit failed: %v", err)
		return
	}
}

func addPSQLEvent(ctx context.Context, tx *sql.Tx, res *withdraw.Response) (err error) {
	tempEvent := modelPSQL.WithdrawalHistory{
		ExchangeNameID: res.Exchange.Name,
		ExchangeID:     res.Exchange.ID,
		Status:         res.Exchange.Status,
		Currency:       res.RequestDetails.Currency.String(),
		Amount:         res.RequestDetails.Amount,
		WithdrawType:   int(res.RequestDetails.Type),
	}

	if res.RequestDetails.Description != "" {
		tempEvent.Description.SetValid(res.RequestDetails.Description)
	}

	err = tempEvent.Insert(ctx, tx, boil.Infer())
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
		}
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
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
			}
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
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
			}
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
		log.Errorf(log.DatabaseMgr, "Failed to generate UUID: %v", errUUID)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
		}
		return
	}

	tempEvent := modelSQLite.WithdrawalHistory{
		ID:             newUUID.String(),
		ExchangeNameID: res.Exchange.Name,
		ExchangeID:     res.Exchange.ID,
		Status:         res.Exchange.Status,
		Currency:       res.RequestDetails.Currency.String(),
		Amount:         res.RequestDetails.Amount,
		WithdrawType:   int64(res.RequestDetails.Type),
	}

	if res.RequestDetails.Description != "" {
		tempEvent.Description.SetValid(res.RequestDetails.Description)
	}

	err = tempEvent.Insert(ctx, tx, boil.Infer())
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
		}
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
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
			}
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
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
			}
			return
		}
	}

	res.ID = newUUID
	return nil
}

// GetEventByUUID return requested withdraw information by ID
func GetEventByUUID(id string) (*withdraw.Response, error) {
	resp, err := getByColumns(generateWhereQuery([]string{"id"}, []string{id}, 1))
	if err != nil {
		return nil, err
	}
	return resp[0], nil
}

// GetEventsByExchange returns all withdrawal requests by exchange
func GetEventsByExchange(exchange string, limit int) ([]*withdraw.Response, error) {
	exch, err := exchangeDB.UUIDByName(exchange)
	if err != nil {
		log.Errorln(log.DatabaseMgr, err)
		return nil, err
	}
	return getByColumns(generateWhereQuery([]string{"exchange_name_id"}, []string{exch.String()}, limit))
}

// GetEventByExchangeID return requested withdraw information by Exchange ID
func GetEventByExchangeID(exchange, id string) (*withdraw.Response, error) {
	exch, err := exchangeDB.UUIDByName(exchange)
	if err != nil {
		log.Errorln(log.DatabaseMgr, err)
		return nil, err
	}
	resp, err := getByColumns(generateWhereQuery([]string{"exchange_name_id", "exchange_id"}, []string{exch.String(), id}, 1))
	if err != nil {
		return nil, err
	}
	return resp[0], err
}

// GetEventsByDate returns requested withdraw information by date range
func GetEventsByDate(exchange string, start, end time.Time, limit int) ([]*withdraw.Response, error) {
	betweenQuery := generateWhereBetweenQuery("created_at", start, end, limit)
	if exchange == "" {
		return getByColumns(betweenQuery)
	}
	exch, err := exchangeDB.UUIDByName(exchange)
	if err != nil {
		log.Errorln(log.DatabaseMgr, err)
		return nil, err
	}
	return getByColumns(append(generateWhereQuery([]string{"exchange_name_id"}, []string{exch.String()}, 0), betweenQuery...))
}

func generateWhereQuery(columns, id []string, limit int) []qm.QueryMod {
	x := len(columns)
	if limit > 0 {
		x++
	}
	queries := make([]qm.QueryMod, 0, x)
	if limit > 0 {
		queries = append(queries, qm.Limit(limit))
	}
	for x := range columns {
		queries = append(queries, qm.Where(columns[x]+"= ?", id[x]))
	}
	return queries
}

func generateWhereBetweenQuery(column string, start, end any, limit int) []qm.QueryMod {
	return []qm.QueryMod{
		qm.Limit(limit),
		qm.Where(column+" BETWEEN ? AND ?", start, end),
	}
}

func getByColumns(q []qm.QueryMod) ([]*withdraw.Response, error) {
	if database.DB.SQL == nil {
		return nil, database.ErrDatabaseSupportDisabled
	}

	var resp []*withdraw.Response
	ctx := context.Background()
	if repository.GetSQLDialect() == database.DBSQLite3 {
		v, err := modelSQLite.WithdrawalHistories(q...).All(ctx, database.DB.SQL)
		if err != nil {
			return nil, err
		}
		for x := range v {
			tempResp := &withdraw.Response{}
			var newUUID uuid.UUID
			newUUID, err = uuid.FromString(v[x].ID)
			if err != nil {
				return nil, err
			}
			tempResp.ID = newUUID
			tempResp.Exchange.ID = v[x].ExchangeID
			tempResp.Exchange.Status = v[x].Status
			tempResp.RequestDetails = withdraw.Request{
				Currency:    currency.NewCode(v[x].Currency),
				Description: v[x].Description.String,
				Amount:      v[x].Amount,
				Type:        withdraw.RequestType(v[x].WithdrawType),
			}

			exchangeName, err := v[x].ExchangeName().One(ctx, database.DB.SQL)
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Unable to get exchange name")
				tempUUID, errUUID := uuid.FromString(v[x].ExchangeNameID)
				if errUUID != nil {
					log.Errorf(log.DatabaseMgr, "invalid exchange name UUID for record %v", v[x].ID)
				} else {
					tempResp.Exchange.UUID = tempUUID
				}
			} else {
				tempResp.Exchange.Name = exchangeName.Name
			}

			createdAtTime, err := time.Parse(time.RFC3339, v[x].CreatedAt)
			if err != nil {
				log.Errorf(log.DatabaseMgr, "record: %v has an incorrect time format ( %v ) - defaulting to empty time: %v", tempResp.ID, v[x].CreatedAt, err)
				tempResp.CreatedAt = time.Time{}
			} else {
				tempResp.CreatedAt = createdAtTime
			}

			updatedAtTime, err := time.Parse(time.RFC3339, v[x].UpdatedAt)
			if err != nil {
				log.Errorf(log.DatabaseMgr, "record: %v has an incorrect time format ( %v ) - defaulting to empty time: %v", tempResp.ID, v[x].UpdatedAt, err)
				tempResp.UpdatedAt = time.Time{}
			} else {
				tempResp.UpdatedAt = updatedAtTime
			}

			if withdraw.RequestType(v[x].WithdrawType) == withdraw.Crypto {
				x, err := v[x].WithdrawalCryptos().One(ctx, database.DB.SQL)
				if err != nil {
					return nil, err
				}
				tempResp.RequestDetails.Crypto.Address = x.Address
				tempResp.RequestDetails.Crypto.AddressTag = x.AddressTag.String
				tempResp.RequestDetails.Crypto.FeeAmount = x.Fee
			} else {
				x, err := v[x].WithdrawalFiats().One(ctx, database.DB.SQL)
				if err != nil {
					return nil, err
				}
				tempResp.RequestDetails.Fiat.Bank.AccountName = x.BankAccountName
				tempResp.RequestDetails.Fiat.Bank.AccountNumber = x.BankAccountNumber
				tempResp.RequestDetails.Fiat.Bank.IBAN = x.Iban
				tempResp.RequestDetails.Fiat.Bank.SWIFTCode = x.SwiftCode
				tempResp.RequestDetails.Fiat.Bank.BSBNumber = x.BSB
			}
			resp = append(resp, tempResp)
		}
	} else {
		v, err := modelPSQL.WithdrawalHistories(q...).All(ctx, database.DB.SQL)
		if err != nil {
			return nil, err
		}

		for x := range v {
			tempResp := &withdraw.Response{}
			newUUID, _ := uuid.FromString(v[x].ID)
			tempResp.ID = newUUID
			tempResp.Exchange.ID = v[x].ExchangeID
			tempResp.Exchange.Status = v[x].Status
			tempResp.RequestDetails = withdraw.Request{
				Currency:    currency.NewCode(v[x].Currency),
				Description: v[x].Description.String,
				Amount:      v[x].Amount,
				Type:        withdraw.RequestType(v[x].WithdrawType),
			}
			tempResp.CreatedAt = v[x].CreatedAt
			tempResp.UpdatedAt = v[x].UpdatedAt

			exchangeName, err := v[x].ExchangeName().One(ctx, database.DB.SQL)
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Unable to get exchange name")
				tempUUID, errUUID := uuid.FromString(v[x].ExchangeNameID)
				if errUUID != nil {
					log.Errorf(log.DatabaseMgr, "invalid exchange name UUID for record %v", v[x].ID)
				} else {
					tempResp.Exchange.UUID = tempUUID
				}
			} else {
				tempResp.Exchange.Name = exchangeName.Name
			}

			if withdraw.RequestType(v[x].WithdrawType) == withdraw.Crypto {
				x, err := v[x].WithdrawalCryptoWithdrawalCryptos().One(ctx, database.DB.SQL)
				if err != nil {
					return nil, err
				}
				tempResp.RequestDetails.Crypto.Address = x.Address
				tempResp.RequestDetails.Crypto.AddressTag = x.AddressTag.String
				tempResp.RequestDetails.Crypto.FeeAmount = x.Fee
			} else if withdraw.RequestType(v[x].WithdrawType) == withdraw.Fiat {
				x, err := v[x].WithdrawalFiatWithdrawalFiats().One(ctx, database.DB.SQL)
				if err != nil {
					return nil, err
				}
				tempResp.RequestDetails.Fiat.Bank.AccountName = x.BankAccountName
				tempResp.RequestDetails.Fiat.Bank.AccountNumber = x.BankAccountNumber
				tempResp.RequestDetails.Fiat.Bank.IBAN = x.Iban
				tempResp.RequestDetails.Fiat.Bank.SWIFTCode = x.SwiftCode
				tempResp.RequestDetails.Fiat.Bank.BSBNumber = x.BSB
			}
			resp = append(resp, tempResp)
		}
	}
	if len(resp) == 0 {
		return nil, common.ErrNoResults
	}
	return resp, nil
}
