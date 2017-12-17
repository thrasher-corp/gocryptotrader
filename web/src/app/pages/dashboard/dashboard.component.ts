import {Component, OnInit }from '@angular/core'; 
import {BuySellComponent} from './../../shared/buy-sell/buy-sell.component';

@Component( {
selector:'app-dashboard', 
templateUrl:'./dashboard.component.html', 
styleUrls:['./dashboard.component.scss'], 
})

export class DashboardComponent implements OnInit {
public dashboard:any;
public expanded:boolean = false;
public trades:BuySellComponent[];

constructor() {
  this.trades = [];
}

ngOnInit() {
  this.resetTiles();
}

public addTrade() {
  if(this.trades.length >= 0 && this.trades.length <= 2) {
    this.trades.push(new BuySellComponent());
  }
}

public removeTrade(trade:BuySellComponent) {
 this.trades.splice(this.trades.indexOf(trade),1);
}

public expandTile(tile:any) {
  for(var i = 0; i< this.dashboard.tiles.length; i++) {
    if(this.dashboard.tiles[i].title === tile.title ) {
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
 this.dashboard = {tiles:[ {
    title:'Trade History:', 
    subTitle:'Trade History', 
    content:'<app-trade-history></app-trade-history>',
    columns:1,
    rows:2,
    },  {
    title:'Price History:', 
    subTitle:'Price History', 
    content:'<app-price-history></app-price-history>',
    columns:2,
    rows:1,
    },  {
    title:'My Orders:', 
    subTitle:'My Orders', 
    content:'<app-my-orders></app-my-orders>',
    columns:1,
    rows:1,
    },  {
    title:'Orders:', 
    subTitle:'Orders', 
    content:'<app-orders></app-orders>',
    columns:1,
    rows:1,
    }, 
    ]}; 
}
}


