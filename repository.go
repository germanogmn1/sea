package main

import (
	"log"

	"github.com/libgit2/git2go"
)

type Repository struct {
	Id     uint
	Remote bool
}

func CloneRepository(url string, path string) (*git.Repository, error) {
	return git.Clone(url, path, &git.CloneOptions{
		RemoteCallbacks: &git.RemoteCallbacks{
			CertificateCheckCallback: certificateCheckCallback,
			CredentialsCallback:      credentialsCallback,
		},
	})
}

func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	// TODO: do real validation here, it will always be invalid for SSH
	if cert.Kind == git.CertificateHostkey {
		return git.ErrOk
	} else {
		log.Print("ERROR: %v certificate type", cert.Kind)
		return git.ErrUser
	}
}

func credentialsCallback(url string, urlUser string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
	if allowedTypes&git.CredTypeSshKey == 0 {
		log.Print("ERROR: SSH Key cred type not allowed")
		return git.ErrUser, nil
	}
	errCode, cred := git.NewCredSshKeyFromAgent(urlUser)
	if errCode != 0 {
		log.Printf("ERROR: NewCredSshKeyFromAgent returned error: %d", errCode)
		return git.ErrUser, nil
	}
	return git.ErrOk, &cred
}
