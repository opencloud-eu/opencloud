// Copyright 2018-2021 CERN
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

package ocdav

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/google/uuid"
	ocdavErrors "github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/prop"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/spacelookup"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"go.opentelemetry.io/otel/attribute"
)

// Most of this is taken from https://github.com/golang/net/blob/master/webdav/lock.go

// From RFC4918 http://www.webdav.org/specs/rfc4918.html#lock-tokens
// This specification encourages servers to create Universally Unique Identifiers (UUIDs) for lock tokens,
// and to use the URI form defined by "A Universally Unique Identifier (UUID) URN Namespace" ([RFC4122]).
// However, servers are free to use any URI (e.g., from another scheme) so long as it meets the uniqueness
// requirements. For example, a valid lock token might be constructed using the "opaquelocktoken" scheme
// defined in Appendix C.
//
// Example: "urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6"
//
// we stick to the recommendation and use the URN Namespace
const lockTokenPrefix = "urn:uuid:"

var requestLockRegex = regexp.MustCompile(`\(<(urn:uuid:[0-9a-fA-F-]+)>\)`)

// TODO(jfd) implement lock
// see Web Distributed Authoring and Versioning (WebDAV) Locking Protocol:
// https://www.greenbytes.de/tech/webdav/draft-reschke-webdav-locking-latest.html
// Webdav supports a Depth: infinity lock, wopi only needs locks on files

// https://www.greenbytes.de/tech/webdav/draft-reschke-webdav-locking-latest.html#write.locks.and.the.if.request.header
// [...] a lock token MUST be submitted in the If header for all locked resources
// that a method may interact with or the method MUST fail. [...]
/*
	COPY /~fielding/index.html HTTP/1.1
	Host: example.com
	Destination: http://example.com/users/f/fielding/index.html
	If: <http://example.com/users/f/fielding/index.html>
		(<opaquelocktoken:f81d4fae-7dec-11d0-a765-00a0c91e6bf6>)
*/

// http://www.webdav.org/specs/rfc4918.html#ELEMENT_lockinfo
type lockInfo struct {
	XMLName   xml.Name  `xml:"lockinfo"`
	Exclusive *struct{} `xml:"lockscope>exclusive"`
	Shared    *struct{} `xml:"lockscope>shared"`
	Write     *struct{} `xml:"locktype>write"`
	Owner     owner     `xml:"owner"`
	LockID    string    `xml:"locktoken>href"`
}

// http://www.webdav.org/specs/rfc4918.html#ELEMENT_owner
type owner struct {
	InnerXML string `xml:",innerxml"`
}

// Condition can match a WebDAV resource, based on a token or ETag.
// Exactly one of Token and ETag should be non-empty.
type Condition struct {
	Not   bool
	Token string
	ETag  string
}

