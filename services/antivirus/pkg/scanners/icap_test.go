package scanners_test

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ic "github.com/opencloud-eu/icap-client"

	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/scanners"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/scanners/mocks"
)

func TestICAP_Scan(t *testing.T) {
	var (
		earlyExitErr = errors.New("stop here")
		testUrl      = "icap://test"
		client       = mocks.NewScanner(t)
		scanner      = &scanners.ICAP{Client: client, URL: testUrl}
	)

	t.Run("it sends a OPTIONS request to determine details", func(t *testing.T) {
		client.EXPECT().Do(mock.Anything).RunAndReturn(func(request ic.Request) (ic.Response, error) {
			assert.Equal(t, ic.MethodOPTIONS, request.Method)
			assert.Equal(t, testUrl, request.URL.String())
			return ic.Response{}, earlyExitErr
		}).Once()

		_, err := scanner.Scan(scanners.Input{})
		assert.ErrorIs(t, earlyExitErr, err) // we can exit early, just in case check the error to be identical to the early exit error
	})

	t.Run("it sends a REQMOD request with all the details", func(t *testing.T) {

		t.Run("request with ContentLength", func(t *testing.T) {
			t.Run("with size", func(t *testing.T) {
				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()

				client.EXPECT().Do(mock.Anything).RunAndReturn(func(request ic.Request) (ic.Response, error) {
					assert.Equal(t, ic.MethodREQMOD, request.Method)
					assert.Equal(t, testUrl, request.URL.String())
					assert.EqualValues(t, 999, request.HTTPRequest.ContentLength)
					return ic.Response{}, earlyExitErr
				}).Once()

				_, err := scanner.Scan(scanners.Input{Size: 999})
				assert.ErrorIs(t, earlyExitErr, err)
			})

			t.Run("without size", func(t *testing.T) {
				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()

				client.EXPECT().Do(mock.Anything).RunAndReturn(func(request ic.Request) (ic.Response, error) {
					assert.Equal(t, ic.MethodREQMOD, request.Method)
					assert.Equal(t, testUrl, request.URL.String())
					assert.EqualValues(t, 0, request.HTTPRequest.ContentLength)
					return ic.Response{}, earlyExitErr
				}).Once()

				_, err := scanner.Scan(scanners.Input{})
				assert.ErrorIs(t, earlyExitErr, err)
			})
		})

		t.Run("request with Content-Type header", func(t *testing.T) {
			t.Run("name contains known extension", func(t *testing.T) {
				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()

				client.EXPECT().Do(mock.Anything).RunAndReturn(func(request ic.Request) (ic.Response, error) {
					assert.Equal(t, "application/pdf", request.HTTPRequest.Header.Get("Content-Type"))
					return ic.Response{}, earlyExitErr
				}).Once()

				_, err := scanner.Scan(scanners.Input{Name: "report.pdf"})
				assert.ErrorIs(t, earlyExitErr, err)
			})

			t.Run("name with unknown extension", func(t *testing.T) {
				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()

				client.EXPECT().Do(mock.Anything).RunAndReturn(func(request ic.Request) (ic.Response, error) {
					assert.Equal(t, "application/octet-stream", request.HTTPRequest.Header.Get("Content-Type"))
					return ic.Response{}, earlyExitErr
				}).Once()

				_, err := scanner.Scan(scanners.Input{Name: "report.unknown"})
				assert.ErrorIs(t, earlyExitErr, err)
			})

			t.Run("name without extension", func(t *testing.T) {
				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()

				client.EXPECT().Do(mock.Anything).RunAndReturn(func(request ic.Request) (ic.Response, error) {
					assert.Equal(t, "httpd/unix-directory", request.HTTPRequest.Header.Get("Content-Type"))
					return ic.Response{}, earlyExitErr
				}).Once()

				_, err := scanner.Scan(scanners.Input{Name: "report"})
				assert.ErrorIs(t, earlyExitErr, err)
			})
		})

		t.Run("request with the OPTIONS response preview size ", func(t *testing.T) {
			t.Run("with PreviewBytes set", func(t *testing.T) {
				client.EXPECT().Do(mock.Anything).Return(ic.Response{PreviewBytes: 444}, nil).Once()

				client.EXPECT().Do(mock.Anything).RunAndReturn(func(request ic.Request) (ic.Response, error) {
					assert.Equal(t, 444, request.PreviewBytes)
					return ic.Response{}, earlyExitErr
				}).Once()

				_, err := scanner.Scan(scanners.Input{Body: bytes.NewReader(make([]byte, 888))})
				assert.ErrorIs(t, earlyExitErr, err)
			})

			t.Run("without PreviewBytes set", func(t *testing.T) {
				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()

				client.EXPECT().Do(mock.Anything).RunAndReturn(func(request ic.Request) (ic.Response, error) {
					assert.Equal(t, 0, request.PreviewBytes)
					return ic.Response{}, earlyExitErr
				}).Once()

				_, err := scanner.Scan(scanners.Input{Body: bytes.NewReader(make([]byte, 888))})
				assert.ErrorIs(t, earlyExitErr, err)
			})
		})
	})

	t.Run("request with the OPTIONS response preview size ", func(t *testing.T) {
		t.Run("with PreviewBytes set", func(t *testing.T) {
			client.EXPECT().Do(mock.Anything).Return(ic.Response{PreviewBytes: 444}, nil).Once()

			client.EXPECT().Do(mock.Anything).RunAndReturn(func(request ic.Request) (ic.Response, error) {
				assert.Equal(t, 444, request.PreviewBytes)
				return ic.Response{}, earlyExitErr
			}).Once()

			_, err := scanner.Scan(scanners.Input{Body: bytes.NewReader(make([]byte, 888))})
			assert.ErrorIs(t, earlyExitErr, err)
		})

		t.Run("it handles virus scan results", func(t *testing.T) {
			t.Run("no virus", func(t *testing.T) {
				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()
				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()

				result, err := scanner.Scan(scanners.Input{})
				assert.Nil(t, err)
				assert.False(t, result.Infected)
			})

			// clamav returns an X-Infection-Found header with the threat description
			t.Run("X-Infection-Found header ", func(t *testing.T) {
				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()
				client.EXPECT().Do(mock.Anything).Return(ic.Response{Header: http.Header{"X-Infection-Found": []string{"Threat=bad threat;"}}}, nil).Once()

				result, err := scanner.Scan(scanners.Input{})
				assert.Nil(t, err)
				assert.True(t, result.Infected)
				assert.Equal(t, "bad threat", result.Description)
			})

			// skyhigh returns the information via the content response
			t.Run("X-Infection-Found header", func(t *testing.T) {
				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()
				client.EXPECT().Do(mock.Anything).Return(ic.Response{ContentResponse: &http.Response{StatusCode: http.StatusForbidden, Status: "some status"}}, nil).Once()

				result, err := scanner.Scan(scanners.Input{})
				assert.Nil(t, err)
				assert.True(t, result.Infected)
				assert.Equal(t, "some status", result.Description)

				client.EXPECT().Do(mock.Anything).Return(ic.Response{}, nil).Once()
				client.EXPECT().Do(mock.Anything).Return(ic.Response{ContentResponse: &http.Response{StatusCode: http.StatusOK}}, nil).Once()

				result, err = scanner.Scan(scanners.Input{})
				assert.Nil(t, err)
				assert.False(t, result.Infected)
			})
		})
	})
}
