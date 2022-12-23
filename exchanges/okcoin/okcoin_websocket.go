package okcoin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// WsConnect initiates a websocket connection
func (o *OKCoin) WsConnect() error {
	if !o.Websocket.IsEnabled() || !o.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192
	err := o.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if o.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			o.Websocket.GetWebsocketURL())
	}

	o.Websocket.Wg.Add(1)
	go o.WsReadData()

	if o.IsWebsocketAuthenticationSupported() {
		err = o.WsLogin(context.TODO())
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v - authentication failed: %v\n",
				o.Name,
				err)
		}
	}

	return nil
}

// WsLogin sends a login request to websocket to enable access to authenticated endpoints
func (o *OKCoin) WsLogin(ctx context.Context) error {
	creds, err := o.GetCredentials(ctx)
	if err != nil {
		return err
	}
	o.Websocket.SetCanUseAuthenticatedEndpoints(true)
	unixTime := time.Now().UTC().Unix()
	signPath := "/users/self/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(strconv.FormatInt(unixTime, 10)+http.MethodGet+signPath),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	base64 := crypto.Base64Encode(hmac)
	request := WebsocketEventRequest{
		Operation: "login",
		Arguments: []string{
			creds.Key,
			creds.ClientID,
			strconv.FormatInt(unixTime, 10),
			base64,
		},
	}
	_, err = o.Websocket.Conn.SendMessageReturnResponse("login", request)
	if err != nil {
		o.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// WsReadData receives and passes on websocket messages for processing
func (o *OKCoin) WsReadData() {
	defer o.Websocket.Wg.Done()

	for {
		resp := o.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := o.WsHandleData(resp.Raw)
		if err != nil {
			o.Websocket.DataHandler <- err
		}
	}
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (o *OKCoin) WsHandleData(respRaw []byte) error {
	var dataResponse WebsocketDataResponse
	err := json.Unmarshal(respRaw, &dataResponse)
	if err != nil {
		return err
	}
	if len(dataResponse.Data) > 0 {
		switch o.GetWsChannelWithoutOrderType(dataResponse.Table) {
		case okcoinWsCandle60s, okcoinWsCandle180s, okcoinWsCandle300s,
			okcoinWsCandle900s, okcoinWsCandle1800s, okcoinWsCandle3600s,
			okcoinWsCandle7200s, okcoinWsCandle14400s, okcoinWsCandle21600s,
			okcoinWsCandle43200s, okcoinWsCandle86400s, okcoinWsCandle604900s:
			return o.wsProcessCandles(respRaw)
		case okcoinWsDepth, okcoinWsDepth5:
			return o.WsProcessOrderBook(respRaw)
		case okcoinWsTicker:
			return o.wsProcessTickers(respRaw)
		case okcoinWsTrade:
			return o.wsProcessTrades(respRaw)
		case okcoinWsOrder:
			return o.wsProcessOrder(respRaw)
		}
		o.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: o.Name + stream.UnhandledMessage + string(respRaw),
		}
		return nil
	}

	var errorResponse WebsocketErrorResponse
	err = json.Unmarshal(respRaw, &errorResponse)
	if err == nil && errorResponse.ErrorCode > 0 {
		return fmt.Errorf("%v error - %v message: %s ",
			o.Name,
			errorResponse.ErrorCode,
			errorResponse.Message)
	}
	var eventResponse WebsocketEventResponse
	err = json.Unmarshal(respRaw, &eventResponse)
	if err == nil && eventResponse.Event != "" {
		if eventResponse.Event == "login" {
			if o.Websocket.Match.Incoming("login") {
				o.Websocket.SetCanUseAuthenticatedEndpoints(eventResponse.Success)
			}
		}
		if o.Verbose {
			log.Debug(log.ExchangeSys,
				o.Name+" - "+eventResponse.Event+" on channel: "+eventResponse.Channel)
		}
	}
	return nil
}

// StringToOrderStatus converts order status IDs to internal types
func StringToOrderStatus(num int64) (order.Status, error) {
	switch num {
	case -2:
		return order.Rejected, nil
	case -1:
		return order.Cancelled, nil
	case 0:
		return order.Active, nil
	case 1:
		return order.PartiallyFilled, nil
	case 2:
		return order.Filled, nil
	case 3:
		return order.New, nil
	case 4:
		return order.PendingCancel, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%v not recognised as order status", num)
	}
}

func (o *OKCoin) wsProcessOrder(respRaw []byte) error {
	var resp WebsocketSpotOrderResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	for i := range resp.Data {
		var oType order.Type
		var oSide order.Side
		var oStatus order.Status
		oType, err = order.StringToOrderType(resp.Data[i].Type)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}
		oSide, err = order.StringToOrderSide(resp.Data[i].Side)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}
		oStatus, err = StringToOrderStatus(resp.Data[i].State)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}

		pair, err := currency.NewPairFromString(resp.Data[i].InstrumentID)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}

		o.Websocket.DataHandler <- &order.Detail{
			ImmediateOrCancel: resp.Data[i].OrderType == 3,
			FillOrKill:        resp.Data[i].OrderType == 2,
			PostOnly:          resp.Data[i].OrderType == 1,
			Price:             resp.Data[i].Price,
			Amount:            resp.Data[i].Size,
			ExecutedAmount:    resp.Data[i].LastFillQty,
			RemainingAmount:   resp.Data[i].Size - resp.Data[i].LastFillQty,
			Exchange:          o.Name,
			OrderID:           resp.Data[i].OrderID,
			Type:              oType,
			Side:              oSide,
			Status:            oStatus,
			AssetType:         o.GetAssetTypeFromTableName(resp.Table),
			Date:              resp.Data[i].CreatedAt,
			Pair:              pair,
		}
	}
	return nil
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (o *OKCoin) wsProcessTickers(respRaw []byte) error {
	var response WebsocketTickerData
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	a := o.GetAssetTypeFromTableName(response.Table)
	for i := range response.Data {
		f := strings.Split(response.Data[i].InstrumentID, delimiterDash)

		c := currency.NewPairWithDelimiter(f[0], f[1], delimiterDash)

		baseVolume := response.Data[i].BaseVolume24h
		if response.Data[i].ContractVolume24h != 0 {
			baseVolume = response.Data[i].ContractVolume24h
		}

		quoteVolume := response.Data[i].QuoteVolume24h
		if response.Data[i].TokenVolume24h != 0 {
			quoteVolume = response.Data[i].TokenVolume24h
		}

		o.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: o.Name,
			Open:         response.Data[i].Open24h,
			Close:        response.Data[i].Last,
			Volume:       baseVolume,
			QuoteVolume:  quoteVolume,
			High:         response.Data[i].High24h,
			Low:          response.Data[i].Low24h,
			Bid:          response.Data[i].BestBid,
			Ask:          response.Data[i].BestAsk,
			Last:         response.Data[i].Last,
			AssetType:    a,
			Pair:         c,
			LastUpdated:  response.Data[i].Timestamp,
		}
	}
	return nil
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (o *OKCoin) wsProcessTrades(respRaw []byte) error {
	if !o.IsSaveTradeDataEnabled() {
		return nil
	}
	var response WebsocketTradeResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}

	a := o.GetAssetTypeFromTableName(response.Table)
	trades := make([]trade.Data, len(response.Data))
	for i := range response.Data {
		f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
		c := currency.NewPairWithDelimiter(f[0], f[1], delimiterDash)

		tSide, err := order.StringToOrderSide(response.Data[i].Side)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				Err:      err,
			}
		}

		amount := response.Data[i].Size
		if response.Data[i].Quantity != 0 {
			amount = response.Data[i].Quantity
		}
		trades[i] = trade.Data{
			Amount:       amount,
			AssetType:    a,
			CurrencyPair: c,
			Exchange:     o.Name,
			Price:        response.Data[i].Price,
			Side:         tSide,
			Timestamp:    response.Data[i].Timestamp,
			TID:          response.Data[i].TradeID,
		}
	}
	return trade.AddTradesToBuffer(o.Name, trades...)
}

