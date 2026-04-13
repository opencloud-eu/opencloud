import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';

import { withTranslation } from 'react-i18next';

import { withStyles } from '@material-ui/core/styles';
import Button from '@material-ui/core/Button';
import Typography from '@material-ui/core/Typography';
import DialogActions from '@material-ui/core/DialogActions';

import ResponsiveScreen from '../../components/ResponsiveScreen';
import { executeHello, executeLogoff } from '../../actions/common';

const styles = theme => ({
  subHeader: {
    marginBottom: theme.spacing(5)
  },
  wrapper: {
    marginTop: theme.spacing(5),
    position: 'relative',
    display: 'inline-block'
  }
});

class Goodbyescreen extends React.PureComponent {
  componentDidMount() {
    this.props.dispatch(executeHello());
  }

  render() {
    const { classes, branding, hello, t } = this.props;

    const loading = hello === null;
    return (
      <ResponsiveScreen loading={loading} branding={branding}>
        {hello !== null && !hello.state && (
          <div>
            <Typography variant="h5" component="h3">
              {t("konnect.goodbye.headline", "Goodbye")}
            </Typography>
            <Typography variant="subtitle1" className={classes.subHeader}>
              {t("konnect.goodbye.subHeader", "you have been signed out from your account")}
            </Typography>
            <Typography gutterBottom>
              {t("konnect.goodbye.message.close", "You can close this window now.")}
            </Typography>
          </div>
        )}
        {hello !== null && hello.state === true && (
          <div>
            <Typography variant="h5" component="h3">
              {t("konnect.goodbye.confirm.headline", "Hello {{displayName}}", { displayName: hello.displayName })}
            </Typography>
            <Typography variant="subtitle1" className={classes.subHeader}>
              {t("konnect.goodbye.confirm.subHeader", "please confirm sign out")}
            </Typography>

            <Typography gutterBottom>
              {t("konnect.goodbye.message.confirm", "Press the button below, to sign out from your account now.")}
            </Typography>

            <DialogActions>
              <div className={classes.wrapper}>
                <Button
                  color="secondary"
                  variant="outlined"
                  className={classes.button}
                  onClick={(event) => this.logoff(event)}
                >
                  {t("konnect.goodbye.signoutButton.label", "Sign out")}
                </Button>
              </div>
            </DialogActions>
          </div>
        )}
      </ResponsiveScreen>
    );
  }

  logoff(event) {
    event.preventDefault();

    this.props.dispatch(executeLogoff()).then((response) => {
      const { history } = this.props;

      if (response.success) {
        this.props.dispatch(executeHello());
        history.push('/goodbye');
      }
    });
  }
}

Goodbyescreen.propTypes = {
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

export default connect(mapStateToProps)(withStyles(styles)(withTranslation()(Goodbyescreen)));