// LockSystem manages access to a collection of named resources. The elements
// in a lock name are separated by slash ('/', U+002F) characters, regardless
// of host operating system convention.
type LockSystem interface {
	// Confirm confirms that the caller can claim all of the locks specified by
	// the given conditions, and that holding the union of all of those locks
	// gives exclusive access to all of the named resources. Up to two resources
	// can be named. Empty names are ignored.
	//
	// Exactly one of release and err will be non-nil. If release is non-nil,
	// all of the requested locks are held until release is called. Calling
	// release does not unlock the lock, in the WebDAV UNLOCK sense, but once
	// Confirm has confirmed that a lock claim is valid, that lock cannot be
	// Confirmed again until it has been released.
	//
	// If Confirm returns ErrConfirmationFailed then the Handler will continue
	// to try any other set of locks presented (a WebDAV HTTP request can
	// present more than one set of locks). If it returns any other non-nil
	// error, the Handler will write a "500 Internal Server Error" HTTP status.
	Confirm(ctx context.Context, now time.Time, name0, name1 string, conditions ...Condition) (release func(), err error)

	// Create creates a lock with the given depth, duration, owner and root
	// (name). The depth will either be negative (meaning infinite) or zero.
	//
	// If Create returns ErrLocked then the Handler will write a "423 Locked"
	// HTTP status. If it returns any other non-nil error, the Handler will
	// write a "500 Internal Server Error" HTTP status.
	//
	// See http://www.webdav.org/specs/rfc4918.html#rfc.section.9.10.6 for
	// when to use each error.
	//
	// The token returned identifies the created lock. It should be an absolute
	// URI as defined by RFC 3986, Section 4.3. In particular, it should not
	// contain whitespace.
	Create(ctx context.Context, now time.Time, details LockDetails) (token string, err error)

	// Refresh refreshes the lock with the given token.
	//
	// If Refresh returns ErrLocked then the Handler will write a "423 Locked"
	// HTTP Status. If Refresh returns ErrNoSuchLock then the Handler will write
	// a "412 Precondition Failed" HTTP Status. If it returns any other non-nil
	// error, the Handler will write a "500 Internal Server Error" HTTP status.
	//
	// See http://www.webdav.org/specs/rfc4918.html#rfc.section.9.10.6 for
	// when to use each error.
	Refresh(ctx context.Context, now time.Time, ref *provider.Reference, token string) error

	// Unlock unlocks the lock with the given token.
	//
	// If Unlock returns ErrForbidden then the Handler will write a "403
	// Forbidden" HTTP Status. If Unlock returns ErrLocked then the Handler
	// will write a "423 Locked" HTTP status. If Unlock returns ErrNoSuchLock
	// then the Handler will write a "409 Conflict" HTTP Status. If it returns
	// any other non-nil error, the Handler will write a "500 Internal Server
	// Error" HTTP status.
	//
	// See http://www.webdav.org/specs/rfc4918.html#rfc.section.9.11.1 for
	// when to use each error.
	Unlock(ctx context.Context, now time.Time, ref *provider.Reference, token string) error
}

// NewCS3LS returns a new CS3 based LockSystem.
func NewCS3LS(s pool.Selectable[gateway.GatewayAPIClient]) LockSystem {
	return &cs3LS{
		selector: s,
	}
}

type cs3LS struct {
	selector pool.Selectable[gateway.GatewayAPIClient]
}

func (cls *cs3LS) Confirm(ctx context.Context, now time.Time, name0, name1 string, conditions ...Condition) (func(), error) {
	return nil, ocdavErrors.ErrNotImplemented
}

func (cls *cs3LS) Create(ctx context.Context, now time.Time, details LockDetails) (string, error) {
	// always assume depth infinity?
	/*
		if !details.ZeroDepth {
		 The CS3 Lock api currently has no depth property, it only locks single resources
			return "", ocdavErrors.ErrUnsupportedLockInfo
		}
	*/

	u := ctxpkg.ContextMustGetUser(ctx)

	// add metadata via opaque
	// TODO: upate cs3api: https://github.com/cs3org/cs3apis/issues/213
	o := utils.AppendPlainToOpaque(nil, "lockownername", u.GetDisplayName())
	o = utils.AppendPlainToOpaque(o, "locktime", now.Format(time.RFC3339))

	lockid := details.LockID
	if lockid == "" {
		// Having a lock token provides no special access rights. Anyone can find out anyone
		// else's lock token by performing lock discovery. Locks must be enforced based upon
		// whatever authentication mechanism is used by the server, not based on the secrecy
		// of the token values.
		// see: http://www.webdav.org/specs/rfc2518.html#n-lock-tokens
		token := uuid.New()

		lockid = lockTokenPrefix + token.String()
	}
	r := &provider.SetLockRequest{
		Ref: details.Root,
		Lock: &provider.Lock{
			Opaque: o,
			Type:   provider.LockType_LOCK_TYPE_EXCL,
			User:   details.UserID, // no way to set an app lock? TODO maybe via the ownerxml
			//AppName: , // TODO use a urn scheme?
			LockId: lockid,
		},
	}
	if details.Duration > 0 {
		expiration := time.Now().UTC().Add(details.Duration)
		r.Lock.Expiration = &types.Timestamp{
			Seconds: uint64(expiration.Unix()),
			Nanos:   uint32(expiration.Nanosecond()),
		}
	}

	client, err := cls.selector.Next()
	if err != nil {
		return "", err
	}

	res, err := client.SetLock(ctx, r)
	if err != nil {
		return "", err
	}
	switch res.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
		return lockid, nil
	default:
		return "", ocdavErrors.NewErrFromStatus(res.GetStatus())
	}

}

