package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

func WebServer(addr string) <-chan error {
	errors := make(chan error, 1)
	go func() {
		defer close(errors)
		if err := InitTemplates(); err != nil {
			errors <- err
			return
		}
		router := httprouter.New()
		router.GET("/", indexHandler)
		router.GET("/updates", updatesHandler)
		router.GET("/build/:rev", showHandler)
		router.POST("/build/:rev/cancel", cancelHandler)
		router.GET("/build/:rev/stream", streamHandler)
		log.Printf("Starting web server on %v", addr)

		errors <- http.ListenAndServe(addr, &HTTPWrapper{router})
	}()
	return errors
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	RenderHtml(w, "index", AllBuilds())
}

func showHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	build := FindBuild(ps.ByName("rev"))
	if build == nil {
		http.NotFound(w, r)
		return
	}
	RenderHtml(w, "show", build)
}

func streamHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	build := FindBuild(ps.ByName("rev"))
	if build == nil {
		http.NotFound(w, r)
		return
	}

	if build.State != BuildRunning {
		w.Write(build.Output)
		return
	}

	running, ok := RunningBuilds.Get(build.Rev)
	if !ok {
		panic("build state is BuildRunning but no running build was found")
	}
	stream := running.Buffer.Stream()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	closed := w.(http.CloseNotifier).CloseNotify()
	var buffer [512]byte
	for {
		select {
		case <-closed:
			return
		default:
			n, err := stream.Read(buffer[:])
			if err == io.EOF {
				return
			}
			w.Write(buffer[:n])
			w.(http.Flusher).Flush()
		}
	}
}

func cancelHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	build, ok := RunningBuilds.Get(ps.ByName("rev"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	build.Cancel()
}

func updatesHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	closed := w.(http.CloseNotifier).CloseNotify()

	for i := 0; ; i++ {
		select {
		case <-closed:
			return
		default:
			fmt.Fprintf(w, "event: %s\n", "inc")
			fmt.Fprintf(w, "data: %d\n\n", i)
			w.(http.Flusher).Flush()
			time.Sleep(1 * time.Second)
		}
	}
}
