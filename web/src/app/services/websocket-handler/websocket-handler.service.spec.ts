import { TestBed, inject } from '@angular/core/testing';

import { WebsocketHandlerService } from './websocket-handler.service';

describe('WebsocketHandlerService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [WebsocketHandlerService]
    });
  });

  it('should be created', inject([WebsocketHandlerService], (service: WebsocketHandlerService) => {
    expect(service).toBeTruthy();
  }));
});
