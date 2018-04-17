import { Component, OnInit, OnDestroy } from '@angular/core';
import { WebsocketResponseHandlerService } from './../../services/websocket-response-handler/websocket-response-handler.service';
import { WebSocketMessageType } from './../../shared/classes/websocket';

@Component({
  selector: 'app-all-updates-ticker',
  templateUrl: './all-updates-ticker.component.html',
  styleUrls: ['./all-updates-ticker.component.scss'],
	providers:    [ WebsocketResponseHandlerService ]
})
export class AllEnabledCurrencyTickersComponent implements OnInit {
  allCurrencies: ExchangeCurrency[];
  private ws: WebsocketResponseHandlerService;
  tickerCard: TickerUpdate;
  showTicker:boolean;
  message:string;

  constructor(private websocketHandler: WebsocketResponseHandlerService) {
    this.allCurrencies = <ExchangeCurrency[]>[];
    websocketHandler.messages.subscribe(msg => {
      if (msg.Event === WebSocketMessageType.TickerUpdate) {
        this.showTicker = false;
        var modal = <ExchangeCurrency>{};
        modal.currencyPair = msg.data.CurrencyPair;
        modal.exchangeName = msg.Exchange;
        var ticker = <TickerUpdate>msg.data;
        this.tickerCard = ticker;
        this.tickerCard.Exchange = msg.Exchange;
        
        if (this.tickerCard.Last > 0) {
          this.showTicker = true;
          this.message = this.tickerCard.Exchange + " " + this.tickerCard.CurrencyPair + "  Last: " + this.tickerCard.Last;
        }
      }
    });
   }
  ngOnInit() { 
  }
  
  ngOnDestroy() {
    this.ws.messages.unsubscribe();
  }
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