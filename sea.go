package main

import (
    "fmt"
    "flag"
    "log"
    "net/http"
    "os/exec"
    "io"
    // "os"
    // "bytes"
    // "bufio"
)

type OutputBuffer struct {
    Content []byte
    Stream chan []byte
    Done bool
}

var cmdOut OutputBuffer

func (o *OutputBuffer) Write(p []byte) (int, error) {
    o.Stream <- p
    o.Content = append(o.Content, p...)
    return len(p), nil
}

func (o *OutputBuffer) ReadChunks() chan []byte {
    response := make(chan []byte)
    go func() {
        if len(o.Content) > 0 {
            response <- o.Content
        }
        if o.Done {
            close(response)
        } else {
            for chunk := range o.Stream {
                response <- chunk
            }
            close(response)
        }
    }()
    return response
}

func execFile(path string, output *OutputBuffer) {
    cmd := exec.Command(path)
    cmdout, err := cmd.StdoutPipe()
    if err != nil {
        log.Fatal(err)
    }
    err = cmd.Start()
    if err != nil {
        log.Fatal(err)
    }
    io.Copy(output, cmdout)
    output.Done = true
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
    cmdOut = OutputBuffer {
        Content: make([]byte, 0),
        Stream: make(chan []byte),
        Done: false,
    }

    var port int
    flag.IntVar(&port, "port", 8080, "HTTP port to listen")
    flag.Parse()

    http.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
        go execFile("./Seafile", &cmdOut)
        fmt.Fprintf(w, "running")
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
