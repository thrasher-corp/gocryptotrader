import React from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import { List, ListItem, ListItemIcon, ListItemText, ListSubheader } from '@material-ui/core';
import { Link } from 'react-router-dom';
import Divider from '@material-ui/core/Divider';
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
  Code as CodeIcon,
} from '@material-ui/icons';

const styles = theme => ({
  hide: {
    display: 'none',
  },
});

const LinkListMenuItem = props => (
  <ListItem button component={Link} to={props.route}>
    <ListItemIcon>
      <props.icon />
    </ListItemIcon>
    <ListItemText primary={props.primaryText} />
  </ListItem>
);

const MenuItems = props => (
  <React.Fragment>
    <List>
      <LinkListMenuItem route="/" icon={DashboardIcon} primaryText="Dashboard" />
      <LinkListMenuItem route="/wallets" icon={WalletIcon} primaryText="Wallets" />
      <LinkListMenuItem route="/trading" icon={TradingIcon} primaryText="Trading" />
      <LinkListMenuItem route="/history" icon={HistoryIcon} primaryText="History" />
      <LinkListMenuItem route="/settings" icon={SettingsIcon} primaryText="Settings" />
      <LinkListMenuItem route="/about" icon={HelpIcon} primaryText="About" />
      <LinkListMenuItem route="/donations" icon={FavoriteIcon} primaryText="Donations" />
    </List>
    <Divider />
    <List subheader={<ListSubheader component="div" className={!props.expanded && props.classes.hide}>External links</ListSubheader>}>
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

MenuItems.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired,
};
export default withStyles(styles, { withTheme: true })(MenuItems);
