import {Component, OnInit, OnDestroy }from '@angular/core'; 
import {WebsocketResponseHandlerService }from './../../services/websocket-response-handler/websocket-response-handler.service'; 
import {WebSocketMessageType }from './../../shared/classes/websocket'; 

@Component( {
  selector:'app-all-updates-ticker', 
  templateUrl:'./all-updates-ticker.component.html', 
  styleUrls:['./all-updates-ticker.component.scss'], 
})
export class AllEnabledCurrencyTickersComponent implements OnInit {
  allCurrencies:ExchangeCurrency[] =  < ExchangeCurrency[] > []; ; 
  private ws:WebsocketResponseHandlerService; 
  tickerCard:TickerUpdate = new TickerUpdate(); 
  showTicker:boolean = true; 
  message:string; 

  constructor(private websocketHandler: WebsocketResponseHandlerService) {
    if (window.localStorage["selectedExchange"] == undefined && 
        window.localStorage["selectedCurrency"] == undefined) {
          this.tickerCard.Exchange = "Loading"; 
          this.tickerCard.CurrencyPair = "..."; 
          this.tickerCard.Last = -1; 
    }else {
          this.tickerCard.Exchange = window.localStorage["selectedExchange"]; 
          this.tickerCard.CurrencyPair = window.localStorage["selectedCurrency"]; 
          this.tickerCard.Last = 0; 
    }
    this.ws = websocketHandler; 
    this.ws.shared.subscribe(msg =>  {
      if (msg.event === WebSocketMessageType.TickerUpdate) {
        console.log(msg.data.CurrencyPair); 
        if (window.localStorage["selectedExchange"] !== undefined && 
        window.localStorage["selectedCurrency"] !== undefined) {
          console.log('found selected currency');
          
            console.log(msg.exchange); 
            if (msg.exchange == window.localStorage["selectedExchange"]) {
            console.log('found exchange'); 
            if (this.stripCurrencyCharacters(msg.data.CurrencyPair) == window.localStorage["selectedCurrency"]) {
            console.log('found currency'); 
              
              this.updateTicker(msg)
            }
          }
        }else {
          console.log('no selected currency');
          this.updateTicker(msg)
        }
      }
    }); 
   }

  private updateTicker(msg:any):void {
    var modal =  < ExchangeCurrency >  {}; 
    modal.currencyPair = msg.data.CurrencyPair; 
    modal.exchangeName = msg.exchange; 
    var ticker =  < TickerUpdate > msg.data; 
    this.tickerCard = ticker; 
    this.tickerCard.Exchange = msg.exchange; 
    this.message = this.tickerCard.Exchange + " " + this.tickerCard.CurrencyPair + "  Last: " + this.tickerCard.Last; 
  }

  ngOnInit() {

  }

  private stripCurrencyCharacters(name:string):string {
    name = name.replace('_', ''); 
    name = name.replace('-', ''); 
    name = name.replace(' ', ''); 
    name = name.toLocaleUpperCase(); 
    return name; 
  }
}

export interface ExchangeCurrency {
  currencyPair:string; 
  exchangeName:string; 
}

export interface CurrencyPair {
  delimiter:string; 
  first_currency:string; 
  second_currency:string; 
}

export class TickerUpdate {
  Pair:CurrencyPair; 
  CurrencyPair:string; 
  Last:number; 
  High:number; 
  Low:number; 
  Bid:number; 
  Ask:number; 
  Volume:number; 
  PriceATH:number; 
  Exchange:string; 
}