import { Injectable, Optional, SkipSelf, NgModule } from '@angular/core';
import { Subject, Observable, Observer } from 'rxjs/Rx';
import { WebSocketMessage } from './../../shared/classes/websocket';

@NgModule()
export class WebsocketService {
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
    let ws = new WebSocket(url);
    let observable = Observable.create(
      (obs: Observer<MessageEvent>) => {
        ws.onmessage = obs.next.bind(obs);
        ws.onerror = obs.error.bind(obs);
        ws.onclose = obs.complete.bind(obs);
        ws.onopen = () => {
          ws.send(JSON.stringify(WebSocketMessage.CreateAuthenticationMessage()));
        };
        return ws.close.bind(ws);
      })
    let observer = {
      next: (data: any) => {
        var counter = 0;
        var interval = setInterval(function () {
          if (counter == 10) {
            clearInterval(interval);
          }
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify(data));
            clearInterval(interval);
          }
          counter++;
        }, 100);
        
        if (ws.readyState !== WebSocket.OPEN) {
          new Error("Failed to send message to websocket after 10 attempts");
        }
      }
    }
    return Subject.create(observer, observable);
  }
}