package helpers

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/beevik/etree"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/mime"
	"github.com/pkg/errors"
)

// AppURLs holds the app urls fetched from the WOPI app discovery endpoint
// It is a type safe wrapper around an atomic pointer to a map
type AppURLs struct {
	urls atomic.Pointer[map[string]map[string]string]
}

func NewAppURLs() *AppURLs {
	a := &AppURLs{}
	a.urls.Store(&map[string]map[string]string{})
	return a
}

func (a *AppURLs) Store(urls map[string]map[string]string) {
	a.urls.Store(&urls)
}

func (a *AppURLs) GetMimeTypes() []string {
	currentURLs := a.urls.Load()
	if currentURLs == nil {
		return []string{}
	}

	mimeTypesMap := make(map[string]bool)
	for _, extensions := range *currentURLs {
		for ext := range extensions {
			m := mime.Detect(false, ext)
			// skip the default
			if m == "application/octet-stream" {
				continue
			}
			mimeTypesMap[m] = true
		}
	}

	// Convert map to slice
	mimeTypes := make([]string, 0, len(mimeTypesMap))
	for mimeType := range mimeTypesMap {
		mimeTypes = append(mimeTypes, mimeType)
	}

	return mimeTypes
}

// GetAppURLFor gets the appURL from the list of appURLs based on the
// action and file extension provided. If there is no match, an empty
// string will be returned.
func (a *AppURLs) GetAppURLFor(action, fileExt string) string {
	currentURLs := a.urls.Load()
	if currentURLs == nil {
		return ""
	}

	if actionURL, ok := (*currentURLs)[action]; ok {
		if actionExtensionURL, ok := actionURL[fileExt]; ok {
			return actionExtensionURL
		}
	}
	return ""
}

// GetAppURLs gets the edit and view urls for different file types from the
// target WOPI app (onlyoffice, collabora, etc) via their "/hosting/discovery"
// endpoint.
func GetAppURLs(cfg *config.Config, logger log.Logger) (map[string]map[string]string, error) {
	wopiAppUrl := cfg.App.Addr + "/hosting/discovery"

	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: cfg.App.Insecure,
			},
		},
	}

	httpResp, err := httpClient.Get(wopiAppUrl)
	if err != nil {
		return nil, err
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		logger.Error().
			Str("WopiAppUrl", wopiAppUrl).
			Int("HttpCode", httpResp.StatusCode).
			Msg("WopiDiscovery: wopi app url failed with unexpected code")
		return nil, errors.New("status code was not 200")
	}

	var appURLs map[string]map[string]string

	appURLs, err = parseWopiDiscovery(httpResp.Body)
	if err != nil {
		logger.Error().
			Err(err).
			Str("WopiAppUrl", wopiAppUrl).
			Msg("WopiDiscovery: failed to parse wopi discovery response")
		return nil, errors.Wrap(err, "error parsing wopi discovery response")
	}

	// We won't log anything if successful
	return appURLs, nil
}

// parseWopiDiscovery parses the response of the "/hosting/discovery" endpoint
func parseWopiDiscovery(body io.Reader) (map[string]map[string]string, error) {
	appURLs := make(map[string]map[string]string)

	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(body); err != nil {
		return nil, err
	}
	root := doc.SelectElement("wopi-discovery")

	for _, netzone := range root.SelectElements("net-zone") {

		if strings.Contains(netzone.SelectAttrValue("name", ""), "external") {
			for _, app := range netzone.SelectElements("app") {
				for _, action := range app.SelectElements("action") {
					access := action.SelectAttrValue("name", "")
					if access == "view" || access == "edit" || access == "view_comment" {
						ext := action.SelectAttrValue("ext", "")
						urlString := action.SelectAttrValue("urlsrc", "")

						if ext == "" || urlString == "" {
							continue
						}

						u, err := url.Parse(urlString)
						if err != nil {
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
