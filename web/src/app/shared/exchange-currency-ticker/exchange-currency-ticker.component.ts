import { Component, OnInit, Input } from '@angular/core';
import { WebsocketHandlerService } from './../../services/websocket-handler/websocket-handler.service';


@Component({
  selector: 'app-exchange-currency-ticker',
  templateUrl: './exchange-currency-ticker.component.html',
  styleUrls: ['./exchange-currency-ticker.component.scss'],
})
export class ExchangeCurrencyTickerComponent implements OnInit {
  @Input('exchange') exchange: string;
  @Input('currency') currency: string;
  ticker: TickerUpdate;
  private ws: WebsocketHandlerService;
  

  constructor(private websocketHandler: WebsocketHandlerService) {
    this.ws = websocketHandler;
    this.ws.messages.subscribe(msg => {
      if (msg.Event === 'ticker_update') {
        if(msg.Exchange !== this.exchange || msg.data.CurrencyPair !== this.currency) {
          console.log('Exg1:' + msg.Exchange + ' exg2:' + this.exchange);
          console.log('Cur1:' + msg.data.CurrencyPair + ' Cur2:' + this.currency);
          return;
        }
        console.log(msg);
        console.log('Data:' + JSON.stringify(msg));
        this.ticker = <TickerUpdate>msg.data;
        
        this.ticker.Exchange = msg.Exchange;
      }
    });
   }

  ngOnInit() {
  }

}

export interface CurrencyPair {
  delimiter: string;
  first_currency: string;
  second_currency: string;
}

export interface TickerUpdate {
  Pair: CurrencyPair;
  CurrencyPair: string;
  Last: number;
  High: number;
  Low: number;
  Bid: number;
  Ask: number;
  Volume: number;
  PriceATH: number;
  Exchange:string;
}
