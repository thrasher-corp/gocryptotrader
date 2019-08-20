package audit

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/models"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Repository that is required for each driver type to implement
type Repository interface {
	AddEventTx(event []*models.AuditEvent)
}

var (
	// Audit repository initialise copy of Audit Repository
	Audit Repository
)

type eventPool struct {
	events  []*models.AuditEvent
	eventMu sync.Mutex
}

var ep eventPool

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

	ep.poolEvents(&tempEvent)
}

func (e *eventPool) poolEvents(event *models.AuditEvent) {
	e.eventMu.Lock()
	defer e.eventMu.Unlock()

	e.events = append(e.events, event)

	database.Conn.Mu.RLock()
	defer database.Conn.Mu.RUnlock()

	if !database.Conn.Connected {
		log.Warnln(log.DatabaseMgr, "connection to database interrupted pooling database writes")
		return
	}

	Audit.AddEventTx(e.events)
	e.events = nil
}
