package scanners

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/mime"

	ic "github.com/opencloud-eu/icap-client"
)

// Scanner is the interface that wraps the basic Do method
type Scanner interface {
	Do(req ic.Request) (ic.Response, error)
}

// NewICAP returns a Scanner talking to an ICAP server
func NewICAP(icapURL string, icapService string, timeout time.Duration) (ICAP, error) {
	endpoint, err := url.Parse(icapURL)
	if err != nil {
		return ICAP{}, err
	}

	endpoint.Scheme = "icap"
	endpoint.Path = icapService

	client, err := ic.NewClient(
		ic.WithICAPConnectionTimeout(timeout),
	)
	if err != nil {
		return ICAP{}, err
	}

	return ICAP{Client: &client, URL: endpoint.String()}, nil
}

// ICAP is responsible for scanning files using an ICAP server
type ICAP struct {
	Client Scanner
	URL    string
}

// Scan scans a file using the ICAP server
func (s ICAP) Scan(in Input) (Result, error) {
	ctx := context.TODO()
	result := Result{}

	optReq, err := ic.NewRequest(ctx, ic.MethodOPTIONS, s.URL, nil, nil)
	if err != nil {
		return result, err
	}

	optRes, err := s.Client.Do(optReq)
	if err != nil {
		return result, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, in.Url, in.Body)
	if err != nil {
		return result, err
	}

	httpReq.ContentLength = in.Size
	if mt := mime.Detect(path.Ext(in.Name) == "", in.Name); mt != "" {
		httpReq.Header.Set("Content-Type", mt)
	}

	req, err := ic.NewRequest(ctx, ic.MethodREQMOD, s.URL, httpReq, nil)
	if err != nil {
		return result, err
	}

	if optRes.PreviewBytes > 0 {
		err = req.SetPreview(optRes.PreviewBytes)
		if err != nil {
			return result, err
		}
	}

	res, err := s.Client.Do(req)
	if err != nil {
		return result, err
	}
	result.ScanTime = time.Now()

	// TODO: make header configurable
	if data, infected := res.Header["X-Infection-Found"]; infected {
		result.Infected = infected

		match := regexp.MustCompile(`Threat=(.*);`).FindStringSubmatch(fmt.Sprint(data))

		if len(match) > 1 {
			result.Description = match[1]
		}
	}

	if result.Infected || res.ContentResponse == nil {
		return result, nil
	}

	// mcafee forwards the scan result as HTML in the content response;
	// status 403 indicates that the file is infected
	result.Infected = res.ContentResponse.StatusCode == http.StatusForbidden
	result.Description = res.ContentResponse.Status

	return result, nil
}
