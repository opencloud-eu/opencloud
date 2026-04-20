package groupware

import (
	"context"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/metrics"
)

// Get the changes tp Emails since a certain State.
// @api:tags email,changes
func (g *Groupware) GetEmailChanges(w http.ResponseWriter, r *http.Request) {
	changes(Email, w, r, g, func(accountId string, sinceState jmap.State, maxChanges uint, ctx jmap.Context) (jmap.EmailChanges, jmap.SessionState, jmap.State, jmap.Language, jmap.Error) {
		return g.jmap.GetEmailChanges(accountId, sinceState, true, g.config.maxBodyValueBytes, maxChanges, ctx)
	})
}

// Get all the emails in a mailbox.
//
// Retrieve the list of all the emails that are in a given mailbox.
//
// The mailbox must be specified by its id, as part of the request URL path.
//
// A limit and an offset may be specified using the query parameters 'limit' and 'offset',
// respectively.
func (g *Groupware) GetAllEmailsInMailbox(w http.ResponseWriter, r *http.Request) { //NOSONAR
	collapseThreads := false
	fetchBodies := false
	withThreads := true
	query(Email, w, r, g, g.defaults.emailLimit,
		func(req Request, accountId, containerId string, offset int, limit uint, ctx jmap.Context) (jmap.EmailSearchResults, jmap.SessionState, jmap.State, jmap.Language, *Error) {
			emails, sessionState, state, lang, jerr := g.jmap.GetAllEmailsInMailbox(accountId, containerId, offset, limit, collapseThreads, fetchBodies, g.config.maxBodyValueBytes, withThreads, ctx)
			if jerr != nil {
				return emails, sessionState, state, lang, req.apiErrorFromJmap(req.observeJmapError(jerr))
			}

			sanitized, err := req.sanitizeEmails(emails.Results)
			if err != nil {
				return emails, sessionState, state, lang, err
			}

			safe := jmap.EmailSearchResults{
				Results:             sanitized,
				Total:               emails.Total,
				Limit:               emails.Limit,
				Position:            emails.Position,
				CanCalculateChanges: emails.CanCalculateChanges,
			}
			return safe, sessionState, state, lang, nil
		},
	)
}

func (g *Groupware) GetEmailsById(w http.ResponseWriter, r *http.Request) { //NOSONAR
	accept := r.Header.Get("Accept")
	if accept == "message/rfc822" {
		g.stream(w, r, func(req Request, w http.ResponseWriter) *Error {
			id, err := req.PathParam(UriParamEmailId)
			if err != nil {
				return err
			}
			ids := strings.Split(id, ",")
			if len(ids) != 1 {
				return req.parameterError(UriParamEmailId, fmt.Sprintf("when the Accept header is set to '%s', the API only supports serving a single email id", accept))
			}

			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return err
			}

			_, ok, err := req.parseBoolParam(QueryParamMarkAsSeen, false)
			if err != nil {
				return err
			}
			if ok {
				return req.parameterError(QueryParamMarkAsSeen, fmt.Sprintf("when the Accept header is set to '%s', the API does not support setting %s", accept, QueryParamMarkAsSeen))
			}

			logger := log.From(req.logger.With().Str(logAccountId, log.SafeString(accountId)).Str("id", log.SafeString(id)).Str("accept", log.SafeString(accept)))
			ctx := req.ctx.WithLogger(logger)
			blobId, _, _, _, jerr := g.jmap.GetEmailBlobId(accountId, id, ctx)
			if jerr != nil {
				return req.apiErrorFromJmap(req.observeJmapError(jerr))
			}
			if blobId == "" {
				return nil
			} else {
				name := blobId + ".eml"
				typ := accept
				accountId, gwerr := req.GetAccountIdForBlob()
				if gwerr != nil {
					return gwerr
				}
				return req.serveBlob(blobId, name, typ, ctx, accountId, w)
			}
		})
	} else {
		g.respond(w, r, func(req Request) Response {
			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return req.error(accountId, err)
			}
			l := req.logger.With().Str(logAccountId, log.SafeString(accountId))

			id, err := req.PathParam(UriParamEmailId)
			if err != nil {
				return req.error(accountId, err)
			}
			ids := strings.Split(id, ",")
			if len(ids) < 1 {
				return req.parameterErrorResponse(single(accountId), UriParamEmailId, fmt.Sprintf("Invalid value for path parameter '%v': '%s': %s", UriParamEmailId, log.SafeString(id), "empty list of mail ids"))
			}

			markAsSeen, ok, err := req.parseBoolParam(QueryParamMarkAsSeen, false)
			if err != nil {
				return req.error(accountId, err)
			}
			if ok {
				l = l.Bool(QueryParamMarkAsSeen, markAsSeen)
			}

			if len(ids) == 1 {
				logger := log.From(l.Str(UriParamEmailId, log.SafeString(id)))
				ctx := req.ctx.WithLogger(logger)
				emails, _, sessionState, state, lang, jerr := g.jmap.GetEmails(accountId, ids, true, g.config.maxBodyValueBytes, markAsSeen, true, ctx)
				if jerr != nil {
					return req.jmapError(accountId, jerr, sessionState, lang)
				}
				if len(emails) < 1 {
					return req.notFound(accountId, sessionState, EmailResponseObjectType, state)
				} else {
					sanitized, err := req.sanitizeEmail(emails[0])
					if err != nil {
						return req.error(accountId, err)
					}
					return req.respond(accountId, sanitized, sessionState, EmailResponseObjectType, state, lang)
				}
			} else {
				logger := log.From(l.Array(UriParamEmailId, log.SafeStringArray(ids)))
				ctx := req.ctx.WithLogger(logger)
				emails, _, sessionState, state, lang, jerr := g.jmap.GetEmails(accountId, ids, true, g.config.maxBodyValueBytes, markAsSeen, false, ctx)
				if jerr != nil {
					return req.jmapError(accountId, jerr, sessionState, lang)
				}
				if len(emails) < 1 {
					return req.notFound(accountId, sessionState, EmailResponseObjectType, state)
				} else {
					sanitized, err := req.sanitizeEmails(emails)
					if err != nil {
						return req.error(accountId, err)
					}
					return req.respond(accountId, sanitized, sessionState, EmailResponseObjectType, state, lang)
				}
			}
		})
	}
}

