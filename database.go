package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"reflect"
	"strconv"
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
	l.m[build.Key()] = build
	l.Unlock()
}

func (l *RunningList) Remove(build RunningBuild) {
	l.Lock()
	delete(l.m, build.Key())
	l.Unlock()
}

func (l *RunningList) Get(build *Build) (RunningBuild, bool) {
	l.RLock()
	entry, ok := l.m[build.Key()]
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

func AllRepositories() (repos []*Repository, err error) {
	err = DB.View(func(tx *bolt.Tx) error {
		rtype := reflect.TypeOf((*Repository)(nil))
		repos, e := listAndDecode(tx, tx.Bucket(dbRepositories), rtype)
		return e
	})
	return
}

func AllBuilds() (builds []*Build, err error) {
	err = DB.View(func(tx *bolt.Tx) error {
		rtype := reflect.TypeOf((*Build)(nil))
		builds, e := listAndDecode(tx, tx.Bucket(dbBuilds), rtype)
		return e
	})
	return
}

func SaveRepository(repo *Repository) error {
	return DB.Update(func(tx *bolt.Tx) (e error) {
		if repo.ID == 0 {
			repo.ID, e = incrementId(tx, dbRepositories)
			if e != nil {
				return
			}
		}
		return encodeAndPut(tx, tx.Bucket(dbRepositories),
			strconv.AppendInt(nil, int64(repo.ID), 10), repo)
	})
}

func SaveBuild(build *Build) error {
	return DB.Update(func(tx *bolt.Tx) error {
		if build.ID == 0 {
			var repo *Repository
			err := getAndDecode(tx, tx.Bucket(dbRepositories),
				[]byte(strconv.Itoa(build.RepositoryID)), repo)
			if err != nil {
				return err
			}
			repo.LastBuildID++
			encodeAndPut(tx, tx.Bucket(dbRepositories),
				strconv.AppendInt(nil, int64(repo.ID), 10), repo)
			build.ID = repo.LastBuildID
		}

		return encodeAndPut(tx, tx.Bucket(dbBuilds), []byte(build.Key()), build)
	})
}

func FindRepository(id int) (*Repository, error) {
	var repo *Repository
	err := DB.View(func(tx *bolt.Tx) error {
		return getAndDecode(tx, tx.Bucket(dbRepositories), []byte(strconv.Itoa(id)), repo)
	})
	return repo, err
}

func FindBuild(key string) (*Build, error) {
	var build *Build
	err := DB.View(func(tx *bolt.Tx) error {
		return getAndDecode(tx, tx.Bucket(dbBuilds), []byte(key), build)
	})
	return build, err
}

func listAndDecode(tx *bolt.Tx, bucket *bolt.Bucket, elemType reflect.Type) ([]interface{}, error) {
	var result []interface{}
	var buffer bytes.Buffer
	cursor := bucket.Cursor()
	for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
		dec := gob.NewDecoder(&buffer)
		_, err := buffer.Write(v)
		if err != nil {
			return nil, err
		}
		value := reflect.New(elemType)
		err = dec.DecodeValue(value)
		if err != nil {
			return nil, err
		}
		result = append(result, value.Pointer())
		buffer.Reset()
	}
	return result, nil
}

func encodeAndPut(tx *bolt.Tx, bucket *bolt.Bucket, key []byte, payload interface{}) error {
	var buffer bytes.Buffer
	err := gob.NewEncoder(&buffer).Encode(payload)
	if err != nil {
		return err
	}
	return bucket.Put(key, buffer.Bytes())
}

func getAndDecode(tx *bolt.Tx, bucket *bolt.Bucket, key []byte, result interface{}) error {
	value := bucket.Get(key)
	if value == nil {
		return nil
	}
	reader := bytes.NewReader(value)
	return gob.NewDecoder(reader).Decode(result)
}

func incrementId(tx *bolt.Tx, sequenceKey []byte) (id int, err error) {
	idsBucket := tx.Bucket(dbIds)
	value := idsBucket.Get(sequenceKey)
	if value != nil {
		id = int(binary.LittleEndian.Uint32(value))
	}
	id++
	var encodedId [4]byte
	binary.LittleEndian.PutUint32(encodedId[:], uint32(id))
	err = idsBucket.Put(sequenceKey, encodedId[:])
	return
}
