import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import classNames from 'classnames';

import { Trans } from 'react-i18next';

import { withStyles } from '@material-ui/core/styles';
import Grid from '@material-ui/core/Grid';
import DialogContent from '@material-ui/core/DialogContent';

import Loading from './Loading';
import { OpenCloudContext } from "../openCloudContext";

const styles = theme => ({
  root: {
    display: 'flex',
    flex: 1,
    zIndex: 999
  },
  content: {
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'center',
    alignItems: 'center'
  },
  actions: {
    marginTop: -40,
    justifyContent: 'flex-start',
    paddingLeft: theme.spacing(3),
    paddingRight: theme.spacing(3)
  },
  wrapper: {
    display: 'flex',
    flex: 1,
    alignItems: 'center'
  }
});

const ResponsiveScreen = (props) => {
  const {
    classes,
    withoutLogo,
    withoutPadding,
    loading,
    children,
    className,
    ...other
  } = props;
  const { theme } = useContext(OpenCloudContext);

  const logo = (theme && !withoutLogo) ? (
        <img src={'/' + theme.common?.logo} className="oc-logo" alt="OpenCloud Logo"/>
    ) : null;

  const content = loading ? <Loading/> : (withoutPadding ? children : <DialogContent>{children}</DialogContent>);

  return (
    <Grid container justifyContent="center" alignItems="center" direction="column" spacing={0}
      className={classNames(classes.root, className)} {...other}>
        <div className={classes.wrapper}>
            <div className={classes.content}>
              {logo}
              <div className={'oc-card'}>
                <div className={'oc-card-body'}>
                {content}
                </div>
              </div>
            </div>
        </div>
        <footer className="oc-footer-message">
              <Trans i18nKey="konnect.footer.slogan"><strong>OpenCloud</strong> - excellent file sharing</Trans>
        </footer>
    </Grid>
  );
};

ResponsiveScreen.defaultProps = {
  withoutLogo: false,
  withoutPadding: false,
  loading: false
};

ResponsiveScreen.propTypes = {
  classes: PropTypes.object.isRequired,
  withoutLogo: PropTypes.bool,
  withoutPadding: PropTypes.bool,
  loading: PropTypes.bool,
  branding: PropTypes.object,
  children: PropTypes.node.isRequired,
  className: PropTypes.string,
  PaperProps: PropTypes.object,
  DialogProps: PropTypes.object
};

export default withStyles(styles)(ResponsiveScreen);
