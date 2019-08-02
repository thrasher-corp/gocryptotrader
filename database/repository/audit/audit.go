package audit

import (
	"github.com/thrasher-/gocryptotrader/database"
	"github.com/thrasher-/gocryptotrader/database/models"
)

// Repository that is required for each driver type to implement
type Repository interface {
	AddEvent(event *models.AuditEvent)
}

var (
	Audit Repository // Global Audit repository
)

// Event allows you to call audit.Event() as long as the audit repository package without the need to include each driver

func Event(msgType, identifier, message string) {
	if database.Conn.SQL == nil {
		return
	}

	if Audit == nil {
		return
	}

	tempEvent := models.AuditEvent{
		Type:       msgType,
		Identifier: identifier,
		Message:    message}

	Audit.AddEvent(&tempEvent)
}
