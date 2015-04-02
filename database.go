package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"sync"

	"github.com/boltdb/bolt"
)

type RunningList struct {
	sync.RWMutex

	// Stores the output buffer of running builds indexed by revision hash.
	// There is a non-zero chance that commits of different git repositories to have
	// the same hash, but it's is very unlikely.
	m map[string]RunningBuild
}

func (l *RunningList) Add(build RunningBuild) {
	l.Lock()
	l.m[build.Rev] = build
	l.Unlock()
}

func (l *RunningList) Remove(rev string) {
	l.Lock()
	delete(l.m, rev)
	l.Unlock()
}

func (l *RunningList) Get(rev string) (RunningBuild, bool) {
	l.RLock()
	entry, ok := l.m[rev]
	l.RUnlock()
	return entry, ok
}

func (l *RunningList) CancelAll() {
	l.RLock()
	for _, build := range l.m {
		build.Cancel()
	}
	l.RUnlock()
}

var RunningBuilds RunningList
var DB *bolt.DB

func InitDB() error {
	RunningBuilds = RunningList{sync.RWMutex{}, make(map[string]RunningBuild)}

	var err error
	DB, err = bolt.Open(Config.DBPath, 0600, nil)
	if err != nil {
		return err
	}
	return DB.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists([]byte("builds"))
		return e
	})
}

func AllBuilds() []*Build {
	var buffer bytes.Buffer
	var dec *gob.Decoder
	var build *Build
	var builds []*Build

	err := DB.View(func(tx *bolt.Tx) error {
		cursor := tx.Bucket([]byte("builds")).Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			dec = gob.NewDecoder(&buffer)
			build = new(Build)
			_, e := buffer.Write(v)
			if e != nil {
				return e
			}
			e = dec.Decode(&build)
			if e != nil {
				return e
			}
			builds = append(builds, build)
			buffer.Reset()
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	return builds
}

func SaveBuild(build *Build) {
	var err error
	var buffer bytes.Buffer

	enc := gob.NewEncoder(&buffer)
	err = enc.Encode(build)
	if err != nil {
		panic(err)
	}

	err = DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("builds"))
		return bucket.Put([]byte(build.Rev), buffer.Bytes())
	})

	if err != nil {
		panic(err)
	}
}

func FindBuild(revPrefix string) *Build {
	var build *Build

	err := DB.View(func(tx *bolt.Tx) error {
		cursor := tx.Bucket([]byte("builds")).Cursor()

		prefix := []byte(revPrefix)
		key, value := cursor.Seek(prefix)
		if key == nil {
			return nil
		}

		// Sanity check: ensure that the prefix is not ambiguous
		nextKey, _ := cursor.Next()
		if bytes.HasPrefix(nextKey, prefix) {
			return errors.New("ambiguous revision prefix")
		}

		build = new(Build)
		reader := bytes.NewReader(value)
		return gob.NewDecoder(reader).Decode(build)
	})

	if err != nil {
		panic(err)
	}

	return build
}