// Get the attachments of an email by its identifier.
//
// @api:tags email
func (g *Groupware) GetEmailAttachments(w http.ResponseWriter, r *http.Request) { //NOSONAR
	contextAppender := func(l zerolog.Context) zerolog.Context { return l }
	q := r.URL.Query()
	var attachmentSelector func(jmap.EmailBodyPart) bool = nil
	if q.Has(QueryParamPartId) {
		partId := q.Get(QueryParamPartId)
		attachmentSelector = func(part jmap.EmailBodyPart) bool { return part.PartId == partId }
		contextAppender = func(l zerolog.Context) zerolog.Context { return l.Str(QueryParamPartId, log.SafeString(partId)) }
	}
	if q.Has(QueryParamAttachmentName) {
		name := q.Get(QueryParamAttachmentName)
		attachmentSelector = func(part jmap.EmailBodyPart) bool { return part.Name == name }
		contextAppender = func(l zerolog.Context) zerolog.Context { return l.Str(QueryParamAttachmentName, log.SafeString(name)) }
	}
	if q.Has(QueryParamAttachmentBlobId) {
		blobId := q.Get(QueryParamAttachmentBlobId)
		attachmentSelector = func(part jmap.EmailBodyPart) bool { return part.BlobId == blobId }
		contextAppender = func(l zerolog.Context) zerolog.Context {
			return l.Str(QueryParamAttachmentBlobId, log.SafeString(blobId))
		}
	}

	if attachmentSelector == nil {
		g.respond(w, r, func(req Request) Response {
			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return req.error(accountId, err)
			}
			l := req.logger.With().Str(logAccountId, log.SafeString(accountId))

			id, err := req.PathParam(UriParamEmailId)
			if err != nil {
				return req.error(accountId, err)
			}

			logger := log.From(l)
			ctx := req.ctx.WithLogger(logger)
			emails, _, sessionState, state, lang, jerr := g.jmap.GetEmails(accountId, single(id), false, 0, false, false, ctx)
			if jerr != nil {
				return req.jmapError(accountId, jerr, sessionState, lang)
			}
			if len(emails) < 1 {
				return req.notFound(accountId, sessionState, EmailResponseObjectType, state)
			}
			email, err := req.sanitizeEmail(emails[0])
			if err != nil {
				return req.error(accountId, err)
			}
			var body []jmap.EmailBodyPart = email.Attachments
			return req.respond(accountId, body, sessionState, EmailResponseObjectType, state, lang)
		})
	} else {
		g.stream(w, r, func(req Request, w http.ResponseWriter) *Error {
			mailAccountId, err := req.GetAccountIdForMail()
			if err != nil {
				return err
			}
			blobAccountId, err := req.GetAccountIdForBlob()
			if err != nil {
				return err
			}

			id, err := req.PathParam(UriParamEmailId)
			if err != nil {
				return err
			}

			l := req.logger.With().
				Str(logAccountId, log.SafeString(mailAccountId)).
				Str(logBlobAccountId, log.SafeString(blobAccountId)).
				Str(UriParamEmailId, log.SafeString(id))
			l = contextAppender(l)
			logger := log.From(l)
			ctx := req.ctx.WithLogger(logger)
			emails, _, _, _, lang, jerr := g.jmap.GetEmails(mailAccountId, single(id), false, 0, false, false, ctx)
			if jerr != nil {
				return req.apiErrorFromJmap(req.observeJmapError(jerr))
			}
			if len(emails) < 1 {
				return nil
			}

			email, err := req.sanitizeEmail(emails[0])
			if err != nil {
				return err
			}
			var attachment *jmap.EmailBodyPart = nil
			for _, part := range email.Attachments {
				if attachmentSelector(part) {
					attachment = &part
					break
				}
			}
			if attachment == nil {
				return nil
			}

			blob, lang, jerr := g.jmap.DownloadBlobStream(blobAccountId, attachment.BlobId, attachment.Name, attachment.Type, ctx)
			if blob != nil && blob.Body != nil {
				defer func(Body io.ReadCloser) {
					err := Body.Close()
					if err != nil {
						logger.Error().Err(err).Msg("failed to close response body")
					}
				}(blob.Body)
			}
			if jerr != nil {
				return req.apiErrorFromJmap(jerr)
			}
			if blob == nil {
				w.WriteHeader(http.StatusNotFound)
				return nil
			}

			if blob.Type != "" {
				w.Header().Add("Content-Type", blob.Type)
			}
			if blob.CacheControl != "" {
				w.Header().Add("Cache-Control", blob.CacheControl)
			}
			if blob.ContentDisposition != "" {
				w.Header().Add("Content-Disposition", blob.ContentDisposition)
			}
			if blob.Size >= 0 {
				w.Header().Add("Content-Size", strconv.Itoa(blob.Size))
			}
			if lang != "" {
				w.Header().Add("Content-Language", string(lang))
			}
			_, cerr := io.Copy(w, blob.Body)
			if cerr != nil {
				return req.observedParameterError(ErrorStreamingResponse)
			}

			return nil
		})
	}
}

func (g *Groupware) getEmailsSince(w http.ResponseWriter, r *http.Request, since jmap.State) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With().Str(QueryParamSince, log.SafeString(string(since)))

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(logAccountId, log.SafeString(accountId))

		maxChanges, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0)
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Uint(QueryParamMaxChanges, maxChanges)
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		changes, sessionState, state, lang, jerr := g.jmap.GetEmailChanges(accountId, since, true, g.config.maxBodyValueBytes, maxChanges, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, changes, sessionState, EmailResponseObjectType, state, lang)
	})
}

type EmailSearchSnippetsResults struct {
	Results    []Snippet  `json:"results,omitempty"`
	Total      uint       `json:"total,omitzero"`
	Limit      uint       `json:"limit,omitzero"`
	QueryState jmap.State `json:"queryState,omitempty"`
}

type EmailWithSnippets struct {
	AccountId string `json:"accountId,omitempty"`
	jmap.Email
	Snippets []SnippetWithoutEmailId `json:"snippets,omitempty"`
}

type Snippet struct {
	AccountId string `json:"accountId,omitempty"`
	jmap.SearchSnippetWithMeta
}

type SnippetWithoutEmailId struct {
	Subject string `json:"subject,omitempty"`
	Preview string `json:"preview,omitempty"`
}

type EmailWithSnippetsSearchResults struct {
	Results    []EmailWithSnippets `json:"results"`
	Total      *uint               `json:"total,omitzero"`
	Position   uint                `json:"position"`
	Limit      uint                `json:"limit,omitzero"`
	QueryState jmap.State          `json:"queryState,omitempty"`
}

type EmailSearchResults struct {
	Results    []jmap.Email `json:"results"`
	Total      uint         `json:"total,omitzero"`
	Limit      uint         `json:"limit,omitzero"`
	QueryState jmap.State   `json:"queryState,omitempty"`
}

