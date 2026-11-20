import React from "react";
import PropTypes from "prop-types";

import { useTranslation } from "react-i18next";

import Button from "@material-ui/core/Button";
import CircularProgress from "@material-ui/core/CircularProgress";
import Typography from "@material-ui/core/Typography";
import Link from "@material-ui/core/Link";

import TextInput from "../../components/TextInput";
import { ErrorMessage } from "../../errors";

// The second step of the login for users with two-factor authentication: the password
// was correct, now a one-time code or a recovery code is required. The login page keeps
// the code in its state, because it is sent together with the username and password.
function MfaStep({ classes, code, error, loading, onChange, onSubmit, onBack }) {
  const { t } = useTranslation();

  const errorProps = error
    ? { "aria-invalid": "true", extraClassName: "error", "aria-describedby": "oc-login-error-message" }
    : { "aria-invalid": "false" };

  return (
    <div className={classes.main}>
      <h1 className={classes.header}>
        {t("opencloud.login.mfa.headline", "Two-factor authentication")}
      </h1>
      <Typography className={classes.mfaText} variant="body2">
        {t(
          "opencloud.login.mfa.message",
          "Enter the code from your authenticator app or one of your recovery codes."
        )}
      </Typography>
      <form action="" className="oc-login-form" onSubmit={onSubmit}>
        <TextInput
          autoFocus
          autoCapitalize="off"
          autoCorrect="off"
          spellCheck="false"
          value={code}
          onChange={onChange}
          autoComplete="one-time-code"
          placeholder={t("opencloud.login.mfa.codeField.label", "Code")}
          label={t("opencloud.login.mfa.codeField.label", "Code")}
          id="oc-login-mfa-code"
          {...errorProps}
        />
        {error && (
          <Typography
            id="oc-login-error-message"
            variant="subtitle2"
            component="span"
            color="error"
            className={classes.message}
          >
            <ErrorMessage error={error}></ErrorMessage>
          </Typography>
        )}
        <div className={classes.wrapper}>
          <Button
            type="submit"
            color="primary"
            variant="contained"
            className="oc-button-primary oc-mt-l"
            disabled={!!loading || !code}
            onClick={onSubmit}
          >
            {t("opencloud.login.mfa.verifyButton.label", "Verify")}
          </Button>
          {loading && <CircularProgress size={24} className={classes.buttonProgress} />}
          <Link
            id="oc-login-mfa-back"
            href="#"
            variant="subtitle2"
            className={classes.mfaBack}
            onClick={onBack}
          >
            {t("opencloud.login.mfa.backButton.label", "Back to login")}
          </Link>
        </div>
      </form>
    </div>
  );
}

MfaStep.propTypes = {
  classes: PropTypes.object.isRequired,
  code: PropTypes.string.isRequired,
  error: PropTypes.object,
  loading: PropTypes.string.isRequired,
  onChange: PropTypes.func.isRequired,
  onSubmit: PropTypes.func.isRequired,
  onBack: PropTypes.func.isRequired,
};

export default MfaStep;
