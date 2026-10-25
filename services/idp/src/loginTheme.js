/**
 * Consumes the login relevant subset of the theme.json served by the web
 * service (see services/web/pkg/theme) and applies it to the login UI.
 */

import { unstable_createMuiStrictModeTheme as createMuiTheme } from '@material-ui/core';

import { themeOptions } from './theme';

const IDP_CSS_PREFIX = '--oc-idp-';

const toCssProperty = (key) => IDP_CSS_PREFIX + key.replace(/[A-Z]/g, (c) => '-' + c.toLowerCase());

const isAbsoluteUrl = (url) => /^(https?:)?\/\/|^data:/i.test(url);

export const resolveThemeUrl = (url) => {
  if (!url || isAbsoluteUrl(url)) {
    return url;
  }
  return '/' + url.replace(/^\/+/, '');
};

export const extractIdpTheme = (theme) => {
  const idp = (theme && theme.clients && theme.clients.idp) || {};
  const colors = idp.colors || {};
  return {
    backgroundImage: resolveThemeUrl(idp.backgroundImage),
    bgIcon: resolveThemeUrl(idp.bgIcon),
    colors: colors,
    favicon: resolveThemeUrl(idp.favicon),
    signinPageLogoURI: resolveThemeUrl(idp.signinPageLogoURI),
    signinPageText: idp.signinPageText,
    themeColor: idp.themeColor,
  };
};

export const applyLoginTheme = (idp, common) => {
  const rootStyle = document.documentElement.style;

  Object.entries(idp.colors).forEach(([key, value]) => {
    if (value) {
      rootStyle.setProperty(toCssProperty(key), value);
    }
  });

  if (idp.themeColor) {
    const meta = document.querySelector('meta[name="theme-color"]');
    if (meta) {
      meta.setAttribute('content', idp.themeColor);
    }
  }

  if (idp.favicon) {
    const link = document.querySelector('link[rel="icon"]');
    if (link) {
      link.setAttribute('href', idp.favicon);
    }
  }

  if (common && common.name) {
    document.title = `Sign in - ${common.name}`;
  }
};

export const buildMuiTheme = (idp) => {
  const c = idp.colors || {};
  const palette = { ...themeOptions.palette };
  if (c.primary) {
    palette.primary = { main: c.primary, contrastText: c.onPrimary || '#ffffff' };
  }
  if (c.secondary) {
    palette.secondary = { main: c.secondary, contrastText: c.onSecondary || '#ffffff' };
  }
  if (c.error) {
    palette.error = { main: c.error, contrastText: '#ffffff' };
  }
  if (c.background) {
    palette.background = { default: c.background, paper: c.surface || '#ffffff' };
  }
  return createMuiTheme({ ...themeOptions, palette });
};
