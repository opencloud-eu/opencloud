import axios from 'axios';
import queryString from 'query-string';

import { newHelloRequest } from '../models/hello';
import { withClientRequestState } from '../utils';
import {
  ExtendedError,
  ERROR_LOGIN_VALIDATE_MISSINGUSERNAME,
  ERROR_LOGIN_VALIDATE_MISSINGPASSWORD,
  ERROR_LOGIN_FAILED,
  ERROR_LOGIN_MFA_INVALID,
  ERROR_LOGIN_MFA_LOCKED,
  ERROR_HTTP_UNEXPECTED_RESPONSE_STATUS,
  ERROR_HTTP_UNEXPECTED_RESPONSE_STATE
} from '../errors';

import * as types from './types';
import { receiveHello } from './common';
import { handleAxiosError } from './utils';

// Modes for logon.
export const ModeLogonUsernameEmptyPasswordCookie = '0';
export const ModeLogonUsernamePassword = '1';

// Values of the response header 'opencloud-mfa' of a rejected logon. It tells if and
// why the second factor (two-factor authentication) of the user rejected it: 'required'
// when no code was sent, or one of these.
const MFAResultInvalid = 'invalid';
const MFAResultLocked = 'locked';

// mfaAttempt is the result of a logon whose password was correct, but whose second
// factor is still missing or was not accepted.
function mfaAttempt(state, result) {
  const errors = {};
  if (result === MFAResultInvalid) {
    errors.code = new Error(ERROR_LOGIN_MFA_INVALID);
  } else if (result === MFAResultLocked) {
    errors.code = new Error(ERROR_LOGIN_MFA_LOCKED);
  }
  return { success: false, state, mfaRequired: true, errors };
}

// Route of the page to manage the two-factor authentication.
export const MFARoute = '/mfa';

export function updateInput(name, value) {
  return {
    type: types.UPDATE_INPUT,
    name,
    value
  };
}

export function receiveValidateLogon(errors) {
  return {
    type: types.RECEIVE_VALIDATE_LOGON,
    errors
  };
}

export function requestLogon(username, password) {
  return {
    type: types.REQUEST_LOGON,
    username,
    password
  };
}

export function receiveLogon(logon) {
  const { success, errors, mfaRequired } = logon;

  return {
    type: types.RECEIVE_LOGON,
    success,
    errors,
    mfaRequired: !success && !!mfaRequired
  };
}

// Leaves the two-factor authentication step of the login.
export function resetMFA() {
  return {
    type: types.RESET_MFA
  };
}

export function requestConsent(allow=false) {
  return {
    type: allow ? types.REQUEST_CONSENT_ALLOW : types.REQUEST_CONSENT_CANCEL
  };
}

export function receiveConsent(logon) {
  const { success, errors } = logon;

  return {
    type: types.RECEIVE_CONSENT,
    success,
    errors
  };
}

export function executeLogon(username, password, mode=ModeLogonUsernamePassword, code='') {
  return function(dispatch, getState) {
    dispatch(requestLogon(username, password));
    dispatch(receiveHello({
      username
    })); // Reset any hello state on logon.

    const { flow, query } = getState().common;

    // Prepare params based on mode.
    const params = [];
    switch (mode) {
      case ModeLogonUsernamePassword:
        // Username with password.
        params.push(username, password, mode);
        if (code) {
          // One-time code or recovery code of the two-factor authentication.
          params.push(code);
        }
        break;

      case ModeLogonUsernameEmptyPasswordCookie:
        // Username with empty password - this only works when the user is already signed in.
        params.push(username, '', mode);
        break;

      // no default
    }

    const r = withClientRequestState({
      params: params,
      hello: newHelloRequest(flow, query)
    });
    return axios.post('./identifier/_/logon', r, {
      headers: {
        'Kopano-Konnect-XSRF': '1'
      }
    }).then(response => {
      switch (response.status) {
        case 200:
          // success.
          return response.data;
        case 204: {
          // login failed. The header tells when it was the second factor.
          const state = response.headers['kopano-konnect-state'];
          const mfaResult = response.headers['opencloud-mfa'];
          if (mfaResult) {
            return mfaAttempt(state, mfaResult);
          }
          return {
            success: false,
            state,
            errors: {
              http: new Error(ERROR_LOGIN_FAILED)
            }
          };
        }
        default:
          // error.
          throw new ExtendedError(ERROR_HTTP_UNEXPECTED_RESPONSE_STATUS, response);
      }
    }).then(response => {
      if (response.state !== r.state) {
        throw new ExtendedError(ERROR_HTTP_UNEXPECTED_RESPONSE_STATE, response);
      }

      let { hello } = response;
      if (!hello) {
        hello = {
          success: response.success,
          username
        };
      }
      dispatch(receiveHello(hello));
      dispatch(receiveLogon(response));
      return Promise.resolve(response);
    }).catch(error => {
      error = handleAxiosError(error);
      const errors = {
        http: error
      };

      dispatch(receiveValidateLogon(errors));
      return {
        success: false,
        errors: errors
      };
    });
  };
}

