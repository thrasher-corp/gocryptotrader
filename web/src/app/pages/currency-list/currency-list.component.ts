import {  Component,  OnInit,  OnDestroy,  Pipe,  PipeTransform} from '@angular/core';
import {  WebsocketResponseHandlerService} from './../../services/websocket-response-handler/websocket-response-handler.service';
import {  WebSocketMessageType,  WebSocketMessage} from './../../shared/classes/websocket';
import {  Config,  CurrencyPairRedux} from './../../shared/classes/config';
import {  EnabledCurrenciesPipe,  IterateMapPipe} from './../../shared/classes/pipes';

@Component({
  selector: 'app-currency-list',
  templateUrl: './currency-list.component.html',
  styleUrls: ['./currency-list.component.scss'],
  providers: [WebsocketResponseHandlerService]
})
export class CurrencyListComponent implements OnInit {
  public settings: Config = new Config();
  private ws: WebsocketResponseHandlerService;
  public exchangeCurrencies: Map < string, CurrencyPairRedux[] > = new Map < string, CurrencyPairRedux[] > ();

  constructor(private websocketHandler: WebsocketResponseHandlerService) {
    this.ws = websocketHandler;
    this.ws.messages.subscribe(msg => {
      if (msg.event === WebSocketMessageType.GetConfig) {
        this.settings.setConfig(msg.data);
        this.getExchangeCurrencies();
      }
    });
  }
  ngOnInit() {
    this.getSettings();
  }

  ngOnDestroy() {
    this.ws.messages.unsubscribe();
  }

  public getExchangeCurrencies(): void {
    for (var i = 0; i < this.settings.Exchanges.length; i++) {
      if (this.settings.Exchanges[i].Enabled === true) {
        this.exchangeCurrencies.set(this.settings.Exchanges[i].Name, this.settings.Exchanges[i].Pairs)
      }
    }
    this.exchangeCurrencies.forEach((value: CurrencyPairRedux[], key: string) => {});
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