// wsProcessCandles converts candle data and sends it to the data handler
func (o *OKCoin) wsProcessCandles(respRaw []byte) error {
	var response WebsocketCandleResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}

	a := o.GetAssetTypeFromTableName(response.Table)
	for i := range response.Data {
		f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
		c := currency.NewPairWithDelimiter(f[0], f[1], delimiterDash)

		timeData, err := time.Parse(time.RFC3339Nano,
			response.Data[i].Candle[0])
		if err != nil {
			return fmt.Errorf("%v Time data could not be parsed: %v",
				o.Name,
				response.Data[i].Candle[0])
		}

		candleIndex := strings.LastIndex(response.Table, okcoinWsCandle)
		candleInterval := response.Table[candleIndex+len(okcoinWsCandle):]

		klineData := stream.KlineData{
			AssetType: a,
			Pair:      c,
			Exchange:  o.Name,
			Timestamp: timeData,
			Interval:  candleInterval,
		}
		klineData.OpenPrice, err = strconv.ParseFloat(response.Data[i].Candle[1], 64)
		if err != nil {
			return err
		}
		klineData.HighPrice, err = strconv.ParseFloat(response.Data[i].Candle[2], 64)
		if err != nil {
			return err
		}
		klineData.LowPrice, err = strconv.ParseFloat(response.Data[i].Candle[3], 64)
		if err != nil {
			return err
		}
		klineData.ClosePrice, err = strconv.ParseFloat(response.Data[i].Candle[4], 64)
		if err != nil {
			return err
		}
		klineData.Volume, err = strconv.ParseFloat(response.Data[i].Candle[5], 64)
		if err != nil {
			return err
		}
		o.Websocket.DataHandler <- klineData
	}
	return nil
}

