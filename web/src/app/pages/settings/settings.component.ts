import { Component, OnInit, OnDestroy, Inject } from '@angular/core';
import { WebsocketResponseHandlerService } from './../../services/websocket-response-handler/websocket-response-handler.service';
import { WebSocketMessageType, WebSocketMessage } from './../../shared/classes/websocket';
import { Config, CurrencyPairRedux, Wallet } from './../../shared/classes/config';
import { MatSnackBar, MatDialog, MatDialogRef, MAT_DIALOG_DATA} from '@angular/material';
import { WalletComponent } from '../wallet/wallet.component';

@Component({
  selector: 'app-dialog-overview-example-dialog',
  template: '<h4>Enabled Currencies</h4><div *ngFor="let currency of data.pairs">'
  + '<mat-checkbox name="{{currency.name}}2" [(ngModel)]="currency.enabled">{{currency.name}}</mat-checkbox>'
  + '</div><button mat-raised-button color="primary" (click)="close()">DONE</button>',
})
export class EnabledCurrenciesDialogueComponent {

  constructor(
    public dialogRef: MatDialogRef<EnabledCurrenciesDialogueComponent>,
    @Inject(MAT_DIALOG_DATA) public data: any) { }

  onNoClick(): void {
    this.dialogRef.close();
  }

  public close(): void {
    this.dialogRef.close();

  }
}

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss'],
})

export class SettingsComponent implements OnInit {
  public settings: Config = new Config();
  private ws: WebsocketResponseHandlerService;
  public ready = false;
  private snackBar: MatSnackBar;
  private dialogue;

  constructor(private websocketHandler: WebsocketResponseHandlerService,
      snackBar: MatSnackBar,
      public dialog: MatDialog) {
    this.ws = websocketHandler;
    this.snackBar = snackBar;
  }

  ngOnInit() {
    this.ws.shared.subscribe(msg => {
      if (msg.event === WebSocketMessageType.GetConfig) {
        this.settings.setConfig(msg.data);
        this.ready = true;
      } else if (msg.event === WebSocketMessageType.SaveConfig) {
        if (msg.error !== null || msg.error.length > 0) {
          this.snackBar.open(msg.error, '', {
            duration: 4000,
          });
        }
        if (msg.error === null || msg.error === '') {
          this.settings.clearCache();
          this.getSettings();
          this.snackBar.open('Success', msg.data, {
            duration: 1000,
          });
        }
      }
    });
    this.getSettings();
  }

  public addWallet(): void {
    this.settings.portfolioAddresses.Addresses.push(<Wallet>{});
  }

  public removeWallet(wallet: any) {
    this.settings.portfolioAddresses.Addresses.splice(this.settings.portfolioAddresses.Addresses.indexOf(wallet), 1);
  }


  public openModal(pairs: any): void {
    const dialogRef = this.dialog.open(EnabledCurrenciesDialogueComponent, {
      width: '20%',
      height: '40%',
      data: { pairs: pairs }
    });
  }

  private getSettings(): void {
    if (this.settings.isConfigCacheValid()) {
      this.settings.setConfig(JSON.parse(window.localStorage['config']));
      this.ready = true;
    } else {
      this.settings.clearCache();
      this.ws.messages.next(WebSocketMessage.GetSettingsMessage());
    }
  }

  private saveSettings(): void {
    this.settings.fromReduxToArray();
    const settingsSave = {
      Event: 'SaveConfig',
      data: this.settings,
    };
    this.ws.messages.next(settingsSave);
  }
}




