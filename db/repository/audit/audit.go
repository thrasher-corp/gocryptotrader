package audit

import (
	"github.com/thrasher-/gocryptotrader/db"
	"github.com/thrasher-/gocryptotrader/db/models"
)

type Repository interface {
	AddEvent(event *models.Event)
}

var (
	Audit Repository
)

func Event(Type, Identifier, Message string) {
	if db.Conn.SQL == nil {
		return
	}

	tempEvent := models.Event{
		Type:    Type,
		Identifier:  Identifier,
		Message: Message}

	Audit.AddEvent(&tempEvent)
}
