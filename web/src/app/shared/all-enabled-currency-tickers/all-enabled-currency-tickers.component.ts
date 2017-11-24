import { Component, OnInit } from '@angular/core';
import { WebsocketHandlerService } from './../../services/websocket-handler/websocket-handler.service';

@Component({
  selector: 'app-all-enabled-currency-tickers',
  templateUrl: './all-enabled-currency-tickers.component.html',
  styleUrls: ['./all-enabled-currency-tickers.component.scss']
})
export class AllEnabledCurrencyTickersComponent implements OnInit {
  private ws: WebsocketHandlerService;
  allCurrencies:ExchangeCurrency[];
  tickerCards: TickerUpdate[];

  constructor(private websocketHandler: WebsocketHandlerService) {
    this.ws = websocketHandler;
    this.allCurrencies = <ExchangeCurrency[]>[];
    this.tickerCards = <TickerUpdate[]>[];
    this.ws.messages.subscribe(msg => {
      if (msg.Event === 'ticker_update') {
        var modal = <ExchangeCurrency>{};
        modal.currencyPair = msg.data.CurrencyPair;
        modal.exchangeName = msg.Exchange;
        var found = false;
        
        for(var i = 0; i< this.allCurrencies.length; i++) {
          if(this.allCurrencies[i].currencyPair === msg.data.CurrencyPair &&
            this.allCurrencies[i].exchangeName === msg.Exchange) {
              found = true;
            }
        }
        if(!found) {
          //time to add
          var ticker = <TickerUpdate>msg.data;
          ticker.Exchange = msg.Exchange;
          this.tickerCards.push(ticker);
          this.allCurrencies.push(modal);
        } else {
          //time to replace
          for(var j = 0; j< this.tickerCards.length; j++) {
            if(this.tickerCards[j].Exchange === msg.Exchange 
              && this.tickerCards[j].CurrencyPair === msg.data.CurrencyPair) {
              var ticker = <TickerUpdate>msg.data;
                this.tickerCards[j] = ticker;
                this.tickerCards[j].Exchange = msg.Exchange;
              return;
              }
          }
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