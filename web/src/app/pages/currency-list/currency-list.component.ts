import {  Component,  OnInit,  OnDestroy,  Pipe,  PipeTransform} from '@angular/core';
import {  WebsocketResponseHandlerService} from './../../services/websocket-response-handler/websocket-response-handler.service';
import {  WebSocketMessageType,  WebSocketMessage} from './../../shared/classes/websocket';
import {  Config,  CurrencyPairRedux} from './../../shared/classes/config';
import {  EnabledCurrenciesPipe,  IterateMapPipe} from './../../shared/classes/pipes';

@Component({
  selector: 'app-currency-list',
  templateUrl: './currency-list.component.html',
  styleUrls: ['./currency-list.component.scss'],
})
export class CurrencyListComponent implements OnInit {
  public settings: Config = new Config();
  private ws: WebsocketResponseHandlerService;
  public selectedCurrency: string;
  public selectedExchange: string;
  public exchangeCurrencies: Map <string,  string[] > = new Map < string, string[] > ();


  constructor(private websocketHandler: WebsocketResponseHandlerService) {
    this.selectedExchange = window.localStorage['selectedExchange'];
    this.selectedCurrency = window.localStorage['selectedCurrency'];
    this.ws = websocketHandler;
    this.ws.shared.subscribe(msg => {
      if (msg.event === WebSocketMessageType.GetConfig) {
        this.settings.setConfig(msg.data);
        this.getExchangeCurrencies();
      }
    });
  }

  ngOnInit() {
    this.getSettings();
  }

  public selectCurrency(exchange: string, currency: string) {
    window.localStorage['selectedExchange'] = exchange;
    window.localStorage['selectedCurrency'] = currency;
    this.selectedExchange = window.localStorage['selectedExchange'];
    this.selectedCurrency = window.localStorage['selectedCurrency'];
  }

  public getExchangeCurrencies(): void {
    for (let i = 0; i < this.settings.exchanges.length; i++) {
      if (this.settings.exchanges[i].enabled === true) {
        for (let j = 0; j < this.settings.exchanges[i].pairs.length; j++) {
          if (this.settings.exchanges[i].pairs[j].enabled) {
          if (this.exchangeCurrencies.has(this.settings.exchanges[i].pairs[j].parsedName)) {
            const array = this.exchangeCurrencies.get(this.settings.exchanges[i].pairs[j].parsedName);
            array.push(this.settings.exchanges[i].name);
            this.exchangeCurrencies.set(this.settings.exchanges[i].pairs[j].parsedName, array);
          } else {
            const exchangeArray = new Array<string>();
            exchangeArray.push(this.settings.exchanges[i].name);
            this.exchangeCurrencies.set(this.settings.exchanges[i].pairs[j].parsedName, exchangeArray);
          }
        }
        }
      }
    }
  }

  private getSettings(): void {
    if (this.settings.isConfigCacheValid()) {
      this.settings.setConfig(JSON.parse(window.localStorage['config']));
      this.getExchangeCurrencies();
    } else {
      this.ws.messages.next(WebSocketMessage.GetSettingsMessage());
    }
  }
}
