import { Component, OnInit } from '@angular/core';
import { WebsocketHandlerService } from './../../services/websocket-handler/websocket-handler.service';

@Component({
  selector: 'app-all-updates-ticker',
  templateUrl: './all-updates-ticker.component.html',
  styleUrls: ['./all-updates-ticker.component.scss']
})
export class AllEnabledCurrencyTickersComponent implements OnInit {
  private ws: WebsocketHandlerService;
  allCurrencies:ExchangeCurrency[];
  tickerCard: TickerUpdate;
  showTicker:boolean;
  message:string;

  constructor(private websocketHandler: WebsocketHandlerService) {
    this.ws = websocketHandler;
    this.allCurrencies = <ExchangeCurrency[]>[];
    this.ws.messages.subscribe(msg => {
      if (msg.Event === 'ticker_update') {
        this.showTicker = false;
        var modal = <ExchangeCurrency>{};
        modal.currencyPair = msg.data.CurrencyPair;
        modal.exchangeName = msg.Exchange;
        var ticker = <TickerUpdate>msg.data;
        this.tickerCard = ticker;
        this.tickerCard.Exchange = msg.Exchange;
        
        if(this.tickerCard.Last > 0) {
        this.showTicker = true;
          this.message =  this.tickerCard.Exchange + " " + this.tickerCard.CurrencyPair + "  Last: " + this.tickerCard.Last;
        }
      }
    });
   }
  ngOnInit() {  }
}

export interface ExchangeCurrency {
  currencyPair: string;
  exchangeName:string;
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