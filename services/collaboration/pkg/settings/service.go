// Package settings stores the office settings that Collabora manages with its
// Options dialog: per-user settings and settings shared by all users, such as
// dictionaries (wordbook), autotext and preference files (xcu).
//
// Collabora talks to these endpoints server side. The editor page gets the base
// URL as the wopi_setting_base_url form parameter, and CheckFileInfo returns
// UserSettings and SharedSettings so the settings apply to opened documents.
package settings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"

	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/collaboration"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/connector/fileinfo"
)

const (
	// TypeUser holds the settings of a single user.
	TypeUser = "userconfig"
	// TypeShared holds the settings shared by all users.
	TypeShared = "systemconfig"

	maxUploadSize = 16 << 20
)

var (
	categoryPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	// Collabora identifies a settings file by "/settings/<type>/<category>/<name>".
	fileIDPattern = regexp.MustCompile(`^/settings/(userconfig|systemconfig)/([a-zA-Z0-9_-]+)/([^/\\]+)$`)

	errForbidden = errors.New("forbidden")
)

// BaseURL returns the settings URL handed to Collabora, below the WOPI source.
func BaseURL(cfg *config.Config) string {
	return strings.TrimRight(cfg.Wopi.WopiSrc, "/") + "/wopi/settings"
}

// Service serves the settings endpoints. All handlers expect the WOPI context
// middleware to have authenticated the request.
type Service struct {
	fs              afero.Fs
	baseURL         string
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

// NewService creates a settings service storing its files in fSys.
func NewService(fSys afero.Fs, baseURL string, gws pool.Selectable[gateway.GatewayAPIClient]) *Service {
	return &Service{fs: fSys, baseURL: baseURL, gatewaySelector: gws}
}

type fileEntry struct {
	URI   string `json:"uri"`
	Stamp string `json:"stamp"`
}

// GetConfig lists the settings files of the requested type, grouped by category.
func (s *Service) GetConfig(w http.ResponseWriter, r *http.Request) {
	logger := zerolog.Ctx(r.Context())
	settingsType := r.URL.Query().Get("type")

	dir, err := s.typeDir(r, settingsType)
	if err != nil {
		writeError(w, err)
		return
	}

	kind := "shared"
	if settingsType == TypeUser {
		kind = "user"
	}
	body := map[string]any{"kind": kind}

	categories, err := afero.ReadDir(s.fs, dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		logger.Error().Err(err).Msg("settings: failed to list categories")
		writeError(w, err)
		return
	}
	for _, category := range categories {
		if !category.IsDir() || !categoryPattern.MatchString(category.Name()) {
			continue
		}
		files, err := afero.ReadDir(s.fs, path.Join(dir, category.Name()))
		if err != nil {
			logger.Error().Err(err).Msg("settings: failed to list files")
			writeError(w, err)
			return
		}
		entries := []fileEntry{}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			entries = append(entries, fileEntry{
				URI:   s.fileURI(settingsType, category.Name(), f.Name(), accessToken(r)),
				Stamp: stamp(f),
			})
		}
		body[category.Name()] = entries
	}

	writeJSON(w, http.StatusOK, body)
}

// GetFile returns the content of a single settings file.
func (s *Service) GetFile(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	category, name := q.Get("category"), q.Get("name")
	if !categoryPattern.MatchString(category) || !validName(name) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	dir, err := s.typeDir(r, q.Get("type"))
	if err != nil {
		writeError(w, err)
		return
	}

	content, err := afero.ReadFile(s.fs, path.Join(dir, category, name))
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(content)
}

// Upload stores a settings file, the request body is the file content.
func (s *Service) Upload(w http.ResponseWriter, r *http.Request) {
	settingsType, category, name, err := parseFileID(r.URL.Query().Get("fileId"))
	if err != nil {
		writeError(w, err)
		return
	}

	dir, err := s.writableTypeDir(r, settingsType)
	if err != nil {
		writeError(w, err)
		return
	}

	content, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxUploadSize))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	categoryDir := path.Join(dir, category)
	if err := s.fs.MkdirAll(categoryDir, 0o750); err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Msg("settings: failed to create directory")
		writeError(w, err)
		return
	}
	filePath := path.Join(categoryDir, name)
	if err := afero.WriteFile(s.fs, filePath, content, 0o640); err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Msg("settings: failed to write file")
		writeError(w, err)
		return
	}

	info, err := s.fs.Stat(filePath)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "success",
		"filename": name,
		"details": fileEntry{
			URI:   s.fileURI(settingsType, category, name, accessToken(r)),
			Stamp: stamp(info),
		},
	})
}

