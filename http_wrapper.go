package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"runtime"
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
	var milliseconds float64 = float64(duration.Nanoseconds()) / 10e6
	log.Printf("Completed %d %s in %.2fms", bw.status,
		http.StatusText(bw.status), milliseconds)
}

func (h *HTTPWrapper) handlePanic(bw *bufferedWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			if bw.flushed {
				// TODO: response already sent to client, now what?
			} else {
				bw.buffer.Reset()
				bw.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(bw, "%v\n\n", err)
				var buf []byte
				runtime.Stack(buf, false)
				bw.Write(buf)
			}
		}
		bw.commit()
	}()
	h.Handler.ServeHTTP(bw, r)
}
