import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ChatbuttonComponent } from './chatbutton.component';

describe('ChatbuttonComponent', () => {
  let component: ChatbuttonComponent;
  let fixture: ComponentFixture<ChatbuttonComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ChatbuttonComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ChatbuttonComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should be created', () => {
    expect(component).toBeTruthy();
  });
});
