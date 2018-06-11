
import {share, map} from 'rxjs/operators';
import { NgModule, Injectable, Optional, SkipSelf } from '@angular/core';
import { Observable, Subject } from 'rxjs/Rx';
import { WebsocketService } from './../../services/websocket/websocket.service';
import { WebSocketMessage } from './../../shared/classes/websocket';

const WEBSOCKET_URL = 'ws://localhost:9050/ws';

@NgModule({
  })
export class WebsocketResponseHandlerService {
	public messages: Subject<any>;	
	public shared: Observable<WebSocketMessage>;
	public isConnected: boolean = false;
	private ws: WebsocketService;

	constructor(@Optional() @SkipSelf() parentModule: WebsocketResponseHandlerService, wsService: WebsocketService) {
		this.ws = wsService;
		this.messages = <Subject<WebSocketMessage>>this.ws
			.connect(WEBSOCKET_URL).pipe(

			map((response: MessageEvent): WebSocketMessage => {
				var interval = setInterval(() => {
					this.isConnected = this.ws.isConnected;
				}, 2000);
				let websocketResponseMessage = JSON.parse(response.data);
				var websocketResponseData = websocketResponseMessage.Data === undefined ? websocketResponseMessage.data : websocketResponseMessage.Data;
				var websocketResponseEvent = websocketResponseMessage.Event === undefined ? websocketResponseMessage.event : websocketResponseMessage.Event;
				let responseMessage = new WebSocketMessage();
				
				responseMessage.event = websocketResponseEvent;
				responseMessage.data = websocketResponseData;
				responseMessage.exchange = websocketResponseMessage.exchange;
				responseMessage.assetType = websocketResponseMessage.assetType;
				responseMessage.error = websocketResponseMessage.error;

				return responseMessage;
			}));
			this.isConnected = this.ws.isConnected;
			
		this.shared = this.messages.pipe(share()); //multicast
	}
}