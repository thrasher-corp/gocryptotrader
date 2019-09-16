package audit

import (
	"context"
	"fmt"

	"github.com/volatiletech/sqlboiler/boil"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/models"
)

func Event(id, msgtype, message string) {
	var ctx = context.Background()
	var tempEvent = models.AuditEvent{
		Type:       msgtype,
		Identifier: id,
		Message:    message,
	}
	fmt.Println(tempEvent)

	err := tempEvent.Insert(ctx, database.DB.SQL, boil.Infer())
	if err != nil {
		fmt.Println(err)
	}
}

func AllEvents() {
	var ctx context.Context
	x, err := models.AuditEvents().All(ctx, database.DB.SQL)
	if err != nil {
		fmt.Println(err)
	}
	for event := range x {
		fmt.Println(x[event])
	}
}
