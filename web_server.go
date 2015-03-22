package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func WebServer(addr string) {
	InitTemplates()
	router := httprouter.New()
	router.GET("/", indexHandler)
	router.GET("/build/:rev", showHandler)
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

// TODO: why this error? http: multiple response.WriteHeader calls
func streamHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	build := FindBuild(ps.ByName("rev"))
	if build == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	for chunk := range build.Output.ReadChunks() {
		w.Write(chunk)
		w.(http.Flusher).Flush()
	}
}
