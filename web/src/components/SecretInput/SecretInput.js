import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import classNames from 'classnames';
import { withStyles } from '@material-ui/core/styles';
import {
  FormControl,
  Input,
  InputLabel,
  InputAdornment,
  IconButton
} from '@material-ui/core';
import {
  Visibility as VisibilityIcon,
  VisibilityOff as VisibilityOffIcon
} from '@material-ui/icons';

const styles = theme => ({
  textField: {
    flexBasis: 200
  },
  formControl: {
    margin: theme.spacing.unit
  }
});
class SecretInput extends Component {
  constructor(props) {
    super(props);

    const { value } = props;
    this.state = {
      value,
      showSecret: false
    };
  }

  handleChange = prop => event => {
    this.setState(prevState => ({
      ...prevState
    }));
  };

  handleMouseDownSecret = event => {
    event.preventDefault();
  };

  handleClickShowSecret = () => {
    this.setState(prevState => ({ showSecret: !prevState.showSecret }));
  };

  render() {
    const { classes, id, label } = this.props;
    const { value } = this.state;

    return (
      <FormControl
        className={classNames(classes.formControl, classes.textField)}
      >
        {label ? <InputLabel htmlFor={id}>{label}</InputLabel> : <Fragment />}
        <Input
          id={id}
          type={this.state.showSecret ? 'text' : 'password'}
          value={value}
          onChange={this.handleChange('Secret')}
          endAdornment={
            <InputAdornment position="end">
              <IconButton
                aria-label={`Toggle ${label} visibility`}
                onClick={this.handleClickShowSecret}
                onMouseDown={this.handleMouseDownSecret}
              >
                {this.state.showSecret ? (
                  <VisibilityOffIcon />
                ) : (
                  <VisibilityIcon />
                )}
              </IconButton>
            </InputAdornment>
          }
        />
      </FormControl>
    );
  }
}
SecretInput.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired,
  id: PropTypes.string.isRequired,
  label: PropTypes.string,
  value: PropTypes.string
};
export default withStyles(styles, { withTheme: true })(SecretInput);
