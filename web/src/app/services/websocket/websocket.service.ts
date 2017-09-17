import { Injectable } from '@angular/core';
import * as Rx from 'rxjs/Rx';

@Injectable()
export class WebsocketService {
  constructor() { }

  private subject: Rx.Subject<MessageEvent>;

  public connect(url): Rx.Subject<MessageEvent> {
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

  private create(url): Rx.Subject<MessageEvent> {
    let ws = new WebSocket(url);
    
    let observable = Rx.Observable.create(
	(obs: Rx.Observer<MessageEvent>) => {
		ws.onmessage = obs.next.bind(obs);
		ws.onerror = obs.error.bind(obs);
    ws.onclose = obs.complete.bind(obs);
		return ws.close.bind(ws);
	})
let observer = {
		next: (data: Object) => {
    if (ws.readyState === WebSocket.OPEN) {
        if(!this.isAuth) {
          //This is a shit initial way to be able to authenticate
          ws.send(JSON.stringify(this.authenticateMessage));
          this.isAuth = true;
        }
				ws.send(JSON.stringify(data));
      }
  }
  }
	return Rx.Subject.create(observer, observable);
  }
}