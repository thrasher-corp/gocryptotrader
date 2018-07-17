import { HomeComponent } from './pages/home/home.component';
import { AboutComponent } from './pages/about/about.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';
import { WalletComponent } from './pages/wallet/wallet.component';
import { DonateComponent } from './pages/donate/donate.component';
import { HistoryComponent } from './pages/history/history.component';
import { TradingComponent } from './pages/trading/trading.component';
import { ExchangeGridComponent } from './pages/exchange-grid/exchange-grid.component';
import { CurrencyListComponent } from './pages/currency-list/currency-list.component';

// Settings
import { SettingsComponent } from './pages/settings/settings.component';

import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';

const routes: Routes = [
    {
        path: '',
        component: HomeComponent
    },
    {
        path: 'about',
        component: AboutComponent
    },
    {
        path: 'dashboard',
        component: DashboardComponent
    },
    {
        path: 'wallet',
        component: WalletComponent
    }
    ,
    {
        path: 'donate',
        component: DonateComponent
    },
    {
        path: 'history',
        component: HistoryComponent
    },
    {
        path: 'trading',
        component: TradingComponent
    },
    {
        path: 'exchange-grid',
        component: ExchangeGridComponent
    },
    {
        path: 'currency-list',
        component: CurrencyListComponent
    },
    // Settings
    {
        path: 'settings',
        component: SettingsComponent
    },
];

@NgModule({
    imports: [RouterModule.forRoot(routes, {useHash: true})],
    exports: [RouterModule]
})
export class AppRoutingModule { }
