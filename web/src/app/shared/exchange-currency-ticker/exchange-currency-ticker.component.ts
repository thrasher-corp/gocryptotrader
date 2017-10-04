import { Component, OnInit, Input } from '@angular/core';
@Component({
  selector: 'app-exchange-currency-ticker',
  templateUrl: './exchange-currency-ticker.component.html',
  styleUrls: ['./exchange-currency-ticker.component.scss'],
})
export class ExchangeCurrencyTickerComponent implements OnInit {
  @Input('ticker') ticker: TickerUpdate;

  constructor() {}
  ngOnInit() { }

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

