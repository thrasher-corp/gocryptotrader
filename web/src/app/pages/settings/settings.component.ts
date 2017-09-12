import { Component, OnInit } from '@angular/core';
import { WebsocketHandlerService } from './../../services/websocket-handler/websocket-handler.service';

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss']
})
export class SettingsComponent implements OnInit {
  private settings:any = null;
  private ws: WebsocketHandlerService;

  constructor(private websocketHandler: WebsocketHandlerService) {
    this.ws = websocketHandler;
    this.ws.messages.subscribe(msg => {
      if(msg.Event == "GetConfig") {
        this.settings = msg.data;
      } else if (msg.Event == "SaveConfig") {
        //something!
      }
    });
  }

  ngOnInit() {
    this.getSettings();
  }

  private getSettingsMessage = {
    Event:'GetConfig',
    data:null
  };

  private getSettings():void {
		console.log('new message from client to websocket: ', this.getSettingsMessage);
    this.ws.messages.next(this.getSettingsMessage);
    this.resendMessageIfPageRefreshed();
  } 

  private resendMessageIfPageRefreshed():void {
    setInterval(()=> {
      if(this.settings === null) {
        console.log('Settings hasnt been set. Trying again');
        this.getSettings();
      }
    }, 1000);
  }

}
