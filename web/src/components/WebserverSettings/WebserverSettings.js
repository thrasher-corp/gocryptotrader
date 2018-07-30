import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classNames from 'classnames';
import { withStyles } from '@material-ui/core/styles';
import {
  Grid,
  Typography,
  FormGroup,
  FormControlLabel,
  Switch,
  TextField
} from '@material-ui/core';
import { SecretInput } from '../';

const styles = theme => ({
  root: {
    display: 'flex',
    flexWrap: 'wrap',
    width: '100%'
  },
  withoutLabel: {
    marginTop: theme.spacing.unit * 3
  },
  textField: {
    flexBasis: 200
  },
  formControl: {
    margin: theme.spacing.unit
  }
});
class WebserverSettings extends Component {
  state = {
    data: {}
  };

  handleChange = prop => event => {
    this.setState(prevState => ({
      ...prevState,
      data: {
        ...prevState.data,
        [prop]: event.target.value
      }
    }));
  };

  render() {
    const { classes, data } = this.props;

    return (
      <form className={classes.root} autoComplete="off">
        <Grid container spacing={24}>
          <Grid item xs={12} sm={6}>
            <Typography variant="subheading" gutterBottom>
              Connection
            </Typography>
            <FormGroup row>
              <TextField
                id="listen"
                label="Listen Address"
                className={classNames(classes.formControl, classes.textField)}
                value={data.listenAddress}
                onChange={this.handleChange('listenAddress')}
                margin="normal"
              />
              <TextField
                id="username"
                label="Username"
                className={classNames(classes.formControl, classes.textField)}
                value={data.adminUsername}
                onChange={this.handleChange('adminUsername')}
                margin="normal"
              />
              <SecretInput
                id="adornment-password"
                label="Password"
                value={data.adminPassword}
              />
            </FormGroup>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Typography variant="subheading" gutterBottom>
              Websocket
            </Typography>
            <FormGroup row>
              <TextField
                id="websocket-limit"
                label="Limit"
                className={classNames(classes.formControl, classes.textField)}
                value={data.websocketConnectionLimit}
                onChange={this.handleChange('websocketConnectionLimit')}
                type="number"
                margin="normal"
              />
              <TextField
                id="websocket-max-auth-failures"
                label="Max auth Failures"
                className={classNames(classes.formControl, classes.textField)}
                value={data.websocketMaxAuthFailures}
                onChange={this.handleChange('websocketMaxAuthFailures')}
                type="number"
                margin="normal"
              />
              <FormControlLabel
                control={
                  <Switch
                    checked={data.websocketAllowInsecureOrigin}
                    onChange={this.handleChange('websocketAllowInsecureOrigin')}
                    value="true"
                  />
                }
                label="Allow insecure"
              />
            </FormGroup>
          </Grid>
        </Grid>
      </form>
    );
  }
}

WebserverSettings.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired,
  data: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(WebserverSettings);
