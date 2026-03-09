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
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/datagateway"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
)

const (
	// PerfMarkerResponseTime corresponds to the interval at which a performance marker is sent back to the TPC client
	PerfMarkerResponseTime float64 = 5
)

// PerfResponse provides a single chunk of permormance marker response
type PerfResponse struct {
	Timestamp time.Time
	Bytes     uint64
	Index     int
	Count     int
}

func (p *PerfResponse) getPerfResponseString() string {
	var sb strings.Builder
	sb.WriteString("Perf Marker\n")
	sb.WriteString("Timestamp: " + strconv.FormatInt(p.Timestamp.Unix(), 10) + "\n")
	sb.WriteString("Stripe Bytes Transferred: " + strconv.FormatUint(p.Bytes, 10) + "\n")
	sb.WriteString("Strip Index: " + strconv.Itoa(p.Index) + "\n")
	sb.WriteString("Total Stripe Count: " + strconv.Itoa(p.Count) + "\n")
	sb.WriteString("End\n")
	return sb.String()
}

// WriteCounter counts the number of bytes transferred and reports
// back to the TPC client about the progress of the transfer
// through the performance marker response stream.
type WriteCounter struct {
	Total    uint64
	PrevTime time.Time
	w        http.ResponseWriter
}

// SendPerfMarker flushes a single chunk (performance marker) as
// part of the chunked transfer encoding scheme.
func (wc *WriteCounter) SendPerfMarker(size uint64) {
	flusher, ok := wc.w.(http.Flusher)
	if !ok {
		panic("expected http.ResponseWriter to be an http.Flusher")
	}
	perfResp := PerfResponse{time.Now(), size, 0, 1}
	pString := perfResp.getPerfResponseString()
	fmt.Fprintln(wc.w, pString)
	flusher.Flush()
}

func (wc *WriteCounter) Write(p []byte) (int, error) {

	n := len(p)
	wc.Total += uint64(n)
	NowTime := time.Now()

	diff := NowTime.Sub(wc.PrevTime).Seconds()
	if diff >= PerfMarkerResponseTime {
		wc.SendPerfMarker(wc.Total)
		wc.PrevTime = NowTime
	}
	return n, nil
}

//
// An example of an HTTP TPC Pull
//
// +-----------------+        GET          +----------------+
// |  Src server     |  <----------------  |  Dest server   |
// |  (Remote)       |  ---------------->  |  (Reva)        |
// +-----------------+       Data          +----------------+
// 												^
// 												|
// 												| COPY
// 												|
// 										   +----------+
// 										   |  Client  |
// 										   +----------+

// handleTPCPull performs a GET request on the remote site and upload it
// the requested reva endpoint.
func (s *svc) handleTPCPull(ctx context.Context, w http.ResponseWriter, r *http.Request, ns string) {
	src := r.Header.Get("Source")
	dst := path.Join(ns, r.URL.Path)
	sublog := appctx.GetLogger(ctx).With().Str("src", src).Str("dst", dst).Logger()

	oh := r.Header.Get(net.HeaderOverwrite)
	overwrite, err := net.ParseOverwrite(oh)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		m := fmt.Sprintf("Overwrite header is set to incorrect value %v", overwrite)
		sublog.Warn().Msgf("HTTP TPC Pull: %s", m)
		b, err := errors.Marshal(http.StatusBadRequest, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}
	sublog.Debug().Bool("overwrite", overwrite).Msg("TPC Pull")

	client, err := s.gatewaySelector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// check if destination exists
	ref := &provider.Reference{Path: dst}
	dstStatReq := &provider.StatRequest{Ref: ref}
	dstStatRes, err := client.Stat(ctx, dstStatReq)
	if err != nil {
		sublog.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if dstStatRes.Status.Code != rpc.Code_CODE_OK && dstStatRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
		errors.HandleErrorStatus(&sublog, w, dstStatRes.Status)
		return
	}
	if dstStatRes.Status.Code == rpc.Code_CODE_OK && oh == "F" {
		sublog.Warn().Bool("overwrite", overwrite).Msg("Destination already exists")
		w.WriteHeader(http.StatusPreconditionFailed) // 412, see https://tools.ietf.org/html/rfc4918#section-9.8.5
		return
	}

	err = s.performHTTPPull(ctx, s.gatewaySelector, r, w, ns)
	if err != nil {
		sublog.Error().Err(err).Msg("error performing TPC Pull")
		return
	}
	fmt.Fprintf(w, "success: Created")
}

