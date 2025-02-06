import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';

import { withTranslation } from 'react-i18next';

import LinearProgress from '@material-ui/core/LinearProgress';
import Grid from '@material-ui/core/Grid';
import Typography from '@material-ui/core/Typography';
import Button from '@material-ui/core/Button';
import renderIf from 'render-if';

import { retryHello } from '../actions/common';
import { ErrorMessage } from '../errors';

class Loading extends React.PureComponent {
  render() {
    const { error, t } = this.props;

    return (
        <Grid item align="center">
          {renderIf(error === null)(() => (
            <LinearProgress className="oc-progress" />
          ))}
          {renderIf(error !== null)(() => (
            <div>
              <Typography variant="h5" gutterBottom align="center">
                {t("konnect.loading.error.headline", "Failed to connect to server")}
              </Typography>
              <Typography align="center" color="error">
                <ErrorMessage error={error}></ErrorMessage>
              </Typography>
              <Button
                autoFocus
                color="primary"
                variant="outlined"
                className="oc-button-primary oc-mt-l"
                onClick={(event) => this.retry(event)}
              >
                {t("konnect.login.retryButton.label", "Retry")}
              </Button>
            </div>
          ))}
        </Grid>
    );
  }

  retry(event) {
    event.preventDefault();

    this.props.dispatch(retryHello());
  }
}

Loading.propTypes = {
  classes: PropTypes.object.isRequired,
  t: PropTypes.func.isRequired,

  error: PropTypes.object,

  dispatch: PropTypes.func.isRequired,
};

const mapStateToProps = (state) => {
  const { error } = state.common;

  return {
    error
  };
};

export default connect(mapStateToProps)(withTranslation()(Loading));
