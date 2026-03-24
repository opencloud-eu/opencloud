package groupware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"

	"github.com/opencloud-eu/opencloud/services/groupware/pkg/metrics"
	groupwaremiddleware "github.com/opencloud-eu/opencloud/services/groupware/pkg/middleware"
)

const (
	// TODO remove this once Stalwart has actual support for Tasks and we don't need to mock it any more
	IgnoreSessionCapabilityChecksForTasks = true

	MaxSortParams = 16
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

func isDefaultAccountId(accountId string) bool {
	return slices.Contains(defaultAccountIds, accountId)
}

func (r *Request) push(typ string, event any) {
	r.g.push(r.user, typ, event)
}

func (r *Request) GetUser() user {
	return r.user
}

func (r *Request) GetRequestId() string {
	return chimiddleware.GetReqID(r.ctx)
}

func (r *Request) GetTraceId() string {
	return groupwaremiddleware.GetTraceID(r.ctx)
}

var (
	errNoPrimaryAccountFallback            = errors.New("no primary account fallback")
	errNoPrimaryAccountForMail             = errors.New("no primary account for mail")
	errNoPrimaryAccountForBlob             = errors.New("no primary account for blob")
	errNoPrimaryAccountForVacationResponse = errors.New("no primary account for vacation response")
	errNoPrimaryAccountForSubmission       = errors.New("no primary account for submission")
	errNoPrimaryAccountForQuota            = errors.New("no primary account for quota")
	// errNoPrimaryAccountForTask             = errors.New("no primary account for task")
	// errNoPrimaryAccountForCalendar         = errors.New("no primary account for calendar")
	// errNoPrimaryAccountForContact          = errors.New("no primary account for contact")
	// errNoPrimaryAccountForSieve            = errors.New("no primary account for sieve")
	// errNoPrimaryAccountForWebsocket        = errors.New("no primary account for websocket")
)

func (r *Request) HeaderParam(name string) (string, *Error) {
	value := r.r.Header.Get(name)
	if value == "" {
		msg := fmt.Sprintf("Missing mandatory request header '%s'", name)
		return "", r.observedParameterError(ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Header: name}),
		)
	} else {
		return value, nil
	}
}

func (r *Request) HeaderParamDoc(name string, _ string) (string, *Error) {
	return r.HeaderParam(name)
}

func (r *Request) OptHeaderParam(name string) string {
	return r.r.Header.Get(name)
}

func (r *Request) OptHeaderParamDoc(name string, _ string) string {
	return r.OptHeaderParam(name)
}

func (r *Request) PathParam(name string) (string, *Error) {
	value := chi.URLParam(r.r, name)
	if value == "" {
		msg := fmt.Sprintf("Missing mandatory path parameter '%s'", name)
		return "", r.observedParameterError(ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: name}),
		)
	} else {
		return value, nil
	}
}

func (r *Request) PathParamDoc(name string, _ string) (string, *Error) {
	return r.PathParam(name)
}

func (r *Request) PathListParamDoc(name string, _ string) ([]string, *Error) {
	value, err := r.PathParam(name)
	if err != nil {
		return nil, err
	}
	return strings.Split(value, ","), nil
}

func (r *Request) AllAccountIds() []string {
	// TODO potentially filter on "subscribed" accounts?
	return structs.Uniq(structs.Keys(r.session.Accounts))
}

func (r *Request) GetAccountIdWithoutFallback() (string, *Error) {
	accountId := chi.URLParam(r.r, UriParamAccountId)
	if accountId == "" || isDefaultAccountId(accountId) {
		r.logger.Error().Err(errNoPrimaryAccountFallback).Msg("failed to determine the accountId")
		return "", apiError(r.errorId(), ErrorNonExistingAccount,
			withDetail("Failed to determine the account to use"),
			withSource(&ErrorSource{Parameter: UriParamAccountId}),
		)
	}
	return accountId, nil
}

func (r *Request) getAccountId(fallback string, err error) (string, *Error) {
	accountId := chi.URLParam(r.r, UriParamAccountId)
	if accountId == "" || isDefaultAccountId(accountId) {
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

func (r *Request) GetAccountIdForMail() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.Mail, errNoPrimaryAccountForMail)
}

func (r *Request) GetAccountIdForBlob() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.Blob, errNoPrimaryAccountForBlob)
}

func (r *Request) GetAccountIdForVacationResponse() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.VacationResponse, errNoPrimaryAccountForVacationResponse)
}

func (r *Request) GetAccountIdForQuota() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.Quota, errNoPrimaryAccountForQuota)
}

func (r *Request) GetAccountIdForSubmission() (string, *Error) {
	return r.getAccountId(r.session.PrimaryAccounts.Blob, errNoPrimaryAccountForSubmission)
}

func (r *Request) GetAccountIdForTask() (string, *Error) {
	// TODO we don't have these yet, not implemented in Stalwart
	// return r.getAccountId(r.session.PrimaryAccounts.Task, errNoPrimaryAccountForTask)
	return r.GetAccountIdForMail()
}

