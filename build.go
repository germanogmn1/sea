package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
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

type CommitInfo struct {
	Revision string
	Message  string
	Branch   string

	AuthorName  string
	AuthorEmail string
}

func GravatarUrl(email []byte, size int) string {
	data := bytes.TrimSpace(bytes.ToLower(email))
	sum := md5.New().Sum(data)
	url := fmt.Sprintf("https://secure.gravatar.com/avatar/%x?size=%d&default=mm", sum, size)
	return url
}

type Build struct {
	ID           int
	RepositoryID int
	Commit       CommitInfo
	State        BuildState
	Path         string
	Output       []byte
	ReturnCode   int
	StartedAt    time.Time
	FinishedAt   time.Time
}

func (b *Build) Key() string {
	return fmt.Sprintf("%d/%d", b.RepositoryID, b.ID)
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
