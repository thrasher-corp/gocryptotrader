import {   Component,  OnInit,  OnDestroy} from '@angular/core';
import {   WebsocketResponseHandlerService } from './../../services/websocket-response-handler/websocket-response-handler.service';
import {  WebSocketMessageType } from './../../shared/classes/websocket';
import {  ExchangeCurrency, TickerUpdate } from './../../shared/classes/ticker';

@Component({
  selector: 'app-all-updates-ticker',
  templateUrl: './all-updates-ticker.component.html',
  styleUrls: ['./all-updates-ticker.component.scss'],
})
export class AllEnabledCurrencyTickersComponent implements OnInit {
  allCurrencies: ExchangeCurrency[] = < ExchangeCurrency[] > [];
  private ws: WebsocketResponseHandlerService;
  tickerCard: TickerUpdate = new TickerUpdate();

  constructor(private websocketHandler: WebsocketResponseHandlerService) {
    this.tickerCard.Exchange = 'Loading';
    this.tickerCard.CurrencyPair = '...';
    this.tickerCard.Last = -1;
    this.ws = websocketHandler;
    this.ws.shared.subscribe(msg => {
      if (msg.event === WebSocketMessageType.TickerUpdate) {
        if (window.localStorage['selectedExchange'] !== undefined &&
          window.localStorage['selectedCurrency'] !== undefined) {

          this.tickerCard.Exchange = window.localStorage['selectedExchange'];
          this.tickerCard.CurrencyPair = window.localStorage['selectedCurrency'];

          if (msg.exchange === this.tickerCard.Exchange &&
              this.stripCurrencyCharacters(msg.data.CurrencyPair) ===  this.tickerCard.CurrencyPair) {
                this.updateTicker(msg);
            }
        } else {
          this.updateTicker(msg);
        }
      }
    });
  }

  private updateTicker(msg: any): void {
    const ticker = <TickerUpdate> msg.data;
    this.tickerCard = ticker;
    this.tickerCard.Exchange = msg.exchange;
  }

  ngOnInit() {
  }

  private stripCurrencyCharacters(name: string): string {
    name = name.replace('_', '');
    name = name.replace('-', '');
    name = name.replace(' ', '');
    name = name.toLocaleUpperCase();
    return name;
  }
}
