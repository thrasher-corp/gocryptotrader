-- +goose Up
-- GateIO is now named Gate, so its stored candles, trades, withdrawals and data history jobs follow it to the new name.
-- A gate row with no history, as seeding the new exchange list before migrating leaves, makes way for the rename. A gate
-- row with history of its own is kept, with the gateio row beside it, as merging the two could duplicate candles and
-- trades.
DELETE FROM exchange WHERE name = 'gate'
    AND EXISTS (SELECT 1 FROM exchange WHERE name = 'gateio')
    AND NOT EXISTS (SELECT 1 FROM candle WHERE candle.exchange_name_id = exchange.id)
    AND NOT EXISTS (SELECT 1 FROM trade WHERE trade.exchange_name_id = exchange.id)
    AND NOT EXISTS (SELECT 1 FROM withdrawal_history WHERE withdrawal_history.exchange_name_id = exchange.id)
    AND NOT EXISTS (SELECT 1 FROM datahistoryjob WHERE datahistoryjob.exchange_name_id = exchange.id OR datahistoryjob.secondary_exchange_id = exchange.id);
UPDATE exchange SET name = 'gate' WHERE name = 'gateio' AND NOT EXISTS (SELECT 1 FROM exchange WHERE name = 'gate');

-- +goose Down
DELETE FROM exchange WHERE name = 'gateio'
    AND EXISTS (SELECT 1 FROM exchange WHERE name = 'gate')
    AND NOT EXISTS (SELECT 1 FROM candle WHERE candle.exchange_name_id = exchange.id)
    AND NOT EXISTS (SELECT 1 FROM trade WHERE trade.exchange_name_id = exchange.id)
    AND NOT EXISTS (SELECT 1 FROM withdrawal_history WHERE withdrawal_history.exchange_name_id = exchange.id)
    AND NOT EXISTS (SELECT 1 FROM datahistoryjob WHERE datahistoryjob.exchange_name_id = exchange.id OR datahistoryjob.secondary_exchange_id = exchange.id);
UPDATE exchange SET name = 'gateio' WHERE name = 'gate' AND NOT EXISTS (SELECT 1 FROM exchange WHERE name = 'gateio');
