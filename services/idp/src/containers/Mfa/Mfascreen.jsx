import { useCallback, useEffect, useState } from "react";
import PropTypes from "prop-types";
import axios from "axios";
import { QRCodeSVG } from "qrcode.react";

import { useTranslation } from "react-i18next";

import { withStyles } from "@material-ui/core/styles";
import Button from "@material-ui/core/Button";
import Typography from "@material-ui/core/Typography";
import Link from "@material-ui/core/Link";

import ResponsiveScreen from "../../components/ResponsiveScreen";
import TextInput from "../../components/TextInput";
import { MFARoute } from "../../actions/login";

function styles(theme) {
  return {
    main: {
      width: "100%",
    },
    header: {
      textAlign: "center",
      marginTop: 0,
      marginBottom: theme.spacing(4),
    },
    text: {
      marginBottom: theme.spacing(3),
    },
    status: {
      fontWeight: 700,
      marginBottom: theme.spacing(1),
    },
    message: {
      display: "block",
      marginTop: 5,
      marginBottom: 5,
    },
    qr: {
      display: "flex",
      justifyContent: "center",
      marginBottom: theme.spacing(3),
    },
    secret: {
      fontFamily: "monospace",
      fontSize: "1rem",
      wordBreak: "break-all",
      userSelect: "all",
    },
    codes: {
      display: "grid",
      gridTemplateColumns: "1fr 1fr",
      gap: theme.spacing(1),
      listStyle: "none",
      padding: 0,
      margin: 0,
      marginBottom: theme.spacing(3),
      fontFamily: "monospace",
      fontSize: "1.0625rem",
      textAlign: "center",
    },
    footer: {
      display: "block",
      textAlign: "center",
      marginTop: theme.spacing(3),
    },
  };
}

// Views of the page.
const VIEW_LOADING = "loading";
const VIEW_DISABLED = "disabled";
const VIEW_SETUP = "setup";
const VIEW_RECOVERY_CODES = "recoveryCodes";
const VIEW_ENABLED = "enabled";
const VIEW_UNAVAILABLE = "unavailable";

function request(method, path, data) {
  return axios.request({
    method,
    url: `./identifier/_/mfa/${path}`,
    data,
    headers: {
      "Kopano-Konnect-XSRF": "1",
    },
    // all responses are handled by the caller
    validateStatus: () => true,
  });
}

