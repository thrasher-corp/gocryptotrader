import { Injectable } from '@angular/core';
import { Observable, Subject } from 'rxjs/Rx';
import { WebsocketService } from './../../services/websocket/websocket.service';
	 

const WEBSOCKET_URL = 'ws://localhost:9050/ws';

export interface Message {
	Event: string,
	data:object,
}


@Injectable()
export class WebsocketHandlerService {
  public messages: Subject<Message>;
  
   private authenticateMessage = {
    Event:'auth',
    data:{"username":"Username","password":"e7cf3ef4f17c3999a94f2c6f612e8a888e5b1026878e4e19398b23bd38ec221a"},
  }

	constructor(wsService: WebsocketService) {
		this.messages = <Subject<Message>>wsService
			.connect(WEBSOCKET_URL) 
			.map((response: MessageEvent): Message => {
        let data = JSON.parse(response.data);
        console.log('Recieved response from websocket. Event: '+ data.Event)
        
				return {
					Event: data.Event,
					data: data.data,
				}
      });
      //Auth straight away!
      console.log('Sending message to websocket. Event:'+this.authenticateMessage.Event+'. Message: ' + JSON.stringify(this.authenticateMessage.data));
      this.messages.next(this.authenticateMessage);
	}
}