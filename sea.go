package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	git "gopkg.in/libgit2/git2go.v22"
)

func StartLocalBuild(hook GitHook, wg *sync.WaitGroup) {
	defer wg.Done()
	check := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	prefix := "sea_" + filepath.Base(hook.RepoPath)
	directory, err := ioutil.TempDir("tmp", prefix)
	defer os.RemoveAll(directory)
	check(err)
	log.Printf("Temp build dir: %s", directory)

	build := &Build{
		Rev:    hook.NewRev,
		State:  BuildWaiting,
		Path:   directory,
		Output: NewEmptyOutputBuffer(),
	}
	AddBuild(build) // Add build before checkout because it can take time...

	repo, err := git.OpenRepository(hook.RepoPath)
	check(err)
	oid, err := git.NewOid(hook.NewRev)
	check(err)
	commit, err := repo.LookupCommit(oid)
	check(err)
	tree, err := commit.Tree()
	check(err)
	err = repo.CheckoutTree(tree, &git.CheckoutOpts{
		Strategy:        git.CheckoutForce,
		TargetDirectory: directory,
	})
	check(err)

	err = build.Exec()
	check(err)
}

// TODO: handle SIGINT nicely (http://www.hydrogen18.com/blog/stop-listening-http-server-go.html)
// TODO: prevent hook script from blocking when writing on pipe
func main() {
	var pipePath, webAddr string
	flag.StringVar(&webAddr, "addr", ":8080", "TCP address to listen on")
	flag.StringVar(&pipePath, "pipe", "./tmp/seapipe", "named pipe to listen for git hooks.")
	flag.Parse()

	go WebServer(webAddr)

	var wg sync.WaitGroup

	sigint := make(chan os.Signal)
	signal.Notify(sigint, os.Interrupt)

	stop := make(chan struct{})
	wg.Add(1)
	hooks, errors := ListenGitHooks(pipePath, &wg, stop)
	for {
		select {
		case hook := <-hooks:
			wg.Add(1)
			go StartLocalBuild(hook, &wg)
		case err := <-errors:
			log.Fatal(err)
		case <-sigint:
			log.Print("Exiting...")
			close(stop)
			for _, build := range buildList {
				if build.State == BuildRunning {
					err := build.Cancel()
					if err != nil {
						log.Print("Failed to stop build:", err)
					}
				}
			}
			wg.Wait()
			os.Exit(130)
		}
	}
}
