package storage

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type BPlusTree struct {
	maxChildren int
	minChildren int
	root        *Node
}

func (this *BPlusTree) fixPut(cur *Node) error {
	var err error
	// if cur is leaf
	if cur.IsLeaf {
		// is ok
		if len(cur.Kvs) <= this.maxChildren {
			return nil
		}
		cur, err = cur.splitLeaf(this.maxChildren)
		if err != nil {
			return err
		}
	}
	// for not leaf
	for {
		if len(cur.Keys) <= this.maxChildren {
			break
		}
		cur, err = cur.splitIndex(this.maxChildren)
		if err != nil {
			return nil
		}
		if cur.Parent == nil && len(cur.Keys) <= this.maxChildren {
			break
		}
	}
	if cur != this.root && cur.Parent == nil {
		this.root = cur
	}
	return nil
}

type Node struct {
	Parent   *Node
	Children []*Node

	IsLeaf bool

	// not leaf
	Keys []string

	// is leaf
	Kvs  [][2]string
	Prev *Node
	Next *Node
}

func (this *Node) insertInLeaf(key string, value string) error {
	if !this.IsLeaf {
		return errors.New("is not leaf")
	}
	left := 0
	right := len(this.Kvs)
	// for most conditions, left is close and right is open
	for left < right {
		mid := left + (right-left)/2
		if this.Kvs[mid][0] == key {
			// update the value
			this.Kvs[mid][1] = value
			return nil
		} else if this.Kvs[mid][0] < key {
			left = mid + 1
		} else {
			right = mid
		}
	}
	// left should insert a key
	this.Kvs = append(this.Kvs, [2]string{})
	// get the min len. And 2 Kvs's length has changed, so don't use len(Kvs)-1 but len(Kvs)-2
	copy(this.Kvs[left+1:], this.Kvs[left:])
	this.Kvs[left] = [2]string{key, value}

	return nil
}
func (this *Node) splitLeaf(maxChildren int) (*Node, error) {
	// it means length is out of maxChildren num
	rightStart := len(this.Kvs) / 2
	left := &Node{
		Parent: this.Parent,
		IsLeaf: true,
		Kvs: func() [][2]string {
			newSpace := make([][2]string, len(this.Kvs)/2, maxChildren+1)
			copy(newSpace, this.Kvs[:rightStart])
			return newSpace
		}(),
	}
	right := &Node{
		Parent: this.Parent,
		IsLeaf: true,
		Kvs: func() [][2]string {
			newSpace := make([][2]string, len(this.Kvs)-len(this.Kvs)/2, maxChildren+1)
			copy(newSpace, this.Kvs[rightStart:])
			return newSpace
		}(),
		Prev: left,
	}
	left.Next = right
	var resParent *Node

	parent := this.Parent

	if parent == nil {
		resParent = &Node{
			IsLeaf:   false,
			Keys:     []string{this.Kvs[rightStart][0]},
			Children: []*Node{left, right},
		}
		left.Parent = resParent
		right.Parent = resParent
	} else {
		curIndexOfParent := func() int {
			for i, x := range parent.Children {
				if x == this {
					return i
				}
			}
			return -1
		}()
		if curIndexOfParent == -1 {
			return nil, errors.New("cannot find itself by Parent")
		}
		// expand capacity
		parent.Children = append(parent.Children, nil)
		copy(parent.Children[curIndexOfParent+1:], parent.Children[curIndexOfParent:])
		parent.Children[curIndexOfParent] = left
		parent.Children[curIndexOfParent+1] = right

		parent.Keys = append(parent.Keys, "")
		copy(parent.Keys[curIndexOfParent+1:], parent.Keys[curIndexOfParent:])
		parent.Keys[curIndexOfParent] = this.Kvs[rightStart][0]

		prev := this.Prev
		next := this.Next
		if prev != nil {
			prev.Next = left
		}
		left.Prev = prev
		right.Next = next
		if next != nil {
			next.Prev = right
		}
		resParent = left.Parent
	}
	return resParent, nil
}

