import 'zone.js/dist/zone-mix';
import 'reflect-metadata';
import 'polyfills';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MdButtonModule, MdCardModule, MdMenuModule, MdToolbarModule, MdIconModule, MdFormFieldModule } from '@angular/material';

import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { HttpModule } from '@angular/http';
import {Injectable} from '@angular/core';



import { AppComponent } from './app.component';
import { HomeComponent } from './pages/home/home.component';
import { AboutComponent } from './pages/about/about.component';
import { NavbarComponent } from './shared/navbar/navbar.component';

import { AppRoutingModule } from './app-routing.module';

import { WebsocketService } from './services/websocket/websocket.service';
import { WebsocketHandlerService } from './services/websocket-handler/websocket-handler.service';
import { ElectronService } from './providers/electron.service';

import * as Rx from 'rxjs/Rx';
import { ChatbuttonComponent } from './shared/chatbutton/chatbutton.component';
import { SettingsComponent } from './pages/settings/settings.component';

@NgModule({
  declarations: [
    AppComponent,
    HomeComponent,
    AboutComponent,
    NavbarComponent,
    ChatbuttonComponent,
    SettingsComponent
  ],
  imports: [
    BrowserModule,
    FormsModule,
    HttpModule,
    AppRoutingModule,
    BrowserAnimationsModule,
    MdButtonModule,
    MdMenuModule,
    MdCardModule,
    MdToolbarModule,
    MdIconModule
  ],
  providers: [ElectronService,WebsocketService,WebsocketHandlerService],
  bootstrap: [AppComponent]
})
export class AppModule {

}