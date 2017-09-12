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
          if(msg.Event == "GetConfig") {
            this.settings = msg.data;
          }
        });
	}

  ngOnInit() {
  }

  private settings:any;

  private getSettingsMessage = {
    Event:'GetConfig',
    data:null,
  };
 

  getSettings():void {
		console.log('new message from client to websocket: ', this.getSettingsMessage);
		this.chatService.messages.next(this.getSettingsMessage);
	}

}
