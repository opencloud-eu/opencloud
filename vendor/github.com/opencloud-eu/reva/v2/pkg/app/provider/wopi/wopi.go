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

package wopi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
	appprovider "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	appregistry "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/app"
	"github.com/opencloud-eu/reva/v2/pkg/app/provider/registry"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/mime"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/pkg/errors"
)

func init() {
	registry.Register("wopi", New)
}

type config struct {
	IOPSecret                 string `mapstructure:"iop_secret" docs:";The IOP secret used to connect to the wopiserver."`
	WopiURL                   string `mapstructure:"wopi_url" docs:";The wopiserver's URL."`
	WopiFolderURLBaseURL      string `mapstructure:"wopi_folder_url_base_url" docs:";The base URL to generate links to navigate back to the containing folder."`
	WopiFolderURLPathTemplate string `mapstructure:"wopi_folder_url_path_template" docs:";The template to generate the folderurl path segments."`
	AppName                   string `mapstructure:"app_name" docs:";The App user-friendly name."`
	AppIconURI                string `mapstructure:"app_icon_uri" docs:";A URI to a static asset which represents the app icon."`
	AppURL                    string `mapstructure:"app_url" docs:";The App URL."`
	AppIntURL                 string `mapstructure:"app_int_url" docs:";The internal app URL in case of dockerized deployments. Defaults to AppURL"`
	AppAPIKey                 string `mapstructure:"app_api_key" docs:";The API key used by the app, if applicable."`
	JWTSecret                 string `mapstructure:"jwt_secret" docs:";The JWT secret to be used to retrieve the token TTL."`
	AppDesktopOnly            bool   `mapstructure:"app_desktop_only" docs:"false;Specifies if the app can be opened only on desktop."`
	InsecureConnections       bool   `mapstructure:"insecure_connections"`
	AppDisableChat            bool   `mapstructure:"app_disable_chat"`
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		return nil, err
	}
	return c, nil
}

type wopiProvider struct {
	conf       *config
	wopiClient *http.Client
	appURLs    map[string]map[string]string // map[viewMode]map[extension]appURL
}

// New returns an implementation of the app.Provider interface that
// connects to an application in the backend.
func New(m map[string]interface{}) (app.Provider, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	if c.AppIntURL == "" {
		c.AppIntURL = c.AppURL
	}
	if c.IOPSecret == "" {
		c.IOPSecret = os.Getenv("REVA_APPPROVIDER_IOPSECRET")
	}
	c.JWTSecret = sharedconf.GetJWTSecret(c.JWTSecret)

	appURLs, err := getAppURLs(c)
	if err != nil {
		return nil, err
	}

	wopiClient := rhttp.GetHTTPClient(
		rhttp.Timeout(time.Duration(5*int64(time.Second))),
		rhttp.Insecure(c.InsecureConnections),
	)
	wopiClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &wopiProvider{
		conf:       c,
		wopiClient: wopiClient,
		appURLs:    appURLs,
	}, nil
}

