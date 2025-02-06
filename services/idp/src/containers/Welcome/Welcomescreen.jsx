import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';

import { withTranslation } from 'react-i18next';

import { withStyles } from '@material-ui/core/styles';
import Button from '@material-ui/core/Button';
import Typography from '@material-ui/core/Typography';
import DialogActions from '@material-ui/core/DialogActions';

import ResponsiveScreen from '../../components/ResponsiveScreen';
import { executeLogoff } from '../../actions/common';

const styles = theme => ({
  button: {
    margin: theme.spacing(1),
    minWidth: 100
  },
  subHeader: {
    marginBottom: theme.spacing(5)
  }
});

class Welcomescreen extends React.PureComponent {
  render() {
    const { classes, branding, hello, t } = this.props;

    const loading = hello === null;
    return (
      <ResponsiveScreen loading={loading} branding={branding}>
        <Typography variant="h5" component="h3">
          {t("konnect.welcome.headline", "Welcome {{displayName}}", {displayName: hello.displayName})}
        </Typography>
        <Typography variant="subtitle1" className={classes.subHeader}>
          {hello.username}
        </Typography>

        <Typography gutterBottom>
          {t("konnect.welcome.message", "You are signed in - awesome!")}
        </Typography>

        <DialogActions>
          <Button
            color="secondary"
            className={classes.button}
            variant="contained"
            onClick={(event) => this.logoff(event)}
          >
            {t("konnect.welcome.signoutButton.label", "Sign out")}
          </Button>
        </DialogActions>
      </ResponsiveScreen>
    );
  }

  logoff(event) {
    event.preventDefault();

    this.props.dispatch(executeLogoff()).then((response) => {
      const { history } = this.props;

      if (response.success) {
        history.push('/identifier');
      }
    });
  }
}

Welcomescreen.propTypes = {
  classes: PropTypes.object.isRequired,
  t: PropTypes.func.isRequired,

  branding: PropTypes.object,
  hello: PropTypes.object,

  dispatch: PropTypes.func.isRequired,
  history: PropTypes.object.isRequired
};

const mapStateToProps = (state) => {
  const { branding, hello } = state.common;

  return {
    branding,
    hello
  };
};

export default connect(mapStateToProps)(withStyles(styles)(withTranslation()(Welcomescreen)));
