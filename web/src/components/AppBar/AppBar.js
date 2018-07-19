import React from 'react';
import PropTypes from 'prop-types';
import classNames from 'classnames';
import { withStyles } from '@material-ui/core/styles';
import { IconButton, Typography } from '@material-ui/core';
import { AppBar, Toolbar } from '@material-ui/core';
import { Menu as MenuIcon } from '@material-ui/icons';
import { AccountMenu } from './..';

const drawerWidth = 240;

const styles = theme => ({
  flex: {
    flexGrow: 1,
  },
  appBar: {
    zIndex: theme.zIndex.drawer + 1,
    transition: theme.transitions.create(['width', 'margin'], {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.leavingScreen,
    }),
  },
  appBarShift: {
    marginLeft: drawerWidth,
    width: `calc(100% - ${drawerWidth}px)`,
    transition: theme.transitions.create(['width', 'margin'], {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.enteringScreen,
    }),
  },
  menuButton: {
    marginLeft: 12,
    marginRight: 36,
  },
  hide: {
    display: 'none',
  },
  toolbar: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    padding: '0 8px',
    ...theme.mixins.toolbar,
  },
});

const MenuAppBar = (props) => {
  const { classes, drawerIsOpen, handleDrawerOpen } = props;

  return (
    <React.Fragment>
      <AppBar position="absolute" className={classNames(classes.appBar, drawerIsOpen && classes.appBarShift)}>
        <Toolbar disableGutters={!drawerIsOpen}>
          <IconButton
            onClick={handleDrawerOpen}
            className={classNames(classes.menuButton, drawerIsOpen && classes.hide)}
            color="inherit"
            aria-label="Menu"
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="title" color="inherit" noWrap className={classes.flex}>
            Crypto Trader
          </Typography>
          <AccountMenu />
        </Toolbar>
      </AppBar>
    </React.Fragment>
  );
};

MenuAppBar.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired,
};

export default withStyles(styles, { withTheme: true })(MenuAppBar);
