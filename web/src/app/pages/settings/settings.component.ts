import { Component, OnInit } from '@angular/core';
import { WebsocketHandlerService } from './../../services/websocket-handler/websocket-handler.service';

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss']
})


export class SettingsComponent implements OnInit {
  public settings: Config = null;
  private ws: WebsocketHandlerService;
  private failCount = 0;
  private timer: any;

  private getSettingsMessage = {
    Event: 'GetConfig',
    data: null,
  };

  constructor(private websocketHandler: WebsocketHandlerService) {
    this.ws = websocketHandler;
    this.ws.messages.subscribe(msg => {
      
      if (msg.Event === 'GetConfig') {
        this.settings = <Config>msg.data;
      } else if (msg.Event === 'SaveConfig') {
        // something!
      }
    });
  }
  ngOnInit() {
    this.getSettings();
  }

  private getSettings(): void {
    this.ws.messages.next(this.getSettingsMessage);
    this.resendMessageIfPageRefreshed();
  }


  private saveSettings(): void {
    //Send the message
    var settingsSave = {
      Event: 'SaveConfig',
      data: this.settings,
    }
    this.ws.messages.next(settingsSave);
  }

//there has to be a better way
  private resendMessageIfPageRefreshed(): void {
    if (this.failCount <= 10) {
      setTimeout(() => {
      if (this.settings === null) {
          //console.log(this.failCount);
          //console.log('Settings hasnt been set. Trying again');
          this.failCount++;
          this.getSettings();
        }
      }, 1000);
    } else {
      // something has gone wrong
      console.log('Could not load settings. Check if GocryptoTrader server is running, otherwise open a ticket');
    }
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


