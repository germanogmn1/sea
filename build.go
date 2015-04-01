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
	BuildRunning BuildState = iota
	BuildFailed
	BuildCanceled
	BuildSuccess
)

var stateNames = [...]string{
	"Running",
	"Failed",
	"Canceled",
	"Success",
}

// fmt.Stringer
func (s BuildState) String() string {
	return stateNames[s]
}

type Build struct {
	Rev        string
	State      BuildState
	Path       string
	Output     []byte
	ReturnCode int

	cancel chan struct{}
}

// TODO: what to do with build state on error? Canceled?
func (b *Build) Exec(output *OutputBuffer) error {
	script := filepath.Join(b.Path, "Seafile")

	cmd := exec.Command(script)
	cmd.Stdout = output
	cmd.Stderr = output
	defer func() {
		output.End()
		b.Output = output.Bytes()
		SaveBuild(b)
		RunningBuilds.Remove(b.Rev)
	}()

	err := cmd.Start()
	if err != nil {
		return err
	}

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
		err = errors.New("build must be running to cancel")
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

	// TODO: how to notify users of errors that ocurred before the build started
	// to execute?
	build := &Build{
		Rev:   hook.NewRev,
		State: BuildRunning,
		Path:  directory,
	}
	out := NewEmptyOutputBuffer()
	RunningBuilds.Add(build, out)
	SaveBuild(build)
	err = build.Exec(out)
	check(err)
}
