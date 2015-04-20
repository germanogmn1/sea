package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"sync"

	"github.com/boltdb/bolt"
)

var (
	DB *bolt.DB

	RunningBuilds RunningList

	// Buckets
	dbIds          = []byte("ids")
	dbRepositories = []byte("repositories")
	dbBuilds       = []byte("builds")

	dbBuckets = [...][]byte{dbIds, dbRepositories, dbBuilds}
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

func InitDB() error {
	RunningBuilds = RunningList{sync.RWMutex{}, make(map[string]RunningBuild)}

	var err error
	DB, err = bolt.Open(Config.DBPath, 0600, nil)
	if err != nil {
		return err
	}

	return DB.Update(func(tx *bolt.Tx) error {
		for _, bucket := range dbBuckets {
			if _, e := tx.CreateBucketIfNotExists(bucket); e != nil {
				return e
			}
		}
		return nil
	})
}

func incrementId(tx *bolt.Tx, bucketName []byte) (id int, idBytes [4]byte, err error) {
	idsBucket := tx.Bucket(dbIds)
	value := idsBucket.Get(bucketName)
	if value != nil {
		id = int(binary.LittleEndian.Uint32(value))
	}
	id++
	binary.LittleEndian.PutUint32(idBytes[:], uint32(id))
	err = idsBucket.Put(bucketName, idBytes[:])
	return
}

func AllRepositories() []*Repository {
	var buffer bytes.Buffer
	var dec *gob.Decoder
	var repo *Repository
	var repos []*Repository

	err := DB.View(func(tx *bolt.Tx) error {
		cursor := tx.Bucket(dbRepositories).Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			dec = gob.NewDecoder(&buffer)
			repo = new(Repository)
			_, e := buffer.Write(v)
			if e != nil {
				return e
			}
			e = dec.Decode(&repo)
			if e != nil {
				return e
			}
			repos = append(repos, repo)
			buffer.Reset()
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	return repos
}

// Will generate a new Id if repo.Id == 0
func SaveRepository(repo *Repository) {
	err := DB.Update(func(tx *bolt.Tx) (e error) {
		var key [4]byte
		if repo.Id == 0 {
			repo.Id, key, e = incrementId(tx, dbRepositories)
			if e != nil {
				return e
			}
		} else {
			binary.LittleEndian.PutUint32(key[:], uint32(repo.Id))
		}
		var buffer bytes.Buffer
		enc := gob.NewEncoder(&buffer)
		if enc.Encode(repo); e != nil {
			return e
		}
		return tx.Bucket(dbRepositories).Put(key[:], buffer.Bytes())
	})
	if err != nil {
		panic(err)
	}
}

func FindRepository(id int) *Repository {
	var key [4]byte
	binary.LittleEndian.PutUint32(key[:], uint32(id))

	var repo *Repository
	err := DB.View(func(tx *bolt.Tx) error {
		value := tx.Bucket(dbRepositories).Get(key[:])
		if value == nil {
			return nil
		}
		repo = new(Repository)
		reader := bytes.NewReader(value)
		return gob.NewDecoder(reader).Decode(repo)
	})
	if err != nil {
		panic(err)
	}

	return repo
}

func AllBuilds() []*Build {
	var buffer bytes.Buffer
	var dec *gob.Decoder
	var build *Build
	var builds []*Build

	err := DB.View(func(tx *bolt.Tx) error {
		cursor := tx.Bucket(dbBuilds).Cursor()
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
		bucket := tx.Bucket(dbBuilds)
		return bucket.Put([]byte(build.Rev), buffer.Bytes())
	})

	if err != nil {
		panic(err)
	}
}

func FindBuild(revPrefix string) *Build {
	var build *Build

	err := DB.View(func(tx *bolt.Tx) error {
		cursor := tx.Bucket(dbBuilds).Cursor()

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