// WsProcessOrderBook Validates the checksum and updates internal orderbook values
func (o *OKCoin) WsProcessOrderBook(respRaw []byte) error {
	var response WebsocketOrderBooksData
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	orderbookMutex.Lock()
	defer orderbookMutex.Unlock()
	a := o.GetAssetTypeFromTableName(response.Table)
	for i := range response.Data {
		f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
		c := currency.NewPairWithDelimiter(f[0], f[1], delimiterDash)

		if response.Action == okcoinWsOrderbookPartial {
			err := o.WsProcessPartialOrderBook(&response.Data[i], c, a)
			if err != nil {
				err2 := o.wsResubscribeToOrderbook(&response)
				if err2 != nil {
					o.Websocket.DataHandler <- err2
				}
				return err
			}
		} else if response.Action == okcoinWsOrderbookUpdate {
			if len(response.Data[i].Asks) == 0 && len(response.Data[i].Bids) == 0 {
				return nil
			}
			err := o.WsProcessUpdateOrderbook(&response.Data[i], c, a)
			if err != nil {
				err2 := o.wsResubscribeToOrderbook(&response)
				if err2 != nil {
					o.Websocket.DataHandler <- err2
				}
				return err
			}
		}
	}
	return nil
}

func (o *OKCoin) wsResubscribeToOrderbook(response *WebsocketOrderBooksData) error {
	a := o.GetAssetTypeFromTableName(response.Table)
	for i := range response.Data {
		f := strings.Split(response.Data[i].InstrumentID, delimiterDash)

		c := currency.NewPairWithDelimiter(f[0], f[1], delimiterDash)

		channelToResubscribe := &stream.ChannelSubscription{
			Channel:  response.Table,
			Currency: c,
			Asset:    a,
		}
		err := o.Websocket.ResubscribeToChannel(channelToResubscribe)
		if err != nil {
			return fmt.Errorf("%s resubscribe to orderbook error %s", o.Name, err)
		}
	}
	return nil
}

// AppendWsOrderbookItems adds websocket orderbook data bid/asks into an
// orderbook item array
func (o *OKCoin) AppendWsOrderbookItems(entries [][]interface{}) ([]orderbook.Item, error) {
	items := make([]orderbook.Item, len(entries))
	for j := range entries {
		amount, err := strconv.ParseFloat(entries[j][1].(string), 64)
		if err != nil {
			return nil, err
		}
		price, err := strconv.ParseFloat(entries[j][0].(string), 64)
		if err != nil {
			return nil, err
		}
		items[j] = orderbook.Item{Amount: amount, Price: price}
	}
	return items, nil
}

