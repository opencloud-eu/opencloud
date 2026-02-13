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

package propfind_test

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/ocdav/config"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/ocdav/net"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/ocdav/propfind"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type selector struct {
	client gateway.GatewayAPIClient
}

func (s selector) Next(opts ...pool.Option) (gateway.GatewayAPIClient, error) {
	return s.client, nil
}

var _ = Describe("PropfindWithDepthInfinity", func() {
	var (
		handler *propfind.Handler
		client  *mocks.GatewayAPIClient
		ctx     context.Context

		readResponse = func(r io.Reader) (*propfind.MultiStatusResponseUnmarshalXML, string, error) {
			buf, err := io.ReadAll(r)
			if err != nil {
				return nil, "", err
			}
			res := &propfind.MultiStatusResponseUnmarshalXML{}
			err = xml.Unmarshal(buf, res)
			if err != nil {
				return nil, "", err
			}

			return res, string(buf), nil
		}

		mockStat = func(ref *sprovider.Reference, info *sprovider.ResourceInfo) {
			client.On("Stat", mock.Anything, mock.MatchedBy(func(req *sprovider.StatRequest) bool {
				return utils.ResourceIDEqual(req.Ref.ResourceId, ref.ResourceId) &&
					(ref.Path == "" || req.Ref.Path == ref.Path)
			})).Return(&sprovider.StatResponse{
				Status: status.NewOK(ctx),
				Info:   info,
			}, nil)
		}
		mockListContainer = func(ref *sprovider.Reference, infos []*sprovider.ResourceInfo) {
			client.On("ListContainer", mock.Anything, mock.MatchedBy(func(req *sprovider.ListContainerRequest) bool {
				match := utils.ResourceIDEqual(req.Ref.ResourceId, ref.ResourceId) &&
					(ref.Path == "" || req.Ref.Path == ref.Path)
				return match
			})).Return(&sprovider.ListContainerResponse{
				Status: status.NewOK(ctx),
				Infos:  infos,
			}, nil)
		}

		foospace           *sprovider.StorageSpace
		fooquxspace        *sprovider.StorageSpace
		fooFileShareSpace  *sprovider.StorageSpace
		fooFileShare2Space *sprovider.StorageSpace
		fooDirShareSpace   *sprovider.StorageSpace
	)

	JustBeforeEach(func() {
		ctx = context.WithValue(context.Background(), net.CtxKeyBaseURI, "http://127.0.0.1:3000")
		client = &mocks.GatewayAPIClient{}
		sel := selector{
			client: client,
		}

		cfg := &config.Config{
			FilesNamespace:              "/users/{{.Username}}",
			WebdavNamespace:             "/users/{{.Username}}",
			AllowPropfindDepthInfinitiy: true,
			NameValidation: config.NameValidation{
				MaxLength:    255,
				InvalidChars: []string{"\f", "\r", "\n", "\\"},
			},
		}

		handler = propfind.NewHandler("127.0.0.1:3000", sel, nil, cfg)

		foospace = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo"),
					},
				},
			},
			Id:   &sprovider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})},
			Root: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"},
			Name: "foospace",
		}
		fooquxspace = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo/qux"),
					},
				},
			},
			Id:   &sprovider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "root"})},
			Root: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "root"},
			Name: "fooquxspace",
		}
		fooFileShareSpace = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo/Shares/sharedFile"),
					},
				},
			},
			Id:   &sprovider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace", OpaqueId: "sharedfile"})},
			Root: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace", OpaqueId: "sharedfile"},
			Name: "fooFileShareSpace",
		}
		fooFileShare2Space = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo/Shares/sharedFile2"),
					},
				},
			},
			Id:   &sprovider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace2", OpaqueId: "sharedfile2"})},
			Root: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace2", OpaqueId: "sharedfile2"},
			Name: "fooFileShareSpace2",
		}
		fooDirShareSpace = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo/Shares/sharedDir"),
					},
				},
			},
			Id:   &sprovider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddir"})},
			Root: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddir"},
			Name: "fooDirShareSpace",
		}

		// For the space mounted a /foo we assign a storageid "foospace" and a root opaqueid "root"
		// it contains four resources
		// - ./bar, file, 100 bytes, opaqueid "bar"
		// - ./baz, file, 1 byte, opaqueid "baz"
		// - ./dir, folder, 30 bytes, opaqueid "dir"
		// - ./dir/entry, file, 30 bytes, opaqueid "direntry"
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:   &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"},
				Type: sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path: ".",
				Size: uint64(131),
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "."},
			[]*sprovider.ResourceInfo{
				{
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "bar"},
					Path:     "bar",
					Size:     100,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
				{
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "baz"},
					Path:     "baz",
					Size:     1,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
				{
					Type:     sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"},
					Path:     "dir",
					Size:     30,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./bar"},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "bar"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:     "./bar",
				Size:     uint64(100),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "bar"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "bar"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:     "./bar",
				Size:     uint64(100),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./baz"},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "baz"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:     "./baz",
				Size:     uint64(1),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "baz"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "baz"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:     "./baz",
				Size:     uint64(1),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./dir"},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path:     "./dir",
				Size:     uint64(30),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./dir&dir"},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path:     "./dir&dir",
				Name:     "dir&dir",
				Size:     uint64(30),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path:     "./dir",
				Size:     uint64(30),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./dir"},
			[]*sprovider.ResourceInfo{
				{
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "direntry"},
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Path:     "entry",
					Size:     30,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./dir&dir"},
			[]*sprovider.ResourceInfo{
				{
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "direntry"},
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Path:     "entry",
					Size:     30,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"}, Path: "."},
			[]*sprovider.ResourceInfo{
				{
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "direntry"},
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Path:     "entry",
					Size:     30,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
			})

		// For the space mounted a /foo/qux we assign a storageid "foospace" and a root opaqueid "root"
		// it contains one resource
		// - ./quux, file, 1000 bytes, opaqueid "quux"
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "root"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:   &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "root"},
				Type: sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path: ".",
				Size: uint64(1000),
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "root"}, Path: "."},
			[]*sprovider.ResourceInfo{
				{
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "quux"},
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Path:     "./quux",
					Size:     1000,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "fooquxspace"},
				},
			})

		// For the space mounted a /foo/Shares/sharedFile we assign a spaceid "fooFileShareSpace" and a root opaqueid "sharedfile"
		// it is a file resource, 2000 bytes, opaqueid "sharedfile"
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace", OpaqueId: "sharedfile"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:    &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace", OpaqueId: "sharedfile"},
				Type:  sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:  ".",
				Size:  uint64(2000),
				Mtime: &typesv1beta1.Timestamp{Seconds: 1},
				Etag:  "1",
			})

		// For the space mounted a /foo/Shares/sharedFile2 we assign a spaceid "fooFileShareSpace2" and a root opaqueid "sharedfile2"
		// it is a file resource, 2500 bytes, opaqueid "sharedfile2"
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace2", OpaqueId: "sharedfile2"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:    &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace2", OpaqueId: "sharedfile2"},
				Type:  sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:  ".",
				Size:  uint64(2500),
				Mtime: &typesv1beta1.Timestamp{Seconds: 2},
				Etag:  "2",
			})

		// For the space mounted a /foo/Shares/sharedFile2 we assign a spaceid "fooDirShareSpace" and a root opaqueid "shareddir"
		// it is a folder containing one resource
		// ./something, file, 1500 bytes, opaqueid "shareddirsomething"
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddir"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:    &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddir"},
				Type:  sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path:  ".",
				Size:  uint64(1500),
				Mtime: &typesv1beta1.Timestamp{Seconds: 3},
				Etag:  "3",
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddir"}, Path: "."},
			[]*sprovider.ResourceInfo{
				{
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddirsomething"},
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Path:     "something",
					Size:     1500,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "fooDirShareSpace"},
				},
			})

		client.On("ListPublicShares", mock.Anything, mock.Anything).Return(
			func(_ context.Context, req *link.ListPublicSharesRequest, _ ...grpc.CallOption) *link.ListPublicSharesResponse {

				var shares []*link.PublicShare
				if len(req.Filters) == 0 {
					shares = []*link.PublicShare{}
				} else {
					term := req.Filters[0].Term.(*link.ListPublicSharesRequest_Filter_ResourceId)
					switch {
					case term != nil && term.ResourceId != nil && term.ResourceId.OpaqueId == "bar":
						shares = []*link.PublicShare{
							{
								Id:         &link.PublicShareId{OpaqueId: "share1"},
								ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "bar"},
							},
						}
					default:
						shares = []*link.PublicShare{}
					}
				}
				return &link.ListPublicSharesResponse{
					Status: status.NewOK(ctx),
					Share:  shares,
				}
			}, nil)
	})

	Describe("NewHandler", func() {
		It("returns a handler", func() {
			Expect(handler).ToNot(BeNil())
		})
	})

	Describe("HandlePathPropfind", func() {
		Context("with just one space", func() {
			JustBeforeEach(func() {
				client.On("ListStorageSpaces", mock.Anything, mock.MatchedBy(func(req *sprovider.ListStorageSpacesRequest) bool {
					p := string(req.Opaque.Map["path"].Value)
					return p == "/" || strings.HasPrefix(p, "/foo")
				})).Return(&sprovider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*sprovider.StorageSpace{foospace},
				}, nil)
				client.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&sprovider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*sprovider.StorageSpace{},
				}, nil)
				mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"}, Path: "."},
					&sprovider.ResourceInfo{
						Id:   &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
						Type: sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
						Path: ".",
						Size: uint64(131),
					})
			})

			It("verifies the depth header", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				req.Header.Set(net.HeaderDepth, "invalid")
				req = req.WithContext(ctx)
				Expect(err).ToNot(HaveOccurred())

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("stats a path", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				req = req.WithContext(ctx)
				Expect(err).ToNot(HaveOccurred())

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(4))

				root := res.Responses[0]
				Expect(root.Href).To(Equal("http:/127.0.0.1:3000/foo/"))
				Expect(string(root.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>131</oc:size>"))

				bar := res.Responses[1]
				Expect(bar.Href).To(Equal("http:/127.0.0.1:3000/foo/bar"))
				Expect(string(bar.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>100</d:getcontentlength>"))

				baz := res.Responses[2]
				Expect(baz.Href).To(Equal("http:/127.0.0.1:3000/foo/baz"))
				Expect(string(baz.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>1</d:getcontentlength>"))

				dir := res.Responses[3]
				Expect(dir.Href).To(Equal("http:/127.0.0.1:3000/foo/dir/"))
				Expect(string(dir.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>30</oc:size>"))
			})

			It("stats a file", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo/bar", strings.NewReader(""))
				req = req.WithContext(ctx)
				Expect(err).ToNot(HaveOccurred())

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(1))

				bar := res.Responses[0]
				Expect(bar.Href).To(Equal("http:/127.0.0.1:3000/foo/bar"))
				Expect(string(bar.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>100</d:getcontentlength>"))
			})
		})

		Context("with one nested file space", func() {
			JustBeforeEach(func() {
				client.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(
					func(_ context.Context, req *sprovider.ListStorageSpacesRequest, _ ...grpc.CallOption) *sprovider.ListStorageSpacesResponse {
						var spaces []*sprovider.StorageSpace
						switch string(req.Opaque.Map["path"].Value) {
						case "/", "/foo":
							spaces = []*sprovider.StorageSpace{foospace, fooFileShareSpace}
						case "/foo/Shares", "/foo/Shares/sharedFile":
							spaces = []*sprovider.StorageSpace{fooFileShareSpace}
						}
						return &sprovider.ListStorageSpacesResponse{
							Status:        status.NewOK(ctx),
							StorageSpaces: spaces,
						}
					},
					nil)
			})

			It("stats the parent", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(5))

				parent := res.Responses[0]
				Expect(parent.Href).To(Equal("http:/127.0.0.1:3000/foo/"))
				Expect(string(parent.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>2131</oc:size>"))

				sf := res.Responses[4]
				Expect(sf.Href).To(Equal("http:/127.0.0.1:3000/foo/Shares/"))
				Expect(string(sf.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>2000</oc:size>"))
			})

			It("stats the embedded space", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo/Shares/sharedFile", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(1))

				sf := res.Responses[0]
				Expect(sf.Href).To(Equal("http:/127.0.0.1:3000/foo/Shares/sharedFile"))
				Expect(string(sf.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>2000</d:getcontentlength>"))
				Expect(string(sf.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getlastmodified>Thu, 01 Jan 1970 00:00:01 GMT</d:getlastmodified>"))
				Expect(string(sf.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring(`<d:getetag>"1"</d:getetag>`))
			})
		})

		Context("with two nested file spaces and a nested directory space", func() {
			JustBeforeEach(func() {
				client.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(
					func(_ context.Context, req *sprovider.ListStorageSpacesRequest, _ ...grpc.CallOption) *sprovider.ListStorageSpacesResponse {
						var spaces []*sprovider.StorageSpace
						switch string(req.Opaque.Map["path"].Value) {
						case "/", "/foo":
							spaces = []*sprovider.StorageSpace{foospace, fooFileShareSpace, fooFileShare2Space, fooDirShareSpace}
						case "/foo/Shares":
							spaces = []*sprovider.StorageSpace{fooFileShareSpace, fooFileShare2Space, fooDirShareSpace}
						case "/foo/Shares/sharedFile":
							spaces = []*sprovider.StorageSpace{fooFileShareSpace}
						case "/foo/Shares/sharedFile2":
							spaces = []*sprovider.StorageSpace{fooFileShare2Space}
						}
						return &sprovider.ListStorageSpacesResponse{
							Status:        status.NewOK(ctx),
							StorageSpaces: spaces,
						}
					},
					nil)
			})

			It("stats the parent", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(5))

				parent := res.Responses[0]
				Expect(parent.Href).To(Equal("http:/127.0.0.1:3000/foo/"))
				Expect(string(parent.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>6131</oc:size>"))

				shares := res.Responses[4]
				Expect(shares.Href).To(Equal("http:/127.0.0.1:3000/foo/Shares/"))
				Expect(string(shares.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>6000</oc:size>"))
				Expect(string(shares.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getlastmodified>Thu, 01 Jan 1970 00:00:03 GMT</d:getlastmodified>"))
				Expect(string(shares.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring(`<d:getetag>"3"</d:getetag>`))
			})

			It("stats the embedded space", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo/Shares/sharedFile", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(1))

				sf := res.Responses[0]
				Expect(sf.Href).To(Equal("http:/127.0.0.1:3000/foo/Shares/sharedFile"))
				Expect(string(sf.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>2000</d:getcontentlength>"))
			})

			It("includes all the things™ when depth is infinity", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)
				req.Header.Add(net.HeaderDepth, "infinity")

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(9))

				paths := []string{}
				for _, r := range res.Responses {
					paths = append(paths, r.Href)
				}
				Expect(paths).To(ConsistOf(
					"http:/127.0.0.1:3000/foo/",
					"http:/127.0.0.1:3000/foo/bar",
					"http:/127.0.0.1:3000/foo/baz",
					"http:/127.0.0.1:3000/foo/dir/",
					"http:/127.0.0.1:3000/foo/dir/entry",
					"http:/127.0.0.1:3000/foo/Shares/sharedFile",
					"http:/127.0.0.1:3000/foo/Shares/sharedFile2",
					"http:/127.0.0.1:3000/foo/Shares/sharedDir/",
					"http:/127.0.0.1:3000/foo/Shares/sharedDir/something",
				))
			})
		})

		Context("with a nested directory space", func() {
			JustBeforeEach(func() {
				client.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(
					func(_ context.Context, req *sprovider.ListStorageSpacesRequest, _ ...grpc.CallOption) *sprovider.ListStorageSpacesResponse {
						var spaces []*sprovider.StorageSpace
						switch string(req.Opaque.Map["path"].Value) {
						case "/", "/foo":
							spaces = []*sprovider.StorageSpace{foospace, fooquxspace}
						case "/foo/qux":
							spaces = []*sprovider.StorageSpace{fooquxspace}
						}
						return &sprovider.ListStorageSpacesResponse{
							Status:        status.NewOK(ctx),
							StorageSpaces: spaces,
						}
					},
					nil)
			})

			// Pending, the code for handling missing parents is still missing
			PIt("handles children with no parent", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusOK))
			})

			It("mounts embedded spaces", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(5))

				root := res.Responses[0]
				Expect(root.Href).To(Equal("http:/127.0.0.1:3000/foo/"))
				Expect(string(root.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>1131</oc:size>"))

				bar := res.Responses[1]
				Expect(bar.Href).To(Equal("http:/127.0.0.1:3000/foo/bar"))
				Expect(string(bar.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>100</d:getcontentlength>"))

				baz := res.Responses[2]
				Expect(baz.Href).To(Equal("http:/127.0.0.1:3000/foo/baz"))
				Expect(string(baz.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>1</d:getcontentlength>"))

				dir := res.Responses[3]
				Expect(dir.Href).To(Equal("http:/127.0.0.1:3000/foo/dir/"))
				Expect(string(dir.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>30</oc:size>"))

				qux := res.Responses[4]
				Expect(qux.Href).To(Equal("http:/127.0.0.1:3000/foo/qux/"))
				Expect(string(qux.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>1000</oc:size>"))
			})

			It("stats the embedded space", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo/qux/", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(2))

				qux := res.Responses[0]
				Expect(qux.Href).To(Equal("http:/127.0.0.1:3000/foo/qux/"))
				Expect(string(qux.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>1000</oc:size>"))

				quux := res.Responses[1]
				Expect(quux.Href).To(Equal("http:/127.0.0.1:3000/foo/qux/quux"))
				Expect(string(quux.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>1000</d:getcontentlength>"))
			})
		})
	})

	Describe("HandleSpacesPropfind", func() {
		/*
			JustBeforeEach(func() {
				client.On("Stat", mock.Anything, mock.Anything).Return(func(_ context.Context, req *sprovider.StatRequest, _ ...grpc.CallOption) *sprovider.StatResponse {
					switch {
					case req.Ref.ResourceId.OpaqueId == "foospace":
						return &sprovider.StatResponse{
							Status: status.NewOK(ctx),
							Info: &sprovider.ResourceInfo{
								Type: sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
								Id:   &sprovider.ResourceId{OpaqueId: "foospaceroot", StorageId: "foospaceroot"},
								Size: 131,
								Path: ".",
							},
						}
					default:
						return &sprovider.StatResponse{
							Status: status.NewNotFound(ctx, "not found"),
						}
					}
				}, nil)
			})
		*/

		It("handles invalid space ids", func() {
			client.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)

			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/", strings.NewReader(""))
			Expect(err).ToNot(HaveOccurred())

			handler.HandleSpacesPropfind(rr, req, "does-not-exist")
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("stats the space root", func() {
			client.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/", strings.NewReader(""))
			Expect(err).ToNot(HaveOccurred())
			req = req.WithContext(ctx)

			spaceID := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})
			spaceIDUrl := net.EncodePath(spaceID)
			handler.HandleSpacesPropfind(rr, req, spaceID)
			Expect(rr.Code).To(Equal(http.StatusMultiStatus))

			res, _, err := readResponse(rr.Result().Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res.Responses)).To(Equal(4))

			root := res.Responses[0]
			Expect(root.Href).To(Equal("http:/127.0.0.1:3000/" + spaceIDUrl + "/"))
			Expect(string(root.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>131</oc:size>"))

			bar := res.Responses[1]
			Expect(bar.Href).To(Equal("http:/127.0.0.1:3000/" + spaceIDUrl + "/bar"))
			Expect(string(bar.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>100</d:getcontentlength>"))
			Expect(string(bar.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:file-parent>provider-1$foospace!foospace</oc:file-parent>"))

			baz := res.Responses[2]
			Expect(baz.Href).To(Equal("http:/127.0.0.1:3000/" + spaceIDUrl + "/baz"))
			Expect(string(baz.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>1</d:getcontentlength>"))
			Expect(string(baz.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:file-parent>provider-1$foospace!foospace</oc:file-parent>"))

			dir := res.Responses[3]
			Expect(dir.Href).To(Equal("http:/127.0.0.1:3000/" + spaceIDUrl + "/dir/"))
			Expect(string(dir.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>30</oc:size>"))
			Expect(string(dir.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:file-parent>provider-1$foospace!foospace</oc:file-parent>"))
		})

		It("stats a file", func() {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/bar", strings.NewReader(""))
			Expect(err).ToNot(HaveOccurred())
			req = req.WithContext(ctx)

			spaceID := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})
			handler.HandleSpacesPropfind(rr, req, spaceID)
			Expect(rr.Code).To(Equal(http.StatusMultiStatus))

			res, _, err := readResponse(rr.Result().Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res.Responses)).To(Equal(1))
			Expect(string(res.Responses[0].Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>100</d:getcontentlength>"))
		})

		It("stats a directory", func() {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/dir", strings.NewReader(""))
			Expect(err).ToNot(HaveOccurred())
			req = req.WithContext(ctx)

			spaceID := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})
			handler.HandleSpacesPropfind(rr, req, spaceID)
			Expect(rr.Code).To(Equal(http.StatusMultiStatus))

			res, _, err := readResponse(rr.Result().Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res.Responses)).To(Equal(2))
			Expect(string(res.Responses[0].Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>30</oc:size>"))
			Expect(string(res.Responses[1].Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>30</d:getcontentlength>"))
		})

		It("stats a directory with xml special characters", func() {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/dir&dir", strings.NewReader(""))
			Expect(err).ToNot(HaveOccurred())
			req = req.WithContext(ctx)

			spaceID := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})
			handler.HandleSpacesPropfind(rr, req, spaceID)
			Expect(rr.Code).To(Equal(http.StatusMultiStatus))

			res, _, err := readResponse(rr.Result().Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res.Responses)).To(Equal(2))
			Expect(string(res.Responses[0].Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:name>dir&amp;dir</oc:name>"))
			Expect(string(res.Responses[1].Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>30</d:getcontentlength>"))
		})

		It("includes all the things™ when depth is infinity", func() {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/", strings.NewReader(""))
			req.Header.Add(net.HeaderDepth, "infinity")
			Expect(err).ToNot(HaveOccurred())
			req = req.WithContext(ctx)

			spaceID := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})
			handler.HandleSpacesPropfind(rr, req, spaceID)
			Expect(rr.Code).To(Equal(http.StatusMultiStatus))

			_, _, err = readResponse(rr.Result().Body)
			Expect(err).ToNot(HaveOccurred())

		})
	})
})

var _ = Describe("PropfindWithoutDepthInfinity", func() {
	var (
		handler *propfind.Handler
		client  *mocks.GatewayAPIClient
		ctx     context.Context

		readResponse = func(r io.Reader) (*propfind.MultiStatusResponseUnmarshalXML, string, error) {
			buf, err := io.ReadAll(r)
			if err != nil {
				return nil, "", err
			}
			res := &propfind.MultiStatusResponseUnmarshalXML{}
			err = xml.Unmarshal(buf, res)
			if err != nil {
				return nil, "", err
			}

			return res, string(buf), nil
		}

		mockStat = func(ref *sprovider.Reference, info *sprovider.ResourceInfo) {
			client.On("Stat", mock.Anything, mock.MatchedBy(func(req *sprovider.StatRequest) bool {
				return utils.ResourceIDEqual(req.Ref.ResourceId, ref.ResourceId) &&
					(ref.Path == "" || req.Ref.Path == ref.Path)
			})).Return(&sprovider.StatResponse{
				Status: status.NewOK(ctx),
				Info:   info,
			}, nil)
		}
		mockListContainer = func(ref *sprovider.Reference, infos []*sprovider.ResourceInfo) {
			client.On("ListContainer", mock.Anything, mock.MatchedBy(func(req *sprovider.ListContainerRequest) bool {
				match := utils.ResourceIDEqual(req.Ref.ResourceId, ref.ResourceId) &&
					(ref.Path == "" || req.Ref.Path == ref.Path)
				return match
			})).Return(&sprovider.ListContainerResponse{
				Status: status.NewOK(ctx),
				Infos:  infos,
			}, nil)
		}

		foospace           *sprovider.StorageSpace
		fooquxspace        *sprovider.StorageSpace
		fooFileShareSpace  *sprovider.StorageSpace
		fooFileShare2Space *sprovider.StorageSpace
		fooDirShareSpace   *sprovider.StorageSpace
	)

	JustBeforeEach(func() {
		ctx = context.WithValue(context.Background(), net.CtxKeyBaseURI, "http://127.0.0.1:3000")
		client = &mocks.GatewayAPIClient{}
		sel := selector{
			client: client,
		}

		cfg := &config.Config{
			FilesNamespace:              "/users/{{.Username}}",
			WebdavNamespace:             "/users/{{.Username}}",
			AllowPropfindDepthInfinitiy: false,
			NameValidation: config.NameValidation{
				MaxLength:    255,
				InvalidChars: []string{"\f", "\r", "\n", "\\"},
			},
		}

		handler = propfind.NewHandler("127.0.0.1:3000", sel, nil, cfg)

		foospace = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo"),
					},
				},
			},
			Id:   &sprovider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})},
			Root: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"},
			Name: "foospace",
		}
		fooquxspace = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo/qux"),
					},
				},
			},
			Id:   &sprovider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "root"})},
			Root: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "root"},
			Name: "fooquxspace",
		}
		fooFileShareSpace = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo/Shares/sharedFile"),
					},
				},
			},
			Id:   &sprovider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace", OpaqueId: "sharedfile"})},
			Root: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace", OpaqueId: "sharedfile"},
			Name: "fooFileShareSpace",
		}
		fooFileShare2Space = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo/Shares/sharedFile2"),
					},
				},
			},
			Id:   &sprovider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace2", OpaqueId: "sharedfile2"})},
			Root: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace2", OpaqueId: "sharedfile2"},
			Name: "fooFileShareSpace2",
		}
		fooDirShareSpace = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo/Shares/sharedDir"),
					},
				},
			},
			Id:   &sprovider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddir"})},
			Root: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddir"},
			Name: "fooDirShareSpace",
		}

		// For the space mounted a /foo we assign a storageid "foospace" and a root opaqueid "root"
		// it contains four resources
		// - ./bar, file, 100 bytes, opaqueid "bar"
		// - ./baz, file, 1 byte, opaqueid "baz"
		// - ./dir, folder, 30 bytes, opaqueid "dir"
		// - ./dir/entry, file, 30 bytes, opaqueid "direntry"
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:   &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"},
				Type: sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path: ".",
				Size: uint64(131),
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "."},
			[]*sprovider.ResourceInfo{
				{
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "bar"},
					Path:     "bar",
					Size:     100,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
				{
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "baz"},
					Path:     "baz",
					Size:     1,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
				{
					Type:     sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"},
					Path:     "dir",
					Size:     30,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./bar"},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "bar"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:     "./bar",
				Size:     uint64(100),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "bar"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "bar"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:     "./bar",
				Size:     uint64(100),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./baz"},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "baz"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:     "./baz",
				Size:     uint64(1),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "baz"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "baz"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:     "./baz",
				Size:     uint64(1),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./dir"},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path:     "./dir",
				Size:     uint64(30),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./dir&dir"},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path:     "./dir&dir",
				Name:     "dir&dir",
				Size:     uint64(30),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"},
				Type:     sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path:     "./dir",
				Size:     uint64(30),
				ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./dir"},
			[]*sprovider.ResourceInfo{
				{
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "direntry"},
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Path:     "entry",
					Size:     30,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"}, Path: "./dir&dir"},
			[]*sprovider.ResourceInfo{
				{
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "direntry"},
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Path:     "entry",
					Size:     30,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "dir"}, Path: "."},
			[]*sprovider.ResourceInfo{
				{
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "direntry"},
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Path:     "entry",
					Size:     30,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
				},
			})

		// For the space mounted a /foo/qux we assign a storageid "foospace" and a root opaqueid "root"
		// it contains one resource
		// - ./quux, file, 1000 bytes, opaqueid "quux"
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "root"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:   &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "root"},
				Type: sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path: ".",
				Size: uint64(1000),
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "root"}, Path: "."},
			[]*sprovider.ResourceInfo{
				{
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "quux"},
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Path:     "./quux",
					Size:     1000,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooquxspace", OpaqueId: "fooquxspace"},
				},
			})

		// For the space mounted a /foo/Shares/sharedFile we assign a spaceid "fooFileShareSpace" and a root opaqueid "sharedfile"
		// it is a file resource, 2000 bytes, opaqueid "sharedfile"
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace", OpaqueId: "sharedfile"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:    &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace", OpaqueId: "sharedfile"},
				Type:  sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:  ".",
				Size:  uint64(2000),
				Mtime: &typesv1beta1.Timestamp{Seconds: 1},
				Etag:  "1",
			})

		// For the space mounted a /foo/Shares/sharedFile2 we assign a spaceid "fooFileShareSpace2" and a root opaqueid "sharedfile2"
		// it is a file resource, 2500 bytes, opaqueid "sharedfile2"
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace2", OpaqueId: "sharedfile2"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:    &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooFileShareSpace2", OpaqueId: "sharedfile2"},
				Type:  sprovider.ResourceType_RESOURCE_TYPE_FILE,
				Path:  ".",
				Size:  uint64(2500),
				Mtime: &typesv1beta1.Timestamp{Seconds: 2},
				Etag:  "2",
			})

		// For the space mounted a /foo/Shares/sharedFile2 we assign a spaceid "fooDirShareSpace" and a root opaqueid "shareddir"
		// it is a folder containing one resource
		// ./something, file, 1500 bytes, opaqueid "shareddirsomething"
		mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddir"}, Path: "."},
			&sprovider.ResourceInfo{
				Id:    &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddir"},
				Type:  sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Path:  ".",
				Size:  uint64(1500),
				Mtime: &typesv1beta1.Timestamp{Seconds: 3},
				Etag:  "3",
			})
		mockListContainer(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddir"}, Path: "."},
			[]*sprovider.ResourceInfo{
				{
					Id:       &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "shareddirsomething"},
					Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
					Path:     "something",
					Size:     1500,
					ParentId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "fooDirShareSpace", OpaqueId: "fooDirShareSpace"},
				},
			})

		client.On("ListPublicShares", mock.Anything, mock.Anything).Return(
			func(_ context.Context, req *link.ListPublicSharesRequest, _ ...grpc.CallOption) *link.ListPublicSharesResponse {

				var shares []*link.PublicShare
				if len(req.Filters) == 0 {
					shares = []*link.PublicShare{}
				} else {
					term := req.Filters[0].Term.(*link.ListPublicSharesRequest_Filter_ResourceId)
					switch {
					case term != nil && term.ResourceId != nil && term.ResourceId.OpaqueId == "bar":
						shares = []*link.PublicShare{
							{
								Id:         &link.PublicShareId{OpaqueId: "share1"},
								ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "bar"},
							},
						}
					default:
						shares = []*link.PublicShare{}
					}
				}
				return &link.ListPublicSharesResponse{
					Status: status.NewOK(ctx),
					Share:  shares,
				}
			}, nil)
	})

	Describe("NewHandler", func() {
		It("returns a handler", func() {
			Expect(handler).ToNot(BeNil())
		})
	})

	Describe("HandlePathPropfind", func() {
		Context("with just one space", func() {
			JustBeforeEach(func() {
				client.On("ListStorageSpaces", mock.Anything, mock.MatchedBy(func(req *sprovider.ListStorageSpacesRequest) bool {
					p := string(req.Opaque.Map["path"].Value)
					return p == "/" || strings.HasPrefix(p, "/foo")
				})).Return(&sprovider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*sprovider.StorageSpace{foospace},
				}, nil)
				client.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&sprovider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*sprovider.StorageSpace{},
				}, nil)
				mockStat(&sprovider.Reference{ResourceId: &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"}, Path: "."},
					&sprovider.ResourceInfo{
						Id:   &sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "foospace"},
						Type: sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
						Path: ".",
						Size: uint64(131),
					})
			})

			It("verifies the depth header", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				req.Header.Set(net.HeaderDepth, "invalid")
				req = req.WithContext(ctx)
				Expect(err).ToNot(HaveOccurred())

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("stats a path", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				req = req.WithContext(ctx)
				Expect(err).ToNot(HaveOccurred())

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(4))

				root := res.Responses[0]
				Expect(root.Href).To(Equal("http:/127.0.0.1:3000/foo/"))
				Expect(string(root.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>131</oc:size>"))

				bar := res.Responses[1]
				Expect(bar.Href).To(Equal("http:/127.0.0.1:3000/foo/bar"))
				Expect(string(bar.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>100</d:getcontentlength>"))

				baz := res.Responses[2]
				Expect(baz.Href).To(Equal("http:/127.0.0.1:3000/foo/baz"))
				Expect(string(baz.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>1</d:getcontentlength>"))

				dir := res.Responses[3]
				Expect(dir.Href).To(Equal("http:/127.0.0.1:3000/foo/dir/"))
				Expect(string(dir.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>30</oc:size>"))
			})

			It("stats a file", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo/bar", strings.NewReader(""))
				req = req.WithContext(ctx)
				Expect(err).ToNot(HaveOccurred())

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(1))

				bar := res.Responses[0]
				Expect(bar.Href).To(Equal("http:/127.0.0.1:3000/foo/bar"))
				Expect(string(bar.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>100</d:getcontentlength>"))
			})
		})

		Context("with one nested file space", func() {
			JustBeforeEach(func() {
				client.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(
					func(_ context.Context, req *sprovider.ListStorageSpacesRequest, _ ...grpc.CallOption) *sprovider.ListStorageSpacesResponse {
						var spaces []*sprovider.StorageSpace
						switch string(req.Opaque.Map["path"].Value) {
						case "/", "/foo":
							spaces = []*sprovider.StorageSpace{foospace, fooFileShareSpace}
						case "/foo/Shares", "/foo/Shares/sharedFile":
							spaces = []*sprovider.StorageSpace{fooFileShareSpace}
						}
						return &sprovider.ListStorageSpacesResponse{
							Status:        status.NewOK(ctx),
							StorageSpaces: spaces,
						}
					},
					nil)
			})

			It("stats the parent", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(5))

				parent := res.Responses[0]
				Expect(parent.Href).To(Equal("http:/127.0.0.1:3000/foo/"))
				Expect(string(parent.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>2131</oc:size>"))

				sf := res.Responses[4]
				Expect(sf.Href).To(Equal("http:/127.0.0.1:3000/foo/Shares/"))
				Expect(string(sf.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>2000</oc:size>"))
			})

			It("stats the embedded space", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo/Shares/sharedFile", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(1))

				sf := res.Responses[0]
				Expect(sf.Href).To(Equal("http:/127.0.0.1:3000/foo/Shares/sharedFile"))
				Expect(string(sf.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>2000</d:getcontentlength>"))
				Expect(string(sf.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getlastmodified>Thu, 01 Jan 1970 00:00:01 GMT</d:getlastmodified>"))
				Expect(string(sf.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring(`<d:getetag>"1"</d:getetag>`))
			})
		})

		Context("with two nested file spaces and a nested directory space", func() {
			JustBeforeEach(func() {
				client.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(
					func(_ context.Context, req *sprovider.ListStorageSpacesRequest, _ ...grpc.CallOption) *sprovider.ListStorageSpacesResponse {
						var spaces []*sprovider.StorageSpace
						switch string(req.Opaque.Map["path"].Value) {
						case "/", "/foo":
							spaces = []*sprovider.StorageSpace{foospace, fooFileShareSpace, fooFileShare2Space, fooDirShareSpace}
						case "/foo/Shares":
							spaces = []*sprovider.StorageSpace{fooFileShareSpace, fooFileShare2Space, fooDirShareSpace}
						case "/foo/Shares/sharedFile":
							spaces = []*sprovider.StorageSpace{fooFileShareSpace}
						case "/foo/Shares/sharedFile2":
							spaces = []*sprovider.StorageSpace{fooFileShare2Space}
						}
						return &sprovider.ListStorageSpacesResponse{
							Status:        status.NewOK(ctx),
							StorageSpaces: spaces,
						}
					},
					nil)
			})

			It("stats the parent", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(5))

				parent := res.Responses[0]
				Expect(parent.Href).To(Equal("http:/127.0.0.1:3000/foo/"))
				Expect(string(parent.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>6131</oc:size>"))

				shares := res.Responses[4]
				Expect(shares.Href).To(Equal("http:/127.0.0.1:3000/foo/Shares/"))
				Expect(string(shares.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>6000</oc:size>"))
				Expect(string(shares.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getlastmodified>Thu, 01 Jan 1970 00:00:03 GMT</d:getlastmodified>"))
				Expect(string(shares.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring(`<d:getetag>"3"</d:getetag>`))
			})

			It("stats the embedded space", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo/Shares/sharedFile", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(1))

				sf := res.Responses[0]
				Expect(sf.Href).To(Equal("http:/127.0.0.1:3000/foo/Shares/sharedFile"))
				Expect(string(sf.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>2000</d:getcontentlength>"))
			})

			It("includes all the things™ when depth is infinity", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)
				req.Header.Add(net.HeaderDepth, "infinity")

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("with a nested directory space", func() {
			JustBeforeEach(func() {
				client.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(
					func(_ context.Context, req *sprovider.ListStorageSpacesRequest, _ ...grpc.CallOption) *sprovider.ListStorageSpacesResponse {
						var spaces []*sprovider.StorageSpace
						switch string(req.Opaque.Map["path"].Value) {
						case "/", "/foo":
							spaces = []*sprovider.StorageSpace{foospace, fooquxspace}
						case "/foo/qux":
							spaces = []*sprovider.StorageSpace{fooquxspace}
						}
						return &sprovider.ListStorageSpacesResponse{
							Status:        status.NewOK(ctx),
							StorageSpaces: spaces,
						}
					},
					nil)
			})

			// Pending, the code for handling missing parents is still missing
			PIt("handles children with no parent", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusOK))
			})

			It("mounts embedded spaces", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(5))

				root := res.Responses[0]
				Expect(root.Href).To(Equal("http:/127.0.0.1:3000/foo/"))
				Expect(string(root.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>1131</oc:size>"))

				bar := res.Responses[1]
				Expect(bar.Href).To(Equal("http:/127.0.0.1:3000/foo/bar"))
				Expect(string(bar.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>100</d:getcontentlength>"))

				baz := res.Responses[2]
				Expect(baz.Href).To(Equal("http:/127.0.0.1:3000/foo/baz"))
				Expect(string(baz.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>1</d:getcontentlength>"))

				dir := res.Responses[3]
				Expect(dir.Href).To(Equal("http:/127.0.0.1:3000/foo/dir/"))
				Expect(string(dir.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>30</oc:size>"))

				qux := res.Responses[4]
				Expect(qux.Href).To(Equal("http:/127.0.0.1:3000/foo/qux/"))
				Expect(string(qux.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>1000</oc:size>"))
			})

			It("stats the embedded space", func() {
				rr := httptest.NewRecorder()
				req, err := http.NewRequest("GET", "/foo/qux/", strings.NewReader(""))
				Expect(err).ToNot(HaveOccurred())
				req = req.WithContext(ctx)

				handler.HandlePathPropfind(rr, req, "/")
				Expect(rr.Code).To(Equal(http.StatusMultiStatus))

				res, _, err := readResponse(rr.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(res.Responses)).To(Equal(2))

				qux := res.Responses[0]
				Expect(qux.Href).To(Equal("http:/127.0.0.1:3000/foo/qux/"))
				Expect(string(qux.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>1000</oc:size>"))

				quux := res.Responses[1]
				Expect(quux.Href).To(Equal("http:/127.0.0.1:3000/foo/qux/quux"))
				Expect(string(quux.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>1000</d:getcontentlength>"))
			})
		})
	})

	Describe("HandleSpacesPropfind", func() {
		/*
			JustBeforeEach(func() {
				client.On("Stat", mock.Anything, mock.Anything).Return(func(_ context.Context, req *sprovider.StatRequest, _ ...grpc.CallOption) *sprovider.StatResponse {
					switch {
					case req.Ref.ResourceId.OpaqueId == "foospace":
						return &sprovider.StatResponse{
							Status: status.NewOK(ctx),
							Info: &sprovider.ResourceInfo{
								Type: sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
								Id:   &sprovider.ResourceId{OpaqueId: "foospaceroot", StorageId: "foospaceroot"},
								Size: 131,
								Path: ".",
							},
						}
					default:
						return &sprovider.StatResponse{
							Status: status.NewNotFound(ctx, "not found"),
						}
					}
				}, nil)
			})
		*/

		It("handles invalid space ids", func() {
			client.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)

			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/", strings.NewReader(""))
			Expect(err).ToNot(HaveOccurred())

			handler.HandleSpacesPropfind(rr, req, "does-not-exist")
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("stats the space root", func() {
			client.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/", strings.NewReader(""))
			Expect(err).ToNot(HaveOccurred())
			req = req.WithContext(ctx)

			spaceID := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})
			spaceIDUrl := net.EncodePath(spaceID)
			handler.HandleSpacesPropfind(rr, req, spaceID)
			Expect(rr.Code).To(Equal(http.StatusMultiStatus))

			res, _, err := readResponse(rr.Result().Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res.Responses)).To(Equal(4))

			root := res.Responses[0]
			Expect(root.Href).To(Equal("http:/127.0.0.1:3000/" + spaceIDUrl + "/"))
			Expect(string(root.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>131</oc:size>"))

			bar := res.Responses[1]
			Expect(bar.Href).To(Equal("http:/127.0.0.1:3000/" + spaceIDUrl + "/bar"))
			Expect(string(bar.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>100</d:getcontentlength>"))
			Expect(string(bar.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:file-parent>provider-1$foospace!foospace</oc:file-parent>"))

			baz := res.Responses[2]
			Expect(baz.Href).To(Equal("http:/127.0.0.1:3000/" + spaceIDUrl + "/baz"))
			Expect(string(baz.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>1</d:getcontentlength>"))
			Expect(string(baz.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:file-parent>provider-1$foospace!foospace</oc:file-parent>"))

			dir := res.Responses[3]
			Expect(dir.Href).To(Equal("http:/127.0.0.1:3000/" + spaceIDUrl + "/dir/"))
			Expect(string(dir.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>30</oc:size>"))
			Expect(string(dir.Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:file-parent>provider-1$foospace!foospace</oc:file-parent>"))
		})

		It("stats a file", func() {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/bar", strings.NewReader(""))
			Expect(err).ToNot(HaveOccurred())
			req = req.WithContext(ctx)

			spaceID := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})
			handler.HandleSpacesPropfind(rr, req, spaceID)
			Expect(rr.Code).To(Equal(http.StatusMultiStatus))

			res, _, err := readResponse(rr.Result().Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res.Responses)).To(Equal(1))
			Expect(string(res.Responses[0].Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>100</d:getcontentlength>"))
		})

		It("stats a directory", func() {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/dir", strings.NewReader(""))
			Expect(err).ToNot(HaveOccurred())
			req = req.WithContext(ctx)

			spaceID := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})
			handler.HandleSpacesPropfind(rr, req, spaceID)
			Expect(rr.Code).To(Equal(http.StatusMultiStatus))

			res, _, err := readResponse(rr.Result().Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res.Responses)).To(Equal(2))
			Expect(string(res.Responses[0].Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:size>30</oc:size>"))
			Expect(string(res.Responses[1].Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>30</d:getcontentlength>"))
		})

		It("stats a directory with xml special characters", func() {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/dir&dir", strings.NewReader(""))
			Expect(err).ToNot(HaveOccurred())
			req = req.WithContext(ctx)

			spaceID := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})
			handler.HandleSpacesPropfind(rr, req, spaceID)
			Expect(rr.Code).To(Equal(http.StatusMultiStatus))

			res, _, err := readResponse(rr.Result().Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res.Responses)).To(Equal(2))
			Expect(string(res.Responses[0].Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<oc:name>dir&amp;dir</oc:name>"))
			Expect(string(res.Responses[1].Propstat[0].Prop[0].InnerXML)).To(ContainSubstring("<d:getcontentlength>30</d:getcontentlength>"))
		})

		It("includes all the things™ when depth is infinity", func() {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/", strings.NewReader(""))
			req.Header.Add(net.HeaderDepth, "infinity")
			Expect(err).ToNot(HaveOccurred())
			req = req.WithContext(ctx)

			spaceID := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "provider-1", SpaceId: "foospace", OpaqueId: "root"})
			handler.HandleSpacesPropfind(rr, req, spaceID)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))

			_, _, err = readResponse(rr.Result().Body)
			Expect(err).To(HaveOccurred())

		})
	})
})
