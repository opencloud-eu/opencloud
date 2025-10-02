package groupware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"

	"github.com/opencloud-eu/opencloud/services/groupware/pkg/metrics"
	groupwaremiddleware "github.com/opencloud-eu/opencloud/services/groupware/pkg/middleware"
)

// using a wrapper class for requests, to group multiple parameters, really to avoid crowding the
// API of handlers but also to make it easier to expand it in the future without having to modify
// the parameter list of every single handler function
type Request struct {
	g       *Groupware
	user    user
	r       *http.Request
	ctx     context.Context
	logger  *log.Logger
	session *jmap.Session
}

func (r Request) push(typ string, event any) {
	r.g.push(r.user, typ, event)
}

func (r Request) GetUser() user {
	return r.user
}

func (r Request) GetRequestId() string {
	return chimiddleware.GetReqID(r.ctx)
}

func (r Request) GetTraceId() string {
	return groupwaremiddleware.GetTraceID(r.ctx)
}

var (
	errNoPrimaryAccountFallback            = errors.New("no primary account fallback")
	errNoPrimaryAccountForMail             = errors.New("no primary account for mail")
	errNoPrimaryAccountForBlob             = errors.New("no primary account for blob")
	errNoPrimaryAccountForVacationResponse = errors.New("no primary account for vacation response")
	errNoPrimaryAccountForSubmission       = errors.New("no primary account for submission")
	errNoPrimaryAccountForTask             = errors.New("no primary account for task")
	errNoPrimaryAccountForCalendar         = errors.New("no primary account for calendar")
	errNoPrimaryAccountForContact          = errors.New("no primary account for contact")
	// errNoPrimaryAccountForSieve            = errors.New("no primary account for sieve")
	// errNoPrimaryAccountForQuota            = errors.New("no primary account for quota")
	// errNoPrimaryAccountForWebsocket        = errors.New("no primary account for websocket")
)

func (r Request) GetAccountIdWithoutFallback() (string, *Error) {
	accountId := chi.URLParam(r.r, UriParamAccountId)
	if accountId == "" || accountId == defaultAccountId {
		r.logger.Error().Err(errNoPrimaryAccountFallback).Msg("failed to determine the accountId")
		return "", apiError(r.errorId(), ErrorNonExistingAccount,
			withDetail("Failed to determine the account to use"),
			withSource(&ErrorSource{Parameter: UriParamAccountId}),
		)
	}
	return accountId, nil
}

func (r Request) getAccountId(fallback string, err error) (string, *Error) {
	accountId := chi.URLParam(r.r, UriParamAccountId)
	if accountId == "" || accountId == defaultAccountId {
		accountId = fallback
	}
	if accountId == "" {
		r.logger.Error().Err(err).Msg("failed to determine the accountId")
		return "", apiError(r.errorId(), ErrorNonExistingAccount,
			withDetail("Failed to determine the account to use"),
			withSource(&ErrorSource{Parameter: UriParamAccountId}),
		)
	}
	return accountId, nil
}

func (r Request) GetAccountIdForMail() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.Mail, errNoPrimaryAccountForMail)
}

func (r Request) GetAccountIdForBlob() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.Blob, errNoPrimaryAccountForBlob)
}

func (r Request) GetAccountIdForVacationResponse() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.VacationResponse, errNoPrimaryAccountForVacationResponse)
}

func (r Request) GetAccountIdForSubmission() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.Blob, errNoPrimaryAccountForSubmission)
}

func (r Request) GetAccountIdForTask() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.Task, errNoPrimaryAccountForTask)
}

func (r Request) GetAccountIdForCalendar() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.Calendar, errNoPrimaryAccountForCalendar)
}

func (r Request) GetAccountIdForContact() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.Contact, errNoPrimaryAccountForContact)
}

