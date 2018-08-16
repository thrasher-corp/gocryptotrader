import { Component, OnInit } from '@angular/core';

export class TradeHistoryOrder {
  public price: number;
  public time: Date;
  public amount: number;
}

@Component({
  selector: 'app-trade-history',
  templateUrl: './trade-history.component.html',
  styleUrls: ['./trade-history.component.scss']
})
export class TradeHistoryComponent implements OnInit {
  public orders: TradeHistoryOrder[] = [];
  constructor() { }

  ngOnInit() {
    const item = new TradeHistoryOrder();
      item.amount = 1,
      item.price = 1,
      item.time = new Date();
      this.orders.push(item);
      this.orders.push(item);
      this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
      this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
  }
}
