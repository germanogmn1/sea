package main

import (
	"strings"

	"github.com/boltdb/bolt"
)

var DB *bolt.DB
var buildsRunning []*Build

func InitDB(dbPath string) error {
	var err error
	DB, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}
	return DB.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists([]byte("builds"))
		return e
	})
}

func AddBuild(b *Build) {
	buildList = append(buildList, b)
}

func FindBuild(rev string) *Build {
	for i := range buildList {
		if strings.HasPrefix(buildList[i].Rev, rev) {
			return buildList[i]
		}
	}
	return nil
}
