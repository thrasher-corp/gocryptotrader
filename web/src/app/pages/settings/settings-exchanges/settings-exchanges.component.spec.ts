import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsExchangesComponent } from './settings-exchanges.component';

describe('SettingsExchangesComponent', () => {
  let component: SettingsExchangesComponent;
  let fixture: ComponentFixture<SettingsExchangesComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SettingsExchangesComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SettingsExchangesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
