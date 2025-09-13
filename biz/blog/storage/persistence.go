package storage

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
	"yuuki798/env-king/biz/blog/myutil"
)

type PersistenceEngine struct {
	snapshotDir string
	tree        *BPlusTree
	wal         *WalWriter
	snapshotC   chan struct{}
	stopC       chan struct{}
}

func NewPersistenceEngine(snapshotDir string, tree *BPlusTree, wal *WalWriter) *PersistenceEngine {
	res := &PersistenceEngine{
		snapshotDir: snapshotDir,
		tree:        tree,
		wal:         wal,
		snapshotC:   make(chan struct{}, 1),
		stopC:       make(chan struct{}),
	}
	err := res.Recover()
	if err != nil {
		panic(err)
	}
	go func() {
		err = res.run()
		if err != nil {
			panic(err)
		}
	}()
	return res
}

// SerializableNode 用于序列化的节点结构，避免循环引用
type SerializableNode struct {
	IsLeaf   bool
	Keys     []string
	Kvs      [][2]string
	Children []*SerializableNode
}

// SerializableTree 用于序列化的 B+ 树结构
type SerializableTree struct {
	MaxChildren int
	MinChildren int
	Root        *SerializableNode
}

func (this *PersistenceEngine) snapshot() error {
	timestamp := time.Now().UnixNano()
	path := filepath.Join(this.snapshotDir, fmt.Sprintf("envking_snapshot_%d.dat", timestamp))
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	// 将 B+ 树转换为可序列化的结构
	serializableTree := this.treeToSerializable()

	enc := gob.NewEncoder(file)
	err = enc.Encode(serializableTree)
	if err != nil {
		file.Close() // 确保在出错时关闭文件
		return err
	}

	// 确保数据写入磁盘
	err = file.Sync()
	if err != nil {
		file.Close() // 确保在出错时关闭文件
		return err
	}

	// 在删除旧快照之前关闭文件
	if err := file.Close(); err != nil {
		return err
	}

	files, err := os.ReadDir(this.snapshotDir)
	if err != nil {
		return err
	}

	//keep5

	snapshots := make([]string, 0, 6)
	re := regexp.MustCompile(`^envking_snapshot_(\d+)\.dat$`)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if re.MatchString(f.Name()) {
			snapshots = append(snapshots, f.Name())
		}
	}

	if len(snapshots) > 5 {
		sort.Slice(snapshots, func(i, j int) bool {
			return snapshots[i] < snapshots[j]
		})
		for _, fname := range snapshots[:len(snapshots)-5] {
			err = os.Remove(filepath.Join(this.snapshotDir, fname))
			if err != nil {
				return err
			}
		}
	}
	this.wal.Clear()
	return nil
}

