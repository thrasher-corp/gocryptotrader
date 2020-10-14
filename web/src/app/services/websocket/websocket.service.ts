import { Optional, SkipSelf, NgModule } from '@angular/core';
import { Subject, Observable, Observer } from 'rxjs';
import { WebSocketMessage } from './../../shared/classes/websocket';

@NgModule()
export class WebsocketService {
  public isConnected = false;
  constructor (@Optional() @SkipSelf() parentModule: WebsocketService) {
    if (parentModule) {
      throw new Error(
        'WebsocketService is already loaded. Import it in the AppModule only');
    }
  }

  private subject: Subject<MessageEvent>;

  public connect(url): Subject<MessageEvent> {
    if (!this.subject) {
      this.subject = this.create(url);
    }
    return this.subject;
  }

  private create(url): Subject<MessageEvent> {
    const ws = new WebSocket(url);
    const observable = Observable.create(
      (obs: Observer<MessageEvent>) => {
        ws.onmessage = obs.next.bind(obs);
        ws.onerror = obs.error.bind(obs);
        ws.onclose = () => {
          this.isConnected = false;
          obs.complete.bind(obs); };
        ws.onopen = () => {
          this.isConnected = true;
          ws.send(JSON.stringify(WebSocketMessage.CreateAuthenticationMessage()));
        };
        return ws.close.bind(ws);
      });
    const observer = {
      next: (data: any) => {
        let counter = 0;
        const interval = setInterval(() => {
          if (counter === 10) {
            clearInterval(interval);
          }
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify(data));
            clearInterval(interval);
            this.isConnected = true;
          }
          counter++;
        }, 400);

        if (ws.readyState !== WebSocket.OPEN) {
          this.isConnected = false;
          throw new Error('Failed to send message to websocket after 10 attempts');
        }
      }
    };
    return Subject.create(observer, observable);
  }
}
