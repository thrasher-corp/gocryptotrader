import { Injectable } from '@angular/core';
import { Observable, Subject } from 'rxjs/Rx';
import { WebsocketService } from './../../services/websocket/websocket.service';
import { WebSocketMessage } from './../../shared/classes/websocket';

const WEBSOCKET_URL = 'ws://localhost:9050/ws';

@Injectable()
export class WebsocketResponseHandlerService {
	public messages: Subject<any>;

	constructor(wsService: WebsocketService) {
		this.messages = <Subject<WebSocketMessage>>wsService
			.connect(WEBSOCKET_URL)
			.map((response: MessageEvent): WebSocketMessage => {
				let websocketResponseMessage = JSON.parse(response.data);
				var websocketResponseData = websocketResponseMessage.Data === undefined ? websocketResponseMessage.data : websocketResponseMessage.Data;
				var websocketResponseEvent = websocketResponseMessage.Event === undefined ? websocketResponseMessage.event : websocketResponseMessage.Event;
				let responseMessage = new WebSocketMessage();
				
				responseMessage.event = websocketResponseEvent;
				responseMessage.data = websocketResponseData;
				responseMessage.exchange = websocketResponseMessage.exchange;
				responseMessage.assetType = websocketResponseMessage.assetType;

				return responseMessage;
			}
		);
	}
}