func (cls *cs3LS) Refresh(ctx context.Context, now time.Time, ref *provider.Reference, token string) error {
	u := ctxpkg.ContextMustGetUser(ctx)

	// add metadata via opaque
	// TODO: upate cs3api: https://github.com/cs3org/cs3apis/issues/213
	o := utils.AppendPlainToOpaque(nil, "lockownername", u.GetDisplayName())
	o = utils.AppendPlainToOpaque(o, "locktime", now.Format(time.RFC3339))

	if token == "" {
		return errors.New("token is empty")
	}

	r := &provider.RefreshLockRequest{
		Ref: ref,
		Lock: &provider.Lock{
			Opaque: o,
			Type:   provider.LockType_LOCK_TYPE_EXCL,
			//AppName: , // TODO use a urn scheme?
			LockId: token,
			User:   u.GetId(),
		},
	}

	client, err := cls.selector.Next()
	if err != nil {
		return err
	}

	res, err := client.RefreshLock(ctx, r)
	if err != nil {
		return err
	}
	switch res.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
		return nil

	default:
		return ocdavErrors.NewErrFromStatus(res.GetStatus())
	}
}

func (cls *cs3LS) Unlock(ctx context.Context, now time.Time, ref *provider.Reference, token string) error {
	u := ctxpkg.ContextMustGetUser(ctx)

	r := &provider.UnlockRequest{
		Ref: ref,
		Lock: &provider.Lock{
			LockId: token, // can be a token or a Coded-URL
			User:   u.Id,
		},
	}

	client, err := cls.selector.Next()
	if err != nil {
		return err
	}

	res, err := client.Unlock(ctx, r)
	if err != nil {
		return err
	}

	newErr := ocdavErrors.NewErrFromStatus(res.GetStatus())
	if newErr != nil {
		appctx.GetLogger(ctx).Error().Str("token", token).Interface("unlock", ref).Msg("could not unlock " + res.GetStatus().GetMessage())
	}
	return newErr
}

// LockDetails are a lock's metadata.
type LockDetails struct {
	// Root is the root resource name being locked. For a zero-depth lock, the
	// root is the only resource being locked.
	Root *provider.Reference
	// Duration is the lock timeout. A negative duration means infinite.
	Duration time.Duration
	// OwnerXML is the verbatim <owner> XML given in a LOCK HTTP request.
	//
	// TODO: does the "verbatim" nature play well with XML namespaces?
	// Does the OwnerXML field need to have more structure? See
	// https://codereview.appspot.com/175140043/#msg2
	OwnerXML string
	UserID   *userpb.UserId
	// ZeroDepth is whether the lock has zero depth. If it does not have zero
	// depth, it has infinite depth.
	ZeroDepth bool
	// OwnerName is the name of the lock owner
	OwnerName string
	// Locktime is the time the lock was created
	Locktime time.Time
	// LockID is the lock token
	LockID string
}

func readLockInfo(r io.Reader) (li lockInfo, status int, err error) {
	c := &countingReader{r: r}
	if err = xml.NewDecoder(c).Decode(&li); err != nil {
		if err == io.EOF {
			if c.n == 0 {
				// An empty body means to refresh the lock.
				// http://www.webdav.org/specs/rfc4918.html#refreshing-locks
				return lockInfo{}, 0, nil
			}
			err = ocdavErrors.ErrInvalidLockInfo
		}
		return lockInfo{}, http.StatusBadRequest, err
	}
	// We only support exclusive (non-shared) write locks. In practice, these are
	// the only types of locks that seem to matter.
	// We are ignoring the any properties in the lock details, and assume an exclusive write lock is requested.
	// https://datatracker.ietf.org/doc/html/rfc4918#section-7 only describes write locks
	//
	// if li.Exclusive == nil || li.Shared != nil {
	//   return lockInfo{}, http.StatusNotImplemented, errors.ErrUnsupportedLockInfo
	// }
	// what should we return if the user requests a shared lock? or leaves out the locktype? the testsuite will only send the property lockscope, not locktype
	// the oc tests cover both shared and exclusive locks. What is the WOPI lock? a shared or an exclusive lock?
	// since it is issued by a service it seems to be an exclusive lock.
	// the owner could be a link to the collaborative app ... to join the session
	return li, 0, nil
}

