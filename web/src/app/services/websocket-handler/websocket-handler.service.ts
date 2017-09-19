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
	public messages: Subject<Message>;

	constructor(wsService: WebsocketService) {
		this.messages = <Subject<Message>>wsService
			.connect(WEBSOCKET_URL) 
			.map((response: MessageEvent): Message => {

				let data = JSON.parse(response.data);

				// debug only
				//console.log('Recieved response from websocket. Data: ' + JSON.stringify(data))
				var dataData = data.Data === null ? data.data : data.Data;
				return {
					Event: data.Event,
					data: dataData,
					Exchange: data.exchange,
					AssetType: data.assetType
				}
			});
		}
}