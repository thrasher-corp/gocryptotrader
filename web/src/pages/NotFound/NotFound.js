import React from 'react';
import PropTypes from 'prop-types';
import { Paper } from '@material-ui/core';
import { withStyles } from '@material-ui/core/styles';
import { pageStyles } from '../styles';

const styles = theme => ({
  ...pageStyles(theme)
});

const NotFound = props => {
  const { classes } = props;
  return (
    <div className={classes.root}>
      <Paper className={classes.paper}>
        <p>We can not find that page.</p>
      </Paper>
    </div>
  );
};

NotFound.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(NotFound);
