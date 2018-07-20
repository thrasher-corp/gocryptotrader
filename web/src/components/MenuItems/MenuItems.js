import React from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import {
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  ListSubheader
} from '@material-ui/core';
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
        <ListItemLink
          to="/donations"
          icon={<FavoriteIcon />}
          primary="Donations"
        />
      </List>
      <Divider />
      <List
        subheader={
          <ListSubheader component="div" className={!expanded && classes.hide}>
            External links
          </ListSubheader>
        }
      >
        <ListItem button>
          <ListItemIcon>
            <CodeIcon />
          </ListItemIcon>
          <ListItemText primary="Github" />
        </ListItem>
        <ListItem button>
          <ListItemIcon>
            <AppsIcon />
          </ListItemIcon>
          <ListItemText primary="Slack" />
        </ListItem>
        <ListItem button>
          <ListItemIcon>
            <AssignmentIcon />
          </ListItemIcon>
          <ListItemText primary="Trello" />
        </ListItem>
      </List>
    </React.Fragment>
  );
};

MenuItems.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};
export default withStyles(styles, { withTheme: true })(MenuItems);
