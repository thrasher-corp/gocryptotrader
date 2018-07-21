import React from "react";
import PropTypes from "prop-types";
import { Paper, Typography, List, ListItem } from "@material-ui/core";
import { withStyles } from "@material-ui/core/styles";
import { DonationAddress, EmptyState, withFetching } from "../../components";

const styles = theme => ({});

const Donate = props => {
  const { classes, data, error, isLoading } = props;

  if (!data || error || isLoading) {
    return (
      <div className={classes.root}>
        <Paper className={classes.paper}>
          <Typography variant="h6" gutterBottom>
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
        <Typography variant="h6" gutterBottom>
          Donate
        </Typography>
        <Typography variant="h5" gutterBottom>
          To support the developers please consider making a donation:
        </Typography>
        <List>
          {data.map((d, index) => (
            <ListItem dense key={index}>
              <DonationAddress {...d} />
            </ListItem>
          ))}
        </List>
      </Paper>
    </div>
  );
};

Donate.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withFetching("data/donation-addresses.json")(
  withStyles(styles, { withTheme: true })(Donate)
);
