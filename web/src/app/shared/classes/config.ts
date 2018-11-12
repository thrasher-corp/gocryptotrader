import { inherits } from 'util';


export interface ForexProvider {
  name: string;
  enabled: boolean;
  verbose: boolean;
  restPollingDelay: number;
  apiKey: string;
  apiKeyLvl: number;
  primaryProvider: boolean;
}

export interface CurrencyPairFormat {
  uppercase: boolean;
  delimiter: string;
}

export interface CurrencyConfig {
  forexProviders: ForexProvider[];
  cryptocurrencies: string;
  currencyPairFormat: CurrencyPairFormat;
  fiatDisplayCurrency: string;
}

export interface Slack {
  name: string;
  enabled: boolean;
  verbose: boolean;
  targetChannel: string;
  verificationToken: string;
}

export interface Contact {
  name: string;
  number: string;
  enabled: boolean;
}

export interface SmsGlobal {
  name: string;
  enabled: boolean;
  verbose: boolean;
  username: string;
  password: string;
  contacts: Contact[];
}

export interface Smtp {
  name: string;
  enabled: boolean;
  verbose: boolean;
  host: string;
  port: string;
  accountName: string;
  accountPassword: string;
  recipientList: string;
}

export interface Telegram {
  name: string;
  enabled: boolean;
  verbose: boolean;
  verificationToken: string;
}

export interface Communications {
  slack: Slack;
  smsGlobal: SmsGlobal;
  smtp: Smtp;
  telegram: Telegram;
}

export interface Address {
  Address: string;
  CoinType: string;
  Balance: number;
  Description: string;
}



export interface Webserver {
  enabled: boolean;
  adminUsername: string;
  adminPassword: string;
  listenAddress: string;
  websocketConnectionLimit: number;
  websocketMaxAuthFailures: number;
  websocketAllowInsecureOrigin: boolean;
}

export interface ConfigCurrencyPairFormat {
  uppercase: boolean;
  delimiter: string;
}

export interface RequestCurrencyPairFormat {
  uppercase: boolean;
}

export interface BankAccount {
  bankName: string;
  bankAddress: string;
  accountName: string;
  accountNumber: string;
  swiftCode: string;
  iban: string;
  supportedCurrencies: string;
}

export interface Exchange {
  name: string;
  enabled: boolean;
  verbose: boolean;
  websocket: boolean;
  useSandbox: boolean;
  restPollingDelay: number;
  httpTimeout: number;
  httpUserAgent: string;
  authenticatedApiSupport: boolean;
  apiKey: string;
  apiSecret: string;
  apiUrl: string;
  apiUrlSecondary: string;
  availablePairs: string;
  enabledPairs: string;
  baseCurrencies: string;
  assetTypes: string;
  supportsAutoPairUpdates: boolean;
  configCurrencyPairFormat: ConfigCurrencyPairFormat;
  requestCurrencyPairFormat: RequestCurrencyPairFormat;
  bankAccounts: BankAccount[];
  pairs: CurrencyPairRedux[];
}




export class CurrencyPairRedux {
  name: string;
  parsedName: string;
  enabled: boolean;
}

export class Config {
  name: string;
  encryptConfig: number;
  globalHTTPTimeout: number;
  currencyConfig: CurrencyConfig;
  communications: Communications;
  portfolioAddresses: PortfolioAddresses;
  webserver: Webserver;
  exchanges: Exchange[];

    public isConfigCacheValid(): boolean {
        const dateStored = +new Date(window.localStorage['configDate']);
        const dateNow = +new Date();
        const dateDifference = Math.abs(dateNow - dateStored);
        const diffMins = Math.floor((dateDifference / 1000) / 60);

        if (isNaN(new Date(dateStored).getTime()) || diffMins > 15) {
            return false;
        } else {
            return true;
        }
    }

  public setConfig(data: any): void {
    const configData = <Config>data;
    this.communications = configData.communications;
    this.currencyConfig = configData.currencyConfig;
    this.encryptConfig = configData.encryptConfig;
    this.globalHTTPTimeout = configData.globalHTTPTimeout;
    this.name = configData.name;
    this.portfolioAddresses = configData.portfolioAddresses;
    this.exchanges = configData.exchanges;
    this.webserver = configData.webserver;
    if (configData.exchanges.length > 0
      && configData.exchanges[0].pairs
      && configData.exchanges[0].pairs.length > 0) {
      console.log('Successfully retrieved well-formed pairs');
      return;
    }
    this.fromArrayToRedux();
    // Rewrite to cache on parsing to redux array
    this.saveToCache();
  }

    public saveToCache(): void {
      window.localStorage['config'] = JSON.stringify(this);
      window.localStorage['configDate'] = new Date().toString();
    }

    public clearCache(): void {
      window.localStorage['config'] = null;
      window.localStorage['configDate'] = null;
    }

