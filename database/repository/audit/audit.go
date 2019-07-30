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

func Event(Type, Identifier, Message string) {
	if database.Conn.SQL == nil {
		return
	}

	tempEvent := models.Event{
		Type:    Type,
		Identifier:  Identifier,
		Message: Message}

	Audit.AddEvent(&tempEvent)
}
