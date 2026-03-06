package infra

import (
	"errors"
	"fmt"

	"go.etcd.io/bbolt"
)

var (
	ErrNotFound = errors.New("store: key not found")
	ErrClosed   = errors.New("store: db closed")
)

// Store 基于 bbolt 的键值存储，按 bucket 组织。
type Store struct {
	db *bbolt.DB
}

// OpenStore 打开或创建 bbolt 数据库文件，返回 Store。调用方需在不用时调用 Close。
func OpenStore(path string, mode *bbolt.Options) (*Store, error) {
	if mode == nil {
		mode = &bbolt.Options{}
	}
	db, err := bbolt.Open(path, 0600, mode)
	if err != nil {
		return nil, fmt.Errorf("store: open bbolt: %w", err)
	}
	return &Store{db: db}, nil
}

// Close 关闭数据库。
func (s *Store) Close() error {
	if s.db == nil {
		return ErrClosed
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// Get 在 bucket 中取 key 对应的值。若 key 不存在返回 ErrNotFound。
func (s *Store) Get(bucket, key []byte) ([]byte, error) {
	if s.db == nil {
		return nil, ErrClosed
	}
	var out []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return ErrNotFound
		}
		v := b.Get(key)
		if v == nil {
			return ErrNotFound
		}
		out = append([]byte(nil), v...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetString 与 Get 相同，使用字符串 bucket 与 key。
func (s *Store) GetString(bucket, key string) ([]byte, error) {
	return s.Get([]byte(bucket), []byte(key))
}

// Put 在 bucket 中写入 key/value。若 bucket 不存在会自动创建。
func (s *Store) Put(bucket, key, value []byte) error {
	if s.db == nil {
		return ErrClosed
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}
		return b.Put(key, value)
	})
}

// PutString 与 Put 相同，使用字符串 bucket 与 key。
func (s *Store) PutString(bucket, key string, value []byte) error {
	return s.Put([]byte(bucket), []byte(key), value)
}

// Delete 删除 bucket 中的 key。key 不存在不报错。
func (s *Store) Delete(bucket, key []byte) error {
	if s.db == nil {
		return ErrClosed
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		return b.Delete(key)
	})
}

// DeleteString 与 Delete 相同，使用字符串 bucket 与 key。
func (s *Store) DeleteString(bucket, key string) error {
	return s.Delete([]byte(bucket), []byte(key))
}

// Has 判断 bucket 中是否存在 key。
func (s *Store) Has(bucket, key []byte) (bool, error) {
	if s.db == nil {
		return false, ErrClosed
	}
	var ok bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		ok = b.Get(key) != nil
		return nil
	})
	return ok, err
}

// HasString 与 Has 相同，使用字符串 bucket 与 key。
func (s *Store) HasString(bucket, key string) (bool, error) {
	return s.Has([]byte(bucket), []byte(key))
}

// ListKeys 返回 bucket 内所有 key（顺序为 bbolt 的字节序）。bucket 不存在则返回 nil。
func (s *Store) ListKeys(bucket []byte) ([][]byte, error) {
	if s.db == nil {
		return nil, ErrClosed
	}
	var keys [][]byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, _ []byte) error {
			keys = append(keys, append([]byte(nil), k...))
			return nil
		})
	})
	return keys, err
}

// ListKeysString 与 ListKeys 相同，bucket 为字符串。
func (s *Store) ListKeysString(bucket string) ([][]byte, error) {
	return s.ListKeys([]byte(bucket))
}

// Update 在可写事务中执行 fn，用于多键原子更新。
func (s *Store) Update(fn func(tx *bbolt.Tx) error) error {
	if s.db == nil {
		return ErrClosed
	}
	return s.db.Update(fn)
}

// View 在只读事务中执行 fn。
func (s *Store) View(fn func(tx *bbolt.Tx) error) error {
	if s.db == nil {
		return ErrClosed
	}
	return s.db.View(fn)
}