func (this *Node) splitIndex(maxChildren int) (*Node, error) {
	if this.IsLeaf {
		return nil, errors.New("is not index but leaf")
	}
	// it means length is out of maxChildren num
	rightStart := len(this.Keys) / 2
	left := &Node{
		Parent: this.Parent,
		IsLeaf: false,
		Keys: func() []string {
			newSpace := make([]string, len(this.Keys)/2, maxChildren+1)
			copy(newSpace, this.Keys[:rightStart])
			return newSpace
		}(),
		Children: this.Children[:len(this.Keys)/2+1],
	}
	right := &Node{
		Parent: this.Parent,
		IsLeaf: false,
		Keys: func() []string {
			newSpace := make([]string, len(this.Keys)-len(this.Keys)/2-1, maxChildren+1)
			copy(newSpace, this.Keys[rightStart+1:])
			return newSpace
		}(),
		Children: this.Children[len(this.Keys)/2+1:],
	}

	for _, x := range left.Children {
		x.Parent = left
	}
	for _, x := range right.Children {
		x.Parent = right
	}
	var resParent *Node

	parent := this.Parent

	if parent == nil {
		resParent = &Node{
			IsLeaf:   false,
			Keys:     []string{this.Keys[rightStart]},
			Children: []*Node{left, right},
		}
		left.Parent = resParent
		right.Parent = resParent
	} else {
		curIndexOfParent := func() int {
			for i, x := range parent.Children {
				if x == this {
					return i
				}
			}
			return -1
		}()
		if curIndexOfParent == -1 {
			return nil, errors.New("cannot find itself by Parent")
		}
		// expand capacity
		parent.Children = append(parent.Children, nil)
		copy(parent.Children[curIndexOfParent+1:], parent.Children[curIndexOfParent:len(parent.Children)-1])
		parent.Children[curIndexOfParent] = left
		parent.Children[curIndexOfParent+1] = right

		parent.Keys = append(parent.Keys, "")
		copy(parent.Keys[curIndexOfParent+1:], parent.Keys[curIndexOfParent:])
		parent.Keys[curIndexOfParent] = this.Keys[rightStart]
		// todo
		resParent = left.Parent
	}
	return resParent, nil
}
func NewBPlusTree(maxChildren int) *BPlusTree {
	if maxChildren < 2 {
		return nil
	}
	return &BPlusTree{
		maxChildren: maxChildren,
		root:        nil,
		minChildren: (maxChildren + 1) / 2,
	}
}

func (this *BPlusTree) Put(key string, value string) error {
	if this.root == nil {
		this.root = &Node{
			IsLeaf: true,
			Kvs:    [][2]string{{key, value}},
		}
		return nil
	}
	cur := this.root
	for {
		//if len(cur.Kvs) > this.maxChildren {
		//	// means struct is error
		//	return errors.New("the struct was error")
		//}
		if cur.IsLeaf && len(cur.Kvs) < this.maxChildren {
			err := cur.insertInLeaf(key, value)
			if err != nil {
				return err
			}
			break
		} else if cur.IsLeaf && len(cur.Kvs) == this.maxChildren {
			err := cur.insertInLeaf(key, value)
			if err != nil {
				return err
			}
			err = this.fixPut(cur)
			if err != nil {
				return err
			}
			//// now if maxChildren is 2, the length is 3,
			//// execute split
			//middleIndex := cur.splitLeaf(this.maxChildren)
			//Parent := cur.Parent
			//Parent.insertInIndex(middleIndex)
			break
		} else if !cur.IsLeaf {
			keys := cur.Keys
			index := sort.Search(len(keys), func(i int) bool { return keys[i] > key }) // find the insert pos
			children := cur.Children
			cur = children[index]
		}
	}
	return nil
}
func (this *BPlusTree) Get(key string) (value string, ok bool) {
	if this.root == nil {
		return "", false
	}
	cur := this.root
	var index int
	for {
		if !cur.IsLeaf {
			index = sort.Search(len(cur.Keys), func(i int) bool {
				return cur.Keys[i] > key
			})
			cur = cur.Children[index]
		} else {
			index = sort.Search(len(cur.Kvs), func(i int) bool {
				return cur.Kvs[i][0] >= key
			})
			// if not exist
			if index >= len(cur.Kvs) {
				return "", false
			}
			if cur.Kvs[index][0] == key {
				return cur.Kvs[index][1], true
			} else {
				return "", false
			}
		}
	}
}
func (this *BPlusTree) Del(key string) (ok bool, err error) {
	if this.root == nil {
		return false, errors.New("key not found")
	}
	cur := this.root
	var index int
	for {
		if !cur.IsLeaf {
			index = sort.Search(len(cur.Keys), func(i int) bool {
				return cur.Keys[i] > key
			})
			cur = cur.Children[index]
		} else {
			index = sort.Search(len(cur.Kvs), func(i int) bool {
				return cur.Kvs[i][0] >= key
			})
			// if not exist
			if index >= len(cur.Kvs) {
				return false, errors.New("key not found")
			}
			if cur.Kvs[index][0] == key {
				cur.Kvs = append(cur.Kvs[:index], cur.Kvs[index+1:]...) // del index's val
				err = this.fixDel(cur)
				if err != nil {
					return false, err
				}
				return true, nil
			} else {
				return false, errors.New("key not found")
			}
		}
	}
}
func (this *BPlusTree) fixDel(node *Node) error {
	// 自底向上修复删除后的不平衡
	for node != nil {
		// 如果是根节点，特殊处理
		if node == this.root {
			return this.fixRoot()
		}

		// 检查节点是否违反最小度约束
		if node.IsLeaf {
			if len(node.Kvs) >= this.minChildren {
				break // 叶子节点满足约束，停止
			}
		} else {
			if len(node.Keys) >= this.minChildren {
				break // 内部节点满足约束，停止
			}
		}

		// 违反约束，需要修复
		node = this.rebalanceNode(node)
	}
	return nil
}

