import { Component, OnInit } from '@angular/core';

@Component({
  selector: 'app-orders',
  templateUrl: './orders.component.html',
  styleUrls: ['./orders.component.scss']
})
export class OrdersComponent implements OnInit {
  public orders:Order[] = [];
  constructor() { }

  ngOnInit() {
    var item = new Order();
    item.amount = 12;
      item.price = 23;
      item.total = 3;
    item.count = 3;
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


export class Order {
  public count:number;
  public total:number;
  public price:number;
  public amount:number;
}