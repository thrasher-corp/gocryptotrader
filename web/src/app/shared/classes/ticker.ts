export interface ExchangeCurrency {
    currencyPair: string;
    exchangeName: string;
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
    Exchange: string;
  }

