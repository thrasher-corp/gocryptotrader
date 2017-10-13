import 'zone.js/dist/zone-mix';
import 'reflect-metadata';
import 'polyfills';

import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { HttpModule } from '@angular/http';
import { NgModule, Injectable } from '@angular/core';
import { FormsModule } from '@angular/forms';

import {
  MatButtonModule,
  MatCardModule,
  MatMenuModule,
  MatToolbarModule,
  MatIconModule,
  MatFormFieldModule,
  MatInputModule,
  MatCheckboxModule,
  MatGridListModule,
  MatProgressSpinnerModule,
  MatSidenavModule,
  MatListModule,
} from '@angular/material';


import { AppComponent } from './app.component';
import { HomeComponent } from './pages/home/home.component';
import { AboutComponent } from './pages/about/about.component';
import { SettingsComponent } from './pages/settings/settings.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';
import { WalletComponent } from './pages/wallet/wallet.component';

import { NavbarComponent } from './shared/navbar/navbar.component';
import { SidebarComponent } from './shared/sidebar/sidebar.component';
import { ExchangeCurrencyTickerComponent } from './shared/exchange-currency-ticker/exchange-currency-ticker.component';
import { AllEnabledCurrencyTickersComponent } from './shared/all-enabled-currency-tickers/all-enabled-currency-tickers.component';
//services
import { WebsocketService } from './services/websocket/websocket.service';
import { WebsocketHandlerService } from './services/websocket-handler/websocket-handler.service';
import { SidebarService } from './services/sidebar/sidebar.service';
import { ElectronService } from './providers/electron.service';
//Routing
import { AppRoutingModule } from './app-routing.module';

import { Wallet } from './shared/classes/wallet';


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
    AllEnabledCurrencyTickersComponent,
    SidebarComponent,
    WalletComponent
  ],
  imports: [
    BrowserModule,
    FormsModule,
    HttpModule,
    AppRoutingModule,
    BrowserAnimationsModule,
    MatButtonModule,
    MatMenuModule,
    MatCardModule,
    MatToolbarModule,
    MatIconModule,
    MatFormFieldModule,
    MatInputModule,
    MatCheckboxModule,
    MatGridListModule,
    MatProgressSpinnerModule,
    MatSidenavModule,
    MatListModule,
  ],
  providers: [ElectronService,WebsocketService,WebsocketHandlerService, SidebarService],
  bootstrap: [AppComponent]
})
export class AppModule {

}