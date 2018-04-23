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
  private selectedOptions: any[]

  private getSettingsMessage = {
    Event: 'GetConfig',
    data: null,
  };

  constructor(private websocketHandler: WebsocketResponseHandlerService) {
    this.ws = websocketHandler;
    this.ws.messages.subscribe(msg => {
      if (msg.event === WebSocketMessageType.GetConfig) {
        this.settings = <Config>msg.data;
        this.fromArrayToRedux();
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
    this.fromReduxToArray()
    var settingsSave = {
      Event: 'SaveConfig',
      data: this.settings,
    }
    this.ws.messages.next(settingsSave);
  }

  private fromArrayToRedux() {
    for (var i = 0; i < this.settings.Exchanges.length; i++) {
      this.settings.Exchanges[i].Pairs = new Array<CurrencyPairRedux>();
      var avail = this.settings.Exchanges[i].AvailablePairs.split(',');
      var enabled = this.settings.Exchanges[i].EnabledPairs.split(',');
      for (var j = 0; j < avail.length; j++) {
        var currencyPair = new CurrencyPairRedux();
        currencyPair.Name = avail[j]
        if (enabled.indexOf(avail[j]) > 0) {
          currencyPair.Enabled = true;
        } else {
          currencyPair.Enabled = false;
        }
        this.settings.Exchanges[i].Pairs.push(currencyPair);
      }
    }
  }


  private fromReduxToArray() {
    for (var i = 0; i < this.settings.Exchanges.length; i++) {
      // Step 1, iterate over the Pairs
      var enabled = this.settings.Exchanges[i].EnabledPairs.split(',');
      console.log('BEFORE: ' + this.settings.Exchanges[i].EnabledPairs)
      for (var j = 0; j < this.settings.Exchanges[i].Pairs.length; j++) {
        if (this.settings.Exchanges[i].Pairs[j].Enabled) {
          if (enabled.indexOf(this.settings.Exchanges[i].Pairs[j].Name) == -1) {
            // Step 3 if its not in the enabled list, add it
            console.log(this.settings.Exchanges[i].Pairs[j].Name + " from " + this.settings.Exchanges[i].Name + " is not in the enabled list and being added")
            enabled.push(this.settings.Exchanges[i].Pairs[j].Name);
          } else {
            console.log(this.settings.Exchanges[i].Pairs[j].Name + " from " + this.settings.Exchanges[i].Name + " is in the enabled list and doing nothing")

          }
        } else {
          if (enabled.indexOf(this.settings.Exchanges[i].Pairs[j].Name) > -1) {
            console.log(this.settings.Exchanges[i].Pairs[j].Name + " from " + this.settings.Exchanges[i].Name + " is in the enabled list and being removed")
            enabled.splice(enabled.indexOf(this.settings.Exchanges[i].Pairs[j].Name), 1);
          } else {
            console.log(this.settings.Exchanges[i].Pairs[j].Name + " from " + this.settings.Exchanges[i].Name + " is not in the enabled list and doing nothing")
          }
        }
      }
      
      //Step 4 JSONifiy the enabled list and set it to the this.settings.Exchanges[i].EnabledPairs
      this.settings.Exchanges[i].EnabledPairs = enabled.join();
      console.log('AFTER: ' + this.settings.Exchanges[i].EnabledPairs)
    }
    
  }
}

export class CurrencyPairRedux {
  Name: string;
  Enabled: boolean;
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
  Pairs: CurrencyPairRedux[];
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