func (g *Groupware) buildEmailFilter(req Request) (bool, jmap.EmailFilterElement, bool, int, uint, *log.Logger, *Error) { //NOSONAR
	mailboxId, _ := req.getStringParam(QueryParamMailboxId, "")                      // the identifier of the Mailbox to which to restrict the search
	text, _ := req.getStringParam(QueryParamSearchText, "")                          // text that must be included in the Email, specifically in From, To, Cc, Bcc, Subject and any text/* body part
	from, _ := req.getStringParam(QueryParamSearchFrom, "")                          // text that must be included in the From header of the Email
	to, _ := req.getStringParam(QueryParamSearchTo, "")                              // text that must be included in the To header of the Email
	cc, _ := req.getStringParam(QueryParamSearchCc, "")                              // text that must be included in the Cc header of the Email
	bcc, _ := req.getStringParam(QueryParamSearchBcc, "")                            // text that must be included in the Bcc header of the Email
	subject, _ := req.getStringParam(QueryParamSearchSubject, "")                    // text that must be included in the Subject of the Email
	body, _ := req.getStringParam(QueryParamSearchBody, "")                          // text that must be included in any text/* part of the body of the Email
	messageId, _ := req.getStringParam(QueryParamSearchMessageId, "")                // value of the Message-ID header of the Email
	notInMailboxIds, _, err := req.parseOptStringListParam(QueryParamNotInMailboxId) // a comma-separated list of identifiers of Mailboxes the Email must *not* be in
	if err != nil {
		return false, nil, false, 0, 0, nil, err
	}
	keywords, _, err := req.parseOptStringListParam(QueryParamSearchKeyword) // the Email must have all those keywords
	if err != nil {
		return false, nil, false, 0, 0, nil, err
	}

	snippets := false

	l := req.logger.With()

	offset, ok, err := req.parseIntParam(QueryParamOffset, 0) // pagination element offset
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Int(QueryParamOffset, offset)
	}

	limit, ok, err := req.parseUIntParam(QueryParamLimit, g.defaults.emailLimit) // maximum number of results (size of a page)
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Uint(QueryParamLimit, limit)
	}

	before, ok, err := req.parseDateParam(QueryParamSearchBefore) // the Email must have been received before this date-time
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Time(QueryParamSearchBefore, before)
	}

	after, ok, err := req.parseDateParam(QueryParamSearchAfter) // the Email must have been received after this date-time
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Time(QueryParamSearchAfter, after)
	}

	if mailboxId != "" {
		l = l.Str(QueryParamMailboxId, log.SafeString(mailboxId))
	}
	if len(notInMailboxIds) > 0 {
		l = l.Array(QueryParamNotInMailboxId, log.SafeStringArray(notInMailboxIds))
	}
	if text != "" {
		l = l.Str(QueryParamSearchText, log.SafeString(text))
	}
	if from != "" {
		l = l.Str(QueryParamSearchFrom, log.SafeString(from))
	}
	if to != "" {
		l = l.Str(QueryParamSearchTo, log.SafeString(to))
	}
	if cc != "" {
		l = l.Str(QueryParamSearchCc, log.SafeString(cc))
	}
	if bcc != "" {
		l = l.Str(QueryParamSearchBcc, log.SafeString(bcc))
	}
	if subject != "" {
		l = l.Str(QueryParamSearchSubject, log.SafeString(subject))
	}
	if body != "" {
		l = l.Str(QueryParamSearchBody, log.SafeString(body))
	}
	if messageId != "" {
		l = l.Str(QueryParamSearchMessageId, log.SafeString(messageId))
	}

	minSize, ok, err := req.parseIntParam(QueryParamSearchMinSize, 0) // the minimum size of the Email
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Int(QueryParamSearchMinSize, minSize)
	}

	maxSize, ok, err := req.parseIntParam(QueryParamSearchMaxSize, 0) // the maximum size of the Email
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Int(QueryParamSearchMaxSize, maxSize)
	}

	logger := log.From(l)

	var filter jmap.EmailFilterElement

	firstFilter := jmap.EmailFilterCondition{
		Text:               text,
		InMailbox:          mailboxId,
		InMailboxOtherThan: notInMailboxIds,
		From:               from,
		To:                 to,
		Cc:                 cc,
		Bcc:                bcc,
		Subject:            subject,
		Body:               body,
		Before:             before,
		After:              after,
		MinSize:            minSize,
		MaxSize:            maxSize,
		Header:             []string{},
	}
	if messageId != "" {
		// The array MUST contain either one or two elements.
		// The first element is the name of the header field to match against.
		// The second (optional) element is the text to look for in the header field value.
		// If not supplied, the message matches simply if it has a header field of the given name.
		firstFilter.Header = []string{"Message-ID", messageId}
	}
	filter = &firstFilter

	if text != "" || subject != "" || body != "" {
		snippets = true
	}

	if len(keywords) > 0 {
		firstFilter.HasKeyword = keywords[0]
		if len(keywords) > 1 {
			firstFilter.HasKeyword = keywords[0]
			filters := make([]jmap.EmailFilterElement, len(keywords)-1)
			for i, keyword := range keywords[1:] {
				filters[i] = jmap.EmailFilterCondition{HasKeyword: keyword}
			}
			filter = &jmap.EmailFilterOperator{
				Operator:   jmap.And,
				Conditions: filters,
			}
		}
	}

	return true, filter, snippets, offset, limit, logger, nil
}

func (g *Groupware) GetEmails(w http.ResponseWriter, r *http.Request) { //NOSONAR
	q := r.URL.Query()
	since := q.Get(QueryParamSince)
	if since == "" {
		since = r.Header.Get(HeaderParamSince)
	}
	if since != "" {
		// get email changes since a given state
		g.getEmailsSince(w, r, jmap.State(since))
	} else {
		// do a search
		g.respond(w, r, func(req Request) Response {
			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return req.error(accountId, err)
			}

			l := req.logger.With().Str(logAccountId, log.SafeString(accountId))

			ok, filter, makesSnippets, offset, limit, logger, err := g.buildEmailFilter(req)
			if !ok {
				return req.error(accountId, err)
			}

			if !filter.IsNotEmpty() {
				filter = nil
			}

			calculateTotal := true
			if b, ok, err := req.parseBoolParam(QueryParamCalculateTotal, true); err != nil {
				return req.error(accountId, err)
			} else if ok {
				calculateTotal = b
				l = l.Bool(QueryParamCalculateTotal, calculateTotal)
			}

			fetchBodies := false
			collapseThreads := false

			logger = log.From(l)
			ctx := req.ctx.WithLogger(logger)

			resultsByAccount, sessionState, state, lang, jerr := g.jmap.QueryEmailsWithSnippets(single(accountId), filter, offset, limit, collapseThreads, calculateTotal, fetchBodies, g.config.maxBodyValueBytes, ctx)
			if jerr != nil {
				return req.jmapError(accountId, jerr, sessionState, lang)
			}

			if results, ok := resultsByAccount[accountId]; ok {
				flattened := make([]EmailWithSnippets, len(results.Results))
				for i, result := range results.Results {
					var snippets []SnippetWithoutEmailId
					if makesSnippets {
						snippets := make([]SnippetWithoutEmailId, len(result.Snippets))
						for j, snippet := range result.Snippets {
							snippets[j] = SnippetWithoutEmailId{
								Subject: snippet.Subject,
								Preview: snippet.Preview,
							}
						}
					} else {
						snippets = nil
					}
					sanitized, err := req.sanitizeEmail(result.Email)
					if err != nil {
						return req.error(accountId, err)
					}
					flattened[i] = EmailWithSnippets{
						Email:    sanitized,
						Snippets: snippets,
					}
				}

				var total *uint = nil
				if calculateTotal {
					total = &results.Total
				}

				return req.respond(accountId, EmailWithSnippetsSearchResults{
					Results:    flattened,
					Total:      total,
					Position:   results.Position,
					Limit:      results.Limit,
					QueryState: results.QueryState,
				}, sessionState, EmailResponseObjectType, state, lang)
			} else {
				return req.notFound(accountId, sessionState, EmailResponseObjectType, state)
			}
		})
	}
}