// 修复根节点
func (this *BPlusTree) fixRoot() error {
	if this.root.IsLeaf {
		return nil // 叶子根节点可以为空
	}
	if len(this.root.Keys) == 0 {
		if len(this.root.Children) == 1 {
			this.root = this.root.Children[0]
			this.root.Parent = nil
		}
	}
	return nil
}

// 重新平衡节点
func (this *BPlusTree) rebalanceNode(node *Node) *Node {
	parent := node.Parent
	nodeIndex := this.findChildIndex(parent, node)

	// 尝试从左兄弟借用
	if nodeIndex > 0 {
		leftSibling := parent.Children[nodeIndex-1]
		if this.canBorrow(leftSibling) {
			this.borrowFromLeft(node, leftSibling, parent, nodeIndex)
			return parent
		}
	}

	// 尝试从右兄弟借用
	if nodeIndex < len(parent.Children)-1 {
		rightSibling := parent.Children[nodeIndex+1]
		if this.canBorrow(rightSibling) {
			this.borrowFromRight(node, rightSibling, parent, nodeIndex)
			return parent
		}
	}

	// 无法借用，进行合并
	if nodeIndex > 0 {
		// 与左兄弟合并
		leftSibling := parent.Children[nodeIndex-1]
		this.mergeWithLeft(node, leftSibling, parent, nodeIndex)
	} else {
		// 与右兄弟合并
		rightSibling := parent.Children[nodeIndex+1]
		this.mergeWithRight(node, rightSibling, parent, nodeIndex)
	}

	return parent
}

// 查找子节点在父节点中的索引
func (this *BPlusTree) findChildIndex(parent *Node, child *Node) int {
	for i, c := range parent.Children {
		if c == child {
			return i
		}
	}
	return -1
}

// 检查是否可以从兄弟节点借用
func (this *BPlusTree) canBorrow(sibling *Node) bool {
	if sibling.IsLeaf {
		return len(sibling.Kvs) > this.minChildren
	}
	return len(sibling.Keys) > this.minChildren
}

// 从左兄弟借用
func (this *BPlusTree) borrowFromLeft(node, leftSibling, parent *Node, nodeIndex int) {
	if node.IsLeaf {
		// 叶子节点：移动最后一个kv
		lastKV := leftSibling.Kvs[len(leftSibling.Kvs)-1]
		leftSibling.Kvs = leftSibling.Kvs[:len(leftSibling.Kvs)-1]
		node.Kvs = append([][2]string{lastKV}, node.Kvs...)
		// 更新父节点的分割key
		parent.Keys[nodeIndex-1] = node.Kvs[0][0]
	} else {
		// 内部节点：下拉父节点key，上提兄弟key
		parentKey := parent.Keys[nodeIndex-1]
		borrowedKey := leftSibling.Keys[len(leftSibling.Keys)-1]
		borrowedChild := leftSibling.Children[len(leftSibling.Children)-1]

		leftSibling.Keys = leftSibling.Keys[:len(leftSibling.Keys)-1]
		leftSibling.Children = leftSibling.Children[:len(leftSibling.Children)-1]

		node.Keys = append([]string{parentKey}, node.Keys...)
		node.Children = append([]*Node{borrowedChild}, node.Children...)
		borrowedChild.Parent = node

		parent.Keys[nodeIndex-1] = borrowedKey
	}
}

// 从右兄弟借用
func (this *BPlusTree) borrowFromRight(node, rightSibling, parent *Node, nodeIndex int) {
	if node.IsLeaf {
		// 叶子节点：移动第一个kv
		firstKV := rightSibling.Kvs[0]
		rightSibling.Kvs = rightSibling.Kvs[1:]
		node.Kvs = append(node.Kvs, firstKV)
		// 更新父节点的分割key
		parent.Keys[nodeIndex] = rightSibling.Kvs[0][0]
	} else {
		// 内部节点：下拉父节点key，上提兄弟key
		parentKey := parent.Keys[nodeIndex]
		borrowedKey := rightSibling.Keys[0]
		borrowedChild := rightSibling.Children[0]

		rightSibling.Keys = rightSibling.Keys[1:]
		rightSibling.Children = rightSibling.Children[1:]

		node.Keys = append(node.Keys, parentKey)
		node.Children = append(node.Children, borrowedChild)
		borrowedChild.Parent = node

		parent.Keys[nodeIndex] = borrowedKey
	}
}

