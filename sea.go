package main

import (
	"flag"
	"io/ioutil"
	"log"
	"path/filepath"

	git "gopkg.in/libgit2/git2go.v22"
)

func StartLocalBuild(hook GitHook) {
	check := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	repo, err := git.OpenRepository(hook.RepoPath)
	check(err)
	oid, err := git.NewOid(hook.NewRev)
	check(err)
	commit, err := repo.LookupCommit(oid)
	check(err)
	tree, err := commit.Tree()
	check(err)
	prefix := "sea_" + filepath.Base(hook.RepoPath)
	directory, err := ioutil.TempDir("tmp", prefix) // TODO: delete dir somewere
	check(err)
	log.Printf("Temp build dir: %s", directory)

	err = repo.CheckoutTree(tree, &git.CheckoutOpts{
		Strategy:        git.CheckoutForce,
		TargetDirectory: directory,
	})
	check(err)

	build := &Build{
		Rev:    hook.NewRev,
		State:  BUILD_WAITING,
		Path:   directory,
		Output: NewEmptyOutputBuffer(),
	}
	AddBuild(build)
	build.Exec()
}

// TODO: handle SIGINT nicely (http://www.hydrogen18.com/blog/stop-listening-http-server-go.html)
func main() {
	var pipePath, webAddr string
	flag.StringVar(&webAddr, "addr", ":8080", "TCP address to listen on")
	flag.StringVar(&pipePath, "pipe", "./tmp/seapipe", "named pipe to listen for git hooks.")
	flag.Parse()

	go WebServer(webAddr)

	hooks, errors := ListenGitHooks(pipePath)
	for {
		select {
		case hook := <-hooks:
			go StartLocalBuild(hook)
		case err := <-errors:
			log.Fatal(err)
		}
	}
}
