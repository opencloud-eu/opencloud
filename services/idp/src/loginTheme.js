/**
 * Consumes the login relevant subset of the theme.json served by the web
 * service (see services/web/pkg/theme) and applies it to the login UI.
 */

import { unstable_createMuiStrictModeTheme as createMuiTheme } from '@material-ui/core';

import { themeOptions } from './theme';

function isAbsoluteUrl(url) {
  return (
    url.startsWith('http://') ||
    url.startsWith('https://') ||
    url.startsWith('//') ||
    url.startsWith('data:')
  );
}

export function resolveThemeUrl(url) {
  if (!url || isAbsoluteUrl(url)) {
    return url;
  }
  let path = url;
  while (path.startsWith('/')) {
    path = path.slice(1);
  }
  return '/' + path;
}

export function extractIdpTheme(theme) {
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
}

export function applyLoginTheme(idp, common) {
  const rootStyle = document.documentElement.style;

  const {
    background,
    error,
    onBackground,
    onPrimary,
    onSecondary,
    onSurface,
    primary,
    secondary,
    surface,
  } = idp.colors;

  if (background) {
    rootStyle.setProperty('--oc-idp-background', background);
  }
  if (error) {
    rootStyle.setProperty('--oc-idp-error', error);
  }
  if (onBackground) {
    rootStyle.setProperty('--oc-idp-on-background', onBackground);
  }
  if (onPrimary) {
    rootStyle.setProperty('--oc-idp-on-primary', onPrimary);
  }
  if (onSecondary) {
    rootStyle.setProperty('--oc-idp-on-secondary', onSecondary);
  }
  if (onSurface) {
    rootStyle.setProperty('--oc-idp-on-surface', onSurface);
  }
  if (primary) {
    rootStyle.setProperty('--oc-idp-primary', primary);
  }
  if (secondary) {
    rootStyle.setProperty('--oc-idp-secondary', secondary);
  }
  if (surface) {
    rootStyle.setProperty('--oc-idp-surface', surface);
  }

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
    document.title = 'Sign in - ' + common.name;
  }
}

export function buildMuiTheme(idp) {
  const c = idp.colors || {};
  const palette = {
    ...themeOptions.palette,
  };
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
  return createMuiTheme({
    ...themeOptions,
    palette: palette,
  });
}
