import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { BuySellComponent } from './buy-sell.component';

describe('BuySellComponent', () => {
  let component: BuySellComponent;
  let fixture: ComponentFixture<BuySellComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ BuySellComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(BuySellComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
