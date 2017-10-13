
export interface CoinTotal {
    coin: string;
    balance: number;
  }
  
  export interface CoinsOffline {
    coin: string;
    balance: number;
    percentage: number;
  }
  
  export interface BTC {
    address: string;
    balance: number;
    percentage: number;
  }
  
  export interface ETH {
    address: string;
    balance: number;
    percentage: number;
  }
  
  export interface LTC {
    address: string;
    balance: number;
    percentage: number;
  }
  
  export interface OfflineSummary {
    BTC: BTC[];
    ETH: ETH[];
    LTC: LTC[];
  }
  
  export interface OnlineSummary {
    BTC: BTC[];
    ETH: ETH[];
    LTC: LTC[];
  }
  
  export interface Wallet {
    coin_totals: CoinTotal[];
    coins_offline: CoinsOffline[];
    offline_summary: OfflineSummary;
    coins_online?: any;
    online_summary: OnlineSummary;
  }