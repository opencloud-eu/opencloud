import React, { useEffect, useMemo } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";

import { useTranslation } from "react-i18next";

import { withStyles } from "@material-ui/core/styles";
import Button from "@material-ui/core/Button";
import CircularProgress from "@material-ui/core/CircularProgress";
import green from "@material-ui/core/colors/green";
import Typography from "@material-ui/core/Typography";
import Link from "@material-ui/core/Link";

import TextInput from "../../components/TextInput";

import {
  updateInput,
  executeLogonIfFormValid,
  advanceLogonFlow,
} from "../../actions/login";
import { ErrorMessage } from "../../errors";

const styles = (theme) => ({
  buttonProgress: {
    color: green[500],
    position: "absolute",
    top: "50%",
    left: "50%",
    marginTop: -12,
    marginLeft: -12,
  },
  main: {
    width: "100%",
  },
  header: {
    textAlign: "center",
    marginTop: 0,
    marginBottom: theme.spacing(6),
  },
  wrapper: {
    position: "relative",
    width: "100%",
    textAlign: "center",
  },
  message: {
    marginTop: 5,
    marginBottom: 5,
  },
  signinPageText: {
    marginBottom: 12
  }
});

function Login(props) {
  const {
    hello,
    query,
    dispatch,
    history,
    loading,
    errors,
    classes,
    username,
    password,
    passwordResetLink,
    branding,
  } = props;

  const { t } = useTranslation();
  const loginFailed = errors.http;
  const hasError = errors.http || errors.username || errors.password;
  const errorMessage = errors.http ? (
    <ErrorMessage error={errors.http}></ErrorMessage>
  ) : errors.username ? (
    <ErrorMessage error={errors.username}></ErrorMessage>
  ) : (
    <ErrorMessage error={errors.password}></ErrorMessage>
  );
  const extraPropsUsername = {
    "aria-invalid": errors.username || errors.http ? "true" : "false",
  };
  const extraPropsPassword = {
    "aria-invalid": errors.password || errors.http ? "true" : "false",
  };

  if (errors.username || errors.http) {
    extraPropsUsername["extraClassName"] = "error";
    extraPropsUsername["aria-describedby"] = "oc-login-error-message";
  }

  if (errors.password || errors.http) {
    extraPropsPassword["extraClassName"] = "error";
    extraPropsPassword["aria-describedby"] = "oc-login-error-message";
  }

  useEffect(() => {
    if (hello && hello.state && history.action !== "PUSH") {
      if (!query.prompt || query.prompt.indexOf("select_account") === -1) {
        dispatch(advanceLogonFlow(true, history));
        return;
      }

      history.replace(
        `/chooseaccount${history.location.search}${history.location.hash}`
      );
      return;
    }
  });

  const handleChange = (name) => (event) => {
    dispatch(updateInput(name, event.target.value));
  };

  const handleNextClick = (event) => {
    event.preventDefault();

    dispatch(executeLogonIfFormValid(username, password, false)).then(
      (response) => {
        if (response.success) {
          dispatch(advanceLogonFlow(response.success, history));
        }
      }
    );
  };

  return (
    <div className={classes.main}>
      <h1 className={classes.header}>
        {" "}
        {t("konnect.login.headline", "Sign in")}
      </h1>
      {branding?.signinPageText && (
        <Typography
          className={classes.signinPageText}
          variant="body2"
          dangerouslySetInnerHTML={{ __html: branding.signinPageText }}
        />
      )}
      <form
        action=""
        className="oc-login-form"
        onSubmit={(event) => handleNextClick(event)}
      >
        <TextInput
          autoFocus
          autoCapitalize="off"
          spellCheck="false"
          value={username}
          onChange={handleChange("username")}
          autoComplete="username"
          placeholder={t("konnect.login.usernameField.label", "Username")}
          label={t("konnect.login.usernameField.label", "Username")}
          id="oc-login-username"
          {...extraPropsUsername}
        />
        <TextInput
          type="password"
          margin="normal"
          onChange={handleChange("password")}
          autoComplete="current-password"
          placeholder={t("konnect.login.passwordField.label", "Password")}
          label={t("konnect.login.passwordField.label", "Password")}
          id="oc-login-password"
          {...extraPropsPassword}
        />
        {hasError && (
          <Typography
            id="oc-login-error-message"
            variant="subtitle2"
            component="span"
            color="error"
            className={classes.message}
          >
            {errorMessage}
          </Typography>
        )}
        <div className={classes.wrapper}>
          {loginFailed && passwordResetLink && (
            <Link
              id="oc-login-password-reset"
              href={passwordResetLink}
              variant="subtitle2"
            >
              {"Reset password?"}
            </Link>
          )}
          <Button
            type="submit"
            color="primary"
            variant="contained"
            className="oc-button-primary oc-mt-l"
            disabled={!!loading}
            onClick={handleNextClick}
          >
            {t("konnect.login.nextButton.label", "Log in")}
          </Button>
          {loading && (
            <CircularProgress size={24} className={classes.buttonProgress} />
          )}
        </div>
      </form>
    </div>
  );
}

Login.propTypes = {
  classes: PropTypes.object.isRequired,

  loading: PropTypes.string.isRequired,
  username: PropTypes.string.isRequired,
  password: PropTypes.string.isRequired,
  passwordResetLink: PropTypes.string.isRequired,
  errors: PropTypes.object.isRequired,
  branding: PropTypes.object,
  hello: PropTypes.object,
  query: PropTypes.object.isRequired,

  dispatch: PropTypes.func.isRequired,
  history: PropTypes.object.isRequired,
};

const mapStateToProps = (state) => {
  const { loading, username, password, errors } = state.login;
  const { branding, hello, query, passwordResetLink } = state.common;

  return {
    loading,
    username,
    password,
    errors,
    branding,
    hello,
    query,
    passwordResetLink,
  };
};

export default connect(mapStateToProps)(withStyles(styles)(Login));
