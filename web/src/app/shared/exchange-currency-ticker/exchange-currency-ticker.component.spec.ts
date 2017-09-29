import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ExchangeCurrencyTickerComponent } from './exchange-currency-ticker.component';

describe('ExchangeCurrencyTickerComponent', () => {
  let component: ExchangeCurrencyTickerComponent;
  let fixture: ComponentFixture<ExchangeCurrencyTickerComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ExchangeCurrencyTickerComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ExchangeCurrencyTickerComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should be created', () => {
    expect(component).toBeTruthy();
  });
});
