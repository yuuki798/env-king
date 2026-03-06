package storage

import (
	"os"
	"testing"
)

func TestSnapshotAndRecovery(t *testing.T) {
	// 清理测试目录
	defer func() {
		_ = os.RemoveAll("./test-snapshot")
	}()

	// 创建测试目录
	err := os.MkdirAll("./test-snapshot", 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// 创建测试用的 B+ 树
	tree := NewBPlusTree(3)

	// 插入一些测试数据
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
		"key5": "value5",
	}

	for key, value := range testData {
		err := tree.Put(key, value)
		if err != nil {
			t.Fatalf("Failed to put %s: %v", key, err)
		}
	}

	// 创建 WAL
	wal, err := NewWalWriter("./test-snapshot/wal.dat", tree)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// 创建持久化引擎
	engine := &PersistenceEngine{
		snapshotDir: "./test-snapshot/snapshots",
		tree:        tree,
		wal:         wal,
		snapshotC:   make(chan struct{}, 1),
		stopC:       make(chan struct{}),
	}

	// 创建快照目录
	err = os.MkdirAll("./test-snapshot/snapshots", 0755)
	if err != nil {
		t.Fatalf("Failed to create snapshot directory: %v", err)
	}

	// 测试快照功能
	t.Run("Snapshot", func(t *testing.T) {
		err = engine.snapshot()
		if err != nil {
			t.Fatalf("Failed to create snapshot: %v", err)
		}

		// 检查快照文件是否存在
		files, err := os.ReadDir("./test-snapshot/snapshots")
		if err != nil {
			t.Fatalf("Failed to read snapshot directory: %v", err)
		}

		if len(files) == 0 {
			t.Fatal("No snapshot file created")
		}

		t.Logf("Snapshot created: %s", files[0].Name())
	})

	// 测试恢复功能
	t.Run("Recovery", func(t *testing.T) {
		// 创建新的树和引擎来测试恢复
		newTree := NewBPlusTree(3)
		newWal, err := NewWalWriter("./test-snapshot/wal.dat", newTree)
		if err != nil {
			t.Fatalf("Failed to create new WAL: %v", err)
		}
		defer newWal.Close()

		newEngine := &PersistenceEngine{
			snapshotDir: "./test-snapshot/snapshots",
			tree:        newTree,
			wal:         newWal,
			snapshotC:   make(chan struct{}, 1),
			stopC:       make(chan struct{}),
		}

		// 恢复快照
		err = newEngine.Recover()
		if err != nil {
			t.Fatalf("Failed to recover snapshot: %v", err)
		}

		// 验证数据是否正确恢复
		for key, expectedValue := range testData {
			value, ok := newTree.Get(key)
			if !ok {
				t.Fatalf("Key %s not found after recovery", key)
			}
			if value != expectedValue {
				t.Fatalf("For key %s, expected %s, got %s", key, expectedValue, value)
			}
		}

		t.Log("Snapshot recovery successful")
	})
}

func TestSnapshotWithEmptyTree(t *testing.T) {
	// 清理测试目录
	defer func() {
		_ = os.RemoveAll("./test-snapshot-empty")
	}()

	// 创建测试目录
	err := os.MkdirAll("./test-snapshot-empty", 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// 创建空的 B+ 树
	tree := NewBPlusTree(3)

	// 创建 WAL
	wal, err := NewWalWriter("./test-snapshot-empty/wal.dat", tree)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// 创建持久化引擎
	engine := &PersistenceEngine{
		snapshotDir: "./test-snapshot-empty/snapshots",
		tree:        tree,
		wal:         wal,
		snapshotC:   make(chan struct{}, 1),
		stopC:       make(chan struct{}),
	}

	// 创建快照目录
	err = os.MkdirAll("./test-snapshot-empty/snapshots", 0755)
	if err != nil {
		t.Fatalf("Failed to create snapshot directory: %v", err)
	}

	// 测试空树的快照
	err = engine.snapshot()
	if err != nil {
		t.Fatalf("Failed to create snapshot for empty tree: %v", err)
	}

	// 测试空树的恢复
	newTree := NewBPlusTree(3)
	newWal, err := NewWalWriter("./test-snapshot-empty/wal.dat", newTree)
	if err != nil {
		t.Fatalf("Failed to create new WAL: %v", err)
	}
	defer newWal.Close()

	newEngine := &PersistenceEngine{
		snapshotDir: "./test-snapshot-empty/snapshots",
		tree:        newTree,
		wal:         newWal,
		snapshotC:   make(chan struct{}, 1),
		stopC:       make(chan struct{}),
	}

	err = newEngine.Recover()
	if err != nil {
		t.Fatalf("Failed to recover empty tree snapshot: %v", err)
	}

	// 验证空树恢复后仍然是空的
	_, ok := newTree.Get("nonexistent")
	if ok {
		t.Fatal("Empty tree should not contain any keys after recovery")
	}

	t.Log("Empty tree snapshot and recovery successful")
}

func TestSnapshotWithComplexTree(t *testing.T) {
	// 清理测试目录
	defer func() {
		_ = os.RemoveAll("./test-snapshot-complex")
	}()

	// 创建测试目录
	err := os.MkdirAll("./test-snapshot-complex", 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// 创建 B+ 树并插入大量数据以触发分裂
	tree := NewBPlusTree(3) // 小容量以快速触发分裂

	// 插入数据直到触发分裂
	for i := 0; i < 20; i++ {
		key := string(rune('a' + i))
		value := "value" + string(rune('0'+i))
		err := tree.Put(key, value)
		if err != nil {
			t.Fatalf("Failed to put %s: %v", key, err)
		}
	}

	// 创建 WAL
	wal, err := NewWalWriter("./test-snapshot-complex/wal.dat", tree)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// 创建持久化引擎
	engine := &PersistenceEngine{
		snapshotDir: "./test-snapshot-complex/snapshots",
		tree:        tree,
		wal:         wal,
		snapshotC:   make(chan struct{}, 1),
		stopC:       make(chan struct{}),
	}

	// 创建快照目录
	err = os.MkdirAll("./test-snapshot-complex/snapshots", 0755)
	if err != nil {
		t.Fatalf("Failed to create snapshot directory: %v", err)
	}

	// 打印原始树结构
	t.Log("Original tree structure:")
	tree.Print()

	// 创建快照
	err = engine.snapshot()
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// 测试恢复
	newTree := NewBPlusTree(3)
	newWal, err := NewWalWriter("./test-snapshot-complex/wal.dat", newTree)
	if err != nil {
		t.Fatalf("Failed to create new WAL: %v", err)
	}
	defer newWal.Close()

	newEngine := &PersistenceEngine{
		snapshotDir: "./test-snapshot-complex/snapshots",
		tree:        newTree,
		wal:         newWal,
		snapshotC:   make(chan struct{}, 1),
		stopC:       make(chan struct{}),
	}

	err = newEngine.Recover()
	if err != nil {
		t.Fatalf("Failed to recover snapshot: %v", err)
	}

	// 打印恢复后的树结构
	t.Log("Recovered tree structure:")
	newTree.Print()

	// 验证所有数据都正确恢复
	for i := 0; i < 20; i++ {
		key := string(rune('a' + i))
		expectedValue := "value" + string(rune('0'+i))
		value, ok := newTree.Get(key)
		if !ok {
			t.Fatalf("Key %s not found after recovery", key)
		}
		if value != expectedValue {
			t.Fatalf("For key %s, expected %s, got %s", key, expectedValue, value)
		}
	}

	t.Log("Complex tree snapshot and recovery successful")
}
