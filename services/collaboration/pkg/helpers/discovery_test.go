package helpers_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/helpers"
)

var _ = Describe("AppURLs", func() {
	var appURLs *helpers.AppURLs

	BeforeEach(func() {
		appURLs = helpers.NewAppURLs()
	})

	Describe("NewAppURLs", func() {
		It("should create a new AppURLs instance with empty map", func() {
			Expect(appURLs).NotTo(BeNil())
			Expect(appURLs.GetMimeTypes()).To(BeEmpty())
			Expect(appURLs.GetAppURLFor("view", ".pdf")).To(BeEmpty())
		})
	})

	Describe("Store and GetAppURLFor", func() {
		It("should store and retrieve app URLs correctly", func() {
			testURLs := map[string]map[string]string{
				"view": {
					".pdf":  "https://example.com/view/pdf",
					".docx": "https://example.com/view/docx",
					".xlsx": "https://example.com/view/xlsx",
				},
				"edit": {
					".docx": "https://example.com/edit/docx",
					".xlsx": "https://example.com/edit/xlsx",
				},
			}

			appURLs.Store(testURLs)

			// Test successful lookups
			Expect(appURLs.GetAppURLFor("view", ".pdf")).To(Equal("https://example.com/view/pdf"))
			Expect(appURLs.GetAppURLFor("view", ".docx")).To(Equal("https://example.com/view/docx"))
			Expect(appURLs.GetAppURLFor("edit", ".docx")).To(Equal("https://example.com/edit/docx"))
		})

		It("should return empty string for non-existent action", func() {
			testURLs := map[string]map[string]string{
				"view": {".pdf": "https://example.com/view/pdf"},
			}

			appURLs.Store(testURLs)

			Expect(appURLs.GetAppURLFor("nonexistent", ".pdf")).To(BeEmpty())
		})

		It("should return empty string for non-existent extension", func() {
			testURLs := map[string]map[string]string{
				"view": {".pdf": "https://example.com/view/pdf"},
			}

			appURLs.Store(testURLs)

			Expect(appURLs.GetAppURLFor("view", ".nonexistent")).To(BeEmpty())
		})

		It("should handle empty maps gracefully", func() {
			emptyURLs := map[string]map[string]string{}
			appURLs.Store(emptyURLs)

			Expect(appURLs.GetAppURLFor("view", ".pdf")).To(BeEmpty())
		})

		It("should handle nil action maps gracefully", func() {
			testURLs := map[string]map[string]string{
				"view": nil,
			}

			appURLs.Store(testURLs)

			Expect(appURLs.GetAppURLFor("view", ".pdf")).To(BeEmpty())
		})
	})

	Describe("GetMimeTypes", func() {
		It("should return empty slice for empty AppURLs", func() {
			mimeTypes := appURLs.GetMimeTypes()
			Expect(mimeTypes).To(BeEmpty())
		})

		It("should return correct mime types for known extensions", func() {
			testURLs := map[string]map[string]string{
				"view": {
					".pdf":  "https://example.com/view/pdf",
					".docx": "https://example.com/view/docx",
					".xlsx": "https://example.com/view/xlsx",
					".pptx": "https://example.com/view/pptx",
				},
				"edit": {
					".txt":  "https://example.com/edit/txt",
					".html": "https://example.com/edit/html",
				},
			}

			appURLs.Store(testURLs)

			mimeTypes := appURLs.GetMimeTypes()

			// Should contain expected mime types (order doesn't matter)
			Expect(mimeTypes).To(ContainElements(
				"application/pdf",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"application/vnd.openxmlformats-officedocument.presentationml.presentation",
				"text/plain",
				"text/html",
			))

			// Should not contain application/octet-stream (filtered out)
			Expect(mimeTypes).NotTo(ContainElement("application/octet-stream"))
		})

		It("should deduplicate mime types across actions", func() {
			testURLs := map[string]map[string]string{
				"view": {
					".pdf": "https://example.com/view/pdf",
					".txt": "https://example.com/view/txt",
				},
				"edit": {
					".pdf": "https://example.com/edit/pdf", // Same extension as view
					".txt": "https://example.com/edit/txt", // Same extension as view
				},
			}

			appURLs.Store(testURLs)

			mimeTypes := appURLs.GetMimeTypes()

			// Should only have unique mime types
			Expect(mimeTypes).To(ContainElements("application/pdf", "text/plain"))
			Expect(len(mimeTypes)).To(Equal(2)) // No duplicates
		})

		It("should filter out application/octet-stream mime types", func() {
			testURLs := map[string]map[string]string{
				"view": {
					".pdf":      "https://example.com/view/pdf",
					".unknown":  "https://example.com/view/unknown", // This might return application/octet-stream
					".fake-ext": "https://example.com/view/fake",    // This might return application/octet-stream
				},
			}

			appURLs.Store(testURLs)

			mimeTypes := appURLs.GetMimeTypes()

			// Should contain PDF but not octet-stream
			Expect(mimeTypes).To(ContainElement("application/pdf"))
			Expect(mimeTypes).NotTo(ContainElement("application/octet-stream"))
		})

		It("should handle empty extension maps", func() {
			testURLs := map[string]map[string]string{
				"view": {},
				"edit": {},
			}

			appURLs.Store(testURLs)

			mimeTypes := appURLs.GetMimeTypes()
			Expect(mimeTypes).To(BeEmpty())
		})

		It("should handle nil extension maps", func() {
			testURLs := map[string]map[string]string{
				"view": nil,
				"edit": nil,
			}

			appURLs.Store(testURLs)

			mimeTypes := appURLs.GetMimeTypes()
			Expect(mimeTypes).To(BeEmpty())
		})
	})

	Describe("Concurrent Access", func() {
		It("should handle concurrent reads and writes safely", func() {
			// This is a basic smoke test for concurrent access
			// In practice, you'd want more sophisticated race testing

			initialURLs := map[string]map[string]string{
				"view": {".pdf": "https://example.com/view/pdf"},
			}
			appURLs.Store(initialURLs)

			done := make(chan bool, 10)

			// Start multiple readers
			for i := 0; i < 5; i++ {
				go func() {
					defer GinkgoRecover()
					for j := 0; j < 100; j++ {
						_ = appURLs.GetAppURLFor("view", ".pdf")
						_ = appURLs.GetMimeTypes()
					}
					done <- true
				}()
			}

			// Start multiple writers
			for i := 0; i < 5; i++ {
				go func(id int) {
					defer GinkgoRecover()
					for j := 0; j < 100; j++ {
						newURLs := map[string]map[string]string{
							"view": {".pdf": "https://example.com/updated/pdf"},
						}
						appURLs.Store(newURLs)
					}
					done <- true
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < 10; i++ {
				<-done
			}

			// Should still be functional after concurrent access
			Expect(appURLs.GetAppURLFor("view", ".pdf")).NotTo(BeEmpty())
		})
	})

	Describe("Real-world scenarios", func() {
		It("should handle realistic WOPI discovery data", func() {
			// Based on the test data from the discovery tests
			realisticURLs := map[string]map[string]string{
				"view": {
					".pdf":  "https://cloud.opencloud.test/hosting/wopi/word/view",
					".djvu": "https://cloud.opencloud.test/hosting/wopi/word/view",
					".docx": "https://cloud.opencloud.test/hosting/wopi/word/view",
					".xls":  "https://cloud.opencloud.test/hosting/wopi/cell/view",
					".xlsb": "https://cloud.opencloud.test/hosting/wopi/cell/view",
				},
				"edit": {
					".docx": "https://cloud.opencloud.test/hosting/wopi/word/edit",
				},
			}

			appURLs.Store(realisticURLs)

			// Test specific lookups
			Expect(appURLs.GetAppURLFor("view", ".pdf")).To(Equal("https://cloud.opencloud.test/hosting/wopi/word/view"))
			Expect(appURLs.GetAppURLFor("edit", ".docx")).To(Equal("https://cloud.opencloud.test/hosting/wopi/word/edit"))
			Expect(appURLs.GetAppURLFor("edit", ".pdf")).To(BeEmpty()) // No edit for PDF

			// Test mime types
			mimeTypes := appURLs.GetMimeTypes()
			Expect(mimeTypes).To(ContainElements(
				"application/pdf",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				"application/vnd.ms-excel",
			))
		})
	})
})

var _ = Describe("Discovery", func() {
	var (
		discoveryContent1 string
		srv               *httptest.Server
	)

	BeforeEach(func() {
		discoveryContent1 = `
<?xml version="1.0" encoding="utf-8"?>
<wopi-discovery>
  <net-zone name="external-http">
    <app name="Word" favIconUrl="https://cloud.opencloud.test/web-apps/apps/documenteditor/main/resources/img/favicon.ico">
      <action name="view" ext="pdf" urlsrc="https://cloud.opencloud.test/hosting/wopi/word/view?&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="embedview" ext="pdf" urlsrc="https://cloud.opencloud.test/hosting/wopi/word/view?embed=1&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="view" ext="djvu" urlsrc="https://cloud.opencloud.test/hosting/wopi/word/view?&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="embedview" ext="djvu" urlsrc="https://cloud.opencloud.test/hosting/wopi/word/view?embed=1&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="view" ext="docx" urlsrc="https://cloud.opencloud.test/hosting/wopi/word/view?&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="embedview" ext="docx" urlsrc="https://cloud.opencloud.test/hosting/wopi/word/view?embed=1&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="editnew" ext="docx" requires="locks,update" urlsrc="https://cloud.opencloud.test/hosting/wopi/word/edit?&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="edit" ext="docx" default="true" requires="locks,update" urlsrc="https://cloud.opencloud.test/hosting/wopi/word/edit?&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
    </app>
    <app name="Excel" favIconUrl="https://cloud.opencloud.test/web-apps/apps/spreadsheeteditor/main/resources/img/favicon.ico">
      <action name="view" ext="xls" urlsrc="https://cloud.opencloud.test/hosting/wopi/cell/view?&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="embedview" ext="xls" urlsrc="https://cloud.opencloud.test/hosting/wopi/cell/view?embed=1&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="convert" ext="xls" targetext="xlsx" requires="update" urlsrc="https://cloud.opencloud.test/hosting/wopi/convert-and-edit/xls/xlsx?&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="view" ext="xlsb" urlsrc="https://cloud.opencloud.test/hosting/wopi/cell/view?&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="embedview" ext="xlsb" urlsrc="https://cloud.opencloud.test/hosting/wopi/cell/view?embed=1&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
      <action name="convert" ext="xlsb" targetext="xlsx" requires="update" urlsrc="https://cloud.opencloud.test/hosting/wopi/convert-and-edit/xlsb/xlsx?&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
    </app>
    <app name="application/vnd.oasis.opendocument.presentation">
      <action name="edit" ext="" default="true" requires="locks,update" urlsrc="https://cloud.opencloud.test/hosting/wopi/slide/edit?&amp;&lt;rs=DC_LLCC&amp;&gt;&lt;dchat=DISABLE_CHAT&amp;&gt;&lt;embed=EMBEDDED&amp;&gt;&lt;fs=FULLSCREEN&amp;&gt;&lt;hid=HOST_SESSION_ID&amp;&gt;&lt;rec=RECORDING&amp;&gt;&lt;sc=SESSION_CONTEXT&amp;&gt;&lt;thm=THEME_ID&amp;&gt;&lt;ui=UI_LLCC&amp;&gt;&lt;wopisrc=WOPI_SOURCE&amp;&gt;&amp;"/>
    </app>
  </net-zone>
  <proof-key oldvalue="BgIAAACkAABSU0ExAAgAAAEAAQD/NVqekFNi8X3p6Bvdlaxm0GGuggW5kKfVEQzPGuOkGVrz6DrOMNR+k7Pq8tONY+1NHgS6Z+v3959em78qclVDuQX77Tkml0xMHAQHN4sAHF9iQJS8gOBUKSVKaHD7Z8YXch6F212YSUSc8QphpDSHWVShU7rcUeLQsd/0pkflh5+um4YKEZhm4Mou3vstp5p12NeffyK1WFZF7q4jB7jclAslYKQsP82YY3DcRwu5Tl/+W0ifVcXze0mI7v1reJ12pKn8ifRiq+0q5oJST3TRSrvmjLg9Gt3ozhVIt2HUi3La7Qh40YOAUXm0g/hUq2BepeOp1C7WSvaOFHXe6Hqq" oldmodulus="qnro3nUUjvZK1i7UqeOlXmCrVPiDtHlRgIPReAjt2nKL1GG3SBXO6N0aPbiM5rtK0XRPUoLmKu2rYvSJ/Kmkdp14a/3uiEl788VVn0hb/l9OuQtH3HBjmM0/LKRgJQuU3LgHI67uRVZYtSJ/n9fYdZqnLfveLsrgZpgRCoabrp+H5Uem9N+x0OJR3LpToVRZhzSkYQrxnERJmF3bhR5yF8Zn+3BoSiUpVOCAvJRAYl8cAIs3BwQcTEyXJjnt+wW5Q1VyKr+bXp/39+tnugQeTe1jjdPy6rOTftQwzjro81oZpOMazwwR1aeQuQWCrmHQZqyV3Rvo6X3xYlOQnlo1/w==" oldexponent="AQAB" value="BgIAAACkAABSU0ExAAgAAAEAAQD/NVqekFNi8X3p6Bvdlaxm0GGuggW5kKfVEQzPGuOkGVrz6DrOMNR+k7Pq8tONY+1NHgS6Z+v3959em78qclVDuQX77Tkml0xMHAQHN4sAHF9iQJS8gOBUKSVKaHD7Z8YXch6F212YSUSc8QphpDSHWVShU7rcUeLQsd/0pkflh5+um4YKEZhm4Mou3vstp5p12NeffyK1WFZF7q4jB7jclAslYKQsP82YY3DcRwu5Tl/+W0ifVcXze0mI7v1reJ12pKn8ifRiq+0q5oJST3TRSrvmjLg9Gt3ozhVIt2HUi3La7Qh40YOAUXm0g/hUq2BepeOp1C7WSvaOFHXe6Hqq" modulus="qnro3nUUjvZK1i7UqeOlXmCrVPiDtHlRgIPReAjt2nKL1GG3SBXO6N0aPbiM5rtK0XRPUoLmKu2rYvSJ/Kmkdp14a/3uiEl788VVn0hb/l9OuQtH3HBjmM0/LKRgJQuU3LgHI67uRVZYtSJ/n9fYdZqnLfveLsrgZpgRCoabrp+H5Uem9N+x0OJR3LpToVRZhzSkYQrxnERJmF3bhR5yF8Zn+3BoSiUpVOCAvJRAYl8cAIs3BwQcTEyXJjnt+wW5Q1VyKr+bXp/39+tnugQeTe1jjdPy6rOTftQwzjro81oZpOMazwwR1aeQuQWCrmHQZqyV3Rvo6X3xYlOQnlo1/w==" exponent="AQAB"/>
</wopi-discovery>
`
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/bad/hosting/discovery":
				w.WriteHeader(500)
			case "/good/hosting/discovery":
				w.Write([]byte(discoveryContent1))
			case "/wrongformat/hosting/discovery":
				w.Write([]byte("Text that <can't> be XML /form<atted/"))
			}
		}))
	})

	AfterEach(func() {
		srv.Close()
	})

	Describe("GetAppURLs", func() {
		It("Good discovery URL", func() {
			cfg := &config.Config{
				App: config.App{
					Addr:     srv.URL + "/good",
					Insecure: true,
				},
			}
			logger := log.NopLogger()

			appUrls, err := helpers.GetAppURLs(cfg, logger)

			expectedAppUrls := map[string]map[string]string{
				"view": map[string]string{
					".pdf":  "https://cloud.opencloud.test/hosting/wopi/word/view",
					".djvu": "https://cloud.opencloud.test/hosting/wopi/word/view",
					".docx": "https://cloud.opencloud.test/hosting/wopi/word/view",
					".xls":  "https://cloud.opencloud.test/hosting/wopi/cell/view",
					".xlsb": "https://cloud.opencloud.test/hosting/wopi/cell/view",
				},
				"edit": map[string]string{
					".docx": "https://cloud.opencloud.test/hosting/wopi/word/edit",
				},
			}

			Expect(err).To(Succeed())
			Expect(appUrls).To(Equal(expectedAppUrls))
		})

		It("Wrong discovery URL", func() {
			cfg := &config.Config{
				App: config.App{
					Addr:     srv.URL + "/bad",
					Insecure: true,
				},
			}
			logger := log.NopLogger()

			appUrls, err := helpers.GetAppURLs(cfg, logger)
			Expect(err).To(HaveOccurred())
			Expect(appUrls).To(BeNil())
		})

		It("Not XML formatted", func() {
			cfg := &config.Config{
				App: config.App{
					Addr:     srv.URL + "/wrongformat",
					Insecure: true,
				},
			}
			logger := log.NopLogger()

			appUrls, err := helpers.GetAppURLs(cfg, logger)
			Expect(err).To(HaveOccurred())
			Expect(appUrls).To(BeNil())
		})
	})
})