// WsProcessPartialOrderBook takes websocket orderbook data and creates an
// orderbook Calculates checksum to ensure it is valid
func (o *OKCoin) WsProcessPartialOrderBook(wsEventData *WebsocketOrderBook, instrument currency.Pair, a asset.Item) error {
	signedChecksum, err := o.CalculatePartialOrderbookChecksum(wsEventData)
	if err != nil {
		return fmt.Errorf("%s channel: %s. Orderbook unable to calculate partial orderbook checksum: %s",
			o.Name,
			a,
			err)
	}
	if signedChecksum != wsEventData.Checksum {
		return fmt.Errorf("%s channel: %s. Orderbook partial for %v checksum invalid",
			o.Name,
			a,
			instrument)
	}
	if o.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s passed checksum for instrument %s",
			o.Name,
			instrument)
	}

	asks, err := o.AppendWsOrderbookItems(wsEventData.Asks)
	if err != nil {
		return err
	}

	bids, err := o.AppendWsOrderbookItems(wsEventData.Bids)
	if err != nil {
		return err
	}

	newOrderBook := orderbook.Base{
		Asks:            asks,
		Bids:            bids,
		Asset:           a,
		LastUpdated:     wsEventData.Timestamp,
		Pair:            instrument,
		Exchange:        o.Name,
		VerifyOrderbook: o.CanVerifyOrderbook,
	}
	return o.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// WsProcessUpdateOrderbook updates an existing orderbook using websocket data
// After merging WS data, it will sort, validate and finally update the existing
// orderbook
func (o *OKCoin) WsProcessUpdateOrderbook(wsEventData *WebsocketOrderBook, instrument currency.Pair, a asset.Item) error {
	update := orderbook.Update{
		Asset:      a,
		Pair:       instrument,
		UpdateTime: wsEventData.Timestamp,
	}

	var err error
	update.Asks, err = o.AppendWsOrderbookItems(wsEventData.Asks)
	if err != nil {
		return err
	}
	update.Bids, err = o.AppendWsOrderbookItems(wsEventData.Bids)
	if err != nil {
		return err
	}

	err = o.Websocket.Orderbook.Update(&update)
	if err != nil {
		return err
	}

	updatedOb, err := o.Websocket.Orderbook.GetOrderbook(instrument, a)
	if err != nil {
		return err
	}
	checksum := o.CalculateUpdateOrderbookChecksum(updatedOb)

	if checksum != wsEventData.Checksum {
		// re-sub
		log.Warnf(log.ExchangeSys, "%s checksum failure for item %s",
			o.Name,
			wsEventData.InstrumentID)
		return errors.New("checksum failed")
	}
	return nil
}

// CalculatePartialOrderbookChecksum alternates over the first 25 bid and ask
// entries from websocket data. The checksum is made up of the price and the
// quantity with a semicolon (:) deliminating them. This will also work when
// there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (o *OKCoin) CalculatePartialOrderbookChecksum(orderbookData *WebsocketOrderBook) (int32, error) {
	var checksum strings.Builder
	for i := 0; i < allowableIterations; i++ {
		if len(orderbookData.Bids)-1 >= i {
			bidPrice, ok := orderbookData.Bids[i][0].(string)
			if !ok {
				return 0, fmt.Errorf("unable to type assert bidPrice")
			}
			bidAmount, ok := orderbookData.Bids[i][1].(string)
			if !ok {
				return 0, fmt.Errorf("unable to type assert bidAmount")
			}
			checksum.WriteString(bidPrice +
				delimiterColon +
				bidAmount +
				delimiterColon)
		}
		if len(orderbookData.Asks)-1 >= i {
			askPrice, ok := orderbookData.Asks[i][0].(string)
			if !ok {
				return 0, fmt.Errorf("unable to type assert askPrice")
			}
			askAmount, ok := orderbookData.Asks[i][1].(string)
			if !ok {
				return 0, fmt.Errorf("unable to type assert askAmount")
			}
			checksum.WriteString(askPrice +
				delimiterColon +
				askAmount +
				delimiterColon)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), delimiterColon)
	return int32(crc32.ChecksumIEEE([]byte(checksumStr))), nil
}

