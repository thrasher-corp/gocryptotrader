import { Component, OnInit, ViewChild } from '@angular/core';
import { ElectronService } from './providers/electron.service';
import { MatSidenav } from '@angular/material';
import { SidebarService } from './services/sidebar/sidebar.service';
import { Router, NavigationEnd } from '@angular/router';
import {WebsocketResponseHandlerService } from './services/websocket-response-handler/websocket-response-handler.service';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
})
export class AppComponent implements OnInit {
  sidebarService: SidebarService;
  public currentUrl: string;
  @ViewChild('sidenav') public sidenav: MatSidenav;
  private ws: WebsocketResponseHandlerService;
  public isConnected = false;

  constructor(public electronService: ElectronService,
    sidebarService: SidebarService,
    private router: Router,
    private websocketHandler: WebsocketResponseHandlerService) {

    if (electronService.isElectron()) {
      console.log('Mode electron');
      // Check if electron is correctly injected (see externals in webpack.config.js)
      console.log('c', electronService.ipcRenderer);
      // Check if nodeJs childProcess is correctly injected (see externals in webpack.config.js)
      console.log('c', electronService.childProcess);
    } else {
      console.log('Mode web');
    }

    this.isConnected = this.websocketHandler.isConnected;
    this.sidebarService = sidebarService;
    router.events.subscribe(event => {

      if (event instanceof NavigationEnd) {
        this.isConnected = this.websocketHandler.isConnected;
        console.log('current url', event.url); // event.url has current url
        this.currentUrl = event.url;
      }
    });
    const interval = setInterval(() => {
      this.isConnected = this.websocketHandler.isConnected;
      }, 2000);

  }

  ngOnInit() {
    this.sidebarService.setSidenav(this.sidenav);
    // This will be replaced with a log in prompt which will then add the credentials to session storage
    window.sessionStorage['username'] = 'admin';
    window.sessionStorage['password'] = 'e7cf3ef4f17c3999a94f2c6f612e8a888e5b1026878e4e19398b23bd38ec221a';
  }
}
