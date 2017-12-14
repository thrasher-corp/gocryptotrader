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
  MatSortModule,
  MatExpansionModule,
  MatLineModule,
  MatTooltipModule,
  MatTabsModule,
} from '@angular/material';


import { AppComponent } from './app.component';
import { HomeComponent } from './pages/home/home.component';
import { AboutComponent } from './pages/about/about.component';
import { SettingsComponent } from './pages/settings/settings.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';
import { WalletComponent } from './pages/wallet/wallet.component';
import { DonateComponent } from './pages/donate/donate.component';

//Shared
import { NavbarComponent } from './shared/navbar/navbar.component';
import { ExchangeCurrencyTickerComponent } from './shared/exchange-currency-ticker/exchange-currency-ticker.component';
import { AllEnabledCurrencyTickersComponent } from './shared/all-enabled-currency-tickers/all-enabled-currency-tickers.component';
import { ThemePickerComponent } from './shared/theme-picker/theme-picker';
//services
import { WebsocketService } from './services/websocket/websocket.service';
import { WebsocketHandlerService } from './services/websocket-handler/websocket-handler.service';
import { SidebarService } from './services/sidebar/sidebar.service';
import { ElectronService } from './providers/electron.service';
import { StyleManagerService } from './services/style-manager/style-manager.service';
import { ThemeStorageService } from './services/theme-storage/theme-storage.service';

//Routing
import { AppRoutingModule } from './app-routing.module';

import { Wallet } from './shared/classes/wallet';


import * as Rx from 'rxjs/Rx';
import { TradeHistoryComponent } from './shared/trade-history/trade-history.component';
import { PriceHistoryComponent } from './shared/price-history/price-history.component';
import { MyOrdersComponent } from './shared/my-orders/my-orders.component';
import { OrdersComponent } from './shared/orders/orders.component';
import { BuySellComponent } from './shared/buy-sell/buy-sell.component';
import { SelectedCurrencyComponent } from './shared/selected-currency/selected-currency.component';
import { TradingComponent } from './pages/trading/trading.component';
import { HistoryComponent } from './pages/history/history.component';
import { BuySellFormComponent } from './shared/buy-sell-form/buy-sell-form.component';
import { ExchangeGridComponent } from './pages/exchange-grid/exchange-grid.component';
import { CurrencyListComponent } from './pages/currency-list/currency-list.component';


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
    WalletComponent,
    ThemePickerComponent,
    TradeHistoryComponent,
    PriceHistoryComponent,
    MyOrdersComponent,
    OrdersComponent,
    BuySellComponent,
    DonateComponent,
    SelectedCurrencyComponent,
    TradingComponent,
    HistoryComponent,
    BuySellFormComponent,
    ExchangeGridComponent,
    CurrencyListComponent,
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
    MatSortModule,
    MatExpansionModule,
    MatLineModule,
    MatTooltipModule,
    MatTabsModule,
  ],
  providers: [
    ElectronService,
    WebsocketService,
    WebsocketHandlerService, 
    SidebarService,
    StyleManagerService,
    ThemeStorageService,
  ],
  bootstrap: [AppComponent]
})
export class AppModule {

}