func (this *PersistenceEngine) run() error {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := this.snapshot()
			if err != nil {
				return err
			}
		case <-this.snapshotC:
			err := this.snapshot()
			if err != nil {
				return err
			}
		case <-this.stopC:
			err := this.snapshot()
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func (this *PersistenceEngine) Recover() error {
	err := myutil.Mkdir(this.snapshotDir)
	if err != nil {
		return err
	}
	log.Println(this.snapshotDir)

	// 检查是否有快照文件
	files, err := os.ReadDir(this.snapshotDir)
	if err != nil {
		return err
	}

	// 过滤出快照文件
	snapshots := make([]string, 0)
	re := regexp.MustCompile(`^envking_snapshot_(\d+)\.dat$`)
	for _, f := range files {
		if !f.IsDir() && re.MatchString(f.Name()) {
			snapshots = append(snapshots, f.Name())
		}
	}

	if len(snapshots) > 0 {
		// 有快照文件，按时间戳排序获取最新的
		sort.Slice(snapshots, func(i, j int) bool {
			return snapshots[i] > snapshots[j]
		})
		latestSnap := snapshots[0]
		err = this.ApplySnap(latestSnap)
		if err != nil {
			// 快照恢复失败必须 panic，因为这是数据完整性的关键
			panic(fmt.Sprintf("Failed to apply snapshot %s: %v", latestSnap, err))
		}
		log.Printf("Successfully applied snapshot: %s", latestSnap)

		// 快照恢复成功后，进行 WAL 恢复
		this.wal.RecoverC <- struct{}{}
		select {
		case <-this.wal.RecoverCallbackC:
			log.Println("WAL recovery completed")
			return nil
		case <-time.After(50 * time.Second):
			return fmt.Errorf("WAL recovery timeout")
		}
	} else {
		// 没有快照文件，检查是否有WAL文件需要恢复
		log.Println("No snapshot found, checking for WAL recovery...")

		// 检查WAL文件是否存在
		walFiles, err := os.ReadDir(this.snapshotDir + "/..")
		if err != nil {
			return err
		}

		hasWalFile := false
		for _, f := range walFiles {
			if !f.IsDir() && f.Name() == "wal.dat" {
				hasWalFile = true
				break
			}
		}

		if hasWalFile {
			// 有WAL文件，进行WAL恢复
			log.Println("WAL files found, starting WAL recovery...")
			this.wal.RecoverC <- struct{}{}
			select {
			case <-this.wal.RecoverCallbackC:
				log.Println("WAL recovery completed")
				return nil
			case <-time.After(50 * time.Second):
				return fmt.Errorf("WAL recovery timeout")
			}
		} else {
			// 既没有快照也没有WAL，全新启动
			log.Println("No snapshot or WAL found, starting fresh")
			return nil
		}
	}
}

// treeToSerializable 将 B+ 树转换为可序列化的结构
func (this *PersistenceEngine) treeToSerializable() *SerializableTree {
	if this.tree == nil {
		return &SerializableTree{
			MaxChildren: 0,
			MinChildren: 0,
			Root:        nil,
		}
	}

	return &SerializableTree{
		MaxChildren: this.tree.maxChildren,
		MinChildren: this.tree.minChildren,
		Root:        this.nodeToSerializable(this.tree.root),
	}
}

// nodeToSerializable 递归转换节点
func (this *PersistenceEngine) nodeToSerializable(node *Node) *SerializableNode {
	if node == nil {
		return nil
	}

	serializableNode := &SerializableNode{
		IsLeaf: node.IsLeaf,
		Keys:   make([]string, len(node.Keys)),
		Kvs:    make([][2]string, len(node.Kvs)),
	}

	copy(serializableNode.Keys, node.Keys)
	copy(serializableNode.Kvs, node.Kvs)

	if !node.IsLeaf {
		serializableNode.Children = make([]*SerializableNode, len(node.Children))
		for i, child := range node.Children {
			serializableNode.Children[i] = this.nodeToSerializable(child)
		}
	}

	return serializableNode
}

// serializableToTree 将可序列化结构转换回 B+ 树
func (this *PersistenceEngine) serializableToTree(serializableTree *SerializableTree) *BPlusTree {
	tree := &BPlusTree{
		maxChildren: serializableTree.MaxChildren,
		minChildren: serializableTree.MinChildren,
		root:        this.serializableToNode(serializableTree.Root),
	}

	// 重建指针关系
	tree.rebuildPointers()

	return tree
}

// serializableToNode 递归转换节点
func (this *PersistenceEngine) serializableToNode(serializableNode *SerializableNode) *Node {
	if serializableNode == nil {
		return nil
	}

	node := &Node{
		IsLeaf: serializableNode.IsLeaf,
		Keys:   make([]string, len(serializableNode.Keys)),
		Kvs:    make([][2]string, len(serializableNode.Kvs)),
	}

	copy(node.Keys, serializableNode.Keys)
	copy(node.Kvs, serializableNode.Kvs)

	if !serializableNode.IsLeaf {
		node.Children = make([]*Node, len(serializableNode.Children))
		for i, child := range serializableNode.Children {
			node.Children[i] = this.serializableToNode(child)
		}
	}

	return node
}

func (this *PersistenceEngine) ApplySnap(snap string) error {
	f, err := os.Open(filepath.Join(this.snapshotDir, snap))
	if err != nil {
		return err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	var serializableTree SerializableTree
	err = dec.Decode(&serializableTree)
	if err != nil {
		return err
	}

	// 将可序列化结构转换回 B+ 树
	restoredTree := this.serializableToTree(&serializableTree)

	// 替换当前树
	*this.tree = *restoredTree

	return nil
}

// TakeSnapshot 手动触发快照
func (this *PersistenceEngine) TakeSnapshot() error {
	return this.snapshot()
}

// Close 关闭持久化引擎，清理资源
func (this *PersistenceEngine) Close() error {
	// 通知停止 goroutine
	close(this.stopC)

	// 关闭 WAL
	if this.wal != nil {
		return this.wal.Close()
	}

	return nil
}
