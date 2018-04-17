import { Component, OnInit, OnDestroy } from '@angular/core';
import { WebsocketResponseHandlerService } from './../../services/websocket-response-handler/websocket-response-handler.service';
import { WebSocketMessageType } from './../../shared/classes/websocket';

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss'],
  providers: [WebsocketResponseHandlerService]
})

export class SettingsComponent implements OnInit, OnDestroy {
  public settings: Config = null;
  private ws: WebsocketResponseHandlerService;
  private failCount = 0;
  private timer: any;

  private getSettingsMessage = {
    Event: 'GetConfig',
    data: null,
  };

  constructor(private websocketHandler: WebsocketResponseHandlerService) {
    this.ws = websocketHandler;
    this.ws.messages.subscribe(msg => {
      if (msg.event === WebSocketMessageType.GetConfig) {
        this.settings = <Config>msg.data;
      } else if (msg.event === WebSocketMessageType.SaveConfig) {
        // something!
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
    this.ws.messages.next(this.getSettingsMessage);
  }


  private saveSettings(): void {
    var settingsSave = {
      Event: 'SaveConfig',
      data: this.settings,
    }
    this.ws.messages.next(settingsSave);
  }
}


export interface CurrencyPairFormat {
  Uppercase: boolean;
  Delimiter: string;
}

export interface PortfolioAddresses {
  Addresses?: any;
}

export interface Contact {
  Name: string;
  Number: string;
  Enabled: boolean;
}

export interface SMSGlobal {
  Enabled: boolean;
  Username: string;
  Password: string;
  Contacts: Contact[];
}

export interface Webserver {
  Enabled: boolean;
  AdminUsername: string;
  AdminPassword: string;
  ListenAddress: string;
  WebsocketConnectionLimit: number;
  WebsocketAllowInsecureOrigin: boolean;
}

export interface ConfigCurrencyPairFormat {
  Uppercase: boolean;
  Index: string;
  Delimiter: string;
}

export interface RequestCurrencyPairFormat {
  Uppercase: boolean;
  Index: string;
  Delimiter: string;
  Separator: string;
}

export interface Exchange {
  Name: string;
  Enabled: boolean;
  Verbose: boolean;
  Websocket: boolean;
  RESTPollingDelay: number;
  AuthenticatedAPISupport: boolean;
  APIKey: string;
  APISecret: string;
  AvailablePairs: string;
  EnabledPairs: string;
  BaseCurrencies: string;
  AssetTypes: string;
  ConfigCurrencyPairFormat: ConfigCurrencyPairFormat;
  RequestCurrencyPairFormat: RequestCurrencyPairFormat;
  ClientID: string;
}

export interface Config {
  Name: string;
  EncryptConfig?: number;
  Cryptocurrencies: string;
  CurrencyExchangeProvider: string;
  CurrencyPairFormat: CurrencyPairFormat;
  PortfolioAddresses: PortfolioAddresses;
  SMSGlobal: SMSGlobal;
  Webserver: Webserver;
  Exchanges: Exchange[];
}


