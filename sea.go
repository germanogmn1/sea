package main

import (
    "fmt"
    // "flag"
    // "log"
    "time"
    "net/http"
    // "os/exec"
    // "io"
    // "os"
    // "bytes"
    // "bufio"
)

type outBuffer struct {
    Content []byte
    Stream chan []byte
    Done bool
}

func (o *outBuffer) Write(p []byte) (int, error) {
    append(o.Content, p...)
    o.Stream <- p
    return len(p), nil
}

func (o *outBuffer) ReadChunks() chan []byte {
    response := make(chan []byte)
    if len(o.Content) > 0 {
        response <- o.Content
    }
    if !o.Done {
        go func() {
            for chunk := range o.Stream {
                response <- chunk
            }
        }
    }
    return response
}

func execFile(path string, output io.Writer) {
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
    cmd.Wait()
}

func stream(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("X-Content-Type-Options", "nosniff")
    for i := 0; i < 10; i++ {
        time.Sleep(1 * time.Second)
        fmt.Fprintf(w, fmt.Sprintf("exec %d\n", i))
        w.(http.Flusher).Flush()
    }
}

func main() {
    var port int
    flag.IntVar(&port, "port", 8080, "HTTP port to listen")
    flag.Parse()

    http.HandleFunc("/stream", stream)
    http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

/*
func main() {
    

    http.HandleFunc("/", handler)
    http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}


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
