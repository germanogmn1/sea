package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sync"
	// "os"
	// "bytes"
	// "bufio"
)

type OutputBuffer struct {
	Content []byte
	Stream  chan []byte
	Mutex   sync.RWMutex
}

var cmdOut OutputBuffer
var executed bool

// io.Writer implementation
func (o *OutputBuffer) Write(p []byte) (int, error) {
	dup := make([]byte, len(p), len(p))
	copy(dup, p)
	select {
	case o.Stream <- dup:
	default:
	}
	o.Mutex.Lock()
	o.Content = append(o.Content, dup...)
	o.Mutex.Unlock()
	return len(p), nil
}

func (o *OutputBuffer) ReadChunks() chan []byte {
	response := make(chan []byte)
	go func() {
		o.Mutex.RLock()
		if len(o.Content) > 0 {
			response <- o.Content
		}
		o.Mutex.RUnlock()
		for chunk := range o.Stream {
			response <- chunk
		}
		close(response)
	}()
	return response
}

func execFile(path string, output *OutputBuffer) {
	cmd := exec.Command(path)
	cmd.Stdout = output
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Wait()
}

func stream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	for chunk := range cmdOut.ReadChunks() {
		w.Write(chunk)
		w.(http.Flusher).Flush()
	}
}

func main() {
	executed = false
	cmdOut = OutputBuffer{
		Stream:  make(chan []byte),
	}

	var port int
	flag.IntVar(&port, "port", 8080, "HTTP port to listen")
	flag.Parse()

	http.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		if executed {
			fmt.Fprintf(w, "already running")
		} else {
			go execFile("./Seafile", &cmdOut)
			fmt.Fprintf(w, "started")
			executed = true
		}
	})
	http.HandleFunc("/stream", stream)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

/*
const (
    BUILD_WAITING = iota
    BUILD_RUNNING
    BUILD_SUCCESS
    BUILD_FAILURE
)

type Build struct {
    Rev string
    State int
    TreePath string
    Output bytes.Buffer
}

var buildList []Build

func buildWorker(buildDef Build) {
    seaPath, err := exec.LookPath(buildDef.TreePath + "/Seafile")
    if err != nil {
        log.Fatal(err)
    }
    cmd := exec.Command(seaPath)
    err = cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
}

func main() {
    var buildRequests := make(chan string)
    for request := range buildRequests {
        build := addtolist
        go buildWorker(build)
    }
}
*/
