import { withTranslation } from 'react-i18next';
import PropTypes from 'prop-types';

export const ERROR_LOGIN_VALIDATE_MISSINGUSERNAME = 'konnect.error.login.validate.missingUsername';
export const ERROR_LOGIN_VALIDATE_MISSINGPASSWORD = 'konnect.error.login.validate.missingPassword';
export const ERROR_LOGIN_FAILED = 'konnect.error.login.failed';
export const ERROR_LOGIN_MFA_INVALID = 'opencloud.error.login.mfa.invalid';
export const ERROR_LOGIN_MFA_LOCKED = 'opencloud.error.login.mfa.locked';
export const ERROR_HTTP_NETWORK_ERROR = 'konnect.error.http.networkError';
export const ERROR_HTTP_UNEXPECTED_RESPONSE_STATUS = 'konnect.error.http.unexpectedResponseStatus';
export const ERROR_HTTP_UNEXPECTED_RESPONSE_STATE = 'konnect.error.http.unexpectedResponseState';

// Error with values.
export class ExtendedError extends Error {
  values = undefined;

  constructor(message, values) {
    super(message);
    if (Error.captureStackTrace !== undefined) {
      Error.captureStackTrace(this, ExtendedError);
    }
    this.values = values;
  }
}

// Component to translate error text with values.
function ErrorMessageComponent(props) {
  const { error, t, values } = props;

  if (!error) {
    return null;
  }

  const id = error.id ? error.id : error.message;
  const messageDescriptor = Object.assign({}, {
    id,
    defaultMessage: error.id ? error.message : undefined,
    values: {
      ...error.values,
      ...values,
    },
  });

  switch (messageDescriptor.id) {
    case ERROR_LOGIN_VALIDATE_MISSINGUSERNAME:
      return t("konnect.error.login.validate.missingUsername", "Enter a valid value.", messageDescriptor.values);
    case ERROR_LOGIN_VALIDATE_MISSINGPASSWORD:
      return t("konnect.error.login.validate.missingPassword", "Enter your password.");
    case ERROR_LOGIN_FAILED:
      return t("konnect.error.login.failed", "Logon failed. Please verify your credentials and try again.");
    case ERROR_LOGIN_MFA_INVALID:
      return t("opencloud.error.login.mfa.invalid", "The code is not correct. Please try again.");
    case ERROR_LOGIN_MFA_LOCKED:
      return t("opencloud.error.login.mfa.locked", "Too many wrong codes. Please wait and try again later.");
    case ERROR_HTTP_NETWORK_ERROR:
      return t("konnect.error.http.networkError", "Network error. Please check your connection and try again.");
    case ERROR_HTTP_UNEXPECTED_RESPONSE_STATUS:
      return t("konnect.error.http.unexpectedResponseStatus", "Unexpected HTTP response: {{status}}. Please check your connection and try again.", messageDescriptor.values);
    case ERROR_HTTP_UNEXPECTED_RESPONSE_STATE:
      return t("konnect.error.http.unexpectedResponseState", "Unexpected response state: {{state}}", messageDescriptor.values);
    default:
  }

  const f = t;
  return f(messageDescriptor.defaultMessage, messageDescriptor.values);
}

ErrorMessageComponent.propTypes = {
  error: PropTypes.object,
  t: PropTypes.func,
  values: PropTypes.any,
};
export const ErrorMessage = withTranslation()(ErrorMessageComponent);
