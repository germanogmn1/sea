package foo

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

type HTTPWrapper struct {
	Handler http.Handler
}

type writerWrapper struct {
	w       http.ResponseWriter
	status  int
	flushed bool
}

func (l *writerWrapper) Header() http.Header { fmt.Println("Header"); return l.w.Header() }
func (l *writerWrapper) Write(b []byte) (int, error) {
	fmt.Printf("Write %q\n", b)
	return l.w.Write(b)
}
func (l *writerWrapper) WriteHeader(status int) {
	l.w.WriteHeader(status)
	l.status = status
}
func (l *writerWrapper) Flush() {
	l.w.(http.Flusher).Flush()
	l.flushed = true
}

func (h *HTTPWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ww := writerWrapper{w, http.StatusOK}
	start := time.Now()
	log.Printf("Started %s %q", r.Method, r.RequestURI)

	h.handlePanic(&ww, r)

	duration := time.Since(start)
	var milliseconds float64 = float64(duration.Nanoseconds()) / 10e6
	log.Printf("Completed %d %s in %.2fms", ww.status,
		http.StatusText(ww.status), milliseconds)
}

func (h *HTTPWrapper) handlePanic(ww *writerWrapper, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			if ww.flushed {
				// TODO: response already sent to client, now what?
			} else {
				// TODO: get stack trace, render error page
			}
		}
	}()
	h.Handler(ww, r)
}