func (r *Request) GetAccountIdForCalendar() (string, *Error) {
	// TODO we don't have these yet, not implemented in Stalwart
	// return r.getAccountId(r.session.PrimaryAccounts.Calendar, errNoPrimaryAccountForCalendar)
	return r.GetAccountIdForMail()
}

func (r *Request) GetAccountIdForContact() (string, *Error) {
	// TODO we don't have these yet, not implemented in Stalwart
	// return r.getAccountId(r.session.PrimaryAccounts.Contact, errNoPrimaryAccountForContact)
	return r.GetAccountIdForMail()
}

func (r *Request) GetAccountForMail() (string, jmap.Account, *Error) {
	accountId, err := r.GetAccountIdForMail()
	if err != nil {
		return "", jmap.Account{}, err
	}

	account, ok := r.session.Accounts[accountId]
	if !ok {
		r.logger.Debug().Msgf("failed to find account '%v'", accountId)
		// TODO metric for inexistent accounts
		return accountId, jmap.Account{}, apiError(r.errorId(), ErrorNonExistingAccount,
			withDetail(fmt.Sprintf("The account '%v' does not exist", log.SafeString(accountId))),
			withSource(&ErrorSource{Parameter: UriParamAccountId}),
		)
	}
	return accountId, account, nil
}

func (r *Request) parameterError(param string, detail string) *Error {
	return r.observedParameterError(ErrorInvalidRequestParameter,
		withDetail(detail),
		withSource(&ErrorSource{Parameter: param}))
}

func (r *Request) parameterErrorResponse(accountIds []string, param string, detail string) Response {
	return r.errorN(accountIds, r.parameterError(param, detail))
}

func (r *Request) getStringParam(param string, defaultValue string) (string, bool) {
	q := r.r.URL.Query()
	if !q.Has(param) {
		return defaultValue, false
	}
	str := q.Get(param)
	if str == "" {
		return defaultValue, false
	}
	return str, true
}

