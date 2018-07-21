import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import classNames from 'classnames';
import { withStyles } from '@material-ui/core/styles';
import { Chip, Avatar, Tooltip } from '@material-ui/core';

const styles = theme => ({
  hidden: {
    position: 'absolute',
    top: '-1000px'
  },
  chip: {
    margin: theme.spacing.unit
  },
  bigAvatar: {
    width: '45px',
    height: '45px'
  },
  btc: {
    color: '#fff',
    backgroundColor: '#eaa800'
  },
  bcc: {
    color: '#fff',
    backgroundColor: '#2f6fd6'
  },
  eth: {
    color: '#fff',
    backgroundColor: '#222'
  }
});
class DonationAddress extends Component {
  state = {
    showTooltip: false,
    textarea: null
  };

  copyToClipboard = () => {
    const { address } = this.props;
    try {
      this.textarea.value = address;
      this.textarea.select();
      document.execCommand('copy');
      this.setState({ showTooltip: true });
      setTimeout(() => this.setState({ showTooltip: false }), 1500);
    } catch (err) {
      console.error('Failed to copy DonationAddress to clipboard.');
    }
  };

  render() {
    const { currency, address, classes } = this.props;
    const { showTooltip } = this.state;

    return (
      <Fragment>
        <Tooltip
          title="Copied to clipboard"
          placement="right"
          TransitionProps={{ timeout: 600 }}
          open={showTooltip}
          disableFocusListener
          disableHoverListener
        >
          <Chip
            avatar={
              <Avatar
                className={classNames(
                  classes.bigAvatar,
                  classes[currency.toLowerCase()]
                )}
              >
                {currency.toUpperCase()}
              </Avatar>
            }
            label={address}
            onClick={this.copyToClipboard}
            className={classes.chip}
          />
        </Tooltip>
        <textarea
          ref={textarea => (this.textarea = textarea)}
          className={classes.hidden}
        />
      </Fragment>
    );
  }
}
DonationAddress.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired,
  currency: PropTypes.string.isRequired,
  address: PropTypes.string.isRequired
};
export default withStyles(styles, { withTheme: true })(DonationAddress);
