import React from 'react';
import PropTypes from 'prop-types';
import Typography from '@material-ui/core/Typography';
import { withStyles } from '@material-ui/core/styles';
import logo from './logo.svg';
import './Home.css';

const styles = theme => ({
  logo: {
    animation: 'App-logo-spin infinite 20s linear',
    height: '200px'
  }
});

const Home = props => {
  const { classes } = props;
  return (
    <React.Fragment>
      <Typography noWrap>
        GoCryptoTrader is a crypto trading bot with back testing support and
        support for a bunch of popular exchanges.
      </Typography>
      <img src={logo} className={classes.logo} alt="logo" />
    </React.Fragment>
  );
};

Home.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(Home);