// Delete removes a settings file.
func (s *Service) Delete(w http.ResponseWriter, r *http.Request) {
	settingsType, category, name, err := parseFileID(r.URL.Query().Get("fileId"))
	if err != nil {
		writeError(w, err)
		return
	}

	dir, err := s.writableTypeDir(r, settingsType)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := s.fs.Remove(path.Join(dir, category, name)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "success"})
}

// SettingsRefs returns the references for CheckFileInfo. The user reference is
// nil for anonymous and public link users, who have no settings of their own.
func (s *Service) SettingsRefs(r *http.Request) (userRef, sharedRef *fileinfo.SettingsRef) {
	token := accessToken(r)
	if dir, err := s.typeDir(r, TypeUser); err == nil {
		userRef = &fileinfo.SettingsRef{URI: s.configURI(TypeUser, token), Stamp: s.dirStamp(dir)}
	}
	if dir, err := s.typeDir(r, TypeShared); err == nil {
		sharedRef = &fileinfo.SettingsRef{URI: s.configURI(TypeShared, token), Stamp: s.dirStamp(dir)}
	}
	return userRef, sharedRef
}

// typeDir returns the directory of a settings type for the requesting user.
func (s *Service) typeDir(r *http.Request, settingsType string) (string, error) {
	switch settingsType {
	case TypeShared:
		return "/" + TypeShared, nil
	case TypeUser:
		user, ok := revactx.ContextGetUser(r.Context())
		if !ok || isAnonymous(user) {
			return "", errForbidden
		}
		// same encoding as the WOPI UserId, safe as a single path element
		userKey := hex.EncodeToString([]byte(user.GetId().GetOpaqueId() + "@" + user.GetId().GetIdp()))
		return path.Join("/", TypeUser, userKey), nil
	default:
		return "", fs.ErrInvalid
	}
}

// writableTypeDir is typeDir, additionally requiring the WebOffice.Manage
// permission to change the shared settings.
func (s *Service) writableTypeDir(r *http.Request, settingsType string) (string, error) {
	dir, err := s.typeDir(r, settingsType)
	if err != nil || settingsType != TypeShared {
		return dir, err
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return "", err
	}
	_, canManage, err := collaboration.CheckPermissions(gatewayClient, r.Context(), collaboration.PermissionWebOfficeManage)
	if err != nil {
		return "", err
	}
	if !canManage {
		return "", errForbidden
	}
	return dir, nil
}

// dirStamp changes whenever a file below dir is added, removed or modified.
func (s *Service) dirStamp(dir string) string {
	var lines []string
	_ = afero.Walk(s.fs, dir, func(p string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			lines = append(lines, p+"\x00"+stamp(info))
		}
		return nil
	})
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:8])
}

func (s *Service) configURI(settingsType, token string) string {
	q := url.Values{"type": {settingsType}, "access_token": {token}}
	return s.baseURL + "?" + q.Encode()
}

func (s *Service) fileURI(settingsType, category, name, token string) string {
	q := url.Values{"type": {settingsType}, "category": {category}, "name": {name}, "access_token": {token}}
	return s.baseURL + "/file?" + q.Encode()
}

func parseFileID(fileID string) (settingsType, category, name string, err error) {
	m := fileIDPattern.FindStringSubmatch(fileID)
	if m == nil || !validName(m[3]) {
		return "", "", "", fs.ErrInvalid
	}
	return m[1], m[2], m[3], nil
}

func validName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00")
}

func isAnonymous(user *userpb.User) bool {
	return user.GetId().GetOpaqueId() == "" || utils.ExistsInOpaque(user.GetOpaque(), "public-share-role")
}

func accessToken(r *http.Request) string {
	return r.URL.Query().Get("access_token")
}

func stamp(info fs.FileInfo) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", info.Name(), info.Size(), info.ModTime().UnixNano())))
	return hex.EncodeToString(sum[:8])
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errForbidden):
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	case errors.Is(err, fs.ErrInvalid):
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	case errors.Is(err, fs.ErrNotExist):
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	default:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
