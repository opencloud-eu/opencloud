package connector_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"

	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/services/collaboration/mocks"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/connector"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/connector/fileinfo"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("HttpAdapter", func() {
	var (
		fc          *mocks.FileConnectorService
		cc          *mocks.ContentConnectorService
		con         *mocks.ConnectorService
		locks       *mocks.LockParser
		httpAdapter *connector.HttpAdapter
	)

	BeforeEach(func() {
		fc = &mocks.FileConnectorService{}
		cc = &mocks.ContentConnectorService{}

		con = &mocks.ConnectorService{}
		con.On("GetContentConnector").Return(cc)
		con.On("GetFileConnector").Return(fc)

		locks = &mocks.LockParser{}
		locks.EXPECT().ParseLock(mock.Anything).RunAndReturn(func(id string) string {
			return id
		})

		httpAdapter = connector.NewHttpAdapterWithConnector(con, locks)
	})

	Describe("GetLock", func() {
		It("General error", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "POST_LOCK")

			w := httptest.NewRecorder()

			fc.On("GetLock", mock.Anything).Times(1).Return(nil, errors.New("Something happened"))

			httpAdapter.GetLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(500))
		})

		It("File not found", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "POST_LOCK")

			w := httptest.NewRecorder()

			fc.On("GetLock", mock.Anything).Times(1).Return(connector.NewResponse(404), nil)

			httpAdapter.GetLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("LockId", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "POST_LOCK")

			w := httptest.NewRecorder()

			fc.On("GetLock", mock.Anything).Times(1).Return(connector.NewResponseWithLock(200, "zzz111"), nil)

			httpAdapter.GetLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(200))
			Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("zzz111"))
		})

		It("Empty LockId", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "POST_LOCK")

			w := httptest.NewRecorder()

			fc.On("GetLock", mock.Anything).Times(1).Return(connector.NewResponseWithLock(200, ""), nil)

			httpAdapter.GetLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(200))
			Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal(""))
		})
	})

	Describe("Lock", func() {
		Describe("Just lock", func() {
			It("General error", func() {
				req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
				req.Header.Set("X-WOPI-Override", "LOCK")
				req.Header.Set(connector.HeaderWopiLock, "abc123")

				w := httptest.NewRecorder()

				fc.On("Lock", mock.Anything, "abc123", "").Times(1).Return(nil, errors.New("Something happened"))

				httpAdapter.Lock(w, req)
				resp := w.Result()
				Expect(resp.StatusCode).To(Equal(500))
			})

			It("No LockId provided", func() {
				req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
				req.Header.Set("X-WOPI-Override", "LOCK")
				req.Header.Set(connector.HeaderWopiLock, "")

				w := httptest.NewRecorder()

				fc.On("Lock", mock.Anything, "", "").Times(1).Return(connector.NewResponse(400), nil)

				httpAdapter.Lock(w, req)
				resp := w.Result()
				Expect(resp.StatusCode).To(Equal(400))
			})

			It("Conflict", func() {
				req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
				req.Header.Set("X-WOPI-Override", "LOCK")
				req.Header.Set(connector.HeaderWopiLock, "abc123")

				w := httptest.NewRecorder()

				fc.On("Lock", mock.Anything, "abc123", "").Times(1).Return(connector.NewResponseLockConflict("zzz111", "Lock Conflict"), nil)

				httpAdapter.Lock(w, req)
				resp := w.Result()
				Expect(resp.StatusCode).To(Equal(409))
				Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("zzz111"))
				Expect(resp.Header.Get(connector.HeaderWopiLockFailureReason)).To(Equal("Lock Conflict"))
			})

			It("Success", func() {
				req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
				req.Header.Set("X-WOPI-Override", "LOCK")
				req.Header.Set(connector.HeaderWopiLock, "abc123")

				w := httptest.NewRecorder()

				fc.On("Lock", mock.Anything, "abc123", "").Times(1).Return(
					connector.NewResponseWithVersionAndLock(
						200,
						&typesv1beta1.Timestamp{Seconds: uint64(1234), Nanos: uint32(567)},
						"abc123",
					), nil)

				httpAdapter.Lock(w, req)
				resp := w.Result()
				Expect(resp.StatusCode).To(Equal(200))
				Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("abc123"))
				Expect(resp.Header.Get(connector.HeaderWopiVersion)).To(Equal("v1234567"))
			})
		})

		Describe("Unlock and relock", func() {
			It("General error", func() {
				req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
				req.Header.Set("X-WOPI-Override", "LOCK")
				req.Header.Set(connector.HeaderWopiLock, "abc123")
				req.Header.Set(connector.HeaderWopiOldLock, "qwerty")

				w := httptest.NewRecorder()

				fc.On("Lock", mock.Anything, "abc123", "qwerty").Times(1).Return(nil, errors.New("Something happened"))

				httpAdapter.Lock(w, req)
				resp := w.Result()
				Expect(resp.StatusCode).To(Equal(500))
			})

			It("No LockId provided", func() {
				req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
				req.Header.Set("X-WOPI-Override", "LOCK")
				req.Header.Set(connector.HeaderWopiLock, "")
				req.Header.Set(connector.HeaderWopiOldLock, "")

				w := httptest.NewRecorder()

				fc.On("Lock", mock.Anything, "", "").Times(1).Return(connector.NewResponse(400), nil)

				httpAdapter.Lock(w, req)
				resp := w.Result()
				Expect(resp.StatusCode).To(Equal(400))
			})

			It("Conflict", func() {
				req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
				req.Header.Set("X-WOPI-Override", "LOCK")
				req.Header.Set(connector.HeaderWopiLock, "abc123")
				req.Header.Set(connector.HeaderWopiOldLock, "qwerty")

				w := httptest.NewRecorder()

				fc.On("Lock", mock.Anything, "abc123", "qwerty").Times(1).Return(connector.NewResponseLockConflict("zzz111", "Lock Conflict"), nil)

				httpAdapter.Lock(w, req)
				resp := w.Result()
				Expect(resp.StatusCode).To(Equal(409))
				Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("zzz111"))
				Expect(resp.Header.Get(connector.HeaderWopiLockFailureReason)).To(Equal("Lock Conflict"))
			})

			It("Success", func() {
				req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
				req.Header.Set("X-WOPI-Override", "LOCK")
				req.Header.Set(connector.HeaderWopiLock, "abc123")
				req.Header.Set(connector.HeaderWopiOldLock, "qwerty")

				w := httptest.NewRecorder()

				fc.On("Lock", mock.Anything, "abc123", "qwerty").Times(1).Return(
					connector.NewResponseWithVersionAndLock(
						200,
						&typesv1beta1.Timestamp{Seconds: uint64(1234), Nanos: uint32(567)},
						"abc123",
					), nil)

				httpAdapter.Lock(w, req)
				resp := w.Result()
				Expect(resp.StatusCode).To(Equal(200))
				Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("abc123"))
				Expect(resp.Header.Get(connector.HeaderWopiVersion)).To(Equal("v1234567"))
			})
		})
	})

	Describe("RefreshLock", func() {
		It("General error", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "REFRESH_LOCK")
			req.Header.Set(connector.HeaderWopiLock, "abc123")

			w := httptest.NewRecorder()

			fc.On("RefreshLock", mock.Anything, "abc123").Times(1).Return(nil, errors.New("Something happened"))

			httpAdapter.RefreshLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(500))
		})

		It("No LockId provided", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "REFRESH_LOCK")
			req.Header.Set(connector.HeaderWopiLock, "")

			w := httptest.NewRecorder()

			fc.On("RefreshLock", mock.Anything, "").Times(1).Return(connector.NewResponse(400), nil)

			httpAdapter.RefreshLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("Conflict", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "REFRESH_LOCK")
			req.Header.Set(connector.HeaderWopiLock, "abc123")

			w := httptest.NewRecorder()

			fc.On("RefreshLock", mock.Anything, "abc123").Times(1).Return(connector.NewResponseLockConflict("zzz111", "Lock Conflict"), nil)

			httpAdapter.RefreshLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(409))
			Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("zzz111"))
			Expect(resp.Header.Get(connector.HeaderWopiLockFailureReason)).To(Equal("Lock Conflict"))
		})

		It("Success", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "REFRESH_LOCK")
			req.Header.Set(connector.HeaderWopiLock, "abc123")

			w := httptest.NewRecorder()

			fc.On("RefreshLock", mock.Anything, "abc123").Times(1).Return(
				connector.NewResponseWithVersionAndLock(
					200,
					&typesv1beta1.Timestamp{Seconds: uint64(1234), Nanos: uint32(5678)},
					"abc123",
				), nil)

			httpAdapter.RefreshLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(200))
			Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("abc123"))
			Expect(resp.Header.Get(connector.HeaderWopiVersion)).To(Equal("v12345678"))
		})
	})

	Describe("Unlock", func() {
		It("General error", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "UNLOCK")
			req.Header.Set(connector.HeaderWopiLock, "abc123")

			w := httptest.NewRecorder()

			fc.On("UnLock", mock.Anything, "abc123").Times(1).Return(nil, errors.New("Something happened"))

			httpAdapter.UnLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(500))
		})

		It("No LockId provided", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "UNLOCK")
			req.Header.Set(connector.HeaderWopiLock, "")

			w := httptest.NewRecorder()

			fc.On("UnLock", mock.Anything, "").Times(1).Return(connector.NewResponse(400), nil)

			httpAdapter.UnLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("Conflict", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "UNLOCK")
			req.Header.Set(connector.HeaderWopiLock, "abc123")

			w := httptest.NewRecorder()

			fc.On("UnLock", mock.Anything, "abc123").Times(1).Return(connector.NewResponseLockConflict("zzz111", "Lock Conflict"), nil)

			httpAdapter.UnLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(409))
			Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("zzz111"))
			Expect(resp.Header.Get(connector.HeaderWopiLockFailureReason)).To(Equal("Lock Conflict"))
		})

		It("Success", func() {
			req := httptest.NewRequest("POST", "/wopi/files/abcdef", nil)
			req.Header.Set("X-WOPI-Override", "UNLOCK")
			req.Header.Set(connector.HeaderWopiLock, "abc123")

			w := httptest.NewRecorder()

			fc.On("UnLock", mock.Anything, "abc123").Times(1).Return(
				connector.NewResponseWithVersion(&typesv1beta1.Timestamp{Seconds: uint64(1234), Nanos: uint32(567)}), nil)

			httpAdapter.UnLock(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(200))
			Expect(resp.Header.Get(connector.HeaderWopiVersion)).To(Equal("v1234567"))
		})
	})

	Describe("CheckFileInfo", func() {
		It("General error", func() {
			req := httptest.NewRequest("GET", "/wopi/files/abcdef", nil)

			w := httptest.NewRecorder()

			fc.On("CheckFileInfo", mock.Anything).Times(1).Return(nil, errors.New("Something happened"))

			httpAdapter.CheckFileInfo(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(500))
		})

		It("Not found", func() {
			// 404 isn't thrown at the moment. Test is here to prove it's possible to
			// throw any error code
			req := httptest.NewRequest("GET", "/wopi/files/abcdef", nil)

			w := httptest.NewRecorder()

			fc.On("CheckFileInfo", mock.Anything).Times(1).Return(connector.NewResponse(404), nil)

			httpAdapter.CheckFileInfo(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("Success", func() {
			req := httptest.NewRequest("GET", "/wopi/files/abcdef", nil)

			w := httptest.NewRecorder()

			// might need more info, but should be enough for the test
			finfo := &fileinfo.Microsoft{
				Size:              123456789,
				BreadcrumbDocName: "testy.docx",
			}
			fc.On("CheckFileInfo", mock.Anything).Times(1).Return(connector.NewResponseSuccessBody(finfo), nil)

			httpAdapter.CheckFileInfo(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(200))

			jsonInfo, _ := io.ReadAll(resp.Body)

			var responseInfo *fileinfo.Microsoft
			json.Unmarshal(jsonInfo, &responseInfo)
			Expect(responseInfo).To(Equal(finfo))
		})
	})

	Describe("GetFile", func() {
		It("General error", func() {
			req := httptest.NewRequest("GET", "/wopi/files/abcdef/contents", nil)

			w := httptest.NewRecorder()

			cc.On("GetFile", mock.Anything, mock.Anything).Times(1).Return(errors.New("Something happened"))

			httpAdapter.GetFile(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(500))
		})

		It("Not found", func() {
			// 404 isn't thrown at the moment. Test is here to prove it's possible to
			// throw any error code
			req := httptest.NewRequest("GET", "/wopi/files/abcdef/contents", nil)

			w := httptest.NewRecorder()

			cc.On("GetFile", mock.Anything, mock.Anything).Times(1).Return(connector.NewConnectorError(404, "Not found"))

			httpAdapter.GetFile(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("Success", func() {
			req := httptest.NewRequest("GET", "/wopi/files/abcdef/contents", nil)

			w := httptest.NewRecorder()

			expectedContent := []byte("This is a fake content for a test file")
			cc.On("GetFile", mock.Anything, mock.Anything).Times(1).Run(func(args mock.Arguments) {
				w := args.Get(1).(io.Writer)
				w.Write(expectedContent)
			}).Return(nil)

			httpAdapter.GetFile(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(200))

			content, _ := io.ReadAll(resp.Body)
			Expect(content).To(Equal(expectedContent))
		})
	})

	Describe("PutFile", func() {
		It("General error", func() {
			contentBody := "this is the new fake content"
			req := httptest.NewRequest("GET", "/wopi/files/abcdef/contents", strings.NewReader(contentBody))
			req.Header.Set(connector.HeaderWopiLock, "abc123")

			w := httptest.NewRecorder()

			cc.On("PutFile", mock.Anything, mock.Anything, int64(len(contentBody)), "abc123").Times(1).Return(nil, errors.New("Something happened"))

			httpAdapter.PutFile(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(500))

			content, _ := io.ReadAll(resp.Body)
			Expect(content).To(Equal([]byte("")))
		})

		It("Connector error", func() {
			contentBody := "this is the new fake content"
			req := httptest.NewRequest("GET", "/wopi/files/abcdef/contents", strings.NewReader(contentBody))
			req.Header.Set(connector.HeaderWopiLock, "abc123")

			w := httptest.NewRecorder()

			cc.On("PutFile", mock.Anything, mock.Anything, int64(len(contentBody)), "abc123").Times(1).Return(nil, connector.NewConnectorError(500, "Something happened"))

			httpAdapter.PutFile(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(500))

			content, _ := io.ReadAll(resp.Body)
			Expect(content).To(Equal([]byte("")))
		})

		It("Conflict", func() {
			contentBody := "this is the new fake content"
			req := httptest.NewRequest("GET", "/wopi/files/abcdef/contents", strings.NewReader(contentBody))
			req.Header.Set(connector.HeaderWopiLock, "abc123")

			w := httptest.NewRecorder()

			cc.On("PutFile", mock.Anything, mock.Anything, int64(len(contentBody)), "abc123").Times(1).Return(
				connector.NewResponseLockConflict("zzz111", "Lock Conflict"), nil)

			httpAdapter.PutFile(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(409))
			Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("zzz111"))
			Expect(resp.Header.Get(connector.HeaderWopiLockFailureReason)).To(Equal("Lock Conflict"))
		})

		It("Success", func() {
			contentBody := "this is the new fake content"
			req := httptest.NewRequest("GET", "/wopi/files/abcdef/contents", strings.NewReader(contentBody))
			req.Header.Set(connector.HeaderWopiLock, "abc123")

			w := httptest.NewRecorder()

			cc.On("PutFile", mock.Anything, mock.Anything, int64(len(contentBody)), "abc123").Times(1).Return(
				connector.NewResponseWithVersionAndLock(
					200,
					&typesv1beta1.Timestamp{Seconds: uint64(1234), Nanos: uint32(567)},
					"abc123",
				), nil)

			httpAdapter.PutFile(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(200))
			Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("abc123"))
			Expect(resp.Header.Get(connector.HeaderWopiVersion)).To(Equal("v1234567"))
		})
	})

	Describe("GetAvatar", func() {
		It("Missing userID returns 400", func() {
			// No chi route context means chi.URLParam returns ""
			req := httptest.NewRequest("GET", "/wopi/avatars/", nil)
			w := httptest.NewRecorder()

			httpAdapter.GetAvatar(w, req)
			Expect(w.Result().StatusCode).To(Equal(400))
		})

		It("ConnectorError propagates the status code", func() {
			req := httptest.NewRequest("GET", "/wopi/avatars/user-123", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", "user-123")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			fc.On("GetAvatar", mock.Anything, "user-123").Times(1).
				Return(nil, connector.NewConnectorError(502, "Bad Gateway"))

			httpAdapter.GetAvatar(w, req)
			Expect(w.Result().StatusCode).To(Equal(502))
		})

		It("General error returns 500", func() {
			req := httptest.NewRequest("GET", "/wopi/avatars/user-123", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", "user-123")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			fc.On("GetAvatar", mock.Anything, "user-123").Times(1).
				Return(nil, errors.New("unexpected failure"))

			httpAdapter.GetAvatar(w, req)
			Expect(w.Result().StatusCode).To(Equal(500))
		})

		It("Success writes Content-Type, Cache-Control and body", func() {
			req := httptest.NewRequest("GET", "/wopi/avatars/user-123", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", "user-123")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			avatarData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // fake JPEG bytes
			fc.On("GetAvatar", mock.Anything, "user-123").Times(1).Return(
				&connector.ConnectorResponse{
					Status: 200,
					Headers: map[string]string{
						"Content-Type":  "image/jpeg",
						"Cache-Control": "public, max-age=300",
					},
					Body: avatarData,
				}, nil)

			httpAdapter.GetAvatar(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(200))
			Expect(resp.Header.Get("Content-Type")).To(Equal("image/jpeg"))
			Expect(resp.Header.Get("Cache-Control")).To(Equal("public, max-age=300"))
			body, _ := io.ReadAll(resp.Body)
			Expect(body).To(Equal(avatarData))
		})
	})

	Describe("PutRelativeFile", func() {
		It("Connector error", func() {
			contentBody := "this is the new fake content"
			req := httptest.NewRequest("GET", "/wopi/files/abcdef/contents", strings.NewReader(contentBody))
			req.Header.Set(connector.HeaderWopiLock, "abc123")
			req.Header.Set(connector.HeaderWopiRT, "relativetarget.docx")

			w := httptest.NewRecorder()

			fc.On("PutRelativeFileRelative", mock.Anything, mock.Anything, mock.Anything, int64(len(contentBody)), "relativetarget.docx").Times(1).Return(nil, connector.NewConnectorError(500, "Something happened"))

			httpAdapter.PutRelativeFile(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(500))

			content, _ := io.ReadAll(resp.Body)
			Expect(content).To(Equal([]byte("")))
		})

		It("Conflict", func() {
			contentBody := "this is the new fake content"
			req := httptest.NewRequest("GET", "/wopi/files/abcdef/contents", strings.NewReader(contentBody))
			req.Header.Set(connector.HeaderWopiLock, "abc123")
			req.Header.Set(connector.HeaderWopiRT, "relativetarget.docx")

			w := httptest.NewRecorder()

			fc.On("PutRelativeFileRelative", mock.Anything, mock.Anything, mock.Anything, int64(len(contentBody)), "relativetarget.docx").Times(1).Return(
				connector.NewResponseLockConflict("zzz111", "Lock Conflict"), nil)

			httpAdapter.PutRelativeFile(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(409))
			Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("zzz111"))
			Expect(resp.Header.Get(connector.HeaderWopiLockFailureReason)).To(Equal("Lock Conflict"))
		})

		It("Success", func() {
			contentBody := "this is the new fake content"
			req := httptest.NewRequest("GET", "/wopi/files/abcdef/contents", strings.NewReader(contentBody))
			req.Header.Set(connector.HeaderWopiLock, "abc123")
			req.Header.Set(connector.HeaderWopiRT, "relativetarget.docx")

			w := httptest.NewRecorder()

			fc.On("PutRelativeFileRelative", mock.Anything, mock.Anything, mock.Anything, int64(len(contentBody)), "relativetarget.docx").Times(1).Return(
				connector.NewResponseWithVersionAndLock(
					200,
					&typesv1beta1.Timestamp{Seconds: uint64(1234), Nanos: uint32(567)},
					"abc123",
				), nil)

			httpAdapter.PutRelativeFile(w, req)
			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(200))
			Expect(resp.Header.Get(connector.HeaderWopiLock)).To(Equal("abc123"))
			Expect(resp.Header.Get(connector.HeaderWopiVersion)).To(Equal("v1234567"))
		})
	})
})
