
export interface CoinTotal {
    coin: string;
    balance: number;
    percentage: number;
    address: string;
    icon: string;
  }

  export interface Summary {
    BTC: CoinTotal[];
    ETH: CoinTotal[];
    LTC: CoinTotal[];
  }

  export interface Wallet {
    coin_totals: CoinTotal[];
    coins_offline: CoinTotal[];
    offline_summary: Summary;
    coins_online: CoinTotal[];
    online_summary: Summary;
  }
