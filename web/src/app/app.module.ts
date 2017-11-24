import 'zone.js/dist/zone-mix';
import 'reflect-metadata';
import 'polyfills';

import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { HttpModule } from '@angular/http';
import { NgModule, Injectable } from '@angular/core';
import { FormsModule } from '@angular/forms';

import {
  MdButtonModule,
  MdCardModule,
  MdMenuModule,
  MdToolbarModule,
  MdIconModule,
  MdFormFieldModule,
  MdInputModule,
  MdCheckboxModule,
  MdGridListModule,
  MdProgressSpinnerModule,
} from '@angular/material';


import { AppComponent } from './app.component';
import { HomeComponent } from './pages/home/home.component';
import { AboutComponent } from './pages/about/about.component';
import { SettingsComponent } from './pages/settings/settings.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';

import { NavbarComponent } from './shared/navbar/navbar.component';
import { ExchangeCurrencyTickerComponent } from './shared/exchange-currency-ticker/exchange-currency-ticker.component';
import { AllEnabledCurrencyTickersComponent } from './shared/all-enabled-currency-tickers/all-enabled-currency-tickers.component';
//services
import { WebsocketService } from './services/websocket/websocket.service';
import { WebsocketHandlerService } from './services/websocket-handler/websocket-handler.service';
import { ElectronService } from './providers/electron.service';
//Routing
import { AppRoutingModule } from './app-routing.module';

import * as Rx from 'rxjs/Rx';


@NgModule({
  declarations: [
    AppComponent,
    HomeComponent,
    AboutComponent,
    NavbarComponent,
    SettingsComponent,
    DashboardComponent,
    ExchangeCurrencyTickerComponent,
    AllEnabledCurrencyTickersComponent
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
    MdIconModule,
    MdFormFieldModule,
    MdInputModule,
    MdCheckboxModule,
    MdGridListModule,
    MdProgressSpinnerModule,
  ],
  providers: [ElectronService,WebsocketService,WebsocketHandlerService],
  bootstrap: [AppComponent]
})
export class AppModule {

}