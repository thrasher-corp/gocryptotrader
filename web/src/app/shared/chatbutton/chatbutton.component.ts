import { Component, OnInit } from '@angular/core';
import { ChatService } from './../../services/chat.service';

@Component({
  selector: 'app-chatbutton',
  templateUrl: './chatbutton.component.html',
  styleUrls: ['./chatbutton.component.scss']
})
export class ChatbuttonComponent implements OnInit {

    constructor(private chatService: ChatService) {
		chatService.messages.subscribe(msg => {			
      console.log("Response from websocket: " + JSON.stringify(msg));
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
    data:{"username":"username","password":"5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"},
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