func (g *Groupware) GetEmailsForAllAccounts(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		allAccountIds := req.AllAccountIds()

		ok, filter, makesSnippets, offset, limit, logger, err := g.buildEmailFilter(req)
		if !ok {
			return req.errorN(allAccountIds, err)
		}
		logger = log.From(req.logger.With().Array(logAccountId, log.SafeStringArray(allAccountIds)))
		ctx := req.ctx.WithLogger(logger)

		if !filter.IsNotEmpty() {
			filter = nil
		}

		if makesSnippets {
			resultsByAccountId, sessionState, state, lang, jerr := g.jmap.QueryEmailSnippets(allAccountIds, filter, offset, limit, ctx)
			if jerr != nil {
				return req.jmapErrorN(allAccountIds, jerr, sessionState, lang)
			}

			var totalOverAllAccounts uint = 0
			total := 0
			for _, results := range resultsByAccountId {
				if results.Total != nil {
					totalOverAllAccounts += *results.Total
				}
				total += len(results.Results)
			}

			flattened := make([]Snippet, total)
			{
				i := 0
				for accountId, results := range resultsByAccountId {
					for _, result := range results.Results {
						flattened[i] = Snippet{
							AccountId:             accountId,
							SearchSnippetWithMeta: result,
						}
					}
				}
			}

			slices.SortFunc(flattened, func(a, b Snippet) int { return a.ReceivedAt.Compare(b.ReceivedAt) })

			// TODO offset and limit over the aggregated results by account

			body := EmailSearchSnippetsResults{
				Results:    flattened,
				Total:      totalOverAllAccounts,
				Limit:      limit,
				QueryState: state,
			}

			return req.respondN(allAccountIds, body, sessionState, EmailResponseObjectType, state, lang)
		} else {
			withThreads := true

			resultsByAccountId, sessionState, state, lang, jerr := g.jmap.QueryEmailSummaries(allAccountIds, filter, limit, withThreads, ctx)
			if jerr != nil {
				return req.jmapErrorN(allAccountIds, jerr, sessionState, lang)
			}

			var totalAcrossAllAccounts uint = 0
			total := 0
			for _, results := range resultsByAccountId {
				totalAcrossAllAccounts += results.Total
				total += len(results.Emails)
			}

			flattened := make([]jmap.Email, total)
			{
				i := 0
				for accountId, results := range resultsByAccountId {
					for _, result := range results.Emails {
						result.AccountId = accountId
						flattened[i] = result
						i++
					}
				}
			}

			slices.SortFunc(flattened, func(a, b jmap.Email) int { return a.ReceivedAt.Compare(b.ReceivedAt) })

			// TODO offset and limit over the aggregated results by account

			body := EmailSearchResults{
				Results:    flattened,
				Total:      totalAcrossAllAccounts,
				Limit:      limit,
				QueryState: state,
			}

			return req.respondN(allAccountIds, body, sessionState, EmailResponseObjectType, state, lang)
		}
	})
}

var draftEmailAutoMailboxRolePrecedence = []string{
	jmap.JmapMailboxRoleDrafts, // we want draft emails to be created in the Mailbox with the drafts role
	jmap.JmapMailboxRoleInbox,  // but if there is none, we will use the Mailbox with the inbox role instead
}

func findDraftsMailboxId(j *jmap.Client, accountId string, req Request, ctx jmap.Context) (string, Response) {
	mailboxIdsPerAccountIds, sessionState, _, lang, jerr := j.SearchMailboxIdsPerRole(single(accountId), draftEmailAutoMailboxRolePrecedence, ctx)
	if jerr != nil {
		return "", req.jmapError(accountId, jerr, sessionState, lang)
	} else {
		for _, role := range draftEmailAutoMailboxRolePrecedence {
			if mailboxId, ok := mailboxIdsPerAccountIds[accountId][role]; ok {
				return mailboxId, Response{}
			}
		}
		// couldn't find a Mailbox with the drafts role for that account,
		// we have to return an error... ?
		return "", req.error(accountId, apiError(req.errorId(), ErrorNoMailboxWithDraftRole))
	}
}

var sentEmailAutoMailboxRolePrecedence = []string{
	jmap.JmapMailboxRoleSent,  // we want sent emails to be created in the Mailbox with the sent role
	jmap.JmapMailboxRoleInbox, // but if there is none, we will use the Mailbox with the inbox role instead
}

var draftAndSentMailboxRoles = structs.Uniq(structs.Concat(draftEmailAutoMailboxRolePrecedence, sentEmailAutoMailboxRolePrecedence))

func findSentMailboxId(j *jmap.Client, accountId string, req Request, ctx jmap.Context) (string, string, Response) { //NOSONAR
	mailboxIdsPerAccountIds, sessionState, _, lang, jerr := j.SearchMailboxIdsPerRole(single(accountId), draftAndSentMailboxRoles, ctx)
	if jerr != nil {
		return "", "", req.jmapError(accountId, jerr, sessionState, lang)
	} else {
		sentMailboxId := ""
		for _, role := range sentEmailAutoMailboxRolePrecedence {
			if mailboxId, ok := mailboxIdsPerAccountIds[accountId][role]; ok {
				sentMailboxId = mailboxId
				break
			}
		}
		if sentMailboxId == "" {
			return "", "", req.error(accountId, apiError(req.errorId(), ErrorNoMailboxWithSentRole))
		}
		draftsMailboxId := ""
		for _, role := range draftEmailAutoMailboxRolePrecedence {
			if mailboxId, ok := mailboxIdsPerAccountIds[accountId][role]; ok {
				draftsMailboxId = mailboxId
				break
			}
		}
		if draftsMailboxId == "" {
			return "", "", req.error(accountId, apiError(req.errorId(), ErrorNoMailboxWithDraftRole))
		}
		return draftsMailboxId, sentMailboxId, Response{}
	}
}

func (g *Groupware) CreateEmail(w http.ResponseWriter, r *http.Request) {
	create(Email, w, r, g,
		func(r Request, accountId string, body *jmap.EmailChange, ctx jmap.Context) (bool, Response) {
			if len(body.MailboxIds) < 1 {
				mailboxId, resp := findDraftsMailboxId(g.jmap, accountId, r, ctx)
				if mailboxId != "" {
					body.MailboxIds[mailboxId] = true
				} else {
					return false, resp
				}
			}
			return true, Response{}
		},
		func(accountId string, body jmap.EmailChange, ctx jmap.Context) (*jmap.Email, jmap.SessionState, jmap.State, jmap.Language, jmap.Error) {
			return g.jmap.CreateEmail(accountId, body, "", ctx)
		},
	)
}

