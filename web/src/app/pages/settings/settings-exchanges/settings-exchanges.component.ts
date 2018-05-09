import { Component, OnInit, OnDestroy } from '@angular/core';
import { WebsocketResponseHandlerService } from './../../../services/websocket-response-handler/websocket-response-handler.service';
import { WebSocketMessageType, WebSocketMessage } from './../../../shared/classes/websocket';
import { Config, CurrencyPairRedux } from './../../../shared/classes/config';
import {MatSnackBar} from '@angular/material';

@Component({
  selector: 'app-settings-exchanges',
  templateUrl: './settings-exchanges.component.html',
  styleUrls: ['./settings-exchanges.component.scss']
})
export class SettingsExchangesComponent implements OnInit {
  public settings: Config = new Config();
  private ws: WebsocketResponseHandlerService;
  public ready: boolean = false;
  private snackBar : MatSnackBar;

  constructor(private websocketHandler: WebsocketResponseHandlerService, snackBar: MatSnackBar) {
    this.ws = websocketHandler;
    this.snackBar = snackBar;
  }
  ngOnInit() {
    this.ws.shared.subscribe(msg => {
      if (msg.event === WebSocketMessageType.GetConfig) {
        this.settings.setConfig(msg.data);
        this.ready = true;
      } else if (msg.event === WebSocketMessageType.SaveConfig) {
        if(msg.error !== null || msg.error.length > 0) {
          this.snackBar.open(msg.error, '', {
            duration: 4000,
          });
        } 
        if(msg.error === null || msg.error === '') {
          this.snackBar.open('Success', msg.data, {
            duration: 1000,
          });
        } 
      }
    });
    this.getSettings();
  }


  private getSettings(): void {
    if(this.settings.isConfigCacheValid()) {
      this.settings.setConfig(JSON.parse(window.localStorage['config']))
      this.ready = true;      
    } else {
      this.settings.clearCache();
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
    this.settings.saveToCache();
  }
}
