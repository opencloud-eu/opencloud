package settings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/spf13/afero"
)

func request(method, target, body string, user *userpb.User) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	if user != nil {
		r = r.WithContext(revactx.ContextSetUser(r.Context(), user))
	}
	return r
}

func alice() *userpb.User {
	return &userpb.User{Id: &userpb.UserId{OpaqueId: "alice", Idp: "idp"}}
}

func TestUserSettingsRoundTrip(t *testing.T) {
	s := NewService(afero.NewMemMapFs(), "https://wopi.test/wopi/settings", nil)
	fileID := url.QueryEscape("/settings/userconfig/wordbook/standard.dic")

	w := httptest.NewRecorder()
	s.Upload(w, request(http.MethodPost, "/wopi/settings/upload?fileId="+fileID+"&access_token=tok", "words", alice()))
	if w.Code != http.StatusOK {
		t.Fatalf("upload: got %d", w.Code)
	}

	w = httptest.NewRecorder()
	s.GetConfig(w, request(http.MethodGet, "/wopi/settings?type=userconfig&access_token=tok", "", alice()))
	var cfg map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &cfg); err != nil || cfg["kind"] != "user" {
		t.Fatalf("config: %d %s", w.Code, w.Body.String())
	}
	entries := cfg["wordbook"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected one wordbook entry, got %v", cfg)
	}
	fileURI, _ := url.Parse(entries[0].(map[string]any)["uri"].(string))
	if fileURI.Query().Get("access_token") != "tok" || fileURI.Query().Get("name") != "standard.dic" {
		t.Fatalf("unexpected file uri %s", fileURI)
	}

	w = httptest.NewRecorder()
	s.GetFile(w, request(http.MethodGet, fileURI.RequestURI(), "", alice()))
	if w.Code != http.StatusOK || w.Body.String() != "words" {
		t.Fatalf("file: %d %q", w.Code, w.Body.String())
	}

	bob := &userpb.User{Id: &userpb.UserId{OpaqueId: "bob", Idp: "idp"}}
	w = httptest.NewRecorder()
	s.GetFile(w, request(http.MethodGet, fileURI.RequestURI(), "", bob))
	if w.Code != http.StatusNotFound {
		t.Fatalf("another user must not see alice's file, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	s.Delete(w, request(http.MethodDelete, "/wopi/settings?fileId="+fileID, "", alice()))
	if w.Code != http.StatusOK {
		t.Fatalf("delete: got %d", w.Code)
	}
}

func TestRejectsInvalidAndAnonymousRequests(t *testing.T) {
	s := NewService(afero.NewMemMapFs(), "https://wopi.test/wopi/settings", nil)

	for _, fileID := range []string{"/settings/userconfig/../x", "/settings/userconfig/wordbook/..", "/settings/other/wordbook/a", "/etc/passwd"} {
		w := httptest.NewRecorder()
		s.Upload(w, request(http.MethodPost, "/wopi/settings/upload?fileId="+url.QueryEscape(fileID), "x", alice()))
		if w.Code != http.StatusBadRequest {
			t.Errorf("fileId %q: got %d, want 400", fileID, w.Code)
		}
	}

	public := &userpb.User{
		Id: &userpb.UserId{OpaqueId: "link", Idp: "idp"},
		Opaque: &typesv1beta1.Opaque{Map: map[string]*typesv1beta1.OpaqueEntry{
			"public-share-role": {Decoder: "plain", Value: []byte("viewer")},
		}},
	}
	w := httptest.NewRecorder()
	s.GetConfig(w, request(http.MethodGet, "/wopi/settings?type=userconfig", "", public))
	if w.Code != http.StatusForbidden {
		t.Errorf("public link user: got %d, want 403", w.Code)
	}

	userRef, sharedRef := s.SettingsRefs(request(http.MethodGet, "/wopi/files/x?access_token=tok", "", public))
	if userRef != nil || sharedRef == nil {
		t.Errorf("public link user must only get shared settings: %v %v", userRef, sharedRef)
	}
}
