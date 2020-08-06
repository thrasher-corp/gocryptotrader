
import {share, map} from 'rxjs/operators';
import { NgModule, Optional, SkipSelf } from '@angular/core';
import { Observable, Subject } from 'rxjs';
import { WebsocketService } from './../../services/websocket/websocket.service';
import { WebSocketMessage } from './../../shared/classes/websocket';

const WEBSOCKET_URL = 'ws://localhost:9051/ws';

@NgModule({
  })
export class WebsocketResponseHandlerService {
    public messages: Subject<any>;
    public shared: Observable<WebSocketMessage>;
    public isConnected = false;
    private ws: WebsocketService;

    constructor(@Optional() @SkipSelf() parentModule: WebsocketResponseHandlerService, wsService: WebsocketService) {
        this.ws = wsService;
        this.messages = <Subject<WebSocketMessage>>this.ws
            .connect(WEBSOCKET_URL).pipe(

            map((response: MessageEvent): WebSocketMessage => {
                const interval = setInterval(() => {
                    this.isConnected = this.ws.isConnected;
                }, 2000);
                const websocketResponseMessage = JSON.parse(response.data);
                const websocketResponseData = websocketResponseMessage.Data === undefined
                    ? websocketResponseMessage.data
                    : websocketResponseMessage.Data;
                const websocketResponseEvent = websocketResponseMessage.Event === undefined
                    ? websocketResponseMessage.event
                    : websocketResponseMessage.Event;
                const responseMessage = new WebSocketMessage();

                responseMessage.event = websocketResponseEvent;
                responseMessage.data = websocketResponseData;
                responseMessage.exchange = websocketResponseMessage.exchange;
                responseMessage.assetType = websocketResponseMessage.assetType;
                responseMessage.error = websocketResponseMessage.error;

                return responseMessage;
            }));
            this.isConnected = this.ws.isConnected;

        this.shared = this.messages.pipe(share()); // multicast
    }
}