    public fromArrayToRedux(): void {
        for (let i = 0; i < this.exchanges.length; i++) {
          this.exchanges[i].pairs = new Array<CurrencyPairRedux>();
          const avail = this.exchanges[i].availablePairs.split(',');
          const enabled = this.exchanges[i].enabledPairs.split(',');
          for (let j = 0; j < avail.length; j++) {
            const currencyPair = new CurrencyPairRedux();
            currencyPair.name = avail[j];
            currencyPair.parsedName = this.stripCurrencyCharacters(avail[j]);
            if (enabled.indexOf(avail[j]) > 0) {
              currencyPair.enabled = true;
            } else {
              currencyPair.enabled = false;
            }
            this.exchanges[i].pairs.push(currencyPair);
          }
        }

      }

    public parseSettings(): void {

    }

    private stripCurrencyCharacters(name: string): string {
        name = name.replace('_', '');
        name = name.replace('-', '');
        name = name.replace(' ', '');
        name = name.toLocaleUpperCase();
        return name;
      }

    public fromReduxToArray(): void {
        for (let i = 0; i < this.exchanges.length; i++) {
          // Step 1, iterate over the Pairs
          const enabled = this.exchanges[i].enabledPairs.split(',');
          for (let j = 0; j < this.exchanges[i].pairs.length; j++) {
            if (this.exchanges[i].pairs[j].enabled) {
              if (enabled.indexOf(this.exchanges[i].pairs[j].name) === -1) {
                // Step 3 if its not in the enabled list, add it
                enabled.push(this.exchanges[i].pairs[j].name);
              }
            } else {
              if (enabled.indexOf(this.exchanges[i].pairs[j].name) > -1) {
                enabled.splice(enabled.indexOf(this.exchanges[i].pairs[j].name), 1);
              }
            }
          }
          // Step 4 JSONifiy the enabled list and set it to the this.settings.Exchanges[i].EnabledPairs
          this.exchanges[i].enabledPairs = enabled.join();
        }
      }
  }


  export interface CurrencyPairFormat {
    Uppercase: boolean;
    Delimiter: string;
  }

  export interface PortfolioAddresses {
    Addresses?: Wallet[];
  }

  export interface Wallet {
    Address: string;
    CoinType: string;
    Balance: number;
    Description: string;

  }

  export class SMSGlobalContact {
    Name: string;
    Number: string;
    Enabled: boolean;
  }


  export interface Webserver {
    Enabled: boolean;
    AdminUsername: string;
    AdminPassword: string;
    ListenAddress: string;
    WebsocketConnectionLimit: number;
    WebsocketAllowInsecureOrigin: boolean;
  }

  export interface ConfigCurrencyPairFormat {
    Uppercase: boolean;
    Index: string;
    Delimiter: string;
  }

  export interface RequestCurrencyPairFormat {
    Uppercase: boolean;
    Index: string;
    Delimiter: string;
    Separator: string;
  }

  export interface Exchange {
    name: string;
    enabled: boolean;
    verbose: boolean;
    websocket: boolean;
    RESTPollingDelay: number;
    authenticatedAPISupport: boolean;
    APIKey: string;
    APISecret: string;
    availablePairs: string;
    enabledPairs: string;
    baseCurrencies: string;
    assetTypes: string;
    configCurrencyPairFormat: ConfigCurrencyPairFormat;
    requestCurrencyPairFormat: RequestCurrencyPairFormat;
    clientID: string;
    pairs: CurrencyPairRedux[];
}


export class Communcations {
  Slack: SlackCommunication;
  SMSGlobal: SMSGlobalCommunication;
  SMTP: SMTPCommunication;
  Telegram: TelegramCommunication;
}


export class SlackCommunication {
  Name: string;
  Enabled: boolean;
  Verbose: boolean;
  TargetChannel: string;
  VerificationToken: string;
}

export class SMSGlobalCommunication {
  Name: string;
  Enabled: boolean;
  Verbose: boolean;
  Username: string;
  Password: string;
  Contacts: SMSGlobalContact[];
}

export class SMTPCommunication {
  Name: string;
  Enabled: boolean;
  Verbose: boolean;
  Host: string;
  Port: number;
  AccountName: string;
  AccountPassword: string;
  RecipentList: string;
}

export class TelegramCommunication {
  Name: string;
  Enabled: boolean;
  Verbose: boolean;
  VerificationToken: string;
}

export class CurrencyConfig {
  ForexProviders: ForexProviders[];
  Cyptocurrencies: string;
  CurrencyPairFormat: CurrencyPairFormat;
  FiatDisplayCurrency: string;
}

export class ForexProviders {
  Name: string;
  Enabled: boolean;
  Verbose: boolean;
  RESTPollingDelay: number;
  APIKey: string;
  PrimaryProvier: boolean;
}

export class CurrencyPairFormat {
  Uppercase: boolean;
  Delimiter: string;
}




