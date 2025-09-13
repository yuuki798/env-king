package storage

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}
	if hub.Buckets == nil {
		t.Fatal("Hub Buckets map is nil")
	}
	if len(hub.Buckets) != 0 {
		t.Fatal("Hub Buckets should be empty initially")
	}
}

func TestHub_CreateBucketAndKVOperations(t *testing.T) {
	// 清理测试目录
	defer func() {
		_ = os.RemoveAll("./test-bucket-1")
		_ = os.RemoveAll("./test-bucket-2")
	}()

	hub := NewHub()

	// 测试创建第一个bucket
	bucket1, err := hub.CreateOrOpenBucket("test-bucket-1")
	if err != nil {
		t.Fatalf("Failed to create bucket1: %v", err)
	}
	if bucket1 == nil {
		t.Fatal("bucket1 is nil")
	}
	if bucket1.name != "test-bucket-1" {
		t.Fatalf("Expected bucket name 'test-bucket-1', got '%s'", bucket1.name)
	}
	// 测试bucket1的KV操作
	t.Run("Bucket1_KV_Operations", func(t *testing.T) {
		// 测试Put操作
		err := bucket1.Put("key1", "value1")
		if err != nil {
			t.Fatalf("Failed to put key1: %v", err)
		}

		err = bucket1.Put("key2", "value2")
		if err != nil {
			t.Fatalf("Failed to put key2: %v", err)
		}

		err = bucket1.Put("key3", "value3")
		if err != nil {
			t.Fatalf("Failed to put key3: %v", err)
		}

		// 测试Get操作
		value, ok := bucket1.Get("key1")
		if !ok {
			t.Fatal("key1 not found")
		}
		log.Println(value)
		bucket1.tree.Print()
		if value != "value1" {
			t.Fatalf("Expected value1, got %s", value)
		}

		value, ok = bucket1.Get("key2")
		if !ok {
			t.Fatal("key2 not found")
		}
		if value != "value2" {
			t.Fatalf("Expected value2, got %s", value)
		}

		// 测试更新操作
		err = bucket1.Put("key1", "updated-value1")
		if err != nil {
			t.Fatalf("Failed to update key1: %v", err)
		}

		value, ok = bucket1.Get("key1")
		if !ok {
			t.Fatal("key1 not found after update")
		}
		if value != "updated-value1" {
			t.Fatalf("Expected updated-value1, got %s", value)
		}

		// 测试Delete操作
		deleted, err := bucket1.Del("key2")
		if err != nil {
			t.Fatalf("Failed to delete key2: %v", err)
		}
		if !deleted {
			t.Fatal("key2 should be deleted")
		}

		// 验证删除后无法获取
		_, ok = bucket1.Get("key2")
		if ok {
			t.Fatal("key2 should not exist after deletion")
		}
		// 测试删除不存在的key
		deleted, err = bucket1.Del("non-existent-key")
		if err == nil {
			t.Fatal("Expected error when deleting non-existent key")
		}
		if deleted {
			t.Fatal("Should not report successful deletion of non-existent key")
		}
	})
	bucket1.tree.Print()
	// 测试大量数据操作
	t.Run("Bulk_Operations", func(t *testing.T) {
		// 在bucket1中插入大量数据
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("bulk-key-%03d", i)
			value := fmt.Sprintf("bulk-value-%03d", i)
			err := bucket1.Put(key, value)
			if err != nil {
				t.Fatalf("Failed to put %s: %v", key, err)
			}
		}

		// 验证插入的数据
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("bulk-key-%03d", i)
			expectedValue := fmt.Sprintf("bulk-value-%03d", i)
			value, ok := bucket1.Get(key)
			if !ok {
				t.Fatalf("Key %s not found", key)
			}
			if value != expectedValue {
				t.Fatalf("For key %s, expected %s, got %s", key, expectedValue, value)
			}
		}

		// 删除部分数据
		for i := 0; i < 50; i++ {
			key := fmt.Sprintf("bulk-key-%03d", i)
			deleted, err := bucket1.Del(key)
			if err != nil {
				t.Fatalf("Failed to delete %s: %v", key, err)
			}
			if !deleted {
				t.Fatalf("Key %s should be deleted", key)
			}
		}

		// 验证删除的数据不存在
		for i := 0; i < 50; i++ {
			key := fmt.Sprintf("bulk-key-%03d", i)
			_, ok := bucket1.Get(key)
			if ok {
				t.Fatalf("Key %s should not exist after deletion", key)
			}
		}

		// 验证剩余数据仍然存在
		for i := 50; i < 100; i++ {
			key := fmt.Sprintf("bulk-key-%03d", i)
			expectedValue := fmt.Sprintf("bulk-value-%03d", i)
			value, ok := bucket1.Get(key)
			if !ok {
				t.Fatalf("Key %s should still exist", key)
			}
			if value != expectedValue {
				t.Fatalf("For key %s, expected %s, got %s", key, expectedValue, value)
			}
		}
	})
	bucket1.tree.Print()
}
func TestRecover(t *testing.T) {
	hub := NewHub()
	// 测试创建第一个bucket
	bucket1, err := hub.CreateOrOpenBucket("test-bucket-1")
	if err != nil {
		t.Fatalf("Failed to create bucket1: %v", err)
	}
	bucket1.tree.Print()
}
