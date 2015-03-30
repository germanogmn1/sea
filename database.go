package main

import (
	"bytes"
	"strings"

	"github.com/boltdb/bolt"
)

var (
	DB            *bolt.DB
	buildsRunning []*Build
)

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
	buildsRunning = append(buildsRunning, b)
}

func FindBuild(rev string) *Build {
	for i := range buildsRunning {
		if strings.HasPrefix(buildsRunning[i].Rev, rev) {
			return buildsRunning[i]
		}
	}

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("builds"))

		prefix := []byte(rev)
		k, v := c.Seek(prefix)
		k, v = c.Next()
		bytes.HasPrefix(k, prefix)

		return nil
	})

	return nil
}