func (p *wopiProvider) GetAppURL(ctx context.Context, resource *provider.ResourceInfo, viewMode appprovider.ViewMode, token, language string) (*appprovider.OpenInAppURL, error) {
	log := appctx.GetLogger(ctx)

	ext := path.Ext(resource.Path)
	wopiurl, err := url.Parse(p.conf.WopiURL)
	if err != nil {
		return nil, err
	}
	wopiurl.Path = path.Join(wopiurl.Path, "/wopi/iop/openinapp")

	httpReq, err := rhttp.NewRequest(ctx, "GET", wopiurl.String(), nil)
	if err != nil {
		return nil, err
	}

	q := httpReq.URL.Query()

	q.Add("endpoint", storagespace.FormatStorageID(resource.GetId().GetStorageId(), resource.GetId().GetSpaceId()))
	q.Add("fileid", resource.GetId().OpaqueId)
	q.Add("viewmode", viewMode.String())

	folderURLPath := templates.WithResourceInfo(resource, p.conf.WopiFolderURLPathTemplate)
	folderURLBaseURL, err := url.Parse(p.conf.WopiFolderURLBaseURL)
	if err != nil {
		return nil, err
	}
	if folderURLPath != "" {
		folderURLBaseURL.Path = path.Join(folderURLBaseURL.Path, folderURLPath)
		q.Add("folderurl", folderURLBaseURL.String())
	}

	u, ok := ctxpkg.ContextGetUser(ctx)
	if ok { // else defaults to "Guest xyz"
		if u.Id.Type == userpb.UserType_USER_TYPE_LIGHTWEIGHT || u.Id.Type == userpb.UserType_USER_TYPE_FEDERATED {
			q.Add("userid", resource.Owner.OpaqueId+"@"+resource.Owner.Idp)
		} else {
			q.Add("userid", u.Id.OpaqueId+"@"+u.Id.Idp)
		}
		var isPublicShare bool
		if u.Opaque != nil {
			if _, ok := u.Opaque.Map["public-share-role"]; ok {
				isPublicShare = true
			}
		}

		if !isPublicShare {
			q.Add("username", u.DisplayName)
		}
	}

	q.Add("appname", p.conf.AppName)

	var viewAppURL string
	if viewAppURLs, ok := p.appURLs["view"]; ok {
		if viewAppURL, ok = viewAppURLs[ext]; ok {
			q.Add("appviewurl", viewAppURL)
		}
	}
	access := "edit"
	if resource.GetSize() == 0 {
		if _, ok := p.appURLs["editnew"]; ok {
			access = "editnew"
		}
	}
	if editAppURLs, ok := p.appURLs[access]; ok {
		if editAppURL, ok := editAppURLs[ext]; ok {
			q.Add("appurl", editAppURL)
		}
	}
	if q.Get("appurl") == "" {
		// assuming that a view action is always available in the /hosting/discovery manifest
		// eg. Collabora does support viewing jpgs but no editing
		// eg. OnlyOffice does support viewing pdfs but no editing
		// there is no known case of supporting edit only without view
		q.Add("appurl", viewAppURL)
	}
	if q.Get("appurl") == "" && q.Get("appviewurl") == "" {
		return nil, errors.New("wopi: neither edit nor view app url found")
	}

	if p.conf.AppIntURL != "" {
		q.Add("appinturl", p.conf.AppIntURL)
	}

	httpReq.URL.RawQuery = q.Encode()

	if p.conf.AppAPIKey != "" {
		httpReq.Header.Set("ApiKey", p.conf.AppAPIKey)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.conf.IOPSecret)
	httpReq.Header.Set("TokenHeader", token)

	// Call the WOPI server and parse the response (body will always contain a payload)
	openRes, err := p.wopiClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "wopi: error performing open request to WOPI server")
	}
	defer openRes.Body.Close()

	body, err := io.ReadAll(openRes.Body)
	if err != nil {
		return nil, err
	}
	if openRes.StatusCode != http.StatusOK {
		// WOPI returned failure: body contains a user-friendly error message (yet perform a sanity check)
		sbody := ""
		if body != nil {
			sbody = string(body)
		}
		log.Warn().Msg(fmt.Sprintf("wopi: WOPI server returned HTTP %s to request %s, error was: %s", openRes.Status, httpReq.URL.String(), sbody))
		return nil, errors.New(sbody)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	tokenTTL, err := p.getAccessTokenTTL(ctx)
	if err != nil {
		return nil, err
	}

	url, err := url.Parse(result["app-url"].(string))
	if err != nil {
		return nil, err
	}

	urlQuery := url.Query()
	if language != "" {
		urlQuery.Set("ui", language)                  // OnlyOffice
		urlQuery.Set("lang", covertLangTag(language)) // Collabora, Impact on the default document language of OnlyOffice
		urlQuery.Set("UI_LLCC", language)             // Office365
	}
	if p.conf.AppDisableChat {
		urlQuery.Set("dchat", "1") // OnlyOffice disable chat
	}

	url.RawQuery = urlQuery.Encode()
	appFullURL := url.String()

	// Depending on whether wopi server returned any form parameters or not,
	// we decide whether the request method is POST or GET
	var formParams map[string]string
	method := "GET"
	if form, ok := result["form-parameters"].(map[string]interface{}); ok {
		if tkn, ok := form["access_token"].(string); ok {
			formParams = map[string]string{
				"access_token":     tkn,
				"access_token_ttl": tokenTTL,
			}
			method = "POST"
		}
	}

	log.Info().Msg(fmt.Sprintf("wopi: returning app URL %s", appFullURL))
	return &appprovider.OpenInAppURL{
		AppUrl:         appFullURL,
		Method:         method,
		FormParameters: formParams,
	}, nil
}

func (p *wopiProvider) GetAppProviderInfo(ctx context.Context) (*appregistry.ProviderInfo, error) {
	// Initially we store the mime types in a map to avoid duplicates
	mimeTypesMap := make(map[string]bool)
	for _, extensions := range p.appURLs {
		for ext := range extensions {
			m := mime.Detect(false, ext)
			mimeTypesMap[m] = true
		}
	}

	mimeTypes := make([]string, 0, len(mimeTypesMap))
	for m := range mimeTypesMap {
		mimeTypes = append(mimeTypes, m)
	}

	return &appregistry.ProviderInfo{
		Name:        p.conf.AppName,
		Icon:        p.conf.AppIconURI,
		DesktopOnly: p.conf.AppDesktopOnly,
		MimeTypes:   mimeTypes,
	}, nil
}

