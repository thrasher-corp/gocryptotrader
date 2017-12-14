import { Injectable } from '@angular/core';
import {Subject, Observable, Observer } from 'rxjs/Rx';

@Injectable()
export class WebsocketService {
  constructor() { }

  private subject: Subject<MessageEvent>;

  public connect(url): Subject<MessageEvent> {
    if (!this.subject) {
      this.subject = this.create(url);
    } 
    return this.subject;
  }

  private authenticateMessage = {
		Event:'auth',
		data:{"username":"admin","password":"e7cf3ef4f17c3999a94f2c6f612e8a888e5b1026878e4e19398b23bd38ec221a"},
	  }

  private isAuth = false;

  private create(url): Subject<MessageEvent> {
    let ws = new WebSocket(url);
    
    let observable = Observable.create(
	(obs: Observer<MessageEvent>) => {
		ws.onmessage = obs.next.bind(obs);
		ws.onerror = obs.error.bind(obs);
    ws.onclose = obs.complete.bind(obs);
		return ws.close.bind(ws);
	})
let observer = {
		next: (data: any) => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(this.authenticateMessage));
      
				ws.send(JSON.stringify(data));
      }
  }
  }
	return Subject.create(observer, observable);
  }
}