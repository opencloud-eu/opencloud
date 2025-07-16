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
	"encoding/xml"
	"io"
	"net/http"

	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/propfind"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/permission"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

const (
	elementNameSearchFiles = "search-files"
	elementNameFilterFiles = "filter-files"
)

func (s *svc) handleReport(w http.ResponseWriter, r *http.Request, ns string) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)
	// fn := path.Join(ns, r.URL.Path)

	rep, status, err := readReport(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("error reading report")
		w.WriteHeader(status)
		return
	}
	if rep.SearchFiles != nil {
		s.doSearchFiles(w, r, rep.SearchFiles)
		return
	}

	if rep.FilterFiles != nil {
		s.doFilterFiles(w, r, rep.FilterFiles, ns)
		return
	}

	// TODO(jfd): implement report

	w.WriteHeader(http.StatusNotImplemented)
}

func (s *svc) doSearchFiles(w http.ResponseWriter, r *http.Request, sf *reportSearchFiles) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *svc) doFilterFiles(w http.ResponseWriter, r *http.Request, ff *reportFilterFiles, namespace string) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)

	if ff.Rules.Favorite {
		// List the users favorite resources.
		client, err := s.gatewaySelector.Next()
		if err != nil {
			log.Error().Err(err).Msg("error selecting next gateway client")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		currentUser := ctxpkg.ContextMustGetUser(ctx)
		ok, err := utils.CheckPermission(ctx, permission.ListFavorites, client)
		if err != nil {
			log.Error().Err(err).Msg("error checking permission")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !ok {
			log.Info().Interface("user", currentUser).Msg("user not allowed to list favorites")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		favorites, err := s.favoritesManager.ListFavorites(ctx, currentUser.Id)
		if err != nil {
			log.Error().Err(err).Msg("error getting favorites")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		infos := make([]*provider.ResourceInfo, 0, len(favorites))
		for i := range favorites {
			statRes, err := client.Stat(ctx, &providerv1beta1.StatRequest{Ref: &providerv1beta1.Reference{ResourceId: favorites[i]}})
			if err != nil {
				log.Error().Err(err).Msg("error getting resource info")
				continue
			}
			if statRes.Status.Code != rpcv1beta1.Code_CODE_OK {
				log.Error().Interface("stat_response", statRes).Msg("error getting resource info")
				continue
			}
			infos = append(infos, statRes.Info)
		}

		prefer := net.ParsePrefer(r.Header.Get("prefer"))
		returnMinimal := prefer[net.HeaderPreferReturn] == "minimal"

		responsesXML, err := propfind.MultistatusResponse(ctx, &propfind.XML{Prop: ff.Prop}, infos, s.c.PublicURL, namespace, nil, returnMinimal, nil)
		if err != nil {
			log.Error().Err(err).Msg("error formatting propfind")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set(net.HeaderDav, "1, 3, extended-mkcol")
		w.Header().Set(net.HeaderContentType, "application/xml; charset=utf-8")
		w.Header().Set(net.HeaderVary, net.HeaderPrefer)
		if returnMinimal {
			w.Header().Set(net.HeaderPreferenceApplied, "return=minimal")
		}
		w.WriteHeader(http.StatusMultiStatus)
		if _, err := w.Write(responsesXML); err != nil {
			log.Err(err).Msg("error writing response")
		}
	}
}

type report struct {
	SearchFiles *reportSearchFiles
	// FilterFiles TODO add this for tag based search
	FilterFiles *reportFilterFiles `xml:"filter-files"`
}
type reportSearchFiles struct {
	XMLName xml.Name                `xml:"search-files"`
	Lang    string                  `xml:"xml:lang,attr,omitempty"`
	Prop    propfind.Props          `xml:"DAV: prop"`
	Search  reportSearchFilesSearch `xml:"search"`
}
type reportSearchFilesSearch struct {
	Pattern string `xml:"search"`
	Limit   int    `xml:"limit"`
	Offset  int    `xml:"offset"`
}

type reportFilterFiles struct {
	XMLName xml.Name               `xml:"filter-files"`
	Lang    string                 `xml:"xml:lang,attr,omitempty"`
	Prop    propfind.Props         `xml:"DAV: prop"`
	Rules   reportFilterFilesRules `xml:"filter-rules"`
}

type reportFilterFilesRules struct {
	Favorite  bool `xml:"favorite"`
	SystemTag int  `xml:"systemtag"`
}

func readReport(r io.Reader) (rep *report, status int, err error) {
	decoder := xml.NewDecoder(r)
	rep = &report{}
	for {
		t, err := decoder.Token()
		if err == io.EOF {
			// io.EOF is a successful end
			return rep, 0, nil
		}
		if err != nil {
			return nil, http.StatusBadRequest, err
		}

		if v, ok := t.(xml.StartElement); ok {
			if v.Name.Local == elementNameSearchFiles {
				var repSF reportSearchFiles
				err = decoder.DecodeElement(&repSF, &v)
				if err != nil {
					return nil, http.StatusBadRequest, err
				}
				rep.SearchFiles = &repSF
			} else if v.Name.Local == elementNameFilterFiles {
				var repFF reportFilterFiles
				err = decoder.DecodeElement(&repFF, &v)
				if err != nil {
					return nil, http.StatusBadRequest, err
				}
				rep.FilterFiles = &repFF
			}
		}
	}
}
