package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/libgit2/git2go"
)

type Repository struct {
	ID     int
	Name   string
	Remote bool
	Url    string

	LastBuildID int
}

func (r *Repository) LocalPath() string {
	return path.Join(Config.ReposPath, fmt.Sprintf("%d.git", r.ID))
}

func StartRepository(r *Repository) (err error) {
	SaveRepository(r)
	if r.Remote {
		_, err = git.Clone(r.Url, r.LocalPath(), &git.CloneOptions{
			Bare: true,
			RemoteCallbacks: &git.RemoteCallbacks{
				CertificateCheckCallback: gitCertificateCheckCallback,
				CredentialsCallback:      gitCredentialsCallback,
			},
		})
	} else {
		_, err = git.InitRepository(r.LocalPath(), true)
	}
	return
}

func (r *Repository) StartBuild(gitRevision string, gitBranch string) error {
	prefix := fmt.Sprintf("sea_%d_", r.ID)
	directory, err := ioutil.TempDir("tmp", prefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	log.Printf("Temp build dir: %s", directory)

	repo, err := git.OpenRepository(r.LocalPath())
	if err != nil {
		return err
	}

	if r.Remote {
		remote, err := repo.LookupRemote("origin")
		if err != nil {
			return err
		}
		remote.SetCallbacks(&git.RemoteCallbacks{
			CertificateCheckCallback: gitCertificateCheckCallback,
			CredentialsCallback:      gitCredentialsCallback,
		})
		err = remote.Fetch(nil, nil, "")
		if err != nil {
			return err
		}
	}

	oid, err := git.NewOid(gitRevision)
	if err != nil {
		return err
	}
	commit, err := repo.LookupCommit(oid)
	if err != nil {
		return err
	}
	tree, err := commit.Tree()
	if err != nil {
		return err
	}

	err = repo.CheckoutTree(tree, &git.CheckoutOpts{
		Strategy:        git.CheckoutForce,
		TargetDirectory: directory,
	})

	if err != nil {
		return err
	}

	commitAuthor := commit.Author()

	// TODO: how to notify users of errors that ocurred before the build started
	// to execute?
	build := NewRunningBuild(&Build{
		RepositoryID: r.ID,
		Commit: CommitInfo{
			Revision:    gitRevision,
			Message:     commit.Message(),
			Branch:      gitBranch,
			AuthorName:  commitAuthor.Name,
			AuthorEmail: commitAuthor.Email,
		},
		State:     BuildRunning,
		Path:      directory,
		StartedAt: time.Now(),
	})
	RunningBuilds.Add(build)
	SaveBuild(build.Build)
	defer RunningBuilds.Remove(build)
	defer SaveBuild(build.Build)
	defer func() { build.FinishedAt = time.Now() }()

	script := filepath.Join(build.Path, "Seafile")

	cmd := exec.Command(script)
	cmd.Stdout = build.Buffer
	cmd.Stderr = build.Buffer
	defer func() {
		build.Buffer.End()
		build.Output = build.Buffer.Bytes()
	}()

	err = cmd.Start()
	if err != nil {
		return err
	}

	waitResult := make(chan error)
	go func() { waitResult <- cmd.Wait() }()

	select {
	case err = <-waitResult:
		if err == nil {
			build.State = BuildSuccess
		} else if exit, ok := err.(*exec.ExitError); ok {
			build.State = BuildFailed
			ws := exit.ProcessState.Sys().(syscall.WaitStatus) // will panic if not Unix
			build.ReturnCode = ws.ExitStatus()
		} else {
			panic(err)
		}
	case <-build.cancel:
		err = syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)
		// ESRCH: process already finished
		if err != nil && err != syscall.ESRCH {
			return err
		}
		build.State = BuildCanceled
	}

	return nil
}

func gitCertificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	// TODO: do real validation here, it will always be invalid for SSH
	if cert.Kind == git.CertificateHostkey || cert.Kind == git.CertificateX509 {
		return git.ErrOk
	} else {
		log.Printf("ERROR: %v certificate type", cert.Kind)
		return git.ErrUser
	}
}

func gitCredentialsCallback(url string, urlUser string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
	if allowedTypes&git.CredTypeSshKey == 0 {
		log.Print("ERROR: SSH Key cred type not allowed")
		return git.ErrUser, nil
	}
	// errCode, cred := git.NewCredSshKeyFromAgent(urlUser)
	errCode, cred := git.NewCredSshKey(urlUser,
		os.Getenv("HOME")+"/.ssh/id_rsa.pub",
		os.Getenv("HOME")+"/.ssh/id_rsa", "")
	if errCode != 0 {
		log.Printf("ERROR: NewCredSshKeyFromAgent returned error: %d", errCode)
		return git.ErrUser, nil
	}
	return git.ErrOk, &cred
}
