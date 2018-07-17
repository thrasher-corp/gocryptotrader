import { Component, OnInit } from '@angular/core';
import { SidebarService } from './../../services/sidebar/sidebar.service';

@Component({
  selector: 'app-navbar',
  templateUrl: './navbar.component.html',
  styleUrls: ['./navbar.component.scss']
})
export class NavbarComponent implements OnInit {
  sidebarService: SidebarService;
  constructor(something: SidebarService) {
    this.sidebarService = something;
   }

  ngOnInit() {
  }

}
