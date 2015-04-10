package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/libgit2/git2go"
)

type Repository struct {
	Id     int
	Name   string
	Remote bool
	Url    string
}

func (r *Repository) LocalPath() string {
	return path.Join(Config.ReposPath, fmt.Sprintf("%d.git", r.Id))
}

func CreateLocalRepository() *Repository {
	r := new(Repository)
	r.Remote = false
	SaveRepository(r)
	git.InitRepository(r.LocalPath(), true)
	return r
}

func CreateRemoteRepository() *Repository {
	r := new(Repository)
	r.Remote = true
	SaveRepository(r)
	git.Clone(r.Url, r.LocalPath(), &git.CloneOptions{
		Bare: true,
		RemoteCallbacks: &git.RemoteCallbacks{
			CertificateCheckCallback: gitCertificateCheckCallback,
			CredentialsCallback:      gitCredentialsCallback,
		},
	})
	return r
}

func (r *Repository) StartBuild(rev string) error {
	prefix := fmt.Sprintf("sea_%d_", r.Id)
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
		remote.Fetch("", nil, nil)
	}

	oid, err := git.NewOid(rev)
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
	return repo.CheckoutTree(tree, &git.CheckoutOpts{
		Strategy:        git.CheckoutForce,
		TargetDirectory: directory,
	})

	// TODO: how to notify users of errors that ocurred before the build started
	// to execute?
	build := NewRunningBuild(&Build{
		RepositoryId: r.Id,
		Rev:          rev,
		State:        BuildRunning,
		Path:         directory,
		StartedAt:    time.Now(),
	})
	RunningBuilds.Add(build)
	SaveBuild(build.Build)
	defer RunningBuilds.Remove(build.Rev)
	defer SaveBuild(build.Build)
	defer func() { build.FinishedAt = time.Now() }()

	return build.Exec()
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
