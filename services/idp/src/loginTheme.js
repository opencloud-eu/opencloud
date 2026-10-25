/**
 * Consumes the login relevant subset of the theme.json served by the web
 * service (see services/web/pkg/theme) and applies it to the login UI.
 */

import { unstable_createMuiStrictModeTheme as createMuiTheme } from '@material-ui/core';

import { themeOptions } from './theme';

const IDP_CSS_PREFIX = '--oc-idp-';

function toCssProperty(key) {
  let name = '';
  for (let i = 0; i < key.length; i++) {
    const ch = key.charAt(i);
    if (ch >= 'A' && ch <= 'Z') {
      name += '-' + ch.toLowerCase();
    } else {
      name += ch;
    }
  }
  return IDP_CSS_PREFIX + name;
}

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

  for (const key in idp.colors) {
    const value = idp.colors[key];
    if (value) {
      rootStyle.setProperty(toCssProperty(key), value);
    }
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
  const palette = Object.assign({}, themeOptions.palette);
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
  return createMuiTheme(Object.assign({}, themeOptions, { palette: palette }));
}
