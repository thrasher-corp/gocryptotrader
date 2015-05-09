package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
)

const (
	ITEM_PRICE            = "PRICE"
	GREATER_THAN          = ">"
	GREATER_THAN_OR_EQUAL = ">="
	LESS_THAN             = "<"
	LESS_THAN_OR_EQUAL    = "<="
	IS_EQUAL              = "=="
	ACTION_SMS_NOTIFY     = "SMS"
	ACTION_CONSOLE_PRINT  = "CONSOLE_PRINT"
)

var (
	ErrInvalidItem      = errors.New("Invalid item.")
	ErrInvalidCondition = errors.New("Invalid conditional option.")
	ErrInvalidAction    = errors.New("Invalid action.")
	ErrExchangeDisabled = errors.New("Desired exchange is disabled.")
)

type Event struct {
	ID        int
	Exchange  string
	Item      string
	Condition string
	Action    string
	Executed  bool
}

var Events []*Event

func AddEvent(Exchange, Item, Condition, Action string) (int, error) {
	err := IsValidEvent(Exchange, Item, Condition, Action)

	if err != nil {
		return 0, err
	}

	Event := &Event{}

	if len(Events) == 0 {
		Event.ID = 0
	} else {
		Event.ID = len(Events) + 1
	}

	Event.Exchange = Exchange
	Event.Item = Item
	Event.Condition = Condition
	Event.Action = Action
	Event.Executed = false
	Events = append(Events, Event)
	return Event.ID, nil
}

func RemoveEvent(EventID int) bool {
	for i, x := range Events {
		if x.ID == EventID {
			Events = append(Events[:i], Events[i+1:]...)
			return true
		}
	}
	return false
}

func GetEventCounter() (int, int) {
	total := len(Events)
	executed := 0

	for _, x := range Events {
		if x.Executed {
			executed++
		}
	}
	return total, executed
}

func (e *Event) ExecuteAction() bool {
	if StringContains(e.Action, ",") {
		action := SplitStrings(e.Action, ",")
		if action[0] == ACTION_SMS_NOTIFY {
			message := fmt.Sprintf("Event triggered: %s", e.EventToString())
			if action[1] == "ALL" {
				SMSSendToAll(message)
			} else {
				SMSNotify(SMSGetNumberByName(action[1]), message)
			}
		}
	} else {
		log.Printf("Event triggered: %s", e.EventToString())
	}
	return true
}

func (e *Event) EventToString() string {
	condition := SplitStrings(e.Condition, ",")
	return fmt.Sprintf("If the %s on %s is %s then %s.", e.Item, e.Exchange, condition[0]+" "+condition[1], e.Action)
}

func (e *Event) CheckCondition() bool {
	lastPrice := 0.00
	condition := SplitStrings(e.Condition, ",")
	targetPrice, _ := strconv.ParseFloat(condition[1], 64)

	if bot.exchange.bitfinex.GetName() == e.Exchange {
		lastPrice = bot.exchange.bitfinex.GetTicker("btcusd").Last
	} else if bot.exchange.bitstamp.GetName() == e.Exchange {
		lastPrice = bot.exchange.bitstamp.GetTicker().Last
	} else if bot.exchange.coinbase.GetName() == e.Exchange {
		lastPrice = bot.exchange.coinbase.GetTicker("BTC-USD").Price
	} else if bot.exchange.coinbase.GetName() == e.Exchange {
		lastPrice = bot.exchange.cryptsy.GetMarkets("BTCUSD")[0].LastTrade.Price
	} else if bot.exchange.lakebtc.GetName() == e.Exchange {
		lastPrice = bot.exchange.lakebtc.GetTicker().CNY.Last
	} else if bot.exchange.btcchina.GetName() == e.Exchange {
		lastPrice = bot.exchange.btcchina.GetTicker("btccny").Last
	} else if bot.exchange.huobi.GetName() == e.Exchange {
		lastPrice = bot.exchange.huobi.GetTicker("btc").Last
	} else if bot.exchange.itbit.GetName() == e.Exchange {
		lastPrice = bot.exchange.itbit.GetTicker("XBTUSD").LastPrice
	} else if bot.exchange.btce.GetName() == e.Exchange {
		lastPrice = bot.exchange.btce.Ticker["btc_usd"].Last
	} else if bot.exchange.btcmarkets.GetName() == e.Exchange {
		lastPrice = bot.exchange.btcmarkets.Ticker["BTC"].LastPrice
	} else if bot.exchange.okcoinChina.GetName() == e.Exchange {
		lastPrice = bot.exchange.okcoinChina.GetTicker("btc_cny").Last
	} else if bot.exchange.okcoinIntl.GetName() == e.Exchange {
		lastPrice = bot.exchange.okcoinIntl.GetTicker("btc_usd").Last
	} else if bot.exchange.anx.GetName() == e.Exchange {
		lastPrice = bot.exchange.anx.GetTicker("BTCUSD").Data.Last.Value
	} else if bot.exchange.kraken.GetName() == e.Exchange {
		lastPrice = bot.exchange.kraken.Ticker["XBTUSD"].Last
	}

	if lastPrice == 0 {
		return false
	}

	switch condition[0] {
	case GREATER_THAN:
		{
			if lastPrice > targetPrice {
				return e.ExecuteAction()
			}
		}
	case GREATER_THAN_OR_EQUAL:
		{
			if lastPrice >= targetPrice {
				return e.ExecuteAction()
			}
		}
	case LESS_THAN:
		{
			if lastPrice < targetPrice {
				return e.ExecuteAction()
			}
		}
	case LESS_THAN_OR_EQUAL:
		{
			if lastPrice <= targetPrice {
				return e.ExecuteAction()
			}
		}
	case IS_EQUAL:
		{
			if lastPrice == targetPrice {
				return e.ExecuteAction()
			}
		}
	}
	return false
}

