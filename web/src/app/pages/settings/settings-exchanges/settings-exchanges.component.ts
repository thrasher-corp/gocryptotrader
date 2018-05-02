import { Component, OnInit, OnDestroy } from '@angular/core';
import { WebsocketResponseHandlerService } from './../../../services/websocket-response-handler/websocket-response-handler.service';
import { WebSocketMessageType, WebSocketMessage } from './../../../shared/classes/websocket';
import { Config, CurrencyPairRedux } from './../../../shared/classes/config';

@Component({
  selector: 'app-settings-exchanges',
  templateUrl: './settings-exchanges.component.html',
  styleUrls: ['./settings-exchanges.component.scss']
})
export class SettingsExchangesComponent implements OnInit {
  public settings: Config = new Config();
  private ws: WebsocketResponseHandlerService;

  constructor(private websocketHandler: WebsocketResponseHandlerService) {
    this.ws = websocketHandler;
   
  }
  ngOnInit() {
    this.ws.shared.subscribe(msg => {
      if (msg.event === WebSocketMessageType.GetConfig) {
        this.settings.setConfig(msg.data);
      } else if (msg.event === WebSocketMessageType.SaveConfig) {
        // check if err is returned, then display some notification
      }
    });
    this.getSettings();
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
