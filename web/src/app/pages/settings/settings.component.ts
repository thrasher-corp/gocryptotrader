import { Component, OnInit, OnDestroy } from '@angular/core';
import { WebsocketResponseHandlerService } from './../../services/websocket-response-handler/websocket-response-handler.service';
import { WebSocketMessageType, WebSocketMessage } from './../../shared/classes/websocket';
import { Config, CurrencyPairRedux } from './../../shared/classes/config';

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss'],
  providers: [WebsocketResponseHandlerService]
})

export class SettingsComponent implements OnInit, OnDestroy {
  public settings: Config = new Config();
  private ws: WebsocketResponseHandlerService;

  constructor(private websocketHandler: WebsocketResponseHandlerService) {
    this.ws = websocketHandler;
    this.ws.messages.subscribe(msg => {
      if (msg.event === WebSocketMessageType.GetConfig) {
        this.settings.setConfig(msg.data);
      } else if (msg.event === WebSocketMessageType.SaveConfig) {
        // check if err is returned, then display some notification
      }
    });
  }
  ngOnInit() {
    this.getSettings();
  }

  ngOnDestroy() {
    this.ws.messages.unsubscribe();
  }

  private getSettings(): void {
    if(this.settings.isConfigCacheValid()) {
      this.settings.setConfig(JSON.parse(window.localStorage['config']))
    } else {
      this.ws.messages.next(WebSocketMessage.GetSettingsMessage());
    }
  }

  private saveSettings(): void {
    
    this.settings.fromReduxToArray()
    var settingsSave = {
      Event: 'SaveConfig',
      data: this.settings,
    }
    this.ws.messages.next(settingsSave);
  }
}

