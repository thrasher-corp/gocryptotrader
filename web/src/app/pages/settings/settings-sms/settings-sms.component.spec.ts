import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsSmsComponent } from './settings-sms.component';

describe('SettingsSmsComponent', () => {
  let component: SettingsSmsComponent;
  let fixture: ComponentFixture<SettingsSmsComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SettingsSmsComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SettingsSmsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
