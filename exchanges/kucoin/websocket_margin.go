package kucoin

import (
	"context"
)

// processCrossMarginPosition relays a Cross Margin Position push. Its debt.ratio and position.status pushes share one
// type, as the docs give a position.status example under the debt.ratio subject
func (e *Exchange) processCrossMarginPosition(ctx context.Context, msg *WebsocketMessage) error {
	switch msg.Subject {
	case "debt.ratio", "position.status":
		return processPayload[WebsocketCrossMarginPosition](ctx, e, msg.Data)
	}
	return unhandledSubject(msg)
}

// processIsolatedMarginPosition relays an Isolated Margin Position push
func (e *Exchange) processIsolatedMarginPosition(ctx context.Context, msg *WebsocketMessage) error {
	if msg.Subject != "positionChange" {
		return unhandledSubject(msg)
	}
	return processPayload[WebsocketIsolatedMarginPosition](ctx, e, msg.Data)
}