func getAppURLs(c *config) (map[string]map[string]string, error) {
	// Initialize WOPI URLs by discovery
	httpcl := rhttp.GetHTTPClient(
		rhttp.Timeout(time.Duration(5*int64(time.Second))),
		rhttp.Insecure(c.InsecureConnections),
	)

	appurl, err := url.Parse(c.AppIntURL)
	if err != nil {
		return nil, err
	}
	appurl.Path = path.Join(appurl.Path, "/hosting/discovery")

	discReq, err := http.NewRequest("GET", appurl.String(), nil)
	if err != nil {
		return nil, err
	}
	discRes, err := httpcl.Do(discReq)
	if err != nil {
		return nil, err
	}
	defer discRes.Body.Close()

	var appURLs map[string]map[string]string

	switch discRes.StatusCode {
	case http.StatusOK:
		appURLs, err = parseWopiDiscovery(discRes.Body)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing wopi discovery response")
		}
	case http.StatusNotFound:
		// this may be a bridge-supported app
		discReq, err = http.NewRequest("GET", c.AppIntURL, nil)
		if err != nil {
			return nil, err
		}
		discRes, err = httpcl.Do(discReq)
		if err != nil {
			return nil, err
		}
		defer discRes.Body.Close()

		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(discRes.Body)
		if err != nil {
			return nil, err
		}

		// scrape app's home page to find the appname
		if !strings.Contains(buf.String(), c.AppName) {
			return nil, errors.New("Application server at " + c.AppURL + " does not match this AppProvider for " + c.AppName)
		}

		// register the supported mimetypes in the AppRegistry: this is hardcoded for the time being
		// TODO(lopresti) move to config
		switch c.AppName {
		case "CodiMD":
			appURLs = getCodimdExtensions(c.AppURL)
		case "Etherpad":
			appURLs = getEtherpadExtensions(c.AppURL)
		default:
			return nil, errors.New("Application server " + c.AppName + " running at " + c.AppURL + " is unsupported")
		}
	}
	return appURLs, nil
}

func (p *wopiProvider) getAccessTokenTTL(ctx context.Context) (string, error) {
	tkn := ctxpkg.ContextMustGetToken(ctx)
	token, err := jwt.ParseWithClaims(tkn, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(p.conf.JWTSecret), nil
	})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		// milliseconds since Jan 1, 1970 UTC as required in https://wopi.readthedocs.io/projects/wopirest/en/latest/concepts.html?highlight=access_token_ttl#term-access-token-ttl
		return strconv.FormatInt(claims.ExpiresAt.Unix()*1000, 10), nil
	}

	return "", errtypes.InvalidCredentials("wopi: invalid token present in ctx")
}

func parseWopiDiscovery(body io.Reader) (map[string]map[string]string, error) {
	appURLs := make(map[string]map[string]string)

	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(body); err != nil {
		return nil, err
	}
	root := doc.SelectElement("wopi-discovery")
	if root == nil {
		return nil, errors.New("wopi-discovery response malformed")
	}

	for _, netzone := range root.SelectElements("net-zone") {

		if strings.Contains(netzone.SelectAttrValue("name", ""), "external") {
			for _, app := range netzone.SelectElements("app") {
				for _, action := range app.SelectElements("action") {
					access := action.SelectAttrValue("name", "")
					if access == "view" || access == "edit" || access == "editnew" {
						ext := action.SelectAttrValue("ext", "")
						urlString := action.SelectAttrValue("urlsrc", "")

						if ext == "" || urlString == "" {
							continue
						}

						u, err := url.Parse(urlString)
						if err != nil {
							// it sucks we cannot log here because this function is run
							// on init without any context.
							// TODO(labkode): add logging when we'll have static logging in boot phase.
							continue
						}

						// remove any malformed query parameter from discovery urls
						q := u.Query()
						for k := range q {
							if strings.Contains(k, "<") || strings.Contains(k, ">") {
								q.Del(k)
							}
						}

						u.RawQuery = q.Encode()

						if _, ok := appURLs[access]; !ok {
							appURLs[access] = make(map[string]string)
						}
						appURLs[access]["."+ext] = u.String()
					}
				}
			}
		}
	}
	return appURLs, nil
}

func getCodimdExtensions(appURL string) map[string]map[string]string {
	// Register custom mime types
	mime.RegisterMime(".zmd", "application/compressed-markdown")

	appURLs := make(map[string]map[string]string)
	appURLs["edit"] = map[string]string{
		".txt": appURL,
		".md":  appURL,
		".zmd": appURL,
	}
	return appURLs
}

func getEtherpadExtensions(appURL string) map[string]map[string]string {
	appURLs := make(map[string]map[string]string)
	appURLs["edit"] = map[string]string{
		".epd": appURL,
	}
	return appURLs
}

// TODO Find better solution
// This conversion was made because no other way to set the default document language to OnlyOffice was found.
func covertLangTag(lang string) string {
	switch lang {
	case "cs":
		return "cs-CZ"
	case "de":
		return "de-DE"
	case "es":
		return "es-ES"
	case "fr":
		return "fr-FR"
	case "it":
		return "it-IT"
	default:
		return "en"
	}
}
