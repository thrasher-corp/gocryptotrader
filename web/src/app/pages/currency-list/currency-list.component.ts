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
  public selectedCurrency :string;
  public selectedExchange :string;
  public exchangeCurrencies: Map <string,  string[] > = new Map < string, string[] > ();


  constructor(private websocketHandler: WebsocketResponseHandlerService) { 
    this.selectedExchange = window.localStorage["selectedExchange"];
    this.selectedCurrency = window.localStorage["selectedCurrency"];
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

  public selectCurrency(exchange:string,currency:string) {
    window.localStorage["selectedExchange"] = exchange;
    window.localStorage["selectedCurrency"] = currency;
    this.selectedExchange = window.localStorage["selectedExchange"];
    this.selectedCurrency = window.localStorage["selectedCurrency"];
  }

  public getExchangeCurrencies(): void {
    for (var i = 0; i < this.settings.Exchanges.length; i++) {
      if (this.settings.Exchanges[i].Enabled === true) {
        for (var j = 0; j < this.settings.Exchanges[i].Pairs.length; j++) {
          if(this.settings.Exchanges[i].Pairs[j].Enabled) {
          if(this.exchangeCurrencies.has(this.settings.Exchanges[i].Pairs[j].ParsedName)) {
            var array = this.exchangeCurrencies.get(this.settings.Exchanges[i].Pairs[j].ParsedName);
            array.push(this.settings.Exchanges[i].Name);
            this.exchangeCurrencies.set(this.settings.Exchanges[i].Pairs[j].ParsedName, array);
          } else {
            var exchangeArray = new Array<string>();
            exchangeArray.push(this.settings.Exchanges[i].Name);
            this.exchangeCurrencies.set(this.settings.Exchanges[i].Pairs[j].ParsedName, exchangeArray);
          }
        }
        }
      }
    }
  }

  private getSettings(): void {
    if (this.settings.isConfigCacheValid()) {
      this.settings.setConfig(JSON.parse(window.localStorage['config']))
      this.getExchangeCurrencies();
    } else {
      this.ws.messages.next(WebSocketMessage.GetSettingsMessage());
    }
  }
}