export function executeConsent(allow=false, scope='') {
  return function(dispatch, getState) {
    dispatch(requestConsent(allow));

    const { query } = getState().common;

    const r = withClientRequestState({
      allow,
      scope,
      client_id: query.client_id || '', // eslint-disable-line camelcase
      redirect_uri: query.redirect_uri || '', // eslint-disable-line camelcase
      ref: query.state || '',
      flow_nonce: query.nonce || '' // eslint-disable-line camelcase
    });
    return axios.post('./identifier/_/consent', r, {
      headers: {
        'Kopano-Konnect-XSRF': '1'
      }
    }).then(response => {
      switch (response.status) {
        case 200:
          // success.
          return response.data;
        case 204:
          // cancel reply.
          return {
            success: true,
            state: response.headers['kopano-konnect-state']
          };
        default:
          // error.
          throw new ExtendedError(ERROR_HTTP_UNEXPECTED_RESPONSE_STATUS, response);
      }
    }).then(response => {
      if (response.state !== r.state) {
        throw new ExtendedError(ERROR_HTTP_UNEXPECTED_RESPONSE_STATE, response);
      }

      dispatch(receiveConsent(response));
      return Promise.resolve(response);
    }).catch(error => {
      error = handleAxiosError(error);
      const errors = {
        http: error
      };

      dispatch(receiveValidateLogon(errors));
      return {
        success: false,
        errors: errors
      };
    });
  };
}

export function validateUsernamePassword(username, password, isSignedIn) {
  return function(dispatch) {
    return new Promise((resolve, reject) => {
      const errors = {};

      if (!username) {
        errors.username = new Error(ERROR_LOGIN_VALIDATE_MISSINGUSERNAME);
      }
      if (!password && !isSignedIn) {
        errors.password = new Error(ERROR_LOGIN_VALIDATE_MISSINGPASSWORD);
      }

      dispatch(receiveValidateLogon(errors));
      if (Object.keys(errors).length === 0) {
        resolve(errors);
      } else {
        reject(errors);
      }
    });
  };
}

export function executeLogonIfFormValid(username, password, isSignedIn, code='') {
  return (dispatch) => {
    return dispatch(
      validateUsernamePassword(username, password, isSignedIn)
    ).then(() => {
      const mode = isSignedIn ? ModeLogonUsernameEmptyPasswordCookie : ModeLogonUsernamePassword;
      return dispatch(executeLogon(username, password, mode, code));
    }).catch((errors) => {
      return {
        success: false,
        errors: errors
      };
    });
  };
}

export function advanceLogonFlow(success, history, done=false, extraQuery={}) {
  return (dispatch, getState) => {
    if (!success) {
      return;
    }

    const { flow, query, hello } = getState().common;
    const q = Object.assign({}, query, extraQuery);

    switch (flow) {
      case 'oauth':
      case 'consent':
      case 'oidc':
        if (hello.details.flow !== flow) {
          // Ignore requested flow if hello flow does not match.
          break;
        }

        if (!done && hello.details.next === 'consent') {
          history.replace(`/consent${history.location.search}${history.location.hash}`);
          return;
        }
        if (hello.details.continue_uri) {
          q.prompt = 'none';
          window.location.replace(hello.details.continue_uri + '?' + queryString.stringify(q));
          return;
        }

        break;

      default:
        // Legacy stupid modes.
        if (q.continue && q.continue.indexOf(document.location.origin) === 0) {
          window.location.replace(q.continue);
          return;
        }
    }

    // Default action.
    if (history.location.pathname === MFARoute) {
      // The flow is advanced by the logon and by the hello it triggers. The
      // first one already brought the user back, see below.
      return;
    }
    let target = '/welcome';
    if (history.location.state && history.location.state.returnTo === MFARoute) {
      // The two-factor authentication page sent the user here to sign in.
      target = MFARoute;
    }
    if (history.action === 'REPLACE') {
      target = target + history.location.search + history.location.hash;
    }

    dispatch(receiveValidateLogon({})); // XXX(longsleep): hack to reset loading and errors.
    history.push(target);
  };
}
