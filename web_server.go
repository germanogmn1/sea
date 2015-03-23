package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

func WebServer(addr string) {
	InitTemplates()
	router := httprouter.New()
	router.GET("/", indexHandler)
	router.GET("/updates", updatesHandler)
	router.GET("/build/:rev", showHandler)
	router.POST("/build/:rev/cancel", cancelHandler)
	router.GET("/build/:rev/stream", streamHandler)
	log.Printf("Starting web server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, &HTTPWrapper{router}))
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	RenderHtml(w, "index", buildList)
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
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	for chunk := range build.Output.ReadChunks() {
		w.Write(chunk)
		w.(http.Flusher).Flush()
	}
}

func cancelHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	build := FindBuild(ps.ByName("rev"))
	if build == nil {
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
