package main

import (
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
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
	RepositoryId int
	Rev          string
	State        BuildState
	Path         string
	Output       []byte
	ReturnCode   int
	StartedAt    time.Time
	FinishedAt   time.Time
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