func IsValidEvent(Exchange, Item, Condition, Action string) error {
	if !IsValidExchange(Exchange) {
		return ErrExchangeDisabled
	}

	if !IsValidItem(Item) {
		return ErrInvalidItem
	}

	if !StringContains(Condition, ",") {
		return ErrInvalidCondition
	}

	condition := SplitStrings(Condition, ",")

	if !IsValidCondition(condition[0]) || len(condition[1]) == 0 {
		return ErrInvalidCondition
	}

	if StringContains(Action, ",") {
		action := SplitStrings(Action, ",")

		if action[0] != ACTION_SMS_NOTIFY {
			return ErrInvalidAction
		}

		if action[1] != "ALL" && SMSGetNumberByName(action[1]) == "" {
			return ErrInvalidAction
		}
	} else {
		if Action != ACTION_CONSOLE_PRINT {
			return ErrInvalidAction
		}
	}
	return nil
}

func CheckEvents() {
	for {
		total, executed := GetEventCounter()
		if total > 0 && executed != total {
			for _, event := range Events {
				if !event.Executed {
					success := event.CheckCondition()
					if success {
						log.Printf("Event %d triggered on %s successfully.\n", event.ID, event.Exchange)
						event.Executed = true
					}
				}
			}
		}
	}
}

func IsValidExchange(Exchange string) bool {
	if bot.exchange.bitfinex.GetName() == Exchange && bot.exchange.bitfinex.IsEnabled() ||
		bot.exchange.bitstamp.GetName() == Exchange && bot.exchange.bitstamp.IsEnabled() ||
		bot.exchange.btcchina.GetName() == Exchange && bot.exchange.btcchina.IsEnabled() ||
		bot.exchange.btce.GetName() == Exchange && bot.exchange.btcchina.IsEnabled() ||
		bot.exchange.btcmarkets.GetName() == Exchange && bot.exchange.btcmarkets.IsEnabled() ||
		bot.exchange.coinbase.GetName() == Exchange && bot.exchange.coinbase.IsEnabled() ||
		bot.exchange.cryptsy.GetName() == Exchange && bot.exchange.cryptsy.IsEnabled() ||
		bot.exchange.huobi.GetName() == Exchange && bot.exchange.huobi.IsEnabled() ||
		bot.exchange.itbit.GetName() == Exchange && bot.exchange.itbit.IsEnabled() ||
		bot.exchange.kraken.GetName() == Exchange && bot.exchange.kraken.IsEnabled() ||
		bot.exchange.lakebtc.GetName() == Exchange && bot.exchange.lakebtc.IsEnabled() ||
		bot.exchange.okcoinChina.GetName() == Exchange && bot.exchange.okcoinChina.IsEnabled() ||
		bot.exchange.okcoinIntl.GetName() == Exchange && bot.exchange.okcoinIntl.IsEnabled() ||
		bot.exchange.anx.GetName() == Exchange && bot.exchange.anx.IsEnabled() {
		return true
	}
	return false
}

func IsValidCondition(Condition string) bool {
	switch Condition {
	case GREATER_THAN, GREATER_THAN_OR_EQUAL, LESS_THAN, LESS_THAN_OR_EQUAL, IS_EQUAL:
		return true
	}
	return false
}

func IsValidAction(Action string) bool {
	switch Action {
	case ACTION_SMS_NOTIFY, ACTION_CONSOLE_PRINT:
		return true
	}
	return false
}

func IsValidItem(Item string) bool {
	switch Item {
	case ITEM_PRICE:
		return true
	}
	return false
}