func (g *Groupware) ReplaceEmail(w http.ResponseWriter, r *http.Request) {
	replaceId := ""
	create(Email, w, r, g,
		func(r Request, accountId string, body *jmap.EmailChange, ctx jmap.Context) (bool, Response) {
			if len(body.MailboxIds) < 1 {
				mailboxId, resp := findDraftsMailboxId(g.jmap, accountId, r, ctx)
				if mailboxId != "" {
					body.MailboxIds[mailboxId] = true
				} else {
					return false, resp
				}
			}
			var err *Error
			replaceId, err = r.PathParam(UriParamEmailId)
			if err != nil {
				return false, r.error(accountId, err)
			}

			return true, Response{}
		},
		func(accountId string, body jmap.EmailChange, ctx jmap.Context) (*jmap.Email, jmap.SessionState, jmap.State, jmap.Language, jmap.Error) {
			ctx = ctx.WithLogger(log.From(ctx.Logger.With().Str("replaceId", replaceId)))
			return g.jmap.CreateEmail(accountId, body, replaceId, ctx)
		},
	)
}

func (g *Groupware) UpdateEmail(w http.ResponseWriter, r *http.Request) {
	modify(Email, w, r, g, g.jmap.UpdateEmail)
}

type emailKeywordUpdates struct {
	Add    []string `json:"add,omitempty"`
	Remove []string `json:"remove,omitempty"`
}

func (e emailKeywordUpdates) IsEmpty() bool {
	return len(e.Add) == 0 && len(e.Remove) == 0
}

// Update the keywords of an email by its identifier.
//
// @api:tags email
func (g *Groupware) UpdateEmailKeywords(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return req.error(accountId, gwerr)
		}
		l.Str(logAccountId, accountId)

		emailId, err := req.PathParam(UriParamEmailId)
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(UriParamEmailId, log.SafeString(emailId))

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)

		var body emailKeywordUpdates
		err = req.body(&body)
		if err != nil {
			return req.error(accountId, err)
		}

		if body.IsEmpty() {
			return req.noop(accountId)
		}

		patch := jmap.PatchObject{}
		for _, keyword := range body.Add {
			patch["keywords/"+keyword] = true //NOSONAR
		}
		for _, keyword := range body.Remove {
			patch["keywords/"+keyword] = nil //NOSONAR
		}
		patches := map[string]jmap.PatchObject{
			emailId: patch,
		}

		result, sessionState, state, lang, jerr := g.jmap.UpdateEmails(accountId, patches, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		if result == nil {
			return req.error(accountId, apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Missing Email Update Response", //NOSONAR
				"An internal API behaved unexpectedly: missing Email update response from JMAP endpoint"))) //NOSONAR
		}
		updatedEmail, ok := result[emailId]
		if !ok {
			return req.error(accountId, apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Wrong Email Update Response ID", //NOSONAR
				"An internal API behaved unexpectedly: wrong Email update ID response from JMAP endpoint"))) //NOSONAR
		}

		return req.respond(accountId, updatedEmail, sessionState, EmailResponseObjectType, state, lang)
	})
}

// Add keywords to an email by its unique identifier.
//
// @api:tags email
func (g *Groupware) AddEmailKeywords(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return req.error(accountId, gwerr)
		}
		l.Str(logAccountId, accountId)

		emailId, err := req.PathParam(UriParamEmailId)
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(UriParamEmailId, log.SafeString(emailId))

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)

		var body []string
		err = req.body(&body)
		if err != nil {
			return req.error(accountId, err)
		}

		if len(body) < 1 {
			return req.noop(accountId)
		}

		patch := jmap.PatchObject{}
		for _, keyword := range body {
			patch["keywords/"+keyword] = true
		}
		patches := map[string]jmap.PatchObject{
			emailId: patch,
		}

		result, sessionState, state, lang, jerr := g.jmap.UpdateEmails(accountId, patches, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		if result == nil {
			return req.error(accountId, apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Missing Email Update Response",
				"An internal API behaved unexpectedly: missing Email update response from JMAP endpoint")))
		}
		updatedEmail, ok := result[emailId]
		if !ok {
			return req.error(accountId, apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Wrong Email Update Response ID",
				"An internal API behaved unexpectedly: wrong Email update ID response from JMAP endpoint")))
		}

		if updatedEmail == nil {
			return req.noContent(accountId, sessionState, EmailResponseObjectType, state)
		} else {
			return req.respond(accountId, updatedEmail, sessionState, EmailResponseObjectType, state, lang)
		}
	})
}

// Remove keywords of an email by its unique identifier.
//
// @api:tags email
func (g *Groupware) RemoveEmailKeywords(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(logAccountId, accountId)

		emailId, err := req.PathParam(UriParamEmailId)
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(UriParamEmailId, log.SafeString(emailId))

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)

		var body []string
		err = req.body(&body)
		if err != nil {
			return req.error(accountId, err)
		}

		if len(body) < 1 {
			return req.noop(accountId)
		}

		patch := jmap.PatchObject{}
		for _, keyword := range body {
			patch["keywords/"+keyword] = nil
		}
		patches := map[string]jmap.PatchObject{
			emailId: patch,
		}

		result, sessionState, state, lang, jerr := g.jmap.UpdateEmails(accountId, patches, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		if result == nil {
			return req.error(accountId, apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Missing Email Update Response",
				"An internal API behaved unexpectedly: missing Email update response from JMAP endpoint")))
		}
		updatedEmail, ok := result[emailId]
		if !ok {
			return req.error(accountId, apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Wrong Email Update Response ID",
				"An internal API behaved unexpectedly: wrong Email update ID response from JMAP endpoint")))
		}

		if updatedEmail == nil {
			return req.noContent(accountId, sessionState, EmailResponseObjectType, state)
		} else {
			return req.respond(accountId, updatedEmail, sessionState, EmailResponseObjectType, state, lang)
		}
	})
}

// Delete an email by its unique identifier.
func (g *Groupware) DeleteEmail(w http.ResponseWriter, r *http.Request) {
	delete(Email, w, r, g, g.jmap.DeleteEmails)
}

// Delete a set of emails by their unique identifiers.
//
// The identifiers of the emails to delete are specified as part of the request
// body, as an array of strings.
func (g *Groupware) DeleteEmails(w http.ResponseWriter, r *http.Request) {
	deleteMany(Email, w, r, g, g.jmap.DeleteEmails)
}

