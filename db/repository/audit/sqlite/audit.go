package audit

import (
	"fmt"
	"github.com/thrasher-/gocryptotrader/db"
	"github.com/thrasher-/gocryptotrader/db/models"
	"github.com/thrasher-/gocryptotrader/db/repository/audit"
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
	_, err := db.Conn.SQL.Exec(query, &event.Type, &event.Identifier, &event.Message)
	if err != nil {
		fmt.Println("Failed to write audit event")
	}
}
