package icapclient

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/egirna/icap"
)

var (
	stop = make(chan os.Signal, 1)
	port = 1344
)

const (
	previewBytes      = 24
	goodFileDetectStr = "GOOD FILE"
	badFileDetectStr  = "BAD FILE"
	goodURL           = "http://goodifle.com"
	badURL            = "http://badfile.com"
)

func startTestServer() {
	icap.HandleFunc("/respmod", respmodHandler)
	icap.HandleFunc("/reqmod", reqmodHandler)

	log.Println("Starting ICAP test server...")

	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	go func() {
		if err := icap.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			log.Println("Failed to start ICAP test server: ", err.Error())
			return
		}
	}()

	time.Sleep(5 * time.Millisecond)

	log.Printf("ICAP test server is running on localhost:%d\n...\n", port)
	<-stop

	log.Println("ICAP test server is shut down!")
}

func stopTestServer() {
	stop <- syscall.SIGKILL
}

func respmodHandler(w icap.ResponseWriter, req *icap.Request) {
	h := w.Header()
	h.Set("ISTag", "ICAP-TEST")
	h.Set("Service", "ICAP-TEST-SERVICE")

	switch req.Method {
	case "OPTIONS":
		h.Set("Methods", "RESPMOD")
		h.Set("Allow", "204")
		if previewBytes > 0 {
			h.Set("Preview", strconv.Itoa(previewBytes))
		}
		h.Set("Transfer-Preview", "*")
		w.WriteHeader(http.StatusOK, nil, false)
	case "RESPMOD":
		defer req.Response.Body.Close()

		if val, exist := req.Header["Allow"]; !exist || (len(val) > 0 && val[0] != "204") {
			w.WriteHeader(http.StatusNoContent, nil, false)
			return
		}

		buf := &bytes.Buffer{}

		if _, err := io.Copy(buf, req.Response.Body); err != nil {
			log.Println("Failed to copy the response body to buffer: ", err.Error())
			w.WriteHeader(http.StatusNoContent, nil, false)
			return
		}

		status := 0
		if strings.Contains(buf.String(), goodFileDetectStr) {
			status = http.StatusNoContent
		}

		if strings.Contains(buf.String(), badFileDetectStr) {
			status = http.StatusOK
		}

		w.WriteHeader(status, nil, false)
	}
}

func reqmodHandler(w icap.ResponseWriter, req *icap.Request) {
	h := w.Header()
	h.Set("ISTag", "ICAP-TEST")
	h.Set("Service", "ICAP-TEST-SERVICE")

	switch req.Method {
	case "OPTIONS":
		h.Set("Methods", "REQMOD")
		h.Set("Allow", "204")
		if previewBytes > 0 {
			h.Set("Preview", strconv.Itoa(previewBytes))
		}
		h.Set("Transfer-Preview", "*")
		w.WriteHeader(http.StatusOK, nil, false)
	case "REQMOD":

		if val, exist := req.Header["Allow"]; !exist || (len(val) > 0 && val[0] != "204") {
			w.WriteHeader(http.StatusNoContent, nil, false)
			return
		}

		fileURL := req.Request.RequestURI

		status := 0
		if fileURL == goodURL {
			status = http.StatusNoContent
		}

		if fileURL == badURL {
			status = http.StatusOK
		}

		w.WriteHeader(status, nil, false)
	}
}

func testServerRunning() bool {
	lstnr, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true
	}
	lstnr.Close()
	return false
}
