package audit

import (
	"github.com/thrasher-/gocryptotrader/database"
	"github.com/thrasher-/gocryptotrader/database/models"
	"github.com/thrasher-/gocryptotrader/database/repository/audit"

	log "github.com/thrasher-/gocryptotrader/logger"
)

type auditRepo struct {
}

func Audit() audit.Repository {
	return &auditRepo{}
}

func (pg *auditRepo) AddEvent(event *models.Event) {
	if pg == nil {
		return
	}
	query := `INSERT INTO audit (type, identifier, message) VALUES($1, $2, $3)`
	tx, err := database.Conn.SQL.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(query, &event.Type, &event.Identifier, &event.Message)
	if err != nil {
		_ = tx.Rollback()
		return
	}
	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		log.Errorf(log.Global, "Failed to write audit event: %v\n", err)
		return
	}
}
