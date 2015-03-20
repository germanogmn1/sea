package main

import (
	"flag"
	"github.com/julienschmidt/httprouter"
	git "gopkg.in/libgit2/git2go.v22"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"
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

func prepareBuild(path, rev string) {
	repo, err := git.OpenRepository(path)
	if err != nil {
		panic(err)
	}
	oid, err := git.NewOid(rev)
	if err != nil {
		panic(err)
	}
	tree, err := repo.LookupTree(oid)
	if err != nil {
		panic(err)
	}
	repo.CheckoutTree(tree, &git.CheckoutOpts{
		// Strategy: git.CheckoutForce, ???
		TargetDirectory: "/tmp/proj/rev",
	})
	// TODO: delete dir somewere

	buildList = append(buildList, Build{
		Rev:        rev,
		State:      BUILD_WAITING,
		ScriptPath: "/tmp/proj/rev/Seafile",
		Output:     NewEmptyOutputBuffer(),
	})
}

func listenGitHooks(pipePath string) {
	assert := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	var file *os.File = nil

	oldmask := syscall.Umask(0)
	err := syscall.Mkfifo(pipePath, 0622)
	syscall.Umask(oldmask)

	defer func() {
		var closeErr error
		if file != nil {
			closeErr = file.Close()
		}
		assert(closeErr)
		assert(os.Remove(pipePath))
	}()
	assert(err)
	// TODO: comment why this have to be read&write mode
	file, err = os.OpenFile(pipePath, os.O_RDWR, 0)
	assert(err)
	readBuff := make([]byte, 0xFF)
	for {
		n, err := file.Read(readBuff)
		assert(err)
		log.Printf("hook: %q", ShellSplit(string(readBuff[:n])))
	}
}

// TODO: handle SIGINT nicely
func main() {
	var pipePath string
	flag.StringVar(&pipePath, "pipe", "/tmp/seapipe",
		"Path to the named pipe where the local build requests will come.")

	done := make(chan struct{})
	go func() {
		listenGitHooks(pipePath)
		done <- struct{}{}
	}()
	<-done
	return
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
