import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { PriceHistoryComponent } from './price-history.component';

describe('PriceHistoryComponent', () => {
  let component: PriceHistoryComponent;
  let fixture: ComponentFixture<PriceHistoryComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ PriceHistoryComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(PriceHistoryComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
