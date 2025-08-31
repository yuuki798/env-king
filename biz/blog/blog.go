package blog

import "time"

// Hub 全局单例
type Hub struct {
	userManager *UserManager
}

// UserManager 管理所有用户的博客
type UserManager struct {
	blogManagers map[string]*blogManager
}

// blogManager 每个用户有一个
type blogManager struct {
	// We use a tree structure to store blogs
	blogTreeRoot *DirectoryNode
	// localPath means the root directory where blogs are stored locally
	localPath        string
	platforms        []string
	platformAdapters map[string]PlatformAdapter
	userName         string
}
type DirectoryNode struct {
	// Each directory corresponds to a category, the root directory is "default". If directory is not the
	// 2nd level, it can connect with "/" to form a complete category path.
	category string
	// the key is the directory name, also the category name
	children map[string]*DirectoryNode
	blogs    []*blog
}
type blog struct {
	// last updated time
	updatedAt time.Time
	category  string
	tag       string
	name      string
	file      []byte
	// Old versions store the previous versions of the blog except the current one.
	// It always keeps the last 10 versions and ordered by updatedAt desc.
	oldVersions []*blog
}
type IBlogManager interface {
	// readFromLocal init all blogs from local path to a blog tree
	readFromLocal() error

	Commit() error
	Push() error
	List() error
	History(page int, path string) error
}
type PlatformAdapter interface {
	Auth() error
	SaveToDraft() error
	PreProcess() error
	Publish() error
	GetList() error
}
