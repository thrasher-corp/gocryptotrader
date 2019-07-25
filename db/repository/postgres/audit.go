package audit

import (
	"github.com/thrasher-/gocryptotrader/db"
	"github.com/thrasher-/gocryptotrader/db/models"
	"github.com/thrasher-/gocryptotrader/db/repository"
)

type auditRepo struct {
}

func NewPSQLAudit() repository.AuditRepository {
	return &auditRepo{}
}

func (pg *auditRepo) AddEvent(event models.Event) error {
	query := `INSERT INTO audit (type, identifier, message) VALUES($1, $2, $3)`
	_, err := db.DBConn.SQL.Exec(query, event.Type, event.UserID, event.Message)
	if err != nil {
		return err
	}
	return nil
}
