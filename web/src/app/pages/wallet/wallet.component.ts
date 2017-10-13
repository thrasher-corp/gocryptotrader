import { Component, OnInit } from '@angular/core';
import { WebsocketHandlerService } from './../../services/websocket-handler/websocket-handler.service';
import { Wallet } from './../../shared/classes/wallet';


@Component({
  selector: 'app-wallet',
  templateUrl: './wallet.component.html',
  styleUrls: ['./wallet.component.scss']
})
export class WalletComponent implements OnInit {
  private ws: WebsocketHandlerService;
  private failCount = 0;
  private timer: any;
  public wallet: Wallet;

  private getWalletMessage = {
    Event: 'GetPortfolio',
    data: null,
  };

  constructor(private websocketHandler: WebsocketHandlerService) {
    this.ws = websocketHandler;
    this.ws.messages.subscribe(msg => {
      if (msg.Event === 'GetPortfolio') {
        console.log(JSON.stringify(msg.data));
        this.wallet = <Wallet>msg.data;
      }
    });
  }
  ngOnInit() {
    this.setWallet();
  }

//there has to be a better way
  private resendMessageIfPageRefreshed(): void {
    if (this.failCount <= 10) {
      setTimeout(() => {
      if (this.wallet === null || this.wallet === undefined) {
          this.failCount++;
          this.setWallet();
        }
      }, 1000);
    } else {
      console.log('Could not load wallet. Check if GocryptoTrader server is running, otherwise open a ticket');
    }
  }

  private setWallet():void {
    this.ws.messages.next(this.getWalletMessage);
    this.resendMessageIfPageRefreshed();
  }
}