func (g *Groupware) SendEmail(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return req.error(accountId, gwerr)
		}
		l.Str(logAccountId, accountId)

		emailId, err := req.PathParam(UriParamEmailId)
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(UriParamEmailId, log.SafeString(emailId))

		identityId, err := req.getMandatoryStringParam(QueryParamIdentityId)
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(QueryParamIdentityId, log.SafeString(identityId))

		var move *jmap.MoveMail = nil
		{
			moveFromMailboxId, _ := req.getStringParam(QueryParamMoveFromMailboxId, "")
			moveToMailboxId, _ := req.getStringParam(QueryParamMoveToMailboxId, "")

			if moveFromMailboxId == "" || moveToMailboxId == "" {
				ctx := req.ctx.WithLogger(log.From(l))
				draftsMailboxId, sentMailboxId, resp := findSentMailboxId(g.jmap, accountId, req, ctx)
				if draftsMailboxId != "" && sentMailboxId != "" {
					if moveFromMailboxId == "" {
						moveFromMailboxId = draftsMailboxId
					}
					if moveToMailboxId == "" {
						moveToMailboxId = sentMailboxId
					}
				} else {
					return resp
				}
			}

			// TODO some parameter to prevent moving the sent email from one Mailbox to another?

			move = &jmap.MoveMail{FromMailboxId: moveFromMailboxId, ToMailboxId: moveToMailboxId}
			l.Str(QueryParamMoveFromMailboxId, log.SafeString(moveFromMailboxId)).Str(QueryParamMoveToMailboxId, log.SafeString(moveFromMailboxId))
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		resp, sessionState, state, lang, jerr := g.jmap.SubmitEmail(accountId, identityId, emailId, move, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, resp, sessionState, EmailResponseObjectType, state, lang)
	})
}

type AboutEmailsEvent struct {
	Id       string        `json:"id"`
	Source   string        `json:"source"`
	Emails   []jmap.Email  `json:"emails"`
	Language jmap.Language `json:"lang"`
}

type AboutEmailResponse struct {
	Email     jmap.Email    `json:"email"`
	RequestId string        `json:"requestId"`
	Language  jmap.Language `json:"lang"`
}

func relatedEmailsFilter(email jmap.Email, beacon time.Time, days uint) jmap.EmailFilterElement {
	filters := []jmap.EmailFilterElement{}
	for _, from := range email.From {
		if from.Email != "" {
			filters = append(filters, jmap.EmailFilterCondition{From: from.Email})
		}
	}
	for _, sender := range email.Sender {
		if sender.Email != "" {
			filters = append(filters, jmap.EmailFilterCondition{From: sender.Email})
		}
	}

	var timeFilter jmap.EmailFilterCondition
	{
		if days > math.MaxInt64 {
			days = math.MaxInt64 // avoid gosec G115 (CWE-190)
		}
		hours := int64(days) * 24
		delta := time.Duration(hours) * time.Hour
		timeFilter = jmap.EmailFilterCondition{
			Before: beacon.Add(delta),
			After:  beacon.Add(-delta),
		}
	}

	var filter jmap.EmailFilterElement
	if len(filters) > 0 {
		filter = jmap.EmailFilterOperator{
			Operator: jmap.And,
			Conditions: []jmap.EmailFilterElement{
				timeFilter,
				jmap.EmailFilterOperator{
					Operator:   jmap.Or,
					Conditions: filters,
				},
			},
		}
	} else {
		filter = timeFilter
	}

	return filter
}

func (g *Groupware) RelatedToEmail(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(logAccountId, log.SafeString(accountId))

		id, err := req.PathParam(UriParamEmailId)
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(logEmailId, log.SafeString(id))

		limit, ok, err := req.parseUIntParam(QueryParamLimit, 10) // TODO configurable default limit
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Uint("limit", limit)
		}

		days, ok, err := req.parseUIntParam(QueryParamDays, 5) // TODO configurable default days
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Uint("days", days)
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)

		reqId := req.GetRequestId()
		getEmailsBefore := time.Now()
		emails, _, sessionState, state, lang, jerr := g.jmap.GetEmails(accountId, single(id), true, g.config.maxBodyValueBytes, false, false, ctx)
		getEmailsDuration := time.Since(getEmailsBefore)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		if len(emails) < 1 {
			req.observe(g.metrics.EmailByIdDuration.WithLabelValues(req.session.JmapEndpoint, metrics.Values.Result.NotFound), getEmailsDuration.Seconds())
			logger.Trace().Msg("failed to find any emails matching id") // the id is already in the log field
			return req.notFound(accountId, sessionState, EmailResponseObjectType, state)
		} else {
			req.observe(g.metrics.EmailByIdDuration.WithLabelValues(req.session.JmapEndpoint, metrics.Values.Result.Found), getEmailsDuration.Seconds())
		}

		email := emails[0]

		beacon := email.ReceivedAt // TODO configurable: either relative to when the email was received, or relative to now
		//beacon := time.Now()
		filter := relatedEmailsFilter(email, beacon, days)

		// bgctx, _ := context.WithTimeout(context.Background(), time.Duration(30)*time.Second) // TODO configurable
		bgctx := context.Background()

		g.job(logger, RelationTypeSameSender, func(jobId uint64, l *log.Logger) {
			before := time.Now()
			ctx = ctx.WithLogger(logger).WithContext(bgctx)
			resultsByAccountId, _, _, lang, jerr := g.jmap.QueryEmails(single(accountId), filter, 0, limit, false, g.config.maxBodyValueBytes, ctx)
			if results, ok := resultsByAccountId[accountId]; ok {
				duration := time.Since(before)
				if jerr != nil {
					_ = req.observeJmapError(jerr)
					l.Error().Err(jerr).Msgf("failed to query %v emails", RelationTypeSameSender)
				} else {
					req.observe(g.metrics.EmailSameSenderDuration.WithLabelValues(req.session.JmapEndpoint), duration.Seconds())
					related, err := req.sanitizeEmails(filterEmails(results.Emails, email))
					if err == nil {
						l.Trace().Msgf("'%v' found %v other emails", RelationTypeSameSender, len(related))
						if len(related) > 0 {
							req.push(RelationEntityEmail, AboutEmailsEvent{Id: reqId, Emails: related, Source: RelationTypeSameSender, Language: lang})
						}
					}
				}
			}
		})

		g.job(logger, RelationTypeSameThread, func(jobId uint64, l *log.Logger) {
			before := time.Now()
			ctx = ctx.WithLogger(logger).WithContext(bgctx)
			emails, _, _, lang, jerr := g.jmap.EmailsInThread(accountId, email.ThreadId, false, g.config.maxBodyValueBytes, ctx)
			duration := time.Since(before)
			if jerr != nil {
				_ = req.observeJmapError(jerr)
				l.Error().Err(jerr).Msgf("failed to list %v emails", RelationTypeSameThread)
			} else {
				req.observe(g.metrics.EmailSameThreadDuration.WithLabelValues(req.session.JmapEndpoint), duration.Seconds())
				related, err := req.sanitizeEmails(filterEmails(emails, email))
				if err == nil {
					l.Trace().Msgf("'%v' found %v other emails", RelationTypeSameThread, len(related))
					if len(related) > 0 {
						req.push(RelationEntityEmail, AboutEmailsEvent{Id: reqId, Emails: related, Source: RelationTypeSameThread, Language: lang})
					}
				}
			}
		})

		sanitized, err := req.sanitizeEmail(email)
		if err != nil {
			return req.error(accountId, err)
		}
		return req.respond(accountId, AboutEmailResponse{
			Email:     sanitized,
			RequestId: reqId,
		}, sessionState, EmailResponseObjectType, state, lang)
	})
}

