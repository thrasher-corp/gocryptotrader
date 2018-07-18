import { Component, OnInit } from '@angular/core';

@Component({
  selector: 'app-sell-form',
  templateUrl: './sell-form.component.html',
  styleUrls: ['./sell-form.component.scss']
})
export class SellFormComponent implements OnInit {
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
