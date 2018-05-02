import { HomeComponent } from './pages/home/home.component';
import { AboutComponent } from './pages/about/about.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';
import { WalletComponent } from './pages/wallet/wallet.component';
import { DonateComponent } from './pages/donate/donate.component';
import { HistoryComponent } from './pages/history/history.component';
import { TradingComponent } from './pages/trading/trading.component';
import { ExchangeGridComponent } from './pages/exchange-grid/exchange-grid.component';
import { CurrencyListComponent } from './pages/currency-list/currency-list.component';

//Settings
import { SettingsComponent } from './pages/settings/settings.component';
import { SettingsCredentialsComponent } from './pages/settings/settings-credentials/settings-credentials.component';
import { SettingsSmsComponent } from './pages/settings/settings-sms/settings-sms.component';
import { SettingsWalletsComponent } from './pages/settings/settings-wallets/settings-wallets.component';
import { SettingsExchangesComponent } from './pages/settings/settings-exchanges/settings-exchanges.component';


import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';

const routes: Routes = [
    {
        path: '',
        component: HomeComponent
    },
    {
        path:'about',
        component: AboutComponent
    },    
    {
        path:'dashboard',
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
    {
        path: 'help',
        component: CurrencyListComponent
    },
    //Settings
    {
        path: 'settings',
        component: SettingsComponent
    },
    {
        path: 'settings/credentials',
        component: SettingsCredentialsComponent
    },
    {
        path: 'settings/wallets',
        component: SettingsWalletsComponent
    },
    {
        path: 'settings/sms',
        component: SettingsSmsComponent
    },
    {
        path: 'settings/exchanges',
        component: SettingsExchangesComponent
    },

];

@NgModule({
    imports: [RouterModule.forRoot(routes, {useHash: true})],
    exports: [RouterModule]
})
export class AppRoutingModule { }
