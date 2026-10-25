import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';

import { withStyles } from '@material-ui/core/styles';
import Grid from '@material-ui/core/Grid';
import DialogContent from '@material-ui/core/DialogContent';

import Loading from './Loading';
import { OpenCloudContext } from "../openCloudContext";
import { resolveThemeUrl } from "../loginTheme";

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
    branding,
    ...other
  } = props;
  const { theme } = useContext(OpenCloudContext);

  const productName = theme?.common?.name;
  const productSlogan = theme?.common?.slogan;

  // footer: use the themable product name and slogan. The name is bolded when
  // the slogan starts with it, otherwise it is prepended. Missing values fall
  // back to the translated default.
  let footer = null;
  if (productSlogan) {
    if (productName && productSlogan.startsWith(productName)) {
      footer = (
        <>
          <strong>{productName}</strong>
          {productSlogan.slice(productName.length)}
        </>
      );
    } else if (productName && !productSlogan.includes(productName)) {
      footer = (
        <>
          <strong>{productName}</strong> - {productSlogan}
        </>
      );
    } else {
      footer = productSlogan;
    }
  }

  // the theme.json logo target URI takes precedence over the branding
  // delivered through the IDP hello endpoint
  const logoTargetURI =
    theme?.clients?.idp?.signinPageLogoURI || branding?.signinPageLogoURI;

  const logo = (theme && !withoutLogo) ? (
        <img src={resolveThemeUrl(theme.common?.logo)} className="oc-logo" alt={theme.common?.name || 'OpenCloud Logo'}/>
    ) : null;

  const content = loading ? <Loading/> : (withoutPadding ? children : <DialogContent>{children}</DialogContent>);

  return (
    <Grid
      container
      justifyContent="center"
      alignItems="center"
      direction="column"
      spacing={0}
      className={[classes.root, className].filter(Boolean).join(' ')}
      {...other}
    >
      <div className={classes.wrapper}>
        <div className={classes.content}>
          {logoTargetURI ? (
            <a
              href={resolveThemeUrl(logoTargetURI)}
              target="_blank"
              rel="noopener noreferrer"
              className='oc-logo-container'
            >
              {logo}
            </a>
          ) : (
            <div className='oc-logo-container'>
              {logo}
            </div>
          )}
          <div className={"oc-card"}>
            <div className={"oc-card-body"}>{content}</div>
          </div>
        </div>
      </div>
      <footer className="oc-footer-message">
        {footer || (
          <Trans i18nKey="konnect.footer.slogan">
            <strong>OpenCloud</strong> - excellent file sharing
          </Trans>
        )}
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
