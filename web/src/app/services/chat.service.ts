import { Injectable } from '@angular/core';
import { Observable, Subject } from 'rxjs/Rx';
import { WebsocketService } from './websocket/websocket.service';

const CHAT_URL = 'ws://localhost:9050/ws';

export interface Message {
	Event: string,
	data:object,
}

@Injectable()
export class ChatService {
	public messages: Subject<Message>;

	constructor(wsService: WebsocketService) {
		this.messages = <Subject<Message>>wsService
			.connect(CHAT_URL) 
			.map((response: MessageEvent): Message => {
				let data = JSON.parse(response.data);
				return {
					Event: data.Event,
					data: data.data,
				}
			});
	}
}