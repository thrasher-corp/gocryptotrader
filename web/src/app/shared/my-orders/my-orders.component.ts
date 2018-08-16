import { Component, OnInit } from '@angular/core';

export class MyOrders {
  public count: number;
  public total: number;
  public price: number;
  public amount: number;
}

@Component({
  selector: 'app-my-orders',
  templateUrl: './my-orders.component.html',
  styleUrls: ['./my-orders.component.scss']
})
export class MyOrdersComponent implements OnInit {
  public orders: MyOrders[] = [];

  constructor() { }

  ngOnInit() {
    const item = new MyOrders();
    item.amount = 1234;
      item.price = 423;
      item.total = 2;
    item.count = 3;
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
    this.orders.push(item);
  }
}
