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
	_, err := database.Conn.SQL.Exec(query, &event.Type, &event.Identifier, &event.Message)
	if err != nil {
		log.Errorf(log.AuditMgr, "Failed to write audit event: %v", err)
	}
}