func (r *Request) getMandatoryStringParam(param string) (string, *Error) {
	str := ""
	q := r.r.URL.Query()
	if q.Has(param) {
		str = q.Get(param)
	}
	if str == "" {
		msg := fmt.Sprintf("Missing required value for query parameter '%v'", param)
		return "", r.observedParameterError(ErrorMissingMandatoryRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return str, nil
}

func (r *Request) parseIntParam(param string, defaultValue int) (int, bool, *Error) {
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

func (r *Request) parseUIntParam(param string, defaultValue uint) (uint, bool, *Error) {
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

func (r *Request) parseDateParam(param string) (time.Time, bool, *Error) {
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

func (r *Request) parseBoolParam(param string, defaultValue bool) (bool, bool, *Error) {
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

func (r *Request) parseMapParam(param string) (map[string]string, bool, *Error) {
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

func (r *Request) parseOptStringListParam(param string) ([]string, bool, *Error) {
	result := []string{}
	q := r.r.URL.Query()
	if !q.Has(param) {
		return nil, false, nil
	}
	for _, value := range q[param] {
		for _, v := range strings.Split(value, ",") {
			if strings.TrimSpace(v) != "" {
				result = append(result, v)
			}
		}
	}
	return result, true, nil
}

func (r *Request) bodydoc(target any, _ string) *Error {
	return r.body(target)
}

func (r *Request) body(target any) *Error {
	body := r.r.Body
	defer func(b io.ReadCloser) {
		err := b.Close()
		if err != nil {
			r.logger.Error().Err(err).Msg("failed to close request body")
		}
	}(body)

	err := json.NewDecoder(body).Decode(target)
	if err != nil {
		r.logger.Warn().Msgf("failed to deserialize the request body: %s", err.Error())
		return r.observedParameterError(ErrorInvalidRequestBody, withSource(&ErrorSource{Pointer: "/"})) // we don't get any details here
	}
	return nil
}

func (r *Request) language() string {
	return r.r.Header.Get("Accept-Language")
}

func (r *Request) observe(obs prometheus.Observer, value float64) {
	metrics.WithExemplar(obs, value, r.GetRequestId(), r.GetTraceId())
}

func (r *Request) observeParameterError(err *Error) *Error {
	if err != nil {
		r.g.metrics.ParameterErrorCounter.WithLabelValues(err.Code).Inc()
	}
	return err
}

func (r *Request) observeJmapError(jerr jmap.Error) jmap.Error {
	if jerr != nil {
		r.g.metrics.JmapErrorCounter.WithLabelValues(r.session.JmapEndpoint, strconv.Itoa(jerr.Code())).Inc()
	}
	return jerr
}

func (r *Request) needTask(accountId string) (bool, Response) {
	if !IgnoreSessionCapabilityChecksForTasks {
		if r.session.Capabilities.Tasks == nil {
			return false, errorResponse(single(accountId), r.apiError(&ErrorMissingTasksSessionCapability), r.session.State, jmap.Language(r.language()))
		}
	}
	return true, Response{}
}

func (r *Request) needTaskForAccount(accountId string) (bool, Response) {
	if ok, resp := r.needTask(accountId); !ok {
		return ok, resp
	}
	account, ok := r.session.Accounts[accountId]
	if !ok {
		return false, errorResponse(single(accountId), r.apiError(&ErrorAccountNotFound), r.session.State, jmap.NoLanguage)
	}
	if account.AccountCapabilities.Tasks == nil {
		return false, errorResponse(single(accountId), r.apiError(&ErrorMissingTasksAccountCapability), r.session.State, jmap.NoLanguage)
	}
	return true, Response{}
}

func (r *Request) needTaskWithAccount() (bool, string, Response) {
	accountId, err := r.GetAccountIdForTask()
	if err != nil {
		return false, "", r.error(accountId, err)
	}
	if ok, resp := r.needTaskForAccount(accountId); !ok {
		return false, accountId, resp
	}
	return true, accountId, Response{}
}

func (r *Request) needCalendar(accountId string) (bool, Response) {
	if r.session.Capabilities.Calendars == nil {
		return false, errorResponse(single(accountId), r.apiError(&ErrorMissingCalendarsSessionCapability), r.session.State, jmap.NoLanguage)
	}
	return true, Response{}
}

func (r *Request) needCalendarForAccount(accountId string) (bool, Response) {
	if ok, resp := r.needCalendar(accountId); !ok {
		return ok, resp
	}
	account, ok := r.session.Accounts[accountId]
	if !ok {
		return false, errorResponse(single(accountId), r.apiError(&ErrorAccountNotFound), r.session.State, jmap.NoLanguage)
	}
	if account.AccountCapabilities.Calendars == nil {
		return false, errorResponse(single(accountId), r.apiError(&ErrorMissingCalendarsAccountCapability), r.session.State, jmap.NoLanguage)
	}
	return true, Response{}
}

func (r *Request) needCalendarWithAccount() (bool, string, Response) {
	accountId, err := r.GetAccountIdForCalendar()
	if err != nil {
		return false, "", r.error(accountId, err)
	}
	if ok, resp := r.needCalendarForAccount(accountId); !ok {
		return false, accountId, resp
	}
	return true, accountId, Response{}
}

func (r *Request) needContact(accountId string) (bool, Response) {
	if r.session.Capabilities.Contacts == nil {
		return false, errorResponse(single(accountId), r.apiError(&ErrorMissingContactsSessionCapability), r.session.State, jmap.NoLanguage)
	}
	return true, Response{}
}

func (r *Request) needContactForAccount(accountId string) (bool, Response) {
	if ok, resp := r.needContact(accountId); !ok {
		return ok, resp
	}
	account, ok := r.session.Accounts[accountId]
	if !ok {
		return false, errorResponse(single(accountId), r.apiError(&ErrorAccountNotFound), r.session.State, jmap.NoLanguage)
	}
	if account.AccountCapabilities.Contacts == nil {
		return false, errorResponse(single(accountId), r.apiError(&ErrorMissingContactsAccountCapability), r.session.State, jmap.NoLanguage)
	}
	return true, Response{}
}

func (r *Request) needContactWithAccount() (bool, string, Response) {
	accountId, err := r.GetAccountIdForContact()
	if err != nil {
		return false, "", r.error(accountId, err)
	}
	if ok, resp := r.needContactForAccount(accountId); !ok {
		return false, accountId, resp
	}
	return true, accountId, Response{}
}

type SortCrit struct {
	Attribute string
	Ascending bool
}

func (r *Request) parseSort(s string, props []string) ([]SortCrit, *Error) {
	parts := strings.SplitN(s, ",", MaxSortParams)
	result := []SortCrit{}
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}

		asc := true
		i := strings.LastIndex(name, ":")
		if i == 0 {
			// invalid spec, e.g. ':asc'
			return nil, r.apiError(&ErrorInvalidSortProperty)
		} else if i > 0 {
			order := name[i+1:]
			name = name[0:i]
			switch order {
			case "", "asc":
				asc = true
			case "desc":
				asc = false
			default:
				return nil, r.apiError(&ErrorInvalidSortSpecification)
			}
		}
		if len(props) > 0 && !slices.Contains(props, name) {
			return nil, r.apiError(&ErrorInvalidSortProperty)
		} else {
			result = append(result, SortCrit{Attribute: name, Ascending: asc})
		}
	}
	return result, nil
}

func mapSort[T any](accountIds []string, req *Request, defaultSort []T, props []string, mapper func(SortCrit) T) ([]T, bool, Response) {
	if sortSpec, ok := req.getStringParam(QueryParamSort, ""); ok && strings.TrimSpace(sortSpec) != "" {
		if sort, err := req.parseSort(sortSpec, props); err != nil {
			return nil, false, errorResponse(accountIds, err, req.session.State, jmap.NoLanguage)
		} else {
			return structs.Map(sort, mapper), true, Response{}
		}
	} else {
		return defaultSort, true, Response{}
	}
}
