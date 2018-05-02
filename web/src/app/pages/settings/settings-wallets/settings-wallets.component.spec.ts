import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsWalletsComponent } from './settings-wallets.component';

describe('SettingsWalletsComponent', () => {
  let component: SettingsWalletsComponent;
  let fixture: ComponentFixture<SettingsWalletsComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SettingsWalletsComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SettingsWalletsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
