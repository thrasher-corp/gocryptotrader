package audit

import (
	"github.com/thrasher-/gocryptotrader/database"
	"github.com/thrasher-/gocryptotrader/database/models"
)

type Repository interface {
	AddEvent(event *models.Event)
}

var (
	Audit Repository
)

func Event(msgType, identifier, message string) {
	if database.Conn.SQL == nil {
		return
	}

	tempEvent := models.Event{
		Type:       msgType,
		Identifier: identifier,
		Message:    message}

	Audit.AddEvent(&tempEvent)
}
