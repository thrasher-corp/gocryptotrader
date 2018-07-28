import React, { Component } from 'react';
import PropTypes from 'prop-types';
import {
  Typography,
  Paper,
  ExpansionPanel,
  ExpansionPanelSummary,
  ExpansionPanelDetails
} from '@material-ui/core';
import { ExpandMore as ExpandMoreIcon } from '@material-ui/icons';
import { withStyles } from '@material-ui/core/styles';
import { EmptyState, WebserverSettings, withFetching } from '../../components';
import { pageStyles } from '../styles';

const styles = theme => ({
  ...pageStyles(theme),
  heading: {
    fontSize: theme.typography.pxToRem(15),
    flexBasis: '15%',
    flexShrink: 0
  },
  secondaryHeading: {
    fontSize: theme.typography.pxToRem(15),
    color: theme.palette.text.secondary
  },
  json: {
    width: '100%',
    height: '500px',
    padding: '1.45em'
  }
});

const configGroups = [
  {
    heading: {
      primary: 'Webserver',
      secondary: 'webserver settings'
    },
    details: props => <WebserverSettings data={props.data.webserver} />
  },
  {
    heading: {
      primary: 'Exchanges',
      secondary: '...'
    },
    details: props => (
      <textarea
        readOnly
        className={props.classes.json}
        value={JSON.stringify(props.data.exchanges, null, 2)}
      />
    )
  },
  {
    heading: {
      primary: 'Currency',
      secondary: '...'
    },
    details: props => (
      <textarea
        readOnly
        className={props.classes.json}
        value={JSON.stringify(props.data.currencyConfig, null, 2)}
      />
    )
  },
  {
    heading: {
      primary: 'Portfolio',
      secondary: '...'
    },
    details: props => (
      <textarea
        readOnly
        className={props.classes.json}
        value={JSON.stringify(props.data.portfolioAddresses, null, 2)}
      />
    )
  },
  {
    heading: {
      primary: 'BankAccounts',
      secondary: '...'
    },
    details: props => (
      <textarea
        readOnly
        className={props.classes.json}
        value={JSON.stringify(props.data.bankAccounts, null, 2)}
      />
    )
  },
  {
    heading: {
      primary: 'Communication',
      secondary: '...'
    },
    details: props => (
      <textarea
        readOnly
        className={props.classes.json}
        value={JSON.stringify(props.data.communications, null, 2)}
      />
    )
  }
];

class Settings extends Component {
  state = {
    expanded: 'panel0'
  };

  handleChange = panel => (event, expanded) => {
    this.setState({
      expanded: expanded ? panel : false
    });
  };

  render() {
    const { classes, data, error, isLoading } = this.props;
    const { expanded } = this.state;

    if (!data || error || isLoading) {
      return (
        <div className={classes.root}>
          <Paper className={classes.paper}>
            <Typography variant="title" gutterBottom>
              Settings
            </Typography>
            <EmptyState data={data} error={error} isLoading={isLoading} />
          </Paper>
        </div>
      );
    }

    return (
      <div className={classes.root}>
        <Paper className={classes.paper}>
          <Typography variant="title" gutterBottom>
            Settings
          </Typography>
          <Typography variant="body1" gutterBottom>
            Finetune your settings for your bot named: <b>{data.name}</b>!
          </Typography>
        </Paper>

        {configGroups.map((group, index) => (
          <ExpansionPanel
            key={index}
            expanded={expanded === 'panel' + index}
            onChange={this.handleChange('panel' + index)}
          >
            <ExpansionPanelSummary expandIcon={<ExpandMoreIcon />}>
              <Typography className={classes.heading}>
                {group.heading.primary}
              </Typography>
              <Typography className={classes.secondaryHeading}>
                {group.heading.secondary}
              </Typography>
            </ExpansionPanelSummary>
            <ExpansionPanelDetails>
              {<group.details {...this.props} />}
            </ExpansionPanelDetails>
          </ExpansionPanel>
        ))}
      </div>
    );
  }
}

Settings.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired,
  isLoading: PropTypes.bool.isRequired,
  data: PropTypes.object,
  error: PropTypes.object
};

export default withFetching('/config/all')(
  withStyles(styles, { withTheme: true })(Settings)
);