func (s *svc) performHTTPPull(ctx context.Context, selector pool.Selectable[gateway.GatewayAPIClient], r *http.Request, w http.ResponseWriter, ns string) error {
	src := r.Header.Get("Source")
	dst := path.Join(ns, r.URL.Path)
	sublog := appctx.GetLogger(ctx)
	sublog.Debug().Str("src", src).Str("dst", dst).Msg("Performing HTTP Pull")

	// get http client for remote
	httpClient := &http.Client{}

	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	// add authentication header
	bearerHeader := r.Header.Get(net.HeaderTransferAuth)
	req.Header.Add("Authorization", bearerHeader)

	// do download
	httpDownloadRes, err := httpClient.Do(req) // lgtm[go/request-forgery]
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer httpDownloadRes.Body.Close()

	if httpDownloadRes.StatusCode == http.StatusNotImplemented {
		w.WriteHeader(http.StatusBadRequest)
		return errtypes.NotSupported("Third-Party copy not supported, source might be a folder")
	}
	if httpDownloadRes.StatusCode != http.StatusOK {
		w.WriteHeader(httpDownloadRes.StatusCode)
		return errtypes.InternalError(fmt.Sprintf("Remote GET returned status code %d", httpDownloadRes.StatusCode))
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return errtypes.InternalError(err.Error())
	}
	// get upload url
	uReq := &provider.InitiateFileUploadRequest{
		Ref: &provider.Reference{Path: dst},
		Opaque: &typespb.Opaque{
			Map: map[string]*typespb.OpaqueEntry{
				"sizedeferred": {
					Value: []byte("true"),
				},
			},
		},
	}
	uRes, err := client.InitiateFileUpload(ctx, uReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	if uRes.Status.Code != rpc.Code_CODE_OK {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("status code %d", uRes.Status.Code)
	}

	var uploadEP, uploadToken string
	for _, p := range uRes.Protocols {
		if p.Protocol == "simple" {
			uploadEP, uploadToken = p.UploadEndpoint, p.Token
		}
	}

	// send performance markers periodically every PerfMarkerResponseTime (5 seconds unless configured)
	w.WriteHeader(http.StatusAccepted)
	wc := WriteCounter{0, time.Now(), w}
	tempReader := io.TeeReader(httpDownloadRes.Body, &wc)

	// do Upload
	httpUploadReq, err := rhttp.NewRequest(ctx, "PUT", uploadEP, tempReader)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	httpUploadReq.Header.Set(datagateway.TokenTransportHeader, uploadToken)
	httpUploadRes, err := s.client.Do(httpUploadReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	defer httpUploadRes.Body.Close()
	if httpUploadRes.StatusCode != http.StatusOK {
		w.WriteHeader(httpUploadRes.StatusCode)
		return err
	}
	return nil
}

//
// An example of an HTTP TPC Push
//
// +-----------------+        PUT          +----------------+
// |  Dest server    |  <----------------  |  Src server    |
// |  (Remote)       |  ---------------->  |  (Reva)        |
// +-----------------+       Done          +----------------+
// 												^
// 												|
// 												| COPY
// 												|
// 										   +----------+
// 										   |  Client  |
// 										   +----------+

// handleTPCPush performs a PUT request on the remote site and while downloading
// data from the requested reva endpoint.
func (s *svc) handleTPCPush(ctx context.Context, w http.ResponseWriter, r *http.Request, ns string) {
	src := path.Join(ns, r.URL.Path)
	dst := r.Header.Get("Destination")
	sublog := appctx.GetLogger(ctx).With().Str("src", src).Str("dst", dst).Logger()

	oh := r.Header.Get(net.HeaderOverwrite)
	overwrite, err := net.ParseOverwrite(oh)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		m := fmt.Sprintf("Overwrite header is set to incorrect value %v", overwrite)
		sublog.Warn().Msgf("HTTP TPC Push: %s", m)
		b, err := errors.Marshal(http.StatusBadRequest, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}

	sublog.Debug().Bool("overwrite", overwrite).Msg("TPC Push")

	client, err := s.gatewaySelector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ref := &provider.Reference{Path: src}
	srcStatReq := &provider.StatRequest{Ref: ref}
	srcStatRes, err := client.Stat(ctx, srcStatReq)
	if err != nil {
		sublog.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if srcStatRes.Status.Code != rpc.Code_CODE_OK && srcStatRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
		errors.HandleErrorStatus(&sublog, w, srcStatRes.Status)
		return
	}
	if srcStatRes.Info.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		sublog.Error().Msg("Third-Party copy of a folder is not supported")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.performHTTPPush(ctx, r, w, srcStatRes.Info, ns)
	if err != nil {
		sublog.Error().Err(err).Msg("error performing TPC Push")
		return
	}
	fmt.Fprintf(w, "success: Created")
}

func (s *svc) performHTTPPush(ctx context.Context, r *http.Request, w http.ResponseWriter, srcInfo *provider.ResourceInfo, ns string) error {
	src := path.Join(ns, r.URL.Path)
	dst := r.Header.Get("Destination")

	sublog := appctx.GetLogger(ctx)
	sublog.Debug().Str("src", src).Str("dst", dst).Msg("Performing HTTP Push")

	// get download url
	dReq := &provider.InitiateFileDownloadRequest{
		Ref: &provider.Reference{Path: src},
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	dRes, err := client.InitiateFileDownload(ctx, dReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	if dRes.Status.Code != rpc.Code_CODE_OK {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("status code %d", dRes.Status.Code)
	}

	var downloadEP, downloadToken string
	for _, p := range dRes.Protocols {
		if p.Protocol == "simple" {
			downloadEP, downloadToken = p.DownloadEndpoint, p.Token
		}
	}

	// do download
	httpDownloadReq, err := rhttp.NewRequest(ctx, "GET", downloadEP, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	httpDownloadReq.Header.Set(datagateway.TokenTransportHeader, downloadToken)

	httpDownloadRes, err := s.client.Do(httpDownloadReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer httpDownloadRes.Body.Close()
	if httpDownloadRes.StatusCode != http.StatusOK {
		w.WriteHeader(httpDownloadRes.StatusCode)
		return fmt.Errorf("remote PUT returned status code %d", httpDownloadRes.StatusCode)
	}

	// send performance markers periodically every PerfMarkerResponseTime (5 seconds unless configured)
	w.WriteHeader(http.StatusAccepted)
	wc := WriteCounter{0, time.Now(), w}
	tempReader := io.TeeReader(httpDownloadRes.Body, &wc)

	// get http client for a remote call
	httpClient := &http.Client{}
	req, err := http.NewRequest("PUT", dst, tempReader)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	// add authentication header and content length
	bearerHeader := r.Header.Get(net.HeaderTransferAuth)
	req.Header.Add("Authorization", bearerHeader)
	req.ContentLength = int64(srcInfo.GetSize())

	// do Upload
	httpUploadRes, err := httpClient.Do(req) // lgtm[go/request-forgery]
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer httpUploadRes.Body.Close()

	if httpUploadRes.StatusCode != http.StatusOK {
		w.WriteHeader(httpUploadRes.StatusCode)
		return err
	}

	return nil
}
