import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import { LinearProgress } from '@material-ui/core';

const styles = theme => ({
  root: {
    flexGrow: 1
  }
});
const EmptyState = props => {
  const { classes, data, error, isLoading } = props;
  if (!data) {
    return (
      <div className={classes.root}>
        {isLoading ? <LinearProgress color="secondary" variant="query" /> : ''}
        <p>No data...</p>
      </div>
    );
  }
  if (error) {
    return (
      <div className={classes.root}>
        <p>
          <b>Something went wrong: </b>
          {error.message}...
        </p>
      </div>
    );
  }
  return <Fragment />;
};

EmptyState.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired,
  data: PropTypes.object,
  error: PropTypes.object,
  isLoading: PropTypes.bool.isRequired
};

export default withStyles(styles, { withTheme: true })(EmptyState);
