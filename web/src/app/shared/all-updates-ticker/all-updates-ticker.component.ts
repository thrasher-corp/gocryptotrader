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
  tickerCard: TickerUpdate = new TickerUpdate();
  showTicker:boolean = true;
  message:string;

  constructor(private websocketHandler: WebsocketResponseHandlerService) {
this.tickerCard.Exchange = "POLONIEX";
this.tickerCard.CurrencyPair = "BTCUSD (Placeholder)";
this.tickerCard.Last = 0;
console.log(window.localStorage["selectedExchange"]);
    this.allCurrencies = <ExchangeCurrency[]>[];
    this.ws = websocketHandler;
    this.ws.messages.subscribe(msg => {
      
      if (msg.event === WebSocketMessageType.TickerUpdate) {

        if(window.localStorage["selectedExchange"] !== undefined &&
        window.localStorage["selectedCurrency"] !== undefined)
        {
          if(msg.exchange == window.localStorage["selectedExchange"]) {
            if(msg.data.CurrencyPair == window.localStorage["selectedCurrency"]) {
              this.updateTicker(msg)
            }
          }
        } else {
          this.updateTicker(msg)
        }
      }
    });
   }

  private updateTicker(msg:any) : void {
    var modal = <ExchangeCurrency>{};
    modal.currencyPair = msg.data.CurrencyPair;
    modal.exchangeName = msg.exchange;
    var ticker = <TickerUpdate>msg.data;
    this.tickerCard = ticker;
    this.tickerCard.Exchange = msg.exchange;
    this.message = this.tickerCard.Exchange + " " + this.tickerCard.CurrencyPair + "  Last: " + this.tickerCard.Last;
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

export class TickerUpdate {
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