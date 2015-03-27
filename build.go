package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	git "github.com/libgit2/git2go"
)

type BuildState uint

const (
	BuildWaiting BuildState = iota
	BuildRunning
	BuildFailed
	BuildCanceled
	BuildSuccess
)

var stateNames = []string{
	"Wating",
	"Running",
	"Failed",
	"Canceled",
	"Success",
}

type Build struct {
	Rev        string
	State      BuildState
	Path       string
	Output     *OutputBuffer
	ReturnCode int

	cancel chan struct{}
}

func (b *Build) StateName() string {
	return stateNames[b.State]
}

// TODO: what to do with build state on error?
func (b *Build) Exec() error {
	script := filepath.Join(b.Path, "Seafile")

	cmd := exec.Command(script)
	cmd.Stdout = b.Output
	cmd.Stderr = b.Output
	defer b.Output.End()

	err := cmd.Start()
	if err != nil {
		return err
	}

	b.State = BuildRunning
	waitResult := make(chan error)
	go func() { waitResult <- cmd.Wait() }()

	b.cancel = make(chan struct{}, 1) // TODO: why this channel have to be buffered?

	select {
	case err = <-waitResult:
		if err == nil {
			b.State = BuildSuccess
		} else if exit, ok := err.(*exec.ExitError); ok {
			b.State = BuildFailed
			ws := exit.ProcessState.Sys().(syscall.WaitStatus) // will panic if not Unix
			b.ReturnCode = ws.ExitStatus()
		} else {
			return err
		}
	case <-b.cancel:
		err = syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)
		if err != nil && err != syscall.ESRCH { // Already finished
			return err
		}

		b.State = BuildCanceled
	}

	return nil
}

func (b *Build) Cancel() (err error) {
	if b.State == BuildRunning {
		b.cancel <- struct{}{} // TODO: maybe this shouldn't block...
	} else {
		err = errors.New("build must be in running state to cancel")
	}
	return
}

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
