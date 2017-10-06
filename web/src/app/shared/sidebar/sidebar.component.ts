import { Component, OnInit, ViewChild } from '@angular/core';
import { MdSidenav } from '@angular/material';
import { SidebarService } from './../../services/sidebar/sidebar.service';

@Component({
  selector: 'app-sidebar',
  templateUrl: './sidebar.component.html',
  styleUrls: ['./sidebar.component.scss']
})
export class SidebarComponent implements OnInit {
  @ViewChild('sidenav') public sidenav: MdSidenav;
  sidebarService: SidebarService
  constructor() { }

  ngOnInit() {
    this.sidebarService.setSidenav(this.sidenav);
    }
  }
