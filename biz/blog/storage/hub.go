package storage

import (
	"sync"
	"yuuki798/env-king/biz/blog/myutil"
)

type Hub struct {
	mu      sync.RWMutex
	Buckets map[string]*Bucket
}

type Bucket struct {
	name              string
	tree              *BPlusTree
	wal               *WalWriter
	persistenceEngine *PersistenceEngine

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{Buckets: map[string]*Bucket{}}
}

func (this *Hub) CreateOrOpenBucket(name string) (*Bucket, error) {
	this.mu.Lock()
	if this.Buckets[name] != nil {
		bucket := this.Buckets[name]
		this.mu.Unlock()
		return bucket, nil
	}
	this.mu.Unlock()
	tree := NewBPlusTree(2)
	myutil.Mkdir("./store/" + name)
	wal, err := NewWalWriter("./store/"+name+"/wal.dat", tree)
	if err != nil {
		return nil, err
	}
	persis := NewPersistenceEngine("./store/"+name+"/snapshots", tree, wal)
	bucket := &Bucket{
		name:              name,
		tree:              tree,
		wal:               wal,
		persistenceEngine: persis,
	}
	this.mu.Lock()
	this.Buckets[name] = bucket
	this.mu.Unlock()
	return bucket, nil
}

func (this *Bucket) Put(key, value string) (err error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	e := &LogEntry{
		op:    PUT_OP,
		key:   key,
		value: value,
	}
	err = this.wal.Append(e)
	if err != nil {
		return err
	}
	return this.tree.Put(key, value)
}
func (this *Bucket) Get(key string) (value string, ok bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.tree.Get(key)
}

func (this *Bucket) Del(key string) (ok bool, err error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	e := &LogEntry{
		op:    DEL_OP,
		key:   key,
		value: "",
	}
	err = this.wal.Append(e)
	if err != nil {
		return false, err
	}
	return this.tree.Del(key)
}
