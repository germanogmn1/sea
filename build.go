package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

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
	StartedAt  time.Time
	FinishedAt time.Time
}

func (b *Build) Duration() time.Duration {
	return b.FinishedAt.Sub(b.StartedAt)
}

type RunningBuild struct {
	*Build
	Buffer *OutputBuffer
	cancel chan struct{}
}

func NewRunningBuild(b *Build) RunningBuild {
	return RunningBuild{
		Build:  b,
		Buffer: NewOutputBuffer(),
		cancel: make(chan struct{}, 1),
	}
}

func (b *RunningBuild) Exec() error {
	script := filepath.Join(b.Path, "Seafile")

	cmd := exec.Command(script)
	cmd.Stdout = b.Buffer
	cmd.Stderr = b.Buffer
	defer func() {
		b.Buffer.End()
		b.Output = b.Buffer.Bytes()
	}()

	err := cmd.Start()
	if err != nil {
		return err
	}

	waitResult := make(chan error)
	go func() { waitResult <- cmd.Wait() }()

	select {
	case err = <-waitResult:
		if err == nil {
			b.State = BuildSuccess
		} else if exit, ok := err.(*exec.ExitError); ok {
			b.State = BuildFailed
			ws := exit.ProcessState.Sys().(syscall.WaitStatus) // will panic if not Unix
			b.ReturnCode = ws.ExitStatus()
		} else {
			panic(err)
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

func (b *RunningBuild) Cancel() {
	close(b.cancel)
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
	build := NewRunningBuild(&Build{
		Rev:       hook.NewRev,
		State:     BuildRunning,
		Path:      directory,
		StartedAt: time.Now(),
	})
	RunningBuilds.Add(build)
	SaveBuild(build.Build)
	defer RunningBuilds.Remove(build.Rev)
	defer SaveBuild(build.Build)
	defer func() { build.FinishedAt = time.Now() }()
	err = build.Exec()
	check(err)
}
