import { Injectable } from '@angular/core';
import { Observable, Subject } from 'rxjs/Rx';
import { WebsocketService } from './../../services/websocket/websocket.service';

const WEBSOCKET_URL = 'ws://localhost:9050/ws';

export interface Message {
	Event: string,
	data:any,
	Exchange:string,
	AssetType:string
}

@Injectable()
export class WebsocketHandlerService {
	public messages: Subject<any>;

	private authenticateMessage = {
		Event:'auth',
		data:{"username":"admin","password":"e7cf3ef4f17c3999a94f2c6f612e8a888e5b1026878e4e19398b23bd38ec221a"},
	  }

	public authenticate() {
		this.messages.next(this.authenticateMessage);
	}

	constructor(wsService: WebsocketService) {
		this.messages = <Subject<Message>>wsService
			.connect(WEBSOCKET_URL) 
			.map((response: MessageEvent): Message => {

				let data = JSON.parse(response.data);
				// variables aren't consistent yet. Here's a hack!
				var dataData = data.Data === undefined ? data.data : data.Data;
				var eventEvent = data.Event === undefined ? data.event : data.Event;
				return {
					Event: eventEvent,
					data: dataData,
					Exchange: data.exchange,
					AssetType: data.assetType
				}
			});
		}
}