package main

import (
	"flag"
	"github.com/julienschmidt/httprouter"
	git "gopkg.in/libgit2/git2go.v22"
	"io"
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
	oid, err := git.NewOid(rev)
	tree, err := repo.LookupTree(oid)
	repo.CheckoutTree(tree, &git.CheckoutOpts{
		// Strategy: git.CheckoutForce, ???
		TargetDirectory: "/tmp/proj/rev",
	})
	// TODO: delete dir somewere

	append(buildList, Build{
		Rev:        rev,
		State:      BUILD_WAITING,
		ScriptPath: "/tmp/proj/rev/Seafile",
		Output:     NewEmptyOutputBuffer(),
	})
}

func listenGitHooks(pipePath string) {
	err := syscall.Mknod(pipePath, syscall.S_IFIFO|0666, 0)
	// TODO: review the pipe cleanup, it's not being removed for some reason
	defer func() {
		err := os.Remove(pipePath)
		if err != nil {
			panic(err)
		}
	}()
	if err != nil {
		panic(err)
	}
	// TODO: comment why this have to be read&write mode
	file, err := os.OpenFile(pipePath, os.O_RDWR, 0666)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	readBuff := make([]byte, 0xFF)
	for {
		n, err := file.Read(readBuff)
		if err != nil {
			if err == io.EOF {
				log.Printf("EOF: %v", err)
			} else {
				panic(err)
			}
		}
		if n > 0 {
			log.Printf("Read: %q", readBuff[:n])
		}
	}
}

func main() {
	var pipePath string
	flag.StringVar(&pipePath, "pipe", "./post_receive.pipe",
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
