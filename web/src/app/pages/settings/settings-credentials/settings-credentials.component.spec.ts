import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsCredentialsComponent } from './settings-credentials.component';

describe('SettingsCredentialsComponent', () => {
  let component: SettingsCredentialsComponent;
  let fixture: ComponentFixture<SettingsCredentialsComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SettingsCredentialsComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SettingsCredentialsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
