package main

import (
    "fmt"
    "flag"
    "log"
    "net/http"
    "os/exec"
)

/*
func main() {
    var port int
    flag.IntVar(&port, "port", 8080, "HTTP port to listen")
    flag.Parse()

    http.HandleFunc("/", handler)
    http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}
*/

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
    Output []string
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
