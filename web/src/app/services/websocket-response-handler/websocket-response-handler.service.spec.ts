import { TestBed, inject } from '@angular/core/testing';

import { WebsocketResponseHandlerService } from './websocket-response-handler.service';

describe('WebsocketHandlerService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [WebsocketResponseHandlerService]
    });
  });

  it('should be created', inject([WebsocketResponseHandlerService], (service: WebsocketResponseHandlerService) => {
    expect(service).toBeTruthy();
  }));
});
