// Copyright 2018-2020 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package siteacc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/opencloud-eu/reva/v2/pkg/siteacc/config"
	"github.com/opencloud-eu/reva/v2/pkg/siteacc/data"
	"github.com/opencloud-eu/reva/v2/pkg/siteacc/html"
	"github.com/opencloud-eu/reva/v2/pkg/siteacc/manager"
	"github.com/pkg/errors"
	"github.com/prometheus/alertmanager/template"
)

const (
	invokerUser = "user"
)

type methodCallback = func(*SiteAccounts, url.Values, []byte, *html.Session) (interface{}, error)
type accessSetterCallback = func(*manager.AccountsManager, *data.Account, bool) error

type endpoint struct {
	Path            string
	Handler         func(*SiteAccounts, endpoint, http.ResponseWriter, *http.Request, *html.Session)
	MethodCallbacks map[string]methodCallback
	IsPublic        bool
}

func createMethodCallbacks(cbGet methodCallback, cbPost methodCallback) map[string]methodCallback {
	callbacks := make(map[string]methodCallback)

	if cbGet != nil {
		callbacks[http.MethodGet] = cbGet
	}

	if cbPost != nil {
		callbacks[http.MethodPost] = cbPost
	}

	return callbacks
}

func getEndpoints() []endpoint {
	endpoints := []endpoint{
		// Form/panel endpoints
		{config.EndpointAdministration, callAdministrationEndpoint, nil, false},
		{config.EndpointAccount, callAccountEndpoint, nil, true},
		// General account endpoints
		{config.EndpointList, callMethodEndpoint, createMethodCallbacks(handleList, nil), false},
		{config.EndpointFind, callMethodEndpoint, createMethodCallbacks(handleFind, nil), false},
		{config.EndpointCreate, callMethodEndpoint, createMethodCallbacks(nil, handleCreate), true},
		{config.EndpointUpdate, callMethodEndpoint, createMethodCallbacks(nil, handleUpdate), false},
		{config.EndpointConfigure, callMethodEndpoint, createMethodCallbacks(nil, handleConfigure), false},
		{config.EndpointRemove, callMethodEndpoint, createMethodCallbacks(nil, handleRemove), false},
		// Site endpoints
		{config.EndpointSiteGet, callMethodEndpoint, createMethodCallbacks(handleSiteGet, nil), false},
		{config.EndpointSiteConfigure, callMethodEndpoint, createMethodCallbacks(nil, handleSiteConfigure), false},
		// Login endpoints
		{config.EndpointLogin, callMethodEndpoint, createMethodCallbacks(nil, handleLogin), true},
		{config.EndpointLogout, callMethodEndpoint, createMethodCallbacks(handleLogout, nil), true},
		{config.EndpointResetPassword, callMethodEndpoint, createMethodCallbacks(nil, handleResetPassword), true},
		{config.EndpointContact, callMethodEndpoint, createMethodCallbacks(nil, handleContact), true},
		// Authentication endpoints
		{config.EndpointVerifyUserToken, callMethodEndpoint, createMethodCallbacks(handleVerifyUserToken, nil), true},
		// Access management endpoints
		{config.EndpointGrantSiteAccess, callMethodEndpoint, createMethodCallbacks(nil, handleGrantSiteAccess), false},
		{config.EndpointGrantGOCDBAccess, callMethodEndpoint, createMethodCallbacks(nil, handleGrantGOCDBAccess), false},
		// Alerting endpoints
		{config.EndpointDispatchAlert, callMethodEndpoint, createMethodCallbacks(nil, handleDispatchAlert), false},
	}

	return endpoints
}

func callAdministrationEndpoint(siteacc *SiteAccounts, ep endpoint, w http.ResponseWriter, r *http.Request, session *html.Session) {
	if err := siteacc.ShowAdministrationPanel(w, r, session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Unable to show the administration panel: %v", err)
	}
}

func callAccountEndpoint(siteacc *SiteAccounts, ep endpoint, w http.ResponseWriter, r *http.Request, session *html.Session) {
	if err := siteacc.ShowAccountPanel(w, r, session); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Unable to show the account panel: %v", err)
	}
}

