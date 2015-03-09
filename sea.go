package main

import (
	"flag"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const (
	BUILD_WAITING = iota
	BUILD_RUNNING
	BUILD_SUCCESS
	BUILD_FAILURE
)

var stateNames = []string{
	"Wating",
	"Running",
	"Success",
	"Failure",
}

type Build struct {
	Rev        string
	State      int
	ScriptPath string
	Output     OutputBuffer
}

func execFile(path string, output io.Writer) {
	cmd := exec.Command(path)
	cmd.Stdout = output
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Wait()
}

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
	var port int
	flag.IntVar(&port, "port", 8080, "HTTP port to listen")
	flag.Parse()

	addr := fmt.Sprintf(":%d", port)

	router := httprouter.New()
	router.GET("/", RequestLogger(IndexHandler))
	router.GET("/build/:rev", RequestLogger(ShowHandler))
	router.GET("/build/:rev/stream", RequestLogger(StreamHandler))
	router.POST("/exec", RequestLogger(ExecHandler))

	log.Printf("Starting web server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

type loggingWrapper struct {
	w      http.ResponseWriter
	status int
}

func (l *loggingWrapper) Header() http.Header         { return l.w.Header() }
func (l *loggingWrapper) Write(b []byte) (int, error) { return l.w.Write(b) }
func (l *loggingWrapper) WriteHeader(status int) {
	l.w.WriteHeader(status)
	l.status = status
}

func RequestLogger(handler httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		start := time.Now()
		logger := loggingWrapper{w, http.StatusOK}
		log.Printf("Started %s %q", r.Method, r.RequestURI)
		handler(&logger, r, ps)
		duration := time.Since(start)
		var milliseconds float64 = float64(duration.Nanoseconds()) / 10e5
		log.Printf("Completed %d %s in %.2fms",
			logger.status, http.StatusText(logger.status), milliseconds)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	RenderHtml(w, "index", buildList)
	// for i := range  {
	// 	build := &buildList[i]
	// 	fmt.Fprintf(w, "%s: %s (%s)\n", build.Rev[:11], stateNames[build.State], build.ScriptPath)
	// }
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

func ExecHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// go execFile("./Seafile", &cmdOut)
	// fmt.Fprintf(w, "started")
}