type countingReader struct {
	n int
	r io.Reader
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

const infiniteTimeout = -1

// parseTimeout parses the Timeout HTTP header, as per section 10.7. If s is
// empty, an infiniteTimeout is returned.
func parseTimeout(s string) (time.Duration, error) {
	if s == "" {
		return infiniteTimeout, nil
	}
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if s == "Infinite" {
		return infiniteTimeout, nil
	}
	const pre = "Second-"
	if !strings.HasPrefix(s, pre) {
		return 0, ocdavErrors.ErrInvalidTimeout
	}
	s = s[len(pre):]
	if s == "" || s[0] < '0' || '9' < s[0] {
		return 0, ocdavErrors.ErrInvalidTimeout
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || 1<<32-1 < n {
		return 0, ocdavErrors.ErrInvalidTimeout
	}
	return time.Duration(n) * time.Second, nil
}

const (
	infiniteDepth = -1
	invalidDepth  = -2
)

// parseDepth maps the strings "0", "1" and "infinity" to 0, 1 and
// infiniteDepth. Parsing any other string returns invalidDepth.
//
// Different WebDAV methods have further constraints on valid depths:
//   - PROPFIND has no further restrictions, as per section 9.1.
//   - COPY accepts only "0" or "infinity", as per section 9.8.3.
//   - MOVE accepts only "infinity", as per section 9.9.2.
//   - LOCK accepts only "0" or "infinity", as per section 9.10.3.
//
// These constraints are enforced by the handleXxx methods.
func parseDepth(s string) int {
	switch s {
	case "0":
		return 0
	case "1":
		return 1
	case "infinity":
		return infiniteDepth
	}
	return invalidDepth
}

/*
the oc 10 wopi app code locks like this:

	$storage->lockNodePersistent($file->getInternalPath(), [
		'token' => $wopiLock,
		'owner' => "{$user->getDisplayName()} via Office Online"
	]);

if owner is empty it defaults to '{displayname} ({email})', which is not a url ... but ... shrug

The LockManager also defaults to exclusive locks:

	$scope = ILock::LOCK_SCOPE_EXCLUSIVE;
	if (isset($lockInfo['scope'])) {
		$scope = $lockInfo['scope'];
	}
*/
func (s *svc) handleLock(w http.ResponseWriter, r *http.Request, ns string) (retStatus int, retErr error) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), fmt.Sprintf("%s %v", r.Method, r.URL.Path))
	defer span.End()

	span.SetAttributes(attribute.String("component", "ocdav"))

	fn := path.Join(ns, r.URL.Path) // TODO do we still need to jail if we query the registry about the spaces?

	// TODO instead of using a string namespace ns pass in the space with the request?
	ref, cs3Status, err := spacelookup.LookupReferenceForPath(ctx, s.gatewaySelector, fn)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if cs3Status.Code != rpc.Code_CODE_OK {
		return http.StatusInternalServerError, ocdavErrors.NewErrFromStatus(cs3Status)
	}

	return s.lockReference(ctx, w, r, ref)
}

func (s *svc) handleSpacesLock(w http.ResponseWriter, r *http.Request, spaceID string) (retStatus int, retErr error) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), fmt.Sprintf("%s %v", r.Method, r.URL.Path))
	defer span.End()

	span.SetAttributes(attribute.String("component", "ocdav"))

	ref, err := spacelookup.MakeStorageSpaceReference(spaceID, r.URL.Path)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid space id")
	}

	return s.lockReference(ctx, w, r, &ref)
}

