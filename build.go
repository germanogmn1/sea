package main

import "time"

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

func (b *RunningBuild) Cancel() {
	close(b.cancel)
}
