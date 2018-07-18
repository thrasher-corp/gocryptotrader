import {
  Component,
  OnInit
} from '@angular/core';
import {
  BuySellComponent
} from './../../shared/buy-sell/buy-sell.component';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss'],
})

export class DashboardComponent implements OnInit {
  public dashboard: any;
  public expanded = false;
  public trades: BuySellComponent[];
  public maxTrades = 3;

  constructor() {
    this.trades = [];
  }

  ngOnInit() {
    this.resetTiles();
  }

  public addTrade() {
    if (this.trades.length >= 0 && this.trades.length < this.maxTrades) {
      this.trades.push(new BuySellComponent());
    }
  }

  public removeTrade(trade: BuySellComponent) {
    this.trades.splice(this.trades.indexOf(trade), 1);
  }

  public expandTile(tile: any) {
    for (let i = 0; i < this.dashboard.tiles.length; i++) {
      if (this.dashboard.tiles[i].title === tile.title) {
        this.dashboard.tiles[i].rows = 2;
        this.dashboard.tiles[i].columns = 3;
        this.expanded = true;
      } else {
        this.dashboard.tiles[i].rows = 0;
        this.dashboard.tiles[i].columns = 0;
      }
    }
  }

  public resetTiles() {
    this.expanded = false;
    this.dashboard = {
      tiles: [{
          id: 1,
          title: 'Trade History:',
          columns: 1,
          rows: 2,
        },
        {
          id: 2,
          title: 'Price History:',
          columns: 2,
          rows: 1,
        },
        {
          id: 3,
          title: 'My Orders:',
          columns: 1,
          rows: 1,
        },
        {
          id: 4,
          title: 'Orders:',
          columns: 1,
          rows: 1,
        },
      ]
    };
  }
}
