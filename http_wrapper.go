package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

type bufferedWriter struct {
	rw      http.ResponseWriter
	buffer  bytes.Buffer
	status  int
	flushed bool
}

func newBufferedWriter(w http.ResponseWriter) *bufferedWriter {
	return &bufferedWriter{
		rw:     w,
		status: http.StatusOK,
	}
}

// http.ResponseWriter
func (bw *bufferedWriter) Header() http.Header         { return bw.rw.Header() }
func (bw *bufferedWriter) Write(b []byte) (int, error) { return bw.buffer.Write(b) }
func (bw *bufferedWriter) WriteHeader(status int)      { bw.status = status }

// http.Flusher
func (bw *bufferedWriter) Flush() {
	bw.commit()
	bw.rw.(http.Flusher).Flush()
	bw.flushed = true
}

func (bw *bufferedWriter) commit() {
	bw.rw.WriteHeader(bw.status)
	bw.rw.Write(bw.buffer.Bytes())
	bw.buffer.Reset()
}

type HTTPWrapper struct {
	Handler http.Handler
}

func (h *HTTPWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bw := newBufferedWriter(w)
	start := time.Now()
	log.Printf("Started %s %q", r.Method, r.RequestURI)

	h.handlePanic(bw, r)

	duration := time.Since(start)
	log.Printf("Completed %d %s in %v", bw.status,
		http.StatusText(bw.status), duration)
}

func (h *HTTPWrapper) handlePanic(bw *bufferedWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			if bw.flushed {
				// TODO: response already sent to client, now what?
			} else {
				log.Printf("panic: %v", err)
				bw.buffer.Reset() // discard previous rendered data
				bw.WriteHeader(http.StatusInternalServerError)
				renderErrorPage(bw, err)
			}
		}
		bw.commit()
	}()
	h.Handler.ServeHTTP(bw, r)
}

const errorPageTmpl = `<!DOCTYPE html>
<html>
	<head>
		<style>
			body { margin: 0; }
			h3, pre { padding: 15px 25px; }
			h3 {
				margin: 0;
				font-family: Helvetica, sans-serif;
				background-color: #c22;
				color: #eee;
			}
		</style>
		<title>PANIC</title>
	</head>
	<body>
		<h3>%v</h3>
		<pre>%s</pre>
	</body>
</html>`

func renderErrorPage(w http.ResponseWriter, err interface{}) {
	stack := make([]byte, 4096)
	stack = stack[:runtime.Stack(stack, false)]
	prettyStack := strings.Replace(string(stack), os.Getenv("GOPATH"), "$GOPATH", -1)

	html := fmt.Sprintf(errorPageTmpl, err, prettyStack)
	w.Write([]byte(html))
}