func (s *svc) lockReference(ctx context.Context, w http.ResponseWriter, r *http.Request, ref *provider.Reference) (retStatus int, retErr error) {
	sublog := appctx.GetLogger(ctx).With().Interface("ref", ref).Logger()
	duration, err := parseTimeout(r.Header.Get(net.HeaderTimeout))
	if err != nil {
		return http.StatusBadRequest, ocdavErrors.ErrInvalidTimeout
	}

	li, status, err := readLockInfo(r.Body)
	if err != nil {
		return status, ocdavErrors.ErrInvalidLockInfo
	}

	u := ctxpkg.ContextMustGetUser(ctx)
	token, now, created := "", time.Now(), false
	ld := LockDetails{UserID: u.Id, Root: ref, Duration: duration, OwnerName: u.GetDisplayName(), Locktime: now, LockID: li.LockID}
	if li == (lockInfo{}) {
		// An empty lockInfo means to refresh the lock.
		ih, ok := parseIfHeader(r.Header.Get(net.HeaderIf))
		if !ok {
			return http.StatusBadRequest, ocdavErrors.ErrInvalidIfHeader
		}
		if len(ih.lists) == 1 && len(ih.lists[0].conditions) == 1 {
			token = ih.lists[0].conditions[0].Token
		}
		if token == "" {
			return http.StatusBadRequest, ocdavErrors.ErrInvalidLockToken
		}
		err = s.LockSystem.Refresh(ctx, now, ref, token)
		if err != nil {
			if err == ocdavErrors.ErrNoSuchLock {
				return http.StatusPreconditionFailed, err
			}
			return http.StatusInternalServerError, err
		}

		ld.LockID = token

	} else {
		// Section 9.10.3 says that "If no Depth header is submitted on a LOCK request,
		// then the request MUST act as if a "Depth:infinity" had been submitted."
		depth := infiniteDepth
		if hdr := r.Header.Get(net.HeaderDepth); hdr != "" {
			depth = parseDepth(hdr)
			if depth != 0 && depth != infiniteDepth {
				// Section 9.10.3 says that "Values other than 0 or infinity must not be
				// used with the Depth header on a LOCK method".
				return http.StatusBadRequest, ocdavErrors.ErrInvalidDepth
			}
		}
		/* our url path has been shifted, so we don't need to do this?
		reqPath, status, err := h.stripPrefix(r.URL.Path)
		if err != nil {
			return status, err
		}
		*/
		// TODO look up username and email
		//  if li.Owner.InnerXML == "" {
		//    // PHP version: 'owner' => "{$user->getDisplayName()} via Office Online"
		//    ld.OwnerXML = ld.UserID.OpaqueId
		//  }
		ld.OwnerXML = li.Owner.InnerXML // TODO optional, should be a URL
		ld.ZeroDepth = depth == 0

		//TODO: @jfd the code tries to create a lock for a file that may not even exist,
		//      should we do that in the decomposedfs as well? the node does not exist
		//      this actually is a name based lock ... ugh
		token, err = s.LockSystem.Create(ctx, now, ld)

		//
		if err != nil {
			switch {
			case errors.Is(err, ocdavErrors.ErrLocked):
				return http.StatusLocked, err
			case errors.Is(err, ocdavErrors.ErrForbidden):
				return http.StatusForbidden, err
			default:
				return http.StatusInternalServerError, err
			}
		}

		defer func() {
			if retErr != nil {
				if err := s.LockSystem.Unlock(ctx, now, ref, token); err != nil {
					appctx.GetLogger(ctx).Error().Err(err).Interface("lock", ld).Msg("could not unlock after failed lock")
				}
			}
		}()

		// Create the resource if it didn't previously exist.
		// TODO use sdk to stat?
		/*
			if _, err := s.FileSystem.Stat(ctx, reqPath); err != nil {
				f, err := s.FileSystem.OpenFile(ctx, reqPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
				if err != nil {
					// TODO: detect missing intermediate dirs and return http.StatusConflict?
					return http.StatusInternalServerError, err
				}
				f.Close()
				created = true
			}
		*/
		// http://www.webdav.org/specs/rfc4918.html#HEADER_Lock-Token says that the
		// Lock-Token value is a Coded-URL. We add angle brackets.
		w.Header().Set("Lock-Token", "<"+token+">")
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	if created {
		// This is "w.WriteHeader(http.StatusCreated)" and not "return
		// http.StatusCreated, nil" because we write our own (XML) response to w
		// and Handler.ServeHTTP would otherwise write "Created".
		w.WriteHeader(http.StatusCreated)
	}
	n, err := writeLockInfo(w, token, ld)
	if err != nil {
		sublog.Err(err).Int("bytes_written", n).Msg("error writing response")
	}
	return 0, nil
}

func writeLockInfo(w io.Writer, token string, ld LockDetails) (int, error) {
	depth := "infinity"
	if ld.ZeroDepth {
		depth = "0"
	}
	href := ld.Root.Path // FIXME add base url and space?

	lockdiscovery := strings.Builder{}
	lockdiscovery.WriteString(xml.Header)
	lockdiscovery.WriteString("<d:prop xmlns:d=\"DAV:\" xmlns:oc=\"http://owncloud.org/ns\"><d:lockdiscovery><d:activelock>\n")
	lockdiscovery.WriteString("  <d:locktype><d:write/></d:locktype>\n")
	lockdiscovery.WriteString("  <d:lockscope><d:exclusive/></d:lockscope>\n")
	lockdiscovery.WriteString(fmt.Sprintf("  <d:depth>%s</d:depth>\n", depth))
	if ld.OwnerXML != "" {
		lockdiscovery.WriteString(fmt.Sprintf("  <d:owner>%s</d:owner>\n", ld.OwnerXML))
	}
	if ld.Duration > 0 {
		timeout := ld.Duration / time.Second
		lockdiscovery.WriteString(fmt.Sprintf("  <d:timeout>Second-%d</d:timeout>\n", timeout))
	} else {
		lockdiscovery.WriteString("  <d:timeout>Infinite</d:timeout>\n")
	}
	if token != "" {
		lockdiscovery.WriteString(fmt.Sprintf("  <d:locktoken><d:href>%s</d:href></d:locktoken>\n", prop.Escape(token)))
	}
	if href != "" {
		lockdiscovery.WriteString(fmt.Sprintf("  <d:lockroot><d:href>%s</d:href></d:lockroot>\n", prop.Escape(href)))
	}
	if ld.OwnerName != "" {
		lockdiscovery.WriteString(fmt.Sprintf("  <oc:ownername>%s</oc:ownername>\n", prop.Escape(ld.OwnerName)))
	}
	if !ld.Locktime.IsZero() {
		lockdiscovery.WriteString(fmt.Sprintf("  <oc:locktime>%s</oc:locktime>\n", prop.Escape(ld.Locktime.Format(time.RFC3339))))
	}

	lockdiscovery.WriteString("</d:activelock></d:lockdiscovery></d:prop>")

	return fmt.Fprint(w, lockdiscovery.String())
}

func (s *svc) handleUnlock(w http.ResponseWriter, r *http.Request, ns string) (status int, err error) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), fmt.Sprintf("%s %v", r.Method, r.URL.Path))
	defer span.End()

	span.SetAttributes(attribute.String("component", "ocdav"))

	fn := path.Join(ns, r.URL.Path) // TODO do we still need to jail if we query the registry about the spaces?

	// TODO instead of using a string namespace ns pass in the space with the request?
	ref, cs3Status, err := spacelookup.LookupReferenceForPath(ctx, s.gatewaySelector, fn)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if cs3Status.Code != rpc.Code_CODE_OK {
		return http.StatusInternalServerError, ocdavErrors.NewErrFromStatus(cs3Status)
	}

	return s.unlockReference(ctx, w, r, ref)
}

