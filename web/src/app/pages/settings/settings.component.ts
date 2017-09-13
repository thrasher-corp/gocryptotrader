import { Component, OnInit } from '@angular/core';
import { WebsocketHandlerService } from './../../services/websocket-handler/websocket-handler.service';


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

export interface RootObject {
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


@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss']
})


export class SettingsComponent implements OnInit {
  private settings:RootObject = null;
  private ws: WebsocketHandlerService;

  constructor(private websocketHandler: WebsocketHandlerService) {
    this.ws = websocketHandler;
    this.ws.messages.subscribe(msg => {
      if(msg.Event === "GetConfig") {
        console.log("lol");
        this.settings = <RootObject>msg.data;
      } 
      else if (msg.Event === "SaveConfig") {
        //something!
      } 
    });
  }

  ngOnInit() {
    this.getSettings();
  }

  private getSettingsMessage = {
    Event:'GetConfig',
    data:null
  };

  private getSettings():void {
		console.log('new message from client to websocket: ', this.getSettingsMessage);
    this.ws.messages.next(this.getSettingsMessage);
    this.resendMessageIfPageRefreshed();
  } 

  private resendMessageIfPageRefreshed():void {
    setInterval(()=> {
      if(this.settings === null) {
        console.log('Settings hasnt been set. Trying again');
        this.getSettings();
      }
    }, 1000);
  }

  

}


