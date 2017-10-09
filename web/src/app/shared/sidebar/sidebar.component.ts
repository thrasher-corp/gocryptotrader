import { Component, OnInit, ViewChild } from '@angular/core';
import { MdSidenav } from '@angular/material';
import { SidebarService } from './../../services/sidebar/sidebar.service';
import { Router, NavigationEnd } from '@angular/router';

@Component({
  selector: 'app-sidebar',
  templateUrl: './sidebar.component.html',
  styleUrls: ['./sidebar.component.scss']
})
export class SidebarComponent implements OnInit {
  @ViewChild('sidenav') public sidenav: MdSidenav;
  sidebarService: SidebarService
public currentUrl:string;

  constructor(something: SidebarService, private router:Router) {
    this.sidebarService = something;

    router.events.subscribe(event => {
      
            if (event instanceof NavigationEnd ) {
              console.log("current url",event.url); // event.url has current url
              this.currentUrl = event.url;
            }
          });
   }

  ngOnInit() {
    this.sidebarService.setSidenav(this.sidenav);
    }
  }