func callMethodEndpoint(siteacc *SiteAccounts, ep endpoint, w http.ResponseWriter, r *http.Request, session *html.Session) {
	// Every request to the accounts service results in a standardized JSON response
	type Response struct {
		Success bool        `json:"success"`
		Error   string      `json:"error,omitempty"`
		Data    interface{} `json:"data,omitempty"`
	}

	// The default response is an unknown requestHandler (for the specified method)
	resp := Response{
		Success: false,
		Error:   fmt.Sprintf("unknown endpoint %v for method %v", r.URL.Path, r.Method),
		Data:    nil,
	}

	if ep.MethodCallbacks != nil {
		// Search for a matching method in the list of callbacks
		for method, cb := range ep.MethodCallbacks {
			if method == r.Method {
				body, _ := io.ReadAll(r.Body)

				if respData, err := cb(siteacc, r.URL.Query(), body, session); err == nil {
					resp.Success = true
					resp.Error = ""
					resp.Data = respData
				} else {
					resp.Success = false
					resp.Error = fmt.Sprintf("%v", err)
					resp.Data = nil
				}
			}
		}
	}

	// Any failure during query handling results in a bad request
	if !resp.Success {
		w.WriteHeader(http.StatusBadRequest)
	}

	// Responses here are always JSON
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	jsonData, _ := json.MarshalIndent(&resp, "", "\t")
	_, _ = w.Write(jsonData)
}

func handleList(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	return siteacc.AccountsManager().CloneAccounts(true), nil
}

func handleFind(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	account, err := findAccount(siteacc, values.Get("by"), values.Get("value"))
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"account": account.Clone(true)}, nil
}

func handleCreate(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	account, err := unmarshalRequestData(body)
	if err != nil {
		return nil, err
	}

	// Create a new account through the accounts manager
	if err := siteacc.AccountsManager().CreateAccount(account); err != nil {
		return nil, errors.Wrap(err, "unable to create account")
	}

	return nil, nil
}

func handleUpdate(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	account, err := unmarshalRequestData(body)
	if err != nil {
		return nil, err
	}

	email, setPassword, err := processInvoker(siteacc, values, session)
	if err != nil {
		return nil, err
	}
	account.Email = email

	// Update the account through the accounts manager
	if err := siteacc.AccountsManager().UpdateAccount(account, setPassword, false); err != nil {
		return nil, errors.Wrap(err, "unable to update account")
	}

	return nil, nil
}

func handleConfigure(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	account, err := unmarshalRequestData(body)
	if err != nil {
		return nil, err
	}

	email, _, err := processInvoker(siteacc, values, session)
	if err != nil {
		return nil, err
	}
	account.Email = email

	// Configure the account through the accounts manager
	if err := siteacc.AccountsManager().ConfigureAccount(account); err != nil {
		return nil, errors.Wrap(err, "unable to configure account")
	}

	return nil, nil
}

func handleRemove(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	account, err := unmarshalRequestData(body)
	if err != nil {
		return nil, err
	}

	// Remove the account through the accounts manager
	if err := siteacc.AccountsManager().RemoveAccount(account); err != nil {
		return nil, errors.Wrap(err, "unable to remove account")
	}

	return nil, nil
}

func handleSiteGet(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	siteID := values.Get("site")
	if siteID == "" {
		return nil, errors.Errorf("no site specified")
	}
	site := siteacc.SitesManager().FindSite(siteID)
	if site == nil {
		return nil, errors.Errorf("no site with ID %v exists", siteID)
	}
	return map[string]interface{}{"site": site.Clone(false)}, nil
}

func handleSiteConfigure(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	email, _, err := processInvoker(siteacc, values, session)
	if err != nil {
		return nil, err
	}
	account, err := siteacc.AccountsManager().FindAccount(manager.FindByEmail, email)
	if err != nil {
		return nil, err
	}

	siteData := &data.Site{}
	if err := json.Unmarshal(body, siteData); err != nil {
		return nil, errors.Wrap(err, "invalid form data")
	}
	siteData.ID = account.Site

	// Configure the site through the sites manager
	if err := siteacc.SitesManager().UpdateSite(siteData); err != nil {
		return nil, errors.Wrap(err, "unable to configure site")
	}

	return nil, nil
}

func handleLogin(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	account, err := unmarshalRequestData(body)
	if err != nil {
		return nil, err
	}

	// Login the user through the users manager
	token, err := siteacc.UsersManager().LoginUser(account.Email, account.Password.Value, values.Get("scope"), session)
	if err != nil {
		return nil, errors.Wrap(err, "unable to login user")
	}

	return token, nil
}

