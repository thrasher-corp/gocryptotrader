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

func Event(Type, UserID, Message string) {
	if db.Conn.SQL == nil {
		return
	}

	tempEvent := models.Event{
		Type:    Type,
		UserID:  UserID,
		Message: Message}

	Audit.AddEvent(&tempEvent)
}