// 与左兄弟合并
func (this *BPlusTree) mergeWithLeft(node, leftSibling, parent *Node, nodeIndex int) {
	if node.IsLeaf {
		// 叶子节点合并
		leftSibling.Kvs = append(leftSibling.Kvs, node.Kvs...)
		// 更新链表指针
		leftSibling.Next = node.Next
		if node.Next != nil {
			node.Next.Prev = leftSibling
		}
	} else {
		// 内部节点合并：下拉父节点key
		parentKey := parent.Keys[nodeIndex-1]
		leftSibling.Keys = append(leftSibling.Keys, parentKey)
		leftSibling.Keys = append(leftSibling.Keys, node.Keys...)
		leftSibling.Children = append(leftSibling.Children, node.Children...)
		// 更新children的parent指针
		for _, child := range node.Children {
			child.Parent = leftSibling
		}
	}

	// 从父节点删除key和child
	parent.Keys = append(parent.Keys[:nodeIndex-1], parent.Keys[nodeIndex:]...)
	parent.Children = append(parent.Children[:nodeIndex], parent.Children[nodeIndex+1:]...)
}

// 与右兄弟合并
func (this *BPlusTree) mergeWithRight(node, rightSibling, parent *Node, nodeIndex int) {
	if node.IsLeaf {
		// 叶子节点合并
		node.Kvs = append(node.Kvs, rightSibling.Kvs...)
		// 更新链表指针
		node.Next = rightSibling.Next
		if rightSibling.Next != nil {
			rightSibling.Next.Prev = node
		}
	} else {
		// 内部节点合并：下拉父节点key
		parentKey := parent.Keys[nodeIndex]
		node.Keys = append(node.Keys, parentKey)
		node.Keys = append(node.Keys, rightSibling.Keys...)
		node.Children = append(node.Children, rightSibling.Children...)
		// 更新children的parent指针
		for _, child := range rightSibling.Children {
			child.Parent = node
		}
	}

	// 从父节点删除key和child
	parent.Keys = append(parent.Keys[:nodeIndex], parent.Keys[nodeIndex+1:]...)
	parent.Children = append(parent.Children[:nodeIndex+1], parent.Children[nodeIndex+2:]...)
}

// rebuildPointers 重建 B+ 树的指针关系
func (this *BPlusTree) rebuildPointers() {
	if this.root == nil {
		return
	}

	// 重建父子关系
	this.rebuildParentPointers(this.root, nil)

	// 重建叶子节点链表
	this.rebuildLeafLinks()
}

// rebuildParentPointers 递归重建父子关系
func (this *BPlusTree) rebuildParentPointers(node *Node, parent *Node) {
	if node == nil {
		return
	}

	node.Parent = parent

	if !node.IsLeaf {
		// 内部节点：重建子节点的父子关系
		for _, child := range node.Children {
			this.rebuildParentPointers(child, node)
		}
	}
}

// rebuildLeafLinks 重建叶子节点链表
func (this *BPlusTree) rebuildLeafLinks() {
	if this.root == nil {
		return
	}

	// 收集所有叶子节点
	var leaves []*Node
	this.collectLeaves(this.root, &leaves)

	// 重建链表关系
	for i := 0; i < len(leaves); i++ {
		if i > 0 {
			leaves[i].Prev = leaves[i-1]
		}
		if i < len(leaves)-1 {
			leaves[i].Next = leaves[i+1]
		}
	}
}

// collectLeaves 递归收集所有叶子节点
func (this *BPlusTree) collectLeaves(node *Node, leaves *[]*Node) {
	if node == nil {
		return
	}

	if node.IsLeaf {
		*leaves = append(*leaves, node)
	} else {
		for _, child := range node.Children {
			this.collectLeaves(child, leaves)
		}
	}
}

func (this *BPlusTree) Print() {
	if this.root == nil {
		fmt.Println("Tree is empty")
		return
	}
	q := make([]*Node, 0)
	q = append(q, this.root)
	for len(q) > 0 {
		curQLen := len(q)
		out := ""
		for _ = range curQLen {
			node := q[0]
			q = q[1:]
			if node.IsLeaf == false {
				out = out + " " + fmt.Sprintf("%v", node.Keys)
				for _, x := range node.Children {
					q = append(q, x)
				}
			} else {
				out = out + "->" + fmt.Sprintf("%v", node.Kvs)
			}
		}
		fmt.Println(strings.TrimPrefix(out, "->"))
	}
	fmt.Println("---")
}
