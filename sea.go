package main

import (
	"flag"
	// "fmt"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	// "reflect"
	"strings"
)

var buildList = []Build{
	Build{
		Rev:        "134d74025fbbbbcac149f206d4157890e145e8c3",
		State:      BUILD_WAITING,
		ScriptPath: "./Seafile",
		Output:     NewEmptyOutputBuffer(),
	},
	Build{
		Rev:        "bbdc1e3744f128dfa744ab5bed520c0e5ab2e116",
		State:      BUILD_SUCCESS,
		ScriptPath: "./Seafile",
		Output:     NewFilledOutputBuffer([]byte("success")),
	},
	Build{
		Rev:        "c21e9b8ff5f55ceeacffeadfd6d5ca4fce8dc6a7",
		State:      BUILD_FAILURE,
		ScriptPath: "./Seafile",
		Output:     NewFilledOutputBuffer([]byte("fail")),
	},
}

func findBuild(rev string) *Build {
	for i := range buildList {
		if strings.HasPrefix(buildList[i].Rev, rev) {
			return &buildList[i]
		}
	}
	return nil
}

func main() {
	InitTemplates()

	// Run HTTP server
	var addr string
	flag.StringVar(&addr, "addr", ":8080", "TCP address to listen on")
	flag.Parse()

	router := httprouter.New()
	router.GET("/", IndexHandler)
	router.GET("/build/:rev", ShowHandler)
	router.GET("/build/:rev/stream", StreamHandler)
	router.POST("/build/:rev/exec", ExecHandler)

	log.Printf("Starting web server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, &HTTPWrapper{router}))
}

func IndexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// panic(reflect.TypeOf(w).String())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	RenderHtml(w, "index", buildList)
}

func ShowHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	build := findBuild(ps.ByName("rev"))
	if build == nil {
		http.NotFound(w, r)
		return
	}

	RenderHtml(w, "show", build)
}

func StreamHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	build := findBuild(ps.ByName("rev"))
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

func ExecHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	build := findBuild(ps.ByName("rev"))
	if build == nil {
		http.NotFound(w, r)
	} else if build.State == BUILD_WAITING {
		// TODO: this check is unsafe!
		go build.Exec()
	} else {
		http.Error(w, "Invalid build state: "+build.StateName(), http.StatusBadRequest)
	}
}
