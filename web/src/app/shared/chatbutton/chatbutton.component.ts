import { Component, OnInit } from '@angular/core';
import { WebsocketHandlerService } from './../../services/websocket-handler/websocket-handler.service';

@Component({
  selector: 'app-chatbutton',
  templateUrl: './chatbutton.component.html',
  styleUrls: ['./chatbutton.component.scss']
})
export class ChatbuttonComponent implements OnInit {

    constructor(private chatService: WebsocketHandlerService) {
      	chatService.messages.subscribe(msg => {
        });
	}

  ngOnInit() {
  }

  private getSettingsMessage = {
    Event:'GetConfig',
    data:null,
  }
  private authenticateMessage = {
    Event:'auth',
    data:{"username":"Username","password":"16f78a7d6317f102bbd95fc9a4f3ff2e3249287690b8bdad6b7810f82b34ace3"},
  }
  
  authenticate():void {
		console.log('new message from client to websocket: ', this.authenticateMessage);
		this.chatService.messages.next(this.authenticateMessage);
	}

  getSettings():void {
		console.log('new message from client to websocket: ', this.getSettingsMessage);
		this.chatService.messages.next(this.getSettingsMessage);
	}

}
