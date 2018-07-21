import React from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import { List, ListSubheader } from '@material-ui/core';
import Divider from '@material-ui/core/Divider';
import { ListItemLink } from './..';
import {
  Dashboard as DashboardIcon,
  Settings as SettingsIcon,
  Apps as AppsIcon,
  AccountBalanceWallet as WalletIcon,
  Help as HelpIcon,
  History as HistoryIcon,
  SwapHoriz as TradingIcon,
  Favorite as FavoriteIcon,
  Assignment as AssignmentIcon,
  Code as CodeIcon
} from '@material-ui/icons';

const styles = theme => ({
  hide: {
    display: 'none'
  }
});

const MenuItems = props => {
  const { classes, expanded } = props;

  return (
    <React.Fragment>
      <List component="nav">
        <ListItemLink to="/" icon={<DashboardIcon />} primary="Dashboard" />
        <ListItemLink to="/wallets" icon={<WalletIcon />} primary="Wallets" />
        <ListItemLink to="/trading" icon={<TradingIcon />} primary="Trading" />
        <ListItemLink to="/history" icon={<HistoryIcon />} primary="History" />
        <ListItemLink
          to="/settings"
          icon={<SettingsIcon />}
          primary="Settings"
        />
        <ListItemLink to="/about" icon={<HelpIcon />} primary="About" />
        <ListItemLink to="/donate" icon={<FavoriteIcon />} primary="Donate" />
      </List>
      <Divider />
      <List
        subheader={
          <ListSubheader component="div" className={!expanded && classes.hide}>
            External links
          </ListSubheader>
        }
      >
        <ListItemLink
          to="https://github.com/thrasher-/gocryptotrader"
          icon={<CodeIcon />}
          primary="Github"
          external
        />
        <ListItemLink
          to="https://gocryptotrader.slack.com"
          icon={<AppsIcon />}
          primary="Slack"
          external
        />
        <ListItemLink
          to="https://trello.com/b/ZAhMhpOy/gocryptotrader"
          icon={<AssignmentIcon />}
          primary="Trello"
          external
        />
      </List>
    </React.Fragment>
  );
};

MenuItems.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};
export default withStyles(styles, { withTheme: true })(MenuItems);
