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

  private message = {
		author: 'tutorialedge',
    message: 'this is a test message',
    Event:'auth',
    username:'user',
    password:'password'
	}

  sendMsg():void {
		console.log('new message from client to websocket: ', this.message);
		this.chatService.messages.next(this.message);
		this.message.message = '';
	}

}
