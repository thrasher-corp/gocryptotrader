import React from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';

const styles = theme => ({});

const About = props => {
  return (
    <React.Fragment>
      <p>
        A cryptocurrency trading bot supporting multiple exchanges written in
        Golang. Join our slack to discuss all things related to GoCryptoTrader!{' '}
        <a href="https://gocryptotrader.herokuapp.com/">GoCryptoTrader Slack</a>
      </p>
    </React.Fragment>
  );
};

About.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(About);