func handleLogout(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	// Logout the user through the users manager
	siteacc.UsersManager().LogoutUser(session)
	return nil, nil
}

func handleResetPassword(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	account, err := unmarshalRequestData(body)
	if err != nil {
		return nil, err
	}

	// Reset the password through the users manager
	if err := siteacc.AccountsManager().ResetPassword(account.Email); err != nil {
		return nil, errors.Wrap(err, "unable to reset password")
	}

	return nil, nil
}

func handleContact(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	if !session.IsUserLoggedIn() {
		return nil, errors.Errorf("no user is currently logged in")
	}

	type jsonData struct {
		Subject string `json:"subject"`
		Message string `json:"message"`
	}
	contactData := &jsonData{}
	if err := json.Unmarshal(body, contactData); err != nil {
		return nil, errors.Wrap(err, "invalid form data")
	}

	// Send an email through the accounts manager
	siteacc.AccountsManager().SendContactForm(session.LoggedInUser().Account, strings.TrimSpace(contactData.Subject), strings.TrimSpace(contactData.Message))
	return nil, nil
}

func handleVerifyUserToken(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	token := values.Get("token")
	if token == "" {
		return nil, errors.Errorf("no token specified")
	}

	user := values.Get("user")
	if user == "" {
		return nil, errors.Errorf("no user specified")
	}

	// Verify the user token using the users manager
	newToken, err := siteacc.UsersManager().VerifyUserToken(token, user, values.Get("scope"))
	if err != nil {
		return nil, errors.Wrap(err, "token verification failed")
	}

	return newToken, nil
}

func handleDispatchAlert(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	alertsData := &template.Data{}
	if err := json.Unmarshal(body, alertsData); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal the alerts data")
	}

	// Dispatch the alerts using the alerts dispatcher
	if err := siteacc.AlertsDispatcher().DispatchAlerts(alertsData, siteacc.AccountsManager().CloneAccounts(true)); err != nil {
		return nil, errors.Wrap(err, "error while dispatching the alerts")
	}

	return nil, nil
}

func handleGrantSiteAccess(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	return handleGrantAccess((*manager.AccountsManager).GrantSiteAccess, siteacc, values, body, session)
}

func handleGrantGOCDBAccess(siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	return handleGrantAccess((*manager.AccountsManager).GrantGOCDBAccess, siteacc, values, body, session)
}

func handleGrantAccess(accessSetter accessSetterCallback, siteacc *SiteAccounts, values url.Values, body []byte, session *html.Session) (interface{}, error) {
	account, err := unmarshalRequestData(body)
	if err != nil {
		return nil, err
	}

	if val := values.Get("status"); len(val) > 0 {
		var grantAccess bool
		switch strings.ToLower(val) {
		case "true":
			grantAccess = true

		case "false":
			grantAccess = false

		default:
			return nil, errors.Errorf("unsupported access status %v", val[0])
		}

		// Grant access to the account through the accounts manager
		if err := accessSetter(siteacc.AccountsManager(), account, grantAccess); err != nil {
			return nil, errors.Wrap(err, "unable to change the access status of the account")
		}
	} else {
		return nil, errors.Errorf("no access status provided")
	}

	return nil, nil
}

func unmarshalRequestData(body []byte) (*data.Account, error) {
	account := &data.Account{}
	if err := json.Unmarshal(body, account); err != nil {
		return nil, errors.Wrap(err, "invalid account data")
	}
	account.Cleanup()
	return account, nil
}

func findAccount(siteacc *SiteAccounts, by string, value string) (*data.Account, error) {
	if len(by) == 0 && len(value) == 0 {
		return nil, errors.Errorf("missing search criteria")
	}

	// Find the account using the accounts manager
	account, err := siteacc.AccountsManager().FindAccount(by, value)
	if err != nil {
		return nil, errors.Wrap(err, "user not found")
	}
	return account, nil
}

func processInvoker(siteacc *SiteAccounts, values url.Values, session *html.Session) (string, bool, error) {
	var email string
	var invokedByUser bool

	switch strings.ToLower(values.Get("invoker")) {
	case invokerUser:
		// If this endpoint was called by the user, set the account email from the stored session
		if !session.IsUserLoggedIn() {
			return "", false, errors.Errorf("no user is currently logged in")
		}

		email = session.LoggedInUser().Account.Email
		invokedByUser = true

	default:
		return "", false, errors.Errorf("no invoker provided")
	}

	return email, invokedByUser, nil
}