func (r Request) GetAccountForMail() (jmap.Account, *Error) {
	accountId, err := r.GetAccountIdForMail()
	if err != nil {
		return jmap.Account{}, err
	}

	account, ok := r.session.Accounts[accountId]
	if !ok {
		r.logger.Debug().Msgf("failed to find account '%v'", accountId)
		// TODO metric for inexistent accounts
		return jmap.Account{}, apiError(r.errorId(), ErrorNonExistingAccount,
			withDetail(fmt.Sprintf("The account '%v' does not exist", log.SafeString(accountId))),
			withSource(&ErrorSource{Parameter: UriParamAccountId}),
		)
	}
	return account, nil
}

func (r Request) parameterError(param string, detail string) *Error {
	return r.observedParameterError(ErrorInvalidRequestParameter,
		withDetail(detail),
		withSource(&ErrorSource{Parameter: param}))
}

func (r Request) parameterErrorResponse(param string, detail string) Response {
	return errorResponse(r.parameterError(param, detail))
}

func (r Request) parseIntParam(param string, defaultValue int) (int, bool, *Error) {
	q := r.r.URL.Query()
	if !q.Has(param) {
		return defaultValue, false, nil
	}

	str := q.Get(param)
	if str == "" {
		return defaultValue, false, nil
	}

	value, err := strconv.ParseInt(str, 10, 0)
	if err != nil {
		// don't include the original error, as it leaks too much about our implementation, e.g.:
		// strconv.ParseInt: parsing \"a\": invalid syntax
		msg := fmt.Sprintf("Invalid numeric value for query parameter '%v': '%s'", param, log.SafeString(str))
		return defaultValue, true, r.observedParameterError(ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return int(value), true, nil
}

func (r Request) parseUIntParam(param string, defaultValue uint) (uint, bool, *Error) {
	q := r.r.URL.Query()
	if !q.Has(param) {
		return defaultValue, false, nil
	}

	str := q.Get(param)
	if str == "" {
		return defaultValue, false, nil
	}

	value, err := strconv.ParseUint(str, 10, 0)
	if err != nil {
		// don't include the original error, as it leaks too much about our implementation, e.g.:
		// strconv.ParseInt: parsing \"a\": invalid syntax
		msg := fmt.Sprintf("Invalid numeric value for query parameter '%v': '%s'", param, log.SafeString(str))
		return defaultValue, true, r.observedParameterError(ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return uint(value), true, nil
}

func (r Request) parseDateParam(param string) (time.Time, bool, *Error) {
	q := r.r.URL.Query()
	if !q.Has(param) {
		return time.Time{}, false, nil
	}

	str := q.Get(param)
	if str == "" {
		return time.Time{}, false, nil
	}

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		msg := fmt.Sprintf("Invalid RFC3339 value for query parameter '%v': '%s': %s", param, log.SafeString(str), err.Error())
		return time.Time{}, true, r.observedParameterError(ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return t, true, nil
}

func (r Request) parseBoolParam(param string, defaultValue bool) (bool, bool, *Error) {
	q := r.r.URL.Query()
	if !q.Has(param) {
		return defaultValue, false, nil
	}

	str := q.Get(param)
	if str == "" {
		return defaultValue, false, nil
	}

	b, err := strconv.ParseBool(str)
	if err != nil {
		msg := fmt.Sprintf("Invalid boolean value for query parameter '%v': '%s': %s", param, log.SafeString(str), err.Error())
		return defaultValue, true, r.observedParameterError(ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return b, true, nil
}

func (r Request) parseMapParam(param string) (map[string]string, bool, *Error) {
	q := r.r.URL.Query()
	if !q.Has(param) {
		return map[string]string{}, false, nil
	}

	result := map[string]string{}
	prefix := param + "."
	for name, values := range q {
		if strings.HasPrefix(name, prefix) {
			if len(values) > 0 {
				key := name[len(prefix)+1:]
				result[key] = values[0]
			}
		}
	}
	return result, true, nil
}

func (r Request) body(target any) *Error {
	body := r.r.Body
	defer func(b io.ReadCloser) {
		err := b.Close()
		if err != nil {
			r.logger.Error().Err(err).Msg("failed to close request body")
		}
	}(body)

	err := json.NewDecoder(body).Decode(target)
	if err != nil {
		return r.observedParameterError(ErrorInvalidRequestBody, withSource(&ErrorSource{Pointer: "/"})) // we don't get any details here
	}
	return nil
}

func (r Request) observe(obs prometheus.Observer, value float64) {
	metrics.WithExemplar(obs, value, r.GetRequestId(), r.GetTraceId())
}

func (r Request) observeParameterError(err *Error) *Error {
	if err != nil {
		r.g.metrics.ParameterErrorCounter.WithLabelValues(err.Code).Inc()
	}
	return err
}

func (r Request) observeJmapError(jerr jmap.Error) jmap.Error {
	if jerr != nil {
		r.g.metrics.JmapErrorCounter.WithLabelValues(r.session.JmapEndpoint, strconv.Itoa(jerr.Code())).Inc()
	}
	return jerr
}

func (r Request) needTask() (bool, Response) {
	if r.session.Capabilities.Tasks == nil {
		return false, errorResponseWithSessionState(r.apiError(&ErrorMissingTasksSessionCapability), r.session.State)
	}
	return true, Response{}
}

func (r Request) needTaskForAccount(accountId string) (bool, Response) {
	if ok, resp := r.needTask(); !ok {
		return ok, resp
	}
	account, ok := r.session.Accounts[accountId]
	if !ok {
		return false, errorResponseWithSessionState(r.apiError(&ErrorAccountNotFound), r.session.State)
	}
	if account.AccountCapabilities.Tasks == nil {
		return false, errorResponseWithSessionState(r.apiError(&ErrorMissingTasksAccountCapability), r.session.State)
	}
	return true, Response{}
}

func (r Request) needTaskWithAccount() (bool, string, Response) {
	accountId, err := r.GetAccountIdForTask()
	if err != nil {
		return false, "", errorResponse(err)
	}
	if ok, resp := r.needTaskForAccount(accountId); !ok {
		return false, accountId, resp
	}
	return true, accountId, Response{}
}

func (r Request) needCalendar() (bool, Response) {
	if r.session.Capabilities.Calendars == nil {
		return false, errorResponseWithSessionState(r.apiError(&ErrorMissingCalendarsSessionCapability), r.session.State)
	}
	return true, Response{}
}

func (r Request) needCalendarForAccount(accountId string) (bool, Response) {
	if ok, resp := r.needCalendar(); !ok {
		return ok, resp
	}
	account, ok := r.session.Accounts[accountId]
	if !ok {
		return false, errorResponseWithSessionState(r.apiError(&ErrorAccountNotFound), r.session.State)
	}
	if account.AccountCapabilities.Calendars == nil {
		return false, errorResponseWithSessionState(r.apiError(&ErrorMissingCalendarsAccountCapability), r.session.State)
	}
	return true, Response{}
}

func (r Request) needCalendarWithAccount() (bool, string, Response) {
	accountId, err := r.GetAccountIdForCalendar()
	if err != nil {
		return false, "", errorResponse(err)
	}
	if ok, resp := r.needCalendarForAccount(accountId); !ok {
		return false, accountId, resp
	}
	return true, accountId, Response{}
}

func (r Request) needContact() (bool, Response) {
	if r.session.Capabilities.Contacts == nil {
		return false, errorResponseWithSessionState(r.apiError(&ErrorMissingContactsSessionCapability), r.session.State)
	}
	return true, Response{}
}

func (r Request) needContactForAccount(accountId string) (bool, Response) {
	if ok, resp := r.needContact(); !ok {
		return ok, resp
	}
	account, ok := r.session.Accounts[accountId]
	if !ok {
		return false, errorResponseWithSessionState(r.apiError(&ErrorAccountNotFound), r.session.State)
	}
	if account.AccountCapabilities.Contacts == nil {
		return false, errorResponseWithSessionState(r.apiError(&ErrorMissingContactsAccountCapability), r.session.State)
	}
	return true, Response{}
}

func (r Request) needContactWithAccount() (bool, string, Response) {
	accountId, err := r.GetAccountIdForContact()
	if err != nil {
		return false, "", errorResponse(err)
	}
	if ok, resp := r.needContactForAccount(accountId); !ok {
		return false, accountId, resp
	}
	return true, accountId, Response{}
}
