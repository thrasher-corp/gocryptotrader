import 'zone.js/dist/zone-mix';
import 'reflect-metadata';
import 'polyfills';

import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { HttpModule } from '@angular/http';
import { NgModule, Injectable } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { AmChartsModule } from '@amcharts/amcharts3-angular';

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
  MatSnackBarModule,
  MatDialogModule,
} from '@angular/material';

// Pages
import { AppComponent } from './app.component';
import { HomeComponent } from './pages/home/home.component';
import { AboutComponent } from './pages/about/about.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';
import { WalletComponent } from './pages/wallet/wallet.component';
import { DonateComponent } from './pages/donate/donate.component';

import { SettingsComponent, EnabledCurrenciesDialogueComponent } from './pages/settings/settings.component';

// Shared
import { NavbarComponent } from './shared/navbar/navbar.component';
import { AllEnabledCurrencyTickersComponent } from './shared/all-updates-ticker/all-updates-ticker.component';
import { ThemePickerComponent } from './shared/theme-picker/theme-picker.component';
import {IterateMapPipe, EnabledCurrenciesPipe} from './shared/classes/pipes';
// services
import { WebsocketService } from './services/websocket/websocket.service';
import { WebsocketResponseHandlerService } from './services/websocket-response-handler/websocket-response-handler.service';
import { SidebarService } from './services/sidebar/sidebar.service';
import { ElectronService } from './providers/electron.service';
import { StyleManagerService } from './services/style-manager/style-manager.service';
import { ThemeStorageService } from './services/theme-storage/theme-storage.service';

// Routing
import { AppRoutingModule } from './app-routing.module';

import { Wallet } from './shared/classes/wallet';

import { TradeHistoryComponent } from './shared/trade-history/trade-history.component';
import { PriceHistoryComponent } from './shared/price-history/price-history.component';
import { MyOrdersComponent } from './shared/my-orders/my-orders.component';
import { OrdersComponent } from './shared/orders/orders.component';
import { BuySellComponent } from './shared/buy-sell/buy-sell.component';
import { SelectedCurrencyComponent } from './shared/selected-currency/selected-currency.component';
import { TradingComponent } from './pages/trading/trading.component';
import { HistoryComponent } from './pages/history/history.component';
import { BuyFormComponent } from './shared/buy-form/buy-form.component';
import { ExchangeGridComponent } from './pages/exchange-grid/exchange-grid.component';
import { CurrencyListComponent } from './pages/currency-list/currency-list.component';
import { SellFormComponent } from './shared/sell-form/sell-form.component';


@NgModule({
  declarations: [
    AppComponent,
    HomeComponent,
    AboutComponent,
    NavbarComponent,
    SettingsComponent,
    DashboardComponent,
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
    BuyFormComponent,
    ExchangeGridComponent,
    CurrencyListComponent,
    SellFormComponent,
    IterateMapPipe,
    EnabledCurrenciesPipe,
    EnabledCurrenciesDialogueComponent
  ],
  entryComponents: [
    EnabledCurrenciesDialogueComponent
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
    MatSnackBarModule,
    MatDialogModule,
    AmChartsModule,
  ],
  providers: [
    ElectronService,
    WebsocketService,
    WebsocketResponseHandlerService,
    SidebarService,
    StyleManagerService,
    ThemeStorageService,
  ],
  bootstrap: [AppComponent]
})
export class AppModule {

}