type EmailSummary struct {
	// The id of the account this Email summary pertains to.
	AccountId string `json:"accountId,omitempty"`

	// The id of the Email object.
	//
	// Note that this is the JMAP object id, NOT the Message-ID header field value of the message [RFC5322].
	Id string `json:"id,omitempty"`

	// The id of the Thread to which this Email belongs.
	ThreadId string `json:"threadId,omitempty"`

	// The number of emails in the thread, including this one.
	ThreadSize int `json:"threadSize,omitzero"`

	// The set of Mailbox ids this Email belongs to.
	//
	// An Email in the mail store MUST belong to one or more Mailboxes at all times (until it is destroyed).
	// The set is represented as an object, with each key being a Mailbox id.
	//
	// The value for each key in the object MUST be true.
	MailboxIds map[string]bool `json:"mailboxIds,omitempty"`

	// A set of keywords that apply to the Email.
	//
	// The set is represented as an object, with the keys being the keywords.
	//
	// The value for each key in the object MUST be true.
	//
	// Keywords are shared with IMAP.
	//
	// The six system keywords from IMAP get special treatment.
	//
	// The following four keywords have their first character changed from \ in IMAP to $ in JMAP and have particular semantic meaning:
	//
	//   - $draft: The Email is a draft the user is composing.
	//   - $seen: The Email has been read.
	//   - $flagged: The Email has been flagged for urgent/special attention.
	//   - $answered: The Email has been replied to.
	//
	// The IMAP \Recent keyword is not exposed via JMAP. The IMAP \Deleted keyword is also not present: IMAP uses a delete+expunge model,
	// which JMAP does not. Any message with the \Deleted keyword MUST NOT be visible via JMAP (and so are not counted in the
	// “totalEmails”, “unreadEmails”, “totalThreads”, and “unreadThreads” Mailbox properties).
	//
	// Users may add arbitrary keywords to an Email.
	// For compatibility with IMAP, a keyword is a case-insensitive string of 1–255 characters in the ASCII subset
	// %x21–%x7e (excludes control chars and space), and it MUST NOT include any of these characters:
	//
	//    ( ) { ] % * " \
	//
	// Because JSON is case sensitive, servers MUST return keywords in lowercase.
	//
	// The [IMAP and JMAP Keywords](https://www.iana.org/assignments/imap-jmap-keywords/) registry as established in
	// [RFC5788] assigns semantic meaning to some other keywords in common use.
	//
	// New keywords may be established here in the future. In particular, note:
	//
	//   - $forwarded: The Email has been forwarded.
	//   - $phishing: The Email is highly likely to be phishing.
	//     Clients SHOULD warn users to take care when viewing this Email and disable links and attachments.
	//   - $junk: The Email is definitely spam.
	//     Clients SHOULD set this flag when users report spam to help train automated spam-detection systems.
	//   - $notjunk: The Email is definitely not spam.
	//     Clients SHOULD set this flag when users indicate an Email is legitimate, to help train automated spam-detection systems.
	Keywords map[string]bool `json:"keywords,omitempty"`

	// The size, in octets, of the raw data for the message [RFC5322]
	// (as referenced by the blobId, i.e., the number of octets in the file the user would download).
	Size int `json:"size"`

	// The date the Email was received by the message store.
	//
	// This is the internal date in IMAP [RFC3501].
	ReceivedAt time.Time `json:"receivedAt,omitzero"`

	// The value is identical to the value of `header:Sender:asAddresses`.
	Sender []jmap.EmailAddress `json:"sender,omitempty"`

	// The value is identical to the value of `header:From:asAddresses`.
	From []jmap.EmailAddress `json:"from,omitempty"`

	// The value is identical to the value of `header:To:asAddresses`.
	To []jmap.EmailAddress `json:"to,omitempty"`

	// The value is identical to the value of `header:Cc:asAddresses`.
	Cc []jmap.EmailAddress `json:"cc,omitempty"`

	// The value is identical to the value of `header:Bcc:asAddresses`.
	Bcc []jmap.EmailAddress `json:"bcc,omitempty"`

	// The value is identical to the value of `header:Subject:asText`.
	Subject string `json:"subject,omitempty"`

	// The value is identical to the value of `header:Date:asDate`.
	SentAt time.Time `json:"sentAt,omitzero"`

	// This is true if there are one or more parts in the message that a client UI should offer as downloadable.
	//
	// A server SHOULD set hasAttachment to true if the attachments list contains at least one item that
	// does not have Content-Disposition: inline.
	//
	// The server MAY ignore parts in this list that are processed automatically in some way or are referenced
	// as embedded images in one of the text/html parts of the message.
	//
	// The server MAY set hasAttachment based on implementation-defined or site-configurable heuristics.
	HasAttachment bool `json:"hasAttachment,omitempty"`

	// A list, traversing depth-first, of all parts in bodyStructure.
	//
	// They must satisfy either of the following conditions:
	//
	// - not of type `multipart/*` and not included in textBody or htmlBody
	// - of type `image/*`, `audio/*`, or `video/*` and not in both textBody and htmlBody
	//
	// None of these parts include subParts, including `message/*` types.
	//
	// Attached messages may be fetched using the Email/parse method and the blobId.
	//
	// Note that a `text/html` body part HTML may reference image parts in attachments by using cid:
	// links to reference the Content-Id, as defined in [RFC2392], or by referencing the Content-Location.
	Attachments []jmap.EmailBodyPart `json:"attachments,omitempty"`

	// A plaintext fragment of the message body.
	//
	// This is intended to be shown as a preview line when listing messages in the mail store and may be truncated
	// when shown.
	//
	// The server may choose which part of the message to include in the preview; skipping quoted sections and
	// salutations and collapsing white space can result in a more useful preview.
	//
	// This MUST NOT be more than 256 characters in length.
	//
	// As this is derived from the message content by the server, and the algorithm for doing so could change over
	// time, fetching this for an Email a second time MAY return a different result.
	// However, the previous value is not considered incorrect, and the change SHOULD NOT cause the Email object
	// to be considered as changed by the server.
	Preview string `json:"preview,omitempty"`
}

func summarizeEmail(accountId string, email jmap.Email) EmailSummary {
	return EmailSummary{
		AccountId:     accountId,
		Id:            email.Id,
		ThreadId:      email.ThreadId,
		ThreadSize:    email.ThreadSize,
		MailboxIds:    email.MailboxIds,
		Keywords:      email.Keywords,
		Size:          email.Size,
		ReceivedAt:    email.ReceivedAt,
		Sender:        email.Sender,
		From:          email.From,
		To:            email.To,
		Cc:            email.Cc,
		Bcc:           email.Bcc,
		Subject:       email.Subject,
		SentAt:        email.SentAt,
		HasAttachment: email.HasAttachment,
		Attachments:   email.Attachments,
		Preview:       email.Preview,
	}
}