func (s *svc) handleSpaceUnlock(w http.ResponseWriter, r *http.Request, spaceID string) (status int, err error) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), fmt.Sprintf("%s %v", r.Method, r.URL.Path))
	defer span.End()

	span.SetAttributes(attribute.String("component", "ocdav"))

	ref, err := spacelookup.MakeStorageSpaceReference(spaceID, r.URL.Path)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid space id")
	}

	return s.unlockReference(ctx, w, r, &ref)
}

func (s *svc) unlockReference(ctx context.Context, _ http.ResponseWriter, r *http.Request, ref *provider.Reference) (retStatus int, retErr error) {
	// http://www.webdav.org/specs/rfc4918.html#HEADER_Lock-Token says that the
	// Lock-Token value should be a Coded-URL OR a token. We strip its angle brackets.
	t := r.Header.Get(net.HeaderLockToken)
	if len(t) > 2 && t[0] == '<' && t[len(t)-1] == '>' {
		t = t[1 : len(t)-1]
	}

	err := s.LockSystem.Unlock(ctx, time.Now(), ref, t)
	switch {
	case err == nil:
		return http.StatusNoContent, nil
	case errors.Is(err, ocdavErrors.ErrLocked):
		return http.StatusLocked, err
	case errors.Is(err, ocdavErrors.ErrForbidden):
		return http.StatusForbidden, err
	}
	return http.StatusInternalServerError, err
}

func requestLock(r *http.Request) string {
	matches := requestLockRegex.FindStringSubmatch(r.Header.Get(net.HeaderIf))
	if len(matches) < 2 {
		return ""
	}

	return matches[1] // the first match is the whole string, the second is the token
}