// CalculateUpdateOrderbookChecksum alternates over the first 25 bid and ask
// entries of a merged orderbook. The checksum is made up of the price and the
// quantity with a semicolon (:) deliminating them. This will also work when
// there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (o *OKCoin) CalculateUpdateOrderbookChecksum(orderbookData *orderbook.Base) int32 {
	var checksum strings.Builder
	for i := 0; i < allowableIterations; i++ {
		if len(orderbookData.Bids)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Bids[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Bids[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + delimiterColon + amount + delimiterColon)
		}
		if len(orderbookData.Asks)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Asks[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Asks[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + delimiterColon + amount + delimiterColon)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), delimiterColon)
	return int32(crc32.ChecksumIEEE([]byte(checksumStr)))
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be
// handled by ManageSubscriptions()
func (o *OKCoin) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	assets := o.GetAssetTypes(true)
	for x := range assets {
		pairs, err := o.GetEnabledPairs(assets[x])
		if err != nil {
			return nil, err
		}

		if assets[x] != asset.Spot {
			o.Websocket.DataHandler <- fmt.Errorf("%w %v", asset.ErrNotSupported, assets[x])
			return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assets[x])
		}
		channels := defaultSpotSubscribedChannels
		if o.IsWebsocketAuthenticationSupported() {
			channels = append(channels,
				okcoinWsSpotMarginAccount,
				okcoinWsSpotAccount,
				okcoinWsSpotOrder)
		}
		for i := range pairs {
			p, err := o.FormatExchangeCurrency(pairs[i], asset.Spot)
			if err != nil {
				return nil, err
			}
			for y := range channels {
				subscriptions = append(subscriptions,
					stream.ChannelSubscription{
						Channel:  channels[y],
						Currency: p,
						Asset:    asset.Spot,
					})
			}
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (o *OKCoin) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	return o.handleSubscriptions("subscribe", channelsToSubscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (o *OKCoin) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return o.handleSubscriptions("unsubscribe", channelsToUnsubscribe)
}

func (o *OKCoin) handleSubscriptions(operation string, subs []stream.ChannelSubscription) error {
	request := WebsocketEventRequest{
		Operation: operation,
	}

	var channels []stream.ChannelSubscription
	for i := 0; i < len(subs); i++ {
		// Temp type to evaluate max byte len after a marshal on batched unsubs
		temp := WebsocketEventRequest{
			Operation: operation,
		}
		temp.Arguments = make([]string, len(request.Arguments))
		copy(temp.Arguments, request.Arguments)

		arg := subs[i].Channel + delimiterColon
		if strings.EqualFold(subs[i].Channel, okcoinWsSpotAccount) {
			arg += subs[i].Currency.Base.String()
		} else {
			arg += subs[i].Currency.String()
		}

		temp.Arguments = append(temp.Arguments, arg)
		chunk, err := json.Marshal(request)
		if err != nil {
			return err
		}

		if len(chunk) > maxConnByteLen {
			// If temp chunk exceeds max byte length determined by the exchange,
			// commit last payload.
			i-- // reverse position in range to reuse channel unsubscription on
			// next iteration
			err = o.Websocket.Conn.SendJSONMessage(request)
			if err != nil {
				return err
			}

			if operation == "unsubscribe" {
				o.Websocket.RemoveSuccessfulUnsubscriptions(channels...)
			} else {
				o.Websocket.AddSuccessfulSubscriptions(channels...)
			}

			// Drop prior unsubs and chunked payload args on successful unsubscription
			channels = nil
			request.Arguments = nil
			continue
		}
		// Add pending chained items
		channels = append(channels, subs[i])
		request.Arguments = temp.Arguments
	}

	// Commit left overs to payload
	err := o.Websocket.Conn.SendJSONMessage(request)
	if err != nil {
		return err
	}

	if operation == "unsubscribe" {
		o.Websocket.RemoveSuccessfulUnsubscriptions(channels...)
	} else {
		o.Websocket.AddSuccessfulSubscriptions(channels...)
	}
	return nil
}

// GetWsChannelWithoutOrderType takes WebsocketDataResponse.Table and returns
// The base channel name eg receive "spot/depth5:BTC-USDT" return "depth5"
func (o *OKCoin) GetWsChannelWithoutOrderType(table string) string {
	index := strings.Index(table, "/")
	if index == -1 {
		return table
	}
	channel := table[index+1:]
	index = strings.Index(channel, ":")
	// Some events do not contain a currency
	if index == -1 {
		return channel
	}

	return channel[:index]
}

// GetAssetTypeFromTableName gets the asset type from the table name
// eg "spot/ticker:BTCUSD" results in "SPOT"
func (o *OKCoin) GetAssetTypeFromTableName(table string) asset.Item {
	assetIndex := strings.Index(table, "/")
	switch table[:assetIndex] {
	case asset.Spot.String():
		return asset.Spot
	default:
		log.Warnf(log.ExchangeSys, "%s unhandled asset type %s",
			o.Name,
			table[:assetIndex])
		return asset.Empty
	}
}