function Mfascreen(props) {
  const { classes, history } = props;
  const { t } = useTranslation();

  const [view, setView] = useState(VIEW_LOADING);
  const [status, setStatus] = useState(null);
  const [setup, setSetup] = useState(null);
  const [recoveryCodes, setRecoveryCodes] = useState([]);
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  function errorMessage(response) {
    switch (response.data && response.data.error) {
      case "invalid_password":
        return t("opencloud.mfa.error.invalidPassword", "The password is not correct.");
      case "invalid_code":
        return t("opencloud.mfa.error.invalidCode", "The code is not correct. Please try again.");
      case "locked":
        return t(
          "opencloud.mfa.error.locked",
          "Too many wrong codes. Please wait a few minutes and try again."
        );
      case "no_pending_setup":
        return t("opencloud.mfa.error.setupExpired", "The setup has expired. Please start again.");
      default:
        return t("opencloud.mfa.error.unexpected", "Something went wrong. Please try again.");
    }
  }

  const signIn = useCallback(() => {
    // the login page brings the user back to this page afterwards
    history.replace("/identifier", { returnTo: MFARoute });
  }, [history]);

  const loadStatus = useCallback(() => {
    return request("get", "status").then((response) => {
      switch (response.status) {
        case 200:
          setStatus(response.data);
          setView(response.data.enabled ? VIEW_ENABLED : VIEW_DISABLED);
          break;
        case 401:
          signIn();
          break;
        default:
          setView(VIEW_UNAVAILABLE);
      }
    }).catch(() => {
      setView(VIEW_UNAVAILABLE);
    });
  }, [signIn]);

  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  // Runs a request and handles what all requests have in common.
  function submit(event, method, path, data, onSuccess) {
    event.preventDefault();
    if (busy) {
      return;
    }

    setBusy(true);
    setError("");
    request(method, path, data).then((response) => {
      if (response.status === 401) {
        signIn();
        return;
      }
      if (response.status >= 200 && response.status < 300) {
        setPassword("");
        setCode("");
        onSuccess(response);
        return;
      }
      setError(errorMessage(response));
    }).catch(() => {
      setError(t("opencloud.mfa.error.unexpected", "Something went wrong. Please try again."));
    }).finally(() => {
      setBusy(false);
    });
  }

  function handleSetup(event) {
    submit(event, "post", "setup", { password }, (response) => {
      setSetup(response.data);
      setView(VIEW_SETUP);
    });
  }

  function handleEnable(event) {
    submit(event, "post", "enable", { code }, (response) => {
      // state updates after a request are not batched: switch the view before
      // dropping the data the current view renders
      setRecoveryCodes(response.data.recoveryCodes);
      setView(VIEW_RECOVERY_CODES);
      setSetup(null);
    });
  }

  function handleRecoveryCodes(event) {
    submit(event, "post", "recovery-codes", { code }, (response) => {
      setRecoveryCodes(response.data.recoveryCodes);
      setView(VIEW_RECOVERY_CODES);
    });
  }

  function handleDisable(event) {
    submit(event, "post", "disable", { code }, () => {
      setView(VIEW_LOADING);
      loadStatus();
    });
  }

  function handleDone(event) {
    event.preventDefault();
    setRecoveryCodes([]);
    setView(VIEW_LOADING);
    loadStatus();
  }

  function handleCancelSetup(event) {
    event.preventDefault();
    setSetup(null);
    setCode("");
    setError("");
    setView(VIEW_DISABLED);
  }

  const errorElement = error && (
    <Typography
      id="oc-mfa-error-message"
      variant="subtitle2"
      component="span"
      color="error"
      role="alert"
      className={classes.message}
    >
      {error}
    </Typography>
  );

  const codeInput = (
    <TextInput
      autoFocus
      autoCapitalize="off"
      autoCorrect="off"
      spellCheck="false"
      value={code}
      onChange={(event) => setCode(event.target.value)}
      autoComplete="one-time-code"
      placeholder={t("opencloud.mfa.codeField.label", "Code")}
      label={t("opencloud.mfa.codeField.label", "Code")}
      id="oc-mfa-code"
      extraClassName={error ? "error" : ""}
      aria-invalid={error ? "true" : "false"}
    />
  );

  let content = null;
  switch (view) {
    case VIEW_DISABLED:
      content = (
        <form action="" className="oc-login-form" onSubmit={handleSetup}>
          <Typography className={classes.status} id="oc-mfa-status">
            {t("opencloud.mfa.disabled.status", "Two-factor authentication is off.")}
          </Typography>
          <Typography className={classes.text} variant="body2">
            {t(
              "opencloud.mfa.disabled.message",
              "Protect your account with a second step. After your password you enter a code from an authenticator app on your phone, for example Google Authenticator, Microsoft Authenticator or FreeOTP. Enter your password to start."
            )}
          </Typography>
          <TextInput
            autoFocus
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
            placeholder={t("opencloud.mfa.passwordField.label", "Password")}
            label={t("opencloud.mfa.passwordField.label", "Password")}
            id="oc-mfa-password"
            extraClassName={error ? "error" : ""}
            aria-invalid={error ? "true" : "false"}
          />
          {errorElement}
          <Button
            type="submit"
            color="primary"
            variant="contained"
            className="oc-button-primary oc-mt-l"
            id="oc-mfa-setup"
            disabled={busy || !password}
          >
            {t("opencloud.mfa.setupButton.label", "Set up")}
          </Button>
        </form>
      );
      break;

    case VIEW_SETUP:
      content = setup && (
        <form action="" className="oc-login-form" onSubmit={handleEnable}>
          <Typography className={classes.text} variant="body2">
            {t(
              "opencloud.mfa.setup.scan",
              "Scan this QR code with your authenticator app. Then enter the 6-digit code the app shows."
            )}
          </Typography>
          <div className={classes.qr}>
            <QRCodeSVG
              value={setup.uri}
              size={200}
              marginSize={2}
              title={t("opencloud.mfa.setup.qrTitle", "QR code for your authenticator app")}
              id="oc-mfa-qr"
            />
          </div>
          <Typography className={classes.text} variant="body2">
            {t("opencloud.mfa.setup.manual", "Cannot scan? Enter this key in the app:")}{" "}
            <span className={classes.secret} id="oc-mfa-secret">{setup.secret}</span>
          </Typography>
          {codeInput}
          {errorElement}
          <Button
            type="submit"
            color="primary"
            variant="contained"
            className="oc-button-primary oc-mt-l"
            id="oc-mfa-enable"
            disabled={busy || !code}
          >
            {t("opencloud.mfa.enableButton.label", "Turn on")}
          </Button>
          <Button
            variant="contained"
            className="oc-button-secondary oc-mt-l"
            id="oc-mfa-cancel"
            onClick={handleCancelSetup}
          >
            {t("opencloud.mfa.cancelButton.label", "Cancel")}
          </Button>
        </form>
      );
      break;

    case VIEW_RECOVERY_CODES:
      content = (
        <div>
          <Typography className={classes.status}>
            {t("opencloud.mfa.recoveryCodes.headline", "Save your recovery codes")}
          </Typography>
          <Typography className={classes.text} variant="body2">
            {t(
              "opencloud.mfa.recoveryCodes.message",
              "Each code works once instead of a code from your app, for example when you lose your phone. Keep them in a safe place. They are not shown again."
            )}
          </Typography>
          <ul className={classes.codes} id="oc-mfa-recovery-codes">
            {recoveryCodes.map((recoveryCode) => (
              <li key={recoveryCode}>{recoveryCode}</li>
            ))}
          </ul>
          <Button
            color="primary"
            variant="contained"
            className="oc-button-primary"
            id="oc-mfa-done"
            onClick={handleDone}
          >
            {t("opencloud.mfa.doneButton.label", "I saved the codes")}
          </Button>
        </div>
      );
      break;

    case VIEW_ENABLED:
      content = status && (
        <form action="" className="oc-login-form" onSubmit={handleDisable}>
          <Typography className={classes.status} id="oc-mfa-status">
            {t("opencloud.mfa.enabled.status", "Two-factor authentication is on.")}
          </Typography>
          <Typography className={classes.text} variant="body2">
            {t(
              "opencloud.mfa.enabled.recoveryCodesLeft",
              "Unused recovery codes: {{count}}",
              { count: status.recoveryCodesLeft }
            )}
          </Typography>
          <Typography className={classes.text} variant="body2">
            {t(
              "opencloud.mfa.enabled.message",
              "To turn it off or to get new recovery codes, enter a code from your authenticator app or a recovery code."
            )}
          </Typography>
          {codeInput}
          {errorElement}
          <Button
            variant="contained"
            className="oc-button-secondary oc-mt-l"
            id="oc-mfa-new-recovery-codes"
            disabled={busy || !code}
            onClick={handleRecoveryCodes}
          >
            {t("opencloud.mfa.recoveryCodesButton.label", "New recovery codes")}
          </Button>
          <Button
            type="submit"
            variant="contained"
            className="oc-button-secondary oc-mt-l"
            id="oc-mfa-disable"
            disabled={busy || !code}
          >
            {t("opencloud.mfa.disableButton.label", "Turn off")}
          </Button>
        </form>
      );
      break;

    case VIEW_UNAVAILABLE:
      content = (
        <Typography className={classes.text} variant="body2" id="oc-mfa-status">
          {t(
            "opencloud.mfa.unavailable",
            "Two-factor authentication is not available at the moment. Please try again later."
          )}
        </Typography>
      );
      break;

    default:
  }

  return (
    // nowrap: this page can be taller than the window, the footer must stay below it
    <ResponsiveScreen loading={view === VIEW_LOADING} withoutPadding wrap="nowrap">
      <div className={classes.main}>
        <h1 className={classes.header}>
          {t("opencloud.mfa.headline", "Two-factor authentication")}
        </h1>
        {status && status.username && view !== VIEW_UNAVAILABLE && (
          <Typography className={classes.text} variant="body2" id="oc-mfa-username">
            {t("opencloud.mfa.account", "Account: {{username}}", { username: status.username })}
          </Typography>
        )}
        {content}
        <Link href="/account" variant="subtitle2" className={classes.footer} id="oc-mfa-back">
          {t("opencloud.mfa.backLink.label", "Back to your account")}
        </Link>
      </div>
    </ResponsiveScreen>
  );
}

Mfascreen.propTypes = {
  classes: PropTypes.object.isRequired,
  history: PropTypes.object.isRequired,
};

export default withStyles(styles)(Mfascreen);
