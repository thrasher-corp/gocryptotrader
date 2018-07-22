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

const styles = theme => ({
  root: {
    flexGrow: 1
  },
  paper: {
    padding: theme.spacing.unit * 2,
    color: theme.palette.text.secondary
  },
  heading: {
    fontSize: theme.typography.pxToRem(15),
    flexBasis: '15%',
    flexShrink: 0
  },
  secondaryHeading: {
    fontSize: theme.typography.pxToRem(15),
    color: theme.palette.text.secondary
  }
});

const configGroups = [
  {
    heading: {
      primary: 'Webserver',
      secondary: 'webserver settings'
    },
    details: 'TODO'
  },
  {
    heading: {
      primary: 'Communication',
      secondary: '...'
    },
    details: 'TODO'
  },
  {
    heading: {
      primary: 'Portfolio',
      secondary: '...'
    },
    details: 'TODO'
  },
  {
    heading: {
      primary: 'Currency',
      secondary: '...'
    },
    details: 'TODO'
  },
  {
    heading: {
      primary: 'Exchanges',
      secondary: '...'
    },
    details: 'TODO'
  },
  {
    heading: {
      primary: 'BankAccounts',
      secondary: '...'
    },
    details: 'TODO'
  }
];

class Settings extends Component {
  state = {
    error: null,
    expanded: null,
    config: {}
  };

  handleChange = panel => (event, expanded) => {
    this.setState({
      expanded: expanded ? panel : false
    });
  };

  async componentDidMount() {
    this.mounted = true;
    try {
      const response = await fetch('/config/all');
      const config = await response.json();

      if (this.mounted) {
        this.setState({ config: config });
      }
    } catch (error) {
      if (this.mounted) {
        this.setState({ error: error });
      }
    }
  }

  componentWillUnmount() {
    this.mounted = false;
  }

  render() {
    const { classes } = this.props;
    const { expanded, config } = this.state;

    return (
      <div className={classes.root}>
        <Paper className={classes.paper}>
          <Typography variant="title" gutterBottom>
            Settings
          </Typography>
          <Typography variant="body1" gutterBottom>
            Finetune your settings for your bot named: <b>{config.name}</b>!
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
            <ExpansionPanelDetails>{group.details}</ExpansionPanelDetails>
          </ExpansionPanel>
        ))}
      </div>
    );
  }
}

Settings.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(Settings);