type emailWithAccountId struct {
	accountId string
	email     jmap.Email
}

type EmailSummaries struct {
	Emails []EmailSummary `json:"emails,omitempty"`
	Total  uint           `json:"total,omitzero"`
	Limit  uint           `json:"limit,omitzero"`
	Offset uint           `json:"offset,omitzero"`
	State  jmap.State     `json:"state,omitempty"`
}

// Get a summary of the latest emails across all the mailboxes, across all of a user's accounts.
//
// Retrieves summaries of the latest emails of a user, in all accounts, across all mailboxes.
//
// The number of total summaries to retrieve is specified using the query parameter `limit`.
//
// The following additional query parameters may be specified to further filter the emails to summarize:
//
// * `seen`: when `true`, emails that have already been seen (read) will be included as well (default is to only include emails that have not been read yet)
// * `undesirable`: when `true`, emails that are flagged as spam or phishing will also be summarized (default is to ignore those)
func (g *Groupware) GetLatestEmailsSummaryForAllAccounts(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()

		allAccountIds := req.AllAccountIds()
		l.Array(logAccountId, log.SafeStringArray(allAccountIds))

		limit, ok, err := req.parseUIntParam(QueryParamLimit, 10) // TODO from configuration
		if err != nil {
			return req.errorN(allAccountIds, err)
		}
		if ok {
			l = l.Uint(QueryParamLimit, limit)
		}

		offset, ok, err := req.parseUIntParam(QueryParamOffset, 0)
		if err != nil {
			return req.errorN(allAccountIds, err)
		}
		if offset > 0 {
			return req.notImplementedN(allAccountIds, EmailResponseObjectType)
		}
		if ok {
			l = l.Uint(QueryParamOffset, limit)
		}

		seen, ok, err := req.parseBoolParam(QueryParamSeen, false)
		if err != nil {
			return req.errorN(allAccountIds, err)
		}
		if ok {
			l = l.Bool(QueryParamSeen, seen)
		}

		undesirable, ok, err := req.parseBoolParam(QueryParamUndesirable, false)
		if err != nil {
			return req.errorN(allAccountIds, err)
		}
		if ok {
			l = l.Bool(QueryParamUndesirable, undesirable)
		}

		var filter jmap.EmailFilterElement = nil // all emails, read and unread
		{
			notKeywords := []string{}
			if !seen {
				notKeywords = append(notKeywords, jmap.JmapKeywordSeen)
			}
			if undesirable {
				notKeywords = append(notKeywords, jmap.JmapKeywordJunk, jmap.JmapKeywordPhishing)
			}
			filter = filterFromNotKeywords(notKeywords)
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)

		emailsSummariesByAccount, sessionState, state, lang, jerr := g.jmap.QueryEmailSummaries(allAccountIds, filter, limit, true, ctx)
		if jerr != nil {
			return req.jmapErrorN(allAccountIds, jerr, sessionState, lang)
		}

		// sort in memory to respect the overall limit
		total := uint(0)
		for _, emails := range emailsSummariesByAccount {
			total += uint(max(len(emails.Emails), 0))
		}
		all := make([]emailWithAccountId, total)
		i := uint(0)
		for accountId, emails := range emailsSummariesByAccount {
			for _, email := range emails.Emails {
				all[i] = emailWithAccountId{accountId: accountId, email: email}
				i++
			}
		}

		slices.SortFunc(all, func(a, b emailWithAccountId) int { return -(a.email.ReceivedAt.Compare(b.email.ReceivedAt)) })

		summaries := make([]EmailSummary, min(limit, total))
		for i = 0; i < limit && i < total; i++ {
			summaries[i] = summarizeEmail(all[i].accountId, all[i].email)
		}

		return req.respondN(allAccountIds, EmailSummaries{
			Emails: summaries,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		}, sessionState, EmailResponseObjectType, state, lang)
	})
}

func filterEmails(all []jmap.Email, skip jmap.Email) []jmap.Email {
	filtered := all[:0]
	for _, email := range all {
		if skip.Id != email.Id {
			filtered = append(filtered, email)
		}
	}
	return filtered
}

func filterFromNotKeywords(keywords []string) jmap.EmailFilterElement {
	switch len(keywords) {
	case 0:
		return nil
	case 1:
		return jmap.EmailFilterCondition{NotKeyword: keywords[0]}
	default:
		conditions := make([]jmap.EmailFilterElement, len(keywords))
		for i, keyword := range keywords {
			conditions[i] = jmap.EmailFilterCondition{NotKeyword: keyword}
		}
		return jmap.EmailFilterOperator{Operator: jmap.And, Conditions: conditions}
	}
}

var sanitizationPolicy *bluemonday.Policy = bluemonday.UGCPolicy()

var sanitizableMediaTypes = []string{
	"text/html",
	"text/xhtml",
}

func (r *Request) sanitizeEmail(source jmap.Email) (jmap.Email, *Error) { //NOSONAR
	if !r.g.config.sanitize {
		return source, nil
	}
	memory := map[string]int{}
	for _, ref := range []*[]jmap.EmailBodyPart{&source.HtmlBody, &source.TextBody} {
		newBody := make([]jmap.EmailBodyPart, len(*ref))
		for i, p := range *ref {
			t, _, err := mime.ParseMediaType(p.Type)
			if err != nil {
				msg := fmt.Sprintf("failed to parse the mime type '%s'", p.Type)
				r.logger.Error().Str("type", log.SafeString(p.Type)).Msg(msg)
				return source, r.apiError(&ErrorFailedToSanitizeEmail, withDetail(msg))
			}
			if slices.Contains(sanitizableMediaTypes, t) {
				if already, done := memory[p.PartId]; !done {
					if part, ok := source.BodyValues[p.PartId]; ok {
						safe := sanitizationPolicy.Sanitize(part.Value)
						part.Value = safe
						source.BodyValues[p.PartId] = part
						newLen := len(safe)
						memory[p.PartId] = newLen
						p.Size = newLen
					}
				} else {
					p.Size = already
				}
			}
			newBody[i] = p
		}
		*ref = newBody
	}

	// we could post-process attachments as well:
	/*
		for _, part := range source.Attachments {
			if part.Type == "" {
				part.Type = "application/octet-stream"
			}
			if part.Name == "" {
				part.Name = "unknown"
			}
		}
	*/

	return source, nil
}

func (r *Request) sanitizeEmails(source []jmap.Email) ([]jmap.Email, *Error) {
	if !r.g.config.sanitize {
		return source, nil
	}
	result := make([]jmap.Email, len(source))
	for i, email := range source {
		sanitized, gwerr := r.sanitizeEmail(email)
		if gwerr != nil {
			return nil, gwerr
		}
		result[i] = sanitized
	}
	return result, nil
}
