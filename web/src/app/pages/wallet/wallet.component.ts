import { Component, OnInit, OnDestroy } from '@angular/core';
import { WebsocketResponseHandlerService } from './../../services/websocket-response-handler/websocket-response-handler.service';
import { Wallet, CoinTotal } from './../../shared/classes/wallet';
import { Sort } from '@angular/material';
import { WebSocketMessageType, WebSocketMessage } from './../../shared/classes/websocket';

@Component({
  selector: 'app-wallet',
  templateUrl: './wallet.component.html',
  styleUrls: ['./wallet.component.scss'],
})

export class WalletComponent implements OnInit {
  private ws: WebsocketResponseHandlerService;
  private failCount = 0;
  private timer: any;
  public wallet: Wallet;
  displayedColumns = ['coin', 'balance'];

  private getWalletMessage = {
    event: 'GetPortfolio',
    data: null,
  };

  constructor(private websocketHandler: WebsocketResponseHandlerService) {
    this.wallet = null;
    this.ws = websocketHandler;
    this.ws.shared.subscribe(msg => {
      if (msg.event === WebSocketMessageType.GetPortfolio) {
        this.wallet = <Wallet>msg.data;
        console.log('wallet: ' + msg.data);
        console.log('message: ' + JSON.stringify(msg));
        console.log('data: ' +  this.wallet);

        if (this.wallet != null && this.wallet.coin_totals != null) {
          this.attachIcon(this.wallet.coin_totals);
          this.attachIcon(this.wallet.coins_offline);
          this.attachIcon(this.wallet.coins_online);

          this.attachIcon(this.wallet.offline_summary.BTC);
          this.attachIcon(this.wallet.offline_summary.ETH);
          this.attachIcon(this.wallet.offline_summary.LTC);

          this.attachIcon(this.wallet.online_summary.BTC);
          this.attachIcon(this.wallet.online_summary.ETH);
          this.attachIcon(this.wallet.online_summary.LTC);
        }
      }
    });
  }

  ngOnInit() {
    this.setWallet();
  }

  private setWallet(): void {
    this.ws.messages.next(this.getWalletMessage);
  }

  public coinIcon(coin: string): string {
    switch (coin) {
      case 'BTC': return 'cc BTC';
      case 'LTC': return 'cc LTC';
      case 'ETH': return 'cc ETH';
    }
  }

  public attachIcon(items: CoinTotal[]): void {
    if (items) {
      for (let i = 0; i < items.length; i++) {
        items[i].icon = this.coinIcon(items[i].coin);
      }
    }
}


}
