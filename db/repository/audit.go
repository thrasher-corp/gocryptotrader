package repository

import (
	"github.com/thrasher-/gocryptotrader/db/models"
)

type AuditRepository interface {
	AddEvent(event models.Event) error
}
