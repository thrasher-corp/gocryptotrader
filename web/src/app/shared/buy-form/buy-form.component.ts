import { Component, OnInit } from '@angular/core';

@Component({
  selector: 'app-buy-form',
  templateUrl: './buy-form.component.html',
  styleUrls: ['./buy-form.component.scss']
})
export class BuyFormComponent implements OnInit {
  public exchangeName: string;
public currencyName: string;
public chooseCurrencyMessage = 'Please select a currency';
public showErrorMessage: boolean;

  constructor() { }

  ngOnInit() {
    if (window.localStorage['selectedExchange'] !== undefined &&
    window.localStorage['selectedCurrency'] !== undefined) {
      this.exchangeName = window.localStorage['selectedExchange'];
      this.currencyName = window.localStorage['selectedCurrency'];
    } else {
      this.showErrorMessage = true;
    }
  }
}
