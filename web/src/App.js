import React, { Component } from 'react';
import PropTypes from 'prop-types';
import CssBaseline from '@material-ui/core/CssBaseline';
import { BrowserRouter as Router, Route, Switch } from 'react-router-dom';
import { MuiThemeProvider, createMuiTheme } from '@material-ui/core/styles';
import { purple, green } from '@material-ui/core/colors';
import { withStyles } from '@material-ui/core/styles';
import { AppBar, AppDrawer, MenuItems } from './components';
import { Home, About, Donate, Settings, NotFound } from './pages';
import './App.css';

const theme = createMuiTheme({
  palette: {
    type: 'dark',
    primary: {
      light: purple[300],
      main: purple[500],
      dark: purple[700]
    },
    secondary: {
      main: green[300]
    },
    contrastThreshold: 3,
    tonalOffset: 0.2
  }
});

const styles = theme => ({
  root: {
    flexGrow: 1,
    height: '100%',
    zIndex: 1,
    overflow: 'hidden',
    position: 'relative',
    display: 'flex'
  },
  content: {
    flexGrow: 1,
    padding: theme.spacing.unit * 3
  },
  toolbar: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    padding: '0 8px',
    ...theme.mixins.toolbar
  }
});

const routes = [
  {
    path: '/',
    exact: true,
    content: Home
  },
  {
    path: '/about',
    exact: true,
    content: About
  },
  {
    path: '/donate',
    exact: true,
    content: Donate
  },
  {
    path: '/settings',
    exact: true,
    content: Settings
  }
];

class App extends Component {
  state = {
    drawerIsOpen: false
  };

  handleDrawerOpen = () => {
    this.setState({ drawerIsOpen: true });
  };

  handleDrawerClose = () => {
    this.setState({ drawerIsOpen: false });
  };

  render() {
    const { classes } = this.props;

    return (
      <MuiThemeProvider theme={theme}>
        <CssBaseline />
        <Router>
          <div className={classes.root}>
            <AppBar
              drawerIsOpen={this.state.drawerIsOpen}
              handleDrawerOpen={this.handleDrawerOpen}
            />
            <AppDrawer
              drawerIsOpen={this.state.drawerIsOpen}
              handleDrawerClose={this.handleDrawerClose}
            >
              <MenuItems expanded={this.state.drawerIsOpen} />
            </AppDrawer>
            <main className={classes.content}>
              <div className={classes.toolbar} />
              <Switch>
                {routes.map((route, index) => (
                  <Route
                    key={index}
                    exact={route.exact}
                    path={route.path}
                    component={route.content}
                  />
                ))}
                <Route component={NotFound} />
              </Switch>
            </main>
          </div>
        </Router>
      </MuiThemeProvider>
    );
  }
}

App.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(App);
